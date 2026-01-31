package Input

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// Error Handling Tests
// ============================================================================

// TestAPIPlugin_ExecuteStep_TimeoutError tests handling of request timeouts
func TestAPIPlugin_ExecuteStep_TimeoutError(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer mockServer.Close()

	plugin := &APIPlugin{}

	// Use a very short timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	stepConfig := pipelines.StepConfig{
		Name:   "api-timeout",
		Plugin: "Input.api",
		Config: map[string]any{
			"url":    mockServer.URL,
			"method": "GET",
		},
		Output: "data",
	}

	globalContext := pipelines.NewPluginContext()
	_, err := plugin.ExecuteStep(ctx, stepConfig, globalContext)

	// Should error due to timeout
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to execute request")
}

// TestAPIPlugin_ExecuteStep_BadStatusCode tests handling of non-2xx status codes
func TestAPIPlugin_ExecuteStep_BadStatusCode(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]any{"error": "Internal Server Error"})
	}))
	defer mockServer.Close()

	plugin := &APIPlugin{}
	ctx := context.Background()

	stepConfig := pipelines.StepConfig{
		Name:   "api-error",
		Plugin: "Input.api",
		Config: map[string]any{
			"url":    mockServer.URL,
			"method": "GET",
		},
		Output: "data",
	}

	globalContext := pipelines.NewPluginContext()
	result, err := plugin.ExecuteStep(ctx, stepConfig, globalContext)

	// Plugin doesn't error on bad status codes, but records it
	require.NoError(t, err)
	assert.NotNil(t, result)

	apiResponse, exists := result.Get("api_response")
	require.True(t, exists)
	responseData, ok := apiResponse.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, http.StatusInternalServerError, responseData["status_code"])
}

// TestAPIPlugin_ExecuteStep_InvalidJSONResponse tests handling of invalid JSON responses
func TestAPIPlugin_ExecuteStep_InvalidJSONResponse(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{invalid json}`))
	}))
	defer mockServer.Close()

	plugin := &APIPlugin{}
	ctx := context.Background()

	stepConfig := pipelines.StepConfig{
		Name:   "api-invalid-json",
		Plugin: "Input.api",
		Config: map[string]any{
			"url":    mockServer.URL,
			"method": "GET",
		},
		Output: "data",
	}

	globalContext := pipelines.NewPluginContext()
	_, err := plugin.ExecuteStep(ctx, stepConfig, globalContext)

	// Should error on invalid JSON
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse JSON response")
}

// TestAPIPlugin_ExecuteStep_EmptyResponse tests handling of empty responses
func TestAPIPlugin_ExecuteStep_EmptyResponse(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(""))
	}))
	defer mockServer.Close()

	plugin := &APIPlugin{}
	ctx := context.Background()

	stepConfig := pipelines.StepConfig{
		Name:   "api-empty",
		Plugin: "Input.api",
		Config: map[string]any{
			"url":    mockServer.URL,
			"method": "GET",
		},
		Output: "data",
	}

	globalContext := pipelines.NewPluginContext()
	_, err := plugin.ExecuteStep(ctx, stepConfig, globalContext)

	// Should error on empty response
	assert.Error(t, err)
}
