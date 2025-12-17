package DigitalTwin

import (
	"fmt"
	"time"
)

// EventType constants for common event types
const (
	// Resource events
	EventResourceUnavailable    = "resource.unavailable"
	EventResourceAvailable      = "resource.available"
	EventResourceCapacityChange = "resource.capacity_change"
	EventResourceAdded          = "resource.added"
	EventResourceRemoved        = "resource.removed"

	// Demand events
	EventDemandSurge         = "demand.surge"
	EventDemandDrop          = "demand.drop"
	EventDemandPatternChange = "demand.pattern_change"

	// Process events
	EventProcessDelay        = "process.delay"
	EventProcessFailure      = "process.failure"
	EventProcessOptimization = "process.optimization"
	EventProcessStarted      = "process.started"
	EventProcessCompleted    = "process.completed"

	// Policy events
	EventPolicyChange           = "policy.change"
	EventPolicyConstraintAdd    = "policy.constraint_add"
	EventPolicyConstraintRemove = "policy.constraint_remove"

	// External events
	EventExternalMarketShift      = "external.market_shift"
	EventExternalRegulatoryChange = "external.regulatory_change"
	EventExternalCompetitorAction = "external.competitor_action"
	EventExternalSupplyDisruption = "external.supply_disruption"
)

// SeverityLevel constants
const (
	SeverityLow      = "low"
	SeverityMedium   = "medium"
	SeverityHigh     = "high"
	SeverityCritical = "critical"
)

// EventHandler is the interface for processing events
type EventHandler interface {
	// HandleEvent processes an event and returns the resulting state changes
	HandleEvent(event *SimulationEvent, entity *TwinEntity, state *TwinState, twin *DigitalTwin) ([]StateChange, error)

	// CanHandle checks if this handler can process the given event type
	CanHandle(eventType string) bool

	// GetEventTypes returns the list of event types this handler supports
	GetEventTypes() []string
}

// StateChange represents a change to an entity's state
type StateChange struct {
	EntityURI   string      `json:"entity_uri"`
	Field       string      `json:"field"` // "status", "capacity", "utilization", "available", "metrics.*"
	OldValue    interface{} `json:"old_value"`
	NewValue    interface{} `json:"new_value"`
	Timestamp   time.Time   `json:"timestamp"`
	Reason      string      `json:"reason"`
	Propagated  bool        `json:"propagated"` // True if change resulted from propagation
	SourceEvent string      `json:"source_event"`
}

// EventProcessor manages event execution
type EventProcessor struct {
	handlers map[string]EventHandler
}

// NewEventProcessor creates a new event processor
func NewEventProcessor() *EventProcessor {
	ep := &EventProcessor{
		handlers: make(map[string]EventHandler),
	}

	// Register default handlers
	ep.RegisterHandler(&ResourceEventHandler{})
	ep.RegisterHandler(&DemandEventHandler{})
	ep.RegisterHandler(&ProcessEventHandler{})
	ep.RegisterHandler(&PolicyEventHandler{})
	ep.RegisterHandler(&ExternalEventHandler{})

	return ep
}

// RegisterHandler adds a new event handler
func (ep *EventProcessor) RegisterHandler(handler EventHandler) {
	for _, eventType := range handler.GetEventTypes() {
		ep.handlers[eventType] = handler
	}
}

// ProcessEvent executes an event and returns state changes
func (ep *EventProcessor) ProcessEvent(event *SimulationEvent, twin *DigitalTwin, state *TwinState) ([]StateChange, error) {
	handler, exists := ep.handlers[event.Type]
	if !exists {
		return nil, fmt.Errorf("no handler registered for event type: %s", event.Type)
	}

	entity := twin.GetEntity(event.TargetURI)
	if entity == nil {
		return nil, fmt.Errorf("target entity not found: %s", event.TargetURI)
	}

	changes, err := handler.HandleEvent(event, entity, state, twin)
	if err != nil {
		return nil, fmt.Errorf("error handling event: %w", err)
	}

	// Apply state changes
	for _, change := range changes {
		if err := ep.applyStateChange(change, state); err != nil {
			return nil, fmt.Errorf("error applying state change: %w", err)
		}
	}

	return changes, nil
}

// applyStateChange applies a state change to the twin state
func (ep *EventProcessor) applyStateChange(change StateChange, state *TwinState) error {
	entityState, exists := state.GetEntityState(change.EntityURI)
	if !exists {
		return fmt.Errorf("entity not found in state: %s", change.EntityURI)
	}

	switch change.Field {
	case "status":
		entityState.Status = change.NewValue.(string)
	case "capacity":
		entityState.Capacity = change.NewValue.(float64)
	case "utilization":
		entityState.Utilization = change.NewValue.(float64)
	case "available":
		entityState.Available = change.NewValue.(bool)
	default:
		// Handle metrics fields (e.g., "metrics.throughput")
		if len(change.Field) > 8 && change.Field[:8] == "metrics." {
			metricName := change.Field[8:]
			if entityState.Metrics == nil {
				entityState.Metrics = make(map[string]float64)
			}
			entityState.Metrics[metricName] = change.NewValue.(float64)
		} else {
			return fmt.Errorf("unknown field: %s", change.Field)
		}
	}

	state.UpdateEntityState(change.EntityURI, entityState)
	return nil
}

// PropagateImpact propagates event impact through relationships
func (ep *EventProcessor) PropagateImpact(event *SimulationEvent, twin *DigitalTwin, state *TwinState, changes []StateChange) ([]StateChange, error) {
	var propagatedChanges []StateChange

	// Get all relationships for the target entity
	relationships := twin.GetRelationships(event.TargetURI)

	for _, rule := range event.Impact.PropagationRules {
		for _, rel := range relationships {
			// Check if relationship type matches
			if rel.Type != rule.RelationshipType {
				continue
			}

			// Determine the related entity (propagate to the other end of the relationship)
			relatedURI := rel.TargetURI
			if rel.TargetURI == event.TargetURI {
				relatedURI = rel.SourceURI
			}

			// Check if related entity exists in state
			relatedState, exists := state.GetEntityState(relatedURI)
			if !exists {
				continue
			}

			// Apply propagation based on original changes
			for _, change := range changes {
				if change.EntityURI != event.TargetURI {
					continue
				}

				// Calculate propagated impact
				propagatedChange := StateChange{
					EntityURI:   relatedURI,
					Field:       change.Field,
					OldValue:    relatedState.Utilization,
					Timestamp:   time.Now(),
					Reason:      fmt.Sprintf("Propagated from %s via %s", event.TargetURI, rel.Type),
					Propagated:  true,
					SourceEvent: event.ID,
				}

				// Apply impact multiplier
				if change.Field == "utilization" {
					oldUtil := relatedState.Utilization
					changeAmount := change.NewValue.(float64) - change.OldValue.(float64)
					newUtil := oldUtil + (changeAmount * rule.ImpactMultiplier * rel.Strength)

					// Clamp to valid range
					if newUtil < 0 {
						newUtil = 0
					}
					if newUtil > 1.0 {
						newUtil = 1.0
						// Mark entity as overloaded if utilization exceeds capacity
						if relatedState.Status != "failed" {
							relatedState.Status = "degraded"
						}
					}

					propagatedChange.NewValue = newUtil
					propagatedChanges = append(propagatedChanges, propagatedChange)

					// Apply the change
					relatedState.Utilization = newUtil
					state.UpdateEntityState(relatedURI, relatedState)
				}
			}
		}
	}

	return propagatedChanges, nil
}

// ResourceEventHandler handles resource-related events
type ResourceEventHandler struct{}

func (h *ResourceEventHandler) GetEventTypes() []string {
	return []string{
		EventResourceUnavailable,
		EventResourceAvailable,
		EventResourceCapacityChange,
		EventResourceAdded,
		EventResourceRemoved,
	}
}

func (h *ResourceEventHandler) CanHandle(eventType string) bool {
	for _, et := range h.GetEventTypes() {
		if et == eventType {
			return true
		}
	}
	return false
}

func (h *ResourceEventHandler) HandleEvent(event *SimulationEvent, entity *TwinEntity, state *TwinState, twin *DigitalTwin) ([]StateChange, error) {
	var changes []StateChange
	entityState, _ := state.GetEntityState(entity.URI)

	switch event.Type {
	case EventResourceUnavailable:
		changes = append(changes, StateChange{
			EntityURI:   entity.URI,
			Field:       "available",
			OldValue:    entityState.Available,
			NewValue:    false,
			Timestamp:   time.Now(),
			Reason:      fmt.Sprintf("Resource unavailable: %v", event.Parameters["reason"]),
			SourceEvent: event.ID,
		})
		changes = append(changes, StateChange{
			EntityURI:   entity.URI,
			Field:       "status",
			OldValue:    entityState.Status,
			NewValue:    "inactive",
			Timestamp:   time.Now(),
			Reason:      "Resource marked unavailable",
			SourceEvent: event.ID,
		})

	case EventResourceAvailable:
		changes = append(changes, StateChange{
			EntityURI:   entity.URI,
			Field:       "available",
			OldValue:    entityState.Available,
			NewValue:    true,
			Timestamp:   time.Now(),
			Reason:      "Resource restored",
			SourceEvent: event.ID,
		})
		changes = append(changes, StateChange{
			EntityURI:   entity.URI,
			Field:       "status",
			OldValue:    entityState.Status,
			NewValue:    "active",
			Timestamp:   time.Now(),
			Reason:      "Resource marked available",
			SourceEvent: event.ID,
		})

	case EventResourceCapacityChange:
		multiplier, ok := event.Parameters["multiplier"].(float64)
		if !ok {
			multiplier = 1.0
		}
		newCapacity := entityState.Capacity * multiplier

		changes = append(changes, StateChange{
			EntityURI:   entity.URI,
			Field:       "capacity",
			OldValue:    entityState.Capacity,
			NewValue:    newCapacity,
			Timestamp:   time.Now(),
			Reason:      fmt.Sprintf("Capacity changed by %.2fx", multiplier),
			SourceEvent: event.ID,
		})
	}

	return changes, nil
}

// DemandEventHandler handles demand-related events
type DemandEventHandler struct{}

func (h *DemandEventHandler) GetEventTypes() []string {
	return []string{
		EventDemandSurge,
		EventDemandDrop,
		EventDemandPatternChange,
	}
}

func (h *DemandEventHandler) CanHandle(eventType string) bool {
	for _, et := range h.GetEventTypes() {
		if et == eventType {
			return true
		}
	}
	return false
}

func (h *DemandEventHandler) HandleEvent(event *SimulationEvent, entity *TwinEntity, state *TwinState, twin *DigitalTwin) ([]StateChange, error) {
	var changes []StateChange
	entityState, _ := state.GetEntityState(entity.URI)

	switch event.Type {
	case EventDemandSurge:
		increaseFactor, ok := event.Parameters["increase_factor"].(float64)
		if !ok {
			increaseFactor = 2.0
		}
		newUtilization := entityState.Utilization * increaseFactor
		if newUtilization > 1.0 {
			newUtilization = 1.0
		}

		changes = append(changes, StateChange{
			EntityURI:   entity.URI,
			Field:       "utilization",
			OldValue:    entityState.Utilization,
			NewValue:    newUtilization,
			Timestamp:   time.Now(),
			Reason:      fmt.Sprintf("Demand surge: %.2fx increase", increaseFactor),
			SourceEvent: event.ID,
		})

		// Update status if overloaded
		if newUtilization >= 0.95 {
			changes = append(changes, StateChange{
				EntityURI:   entity.URI,
				Field:       "status",
				OldValue:    entityState.Status,
				NewValue:    "degraded",
				Timestamp:   time.Now(),
				Reason:      "Overutilized due to demand surge",
				SourceEvent: event.ID,
			})
		}

	case EventDemandDrop:
		decreaseFactor, ok := event.Parameters["decrease_factor"].(float64)
		if !ok {
			decreaseFactor = 0.5
		}
		newUtilization := entityState.Utilization * decreaseFactor

		changes = append(changes, StateChange{
			EntityURI:   entity.URI,
			Field:       "utilization",
			OldValue:    entityState.Utilization,
			NewValue:    newUtilization,
			Timestamp:   time.Now(),
			Reason:      fmt.Sprintf("Demand drop: %.2fx decrease", decreaseFactor),
			SourceEvent: event.ID,
		})
	}

	return changes, nil
}

// ProcessEventHandler handles process-related events
type ProcessEventHandler struct{}

func (h *ProcessEventHandler) GetEventTypes() []string {
	return []string{
		EventProcessDelay,
		EventProcessFailure,
		EventProcessOptimization,
		EventProcessStarted,
		EventProcessCompleted,
	}
}

func (h *ProcessEventHandler) CanHandle(eventType string) bool {
	for _, et := range h.GetEventTypes() {
		if et == eventType {
			return true
		}
	}
	return false
}

func (h *ProcessEventHandler) HandleEvent(event *SimulationEvent, entity *TwinEntity, state *TwinState, twin *DigitalTwin) ([]StateChange, error) {
	var changes []StateChange
	entityState, _ := state.GetEntityState(entity.URI)

	switch event.Type {
	case EventProcessFailure:
		changes = append(changes, StateChange{
			EntityURI:   entity.URI,
			Field:       "status",
			OldValue:    entityState.Status,
			NewValue:    "failed",
			Timestamp:   time.Now(),
			Reason:      fmt.Sprintf("Process failure: %v", event.Parameters["reason"]),
			SourceEvent: event.ID,
		})
		changes = append(changes, StateChange{
			EntityURI:   entity.URI,
			Field:       "available",
			OldValue:    entityState.Available,
			NewValue:    false,
			Timestamp:   time.Now(),
			Reason:      "Process failure",
			SourceEvent: event.ID,
		})

	case EventProcessOptimization:
		efficiencyGain, ok := event.Parameters["efficiency_gain"].(float64)
		if !ok {
			efficiencyGain = 0.2
		}
		newUtilization := entityState.Utilization * (1.0 - efficiencyGain)

		changes = append(changes, StateChange{
			EntityURI:   entity.URI,
			Field:       "utilization",
			OldValue:    entityState.Utilization,
			NewValue:    newUtilization,
			Timestamp:   time.Now(),
			Reason:      fmt.Sprintf("Process optimized: %.1f%% efficiency gain", efficiencyGain*100),
			SourceEvent: event.ID,
		})
	}

	return changes, nil
}

// PolicyEventHandler handles policy-related events
type PolicyEventHandler struct{}

func (h *PolicyEventHandler) GetEventTypes() []string {
	return []string{
		EventPolicyChange,
		EventPolicyConstraintAdd,
		EventPolicyConstraintRemove,
	}
}

func (h *PolicyEventHandler) CanHandle(eventType string) bool {
	for _, et := range h.GetEventTypes() {
		if et == eventType {
			return true
		}
	}
	return false
}

func (h *PolicyEventHandler) HandleEvent(event *SimulationEvent, entity *TwinEntity, state *TwinState, twin *DigitalTwin) ([]StateChange, error) {
	var changes []StateChange
	entityState, _ := state.GetEntityState(entity.URI)

	switch event.Type {
	case EventPolicyConstraintAdd:
		// Constraints typically reduce capacity or increase utilization
		impact, ok := event.Parameters["capacity_impact"].(float64)
		if !ok {
			impact = 0.9 // 10% reduction
		}
		newCapacity := entityState.Capacity * impact

		changes = append(changes, StateChange{
			EntityURI:   entity.URI,
			Field:       "capacity",
			OldValue:    entityState.Capacity,
			NewValue:    newCapacity,
			Timestamp:   time.Now(),
			Reason:      fmt.Sprintf("Policy constraint added: %v", event.Parameters["constraint"]),
			SourceEvent: event.ID,
		})
	}

	return changes, nil
}

// ExternalEventHandler handles external events
type ExternalEventHandler struct{}

func (h *ExternalEventHandler) GetEventTypes() []string {
	return []string{
		EventExternalMarketShift,
		EventExternalRegulatoryChange,
		EventExternalCompetitorAction,
		EventExternalSupplyDisruption,
	}
}

func (h *ExternalEventHandler) CanHandle(eventType string) bool {
	for _, et := range h.GetEventTypes() {
		if et == eventType {
			return true
		}
	}
	return false
}

func (h *ExternalEventHandler) HandleEvent(event *SimulationEvent, entity *TwinEntity, state *TwinState, twin *DigitalTwin) ([]StateChange, error) {
	var changes []StateChange
	entityState, _ := state.GetEntityState(entity.URI)

	switch event.Type {
	case EventExternalSupplyDisruption:
		// Supply disruptions affect availability and capacity
		changes = append(changes, StateChange{
			EntityURI:   entity.URI,
			Field:       "available",
			OldValue:    entityState.Available,
			NewValue:    false,
			Timestamp:   time.Now(),
			Reason:      fmt.Sprintf("Supply disruption: %v", event.Parameters["reason"]),
			SourceEvent: event.ID,
		})

	case EventExternalMarketShift:
		// Market shifts affect demand/utilization
		impact, ok := event.Parameters["demand_impact"].(float64)
		if !ok {
			impact = 1.5
		}
		newUtilization := entityState.Utilization * impact
		if newUtilization > 1.0 {
			newUtilization = 1.0
		}

		changes = append(changes, StateChange{
			EntityURI:   entity.URI,
			Field:       "utilization",
			OldValue:    entityState.Utilization,
			NewValue:    newUtilization,
			Timestamp:   time.Now(),
			Reason:      fmt.Sprintf("Market shift: %.2fx demand impact", impact),
			SourceEvent: event.ID,
		})
	}

	return changes, nil
}

// GenerateEventID creates a unique event ID
func GenerateEventID(eventType string, timestamp int) string {
	return fmt.Sprintf("%s_%d_%d", eventType, timestamp, time.Now().UnixNano())
}

// CreateEvent is a helper to create a basic simulation event
func CreateEvent(eventType, targetURI string, timestamp int, parameters map[string]interface{}) *SimulationEvent {
	return &SimulationEvent{
		ID:         GenerateEventID(eventType, timestamp),
		Type:       eventType,
		TargetURI:  targetURI,
		Timestamp:  timestamp,
		Parameters: parameters,
		Impact: EventImpact{
			AffectedEntities: []string{targetURI},
			StateChanges:     make(map[string]interface{}),
			PropagationRules: []PropagationRule{},
			Severity:         SeverityMedium,
		},
	}
}

// WithPropagation adds propagation rules to an event
func (e *SimulationEvent) WithPropagation(relationshipType string, multiplier float64, delay int) *SimulationEvent {
	e.Impact.PropagationRules = append(e.Impact.PropagationRules, PropagationRule{
		RelationshipType: relationshipType,
		ImpactMultiplier: multiplier,
		Delay:            delay,
		Condition:        make(map[string]interface{}),
	})
	return e
}

// WithSeverity sets the severity of an event
func (e *SimulationEvent) WithSeverity(severity string) *SimulationEvent {
	e.Impact.Severity = severity
	return e
}
