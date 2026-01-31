package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// API-Based Ingestion Tests
// ============================================================================

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
