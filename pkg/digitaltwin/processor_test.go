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

func seedProcessorProject(t *testing.T, store metadatastore.MetadataStore, projectID, ontologyID string) {
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

func setupProcessorTest(t *testing.T) (*Processor, metadatastore.MetadataStore, *queue.Queue, func()) {
	t.Helper()
	store, err := metadatastore.NewSQLiteStore(filepath.Join(t.TempDir(), "processor.db"))
	if err != nil {
		t.Fatalf("failed to create metadata store: %v", err)
	}
	q, err := queue.NewQueue(store)
	if err != nil {
		t.Fatalf("failed to create queue: %v", err)
	}
	service := NewService(store, nil, ontology.NewService(store), storage.NewService(store), nil, q)
	processor := NewProcessor(store, service, nil, q)
	return processor, store, q, func() {
		_ = q.Close()
		_ = store.Close()
	}
}

func TestEvaluateAlertEventsQueuesPendingManualApproval(t *testing.T) {
	processor, store, q, cleanup := setupProcessorTest(t)
	defer cleanup()

	seedProcessorProject(t, store, "project-1", "ontology-1")

	now := time.Now().UTC()
	twin := &models.DigitalTwin{ID: "twin-1", ProjectID: "project-1", OntologyID: "ontology-1", Name: "Factory", Status: "active", CreatedAt: now, UpdatedAt: now}
	pipeline := &models.Pipeline{ID: "pipe-1", ProjectID: twin.ProjectID, Name: "Export", Type: models.PipelineTypeOutput, Status: models.PipelineStatusActive, CreatedAt: now, UpdatedAt: now}
	entity := &models.Entity{ID: "entity-1", DigitalTwinID: twin.ID, Type: "Machine", Attributes: map[string]interface{}{"temperature": 91}, CreatedAt: now, UpdatedAt: now}
	action := &models.Action{
		ID:            "action-1",
		DigitalTwinID: twin.ID,
		Name:          "Overheat export",
		Enabled:       true,
		Condition:     &models.ActionCondition{Attribute: "temperature", Operator: "gt", Threshold: 80, EntityType: "Machine"},
		Trigger:       &models.ActionTrigger{PipelineID: pipeline.ID, ApprovalMode: models.ActionApprovalModeManual, Parameters: map[string]interface{}{"alert_category": "overheat"}},
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	run := &models.TwinProcessingRun{ID: "run-1", ProjectID: twin.ProjectID, DigitalTwinID: twin.ID, RequestedAt: now}

	for _, resource := range []any{twin, pipeline, entity, action} {
		switch v := resource.(type) {
		case *models.DigitalTwin:
			if err := store.SaveDigitalTwin(v); err != nil {
				t.Fatalf("failed to save twin: %v", err)
			}
		case *models.Pipeline:
			if err := store.SavePipeline(v); err != nil {
				t.Fatalf("failed to save pipeline: %v", err)
			}
		case *models.Entity:
			if err := store.SaveEntity(v); err != nil {
				t.Fatalf("failed to save entity: %v", err)
			}
		case *models.Action:
			if err := store.SaveAction(v); err != nil {
				t.Fatalf("failed to save action: %v", err)
			}
		}
	}
	if err := store.SaveTwinProcessingRun(run); err != nil {
		t.Fatalf("failed to save processing run: %v", err)
	}

	result, err := processor.evaluateAlertEvents(run)
	if err != nil {
		t.Fatalf("evaluateAlertEvents returned error: %v", err)
	}
	if result.AlertCount != 1 {
		t.Fatalf("expected 1 alert, got %d", result.AlertCount)
	}
	if result.PendingApprovalCount != 1 {
		t.Fatalf("expected 1 pending approval, got %d", result.PendingApprovalCount)
	}
	if result.TriggeredActionCount != 0 {
		t.Fatalf("expected 0 triggered actions, got %d", result.TriggeredActionCount)
	}

	alerts, err := processor.ListAlerts(twin.ID, 10)
	if err != nil {
		t.Fatalf("ListAlerts returned error: %v", err)
	}
	if len(alerts) != 1 {
		t.Fatalf("expected 1 persisted alert, got %d", len(alerts))
	}
	alert := alerts[0]
	if alert.ApprovalStatus != models.AlertApprovalStatusPending {
		t.Fatalf("expected pending approval status, got %s", alert.ApprovalStatus)
	}
	if alert.RequestedExportPipelineID != pipeline.ID {
		t.Fatalf("expected requested pipeline %s, got %s", pipeline.ID, alert.RequestedExportPipelineID)
	}
	if alert.TriggeredWorkTaskID != "" {
		t.Fatalf("expected no triggered work task before approval, got %s", alert.TriggeredWorkTaskID)
	}

	tasks, err := q.ListWorkTasks()
	if err != nil {
		t.Fatalf("failed to list queued tasks: %v", err)
	}
	if len(tasks) != 0 {
		t.Fatalf("expected no queued export task before approval, got %d", len(tasks))
	}
}

func TestReviewAlertApproveQueuesExportAndUpdatesAction(t *testing.T) {
	processor, store, q, cleanup := setupProcessorTest(t)
	defer cleanup()

	seedProcessorProject(t, store, "project-1", "ontology-1")

	now := time.Now().UTC()
	twin := &models.DigitalTwin{ID: "twin-1", ProjectID: "project-1", OntologyID: "ontology-1", Name: "Factory", Status: "active", CreatedAt: now, UpdatedAt: now}
	pipeline := &models.Pipeline{ID: "pipe-1", ProjectID: twin.ProjectID, Name: "Export", Type: models.PipelineTypeOutput, Status: models.PipelineStatusActive, CreatedAt: now, UpdatedAt: now}
	action := &models.Action{ID: "action-1", DigitalTwinID: twin.ID, Name: "Overheat export", Enabled: true, Trigger: &models.ActionTrigger{PipelineID: pipeline.ID, ApprovalMode: models.ActionApprovalModeManual}, CreatedAt: now, UpdatedAt: now}
	run := &models.TwinProcessingRun{ID: "run-1", ProjectID: twin.ProjectID, DigitalTwinID: twin.ID, RequestedAt: now}
	alert := &models.AlertEvent{
		ID:                        "alert-1",
		ProjectID:                 twin.ProjectID,
		DigitalTwinID:             twin.ID,
		ProcessingRunID:           run.ID,
		ActionID:                  action.ID,
		ApprovalStatus:            models.AlertApprovalStatusPending,
		RequestedExportPipelineID: pipeline.ID,
		RequestedTriggerParams:    map[string]any{"alert_category": "overheat"},
		CreatedAt:                 now,
	}

	if err := store.SaveDigitalTwin(twin); err != nil {
		t.Fatalf("failed to save twin: %v", err)
	}
	if err := store.SavePipeline(pipeline); err != nil {
		t.Fatalf("failed to save pipeline: %v", err)
	}
	if err := store.SaveTwinProcessingRun(run); err != nil {
		t.Fatalf("failed to save processing run: %v", err)
	}
	if err := store.SaveAction(action); err != nil {
		t.Fatalf("failed to save action: %v", err)
	}
	if err := store.SaveAlertEvent(alert); err != nil {
		t.Fatalf("failed to save alert: %v", err)
	}

	updated, err := processor.ReviewAlert(twin.ID, alert.ID, &models.AlertApprovalRequest{Decision: models.AlertApprovalDecisionApprove, Actor: "tester", Note: "approved"})
	if err != nil {
		t.Fatalf("ReviewAlert returned error: %v", err)
	}
	if updated.ApprovalStatus != models.AlertApprovalStatusApproved {
		t.Fatalf("expected approved alert, got %s", updated.ApprovalStatus)
	}
	if updated.TriggeredWorkTaskID == "" {
		t.Fatal("expected triggered work task id to be recorded")
	}

	queuedTask, err := q.GetWorkTask(updated.TriggeredWorkTaskID)
	if err != nil {
		t.Fatalf("expected queued task to exist: %v", err)
	}
	if queuedTask.TaskSpec.PipelineID != pipeline.ID {
		t.Fatalf("expected pipeline %s, got %s", pipeline.ID, queuedTask.TaskSpec.PipelineID)
	}
	if queuedTask.TaskSpec.Parameters["alert_id"] != alert.ID {
		t.Fatalf("expected alert_id parameter to be preserved, got %#v", queuedTask.TaskSpec.Parameters["alert_id"])
	}

	reloadedAction, err := store.GetAction(action.ID)
	if err != nil {
		t.Fatalf("failed to reload action: %v", err)
	}
	if reloadedAction.TriggerCount != 1 {
		t.Fatalf("expected action trigger count 1, got %d", reloadedAction.TriggerCount)
	}
	if reloadedAction.LastTriggered == nil {
		t.Fatal("expected action last_triggered to be set")
	}
}
