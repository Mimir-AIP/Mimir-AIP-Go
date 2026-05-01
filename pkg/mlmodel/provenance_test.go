package mlmodel

import (
	"testing"

	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

type mlProvenanceStoragePlugin struct{ sample []*models.CIR }

func (m *mlProvenanceStoragePlugin) Initialize(config *models.PluginConfig) error { return nil }
func (m *mlProvenanceStoragePlugin) CreateSchema(ontology *models.OntologyDefinition) error {
	return nil
}
func (m *mlProvenanceStoragePlugin) Store(cir *models.CIR) (*models.StorageResult, error) {
	return &models.StorageResult{Success: true, AffectedItems: 1}, nil
}
func (m *mlProvenanceStoragePlugin) Retrieve(query *models.CIRQuery) ([]*models.CIR, error) {
	return m.sample, nil
}
func (m *mlProvenanceStoragePlugin) Update(query *models.CIRQuery, updates *models.CIRUpdate) (*models.StorageResult, error) {
	return &models.StorageResult{Success: true, AffectedItems: 0}, nil
}
func (m *mlProvenanceStoragePlugin) Delete(query *models.CIRQuery) (*models.StorageResult, error) {
	return &models.StorageResult{Success: true, AffectedItems: 0}, nil
}
func (m *mlProvenanceStoragePlugin) GetMetadata() (*models.StorageMetadata, error) {
	return &models.StorageMetadata{StorageType: "ml-provenance"}, nil
}
func (m *mlProvenanceStoragePlugin) HealthCheck() (bool, error) { return true, nil }

func TestStartTrainingRecordsOntologyFeatureProvenance(t *testing.T) {
	svc, cleanup := setupTrainingService(t)
	defer cleanup()
	ontologyRecord, err := svc.ontologyService.CreateOntology(&models.OntologyCreateRequest{ProjectID: "project-1", Name: "ML Ontology", Content: `@prefix : <http://example.org/mimir#> .
@prefix owl: <http://www.w3.org/2002/07/owl#> .
@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .
@prefix xsd: <http://www.w3.org/2001/XMLSchema#> .

:Reading a owl:Class .
:temperature a owl:DatatypeProperty ;
  rdfs:domain :Reading ;
  rdfs:range xsd:decimal .`, Status: "active"})
	if err != nil {
		t.Fatalf("failed to create ontology: %v", err)
	}
	compiled, err := svc.store.GetCompiledOntology(ontologyRecord.ID)
	if err != nil {
		t.Fatalf("failed to load compiled ontology: %v", err)
	}
	cir := models.NewCIR(models.SourceTypeAPI, "api://ml", models.DataFormatJSON, map[string]interface{}{"temperature": 25.0})
	cir.Metadata.Ontology = &models.CIRSemanticMapping{OntologyID: ontologyRecord.ID, ContentHash: compiled.ContentHash, ClassID: "Reading", Properties: map[string]models.SemanticProperty{"temperature": {PropertyID: "temperature", SourceField: "temperature", Value: 25.0, Range: []string{"decimal"}, Kind: "datatype"}}}
	svc.storageService.RegisterPlugin("ml-provenance", &mlProvenanceStoragePlugin{sample: []*models.CIR{cir}})
	cfg, err := svc.storageService.CreateStorageConfigWithOntology("project-1", "ml-provenance", map[string]interface{}{"connection_string": "mock://ml"}, ontologyRecord.ID)
	if err != nil {
		t.Fatalf("failed to create storage config: %v", err)
	}
	model, err := svc.CreateModel(&models.ModelCreateRequest{ProjectID: "project-1", OntologyID: ontologyRecord.ID, Name: "model", Type: models.ModelTypeRegression})
	if err != nil {
		t.Fatalf("failed to create model: %v", err)
	}
	trained, err := svc.StartTraining(&models.ModelTrainingRequest{ModelID: model.ID, StorageIDs: []string{cfg.ID}})
	if err != nil {
		t.Fatalf("StartTraining failed: %v", err)
	}
	raw := trained.Metadata[modelMetadataFeatureProvenance]
	provenance, ok := raw.(*models.MLFeatureProvenance)
	if !ok {
		t.Fatalf("expected MLFeatureProvenance metadata, got %T %+v", raw, raw)
	}
	if provenance.ContentHash != compiled.ContentHash {
		t.Fatalf("expected content hash %s, got %s", compiled.ContentHash, provenance.ContentHash)
	}
	if len(provenance.Features) != 1 || provenance.Features[0].PropertyID != "temperature" {
		t.Fatalf("expected temperature feature provenance, got %+v", provenance.Features)
	}
}
