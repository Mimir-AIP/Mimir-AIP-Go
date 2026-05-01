package storage

import (
	"testing"
	"time"

	"github.com/mimir-aip/mimir-aip-go/pkg/metadatastore"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	ontologysvc "github.com/mimir-aip/mimir-aip-go/pkg/ontology"
)

type retrieveSamplePlugin struct{ sample []*models.CIR }

func (m *retrieveSamplePlugin) Initialize(config *models.PluginConfig) error           { return nil }
func (m *retrieveSamplePlugin) CreateSchema(ontology *models.OntologyDefinition) error { return nil }
func (m *retrieveSamplePlugin) Store(cir *models.CIR) (*models.StorageResult, error) {
	return &models.StorageResult{Success: true, AffectedItems: 1}, nil
}
func (m *retrieveSamplePlugin) Retrieve(query *models.CIRQuery) ([]*models.CIR, error) {
	return m.sample, nil
}
func (m *retrieveSamplePlugin) Update(query *models.CIRQuery, updates *models.CIRUpdate) (*models.StorageResult, error) {
	return &models.StorageResult{Success: true, AffectedItems: 0}, nil
}
func (m *retrieveSamplePlugin) Delete(query *models.CIRQuery) (*models.StorageResult, error) {
	return &models.StorageResult{Success: true, AffectedItems: 0}, nil
}
func (m *retrieveSamplePlugin) GetMetadata() (*models.StorageMetadata, error) {
	return &models.StorageMetadata{StorageType: "retrieve-sample"}, nil
}
func (m *retrieveSamplePlugin) HealthCheck() (bool, error) { return true, nil }

func TestRetrieveByOntologyFiltersSemanticCIR(t *testing.T) {
	store, err := metadatastore.NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()
	saveTestProject(t, store, "project-1")
	ontologyRecord, err := ontologysvc.NewService(store).CreateOntology(&models.OntologyCreateRequest{ProjectID: "project-1", Name: "Readings", Content: semanticMappingOntology, Status: "active"})
	if err != nil {
		t.Fatalf("failed to create ontology: %v", err)
	}
	compiled, err := store.GetCompiledOntology(ontologyRecord.ID)
	if err != nil {
		t.Fatalf("failed to get compiled ontology: %v", err)
	}
	matching := models.NewCIR(models.SourceTypeAPI, "api://readings/1", models.DataFormatJSON, map[string]interface{}{"temperature": 82.5})
	matching.Source.Timestamp = time.Now().UTC()
	matching.Metadata.Ontology = &models.CIRSemanticMapping{OntologyID: ontologyRecord.ID, ContentHash: compiled.ContentHash, ClassID: "SensorReading", Properties: map[string]models.SemanticProperty{"temperature": {PropertyID: "temperature", SourceField: "temperature", Value: 82.5, Range: []string{"decimal"}, Kind: "datatype"}}}
	other := models.NewCIR(models.SourceTypeAPI, "api://readings/2", models.DataFormatJSON, map[string]interface{}{"temperature": 12.5})
	other.Metadata.Ontology = &models.CIRSemanticMapping{OntologyID: ontologyRecord.ID, ContentHash: compiled.ContentHash, ClassID: "SensorReading", Properties: map[string]models.SemanticProperty{"temperature": {PropertyID: "temperature", SourceField: "temperature", Value: 12.5, Range: []string{"decimal"}, Kind: "datatype"}}}

	svc := NewService(store)
	svc.RegisterPlugin("retrieve-sample", &retrieveSamplePlugin{sample: []*models.CIR{matching, other}})
	cfg, err := svc.CreateStorageConfigWithOntology("project-1", "retrieve-sample", map[string]interface{}{"connection_string": "mock://retrieve"}, ontologyRecord.ID)
	if err != nil {
		t.Fatalf("failed to create storage config: %v", err)
	}
	resp, err := svc.RetrieveByOntology("project-1", ontologyRecord.ID, &models.OntologyRetrieveRequest{ProjectID: "project-1", ClassID: "SensorReading", Properties: []string{"temperature"}, StorageIDs: []string{cfg.ID}, Filters: []models.OntologyRetrieveFilter{{Property: "temperature", Operator: "gt", Value: 80.0}}, Limit: 10})
	if err != nil {
		t.Fatalf("RetrieveByOntology failed: %v", err)
	}
	if resp.Count != 1 {
		t.Fatalf("expected one result, got %+v", resp)
	}
	if resp.Results[0].Properties["temperature"] != 82.5 {
		t.Fatalf("expected projected temperature, got %+v", resp.Results[0].Properties)
	}
}
