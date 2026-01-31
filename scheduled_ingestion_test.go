package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// Scheduled Ingestion and Job Lifecycle Tests
// ============================================================================

// TestContinuousIngestion_ScheduleAndLifecycle tests creating and managing scheduled jobs
func TestContinuousIngestion_ScheduleAndLifecycle(t *testing.T) {
	server := NewServer()
	require.NotNil(t, server)

	// Create a mock API server
	mockAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]any{
			"data": []map[string]any{
				{"id": "1", "value": 100},
				{"id": "2", "value": 200},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer mockAPI.Close()

	var pipelineID string

	// Create a pipeline
	t.Run("Create pipeline for scheduling", func(t *testing.T) {
		pipelineDef := map[string]any{
			"metadata": map[string]any{
				"name":    "scheduled-ingestion-pipeline",
				"enabled": true,
			},
			"config": map[string]any{
				"name":    "scheduled-ingestion-pipeline",
				"enabled": true,
				"steps": []map[string]any{
					{
						"name":   "fetch-data",
						"plugin": "Input.api",
						"config": map[string]any{
							"url":    mockAPI.URL,
							"method": "GET",
						},
						"output": "api_data",
					},
				},
			},
		}

		body, _ := json.Marshal(pipelineDef)
		req := httptest.NewRequest("POST", "/api/v1/pipelines", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		require.True(t, w.Code == http.StatusOK || w.Code == http.StatusCreated, "Expected 200 or 201, got %d", w.Code)

		var response map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		pipeline, ok := response["pipeline"].(map[string]any)
		require.True(t, ok)

		pipelineID = pipeline["id"].(string)
		assert.NotEmpty(t, pipelineID)
	})

	// Step 4: Create a scheduled job for continuous ingestion
	t.Run("Create scheduled job for continuous ingestion", func(t *testing.T) {
		require.NotEmpty(t, pipelineID)

		createReq := map[string]any{
			"id":        "auto-customer-ingestion",
			"name":      "Automated Customer Data Ingestion",
			"pipeline":  pipelineID,
			"cron_expr": "*/5 * * * *", // Every 5 minutes
			"enabled":   true,
		}

		body, _ := json.Marshal(createReq)
		req := httptest.NewRequest("POST", "/api/v1/scheduler/jobs", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusCreated, "Expected 200 or 201, got %d", w.Code)

		var response map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "auto-customer-ingestion", response["job_id"])
		assert.Contains(t, response["message"], "created")
	})

	// Step 5: Verify the scheduled job exists
	t.Run("Verify scheduled job", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/scheduler/jobs/auto-customer-ingestion", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var job map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &job)
		require.NoError(t, err)

		assert.Equal(t, "auto-customer-ingestion", job["id"])
		assert.Equal(t, "Automated Customer Data Ingestion", job["name"])
		assert.Equal(t, pipelineID, job["pipeline"])
		assert.Equal(t, "*/5 * * * *", job["cron_expr"])
		assert.True(t, job["enabled"].(bool))
	})
}

// TestContinuousIngestion_JobLifecycle tests the full lifecycle of a scheduled job
func TestContinuousIngestion_JobLifecycle(t *testing.T) {
	server := NewServer()
	require.NotNil(t, server)

	tmpDir := t.TempDir()
	csvFile := filepath.Join(tmpDir, "data.csv")
	os.WriteFile(csvFile, []byte("name,value\nTest,100"), 0644)

	// Create a pipeline
	pipelineDef := map[string]any{
		"metadata": map[string]any{
			"name":    "lifecycle-test-pipeline",
			"enabled": true,
		},
		"config": map[string]any{
			"name":    "lifecycle-test-pipeline",
			"enabled": true,
			"steps": []map[string]any{
				{
					"name":   "read-csv",
					"plugin": "Input.csv",
					"config": map[string]any{
						"file_path":   csvFile,
						"has_headers": true,
					},
					"output": "data",
				},
			},
		},
	}

	body, _ := json.Marshal(pipelineDef)
	req := httptest.NewRequest("POST", "/api/v1/pipelines", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)
	require.True(t, w.Code == http.StatusOK || w.Code == http.StatusCreated, "Expected 200 or 201, got %d", w.Code)

	var createResponse map[string]any
	json.Unmarshal(w.Body.Bytes(), &createResponse)
	pipeline, _ := createResponse["pipeline"].(map[string]any)
	pipelineID := pipeline["id"].(string)

	// Test job lifecycle
	t.Run("Create job", func(t *testing.T) {
		createReq := map[string]any{
			"id":        "lifecycle-job",
			"name":      "Lifecycle Test Job",
			"pipeline":  pipelineID,
			"cron_expr": "0 0 * * *",
		}

		body, _ := json.Marshal(createReq)
		req := httptest.NewRequest("POST", "/api/v1/scheduler/jobs", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusCreated, "Expected 200 or 201, got %d", w.Code)
	})

	t.Run("Disable job", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/scheduler/jobs/lifecycle-job/disable", nil)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		// Verify job is disabled
		req = httptest.NewRequest("GET", "/api/v1/scheduler/jobs/lifecycle-job", nil)
		w = httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		var job map[string]any
		json.Unmarshal(w.Body.Bytes(), &job)
		assert.False(t, job["enabled"].(bool))
	})

	t.Run("Enable job", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/scheduler/jobs/lifecycle-job/enable", nil)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		// Verify job is enabled
		req = httptest.NewRequest("GET", "/api/v1/scheduler/jobs/lifecycle-job", nil)
		w = httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		var job map[string]any
		json.Unmarshal(w.Body.Bytes(), &job)
		assert.True(t, job["enabled"].(bool))
	})

	t.Run("Update job", func(t *testing.T) {
		updateReq := map[string]any{
			"cron_expr": "*/15 * * * *",
		}

		body, _ := json.Marshal(updateReq)
		req := httptest.NewRequest("PUT", "/api/v1/scheduler/jobs/lifecycle-job", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		// Verify job was updated
		req = httptest.NewRequest("GET", "/api/v1/scheduler/jobs/lifecycle-job", nil)
		w = httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		var job map[string]any
		json.Unmarshal(w.Body.Bytes(), &job)
		assert.Equal(t, "*/15 * * * *", job["cron_expr"])
	})

	t.Run("Delete job", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/v1/scheduler/jobs/lifecycle-job", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusNoContent)

		// Verify job is deleted
		req = httptest.NewRequest("GET", "/api/v1/scheduler/jobs/lifecycle-job", nil)
		w = httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}
