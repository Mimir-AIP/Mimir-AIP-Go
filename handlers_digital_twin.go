package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/DigitalTwin"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// CreateTwinRequest represents request to create a digital twin
type CreateTwinRequest struct {
	OntologyID  string `json:"ontology_id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	ModelType   string `json:"model_type"`      // "organization", "department", "process", "individual"
	Query       string `json:"query,omitempty"` // Optional SPARQL query to initialize entities
}

// CreateScenarioRequest represents request to create a simulation scenario
type CreateScenarioRequest struct {
	Name        string                        `json:"name"`
	Description string                        `json:"description,omitempty"`
	Type        string                        `json:"scenario_type,omitempty"`
	Duration    int                           `json:"duration"`
	Events      []DigitalTwin.SimulationEvent `json:"events"`
}

// RunSimulationRequest represents request to run a simulation
type RunSimulationRequest struct {
	SnapshotInterval int `json:"snapshot_interval,omitempty"`
	MaxSteps         int `json:"max_steps,omitempty"`
}

// handleCreateTwin creates a new digital twin from an ontology
func (s *Server) handleCreateTwin(w http.ResponseWriter, r *http.Request) {
	var req CreateTwinRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, fmt.Sprintf("Invalid request: %v", err))
		return
	}

	if s.persistence == nil || s.tdb2Backend == nil {
		writeErrorResponse(w, http.StatusServiceUnavailable, "Digital twin service not available")
		return
	}

	// Validate required fields
	if req.OntologyID == "" || req.Name == "" {
		writeErrorResponse(w, http.StatusBadRequest, "ontology_id and name are required")
		return
	}
	if req.ModelType == "" {
		req.ModelType = "organization"
	}

	// Verify ontology exists
	_, err := s.persistence.GetOntology(context.Background(), req.OntologyID)
	if err != nil {
		writeErrorResponse(w, http.StatusNotFound, fmt.Sprintf("Ontology not found: %s", req.OntologyID))
		return
	}

	// Query entities and relationships from knowledge graph using TDB2Graph
	query := req.Query
	if query == "" {
		// Default query to get all entities
		query = `
			SELECT ?entity ?type ?label
			WHERE {
				?entity a ?type .
				OPTIONAL { ?entity rdfs:label ?label }
			}
		`
	}

	results, err := s.tdb2Backend.QuerySPARQL(context.Background(), query)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to query knowledge graph: %v", err))
		return
	}

	// Build digital twin from results
	twin := &DigitalTwin.DigitalTwin{
		ID:            uuid.New().String(),
		OntologyID:    req.OntologyID,
		Name:          req.Name,
		Description:   req.Description,
		ModelType:     req.ModelType,
		BaseState:     make(map[string]interface{}),
		Entities:      []DigitalTwin.TwinEntity{},
		Relationships: []DigitalTwin.TwinRelationship{},
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// Parse entities from SPARQL results
	for _, binding := range results.Bindings {
		entityURI := ""
		if entity, ok := binding["entity"]; ok {
			entityURI = entity.Value
		}

		entityType := ""
		if typeVal, ok := binding["type"]; ok {
			entityType = typeVal.Value
		}

		entityLabel := entityURI // Default to URI if no label
		if label, ok := binding["label"]; ok {
			entityLabel = label.Value
		}

		if entityURI != "" {
			entity := DigitalTwin.TwinEntity{
				URI:        entityURI,
				Type:       entityType,
				Label:      entityLabel,
				Properties: make(map[string]interface{}),
				State: DigitalTwin.EntityState{
					Status:      "active",
					Capacity:    100.0,
					Utilization: 0.5,
					Available:   true,
					Metrics:     make(map[string]float64),
					LastUpdated: time.Now(),
				},
			}
			twin.Entities = append(twin.Entities, entity)
		}
	}

	// Query relationships
	relQuery := `
		SELECT ?source ?target ?predicate
		WHERE {
			?source ?predicate ?target .
			FILTER(isIRI(?target))
		}
	`
	relResults, err := s.tdb2Backend.QuerySPARQL(context.Background(), relQuery)
	if err == nil {
		for _, binding := range relResults.Bindings {
			sourceURI := ""
			targetURI := ""
			predicate := ""

			if source, ok := binding["source"]; ok {
				sourceURI = source.Value
			}
			if target, ok := binding["target"]; ok {
				targetURI = target.Value
			}
			if pred, ok := binding["predicate"]; ok {
				predicate = pred.Value
			}

			if sourceURI != "" && targetURI != "" && predicate != "" {
				rel := DigitalTwin.TwinRelationship{
					ID:         uuid.New().String(),
					SourceURI:  sourceURI,
					TargetURI:  targetURI,
					Type:       predicate,
					Properties: make(map[string]interface{}),
					Strength:   1.0, // Default strength
				}
				twin.Relationships = append(twin.Relationships, rel)
			}
		}
	}

	// Store twin in database
	twinJSON, err := twin.ToJSON()
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to serialize twin: %v", err))
		return
	}

	db := s.persistence.GetDB()
	query = `
		INSERT INTO digital_twins (id, ontology_id, name, description, model_type, base_state, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err = db.Exec(query, twin.ID, twin.OntologyID, twin.Name, twin.Description, twin.ModelType, twinJSON, twin.CreatedAt, twin.UpdatedAt)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to save twin: %v", err))
		return
	}

	writeSuccessResponse(w, map[string]interface{}{
		"twin_id":            twin.ID,
		"name":               twin.Name,
		"model_type":         twin.ModelType,
		"entity_count":       len(twin.Entities),
		"relationship_count": len(twin.Relationships),
		"message":            "Digital twin created successfully",
	})
}

// handleListTwins lists all digital twins
func (s *Server) handleListTwins(w http.ResponseWriter, r *http.Request) {
	if s.persistence == nil {
		writeErrorResponse(w, http.StatusServiceUnavailable, "Digital twin service not available")
		return
	}

	db := s.persistence.GetDB()
	query := `
		SELECT id, ontology_id, name, description, model_type, created_at
		FROM digital_twins
		ORDER BY created_at DESC
	`
	rows, err := db.Query(query)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to query twins: %v", err))
		return
	}
	defer rows.Close()

	twins := []map[string]interface{}{}
	for rows.Next() {
		var id, ontologyID, name, description, modelType string
		var createdAt time.Time

		err := rows.Scan(&id, &ontologyID, &name, &description, &modelType, &createdAt)
		if err != nil {
			continue
		}

		twins = append(twins, map[string]interface{}{
			"id":          id,
			"ontology_id": ontologyID,
			"name":        name,
			"description": description,
			"model_type":  modelType,
			"created_at":  createdAt,
		})
	}

	writeSuccessResponse(w, map[string]interface{}{
		"twins": twins,
		"count": len(twins),
	})
}

// handleGetTwin retrieves a digital twin by ID
func (s *Server) handleGetTwin(w http.ResponseWriter, r *http.Request) {
	if s.persistence == nil {
		writeErrorResponse(w, http.StatusServiceUnavailable, "Digital twin service not available")
		return
	}

	vars := mux.Vars(r)
	twinID := vars["id"]

	db := s.persistence.GetDB()
	var twinJSON string
	var ontologyID, name, description, modelType string
	var createdAt, updatedAt time.Time

	query := `
		SELECT id, ontology_id, name, description, model_type, base_state, created_at, updated_at
		FROM digital_twins
		WHERE id = ?
	`
	err := db.QueryRow(query, twinID).Scan(&twinID, &ontologyID, &name, &description, &modelType, &twinJSON, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		writeErrorResponse(w, http.StatusNotFound, "Digital twin not found")
		return
	}
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to retrieve twin: %v", err))
		return
	}

	var twin DigitalTwin.DigitalTwin
	if err := twin.FromJSON(twinJSON); err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to parse twin: %v", err))
		return
	}

	writeSuccessResponse(w, twin)
}

// handleGetTwinState retrieves the current state of a digital twin
func (s *Server) handleGetTwinState(w http.ResponseWriter, r *http.Request) {
	if s.persistence == nil {
		writeErrorResponse(w, http.StatusServiceUnavailable, "Digital twin service not available")
		return
	}

	vars := mux.Vars(r)
	twinID := vars["id"]

	db := s.persistence.GetDB()
	var twinJSON string
	query := `SELECT base_state FROM digital_twins WHERE id = ?`
	err := db.QueryRow(query, twinID).Scan(&twinJSON)
	if err == sql.ErrNoRows {
		writeErrorResponse(w, http.StatusNotFound, "Digital twin not found")
		return
	}
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to retrieve twin: %v", err))
		return
	}

	var twin DigitalTwin.DigitalTwin
	if err := twin.FromJSON(twinJSON); err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to parse twin: %v", err))
		return
	}

	// Build current state
	state := &DigitalTwin.TwinState{
		Timestamp:     time.Now(),
		Step:          0,
		Entities:      make(map[string]DigitalTwin.EntityState),
		GlobalMetrics: make(map[string]float64),
		ActiveEvents:  []string{},
		Flags:         make(map[string]bool),
	}

	for _, entity := range twin.Entities {
		state.Entities[entity.URI] = entity.State
	}

	state.GlobalMetrics["total_entities"] = float64(len(twin.Entities))
	state.GlobalMetrics["total_relationships"] = float64(len(twin.Relationships))
	state.GlobalMetrics["average_utilization"] = state.CalculateAverageUtilization()
	state.Flags["stable"] = state.IsStable()

	writeSuccessResponse(w, state)
}

// handleCreateScenario creates a new simulation scenario
func (s *Server) handleCreateScenario(w http.ResponseWriter, r *http.Request) {
	if s.persistence == nil {
		writeErrorResponse(w, http.StatusServiceUnavailable, "Digital twin service not available")
		return
	}

	vars := mux.Vars(r)
	twinID := vars["id"]

	db := s.persistence.GetDB()
	// Verify twin exists
	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM digital_twins WHERE id = ?)", twinID).Scan(&exists)
	if err != nil || !exists {
		writeErrorResponse(w, http.StatusNotFound, "Digital twin not found")
		return
	}

	var req CreateScenarioRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, fmt.Sprintf("Invalid request: %v", err))
		return
	}

	scenario := &DigitalTwin.SimulationScenario{
		ID:          uuid.New().String(),
		TwinID:      twinID,
		Name:        req.Name,
		Description: req.Description,
		Type:        req.Type,
		Events:      req.Events,
		Duration:    req.Duration,
		CreatedAt:   time.Now(),
	}

	eventsJSON, err := json.Marshal(scenario.Events)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to serialize events: %v", err))
		return
	}

	query := `
		INSERT INTO simulation_scenarios (id, twin_id, name, description, scenario_type, events, duration, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err = db.Exec(query, scenario.ID, scenario.TwinID, scenario.Name, scenario.Description, scenario.Type, string(eventsJSON), scenario.Duration, scenario.CreatedAt)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to save scenario: %v", err))
		return
	}

	writeSuccessResponse(w, map[string]interface{}{
		"scenario_id": scenario.ID,
		"name":        scenario.Name,
		"event_count": len(scenario.Events),
		"message":     "Scenario created successfully",
	})
}

// handleListScenarios lists all scenarios for a twin
func (s *Server) handleListScenarios(w http.ResponseWriter, r *http.Request) {
	if s.persistence == nil {
		writeErrorResponse(w, http.StatusServiceUnavailable, "Digital twin service not available")
		return
	}

	vars := mux.Vars(r)
	twinID := vars["id"]

	db := s.persistence.GetDB()
	query := `
		SELECT id, name, description, scenario_type, duration, created_at
		FROM simulation_scenarios
		WHERE twin_id = ?
		ORDER BY created_at DESC
	`
	rows, err := db.Query(query, twinID)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to query scenarios: %v", err))
		return
	}
	defer rows.Close()

	scenarios := []map[string]interface{}{}
	for rows.Next() {
		var id, name, description, scenarioType string
		var duration int
		var createdAt time.Time

		err := rows.Scan(&id, &name, &description, &scenarioType, &duration, &createdAt)
		if err != nil {
			continue
		}

		scenarios = append(scenarios, map[string]interface{}{
			"id":            id,
			"name":          name,
			"description":   description,
			"scenario_type": scenarioType,
			"duration":      duration,
			"created_at":    createdAt,
		})
	}

	writeSuccessResponse(w, map[string]interface{}{
		"scenarios": scenarios,
		"count":     len(scenarios),
	})
}

// handleRunSimulation executes a simulation scenario
func (s *Server) handleRunSimulation(w http.ResponseWriter, r *http.Request) {
	if s.persistence == nil {
		writeErrorResponse(w, http.StatusServiceUnavailable, "Digital twin service not available")
		return
	}

	vars := mux.Vars(r)
	twinID := vars["id"]
	scenarioID := vars["sid"]

	db := s.persistence.GetDB()
	// Load twin
	var twinJSON string
	query := `SELECT base_state FROM digital_twins WHERE id = ?`
	err := db.QueryRow(query, twinID).Scan(&twinJSON)
	if err != nil {
		writeErrorResponse(w, http.StatusNotFound, "Digital twin not found")
		return
	}

	var twin DigitalTwin.DigitalTwin
	if err := twin.FromJSON(twinJSON); err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to parse twin: %v", err))
		return
	}

	// Load scenario
	var eventsJSON string
	var scenarioName, scenarioType string
	var duration int
	query = `SELECT name, scenario_type, events, duration FROM simulation_scenarios WHERE id = ?`
	err = db.QueryRow(query, scenarioID).Scan(&scenarioName, &scenarioType, &eventsJSON, &duration)
	if err != nil {
		writeErrorResponse(w, http.StatusNotFound, "Scenario not found")
		return
	}

	var events []DigitalTwin.SimulationEvent
	if err := json.Unmarshal([]byte(eventsJSON), &events); err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to parse events: %v", err))
		return
	}

	scenario := &DigitalTwin.SimulationScenario{
		ID:       scenarioID,
		TwinID:   twinID,
		Name:     scenarioName,
		Type:     scenarioType,
		Events:   events,
		Duration: duration,
	}

	// Parse request options
	var req RunSimulationRequest
	json.NewDecoder(r.Body).Decode(&req)

	// Create simulation engine
	engine := DigitalTwin.NewSimulationEngine(&twin)
	if req.SnapshotInterval > 0 {
		engine.SetSnapshotInterval(req.SnapshotInterval)
	}
	if req.MaxSteps > 0 {
		engine.SetMaxSteps(req.MaxSteps)
	}

	// Run simulation
	run, err := engine.RunSimulation(scenario)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Simulation failed: %v", err))
		return
	}

	// Store run in database
	initialStateJSON, _ := json.Marshal(run.InitialState)
	finalStateJSON, _ := json.Marshal(run.FinalState)
	metricsJSON, _ := json.Marshal(run.Metrics)
	eventsLogJSON, _ := json.Marshal(run.EventsLog)

	query = `
		INSERT INTO simulation_runs (id, scenario_id, status, start_time, end_time, initial_state, final_state, metrics, events_log, error_message)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err = db.Exec(query, run.ID, run.ScenarioID, run.Status, run.StartTime, run.EndTime, string(initialStateJSON), string(finalStateJSON), string(metricsJSON), string(eventsLogJSON), run.Error)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to save run: %v", err))
		return
	}

	// Store snapshots
	tsm := DigitalTwin.NewTemporalStateManager(db)
	for _, snapshot := range run.Snapshots {
		if err := tsm.StoreSnapshot(snapshot); err != nil {
			// Log error but don't fail the request
			fmt.Printf("Failed to store snapshot: %v\n", err)
		}
	}

	writeSuccessResponse(w, map[string]interface{}{
		"run_id":  run.ID,
		"status":  run.Status,
		"metrics": run.Metrics,
		"message": "Simulation completed successfully",
	})
}

// handleGetSimulationRun retrieves simulation run results
func (s *Server) handleGetSimulationRun(w http.ResponseWriter, r *http.Request) {
	if s.persistence == nil {
		writeErrorResponse(w, http.StatusServiceUnavailable, "Digital twin service not available")
		return
	}

	vars := mux.Vars(r)
	runID := vars["rid"]

	db := s.persistence.GetDB()
	var run DigitalTwin.SimulationRun
	var initialStateJSON, finalStateJSON, metricsJSON, eventsLogJSON, errorMsg string

	query := `
		SELECT id, scenario_id, status, start_time, end_time, initial_state, final_state, metrics, events_log, error_message
		FROM simulation_runs
		WHERE id = ?
	`
	err := db.QueryRow(query, runID).Scan(&run.ID, &run.ScenarioID, &run.Status, &run.StartTime, &run.EndTime, &initialStateJSON, &finalStateJSON, &metricsJSON, &eventsLogJSON, &errorMsg)
	if err == sql.ErrNoRows {
		writeErrorResponse(w, http.StatusNotFound, "Simulation run not found")
		return
	}
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to retrieve run: %v", err))
		return
	}

	json.Unmarshal([]byte(initialStateJSON), &run.InitialState)
	json.Unmarshal([]byte(finalStateJSON), &run.FinalState)
	json.Unmarshal([]byte(metricsJSON), &run.Metrics)
	json.Unmarshal([]byte(eventsLogJSON), &run.EventsLog)
	run.Error = errorMsg

	writeSuccessResponse(w, run)
}

// handleGetSimulationTimeline retrieves temporal state timeline for a run
func (s *Server) handleGetSimulationTimeline(w http.ResponseWriter, r *http.Request) {
	if s.persistence == nil {
		writeErrorResponse(w, http.StatusServiceUnavailable, "Digital twin service not available")
		return
	}

	vars := mux.Vars(r)
	runID := vars["rid"]

	db := s.persistence.GetDB()
	tsm := DigitalTwin.NewTemporalStateManager(db)
	snapshots, err := tsm.GetSnapshotsByRunID(runID)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to retrieve timeline: %v", err))
		return
	}

	writeSuccessResponse(w, map[string]interface{}{
		"run_id":    runID,
		"snapshots": snapshots,
		"count":     len(snapshots),
	})
}

// handleAnalyzeImpact performs impact analysis on a simulation run
func (s *Server) handleAnalyzeImpact(w http.ResponseWriter, r *http.Request) {
	if s.persistence == nil {
		writeErrorResponse(w, http.StatusServiceUnavailable, "Digital twin service not available")
		return
	}

	vars := mux.Vars(r)
	twinID := vars["id"]
	runID := vars["rid"]

	db := s.persistence.GetDB()
	// Load twin
	var twinJSON string
	query := `SELECT base_state FROM digital_twins WHERE id = ?`
	err := db.QueryRow(query, twinID).Scan(&twinJSON)
	if err != nil {
		writeErrorResponse(w, http.StatusNotFound, "Digital twin not found")
		return
	}

	var twin DigitalTwin.DigitalTwin
	if err := twin.FromJSON(twinJSON); err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to parse twin: %v", err))
		return
	}

	// Load run
	var run DigitalTwin.SimulationRun
	var initialStateJSON, finalStateJSON, metricsJSON, eventsLogJSON, errorMsg string

	query = `
		SELECT id, scenario_id, status, start_time, end_time, initial_state, final_state, metrics, events_log, error_message
		FROM simulation_runs
		WHERE id = ?
	`
	err = db.QueryRow(query, runID).Scan(&run.ID, &run.ScenarioID, &run.Status, &run.StartTime, &run.EndTime, &initialStateJSON, &finalStateJSON, &metricsJSON, &eventsLogJSON, &errorMsg)
	if err != nil {
		writeErrorResponse(w, http.StatusNotFound, "Simulation run not found")
		return
	}

	json.Unmarshal([]byte(initialStateJSON), &run.InitialState)
	json.Unmarshal([]byte(finalStateJSON), &run.FinalState)
	json.Unmarshal([]byte(metricsJSON), &run.Metrics)
	json.Unmarshal([]byte(eventsLogJSON), &run.EventsLog)
	run.Error = errorMsg

	// Analyze impact
	engine := DigitalTwin.NewSimulationEngine(&twin)
	analysis, err := engine.AnalyzeImpact(&run)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to analyze impact: %v", err))
		return
	}

	writeSuccessResponse(w, analysis)
}
