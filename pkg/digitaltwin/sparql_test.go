package digitaltwin

import (
	"strings"
	"testing"

	"github.com/mimir-aip/mimir-aip-go/pkg/metadatastore"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"github.com/mimir-aip/mimir-aip-go/pkg/ontology"
)

func TestSPARQLEngineRejectsMalformedQuery(t *testing.T) {
	store, err := metadatastore.NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	engine := NewSPARQLEngine(store, ontology.NewService(store))
	_, err = engine.Execute(&models.DigitalTwin{ID: "twin-1"}, &models.QueryRequest{Query: "SELECT ?entity WHERE ?entity :name ?name"})
	if err == nil {
		t.Fatalf("expected malformed query to fail")
	}
	if !strings.Contains(err.Error(), "invalid supported SPARQL subset query") {
		t.Fatalf("expected explicit parse error, got %v", err)
	}
}
