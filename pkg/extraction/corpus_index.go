package extraction

func newCorpusIndex() *corpusIndex {
	return &corpusIndex{
		docFreq:  make(map[string]int),
		termFreq: make(map[string]int),
		capCount: make(map[string]int),
		fullVal:  make(map[string]int),
		ngramLen: make(map[string]int),
		fields:   make(map[string]map[string]bool),
		coOcc:    make(map[string]map[string]int),
	}
}

func (idx *corpusIndex) addRecord(rec extractionRecord) {
	idx.N++
	inDoc := make(map[string]bool, len(rec.ngrams))

	for _, occ := range rec.ngrams {
		t := occ.text
		idx.termFreq[t]++
		if occ.capFirst {
			idx.capCount[t]++
		}
		if occ.isFullVal {
			idx.fullVal[t]++
		}
		if _, set := idx.ngramLen[t]; !set {
			idx.ngramLen[t] = wordCount(t)
		}
		if idx.fields[t] == nil {
			idx.fields[t] = make(map[string]bool)
		}
		idx.fields[t][occ.fieldKey] = true
		inDoc[t] = true
	}

	for t := range inDoc {
		idx.docFreq[t]++
	}
	idx.perDoc = append(idx.perDoc, inDoc)
}

// buildCoOccurrence fills idx.coOcc for the surviving candidate set.
func (idx *corpusIndex) buildCoOccurrence(candidates map[string]float64) {
	for _, docSet := range idx.perDoc {
		// Collect which candidates appear in this record.
		var present []string
		for t := range docSet {
			if _, ok := candidates[t]; ok {
				present = append(present, t)
			}
		}
		for i := 0; i < len(present); i++ {
			for j := i + 1; j < len(present); j++ {
				a, b := present[i], present[j]
				if idx.coOcc[a] == nil {
					idx.coOcc[a] = make(map[string]int)
				}
				if idx.coOcc[b] == nil {
					idx.coOcc[b] = make(map[string]int)
				}
				idx.coOcc[a][b]++
				idx.coOcc[b][a]++
			}
		}
	}
}

// ─── Scoring ─────────────────────────────────────────────────────────────────

// entityScore computes the composite salience score for a single n-gram.
// All sub-scores are in [0, 1] and the final combined score is in [0, 1].
func (idx *corpusIndex) entityScore(t string) float64 {
	df := idx.docFreq[t]
	tf := idx.termFreq[t]
	if df == 0 || tf == 0 {
		return 0
	}
	N := idx.N

	// 1. Rarity — BM25 IDF normalised to [0, 1].
	//    Unlike plain IDF, BM25 IDF gives a non-zero floor for terms that
	//    appear in every document (important for focused corpora where the
	//    subject entity is referenced in all records).
	//    A mild penalty for single-occurrence tokens reduces noise.
	rarityRaw := bm25IDFScore(N, df)
	if df == 1 {
		rarityRaw *= 0.60
	}

	// 2. Capitalization consistency — fraction of occurrences with uppercase first rune.
	capScore := float64(idx.capCount[t]) / float64(tf)

	// 3. Phrase-length preference — peaks at 2-3 words.
	var lengthScore float64
	switch idx.ngramLen[t] {
	case 1:
		lengthScore = 0.50
	case 2:
		lengthScore = 0.85
	case 3:
		lengthScore = 0.90
	case 4:
		lengthScore = 0.70
	default:
		lengthScore = 0.35
	}

	// 4. Value completeness — how often this n-gram is the entire field value.
	// For single-word tokens we scale by capitalisation consistency: an
	// all-lowercase complete field value (e.g. "true", "false", "active") is
	// almost certainly a boolean or generic flag rather than a named entity.
	rawValueScore := float64(idx.fullVal[t]) / float64(tf)
	valueScore := rawValueScore
	if idx.ngramLen[t] == 1 {
		valueScore *= capScore
	}

	// 5. Field focus — bonus when the token appears in very few distinct field keys,
	//    indicating it is specific to a semantic slot rather than scattered noise.
	fieldCount := len(idx.fields[t])
	var focusScore float64
	switch {
	case fieldCount == 1:
		focusScore = 1.0
	case fieldCount <= 3:
		focusScore = 0.7
	case fieldCount <= 6:
		focusScore = 0.4
	default:
		focusScore = 0.1
	}

	// 6. Phrase cohesion — minimum pairwise PMI over consecutive word pairs.
	//    High values indicate the words in a multi-word n-gram genuinely
	//    attract each other across the corpus rather than appearing together
	//    by coincidence.  Single-word tokens and phrases containing internal
	//    stopwords receive a neutral 0.5 score (PMI is confounded in those
	//    cases and adding signal would introduce noise).
	cohesionScore := phraseCohesion(t, idx)

	// Weighted combination (weights sum to 1.0).
	score := 0.25*rarityRaw +
		0.28*capScore +
		0.12*lengthScore +
		0.13*valueScore +
		0.10*focusScore +
		0.12*cohesionScore

	// Additive morphology bonus for surface-form cues that signal proper nouns
	// without requiring corpus signal: ALL_CAPS abbreviations (CEO, API),
	// CamelCase brand names (ThinkPad, LinkedIn), hyphenated compounds.
	score += morphologyBoost(t)

	return score
}

// ─── Filtering ────────────────────────────────────────────────────────────────

// isFiltered returns true for tokens that should never be entity candidates:
// pure numbers, single-character tokens, whitespace-only strings, very common
