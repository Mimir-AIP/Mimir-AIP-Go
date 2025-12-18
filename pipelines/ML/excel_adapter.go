package ml

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
)

// ExcelDataAdapter converts Excel files to UnifiedDataset
type ExcelDataAdapter struct {
	BaseDataAdapter
	pluginRegistry *pipelines.PluginRegistry
}

// NewExcelDataAdapter creates a new Excel adapter
func NewExcelDataAdapter(registry *pipelines.PluginRegistry) *ExcelDataAdapter {
	return &ExcelDataAdapter{
		BaseDataAdapter: NewBaseDataAdapter("excel", "Extract data from Excel files (.xlsx)"),
		pluginRegistry:  registry,
	}
}

// Supports checks if this adapter can handle the config
func (e *ExcelDataAdapter) Supports(config DataSourceConfig) bool {
	if config.Type == "excel" || config.Type == "xlsx" {
		return true
	}

	// Check if data contains Excel file configuration
	if dataMap, ok := config.Data.(map[string]interface{}); ok {
		if filePath, hasFilePath := dataMap["file_path"].(string); hasFilePath {
			// Check file extension
			if len(filePath) > 5 && (filePath[len(filePath)-5:] == ".xlsx" || filePath[len(filePath)-4:] == ".xls") {
				return true
			}
		}
	}

	return false
}

// ValidateConfig validates the Excel configuration
func (e *ExcelDataAdapter) ValidateConfig(config DataSourceConfig) error {
	dataMap, ok := config.Data.(map[string]interface{})
	if !ok {
		return fmt.Errorf("data must be a map with Excel configuration")
	}

	// Must have file_path
	filePath, hasFilePath := dataMap["file_path"].(string)
	if !hasFilePath || filePath == "" {
		return fmt.Errorf("file_path is required for Excel files")
	}

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("Excel file does not exist: %s", filePath)
	}

	return nil
}

// Extract converts Excel data to UnifiedDataset
func (e *ExcelDataAdapter) Extract(ctx context.Context, config DataSourceConfig) (*UnifiedDataset, error) {
	dataMap, ok := config.Data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("data must be a map")
	}

	filePath, _ := dataMap["file_path"].(string)

	// Extract options
	hasHeaders := true
	if h, ok := dataMap["has_headers"].(bool); ok {
		hasHeaders = h
	}

	sheetName := ""
	if sn, ok := dataMap["sheet_name"].(string); ok {
		sheetName = sn
	}

	// Use Excel plugin from registry
	excelPlugin, err := e.pluginRegistry.GetPlugin("Input", "excel")
	if err != nil {
		return nil, fmt.Errorf("Excel plugin not found in registry: %w", err)
	}

	stepConfig := pipelines.StepConfig{
		Name:   "excel_extraction",
		Plugin: "Input.excel",
		Config: map[string]any{
			"file_path":   filePath,
			"has_headers": hasHeaders,
			"sheet_name":  sheetName,
		},
		Output: "excel_data",
	}

	pluginContext := pipelines.NewPluginContext()
	result, err := excelPlugin.ExecuteStep(ctx, stepConfig, pluginContext)

	// Note: Excel plugin currently requires excelize library
	// If parsing fails due to missing library, return helpful error
	if err != nil {
		if ctx := result; ctx != nil {
			// Try to extract partial data if available
			if excelDataInterface, ok := ctx.Get("excel_data"); ok {
				if excelData, ok := excelDataInterface.(map[string]any); ok {
					// Check if it's the "library required" error
					if rowsInterface, hasRows := excelData["rows"]; hasRows {
						if rows, ok := rowsInterface.([]map[string]any); ok && len(rows) == 0 {
							return nil, fmt.Errorf("Excel parsing requires excelize library. Please install: go get github.com/xuri/excelize/v2\n\nAlternatively, export your Excel file to CSV format and use the CSV adapter")
						}
					}
				}
			}
		}
		return nil, fmt.Errorf("Excel plugin execution failed: %w\n\nNote: Excel support requires excelize library or converting to CSV", err)
	}

	// Extract result
	excelDataInterface, ok := result.Get("excel_data")
	if !ok {
		return nil, fmt.Errorf("Excel plugin did not return expected output")
	}

	excelData, ok := excelDataInterface.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("Excel plugin output is not a map")
	}

	// Convert to UnifiedDataset
	return e.convertToDataset(excelData, config.Options)
}

// convertToDataset converts Excel plugin output to UnifiedDataset
func (e *ExcelDataAdapter) convertToDataset(excelData map[string]any, options DataSourceOptions) (*UnifiedDataset, error) {
	// Extract columns
	columnsInterface, ok := excelData["columns"]
	if !ok {
		return nil, fmt.Errorf("Excel data missing 'columns' field")
	}

	columnNames, ok := columnsInterface.([]string)
	if !ok {
		return nil, fmt.Errorf("columns field is not []string")
	}

	// Extract rows
	rowsInterface, ok := excelData["rows"]
	if !ok {
		return nil, fmt.Errorf("Excel data missing 'rows' field")
	}

	rows, ok := rowsInterface.([]map[string]any)
	if !ok {
		return nil, fmt.Errorf("rows field is not []map[string]any")
	}

	// Apply row limit if specified
	if options.Limit > 0 && len(rows) > options.Limit {
		rows = rows[:options.Limit]
	}

	// Create dataset
	dataset := NewUnifiedDataset("excel")
	dataset.Rows = rows
	dataset.RowCount = len(rows)
	dataset.ColumnCount = len(columnNames)

	// Build column metadata with type inference
	dataset.Columns = make([]ColumnMetadata, len(columnNames))
	for i, colName := range columnNames {
		colMeta := ColumnMetadata{
			Name:  colName,
			Index: i,
		}

		// Infer data type from first non-empty value in rows
		colMeta.DataType = e.inferColumnType(rows, colName)
		colMeta.IsNumeric = (colMeta.DataType == "numeric" || colMeta.DataType == "integer")
		colMeta.IsTimeSeries = (colMeta.DataType == "datetime")
		colMeta.IsDateTime = (colMeta.DataType == "datetime")

		// Compute stats for numeric columns
		if colMeta.IsNumeric {
			colMeta.Stats = e.computeColumnStats(rows, colName)
		}

		// Count nulls
		nullCount := 0
		for _, row := range rows {
			if row[colName] == nil {
				nullCount++
			}
		}
		colMeta.HasNulls = (nullCount > 0)
		colMeta.NullCount = nullCount

		// Apply type hints if provided
		if options.TypeHints != nil {
			if hintType, exists := options.TypeHints[colName]; exists {
				colMeta.DataType = hintType
				colMeta.IsNumeric = (hintType == "numeric" || hintType == "integer")
				colMeta.IsTimeSeries = (hintType == "datetime")
				colMeta.IsDateTime = (hintType == "datetime")
			}
		}

		dataset.Columns[i] = colMeta
	}

	// Detect time series structure
	dataset.TimeSeriesConfig = detectTimeSeriesStructure(dataset)

	// Store Excel metadata
	if parsedAt, ok := excelData["parsed_at"].(string); ok {
		dataset.SourceInfo["parsed_at"] = parsedAt
	}
	if filePath, ok := excelData["file_path"].(string); ok {
		dataset.SourceInfo["file_path"] = filePath
	}
	if sheetName, ok := excelData["sheet_name"].(string); ok {
		dataset.SourceInfo["sheet_name"] = sheetName
	}
	if availableSheets, ok := excelData["available_sheets"].([]string); ok {
		dataset.SourceInfo["available_sheets"] = availableSheets
	}

	return dataset, nil
}

// inferColumnType infers the data type of a column from sample values
func (e *ExcelDataAdapter) inferColumnType(rows []map[string]any, columnName string) string {
	if len(rows) == 0 {
		return "string"
	}

	numericCount := 0
	datetimeCount := 0
	boolCount := 0
	sampleSize := 10
	if len(rows) < sampleSize {
		sampleSize = len(rows)
	}

	for i := 0; i < sampleSize; i++ {
		value := rows[i][columnName]

		// Check if numeric
		if _, ok := value.(float64); ok {
			numericCount++
			continue
		}
		if _, ok := value.(int); ok {
			numericCount++
			continue
		}

		// Check if boolean
		if _, ok := value.(bool); ok {
			boolCount++
			continue
		}

		// Check if datetime (string that can be parsed)
		if strVal, ok := value.(string); ok {
			if e.isDateTime(strVal) {
				datetimeCount++
				continue
			}
		}
	}

	// Determine type based on majority
	if numericCount >= sampleSize/2 {
		return "numeric"
	}
	if datetimeCount >= sampleSize/2 {
		return "datetime"
	}
	if boolCount >= sampleSize/2 {
		return "boolean"
	}

	return "string"
}

// isDateTime checks if a string represents a datetime
func (e *ExcelDataAdapter) isDateTime(value string) bool {
	dateFormats := []string{
		"2006-01-02",
		"2006-01-02 15:04:05",
		"2006/01/02",
		"01/02/2006",
		"02-Jan-2006",
		time.RFC3339,
		time.RFC822,
	}

	for _, format := range dateFormats {
		if _, err := time.Parse(format, value); err == nil {
			return true
		}
	}

	return false
}

// computeColumnStats computes statistics for numeric columns
func (e *ExcelDataAdapter) computeColumnStats(rows []map[string]any, columnName string) *ColumnStats {
	stats := &ColumnStats{
		Count: 0,
	}

	var values []float64

	for _, row := range rows {
		value := row[columnName]
		if value == nil {
			continue
		}

		stats.Count++

		// Convert to float64
		var numVal float64
		switch v := value.(type) {
		case float64:
			numVal = v
		case int:
			numVal = float64(v)
		case string:
			parsed, err := strconv.ParseFloat(v, 64)
			if err != nil {
				continue
			}
			numVal = parsed
		default:
			continue
		}

		values = append(values, numVal)
	}

	if len(values) == 0 {
		return stats
	}

	// Compute min, max, mean, sum
	min := values[0]
	max := values[0]
	sum := 0.0

	for _, v := range values {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
		sum += v
	}

	stats.Min = min
	stats.Max = max
	stats.Sum = sum
	stats.Mean = sum / float64(len(values))

	return stats
}
