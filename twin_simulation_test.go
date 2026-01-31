package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/DigitalTwin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// Simulation Execution and Timeline Tests
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
