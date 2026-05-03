package digitaltwin

import (
	"fmt"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"log"
	"time"
)

// GetEntity retrieves an entity by ID.
func (s *Service) GetEntity(id string) (*models.Entity, error) {
	entity, err := s.store.GetEntity(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get entity: %w", err)
	}
	return entity, nil
}

// GetEntityHistory lists historical snapshots for one entity, newest first.
func (s *Service) GetEntityHistory(entityID string, limit int) ([]*models.EntityRevision, error) {
	revisions, err := s.store.ListEntityRevisions(entityID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list entity history: %w", err)
	}
	return revisions, nil
}

// UpdateEntity updates entity attributes (stores as delta modifications)
func (s *Service) UpdateEntity(entityID string, req *models.EntityUpdateRequest) (*models.Entity, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	entity, err := s.store.GetEntity(entityID)
	if err != nil {
		return nil, fmt.Errorf("failed to get entity: %w", err)
	}

	// Store modifications as deltas
	if entity.Modifications == nil {
		entity.Modifications = make(map[string]interface{})
	}

	for key, value := range req.Attributes {
		entity.Modifications[key] = value
		// Also update current attributes
		if entity.Attributes == nil {
			entity.Attributes = make(map[string]interface{})
		}
		entity.Attributes[key] = value
	}

	entity.IsModified = true
	entity.UpdatedAt = time.Now().UTC()

	if err := s.store.SaveEntity(entity); err != nil {
		return nil, fmt.Errorf("failed to update entity: %w", err)
	}

	// Invalidate predictions for this entity
	if err := s.invalidatePredictionsForEntity(entityID); err != nil {
		// Log error but don't fail
		fmt.Printf("Warning: failed to invalidate predictions: %v\n", err)
	}

	return entity, nil
}

// ListEntities lists all entities for a digital twin.
func (s *Service) ListEntities(twinID string) ([]*models.Entity, error) {
	entities, err := s.store.ListEntitiesByDigitalTwin(twinID)
	if err != nil {
		return nil, fmt.Errorf("failed to list entities: %w", err)
	}
	return entities, nil
}

func (s *Service) GetRelatedEntities(twinID, entityID, relationshipType string) ([]*models.Entity, error) {
	source, err := s.getOwnedEntity(twinID, entityID)
	if err != nil {
		return nil, err
	}

	var results []*models.Entity
	for _, rel := range source.Relationships {
		if relationshipType != "" && rel.Type != relationshipType {
			continue
		}
		target, err := s.getOwnedEntity(twinID, rel.TargetID)
		if err != nil {
			log.Printf("GetRelatedEntities: failed to get entity %s: %v", rel.TargetID, err)
			continue
		}
		results = append(results, target)
	}
	return results, nil
}

// invalidatePredictionsForEntity removes cached predictions for an entity
func (s *Service) invalidatePredictionsForEntity(entityID string) error {
	predictions, err := s.store.ListPredictionsByEntity(entityID)
	if err != nil {
		return err
	}

	for _, pred := range predictions {
		if err := s.store.DeletePrediction(pred.ID); err != nil {
			return err
		}
	}

	return nil
}
