package mlmodel

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/mimir-aip/mimir-aip-go/pkg/metadatastore"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"github.com/mimir-aip/mimir-aip-go/pkg/ontology"
	"github.com/mimir-aip/mimir-aip-go/pkg/storage"
)

// Service manages ML models and training
type Service struct {
	store                metadatastore.MetadataStore
	ontologyService      *ontology.Service
	storageService       *storage.Service
	recommendationEngine *RecommendationEngine
}

// NewService creates a new ML model service
func NewService(
	store metadatastore.MetadataStore,
	ontologyService *ontology.Service,
	storageService *storage.Service,
) *Service {
	return &Service{
		store:                store,
		ontologyService:      ontologyService,
		storageService:       storageService,
		recommendationEngine: NewRecommendationEngine(),
	}
}

// CreateModel creates a new ML model
func (s *Service) CreateModel(req *models.ModelCreateRequest) (*models.MLModel, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// Verify ontology exists
	ontology, err := s.ontologyService.GetOntology(req.OntologyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get ontology: %w", err)
	}

	if ontology.ProjectID != req.ProjectID {
		return nil, fmt.Errorf("ontology does not belong to project")
	}

	model := &models.MLModel{
		ID:             uuid.New().String(),
		ProjectID:      req.ProjectID,
		OntologyID:     req.OntologyID,
		Name:           req.Name,
		Description:    req.Description,
		Type:           req.Type,
		Status:         models.ModelStatusDraft,
		Version:        "1.0",
		TrainingConfig: req.TrainingConfig,
		Metadata:       req.Metadata,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}

	// Set default training config if not provided
	if model.TrainingConfig == nil {
		model.TrainingConfig = &models.TrainingConfig{
			TrainTestSplit: 0.8,
			RandomSeed:     42,
		}
	}

	if err := s.store.SaveMLModel(model); err != nil {
		return nil, fmt.Errorf("failed to save model: %w", err)
	}

	return model, nil
}

// GetModel retrieves an ML model by ID
func (s *Service) GetModel(id string) (*models.MLModel, error) {
	model, err := s.store.GetMLModel(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get model: %w", err)
	}
	return model, nil
}

// UpdateModel updates an existing ML model
func (s *Service) UpdateModel(id string, req *models.ModelUpdateRequest) (*models.MLModel, error) {
	model, err := s.store.GetMLModel(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get model: %w", err)
	}

	// Update fields
	if req.Name != nil {
		model.Name = *req.Name
	}
	if req.Description != nil {
		model.Description = *req.Description
	}
	if req.Status != nil {
		model.Status = *req.Status
	}
	if req.TrainingMetrics != nil {
		model.TrainingMetrics = req.TrainingMetrics
	}
	if req.PerformanceMetrics != nil {
		model.PerformanceMetrics = req.PerformanceMetrics
	}
	if req.Metadata != nil {
		if model.Metadata == nil {
			model.Metadata = make(map[string]interface{})
		}
		for k, v := range req.Metadata {
			model.Metadata[k] = v
		}
	}

	model.UpdatedAt = time.Now().UTC()

	if err := s.store.SaveMLModel(model); err != nil {
		return nil, fmt.Errorf("failed to update model: %w", err)
	}

	return model, nil
}

// DeleteModel deletes an ML model
func (s *Service) DeleteModel(id string) error {
	if err := s.store.DeleteMLModel(id); err != nil {
		return fmt.Errorf("failed to delete model: %w", err)
	}
	return nil
}

// ListProjectModels lists all models for a project
func (s *Service) ListProjectModels(projectID string) ([]*models.MLModel, error) {
	models, err := s.store.ListMLModelsByProject(projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list models: %w", err)
	}
	return models, nil
}

// RecommendModelType recommends the best model type for a project
func (s *Service) RecommendModelType(projectID, ontologyID string) (*models.ModelRecommendation, error) {
	// Get the ontology
	ontology, err := s.ontologyService.GetOntology(ontologyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get ontology: %w", err)
	}

	if ontology.ProjectID != projectID {
		return nil, fmt.Errorf("ontology does not belong to project")
	}

	// Get storage configs for the project to analyze data
	storageConfigs, err := s.storageService.GetProjectStorageConfigs(projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get storage configs: %w", err)
	}

	// Analyze data from storage configs
	dataSummary := s.analyzeData(storageConfigs)

	// Get recommendation
	recommendation, err := s.recommendationEngine.RecommendModelType(ontology, dataSummary)
	if err != nil {
		return nil, fmt.Errorf("failed to recommend model type: %w", err)
	}

	return recommendation, nil
}

// analyzeData analyzes storage configs to create a data summary
func (s *Service) analyzeData(storageConfigs []*models.StorageConfig) *models.DataAnalysis {
	// For now, provide a simple analysis
	// In production, this would actually query the storage to count records, analyze features, etc.

	size := "small"
	recordCount := int64(0)

	if len(storageConfigs) > 0 {
		// Estimate based on number of storage configs
		// This is a placeholder - in real implementation, would query actual data
		if len(storageConfigs) > 5 {
			size = "large"
			recordCount = 10000
		} else if len(storageConfigs) > 2 {
			size = "medium"
			recordCount = 1000
		} else {
			size = "small"
			recordCount = 100
		}
	}

	return &models.DataAnalysis{
		Size:            size,
		RecordCount:     recordCount,
		HasUnstructured: false, // Placeholder
		FeatureCount:    0,     // Placeholder
	}
}

// StartTraining initiates model training
// In a real implementation, this would submit a training job to the worker queue
// For now, we'll update the model status to indicate training has started
func (s *Service) StartTraining(req *models.ModelTrainingRequest) (*models.MLModel, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// Get the model
	model, err := s.store.GetMLModel(req.ModelID)
	if err != nil {
		return nil, fmt.Errorf("failed to get model: %w", err)
	}

	// Verify model is in correct state to start training
	if model.Status == models.ModelStatusTraining {
		return nil, fmt.Errorf("model is already training")
	}

	// Update training config if provided
	if req.TrainingConfig != nil {
		model.TrainingConfig = req.TrainingConfig
	}

	// Update model status
	model.Status = models.ModelStatusTraining
	model.UpdatedAt = time.Now().UTC()

	// Initialize training metrics
	model.TrainingMetrics = &models.TrainingMetrics{
		Epoch:              0,
		TrainingLoss:       0,
		ValidationLoss:     0,
		TrainingAccuracy:   0,
		ValidationAccuracy: 0,
		LearningCurve:      []models.LearningCurvePoint{},
	}

	if err := s.store.SaveMLModel(model); err != nil {
		return nil, fmt.Errorf("failed to update model: %w", err)
	}

	// TODO: Submit training job to worker queue
	// This would create a WorkTask with type "model_training" and necessary parameters
	// The worker would then execute Python-based ML training code

	return model, nil
}

// UpdateTrainingProgress updates training progress metrics
// This would be called by workers during training to report progress
func (s *Service) UpdateTrainingProgress(modelID string, metrics *models.TrainingMetrics) error {
	model, err := s.store.GetMLModel(modelID)
	if err != nil {
		return fmt.Errorf("failed to get model: %w", err)
	}

	model.TrainingMetrics = metrics
	model.UpdatedAt = time.Now().UTC()

	if err := s.store.SaveMLModel(model); err != nil {
		return fmt.Errorf("failed to update model: %w", err)
	}

	return nil
}

// CompleteTraining marks training as complete and stores performance metrics
func (s *Service) CompleteTraining(modelID, modelArtifactPath string, performanceMetrics *models.PerformanceMetrics) error {
	model, err := s.store.GetMLModel(modelID)
	if err != nil {
		return fmt.Errorf("failed to get model: %w", err)
	}

	model.Status = models.ModelStatusTrained
	model.ModelArtifactPath = modelArtifactPath
	model.PerformanceMetrics = performanceMetrics
	now := time.Now().UTC()
	model.TrainedAt = &now
	model.UpdatedAt = now

	if err := s.store.SaveMLModel(model); err != nil {
		return fmt.Errorf("failed to update model: %w", err)
	}

	return nil
}

// FailTraining marks training as failed
func (s *Service) FailTraining(modelID, reason string) error {
	model, err := s.store.GetMLModel(modelID)
	if err != nil {
		return fmt.Errorf("failed to get model: %w", err)
	}

	model.Status = models.ModelStatusFailed
	model.UpdatedAt = time.Now().UTC()

	if model.Metadata == nil {
		model.Metadata = make(map[string]interface{})
	}
	model.Metadata["failure_reason"] = reason

	if err := s.store.SaveMLModel(model); err != nil {
		return fmt.Errorf("failed to update model: %w", err)
	}

	return nil
}
