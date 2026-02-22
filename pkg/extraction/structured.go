package extraction

import (
	"fmt"
	"strings"

	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

// ExtractFromStructuredCIR extracts entities and relationships from structured CIR data
// Follows the algorithm defined in EntityExtractionPlan.md
func ExtractFromStructuredCIR(cir *models.CIR) (*models.ExtractionResult, error) {
	if cir == nil {
		return nil, fmt.Errorf("CIR cannot be nil")
	}

	// Validate CIR data is a list
	dataList, ok := cir.Data.([]interface{})
	if !ok {
		return nil, fmt.Errorf("CIR data must be an array for structured extraction")
	}

	entities := []models.ExtractedEntity{}
	relationships := []models.ExtractedRelationship{}

	// Extract entities from each data item
	for _, item := range dataList {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue // Skip non-map items
		}

		// Use the first property as name, or 'name' if present
		var nameKey string
		var entityName string

		if name, exists := itemMap["name"]; exists {
			nameKey = "name"
			entityName = normalizeText(fmt.Sprintf("%v", name))
		} else {
			// Use first key as name
			for key, value := range itemMap {
				nameKey = key
				entityName = normalizeText(fmt.Sprintf("%v", value))
				break
			}
		}

		// Create entity
		entity := models.ExtractedEntity{
			Name:       entityName,
			Attributes: make(map[string]interface{}),
			Source:     "structured",
			Confidence: 0.9, // High confidence for structured data
		}

		// Extract attributes from remaining properties
		for key, value := range itemMap {
			if key != nameKey && value != nil {
				attrName := normalizeText(key)
				entity.Attributes[attrName] = value
			}
		}

		entities = append(entities, entity)
	}

	// Infer relationships based on predefined patterns
	relationshipPatterns := models.DefaultRelationshipPatterns

	for _, pattern := range relationshipPatterns {
		for i := range entities {
			entity := &entities[i]
			targetValue, exists := entity.Attributes[pattern.Attribute]
			if !exists {
				continue
			}

			// Find target entity by name
			targetName := normalizeText(fmt.Sprintf("%v", targetValue))
			targetEntity := findEntityByName(entities, targetName)
			if targetEntity != nil {
				relationships = append(relationships, models.ExtractedRelationship{
					Entity1:    entity,
					Entity2:    targetEntity,
					Relation:   pattern.Relation,
					Confidence: pattern.Confidence,
				})
			}
		}
	}

	return &models.ExtractionResult{
		Entities:      entities,
		Relationships: relationships,
		Source:        "structured",
	}, nil
}

// findEntityByName finds an entity by normalized name
func findEntityByName(entities []models.ExtractedEntity, name string) *models.ExtractedEntity {
	for i := range entities {
		if entities[i].Name == name {
			return &entities[i]
		}
	}
	return nil
}

// normalizeText performs basic text normalization
func normalizeText(text string) string {
	// Lowercase, trim, and remove extra spaces
	normalized := strings.ToLower(strings.TrimSpace(text))
	normalized = strings.Join(strings.Fields(normalized), " ")
	return normalized
}
