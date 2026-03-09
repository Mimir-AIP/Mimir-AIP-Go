package digitaltwin

import (
	"fmt"
	"testing"
	"time"

	"github.com/mimir-aip/mimir-aip-go/pkg/metadatastore"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"github.com/mimir-aip/mimir-aip-go/pkg/ontology"
)

func benchEntities(count int) []*models.Entity {
	entities := make([]*models.Entity, 0, count)
	now := time.Now().UTC()
	for i := 0; i < count; i++ {
		id := fmt.Sprintf("machine-%d", i)
		entities = append(entities, &models.Entity{
			ID:            id,
			DigitalTwinID: "bench-twin",
			Type:          "Machine",
			Attributes: map[string]interface{}{
				"name":        fmt.Sprintf("Machine-%d", i),
				"temperature": float64(i % 120),
				"line":        fmt.Sprintf("L-%d", i%8),
			},
			CreatedAt: now,
			UpdatedAt: now,
		})
	}
	return entities
}

func BenchmarkEvaluateSPARQL_FilteredTypeQuery(b *testing.B) {
	query := &SPARQLQuery{
		Variables: []string{"machine", "name", "temperature"},
		Patterns: []TriplePattern{
			{Subject: "machine", Predicate: "a", Object: "Machine", IsVar: [3]bool{true, false, false}},
			{Subject: "machine", Predicate: "name", Object: "name", IsVar: [3]bool{true, false, true}},
			{Subject: "machine", Predicate: "temperature", Object: "temperature", IsVar: [3]bool{true, false, true}},
		},
		Filters: []FilterExpr{{Variable: "temperature", Operator: "gt", Value: 80.0}},
		Limit:   500,
	}
	entities := benchEntities(20000)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rows := evaluateSPARQL(query, entities)
		if len(rows) == 0 {
			b.Fatal("expected SPARQL rows")
		}
	}
}

func BenchmarkSPARQLEngineExecute_EndToEnd(b *testing.B) {
	store, err := metadatastore.NewSQLiteStore(":memory:")
	if err != nil {
		b.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	twin := &models.DigitalTwin{
		ID:        "bench-twin",
		ProjectID: "project-bench",
		Name:      "Bench Twin",
		Status:    "active",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	if err := store.SaveDigitalTwin(twin); err != nil {
		b.Fatalf("failed to save twin: %v", err)
	}

	for _, entity := range benchEntities(12000) {
		if err := store.SaveEntity(entity); err != nil {
			b.Fatalf("failed to save entity: %v", err)
		}
	}

	engine := NewSPARQLEngine(store, ontology.NewService(store))
	req := &models.QueryRequest{Query: "SELECT ?machine ?name ?temperature WHERE { ?machine a :Machine . ?machine :name ?name . ?machine :temperature ?temperature . FILTER(?temperature > 80) } LIMIT 200"}

	initial, err := engine.Execute(twin, req)
	if err != nil {
		b.Fatalf("initial execute failed: %v", err)
	}
	if initial.Metadata["query_type"] != "sparql" {
		b.Fatalf("expected parsed SPARQL query execution, got metadata: %+v", initial.Metadata)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := engine.Execute(twin, req)
		if err != nil {
			b.Fatalf("Execute failed: %v", err)
		}
		if result.Count == 0 {
			b.Fatal("expected non-zero query results")
		}
	}
}
