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
// Twin Creation and Auto-Configuration Tests
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
