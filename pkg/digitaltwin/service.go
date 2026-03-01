package digitaltwin

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mimir-aip/mimir-aip-go/pkg/metadatastore"
	"github.com/mimir-aip/mimir-aip-go/pkg/mlmodel"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"github.com/mimir-aip/mimir-aip-go/pkg/ontology"
	"github.com/mimir-aip/mimir-aip-go/pkg/queue"
	"github.com/mimir-aip/mimir-aip-go/pkg/storage"
)

// Service manages digital twin operations
type Service struct {
	store           metadatastore.MetadataStore
	ontologyService *ontology.Service
	storageService  *storage.Service
	mlService       *mlmodel.Service
	queue           *queue.Queue
	inferenceEngine *InferenceEngine
	sparqlEngine    *SPARQLEngine
	scenarioManager *ScenarioManager
	actionManager   *ActionManager
}

// NewService creates a new digital twin service
func NewService(
	store metadatastore.MetadataStore,
	ontologyService *ontology.Service,
	storageService *storage.Service,
	mlService *mlmodel.Service,
	q *queue.Queue,
) *Service {
	s := &Service{
		store:           store,
		ontologyService: ontologyService,
		storageService:  storageService,
		mlService:       mlService,
		queue:           q,
	}

	// Initialize sub-components
	s.inferenceEngine = NewInferenceEngine(mlService, store)
	s.sparqlEngine = NewSPARQLEngine(store, ontologyService)
	s.scenarioManager = NewScenarioManager(store, s.inferenceEngine)
	s.actionManager = NewActionManager(store, q)

	return s
}

// CreateDigitalTwin creates a new digital twin instance
func (s *Service) CreateDigitalTwin(req *models.DigitalTwinCreateRequest) (*models.DigitalTwin, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// Verify ontology exists
	ont, err := s.ontologyService.GetOntology(req.OntologyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get ontology: %w", err)
	}

	if ont.ProjectID != req.ProjectID {
		return nil, fmt.Errorf("ontology does not belong to project")
	}

	now := time.Now().UTC()
	twin := &models.DigitalTwin{
		ID:          uuid.New().String(),
		ProjectID:   req.ProjectID,
		OntologyID:  req.OntologyID,
		Name:        req.Name,
		Description: req.Description,
		Status:      "active",
		Config:      req.Config,
		Metadata:    make(map[string]interface{}),
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Set default config if not provided
	if twin.Config == nil {
		twin.Config = &models.DigitalTwinConfig{
			CacheTTL:           300, // 5 minutes
			AutoSync:           false,
			SyncInterval:       3600, // 1 hour
			EnablePredictions:  true,
			PredictionCacheTTL: 1800, // 30 minutes
			IndexingStrategy:   "lazy",
		}
	}

	if err := s.store.SaveDigitalTwin(twin); err != nil {
		return nil, fmt.Errorf("failed to save digital twin: %w", err)
	}

	// Initialize from ontology (populate entities from ontology blueprint)
	if err := s.initializeFromOntology(twin, ont); err != nil {
		return nil, fmt.Errorf("failed to initialize from ontology: %w", err)
	}

	return twin, nil
}

// GetDigitalTwin retrieves a digital twin by ID
func (s *Service) GetDigitalTwin(id string) (*models.DigitalTwin, error) {
	twin, err := s.store.GetDigitalTwin(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get digital twin: %w", err)
	}
	return twin, nil
}

// UpdateDigitalTwin updates an existing digital twin
func (s *Service) UpdateDigitalTwin(id string, req *models.DigitalTwinUpdateRequest) (*models.DigitalTwin, error) {
	twin, err := s.store.GetDigitalTwin(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get digital twin: %w", err)
	}

	// Apply updates
	if req.Name != nil {
		twin.Name = *req.Name
	}
	if req.Description != nil {
		twin.Description = *req.Description
	}
	if req.Status != nil {
		twin.Status = *req.Status
	}
	if req.Config != nil {
		twin.Config = req.Config
	}

	twin.UpdatedAt = time.Now().UTC()

	if err := s.store.SaveDigitalTwin(twin); err != nil {
		return nil, fmt.Errorf("failed to update digital twin: %w", err)
	}

	return twin, nil
}

// DeleteDigitalTwin deletes a digital twin
func (s *Service) DeleteDigitalTwin(id string) error {
	if err := s.store.DeleteDigitalTwin(id); err != nil {
		return fmt.Errorf("failed to delete digital twin: %w", err)
	}
	return nil
}

// ListDigitalTwins lists all digital twins
func (s *Service) ListDigitalTwins() ([]*models.DigitalTwin, error) {
	twins, err := s.store.ListDigitalTwins()
	if err != nil {
		return nil, fmt.Errorf("failed to list digital twins: %w", err)
	}
	return twins, nil
}

// ListDigitalTwinsByProject lists digital twins for a specific project
func (s *Service) ListDigitalTwinsByProject(projectID string) ([]*models.DigitalTwin, error) {
	twins, err := s.store.ListDigitalTwinsByProject(projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list digital twins: %w", err)
	}
	return twins, nil
}

// SyncWithStorage synchronizes digital twin entities with CIR data from storage.
//
// Entity resolution: when the same logical entity appears in multiple storage
// sources (e.g. student_id=42 in grades_db and in attendance_db), the records
// are merged into a single canonical entity rather than creating duplicates.
// Resolution is driven entirely by key-field value matching — no configuration
// required.
//
// Relationship wiring: after all sources are synced, entities of different
// types that share a common key-field value (the join detected during
// extraction) are linked via EntityRelationship entries.
func (s *Service) SyncWithStorage(twinID string) error {
	twin, err := s.store.GetDigitalTwin(twinID)
	if err != nil {
		return fmt.Errorf("failed to get digital twin: %w", err)
	}

	ont, err := s.ontologyService.GetOntology(twin.OntologyID)
	if err != nil {
		return fmt.Errorf("failed to get ontology: %w", err)
	}

	if twin.Config != nil && len(twin.Config.StorageIDs) > 0 {
		for _, storageID := range twin.Config.StorageIDs {
			if err := s.syncFromStorage(twin, ont, storageID); err != nil {
				return fmt.Errorf("failed to sync from storage %s: %w", storageID, err)
			}
		}
	}

	// Wire cross-type relationships based on shared key field values.
	if err := s.wireRelationships(twin.ID); err != nil {
		// Non-fatal: log and continue.
		fmt.Printf("Warning: failed to wire cross-type relationships for twin %s: %v\n", twin.ID, err)
	}

	now := time.Now().UTC()
	twin.LastSyncAt = &now
	twin.UpdatedAt = now

	if err := s.store.SaveDigitalTwin(twin); err != nil {
		return fmt.Errorf("failed to update digital twin: %w", err)
	}

	return nil
}

// GetEntity retrieves an entity by ID
func (s *Service) GetEntity(id string) (*models.Entity, error) {
	entity, err := s.store.GetEntity(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get entity: %w", err)
	}

	// If entity has source data, merge with CIR data
	if entity.SourceDataID != nil {
		if err := s.populateEntityFromSource(entity); err != nil {
			// Log error but don't fail - entity might still have modifications
			fmt.Printf("Warning: failed to populate entity from source: %v\n", err)
		}
	}

	return entity, nil
}

// UpdateEntity updates entity attributes (stores as delta modifications)
func (s *Service) UpdateEntity(entityID string, req *models.EntityUpdateRequest) (*models.Entity, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	entity, err := s.store.GetEntity(entityID)
	if err != nil {
		return nil, fmt.Errorf("failed to get entity: %w", err)
	}

	// Store modifications as deltas
	if entity.Modifications == nil {
		entity.Modifications = make(map[string]interface{})
	}

	for key, value := range req.Attributes {
		entity.Modifications[key] = value
		// Also update current attributes
		if entity.Attributes == nil {
			entity.Attributes = make(map[string]interface{})
		}
		entity.Attributes[key] = value
	}

	entity.IsModified = true
	entity.UpdatedAt = time.Now().UTC()

	if err := s.store.SaveEntity(entity); err != nil {
		return nil, fmt.Errorf("failed to update entity: %w", err)
	}

	// Invalidate predictions for this entity
	if err := s.invalidatePredictionsForEntity(entityID); err != nil {
		// Log error but don't fail
		fmt.Printf("Warning: failed to invalidate predictions: %v\n", err)
	}

	return entity, nil
}

// ListEntities lists all entities for a digital twin
func (s *Service) ListEntities(twinID string) ([]*models.Entity, error) {
	entities, err := s.store.ListEntitiesByDigitalTwin(twinID)
	if err != nil {
		return nil, fmt.Errorf("failed to list entities: %w", err)
	}

	// Populate entities from source data
	for _, entity := range entities {
		if entity.SourceDataID != nil {
			if err := s.populateEntityFromSource(entity); err != nil {
				// Log error but continue
				fmt.Printf("Warning: failed to populate entity %s: %v\n", entity.ID, err)
			}
		}
	}

	return entities, nil
}

// Query executes a SPARQL query on the digital twin
func (s *Service) Query(twinID string, req *models.QueryRequest) (*models.QueryResult, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	twin, err := s.store.GetDigitalTwin(twinID)
	if err != nil {
		return nil, fmt.Errorf("failed to get digital twin: %w", err)
	}

	// Execute SPARQL query
	result, err := s.sparqlEngine.Execute(twin, req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}

	return result, nil
}

// Predict runs ML model prediction on entity data.
//
// When EntityID is provided, the service automatically enriches the input
// feature map with attributes from directly related entities, prefixed by
// their type (e.g. "attendance.days_absent").  This lets models trained on
// cross-source features produce accurate predictions even when called with
// only an entity reference rather than a manually constructed feature vector.
func (s *Service) Predict(twinID string, req *models.PredictionRequest) (*models.Prediction, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	twin, err := s.store.GetDigitalTwin(twinID)
	if err != nil {
		return nil, fmt.Errorf("failed to get digital twin: %w", err)
	}

	// Check if predictions are enabled
	if twin.Config != nil && !twin.Config.EnablePredictions {
		return nil, fmt.Errorf("predictions are disabled for this digital twin")
	}

	// Enrich the input with related entity attributes when an entity ID is
	// given.  This is non-fatal: if enrichment fails the caller-supplied
	// input (or an empty map) is used as-is.
	if req.EntityID != "" {
		if err := s.enrichPredictionInput(req); err != nil {
			fmt.Printf("Warning: failed to enrich prediction input for entity %s: %v\n", req.EntityID, err)
		}
	}

	// Run prediction through inference engine
	prediction, err := s.inferenceEngine.Predict(twin, req)
	if err != nil {
		return nil, fmt.Errorf("failed to run prediction: %w", err)
	}

	// Save prediction
	if err := s.store.SavePrediction(prediction); err != nil {
		return nil, fmt.Errorf("failed to save prediction: %w", err)
	}

	// Check if prediction triggers any actions
	if err := s.actionManager.EvaluateActions(twin.ID, prediction); err != nil {
		// Log error but don't fail
		fmt.Printf("Warning: failed to evaluate actions: %v\n", err)
	}

	return prediction, nil
}

// enrichPredictionInput loads the target entity's attributes into req.Input
// (if the caller did not supply them) and then merges attributes from all
// directly related entities under type-prefixed keys.
//
// Example: a Student entity with a related AttendanceRecord produces keys like:
//
//	"avg_grade"              → from Student.Attributes
//	"attendancerecord.days_absent" → from the related AttendanceRecord
//
// Related entity attributes are added only when the key is not already present,
// so caller-supplied values always take precedence.  Depth is limited to
// direct (depth-1) relationships to keep the feature vector bounded.
func (s *Service) enrichPredictionInput(req *models.PredictionRequest) error {
	entity, err := s.store.GetEntity(req.EntityID)
	if err != nil {
		return fmt.Errorf("failed to load entity: %w", err)
	}

	// Auto-populate input from entity attributes when the caller omitted them.
	if len(req.Input) == 0 {
		req.Input = make(map[string]interface{}, len(entity.Attributes))
		for k, v := range entity.Attributes {
			req.Input[k] = v
		}
	}

	// Merge related entity attributes with a type prefix.
	for _, rel := range entity.Relationships {
		related, err := s.store.GetEntity(rel.TargetID)
		if err != nil {
			continue // non-fatal: skip missing related entities
		}
		prefix := strings.ToLower(rel.TargetType) + "."
		for k, v := range related.Attributes {
			key := prefix + k
			if _, exists := req.Input[key]; !exists {
				req.Input[key] = v
			}
		}
	}

	return nil
}

// BatchPredict runs batch predictions
func (s *Service) BatchPredict(twinID string, req *models.BatchPredictionRequest) ([]*models.Prediction, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	twin, err := s.store.GetDigitalTwin(twinID)
	if err != nil {
		return nil, fmt.Errorf("failed to get digital twin: %w", err)
	}

	if twin.Config != nil && !twin.Config.EnablePredictions {
		return nil, fmt.Errorf("predictions are disabled for this digital twin")
	}

	predictions, err := s.inferenceEngine.BatchPredict(twin, req)
	if err != nil {
		return nil, fmt.Errorf("failed to run batch predictions: %w", err)
	}

	// Save all predictions
	for _, prediction := range predictions {
		if err := s.store.SavePrediction(prediction); err != nil {
			fmt.Printf("Warning: failed to save prediction %s: %v\n", prediction.ID, err)
		}
	}

	return predictions, nil
}

// CreateScenario creates a new what-if scenario
func (s *Service) CreateScenario(twinID string, req *models.ScenarioCreateRequest) (*models.Scenario, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	twin, err := s.store.GetDigitalTwin(twinID)
	if err != nil {
		return nil, fmt.Errorf("failed to get digital twin: %w", err)
	}

	scenario, err := s.scenarioManager.CreateScenario(twin, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create scenario: %w", err)
	}

	return scenario, nil
}

// GetScenario retrieves a scenario by ID
func (s *Service) GetScenario(id string) (*models.Scenario, error) {
	scenario, err := s.store.GetScenario(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get scenario: %w", err)
	}
	return scenario, nil
}

// ListScenarios lists scenarios for a digital twin
func (s *Service) ListScenarios(twinID string) ([]*models.Scenario, error) {
	scenarios, err := s.store.ListScenariosByDigitalTwin(twinID)
	if err != nil {
		return nil, fmt.Errorf("failed to list scenarios: %w", err)
	}
	return scenarios, nil
}

// DeleteScenario deletes a scenario
func (s *Service) DeleteScenario(id string) error {
	if err := s.store.DeleteScenario(id); err != nil {
		return fmt.Errorf("failed to delete scenario: %w", err)
	}
	return nil
}

// CreateAction creates a new conditional action
func (s *Service) CreateAction(twinID string, req *models.ActionCreateRequest) (*models.Action, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	twin, err := s.store.GetDigitalTwin(twinID)
	if err != nil {
		return nil, fmt.Errorf("failed to get digital twin: %w", err)
	}

	action, err := s.actionManager.CreateAction(twin, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create action: %w", err)
	}

	return action, nil
}

// GetAction retrieves an action by ID
func (s *Service) GetAction(id string) (*models.Action, error) {
	action, err := s.store.GetAction(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get action: %w", err)
	}
	return action, nil
}

// ListActions lists actions for a digital twin
func (s *Service) ListActions(twinID string) ([]*models.Action, error) {
	actions, err := s.store.ListActionsByDigitalTwin(twinID)
	if err != nil {
		return nil, fmt.Errorf("failed to list actions: %w", err)
	}
	return actions, nil
}

// DeleteAction deletes an action
func (s *Service) DeleteAction(id string) error {
	if err := s.store.DeleteAction(id); err != nil {
		return fmt.Errorf("failed to delete action: %w", err)
	}
	return nil
}

// Helper functions

// ontologyClass holds parsed class information from an OWL/Turtle ontology
type ontologyClass struct {
	Name       string
	Label      string
	Properties []string
}

// initializeFromOntology parses the OWL Turtle content and records entity type metadata
func (s *Service) initializeFromOntology(twin *models.DigitalTwin, ont *models.Ontology) error {
	if ont.Content == "" {
		return nil
	}

	classes := parseOntologyClasses(ont.Content)
	if len(classes) == 0 {
		return nil
	}

	if twin.Metadata == nil {
		twin.Metadata = make(map[string]interface{})
	}

	entityTypes := make(map[string]interface{}, len(classes))
	for _, cls := range classes {
		entityTypes[cls.Name] = map[string]interface{}{
			"label":      cls.Label,
			"properties": cls.Properties,
		}
	}
	twin.Metadata["entity_types"] = entityTypes

	return s.store.SaveDigitalTwin(twin)
}

// parseOntologyClasses extracts class and property definitions from Turtle content
func parseOntologyClasses(turtleContent string) []ontologyClass {
	var classes []ontologyClass
	lines := strings.Split(turtleContent, "\n")

	classMap := make(map[string]*ontologyClass)
	domainMap := make(map[string]string) // property → class name

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Detect owl:Class declarations: ":ClassName a owl:Class" or ":ClassName rdf:type owl:Class"
		if (strings.Contains(line, "owl:Class") || strings.Contains(line, "owl:class")) &&
			(strings.Contains(line, " a ") || strings.Contains(line, "rdf:type")) {
			name := extractTurtleSubject(line)
			if name != "" && classMap[name] == nil {
				cls := &ontologyClass{Name: name, Label: name}
				classMap[name] = cls
			}
		}

		// Detect rdfs:label: ":ClassName rdfs:label "Label""
		if strings.Contains(line, "rdfs:label") {
			subject := extractTurtleSubject(line)
			label := extractStringLiteral(line)
			if subject != "" && label != "" {
				if cls, ok := classMap[subject]; ok {
					cls.Label = label
				}
			}
		}

		// Detect owl:DatatypeProperty declarations with rdfs:domain
		if strings.Contains(line, "owl:DatatypeProperty") || strings.Contains(line, "owl:ObjectProperty") {
			name := extractTurtleSubject(line)
			if name != "" {
				domainMap[name] = "" // register property, domain resolved later
			}
		}

		// Detect rdfs:domain: ":property rdfs:domain :ClassName"
		if strings.Contains(line, "rdfs:domain") {
			subject := extractTurtleSubject(line)
			domain := extractTurtleObject(line)
			if subject != "" && domain != "" {
				domainMap[subject] = domain
				if cls, ok := classMap[domain]; ok {
					if !containsString(cls.Properties, subject) {
						cls.Properties = append(cls.Properties, subject)
					}
				}
			}
		}
	}

	// Post-process: assign any unresolved properties to classes
	for prop, domain := range domainMap {
		if domain == "" {
			continue
		}
		if cls, ok := classMap[domain]; ok {
			if !containsString(cls.Properties, prop) {
				cls.Properties = append(cls.Properties, prop)
			}
		}
	}

	for _, cls := range classMap {
		classes = append(classes, *cls)
	}
	return classes
}

func extractTurtleSubject(line string) string {
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return ""
	}
	s := parts[0]
	// Strip leading colon if present
	s = strings.TrimPrefix(s, ":")
	// Strip trailing colon if prefix:local format
	if idx := strings.LastIndex(s, ":"); idx >= 0 {
		s = s[idx+1:]
	}
	return strings.Trim(s, "<>.,;")
}

func extractTurtleObject(line string) string {
	parts := strings.Fields(line)
	if len(parts) < 3 {
		return ""
	}
	s := parts[len(parts)-1]
	s = strings.TrimPrefix(s, ":")
	if idx := strings.LastIndex(s, ":"); idx >= 0 {
		s = s[idx+1:]
	}
	return strings.Trim(s, "<>.,;")
}

func extractStringLiteral(line string) string {
	start := strings.Index(line, `"`)
	if start < 0 {
		return ""
	}
	end := strings.Index(line[start+1:], `"`)
	if end < 0 {
		return ""
	}
	return line[start+1 : start+1+end]
}

func containsString(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

// syncFromStorage retrieves CIR data from a storage source and creates or
// merges entities in the digital twin.
//
// Entity resolution: before creating a new entity, the function checks whether
// an entity of the same type with the same key-field value already exists
// (e.g. from a previous pipeline that wrote to a different storage).  If found,
// the new attributes are merged into the existing entity rather than creating
// a duplicate.  The source storage ID is appended to the entity's source list.
func (s *Service) syncFromStorage(twin *models.DigitalTwin, ont *models.Ontology, storageID string) error {
	cirs, err := s.storageService.Retrieve(storageID, &models.CIRQuery{})
	if err != nil {
		return fmt.Errorf("failed to retrieve CIR data from storage %s: %w", storageID, err)
	}

	// Build ontology class names for entity type inference.
	var classNames []string
	if entityTypes, ok := twin.Metadata["entity_types"]; ok {
		if etMap, ok := entityTypes.(map[string]interface{}); ok {
			for name := range etMap {
				classNames = append(classNames, name)
			}
		}
	}

	// Pre-load existing entities per type into an in-memory index to avoid
	// N×M database queries during resolution.  The index is keyed by
	// (entityType, keyField, keyValue) → entity pointer.
	//
	// We build this lazily per entity type on first encounter.
	typeIndex := make(map[string]map[string]*models.Entity) // entityType → (keyField+":"+keyValue → entity)

	loadTypeIndex := func(entityType string) {
		if _, loaded := typeIndex[entityType]; loaded {
			return
		}
		existing, err := s.store.ListEntitiesByTypeInTwin(twin.ID, entityType)
		if err != nil {
			typeIndex[entityType] = make(map[string]*models.Entity)
			return
		}
		idx := make(map[string]*models.Entity, len(existing))
		for _, e := range existing {
			for _, kf := range detectKeyFields(e.Attributes) {
				kv := keyValue(e.Attributes[kf])
				if kv != "" {
					idx[kf+":"+kv] = e
				}
			}
		}
		typeIndex[entityType] = idx
	}

	now := time.Now().UTC()

	for _, cir := range cirs {
		dataMap, err := cir.GetDataAsMap()
		if err != nil {
			continue
		}

		entityType := inferEntityTypeFromCIR(cir, classNames)
		sourceID := cir.Source.URI

		attrs := make(map[string]interface{}, len(dataMap))
		for k, v := range dataMap {
			attrs[k] = v
		}

		// Attempt entity resolution: find an existing entity of the same type
		// that shares a key-field value with this CIR record.
		loadTypeIndex(entityType)
		idx := typeIndex[entityType]

		var resolved *models.Entity
		keyFields := detectKeyFields(attrs)
		for _, kf := range keyFields {
			kv := keyValue(attrs[kf])
			if kv == "" {
				continue
			}
			if existing, ok := idx[kf+":"+kv]; ok {
				resolved = existing
				break
			}
		}

		if resolved != nil {
			// Merge: add new attributes without overwriting existing values.
			for k, v := range attrs {
				if _, exists := resolved.Attributes[k]; !exists {
					resolved.Attributes[k] = v
				}
			}
			// Append this storage to the source list.
			appendSourceID(resolved, storageID)
			resolved.UpdatedAt = now

			if err := s.store.SaveEntity(resolved); err != nil {
				fmt.Printf("Warning: failed to merge entity from storage %s: %v\n", storageID, err)
			}
			// Update index with any new key values the merged attrs might expose.
			for _, kf := range keyFields {
				kv := keyValue(attrs[kf])
				if kv != "" {
					idx[kf+":"+kv] = resolved
				}
			}
		} else {
			// Create new entity.
			entity := &models.Entity{
				ID:            uuid.New().String(),
				DigitalTwinID: twin.ID,
				Type:          entityType,
				Attributes:    attrs,
				SourceDataID:  &sourceID,
				IsModified:    false,
				Modifications: make(map[string]interface{}),
				ComputedValues: map[string]interface{}{
					"source_ids": []interface{}{storageID},
				},
				CreatedAt: now,
				UpdatedAt: now,
			}
			if err := s.store.SaveEntity(entity); err != nil {
				fmt.Printf("Warning: failed to save entity from CIR: %v\n", err)
				continue
			}
			// Register in index so subsequent CIRs from this storage can match it.
			for _, kf := range keyFields {
				kv := keyValue(attrs[kf])
				if kv != "" {
					idx[kf+":"+kv] = entity
				}
			}
		}
	}

	return nil
}

// appendSourceID adds storageID to an entity's ComputedValues["source_ids"]
// list if it is not already present.
func appendSourceID(entity *models.Entity, storageID string) {
	if entity.ComputedValues == nil {
		entity.ComputedValues = make(map[string]interface{})
	}
	existing, _ := entity.ComputedValues["source_ids"].([]interface{})
	for _, v := range existing {
		if s, _ := v.(string); s == storageID {
			return // already present
		}
	}
	entity.ComputedValues["source_ids"] = append(existing, storageID)
}

// wireRelationships links entities of different types that share a common
// key-field value.  This is the runtime equivalent of foreign-key resolution:
// a GradeRecord with student_id=42 and an AttendanceRecord with student_id=42
// will be linked bidirectionally via an EntityRelationship of type
// "relatedByStudentId".
//
// The algorithm is data-agnostic: it discovers shared key fields by comparing
// attribute name sets across entity types, then joins on matching values.
func (s *Service) wireRelationships(twinID string) error {
	allEntities, err := s.store.ListEntitiesByDigitalTwin(twinID)
	if err != nil {
		return fmt.Errorf("failed to list entities: %w", err)
	}

	// Group entities by type.
	byType := make(map[string][]*models.Entity)
	for _, e := range allEntities {
		byType[e.Type] = append(byType[e.Type], e)
	}

	types := make([]string, 0, len(byType))
	for t := range byType {
		types = append(types, t)
	}

	if len(types) < 2 {
		return nil // nothing to link
	}

	// Track which entity pairs have already been linked to avoid duplicates.
	linked := make(map[string]bool)

	// For each pair of distinct entity types, find common key fields and
	// build index-based joins.
	for i := 0; i < len(types); i++ {
		for j := i + 1; j < len(types); j++ {
			typeA, typeB := types[i], types[j]
			entitiesA := byType[typeA]
			entitiesB := byType[typeB]

			// Find key field names that appear in both entity types.
			sharedKeys := commonKeyFieldNames(entitiesA, entitiesB)
			if len(sharedKeys) == 0 {
				continue
			}

			// Build value index for type B: field → value → entity
			bIndex := make(map[string]map[string]*models.Entity) // field → value → entity
			for _, e := range entitiesB {
				for _, kf := range sharedKeys {
					kv := keyValue(e.Attributes[kf])
					if kv == "" {
						continue
					}
					if bIndex[kf] == nil {
						bIndex[kf] = make(map[string]*models.Entity)
					}
					bIndex[kf][kv] = e
				}
			}

			// Match entities from type A against the index.
			for _, eA := range entitiesA {
				for _, kf := range sharedKeys {
					kv := keyValue(eA.Attributes[kf])
					if kv == "" {
						continue
					}
					eB, ok := bIndex[kf][kv]
					if !ok {
						continue
					}

					relType := "relatedBy" + toCamelCaseRel(kf)

					// A → B
					fwdKey := eA.ID + "|" + eB.ID + "|" + relType
					if !linked[fwdKey] {
						linked[fwdKey] = true
						if !hasRelationship(eA, relType, eB.ID) {
							eA.Relationships = append(eA.Relationships, &models.EntityRelationship{
								Type:       relType,
								TargetID:   eB.ID,
								TargetType: typeB,
							})
						}
					}
					// B → A (bidirectional)
					bwdKey := eB.ID + "|" + eA.ID + "|" + relType
					if !linked[bwdKey] {
						linked[bwdKey] = true
						if !hasRelationship(eB, relType, eA.ID) {
							eB.Relationships = append(eB.Relationships, &models.EntityRelationship{
								Type:       relType,
								TargetID:   eA.ID,
								TargetType: typeA,
							})
						}
					}
				}
			}

			// Persist updated entities.
			for _, e := range entitiesA {
				if err := s.store.SaveEntity(e); err != nil {
					fmt.Printf("Warning: failed to save wired entity %s: %v\n", e.ID, err)
				}
			}
			for _, e := range entitiesB {
				if err := s.store.SaveEntity(e); err != nil {
					fmt.Printf("Warning: failed to save wired entity %s: %v\n", e.ID, err)
				}
			}
		}
	}

	return nil
}

// commonKeyFieldNames returns key-like attribute names that appear in both
// entity type sets.  Only the first 20 entities per type are sampled to keep
// the operation bounded for large twins.
func commonKeyFieldNames(entitiesA, entitiesB []*models.Entity) []string {
	sampleA := entitiesA
	if len(sampleA) > 20 {
		sampleA = sampleA[:20]
	}
	sampleB := entitiesB
	if len(sampleB) > 20 {
		sampleB = sampleB[:20]
	}

	keysA := make(map[string]bool)
	for _, e := range sampleA {
		for _, kf := range detectKeyFields(e.Attributes) {
			keysA[kf] = true
		}
	}

	var shared []string
	seenB := make(map[string]bool)
	for _, e := range sampleB {
		for _, kf := range detectKeyFields(e.Attributes) {
			if keysA[kf] && !seenB[kf] {
				seenB[kf] = true
				shared = append(shared, kf)
			}
		}
	}
	return shared
}

// hasRelationship returns true if entity already has a relationship of the
// given type pointing to targetID.
func hasRelationship(e *models.Entity, relType, targetID string) bool {
	for _, r := range e.Relationships {
		if r.Type == relType && r.TargetID == targetID {
			return true
		}
	}
	return false
}

// toCamelCaseRel converts a field name like "student_id" → "StudentId"
// for use in relationship type names.
func toCamelCaseRel(s string) string {
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == '_' || r == '-' || r == ' '
	})
	var b strings.Builder
	for _, p := range parts {
		if len(p) == 0 {
			continue
		}
		b.WriteString(strings.ToUpper(p[:1]) + p[1:])
	}
	return b.String()
}

// detectKeyFields is re-exported here for use within the digitaltwin package.
// It returns attribute names that look like stable join keys.
func detectKeyFields(attributes map[string]interface{}) []string {
	keyNameSuffixes := []string{"id", "key", "code", "number", "uuid", "ref", "identifier", "no", "num", "email", "username", "token"}
	var keys []string
	for k := range attributes {
		lower := strings.ToLower(k)
		for _, suffix := range keyNameSuffixes {
			if strings.HasSuffix(lower, suffix) || lower == suffix {
				keys = append(keys, k)
				break
			}
		}
	}
	sort.Strings(keys)
	return keys
}

// keyValue returns a normalised string representation of an attribute value
// for equality comparison across data sources.
func keyValue(v interface{}) string {
	if v == nil {
		return ""
	}
	s := fmt.Sprintf("%v", v)
	if strings.Contains(s, ".") {
		s = strings.TrimRight(s, "0")
		s = strings.TrimRight(s, ".")
	}
	return strings.TrimSpace(s)
}

// inferEntityTypeFromCIR tries to infer the entity type from a CIR record
func inferEntityTypeFromCIR(cir *models.CIR, ontologyClasses []string) string {
	// Check CIR parameter first
	if v, ok := cir.GetParameter("entity_type"); ok {
		if s, ok := v.(string); ok && s != "" {
			return s
		}
	}

	// Match data keys against ontology class property names
	dataMap, err := cir.GetDataAsMap()
	if err != nil {
		return "unknown"
	}

	keys := make([]string, 0, len(dataMap))
	for k := range dataMap {
		keys = append(keys, strings.ToLower(k))
	}

	// Simple heuristic: look for class name hints in data keys
	for _, className := range ontologyClasses {
		lc := strings.ToLower(className)
		for _, k := range keys {
			if k == lc || k == "type" || k == "entity_type" {
				if v, ok := dataMap[k]; ok {
					if s, ok := v.(string); ok && strings.EqualFold(s, className) {
						return className
					}
				}
			}
		}
	}

	// Fall back to URI last path segment
	uri := cir.Source.URI
	if uri != "" {
		parts := strings.Split(strings.TrimRight(uri, "/"), "/")
		if len(parts) > 0 {
			last := parts[len(parts)-1]
			if last != "" {
				return last
			}
		}
	}

	return "default"
}

// populateEntityFromSource merges source CIR data with entity modifications.
//
// For merged entities (those produced by cross-source entity resolution, which
// store multiple source IDs in ComputedValues["source_ids"]), the Attributes
// map was already fully populated at sync time from all contributing sources.
// Re-fetching from a single CIR would lose the other sources' data, so we
// instead just apply any delta modifications on top of the stored merge.
//
// For single-source entities the original behaviour is preserved: re-fetch the
// CIR from storage and build a fresh merged view of source + modifications.
func (s *Service) populateEntityFromSource(entity *models.Entity) error {
	if entity.SourceDataID == nil {
		return nil
	}

	// Detect merged multi-source entities.
	if entity.ComputedValues != nil {
		if sourceIDs, ok := entity.ComputedValues["source_ids"].([]interface{}); ok {
			if len(sourceIDs) > 1 {
				// Already fully merged at sync time — only apply deltas.
				for k, v := range entity.Modifications {
					if entity.Attributes == nil {
						entity.Attributes = make(map[string]interface{})
					}
					entity.Attributes[k] = v
				}
				return nil
			}
		}
	}

	// Determine which single storage holds this entity.
	storageID := ""
	if entity.ComputedValues != nil {
		// New format: source_ids list with exactly one entry.
		if sourceIDs, ok := entity.ComputedValues["source_ids"].([]interface{}); ok && len(sourceIDs) == 1 {
			if sid, ok := sourceIDs[0].(string); ok {
				storageID = sid
			}
		}
		// Legacy format: singular storage_id string.
		if storageID == "" {
			if sid, ok := entity.ComputedValues["storage_id"].(string); ok {
				storageID = sid
			}
		}
	}
	if storageID == "" {
		return nil
	}

	// Retrieve all CIRs and find the matching one by URI.
	cirs, err := s.storageService.Retrieve(storageID, &models.CIRQuery{})
	if err != nil {
		return nil // non-fatal
	}

	for _, cir := range cirs {
		if cir.Source.URI == *entity.SourceDataID {
			sourceData, err := cir.GetDataAsMap()
			if err != nil {
				continue
			}
			// Build merged view: source + delta modifications.
			merged := make(map[string]interface{}, len(sourceData))
			for k, v := range sourceData {
				merged[k] = v
			}
			for k, v := range entity.Modifications {
				merged[k] = v
			}
			entity.Attributes = merged
			return nil
		}
	}
	return nil
}

// StartCacheEviction runs a background goroutine that periodically deletes expired predictions.
// Call this once after creating the service: go svc.StartCacheEviction(ctx)
func (s *Service) StartCacheEviction(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			twins, err := s.store.ListDigitalTwins()
			if err != nil {
				log.Printf("cache eviction: failed to list digital twins: %v", err)
				continue
			}
			for _, twin := range twins {
				if err := s.store.DeleteExpiredPredictions(twin.ID); err != nil {
					log.Printf("cache eviction: error for twin %s: %v", twin.ID, err)
				}
			}
			log.Printf("cache eviction: completed tick for %d digital twins", len(twins))
		}
	}
}

// GetRelatedEntities returns entities related to the given entity via the specified relationship type.
// If relationshipType is empty, all relationships are traversed.
func (s *Service) GetRelatedEntities(twinID, entityID, relationshipType string) ([]*models.Entity, error) {
	source, err := s.store.GetEntity(entityID)
	if err != nil {
		return nil, fmt.Errorf("failed to get source entity: %w", err)
	}

	var results []*models.Entity
	for _, rel := range source.Relationships {
		if relationshipType != "" && rel.Type != relationshipType {
			continue
		}
		target, err := s.store.GetEntity(rel.TargetID)
		if err != nil {
			// Log but continue
			log.Printf("GetRelatedEntities: failed to get entity %s: %v", rel.TargetID, err)
			continue
		}
		results = append(results, target)
	}
	return results, nil
}

// invalidatePredictionsForEntity removes cached predictions for an entity
func (s *Service) invalidatePredictionsForEntity(entityID string) error {
	predictions, err := s.store.ListPredictionsByEntity(entityID)
	if err != nil {
		return err
	}

	for _, pred := range predictions {
		if err := s.store.DeletePrediction(pred.ID); err != nil {
			return err
		}
	}

	return nil
}
