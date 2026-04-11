package api

import (
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

func setupMLHandlerTest(t *testing.T) (*MLModelHandler, metadatastore.MetadataStore, func()) {
	t.Helper()
	store, err := metadatastore.NewSQLiteStore(filepath.Join(t.TempDir(), "ml-handler.db"))
	if err != nil {
		t.Fatalf("failed to create sqlite store: %v", err)
	}
	q, err := queue.NewQueue(store)
	if err != nil {
		t.Fatalf("failed to create queue: %v", err)
	}
	now := time.Now().UTC()
	for _, project := range []*models.Project{
		{ID: "project-a", Name: "project-a", Description: "A", Version: "v1", Status: models.ProjectStatusActive, Metadata: models.ProjectMetadata{CreatedAt: now, UpdatedAt: now}},
		{ID: "project-b", Name: "project-b", Description: "B", Version: "v1", Status: models.ProjectStatusActive, Metadata: models.ProjectMetadata{CreatedAt: now, UpdatedAt: now}},
	} {
		if err := store.SaveProject(project); err != nil {
			t.Fatalf("failed to save project: %v", err)
		}
	}
	ontSvc := ontology.NewService(store)
	if err := store.SaveOntology(&models.Ontology{ID: "ontology-a", ProjectID: "project-a", Name: "ontology-a", Content: "@prefix : <http://example.org/mimir#> .\n@prefix owl: <http://www.w3.org/2002/07/owl#> .\n\n:Entity a owl:Class .", Status: "active", CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatalf("failed to create ontology: %v", err)
	}
	storageSvc := storage.NewService(store)
	service := mlmodel.NewService(store, ontSvc, storageSvc, q)
	return NewMLModelHandler(service), store, func() {
		_ = q.Close()
		_ = store.Close()
	}
}

func TestMLModelGetRequiresProjectOwnership(t *testing.T) {
	handler, store, cleanup := setupMLHandlerTest(t)
	defer cleanup()

	if err := store.SaveMLModel(&models.MLModel{ID: "model-1", ProjectID: "project-a", OntologyID: "ontology-a", Name: "Model", Type: models.ModelTypeDecisionTree, Status: models.ModelStatusDraft, Version: "1.0.0", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}); err != nil {
		t.Fatalf("failed to save model: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/ml-models/model-1?project_id=project-b", nil)
	resp := httptest.NewRecorder()
	handler.HandleMLModel(resp, req)
	if resp.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden, got %d body=%s", resp.Code, resp.Body.String())
	}
}

func TestMLModelDeleteReturnsConflictWhenReferenced(t *testing.T) {
	handler, store, cleanup := setupMLHandlerTest(t)
	defer cleanup()

	model := &models.MLModel{ID: "model-1", ProjectID: "project-a", OntologyID: "ontology-a", Name: "Model", Type: models.ModelTypeDecisionTree, Status: models.ModelStatusDraft, Version: "1.0.0", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	if err := store.SaveMLModel(model); err != nil {
		t.Fatalf("failed to save model: %v", err)
	}
	if err := store.SaveDigitalTwin(&models.DigitalTwin{ID: "twin-1", ProjectID: "project-a", OntologyID: "ontology-a", Name: "Twin", Status: "active", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}); err != nil {
		t.Fatalf("failed to save twin: %v", err)
	}
	action := &models.Action{ID: "action-1", DigitalTwinID: "twin-1", Name: "Action", Enabled: true, Condition: &models.ActionCondition{ModelID: model.ID, Operator: "gt", Threshold: 0.8}, Trigger: &models.ActionTrigger{PipelineID: "pipeline-1"}, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	if err := store.SaveAction(action); err != nil {
		t.Fatalf("failed to save action: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/ml-models/model-1?project_id=project-a", nil)
	resp := httptest.NewRecorder()
	handler.HandleMLModel(resp, req)
	if resp.Code != http.StatusConflict {
		t.Fatalf("expected 409 Conflict, got %d body=%s", resp.Code, resp.Body.String())
	}
}
