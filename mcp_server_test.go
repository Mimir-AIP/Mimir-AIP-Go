package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockPlugin for testing MCP server
type MockPlugin struct {
	pluginType string
	pluginName string
	shouldFail bool
	executed   bool
}

func (m *MockPlugin) ExecuteStep(ctx context.Context, stepConfig pipelines.StepConfig, globalContext *pipelines.PluginContext) (*pipelines.PluginContext, error) {
	m.executed = true
	if m.shouldFail {
		return nil, assert.AnError
	}

	result := pipelines.NewPluginContext()
	result.Set(stepConfig.Output, map[string]any{
		"plugin":  m.pluginName,
		"success": true,
	})
	return result, nil
}

func (m *MockPlugin) GetPluginType() string { return m.pluginType }
func (m *MockPlugin) GetPluginName() string { return m.pluginName }
func (m *MockPlugin) ValidateConfig(config map[string]any) error {
	if m.shouldFail {
		return assert.AnError
	}
	return nil
}

func TestNewMCPServer(t *testing.T) {
	registry := pipelines.NewPluginRegistry()
	server := NewMCPServer(registry)

	assert.NotNil(t, server)
	assert.Equal(t, registry, server.registry)
}

func TestMCPServerInitialize(t *testing.T) {
	registry := pipelines.NewPluginRegistry()
	server := NewMCPServer(registry)

	err := server.Initialize()
	assert.NoError(t, err)
}

func TestMCPServerServeHTTP(t *testing.T) {
	registry := pipelines.NewPluginRegistry()
	server := NewMCPServer(registry)

	// Test tool discovery endpoint
	req := httptest.NewRequest("GET", "/mcp/tools", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "tools")
}

func TestMCPServerServeHTTPInvalidPath(t *testing.T) {
	registry := pipelines.NewPluginRegistry()
	server := NewMCPServer(registry)

	// Test invalid path
	req := httptest.NewRequest("GET", "/invalid", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestMCPServerServeHTTPInvalidMethod(t *testing.T) {
	registry := pipelines.NewPluginRegistry()
	server := NewMCPServer(registry)

	// Test invalid method for tool discovery
	req := httptest.NewRequest("POST", "/mcp/tools", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestMCPServerHandleToolDiscovery(t *testing.T) {
	registry := pipelines.NewPluginRegistry()

	// Register some mock plugins
	mockInput := &MockPlugin{pluginType: "Input", pluginName: "test"}
	mockOutput := &MockPlugin{pluginType: "Output", pluginName: "test"}
	registry.RegisterPlugin(mockInput)
	registry.RegisterPlugin(mockOutput)

	server := NewMCPServer(registry)

	req := httptest.NewRequest("GET", "/mcp/tools", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "tools")

	tools, ok := response["tools"].([]any)
	assert.True(t, ok)
	assert.Len(t, tools, 2) // Two plugins registered

	// Check tool structure
	for _, tool := range tools {
		toolMap, ok := tool.(map[string]any)
		assert.True(t, ok)
		assert.Contains(t, toolMap, "name")
		assert.Contains(t, toolMap, "description")
		assert.Contains(t, toolMap, "inputSchema")
	}
}

func TestMCPServerHandleToolExecution(t *testing.T) {
	registry := pipelines.NewPluginRegistry()

	// Register a mock plugin
	mockPlugin := &MockPlugin{pluginType: "Input", pluginName: "test"}
	registry.RegisterPlugin(mockPlugin)

	server := NewMCPServer(registry)

	// Prepare request body
	requestBody := map[string]any{
		"tool_name": "Input.test",
		"arguments": map[string]any{
			"step_config": map[string]any{
				"name": "test-step",
				"config": map[string]any{
					"param1": "value1",
				},
				"output": "test-output",
			},
			"context": map[string]any{
				"existing_key": "existing_value",
			},
		},
	}

	bodyBytes, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/mcp/tools/execute", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response map[string]any
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response["success"].(bool))
	assert.Contains(t, response, "result")
}

func TestMCPServerHandleToolExecutionInvalidJSON(t *testing.T) {
	registry := pipelines.NewPluginRegistry()
	server := NewMCPServer(registry)

	// Invalid JSON
	req := httptest.NewRequest("POST", "/mcp/tools/execute", bytes.NewReader([]byte("{invalid json")))
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestMCPServerHandleToolExecutionInvalidToolName(t *testing.T) {
	registry := pipelines.NewPluginRegistry()
	server := NewMCPServer(registry)

	// Invalid tool name format
	requestBody := map[string]any{
		"tool_name": "invalid", // Missing dot separator
		"arguments": map[string]any{
			"step_config": map[string]any{
				"name": "test-step",
			},
		},
	}

	bodyBytes, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/mcp/tools/execute", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestMCPServerHandleToolExecutionNonExistentPlugin(t *testing.T) {
	registry := pipelines.NewPluginRegistry()
	server := NewMCPServer(registry)

	// Non-existent plugin
	requestBody := map[string]any{
		"tool_name": "Nonexistent.plugin",
		"arguments": map[string]any{
			"step_config": map[string]any{
				"name": "test-step",
			},
		},
	}

	bodyBytes, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/mcp/tools/execute", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestMCPServerHandleToolExecutionPluginError(t *testing.T) {
	registry := pipelines.NewPluginRegistry()

	// Register a failing plugin
	failingPlugin := &MockPlugin{pluginType: "Input", pluginName: "failing", shouldFail: true}
	registry.RegisterPlugin(failingPlugin)

	server := NewMCPServer(registry)

	requestBody := map[string]any{
		"tool_name": "Input.failing",
		"arguments": map[string]any{
			"step_config": map[string]any{
				"name": "test-step",
			},
		},
	}

	bodyBytes, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/mcp/tools/execute", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetStringValue(t *testing.T) {
	tests := []struct {
		name     string
		data     map[string]any
		key      string
		expected string
	}{
		{
			name:     "existing string value",
			data:     map[string]any{"key": "value"},
			key:      "key",
			expected: "value",
		},
		{
			name:     "non-string value",
			data:     map[string]any{"key": 123},
			key:      "key",
			expected: "",
		},
		{
			name:     "non-existent key",
			data:     map[string]any{"other": "value"},
			key:      "key",
			expected: "",
		},
		{
			name:     "nil data",
			data:     nil,
			key:      "key",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getStringValue(tt.data, tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetMapValue(t *testing.T) {
	tests := []struct {
		name     string
		data     map[string]any
		key      string
		expected map[string]any
	}{
		{
			name:     "existing map value",
			data:     map[string]any{"key": map[string]any{"nested": "value"}},
			key:      "key",
			expected: map[string]any{"nested": "value"},
		},
		{
			name:     "non-map value",
			data:     map[string]any{"key": "string"},
			key:      "key",
			expected: map[string]any{},
		},
		{
			name:     "non-existent key",
			data:     map[string]any{"other": "value"},
			key:      "key",
			expected: map[string]any{},
		},
		{
			name:     "nil data",
			data:     nil,
			key:      "key",
			expected: map[string]any{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getMapValue(tt.data, tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMCPServerToolDiscoveryEmptyRegistry(t *testing.T) {
	registry := pipelines.NewPluginRegistry()
	server := NewMCPServer(registry)

	req := httptest.NewRequest("GET", "/mcp/tools", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	tools, ok := response["tools"].([]any)
	assert.True(t, ok)
	assert.Len(t, tools, 0) // No plugins registered
}

func TestMCPServerToolExecutionWithComplexArguments(t *testing.T) {
	registry := pipelines.NewPluginRegistry()

	// Register a mock plugin
	mockPlugin := &MockPlugin{pluginType: "Input", pluginName: "test"}
	registry.RegisterPlugin(mockPlugin)

	server := NewMCPServer(registry)

	// Complex request with nested arguments
	requestBody := map[string]any{
		"tool_name": "Input.test",
		"arguments": map[string]any{
			"step_config": map[string]any{
				"name": "complex-step",
				"config": map[string]any{
					"nested_param": map[string]any{
						"sub_param": "sub_value",
						"number":    42,
						"boolean":   true,
					},
					"array_param": []any{"item1", "item2", "item3"},
				},
				"output": "complex-output",
			},
			"context": map[string]any{
				"existing_data": map[string]any{
					"key1": "value1",
					"key2": 123,
				},
				"simple_string": "test",
			},
		},
	}

	bodyBytes, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/mcp/tools/execute", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]any
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response["success"].(bool))
}

func TestMCPServerToolExecutionMissingRequiredFields(t *testing.T) {
	registry := pipelines.NewPluginRegistry()

	// Register a mock plugin first
	mockPlugin := &MockPlugin{pluginType: "Input", pluginName: "test"}
	registry.RegisterPlugin(mockPlugin)

	server := NewMCPServer(registry)

	// Missing step_config name
	requestBody := map[string]any{
		"tool_name": "Input.test",
		"arguments": map[string]any{
			"step_config": map[string]any{
				"config": map[string]any{},
				// Missing "name" field
			},
		},
	}

	bodyBytes, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/mcp/tools/execute", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	// Should still succeed as validation is handled by the plugin
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMCPServerMultiplePlugins(t *testing.T) {
	registry := pipelines.NewPluginRegistry()

	// Register multiple plugins of different types
	inputPlugin := &MockPlugin{pluginType: "Input", pluginName: "api"}
	outputPlugin := &MockPlugin{pluginType: "Output", pluginName: "html"}
	processingPlugin := &MockPlugin{pluginType: "Processing", pluginName: "transform"}

	registry.RegisterPlugin(inputPlugin)
	registry.RegisterPlugin(outputPlugin)
	registry.RegisterPlugin(processingPlugin)

	server := NewMCPServer(registry)

	req := httptest.NewRequest("GET", "/mcp/tools", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	tools, ok := response["tools"].([]any)
	assert.True(t, ok)
	assert.Len(t, tools, 3) // Three plugins registered

	// Check that all plugin types are represented
	toolNames := make([]string, 0, 3)
	for _, tool := range tools {
		toolMap := tool.(map[string]any)
		toolNames = append(toolNames, toolMap["name"].(string))
	}

	assert.Contains(t, toolNames, "Input.api")
	assert.Contains(t, toolNames, "Output.html")
	assert.Contains(t, toolNames, "Processing.transform")
}

func TestMCPServerConcurrentRequests(t *testing.T) {
	registry := pipelines.NewPluginRegistry()

	// Register a mock plugin
	mockPlugin := &MockPlugin{pluginType: "Input", pluginName: "test"}
	registry.RegisterPlugin(mockPlugin)

	server := NewMCPServer(registry)

	// Test concurrent tool discovery requests
	done := make(chan bool, 5)

	for i := 0; i < 5; i++ {
		go func() {
			req := httptest.NewRequest("GET", "/mcp/tools", nil)
			w := httptest.NewRecorder()

			server.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			done <- true
		}()
	}

	// Wait for all requests to complete
	for i := 0; i < 5; i++ {
		<-done
	}
}

func TestMCPServerContentTypeHeaders(t *testing.T) {
	registry := pipelines.NewPluginRegistry()

	// Register a mock plugin for tool execution test
	mockPlugin := &MockPlugin{pluginType: "Input", pluginName: "test"}
	registry.RegisterPlugin(mockPlugin)

	server := NewMCPServer(registry)

	// Test tool discovery
	req := httptest.NewRequest("GET", "/mcp/tools", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	// Test tool execution
	requestBody := map[string]any{
		"tool_name": "Input.test",
		"arguments": map[string]any{
			"step_config": map[string]any{
				"name": "test-step",
			},
		},
	}

	bodyBytes, _ := json.Marshal(requestBody)
	req = httptest.NewRequest("POST", "/mcp/tools/execute", bytes.NewReader(bodyBytes))
	w = httptest.NewRecorder()

	server.ServeHTTP(w, req)

	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
}
