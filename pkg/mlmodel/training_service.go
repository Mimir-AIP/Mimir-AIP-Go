package mlmodel

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"os"
	"path/filepath"
	"time"
)

func (s *Service) StartTraining(req *models.ModelTrainingRequest) (*models.MLModel, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	model, err := s.store.GetMLModel(req.ModelID)
	if err != nil {
		return nil, fmt.Errorf("failed to get model: %w", err)
	}
	if model.Status == models.ModelStatusTraining {
		return nil, fmt.Errorf("model is already training")
	}
	providerName, providerModel, err := normalizeProviderIdentity(model)
	if err != nil {
		return nil, err
	}
	if req.TrainingConfig != nil {
		model.TrainingConfig = req.TrainingConfig
	}
	provenance, err := s.buildFeatureProvenance(model, req.StorageIDs)
	if err != nil {
		return nil, err
	}
	if model.Metadata == nil {
		model.Metadata = map[string]interface{}{}
	}
	model.Metadata[modelMetadataFeatureProvenance] = provenance
	workTask := &models.WorkTask{
		ID:          uuid.New().String(),
		Type:        models.WorkTaskTypeMLTraining,
		ProjectID:   model.ProjectID,
		Priority:    5,
		Status:      models.WorkTaskStatusQueued,
		SubmittedAt: time.Now().UTC(),
		TaskSpec: models.TaskSpec{
			ModelID:   model.ID,
			ProjectID: model.ProjectID,
			Parameters: map[string]any{
				"model_id":        model.ID,
				"ontology_id":     model.OntologyID,
				"storage_ids":     req.StorageIDs,
				"config":          model.TrainingConfig,
				"provider":        providerName,
				"provider_model":  providerModel,
				"provider_config": model.ProviderConfig,
			},
		},
		ResourceRequirements: models.ResourceRequirements{
			CPU:    "2000m",
			Memory: "4Gi",
			GPU:    false,
		},
		DataAccess: models.DataAccess{
			InputDatasets: req.StorageIDs,
		},
	}

	// Update model status and persist the canonical async handle before workers pick up the task.
	model.Status = models.ModelStatusTraining
	model.TrainingTaskID = workTask.ID
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

	// Enqueue the training task.
	if err := s.queue.Enqueue(workTask); err != nil {
		// Roll back the persisted async handle on queue failure so callers do not observe a phantom task.
		model.Status = models.ModelStatusDraft
		model.TrainingTaskID = ""
		model.UpdatedAt = time.Now().UTC()
		s.store.SaveMLModel(model)
		return nil, fmt.Errorf("failed to enqueue training task: %w", err)
	}

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

func modelArtifactBaseDir() string {
	if dir := os.Getenv("MODEL_ARTIFACT_DIR"); dir != "" {
		return dir
	}
	return filepath.Join(os.TempDir(), "mimir-aip", "model-artifacts")
}

func persistModelArtifact(modelID string, artifactData []byte) (string, error) {
	if len(artifactData) == 0 {
		return "", fmt.Errorf("model artifact data is required")
	}
	artifactDir := filepath.Join(modelArtifactBaseDir(), modelID)
	if err := os.MkdirAll(artifactDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create artifact directory: %w", err)
	}
	artifactPath := filepath.Join(artifactDir, "model.json")
	if err := os.WriteFile(artifactPath, artifactData, 0o644); err != nil {
		return "", fmt.Errorf("failed to write model artifact: %w", err)
	}
	return artifactPath, nil
}

// CompleteTraining marks training as complete, persists the artifact into orchestrator-visible storage, and stores performance metrics.
func (s *Service) CompleteTraining(modelID string, artifactData []byte, performanceMetrics *models.PerformanceMetrics) error {
	model, err := s.store.GetMLModel(modelID)
	if err != nil {
		return fmt.Errorf("failed to get model: %w", err)
	}

	artifactPath, err := persistModelArtifact(modelID, artifactData)
	if err != nil {
		return err
	}

	model.Status = models.ModelStatusTrained
	model.TrainingTaskID = ""
	model.ModelArtifactPath = artifactPath
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
	model.TrainingTaskID = ""
	model.UpdatedAt = time.Now().UTC()

	if model.Metadata == nil {
		model.Metadata = make(map[string]any)
	}
	model.Metadata["failure_reason"] = reason

	if err := s.store.SaveMLModel(model); err != nil {
		return fmt.Errorf("failed to update model: %w", err)
	}

	return nil
}
