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

// EvaluateActions evaluates all model-output-based actions for a digital twin against a prediction.
// This is called automatically after every Predict call.
func (m *ActionManager) EvaluateActions(twinID string, prediction *models.Prediction) error {
	actions, err := m.store.ListActionsByDigitalTwin(twinID)
	if err != nil {
		return fmt.Errorf("failed to list actions: %w", err)
	}

	for _, action := range actions {
		if !action.Enabled || action.Condition == nil {
			continue
		}
		// This path handles model-output conditions (ModelID set OR no Attribute set).
		if action.Condition.Attribute != "" {
			continue // handled by EvaluateEntityActions
		}
		if m.evaluatePredictionCondition(action.Condition, prediction) {
			if err := m.triggerAction(action); err != nil {
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
	actions, err := m.store.ListActionsByDigitalTwin(twinID)
	if err != nil {
		return fmt.Errorf("failed to list actions: %w", err)
	}

	for _, action := range actions {
		if !action.Enabled || action.Condition == nil {
			continue
		}
		// Only handle attribute-based conditions here.
		if action.Condition.Attribute == "" {
			continue
		}
		// Filter by entity type when specified.
		if action.Condition.EntityType != "" && action.Condition.EntityType != entity.Type {
			continue
		}
		if m.evaluateEntityCondition(action.Condition, entity) {
			if err := m.triggerAction(action); err != nil {
				fmt.Printf("Warning: failed to trigger action %s: %v\n", action.ID, err)
			}
		}
	}

	return nil
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
