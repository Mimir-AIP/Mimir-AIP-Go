package queue

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/mimir-aip/mimir-aip-go/pkg/metadatastore"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

// TestEnqueueDequeue tests basic queue operations
func TestEnqueueDequeue(t *testing.T) {
	q, err := NewQueue(nil)
	if err != nil {
		t.Fatalf("Failed to create queue: %v", err)
	}

	// Create a test work task
	task := &models.WorkTask{
		ID:          "test-task-1",
		Type:        models.WorkTaskTypePipelineExecution,
		Status:      models.WorkTaskStatusQueued,
		Priority:    1,
		SubmittedAt: time.Now(),
		ProjectID:   "test-project",
		TaskSpec: models.TaskSpec{
			PipelineID: "test-pipeline",
			Parameters: map[string]interface{}{},
		},
	}

	// Enqueue the task
	if err := q.Enqueue(task); err != nil {
		t.Fatalf("Failed to enqueue task: %v", err)
	}

	// Dequeue the task
	dequeuedTask, err := q.Dequeue()
	if err != nil {
		t.Fatalf("Failed to dequeue task: %v", err)
	}

	if dequeuedTask == nil {
		t.Fatal("Dequeued task is nil")
	}

	if dequeuedTask.ID != task.ID {
		t.Errorf("Expected task ID %s, got %s", task.ID, dequeuedTask.ID)
	}
}

// TestQueueLength tests queue length tracking
func TestQueueLength(t *testing.T) {
	q, err := NewQueue(nil)
	if err != nil {
		t.Fatalf("Failed to create queue: %v", err)
	}

	initialLength, err := q.QueueLength()
	if err != nil {
		t.Fatalf("Failed to get queue length: %v", err)
	}

	// Create and enqueue a task
	task := &models.WorkTask{
		ID:          "test-task-2",
		Type:        models.WorkTaskTypePipelineExecution,
		Status:      models.WorkTaskStatusQueued,
		Priority:    1,
		SubmittedAt: time.Now(),
		ProjectID:   "test-project",
	}

	if err := q.Enqueue(task); err != nil {
		t.Fatalf("Failed to enqueue task: %v", err)
	}

	newLength, err := q.QueueLength()
	if err != nil {
		t.Fatalf("Failed to get queue length: %v", err)
	}

	if newLength != initialLength+1 {
		t.Errorf("Expected queue length %d, got %d", initialLength+1, newLength)
	}
}

// TestWorkTaskStatusUpdate tests work task status updates
func TestWorkTaskStatusUpdate(t *testing.T) {
	q, err := NewQueue(nil)
	if err != nil {
		t.Fatalf("Failed to create queue: %v", err)
	}

	// Create and enqueue a task
	task := &models.WorkTask{
		ID:          "test-task-3",
		Type:        models.WorkTaskTypePipelineExecution,
		Status:      models.WorkTaskStatusQueued,
		Priority:    1,
		SubmittedAt: time.Now(),
		ProjectID:   "test-project",
	}

	if err := q.Enqueue(task); err != nil {
		t.Fatalf("Failed to enqueue task: %v", err)
	}

	// Update task status
	if err := q.UpdateWorkTaskStatus(task.ID, models.WorkTaskStatusExecuting, ""); err != nil {
		t.Fatalf("Failed to update task status: %v", err)
	}

	// Get the updated task
	updatedTask, err := q.GetWorkTask(task.ID)
	if err != nil {
		t.Fatalf("Failed to get task: %v", err)
	}

	if updatedTask.Status != models.WorkTaskStatusExecuting {
		t.Errorf("Expected status %s, got %s", models.WorkTaskStatusExecuting, updatedTask.Status)
	}
}

type recordingListener struct {
	seen []*models.WorkTask
}

func (l *recordingListener) OnWorkTaskStatusChanged(task *models.WorkTask) {
	copy := *task
	l.seen = append(l.seen, &copy)
}

func TestApplyWorkTaskResultStoresWorkerOutput(t *testing.T) {
	q, err := NewQueue(nil)
	if err != nil {
		t.Fatalf("Failed to create queue: %v", err)
	}
	listener := &recordingListener{}
	q.RegisterListener(listener)

	task := &models.WorkTask{
		ID:          "test-task-4",
		Type:        models.WorkTaskTypePipelineExecution,
		Status:      models.WorkTaskStatusExecuting,
		Priority:    1,
		SubmittedAt: time.Now(),
		ProjectID:   "test-project",
		TaskSpec: models.TaskSpec{
			PipelineID: "pipeline-1",
			Parameters: map[string]interface{}{"pipeline_type": string(models.PipelineTypeIngestion)},
		},
	}
	if err := q.Enqueue(task); err != nil {
		t.Fatalf("Failed to enqueue task: %v", err)
	}

	result := &models.WorkTaskResult{
		WorkTaskID:     task.ID,
		Status:         models.WorkTaskStatusCompleted,
		OutputLocation: "/tmp/output.json",
		Metadata:       map[string]interface{}{"pipeline_type": string(models.PipelineTypeIngestion), "rows": 42},
	}
	if err := q.ApplyWorkTaskResult(task.ID, result); err != nil {
		t.Fatalf("Failed to apply work task result: %v", err)
	}

	updatedTask, err := q.GetWorkTask(task.ID)
	if err != nil {
		t.Fatalf("Failed to get task: %v", err)
	}
	if updatedTask.Status != models.WorkTaskStatusCompleted {
		t.Fatalf("Expected completed status, got %s", updatedTask.Status)
	}
	if updatedTask.OutputLocation != result.OutputLocation {
		t.Fatalf("Expected output location %s, got %s", result.OutputLocation, updatedTask.OutputLocation)
	}
	if updatedTask.ResultMetadata["rows"] != 42 {
		t.Fatalf("Expected result metadata rows=42, got %#v", updatedTask.ResultMetadata)
	}
	if len(listener.seen) != 1 {
		t.Fatalf("Expected one listener notification, got %d", len(listener.seen))
	}
	if listener.seen[0].OutputLocation != result.OutputLocation {
		t.Fatalf("Expected listener snapshot output location %s, got %s", result.OutputLocation, listener.seen[0].OutputLocation)
	}
}

func TestSnapshotAggregatesTaskCounts(t *testing.T) {
	q, err := NewQueue(nil)
	if err != nil {
		t.Fatalf("Failed to create queue: %v", err)
	}
	defer q.Close()

	tasks := []*models.WorkTask{
		{ID: "queued-task", Type: models.WorkTaskTypePipelineExecution, Status: models.WorkTaskStatusQueued, Priority: 1, SubmittedAt: time.Now(), ProjectID: "test-project"},
		{ID: "failed-task", Type: models.WorkTaskTypeMLTraining, Status: models.WorkTaskStatusFailed, Priority: 1, SubmittedAt: time.Now(), ProjectID: "test-project"},
	}
	for _, task := range tasks {
		if err := q.Enqueue(task); err != nil {
			t.Fatalf("Failed to enqueue task %s: %v", task.ID, err)
		}
	}

	snapshot := q.Snapshot()
	if snapshot.QueueLength != 2 {
		t.Fatalf("expected queue length 2, got %d", snapshot.QueueLength)
	}
	if snapshot.TotalTasks != 2 {
		t.Fatalf("expected total tasks 2, got %d", snapshot.TotalTasks)
	}
	if snapshot.FailedTasks != 1 {
		t.Fatalf("expected failed task count 1, got %d", snapshot.FailedTasks)
	}
	if snapshot.TasksByStatus[string(models.WorkTaskStatusQueued)] != 1 {
		t.Fatalf("expected queued status count 1, got %#v", snapshot.TasksByStatus)
	}
	if snapshot.TasksByType[string(models.WorkTaskTypeMLTraining)] != 1 {
		t.Fatalf("expected ml_training type count 1, got %#v", snapshot.TasksByType)
	}
}

func TestQueueReloadsPersistedTasks(t *testing.T) {
	store, err := metadatastore.NewSQLiteStore(filepath.Join(t.TempDir(), "queue.db"))
	if err != nil {
		t.Fatalf("failed to create sqlite store: %v", err)
	}
	defer store.Close()

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

	q, err := NewQueue(store)
	if err != nil {
		t.Fatalf("failed to create durable queue: %v", err)
	}
	task := &models.WorkTask{
		ID:          "persisted-task",
		Type:        models.WorkTaskTypeMLTraining,
		Status:      models.WorkTaskStatusQueued,
		Priority:    2,
		SubmittedAt: now,
		ProjectID:   project.ID,
		TaskSpec:    models.TaskSpec{ModelID: "model-1", ProjectID: project.ID},
	}
	if err := q.Enqueue(task); err != nil {
		t.Fatalf("failed to enqueue task: %v", err)
	}

	reloaded, err := NewQueue(store)
	if err != nil {
		t.Fatalf("failed to reload durable queue: %v", err)
	}
	defer reloaded.Close()

	persisted, err := reloaded.GetWorkTask(task.ID)
	if err != nil {
		t.Fatalf("expected persisted task to reload: %v", err)
	}
	if persisted.Status != models.WorkTaskStatusQueued {
		t.Fatalf("expected queued status after reload, got %s", persisted.Status)
	}
	peeked, err := reloaded.PeekNext()
	if err != nil {
		t.Fatalf("expected queued task to be available after reload: %v", err)
	}
	if peeked == nil || peeked.ID != task.ID {
		t.Fatalf("expected task %s at front of queue, got %#v", task.ID, peeked)
	}
}
