package storage

import (
	"testing"

	"github.com/mimir-aip/mimir-aip-go/pkg/metadatastore"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

type mockStoragePlugin struct{}

func (m *mockStoragePlugin) Initialize(config *models.PluginConfig) error           { return nil }
func (m *mockStoragePlugin) CreateSchema(ontology *models.OntologyDefinition) error { return nil }
func (m *mockStoragePlugin) Store(cir *models.CIR) (*models.StorageResult, error) {
	return &models.StorageResult{Success: true, AffectedItems: 1}, nil
}
func (m *mockStoragePlugin) Retrieve(query *models.CIRQuery) ([]*models.CIR, error) {
	return []*models.CIR{}, nil
}
func (m *mockStoragePlugin) Update(query *models.CIRQuery, updates *models.CIRUpdate) (*models.StorageResult, error) {
	return &models.StorageResult{Success: true, AffectedItems: 0}, nil
}
func (m *mockStoragePlugin) Delete(query *models.CIRQuery) (*models.StorageResult, error) {
	return &models.StorageResult{Success: true, AffectedItems: 0}, nil
}
func (m *mockStoragePlugin) GetMetadata() (*models.StorageMetadata, error) {
	return &models.StorageMetadata{StorageType: "mock"}, nil
}
func (m *mockStoragePlugin) HealthCheck() (bool, error) { return true, nil }

func TestInitializeStorageDoesNotWriteFakeOntologyID(t *testing.T) {
	store, err := metadatastore.NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create metadata store: %v", err)
	}
	defer store.Close()

	svc := NewService(store)
	svc.RegisterPlugin("mock", &mockStoragePlugin{})

	cfg, err := svc.CreateStorageConfig("project-1", "mock", map[string]interface{}{"connection_string": "mock://"})
	if err != nil {
		t.Fatalf("failed to create storage config: %v", err)
	}

	err = svc.InitializeStorage(cfg.ID, &models.OntologyDefinition{Entities: []models.EntityDefinition{}})
	if err != nil {
		t.Fatalf("InitializeStorage returned error: %v", err)
	}

	updated, err := svc.GetStorageConfig(cfg.ID)
	if err != nil {
		t.Fatalf("failed to reload storage config: %v", err)
	}
	if updated.OntologyID != "" {
		t.Fatalf("expected ontology_id to remain empty, got %q", updated.OntologyID)
	}
}
