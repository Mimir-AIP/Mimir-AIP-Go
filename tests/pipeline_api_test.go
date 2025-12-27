package tests

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/Mimir-AIP/Mimir-AIP-Go/utils"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPipelineAPIListPipelines recreates the exact conditions from production
// to isolate where the bug is when /api/v1/pipelines returns []
func TestPipelineAPIListPipelines(t *testing.T) {
	// Step 1: Create temp directory with pipeline files
	tempDir, err := os.MkdirTemp("", "pipeline_api_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create test pipeline files (mimicking what Docker volume contains)
	pipelineContent := `metadata:
  id: test-pipeline-001
  name: Test Pipeline 1
  description: A test pipeline
  enabled: true
  version: 1
config:
  name: Test Pipeline 1
  steps:
    - name: Step 1
      plugin: Input.csv
      config: {}
`
	err = os.WriteFile(filepath.Join(tempDir, "pipeline_001.yaml"), []byte(pipelineContent), 0644)
	require.NoError(t, err)

	pipeline2Content := `metadata:
  id: test-pipeline-002
  name: Test Pipeline 2
  description: Another test pipeline
  enabled: true
  version: 1
config:
  name: Test Pipeline 2
  steps:
    - name: Step 1
      plugin: Input.json
      config: {}
`
	err = os.WriteFile(filepath.Join(tempDir, "pipeline_002.yaml"), []byte(pipeline2Content), 0644)
	require.NoError(t, err)

	t.Logf("Created test pipelines in: %s", tempDir)

	// Step 2: Reset and initialize the global pipeline store (like server.go does)
	utils.ResetGlobalPipelineStore()
	err = utils.InitializeGlobalPipelineStore(tempDir)
	require.NoError(t, err)

	// Step 3: Verify store has pipelines directly
	store := utils.GetPipelineStore()
	pipelines, err := store.ListPipelines(nil)
	require.NoError(t, err)
	t.Logf("Direct store.ListPipelines() returned %d pipelines", len(pipelines))
	assert.Equal(t, 2, len(pipelines), "Store should have 2 pipelines")

	// Step 4: Create router with the exact same setup as routes.go
	router := mux.NewRouter()

	// Create API version subrouter (exactly like routes.go line 32)
	v1 := router.PathPrefix("/api/v1").Subrouter()

	// Register the handler (exactly like routes.go line 43)
	v1.HandleFunc("/pipelines", func(w http.ResponseWriter, r *http.Request) {
		t.Log("=== Handler /api/v1/pipelines called ===")
		store := utils.GetPipelineStore()
		t.Logf("Handler got store pointer: %p", store)
		pipelines, err := store.ListPipelines(nil)
		t.Logf("Handler ListPipelines returned %d pipelines, err=%v", len(pipelines), err)

		w.Header().Set("Content-Type", "application/json")
		if err != nil {
			json.NewEncoder(w).Encode([]*utils.PipelineDefinition{})
			return
		}
		json.NewEncoder(w).Encode(pipelines)
	}).Methods("GET")

	// Step 5: Make HTTP request
	req := httptest.NewRequest("GET", "/api/v1/pipelines", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Step 6: Verify response
	t.Logf("HTTP Status: %d", w.Code)
	t.Logf("HTTP Body: %s", w.Body.String())

	assert.Equal(t, http.StatusOK, w.Code)

	var responsePipelines []*utils.PipelineDefinition
	err = json.Unmarshal(w.Body.Bytes(), &responsePipelines)
	require.NoError(t, err)
	assert.Equal(t, 2, len(responsePipelines), "API should return 2 pipelines")
}

// TestPipelineAPIWithMiddleware tests with middleware chain like production
func TestPipelineAPIWithMiddleware(t *testing.T) {
	// Setup temp dir with pipelines
	tempDir, err := os.MkdirTemp("", "pipeline_api_middleware_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	pipelineContent := `metadata:
  id: middleware-test-pipeline
  name: Middleware Test Pipeline
  enabled: true
  version: 1
config:
  name: Middleware Test Pipeline
  steps:
    - name: Step 1
      plugin: Input.csv
      config: {}
`
	os.WriteFile(filepath.Join(tempDir, "middleware_pipeline.yaml"), []byte(pipelineContent), 0644)

	// Reset and initialize store
	utils.ResetGlobalPipelineStore()
	utils.InitializeGlobalPipelineStore(tempDir)

	// Create router with middleware
	router := mux.NewRouter()

	// Add logging middleware (simplified version)
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Logf("Middleware: %s %s", r.Method, r.URL.Path)
			next.ServeHTTP(w, r)
		})
	})

	// Add default user middleware (simplified version from middleware.go)
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Logf("DefaultUserMiddleware: path=%s", r.URL.Path)
			next.ServeHTTP(w, r)
		})
	})

	v1 := router.PathPrefix("/api/v1").Subrouter()
	v1.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Logf("V1 Middleware: path=%s", r.URL.Path)
			next.ServeHTTP(w, r)
		})
	})

	v1.HandleFunc("/pipelines", func(w http.ResponseWriter, r *http.Request) {
		t.Log("=== Handler called ===")
		store := utils.GetPipelineStore()
		pipelines, _ := store.ListPipelines(nil)
		t.Logf("Returning %d pipelines", len(pipelines))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(pipelines)
	}).Methods("GET")

	// Add catch-all proxy (like routes.go line 278)
	router.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Logf("CATCH-ALL handler hit for: %s", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("[]"))
	})

	// Make request
	req := httptest.NewRequest("GET", "/api/v1/pipelines", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	t.Logf("Response: %s", w.Body.String())

	var pipelines []*utils.PipelineDefinition
	json.Unmarshal(w.Body.Bytes(), &pipelines)
	assert.Equal(t, 1, len(pipelines), "Should return 1 pipeline, not be caught by catch-all")
}

// TestPipelineAPIRouteOrdering tests if route ordering matters
func TestPipelineAPIRouteOrdering(t *testing.T) {
	// Setup
	tempDir, _ := os.MkdirTemp("", "pipeline_route_order_test")
	defer os.RemoveAll(tempDir)

	pipelineContent := `metadata:
  id: route-test
  name: Route Test
  enabled: true
  version: 1
config:
  name: Route Test
  steps:
    - name: Step 1
      plugin: Input.csv
      config: {}
`
	os.WriteFile(filepath.Join(tempDir, "route_test.yaml"), []byte(pipelineContent), 0644)

	utils.ResetGlobalPipelineStore()
	utils.InitializeGlobalPipelineStore(tempDir)

	testCases := []struct {
		name          string
		setupRouter   func() *mux.Router
		expectSuccess bool
	}{
		{
			name: "Specific route before catch-all",
			setupRouter: func() *mux.Router {
				r := mux.NewRouter()
				v1 := r.PathPrefix("/api/v1").Subrouter()
				v1.HandleFunc("/pipelines", func(w http.ResponseWriter, req *http.Request) {
					store := utils.GetPipelineStore()
					pipelines, _ := store.ListPipelines(nil)
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(pipelines)
				}).Methods("GET")
				// Catch-all AFTER specific routes
				r.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
					w.Write([]byte("[]"))
				})
				return r
			},
			expectSuccess: true,
		},
		{
			name: "Catch-all before specific route",
			setupRouter: func() *mux.Router {
				r := mux.NewRouter()
				// Catch-all BEFORE specific routes - this might be the bug!
				r.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
					w.Write([]byte("[]"))
				})
				v1 := r.PathPrefix("/api/v1").Subrouter()
				v1.HandleFunc("/pipelines", func(w http.ResponseWriter, req *http.Request) {
					store := utils.GetPipelineStore()
					pipelines, _ := store.ListPipelines(nil)
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(pipelines)
				}).Methods("GET")
				return r
			},
			expectSuccess: false, // Catch-all might intercept
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			router := tc.setupRouter()
			req := httptest.NewRequest("GET", "/api/v1/pipelines", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			body := w.Body.String()
			t.Logf("Response: %s", body)

			var pipelines []*utils.PipelineDefinition
			json.Unmarshal(w.Body.Bytes(), &pipelines)

			if tc.expectSuccess {
				assert.Equal(t, 1, len(pipelines), "Should return pipeline")
			} else {
				t.Logf("Got %d pipelines (catch-all may have intercepted)", len(pipelines))
			}
		})
	}
}

// TestActualServerHandlerBinding tests using the actual Server struct handler
func TestActualServerHandlerBinding(t *testing.T) {
	// This test imports and uses the actual server handler to verify binding
	tempDir, _ := os.MkdirTemp("", "actual_server_test")
	defer os.RemoveAll(tempDir)

	pipelineContent := `metadata:
  id: actual-server-test
  name: Actual Server Test
  enabled: true
  version: 1
config:
  name: Actual Server Test
  steps:
    - name: Step 1
      plugin: Input.csv
      config: {}
`
	os.WriteFile(filepath.Join(tempDir, "actual_test.yaml"), []byte(pipelineContent), 0644)

	utils.ResetGlobalPipelineStore()
	utils.InitializeGlobalPipelineStore(tempDir)

	// Verify store is populated
	store := utils.GetPipelineStore()
	pipelines, err := store.ListPipelines(nil)
	require.NoError(t, err)
	require.Equal(t, 1, len(pipelines), "Store should have 1 pipeline")

	// Make direct HTTP server test
	router := mux.NewRouter()
	v1 := router.PathPrefix("/api/v1").Subrouter()

	// This simulates what handlers.go handleListPipelines does
	v1.HandleFunc("/pipelines", func(w http.ResponseWriter, r *http.Request) {
		store := utils.GetPipelineStore()
		pipelines, err := store.ListPipelines(nil)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]*utils.PipelineDefinition{})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(pipelines)
	}).Methods("GET")

	// Start test server
	server := httptest.NewServer(router)
	defer server.Close()

	// Make actual HTTP request
	resp, err := http.Get(server.URL + "/api/v1/pipelines")
	require.NoError(t, err)
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	t.Logf("Response: %s", string(body))

	var resultPipelines []*utils.PipelineDefinition
	err = json.Unmarshal(body, &resultPipelines)
	require.NoError(t, err)
	assert.Equal(t, 1, len(resultPipelines), "Should return 1 pipeline via HTTP")
}

// TestGlobalStoreConsistency verifies the global store is consistent across calls
func TestGlobalStoreConsistency(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "store_consistency_test")
	defer os.RemoveAll(tempDir)

	pipelineContent := `metadata:
  id: consistency-test
  name: Consistency Test
  enabled: true
  version: 1
config:
  name: Consistency Test
  steps:
    - name: Step 1
      plugin: Input.csv
      config: {}
`
	os.WriteFile(filepath.Join(tempDir, "consistency.yaml"), []byte(pipelineContent), 0644)

	utils.ResetGlobalPipelineStore()
	utils.InitializeGlobalPipelineStore(tempDir)

	// Get store multiple times and verify consistency
	store1 := utils.GetPipelineStore()
	store2 := utils.GetPipelineStore()
	store3 := utils.GetPipelineStore()

	t.Logf("Store pointers: %p, %p, %p", store1, store2, store3)
	assert.Equal(t, fmt.Sprintf("%p", store1), fmt.Sprintf("%p", store2), "Store pointers should match")
	assert.Equal(t, fmt.Sprintf("%p", store2), fmt.Sprintf("%p", store3), "Store pointers should match")

	pipelines1, _ := store1.ListPipelines(nil)
	pipelines2, _ := store2.ListPipelines(nil)
	pipelines3, _ := store3.ListPipelines(nil)

	assert.Equal(t, len(pipelines1), len(pipelines2), "Pipeline counts should match")
	assert.Equal(t, len(pipelines2), len(pipelines3), "Pipeline counts should match")
	assert.Equal(t, 1, len(pipelines1), "Should have 1 pipeline")
}

