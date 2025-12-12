package utils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsePipeline(t *testing.T) {
	// Create temporary directory for test files
	tempDir, err := os.MkdirTemp("", "pipeline_parser_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)
	_ = os.Chdir(tempDir)

	// Create schema directory and file
	schemaDir := filepath.Join(tempDir, "schema")
	err = os.MkdirAll(schemaDir, 0755)
	require.NoError(t, err)

	schemaContent := `
required:
  - name
  - enabled
properties:
  name:
    type: string
  enabled:
    type: boolean
`
	schemaPath := filepath.Join(schemaDir, "pipeline_schema.yaml")
	err = os.WriteFile(schemaPath, []byte(schemaContent), 0644)
	require.NoError(t, err)

	schema, err := getSchema()
	assert.NoError(t, err)
	assert.NotNil(t, schema)

	// Check that required fields are present
	required, ok := schema["required"].([]any)
	assert.True(t, ok)
	assert.Contains(t, required, "name")
	assert.Contains(t, required, "enabled")
}

func TestParsePipelineFileNotFound(t *testing.T) {
	_, err := ParsePipeline("/nonexistent/file.yaml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read pipeline file")
}

func TestValidatePipelineConfigFileNotFound(t *testing.T) {
	originalDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(originalDir) }()

	tempDir, err := os.MkdirTemp("", "validation_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)
	_ = os.Chdir(tempDir)

	// Create schema directory
	schemaDir := filepath.Join(tempDir, "schema")
	err = os.MkdirAll(schemaDir, 0755)
	require.NoError(t, err)

	schemaContent := `required: []`
	schemaPath := filepath.Join(schemaDir, "pipeline_schema.yaml")
	err = os.WriteFile(schemaPath, []byte(schemaContent), 0644)
	require.NoError(t, err)

	valid, err := ValidatePipelineConfig("/nonexistent/file.yaml")
	assert.False(t, valid)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read pipeline file")
}

func TestPipelineConfigStruct(t *testing.T) {
	config := &PipelineConfig{
		Name:        "test-pipeline",
		Enabled:     true,
		Description: "Test description",
		Steps: []pipelines.StepConfig{
			{
				Name:   "step1",
				Plugin: "Input.test",
				Config: map[string]any{"param": "value"},
				Output: "output1",
			},
		},
	}

	assert.Equal(t, "test-pipeline", config.Name)
	assert.True(t, config.Enabled)
	assert.Equal(t, "Test description", config.Description)
	assert.Equal(t, 1, len(config.Steps))
	assert.Equal(t, "step1", config.Steps[0].Name)
	assert.Equal(t, "Input.test", config.Steps[0].Plugin)
	assert.Equal(t, "value", config.Steps[0].Config["param"])
	assert.Equal(t, "output1", config.Steps[0].Output)
}

func TestConfigFileStruct(t *testing.T) {
	configFile := &ConfigFile{
		Pipelines: []PipelineConfig{
			{
				Name:    "pipeline1",
				Enabled: true,
				Steps:   []pipelines.StepConfig{},
			},
			{
				Name:    "pipeline2",
				Enabled: false,
				Steps:   []pipelines.StepConfig{},
			},
		},
	}

	assert.Equal(t, 2, len(configFile.Pipelines))
	assert.Equal(t, "pipeline1", configFile.Pipelines[0].Name)
	assert.True(t, configFile.Pipelines[0].Enabled)
	assert.Equal(t, "pipeline2", configFile.Pipelines[1].Name)
	assert.False(t, configFile.Pipelines[1].Enabled)
}
