package DigitalTwin

import (
	"encoding/json"
	"time"
)

// DigitalTwin represents a business model created from a knowledge graph
type DigitalTwin struct {
	ID            string                 `json:"id"`
	OntologyID    string                 `json:"ontology_id"`
	Name          string                 `json:"name"`
	Description   string                 `json:"description"`
	ModelType     string                 `json:"model_type"` // "organization", "department", "process", "individual"
	BaseState     map[string]interface{} `json:"base_state"`
	Entities      []TwinEntity           `json:"entities"`
	Relationships []TwinRelationship     `json:"relationships"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
}

// TwinEntity represents an entity in the digital twin
type TwinEntity struct {
	URI        string                 `json:"uri"`
	Type       string                 `json:"type"`
	Label      string                 `json:"label"`
	Properties map[string]interface{} `json:"properties"`
	State      EntityState            `json:"state"`
}

// EntityState tracks dynamic entity properties during simulation
type EntityState struct {
	Status      string             `json:"status"` // "active", "inactive", "degraded", "failed"
	Capacity    float64            `json:"capacity"`
	Utilization float64            `json:"utilization"` // 0.0 to 1.0
	Available   bool               `json:"available"`
	Metrics     map[string]float64 `json:"metrics"`
	LastUpdated time.Time          `json:"last_updated"`
}

// TwinRelationship represents a relationship between entities
type TwinRelationship struct {
	ID         string                 `json:"id"`
	SourceURI  string                 `json:"source_uri"`
	TargetURI  string                 `json:"target_uri"`
	Type       string                 `json:"type"` // relationship predicate
	Properties map[string]interface{} `json:"properties"`
	Strength   float64                `json:"strength"` // 0.0 to 1.0, for impact propagation
}

// SimulationScenario defines a what-if scenario
type SimulationScenario struct {
	ID          string            `json:"id"`
	TwinID      string            `json:"twin_id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Type        string            `json:"scenario_type"` // "supply_shock", "demand_surge", etc.
	Events      []SimulationEvent `json:"events"`
	Duration    int               `json:"duration"` // Duration in simulation steps
	CreatedAt   time.Time         `json:"created_at"`
}

// SimulationEvent represents a discrete event in a scenario
type SimulationEvent struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"` // e.g., "resource.unavailable"
	TargetURI  string                 `json:"target_uri"`
	Timestamp  int                    `json:"timestamp"` // Step number when event occurs
	Parameters map[string]interface{} `json:"parameters"`
	Impact     EventImpact            `json:"impact"`
}

// EventImpact defines how an event affects the twin
type EventImpact struct {
	AffectedEntities []string               `json:"affected_entities"`
	StateChanges     map[string]interface{} `json:"state_changes"`
	PropagationRules []PropagationRule      `json:"propagation_rules"`
	Severity         string                 `json:"severity"` // "low", "medium", "high", "critical"
}

// PropagationRule defines how an event impact propagates through relationships
type PropagationRule struct {
	RelationshipType string                 `json:"relationship_type"`
	ImpactMultiplier float64                `json:"impact_multiplier"` // How much impact carries over
	Delay            int                    `json:"delay"`             // Steps before propagation occurs
	Condition        map[string]interface{} `json:"condition"`         // Optional condition for propagation
}

// SimulationRun tracks the execution of a scenario
type SimulationRun struct {
	ID           string                 `json:"id"`
	ScenarioID   string                 `json:"scenario_id"`
	Status       string                 `json:"status"` // "pending", "running", "completed", "failed"
	StartTime    time.Time              `json:"start_time"`
	EndTime      time.Time              `json:"end_time"`
	InitialState map[string]interface{} `json:"initial_state"`
	FinalState   map[string]interface{} `json:"final_state"`
	Metrics      SimulationMetrics      `json:"metrics"`
	EventsLog    []EventLogEntry        `json:"events_log"`
	Snapshots    []StateSnapshot        `json:"snapshots,omitempty"` // Optional: for detailed tracking
	Error        string                 `json:"error,omitempty"`
}

// SimulationMetrics contains key performance indicators from a simulation
type SimulationMetrics struct {
	TotalSteps         int                `json:"total_steps"`
	EventsProcessed    int                `json:"events_processed"`
	EntitiesAffected   int                `json:"entities_affected"`
	AverageUtilization float64            `json:"average_utilization"`
	PeakUtilization    float64            `json:"peak_utilization"`
	BottleneckEntities []string           `json:"bottleneck_entities"`
	CriticalEvents     int                `json:"critical_events"`
	SystemStability    float64            `json:"system_stability"` // 0.0 to 1.0
	CustomMetrics      map[string]float64 `json:"custom_metrics"`
	ImpactSummary      string             `json:"impact_summary"`
	Recommendations    []string           `json:"recommendations"`
}

// EventLogEntry records an event execution
type EventLogEntry struct {
	Step         int                    `json:"step"`
	Timestamp    time.Time              `json:"timestamp"`
	EventID      string                 `json:"event_id"`
	EventType    string                 `json:"event_type"`
	TargetURI    string                 `json:"target_uri"`
	Success      bool                   `json:"success"`
	StateChanges map[string]interface{} `json:"state_changes"`
	PropagatedTo []string               `json:"propagated_to"`
	Message      string                 `json:"message"`
}

// StateSnapshot captures the complete state at a point in time
type StateSnapshot struct {
	RunID       string                 `json:"run_id"`
	Step        int                    `json:"step"`
	Timestamp   time.Time              `json:"timestamp"`
	State       map[string]interface{} `json:"state"`
	Metrics     map[string]float64     `json:"metrics"`
	Description string                 `json:"description,omitempty"`
}

// TwinState represents the complete state of the digital twin at any point
type TwinState struct {
	Timestamp     time.Time              `json:"timestamp"`
	Step          int                    `json:"step"`
	Entities      map[string]EntityState `json:"entities"` // URI -> State
	GlobalMetrics map[string]float64     `json:"global_metrics"`
	ActiveEvents  []string               `json:"active_events"` // Events currently affecting the system
	Flags         map[string]bool        `json:"flags"`         // System-level flags (e.g., "overloaded", "stable")
}

// ScenarioTemplate defines reusable scenario patterns
type ScenarioTemplate struct {
	Name            string                                                `json:"name"`
	Description     string                                                `json:"description"`
	Category        string                                                `json:"category"` // "healthcare", "supply_chain", "finance", etc.
	Parameters      []TemplateParameter                                   `json:"parameters"`
	EventsGenerator func(params map[string]interface{}) []SimulationEvent `json:"-"`
}

// TemplateParameter defines a configurable parameter for a scenario template
type TemplateParameter struct {
	Name        string                 `json:"name"`
	Type        string                 `json:"type"` // "string", "number", "entity_uri", "duration"
	Description string                 `json:"description"`
	Required    bool                   `json:"required"`
	Default     interface{}            `json:"default,omitempty"`
	Constraints map[string]interface{} `json:"constraints,omitempty"` // e.g., min, max, enum
}

// ImpactAnalysis provides insights from simulation results
type ImpactAnalysis struct {
	RunID              string                 `json:"run_id"`
	OverallImpact      string                 `json:"overall_impact"` // "minimal", "moderate", "severe", "critical"
	AffectedEntities   []EntityImpactSummary  `json:"affected_entities"`
	CriticalPath       []string               `json:"critical_path"` // URIs of bottleneck entities
	RecoveryTime       int                    `json:"recovery_time"` // Steps to return to baseline
	AlternativeActions []ActionRecommendation `json:"alternative_actions"`
	RiskScore          float64                `json:"risk_score"` // 0.0 to 1.0
	Insights           []string               `json:"insights"`
}

// EntityImpactSummary summarizes impact on a specific entity
type EntityImpactSummary struct {
	URI               string             `json:"uri"`
	Label             string             `json:"label"`
	ImpactType        string             `json:"impact_type"` // "direct", "propagated", "cascading"
	UtilizationChange float64            `json:"utilization_change"`
	StatusChanges     []string           `json:"status_changes"`
	MetricChanges     map[string]float64 `json:"metric_changes"`
	TimeToImpact      int                `json:"time_to_impact"` // Steps from scenario start
	Duration          int                `json:"duration"`       // Steps of impact
}

// ActionRecommendation suggests mitigating actions
type ActionRecommendation struct {
	Action          string   `json:"action"`
	Description     string   `json:"description"`
	Urgency         string   `json:"urgency"`     // "low", "medium", "high", "critical"
	Feasibility     float64  `json:"feasibility"` // 0.0 to 1.0
	ExpectedBenefit string   `json:"expected_benefit"`
	TargetEntities  []string `json:"target_entities"`
}

// Helper methods

// ToJSON converts a DigitalTwin to JSON string
func (dt *DigitalTwin) ToJSON() (string, error) {
	data, err := json.Marshal(dt)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// FromJSON populates a DigitalTwin from JSON string
func (dt *DigitalTwin) FromJSON(jsonStr string) error {
	return json.Unmarshal([]byte(jsonStr), dt)
}

// GetEntity retrieves an entity by URI
func (dt *DigitalTwin) GetEntity(uri string) *TwinEntity {
	for i := range dt.Entities {
		if dt.Entities[i].URI == uri {
			return &dt.Entities[i]
		}
	}
	return nil
}

// GetRelationships returns all relationships for a given entity
func (dt *DigitalTwin) GetRelationships(uri string) []TwinRelationship {
	var rels []TwinRelationship
	for _, rel := range dt.Relationships {
		if rel.SourceURI == uri || rel.TargetURI == uri {
			rels = append(rels, rel)
		}
	}
	return rels
}

// GetRelatedEntities returns URIs of entities related to the given entity
func (dt *DigitalTwin) GetRelatedEntities(uri string) []string {
	var related []string
	for _, rel := range dt.Relationships {
		if rel.SourceURI == uri {
			related = append(related, rel.TargetURI)
		} else if rel.TargetURI == uri {
			related = append(related, rel.SourceURI)
		}
	}
	return related
}

// Clone creates a deep copy of the TwinState
func (ts *TwinState) Clone() *TwinState {
	clone := &TwinState{
		Timestamp:     ts.Timestamp,
		Step:          ts.Step,
		Entities:      make(map[string]EntityState),
		GlobalMetrics: make(map[string]float64),
		ActiveEvents:  make([]string, len(ts.ActiveEvents)),
		Flags:         make(map[string]bool),
	}

	for k, v := range ts.Entities {
		clone.Entities[k] = v
	}
	for k, v := range ts.GlobalMetrics {
		clone.GlobalMetrics[k] = v
	}
	copy(clone.ActiveEvents, ts.ActiveEvents)
	for k, v := range ts.Flags {
		clone.Flags[k] = v
	}

	return clone
}

// UpdateEntityState updates the state of an entity
func (ts *TwinState) UpdateEntityState(uri string, newState EntityState) {
	newState.LastUpdated = ts.Timestamp
	ts.Entities[uri] = newState
}

// GetEntityState retrieves the state of an entity
func (ts *TwinState) GetEntityState(uri string) (EntityState, bool) {
	state, exists := ts.Entities[uri]
	return state, exists
}

// CalculateAverageUtilization calculates the average utilization across all entities
func (ts *TwinState) CalculateAverageUtilization() float64 {
	if len(ts.Entities) == 0 {
		return 0.0
	}

	total := 0.0
	count := 0
	for _, entity := range ts.Entities {
		if entity.Capacity > 0 { // Only count entities with capacity
			total += entity.Utilization
			count++
		}
	}

	if count == 0 {
		return 0.0
	}

	return total / float64(count)
}

// CalculatePeakUtilization finds the highest utilization among all entities
func (ts *TwinState) CalculatePeakUtilization() float64 {
	peak := 0.0
	for _, entity := range ts.Entities {
		if entity.Utilization > peak {
			peak = entity.Utilization
		}
	}
	return peak
}

// IsStable checks if the system is in a stable state
func (ts *TwinState) IsStable() bool {
	// System is stable if no entities are overutilized and all are available
	for _, entity := range ts.Entities {
		if entity.Utilization > 0.95 || !entity.Available || entity.Status == "failed" {
			return false
		}
	}
	return true
}

// GetBottlenecks identifies entities operating at high utilization
func (ts *TwinState) GetBottlenecks(threshold float64) []string {
	var bottlenecks []string
	for uri, entity := range ts.Entities {
		if entity.Utilization >= threshold {
			bottlenecks = append(bottlenecks, uri)
		}
	}
	return bottlenecks
}
