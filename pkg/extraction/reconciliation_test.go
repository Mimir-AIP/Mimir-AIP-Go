package extraction

import (
	"testing"

	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

func TestReconcileEntities(t *testing.T) {
	// Test case 1: Reconcile structured and unstructured entities
	t.Run("Reconcile duplicate entities from different sources", func(t *testing.T) {
		// Structured data entities
		structuredResults := &models.ExtractionResult{
			Entities: []models.ExtractedEntity{
				{
					Name:       "alice",
					Attributes: map[string]interface{}{"email": "alice@example.com"},
					Source:     "structured",
					Confidence: 0.9,
				},
				{
					Name:       "bob",
					Attributes: map[string]interface{}{"email": "bob@example.com"},
					Source:     "structured",
					Confidence: 0.9,
				},
			},
			Relationships: []models.ExtractedRelationship{},
			Source:        "structured",
		}

		// Unstructured data entities (duplicate Alice with different attributes)
		unstructuredResults := &models.ExtractionResult{
			Entities: []models.ExtractedEntity{
				{
					Name:       "alice",
					Attributes: map[string]interface{}{"title": "Senior Engineer"},
					Source:     "unstructured",
					Confidence: 0.7,
				},
				{
					Name:       "charlie",
					Attributes: map[string]interface{}{"department": "Sales"},
					Source:     "unstructured",
					Confidence: 0.8,
				},
			},
			Relationships: []models.ExtractedRelationship{},
			Source:        "unstructured",
		}

		result := ReconcileEntities(structuredResults, unstructuredResults)

		// Should have 3 unique entities (alice, bob, charlie)
		if len(result.Entities) != 3 {
			t.Errorf("Expected 3 reconciled entities, got %d", len(result.Entities))
		}

		// Find reconciled Alice
		var reconciledAlice *models.ExtractedEntity
		for i := range result.Entities {
			if result.Entities[i].Name == "alice" {
				reconciledAlice = &result.Entities[i]
				break
			}
		}

		if reconciledAlice == nil {
			t.Fatal("Reconciled Alice entity not found")
		}

		// Verify Alice has attributes from both sources
		if reconciledAlice.Attributes["email"] != "alice@example.com" {
			t.Error("Alice should have email from structured source")
		}

		if reconciledAlice.Attributes["title"] != "Senior Engineer" {
			t.Error("Alice should have title from unstructured source")
		}

		// Verify confidence is averaged
		expectedConfidence := (0.9 + 0.7) / 2.0
		if reconciledAlice.Confidence != expectedConfidence {
			t.Errorf("Expected confidence %f, got %f", expectedConfidence, reconciledAlice.Confidence)
		}

		// Verify sources are tracked
		if len(reconciledAlice.Sources) < 2 {
			t.Errorf("Expected at least 2 sources, got %d", len(reconciledAlice.Sources))
		}
	})

	// Test case 2: Deduplication of relationships
	t.Run("Deduplicate relationships", func(t *testing.T) {
		alice := models.ExtractedEntity{Name: "alice", Confidence: 0.9, Source: "structured"}
		bob := models.ExtractedEntity{Name: "bob", Confidence: 0.9, Source: "structured"}

		structuredResults := &models.ExtractionResult{
			Entities: []models.ExtractedEntity{alice, bob},
			Relationships: []models.ExtractedRelationship{
				{Entity1: &alice, Entity2: &bob, Relation: "reports_to", Confidence: 0.85},
			},
			Source: "structured",
		}

		// Duplicate relationship from unstructured source
		aliceDup := models.ExtractedEntity{Name: "alice", Confidence: 0.7, Source: "unstructured"}
		bobDup := models.ExtractedEntity{Name: "bob", Confidence: 0.7, Source: "unstructured"}

		unstructuredResults := &models.ExtractionResult{
			Entities: []models.ExtractedEntity{aliceDup, bobDup},
			Relationships: []models.ExtractedRelationship{
				{Entity1: &aliceDup, Entity2: &bobDup, Relation: "reports_to", Confidence: 0.75},
			},
			Source: "unstructured",
		}

		result := ReconcileEntities(structuredResults, unstructuredResults)

		// Should have only 1 reports_to relationship (deduplicated)
		if len(result.Relationships) != 1 {
			t.Errorf("Expected 1 deduplicated relationship, got %d", len(result.Relationships))
		}
	})

	// Test case 3: Handle nil results
	t.Run("Handle nil structured results", func(t *testing.T) {
		unstructuredResults := &models.ExtractionResult{
			Entities: []models.ExtractedEntity{
				{Name: "alice", Confidence: 0.8, Source: "unstructured"},
			},
			Relationships: []models.ExtractedRelationship{},
			Source:        "unstructured",
		}

		result := ReconcileEntities(nil, unstructuredResults)

		if len(result.Entities) != 1 {
			t.Errorf("Expected 1 entity, got %d", len(result.Entities))
		}
	})

	t.Run("Handle nil unstructured results", func(t *testing.T) {
		structuredResults := &models.ExtractionResult{
			Entities: []models.ExtractedEntity{
				{Name: "alice", Confidence: 0.9, Source: "structured"},
			},
			Relationships: []models.ExtractedRelationship{},
			Source:        "structured",
		}

		result := ReconcileEntities(structuredResults, nil)

		if len(result.Entities) != 1 {
			t.Errorf("Expected 1 entity, got %d", len(result.Entities))
		}
	})
}

func TestNormalizeEntityName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Alice Corp", "alice corporation"},
		{"The Company Inc", "company incorporated"},
		{"Bob & Associates", "bob and associates"},
		{"  Extra  Spaces  ", "extra spaces"},
		{"Company, LLC.", "company llc"},
	}

	for _, test := range tests {
		result := normalizeEntityName(test.input)
		if result != test.expected {
			t.Errorf("normalizeEntityName(%q) = %q, expected %q", test.input, result, test.expected)
		}
	}
}

func TestMergeEntityGroup(t *testing.T) {
	// Test merging entities with different confidences and sources
	t.Run("Prefer higher confidence entity", func(t *testing.T) {
		group := []models.ExtractedEntity{
			{
				Name:       "alice",
				Attributes: map[string]interface{}{"email": "alice@example.com"},
				Source:     "structured",
				Confidence: 0.9,
			},
			{
				Name:       "alice",
				Attributes: map[string]interface{}{"title": "Engineer"},
				Source:     "unstructured",
				Confidence: 0.7,
			},
		}

		merged := mergeEntityGroup(group)

		// Should use name from highest confidence entity (structured)
		if merged.Name != "alice" {
			t.Errorf("Expected name 'alice', got %s", merged.Name)
		}

		// Should have attributes from both
		if merged.Attributes["email"] != "alice@example.com" {
			t.Error("Missing email attribute")
		}

		if merged.Attributes["title"] != "Engineer" {
			t.Error("Missing title attribute")
		}

		// Should have averaged confidence
		expectedConfidence := (0.9 + 0.7) / 2.0
		if merged.Confidence != expectedConfidence {
			t.Errorf("Expected confidence %f, got %f", expectedConfidence, merged.Confidence)
		}
	})

	// Test single entity group
	t.Run("Single entity group returns same entity", func(t *testing.T) {
		group := []models.ExtractedEntity{
			{
				Name:       "bob",
				Attributes: map[string]interface{}{"email": "bob@example.com"},
				Source:     "structured",
				Confidence: 0.9,
			},
		}

		merged := mergeEntityGroup(group)

		if merged.Name != "bob" {
			t.Errorf("Expected name 'bob', got %s", merged.Name)
		}

		if merged.Confidence != 0.9 {
			t.Errorf("Expected confidence 0.9, got %f", merged.Confidence)
		}
	})
}
