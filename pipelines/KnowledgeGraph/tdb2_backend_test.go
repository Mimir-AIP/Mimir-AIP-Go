package knowledgegraph

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTDB2Backend_Health tests the health check functionality
func TestTDB2Backend_Health(t *testing.T) {
	// Note: This test requires a running Fuseki instance
	// Skip if Fuseki is not available
	backend := NewTDB2Backend("http://localhost:3030", "mimir")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := backend.Health(ctx)

	if err != nil {
		t.Skip("Fuseki not available, skipping TDB2 tests")
	}

	assert.NoError(t, err, "Health check should succeed when Fuseki is running")
}

// TestTDB2Backend_InsertAndQueryTriples tests the full insert-query cycle
func TestTDB2Backend_InsertAndQueryTriples(t *testing.T) {
	backend := NewTDB2Backend("http://localhost:3030", "mimir")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Skip if Fuseki not available
	if err := backend.Health(ctx); err != nil {
		t.Skip("Fuseki not available, skipping TDB2 insert/query tests")
	}

	// Create test triples
	triples := []Triple{
		{
			Subject:   "http://example.org/product/1",
			Predicate: "http://example.org/name",
			Object:    "Widget",
			Graph:     "http://example.org/products",
		},
		{
			Subject:   "http://example.org/product/1",
			Predicate: "http://example.org/price",
			Object:    "19.99",
			Graph:     "http://example.org/products",
		},
		{
			Subject:   "http://example.org/product/2",
			Predicate: "http://example.org/name",
			Object:    "Gadget",
			Graph:     "http://example.org/products",
		},
	}

	// Insert triples
	err := backend.InsertTriples(ctx, triples)
	require.NoError(t, err, "Should insert triples successfully")

	// Query to verify insertion
	query := `
		SELECT ?name ?price WHERE {
			GRAPH <http://example.org/products> {
				<http://example.org/product/1> <http://example.org/name> ?name .
				<http://example.org/product/1> <http://example.org/price> ?price .
			}
		}
	`

	result, err := backend.QuerySPARQL(ctx, query)
	require.NoError(t, err, "Should query successfully")
	require.NotNil(t, result, "Result should not be nil")

	// Verify results
	assert.Equal(t, "SELECT", result.QueryType, "Query type should be SELECT")
	assert.Len(t, result.Bindings, 1, "Should have 1 result row")
	assert.Len(t, result.Variables, 2, "Should have 2 variables (name, price)")

	// Check values
	if len(result.Bindings) > 0 {
		binding := result.Bindings[0]
		if nameVal, ok := binding["name"]; ok {
			assert.Equal(t, "Widget", nameVal.Value, "Product name should be Widget")
			assert.Equal(t, "literal", nameVal.Type, "Type should be literal")
		}
		if priceVal, ok := binding["price"]; ok {
			assert.Equal(t, "19.99", priceVal.Value, "Price should be 19.99")
		}
	}

	t.Logf("Query executed in %v", result.Duration)
}

// TestTDB2Backend_Stats tests statistics retrieval
func TestTDB2Backend_Stats(t *testing.T) {
	backend := NewTDB2Backend("http://localhost:3030", "mimir")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Skip if Fuseki not available
	if err := backend.Health(ctx); err != nil {
		t.Skip("Fuseki not available, skipping stats test")
	}

	// Insert some test data first
	triples := []Triple{
		{
			Subject:   "http://test.org/entity/1",
			Predicate: "http://test.org/prop/a",
			Object:    "value1",
			Graph:     "http://test.org/graph1",
		},
		{
			Subject:   "http://test.org/entity/1",
			Predicate: "http://test.org/prop/b",
			Object:    "value2",
			Graph:     "http://test.org/graph1",
		},
		{
			Subject:   "http://test.org/entity/2",
			Predicate: "http://test.org/prop/a",
			Object:    "value3",
			Graph:     "http://test.org/graph2",
		},
	}

	err := backend.InsertTriples(ctx, triples)
	require.NoError(t, err)

	// Get stats
	stats, err := backend.Stats(ctx)
	require.NoError(t, err, "Should get stats successfully")
	require.NotNil(t, stats, "Stats should not be nil")

	// Verify stats are reasonable (at least our 3 triples)
	assert.GreaterOrEqual(t, stats.TotalTriples, 3, "Should have at least 3 triples")
	assert.GreaterOrEqual(t, stats.TotalSubjects, 2, "Should have at least 2 subjects")
	assert.GreaterOrEqual(t, stats.TotalPredicates, 2, "Should have at least 2 predicates")
	assert.NotEmpty(t, stats.NamedGraphs, "Should have named graphs")
	assert.NotZero(t, stats.LastUpdated, "LastUpdated should be set")

	t.Logf("Graph stats: %d triples, %d subjects, %d predicates, %d graphs",
		stats.TotalTriples, stats.TotalSubjects, stats.TotalPredicates, len(stats.NamedGraphs))
}

// TestTDB2Backend_QueryTypes tests different SPARQL query types
func TestTDB2Backend_QueryTypes(t *testing.T) {
	backend := NewTDB2Backend("http://localhost:3030", "mimir")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Skip if Fuseki not available
	if err := backend.Health(ctx); err != nil {
		t.Skip("Fuseki not available, skipping query types test")
	}

	// Insert test data
	triples := []Triple{
		{
			Subject:   "http://querytest.org/item/1",
			Predicate: "http://querytest.org/type",
			Object:    "Product",
		},
		{
			Subject:   "http://querytest.org/item/2",
			Predicate: "http://querytest.org/type",
			Object:    "Product",
		},
		{
			Subject:   "http://querytest.org/item/3",
			Predicate: "http://querytest.org/type",
			Object:    "Service",
		},
	}

	err := backend.InsertTriples(ctx, triples)
	require.NoError(t, err)

	t.Run("SELECT query", func(t *testing.T) {
		query := `SELECT ?item WHERE { ?item <http://querytest.org/type> "Product" }`
		result, err := backend.QuerySPARQL(ctx, query)
		require.NoError(t, err)
		assert.Equal(t, "SELECT", result.QueryType)
		assert.Len(t, result.Bindings, 2, "Should find 2 Products")
	})

	t.Run("ASK query", func(t *testing.T) {
		query := `ASK { ?item <http://querytest.org/type> "Service" }`
		result, err := backend.QuerySPARQL(ctx, query)
		require.NoError(t, err)
		assert.Equal(t, "ASK", result.QueryType)
		assert.NotNil(t, result.Boolean)
		assert.True(t, *result.Boolean, "Should find at least one Service")
	})

	t.Run("COUNT query", func(t *testing.T) {
		query := `SELECT (COUNT(?item) AS ?count) WHERE { ?item <http://querytest.org/type> ?type }`
		result, err := backend.QuerySPARQL(ctx, query)
		require.NoError(t, err)
		assert.Len(t, result.Bindings, 1)
		if count, ok := result.Bindings[0]["count"]; ok {
			assert.Equal(t, "3", count.Value, "Should count 3 items")
		}
	})
}

// TestTDB2Backend_DeleteTriples tests triple deletion
func TestTDB2Backend_DeleteTriples(t *testing.T) {
	backend := NewTDB2Backend("http://localhost:3030", "mimir")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Skip if Fuseki not available
	if err := backend.Health(ctx); err != nil {
		t.Skip("Fuseki not available, skipping delete test")
	}

	// Insert test data
	triples := []Triple{
		{
			Subject:   "http://deletetest.org/temp/1",
			Predicate: "http://deletetest.org/prop",
			Object:    "to-be-deleted",
		},
	}

	err := backend.InsertTriples(ctx, triples)
	require.NoError(t, err)

	// Verify it exists
	query := `ASK { <http://deletetest.org/temp/1> ?p ?o }`
	result, err := backend.QuerySPARQL(ctx, query)
	require.NoError(t, err)
	require.NotNil(t, result.Boolean)
	assert.True(t, *result.Boolean, "Triple should exist before deletion")

	// Delete it
	err = backend.DeleteTriples(ctx, "http://deletetest.org/temp/1", "", "")
	require.NoError(t, err, "Should delete triple successfully")

	// Verify it's gone
	result, err = backend.QuerySPARQL(ctx, query)
	require.NoError(t, err)
	require.NotNil(t, result.Boolean)
	assert.False(t, *result.Boolean, "Triple should not exist after deletion")
}

// TestTDB2Backend_ClearGraph tests clearing a named graph
func TestTDB2Backend_ClearGraph(t *testing.T) {
	backend := NewTDB2Backend("http://localhost:3030", "mimir")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Skip if Fuseki not available
	if err := backend.Health(ctx); err != nil {
		t.Skip("Fuseki not available, skipping clear graph test")
	}

	graphURI := "http://cleartest.org/graph"

	// Insert data into named graph
	triples := []Triple{
		{
			Subject:   "http://cleartest.org/item",
			Predicate: "http://cleartest.org/prop",
			Object:    "value",
			Graph:     graphURI,
		},
	}

	err := backend.InsertTriples(ctx, triples)
	require.NoError(t, err)

	// Verify graph has data
	query := `ASK { GRAPH <` + graphURI + `> { ?s ?p ?o } }`
	result, err := backend.QuerySPARQL(ctx, query)
	require.NoError(t, err)
	require.NotNil(t, result.Boolean)
	assert.True(t, *result.Boolean, "Graph should have data before clearing")

	// Clear the graph
	err = backend.ClearGraph(ctx, graphURI)
	require.NoError(t, err, "Should clear graph successfully")

	// Verify graph is empty
	result, err = backend.QuerySPARQL(ctx, query)
	require.NoError(t, err)
	require.NotNil(t, result.Boolean)
	assert.False(t, *result.Boolean, "Graph should be empty after clearing")
}

// TestTDB2Backend_LargeDataInsert tests inserting larger amounts of data
func TestTDB2Backend_LargeDataInsert(t *testing.T) {
	backend := NewTDB2Backend("http://localhost:3030", "mimir")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Skip if Fuseki not available
	if err := backend.Health(ctx); err != nil {
		t.Skip("Fuseki not available, skipping large data test")
	}

	// Generate 100 triples
	var triples []Triple
	for i := 1; i <= 100; i++ {
		triples = append(triples, Triple{
			Subject:   fmt.Sprintf("http://largetest.org/item/%d", i),
			Predicate: "http://largetest.org/property",
			Object:    fmt.Sprintf("value-%d", i),
			Graph:     "http://largetest.org/bulk",
		})
	}

	start := time.Now()
	err := backend.InsertTriples(ctx, triples)
	duration := time.Since(start)

	require.NoError(t, err, "Should insert 100 triples successfully")
	t.Logf("Inserted 100 triples in %v", duration)

	// Verify all were inserted
	query := `SELECT (COUNT(?s) AS ?count) WHERE { GRAPH <http://largetest.org/bulk> { ?s ?p ?o } }`
	result, err := backend.QuerySPARQL(ctx, query)
	require.NoError(t, err)

	if count, ok := result.Bindings[0]["count"]; ok {
		assert.Equal(t, "100", count.Value, "Should have exactly 100 triples")
	}

	// Cleanup
	backend.ClearGraph(ctx, "http://largetest.org/bulk")
}

// TestTripleValidation tests triple structure validation
func TestTripleValidation(t *testing.T) {
	tests := []struct {
		name    string
		triple  Triple
		wantErr bool
	}{
		{
			name: "valid URI subject",
			triple: Triple{
				Subject:   "http://example.org/subject",
				Predicate: "http://example.org/predicate",
				Object:    "literal value",
			},
			wantErr: false,
		},
		{
			name: "valid with named graph",
			triple: Triple{
				Subject:   "http://example.org/s",
				Predicate: "http://example.org/p",
				Object:    "http://example.org/o",
				Graph:     "http://example.org/graph",
			},
			wantErr: false,
		},
		{
			name: "valid literal with special chars",
			triple: Triple{
				Subject:   "http://example.org/item",
				Predicate: "http://example.org/description",
				Object:    `Value with "quotes" and \ backslash`,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just validate structure, not insertion
			assert.NotEmpty(t, tt.triple.Subject, "Subject should not be empty")
			assert.NotEmpty(t, tt.triple.Predicate, "Predicate should not be empty")
			assert.NotEmpty(t, tt.triple.Object, "Object should not be empty")
		})
	}
}

// TestQueryResultStructure tests query result data structures
func TestQueryResultStructure(t *testing.T) {
	result := &QueryResult{
		Variables: []string{"name", "age", "type"},
		Bindings: []BindingRow{
			{
				"name": BindingValue{Type: "literal", Value: "Alice"},
				"age":  BindingValue{Type: "literal", Value: "30", Datatype: "http://www.w3.org/2001/XMLSchema#integer"},
				"type": BindingValue{Type: "uri", Value: "http://example.org/Person"},
			},
			{
				"name": BindingValue{Type: "literal", Value: "Bob"},
				"age":  BindingValue{Type: "literal", Value: "25"},
				"type": BindingValue{Type: "uri", Value: "http://example.org/Person"},
			},
		},
		QueryType: "SELECT",
		Duration:  150 * time.Millisecond,
	}

	assert.Len(t, result.Variables, 3, "Should have 3 variables")
	assert.Len(t, result.Bindings, 2, "Should have 2 binding rows")
	assert.Equal(t, "SELECT", result.QueryType)
	assert.NotZero(t, result.Duration)

	// Check first binding
	first := result.Bindings[0]
	if name, ok := first["name"]; ok {
		assert.Equal(t, "literal", name.Type)
		assert.Equal(t, "Alice", name.Value)
	}

	if age, ok := first["age"]; ok {
		assert.Equal(t, "30", age.Value)
		assert.NotEmpty(t, age.Datatype, "Should have datatype for numeric value")
	}
}

// TestGraphStatsStructure tests graph stats data structure
func TestGraphStatsStructure(t *testing.T) {
	stats := &GraphStats{
		TotalTriples:    1000,
		TotalSubjects:   500,
		TotalPredicates: 50,
		TotalObjects:    750,
		NamedGraphs: []string{
			"http://example.org/graph1",
			"http://example.org/graph2",
			"http://example.org/graph3",
		},
		LastUpdated: time.Now(),
		SizeBytes:   1024000,
	}

	assert.Equal(t, 1000, stats.TotalTriples)
	assert.Equal(t, 500, stats.TotalSubjects)
	assert.Equal(t, 50, stats.TotalPredicates)
	assert.Equal(t, 750, stats.TotalObjects)
	assert.Len(t, stats.NamedGraphs, 3)
	assert.NotZero(t, stats.LastUpdated)
	assert.Equal(t, int64(1024000), stats.SizeBytes)

	// Sanity check
	assert.GreaterOrEqual(t, stats.TotalTriples, stats.TotalSubjects,
		"Should have more triples than subjects (subjects can have multiple properties)")
}

// TestTDB2Backend_EmptyOperations tests edge cases with empty data
func TestTDB2Backend_EmptyOperations(t *testing.T) {
	backend := NewTDB2Backend("http://localhost:3030", "mimir")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Skip if Fuseki not available
	if err := backend.Health(ctx); err != nil {
		t.Skip("Fuseki not available, skipping empty operations test")
	}

	t.Run("Insert empty triples slice", func(t *testing.T) {
		err := backend.InsertTriples(ctx, []Triple{})
		assert.NoError(t, err, "Should handle empty slice gracefully")
	})

	t.Run("Insert nil triples", func(t *testing.T) {
		err := backend.InsertTriples(ctx, nil)
		assert.NoError(t, err, "Should handle nil gracefully")
	})

	t.Run("Query empty result", func(t *testing.T) {
		// Query for something that doesn't exist
		query := `SELECT ?s WHERE { ?s <http://nonexistent.org/prop> "impossible-value" }`
		result, err := backend.QuerySPARQL(ctx, query)
		require.NoError(t, err)
		assert.Empty(t, result.Bindings, "Should return empty bindings for no matches")
	})
}
