package Input

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// Basic HTTP Operations Tests
// ============================================================================

// TestAPIPlugin_ExecuteStep_BasicGET tests basic GET request to API endpoint
func TestAPIPlugin_ExecuteStep_BasicGET(t *testing.T) {
	// Create a mock server that returns JSON data
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)

		response := map[string]any{
			"id":      "12345",
			"name":    "Test Product",
			"price":   29.99,
			"inStock": true,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	plugin := &APIPlugin{}
	ctx := context.Background()

	stepConfig := pipelines.StepConfig{
		Name:   "api-fetch",
		Plugin: "Input.api",
		Config: map[string]any{
			"url":    mockServer.URL,
			"method": "GET",
		},
		Output: "api_data",
	}

	globalContext := pipelines.NewPluginContext()
	result, err := plugin.ExecuteStep(ctx, stepConfig, globalContext)

	require.NoError(t, err)
	assert.NotNil(t, result)

	// Check extracted data
	extractedData, exists := result.Get("extracted_data")
	require.True(t, exists, "extracted_data should exist in result")

	data, ok := extractedData.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, 1, data["row_count"])

	rows, ok := data["rows"].([]map[string]interface{})
	require.True(t, ok)
	require.Len(t, rows, 1)

	firstRow := rows[0]
	assert.Equal(t, "12345", firstRow["id"])
	assert.Equal(t, "Test Product", firstRow["name"])
	assert.Equal(t, float64(29.99), firstRow["price"])
	assert.Equal(t, true, firstRow["inStock"])
}

// TestAPIPlugin_ExecuteStep_POSTRequest tests POST requests
func TestAPIPlugin_ExecuteStep_POSTRequest(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		response := map[string]any{
			"results": []map[string]any{
				{"id": "1", "title": "Result 1"},
				{"id": "2", "title": "Result 2"},
			},
			"total": 2,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	plugin := &APIPlugin{}
	ctx := context.Background()

	stepConfig := pipelines.StepConfig{
		Name:   "api-post",
		Plugin: "Input.api",
		Config: map[string]any{
			"url":    mockServer.URL,
			"method": "POST",
			"headers": map[string]string{
				"Content-Type": "application/json",
			},
		},
		Output: "search_results",
	}

	globalContext := pipelines.NewPluginContext()
	result, err := plugin.ExecuteStep(ctx, stepConfig, globalContext)

	require.NoError(t, err)
	assert.NotNil(t, result)

	// Verify status code
	apiResponse, exists := result.Get("api_response")
	require.True(t, exists)
	responseData, ok := apiResponse.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, http.StatusCreated, responseData["status_code"])
}

// TestAPIPlugin_ExecuteStep_PUTRequest tests PUT requests
func TestAPIPlugin_ExecuteStep_PUTRequest(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PUT", r.Method)

		response := map[string]any{
			"id":      "123",
			"updated": true,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	plugin := &APIPlugin{}
	ctx := context.Background()

	stepConfig := pipelines.StepConfig{
		Name:   "api-put",
		Plugin: "Input.api",
		Config: map[string]any{
			"url":    mockServer.URL,
			"method": "PUT",
			"headers": map[string]string{
				"Content-Type": "application/json",
			},
		},
		Output: "update_result",
	}

	globalContext := pipelines.NewPluginContext()
	result, err := plugin.ExecuteStep(ctx, stepConfig, globalContext)

	require.NoError(t, err)
	assert.NotNil(t, result)
}

// TestAPIPlugin_ExecuteStep_DELETERequest tests DELETE requests
func TestAPIPlugin_ExecuteStep_DELETERequest(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)

		response := map[string]any{"deleted": true}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	plugin := &APIPlugin{}
	ctx := context.Background()

	stepConfig := pipelines.StepConfig{
		Name:   "api-delete",
		Plugin: "Input.api",
		Config: map[string]any{
			"url":    mockServer.URL,
			"method": "DELETE",
		},
		Output: "delete_result",
	}

	globalContext := pipelines.NewPluginContext()
	result, err := plugin.ExecuteStep(ctx, stepConfig, globalContext)

	require.NoError(t, err)
	assert.NotNil(t, result)
}

// TestAPIPlugin_ExecuteStep_ArrayResponse tests handling of array JSON responses
func TestAPIPlugin_ExecuteStep_ArrayResponse(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := []map[string]any{
			{"id": "1", "name": "Item 1"},
			{"id": "2", "name": "Item 2"},
			{"id": "3", "name": "Item 3"},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	plugin := &APIPlugin{}
	ctx := context.Background()

	stepConfig := pipelines.StepConfig{
		Name:   "api-array-fetch",
		Plugin: "Input.api",
		Config: map[string]any{
			"url":    mockServer.URL,
			"method": "GET",
		},
		Output: "items",
	}

	globalContext := pipelines.NewPluginContext()
	result, err := plugin.ExecuteStep(ctx, stepConfig, globalContext)

	require.NoError(t, err)
	assert.NotNil(t, result)

	// Verify all items extracted
	extractedData, exists := result.Get("extracted_data")
	require.True(t, exists)
	data, ok := extractedData.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, 3, data["row_count"])

	rows, ok := data["rows"].([]map[string]interface{})
	require.True(t, ok)
	assert.Len(t, rows, 3)
}

// TestAPIPlugin_ExecuteStep_NestedJSONResponse tests handling of nested JSON responses
func TestAPIPlugin_ExecuteStep_NestedJSONResponse(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]any{
			"data": map[string]any{
				"users": []map[string]any{
					{"id": "1", "name": "Alice"},
					{"id": "2", "name": "Bob"},
				},
				"pagination": map[string]any{
					"page":     1,
					"per_page": 10,
					"total":    2,
				},
			},
			"status": "success",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	plugin := &APIPlugin{}
	ctx := context.Background()

	stepConfig := pipelines.StepConfig{
		Name:   "api-nested-fetch",
		Plugin: "Input.api",
		Config: map[string]any{
			"url":    mockServer.URL,
			"method": "GET",
		},
		Output: "nested_data",
	}

	globalContext := pipelines.NewPluginContext()
	result, err := plugin.ExecuteStep(ctx, stepConfig, globalContext)

	require.NoError(t, err)
	assert.NotNil(t, result)

	// Verify full response stored
	apiResponse, exists := result.Get("api_response")
	require.True(t, exists)
	responseData, ok := apiResponse.(map[string]any)
	require.True(t, ok)

	jsonData, ok := responseData["json_data"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "success", jsonData["status"])
}

// TestAPIPlugin_ExecuteStep_QueryParameters tests URL query parameters
func TestAPIPlugin_ExecuteStep_QueryParameters(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify query params
		query := r.URL.Query()
		assert.Equal(t, "10", query.Get("limit"))
		assert.Equal(t, "name", query.Get("sort"))
		assert.Equal(t, "asc", query.Get("order"))

		response := map[string]any{"received": true}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	plugin := &APIPlugin{}
	ctx := context.Background()

	stepConfig := pipelines.StepConfig{
		Name:   "api-query-params",
		Plugin: "Input.api",
		Config: map[string]any{
			"url":    mockServer.URL + "?limit=10&sort=name&order=asc",
			"method": "GET",
		},
		Output: "data",
	}

	globalContext := pipelines.NewPluginContext()
	result, err := plugin.ExecuteStep(ctx, stepConfig, globalContext)

	require.NoError(t, err)
	assert.NotNil(t, result)
}

// TestAPIPlugin_ExecuteStep_MultipleRequests tests making multiple sequential requests
func TestAPIPlugin_ExecuteStep_MultipleRequests(t *testing.T) {
	requestCount := 0
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		response := map[string]any{
			"request_number": requestCount,
			"timestamp":      time.Now().Unix(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	plugin := &APIPlugin{}
	ctx := context.Background()

	// Make 3 requests
	for i := 0; i < 3; i++ {
		stepConfig := pipelines.StepConfig{
			Name:   fmt.Sprintf("api-request-%d", i),
			Plugin: "Input.api",
			Config: map[string]any{
				"url":    mockServer.URL,
				"method": "GET",
			},
			Output: fmt.Sprintf("data_%d", i),
		}

		globalContext := pipelines.NewPluginContext()
		result, err := plugin.ExecuteStep(ctx, stepConfig, globalContext)

		require.NoError(t, err)
		assert.NotNil(t, result)
	}

	assert.Equal(t, 3, requestCount)
}

// TestAPIPlugin_ValidateConfig_MissingURL tests validation without URL
func TestAPIPlugin_ValidateConfig_MissingURL(t *testing.T) {
	plugin := &APIPlugin{}

	err := plugin.ValidateConfig(map[string]any{
		"method": "GET",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "url is required")
}

// TestAPIPlugin_ValidateConfig_MissingMethod tests validation without method
func TestAPIPlugin_ValidateConfig_MissingMethod(t *testing.T) {
	plugin := &APIPlugin{}

	err := plugin.ValidateConfig(map[string]any{
		"url": "http://example.com",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "method is required")
}

// TestAPIPlugin_ValidateConfig_Valid tests valid configuration
func TestAPIPlugin_ValidateConfig_Valid(t *testing.T) {
	plugin := &APIPlugin{}

	err := plugin.ValidateConfig(map[string]any{
		"url":    "http://example.com",
		"method": "GET",
	})

	assert.NoError(t, err)
}
