package ontology

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
	AI "github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/AI"
	knowledgegraph "github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/KnowledgeGraph"
	"github.com/google/uuid"
)

// ExtractionPlugin handles entity extraction from various data sources
type ExtractionPlugin struct {
	db          *sql.DB
	tdb2Backend *knowledgegraph.TDB2Backend
	llmClient   AI.LLMClient
}

// NewExtractionPlugin creates a new extraction plugin
func NewExtractionPlugin(db *sql.DB, tdb2Backend *knowledgegraph.TDB2Backend, llmClient AI.LLMClient) *ExtractionPlugin {
	return &ExtractionPlugin{
		db:          db,
		tdb2Backend: tdb2Backend,
		llmClient:   llmClient,
	}
}

// ExecuteStep implements BasePlugin.ExecuteStep
func (p *ExtractionPlugin) ExecuteStep(ctx context.Context, stepConfig pipelines.StepConfig, globalContext *pipelines.PluginContext) (*pipelines.PluginContext, error) {
	operation, ok := stepConfig.Config["operation"].(string)
	if !ok {
		return nil, fmt.Errorf("operation not specified in config")
	}

	switch operation {
	case "extract":
		return p.handleExtract(ctx, stepConfig, globalContext)
	case "get_job":
		return p.handleGetJob(ctx, stepConfig, globalContext)
	case "list_jobs":
		return p.handleListJobs(ctx, stepConfig, globalContext)
	default:
		return nil, fmt.Errorf("unknown operation: %s", operation)
	}
}

// GetPluginType implements BasePlugin.GetPluginType
func (p *ExtractionPlugin) GetPluginType() string {
	return "Ontology"
}

// GetPluginName implements BasePlugin.GetPluginName
func (p *ExtractionPlugin) GetPluginName() string {
	return "extraction"
}

// ValidateConfig implements BasePlugin.ValidateConfig
func (p *ExtractionPlugin) ValidateConfig(config map[string]any) error {
	operation, ok := config["operation"].(string)
	if !ok {
		return fmt.Errorf("operation is required")
	}

	validOperations := []string{"extract", "get_job", "list_jobs"}
	valid := false
	for _, op := range validOperations {
		if operation == op {
			valid = true
			break
		}
	}

	if !valid {
		return fmt.Errorf("invalid operation: %s", operation)
	}

	// For extract operation, validate required fields
	if operation == "extract" {
		if _, ok := config["ontology_id"].(string); !ok {
			return fmt.Errorf("ontology_id is required for extract operation")
		}
		if _, ok := config["source_type"].(string); !ok {
			return fmt.Errorf("source_type is required for extract operation")
		}
	}

	return nil
}

// GetInputSchema returns the JSON Schema for agent-friendly ontology extraction
func (p *ExtractionPlugin) GetInputSchema() map[string]any {
	return map[string]any{
		"type":        "object",
		"description": "Extract entities and relationships from data according to the ontology schema. Supports CSV, JSON, text, and HTML sources with deterministic or LLM-powered extraction.",
		"properties": map[string]any{
			"ontology_id": map[string]any{
				"type":        "string",
				"description": "ID of the ontology schema to use for extraction",
			},
			"data": map[string]any{
				"type":        "object",
				"description": "Data to extract entities from",
			},
			"source_type": map[string]any{
				"type":        "string",
				"description": "Type of data source (csv, json, text, html)",
				"enum":        []string{"csv", "json", "text", "html"},
			},
			"extraction_type": map[string]any{
				"type":        "string",
				"description": "Extraction method (deterministic, llm, hybrid)",
				"enum":        []string{"deterministic", "llm", "hybrid"},
				"default":     "hybrid",
			},
			"job_name": map[string]any{
				"type":        "string",
				"description": "Name for the extraction job",
			},
			"operation": map[string]any{
				"type":        "string",
				"description": "Operation to perform (extract, get_job, list_jobs)",
				"enum":        []string{"extract", "get_job", "list_jobs"},
				"default":     "extract",
			},
		},
		"required": []string{"ontology_id", "data", "source_type"},
	}
}

// handleExtract performs entity extraction
func (p *ExtractionPlugin) handleExtract(ctx context.Context, stepConfig pipelines.StepConfig, globalContext *pipelines.PluginContext) (*pipelines.PluginContext, error) {
	// Extract configuration
	ontologyID, _ := stepConfig.Config["ontology_id"].(string)
	sourceType, _ := stepConfig.Config["source_type"].(string)
	extractionType, _ := stepConfig.Config["extraction_type"].(string)
	jobName, _ := stepConfig.Config["job_name"].(string)
	data := stepConfig.Config["data"] // Can be string, map, or array

	if jobName == "" {
		jobName = fmt.Sprintf("Extraction-%s", time.Now().Format("20060102-150405"))
	}

	// Default to hybrid extraction if not specified
	if extractionType == "" {
		extractionType = string(ExtractionHybrid)
	}

	// Create extraction job record
	jobID := uuid.New().String()
	job := &ExtractionJob{
		ID:                jobID,
		OntologyID:        ontologyID,
		JobName:           jobName,
		Status:            ExtractionPending,
		ExtractionType:    ExtractionType(extractionType),
		SourceType:        sourceType,
		EntitiesExtracted: 0,
		TriplesGenerated:  0,
		CreatedAt:         time.Now(),
	}

	// Insert job into database
	if err := p.createExtractionJob(ctx, job); err != nil {
		return nil, fmt.Errorf("failed to create extraction job: %w", err)
	}

	// Update job status to running
	job.Status = ExtractionRunning
	now := time.Now()
	job.StartedAt = &now
	if err := p.updateExtractionJobStatus(ctx, jobID, ExtractionRunning, nil); err != nil {
		return nil, fmt.Errorf("failed to update job status: %w", err)
	}

	// Load ontology context
	ontologyContext, err := p.loadOntologyContext(ctx, ontologyID)
	if err != nil {
		p.failJob(ctx, jobID, fmt.Sprintf("Failed to load ontology: %v", err))
		return nil, fmt.Errorf("failed to load ontology: %w", err)
	}

	// Create appropriate extractor
	config := ExtractionConfig{
		OntologyID:     ontologyID,
		SourceType:     sourceType,
		ExtractionType: ExtractionType(extractionType),
	}

	var extractor Extractor
	switch ExtractionType(extractionType) {
	case ExtractionDeterministic:
		extractor = NewDeterministicExtractor(config)
	case ExtractionLLM:
		if p.llmClient == nil {
			p.failJob(ctx, jobID, "LLM client not configured")
			return nil, fmt.Errorf("LLM client not available")
		}
		extractor = NewLLMExtractor(config, p.llmClient)
	case ExtractionHybrid:
		if p.llmClient == nil {
			// Fall back to deterministic only
			extractor = NewDeterministicExtractor(config)
		} else {
			extractor = NewHybridExtractor(config, p.llmClient)
		}
	default:
		p.failJob(ctx, jobID, fmt.Sprintf("Unknown extraction type: %s", extractionType))
		return nil, fmt.Errorf("unknown extraction type: %s", extractionType)
	}

	// Perform extraction
	result, err := extractor.Extract(data, ontologyContext)
	if err != nil {
		p.failJob(ctx, jobID, fmt.Sprintf("Extraction failed: %v", err))
		return nil, fmt.Errorf("extraction failed: %w", err)
	}

	// Store triples in TDB2
	if len(result.Triples) > 0 {
		kgTriples := make([]knowledgegraph.Triple, len(result.Triples))
		for i, t := range result.Triples {
			kgTriples[i] = knowledgegraph.Triple{
				Subject:   t.Subject,
				Predicate: t.Predicate,
				Object:    t.Object,
				Graph:     ontologyContext.Metadata.TDB2Graph,
			}
		}

		if err := p.tdb2Backend.InsertTriples(ctx, kgTriples); err != nil {
			fmt.Printf("ERROR: Failed to insert triples into TDB2: %v\n", err)
			p.failJob(ctx, jobID, fmt.Sprintf("Failed to store triples: %v", err))
			return nil, fmt.Errorf("failed to store triples in TDB2: %w", err)
		}
		fmt.Printf("INFO: Successfully inserted %d triples into TDB2 for ontology %s\n", len(kgTriples), ontologyID)
	}

	// Store extracted entities in database
	for _, entity := range result.Entities {
		if err := p.storeExtractedEntity(ctx, jobID, &entity); err != nil {
			// Log warning but don't fail the job
			fmt.Printf("Warning: failed to store entity %s: %v\n", entity.URI, err)
		}
	}

	// Update job with results
	completedAt := time.Now()
	job.Status = ExtractionCompleted
	job.CompletedAt = &completedAt
	job.EntitiesExtracted = result.EntitiesExtracted
	job.TriplesGenerated = result.TriplesGenerated

	if err := p.updateExtractionJob(ctx, job); err != nil {
		return nil, fmt.Errorf("failed to update job with results: %w", err)
	}

	// Update ontology status to "active" after successful extraction
	updateStatusQuery := `UPDATE ontologies SET status = 'active', updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	if _, err := p.db.ExecContext(ctx, updateStatusQuery, ontologyID); err != nil {
		// Log warning but don't fail the job since extraction succeeded
		fmt.Printf("Warning: failed to update ontology status to active for %s: %v\n", ontologyID, err)
	}

	// Return result in context
	resultContext := pipelines.NewPluginContext()
	resultContext.Set("job_id", jobID)
	resultContext.Set("status", string(ExtractionCompleted))
	resultContext.Set("entities_extracted", result.EntitiesExtracted)
	resultContext.Set("triples_generated", result.TriplesGenerated)
	resultContext.Set("confidence", result.Confidence)
	resultContext.Set("warnings", result.Warnings)

	return resultContext, nil
}

// handleGetJob retrieves an extraction job
func (p *ExtractionPlugin) handleGetJob(ctx context.Context, stepConfig pipelines.StepConfig, globalContext *pipelines.PluginContext) (*pipelines.PluginContext, error) {
	jobID, ok := stepConfig.Config["job_id"].(string)
	if !ok {
		return nil, fmt.Errorf("job_id is required")
	}

	job, err := p.getExtractionJob(ctx, jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to get extraction job: %w", err)
	}

	// Get extracted entities
	entities, err := p.getExtractedEntities(ctx, jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to get extracted entities: %w", err)
	}

	resultContext := pipelines.NewPluginContext()
	resultContext.Set("job", job)
	resultContext.Set("entities", entities)

	return resultContext, nil
}

// handleListJobs lists extraction jobs
func (p *ExtractionPlugin) handleListJobs(ctx context.Context, stepConfig pipelines.StepConfig, globalContext *pipelines.PluginContext) (*pipelines.PluginContext, error) {
	ontologyID, _ := stepConfig.Config["ontology_id"].(string)
	status, _ := stepConfig.Config["status"].(string)

	jobs, err := p.listExtractionJobs(ctx, ontologyID, status)
	if err != nil {
		return nil, fmt.Errorf("failed to list extraction jobs: %w", err)
	}

	// Ensure we return empty array instead of nil
	if jobs == nil {
		jobs = []*ExtractionJob{}
	}

	resultContext := pipelines.NewPluginContext()
	resultContext.Set("jobs", jobs)

	return resultContext, nil
}

// Database operations

func (p *ExtractionPlugin) createExtractionJob(ctx context.Context, job *ExtractionJob) error {
	metadataJSON, _ := json.Marshal(job.Metadata)

	query := `
		INSERT INTO extraction_jobs (id, ontology_id, pipeline_id, job_name, status, extraction_type, source_type, source_path, entities_extracted, triples_generated, error_message, metadata)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := p.db.ExecContext(ctx, query,
		job.ID,
		job.OntologyID,
		job.PipelineID,
		job.JobName,
		string(job.Status),
		string(job.ExtractionType),
		job.SourceType,
		job.SourcePath,
		job.EntitiesExtracted,
		job.TriplesGenerated,
		job.ErrorMessage,
		string(metadataJSON),
	)

	return err
}

func (p *ExtractionPlugin) updateExtractionJobStatus(ctx context.Context, jobID string, status ExtractionJobStatus, errorMsg *string) error {
	var query string
	var args []any

	if status == ExtractionRunning {
		query = `UPDATE extraction_jobs SET status = ?, started_at = CURRENT_TIMESTAMP WHERE id = ?`
		args = []any{string(status), jobID}
	} else if status == ExtractionCompleted || status == ExtractionFailed {
		if errorMsg != nil {
			query = `UPDATE extraction_jobs SET status = ?, completed_at = CURRENT_TIMESTAMP, error_message = ? WHERE id = ?`
			args = []any{string(status), *errorMsg, jobID}
		} else {
			query = `UPDATE extraction_jobs SET status = ?, completed_at = CURRENT_TIMESTAMP WHERE id = ?`
			args = []any{string(status), jobID}
		}
	} else {
		query = `UPDATE extraction_jobs SET status = ? WHERE id = ?`
		args = []any{string(status), jobID}
	}

	_, err := p.db.ExecContext(ctx, query, args...)
	return err
}

func (p *ExtractionPlugin) updateExtractionJob(ctx context.Context, job *ExtractionJob) error {
	query := `
		UPDATE extraction_jobs 
		SET status = ?, entities_extracted = ?, triples_generated = ?, completed_at = CURRENT_TIMESTAMP 
		WHERE id = ?
	`

	_, err := p.db.ExecContext(ctx, query,
		string(job.Status),
		job.EntitiesExtracted,
		job.TriplesGenerated,
		job.ID,
	)

	return err
}

func (p *ExtractionPlugin) getExtractionJob(ctx context.Context, jobID string) (*ExtractionJob, error) {
	job := &ExtractionJob{}
	var metadataJSON sql.NullString
	var startedAt, completedAt sql.NullTime

	query := `
		SELECT id, ontology_id, pipeline_id, job_name, status, extraction_type, source_type, source_path, 
		       entities_extracted, triples_generated, error_message, started_at, completed_at, created_at, metadata
		FROM extraction_jobs 
		WHERE id = ?
	`

	err := p.db.QueryRowContext(ctx, query, jobID).Scan(
		&job.ID,
		&job.OntologyID,
		&job.PipelineID,
		&job.JobName,
		&job.Status,
		&job.ExtractionType,
		&job.SourceType,
		&job.SourcePath,
		&job.EntitiesExtracted,
		&job.TriplesGenerated,
		&job.ErrorMessage,
		&startedAt,
		&completedAt,
		&job.CreatedAt,
		&metadataJSON,
	)

	if err != nil {
		return nil, err
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

	return job, nil
}

func (p *ExtractionPlugin) listExtractionJobs(ctx context.Context, ontologyID, status string) ([]*ExtractionJob, error) {
	var query string
	var args []any

	if ontologyID != "" && status != "" {
		query = `SELECT id, ontology_id, job_name, status, extraction_type, entities_extracted, triples_generated, created_at FROM extraction_jobs WHERE ontology_id = ? AND status = ? ORDER BY created_at DESC LIMIT 100`
		args = []any{ontologyID, status}
	} else if ontologyID != "" {
		query = `SELECT id, ontology_id, job_name, status, extraction_type, entities_extracted, triples_generated, created_at FROM extraction_jobs WHERE ontology_id = ? ORDER BY created_at DESC LIMIT 100`
		args = []any{ontologyID}
	} else if status != "" {
		query = `SELECT id, ontology_id, job_name, status, extraction_type, entities_extracted, triples_generated, created_at FROM extraction_jobs WHERE status = ? ORDER BY created_at DESC LIMIT 100`
		args = []any{status}
	} else {
		query = `SELECT id, ontology_id, job_name, status, extraction_type, entities_extracted, triples_generated, created_at FROM extraction_jobs ORDER BY created_at DESC LIMIT 100`
		args = []any{}
	}

	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []*ExtractionJob
	for rows.Next() {
		job := &ExtractionJob{}
		err := rows.Scan(
			&job.ID,
			&job.OntologyID,
			&job.JobName,
			&job.Status,
			&job.ExtractionType,
			&job.EntitiesExtracted,
			&job.TriplesGenerated,
			&job.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}

	return jobs, nil
}

func (p *ExtractionPlugin) storeExtractedEntity(ctx context.Context, jobID string, entity *Entity) error {
	propertiesJSON, _ := json.Marshal(entity.Properties)

	query := `
		INSERT INTO extracted_entities (job_id, entity_uri, entity_type, entity_label, confidence, source_text, properties)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	_, err := p.db.ExecContext(ctx, query,
		jobID,
		entity.URI,
		entity.Type,
		entity.Label,
		entity.Confidence,
		entity.SourceText,
		string(propertiesJSON),
	)

	return err
}

func (p *ExtractionPlugin) getExtractedEntities(ctx context.Context, jobID string) ([]ExtractedEntity, error) {
	query := `
		SELECT id, job_id, entity_uri, entity_type, entity_label, confidence, source_text, properties, created_at
		FROM extracted_entities 
		WHERE job_id = ?
		ORDER BY created_at DESC
	`

	rows, err := p.db.QueryContext(ctx, query, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entities []ExtractedEntity
	for rows.Next() {
		entity := ExtractedEntity{}
		var propertiesJSON string

		err := rows.Scan(
			&entity.ID,
			&entity.JobID,
			&entity.EntityURI,
			&entity.EntityType,
			&entity.EntityLabel,
			&entity.Confidence,
			&entity.SourceText,
			&propertiesJSON,
			&entity.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		json.Unmarshal([]byte(propertiesJSON), &entity.Properties)
		entities = append(entities, entity)
	}

	return entities, nil
}

func (p *ExtractionPlugin) failJob(ctx context.Context, jobID, errorMsg string) {
	p.updateExtractionJobStatus(ctx, jobID, ExtractionFailed, &errorMsg)
}

// loadOntologyContext loads full ontology context including classes and properties
func (p *ExtractionPlugin) loadOntologyContext(ctx context.Context, ontologyID string) (*OntologyContext, error) {
	// Load ontology metadata
	metadata, err := p.getOntologyMetadata(ctx, ontologyID)
	if err != nil {
		return nil, fmt.Errorf("failed to load ontology metadata: %w", err)
	}

	// Load classes
	classes, err := p.getOntologyClasses(ctx, ontologyID)
	if err != nil {
		return nil, fmt.Errorf("failed to load ontology classes: %w", err)
	}

	// Load properties
	properties, err := p.getOntologyProperties(ctx, ontologyID)
	if err != nil {
		return nil, fmt.Errorf("failed to load ontology properties: %w", err)
	}

	// Infer base URI from ontology or use default
	baseURI := metadata.TDB2Graph
	if baseURI == "" {
		baseURI = fmt.Sprintf("http://mimir-aip.io/ontology/%s", ontologyID)
	}

	return &OntologyContext{
		Metadata:   metadata,
		BaseURI:    baseURI,
		Classes:    classes,
		Properties: properties,
	}, nil
}

func (p *ExtractionPlugin) getOntologyMetadata(ctx context.Context, ontologyID string) (*OntologyMetadata, error) {
	metadata := &OntologyMetadata{}

	query := `
		SELECT id, name, description, version, file_path, tdb2_graph, format, status, created_at, updated_at, created_by
		FROM ontologies 
		WHERE id = ?
	`

	var createdBy sql.NullString
	err := p.db.QueryRowContext(ctx, query, ontologyID).Scan(
		&metadata.ID,
		&metadata.Name,
		&metadata.Description,
		&metadata.Version,
		&metadata.FilePath,
		&metadata.TDB2Graph,
		&metadata.Format,
		&metadata.Status,
		&metadata.CreatedAt,
		&metadata.UpdatedAt,
		&createdBy,
	)
	if err != nil {
		return nil, err
	}

	if createdBy.Valid {
		metadata.CreatedBy = createdBy.String
	}

	return metadata, nil
}

func (p *ExtractionPlugin) getOntologyClasses(ctx context.Context, ontologyID string) ([]OntologyClass, error) {
	query := `
		SELECT uri, label, description
		FROM ontology_classes 
		WHERE ontology_id = ?
	`

	rows, err := p.db.QueryContext(ctx, query, ontologyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var classes []OntologyClass
	for rows.Next() {
		class := OntologyClass{}
		var label, description sql.NullString

		err := rows.Scan(&class.URI, &label, &description)
		if err != nil {
			return nil, err
		}

		if label.Valid {
			class.Label = label.String
		}
		if description.Valid {
			class.Description = description.String
		}

		classes = append(classes, class)
	}

	return classes, nil
}

func (p *ExtractionPlugin) getOntologyProperties(ctx context.Context, ontologyID string) ([]OntologyProperty, error) {
	query := `
		SELECT uri, label, description, property_type
		FROM ontology_properties 
		WHERE ontology_id = ?
	`

	rows, err := p.db.QueryContext(ctx, query, ontologyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var properties []OntologyProperty
	for rows.Next() {
		prop := OntologyProperty{}
		var label, description sql.NullString

		err := rows.Scan(&prop.URI, &label, &description, &prop.PropertyType)
		if err != nil {
			return nil, err
		}

		if label.Valid {
			prop.Label = label.String
		}
		if description.Valid {
			prop.Description = description.String
		}

		properties = append(properties, prop)
	}

	return properties, nil
}
