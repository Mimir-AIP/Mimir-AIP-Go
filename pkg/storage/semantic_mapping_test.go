package storage

import (
	"testing"
	"time"

	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	ontologysvc "github.com/mimir-aip/mimir-aip-go/pkg/ontology"
)

const semanticMappingOntology = `@prefix : <http://example.org/mimir#> .
@prefix owl: <http://www.w3.org/2002/07/owl#> .
@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .
@prefix xsd: <http://www.w3.org/2001/XMLSchema#> .

:SensorReading a owl:Class ;
  rdfs:label "Sensor Reading" .

:temperature a owl:DatatypeProperty ;
  rdfs:domain :SensorReading ;
  rdfs:range xsd:decimal .

:observedAt a owl:DatatypeProperty ;
  rdfs:domain :SensorReading ;
  rdfs:range xsd:dateTime .
`

func TestMapCIRToOntologyMatchesClassPropertiesAndViolations(t *testing.T) {
	compiled, err := ontologysvc.CompileOntologyContent("ontology-1", "project-1", "Readings", "1.0", semanticMappingOntology)
	if err != nil {
		t.Fatalf("compile ontology: %v", err)
	}
	cir := models.NewCIR(models.SourceTypeAPI, "api://sensor-readings", models.DataFormatJSON, map[string]interface{}{
		"entity_type": "sensor_reading",
		"Temperature": 82.5,
		"observed_at": "not-a-date",
		"raw_vendor":  "acme",
	})
	cir.Source.Timestamp = time.Now().UTC()

	mapping, err := mapCIRToOntology(cir, compiled)
	if err != nil {
		t.Fatalf("mapCIRToOntology failed: %v", err)
	}
	if mapping.ClassID != "SensorReading" {
		t.Fatalf("expected SensorReading class, got %q", mapping.ClassID)
	}
	if mapping.Properties["temperature"].SourceField != "Temperature" {
		t.Fatalf("expected temperature property mapping, got %+v", mapping.Properties)
	}
	if mapping.Properties["observedAt"].SourceField != "observed_at" {
		t.Fatalf("expected observedAt property mapping, got %+v", mapping.Properties)
	}
	if len(mapping.UnmappedFields) != 1 || mapping.UnmappedFields[0] != "raw_vendor" {
		t.Fatalf("expected raw_vendor unmapped, got %+v", mapping.UnmappedFields)
	}
	if len(mapping.Violations) != 1 || mapping.Violations[0].Code != "range_mismatch" {
		t.Fatalf("expected range_mismatch violation, got %+v", mapping.Violations)
	}
}
