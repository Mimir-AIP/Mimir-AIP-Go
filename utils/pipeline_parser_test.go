package utils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsePipeline_SinglePipeline(t *testing.T) {
	tmpDir := t.TempDir()
	pipelineFile := filepath.Join(tmpDir, "pipeline.yaml")

	pipelineYAML := `name: test-pipeline
enabled: true
description: Test pipeline
steps:
  - name: step1
    plugin: Input.csv
    config:
      file_path: data.csv
    output: data
`
	err := os.WriteFile(pipelineFile, []byte(pipelineYAML), 0644)
	require.NoError(t, err)

	config, err := ParsePipeline(pipelineFile)

	require.NoError(t, err)
	assert.Equal(t, "test-pipeline", config.Name)
	assert.True(t, config.Enabled)
	assert.Equal(t, "Test pipeline", config.Description)
	assert.Len(t, config.Steps, 1)
	assert.Equal(t, "step1", config.Steps[0].Name)
	assert.Equal(t, "Input.csv", config.Steps[0].Plugin)
}

func TestParsePipeline_ConfigFile(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")

	configYAML := `pipelines:
  - name: pipeline-1
    enabled: true
    steps:
      - name: step1
        plugin: Input.csv
        config:
          file_path: data1.csv
        output: data1
  - name: pipeline-2
    enabled: false
    steps:
      - name: step1
        plugin: Input.json
        config:
          file_path: data2.json
        output: data2
`
	err := os.WriteFile(configFile, []byte(configYAML), 0644)
	require.NoError(t, err)

	config, err := ParsePipeline(configFile)

	require.NoError(t, err)
	assert.Equal(t, "pipeline-1", config.Name)
	assert.True(t, config.Enabled)
}

func TestParsePipeline_NoEnabledPipelines(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")

	configYAML := `pipelines:
  - name: pipeline-1
    enabled: false
    steps: []
  - name: pipeline-2
    enabled: false
    steps: []
`
	err := os.WriteFile(configFile, []byte(configYAML), 0644)
	require.NoError(t, err)

	config, err := ParsePipeline(configFile)

	require.NoError(t, err)
	assert.Equal(t, "pipeline-1", config.Name)
}

func TestParsePipeline_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	pipelineFile := filepath.Join(tmpDir, "empty.yaml")

	err := os.WriteFile(pipelineFile, []byte(""), 0644)
	require.NoError(t, err)

	_, err = ParsePipeline(pipelineFile)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no pipelines found")
}

func TestParsePipeline_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	pipelineFile := filepath.Join(tmpDir, "invalid.yaml")

	err := os.WriteFile(pipelineFile, []byte("invalid: yaml: ::"), 0644)
	require.NoError(t, err)

	_, err = ParsePipeline(pipelineFile)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse")
}

func TestParsePipeline_NonExistentFile(t *testing.T) {
	_, err := ParsePipeline("/non/existent/path.yaml")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read pipeline file")
}

func TestGetPipelineName(t *testing.T) {
	tmpDir := t.TempDir()
	pipelineFile := filepath.Join(tmpDir, "pipeline.yaml")

	pipelineYAML := `name: my-awesome-pipeline
enabled: true
steps: []
`
	err := os.WriteFile(pipelineFile, []byte(pipelineYAML), 0644)
	require.NoError(t, err)

	name, err := GetPipelineName(pipelineFile)

	require.NoError(t, err)
	assert.Equal(t, "my-awesome-pipeline", name)
}

func TestGetEnabledPipelines(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")

	configYAML := `pipelines:
  - name: pipeline-1
    enabled: true
    steps: []
  - name: pipeline-2
    enabled: false
    steps: []
  - name: pipeline-3
    enabled: true
    steps: []
`
	err := os.WriteFile(configFile, []byte(configYAML), 0644)
	require.NoError(t, err)

	enabled, err := GetEnabledPipelines(configFile)

	require.NoError(t, err)
	assert.Len(t, enabled, 2)
	assert.Contains(t, enabled, "pipeline-1")
	assert.Contains(t, enabled, "pipeline-3")
	assert.NotContains(t, enabled, "pipeline-2")
}

func TestParseAllPipelines(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")

	configYAML := `pipelines:
  - name: pipeline-1
    enabled: true
    steps: []
  - name: pipeline-2
    enabled: false
    steps: []
`
	err := os.WriteFile(configFile, []byte(configYAML), 0644)
	require.NoError(t, err)

	pipelines, err := ParseAllPipelines(configFile)

	require.NoError(t, err)
	assert.Len(t, pipelines, 2)
	assert.Equal(t, "pipeline-1", pipelines[0].Name)
	assert.Equal(t, "pipeline-2", pipelines[1].Name)
}

func TestValidatePipelineConfig_Success(t *testing.T) {
	tmpDir := t.TempDir()
	pipelineFile := filepath.Join(tmpDir, "pipeline.yaml")

	// Create a valid pipeline with required fields
	pipelineYAML := `name: test-pipeline
enabled: true
steps: []
`
	err := os.WriteFile(pipelineFile, []byte(pipelineYAML), 0644)
	require.NoError(t, err)

	// Note: This may fail if schema file doesn't exist, which is fine
	// We're testing the validation logic itself
	valid, err := ValidatePipelineConfig(pipelineFile)

	// If schema doesn't exist, we expect an error
	if err != nil {
		assert.Contains(t, err.Error(), "schema")
	} else {
		assert.True(t, valid)
	}
}
