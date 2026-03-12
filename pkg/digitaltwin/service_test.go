package digitaltwin

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/mimir-aip/mimir-aip-go/pkg/metadatastore"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"github.com/mimir-aip/mimir-aip-go/pkg/ontology"
	"github.com/mimir-aip/mimir-aip-go/pkg/queue"
	"github.com/mimir-aip/mimir-aip-go/pkg/storage"
)

func setupDigitalTwinService(t *testing.T) (*Service, *queue.Queue, func()) {
	t.Helper()
	store, err := metadatastore.NewSQLiteStore(filepath.Join(t.TempDir(), "digitaltwin.db"))
	if err != nil {
		t.Fatalf("failed to create metadata store: %v", err)
	}
	q, err := queue.NewQueue()
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
