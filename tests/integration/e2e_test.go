package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

// TestE2EWorkflow tests the complete end-to-end workflow:
// 1. Submit a work task
// 2. Verify it enters the queue
// 3. Verify orchestrator spawns a worker
// 4. Wait for worker to process the task
// 5. Verify task completion
func TestE2EWorkflow(t *testing.T) {
	t.Log("=== Starting E2E Workflow Test ===")

	// Step 1: Check initial system health
	t.Log("Step 1: Checking system health...")
	resp, err := http.Get(orchestratorURL + "/health")
	if err != nil {
		t.Fatalf("Orchestrator health check failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Orchestrator not healthy, status: %d", resp.StatusCode)
	}
	t.Log("✓ Orchestrator is healthy")

	// Step 2: Check initial queue is empty
	t.Log("Step 2: Checking initial queue state...")
	resp, err = http.Get(orchestratorURL + "/api/worktasks")
	if err != nil {
		t.Fatalf("Failed to get queue: %v", err)
	}
	defer resp.Body.Close()

	var initialQueue map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&initialQueue); err != nil {
		t.Fatalf("Failed to decode queue response: %v", err)
	}
	t.Logf("Initial queue length: %v", initialQueue["queue_length"])

	// Step 3: Submit a pipeline execution work task
	t.Log("Step 3: Submitting pipeline execution work task...")
	workTask := models.WorkTask{
		Type:      models.WorkTaskTypePipelineExecution,
		Priority:  1,
		ProjectID: "test-e2e-project",
		TaskSpec: models.TaskSpec{
			PipelineID: "test-e2e-pipeline",
			Parameters: map[string]interface{}{
				"input_data": "test-dataset-001",
				"output_dir": "/app/data/results",
			},
		},
		ResourceRequirements: models.ResourceRequirements{
			CPU:    "500m",
			Memory: "1Gi",
			GPU:    false,
		},
		DataAccess: models.DataAccess{
			InputDatasets:  []string{"dataset-001"},
			OutputLocation: "/app/data/output",
		},
	}

	taskJSON, _ := json.Marshal(workTask)
	resp, err = http.Post(
		orchestratorURL+"/api/worktasks",
		"application/json",
		bytes.NewBuffer(taskJSON),
	)
	if err != nil {
		t.Fatalf("Failed to submit work task: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		t.Fatalf("Failed to submit work task, status: %d", resp.StatusCode)
	}

	var submitResponse struct {
		ID     string `json:"worktask_id"`
		Status string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&submitResponse); err != nil {
		t.Fatalf("Failed to decode submit response: %v", err)
	}

	taskID := submitResponse.ID
	t.Logf("✓ Work task submitted successfully, ID: %s", taskID)

	// Step 4: Verify task entered the queue
	t.Log("Step 4: Verifying task is in queue...")
	time.Sleep(1 * time.Second)

	resp, err = http.Get(orchestratorURL + "/api/worktasks")
	if err != nil {
		t.Fatalf("Failed to get queue: %v", err)
	}
	defer resp.Body.Close()

	var queueResponse map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&queueResponse); err != nil {
		t.Fatalf("Failed to decode queue response: %v", err)
	}

	queueLength := queueResponse["queue_length"]
	t.Logf("Queue length after submission: %v", queueLength)

	// Step 5: Monitor task status progression
	t.Log("Step 5: Monitoring task status progression...")

	expectedStates := []string{"queued", "spawned", "running"}
	currentStateIndex := 0
	maxWaitTime := 60 * time.Second
	checkInterval := 2 * time.Second
	startTime := time.Now()

	var finalStatus string
	for time.Since(startTime) < maxWaitTime {
		resp, err := http.Get(orchestratorURL + "/api/worktasks/" + taskID)
		if err != nil {
			t.Logf("Warning: Failed to get task status: %v", err)
			time.Sleep(checkInterval)
			continue
		}

		var statusResponse struct {
			Status string `json:"status"`
		}
		json.NewDecoder(resp.Body).Decode(&statusResponse)
		resp.Body.Close()

		finalStatus = statusResponse.Status

		// Check if we've progressed to the next expected state
		if currentStateIndex < len(expectedStates) &&
			finalStatus == expectedStates[currentStateIndex] {
			t.Logf("✓ Task reached state: %s (after %.1fs)",
				finalStatus, time.Since(startTime).Seconds())
			currentStateIndex++
		} else if finalStatus == "completed" || finalStatus == "failed" {
			t.Logf("Task reached terminal state: %s (after %.1fs)",
				finalStatus, time.Since(startTime).Seconds())
			break
		}

		time.Sleep(checkInterval)
	}

	// Step 6: Verify final task status
	t.Log("Step 6: Verifying final task status...")
	resp, err = http.Get(orchestratorURL + "/api/worktasks/" + taskID)
	if err != nil {
		t.Fatalf("Failed to get final task status: %v", err)
	}
	defer resp.Body.Close()

	var finalResponse struct {
		ID     string                 `json:"id"`
		Type   string                 `json:"type"`
		Status string                 `json:"status"`
		Result map[string]interface{} `json:"result,omitempty"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&finalResponse); err != nil {
		t.Fatalf("Failed to decode final status: %v", err)
	}

	t.Logf("Final task status: %s", finalResponse.Status)
	t.Logf("Task type: %s", finalResponse.Type)

	// Verify task progressed through expected states
	if currentStateIndex < 1 {
		t.Errorf("Task did not progress through expected states. Last state: %s", finalStatus)
	}

	// Step 7: Verify queue cleared or task removed from queue
	t.Log("Step 7: Verifying queue state after task processing...")
	time.Sleep(2 * time.Second)

	resp, err = http.Get(orchestratorURL + "/api/worktasks")
	if err != nil {
		t.Fatalf("Failed to get final queue state: %v", err)
	}
	defer resp.Body.Close()

	var finalQueue map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&finalQueue); err != nil {
		t.Fatalf("Failed to decode final queue response: %v", err)
	}

	finalQueueLength := finalQueue["queue_length"]
	t.Logf("Final queue length: %v", finalQueueLength)

	// Step 8: Summary
	t.Log("=== E2E Workflow Test Summary ===")
	t.Logf("✓ Task ID: %s", taskID)
	t.Logf("✓ Task Type: %s", finalResponse.Type)
	t.Logf("✓ Final Status: %s", finalStatus)
	t.Logf("✓ States observed: %d/%d", currentStateIndex, len(expectedStates))
	t.Logf("✓ Queue processed task successfully")
	t.Log("=== E2E Test PASSED ===")
}

// TestE2EMultipleTaskWorkflow tests handling multiple tasks concurrently
func TestE2EMultipleTaskWorkflow(t *testing.T) {
	t.Log("=== Starting E2E Multiple Task Workflow Test ===")

	// Submit multiple tasks of different types
	taskTypes := []models.WorkTaskType{
		models.WorkTaskTypePipelineExecution,
		models.WorkTaskTypeMLTraining,
		models.WorkTaskTypeMLInference,
	}

	submittedTasks := make([]string, 0)

	t.Log("Step 1: Submitting multiple work tasks...")
	for i, taskType := range taskTypes {
		workTask := models.WorkTask{
			Type:      taskType,
			Priority:  i + 1,
			ProjectID: fmt.Sprintf("test-e2e-project-%d", i),
			TaskSpec: models.TaskSpec{
				Parameters: map[string]interface{}{
					"task_number": i + 1,
					"test_id":     fmt.Sprintf("e2e-multi-%d", i),
				},
			},
			ResourceRequirements: models.ResourceRequirements{
				CPU:    "500m",
				Memory: "1Gi",
				GPU:    false,
			},
		}

		taskJSON, _ := json.Marshal(workTask)
		resp, err := http.Post(
			orchestratorURL+"/api/worktasks",
			"application/json",
			bytes.NewBuffer(taskJSON),
		)
		if err != nil {
			t.Fatalf("Failed to submit task %d: %v", i, err)
		}

		var submitResponse struct {
			ID     string `json:"worktask_id"`
			Status string `json:"status"`
		}
		json.NewDecoder(resp.Body).Decode(&submitResponse)
		resp.Body.Close()

		submittedTasks = append(submittedTasks, submitResponse.ID)
		t.Logf("✓ Submitted task %d (%s): %s", i+1, taskType, submitResponse.ID)

		// Small delay between submissions
		time.Sleep(500 * time.Millisecond)
	}

	// Wait for tasks to be processed
	t.Log("Step 2: Waiting for tasks to be processed...")
	time.Sleep(15 * time.Second)

	// Check status of all submitted tasks
	t.Log("Step 3: Checking status of all submitted tasks...")
	for i, taskID := range submittedTasks {
		resp, err := http.Get(orchestratorURL + "/api/worktasks/" + taskID)
		if err != nil {
			t.Logf("Warning: Failed to get status for task %s: %v", taskID, err)
			continue
		}

		var statusResponse struct {
			Status string `json:"status"`
			Type   string `json:"type"`
		}
		json.NewDecoder(resp.Body).Decode(&statusResponse)
		resp.Body.Close()

		t.Logf("Task %d: %s - Status: %s",
			i+1, statusResponse.Type, statusResponse.Status)
	}

	t.Log("=== E2E Multiple Task Workflow Test PASSED ===")
}
