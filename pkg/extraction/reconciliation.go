package extraction

import (
	"strings"

	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

// ReconcileEntities merges entities from structured and unstructured sources
// Follows the algorithm defined in EntityExtractionPlan.md
func ReconcileEntities(structuredResults, unstructuredResults *models.ExtractionResult) *models.ExtractionResult {
	// Combine all extracted data
	allEntities := []models.ExtractedEntity{}
	allRelationships := []models.ExtractedRelationship{}
	allAttributes := []models.ExtractedAttribute{}

	if structuredResults != nil {
		allEntities = append(allEntities, structuredResults.Entities...)
		allRelationships = append(allRelationships, structuredResults.Relationships...)
		if structuredResults.Attributes != nil {
			allAttributes = append(allAttributes, structuredResults.Attributes...)
		}
	}

	if unstructuredResults != nil {
		allEntities = append(allEntities, unstructuredResults.Entities...)
		allRelationships = append(allRelationships, unstructuredResults.Relationships...)
		if unstructuredResults.Attributes != nil {
			allAttributes = append(allAttributes, unstructuredResults.Attributes...)
		}
	}

	// Step 1: Group entities by normalized name
	entityGroups := groupEntitiesByNormalizedName(allEntities)

	// Step 2: Merge entity groups
	reconciledEntities := []models.ExtractedEntity{}
	entityMapping := make(map[*models.ExtractedEntity]*models.ExtractedEntity)

	for _, group := range entityGroups {
		mergedEntity := mergeEntityGroup(group)
		reconciledEntities = append(reconciledEntities, mergedEntity)

		// Map each original entity to the merged entity
		for i := range group {
			entityMapping[&group[i]] = &reconciledEntities[len(reconciledEntities)-1]
		}
	}

	// Step 3: Reconcile relationships
	reconciledRelationships := []models.ExtractedRelationship{}
	seenRelationships := make(map[string]bool)

	for _, relationship := range allRelationships {
		// Map relationship entities to reconciled entities
		reconciledEntity1, ok1 := entityMapping[relationship.Entity1]
		if !ok1 {
			reconciledEntity1 = relationship.Entity1
		}

		reconciledEntity2, ok2 := entityMapping[relationship.Entity2]
		if !ok2 {
			reconciledEntity2 = relationship.Entity2
		}

		// Create a unique key for deduplication
		relKey := reconciledEntity1.Name + "|" + reconciledEntity2.Name + "|" + relationship.Relation

		if !seenRelationships[relKey] {
			reconciledRelationships = append(reconciledRelationships, models.ExtractedRelationship{
				Entity1:    reconciledEntity1,
				Entity2:    reconciledEntity2,
				Relation:   relationship.Relation,
				Confidence: relationship.Confidence,
			})
			seenRelationships[relKey] = true
		}
	}

	// Step 4: Assign attributes to reconciled entities
	for _, attribute := range allAttributes {
		reconciledEntity, ok := entityMapping[attribute.Entity]
		if ok {
			attrName := attribute.Attribute
			// Only add if not already present to avoid conflicts
			if reconciledEntity.Attributes == nil {
				reconciledEntity.Attributes = make(map[string]interface{})
			}
			if _, exists := reconciledEntity.Attributes[attrName]; !exists {
				reconciledEntity.Attributes[attrName] = attribute
			}
		}
	}

	return &models.ExtractionResult{
		Entities:      reconciledEntities,
		Relationships: reconciledRelationships,
		Source:        "reconciled",
	}
}

// groupEntitiesByNormalizedName groups entities by normalized name
func groupEntitiesByNormalizedName(entities []models.ExtractedEntity) [][]models.ExtractedEntity {
	groups := make(map[string][]models.ExtractedEntity)

	for _, entity := range entities {
		normalizedName := normalizeEntityName(entity.Name)
		groups[normalizedName] = append(groups[normalizedName], entity)
	}

	// Convert map to slice
	result := make([][]models.ExtractedEntity, 0, len(groups))
	for _, group := range groups {
		result = append(result, group)
	}

	return result
}

// mergeEntityGroup merges a group of entities into a single entity
func mergeEntityGroup(entityGroup []models.ExtractedEntity) models.ExtractedEntity {
	if len(entityGroup) == 1 {
		return entityGroup[0]
	}

	// Select primary entity (highest confidence, prefer structured data)
	primaryIdx := 0
	for i := 1; i < len(entityGroup); i++ {
		current := entityGroup[i]
		primary := entityGroup[primaryIdx]

		// Compare confidence first, then prefer structured source
		if current.Confidence > primary.Confidence {
			primaryIdx = i
		} else if current.Confidence == primary.Confidence && current.Source == "structured" && primary.Source != "structured" {
			primaryIdx = i
		}
	}

	primaryEntity := entityGroup[primaryIdx]

	// Merge attributes
	mergedAttributes := make(map[string]interface{})
	for _, entity := range entityGroup {
		if entity.Attributes == nil {
			continue
		}
		for attrName, attrValue := range entity.Attributes {
			if _, exists := mergedAttributes[attrName]; !exists {
				mergedAttributes[attrName] = attrValue
			} else if mergedAttributes[attrName] == nil && attrValue != nil {
				// Prefer non-empty values
				mergedAttributes[attrName] = attrValue
			}
			// If both have values and differ, keep the primary entity's value
		}
	}

	// Calculate average confidence
	totalConfidence := 0.0
	for _, entity := range entityGroup {
		totalConfidence += entity.Confidence
	}
	avgConfidence := totalConfidence / float64(len(entityGroup))

	// Collect unique sources
	sourcesMap := make(map[string]bool)
	for _, entity := range entityGroup {
		sourcesMap[entity.Source] = true
		if entity.Sources != nil {
			for _, src := range entity.Sources {
				sourcesMap[src] = true
			}
		}
	}

	sources := make([]string, 0, len(sourcesMap))
	for src := range sourcesMap {
		sources = append(sources, src)
	}

	// Create merged entity
	return models.ExtractedEntity{
		Name:       primaryEntity.Name,
		Attributes: mergedAttributes,
		Confidence: avgConfidence,
		Source:     "reconciled",
		Sources:    sources,
	}
}

// normalizeEntityName performs comprehensive normalization for entity name matching
func normalizeEntityName(name string) string {
	// Comprehensive normalization
	normalized := strings.ToLower(strings.TrimSpace(name))

	// Remove common punctuation
	normalized = strings.ReplaceAll(normalized, ",", "")
	normalized = strings.ReplaceAll(normalized, ".", "")

	// Remove leading articles (the, a, an) at the beginning only
	if strings.HasPrefix(normalized, "the ") {
		normalized = strings.TrimPrefix(normalized, "the ")
	}
	if strings.HasPrefix(normalized, "a ") {
		normalized = strings.TrimPrefix(normalized, "a ")
	}
	if strings.HasPrefix(normalized, "an ") {
		normalized = strings.TrimPrefix(normalized, "an ")
	}

	// Handle common abbreviations or variations
	normalized = strings.ReplaceAll(normalized, "&", "and")
	normalized = strings.ReplaceAll(normalized, "corp", "corporation")
	normalized = strings.ReplaceAll(normalized, "inc", "incorporated")

	// Remove extra spaces
	normalized = strings.Join(strings.Fields(normalized), " ")

	return normalized
}
