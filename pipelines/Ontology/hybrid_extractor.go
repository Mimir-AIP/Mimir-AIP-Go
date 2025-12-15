package ontology

import (
	"fmt"

	AI "github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/AI"
)

// HybridExtractor combines deterministic and LLM extractors
type HybridExtractor struct {
	config              ExtractionConfig
	deterministicExt    *DeterministicExtractor
	llmExt              *LLMExtractor
	preferDeterministic bool // Whether to prefer deterministic over LLM
}

// NewHybridExtractor creates a new hybrid extractor
func NewHybridExtractor(config ExtractionConfig, llmClient AI.LLMClient) *HybridExtractor {
	return &HybridExtractor{
		config:              config,
		deterministicExt:    NewDeterministicExtractor(config),
		llmExt:              NewLLMExtractor(config, llmClient),
		preferDeterministic: true, // Default to prefer deterministic (faster, cheaper)
	}
}

// Extract performs hybrid extraction
func (e *HybridExtractor) Extract(data any, ontology *OntologyContext) (*ExtractionResult, error) {
	// Determine which extractor to use based on source type
	sourceType := e.config.SourceType

	// For structured data (CSV, JSON), use deterministic first
	if sourceType == SourceTypeCSV || sourceType == SourceTypeJSON {
		result, err := e.deterministicExt.Extract(data, ontology)
		if err == nil && result != nil {
			// If deterministic extraction succeeded, optionally enhance with LLM
			if e.shouldEnhanceWithLLM(result) {
				return e.enhanceWithLLM(result, data, ontology)
			}
			return result, nil
		}

		// If deterministic failed, try LLM as fallback
		if e.llmExt != nil {
			llmResult, llmErr := e.llmExt.Extract(data, ontology)
			if llmErr == nil {
				llmResult.ExtractionType = ExtractionHybrid
				llmResult.Warnings = append(llmResult.Warnings, fmt.Sprintf("Deterministic extraction failed: %v, fell back to LLM", err))
				return llmResult, nil
			}
		}

		return nil, fmt.Errorf("both deterministic and LLM extraction failed: %w", err)
	}

	// For unstructured data (text, HTML), use LLM
	if sourceType == SourceTypeText || sourceType == SourceTypeHTML {
		if e.llmExt != nil {
			result, err := e.llmExt.Extract(data, ontology)
			if err == nil {
				result.ExtractionType = ExtractionHybrid
				return result, nil
			}
			return nil, fmt.Errorf("LLM extraction failed: %w", err)
		}
		return nil, fmt.Errorf("LLM extractor not available for unstructured data")
	}

	return nil, fmt.Errorf("unsupported source type for hybrid extraction: %s", sourceType)
}

// GetType returns the extraction type
func (e *HybridExtractor) GetType() ExtractionType {
	return ExtractionHybrid
}

// GetSupportedSourceTypes returns supported source types
func (e *HybridExtractor) GetSupportedSourceTypes() []string {
	return []string{SourceTypeCSV, SourceTypeJSON, SourceTypeText, SourceTypeHTML}
}

// shouldEnhanceWithLLM determines if LLM enhancement is needed
func (e *HybridExtractor) shouldEnhanceWithLLM(result *ExtractionResult) bool {
	// Only enhance if:
	// 1. LLM is available
	// 2. Few entities were extracted (< 5)
	// 3. Low overall confidence (< 0.7)
	// 4. Many warnings

	if e.llmExt == nil {
		return false
	}

	if result.EntitiesExtracted < 5 {
		return true
	}

	if result.Confidence < 0.7 {
		return true
	}

	if len(result.Warnings) > 3 {
		return true
	}

	return false
}

// enhanceWithLLM uses LLM to enhance deterministic extraction results
func (e *HybridExtractor) enhanceWithLLM(determResult *ExtractionResult, data any, ontology *OntologyContext) (*ExtractionResult, error) {
	// Convert data to string for LLM
	dataStr, ok := data.(string)
	if !ok {
		// Can't enhance with LLM if data isn't string
		return determResult, nil
	}

	// Try LLM extraction
	llmResult, err := e.llmExt.Extract(dataStr, ontology)
	if err != nil {
		// If LLM fails, just return deterministic result with warning
		determResult.Warnings = append(determResult.Warnings, fmt.Sprintf("LLM enhancement failed: %v", err))
		return determResult, nil
	}

	// Merge results
	merged := e.mergeResults(determResult, llmResult)
	merged.ExtractionType = ExtractionHybrid
	merged.Warnings = append(merged.Warnings, "Result combines deterministic extraction and LLM enhancement")

	return merged, nil
}

// mergeResults merges deterministic and LLM extraction results
func (e *HybridExtractor) mergeResults(determ, llm *ExtractionResult) *ExtractionResult {
	merged := &ExtractionResult{
		Entities:       make([]Entity, 0),
		Triples:        make([]Triple, 0),
		ExtractionType: ExtractionHybrid,
		Warnings:       make([]string, 0),
	}

	// Track seen URIs to avoid duplicates
	seenEntityURIs := make(map[string]bool)
	seenTriples := make(map[string]bool)

	// Add deterministic entities (higher priority)
	for _, entity := range determ.Entities {
		if !seenEntityURIs[entity.URI] {
			merged.Entities = append(merged.Entities, entity)
			seenEntityURIs[entity.URI] = true
		}
	}

	// Add LLM entities that don't conflict
	for _, entity := range llm.Entities {
		if !seenEntityURIs[entity.URI] {
			// Lower confidence for LLM entities when merging
			entity.Confidence *= 0.9
			merged.Entities = append(merged.Entities, entity)
			seenEntityURIs[entity.URI] = true
		}
	}

	// Add deterministic triples
	for _, triple := range determ.Triples {
		key := fmt.Sprintf("%s|%s|%s", triple.Subject, triple.Predicate, triple.Object)
		if !seenTriples[key] {
			merged.Triples = append(merged.Triples, triple)
			seenTriples[key] = true
		}
	}

	// Add LLM triples that don't conflict
	for _, triple := range llm.Triples {
		key := fmt.Sprintf("%s|%s|%s", triple.Subject, triple.Predicate, triple.Object)
		if !seenTriples[key] {
			merged.Triples = append(merged.Triples, triple)
			seenTriples[key] = true
		}
	}

	// Calculate merged confidence (weighted average)
	determWeight := 0.7
	llmWeight := 0.3
	merged.Confidence = (determ.Confidence * determWeight) + (llm.Confidence * llmWeight)

	// Update counts
	merged.EntitiesExtracted = len(merged.Entities)
	merged.TriplesGenerated = len(merged.Triples)

	// Merge warnings
	merged.Warnings = append(merged.Warnings, determ.Warnings...)
	merged.Warnings = append(merged.Warnings, llm.Warnings...)

	return merged
}
