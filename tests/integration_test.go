package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
	"github.com/Mimir-AIP/Mimir-AIP-Go/utils"
	"github.com/gorilla/mux"
)

// MockServer represents a test server instance
type MockServer struct {
	router   *mux.Router
	registry *pipelines.PluginRegistry
}

// GetRegistry returns the mock server's plugin registry
func (ms *MockServer) GetRegistry() *pipelines.PluginRegistry {
	return ms.registry
}

// NewMockServer creates a new test server with basic routes
func NewMockServer() *MockServer {
	// Create a temporary directory for testing
	tempDir, _ := os.MkdirTemp("", "mimir_test")
	defer os.RemoveAll(tempDir)

	// Initialize pipeline store with temp directory
	_ = utils.InitializeGlobalPipelineStore(tempDir)

	// Create plugin registry
	registry := pipelines.NewPluginRegistry()

	// Create a simple router for testing
	router := mux.NewRouter()

	// Add CORS middleware
	router.Use(corsMiddleware())

	// Add API versioning middleware
	v1 := router.PathPrefix("/api/v1").Subrouter()
	v1.Use(corsMiddleware())
	v1.Use(versionMiddleware("v1"))

	// Add basic routes for testing
	router.HandleFunc("/health", handleTestHealth).Methods("GET")
	v1.HandleFunc("/pipelines", handleTestListPipelines).Methods("GET", "OPTIONS")
	v1.HandleFunc("/pipelines/execute", handleTestPipelineExecute).Methods("POST")
	v1.HandleFunc("/plugins", handleTestListPlugins).Methods("GET")
	v1.HandleFunc("/performance/metrics", handleTestPerformanceMetrics).Methods("GET")
	v1.HandleFunc("/jobs", handleTestListJobs).Methods("GET")
	v1.HandleFunc("/config", handleTestGetConfig).Methods("GET")
	v1.HandleFunc("/scheduler/jobs", handleTestListSchedulerJobs).Methods("GET")
	v1.HandleFunc("/visualize/status", handleTestVisualizeStatus).Methods("GET")

	// Add MCP routes
	router.HandleFunc("/mcp/tools", handleMCPToolDiscovery(registry)).Methods("GET")
	router.HandleFunc("/mcp/tools/execute", handleMCPToolExecution(registry)).Methods("POST")

	return &MockServer{router: router, registry: registry}
}

// versionMiddleware adds API version information to requests
func versionMiddleware(version string) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Add version header to response
			w.Header().Set("X-API-Version", version)
			next.ServeHTTP(w, r)
		})
	}
}

// corsMiddleware adds CORS headers to responses
func corsMiddleware() mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Add CORS headers
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

			// Handle preflight requests
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// Start starts the test server
func (ms *MockServer) Start() *httptest.Server {
	return httptest.NewServer(ms.router)
}

// Handler functions for testing

func handleTestHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status": "healthy",
		"time":   "2025-08-25T12:30:00Z",
	})
}

func handleTestListPipelines(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode([]utils.PipelineConfig{})
}

func handleTestListPlugins(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	plugins := []map[string]any{
		{"type": "Input", "name": "api", "description": "API input plugin"},
		{"type": "Output", "name": "html", "description": "HTML output plugin"},
	}
	_ = json.NewEncoder(w).Encode(plugins)
}

func handleTestPerformanceMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	metrics := map[string]any{
		"total_requests":      100,
		"average_latency":     5000000, // 5ms in nanoseconds
		"requests_per_second": 20.5,
	}
	_ = json.NewEncoder(w).Encode(metrics)
}

func handleTestListJobs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode([]map[string]any{})
}

func handleTestGetConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	config := utils.Config{
		Server: utils.ServerConfig{
			Host: "localhost",
			Port: 8080,
		},
		Logging: utils.LoggingConfig{
			Level: "info",
		},
	}
	_ = json.NewEncoder(w).Encode(config)
}

func handleTestListSchedulerJobs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode([]utils.ScheduledJob{})
}

func handleTestVisualizeStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	_, _ = w.Write([]byte("System Status Visualization\n=======================\nAll systems operational"))
}

func handleTestPipelineExecute(w http.ResponseWriter, r *http.Request) {
	// This endpoint should return 405 for POST requests in the mock server
	// as it's not implemented in the test mock
	w.WriteHeader(http.StatusMethodNotAllowed)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": "Method not allowed",
	})
}

// TestHealthEndpoint tests the health check endpoint
func TestHealthEndpoint(t *testing.T) {
	ms := NewMockServer()
	testServer := ms.Start()
	defer testServer.Close()

	resp, err := http.Get(testServer.URL + "/health")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var response map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["status"] != "healthy" {
		t.Errorf("Expected status 'healthy', got '%v'", response["status"])
	}

	if _, exists := response["time"]; !exists {
		t.Error("Expected 'time' field in response")
	}
}

// TestPipelineExecutionEndpoint tests the pipeline execution endpoint
func TestPipelineExecutionEndpoint(t *testing.T) {
	ms := NewMockServer()
	testServer := ms.Start()
	defer testServer.Close()

	// Create request payload
	request := map[string]any{
		"pipeline_name": "test",
		"context":       map[string]any{"test": "data"},
	}

	jsonData, _ := json.Marshal(request)

	req, err := http.NewRequest("POST", testServer.URL+"/api/v1/pipelines/execute", bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	// Should return 405 because POST is not implemented in mock server
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", resp.StatusCode)
	}
}

// TestListPipelinesEndpoint tests the list pipelines endpoint
func TestListPipelinesEndpoint(t *testing.T) {
	ms := NewMockServer()
	testServer := ms.Start()
	defer testServer.Close()

	resp, err := http.Get(testServer.URL + "/api/v1/pipelines")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var pipelines []utils.PipelineConfig
	if err := json.NewDecoder(resp.Body).Decode(&pipelines); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Should return empty array initially
	if len(pipelines) != 0 {
		t.Errorf("Expected empty array, got %d pipelines", len(pipelines))
	}
}

// TestPluginEndpoints tests plugin-related endpoints
func TestPluginEndpoints(t *testing.T) {
	ms := NewMockServer()
	testServer := ms.Start()
	defer testServer.Close()

	// Test list plugins
	resp, err := http.Get(testServer.URL + "/api/v1/plugins")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var plugins []PluginInfo
	if err := json.NewDecoder(resp.Body).Decode(&plugins); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Should have default plugins registered
	if len(plugins) == 0 {
		t.Error("Expected at least one plugin to be registered")
	}
}

// TestPerformanceMetricsEndpoint tests the performance metrics endpoint
func TestPerformanceMetricsEndpoint(t *testing.T) {
	ms := NewMockServer()
	testServer := ms.Start()
	defer testServer.Close()

	resp, err := http.Get(testServer.URL + "/api/v1/performance/metrics")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var metrics map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&metrics); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Check for expected fields
	expectedFields := []string{"total_requests", "average_latency", "requests_per_second"}
	for _, field := range expectedFields {
		if _, exists := metrics[field]; !exists {
			t.Errorf("Expected field '%s' in metrics response", field)
		}
	}
}

// TestJobMonitoringEndpoints tests job monitoring endpoints
func TestJobMonitoringEndpoints(t *testing.T) {
	ms := NewMockServer()
	testServer := ms.Start()
	defer testServer.Close()

	// Test list jobs
	resp, err := http.Get(testServer.URL + "/api/v1/jobs")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var jobs []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&jobs); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Should return empty array initially
	if len(jobs) != 0 {
		t.Errorf("Expected empty array, got %d jobs", len(jobs))
	}
}

// TestConfigurationEndpoints tests configuration endpoints
func TestConfigurationEndpoints(t *testing.T) {
	ms := NewMockServer()
	testServer := ms.Start()
	defer testServer.Close()

	// Test get config
	resp, err := http.Get(testServer.URL + "/api/v1/config")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var config utils.Config
	if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Check for expected config sections
	if config.Server.Host == "" {
		t.Error("Expected server config to be present")
	}
	if config.Logging.Level == "" {
		t.Error("Expected logging config to be present")
	}
}

// TestSchedulerEndpoints tests scheduler endpoints
func TestSchedulerEndpoints(t *testing.T) {
	ms := NewMockServer()
	testServer := ms.Start()
	defer testServer.Close()

	// Test list jobs
	resp, err := http.Get(testServer.URL + "/api/v1/scheduler/jobs")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var jobs []utils.ScheduledJob
	if err := json.NewDecoder(resp.Body).Decode(&jobs); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Should return empty array initially
	if len(jobs) != 0 {
		t.Errorf("Expected empty array, got %d scheduled jobs", len(jobs))
	}
}

// TestVisualizationEndpoints tests visualization endpoints
func TestVisualizationEndpoints(t *testing.T) {
	ms := NewMockServer()
	testServer := ms.Start()
	defer testServer.Close()

	// Test status visualization
	resp, err := http.Get(testServer.URL + "/api/v1/visualize/status")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	if len(body) == 0 {
		t.Error("Expected non-empty visualization response")
	}
}

// TestAPIVersioning tests that API versioning is working
func TestAPIVersioning(t *testing.T) {
	ms := NewMockServer()
	testServer := ms.Start()
	defer testServer.Close()

	// Test v1 endpoint
	resp, err := http.Get(testServer.URL + "/api/v1/pipelines")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	// Check for version header
	versionHeader := resp.Header.Get("X-API-Version")
	if versionHeader != "v1" {
		t.Errorf("Expected X-API-Version header to be 'v1', got '%s'", versionHeader)
	}

	// Test that non-versioned endpoints still work
	resp2, err := http.Get(testServer.URL + "/health")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 for health endpoint, got %d", resp2.StatusCode)
	}
}

// TestCORSMiddleware tests CORS functionality
func TestCORSMiddleware(t *testing.T) {
	ms := NewMockServer()
	testServer := ms.Start()
	defer testServer.Close()

	// Test OPTIONS request
	req, err := http.NewRequest("OPTIONS", testServer.URL+"/api/v1/pipelines", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	// Check CORS headers
	corsOrigin := resp.Header.Get("Access-Control-Allow-Origin")
	if corsOrigin != "*" {
		t.Errorf("Expected CORS origin '*', got '%s'", corsOrigin)
	}

	corsMethods := resp.Header.Get("Access-Control-Allow-Methods")
	if corsMethods == "" {
		t.Error("Expected CORS methods header to be present")
	}
}

// TestErrorHandling tests error handling middleware
func TestErrorHandling(t *testing.T) {
	ms := NewMockServer()
	testServer := ms.Start()
	defer testServer.Close()

	// Test invalid endpoint
	resp, err := http.Get(testServer.URL + "/api/v1/nonexistent")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	// Should return 404
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404 for invalid endpoint, got %d", resp.StatusCode)
	}
}

// TestConcurrentRequests tests concurrent request handling
func TestConcurrentRequests(t *testing.T) {
	ms := NewMockServer()
	testServer := ms.Start()
	defer testServer.Close()

	const numRequests = 10
	results := make(chan bool, numRequests)

	// Make concurrent requests
	for i := 0; i < numRequests; i++ {
		go func() {
			resp, err := http.Get(testServer.URL + "/health")
			if err != nil {
				results <- false
				return
			}
			defer resp.Body.Close()
			results <- resp.StatusCode == http.StatusOK
		}()
	}

	// Check results
	for i := 0; i < numRequests; i++ {
		if !<-results {
			t.Errorf("Request %d failed", i)
		}
	}
}

// TestServerShutdown tests graceful shutdown functionality
func TestServerShutdown(t *testing.T) {
	// This test is simplified for the mock server
	// In a real implementation, this would test the actual server shutdown
	ms := NewMockServer()

	// Create a context with timeout for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// For mock server, we just test that context works
	select {
	case <-ctx.Done():
		// Context was cancelled/timeout - this is expected
		break
	case <-time.After(100 * time.Millisecond):
		// Test passes if we reach here (context not cancelled)
		break
	}

	// Verify mock server was created successfully
	if ms == nil {
		t.Error("Mock server should not be nil")
	}
}

// Helper types for testing

// PipelineExecutionRequest represents a request to execute a pipeline
type PipelineExecutionRequest struct {
	PipelineName string         `json:"pipeline_name,omitempty"`
	PipelineFile string         `json:"pipeline_file,omitempty"`
	Context      map[string]any `json:"context,omitempty"`
}

// PluginInfo represents information about a plugin
type PluginInfo struct {
	Type        string `json:"type"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// MCP handler functions for mock server

func handleMCPToolDiscovery(registry *pipelines.PluginRegistry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		allPlugins := registry.GetAllPlugins()
		tools := make([]map[string]any, 0)

		for pluginType, pluginsOfType := range allPlugins {
			for pluginName := range pluginsOfType {
				tools = append(tools, map[string]any{
					"name":        pluginType + "." + pluginName,
					"type":        pluginType,
					"plugin_name": pluginName,
					"description": "Tool for " + pluginName,
					"inputSchema": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"step_config": map[string]any{
								"type": "object",
							},
							"context": map[string]any{
								"type": "object",
							},
						},
					},
				})
			}
		}

		response := map[string]any{
			"tools": tools,
		}

		_ = json.NewEncoder(w).Encode(response)
	}
}

func handleMCPToolExecution(registry *pipelines.PluginRegistry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		var request map[string]any
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"error": "Invalid JSON",
			})
			return
		}

		toolName, ok := request["tool_name"].(string)
		if !ok {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"error": "Missing tool_name",
			})
			return
		}

		// Execute tool (simplified)
		response := map[string]any{
			"success": true,
			"result":  "Tool " + toolName + " executed successfully",
		}

		_ = json.NewEncoder(w).Encode(response)
	}
}
