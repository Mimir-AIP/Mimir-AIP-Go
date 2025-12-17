package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/DigitalTwin"
	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/Storage"
	"github.com/Mimir-AIP/Mimir-AIP-Go/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDigitalTwinCreation tests basic digital twin creation and serialization
func TestDigitalTwinCreation(t *testing.T) {
	twin := DigitalTwin.NewDigitalTwin("test-twin-1", "test-ontology-1", "organization", "Test Organization Twin")

	assert.Equal(t, "test-twin-1", twin.ID)
	assert.Equal(t, "test-ontology-1", twin.OntologyID)
	assert.Equal(t, "organization", twin.ModelType)
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

	// Add relationship
	rel := DigitalTwin.NewTwinRelationship("rel-1", "dept:sales", "dept:engineering", "collaboratesWith", 0.8)
	twin.AddRelationship(rel)

	// Verify
	assert.Len(t, twin.Entities, 2)
	assert.Len(t, twin.Relationships, 1)

	// Get entity
	entity := twin.GetEntity("dept:sales")
	require.NotNil(t, entity)
	assert.Equal(t, "Sales Department", entity.Label)

	capacity, exists := entity.GetPropertyFloat("capacity")
	assert.True(t, exists)
	assert.Equal(t, 100.0, capacity)

	// Get relationships
	rels := twin.GetRelationshipsFrom("dept:sales")
	assert.Len(t, rels, 1)
	assert.Equal(t, "dept:engineering", rels[0].TargetURI)
}

// TestEventCreationAndProcessing tests event system
func TestEventCreationAndProcessing(t *testing.T) {
	// Create a simple event
	event := DigitalTwin.CreateEvent("resource.unavailable", "dept:sales", 10, map[string]interface{}{
		"percentage": 0.5,
		"reason":     "maintenance",
	})

	assert.Equal(t, "resource.unavailable", event.Type)
	assert.Equal(t, "dept:sales", event.TargetURI)
	assert.Equal(t, 10, event.Timestamp)
	assert.Equal(t, 0.5, event.Parameters["percentage"])

	// Test event with propagation
	event2 := DigitalTwin.CreateEvent("demand.surge", "dept:sales", 20, map[string]interface{}{
		"increase_factor": 2.0,
	}).WithPropagation("collaboratesWith", 0.8, 1)

	assert.Len(t, event2.PropagationRules, 1)
	assert.Equal(t, "collaboratesWith", event2.PropagationRules[0].RelationshipType)
	assert.Equal(t, 0.8, event2.PropagationRules[0].ImpactMultiplier)
}

// TestScenarioBuilder tests scenario creation
func TestScenarioBuilder(t *testing.T) {
	twin := DigitalTwin.NewDigitalTwin("test-twin-3", "test-ontology-3", "organization", "Test Twin for Scenarios")

	// Add entities
	dept1 := DigitalTwin.NewTwinEntity("dept:sales", "Department", "Sales")
	dept1.AddProperty("capacity", 100.0)
	twin.AddEntity(dept1)

	// Build scenario
	builder := DigitalTwin.NewScenarioBuilder(twin, "Test Scenario")
	builder.SetDescription("Testing scenario builder")
	builder.SetDuration(100)

	event1 := DigitalTwin.CreateEvent("resource.unavailable", "dept:sales", 10, map[string]interface{}{
		"percentage": 0.3,
	})
	builder.AddEvent(event1)

	event2 := DigitalTwin.CreateEvent("demand.surge", "dept:sales", 20, map[string]interface{}{
		"increase_factor": 1.5,
	})
	builder.AddEvent(event2)

	scenario := builder.Build()

	assert.Equal(t, "Test Scenario", scenario.Name)
	assert.Equal(t, 100, scenario.Duration)
	assert.Len(t, scenario.Events, 2)
	assert.Equal(t, twin.ID, scenario.TwinID)

	// Test scenario serialization
	jsonData, err := json.Marshal(scenario)
	require.NoError(t, err)
	assert.Contains(t, string(jsonData), "Test Scenario")
}

// TestSimpleSimulation tests basic simulation execution
func TestSimpleSimulation(t *testing.T) {
	// Create twin
	twin := DigitalTwin.NewDigitalTwin("test-twin-4", "test-ontology-4", "organization", "Simple Simulation Test")

	// Add entity with capacity
	entity := DigitalTwin.NewTwinEntity("resource:server1", "Server", "Web Server 1")
	entity.AddProperty("capacity", 100.0)
	entity.AddProperty("utilization", 0.5)
	entity.AddProperty("status", "operational")
	twin.AddEntity(entity)

	// Create scenario: increase demand at step 10
	builder := DigitalTwin.NewScenarioBuilder(twin, "Demand Surge Test")
	builder.SetDuration(30)

	event := DigitalTwin.CreateEvent("demand.surge", "resource:server1", 10, map[string]interface{}{
		"increase_factor": 2.0,
	})
	builder.AddEvent(event)

	scenario := builder.Build()

	// Run simulation
	engine := DigitalTwin.NewSimulationEngine()
	run, err := engine.RunSimulation(scenario, twin, 5) // snapshot every 5 steps

	require.NoError(t, err)
	assert.Equal(t, "completed", run.Status)
	assert.Equal(t, 30, run.Metrics.TotalSteps)
	assert.Equal(t, 1, run.Metrics.EventsProcessed)
	assert.GreaterOrEqual(t, run.Metrics.EntitiesAffected, 1)
	assert.NotEmpty(t, run.EventsLog)

	// Check that utilization increased
	finalState := run.FinalState
	finalEntity := finalState.GetEntity("resource:server1")
	require.NotNil(t, finalEntity)

	finalUtil, exists := finalEntity.GetPropertyFloat("utilization")
	assert.True(t, exists)
	assert.Greater(t, finalUtil, 0.5, "Utilization should have increased from baseline")
}

// TestEventPropagation tests impact propagation through relationships
func TestEventPropagation(t *testing.T) {
	// Create twin with connected entities
	twin := DigitalTwin.NewDigitalTwin("test-twin-5", "test-ontology-5", "supply_chain", "Propagation Test")

	// Add entities
	supplier := DigitalTwin.NewTwinEntity("supplier:a", "Supplier", "Supplier A")
	supplier.AddProperty("capacity", 1000.0)
	supplier.AddProperty("reliability", 0.95)
	twin.AddEntity(supplier)

	manufacturer := DigitalTwin.NewTwinEntity("manufacturer:b", "Manufacturer", "Manufacturer B")
	manufacturer.AddProperty("capacity", 500.0)
	manufacturer.AddProperty("utilization", 0.7)
	twin.AddEntity(manufacturer)

	distributor := DigitalTwin.NewTwinEntity("distributor:c", "Distributor", "Distributor C")
	distributor.AddProperty("capacity", 800.0)
	distributor.AddProperty("utilization", 0.6)
	twin.AddEntity(distributor)

	// Add relationships: supplier -> manufacturer -> distributor
	rel1 := DigitalTwin.NewTwinRelationship("rel-1", "supplier:a", "manufacturer:b", "supplies", 1.0)
	rel2 := DigitalTwin.NewTwinRelationship("rel-2", "manufacturer:b", "distributor:c", "supplies", 1.0)
	twin.AddRelationship(rel1)
	twin.AddRelationship(rel2)

	// Create scenario: supplier disruption with propagation
	builder := DigitalTwin.NewScenarioBuilder(twin, "Supply Chain Disruption")
	builder.SetDuration(50)

	event := DigitalTwin.CreateEvent("resource.unavailable", "supplier:a", 10, map[string]interface{}{
		"percentage": 0.5, // 50% capacity loss
		"reason":     "factory_fire",
	}).WithPropagation("supplies", 0.8, 2) // Propagate through "supplies" relationships, 2 hops max

	builder.AddEvent(event)
	scenario := builder.Build()

	// Run simulation
	engine := DigitalTwin.NewSimulationEngine()
	run, err := engine.RunSimulation(scenario, twin, 10)

	require.NoError(t, err)
	assert.Equal(t, "completed", run.Status)

	// Check that impact propagated to all 3 entities
	assert.GreaterOrEqual(t, run.Metrics.EntitiesAffected, 3, "Impact should propagate to all entities in chain")

	// Verify final state shows reduced capacity throughout chain
	finalSupplier := run.FinalState.GetEntity("supplier:a")
	finalManufacturer := run.FinalState.GetEntity("manufacturer:b")
	finalDistributor := run.FinalState.GetEntity("distributor:c")

	require.NotNil(t, finalSupplier)
	require.NotNil(t, finalManufacturer)
	require.NotNil(t, finalDistributor)

	// Supplier should be directly affected (50% capacity loss)
	supplierCap, _ := finalSupplier.GetPropertyFloat("capacity")
	assert.Less(t, supplierCap, 1000.0, "Supplier capacity should be reduced")

	// Check events log for propagation
	propagationFound := false
	for _, logEntry := range run.EventsLog {
		if logEntry.Type == "state_change" && logEntry.EntityURI != "supplier:a" {
			propagationFound = true
			break
		}
	}
	assert.True(t, propagationFound, "Should find propagated state changes in events log")
}

// TestComplexScenario tests a more realistic scenario with multiple events
func TestComplexScenario(t *testing.T) {
	// Create organizational twin
	twin := DigitalTwin.NewDigitalTwin("test-twin-6", "test-ontology-6", "organization", "Hospital Network")

	// Add entities
	er := DigitalTwin.NewTwinEntity("dept:emergency", "Department", "Emergency Room")
	er.AddProperty("capacity", 50.0)
	er.AddProperty("utilization", 0.7)
	er.AddProperty("staff_count", 20.0)
	twin.AddEntity(er)

	icu := DigitalTwin.NewTwinEntity("dept:icu", "Department", "ICU")
	icu.AddProperty("capacity", 30.0)
	icu.AddProperty("utilization", 0.8)
	icu.AddProperty("staff_count", 15.0)
	twin.AddEntity(icu)

	surgery := DigitalTwin.NewTwinEntity("dept:surgery", "Department", "Surgery")
	surgery.AddProperty("capacity", 20.0)
	surgery.AddProperty("utilization", 0.6)
	surgery.AddProperty("staff_count", 25.0)
	twin.AddEntity(surgery)

	// Add relationships
	rel1 := DigitalTwin.NewTwinRelationship("rel-1", "dept:emergency", "dept:icu", "transfers_to", 0.9)
	rel2 := DigitalTwin.NewTwinRelationship("rel-2", "dept:emergency", "dept:surgery", "transfers_to", 0.7)
	twin.AddRelationship(rel1)
	twin.AddRelationship(rel2)

	// Create complex scenario: pandemic surge
	builder := DigitalTwin.NewScenarioBuilder(twin, "Pandemic Response Simulation")
	builder.SetDescription("Simulates hospital response to patient surge during pandemic")
	builder.SetDuration(100)

	// Event 1: Patient surge at ER (step 10)
	event1 := DigitalTwin.CreateEvent("demand.surge", "dept:emergency", 10, map[string]interface{}{
		"increase_factor": 3.0,
		"reason":          "pandemic_outbreak",
	}).WithPropagation("transfers_to", 0.6, 1)
	builder.AddEvent(event1)

	// Event 2: Staff shortage (step 20)
	event2 := DigitalTwin.CreateEvent("resource.unavailable", "dept:emergency", 20, map[string]interface{}{
		"percentage": 0.25,
		"reason":     "staff_illness",
	})
	builder.AddEvent(event2)

	// Event 3: Additional capacity added (step 50)
	event3 := DigitalTwin.CreateEvent("resource.add", "dept:icu", 50, map[string]interface{}{
		"amount": 10.0,
		"reason": "emergency_beds_deployed",
	})
	builder.AddEvent(event3)

	scenario := builder.Build()

	// Run simulation
	engine := DigitalTwin.NewSimulationEngine()
	run, err := engine.RunSimulation(scenario, twin, 10)

	require.NoError(t, err)
	assert.Equal(t, "completed", run.Status)
	assert.Equal(t, 100, run.Metrics.TotalSteps)
	assert.Equal(t, 3, run.Metrics.EventsProcessed)
	assert.GreaterOrEqual(t, run.Metrics.EntitiesAffected, 3)

	// Check metrics
	assert.Greater(t, run.Metrics.PeakUtilization, 0.8, "Peak utilization should be high during surge")
	assert.NotEmpty(t, run.Metrics.BottleneckEntities, "Should identify bottleneck entities")
	assert.NotEmpty(t, run.Metrics.Recommendations, "Should provide recommendations")

	// Verify snapshots were taken
	assert.GreaterOrEqual(t, len(run.Snapshots), 10, "Should have at least 10 snapshots (every 10 steps)")

	// Check impact analysis
	analysis := DigitalTwin.AnalyzeSimulationImpact(run, twin)
	assert.NotEmpty(t, analysis.OverallImpact)
	assert.Greater(t, analysis.RiskScore, 0.0)
	assert.NotEmpty(t, analysis.AffectedEntities)
	assert.NotEmpty(t, analysis.CriticalPath)
}

// TestDatabasePersistence tests twin/scenario/run persistence
func TestDatabasePersistence(t *testing.T) {
	// Initialize test database
	logger := utils.GetLogger()
	persistence, err := Storage.NewPersistence(":memory:", logger)
	require.NoError(t, err)
	defer persistence.Close()

	db := persistence.GetDB()
	require.NotNil(t, db)

	// Create and save a twin
	twin := DigitalTwin.NewDigitalTwin("persist-twin-1", "ontology-1", "organization", "Persistence Test Twin")
	entity := DigitalTwin.NewTwinEntity("entity:1", "TestEntity", "Test Entity 1")
	entity.AddProperty("value", 42.0)
	twin.AddEntity(entity)

	twinJSON, err := twin.ToJSON()
	require.NoError(t, err)

	// Insert twin
	_, err = db.Exec(`
		INSERT INTO digital_twins (id, ontology_id, name, description, model_type, base_state, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, twin.ID, twin.OntologyID, twin.Name, twin.Description, twin.ModelType, string(twinJSON), time.Now(), time.Now())
	require.NoError(t, err)

	// Retrieve twin
	var retrievedJSON string
	err = db.QueryRow("SELECT base_state FROM digital_twins WHERE id = ?", twin.ID).Scan(&retrievedJSON)
	require.NoError(t, err)

	retrievedTwin, err := DigitalTwin.TwinFromJSON([]byte(retrievedJSON))
	require.NoError(t, err)
	assert.Equal(t, twin.ID, retrievedTwin.ID)
	assert.Len(t, retrievedTwin.Entities, 1)

	// Create and save scenario
	builder := DigitalTwin.NewScenarioBuilder(twin, "Test Scenario")
	builder.SetDuration(50)
	event := DigitalTwin.CreateEvent("test.event", "entity:1", 10, map[string]interface{}{"test": true})
	builder.AddEvent(event)
	scenario := builder.Build()

	eventsJSON, err := json.Marshal(scenario.Events)
	require.NoError(t, err)

	_, err = db.Exec(`
		INSERT INTO simulation_scenarios (id, twin_id, name, description, scenario_type, events, duration, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, scenario.ID, scenario.TwinID, scenario.Name, scenario.Description, scenario.ScenarioType, string(eventsJSON), scenario.Duration, time.Now())
	require.NoError(t, err)

	// Retrieve scenario
	var retrievedEventsJSON string
	err = db.QueryRow("SELECT events FROM simulation_scenarios WHERE id = ?", scenario.ID).Scan(&retrievedEventsJSON)
	require.NoError(t, err)

	var retrievedEvents []DigitalTwin.SimulationEvent
	err = json.Unmarshal([]byte(retrievedEventsJSON), &retrievedEvents)
	require.NoError(t, err)
	assert.Len(t, retrievedEvents, 1)
	assert.Equal(t, "test.event", retrievedEvents[0].Type)

	// Create and save run
	engine := DigitalTwin.NewSimulationEngine()
	run, err := engine.RunSimulation(scenario, twin, 10)
	require.NoError(t, err)

	initialStateJSON, _ := json.Marshal(run.InitialState)
	finalStateJSON, _ := json.Marshal(run.FinalState)
	metricsJSON, _ := json.Marshal(run.Metrics)
	eventsLogJSON, _ := json.Marshal(run.EventsLog)

	_, err = db.Exec(`
		INSERT INTO simulation_runs (id, scenario_id, status, start_time, end_time, initial_state, final_state, metrics, events_log)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, run.ID, run.ScenarioID, run.Status, run.StartTime, run.EndTime, string(initialStateJSON), string(finalStateJSON), string(metricsJSON), string(eventsLogJSON))
	require.NoError(t, err)

	// Save snapshots
	for i, snapshot := range run.Snapshots {
		snapshotJSON, _ := json.Marshal(snapshot.State)
		snapshotMetricsJSON, _ := json.Marshal(snapshot.Metrics)

		_, err = db.Exec(`
			INSERT INTO twin_state_snapshots (run_id, timestamp, step_number, state, description, metrics)
			VALUES (?, ?, ?, ?, ?, ?)
		`, run.ID, snapshot.Timestamp, snapshot.StepNumber, string(snapshotJSON), snapshot.Description, string(snapshotMetricsJSON))
		require.NoError(t, err, fmt.Sprintf("Failed to insert snapshot %d", i))
	}

	// Retrieve run
	var retrievedStatus string
	var retrievedMetricsJSON string
	err = db.QueryRow("SELECT status, metrics FROM simulation_runs WHERE id = ?", run.ID).Scan(&retrievedStatus, &retrievedMetricsJSON)
	require.NoError(t, err)
	assert.Equal(t, "completed", retrievedStatus)

	// Retrieve snapshots
	rows, err := db.Query("SELECT step_number FROM twin_state_snapshots WHERE run_id = ? ORDER BY step_number", run.ID)
	require.NoError(t, err)
	defer rows.Close()

	snapshotCount := 0
	for rows.Next() {
		var stepNum int
		rows.Scan(&stepNum)
		snapshotCount++
	}
	assert.GreaterOrEqual(t, snapshotCount, 5, "Should have saved multiple snapshots")
}

// TestScenarioTemplates tests built-in scenario templates
func TestScenarioTemplates(t *testing.T) {
	twin := DigitalTwin.NewDigitalTwin("template-test", "ontology-1", "organization", "Template Test")

	entity := DigitalTwin.NewTwinEntity("resource:critical", "Resource", "Critical Resource")
	entity.AddProperty("capacity", 100.0)
	twin.AddEntity(entity)

	// Test resource unavailability template
	scenario1 := DigitalTwin.ResourceUnavailabilityScenario(twin, "resource:critical", 10, 30, "maintenance")
	assert.Equal(t, "Resource Unavailability: resource:critical", scenario1.Name)
	assert.Equal(t, "resource_unavailability", scenario1.ScenarioType)
	assert.Len(t, scenario1.Events, 2) // Start and end events

	// Test demand surge template
	scenario2 := DigitalTwin.DemandSurgeScenario(twin, "resource:critical", 5, 20, 2.5, "seasonal_peak")
	assert.Equal(t, "Demand Surge: resource:critical", scenario2.Name)
	assert.Equal(t, "demand_surge", scenario2.ScenarioType)
	assert.Len(t, scenario2.Events, 2)

	// Test capacity adjustment template
	scenario3 := DigitalTwin.CapacityAdjustmentScenario(twin, "resource:critical", 15, 50.0, "expansion")
	assert.Equal(t, "Capacity Adjustment: resource:critical", scenario3.Name)
	assert.Equal(t, "capacity_adjustment", scenario3.ScenarioType)
	assert.Len(t, scenario3.Events, 1)

	// Run one to verify it works
	engine := DigitalTwin.NewSimulationEngine()
	run, err := engine.RunSimulation(scenario1, twin, 0) // No snapshots
	require.NoError(t, err)
	assert.Equal(t, "completed", run.Status)
}

// TestTemporalStateAnalysis tests trend and anomaly detection
func TestTemporalStateAnalysis(t *testing.T) {
	// Create twin and scenario with gradual degradation
	twin := DigitalTwin.NewDigitalTwin("temporal-test", "ontology-1", "system", "Temporal Analysis Test")

	entity := DigitalTwin.NewTwinEntity("system:main", "System", "Main System")
	entity.AddProperty("health", 1.0)
	entity.AddProperty("throughput", 100.0)
	twin.AddEntity(entity)

	// Create scenario with events that gradually degrade the system
	builder := DigitalTwin.NewScenarioBuilder(twin, "Gradual Degradation")
	builder.SetDuration(100)

	// Add events at regular intervals to create a trend
	for i := 10; i <= 90; i += 10 {
		event := DigitalTwin.CreateEvent("performance.degradation", "system:main", i, map[string]interface{}{
			"impact": 0.1,
		})
		builder.AddEvent(event)
	}

	scenario := builder.Build()

	// Run simulation with frequent snapshots
	engine := DigitalTwin.NewSimulationEngine()
	run, err := engine.RunSimulation(scenario, twin, 5)
	require.NoError(t, err)

	// Analyze trends
	ctx := context.Background()
	trends, err := DigitalTwin.AnalyzeTrends(ctx, run.ID, "system:main", "health", nil)
	require.NoError(t, err)
	assert.Equal(t, "decreasing", trends.Direction, "Health should be trending downward")

	// Extract metric history
	history, err := DigitalTwin.ExtractMetricHistory(ctx, run.ID, "system:main", "health", nil)
	require.NoError(t, err)
	assert.Greater(t, len(history), 10, "Should have multiple data points")

	// Verify values are decreasing
	for i := 1; i < len(history); i++ {
		assert.LessOrEqual(t, history[i].Value, history[i-1].Value, "Health should decrease over time")
	}
}

// TestCustomEventTypes tests that the system accepts arbitrary event types
func TestCustomEventTypes(t *testing.T) {
	twin := DigitalTwin.NewDigitalTwin("custom-events", "ontology-1", "custom", "Custom Events Test")

	entity := DigitalTwin.NewTwinEntity("custom:entity", "CustomType", "Custom Entity")
	entity.AddProperty("custom_metric", 100.0)
	twin.AddEntity(entity)

	// Create scenario with completely custom event types
	builder := DigitalTwin.NewScenarioBuilder(twin, "Custom Events Scenario")
	builder.SetDuration(50)

	// Custom event type 1
	event1 := DigitalTwin.CreateEvent("mycompany.custom.event.type1", "custom:entity", 10, map[string]interface{}{
		"arbitrary_field":  "test_value",
		"numeric_field":    42.7,
		"boolean_field":    true,
		"nested_structure": map[string]interface{}{"nested": "value"},
	})
	builder.AddEvent(event1)

	// Custom event type 2
	event2 := DigitalTwin.CreateEvent("domain.specific.event", "custom:entity", 20, map[string]interface{}{
		"domain_parameter": "important_value",
	})
	builder.AddEvent(event2)

	scenario := builder.Build()

	// System should accept and process these custom events
	engine := DigitalTwin.NewSimulationEngine()
	run, err := engine.RunSimulation(scenario, twin, 0)
	require.NoError(t, err)
	assert.Equal(t, "completed", run.Status)
	assert.Equal(t, 2, run.Metrics.EventsProcessed)

	// Verify events were logged
	assert.Len(t, run.EventsLog, 2)
	assert.Equal(t, "mycompany.custom.event.type1", run.EventsLog[0].EventType)
	assert.Equal(t, "domain.specific.event", run.EventsLog[1].EventType)
}

// TestSystemFailureDetection tests that simulation detects system failures
func TestSystemFailureDetection(t *testing.T) {
	twin := DigitalTwin.NewDigitalTwin("failure-test", "ontology-1", "system", "Failure Detection Test")

	entity := DigitalTwin.NewTwinEntity("critical:component", "Component", "Critical Component")
	entity.AddProperty("capacity", 100.0)
	entity.AddProperty("utilization", 0.8)
	twin.AddEntity(entity)

	// Create catastrophic event
	builder := DigitalTwin.NewScenarioBuilder(twin, "Catastrophic Failure")
	builder.SetDuration(50)

	event := DigitalTwin.CreateEvent("system.catastrophic_failure", "critical:component", 10, map[string]interface{}{
		"failure_type": "complete",
		"impact":       1.0,
	})
	builder.AddEvent(event)

	scenario := builder.Build()

	engine := DigitalTwin.NewSimulationEngine()
	run, err := engine.RunSimulation(scenario, twin, 5)

	// Simulation should complete even with failures
	require.NoError(t, err)
	assert.Equal(t, "completed", run.Status)

	// Check for critical events in metrics
	assert.Greater(t, run.Metrics.CriticalEvents, 0, "Should detect critical events")
	assert.Less(t, run.Metrics.SystemStability, 0.5, "System stability should be low after catastrophic failure")
}
