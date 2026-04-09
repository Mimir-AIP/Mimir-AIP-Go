package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/mimir-aip/mimir-aip-go/pkg/metadatastore"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"github.com/mimir-aip/mimir-aip-go/pkg/project"
	"github.com/mimir-aip/mimir-aip-go/pkg/queue"
)

func TestProjectCloneRouteUsesProjectScopedPath(t *testing.T) {
	store, err := metadatastore.NewSQLiteStore(filepath.Join(t.TempDir(), "project-handler.db"))
	if err != nil {
		t.Fatalf("failed to create metadata store: %v", err)
	}
	defer store.Close()

	q, err := queue.NewQueue(nil)
	if err != nil {
		t.Fatalf("failed to create queue: %v", err)
	}
	defer q.Close()

	service := project.NewService(store)
	handler := NewProjectHandler(service, nil)
	server := NewServer(q, "0", "")
	server.RegisterHandler("/api/projects", handler.HandleProjects)
	server.RegisterHandler("/api/projects/", handler.HandleProject)

	source, err := service.Create(&models.ProjectCreateRequest{Name: "source-project"})
	if err != nil {
		t.Fatalf("failed to create source project: %v", err)
	}

	body, err := json.Marshal(map[string]string{"name": "api-clone"})
	if err != nil {
		t.Fatalf("failed to marshal clone body: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/projects/"+source.ID+"/clone", bytes.NewReader(body))
	resp := httptest.NewRecorder()
	server.mux.ServeHTTP(resp, req)
	if resp.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created from project-scoped clone route, got %d body=%s", resp.Code, resp.Body.String())
	}

	var cloned models.Project
	if err := json.Unmarshal(resp.Body.Bytes(), &cloned); err != nil {
		t.Fatalf("failed to decode clone response: %v", err)
	}
	if cloned.Name != "api-clone" {
		t.Fatalf("expected cloned project name api-clone, got %s", cloned.Name)
	}
	if cloned.Status != models.ProjectStatusDraft {
		t.Fatalf("expected cloned project status draft, got %s", cloned.Status)
	}

	legacyReq := httptest.NewRequest(http.MethodPost, "/api/projects/clone", bytes.NewReader(body))
	legacyResp := httptest.NewRecorder()
	server.mux.ServeHTTP(legacyResp, legacyReq)
	if legacyResp.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected legacy clone path to be rejected with 405, got %d body=%s", legacyResp.Code, legacyResp.Body.String())
	}
}

func TestProjectArchiveRouteUsesSeparateEndpoint(t *testing.T) {
	store, err := metadatastore.NewSQLiteStore(filepath.Join(t.TempDir(), "project-archive.db"))
	if err != nil {
		t.Fatalf("failed to create metadata store: %v", err)
	}
	defer store.Close()

	q, err := queue.NewQueue(nil)
	if err != nil {
		t.Fatalf("failed to create queue: %v", err)
	}
	defer q.Close()

	service := project.NewService(store)
	handler := NewProjectHandler(service, nil)
	server := NewServer(q, "0", "")
	server.RegisterHandler("/api/projects", handler.HandleProjects)
	server.RegisterHandler("/api/projects/", handler.HandleProject)

	source, err := service.Create(&models.ProjectCreateRequest{Name: "archive-source"})
	if err != nil {
		t.Fatalf("failed to create source project: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/projects/"+source.ID+"/archive", nil)
	resp := httptest.NewRecorder()
	server.mux.ServeHTTP(resp, req)
	if resp.Code != http.StatusNoContent {
		t.Fatalf("expected 204 No Content from archive route, got %d body=%s", resp.Code, resp.Body.String())
	}

	archived, err := service.Get(source.ID)
	if err != nil {
		t.Fatalf("failed to reload archived project: %v", err)
	}
	if archived.Status != models.ProjectStatusArchived {
		t.Fatalf("expected archived status, got %s", archived.Status)
	}
}

func TestProjectDeleteRoutePermanentlyDeletesProject(t *testing.T) {
	store, err := metadatastore.NewSQLiteStore(filepath.Join(t.TempDir(), "project-delete.db"))
	if err != nil {
		t.Fatalf("failed to create metadata store: %v", err)
	}
	defer store.Close()

	q, err := queue.NewQueue(store)
	if err != nil {
		t.Fatalf("failed to create queue: %v", err)
	}
	defer q.Close()

	service := project.NewService(store)
	service.SetTaskCleaner(q)
	handler := NewProjectHandler(service, nil)
	server := NewServer(q, "0", "")
	server.RegisterHandler("/api/projects", handler.HandleProjects)
	server.RegisterHandler("/api/projects/", handler.HandleProject)

	projectRecord, err := service.Create(&models.ProjectCreateRequest{Name: "delete-source"})
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}
	if err := q.Enqueue(&models.WorkTask{ID: "delete-task", Type: models.WorkTaskTypePipelineExecution, Status: models.WorkTaskStatusQueued, Priority: 1, SubmittedAt: projectRecord.Metadata.CreatedAt, ProjectID: projectRecord.ID}); err != nil {
		t.Fatalf("failed to enqueue work task: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/projects/"+projectRecord.ID, nil)
	resp := httptest.NewRecorder()
	server.mux.ServeHTTP(resp, req)
	if resp.Code != http.StatusNoContent {
		t.Fatalf("expected 204 No Content from delete route, got %d body=%s", resp.Code, resp.Body.String())
	}

	if _, err := service.Get(projectRecord.ID); err == nil {
		t.Fatal("expected project to be permanently deleted")
	}
	tasks, err := q.ListWorkTasks()
	if err != nil {
		t.Fatalf("failed to inspect queue tasks: %v", err)
	}
	if len(tasks) != 0 {
		t.Fatalf("expected queue tasks for deleted project to be removed, got %d", len(tasks))
	}
}
