package extraction

import (
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

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
	N        int                        // total record count
	docFreq  map[string]int             // n-gram → # distinct records
	termFreq map[string]int             // n-gram → total occurrence count
	capCount map[string]int             // n-gram → # occurrences where capFirst
	fullVal  map[string]int             // n-gram → # times it was the full field value
	ngramLen map[string]int             // n-gram → word count
	fields   map[string]map[string]bool // n-gram → set of field keys
	coOcc    map[string]map[string]int  // n-gram → n-gram → # co-occurring records
	perDoc   []map[string]bool          // per-record n-gram sets (used for coOcc build)
}
