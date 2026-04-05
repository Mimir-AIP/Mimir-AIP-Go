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

type twinSampleStoragePlugin struct {
	sample []*models.CIR
}

func (m *twinSampleStoragePlugin) Initialize(config *models.PluginConfig) error           { return nil }
func (m *twinSampleStoragePlugin) CreateSchema(ontology *models.OntologyDefinition) error { return nil }
func (m *twinSampleStoragePlugin) Store(cir *models.CIR) (*models.StorageResult, error) {
	return &models.StorageResult{Success: true, AffectedItems: 1}, nil
}
func (m *twinSampleStoragePlugin) Retrieve(query *models.CIRQuery) ([]*models.CIR, error) {
	return m.sample, nil
}
func (m *twinSampleStoragePlugin) Update(query *models.CIRQuery, updates *models.CIRUpdate) (*models.StorageResult, error) {
	return &models.StorageResult{Success: true, AffectedItems: 0}, nil
}
func (m *twinSampleStoragePlugin) Delete(query *models.CIRQuery) (*models.StorageResult, error) {
	return &models.StorageResult{Success: true, AffectedItems: 0}, nil
}
func (m *twinSampleStoragePlugin) GetMetadata() (*models.StorageMetadata, error) {
	return &models.StorageMetadata{StorageType: "twin-sample"}, nil
}
func (m *twinSampleStoragePlugin) HealthCheck() (bool, error) { return true, nil }

func setupDigitalTwinService(t *testing.T) (*Service, *storage.Service, *queue.Queue, func()) {
	t.Helper()
	store, err := metadatastore.NewSQLiteStore(filepath.Join(t.TempDir(), "digitaltwin.db"))
	if err != nil {
		t.Fatalf("failed to create metadata store: %v", err)
	}
	q, err := queue.NewQueue(store)
	if err != nil {
		t.Fatalf("failed to create queue: %v", err)
	}
	storageSvc := storage.NewService(store)
	service := NewService(store, nil, ontology.NewService(store), storageSvc, nil, q)
	return service, storageSvc, q, func() {
		_ = q.Close()
		_ = store.Close()
	}
}

func TestEnqueueSyncQueuesWorkAndMarksTwinSyncing(t *testing.T) {
	service, _, q, cleanup := setupDigitalTwinService(t)
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
	service, _, _, cleanup := setupDigitalTwinService(t)
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

func TestSyncWithStorageUsesSourcePriorityReconciliation(t *testing.T) {
	service, storageSvc, _, cleanup := setupDigitalTwinService(t)
	defer cleanup()

	seedDigitalTwinProject(t, service.store, "project-1", "ontology-1")
	now := time.Now().UTC()
	storageSvc.RegisterPlugin("sample-a", &twinSampleStoragePlugin{sample: []*models.CIR{func() *models.CIR {
		cir := models.NewCIR(models.SourceTypeDatabase, "db://a/m1", models.DataFormatJSON, map[string]interface{}{"machine_id": "M-1", "temperature": 80})
		cir.Source.Timestamp = now.Add(-5 * time.Minute)
		cir.SetParameter("entity_type", "Machine")
		return cir
	}()}})
	storageSvc.RegisterPlugin("sample-b", &twinSampleStoragePlugin{sample: []*models.CIR{func() *models.CIR {
		cir := models.NewCIR(models.SourceTypeDatabase, "db://b/m1", models.DataFormatJSON, map[string]interface{}{"machine_id": "M-1", "temperature": 95})
		cir.Source.Timestamp = now
		cir.SetParameter("entity_type", "Machine")
		return cir
	}()}})

	cfgA, err := storageSvc.CreateStorageConfig("project-1", "sample-a", map[string]interface{}{"connection_string": "mock://a"})
	if err != nil {
		t.Fatalf("failed to create storage config A: %v", err)
	}
	cfgB, err := storageSvc.CreateStorageConfig("project-1", "sample-b", map[string]interface{}{"connection_string": "mock://b"})
	if err != nil {
		t.Fatalf("failed to create storage config B: %v", err)
	}
	twin := &models.DigitalTwin{
		ID:         "twin-priority",
		ProjectID:  "project-1",
		OntologyID: "ontology-1",
		Name:       "Priority Twin",
		Status:     "active",
		Config: &models.DigitalTwinConfig{
			StorageIDs: []string{cfgA.ID, cfgB.ID},
			Reconciliation: &models.TwinReconciliationPolicy{
				Strategy:       "source_priority",
				SourcePriority: []string{cfgB.ID, cfgA.ID},
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := service.store.SaveDigitalTwin(twin); err != nil {
		t.Fatalf("failed to save digital twin: %v", err)
	}

	if err := service.SyncWithStorage(twin.ID); err != nil {
		t.Fatalf("SyncWithStorage returned error: %v", err)
	}
	entities, err := service.ListEntities(twin.ID)
	if err != nil {
		t.Fatalf("ListEntities returned error: %v", err)
	}
	if len(entities) != 1 {
		t.Fatalf("expected one reconciled entity, got %d", len(entities))
	}
	entity := entities[0]
	if fmt.Sprintf("%v", entity.Attributes["temperature"]) != "95" {
		t.Fatalf("expected higher-priority source value 95, got %#v", entity.Attributes["temperature"])
	}
	sources, ok := entity.ComputedValues["attribute_sources"].(map[string]interface{})
	if !ok || fmt.Sprintf("%v", sources["temperature"]) != cfgB.ID {
		t.Fatalf("expected attribute source %s, got %#v", cfgB.ID, entity.ComputedValues["attribute_sources"])
	}
	conflicts, ok := entity.ComputedValues["reconciliation_conflicts"].(map[string]interface{})
	if !ok || len(conflicts) == 0 {
		t.Fatalf("expected reconciliation conflict metadata, got %#v", entity.ComputedValues["reconciliation_conflicts"])
	}
}

func TestSyncWithStorageUsesFreshestReconciliation(t *testing.T) {
	service, storageSvc, _, cleanup := setupDigitalTwinService(t)
	defer cleanup()

	seedDigitalTwinProject(t, service.store, "project-1", "ontology-1")
	now := time.Now().UTC()
	storageSvc.RegisterPlugin("fresh-a", &twinSampleStoragePlugin{sample: []*models.CIR{func() *models.CIR {
		cir := models.NewCIR(models.SourceTypeDatabase, "db://a/m1", models.DataFormatJSON, map[string]interface{}{"machine_id": "M-1", "temperature": 88})
		cir.Source.Timestamp = now
		cir.SetParameter("entity_type", "Machine")
		return cir
	}()}})
	storageSvc.RegisterPlugin("fresh-b", &twinSampleStoragePlugin{sample: []*models.CIR{func() *models.CIR {
		cir := models.NewCIR(models.SourceTypeDatabase, "db://b/m1", models.DataFormatJSON, map[string]interface{}{"machine_id": "M-1", "temperature": 72})
		cir.Source.Timestamp = now.Add(-10 * time.Minute)
		cir.SetParameter("entity_type", "Machine")
		return cir
	}()}})

	cfgA, err := storageSvc.CreateStorageConfig("project-1", "fresh-a", map[string]interface{}{"connection_string": "mock://a"})
	if err != nil {
		t.Fatalf("failed to create storage config A: %v", err)
	}
	cfgB, err := storageSvc.CreateStorageConfig("project-1", "fresh-b", map[string]interface{}{"connection_string": "mock://b"})
	if err != nil {
		t.Fatalf("failed to create storage config B: %v", err)
	}
	twin := &models.DigitalTwin{
		ID:         "twin-freshest",
		ProjectID:  "project-1",
		OntologyID: "ontology-1",
		Name:       "Fresh Twin",
		Status:     "active",
		Config: &models.DigitalTwinConfig{
			StorageIDs: []string{cfgB.ID, cfgA.ID},
			Reconciliation: &models.TwinReconciliationPolicy{
				Strategy: "freshest",
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := service.store.SaveDigitalTwin(twin); err != nil {
		t.Fatalf("failed to save digital twin: %v", err)
	}
	if err := service.SyncWithStorage(twin.ID); err != nil {
		t.Fatalf("SyncWithStorage returned error: %v", err)
	}
	entities, err := service.ListEntities(twin.ID)
	if err != nil {
		t.Fatalf("ListEntities returned error: %v", err)
	}
	if len(entities) != 1 {
		t.Fatalf("expected one reconciled entity, got %d", len(entities))
	}
	if fmt.Sprintf("%v", entities[0].Attributes["temperature"]) != "88" {
		t.Fatalf("expected freshest value 88, got %#v", entities[0].Attributes["temperature"])
	}
}
