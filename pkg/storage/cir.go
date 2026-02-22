package storage

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

// CreateCIRFromCSV creates a CIR object from CSV data
func CreateCIRFromCSV(csvData string, sourceURI string) (*models.CIR, error) {
	reader := csv.NewReader(strings.NewReader(csvData))

	// Read all records
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to parse CSV: %w", err)
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("CSV data is empty")
	}

	// First row is headers
	headers := records[0]
	dataRecords := records[1:]

	// Convert to array of maps
	data := make([]map[string]interface{}, 0, len(dataRecords))
	for _, record := range dataRecords {
		if len(record) != len(headers) {
			continue // Skip malformed rows
		}

		row := make(map[string]interface{})
		for i, header := range headers {
			row[header] = record[i]
		}
		data = append(data, row)
	}

	// Create CIR
	cir := models.NewCIR(models.SourceTypeFile, sourceURI, models.DataFormatCSV, data)

	// Add metadata
	cir.Metadata.RecordCount = len(data)
	cir.Metadata.Encoding = "utf-8"

	// Add schema inference
	schemaInference := map[string]interface{}{
		"columns": headers,
		"types":   inferColumnTypes(headers, data),
	}
	cir.SetSchemaInference(schemaInference)

	// Update size
	cir.UpdateSize()

	return cir, nil
}

// CreateCIRFromJSON creates a CIR object from JSON data
func CreateCIRFromJSON(jsonData string, sourceURI string, sourceType models.SourceType) (*models.CIR, error) {
	var data interface{}
	if err := json.Unmarshal([]byte(jsonData), &data); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Create CIR
	cir := models.NewCIR(sourceType, sourceURI, models.DataFormatJSON, data)

	// Add metadata
	cir.Metadata.Encoding = "utf-8"

	// Count records if it's an array
	if arr, ok := data.([]interface{}); ok {
		cir.Metadata.RecordCount = len(arr)
	} else {
		cir.Metadata.RecordCount = 1
	}

	// Update size
	cir.UpdateSize()

	return cir, nil
}

// CreateCIRFromText creates a CIR object from plain text
func CreateCIRFromText(textData string, sourceURI string, sourceType models.SourceType) (*models.CIR, error) {
	// Create CIR
	cir := models.NewCIR(sourceType, sourceURI, models.DataFormatText, textData)

	// Add metadata
	cir.Metadata.Encoding = "utf-8"
	cir.Metadata.RecordCount = 1

	// Add quality metrics for text
	qualityMetrics := map[string]interface{}{
		"word_count":      len(strings.Fields(textData)),
		"sentence_count":  strings.Count(textData, ".") + strings.Count(textData, "!") + strings.Count(textData, "?"),
		"character_count": len(textData),
	}
	cir.SetQualityMetrics(qualityMetrics)

	// Update size
	cir.UpdateSize()

	return cir, nil
}

// ValidateCIRQuery validates a CIR query
func ValidateCIRQuery(query *models.CIRQuery) error {
	if query == nil {
		return fmt.Errorf("query cannot be nil")
	}

	// Validate operators
	validOperators := map[string]bool{
		"eq": true, "neq": true, "gt": true, "gte": true,
		"lt": true, "lte": true, "in": true, "like": true,
	}

	for _, condition := range query.Filters {
		if !validOperators[condition.Operator] {
			return fmt.Errorf("invalid operator: %s", condition.Operator)
		}
	}

	// Validate order by directions
	for _, orderBy := range query.OrderBy {
		if orderBy.Direction != "asc" && orderBy.Direction != "desc" {
			return fmt.Errorf("invalid order direction: %s", orderBy.Direction)
		}
	}

	return nil
}

// FilterCIRData filters an array of CIR data based on conditions
func FilterCIRData(data []interface{}, conditions []models.CIRCondition) ([]interface{}, error) {
	if len(conditions) == 0 {
		return data, nil
	}

	filtered := make([]interface{}, 0)

	for _, item := range data {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		matches := true
		for _, condition := range conditions {
			value, exists := itemMap[condition.Attribute]
			if !exists {
				matches = false
				break
			}

			if !evaluateCondition(value, condition.Operator, condition.Value) {
				matches = false
				break
			}
		}

		if matches {
			filtered = append(filtered, item)
		}
	}

	return filtered, nil
}

// evaluateCondition evaluates a single condition
func evaluateCondition(value interface{}, operator string, expected interface{}) bool {
	switch operator {
	case "eq":
		return fmt.Sprintf("%v", value) == fmt.Sprintf("%v", expected)
	case "neq":
		return fmt.Sprintf("%v", value) != fmt.Sprintf("%v", expected)
	case "like":
		valueStr := fmt.Sprintf("%v", value)
		expectedStr := fmt.Sprintf("%v", expected)
		return strings.Contains(strings.ToLower(valueStr), strings.ToLower(expectedStr))
	// For numeric comparisons, we'd need type-aware logic
	// This is a simplified implementation
	default:
		return false
	}
}

// inferColumnTypes attempts to infer column types from data
func inferColumnTypes(headers []string, data []map[string]interface{}) []string {
	types := make([]string, len(headers))

	for i := range headers {
		types[i] = "string" // Default to string
	}

	// In a more sophisticated implementation, we would analyze the data
	// to infer types like number, date, boolean, etc.

	return types
}

// MergeCIRData merges multiple CIR objects into one
// Useful for combining data from multiple ingestion runs
func MergeCIRData(cirs []*models.CIR) (*models.CIR, error) {
	if len(cirs) == 0 {
		return nil, fmt.Errorf("no CIR objects to merge")
	}

	if len(cirs) == 1 {
		return cirs[0], nil
	}

	// Use first CIR as base
	merged := &models.CIR{
		Version: cirs[0].Version,
		Source:  cirs[0].Source,
		Metadata: models.CIRMetadata{
			Encoding: cirs[0].Metadata.Encoding,
		},
	}

	// Merge data - assume all are arrays
	mergedData := make([]interface{}, 0)
	totalRecords := 0

	for _, cir := range cirs {
		if arr, err := cir.GetDataAsArray(); err == nil {
			mergedData = append(mergedData, arr...)
			totalRecords += len(arr)
		}
	}

	merged.Data = mergedData
	merged.Metadata.RecordCount = totalRecords
	merged.UpdateSize()

	return merged, nil
}
