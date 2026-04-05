package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/mimir-aip/mimir-aip-go/pkg/digitaltwin"
	"github.com/mimir-aip/mimir-aip-go/pkg/metadatastore"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"github.com/mimir-aip/mimir-aip-go/pkg/ontology"
	"github.com/mimir-aip/mimir-aip-go/pkg/queue"
	"github.com/mimir-aip/mimir-aip-go/pkg/storage"
)

func TestDigitalTwinSyncHistoryEndpoints(t *testing.T) {
	store, err := metadatastore.NewSQLiteStore(filepath.Join(t.TempDir(), "dt-history.db"))
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
	ont := &models.Ontology{ID: "ontology-1", ProjectID: project.ID, Name: "ontology-1", Description: "test", Version: "1.0", Content: "@prefix : <http://example.org/> .", Status: "active", CreatedAt: now, UpdatedAt: now}
	if err := store.SaveOntology(ont); err != nil {
		t.Fatalf("failed to save ontology: %v", err)
	}
	service := digitaltwin.NewService(store, nil, ontology.NewService(store), storage.NewService(store), nil, q)
	handler := NewDigitalTwinHandler(service, nil, nil)

	run := &models.TwinSyncRun{ID: "run-1", DigitalTwinID: "twin-1", TriggerType: "manual", TriggeredBy: "tester", StartedAt: now, Status: "completed", Summary: map[string]interface{}{"processed_sources": 1}}
	twin := &models.DigitalTwin{ID: "twin-1", ProjectID: project.ID, OntologyID: ont.ID, Name: "Twin", Status: "active", CreatedAt: now, UpdatedAt: now}
	if err := store.SaveDigitalTwin(twin); err != nil {
		t.Fatalf("failed to save twin: %v", err)
	}
	if err := store.SaveTwinSyncRun(run); err != nil {
		t.Fatalf("failed to save sync run: %v", err)
	}

	listResp := httptest.NewRecorder()
	listReq := httptest.NewRequest(http.MethodGet, "/api/digital-twins/twin-1/history/runs", nil)
	handler.HandleDigitalTwin(listResp, listReq)
	if listResp.Code != http.StatusOK {
		t.Fatalf("expected 200 listing sync runs, got %d body=%s", listResp.Code, listResp.Body.String())
	}
	var runs []map[string]interface{}
	if err := json.Unmarshal(listResp.Body.Bytes(), &runs); err != nil {
		t.Fatalf("failed to decode list response: %v", err)
	}
	if len(runs) != 1 || runs[0]["id"] != "run-1" {
		t.Fatalf("unexpected sync runs payload: %#v", runs)
	}

	getResp := httptest.NewRecorder()
	getReq := httptest.NewRequest(http.MethodGet, "/api/digital-twins/twin-1/history/runs/run-1", nil)
	handler.HandleDigitalTwin(getResp, getReq)
	if getResp.Code != http.StatusOK {
		t.Fatalf("expected 200 getting sync run, got %d body=%s", getResp.Code, getResp.Body.String())
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(getResp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode get response: %v", err)
	}
	if payload["id"] != "run-1" || payload["triggered_by"] != "tester" {
		t.Fatalf("unexpected sync run payload: %#v", payload)
	}
}
