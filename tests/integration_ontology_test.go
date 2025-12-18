package tests

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/KnowledgeGraph"
	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/Ontology"
	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/Storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOntologyUploadWorkflow tests the complete ontology upload workflow
func TestOntologyUploadWorkflow(t *testing.T) {
	// Skip if in CI without proper environment
	if os.Getenv("FUSEKI_URL") == "" {
		t.Skip("Skipping integration test: FUSEKI_URL not set")
	}

	// Create temporary database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_mimir.db")

	// Initialize persistence backend
	persistence, err := storage.NewPersistenceBackend(dbPath)
	require.NoError(t, err, "Failed to create persistence backend")
	defer persistence.Close()

	// Initialize TDB2 backend
	fusekiURL := os.Getenv("FUSEKI_URL")
	if fusekiURL == "" {
		fusekiURL = "http://localhost:3030"
	}
	tdb2Backend := knowledgegraph.NewTDB2Backend(fusekiURL, "test")

	// Check Fuseki health
	ctx := context.Background()
	err = tdb2Backend.Health(ctx)
	if err != nil {
		t.Skipf("Fuseki not available at %s: %v", fusekiURL, err)
	}

	// Create ontology directory
	ontologyDir := filepath.Join(tmpDir, "ontologies")
	err = os.MkdirAll(ontologyDir, 0755)
	require.NoError(t, err, "Failed to create ontology directory")

	// Create ontology management plugin
	plugin := ontology.NewManagementPlugin(persistence, tdb2Backend, ontologyDir)

	// Load test ontology
	ontologyData, err := os.ReadFile("../test_data/simple_ontology.ttl")
	require.NoError(t, err, "Failed to read test ontology")

	// Test 1: Upload ontology
	t.Run("UploadOntology", func(t *testing.T) {
		stepConfig := pipelines.StepConfig{
			Name:   "upload_test",
			Plugin: "Ontology.management",
			Config: map[string]any{
				"operation":     "upload",
				"name":          "test-ontology",
				"description":   "Test ontology for integration testing",
				"version":       "1.0.0",
				"format":        "turtle",
				"ontology_data": string(ontologyData),
				"created_by":    "test-user",
			},
		}

		globalContext := pipelines.NewPluginContext()
		result, err := plugin.ExecuteStep(ctx, stepConfig, globalContext)
		require.NoError(t, err, "Upload should succeed")

		ontologyID, ok := result.Get("ontology_id")
		require.True(t, ok, "Result should contain ontology_id")
		assert.NotEmpty(t, ontologyID, "Ontology ID should not be empty")

		// Store ontology ID for later tests
		globalContext.Set("test_ontology_id", ontologyID)
	})

	// Test 2: List ontologies
	t.Run("ListOntologies", func(t *testing.T) {
		ontologies, err := persistence.ListOntologies(ctx, "")
		require.NoError(t, err, "List should succeed")
		assert.Len(t, ontologies, 1, "Should have one ontology")
		assert.Equal(t, "test-ontology", ontologies[0].Name)
		assert.Equal(t, "1.0.0", ontologies[0].Version)
	})

	// Test 3: Get ontology
	t.Run("GetOntology", func(t *testing.T) {
		ontologies, err := persistence.ListOntologies(ctx, "")
		require.NoError(t, err)
		require.Len(t, ontologies, 1)

		ontology, err := persistence.GetOntology(ctx, ontologies[0].ID)
		require.NoError(t, err, "Get should succeed")
		assert.Equal(t, "test-ontology", ontology.Name)
		assert.Equal(t, "1.0.0", ontology.Version)
		assert.Equal(t, "active", ontology.Status)
	})

	// Test 4: Validate ontology
	t.Run("ValidateOntology", func(t *testing.T) {
		stepConfig := pipelines.StepConfig{
			Name:   "validate_test",
			Plugin: "Ontology.management",
			Config: map[string]any{
				"operation":     "validate",
				"ontology_data": string(ontologyData),
				"format":        "turtle",
			},
		}

		globalContext := pipelines.NewPluginContext()
		result, err := plugin.ExecuteStep(ctx, stepConfig, globalContext)
		require.NoError(t, err, "Validation should succeed")

		valid, ok := result.Get("valid")
		require.True(t, ok, "Result should contain valid field")
		assert.True(t, valid.(bool), "Ontology should be valid")
	})

	// Test 5: Query knowledge graph
	t.Run("QueryKnowledgeGraph", func(t *testing.T) {
		// Simple query to count triples
		query := "SELECT (COUNT(*) AS ?count) WHERE { ?s ?p ?o }"
		queryResult, err := tdb2Backend.QuerySPARQL(ctx, query)
		require.NoError(t, err, "Query should succeed")

		assert.NotEmpty(t, queryResult.Variables, "Should have variables")
		assert.NotEmpty(t, queryResult.Bindings, "Should have bindings")
	})

	// Test 6: Get statistics
	t.Run("GetStatistics", func(t *testing.T) {
		stats, err := tdb2Backend.Stats(ctx)
		require.NoError(t, err, "Stats should succeed")
		assert.Greater(t, stats.TotalTriples, 0, "Should have triples")
	})

	// Test 7: Delete ontology
	t.Run("DeleteOntology", func(t *testing.T) {
		ontologies, err := persistence.ListOntologies(ctx, "")
		require.NoError(t, err)
		require.Len(t, ontologies, 1)

		stepConfig := pipelines.StepConfig{
			Name:   "delete_test",
			Plugin: "Ontology.management",
			Config: map[string]any{
				"operation":   "delete",
				"ontology_id": ontologies[0].ID,
			},
		}

		globalContext := pipelines.NewPluginContext()
		result, err := plugin.ExecuteStep(ctx, stepConfig, globalContext)
		require.NoError(t, err, "Delete should succeed")

		status, ok := result.Get("status")
		require.True(t, ok, "Result should contain status")
		assert.Equal(t, "deleted", status)

		// Verify deletion
		ontologies, err = persistence.ListOntologies(ctx, "")
		require.NoError(t, err)
		assert.Len(t, ontologies, 0, "Should have no ontologies after deletion")
	})
}

// TestOntologyValidation tests ontology format validation
func TestOntologyValidation(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_mimir.db")

	persistence, err := storage.NewPersistenceBackend(dbPath)
	require.NoError(t, err)
	defer persistence.Close()

	tdb2Backend := knowledgegraph.NewTDB2Backend("http://localhost:3030", "test")
	ontologyDir := filepath.Join(tmpDir, "ontologies")
	os.MkdirAll(ontologyDir, 0755)

	plugin := ontology.NewManagementPlugin(persistence, tdb2Backend, ontologyDir)
	ctx := context.Background()

	tests := []struct {
		name          string
		ontologyData  string
		format        string
		expectedValid bool
	}{
		{
			name: "Valid Turtle",
			ontologyData: `@prefix : <http://example.org/> .
:Person a owl:Class .`,
			format:        "turtle",
			expectedValid: true,
		},
		{
			name:          "Empty ontology",
			ontologyData:  "",
			format:        "turtle",
			expectedValid: false,
		},
		{
			name: "Missing triple terminator",
			ontologyData: `@prefix : <http://example.org/>
:Person a owl:Class`,
			format:        "turtle",
			expectedValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stepConfig := pipelines.StepConfig{
				Name:   "validate_test",
				Plugin: "Ontology.management",
				Config: map[string]any{
					"operation":     "validate",
					"ontology_data": tt.ontologyData,
					"format":        tt.format,
				},
			}

			globalContext := pipelines.NewPluginContext()
			result, err := plugin.ExecuteStep(ctx, stepConfig, globalContext)
			require.NoError(t, err, "Validation should not error")

			valid, ok := result.Get("valid")
			require.True(t, ok, "Result should contain valid field")
			// PluginContext.Set wraps non-standard types in JSONData with {"value": actual}
			validMap, isMap := valid.(map[string]any)
			if isMap {
				assert.Equal(t, tt.expectedValid, validMap["value"].(bool), "Validation result mismatch")
			} else {
				assert.Equal(t, tt.expectedValid, valid.(bool), "Validation result mismatch")
			}
		})
	}
}

// TestTDB2BackendOperations tests TDB2 backend operations
func TestTDB2BackendOperations(t *testing.T) {
	if os.Getenv("FUSEKI_URL") == "" {
		t.Skip("Skipping integration test: FUSEKI_URL not set")
	}

	fusekiURL := os.Getenv("FUSEKI_URL")
	if fusekiURL == "" {
		fusekiURL = "http://localhost:3030"
	}
	backend := knowledgegraph.NewTDB2Backend(fusekiURL, "test")
	ctx := context.Background()

	// Check health
	err := backend.Health(ctx)
	if err != nil {
		t.Skipf("Fuseki not available: %v", err)
	}

	// Clear test graph before starting
	testGraph := "http://example.org/test-graph"
	backend.ClearGraph(ctx, testGraph)

	// Test triple insertion
	t.Run("InsertTriples", func(t *testing.T) {
		triples := []knowledgegraph.Triple{
			{
				Subject:   "http://example.org/person1",
				Predicate: "http://www.w3.org/1999/02/22-rdf-syntax-ns#type",
				Object:    "http://example.org/Person",
				Graph:     testGraph,
			},
			{
				Subject:   "http://example.org/person1",
				Predicate: "http://example.org/hasName",
				Object:    "Alice",
				Graph:     testGraph,
			},
		}

		err := backend.InsertTriples(ctx, triples)
		assert.NoError(t, err, "Insert should succeed")
	})

	// Test SPARQL query
	t.Run("QuerySPARQL", func(t *testing.T) {
		query := `SELECT ?s ?p ?o WHERE {
			GRAPH <http://example.org/test-graph> {
				?s ?p ?o
			}
		} LIMIT 10`

		result, err := backend.QuerySPARQL(ctx, query)
		require.NoError(t, err, "Query should succeed")
		assert.NotEmpty(t, result.Bindings, "Should have results")
		assert.Len(t, result.Variables, 3, "Should have 3 variables")
	})

	// Test statistics
	t.Run("GetStats", func(t *testing.T) {
		stats, err := backend.Stats(ctx)
		require.NoError(t, err, "Stats should succeed")
		assert.Greater(t, stats.TotalTriples, 0, "Should have triples")
	})

	// Clean up
	t.Run("Cleanup", func(t *testing.T) {
		err := backend.ClearGraph(ctx, testGraph)
		assert.NoError(t, err, "Clear should succeed")
	})
}
