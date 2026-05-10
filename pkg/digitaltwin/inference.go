package digitaltwin

import (
	"fmt"
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

func (e *InferenceEngine) loadOwnedTrainedModel(twin *models.DigitalTwin, modelID string) (*models.MLModel, error) {
	if twin == nil {
		return nil, fmt.Errorf("digital twin is required")
	}
	model, err := e.mlService.GetModelForProject(twin.ProjectID, modelID)
	if err != nil {
		return nil, fmt.Errorf("failed to get model: %w", err)
	}
	if model.OntologyID != "" && twin.OntologyID != "" && model.OntologyID != twin.OntologyID {
		return nil, fmt.Errorf("ml model %s is bound to ontology %s, not digital twin ontology %s", model.ID, model.OntologyID, twin.OntologyID)
	}
	if model.Status != models.ModelStatusTrained {
		return nil, fmt.Errorf("model is not trained (status: %s)", model.Status)
	}
	return model, nil
}

// Predict runs a single prediction
func (e *InferenceEngine) Predict(twin *models.DigitalTwin, req *models.PredictionRequest) (*models.Prediction, error) {
	if req.UseCache {
		cached, err := e.getCachedPrediction(twin.ID, req.ModelID, req.EntityID)
		if err == nil && cached != nil {
			return cached, nil
		}
	}

	model, err := e.loadOwnedTrainedModel(twin, req.ModelID)
	if err != nil {
		return nil, err
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
	model, err := e.loadOwnedTrainedModel(twin, req.ModelID)
	if err != nil {
		return nil, err
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

// runInference executes the actual inference using the ML service's provider abstraction.
func (e *InferenceEngine) runInference(model *models.MLModel, input map[string]interface{}) (interface{}, float64, error) {
	return e.mlService.InferModel(model.ID, input)
}

// predictRegression runs linear regression: y = w·x + b
func (e *InferenceEngine) predictRegression(artifact *ModelArtifact, features []float64) interface{} {
	modelDataRaw, ok := artifact.Parameters["model_data"]
	if !ok {
		return 0.0
	}
	modelData, ok := modelDataRaw.(map[string]interface{})
	if !ok {
		return 0.0
	}
	weightsRaw, ok := modelData["coefficients"]
	if !ok {
		return 0.0
	}
	weightsSlice, ok := weightsRaw.([]interface{})
	if !ok {
		return 0.0
	}

	intercept := 0.0
	if b, ok := modelData["intercept"]; ok {
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
