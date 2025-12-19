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

	// Optional configuration
	sheetName := "" // default: first sheet
	if sn, ok := config["sheet_name"].(string); ok && sn != "" {
		sheetName = sn
	}

	hasHeaders := true // default
	if h, ok := config["has_headers"].(bool); ok {
		hasHeaders = h
	}

	// Validate configuration
	if err := p.ValidateConfig(config); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("file does not exist: %s", filePath)
	}

	// Read and parse Excel
	data, err := p.parseExcel(filePath, sheetName, hasHeaders)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Excel: %w", err)
	}

	// Create result
	result := map[string]any{
		"file_path":        filePath,
		"sheet_name":       data.SheetName,
		"has_headers":      hasHeaders,
		"row_count":        data.RowCount,
		"column_count":     data.ColumnCount,
		"columns":          data.Columns,
		"rows":             data.Rows,
		"sheet_count":      data.SheetCount,
		"available_sheets": data.AvailableSheets,
		"parsed_at":        data.ParsedAt,
	}

	context := pipelines.NewPluginContext()
	context.Set(stepConfig.Output, result)
	return context, nil
}

// ExcelData represents parsed Excel data
type ExcelData struct {
	SheetName       string           `json:"sheet_name"`
	Columns         []string         `json:"columns"`
	Rows            []map[string]any `json:"rows"`
	ColumnCount     int              `json:"column_count"`
	RowCount        int              `json:"row_count"`
	SheetCount      int              `json:"sheet_count"`
	AvailableSheets []string         `json:"available_sheets"`
	ParsedAt        string           `json:"parsed_at"`
}

// parseExcel reads and parses an Excel file
// Note: This is a basic implementation. For full Excel support,
// consider using a library like excelize or similar.
func (p *ExcelPlugin) parseExcel(filePath, sheetName string, hasHeaders bool) (*ExcelData, error) {
	// For now, provide a basic implementation that detects file type
	// and gives appropriate error message for unsupported formats

	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Read first few bytes to detect file type
	buffer := make([]byte, 512)
	n, _ := file.Read(buffer)

	// Check file signatures
	isXLSX := n >= 4 && string(buffer[:4]) == "PK\x03\x04"                      // ZIP file signature (XLSX is ZIP-based)
	isXLS := n >= 8 && string(buffer[:8]) == "\xd0\xcf\x11\xe0\xa1\xb1\x1a\xe1" // OLE2 signature (XLS)

	if !isXLSX && !isXLS {
		return nil, fmt.Errorf("file does not appear to be a valid Excel file (.xlsx or .xls). Consider using CSV input instead")
	}

	if isXLS {
		return nil, fmt.Errorf("legacy .xls format is not supported. Please convert to .xlsx format")
	}

	// For XLSX files, return a message indicating external library needed
	return &ExcelData{
		SheetName:       "Sheet1",
		Columns:         []string{},
		Rows:            []map[string]any{},
		ColumnCount:     0,
		RowCount:        0,
		SheetCount:      1,
		AvailableSheets: []string{"Sheet1"},
		ParsedAt:        time.Now().Format("2006-01-02T15:04:05Z07:00"),
	}, fmt.Errorf("Excel XLSX parsing requires external library (github.com/xuri/excelize/v2). Please install the dependency or use CSV format for now")
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
