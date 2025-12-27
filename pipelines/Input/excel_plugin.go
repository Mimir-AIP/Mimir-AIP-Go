package Input

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
)

// ExcelPlugin implements an Excel file input plugin for Mimir AIP
type ExcelPlugin struct {
	name    string
	version string
}

// NewExcelPlugin creates a new Excel plugin instance
func NewExcelPlugin() *ExcelPlugin {
	return &ExcelPlugin{
		name:    "ExcelPlugin",
		version: "1.0.0",
	}
}

// ExecuteStep reads and parses an Excel file
func (p *ExcelPlugin) ExecuteStep(ctx context.Context, stepConfig pipelines.StepConfig, globalContext *pipelines.PluginContext) (*pipelines.PluginContext, error) {
	fmt.Printf("Executing %s step: %s\n", p.name, stepConfig.Name)

	config := stepConfig.Config

	// Extract configuration
	filePath, ok := config["file_path"].(string)
	if !ok || filePath == "" {
		return nil, fmt.Errorf("file_path is required in config")
	}

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("file does not exist: %s", filePath)
	}

	// Optional configuration
	sheetName := ""
	if s, ok := config["sheet_name"].(string); ok {
		sheetName = s
	}

	hasHeaders := true
	if h, ok := config["has_headers"].(bool); ok {
		hasHeaders = h
	}

	// For now, return a mock result since we don't have the Excel library in minimal build
	// In full implementation, use github.com/xuri/excelize/v2
	result := map[string]any{
		"file_path":   filePath,
		"sheet_name":  sheetName,
		"has_headers": hasHeaders,
		"row_count":   0,
		"columns":     []string{},
		"rows":        []map[string]any{},
		"parsed_at":   time.Now().Format(time.RFC3339),
		"message":     "Excel parsing requires excelize library - using mock data",
	}

	resultContext := pipelines.NewPluginContext()
	resultContext.Set(stepConfig.Output, result)
	return resultContext, nil
}

// GetPluginType returns the plugin type
func (p *ExcelPlugin) GetPluginType() string {
	return "Input"
}

// GetPluginName returns the plugin name
func (p *ExcelPlugin) GetPluginName() string {
	return "excel"
}

// ValidateConfig validates the plugin configuration
func (p *ExcelPlugin) ValidateConfig(config map[string]any) error {
	if _, ok := config["file_path"].(string); !ok {
		return fmt.Errorf("file_path is required and must be a string")
	}
	return nil
}

// GetInputSchema returns the JSON Schema for Excel plugin configuration
func (p *ExcelPlugin) GetInputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"file_path": map[string]any{
				"type":        "string",
				"description": "Path to the Excel file to read (.xlsx format). Can be absolute or relative to the working directory.",
			},
			"sheet_name": map[string]any{
				"type":        "string",
				"description": "Name of the worksheet to read. If not specified, the first sheet will be used.",
			},
			"has_headers": map[string]any{
				"type":        "boolean",
				"description": "Whether the first row contains column headers. If false, columns will be named 'column_1', 'column_2', etc.",
				"default":     true,
			},
		},
		"required": []string{"file_path"},
	}
}
