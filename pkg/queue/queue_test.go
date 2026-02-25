package queue

import (
	"testing"
	"time"

	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

// TestEnqueueDequeue tests basic queue operations
func TestEnqueueDequeue(t *testing.T) {
	q, err := NewQueue()
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
	q, err := NewQueue()
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
	q, err := NewQueue()
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
