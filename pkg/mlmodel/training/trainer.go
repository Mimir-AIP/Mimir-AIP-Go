package training

import (
	"fmt"

	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

// Trainer interface defines the contract for ML model training
type Trainer interface {
	// Train trains the model with the provided data
	Train(data *TrainingData, config *models.TrainingConfig) (*TrainingResult, error)

	// Validate tests the trained model on validation data
	Validate(data *TrainingData) (*ValidationResult, error)

	// GetType returns the model type this trainer handles
	GetType() models.ModelType
}

// TrainingData holds the data for training and validation
type TrainingData struct {
	TrainFeatures [][]float64            // Training features (rows x features)
	TrainLabels   []float64              // Training labels/targets
	TestFeatures  [][]float64            // Test features
	TestLabels    []float64              // Test labels/targets
	FeatureNames  []string               // Names of features
	Metadata      map[string]interface{} // Additional metadata
}

// TrainingResult holds the results of model training
type TrainingResult struct {
	ModelData          interface{}                // Trained model (serializable)
	TrainingMetrics    *models.TrainingMetrics    // Metrics during training
	PerformanceMetrics *models.PerformanceMetrics // Final performance on test set
	FeatureImportance  map[string]float64         // Feature importance scores
}

// ValidationResult holds validation metrics
type ValidationResult struct {
	Loss     float64
	Accuracy float64
	Metrics  map[string]float64
}

// TrainerFactory creates trainers for different model types
type TrainerFactory struct {
	trainers map[models.ModelType]Trainer
}

// NewTrainerFactory creates a new trainer factory
func NewTrainerFactory() *TrainerFactory {
	factory := &TrainerFactory{
		trainers: make(map[models.ModelType]Trainer),
	}

	// Register trainers for each model type
	factory.trainers[models.ModelTypeDecisionTree] = NewDecisionTreeTrainer()
	factory.trainers[models.ModelTypeRandomForest] = NewRandomForestTrainer()
	factory.trainers[models.ModelTypeRegression] = NewRegressionTrainer()
	factory.trainers[models.ModelTypeNeuralNetwork] = NewNeuralNetworkTrainer()

	return factory
}

// GetTrainer returns the appropriate trainer for a model type
func (f *TrainerFactory) GetTrainer(modelType models.ModelType) (Trainer, error) {
	trainer, ok := f.trainers[modelType]
	if !ok {
		return nil, fmt.Errorf("no trainer available for model type: %s", modelType)
	}
	return trainer, nil
}
