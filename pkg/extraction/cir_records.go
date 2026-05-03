package extraction

import (
	"fmt"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"strings"
	"unicode"
	"unicode/utf8"
)

func normaliseFieldKey(fk string) string {
	fk = strings.ToLower(strings.TrimSpace(fk))
	fk = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return r
		}
		return '_'
	}, fk)
	return strings.Trim(fk, "_")
}

// ─── Tokenisation ────────────────────────────────────────────────────────────

// cirToRows converts a CIR item into one or more extractionRecords.
//
// The CIR format allows Data to be any of:
//   - []interface{}  — tabular/structured: each element is a row in its own right
//     (e.g. a CSV file, a database result set, a JSON array of objects).
//     Each row becomes a separate corpus document so that IDF and NPMI
//     statistics are computed at row granularity, not table granularity.
//   - map[string]interface{} — a single document (possibly hybrid: structured
//     fields alongside free-text descriptions).  The whole map is one record.
//   - string — a raw text blob.  Treated as one record.
//   - anything else — stringified and treated as one record if it looks like
//     a meaningful identifier.
//
// This function is the single point where CIR format meets the statistical
// extraction algorithm, so all CIR layouts are handled uniformly.
func cirToRows(cir *models.CIR) []extractionRecord {
	if cir == nil {
		return nil
	}

	// Top-level array → one record per element.
	if arr, ok := cir.Data.([]interface{}); ok {
		rows := make([]extractionRecord, 0, len(arr))
		for _, elem := range arr {
			var rec extractionRecord
			extractFromValue("", elem, &rec)
			if len(rec.ngrams) > 0 {
				rows = append(rows, rec)
			}
		}
		return rows
	}

	// Single document (map, string, or other).
	var rec extractionRecord
	extractFromValue("", cir.Data, &rec)
	if len(rec.ngrams) == 0 {
		return nil
	}
	return []extractionRecord{rec}
}

// extractFromValue recursively walks a data value and tokenises all string
// leaves into n-gram occurrences.  It understands maps (field key → value),
// arrays (elements inherit the parent field key), strings (tokenised
// directly), and any other scalar (stringified only if it looks like an
// identifier, e.g. "SENSOR-4821").
func extractFromValue(fieldKey string, val interface{}, rec *extractionRecord) {
	if val == nil {
		return
	}
	switch v := val.(type) {
	case string:
		if strings.TrimSpace(v) != "" {
			rec.ngrams = append(rec.ngrams, tokeniseFieldValue(fieldKey, v)...)
		}
	case map[string]interface{}:
		for k, child := range v {
			extractFromValue(k, child, rec)
		}
	case []interface{}:
		// Nested arrays within a document share the parent field key.
		for _, child := range v {
			extractFromValue(fieldKey, child, rec)
		}
	default:
		// Numbers, booleans etc. — only include if the string representation
		// looks like a meaningful identifier (contains both letters and other
		// characters, e.g. "SENSOR-4821", "v2.3-beta").
		s := fmt.Sprintf("%v", v)
		if looksLikeIdentifier(s) {
			rec.ngrams = append(rec.ngrams, tokeniseFieldValue(fieldKey, s)...)
		}
	}
}

// looksLikeIdentifier returns true for non-numeric string representations
// that might be entity identifiers (contain letters mixed with other chars).
func looksLikeIdentifier(s string) bool {
	hasLetter := false
	hasOther := false
	for _, r := range s {
		if unicode.IsLetter(r) {
			hasLetter = true
		} else {
			hasOther = true
		}
	}
	return hasLetter && hasOther
}

// tokeniseFieldValue splits a field value into all n-grams of size 1..maxNgramLen.
// It uses sentence-boundary segmentation (from nlp.go) so that n-grams never
// straddle sentence breaks ("disease. The hospital" never produces a bigram).
// Multi-word n-grams whose first or last word is a stopword are discarded —
// they are unlikely to be named entities and pollute the index.
// Capitalisation and full-value flags are recorded for each occurrence.
func tokeniseFieldValue(fieldKey, text string) []ngramOcc {
	trimmedText := strings.TrimSpace(text)
	sentences := splitSentences(text)
	if len(sentences) == 0 {
		return nil
	}

	var occs []ngramOcc
	wordOffset := 0

	for _, words := range sentences {
		for start := 0; start < len(words); start++ {
			for size := 1; size <= maxNgramLen && start+size <= len(words); size++ {
				// Discard multi-word n-grams that begin or end with a stopword.
				// Single-word stopwords are allowed through here and filtered
				// later by isFiltered so that single-word scoring dimensions
				// (capScore, valueScore) still penalise them naturally before
				// the explicit stopword gate.
				if size > 1 {
					if isStopword(words[start]) || isStopword(words[start+size-1]) {
						continue
					}
				}

				phrase := strings.Join(words[start:start+size], " ")
				if phrase == "" {
					continue
				}

				firstRune, _ := utf8.DecodeRuneInString(phrase)
				capFirst := unicode.IsUpper(firstRune)
				isFullVal := strings.EqualFold(phrase, trimmedText)

				occs = append(occs, ngramOcc{
					text:      phrase,
					fieldKey:  fieldKey,
					position:  wordOffset + start,
					capFirst:  capFirst,
					isFullVal: isFullVal,
				})
			}
		}
		wordOffset += len(words)
	}
	return occs
}

// ─── Utilities ────────────────────────────────────────────────────────────────

// wordCount returns the number of whitespace-separated words in s.
func wordCount(s string) int {
	return len(strings.Fields(s))
}
