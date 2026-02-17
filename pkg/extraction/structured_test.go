package extraction

import (
	"testing"
	"time"

	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

func TestExtractFromStructuredCIR(t *testing.T) {
	// Test case 1: Extract entities from user data
	t.Run("Extract user entities with relationships", func(t *testing.T) {
		cir := &models.CIR{
			Version: "1.0",
			Source: models.CIRSource{
				Type:      models.SourceTypeFile,
				URI:       "test.csv",
				Timestamp: time.Now(),
				Format:    models.DataFormatCSV,
			},
			Data: []interface{}{
				map[string]interface{}{
					"name":       "Alice",
					"email":      "alice@example.com",
					"department": "Engineering",
					"manager":    "Bob",
				},
				map[string]interface{}{
					"name":       "Bob",
					"email":      "bob@example.com",
					"department": "Engineering",
				},
				map[string]interface{}{
					"name":       "Charlie",
					"email":      "charlie@example.com",
					"department": "Sales",
					"manager":    "Bob",
				},
			},
		}

		result, err := ExtractFromStructuredCIR(cir)
		if err != nil {
			t.Fatalf("ExtractFromStructuredCIR failed: %v", err)
		}

		// Verify entities
		if len(result.Entities) != 3 {
			t.Errorf("Expected 3 entities, got %d", len(result.Entities))
		}

		// Verify entity attributes
		alice := findEntityInResult(result.Entities, "alice")
		if alice == nil {
			t.Fatal("Alice entity not found")
		}

		if alice.Attributes["email"] != "alice@example.com" {
			t.Errorf("Alice email incorrect: %v", alice.Attributes["email"])
		}

		if alice.Attributes["department"] != "Engineering" {
			t.Errorf("Alice department incorrect: %v", alice.Attributes["department"])
		}

		// Verify relationships (Alice reports to Bob)
		aliceReportsRelCount := 0
		for _, rel := range result.Relationships {
			if rel.Entity1.Name == "alice" && rel.Entity2.Name == "bob" && rel.Relation == "reports_to" {
				aliceReportsRelCount++
			}
		}

		if aliceReportsRelCount != 1 {
			t.Errorf("Expected 1 reports_to relationship from Alice to Bob, got %d", aliceReportsRelCount)
		}

		// Verify confidence
		if alice.Confidence != 0.9 {
			t.Errorf("Expected confidence 0.9 for structured data, got %f", alice.Confidence)
		}

		// Verify source
		if alice.Source != "structured" {
			t.Errorf("Expected source 'structured', got %s", alice.Source)
		}
	})

	// Test case 2: Invalid CIR data
	t.Run("Handle invalid CIR data", func(t *testing.T) {
		cir := &models.CIR{
			Version: "1.0",
			Source: models.CIRSource{
				Type:      models.SourceTypeFile,
				URI:       "test.csv",
				Timestamp: time.Now(),
				Format:    models.DataFormatCSV,
			},
			Data: "not an array", // Invalid data type
		}

		_, err := ExtractFromStructuredCIR(cir)
		if err == nil {
			t.Error("Expected error for invalid CIR data, got nil")
		}
	})

	// Test case 3: Empty data
	t.Run("Handle empty data", func(t *testing.T) {
		cir := &models.CIR{
			Version: "1.0",
			Source: models.CIRSource{
				Type:      models.SourceTypeFile,
				URI:       "test.csv",
				Timestamp: time.Now(),
				Format:    models.DataFormatCSV,
			},
			Data: []interface{}{},
		}

		result, err := ExtractFromStructuredCIR(cir)
		if err != nil {
			t.Fatalf("ExtractFromStructuredCIR failed: %v", err)
		}

		if len(result.Entities) != 0 {
			t.Errorf("Expected 0 entities for empty data, got %d", len(result.Entities))
		}
	})
}

func TestNormalizeText(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Alice", "alice"},
		{"  Bob  ", "bob"},
		{"Charlie   Smith", "charlie smith"},
		{"DAVID", "david"},
	}

	for _, test := range tests {
		result := normalizeText(test.input)
		if result != test.expected {
			t.Errorf("normalizeText(%q) = %q, expected %q", test.input, result, test.expected)
		}
	}
}

func TestFindEntityByName(t *testing.T) {
	entities := []models.ExtractedEntity{
		{Name: "alice", Confidence: 0.9},
		{Name: "bob", Confidence: 0.9},
		{Name: "charlie", Confidence: 0.9},
	}

	// Test finding existing entity
	entity := findEntityByName(entities, "bob")
	if entity == nil {
		t.Error("Expected to find Bob, got nil")
	} else if entity.Name != "bob" {
		t.Errorf("Expected entity name 'bob', got %s", entity.Name)
	}

	// Test finding non-existent entity
	entity = findEntityByName(entities, "david")
	if entity != nil {
		t.Error("Expected nil for non-existent entity, got entity")
	}
}

// Helper function to find entity by name in result
func findEntityInResult(entities []models.ExtractedEntity, name string) *models.ExtractedEntity {
	for i := range entities {
		if entities[i].Name == name {
			return &entities[i]
		}
	}
	return nil
}
