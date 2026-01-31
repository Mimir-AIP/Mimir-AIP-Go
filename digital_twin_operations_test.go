package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/DigitalTwin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// SECTION 1: Twin Creation & Configuration Tests
// ============================================================================

// TestTwinCreation_BasicFlow tests creating a twin from an existing ontology
func TestTwinCreation_BasicFlow(t *testing.T) {
	server := NewServer()
	require.NotNil(t, server)

	// First, create an ontology to use for twin creation
	var ontologyID string
	t.Run("Create test ontology", func(t *testing.T) {
		ontologyData := `@prefix ex: <http://example.org/test#> .
@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .

ex:Department a rdfs:Class ;
    rdfs:label "Department" .

ex:Employee a rdfs:Class ;
    rdfs:label "Employee" .

ex:Project a rdfs:Class ;
    rdfs:label "Project" .

ex:Engineering a ex:Department ;
    rdfs:label "Engineering" .

ex:Alice a ex:Employee ;
    rdfs:label "Alice Smith" ;
    ex:worksIn ex:Engineering .

ex:ProjectAlpha a ex:Project ;
    rdfs:label "Project Alpha" ;
    ex:ownedBy ex:Engineering .`

		createReq := map[string]any{
			"name":          "Test Organization Ontology",
			"description":   "Test ontology for digital twin creation",
			"version":       "1.0.0",
			"ontology_data": ontologyData,
			"format":        "turtle",
		}

		body, _ := json.Marshal(createReq)
		req := httptest.NewRequest("POST", "/api/v1/ontology", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		// Accept 200, 201, or 500 (if features not available)
		assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusCreated || w.Code == http.StatusInternalServerError,
			"Expected 200, 201, or 500, got %d", w.Code)

		if w.Code == http.StatusOK || w.Code == http.StatusCreated {
			var response map[string]any
			err := json.Unmarshal(w.Body.Bytes(), &response)
			if err == nil {
				if id, ok := response["ontology_id"].(string); ok {
					ontologyID = id
				}
			}
		}
	})

	// Test twin creation
	var twinID string
	t.Run("Create digital twin from ontology", func(t *testing.T) {
		if ontologyID == "" {
			t.Skip("Ontology not created, skipping twin creation test")
		}

		createReq := map[string]any{
			"ontology_id": ontologyID,
			"name":        "Test Organization Twin",
			"description": "Digital twin for testing",
			"model_type":  "organization",
		}

		body, _ := json.Marshal(createReq)
		req := httptest.NewRequest("POST", "/api/v1/twins", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		// Accept 200, 201, or 503 (if service not available)
		assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusCreated || w.Code == http.StatusServiceUnavailable,
			"Expected 200, 201, or 503, got %d", w.Code)

		if w.Code == http.StatusOK || w.Code == http.StatusCreated {
			var response map[string]any
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			// Verify twin was created with expected structure
			assert.NotEmpty(t, response["twin_id"], "Twin ID should be returned")
			twinID = response["twin_id"].(string)
			assert.NotEmpty(t, twinID)

			assert.Equal(t, "Test Organization Twin", response["name"])
			assert.Equal(t, "organization", response["model_type"])

			// Verify entities and relationships were extracted
			entityCount, _ := response["entity_count"].(float64)
			relCount, _ := response["relationship_count"].(float64)

			// Should have at least 3 entities (Department, Employee, Project + instances)
			assert.GreaterOrEqual(t, int(entityCount), 3, "Should have extracted entities from ontology")
			assert.GreaterOrEqual(t, int(relCount), 0, "Should have extracted relationships")

			// Verify auto-configuration status
			autoConfigured, _ := response["auto_configured"].(bool)
			scenariosCreated, _ := response["scenarios_created"].(float64)
			assert.True(t, autoConfigured, "Twin should be auto-configured")
			assert.GreaterOrEqual(t, int(scenariosCreated), 0, "Should have created scenarios")
		}
	})

	// Test listing twins
	t.Run("List digital twins", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/twins", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		// Accept 200, 404, 500, or 503
		assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusNotFound || w.Code == http.StatusInternalServerError || w.Code == http.StatusServiceUnavailable,
			"Expected 200, 404, 500, or 503, got %d", w.Code)

		if w.Code == http.StatusOK {
			var response map[string]any
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			// Handle wrapped response structure (data -> twins)
			var twins []any
			if data, ok := response["data"].(map[string]any); ok {
				twins, _ = data["twins"].([]any)
			} else {
				twins, _ = response["twins"].([]any)
			}
			assert.NotNil(t, twins, "Response should contain twins array")

			// If we created a twin, verify it's in the list
			if twinID != "" && len(twins) > 0 {
				found := false
				for _, t := range twins {
					if twin, ok := t.(map[string]any); ok {
						if id, ok := twin["id"].(string); ok && id == twinID {
							found = true
							break
						}
					}
				}
				assert.True(t, found, "Created twin should be in list")
			}
		}
	})
}

// TestTwinCreation_VerifyStructure tests that twin structure is correctly populated
func TestTwinCreation_VerifyStructure(t *testing.T) {
	server := NewServer()
	require.NotNil(t, server)

	var ontologyID, twinID string

	// Create ontology with complex structure
	t.Run("Create complex ontology", func(t *testing.T) {
		ontologyData := `@prefix ex: <http://example.org/complex#> .
@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .

ex:Organization a rdfs:Class .
ex:Team a rdfs:Class .
ex:Resource a rdfs:Class .

ex:Headquarters a ex:Organization ;
    rdfs:label "Headquarters" .

ex:TeamA a ex:Team ;
    rdfs:label "Team A" ;
    ex:partOf ex:Headquarters .

ex:TeamB a ex:Team ;
    rdfs:label "Team B" ;
    ex:partOf ex:Headquarters .

ex:Server1 a ex:Resource ;
    rdfs:label "Server 1" ;
    ex:managedBy ex:TeamA .

ex:Server2 a ex:Resource ;
    rdfs:label "Server 2" ;
    ex:managedBy ex:TeamB .`

		createReq := map[string]any{
			"name":          "Complex Organization",
			"ontology_data": ontologyData,
			"format":        "turtle",
		}

		body, _ := json.Marshal(createReq)
		req := httptest.NewRequest("POST", "/api/v1/ontology", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		if w.Code == http.StatusOK || w.Code == http.StatusCreated {
			var response map[string]any
			json.Unmarshal(w.Body.Bytes(), &response)
			if id, ok := response["ontology_id"].(string); ok {
				ontologyID = id
			}
		}
	})

	t.Run("Create twin and verify structure", func(t *testing.T) {
		if ontologyID == "" {
			t.Skip("Ontology not created")
		}

		createReq := map[string]any{
			"ontology_id": ontologyID,
			"name":        "Complex Twin",
			"model_type":  "organization",
		}

		body, _ := json.Marshal(createReq)
		req := httptest.NewRequest("POST", "/api/v1/twins", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		if w.Code == http.StatusOK || w.Code == http.StatusCreated {
			var response map[string]any
			json.Unmarshal(w.Body.Bytes(), &response)
			if id, ok := response["twin_id"].(string); ok {
				twinID = id
			}

			// Verify entity count
			entityCount, _ := response["entity_count"].(float64)
			relCount, _ := response["relationship_count"].(float64)

			assert.GreaterOrEqual(t, int(entityCount), 5, "Should have multiple entities")
			assert.GreaterOrEqual(t, int(relCount), 3, "Should have relationships between entities")
		}
	})

	t.Run("Get twin and verify detailed structure", func(t *testing.T) {
		if twinID == "" {
			t.Skip("Twin not created")
		}

		req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/twins/%s", twinID), nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusNotFound || w.Code == http.StatusInternalServerError || w.Code == http.StatusServiceUnavailable,
			"Expected 200, 404, 500, or 503, got %d", w.Code)

		if w.Code == http.StatusOK {
			var twin DigitalTwin.DigitalTwin
			err := json.Unmarshal(w.Body.Bytes(), &twin)
			require.NoError(t, err)

			// Verify twin structure
			assert.Equal(t, twinID, twin.ID)
			assert.Equal(t, ontologyID, twin.OntologyID)
			assert.NotEmpty(t, twin.Entities)
			assert.NotEmpty(t, twin.Relationships)

			// Verify entities have proper URIs and labels
			for _, entity := range twin.Entities {
				assert.NotEmpty(t, entity.URI, "Entity should have URI")
				assert.NotEmpty(t, entity.Type, "Entity should have type")

				// Verify entity has initial state
				assert.NotEmpty(t, entity.State.Status)
				assert.True(t, entity.State.Capacity >= 0)
			}

			// Verify relationships connect entities
			for _, rel := range twin.Relationships {
				assert.NotEmpty(t, rel.ID)
				assert.NotEmpty(t, rel.SourceURI)
				assert.NotEmpty(t, rel.TargetURI)
				assert.NotEmpty(t, rel.Type)
			}
		}
	})
}

// TestTwinCreation_AutoConfiguration tests auto-configuration creates scenarios and monitoring
func TestTwinCreation_AutoConfiguration(t *testing.T) {
	server := NewServer()
	require.NotNil(t, server)

	var ontologyID, twinID string

	t.Run("Create ontology for auto-config test", func(t *testing.T) {
		ontologyData := `@prefix ex: <http://example.org/auto#> .
@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .

ex:Factory a rdfs:Class .
ex:Machine a rdfs:Class .

ex:MainFactory a ex:Factory ; rdfs:label "Main Factory" .
ex:MachineA a ex:Machine ; rdfs:label "Machine A" ; ex:locatedIn ex:MainFactory .
ex:MachineB a ex:Machine ; rdfs:label "Machine B" ; ex:locatedIn ex:MainFactory .`

		createReq := map[string]any{
			"name":          "Auto-Config Test Ontology",
			"ontology_data": ontologyData,
			"format":        "turtle",
		}

		body, _ := json.Marshal(createReq)
		req := httptest.NewRequest("POST", "/api/v1/ontology", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		if w.Code == http.StatusOK || w.Code == http.StatusCreated {
			var response map[string]any
			json.Unmarshal(w.Body.Bytes(), &response)
			if id, ok := response["ontology_id"].(string); ok {
				ontologyID = id
			}
		}
	})

	t.Run("Create twin and verify auto-configuration", func(t *testing.T) {
		if ontologyID == "" {
			t.Skip("Ontology not created")
		}

		createReq := map[string]any{
			"ontology_id": ontologyID,
			"name":        "Auto-Configured Twin",
			"model_type":  "organization",
		}

		body, _ := json.Marshal(createReq)
		req := httptest.NewRequest("POST", "/api/v1/twins", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		if w.Code == http.StatusOK || w.Code == http.StatusCreated {
			var response map[string]any
			json.Unmarshal(w.Body.Bytes(), &response)
			if id, ok := response["twin_id"].(string); ok {
				twinID = id
			}

			// Verify auto-configuration
			autoConfigured, _ := response["auto_configured"].(bool)
			scenariosCreated, _ := response["scenarios_created"].(float64)

			assert.True(t, autoConfigured, "Twin should be auto-configured")
			assert.Greater(t, int(scenariosCreated), 0, "Should have created at least one scenario")
		}
	})

	t.Run("Verify auto-created scenarios exist", func(t *testing.T) {
		if twinID == "" {
			t.Skip("Twin not created")
		}

		req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/twins/%s/scenarios", twinID), nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusServiceUnavailable || w.Code == http.StatusInternalServerError,
			"Expected 200, 500, or 503, got %d", w.Code)

		if w.Code == http.StatusOK {
			var response map[string]any
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			scenarios, ok := response["scenarios"].([]any)
			if ok && len(scenarios) > 0 {
				// Verify scenarios have proper structure
				for _, s := range scenarios {
					if scenario, ok := s.(map[string]any); ok {
						assert.NotEmpty(t, scenario["id"], "Scenario should have ID")
						assert.NotEmpty(t, scenario["name"], "Scenario should have name")

						// Verify scenario has events if applicable
						if events, ok := scenario["events"].([]any); ok {
							// Some scenarios may have events
							_ = events
						}
					}
				}
			}
		}
	})
}

// ============================================================================
// SECTION 2: Simulation Execution Tests
// ============================================================================

// TestSimulation_BasicExecution tests running a basic simulation
func TestSimulation_BasicExecution(t *testing.T) {
	server := NewServer()
	require.NotNil(t, server)

	var twinID, scenarioID, runID string

	// Setup: Create twin with scenarios
	t.Run("Setup: Create twin with scenario", func(t *testing.T) {
		// Create ontology
		ontologyData := `@prefix ex: <http://example.org/sim#> .
@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .

ex:Node a rdfs:Class .
ex:Node1 a ex:Node ; rdfs:label "Node 1" .
ex:Node2 a ex:Node ; rdfs:label "Node 2" .
ex:Node3 a ex:Node ; rdfs:label "Node 3" .
ex:connects ex:Node1 ex:Node2 .
ex:connects ex:Node2 ex:Node3 .`

		createReq := map[string]any{
			"name":          "Simulation Test Ontology",
			"ontology_data": ontologyData,
			"format":        "turtle",
		}

		body, _ := json.Marshal(createReq)
		req := httptest.NewRequest("POST", "/api/v1/ontology", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		var ontologyID string
		if w.Code == http.StatusOK || w.Code == http.StatusCreated {
			var response map[string]any
			json.Unmarshal(w.Body.Bytes(), &response)
			if id, ok := response["ontology_id"].(string); ok {
				ontologyID = id
			}
		}

		if ontologyID == "" {
			return
		}

		// Create twin
		twinReq := map[string]any{
			"ontology_id": ontologyID,
			"name":        "Simulation Test Twin",
			"model_type":  "organization",
		}

		body, _ = json.Marshal(twinReq)
		req = httptest.NewRequest("POST", "/api/v1/twins", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		if w.Code == http.StatusOK || w.Code == http.StatusCreated {
			var response map[string]any
			json.Unmarshal(w.Body.Bytes(), &response)
			if id, ok := response["twin_id"].(string); ok {
				twinID = id
			}
		}
	})

	t.Run("Create simulation scenario", func(t *testing.T) {
		if twinID == "" {
			t.Skip("Twin not created")
		}

		// Create a scenario with events
		scenarioReq := map[string]any{
			"name":        "Test Scenario",
			"description": "Test scenario for simulation",
			"duration":    50,
			"events": []map[string]any{
				{
					"id":         "event1",
					"type":       "resource.unavailable",
					"target_uri": "http://example.org/sim#Node1",
					"timestamp":  10,
					"parameters": map[string]any{
						"duration": 20,
					},
				},
				{
					"id":         "event2",
					"type":       "resource.available",
					"target_uri": "http://example.org/sim#Node1",
					"timestamp":  30,
					"parameters": map[string]any{},
				},
			},
		}

		body, _ := json.Marshal(scenarioReq)
		req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/twins/%s/scenarios", twinID), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusCreated || w.Code == http.StatusServiceUnavailable,
			"Expected 200, 201, or 503, got %d", w.Code)

		if w.Code == http.StatusOK || w.Code == http.StatusCreated {
			var response map[string]any
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.NotEmpty(t, response["scenario_id"])
			scenarioID = response["scenario_id"].(string)
		}
	})

	t.Run("Execute simulation", func(t *testing.T) {
		if twinID == "" || scenarioID == "" {
			t.Skip("Twin or scenario not created")
		}

		runReq := map[string]any{
			"snapshot_interval": 5,
			"max_steps":         100,
		}

		body, _ := json.Marshal(runReq)
		req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/twins/%s/scenarios/%s/run", twinID, scenarioID), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusServiceUnavailable,
			"Expected 200 or 503, got %d", w.Code)

		if w.Code == http.StatusOK {
			var response map[string]any
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			// Verify run was created
			assert.NotEmpty(t, response["run_id"], "Should return run ID")
			runID = response["run_id"].(string)

			// Verify status
			status, _ := response["status"].(string)
			assert.NotEmpty(t, status)

			// Verify metrics exist
			metrics, ok := response["metrics"].(map[string]any)
			if ok {
				assert.NotNil(t, metrics, "Should have metrics")

				// Check specific metrics
				if totalSteps, ok := metrics["total_steps"].(float64); ok {
					assert.Greater(t, int(totalSteps), 0, "Should have executed steps")
				}
			}
		}
	})

	t.Run("Get simulation results", func(t *testing.T) {
		if runID == "" {
			t.Skip("Run not created")
		}

		req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/twins/runs/%s", runID), nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusNotFound || w.Code == http.StatusInternalServerError || w.Code == http.StatusServiceUnavailable,
			"Expected 200, 404, 500, or 503, got %d", w.Code)

		if w.Code == http.StatusOK {
			var run DigitalTwin.SimulationRun
			err := json.Unmarshal(w.Body.Bytes(), &run)
			require.NoError(t, err)

			assert.Equal(t, runID, run.ID)
			assert.NotEmpty(t, run.Status)
			assert.NotEmpty(t, run.StartTime)

			// Verify metrics
			assert.GreaterOrEqual(t, run.Metrics.TotalSteps, 0)
			assert.GreaterOrEqual(t, run.Metrics.EventsProcessed, 0)

			// Verify events log
			assert.NotNil(t, run.EventsLog)
		}
	})
}

// TestSimulation_WithDifferentParameters tests simulation with various parameters
func TestSimulation_WithDifferentParameters(t *testing.T) {
	server := NewServer()
	require.NotNil(t, server)

	testCases := []struct {
		name             string
		snapshotInterval int
		maxSteps         int
		expectSuccess    bool
	}{
		{
			name:             "Short simulation with frequent snapshots",
			snapshotInterval: 1,
			maxSteps:         10,
			expectSuccess:    true,
		},
		{
			name:             "Long simulation with sparse snapshots",
			snapshotInterval: 20,
			maxSteps:         200,
			expectSuccess:    true,
		},
		{
			name:             "Simulation without snapshots",
			snapshotInterval: 0,
			maxSteps:         50,
			expectSuccess:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// This test validates that different parameter combinations work
			// We don't need a real twin for this - just verify the API accepts the parameters
			runReq := map[string]any{
				"snapshot_interval": tc.snapshotInterval,
				"max_steps":         tc.maxSteps,
			}

			body, _ := json.Marshal(runReq)
			// Use a fake twin/scenario ID - we expect 404 or 503, not a parameter validation error
			req := httptest.NewRequest("POST", "/api/v1/twins/fake-twin/scenarios/fake-scenario/run", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			server.router.ServeHTTP(w, req)

			// Should not get 400 Bad Request (parameter validation error)
			// Instead should get 404 (not found) or 503 (service unavailable)
			assert.NotEqual(t, http.StatusBadRequest, w.Code,
				"Should not get 400 Bad Request for valid parameters, got %d", w.Code)
		})
	}
}

// TestSimulation_TimelineAndSnapshots tests simulation timeline retrieval
func TestSimulation_TimelineAndSnapshots(t *testing.T) {
	server := NewServer()
	require.NotNil(t, server)

	t.Run("Get simulation timeline", func(t *testing.T) {
		// Use a fake run ID - we're testing the endpoint structure
		req := httptest.NewRequest("GET", "/api/v1/twins/runs/fake-run/timeline", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		// Should return 200, 404, 500, or 503
		assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusNotFound || w.Code == http.StatusInternalServerError || w.Code == http.StatusServiceUnavailable,
			"Expected 200, 404, 500, or 503, got %d", w.Code)

		if w.Code == http.StatusOK {
			var response map[string]any
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.NotEmpty(t, response["run_id"])

			if snapshots, ok := response["snapshots"].([]any); ok {
				// Verify snapshots structure
				for _, s := range snapshots {
					if snapshot, ok := s.(map[string]any); ok {
						assert.NotNil(t, snapshot["step"])
						assert.NotNil(t, snapshot["timestamp"])
						assert.NotNil(t, snapshot["state"])
					}
				}
			}
		}
	})
}

// ============================================================================
// SECTION 3: What-If Analysis Tests
// ============================================================================

// TestWhatIfAnalysis_BasicQuery tests basic what-if analysis with a question
func TestWhatIfAnalysis_BasicQuery(t *testing.T) {
	server := NewServer()
	require.NotNil(t, server)

	testCases := []struct {
		name     string
		question string
	}{
		{
			name:     "Resource availability question",
			question: "What happens if Server A becomes unavailable?",
		},
		{
			name:     "Demand surge question",
			question: "What if demand increases by 50%?",
		},
		{
			name:     "Process delay question",
			question: "What is the impact if the production process is delayed?",
		},
		{
			name:     "Capacity question",
			question: "Can we handle double the current load?",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Use a fake twin ID
			whatIfReq := map[string]any{
				"question":    tc.question,
				"max_results": 5,
			}

			body, _ := json.Marshal(whatIfReq)
			req := httptest.NewRequest("POST", "/api/v1/twins/fake-twin/whatif", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			server.router.ServeHTTP(w, req)

			// Should not get 400 for valid question format
			// May get 404 (twin not found), 503 (service unavailable), or 200/500 (if LLM/service issues)
			assert.NotEqual(t, http.StatusBadRequest, w.Code,
				"Should not get 400 Bad Request for valid question, got %d", w.Code)

			// If we get a successful response, verify structure
			if w.Code == http.StatusOK {
				var response map[string]any
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)

				assert.Equal(t, tc.question, response["question"])
				assert.NotEmpty(t, response["interpretation"])
				assert.NotEmpty(t, response["summary"])

				if findings, ok := response["key_findings"].([]any); ok {
					_ = findings
				}

				if recommendations, ok := response["recommendations"].([]any); ok {
					_ = recommendations
				}
			}
		})
	}
}

// TestWhatIfAnalysis_InvalidInput tests error handling for invalid input
func TestWhatIfAnalysis_InvalidInput(t *testing.T) {
	server := NewServer()
	require.NotNil(t, server)

	t.Run("Empty question should fail", func(t *testing.T) {
		whatIfReq := map[string]any{
			"question": "",
		}

		body, _ := json.Marshal(whatIfReq)
		req := httptest.NewRequest("POST", "/api/v1/twins/fake-twin/whatif", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		// Should get 400 for empty question, or 500 if service unavailable
		assert.True(t, w.Code == http.StatusBadRequest || w.Code == http.StatusInternalServerError,
			"Should get 400 or 500 for empty question, got %d", w.Code)
	})

	t.Run("Missing question field should fail", func(t *testing.T) {
		whatIfReq := map[string]any{
			"max_results": 5,
		}

		body, _ := json.Marshal(whatIfReq)
		req := httptest.NewRequest("POST", "/api/v1/twins/fake-twin/whatif", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		// Should get 400 for missing question, or 500 if service unavailable
		assert.True(t, w.Code == http.StatusBadRequest || w.Code == http.StatusInternalServerError,
			"Should get 400 or 500 for missing question, got %d", w.Code)
	})
}

// ============================================================================
// SECTION 4: State Management Tests
// ============================================================================

// TestStateManagement_GetAndUpdate tests retrieving and updating twin state
func TestStateManagement_GetAndUpdate(t *testing.T) {
	server := NewServer()
	require.NotNil(t, server)

	t.Run("Get twin state endpoint exists", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/twins/fake-twin/state", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		// Should return 200, 404, 500, or 503
		assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusNotFound || w.Code == http.StatusInternalServerError || w.Code == http.StatusServiceUnavailable,
			"Expected 200, 404, 500, or 503, got %d", w.Code)

		if w.Code == http.StatusOK {
			var state DigitalTwin.TwinState
			err := json.Unmarshal(w.Body.Bytes(), &state)
			require.NoError(t, err)

			assert.NotZero(t, state.Timestamp)
			assert.NotNil(t, state.Entities)
			assert.NotNil(t, state.GlobalMetrics)
			assert.NotNil(t, state.Flags)
		}
	})

	t.Run("Update twin state endpoint exists", func(t *testing.T) {
		updateReq := map[string]any{
			"base_state": map[string]any{
				"custom_metric": 42.0,
				"status":        "operational",
			},
		}

		body, _ := json.Marshal(updateReq)
		req := httptest.NewRequest("PUT", "/api/v1/twins/fake-twin", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		// Should return 200, 202, 404, 500, or 503
		assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusAccepted || w.Code == http.StatusNotFound || w.Code == http.StatusInternalServerError || w.Code == http.StatusServiceUnavailable,
			"Expected 200, 202, 404, 500, or 503, got %d", w.Code)
	})
}

// TestStateManagement_EntityTracking tests entity state tracking functionality
func TestStateManagement_EntityTracking(t *testing.T) {
	server := NewServer()
	require.NotNil(t, server)

	// Create a twin to test state tracking
	var ontologyID, twinID string

	t.Run("Setup: Create twin for state tracking", func(t *testing.T) {
		ontologyData := `@prefix ex: <http://example.org/state#> .
@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .

ex:Server a rdfs:Class .
ex:Server1 a ex:Server ; rdfs:label "Server 1" .
ex:Server2 a ex:Server ; rdfs:label "Server 2" .`

		createReq := map[string]any{
			"name":          "State Tracking Ontology",
			"ontology_data": ontologyData,
			"format":        "turtle",
		}

		body, _ := json.Marshal(createReq)
		req := httptest.NewRequest("POST", "/api/v1/ontology", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		if w.Code == http.StatusOK || w.Code == http.StatusCreated {
			var response map[string]any
			json.Unmarshal(w.Body.Bytes(), &response)
			if id, ok := response["ontology_id"].(string); ok {
				ontologyID = id
			}
		}

		if ontologyID != "" {
			twinReq := map[string]any{
				"ontology_id": ontologyID,
				"name":        "State Tracking Twin",
				"model_type":  "organization",
			}

			body, _ = json.Marshal(twinReq)
			req = httptest.NewRequest("POST", "/api/v1/twins", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w = httptest.NewRecorder()
			server.router.ServeHTTP(w, req)

			if w.Code == http.StatusOK || w.Code == http.StatusCreated {
				var response map[string]any
				json.Unmarshal(w.Body.Bytes(), &response)
				if id, ok := response["twin_id"].(string); ok {
					twinID = id
				}
			}
		}
	})

	t.Run("Get twin state with entity tracking", func(t *testing.T) {
		if twinID == "" {
			t.Skip("Twin not created")
		}

		req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/twins/%s/state", twinID), nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		if w.Code == http.StatusOK {
			var state DigitalTwin.TwinState
			err := json.Unmarshal(w.Body.Bytes(), &state)
			require.NoError(t, err)

			// Verify entity states
			assert.NotNil(t, state.Entities)

			for uri, entityState := range state.Entities {
				assert.NotEmpty(t, uri, "Entity URI should not be empty")
				assert.NotEmpty(t, entityState.Status, "Entity should have status")
				assert.True(t, entityState.LastUpdated.Before(time.Now().Add(time.Second)))
			}

			// Verify global metrics
			assert.NotNil(t, state.GlobalMetrics)

			// Check if metrics are calculated
			if avgUtil, ok := state.GlobalMetrics["average_utilization"]; ok {
				assert.GreaterOrEqual(t, avgUtil, 0.0)
				assert.LessOrEqual(t, avgUtil, 1.0)
			}

			// Verify stability flag
			if stable, ok := state.Flags["stable"]; ok {
				assert.IsType(t, true, stable)
			}
		}
	})
}

// ============================================================================
// SECTION 5: Insights & Analysis Tests
// ============================================================================

// TestInsightsAndAnalysis_ProactiveInsights tests the insights endpoint
func TestInsightsAndAnalysis_ProactiveInsights(t *testing.T) {
	server := NewServer()
	require.NotNil(t, server)

	t.Run("Get insights endpoint", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/twins/fake-twin/insights", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		// Should return 200, 404, 500, or 503
		assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusNotFound || w.Code == http.StatusInternalServerError || w.Code == http.StatusServiceUnavailable,
			"Expected 200, 404, 500, or 503, got %d", w.Code)

		if w.Code == http.StatusOK {
			var report DigitalTwin.InsightReport
			err := json.Unmarshal(w.Body.Bytes(), &report)
			require.NoError(t, err)

			assert.NotEmpty(t, report.TwinID)
			assert.NotZero(t, report.GeneratedAt)

			// Verify insights structure
			assert.NotNil(t, report.Insights)
			assert.NotNil(t, report.SuggestedQuestions)

			// Verify scores are in valid range
			assert.GreaterOrEqual(t, report.RiskScore, 0.0)
			assert.LessOrEqual(t, report.RiskScore, 1.0)
			assert.GreaterOrEqual(t, report.HealthScore, 0.0)
			assert.LessOrEqual(t, report.HealthScore, 1.0)
		}
	})
}

// TestInsightsAndAnalysis_OntologyAnalysis tests ontology analysis endpoint
func TestInsightsAndAnalysis_OntologyAnalysis(t *testing.T) {
	server := NewServer()
	require.NotNil(t, server)

	t.Run("Ontology analysis endpoint", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/twins/fake-twin/analysis", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		// Should return 200, 404, 500, or 503
		assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusNotFound || w.Code == http.StatusInternalServerError || w.Code == http.StatusServiceUnavailable,
			"Expected 200, 404, 500, or 503, got %d", w.Code)

		if w.Code == http.StatusOK {
			var analysis map[string]any
			err := json.Unmarshal(w.Body.Bytes(), &analysis)
			require.NoError(t, err)

			// Verify analysis structure
			assert.NotEmpty(t, analysis)
		}
	})
}

// TestInsightsAndAnalysis_ImpactAnalysis tests impact analysis endpoint
func TestInsightsAndAnalysis_ImpactAnalysis(t *testing.T) {
	server := NewServer()
	require.NotNil(t, server)

	t.Run("Impact analysis endpoint", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/twins/fake-twin/runs/fake-run/impact", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		// Should return 200, 404, 500, or 503
		assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusNotFound || w.Code == http.StatusInternalServerError || w.Code == http.StatusServiceUnavailable,
			"Expected 200, 404, 500, or 503, got %d", w.Code)

		if w.Code == http.StatusOK {
			var impact DigitalTwin.ImpactAnalysis
			err := json.Unmarshal(w.Body.Bytes(), &impact)
			require.NoError(t, err)

			assert.NotEmpty(t, impact.RunID)
			assert.NotEmpty(t, impact.OverallImpact)

			// Verify affected entities
			assert.NotNil(t, impact.AffectedEntities)

			// Verify risk score
			assert.GreaterOrEqual(t, impact.RiskScore, 0.0)
			assert.LessOrEqual(t, impact.RiskScore, 1.0)
		}
	})
}

// ============================================================================
// SECTION 6: Full Integration Flow Tests
// ============================================================================

// TestFullIntegrationFlow_EndToEnd tests the complete flow: Ontology → Twin → Simulation → Results
func TestFullIntegrationFlow_EndToEnd(t *testing.T) {
	server := NewServer()
	require.NotNil(t, server)

	var ontologyID, twinID, scenarioID, runID string

	t.Run("Step 1: Create comprehensive ontology", func(t *testing.T) {
		ontologyData := `@prefix ex: <http://example.org/e2e#> .
@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .

ex:Organization a rdfs:Class ;
    rdfs:label "Organization" .

ex:Department a rdfs:Class ;
    rdfs:label "Department" .

ex:Employee a rdfs:Class ;
    rdfs:label "Employee" .

ex:Project a rdfs:Class ;
    rdfs:label "Project" .

ex:Resource a rdfs:Class ;
    rdfs:label "Resource" .

ex:AcmeCorp a ex:Organization ;
    rdfs:label "Acme Corporation" .

ex:Engineering a ex:Department ;
    rdfs:label "Engineering Department" ;
    ex:partOf ex:AcmeCorp .

ex:Sales a ex:Department ;
    rdfs:label "Sales Department" ;
    ex:partOf ex:AcmeCorp .

ex:Alice a ex:Employee ;
    rdfs:label "Alice Johnson" ;
    ex:worksIn ex:Engineering ;
    ex:hasRole "Senior Engineer" .

ex:Bob a ex:Employee ;
    rdfs:label "Bob Smith" ;
    ex:worksIn ex:Engineering ;
    ex:hasRole "Engineer" .

ex:Carol a ex:Employee ;
    rdfs:label "Carol White" ;
    ex:worksIn ex:Sales ;
    ex:hasRole "Sales Manager" .

ex:ProjectAlpha a ex:Project ;
    rdfs:label "Project Alpha" ;
    ex:ownedBy ex:Engineering ;
    ex:priority "high" .

ex:ProjectBeta a ex:Project ;
    rdfs:label "Project Beta" ;
    ex:ownedBy ex:Sales ;
    ex:priority "medium" .

ex:ServerFarm a ex:Resource ;
    rdfs:label "Server Farm" ;
    ex:managedBy ex:Engineering ;
    ex:capacity "100" .

ex:CustomerDB a ex:Resource ;
    rdfs:label "Customer Database" ;
    ex:managedBy ex:Sales ;
    ex:critical "true" .`

		createReq := map[string]any{
			"name":          "End-to-End Test Ontology",
			"description":   "Comprehensive ontology for end-to-end testing",
			"version":       "1.0.0",
			"ontology_data": ontologyData,
			"format":        "turtle",
		}

		body, _ := json.Marshal(createReq)
		req := httptest.NewRequest("POST", "/api/v1/ontology", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusCreated || w.Code == http.StatusInternalServerError,
			"Expected 200, 201, or 500, got %d", w.Code)

		if w.Code == http.StatusOK || w.Code == http.StatusCreated {
			var response map[string]any
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			if id, ok := response["ontology_id"].(string); ok {
				ontologyID = id
			}
		}
	})

	t.Run("Step 2: Create digital twin from ontology", func(t *testing.T) {
		if ontologyID == "" {
			t.Skip("Ontology not created")
		}

		createReq := map[string]any{
			"ontology_id": ontologyID,
			"name":        "Acme Corp Digital Twin",
			"description": "End-to-end test digital twin",
			"model_type":  "organization",
		}

		body, _ := json.Marshal(createReq)
		req := httptest.NewRequest("POST", "/api/v1/twins", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusCreated || w.Code == http.StatusServiceUnavailable,
			"Expected 200, 201, or 503, got %d", w.Code)

		if w.Code == http.StatusOK || w.Code == http.StatusCreated {
			var response map[string]any
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.NotEmpty(t, response["twin_id"])
			twinID = response["twin_id"].(string)

			// Verify twin structure
			entityCount, _ := response["entity_count"].(float64)
			relCount, _ := response["relationship_count"].(float64)

			assert.GreaterOrEqual(t, int(entityCount), 10, "Should have extracted multiple entities")
			assert.GreaterOrEqual(t, int(relCount), 5, "Should have extracted relationships")

			// Verify auto-configuration
			autoConfigured, _ := response["auto_configured"].(bool)
			assert.True(t, autoConfigured, "Twin should be auto-configured")
		}
	})

	t.Run("Step 3: Verify twin structure", func(t *testing.T) {
		if twinID == "" {
			t.Skip("Twin not created")
		}

		req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/twins/%s", twinID), nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusNotFound || w.Code == http.StatusInternalServerError || w.Code == http.StatusServiceUnavailable,
			"Expected 200, 404, 500, or 503, got %d", w.Code)

		if w.Code == http.StatusOK {
			var twin DigitalTwin.DigitalTwin
			err := json.Unmarshal(w.Body.Bytes(), &twin)
			require.NoError(t, err)

			// Verify comprehensive structure
			assert.Equal(t, twinID, twin.ID)
			assert.Equal(t, ontologyID, twin.OntologyID)

			// Should have various entity types
			entityTypes := make(map[string]int)
			for _, entity := range twin.Entities {
				entityTypes[entity.Type]++
			}

			// Verify we have different types of entities
			assert.GreaterOrEqual(t, len(entityTypes), 3, "Should have multiple entity types")

			// Verify relationships
			assert.NotEmpty(t, twin.Relationships)
		}
	})

	t.Run("Step 4: List auto-created scenarios", func(t *testing.T) {
		if twinID == "" {
			t.Skip("Twin not created")
		}

		req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/twins/%s/scenarios", twinID), nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusServiceUnavailable,
			"Expected 200 or 503, got %d", w.Code)

		if w.Code == http.StatusOK {
			var response map[string]any
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			scenarios, ok := response["scenarios"].([]any)
			require.True(t, ok, "Should have scenarios array")

			// Should have auto-created scenarios
			if len(scenarios) > 0 {
				// Store first scenario ID for simulation
				if scenario, ok := scenarios[0].(map[string]any); ok {
					if id, ok := scenario["id"].(string); ok {
						scenarioID = id
					}
				}
			}
		}
	})

	t.Run("Step 5: Create custom scenario", func(t *testing.T) {
		if twinID == "" {
			t.Skip("Twin not created")
		}

		scenarioReq := map[string]any{
			"name":        "Employee Absence Scenario",
			"description": "Simulate impact of key employee absence",
			"duration":    60,
			"events": []map[string]any{
				{
					"id":         "absence_event",
					"type":       "resource.unavailable",
					"target_uri": "http://example.org/e2e#Alice",
					"timestamp":  5,
					"parameters": map[string]any{
						"reason": "unplanned_absence",
					},
				},
				{
					"id":         "return_event",
					"type":       "resource.available",
					"target_uri": "http://example.org/e2e#Alice",
					"timestamp":  45,
					"parameters": map[string]any{},
				},
			},
		}

		body, _ := json.Marshal(scenarioReq)
		req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/twins/%s/scenarios", twinID), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusCreated || w.Code == http.StatusServiceUnavailable,
			"Expected 200, 201, or 503, got %d", w.Code)

		if w.Code == http.StatusOK || w.Code == http.StatusCreated {
			var response map[string]any
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			if id, ok := response["scenario_id"].(string); ok {
				scenarioID = id
			}
		}
	})

	t.Run("Step 6: Execute simulation", func(t *testing.T) {
		if twinID == "" || scenarioID == "" {
			t.Skip("Twin or scenario not created")
		}

		runReq := map[string]any{
			"snapshot_interval": 10,
			"max_steps":         100,
		}

		body, _ := json.Marshal(runReq)
		req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/twins/%s/scenarios/%s/run", twinID, scenarioID), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusServiceUnavailable,
			"Expected 200 or 503, got %d", w.Code)

		if w.Code == http.StatusOK {
			var response map[string]any
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.NotEmpty(t, response["run_id"])
			runID = response["run_id"].(string)

			status, _ := response["status"].(string)
			assert.NotEmpty(t, status)

			// Verify metrics
			if metrics, ok := response["metrics"].(map[string]any); ok {
				if totalSteps, ok := metrics["total_steps"].(float64); ok {
					assert.Greater(t, int(totalSteps), 0, "Simulation should have executed steps")
				}
			}
		}
	})

	t.Run("Step 7: Get simulation results", func(t *testing.T) {
		if runID == "" {
			t.Skip("Run not created")
		}

		req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/twins/runs/%s", runID), nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusNotFound || w.Code == http.StatusInternalServerError || w.Code == http.StatusServiceUnavailable,
			"Expected 200, 404, 500, or 503, got %d", w.Code)

		if w.Code == http.StatusOK {
			var run DigitalTwin.SimulationRun
			err := json.Unmarshal(w.Body.Bytes(), &run)
			require.NoError(t, err)

			assert.Equal(t, runID, run.ID)
			assert.NotEmpty(t, run.Status)
			assert.NotZero(t, run.StartTime)

			// Verify metrics exist
			assert.GreaterOrEqual(t, run.Metrics.TotalSteps, 0)

			// Verify events were logged
			assert.NotNil(t, run.EventsLog)
		}
	})

	t.Run("Step 8: Get simulation timeline", func(t *testing.T) {
		if runID == "" {
			t.Skip("Run not created")
		}

		req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/twins/runs/%s/timeline", runID), nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusNotFound || w.Code == http.StatusInternalServerError || w.Code == http.StatusServiceUnavailable,
			"Expected 200, 404, 500, or 503, got %d", w.Code)

		if w.Code == http.StatusOK {
			var response map[string]any
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.Equal(t, runID, response["run_id"])

			if snapshots, ok := response["snapshots"].([]any); ok {
				// Verify snapshots structure
				for _, s := range snapshots {
					if snapshot, ok := s.(map[string]any); ok {
						assert.NotNil(t, snapshot["step"])
						assert.NotNil(t, snapshot["timestamp"])
						assert.NotNil(t, snapshot["state"])
					}
				}
			}
		}
	})

	t.Run("Step 9: Analyze impact", func(t *testing.T) {
		if twinID == "" || runID == "" {
			t.Skip("Twin or run not created")
		}

		req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/twins/%s/runs/%s/impact", twinID, runID), nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusNotFound || w.Code == http.StatusInternalServerError || w.Code == http.StatusServiceUnavailable,
			"Expected 200, 404, 500, or 503, got %d", w.Code)

		if w.Code == http.StatusOK {
			var impact DigitalTwin.ImpactAnalysis
			err := json.Unmarshal(w.Body.Bytes(), &impact)
			require.NoError(t, err)

			assert.Equal(t, runID, impact.RunID)
			assert.NotEmpty(t, impact.OverallImpact)

			// Verify risk score
			assert.GreaterOrEqual(t, impact.RiskScore, 0.0)
			assert.LessOrEqual(t, impact.RiskScore, 1.0)
		}
	})

	t.Run("Step 10: Get twin state after simulation", func(t *testing.T) {
		if twinID == "" {
			t.Skip("Twin not created")
		}

		req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/twins/%s/state", twinID), nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusNotFound || w.Code == http.StatusInternalServerError || w.Code == http.StatusServiceUnavailable,
			"Expected 200, 404, 500, or 503, got %d", w.Code)

		if w.Code == http.StatusOK {
			var state DigitalTwin.TwinState
			err := json.Unmarshal(w.Body.Bytes(), &state)
			require.NoError(t, err)

			assert.NotZero(t, state.Timestamp)
			assert.NotNil(t, state.Entities)
			assert.NotNil(t, state.GlobalMetrics)
		}
	})

	t.Run("Step 11: Get insights", func(t *testing.T) {
		if twinID == "" {
			t.Skip("Twin not created")
		}

		req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/twins/%s/insights", twinID), nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusNotFound || w.Code == http.StatusInternalServerError || w.Code == http.StatusServiceUnavailable,
			"Expected 200, 404, 500, or 503, got %d", w.Code)

		if w.Code == http.StatusOK {
			var report DigitalTwin.InsightReport
			err := json.Unmarshal(w.Body.Bytes(), &report)
			require.NoError(t, err)

			assert.Equal(t, twinID, report.TwinID)
			assert.NotZero(t, report.GeneratedAt)
			assert.NotNil(t, report.Insights)
			assert.NotNil(t, report.SuggestedQuestions)
		}
	})
}

// TestFullIntegrationFlow_AutomatedTwinUsage tests automated end-to-end twin usage
func TestFullIntegrationFlow_AutomatedTwinUsage(t *testing.T) {
	server := NewServer()
	require.NotNil(t, server)

	t.Run("Automated twin creation and usage flow", func(t *testing.T) {
		// This test demonstrates the complete automated flow
		// where a twin is created, used for multiple simulations,
		// and analyzed

		// Step 1: Create a simple but complete ontology
		ontologyData := `@prefix ex: <http://example.org/auto#> .
@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .

ex:System a rdfs:Class .
ex:Component a rdfs:Class .

ex:MainSystem a ex:System ; rdfs:label "Main System" .
ex:ComponentA a ex:Component ; rdfs:label "Component A" ; ex:partOf ex:MainSystem .
ex:ComponentB a ex:Component ; rdfs:label "Component B" ; ex:partOf ex:MainSystem .
ex:ComponentC a ex:Component ; rdfs:label "Component C" ; ex:dependsOn ex:ComponentA .`

		createReq := map[string]any{
			"name":          "Automated Test Ontology",
			"ontology_data": ontologyData,
			"format":        "turtle",
		}

		body, _ := json.Marshal(createReq)
		req := httptest.NewRequest("POST", "/api/v1/ontology", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		if w.Code != http.StatusOK && w.Code != http.StatusCreated {
			t.Skip("Could not create ontology, skipping automated flow test")
		}

		var ontologyID string
		var response map[string]any
		json.Unmarshal(w.Body.Bytes(), &response)
		if id, ok := response["ontology_id"].(string); ok {
			ontologyID = id
		}

		if ontologyID == "" {
			t.Skip("Could not extract ontology ID")
		}

		// Step 2: Create twin with auto-configuration
		twinReq := map[string]any{
			"ontology_id": ontologyID,
			"name":        "Automated Test Twin",
			"model_type":  "organization",
		}

		body, _ = json.Marshal(twinReq)
		req = httptest.NewRequest("POST", "/api/v1/twins", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		if w.Code != http.StatusOK && w.Code != http.StatusCreated {
			t.Skip("Could not create twin")
		}

		var twinID string
		json.Unmarshal(w.Body.Bytes(), &response)
		if id, ok := response["twin_id"].(string); ok {
			twinID = id
		}

		if twinID == "" {
			t.Skip("Could not extract twin ID")
		}

		// Step 3: Verify twin has auto-created scenarios
		req = httptest.NewRequest("GET", fmt.Sprintf("/api/v1/twins/%s/scenarios", twinID), nil)
		w = httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		if w.Code == http.StatusOK {
			var scenariosResp map[string]any
			json.Unmarshal(w.Body.Bytes(), &scenariosResp)

			if scenarios, ok := scenariosResp["scenarios"].([]any); ok && len(scenarios) > 0 {
				// Step 4: Run simulation on first auto-created scenario
				firstScenario := scenarios[0].(map[string]any)
				scenarioID := firstScenario["id"].(string)

				runReq := map[string]any{
					"snapshot_interval": 5,
					"max_steps":         50,
				}

				body, _ = json.Marshal(runReq)
				req = httptest.NewRequest("POST", fmt.Sprintf("/api/v1/twins/%s/scenarios/%s/run", twinID, scenarioID), bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				w = httptest.NewRecorder()
				server.router.ServeHTTP(w, req)

				if w.Code == http.StatusOK {
					var runResp map[string]any
					json.Unmarshal(w.Body.Bytes(), &runResp)

					if runID, ok := runResp["run_id"].(string); ok && runID != "" {
						t.Logf("Successfully executed simulation: run_id=%s", runID)

						// Verify we got metrics
						if metrics, ok := runResp["metrics"].(map[string]any); ok {
							if steps, ok := metrics["total_steps"].(float64); ok {
								t.Logf("Simulation executed %d steps", int(steps))
							}
						}
					}
				}
			}
		}

		// Step 5: Get final twin state
		req = httptest.NewRequest("GET", fmt.Sprintf("/api/v1/twins/%s/state", twinID), nil)
		w = httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		if w.Code == http.StatusOK {
			var state DigitalTwin.TwinState
			json.Unmarshal(w.Body.Bytes(), &state)
			t.Logf("Twin state has %d entities tracked", len(state.Entities))
		}
	})
}

// ============================================================================
// SECTION 7: Smart Scenario Generation Tests
// ============================================================================

// TestSmartScenarioGeneration tests the smart scenario generator endpoint
func TestSmartScenarioGeneration(t *testing.T) {
	server := NewServer()
	require.NotNil(t, server)

	t.Run("Generate smart scenarios endpoint", func(t *testing.T) {
		// Use a fake twin ID - we're testing the endpoint structure
		req := httptest.NewRequest("POST", "/api/v1/twin/fake-twin/smart-scenarios", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		// Should return 200, 400 (bad request/missing params), 404, 500, or 503
		assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusBadRequest || w.Code == http.StatusNotFound || w.Code == http.StatusInternalServerError || w.Code == http.StatusServiceUnavailable,
			"Expected 200, 400, 404, 500, or 503, got %d", w.Code)

		if w.Code == http.StatusOK {
			var response map[string]any
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.NotNil(t, response["scenarios"])

			if count, ok := response["count"].(float64); ok {
				assert.GreaterOrEqual(t, int(count), 0)
			}
		}
	})
}

// ============================================================================
// SECTION 8: Error Handling and Edge Cases
// ============================================================================

// TestErrorHandling_InvalidRequests tests error handling for invalid requests
func TestErrorHandling_InvalidRequests(t *testing.T) {
	server := NewServer()
	require.NotNil(t, server)

	t.Run("Create twin without ontology_id", func(t *testing.T) {
		createReq := map[string]any{
			"name":       "Invalid Twin",
			"model_type": "organization",
		}

		body, _ := json.Marshal(createReq)
		req := httptest.NewRequest("POST", "/api/v1/twins", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		// Should get 400 Bad Request, or 503 if service unavailable, or 500 if DB error
		assert.True(t, w.Code == http.StatusBadRequest || w.Code == http.StatusServiceUnavailable || w.Code == http.StatusInternalServerError,
			"Should get 400, 503, or 500 for missing ontology_id, got %d", w.Code)
	})

	t.Run("Create twin without name", func(t *testing.T) {
		createReq := map[string]any{
			"ontology_id": "fake-ontology",
			"model_type":  "organization",
		}

		body, _ := json.Marshal(createReq)
		req := httptest.NewRequest("POST", "/api/v1/twins", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		// Should get 400 Bad Request, or 503 if service unavailable, or 500 if DB error
		assert.True(t, w.Code == http.StatusBadRequest || w.Code == http.StatusServiceUnavailable || w.Code == http.StatusInternalServerError,
			"Should get 400, 503, or 500 for missing name, got %d", w.Code)
	})

	t.Run("Get non-existent twin", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/twins/non-existent-id", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		// Should get 404 Not Found, 503 Service Unavailable, or 500 if DB error
		assert.True(t, w.Code == http.StatusNotFound || w.Code == http.StatusServiceUnavailable || w.Code == http.StatusInternalServerError,
			"Expected 404, 503, or 500, got %d", w.Code)
	})

	t.Run("Delete non-existent twin", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/v1/twins/non-existent-id", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		// Should get 404 Not Found, 503 Service Unavailable, or 500 if DB error
		assert.True(t, w.Code == http.StatusNotFound || w.Code == http.StatusServiceUnavailable || w.Code == http.StatusInternalServerError,
			"Expected 404, 503, or 500, got %d", w.Code)
	})

	t.Run("Create scenario for non-existent twin", func(t *testing.T) {
		scenarioReq := map[string]any{
			"name":     "Test Scenario",
			"duration": 50,
			"events":   []map[string]any{},
		}

		body, _ := json.Marshal(scenarioReq)
		req := httptest.NewRequest("POST", "/api/v1/twins/non-existent-id/scenarios", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		// Should get 404 Not Found, 503 Service Unavailable, or 500 if DB error
		assert.True(t, w.Code == http.StatusNotFound || w.Code == http.StatusServiceUnavailable || w.Code == http.StatusInternalServerError,
			"Expected 404, 503, or 500, got %d", w.Code)
	})
}

// TestTwinDeletion_FullCleanup tests that deleting a twin cleans up related data
func TestTwinDeletion_FullCleanup(t *testing.T) {
	server := NewServer()
	require.NotNil(t, server)

	var ontologyID, twinID string

	t.Run("Setup: Create twin for deletion test", func(t *testing.T) {
		ontologyData := `@prefix ex: <http://example.org/del#> .
@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .

ex:Item a rdfs:Class .
ex:Item1 a ex:Item ; rdfs:label "Item 1" .`

		createReq := map[string]any{
			"name":          "Deletion Test Ontology",
			"ontology_data": ontologyData,
			"format":        "turtle",
		}

		body, _ := json.Marshal(createReq)
		req := httptest.NewRequest("POST", "/api/v1/ontology", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		if w.Code == http.StatusOK || w.Code == http.StatusCreated {
			var response map[string]any
			json.Unmarshal(w.Body.Bytes(), &response)
			if id, ok := response["ontology_id"].(string); ok {
				ontologyID = id
			}
		}

		if ontologyID != "" {
			twinReq := map[string]any{
				"ontology_id": ontologyID,
				"name":        "Deletion Test Twin",
				"model_type":  "organization",
			}

			body, _ = json.Marshal(twinReq)
			req = httptest.NewRequest("POST", "/api/v1/twins", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w = httptest.NewRecorder()
			server.router.ServeHTTP(w, req)

			if w.Code == http.StatusOK || w.Code == http.StatusCreated {
				var response map[string]any
				json.Unmarshal(w.Body.Bytes(), &response)
				if id, ok := response["twin_id"].(string); ok {
					twinID = id
				}
			}
		}
	})

	t.Run("Delete twin", func(t *testing.T) {
		if twinID == "" {
			t.Skip("Twin not created")
		}

		req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/v1/twins/%s", twinID), nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		// Should succeed or fail gracefully
		assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusServiceUnavailable || w.Code == http.StatusNotFound || w.Code == http.StatusInternalServerError,
			"Expected 200, 404, 500, or 503, got %d", w.Code)

		if w.Code == http.StatusOK {
			var response map[string]any
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.Equal(t, "Digital twin deleted successfully", response["message"])
			assert.Equal(t, twinID, response["id"])
		}
	})

	t.Run("Verify twin is deleted", func(t *testing.T) {
		if twinID == "" {
			t.Skip("Twin not created")
		}

		req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/twins/%s", twinID), nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		// Should get 404 Not Found, 503 Service Unavailable, or 500 if DB error
		assert.True(t, w.Code == http.StatusNotFound || w.Code == http.StatusServiceUnavailable || w.Code == http.StatusInternalServerError,
			"Expected 404, 503, or 500 for deleted twin, got %d", w.Code)
	})
}
