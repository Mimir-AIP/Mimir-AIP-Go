package extraction

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"github.com/mimir-aip/mimir-aip-go/pkg/storage"
)

// Service handles entity extraction operations
type Service struct {
	storageService *storage.Service
}

// NewService creates a new extraction service
func NewService(storageService *storage.Service) *Service {
	return &Service{
		storageService: storageService,
	}
}

// ExtractFromStorage extracts entities from data in storage
func (s *Service) ExtractFromStorage(projectID string, storageIDs []string, includeStructured, includeUnstructured bool) (*models.ExtractionResult, error) {
	var structuredResult *models.ExtractionResult
	var unstructuredResult *models.ExtractionResult

	// Extract from structured data
	if includeStructured {
		result, err := s.extractStructuredFromStorage(projectID, storageIDs)
		if err != nil {
			return nil, fmt.Errorf("structured extraction failed: %w", err)
		}
		structuredResult = result
	}

	// Extract from unstructured data
	if includeUnstructured {
		result, err := s.extractUnstructuredFromStorage(projectID, storageIDs)
		if err != nil {
			return nil, fmt.Errorf("unstructured extraction failed: %w", err)
		}
		unstructuredResult = result
	}

	// Reconcile results
	reconciledResult := ReconcileEntities(structuredResult, unstructuredResult)

	return reconciledResult, nil
}

// extractStructuredFromStorage retrieves CIR data from storage and extracts structured entities
func (s *Service) extractStructuredFromStorage(projectID string, storageIDs []string) (*models.ExtractionResult, error) {
	allEntities := []models.ExtractedEntity{}
	allRelationships := []models.ExtractedRelationship{}

	for _, storageID := range storageIDs {
		// Retrieve all CIR data from this storage
		query := &models.CIRQuery{
			Limit: 1000, // Process in batches
		}

		cirItems, err := s.storageService.Retrieve(storageID, query)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve data from storage %s: %w", storageID, err)
		}

		// Extract entities from each CIR item
		for _, cir := range cirItems {
			result, err := ExtractFromStructuredCIR(cir)
			if err != nil {
				// Log error but continue processing
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

// extractUnstructuredFromStorage retrieves CIR data and extracts unstructured entities using
// heuristic text analysis.
// TODO: Replace with ML-quality NLP extraction; eventually integrate a trained model for entity recognition
func (s *Service) extractUnstructuredFromStorage(projectID string, storageIDs []string) (*models.ExtractionResult, error) {
	// Field name prefixes that commonly identify entity-bearing values
	entityFieldPrefixes := []string{"name", "person", "company", "organization", "location", "email", "phone"}

	// Regex for "Key: Value" patterns (e.g. "Name: Alice", "Company: Acme Corp")
	kvPattern := regexp.MustCompile(`(?i)(\w+):\s+([^\n,;]{1,80})`)
	// Regex for sequences of capitalized words (candidate entity names)
	capitalPattern := regexp.MustCompile(`\b([A-Z][a-z]+(?:\s+[A-Z][a-z]+)*)\b`)

	seen := make(map[string]bool)
	allEntities := []models.ExtractedEntity{}

	for _, storageID := range storageIDs {
		cirItems, err := s.storageService.Retrieve(storageID, &models.CIRQuery{Limit: 1000})
		if err != nil {
			continue
		}

		for _, cir := range cirItems {
			dataMap, ok := cir.Data.(map[string]interface{})
			if !ok {
				continue
			}

			for fieldKey, fieldVal := range dataMap {
				strVal, ok := fieldVal.(string)
				if !ok || strVal == "" {
					continue
				}

				normalizedField := strings.ToLower(fieldKey)

				if len(strVal) < 50 {
					// Short string: treat as a candidate key-value attribute
					conf := 0.7
					for _, prefix := range entityFieldPrefixes {
						if strings.HasPrefix(normalizedField, prefix) {
							conf = 0.8
							break
						}
					}
					if !seen[strVal] {
						seen[strVal] = true
						allEntities = append(allEntities, models.ExtractedEntity{
							Name:       strVal,
							Attributes: map[string]interface{}{fieldKey: strVal},
							Source:     "unstructured",
							Confidence: conf,
						})
					}
				} else {
					// Long string: apply regex patterns to find named entities

					// "Key: Value" patterns
					for _, match := range kvPattern.FindAllStringSubmatch(strVal, -1) {
						if len(match) == 3 {
							entityName := strings.TrimSpace(match[2])
							if entityName != "" && !seen[entityName] {
								seen[entityName] = true
								allEntities = append(allEntities, models.ExtractedEntity{
									Name:       entityName,
									Attributes: map[string]interface{}{match[1]: entityName},
									Source:     "unstructured",
									Confidence: 0.75,
								})
							}
						}
					}

					// Capitalized word sequences
					for _, match := range capitalPattern.FindAllString(strVal, -1) {
						entityName := strings.TrimSpace(match)
						if entityName != "" && !seen[entityName] {
							seen[entityName] = true
							allEntities = append(allEntities, models.ExtractedEntity{
								Name:       entityName,
								Attributes: map[string]interface{}{"field": fieldKey},
								Source:     "unstructured",
								Confidence: 0.5,
							})
						}
					}
				}
			}
		}
	}

	return &models.ExtractionResult{
		Entities:      allEntities,
		Relationships: []models.ExtractedRelationship{},
		Source:        "unstructured",
	}, nil
}
