package utils

import (
	"context"
	"fmt"
	"time"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
)

// PipelineTriggerConfig defines which pipeline to run when an anomaly is detected
type PipelineTriggerConfig struct {
	// PipelineID is the specific pipeline to execute (optional - if empty, searches by tag)
	PipelineID string `json:"pipeline_id,omitempty"`
	// Tag is the tag to search for when PipelineID is empty (e.g., "alert", "export")
	Tag string `json:"tag,omitempty"`
	// Context to pass to the pipeline (merged with anomaly data)
	Context map[string]any `json:"context,omitempty"`
}

// AnomalyPipelineTrigger handles triggering pipelines on anomaly detection
type AnomalyPipelineTrigger struct {
	registry      *pipelines.PluginRegistry
	pipelineStore *PipelineStore
	logger        *Logger
}

// NewAnomalyPipelineTrigger creates a new anomaly pipeline trigger
func NewAnomalyPipelineTrigger(registry *pipelines.PluginRegistry, store *PipelineStore) *AnomalyPipelineTrigger {
	return &AnomalyPipelineTrigger{
		registry:      registry,
		pipelineStore: store,
		logger:        GetLogger(),
	}
}

// HandleAnomalyDetected handles anomaly.detected events by triggering pipelines
func (t *AnomalyPipelineTrigger) HandleAnomalyDetected(event Event) error {
	t.logger.Info("Anomaly detected - triggering pipelines",
		String("event_type", event.Type),
		String("source", event.Source))

	// Extract anomaly info from event payload
	ontologyID, _ := event.Payload["ontology_id"].(string)
	entityID, _ := event.Payload["entity_id"].(string)
	metricName, _ := event.Payload["metric_name"].(string)
	alertType, _ := event.Payload["alert_type"].(string)
	severity, _ := event.Payload["severity"].(string)
	message, _ := event.Payload["message"].(string)
	value, _ := event.Payload["value"].(float64)

	// Get trigger configuration from event (or use defaults)
	config := t.getTriggerConfig(event.Payload)

	// Find pipeline to execute
	pipeline, err := t.findPipeline(config)
	if err != nil {
		t.logger.Error("Failed to find pipeline for anomaly", err,
			String("tag", config.Tag),
			String("pipeline_id", config.PipelineID))
		return nil // Don't fail - just log
	}

	if pipeline == nil {
		t.logger.Warn("No pipeline found for anomaly",
			String("tag", config.Tag),
			String("pipeline_id", config.PipelineID))
		return nil
	}

	t.logger.Info("Executing pipeline for anomaly",
		String("pipeline_id", pipeline.Metadata.ID),
		String("pipeline_name", pipeline.Metadata.Name),
		String("metric", metricName),
		String("severity", severity))

	// Build context with anomaly data
	ctx := context.Background()
	globalContext := pipelines.NewPluginContext()

	// Add anomaly data to context
	globalContext.Set("anomaly.ontology_id", ontologyID)
	globalContext.Set("anomaly.entity_id", entityID)
	globalContext.Set("anomaly.metric_name", metricName)
	globalContext.Set("anomaly.type", alertType)
	globalContext.Set("anomaly.severity", severity)
	globalContext.Set("anomaly.message", message)
	globalContext.Set("anomaly.value", value)
	globalContext.Set("anomaly.timestamp", time.Now().Format(time.RFC3339))
	globalContext.Set("anomaly.triggered_at", event.Timestamp.Format(time.RFC3339))

	// Add configured context (can override defaults)
	for k, v := range config.Context {
		globalContext.Set(k, v)
	}

	// Add original event payload
	for k, v := range event.Payload {
		if _, exists := globalContext.Get(k); !exists {
			globalContext.Set(k, v)
		}
	}

	// Execute pipeline
	_, err = ExecutePipelineWithRegistry(ctx, &pipeline.Config, t.registry)
	if err != nil {
		t.logger.Error("Pipeline execution failed for anomaly", err,
			String("pipeline_id", pipeline.Metadata.ID),
			String("metric", metricName))
		return nil // Don't fail the event handler
	}

	t.logger.Info("Pipeline executed successfully for anomaly",
		String("pipeline_id", pipeline.Metadata.ID),
		String("metric", metricName))

	return nil
}

// getTriggerConfig extracts trigger configuration from event payload or returns defaults
func (t *AnomalyPipelineTrigger) getTriggerConfig(payload map[string]any) *PipelineTriggerConfig {
	config := &PipelineTriggerConfig{
		Tag:     "alert", // Default tag to search for
		Context: make(map[string]any),
	}

	// Check if pipeline_id is specified in payload
	if pipelineID, ok := payload["trigger_pipeline_id"].(string); ok && pipelineID != "" {
		config.PipelineID = pipelineID
	}

	// Check if tag is specified in payload
	if tag, ok := payload["trigger_tag"].(string); ok && tag != "" {
		config.Tag = tag
	}

	// Check for additional context
	if ctx, ok := payload["trigger_context"].(map[string]any); ok {
		config.Context = ctx
	}

	return config
}

// findPipeline finds a pipeline to execute based on configuration
func (t *AnomalyPipelineTrigger) findPipeline(config *PipelineTriggerConfig) (*PipelineDefinition, error) {
	// If specific pipeline ID provided, use that
	if config.PipelineID != "" {
		return t.pipelineStore.GetPipeline(config.PipelineID)
	}

	// Otherwise, search by tag
	pipelines, err := t.pipelineStore.ListPipelines(map[string]any{
		"tags": []string{config.Tag},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list pipelines: %w", err)
	}

	if len(pipelines) == 0 {
		return nil, nil
	}

	// Return first enabled pipeline with matching tag
	for _, p := range pipelines {
		if p.Metadata.Enabled {
			return p, nil
		}
	}

	// No enabled pipelines found
	return nil, nil
}

// InitializeAnomalyPipelineTrigger sets up the anomaly pipeline trigger
// This should be called during application startup
func InitializeAnomalyPipelineTrigger(registry *pipelines.PluginRegistry, store *PipelineStore) {
	trigger := NewAnomalyPipelineTrigger(registry, store)

	// Subscribe to anomaly detection events
	GetEventBus().Subscribe(EventAnomalyDetected, trigger.HandleAnomalyDetected)

	GetLogger().Info("Anomaly pipeline trigger initialized")
}
