package Input

import (
	"context"
	"fmt"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
)

// XMLPlugin implements an XML file input plugin for Mimir AIP
// This is a DEMO plugin to show automatic discovery
type XMLPlugin struct {
	name    string
	version string
}

// NewXMLPlugin creates a new XML plugin instance
func NewXMLPlugin() *XMLPlugin {
	return &XMLPlugin{
		name:    "XMLPlugin",
		version: "1.0.0",
	}
}

// ExecuteStep reads and parses an XML file
func (p *XMLPlugin) ExecuteStep(ctx context.Context, stepConfig pipelines.StepConfig, globalContext *pipelines.PluginContext) (*pipelines.PluginContext, error) {
	fmt.Printf("Executing %s step: %s\n", p.name, stepConfig.Name)

	// This is a demo plugin - just return success
	result := pipelines.NewPluginContext()
	result.Set(stepConfig.Output, map[string]any{
		"status":  "success",
		"message": "XML plugin executed (demo only)",
	})
	return result, nil
}

// GetPluginType returns the plugin type
func (p *XMLPlugin) GetPluginType() string {
	return "Input"
}

// GetPluginName returns the plugin name
func (p *XMLPlugin) GetPluginName() string {
	return "xml"
}

// ValidateConfig validates the plugin configuration
func (p *XMLPlugin) ValidateConfig(config map[string]any) error {
	if _, ok := config["file_path"].(string); !ok {
		return fmt.Errorf("file_path is required and must be a string")
	}
	return nil
}

// GetInputSchema returns the JSON Schema for XML plugin configuration
func (p *XMLPlugin) GetInputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"file_path": map[string]any{
				"type":        "string",
				"description": "Path to the XML file to read. Can be absolute or relative to the working directory.",
			},
			"validate_schema": map[string]any{
				"type":        "boolean",
				"description": "Validate XML against XSD schema if provided",
				"default":     false,
			},
			"schema_path": map[string]any{
				"type":        "string",
				"description": "Path to XSD schema file for validation (optional)",
			},
			"namespaces": map[string]any{
				"type":        "object",
				"description": "XML namespace prefixes and URIs",
				"additionalProperties": map[string]any{
					"type": "string",
				},
			},
		},
		"required": []string{"file_path"},
	}
}
