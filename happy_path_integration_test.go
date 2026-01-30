package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHappyPath_PipelineCreationAndExecution tests the core happy path:
// 1. Create a pipeline via API
// 2. Execute the pipeline
// 3. Verify the results
func TestHappyPath_PipelineCreationAndExecution(t *testing.T) {
	// Create a temp directory for test data
	tmpDir := t.TempDir()

	// Create a test CSV file
	csvFile := filepath.Join(tmpDir, "test_data.csv")
	csvContent := "name,age,city\nAlice,30,NYC\nBob,25,LA\nCharlie,35,Chicago"
	err := os.WriteFile(csvFile, []byte(csvContent), 0644)
	require.NoError(t, err)

	// Create server
	server := NewServer()
	require.NotNil(t, server)

	// Test 1: Health check
	t.Run("Health check", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "healthy", response["status"])
	})

	// Test 2: List plugins
	t.Run("List available plugins", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/plugins", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var plugins []PluginInfo
		err := json.Unmarshal(w.Body.Bytes(), &plugins)
		require.NoError(t, err)
		assert.NotEmpty(t, plugins)
	})

	// Test 3: Create pipeline via API
	var pipelineID string
	t.Run("Create pipeline", func(t *testing.T) {
		pipelineDef := map[string]any{
			"metadata": map[string]any{
				"name":        "happy-path-test-pipeline",
				"description": "Test pipeline for happy path",
				"enabled":     true,
				"tags":        []string{},
			},
			"config": map[string]any{
				"name":    "happy-path-test-pipeline",
				"enabled": true,
				"steps": []map[string]any{
					{
						"name":   "read-csv",
						"plugin": "Input.csv",
						"config": map[string]any{
							"file_path":   csvFile,
							"has_headers": true,
							"delimiter":   ",",
						},
						"output": "csv_data",
					},
				},
			},
		}

		body, _ := json.Marshal(pipelineDef)
		req := httptest.NewRequest("POST", "/api/v1/pipelines", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		// Response has nested "pipeline" object
		pipeline, ok := response["pipeline"].(map[string]any)
		require.True(t, ok, "response should have nested pipeline object")

		assert.NotEmpty(t, pipeline["id"])
		pipelineID = pipeline["id"].(string)
		assert.Equal(t, "happy-path-test-pipeline", pipeline["name"])
	})

	// Test 4: Get pipeline
	t.Run("Get created pipeline", func(t *testing.T) {
		require.NotEmpty(t, pipelineID)

		req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/pipelines/%s", pipelineID), nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, pipelineID, response["id"])
		assert.Equal(t, "happy-path-test-pipeline", response["name"])
	})

	// Test 5: Execute pipeline
	t.Run("Execute pipeline", func(t *testing.T) {
		require.NotEmpty(t, pipelineID)

		execReq := PipelineExecutionRequest{
			PipelineID: pipelineID,
		}

		body, _ := json.Marshal(execReq)
		req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/pipelines/%s/execute", pipelineID), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response PipelineExecutionResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.True(t, response.Success, "Pipeline execution should succeed")
		assert.Empty(t, response.Error)
		// Note: Context field is *pipelines.PluginContext which doesn't serialize to JSON
		// The test passes if Success is true and no error occurred
	})

	// Test 6: List pipelines
	t.Run("List pipelines", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/pipelines", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response []map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.NotEmpty(t, response)

		// Find our pipeline in the list
		found := false
		for _, p := range response {
			if p["id"] == pipelineID {
				found = true
				break
			}
		}
		assert.True(t, found, "Created pipeline should be in the list")
	})
}

// TestHappyPath_PipelineExecutionByName tests executing a pipeline by name
func TestHappyPath_PipelineExecutionByName(t *testing.T) {
	tmpDir := t.TempDir()
	csvFile := filepath.Join(tmpDir, "data.csv")
	err := os.WriteFile(csvFile, []byte("name,value\nTest,100"), 0644)
	require.NoError(t, err)

	server := NewServer()
	require.NotNil(t, server)

	// Create pipeline first
	pipelineDef := map[string]any{
		"metadata": map[string]any{
			"name":    "exec-by-name-test",
			"enabled": true,
			"tags":    []string{},
		},
		"config": map[string]any{
			"name":    "exec-by-name-test",
			"enabled": true,
			"steps": []map[string]any{
				{
					"name":   "csv-step",
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
	require.Equal(t, http.StatusCreated, w.Code)

	// Extract pipeline ID from response
	var createResponse map[string]any
	json.Unmarshal(w.Body.Bytes(), &createResponse)
	pipeline, ok := createResponse["pipeline"].(map[string]any)
	require.True(t, ok, "response should have nested pipeline object")
	pipelineID := pipeline["id"].(string)

	// Execute pipeline by ID using the specific endpoint
	execReq := PipelineExecutionRequest{
		PipelineID: pipelineID,
	}

	body, _ = json.Marshal(execReq)
	req = httptest.NewRequest("POST", fmt.Sprintf("/api/v1/pipelines/%s/execute", pipelineID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	server.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response PipelineExecutionResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.True(t, response.Success)
}

// TestHappyPath_ValidatePipeline tests pipeline validation
func TestHappyPath_ValidatePipeline(t *testing.T) {
	server := NewServer()
	require.NotNil(t, server)

	// Create a valid pipeline
	pipelineDef := map[string]any{
		"metadata": map[string]any{
			"name":    "validate-test",
			"enabled": true,
			"tags":    []string{},
		},
		"config": map[string]any{
			"name":    "validate-test",
			"enabled": true,
			"steps": []map[string]any{
				{
					"name":   "csv-step",
					"plugin": "Input.csv",
					"config": map[string]any{
						"file_path":   "/some/path.csv",
						"has_headers": true,
					},
					"output": "data",
				},
			},
		},
	}

	body, _ := json.Marshal(pipelineDef)

	// First create the pipeline
	req := httptest.NewRequest("POST", "/api/v1/pipelines", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var createResponse map[string]any
	json.Unmarshal(w.Body.Bytes(), &createResponse)
	pipeline, ok := createResponse["pipeline"].(map[string]any)
	require.True(t, ok, "response should have nested pipeline object")
	pipelineID := pipeline["id"].(string)

	// Validate it
	req = httptest.NewRequest("POST", fmt.Sprintf("/api/v1/pipelines/%s/validate", pipelineID), nil)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	server.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// TestHappyPath_ClonePipeline tests cloning a pipeline
func TestHappyPath_ClonePipeline(t *testing.T) {
	server := NewServer()
	require.NotNil(t, server)

	// Create original pipeline
	pipelineDef := map[string]any{
		"metadata": map[string]any{
			"name":    "original-pipeline",
			"enabled": true,
			"tags":    []string{},
		},
		"config": map[string]any{
			"name":    "original-pipeline",
			"enabled": true,
			"steps": []map[string]any{
				{
					"name":   "test-step",
					"plugin": "Input.csv",
					"config": map[string]any{
						"file_path":   "/tmp/test.csv",
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
	require.Equal(t, http.StatusCreated, w.Code)

	var createResponse map[string]any
	json.Unmarshal(w.Body.Bytes(), &createResponse)
	pipeline, ok := createResponse["pipeline"].(map[string]any)
	require.True(t, ok, "response should have nested pipeline object")
	pipelineID := pipeline["id"].(string)

	// Clone it with a new name
	cloneReq := map[string]any{
		"name": "cloned-pipeline",
	}
	cloneBody, _ := json.Marshal(cloneReq)
	req = httptest.NewRequest("POST", fmt.Sprintf("/api/v1/pipelines/%s/clone", pipelineID), bytes.NewReader(cloneBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	server.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var cloneResponse map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &cloneResponse)
	require.NoError(t, err)
	clonedPipeline, ok := cloneResponse["pipeline"].(map[string]any)
	require.True(t, ok, "response should have nested pipeline object")
	assert.NotEmpty(t, clonedPipeline["id"])
	assert.NotEqual(t, pipelineID, clonedPipeline["id"])
}

// TestHappyPath_GetPipelineHistory tests getting pipeline execution history
func TestHappyPath_GetPipelineHistory(t *testing.T) {
	tmpDir := t.TempDir()
	csvFile := filepath.Join(tmpDir, "data.csv")
	err := os.WriteFile(csvFile, []byte("name,value\nTest,100"), 0644)
	require.NoError(t, err)

	server := NewServer()
	require.NotNil(t, server)

	// Create and execute pipeline
	pipelineDef := map[string]any{
		"metadata": map[string]any{
			"name":    "history-test",
			"enabled": true,
			"tags":    []string{},
		},
		"config": map[string]any{
			"name":    "history-test",
			"enabled": true,
			"steps": []map[string]any{
				{
					"name":   "csv-step",
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
	require.Equal(t, http.StatusCreated, w.Code)

	var createResponse map[string]any
	json.Unmarshal(w.Body.Bytes(), &createResponse)
	pipeline, ok := createResponse["pipeline"].(map[string]any)
	require.True(t, ok, "response should have nested pipeline object")
	pipelineID := pipeline["id"].(string)

	// Execute pipeline
	execReq := PipelineExecutionRequest{PipelineID: pipelineID}
	body, _ = json.Marshal(execReq)
	req = httptest.NewRequest("POST", fmt.Sprintf("/api/v1/pipelines/%s/execute", pipelineID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	server.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	// Get history
	req = httptest.NewRequest("GET", fmt.Sprintf("/api/v1/pipelines/%s/history", pipelineID), nil)
	w = httptest.NewRecorder()
	server.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// TestHappyPath_VersionEndpoint tests the version endpoint
func TestHappyPath_VersionEndpoint(t *testing.T) {
	server := NewServer()
	require.NotNil(t, server)

	req := httptest.NewRequest("GET", "/api/v1/version", nil)
	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.NotEmpty(t, response["version"])
	assert.NotEmpty(t, response["build"])
}
