package storage

import (
	"testing"

	"github.com/mimir-aip/mimir-aip-go/pkg/metadatastore"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	ontologysvc "github.com/mimir-aip/mimir-aip-go/pkg/ontology"
)

func TestAggregateByOntologyGroupsSemanticValues(t *testing.T) {
	store, err := metadatastore.NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()
	saveTestProject(t, store, "project-1")
	ontologyRecord, err := ontologysvc.NewService(store).CreateOntology(&models.OntologyCreateRequest{ProjectID: "project-1", Name: "Readings", Content: semanticMappingOntology + `
:location a owl:DatatypeProperty ;
  rdfs:domain :SensorReading ;
  rdfs:range xsd:string .`, Status: "active"})
	if err != nil {
		t.Fatalf("failed to create ontology: %v", err)
	}
	compiled, err := store.GetCompiledOntology(ontologyRecord.ID)
	if err != nil {
		t.Fatalf("failed to get compiled ontology: %v", err)
	}
	sample := []*models.CIR{
		semanticAggregateCIR(ontologyRecord.ID, compiled.ContentHash, 10.0, "A"),
		semanticAggregateCIR(ontologyRecord.ID, compiled.ContentHash, 20.0, "A"),
		semanticAggregateCIR(ontologyRecord.ID, compiled.ContentHash, 40.0, "B"),
	}
	svc := NewService(store)
	svc.RegisterPlugin("aggregate-sample", &retrieveSamplePlugin{sample: sample})
	if _, err := svc.CreateStorageConfigWithOntology("project-1", "aggregate-sample", map[string]interface{}{"connection_string": "mock://aggregate"}, ontologyRecord.ID); err != nil {
		t.Fatalf("failed to create storage config: %v", err)
	}
	resp, err := svc.AggregateByOntology("project-1", ontologyRecord.ID, &models.OntologyAggregateRequest{ProjectID: "project-1", ClassID: "SensorReading", Metric: models.OntologyAggregateMetric{Property: "temperature", Function: "avg"}, GroupBy: []string{"location"}})
	if err != nil {
		t.Fatalf("AggregateByOntology failed: %v", err)
	}
	if resp.Count != 2 {
		t.Fatalf("expected 2 groups, got %+v", resp)
	}
	values := map[string]float64{}
	for _, group := range resp.Groups {
		values[group.Key["location"].(string)] = group.Value
	}
	if values["A"] != 15.0 || values["B"] != 40.0 {
		t.Fatalf("unexpected group averages: %+v", values)
	}
}

func semanticAggregateCIR(ontologyID, hash string, temperature float64, location string) *models.CIR {
	cir := models.NewCIR(models.SourceTypeAPI, "api://readings", models.DataFormatJSON, map[string]interface{}{"temperature": temperature, "location": location})
	cir.Metadata.Ontology = &models.CIRSemanticMapping{OntologyID: ontologyID, ContentHash: hash, ClassID: "SensorReading", Properties: map[string]models.SemanticProperty{
		"temperature": {PropertyID: "temperature", SourceField: "temperature", Value: temperature, Range: []string{"decimal"}, Kind: "datatype"},
		"location":    {PropertyID: "location", SourceField: "location", Value: location, Range: []string{"string"}, Kind: "datatype"},
	}}
	return cir
}
