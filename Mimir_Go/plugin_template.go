// Plugin Template for Mimir AIP Go
// Copy this template to create new plugins for the Mimir AIP platform
//
// Instructions:
// 1. Copy this file to your plugin directory
// 2. Replace "Template" with your plugin name
// 3. Implement the required methods
// 4. Update the plugin registration
// 5. Add your plugin-specific configuration

package main

import (
	"context"
	"fmt"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
)

// TemplatePlugin implements a template plugin for Mimir AIP
type TemplatePlugin struct {
	// Add plugin-specific fields here
	name    string
	version string
}

// NewTemplatePlugin creates a new instance of the template plugin
func NewTemplatePlugin() *TemplatePlugin {
	return &TemplatePlugin{
		name:    "TemplatePlugin",
		version: "1.0.0",
	}
}

// ExecuteStep executes a single pipeline step
func (p *TemplatePlugin) ExecuteStep(ctx context.Context, stepConfig pipelines.StepConfig, globalContext pipelines.PluginContext) (pipelines.PluginContext, error) {
	// Log the step execution
	fmt.Printf("Executing %s step: %s\n", p.name, stepConfig.Name)

	// Extract configuration
	config := stepConfig.Config

	// Validate required configuration
	if err := p.ValidateConfig(config); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	// Process the step
	result, err := p.processStep(config, globalContext)
	if err != nil {
		return nil, fmt.Errorf("step processing failed: %w", err)
	}

	// Return updated context
	return pipelines.PluginContext{
		stepConfig.Output: result,
	}, nil
}

// GetPluginType returns the plugin type
func (p *TemplatePlugin) GetPluginType() string {
	// Change this to match your plugin type:
	// "Input", "Data_Processing", "AIModels", "Output"
	return "Data_Processing"
}

// GetPluginName returns the plugin name
func (p *TemplatePlugin) GetPluginName() string {
	// This should match the plugin reference in YAML (e.g., "Data_Processing.template")
	return "template"
}

// ValidateConfig validates the plugin configuration
func (p *TemplatePlugin) ValidateConfig(config map[string]interface{}) error {
	// Add your configuration validation logic here

	// Example validation:
	// if config["required_field"] == nil {
	//     return fmt.Errorf("required_field is required")
	// }

	return nil
}

// processStep contains the main plugin logic
func (p *TemplatePlugin) processStep(config map[string]interface{}, context pipelines.PluginContext) (interface{}, error) {
	// Implement your plugin logic here

	// Example: Simple data transformation
	input := "default_input"
	if val, exists := config["input"]; exists {
		if str, ok := val.(string); ok {
			input = str
		}
	}

	// Process the input
	result := map[string]interface{}{
		"original":       input,
		"processed":      fmt.Sprintf("PROCESSED_%s", input),
		"timestamp":      fmt.Sprintf("%d", context["timestamp"]),
		"plugin_version": p.version,
	}

	return result, nil
}

// Example usage in pipeline YAML:
//
// pipelines:
//   - name: "Example Pipeline"
//     steps:
//       - name: "Template Step"
//         plugin: "Data_Processing.template"
//         config:
//           input: "test_data"
//           option1: "value1"
//         output: "template_result"

// This template provides:
// - Basic plugin structure
// - Configuration validation
// - Error handling
// - Context management
// - Logging support
//
// For more advanced plugins, you can:
// - Add initialization methods
// - Implement cleanup logic
// - Add health checks
// - Support streaming data
// - Add metrics collection
