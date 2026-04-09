package api

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/mimir-aip/mimir-aip-go/pkg/metadatastore"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"github.com/mimir-aip/mimir-aip-go/pkg/queue"
	"github.com/mimir-aip/mimir-aip-go/pkg/storage"
)

type testStoragePlugin struct{}

func (m *testStoragePlugin) Initialize(config *models.PluginConfig) error           { return nil }
func (m *testStoragePlugin) CreateSchema(ontology *models.OntologyDefinition) error { return nil }
func (m *testStoragePlugin) Store(cir *models.CIR) (*models.StorageResult, error) {
	return &models.StorageResult{Success: true, AffectedItems: 1}, nil
}
func (m *testStoragePlugin) Retrieve(query *models.CIRQuery) ([]*models.CIR, error) {
	return []*models.CIR{}, nil
}
func (m *testStoragePlugin) Update(query *models.CIRQuery, updates *models.CIRUpdate) (*models.StorageResult, error) {
	return &models.StorageResult{Success: true, AffectedItems: 0}, nil
}
func (m *testStoragePlugin) Delete(query *models.CIRQuery) (*models.StorageResult, error) {
	return &models.StorageResult{Success: true, AffectedItems: 0}, nil
}
func (m *testStoragePlugin) GetMetadata() (*models.StorageMetadata, error) {
	return &models.StorageMetadata{StorageType: "mock"}, nil
}
func (m *testStoragePlugin) HealthCheck() (bool, error) { return true, nil }

func TestDeleteStorageConfigReturnsConflictWhenReferenced(t *testing.T) {
	store, err := metadatastore.NewSQLiteStore(filepath.Join(t.TempDir(), "storage-handler.db"))
	if err != nil {
		t.Fatalf("failed to create metadata store: %v", err)
	}
	defer store.Close()

	q, err := queue.NewQueue(nil)
	if err != nil {
		t.Fatalf("failed to create queue: %v", err)
	}
	defer q.Close()

	project := &models.Project{
		ID:          "project-storage-handler",
		Name:        "project-storage-handler",
		Description: "test project",
		Version:     "v1",
		Status:      models.ProjectStatusActive,
		Metadata: models.ProjectMetadata{
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		},
	}
	if err := store.SaveProject(project); err != nil {
		t.Fatalf("failed to save project: %v", err)
	}

	storageSvc := storage.NewService(store)
	storageSvc.RegisterPlugin("mock", &testStoragePlugin{})
	handler := NewStorageHandler(storageSvc)
	server := NewServer(q, "0", "")
	server.RegisterHandler("/api/storage/configs", handler.HandleStorageConfigs)
	server.RegisterHandler("/api/storage/configs/", handler.HandleStorageConfig)

	cfg, err := storageSvc.CreateStorageConfig(project.ID, "mock", map[string]interface{}{"connection_string": "mock://handler"})
	if err != nil {
		t.Fatalf("failed to create storage config: %v", err)
	}

	pipeline := &models.Pipeline{
		ID:        "pipeline-storage-handler",
		ProjectID: project.ID,
		Name:      "handler-ingest",
		Type:      models.PipelineTypeIngestion,
		Steps: []models.PipelineStep{{
			Name:   "store",
			Plugin: "builtin",
			Action: "store_cir",
			Parameters: map[string]interface{}{
				"storage_id": cfg.ID,
			},
		}},
		Status:    models.PipelineStatusActive,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	if err := store.SavePipeline(pipeline); err != nil {
		t.Fatalf("failed to save pipeline: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/storage/configs/"+cfg.ID, nil)
	resp := httptest.NewRecorder()
	server.mux.ServeHTTP(resp, req)
	if resp.Code != http.StatusConflict {
		t.Fatalf("expected 409 Conflict, got %d body=%s", resp.Code, resp.Body.String())
	}
}
