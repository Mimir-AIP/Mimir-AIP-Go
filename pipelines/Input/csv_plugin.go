package Input

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
)

// CSVPlugin implements a CSV file input plugin for Mimir AIP
type CSVPlugin struct {
	name    string
	version string
}

// NewCSVPlugin creates a new CSV plugin instance
func NewCSVPlugin() *CSVPlugin {
	return &CSVPlugin{
		name:    "CSVPlugin",
		version: "1.0.0",
	}
}

// ExecuteStep reads and parses a CSV file
func (p *CSVPlugin) ExecuteStep(ctx context.Context, stepConfig pipelines.StepConfig, globalContext *pipelines.PluginContext) (*pipelines.PluginContext, error) {
	fmt.Printf("Executing %s step: %s\n", p.name, stepConfig.Name)

	config := stepConfig.Config

	// Extract configuration
	filePath, ok := config["file_path"].(string)
	if !ok || filePath == "" {
		return nil, fmt.Errorf("file_path is required in config")
	}

	// Optional configuration
	hasHeaders := true // default
	if h, ok := config["has_headers"].(bool); ok {
		hasHeaders = h
	}

	delimiter := "," // default
	if d, ok := config["delimiter"].(string); ok && len(d) == 1 {
		delimiter = d
	}

	// Validate configuration
	if err := p.ValidateConfig(config); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("file does not exist: %s", filePath)
	}

	// Read and parse CSV
	data, err := p.parseCSV(filePath, hasHeaders, rune(delimiter[0]))
	if err != nil {
		return nil, fmt.Errorf("failed to parse CSV: %w", err)
	}

	// Create result
	result := map[string]any{
		"file_path":    filePath,
		"has_headers":  hasHeaders,
		"delimiter":    delimiter,
		"row_count":    len(data.Rows),
		"column_count": data.ColumnCount,
		"columns":      data.Columns,
		"rows":         data.Rows,
		"parsed_at":    data.ParsedAt,
	}

	context := pipelines.NewPluginContext()
	context.Set(stepConfig.Output, result)
	return context, nil
}

// CSVData represents parsed CSV data
type CSVData struct {
	Columns     []string         `json:"columns"`
	Rows        []map[string]any `json:"rows"`
	ColumnCount int              `json:"column_count"`
	RowCount    int              `json:"row_count"`
	ParsedAt    string           `json:"parsed_at"`
}

// parseCSV reads and parses a CSV file
func (p *CSVPlugin) parseCSV(filePath string, hasHeaders bool, delimiter rune) (*CSVData, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comma = delimiter
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = true

	// Read all records
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV: %w", err)
	}

	if len(records) == 0 {
		return &CSVData{
			Columns:     []string{},
			Rows:        []map[string]any{},
			ColumnCount: 0,
			RowCount:    0,
			ParsedAt:    time.Now().Format("2006-01-02T15:04:05Z07:00"),
		}, nil
	}

	var columns []string
	var rows []map[string]any

	// Determine columns
	if hasHeaders && len(records) > 0 {
		columns = records[0]
		records = records[1:]
	} else {
		// Generate column names
		columns = make([]string, len(records[0]))
		for i := range columns {
			columns[i] = fmt.Sprintf("column_%d", i+1)
		}
	}

	// Process rows
	for _, record := range records {
		row := make(map[string]any)
		for i, value := range record {
			if i < len(columns) {
				// Try to parse as number
				if numVal, err := strconv.ParseFloat(strings.TrimSpace(value), 64); err == nil {
					row[columns[i]] = numVal
				} else if boolVal, err := strconv.ParseBool(strings.TrimSpace(value)); err == nil {
					row[columns[i]] = boolVal
				} else {
					row[columns[i]] = strings.TrimSpace(value)
				}
			}
		}
		rows = append(rows, row)
	}

	return &CSVData{
		Columns:     columns,
		Rows:        rows,
		ColumnCount: len(columns),
		RowCount:    len(rows),
		ParsedAt:    time.Now().Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}

// GetPluginType returns the plugin type
func (p *CSVPlugin) GetPluginType() string {
	return "Input"
}

// GetPluginName returns the plugin name
func (p *CSVPlugin) GetPluginName() string {
	return "csv"
}

// ValidateConfig validates the plugin configuration
func (p *CSVPlugin) ValidateConfig(config map[string]any) error {
	if _, ok := config["file_path"].(string); !ok {
		return fmt.Errorf("file_path is required and must be a string")
	}

	if delimiter, ok := config["delimiter"].(string); ok {
		if len(delimiter) != 1 {
			return fmt.Errorf("delimiter must be a single character")
		}
	}

	return nil
}

func main() {
	// Example usage
	plugin := NewCSVPlugin()

	ctx := context.Background()
	stepConfig := pipelines.StepConfig{
		Name:   "csv-input",
		Plugin: "Input.csv",
		Config: map[string]any{
			"file_path":   "/path/to/data.csv",
			"has_headers": true,
			"delimiter":   ",",
		},
		Output: "csv_data",
	}

	globalContext := pipelines.NewPluginContext()

	result, err := plugin.ExecuteStep(ctx, stepConfig, globalContext)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("CSV parsing completed successfully\n")
	if data, ok := result.Get("csv_data"); ok {
		fmt.Printf("Result: %+v\n", data)
	}
}
