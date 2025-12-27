package tests

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
	"github.com/Mimir-AIP/Mimir-AIP-Go/utils"
)

// MockPlugin is a test implementation of the BasePlugin interface
type MockPlugin struct {
	name          string
	pluginType    string
	shouldFail    bool
	executionTime time.Duration
	result        any
}

func NewMockPlugin(name, pluginType string, shouldFail bool) *MockPlugin {
	return &MockPlugin{
		name:       name,
		pluginType: pluginType,
		shouldFail: shouldFail,
		result:     map[string]any{"status": "success"},
	}
}

func (mp *MockPlugin) ExecuteStep(ctx context.Context, stepConfig pipelines.StepConfig, globalContext *pipelines.PluginContext) (*pipelines.PluginContext, error) {
	if mp.executionTime > 0 {
		select {
		case <-time.After(mp.executionTime):
			// Execution completed normally
		case <-ctx.Done():
			// Context was cancelled (timeout or cancellation)
			return pipelines.NewPluginContext(), fmt.Errorf("PLUGIN_EXECUTION_TIMEOUT: Mock plugin execution timed out")
		}
	}

	if mp.shouldFail {
		return pipelines.NewPluginContext(), fmt.Errorf("PLUGIN_EXECUTION_FAILED: Mock plugin execution failed")
	}

	result := pipelines.NewPluginContext()
	result.Set(stepConfig.Output, mp.result)
	return result, nil
}

func (mp *MockPlugin) GetPluginType() string {
	return mp.pluginType
}

func (mp *MockPlugin) GetPluginName() string {
	return mp.name
}

func (mp *MockPlugin) ValidateConfig(config map[string]any) error {
	if config["invalid"] != nil {
		return fmt.Errorf("VALIDATION_ERROR: Invalid configuration")
	}
	return nil
}

func (mp *MockPlugin) GetInputSchema() map[string]any {
	return map[string]any{}
}

func TestPipelineExecution_Success(t *testing.T) {
	// Create a simple pipeline configuration
	config := &utils.PipelineConfig{
		Name: "Test Pipeline",
		Steps: []pipelines.StepConfig{
			{
				Name:   "Step 1",
				Plugin: "Data_Processing.mock",
				Config: map[string]any{},
				Output: "step1_output",
			},
			{
				Name:   "Step 2",
				Plugin: "Data_Processing.mock",
				Config: map[string]any{},
				Output: "step2_output",
			},
		},
	}

	// Register mock plugins
	pluginRegistry := pipelines.NewPluginRegistry()
	mockPlugin := NewMockPlugin("mock", "Data_Processing", false)
	_ = pluginRegistry.RegisterPlugin(mockPlugin)

	// Execute pipeline
	result, err := utils.ExecutePipelineWithRegistry(context.Background(), config, pluginRegistry)

	// Assertions
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	if !result.Success {
		t.Fatal("Expected pipeline to succeed")
	}

	if result.ExecutedAt == "" {
		t.Fatal("Expected execution timestamp")
	}

	// Check that context contains expected outputs
	if _, exists := result.Context.Get("step1_output"); !exists {
		t.Fatal("Expected step1_output in context")
	}
	if _, exists := result.Context.Get("step2_output"); !exists {
		t.Fatal("Expected step2_output in context")
	}
}

func TestPipelineExecution_Failure(t *testing.T) {
	// Create a pipeline configuration with a failing step
	config := &utils.PipelineConfig{
		Name: "Test Pipeline",
		Steps: []pipelines.StepConfig{
			{
				Name:   "Step 1",
				Plugin: "Data_Processing.mock",
				Config: map[string]any{},
				Output: "step1_output",
			},
			{
				Name:   "Failing Step",
				Plugin: "Data_Processing.mock_fail",
				Config: map[string]any{},
				Output: "failing_output",
			},
		},
	}

	// Register plugins
	pluginRegistry := pipelines.NewPluginRegistry()
	_ = pluginRegistry.RegisterPlugin(NewMockPlugin("mock", "Data_Processing", false))
	_ = pluginRegistry.RegisterPlugin(NewMockPlugin("mock_fail", "Data_Processing", true))

	// Execute pipeline
	result, err := utils.ExecutePipelineWithRegistry(context.Background(), config, pluginRegistry)

	// Assertions - now expects err == nil with result.Success == false
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	if result.Success {
		t.Fatal("Expected pipeline to fail")
	}

	if result.Error == "" {
		t.Fatal("Expected error message")
	}
}

func TestPipelineExecution_Timeout(t *testing.T) {
	// Create a pipeline with a slow step
	slowPlugin := NewMockPlugin("slow_mock", "Data_Processing", false)
	slowPlugin.executionTime = 2 * time.Second

	config := &utils.PipelineConfig{
		Name: "Test Pipeline",
		Steps: []pipelines.StepConfig{
			{
				Name:   "Slow Step",
				Plugin: "Data_Processing.slow_mock",
				Config: map[string]any{},
				Output: "slow_output",
			},
		},
	}

	// Register plugin
	pluginRegistry := pipelines.NewPluginRegistry()
	_ = pluginRegistry.RegisterPlugin(slowPlugin)

	// Execute pipeline with timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	result, err := utils.ExecutePipelineWithRegistry(ctx, config, pluginRegistry)

	// Assertions - now expects err == nil with result.Success == false
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	if result.Success {
		t.Fatal("Expected pipeline to fail due to timeout")
	}
}

func TestPipelineExecution_ContextPropagation(t *testing.T) {
	// Create a pipeline that passes data between steps
	config := &utils.PipelineConfig{
		Name: "Test Pipeline",
		Steps: []pipelines.StepConfig{
			{
				Name:   "Generate Data",
				Plugin: "Data_Processing.data_generator",
				Config: map[string]any{
					"data": "test_data",
				},
				Output: "generated_data",
			},
			{
				Name:   "Process Data",
				Plugin: "Data_Processing.data_processor",
				Config: map[string]any{},
				Output: "processed_data",
			},
		},
	}

	// Create plugins that use context
	dataGenerator := &MockPlugin{
		name:       "data_generator",
		pluginType: "Data_Processing",
		shouldFail: false,
		result:     "test_data",
	}

	dataProcessor := &MockPlugin{
		name:       "data_processor",
		pluginType: "Data_Processing",
		shouldFail: false,
		result:     "processed_test_data",
	}

	// Register plugins
	pluginRegistry := pipelines.NewPluginRegistry()
	_ = pluginRegistry.RegisterPlugin(dataGenerator)
	_ = pluginRegistry.RegisterPlugin(dataProcessor)

	// Execute pipeline
	result, err := utils.ExecutePipelineWithRegistry(context.Background(), config, pluginRegistry)

	// Assertions
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !result.Success {
		t.Fatal("Expected pipeline to succeed")
	}

	// Check that data was passed between steps
	if _, exists := result.Context.Get("generated_data"); !exists {
		t.Fatal("Expected generated_data in context")
	}
	if _, exists := result.Context.Get("processed_data"); !exists {
		t.Fatal("Expected processed_data in context")
	}
}

func TestPipelineExecution_ParallelSteps(t *testing.T) {
	// Create a pipeline with parallel steps
	config := &utils.PipelineConfig{
		Name: "Test Pipeline",
		Steps: []pipelines.StepConfig{
			{
				Name:   "Parallel Step 1",
				Plugin: "Data_Processing.mock",
				Config: map[string]any{},
				Output: "output1",
			},
			{
				Name:   "Parallel Step 2",
				Plugin: "Data_Processing.mock",
				Config: map[string]any{},
				Output: "output2",
			},
		},
	}

	// Register plugin
	pluginRegistry := pipelines.NewPluginRegistry()
	_ = pluginRegistry.RegisterPlugin(NewMockPlugin("mock", "Data_Processing", false))

	// Execute pipeline
	result, err := utils.ExecutePipelineWithRegistry(context.Background(), config, pluginRegistry)

	// Assertions
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !result.Success {
		t.Fatal("Expected pipeline to succeed")
	}

	// Check that both outputs are present
	if _, exists := result.Context.Get("output1"); !exists {
		t.Fatal("Expected output1 in context")
	}
	if _, exists := result.Context.Get("output2"); !exists {
		t.Fatal("Expected output2 in context")
	}
}

func TestPipelineExecution_ConfigurationValidation(t *testing.T) {
	// Create a pipeline with invalid configuration
	config := &utils.PipelineConfig{
		Name: "Test Pipeline",
		Steps: []pipelines.StepConfig{
			{
				Name:   "Invalid Step",
				Plugin: "Data_Processing.mock",
				Config: map[string]any{
					"invalid": true,
				},
				Output: "invalid_output",
			},
		},
	}

	// Register plugin
	pluginRegistry := pipelines.NewPluginRegistry()
	_ = pluginRegistry.RegisterPlugin(NewMockPlugin("mock", "Data_Processing", false))

	// Execute pipeline
	result, err := utils.ExecutePipelineWithRegistry(context.Background(), config, pluginRegistry)

	// Assertions - now expects err == nil with result.Success == false
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	if result.Success {
		t.Fatal("Expected pipeline to fail due to validation error")
	}
}
