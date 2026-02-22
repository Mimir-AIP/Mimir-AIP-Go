package models

import (
	"testing"
	"time"
)

// TestWorkTaskCreation tests basic work task creation
func TestWorkTaskCreation(t *testing.T) {
	task := &WorkTask{
		ID:          "test-123",
		Type:        WorkTaskTypePipelineExecution,
		Status:      WorkTaskStatusQueued,
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

	if task.ID != "test-123" {
		t.Errorf("Expected ID test-123, got %s", task.ID)
	}

	if task.Type != WorkTaskTypePipelineExecution {
		t.Errorf("Expected type %s, got %s", WorkTaskTypePipelineExecution, task.Type)
	}

	if task.Status != WorkTaskStatusQueued {
		t.Errorf("Expected status %s, got %s", WorkTaskStatusQueued, task.Status)
	}
}

// TestWorkTaskTypes tests all work task types are valid
func TestWorkTaskTypes(t *testing.T) {
	taskTypes := []WorkTaskType{
		WorkTaskTypePipelineExecution,
		WorkTaskTypeMLTraining,
		WorkTaskTypeMLInference,
		WorkTaskTypeDigitalTwinUpdate,
	}

	for _, taskType := range taskTypes {
		task := &WorkTask{
			ID:          "test",
			Type:        taskType,
			Status:      WorkTaskStatusQueued,
			Priority:    1,
			SubmittedAt: time.Now(),
			ProjectID:   "test-project",
		}

		if task.Type != taskType {
			t.Errorf("Expected work task type %s, got %s", taskType, task.Type)
		}
	}
}

// TestWorkTaskStatuses tests all work task statuses are valid
func TestWorkTaskStatuses(t *testing.T) {
	statuses := []WorkTaskStatus{
		WorkTaskStatusQueued,
		WorkTaskStatusScheduled,
		WorkTaskStatusSpawned,
		WorkTaskStatusExecuting,
		WorkTaskStatusCompleted,
		WorkTaskStatusFailed,
		WorkTaskStatusTimeout,
		WorkTaskStatusCancelled,
	}

	for _, status := range statuses {
		task := &WorkTask{
			ID:          "test",
			Type:        WorkTaskTypePipelineExecution,
			Status:      status,
			Priority:    1,
			SubmittedAt: time.Now(),
			ProjectID:   "test-project",
		}

		if task.Status != status {
			t.Errorf("Expected work task status %s, got %s", status, task.Status)
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
