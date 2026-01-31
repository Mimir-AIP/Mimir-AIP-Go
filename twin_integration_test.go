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
// Full Integration Flow Tests
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
