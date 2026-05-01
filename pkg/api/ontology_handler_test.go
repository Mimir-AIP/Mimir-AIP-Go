package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/mimir-aip/mimir-aip-go/pkg/metadatastore"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"github.com/mimir-aip/mimir-aip-go/pkg/ontology"
)

func setupOntologyHandlerTest(t *testing.T) (*OntologyHandler, metadatastore.MetadataStore, func()) {
	t.Helper()
	store, err := metadatastore.NewSQLiteStore(filepath.Join(t.TempDir(), "ontology-handler.db"))
	if err != nil {
		t.Fatalf("failed to create sqlite store: %v", err)
	}
	now := time.Now().UTC()
	for _, project := range []*models.Project{
		{ID: "project-a", Name: "project-a", Description: "A", Version: "v1", Status: models.ProjectStatusActive, Metadata: models.ProjectMetadata{CreatedAt: now, UpdatedAt: now}},
		{ID: "project-b", Name: "project-b", Description: "B", Version: "v1", Status: models.ProjectStatusActive, Metadata: models.ProjectMetadata{CreatedAt: now, UpdatedAt: now}},
	} {
		if err := store.SaveProject(project); err != nil {
			t.Fatalf("failed to save project: %v", err)
		}
	}
	service := ontology.NewService(store)
	return NewOntologyHandler(service), store, func() { _ = store.Close() }
}

func TestOntologyGetRequiresProjectOwnership(t *testing.T) {
	handler, store, cleanup := setupOntologyHandlerTest(t)
	defer cleanup()

	record := &models.Ontology{ID: "ontology-1", ProjectID: "project-a", Name: "Ontology", Content: "@prefix : <http://example.org/mimir#> .\n@prefix owl: <http://www.w3.org/2002/07/owl#> .\n\n:Entity a owl:Class .", Status: "draft", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	if err := store.SaveOntology(record); err != nil {
		t.Fatalf("failed to save ontology: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/ontologies/ontology-1?project_id=project-b", nil)
	resp := httptest.NewRecorder()
	handler.HandleOntology(resp, req)
	if resp.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden, got %d body=%s", resp.Code, resp.Body.String())
	}
}

func TestOntologyDeleteReturnsConflictWhenReferenced(t *testing.T) {
	handler, store, cleanup := setupOntologyHandlerTest(t)
	defer cleanup()

	record := &models.Ontology{ID: "ontology-1", ProjectID: "project-a", Name: "Ontology", Content: "@prefix : <http://example.org/mimir#> .\n@prefix owl: <http://www.w3.org/2002/07/owl#> .\n\n:Entity a owl:Class .", Status: "active", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	if err := store.SaveOntology(record); err != nil {
		t.Fatalf("failed to save ontology: %v", err)
	}
	if err := store.SaveMLModel(&models.MLModel{ID: "model-1", ProjectID: "project-a", OntologyID: record.ID, Name: "Model", Type: models.ModelTypeDecisionTree, Status: models.ModelStatusDraft, Version: "1.0.0", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}); err != nil {
		t.Fatalf("failed to save ml model: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/ontologies/ontology-1?project_id=project-a", nil)
	resp := httptest.NewRecorder()
	handler.HandleOntology(resp, req)
	if resp.Code != http.StatusConflict {
		t.Fatalf("expected 409 Conflict, got %d body=%s", resp.Code, resp.Body.String())
	}
}

func TestOntologyValidationEndpointReturnsDiagnostics(t *testing.T) {
	handler, _, cleanup := setupOntologyHandlerTest(t)
	defer cleanup()

	body := bytes.NewBufferString(`{"content":"@prefix : <http://example.org/mimir#> .\n\n:Sensor a missing:Class ."}`)
	req := httptest.NewRequest(http.MethodPost, "/api/ontologies/validate", body)
	resp := httptest.NewRecorder()
	handler.HandleOntologyValidation(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d body=%s", resp.Code, resp.Body.String())
	}
	var validation models.OntologyValidationResponse
	if err := json.NewDecoder(resp.Body).Decode(&validation); err != nil {
		t.Fatalf("failed to decode validation response: %v", err)
	}
	if validation.Valid {
		t.Fatalf("expected invalid response")
	}
	if len(validation.Diagnostics) == 0 {
		t.Fatalf("expected diagnostics")
	}
}

func TestOntologyCompiledEndpointRequiresProjectOwnership(t *testing.T) {
	handler, store, cleanup := setupOntologyHandlerTest(t)
	defer cleanup()

	created, err := ontology.NewService(store).CreateOntology(&models.OntologyCreateRequest{
		ProjectID: "project-a",
		Name:      "Compiled",
		Content:   "@prefix : <http://example.org/mimir#> .\n@prefix owl: <http://www.w3.org/2002/07/owl#> .\n\n:Entity a owl:Class .",
		Status:    "draft",
	})
	if err != nil {
		t.Fatalf("CreateOntology failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/ontologies/"+created.ID+"/compiled?project_id=project-b", nil)
	resp := httptest.NewRecorder()
	handler.HandleOntology(resp, req)
	if resp.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden, got %d body=%s", resp.Code, resp.Body.String())
	}
}

func TestOntologyCompiledEndpointReturnsCanonicalGraph(t *testing.T) {
	handler, store, cleanup := setupOntologyHandlerTest(t)
	defer cleanup()

	created, err := ontology.NewService(store).CreateOntology(&models.OntologyCreateRequest{
		ProjectID: "project-a",
		Name:      "Compiled",
		Content:   "@prefix : <http://example.org/mimir#> .\n@prefix owl: <http://www.w3.org/2002/07/owl#> .\n\n:Entity a owl:Class .",
		Status:    "draft",
	})
	if err != nil {
		t.Fatalf("CreateOntology failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/ontologies/"+created.ID+"/compiled?project_id=project-a", nil)
	resp := httptest.NewRecorder()
	handler.HandleOntology(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d body=%s", resp.Code, resp.Body.String())
	}
	var compiled models.CompiledOntology
	if err := json.NewDecoder(resp.Body).Decode(&compiled); err != nil {
		t.Fatalf("failed to decode compiled ontology: %v", err)
	}
	if len(compiled.Classes) != 1 || compiled.Classes[0].ID != "Entity" {
		t.Fatalf("unexpected compiled graph: %+v", compiled)
	}
}

func TestOntologySearchEndpointReturnsSemanticTerms(t *testing.T) {
	handler, store, cleanup := setupOntologyHandlerTest(t)
	defer cleanup()

	created, err := ontology.NewService(store).CreateOntology(&models.OntologyCreateRequest{
		ProjectID: "project-a",
		Name:      "Searchable",
		Content:   "@prefix : <http://example.org/mimir#> .\n@prefix owl: <http://www.w3.org/2002/07/owl#> .\n@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .\n\n:Sensor a owl:Class ; rdfs:label \"Sensor\" .",
		Status:    "draft",
	})
	if err != nil {
		t.Fatalf("CreateOntology failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/ontologies/"+created.ID+"/search?project_id=project-a&q=sensor", nil)
	resp := httptest.NewRecorder()
	handler.HandleOntology(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d body=%s", resp.Code, resp.Body.String())
	}
	var results []models.OntologySearchResult
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		t.Fatalf("failed to decode search results: %v", err)
	}
	if len(results) == 0 || results[0].Term.ID != "Sensor" {
		t.Fatalf("expected Sensor search result, got %+v", results)
	}
}
