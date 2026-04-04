package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/mimir-aip/mimir-aip-go/pkg/metadatastore"
	"github.com/mimir-aip/mimir-aip-go/pkg/mlmodel"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"github.com/mimir-aip/mimir-aip-go/pkg/ontology"
	"github.com/mimir-aip/mimir-aip-go/pkg/queue"
	"github.com/mimir-aip/mimir-aip-go/pkg/storage"
)

func TestTrainingAsyncContractIsSelfContained(t *testing.T) {
	store, err := metadatastore.NewSQLiteStore(filepath.Join(t.TempDir(), "async-training.db"))
	if err != nil {
		t.Fatalf("failed to create metadata store: %v", err)
	}
	defer store.Close()

	q, err := queue.NewQueue(store)
	if err != nil {
		t.Fatalf("failed to create queue: %v", err)
	}
	defer q.Close()

	now := time.Now().UTC()
	project := &models.Project{
		ID:          "project-1",
		Name:        "project-1",
		Description: "test project",
		Version:     "v1",
		Status:      models.ProjectStatusActive,
		Metadata:    models.ProjectMetadata{CreatedAt: now, UpdatedAt: now},
	}
	if err := store.SaveProject(project); err != nil {
		t.Fatalf("failed to save project: %v", err)
	}
	ont := &models.Ontology{
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
	if err := store.SaveOntology(ont); err != nil {
		t.Fatalf("failed to save ontology: %v", err)
	}

	storageSvc := storage.NewService(store)
	mlSvc := mlmodel.NewService(store, ontology.NewService(store), storageSvc, q)
	model, err := mlSvc.CreateModel(&models.ModelCreateRequest{
		ProjectID:   project.ID,
		OntologyID:  ont.ID,
		Name:        "classifier",
		Description: "test model",
		Type:        models.ModelTypeDecisionTree,
	})
	if err != nil {
		t.Fatalf("failed to create model: %v", err)
	}

	server := NewServer(q, "0", "")
	handler := NewMLModelHandler(mlSvc)
	server.RegisterHandler("/api/ml-models/train", handler.HandleMLModelTraining)
	server.RegisterHandler("/api/ml-models/", handler.HandleMLModel)

	httpServer := httptest.NewServer(server.mux)
	defer httpServer.Close()

	body, err := json.Marshal(models.ModelTrainingRequest{ModelID: model.ID, StorageIDs: []string{"storage-1"}})
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}
	resp, err := http.Post(httpServer.URL+"/api/ml-models/train", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("failed to call training endpoint: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202 Accepted, got %d", resp.StatusCode)
	}

	var trainedModel map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&trainedModel); err != nil {
		t.Fatalf("failed to decode training response: %v", err)
	}
	trainingTaskID, _ := trainedModel["training_task_id"].(string)
	if trainingTaskID == "" {
		t.Fatalf("expected training_task_id in response, got %#v", trainedModel)
	}

	workTaskResp, err := http.Get(httpServer.URL + "/api/worktasks/" + trainingTaskID)
	if err != nil {
		t.Fatalf("failed to fetch work task: %v", err)
	}
	defer workTaskResp.Body.Close()
	if workTaskResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 fetching work task, got %d", workTaskResp.StatusCode)
	}

	var taskPayload map[string]interface{}
	if err := json.NewDecoder(workTaskResp.Body).Decode(&taskPayload); err != nil {
		t.Fatalf("failed to decode work task payload: %v", err)
	}
	if taskPayload["worktask_id"] != trainingTaskID {
		t.Fatalf("expected worktask_id %s, got %#v", trainingTaskID, taskPayload["worktask_id"])
	}
	if taskPayload["status"] != string(models.WorkTaskStatusQueued) {
		t.Fatalf("expected queued task status, got %#v", taskPayload["status"])
	}

	modelResp, err := http.Get(httpServer.URL + "/api/ml-models/" + model.ID)
	if err != nil {
		t.Fatalf("failed to fetch model: %v", err)
	}
	defer modelResp.Body.Close()
	if modelResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 fetching model, got %d", modelResp.StatusCode)
	}

	var reloadedModel map[string]interface{}
	if err := json.NewDecoder(modelResp.Body).Decode(&reloadedModel); err != nil {
		t.Fatalf("failed to decode reloaded model: %v", err)
	}
	if reloadedModel["training_task_id"] != trainingTaskID {
		t.Fatalf("expected model training_task_id %s, got %#v", trainingTaskID, reloadedModel["training_task_id"])
	}
}

func TestPersistedWorkTaskSurvivesServerRestart(t *testing.T) {
	store, err := metadatastore.NewSQLiteStore(filepath.Join(t.TempDir(), "worktask-restart.db"))
	if err != nil {
		t.Fatalf("failed to create metadata store: %v", err)
	}
	defer store.Close()

	now := time.Now().UTC()
	project := &models.Project{
		ID:          "project-1",
		Name:        "project-1",
		Description: "test project",
		Version:     "v1",
		Status:      models.ProjectStatusActive,
		Metadata:    models.ProjectMetadata{CreatedAt: now, UpdatedAt: now},
	}
	if err := store.SaveProject(project); err != nil {
		t.Fatalf("failed to save project: %v", err)
	}

	q, err := queue.NewQueue(store)
	if err != nil {
		t.Fatalf("failed to create queue: %v", err)
	}
	firstServer := httptest.NewServer(NewServer(q, "0", "").mux)
	body, err := json.Marshal(models.WorkTaskSubmissionRequest{
		Type:      models.WorkTaskTypePipelineExecution,
		ProjectID: project.ID,
		TaskSpec:  models.TaskSpec{ProjectID: project.ID, PipelineID: "pipeline-1"},
	})
	if err != nil {
		t.Fatalf("failed to marshal task submission: %v", err)
	}
	resp, err := http.Post(firstServer.URL+"/api/worktasks", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("failed to submit work task: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d", resp.StatusCode)
	}
	var submitted map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&submitted); err != nil {
		t.Fatalf("failed to decode submitted task: %v", err)
	}
	taskID, _ := submitted["worktask_id"].(string)
	if taskID == "" {
		t.Fatalf("expected worktask_id in submission response, got %#v", submitted)
	}
	firstServer.Close()

	reloadedQueue, err := queue.NewQueue(store)
	if err != nil {
		t.Fatalf("failed to reload queue: %v", err)
	}
	secondServer := httptest.NewServer(NewServer(reloadedQueue, "0", "").mux)
	defer secondServer.Close()

	getResp, err := http.Get(secondServer.URL + "/api/worktasks/" + taskID)
	if err != nil {
		t.Fatalf("failed to get persisted task after restart: %v", err)
	}
	defer getResp.Body.Close()
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK after restart, got %d", getResp.StatusCode)
	}
	var persisted map[string]interface{}
	if err := json.NewDecoder(getResp.Body).Decode(&persisted); err != nil {
		t.Fatalf("failed to decode persisted task: %v", err)
	}
	if persisted["worktask_id"] != taskID {
		t.Fatalf("expected persisted worktask_id %s, got %#v", taskID, persisted["worktask_id"])
	}
	if persisted["status"] != string(models.WorkTaskStatusQueued) {
		t.Fatalf("expected persisted queued status, got %#v", persisted["status"])
	}
}
