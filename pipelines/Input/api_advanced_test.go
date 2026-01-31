package Input

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// Advanced API Features Tests (Auth, Polling, Metadata)
// ============================================================================

// TestAPIPlugin_ExecuteStep_AuthenticationHeaders tests API requests with authentication headers
func TestAPIPlugin_ExecuteStep_AuthenticationHeaders(t *testing.T) {
	expectedToken := "Bearer test-token-12345"
	expectedAPIKey := "secret-api-key"

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify headers
		authHeader := r.Header.Get("Authorization")
		xAPIKey := r.Header.Get("X-API-Key")

		assert.Equal(t, expectedToken, authHeader)
		assert.Equal(t, expectedAPIKey, xAPIKey)

		response := map[string]any{"authenticated": true}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	plugin := &APIPlugin{}
	ctx := context.Background()

	stepConfig := pipelines.StepConfig{
		Name:   "api-auth-fetch",
		Plugin: "Input.api",
		Config: map[string]any{
			"url":    mockServer.URL,
			"method": "GET",
			"headers": map[string]string{
				"Authorization": expectedToken,
				"X-API-Key":     expectedAPIKey,
			},
		},
		Output: "api_data",
	}

	globalContext := pipelines.NewPluginContext()
	result, err := plugin.ExecuteStep(ctx, stepConfig, globalContext)

	require.NoError(t, err)
	assert.NotNil(t, result)

	// Verify response stored
	apiResponse, exists := result.Get("api_response")
	require.True(t, exists)
	responseData, ok := apiResponse.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, http.StatusOK, responseData["status_code"])
}

// TestAPIPlugin_ExecuteStep_CustomHeaders tests various custom headers
func TestAPIPlugin_ExecuteStep_CustomHeaders(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify custom headers
		assert.Equal(t, "application/json", r.Header.Get("Accept"))
		assert.Equal(t, "Mimir-Test/1.0", r.Header.Get("User-Agent"))
		assert.Equal(t, "12345", r.Header.Get("X-Request-ID"))
		assert.Equal(t, "test-value", r.Header.Get("X-Custom-Header"))

		response := map[string]any{"headers_received": true}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	plugin := &APIPlugin{}
	ctx := context.Background()

	stepConfig := pipelines.StepConfig{
		Name:   "api-headers",
		Plugin: "Input.api",
		Config: map[string]any{
			"url":    mockServer.URL,
			"method": "GET",
			"headers": map[string]string{
				"Accept":          "application/json",
				"User-Agent":      "Mimir-Test/1.0",
				"X-Request-ID":    "12345",
				"X-Custom-Header": "test-value",
			},
		},
		Output: "data",
	}

	globalContext := pipelines.NewPluginContext()
	result, err := plugin.ExecuteStep(ctx, stepConfig, globalContext)

	require.NoError(t, err)
	assert.NotNil(t, result)
}

// TestAPIPlugin_ExecuteStep_PollingInterval tests poll_interval configuration
func TestAPIPlugin_ExecuteStep_PollingInterval(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]any{"status": "ok"}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	plugin := &APIPlugin{}
	ctx := context.Background()

	stepConfig := pipelines.StepConfig{
		Name:   "api-poll",
		Plugin: "Input.api",
		Config: map[string]any{
			"url":           mockServer.URL,
			"method":        "GET",
			"poll_interval": 60, // 60 seconds
		},
		Output: "poll_data",
	}

	globalContext := pipelines.NewPluginContext()
	result, err := plugin.ExecuteStep(ctx, stepConfig, globalContext)

	require.NoError(t, err)
	assert.NotNil(t, result)

	// Verify metadata includes poll information
	sourceType, exists := result.GetMetadata("source_type")
	require.True(t, exists)
	assert.Equal(t, "rest_api", sourceType)
}

// TestAPIPlugin_ExecuteStep_MetadataTracking tests metadata is properly set
func TestAPIPlugin_ExecuteStep_MetadataTracking(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Rate-Limit", "100")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{"test": true})
	}))
	defer mockServer.Close()

	plugin := &APIPlugin{}
	ctx := context.Background()

	stepConfig := pipelines.StepConfig{
		Name:   "api-metadata",
		Plugin: "Input.api",
		Config: map[string]any{
			"url":    mockServer.URL,
			"method": "GET",
		},
		Output: "data",
	}

	globalContext := pipelines.NewPluginContext()
	result, err := plugin.ExecuteStep(ctx, stepConfig, globalContext)

	require.NoError(t, err)
	assert.NotNil(t, result)

	// Check metadata
	sourceType, exists := result.GetMetadata("source_type")
	require.True(t, exists)
	assert.Equal(t, "rest_api", sourceType)

	apiURL, exists := result.GetMetadata("api_url")
	require.True(t, exists)
	assert.Equal(t, mockServer.URL, apiURL)

	apiMethod, exists := result.GetMetadata("api_method")
	require.True(t, exists)
	assert.Equal(t, "GET", apiMethod)

	statusCode, exists := result.GetMetadata("status_code")
	require.True(t, exists)
	assert.Equal(t, http.StatusOK, statusCode)

	contentType, exists := result.GetMetadata("content_type")
	require.True(t, exists)
	assert.Equal(t, "application/json", contentType)

	extractedAt, exists := result.GetMetadata("extracted_at")
	require.True(t, exists)
	assert.NotEmpty(t, extractedAt)
}

// TestAPIPlugin_ValidateConfig_ValidWithHeaders tests validation with headers
func TestAPIPlugin_ValidateConfig_ValidWithHeaders(t *testing.T) {
	plugin := &APIPlugin{}

	err := plugin.ValidateConfig(map[string]any{
		"url":    "http://example.com",
		"method": "POST",
		"headers": map[string]string{
			"Authorization": "Bearer token",
		},
	})

	assert.NoError(t, err)
}

// TestAPIPlugin_GetPluginType tests plugin type
func TestAPIPlugin_GetPluginType(t *testing.T) {
	plugin := &APIPlugin{}
	assert.Equal(t, "Input", plugin.GetPluginType())
}

// TestAPIPlugin_GetPluginName tests plugin name
func TestAPIPlugin_GetPluginName(t *testing.T) {
	plugin := &APIPlugin{}
	assert.Equal(t, "api", plugin.GetPluginName())
}

// TestAPIPlugin_GetInputSchema tests schema definition
func TestAPIPlugin_GetInputSchema(t *testing.T) {
	plugin := &APIPlugin{}
	schema := plugin.GetInputSchema()

	assert.NotNil(t, schema)
	assert.Equal(t, "object", schema["type"])

	properties, ok := schema["properties"].(map[string]any)
	require.True(t, ok)

	// Check required fields exist
	assert.Contains(t, properties, "url")
	assert.Contains(t, properties, "method")
	assert.Contains(t, properties, "headers")
	assert.Contains(t, properties, "poll_interval")
}

// TestAPIPlugin_ExecuteStep_RealHTTPBin tests against real httpbin.org endpoint
// This test can be skipped if network is unavailable
func TestAPIPlugin_ExecuteStep_RealHTTPBin(t *testing.T) {
	// Skip if running in CI or network unavailable
	if testing.Short() {
		t.Skip("Skipping network test in short mode")
	}

	plugin := &APIPlugin{}
	ctx := context.Background()

	stepConfig := pipelines.StepConfig{
		Name:   "api-httpbin",
		Plugin: "Input.api",
		Config: map[string]any{
			"url":    "https://httpbin.org/get",
			"method": "GET",
			"headers": map[string]string{
				"Accept": "application/json",
			},
		},
		Output: "httpbin_data",
	}

	globalContext := pipelines.NewPluginContext()
	result, err := plugin.ExecuteStep(ctx, stepConfig, globalContext)

	// This may fail if httpbin is down, so use assert instead of require
	if err != nil {
		t.Skipf("Skipping: httpbin.org unavailable: %v", err)
	}

	assert.NotNil(t, result)

	apiResponse, exists := result.Get("api_response")
	if exists {
		responseData, ok := apiResponse.(map[string]any)
		if ok {
			assert.Equal(t, http.StatusOK, responseData["status_code"])
		}
	}
}
