package utils

import (
	stdcontext "context"
	"fmt"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/Storage"
	"github.com/google/uuid"
)

// PipelineExtractionHandler handles automatic extraction when pipelines complete
type PipelineExtractionHandler struct {
	registry      *pipelines.PluginRegistry
	logger        *Logger
	pipelineStore *PipelineStore
	persistence   *storage.PersistenceBackend
}

// NewPipelineExtractionHandler creates a new extraction handler
func NewPipelineExtractionHandler(registry *pipelines.PluginRegistry, store *PipelineStore, persistence *storage.PersistenceBackend) *PipelineExtractionHandler {
	return &PipelineExtractionHandler{
		registry:      registry,
		logger:        GetLogger(),
		pipelineStore: store,
		persistence:   persistence,
	}
}

// HandlePipelineCompleted handles pipeline completion events
func (h *PipelineExtractionHandler) HandlePipelineCompleted(event Event) error {
	h.logger.Info("[AUTO-EXTRACTION] Handler invoked - START",
		String("event_type", event.Type),
		String("source", event.Source))

	// Extract pipeline info from event payload
	// Note: Event payload contains pipeline_id and pipeline_name (not pipeline_file)
	pipelineID, hasID := event.Payload["pipeline_id"].(string)
	pipelineName, hasName := event.Payload["pipeline_name"].(string)
	context, hasContext := event.Payload["context"].(*pipelines.PluginContext)
	h.logger.Info("[AUTO-EXTRACTION] Payload extracted",
		String("pipeline_id", pipelineID),
		Bool("has_id", hasID),
		String("pipeline_name", pipelineName),
		Bool("has_name", hasName),
		Bool("has_context", hasContext))

	if !hasID && !hasName {
		h.logger.Error("Pipeline completion event missing both pipeline_id and pipeline_name",
			fmt.Errorf("missing identification"))
		return fmt.Errorf("pipeline completion event missing identification fields")
	}

	h.logger.Info("Pipeline completed",
		String("pipeline_id", pipelineID),
		String("pipeline_name", pipelineName))

	// Get pipeline definition from store to check if auto-extraction is enabled
	var pipeline *PipelineDefinition

	// Try to find pipeline by ID first (more reliable)
	var lookupErr error
	if hasID && pipelineID != "" {
		h.logger.Info("[AUTO-EXTRACTION] Looking up pipeline by ID", String("pipeline_id", pipelineID))
		p, err := h.pipelineStore.GetPipeline(pipelineID)
		if err != nil {
			lookupErr = err
			h.logger.Error("[AUTO-EXTRACTION] Failed to get pipeline by ID", err, String("pipeline_id", pipelineID))
		} else if p == nil {
			h.logger.Warn("[AUTO-EXTRACTION] Pipeline not found by ID (nil result)", String("pipeline_id", pipelineID))
		} else {
			pipeline = p
			h.logger.Info("[AUTO-EXTRACTION] Successfully found pipeline by ID",
				String("pipeline_id", pipelineID),
				String("pipeline_name", p.Metadata.Name))
		}
	}

	// Fallback to lookup by name
	if pipeline == nil && hasName && pipelineName != "" {
		pipelines, err := h.pipelineStore.ListPipelines(map[string]any{"name": pipelineName})
		if err != nil {
			h.logger.Error("Failed to list pipelines by name", err,
				String("pipeline_name", pipelineName))
			return fmt.Errorf("failed to find pipeline: %w", err)
		}
		if len(pipelines) == 0 {
			h.logger.Error("Pipeline not found in store",
				fmt.Errorf("pipeline %s not found", pipelineName))
			return fmt.Errorf("pipeline not found: %s", pipelineName)
		}
		pipeline = pipelines[0]
	}

	if pipeline == nil {
		h.logger.Error("[AUTO-EXTRACTION] Could not find pipeline in store",
			fmt.Errorf("pipeline not found"),
			String("pipeline_id", pipelineID),
			String("pipeline_name", pipelineName),
			Error(lookupErr))
		return fmt.Errorf("pipeline not found in store")
	}

	h.logger.Info("[AUTO-EXTRACTION] Pipeline found - checking auto_extract_ontology flag",
		String("pipeline_id", pipeline.Metadata.ID),
		String("pipeline_name", pipeline.Metadata.Name),
		Bool("auto_extract_ontology", pipeline.Metadata.AutoExtractOntology),
		String("target_ontology_id", pipeline.Metadata.TargetOntologyID))

	// Check if auto-extraction is enabled
	if !pipeline.Metadata.AutoExtractOntology {
		h.logger.Info("[AUTO-EXTRACTION] Auto-extraction NOT enabled for pipeline - skipping",
			String("pipeline_name", pipelineName),
			String("pipeline_id", pipeline.Metadata.ID))
		return nil // Not an error, just not enabled
	}

	h.logger.Info("[AUTO-EXTRACTION] Auto-extraction IS enabled",
		String("pipeline_name", pipelineName),
		String("target_ontology_id", pipeline.Metadata.TargetOntologyID))

	// Check if target ontology is specified - if not, auto-create one
	targetOntologyID := pipeline.Metadata.TargetOntologyID
	if targetOntologyID == "" {
		h.logger.Info("Auto-extraction enabled but no target ontology specified - auto-creating",
			String("pipeline_name", pipelineName))

		// Create a new ontology for this pipeline's data
		newOntologyID, err := h.createOntologyForPipeline(pipeline)
		if err != nil {
			h.logger.Error("Failed to auto-create ontology", err,
				String("pipeline_name", pipelineName))
			return fmt.Errorf("failed to auto-create ontology: %w", err)
		}

		targetOntologyID = newOntologyID
		h.logger.Info("Auto-created ontology for pipeline",
			String("pipeline_name", pipelineName),
			String("ontology_id", targetOntologyID))

		// Update pipeline with new ontology ID
		pipeline.Metadata.TargetOntologyID = targetOntologyID
		_, err = h.pipelineStore.UpdatePipeline(pipeline.Metadata.ID, &pipeline.Metadata, &pipeline.Config, "auto-extraction")
		if err != nil {
			h.logger.Warn("Failed to update pipeline with new ontology ID",
				Error(err))
			// Continue anyway - we have the ID
		}
	}

	targetOntologyID = pipeline.Metadata.TargetOntologyID
	h.logger.Info("[AUTO-EXTRACTION] About to get extraction plugin",
		String("pipeline_name", pipelineName),
		String("target_ontology", targetOntologyID))

	// Get extraction plugin
	plugin, err := h.registry.GetPlugin("Ontology", "extraction")
	if err != nil {
		h.logger.Error("[AUTO-EXTRACTION] Extraction plugin not available", err)
		return fmt.Errorf("extraction plugin not available: %w", err)
	}

	h.logger.Info("[AUTO-EXTRACTION] Extraction plugin found - preparing data")

	// Prepare extraction data from pipeline context
	extractionData := h.prepareExtractionData(context)
	h.logger.Info("[AUTO-EXTRACTION] Extraction data prepared",
		String("data_type", fmt.Sprintf("%T", extractionData)))

	// Create step config for extraction
	stepConfig := pipelines.StepConfig{
		Name:   "auto_extract_entities",
		Plugin: "Ontology.extraction",
		Config: map[string]any{
			"operation":       "extract",
			"ontology_id":     targetOntologyID,
			"job_name":        fmt.Sprintf("Auto-extraction from %s", pipelineName),
			"source_type":     "json",
			"extraction_type": "hybrid", // Use hybrid extraction by default
			"data":            extractionData,
		},
	}

	h.logger.Info("[AUTO-EXTRACTION] About to execute extraction plugin",
		String("ontology_id", targetOntologyID),
		String("job_name", stepConfig.Config["job_name"].(string)))

	// Execute extraction plugin
	ctx := stdcontext.Background()
	globalContext := pipelines.NewPluginContext()
	result, err := plugin.ExecuteStep(ctx, stepConfig, globalContext)
	if err != nil {
		h.logger.Error("[AUTO-EXTRACTION] Plugin execution FAILED", err,
			String("pipeline_name", pipelineName),
			String("ontology_id", targetOntologyID))
		return fmt.Errorf("auto-extraction failed: %w", err)
	}

	// Log extraction results
	jobID, _ := result.Get("job_id")
	entitiesExtracted, _ := result.Get("entities_extracted")
	triplesGenerated, _ := result.Get("triples_generated")
	status, _ := result.Get("status")

	h.logger.Info("[AUTO-EXTRACTION] Plugin execution SUCCESS",
		String("pipeline_name", pipelineName),
		String("job_id", fmt.Sprintf("%v", jobID)),
		String("status", fmt.Sprintf("%v", status)),
		String("entities_extracted", fmt.Sprintf("%v", entitiesExtracted)),
		String("triples_generated", fmt.Sprintf("%v", triplesGenerated)))

	// Publish extraction completed event
	h.logger.Info("[AUTO-EXTRACTION] Publishing extraction.completed event")
	GetEventBus().Publish(Event{
		Type:   EventExtractionCompleted,
		Source: "pipeline-auto-extraction",
		Payload: map[string]any{
			"pipeline_name":      pipelineName,
			"ontology_id":        targetOntologyID,
			"job_id":             jobID,
			"entities_extracted": entitiesExtracted,
			"triples_generated":  triplesGenerated,
		},
	})
	h.logger.Info("[AUTO-EXTRACTION] extraction.completed event published - END")

	return nil
}

// prepareExtractionData converts pipeline context to extraction-ready data
// Returns data in a format suitable for extraction (array of objects or string)
func (h *PipelineExtractionHandler) prepareExtractionData(ctx *pipelines.PluginContext) any {
	if ctx == nil {
		h.logger.Debug("[AUTO-EXTRACTION] prepareExtractionData: context is nil")
		return []map[string]any{}
	}

	keys := ctx.Keys()
	h.logger.Debug("[AUTO-EXTRACTION] prepareExtractionData: context keys", Int("key_count", len(keys)))

	// Look for array data in context (most likely to be extractable records)
	for _, key := range keys {
		if value, exists := ctx.Get(key); exists {
			// Check if value is an array of maps (typical for CSV/JSON data)
			switch v := value.(type) {
			case []map[string]any:
				h.logger.Info("[AUTO-EXTRACTION] Found array data in context", String("key", key), Int("records", len(v)))
				return v
			case []any:
				// Convert []any to []map[string]any
				var records []map[string]any
				for _, item := range v {
					if m, ok := item.(map[string]any); ok {
						records = append(records, m)
					}
				}
				if len(records) > 0 {
					h.logger.Info("[AUTO-EXTRACTION] Found array data in context (converted)", String("key", key), Int("records", len(records)))
					return records
				}
			case string:
				// Could be CSV or JSON string
				if len(v) > 0 {
					h.logger.Info("[AUTO-EXTRACTION] Found string data in context", String("key", key), Int("length", len(v)))
					return v
				}
			case map[string]any:
				// Single record - wrap in array
				h.logger.Info("[AUTO-EXTRACTION] Found map data in context, wrapping in array", String("key", key))
				return []map[string]any{v}
			}
		}
	}

	h.logger.Warn("[AUTO-EXTRACTION] No suitable data found in context for extraction")
	return []map[string]any{}
}

// createOntologyForPipeline creates a new ontology for pipeline data extraction
func (h *PipelineExtractionHandler) createOntologyForPipeline(pipeline *PipelineDefinition) (string, error) {
	if h.persistence == nil {
		return "", fmt.Errorf("persistence backend not available")
	}

	// Generate a unique ontology ID based on pipeline name
	ontologyID := fmt.Sprintf("pipeline_%s_%s", pipeline.Metadata.Name, uuid.New().String()[:8])

	// Create the ontology
	ont := &storage.Ontology{
		ID:          ontologyID,
		Name:        fmt.Sprintf("Auto-created ontology for pipeline: %s", pipeline.Metadata.Name),
		Description: fmt.Sprintf("Automatically created ontology for pipeline %s (ID: %s)", pipeline.Metadata.Name, pipeline.Metadata.ID),
		Version:     "1.0.0",
		Format:      "turtle",
		Status:      "active",
		CreatedBy:   "auto-extraction",
		Metadata:    "{}",
	}

	// Persist the ontology
	ctx := stdcontext.Background()
	if err := h.persistence.CreateOntology(ctx, ont); err != nil {
		return "", fmt.Errorf("failed to create ontology in persistence: %w", err)
	}

	h.logger.Info("Created new ontology for pipeline",
		String("pipeline_name", pipeline.Metadata.Name),
		String("ontology_id", ontologyID))

	return ontologyID, nil
}

// InitializePipelineAutoExtraction sets up automatic extraction for pipelines
// This should be called during application startup
func InitializePipelineAutoExtraction(registry *pipelines.PluginRegistry, store *PipelineStore, persistence *storage.PersistenceBackend) {
	handler := NewPipelineExtractionHandler(registry, store, persistence)

	// Subscribe to pipeline completion events
	GetEventBus().Subscribe(EventPipelineCompleted, handler.HandlePipelineCompleted)

	GetLogger().Info("Pipeline auto-extraction initialized")
}
