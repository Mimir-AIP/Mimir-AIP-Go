package digitaltwin

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/mimir-aip/mimir-aip-go/pkg/analysis"
	"github.com/mimir-aip/mimir-aip-go/pkg/metadatastore"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"github.com/mimir-aip/mimir-aip-go/pkg/queue"
)

// Processor coordinates explicit, worker-backed twin processing runs.
type Processor struct {
	store           metadatastore.MetadataStore
	twinService     *Service
	analysisService *analysis.Service
	queue           *queue.Queue
}

func NewProcessor(store metadatastore.MetadataStore, twinService *Service, analysisService *analysis.Service, q *queue.Queue) *Processor {
	return &Processor{store: store, twinService: twinService, analysisService: analysisService, queue: q}
}

func (p *Processor) ListRuns(twinID string, limit int) ([]*models.TwinProcessingRun, error) {
	if twinID == "" {
		return nil, fmt.Errorf("digital_twin_id is required")
	}
	return p.store.ListTwinProcessingRunsByDigitalTwin(twinID, limit)
}

func (p *Processor) GetRun(twinID, runID string) (*models.TwinProcessingRun, error) {
	run, err := p.store.GetTwinProcessingRun(runID)
	if err != nil {
		return nil, err
	}
	if run.DigitalTwinID != twinID {
		return nil, fmt.Errorf("twin processing run %s does not belong to digital twin %s", runID, twinID)
	}
	return run, nil
}

func (p *Processor) ListAlerts(twinID string, limit int) ([]*models.AlertEvent, error) {
	if twinID == "" {
		return nil, fmt.Errorf("digital_twin_id is required")
	}
	return p.store.ListAlertEventsByDigitalTwin(twinID, limit)
}

func (p *Processor) ReviewAlert(twinID, alertID string, req *models.AlertApprovalRequest) (*models.AlertEvent, error) {
	if twinID == "" {
		return nil, fmt.Errorf("digital_twin_id is required")
	}
	if alertID == "" {
		return nil, fmt.Errorf("alert_id is required")
	}
	if req == nil {
		return nil, fmt.Errorf("alert approval request is required")
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}
	alert, err := p.store.GetAlertEvent(alertID)
	if err != nil {
		return nil, fmt.Errorf("failed to get alert event: %w", err)
	}
	if alert.DigitalTwinID != twinID {
		return nil, fmt.Errorf("alert event %s does not belong to digital twin %s", alertID, twinID)
	}
	if alert.ApprovalStatus != models.AlertApprovalStatusPending {
		return nil, fmt.Errorf("alert event %s is not awaiting approval", alertID)
	}
	resolvedAt := time.Now().UTC()
	alert.ApprovalResolvedAt = &resolvedAt
	alert.ApprovalActor = req.Actor
	alert.ApprovalNote = req.Note
	switch req.Decision {
	case models.AlertApprovalDecisionApprove:
		workTask, err := p.twinService.actionManager.TriggerApprovedAlertEvent(alert)
		if err != nil {
			return nil, err
		}
		alert.ApprovalStatus = models.AlertApprovalStatusApproved
		alert.TriggeredExportPipelineID = alert.RequestedExportPipelineID
		alert.TriggeredWorkTaskID = workTask.ID
	case models.AlertApprovalDecisionReject:
		alert.ApprovalStatus = models.AlertApprovalStatusRejected
	default:
		return nil, fmt.Errorf("unsupported approval decision %q", req.Decision)
	}
	if err := p.store.SaveAlertEvent(alert); err != nil {
		return nil, fmt.Errorf("failed to persist reviewed alert event: %w", err)
	}
	return alert, nil
}

func (p *Processor) RequestRun(twinID string, req *models.TwinProcessingRunCreateRequest) (*models.TwinProcessingRun, error) {
	if twinID == "" {
		return nil, fmt.Errorf("digital_twin_id is required")
	}
	if req == nil {
		return nil, fmt.Errorf("twin processing run request is required")
	}
	if p.queue == nil {
		return nil, fmt.Errorf("work queue is not configured")
	}
	twin, err := p.store.GetDigitalTwin(twinID)
	if err != nil {
		return nil, fmt.Errorf("failed to get digital twin: %w", err)
	}
	if active, err := p.store.GetActiveTwinProcessingRun(twinID); err != nil {
		return nil, err
	} else if active != nil {
		if !active.RerunRequested {
			active.RerunRequested = true
			if err := p.store.SaveTwinProcessingRun(active); err != nil {
				return nil, fmt.Errorf("failed to mark rerun_requested: %w", err)
			}
			return active, nil
		}
		return active, nil
	}

	now := time.Now().UTC()
	run := &models.TwinProcessingRun{
		ID:            uuid.New().String(),
		ProjectID:     twin.ProjectID,
		DigitalTwinID: twinID,
		Status:        models.TwinProcessingRunStatusQueued,
		TriggerType:   req.TriggerType,
		TriggerRef:    req.TriggerRef,
		AutomationID:  req.AutomationID,
		RequestedAt:   now,
		StageStates: map[string]models.TwinProcessingStageState{
			"sync":     {Status: models.TwinProcessingStageStatusQueued},
			"insights": {Status: models.TwinProcessingStageStatusQueued},
			"alerts":   {Status: models.TwinProcessingStageStatusQueued},
			"actions":  {Status: models.TwinProcessingStageStatusQueued},
		},
		Metrics: map[string]any{},
	}
	if err := p.store.SaveTwinProcessingRun(run); err != nil {
		return nil, fmt.Errorf("failed to save twin processing run: %w", err)
	}

	workTask := &models.WorkTask{
		ID:          uuid.New().String(),
		Type:        models.WorkTaskTypeDigitalTwinProcessing,
		Status:      models.WorkTaskStatusQueued,
		Priority:    4,
		SubmittedAt: now,
		ProjectID:   twin.ProjectID,
		TaskSpec: models.TaskSpec{
			ProjectID: twin.ProjectID,
			Parameters: map[string]any{
				"digital_twin_id":   twinID,
				"processing_run_id": run.ID,
				"trigger_type":      req.TriggerType,
				"trigger_ref":       req.TriggerRef,
				"automation_id":     req.AutomationID,
			},
		},
	}
	if err := p.queue.Enqueue(workTask); err != nil {
		run.Status = models.TwinProcessingRunStatusFailed
		run.CompletedAt = &now
		run.Error = fmt.Sprintf("failed to enqueue twin processing task: %v", err)
		if saveErr := p.store.SaveTwinProcessingRun(run); saveErr != nil {
			return nil, fmt.Errorf("failed to enqueue twin processing task: %v (also failed to persist run error: %w)", err, saveErr)
		}
		return nil, fmt.Errorf("failed to enqueue twin processing task: %w", err)
	}
	run.Metrics["work_task_id"] = workTask.ID
	if err := p.store.SaveTwinProcessingRun(run); err != nil {
		return nil, fmt.Errorf("failed to persist twin processing work task id: %w", err)
	}
	return run, nil
}

func (p *Processor) ExecuteRun(runID string) (*models.TwinProcessingRun, error) {
	if runID == "" {
		return nil, fmt.Errorf("run id is required")
	}
	run, err := p.store.GetTwinProcessingRun(runID)
	if err != nil {
		return nil, err
	}
	if run.Status == models.TwinProcessingRunStatusSuperseded {
		return run, nil
	}

	startedAt := time.Now().UTC()
	run.Status = models.TwinProcessingRunStatusRunning
	run.StartedAt = &startedAt
	markStageRunning(run, "sync", startedAt)
	markStageQueued(run, "insights")
	markStageQueued(run, "alerts")
	markStageQueued(run, "actions")
	if err := p.store.SaveTwinProcessingRun(run); err != nil {
		return nil, fmt.Errorf("failed to persist running twin processing run: %w", err)
	}

	if err := p.twinService.SyncWithStorage(run.DigitalTwinID); err != nil {
		return p.failRun(run, "sync", err)
	}
	markStageCompleted(run, "sync", time.Now().UTC(), nil)

	markStageRunning(run, "insights", time.Now().UTC())
	insightRun, insights, err := p.analysisService.GenerateInsightsForStorageIDs(run.ProjectID, p.twinService.storageIDsForTwin(run.DigitalTwinID))
	if err != nil {
		return p.failRun(run, "insights", err)
	}
	markStageCompleted(run, "insights", time.Now().UTC(), map[string]any{
		"insight_run_id": insightRun.ID,
		"insight_count":  len(insights),
	})
	markStageRunning(run, "alerts", time.Now().UTC())
	alertResult, err := p.evaluateAlertEvents(run)
	if err != nil {
		return p.failRun(run, "alerts", err)
	}
	markStageCompleted(run, "alerts", time.Now().UTC(), map[string]any{
		"alert_count":            alertResult.AlertCount,
		"pending_approval_count": alertResult.PendingApprovalCount,
	})

	markStageRunning(run, "actions", time.Now().UTC())
	markStageCompleted(run, "actions", time.Now().UTC(), map[string]any{
		"triggered_action_count": alertResult.TriggeredActionCount,
		"action_error_count":     alertResult.ActionErrorCount,
		"pending_approval_count": alertResult.PendingApprovalCount,
	})

	completedAt := time.Now().UTC()
	run.Status = models.TwinProcessingRunStatusCompleted
	run.CompletedAt = &completedAt
	run.Metrics["insight_count"] = len(insights)
	run.Metrics["insight_run_id"] = insightRun.ID
	run.Metrics["alert_count"] = alertResult.AlertCount
	run.Metrics["triggered_action_count"] = alertResult.TriggeredActionCount
	run.Metrics["action_error_count"] = alertResult.ActionErrorCount
	run.Metrics["pending_approval_count"] = alertResult.PendingApprovalCount
	if err := p.store.SaveTwinProcessingRun(run); err != nil {
		return nil, fmt.Errorf("failed to persist completed twin processing run: %w", err)
	}

	if run.RerunRequested {
		followUp := &models.TwinProcessingRunCreateRequest{
			TriggerType:  run.TriggerType,
			TriggerRef:   run.TriggerRef,
			AutomationID: run.AutomationID,
		}
		queued, err := p.RequestRun(run.DigitalTwinID, followUp)
		if err != nil {
			return nil, fmt.Errorf("completed run but failed to enqueue coalesced rerun: %w", err)
		}
		run.RerunRequested = false
		run.SupersededByRunID = queued.ID
		if err := p.store.SaveTwinProcessingRun(run); err != nil {
			return nil, fmt.Errorf("failed to persist coalesced rerun linkage: %w", err)
		}
	}
	return run, nil
}

func (p *Processor) failRun(run *models.TwinProcessingRun, stage string, cause error) (*models.TwinProcessingRun, error) {
	completedAt := time.Now().UTC()
	run.Status = models.TwinProcessingRunStatusFailed
	run.CompletedAt = &completedAt
	run.Error = cause.Error()
	markStageFailed(run, stage, completedAt, cause)
	if err := p.store.SaveTwinProcessingRun(run); err != nil {
		return nil, fmt.Errorf("failed to persist failed twin processing run: %v (additional save error: %w)", cause, err)
	}
	return nil, cause
}

func markStageQueued(run *models.TwinProcessingRun, stage string) {
	if run.StageStates == nil {
		run.StageStates = map[string]models.TwinProcessingStageState{}
	}
	run.StageStates[stage] = models.TwinProcessingStageState{Status: models.TwinProcessingStageStatusQueued}
}

func markStageRunning(run *models.TwinProcessingRun, stage string, at time.Time) {
	if run.StageStates == nil {
		run.StageStates = map[string]models.TwinProcessingStageState{}
	}
	state := run.StageStates[stage]
	state.Status = models.TwinProcessingStageStatusRunning
	state.StartedAt = &at
	state.CompletedAt = nil
	state.Error = ""
	run.StageStates[stage] = state
}

func markStageCompleted(run *models.TwinProcessingRun, stage string, at time.Time, metrics map[string]any) {
	if run.StageStates == nil {
		run.StageStates = map[string]models.TwinProcessingStageState{}
	}
	state := run.StageStates[stage]
	state.Status = models.TwinProcessingStageStatusCompleted
	if state.StartedAt == nil {
		state.StartedAt = &at
	}
	state.CompletedAt = &at
	state.Metrics = metrics
	state.Error = ""
	run.StageStates[stage] = state
}

func markStageFailed(run *models.TwinProcessingRun, stage string, at time.Time, cause error) {
	if run.StageStates == nil {
		run.StageStates = map[string]models.TwinProcessingStageState{}
	}
	state := run.StageStates[stage]
	state.Status = models.TwinProcessingStageStatusFailed
	if state.StartedAt == nil {
		state.StartedAt = &at
	}
	state.CompletedAt = &at
	state.Error = cause.Error()
	run.StageStates[stage] = state
}

type alertEvaluationResult struct {
	AlertCount           int
	TriggeredActionCount int
	ActionErrorCount     int
	PendingApprovalCount int
}

func (p *Processor) evaluateAlertEvents(run *models.TwinProcessingRun) (*alertEvaluationResult, error) {
	entities, err := p.store.ListEntitiesByDigitalTwin(run.DigitalTwinID)
	if err != nil {
		return nil, fmt.Errorf("failed to list entities for alert evaluation: %w", err)
	}
	actions, err := p.twinService.actionManager.ListEnabledActions(run.DigitalTwinID)
	if err != nil {
		return nil, fmt.Errorf("failed to list enabled actions for alert evaluation: %w", err)
	}
	result := &alertEvaluationResult{}
	for _, entity := range entities {
		for _, action := range actions {
			if !p.twinService.actionManager.MatchesEntityAction(action, entity) {
				continue
			}
			now := time.Now().UTC()
			alert := &models.AlertEvent{
				ID:              uuid.New().String(),
				ProjectID:       run.ProjectID,
				DigitalTwinID:   run.DigitalTwinID,
				ProcessingRunID: run.ID,
				AutomationID:    run.AutomationID,
				ActionID:        action.ID,
				Severity:        alertSeverityForAction(action),
				Category:        alertCategoryForAction(action),
				Title:           alertTitleForAction(action, entity),
				Message:         alertMessageForAction(action, entity),
				OntologyContext: map[string]any{
					"entity_id":   entity.ID,
					"entity_type": entity.Type,
					"attribute":   action.Condition.Attribute,
					"operator":    action.Condition.Operator,
				},
				EntityRefs:                []string{entity.ID},
				RequestedExportPipelineID: action.Trigger.PipelineID,
				RequestedTriggerParams:    cloneMap(action.Trigger.Parameters),
				CreatedAt:                 now,
			}
			if action.Trigger.ApprovalMode == models.ActionApprovalModeManual {
				alert.ApprovalStatus = models.AlertApprovalStatusPending
				alert.ApprovalRequestedAt = &now
				alert.Message = fmt.Sprintf("%s Awaiting manual approval.", alert.Message)
				result.PendingApprovalCount++
			} else {
				alert.ApprovalStatus = models.AlertApprovalStatusNotRequired
				workTask, triggerErr := p.twinService.actionManager.TriggerAction(action, map[string]any{
					"alert_id":          alert.ID,
					"processing_run_id": run.ID,
					"digital_twin_id":   run.DigitalTwinID,
					"entity_id":         entity.ID,
					"entity_type":       entity.Type,
					"alert_title":       alert.Title,
					"alert_message":     alert.Message,
					"alert_severity":    alert.Severity,
					"alert_category":    alert.Category,
				})
				if triggerErr != nil {
					result.ActionErrorCount++
					alert.Message = fmt.Sprintf("%s Trigger failed: %v", alert.Message, triggerErr)
				} else {
					result.TriggeredActionCount++
					alert.TriggeredExportPipelineID = action.Trigger.PipelineID
					alert.TriggeredWorkTaskID = workTask.ID
				}
			}
			if err := p.store.SaveAlertEvent(alert); err != nil {
				return nil, fmt.Errorf("failed to persist alert event: %w", err)
			}
			result.AlertCount++
		}
	}
	return result, nil
}

func cloneMap(input map[string]any) map[string]any {
	if len(input) == 0 {
		return nil
	}
	cloned := make(map[string]any, len(input))
	for key, value := range input {
		cloned[key] = value
	}
	return cloned
}

func alertSeverityForAction(action *models.Action) models.AlertSeverity {
	if action != nil && action.Trigger != nil {
		if raw, ok := action.Trigger.Parameters["alert_severity"].(string); ok {
			switch models.InsightSeverity(raw) {
			case models.InsightSeverityLow, models.InsightSeverityMedium, models.InsightSeverityHigh, models.InsightSeverityCritical:
				return models.AlertSeverity(raw)
			}
		}
	}
	return models.AlertSeverity(models.InsightSeverityHigh)
}

func alertCategoryForAction(action *models.Action) string {
	if action != nil && action.Trigger != nil {
		if raw, ok := action.Trigger.Parameters["alert_category"].(string); ok && raw != "" {
			return raw
		}
	}
	return "twin_action_match"
}

func alertTitleForAction(action *models.Action, entity *models.Entity) string {
	if action != nil && action.Trigger != nil {
		if raw, ok := action.Trigger.Parameters["alert_title"].(string); ok && raw != "" {
			return raw
		}
	}
	if action != nil && action.Name != "" {
		return action.Name
	}
	return fmt.Sprintf("Twin alert for entity %s", entity.ID)
}

func alertMessageForAction(action *models.Action, entity *models.Entity) string {
	if action != nil && action.Trigger != nil {
		if raw, ok := action.Trigger.Parameters["alert_message"].(string); ok && raw != "" {
			return raw
		}
	}
	if action != nil && action.Condition != nil {
		return fmt.Sprintf("Entity %s (%s) satisfied %s %s %v.", entity.ID, entity.Type, action.Condition.Attribute, action.Condition.Operator, action.Condition.Threshold)
	}
	return fmt.Sprintf("Entity %s (%s) triggered an alert.", entity.ID, entity.Type)
}
