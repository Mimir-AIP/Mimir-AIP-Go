package tests

import (
	"context"
	"fmt"
	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
	MockAIModel "github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/AI/MockAIModel"
	"testing"
)

// TestPluginRegistry tests the plugin registry functionality
func TestPluginRegistry(t *testing.T) {
	registry := pipelines.NewPluginRegistry()
	// Test registering a plugin
	mockPlugin := NewMockPlugin("test_plugin", "Data_Processing", false)
	err := registry.RegisterPlugin(mockPlugin)
	if err != nil {
		t.Fatalf("Failed to register plugin: %v", err)
	}
	// Test retrieving the plugin
	retrievedPlugin, err := registry.GetPlugin("Data_Processing", "test_plugin")
	if err != nil {
		t.Fatalf("Failed to get plugin: %v", err)
	}
	if retrievedPlugin.GetPluginName() != "test_plugin" {
		t.Errorf("Expected plugin name 'test_plugin', got '%s'", retrievedPlugin.GetPluginName())
	}
	if retrievedPlugin.GetPluginType() != "Data_Processing" {
		t.Errorf("Expected plugin type 'Data_Processing', got '%s'", retrievedPlugin.GetPluginType())
	}
}

// TestPluginRegistry_DuplicateRegistration tests registering duplicate plugins
func TestPluginRegistry_DuplicateRegistration(t *testing.T) {
	registry := pipelines.NewPluginRegistry()
	mockPlugin1 := NewMockPlugin("duplicate_plugin", "Data_Processing", false)
	mockPlugin2 := NewMockPlugin("duplicate_plugin", "Data_Processing", false)
	// Register first plugin
	err := registry.RegisterPlugin(mockPlugin1)
	if err != nil {
		t.Fatalf("Failed to register first plugin: %v", err)
	}
	// Try to register duplicate plugin
	err = registry.RegisterPlugin(mockPlugin2)
	if err == nil {
		t.Fatal("Expected error when registering duplicate plugin, got nil")
	}
}

// TestPluginRegistry_GetPluginsByType tests getting plugins by type
func TestPluginRegistry_GetPluginsByType(t *testing.T) {
	registry := pipelines.NewPluginRegistry()
	// Register multiple plugins of different types
	registry.RegisterPlugin(NewMockPlugin("data_plugin1", "Data_Processing", false))
	registry.RegisterPlugin(NewMockPlugin("data_plugin2", "Data_Processing", false))
	registry.RegisterPlugin(NewMockPlugin("input_plugin1", "Input", false))
	// Get Data_Processing plugins
	dataPlugins := registry.GetPluginsByType("Data_Processing")
	if len(dataPlugins) != 2 {
		t.Fatalf("Expected 2 Data_Processing plugins, got %d", len(dataPlugins))
	}
	// Get Input plugins
	inputPlugins := registry.GetPluginsByType("Input")
	if len(inputPlugins) != 1 {
		t.Fatalf("Expected 1 Input plugin, got %d", len(inputPlugins))
	}
	// Get non-existent type
	emptyPlugins := registry.GetPluginsByType("NonExistent")
	if len(emptyPlugins) != 0 {
		t.Fatalf("Expected 0 plugins for non-existent type, got %d", len(emptyPlugins))
	}
}

// TestPluginRegistry_GetAllPlugins tests getting all registered plugins
func TestPluginRegistry_GetAllPlugins(t *testing.T) {
	registry := pipelines.NewPluginRegistry()
	// Register plugins of different types
	registry.RegisterPlugin(NewMockPlugin("data_plugin", "Data_Processing", false))
	registry.RegisterPlugin(NewMockPlugin("input_plugin", "Input", false))
	registry.RegisterPlugin(NewMockPlugin("output_plugin", "Output", false))
	allPlugins := registry.GetAllPlugins()
	// Check that we have 3 types
	if len(allPlugins) != 3 {
		t.Fatalf("Expected 3 plugin types, got %d", len(allPlugins))
	}
	// Check each type has the correct count
	if len(allPlugins["Data_Processing"]) != 1 {
		t.Errorf("Expected 1 Data_Processing plugin, got %d", len(allPlugins["Data_Processing"]))
	}
	if len(allPlugins["Input"]) != 1 {
		t.Errorf("Expected 1 Input plugin, got %d", len(allPlugins["Input"]))
	}
	if len(allPlugins["Output"]) != 1 {
		t.Errorf("Expected 1 Output plugin, got %d", len(allPlugins["Output"]))
	}
}

// TestPluginRegistry_ListPluginTypes tests listing all plugin types
func TestPluginRegistry_ListPluginTypes(t *testing.T) {
	registry := pipelines.NewPluginRegistry()
	// Register plugins of different types
	registry.RegisterPlugin(NewMockPlugin("data_plugin", "Data_Processing", false))
	registry.RegisterPlugin(NewMockPlugin("input_plugin", "Input", false))
	registry.RegisterPlugin(NewMockPlugin("ai_plugin", "AIModels", false))
	types := registry.ListPluginTypes()
	// Check that we have 3 types
	if len(types) != 3 {
		t.Fatalf("Expected 3 plugin types, got %d", len(types))
	}
	// Check that all expected types are present
	expectedTypes := map[string]bool{
		"Data_Processing": false,
		"Input":           false,
		"AIModels":        false,
	}
	for _, pluginType := range types {
		if _, exists := expectedTypes[pluginType]; exists {
			expectedTypes[pluginType] = true
		} else {
			t.Errorf("Unexpected plugin type: %s", pluginType)
		}
	}
	// Check that all expected types were found
	for pluginType, found := range expectedTypes {
		if !found {
			t.Errorf("Expected plugin type not found: %s", pluginType)
		}
	}
}

// TestPluginValidation tests plugin configuration validation
func TestPluginValidation(t *testing.T) {
	mockPlugin := NewMockPlugin("validation_plugin", "Data_Processing", false)
	// Test valid configuration
	err := mockPlugin.ValidateConfig(map[string]interface{}{})
	if err != nil {
		t.Fatalf("Valid configuration should not fail: %v", err)
	}
	// Test invalid configuration
	err = mockPlugin.ValidateConfig(map[string]interface{}{"invalid": true})
	if err == nil {
		t.Fatal("Invalid configuration should fail validation")
	}
}

// --- Edge Case Tests for Plugin Execution ---

// TestPluginExecution_MalformedConfig tests execution with a malformed config
func TestPluginExecution_MalformedConfig(t *testing.T) {
	plugin := MockAIModel.NewMockEchoModel()
	// Missing "input" field
	config := map[string]interface{}{"temperature": 0.5}
	err := plugin.ValidateConfig(config)
	if err == nil {
		t.Error("Expected validation error for missing input, got nil")
	}
	// Try execution
	stepConfig := pipelines.StepConfig{
		Name:   "Malformed Config Step",
		Plugin: "AIModels.mock_echo",
		Config: config,
		Output: "output",
	}
	_, execErr := plugin.ExecuteStep(context.Background(), stepConfig, pipelines.NewPluginContext())
	if execErr == nil {
		t.Error("Expected execution error for missing input, got nil")
	}
}

// panicPlugin simulates a plugin that panics
type panicPlugin struct{}

func (p *panicPlugin) ExecuteStep(ctx context.Context, stepConfig pipelines.StepConfig, globalContext *pipelines.PluginContext) (*pipelines.PluginContext, error) {
	panic("simulated plugin panic")
}
func (p *panicPlugin) GetPluginType() string                              { return "Test" }
func (p *panicPlugin) GetPluginName() string                              { return "panic_plugin" }
func (p *panicPlugin) ValidateConfig(config map[string]interface{}) error { return nil }

// TestPluginExecution_Panic tests that plugin panics are handled gracefully
func TestPluginExecution_Panic(t *testing.T) {
	registry := pipelines.NewPluginRegistry()
	registry.RegisterPlugin(&panicPlugin{})
	config := &pipelines.StepConfig{
		Name:   "Panic Step",
		Plugin: "Test.panic_plugin",
		Config: map[string]interface{}{},
		Output: "output",
	}
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic to be recovered, but it was not")
		}
	}()
	// Direct call to ExecuteStep to simulate pipeline execution
	plugin, _ := registry.GetPlugin("Test", "panic_plugin")
	plugin.ExecuteStep(context.Background(), *config, pipelines.NewPluginContext())
}

// TestPluginExecution_RealPlugin tests successful execution of a real plugin
func TestPluginExecution_RealPlugin(t *testing.T) {
	plugin := MockAIModel.NewMockEchoModel()
	config := map[string]interface{}{
		"input":       "Hello, world!",
		"temperature": 0.5,
		"max_tokens":  10,
	}
	stepConfig := pipelines.StepConfig{
		Name:   "Echo Step",
		Plugin: "AIModels.mock_echo",
		Config: config,
		Output: "output",
	}
	result, err := plugin.ExecuteStep(context.Background(), stepConfig, pipelines.NewPluginContext())
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if result == nil {
		t.Fatal("Expected result, got nil")
	}
	val, exists := result.Get("output")
	if !exists {
		t.Error("Expected output in context")
	}
	respMap, ok := val.(map[string]interface{})
	if !ok || respMap["response"] == "" {
		t.Error("Expected response in output map")
	}
}

// --- End Edge Case Tests ---

// --- LLM/Agentic Workflow Tests ---

// MockOpenAIPlugin simulates OpenAIPlugin for testing
// It returns a fixed response or error based on config
// (In real tests, use testify/mock or httpmock for API calls)
type MockOpenAIPlugin struct {
	ShouldError bool
	ErrorType   string // "api_key", "api_error", "timeout"
}

func (p *MockOpenAIPlugin) ExecuteStep(ctx context.Context, stepConfig pipelines.StepConfig, globalContext *pipelines.PluginContext) (*pipelines.PluginContext, error) {
	if p.ShouldError {
		switch p.ErrorType {
		case "api_key":
			return nil, fmt.Errorf("OpenAI API key is required")
		case "api_error":
			return nil, fmt.Errorf("API error: 500 - Internal Server Error")
		case "timeout":
			return nil, fmt.Errorf("API request failed: timeout")
		}
	}
	// Simulate normal response
	result := pipelines.NewPluginContext()
	result.Set(stepConfig.Output, map[string]interface{}{
		"content":       "This is a mock LLM response.",
		"model":         "gpt-3.5-turbo",
		"usage":         map[string]interface{}{"tokens": 42},
		"finish_reason": "stop",
		"request_id":    "mock-id-123",
		"timestamp":     "2025-11-25T10:00:00Z",
	})
	return result, nil
}
func (p *MockOpenAIPlugin) GetPluginType() string                              { return "AIModels" }
func (p *MockOpenAIPlugin) GetPluginName() string                              { return "openai" }
func (p *MockOpenAIPlugin) ValidateConfig(config map[string]interface{}) error { return nil }

// TestOpenAIPlugin_ChatSuccess tests successful chat completion
func TestOpenAIPlugin_ChatSuccess(t *testing.T) {
	plugin := &MockOpenAIPlugin{}
	config := map[string]interface{}{
		"operation": "chat",
		"messages": []interface{}{
			map[string]interface{}{"role": "user", "content": "Hello!"},
		},
		"api_key": "test-key",
	}
	stepConfig := pipelines.StepConfig{
		Name:   "Chat Step",
		Plugin: "AIModels.openai",
		Config: config,
		Output: "output",
	}
	result, err := plugin.ExecuteStep(context.Background(), stepConfig, pipelines.NewPluginContext())
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	val, exists := result.Get("output")
	if !exists {
		t.Error("Expected output in context")
	}
	respMap, ok := val.(map[string]interface{})
	if !ok || respMap["content"] == "" {
		t.Error("Expected content in output map")
	}
}

// TestOpenAIPlugin_MissingAPIKey tests missing API key error
func TestOpenAIPlugin_MissingAPIKey(t *testing.T) {
	plugin := &MockOpenAIPlugin{ShouldError: true, ErrorType: "api_key"}
	config := map[string]interface{}{
		"operation": "chat",
		"messages": []interface{}{
			map[string]interface{}{"role": "user", "content": "Hello!"},
		},
	}
	stepConfig := pipelines.StepConfig{
		Name:   "Chat Step",
		Plugin: "AIModels.openai",
		Config: config,
		Output: "output",
	}
	_, err := plugin.ExecuteStep(context.Background(), stepConfig, pipelines.NewPluginContext())
	if err == nil || err.Error() != "OpenAI API key is required" {
		t.Errorf("Expected API key error, got: %v", err)
	}
}

// TestOpenAIPlugin_APIError tests API error handling
func TestOpenAIPlugin_APIError(t *testing.T) {
	plugin := &MockOpenAIPlugin{ShouldError: true, ErrorType: "api_error"}
	config := map[string]interface{}{
		"operation": "chat",
		"messages": []interface{}{
			map[string]interface{}{"role": "user", "content": "Hello!"},
		},
		"api_key": "test-key",
	}
	stepConfig := pipelines.StepConfig{
		Name:   "Chat Step",
		Plugin: "AIModels.openai",
		Config: config,
		Output: "output",
	}
	_, err := plugin.ExecuteStep(context.Background(), stepConfig, pipelines.NewPluginContext())
	if err == nil || err.Error() != "API error: 500 - Internal Server Error" {
		t.Errorf("Expected API error, got: %v", err)
	}
}

// TestOpenAIPlugin_Timeout tests timeout error handling
func TestOpenAIPlugin_Timeout(t *testing.T) {
	plugin := &MockOpenAIPlugin{ShouldError: true, ErrorType: "timeout"}
	config := map[string]interface{}{
		"operation": "chat",
		"messages": []interface{}{
			map[string]interface{}{"role": "user", "content": "Hello!"},
		},
		"api_key": "test-key",
	}
	stepConfig := pipelines.StepConfig{
		Name:   "Chat Step",
		Plugin: "AIModels.openai",
		Config: config,
		Output: "output",
	}
	_, err := plugin.ExecuteStep(context.Background(), stepConfig, pipelines.NewPluginContext())
	if err == nil || err.Error() != "API request failed: timeout" {
		t.Errorf("Expected timeout error, got: %v", err)
	}
}

// TestAgenticChain_Success simulates a multi-step agentic workflow (LLM + transform)
func TestAgenticChain_Success(t *testing.T) {
	llmPlugin := &MockOpenAIPlugin{}
	transformPlugin := MockAIModel.NewMockEchoModel()
	// Step 1: LLM response
	llmConfig := map[string]interface{}{
		"operation": "chat",
		"messages": []interface{}{
			map[string]interface{}{"role": "user", "content": "Summarize AI."},
		},
		"api_key": "test-key",
	}
	llmStep := pipelines.StepConfig{
		Name:   "LLM Step",
		Plugin: "AIModels.openai",
		Config: llmConfig,
		Output: "llm_output",
	}
	llmResult, err := llmPlugin.ExecuteStep(context.Background(), llmStep, pipelines.NewPluginContext())
	if err != nil {
		t.Fatalf("LLM step failed: %v", err)
	}
	llmVal, _ := llmResult.Get("llm_output")
	llmContent := llmVal.(map[string]interface{})["content"].(string)
	// Step 2: Transform response
	transformConfig := map[string]interface{}{
		"input":       llmContent,
		"temperature": 0.5,
		"max_tokens":  10,
	}
	transformStep := pipelines.StepConfig{
		Name:   "Transform Step",
		Plugin: "AIModels.mock_echo",
		Config: transformConfig,
		Output: "final_output",
	}
	finalResult, err := transformPlugin.ExecuteStep(context.Background(), transformStep, pipelines.NewPluginContext())
	if err != nil {
		t.Fatalf("Transform step failed: %v", err)
	}
	val, exists := finalResult.Get("final_output")
	if !exists {
		t.Error("Expected final_output in context")
	}
	respMap, ok := val.(map[string]interface{})
	if !ok || respMap["response"] == "" {
		t.Error("Expected response in final output map")
	}
}

// --- End LLM/Agentic Workflow Tests ---

// networkErrorPlugin simulates a plugin that always returns a network error
type networkErrorPlugin struct{}

func (p *networkErrorPlugin) ExecuteStep(ctx context.Context, stepConfig pipelines.StepConfig, globalContext *pipelines.PluginContext) (*pipelines.PluginContext, error) {
	return nil, fmt.Errorf("network error: unable to reach endpoint")
}
func (p *networkErrorPlugin) GetPluginType() string                              { return "Test" }
func (p *networkErrorPlugin) GetPluginName() string                              { return "network_error_plugin" }
func (p *networkErrorPlugin) ValidateConfig(config map[string]interface{}) error { return nil }

// TestPluginExecution_NetworkError tests network error handling
func TestPluginExecution_NetworkError(t *testing.T) {
	registry := pipelines.NewPluginRegistry()
	registry.RegisterPlugin(&networkErrorPlugin{})
	stepConfig := pipelines.StepConfig{
		Name:   "Network Error Step",
		Plugin: "Test.network_error_plugin",
		Config: map[string]interface{}{},
		Output: "output",
	}
	plugin, _ := registry.GetPlugin("Test", "network_error_plugin")
	_, err := plugin.ExecuteStep(context.Background(), stepConfig, pipelines.NewPluginContext())
	if err == nil || err.Error() != "network error: unable to reach endpoint" {
		t.Errorf("Expected network error, got: %v", err)
	}
}

// TestPluginExecution_InvalidData tests handling of corrupted/invalid data
func TestPluginExecution_InvalidData(t *testing.T) {
	plugin := MockAIModel.NewMockEchoModel()
	// Pass an integer instead of string for "input"
	config := map[string]interface{}{"input": 12345}
	stepConfig := pipelines.StepConfig{
		Name:   "Invalid Data Step",
		Plugin: "AIModels.mock_echo",
		Config: config,
		Output: "output",
	}
	_, err := plugin.ExecuteStep(context.Background(), stepConfig, pipelines.NewPluginContext())
	if err == nil {
		t.Error("Expected error for invalid input type, got nil")
	}
}
