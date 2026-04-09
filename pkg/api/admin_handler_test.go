package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	adminpkg "github.com/mimir-aip/mimir-aip-go/pkg/admin"
	"github.com/mimir-aip/mimir-aip-go/pkg/metadatastore"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"github.com/mimir-aip/mimir-aip-go/pkg/queue"
)

func TestFactoryResetEndpointClearsMetadataAndQueue(t *testing.T) {
	store, err := metadatastore.NewSQLiteStore(filepath.Join(t.TempDir(), "admin-reset.db"))
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
	project := &models.Project{ID: "project-1", Name: "project-1", Description: "test", Version: "v1", Status: models.ProjectStatusActive, Metadata: models.ProjectMetadata{CreatedAt: now, UpdatedAt: now}}
	if err := store.SaveProject(project); err != nil {
		t.Fatalf("failed to save project: %v", err)
	}
	if err := q.Enqueue(&models.WorkTask{ID: "task-1", Type: models.WorkTaskTypePipelineExecution, Status: models.WorkTaskStatusCompleted, Priority: 1, SubmittedAt: now, ProjectID: project.ID}); err != nil {
		t.Fatalf("failed to enqueue historical task: %v", err)
	}

	handler := NewAdminHandler(adminpkg.NewService(store, q))
	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/admin/settings/factory-reset", nil)
	handler.HandleAdminSettings(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200 resetting metadata, got %d body=%s", resp.Code, resp.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode reset response: %v", err)
	}
	if payload["message"] == "" {
		t.Fatalf("expected reset response message, got %#v", payload)
	}

	projects, err := store.ListProjects()
	if err != nil {
		t.Fatalf("failed to list projects after reset: %v", err)
	}
	if len(projects) != 0 {
		t.Fatalf("expected no projects after reset, got %d", len(projects))
	}
	tasks, err := q.ListWorkTasks()
	if err != nil {
		t.Fatalf("failed to list tasks after reset: %v", err)
	}
	if len(tasks) != 0 {
		t.Fatalf("expected queue to be empty after reset, got %d tasks", len(tasks))
	}
}

func TestFactoryResetEndpointBlocksQueuedOrActiveTasks(t *testing.T) {
	store, err := metadatastore.NewSQLiteStore(filepath.Join(t.TempDir(), "admin-reset-blocked.db"))
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
	project := &models.Project{ID: "project-1", Name: "project-1", Description: "test", Version: "v1", Status: models.ProjectStatusActive, Metadata: models.ProjectMetadata{CreatedAt: now, UpdatedAt: now}}
	if err := store.SaveProject(project); err != nil {
		t.Fatalf("failed to save project: %v", err)
	}
	if err := q.Enqueue(&models.WorkTask{ID: "task-queued", Type: models.WorkTaskTypePipelineExecution, Status: models.WorkTaskStatusQueued, Priority: 1, SubmittedAt: now, ProjectID: project.ID}); err != nil {
		t.Fatalf("failed to enqueue queued task: %v", err)
	}

	handler := NewAdminHandler(adminpkg.NewService(store, q))
	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/admin/settings/factory-reset", nil)
	handler.HandleAdminSettings(resp, req)
	if resp.Code != http.StatusConflict {
		t.Fatalf("expected 409 when reset is blocked, got %d body=%s", resp.Code, resp.Body.String())
	}
}
