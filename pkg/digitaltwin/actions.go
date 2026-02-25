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

// CreateAction creates a new conditional action
func (m *ActionManager) CreateAction(twin *models.DigitalTwin, req *models.ActionCreateRequest) (*models.Action, error) {
	now := time.Now().UTC()
	action := &models.Action{
		ID:            uuid.New().String(),
		DigitalTwinID: twin.ID,
		Name:          req.Name,
		Description:   req.Description,
		Enabled:       req.Enabled,
		Condition:     req.Condition,
		Trigger:       req.Trigger,
		TriggerCount:  0,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := m.store.SaveAction(action); err != nil {
		return nil, fmt.Errorf("failed to save action: %w", err)
	}

	return action, nil
}

// EvaluateActions evaluates all actions for a digital twin against a prediction
func (m *ActionManager) EvaluateActions(twinID string, prediction *models.Prediction) error {
	// Get all enabled actions for this digital twin
	actions, err := m.store.ListActionsByDigitalTwin(twinID)
	if err != nil {
		return fmt.Errorf("failed to list actions: %w", err)
	}

	for _, action := range actions {
		if !action.Enabled {
			continue
		}

		// Check if action condition is met
		if m.evaluateCondition(action.Condition, prediction) {
			// Trigger the action
			if err := m.triggerAction(action); err != nil {
				fmt.Printf("Warning: failed to trigger action %s: %v\n", action.ID, err)
				continue
			}
		}
	}

	return nil
}

// evaluateCondition checks if an action condition is met by a prediction
func (m *ActionManager) evaluateCondition(condition *models.ActionCondition, prediction *models.Prediction) bool {
	// Check if condition is for this model
	if condition.ModelID != "" && condition.ModelID != prediction.ModelID {
		return false
	}

	// Get the value to compare
	var value float64
	switch v := prediction.Output.(type) {
	case float64:
		value = v
	case int:
		value = float64(v)
	case int64:
		value = float64(v)
	default:
		// Can't compare non-numeric values
		return false
	}

	// Get threshold
	threshold, ok := m.toFloat64(condition.Threshold)
	if !ok {
		return false
	}

	// Evaluate operator
	switch condition.Operator {
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

// triggerAction triggers an action by submitting a pipeline execution task
func (m *ActionManager) triggerAction(action *models.Action) error {
	// Create a work task to execute the pipeline
	workTask := &models.WorkTask{
		ID:          uuid.New().String(),
		Type:        models.WorkTaskTypePipelineExecution,
		Priority:    8, // High priority for action triggers
		Status:      models.WorkTaskStatusQueued,
		SubmittedAt: time.Now().UTC(),
		TaskSpec: models.TaskSpec{
			PipelineID: action.Trigger.PipelineID,
			Parameters: action.Trigger.Parameters,
		},
	}

	// Enqueue the task
	if err := m.queue.Enqueue(workTask); err != nil {
		return fmt.Errorf("failed to enqueue pipeline execution: %w", err)
	}

	// Update action trigger count and last triggered time
	now := time.Now().UTC()
	action.LastTriggered = &now
	action.TriggerCount++
	action.UpdatedAt = now

	if err := m.store.SaveAction(action); err != nil {
		return fmt.Errorf("failed to update action: %w", err)
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
