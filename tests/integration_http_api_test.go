package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/Mimir-AIP/Mimir-AIP-Go/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHTTPAPICompleteWorkflow tests a complete end-to-end HTTP API workflow
func TestHTTPAPICompleteWorkflow(t *testing.T) {
	// Setup test environment
	tempDir, err := os.MkdirTemp("", "mimir_http_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Initialize pipeline store
	err = utils.InitializeGlobalPipelineStore(tempDir)
	require.NoError(t, err)

	// Create test server
	ms := NewMockServer()
	testServer := ms.Start()
	defer testServer.Close()

	t.Run("Complete Pipeline Execution Flow", func(t *testing.T) {
		// Create a test pipeline file
		pipelineContent := `
pipelines:
  - name: "HTTP Integration Test"
    enabled: true
    description: "Tests HTTP API integration"
    steps:
      - name: "Fetch Data"
        plugin: "Input.api"
        config:
          url: "https://httpbin.org/json"
          method: "GET"
        output: "api_data"
      - name: "Generate Report"
        plugin: "Output.html"
        config:
          title: "HTTP Test Report"
        output: "html_report"
`

		pipelineFile := filepath.Join(tempDir, "http_test.yaml")
		err := os.WriteFile(pipelineFile, []byte(pipelineContent), 0644)
		require.NoError(t, err)

		// Step 1: Verify health endpoint
		resp, err := http.Get(testServer.URL + "/health")
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var healthResp map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&healthResp)
		require.NoError(t, err)
		assert.Equal(t, "healthy", healthResp["status"])

		// Step 2: List available plugins
		resp, err = http.Get(testServer.URL + "/api/v1/plugins")
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var plugins []PluginInfo
		err = json.NewDecoder(resp.Body).Decode(&plugins)
		require.NoError(t, err)
		assert.Greater(t, len(plugins), 0, "Should have at least one plugin")

		// Step 3: Check performance metrics endpoint
		resp, err = http.Get(testServer.URL + "/api/v1/performance/metrics")
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var metrics map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&metrics)
		require.NoError(t, err)
		assert.Contains(t, metrics, "total_requests")
		assert.Contains(t, metrics, "average_latency")
	})
}

// TestHTTPAPIErrorHandling tests error handling in HTTP API
func TestHTTPAPIErrorHandling(t *testing.T) {
	ms := NewMockServer()
	testServer := ms.Start()
	defer testServer.Close()

	t.Run("Invalid JSON Request", func(t *testing.T) {
		resp, err := http.Post(
			testServer.URL+"/api/v1/pipelines/execute",
			"application/json",
			bytes.NewBuffer([]byte("invalid json {{")),
		)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should return bad request
		assert.True(t, resp.StatusCode >= 400 && resp.StatusCode < 500)
	})

	t.Run("Non-existent Endpoint", func(t *testing.T) {
		resp, err := http.Get(testServer.URL + "/api/v1/nonexistent/endpoint")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("Method Not Allowed", func(t *testing.T) {
		resp, err := http.Post(testServer.URL+"/health", "application/json", nil)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
	})
}

// TestHTTPAPIConcurrentRequests tests handling of concurrent requests
func TestHTTPAPIConcurrentRequests(t *testing.T) {
	ms := NewMockServer()
	testServer := ms.Start()
	defer testServer.Close()

	const numRequests = 50
	const numWorkers = 10

	var wg sync.WaitGroup
	results := make(chan int, numRequests)
	startTime := time.Now()

	// Make concurrent requests to health endpoint
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < numRequests/numWorkers; j++ {
				resp, err := http.Get(testServer.URL + "/health")
				if err == nil {
					results <- resp.StatusCode
					resp.Body.Close()
				} else {
					results <- 0
				}
			}
		}(i)
	}

	wg.Wait()
	close(results)

	duration := time.Since(startTime)
	t.Logf("Completed %d concurrent requests in %v", numRequests, duration)

	// Check results
	successCount := 0
	for statusCode := range results {
		if statusCode == http.StatusOK {
			successCount++
		}
	}

	t.Logf("Successful requests: %d/%d (%.1f%%)", successCount, numRequests, float64(successCount)/float64(numRequests)*100)
	assert.Greater(t, successCount, numRequests*95/100, "At least 95% of concurrent requests should succeed")
}

// TestHTTPAPIMiddlewareChain tests middleware integration
func TestHTTPAPIMiddlewareChain(t *testing.T) {
	ms := NewMockServer()
	testServer := ms.Start()
	defer testServer.Close()

	t.Run("CORS Headers", func(t *testing.T) {
		req, _ := http.NewRequest("OPTIONS", testServer.URL+"/api/v1/pipelines", nil)
		req.Header.Set("Origin", "http://localhost:3000")

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Check CORS headers are present
		assert.NotEmpty(t, resp.Header.Get("Access-Control-Allow-Origin"))
		assert.NotEmpty(t, resp.Header.Get("Access-Control-Allow-Methods"))
		assert.NotEmpty(t, resp.Header.Get("Access-Control-Allow-Headers"))
	})

	t.Run("API Versioning", func(t *testing.T) {
		resp, err := http.Get(testServer.URL + "/api/v1/pipelines")
		require.NoError(t, err)
		defer resp.Body.Close()

		versionHeader := resp.Header.Get("X-API-Version")
		assert.Equal(t, "v1", versionHeader)
	})

	t.Run("Content Type Headers", func(t *testing.T) {
		resp, err := http.Get(testServer.URL + "/api/v1/plugins")
		require.NoError(t, err)
		defer resp.Body.Close()

		contentType := resp.Header.Get("Content-Type")
		assert.Contains(t, contentType, "application/json")
	})
}

// TestHTTPAPIRateLimiting tests rate limiting behavior (if implemented)
func TestHTTPAPIPerformanceMetrics(t *testing.T) {
	ms := NewMockServer()
	testServer := ms.Start()
	defer testServer.Close()

	// Make several requests to generate metrics
	for i := 0; i < 10; i++ {
		resp, err := http.Get(testServer.URL + "/health")
		require.NoError(t, err)
		resp.Body.Close()
		time.Sleep(10 * time.Millisecond)
	}

	// Fetch performance metrics
	resp, err := http.Get(testServer.URL + "/api/v1/performance/metrics")
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var metrics map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&metrics)
	require.NoError(t, err)

	// Verify metrics structure
	requiredMetrics := []string{"total_requests", "average_latency", "requests_per_second"}
	for _, metric := range requiredMetrics {
		assert.Contains(t, metrics, metric, "Metric %s should be present", metric)
	}

	// Verify metric values are reasonable
	if totalRequests, ok := metrics["total_requests"].(float64); ok {
		assert.Greater(t, totalRequests, 0.0, "Total requests should be greater than 0")
	}
}

// TestHTTPAPIResponseTiming tests response time characteristics
func TestHTTPAPIResponseTiming(t *testing.T) {
	ms := NewMockServer()
	testServer := ms.Start()
	defer testServer.Close()

	const numTests = 20
	var durations []time.Duration

	for i := 0; i < numTests; i++ {
		start := time.Now()
		resp, err := http.Get(testServer.URL + "/health")
		duration := time.Since(start)

		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()

		durations = append(durations, duration)
	}

	// Calculate statistics
	var total time.Duration
	min := durations[0]
	max := durations[0]

	for _, d := range durations {
		total += d
		if d < min {
			min = d
		}
		if d > max {
			max = d
		}
	}

	avg := total / time.Duration(numTests)

	t.Logf("Response time statistics:")
	t.Logf("  Average: %v", avg)
	t.Logf("  Min: %v", min)
	t.Logf("  Max: %v", max)
	t.Logf("  Variance: %v", max-min)

	// Assert reasonable performance
	assert.Less(t, avg, 100*time.Millisecond, "Average response time should be under 100ms")
	assert.Less(t, max, 500*time.Millisecond, "Max response time should be under 500ms")
}

// TestHTTPAPIAuthentication tests authentication flow
func TestHTTPAPIAuthentication(t *testing.T) {
	ms := NewMockServer()
	testServer := ms.Start()
	defer testServer.Close()

	t.Run("Public Endpoints No Auth Required", func(t *testing.T) {
		publicEndpoints := []string{
			"/health",
			"/api/v1/plugins",
			"/api/v1/pipelines",
		}

		for _, endpoint := range publicEndpoints {
			resp, err := http.Get(testServer.URL + endpoint)
			require.NoError(t, err, "Public endpoint %s should be accessible", endpoint)
			resp.Body.Close()
			assert.NotEqual(t, http.StatusUnauthorized, resp.StatusCode)
			assert.NotEqual(t, http.StatusForbidden, resp.StatusCode)
		}
	})
}
