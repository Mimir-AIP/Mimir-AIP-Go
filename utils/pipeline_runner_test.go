package utils

import (
	"context"
	"testing"
	"time"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockPlugin for testing
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
	result.Set(stepConfig.Output, map[string]interface{}{
		"plugin":  m.pluginName,
		"success": true,
	})
	return result, nil
}

func (m *MockPlugin) GetPluginType() string { return m.pluginType }
func (m *MockPlugin) GetPluginName() string { return m.pluginName }
func (m *MockPlugin) ValidateConfig(config map[string]interface{}) error {
	if m.shouldFail {
		return assert.AnError
	}
	return nil
}

func TestExecutePipeline(t *testing.T) {
	tests := []struct {
		name          string
		config        *PipelineConfig
		expectError   bool
		expectSuccess bool
	}{
		{
			name: "successful pipeline execution",
			config: &PipelineConfig{
				Name:    "test-pipeline",
				Enabled: true,
				Steps: []pipelines.StepConfig{
					{
						Name:   "step1",
						Plugin: "Input.test",
						Config: map[string]interface{}{},
						Output: "output1",
					},
					{
						Name:   "step2",
						Plugin: "Output.test",
						Config: map[string]interface{}{},
						Output: "output2",
					},
				},
			},
			expectError:   false,
			expectSuccess: true,
		},
		{
			name: "pipeline with invalid plugin reference",
			config: &PipelineConfig{
				Name:    "invalid-pipeline",
				Enabled: true,
				Steps: []pipelines.StepConfig{
					{
						Name:   "step1",
						Plugin: "invalid", // Missing dot separator
						Config: map[string]interface{}{},
					},
				},
			},
			expectError:   true,
			expectSuccess: false,
		},
		{
			name: "empty pipeline",
			config: &PipelineConfig{
				Name:    "empty-pipeline",
				Enabled: true,
				Steps:   []pipelines.StepConfig{},
			},
			expectError:   false,
			expectSuccess: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := pipelines.NewPluginRegistry()

			// Register mock plugins
			mockInput := &MockPlugin{pluginType: "Input", pluginName: "test"}
			mockOutput := &MockPlugin{pluginType: "Output", pluginName: "test"}
			registry.RegisterPlugin(mockInput)
			registry.RegisterPlugin(mockOutput)

			ctx := context.Background()
			result, err := ExecutePipelineWithRegistry(ctx, tt.config, registry)

			if tt.expectError {
				assert.Error(t, err)
				// For invalid plugin reference, ExecutePipelineWithRegistry returns a result with error
				if result != nil {
					assert.False(t, result.Success)
					assert.NotEmpty(t, result.Error)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.expectSuccess, result.Success)
			}
		})
	}
}

func TestExecuteStep(t *testing.T) {
	registry := pipelines.NewPluginRegistry()
	mockPlugin := &MockPlugin{pluginType: "Input", pluginName: "test"}
	registry.RegisterPlugin(mockPlugin)

	tests := []struct {
		name        string
		stepConfig  pipelines.StepConfig
		expectError bool
	}{
		{
			name: "valid step execution",
			stepConfig: pipelines.StepConfig{
				Name:   "test-step",
				Plugin: "Input.test",
				Config: map[string]interface{}{},
				Output: "test-output",
			},
			expectError: false,
		},
		{
			name: "invalid plugin reference",
			stepConfig: pipelines.StepConfig{
				Name:   "invalid-step",
				Plugin: "invalid",
				Config: map[string]interface{}{},
			},
			expectError: true,
		},
		{
			name: "nonexistent plugin",
			stepConfig: pipelines.StepConfig{
				Name:   "nonexistent-step",
				Plugin: "Nonexistent.plugin",
				Config: map[string]interface{}{},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			globalContext := pipelines.NewPluginContext()

			result, err := executeStep(ctx, registry, tt.stepConfig, globalContext)

			if tt.expectError {
				assert.Error(t, err)
				assert.NotNil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}

func TestExecutePipelineWithRegistry_ContextPassing(t *testing.T) {
	registry := pipelines.NewPluginRegistry()

	// Create a plugin that sets data in context
	contextPlugin := &MockPlugin{pluginType: "Input", pluginName: "context"}
	registry.RegisterPlugin(contextPlugin)

	config := &PipelineConfig{
		Name:    "context-test",
		Enabled: true,
		Steps: []pipelines.StepConfig{
			{
				Name:   "step1",
				Plugin: "Input.context",
				Config: map[string]interface{}{},
				Output: "step1_output",
			},
		},
	}

	ctx := context.Background()
	result, err := ExecutePipelineWithRegistry(ctx, config, registry)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success)

	// Check that step output was merged into global context
	value, exists := result.Context.Get("step1_output")
	assert.True(t, exists)
	assert.NotNil(t, value)
}

func TestExecutePipelineWithRegistry_StepFailure(t *testing.T) {
	registry := pipelines.NewPluginRegistry()

	// Create a plugin that will fail
	failingPlugin := &MockPlugin{pluginType: "Input", pluginName: "failing", shouldFail: true}
	registry.RegisterPlugin(failingPlugin)

	config := &PipelineConfig{
		Name:    "failing-pipeline",
		Enabled: true,
		Steps: []pipelines.StepConfig{
			{
				Name:   "step1",
				Plugin: "Input.failing",
				Config: map[string]interface{}{},
			},
		},
	}

	ctx := context.Background()
	result, err := ExecutePipelineWithRegistry(ctx, config, registry)

	assert.Error(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "step 1 (step1) failed")
}

func TestExecutePipelineWithRegistry_ConfigurationValidation(t *testing.T) {
	registry := pipelines.NewPluginRegistry()

	// Create a plugin that fails validation
	invalidPlugin := &MockPlugin{pluginType: "Input", pluginName: "invalid", shouldFail: true}
	registry.RegisterPlugin(invalidPlugin)

	config := &PipelineConfig{
		Name:    "validation-fail",
		Enabled: true,
		Steps: []pipelines.StepConfig{
			{
				Name:   "step1",
				Plugin: "Input.invalid",
				Config: map[string]interface{}{},
			},
		},
	}

	ctx := context.Background()
	result, err := ExecutePipelineWithRegistry(ctx, config, registry)

	assert.Error(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "configuration validation failed")
}

func TestRunPipeline(t *testing.T) {
	tests := []struct {
		name        string
		pipeline    string
		expectError bool
	}{
		{
			name:        "empty pipeline path",
			pipeline:    "",
			expectError: true,
		},
		{
			name:        "non-yaml file",
			pipeline:    "test.txt",
			expectError: true,
		},
		{
			name:        "valid yaml extension",
			pipeline:    "test.yaml",
			expectError: true, // Will fail because file doesn't exist
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := RunPipeline(tt.pipeline)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPipelineExecutionResult(t *testing.T) {
	result := &PipelineExecutionResult{
		Success:    true,
		Context:    pipelines.NewPluginContext(),
		ExecutedAt: time.Now().Format(time.RFC3339),
	}

	assert.True(t, result.Success)
	assert.NotNil(t, result.Context)
	assert.NotEmpty(t, result.ExecutedAt)

	// Test error case
	errorResult := &PipelineExecutionResult{
		Success: false,
		Error:   "test error",
	}

	assert.False(t, errorResult.Success)
	assert.Equal(t, "test error", errorResult.Error)
}

func TestRegisterDefaultPlugins(t *testing.T) {
	registry := pipelines.NewPluginRegistry()

	err := registerDefaultPlugins(registry)
	assert.NoError(t, err)

	// Check that plugins were registered
	_, err = registry.GetPlugin("Input", "api")
	assert.NoError(t, err)

	_, err = registry.GetPlugin("Output", "html")
	assert.NoError(t, err)
}

func TestRealAPIPlugin(t *testing.T) {
	plugin := &RealAPIPlugin{}

	// Test plugin metadata
	assert.Equal(t, "Input", plugin.GetPluginType())
	assert.Equal(t, "api", plugin.GetPluginName())

	// Test validation
	t.Run("valid config", func(t *testing.T) {
		config := map[string]interface{}{
			"url": "https://example.com",
		}
		err := plugin.ValidateConfig(config)
		assert.NoError(t, err)
	})

	t.Run("missing url", func(t *testing.T) {
		config := map[string]interface{}{}
		err := plugin.ValidateConfig(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "url is required")
	})
}

func TestMockHTMLPlugin(t *testing.T) {
	plugin := &MockHTMLPlugin{}

	// Test plugin metadata
	assert.Equal(t, "Output", plugin.GetPluginType())
	assert.Equal(t, "html", plugin.GetPluginName())

	// Test validation (always succeeds for mock)
	config := map[string]interface{}{}
	err := plugin.ValidateConfig(config)
	assert.NoError(t, err)

	// Test execution
	ctx := context.Background()
	stepConfig := pipelines.StepConfig{
		Name:   "test-step",
		Plugin: "Output.html",
		Config: config,
	}
	globalContext := pipelines.NewPluginContext()

	result, err := plugin.ExecuteStep(ctx, stepConfig, globalContext)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Check that mock result was set
	value, exists := result.Get("report_generated")
	assert.True(t, exists)
	// The MockHTMLPlugin sets a boolean, but PluginContext auto-wraps it
	// Check if it's a boolean directly or wrapped in JSONData
	if boolVal, ok := value.(bool); ok {
		assert.True(t, boolVal)
	} else if jsonVal, ok := value.(map[string]interface{}); ok {
		assert.True(t, jsonVal["value"].(bool))
	}
}
