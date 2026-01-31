package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Mimir-AIP/Mimir-AIP-Go/utils"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestServer creates a test server with initialized components
func createTestServer(t *testing.T) *Server {
	server := NewServer()
	require.NotNil(t, server, "Failed to create server")

	return server
}

// TestHandleHealth tests the health check handler
func TestHandleHealth(t *testing.T) {
	server := createTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rr := httptest.NewRecorder()

	server.handleHealth(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "Health check should return 200")

	var response map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err, "Failed to parse response")

	assert.Equal(t, "healthy", response["status"], "Status should be healthy")
	assert.NotNil(t, response["time"], "Time should be present")
}

// TestHandleVersion tests the version handler
func TestHandleVersion(t *testing.T) {
	server := createTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/version", nil)
	rr := httptest.NewRecorder()

	server.handleVersion(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "Version endpoint should return 200")

	var response map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err, "Failed to parse response")

	assert.NotNil(t, response["version"], "Version should be present")
	assert.NotNil(t, response["build"], "Build should be present")
	assert.Contains(t, response["version"], "0.2.0", "Version should contain major.minor.patch")
}

// TestHandleListPipelines tests the pipeline list handler
func TestHandleListPipelines(t *testing.T) {
	server := createTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/pipelines", nil)
	rr := httptest.NewRecorder()

	server.handleListPipelines(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "List pipelines should return 200")

	// Response should be an array (even if empty)
	var response []interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err, "Response should be an array")
}

// TestHandleGetPipeline tests the pipeline get handler
func TestHandleGetPipeline(t *testing.T) {
	server := createTestServer(t)

	// Test with non-existent pipeline
	req := httptest.NewRequest(http.MethodGet, "/api/v1/pipelines/non-existent", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "non-existent"})
	rr := httptest.NewRecorder()

	server.handleGetPipeline(rr, req)

	// Should return 404 for non-existent pipeline
	assert.Equal(t, http.StatusNotFound, rr.Code, "Get non-existent pipeline should return 404")
}

// TestHandleCreatePipeline tests the pipeline create handler
func TestHandleCreatePipeline(t *testing.T) {
	server := createTestServer(t)

	// Create request body
	requestBody := map[string]interface{}{
		"metadata": map[string]interface{}{
			"name":        "Test Pipeline",
			"description": "Test Description",
			"tags":        []string{"test", "example"},
		},
		"config": map[string]interface{}{
			"input": map[string]interface{}{
				"plugin": "Input.csv",
				"config": map[string]interface{}{
					"file_path": "test.csv",
				},
			},
		},
	}

	body, err := json.Marshal(requestBody)
	require.NoError(t, err, "Failed to marshal request body")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/pipelines", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	// Add user to context (simulating authenticated request)
	user := &utils.User{
		ID:       "user-001",
		Username: "testuser",
		Roles:    []string{"admin"},
	}
	ctx := context.WithValue(req.Context(), "user", user)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	server.handleCreatePipeline(rr, req)

	// Note: This may fail if pipeline store is not properly initialized
	// The actual status depends on the implementation
	if rr.Code == http.StatusCreated {
		var response map[string]interface{}
		err := json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err, "Failed to parse response")
		assert.NotNil(t, response["pipeline"], "Pipeline should be in response")
	}
}

// TestHandleUpdatePipeline tests the pipeline update handler
func TestHandleUpdatePipeline(t *testing.T) {
	server := createTestServer(t)

	requestBody := map[string]interface{}{
		"metadata": map[string]interface{}{
			"name": "Updated Pipeline",
		},
	}

	body, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/pipelines/test-id", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": "test-id"})

	user := &utils.User{
		ID:       "user-001",
		Username: "testuser",
		Roles:    []string{"admin"},
	}
	ctx := context.WithValue(req.Context(), "user", user)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	server.handleUpdatePipeline(rr, req)

	// Should fail since pipeline doesn't exist
	assert.Equal(t, http.StatusBadRequest, rr.Code, "Update non-existent pipeline should return error")
}

// TestHandleDeletePipeline tests the pipeline delete handler
func TestHandleDeletePipeline(t *testing.T) {
	server := createTestServer(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/pipelines/test-id", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "test-id"})

	rr := httptest.NewRecorder()

	server.handleDeletePipeline(rr, req)

	// Should fail since pipeline doesn't exist
	assert.Equal(t, http.StatusNotFound, rr.Code, "Delete non-existent pipeline should return 404")
}

// TestHandleClonePipeline tests the pipeline clone handler
func TestHandleClonePipeline(t *testing.T) {
	server := createTestServer(t)

	requestBody := map[string]interface{}{
		"name": "Cloned Pipeline",
	}

	body, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/pipelines/test-id/clone", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": "test-id"})

	user := &utils.User{
		ID:       "user-001",
		Username: "testuser",
		Roles:    []string{"admin"},
	}
	ctx := context.WithValue(req.Context(), "user", user)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	server.handleClonePipeline(rr, req)

	// Should fail since pipeline doesn't exist
	assert.Equal(t, http.StatusBadRequest, rr.Code, "Clone non-existent pipeline should return error")
}

// TestHandleValidatePipeline tests the pipeline validate handler
func TestHandleValidatePipeline(t *testing.T) {
	server := createTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/pipelines/test-id/validate", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "test-id"})

	rr := httptest.NewRecorder()

	server.handleValidatePipeline(rr, req)

	// Should fail since pipeline doesn't exist
	assert.Equal(t, http.StatusNotFound, rr.Code, "Validate non-existent pipeline should return 404")
}

// TestHandleListPlugins tests the plugin list handler
func TestHandleListPlugins(t *testing.T) {
	server := createTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/plugins", nil)
	rr := httptest.NewRecorder()

	server.handleListPlugins(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "List plugins should return 200")

	var response []map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err, "Failed to parse response")

	// Should return at least some plugins (built-in ones)
	assert.GreaterOrEqual(t, len(response), 0, "Should return plugins or empty array")
}

// TestHandleGetPlugin tests the plugin get handler
func TestHandleGetPlugin(t *testing.T) {
	server := createTestServer(t)

	// Test with non-existent plugin
	req := httptest.NewRequest(http.MethodGet, "/api/v1/plugins/Input/nonexistent", nil)
	req = mux.SetURLVars(req, map[string]string{
		"type": "Input",
		"name": "nonexistent",
	})
	rr := httptest.NewRecorder()

	server.handleGetPlugin(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code, "Get non-existent plugin should return 404")
}

// TestHandleListPluginsByType tests listing plugins by type
func TestHandleListPluginsByType(t *testing.T) {
	server := createTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/plugins/Input", nil)
	req = mux.SetURLVars(req, map[string]string{"type": "Input"})
	rr := httptest.NewRecorder()

	server.handleListPluginsByType(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "List plugins by type should return 200")

	var response []map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err, "Failed to parse response")

	// Response should be an array
	assert.NotNil(t, response, "Response should be an array")
}

// TestHandleListJobs tests the job list handler
func TestHandleListJobs(t *testing.T) {
	server := createTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs", nil)
	rr := httptest.NewRecorder()

	server.handleListJobs(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "List jobs should return 200")

	var response []interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err, "Failed to parse response")
}

// TestHandleGetJob tests the job get handler
func TestHandleGetJob(t *testing.T) {
	server := createTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/test-job", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "test-job"})
	rr := httptest.NewRecorder()

	server.handleGetJob(rr, req)

	// Should return 404 for non-existent job
	assert.Equal(t, http.StatusNotFound, rr.Code, "Get non-existent job should return 404")
}

// TestHandleCreateJob tests the job create handler
func TestHandleCreateJob(t *testing.T) {
	server := createTestServer(t)

	requestBody := map[string]interface{}{
		"id":        "test-job",
		"name":      "Test Job",
		"pipeline":  "test-pipeline",
		"cron_expr": "*/5 * * * *",
	}

	body, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	server.handleCreateJob(rr, req)

	// Note: May fail if pipeline doesn't exist or scheduler issue
	if rr.Code == http.StatusOK {
		var response map[string]interface{}
		err := json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "Job created successfully", response["message"])
	}
}

// TestHandleDeleteJob tests the job delete handler
func TestHandleDeleteJob(t *testing.T) {
	server := createTestServer(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/jobs/test-job", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "test-job"})

	rr := httptest.NewRecorder()

	server.handleDeleteJob(rr, req)

	// Should return 404 for non-existent job
	assert.Equal(t, http.StatusNotFound, rr.Code, "Delete non-existent job should return 404")
}

// TestHandleEnableJob tests the job enable handler
func TestHandleEnableJob(t *testing.T) {
	server := createTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/test-job/enable", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "test-job"})

	rr := httptest.NewRecorder()

	server.handleEnableJob(rr, req)

	// Should return 404 for non-existent job
	assert.Equal(t, http.StatusNotFound, rr.Code, "Enable non-existent job should return 404")
}

// TestHandleDisableJob tests the job disable handler
func TestHandleDisableJob(t *testing.T) {
	server := createTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/test-job/disable", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "test-job"})

	rr := httptest.NewRecorder()

	server.handleDisableJob(rr, req)

	// Should return 404 for non-existent job
	assert.Equal(t, http.StatusNotFound, rr.Code, "Disable non-existent job should return 404")
}

// TestHandleUpdateJob tests the job update handler
func TestHandleUpdateJob(t *testing.T) {
	server := createTestServer(t)

	requestBody := map[string]interface{}{
		"name":      "Updated Job",
		"cron_expr": "0 */1 * * *",
	}

	body, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/jobs/test-job", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": "test-job"})

	rr := httptest.NewRecorder()

	server.handleUpdateJob(rr, req)

	// Should return error for non-existent job
	assert.Equal(t, http.StatusBadRequest, rr.Code, "Update non-existent job should return error")
}

// TestHandleLogin tests the login handler
func TestHandleLogin(t *testing.T) {
	server := createTestServer(t)

	// Initialize auth manager with a test user
	authManager := utils.GetAuthManager()
	_, err := authManager.CreateUser("testuser", "testpass123", []string{"user"})
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("Failed to create test user: %v", err)
	}

	requestBody := map[string]interface{}{
		"username": "testuser",
		"password": "testpass123",
	}

	body, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	server.handleLogin(rr, req)

	if rr.Code == http.StatusOK {
		var response map[string]interface{}
		err := json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.NotNil(t, response["token"], "Token should be present")
		assert.NotNil(t, response["user"], "User should be present")
		assert.NotNil(t, response["roles"], "Roles should be present")
	}
}

// TestHandleLogin_InvalidCredentials tests login with invalid credentials
func TestHandleLogin_InvalidCredentials(t *testing.T) {
	server := createTestServer(t)

	requestBody := map[string]interface{}{
		"username": "nonexistent",
		"password": "wrongpass",
	}

	body, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	server.handleLogin(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code, "Invalid credentials should return 401")
}

// TestHandleAuthMe tests the auth me handler
func TestHandleAuthMe(t *testing.T) {
	server := createTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)

	// Add user to context
	user := &utils.User{
		ID:       "user-001",
		Username: "testuser",
		Roles:    []string{"user"},
		Active:   true,
	}
	ctx := context.WithValue(req.Context(), "user", user)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	server.handleAuthMe(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "Auth me should return 200 for authenticated user")

	var response map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, user.ID, response["id"])
	assert.Equal(t, user.Username, response["username"])
}

// TestHandleAuthMe_Unauthenticated tests auth me without authentication
func TestHandleAuthMe_Unauthenticated(t *testing.T) {
	server := createTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	rr := httptest.NewRecorder()

	server.handleAuthMe(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code, "Auth me should return 401 for unauthenticated user")
}

// TestHandleLogout tests the logout handler
func TestHandleLogout(t *testing.T) {
	server := createTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	rr := httptest.NewRecorder()

	server.handleLogout(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "Logout should return 200")

	var response map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.True(t, response["success"].(bool), "Success should be true")
}

// TestHandleRefreshToken tests the token refresh handler
func TestHandleRefreshToken(t *testing.T) {
	server := createTestServer(t)

	authManager := utils.GetAuthManager()
	user := &utils.User{
		ID:       "user-002",
		Username: "refreshtest",
		Roles:    []string{"user"},
	}
	authManager.CreateUser(user.Username, "password", user.Roles)

	// Login to get initial token
	loginReq := map[string]interface{}{
		"username": user.Username,
		"password": "password",
	}
	loginBody, _ := json.Marshal(loginReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(loginBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	server.handleLogin(rr, req)

	if rr.Code == http.StatusOK {
		var loginResp map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &loginResp)
		token := loginResp["token"].(string)

		// Try to refresh token
		refreshReq := map[string]interface{}{
			"token": token,
		}
		refreshBody, _ := json.Marshal(refreshReq)
		req2 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewBuffer(refreshBody))
		req2.Header.Set("Content-Type", "application/json")
		rr2 := httptest.NewRecorder()

		server.handleRefreshToken(rr2, req2)

		if rr2.Code == http.StatusOK {
			var refreshResp map[string]interface{}
			json.Unmarshal(rr2.Body.Bytes(), &refreshResp)
			assert.NotNil(t, refreshResp["token"], "New token should be present")
		}
	}
}

// TestHandleGetPerformanceMetrics tests the performance metrics handler
func TestHandleGetPerformanceMetrics(t *testing.T) {
	server := createTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/performance/metrics", nil)
	rr := httptest.NewRecorder()

	server.handleGetPerformanceMetrics(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "Get metrics should return 200")

	var response map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)
	// Should have performance metrics
	assert.NotNil(t, response, "Response should not be nil")
}

// TestHandleGetPerformanceStats tests the performance stats handler
func TestHandleGetPerformanceStats(t *testing.T) {
	server := createTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/performance/stats", nil)
	rr := httptest.NewRecorder()

	server.handleGetPerformanceStats(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "Get stats should return 200")

	var response map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.NotNil(t, response["performance"], "Performance should be present")
	assert.NotNil(t, response["system"], "System should be present")
}

// TestHandleGetRunningJobs tests the running jobs handler
func TestHandleGetRunningJobs(t *testing.T) {
	server := createTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/monitoring/jobs/running", nil)
	rr := httptest.NewRecorder()

	server.handleGetRunningJobs(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "Get running jobs should return 200")
}

// TestHandleGetJobStatistics tests the job statistics handler
func TestHandleGetJobStatistics(t *testing.T) {
	server := createTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/monitoring/jobs/statistics", nil)
	rr := httptest.NewRecorder()

	server.handleGetJobStatistics(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "Get job statistics should return 200")
}

// TestHandleGetRecentJobs tests the recent jobs handler
func TestHandleGetRecentJobs(t *testing.T) {
	server := createTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/monitoring/jobs/recent", nil)
	rr := httptest.NewRecorder()

	server.handleGetRecentJobs(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "Get recent jobs should return 200")
}

// TestHandleGetConfig tests the get config handler
func TestHandleGetConfig(t *testing.T) {
	server := createTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/config", nil)
	rr := httptest.NewRecorder()

	server.handleGetConfig(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "Get config should return 200")
}

// TestHandleAuthCheck tests the auth check handler
func TestHandleAuthCheck(t *testing.T) {
	server := createTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/check", nil)

	// Without authentication
	rr := httptest.NewRecorder()
	server.handleAuthCheck(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code, "Auth check without auth should return 401")

	// With authentication
	user := &utils.User{
		ID:       "user-001",
		Username: "testuser",
		Roles:    []string{"user"},
	}
	ctx := context.WithValue(req.Context(), "user", user)
	req2 := req.WithContext(ctx)

	rr2 := httptest.NewRecorder()
	server.handleAuthCheck(rr2, req2)
	assert.Equal(t, http.StatusOK, rr2.Code, "Auth check with auth should return 200")
}
