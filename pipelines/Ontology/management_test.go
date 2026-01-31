package ontology

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
	storage "github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/Storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestManagementPlugin creates a test management plugin
func createTestManagementPlugin(t *testing.T) (*ManagementPlugin, string) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	persistence, err := storage.NewPersistenceBackend(dbPath)
	require.NoError(t, err, "Failed to create persistence backend")

	ontologyDir := filepath.Join(tmpDir, "ontologies")
	err = os.MkdirAll(ontologyDir, 0755)
	require.NoError(t, err)

	// Create plugin without TDB2 backend for most tests
	plugin := NewManagementPlugin(persistence, nil, ontologyDir)

	return plugin, tmpDir
}

// TestNewManagementPlugin tests plugin creation
func TestNewManagementPlugin(t *testing.T) {
	plugin, _ := createTestManagementPlugin(t)
	require.NotNil(t, plugin, "Plugin should not be nil")
	assert.NotNil(t, plugin.persistence, "Persistence should be set")
	assert.NotNil(t, plugin.ontologyDir, "Ontology directory should be set")
}

// TestManagementPlugin_GetPluginType tests plugin type
func TestManagementPlugin_GetPluginType(t *testing.T) {
	plugin, _ := createTestManagementPlugin(t)
	assert.Equal(t, "Ontology", plugin.GetPluginType())
}

// TestManagementPlugin_GetPluginName tests plugin name
func TestManagementPlugin_GetPluginName(t *testing.T) {
	plugin, _ := createTestManagementPlugin(t)
	assert.Equal(t, "management", plugin.GetPluginName())
}

// TestManagementPlugin_ValidateConfig tests config validation
func TestManagementPlugin_ValidateConfig(t *testing.T) {
	plugin, _ := createTestManagementPlugin(t)

	// Valid operations
	validOps := []string{"upload", "validate", "list", "get", "update", "delete", "stats"}
	for _, op := range validOps {
		err := plugin.ValidateConfig(map[string]interface{}{"operation": op})
		assert.NoError(t, err, "Should validate operation: %s", op)
	}

	// Missing operation
	err := plugin.ValidateConfig(map[string]interface{}{})
	assert.Error(t, err, "Should error without operation")

	// Invalid operation
	err = plugin.ValidateConfig(map[string]interface{}{"operation": "invalid"})
	assert.Error(t, err, "Should error with invalid operation")
}

// TestManagementPlugin_GetInputSchema tests input schema
func TestManagementPlugin_GetInputSchema(t *testing.T) {
	plugin, _ := createTestManagementPlugin(t)
	schema := plugin.GetInputSchema()

	assert.NotNil(t, schema)
	assert.Equal(t, "object", schema["type"])
	assert.NotNil(t, schema["properties"])
	assert.NotNil(t, schema["properties"].(map[string]interface{})["operation"])
}

// TestManagementPlugin_ExecuteStep_Upload tests upload operation
func TestManagementPlugin_ExecuteStep_Upload(t *testing.T) {
	plugin, _ := createTestManagementPlugin(t)
	ctx := context.Background()

	ontologyData := `
@prefix ex: <http://example.org/> .
ex:TestClass a owl:Class .
`

	stepConfig := pipelines.StepConfig{
		Name:   "ontology_upload",
		Plugin: "Ontology.management",
		Config: map[string]interface{}{
			"operation":     "upload",
			"name":          "Test Ontology",
			"description":   "Test Description",
			"version":       "1.0.0",
			"format":        "turtle",
			"ontology_data": ontologyData,
			"created_by":    "test-user",
		},
	}

	globalContext := pipelines.NewPluginContext()
	result, err := plugin.ExecuteStep(ctx, stepConfig, globalContext)
	require.NoError(t, err, "Should upload ontology without error")
	require.NotNil(t, result, "Should return result")

	// Verify result
	ontologyID, exists := result.Get("ontology_id")
	assert.True(t, exists, "Should have ontology_id")
	assert.NotEmpty(t, ontologyID, "Ontology ID should not be empty")

	status, _ := result.Get("status")
	assert.Equal(t, "success", status)
}

// TestManagementPlugin_ExecuteStep_Upload_MissingFields tests upload with missing fields
func TestManagementPlugin_ExecuteStep_Upload_MissingFields(t *testing.T) {
	plugin, _ := createTestManagementPlugin(t)
	ctx := context.Background()

	stepConfig := pipelines.StepConfig{
		Name:   "ontology_upload",
		Plugin: "Ontology.management",
		Config: map[string]interface{}{
			"operation": "upload",
			"name":      "Test Ontology",
			// Missing version and ontology_data
		},
	}

	globalContext := pipelines.NewPluginContext()
	_, err := plugin.ExecuteStep(ctx, stepConfig, globalContext)
	assert.Error(t, err, "Should error with missing required fields")
}

// TestManagementPlugin_ExecuteStep_Validate tests validate operation
func TestManagementPlugin_ExecuteStep_Validate(t *testing.T) {
	plugin, _ := createTestManagementPlugin(t)
	ctx := context.Background()

	validTurtle := `
@prefix ex: <http://example.org/> .
ex:Class1 a owl:Class .
`

	stepConfig := pipelines.StepConfig{
		Name:   "ontology_validate",
		Plugin: "Ontology.management",
		Config: map[string]interface{}{
			"operation":     "validate",
			"ontology_data": validTurtle,
			"format":        "turtle",
		},
	}

	globalContext := pipelines.NewPluginContext()
	result, err := plugin.ExecuteStep(ctx, stepConfig, globalContext)
	require.NoError(t, err, "Should validate without error")
	require.NotNil(t, result)

	valid, exists := result.Get("valid")
	assert.True(t, exists, "Should have valid field")
	assert.True(t, valid.(bool), "Should be valid")
}

// TestManagementPlugin_ExecuteStep_Validate_Invalid tests validation of invalid data
func TestManagementPlugin_ExecuteStep_Validate_Invalid(t *testing.T) {
	plugin, _ := createTestManagementPlugin(t)
	ctx := context.Background()

	// Invalid Turtle (no proper terminator)
	invalidTurtle := `@prefix ex: <http://example.org/> ex:Class1 a owl:Class`

	stepConfig := pipelines.StepConfig{
		Name:   "ontology_validate",
		Plugin: "Ontology.management",
		Config: map[string]interface{}{
			"operation":     "validate",
			"ontology_data": invalidTurtle,
			"format":        "turtle",
		},
	}

	globalContext := pipelines.NewPluginContext()
	result, err := plugin.ExecuteStep(ctx, stepConfig, globalContext)
	require.NoError(t, err, "Validation should complete")
	require.NotNil(t, result)

	valid, _ := result.Get("valid")
	assert.False(t, valid.(bool), "Should be invalid")

	errors, exists := result.Get("errors")
	assert.True(t, exists, "Should have errors field")
	assert.NotNil(t, errors)
}

// TestManagementPlugin_ExecuteStep_List tests list operation
func TestManagementPlugin_ExecuteStep_List(t *testing.T) {
	plugin, tmpDir := createTestManagementPlugin(t)
	ctx := context.Background()

	// Create a test ontology first
	createTestOntologyFile(t, tmpDir, "ont-001", "Test Ontology")

	stepConfig := pipelines.StepConfig{
		Name:   "ontology_list",
		Plugin: "Ontology.management",
		Config: map[string]interface{}{
			"operation": "list",
		},
	}

	globalContext := pipelines.NewPluginContext()
	result, err := plugin.ExecuteStep(ctx, stepConfig, globalContext)
	require.NoError(t, err, "Should list without error")
	require.NotNil(t, result)

	ontologies, exists := result.Get("ontologies")
	assert.True(t, exists, "Should have ontologies")
	assert.NotNil(t, ontologies)

	count, _ := result.Get("count")
	assert.GreaterOrEqual(t, count.(int), 0, "Count should be >= 0")
}

// TestManagementPlugin_ExecuteStep_List_WithStatus tests list with status filter
func TestManagementPlugin_ExecuteStep_List_WithStatus(t *testing.T) {
	plugin, _ := createTestManagementPlugin(t)
	ctx := context.Background()

	stepConfig := pipelines.StepConfig{
		Name:   "ontology_list",
		Plugin: "Ontology.management",
		Config: map[string]interface{}{
			"operation": "list",
			"status":    "active",
		},
	}

	globalContext := pipelines.NewPluginContext()
	result, err := plugin.ExecuteStep(ctx, stepConfig, globalContext)
	require.NoError(t, err)
	require.NotNil(t, result)
}

// TestManagementPlugin_ExecuteStep_Get tests get operation
func TestManagementPlugin_ExecuteStep_Get(t *testing.T) {
	plugin, tmpDir := createTestManagementPlugin(t)
	ctx := context.Background()

	// Create a test ontology
	createTestOntologyFile(t, tmpDir, "ont-get-001", "Get Test Ontology")

	stepConfig := pipelines.StepConfig{
		Name:   "ontology_get",
		Plugin: "Ontology.management",
		Config: map[string]interface{}{
			"operation":       "get",
			"ontology_id":     "ont-get-001",
			"include_content": false,
		},
	}

	globalContext := pipelines.NewPluginContext()
	result, err := plugin.ExecuteStep(ctx, stepConfig, globalContext)
	require.NoError(t, err, "Should get ontology without error")
	require.NotNil(t, result)

	ontology, exists := result.Get("ontology")
	assert.True(t, exists, "Should have ontology")
	assert.NotNil(t, ontology)
}

// TestManagementPlugin_ExecuteStep_Get_NotFound tests get with non-existent ID
func TestManagementPlugin_ExecuteStep_Get_NotFound(t *testing.T) {
	plugin, _ := createTestManagementPlugin(t)
	ctx := context.Background()

	stepConfig := pipelines.StepConfig{
		Name:   "ontology_get",
		Plugin: "Ontology.management",
		Config: map[string]interface{}{
			"operation":   "get",
			"ontology_id": "non-existent-id",
		},
	}

	globalContext := pipelines.NewPluginContext()
	_, err := plugin.ExecuteStep(ctx, stepConfig, globalContext)
	assert.Error(t, err, "Should error for non-existent ontology")
}

// TestManagementPlugin_ExecuteStep_Update tests update operation
func TestManagementPlugin_ExecuteStep_Update(t *testing.T) {
	plugin, tmpDir := createTestManagementPlugin(t)
	ctx := context.Background()

	// Create a test ontology first
	createTestOntologyFile(t, tmpDir, "ont-update-001", "Update Test Ontology")

	updatedData := `
@prefix ex: <http://example.org/> .
ex:UpdatedClass a owl:Class .
`

	stepConfig := pipelines.StepConfig{
		Name:   "ontology_update",
		Plugin: "Ontology.management",
		Config: map[string]interface{}{
			"operation":     "update",
			"ontology_id":   "ont-update-001",
			"name":          "Updated Name",
			"description":   "Updated Description",
			"ontology_data": updatedData,
			"modified_by":   "test-user",
		},
	}

	globalContext := pipelines.NewPluginContext()
	result, err := plugin.ExecuteStep(ctx, stepConfig, globalContext)
	require.NoError(t, err, "Should update without error")
	require.NotNil(t, result)

	status, _ := result.Get("status")
	assert.Equal(t, "updated", status)
}

// TestManagementPlugin_ExecuteStep_Delete tests delete operation
func TestManagementPlugin_ExecuteStep_Delete(t *testing.T) {
	plugin, tmpDir := createTestManagementPlugin(t)
	ctx := context.Background()

	// Create a test ontology
	createTestOntologyFile(t, tmpDir, "ont-delete-001", "Delete Test Ontology")

	stepConfig := pipelines.StepConfig{
		Name:   "ontology_delete",
		Plugin: "Ontology.management",
		Config: map[string]interface{}{
			"operation":   "delete",
			"ontology_id": "ont-delete-001",
		},
	}

	globalContext := pipelines.NewPluginContext()
	result, err := plugin.ExecuteStep(ctx, stepConfig, globalContext)
	require.NoError(t, err, "Should delete without error")
	require.NotNil(t, result)

	status, _ := result.Get("status")
	assert.Equal(t, "deleted", status)
}

// TestManagementPlugin_ExecuteStep_Delete_NotFound tests delete with non-existent ID
func TestManagementPlugin_ExecuteStep_Delete_NotFound(t *testing.T) {
	plugin, _ := createTestManagementPlugin(t)
	ctx := context.Background()

	stepConfig := pipelines.StepConfig{
		Name:   "ontology_delete",
		Plugin: "Ontology.management",
		Config: map[string]interface{}{
			"operation":   "delete",
			"ontology_id": "non-existent-id",
		},
	}

	globalContext := pipelines.NewPluginContext()
	_, err := plugin.ExecuteStep(ctx, stepConfig, globalContext)
	assert.Error(t, err, "Should error for non-existent ontology")
}

// TestManagementPlugin_ExecuteStep_Stats tests stats operation
func TestManagementPlugin_ExecuteStep_Stats(t *testing.T) {
	plugin, tmpDir := createTestManagementPlugin(t)
	ctx := context.Background()

	// Create a test ontology
	createTestOntologyFile(t, tmpDir, "ont-stats-001", "Stats Test Ontology")

	stepConfig := pipelines.StepConfig{
		Name:   "ontology_stats",
		Plugin: "Ontology.management",
		Config: map[string]interface{}{
			"operation":   "stats",
			"ontology_id": "ont-stats-001",
		},
	}

	globalContext := pipelines.NewPluginContext()
	result, err := plugin.ExecuteStep(ctx, stepConfig, globalContext)
	require.NoError(t, err, "Should get stats without error")
	require.NotNil(t, result)

	stats, exists := result.Get("stats")
	assert.True(t, exists, "Should have stats")
	assert.NotNil(t, stats)

	ontologyName, _ := result.Get("ontology_name")
	assert.Equal(t, "Stats Test Ontology", ontologyName)
}

// TestManagementPlugin_validateOntologyData tests ontology validation
func TestManagementPlugin_validateOntologyData(t *testing.T) {
	plugin, _ := createTestManagementPlugin(t)

	tests := []struct {
		name        string
		data        string
		format      string
		expectValid bool
	}{
		{
			name:        "Valid Turtle",
			data:        `@prefix ex: <http://example.org/> . ex:Class a owl:Class .`,
			format:      "turtle",
			expectValid: true,
		},
		{
			name:        "Empty data",
			data:        "",
			format:      "turtle",
			expectValid: false,
		},
		{
			name:        "Whitespace only",
			data:        "   ",
			format:      "turtle",
			expectValid: false,
		},
		{
			name:        "Valid RDF/XML",
			data:        `<?xml version="1.0"?><rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#"></rdf:RDF>`,
			format:      "rdfxml",
			expectValid: true,
		},
		{
			name:        "Invalid RDF/XML",
			data:        `<invalid>data</invalid>`,
			format:      "rdfxml",
			expectValid: false,
		},
		{
			name:        "Valid N-Triples",
			data:        `<http://example.org/s> <http://example.org/p> <http://example.org/o> .`,
			format:      "ntriples",
			expectValid: true,
		},
		{
			name:        "Valid JSON-LD",
			data:        `{"@context": {}, "@type": "Thing"}`,
			format:      "jsonld",
			expectValid: true,
		},
		{
			name:        "Invalid JSON-LD",
			data:        `not json`,
			format:      "jsonld",
			expectValid: false,
		},
		{
			name:        "Unknown format",
			data:        "some data",
			format:      "unknown",
			expectValid: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := plugin.validateOntologyData(test.data, test.format)
			assert.Equal(t, test.expectValid, result.Valid,
				"Validation result for %s should be valid=%v", test.name, test.expectValid)
		})
	}
}

// TestManagementPlugin_validateTurtle tests Turtle validation
func TestManagementPlugin_validateTurtle(t *testing.T) {
	plugin, _ := createTestManagementPlugin(t)

	validTurtle := `@prefix ex: <http://example.org/> .
ex:Class1 a owl:Class .
ex:Class2 a owl:Class .`

	result := plugin.validateTurtle(validTurtle)
	assert.True(t, result.Valid, "Should be valid Turtle")

	// Turtle without proper terminator
	invalidTurtle := `@prefix ex: <http://example.org/> ex:Class a owl:Class`
	result = plugin.validateTurtle(invalidTurtle)
	assert.False(t, result.Valid, "Should be invalid without terminator")
}

// TestManagementPlugin_validateRDFXML tests RDF/XML validation
func TestManagementPlugin_validateRDFXML(t *testing.T) {
	plugin, _ := createTestManagementPlugin(t)

	validRDF := `<?xml version="1.0"?>
<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
</rdf:RDF>`

	result := plugin.validateRDFXML(validRDF)
	assert.True(t, result.Valid, "Should be valid RDF/XML")

	invalidRDF := `<invalid>data</invalid>`
	result = plugin.validateRDFXML(invalidRDF)
	assert.False(t, result.Valid, "Should be invalid without rdf:RDF")
}

// TestManagementPlugin_validateNTriples tests N-Triples validation
func TestManagementPlugin_validateNTriples(t *testing.T) {
	plugin, _ := createTestManagementPlugin(t)

	validNTriples := `<http://example.org/s> <http://example.org/p> <http://example.org/o> .
<http://example.org/s2> <http://example.org/p2> "literal" .`

	result := plugin.validateNTriples(validNTriples)
	assert.True(t, result.Valid, "Should be valid N-Triples")

	invalidNTriples := "not a valid triple"
	result = plugin.validateNTriples(invalidNTriples)
	assert.False(t, result.Valid, "Should be invalid without triples")
}

// TestManagementPlugin_validateJSONLD tests JSON-LD validation
func TestManagementPlugin_validateJSONLD(t *testing.T) {
	plugin, _ := createTestManagementPlugin(t)

	validJSONLD := `{
		"@context": {"ex": "http://example.org/"},
		"@type": "ex:Thing"
	}`

	result := plugin.validateJSONLD(validJSONLD)
	assert.True(t, result.Valid, "Should be valid JSON-LD")

	invalidJSONLD := `not json at all`
	result = plugin.validateJSONLD(invalidJSONLD)
	assert.False(t, result.Valid, "Should be invalid JSON")
}

// TestSanitizeFilename tests filename sanitization
func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Normal Name", "Normal_Name"},
		{"Name/With/Slashes", "Name-With-Slashes"},
		{"Name\\With\\Backslashes", "Name-With-Backslashes"},
		{"Name:With:Colons", "Name-With-Colons"},
		{"Name*With*Stars", "Name-With-Stars"},
		{"Name?With?Questions", "Name-With-Questions"},
		{`Name"With"Quotes`, "Name-With-Quotes"},
		{"Name<With>Angles", "Name-With-Angles"},
		{"Name|With|Pipes", "Name-With-Pipes"},
	}

	for _, test := range tests {
		result := sanitizeFilename(test.input)
		assert.Equal(t, test.expected, result,
			"Sanitizing '%s' should give '%s'", test.input, test.expected)
	}
}

// TestGetFileExtension tests file extension mapping
func TestGetFileExtension(t *testing.T) {
	tests := []struct {
		format   string
		expected string
	}{
		{"turtle", "ttl"},
		{"ttl", "ttl"},
		{"rdfxml", "rdf"},
		{"ntriples", "nt"},
		{"jsonld", "jsonld"},
		{"unknown", "ttl"},
	}

	for _, test := range tests {
		result := getFileExtension(test.format)
		assert.Equal(t, test.expected, result,
			"Format '%s' should give extension '%s'", test.format, test.expected)
	}
}

// TestValidationResult_Structure tests validation result structure
func TestValidationResult_Structure(t *testing.T) {
	result := ValidationResult{
		Valid: true,
		Errors: []ValidationError{
			{Severity: "error", Message: "Test error"},
		},
		Warnings: []ValidationError{
			{Severity: "warning", Message: "Test warning"},
		},
	}

	assert.True(t, result.Valid)
	assert.Len(t, result.Errors, 1)
	assert.Len(t, result.Warnings, 1)
}

// TestValidationError_Structure tests validation error structure
func TestValidationError_Structure(t *testing.T) {
	err := ValidationError{
		Severity: "error",
		Message:  "Something went wrong",
	}

	assert.Equal(t, "error", err.Severity)
	assert.Equal(t, "Something went wrong", err.Message)
}

// TestOntologyStats_Structure tests ontology stats structure
func TestOntologyStats_Structure(t *testing.T) {
	stats := OntologyStats{
		OntologyID:   "ont-001",
		TotalTriples: 1500,
	}

	assert.Equal(t, "ont-001", stats.OntologyID)
	assert.Equal(t, 1500, stats.TotalTriples)
}

// Helper function to create a test ontology
func createTestOntologyFile(t *testing.T, tmpDir, id, name string) {
	plugin, _ := createTestManagementPlugin(t)
	// Reuse the plugin's persistence
	// We need to insert directly into the database

	ontologyDir := filepath.Join(tmpDir, "ontologies")
	fileName := fmt.Sprintf("%s-1.0.0.ttl", sanitizeFilename(name))
	filePath := filepath.Join(ontologyDir, fileName)

	ontologyData := `@prefix ex: <http://example.org/> .
ex:TestClass a owl:Class .`

	err := os.WriteFile(filePath, []byte(ontologyData), 0644)
	require.NoError(t, err, "Failed to write ontology file")

	// Insert into database
	db := plugin.persistence.GetDB()
	_, err = db.Exec(`
		INSERT INTO ontologies (id, name, version, file_path, tdb2_graph, format, status, auto_version)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, id, name, "1.0.0", filePath, fmt.Sprintf("http://mimir-aip.io/ontology/%s", id), "turtle", "active", true)
	require.NoError(t, err, "Failed to insert ontology into database")
}
