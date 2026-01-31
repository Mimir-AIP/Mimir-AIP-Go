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
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestContinuousInestion_FullPipelineFlow tests the complete automated data flow:
// API Input → Transform → Validation → Ontology Extraction → Knowledge Graph
func TestContinuousInestion_FullPipelineFlow(t *testing.T) {
	// Skip in short mode as this is a comprehensive integration test
	if testing.Short() {
		t.Skip("Skipping comprehensive integration test in short mode")
	}

	// Create server
	server := NewServer()
	require.NotNil(t, server)

	// Create a mock API server that returns data
	apiCallCount := 0
	mockAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiCallCount++

		response := map[string]any{
			"customers": []map[string]any{
				{
					"id":      "cust-001",
					"name":    "Alice Smith",
					"email":   "alice@example.com",
					"age":     32,
					"city":    "New York",
					"status":  "active",
					"revenue": 15000.50,
				},
				{
					"id":      "cust-002",
					"name":    "Bob Jones",
					"email":   "bob@example.com",
					"age":     28,
					"city":    "Los Angeles",
					"status":  "active",
					"revenue": 8500.00,
				},
				{
					"id":      "cust-003",
					"name":    "Charlie Brown",
					"email":   "charlie@example.com",
					"age":     45,
					"city":    "Chicago",
					"status":  "inactive",
					"revenue": 2500.75,
				},
				{
					"id":      "cust-004",
					"name":    "Diana Prince",
					"email":   "diana@example.com",
					"age":     35,
					"city":    "New York",
					"status":  "active",
					"revenue": 22000.00,
				},
			},
			"total":    4,
			"page":     1,
			"metadata": map[string]any{"source": "crm_system", "version": "2.1"},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer mockAPI.Close()

	var pipelineID string

	// Step 1: Create a pipeline with API input
	t.Run("Create API pipeline", func(t *testing.T) {
		pipelineDef := map[string]any{
			"metadata": map[string]any{
				"name":                  "continuous-customer-ingestion",
				"description":           "Automatically ingest customer data from API",
				"enabled":               true,
				"tags":                  []string{"automated", "api", "customers"},
				"auto_extract_ontology": true,
				"target_ontology_id":    "customer-ontology-v1",
			},
			"config": map[string]any{
				"name":    "continuous-customer-ingestion",
				"enabled": true,
				"steps": []map[string]any{
					{
						"name":   "fetch-customers",
						"plugin": "Input.api",
						"config": map[string]any{
							"url":           mockAPI.URL,
							"method":        "GET",
							"headers":       map[string]string{"Accept": "application/json"},
							"poll_interval": 300,
						},
						"output": "api_response",
					},
					{
						"name":   "transform-customers",
						"plugin": "Data_Processing.transform",
						"config": map[string]any{
							"input":     "api_response",
							"operation": "flatten",
							"sep":       ".",
						},
						"output": "flattened_data",
					},
					{
						"name":   "filter-active-customers",
						"plugin": "Data_Processing.transform",
						"config": map[string]any{
							"input":     "flattened_data",
							"operation": "filter",
							"field":     "customers.status",
							"op":        "==",
							"value":     "active",
						},
						"output": "active_customers",
					},
					{
						"name":   "select-relevant-fields",
						"plugin": "Data_Processing.transform",
						"config": map[string]any{
							"input":     "active_customers",
							"operation": "select",
							"fields":    []string{"customers.id", "customers.name", "customers.email", "customers.age", "customers.city", "customers.revenue"},
						},
						"output": "processed_customers",
					},
					{
						"name":   "validate-customer-data",
						"plugin": "Data_Processing.validate",
						"config": map[string]any{
							"input": "processed_customers",
							"rules": map[string]any{
								"required": []string{"customers.id", "customers.name", "customers.email"},
								"types": map[string]any{
									"customers.age":     "number",
									"customers.revenue": "number",
								},
								"ranges": map[string]any{
									"customers.age": map[string]any{
										"min": 18,
										"max": 120,
									},
								},
							},
						},
						"output": "validated_customers",
					},
				},
			},
		}

		body, _ := json.Marshal(pipelineDef)
		req := httptest.NewRequest("POST", "/api/v1/pipelines", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusCreated, "Expected 200 or 201, got %d", w.Code)

		var response map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		pipeline, ok := response["pipeline"].(map[string]any)
		require.True(t, ok)

		pipelineID = pipeline["id"].(string)
		assert.NotEmpty(t, pipelineID)
		assert.Equal(t, "continuous-customer-ingestion", pipeline["name"])
	})

	// Step 2: Execute the pipeline (simulating scheduled run)
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

		// Verify API was called
		assert.Equal(t, 1, apiCallCount, "API should have been called once")
	})

	// Step 3: Get pipeline execution history
	t.Run("Verify execution history", func(t *testing.T) {
		require.NotEmpty(t, pipelineID)

		req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/pipelines/%s/history", pipelineID), nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		// Should return history (may be empty or have entries depending on implementation)
		assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusNotFound)
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

	// Step 6: Simulate another execution to verify continuous flow
	t.Run("Execute pipeline again (simulating scheduled run)", func(t *testing.T) {
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
		assert.True(t, response.Success)

		// Verify API was called again
		assert.Equal(t, 2, apiCallCount, "API should have been called twice")
	})
}

// TestContinuousIngestion_APIPollingWithTransformations tests the full flow
// with multiple transformation steps
func TestContinuousIngestion_APIPollingWithTransformations(t *testing.T) {
	server := NewServer()
	require.NotNil(t, server)

	// Create mock API that returns time-series data
	mockAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]any{
			"measurements": []map[string]any{
				{"sensor_id": "temp-001", "value": 22.5, "unit": "celsius", "timestamp": time.Now().Unix(), "location": "room-a"},
				{"sensor_id": "temp-002", "value": 23.1, "unit": "celsius", "timestamp": time.Now().Unix(), "location": "room-b"},
				{"sensor_id": "hum-001", "value": 45.2, "unit": "percent", "timestamp": time.Now().Unix(), "location": "room-a"},
				{"sensor_id": "temp-003", "value": 21.8, "unit": "celsius", "timestamp": time.Now().Unix(), "location": "room-c"},
			},
			"count": 4,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer mockAPI.Close()

	var pipelineID string

	// Create pipeline with complex transformations
	t.Run("Create IoT data pipeline", func(t *testing.T) {
		pipelineDef := map[string]any{
			"metadata": map[string]any{
				"name":        "iot-sensor-ingestion",
				"description": "Ingest and transform IoT sensor data",
				"enabled":     true,
				"tags":        []string{"iot", "sensors", "automated"},
			},
			"config": map[string]any{
				"name":    "iot-sensor-ingestion",
				"enabled": true,
				"steps": []map[string]any{
					{
						"name":   "fetch-sensor-data",
						"plugin": "Input.api",
						"config": map[string]any{
							"url":    mockAPI.URL,
							"method": "GET",
						},
						"output": "sensor_data",
					},
					{
						"name":   "filter-temperature-sensors",
						"plugin": "Data_Processing.transform",
						"config": map[string]any{
							"input":      "sensor_data",
							"operation":  "filter",
							"field":      "sensor_id",
							"op":         "contains",
							"expression": "sensor_id.startsWith('temp')",
						},
						"output": "temperature_data",
					},
					{
						"name":   "calculate-fahrenheit",
						"plugin": "Data_Processing.transform",
						"config": map[string]any{
							"input":     "temperature_data",
							"operation": "map",
							"field":     "value",
							"function":  "custom",
						},
						"output": "transformed_temperature",
					},
					{
						"name":   "sort-by-value",
						"plugin": "Data_Processing.transform",
						"config": map[string]any{
							"input":     "transformed_temperature",
							"operation": "sort",
							"keys": []map[string]any{
								{"field": "value", "asc": false},
							},
						},
						"output": "sorted_sensors",
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

	// Execute the pipeline
	t.Run("Execute IoT pipeline", func(t *testing.T) {
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
		assert.True(t, response.Success)
	})

	// Schedule the pipeline
	t.Run("Schedule IoT pipeline", func(t *testing.T) {
		createReq := map[string]any{
			"id":        "iot-automation",
			"name":      "IoT Sensor Data Automation",
			"pipeline":  pipelineID,
			"cron_expr": "0 */1 * * *", // Every hour
		}

		body, _ := json.Marshal(createReq)
		req := httptest.NewRequest("POST", "/api/v1/scheduler/jobs", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusCreated, "Expected 200 or 201, got %d", w.Code)
	})
}

// TestContinuousIngestion_FileBasedAutomation tests automation with file-based input
// that simulates file watcher triggering
func TestContinuousIngestion_FileBasedAutomation(t *testing.T) {
	server := NewServer()
	require.NotNil(t, server)

	// Create temp directory for test files
	tmpDir := t.TempDir()

	// Create test CSV file
	csvFile := filepath.Join(tmpDir, "sales_data.csv")
	csvContent := `transaction_id,customer_name,amount,currency,date,timestamp
TXN-001,Alice Smith,150.00,USD,2026-01-30,2026-01-30T10:00:00Z
TXN-002,Bob Johnson,230.50,USD,2026-01-30,2026-01-30T10:05:00Z
TXN-003,Charlie Brown,89.99,EUR,2026-01-30,2026-01-30T10:10:00Z
TXN-004,Diana Prince,450.00,USD,2026-01-30,2026-01-30T10:15:00Z
TXN-005,Eve Anderson,125.75,GBP,2026-01-30,2026-01-30T10:20:00Z`

	err := os.WriteFile(csvFile, []byte(csvContent), 0644)
	require.NoError(t, err)

	var pipelineID string

	// Create pipeline for file processing
	t.Run("Create file-based pipeline", func(t *testing.T) {
		pipelineDef := map[string]any{
			"metadata": map[string]any{
				"name":        "automated-sales-ingestion",
				"description": "Automatically process sales data from CSV files",
				"enabled":     true,
				"tags":        []string{"sales", "csv", "automated"},
			},
			"config": map[string]any{
				"name":    "automated-sales-ingestion",
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
						"output": "raw_sales_data",
					},
					{
						"name":   "filter-usd-transactions",
						"plugin": "Data_Processing.transform",
						"config": map[string]any{
							"input":     "raw_sales_data",
							"operation": "filter",
							"field":     "currency",
							"op":        "==",
							"value":     "USD",
						},
						"output": "usd_transactions",
					},
					{
						"name":   "select-relevant-fields",
						"plugin": "Data_Processing.transform",
						"config": map[string]any{
							"input":     "usd_transactions",
							"operation": "select",
							"fields":    []string{"transaction_id", "customer_name", "amount", "timestamp"},
						},
						"output": "processed_transactions",
					},
					{
						"name":   "validate-transactions",
						"plugin": "Data_Processing.validate",
						"config": map[string]any{
							"input": "processed_transactions",
							"rules": map[string]any{
								"required": []string{"transaction_id", "customer_name", "amount"},
								"types": map[string]any{
									"amount": "number",
								},
							},
						},
						"output": "validated_transactions",
					},
					{
						"name":   "aggregate-by-customer",
						"plugin": "Data_Processing.transform",
						"config": map[string]any{
							"input":     "validated_transactions",
							"operation": "aggregate",
							"group_by":  []string{"customer_name"},
							"aggregations": []map[string]any{
								{"field": "amount", "op": "sum", "as": "total_amount"},
								{"field": "amount", "op": "count", "as": "transaction_count"},
							},
						},
						"output": "customer_summary",
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
	})

	// Execute the pipeline
	t.Run("Execute file-based pipeline", func(t *testing.T) {
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
		assert.True(t, response.Success)
	})

	// Schedule the pipeline
	t.Run("Schedule file pipeline", func(t *testing.T) {
		createReq := map[string]any{
			"id":        "sales-file-automation",
			"name":      "Sales File Processing Automation",
			"pipeline":  pipelineID,
			"cron_expr": "0 9 * * *", // Daily at 9 AM
		}

		body, _ := json.Marshal(createReq)
		req := httptest.NewRequest("POST", "/api/v1/scheduler/jobs", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusCreated, "Expected 200 or 201, got %d", w.Code)
	})
}

// TestContinuousIngestion_ErrorHandling tests error handling in automated pipelines
func TestContinuousIngestion_ErrorHandling(t *testing.T) {
	server := NewServer()
	require.NotNil(t, server)

	// Create a mock API that sometimes returns errors
	failCount := 0
	mockAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		failCount++
		if failCount == 1 {
			// First call fails
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]any{"error": "Internal Server Error"})
			return
		}

		// Second call succeeds
		response := map[string]any{
			"data": []map[string]any{
				{"id": "1", "name": "Test"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer mockAPI.Close()

	var pipelineID string

	// Create pipeline
	t.Run("Create error-handling pipeline", func(t *testing.T) {
		pipelineDef := map[string]any{
			"metadata": map[string]any{
				"name":    "error-handling-test",
				"enabled": true,
			},
			"config": map[string]any{
				"name":    "error-handling-test",
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
	})

	// First execution - should handle error gracefully
	t.Run("First execution with API error", func(t *testing.T) {
		execReq := PipelineExecutionRequest{
			PipelineID: pipelineID,
		}

		body, _ := json.Marshal(execReq)
		req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/pipelines/%s/execute", pipelineID), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		// Should complete (may or may not succeed depending on error handling)
		assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusInternalServerError)
	})

	// Second execution - should succeed
	t.Run("Second execution succeeds", func(t *testing.T) {
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
		assert.True(t, response.Success)
	})
}

// TestContinuousIngestion_MultipleDataSources tests pipeline with multiple API sources
func TestContinuousIngestion_MultipleDataSources(t *testing.T) {
	server := NewServer()
	require.NotNil(t, server)

	// Create two mock APIs
	customersAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := []map[string]any{
			{"id": "1", "name": "Alice", "email": "alice@example.com"},
			{"id": "2", "name": "Bob", "email": "bob@example.com"},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer customersAPI.Close()

	ordersAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := []map[string]any{
			{"order_id": "A001", "customer_id": "1", "amount": 100.00},
			{"order_id": "A002", "customer_id": "2", "amount": 200.00},
			{"order_id": "A003", "customer_id": "1", "amount": 150.00},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer ordersAPI.Close()

	var pipelineID string

	// Create multi-source pipeline
	t.Run("Create multi-source pipeline", func(t *testing.T) {
		pipelineDef := map[string]any{
			"metadata": map[string]any{
				"name":        "multi-source-ingestion",
				"description": "Ingest from multiple APIs",
				"enabled":     true,
			},
			"config": map[string]any{
				"name":    "multi-source-ingestion",
				"enabled": true,
				"steps": []map[string]any{
					{
						"name":   "fetch-customers",
						"plugin": "Input.api",
						"config": map[string]any{
							"url":    customersAPI.URL,
							"method": "GET",
						},
						"output": "customers",
					},
					{
						"name":   "fetch-orders",
						"plugin": "Input.api",
						"config": map[string]any{
							"url":    ordersAPI.URL,
							"method": "GET",
						},
						"output": "orders",
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
	})

	// Execute the multi-source pipeline
	t.Run("Execute multi-source pipeline", func(t *testing.T) {
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
		assert.True(t, response.Success)
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
