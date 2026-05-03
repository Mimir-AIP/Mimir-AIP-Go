package digitaltwin

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

func (s *Service) wireRelationships(twinID string, run *models.TwinSyncRun) (int, error) {
	allEntities, err := s.store.ListEntitiesByDigitalTwin(twinID)
	if err != nil {
		return 0, fmt.Errorf("failed to list entities: %w", err)
	}
	byType := make(map[string][]*models.Entity)
	entityByID := make(map[string]*models.Entity)
	currentRelationships := make(map[string]*models.EntityRelationship)
	for _, e := range allEntities {
		byType[e.Type] = append(byType[e.Type], e)
		entityByID[e.ID] = e
		for _, rel := range e.Relationships {
			if rel == nil {
				continue
			}
			currentRelationships[relationshipKey(e.ID, rel.Type, rel.TargetID)] = rel
		}
		e.Relationships = nil
	}
	types := make([]string, 0, len(byType))
	for t := range byType {
		types = append(types, t)
	}
	if len(types) < 2 {
		return latestRelationshipRevisionHighWatermark(s.store, twinID), nil
	}
	desiredRelationships := make(map[string]*models.EntityRelationship)
	for i := 0; i < len(types); i++ {
		for j := i + 1; j < len(types); j++ {
			typeA, typeB := types[i], types[j]
			entitiesA := byType[typeA]
			entitiesB := byType[typeB]
			sharedKeys := commonKeyFieldNames(entitiesA, entitiesB)
			if len(sharedKeys) == 0 {
				continue
			}
			bIndex := make(map[string]map[string]*models.Entity)
			for _, e := range entitiesB {
				for _, kf := range sharedKeys {
					kv := keyValue(e.Attributes[kf])
					if kv == "" {
						continue
					}
					if bIndex[kf] == nil {
						bIndex[kf] = make(map[string]*models.Entity)
					}
					bIndex[kf][kv] = e
				}
			}
			for _, eA := range entitiesA {
				for _, kf := range sharedKeys {
					kv := keyValue(eA.Attributes[kf])
					if kv == "" {
						continue
					}
					eB, ok := bIndex[kf][kv]
					if !ok {
						continue
					}
					relType := "relatedBy" + toCamelCaseRel(kf)
					addDesiredRelationship(eA, eB, relType, typeB, desiredRelationships)
					addDesiredRelationship(eB, eA, relType, typeA, desiredRelationships)
				}
			}
		}
	}
	relationshipHighWatermark, err := s.persistRelationshipDiffs(twinID, run, currentRelationships, desiredRelationships)
	if err != nil {
		return 0, err
	}
	for _, entity := range entityByID {
		if err := s.store.SaveEntity(entity); err != nil {
			fmt.Printf("Warning: failed to save wired entity %s: %v\n", entity.ID, err)
		}
	}
	return relationshipHighWatermark, nil
}

func addDesiredRelationship(source, target *models.Entity, relType, targetType string, desired map[string]*models.EntityRelationship) {
	key := relationshipKey(source.ID, relType, target.ID)
	if _, exists := desired[key]; exists {
		return
	}
	rel := &models.EntityRelationship{Type: relType, TargetID: target.ID, TargetType: targetType}
	source.Relationships = append(source.Relationships, rel)
	desired[key] = rel
}

func relationshipKey(sourceID, relType, targetID string) string {
	return sourceID + "|" + relType + "|" + targetID
}

func (s *Service) persistRelationshipDiffs(twinID string, run *models.TwinSyncRun, current, desired map[string]*models.EntityRelationship) (int, error) {
	now := time.Now().UTC()
	currentHighWatermark := latestRelationshipRevisionHighWatermark(s.store, twinID)
	nextRevision := currentHighWatermark
	for key, desiredRel := range desired {
		currentRel, exists := current[key]
		if exists && relationshipsEquivalent(currentRel, desiredRel) {
			delete(current, key)
			continue
		}
		nextRevision++
		revision := &models.RelationshipRevision{
			ID:               uuid.New().String(),
			DigitalTwinID:    twinID,
			SyncRunID:        optSyncRunID(run),
			SourceEntityID:   relationshipSourceID(key),
			TargetEntityID:   desiredRel.TargetID,
			RelationshipType: desiredRel.Type,
			Revision:         nextRevision,
			ChangeType:       relationshipChangeType(exists),
			DeltaData:        relationshipDelta(currentRel, desiredRel),
			FullState:        relationshipFullState(desiredRel),
			Provenance:       relationshipProvenance(run),
			RecordedAt:       now,
			OntologyVersion:  optOntologyVersion(run),
		}
		if err := s.store.SaveRelationshipRevision(revision); err != nil {
			return 0, fmt.Errorf("failed to save relationship revision: %w", err)
		}
		delete(current, key)
	}
	for key, currentRel := range current {
		nextRevision++
		revision := &models.RelationshipRevision{
			ID:               uuid.New().String(),
			DigitalTwinID:    twinID,
			SyncRunID:        optSyncRunID(run),
			SourceEntityID:   relationshipSourceID(key),
			TargetEntityID:   currentRel.TargetID,
			RelationshipType: currentRel.Type,
			Revision:         nextRevision,
			ChangeType:       "removed", DeltaData: relationshipDelta(currentRel, nil),
			Provenance:      relationshipProvenance(run),
			RecordedAt:      now,
			OntologyVersion: optOntologyVersion(run),
		}
		if err := s.store.SaveRelationshipRevision(revision); err != nil {
			return 0, fmt.Errorf("failed to save removed relationship revision: %w", err)
		}
	}
	return nextRevision, nil
}

func relationshipSourceID(key string) string {
	parts := strings.Split(key, "|")
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

func relationshipChangeType(existed bool) string {
	if existed {
		return "updated"
	}
	return "added"
}

func relationshipsEquivalent(a, b *models.EntityRelationship) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	if a.Type != b.Type || a.TargetID != b.TargetID || a.TargetType != b.TargetType {
		return false
	}
	return fmt.Sprintf("%v", a.Properties) == fmt.Sprintf("%v", b.Properties)
}

func relationshipDelta(before, after *models.EntityRelationship) map[string]interface{} {
	return map[string]interface{}{"before": relationshipFullState(before), "after": relationshipFullState(after)}
}

func relationshipFullState(rel *models.EntityRelationship) map[string]interface{} {
	if rel == nil {
		return nil
	}
	return map[string]interface{}{"type": rel.Type, "target_id": rel.TargetID, "target_type": rel.TargetType, "properties": cloneJSONMap(rel.Properties)}
}

func relationshipProvenance(run *models.TwinSyncRun) map[string]interface{} {
	if run == nil {
		return nil
	}
	return map[string]interface{}{"sync_run_id": run.ID, "trigger_type": run.TriggerType, "triggered_by": run.TriggeredBy}
}

func optSyncRunID(run *models.TwinSyncRun) string {
	if run == nil {
		return ""
	}
	return run.ID
}

func optOntologyVersion(run *models.TwinSyncRun) string {
	if run == nil {
		return ""
	}
	return run.OntologyVersion
}

// commonKeyFieldNames returns key-like attribute names that appear in both
// entity type sets.  Only the first 20 entities per type are sampled to keep
