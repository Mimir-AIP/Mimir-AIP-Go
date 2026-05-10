package mlmodel

import (
	"fmt"
	"github.com/mimir-aip/mimir-aip-go/pkg/mlmodel/training"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"log"
	"math"
)

func (s *Service) inferModel(model *models.MLModel, input map[string]any) (any, float64, error) {
	provider, err := s.resolveProviderForModel(model)
	if err != nil {
		return nil, 0, err
	}
	if err := s.ensureModelOntologyCompatible(model); err != nil {
		return nil, 0, err
	}
	if err := provider.ValidateModel(model); err != nil {
		return nil, 0, err
	}
	result, err := provider.Infer(&ProviderInferRequest{Model: model, Input: input})
	if err != nil {
		return nil, 0, err
	}
	return result.Output, result.Confidence, nil
}

func (s *Service) InferModel(modelID string, input map[string]any) (any, float64, error) {
	model, err := s.store.GetMLModel(modelID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get model: %w", err)
	}
	return s.inferModel(model, input)
}

// ValidateModel runs the trained model artifact against the provided data and returns performance metrics.
// It resolves the model provider and runs normalized inference over each test row.
func (s *Service) ValidateModel(modelID string, data *training.TrainingData) (*models.PerformanceMetrics, error) {
	model, err := s.store.GetMLModel(modelID)
	if err != nil {
		return nil, fmt.Errorf("failed to get model: %w", err)
	}
	predictions := make([]float64, len(data.TestFeatures))
	for i, features := range data.TestFeatures {
		input := make(map[string]any, len(data.FeatureNames))
		for idx, featureName := range data.FeatureNames {
			if idx < len(features) {
				input[featureName] = features[idx]
			}
		}
		pred, _, err := s.inferModel(model, input)
		if err != nil {
			log.Printf("Warning: inference failed for row %d during validation: %v", i, err)
			continue
		}
		switch v := pred.(type) {
		case float64:
			predictions[i] = v
		case int:
			predictions[i] = float64(v)
		default:
			return nil, fmt.Errorf("unsupported prediction output type %T during validation", pred)
		}
	}
	return computePerformanceMetrics(model.Type, predictions, data.TestLabels), nil
}

// computePerformanceMetrics calculates performance metrics for predictions vs actual labels
func computePerformanceMetrics(modelType models.ModelType, predictions, actual []float64) *models.PerformanceMetrics {
	if len(predictions) == 0 || len(predictions) != len(actual) {
		return &models.PerformanceMetrics{}
	}

	switch modelType {
	case models.ModelTypeRegression:
		sumSq, sumAbs := 0.0, 0.0
		meanActual := 0.0
		for _, v := range actual {
			meanActual += v
		}
		meanActual /= float64(len(actual))
		ssTot, ssRes := 0.0, 0.0
		for i := range predictions {
			diff := predictions[i] - actual[i]
			sumSq += diff * diff
			sumAbs += math.Abs(diff)
			ssRes += diff * diff
			ssTot += math.Pow(actual[i]-meanActual, 2)
		}
		rmse := math.Sqrt(sumSq / float64(len(predictions)))
		mae := sumAbs / float64(len(predictions))
		r2 := 0.0
		if ssTot > 0 {
			r2 = 1.0 - (ssRes / ssTot)
		}
		return &models.PerformanceMetrics{RMSE: rmse, MAE: mae, R2Score: r2}
	default:
		correct := 0
		for i := range predictions {
			if math.Round(predictions[i]) == math.Round(actual[i]) {
				correct++
			}
		}
		accuracy := float64(correct) / float64(len(predictions))
		f1 := 0.0
		if accuracy > 0 {
			f1 = 2 * (accuracy * accuracy) / (accuracy + accuracy)
		}
		return &models.PerformanceMetrics{
			Accuracy:  accuracy,
			Precision: accuracy,
			Recall:    accuracy,
			F1Score:   f1,
		}
	}
}
