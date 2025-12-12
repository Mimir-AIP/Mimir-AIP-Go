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

	tests := []struct {
		name          string
		yamlContent   string
		expectError   bool
		expectedName  string
		expectedSteps int
	}{
		{
			name: "valid single pipeline",
			yamlContent: `
name: test-pipeline
enabled: true
description: A test pipeline
steps:
  - name: step1
    plugin: Input.test
    config:
      param1: value1
    output: output1
  - name: step2
    plugin: Output.test
    config: {}
    output: output2
`,
			expectError:   false,
			expectedName:  "test-pipeline",
			expectedSteps: 2,
		},
		{
			name: "valid config file with multiple pipelines",
			yamlContent: `
pipelines:
  - name: pipeline1
    enabled: true
    steps:
      - name: step1
        plugin: Input.test
        config: {}
  - name: pipeline2
    enabled: false
    steps:
      - name: step1
        plugin: Output.test
        config: {}
`,
			expectError:   false,
			expectedName:  "pipeline1", // Should return first enabled pipeline
			expectedSteps: 1,
		},
		{
			name: "config file with no enabled pipelines",
			yamlContent: `
pipelines:
  - name: pipeline1
    enabled: false
    steps:
      - name: step1
        plugin: Input.test
        config: {}
  - name: pipeline2
    enabled: false
    steps:
      - name: step1
        plugin: Output.test
        config: {}
`,
			expectError:   false,
			expectedName:  "pipeline1", // Should return first pipeline when none are enabled
			expectedSteps: 1,
		},
		{
			name: "invalid YAML",
			yamlContent: `
name: test-pipeline
enabled: true
invalid_yaml: [
steps:
  - name: step1
    plugin: Input.test
`,
			expectError: true,
		},
		{
			name:        "empty file",
			yamlContent: "",
			expectError: true,
		},
		{
			name: "no pipelines in config file",
			yamlContent: `
pipelines: []
`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Write test file
			filePath := filepath.Join(tempDir, "test.yaml")
			err := os.WriteFile(filePath, []byte(tt.yamlContent), 0644)
			require.NoError(t, err)

			// Parse pipeline
			config, err := ParsePipeline(filePath)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, config)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, config)
				assert.Equal(t, tt.expectedName, config.Name)
				assert.Equal(t, tt.expectedSteps, len(config.Steps))
			}
		})
	}
}

func TestValidatePipelineConfig(t *testing.T) {
	// Create temporary directory and schema file
	tempDir, err := os.MkdirTemp("", "pipeline_validation_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a mock schema file
	schemaContent := `
required:
  - name
  - enabled
  - steps
properties:
  name:
    type: string
  enabled:
    type: boolean
  steps:
    type: array
`
	schemaPath := filepath.Join(tempDir, "schema", "pipeline_schema.yaml")
	err = os.MkdirAll(filepath.Dir(schemaPath), 0755)
	require.NoError(t, err)
	err = os.WriteFile(schemaPath, []byte(schemaContent), 0644)
	require.NoError(t, err)

	// Change working directory to temp dir for schema resolution
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	tests := []struct {
		name        string
		yamlContent string
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid pipeline config",
			yamlContent: `
name: test-pipeline
enabled: true
steps:
  - name: step1
    plugin: Input.test
    config: {}
`,
			expectError: false,
		},
		{
			name: "missing required field - name",
			yamlContent: `
enabled: true
steps:
  - name: step1
    plugin: Input.test
    config: {}
`,
			expectError: true,
			errorMsg:    "missing required key: name",
		},
		{
			name: "missing required field - enabled",
			yamlContent: `
name: test-pipeline
steps:
  - name: step1
    plugin: Input.test
    config: {}
`,
			expectError: true,
			errorMsg:    "missing required key: enabled",
		},
		{
			name: "missing required field - steps",
			yamlContent: `
name: test-pipeline
enabled: true
`,
			expectError: true,
			errorMsg:    "missing required key: steps",
		},
		{
			name:        "empty config",
			yamlContent: "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Write test file
			filePath := filepath.Join(tempDir, "test.yaml")
			err := os.WriteFile(filePath, []byte(tt.yamlContent), 0644)
			require.NoError(t, err)

			// Validate pipeline config
			valid, err := ValidatePipelineConfig(filePath)

			if tt.expectError {
				assert.False(t, valid)
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.True(t, valid)
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetPipelineName(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "pipeline_name_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	yamlContent := `
name: test-pipeline-name
enabled: true
steps:
  - name: step1
    plugin: Input.test
    config: {}
`
	filePath := filepath.Join(tempDir, "test.yaml")
	err = os.WriteFile(filePath, []byte(yamlContent), 0644)
	require.NoError(t, err)

	name, err := GetPipelineName(filePath)
	assert.NoError(t, err)
	assert.Equal(t, "test-pipeline-name", name)
}

func TestGetEnabledPipelines(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "enabled_pipelines_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	yamlContent := `
pipelines:
  - name: pipeline1
    enabled: true
    steps:
      - name: step1
        plugin: Input.test
        config: {}
  - name: pipeline2
    enabled: false
    steps:
      - name: step1
        plugin: Output.test
        config: {}
  - name: pipeline3
    enabled: true
    steps:
      - name: step1
        plugin: Processing.test
        config: {}
`
	filePath := filepath.Join(tempDir, "config.yaml")
	err = os.WriteFile(filePath, []byte(yamlContent), 0644)
	require.NoError(t, err)

	enabledPipelines, err := GetEnabledPipelines(filePath)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(enabledPipelines))
	assert.Contains(t, enabledPipelines, "pipeline1")
	assert.Contains(t, enabledPipelines, "pipeline3")
	assert.NotContains(t, enabledPipelines, "pipeline2")
}

func TestParseAllPipelines(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "all_pipelines_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	yamlContent := `
pipelines:
  - name: pipeline1
    enabled: true
    description: First pipeline
    steps:
      - name: step1
        plugin: Input.test
        config: {}
  - name: pipeline2
    enabled: false
    description: Second pipeline
    steps:
      - name: step1
        plugin: Output.test
        config: {}
`
	filePath := filepath.Join(tempDir, "config.yaml")
	err = os.WriteFile(filePath, []byte(yamlContent), 0644)
	require.NoError(t, err)

	pipelines, err := ParseAllPipelines(filePath)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(pipelines))
	assert.Equal(t, "pipeline1", pipelines[0].Name)
	assert.Equal(t, "pipeline2", pipelines[1].Name)
	assert.True(t, pipelines[0].Enabled)
	assert.False(t, pipelines[1].Enabled)
	assert.Equal(t, "First pipeline", pipelines[0].Description)
	assert.Equal(t, "Second pipeline", pipelines[1].Description)
}

func TestGetSchema(t *testing.T) {
	// Test with non-existent schema file
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	tempDir, err := os.MkdirTemp("", "schema_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)
	os.Chdir(tempDir)

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
	defer os.Chdir(originalDir)

	tempDir, err := os.MkdirTemp("", "validation_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)
	os.Chdir(tempDir)

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
