package storage

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

func mapCIRToOntology(cir *models.CIR, compiled *models.CompiledOntology) (*models.CIRSemanticMapping, error) {
	if cir == nil {
		return nil, fmt.Errorf("CIR cannot be nil")
	}
	if compiled == nil {
		return nil, fmt.Errorf("compiled ontology cannot be nil")
	}
	dataMap, err := cir.GetDataAsMap()
	if err != nil {
		return nil, fmt.Errorf("semantic mapping requires object CIR data: %w", err)
	}
	classID, matchedBy := resolveSemanticClass(cir, dataMap, compiled)
	mapping := &models.CIRSemanticMapping{
		OntologyID:  compiled.OntologyID,
		ContentHash: compiled.ContentHash,
		ClassID:     classID,
		MatchedBy:   matchedBy,
		Properties:  map[string]models.SemanticProperty{},
	}
	if classID == "" {
		mapping.Violations = append(mapping.Violations, models.SemanticMappingViolation{Code: "class_unresolved", Message: "CIR record could not be matched to an ontology class"})
	}

	properties := propertiesForClass(compiled, classID)
	unmapped := make([]string, 0)
	for field, value := range dataMap {
		if isSemanticControlField(field) {
			continue
		}
		property, ok := resolveSemanticProperty(field, properties)
		if !ok {
			unmapped = append(unmapped, field)
			continue
		}
		if violation := validateSemanticValue(field, value, property.Range); violation != nil {
			mapping.Violations = append(mapping.Violations, *violation)
		}
		mapping.Properties[property.ID] = models.SemanticProperty{PropertyID: property.ID, SourceField: field, Value: value, Range: property.Range, Kind: property.Kind}
	}
	sort.Strings(unmapped)
	mapping.UnmappedFields = unmapped
	return mapping, nil
}

func resolveSemanticClass(cir *models.CIR, dataMap map[string]interface{}, compiled *models.CompiledOntology) (string, string) {
	classIndex := make(map[string]string, len(compiled.Classes)*3)
	for _, class := range compiled.Classes {
		for _, candidate := range append([]string{class.ID, class.Name, class.Label}, class.SearchTerms...) {
			if normalized := normalizeSemanticName(candidate); normalized != "" {
				classIndex[normalized] = class.ID
			}
		}
	}
	candidates := []struct{ value, source string }{
		{metadataString(cir.Metadata.SchemaInference, "entity_type"), "metadata.schema_inference.entity_type"},
		{metadataString(cir.Metadata.SchemaInference, "class_id"), "metadata.schema_inference.class_id"},
	}
	for _, field := range []string{"entity_type", "class_id", "type"} {
		if value, ok := dataMap[field]; ok {
			candidates = append(candidates, struct{ value, source string }{fmt.Sprint(value), "data." + field})
		}
	}
	for _, segment := range sourceURISegments(cir.Source.URI) {
		candidates = append(candidates, struct{ value, source string }{segment, "source.uri"})
	}
	for _, candidate := range candidates {
		if classID, ok := classIndex[normalizeSemanticName(candidate.value)]; ok {
			return classID, candidate.source
		}
	}
	return "", ""
}

func propertiesForClass(compiled *models.CompiledOntology, classID string) []models.CompiledOntologyProperty {
	if compiled == nil {
		return nil
	}
	out := make([]models.CompiledOntologyProperty, 0)
	for _, property := range compiled.Properties {
		if property.Kind != "datatype" && property.Kind != "object" {
			continue
		}
		if classID == "" || len(property.Domain) == 0 || stringSliceContains(property.Domain, classID) {
			out = append(out, property)
		}
	}
	return out
}

func resolveSemanticProperty(field string, properties []models.CompiledOntologyProperty) (models.CompiledOntologyProperty, bool) {
	fieldKey := normalizeSemanticName(field)
	for _, property := range properties {
		for _, candidate := range append([]string{property.ID, property.Name, property.Label}, property.SearchTerms...) {
			if normalizeSemanticName(candidate) == fieldKey {
				return property, true
			}
		}
	}
	return models.CompiledOntologyProperty{}, false
}

func validateSemanticValue(field string, value interface{}, ranges []string) *models.SemanticMappingViolation {
	if len(ranges) == 0 || value == nil {
		return nil
	}
	expected := strings.ToLower(ranges[0])
	valid := true
	switch expected {
	case "integer", "int", "long", "short", "decimal", "float", "double", "number":
		valid = isNumber(value)
	case "boolean", "bool":
		_, valid = value.(bool)
	case "date", "datetime", "time":
		valid = isTimeLike(value)
	case "string":
		_, valid = value.(string)
	default:
		return nil
	}
	if valid {
		return nil
	}
	return &models.SemanticMappingViolation{Code: "range_mismatch", Message: "CIR field value does not match ontology property range", Field: field, Expected: expected, Actual: reflect.TypeOf(value).String()}
}

func isNumber(value interface{}) bool {
	switch value.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return true
	default:
		return false
	}
}

func isTimeLike(value interface{}) bool {
	switch v := value.(type) {
	case time.Time:
		return true
	case string:
		if v == "" {
			return false
		}
		_, err := time.Parse(time.RFC3339, v)
		if err == nil {
			return true
		}
		_, err = time.Parse("2006-01-02", v)
		return err == nil
	default:
		return false
	}
}

func metadataString(values map[string]interface{}, key string) string {
	if values == nil {
		return ""
	}
	if value, ok := values[key]; ok {
		return fmt.Sprint(value)
	}
	return ""
}

func sourceURISegments(uri string) []string {
	uri = strings.TrimSpace(uri)
	if uri == "" {
		return nil
	}
	replacer := strings.NewReplacer("://", "/", "\\", "/", "?", "/", "#", "/", ":", "/")
	parts := strings.Split(replacer.Replace(uri), "/")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func normalizeSemanticName(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	var b strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return strings.ToLower(b.String())
}

func isSemanticControlField(field string) bool {
	switch strings.ToLower(field) {
	case "entity_type", "class_id", "type":
		return true
	default:
		return false
	}
}

func stringSliceContains(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}
