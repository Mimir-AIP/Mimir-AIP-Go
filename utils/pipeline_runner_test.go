package utils

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecutePipeline_Success(t *testing.T) {
	// Create a temp CSV file
	tmpDir := t.TempDir()
	csvFile := filepath.Join(tmpDir, "test.csv")
	err := os.WriteFile(csvFile, []byte("name,age\nAlice,30\nBob,25"), 0644)
	require.NoError(t, err)

	// Create pipeline config
	config := &PipelineConfig{
		Name:    "test-pipeline",
		Enabled: true,
		Steps: []pipelines.StepConfig{
			{
				Name:   "read-csv",
				Plugin: "Input.csv",
				Config: map[string]any{
					"file_path":   csvFile,
					"has_headers": true,
				},
				Output: "csv_data",
			},
		},
	}

	// Execute pipeline
	ctx := context.Background()
	result, err := ExecutePipeline(ctx, config)

	// Assert
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Empty(t, result.Error)
	assert.NotNil(t, result.Context)

	// Verify CSV data was parsed
	csvData, exists := result.Context.Get("csv_data")
	assert.True(t, exists, "csv_data should exist in context")
	assert.NotNil(t, csvData)
}

func TestExecutePipeline_MissingPlugin(t *testing.T) {
	config := &PipelineConfig{
		Name:    "test-pipeline",
		Enabled: true,
		Steps: []pipelines.StepConfig{
			{
				Name:   "bad-step",
				Plugin: "Input.nonexistent",
				Config: map[string]any{},
				Output: "data",
			},
		},
	}

	ctx := context.Background()
	result, err := ExecutePipeline(ctx, config)

	// Should return result with Success=false, not error
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "failed to get plugin")
}

func TestExecutePipeline_InvalidPluginFormat(t *testing.T) {
	config := &PipelineConfig{
		Name:    "test-pipeline",
		Enabled: true,
		Steps: []pipelines.StepConfig{
			{
				Name:   "bad-step",
				Plugin: "invalid-format", // Missing dot separator
				Config: map[string]any{},
				Output: "data",
			},
		},
	}

	ctx := context.Background()
	result, err := ExecutePipeline(ctx, config)

	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "invalid plugin reference format")
}

func TestExecutePipelineWithRegistry_Success(t *testing.T) {
	tmpDir := t.TempDir()
	csvFile := filepath.Join(tmpDir, "test.csv")
	err := os.WriteFile(csvFile, []byte("name,age\nAlice,30"), 0644)
	require.NoError(t, err)

	// Create custom registry
	registry := pipelines.NewPluginRegistry()
	csvPlugin := &MockCSVPlugin{}
	err = registry.RegisterPlugin(csvPlugin)
	require.NoError(t, err)

	config := &PipelineConfig{
		Name: "test-pipeline",
		Steps: []pipelines.StepConfig{
			{
				Name:   "read-csv",
				Plugin: "Input.mockcsv",
				Config: map[string]any{
					"file_path": csvFile,
				},
				Output: "csv_data",
			},
		},
	}

	ctx := context.Background()
	result, err := ExecutePipelineWithRegistry(ctx, config, registry)

	require.NoError(t, err)
	assert.True(t, result.Success)
}

func TestRunPipeline_Success(t *testing.T) {
	tmpDir := t.TempDir()
	pipelineFile := filepath.Join(tmpDir, "pipeline.yaml")
	csvFile := filepath.Join(tmpDir, "data.csv")

	// Create CSV file
	err := os.WriteFile(csvFile, []byte("name,value\nTest,100"), 0644)
	require.NoError(t, err)

	// Create pipeline YAML
	pipelineYAML := `name: test-pipeline
enabled: true
steps:
  - name: read-csv
    plugin: Input.csv
    config:
      file_path: ` + csvFile + `
      has_headers: true
    output: csv_data
`
	err = os.WriteFile(pipelineFile, []byte(pipelineYAML), 0644)
	require.NoError(t, err)

	err = RunPipeline(pipelineFile)
	assert.NoError(t, err)
}

func TestRunPipeline_InvalidFile(t *testing.T) {
	// Test empty path
	err := RunPipeline("")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "pipeline path cannot be empty")

	// Test non-yaml file
	err = RunPipeline("test.txt")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must end with .yaml")
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
}

// MockCSVPlugin is a test double for the CSV plugin
type MockCSVPlugin struct{}

func (p *MockCSVPlugin) ExecuteStep(ctx context.Context, stepConfig pipelines.StepConfig, globalContext *pipelines.PluginContext) (*pipelines.PluginContext, error) {
	result := pipelines.NewPluginContext()
	result.Set(stepConfig.Output, map[string]any{
		"mock": true,
	})
	return result, nil
}

func (p *MockCSVPlugin) GetPluginType() string { return "Input" }
func (p *MockCSVPlugin) GetPluginName() string { return "mockcsv" }
func (p *MockCSVPlugin) ValidateConfig(config map[string]any) error {
	if _, ok := config["file_path"]; !ok {
		return assert.AnError
	}
	return nil
}
func (p *MockCSVPlugin) GetInputSchema() map[string]any {
	return map[string]any{"type": "object"}
}
