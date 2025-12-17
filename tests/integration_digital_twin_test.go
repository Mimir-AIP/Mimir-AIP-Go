package tests

import (
	"testing"
	"time"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/DigitalTwin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDigitalTwinCreation tests basic digital twin creation and serialization
func TestDigitalTwinCreation(t *testing.T) {
	twin := DigitalTwin.NewDigitalTwin("test-twin-1", "test-ontology-1", "organization", "Test Organization Twin")

	assert.Equal(t, "test-twin-1", twin.ID)
	assert.Equal(t, "test-ontology-1", twin.OntologyID)
	assert.Equal(t, "organization", twin.ModelType)
	assert.Equal(t, "Test Organization Twin", twin.Name)
	assert.NotNil(t, twin.Entities)
	assert.NotNil(t, twin.Relationships)
	assert.NotNil(t, twin.BaseState)
	assert.NotZero(t, twin.CreatedAt)

	// Test JSON serialization
	jsonData, err := twin.ToJSON()
	require.NoError(t, err)
	assert.Contains(t, string(jsonData), "test-twin-1")

	// Test deserialization
	twin2, err := DigitalTwin.TwinFromJSON(jsonData)
	require.NoError(t, err)
	assert.Equal(t, twin.ID, twin2.ID)
	assert.Equal(t, twin.OntologyID, twin2.OntologyID)
}

// TestDigitalTwinWithEntities tests adding entities and relationships
func TestDigitalTwinWithEntities(t *testing.T) {
	twin := DigitalTwin.NewDigitalTwin("test-twin-2", "test-ontology-2", "organization", "Test Twin with Entities")

	// Add entities
	dept1 := DigitalTwin.NewTwinEntity("dept:sales", "Department", "Sales Department")
	dept1.AddProperty("capacity", 100.0)
	dept1.AddProperty("utilization", 0.75)

	dept2 := DigitalTwin.NewTwinEntity("dept:engineering", "Department", "Engineering Department")
	dept2.AddProperty("capacity", 50.0)
	dept2.AddProperty("utilization", 0.90)

	twin.AddEntity(dept1)
	twin.AddEntity(dept2)

	assert.Len(t, twin.Entities, 2)

	// Add relationship
	rel := DigitalTwin.NewTwinRelationship("rel-1", "dept:sales", "dept:engineering", "collaboratesWith", 0.8)
	twin.AddRelationship(rel)

	assert.Len(t, twin.Relationships, 1)

	// Test GetEntity
	foundEntity := twin.GetEntity("dept:sales")
	require.NotNil(t, foundEntity)
	assert.Equal(t, "Sales Department", foundEntity.Label)
	assert.Equal(t, 100.0, foundEntity.Properties["capacity"])

	// Test GetRelationships
	rels := twin.GetRelationships("dept:sales")
	assert.Len(t, rels, 1)
	assert.Equal(t, "collaboratesWith", rels[0].Type)

	// Test GetRelatedEntities
	related := twin.GetRelatedEntities("dept:sales")
	assert.Contains(t, related, "dept:engineering")
}

// TestEventCreation tests event creation with parameters
func TestEventCreation(t *testing.T) {
	// Test basic event creation
	event := DigitalTwin.CreateEvent("resource.unavailable", "dept:sales", 10, map[string]interface{}{
		"percentage": 0.5,
	})

	assert.Equal(t, "resource.unavailable", event.Type)
	assert.Equal(t, "dept:sales", event.TargetURI)
	assert.Equal(t, 10, event.Timestamp)
	assert.Equal(t, 0.5, event.Parameters["percentage"])

	// Test event with propagation
	event2 := DigitalTwin.CreateEvent("demand.surge", "dept:sales", 20, map[string]interface{}{
		"increase_factor": 2.0,
	}).WithPropagation("collaboratesWith", 0.8, 1)

	assert.Len(t, event2.Impact.PropagationRules, 1)
	assert.Equal(t, "collaboratesWith", event2.Impact.PropagationRules[0].RelationshipType)
	assert.Equal(t, 0.8, event2.Impact.PropagationRules[0].ImpactMultiplier)

	// Test event with severity
	event3 := DigitalTwin.CreateEvent("system.failure", "server:main", 5, nil).
		WithSeverity("critical")
	assert.Equal(t, "critical", event3.Impact.Severity)
}

// TestScenarioBuilder tests scenario creation
func TestScenarioBuilder(t *testing.T) {
	twin := DigitalTwin.NewDigitalTwin("test-twin-3", "test-ontology-3", "organization", "Test Twin for Scenarios")

	// Add entity
	dept1 := DigitalTwin.NewTwinEntity("dept:sales", "Department", "Sales")
	dept1.AddProperty("capacity", 100.0)
	dept1.State.Capacity = 100.0
	dept1.State.Utilization = 0.5
	twin.AddEntity(dept1)

	// Build scenario using fluent API
	event1 := DigitalTwin.CreateEvent("resource.unavailable", "dept:sales", 10, map[string]interface{}{
		"percentage": 0.3,
	})
	event2 := DigitalTwin.CreateEvent("demand.surge", "dept:sales", 20, map[string]interface{}{
		"increase_factor": 1.5,
	})

	builder := DigitalTwin.NewScenarioBuilder(twin, "Test Scenario")
	scenario := builder.
		WithDescription("Testing scenario builder").
		WithDuration(100).
		AddEvent(*event1).
		AddEvent(*event2).
		Build()

	assert.Equal(t, "Test Scenario", scenario.Name)
	assert.Equal(t, "Testing scenario builder", scenario.Description)
	assert.Equal(t, 100, scenario.Duration)
	assert.Len(t, scenario.Events, 2)
	assert.NotEmpty(t, scenario.ID)
}

// TestSimulationEngine tests running a simulation
func TestSimulationEngine(t *testing.T) {
	// Create twin with entities
	twin := DigitalTwin.NewDigitalTwin("sim-twin", "sim-ontology", "organization", "Simulation Test Twin")

	// Add entities with state
	dept1 := DigitalTwin.NewTwinEntity("dept:sales", "Department", "Sales")
	dept1.State.Capacity = 100.0
	dept1.State.Utilization = 0.5
	dept1.State.Status = "active"
	dept1.State.Available = true
	twin.AddEntity(dept1)

	dept2 := DigitalTwin.NewTwinEntity("dept:support", "Department", "Support")
	dept2.State.Capacity = 50.0
	dept2.State.Utilization = 0.6
	dept2.State.Status = "active"
	dept2.State.Available = true
	twin.AddEntity(dept2)

	// Add relationship
	rel := DigitalTwin.NewTwinRelationship("rel-1", "dept:sales", "dept:support", "dependsOn", 0.7)
	twin.AddRelationship(rel)

	// Build scenario
	event := DigitalTwin.CreateEvent("demand.surge", "dept:sales", 5, map[string]interface{}{
		"increase_factor": 1.5,
	})

	scenario := DigitalTwin.NewScenarioBuilder(twin, "Demand Surge Test").
		WithDuration(20).
		AddEvent(*event).
		Build()

	// Run simulation
	engine := DigitalTwin.NewSimulationEngine(twin)
	engine.SetMaxSteps(30)
	engine.SetSnapshotInterval(5)

	run, err := engine.RunSimulation(scenario)
	require.NoError(t, err)

	assert.Equal(t, "completed", run.Status)
	assert.NotEmpty(t, run.ID)
	assert.NotZero(t, run.StartTime)
	assert.NotZero(t, run.EndTime)
	assert.NotNil(t, run.FinalState)
	assert.NotNil(t, run.Metrics)
	assert.True(t, run.Metrics.TotalSteps > 0)
}

// TestEventProcessor tests the event processing system
func TestEventProcessor(t *testing.T) {
	processor := DigitalTwin.NewEventProcessor()

	// Test that processor is initialized
	assert.NotNil(t, processor)

	// Create a twin state
	state := &DigitalTwin.TwinState{
		Timestamp:     time.Now(),
		Step:          0,
		Entities:      make(map[string]DigitalTwin.EntityState),
		GlobalMetrics: make(map[string]float64),
		ActiveEvents:  []string{},
		Flags:         make(map[string]bool),
	}

	// Add entity state
	state.Entities["dept:sales"] = DigitalTwin.EntityState{
		Status:      "active",
		Capacity:    100.0,
		Utilization: 0.5,
		Available:   true,
		Metrics:     make(map[string]float64),
		LastUpdated: time.Now(),
	}

	// Test state operations
	assert.True(t, state.IsStable())
	assert.Equal(t, 0.5, state.CalculateAverageUtilization())
	assert.Equal(t, 0.5, state.CalculatePeakUtilization())
	assert.Empty(t, state.GetBottlenecks(0.9))

	// Test clone
	cloned := state.Clone()
	assert.Equal(t, state.Step, cloned.Step)
	assert.Equal(t, len(state.Entities), len(cloned.Entities))
}

// TestTwinState tests TwinState operations
func TestTwinState(t *testing.T) {
	state := &DigitalTwin.TwinState{
		Timestamp:     time.Now(),
		Step:          10,
		Entities:      make(map[string]DigitalTwin.EntityState),
		GlobalMetrics: make(map[string]float64),
		ActiveEvents:  []string{"event-1"},
		Flags:         make(map[string]bool),
	}

	// Add some entities
	state.UpdateEntityState("server:1", DigitalTwin.EntityState{
		Status:      "active",
		Capacity:    100.0,
		Utilization: 0.96, // Above 0.95 threshold
		Available:   true,
		Metrics:     map[string]float64{"cpu": 0.8},
	})

	state.UpdateEntityState("server:2", DigitalTwin.EntityState{
		Status:      "active",
		Capacity:    100.0,
		Utilization: 0.3,
		Available:   true,
		Metrics:     map[string]float64{"cpu": 0.2},
	})

	// Test GetEntityState
	entityState, exists := state.GetEntityState("server:1")
	assert.True(t, exists)
	assert.Equal(t, 0.96, entityState.Utilization)

	// Test CalculateAverageUtilization
	avgUtil := state.CalculateAverageUtilization()
	assert.InDelta(t, 0.625, avgUtil, 0.001) // (0.95 + 0.3) / 2

	// Test CalculatePeakUtilization
	peakUtil := state.CalculatePeakUtilization()
	assert.Equal(t, 0.95, peakUtil)

	// Test GetBottlenecks
	bottlenecks := state.GetBottlenecks(0.9)
	assert.Len(t, bottlenecks, 1)
	assert.Contains(t, bottlenecks, "server:1")

	// Test IsStable - should be false because server:1 is at 0.95 utilization
	assert.False(t, state.IsStable())
}

// TestImpactAnalysis tests the impact analysis feature
func TestImpactAnalysis(t *testing.T) {
	// Create twin
	twin := DigitalTwin.NewDigitalTwin("impact-twin", "impact-ontology", "organization", "Impact Analysis Test")

	// Add entities
	server := DigitalTwin.NewTwinEntity("server:main", "Server", "Main Server")
	server.State.Capacity = 100.0
	server.State.Utilization = 0.7
	server.State.Status = "active"
	server.State.Available = true
	twin.AddEntity(server)

	app := DigitalTwin.NewTwinEntity("app:web", "Application", "Web App")
	app.State.Capacity = 50.0
	app.State.Utilization = 0.5
	app.State.Status = "active"
	app.State.Available = true
	twin.AddEntity(app)

	// Add relationship
	rel := DigitalTwin.NewTwinRelationship("rel-1", "app:web", "server:main", "hostedOn", 0.9)
	twin.AddRelationship(rel)

	// Create scenario with server failure
	event := DigitalTwin.CreateEvent("resource.unavailable", "server:main", 5, map[string]interface{}{
		"percentage": 0.5,
	}).WithSeverity("high").WithPropagation("hostedOn", 0.8, 1)

	scenario := DigitalTwin.NewScenarioBuilder(twin, "Server Failure Analysis").
		WithDescription("Analyze impact of server capacity reduction").
		WithDuration(30).
		AddEvent(*event).
		Build()

	// Run simulation
	engine := DigitalTwin.NewSimulationEngine(twin)
	run, err := engine.RunSimulation(scenario)
	require.NoError(t, err)
	assert.Equal(t, "completed", run.Status)

	// Analyze impact
	analysis, err := engine.AnalyzeImpact(run)
	require.NoError(t, err)
	assert.NotNil(t, analysis)
	assert.NotEmpty(t, analysis.OverallImpact)
	assert.NotEmpty(t, analysis.Insights)
}

// TestScenarioTemplates tests the template registry
func TestScenarioTemplates(t *testing.T) {
	registry := DigitalTwin.GetDefaultTemplateRegistry()
	assert.NotNil(t, registry)

	// Get available templates
	templates := registry.ListTemplates()
	assert.NotEmpty(t, templates) // Should have default templates

	// Test getting a specific template
	template, exists := registry.GetTemplate("capacity_reduction")
	if exists {
		assert.Equal(t, "capacity_reduction", template.Name)
		assert.NotEmpty(t, template.Description)
	}
}
