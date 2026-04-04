package connectors

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mimir-aip/mimir-aip-go/pkg/metadatastore"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"github.com/mimir-aip/mimir-aip-go/pkg/pipeline"
	"github.com/mimir-aip/mimir-aip-go/pkg/queue"
	"github.com/mimir-aip/mimir-aip-go/pkg/scheduler"
	"github.com/mimir-aip/mimir-aip-go/pkg/storage"
)

func saveConnectorProject(t *testing.T, store *metadatastore.SQLiteStore, projectID string) {
	t.Helper()
	project := &models.Project{
		ID:          projectID,
		Name:        projectID,
		Description: "test project",
		Version:     "v1",
		Status:      models.ProjectStatusActive,
		Metadata: models.ProjectMetadata{
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		},
	}
	if err := store.SaveProject(project); err != nil {
		t.Fatalf("failed to save project %s: %v", projectID, err)
	}
}

type connectorTestStoragePlugin struct{}

func (p *connectorTestStoragePlugin) Initialize(config *models.PluginConfig) error { return nil }
func (p *connectorTestStoragePlugin) CreateSchema(ontology *models.OntologyDefinition) error {
	return nil
}
func (p *connectorTestStoragePlugin) Store(cir *models.CIR) (*models.StorageResult, error) {
	return &models.StorageResult{Success: true, AffectedItems: 1}, nil
}
func (p *connectorTestStoragePlugin) Retrieve(query *models.CIRQuery) ([]*models.CIR, error) {
	return nil, nil
}
func (p *connectorTestStoragePlugin) Update(query *models.CIRQuery, updates *models.CIRUpdate) (*models.StorageResult, error) {
	return &models.StorageResult{Success: true, AffectedItems: 0}, nil
}
func (p *connectorTestStoragePlugin) Delete(query *models.CIRQuery) (*models.StorageResult, error) {
	return &models.StorageResult{Success: true, AffectedItems: 0}, nil
}
func (p *connectorTestStoragePlugin) GetMetadata() (*models.StorageMetadata, error) {
	return &models.StorageMetadata{StorageType: "test"}, nil
}
func (p *connectorTestStoragePlugin) HealthCheck() (bool, error) { return true, nil }

func setupConnectorService(t *testing.T) (*Service, *pipeline.Service, *storage.Service, func()) {
	t.Helper()
	store, err := metadatastore.NewSQLiteStore(filepath.Join(t.TempDir(), "connectors.db"))
	if err != nil {
		t.Fatalf("failed to create metadata store: %v", err)
	}
	saveConnectorProject(t, store, "project-a")
	saveConnectorProject(t, store, "project-b")

	pipelineSvc := pipeline.NewService(store)
	q, err := queue.NewQueue(store)
	if err != nil {
		t.Fatalf("failed to create queue: %v", err)
	}
	schedulerSvc := scheduler.NewService(store, pipelineSvc, q)
	storageSvc := storage.NewService(store)
	storageSvc.RegisterPlugin("test", &connectorTestStoragePlugin{})
	service := NewService(pipelineSvc, schedulerSvc, storageSvc)
	return service, pipelineSvc, storageSvc, func() {
		_ = q.Close()
		_ = store.Close()
	}
}

func TestMaterializeRejectsCrossProjectStorage(t *testing.T) {
	service, _, storageSvc, cleanup := setupConnectorService(t)
	defer cleanup()

	foreignStorage, err := storageSvc.CreateStorageConfig("project-b", "test", map[string]any{"dsn": "mock://foreign"})
	if err != nil {
		t.Fatalf("failed to create storage config: %v", err)
	}

	_, err = service.Materialize(&models.ConnectorSetupRequest{
		ProjectID: "project-a",
		Kind:      "rss_poll",
		Name:      "foreign storage connector",
		StorageID: foreignStorage.ID,
		SourceConfig: map[string]any{
			"url": "https://example.com/feed.xml",
		},
	})
	if err == nil {
		t.Fatal("expected cross-project storage validation error")
	}
	if !strings.Contains(err.Error(), "belongs to project") {
		t.Fatalf("expected project ownership error, got %v", err)
	}
}

func TestMaterializeRollsBackPipelineWhenScheduleCreationFails(t *testing.T) {
	service, pipelineSvc, storageSvc, cleanup := setupConnectorService(t)
	defer cleanup()

	localStorage, err := storageSvc.CreateStorageConfig("project-a", "test", map[string]any{"dsn": "mock://local"})
	if err != nil {
		t.Fatalf("failed to create storage config: %v", err)
	}

	_, err = service.Materialize(&models.ConnectorSetupRequest{
		ProjectID: "project-a",
		Kind:      "rss_poll",
		Name:      "broken schedule connector",
		StorageID: localStorage.ID,
		SourceConfig: map[string]any{
			"url": "https://example.com/feed.xml",
		},
		Schedule: &models.ConnectorScheduleRequest{
			CronSchedule: "not-a-valid-cron",
			Enabled:      true,
		},
	})
	if err == nil {
		t.Fatal("expected schedule creation failure")
	}
	if !strings.Contains(err.Error(), "failed to create connector schedule") {
		t.Fatalf("expected schedule failure error, got %v", err)
	}

	pipelines, err := pipelineSvc.ListByProject("project-a")
	if err != nil {
		t.Fatalf("failed to list project pipelines: %v", err)
	}
	if len(pipelines) != 0 {
		t.Fatalf("expected rollback to remove created pipeline, found %d pipelines", len(pipelines))
	}
}
