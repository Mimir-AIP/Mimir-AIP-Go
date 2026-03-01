package extraction

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/mimir-aip/mimir-aip-go/pkg/llm"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"github.com/mimir-aip/mimir-aip-go/pkg/storage"
)

// ─── Tuneable parameters ──────────────────────────────────────────────────────

const (
	maxNgramLen    = 4    // maximum n-gram size (words)
	minTokenChars  = 2    // minimum character length of a single-word token
	minEntityScore = 0.40 // minimum composite score to treat a token as an entity
	minRelNPMI     = 0.20 // minimum normalised PMI to emit a relationship
	maxEntities    = 500  // safety cap on emitted entities per extraction run
)

// ─── Service ──────────────────────────────────────────────────────────────────

// Service handles entity extraction operations
type Service struct {
	storageService *storage.Service
	llm            *llm.Service // nil = LLM disabled; always check IsEnabled()
}

// NewService creates a new extraction service
func NewService(storageService *storage.Service) *Service {
	return &Service{storageService: storageService}
}

// WithLLM returns a copy of the Service with the LLM service attached.
func (s *Service) WithLLM(llmSvc *llm.Service) *Service {
	return &Service{storageService: s.storageService, llm: llmSvc}
}

// ExtractFromStorage extracts entities from data in storage.
//
// When multiple storage sources are provided the function also performs
// cross-source link detection: it compares the statistical profile of each
// column across all storage sources and reports pairs that are likely to be
// foreign-key-style join points (e.g. student_id appearing in both a grades DB
// and an attendance DB).  No domain configuration is required.
func (s *Service) ExtractFromStorage(projectID string, storageIDs []string, includeStructured, includeUnstructured bool) (*models.ExtractionResult, error) {
	var structuredResult *models.ExtractionResult
	var unstructuredResult *models.ExtractionResult

	// Collect column profiles per storage for cross-source link detection.
	// We always retrieve CIR data here regardless of includeStructured so that
	// cross-source links are detected even when only unstructured is requested.
	var allProfiles []models.ColumnProfile
	cirsByStorage := make(map[string][]*models.CIR, len(storageIDs))

	for _, storageID := range storageIDs {
		cirs, err := s.storageService.Retrieve(storageID, &models.CIRQuery{Limit: 1000})
		if err != nil {
			// Non-fatal: log and continue so one bad source doesn't abort everything.
			fmt.Printf("Warning: cross-source profiling: failed to retrieve from %s: %v\n", storageID, err)
			continue
		}
		cirsByStorage[storageID] = cirs
		profiles := BuildColumnProfilesFromCIRs(storageID, cirs)
		allProfiles = append(allProfiles, profiles...)
	}

	if includeStructured {
		result, err := s.extractStructuredFromStorageWithCIRs(projectID, storageIDs, cirsByStorage)
		if err != nil {
			return nil, fmt.Errorf("structured extraction failed: %w", err)
		}
		structuredResult = result
	}

	if includeUnstructured {
		result, err := s.extractUnstructuredFromStorageWithCIRs(projectID, cirsByStorage)
		if err != nil {
			return nil, fmt.Errorf("unstructured extraction failed: %w", err)
		}
		unstructuredResult = result
	}

	result := ReconcileEntities(structuredResult, unstructuredResult)

	// Detect cross-source links when more than one storage source contributed data.
	if len(storageIDs) > 1 {
		result.CrossSourceLinks = DetectCrossSourceLinks(allProfiles)
	}

	return result, nil
}

// ─── Structured path ──────────────────────────────────────────────────────────

// extractStructuredFromStorageWithCIRs runs structured extraction using
// pre-fetched CIR data (avoiding a second round of storage calls).
func (s *Service) extractStructuredFromStorageWithCIRs(_ string, storageIDs []string, cirsByStorage map[string][]*models.CIR) (*models.ExtractionResult, error) {
	allEntities := []models.ExtractedEntity{}
	allRelationships := []models.ExtractedRelationship{}

	for _, storageID := range storageIDs {
		cirItems := cirsByStorage[storageID]
		for _, cir := range cirItems {
			// Try row-entity extraction first for structured record tables.
			// This produces one entity per row with the entity type inferred
			// from the key column name (e.g. "student_id" → type "Student"),
			// and preserves numeric/boolean column values as typed attributes.
			// This is more accurate than column-value extraction for DB tables.
			if tabResult, ok := ExtractSchemaFromTabularCIR(cir); ok {
				allEntities = append(allEntities, tabResult.Entities...)
				allRelationships = append(allRelationships, tabResult.Relationships...)
				continue
			}

			// Fall back to column-value entity extraction for non-record CIRs.
			result, err := ExtractFromStructuredCIR(cir)
			if err != nil {
				continue
			}
			if result != nil {
				allEntities = append(allEntities, result.Entities...)
				allRelationships = append(allRelationships, result.Relationships...)
			}
		}
	}

	return &models.ExtractionResult{
		Entities:      allEntities,
		Relationships: allRelationships,
		Source:        "structured",
	}, nil
}

// ─── Unstructured path: corpus-level statistical extraction ──────────────────
//
// The algorithm is entirely data-agnostic: it discovers entities by their
// statistical properties across the corpus rather than by matching
// domain-specific patterns.
//
// NLP pre-processing (pkg/extraction/nlp.go):
//   - Sentence boundary detection (prevents cross-sentence n-grams).
//   - Stopword-boundary filtering (multi-word n-grams whose first or last
//     word is a common function word are discarded early).
//   - BM25 IDF for rarity scoring (non-zero signal even for universal terms).
//   - Phrase cohesion via minimum pairwise PMI (filters accidental word
//     sequences that never co-occur in other fields).
//   - Morphology boost for ALL_CAPS abbreviations and CamelCase names.
//   - Fuzzy deduplication with Levenshtein edit distance.
//
// Six scoring dimensions — all derived from the data itself:
//
//  1. Rarity (BM25 IDF): tokens that appear in some records but not all
//     are more specific and more likely to be named entities.
//
//  2. Capitalization consistency: a token that is capitalised in the same
//     way across every occurrence is more likely a proper noun.
//
//  3. Phrase length: 2-3 word phrases outperform single words (less
//     ambiguous) and 4-word phrases (often too broad).
//
//  4. Value completeness: a token that is the entire value of a field is
//     more likely a standalone named entity than a fragment of longer text.
//
//  5. Field cardinality bonus: tokens that appear in very few distinct
//     field keys are more "focused" and score higher.
//
//  6. Phrase cohesion: minimum pairwise PMI over consecutive word pairs
//     rewards phrases whose words strongly attract each other.
//
// Relationships are discovered via Normalised PMI (NPMI) on pairs of
// entity candidates that co-occur in the same records.

// extractUnstructuredFromStorageWithCIRs runs unstructured (NLP) extraction
// using pre-fetched CIR data.
func (s *Service) extractUnstructuredFromStorageWithCIRs(_ string, cirsByStorage map[string][]*models.CIR) (*models.ExtractionResult, error) {
	var records []extractionRecord
	for _, cirItems := range cirsByStorage {
		for _, cir := range cirItems {
			// cirToRows expands array-format CIR data into per-row records so
			// that corpus statistics (IDF, NPMI) are computed at row granularity
			// rather than collapsing an entire table into one document.
			records = append(records, cirToRows(cir)...)
		}
	}
	result := extractFromRecords(records)

	if s.llm != nil && s.llm.IsEnabled() && len(result.Entities) > 0 {
		s.applyLLMEntityLabels(result, cirsByStorage)
	}
	return result, nil
}

// applyLLMEntityLabels calls the LLM to assign entity types to unstructured
// entities that do not yet have an entity_type attribute.
func (s *Service) applyLLMEntityLabels(result *models.ExtractionResult, cirsByStorage map[string][]*models.CIR) {
	var names []string
	for _, e := range result.Entities {
		if e.Source == "unstructured" {
			if _, hasType := e.Attributes["entity_type"]; !hasType {
				names = append(names, e.Name)
			}
		}
	}
	if len(names) == 0 {
		return
	}

	var storageIDs []string
	colSet := make(map[string]bool)
	for sid, cirs := range cirsByStorage {
		storageIDs = append(storageIDs, sid)
		for _, cir := range cirs {
			if m, ok := cir.Data.(map[string]interface{}); ok {
				for k := range m {
					colSet[k] = true
				}
			}
		}
	}
	contextCols := make([]string, 0, len(colSet))
	for k := range colSet {
		contextCols = append(contextCols, k)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	labels := s.llm.LabelEntityTypes(ctx, names, strings.Join(storageIDs, ", "), contextCols)

	for i := range result.Entities {
		if result.Entities[i].Source != "unstructured" {
			continue
		}
		if label, ok := labels[result.Entities[i].Name]; ok && label != "" {
			if result.Entities[i].Attributes == nil {
				result.Entities[i].Attributes = make(map[string]interface{})
			}
			result.Entities[i].Attributes["entity_type"] = label
		}
	}
}

// extractFromRecords is the core statistical algorithm, separated from the
// storage layer so it can be called directly in tests.
func extractFromRecords(records []extractionRecord) *models.ExtractionResult {
	if len(records) == 0 {
		return &models.ExtractionResult{Source: "unstructured"}
	}

	// Phase 1 — build corpus index
	idx := newCorpusIndex()
	for _, rec := range records {
		idx.addRecord(rec)
	}

	// Phase 2 — score every n-gram that survived the basic filter
	candidates := make(map[string]float64, len(idx.docFreq))
	for t := range idx.docFreq {
		if isFiltered(t) {
			continue
		}
		if sc := idx.entityScore(t); sc >= minEntityScore {
			candidates[t] = sc
		}
	}

	// Phase 3 — subsumption pruning: demote shorter n-grams that are always
	// contained within a higher-scoring, longer n-gram.
	candidates = subsumptionPrune(candidates, idx)

	// Phase 3b — fuzzy deduplication: merge near-identical surface forms
	// (e.g. "Acme Corp." ≈ "Acme Corp", "St. Mary's" ≈ "St Marys") keeping
	// only the highest-scoring variant in each duplicate group.
	candidates = fuzzyDeduplicate(candidates)

	// Phase 4 — build co-occurrence matrix (only for surviving candidates to
	// keep memory bounded).
	idx.buildCoOccurrence(candidates)

	// Phase 5 — emit entities and relationships
	entities := candidatesToEntities(candidates, idx)
	relationships := computeRelationships(entities, idx)

	return &models.ExtractionResult{
		Entities:      entities,
		Relationships: relationships,
		Source:        "unstructured",
	}
}

// ─── Data structures ─────────────────────────────────────────────────────────

// extractionRecord holds the tokenised content of a single CIR record.
type extractionRecord struct {
	ngrams []ngramOcc
}

// ngramOcc captures one n-gram occurrence in a field.
type ngramOcc struct {
	text      string // the n-gram text (preserves original casing)
	fieldKey  string // field the token came from
	position  int    // word offset within the field value (0 = start)
	capFirst  bool   // true when the first rune is uppercase
	isFullVal bool   // true when this n-gram equals the entire trimmed field value
}

// corpusIndex accumulates statistics across all ingested records.
type corpusIndex struct {
	N        int                         // total record count
	docFreq  map[string]int              // n-gram → # distinct records
	termFreq map[string]int              // n-gram → total occurrence count
	capCount map[string]int              // n-gram → # occurrences where capFirst
	fullVal  map[string]int              // n-gram → # times it was the full field value
	ngramLen map[string]int              // n-gram → word count
	fields   map[string]map[string]bool  // n-gram → set of field keys
	coOcc    map[string]map[string]int   // n-gram → n-gram → # co-occurring records
	perDoc   []map[string]bool           // per-record n-gram sets (used for coOcc build)
}

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
// function words, and strings that are almost entirely punctuation.
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
// sorted by score descending, capped at maxEntities.
func candidatesToEntities(candidates map[string]float64, idx *corpusIndex) []models.ExtractedEntity {
	type scored struct {
		text  string
		score float64
	}
	sorted := make([]scored, 0, len(candidates))
	for t, sc := range candidates {
		sorted = append(sorted, scored{t, sc})
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].score > sorted[j].score
	})

	if len(sorted) > maxEntities {
		sorted = sorted[:maxEntities]
	}

	entities := make([]models.ExtractedEntity, 0, len(sorted))
	for _, s := range sorted {
		attrs := map[string]interface{}{
			"doc_frequency":      idx.docFreq[s.text],
			"total_occurrences":  idx.termFreq[s.text],
			"cap_consistency":    math.Round(float64(idx.capCount[s.text])/float64(idx.termFreq[s.text])*100) / 100,
		}
		// Record which field keys this entity appeared under.
		fieldKeys := make([]string, 0, len(idx.fields[s.text]))
		for fk := range idx.fields[s.text] {
			fieldKeys = append(fieldKeys, fk)
		}
		sort.Strings(fieldKeys)
		if len(fieldKeys) > 0 {
			attrs["fields"] = fieldKeys
		}

		entities = append(entities, models.ExtractedEntity{
			Name:       s.text,
			Attributes: attrs,
			Source:     "unstructured",
			Confidence: math.Min(s.score, 0.95),
		})
	}
	return entities
}

// ─── Relationship emission ────────────────────────────────────────────────────

// computeRelationships emits relationships for entity pairs whose normalised
// PMI exceeds minRelNPMI.
//
// Normalised PMI (NPMI) is defined as:
//
//	NPMI(a, b) = PMI(a, b) / −log P(a, b)
//	           = log(P(a,b) / P(a)·P(b)) / −log P(a,b)
//
// It lies in [−1, +1]; values near +1 indicate near-perfect co-occurrence.
// The relationship label is derived from the intersection of the field keys
// that each entity appears under.
func computeRelationships(entities []models.ExtractedEntity, idx *corpusIndex) []models.ExtractedRelationship {
	if idx.N == 0 || len(entities) < 2 {
		return nil
	}

	N := float64(idx.N)
	var rels []models.ExtractedRelationship

	for i := 0; i < len(entities); i++ {
		for j := i + 1; j < len(entities); j++ {
			a := entities[i].Name
			b := entities[j].Name

			cooccur, ok := idx.coOcc[a][b]
			if !ok || cooccur == 0 {
				continue
			}

			pA := float64(idx.docFreq[a]) / N
			pB := float64(idx.docFreq[b]) / N
			pAB := float64(cooccur) / N

			if pA == 0 || pB == 0 || pAB == 0 {
				continue
			}

			pmi := math.Log(pAB / (pA * pB))
			npmi := pmi / (-math.Log(pAB))

			if npmi < minRelNPMI {
				continue
			}

			relLabel := deriveRelation(a, b, idx)
			conf := math.Min((npmi+1)/2*math.Min(entities[i].Confidence, entities[j].Confidence), 0.95)

			e1 := &models.ExtractedEntity{Name: a, Source: "unstructured"}
			e2 := &models.ExtractedEntity{Name: b, Source: "unstructured"}
			rels = append(rels, models.ExtractedRelationship{
				Entity1:    e1,
				Entity2:    e2,
				Relation:   relLabel,
				Confidence: conf,
			})
		}
	}
	return rels
}

// deriveRelation produces a human-readable relationship label.
// If both entities appear under the same field key, that field is used.
// If they appear under different field keys, we combine the field names.
// Otherwise, we fall back to "co_occurs".
func deriveRelation(a, b string, idx *corpusIndex) string {
	aFields := idx.fields[a]
	bFields := idx.fields[b]

	// Shared field keys.
	var shared []string
	for f := range aFields {
		if bFields[f] {
			shared = append(shared, f)
		}
	}
	if len(shared) > 0 {
		sort.Strings(shared)
		return "co_occurs_in_" + strings.Join(shared, "_")
	}

	// Different field keys — combine them to express a cross-field relation.
	aList := make([]string, 0, len(aFields))
	bList := make([]string, 0, len(bFields))
	for f := range aFields {
		aList = append(aList, f)
	}
	for f := range bFields {
		bList = append(bList, f)
	}
	sort.Strings(aList)
	sort.Strings(bList)
	if len(aList) > 0 && len(bList) > 0 {
		return normaliseFieldKey(aList[0]) + "_to_" + normaliseFieldKey(bList[0])
	}

	return "co_occurs"
}

// normaliseFieldKey converts a field key to a snake_case relation label.
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
