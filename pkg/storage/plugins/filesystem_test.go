package plugins_test

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"github.com/mimir-aip/mimir-aip-go/pkg/storage/plugins"
)

func TestFilesystemPlugin(t *testing.T) {
	// Create temp directory for test
	tempDir, err := ioutil.TempDir("", "storage-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Initialize plugin
	plugin := plugins.NewFilesystemPlugin()

	config := &models.PluginConfig{
		ConnectionString: tempDir,
	}

	if err := plugin.Initialize(config); err != nil {
		t.Fatalf("Failed to initialize plugin: %v", err)
	}

	// Test health check
	healthy, err := plugin.HealthCheck()
	if err != nil {
		t.Fatalf("Health check failed: %v", err)
	}
	if !healthy {
		t.Error("Expected plugin to be healthy")
	}

	// Test create schema
	ontology := &models.OntologyDefinition{
		Entities: []models.EntityDefinition{
			{
				Name: "Employee",
				Attributes: []models.AttributeDefinition{
					{Name: "id", Type: "string"},
					{Name: "name", Type: "string"},
				},
				PrimaryKey: []string{"id"},
			},
		},
		Relationships: []models.RelationshipDefinition{},
	}

	if err := plugin.CreateSchema(ontology); err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	// Test store
	cir := models.NewCIR(
		models.SourceTypeFile,
		"/test/employees.csv",
		models.DataFormatCSV,
		map[string]interface{}{
			"id":   "1",
			"name": "John Doe",
		},
	)
	cir.SetParameter("entity_type", "Employee")

	result, err := plugin.Store(cir)
	if err != nil {
		t.Fatalf("Failed to store data: %v", err)
	}

	if !result.Success {
		t.Error("Expected store to succeed")
	}

	if result.AffectedItems != 1 {
		t.Errorf("Expected 1 affected item, got %d", result.AffectedItems)
	}

	// Test retrieve
	query := &models.CIRQuery{
		EntityType: "Employee",
	}

	results, err := plugin.Retrieve(query)
	if err != nil {
		t.Fatalf("Failed to retrieve data: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}

	// Test retrieve with filter
	queryWithFilter := &models.CIRQuery{
		EntityType: "Employee",
		Filters: []models.CIRCondition{
			{
				Attribute: "id",
				Operator:  "eq",
				Value:     "1",
			},
		},
	}

	filteredResults, err := plugin.Retrieve(queryWithFilter)
	if err != nil {
		t.Fatalf("Failed to retrieve filtered data: %v", err)
	}

	if len(filteredResults) != 1 {
		t.Errorf("Expected 1 filtered result, got %d", len(filteredResults))
	}

	// Test delete
	deleteQuery := &models.CIRQuery{
		EntityType: "Employee",
		Filters: []models.CIRCondition{
			{
				Attribute: "id",
				Operator:  "eq",
				Value:     "1",
			},
		},
	}

	deleteResult, err := plugin.Delete(deleteQuery)
	if err != nil {
		t.Fatalf("Failed to delete data: %v", err)
	}

	if !deleteResult.Success {
		t.Error("Expected delete to succeed")
	}

	if deleteResult.AffectedItems != 1 {
		t.Errorf("Expected 1 deleted item, got %d", deleteResult.AffectedItems)
	}

	// Verify deletion
	verifyResults, err := plugin.Retrieve(query)
	if err != nil {
		t.Fatalf("Failed to verify deletion: %v", err)
	}

	if len(verifyResults) != 0 {
		t.Errorf("Expected 0 results after deletion, got %d", len(verifyResults))
	}
}

func TestFilesystemPluginStoreArray(t *testing.T) {
	// Create temp directory for test
	tempDir, err := ioutil.TempDir("", "storage-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Initialize plugin
	plugin := plugins.NewFilesystemPlugin()

	config := &models.PluginConfig{
		ConnectionString: tempDir,
	}

	if err := plugin.Initialize(config); err != nil {
		t.Fatalf("Failed to initialize plugin: %v", err)
	}

	// Store array of items
	cir := models.NewCIR(
		models.SourceTypeAPI,
		"https://api.example.com/employees",
		models.DataFormatJSON,
		[]interface{}{
			map[string]interface{}{"id": "1", "name": "John"},
			map[string]interface{}{"id": "2", "name": "Jane"},
			map[string]interface{}{"id": "3", "name": "Bob"},
		},
	)
	cir.SetParameter("entity_type", "Employee")

	result, err := plugin.Store(cir)
	if err != nil {
		t.Fatalf("Failed to store array: %v", err)
	}

	if result.AffectedItems != 3 {
		t.Errorf("Expected 3 affected items, got %d", result.AffectedItems)
	}

	// Retrieve all
	query := &models.CIRQuery{
		EntityType: "Employee",
	}

	results, err := plugin.Retrieve(query)
	if err != nil {
		t.Fatalf("Failed to retrieve data: %v", err)
	}

	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}
}

func TestFilesystemPluginGetMetadata(t *testing.T) {
	plugin := plugins.NewFilesystemPlugin()

	metadata, err := plugin.GetMetadata()
	if err != nil {
		t.Fatalf("Failed to get metadata: %v", err)
	}

	if metadata.StorageType != "filesystem" {
		t.Errorf("Expected storage type 'filesystem', got '%s'", metadata.StorageType)
	}

	if len(metadata.Capabilities) == 0 {
		t.Error("Expected capabilities to be listed")
	}
}
