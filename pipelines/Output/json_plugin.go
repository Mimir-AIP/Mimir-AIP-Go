// JSON Output Plugin
// Writes data to JSON format (string or file)

package Output

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
)

// JSONPlugin implements JSON output functionality
type JSONPlugin struct {
	name    string
	version string
}

// NewJSONPlugin creates a new JSON output plugin instance
func NewJSONPlugin() *JSONPlugin {
	return &JSONPlugin{
		name:    "JSONOutputPlugin",
		version: "1.0.0",
	}
}

// ExecuteStep writes data to JSON format
func (p *JSONPlugin) ExecuteStep(ctx context.Context, stepConfig pipelines.StepConfig, globalContext *pipelines.PluginContext) (*pipelines.PluginContext, error) {
	config := stepConfig.Config

	// Validate configuration
	if err := p.ValidateConfig(config); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	// Get input data
	inputKey, ok := config["input"].(string)
	if !ok || inputKey == "" {
		return nil, fmt.Errorf("input key is required")
	}

	data, exists := globalContext.Get(inputKey)
	if !exists {
		return nil, fmt.Errorf("input data '%s' not found in context", inputKey)
	}

	// Get pretty print option (default: true)
	prettyPrint := true
	if pretty, ok := config["pretty_print"].(bool); ok {
		prettyPrint = pretty
	}

	// Serialize to JSON
	var jsonData []byte
	var err error
	if prettyPrint {
		jsonData, err = json.MarshalIndent(data, "", "  ")
	} else {
		jsonData, err = json.Marshal(data)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON: %w", err)
	}

	result := map[string]any{
		"success": true,
	}

	// Check if we should write to file or return as string
	if filePath, ok := config["file_path"].(string); ok && filePath != "" {
		// Write to file
		dir := filepath.Dir(filePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory: %w", err)
		}

		if err := os.WriteFile(filePath, jsonData, 0644); err != nil {
			return nil, fmt.Errorf("failed to write JSON file: %w", err)
		}

		result["file_path"] = filePath
		result["bytes_written"] = len(jsonData)
	} else {
		// Return as string
		result["json_string"] = string(jsonData)
	}

	// Create output context
	outputContext := pipelines.NewPluginContext()
	outputKey := stepConfig.Output
	if outputKey == "" {
		outputKey = "json_output"
	}
	outputContext.Set(outputKey, result)

	return outputContext, nil
}

// GetPluginType returns the plugin type
func (p *JSONPlugin) GetPluginType() string {
	return "Output"
}

// GetPluginName returns the plugin name
func (p *JSONPlugin) GetPluginName() string {
	return "json"
}

// ValidateConfig validates the plugin configuration
func (p *JSONPlugin) ValidateConfig(config map[string]any) error {
	if config["input"] == nil {
		return fmt.Errorf("input is required")
	}

	if input, ok := config["input"].(string); !ok || input == "" {
		return fmt.Errorf("input must be a non-empty string")
	}

	return nil
}

// GetInputSchema returns the JSON Schema for plugin configuration
// NOTE: This only contains plugin-level settings. Step-level parameters like
// 'input', 'file_path', etc. should be defined in pipeline steps.
func (p *JSONPlugin) GetInputSchema() map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
}
