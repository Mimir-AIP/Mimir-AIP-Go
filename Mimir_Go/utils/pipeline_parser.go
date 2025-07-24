// File responsible for parsing the pipeline configuration file yaml,
// a pipeline file path will be passed to this by the main function and this will return true/false if the file is valid or not,
// if it is not valid, it will also return an error with the reason why it is not valid
package utils

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

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

	// TODO Add validation logic against the schema
	if len(pipelineConfig) == 0 {
		return false, fmt.Errorf("pipeline config is empty")
	}

	return true, nil
}
