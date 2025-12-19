// JSON Input Plugin
// Reads and parses JSON data from a string or file

package Input

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
)

// JSONPlugin implements JSON input functionality
type JSONPlugin struct {
	name    string
	version string
}

// NewJSONPlugin creates a new JSON input plugin instance
func NewJSONPlugin() *JSONPlugin {
	return &JSONPlugin{
		name:    "JSONPlugin",
		version: "1.0.0",
	}
}

// ExecuteStep reads and parses JSON data
func (p *JSONPlugin) ExecuteStep(ctx context.Context, stepConfig pipelines.StepConfig, globalContext *pipelines.PluginContext) (*pipelines.PluginContext, error) {
	config := stepConfig.Config

	// Validate configuration
	if err := p.ValidateConfig(config); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	var data any
	var err error

	// Check if we have a JSON string directly in config
	if jsonStr, ok := config["json_string"].(string); ok && jsonStr != "" {
		// Parse JSON string directly
		err = json.Unmarshal([]byte(jsonStr), &data)
		if err != nil {
			return nil, fmt.Errorf("failed to parse JSON string: %w", err)
		}
	} else if filePath, ok := config["file_path"].(string); ok && filePath != "" {
		// Read JSON from file
		fileData, readErr := os.ReadFile(filePath)
		if readErr != nil {
			return nil, fmt.Errorf("failed to read JSON file: %w", readErr)
		}

		err = json.Unmarshal(fileData, &data)
		if err != nil {
			return nil, fmt.Errorf("failed to parse JSON file: %w", err)
		}
	} else {
		return nil, fmt.Errorf("either json_string or file_path must be provided")
	}

	// Create output context
	outputContext := pipelines.NewPluginContext()
	outputKey := stepConfig.Output
	if outputKey == "" {
		outputKey = "json_data"
	}
	outputContext.Set(outputKey, data)

	return outputContext, nil
}

// GetPluginType returns the plugin type
func (p *JSONPlugin) GetPluginType() string {
	return "Input"
}

// GetPluginName returns the plugin name
func (p *JSONPlugin) GetPluginName() string {
	return "json"
}

// ValidateConfig validates the plugin configuration
func (p *JSONPlugin) ValidateConfig(config map[string]any) error {
	hasJSONString := config["json_string"] != nil && config["json_string"].(string) != ""
	hasFilePath := config["file_path"] != nil && config["file_path"].(string) != ""

	if !hasJSONString && !hasFilePath {
		return fmt.Errorf("either json_string or file_path is required")
	}

	return nil
}

// GetInputSchema returns the JSON Schema for plugin configuration
func (p *JSONPlugin) GetInputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"json_string": map[string]any{
				"type":        "string",
				"description": "JSON string to parse directly",
			},
			"file_path": map[string]any{
				"type":        "string",
				"description": "Path to JSON file to read and parse",
			},
		},
		"oneOf": []map[string]any{
			{"required": []string{"json_string"}},
			{"required": []string{"file_path"}},
		},
		"description": "Reads and parses JSON data from a string or file",
	}
}
