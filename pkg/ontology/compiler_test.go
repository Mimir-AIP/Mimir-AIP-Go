package ontology

import (
	"testing"

	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

const compiledOntologyFixture = `@prefix : <http://example.org/mimir#> .
@prefix owl: <http://www.w3.org/2002/07/owl#> .
@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .
@prefix xsd: <http://www.w3.org/2001/XMLSchema#> .

:Asset a owl:Class ;
    rdfs:label "Asset" .

:Sensor a owl:Class ;
    rdfs:label "Sensor" ;
    rdfs:subClassOf :Asset .

:Reading a owl:Class .

:hasReading a owl:ObjectProperty ;
    rdfs:domain :Sensor ;
    rdfs:range :Reading ;
    rdfs:label "has reading" .

:temperature a owl:DatatypeProperty ;
    rdfs:domain :Reading ;
    rdfs:range xsd:decimal .
`

func TestCompileOntologyBuildsCanonicalGraph(t *testing.T) {
	compiled, err := CompileOntology(&models.Ontology{
		ID:        "ontology-1",
		ProjectID: "project-1",
		Name:      "Plant ontology",
		Version:   "1.0.0",
		Content:   compiledOntologyFixture,
	})
	if err != nil {
		t.Fatalf("expected compile success, got %v diagnostics=%+v", err, compiled.Diagnostics)
	}
	if compiled.OntologyID != "ontology-1" || compiled.ProjectID != "project-1" {
		t.Fatalf("compiled ontology lost identity: %+v", compiled)
	}
	if len(compiled.Classes) != 3 {
		t.Fatalf("expected 3 classes, got %d: %+v", len(compiled.Classes), compiled.Classes)
	}
	if len(compiled.Properties) != 2 {
		t.Fatalf("expected 2 properties, got %d: %+v", len(compiled.Properties), compiled.Properties)
	}
	if len(compiled.Relations) != 1 {
		t.Fatalf("expected 1 relation, got %d: %+v", len(compiled.Relations), compiled.Relations)
	}
	relation := compiled.Relations[0]
	if relation.Name != "hasReading" || relation.FromClass != "Sensor" || relation.ToClass != "Reading" {
		t.Fatalf("unexpected relation: %+v", relation)
	}
	if len(compiled.SearchTerms) == 0 {
		t.Fatalf("expected search terms")
	}
	assertHasSearchTerm(t, compiled.SearchTerms, "class", "Sensor")
	assertHasSearchTerm(t, compiled.SearchTerms, "property", "temperature")
	assertHasSearchTerm(t, compiled.SearchTerms, "relation", "hasReading:Sensor:Reading")
}

func TestCompileOntologyRejectsUndeclaredPrefix(t *testing.T) {
	compiled, err := CompileOntologyContent("ontology-1", "project-1", "Bad", "1.0", `@prefix : <http://example.org/mimir#> .

:Sensor a missing:Class .
`)
	if err == nil {
		t.Fatalf("expected compile error")
	}
	if !hasDiagnostic(compiled.Diagnostics, "invalid_object", models.OntologyDiagnosticError) {
		t.Fatalf("expected invalid_object diagnostic, got %+v", compiled.Diagnostics)
	}
}

func TestCompileOntologyAllowsWarningsWithoutFailing(t *testing.T) {
	compiled, err := CompileOntologyContent("ontology-1", "project-1", "Sparse", "1.0", `@prefix : <http://example.org/mimir#> .
@prefix owl: <http://www.w3.org/2002/07/owl#> .

:hasThing a owl:ObjectProperty .
`)
	if err != nil {
		t.Fatalf("expected warnings-only compile success, got %v diagnostics=%+v", err, compiled.Diagnostics)
	}
	if !hasDiagnostic(compiled.Diagnostics, "no_classes", models.OntologyDiagnosticWarning) {
		t.Fatalf("expected no_classes warning, got %+v", compiled.Diagnostics)
	}
	if !hasDiagnostic(compiled.Diagnostics, "incomplete_relation", models.OntologyDiagnosticWarning) {
		t.Fatalf("expected incomplete_relation warning, got %+v", compiled.Diagnostics)
	}
}

func assertHasSearchTerm(t *testing.T, terms []models.OntologySearchTerm, kind, id string) {
	t.Helper()
	for _, term := range terms {
		if term.Kind == kind && term.ID == id {
			return
		}
	}
	t.Fatalf("missing %s search term %q in %+v", kind, id, terms)
}

func hasDiagnostic(diagnostics []models.OntologyDiagnostic, code string, severity models.OntologyDiagnosticSeverity) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code && diagnostic.Severity == severity {
			return true
		}
	}
	return false
}
