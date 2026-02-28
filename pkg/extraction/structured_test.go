package extraction

import (
	"strings"
	"testing"
	"time"

	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

// ─── Helpers ──────────────────────────────────────────────────────────────────

// findEntityInResult returns the first entity whose Name matches (case-sensitive).
func findEntityInResult(entities []models.ExtractedEntity, name string) *models.ExtractedEntity {
	for i := range entities {
		if entities[i].Name == name {
			return &entities[i]
		}
	}
	return nil
}

// entityNamesInResult returns all entity names as a slice.
func entityNamesInResult(entities []models.ExtractedEntity) []string {
	out := make([]string, len(entities))
	for i, e := range entities {
		out[i] = e.Name
	}
	return out
}

// hasRelationLabel returns true if any relationship has the given label.
func hasRelationLabel(rels []models.ExtractedRelationship, label string) bool {
	for _, r := range rels {
		if r.Relation == label {
			return true
		}
	}
	return false
}

// ─── Core extraction tests ────────────────────────────────────────────────────

// TestExtractFromStructuredCIR verifies schema-inductive extraction across
// several scenarios: error handling, categorical vs identifier classification,
// relationship labelling, and confidence scaling.
func TestExtractFromStructuredCIR(t *testing.T) {

	// ── invalid / empty inputs ──────────────────────────────────────────────

	t.Run("Handle invalid CIR data", func(t *testing.T) {
		cir := &models.CIR{
			Version: "1.0",
			Source:  models.CIRSource{Type: models.SourceTypeFile, URI: "test.csv", Format: models.DataFormatCSV, Timestamp: time.Now()},
			Data:    "not an array",
		}
		_, err := ExtractFromStructuredCIR(cir)
		if err == nil {
			t.Error("expected error for non-array CIR data")
		}
	})

	t.Run("Handle nil CIR", func(t *testing.T) {
		_, err := ExtractFromStructuredCIR(nil)
		if err == nil {
			t.Error("expected error for nil CIR")
		}
	})

	t.Run("Handle empty data", func(t *testing.T) {
		cir := &models.CIR{Data: []interface{}{}}
		result, err := ExtractFromStructuredCIR(cir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Entities) != 0 {
			t.Errorf("expected 0 entities for empty data; got %d", len(result.Entities))
		}
	})

	// ── entity extraction ───────────────────────────────────────────────────

	// Six-row HR dataset: "employee" is high-cardinality (identifier),
	// "department" is low-cardinality (categorical, 2/6 = 0.33 ≤ 0.35).
	t.Run("Identifier and categorical columns extracted as entities", func(t *testing.T) {
		cir := &models.CIR{
			Data: []interface{}{
				map[string]interface{}{"employee": "Alice", "department": "Engineering", "location": "London"},
				map[string]interface{}{"employee": "Bob", "department": "Engineering", "location": "London"},
				map[string]interface{}{"employee": "Carol", "department": "Engineering", "location": "London"},
				map[string]interface{}{"employee": "Diana", "department": "Sales", "location": "Madrid"},
				map[string]interface{}{"employee": "Eve", "department": "Sales", "location": "Madrid"},
				map[string]interface{}{"employee": "Frank", "department": "Sales", "location": "Madrid"},
			},
		}

		result, err := ExtractFromStructuredCIR(cir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		names := entityNamesInResult(result.Entities)

		// Individual employees from the identifier column.
		for _, emp := range []string{"Alice", "Bob", "Carol", "Diana", "Eve", "Frank"} {
			if findEntityInResult(result.Entities, emp) == nil {
				t.Errorf("expected employee entity %q; got %v", emp, names)
			}
		}
		// Category entities from the low-cardinality column.
		for _, dept := range []string{"Engineering", "Sales"} {
			if findEntityInResult(result.Entities, dept) == nil {
				t.Errorf("expected department entity %q; got %v", dept, names)
			}
		}
	})

	t.Run("Entity type attribute reflects column name in PascalCase", func(t *testing.T) {
		cir := &models.CIR{
			Data: []interface{}{
				map[string]interface{}{"deal_type": "Acquisition"},
				map[string]interface{}{"deal_type": "Merger"},
			},
		}
		result, err := ExtractFromStructuredCIR(cir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		e := findEntityInResult(result.Entities, "Acquisition")
		if e == nil {
			t.Fatal("expected Acquisition entity")
		}
		if e.Attributes["entity_type"] != "DealType" {
			t.Errorf("expected entity_type=DealType; got %v", e.Attributes["entity_type"])
		}
	})

	t.Run("Numeric and boolean columns are not extracted as entities", func(t *testing.T) {
		cir := &models.CIR{
			Data: []interface{}{
				map[string]interface{}{"product": "Widget", "price": 9.99, "in_stock": true, "qty": 100},
				map[string]interface{}{"product": "Gadget", "price": 24.99, "in_stock": false, "qty": 5},
				map[string]interface{}{"product": "Doohickey", "price": 4.99, "in_stock": true, "qty": 200},
			},
		}
		result, err := ExtractFromStructuredCIR(cir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Only "product" values should be entities; price/qty/in_stock are attributes.
		for _, e := range result.Entities {
			switch e.Name {
			case "9.99", "24.99", "4.99", "100", "5", "200", "true", "false":
				t.Errorf("numeric/boolean value %q should not be extracted as entity", e.Name)
			}
		}
		if findEntityInResult(result.Entities, "Widget") == nil {
			t.Errorf("expected Widget entity; got %v", entityNamesInResult(result.Entities))
		}
	})

	// ── confidence scaling ──────────────────────────────────────────────────

	t.Run("Categorical entity confidence scales with frequency", func(t *testing.T) {
		// status appears in 5/6 rows as "Active" and 1/6 as "Inactive".
		// "Active" should have higher confidence than "Inactive".
		cir := &models.CIR{
			Data: []interface{}{
				map[string]interface{}{"name": "A", "status": "Active"},
				map[string]interface{}{"name": "B", "status": "Active"},
				map[string]interface{}{"name": "C", "status": "Active"},
				map[string]interface{}{"name": "D", "status": "Active"},
				map[string]interface{}{"name": "E", "status": "Active"},
				map[string]interface{}{"name": "F", "status": "Inactive"},
			},
		}
		result, err := ExtractFromStructuredCIR(cir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		active := findEntityInResult(result.Entities, "Active")
		inactive := findEntityInResult(result.Entities, "Inactive")
		if active == nil || inactive == nil {
			t.Fatalf("expected both Active and Inactive entities; got %v", entityNamesInResult(result.Entities))
		}
		if active.Confidence <= inactive.Confidence {
			t.Errorf("Active (%.2f) should have higher confidence than Inactive (%.2f)", active.Confidence, inactive.Confidence)
		}
	})

	t.Run("Identifier column entities have flat 0.85 confidence", func(t *testing.T) {
		cir := &models.CIR{
			Data: []interface{}{
				map[string]interface{}{"employee": "Alice"},
				map[string]interface{}{"employee": "Bob"},
				map[string]interface{}{"employee": "Carol"},
			},
		}
		result, err := ExtractFromStructuredCIR(cir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		for _, e := range result.Entities {
			if e.Confidence != 0.85 {
				t.Errorf("identifier entity %q: expected confidence 0.85, got %.2f", e.Name, e.Confidence)
			}
		}
	})

	// ── relationship extraction ─────────────────────────────────────────────

	t.Run("Relationships use column-name-derived labels, not hardcoded predicates", func(t *testing.T) {
		// With columns "acquirer" and "sector", any relationship must be
		// labelled "acquirer_to_sector" — never "belongs_to" or "reports_to".
		cir := &models.CIR{
			Data: []interface{}{
				map[string]interface{}{"acquirer": "Apex Capital", "sector": "Technology"},
				map[string]interface{}{"acquirer": "Apex Capital", "sector": "Technology"},
				map[string]interface{}{"acquirer": "Meridian Partners", "sector": "Energy"},
				map[string]interface{}{"acquirer": "Meridian Partners", "sector": "Energy"},
				map[string]interface{}{"acquirer": "Apex Capital", "sector": "Technology"},
				map[string]interface{}{"acquirer": "Meridian Partners", "sector": "Energy"},
			},
		}
		result, err := ExtractFromStructuredCIR(cir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Relationships) == 0 {
			t.Fatal("expected relationships between co-occurring column values")
		}
		// Check that NO hardcoded label slipped through.
		for _, r := range result.Relationships {
			if r.Relation == "reports_to" || r.Relation == "belongs_to" ||
				r.Relation == "located_in" || r.Relation == "member_of" {
				t.Errorf("hardcoded relationship label %q found; expected column-derived label", r.Relation)
			}
		}
		// Relationship label must be derived from the column names.
		expectedLabel := "acquirer_to_sector"
		if !hasRelationLabel(result.Relationships, expectedLabel) {
			labels := make([]string, len(result.Relationships))
			for i, r := range result.Relationships {
				labels[i] = r.Relation
			}
			t.Errorf("expected label %q; got %v", expectedLabel, labels)
		}
	})

	t.Run("Tight co-occurrence yields high relationship confidence", func(t *testing.T) {
		// "Engineering" and "London" always appear together — should give
		// near-maximum confidence.  "Sales" and "Madrid" also always together.
		cir := &models.CIR{
			Data: []interface{}{
				map[string]interface{}{"dept": "Engineering", "city": "London"},
				map[string]interface{}{"dept": "Engineering", "city": "London"},
				map[string]interface{}{"dept": "Engineering", "city": "London"},
				map[string]interface{}{"dept": "Sales", "city": "Madrid"},
				map[string]interface{}{"dept": "Sales", "city": "Madrid"},
				map[string]interface{}{"dept": "Sales", "city": "Madrid"},
			},
		}
		result, err := ExtractFromStructuredCIR(cir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		for _, r := range result.Relationships {
			if r.Confidence < 0.85 {
				t.Errorf("tight co-occurrence relationship %q→%q: expected confidence ≥0.85, got %.2f",
					r.Entity1.Name, r.Entity2.Name, r.Confidence)
			}
		}
	})

	t.Run("No relationships emitted when only one entity-bearing column", func(t *testing.T) {
		// Only "status" carries entities; no pairs → no relationships.
		cir := &models.CIR{
			Data: []interface{}{
				map[string]interface{}{"status": "Active", "count": 10},
				map[string]interface{}{"status": "Inactive", "count": 3},
			},
		}
		result, err := ExtractFromStructuredCIR(cir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Relationships) != 0 {
			t.Errorf("expected 0 relationships with one entity column; got %d", len(result.Relationships))
		}
	})

	// ── source metadata ─────────────────────────────────────────────────────

	t.Run("Source field is set correctly", func(t *testing.T) {
		cir := &models.CIR{
			Data: []interface{}{
				map[string]interface{}{"name": "Alice"},
			},
		}
		result, err := ExtractFromStructuredCIR(cir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Source != "structured" {
			t.Errorf("expected source=structured; got %q", result.Source)
		}
		for _, e := range result.Entities {
			if e.Source != "structured" {
				t.Errorf("entity %q: expected source=structured; got %q", e.Name, e.Source)
			}
		}
	})
}

// ─── Column classification unit tests ────────────────────────────────────────

func TestColumnClassification(t *testing.T) {
	t.Run("Low cardinality column classified as categorical", func(t *testing.T) {
		c := &colInfo{
			name:        "department",
			valueCounts: map[string]int{"Engineering": 8, "Sales": 4},
			totalRows:   12,
			stringRows:  12,
		}
		classifyColumn(c)
		if c.role != roleCategorical {
			t.Errorf("expected roleCategorical (cardinality=%.2f); got %d", c.cardinalityRatio(), c.role)
		}
	})

	t.Run("High cardinality column classified as identifier", func(t *testing.T) {
		counts := map[string]int{}
		for i := 0; i < 20; i++ {
			counts[strings.Repeat("a", i+1)] = 1
		}
		c := &colInfo{
			name:        "employee",
			valueCounts: counts,
			totalRows:   20,
			stringRows:  20,
		}
		classifyColumn(c)
		if c.role != roleIdentifier {
			t.Errorf("expected roleIdentifier (cardinality=%.2f); got %d", c.cardinalityRatio(), c.role)
		}
	})

	t.Run("Column with no string values classified as attribute", func(t *testing.T) {
		c := &colInfo{name: "price", valueCounts: map[string]int{}, totalRows: 5, stringRows: 0}
		classifyColumn(c)
		if c.role != roleAttribute {
			t.Errorf("expected roleAttribute; got %d", c.role)
		}
	})
}

// ─── colNameToType unit tests ─────────────────────────────────────────────────

func TestColNameToType(t *testing.T) {
	cases := []struct{ in, want string }{
		{"department", "Department"},
		{"employee_name", "EmployeeName"},
		{"deal-type", "DealType"},
		{"source column", "SourceColumn"},
		{"status", "Status"},
	}
	for _, tc := range cases {
		got := colNameToType(tc.in)
		if got != tc.want {
			t.Errorf("colNameToType(%q) = %q; want %q", tc.in, got, tc.want)
		}
	}
}

// ─── normalizeText unit test (retained utility) ───────────────────────────────

func TestNormalizeText(t *testing.T) {
	cases := []struct{ input, want string }{
		{"Alice", "alice"},
		{"  Bob  ", "bob"},
		{"Charlie   Smith", "charlie smith"},
		{"DAVID", "david"},
	}
	for _, tc := range cases {
		if got := normalizeText(tc.input); got != tc.want {
			t.Errorf("normalizeText(%q) = %q; want %q", tc.input, got, tc.want)
		}
	}
}
