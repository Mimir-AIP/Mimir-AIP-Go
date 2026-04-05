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
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"github.com/mimir-aip/mimir-aip-go/pkg/pipeline"
	"github.com/mimir-aip/mimir-aip-go/pkg/queue"
)

func setupPipelineHandlerTest(t *testing.T) (*PipelineHandler, metadatastore.MetadataStore, *queue.Queue, func()) {
	t.Helper()
	store, err := metadatastore.NewSQLiteStore(filepath.Join(t.TempDir(), "pipeline-handler.db"))
	if err != nil {
		t.Fatalf("failed to create sqlite store: %v", err)
	}
	q, err := queue.NewQueue(store)
	if err != nil {
		t.Fatalf("failed to create queue: %v", err)
	}
	now := time.Now().UTC()
	project := &models.Project{ID: "project-1", Name: "project-1", Description: "test project", Version: "v1", Status: models.ProjectStatusActive, Metadata: models.ProjectMetadata{CreatedAt: now, UpdatedAt: now}}
	if err := store.SaveProject(project); err != nil {
		t.Fatalf("failed to save project: %v", err)
	}
	service := pipeline.NewService(store)
	return NewPipelineHandler(service, q), store, q, func() {
		_ = q.Close()
		_ = store.Close()
	}
}

func TestCreatePipelineRedactsWebhookSecret(t *testing.T) {
	handler, _, _, cleanup := setupPipelineHandlerTest(t)
	defer cleanup()

	body, _ := json.Marshal(models.PipelineCreateRequest{
		ProjectID:     "project-1",
		Name:          "ingest",
		Type:          models.PipelineTypeIngestion,
		Steps:         []models.PipelineStep{{Name: "step-1", Plugin: "default", Action: "noop"}},
		TriggerConfig: &models.PipelineTriggerConfig{AllowManual: true, Webhook: true, Secret: "super-secret"},
	})
	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/pipelines", bytes.NewReader(body))
	handler.HandlePipelines(resp, req)
	if resp.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d", resp.Code)
	}
	var payload map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	trigger, ok := payload["trigger_config"].(map[string]any)
	if !ok {
		t.Fatalf("expected trigger_config in response, got %#v", payload)
	}
	if secret, ok := trigger["secret"]; ok && secret != "" {
		t.Fatalf("expected webhook secret to be redacted, got %#v", trigger)
	}
}

func TestPipelineWebhookQueuesExecution(t *testing.T) {
	handler, store, q, cleanup := setupPipelineHandlerTest(t)
	defer cleanup()

	pipelineRecord := &models.Pipeline{
		ID:            "pipeline-1",
		ProjectID:     "project-1",
		Name:          "ingest",
		Type:          models.PipelineTypeIngestion,
		Steps:         []models.PipelineStep{{Name: "step-1", Plugin: "default", Action: "noop"}},
		TriggerConfig: &models.PipelineTriggerConfig{AllowManual: false, Webhook: true, Secret: "token-123"},
		Status:        models.PipelineStatusActive,
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}
	if err := store.SavePipeline(pipelineRecord); err != nil {
		t.Fatalf("failed to save pipeline: %v", err)
	}

	triggerBody, _ := json.Marshal(models.PipelineTriggerRequest{Parameters: map[string]any{"batch": "nightly"}, SourceEventID: "evt-1"})
	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/pipelines/pipeline-1/webhook", bytes.NewReader(triggerBody))
	req.Header.Set("X-Mimir-Webhook-Token", "token-123")
	handler.HandlePipeline(resp, req)
	if resp.Code != http.StatusAccepted {
		t.Fatalf("expected 202 Accepted, got %d body=%s", resp.Code, resp.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	workTaskID, _ := payload["work_task_id"].(string)
	if workTaskID == "" {
		t.Fatalf("expected work_task_id, got %#v", payload)
	}
	queuedTask, err := q.GetWorkTask(workTaskID)
	if err != nil {
		t.Fatalf("expected queued work task: %v", err)
	}
	if queuedTask.TaskSpec.Parameters["trigger_type"] != "webhook" {
		t.Fatalf("expected webhook trigger type, got %#v", queuedTask.TaskSpec.Parameters["trigger_type"])
	}
	if queuedTask.TaskSpec.Parameters["source_event_id"] != "evt-1" {
		t.Fatalf("expected source_event_id to be preserved, got %#v", queuedTask.TaskSpec.Parameters["source_event_id"])
	}
}

func TestPipelineTriggerRespectsManualToggle(t *testing.T) {
	handler, store, _, cleanup := setupPipelineHandlerTest(t)
	defer cleanup()

	pipelineRecord := &models.Pipeline{
		ID:            "pipeline-2",
		ProjectID:     "project-1",
		Name:          "ingest",
		Type:          models.PipelineTypeIngestion,
		Steps:         []models.PipelineStep{{Name: "step-1", Plugin: "default", Action: "noop"}},
		TriggerConfig: &models.PipelineTriggerConfig{AllowManual: false},
		Status:        models.PipelineStatusActive,
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}
	if err := store.SavePipeline(pipelineRecord); err != nil {
		t.Fatalf("failed to save pipeline: %v", err)
	}

	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/pipelines/pipeline-2/trigger", bytes.NewReader([]byte(`{"trigger_type":"manual"}`)))
	handler.HandlePipeline(resp, req)
	if resp.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden, got %d body=%s", resp.Code, resp.Body.String())
	}
}
