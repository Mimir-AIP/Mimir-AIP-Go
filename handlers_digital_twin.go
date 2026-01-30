package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/DigitalTwin"
	"github.com/Mimir-AIP/Mimir-AIP-Go/utils"
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
		// Default query to get all entities from ontology's named graph
		graphURI := fmt.Sprintf("http://mimir.ai/ontology/%s", req.OntologyID)
		query = fmt.Sprintf(`
			PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
			SELECT ?entity ?type ?label
			WHERE {
				GRAPH <%s> {
					?entity a ?type .
					OPTIONAL { ?entity rdfs:label ?label }
				}
			}
		`, graphURI)
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

	// Query relationships from ontology's named graph
	graphURI := fmt.Sprintf("http://mimir.ai/ontology/%s", req.OntologyID)
	relQuery := fmt.Sprintf(`
		SELECT ?source ?target ?predicate
		WHERE {
			GRAPH <%s> {
				?source ?predicate ?target .
				FILTER(isIRI(?target))
			}
		}
	`, graphURI)
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

	// Auto-configure twin with scenarios and monitoring rules
	scenarioCount := s.autoConfigureTwin(twin.ID, twin.ModelType, db)

	writeSuccessResponse(w, map[string]interface{}{
		"twin_id":            twin.ID,
		"name":               twin.Name,
		"model_type":         twin.ModelType,
		"entity_count":       len(twin.Entities),
		"relationship_count": len(twin.Relationships),
		"scenarios_created":  scenarioCount,
		"auto_configured":    true,
		"message":            "Digital twin created and auto-configured successfully",
	})
}

// autoConfigureTwin sets up scenarios and monitoring for a newly created twin
func (s *Server) autoConfigureTwin(twinID, modelType string, db *sql.DB) int {
	creator := utils.NewTwinAutoCreator(db)

	// Auto-configure with scenarios and monitoring
	if err := creator.AutoConfigureTwin(twinID, "", modelType); err != nil {
		utils.GetLogger().Warn(fmt.Sprintf("Failed to auto-configure twin %s: %v", twinID, err))
		return 0
	}

	// Count created scenarios
	var count int
	query := `SELECT COUNT(*) FROM simulation_scenarios WHERE twin_id = ?`
	db.QueryRow(query, twinID).Scan(&count)

	return count
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

// handleUpdateTwin updates a digital twin by ID
func (s *Server) handleUpdateTwin(w http.ResponseWriter, r *http.Request) {
	if s.persistence == nil {
		writeErrorResponse(w, http.StatusServiceUnavailable, "Digital twin service not available")
		return
	}

	vars := mux.Vars(r)
	twinID := vars["id"]

	var req struct {
		Name        string                 `json:"name,omitempty"`
		Description string                 `json:"description,omitempty"`
		ModelType   string                 `json:"model_type,omitempty"`
		BaseState   map[string]interface{} `json:"base_state,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, fmt.Sprintf("Invalid request body: %v", err))
		return
	}

	// Check if twin exists
	db := s.persistence.GetDB()
	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM digital_twins WHERE id = ?)", twinID).Scan(&exists)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to check twin existence")
		return
	}
	if !exists {
		writeErrorResponse(w, http.StatusNotFound, "Digital twin not found")
		return
	}

	// Build UPDATE query dynamically based on provided fields
	updates := []string{}
	args := []interface{}{}

	if req.Name != "" {
		updates = append(updates, "name = ?")
		args = append(args, req.Name)
	}
	if req.Description != "" {
		updates = append(updates, "description = ?")
		args = append(args, req.Description)
	}
	if req.ModelType != "" {
		updates = append(updates, "model_type = ?")
		args = append(args, req.ModelType)
	}
	if req.BaseState != nil {
		stateJSON, err := json.Marshal(req.BaseState)
		if err != nil {
			writeErrorResponse(w, http.StatusBadRequest, fmt.Sprintf("Invalid base_state: %v", err))
			return
		}
		updates = append(updates, "base_state = ?")
		args = append(args, string(stateJSON))
	}

	if len(updates) == 0 {
		writeErrorResponse(w, http.StatusBadRequest, "No fields to update")
		return
	}

	// Always update updated_at timestamp
	updates = append(updates, "updated_at = ?")
	args = append(args, time.Now())

	// Add twin ID as final argument
	args = append(args, twinID)

	// Execute UPDATE
	query := fmt.Sprintf("UPDATE digital_twins SET %s WHERE id = ?", strings.Join(updates, ", "))
	_, err = db.Exec(query, args...)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to update twin: %v", err))
		return
	}

	// Fetch updated twin
	var twin DigitalTwin.DigitalTwin
	var baseStateJSON string
	err = db.QueryRow(`
		SELECT id, ontology_id, name, description, model_type, base_state, created_at, updated_at
		FROM digital_twins WHERE id = ?
	`, twinID).Scan(&twin.ID, &twin.OntologyID, &twin.Name, &twin.Description, &twin.ModelType, &baseStateJSON, &twin.CreatedAt, &twin.UpdatedAt)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to fetch updated twin: %v", err))
		return
	}

	if err := json.Unmarshal([]byte(baseStateJSON), &twin.BaseState); err != nil {
		twin.BaseState = make(map[string]interface{})
	}

	writeJSONResponse(w, http.StatusOK, map[string]any{
		"message": "Digital twin updated successfully",
		"data":    twin,
	})
}

// handleDeleteTwin deletes a digital twin by ID
func (s *Server) handleDeleteTwin(w http.ResponseWriter, r *http.Request) {
	if s.persistence == nil {
		writeErrorResponse(w, http.StatusServiceUnavailable, "Digital twin service not available")
		return
	}

	vars := mux.Vars(r)
	twinID := vars["id"]

	db := s.persistence.GetDB()

	// Check if twin exists
	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM digital_twins WHERE id = ?)", twinID).Scan(&exists)
	if err != nil || !exists {
		writeErrorResponse(w, http.StatusNotFound, "Digital twin not found")
		return
	}

	// Delete associated scenarios first (foreign key constraint)
	_, err = db.Exec("DELETE FROM simulation_scenarios WHERE twin_id = ?", twinID)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to delete twin scenarios: %v", err))
		return
	}

	// Delete simulation runs for this twin's scenarios
	_, err = db.Exec(`
		DELETE FROM simulation_runs 
		WHERE scenario_id IN (SELECT id FROM simulation_scenarios WHERE twin_id = ?)
	`, twinID)
	if err != nil {
		// Log but don't fail - runs may have already been deleted
		utils.GetLogger().Warn(fmt.Sprintf("Failed to delete simulation runs for twin %s: %v", twinID, err))
	}

	// Delete the twin
	_, err = db.Exec("DELETE FROM digital_twins WHERE id = ?", twinID)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to delete twin: %v", err))
		return
	}

	writeSuccessResponse(w, map[string]interface{}{
		"message": "Digital twin deleted successfully",
		"id":      twinID,
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

	// Populate metadata from database
	twin.ID = twinID
	twin.OntologyID = ontologyID
	twin.Name = name
	twin.Description = description
	twin.ModelType = modelType
	twin.CreatedAt = createdAt
	twin.UpdatedAt = updatedAt

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
		SELECT id, name, description, scenario_type, events, duration, created_at
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
		var id, name, description, scenarioType, eventsJSON string
		var duration int
		var createdAt time.Time

		err := rows.Scan(&id, &name, &description, &scenarioType, &eventsJSON, &duration, &createdAt)
		if err != nil {
			continue
		}

		// Parse events JSON
		var events []DigitalTwin.SimulationEvent
		if eventsJSON != "" {
			if err := json.Unmarshal([]byte(eventsJSON), &events); err != nil {
				// If parsing fails, use empty array
				events = []DigitalTwin.SimulationEvent{}
			}
		} else {
			events = []DigitalTwin.SimulationEvent{}
		}

		scenarios = append(scenarios, map[string]interface{}{
			"id":            id,
			"name":          name,
			"description":   description,
			"scenario_type": scenarioType,
			"events":        events,
			"duration":      duration,
			"created_at":    createdAt,
		})
	}

	// Auto-generate default scenarios if none exist for this twin
	if len(scenarios) == 0 {
		// Load twin to check if it exists and generate scenarios
		var twinJSON string
		twinQuery := `SELECT base_state FROM digital_twins WHERE id = ?`
		err := db.QueryRow(twinQuery, twinID).Scan(&twinJSON)
		if err != nil {
			// Twin doesn't exist - return 404
			if err == sql.ErrNoRows {
				writeErrorResponse(w, http.StatusNotFound, "Digital twin not found")
				return
			}
			// Other database error
			writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to load twin: %v", err))
			return
		}

		// Twin exists - generate scenarios
		var twin DigitalTwin.DigitalTwin
		if err := twin.FromJSON(twinJSON); err != nil {
			writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to parse twin: %v", err))
			return
		}
		twin.ID = twinID

		// Generate default scenarios
		defaultScenarios := s.generateDefaultScenariosForTwin(&twin)

		// Save them to database with transaction to prevent locks
		tx, err := db.Begin()
		if err != nil {
			// Log warning but continue without saving scenarios
			utils.GetLogger().Warn(fmt.Sprintf("Failed to start transaction for scenario generation: %v", err))
		} else {
			// Check again within transaction to prevent race condition (double-check pattern)
			var count int
			err = tx.QueryRow("SELECT COUNT(*) FROM simulation_scenarios WHERE twin_id = ?", twinID).Scan(&count)
			if err == nil && count == 0 {
				// Save scenarios with conflict handling
				for _, scenario := range defaultScenarios {
					eventsJSON, _ := json.Marshal(scenario.Events)
					insertQuery := `
						INSERT OR IGNORE INTO simulation_scenarios (id, twin_id, name, description, scenario_type, events, duration, created_at)
						VALUES (?, ?, ?, ?, ?, ?, ?, ?)
					`
					_, err := tx.Exec(insertQuery, scenario.ID, scenario.TwinID, scenario.Name, scenario.Description, scenario.Type, string(eventsJSON), scenario.Duration, scenario.CreatedAt)
					if err == nil {
						// Add to response
						scenarios = append(scenarios, map[string]interface{}{
							"id":            scenario.ID,
							"name":          scenario.Name,
							"description":   scenario.Description,
							"scenario_type": scenario.Type,
							"events":        scenario.Events,
							"duration":      scenario.Duration,
							"created_at":    scenario.CreatedAt,
						})
					} else {
						utils.GetLogger().Warn(fmt.Sprintf("Failed to insert scenario %s: %v", scenario.Name, err))
					}
				}
			}
			// Commit transaction
			if err := tx.Commit(); err != nil {
				utils.GetLogger().Warn(fmt.Sprintf("Failed to commit scenario transaction: %v", err))
				tx.Rollback()
			}
		}
	}

	writeSuccessResponse(w, map[string]interface{}{
		"scenarios": scenarios,
		"count":     len(scenarios),
	})
}

// handleRunSimulation executes a simulation scenario
func (s *Server) handleRunSimulation(w http.ResponseWriter, r *http.Request) {
	log.Printf("DEBUG: handleRunSimulation called")

	if s.persistence == nil {
		log.Printf("DEBUG: Persistence is nil")
		writeErrorResponse(w, http.StatusServiceUnavailable, "Digital twin service not available")
		return
	}

	vars := mux.Vars(r)
	twinID := vars["id"]
	scenarioID := vars["sid"]
	log.Printf("DEBUG: TwinID=%s, ScenarioID=%s", twinID, scenarioID)

	db := s.persistence.GetDB()
	// Load twin
	var twinJSON string
	query := `SELECT base_state FROM digital_twins WHERE id = ?`
	err := db.QueryRow(query, twinID).Scan(&twinJSON)
	if err != nil {
		log.Printf("DEBUG: Failed to load twin: %v", err)
		writeErrorResponse(w, http.StatusNotFound, "Digital twin not found")
		return
	}
	log.Printf("DEBUG: Twin loaded, JSON length: %d", len(twinJSON))

	var twin DigitalTwin.DigitalTwin
	if err := twin.FromJSON(twinJSON); err != nil {
		log.Printf("DEBUG: Failed to parse twin: %v", err)
		writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to parse twin: %v", err))
		return
	}
	log.Printf("DEBUG: Twin parsed successfully, ID=%s", twin.ID)

	// Load scenario
	var eventsJSON string
	var scenarioName, scenarioType string
	var duration int
	query = `SELECT name, scenario_type, events, duration FROM simulation_scenarios WHERE id = ?`
	err = db.QueryRow(query, scenarioID).Scan(&scenarioName, &scenarioType, &eventsJSON, &duration)
	if err != nil {
		log.Printf("DEBUG: Failed to load scenario: %v", err)
		writeErrorResponse(w, http.StatusNotFound, "Scenario not found")
		return
	}
	log.Printf("DEBUG: Scenario loaded: name=%s, type=%s, duration=%d", scenarioName, scenarioType, duration)

	var events []DigitalTwin.SimulationEvent
	if err := json.Unmarshal([]byte(eventsJSON), &events); err != nil {
		log.Printf("DEBUG: Failed to parse events: %v", err)
		writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to parse events: %v", err))
		return
	}
	log.Printf("DEBUG: Parsed %d events", len(events))

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
	log.Printf("DEBUG: Request options: SnapshotInterval=%d, MaxSteps=%d", req.SnapshotInterval, req.MaxSteps)

	// Create simulation engine with ML integration if available
	var engine *DigitalTwin.SimulationEngine
	if db != nil {
		engine = DigitalTwin.NewSimulationEngineWithML(&twin, db)
		if engine.IsUsingML() {
			log.Printf("DEBUG: Using ML-enhanced simulation engine")
		} else {
			log.Printf("DEBUG: No ML models available, using rule-based simulation")
		}
	} else {
		engine = DigitalTwin.NewSimulationEngine(&twin)
	}

	if req.SnapshotInterval > 0 {
		engine.SetSnapshotInterval(req.SnapshotInterval)
	}
	if req.MaxSteps > 0 {
		engine.SetMaxSteps(req.MaxSteps)
	}
	log.Printf("DEBUG: Simulation engine created")

	// Run simulation
	log.Printf("DEBUG: Starting simulation...")
	run, err := engine.RunSimulation(scenario)
	if err != nil {
		log.Printf("DEBUG: Simulation failed: %v", err)
		writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Simulation failed: %v", err))
		return
	}
	log.Printf("DEBUG: Simulation completed: RunID=%s, Status=%s, Metrics=%+v", run.ID, run.Status, run.Metrics)

	// Store run in database
	initialStateJSON, _ := json.Marshal(run.InitialState)
	finalStateJSON, _ := json.Marshal(run.FinalState)
	metricsJSON, _ := json.Marshal(run.Metrics)
	eventsLogJSON, _ := json.Marshal(run.EventsLog)
	log.Printf("DEBUG: Marshaled run data for storage")

	query = `
		INSERT INTO simulation_runs (id, scenario_id, status, start_time, end_time, initial_state, final_state, metrics, events_log, error_message)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err = db.Exec(query, run.ID, run.ScenarioID, run.Status, run.StartTime, run.EndTime, string(initialStateJSON), string(finalStateJSON), string(metricsJSON), string(eventsLogJSON), run.Error)
	if err != nil {
		log.Printf("DEBUG: Failed to save run to database: %v", err)
		writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to save run: %v", err))
		return
	}
	log.Printf("DEBUG: Run saved to database")

	// Store snapshots
	tsm := DigitalTwin.NewTemporalStateManager(db)
	for i, snapshot := range run.Snapshots {
		if err := tsm.StoreSnapshot(snapshot); err != nil {
			// Log error but don't fail the request
			log.Printf("Failed to store snapshot %d: %v", i, err)
		}
	}
	log.Printf("DEBUG: Stored %d snapshots", len(run.Snapshots))

	// Fix NaN values in metrics before returning (JSON doesn't support NaN)
	if math.IsNaN(run.Metrics.SystemStability) {
		run.Metrics.SystemStability = 1.0 // Default to stable if calculation produces NaN
	}
	if math.IsNaN(run.Metrics.AverageUtilization) {
		run.Metrics.AverageUtilization = 0.0
	}
	if math.IsNaN(run.Metrics.PeakUtilization) {
		run.Metrics.PeakUtilization = 0.0
	}

	responseData := map[string]interface{}{
		"run_id":  run.ID,
		"status":  run.Status,
		"metrics": run.Metrics,
		"message": "Simulation completed successfully",
	}
	log.Printf("DEBUG: About to write success response: %+v", responseData)
	writeSuccessResponse(w, responseData)
	log.Printf("DEBUG: Success response written")
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

// generateDefaultScenariosForTwin generates realistic simulation scenarios for a newly created Digital Twin
func (s *Server) generateDefaultScenariosForTwin(twin *DigitalTwin.DigitalTwin) []DigitalTwin.SimulationScenario {
	scenarios := []DigitalTwin.SimulationScenario{}

	// Scenario 1: Baseline - Normal operations with no events
	baselineScenario := DigitalTwin.SimulationScenario{
		ID:          fmt.Sprintf("scenario_%s_baseline", twin.ID),
		TwinID:      twin.ID,
		Name:        "Baseline Operations",
		Description: "Normal operating conditions with no disruptions. Establishes performance baseline for comparison.",
		Type:        "baseline",
		Events:      []DigitalTwin.SimulationEvent{},
		Duration:    30,
		CreatedAt:   time.Now(),
	}
	scenarios = append(scenarios, baselineScenario)

	// Scenario 2: Data Quality Issues
	dataQualityScenario := DigitalTwin.SimulationScenario{
		ID:          fmt.Sprintf("scenario_%s_data_quality", twin.ID),
		TwinID:      twin.ID,
		Name:        "Data Quality Issues",
		Description: "Simulates data quality problems including missing values, invalid data, and entity unavailability.",
		Type:        "data_quality_issue",
		Events:      []DigitalTwin.SimulationEvent{},
		Duration:    40,
		CreatedAt:   time.Now(),
	}
	scenarios = append(scenarios, dataQualityScenario)

	// Scenario 3: Capacity Test
	capacityScenario := DigitalTwin.SimulationScenario{
		ID:          fmt.Sprintf("scenario_%s_capacity", twin.ID),
		TwinID:      twin.ID,
		Name:        "Capacity Stress Test",
		Description: "Tests system behavior under high load conditions with demand surges and increased utilization.",
		Type:        "capacity_test",
		Events:      []DigitalTwin.SimulationEvent{},
		Duration:    50,
		CreatedAt:   time.Now(),
	}
	scenarios = append(scenarios, capacityScenario)

	return scenarios
}

// handleWhatIfAnalysis performs natural language what-if analysis
func (s *Server) handleWhatIfAnalysis(w http.ResponseWriter, r *http.Request) {
	log.Printf("[WHATIF] Starting what-if analysis")

	if s.persistence == nil {
		log.Printf("[WHATIF] ERROR: persistence is nil")
		writeErrorResponse(w, http.StatusServiceUnavailable, "Digital twin service not available")
		return
	}

	vars := mux.Vars(r)
	twinID := vars["id"]
	log.Printf("[WHATIF] Twin ID: %s", twinID)

	// Parse request
	var req struct {
		Question   string `json:"question"`
		MaxResults int    `json:"max_results,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[WHATIF] ERROR: Failed to parse request: %v", err)
		writeErrorResponse(w, http.StatusBadRequest, fmt.Sprintf("Invalid request: %v", err))
		return
	}

	log.Printf("[WHATIF] Question: %s", req.Question)

	if req.Question == "" {
		log.Printf("[WHATIF] ERROR: question is empty")
		writeErrorResponse(w, http.StatusBadRequest, "question is required")
		return
	}

	// Load twin
	log.Printf("[WHATIF] Loading twin from database")
	db := s.persistence.GetDB()
	if db == nil {
		log.Printf("[WHATIF] ERROR: database is nil")
		writeErrorResponse(w, http.StatusInternalServerError, "Database not available")
		return
	}

	var twinJSON string
	query := `SELECT base_state FROM digital_twins WHERE id = ?`
	log.Printf("[WHATIF] Executing query: %s with id=%s", query, twinID)
	err := db.QueryRow(query, twinID).Scan(&twinJSON)
	if err != nil {
		log.Printf("[WHATIF] ERROR: Failed to load twin: %v", err)
		writeErrorResponse(w, http.StatusNotFound, fmt.Sprintf("Digital twin not found: %v", err))
		return
	}

	log.Printf("[WHATIF] Twin JSON loaded, length: %d bytes", len(twinJSON))

	var twin DigitalTwin.DigitalTwin
	if err := twin.FromJSON(twinJSON); err != nil {
		log.Printf("[WHATIF] ERROR: Failed to parse twin JSON: %v", err)
		writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to parse twin: %v", err))
		return
	}
	twin.ID = twinID

	log.Printf("[WHATIF] Twin parsed successfully, entities: %d", len(twin.Entities))

	// Validate twin has entities
	if len(twin.Entities) == 0 {
		log.Printf("[WHATIF] ERROR: Twin has no entities")
		writeErrorResponse(w, http.StatusBadRequest, "Twin has no entities - cannot perform what-if analysis")
		return
	}

	// Create what-if engine with the configured LLM client and ML integration
	log.Printf("[WHATIF] Checking LLM client")
	if s.llmClient == nil {
		log.Printf("[WHATIF] ERROR: LLM client is nil")
		writeErrorResponse(w, http.StatusInternalServerError, "LLM client not configured")
		return
	}

	log.Printf("[WHATIF] Creating WhatIfEngine")
	whatIfEngine := DigitalTwin.NewWhatIfEngineWithDB(s.llmClient, db)
	if whatIfEngine == nil {
		log.Printf("[WHATIF] ERROR: WhatIfEngine is nil")
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to create what-if engine")
		return
	}

	log.Printf("[WHATIF] Running AnalyzeQuestion")
	// Run analysis
	response, err := whatIfEngine.AnalyzeQuestion(context.Background(), DigitalTwin.WhatIfQuery{
		Question:   req.Question,
		TwinID:     twinID,
		MaxResults: req.MaxResults,
	}, &twin)
	if err != nil {
		log.Printf("[WHATIF] ERROR: Analysis failed: %v", err)
		writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Analysis failed: %v", err))
		return
	}

	log.Printf("[WHATIF] Analysis completed successfully")
	writeSuccessResponse(w, response)
}

// handleGenerateSmartScenarios generates intelligent scenarios based on ontology
func (s *Server) handleGenerateSmartScenarios(w http.ResponseWriter, r *http.Request) {
	if s.persistence == nil {
		writeErrorResponse(w, http.StatusServiceUnavailable, "Digital twin service not available")
		return
	}

	vars := mux.Vars(r)
	twinID := vars["id"]

	// Load twin
	db := s.persistence.GetDB()
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
	twin.ID = twinID

	// Create smart scenario generator with the configured LLM client
	generator := DigitalTwin.NewSmartScenarioGenerator(s.llmClient)

	// Generate scenarios
	scenarios, err := generator.GenerateScenariosForTwin(context.Background(), &twin)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to generate scenarios: %v", err))
		return
	}

	// Optionally save scenarios to database
	saveParam := r.URL.Query().Get("save")
	savedCount := 0
	if saveParam == "true" {
		for _, gs := range scenarios {
			eventsJSON, _ := json.Marshal(gs.Scenario.Events)
			insertQuery := `
				INSERT INTO simulation_scenarios (id, twin_id, name, description, scenario_type, events, duration, created_at)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			`
			_, err := db.Exec(insertQuery, gs.Scenario.ID, gs.Scenario.TwinID, gs.Scenario.Name, gs.Scenario.Description, gs.Scenario.Type, string(eventsJSON), gs.Scenario.Duration, gs.Scenario.CreatedAt)
			if err == nil {
				savedCount++
			}
		}
	}

	writeSuccessResponse(w, map[string]interface{}{
		"scenarios":   scenarios,
		"count":       len(scenarios),
		"saved_count": savedCount,
	})
}

// handleGetInsights generates proactive insights for a digital twin
func (s *Server) handleGetInsights(w http.ResponseWriter, r *http.Request) {
	if s.persistence == nil {
		writeErrorResponse(w, http.StatusServiceUnavailable, "Digital twin service not available")
		return
	}

	vars := mux.Vars(r)
	twinID := vars["id"]

	// Load twin
	db := s.persistence.GetDB()
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
	twin.ID = twinID

	// Create insight suggester with the configured LLM client
	suggester := DigitalTwin.NewInsightSuggester(s.llmClient)

	// Generate insights
	report, err := suggester.GenerateInsights(context.Background(), &twin)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to generate insights: %v", err))
		return
	}

	writeSuccessResponse(w, report)
}

// handleAnalyzeOntology analyzes the ontology for patterns and risks
func (s *Server) handleAnalyzeOntology(w http.ResponseWriter, r *http.Request) {
	if s.persistence == nil {
		writeErrorResponse(w, http.StatusServiceUnavailable, "Digital twin service not available")
		return
	}

	vars := mux.Vars(r)
	twinID := vars["id"]

	// Load twin
	db := s.persistence.GetDB()
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

	// Create analyzer with the configured LLM client
	analyzer := DigitalTwin.NewOntologyAnalyzer(s.llmClient)

	// Run analysis
	analysis, err := analyzer.AnalyzeOntology(context.Background(), twin.Entities, twin.Relationships)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Analysis failed: %v", err))
		return
	}

	writeSuccessResponse(w, analysis)
}
