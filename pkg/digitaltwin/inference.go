package digitaltwin

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/mimir-aip/mimir-aip-go/pkg/metadatastore"
	"github.com/mimir-aip/mimir-aip-go/pkg/mlmodel"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

// InferenceEngine handles ML model predictions for digital twin entities
type InferenceEngine struct {
	mlService *mlmodel.Service
	store     metadatastore.MetadataStore
}

// NewInferenceEngine creates a new inference engine
func NewInferenceEngine(mlService *mlmodel.Service, store metadatastore.MetadataStore) *InferenceEngine {
	return &InferenceEngine{
		mlService: mlService,
		store:     store,
	}
}

// Predict runs a single prediction
func (e *InferenceEngine) Predict(twin *models.DigitalTwin, req *models.PredictionRequest) (*models.Prediction, error) {
	// Check cache if enabled
	if req.UseCache {
		cached, err := e.getCachedPrediction(twin.ID, req.ModelID, req.EntityID)
		if err == nil && cached != nil {
			return cached, nil
		}
	}

	// Get the ML model
	model, err := e.mlService.GetModel(req.ModelID)
	if err != nil {
		return nil, fmt.Errorf("failed to get model: %w", err)
	}

	// Verify model is trained
	if model.Status != models.ModelStatusTrained {
		return nil, fmt.Errorf("model is not trained (status: %s)", model.Status)
	}

	// Run inference
	output, confidence, err := e.runInference(model, req.Input)
	if err != nil {
		return nil, fmt.Errorf("failed to run inference: %w", err)
	}

	// Create prediction
	now := time.Now().UTC()
	cacheTTL := 1800 // 30 minutes default
	if twin.Config != nil && twin.Config.PredictionCacheTTL > 0 {
		cacheTTL = twin.Config.PredictionCacheTTL
	}

	prediction := &models.Prediction{
		ID:             uuid.New().String(),
		DigitalTwinID:  twin.ID,
		ModelID:        req.ModelID,
		EntityID:       req.EntityID,
		EntityType:     req.EntityType,
		PredictionType: "point",
		Input:          req.Input,
		Output:         output,
		Confidence:     confidence,
		CachedAt:       now,
		ExpiresAt:      now.Add(time.Duration(cacheTTL) * time.Second),
		Metadata:       make(map[string]interface{}),
	}

	return prediction, nil
}

// BatchPredict runs batch predictions
func (e *InferenceEngine) BatchPredict(twin *models.DigitalTwin, req *models.BatchPredictionRequest) ([]*models.Prediction, error) {
	// Get the ML model
	model, err := e.mlService.GetModel(req.ModelID)
	if err != nil {
		return nil, fmt.Errorf("failed to get model: %w", err)
	}

	if model.Status != models.ModelStatusTrained {
		return nil, fmt.Errorf("model is not trained (status: %s)", model.Status)
	}

	predictions := make([]*models.Prediction, 0, len(req.Inputs))
	now := time.Now().UTC()
	cacheTTL := 1800
	if twin.Config != nil && twin.Config.PredictionCacheTTL > 0 {
		cacheTTL = twin.Config.PredictionCacheTTL
	}

	for _, input := range req.Inputs {
		output, confidence, err := e.runInference(model, input)
		if err != nil {
			// Log error but continue with other predictions
			fmt.Printf("Warning: failed to run inference: %v\n", err)
			continue
		}

		prediction := &models.Prediction{
			ID:             uuid.New().String(),
			DigitalTwinID:  twin.ID,
			ModelID:        req.ModelID,
			PredictionType: "batch",
			Input:          input,
			Output:         output,
			Confidence:     confidence,
			CachedAt:       now,
			ExpiresAt:      now.Add(time.Duration(cacheTTL) * time.Second),
			Metadata:       make(map[string]interface{}),
		}

		predictions = append(predictions, prediction)
	}

	return predictions, nil
}

// ModelArtifact represents a trained model artifact
type ModelArtifact struct {
	ModelType    string                 `json:"model_type"`
	FeatureNames []string               `json:"feature_names"`
	Parameters   map[string]interface{} `json:"parameters"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// runInference executes the actual inference using the trained model
func (e *InferenceEngine) runInference(model *models.MLModel, input map[string]interface{}) (interface{}, float64, error) {
	// Load the trained model artifact
	if model.ModelArtifactPath == "" {
		return nil, 0, fmt.Errorf("model artifact path is empty")
	}

	// Read model artifact
	artifactData, err := os.ReadFile(model.ModelArtifactPath)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read model artifact: %w", err)
	}

	var artifact ModelArtifact
	if err := json.Unmarshal(artifactData, &artifact); err != nil {
		return nil, 0, fmt.Errorf("failed to unmarshal artifact: %w", err)
	}

	// Convert input map to feature vector
	features := make([]float64, len(artifact.FeatureNames))
	for i, featureName := range artifact.FeatureNames {
		if val, ok := input[featureName]; ok {
			// Convert to float64
			switch v := val.(type) {
			case float64:
				features[i] = v
			case int:
				features[i] = float64(v)
			case int64:
				features[i] = float64(v)
			default:
				return nil, 0, fmt.Errorf("unsupported feature type for %s: %T", featureName, v)
			}
		} else {
			// Missing feature - use 0 as default
			features[i] = 0
		}
	}

	// Run prediction based on model type
	switch artifact.ModelType {
	case "decision_tree":
		prediction := e.predictDecisionTree(&artifact, features)
		confidence := 0.85 // Simplified - would calculate from tree depth/purity
		return prediction, confidence, nil

	case "random_forest":
		prediction := e.predictRandomForest(&artifact, features)
		confidence := 0.90
		return prediction, confidence, nil

	case "regression":
		prediction := e.predictRegression(&artifact, features)
		confidence := 1.0 // Regression always returns deterministic output
		return prediction, confidence, nil

	case "neural_network":
		prediction := e.predictNeuralNetwork(&artifact, features)
		confidence := 0.88
		return prediction, confidence, nil

	default:
		return nil, 0, fmt.Errorf("unsupported model type: %s", artifact.ModelType)
	}
}

// predictDecisionTree runs decision tree prediction
func (e *InferenceEngine) predictDecisionTree(artifact *ModelArtifact, features []float64) interface{} {
	// Simplified implementation - would traverse the actual tree structure
	// For now, return a placeholder
	if len(artifact.Parameters) > 0 {
		if pred, ok := artifact.Parameters["default_prediction"]; ok {
			return pred
		}
	}
	return 0.0
}

// predictRandomForest runs random forest prediction
func (e *InferenceEngine) predictRandomForest(artifact *ModelArtifact, features []float64) interface{} {
	// Simplified - would average predictions from multiple trees
	if len(artifact.Parameters) > 0 {
		if pred, ok := artifact.Parameters["default_prediction"]; ok {
			return pred
		}
	}
	return 0.0
}

// predictRegression runs linear regression prediction
func (e *InferenceEngine) predictRegression(artifact *ModelArtifact, features []float64) interface{} {
	// y = w1*x1 + w2*x2 + ... + b
	weights, ok := artifact.Parameters["weights"]
	if !ok {
		return 0.0
	}

	weightsSlice, ok := weights.([]interface{})
	if !ok {
		return 0.0
	}

	intercept := 0.0
	if b, ok := artifact.Parameters["intercept"]; ok {
		if bFloat, ok := b.(float64); ok {
			intercept = bFloat
		}
	}

	prediction := intercept
	for i, w := range weightsSlice {
		if i < len(features) {
			if wFloat, ok := w.(float64); ok {
				prediction += wFloat * features[i]
			}
		}
	}

	return prediction
}

// predictNeuralNetwork runs neural network prediction
func (e *InferenceEngine) predictNeuralNetwork(artifact *ModelArtifact, features []float64) interface{} {
	// Simplified - would perform forward pass through network layers
	if len(artifact.Parameters) > 0 {
		if pred, ok := artifact.Parameters["default_prediction"]; ok {
			return pred
		}
	}
	return 0.0
}

// getCachedPrediction retrieves a cached prediction if valid
func (e *InferenceEngine) getCachedPrediction(twinID, modelID, entityID string) (*models.Prediction, error) {
	predictions, err := e.store.ListPredictionsByDigitalTwin(twinID)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	for _, pred := range predictions {
		if pred.ModelID == modelID && pred.EntityID == entityID {
			// Check if cache is still valid
			if pred.ExpiresAt.After(now) {
				return pred, nil
			}
		}
	}

	return nil, nil
}
