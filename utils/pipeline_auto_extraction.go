package utils

import (
	stdcontext "context"
	"fmt"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
)

// PipelineExtractionHandler handles automatic extraction when pipelines complete
type PipelineExtractionHandler struct {
	registry      *pipelines.PluginRegistry
	logger        *Logger
	pipelineStore *PipelineStore
}

// NewPipelineExtractionHandler creates a new extraction handler
func NewPipelineExtractionHandler(registry *pipelines.PluginRegistry, store *PipelineStore) *PipelineExtractionHandler {
	return &PipelineExtractionHandler{
		registry:      registry,
		logger:        GetLogger(),
		pipelineStore: store,
	}
}

// HandlePipelineCompleted handles pipeline completion events
func (h *PipelineExtractionHandler) HandlePipelineCompleted(event Event) error {
	h.logger.Info("Handling pipeline completion event",
		String("event_type", event.Type),
		String("source", event.Source))

	// Extract pipeline info from event payload
	pipelineFile, ok := event.Payload["pipeline_file"].(string)
	if !ok {
		h.logger.Warn("Pipeline completion event missing pipeline_file")
		return nil // Don't fail the event handler
	}

	pipelineName, _ := event.Payload["pipeline_name"].(string)
	context, _ := event.Payload["context"].(*pipelines.PluginContext)

	h.logger.Info("Pipeline completed",
		String("pipeline_file", pipelineFile),
		String("pipeline_name", pipelineName))

	// Get pipeline definition from store to check if auto-extraction is enabled
	var pipeline *PipelineDefinition
	var err error

	if pipelineName != "" {
		// Try to find pipeline by name
		pipelines, err := h.pipelineStore.ListPipelines(map[string]any{"name": pipelineName})
		if err != nil || len(pipelines) == 0 {
			h.logger.Warn("Could not find pipeline in store",
				String("pipeline_name", pipelineName),
				Error(err))
			return nil
		}
		pipeline = pipelines[0]
	} else {
		h.logger.Warn("No pipeline name provided, cannot check auto-extraction settings")
		return nil
	}

	// Check if auto-extraction is enabled
	if !pipeline.Metadata.AutoExtractOntology {
		h.logger.Debug("Auto-extraction not enabled for pipeline",
			String("pipeline_name", pipelineName))
		return nil
	}

	// Check if target ontology is specified
	if pipeline.Metadata.TargetOntologyID == "" {
		h.logger.Warn("Auto-extraction enabled but no target ontology specified",
			String("pipeline_name", pipelineName))
		return nil
	}

	h.logger.Info("Triggering auto-extraction",
		String("pipeline_name", pipelineName),
		String("target_ontology", pipeline.Metadata.TargetOntologyID))

	// Get extraction plugin
	plugin, err := h.registry.GetPlugin("Ontology", "extraction")
	if err != nil {
		h.logger.Error("Extraction plugin not available", err)
		return fmt.Errorf("extraction plugin not available: %w", err)
	}

	// Prepare extraction data from pipeline context
	extractionData := h.prepareExtractionData(context)

	// Create step config for extraction
	stepConfig := pipelines.StepConfig{
		Name:   "auto_extract_entities",
		Plugin: "Ontology.extraction",
		Config: map[string]any{
			"operation":       "extract",
			"ontology_id":     pipeline.Metadata.TargetOntologyID,
			"job_name":        fmt.Sprintf("Auto-extraction from %s", pipelineName),
			"source_type":     "pipeline_output",
			"extraction_type": "hybrid", // Use hybrid extraction by default
			"data":            extractionData,
		},
	}

	// Execute extraction plugin
	ctx := stdcontext.Background()
	globalContext := pipelines.NewPluginContext()
	result, err := plugin.ExecuteStep(ctx, stepConfig, globalContext)
	if err != nil {
		h.logger.Error("Auto-extraction failed", err,
			String("pipeline_name", pipelineName),
			String("ontology_id", pipeline.Metadata.TargetOntologyID))
		return fmt.Errorf("auto-extraction failed: %w", err)
	}

	// Log extraction results
	jobID, _ := result.Get("job_id")
	entitiesExtracted, _ := result.Get("entities_extracted")
	triplesGenerated, _ := result.Get("triples_generated")

	h.logger.Info("Auto-extraction completed successfully",
		String("pipeline_name", pipelineName),
		String("job_id", fmt.Sprintf("%v", jobID)),
		String("entities_extracted", fmt.Sprintf("%v", entitiesExtracted)),
		String("triples_generated", fmt.Sprintf("%v", triplesGenerated)))

	// Publish extraction completed event
	GetEventBus().Publish(Event{
		Type:   EventExtractionCompleted,
		Source: "pipeline-auto-extraction",
		Payload: map[string]any{
			"pipeline_name":      pipelineName,
			"ontology_id":        pipeline.Metadata.TargetOntologyID,
			"job_id":             jobID,
			"entities_extracted": entitiesExtracted,
			"triples_generated":  triplesGenerated,
		},
	})

	return nil
}

// prepareExtractionData converts pipeline context to extraction-ready data
func (h *PipelineExtractionHandler) prepareExtractionData(ctx *pipelines.PluginContext) map[string]any {
	if ctx == nil {
		return map[string]any{}
	}

	// Convert context to map
	data := make(map[string]any)
	for _, key := range ctx.Keys() {
		if value, exists := ctx.Get(key); exists {
			data[key] = value
		}
	}

	return data
}

// InitializePipelineAutoExtraction sets up automatic extraction for pipelines
// This should be called during application startup
func InitializePipelineAutoExtraction(registry *pipelines.PluginRegistry, store *PipelineStore) {
	handler := NewPipelineExtractionHandler(registry, store)

	// Subscribe to pipeline completion events
	GetEventBus().Subscribe(EventPipelineCompleted, handler.HandlePipelineCompleted)

	GetLogger().Info("Pipeline auto-extraction initialized")
}
