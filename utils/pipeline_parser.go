// File responsible for parsing the pipeline configuration file yaml,
// a pipeline file path will be passed to this by the main function and this will return true/false if the file is valid or not,
// if it is not valid, it will also return an error with the reason why it is not valid
package utils

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
	"gopkg.in/yaml.v3"
)

// PipelineConfig represents a parsed pipeline configuration
type PipelineConfig struct {
	Name        string                 `yaml:"name"`
	Enabled     bool                   `yaml:"enabled"`
	Steps       []pipelines.StepConfig `yaml:"steps"`
	Description string                 `yaml:"description,omitempty"`
}

// ConfigFile represents the top-level configuration file structure
type ConfigFile struct {
	Pipelines []PipelineConfig `yaml:"pipelines"`
}

// Validate config against the schema(Legacy configs may not have schema defined)
func getSchema() (map[string]interface{}, error) {
	schemaPath := filepath.Join("schema", "pipeline_schema.yaml")
	schemaFile, err := os.ReadFile(schemaPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read schema file: %w", err)
	}

	var schema map[string]interface{}
	if err := yaml.Unmarshal(schemaFile, &schema); err != nil {
		return nil, fmt.Errorf("failed to unmarshal schema: %w", err)
	}

	return schema, nil
}

// ValidatePipelineConfig validates the pipeline configuration file against the schema
func ValidatePipelineConfig(pipelineFilePath string) (bool, error) {
	schema, err := getSchema()
	if err != nil {
		return false, fmt.Errorf("error getting schema: %w", err)
	}

	pipelineFile, err := os.ReadFile(pipelineFilePath)
	if err != nil {
		return false, fmt.Errorf("failed to read pipeline file: %w", err)
	}

	var pipelineConfig map[string]interface{}
	if err := yaml.Unmarshal(pipelineFile, &pipelineConfig); err != nil {
		return false, fmt.Errorf("failed to unmarshal pipeline config: %w", err)
	}
	if len(pipelineConfig) == 0 {
		return false, fmt.Errorf("pipeline config is empty")
	}

	// Basic validation: check that all required top-level keys in the schema exist in the pipeline config
	requiredKeys, ok := schema["required"].([]interface{})
	if ok {
		for _, key := range requiredKeys {
			keyStr, ok := key.(string)
			if !ok {
				continue
			}
			if _, exists := pipelineConfig[keyStr]; !exists {
				return false, fmt.Errorf("missing required key: %s", keyStr)
			}
		}
	}
	//TODO add advanced validation

	return true, nil
}

// ParsePipeline parses a pipeline configuration file
func ParsePipeline(pipelinePath string) (*PipelineConfig, error) {
	// Read the pipeline file
	data, err := os.ReadFile(pipelinePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read pipeline file: %w", err)
	}

	// Try to parse as a single pipeline first
	var singlePipeline PipelineConfig
	if err := yaml.Unmarshal(data, &singlePipeline); err == nil && singlePipeline.Name != "" {
		return &singlePipeline, nil
	}

	// If that fails, try to parse as a config file with multiple pipelines
	var configFile ConfigFile
	if err := yaml.Unmarshal(data, &configFile); err != nil {
		return nil, fmt.Errorf("failed to parse pipeline YAML: %w", err)
	}

	// If it's a config file, look for the first enabled pipeline
	for _, pipeline := range configFile.Pipelines {
		if pipeline.Enabled {
			return &pipeline, nil
		}
	}

	// If no enabled pipeline found, return the first one
	if len(configFile.Pipelines) > 0 {
		return &configFile.Pipelines[0], nil
	}

	return nil, fmt.Errorf("no pipelines found in config file")
}

// GetPipelineName extracts the name of a pipeline from its file
func GetPipelineName(pipelinePath string) (string, error) {
	config, err := ParsePipeline(pipelinePath)
	if err != nil {
		return "", err
	}
	return config.Name, nil
}

// GetEnabledPipelines reads the config file and returns enabled pipelines
func GetEnabledPipelines(configPath string) ([]string, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var configFile ConfigFile
	if err := yaml.Unmarshal(data, &configFile); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	var enabledPipelines []string
	for _, pipeline := range configFile.Pipelines {
		if pipeline.Enabled {
			enabledPipelines = append(enabledPipelines, pipeline.Name)
		}
	}

	return enabledPipelines, nil
}

// ParseAllPipelines parses all pipelines from a config file
func ParseAllPipelines(configPath string) ([]PipelineConfig, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var configFile ConfigFile
	if err := yaml.Unmarshal(data, &configFile); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return configFile.Pipelines, nil
}
