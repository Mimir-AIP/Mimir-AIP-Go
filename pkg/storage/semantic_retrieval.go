package storage

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

func (s *Service) RetrieveByOntology(projectID, ontologyID string, req *models.OntologyRetrieveRequest) (*models.OntologyRetrieveResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("retrieve request is required")
	}
	if strings.TrimSpace(projectID) == "" {
		return nil, fmt.Errorf("project_id is required")
	}
	compiled, err := s.store.GetCompiledOntology(ontologyID)
	if err != nil {
		return nil, fmt.Errorf("compiled ontology not found: %w", err)
	}
	if compiled.ProjectID != projectID {
		return nil, fmt.Errorf("ontology %s belongs to project %s, not %s", ontologyID, compiled.ProjectID, projectID)
	}
	classID := resolveClassIDForRetrieval(req.ClassID, compiled)
	if strings.TrimSpace(req.ClassID) != "" && classID == "" {
		return nil, fmt.Errorf("ontology class not found: %s", req.ClassID)
	}
	propertyIDs, err := resolvePropertyIDsForRetrieval(req.Properties, classID, compiled)
	if err != nil {
		return nil, err
	}
	filters, err := resolveFiltersForRetrieval(req.Filters, classID, compiled)
	if err != nil {
		return nil, err
	}
	limit := req.Limit
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	configs, err := s.store.ListStorageConfigsByProject(projectID)
	if err != nil {
		return nil, fmt.Errorf("list storage configs: %w", err)
	}
	allowedStorage := stringSet(req.StorageIDs)
	results := make([]models.OntologyRetrieveResult, 0)
	for _, cfg := range configs {
		if cfg == nil || cfg.OntologyID != ontologyID || !cfg.Active {
			continue
		}
		if len(allowedStorage) > 0 {
			if _, ok := allowedStorage[cfg.ID]; !ok {
				continue
			}
		}
		cirs, err := s.retrieveWithConfig(cfg, &models.CIRQuery{Limit: limit})
		if err != nil {
			return nil, fmt.Errorf("retrieve from storage %s: %w", cfg.ID, err)
		}
		for _, cir := range cirs {
			if cir == nil || cir.Metadata.Ontology == nil {
				continue
			}
			mapping := cir.Metadata.Ontology
			if mapping.OntologyID != ontologyID || mapping.ContentHash != compiled.ContentHash {
				continue
			}
			if classID != "" && mapping.ClassID != classID {
				continue
			}
			if !semanticFiltersMatch(mapping, filters) {
				continue
			}
			results = append(results, models.OntologyRetrieveResult{StorageID: cfg.ID, ClassID: mapping.ClassID, Properties: projectSemanticProperties(mapping, propertyIDs), Source: cir.Source, Violations: mapping.Violations})
			if len(results) >= limit {
				return &models.OntologyRetrieveResponse{OntologyID: ontologyID, ContentHash: compiled.ContentHash, Results: results, Count: len(results)}, nil
			}
		}
	}
	return &models.OntologyRetrieveResponse{OntologyID: ontologyID, ContentHash: compiled.ContentHash, Results: results, Count: len(results)}, nil
}

type resolvedRetrieveFilter struct {
	PropertyID string
	Operator   string
	Value      interface{}
}

func resolveClassIDForRetrieval(class string, compiled *models.CompiledOntology) string {
	class = normalizeSemanticName(class)
	if class == "" {
		return ""
	}
	for _, candidate := range compiled.Classes {
		for _, value := range append([]string{candidate.ID, candidate.Name, candidate.Label}, candidate.SearchTerms...) {
			if normalizeSemanticName(value) == class {
				return candidate.ID
			}
		}
	}
	return ""
}

func resolvePropertyIDsForRetrieval(properties []string, classID string, compiled *models.CompiledOntology) (map[string]struct{}, error) {
	if len(properties) == 0 {
		return nil, nil
	}
	out := map[string]struct{}{}
	available := propertiesForClass(compiled, classID)
	for _, requested := range properties {
		property, ok := resolveSemanticProperty(requested, available)
		if !ok {
			return nil, fmt.Errorf("ontology property not found: %s", requested)
		}
		out[property.ID] = struct{}{}
	}
	return out, nil
}

func resolveFiltersForRetrieval(filters []models.OntologyRetrieveFilter, classID string, compiled *models.CompiledOntology) ([]resolvedRetrieveFilter, error) {
	out := make([]resolvedRetrieveFilter, 0, len(filters))
	available := propertiesForClass(compiled, classID)
	for _, filter := range filters {
		property, ok := resolveSemanticProperty(filter.Property, available)
		if !ok {
			return nil, fmt.Errorf("ontology filter property not found: %s", filter.Property)
		}
		operator := strings.ToLower(strings.TrimSpace(filter.Operator))
		if operator == "" {
			operator = "eq"
		}
		out = append(out, resolvedRetrieveFilter{PropertyID: property.ID, Operator: operator, Value: filter.Value})
	}
	return out, nil
}

func semanticFiltersMatch(mapping *models.CIRSemanticMapping, filters []resolvedRetrieveFilter) bool {
	for _, filter := range filters {
		property, ok := mapping.Properties[filter.PropertyID]
		if !ok {
			return false
		}
		if !compareSemanticValue(property.Value, filter.Operator, filter.Value) {
			return false
		}
	}
	return true
}

func compareSemanticValue(actual interface{}, operator string, expected interface{}) bool {
	switch operator {
	case "eq", "=", "==":
		return reflect.DeepEqual(actual, expected) || fmt.Sprint(actual) == fmt.Sprint(expected)
	case "neq", "!=":
		return !compareSemanticValue(actual, "eq", expected)
	case "gt", "gte", "lt", "lte":
		actualNumber, actualOK := toFloat(actual)
		expectedNumber, expectedOK := toFloat(expected)
		if !actualOK || !expectedOK {
			return false
		}
		switch operator {
		case "gt":
			return actualNumber > expectedNumber
		case "gte":
			return actualNumber >= expectedNumber
		case "lt":
			return actualNumber < expectedNumber
		case "lte":
			return actualNumber <= expectedNumber
		}
	case "contains":
		return strings.Contains(strings.ToLower(fmt.Sprint(actual)), strings.ToLower(fmt.Sprint(expected)))
	}
	return false
}

func toFloat(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case int:
		return float64(v), true
	case int8:
		return float64(v), true
	case int16:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	case uint:
		return float64(v), true
	case uint8:
		return float64(v), true
	case uint16:
		return float64(v), true
	case uint32:
		return float64(v), true
	case uint64:
		return float64(v), true
	case float32:
		return float64(v), true
	case float64:
		return v, true
	default:
		return 0, false
	}
}

func projectSemanticProperties(mapping *models.CIRSemanticMapping, propertyIDs map[string]struct{}) map[string]interface{} {
	out := map[string]interface{}{}
	keys := make([]string, 0, len(mapping.Properties))
	for key := range mapping.Properties {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if len(propertyIDs) > 0 {
			if _, ok := propertyIDs[key]; !ok {
				continue
			}
		}
		out[key] = mapping.Properties[key].Value
	}
	return out
}

func stringSet(values []string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			out[value] = struct{}{}
		}
	}
	return out
}
