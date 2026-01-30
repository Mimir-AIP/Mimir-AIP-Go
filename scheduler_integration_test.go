package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestScheduler_CreateAndListJobs tests creating and listing scheduled jobs
func TestScheduler_CreateAndListJobs(t *testing.T) {
	server := NewServer()
	require.NotNil(t, server)

	// Test 1: List jobs (should work even if empty)
	t.Run("List jobs initially", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/scheduler/jobs", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var jobs []map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &jobs)
		require.NoError(t, err)
		// Should be an array (could be empty)
		assert.NotNil(t, jobs)
	})

	// Test 2: Create a scheduled job
	var jobID string
	t.Run("Create scheduled job", func(t *testing.T) {
		createReq := map[string]any{
			"id":        "test-job-1",
			"name":      "Test Scheduled Job",
			"pipeline":  "test-pipeline",
			"cron_expr": "*/5 * * * *",
		}

		body, _ := json.Marshal(createReq)
		req := httptest.NewRequest("POST", "/api/v1/scheduler/jobs", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		// Should return 200 for successful creation
		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "Job created successfully", response["message"])
		assert.Equal(t, "test-job-1", response["job_id"])

		jobID = "test-job-1"
	})

	// Test 3: List jobs again (should now include our job)
	t.Run("List jobs after creation", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/scheduler/jobs", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var jobs []map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &jobs)
		require.NoError(t, err)

		// Find our job
		found := false
		for _, job := range jobs {
			if job["id"] == "test-job-1" {
				found = true
				assert.Equal(t, "Test Scheduled Job", job["name"])
				assert.Equal(t, "test-pipeline", job["pipeline"])
				assert.Equal(t, "*/5 * * * *", job["cron_expr"])
				break
			}
		}
		assert.True(t, found, "Created job should be in the list")
	})

	// Test 4: Get specific job
	t.Run("Get specific job", func(t *testing.T) {
		require.NotEmpty(t, jobID)

		req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/scheduler/jobs/%s", jobID), nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var job map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &job)
		require.NoError(t, err)

		assert.Equal(t, jobID, job["id"])
		assert.Equal(t, "Test Scheduled Job", job["name"])
		assert.Equal(t, "test-pipeline", job["pipeline"])
	})
}

// TestScheduler_EnableDisableJob tests enabling and disabling jobs
func TestScheduler_EnableDisableJob(t *testing.T) {
	server := NewServer()
	require.NotNil(t, server)

	// First, create a job
	createReq := map[string]any{
		"id":        "test-enable-job",
		"name":      "Test Enable/Disable Job",
		"pipeline":  "test-pipeline",
		"cron_expr": "0 * * * *",
	}

	body, _ := json.Marshal(createReq)
	req := httptest.NewRequest("POST", "/api/v1/scheduler/jobs", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Skip("Job creation failed, skipping enable/disable test")
	}

	jobID := "test-enable-job"

	t.Run("Disable job", func(t *testing.T) {
		req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/scheduler/jobs/%s/disable", jobID), nil)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "Job disabled successfully", response["message"])
		assert.Equal(t, jobID, response["job_id"])

		// Verify job is disabled
		req = httptest.NewRequest("GET", fmt.Sprintf("/api/v1/scheduler/jobs/%s", jobID), nil)
		w = httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var job map[string]any
		err = json.Unmarshal(w.Body.Bytes(), &job)
		require.NoError(t, err)

		enabled, ok := job["enabled"].(bool)
		require.True(t, ok, "enabled field should be a boolean")
		assert.False(t, enabled, "Job should be disabled")
	})

	t.Run("Enable job", func(t *testing.T) {
		req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/scheduler/jobs/%s/enable", jobID), nil)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "Job enabled successfully", response["message"])
		assert.Equal(t, jobID, response["job_id"])

		// Verify job is enabled
		req = httptest.NewRequest("GET", fmt.Sprintf("/api/v1/scheduler/jobs/%s", jobID), nil)
		w = httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var job map[string]any
		err = json.Unmarshal(w.Body.Bytes(), &job)
		require.NoError(t, err)

		enabled, ok := job["enabled"].(bool)
		require.True(t, ok, "enabled field should be a boolean")
		assert.True(t, enabled, "Job should be enabled")
	})
}

// TestScheduler_UpdateJob tests updating a scheduled job
func TestScheduler_UpdateJob(t *testing.T) {
	server := NewServer()
	require.NotNil(t, server)

	// First, create a job
	createReq := map[string]any{
		"id":        "test-update-job",
		"name":      "Test Update Job",
		"pipeline":  "test-pipeline",
		"cron_expr": "0 * * * *",
	}

	body, _ := json.Marshal(createReq)
	req := httptest.NewRequest("POST", "/api/v1/scheduler/jobs", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Skip("Job creation failed, skipping update test")
	}

	jobID := "test-update-job"

	t.Run("Update job name", func(t *testing.T) {
		updateReq := map[string]any{
			"name": "Updated Job Name",
		}

		body, _ := json.Marshal(updateReq)
		req := httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/scheduler/jobs/%s", jobID), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		// Verify job was updated
		req = httptest.NewRequest("GET", fmt.Sprintf("/api/v1/scheduler/jobs/%s", jobID), nil)
		w = httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var job map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &job)
		require.NoError(t, err)

		assert.Equal(t, "Updated Job Name", job["name"])
	})

	t.Run("Update job cron expression", func(t *testing.T) {
		updateReq := map[string]any{
			"cron_expr": "*/10 * * * *",
		}

		body, _ := json.Marshal(updateReq)
		req := httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/scheduler/jobs/%s", jobID), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		// Verify job was updated
		req = httptest.NewRequest("GET", fmt.Sprintf("/api/v1/scheduler/jobs/%s", jobID), nil)
		w = httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var job map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &job)
		require.NoError(t, err)

		assert.Equal(t, "*/10 * * * *", job["cron_expr"])
	})

	t.Run("Update multiple fields", func(t *testing.T) {
		updateReq := map[string]any{
			"name":      "Fully Updated Job",
			"pipeline":  "updated-pipeline",
			"cron_expr": "0 12 * * *",
		}

		body, _ := json.Marshal(updateReq)
		req := httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/scheduler/jobs/%s", jobID), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		// Verify job was updated
		req = httptest.NewRequest("GET", fmt.Sprintf("/api/v1/scheduler/jobs/%s", jobID), nil)
		w = httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var job map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &job)
		require.NoError(t, err)

		assert.Equal(t, "Fully Updated Job", job["name"])
		assert.Equal(t, "updated-pipeline", job["pipeline"])
		assert.Equal(t, "0 12 * * *", job["cron_expr"])
	})
}

// TestScheduler_DeleteJob tests deleting a scheduled job
func TestScheduler_DeleteJob(t *testing.T) {
	server := NewServer()
	require.NotNil(t, server)

	// First, create a job
	createReq := map[string]any{
		"id":        "test-delete-job",
		"name":      "Test Delete Job",
		"pipeline":  "test-pipeline",
		"cron_expr": "0 * * * *",
	}

	body, _ := json.Marshal(createReq)
	req := httptest.NewRequest("POST", "/api/v1/scheduler/jobs", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Skip("Job creation failed, skipping delete test")
	}

	jobID := "test-delete-job"

	t.Run("Delete job", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/v1/scheduler/jobs/%s", jobID), nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		// Should return 200 or 204
		assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusNoContent,
			"Expected 200 or 204, got %d", w.Code)

		// Verify job is deleted
		req = httptest.NewRequest("GET", fmt.Sprintf("/api/v1/scheduler/jobs/%s", jobID), nil)
		w = httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

// TestScheduler_GetNonExistentJob tests getting a job that doesn't exist
func TestScheduler_GetNonExistentJob(t *testing.T) {
	server := NewServer()
	require.NotNil(t, server)

	t.Run("Get non-existent job", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/scheduler/jobs/non-existent-job-id", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

// TestScheduler_EnableNonExistentJob tests enabling a job that doesn't exist
func TestScheduler_EnableNonExistentJob(t *testing.T) {
	server := NewServer()
	require.NotNil(t, server)

	t.Run("Enable non-existent job", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/scheduler/jobs/non-existent-job-id/enable", nil)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("Disable non-existent job", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/scheduler/jobs/non-existent-job-id/disable", nil)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

// TestScheduler_CreateJobWithInvalidCron tests creating a job with invalid cron expression
func TestScheduler_CreateJobWithInvalidCron(t *testing.T) {
	server := NewServer()
	require.NotNil(t, server)

	t.Run("Create job with invalid cron", func(t *testing.T) {
		createReq := map[string]any{
			"id":        "test-invalid-cron",
			"name":      "Test Invalid Cron",
			"pipeline":  "test-pipeline",
			"cron_expr": "invalid-cron",
		}

		body, _ := json.Marshal(createReq)
		req := httptest.NewRequest("POST", "/api/v1/scheduler/jobs", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		// Should return 400 for invalid cron
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// TestScheduler_CreateJobWithMissingFields tests creating a job with missing required fields
func TestScheduler_CreateJobWithMissingFields(t *testing.T) {
	server := NewServer()
	require.NotNil(t, server)

	t.Run("Create job without id", func(t *testing.T) {
		createReq := map[string]any{
			"name":      "Test Job",
			"pipeline":  "test-pipeline",
			"cron_expr": "0 * * * *",
		}

		body, _ := json.Marshal(createReq)
		req := httptest.NewRequest("POST", "/api/v1/scheduler/jobs", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		// Should return 400 for missing id
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Create job without name", func(t *testing.T) {
		createReq := map[string]any{
			"id":        "test-job",
			"pipeline":  "test-pipeline",
			"cron_expr": "0 * * * *",
		}

		body, _ := json.Marshal(createReq)
		req := httptest.NewRequest("POST", "/api/v1/scheduler/jobs", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		// Should return 400 for missing name
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Create job without pipeline", func(t *testing.T) {
		createReq := map[string]any{
			"id":        "test-job",
			"name":      "Test Job",
			"cron_expr": "0 * * * *",
		}

		body, _ := json.Marshal(createReq)
		req := httptest.NewRequest("POST", "/api/v1/scheduler/jobs", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		// Should return 400 for missing pipeline
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Create job without cron_expr", func(t *testing.T) {
		createReq := map[string]any{
			"id":       "test-job",
			"name":     "Test Job",
			"pipeline": "test-pipeline",
		}

		body, _ := json.Marshal(createReq)
		req := httptest.NewRequest("POST", "/api/v1/scheduler/jobs", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		// Should return 400 for missing cron_expr
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// TestScheduler_CreateDuplicateJob tests creating a job with duplicate ID
func TestScheduler_CreateDuplicateJob(t *testing.T) {
	server := NewServer()
	require.NotNil(t, server)

	// First, create a job
	createReq := map[string]any{
		"id":        "test-duplicate-job",
		"name":      "Test Duplicate Job",
		"pipeline":  "test-pipeline",
		"cron_expr": "0 * * * *",
	}

	body, _ := json.Marshal(createReq)
	req := httptest.NewRequest("POST", "/api/v1/scheduler/jobs", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Skip("Job creation failed, skipping duplicate test")
	}

	t.Run("Create duplicate job", func(t *testing.T) {
		// Try to create the same job again
		body, _ := json.Marshal(createReq)
		req := httptest.NewRequest("POST", "/api/v1/scheduler/jobs", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		// Should return 400 for duplicate ID
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// TestScheduler_JobLogs tests getting job logs
func TestScheduler_JobLogs(t *testing.T) {
	server := NewServer()
	require.NotNil(t, server)

	// First, create a job
	createReq := map[string]any{
		"id":        "test-logs-job",
		"name":      "Test Logs Job",
		"pipeline":  "test-pipeline",
		"cron_expr": "0 * * * *",
	}

	body, _ := json.Marshal(createReq)
	req := httptest.NewRequest("POST", "/api/v1/scheduler/jobs", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)

	jobID := "test-logs-job"

	t.Run("Get job logs", func(t *testing.T) {
		req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/scheduler/jobs/%s/logs", jobID), nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		// May return 200 or 404 depending on implementation
		assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusNotFound,
			"Expected 200 or 404, got %d", w.Code)
	})
}

// TestScheduler_MultipleJobs tests creating and managing multiple jobs
func TestScheduler_MultipleJobs(t *testing.T) {
	server := NewServer()
	require.NotNil(t, server)

	// Create multiple jobs
	jobs := []map[string]any{
		{
			"id":        "multi-job-1",
			"name":      "Multi Job 1",
			"pipeline":  "pipeline-1",
			"cron_expr": "*/5 * * * *",
		},
		{
			"id":        "multi-job-2",
			"name":      "Multi Job 2",
			"pipeline":  "pipeline-2",
			"cron_expr": "*/10 * * * *",
		},
		{
			"id":        "multi-job-3",
			"name":      "Multi Job 3",
			"pipeline":  "pipeline-3",
			"cron_expr": "0 * * * *",
		},
	}

	for _, job := range jobs {
		body, _ := json.Marshal(job)
		req := httptest.NewRequest("POST", "/api/v1/scheduler/jobs", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	}

	t.Run("List all jobs", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/scheduler/jobs", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var jobList []map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &jobList)
		require.NoError(t, err)

		// Count our jobs
		count := 0
		for _, job := range jobList {
			id := job["id"].(string)
			if id == "multi-job-1" || id == "multi-job-2" || id == "multi-job-3" {
				count++
			}
		}
		assert.Equal(t, 3, count, "All 3 jobs should be in the list")
	})

	// Clean up - delete all jobs
	for _, job := range jobs {
		req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/v1/scheduler/jobs/%s", job["id"]), nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
	}
}
