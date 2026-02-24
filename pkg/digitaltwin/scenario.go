package digitaltwin

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/mimir-aip/mimir-aip-go/pkg/metadatastore"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

// ScenarioManager handles scenario creation and management
type ScenarioManager struct {
	store           metadatastore.MetadataStore
	inferenceEngine *InferenceEngine
}

// NewScenarioManager creates a new scenario manager
func NewScenarioManager(store metadatastore.MetadataStore, inferenceEngine *InferenceEngine) *ScenarioManager {
	return &ScenarioManager{
		store:           store,
		inferenceEngine: inferenceEngine,
	}
}

// CreateScenario creates a new what-if scenario, optionally running predictions immediately
func (m *ScenarioManager) CreateScenario(twin *models.DigitalTwin, req *models.ScenarioCreateRequest) (*models.Scenario, error) {
	baseState := req.BaseState
	if baseState == "" {
		baseState = "current"
	}

	now := time.Now().UTC()
	scenario := &models.Scenario{
		ID:            uuid.New().String(),
		DigitalTwinID: twin.ID,
		Name:          req.Name,
		Description:   req.Description,
		BaseState:     baseState,
		Modifications: req.Modifications,
		Predictions:   make([]*models.ScenarioPrediction, 0),
		Status:        "active",
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := m.store.SaveScenario(scenario); err != nil {
		return nil, fmt.Errorf("failed to save scenario: %w", err)
	}

	if req.RunPredictions {
		if err := m.ApplyScenarioModifications(scenario, twin); err != nil {
			// Log but don't fail the scenario creation
			fmt.Printf("Warning: failed to run scenario predictions: %v\n", err)
		} else {
			if err := m.store.SaveScenario(scenario); err != nil {
				return nil, fmt.Errorf("failed to update scenario with predictions: %w", err)
			}
		}
	}

	return scenario, nil
}

// ApplyScenarioModifications runs ML predictions on the modified entity state
// for each modification in the scenario, and stores ScenarioPrediction results.
func (m *ScenarioManager) ApplyScenarioModifications(scenario *models.Scenario, twin *models.DigitalTwin) error {
	if len(scenario.Modifications) == 0 {
		return nil
	}

	// Collect unique entity IDs referenced by the modifications
	entityIDs := make(map[string]bool)
	for _, mod := range scenario.Modifications {
		entityIDs[mod.EntityID] = true
	}

	// For each affected entity, apply in-memory modifications and run predictions
	for entityID := range entityIDs {
		entity, err := m.store.GetEntity(entityID)
		if err != nil {
			fmt.Printf("Warning: scenario entity %s not found: %v\n", entityID, err)
			continue
		}

		// Clone the entity's attributes (in-memory only, no persistence)
		modifiedAttrs := cloneAttrs(entity.Attributes)

		// Apply all modifications targeting this entity
		for _, mod := range scenario.Modifications {
			if mod.EntityID != entityID {
				continue
			}
			modifiedAttrs[mod.Attribute] = mod.NewValue
		}

		// Build an input map for inference from the modified attributes
		inputMap := make(map[string]interface{}, len(modifiedAttrs))
		for k, v := range modifiedAttrs {
			inputMap[k] = v
		}

		// Run predictions against all trained models in the digital twin
		predictions := m.runPredictionsForInput(twin, entity, inputMap, modifiedAttrs)
		scenario.Predictions = append(scenario.Predictions, predictions...)
	}

	scenario.UpdatedAt = time.Now().UTC()
	return nil
}

// runPredictionsForInput runs all available ML models on the given input and returns
// ScenarioPrediction results for each model that succeeds.
func (m *ScenarioManager) runPredictionsForInput(
	twin *models.DigitalTwin,
	entity *models.Entity,
	inputMap map[string]interface{},
	modifiedAttrs map[string]interface{},
) []*models.ScenarioPrediction {
	var results []*models.ScenarioPrediction

	// List all models linked to the digital twin's project
	models_, err := m.inferenceEngine.mlService.ListProjectModels(twin.ProjectID)
	if err != nil || len(models_) == 0 {
		return results
	}

	for _, mlModel := range models_ {
		if mlModel.Status != models.ModelStatusTrained {
			continue
		}

		req := &models.PredictionRequest{
			ModelID:    mlModel.ID,
			EntityID:   entity.ID,
			EntityType: entity.Type,
			Input:      inputMap,
			UseCache:   false,
		}

		prediction, err := m.inferenceEngine.Predict(twin, req)
		if err != nil {
			fmt.Printf("Warning: scenario prediction failed for model %s: %v\n", mlModel.ID, err)
			continue
		}

		impact := describeImpact(mlModel.Name, prediction.Output, entity.Attributes, modifiedAttrs)

		results = append(results, &models.ScenarioPrediction{
			ModelID:    mlModel.ID,
			ModelName:  mlModel.Name,
			EntityID:   entity.ID,
			EntityType: entity.Type,
			Prediction: prediction.Output,
			Confidence: prediction.Confidence,
			Impact:     impact,
			Metadata: map[string]interface{}{
				"modified_attributes": modifiedAttrs,
			},
		})
	}

	return results
}

// describeImpact produces a human-readable summary of the prediction change
func describeImpact(modelName string, output interface{}, original, modified map[string]interface{}) string {
	// Build a list of what changed
	changed := make([]string, 0)
	for k, newVal := range modified {
		origVal, exists := original[k]
		if !exists || fmt.Sprintf("%v", origVal) != fmt.Sprintf("%v", newVal) {
			changed = append(changed, fmt.Sprintf("%s: %v → %v", k, origVal, newVal))
		}
	}

	if len(changed) == 0 {
		return fmt.Sprintf("Model %s predicts: %v", modelName, output)
	}
	return fmt.Sprintf("Model %s predicts %v after changing %v", modelName, output, changed)
}

// cloneAttrs makes a shallow copy of an attribute map
func cloneAttrs(attrs map[string]interface{}) map[string]interface{} {
	if attrs == nil {
		return make(map[string]interface{})
	}
	clone := make(map[string]interface{}, len(attrs))
	for k, v := range attrs {
		clone[k] = v
	}
	return clone
}
