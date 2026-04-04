package storage

import (
	"testing"
	"time"

	"github.com/mimir-aip/mimir-aip-go/pkg/metadatastore"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

func saveTestProject(t *testing.T, store *metadatastore.SQLiteStore, projectID string) {
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
		t.Fatalf("failed to save test project %s: %v", projectID, err)
	}
}

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

type statefulStoragePlugin struct {
	connectionString string
}

func (m *statefulStoragePlugin) Initialize(config *models.PluginConfig) error {
	m.connectionString = config.ConnectionString
	return nil
}

func (m *statefulStoragePlugin) CreateSchema(ontology *models.OntologyDefinition) error { return nil }

func (m *statefulStoragePlugin) Store(cir *models.CIR) (*models.StorageResult, error) {
	return &models.StorageResult{Success: true, AffectedItems: 1}, nil
}

func (m *statefulStoragePlugin) Retrieve(query *models.CIRQuery) ([]*models.CIR, error) {
	return []*models.CIR{
		{
			Version: models.CIRVersion,
			Source:  models.CIRSource{Type: models.SourceTypeDatabase, URI: m.connectionString, Timestamp: time.Now().UTC(), Format: models.DataFormatJSON},
			Data:    map[string]interface{}{"connection_string": m.connectionString},
		},
	}, nil
}

func (m *statefulStoragePlugin) Update(query *models.CIRQuery, updates *models.CIRUpdate) (*models.StorageResult, error) {
	return &models.StorageResult{Success: true, AffectedItems: 0}, nil
}

func (m *statefulStoragePlugin) Delete(query *models.CIRQuery) (*models.StorageResult, error) {
	return &models.StorageResult{Success: true, AffectedItems: 0}, nil
}

func (m *statefulStoragePlugin) GetMetadata() (*models.StorageMetadata, error) {
	return &models.StorageMetadata{StorageType: m.connectionString}, nil
}

func (m *statefulStoragePlugin) HealthCheck() (bool, error) { return true, nil }

func TestStorageOperationsUseIsolatedPluginInstances(t *testing.T) {
	store, err := metadatastore.NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create metadata store: %v", err)
	}
	defer store.Close()

	saveTestProject(t, store, "project-1")

	svc := NewService(store)
	svc.RegisterPlugin("stateful", &statefulStoragePlugin{})

	firstCfg, err := svc.CreateStorageConfig("project-1", "stateful", map[string]interface{}{"connection_string": "mock://first"})
	if err != nil {
		t.Fatalf("failed to create first storage config: %v", err)
	}
	secondCfg, err := svc.CreateStorageConfig("project-1", "stateful", map[string]interface{}{"connection_string": "mock://second"})
	if err != nil {
		t.Fatalf("failed to create second storage config: %v", err)
	}

	firstResults, err := svc.Retrieve(firstCfg.ID, &models.CIRQuery{Limit: 1})
	if err != nil {
		t.Fatalf("failed to retrieve from first storage config: %v", err)
	}
	secondResults, err := svc.Retrieve(secondCfg.ID, &models.CIRQuery{Limit: 1})
	if err != nil {
		t.Fatalf("failed to retrieve from second storage config: %v", err)
	}

	firstConnection := firstResults[0].Data.(map[string]interface{})["connection_string"]
	secondConnection := secondResults[0].Data.(map[string]interface{})["connection_string"]
	if firstConnection != "mock://first" {
		t.Fatalf("expected first storage config to keep its own plugin state, got %v", firstConnection)
	}
	if secondConnection != "mock://second" {
		t.Fatalf("expected second storage config to keep its own plugin state, got %v", secondConnection)
	}
}

func TestInitializeStorageDoesNotWriteFakeOntologyID(t *testing.T) {
	store, err := metadatastore.NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create metadata store: %v", err)
	}
	defer store.Close()

	saveTestProject(t, store, "project-1")

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

type mockSampleStoragePlugin struct {
	sample []*models.CIR
}

func (m *mockSampleStoragePlugin) Initialize(config *models.PluginConfig) error           { return nil }
func (m *mockSampleStoragePlugin) CreateSchema(ontology *models.OntologyDefinition) error { return nil }
func (m *mockSampleStoragePlugin) Store(cir *models.CIR) (*models.StorageResult, error) {
	return &models.StorageResult{Success: true, AffectedItems: 1}, nil
}
func (m *mockSampleStoragePlugin) Retrieve(query *models.CIRQuery) ([]*models.CIR, error) {
	if query != nil && query.Limit > 0 && len(m.sample) > query.Limit {
		return m.sample[:query.Limit], nil
	}
	return m.sample, nil
}
func (m *mockSampleStoragePlugin) Update(query *models.CIRQuery, updates *models.CIRUpdate) (*models.StorageResult, error) {
	return &models.StorageResult{Success: true, AffectedItems: 0}, nil
}
func (m *mockSampleStoragePlugin) Delete(query *models.CIRQuery) (*models.StorageResult, error) {
	return &models.StorageResult{Success: true, AffectedItems: 0}, nil
}
func (m *mockSampleStoragePlugin) GetMetadata() (*models.StorageMetadata, error) {
	return &models.StorageMetadata{StorageType: "mock"}, nil
}
func (m *mockSampleStoragePlugin) HealthCheck() (bool, error) { return true, nil }

func TestGetIngestionHealth_ComputesHealthySource(t *testing.T) {
	store, err := metadatastore.NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create metadata store: %v", err)
	}
	defer store.Close()

	now := time.Now().UTC()
	sample := []*models.CIR{
		{
			Version: models.CIRVersion,
			Source:  models.CIRSource{Type: models.SourceTypeDatabase, URI: "db://repairs", Timestamp: now.Add(-15 * time.Minute), Format: models.DataFormatJSON},
			Data:    map[string]interface{}{"repair_id": "R1", "part_id": "P1", "margin": 12.4},
		},
		{
			Version: models.CIRVersion,
			Source:  models.CIRSource{Type: models.SourceTypeDatabase, URI: "db://repairs", Timestamp: now.Add(-5 * time.Minute), Format: models.DataFormatJSON},
			Data:    map[string]interface{}{"repair_id": "R2", "part_id": "P2", "margin": 8.2},
		},
	}

	saveTestProject(t, store, "project-health")

	svc := NewService(store)
	svc.RegisterPlugin("sample", &mockSampleStoragePlugin{sample: sample})

	cfg, err := svc.CreateStorageConfig("project-health", "sample", map[string]interface{}{"connection_string": "mock://"})
	if err != nil {
		t.Fatalf("failed to create storage config: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected storage config")
	}

	report, err := svc.GetIngestionHealth("project-health")
	if err != nil {
		t.Fatalf("GetIngestionHealth failed: %v", err)
	}
	if len(report.Sources) != 1 {
		t.Fatalf("expected 1 source, got %d", len(report.Sources))
	}
	src := report.Sources[0]
	if src.Status != models.IngestionHealthHealthy {
		t.Fatalf("expected healthy source status, got %s (score=%.3f)", src.Status, src.OverallScore)
	}
	if src.FreshnessScore < 0.8 {
		t.Fatalf("expected high freshness score, got %.3f", src.FreshnessScore)
	}
	if report.Status != models.IngestionHealthHealthy {
		t.Fatalf("expected healthy project status, got %s", report.Status)
	}
}

func TestGetIngestionHealth_DetectsDriftAndLowCompleteness(t *testing.T) {
	store, err := metadatastore.NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create metadata store: %v", err)
	}
	defer store.Close()

	now := time.Now().UTC()
	sample := []*models.CIR{
		{
			Version: models.CIRVersion,
			Source:  models.CIRSource{Type: models.SourceTypeDatabase, URI: "db://events", Timestamp: now.Add(-8 * 24 * time.Hour), Format: models.DataFormatJSON},
			Data: []interface{}{
				map[string]interface{}{"event_id": "E1", "area": "north", "severity": "high"},
				map[string]interface{}{"event_id": "E2", "area": "", "severity": nil, "notes": "new field"},
			},
		},
	}

	saveTestProject(t, store, "project-drift")

	svc := NewService(store)
	svc.RegisterPlugin("sample", &mockSampleStoragePlugin{sample: sample})

	_, err = svc.CreateStorageConfig("project-drift", "sample", map[string]interface{}{"connection_string": "mock://"})
	if err != nil {
		t.Fatalf("failed to create storage config: %v", err)
	}

	report, err := svc.GetIngestionHealth("project-drift")
	if err != nil {
		t.Fatalf("GetIngestionHealth failed: %v", err)
	}
	if len(report.Sources) != 1 {
		t.Fatalf("expected 1 source, got %d", len(report.Sources))
	}
	src := report.Sources[0]
	if src.SchemaDriftScore >= 1.0 {
		t.Fatalf("expected drift score below 1 due to mixed schema, got %.3f", src.SchemaDriftScore)
	}
	if src.CompletenessScore >= 1.0 {
		t.Fatalf("expected completeness score below 1 due to missing values, got %.3f", src.CompletenessScore)
	}
	if src.Status == models.IngestionHealthHealthy {
		t.Fatalf("expected non-healthy status for degraded source, got %s", src.Status)
	}
}
