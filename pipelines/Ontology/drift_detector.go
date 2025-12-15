package ontology

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/AI"
	knowledgegraph "github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/KnowledgeGraph"
)

// DriftDetector analyzes data to detect ontology drift
type DriftDetector struct {
	db          *sql.DB
	llmClient   AI.LLMClient
	tdb2Backend *knowledgegraph.TDB2Backend
}

// NewDriftDetector creates a new drift detector
func NewDriftDetector(db *sql.DB, llmClient AI.LLMClient, tdb2Backend *knowledgegraph.TDB2Backend) *DriftDetector {
	return &DriftDetector{
		db:          db,
		llmClient:   llmClient,
		tdb2Backend: tdb2Backend,
	}
}

// DetectDriftFromData analyzes extracted data to detect ontology drift
func (d *DriftDetector) DetectDriftFromData(ctx context.Context, ontologyID string, data interface{}, dataSource string) (int, error) {
	// Create drift detection record
	detectionID, err := d.createDriftDetection(ctx, ontologyID, "data_analysis", dataSource)
	if err != nil {
		return 0, fmt.Errorf("failed to create drift detection: %w", err)
	}

	// Get ontology context
	ontologyContext, err := d.getOntologyContext(ctx, ontologyID)
	if err != nil {
		d.updateDriftDetection(ctx, detectionID, DriftFailed, 0, fmt.Sprintf("failed to get ontology context: %v", err))
		return 0, fmt.Errorf("failed to get ontology context: %w", err)
	}

	// Analyze data for drift
	suggestions, err := d.analyzeDataForDrift(ctx, ontologyID, data, ontologyContext)
	if err != nil {
		d.updateDriftDetection(ctx, detectionID, DriftFailed, 0, fmt.Sprintf("analysis failed: %v", err))
		return 0, fmt.Errorf("drift analysis failed: %w", err)
	}

	// Store suggestions
	suggestionsCreated := 0
	for _, suggestion := range suggestions {
		if err := d.storeSuggestion(ctx, suggestion); err != nil {
			// Log error but continue
			continue
		}
		suggestionsCreated++
	}

	// Update drift detection status
	d.updateDriftDetection(ctx, detectionID, DriftCompleted, suggestionsCreated, "")

	return suggestionsCreated, nil
}

// DetectDriftFromExtractionJob analyzes extraction job results for drift
func (d *DriftDetector) DetectDriftFromExtractionJob(ctx context.Context, jobID string) (int, error) {
	// Get extraction job
	job, err := d.getExtractionJob(ctx, jobID)
	if err != nil {
		return 0, fmt.Errorf("failed to get extraction job: %w", err)
	}

	if job.Status != ExtractionCompleted {
		return 0, fmt.Errorf("extraction job %s is not completed (status: %s)", jobID, job.Status)
	}

	// Create drift detection record
	detectionID, err := d.createDriftDetection(ctx, job.OntologyID, "extraction_job", fmt.Sprintf("job:%s", jobID))
	if err != nil {
		return 0, fmt.Errorf("failed to create drift detection: %w", err)
	}

	// Get extracted entities
	entities, err := d.getExtractedEntities(ctx, jobID)
	if err != nil {
		d.updateDriftDetection(ctx, detectionID, DriftFailed, 0, fmt.Sprintf("failed to get entities: %v", err))
		return 0, fmt.Errorf("failed to get extracted entities: %w", err)
	}

	// Get ontology context
	ontologyContext, err := d.getOntologyContext(ctx, job.OntologyID)
	if err != nil {
		d.updateDriftDetection(ctx, detectionID, DriftFailed, 0, fmt.Sprintf("failed to get ontology context: %v", err))
		return 0, fmt.Errorf("failed to get ontology context: %w", err)
	}

	// Analyze entities for drift
	suggestions, err := d.analyzeEntitiesForDrift(ctx, job.OntologyID, entities, ontologyContext)
	if err != nil {
		d.updateDriftDetection(ctx, detectionID, DriftFailed, 0, fmt.Sprintf("analysis failed: %v", err))
		return 0, fmt.Errorf("drift analysis failed: %w", err)
	}

	// Store suggestions
	suggestionsCreated := 0
	for _, suggestion := range suggestions {
		if err := d.storeSuggestion(ctx, suggestion); err != nil {
			// Log error but continue
			continue
		}
		suggestionsCreated++
	}

	// Update drift detection status
	d.updateDriftDetection(ctx, detectionID, DriftCompleted, suggestionsCreated, "")

	return suggestionsCreated, nil
}

// MonitorKnowledgeGraphDrift periodically checks the knowledge graph for drift
func (d *DriftDetector) MonitorKnowledgeGraphDrift(ctx context.Context, ontologyID string) (int, error) {
	// Create drift detection record
	detectionID, err := d.createDriftDetection(ctx, ontologyID, "knowledge_graph_scan", "tdb2")
	if err != nil {
		return 0, fmt.Errorf("failed to create drift detection: %w", err)
	}

	// Get ontology metadata
	ontology, err := d.getOntology(ctx, ontologyID)
	if err != nil {
		d.updateDriftDetection(ctx, detectionID, DriftFailed, 0, fmt.Sprintf("failed to get ontology: %v", err))
		return 0, fmt.Errorf("failed to get ontology: %w", err)
	}

	// Query knowledge graph for entity patterns
	query := fmt.Sprintf(`
		SELECT DISTINCT ?type ?property ?value WHERE {
			GRAPH <%s> {
				?entity a ?type .
				?entity ?property ?value .
			}
		}
		LIMIT 1000
	`, ontology.TDB2Graph)

	result, err := d.tdb2Backend.QuerySPARQL(ctx, query)
	if err != nil {
		d.updateDriftDetection(ctx, detectionID, DriftFailed, 0, fmt.Sprintf("SPARQL query failed: %v", err))
		return 0, fmt.Errorf("failed to query knowledge graph: %w", err)
	}

	// Get ontology context
	ontologyContext, err := d.getOntologyContext(ctx, ontologyID)
	if err != nil {
		d.updateDriftDetection(ctx, detectionID, DriftFailed, 0, fmt.Sprintf("failed to get ontology context: %v", err))
		return 0, fmt.Errorf("failed to get ontology context: %w", err)
	}

	// Analyze SPARQL results for drift
	suggestions, err := d.analyzeSPARQLResultsForDrift(ctx, ontologyID, result, ontologyContext)
	if err != nil {
		d.updateDriftDetection(ctx, detectionID, DriftFailed, 0, fmt.Sprintf("analysis failed: %v", err))
		return 0, fmt.Errorf("drift analysis failed: %w", err)
	}

	// Store suggestions
	suggestionsCreated := 0
	for _, suggestion := range suggestions {
		if err := d.storeSuggestion(ctx, suggestion); err != nil {
			continue
		}
		suggestionsCreated++
	}

	d.updateDriftDetection(ctx, detectionID, DriftCompleted, suggestionsCreated, "")
	return suggestionsCreated, nil
}

// analyzeDataForDrift uses LLM to analyze raw data for drift patterns
func (d *DriftDetector) analyzeDataForDrift(ctx context.Context, ontologyID string, data interface{}, ontologyCtx *OntologyContext) ([]OntologySuggestion, error) {
	// Convert data to string representation
	dataStr, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data: %w", err)
	}

	// Build ontology summary for LLM
	ontologySummary := d.buildOntologySummary(ontologyCtx)

	// Create LLM prompt
	prompt := fmt.Sprintf(`You are an ontology expert analyzing data for schema drift.

Current Ontology Summary:
%s

Data Sample:
%s

Analyze the data and identify:
1. New entity types that don't exist in the current ontology
2. New properties that entities have but aren't defined in the ontology
3. Relationships between entities that aren't captured
4. Data patterns that suggest missing classes or properties

For each finding, provide:
- suggestion_type: "add_class", "add_property", "modify_class", or "modify_property"
- entity_type: "class" or "property"
- entity_uri: Suggested URI (e.g., "http://example.org/ontology#NewClass")
- confidence: 0.0-1.0 based on evidence strength
- reasoning: Explanation of why this change is suggested
- risk_level: "low", "medium", "high", or "critical"

Respond ONLY with a JSON array of suggestions in this exact format:
[
  {
    "suggestion_type": "add_class",
    "entity_type": "class",
    "entity_uri": "http://example.org/ontology#Product",
    "confidence": 0.85,
    "reasoning": "Data contains multiple records with product information (name, price, SKU) that aren't modeled",
    "risk_level": "medium"
  }
]

If no drift is detected, respond with an empty array: []`, ontologySummary, string(dataStr))

	// Call LLM
	if d.llmClient == nil {
		return nil, fmt.Errorf("LLM client not available")
	}

	response, err := d.llmClient.CompleteSimple(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("LLM request failed: %w", err)
	}

	// Parse response
	var rawSuggestions []struct {
		SuggestionType string  `json:"suggestion_type"`
		EntityType     string  `json:"entity_type"`
		EntityURI      string  `json:"entity_uri"`
		Confidence     float64 `json:"confidence"`
		Reasoning      string  `json:"reasoning"`
		RiskLevel      string  `json:"risk_level"`
	}

	// Extract JSON from response (may have markdown code blocks)
	jsonStr := d.extractJSON(response)
	if err := json.Unmarshal([]byte(jsonStr), &rawSuggestions); err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	// Convert to OntologySuggestion
	suggestions := make([]OntologySuggestion, 0, len(rawSuggestions))
	for _, raw := range rawSuggestions {
		suggestions = append(suggestions, OntologySuggestion{
			OntologyID:     ontologyID,
			SuggestionType: SuggestionType(raw.SuggestionType),
			EntityType:     raw.EntityType,
			EntityURI:      raw.EntityURI,
			Confidence:     raw.Confidence,
			Reasoning:      raw.Reasoning,
			Status:         SuggestionPending,
			RiskLevel:      RiskLevel(raw.RiskLevel),
			CreatedAt:      time.Now(),
		})
	}

	return suggestions, nil
}

// analyzeEntitiesForDrift analyzes extracted entities for drift patterns
func (d *DriftDetector) analyzeEntitiesForDrift(ctx context.Context, ontologyID string, entities []ExtractedEntity, ontologyCtx *OntologyContext) ([]OntologySuggestion, error) {
	suggestions := make([]OntologySuggestion, 0)

	// Track entity types and properties
	entityTypes := make(map[string]int)
	propertyUsage := make(map[string]map[string]int) // entityType -> property -> count

	for _, entity := range entities {
		entityTypes[entity.EntityType]++

		if propertyUsage[entity.EntityType] == nil {
			propertyUsage[entity.EntityType] = make(map[string]int)
		}

		// Count properties
		if entity.Properties != nil {
			for propName := range entity.Properties {
				propertyUsage[entity.EntityType][propName]++
			}
		}
	}

	// Check for unknown entity types
	for entityType, count := range entityTypes {
		if !d.isKnownClass(entityType, ontologyCtx) {
			suggestions = append(suggestions, OntologySuggestion{
				OntologyID:     ontologyID,
				SuggestionType: SuggestionAddClass,
				EntityType:     "class",
				EntityURI:      entityType,
				Confidence:     d.calculateConfidence(count, len(entities)),
				Reasoning:      fmt.Sprintf("Found %d instances of entity type '%s' which is not defined in the ontology", count, entityType),
				Status:         SuggestionPending,
				RiskLevel:      d.assessRisk(count, len(entities), "class"),
				CreatedAt:      time.Now(),
			})
		}
	}

	// Check for unknown properties
	for entityType, properties := range propertyUsage {
		for propName, count := range properties {
			if !d.isKnownProperty(propName, ontologyCtx) {
				suggestions = append(suggestions, OntologySuggestion{
					OntologyID:     ontologyID,
					SuggestionType: SuggestionAddProperty,
					EntityType:     "property",
					EntityURI:      propName,
					Confidence:     d.calculateConfidence(count, len(entities)),
					Reasoning:      fmt.Sprintf("Property '%s' used %d times on entity type '%s' but not defined in ontology", propName, count, entityType),
					Status:         SuggestionPending,
					RiskLevel:      d.assessRisk(count, len(entities), "property"),
					CreatedAt:      time.Now(),
				})
			}
		}
	}

	return suggestions, nil
}

// analyzeSPARQLResultsForDrift analyzes SPARQL query results for drift
func (d *DriftDetector) analyzeSPARQLResultsForDrift(ctx context.Context, ontologyID string, result *knowledgegraph.QueryResult, ontologyCtx *OntologyContext) ([]OntologySuggestion, error) {
	suggestions := make([]OntologySuggestion, 0)

	// Track entity types and properties from knowledge graph
	entityTypes := make(map[string]int)
	propertyUsage := make(map[string]int)

	// Parse SPARQL results
	if result.Bindings != nil {
		for _, binding := range result.Bindings {
			// Extract type
			if typeVal, ok := binding["type"]; ok {
				entityTypes[typeVal.Value]++
			}

			// Extract property
			if propVal, ok := binding["property"]; ok {
				propertyUsage[propVal.Value]++
			}
		}
	}

	totalBindings := len(result.Bindings)

	// Check for unknown classes
	for classURI, count := range entityTypes {
		if !d.isKnownClass(classURI, ontologyCtx) && !isBuiltInClass(classURI) {
			suggestions = append(suggestions, OntologySuggestion{
				OntologyID:     ontologyID,
				SuggestionType: SuggestionAddClass,
				EntityType:     "class",
				EntityURI:      classURI,
				Confidence:     d.calculateConfidence(count, totalBindings),
				Reasoning:      fmt.Sprintf("Class '%s' found %d times in knowledge graph but not defined in ontology", classURI, count),
				Status:         SuggestionPending,
				RiskLevel:      d.assessRisk(count, totalBindings, "class"),
				CreatedAt:      time.Now(),
			})
		}
	}

	// Check for unknown properties
	for propURI, count := range propertyUsage {
		if !d.isKnownProperty(propURI, ontologyCtx) && !isBuiltInProperty(propURI) {
			suggestions = append(suggestions, OntologySuggestion{
				OntologyID:     ontologyID,
				SuggestionType: SuggestionAddProperty,
				EntityType:     "property",
				EntityURI:      propURI,
				Confidence:     d.calculateConfidence(count, totalBindings),
				Reasoning:      fmt.Sprintf("Property '%s' used %d times in knowledge graph but not defined in ontology", propURI, count),
				Status:         SuggestionPending,
				RiskLevel:      d.assessRisk(count, totalBindings, "property"),
				CreatedAt:      time.Now(),
			})
		}
	}

	return suggestions, nil
}

// Helper functions

func (d *DriftDetector) isKnownClass(classURI string, ctx *OntologyContext) bool {
	for _, class := range ctx.Classes {
		if class.URI == classURI {
			return true
		}
	}
	return false
}

func (d *DriftDetector) isKnownProperty(propURI string, ctx *OntologyContext) bool {
	for _, prop := range ctx.Properties {
		if prop.URI == propURI {
			return true
		}
	}
	return false
}

func (d *DriftDetector) calculateConfidence(occurrences, total int) float64 {
	if total == 0 {
		return 0.5
	}
	// Confidence based on frequency: more occurrences = higher confidence
	ratio := float64(occurrences) / float64(total)
	// Scale to 0.5-0.95 range
	return 0.5 + (ratio * 0.45)
}

func (d *DriftDetector) assessRisk(occurrences, total int, entityType string) RiskLevel {
	ratio := float64(occurrences) / float64(total)

	// Classes are higher risk than properties
	if entityType == "class" {
		if ratio > 0.3 {
			return RiskLevelHigh
		} else if ratio > 0.1 {
			return RiskLevelMedium
		}
		return RiskLevelLow
	}

	// Properties
	if ratio > 0.5 {
		return RiskLevelHigh
	} else if ratio > 0.2 {
		return RiskLevelMedium
	}
	return RiskLevelLow
}

func (d *DriftDetector) buildOntologySummary(ctx *OntologyContext) string {
	var sb strings.Builder

	sb.WriteString("Classes:\n")
	for _, class := range ctx.Classes {
		sb.WriteString(fmt.Sprintf("- %s (%s)\n", class.URI, class.Label))
	}

	sb.WriteString("\nProperties:\n")
	for _, prop := range ctx.Properties {
		sb.WriteString(fmt.Sprintf("- %s (%s): %s -> %s\n", prop.URI, prop.Label, strings.Join(prop.Domain, ", "), strings.Join(prop.Range, ", ")))
	}

	return sb.String()
}

func (d *DriftDetector) extractJSON(text string) string {
	// Remove markdown code blocks if present
	text = strings.TrimSpace(text)
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	return strings.TrimSpace(text)
}

func isBuiltInClass(uri string) bool {
	builtIns := []string{
		"http://www.w3.org/2002/07/owl#Thing",
		"http://www.w3.org/2000/01/rdf-schema#Resource",
		"http://www.w3.org/2000/01/rdf-schema#Class",
		"http://www.w3.org/2002/07/owl#Class",
	}
	for _, built := range builtIns {
		if uri == built {
			return true
		}
	}
	return false
}

func isBuiltInProperty(uri string) bool {
	builtIns := []string{
		"http://www.w3.org/1999/02/22-rdf-syntax-ns#type",
		"http://www.w3.org/2000/01/rdf-schema#label",
		"http://www.w3.org/2000/01/rdf-schema#comment",
		"http://www.w3.org/2002/07/owl#sameAs",
	}
	for _, built := range builtIns {
		if uri == built {
			return true
		}
	}
	return false
}

// Database operations

func (d *DriftDetector) createDriftDetection(ctx context.Context, ontologyID, detectionType, dataSource string) (int, error) {
	query := `INSERT INTO drift_detections (ontology_id, detection_type, data_source, status, started_at) 
	          VALUES (?, ?, ?, ?, ?)`
	result, err := d.db.ExecContext(ctx, query, ontologyID, detectionType, dataSource, DriftRunning, time.Now())
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	return int(id), err
}

func (d *DriftDetector) updateDriftDetection(ctx context.Context, id int, status DriftDetectionStatus, suggestionsGenerated int, errorMsg string) error {
	query := `UPDATE drift_detections 
	          SET status = ?, suggestions_generated = ?, completed_at = ?, error_message = ? 
	          WHERE id = ?`
	_, err := d.db.ExecContext(ctx, query, status, suggestionsGenerated, time.Now(), errorMsg, id)
	return err
}

func (d *DriftDetector) storeSuggestion(ctx context.Context, suggestion OntologySuggestion) error {
	query := `INSERT INTO ontology_suggestions 
	          (ontology_id, suggestion_type, entity_type, entity_uri, confidence, reasoning, status, risk_level, created_at) 
	          VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := d.db.ExecContext(ctx, query,
		suggestion.OntologyID,
		suggestion.SuggestionType,
		suggestion.EntityType,
		suggestion.EntityURI,
		suggestion.Confidence,
		suggestion.Reasoning,
		suggestion.Status,
		suggestion.RiskLevel,
		suggestion.CreatedAt,
	)
	return err
}

func (d *DriftDetector) getOntology(ctx context.Context, ontologyID string) (*OntologyMetadata, error) {
	query := `SELECT id, name, description, version, file_path, tdb2_graph, format, status, created_at, updated_at, created_by, metadata 
	          FROM ontologies WHERE id = ?`
	row := d.db.QueryRowContext(ctx, query, ontologyID)

	var ont OntologyMetadata
	var metadataJSON sql.NullString
	err := row.Scan(&ont.ID, &ont.Name, &ont.Description, &ont.Version, &ont.FilePath, &ont.TDB2Graph,
		&ont.Format, &ont.Status, &ont.CreatedAt, &ont.UpdatedAt, &ont.CreatedBy, &metadataJSON)
	if err != nil {
		return nil, err
	}

	if metadataJSON.Valid {
		json.Unmarshal([]byte(metadataJSON.String), &ont.Metadata)
	}

	return &ont, nil
}

func (d *DriftDetector) getOntologyContext(ctx context.Context, ontologyID string) (*OntologyContext, error) {
	// Get classes
	classQuery := `SELECT uri, label, description, parent_uris FROM ontology_classes WHERE ontology_id = ?`
	classRows, err := d.db.QueryContext(ctx, classQuery, ontologyID)
	if err != nil {
		return nil, err
	}
	defer classRows.Close()

	classes := make([]OntologyClass, 0)
	for classRows.Next() {
		var class OntologyClass
		var parentURIsJSON sql.NullString
		err := classRows.Scan(&class.URI, &class.Label, &class.Description, &parentURIsJSON)
		if err != nil {
			continue
		}
		if parentURIsJSON.Valid {
			json.Unmarshal([]byte(parentURIsJSON.String), &class.ParentURIs)
		}
		classes = append(classes, class)
	}

	// Get properties
	propQuery := `SELECT uri, label, property_type, domain, range, description FROM ontology_properties WHERE ontology_id = ?`
	propRows, err := d.db.QueryContext(ctx, propQuery, ontologyID)
	if err != nil {
		return nil, err
	}
	defer propRows.Close()

	properties := make([]OntologyProperty, 0)
	for propRows.Next() {
		var prop OntologyProperty
		var domainJSON, rangeJSON sql.NullString
		err := propRows.Scan(&prop.URI, &prop.Label, &prop.PropertyType, &domainJSON, &rangeJSON, &prop.Description)
		if err != nil {
			continue
		}
		if domainJSON.Valid {
			json.Unmarshal([]byte(domainJSON.String), &prop.Domain)
		}
		if rangeJSON.Valid {
			json.Unmarshal([]byte(rangeJSON.String), &prop.Range)
		}
		properties = append(properties, prop)
	}

	// Get ontology metadata
	ontology, err := d.getOntology(ctx, ontologyID)
	if err != nil {
		return nil, err
	}

	return &OntologyContext{
		Metadata:   ontology,
		BaseURI:    ontology.TDB2Graph,
		Classes:    classes,
		Properties: properties,
	}, nil
}

func (d *DriftDetector) getExtractionJob(ctx context.Context, jobID string) (*ExtractionJob, error) {
	query := `SELECT id, ontology_id, pipeline_id, job_name, status, extraction_type, source_type, source_path,
	          entities_extracted, triples_generated, error_message, started_at, completed_at, created_at, metadata
	          FROM extraction_jobs WHERE id = ?`
	row := d.db.QueryRowContext(ctx, query, jobID)

	var job ExtractionJob
	var pipelineID, sourcePath, errorMessage, metadataJSON sql.NullString
	var startedAt, completedAt sql.NullTime
	err := row.Scan(&job.ID, &job.OntologyID, &pipelineID, &job.JobName, &job.Status, &job.ExtractionType,
		&job.SourceType, &sourcePath, &job.EntitiesExtracted, &job.TriplesGenerated, &errorMessage,
		&startedAt, &completedAt, &job.CreatedAt, &metadataJSON)
	if err != nil {
		return nil, err
	}

	if pipelineID.Valid {
		job.PipelineID = pipelineID.String
	}
	if sourcePath.Valid {
		job.SourcePath = sourcePath.String
	}
	if errorMessage.Valid {
		job.ErrorMessage = errorMessage.String
	}
	if startedAt.Valid {
		job.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		job.CompletedAt = &completedAt.Time
	}
	if metadataJSON.Valid {
		json.Unmarshal([]byte(metadataJSON.String), &job.Metadata)
	}

	return &job, nil
}

func (d *DriftDetector) getExtractedEntities(ctx context.Context, jobID string) ([]ExtractedEntity, error) {
	query := `SELECT id, job_id, entity_uri, entity_type, entity_label, confidence, source_text, properties, created_at
	          FROM extracted_entities WHERE job_id = ?`
	rows, err := d.db.QueryContext(ctx, query, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entities := make([]ExtractedEntity, 0)
	for rows.Next() {
		var entity ExtractedEntity
		var entityLabel, sourceText, propertiesJSON sql.NullString
		var confidence sql.NullFloat64
		err := rows.Scan(&entity.ID, &entity.JobID, &entity.EntityURI, &entity.EntityType, &entityLabel,
			&confidence, &sourceText, &propertiesJSON, &entity.CreatedAt)
		if err != nil {
			continue
		}

		if entityLabel.Valid {
			entity.EntityLabel = entityLabel.String
		}
		if confidence.Valid {
			entity.Confidence = confidence.Float64
		}
		if sourceText.Valid {
			entity.SourceText = sourceText.String
		}
		if propertiesJSON.Valid {
			json.Unmarshal([]byte(propertiesJSON.String), &entity.Properties)
		}

		entities = append(entities, entity)
	}

	return entities, nil
}

// DriftDetectionPlugin implements BasePlugin for drift detection
type DriftDetectionPlugin struct {
	detector *DriftDetector
}

// NewDriftDetectionPlugin creates a new drift detection plugin
func NewDriftDetectionPlugin(db *sql.DB, llmClient AI.LLMClient, tdb2Backend *knowledgegraph.TDB2Backend) *DriftDetectionPlugin {
	return &DriftDetectionPlugin{
		detector: NewDriftDetector(db, llmClient, tdb2Backend),
	}
}

// ExecuteStep implements BasePlugin.ExecuteStep
func (p *DriftDetectionPlugin) ExecuteStep(ctx context.Context, stepConfig pipelines.StepConfig, globalContext *pipelines.PluginContext) (*pipelines.PluginContext, error) {
	ontologyID, ok := stepConfig.Config["ontology_id"].(string)
	if !ok {
		return nil, fmt.Errorf("ontology_id is required")
	}

	resultContext := pipelines.NewPluginContext()

	// Check for extraction_job_id or data
	if jobID, ok := stepConfig.Config["extraction_job_id"].(string); ok {
		// Analyze extraction job
		suggestionsCount, err := p.detector.DetectDriftFromExtractionJob(ctx, jobID)
		if err != nil {
			return nil, err
		}
		resultContext.Set("suggestions_generated", suggestionsCount)
		resultContext.Set("source", fmt.Sprintf("extraction_job:%s", jobID))
	} else if data, ok := stepConfig.Config["data"]; ok {
		// Analyze raw data
		dataSource := "unknown"
		if ds, ok := stepConfig.Config["data_source"].(string); ok {
			dataSource = ds
		}
		suggestionsCount, err := p.detector.DetectDriftFromData(ctx, ontologyID, data, dataSource)
		if err != nil {
			return nil, err
		}
		resultContext.Set("suggestions_generated", suggestionsCount)
		resultContext.Set("source", dataSource)
	} else {
		// Default: monitor knowledge graph
		suggestionsCount, err := p.detector.MonitorKnowledgeGraphDrift(ctx, ontologyID)
		if err != nil {
			return nil, err
		}
		resultContext.Set("suggestions_generated", suggestionsCount)
		resultContext.Set("source", "knowledge_graph_scan")
	}

	return resultContext, nil
}

// GetPluginType implements BasePlugin.GetPluginType
func (p *DriftDetectionPlugin) GetPluginType() string {
	return "Ontology"
}

// GetPluginName implements BasePlugin.GetPluginName
func (p *DriftDetectionPlugin) GetPluginName() string {
	return "drift_detection"
}

// ValidateConfig implements BasePlugin.ValidateConfig
func (p *DriftDetectionPlugin) ValidateConfig(config map[string]any) error {
	if _, ok := config["ontology_id"].(string); !ok {
		return fmt.Errorf("ontology_id is required")
	}
	return nil
}
