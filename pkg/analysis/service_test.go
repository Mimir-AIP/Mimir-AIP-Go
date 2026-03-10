package analysis

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/mimir-aip/mimir-aip-go/pkg/extraction"
	"github.com/mimir-aip/mimir-aip-go/pkg/metadatastore"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"github.com/mimir-aip/mimir-aip-go/pkg/storage"
)

type testStoragePlugin struct {
	sample []*models.CIR
}

func (m *testStoragePlugin) Initialize(config *models.PluginConfig) error           { return nil }
func (m *testStoragePlugin) CreateSchema(ontology *models.OntologyDefinition) error { return nil }
func (m *testStoragePlugin) Store(cir *models.CIR) (*models.StorageResult, error) {
	return &models.StorageResult{Success: true, AffectedItems: 1}, nil
}
func (m *testStoragePlugin) Retrieve(query *models.CIRQuery) ([]*models.CIR, error) {
	return m.sample, nil
}
func (m *testStoragePlugin) Update(query *models.CIRQuery, updates *models.CIRUpdate) (*models.StorageResult, error) {
	return &models.StorageResult{Success: true, AffectedItems: 0}, nil
}
func (m *testStoragePlugin) Delete(query *models.CIRQuery) (*models.StorageResult, error) {
	return &models.StorageResult{Success: true, AffectedItems: 0}, nil
}
func (m *testStoragePlugin) GetMetadata() (*models.StorageMetadata, error) {
	return &models.StorageMetadata{StorageType: "test"}, nil
}
func (m *testStoragePlugin) HealthCheck() (bool, error) { return true, nil }

func setupAnalysisService(t *testing.T) (*Service, *storage.Service, func()) {
	t.Helper()
	tmpDir := t.TempDir()
	store, err := metadatastore.NewSQLiteStore(filepath.Join(tmpDir, "analysis.db"))
	if err != nil {
		t.Fatalf("failed to create metadata store: %v", err)
	}
	storageSvc := storage.NewService(store)
	extractionSvc := extraction.NewService(storageSvc)
	return NewService(store, extractionSvc, storageSvc), storageSvc, func() { _ = store.Close() }
}

func TestAdjustedLinkPolicyUsesFeedback(t *testing.T) {
	service, _, cleanup := setupAnalysisService(t)
	defer cleanup()

	now := time.Now().UTC()
	statuses := []models.ReviewItemStatus{
		models.ReviewItemStatusRejected,
		models.ReviewItemStatusRejected,
		models.ReviewItemStatusRejected,
		models.ReviewItemStatusAccepted,
	}
	for _, status := range statuses {
		item := &models.ReviewItem{
			ID:          string(status) + now.Format(time.RFC3339Nano),
			ProjectID:   "project-feedback",
			RunID:       "run-1",
			FindingType: "cross_source_link",
			Status:      status,
			Confidence:  0.6,
			Payload:     map[string]any{"storage_a": "left"},
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		if err := service.store.SaveReviewItem(item); err != nil {
			t.Fatalf("failed to save review item: %v", err)
		}
		now = now.Add(time.Millisecond)
	}

	policy, metrics, err := service.adjustedLinkPolicy("project-feedback")
	if err != nil {
		t.Fatalf("adjustedLinkPolicy returned error: %v", err)
	}
	defaults := extraction.DefaultLinkPolicy()
	if policy.ReviewThreshold <= defaults.ReviewThreshold {
		t.Fatalf("expected review threshold to increase after repeated rejections, got %.2f <= %.2f", policy.ReviewThreshold, defaults.ReviewThreshold)
	}
	if policy.AutoAcceptThreshold <= defaults.AutoAcceptThreshold {
		t.Fatalf("expected auto-accept threshold to increase after repeated rejections, got %.2f <= %.2f", policy.AutoAcceptThreshold, defaults.AutoAcceptThreshold)
	}
	if metrics["rejected_feedback"] != 3 {
		t.Fatalf("expected rejected_feedback metric to be 3, got %#v", metrics["rejected_feedback"])
	}
}

func TestGenerateProjectInsightsCreatesPersistedSpikeInsight(t *testing.T) {
	service, storageSvc, cleanup := setupAnalysisService(t)
	defer cleanup()

	now := time.Now().UTC().Truncate(24 * time.Hour)
	sample := make([]*models.CIR, 0)
	for dayOffset := 4; dayOffset >= 1; dayOffset-- {
		ts := now.AddDate(0, 0, -dayOffset)
		sample = append(sample, &models.CIR{Version: models.CIRVersion, Source: models.CIRSource{Type: models.SourceTypeDatabase, URI: "db://events", Timestamp: ts, Format: models.DataFormatJSON}, Data: map[string]any{"event_id": dayOffset, "region": "north", "severity": "high"}})
	}
	for i := 0; i < 6; i++ {
		ts := now
		sample = append(sample, &models.CIR{Version: models.CIRVersion, Source: models.CIRSource{Type: models.SourceTypeDatabase, URI: "db://events", Timestamp: ts, Format: models.DataFormatJSON}, Data: map[string]any{"event_id": i + 10, "region": "north", "severity": "high", "signal": "spike"}})
	}

	storageSvc.RegisterPlugin("insight-sample", &testStoragePlugin{sample: sample})
	cfg, err := storageSvc.CreateStorageConfig("project-insights", "insight-sample", map[string]any{"connection_string": "mock://insights"})
	if err != nil {
		t.Fatalf("failed to create storage config: %v", err)
	}

	run, insights, err := service.GenerateProjectInsights("project-insights")
	if err != nil {
		t.Fatalf("GenerateProjectInsights returned error: %v", err)
	}
	if run == nil || run.Kind != models.AnalysisRunKindInsights {
		t.Fatalf("expected persisted insights run, got %#v", run)
	}
	if len(insights) == 0 {
		t.Fatal("expected at least one insight to be generated")
	}
	persisted, err := service.ListInsights("project-insights", "", 0)
	if err != nil {
		t.Fatalf("failed to list persisted insights: %v", err)
	}
	if len(persisted) == 0 {
		t.Fatal("expected persisted insights to be queryable")
	}
	if persisted[0].Evidence["storage_id"] != cfg.ID {
		t.Fatalf("expected insight evidence to include storage_id %s, got %#v", cfg.ID, persisted[0].Evidence)
	}
	if persisted[0].Confidence <= 0 {
		t.Fatalf("expected positive confidence, got %f", persisted[0].Confidence)
	}
}
