package extraction

import (
	"context"
	"fmt"
	"github.com/mimir-aip/mimir-aip-go/pkg/llm"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"github.com/mimir-aip/mimir-aip-go/pkg/storage"
	"strings"
	"time"
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
		cirs, err := s.retrieveAllFromStorage(projectID, storageID)
		if err != nil {
			return nil, fmt.Errorf("retrieve extraction data from storage %s: %w", storageID, err)
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

const extractionRetrievePageSize = 1000

func (s *Service) retrieveAllFromStorage(projectID, storageID string) ([]*models.CIR, error) {
	var all []*models.CIR
	for offset := 0; ; {
		page, err := s.storageService.RetrieveForProject(projectID, storageID, &models.CIRQuery{Limit: extractionRetrievePageSize, Offset: offset})
		if err != nil {
			return nil, err
		}
		all = append(all, page...)
		if len(page) < extractionRetrievePageSize {
			return all, nil
		}
		offset += len(page)
	}
}

// ─── Structured path ──────────────────────────────────────────────────────────

// extractStructuredFromStorageWithCIRs runs structured extraction using
// pre-fetched CIR data (avoiding a second round of storage calls).
func (s *Service) extractStructuredFromStorageWithCIRs(_ string, storageIDs []string, cirsByStorage map[string][]*models.CIR) (*models.ExtractionResult, error) {
	allEntities := []models.ExtractedEntity{}
	allRelationships := []models.ExtractedRelationship{}

	for _, storageID := range storageIDs {
		cirItems := cirsByStorage[storageID]
		if tabularCIR := tabularCIRFromRetrievedRows(cirItems); tabularCIR != nil {
			if tabResult, ok := ExtractSchemaFromTabularCIR(tabularCIR); ok {
				allEntities = append(allEntities, tabResult.Entities...)
				allRelationships = append(allRelationships, tabResult.Relationships...)
				continue
			}
		}
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

func tabularCIRFromRetrievedRows(cirs []*models.CIR) *models.CIR {
	rows := make([]interface{}, 0, len(cirs))
	var template *models.CIR
	for _, cir := range cirs {
		if cir == nil {
			continue
		}
		row, ok := cir.Data.(map[string]interface{})
		if !ok {
			return nil
		}
		if template == nil {
			template = cir
		}
		rows = append(rows, row)
	}
	if template == nil || len(rows) == 0 {
		return nil
	}
	tabular := models.NewCIR(template.Source.Type, template.Source.URI, template.Source.Format, rows)
	tabular.Metadata = template.Metadata
	return tabular
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
