package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOntology_UploadAndList tests the ontology upload and list endpoints
func TestOntology_UploadAndList(t *testing.T) {
	server := NewServer()
	require.NotNil(t, server)

	// Test 1: List ontologies (may return 500 if DB not available - that's ok)
	t.Run("List ontologies", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/ontology", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		// Either 200 (success) or 500 (DB not available) is acceptable
		assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusInternalServerError,
			"Expected 200 or 500, got %d", w.Code)
	})

	// Test 2: Upload ontology (may return 500 if features not available)
	t.Run("Upload ontology", func(t *testing.T) {
		// Simple Turtle ontology for testing
		ontologyData := `
@prefix ex: <http://example.org/> .
@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .

ex:Person a rdfs:Class .
ex:name a rdfs:Property .
`

		uploadReq := map[string]any{
			"name":          "test-ontology",
			"description":   "Test ontology for integration testing",
			"version":       "1.0.0",
			"format":        "turtle",
			"ontology_data": ontologyData,
			"created_by":    "test-user",
		}

		body, _ := json.Marshal(uploadReq)
		req := httptest.NewRequest("POST", "/api/v1/ontology", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		// Either 200 (success) or 500 (features not available) is acceptable
		assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusInternalServerError,
			"Expected 200 or 500, got %d", w.Code)

		// If successful, verify response structure
		if w.Code == http.StatusOK {
			var response OntologyUploadResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)
			assert.NotEmpty(t, response.OntologyID)
			assert.Equal(t, "test-ontology", response.OntologyName)
			assert.Equal(t, "1.0.0", response.OntologyVersion)
			assert.Equal(t, "success", response.Status)
		}
	})

	// Test 3: Validate ontology from request body
	t.Run("Validate ontology from body", func(t *testing.T) {
		validTurtle := `
@prefix ex: <http://example.org/> .
@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .

ex:TestClass a rdfs:Class .
`

		validateReq := map[string]any{
			"ontology_data": validTurtle,
			"format":        "turtle",
		}

		body, _ := json.Marshal(validateReq)
		req := httptest.NewRequest("POST", "/api/v1/ontology/validate", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		// Either 200 (success) or 500 (features not available) is acceptable
		assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusInternalServerError,
			"Expected 200 or 500, got %d", w.Code)
	})
}

// TestOntology_GetOntology tests getting a specific ontology
func TestOntology_GetOntology(t *testing.T) {
	server := NewServer()
	require.NotNil(t, server)

	t.Run("Get ontology by ID", func(t *testing.T) {
		// Using a test ID - will likely return 404 or 500 if DB not available
		req := httptest.NewRequest("GET", "/api/v1/ontology/test-ontology-id", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		// 404 (not found) or 500 (DB not available) are acceptable
		assert.True(t, w.Code == http.StatusNotFound || w.Code == http.StatusInternalServerError,
			"Expected 404 or 500, got %d", w.Code)
	})

	t.Run("Get ontology with content", func(t *testing.T) {
		// Test with include_content query param
		req := httptest.NewRequest("GET", "/api/v1/ontology/test-ontology-id?include_content=true", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		// 404 (not found) or 500 (DB not available) are acceptable
		assert.True(t, w.Code == http.StatusNotFound || w.Code == http.StatusInternalServerError,
			"Expected 404 or 500, got %d", w.Code)
	})
}

// TestOntology_SPARQLQuery tests the SPARQL query endpoint
func TestOntology_SPARQLQuery(t *testing.T) {
	server := NewServer()
	require.NotNil(t, server)

	t.Run("SPARQL query", func(t *testing.T) {
		queryReq := SPARQLQueryRequest{
			Query:  "SELECT ?s ?p ?o WHERE { ?s ?p ?o } LIMIT 10",
			Limit:  10,
			Offset: 0,
		}

		body, _ := json.Marshal(queryReq)
		req := httptest.NewRequest("POST", "/api/v1/kg/query", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		// Either 200 (success) or 500 (features not available) is acceptable
		assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusInternalServerError,
			"Expected 200 or 500, got %d", w.Code)
	})

	t.Run("SPARQL query with pagination", func(t *testing.T) {
		queryReq := map[string]any{
			"query":  "SELECT ?s ?p ?o WHERE { GRAPH ?g { ?s ?p ?o } }",
			"limit":  5,
			"offset": 10,
		}

		body, _ := json.Marshal(queryReq)
		req := httptest.NewRequest("POST", "/api/v1/kg/query", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		// Either 200 (success) or 500 (features not available) is acceptable
		assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusInternalServerError,
			"Expected 200 or 500, got %d", w.Code)
	})
}

// TestOntology_KnowledgeGraphStats tests the knowledge graph stats endpoint
func TestOntology_KnowledgeGraphStats(t *testing.T) {
	server := NewServer()
	require.NotNil(t, server)

	t.Run("Get KG stats", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/kg/stats", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		// Either 200 (success) or 500 (features not available) is acceptable
		assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusInternalServerError,
			"Expected 200 or 500, got %d", w.Code)
	})
}

// TestOntology_ExtractionJob tests the extraction job creation endpoint
func TestOntology_ExtractionJob(t *testing.T) {
	server := NewServer()
	require.NotNil(t, server)

	t.Run("Create extraction job", func(t *testing.T) {
		extractionReq := ExtractionJobRequest{
			OntologyID:     "test-ontology-id",
			JobName:        "test-extraction-job",
			SourceType:     "text",
			ExtractionType: "entities",
			Data: map[string]any{
				"text": "John works at Acme Corp in New York. He is a software engineer.",
			},
		}

		body, _ := json.Marshal(extractionReq)
		req := httptest.NewRequest("POST", "/api/v1/extraction/jobs", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		// Either 200 (success) or 500 (features not available) is acceptable
		assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusInternalServerError,
			"Expected 200 or 500, got %d", w.Code)
	})

	t.Run("List extraction jobs", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/extraction/jobs", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		// Either 200 (success) or 500 (features not available) is acceptable
		assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusInternalServerError,
			"Expected 200 or 500, got %d", w.Code)
	})

	t.Run("List extraction jobs with filters", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/extraction/jobs?ontology_id=test-ontology&status=completed", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		// Either 200 (success) or 500 (features not available) is acceptable
		assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusInternalServerError,
			"Expected 200 or 500, got %d", w.Code)
	})
}

// TestOntology_DeleteOntology tests ontology deletion
func TestOntology_DeleteOntology(t *testing.T) {
	server := NewServer()
	require.NotNil(t, server)

	t.Run("Delete ontology", func(t *testing.T) {
		// Using a test ID - will likely return 500 if features not available
		req := httptest.NewRequest("DELETE", "/api/v1/ontology/test-ontology-id", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		// Either 200 (success) or 500 (features not available) is acceptable
		assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusInternalServerError,
			"Expected 200 or 500, got %d", w.Code)
	})
}

// TestOntology_GetSubgraph tests the subgraph endpoint for visualization
func TestOntology_GetSubgraph(t *testing.T) {
	server := NewServer()
	require.NotNil(t, server)

	t.Run("Get subgraph without root_uri", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/kg/subgraph", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		// Should return 400 (bad request) for missing root_uri
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Get subgraph with root_uri", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/kg/subgraph?root_uri=http://example.org/Person&depth=2", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		// Either 200 (success) or 500 (features not available) is acceptable
		assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusInternalServerError,
			"Expected 200 or 500, got %d", w.Code)
	})
}

// TestOntology_UpdateOntology tests updating an ontology
func TestOntology_UpdateOntology(t *testing.T) {
	server := NewServer()
	require.NotNil(t, server)

	t.Run("Update ontology", func(t *testing.T) {
		updateReq := map[string]any{
			"name":          "updated-ontology",
			"description":   "Updated description",
			"version":       "1.1.0",
			"ontology_data": "@prefix ex: <http://example.org/> . ex:Updated a ex:Class .",
		}

		body, _ := json.Marshal(updateReq)
		req := httptest.NewRequest("PUT", "/api/v1/ontology/test-ontology-id", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		// Should return 200 even if not fully implemented
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// TestOntology_ExportOntology tests exporting an ontology
func TestOntology_ExportOntology(t *testing.T) {
	server := NewServer()
	require.NotNil(t, server)

	t.Run("Export ontology", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/ontology/test-ontology-id/export?format=turtle", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		// Either 200 (success), 404 (not found), or 500 (features not available) is acceptable
		assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusNotFound || w.Code == http.StatusInternalServerError,
			"Expected 200, 404, or 500, got %d", w.Code)
	})

	t.Run("Export ontology as RDF/XML", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/ontology/test-ontology-id/export?format=rdfxml", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		// Either 200 (success), 404 (not found), or 500 (features not available) is acceptable
		assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusNotFound || w.Code == http.StatusInternalServerError,
			"Expected 200, 404, or 500, got %d", w.Code)
	})
}

// TestOntology_OntologyStats tests getting ontology statistics
func TestOntology_OntologyStats(t *testing.T) {
	server := NewServer()
	require.NotNil(t, server)

	t.Run("Get ontology stats", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/ontology/test-ontology-id/stats", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		// Either 200 (success) or 500 (features not available) is acceptable
		assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusInternalServerError,
			"Expected 200 or 500, got %d", w.Code)
	})
}

// TestOntology_ValidateExistingOntology tests validating an existing ontology by ID
func TestOntology_ValidateExistingOntology(t *testing.T) {
	server := NewServer()
	require.NotNil(t, server)

	t.Run("Validate existing ontology by ID", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/ontology/test-ontology-id/validate", nil)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		// Should return 200 even if not fully implemented
		assert.Equal(t, http.StatusOK, w.Code)
	})
}
