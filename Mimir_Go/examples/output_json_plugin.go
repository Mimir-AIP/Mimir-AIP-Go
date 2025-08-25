// JSON Output Plugin Example
// This example shows how to create an Output plugin for Mimir AIP

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
)

// JSONOutputPlugin implements JSON file output functionality
type JSONOutputPlugin struct {
	name    string
	version string
}

// NewJSONOutputPlugin creates a new JSON output plugin instance
func NewJSONOutputPlugin() *JSONOutputPlugin {
	return &JSONOutputPlugin{
		name:    "JSONOutputPlugin",
		version: "1.0.0",
	}
}

// ExecuteStep writes data to JSON files
func (p *JSONOutputPlugin) ExecuteStep(ctx context.Context, stepConfig pipelines.StepConfig, globalContext *pipelines.PluginContext) (*pipelines.PluginContext, error) {
	fmt.Printf("Executing %s step: %s\n", p.name, stepConfig.Name)

	config := stepConfig.Config

	// Validate configuration
	if err := p.ValidateConfig(config); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	// Get output file path
	filePath, _ := config["file"].(string)
	if filePath == "" {
		return nil, fmt.Errorf("file path is required")
	}

	// Get data to write
	data := p.getOutputData(config, globalContext)
	if data == nil {
		return nil, fmt.Errorf("no data to write")
	}

	// Get output options
	options := p.getOutputOptions(config)

	// Write JSON file
	result, err := p.writeJSONFile(filePath, data, options)
	if err != nil {
		return nil, fmt.Errorf("failed to write JSON file: %w", err)
	}

	context := pipelines.NewPluginContext()
	context.Set(stepConfig.Output, result)
	return context, nil
}

// GetPluginType returns the plugin type
func (p *JSONOutputPlugin) GetPluginType() string {
	return "Output"
}

// GetPluginName returns the plugin name
func (p *JSONOutputPlugin) GetPluginName() string {
	return "json"
}

// ValidateConfig validates the plugin configuration
func (p *JSONOutputPlugin) ValidateConfig(config map[string]interface{}) error {
	if config["file"] == nil {
		return fmt.Errorf("file is required")
	}

	if file, ok := config["file"].(string); !ok || file == "" {
		return fmt.Errorf("file must be a non-empty string")
	}

	return nil
}

// getOutputData extracts data to be written
func (p *JSONOutputPlugin) getOutputData(config map[string]interface{}, context *pipelines.PluginContext) interface{} {
	// Check if input is specified
	if inputKey, ok := config["input"].(string); ok {
		if data, exists := context.Get(inputKey); exists {
			return data
		}
	}

	// Check for direct data in config
	if data, exists := config["data"]; exists {
		return data
	}

	// Return entire context if no specific input
	return context
}

// getOutputOptions extracts output formatting options
func (p *JSONOutputPlugin) getOutputOptions(config map[string]interface{}) map[string]interface{} {
	options := make(map[string]interface{})

	// Pretty printing
	if pretty, ok := config["pretty"].(bool); ok {
		options["pretty"] = pretty
	} else {
		options["pretty"] = true // default
	}

	// Indent string
	if indent, ok := config["indent"].(string); ok {
		options["indent"] = indent
	} else {
		options["indent"] = "  " // default
	}

	// Append mode
	if append, ok := config["append"].(bool); ok {
		options["append"] = append
	} else {
		options["append"] = false // default
	}

	// Create directories
	if mkdir, ok := config["mkdir"].(bool); ok {
		options["mkdir"] = mkdir
	} else {
		options["mkdir"] = true // default
	}

	return options
}

// writeJSONFile writes data to a JSON file
func (p *JSONOutputPlugin) writeJSONFile(filePath string, data interface{}, options map[string]interface{}) (map[string]interface{}, error) {
	// Create directory if needed
	if options["mkdir"].(bool) {
		dir := filepath.Dir(filePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory: %w", err)
		}
	}

	// Serialize JSON
	var jsonData []byte
	var err error

	if options["pretty"].(bool) {
		indent := options["indent"].(string)
		jsonData, err = json.MarshalIndent(data, "", indent)
	} else {
		jsonData, err = json.Marshal(data)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON: %w", err)
	}

	// Add newline for better readability
	jsonData = append(jsonData, '\n')

	// Open file
	flag := os.O_CREATE | os.O_WRONLY
	if options["append"].(bool) {
		flag |= os.O_APPEND
	} else {
		flag |= os.O_TRUNC
	}

	file, err := os.OpenFile(filePath, flag, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Write data
	bytesWritten, err := file.Write(jsonData)
	if err != nil {
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	// Get file info
	fileInfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	// Build result
	result := map[string]interface{}{
		"file_path":      filePath,
		"bytes_written":  bytesWritten,
		"file_size":      fileInfo.Size(),
		"written_at":     time.Now().Format(time.RFC3339),
		"data_type":      fmt.Sprintf("%T", data),
		"pretty_printed": options["pretty"].(bool),
		"appended":       options["append"].(bool),
	}

	return result, nil
}

// Example usage in pipeline YAML:
//
// pipelines:
//   - name: "Data Export Pipeline"
//     steps:
//       - name: "Fetch API Data"
//         plugin: "Input.api"
//         config:
//           url: "https://api.example.com/data"
//         output: "api_data"
//       - name: "Process Data"
//         plugin: "Data_Processing.transform"
//         config:
//           operation: "filter"
//           pattern: "important"
//           input: "api_data"
//         output: "filtered_data"
//       - name: "Export to JSON"
//         plugin: "Output.json"
//         config:
//           file: "./output/data.json"
//           input: "filtered_data"
//           pretty: true
//           indent: "  "
//         output: "export_result"
//       - name: "Export Metadata"
//         plugin: "Output.json"
//         config:
//           file: "./output/metadata.json"
//           data:
//             pipeline: "Data Export Pipeline"
//             exported_at: "2024-01-01T12:00:00Z"
//             record_count: 150
//           pretty: true
//         output: "metadata_result"
