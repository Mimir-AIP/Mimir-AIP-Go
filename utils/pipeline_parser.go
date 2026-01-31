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
func getSchema() (map[string]any, error) {
	schemaPath := filepath.Join("schema", "pipeline_schema.yaml")
	schemaFile, err := os.ReadFile(schemaPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read schema file: %w", err)
	}

	var schema map[string]any
	if err := yaml.Unmarshal(schemaFile, &schema); err != nil {
		return nil, fmt.Errorf("failed to unmarshal schema: %w", err)
	}

	return schema, nil
}

// ValidatePipelineConfig validates the pipeline configuration file against the schema
// Supports both ConfigFile format (with pipelines array) and single PipelineConfig format
func ValidatePipelineConfig(pipelineFilePath string) (bool, error) {
	schema, schemaErr := getSchema()

	pipelineFile, err := os.ReadFile(pipelineFilePath)
	if err != nil {
		return false, fmt.Errorf("failed to read pipeline file: %w", err)
	}

	var pipelineConfig map[string]any
	if err := yaml.Unmarshal(pipelineFile, &pipelineConfig); err != nil {
		return false, fmt.Errorf("failed to unmarshal pipeline config: %w", err)
	}
	if len(pipelineConfig) == 0 {
		return false, fmt.Errorf("pipeline config is empty")
	}

	// If schema is not available, do basic validation
	if schemaErr != nil {
		// Detect format and do basic validation
		if _, hasPipelines := pipelineConfig["pipelines"]; hasPipelines {
			return validateConfigFileBasic(pipelineConfig)
		}
		return validateSinglePipelineBasic(pipelineConfig)
	}

	// Detect format: ConfigFile has 'pipelines' key, single pipeline has 'name' directly
	if _, hasPipelines := pipelineConfig["pipelines"]; hasPipelines {
		// ConfigFile format - validate against top-level schema
		return validateAgainstSchema(pipelineConfig, schema)
	}

	// Single pipeline format - validate against nested pipeline schema
	pipelineSchema := extractPipelineSchema(schema)
	if pipelineSchema == nil {
		// No nested schema available, do basic validation
		return validateSinglePipelineBasic(pipelineConfig)
	}

	return validateAgainstSchema(pipelineConfig, pipelineSchema)
}

// extractPipelineSchema extracts the pipeline item schema from the full schema
func extractPipelineSchema(schema map[string]any) map[string]any {
	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		return nil
	}

	pipelinesProp, ok := properties["pipelines"].(map[string]any)
	if !ok {
		return nil
	}

	items, ok := pipelinesProp["items"].(map[string]any)
	if !ok {
		return nil
	}

	return items
}

// validateAgainstSchema validates a config against a schema
func validateAgainstSchema(config map[string]any, schema map[string]any) (bool, error) {
	// Check required keys
	requiredKeys, ok := schema["required"].([]any)
	if ok {
		for _, key := range requiredKeys {
			keyStr, ok := key.(string)
			if !ok {
				continue
			}
			if _, exists := config[keyStr]; !exists {
				return false, fmt.Errorf("missing required key: %s", keyStr)
			}
		}
	}

	return true, nil
}

// validateSinglePipelineBasic performs basic validation for single pipeline format
func validateSinglePipelineBasic(config map[string]any) (bool, error) {
	// Single pipeline requires 'name' and 'steps'
	if _, exists := config["name"]; !exists {
		return false, fmt.Errorf("missing required key: name")
	}
	if _, exists := config["steps"]; !exists {
		return false, fmt.Errorf("missing required key: steps")
	}
	return true, nil
}

// validateConfigFileBasic performs basic validation for ConfigFile format
func validateConfigFileBasic(config map[string]any) (bool, error) {
	// ConfigFile requires 'pipelines' array
	pipelines, exists := config["pipelines"]
	if !exists {
		return false, fmt.Errorf("missing required key: pipelines")
	}

	// Check that pipelines is actually an array
	if _, ok := pipelines.([]any); !ok {
		return false, fmt.Errorf("pipelines must be an array")
	}

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
