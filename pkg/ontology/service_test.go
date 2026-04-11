package ontology

import (
	"errors"
	"testing"
	"time"

	"github.com/mimir-aip/mimir-aip-go/pkg/metadatastore"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

func saveOntologyTestProject(t *testing.T, store metadatastore.MetadataStore, projectID string) {
	t.Helper()
	now := time.Now().UTC()
	project := &models.Project{
		ID:          projectID,
		Name:        projectID,
		Description: "test project",
		Version:     "v1",
		Status:      models.ProjectStatusActive,
		Metadata:    models.ProjectMetadata{CreatedAt: now, UpdatedAt: now},
	}
	if err := store.SaveProject(project); err != nil {
		t.Fatalf("failed to save test project %s: %v", projectID, err)
	}
}

func setupOntologyService(t *testing.T) (*Service, *metadatastore.SQLiteStore) {
	t.Helper()
	store, err := metadatastore.NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	saveOntologyTestProject(t, store, "test-project-id")
	saveOntologyTestProject(t, store, "other-project-id")
	return NewService(store), store
}

func TestCreateOntology(t *testing.T) {
	service, store := setupOntologyService(t)
	defer store.Close()

	req := &models.OntologyCreateRequest{
		ProjectID:   "test-project-id",
		Name:        "Test Ontology",
		Description: "A test ontology",
		Version:     "1.0",
		Content:     "@prefix : <http://example.org/mimir#> .\n@prefix owl: <http://www.w3.org/2002/07/owl#> .\n\n:Entity a owl:Class .",
		Status:      "draft",
		IsGenerated: false,
	}

	ontology, err := service.CreateOntology(req)
	if err != nil {
		t.Fatalf("CreateOntology failed: %v", err)
	}
	if ontology.ID == "" {
		t.Error("Ontology ID should not be empty")
	}
	if ontology.Name != "Test Ontology" {
		t.Errorf("Expected name 'Test Ontology', got %s", ontology.Name)
	}
	if ontology.Status != "draft" {
		t.Errorf("Expected status 'draft', got %s", ontology.Status)
	}
}

func TestCreateOntologyRejectsMissingProject(t *testing.T) {
	service, store := setupOntologyService(t)
	defer store.Close()

	_, err := service.CreateOntology(&models.OntologyCreateRequest{
		ProjectID: "missing-project-id",
		Name:      "Invalid",
		Content:   "@prefix : <http://example.org/mimir#> .\n@prefix owl: <http://www.w3.org/2002/07/owl#> .\n\n:Entity a owl:Class .",
		Status:    "draft",
	})
	if err == nil {
		t.Fatal("expected create to fail when project does not exist")
	}
}

func TestGetOntologyForProjectRejectsMismatch(t *testing.T) {
	service, store := setupOntologyService(t)
	defer store.Close()

	created, err := service.CreateOntology(&models.OntologyCreateRequest{
		ProjectID: "test-project-id",
		Name:      "Test Ontology",
		Content:   "@prefix : <http://example.org/mimir#> .\n@prefix owl: <http://www.w3.org/2002/07/owl#> .\n\n:Entity a owl:Class .",
		Status:    "draft",
	})
	if err != nil {
		t.Fatalf("CreateOntology failed: %v", err)
	}

	_, err = service.GetOntologyForProject("other-project-id", created.ID)
	var mismatchErr *OntologyProjectMismatchError
	if !errors.As(err, &mismatchErr) {
		t.Fatalf("expected OntologyProjectMismatchError, got %v", err)
	}
}

func TestGenerateFromExtraction(t *testing.T) {
	service, store := setupOntologyService(t)
	defer store.Close()

	alice := models.ExtractedEntity{Name: "alice", Attributes: map[string]interface{}{"email": "alice@example.com", "department": "Engineering", "age": 30, "entity_type": "Employee"}, Source: "structured", Confidence: 0.9}
	bob := models.ExtractedEntity{Name: "bob", Attributes: map[string]interface{}{"email": "bob@example.com", "department": "Engineering", "entity_type": "Employee"}, Source: "structured", Confidence: 0.9}
	extractionResult := &models.ExtractionResult{
		Entities:      []models.ExtractedEntity{alice, bob},
		Relationships: []models.ExtractedRelationship{{Entity1: &alice, Entity2: &bob, Relation: "reports_to", Confidence: 0.85}},
		Source:        "structured",
	}

	ontology, err := service.GenerateFromExtraction("test-project-id", "Generated Ontology", extractionResult)
	if err != nil {
		t.Fatalf("GenerateFromExtraction failed: %v", err)
	}
	if !ontology.IsGenerated {
		t.Error("Ontology should be marked as generated")
	}
	if ontology.Status != "active" {
		t.Errorf("Expected status 'active', got %s", ontology.Status)
	}
	if !containsString(ontology.Content, "@prefix") || !containsString(ontology.Content, "owl:Class") || !containsString(ontology.Content, "owl:DatatypeProperty") || !containsString(ontology.Content, "owl:ObjectProperty") {
		t.Fatalf("generated ontology content is missing expected Turtle declarations: %s", ontology.Content)
	}
}

func TestUpdateOntology(t *testing.T) {
	service, store := setupOntologyService(t)
	defer store.Close()

	created, err := service.CreateOntology(&models.OntologyCreateRequest{
		ProjectID: "test-project-id",
		Name:      "Test Ontology",
		Content:   "@prefix : <http://example.org/mimir#> .\n@prefix owl: <http://www.w3.org/2002/07/owl#> .\n\n:Entity a owl:Class .",
		Status:    "draft",
	})
	if err != nil {
		t.Fatalf("CreateOntology failed: %v", err)
	}

	newName := "Updated Ontology"
	newStatus := "active"
	updated, err := service.UpdateOntology(created.ID, &models.OntologyUpdateRequest{Name: &newName, Status: &newStatus})
	if err != nil {
		t.Fatalf("UpdateOntology failed: %v", err)
	}
	if updated.Name != "Updated Ontology" {
		t.Errorf("Expected name 'Updated Ontology', got %s", updated.Name)
	}
	if updated.Status != "active" {
		t.Errorf("Expected status 'active', got %s", updated.Status)
	}
}

func TestDeleteOntologyRejectsReferencedOntology(t *testing.T) {
	service, store := setupOntologyService(t)
	defer store.Close()

	created, err := service.CreateOntology(&models.OntologyCreateRequest{
		ProjectID: "test-project-id",
		Name:      "Test Ontology",
		Content:   "@prefix : <http://example.org/mimir#> .\n@prefix owl: <http://www.w3.org/2002/07/owl#> .\n\n:Entity a owl:Class .",
		Status:    "active",
	})
	if err != nil {
		t.Fatalf("CreateOntology failed: %v", err)
	}

	if err := store.SaveMLModel(&models.MLModel{ID: "model-1", ProjectID: "test-project-id", OntologyID: created.ID, Name: "Model", Type: models.ModelTypeDecisionTree, Status: models.ModelStatusDraft, Version: "1.0.0", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}); err != nil {
		t.Fatalf("failed to save ml model: %v", err)
	}
	if err := store.SaveDigitalTwin(&models.DigitalTwin{ID: "twin-1", ProjectID: "test-project-id", OntologyID: created.ID, Name: "Twin", Status: "active", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}); err != nil {
		t.Fatalf("failed to save digital twin: %v", err)
	}
	if err := store.SaveStorageConfig(&models.StorageConfig{ID: "storage-1", ProjectID: "test-project-id", PluginType: "filesystem", Config: map[string]interface{}{"path": "/tmp/test"}, OntologyID: created.ID, Active: true, CreatedAt: time.Now().UTC().Format(time.RFC3339), UpdatedAt: time.Now().UTC().Format(time.RFC3339)}); err != nil {
		t.Fatalf("failed to save storage config: %v", err)
	}

	err = service.DeleteOntology(created.ID)
	var inUseErr *OntologyInUseError
	if !errors.As(err, &inUseErr) {
		t.Fatalf("expected OntologyInUseError, got %v", err)
	}
}

func TestDeleteOntology(t *testing.T) {
	service, store := setupOntologyService(t)
	defer store.Close()

	created, err := service.CreateOntology(&models.OntologyCreateRequest{
		ProjectID: "test-project-id",
		Name:      "Test Ontology",
		Content:   "@prefix : <http://example.org/mimir#> .\n@prefix owl: <http://www.w3.org/2002/07/owl#> .\n\n:Entity a owl:Class .",
		Status:    "draft",
	})
	if err != nil {
		t.Fatalf("CreateOntology failed: %v", err)
	}

	err = service.DeleteOntology(created.ID)
	if err != nil {
		t.Fatalf("DeleteOntology failed: %v", err)
	}
	_, err = service.GetOntology(created.ID)
	if err == nil {
		t.Error("Expected error when retrieving deleted ontology, got nil")
	}
}

func TestGetProjectOntologies(t *testing.T) {
	service, store := setupOntologyService(t)
	defer store.Close()

	projectID := "test-project-id"
	for i := 0; i < 3; i++ {
		_, err := service.CreateOntology(&models.OntologyCreateRequest{
			ProjectID: projectID,
			Name:      string(rune('A' + i)),
			Content:   "@prefix : <http://example.org/mimir#> .\n@prefix owl: <http://www.w3.org/2002/07/owl#> .\n\n:Entity a owl:Class .",
			Status:    "draft",
		})
		if err != nil {
			t.Fatalf("CreateOntology failed: %v", err)
		}
	}

	ontologies, err := service.GetProjectOntologies(projectID)
	if err != nil {
		t.Fatalf("GetProjectOntologies failed: %v", err)
	}
	if len(ontologies) != 3 {
		t.Errorf("Expected 3 ontologies, got %d", len(ontologies))
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestGenerateFromExtraction_UpsertsGeneratedOntology(t *testing.T) {
	service, store := setupOntologyService(t)
	defer store.Close()
	projectID := "test-project-id"
	name := "auto-test-project-id"

	first := &models.ExtractionResult{Entities: []models.ExtractedEntity{{Name: "entity-a", Attributes: map[string]interface{}{"entity_type": "Device", "serial": "A-1"}, Source: "structured", Confidence: 0.9}}}
	firstOntology, err := service.GenerateFromExtraction(projectID, name, first)
	if err != nil {
		t.Fatalf("first GenerateFromExtraction failed: %v", err)
	}
	second := &models.ExtractionResult{Entities: []models.ExtractedEntity{{Name: "entity-a", Attributes: map[string]interface{}{"entity_type": "Device", "serial": "A-1", "region": "west"}, Source: "structured", Confidence: 0.9}}}
	secondOntology, err := service.GenerateFromExtraction(projectID, name, second)
	if err != nil {
		t.Fatalf("second GenerateFromExtraction failed: %v", err)
	}
	if secondOntology.ID != firstOntology.ID {
		t.Fatalf("expected generated ontology upsert to reuse ID %s, got %s", firstOntology.ID, secondOntology.ID)
	}
	if !containsString(secondOntology.Content, "region") {
		t.Fatalf("expected updated ontology content to include new attribute, got %s", secondOntology.Content)
	}
	ontologies, err := service.GetProjectOntologies(projectID)
	if err != nil {
		t.Fatalf("GetProjectOntologies failed: %v", err)
	}
	generatedCount := 0
	for _, ont := range ontologies {
		if ont.IsGenerated && ont.Name == name {
			generatedCount++
		}
	}
	if generatedCount != 1 {
		t.Fatalf("expected exactly one generated ontology named %s, found %d", name, generatedCount)
	}
}
