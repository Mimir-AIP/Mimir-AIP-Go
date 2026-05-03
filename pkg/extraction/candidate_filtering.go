package extraction

import (
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

func isFiltered(t string) bool {
	t = strings.TrimSpace(t)
	if len(t) == 0 {
		return true
	}

	// Reject single-rune tokens.
	if utf8.RuneCountInString(t) < 2 {
		return true
	}

	// Reject strings that are entirely digits (possibly with separators).
	allNumeric := true
	for _, r := range t {
		if !unicode.IsDigit(r) && r != '.' && r != ',' && r != '-' && r != '+' && r != '/' {
			allNumeric = false
			break
		}
	}
	if allNumeric {
		return true
	}

	// Reject strings with no letters at all.
	hasLetter := false
	for _, r := range t {
		if unicode.IsLetter(r) {
			hasLetter = true
			break
		}
	}
	if !hasLetter {
		return true
	}

	// Reject very short all-lowercase single words — almost certainly function words.
	if wordCount(t) == 1 && utf8.RuneCountInString(t) <= 3 && strings.ToLower(t) == t {
		return true
	}

	// Reject single-word stopwords — common function words that carry no
	// entity signal regardless of length (e.g. "however", "therefore").
	if wordCount(t) == 1 && isStopword(t) {
		return true
	}

	return false
}

// ─── Subsumption pruning ──────────────────────────────────────────────────────

// subsumptionPrune removes shorter n-grams that are effectively redundant
// because a longer, higher-scoring n-gram subsumes them.
//
// A token A is subsumed by token B when:
//   - B contains A as a contiguous word-level substring, AND
//   - B appears in at least 85% of the records that A appears in
//     (meaning A rarely occurs without B), AND
//   - score(B) >= score(A) - 0.05 (B is at least as informative).
//
// When A is subsumed, its score is reduced by 0.25; if the reduced score
// falls below minEntityScore it is dropped entirely.
func subsumptionPrune(candidates map[string]float64, idx *corpusIndex) map[string]float64 {
	// Sort candidates by word count descending so longer n-grams are processed first.
	type entry struct {
		text  string
		score float64
		wc    int
	}
	entries := make([]entry, 0, len(candidates))
	for t, sc := range candidates {
		entries = append(entries, entry{t, sc, wordCount(t)})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].wc != entries[j].wc {
			return entries[i].wc > entries[j].wc
		}
		return entries[i].score > entries[j].score
	})

	result := make(map[string]float64, len(candidates))
	for _, e := range entries {
		result[e.text] = e.score
	}

	for i := 0; i < len(entries); i++ {
		longer := entries[i]
		longerWords := strings.Fields(longer.text)

		for j := i + 1; j < len(entries); j++ {
			shorter := entries[j]
			if shorter.wc >= longer.wc {
				continue
			}

			// Check word-level containment.
			if !wordSubstring(strings.Fields(shorter.text), longerWords) {
				continue
			}

			// Check coverage: does the longer form appear wherever the shorter does?
			dfShorter := idx.docFreq[shorter.text]
			dfLonger := idx.docFreq[longer.text]
			if dfShorter == 0 {
				continue
			}
			coverage := float64(dfLonger) / float64(dfShorter)
			if coverage < 0.85 {
				continue // shorter appears standalone too often
			}

			// Check that the longer form is roughly as good.
			if longer.score < shorter.score-0.05 {
				continue
			}

			// Demote the shorter form.
			newScore := result[shorter.text] - 0.25
			if newScore < minEntityScore {
				delete(result, shorter.text)
			} else {
				result[shorter.text] = newScore
			}
		}
	}

	return result
}

// wordSubstring returns true if needle (as a slice of words) is a contiguous
// subsequence of haystack.
func wordSubstring(needle, haystack []string) bool {
	if len(needle) == 0 || len(needle) > len(haystack) {
		return false
	}
outer:
	for i := 0; i <= len(haystack)-len(needle); i++ {
		for j, w := range needle {
			if !strings.EqualFold(haystack[i+j], w) {
				continue outer
			}
		}
		return true
	}
	return false
}

// ─── Entity emission ─────────────────────────────────────────────────────────

// candidatesToEntities converts scored candidates to ExtractedEntity values,
