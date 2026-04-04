package mlmodel

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mimir-aip/mimir-aip-go/pkg/metadatastore"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"github.com/mimir-aip/mimir-aip-go/pkg/ontology"
	"github.com/mimir-aip/mimir-aip-go/pkg/queue"
	"github.com/mimir-aip/mimir-aip-go/pkg/storage"
)

func setupTrainingService(t *testing.T) (*Service, func()) {
	t.Helper()

	store, err := metadatastore.NewSQLiteStore(filepath.Join(t.TempDir(), "mlmodel.db"))
	if err != nil {
		t.Fatalf("failed to create metadata store: %v", err)
	}

	now := time.Now().UTC()
	project := &models.Project{
		ID:          "project-1",
		Name:        "project-1",
		Description: "test project",
		Version:     "v1",
		Status:      models.ProjectStatusActive,
		Metadata: models.ProjectMetadata{
			CreatedAt: now,
			UpdatedAt: now,
		},
	}
	if err := store.SaveProject(project); err != nil {
		t.Fatalf("failed to save project: %v", err)
	}

	onto := &models.Ontology{
		ID:          "ontology-1",
		ProjectID:   project.ID,
		Name:        "ontology-1",
		Description: "test ontology",
		Version:     "1.0",
		Content:     "@prefix : <http://example.org/> .",
		Status:      "active",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := store.SaveOntology(onto); err != nil {
		t.Fatalf("failed to save ontology: %v", err)
	}

	q, err := queue.NewQueue(store)
	if err != nil {
		t.Fatalf("failed to create queue: %v", err)
	}

	storageSvc := storage.NewService(store)
	ontologySvc := ontology.NewService(store)
	return NewService(store, ontologySvc, storageSvc, q), func() {
		_ = q.Close()
		_ = store.Close()
	}
}

func TestStartTrainingPersistsTrainingTaskID(t *testing.T) {
	svc, cleanup := setupTrainingService(t)
	defer cleanup()

	model, err := svc.CreateModel(&models.ModelCreateRequest{
		ProjectID:   "project-1",
		OntologyID:  "ontology-1",
		Name:        "classifier",
		Description: "test model",
		Type:        models.ModelTypeDecisionTree,
	})
	if err != nil {
		t.Fatalf("failed to create model: %v", err)
	}

	trainedModel, err := svc.StartTraining(&models.ModelTrainingRequest{ModelID: model.ID, StorageIDs: []string{"storage-1"}})
	if err != nil {
		t.Fatalf("StartTraining returned error: %v", err)
	}
	if trainedModel.TrainingTaskID == "" {
		t.Fatal("expected StartTraining to persist a training task id")
	}
	if trainedModel.Status != models.ModelStatusTraining {
		t.Fatalf("expected model status training, got %s", trainedModel.Status)
	}

	persisted, err := svc.GetModel(model.ID)
	if err != nil {
		t.Fatalf("failed to reload model: %v", err)
	}
	if persisted.TrainingTaskID != trainedModel.TrainingTaskID {
		t.Fatalf("expected persisted training task id %s, got %s", trainedModel.TrainingTaskID, persisted.TrainingTaskID)
	}

	task, err := svc.queue.GetWorkTask(trainedModel.TrainingTaskID)
	if err != nil {
		t.Fatalf("failed to load queued training task: %v", err)
	}
	if task.TaskSpec.ModelID != model.ID {
		t.Fatalf("expected queued task model_id %s, got %s", model.ID, task.TaskSpec.ModelID)
	}
	if task.Status != models.WorkTaskStatusQueued {
		t.Fatalf("expected queued task status, got %s", task.Status)
	}
}

func TestCompleteTrainingPersistsArtifactToConfiguredDirectory(t *testing.T) {
	svc, cleanup := setupTrainingService(t)
	defer cleanup()

	t.Setenv("MODEL_ARTIFACT_DIR", t.TempDir())

	model, err := svc.CreateModel(&models.ModelCreateRequest{
		ProjectID:   "project-1",
		OntologyID:  "ontology-1",
		Name:        "classifier",
		Description: "test model",
		Type:        models.ModelTypeDecisionTree,
	})
	if err != nil {
		t.Fatalf("failed to create model: %v", err)
	}

	artifact := []byte(`{"model_type":"decision_tree","parameters":{"model_data":{}},"metadata":{"trained_at":"2026-01-01T00:00:00Z"}}`)
	metrics := &models.PerformanceMetrics{Accuracy: 0.91, Precision: 0.88, Recall: 0.90, F1Score: 0.89}

	if err := svc.CompleteTraining(model.ID, artifact, metrics); err != nil {
		t.Fatalf("CompleteTraining returned error: %v", err)
	}

	persisted, err := svc.GetModel(model.ID)
	if err != nil {
		t.Fatalf("failed to reload model: %v", err)
	}
	if persisted.ModelArtifactPath == "" {
		t.Fatal("expected persisted model artifact path")
	}
	if persisted.TrainingTaskID != "" {
		t.Fatalf("expected training task id to clear after completion, got %s", persisted.TrainingTaskID)
	}

	artifactBytes, err := os.ReadFile(persisted.ModelArtifactPath)
	if err != nil {
		t.Fatalf("failed to read persisted artifact: %v", err)
	}
	if string(artifactBytes) != string(artifact) {
		t.Fatalf("expected persisted artifact to match uploaded bytes")
	}
}
