package extraction

import (
	"fmt"

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

// extractUnstructuredFromStorage retrieves CIR data and extracts unstructured entities
// Note: Unstructured extraction is not yet implemented, returning empty result
func (s *Service) extractUnstructuredFromStorage(projectID string, storageIDs []string) (*models.ExtractionResult, error) {
	// TODO: Implement unstructured extraction when needed
	// For now, return empty result
	return &models.ExtractionResult{
		Entities:      []models.ExtractedEntity{},
		Relationships: []models.ExtractedRelationship{},
		Source:        "unstructured",
	}, nil
}
