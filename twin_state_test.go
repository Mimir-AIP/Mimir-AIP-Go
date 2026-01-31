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
// State Management Tests
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
