package extraction

// nlp.go — pure-Go NLP utilities used by the statistical extraction algorithm.
//
// No external dependencies.  All techniques are hand-crafted and data-agnostic:
// they operate on statistical properties of the corpus, not on domain knowledge.

import (
	"math"
	"strings"
	"unicode"
	"unicode/utf8"
)

// ─── Stopwords ────────────────────────────────────────────────────────────────

// stopwords is a set of common English function words.
// Multi-word n-gram candidates that start or end with a stopword are rejected
// during tokenisation because they are unlikely to be named entities.
// Single-word stopwords are filtered separately by isFiltered → isStopword.
var stopwords = func() map[string]bool {
	words := []string{
		// articles
		"a", "an", "the",
		// coordinating & subordinating conjunctions
		"and", "but", "or", "nor", "for", "yet", "so",
		"because", "although", "though", "while", "whether",
		"if", "unless", "until", "since", "whereas",
		// prepositions
		"in", "on", "at", "to", "of", "with", "by", "from", "up",
		"about", "into", "through", "during", "before", "after",
		"above", "below", "between", "out", "off", "over", "under",
		"around", "within", "without", "across", "behind", "beyond",
		"near", "beside", "against", "along", "among", "except",
		"per", "upon", "toward", "towards", "onto", "via", "versus",
		// pronouns
		"i", "me", "my", "we", "our", "you", "your",
		"he", "him", "his", "she", "her", "it", "its",
		"they", "them", "their", "this", "these", "those", "that",
		"what", "which", "who", "whom", "whose",
		// auxiliary / high-frequency verbs
		"is", "are", "was", "were", "be", "been", "being",
		"have", "has", "had", "do", "does", "did",
		"will", "would", "could", "should", "may", "might",
		"must", "shall", "can", "need", "ought",
		// common adverbs & determiners
		"not", "no", "only", "also", "just", "both", "all", "each",
		"few", "more", "most", "other", "some", "such", "than", "too",
		"very", "so", "as", "then", "when", "where", "how", "why",
		"once", "here", "there", "now", "again", "further",
		"however", "therefore", "thus", "hence", "already", "still",
		"else", "even", "ever", "well", "often", "always", "never",
		"already", "rather", "quite", "almost", "enough",
		// high-frequency verbs that add no entity signal
		"get", "got", "go", "went", "come", "came", "make", "made",
		"take", "took", "see", "saw", "know", "knew", "think",
		"give", "gave", "find", "found", "use", "used", "say", "said",
		"tell", "told", "call", "show", "seem", "look", "feel",
		"let", "put", "set", "keep", "start", "try", "work",
	}
	m := make(map[string]bool, len(words)*2)
	for _, w := range words {
		m[w] = true
	}
	return m
}()

// isStopword returns true if the lower-cased word is a common function word.
func isStopword(word string) bool {
	return stopwords[strings.ToLower(word)]
}

// ─── Sentence segmentation ────────────────────────────────────────────────────

// splitSentences divides text into sentence-level slices of cleaned words.
// N-grams are generated within sentence boundaries to prevent cross-sentence
// phrases such as "disease. The hospital" producing a spurious 3-gram
// "disease The hospital".
//
// Sentence boundaries are detected when a word ends with ".", "!", or "?"
// AND the next word starts with an uppercase letter AND the word is not a
// known abbreviation (Dr., Inc., U.S.A. etc.).
//
// Each returned []string is a list of cleaned words (punctuation stripped
// from boundaries, internal punctuation preserved, e.g. "St. Mary's").
func splitSentences(text string) [][]string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}

	rawWords := strings.Fields(text)
	var sentences [][]string
	var current []string

	for i, raw := range rawWords {
		cleaned := cleanWord(raw)
		if cleaned != "" {
			current = append(current, cleaned)
		}

		// Detect sentence-ending punctuation, but not inside abbreviations.
		if i < len(rawWords)-1 {
			if wordEndsSentence(raw) {
				nextRune, _ := utf8.DecodeRuneInString(rawWords[i+1])
				if unicode.IsUpper(nextRune) {
					sentences = append(sentences, current)
					current = nil
				}
			}
		}
	}
	if len(current) > 0 {
		sentences = append(sentences, current)
	}
	return sentences
}

// wordEndsSentence returns true if the word carries terminal punctuation
// that is NOT an abbreviation marker.
func wordEndsSentence(raw string) bool {
	if strings.HasSuffix(raw, "!") || strings.HasSuffix(raw, "?") {
		return true
	}
	if strings.HasSuffix(raw, ".") {
		return !isAbbreviationWord(raw)
	}
	return false
}

// isAbbreviationWord returns true for tokens that end with a period but do NOT
// mark a sentence boundary: title abbreviations, corporate suffixes, single
// initials, short all-caps strings (acronyms), and common Latin abbreviations.
func isAbbreviationWord(raw string) bool {
	base := strings.ToLower(strings.TrimSuffix(raw, "."))
	// Single letter (initial): J. K. Q.
	if utf8.RuneCountInString(base) == 1 {
		return true
	}
	// Short all-caps (acronym): USA. NATO. IBM.
	if base == strings.ToUpper(base) && utf8.RuneCountInString(base) <= 5 {
		return true
	}
	knownAbbreviations := []string{
		"dr", "mr", "mrs", "ms", "prof", "sr", "jr", "rev", "gov", "gen", "col",
		"inc", "ltd", "corp", "co", "llc", "plc", "llp",
		"st", "ave", "blvd", "rd", "dept",
		"vs", "etc", "eg", "ie", "approx", "est",
		"jan", "feb", "mar", "apr", "jun", "jul", "aug", "sep", "oct", "nov", "dec",
		"vol", "no", "fig", "ed", "eds", "pp", "op",
	}
	for _, abbr := range knownAbbreviations {
		if base == abbr {
			return true
		}
	}
	return false
}

// cleanWord strips leading/trailing punctuation while preserving internal
// structure meaningful to entities: hyphens, apostrophes, and periods
// (for abbreviations like "St." and "U.S.A.").
func cleanWord(raw string) string {
	return strings.TrimFunc(raw, func(r rune) bool {
		return unicode.IsPunct(r) && r != '-' && r != '\'' && r != '.'
	})
}

// ─── BM25 IDF ─────────────────────────────────────────────────────────────────

// bm25IDFScore returns the BM25 inverse document frequency, normalised to [0,1].
//
// Standard IDF collapses to 0 when df == N (a term in every document gets no
// signal at all).  BM25 IDF gives a non-zero floor in this case, which is
// important for focused corpora where the subject entity appears in every record
// (e.g., all 20 records in a dataset are about the same product).
//
// Formula:  log((N − df + 0.5) / (df + 0.5) + 1)
// Normalised by dividing by the maximum value achievable (when df == 1).
func bm25IDFScore(N, df int) float64 {
	if N == 0 || df == 0 {
		return 0
	}
	raw := math.Log((float64(N)-float64(df)+0.5)/(float64(df)+0.5) + 1)
	// Maximum raw value occurs at df == 1.
	maxRaw := math.Log((float64(N)-1+0.5)/(1+0.5) + 1)
	if maxRaw == 0 {
		return 0
	}
	s := raw / maxRaw
	if s < 0 {
		return 0
	}
	if s > 1 {
		return 1
	}
	return s
}

// ─── Phrase cohesion ──────────────────────────────────────────────────────────

// phraseCohesion returns a [0, 1] score measuring how strongly the words in
// a multi-word n-gram attract each other across the corpus.
//
// For a bigram "w1 w2":
//
//	PMI(w1,w2) = log( P(w1,w2) / P(w1)·P(w2) )
//	           = log( docFreq[w1 w2] · N / (docFreq[w1] · docFreq[w2]) )
//
// For longer n-grams the minimum pairwise PMI over consecutive word pairs
// is used (weakest-link principle).
//
// N-grams that contain internal stopwords receive a neutral score (0.5)
// because the stopword confounds PMI — e.g. "of" in "University of California"
// has extremely high co-occurrence with both neighbours and would inflate the
// cohesion score artificially.
//
// Returns 0.5 (neutral) for single-word tokens.
func phraseCohesion(ngram string, idx *corpusIndex) float64 {
	words := strings.Fields(ngram)
	if len(words) <= 1 {
		return 0.5
	}

	N := float64(idx.N)
	if N == 0 {
		return 0.5
	}

	// Neutral score for phrases with internal stopwords.
	for _, w := range words {
		if isStopword(w) {
			return 0.5
		}
	}

	minPMI := math.MaxFloat64
	for i := 0; i < len(words)-1; i++ {
		bigram := words[i] + " " + words[i+1]
		dfBigram := float64(idx.docFreq[bigram])
		dfW1 := float64(idx.docFreq[words[i]])
		dfW2 := float64(idx.docFreq[words[i+1]])

		if dfBigram == 0 || dfW1 == 0 || dfW2 == 0 {
			// Constituent words indexed but bigram never stored — treat as neutral.
			return 0.5
		}

		pmi := math.Log(dfBigram * N / (dfW1 * dfW2))
		if pmi < minPMI {
			minPMI = pmi
		}
	}

	if minPMI == math.MaxFloat64 {
		return 0.5
	}

	// Normalise PMI to [0, 1].
	// Max PMI ≈ log(N) (perfect co-occurrence).
	// Min PMI ≈ −log(N) (perfect anti-correlation).
	maxPMI := math.Log(N)
	if maxPMI <= 0 {
		return 0.5
	}
	normalised := minPMI / maxPMI
	if normalised > 1 {
		normalised = 1
	} else if normalised < -1 {
		normalised = -1
	}
	return (normalised + 1) / 2
}

// ─── Token morphology ─────────────────────────────────────────────────────────

// morphologyBoost returns an additive confidence bonus based on surface-form
// properties of a single-word token that serve as proxy signals for
// "proper-noun probability":
//
//   - ALL_CAPS abbreviation (≥2 uppercase, ≤6 chars, no lowercase): CEO, API → +0.08
//   - CamelCase brand/product name (uppercase not only at position 0): ThinkPad → +0.06
//   - Hyphenated compound (co-founder, built-in): +0.03
//
// Multi-word n-grams return 0 (their length and cohesion scores already carry
// this signal).
func morphologyBoost(token string) float64 {
	if wordCount(token) > 1 {
		return 0
	}

	runes := []rune(token)
	var upperCount, lowerCount int
	hasMidUpper := false
	for i, r := range runes {
		switch {
		case unicode.IsUpper(r):
			upperCount++
			if i > 0 {
				hasMidUpper = true
			}
		case unicode.IsLower(r):
			lowerCount++
		}
	}

	// ALL_CAPS abbreviation: CEO, API, NATO, IoT (allow ≤1 non-upper letter for IoT-style)
	if upperCount >= 2 && lowerCount <= 1 && len(runes) <= 6 {
		return 0.08
	}

	// CamelCase: ThinkPad, LinkedIn, macOS (has uppercase after position 0)
	if hasMidUpper {
		return 0.06
	}

	// Hyphenated compound: co-founder, end-to-end, state-of-the-art
	if strings.Contains(token, "-") {
		return 0.03
	}

	return 0
}

// ─── Fuzzy deduplication ──────────────────────────────────────────────────────

// fuzzyDeduplicate merges near-duplicate entity candidates before emitting
// the final entity list.  Two candidates are considered duplicates when:
//
//  1. Their punctuation-normalised lowercased forms are equal
//     (e.g. "St. Mary's" ≈ "St Marys"), OR
//  2. Both strings are ≥ minFuzzyLen runes long AND their normalised forms
//     have a Levenshtein distance ≤ maxFuzzyDist.
//
// In each duplicate group the highest-scoring candidate survives; others are
// removed.
func fuzzyDeduplicate(candidates map[string]float64) map[string]float64 {
	const maxFuzzyDist = 2
	const minFuzzyLen = 12 // only fuzzy-match longer tokens to avoid over-merging short distinct names

	type entry struct {
		text  string
		score float64
		norm  string // punctuation-normalised form for comparison
	}

	entries := make([]entry, 0, len(candidates))
	for t, sc := range candidates {
		entries = append(entries, entry{t, sc, punctNorm(t)})
	}

	absorbed := make(map[int]bool, len(entries))
	result := make(map[string]float64, len(candidates))

	for i := 0; i < len(entries); i++ {
		if absorbed[i] {
			continue
		}
		winner := entries[i]

		for j := i + 1; j < len(entries); j++ {
			if absorbed[j] {
				continue
			}
			b := entries[j]
			isDup := winner.norm == b.norm

			if !isDup &&
				utf8.RuneCountInString(winner.text) >= minFuzzyLen &&
				utf8.RuneCountInString(b.text) >= minFuzzyLen &&
				levenshtein(winner.norm, b.norm) <= maxFuzzyDist {
				isDup = true
			}

			if isDup {
				absorbed[j] = true
				if b.score > winner.score {
					// The new winner came from a later slot; retire the old one.
					absorbed[i] = true
					winner = b
				}
			}
		}

		result[winner.text] = winner.score
	}

	return result
}

// punctNorm strips all punctuation and whitespace-normalises a string for
// fuzzy comparison.  "St. Mary's Medical" → "st marys medical".
func punctNorm(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		} else {
			b.WriteRune(' ')
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

// levenshtein computes the edit distance between two strings (rune-level).
func levenshtein(a, b string) int {
	ar := []rune(a)
	br := []rune(b)
	la, lb := len(ar), len(br)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}

	row := make([]int, lb+1)
	for j := range row {
		row[j] = j
	}

	for i := 1; i <= la; i++ {
		prev := i
		for j := 1; j <= lb; j++ {
			cost := 1
			if ar[i-1] == br[j-1] {
				cost = 0
			}
			curr := minInt(row[j]+1, minInt(prev+1, row[j-1]+cost))
			row[j-1] = prev
			prev = curr
		}
		row[lb] = prev
	}
	return row[lb]
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
