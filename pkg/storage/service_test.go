package storage

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/mimir-aip/mimir-aip-go/pkg/metadatastore"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	ontologysvc "github.com/mimir-aip/mimir-aip-go/pkg/ontology"
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

type cirCapturingStoragePlugin struct {
	captured **models.CIR
}

func (m *cirCapturingStoragePlugin) Initialize(config *models.PluginConfig) error { return nil }
func (m *cirCapturingStoragePlugin) CreateSchema(ontology *models.OntologyDefinition) error {
	return nil
}
func (m *cirCapturingStoragePlugin) Store(cir *models.CIR) (*models.StorageResult, error) {
	if m.captured != nil {
		*m.captured = cir
	}
	return &models.StorageResult{Success: true, AffectedItems: 1}, nil
}
func (m *cirCapturingStoragePlugin) Retrieve(query *models.CIRQuery) ([]*models.CIR, error) {
	return []*models.CIR{}, nil
}
func (m *cirCapturingStoragePlugin) Update(query *models.CIRQuery, updates *models.CIRUpdate) (*models.StorageResult, error) {
	return &models.StorageResult{Success: true, AffectedItems: 0}, nil
}
func (m *cirCapturingStoragePlugin) Delete(query *models.CIRQuery) (*models.StorageResult, error) {
	return &models.StorageResult{Success: true, AffectedItems: 0}, nil
}
func (m *cirCapturingStoragePlugin) GetMetadata() (*models.StorageMetadata, error) {
	return &models.StorageMetadata{StorageType: "cir-capture"}, nil
}
func (m *cirCapturingStoragePlugin) HealthCheck() (bool, error) { return true, nil }

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

type schemaCapturingStoragePlugin struct {
	captured **models.OntologyDefinition
}

func (m *schemaCapturingStoragePlugin) Initialize(config *models.PluginConfig) error { return nil }
func (m *schemaCapturingStoragePlugin) CreateSchema(ontology *models.OntologyDefinition) error {
	if m.captured != nil {
		*m.captured = ontology
	}
	return nil
}
func (m *schemaCapturingStoragePlugin) Store(cir *models.CIR) (*models.StorageResult, error) {
	return &models.StorageResult{Success: true, AffectedItems: 1}, nil
}
func (m *schemaCapturingStoragePlugin) Retrieve(query *models.CIRQuery) ([]*models.CIR, error) {
	return []*models.CIR{}, nil
}
func (m *schemaCapturingStoragePlugin) Update(query *models.CIRQuery, updates *models.CIRUpdate) (*models.StorageResult, error) {
	return &models.StorageResult{Success: true, AffectedItems: 0}, nil
}
func (m *schemaCapturingStoragePlugin) Delete(query *models.CIRQuery) (*models.StorageResult, error) {
	return &models.StorageResult{Success: true, AffectedItems: 0}, nil
}
func (m *schemaCapturingStoragePlugin) GetMetadata() (*models.StorageMetadata, error) {
	return &models.StorageMetadata{StorageType: "schema-capture"}, nil
}
func (m *schemaCapturingStoragePlugin) HealthCheck() (bool, error) { return true, nil }

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

func TestStoreForProjectAddsOntologyMapping(t *testing.T) {
	store, err := metadatastore.NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create metadata store: %v", err)
	}
	defer store.Close()
	saveTestProject(t, store, "project-1")

	ontologyRecord, err := ontologysvc.NewService(store).CreateOntology(&models.OntologyCreateRequest{
		ProjectID: "project-1",
		Name:      "Readings",
		Content:   semanticMappingOntology,
		Status:    "active",
	})
	if err != nil {
		t.Fatalf("failed to create ontology: %v", err)
	}

	var captured *models.CIR
	svc := NewService(store)
	svc.RegisterPlugin("cir-capture", &cirCapturingStoragePlugin{captured: &captured})
	cfg, err := svc.CreateStorageConfigWithOntology("project-1", "cir-capture", map[string]interface{}{"connection_string": "mock://capture"}, ontologyRecord.ID)
	if err != nil {
		t.Fatalf("CreateStorageConfigWithOntology failed: %v", err)
	}

	cir := models.NewCIR(models.SourceTypeAPI, "api://sensor-readings", models.DataFormatJSON, map[string]interface{}{"entity_type": "SensorReading", "temperature": 22.4})
	if _, err := svc.StoreForProject("project-1", cfg.ID, cir); err != nil {
		t.Fatalf("StoreForProject failed: %v", err)
	}
	if captured == nil || captured.Metadata.Ontology == nil {
		t.Fatalf("expected stored CIR to include ontology mapping, got %+v", captured)
	}
	if captured.Metadata.Ontology.ClassID != "SensorReading" {
		t.Fatalf("expected SensorReading mapping, got %+v", captured.Metadata.Ontology)
	}
}

func TestCreateStorageConfigWithOntologyInitializesSchema(t *testing.T) {
	store, err := metadatastore.NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create metadata store: %v", err)
	}
	defer store.Close()
	saveTestProject(t, store, "project-1")

	ontologyService := ontologysvc.NewService(store)
	ontologyRecord, err := ontologyService.CreateOntology(&models.OntologyCreateRequest{
		ProjectID: "project-1",
		Name:      "Plant",
		Content: `@prefix : <http://example.org/mimir#> .
		@prefix owl: <http://www.w3.org/2002/07/owl#> .
		@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .
		@prefix xsd: <http://www.w3.org/2001/XMLSchema#> .

		:Sensor a owl:Class .
		:Reading a owl:Class .
		:temperature a owl:DatatypeProperty ;
		  rdfs:domain :Reading ;
		  rdfs:range xsd:decimal .
		:hasReading a owl:ObjectProperty ;
		  rdfs:domain :Sensor ;
		  rdfs:range :Reading .`,
		Status: "active",
	})
	if err != nil {
		t.Fatalf("failed to create ontology: %v", err)
	}

	var captured *models.OntologyDefinition
	plugin := &schemaCapturingStoragePlugin{captured: &captured}
	svc := NewService(store)
	svc.RegisterPlugin("schema-capture", plugin)
	cfg, err := svc.CreateStorageConfigWithOntology("project-1", "schema-capture", map[string]interface{}{"connection_string": "mock://schema"}, ontologyRecord.ID)
	if err != nil {
		t.Fatalf("CreateStorageConfigWithOntology failed: %v", err)
	}
	if cfg.OntologyID != ontologyRecord.ID {
		t.Fatalf("expected storage config ontology_id %s, got %s", ontologyRecord.ID, cfg.OntologyID)
	}
	if captured == nil {
		t.Fatalf("expected plugin schema initialization")
	}
	if len(captured.Entities) != 2 {
		t.Fatalf("expected 2 entities, got %+v", captured.Entities)
	}
	if len(captured.Relationships) != 1 || captured.Relationships[0].Name != "hasReading" {
		t.Fatalf("expected hasReading relationship, got %+v", captured.Relationships)
	}
}

func TestCreateStorageConfigWithOntologyRejectsProjectMismatch(t *testing.T) {
	store, err := metadatastore.NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create metadata store: %v", err)
	}
	defer store.Close()
	saveTestProject(t, store, "project-1")
	saveTestProject(t, store, "project-2")

	ontologyRecord, err := ontologysvc.NewService(store).CreateOntology(&models.OntologyCreateRequest{
		ProjectID: "project-2",
		Name:      "Other",
		Content:   "@prefix : <http://example.org/mimir#> .\n@prefix owl: <http://www.w3.org/2002/07/owl#> .\n\n:Entity a owl:Class .",
		Status:    "active",
	})
	if err != nil {
		t.Fatalf("failed to create ontology: %v", err)
	}

	svc := NewService(store)
	svc.RegisterPlugin("schema-capture", &schemaCapturingStoragePlugin{})
	_, err = svc.CreateStorageConfigWithOntology("project-1", "schema-capture", map[string]interface{}{"connection_string": "mock://schema"}, ontologyRecord.ID)
	if err == nil || !strings.Contains(err.Error(), "belongs to project") {
		t.Fatalf("expected project mismatch error, got %v", err)
	}
}

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

func TestDeleteStorageConfigRejectsReferencedResources(t *testing.T) {
	store, err := metadatastore.NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create metadata store: %v", err)
	}
	defer store.Close()

	saveTestProject(t, store, "project-delete-refs")

	svc := NewService(store)
	svc.RegisterPlugin("mock", &mockStoragePlugin{})

	cfg, err := svc.CreateStorageConfig("project-delete-refs", "mock", map[string]interface{}{"connection_string": "mock://refs"})
	if err != nil {
		t.Fatalf("failed to create storage config: %v", err)
	}

	pipeline := &models.Pipeline{
		ID:        "pipeline-delete-refs",
		ProjectID: "project-delete-refs",
		Name:      "ingest",
		Type:      models.PipelineTypeIngestion,
		Steps: []models.PipelineStep{{
			Name:       "store",
			Plugin:     "builtin",
			Action:     "store_cir",
			Parameters: map[string]interface{}{"storage_id": cfg.ID},
		}},
		Status:    models.PipelineStatusActive,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	if err := store.SavePipeline(pipeline); err != nil {
		t.Fatalf("failed to save pipeline: %v", err)
	}

	twin := &models.DigitalTwin{
		ID:         "twin-delete-refs",
		ProjectID:  "project-delete-refs",
		OntologyID: "ontology-delete-refs",
		Name:       "Twin",
		Status:     "active",
		Config:     &models.DigitalTwinConfig{StorageIDs: []string{cfg.ID}},
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}
	if err := store.SaveOntology(&models.Ontology{
		ID:        "ontology-delete-refs",
		ProjectID: "project-delete-refs",
		Name:      "Ontology",
		Content:   "@prefix ex: <http://example.com/> .",
		Status:    "active",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("failed to save ontology: %v", err)
	}
	if err := store.SaveDigitalTwin(twin); err != nil {
		t.Fatalf("failed to save digital twin: %v", err)
	}

	err = svc.DeleteStorageConfig(cfg.ID)
	var inUseErr *StorageConfigInUseError
	if !errors.As(err, &inUseErr) {
		t.Fatalf("expected StorageConfigInUseError, got %v", err)
	}
	if len(inUseErr.References) != 2 {
		t.Fatalf("expected 2 references, got %#v", inUseErr.References)
	}
	if !strings.Contains(inUseErr.Error(), pipeline.ID) || !strings.Contains(inUseErr.Error(), twin.ID) {
		t.Fatalf("expected error to mention referencing resources, got %v", inUseErr)
	}

	if _, err := store.GetStorageConfig(cfg.ID); err != nil {
		t.Fatalf("expected referenced storage config to remain persisted, got %v", err)
	}
}

func TestDeleteStorageConfigDeletesUnreferencedConfig(t *testing.T) {
	store, err := metadatastore.NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create metadata store: %v", err)
	}
	defer store.Close()

	saveTestProject(t, store, "project-delete-ok")

	svc := NewService(store)
	svc.RegisterPlugin("mock", &mockStoragePlugin{})

	cfg, err := svc.CreateStorageConfig("project-delete-ok", "mock", map[string]interface{}{"connection_string": "mock://ok"})
	if err != nil {
		t.Fatalf("failed to create storage config: %v", err)
	}

	if err := svc.DeleteStorageConfig(cfg.ID); err != nil {
		t.Fatalf("expected storage config deletion to succeed, got %v", err)
	}

	if _, err := store.GetStorageConfig(cfg.ID); err == nil {
		t.Fatal("expected storage config to be deleted")
	}
}

func TestCreateStorageConfigRejectsMissingProject(t *testing.T) {
	store, err := metadatastore.NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create metadata store: %v", err)
	}
	defer store.Close()

	svc := NewService(store)
	svc.RegisterPlugin("mock", &mockStoragePlugin{})

	_, err = svc.CreateStorageConfig("missing-project", "mock", map[string]interface{}{"connection_string": "mock://"})
	if err == nil {
		t.Fatal("expected create to fail when project does not exist")
	}
}

func TestStoreForProjectRejectsProjectMismatch(t *testing.T) {
	store, err := metadatastore.NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create metadata store: %v", err)
	}
	defer store.Close()

	saveTestProject(t, store, "project-a")
	saveTestProject(t, store, "project-b")

	svc := NewService(store)
	svc.RegisterPlugin("mock", &mockStoragePlugin{})

	cfg, err := svc.CreateStorageConfig("project-a", "mock", map[string]interface{}{"connection_string": "mock://a"})
	if err != nil {
		t.Fatalf("failed to create storage config: %v", err)
	}

	_, err = svc.StoreForProject("project-b", cfg.ID, &models.CIR{Version: models.CIRVersion, Source: models.CIRSource{Type: models.SourceTypeAPI, URI: "manual", Timestamp: time.Now().UTC(), Format: models.DataFormatJSON}, Data: map[string]interface{}{"k": "v"}})
	var mismatchErr *StorageConfigProjectMismatchError
	if !errors.As(err, &mismatchErr) {
		t.Fatalf("expected StorageConfigProjectMismatchError, got %v", err)
	}
}
