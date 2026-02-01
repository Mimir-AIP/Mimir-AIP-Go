package ml

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/KnowledgeGraph"
	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/Storage"
	"github.com/Mimir-AIP/Mimir-AIP-Go/utils"
)

// AutoMLHandler handles automatic ML model training when extractions complete
type AutoMLHandler struct {
	storage     *storage.PersistenceBackend
	kgClient    *knowledgegraph.TDB2Backend
	autoTrainer *AutoTrainer
	logger      *utils.Logger
}

// NewAutoMLHandler creates a new auto-ML handler
func NewAutoMLHandler(store *storage.PersistenceBackend, kg *knowledgegraph.TDB2Backend) *AutoMLHandler {
	return &AutoMLHandler{
		storage:     store,
		kgClient:    kg,
		autoTrainer: NewAutoTrainer(store, kg),
		logger:      utils.GetLogger(),
	}
}

// HandleExtractionCompleted handles extraction.completed events
func (h *AutoMLHandler) HandleExtractionCompleted(event utils.Event) error {
	h.logger.Info("Handling extraction completion event",
		utils.String("event_type", event.Type),
		utils.String("source", event.Source))

	// Extract ontology info from event payload
	ontologyID, ok := event.Payload["ontology_id"].(string)
	if !ok || ontologyID == "" {
		h.logger.Warn("Extraction event missing ontology_id")
		return nil // Don't fail the event handler
	}

	entitiesExtracted, _ := event.Payload["entities_extracted"]
	triplesGenerated, _ := event.Payload["triples_generated"]

	h.logger.Info("Extraction completed",
		utils.String("ontology_id", ontologyID),
		utils.String("entities", fmt.Sprintf("%v", entitiesExtracted)),
		utils.String("triples", fmt.Sprintf("%v", triplesGenerated)))

	// Check if this ontology should auto-train models
	autoTrain, err := h.shouldAutoTrain(ontologyID)
	if err != nil {
		h.logger.Error("Failed to check auto-train settings", err,
			utils.String("ontology_id", ontologyID))
		return nil // Don't fail
	}

	if !autoTrain {
		h.logger.Debug("Auto-training not enabled for ontology",
			utils.String("ontology_id", ontologyID))
		return nil
	}

	h.logger.Info("Triggering automatic model training",
		utils.String("ontology_id", ontologyID))

	// Publish model.training.started event
	utils.GetEventBus().Publish(utils.Event{
		Type:   utils.EventModelTrainingStarted,
		Source: "auto-ml-handler",
		Payload: map[string]any{
			"ontology_id": ontologyID,
			"trigger":     "extraction_completed",
		},
	})

	// Configure auto-training options
	options := &AutoTrainOptions{
		EnableRegression:     true,
		EnableClassification: true,
		EnableMonitoring:     true,
		MinConfidence:        0.6, // Only train models with >60% confidence
		MaxModels:            10,  // Limit to prevent resource exhaustion
		ForceAll:             false,
	}

	// Train models from ontology (uses KGDataExtractor internally)
	ctx := context.Background()
	result, err := h.autoTrainer.TrainFromOntology(ctx, ontologyID, options)
	if err != nil {
		h.logger.Error("Auto-training failed", err,
			utils.String("ontology_id", ontologyID))

		// Publish failure event
		utils.GetEventBus().Publish(utils.Event{
			Type:   utils.EventModelTrainingFailed,
			Source: "auto-ml-handler",
			Payload: map[string]any{
				"ontology_id": ontologyID,
				"error":       err.Error(),
			},
		})

		return fmt.Errorf("auto-training failed: %w", err)
	}

	h.logger.Info("Auto-training completed successfully",
		utils.String("ontology_id", ontologyID),
		utils.Int("models_created", result.ModelsCreated),
		utils.Int("models_failed", result.ModelsFailed))

	// Enable auto-twin creation to continue the automation chain
	if err := h.enableAutoCreateTwins(ontologyID); err != nil {
		h.logger.Warn("Failed to enable auto_create_twins for ontology",
			utils.Error(err),
			utils.String("ontology_id", ontologyID))
		// Continue anyway - this shouldn't block the event publishing
	}

	// Publish model.training.completed event for each model
	for _, model := range result.TrainedModels {
		utils.GetEventBus().Publish(utils.Event{
			Type:   utils.EventModelTrainingCompleted,
			Source: "auto-ml-handler",
			Payload: map[string]any{
				"ontology_id":     ontologyID,
				"model_id":        model.ModelID,
				"target_property": model.TargetProperty,
				"model_type":      model.ModelType,
				"accuracy":        model.Accuracy,
				"r2_score":        model.R2Score,
				"sample_count":    model.SampleCount,
				"feature_count":   model.FeatureCount,
			},
		})
	}

	return nil
}

// shouldAutoTrain checks if automatic training is enabled for an ontology
func (h *AutoMLHandler) shouldAutoTrain(ontologyID string) (bool, error) {
	// Query ontology metadata for auto_train_models flag
	query := `
		SELECT auto_train_models
		FROM ontologies
		WHERE id = ?
	`

	var autoTrain sql.NullBool
	err := h.storage.GetDB().QueryRow(query, ontologyID).Scan(&autoTrain)
	if err == sql.ErrNoRows {
		// Ontology doesn't have auto_train setting, default to false
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to query ontology settings: %w", err)
	}

	if !autoTrain.Valid {
		return false, nil
	}

	return autoTrain.Bool, nil
}

// enableAutoCreateTwins sets the auto_create_twins flag to true for an ontology
func (h *AutoMLHandler) enableAutoCreateTwins(ontologyID string) error {
	query := `UPDATE ontologies SET auto_create_twins = 1 WHERE id = ?`
	_, err := h.storage.GetDB().Exec(query, ontologyID)
	if err != nil {
		return fmt.Errorf("failed to enable auto_create_twins: %w", err)
	}
	h.logger.Info("Enabled auto_create_twins for ontology",
		utils.String("ontology_id", ontologyID))
	return nil
}

// InitializeAutoMLHandler sets up automatic ML training
// This should be called during application startup
func InitializeAutoMLHandler(store *storage.PersistenceBackend, kg *knowledgegraph.TDB2Backend) {
	if store == nil || kg == nil {
		utils.GetLogger().Warn("Auto-ML handler not initialized (missing dependencies)")
		return
	}

	handler := NewAutoMLHandler(store, kg)

	// Subscribe to extraction completion events
	utils.GetEventBus().Subscribe(utils.EventExtractionCompleted, handler.HandleExtractionCompleted)

	utils.GetLogger().Info("Auto-ML handler initialized")
}
