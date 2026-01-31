package ontology

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEntityExtractionEffectiveness tests how well the extractor identifies entities
func TestEntityExtractionEffectiveness(t *testing.T) {
	t.Run("Extraction accuracy - well structured data", func(t *testing.T) {
		csvData := "id,name,category,price\nPROD-001,Widget,Product,19.99\nPROD-002,Gadget,Product,29.99\nPROD-003,Gizmo,Service,49.99"

		extractor := NewDeterministicExtractor(ExtractionConfig{
			SourceType: SourceTypeCSV,
		})

		ontology := &OntologyContext{
			BaseURI: "http://test.org/",
			Classes: []RDFClass{
				{URI: "http://test.org/Product", Label: "Product"},
				{URI: "http://test.org/Service", Label: "Service"},
			},
			Properties: []RDFProperty{
				{URI: "http://test.org/name", Label: "name"},
				{URI: "http://test.org/category", Label: "category"},
				{URI: "http://test.org/price", Label: "price"},
			},
		}

		result, err := extractor.Extract(csvData, ontology)
		require.NoError(t, err)
		require.NotNil(t, result)

		assert.Len(t, result.Entities, 3, "Should extract 3 entities from CSV")
		assert.GreaterOrEqual(t, len(result.Triples), 9, "Should have at least 9 triples")

		t.Logf("Extraction Results: Entities=%d, Triples=%d, Confidence=%.2f, Warnings=%d",
			len(result.Entities), len(result.Triples), result.Confidence, len(result.Warnings))
	})

	t.Run("Extraction completeness", func(t *testing.T) {
		csvData := `id,name,type,department,salary,active
EMP-001,Alice,Engineer,IT,80000,true
EMP-002,Bob,Manager,HR,90000,true
EMP-003,Charlie,Analyst,Finance,70000,false
EMP-004,Diana,Director,IT,120000,true`

		extractor := NewDeterministicExtractor(ExtractionConfig{
			SourceType: SourceTypeCSV,
		})

		ontology := &OntologyContext{
			BaseURI: "http://company.org/",
			Classes: []RDFClass{
				{URI: "http://company.org/Employee", Label: "Employee"},
			},
			Properties: []RDFProperty{
				{URI: "http://company.org/name", Label: "name"},
				{URI: "http://company.org/type", Label: "type"},
				{URI: "http://company.org/department", Label: "department"},
				{URI: "http://company.org/salary", Label: "salary"},
				{URI: "http://company.org/active", Label: "active"},
			},
		}

		result, err := extractor.Extract(csvData, ontology)
		require.NoError(t, err)

		assert.Len(t, result.Entities, 4, "Should extract all 4 employees")
		t.Logf("Extraction completeness: %d triples created", len(result.Triples))
	})
}
