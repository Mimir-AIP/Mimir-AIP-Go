package extraction

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

// categoricalThreshold is the maximum cardinality ratio (unique / total rows)
// for a column to be classified as categorical.  A column where fewer than
// this fraction of its values are unique holds a small set of shared labels
// (e.g. department, country, status) rather than individual instance names.
const categoricalThreshold = 0.35

// columnRole describes how a column participates in entity extraction.
type columnRole int

const (
	// roleIdentifier: high-cardinality string column — each distinct value is
	// its own entity instance (e.g. employee name, product SKU, case number).
	roleIdentifier columnRole = iota

	// roleCategorical: low-cardinality string column — the column defines a
	// shared classification; each unique value is a category entity
	// (e.g. department, country, deal type, verdict).
	roleCategorical

	// roleAttribute: numeric / boolean column — values are scalar properties,
	// not entities (e.g. price, quantity, is_active).
	roleAttribute
)

// colInfo holds per-column statistics derived from the data itself.
type colInfo struct {
	name        string
	role        columnRole
	valueCounts map[string]int // string value → occurrence count across all rows
	totalRows   int            // rows where this column was present
	stringRows  int            // rows with a non-empty string value
}

func (c *colInfo) uniqueCount() int { return len(c.valueCounts) }

func (c *colInfo) cardinalityRatio() float64 {
	if c.totalRows == 0 {
		return 0
	}
	return float64(c.uniqueCount()) / float64(c.totalRows)
}

// ExtractFromStructuredCIR extracts entities and relationships from a CIR
// whose Data is a []interface{} (tabular data — CSV, database result sets,
// JSON arrays of objects).
//
// The algorithm is schema-inductive: it classifies each column by its
// statistical properties (cardinality, value types) and derives entity types
// and relationship predicates entirely from the column names in the data.
// No hardcoded field names or relationship patterns are used.
//
// Column classification:
//   - Numeric / boolean columns → scalar attributes, no entity extraction.
//   - Low-cardinality string columns (≤ categoricalThreshold unique ratio) →
//     categorical entities: each unique value is one entity whose type is the
//     column name (e.g. "Engineering" of type Department).
//   - High-cardinality string columns → identifier entities: each row's value
//     is its own entity instance (e.g. "Alice Johnson" of type EmployeeName).
//
// Relationship extraction: for every pair of entity-bearing columns that
// co-occur in the same row, a relationship is emitted whose predicate is
// derived directly from the column names: colA_to_colB.  Confidence scales
// with how consistently that specific pair of values co-occurs across the
// dataset.
func ExtractFromStructuredCIR(cir *models.CIR) (*models.ExtractionResult, error) {
	if cir == nil {
		return nil, fmt.Errorf("CIR cannot be nil")
	}
	rows, ok := cir.Data.([]interface{})
	if !ok {
		return nil, fmt.Errorf("CIR data must be an array for structured extraction")
	}
	if len(rows) == 0 {
		return &models.ExtractionResult{Source: "structured"}, nil
	}

	// Phase 1 — collect column statistics from the data.
	cols := inferColumns(rows)
	if len(cols) == 0 {
		return &models.ExtractionResult{Source: "structured"}, nil
	}

	// Phase 2 — classify each column based on its statistical properties.
	for i := range cols {
		classifyColumn(&cols[i])
	}

	// Phase 3 — extract entity instances from entity-bearing columns.
	entities, entityIndex := extractStructuredEntities(cols, len(rows))

	// Phase 4 — extract relationships from cross-column co-occurrence.
	relationships := extractStructuredRelationships(cols, rows, entityIndex)

	return &models.ExtractionResult{
		Entities:      entities,
		Relationships: relationships,
		Source:        "structured",
	}, nil
}

// ─── Phase 1: Schema inference ────────────────────────────────────────────────

// inferColumns scans all rows and builds a colInfo for every column present.
// Column order is preserved (first-seen order from the row data).
func inferColumns(rows []interface{}) []colInfo {
	colMap := make(map[string]*colInfo)
	var colOrder []string

	for _, rawRow := range rows {
		row, ok := rawRow.(map[string]interface{})
		if !ok {
			continue
		}
		for key, val := range row {
			ci, seen := colMap[key]
			if !seen {
				ci = &colInfo{name: key, valueCounts: make(map[string]int)}
				colMap[key] = ci
				colOrder = append(colOrder, key)
			}
			ci.totalRows++
			if s := stringVal(val); s != "" {
				ci.valueCounts[s]++
				ci.stringRows++
			}
		}
	}

	result := make([]colInfo, 0, len(colOrder))
	for _, name := range colOrder {
		result = append(result, *colMap[name])
	}
	return result
}

// stringVal converts a value to its trimmed string form.
// Returns "" for booleans, pure numerics, and blank strings — those are
// treated as scalar attributes, not entity candidates.
func stringVal(v interface{}) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case bool:
		return ""
	case float64, float32, int, int64, int32:
		return ""
	case string:
		s := strings.TrimSpace(t)
		if s == "" {
			return ""
		}
		if _, err := strconv.ParseFloat(s, 64); err == nil {
			return "" // numeric string → attribute
		}
		return s
	default:
		s := strings.TrimSpace(fmt.Sprintf("%v", t))
		if _, err := strconv.ParseFloat(s, 64); err == nil {
			return ""
		}
		return s
	}
}

// ─── Phase 2: Column classification ──────────────────────────────────────────

// classifyColumn assigns a role to the column based on its value type and
// cardinality ratio (unique values / total rows).
func classifyColumn(c *colInfo) {
	if c.stringRows == 0 {
		c.role = roleAttribute
		return
	}
	if c.cardinalityRatio() <= categoricalThreshold {
		c.role = roleCategorical
	} else {
		c.role = roleIdentifier
	}
}

// ─── Phase 3: Entity extraction ───────────────────────────────────────────────

// entityKey uniquely addresses an entity by its source column and value.
type entityKey struct{ col, val string }

// extractStructuredEntities emits one entity per distinct (column, value) pair
// for entity-bearing columns (categorical or identifier).
//
// For categorical columns, confidence scales with occurrence frequency —
// a value that appears in many rows is a more reliable canonical entity.
// For identifier columns, confidence is a flat 0.85 (each value is a
// distinct instance; frequency is not meaningful).
//
// The slice is pre-allocated at full capacity so that the returned pointer
// index remains valid for the lifetime of the result.
func extractStructuredEntities(cols []colInfo, totalRows int) ([]models.ExtractedEntity, map[entityKey]*models.ExtractedEntity) {
	capacity := 0
	for _, c := range cols {
		if c.role != roleAttribute {
			capacity += c.uniqueCount()
		}
	}
	entities := make([]models.ExtractedEntity, 0, capacity)

	for _, col := range cols {
		if col.role == roleAttribute {
			continue
		}
		for val, count := range col.valueCounts {
			if val == "" {
				continue
			}
			var conf float64
			if col.role == roleCategorical {
				// More frequently recurring category → more reliable entity.
				conf = math.Min(0.55+0.35*float64(count)/float64(totalRows), 0.92)
			} else {
				conf = 0.85
			}
			entities = append(entities, models.ExtractedEntity{
				Name: val,
				Attributes: map[string]interface{}{
					"entity_type":      colNameToType(col.name),
					"source_column":    col.name,
					"occurrence_count": count,
				},
				Source:     "structured",
				Confidence: conf,
			})
		}
	}

	// Sort for determinism: highest confidence first, then alphabetical.
	sort.Slice(entities, func(i, j int) bool {
		if entities[i].Confidence != entities[j].Confidence {
			return entities[i].Confidence > entities[j].Confidence
		}
		return entities[i].Name < entities[j].Name
	})

	// Build pointer index.  The slice is fixed-size from this point on.
	index := make(map[entityKey]*models.ExtractedEntity, len(entities))
	for i := range entities {
		col, _ := entities[i].Attributes["source_column"].(string)
		index[entityKey{col, entities[i].Name}] = &entities[i]
	}
	return entities, index
}

// colNameToType converts a snake_case or kebab-case column name to a
// PascalCase entity type label suitable for use as an ontology class name.
// "employee_name" → "EmployeeName", "deal-type" → "DealType".
func colNameToType(col string) string {
	parts := strings.FieldsFunc(col, func(r rune) bool {
		return r == '_' || r == '-' || r == ' '
	})
	var b strings.Builder
	for _, p := range parts {
		runes := []rune(p)
		if len(runes) == 0 {
			continue
		}
		b.WriteRune(unicode.ToUpper(runes[0]))
		b.WriteString(string(runes[1:]))
	}
	return b.String()
}

// ─── Phase 4: Relationship extraction ────────────────────────────────────────

// relKey uniquely identifies an undirected co-occurrence pair.
// fromCol ≤ toCol lexicographically to enforce unordered semantics.
type relKey struct{ fromCol, fromVal, toCol, toVal string }

// extractStructuredRelationships emits a relationship for every unordered pair
// of entity-bearing columns that co-occur in the same row.
//
// The predicate is derived entirely from the column names in the data:
//
//	(colA, colB) → "colA_to_colB"   (colA ≤ colB alphabetically)
//
// Confidence reflects how consistently that specific pair of values appears
// together across the dataset, normalised by the rarer entity's occurrence
// count (a tight pair = high confidence).
func extractStructuredRelationships(cols []colInfo, rows []interface{}, entityIndex map[entityKey]*models.ExtractedEntity) []models.ExtractedRelationship {
	var entityCols []colInfo
	for _, c := range cols {
		if c.role != roleAttribute {
			entityCols = append(entityCols, c)
		}
	}
	if len(entityCols) < 2 {
		return nil
	}

	// Count co-occurrences for every unordered (col, val) pair.
	pairCounts := make(map[relKey]int)

	for _, rawRow := range rows {
		row, ok := rawRow.(map[string]interface{})
		if !ok {
			continue
		}

		// Collect which (col, val) pairs are present in this row.
		var present []entityKey
		for _, col := range entityCols {
			if s := stringVal(row[col.name]); s != "" {
				present = append(present, entityKey{col.name, s})
			}
		}

		// Emit all unordered pairs; canonical col order = lexicographic.
		for i := 0; i < len(present); i++ {
			for j := i + 1; j < len(present); j++ {
				a, b := present[i], present[j]
				if a.col > b.col {
					a, b = b, a
				}
				pairCounts[relKey{a.col, a.val, b.col, b.val}]++
			}
		}
	}

	rels := make([]models.ExtractedRelationship, 0, len(pairCounts))
	for rk, count := range pairCounts {
		e1 := entityIndex[entityKey{rk.fromCol, rk.fromVal}]
		e2 := entityIndex[entityKey{rk.toCol, rk.toVal}]
		if e1 == nil || e2 == nil {
			continue
		}

		// Predicate derived from column names — no hardcoded vocabulary.
		label := normaliseFieldKey(rk.fromCol) + "_to_" + normaliseFieldKey(rk.toCol)

		// Confidence: what fraction of the rarer entity's rows include the
		// other entity?  A high fraction means the two values reliably appear
		// together (tight relationship); a low fraction means incidental co-occurrence.
		e1Count, _ := e1.Attributes["occurrence_count"].(int)
		e2Count, _ := e2.Attributes["occurrence_count"].(int)
		minCount := e1Count
		if e2Count < minCount {
			minCount = e2Count
		}
		conf := 0.50
		if minCount > 0 {
			conf = math.Min(0.50+0.40*float64(count)/float64(minCount), 0.90)
		}

		rels = append(rels, models.ExtractedRelationship{
			Entity1:    e1,
			Entity2:    e2,
			Relation:   label,
			Confidence: conf,
		})
	}

	return rels
}

// ─── Tabular row-entity extraction ───────────────────────────────────────────
//
// For structured data that is clearly a database result set (an array of maps
// with a consistent schema and at least one key column), this path produces
// much more accurate ontologies than ExtractFromStructuredCIR:
//
//   Old path:  each column VALUE is an entity instance ("Alice" of type "FirstName")
//   This path: each ROW is an entity instance ("42" of type "Student"), and every
//              column value (including numerics) becomes an entity attribute.
//
// The entity type is inferred in priority order:
//  1. Explicit "entity_type" or "table_name" CIR parameter.
//  2. Primary key column suffix stripped to PascalCase ("student_id" → "Student").
//  3. CIR URI last path segment ("storage://x/attendance" → "Attendance").
//
// Returns (nil, false) when the data is not suitable (no key column, no type
// inference possible), so the caller can fall back to ExtractFromStructuredCIR.

// keyNameSuffixes are column name tokens that signal an identifier / primary key.
// Mirrors the list in crosssource.go so that tabular extraction and column
// profiling agree on what constitutes a key column.
var keyNameSuffixes = []string{
	"id", "key", "code", "number", "uuid", "ref",
	"identifier", "no", "num", "email", "username", "token",
}

// ExtractSchemaFromTabularCIR extracts one entity per row from a CIR whose
// data is a flat array of maps (database table result set, CSV rows, JSON
// array of records).
//
// The function returns (nil, false) when the CIR is not suitable for row-entity
// mode, in which case the caller should fall back to ExtractFromStructuredCIR.
func ExtractSchemaFromTabularCIR(cir *models.CIR) (*models.ExtractionResult, bool) {
	if cir == nil {
		return nil, false
	}
	rows, ok := cir.Data.([]interface{})
	if !ok || len(rows) == 0 {
		return nil, false
	}
	// Require first element to be an object row.
	if _, ok := rows[0].(map[string]interface{}); !ok {
		return nil, false
	}

	cols := inferColumns(rows)
	if len(cols) == 0 {
		return nil, false
	}

	entityType := detectTabularEntityType(cir, cols, rows)
	if entityType == "" || entityType == "Entity" {
		return nil, false
	}

	// Use cardinality to pick the true primary key (highest uniqueness ratio).
	primaryKey := bestKeyColumnByCardinality(findAllKeyColumns(cols), rows)

	seen := make(map[string]bool)
	var entities []models.ExtractedEntity

	for _, rawRow := range rows {
		row, ok := rawRow.(map[string]interface{})
		if !ok {
			continue
		}

		// Stable entity name from primary key value, or a row fingerprint.
		entityName := ""
		if primaryKey != "" {
			if v, ok := row[primaryKey]; ok {
				entityName = fmt.Sprintf("%v", v)
			}
		}
		if entityName == "" {
			entityName = rowFingerprint(row)
		}
		if seen[entityName] {
			continue
		}
		seen[entityName] = true

		// All columns become attributes; entity_type explicitly set.
		attrs := make(map[string]interface{}, len(row)+1)
		attrs["entity_type"] = entityType
		for k, v := range row {
			attrs[k] = v
		}

		entities = append(entities, models.ExtractedEntity{
			Name:       entityName,
			Attributes: attrs,
			Source:     "structured",
			Confidence: 0.90,
		})
	}

	if len(entities) == 0 {
		return nil, false
	}

	return &models.ExtractionResult{
		Entities: entities,
		Source:   "structured",
	}, true
}

// detectTabularEntityType determines the entity type for a flat record CIR.
// rows is required for cardinality-based primary key selection when multiple
// candidate key columns exist.
func detectTabularEntityType(cir *models.CIR, cols []colInfo, rows []interface{}) string {
	// 1. Explicit CIR parameter.
	for _, param := range []string{"entity_type", "table_name", "source_table"} {
		if v, ok := cir.GetParameter(param); ok {
			if s, ok := v.(string); ok && s != "" {
				return colNameToType(s)
			}
		}
	}

	// 2. Primary key column suffix stripping.
	// When multiple key columns exist, cardinality distinguishes the true PK
	// (all-unique values) from foreign keys (repeated values).
	keyColumns := findAllKeyColumns(cols)
	keyCol := bestKeyColumnByCardinality(keyColumns, rows)
	if keyCol != "" {
		if t := stripKeySuffix(keyCol); t != "Entity" {
			return t
		}
	}

	// 3. URI last meaningful path segment.
	uri := cir.Source.URI
	if uri != "" {
		parts := strings.Split(strings.TrimRight(uri, "/"), "/")
		for i := len(parts) - 1; i >= 0; i-- {
			seg := parts[i]
			if seg != "" && !strings.HasPrefix(strings.ToLower(seg), "storage") {
				return colNameToType(seg)
			}
		}
	}

	return "Entity"
}

// findAllKeyColumns returns all column names that match a known identifier suffix
// (e.g. _id, _key, _code). The caller can then choose among them by cardinality.
func findAllKeyColumns(cols []colInfo) []string {
	var keyColumns []string
	for _, col := range cols {
		lower := strings.ToLower(col.name)
		for _, suffix := range keyNameSuffixes {
			if lower == suffix || strings.HasSuffix(lower, "_"+suffix) {
				keyColumns = append(keyColumns, col.name)
				break
			}
		}
	}
	return keyColumns
}

// bestKeyColumnByCardinality picks the column with the highest value cardinality
// across the provided rows. A true primary key has all-unique values (ratio ≈ 1.0)
// while foreign keys carry repeated values (ratio < 1.0).
//
// Falls back to the shortest column name when rows is empty or cardinalities tie.
func bestKeyColumnByCardinality(keyColumns []string, rows []interface{}) string {
	if len(keyColumns) == 0 {
		return ""
	}
	if len(keyColumns) == 1 {
		return keyColumns[0]
	}

	// Count unique values per candidate key column.
	valueSets := make(map[string]map[string]struct{}, len(keyColumns))
	for _, col := range keyColumns {
		valueSets[col] = make(map[string]struct{})
	}
	totalRows := 0
	for _, rawRow := range rows {
		row, ok := rawRow.(map[string]interface{})
		if !ok {
			continue
		}
		totalRows++
		for _, col := range keyColumns {
			v := fmt.Sprintf("%v", row[col])
			valueSets[col][v] = struct{}{}
		}
	}

	if totalRows == 0 {
		// No row data: fall back to shortest name.
		best := keyColumns[0]
		for _, c := range keyColumns[1:] {
			if len(c) < len(best) {
				best = c
			}
		}
		return best
	}

	// Pick the column with the highest cardinality ratio.
	bestCol := keyColumns[0]
	bestRatio := float64(len(valueSets[bestCol])) / float64(totalRows)
	for _, col := range keyColumns[1:] {
		ratio := float64(len(valueSets[col])) / float64(totalRows)
		if ratio > bestRatio {
			bestRatio = ratio
			bestCol = col
		}
	}
	return bestCol
}

// detectPrimaryKeyColumn returns the column most likely to be the primary key.
// When entityTypeHint is set (lower-case), the function prefers a key column
// whose name starts with that prefix (e.g. "student" → prefers "student_id").
// Among equally-prefixed columns, the shortest name wins.
func detectPrimaryKeyColumn(cols []colInfo, entityTypeHint string) string {
	var keyColumns []string
	for _, col := range cols {
		lower := strings.ToLower(col.name)
		for _, suffix := range keyNameSuffixes {
			if lower == suffix || strings.HasSuffix(lower, "_"+suffix) {
				keyColumns = append(keyColumns, col.name)
				break
			}
		}
	}
	if len(keyColumns) == 0 {
		return ""
	}
	if len(keyColumns) == 1 {
		return keyColumns[0]
	}
	// Prefer column whose name starts with the entity type hint.
	if entityTypeHint != "" {
		for _, kc := range keyColumns {
			if strings.HasPrefix(strings.ToLower(kc), entityTypeHint) {
				return kc
			}
		}
	}
	// Fall back: shortest name (primary keys tend to be briefer than foreign keys).
	sort.Slice(keyColumns, func(i, j int) bool {
		return len(keyColumns[i]) < len(keyColumns[j])
	})
	return keyColumns[0]
}

// stripKeySuffix removes the known identifier suffix from a column name and
// returns the remainder as a PascalCase entity type.
//
//	"student_id"      → "Student"
//	"order_code"      → "Order"
//	"grade_record_id" → "GradeRecord"
//	"id"              → "Entity"  (bare suffix, no meaningful prefix)
func stripKeySuffix(colName string) string {
	lower := strings.ToLower(colName)
	for _, suffix := range keyNameSuffixes {
		sep := "_" + suffix
		if strings.HasSuffix(lower, sep) {
			base := colName[:len(colName)-len(sep)]
			if base == "" {
				return "Entity"
			}
			return colNameToType(base)
		}
		if lower == suffix {
			return "Entity"
		}
	}
	return colNameToType(colName)
}

// rowFingerprint returns a deterministic string identifying a row from its
// sorted key=value pairs, used when no primary key column is available.
func rowFingerprint(row map[string]interface{}) string {
	keys := make([]string, 0, len(row))
	for k := range row {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%v", k, row[k]))
	}
	return strings.Join(parts, "|")
}

// ─── Shared utilities ─────────────────────────────────────────────────────────

// normalizeText performs basic text normalisation: lowercase, trim, collapse spaces.
func normalizeText(text string) string {
	normalized := strings.ToLower(strings.TrimSpace(text))
	return strings.Join(strings.Fields(normalized), " ")
}
