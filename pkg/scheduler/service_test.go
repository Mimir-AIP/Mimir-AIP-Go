package scheduler

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/mimir-aip/mimir-aip-go/pkg/metadatastore"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"github.com/mimir-aip/mimir-aip-go/pkg/pipeline"
	"github.com/mimir-aip/mimir-aip-go/pkg/queue"
)

func setupSchedulerService(t *testing.T) (*Service, func()) {
	t.Helper()

	store, err := metadatastore.NewSQLiteStore(filepath.Join(t.TempDir(), "scheduler.db"))
	if err != nil {
		t.Fatalf("failed to create metadata store: %v", err)
	}

	project := &models.Project{
		ID:          "project-1",
		Name:        "project-1",
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

	pipelineRecord := &models.Pipeline{
		ID:          "pipeline-1",
		ProjectID:   project.ID,
		Name:        "pipeline-1",
		Type:        models.PipelineTypeIngestion,
		Description: "test pipeline",
		Status:      models.PipelineStatusActive,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	if err := store.SavePipeline(pipelineRecord); err != nil {
		t.Fatalf("failed to save pipeline: %v", err)
	}

	pipelineSvc := pipeline.NewService(store)
	q, err := queue.NewQueue(store)
	if err != nil {
		t.Fatalf("failed to create queue: %v", err)
	}

	return NewService(store, pipelineSvc, q), func() {
		_ = q.Close()
		_ = store.Close()
	}
}

func TestUpdateInvalidCronPreservesExistingSchedule(t *testing.T) {
	svc, cleanup := setupSchedulerService(t)
	defer cleanup()

	job, err := svc.Create(&models.ScheduleCreateRequest{
		ProjectID:    "project-1",
		Name:         "hourly",
		Pipelines:    []string{"pipeline-1"},
		CronSchedule: "0 * * * *",
		Enabled:      true,
	})
	if err != nil {
		t.Fatalf("failed to create schedule: %v", err)
	}

	oldEntryID, ok := svc.jobEntry(job.ID)
	if !ok {
		t.Fatal("expected created schedule to be registered in cron map")
	}

	invalidCron := "not-a-cron"
	if _, err := svc.Update(job.ID, &models.ScheduleUpdateRequest{CronSchedule: &invalidCron}); err == nil {
		t.Fatal("expected invalid cron update to fail")
	}

	entryID, ok := svc.jobEntry(job.ID)
	if !ok {
		t.Fatal("expected original cron entry to remain registered after failed update")
	}
	if entryID != oldEntryID {
		t.Fatalf("expected cron entry %v to remain, got %v", oldEntryID, entryID)
	}

	persisted, err := svc.Get(job.ID)
	if err != nil {
		t.Fatalf("failed to reload schedule: %v", err)
	}
	if persisted.CronSchedule != "0 * * * *" {
		t.Fatalf("expected cron schedule to remain unchanged, got %s", persisted.CronSchedule)
	}
}

func TestDisableRemovesCronEntryAfterSave(t *testing.T) {
	svc, cleanup := setupSchedulerService(t)
	defer cleanup()

	job, err := svc.Create(&models.ScheduleCreateRequest{
		ProjectID:    "project-1",
		Name:         "hourly",
		Pipelines:    []string{"pipeline-1"},
		CronSchedule: "0 * * * *",
		Enabled:      true,
	})
	if err != nil {
		t.Fatalf("failed to create schedule: %v", err)
	}

	disabled := false
	updated, err := svc.Update(job.ID, &models.ScheduleUpdateRequest{Enabled: &disabled})
	if err != nil {
		t.Fatalf("failed to disable schedule: %v", err)
	}
	if updated.Enabled {
		t.Fatal("expected schedule to be disabled")
	}
	if updated.NextRun != nil {
		t.Fatalf("expected disabled schedule to clear next_run, got %v", updated.NextRun)
	}
	if _, ok := svc.jobEntry(job.ID); ok {
		t.Fatal("expected disabled schedule to be removed from cron map")
	}

	persisted, err := svc.Get(job.ID)
	if err != nil {
		t.Fatalf("failed to reload disabled schedule: %v", err)
	}
	if persisted.Enabled {
		t.Fatal("expected persisted schedule to be disabled")
	}
	if persisted.NextRun != nil {
		t.Fatalf("expected persisted next_run to be nil, got %v", persisted.NextRun)
	}
}

func TestExecuteJobQueuesScheduledWorkTasks(t *testing.T) {
	svc, cleanup := setupSchedulerService(t)
	defer cleanup()

	now := time.Now().UTC()
	job := &models.Schedule{
		ID:           "schedule-1",
		ProjectID:    "project-1",
		Name:         "hourly",
		Pipelines:    []string{"pipeline-1"},
		CronSchedule: "0 * * * *",
		Enabled:      true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := svc.store.SaveSchedule(job); err != nil {
		t.Fatalf("failed to save schedule: %v", err)
	}

	svc.executeJob(job)

	tasks, err := svc.queue.ListWorkTasks()
	if err != nil {
		t.Fatalf("failed to list work tasks: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected 1 queued work task, got %d", len(tasks))
	}
	task := tasks[0]
	if task.Status != models.WorkTaskStatusScheduled {
		t.Fatalf("expected scheduled task status, got %s", task.Status)
	}
	if task.TaskSpec.Parameters["trigger_type"] != "scheduled" {
		t.Fatalf("expected trigger_type scheduled, got %#v", task.TaskSpec.Parameters["trigger_type"])
	}
	if task.TaskSpec.Parameters["triggered_by"] != job.ID {
		t.Fatalf("expected triggered_by %s, got %#v", job.ID, task.TaskSpec.Parameters["triggered_by"])
	}
}
