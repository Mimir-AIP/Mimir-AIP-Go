package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
	"github.com/Mimir-AIP/Mimir-AIP-Go/utils"
)

// ChainReactionHandler orchestrates the automated response to data changes
type ChainReactionHandler struct {
	registry    *pipelines.PluginRegistry
	scheduler   *utils.Scheduler
	enabled     bool
	reactionLog []ChainReactionEvent
}

// ChainReactionEvent represents a step in the chain reaction
type ChainReactionEvent struct {
	ID        string                 `json:"id"`
	Timestamp time.Time              `json:"timestamp"`
	Trigger   string                 `json:"trigger"`
	Step      string                 `json:"step"`
	Status    string                 `json:"status"`
	Details   map[string]interface{} `json:"details"`
	Duration  time.Duration          `json:"duration"`
	NextStep  string                 `json:"next_step,omitempty"`
}

// NewChainReactionHandler creates a new chain reaction handler
func NewChainReactionHandler(registry *pipelines.PluginRegistry, scheduler *utils.Scheduler) *ChainReactionHandler {
	return &ChainReactionHandler{
		registry:    registry,
		scheduler:   scheduler,
		enabled:     true,
		reactionLog: make([]ChainReactionEvent, 0),
	}
}

// Initialize sets up all event listeners for chain reaction
func (h *ChainReactionHandler) Initialize(eventBus *utils.EventBus) {
	log.Println("🔗 Initializing Chain Reaction System...")

	// Subscribe to pipeline completion events
	eventBus.Subscribe(utils.EventPipelineCompleted, h.handlePipelineCompleted)

	// Subscribe to drift detected events
	eventBus.Subscribe("drift_detected", h.handleDriftDetected)

	// Subscribe to extraction completed events
	eventBus.Subscribe("extraction_completed", h.handleExtractionCompleted)

	// Subscribe to ML training completed events
	eventBus.Subscribe("training_completed", h.handleTrainingCompleted)

	// Subscribe to ontology version created events
	eventBus.Subscribe("ontology_version_created", h.handleOntologyVersionCreated)

	log.Println("✅ Chain Reaction System initialized and listening for events")
}

// handlePipelineCompleted responds to pipeline completion events
func (h *ChainReactionHandler) handlePipelineCompleted(event utils.Event) error {
	if !h.enabled {
		return nil
	}

	payload := event.Payload

	pipelineID, _ := payload["pipeline_id"].(string)
	pipelineName, _ := payload["pipeline_name"].(string)
	triggeredBy, _ := payload["triggered_by"].(string)

	// Log the event
	eventLog := ChainReactionEvent{
		ID:        fmt.Sprintf("reaction-%d", time.Now().Unix()),
		Timestamp: time.Now(),
		Trigger:   "pipeline_completed",
		Step:      "ingestion",
		Status:    "completed",
		Details: map[string]interface{}{
			"pipeline_id":   pipelineID,
			"pipeline_name": pipelineName,
			"triggered_by":  triggeredBy,
		},
	}
	h.reactionLog = append(h.reactionLog, eventLog)

	log.Printf("🔗 Chain Reaction: Pipeline %s completed (triggered by: %s)", pipelineName, triggeredBy)

	// If triggered by drift detection, extraction will be triggered by extraction handler
	// If it's a scheduled job, extraction will also be triggered
	// The extraction handler listens for pipeline completion and triggers auto-extraction
	return nil
}

// handleDriftDetected responds to drift detection events
func (h *ChainReactionHandler) handleDriftDetected(event utils.Event) error {
	if !h.enabled {
		return nil
	}

	payload := event.Payload

	ontologyID, _ := payload["ontology_id"].(string)
	pipelineID, _ := payload["pipeline_id"].(string)
	autoRemediate, _ := payload["auto_remediate"].(bool)

	driftData, ok := payload["drift"].(map[string]interface{})
	if !ok {
		return nil
	}

	field, _ := driftData["field"].(string)
	changePercent, _ := driftData["change_percent"].(float64)
	severity, _ := driftData["severity"].(string)

	// Log the drift event
	eventLog := ChainReactionEvent{
		ID:        fmt.Sprintf("drift-%d", time.Now().Unix()),
		Timestamp: time.Now(),
		Trigger:   "drift_detected",
		Step:      "drift_detection",
		Status:    "alert",
		Details: map[string]interface{}{
			"ontology_id":    ontologyID,
			"pipeline_id":    pipelineID,
			"field":          field,
			"change_percent": changePercent,
			"severity":       severity,
			"auto_remediate": autoRemediate,
		},
		NextStep: "extraction",
	}
	h.reactionLog = append(h.reactionLog, eventLog)

	log.Printf("🚨 Chain Reaction: Drift detected in %s (%s: %.2f%% change, severity: %s)",
		ontologyID, field, changePercent, severity)

	// Auto-remediate if enabled
	if autoRemediate && pipelineID != "" {
		log.Printf("🔄 Chain Reaction: Auto-remediation triggered for pipeline %s", pipelineID)

		// Trigger pipeline execution
		go h.triggerPipelineRemediation(pipelineID, ontologyID, eventLog.ID)
	}
	return nil
}

// triggerPipelineRemediation triggers pipeline execution for drift remediation
func (h *ChainReactionHandler) triggerPipelineRemediation(pipelineID, ontologyID, driftID string) {
	startTime := time.Now()

	// Get pipeline from store
	store := utils.GetPipelineStore()
	pipeline, err := store.GetPipeline(pipelineID)
	if err != nil {
		log.Printf("❌ Chain Reaction: Failed to get pipeline %s for remediation: %v", pipelineID, err)
		return
	}

	log.Printf("🔄 Chain Reaction: Executing pipeline %s for drift remediation...", pipeline.Name)

	// Execute pipeline
	ctx := context.Background()
	result, err := utils.ExecutePipelineWithRegistry(ctx, &pipeline.Config, h.registry)

	if err != nil {
		log.Printf("❌ Chain Reaction: Pipeline execution failed: %v", err)
		return
	}

	if result.Success {
		duration := time.Since(startTime)
		log.Printf("✅ Chain Reaction: Pipeline execution completed in %v", duration)

		// Publish pipeline completion event with drift context
		utils.GetEventBus().Publish(utils.Event{
			Type:   utils.EventPipelineCompleted,
			Source: "chain-reaction-remediation",
			Payload: map[string]interface{}{
				"pipeline_id":   pipelineID,
				"pipeline_name": pipeline.Name,
				"context":       result.Context,
				"triggered_by":  "drift_detection",
				"drift_id":      driftID,
				"remediation":   true,
			},
		})
	}
}

// handleExtractionCompleted responds to extraction completion events
func (h *ChainReactionHandler) handleExtractionCompleted(event utils.Event) error {
	if !h.enabled {
		return nil
	}

	payload := event.Payload

	ontologyID, _ := payload["ontology_id"].(string)
	newEntities, _ := payload["new_entities"].(int)
	updatedEntities, _ := payload["updated_entities"].(int)

	eventLog := ChainReactionEvent{
		ID:        fmt.Sprintf("extract-%d", time.Now().Unix()),
		Timestamp: time.Now(),
		Trigger:   "extraction_completed",
		Step:      "ontology_extraction",
		Status:    "completed",
		Details: map[string]interface{}{
			"ontology_id":      ontologyID,
			"new_entities":     newEntities,
			"updated_entities": updatedEntities,
		},
		NextStep: "versioning",
	}
	h.reactionLog = append(h.reactionLog, eventLog)

	log.Printf("📊 Chain Reaction: Extraction completed for %s (%d new, %d updated)",
		ontologyID, newEntities, updatedEntities)

	// If significant changes detected, versioning will be triggered automatically
	// The ontology versioning is handled by the ontology update handler
	return nil
}

// handleOntologyVersionCreated responds to ontology versioning events
func (h *ChainReactionHandler) handleOntologyVersionCreated(event utils.Event) error {
	if !h.enabled {
		return nil
	}

	payload := event.Payload

	ontologyID, _ := payload["ontology_id"].(string)
	versionID, _ := payload["version_id"].(string)
	versionNumber, _ := payload["version_number"].(int)

	eventLog := ChainReactionEvent{
		ID:        fmt.Sprintf("version-%d", time.Now().Unix()),
		Timestamp: time.Now(),
		Trigger:   "ontology_version_created",
		Step:      "versioning",
		Status:    "completed",
		Details: map[string]interface{}{
			"ontology_id":    ontologyID,
			"version_id":     versionID,
			"version_number": versionNumber,
		},
		NextStep: "ml_retrain",
	}
	h.reactionLog = append(h.reactionLog, eventLog)

	log.Printf("📦 Chain Reaction: Ontology version %d created for %s", versionNumber, ontologyID)

	// Auto-ML training will be triggered by the auto-ML handler
	// It subscribes to extraction events and triggers training
	return nil
}

// handleTrainingCompleted responds to ML training completion events
func (h *ChainReactionHandler) handleTrainingCompleted(event utils.Event) error {
	if !h.enabled {
		return nil
	}

	payload := event.Payload

	modelID, _ := payload["model_id"].(string)
	modelName, _ := payload["model_name"].(string)
	accuracy, _ := payload["accuracy"].(float64)
	ontologyID, _ := payload["ontology_id"].(string)

	eventLog := ChainReactionEvent{
		ID:        fmt.Sprintf("train-%d", time.Now().Unix()),
		Timestamp: time.Now(),
		Trigger:   "training_completed",
		Step:      "ml_training",
		Status:    "completed",
		Details: map[string]interface{}{
			"model_id":    modelID,
			"model_name":  modelName,
			"accuracy":    accuracy,
			"ontology_id": ontologyID,
		},
		NextStep: "twin_update",
	}
	h.reactionLog = append(h.reactionLog, eventLog)

	log.Printf("🤖 Chain Reaction: ML model %s trained (accuracy: %.2f%%)", modelName, accuracy)

	// Digital twin update will be triggered by the auto-ML handler
	// It calls enableAutoCreateTwins after training
	return nil
}

// GetReactionLog returns the chain reaction event log
func (h *ChainReactionHandler) GetReactionLog(limit int) []ChainReactionEvent {
	if limit <= 0 || limit > len(h.reactionLog) {
		return h.reactionLog
	}
	return h.reactionLog[len(h.reactionLog)-limit:]
}

// ClearReactionLog clears the event log
func (h *ChainReactionHandler) ClearReactionLog() {
	h.reactionLog = make([]ChainReactionEvent, 0)
}

// Enable enables the chain reaction system
func (h *ChainReactionHandler) Enable() {
	h.enabled = true
	log.Println("✅ Chain Reaction System enabled")
}

// Disable disables the chain reaction system
func (h *ChainReactionHandler) Disable() {
	h.enabled = false
	log.Println("⏸️  Chain Reaction System disabled")
}

// IsEnabled returns whether the chain reaction system is enabled
func (h *ChainReactionHandler) IsEnabled() bool {
	return h.enabled
}

// Global chain reaction handler instance
var chainReactionHandler *ChainReactionHandler

// InitializeChainReaction initializes the global chain reaction handler
func InitializeChainReaction(registry *pipelines.PluginRegistry, scheduler *utils.Scheduler, eventBus *utils.EventBus) {
	chainReactionHandler = NewChainReactionHandler(registry, scheduler)
	chainReactionHandler.Initialize(eventBus)
	log.Println("🚀 Global Chain Reaction System initialized")
}

// GetChainReactionHandler returns the global chain reaction handler
func GetChainReactionHandler() *ChainReactionHandler {
	return chainReactionHandler
}
