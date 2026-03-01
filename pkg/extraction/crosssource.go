package extraction

// Cross-source link detection
//
// Two columns from separate storage sources are considered "linked" (i.e. a
// foreign-key-style join point) when their value sets significantly overlap AND
// at least one looks like an identifier column.  No domain-specific
// configuration is needed — the algorithm is entirely data-driven.
//
// Three signals combine into a single confidence score:
//
//  1. Value overlap  – Jaccard(valuesA, valuesB).
//     The primary signal: if student_id={1..200} in the grades DB also
//     appears as student_id={1..200} in the attendance DB the Jaccard ≈ 1.0.
//
//  2. Name similarity – token-Jaccard of normalised column name tokens.
//     "student_id", "studentId", "StudentID" → tokens {student, id}.
//     High name similarity amplifies a moderate value-overlap signal.
//     Low name similarity (e.g. "sid" vs "student_id") does NOT suppress a
//     strong value-overlap signal — the value overlap carries the detection.
//
//  3. Key-likeness gate – at least one column must be high-cardinality
//     (≥50% unique values) or carry a conventional identifier name token
//     ("id", "key", "code", etc.).  This prevents low-cardinality vocabulary
//     columns (status={"active","inactive"}) from being misidentified as join
//     keys even when their small value sets perfectly overlap.

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"unicode"

	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

// ─── Tuneable parameters ──────────────────────────────────────────────────────

const (
	// maxValueSample caps stored distinct values per column to keep memory bounded
	// while preserving Jaccard accuracy for typical dataset sizes.
	maxValueSample = 5000

	// minValueOverlap: Jaccard pairs below this threshold are skipped before
	// computing the other signals (fast-path rejection).
	minValueOverlap = 0.10

	// minLinkConfidence: minimum combined score to emit a CrossSourceLink.
	minLinkConfidence = 0.35

	// highCardinalityThreshold: cardinality ratio above which a column is
	// treated as high-cardinality (likely an identifier, not a category).
	// Set to 0.60 so that moderately-repeating numeric columns (e.g. scores,
	// days_absent) are not mistaken for join keys.
	highCardinalityThreshold = 0.60
)

// keyNameTokens are word tokens that, when present in a column name, strongly
// suggest the column is a unique identifier.
var keyNameTokens = map[string]bool{
	"id": true, "key": true, "code": true, "number": true,
	"uuid": true, "ref": true, "identifier": true, "no": true,
	"num": true, "email": true, "username": true, "token": true,
}

// ─── Phase 1: Column profile construction ────────────────────────────────────

// BuildColumnProfilesFromCIRs derives a ColumnProfile for every column found
// across all CIR records from a single storage source.
//
// Unlike the structured extraction path (which uses stringVal to exclude
// numeric values for NLP entity detection), this function coerces ALL value
// types to strings.  This is intentional: numeric IDs like student_id=42 are
// the most common cross-source join keys and must be captured in the value
// sample for Jaccard overlap to work.
func BuildColumnProfilesFromCIRs(storageID string, cirs []*models.CIR) []models.ColumnProfile {
	entityType := ""

	// Accumulate per-column statistics.
	type colAcc struct {
		valueSample map[string]bool
		totalRows   int
		numericRows int
	}
	colMap := make(map[string]*colAcc)
	var colOrder []string

	for _, cir := range cirs {
		if cir == nil {
			continue
		}
		// Prefer explicit entity_type parameter set during ingestion.
		if et, ok := cir.GetParameter("entity_type"); ok {
			if s, _ := et.(string); s != "" && entityType == "" {
				entityType = s
			}
		}
		if entityType == "" {
			entityType = uriLastSegment(cir.Source.URI)
		}

		// Expand CIR data into flat rows.
		var rows []map[string]interface{}
		switch d := cir.Data.(type) {
		case []interface{}:
			for _, item := range d {
				if m, ok := item.(map[string]interface{}); ok {
					rows = append(rows, m)
				}
			}
		case map[string]interface{}:
			rows = append(rows, d)
		}

		for _, row := range rows {
			for col, val := range row {
				acc, seen := colMap[col]
				if !seen {
					acc = &colAcc{valueSample: make(map[string]bool)}
					colMap[col] = acc
					colOrder = append(colOrder, col)
				}
				acc.totalRows++
				// coerceToProfileString captures ALL value types including
				// integers and floats — essential for numeric ID matching.
				s := coerceToProfileString(val)
				if s != "" && len(acc.valueSample) < maxValueSample {
					acc.valueSample[s] = true
				}
				if looksLikeNumericValue(val) {
					acc.numericRows++
				}
			}
		}
	}

	if len(colOrder) == 0 {
		return nil
	}

	profiles := make([]models.ColumnProfile, 0, len(colOrder))
	for _, col := range colOrder {
		acc := colMap[col]
		if acc.totalRows == 0 {
			continue
		}
		unique := len(acc.valueSample)
		ratio := float64(unique) / float64(acc.totalRows)
		isNumeric := acc.numericRows > acc.totalRows/2

		profiles = append(profiles, models.ColumnProfile{
			StorageID:        storageID,
			EntityType:       entityType,
			ColumnName:       col,
			ValueSample:      acc.valueSample,
			TotalRows:        acc.totalRows,
			UniqueCount:      unique,
			CardinalityRatio: ratio,
			IsNumeric:        isNumeric,
			IsLikelyKey:      isKeyColumn(col, ratio),
		})
	}
	return profiles
}

// coerceToProfileString converts any value to a canonical string for use in
// cross-source value-overlap computation.  Unlike stringVal in structured.go,
// this deliberately includes numeric values because integer/float IDs are the
// most common cross-source join keys.
func coerceToProfileString(v interface{}) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		s := strings.TrimSpace(t)
		// Normalise numeric strings: "1.0" → "1", "42.00" → "42"
		return normaliseNumericStr(s)
	case float64:
		// Use %g to avoid scientific notation for typical IDs; strip trailing zeros.
		s := fmt.Sprintf("%g", t)
		return s
	case float32:
		s := fmt.Sprintf("%g", float64(t))
		return s
	case int:
		return fmt.Sprintf("%d", t)
	case int64:
		return fmt.Sprintf("%d", t)
	case int32:
		return fmt.Sprintf("%d", t)
	case bool:
		if t {
			return "true"
		}
		return "false"
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", v))
	}
}

// normaliseNumericStr strips trailing decimal zeros so that "1", "1.0", and
// "1.000" all map to the same canonical value for Jaccard comparison.
func normaliseNumericStr(s string) string {
	if !strings.Contains(s, ".") {
		return s
	}
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	return s
}

// looksLikeNumericValue returns true for native numeric types.
func looksLikeNumericValue(v interface{}) bool {
	switch v.(type) {
	case float32, float64, int, int32, int64:
		return true
	}
	return false
}

// isKeyColumn returns true when the column appears to be an identifier column:
// either its name contains a conventional key token, or its cardinality ratio
// is high (most values are unique, as expected for a primary/foreign key).
func isKeyColumn(name string, cardRatio float64) bool {
	if cardRatio >= highCardinalityThreshold {
		return true
	}
	for _, tok := range normaliseColumnTokens(name) {
		if keyNameTokens[tok] {
			return true
		}
	}
	return false
}

// ─── Phase 2: Cross-source link detection ────────────────────────────────────

// DetectCrossSourceLinks scores every cross-storage column pair and returns
// those with confidence ≥ minLinkConfidence, sorted by confidence descending.
// Only pairs from *different* storage sources are compared.
func DetectCrossSourceLinks(profiles []models.ColumnProfile) []models.CrossSourceLink {
	if len(profiles) < 2 {
		return nil
	}

	// Group profiles by storage so we never compare a column with itself.
	byStorage := make(map[string][]models.ColumnProfile)
	var storageOrder []string
	for _, p := range profiles {
		if _, seen := byStorage[p.StorageID]; !seen {
			storageOrder = append(storageOrder, p.StorageID)
		}
		byStorage[p.StorageID] = append(byStorage[p.StorageID], p)
	}

	if len(storageOrder) < 2 {
		return nil
	}

	var links []models.CrossSourceLink
	for i := 0; i < len(storageOrder); i++ {
		for j := i + 1; j < len(storageOrder); j++ {
			profA := byStorage[storageOrder[i]]
			profB := byStorage[storageOrder[j]]
			for _, a := range profA {
				for _, b := range profB {
					if link, ok := scorePair(a, b); ok {
						links = append(links, link)
					}
				}
			}
		}
	}

	sort.Slice(links, func(i, j int) bool {
		return links[i].Confidence > links[j].Confidence
	})
	return deduplicateLinks(links)
}

// scorePair computes the confidence that two ColumnProfiles represent the same
// domain concept (i.e. are a natural join key across storage sources).
func scorePair(a, b models.ColumnProfile) (models.CrossSourceLink, bool) {
	if len(a.ValueSample) == 0 || len(b.ValueSample) == 0 {
		return models.CrossSourceLink{}, false
	}

	// Gate: BOTH columns must look like identifier columns.
	// A low-cardinality categorical (status, grade) or a numeric measurement
	// (score, days_absent) may happen to overlap in value range with a real
	// join key but is not itself a join key.  Requiring both to be key-like
	// eliminates these false positives while still allowing "sid" ↔ "student_id"
	// where both are key-like (one by high cardinality, one by name token).
	if !a.IsLikelyKey || !b.IsLikelyKey {
		return models.CrossSourceLink{}, false
	}

	// Signal 1: value overlap (Jaccard).  Fast-path: skip before computing
	// the more expensive name similarity if overlap is below threshold.
	overlap, shared := jaccardOverlap(a.ValueSample, b.ValueSample)
	if overlap < minValueOverlap {
		return models.CrossSourceLink{}, false
	}

	// Signal 2: column name similarity.
	nameSim := columnNameSimilarity(a.ColumnName, b.ColumnName)

	// Signal 3: key-likeness bonus.
	keyBonus := 0.0
	switch {
	case a.IsLikelyKey && b.IsLikelyKey:
		keyBonus = 0.20
	case a.IsLikelyKey || b.IsLikelyKey:
		keyBonus = 0.10
	}

	// Combined confidence: value overlap is the dominant signal.
	confidence := math.Min(0.50*overlap+0.30*nameSim+keyBonus, 0.97)
	if confidence < minLinkConfidence {
		return models.CrossSourceLink{}, false
	}

	return models.CrossSourceLink{
		StorageA:         a.StorageID,
		ColumnA:          a.ColumnName,
		EntityTypeA:      a.EntityType,
		StorageB:         b.StorageID,
		ColumnB:          b.ColumnName,
		EntityTypeB:      b.EntityType,
		Confidence:       roundTo3(confidence),
		NameSimilarity:   roundTo3(nameSim),
		ValueOverlap:     roundTo3(overlap),
		SharedValueCount: shared,
	}, true
}

// ─── Similarity helpers ───────────────────────────────────────────────────────

// jaccardOverlap returns the Jaccard similarity index and the raw intersection
// count between two value sets.  Iterates over the smaller set for efficiency.
func jaccardOverlap(a, b map[string]bool) (float64, int) {
	if len(a) == 0 || len(b) == 0 {
		return 0, 0
	}
	small, large := a, b
	if len(a) > len(b) {
		small, large = b, a
	}
	intersection := 0
	for v := range small {
		if large[v] {
			intersection++
		}
	}
	union := len(a) + len(b) - intersection
	if union == 0 {
		return 0, 0
	}
	return float64(intersection) / float64(union), intersection
}

// columnNameSimilarity returns a [0,1] score by comparing the normalised token
// sets of two column names using Jaccard similarity.
//
// Examples:
//
//	"student_id"  → {student, id}
//	"studentId"   → {student, id}  → similarity 1.0
//	"sid"         → {sid}          → similarity 0.0 with {student,id}
//	                BUT value overlap will still detect this pair.
func columnNameSimilarity(a, b string) float64 {
	tokA := normaliseColumnTokens(a)
	tokB := normaliseColumnTokens(b)
	if len(tokA) == 0 || len(tokB) == 0 {
		return 0
	}
	setA := make(map[string]bool, len(tokA))
	setB := make(map[string]bool, len(tokB))
	for _, t := range tokA {
		setA[t] = true
	}
	for _, t := range tokB {
		setB[t] = true
	}
	intersection := 0
	for t := range setA {
		if setB[t] {
			intersection++
		}
	}
	union := len(setA) + len(setB) - intersection
	if union == 0 {
		return 0
	}
	return float64(intersection) / float64(union)
}

// normaliseColumnTokens splits a column name into lowercase word tokens,
// handling snake_case, camelCase, PascalCase, kebab-case, and ALLCAPS.
//
// Examples:
//
//	"student_id"  → ["student", "id"]
//	"studentId"   → ["student", "id"]
//	"StudentID"   → ["student", "id"]
//	"STUDENT_ID"  → ["student", "id"]
func normaliseColumnTokens(name string) []string {
	// Step 1: split on non-alphanumeric separators.
	parts := strings.FieldsFunc(name, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	var tokens []string
	for _, part := range parts {
		// Step 2: split camelCase / PascalCase within each part.
		var cur strings.Builder
		runes := []rune(part)
		for i, r := range runes {
			if i > 0 && unicode.IsUpper(r) {
				prev := runes[i-1]
				nextLower := i+1 < len(runes) && unicode.IsLower(runes[i+1])
				if unicode.IsLower(prev) || (unicode.IsUpper(prev) && nextLower) {
					if t := strings.ToLower(cur.String()); t != "" {
						tokens = append(tokens, t)
					}
					cur.Reset()
				}
			}
			cur.WriteRune(r)
		}
		if t := strings.ToLower(cur.String()); t != "" {
			tokens = append(tokens, t)
		}
	}
	return tokens
}

// ─── Key field detection for entity resolution ───────────────────────────────

// detectKeyFields returns the attribute names in a record that are likely to
// be stable identifiers (suitable as join keys for entity resolution).
// It uses the same heuristics as isKeyColumn: key-like token in the name.
// Numeric keys and string keys are both included.
func detectKeyFields(attributes map[string]interface{}) []string {
	var keys []string
	for k := range attributes {
		if isKeyColumn(k, 1.0) { // pass 1.0 so only name-based heuristic applies here
			keys = append(keys, k)
		}
	}
	sort.Strings(keys) // deterministic order
	return keys
}

// keyValue returns a canonical string representation of a key field value,
// used for equality comparison across records.
func keyValue(v interface{}) string {
	if v == nil {
		return ""
	}
	s := fmt.Sprintf("%v", v)
	// Normalise numeric representations: "1.0" == "1", "42.000" == "42"
	if strings.Contains(s, ".") {
		s = strings.TrimRight(s, "0")
		s = strings.TrimRight(s, ".")
	}
	return strings.TrimSpace(s)
}

// ─── Deduplication ────────────────────────────────────────────────────────────

// deduplicateLinks removes redundant cross-source links, keeping the
// highest-confidence link per canonical (entityTypeA, columnA, entityTypeB, columnB)
// pair.  Canonical order: entityTypeA ≤ entityTypeB lexicographically.
func deduplicateLinks(links []models.CrossSourceLink) []models.CrossSourceLink {
	type pairKey struct{ ea, colA, eb, colB string }
	seen := make(map[pairKey]bool, len(links))
	result := make([]models.CrossSourceLink, 0, len(links))
	for _, l := range links {
		ea, colA, eb, colB := l.EntityTypeA, l.ColumnA, l.EntityTypeB, l.ColumnB
		// Canonical ordering so A↔B == B↔A.
		if ea > eb || (ea == eb && colA > colB) {
			ea, colA, eb, colB = eb, colB, ea, colA
		}
		k := pairKey{ea, colA, eb, colB}
		if !seen[k] {
			seen[k] = true
			result = append(result, l)
		}
	}
	return result
}

// ─── Utilities ────────────────────────────────────────────────────────────────

func uriLastSegment(uri string) string {
	uri = strings.TrimRight(uri, "/")
	idx := strings.LastIndex(uri, "/")
	if idx < 0 {
		return uri
	}
	seg := uri[idx+1:]
	// Strip query string / fragment.
	if i := strings.IndexAny(seg, "?#"); i >= 0 {
		seg = seg[:i]
	}
	return seg
}

func roundTo3(f float64) float64 {
	return math.Round(f*1000) / 1000
}
