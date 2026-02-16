package queue

import (
	"testing"
	"time"

	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

// TestEnqueueDequeue tests basic queue operations
func TestEnqueueDequeue(t *testing.T) {
	// This test requires a running Redis instance
	// Skip if REDIS_URL is not set
	t.Skip("Integration test - requires Redis")

	q, err := NewQueue("redis://localhost:6379")
	if err != nil {
		t.Fatalf("Failed to create queue: %v", err)
	}
	defer q.Close()

	// Create a test job
	job := &models.Job{
		ID:          "test-job-1",
		Type:        models.JobTypePipelineExecution,
		Status:      models.JobStatusQueued,
		Priority:    1,
		SubmittedAt: time.Now(),
		ProjectID:   "test-project",
		TaskSpec: models.TaskSpec{
			PipelineID: "test-pipeline",
			Parameters: map[string]interface{}{},
		},
	}

	// Enqueue the job
	if err := q.Enqueue(job); err != nil {
		t.Fatalf("Failed to enqueue job: %v", err)
	}

	// Dequeue the job
	dequeuedJob, err := q.Dequeue()
	if err != nil {
		t.Fatalf("Failed to dequeue job: %v", err)
	}

	if dequeuedJob == nil {
		t.Fatal("Dequeued job is nil")
	}

	if dequeuedJob.ID != job.ID {
		t.Errorf("Expected job ID %s, got %s", job.ID, dequeuedJob.ID)
	}
}

// TestQueueLength tests queue length tracking
func TestQueueLength(t *testing.T) {
	t.Skip("Integration test - requires Redis")

	q, err := NewQueue("redis://localhost:6379")
	if err != nil {
		t.Fatalf("Failed to create queue: %v", err)
	}
	defer q.Close()

	initialLength, err := q.QueueLength()
	if err != nil {
		t.Fatalf("Failed to get queue length: %v", err)
	}

	// Create and enqueue a job
	job := &models.Job{
		ID:          "test-job-2",
		Type:        models.JobTypePipelineExecution,
		Status:      models.JobStatusQueued,
		Priority:    1,
		SubmittedAt: time.Now(),
		ProjectID:   "test-project",
	}

	if err := q.Enqueue(job); err != nil {
		t.Fatalf("Failed to enqueue job: %v", err)
	}

	newLength, err := q.QueueLength()
	if err != nil {
		t.Fatalf("Failed to get queue length: %v", err)
	}

	if newLength != initialLength+1 {
		t.Errorf("Expected queue length %d, got %d", initialLength+1, newLength)
	}
}

// TestJobStatusUpdate tests job status updates
func TestJobStatusUpdate(t *testing.T) {
	t.Skip("Integration test - requires Redis")

	q, err := NewQueue("redis://localhost:6379")
	if err != nil {
		t.Fatalf("Failed to create queue: %v", err)
	}
	defer q.Close()

	// Create and enqueue a job
	job := &models.Job{
		ID:          "test-job-3",
		Type:        models.JobTypePipelineExecution,
		Status:      models.JobStatusQueued,
		Priority:    1,
		SubmittedAt: time.Now(),
		ProjectID:   "test-project",
	}

	if err := q.Enqueue(job); err != nil {
		t.Fatalf("Failed to enqueue job: %v", err)
	}

	// Update job status
	if err := q.UpdateJobStatus(job.ID, models.JobStatusExecuting, ""); err != nil {
		t.Fatalf("Failed to update job status: %v", err)
	}

	// Get the updated job
	updatedJob, err := q.GetJob(job.ID)
	if err != nil {
		t.Fatalf("Failed to get job: %v", err)
	}

	if updatedJob.Status != models.JobStatusExecuting {
		t.Errorf("Expected status %s, got %s", models.JobStatusExecuting, updatedJob.Status)
	}
}
