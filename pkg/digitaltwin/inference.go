package digitaltwin

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/mimir-aip/mimir-aip-go/pkg/metadatastore"
	"github.com/mimir-aip/mimir-aip-go/pkg/mlmodel"
	mltraining "github.com/mimir-aip/mimir-aip-go/pkg/mlmodel/training"
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
	if req.UseCache {
		cached, err := e.getCachedPrediction(twin.ID, req.ModelID, req.EntityID)
		if err == nil && cached != nil {
			return cached, nil
		}
	}

	model, err := e.mlService.GetModel(req.ModelID)
	if err != nil {
		return nil, fmt.Errorf("failed to get model: %w", err)
	}

	if model.Status != models.ModelStatusTrained {
		return nil, fmt.Errorf("model is not trained (status: %s)", model.Status)
	}

	output, confidence, err := e.runInference(model, req.Input)
	if err != nil {
		return nil, fmt.Errorf("failed to run inference: %w", err)
	}

	now := time.Now().UTC()
	cacheTTL := 1800
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
			fmt.Printf("Warning: failed to run inference: %v\n", err)
			continue
		}

		predictions = append(predictions, &models.Prediction{
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
		})
	}

	return predictions, nil
}

// ModelArtifact represents a trained model artifact stored on disk
type ModelArtifact struct {
	ModelType    string                 `json:"model_type"`
	FeatureNames []string               `json:"feature_names"`
	Parameters   map[string]interface{} `json:"parameters"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// runInference executes the actual inference using the trained model artifact
func (e *InferenceEngine) runInference(model *models.MLModel, input map[string]interface{}) (interface{}, float64, error) {
	if model.ModelArtifactPath == "" {
		return nil, 0, fmt.Errorf("model artifact path is empty")
	}

	artifactData, err := os.ReadFile(model.ModelArtifactPath)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read model artifact: %w", err)
	}

	var artifact ModelArtifact
	if err := json.Unmarshal(artifactData, &artifact); err != nil {
		return nil, 0, fmt.Errorf("failed to unmarshal artifact: %w", err)
	}

	// Build feature vector from artifact's feature name ordering
	features := make([]float64, len(artifact.FeatureNames))
	for i, featureName := range artifact.FeatureNames {
		if val, ok := input[featureName]; ok {
			switch v := val.(type) {
			case float64:
				features[i] = v
			case int:
				features[i] = float64(v)
			case int64:
				features[i] = float64(v)
			case bool:
				if v {
					features[i] = 1
				}
			default:
				features[i] = 0
			}
		}
	}

	switch artifact.ModelType {
	case "decision_tree":
		prediction, err := e.predictDecisionTree(&artifact, features)
		if err != nil {
			return nil, 0, err
		}
		confidence := 0.85
		if model.PerformanceMetrics != nil && model.PerformanceMetrics.Accuracy > 0 {
			confidence = model.PerformanceMetrics.Accuracy
		}
		return prediction, confidence, nil

	case "random_forest":
		prediction, err := e.predictRandomForest(&artifact, features)
		if err != nil {
			return nil, 0, err
		}
		confidence := 0.90
		if model.PerformanceMetrics != nil && model.PerformanceMetrics.Accuracy > 0 {
			confidence = model.PerformanceMetrics.Accuracy
		}
		return prediction, confidence, nil

	case "regression":
		prediction := e.predictRegression(&artifact, features)
		confidence := 1.0
		if model.PerformanceMetrics != nil && model.PerformanceMetrics.R2Score > 0 {
			confidence = math.Max(0, model.PerformanceMetrics.R2Score)
		}
		return prediction, confidence, nil

	case "neural_network":
		prediction, err := e.predictNeuralNetwork(&artifact, features)
		if err != nil {
			return nil, 0, err
		}
		confidence := 0.88
		if model.PerformanceMetrics != nil && model.PerformanceMetrics.Accuracy > 0 {
			confidence = model.PerformanceMetrics.Accuracy
		}
		return prediction, confidence, nil

	default:
		return nil, 0, fmt.Errorf("unsupported model type: %s", artifact.ModelType)
	}
}

// predictDecisionTree traverses the decision tree stored in the artifact
func (e *InferenceEngine) predictDecisionTree(artifact *ModelArtifact, features []float64) (interface{}, error) {
	modelDataRaw, ok := artifact.Parameters["model_data"]
	if !ok {
		return 0.0, nil
	}

	modelJSON, err := json.Marshal(modelDataRaw)
	if err != nil {
		return 0.0, fmt.Errorf("failed to marshal tree data: %w", err)
	}

	var node mltraining.DecisionTreeModel
	if err := json.Unmarshal(modelJSON, &node); err != nil {
		return 0.0, fmt.Errorf("failed to unmarshal decision tree: %w", err)
	}

	return mltraining.TraverseTree(&node, features), nil
}

// predictRandomForest runs ensemble prediction with majority vote
func (e *InferenceEngine) predictRandomForest(artifact *ModelArtifact, features []float64) (interface{}, error) {
	modelDataRaw, ok := artifact.Parameters["model_data"]
	if !ok {
		return 0.0, nil
	}

	modelJSON, err := json.Marshal(modelDataRaw)
	if err != nil {
		return 0.0, fmt.Errorf("failed to marshal RF data: %w", err)
	}

	var rf mltraining.RandomForestArtifact
	if err := json.Unmarshal(modelJSON, &rf); err != nil {
		return 0.0, fmt.Errorf("failed to unmarshal random forest: %w", err)
	}

	// Majority vote across trees
	votes := make(map[float64]int)
	for _, tree := range rf.Trees {
		pred := math.Round(mltraining.TraverseTree(tree, features))
		votes[pred]++
	}

	bestCount := 0
	bestClass := 0.0
	for class, count := range votes {
		if count > bestCount {
			bestCount = count
			bestClass = class
		}
	}
	return bestClass, nil
}

// predictRegression runs linear regression: y = w·x + b
func (e *InferenceEngine) predictRegression(artifact *ModelArtifact, features []float64) interface{} {
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

// predictNeuralNetwork runs a forward pass through the stored network weights
func (e *InferenceEngine) predictNeuralNetwork(artifact *ModelArtifact, features []float64) (interface{}, error) {
	weightsRaw, ok := artifact.Parameters["weights"]
	if !ok {
		return 0.0, nil
	}
	biasesRaw, ok := artifact.Parameters["biases"]
	if !ok {
		return 0.0, nil
	}

	weightsJSON, err := json.Marshal(weightsRaw)
	if err != nil {
		return 0.0, fmt.Errorf("failed to marshal NN weights: %w", err)
	}
	biasesJSON, err := json.Marshal(biasesRaw)
	if err != nil {
		return 0.0, fmt.Errorf("failed to marshal NN biases: %w", err)
	}

	var weights [][][]float64
	var biases [][]float64
	if err := json.Unmarshal(weightsJSON, &weights); err != nil {
		return 0.0, fmt.Errorf("failed to unmarshal NN weights: %w", err)
	}
	if err := json.Unmarshal(biasesJSON, &biases); err != nil {
		return 0.0, fmt.Errorf("failed to unmarshal NN biases: %w", err)
	}

	// Forward pass
	a := make([]float64, len(features))
	copy(a, features)

	for l, w := range weights {
		outSize := len(w)
		z := make([]float64, outSize)
		for j := 0; j < outSize; j++ {
			z[j] = biases[l][j]
			for k, ak := range a {
				if k < len(w[j]) {
					z[j] += w[j][k] * ak
				}
			}
		}
		a = make([]float64, outSize)
		isOutput := l == len(weights)-1
		for j := range z {
			if isOutput {
				a[j] = inferSigmoid(z[j])
			} else {
				a[j] = inferRelu(z[j])
			}
		}
	}

	if len(a) > 0 {
		return a[0], nil
	}
	return 0.0, nil
}

func inferSigmoid(x float64) float64 { return 1.0 / (1.0 + math.Exp(-x)) }
func inferRelu(x float64) float64 {
	if x > 0 {
		return x
	}
	return 0
}

// getCachedPrediction retrieves a cached prediction if still valid
func (e *InferenceEngine) getCachedPrediction(twinID, modelID, entityID string) (*models.Prediction, error) {
	predictions, err := e.store.ListPredictionsByDigitalTwin(twinID)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	for _, pred := range predictions {
		if pred.ModelID == modelID && pred.EntityID == entityID {
			if pred.ExpiresAt.After(now) {
				return pred, nil
			}
		}
	}
	return nil, nil
}
