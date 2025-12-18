package ml

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// JSONDataAdapter converts JSON arrays to UnifiedDataset
type JSONDataAdapter struct {
	BaseDataAdapter
}

// NewJSONDataAdapter creates a new JSON adapter
func NewJSONDataAdapter() *JSONDataAdapter {
	return &JSONDataAdapter{
		BaseDataAdapter: NewBaseDataAdapter("json", "Extract data from JSON arrays"),
	}
}

// Supports checks if this adapter can handle the config
func (j *JSONDataAdapter) Supports(config DataSourceConfig) bool {
	if config.Type == "json" {
		return true
	}

	// Check if data looks like JSON
	if dataMap, ok := config.Data.(map[string]interface{}); ok {
		if _, hasContent := dataMap["content"]; hasContent {
			return true
		}
		if _, hasFilePath := dataMap["file_path"]; hasFilePath {
			return true
		}
		if _, hasArray := dataMap["array"]; hasArray {
			return true
		}
	}

	return false
}

// ValidateConfig validates the JSON configuration
func (j *JSONDataAdapter) ValidateConfig(config DataSourceConfig) error {
	dataMap, ok := config.Data.(map[string]interface{})
	if !ok {
		return fmt.Errorf("data must be a map with JSON configuration")
	}

	// Must have either file_path, content, or array
	_, hasFilePath := dataMap["file_path"]
	_, hasContent := dataMap["content"]
	_, hasArray := dataMap["array"]

	if !hasFilePath && !hasContent && !hasArray {
		return fmt.Errorf("either file_path, content, or array is required")
	}

	return nil
}

// Extract converts JSON data to UnifiedDataset
func (j *JSONDataAdapter) Extract(ctx context.Context, config DataSourceConfig) (*UnifiedDataset, error) {
	dataMap, ok := config.Data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("data must be a map")
	}

	var jsonData interface{}
	var err error

	// Handle different input methods
	if arrayData, ok := dataMap["array"]; ok {
		// Direct array of objects
		jsonData = arrayData
	} else if filePath, ok := dataMap["file_path"].(string); ok {
		// Read from file
		jsonData, err = j.readJSONFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read JSON file: %w", err)
		}
	} else if content, ok := dataMap["content"].(string); ok {
		// Parse from string
		jsonData, err = j.parseJSONString(content)
		if err != nil {
			return nil, fmt.Errorf("failed to parse JSON content: %w", err)
		}
	} else {
		return nil, fmt.Errorf("no valid JSON source provided")
	}

	// Extract path if specified (for nested JSON)
	if path, ok := dataMap["path"].(string); ok {
		jsonData, err = j.extractPath(jsonData, path)
		if err != nil {
			return nil, fmt.Errorf("failed to extract path '%s': %w", path, err)
		}
	}

	// Convert to dataset
	return j.convertToDataset(jsonData, config.Options)
}

// readJSONFile reads and parses a JSON file
func (j *JSONDataAdapter) readJSONFile(filePath string) (interface{}, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var result interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return result, nil
}

// parseJSONString parses a JSON string
func (j *JSONDataAdapter) parseJSONString(content string) (interface{}, error) {
	// Try base64 decode first
	if decoded, err := base64.StdEncoding.DecodeString(content); err == nil {
		content = string(decoded)
	}

	var result interface{}
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return result, nil
}

// extractPath extracts a nested path from JSON object
// Path format: "data.items.records" (dot-separated)
func (j *JSONDataAdapter) extractPath(data interface{}, path string) (interface{}, error) {
	if path == "" {
		return data, nil
	}

	parts := strings.Split(path, ".")
	current := data

	for _, part := range parts {
		switch v := current.(type) {
		case map[string]interface{}:
			val, ok := v[part]
			if !ok {
				return nil, fmt.Errorf("path component not found: %s", part)
			}
			current = val
		default:
			return nil, fmt.Errorf("cannot traverse path through non-object at: %s", part)
		}
	}

	return current, nil
}

// convertToDataset converts JSON array to UnifiedDataset
func (j *JSONDataAdapter) convertToDataset(jsonData interface{}, options DataSourceOptions) (*UnifiedDataset, error) {
	// JSON data must be an array of objects
	arrayData, ok := jsonData.([]interface{})
	if !ok {
		return nil, fmt.Errorf("JSON data must be an array, got: %T", jsonData)
	}

	if len(arrayData) == 0 {
		return j.createEmptyDataset(), nil
	}

	// Convert array to rows
	rows := make([]map[string]interface{}, 0, len(arrayData))
	for i, item := range arrayData {
		row, ok := item.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("array item %d is not an object, got: %T", i, item)
		}
		rows = append(rows, row)
	}

	// Apply row limit if specified
	if options.Limit > 0 && len(rows) > options.Limit {
		rows = rows[:options.Limit]
	}

	// Infer schema from first row
	columnNames := j.extractColumnNames(rows[0], options.SelectedColumns)

	// Create dataset
	dataset := NewUnifiedDataset("json")
	dataset.Rows = rows
	dataset.RowCount = len(rows)
	dataset.ColumnCount = len(columnNames)

	// Build column metadata
	dataset.Columns = make([]ColumnMetadata, len(columnNames))
	for i, colName := range columnNames {
		colMeta := ColumnMetadata{
			Name:  colName,
			Index: i,
		}

		// Infer data type
		colMeta.DataType = j.inferColumnType(rows, colName)
		colMeta.IsNumeric = (colMeta.DataType == "numeric" || colMeta.DataType == "integer")
		colMeta.IsTimeSeries = (colMeta.DataType == "datetime")
		colMeta.IsDateTime = (colMeta.DataType == "datetime")

		// Apply type hints if provided
		if options.TypeHints != nil {
			if hintType, exists := options.TypeHints[colName]; exists {
				colMeta.DataType = hintType
				colMeta.IsNumeric = (hintType == "numeric" || hintType == "integer")
				colMeta.IsTimeSeries = (hintType == "datetime")
				colMeta.IsDateTime = (hintType == "datetime")
			}
		}

		// Compute stats for numeric columns
		if colMeta.IsNumeric {
			colMeta.Stats = j.computeColumnStats(rows, colName)
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

		dataset.Columns[i] = colMeta
	}

	// Detect time series structure
	dataset.TimeSeriesConfig = detectTimeSeriesStructure(dataset)

	// Store metadata
	dataset.SourceInfo["array_length"] = len(arrayData)

	return dataset, nil
}

// extractColumnNames gets column names from a row, optionally filtered
func (j *JSONDataAdapter) extractColumnNames(row map[string]interface{}, selectedColumns []string) []string {
	var columns []string

	if len(selectedColumns) > 0 {
		// Use selected columns (in order)
		for _, col := range selectedColumns {
			if _, exists := row[col]; exists {
				columns = append(columns, col)
			}
		}
	} else {
		// Use all columns from row
		for colName := range row {
			columns = append(columns, colName)
		}
	}

	return columns
}

// inferColumnType infers the data type of a column from sample values
func (j *JSONDataAdapter) inferColumnType(rows []map[string]interface{}, columnName string) string {
	if len(rows) == 0 {
		return "string"
	}

	numericCount := 0
	datetimeCount := 0
	boolCount := 0
	nullCount := 0
	sampleSize := 10
	if len(rows) < sampleSize {
		sampleSize = len(rows)
	}

	for i := 0; i < sampleSize; i++ {
		value := rows[i][columnName]

		if value == nil {
			nullCount++
			continue
		}

		// Check JSON native types
		switch v := value.(type) {
		case float64, int, int64:
			numericCount++
		case bool:
			boolCount++
		case string:
			// Check if datetime string
			if j.isDateTime(v) {
				datetimeCount++
			}
		}
	}

	// Determine type based on majority (excluding nulls)
	nonNullCount := sampleSize - nullCount
	if nonNullCount == 0 {
		return "string"
	}

	if numericCount >= nonNullCount/2 {
		return "numeric"
	}
	if datetimeCount >= nonNullCount/2 {
		return "datetime"
	}
	if boolCount >= nonNullCount/2 {
		return "boolean"
	}

	return "string"
}

// isDateTime checks if a string represents a datetime
func (j *JSONDataAdapter) isDateTime(value string) bool {
	dateFormats := []string{
		"2006-01-02",
		"2006-01-02T15:04:05Z07:00",
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
func (j *JSONDataAdapter) computeColumnStats(rows []map[string]interface{}, columnName string) *ColumnStats {
	stats := &ColumnStats{
		Count: 0,
	}

	var values []float64

	for _, row := range rows {
		value := row[columnName]
		if value == nil {
			continue
		}

		// Convert to float64
		var numVal float64
		switch v := value.(type) {
		case float64:
			numVal = v
		case int:
			numVal = float64(v)
		case int64:
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
		stats.Count++
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

// createEmptyDataset creates an empty dataset
func (j *JSONDataAdapter) createEmptyDataset() *UnifiedDataset {
	dataset := NewUnifiedDataset("json")
	dataset.Rows = []map[string]interface{}{}
	dataset.Columns = []ColumnMetadata{}
	dataset.RowCount = 0
	dataset.ColumnCount = 0
	return dataset
}
