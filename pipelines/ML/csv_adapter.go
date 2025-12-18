package ml

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
)

// CSVDataAdapter converts CSV files to UnifiedDataset
type CSVDataAdapter struct {
	BaseDataAdapter
	pluginRegistry *pipelines.PluginRegistry
}

// NewCSVDataAdapter creates a new CSV adapter
func NewCSVDataAdapter(registry *pipelines.PluginRegistry) *CSVDataAdapter {
	return &CSVDataAdapter{
		BaseDataAdapter: NewBaseDataAdapter("csv", "Extract data from CSV files"),
		pluginRegistry:  registry,
	}
}

// Supports checks if this adapter can handle the config
func (c *CSVDataAdapter) Supports(config DataSourceConfig) bool {
	// Support CSV type or any data that looks like CSV
	if config.Type == "csv" {
		return true
	}

	// Check if data contains CSV-like content
	if dataMap, ok := config.Data.(map[string]interface{}); ok {
		if _, hasFilePath := dataMap["file_path"]; hasFilePath {
			return true
		}
		if content, hasContent := dataMap["content"].(string); hasContent && strings.Contains(content, ",") {
			return true
		}
	}

	return false
}

// ValidateConfig validates the CSV configuration
func (c *CSVDataAdapter) ValidateConfig(config DataSourceConfig) error {
	dataMap, ok := config.Data.(map[string]interface{})
	if !ok {
		return fmt.Errorf("data must be a map with CSV configuration")
	}

	// Must have either file_path or content
	_, hasFilePath := dataMap["file_path"]
	_, hasContent := dataMap["content"]

	if !hasFilePath && !hasContent {
		return fmt.Errorf("either file_path or content is required")
	}

	// Validate delimiter if provided
	if delimiter, ok := dataMap["delimiter"].(string); ok {
		if len(delimiter) != 1 {
			return fmt.Errorf("delimiter must be a single character, got: %s", delimiter)
		}
	}

	return nil
}

// Extract converts CSV data to UnifiedDataset
func (c *CSVDataAdapter) Extract(ctx context.Context, config DataSourceConfig) (*UnifiedDataset, error) {
	dataMap, ok := config.Data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("data must be a map")
	}

	// Handle both file path and direct content
	var filePath string
	if fp, ok := dataMap["file_path"].(string); ok {
		filePath = fp
	} else if content, ok := dataMap["content"].(string); ok {
		// Write content to temporary file
		tmpFile, err := c.createTempCSVFile(content)
		if err != nil {
			return nil, fmt.Errorf("failed to create temp CSV file: %w", err)
		}
		defer os.Remove(tmpFile)
		filePath = tmpFile
	} else {
		return nil, fmt.Errorf("either file_path or content is required")
	}

	// Extract options
	hasHeaders := true
	if h, ok := dataMap["has_headers"].(bool); ok {
		hasHeaders = h
	}

	delimiter := ","
	if d, ok := dataMap["delimiter"].(string); ok && len(d) == 1 {
		delimiter = d
	}

	// Use CSV plugin from registry
	csvPlugin, err := c.pluginRegistry.GetPlugin("Input", "csv")
	if err != nil {
		return nil, fmt.Errorf("CSV plugin not found in registry: %w", err)
	}

	stepConfig := pipelines.StepConfig{
		Name:   "csv_extraction",
		Plugin: "Input.csv",
		Config: map[string]any{
			"file_path":   filePath,
			"has_headers": hasHeaders,
			"delimiter":   delimiter,
		},
		Output: "csv_data",
	}

	pluginContext := pipelines.NewPluginContext()
	result, err := csvPlugin.ExecuteStep(ctx, stepConfig, pluginContext)
	if err != nil {
		return nil, fmt.Errorf("CSV plugin execution failed: %w", err)
	}

	// Extract result
	csvDataInterface, ok := result.Get("csv_data")
	if !ok {
		return nil, fmt.Errorf("CSV plugin did not return expected output")
	}

	csvData, ok := csvDataInterface.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("CSV plugin output is not a map")
	}

	// Convert to UnifiedDataset
	return c.convertToDataset(csvData, config.Options)
}

// convertToDataset converts CSV plugin output to UnifiedDataset
func (c *CSVDataAdapter) convertToDataset(csvData map[string]any, options DataSourceOptions) (*UnifiedDataset, error) {
	// Extract columns
	columnsInterface, ok := csvData["columns"]
	if !ok {
		return nil, fmt.Errorf("CSV data missing 'columns' field")
	}

	columnNames, ok := columnsInterface.([]string)
	if !ok {
		return nil, fmt.Errorf("columns field is not []string")
	}

	// Extract rows
	rowsInterface, ok := csvData["rows"]
	if !ok {
		return nil, fmt.Errorf("CSV data missing 'rows' field")
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
	dataset := NewUnifiedDataset("csv")
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
		colMeta.DataType = c.inferColumnType(rows, colName)
		colMeta.IsNumeric = (colMeta.DataType == "numeric" || colMeta.DataType == "integer")
		colMeta.IsTimeSeries = (colMeta.DataType == "datetime")

		// Compute stats for numeric columns
		if colMeta.IsNumeric {
			colMeta.Stats = c.computeColumnStats(rows, colName)
		}

		// Apply type hints if provided
		if options.TypeHints != nil {
			if hintType, exists := options.TypeHints[colName]; exists {
				colMeta.DataType = hintType
				colMeta.IsNumeric = (hintType == "numeric" || hintType == "integer")
				colMeta.IsTimeSeries = (hintType == "datetime")
			}
		}

		dataset.Columns[i] = colMeta
	}

	// Detect time series structure
	dataset.TimeSeriesConfig = detectTimeSeriesStructure(dataset)

	// Store CSV metadata
	if parsedAt, ok := csvData["parsed_at"].(string); ok {
		dataset.SourceInfo["parsed_at"] = parsedAt
	}
	if filePath, ok := csvData["file_path"].(string); ok {
		dataset.SourceInfo["file_path"] = filePath
	}

	return dataset, nil
}

// inferColumnType infers the data type of a column from sample values
func (c *CSVDataAdapter) inferColumnType(rows []map[string]any, columnName string) string {
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
			if c.isDateTime(strVal) {
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
func (c *CSVDataAdapter) isDateTime(value string) bool {
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
func (c *CSVDataAdapter) computeColumnStats(rows []map[string]any, columnName string) *ColumnStats {
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

// createTempCSVFile creates a temporary CSV file from content string
func (c *CSVDataAdapter) createTempCSVFile(content string) (string, error) {
	// Check if content is base64 encoded
	if decoded, err := base64.StdEncoding.DecodeString(content); err == nil {
		content = string(decoded)
	}

	// Create temp file
	tmpDir := os.TempDir()
	tmpFile := filepath.Join(tmpDir, fmt.Sprintf("mimir_csv_%d.csv", time.Now().UnixNano()))

	err := os.WriteFile(tmpFile, []byte(content), 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write temp file: %w", err)
	}

	return tmpFile, nil
}

// detectTimeSeriesStructure detects if dataset has time series structure
func detectTimeSeriesStructure(dataset *UnifiedDataset) *TimeSeriesInfo {
	// Look for datetime column
	var timeCol *ColumnMetadata
	for i := range dataset.Columns {
		if dataset.Columns[i].IsTimeSeries {
			timeCol = &dataset.Columns[i]
			break
		}
	}

	if timeCol == nil {
		return nil
	}

	// Find numeric columns that could be metrics
	var metricCols []string
	for _, col := range dataset.Columns {
		if col.IsNumeric && col.Name != timeCol.Name {
			metricCols = append(metricCols, col.Name)
		}
	}

	if len(metricCols) == 0 {
		return nil
	}

	return &TimeSeriesInfo{
		DateColumn:    timeCol.Name,
		MetricColumns: metricCols,
		Frequency:     "irregular", // Would need analysis to determine
		IsSorted:      false,       // Would need sorting check
		HasGaps:       false,       // Would need gap detection
	}
}
