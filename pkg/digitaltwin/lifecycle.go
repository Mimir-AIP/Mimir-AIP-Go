package digitaltwin

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"time"
)

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
