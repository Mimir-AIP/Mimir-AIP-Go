package models

import (
	"fmt"
	"time"
)

// ModelType represents the type of ML model
type ModelType string

const (
	ModelTypeDecisionTree  ModelType = "decision_tree"
	ModelTypeRandomForest  ModelType = "random_forest"
	ModelTypeRegression    ModelType = "regression"
	ModelTypeNeuralNetwork ModelType = "neural_network"
)

// ModelStatus represents the current status of an ML model
type ModelStatus string

const (
	ModelStatusDraft      ModelStatus = "draft"      // Model created but not trained
	ModelStatusTraining   ModelStatus = "training"   // Model is currently training
	ModelStatusTrained    ModelStatus = "trained"    // Model training completed successfully
	ModelStatusFailed     ModelStatus = "failed"     // Model training failed
	ModelStatusDeprecated ModelStatus = "deprecated" // Model performance degraded
	ModelStatusArchived   ModelStatus = "archived"   // Model archived
)

// MLModel represents a machine learning model
type MLModel struct {
	ID                  string                 `json:"id"`
	ProjectID           string                 `json:"project_id"`
	OntologyID          string                 `json:"ontology_id"` // Associated ontology
	Name                string                 `json:"name"`
	Description         string                 `json:"description"`
	Type                ModelType              `json:"type"`
	Status              ModelStatus            `json:"status"`
	Version             string                 `json:"version"`
	IsRecommended       bool                   `json:"is_recommended"`       // Was this the recommended type
	RecommendationScore int                    `json:"recommendation_score"` // Score from recommendation engine
	TrainingConfig      *TrainingConfig        `json:"training_config,omitempty"`
	TrainingMetrics     *TrainingMetrics       `json:"training_metrics,omitempty"`
	ModelArtifactPath   string                 `json:"model_artifact_path,omitempty"` // Path to trained model file
	PerformanceMetrics  *PerformanceMetrics    `json:"performance_metrics,omitempty"`
	Metadata            map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt           time.Time              `json:"created_at"`
	UpdatedAt           time.Time              `json:"updated_at"`
	TrainedAt           *time.Time             `json:"trained_at,omitempty"`
}

// TrainingConfig holds configuration for model training
type TrainingConfig struct {
	TrainTestSplit      float64                `json:"train_test_split"` // e.g., 0.8 for 80% training, 20% testing
	RandomSeed          int                    `json:"random_seed"`
	MaxIterations       int                    `json:"max_iterations,omitempty"`
	LearningRate        float64                `json:"learning_rate,omitempty"`
	BatchSize           int                    `json:"batch_size,omitempty"`
	EarlyStoppingRounds int                    `json:"early_stopping_rounds,omitempty"`
	Hyperparameters     map[string]interface{} `json:"hyperparameters,omitempty"`
}

// TrainingMetrics holds metrics collected during training
type TrainingMetrics struct {
	Epoch              int                    `json:"epoch"`
	TrainingLoss       float64                `json:"training_loss"`
	ValidationLoss     float64                `json:"validation_loss"`
	TrainingAccuracy   float64                `json:"training_accuracy,omitempty"`
	ValidationAccuracy float64                `json:"validation_accuracy,omitempty"`
	LearningCurve      []LearningCurvePoint   `json:"learning_curve,omitempty"`
	AdditionalMetrics  map[string]interface{} `json:"additional_metrics,omitempty"`
}

// LearningCurvePoint represents a point in the learning curve
type LearningCurvePoint struct {
	Epoch              int     `json:"epoch"`
	TrainingLoss       float64 `json:"training_loss"`
	ValidationLoss     float64 `json:"validation_loss"`
	TrainingAccuracy   float64 `json:"training_accuracy,omitempty"`
	ValidationAccuracy float64 `json:"validation_accuracy,omitempty"`
}

// PerformanceMetrics holds model performance metrics
type PerformanceMetrics struct {
	Accuracy          float64                `json:"accuracy,omitempty"`
	Precision         float64                `json:"precision,omitempty"`
	Recall            float64                `json:"recall,omitempty"`
	F1Score           float64                `json:"f1_score,omitempty"`
	RMSE              float64                `json:"rmse,omitempty"` // For regression
	MAE               float64                `json:"mae,omitempty"`  // For regression
	R2Score           float64                `json:"r2_score,omitempty"`
	ConfusionMatrix   [][]int                `json:"confusion_matrix,omitempty"`
	FeatureImportance map[string]float64     `json:"feature_importance,omitempty"`
	AdditionalMetrics map[string]interface{} `json:"additional_metrics,omitempty"`
}

// ModelRecommendation represents a model type recommendation
type ModelRecommendation struct {
	RecommendedType  ModelType         `json:"recommended_type"`
	Score            int               `json:"score"`
	Reasoning        string            `json:"reasoning"`
	AllScores        map[ModelType]int `json:"all_scores"`
	OntologyAnalysis *OntologyAnalysis `json:"ontology_analysis"`
	DataAnalysis     *DataAnalysis     `json:"data_analysis"`
}

// OntologyAnalysis holds analysis of the ontology for recommendation
type OntologyAnalysis struct {
	NumEntities      int            `json:"num_entities"`
	NumAttributes    int            `json:"num_attributes"`
	NumRelationships int            `json:"num_relationships"`
	NumericalRatio   float64        `json:"numerical_ratio"`
	CategoricalRatio float64        `json:"categorical_ratio"`
	DataTypes        map[string]int `json:"data_types"`
	Complexity       string         `json:"complexity"` // "low", "medium", "high"
}

// DataAnalysis holds analysis of ingested data for recommendation
type DataAnalysis struct {
	Size            string `json:"size"` // "small", "medium", "large"
	RecordCount     int64  `json:"record_count"`
	HasUnstructured bool   `json:"has_unstructured"`
	FeatureCount    int    `json:"feature_count"`
}

// ModelCreateRequest represents a request to create a new ML model
type ModelCreateRequest struct {
	ProjectID      string                 `json:"project_id"`
	OntologyID     string                 `json:"ontology_id"`
	Name           string                 `json:"name"`
	Description    string                 `json:"description"`
	Type           ModelType              `json:"type"`
	TrainingConfig *TrainingConfig        `json:"training_config,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// Validate checks if the ModelCreateRequest is valid
func (r *ModelCreateRequest) Validate() error {
	if r.ProjectID == "" {
		return fmt.Errorf("project_id is required")
	}
	if r.OntologyID == "" {
		return fmt.Errorf("ontology_id is required")
	}
	if r.Name == "" {
		return fmt.Errorf("name is required")
	}
	if r.Type == "" {
		return fmt.Errorf("type is required")
	}
	// Validate model type
	validTypes := []ModelType{
		ModelTypeDecisionTree,
		ModelTypeRandomForest,
		ModelTypeRegression,
		ModelTypeNeuralNetwork,
	}
	valid := false
	for _, t := range validTypes {
		if r.Type == t {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("invalid model type: %s", r.Type)
	}
	return nil
}

// ModelUpdateRequest represents a request to update an ML model
type ModelUpdateRequest struct {
	Name               *string                `json:"name,omitempty"`
	Description        *string                `json:"description,omitempty"`
	Status             *ModelStatus           `json:"status,omitempty"`
	TrainingMetrics    *TrainingMetrics       `json:"training_metrics,omitempty"`
	PerformanceMetrics *PerformanceMetrics    `json:"performance_metrics,omitempty"`
	Metadata           map[string]interface{} `json:"metadata,omitempty"`
}

// ModelRecommendationRequest represents a request for model recommendation
type ModelRecommendationRequest struct {
	ProjectID  string `json:"project_id"`
	OntologyID string `json:"ontology_id"`
}

// Validate checks if the ModelRecommendationRequest is valid
func (r *ModelRecommendationRequest) Validate() error {
	if r.ProjectID == "" {
		return fmt.Errorf("project_id is required")
	}
	if r.OntologyID == "" {
		return fmt.Errorf("ontology_id is required")
	}
	return nil
}

// ModelTrainingRequest represents a request to train a model
type ModelTrainingRequest struct {
	ModelID        string          `json:"model_id"`
	StorageIDs     []string        `json:"storage_ids"` // Storage configs to use for training data
	TrainingConfig *TrainingConfig `json:"training_config,omitempty"`
}

// Validate checks if the ModelTrainingRequest is valid
func (r *ModelTrainingRequest) Validate() error {
	if r.ModelID == "" {
		return fmt.Errorf("model_id is required")
	}
	// StorageIDs is optional - if empty, worker will generate synthetic data for demo purposes
	return nil
}
