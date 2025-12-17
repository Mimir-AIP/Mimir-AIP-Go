package DigitalTwin

import (
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

// SimulationEngine executes digital twin simulations
type SimulationEngine struct {
	twin             *DigitalTwin
	processor        *EventProcessor
	snapshotInterval int // Take snapshot every N steps (0 = disable)
	maxSteps         int
	mu               sync.RWMutex
}

// NewSimulationEngine creates a new simulation engine for a digital twin
func NewSimulationEngine(twin *DigitalTwin) *SimulationEngine {
	return &SimulationEngine{
		twin:             twin,
		processor:        NewEventProcessor(),
		snapshotInterval: 10, // Default: snapshot every 10 steps
		maxSteps:         1000,
	}
}

// SetSnapshotInterval configures how often state snapshots are taken
func (se *SimulationEngine) SetSnapshotInterval(interval int) {
	se.snapshotInterval = interval
}

// SetMaxSteps sets the maximum number of simulation steps
func (se *SimulationEngine) SetMaxSteps(maxSteps int) {
	se.maxSteps = maxSteps
}

// RunSimulation executes a simulation scenario
func (se *SimulationEngine) RunSimulation(scenario *SimulationScenario) (*SimulationRun, error) {
	se.mu.Lock()
	defer se.mu.Unlock()

	// Initialize simulation run
	run := &SimulationRun{
		ID:         uuid.New().String(),
		ScenarioID: scenario.ID,
		Status:     "running",
		StartTime:  time.Now(),
		EventsLog:  []EventLogEntry{},
		Snapshots:  []StateSnapshot{},
	}

	// Initialize state from twin
	state, err := se.initializeState()
	if err != nil {
		run.Status = "failed"
		run.Error = fmt.Sprintf("failed to initialize state: %v", err)
		run.EndTime = time.Now()
		return run, err
	}

	// Store initial state
	initialStateJSON, _ := json.Marshal(state)
	run.InitialState = make(map[string]interface{})
	json.Unmarshal(initialStateJSON, &run.InitialState)

	// Take initial snapshot
	if se.snapshotInterval > 0 {
		run.Snapshots = append(run.Snapshots, se.createSnapshot(run.ID, state, "Initial state"))
	}

	// Sort events by timestamp
	events := make([]*SimulationEvent, len(scenario.Events))
	for i := range scenario.Events {
		events[i] = &scenario.Events[i]
	}
	sort.Slice(events, func(i, j int) bool {
		return events[i].Timestamp < events[j].Timestamp
	})

	// Determine simulation duration
	duration := scenario.Duration
	if duration == 0 && len(events) > 0 {
		// If no duration specified, run until last event + buffer
		duration = events[len(events)-1].Timestamp + 10
	}
	if duration > se.maxSteps {
		duration = se.maxSteps
	}

	// Execute simulation steps
	eventIndex := 0
	for step := 0; step < duration; step++ {
		state.Step = step
		state.Timestamp = time.Now()

		// Process all events scheduled for this step
		for eventIndex < len(events) && events[eventIndex].Timestamp <= step {
			event := events[eventIndex]

			logEntry := EventLogEntry{
				Step:      step,
				Timestamp: time.Now(),
				EventID:   event.ID,
				EventType: event.Type,
				TargetURI: event.TargetURI,
			}

			// Process the event
			changes, err := se.processor.ProcessEvent(event, se.twin, state)
			if err != nil {
				logEntry.Success = false
				logEntry.Message = fmt.Sprintf("Error: %v", err)
				run.EventsLog = append(run.EventsLog, logEntry)
				eventIndex++
				continue
			}

			logEntry.Success = true
			logEntry.StateChanges = make(map[string]interface{})
			for _, change := range changes {
				logEntry.StateChanges[fmt.Sprintf("%s.%s", change.EntityURI, change.Field)] = change.NewValue
			}

			// Propagate impact through relationships
			propagatedChanges, err := se.processor.PropagateImpact(event, se.twin, state, changes)
			if err != nil {
				logEntry.Message = fmt.Sprintf("Warning: propagation error: %v", err)
			} else if len(propagatedChanges) > 0 {
				propagatedURIs := make([]string, 0, len(propagatedChanges))
				for _, pc := range propagatedChanges {
					propagatedURIs = append(propagatedURIs, pc.EntityURI)
					logEntry.StateChanges[fmt.Sprintf("%s.%s", pc.EntityURI, pc.Field)] = pc.NewValue
				}
				logEntry.PropagatedTo = propagatedURIs
				logEntry.Message = fmt.Sprintf("Propagated to %d entities", len(propagatedURIs))
			}

			run.EventsLog = append(run.EventsLog, logEntry)
			eventIndex++
		}

		// Update global metrics
		se.updateGlobalMetrics(state)

		// Take periodic snapshots
		if se.snapshotInterval > 0 && step%se.snapshotInterval == 0 && step > 0 {
			run.Snapshots = append(run.Snapshots, se.createSnapshot(run.ID, state, fmt.Sprintf("Step %d", step)))
		}

		// Check for system failures
		if se.checkSystemFailure(state) {
			run.EventsLog = append(run.EventsLog, EventLogEntry{
				Step:      step,
				Timestamp: time.Now(),
				EventType: "system.failure",
				Success:   true,
				Message:   "System reached critical failure state",
			})
			break
		}
	}

	// Store final state
	finalStateJSON, _ := json.Marshal(state)
	run.FinalState = make(map[string]interface{})
	json.Unmarshal(finalStateJSON, &run.FinalState)

	// Take final snapshot
	if se.snapshotInterval > 0 {
		run.Snapshots = append(run.Snapshots, se.createSnapshot(run.ID, state, "Final state"))
	}

	// Calculate metrics
	run.Metrics = se.calculateMetrics(run, state)

	// Mark as completed
	run.Status = "completed"
	run.EndTime = time.Now()

	return run, nil
}

// initializeState creates the initial state from the digital twin
func (se *SimulationEngine) initializeState() (*TwinState, error) {
	state := &TwinState{
		Timestamp:     time.Now(),
		Step:          0,
		Entities:      make(map[string]EntityState),
		GlobalMetrics: make(map[string]float64),
		ActiveEvents:  []string{},
		Flags:         make(map[string]bool),
	}

	// Initialize entity states from twin entities
	for _, entity := range se.twin.Entities {
		// Use existing state if available, otherwise create default
		if entity.State.LastUpdated.IsZero() {
			state.Entities[entity.URI] = EntityState{
				Status:      "active",
				Capacity:    100.0, // Default capacity
				Utilization: 0.5,   // Default 50% utilization
				Available:   true,
				Metrics:     make(map[string]float64),
				LastUpdated: time.Now(),
			}
		} else {
			state.Entities[entity.URI] = entity.State
		}
	}

	// Initialize global metrics
	state.GlobalMetrics["total_entities"] = float64(len(se.twin.Entities))
	state.GlobalMetrics["total_relationships"] = float64(len(se.twin.Relationships))
	state.Flags["stable"] = true

	return state, nil
}

// createSnapshot creates a state snapshot
func (se *SimulationEngine) createSnapshot(runID string, state *TwinState, description string) StateSnapshot {
	stateJSON, _ := json.Marshal(state)
	stateMap := make(map[string]interface{})
	json.Unmarshal(stateJSON, &stateMap)

	return StateSnapshot{
		RunID:       runID,
		Step:        state.Step,
		Timestamp:   time.Now(),
		State:       stateMap,
		Metrics:     state.GlobalMetrics,
		Description: description,
	}
}

// updateGlobalMetrics updates global metrics based on current state
func (se *SimulationEngine) updateGlobalMetrics(state *TwinState) {
	state.GlobalMetrics["average_utilization"] = state.CalculateAverageUtilization()
	state.GlobalMetrics["peak_utilization"] = state.CalculatePeakUtilization()

	// Count entities by status
	activeCount := 0
	failedCount := 0
	degradedCount := 0

	for _, entity := range state.Entities {
		switch entity.Status {
		case "active":
			activeCount++
		case "failed":
			failedCount++
		case "degraded":
			degradedCount++
		}
	}

	state.GlobalMetrics["active_entities"] = float64(activeCount)
	state.GlobalMetrics["failed_entities"] = float64(failedCount)
	state.GlobalMetrics["degraded_entities"] = float64(degradedCount)

	// Update system stability flag
	state.Flags["stable"] = state.IsStable()
	state.Flags["has_failures"] = failedCount > 0
	state.Flags["has_degraded"] = degradedCount > 0
}

// checkSystemFailure checks if the system has reached a critical failure state
func (se *SimulationEngine) checkSystemFailure(state *TwinState) bool {
	failedCount := int(state.GlobalMetrics["failed_entities"])
	totalEntities := int(state.GlobalMetrics["total_entities"])

	// System fails if >50% of entities are failed
	if totalEntities > 0 && float64(failedCount)/float64(totalEntities) > 0.5 {
		return true
	}

	return false
}

// calculateMetrics generates summary metrics for the simulation run
func (se *SimulationEngine) calculateMetrics(run *SimulationRun, finalState *TwinState) SimulationMetrics {
	metrics := SimulationMetrics{
		TotalSteps:      finalState.Step,
		EventsProcessed: len(run.EventsLog),
		CustomMetrics:   make(map[string]float64),
	}

	// Count unique entities affected
	affectedEntities := make(map[string]bool)
	criticalEvents := 0

	for _, logEntry := range run.EventsLog {
		affectedEntities[logEntry.TargetURI] = true
		for _, uri := range logEntry.PropagatedTo {
			affectedEntities[uri] = true
		}

		// Count critical events (failures, unavailability)
		if logEntry.EventType == EventResourceUnavailable ||
			logEntry.EventType == EventProcessFailure ||
			logEntry.EventType == EventExternalSupplyDisruption {
			criticalEvents++
		}
	}

	metrics.EntitiesAffected = len(affectedEntities)
	metrics.CriticalEvents = criticalEvents

	// Calculate average and peak utilization from snapshots
	if len(run.Snapshots) > 0 {
		totalAvg := 0.0
		maxPeak := 0.0

		for _, snapshot := range run.Snapshots {
			if avg, ok := snapshot.Metrics["average_utilization"]; ok {
				totalAvg += avg
			}
			if peak, ok := snapshot.Metrics["peak_utilization"]; ok {
				if peak > maxPeak {
					maxPeak = peak
				}
			}
		}

		metrics.AverageUtilization = totalAvg / float64(len(run.Snapshots))
		metrics.PeakUtilization = maxPeak
	} else {
		metrics.AverageUtilization = finalState.GlobalMetrics["average_utilization"]
		metrics.PeakUtilization = finalState.GlobalMetrics["peak_utilization"]
	}

	// Identify bottleneck entities (>90% utilization)
	metrics.BottleneckEntities = finalState.GetBottlenecks(0.9)

	// Calculate system stability score
	metrics.SystemStability = se.calculateStabilityScore(finalState, run)

	// Generate impact summary
	metrics.ImpactSummary = se.generateImpactSummary(run, finalState)

	// Generate recommendations
	metrics.Recommendations = se.generateRecommendations(finalState, metrics)

	return metrics
}

// calculateStabilityScore calculates a 0-1 score representing system stability
func (se *SimulationEngine) calculateStabilityScore(state *TwinState, run *SimulationRun) float64 {
	score := 1.0

	// Penalize for failed entities
	failedRatio := state.GlobalMetrics["failed_entities"] / state.GlobalMetrics["total_entities"]
	score -= failedRatio * 0.5

	// Penalize for degraded entities
	degradedRatio := state.GlobalMetrics["degraded_entities"] / state.GlobalMetrics["total_entities"]
	score -= degradedRatio * 0.3

	// Penalize for high utilization
	if state.GlobalMetrics["average_utilization"] > 0.8 {
		score -= 0.2
	}

	// Penalize for bottlenecks
	bottleneckCount := len(state.GetBottlenecks(0.9))
	if bottleneckCount > 0 {
		score -= float64(bottleneckCount) * 0.1
	}

	if score < 0 {
		score = 0
	}

	return score
}

// generateImpactSummary creates a human-readable summary of simulation impact
func (se *SimulationEngine) generateImpactSummary(run *SimulationRun, state *TwinState) string {
	summary := fmt.Sprintf("Simulation completed in %d steps. ", state.Step)

	if len(run.EventsLog) > 0 {
		summary += fmt.Sprintf("Processed %d events affecting %d entities. ",
			len(run.EventsLog),
			int(state.GlobalMetrics["total_entities"]))
	}

	failedCount := int(state.GlobalMetrics["failed_entities"])
	degradedCount := int(state.GlobalMetrics["degraded_entities"])

	if failedCount > 0 {
		summary += fmt.Sprintf("%d entities failed. ", failedCount)
	}
	if degradedCount > 0 {
		summary += fmt.Sprintf("%d entities degraded. ", degradedCount)
	}

	avgUtil := state.GlobalMetrics["average_utilization"]
	peakUtil := state.GlobalMetrics["peak_utilization"]

	summary += fmt.Sprintf("Average utilization: %.1f%%, Peak: %.1f%%. ", avgUtil*100, peakUtil*100)

	if state.Flags["stable"] {
		summary += "System remained stable."
	} else {
		summary += "System became unstable."
	}

	return summary
}

// generateRecommendations creates actionable recommendations based on simulation results
func (se *SimulationEngine) generateRecommendations(state *TwinState, metrics SimulationMetrics) []string {
	var recommendations []string

	// Recommend based on bottlenecks
	if len(metrics.BottleneckEntities) > 0 {
		recommendations = append(recommendations,
			fmt.Sprintf("Add capacity to %d bottleneck entities to improve throughput", len(metrics.BottleneckEntities)))
	}

	// Recommend based on failures
	failedCount := int(state.GlobalMetrics["failed_entities"])
	if failedCount > 0 {
		recommendations = append(recommendations,
			fmt.Sprintf("Investigate and resolve %d failed entities to restore full capacity", failedCount))
	}

	// Recommend based on average utilization
	if metrics.AverageUtilization > 0.8 {
		recommendations = append(recommendations,
			"System is running at high utilization - consider adding redundancy")
	}

	// Recommend based on stability
	if metrics.SystemStability < 0.7 {
		recommendations = append(recommendations,
			"System stability is low - implement failover mechanisms and increase resource buffers")
	}

	// Recommend based on critical events
	if metrics.CriticalEvents > 3 {
		recommendations = append(recommendations,
			"High number of critical events detected - review disaster recovery procedures")
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations, "System performed well under simulated conditions")
	}

	return recommendations
}

// AnalyzeImpact performs detailed impact analysis on simulation results
func (se *SimulationEngine) AnalyzeImpact(run *SimulationRun) (*ImpactAnalysis, error) {
	if run.Status != "completed" {
		return nil, fmt.Errorf("cannot analyze incomplete simulation run")
	}

	analysis := &ImpactAnalysis{
		RunID:              run.ID,
		AffectedEntities:   []EntityImpactSummary{},
		CriticalPath:       []string{},
		Insights:           []string{},
		AlternativeActions: []ActionRecommendation{},
	}

	// Determine overall impact level
	if run.Metrics.SystemStability > 0.8 {
		analysis.OverallImpact = "minimal"
	} else if run.Metrics.SystemStability > 0.6 {
		analysis.OverallImpact = "moderate"
	} else if run.Metrics.SystemStability > 0.3 {
		analysis.OverallImpact = "severe"
	} else {
		analysis.OverallImpact = "critical"
	}

	// Calculate risk score
	analysis.RiskScore = 1.0 - run.Metrics.SystemStability

	// Analyze entity-level impacts
	entityImpacts := make(map[string]*EntityImpactSummary)

	for _, logEntry := range run.EventsLog {
		if !logEntry.Success {
			continue
		}

		// Track direct impact
		if _, exists := entityImpacts[logEntry.TargetURI]; !exists {
			entityImpacts[logEntry.TargetURI] = &EntityImpactSummary{
				URI:           logEntry.TargetURI,
				ImpactType:    "direct",
				StatusChanges: []string{},
				MetricChanges: make(map[string]float64),
				TimeToImpact:  logEntry.Step,
			}
		}

		impact := entityImpacts[logEntry.TargetURI]
		impact.StatusChanges = append(impact.StatusChanges, logEntry.EventType)

		// Track propagated impacts
		for _, propagatedURI := range logEntry.PropagatedTo {
			if _, exists := entityImpacts[propagatedURI]; !exists {
				entityImpacts[propagatedURI] = &EntityImpactSummary{
					URI:           propagatedURI,
					ImpactType:    "propagated",
					StatusChanges: []string{},
					MetricChanges: make(map[string]float64),
					TimeToImpact:  logEntry.Step,
				}
			}
		}
	}

	// Convert to slice
	for uri, impact := range entityImpacts {
		if entity := se.twin.GetEntity(uri); entity != nil {
			impact.Label = entity.Label
		}
		analysis.AffectedEntities = append(analysis.AffectedEntities, *impact)
	}

	// Identify critical path (most impacted entities)
	if len(run.Metrics.BottleneckEntities) > 0 {
		analysis.CriticalPath = run.Metrics.BottleneckEntities
	}

	// Generate insights
	analysis.Insights = append(analysis.Insights, run.Metrics.ImpactSummary)

	if len(analysis.AffectedEntities) > 0 {
		analysis.Insights = append(analysis.Insights,
			fmt.Sprintf("Impact cascaded to %d entities through relationship propagation", len(analysis.AffectedEntities)))
	}

	// Generate alternative actions
	for _, rec := range run.Metrics.Recommendations {
		analysis.AlternativeActions = append(analysis.AlternativeActions, ActionRecommendation{
			Action:          rec,
			Description:     rec,
			Urgency:         se.determineUrgency(analysis.RiskScore),
			Feasibility:     0.8,
			ExpectedBenefit: "Improved system stability and reduced bottlenecks",
			TargetEntities:  run.Metrics.BottleneckEntities,
		})
	}

	return analysis, nil
}

// determineUrgency determines action urgency based on risk score
func (se *SimulationEngine) determineUrgency(riskScore float64) string {
	if riskScore > 0.8 {
		return "critical"
	} else if riskScore > 0.6 {
		return "high"
	} else if riskScore > 0.3 {
		return "medium"
	}
	return "low"
}

// CompareRuns compares multiple simulation runs
func CompareRuns(runs []*SimulationRun) map[string]interface{} {
	if len(runs) == 0 {
		return map[string]interface{}{"error": "no runs to compare"}
	}

	comparison := map[string]interface{}{
		"run_count": len(runs),
		"runs":      []map[string]interface{}{},
	}

	totalSteps := 0
	totalEvents := 0
	avgStability := 0.0

	for _, run := range runs {
		totalSteps += run.Metrics.TotalSteps
		totalEvents += run.Metrics.EventsProcessed
		avgStability += run.Metrics.SystemStability

		comparison["runs"] = append(comparison["runs"].([]map[string]interface{}), map[string]interface{}{
			"id":              run.ID,
			"status":          run.Status,
			"stability":       run.Metrics.SystemStability,
			"events":          run.Metrics.EventsProcessed,
			"bottlenecks":     len(run.Metrics.BottleneckEntities),
			"avg_utilization": run.Metrics.AverageUtilization,
		})
	}

	comparison["average_steps"] = totalSteps / len(runs)
	comparison["average_events"] = totalEvents / len(runs)
	comparison["average_stability"] = avgStability / float64(len(runs))

	return comparison
}
