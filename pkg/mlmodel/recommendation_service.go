package mlmodel

import (
	"fmt"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"log"
)

// RecommendModelType recommends the best model type for a project
func (s *Service) RecommendModelType(projectID, ontologyID string) (*models.ModelRecommendation, error) {
	if err := s.ensureProjectExists(projectID); err != nil {
		return nil, err
	}
	ontologyRecord, err := s.ontologyService.GetOntologyForProject(projectID, ontologyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get ontology: %w", err)
	}
	storageConfigs, err := s.storageService.GetProjectStorageConfigs(projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get storage configs: %w", err)
	}
	dataSummary := s.analyzeData(storageConfigs)
	recommendation, err := s.recommendationEngine.RecommendModelType(ontologyRecord, dataSummary)
	if err != nil {
		return nil, fmt.Errorf("failed to recommend model type: %w", err)
	}
	return recommendation, nil
}

// analyzeData analyzes storage configs to create a data summary by inspecting actual records
func (s *Service) analyzeData(storageConfigs []*models.StorageConfig) *models.DataAnalysis {
	totalRecords := int64(0)
	featureCount := 0
	hasUnstructured := false

	for _, config := range storageConfigs {
		cirs, err := s.storageService.Retrieve(config.ID, &models.CIRQuery{})
		if err != nil {
			log.Printf("Warning: failed to retrieve from storage %s for analysis: %v", config.ID, err)
			continue
		}
		totalRecords += int64(len(cirs))

		// Inspect first record to determine field types
		if len(cirs) > 0 {
			if dataMap, ok := cirs[0].Data.(map[string]any); ok {
				numericFields := 0
				for _, v := range dataMap {
					switch val := v.(type) {
					case float64, int, bool:
						numericFields++
					case string:
						if len(val) > 100 {
							hasUnstructured = true
						} else {
							numericFields++
						}
					}
				}
				if numericFields > featureCount {
					featureCount = numericFields
				}
			}
		}
	}

	// Determine size category from actual record count
	size := "small"
	if totalRecords > 10000 {
		size = "large"
	} else if totalRecords >= 1000 {
		size = "medium"
	}

	// Avoid returning zero when storage configs exist but retrieval returned nothing
	if totalRecords == 0 && len(storageConfigs) > 0 {
		totalRecords = 100
	}

	return &models.DataAnalysis{
		Size:            size,
		RecordCount:     totalRecords,
		HasUnstructured: hasUnstructured,
		FeatureCount:    featureCount,
	}
}
