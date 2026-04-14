package mlmodel

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"time"

	"github.com/mimir-aip/mimir-aip-go/pkg/mlmodel/training"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

type BuiltinProvider struct{}

func NewBuiltinProvider() *BuiltinProvider { return &BuiltinProvider{} }

func (p *BuiltinProvider) Metadata() models.MLProviderMetadata {
	return models.MLProviderMetadata{
		Name:               "builtin",
		DisplayName:        "Builtin Models",
		Description:        "First-party tabular ML implementations bundled with Mimir.",
		SupportsTraining:   true,
		SupportsInference:  true,
		SupportsMonitoring: true,
		Models: []models.MLProviderModel{
			{Name: string(models.ModelTypeDecisionTree), DisplayName: "Decision Tree", Capabilities: []models.MLProviderCapability{models.MLProviderCapabilityTrain, models.MLProviderCapabilityInfer, models.MLProviderCapabilityClassify}},
			{Name: string(models.ModelTypeRandomForest), DisplayName: "Random Forest", Capabilities: []models.MLProviderCapability{models.MLProviderCapabilityTrain, models.MLProviderCapabilityInfer, models.MLProviderCapabilityClassify}},
			{Name: string(models.ModelTypeRegression), DisplayName: "Regression", Capabilities: []models.MLProviderCapability{models.MLProviderCapabilityTrain, models.MLProviderCapabilityInfer, models.MLProviderCapabilityRegress}},
			{Name: string(models.ModelTypeNeuralNetwork), DisplayName: "Neural Network", Capabilities: []models.MLProviderCapability{models.MLProviderCapabilityTrain, models.MLProviderCapabilityInfer, models.MLProviderCapabilityClassify, models.MLProviderCapabilityRegress}},
		},
	}
}

func (p *BuiltinProvider) ValidateModel(model *models.MLModel) error {
	_, providerModel, err := normalizeProviderIdentity(model)
	if err != nil {
		return err
	}
	switch providerModel {
	case string(models.ModelTypeDecisionTree), string(models.ModelTypeRandomForest), string(models.ModelTypeRegression), string(models.ModelTypeNeuralNetwork):
		return nil
	default:
		return fmt.Errorf("unsupported builtin provider model: %s", providerModel)
	}
}

func (p *BuiltinProvider) Train(req *ProviderTrainRequest) (*ProviderTrainResult, error) {
	if req == nil || req.Model == nil {
		return nil, fmt.Errorf("model is required")
	}
	if err := p.ValidateModel(req.Model); err != nil {
		return nil, err
	}
	trainingData, ok := req.TrainingData.(*training.TrainingData)
	if !ok || trainingData == nil {
		return nil, fmt.Errorf("training data is required")
	}
	factory := training.NewTrainerFactory()
	trainer, err := factory.GetTrainer(req.Model.Type)
	if err != nil {
		return nil, fmt.Errorf("failed to get trainer: %w", err)
	}
	result, err := trainer.Train(trainingData, req.Model.TrainingConfig)
	if err != nil {
		return nil, err
	}
	providerArtifact := map[string]any{
		"provider":       "builtin",
		"provider_model": string(req.Model.Type),
		"feature_names":  trainingData.FeatureNames,
		"parameters": map[string]any{
			"model_data":         result.ModelData,
			"feature_importance": result.FeatureImportance,
		},
		"metadata": map[string]any{
			"trained_at": time.Now().UTC().Format(time.RFC3339),
		},
	}
	artifactData, err := json.Marshal(providerArtifact)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal builtin provider artifact: %w", err)
	}
	return &ProviderTrainResult{
		ArtifactData:       artifactData,
		PerformanceMetrics: result.PerformanceMetrics,
		TrainingMetrics:    result.TrainingMetrics,
		AdditionalMetadata: map[string]any{"feature_importance": result.FeatureImportance},
	}, nil
}

func (p *BuiltinProvider) Infer(req *ProviderInferRequest) (*ProviderInferResult, error) {
	if req == nil || req.Model == nil {
		return nil, fmt.Errorf("model is required")
	}
	if err := p.ValidateModel(req.Model); err != nil {
		return nil, err
	}
	artifact, err := readBuiltinArtifact(req.Model.ModelArtifactPath)
	if err != nil {
		return nil, err
	}
	features := buildFeatureVector(artifact.FeatureNames, req.Input)
	output, confidence, err := inferBuiltinArtifact(req.Model, artifact, features)
	if err != nil {
		return nil, err
	}
	return &ProviderInferResult{Output: output, Confidence: confidence, Metadata: map[string]any{"provider": "builtin", "provider_model": artifact.ProviderModel}}, nil
}

type BuiltinArtifact struct {
	Provider      string         `json:"provider,omitempty"`
	ProviderModel string         `json:"provider_model,omitempty"`
	ModelType     string         `json:"model_type,omitempty"`
	FeatureNames  []string       `json:"feature_names"`
	Parameters    map[string]any `json:"parameters"`
	Metadata      map[string]any `json:"metadata,omitempty"`
}

func buildFeatureVector(featureNames []string, input map[string]any) []float64 {
	features := make([]float64, len(featureNames))
	for i, featureName := range featureNames {
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
			}
		}
	}
	return features
}

func ReadBuiltinArtifactForWorker(path string) (*BuiltinArtifact, error) {
	return readBuiltinArtifact(path)
}

func readBuiltinArtifact(path string) (*BuiltinArtifact, error) {
	if path == "" {
		return nil, fmt.Errorf("model artifact path is empty")
	}
	artifactData, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read model artifact: %w", err)
	}
	var artifact BuiltinArtifact
	if err := json.Unmarshal(artifactData, &artifact); err != nil {
		return nil, fmt.Errorf("failed to unmarshal artifact: %w", err)
	}
	if artifact.ProviderModel == "" {
		artifact.ProviderModel = artifact.ModelType
	}
	return &artifact, nil
}

func inferBuiltinArtifact(model *models.MLModel, artifact *BuiltinArtifact, features []float64) (any, float64, error) {
	switch artifact.ProviderModel {
	case string(models.ModelTypeDecisionTree):
		prediction, err := predictDecisionTreeArtifact(artifact, features)
		if err != nil {
			return nil, 0, err
		}
		confidence := 0.85
		if model.PerformanceMetrics != nil && model.PerformanceMetrics.Accuracy > 0 {
			confidence = model.PerformanceMetrics.Accuracy
		}
		return prediction, confidence, nil
	case string(models.ModelTypeRandomForest):
		prediction, err := predictRandomForestArtifact(artifact, features)
		if err != nil {
			return nil, 0, err
		}
		confidence := 0.90
		if model.PerformanceMetrics != nil && model.PerformanceMetrics.Accuracy > 0 {
			confidence = model.PerformanceMetrics.Accuracy
		}
		return prediction, confidence, nil
	case string(models.ModelTypeRegression):
		prediction, err := predictRegressionArtifact(artifact, features)
		if err != nil {
			return nil, 0, err
		}
		confidence := 1.0
		if model.PerformanceMetrics != nil && model.PerformanceMetrics.R2Score > 0 {
			confidence = math.Max(0, model.PerformanceMetrics.R2Score)
		}
		return prediction, confidence, nil
	case string(models.ModelTypeNeuralNetwork):
		prediction, err := predictNeuralNetworkArtifact(artifact, features)
		if err != nil {
			return nil, 0, err
		}
		confidence := 0.88
		if model.PerformanceMetrics != nil && model.PerformanceMetrics.Accuracy > 0 {
			confidence = model.PerformanceMetrics.Accuracy
		}
		return prediction, confidence, nil
	default:
		return nil, 0, fmt.Errorf("unsupported builtin model type: %s", artifact.ProviderModel)
	}
}

func predictDecisionTreeArtifact(artifact *BuiltinArtifact, features []float64) (any, error) {
	modelDataRaw, ok := artifact.Parameters["model_data"]
	if !ok {
		return 0.0, nil
	}
	modelJSON, err := json.Marshal(modelDataRaw)
	if err != nil {
		return 0.0, fmt.Errorf("failed to marshal tree data: %w", err)
	}
	var node training.DecisionTreeModel
	if err := json.Unmarshal(modelJSON, &node); err != nil {
		return 0.0, fmt.Errorf("failed to unmarshal decision tree: %w", err)
	}
	return training.TraverseTree(&node, features), nil
}

func predictRandomForestArtifact(artifact *BuiltinArtifact, features []float64) (any, error) {
	modelDataRaw, ok := artifact.Parameters["model_data"]
	if !ok {
		return 0.0, nil
	}
	modelJSON, err := json.Marshal(modelDataRaw)
	if err != nil {
		return 0.0, fmt.Errorf("failed to marshal RF data: %w", err)
	}
	var rf training.RandomForestArtifact
	if err := json.Unmarshal(modelJSON, &rf); err != nil {
		return 0.0, fmt.Errorf("failed to unmarshal random forest: %w", err)
	}
	votes := make(map[float64]int)
	for _, tree := range rf.Trees {
		pred := math.Round(training.TraverseTree(tree, features))
		votes[pred]++
	}
	bestCount, bestClass := 0, 0.0
	for class, count := range votes {
		if count > bestCount {
			bestCount = count
			bestClass = class
		}
	}
	return bestClass, nil
}

func predictRegressionArtifact(artifact *BuiltinArtifact, features []float64) (any, error) {
	modelDataRaw, ok := artifact.Parameters["model_data"]
	if !ok {
		return 0.0, nil
	}
	mdMap, ok := modelDataRaw.(map[string]any)
	if !ok {
		return 0.0, fmt.Errorf("invalid model_data format for regression")
	}
	intercept := 0.0
	if b, ok := mdMap["intercept"].(float64); ok {
		intercept = b
	}
	pred := intercept
	if coeffsRaw, ok := mdMap["coefficients"].([]any); ok {
		for i, c := range coeffsRaw {
			if i < len(features) {
				if cf, ok := c.(float64); ok {
					pred += cf * features[i]
				}
			}
		}
	}
	return pred, nil
}

func predictNeuralNetworkArtifact(artifact *BuiltinArtifact, features []float64) (any, error) {
	mdMap, ok := artifact.Parameters["model_data"].(map[string]any)
	if !ok {
		return 0.0, fmt.Errorf("invalid model_data format for neural_network")
	}
	weightsJSON, err := json.Marshal(mdMap["weights"])
	if err != nil {
		return 0.0, fmt.Errorf("failed to marshal NN weights: %w", err)
	}
	biasesJSON, err := json.Marshal(mdMap["biases"])
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
	a := make([]float64, len(features))
	copy(a, features)
	for l, w := range weights {
		outSize := len(w)
		z := make([]float64, outSize)
		for j := range z {
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
				a[j] = 1.0 / (1.0 + math.Exp(-z[j]))
			} else if z[j] > 0 {
				a[j] = z[j]
			}
		}
	}
	if len(a) > 0 {
		return a[0], nil
	}
	return 0.0, nil
}
