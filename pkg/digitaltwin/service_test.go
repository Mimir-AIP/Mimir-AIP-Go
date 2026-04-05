package digitaltwin

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/mimir-aip/mimir-aip-go/pkg/metadatastore"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"github.com/mimir-aip/mimir-aip-go/pkg/ontology"
	"github.com/mimir-aip/mimir-aip-go/pkg/queue"
	"github.com/mimir-aip/mimir-aip-go/pkg/storage"
)

func seedDigitalTwinProject(t *testing.T, store metadatastore.MetadataStore, projectID, ontologyID string) {
	t.Helper()
	now := time.Now().UTC()
	project := &models.Project{
		ID:          projectID,
		Name:        projectID,
		Description: "test project",
		Version:     "v1",
		Status:      models.ProjectStatusActive,
		Metadata:    models.ProjectMetadata{CreatedAt: now, UpdatedAt: now},
	}
	if err := store.SaveProject(project); err != nil {
		t.Fatalf("failed to save project: %v", err)
	}
	ontology := &models.Ontology{
		ID:          ontologyID,
		ProjectID:   projectID,
		Name:        ontologyID,
		Description: "test ontology",
		Version:     "1.0",
		Content:     "@prefix : <http://example.org/> .",
		Status:      "active",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := store.SaveOntology(ontology); err != nil {
		t.Fatalf("failed to save ontology: %v", err)
	}
}

func setupDigitalTwinService(t *testing.T) (*Service, *queue.Queue, func()) {
	t.Helper()
	store, err := metadatastore.NewSQLiteStore(filepath.Join(t.TempDir(), "digitaltwin.db"))
	if err != nil {
		t.Fatalf("failed to create metadata store: %v", err)
	}
	q, err := queue.NewQueue(store)
	if err != nil {
		t.Fatalf("failed to create queue: %v", err)
	}
	service := NewService(store, nil, ontology.NewService(store), storage.NewService(store), nil, q)
	return service, q, func() {
		_ = q.Close()
		_ = store.Close()
	}
}

func TestEnqueueSyncQueuesWorkAndMarksTwinSyncing(t *testing.T) {
	service, q, cleanup := setupDigitalTwinService(t)
	defer cleanup()

	seedDigitalTwinProject(t, service.store, "project-1", "ontology-1")

	now := time.Now().UTC()
	twin := &models.DigitalTwin{
		ID:         "twin-1",
		ProjectID:  "project-1",
		OntologyID: "ontology-1",
		Name:       "Factory Twin",
		Status:     "active",
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := service.store.SaveDigitalTwin(twin); err != nil {
		t.Fatalf("failed to seed digital twin: %v", err)
	}

	task, err := service.EnqueueSync(twin.ID)
	if err != nil {
		t.Fatalf("EnqueueSync returned error: %v", err)
	}
	if task.Type != models.WorkTaskTypeDigitalTwinProcessing {
		t.Fatalf("expected digital_twin_processing task type, got %s", task.Type)
	}
	if task.ProjectID != twin.ProjectID {
		t.Fatalf("expected task project %s, got %s", twin.ProjectID, task.ProjectID)
	}
	queuedTask, err := q.GetWorkTask(task.ID)
	if err != nil {
		t.Fatalf("expected queued task to be retrievable: %v", err)
	}
	if queuedTask.Status != models.WorkTaskStatusQueued {
		t.Fatalf("expected queued status, got %s", queuedTask.Status)
	}
	updatedTwin, err := service.store.GetDigitalTwin(twin.ID)
	if err != nil {
		t.Fatalf("failed to reload digital twin: %v", err)
	}
	if updatedTwin.Status != "syncing" {
		t.Fatalf("expected twin status syncing, got %s", updatedTwin.Status)
	}
}

func TestEntityHistoryCapturesUpdates(t *testing.T) {
	service, _, cleanup := setupDigitalTwinService(t)
	defer cleanup()

	seedDigitalTwinProject(t, service.store, "project-1", "ontology-1")
	now := time.Now().UTC()
	twin := &models.DigitalTwin{ID: "twin-1", ProjectID: "project-1", OntologyID: "ontology-1", Name: "Factory Twin", Status: "active", CreatedAt: now, UpdatedAt: now}
	if err := service.store.SaveDigitalTwin(twin); err != nil {
		t.Fatalf("failed to seed digital twin: %v", err)
	}
	entity := &models.Entity{
		ID:            "entity-1",
		DigitalTwinID: twin.ID,
		Type:          "Machine",
		Attributes:    map[string]interface{}{"temperature": 80},
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := service.store.SaveEntity(entity); err != nil {
		t.Fatalf("failed to save entity: %v", err)
	}

	updated, err := service.UpdateEntity(entity.ID, &models.EntityUpdateRequest{Attributes: map[string]interface{}{"temperature": 91}})
	if err != nil {
		t.Fatalf("UpdateEntity returned error: %v", err)
	}
	if updated.Attributes["temperature"] != 91 {
		t.Fatalf("expected updated temperature 91, got %#v", updated.Attributes["temperature"])
	}

	history, err := service.GetEntityHistory(entity.ID, 10)
	if err != nil {
		t.Fatalf("GetEntityHistory returned error: %v", err)
	}
	if len(history) < 2 {
		t.Fatalf("expected at least 2 entity revisions, got %d", len(history))
	}
	if history[0].Revision <= history[1].Revision {
		t.Fatalf("expected newest revision first, got %#v", history)
	}
	if fmt.Sprintf("%v", history[0].Attributes["temperature"]) != "91" {
		t.Fatalf("expected latest revision temperature 91, got %#v", history[0].Attributes["temperature"])
	}
}
