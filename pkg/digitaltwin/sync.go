package digitaltwin

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/mimir-aip/mimir-aip-go/pkg/metadatastore"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"log"
	"strings"
	"time"
)

// EnqueueSync marks a twin as syncing and submits worker-backed sync work.
func (s *Service) EnqueueSync(twinID string) (*models.WorkTask, error) {
	if s.queue == nil {
		return nil, fmt.Errorf("work queue is not configured")
	}
	twin, err := s.store.GetDigitalTwin(twinID)
	if err != nil {
		return nil, fmt.Errorf("failed to get digital twin: %w", err)
	}
	now := time.Now().UTC()
	previousStatus := twin.Status
	previousUpdatedAt := twin.UpdatedAt
	twin.Status = "syncing"
	twin.UpdatedAt = now
	if err := s.store.SaveDigitalTwin(twin); err != nil {
		return nil, fmt.Errorf("failed to update digital twin status: %w", err)
	}
	task := &models.WorkTask{
		ID:          uuid.New().String(),
		Type:        models.WorkTaskTypeDigitalTwinProcessing,
		ProjectID:   twin.ProjectID,
		Priority:    5,
		Status:      models.WorkTaskStatusQueued,
		SubmittedAt: now,
		TaskSpec: models.TaskSpec{
			ProjectID:  twin.ProjectID,
			Parameters: map[string]any{"digital_twin_id": twinID, "sync_trigger_type": "manual", "sync_triggered_by": "enqueue_sync"},
		},
	}
	if err := s.queue.Enqueue(task); err != nil {
		twin.Status = previousStatus
		twin.UpdatedAt = previousUpdatedAt
		_ = s.store.SaveDigitalTwin(twin)
		return nil, fmt.Errorf("failed to enqueue digital twin sync: %w", err)
	}
	return task, nil
}

func (s *Service) SyncWithStorage(twinID string) error {
	return s.SyncWithStorageWithOptions(twinID, nil)
}

func (s *Service) SyncWithStorageWithOptions(twinID string, opts *models.TwinSyncOptions) error {
	twin, err := s.store.GetDigitalTwin(twinID)
	if err != nil {
		return fmt.Errorf("failed to get digital twin: %w", err)
	}
	ont, err := s.ontologyService.GetOntology(twin.OntologyID)
	if err != nil {
		s.markTwinSyncFailed(twin)
		return fmt.Errorf("failed to get ontology: %w", err)
	}
	policy := defaultReconciliationPolicy(twin)
	sourceIDs := []string{}
	if twin.Config != nil {
		sourceIDs = append(sourceIDs, twin.Config.StorageIDs...)
	}
	run := &models.TwinSyncRun{
		ID:                     uuid.New().String(),
		DigitalTwinID:          twinID,
		TriggerType:            defaultString(optValue(opts, func(o *models.TwinSyncOptions) string { return o.TriggerType }), "system"),
		TriggeredBy:            optValue(opts, func(o *models.TwinSyncOptions) string { return o.TriggeredBy }),
		SourceIDs:              sourceIDs,
		OntologyVersion:        ont.Version,
		ReconciliationStrategy: policy.Strategy,
		StartedAt:              time.Now().UTC(),
		Status:                 "running",
		Summary:                map[string]interface{}{},
	}
	if err := s.store.SaveTwinSyncRun(run); err != nil {
		return fmt.Errorf("failed to save twin sync run: %w", err)
	}
	processedSources := 0
	if twin.Config != nil && len(twin.Config.StorageIDs) > 0 {
		for _, storageID := range twin.Config.StorageIDs {
			if err := s.syncFromStorage(twin, ont, storageID); err != nil {
				s.markTwinSyncFailed(twin)
				completedAt := time.Now().UTC()
				run.CompletedAt = &completedAt
				run.Status = "failed"
				run.Error = err.Error()
				run.Summary["processed_sources"] = processedSources
				_ = s.store.SaveTwinSyncRun(run)
				return fmt.Errorf("failed to sync from storage %s: %w", storageID, err)
			}
			processedSources++
		}
	}
	relationshipHighWatermark, err := s.wireRelationships(twin.ID, run)
	if err != nil {
		fmt.Printf("Warning: failed to wire cross-type relationships for twin %s: %v\n", twin.ID, err)
	}
	now := time.Now().UTC()
	twin.Status = "active"
	twin.LastSyncAt = &now
	twin.UpdatedAt = now
	if err := s.store.SaveDigitalTwin(twin); err != nil {
		return fmt.Errorf("failed to update digital twin: %w", err)
	}
	completedAt := time.Now().UTC()
	run.CompletedAt = &completedAt
	run.Status = "completed"
	run.Summary["processed_sources"] = processedSources
	run.Summary["entity_revision_high_watermark"] = latestEntityRevisionHighWatermark(s.store, twin.ID)
	run.Summary["relationship_revision_high_watermark"] = relationshipHighWatermark
	run.EntityRevisionHighWatermark = latestEntityRevisionHighWatermark(s.store, twin.ID)
	if snapshot, snapshotErr := s.captureTwinSnapshot(twin.ID, run, relationshipHighWatermark); snapshotErr != nil {
		return fmt.Errorf("failed to capture twin snapshot: %w", snapshotErr)
	} else {
		run.BaseSnapshotID = snapshot.ID
		run.Summary["snapshot_id"] = snapshot.ID
	}
	if err := s.store.SaveTwinSyncRun(run); err != nil {
		return fmt.Errorf("failed to finalize twin sync run: %w", err)
	}
	return nil
}

func (s *Service) captureTwinSnapshot(twinID string, run *models.TwinSyncRun, relationshipHighWatermark int) (*models.TwinSnapshot, error) {
	entities, err := s.store.ListEntitiesByDigitalTwin(twinID)
	if err != nil {
		return nil, fmt.Errorf("failed to list entities for snapshot: %w", err)
	}
	relationships := flattenRelationships(entities)
	entityState, err := json.Marshal(entities)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal snapshot entities: %w", err)
	}
	relationshipState, err := json.Marshal(relationships)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal snapshot relationships: %w", err)
	}
	snapshot := &models.TwinSnapshot{
		ID:                                uuid.New().String(),
		DigitalTwinID:                     twinID,
		SyncRunID:                         run.ID,
		SnapshotKind:                      "full",
		EntityState:                       entityState,
		RelationshipState:                 relationshipState,
		CreatedAt:                         time.Now().UTC(),
		EntityRevisionHighWatermark:       run.EntityRevisionHighWatermark,
		RelationshipRevisionHighWatermark: relationshipHighWatermark,
		Metadata: map[string]interface{}{
			"trigger_type":            run.TriggerType,
			"reconciliation_strategy": run.ReconciliationStrategy,
			"entity_count":            len(entities),
			"relationship_count":      len(relationships),
		},
	}
	if err := s.store.SaveTwinSnapshot(snapshot); err != nil {
		return nil, err
	}
	return snapshot, nil
}

func flattenRelationships(entities []*models.Entity) []*models.EntityRelationship {
	relationships := make([]*models.EntityRelationship, 0)
	seen := make(map[string]bool)
	for _, entity := range entities {
		for _, rel := range entity.Relationships {
			if rel == nil {
				continue
			}
			key := entity.ID + "|" + rel.Type + "|" + rel.TargetID
			if seen[key] {
				continue
			}
			seen[key] = true
			copy := *rel
			copy.Properties = cloneJSONMap(rel.Properties)
			relationships = append(relationships, &copy)
		}
	}
	return relationships
}

func latestRelationshipRevisionHighWatermark(store metadatastore.MetadataStore, twinID string) int {
	revisions, err := store.ListRelationshipRevisions(twinID, "", 1)
	if err != nil || len(revisions) == 0 {
		return 0
	}
	return revisions[0].Revision
}

func optValue(opts *models.TwinSyncOptions, selector func(*models.TwinSyncOptions) string) string {
	if opts == nil {
		return ""
	}
	return selector(opts)
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func latestEntityRevisionHighWatermark(store metadatastore.MetadataStore, twinID string) int {
	entities, err := store.ListEntitiesByDigitalTwin(twinID)
	if err != nil {
		return 0
	}
	maxRevision := 0
	for _, entity := range entities {
		revisions, err := store.ListEntityRevisions(entity.ID, 1)
		if err != nil || len(revisions) == 0 {
			continue
		}
		if revisions[0].Revision > maxRevision {
			maxRevision = revisions[0].Revision
		}
	}
	return maxRevision
}

func (s *Service) markTwinSyncFailed(twin *models.DigitalTwin) {
	if twin == nil {
		return
	}
	twin.Status = "error"
	twin.UpdatedAt = time.Now().UTC()
	if err := s.store.SaveDigitalTwin(twin); err != nil {
		log.Printf("Warning: failed to persist digital twin sync failure for %s: %v", twin.ID, err)
	}
}

func (s *Service) syncFromStorage(twin *models.DigitalTwin, ont *models.Ontology, storageID string) error {
	cirs, err := s.storageService.RetrieveForProject(twin.ProjectID, storageID, &models.CIRQuery{})
	if err != nil {
		return fmt.Errorf("failed to retrieve CIR data from storage %s: %w", storageID, err)
	}

	// Build ontology class names from the persisted compiled ontology. Metadata is a fallback for old rows.
	var classNames []string
	if compiled, err := s.ontologyService.GetCompiledOntologyForProject(twin.ProjectID, twin.OntologyID); err == nil {
		for _, class := range compiled.Classes {
			classNames = append(classNames, class.ID)
		}
	} else if entityTypes, ok := twin.Metadata["entity_types"]; ok {
		if etMap, ok := entityTypes.(map[string]interface{}); ok {
			for name := range etMap {
				classNames = append(classNames, name)
			}
		}
	}

	// Pre-load existing entities per type into an in-memory index to avoid
	// N×M database queries during resolution.  The index is keyed by
	// (entityType, keyField, keyValue) → entity pointer.
	//
	// We build this lazily per entity type on first encounter.
	typeIndex := make(map[string]map[string]*models.Entity) // entityType → (keyField+":"+keyValue → entity)

	loadTypeIndex := func(entityType string) {
		if _, loaded := typeIndex[entityType]; loaded {
			return
		}
		existing, err := s.store.ListEntitiesByTypeInTwin(twin.ID, entityType)
		if err != nil {
			typeIndex[entityType] = make(map[string]*models.Entity)
			return
		}
		idx := make(map[string]*models.Entity, len(existing))
		for _, e := range existing {
			for _, kf := range detectKeyFields(e.Attributes) {
				kv := keyValue(e.Attributes[kf])
				if kv != "" {
					idx[kf+":"+kv] = e
				}
			}
		}
		typeIndex[entityType] = idx
	}

	now := time.Now().UTC()

	policy := defaultReconciliationPolicy(twin)

	for _, cir := range cirs {
		dataMap, err := cir.GetDataAsMap()
		if err != nil {
			continue
		}

		entityType := inferEntityTypeFromCIR(cir, classNames)
		attrs := make(map[string]interface{}, len(dataMap))
		if cir.Metadata.Ontology != nil {
			if cir.Metadata.Ontology.ClassID != "" {
				entityType = cir.Metadata.Ontology.ClassID
			}
			for propertyID, property := range cir.Metadata.Ontology.Properties {
				attrs[propertyID] = property.Value
			}
			if len(attrs) == 0 {
				for k, v := range dataMap {
					attrs[k] = v
				}
			}
		} else {
			for k, v := range dataMap {
				attrs[k] = v
			}
		}
		sourceID := cir.Source.URI

		// Attempt entity resolution: find an existing entity of the same type
		// that shares a key-field value with this CIR record.
		loadTypeIndex(entityType)
		idx := typeIndex[entityType]

		var resolved *models.Entity
		keyFields := detectKeyFields(attrs)
		for _, kf := range keyFields {
			kv := keyValue(attrs[kf])
			if kv == "" {
				continue
			}
			if existing, ok := idx[kf+":"+kv]; ok {
				resolved = existing
				break
			}
		}

		if resolved != nil {
			mergeEntityAttributes(resolved, attrs, storageID, cir.Source.Timestamp.UTC(), policy)
			appendSourceID(resolved, storageID)
			resolved.UpdatedAt = now
			if cir.Metadata.Ontology != nil {
				if resolved.ComputedValues == nil {
					resolved.ComputedValues = map[string]interface{}{}
				}
				resolved.ComputedValues["ontology_mapping"] = semanticMappingSummary(cir.Metadata.Ontology)
			}

			if err := s.store.SaveEntity(resolved); err != nil {
				fmt.Printf("Warning: failed to merge entity from storage %s: %v\n", storageID, err)
			}
			for _, kf := range keyFields {
				kv := keyValue(resolved.Attributes[kf])
				if kv != "" {
					idx[kf+":"+kv] = resolved
				}
			}
		} else {
			entity := &models.Entity{
				ID:            uuid.New().String(),
				DigitalTwinID: twin.ID,
				Type:          entityType,
				Attributes:    attrs,
				SourceDataID:  &sourceID,
				IsModified:    false,
				Modifications: make(map[string]interface{}),
				ComputedValues: map[string]interface{}{
					"source_ids":               []interface{}{storageID},
					"attribute_sources":        attributeSourceMap(attrs, storageID),
					"attribute_timestamps":     attributeTimestampMap(attrs, cir.Source.Timestamp.UTC()),
					"reconciliation_conflicts": map[string]interface{}{},
					"ontology_mapping":         semanticMappingSummary(cir.Metadata.Ontology),
				},
				CreatedAt: now,
				UpdatedAt: now,
			}
			if err := s.store.SaveEntity(entity); err != nil {
				fmt.Printf("Warning: failed to save entity from CIR: %v\n", err)
				continue
			}
			// Register in index so subsequent CIRs from this storage can match it.
			for _, kf := range keyFields {
				kv := keyValue(attrs[kf])
				if kv != "" {
					idx[kf+":"+kv] = entity
				}
			}
		}
	}

	return nil
}

func semanticMappingSummary(mapping *models.CIRSemanticMapping) map[string]interface{} {
	if mapping == nil {
		return nil
	}
	return map[string]interface{}{
		"ontology_id":     mapping.OntologyID,
		"content_hash":    mapping.ContentHash,
		"class_id":        mapping.ClassID,
		"matched_by":      mapping.MatchedBy,
		"property_count":  len(mapping.Properties),
		"unmapped_fields": mapping.UnmappedFields,
		"violations":      mapping.Violations,
	}
}

func defaultReconciliationPolicy(twin *models.DigitalTwin) *models.TwinReconciliationPolicy {
	if twin != nil && twin.Config != nil && twin.Config.Reconciliation != nil {
		policy := *twin.Config.Reconciliation
		if policy.Strategy == "" {
			policy.Strategy = "source_priority"
		}
		return &policy
	}
	return &models.TwinReconciliationPolicy{Strategy: "source_priority"}
}
