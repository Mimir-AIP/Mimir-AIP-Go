package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
	"github.com/Mimir-AIP/Mimir-AIP-Go/utils"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// IntegrationTestSuite represents the complete test environment
type IntegrationTestSuite struct {
	server        *Server
	testServer    *httptest.Server
	tempDir       string
	registry      *pipelines.PluginRegistry
	scheduler     *utils.Scheduler
	monitor       *utils.JobMonitor
	configManager *utils.ConfigManager
}

// SetupIntegrationTestSuite creates a complete test environment
func SetupIntegrationTestSuite(t *testing.T) *IntegrationTestSuite {
	// Create temporary directory for test data
	tempDir, err := os.MkdirTemp("", "mimir_integration_test")
	require.NoError(t, err)

	// Initialize pipeline store with temp directory
	err = utils.InitializeGlobalPipelineStore(tempDir)
	require.NoError(t, err)

	// Create plugin registry
	registry := pipelines.NewPluginRegistry()

	// Register test plugins
	apiPlugin := &utils.RealAPIPlugin{}
	htmlPlugin := &utils.MockHTMLPlugin{}

	err = registry.RegisterPlugin(apiPlugin)
	require.NoError(t, err)
	err = registry.RegisterPlugin(htmlPlugin)
	require.NoError(t, err)

	// Create scheduler
	scheduler := utils.NewScheduler(registry)

	// Create job monitor
	monitor := utils.NewJobMonitor(100)

	// Create config manager
	configManager := utils.GetConfigManager()

	// Create server instance
	server := &Server{
		router:    mux.NewRouter(),
		registry:  registry,
		mcpServer: NewMCPServer(registry),
		scheduler: scheduler,
		monitor:   monitor,
		config:    configManager,
	}

	// Setup routes
	server.setupRoutes()

	// Create test server
	testServer := httptest.NewServer(server.router)

	suite := &IntegrationTestSuite{
		server:        server,
		testServer:    testServer,
		tempDir:       tempDir,
		registry:      registry,
		scheduler:     scheduler,
		monitor:       monitor,
		configManager: configManager,
	}

	// Start scheduler
	err = scheduler.Start()
	require.NoError(t, err)

	return suite
}

// Cleanup cleans up the test environment
func (suite *IntegrationTestSuite) Cleanup() {
	if suite.testServer != nil {
		suite.testServer.Close()
	}
	if suite.scheduler != nil {
		suite.scheduler.Stop()
	}
	if suite.tempDir != "" {
		os.RemoveAll(suite.tempDir)
	}
}

// TestHTTPAPIIntegration tests complete HTTP API workflows
func TestHTTPAPIIntegration(t *testing.T) {
	suite := SetupIntegrationTestSuite(t)
	defer suite.Cleanup()

	t.Run("Complete Pipeline Execution Workflow", func(t *testing.T) {
		testCompletePipelineWorkflow(t, suite)
	})

	t.Run("Authentication and Authorization Flow", func(t *testing.T) {
		testAuthenticationFlow(t, suite)
	})

	t.Run("Error Handling and Recovery", func(t *testing.T) {
		testErrorHandling(t, suite)
	})

	t.Run("Concurrent Request Handling", func(t *testing.T) {
		testConcurrentRequests(t, suite)
	})

	t.Run("Middleware Chain Integration", func(t *testing.T) {
		testMiddlewareChain(t, suite)
	})
}

// testCompletePipelineWorkflow tests a complete pipeline execution from API call to result
func testCompletePipelineWorkflow(t *testing.T, suite *IntegrationTestSuite) {
	// Create a test pipeline file
	pipelineContent := `
name: "Integration Test Pipeline"
description: "A comprehensive test pipeline for integration testing"
steps:
  - name: "Fetch Test Data"
    plugin: "Input.api"
    config:
      url: "https://httpbin.org/json"
      method: "GET"
      headers:
        User-Agent: "Mimir-AIP-Integration-Test"
    output: "api_data"
  
  - name: "Process Data"
    plugin: "Output.html"
    config:
      title: "Integration Test Report"
      template: "standard"
    output: "html_report"
`

	pipelineFile := filepath.Join(suite.tempDir, "integration_test_pipeline.yaml")
	err := os.WriteFile(pipelineFile, []byte(pipelineContent), 0644)
	require.NoError(t, err)

	// Step 1: Execute the pipeline
	execReq := PipelineExecutionRequest{
		PipelineFile: pipelineFile,
		Context: map[string]interface{}{
			"test_id":   "integration_test_001",
			"test_run":  time.Now().Format(time.RFC3339),
			"test_mode": "integration",
		},
	}

	reqBody, _ := json.Marshal(execReq)
	resp, err := http.Post(
		suite.testServer.URL+"/api/v1/pipelines/execute",
		"application/json",
		bytes.NewBuffer(reqBody),
	)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var execResp PipelineExecutionResponse
	err = json.NewDecoder(resp.Body).Decode(&execResp)
	require.NoError(t, err)

	require.True(t, execResp.Success, "Pipeline execution should succeed")
	assert.NotEmpty(t, execResp.ExecutedAt)
	assert.NotNil(t, execResp.Context)

	// Step 2: Verify job was recorded in monitor
	jobListResp, err := http.Get(suite.testServer.URL + "/api/v1/jobs")
	require.NoError(t, err)
	defer jobListResp.Body.Close()

	require.Equal(t, http.StatusOK, jobListResp.StatusCode)

	var jobs []map[string]interface{}
	err = json.NewDecoder(jobListResp.Body).Decode(&jobs)
	require.NoError(t, err)

	assert.Greater(t, len(jobs), 0, "At least one job should be recorded")

	// Step 3: Check performance metrics
	metricsResp, err := http.Get(suite.testServer.URL + "/api/v1/performance/metrics")
	require.NoError(t, err)
	defer metricsResp.Body.Close()

	require.Equal(t, http.StatusOK, metricsResp.StatusCode)

	var metrics map[string]interface{}
	err = json.NewDecoder(metricsResp.Body).Decode(&metrics)
	require.NoError(t, err)

	assert.Contains(t, metrics, "total_requests")
	assert.Contains(t, metrics, "average_latency")
	assert.Contains(t, metrics, "requests_per_second")

	// Step 4: Verify pipeline is listed
	pipelinesResp, err := http.Get(suite.testServer.URL + "/api/v1/pipelines")
	require.NoError(t, err)
	defer pipelinesResp.Body.Close()

	require.Equal(t, http.StatusOK, pipelinesResp.StatusCode)

	var pipelines []utils.PipelineConfig
	err = json.NewDecoder(pipelinesResp.Body).Decode(&pipelines)
	require.NoError(t, err)

	// Note: This might be empty if config.yaml doesn't exist, which is fine for integration test
	assert.NotNil(t, pipelines)
}

// testAuthenticationFlow tests authentication and authorization mechanisms
func testAuthenticationFlow(t *testing.T, suite *IntegrationTestSuite) {
	// Test health endpoint (no auth required)
	resp, err := http.Get(suite.testServer.URL + "/health")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// Test protected endpoint without auth (should fail if auth is enabled)
	// For now, we'll test the structure exists
	protectedReq, _ := http.NewRequest(
		"POST",
		suite.testServer.URL+"/protected/pipelines",
		bytes.NewBuffer([]byte(`{"pipeline_name": "test"}`)),
	)
	protectedReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err = client.Do(protectedReq)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Response should indicate authentication requirement (401 or 403)
	assert.True(t, resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden)

	// Test authentication endpoints exist
	authEndpoints := []string{
		"/api/v1/auth/login",
		"/api/v1/auth/refresh",
		"/api/v1/auth/me",
	}

	for _, endpoint := range authEndpoints {
		req, _ := http.NewRequest("GET", suite.testServer.URL+endpoint, nil)
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should not be 404 (endpoint exists)
		assert.NotEqual(t, http.StatusNotFound, resp.StatusCode)
	}
}

// testErrorHandling tests error scenarios and recovery
func testErrorHandling(t *testing.T, suite *IntegrationTestSuite) {
	t.Run("Invalid Pipeline Execution", func(t *testing.T) {
		// Test with non-existent pipeline
		invalidReq := PipelineExecutionRequest{
			PipelineFile: "/non/existent/pipeline.yaml",
		}

		reqBody, _ := json.Marshal(invalidReq)
		resp, err := http.Post(
			suite.testServer.URL+"/api/v1/pipelines/execute",
			"application/json",
			bytes.NewBuffer(reqBody),
		)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		var errorResp map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&errorResp)
		require.NoError(t, err)

		assert.Contains(t, errorResp, "error")
	})

	t.Run("Invalid Request Body", func(t *testing.T) {
		resp, err := http.Post(
			suite.testServer.URL+"/api/v1/pipelines/execute",
			"application/json",
			bytes.NewBuffer([]byte("invalid json")),
		)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("Non-existent Endpoints", func(t *testing.T) {
		resp, err := http.Get(suite.testServer.URL + "/api/v1/nonexistent")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}

// testConcurrentRequests tests the system under concurrent load
func testConcurrentRequests(t *testing.T, suite *IntegrationTestSuite) {
	const numRequests = 20
	const numWorkers = 5

	// Create a simple test pipeline
	pipelineContent := `
name: "Concurrent Test Pipeline"
steps:
  - name: "Simple API Call"
    plugin: "Input.api"
    config:
      url: "https://httpbin.org/delay/1"
      method: "GET"
    output: "result"
`

	pipelineFile := filepath.Join(suite.tempDir, "concurrent_test_pipeline.yaml")
	err := os.WriteFile(pipelineFile, []byte(pipelineContent), 0644)
	require.NoError(t, err)

	var wg sync.WaitGroup
	results := make(chan error, numRequests)
	startTime := time.Now()

	// Launch workers
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < numRequests/numWorkers; j++ {
				err := executePipelineRequest(suite.testServer.URL, pipelineFile, workerID, j)
				results <- err
			}
		}(i)
	}

	// Wait for all workers to complete
	wg.Wait()
	close(results)

	// Check results
	successCount := 0
	errorCount := 0
	for err := range results {
		if err != nil {
			errorCount++
			t.Logf("Request failed: %v", err)
		} else {
			successCount++
		}
	}

	duration := time.Since(startTime)
	t.Logf("Concurrent test completed in %v", duration)
	t.Logf("Successful requests: %d, Failed requests: %d", successCount, errorCount)

	// Most requests should succeed
	assert.Greater(t, successCount, numRequests*8/10, "At least 80% of requests should succeed")
}

// executePipelineRequest is a helper function for concurrent testing
func executePipelineRequest(baseURL, pipelineFile string, workerID, requestID int) error {
	execReq := PipelineExecutionRequest{
		PipelineFile: pipelineFile,
		Context: map[string]interface{}{
			"worker_id":  workerID,
			"request_id": requestID,
		},
	}

	reqBody, _ := json.Marshal(execReq)

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Post(
		baseURL+"/api/v1/pipelines/execute",
		"application/json",
		bytes.NewBuffer(reqBody),
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// testMiddlewareChain tests that all middleware components work together
func testMiddlewareChain(t *testing.T, suite *IntegrationTestSuite) {
	// Test CORS headers
	req, _ := http.NewRequest("OPTIONS", suite.testServer.URL+"/api/v1/pipelines", nil)
	req.Header.Set("Origin", "http://localhost:3000")

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Check CORS headers
	corsHeaders := []string{
		"Access-Control-Allow-Origin",
		"Access-Control-Allow-Methods",
		"Access-Control-Allow-Headers",
	}

	for _, header := range corsHeaders {
		assert.NotEmpty(t, resp.Header.Get(header), "CORS header %s should be present", header)
	}

	// Test API versioning
	resp, err = http.Get(suite.testServer.URL + "/api/v1/pipelines")
	require.NoError(t, err)
	defer resp.Body.Close()

	versionHeader := resp.Header.Get("X-API-Version")
	assert.Equal(t, "v1", versionHeader)

	// Test security headers
	securityHeaders := []string{
		"X-Content-Type-Options",
		"X-Frame-Options",
		"X-XSS-Protection",
	}

	for _, header := range securityHeaders {
		assert.NotEmpty(t, resp.Header.Get(header), "Security header %s should be present", header)
	}
}

// TestPipelineIntegration tests multi-step pipeline execution with real plugins
func TestPipelineIntegration(t *testing.T) {
	suite := SetupIntegrationTestSuite(t)
	defer suite.Cleanup()

	t.Run("Multi-Step Pipeline Execution", func(t *testing.T) {
		testMultiStepPipeline(t, suite)
	})

	t.Run("Context Passing Between Steps", func(t *testing.T) {
		testContextPassing(t, suite)
	})

	t.Run("Pipeline Error Recovery", func(t *testing.T) {
		testPipelineErrorRecovery(t, suite)
	})

	t.Run("Pipeline Performance", func(t *testing.T) {
		testPipelinePerformance(t, suite)
	})
}

// testMultiStepPipeline tests execution of pipelines with multiple steps
func testMultiStepPipeline(t *testing.T, suite *IntegrationTestSuite) {
	// Create a complex multi-step pipeline
	pipelineContent := `
name: "Multi-Step Integration Test"
description: "Tests complex pipeline with multiple steps and data flow"
steps:
  - name: "Fetch User Data"
    plugin: "Input.api"
    config:
      url: "https://httpbin.org/json"
      method: "GET"
      headers:
        Accept: "application/json"
    output: "user_data"
  
  - name: "Fetch Additional Info"
    plugin: "Input.api"
    config:
      url: "https://httpbin.org/uuid"
      method: "GET"
    output: "additional_info"
  
  - name: "Generate Final Report"
    plugin: "Output.html"
    config:
      title: "Multi-Step Test Report"
      include_metadata: true
    output: "final_report"
`

	pipelineFile := filepath.Join(suite.tempDir, "multi_step_test.yaml")
	err := os.WriteFile(pipelineFile, []byte(pipelineContent), 0644)
	require.NoError(t, err)

	// Execute the pipeline
	config, err := utils.ParsePipeline(pipelineFile)
	require.NoError(t, err)

	ctx := context.Background()
	result, err := utils.ExecutePipelineWithRegistry(ctx, config, suite.registry)
	require.NoError(t, err)

	require.True(t, result.Success, "Multi-step pipeline should succeed")
	assert.NotNil(t, result.Context)
	assert.NotEmpty(t, result.ExecutedAt)

	// Verify that all steps executed and context contains expected data
	expectedOutputs := []string{"user_data", "additional_info", "final_report"}
	for _, output := range expectedOutputs {
		value, exists := result.Context.Get(output)
		assert.True(t, exists, "Output %s should exist in context", output)
		assert.NotNil(t, value, "Output %s should not be nil", output)
	}
}

// testContextPassing tests that context is properly passed between pipeline steps
func testContextPassing(t *testing.T, suite *IntegrationTestSuite) {
	// Create pipeline that explicitly tests context passing
	pipelineContent := `
name: "Context Passing Test"
description: "Tests context passing between pipeline steps"
steps:
  - name: "Initial Data"
    plugin: "Input.api"
    config:
      url: "https://httpbin.org/json"
      method: "GET"
    output: "initial_data"
  
  - name: "Process Data"
    plugin: "Output.html"
    config:
      title: "Context Test Report"
      use_context: true
    output: "processed_data"
`

	pipelineFile := filepath.Join(suite.tempDir, "context_test.yaml")
	err := os.WriteFile(pipelineFile, []byte(pipelineContent), 0644)
	require.NoError(t, err)

	// Execute with initial context
	initialContext := map[string]interface{}{
		"test_id":        "context_test_001",
		"test_timestamp": time.Now().Unix(),
		"test_metadata": map[string]interface{}{
			"version": "1.0",
			"mode":    "test",
		},
	}

	config, err := utils.ParsePipeline(pipelineFile)
	require.NoError(t, err)

	ctx := context.Background()
	globalContext := pipelines.NewPluginContext()
	for k, v := range initialContext {
		globalContext.Set(k, v)
	}

	result, err := utils.ExecutePipelineWithRegistry(ctx, config, suite.registry)
	require.NoError(t, err)

	require.True(t, result.Success)

	// Verify initial context is preserved
	for key, expectedValue := range initialContext {
		actualValue, exists := result.Context.Get(key)
		assert.True(t, exists, "Initial context key %s should be preserved", key)
		assert.Equal(t, expectedValue, actualValue, "Initial context value for %s should match", key)
	}

	// Verify step outputs are added
	stepOutputs := []string{"initial_data", "processed_data"}
	for _, output := range stepOutputs {
		_, exists := result.Context.Get(output)
		assert.True(t, exists, "Step output %s should be added to context", output)
	}
}

// testPipelineErrorRecovery tests error handling in pipeline execution
func testPipelineErrorRecovery(t *testing.T, suite *IntegrationTestSuite) {
	// Create pipeline with intentional error
	pipelineContent := `
name: "Error Recovery Test"
description: "Tests error handling in pipeline execution"
steps:
  - name: "Valid Step"
    plugin: "Input.api"
    config:
      url: "https://httpbin.org/json"
      method: "GET"
    output: "valid_result"
  
  - name: "Invalid Step"
    plugin: "Input.api"
    config:
      url: "invalid-url"
      method: "GET"
    output: "invalid_result"
  
  - name: "Recovery Step"
    plugin: "Output.html"
    config:
      title: "Error Recovery Report"
    output: "recovery_result"
`

	pipelineFile := filepath.Join(suite.tempDir, "error_test.yaml")
	err := os.WriteFile(pipelineFile, []byte(pipelineContent), 0644)
	require.NoError(t, err)

	config, err := utils.ParsePipeline(pipelineFile)
	require.NoError(t, err)

	ctx := context.Background()
	result, err := utils.ExecutePipelineWithRegistry(ctx, config, suite.registry)
	require.NoError(t, err)

	// Pipeline should fail due to invalid step
	require.False(t, result.Success, "Pipeline with invalid step should fail")
	assert.NotEmpty(t, result.Error, "Error message should be provided")
	assert.Contains(t, result.Error, "Invalid Step", "Error should mention the failing step")

	// Context should contain results from successful steps before failure
	validResult, exists := result.Context.Get("valid_result")
	assert.True(t, exists, "Results from successful steps should be preserved")
	assert.NotNil(t, validResult, "Valid result should not be nil")

	// Results from failed and subsequent steps should not exist
	_, exists = result.Context.Get("invalid_result")
	assert.False(t, exists, "Results from failed steps should not exist")

	_, exists = result.Context.Get("recovery_result")
	assert.False(t, exists, "Results from steps after failure should not exist")
}

// testPipelinePerformance tests pipeline execution performance
func testPipelinePerformance(t *testing.T, suite *IntegrationTestSuite) {
	// Create performance test pipeline
	pipelineContent := `
name: "Performance Test Pipeline"
description: "Tests pipeline execution performance"
steps:
  - name: "Performance Test Step"
    plugin: "Input.api"
    config:
      url: "https://httpbin.org/json"
      method: "GET"
    output: "perf_result"
`

	pipelineFile := filepath.Join(suite.tempDir, "performance_test.yaml")
	err := os.WriteFile(pipelineFile, []byte(pipelineContent), 0644)
	require.NoError(t, err)

	config, err := utils.ParsePipeline(pipelineFile)
	require.NoError(t, err)

	const numExecutions = 10
	var durations []time.Duration

	for i := 0; i < numExecutions; i++ {
		start := time.Now()

		ctx := context.Background()
		result, err := utils.ExecutePipelineWithRegistry(ctx, config, suite.registry)

		duration := time.Since(start)
		durations = append(durations, duration)

		require.NoError(t, err)
		require.True(t, result.Success, "Pipeline execution %d should succeed", i)
	}

	// Calculate performance statistics
	var totalDuration time.Duration
	minDuration := durations[0]
	maxDuration := durations[0]

	for _, duration := range durations {
		totalDuration += duration
		if duration < minDuration {
			minDuration = duration
		}
		if duration > maxDuration {
			maxDuration = duration
		}
	}

	avgDuration := totalDuration / time.Duration(numExecutions)

	t.Logf("Performance Test Results:")
	t.Logf("  Executions: %d", numExecutions)
	t.Logf("  Average: %v", avgDuration)
	t.Logf("  Min: %v", minDuration)
	t.Logf("  Max: %v", maxDuration)

	// Performance assertions (adjust based on expected performance)
	assert.Less(t, avgDuration, 5*time.Second, "Average execution time should be less than 5 seconds")
	assert.Less(t, maxDuration-minDuration, 3*time.Second, "Execution time variance should be reasonable")
}
