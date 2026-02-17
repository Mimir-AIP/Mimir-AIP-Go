package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

var (
	orchestratorURL = getEnv("ORCHESTRATOR_URL", "http://localhost:8080")
	frontendURL     = getEnv("FRONTEND_URL", "http://localhost:80")
)

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// TestHealthCheck tests the orchestrator health endpoint
func TestHealthCheck(t *testing.T) {
	resp, err := http.Get(orchestratorURL + "/health")
	if err != nil {
		t.Fatalf("Failed to connect to orchestrator: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result["status"] != "healthy" {
		t.Errorf("Expected status 'healthy', got '%s'", result["status"])
	}
}

// TestReadinessCheck tests the orchestrator readiness endpoint
func TestReadinessCheck(t *testing.T) {
	resp, err := http.Get(orchestratorURL + "/ready")
	if err != nil {
		t.Fatalf("Failed to connect to orchestrator: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result["status"] != "ready" {
		t.Errorf("Expected status 'ready', got '%s'", result["status"])
	}
}

// TestWorkTaskSubmission tests work task submission through the API
func TestWorkTaskSubmission(t *testing.T) {
	taskRequest := models.WorkTaskSubmissionRequest{
		Type:      models.WorkTaskTypePipelineExecution,
		Priority:  1,
		ProjectID: "test-project",
		TaskSpec: models.TaskSpec{
			PipelineID: "test-pipeline",
			ProjectID:  "test-project",
			Parameters: map[string]interface{}{
				"test_param": "test_value",
			},
		},
		ResourceRequirements: models.ResourceRequirements{
			CPU:    "500m",
			Memory: "1Gi",
			GPU:    false,
		},
		DataAccess: models.DataAccess{
			InputDatasets:  []string{},
			OutputLocation: "s3://test-bucket/results/",
		},
	}

	jsonData, err := json.Marshal(taskRequest)
	if err != nil {
		t.Fatalf("Failed to marshal work task request: %v", err)
	}

	resp, err := http.Post(
		orchestratorURL+"/api/worktasks",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		t.Fatalf("Failed to submit work task: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", resp.StatusCode)
	}

	var task models.WorkTask
	if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if task.ID == "" {
		t.Error("Expected non-empty work task ID")
	}

	if task.Type != models.WorkTaskTypePipelineExecution {
		t.Errorf("Expected work task type %s, got %s", models.WorkTaskTypePipelineExecution, task.Type)
	}

	if task.Status != models.WorkTaskStatusQueued {
		t.Errorf("Expected work task status %s, got %s", models.WorkTaskStatusQueued, task.Status)
	}

	// Wait a bit and check if the work task was processed
	time.Sleep(10 * time.Second)

	// Get the work task status
	getResp, err := http.Get(orchestratorURL + "/api/worktasks/" + task.ID)
	if err != nil {
		t.Fatalf("Failed to get work task: %v", err)
	}
	defer getResp.Body.Close()

	var updatedTask models.WorkTask
	if err := json.NewDecoder(getResp.Body).Decode(&updatedTask); err != nil {
		t.Fatalf("Failed to decode work task response: %v", err)
	}

	t.Logf("Work task status after 10s: %s", updatedTask.Status)
}

// TestQueueLength tests the queue length endpoint
func TestQueueLength(t *testing.T) {
	resp, err := http.Get(orchestratorURL + "/api/worktasks")
	if err != nil {
		t.Fatalf("Failed to get queue length: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if _, ok := result["queue_length"]; !ok {
		t.Error("Expected queue_length in response")
	}

	t.Logf("Current queue length: %v", result["queue_length"])
}

// TestFrontendAvailability tests if the frontend is accessible
func TestFrontendAvailability(t *testing.T) {
	resp, err := http.Get(frontendURL)
	if err != nil {
		t.Fatalf("Failed to connect to frontend: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

// TestMultipleWorkTaskSubmissions tests submitting multiple work tasks
func TestMultipleWorkTaskSubmissions(t *testing.T) {
	taskTypes := []models.WorkTaskType{
		models.WorkTaskTypePipelineExecution,
		models.WorkTaskTypeMLTraining,
		models.WorkTaskTypeMLInference,
		models.WorkTaskTypeDigitalTwinUpdate,
	}

	submittedTasks := make([]string, 0)

	for i, taskType := range taskTypes {
		taskRequest := models.WorkTaskSubmissionRequest{
			Type:      taskType,
			Priority:  1,
			ProjectID: fmt.Sprintf("test-project-%d", i),
			TaskSpec: models.TaskSpec{
				PipelineID: fmt.Sprintf("test-pipeline-%d", i),
				ProjectID:  fmt.Sprintf("test-project-%d", i),
				Parameters: map[string]interface{}{},
			},
			ResourceRequirements: models.ResourceRequirements{
				CPU:    "500m",
				Memory: "1Gi",
				GPU:    false,
			},
			DataAccess: models.DataAccess{
				InputDatasets:  []string{},
				OutputLocation: fmt.Sprintf("s3://test-bucket/results/%d/", i),
			},
		}

		jsonData, err := json.Marshal(taskRequest)
		if err != nil {
			t.Fatalf("Failed to marshal work task request: %v", err)
		}

		resp, err := http.Post(
			orchestratorURL+"/api/worktasks",
			"application/json",
			bytes.NewBuffer(jsonData),
		)
		if err != nil {
			t.Fatalf("Failed to submit work task: %v", err)
		}

		var task models.WorkTask
		if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
			resp.Body.Close()
			t.Fatalf("Failed to decode response: %v", err)
		}
		resp.Body.Close()

		submittedTasks = append(submittedTasks, task.ID)
		t.Logf("Submitted work task %s of type %s", task.ID, taskType)
	}

	t.Logf("Successfully submitted %d work tasks", len(submittedTasks))

	// Wait for work tasks to be processed
	time.Sleep(15 * time.Second)

	// Check status of all work tasks
	for _, taskID := range submittedTasks {
		resp, err := http.Get(orchestratorURL + "/api/worktasks/" + taskID)
		if err != nil {
			t.Logf("Warning: Failed to get work task %s: %v", taskID, err)
			continue
		}

		var task models.WorkTask
		if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
			resp.Body.Close()
			t.Logf("Warning: Failed to decode work task %s: %v", taskID, err)
			continue
		}
		resp.Body.Close()

		t.Logf("Work task %s status: %s", taskID, task.Status)
	}
}
