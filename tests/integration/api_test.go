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

// TestJobSubmission tests job submission through the API
func TestJobSubmission(t *testing.T) {
	jobRequest := models.JobSubmissionRequest{
		Type:      models.JobTypePipelineExecution,
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

	jsonData, err := json.Marshal(jobRequest)
	if err != nil {
		t.Fatalf("Failed to marshal job request: %v", err)
	}

	resp, err := http.Post(
		orchestratorURL+"/api/jobs",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		t.Fatalf("Failed to submit job: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", resp.StatusCode)
	}

	var job models.Job
	if err := json.NewDecoder(resp.Body).Decode(&job); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if job.ID == "" {
		t.Error("Expected non-empty job ID")
	}

	if job.Type != models.JobTypePipelineExecution {
		t.Errorf("Expected job type %s, got %s", models.JobTypePipelineExecution, job.Type)
	}

	if job.Status != models.JobStatusQueued {
		t.Errorf("Expected job status %s, got %s", models.JobStatusQueued, job.Status)
	}

	// Wait a bit and check if the job was processed
	time.Sleep(10 * time.Second)

	// Get the job status
	getResp, err := http.Get(orchestratorURL + "/api/jobs/" + job.ID)
	if err != nil {
		t.Fatalf("Failed to get job: %v", err)
	}
	defer getResp.Body.Close()

	var updatedJob models.Job
	if err := json.NewDecoder(getResp.Body).Decode(&updatedJob); err != nil {
		t.Fatalf("Failed to decode job response: %v", err)
	}

	t.Logf("Job status after 10s: %s", updatedJob.Status)
}

// TestQueueLength tests the queue length endpoint
func TestQueueLength(t *testing.T) {
	resp, err := http.Get(orchestratorURL + "/api/jobs")
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

// TestMultipleJobSubmissions tests submitting multiple jobs
func TestMultipleJobSubmissions(t *testing.T) {
	jobTypes := []models.JobType{
		models.JobTypePipelineExecution,
		models.JobTypeMLTraining,
		models.JobTypeMLInference,
		models.JobTypeDigitalTwinUpdate,
	}

	submittedJobs := make([]string, 0)

	for i, jobType := range jobTypes {
		jobRequest := models.JobSubmissionRequest{
			Type:      jobType,
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

		jsonData, err := json.Marshal(jobRequest)
		if err != nil {
			t.Fatalf("Failed to marshal job request: %v", err)
		}

		resp, err := http.Post(
			orchestratorURL+"/api/jobs",
			"application/json",
			bytes.NewBuffer(jsonData),
		)
		if err != nil {
			t.Fatalf("Failed to submit job: %v", err)
		}

		var job models.Job
		if err := json.NewDecoder(resp.Body).Decode(&job); err != nil {
			resp.Body.Close()
			t.Fatalf("Failed to decode response: %v", err)
		}
		resp.Body.Close()

		submittedJobs = append(submittedJobs, job.ID)
		t.Logf("Submitted job %s of type %s", job.ID, jobType)
	}

	t.Logf("Successfully submitted %d jobs", len(submittedJobs))

	// Wait for jobs to be processed
	time.Sleep(15 * time.Second)

	// Check status of all jobs
	for _, jobID := range submittedJobs {
		resp, err := http.Get(orchestratorURL + "/api/jobs/" + jobID)
		if err != nil {
			t.Logf("Warning: Failed to get job %s: %v", jobID, err)
			continue
		}

		var job models.Job
		if err := json.NewDecoder(resp.Body).Decode(&job); err != nil {
			resp.Body.Close()
			t.Logf("Warning: Failed to decode job %s: %v", jobID, err)
			continue
		}
		resp.Body.Close()

		t.Logf("Job %s status: %s", jobID, job.Status)
	}
}
