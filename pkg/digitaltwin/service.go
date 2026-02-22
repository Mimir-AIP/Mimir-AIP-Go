package digitaltwin

import (
	"fmt"
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

// SyncWithStorage synchronizes digital twin entities with CIR data from storage
func (s *Service) SyncWithStorage(twinID string) error {
	twin, err := s.store.GetDigitalTwin(twinID)
	if err != nil {
		return fmt.Errorf("failed to get digital twin: %w", err)
	}

	// Get ontology to know entity types
	ont, err := s.ontologyService.GetOntology(twin.OntologyID)
	if err != nil {
		return fmt.Errorf("failed to get ontology: %w", err)
	}

	// Sync entities from storage configs
	if twin.Config != nil && len(twin.Config.StorageIDs) > 0 {
		for _, storageID := range twin.Config.StorageIDs {
			if err := s.syncFromStorage(twin, ont, storageID); err != nil {
				return fmt.Errorf("failed to sync from storage %s: %w", storageID, err)
			}
		}
	}

	// Update last sync time
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

// Predict runs ML model prediction on entity data
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

// initializeFromOntology populates digital twin structure from ontology blueprint
func (s *Service) initializeFromOntology(twin *models.DigitalTwin, ont *models.Ontology) error {
	// Parse ontology to extract entity types
	// For now, this is a simplified version
	// In a full implementation, we would parse the OWL/Turtle content
	// and create entity templates based on class definitions

	// This will be expanded when we implement the SPARQL engine
	// which will need to understand the ontology structure

	return nil
}

// syncFromStorage syncs entities from a storage source
func (s *Service) syncFromStorage(twin *models.DigitalTwin, ont *models.Ontology, storageID string) error {
	// Get storage config
	_, err := s.storageService.GetStorageConfig(storageID)
	if err != nil {
		return fmt.Errorf("failed to get storage config: %w", err)
	}

	// List CIR data from storage
	// Note: storage.Service doesn't have ListData method
	// In a full implementation, we would:
	// 1. Use storage plugin to retrieve all CIR data
	// 2. Parse entity types from CIR context
	// 3. Create/update entities with source references

	// For now, this is a placeholder for the sync logic
	// The actual implementation would require extending the storage service
	// with a method to list all data from a storage config

	return nil
}

// populateEntityFromSource populates entity attributes from source CIR data
func (s *Service) populateEntityFromSource(entity *models.Entity) error {
	if entity.SourceDataID == nil {
		return nil
	}

	// Get CIR data from storage
	// We need to find which storage config has this data
	// For now, we'll skip this and rely on synced data
	// In a full implementation, we would:
	// 1. Track which storage config each entity came from
	// 2. Retrieve the CIR data
	// 3. Merge with modifications

	return nil
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
