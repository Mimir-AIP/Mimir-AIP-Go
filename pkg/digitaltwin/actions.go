package digitaltwin

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/mimir-aip/mimir-aip-go/pkg/metadatastore"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"github.com/mimir-aip/mimir-aip-go/pkg/queue"
)

// ActionManager handles conditional actions and triggers
type ActionManager struct {
	store metadatastore.MetadataStore
	queue *queue.Queue
}

// NewActionManager creates a new action manager
func NewActionManager(store metadatastore.MetadataStore, q *queue.Queue) *ActionManager {
	return &ActionManager{
		store: store,
		queue: q,
	}
}

// ListEnabledActions lists enabled actions for one twin.
func (m *ActionManager) ListEnabledActions(twinID string) ([]*models.Action, error) {
	actions, err := m.store.ListActionsByDigitalTwin(twinID)
	if err != nil {
		return nil, fmt.Errorf("failed to list actions: %w", err)
	}
	enabled := make([]*models.Action, 0, len(actions))
	for _, action := range actions {
		if action.Enabled && action.Condition != nil && action.Trigger != nil {
			enabled = append(enabled, action)
		}
	}
	return enabled, nil
}

// CreateAction creates a new conditional action
func (m *ActionManager) CreateAction(twin *models.DigitalTwin, req *models.ActionCreateRequest) (*models.Action, error) {
	now := time.Now().UTC()
	trigger := *req.Trigger
	if trigger.ApprovalMode == "" {
		trigger.ApprovalMode = models.ActionApprovalModeAutomatic
	}
	action := &models.Action{
		ID:            uuid.New().String(),
		DigitalTwinID: twin.ID,
		Name:          req.Name,
		Description:   req.Description,
		Enabled:       req.Enabled,
		Condition:     req.Condition,
		Trigger:       &trigger,
		TriggerCount:  0,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := m.store.SaveAction(action); err != nil {
		return nil, fmt.Errorf("failed to save action: %w", err)
	}

	return action, nil
}

// EvaluateActions evaluates all model-output-based actions for a digital twin against a prediction.
// This is called automatically after every Predict call.
func (m *ActionManager) EvaluateActions(twinID string, prediction *models.Prediction) error {
	actions, err := m.ListEnabledActions(twinID)
	if err != nil {
		return err
	}

	for _, action := range actions {
		// This path handles model-output conditions (ModelID set OR no Attribute set).
		if action.Condition.Attribute != "" {
			continue // handled by EvaluateEntityActions
		}
		if m.evaluatePredictionCondition(action.Condition, prediction) {
			if _, err := m.TriggerAction(action, nil); err != nil {
				fmt.Printf("Warning: failed to trigger action %s: %v\n", action.ID, err)
			}
		}
	}

	return nil
}

// EvaluateEntityActions evaluates all attribute-based actions for a digital twin
// against a single entity's current attribute values.
// This should be called whenever an entity is synced or updated.
func (m *ActionManager) EvaluateEntityActions(twinID string, entity *models.Entity) error {
	actions, err := m.ListEnabledActions(twinID)
	if err != nil {
		return err
	}

	for _, action := range actions {
		if !m.MatchesEntityAction(action, entity) {
			continue
		}
		if _, err := m.TriggerAction(action, nil); err != nil {
			fmt.Printf("Warning: failed to trigger action %s: %v\n", action.ID, err)
		}
	}

	return nil
}

// MatchesEntityAction reports whether an entity satisfies one action's attribute-based condition.
func (m *ActionManager) MatchesEntityAction(action *models.Action, entity *models.Entity) bool {
	if action == nil || action.Condition == nil || entity == nil {
		return false
	}
	if action.Condition.Attribute == "" {
		return false
	}
	if action.Condition.EntityType != "" && action.Condition.EntityType != entity.Type {
		return false
	}
	return m.evaluateEntityCondition(action.Condition, entity)
}

// evaluatePredictionCondition checks if a model-output action condition is met.
func (m *ActionManager) evaluatePredictionCondition(condition *models.ActionCondition, prediction *models.Prediction) bool {
	if condition.ModelID != "" && condition.ModelID != prediction.ModelID {
		return false
	}

	var value float64
	switch v := prediction.Output.(type) {
	case float64:
		value = v
	case int:
		value = float64(v)
	case int64:
		value = float64(v)
	default:
		return false
	}

	threshold, ok := m.toFloat64(condition.Threshold)
	if !ok {
		return false
	}

	return m.compareFloats(value, condition.Operator, threshold)
}

// evaluateEntityCondition checks if an attribute-based action condition is met
// by inspecting the entity's current attribute values.
func (m *ActionManager) evaluateEntityCondition(condition *models.ActionCondition, entity *models.Entity) bool {
	attrVal, ok := entity.Attributes[condition.Attribute]
	if !ok {
		return false
	}

	threshold, ok := m.toFloat64(condition.Threshold)
	if !ok {
		// Fall back to string comparison for eq/ne
		vs := fmt.Sprintf("%v", attrVal)
		es := fmt.Sprintf("%v", condition.Threshold)
		switch condition.Operator {
		case "eq":
			return vs == es
		case "ne":
			return vs != es
		}
		return false
	}

	value, ok := m.toFloat64(attrVal)
	if !ok {
		return false
	}

	return m.compareFloats(value, condition.Operator, threshold)
}

// compareFloats applies a comparison operator to two float64 values.
func (m *ActionManager) compareFloats(value float64, operator string, threshold float64) bool {
	switch operator {
	case "gt":
		return value > threshold
	case "gte":
		return value >= threshold
	case "lt":
		return value < threshold
	case "lte":
		return value <= threshold
	case "eq":
		return value == threshold
	case "ne":
		return value != threshold
	default:
		return false
	}
}

// TriggerAction triggers an automatic action by submitting an export pipeline execution task.
func (m *ActionManager) TriggerAction(action *models.Action, extraParameters map[string]any) (*models.WorkTask, error) {
	if action == nil || action.Trigger == nil {
		return nil, fmt.Errorf("action trigger is required")
	}
	if action.Trigger.ApprovalMode == models.ActionApprovalModeManual {
		return nil, fmt.Errorf("action %s requires manual approval", action.ID)
	}
	twin, err := m.store.GetDigitalTwin(action.DigitalTwinID)
	if err != nil {
		return nil, fmt.Errorf("failed to get digital twin for action %s: %w", action.ID, err)
	}
	workTask, err := m.enqueuePipelineExecution(twin, action.Trigger.PipelineID, action.Trigger.Parameters, extraParameters)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	if err := m.markActionTriggered(action.ID, &now); err != nil {
		return nil, err
	}
	return workTask, nil
}

// TriggerApprovedAlertEvent executes a previously approved pending alert export.
func (m *ActionManager) TriggerApprovedAlertEvent(alert *models.AlertEvent) (*models.WorkTask, error) {
	if alert == nil {
		return nil, fmt.Errorf("alert event is required")
	}
	if alert.RequestedExportPipelineID == "" {
		return nil, fmt.Errorf("alert %s has no requested export pipeline", alert.ID)
	}
	twin, err := m.store.GetDigitalTwin(alert.DigitalTwinID)
	if err != nil {
		return nil, fmt.Errorf("failed to get digital twin for alert %s: %w", alert.ID, err)
	}
	workTask, err := m.enqueuePipelineExecution(twin, alert.RequestedExportPipelineID, alert.RequestedTriggerParams, map[string]any{
		"alert_id":          alert.ID,
		"processing_run_id": alert.ProcessingRunID,
		"digital_twin_id":   alert.DigitalTwinID,
		"approval_status":   string(alert.ApprovalStatus),
	})
	if err != nil {
		return nil, err
	}
	if alert.ActionID != "" {
		now := time.Now().UTC()
		if err := m.markActionTriggered(alert.ActionID, &now); err != nil {
			return nil, err
		}
	}
	return workTask, nil
}

func (m *ActionManager) enqueuePipelineExecution(twin *models.DigitalTwin, pipelineID string, baseParameters map[string]any, extraParameters map[string]any) (*models.WorkTask, error) {
	if twin == nil {
		return nil, fmt.Errorf("digital twin is required")
	}
	if pipelineID == "" {
		return nil, fmt.Errorf("pipeline id is required")
	}
	pipeline, err := m.store.GetPipeline(pipelineID)
	if err != nil {
		return nil, fmt.Errorf("failed to get action pipeline %s: %w", pipelineID, err)
	}
	if pipeline.ProjectID != twin.ProjectID {
		return nil, fmt.Errorf("action pipeline %s belongs to project %s, not %s", pipeline.ID, pipeline.ProjectID, twin.ProjectID)
	}
	if pipeline.Type != models.PipelineTypeOutput {
		return nil, fmt.Errorf("action pipeline %s must be an output pipeline", pipeline.ID)
	}
	parameters := make(map[string]any, len(baseParameters)+len(extraParameters))
	for key, value := range baseParameters {
		parameters[key] = value
	}
	for key, value := range extraParameters {
		parameters[key] = value
	}
	workTask := &models.WorkTask{
		ID:          uuid.New().String(),
		Type:        models.WorkTaskTypePipelineExecution,
		Priority:    8,
		Status:      models.WorkTaskStatusQueued,
		SubmittedAt: time.Now().UTC(),
		ProjectID:   twin.ProjectID,
		TaskSpec: models.TaskSpec{
			ProjectID:  twin.ProjectID,
			PipelineID: pipelineID,
			Parameters: parameters,
		},
	}
	if err := m.queue.Enqueue(workTask); err != nil {
		return nil, fmt.Errorf("failed to enqueue pipeline execution: %w", err)
	}
	return workTask, nil
}

func (m *ActionManager) markActionTriggered(actionID string, at *time.Time) error {
	if actionID == "" || at == nil {
		return nil
	}
	action, err := m.store.GetAction(actionID)
	if err != nil {
		return fmt.Errorf("failed to reload action %s: %w", actionID, err)
	}
	action.LastTriggered = at
	action.TriggerCount++
	action.UpdatedAt = *at
	if err := m.store.SaveAction(action); err != nil {
		return fmt.Errorf("failed to update action %s: %w", actionID, err)
	}
	return nil
}

// toFloat64 converts an interface{} to float64
func (m *ActionManager) toFloat64(val interface{}) (float64, bool) {
	switch v := val.(type) {
	case float64:
		return v, true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	default:
		return 0, false
	}
}
