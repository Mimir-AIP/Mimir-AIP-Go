package ontology

import (
	"testing"

	"github.com/mimir-aip/mimir-aip-go/pkg/metadatastore"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

func TestCreateOntology(t *testing.T) {
	// Create in-memory SQLite database for testing
	store, err := metadatastore.NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	service := NewService(store)

	// Create a test ontology
	req := &models.OntologyCreateRequest{
		ProjectID:   "test-project-id",
		Name:        "Test Ontology",
		Description: "A test ontology",
		Version:     "1.0",
		Content:     "@prefix : <http://example.org/mimir#> .",
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

func TestGetOntology(t *testing.T) {
	store, err := metadatastore.NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	service := NewService(store)

	// Create an ontology
	req := &models.OntologyCreateRequest{
		ProjectID: "test-project-id",
		Name:      "Test Ontology",
		Content:   "@prefix : <http://example.org/mimir#> .",
		Status:    "draft",
	}

	created, err := service.CreateOntology(req)
	if err != nil {
		t.Fatalf("CreateOntology failed: %v", err)
	}

	// Retrieve the ontology
	retrieved, err := service.GetOntology(created.ID)
	if err != nil {
		t.Fatalf("GetOntology failed: %v", err)
	}

	if retrieved.ID != created.ID {
		t.Errorf("Expected ID %s, got %s", created.ID, retrieved.ID)
	}

	if retrieved.Name != "Test Ontology" {
		t.Errorf("Expected name 'Test Ontology', got %s", retrieved.Name)
	}
}

func TestGenerateFromExtraction(t *testing.T) {
	store, err := metadatastore.NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	service := NewService(store)

	// Create mock extraction results
	alice := models.ExtractedEntity{
		Name: "alice",
		Attributes: map[string]interface{}{
			"email":      "alice@example.com",
			"department": "Engineering",
			"age":        30,
		},
		Source:     "structured",
		Confidence: 0.9,
	}

	bob := models.ExtractedEntity{
		Name: "bob",
		Attributes: map[string]interface{}{
			"email":      "bob@example.com",
			"department": "Engineering",
		},
		Source:     "structured",
		Confidence: 0.9,
	}

	extractionResult := &models.ExtractionResult{
		Entities: []models.ExtractedEntity{alice, bob},
		Relationships: []models.ExtractedRelationship{
			{
				Entity1:    &alice,
				Entity2:    &bob,
				Relation:   "reports_to",
				Confidence: 0.85,
			},
		},
		Source: "structured",
	}

	// Generate ontology from extraction
	ontology, err := service.GenerateFromExtraction("test-project-id", "Generated Ontology", extractionResult)
	if err != nil {
		t.Fatalf("GenerateFromExtraction failed: %v", err)
	}

	if !ontology.IsGenerated {
		t.Error("Ontology should be marked as generated")
	}

	if ontology.Status != "draft" {
		t.Errorf("Expected status 'draft', got %s", ontology.Status)
	}

	// Check that Turtle content contains expected elements
	if !containsString(ontology.Content, "@prefix") {
		t.Error("Turtle content should contain @prefix declarations")
	}

	if !containsString(ontology.Content, "owl:Class") {
		t.Error("Turtle content should contain owl:Class declarations")
	}

	if !containsString(ontology.Content, "owl:DatatypeProperty") {
		t.Error("Turtle content should contain owl:DatatypeProperty declarations")
	}

	if !containsString(ontology.Content, "owl:ObjectProperty") {
		t.Error("Turtle content should contain owl:ObjectProperty declarations")
	}
}

func TestUpdateOntology(t *testing.T) {
	store, err := metadatastore.NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	service := NewService(store)

	// Create an ontology
	req := &models.OntologyCreateRequest{
		ProjectID: "test-project-id",
		Name:      "Test Ontology",
		Content:   "@prefix : <http://example.org/mimir#> .",
		Status:    "draft",
	}

	created, err := service.CreateOntology(req)
	if err != nil {
		t.Fatalf("CreateOntology failed: %v", err)
	}

	// Update the ontology
	newName := "Updated Ontology"
	newStatus := "active"
	updateReq := &models.OntologyUpdateRequest{
		Name:   &newName,
		Status: &newStatus,
	}

	updated, err := service.UpdateOntology(created.ID, updateReq)
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

func TestDeleteOntology(t *testing.T) {
	store, err := metadatastore.NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	service := NewService(store)

	// Create an ontology
	req := &models.OntologyCreateRequest{
		ProjectID: "test-project-id",
		Name:      "Test Ontology",
		Content:   "@prefix : <http://example.org/mimir#> .",
		Status:    "draft",
	}

	created, err := service.CreateOntology(req)
	if err != nil {
		t.Fatalf("CreateOntology failed: %v", err)
	}

	// Delete the ontology
	err = service.DeleteOntology(created.ID)
	if err != nil {
		t.Fatalf("DeleteOntology failed: %v", err)
	}

	// Try to retrieve the deleted ontology
	_, err = service.GetOntology(created.ID)
	if err == nil {
		t.Error("Expected error when retrieving deleted ontology, got nil")
	}
}

func TestGetProjectOntologies(t *testing.T) {
	store, err := metadatastore.NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	service := NewService(store)

	projectID := "test-project-id"

	// Create multiple ontologies for the same project
	for i := 0; i < 3; i++ {
		req := &models.OntologyCreateRequest{
			ProjectID: projectID,
			Name:      "Test Ontology",
			Content:   "@prefix : <http://example.org/mimir#> .",
			Status:    "draft",
		}

		_, err := service.CreateOntology(req)
		if err != nil {
			t.Fatalf("CreateOntology failed: %v", err)
		}
	}

	// Retrieve ontologies for the project
	ontologies, err := service.GetProjectOntologies(projectID)
	if err != nil {
		t.Fatalf("GetProjectOntologies failed: %v", err)
	}

	if len(ontologies) != 3 {
		t.Errorf("Expected 3 ontologies, got %d", len(ontologies))
	}
}

// Helper function to check if a string contains a substring
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
