package extraction

import (
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"math"
	"sort"
	"strings"
)

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
			"doc_frequency":     idx.docFreq[s.text],
			"total_occurrences": idx.termFreq[s.text],
			"cap_consistency":   math.Round(float64(idx.capCount[s.text])/float64(idx.termFreq[s.text])*100) / 100,
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
