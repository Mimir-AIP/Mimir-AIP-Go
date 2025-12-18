package ml

import (
	"testing"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/Ontology"
)

func TestIsNumericRange(t *testing.T) {
	tests := []struct {
		name     string
		ranges   []string
		expected bool
	}{
		{"decimal", []string{"http://www.w3.org/2001/XMLSchema#decimal"}, true},
		{"integer", []string{"http://www.w3.org/2001/XMLSchema#integer"}, true},
		{"float", []string{"http://www.w3.org/2001/XMLSchema#float"}, true},
		{"double", []string{"http://www.w3.org/2001/XMLSchema#double"}, true},
		{"string", []string{"http://www.w3.org/2001/XMLSchema#string"}, false},
		{"boolean", []string{"http://www.w3.org/2001/XMLSchema#boolean"}, false},
		{"empty", []string{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isNumericRange(tt.ranges)
			if result != tt.expected {
				t.Errorf("isNumericRange(%v) = %v, want %v", tt.ranges, result, tt.expected)
			}
		})
	}
}

func TestIsCategoricalRange(t *testing.T) {
	tests := []struct {
		name     string
		ranges   []string
		expected bool
	}{
		{"string", []string{"http://www.w3.org/2001/XMLSchema#string"}, true},
		{"boolean", []string{"http://www.w3.org/2001/XMLSchema#boolean"}, true},
		{"uri", []string{"http://example.org/Category"}, true},
		{"decimal", []string{"http://www.w3.org/2001/XMLSchema#decimal"}, false},
		{"empty", []string{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isCategoricalRange(tt.ranges)
			if result != tt.expected {
				t.Errorf("isCategoricalRange(%v) = %v, want %v", tt.ranges, result, tt.expected)
			}
		})
	}
}

func TestIsCommonTargetLabel(t *testing.T) {
	tests := []struct {
		label    string
		expected bool
	}{
		{"price", true},
		{"cost", true},
		{"revenue", true},
		{"category", true},
		{"total_price", true},
		{"sale_amount", true},
		{"random_property", false},
		{"description", false},
	}

	for _, tt := range tests {
		t.Run(tt.label, func(t *testing.T) {
			result := isCommonTargetLabel(tt.label)
			if result != tt.expected {
				t.Errorf("isCommonTargetLabel(%s) = %v, want %v", tt.label, result, tt.expected)
			}
		})
	}
}

func TestHasOverlappingDomains(t *testing.T) {
	tests := []struct {
		name     string
		domains1 []string
		domains2 []string
		expected bool
	}{
		{
			"same domain",
			[]string{"http://example.org/Product"},
			[]string{"http://example.org/Product"},
			true,
		},
		{
			"different domains",
			[]string{"http://example.org/Product"},
			[]string{"http://example.org/Customer"},
			false,
		},
		{
			"empty domains - assume overlap",
			[]string{},
			[]string{"http://example.org/Product"},
			true,
		},
		{
			"multiple domains with overlap",
			[]string{"http://example.org/Product", "http://example.org/Item"},
			[]string{"http://example.org/Item", "http://example.org/Thing"},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasOverlappingDomains(tt.domains1, tt.domains2)
			if result != tt.expected {
				t.Errorf("hasOverlappingDomains(%v, %v) = %v, want %v",
					tt.domains1, tt.domains2, result, tt.expected)
			}
		})
	}
}

func TestFindSuggestedFeatures(t *testing.T) {
	oa := &OntologyAnalyzer{}

	targetProp := ontology.OntologyProperty{
		URI:          "http://example.org/price",
		Label:        "price",
		PropertyType: ontology.PropertyTypeDatatype,
		Domain:       []string{"http://example.org/Product"},
		Range:        []string{"http://www.w3.org/2001/XMLSchema#decimal"},
	}

	allProperties := []ontology.OntologyProperty{
		targetProp,
		{
			URI:          "http://example.org/category",
			Label:        "category",
			PropertyType: ontology.PropertyTypeDatatype,
			Domain:       []string{"http://example.org/Product"},
			Range:        []string{"http://www.w3.org/2001/XMLSchema#string"},
		},
		{
			URI:          "http://example.org/stock_level",
			Label:        "stock_level",
			PropertyType: ontology.PropertyTypeDatatype,
			Domain:       []string{"http://example.org/Product"},
			Range:        []string{"http://www.w3.org/2001/XMLSchema#integer"},
		},
		{
			URI:          "http://example.org/customer_name",
			Label:        "customer_name",
			PropertyType: ontology.PropertyTypeDatatype,
			Domain:       []string{"http://example.org/Customer"},
			Range:        []string{"http://www.w3.org/2001/XMLSchema#string"},
		},
	}

	features := oa.findSuggestedFeatures(targetProp, allProperties)

	// Should find category and stock_level (same domain as price)
	// Should NOT find price itself or customer_name (different domain)
	if len(features) != 2 {
		t.Errorf("Expected 2 features, got %d: %v", len(features), features)
	}

	foundCategory := false
	foundStock := false
	for _, f := range features {
		if f == "category" {
			foundCategory = true
		}
		if f == "stock_level" {
			foundStock = true
		}
	}

	if !foundCategory {
		t.Error("Expected to find 'category' in suggested features")
	}
	if !foundStock {
		t.Error("Expected to find 'stock_level' in suggested features")
	}
}

func TestCalculateConfidence(t *testing.T) {
	oa := &OntologyAnalyzer{}

	tests := []struct {
		name         string
		prop         ontology.OntologyProperty
		featureCount int
		dataCount    int
		minExpected  float64
		maxExpected  float64
	}{
		{
			"low confidence - few features, little data",
			ontology.OntologyProperty{Label: "random_prop"},
			1,
			10,
			0.4,
			0.6,
		},
		{
			"high confidence - many features, lots of data, common label",
			ontology.OntologyProperty{Label: "price"},
			5,
			1000,
			0.8,
			0.96,
		},
		{
			"medium confidence - good data, few features",
			ontology.OntologyProperty{Label: "total"},
			2,
			200,
			0.6,
			0.8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			confidence := oa.calculateConfidence(tt.prop, tt.featureCount, tt.dataCount)
			if confidence < tt.minExpected || confidence > tt.maxExpected {
				t.Errorf("calculateConfidence() = %f, want between %f and %f",
					confidence, tt.minExpected, tt.maxExpected)
			}
		})
	}
}

func TestGenerateSummary(t *testing.T) {
	oa := &OntologyAnalyzer{}

	tests := []struct {
		name         string
		capabilities *MLCapabilities
		contains     string
	}{
		{
			"regression only",
			&MLCapabilities{
				RegressionTargets: []MLTarget{
					{PropertyLabel: "price"},
					{PropertyLabel: "cost"},
				},
				ClassificationTargets: []MLTarget{},
				TimeSeriesMetrics:     []TimeSeriesMetric{},
			},
			"predict price, cost",
		},
		{
			"classification only",
			&MLCapabilities{
				RegressionTargets: []MLTarget{},
				ClassificationTargets: []MLTarget{
					{PropertyLabel: "category"},
				},
				TimeSeriesMetrics: []TimeSeriesMetric{},
			},
			"classify category",
		},
		{
			"monitoring only",
			&MLCapabilities{
				RegressionTargets:     []MLTarget{},
				ClassificationTargets: []MLTarget{},
				TimeSeriesMetrics: []TimeSeriesMetric{
					{PropertyLabel: "stock_level"},
					{PropertyLabel: "price"},
				},
			},
			"monitor 2 metrics",
		},
		{
			"empty capabilities",
			&MLCapabilities{
				RegressionTargets:     []MLTarget{},
				ClassificationTargets: []MLTarget{},
				TimeSeriesMetrics:     []TimeSeriesMetric{},
			},
			"No ML capabilities",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summary := oa.generateSummary(tt.capabilities)
			if !contains(summary, tt.contains) {
				t.Errorf("Summary '%s' does not contain '%s'", summary, tt.contains)
			}
		})
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
