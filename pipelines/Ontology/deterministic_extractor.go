package ontology

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/agnivade/levenshtein"
)

// DeterministicExtractor extracts entities from structured data (CSV, JSON)
type DeterministicExtractor struct {
	config ExtractionConfig
}

// NewDeterministicExtractor creates a new deterministic extractor
func NewDeterministicExtractor(config ExtractionConfig) *DeterministicExtractor {
	return &DeterministicExtractor{
		config: config,
	}
}

// Extract extracts entities from structured data
func (e *DeterministicExtractor) Extract(data any, ontology *OntologyContext) (*ExtractionResult, error) {
	switch e.config.SourceType {
	case SourceTypeCSV:
		return e.extractFromCSV(data, ontology)
	case SourceTypeJSON:
		return e.extractFromJSON(data, ontology)
	default:
		return nil, fmt.Errorf("unsupported source type for deterministic extraction: %s", e.config.SourceType)
	}
}

// GetType returns the extraction type
func (e *DeterministicExtractor) GetType() ExtractionType {
	return ExtractionDeterministic
}

// GetSupportedSourceTypes returns supported source types
func (e *DeterministicExtractor) GetSupportedSourceTypes() []string {
	return []string{SourceTypeCSV, SourceTypeJSON}
}

// extractFromCSV extracts entities from CSV data
func (e *DeterministicExtractor) extractFromCSV(data any, ontology *OntologyContext) (*ExtractionResult, error) {
	csvContent, ok := data.(string)
	if !ok {
		return nil, fmt.Errorf("CSV data must be a string")
	}

	reader := csv.NewReader(strings.NewReader(csvContent))
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to parse CSV: %w", err)
	}

	if len(records) == 0 {
		return &ExtractionResult{
			ExtractionType: ExtractionDeterministic,
			Confidence:     1.0,
		}, nil
	}

	// Extract headers
	headers := records[0]

	// Infer mappings if not provided
	mappings := e.config.Mappings
	if len(mappings) == 0 {
		mappings = e.inferMappings(headers, ontology)
	}

	// Build mapping lookup
	fieldToProperty := make(map[string]string)
	for _, mapping := range mappings {
		fieldToProperty[mapping.SourceField] = mapping.PropertyURI
	}

	var entities []Entity
	var triples []Triple
	warnings := []string{}

	// Process data rows
	for i, record := range records[1:] {
		if len(record) != len(headers) {
			warnings = append(warnings, fmt.Sprintf("Row %d has %d columns but expected %d", i+1, len(record), len(headers)))
			continue
		}

		// Generate entity URI
		entityURI := fmt.Sprintf("%s/entity_%d", ontology.BaseURI, i+1)

		// Infer entity type from data
		entityType := e.inferEntityType(headers, record, ontology)

		// Create entity
		entity := Entity{
			URI:        entityURI,
			Type:       entityType,
			Properties: make(map[string]any),
			Confidence: 1.0, // Deterministic extraction is always confident
		}

		// Add type triple
		triples = append(triples, Triple{
			Subject:   entityURI,
			Predicate: "http://www.w3.org/1999/02/22-rdf-syntax-ns#type",
			Object:    entityType,
		})

		// Map columns to properties
		for j, value := range record {
			if j >= len(headers) {
				continue
			}

			header := headers[j]
			propURI, ok := fieldToProperty[header]
			if !ok {
				// Create temporary property if no mapping found
				propURI = fmt.Sprintf("%s/prop_%s", ontology.BaseURI, normalizeFieldName(header))
			}

			// Store in entity properties
			entity.Properties[propURI] = value

			// Create triple
			if value != "" {
				triple := Triple{
					Subject:   entityURI,
					Predicate: propURI,
					Object:    value,
				}

				// Try to determine datatype
				datatype := inferDatatype(value)
				if datatype != "" {
					triple.Datatype = datatype
				}

				triples = append(triples, triple)
			}
		}

		// Add label if we can infer one
		if label := e.inferLabel(headers, record); label != "" {
			entity.Label = label
			triples = append(triples, Triple{
				Subject:   entityURI,
				Predicate: "http://www.w3.org/2000/01/rdf-schema#label",
				Object:    label,
			})
		}

		entities = append(entities, entity)
	}

	return &ExtractionResult{
		Entities:          entities,
		Triples:           triples,
		EntitiesExtracted: len(entities),
		TriplesGenerated:  len(triples),
		Confidence:        1.0,
		ExtractionType:    ExtractionDeterministic,
		Warnings:          warnings,
	}, nil
}

// extractFromJSON extracts entities from JSON data
func (e *DeterministicExtractor) extractFromJSON(data any, ontology *OntologyContext) (*ExtractionResult, error) {
	var jsonData []map[string]any

	// Handle different JSON input types
	switch v := data.(type) {
	case string:
		if err := json.Unmarshal([]byte(v), &jsonData); err != nil {
			return nil, fmt.Errorf("failed to parse JSON: %w", err)
		}
	case []map[string]any:
		jsonData = v
	case []any:
		// Convert to []map[string]any
		for _, item := range v {
			if m, ok := item.(map[string]any); ok {
				jsonData = append(jsonData, m)
			}
		}
	default:
		return nil, fmt.Errorf("JSON data must be a string or array of objects")
	}

	if len(jsonData) == 0 {
		return &ExtractionResult{
			ExtractionType: ExtractionDeterministic,
			Confidence:     1.0,
		}, nil
	}

	// Extract all field names from first object
	var fields []string
	for key := range jsonData[0] {
		fields = append(fields, key)
	}

	// Infer mappings if not provided
	mappings := e.config.Mappings
	if len(mappings) == 0 {
		mappings = e.inferMappings(fields, ontology)
	}

	// Build mapping lookup
	fieldToProperty := make(map[string]string)
	for _, mapping := range mappings {
		fieldToProperty[mapping.SourceField] = mapping.PropertyURI
	}

	var entities []Entity
	var triples []Triple
	warnings := []string{}

	// Process each JSON object
	for i, obj := range jsonData {
		// Generate entity URI
		entityURI := fmt.Sprintf("%s/entity_%d", ontology.BaseURI, i+1)

		// Try to infer entity type
		entityType := e.inferEntityTypeFromJSON(obj, ontology)

		// Create entity
		entity := Entity{
			URI:        entityURI,
			Type:       entityType,
			Properties: make(map[string]any),
			Confidence: 1.0,
		}

		// Add type triple
		triples = append(triples, Triple{
			Subject:   entityURI,
			Predicate: "http://www.w3.org/1999/02/22-rdf-syntax-ns#type",
			Object:    entityType,
		})

		// Map JSON fields to properties
		for field, value := range obj {
			propURI, ok := fieldToProperty[field]
			if !ok {
				propURI = fmt.Sprintf("%s/prop_%s", ontology.BaseURI, normalizeFieldName(field))
			}

			// Store in entity properties
			entity.Properties[propURI] = value

			// Create triple
			if value != nil {
				valueStr := fmt.Sprintf("%v", value)
				if valueStr != "" {
					triple := Triple{
						Subject:   entityURI,
						Predicate: propURI,
						Object:    valueStr,
					}

					// Try to determine datatype
					datatype := inferDatatypeFromValue(value)
					if datatype != "" {
						triple.Datatype = datatype
					}

					triples = append(triples, triple)
				}
			}
		}

		// Try to add label
		if label := e.inferLabelFromJSON(obj); label != "" {
			entity.Label = label
			triples = append(triples, Triple{
				Subject:   entityURI,
				Predicate: "http://www.w3.org/2000/01/rdf-schema#label",
				Object:    label,
			})
		}

		entities = append(entities, entity)
	}

	return &ExtractionResult{
		Entities:          entities,
		Triples:           triples,
		EntitiesExtracted: len(entities),
		TriplesGenerated:  len(triples),
		Confidence:        1.0,
		ExtractionType:    ExtractionDeterministic,
		Warnings:          warnings,
	}, nil
}

// inferMappings uses fuzzy matching to map field names to ontology properties
func (e *DeterministicExtractor) inferMappings(fields []string, ontology *OntologyContext) []PropertyMapping {
	var mappings []PropertyMapping

	for _, field := range fields {
		normalized := normalizeFieldName(field)

		bestMatch := ""
		bestScore := 0.0

		for _, prop := range ontology.Properties {
			propLabel := strings.ToLower(prop.Label)
			propLocalName := extractLocalName(prop.URI)

			// Calculate similarity scores
			labelScore := stringSimilarity(normalized, propLabel)
			uriScore := stringSimilarity(normalized, propLocalName)

			score := maxFloat(labelScore, uriScore)

			if score > bestScore {
				bestScore = score
				bestMatch = prop.URI
			}
		}

		// Only use mapping if confidence is high enough
		if bestScore > 0.6 {
			mappings = append(mappings, PropertyMapping{
				SourceField: field,
				PropertyURI: bestMatch,
				Confidence:  bestScore,
			})
		} else {
			// Create a default mapping
			mappings = append(mappings, PropertyMapping{
				SourceField: field,
				PropertyURI: fmt.Sprintf("%s/prop_%s", ontology.BaseURI, normalized),
				Confidence:  0.5,
			})
		}
	}

	return mappings
}

// inferEntityType tries to infer the entity type from the data
func (e *DeterministicExtractor) inferEntityType(headers []string, record []string, ontology *OntologyContext) string {
	// Look for explicit type field
	for i, header := range headers {
		if strings.ToLower(header) == "type" || strings.ToLower(header) == "class" {
			if i < len(record) {
				typeName := record[i]
				// Try to find matching class
				for _, class := range ontology.Classes {
					if strings.EqualFold(class.Label, typeName) || strings.HasSuffix(class.URI, "/"+typeName) {
						return class.URI
					}
				}
			}
		}
	}

	// Default to first class in ontology if available
	if len(ontology.Classes) > 0 {
		return ontology.Classes[0].URI
	}

	// Fallback to owl:Thing
	return "http://www.w3.org/2002/07/owl#Thing"
}

// inferEntityTypeFromJSON tries to infer entity type from JSON object
func (e *DeterministicExtractor) inferEntityTypeFromJSON(obj map[string]any, ontology *OntologyContext) string {
	// Look for explicit type field
	if typeVal, ok := obj["type"]; ok {
		typeName := fmt.Sprintf("%v", typeVal)
		for _, class := range ontology.Classes {
			if strings.EqualFold(class.Label, typeName) || strings.HasSuffix(class.URI, "/"+typeName) {
				return class.URI
			}
		}
	}

	// Default to first class
	if len(ontology.Classes) > 0 {
		return ontology.Classes[0].URI
	}

	return "http://www.w3.org/2002/07/owl#Thing"
}

// inferLabel tries to find a good label for the entity
func (e *DeterministicExtractor) inferLabel(headers []string, record []string) string {
	// Look for common label fields
	labelFields := []string{"name", "label", "title", "id"}

	for _, labelField := range labelFields {
		for i, header := range headers {
			if strings.EqualFold(header, labelField) && i < len(record) {
				if record[i] != "" {
					return record[i]
				}
			}
		}
	}

	// Use first non-empty field
	for _, value := range record {
		if value != "" {
			return value
		}
	}

	return ""
}

// inferLabelFromJSON tries to find a good label from JSON object
func (e *DeterministicExtractor) inferLabelFromJSON(obj map[string]any) string {
	labelFields := []string{"name", "label", "title", "id"}

	for _, field := range labelFields {
		if val, ok := obj[field]; ok && val != nil {
			return fmt.Sprintf("%v", val)
		}
	}

	// Use first string value
	for _, val := range obj {
		if val != nil {
			return fmt.Sprintf("%v", val)
		}
	}

	return ""
}

// Helper functions

func normalizeFieldName(field string) string {
	// Convert to lowercase and replace spaces/underscores with nothing
	normalized := strings.ToLower(field)
	normalized = strings.ReplaceAll(normalized, " ", "_")
	normalized = strings.ReplaceAll(normalized, "-", "_")
	return normalized
}

func extractLocalName(uri string) string {
	parts := strings.Split(uri, "/")
	if len(parts) > 0 {
		localName := parts[len(parts)-1]
		// Also try splitting by #
		parts = strings.Split(localName, "#")
		if len(parts) > 1 {
			return parts[len(parts)-1]
		}
		return localName
	}
	return uri
}

func stringSimilarity(a, b string) float64 {
	a = strings.ToLower(strings.TrimSpace(a))
	b = strings.ToLower(strings.TrimSpace(b))

	if a == b {
		return 1.0
	}

	// Use Levenshtein distance
	distance := levenshtein.ComputeDistance(a, b)
	maxLen := maxInt(len(a), len(b))

	if maxLen == 0 {
		return 0.0
	}

	return 1.0 - (float64(distance) / float64(maxLen))
}

func inferDatatype(value string) string {
	// Try to infer XSD datatype from string value
	if value == "" {
		return ""
	}

	// Check for boolean
	if value == "true" || value == "false" {
		return "http://www.w3.org/2001/XMLSchema#boolean"
	}

	// Check for integer
	if _, err := fmt.Sscanf(value, "%d", new(int)); err == nil {
		return "http://www.w3.org/2001/XMLSchema#integer"
	}

	// Check for decimal
	if _, err := fmt.Sscanf(value, "%f", new(float64)); err == nil {
		return "http://www.w3.org/2001/XMLSchema#decimal"
	}

	// Default to string
	return "http://www.w3.org/2001/XMLSchema#string"
}

func inferDatatypeFromValue(value any) string {
	switch value.(type) {
	case bool:
		return "http://www.w3.org/2001/XMLSchema#boolean"
	case int, int32, int64:
		return "http://www.w3.org/2001/XMLSchema#integer"
	case float32, float64:
		return "http://www.w3.org/2001/XMLSchema#decimal"
	default:
		return "http://www.w3.org/2001/XMLSchema#string"
	}
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
