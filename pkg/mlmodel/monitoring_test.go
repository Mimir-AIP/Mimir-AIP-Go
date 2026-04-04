package mlmodel

import (
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/mimir-aip/mimir-aip-go/pkg/metadatastore"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"github.com/mimir-aip/mimir-aip-go/pkg/ontology"
	"github.com/mimir-aip/mimir-aip-go/pkg/queue"
	"github.com/mimir-aip/mimir-aip-go/pkg/storage"
)

type monitoringSamplePlugin struct {
	sample []*models.CIR
}

func (m *monitoringSamplePlugin) Initialize(config *models.PluginConfig) error { return nil }
func (m *monitoringSamplePlugin) CreateSchema(ontology *models.OntologyDefinition) error {
	return nil
}
func (m *monitoringSamplePlugin) Store(cir *models.CIR) (*models.StorageResult, error) {
	return &models.StorageResult{Success: true, AffectedItems: 1}, nil
}
func (m *monitoringSamplePlugin) Retrieve(query *models.CIRQuery) ([]*models.CIR, error) {
	return m.sample, nil
}
func (m *monitoringSamplePlugin) Update(query *models.CIRQuery, updates *models.CIRUpdate) (*models.StorageResult, error) {
	return &models.StorageResult{Success: true, AffectedItems: 0}, nil
}
func (m *monitoringSamplePlugin) Delete(query *models.CIRQuery) (*models.StorageResult, error) {
	return &models.StorageResult{Success: true, AffectedItems: 0}, nil
}
func (m *monitoringSamplePlugin) GetMetadata() (*models.StorageMetadata, error) {
	return &models.StorageMetadata{StorageType: "monitoring-sample"}, nil
}
func (m *monitoringSamplePlugin) HealthCheck() (bool, error) { return true, nil }

func setupMonitoringService(t *testing.T) (*MonitoringService, *Service, *storage.Service, func()) {
	t.Helper()

	store, err := metadatastore.NewSQLiteStore(filepath.Join(t.TempDir(), "monitoring.db"))
	if err != nil {
		t.Fatalf("failed to create metadata store: %v", err)
	}
	now := time.Now().UTC()
	project := &models.Project{
		ID:          "project-1",
		Name:        "project-1",
		Description: "test project",
		Version:     "v1",
		Status:      models.ProjectStatusActive,
		Metadata:    models.ProjectMetadata{CreatedAt: now, UpdatedAt: now},
	}
	if err := store.SaveProject(project); err != nil {
		t.Fatalf("failed to save project: %v", err)
	}
	ont := &models.Ontology{
		ID:          "ontology-1",
		ProjectID:   project.ID,
		Name:        "ontology-1",
		Description: "test ontology",
		Version:     "1.0",
		Content:     "@prefix : <http://example.org/> .",
		Status:      "active",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := store.SaveOntology(ont); err != nil {
		t.Fatalf("failed to save ontology: %v", err)
	}
	q, err := queue.NewQueue(store)
	if err != nil {
		t.Fatalf("failed to create queue: %v", err)
	}
	storageSvc := storage.NewService(store)
	mlSvc := NewService(store, ontology.NewService(store), storageSvc, q)
	monitoringSvc := NewMonitoringService(store, mlSvc, storageSvc)
	return monitoringSvc, mlSvc, storageSvc, func() {
		_ = q.Close()
		_ = store.Close()
	}
}

func TestExtractFeaturesAndLabelsUsesArtifactOrdering(t *testing.T) {
	cirs := []*models.CIR{{
		Version: models.CIRVersion,
		Source:  models.CIRSource{Type: models.SourceTypeDatabase, URI: "db://monitor", Timestamp: time.Now().UTC(), Format: models.DataFormatJSON},
		Data: map[string]interface{}{
			"feature_b": 2.0,
			"label":     1.0,
			"feature_a": 1.0,
		},
	}}

	features, labels := extractFeaturesAndLabels(cirs, []string{"feature_a", "feature_b"}, "label")
	if len(features) != 1 || len(labels) != 1 {
		t.Fatalf("expected one extracted row, got features=%d labels=%d", len(features), len(labels))
	}
	if features[0][0] != 1.0 || features[0][1] != 2.0 {
		t.Fatalf("expected artifact feature ordering [1 2], got %#v", features[0])
	}
	if labels[0] != 1.0 {
		t.Fatalf("expected label 1.0, got %v", labels[0])
	}
}

func TestCheckModelUsesArtifactFeatureSchema(t *testing.T) {
	monitoringSvc, mlSvc, storageSvc, cleanup := setupMonitoringService(t)
	defer cleanup()

	storageSvc.RegisterPlugin("monitoring-sample", &monitoringSamplePlugin{sample: []*models.CIR{
		{
			Version: models.CIRVersion,
			Source:  models.CIRSource{Type: models.SourceTypeDatabase, URI: "db://monitor", Timestamp: time.Now().UTC(), Format: models.DataFormatJSON},
			Data: map[string]interface{}{
				"label":     1.0,
				"feature_b": 1.0,
				"feature_a": 1.0,
			},
		},
	}})
	if _, err := storageSvc.CreateStorageConfig("project-1", "monitoring-sample", map[string]interface{}{"connection_string": "mock://monitor"}); err != nil {
		t.Fatalf("failed to create storage config: %v", err)
	}

	t.Setenv("MODEL_ARTIFACT_DIR", t.TempDir())
	artifactBytes, err := json.Marshal(map[string]any{
		"model_type":    "decision_tree",
		"feature_names": []string{"feature_a", "feature_b"},
		"parameters": map[string]any{
			"model_data": map[string]any{"is_leaf": true, "value": 1.0},
		},
		"metadata": map[string]any{"label_column": "label"},
	})
	if err != nil {
		t.Fatalf("failed to marshal artifact: %v", err)
	}

	model, err := mlSvc.CreateModel(&models.ModelCreateRequest{ProjectID: "project-1", OntologyID: "ontology-1", Name: "tree", Type: models.ModelTypeDecisionTree})
	if err != nil {
		t.Fatalf("failed to create model: %v", err)
	}
	if err := mlSvc.CompleteTraining(model.ID, artifactBytes, &models.PerformanceMetrics{Accuracy: 1.0}); err != nil {
		t.Fatalf("failed to complete training: %v", err)
	}

	persisted, err := mlSvc.GetModel(model.ID)
	if err != nil {
		t.Fatalf("failed to reload model: %v", err)
	}
	if err := monitoringSvc.CheckModel(persisted); err != nil {
		t.Fatalf("CheckModel returned error: %v", err)
	}
}
