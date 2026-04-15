package digitaltwin

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	automationpkg "github.com/mimir-aip/mimir-aip-go/pkg/automation"
	"github.com/mimir-aip/mimir-aip-go/pkg/metadatastore"
	"github.com/mimir-aip/mimir-aip-go/pkg/mlmodel"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"github.com/mimir-aip/mimir-aip-go/pkg/ontology"
	"github.com/mimir-aip/mimir-aip-go/pkg/queue"
	"github.com/mimir-aip/mimir-aip-go/pkg/storage"
)

// Service manages digital twin operations.
type Service struct {
	store             metadatastore.MetadataStore
	automationService *automationpkg.Service
	ontologyService   *ontology.Service
	storageService    *storage.Service
	mlService         *mlmodel.Service
	queue             *queue.Queue
	inferenceEngine   *InferenceEngine
	sparqlEngine      *SPARQLEngine
	scenarioManager   *ScenarioManager
	actionManager     *ActionManager
}

type DigitalTwinProjectMismatchError struct {
	DigitalTwinID     string
	ExpectedProjectID string
	ActualProjectID   string
}

func (e *DigitalTwinProjectMismatchError) Error() string {
	return fmt.Sprintf("digital twin %s belongs to project %s, not %s", e.DigitalTwinID, e.ActualProjectID, e.ExpectedProjectID)
}

func (s *Service) getOwnedDigitalTwin(projectID, twinID string) (*models.DigitalTwin, error) {
	if strings.TrimSpace(projectID) == "" {
		return nil, fmt.Errorf("project_id is required")
	}
	twin, err := s.store.GetDigitalTwin(twinID)
	if err != nil {
		return nil, fmt.Errorf("failed to get digital twin: %w", err)
	}
	if twin.ProjectID != projectID {
		return nil, &DigitalTwinProjectMismatchError{
			DigitalTwinID:     twin.ID,
			ExpectedProjectID: projectID,
			ActualProjectID:   twin.ProjectID,
		}
	}
	return twin, nil
}

func (s *Service) getOwnedEntity(twinID, entityID string) (*models.Entity, error) {
	entity, err := s.store.GetEntity(entityID)
	if err != nil {
		return nil, fmt.Errorf("failed to get entity: %w", err)
	}
	if entity.DigitalTwinID != twinID {
		return nil, fmt.Errorf("entity %s does not belong to digital twin %s", entityID, twinID)
	}
	return entity, nil
}

func (s *Service) getOwnedScenario(twinID, scenarioID string) (*models.Scenario, error) {
	scenario, err := s.store.GetScenario(scenarioID)
	if err != nil {
		return nil, fmt.Errorf("failed to get scenario: %w", err)
	}
	if scenario.DigitalTwinID != twinID {
		return nil, fmt.Errorf("scenario %s does not belong to digital twin %s", scenarioID, twinID)
	}
	return scenario, nil
}

func (s *Service) getOwnedAction(twinID, actionID string) (*models.Action, error) {
	action, err := s.store.GetAction(actionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get action: %w", err)
	}
	if action.DigitalTwinID != twinID {
		return nil, fmt.Errorf("action %s does not belong to digital twin %s", actionID, twinID)
	}
	return action, nil
}

func (s *Service) GetDigitalTwinForProject(projectID, twinID string) (*models.DigitalTwin, error) {
	return s.getOwnedDigitalTwin(projectID, twinID)
}

func (s *Service) UpdateDigitalTwinForProject(projectID, twinID string, req *models.DigitalTwinUpdateRequest) (*models.DigitalTwin, error) {
	if _, err := s.getOwnedDigitalTwin(projectID, twinID); err != nil {
		return nil, err
	}
	return s.UpdateDigitalTwin(twinID, req)
}

func (s *Service) DeleteDigitalTwinForProject(projectID, twinID string) error {
	if _, err := s.getOwnedDigitalTwin(projectID, twinID); err != nil {
		return err
	}
	return s.DeleteDigitalTwin(twinID)
}

func (s *Service) EnqueueSyncForProject(projectID, twinID string) (*models.WorkTask, error) {
	twin, err := s.getOwnedDigitalTwin(projectID, twinID)
	if err != nil {
		return nil, err
	}
	return s.EnqueueSync(twin.ID)
}

func (s *Service) SyncWithStorageForProject(projectID, twinID string, opts *models.TwinSyncOptions) error {
	twin, err := s.getOwnedDigitalTwin(projectID, twinID)
	if err != nil {
		return err
	}
	return s.SyncWithStorageWithOptions(twin.ID, opts)
}

func (s *Service) GetStateAtRunForProject(projectID, twinID, runID string) (*models.ReconstructedTwinState, error) {
	twin, err := s.getOwnedDigitalTwin(projectID, twinID)
	if err != nil {
		return nil, err
	}
	return s.GetStateAtRun(twin.ID, runID)
}

func (s *Service) ListSyncRunsForProject(projectID, twinID string, limit int) ([]*models.TwinSyncRun, error) {
	twin, err := s.getOwnedDigitalTwin(projectID, twinID)
	if err != nil {
		return nil, err
	}
	return s.ListSyncRuns(twin.ID, limit)
}

func (s *Service) GetSyncRunForProject(projectID, twinID, runID string) (*models.TwinSyncRun, error) {
	twin, err := s.getOwnedDigitalTwin(projectID, twinID)
	if err != nil {
		return nil, err
	}
	return s.GetSyncRun(twin.ID, runID)
}

func (s *Service) QueryForProject(projectID, twinID string, req *models.QueryRequest) (*models.QueryResult, error) {
	twin, err := s.getOwnedDigitalTwin(projectID, twinID)
	if err != nil {
		return nil, err
	}
	return s.Query(twin.ID, req)
}

func (s *Service) PredictForProject(projectID, twinID string, req *models.PredictionRequest) (*models.Prediction, error) {
	twin, err := s.getOwnedDigitalTwin(projectID, twinID)
	if err != nil {
		return nil, err
	}
	return s.Predict(twin.ID, req)
}

func (s *Service) BatchPredictForProject(projectID, twinID string, req *models.BatchPredictionRequest) ([]*models.Prediction, error) {
	twin, err := s.getOwnedDigitalTwin(projectID, twinID)
	if err != nil {
		return nil, err
	}
	return s.BatchPredict(twin.ID, req)
}

func (s *Service) GetEntityInTwin(twinID, entityID string) (*models.Entity, error) {
	return s.getOwnedEntity(twinID, entityID)
}

func (s *Service) UpdateEntityInTwin(twinID, entityID string, req *models.EntityUpdateRequest) (*models.Entity, error) {
	if _, err := s.getOwnedEntity(twinID, entityID); err != nil {
		return nil, err
	}
	return s.UpdateEntity(entityID, req)
}

func (s *Service) GetEntityHistoryInTwin(twinID, entityID string, limit int) ([]*models.EntityRevision, error) {
	if _, err := s.getOwnedEntity(twinID, entityID); err != nil {
		return nil, err
	}
	return s.GetEntityHistory(entityID, limit)
}

func (s *Service) GetScenarioInTwin(twinID, scenarioID string) (*models.Scenario, error) {
	return s.getOwnedScenario(twinID, scenarioID)
}

func (s *Service) DeleteScenarioInTwin(twinID, scenarioID string) error {
	if _, err := s.getOwnedScenario(twinID, scenarioID); err != nil {
		return err
	}
	return s.DeleteScenario(scenarioID)
}

func (s *Service) GetActionInTwin(twinID, actionID string) (*models.Action, error) {
	return s.getOwnedAction(twinID, actionID)
}

func (s *Service) DeleteActionInTwin(twinID, actionID string) error {
	if _, err := s.getOwnedAction(twinID, actionID); err != nil {
		return err
	}
	return s.DeleteAction(actionID)
}

func (s *Service) ListActionsForProject(projectID, twinID string) ([]*models.Action, error) {
	twin, err := s.getOwnedDigitalTwin(projectID, twinID)
	if err != nil {
		return nil, err
	}
	return s.ListActions(twin.ID)
}

func (s *Service) ListScenariosForProject(projectID, twinID string) ([]*models.Scenario, error) {
	twin, err := s.getOwnedDigitalTwin(projectID, twinID)
	if err != nil {
		return nil, err
	}
	return s.ListScenarios(twin.ID)
}

func (s *Service) ListEntitiesForProject(projectID, twinID string) ([]*models.Entity, error) {
	twin, err := s.getOwnedDigitalTwin(projectID, twinID)
	if err != nil {
		return nil, err
	}
	return s.ListEntities(twin.ID)
}

func (s *Service) CreateScenarioForProject(projectID, twinID string, req *models.ScenarioCreateRequest) (*models.Scenario, error) {
	twin, err := s.getOwnedDigitalTwin(projectID, twinID)
	if err != nil {
		return nil, err
	}
	return s.CreateScenario(twin.ID, req)
}

func (s *Service) CreateActionForProject(projectID, twinID string, req *models.ActionCreateRequest) (*models.Action, error) {
	twin, err := s.getOwnedDigitalTwin(projectID, twinID)
	if err != nil {
		return nil, err
	}
	return s.CreateAction(twin.ID, req)
}

func (s *Service) GetRelatedEntitiesForProject(projectID, twinID, entityID, relationshipType string) ([]*models.Entity, error) {
	twin, err := s.getOwnedDigitalTwin(projectID, twinID)
	if err != nil {
		return nil, err
	}
	return s.GetRelatedEntities(twin.ID, entityID, relationshipType)
}

// NewService creates a new digital twin service.
func NewService(
	store metadatastore.MetadataStore,
	automationService *automationpkg.Service,
	ontologyService *ontology.Service,
	storageService *storage.Service,
	mlService *mlmodel.Service,
	q *queue.Queue,
) *Service {
	s := &Service{
		store:             store,
		automationService: automationService,
		ontologyService:   ontologyService,
		storageService:    storageService,
		mlService:         mlService,
		queue:             q,
	}

	// Initialize sub-components.
	s.inferenceEngine = NewInferenceEngine(mlService, store)
	s.sparqlEngine = NewSPARQLEngine(store, ontologyService)
	s.scenarioManager = NewScenarioManager(store, s.inferenceEngine)
	s.actionManager = NewActionManager(store, q)

	return s
}

// CreateDigitalTwin creates a new digital twin instance
func (s *Service) CreateDigitalTwin(req *models.DigitalTwinCreateRequest) (*models.DigitalTwin, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// Verify ontology exists
	ont, err := s.ontologyService.GetOntology(req.OntologyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get ontology: %w", err)
	}

	if ont.ProjectID != req.ProjectID {
		return nil, fmt.Errorf("ontology does not belong to project")
	}

	now := time.Now().UTC()
	twin := &models.DigitalTwin{
		ID:          uuid.New().String(),
		ProjectID:   req.ProjectID,
		OntologyID:  req.OntologyID,
		Name:        req.Name,
		Description: req.Description,
		Status:      "active",
		Config:      req.Config,
		Metadata:    make(map[string]interface{}),
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Set default config if not provided.
	if twin.Config == nil {
		twin.Config = &models.DigitalTwinConfig{
			EnablePredictions:  true,
			PredictionCacheTTL: 1800, // 30 minutes
		}
	}

	if err := s.store.SaveDigitalTwin(twin); err != nil {
		return nil, fmt.Errorf("failed to save digital twin: %w", err)
	}

	if err := s.ensureDefaultProcessingAutomation(twin); err != nil {
		return nil, err
	}

	// Initialize from ontology (populate entities from ontology blueprint).
	if err := s.initializeFromOntology(twin, ont); err != nil {
		return nil, fmt.Errorf("failed to initialize from ontology: %w", err)
	}

	return twin, nil
}

func (s *Service) ensureDefaultProcessingAutomation(twin *models.DigitalTwin) error {
	if s.automationService == nil || twin == nil {
		return nil
	}
	automations, err := s.automationService.ListByProject(twin.ProjectID)
	if err != nil {
		return fmt.Errorf("failed to list project automations: %w", err)
	}
	for _, automation := range automations {
		if automation.TargetType == models.AutomationTargetTypeDigitalTwin &&
			automation.TargetID == twin.ID &&
			automation.TriggerType == models.AutomationTriggerTypePipelineCompleted &&
			automation.ActionType == models.AutomationActionTypeProcessTwin {
			return nil
		}
	}
	_, err = s.automationService.Create(&models.AutomationCreateRequest{
		ProjectID:   twin.ProjectID,
		Name:        twin.Name + " processing",
		Description: "Default automation: process this twin after ingestion pipelines complete.",
		TargetType:  models.AutomationTargetTypeDigitalTwin,
		TargetID:    twin.ID,
		TriggerType: models.AutomationTriggerTypePipelineCompleted,
		TriggerConfig: map[string]any{
			"pipeline_types": []string{string(models.PipelineTypeIngestion)},
		},
		ActionType: models.AutomationActionTypeProcessTwin,
	})
	if err != nil {
		return fmt.Errorf("failed to create default twin processing automation: %w", err)
	}
	return nil
}

func (s *Service) storageIDsForTwin(twinID string) []string {
	twin, err := s.store.GetDigitalTwin(twinID)
	if err != nil || twin == nil || twin.Config == nil {
		return nil
	}
	return append([]string(nil), twin.Config.StorageIDs...)
}

// GetDigitalTwin retrieves a digital twin by ID.
func (s *Service) GetDigitalTwin(id string) (*models.DigitalTwin, error) {
	twin, err := s.store.GetDigitalTwin(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get digital twin: %w", err)
	}
	return twin, nil
}

// UpdateDigitalTwin updates an existing digital twin
func (s *Service) UpdateDigitalTwin(id string, req *models.DigitalTwinUpdateRequest) (*models.DigitalTwin, error) {
	twin, err := s.store.GetDigitalTwin(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get digital twin: %w", err)
	}

	// Apply updates
	if req.Name != nil {
		twin.Name = *req.Name
	}
	if req.Description != nil {
		twin.Description = *req.Description
	}
	if req.Status != nil {
		twin.Status = *req.Status
	}
	if req.Config != nil {
		twin.Config = req.Config
	}

	twin.UpdatedAt = time.Now().UTC()

	if err := s.store.SaveDigitalTwin(twin); err != nil {
		return nil, fmt.Errorf("failed to update digital twin: %w", err)
	}

	return twin, nil
}

// DeleteDigitalTwin deletes a digital twin
func (s *Service) DeleteDigitalTwin(id string) error {
	if err := s.store.DeleteDigitalTwin(id); err != nil {
		return fmt.Errorf("failed to delete digital twin: %w", err)
	}
	return nil
}

// ListDigitalTwins lists all digital twins
func (s *Service) ListDigitalTwins() ([]*models.DigitalTwin, error) {
	twins, err := s.store.ListDigitalTwins()
	if err != nil {
		return nil, fmt.Errorf("failed to list digital twins: %w", err)
	}
	return twins, nil
}

// ListDigitalTwinsByProject lists digital twins for a specific project
func (s *Service) ListDigitalTwinsByProject(projectID string) ([]*models.DigitalTwin, error) {
	twins, err := s.store.ListDigitalTwinsByProject(projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list digital twins: %w", err)
	}
	return twins, nil
}

// ListSyncRuns returns persisted sync/materialization runs for one twin.
func (s *Service) ListSyncRuns(twinID string, limit int) ([]*models.TwinSyncRun, error) {
	return s.store.ListTwinSyncRuns(twinID, limit)
}

// GetSyncRun returns one sync/materialization run for a twin.
func (s *Service) GetSyncRun(twinID, runID string) (*models.TwinSyncRun, error) {
	run, err := s.store.GetTwinSyncRun(runID)
	if err != nil {
		return nil, err
	}
	if run.DigitalTwinID != twinID {
		return nil, fmt.Errorf("twin sync run %s does not belong to digital twin %s", runID, twinID)
	}
	return run, nil
}

// ListSnapshots returns persisted checkpoints for one twin.
func (s *Service) ListSnapshots(twinID string, limit int) ([]*models.TwinSnapshot, error) {
	return s.store.ListTwinSnapshots(twinID, limit)
}

// GetStateAtRun reconstructs the twin graph from the checkpoint captured for one sync run.
func (s *Service) GetStateAtRun(twinID, runID string) (*models.ReconstructedTwinState, error) {
	snapshot, err := s.store.GetTwinSnapshotByRun(twinID, runID)
	if err != nil {
		return nil, err
	}
	return reconstructStateFromSnapshot(snapshot)
}

func reconstructStateFromSnapshot(snapshot *models.TwinSnapshot) (*models.ReconstructedTwinState, error) {
	if snapshot == nil {
		return nil, fmt.Errorf("snapshot is required")
	}
	entities := make([]*models.Entity, 0)
	if len(snapshot.EntityState) > 0 {
		if err := json.Unmarshal(snapshot.EntityState, &entities); err != nil {
			return nil, fmt.Errorf("failed to decode snapshot entity state: %w", err)
		}
	}
	relationships := make([]*models.EntityRelationship, 0)
	if len(snapshot.RelationshipState) > 0 {
		if err := json.Unmarshal(snapshot.RelationshipState, &relationships); err != nil {
			return nil, fmt.Errorf("failed to decode snapshot relationship state: %w", err)
		}
	}
	return &models.ReconstructedTwinState{
		DigitalTwinID: snapshot.DigitalTwinID,
		SyncRunID:     snapshot.SyncRunID,
		SnapshotID:    snapshot.ID,
		Entities:      entities,
		Relationships: relationships,
		Metadata:      snapshot.Metadata,
	}, nil
}

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

// Query executes a SPARQL query on the digital twin
func (s *Service) Query(twinID string, req *models.QueryRequest) (*models.QueryResult, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	twin, err := s.store.GetDigitalTwin(twinID)
	if err != nil {
		return nil, fmt.Errorf("failed to get digital twin: %w", err)
	}

	// Execute SPARQL query
	result, err := s.sparqlEngine.Execute(twin, req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}

	return result, nil
}

// Predict runs ML model prediction on entity data.
//
// When EntityID is provided, the service automatically enriches the input
// feature map with attributes from directly related entities, prefixed by
// their type (e.g. "attendance.days_absent").  This lets models trained on
// cross-source features produce accurate predictions even when called with
// only an entity reference rather than a manually constructed feature vector.
func (s *Service) Predict(twinID string, req *models.PredictionRequest) (*models.Prediction, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	twin, err := s.store.GetDigitalTwin(twinID)
	if err != nil {
		return nil, fmt.Errorf("failed to get digital twin: %w", err)
	}

	// Check if predictions are enabled
	if twin.Config != nil && !twin.Config.EnablePredictions {
		return nil, fmt.Errorf("predictions are disabled for this digital twin")
	}

	// Enrich the input with related entity attributes when an entity ID is
	// given.  This is non-fatal: if enrichment fails the caller-supplied
	// input (or an empty map) is used as-is.
	if req.EntityID != "" {
		if _, err := s.getOwnedEntity(twin.ID, req.EntityID); err != nil {
			return nil, err
		}
		if err := s.enrichPredictionInput(twin.ID, req); err != nil {
			fmt.Printf("Warning: failed to enrich prediction input for entity %s: %v\n", req.EntityID, err)
		}
	}

	// Run prediction through inference engine
	prediction, err := s.inferenceEngine.Predict(twin, req)
	if err != nil {
		return nil, fmt.Errorf("failed to run prediction: %w", err)
	}

	// Save prediction
	if err := s.store.SavePrediction(prediction); err != nil {
		return nil, fmt.Errorf("failed to save prediction: %w", err)
	}

	// Check if prediction triggers any actions
	if err := s.actionManager.EvaluateActions(twin.ID, prediction); err != nil {
		// Log error but don't fail
		fmt.Printf("Warning: failed to evaluate actions: %v\n", err)
	}

	return prediction, nil
}

// enrichPredictionInput loads the target entity's attributes into req.Input
// (if the caller did not supply them) and then merges attributes from all
// directly related entities under type-prefixed keys.
//
// Example: a Student entity with a related AttendanceRecord produces keys like:
//
//	"avg_grade"              → from Student.Attributes
//	"attendancerecord.days_absent" → from the related AttendanceRecord
//
// Related entity attributes are added only when the key is not already present,
// so caller-supplied values always take precedence.  Depth is limited to
// direct (depth-1) relationships to keep the feature vector bounded.
func (s *Service) enrichPredictionInput(twinID string, req *models.PredictionRequest) error {
	entity, err := s.getOwnedEntity(twinID, req.EntityID)
	if err != nil {
		return err
	}

	// Auto-populate input from entity attributes when the caller omitted them.
	if len(req.Input) == 0 {
		req.Input = make(map[string]interface{}, len(entity.Attributes))
		for k, v := range entity.Attributes {
			req.Input[k] = v
		}
	}

	// Merge related entity attributes with a type prefix.
	for _, rel := range entity.Relationships {
		related, err := s.getOwnedEntity(twinID, rel.TargetID)
		if err != nil {
			continue // non-fatal: skip missing or cross-twin related entities
		}
		prefix := strings.ToLower(rel.TargetType) + "."
		for k, v := range related.Attributes {
			key := prefix + k
			if _, exists := req.Input[key]; !exists {
				req.Input[key] = v
			}
		}
	}

	return nil
}

// BatchPredict runs batch predictions
func (s *Service) BatchPredict(twinID string, req *models.BatchPredictionRequest) ([]*models.Prediction, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	twin, err := s.store.GetDigitalTwin(twinID)
	if err != nil {
		return nil, fmt.Errorf("failed to get digital twin: %w", err)
	}

	if twin.Config != nil && !twin.Config.EnablePredictions {
		return nil, fmt.Errorf("predictions are disabled for this digital twin")
	}

	predictions, err := s.inferenceEngine.BatchPredict(twin, req)
	if err != nil {
		return nil, fmt.Errorf("failed to run batch predictions: %w", err)
	}

	// Save all predictions
	for _, prediction := range predictions {
		if err := s.store.SavePrediction(prediction); err != nil {
			fmt.Printf("Warning: failed to save prediction %s: %v\n", prediction.ID, err)
		}
	}

	return predictions, nil
}

// CreateScenario creates a new what-if scenario
func (s *Service) CreateScenario(twinID string, req *models.ScenarioCreateRequest) (*models.Scenario, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	twin, err := s.store.GetDigitalTwin(twinID)
	if err != nil {
		return nil, fmt.Errorf("failed to get digital twin: %w", err)
	}

	scenario, err := s.scenarioManager.CreateScenario(twin, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create scenario: %w", err)
	}

	return scenario, nil
}

// GetScenario retrieves a scenario by ID
func (s *Service) GetScenario(id string) (*models.Scenario, error) {
	scenario, err := s.store.GetScenario(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get scenario: %w", err)
	}
	return scenario, nil
}

// ListScenarios lists scenarios for a digital twin
func (s *Service) ListScenarios(twinID string) ([]*models.Scenario, error) {
	scenarios, err := s.store.ListScenariosByDigitalTwin(twinID)
	if err != nil {
		return nil, fmt.Errorf("failed to list scenarios: %w", err)
	}
	return scenarios, nil
}

// DeleteScenario deletes a scenario
func (s *Service) DeleteScenario(id string) error {
	if err := s.store.DeleteScenario(id); err != nil {
		return fmt.Errorf("failed to delete scenario: %w", err)
	}
	return nil
}

// CreateAction creates a new conditional action
func (s *Service) CreateAction(twinID string, req *models.ActionCreateRequest) (*models.Action, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	twin, err := s.store.GetDigitalTwin(twinID)
	if err != nil {
		return nil, fmt.Errorf("failed to get digital twin: %w", err)
	}

	action, err := s.actionManager.CreateAction(twin, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create action: %w", err)
	}

	return action, nil
}

// GetAction retrieves an action by ID
func (s *Service) GetAction(id string) (*models.Action, error) {
	action, err := s.store.GetAction(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get action: %w", err)
	}
	return action, nil
}

// ListActions lists actions for a digital twin
func (s *Service) ListActions(twinID string) ([]*models.Action, error) {
	actions, err := s.store.ListActionsByDigitalTwin(twinID)
	if err != nil {
		return nil, fmt.Errorf("failed to list actions: %w", err)
	}
	return actions, nil
}

// DeleteAction deletes an action
func (s *Service) DeleteAction(id string) error {
	if err := s.store.DeleteAction(id); err != nil {
		return fmt.Errorf("failed to delete action: %w", err)
	}
	return nil
}

// Helper functions

// ontologyClass holds parsed class information from an OWL/Turtle ontology
type ontologyClass struct {
	Name       string
	Label      string
	Properties []string
}

// initializeFromOntology parses the OWL Turtle content and records entity type metadata
func (s *Service) initializeFromOntology(twin *models.DigitalTwin, ont *models.Ontology) error {
	if ont.Content == "" {
		return nil
	}

	classes := parseOntologyClasses(ont.Content)
	if len(classes) == 0 {
		return nil
	}

	if twin.Metadata == nil {
		twin.Metadata = make(map[string]interface{})
	}

	entityTypes := make(map[string]interface{}, len(classes))
	for _, cls := range classes {
		entityTypes[cls.Name] = map[string]interface{}{
			"label":      cls.Label,
			"properties": cls.Properties,
		}
	}
	twin.Metadata["entity_types"] = entityTypes

	return s.store.SaveDigitalTwin(twin)
}

// parseOntologyClasses extracts class and property definitions from Turtle content
func parseOntologyClasses(turtleContent string) []ontologyClass {
	var classes []ontologyClass
	lines := strings.Split(turtleContent, "\n")

	classMap := make(map[string]*ontologyClass)
	domainMap := make(map[string]string) // property → class name

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Detect owl:Class declarations: ":ClassName a owl:Class" or ":ClassName rdf:type owl:Class"
		if (strings.Contains(line, "owl:Class") || strings.Contains(line, "owl:class")) &&
			(strings.Contains(line, " a ") || strings.Contains(line, "rdf:type")) {
			name := extractTurtleSubject(line)
			if name != "" && classMap[name] == nil {
				cls := &ontologyClass{Name: name, Label: name}
				classMap[name] = cls
			}
		}

		// Detect rdfs:label: ":ClassName rdfs:label "Label""
		if strings.Contains(line, "rdfs:label") {
			subject := extractTurtleSubject(line)
			label := extractStringLiteral(line)
			if subject != "" && label != "" {
				if cls, ok := classMap[subject]; ok {
					cls.Label = label
				}
			}
		}

		// Detect owl:DatatypeProperty declarations with rdfs:domain
		if strings.Contains(line, "owl:DatatypeProperty") || strings.Contains(line, "owl:ObjectProperty") {
			name := extractTurtleSubject(line)
			if name != "" {
				domainMap[name] = "" // register property, domain resolved later
			}
		}

		// Detect rdfs:domain: ":property rdfs:domain :ClassName"
		if strings.Contains(line, "rdfs:domain") {
			subject := extractTurtleSubject(line)
			domain := extractTurtleObject(line)
			if subject != "" && domain != "" {
				domainMap[subject] = domain
				if cls, ok := classMap[domain]; ok {
					if !containsString(cls.Properties, subject) {
						cls.Properties = append(cls.Properties, subject)
					}
				}
			}
		}
	}

	// Post-process: assign any unresolved properties to classes
	for prop, domain := range domainMap {
		if domain == "" {
			continue
		}
		if cls, ok := classMap[domain]; ok {
			if !containsString(cls.Properties, prop) {
				cls.Properties = append(cls.Properties, prop)
			}
		}
	}

	for _, cls := range classMap {
		classes = append(classes, *cls)
	}
	return classes
}

func extractTurtleSubject(line string) string {
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return ""
	}
	s := parts[0]
	// Strip leading colon if present
	s = strings.TrimPrefix(s, ":")
	// Strip trailing colon if prefix:local format
	if idx := strings.LastIndex(s, ":"); idx >= 0 {
		s = s[idx+1:]
	}
	return strings.Trim(s, "<>.,;")
}

func extractTurtleObject(line string) string {
	parts := strings.Fields(line)
	if len(parts) < 3 {
		return ""
	}
	s := parts[len(parts)-1]
	s = strings.TrimPrefix(s, ":")
	if idx := strings.LastIndex(s, ":"); idx >= 0 {
		s = s[idx+1:]
	}
	return strings.Trim(s, "<>.,;")
}

func extractStringLiteral(line string) string {
	start := strings.Index(line, `"`)
	if start < 0 {
		return ""
	}
	end := strings.Index(line[start+1:], `"`)
	if end < 0 {
		return ""
	}
	return line[start+1 : start+1+end]
}

func containsString(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

// syncFromStorage retrieves CIR data from a storage source and creates or
// merges entities in the digital twin.
//
// Entity resolution: before creating a new entity, the function checks whether
// an entity of the same type with the same key-field value already exists
// (e.g. from a previous pipeline that wrote to a different storage).  If found,
// the new attributes are merged into the existing entity rather than creating
// a duplicate.  The source storage ID is appended to the entity's source list.
func (s *Service) syncFromStorage(twin *models.DigitalTwin, ont *models.Ontology, storageID string) error {
	cirs, err := s.storageService.RetrieveForProject(twin.ProjectID, storageID, &models.CIRQuery{})
	if err != nil {
		return fmt.Errorf("failed to retrieve CIR data from storage %s: %w", storageID, err)
	}

	// Build ontology class names for entity type inference.
	var classNames []string
	if entityTypes, ok := twin.Metadata["entity_types"]; ok {
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
		sourceID := cir.Source.URI

		attrs := make(map[string]interface{}, len(dataMap))
		for k, v := range dataMap {
			attrs[k] = v
		}

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

func mergeEntityAttributes(entity *models.Entity, incoming map[string]interface{}, storageID string, observedAt time.Time, policy *models.TwinReconciliationPolicy) {
	if entity.Attributes == nil {
		entity.Attributes = make(map[string]interface{})
	}
	if entity.ComputedValues == nil {
		entity.ComputedValues = make(map[string]interface{})
	}
	attributeSources := ensureStringMap(entity.ComputedValues, "attribute_sources")
	attributeTimestamps := ensureStringMap(entity.ComputedValues, "attribute_timestamps")
	conflicts := ensureConflictMap(entity.ComputedValues)
	for key, incomingValue := range incoming {
		currentValue, exists := entity.Attributes[key]
		if !exists {
			entity.Attributes[key] = incomingValue
			attributeSources[key] = storageID
			attributeTimestamps[key] = observedAt.Format(time.RFC3339)
			continue
		}
		if valuesEquivalent(currentValue, incomingValue) {
			if _, ok := attributeSources[key]; !ok {
				attributeSources[key] = storageID
			}
			if _, ok := attributeTimestamps[key]; !ok {
				attributeTimestamps[key] = observedAt.Format(time.RFC3339)
			}
			continue
		}
		currentSource := attributeSources[key]
		currentObservedAt, _ := time.Parse(time.RFC3339, attributeTimestamps[key])
		chosenSource, chosenValue := resolveReconciledValue(currentValue, currentSource, currentObservedAt, incomingValue, storageID, observedAt, policy)
		entity.Attributes[key] = chosenValue
		attributeSources[key] = chosenSource
		attributeTimestamps[key] = maxTime(currentObservedAt, observedAt).Format(time.RFC3339)
		conflicts[key] = appendUniqueString(conflicts[key], fmt.Sprintf("%s=%v vs %s=%v", currentSource, currentValue, storageID, incomingValue))
	}
	entity.ComputedValues["attribute_sources"] = stringMapToInterfaceMap(attributeSources)
	entity.ComputedValues["attribute_timestamps"] = stringMapToInterfaceMap(attributeTimestamps)
	entity.ComputedValues["reconciliation_conflicts"] = conflictMapToInterfaceMap(conflicts)
}

func cloneJSONMap(values map[string]interface{}) map[string]interface{} {
	if values == nil {
		return nil
	}
	cloned := make(map[string]interface{}, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

func resolveReconciledValue(currentValue interface{}, currentSource string, currentObservedAt time.Time, incomingValue interface{}, incomingSource string, incomingObservedAt time.Time, policy *models.TwinReconciliationPolicy) (string, interface{}) {
	if policy == nil {
		policy = &models.TwinReconciliationPolicy{Strategy: "source_priority"}
	}
	switch policy.Strategy {
	case "freshest":
		if incomingObservedAt.After(currentObservedAt) {
			return incomingSource, incomingValue
		}
		return currentSource, currentValue
	case "source_priority", "":
		currentRank := sourcePriorityRank(currentSource, policy.SourcePriority)
		incomingRank := sourcePriorityRank(incomingSource, policy.SourcePriority)
		if incomingRank < currentRank {
			return incomingSource, incomingValue
		}
		if incomingRank == currentRank && incomingObservedAt.After(currentObservedAt) {
			return incomingSource, incomingValue
		}
		return currentSource, currentValue
	default:
		if incomingObservedAt.After(currentObservedAt) {
			return incomingSource, incomingValue
		}
		return currentSource, currentValue
	}
}

func sourcePriorityRank(sourceID string, priorities []string) int {
	for i, candidate := range priorities {
		if candidate == sourceID {
			return i
		}
	}
	return len(priorities) + 1
}

func attributeSourceMap(attrs map[string]interface{}, storageID string) map[string]interface{} {
	out := make(map[string]interface{}, len(attrs))
	for key := range attrs {
		out[key] = storageID
	}
	return out
}

func attributeTimestampMap(attrs map[string]interface{}, observedAt time.Time) map[string]interface{} {
	formatted := observedAt.Format(time.RFC3339)
	out := make(map[string]interface{}, len(attrs))
	for key := range attrs {
		out[key] = formatted
	}
	return out
}

func ensureStringMap(values map[string]interface{}, key string) map[string]string {
	result := make(map[string]string)
	if values == nil {
		return result
	}
	raw, _ := values[key].(map[string]interface{})
	for k, v := range raw {
		result[k] = fmt.Sprintf("%v", v)
	}
	return result
}

func ensureConflictMap(values map[string]interface{}) map[string][]string {
	result := make(map[string][]string)
	if values == nil {
		return result
	}
	raw, _ := values["reconciliation_conflicts"].(map[string]interface{})
	for key, value := range raw {
		if items, ok := value.([]interface{}); ok {
			for _, item := range items {
				result[key] = append(result[key], fmt.Sprintf("%v", item))
			}
		}
	}
	return result
}

func stringMapToInterfaceMap(values map[string]string) map[string]interface{} {
	out := make(map[string]interface{}, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}

func conflictMapToInterfaceMap(values map[string][]string) map[string]interface{} {
	out := make(map[string]interface{}, len(values))
	for key, value := range values {
		items := make([]interface{}, 0, len(value))
		for _, item := range value {
			items = append(items, item)
		}
		out[key] = items
	}
	return out
}

func appendUniqueString(items []string, value string) []string {
	for _, item := range items {
		if item == value {
			return items
		}
	}
	return append(items, value)
}

func valuesEquivalent(a, b interface{}) bool {
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

func maxTime(a, b time.Time) time.Time {
	if b.After(a) {
		return b
	}
	return a
}

// appendSourceID adds storageID to an entity's ComputedValues["source_ids"]
// list if it is not already present.
func appendSourceID(entity *models.Entity, storageID string) {
	if entity.ComputedValues == nil {
		entity.ComputedValues = make(map[string]interface{})
	}
	existing, _ := entity.ComputedValues["source_ids"].([]interface{})
	for _, v := range existing {
		if s, _ := v.(string); s == storageID {
			return // already present
		}
	}
	entity.ComputedValues["source_ids"] = append(existing, storageID)
}

// wireRelationships links entities of different types that share a common
// key-field value, records temporal edge changes, and rewrites the current
// materialized relationship graph in one pass.
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
// the operation bounded for large twins.
func commonKeyFieldNames(entitiesA, entitiesB []*models.Entity) []string {
	sampleA := entitiesA
	if len(sampleA) > 20 {
		sampleA = sampleA[:20]
	}
	sampleB := entitiesB
	if len(sampleB) > 20 {
		sampleB = sampleB[:20]
	}

	keysA := make(map[string]bool)
	for _, e := range sampleA {
		for _, kf := range detectKeyFields(e.Attributes) {
			keysA[kf] = true
		}
	}

	var shared []string
	seenB := make(map[string]bool)
	for _, e := range sampleB {
		for _, kf := range detectKeyFields(e.Attributes) {
			if keysA[kf] && !seenB[kf] {
				seenB[kf] = true
				shared = append(shared, kf)
			}
		}
	}
	return shared
}

// hasRelationship returns true if entity already has a relationship of the
// given type pointing to targetID.
func hasRelationship(e *models.Entity, relType, targetID string) bool {
	for _, r := range e.Relationships {
		if r.Type == relType && r.TargetID == targetID {
			return true
		}
	}
	return false
}

// toCamelCaseRel converts a field name like "student_id" → "StudentId"
// for use in relationship type names.
func toCamelCaseRel(s string) string {
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == '_' || r == '-' || r == ' '
	})
	var b strings.Builder
	for _, p := range parts {
		if len(p) == 0 {
			continue
		}
		b.WriteString(strings.ToUpper(p[:1]) + p[1:])
	}
	return b.String()
}

// detectKeyFields is re-exported here for use within the digitaltwin package.
// It returns attribute names that look like stable join keys.
func detectKeyFields(attributes map[string]interface{}) []string {
	keyNameSuffixes := []string{"id", "key", "code", "number", "uuid", "ref", "identifier", "no", "num", "email", "username", "token"}
	var keys []string
	for k := range attributes {
		lower := strings.ToLower(k)
		for _, suffix := range keyNameSuffixes {
			if strings.HasSuffix(lower, suffix) || lower == suffix {
				keys = append(keys, k)
				break
			}
		}
	}
	sort.Strings(keys)
	return keys
}

// keyValue returns a normalised string representation of an attribute value
// for equality comparison across data sources.
func keyValue(v interface{}) string {
	if v == nil {
		return ""
	}
	s := fmt.Sprintf("%v", v)
	if strings.Contains(s, ".") {
		s = strings.TrimRight(s, "0")
		s = strings.TrimRight(s, ".")
	}
	return strings.TrimSpace(s)
}

// inferEntityTypeFromCIR tries to infer the entity type from a CIR record
func inferEntityTypeFromCIR(cir *models.CIR, ontologyClasses []string) string {
	// Check CIR parameter first
	if v, ok := cir.GetParameter("entity_type"); ok {
		if s, ok := v.(string); ok && s != "" {
			return s
		}
	}

	// Match data keys against ontology class property names
	dataMap, err := cir.GetDataAsMap()
	if err != nil {
		return "unknown"
	}

	keys := make([]string, 0, len(dataMap))
	for k := range dataMap {
		keys = append(keys, strings.ToLower(k))
	}

	// Simple heuristic: look for class name hints in data keys
	for _, className := range ontologyClasses {
		lc := strings.ToLower(className)
		for _, k := range keys {
			if k == lc || k == "type" || k == "entity_type" {
				if v, ok := dataMap[k]; ok {
					if s, ok := v.(string); ok && strings.EqualFold(s, className) {
						return className
					}
				}
			}
		}
	}

	// Fall back to URI last path segment
	uri := cir.Source.URI
	if uri != "" {
		parts := strings.Split(strings.TrimRight(uri, "/"), "/")
		if len(parts) > 0 {
			last := parts[len(parts)-1]
			if last != "" {
				return last
			}
		}
	}

	return "default"
}

// StartCacheEviction runs a background goroutine that periodically deletes expired predictions.
// Call this once after creating the service: go svc.StartCacheEviction(ctx)
func (s *Service) StartCacheEviction(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			twins, err := s.store.ListDigitalTwins()
			if err != nil {
				log.Printf("cache eviction: failed to list digital twins: %v", err)
				continue
			}
			for _, twin := range twins {
				if err := s.store.DeleteExpiredPredictions(twin.ID); err != nil {
					log.Printf("cache eviction: error for twin %s: %v", twin.ID, err)
				}
			}
			log.Printf("cache eviction: completed tick for %d digital twins", len(twins))
		}
	}
}

// GetRelatedEntities returns entities related to the given entity via the specified relationship type.
// If relationshipType is empty, all relationships are traversed.
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
