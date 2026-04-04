package api

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/mimir-aip/mimir-aip-go/pkg/metadatastore"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"github.com/mimir-aip/mimir-aip-go/pkg/queue"
)

func TestProjectStateSummaryMarksPendingTwinApprovalsAsAttention(t *testing.T) {
	store, err := metadatastore.NewSQLiteStore(filepath.Join(t.TempDir(), "state-summary.db"))
	if err != nil {
		t.Fatalf("failed to create metadata store: %v", err)
	}
	defer store.Close()
	q, err := queue.NewQueue()
	if err != nil {
		t.Fatalf("failed to create queue: %v", err)
	}
	defer q.Close()

	now := time.Now().UTC()
	project := &models.Project{
		ID:          "project-1",
		Name:        "demo",
		Description: "demo",
		Status:      models.ProjectStatusActive,
		Metadata:    models.ProjectMetadata{CreatedAt: now, UpdatedAt: now},
		Settings:    models.ProjectSettings{Timezone: "UTC", Environment: "development", OnboardingMode: models.ProjectOnboardingModeAdvanced},
	}
	ontology := &models.Ontology{ID: "ontology-1", ProjectID: project.ID, Name: "Factory Ontology", Content: "@prefix : <http://example.org/> .", Status: "active", Version: "1.0", CreatedAt: now, UpdatedAt: now}
	twin := &models.DigitalTwin{ID: "twin-1", ProjectID: project.ID, OntologyID: ontology.ID, Name: "Factory", Status: "active", CreatedAt: now, UpdatedAt: now}
	run := &models.TwinProcessingRun{ID: "run-1", ProjectID: project.ID, DigitalTwinID: twin.ID, Status: models.TwinProcessingRunStatusCompleted, TriggerType: models.TwinProcessingTriggerTypeManual, RequestedAt: now, CompletedAt: &now}
	alert := &models.AlertEvent{ID: "alert-1", ProjectID: project.ID, DigitalTwinID: twin.ID, ProcessingRunID: run.ID, ApprovalStatus: models.AlertApprovalStatusPending, Severity: models.InsightSeverityHigh, Category: "export", Title: "Pending approval", Message: "Awaiting review", CreatedAt: now}
	if err := store.SaveProject(project); err != nil {
		t.Fatalf("failed to save project: %v", err)
	}
	if err := store.SaveOntology(ontology); err != nil {
		t.Fatalf("failed to save ontology: %v", err)
	}
	if err := store.SaveDigitalTwin(twin); err != nil {
		t.Fatalf("failed to save twin: %v", err)
	}
	if err := store.SaveTwinProcessingRun(run); err != nil {
		t.Fatalf("failed to save processing run: %v", err)
	}
	if err := store.SaveAlertEvent(alert); err != nil {
		t.Fatalf("failed to save alert: %v", err)
	}

	provider := NewProjectStateProvider(store, q)
	summary, err := provider.Summary(project.ID)
	if err != nil {
		t.Fatalf("Summary returned error: %v", err)
	}
	section := summary.Sections["Digital Twins"]
	if section.Status != models.ProjectSectionStateAttention {
		t.Fatalf("expected Digital Twins attention status, got %s", section.Status)
	}
	if !section.Pulse {
		t.Fatal("expected Digital Twins section to pulse for pending approval")
	}
	if section.Count != 1 {
		t.Fatalf("expected pending approval count 1, got %d", section.Count)
	}
}
