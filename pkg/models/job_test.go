package models

import (
	"testing"
	"time"
)

// TestJobCreation tests basic job creation
func TestJobCreation(t *testing.T) {
	job := &Job{
		ID:          "test-123",
		Type:        JobTypePipelineExecution,
		Status:      JobStatusQueued,
		Priority:    1,
		SubmittedAt: time.Now(),
		ProjectID:   "project-456",
		TaskSpec: TaskSpec{
			PipelineID: "pipeline-789",
			Parameters: map[string]interface{}{
				"param1": "value1",
			},
		},
		ResourceRequirements: ResourceRequirements{
			CPU:    "500m",
			Memory: "1Gi",
			GPU:    false,
		},
	}

	if job.ID != "test-123" {
		t.Errorf("Expected ID test-123, got %s", job.ID)
	}

	if job.Type != JobTypePipelineExecution {
		t.Errorf("Expected type %s, got %s", JobTypePipelineExecution, job.Type)
	}

	if job.Status != JobStatusQueued {
		t.Errorf("Expected status %s, got %s", JobStatusQueued, job.Status)
	}
}

// TestJobTypes tests all job types are valid
func TestJobTypes(t *testing.T) {
	jobTypes := []JobType{
		JobTypePipelineExecution,
		JobTypeMLTraining,
		JobTypeMLInference,
		JobTypeDigitalTwinUpdate,
	}

	for _, jobType := range jobTypes {
		job := &Job{
			ID:          "test",
			Type:        jobType,
			Status:      JobStatusQueued,
			Priority:    1,
			SubmittedAt: time.Now(),
			ProjectID:   "test-project",
		}

		if job.Type != jobType {
			t.Errorf("Expected job type %s, got %s", jobType, job.Type)
		}
	}
}

// TestJobStatuses tests all job statuses are valid
func TestJobStatuses(t *testing.T) {
	statuses := []JobStatus{
		JobStatusQueued,
		JobStatusScheduled,
		JobStatusSpawned,
		JobStatusExecuting,
		JobStatusCompleted,
		JobStatusFailed,
		JobStatusTimeout,
		JobStatusCancelled,
	}

	for _, status := range statuses {
		job := &Job{
			ID:          "test",
			Type:        JobTypePipelineExecution,
			Status:      status,
			Priority:    1,
			SubmittedAt: time.Now(),
			ProjectID:   "test-project",
		}

		if job.Status != status {
			t.Errorf("Expected job status %s, got %s", status, job.Status)
		}
	}
}

// TestResourceRequirements tests resource requirements
func TestResourceRequirements(t *testing.T) {
	req := ResourceRequirements{
		CPU:    "2",
		Memory: "4Gi",
		GPU:    true,
	}

	if req.CPU != "2" {
		t.Errorf("Expected CPU 2, got %s", req.CPU)
	}

	if req.Memory != "4Gi" {
		t.Errorf("Expected Memory 4Gi, got %s", req.Memory)
	}

	if !req.GPU {
		t.Error("Expected GPU to be true")
	}
}
