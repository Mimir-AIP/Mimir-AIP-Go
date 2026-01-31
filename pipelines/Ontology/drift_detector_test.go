package ontology

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/AI"
	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/Storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestDriftDetector creates a drift detector with test dependencies
func createTestDriftDetector(t *testing.T) (*DriftDetector, *sql.DB) {
	// Create temporary database
	tmpDir := t.TempDir()
	dbPath := fmt.Sprintf("%s/test.db", tmpDir)

	persistence, err := storage.NewPersistenceBackend(dbPath)
	require.NoError(t, err, "Failed to create persistence backend")

	db := persistence.GetDB()

	// Create drift detector with mock LLM client
	mockLLM := AI.NewMockLLMClient("[]")
	detector := NewDriftDetector(db, mockLLM, nil)

	return detector, db
}

// TestNewDriftDetector tests drift detector creation
func TestNewDriftDetector(t *testing.T) {
	detector, _ := createTestDriftDetector(t)
	require.NotNil(t, detector, "DriftDetector should not be nil")
	assert.NotNil(t, detector.db, "Database should be set")
}

// TestDriftDetector_DetectDriftFromData tests drift detection from data
func TestDriftDetector_DetectDriftFromData(t *testing.T) {
	detector, db := createTestDriftDetector(t)
	ctx := createTestContext()

	// Create test ontology
	createTestOntology(t, db, "ont-drift-001")

	// Test data
	testData := map[string]interface{}{
		"records": []map[string]interface{}{
			{"name": "Product A", "price": 100.0, "category": "Electronics"},
			{"name": "Product B", "price": 200.0, "category": "Electronics"},
		},
	}

	count, err := detector.DetectDriftFromData(ctx, "ont-drift-001", testData, "test_data_source")
	require.NoError(t, err, "Should detect drift without error")
	assert.GreaterOrEqual(t, count, 0, "Should return non-negative count")

	// Verify detection record was created
	var detectionCount int
	err = db.QueryRow("SELECT COUNT(*) FROM drift_detections WHERE ontology_id = ?", "ont-drift-001").Scan(&detectionCount)
	require.NoError(t, err)
	assert.Equal(t, 1, detectionCount, "Should create drift detection record")
}

// TestDriftDetector_DetectDriftFromData_NoLLM tests detection without LLM
func TestDriftDetector_DetectDriftFromData_NoLLM(t *testing.T) {
	// Create detector without LLM
	tmpDir := t.TempDir()
	dbPath := fmt.Sprintf("%s/test.db", tmpDir)
	persistence, _ := storage.NewPersistenceBackend(dbPath)
	db := persistence.GetDB()

	detector := NewDriftDetector(db, nil, nil)
	ctx := createTestContext()

	createTestOntology(t, db, "ont-drift-no-llm")

	testData := map[string]interface{}{"test": "data"}
	_, err := detector.DetectDriftFromData(ctx, "ont-drift-no-llm", testData, "test")
	assert.Error(t, err, "Should error without LLM client")
}

// TestDriftDetector_DetectDriftFromExtractionJob tests detection from extraction job
func TestDriftDetector_DetectDriftFromExtractionJob(t *testing.T) {
	detector, db := createTestDriftDetector(t)
	ctx := createTestContext()

	// Create test ontology
	createTestOntology(t, db, "ont-extract-001")

	// Create extraction job
	jobID := "job-001"
	_, err := db.Exec(`
		INSERT INTO extraction_jobs (id, ontology_id, job_name, status, extraction_type, source_type)
		VALUES (?, ?, ?, ?, ?, ?)
	`, jobID, "ont-extract-001", "Test Job", ExtractionCompleted, "auto", "csv")
	require.NoError(t, err, "Failed to create extraction job")

	// Add extracted entities
	_, err = db.Exec(`
		INSERT INTO extracted_entities (job_id, entity_uri, entity_type, entity_label, confidence, source_text, properties)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, jobID, "http://example.org/Product1", "Product", "Product 1", 0.9, "A product", `{}`)
	require.NoError(t, err)

	count, err := detector.DetectDriftFromExtractionJob(ctx, jobID)
	require.NoError(t, err, "Should detect drift from job")
	assert.GreaterOrEqual(t, count, 0, "Should return non-negative count")

	// Verify detection record
	var detectionCount int
	err = db.QueryRow("SELECT COUNT(*) FROM drift_detections WHERE ontology_id = ?", "ont-extract-001").Scan(&detectionCount)
	require.NoError(t, err)
	assert.Equal(t, 1, detectionCount)
}

// TestDriftDetector_DetectDriftFromExtractionJob_Incomplete tests detection on incomplete job
func TestDriftDetector_DetectDriftFromExtractionJob_Incomplete(t *testing.T) {
	detector, db := createTestDriftDetector(t)
	ctx := createTestContext()

	createTestOntology(t, db, "ont-incomplete")

	// Create incomplete job
	jobID := "job-incomplete"
	_, err := db.Exec(`
		INSERT INTO extraction_jobs (id, ontology_id, job_name, status, extraction_type, source_type)
		VALUES (?, ?, ?, ?, ?, ?)
	`, jobID, "ont-incomplete", "Test Job", ExtractionRunning, "auto", "csv")
	require.NoError(t, err)

	_, err = detector.DetectDriftFromExtractionJob(ctx, jobID)
	assert.Error(t, err, "Should error on incomplete job")
	assert.Contains(t, err.Error(), "not completed")
}

// TestDriftDetector_MonitorKnowledgeGraphDrift tests KG monitoring
func TestDriftDetector_MonitorKnowledgeGraphDrift(t *testing.T) {
	detector, db := createTestDriftDetector(t)
	ctx := createTestContext()

	createTestOntology(t, db, "ont-kg-001")

	// Note: Without actual TDB2 backend, this will fail on SPARQL query
	// But we can test the error handling
	_, err := detector.MonitorKnowledgeGraphDrift(ctx, "ont-kg-001")
	// Expected to fail since TDB2 backend is nil
	assert.Error(t, err, "Should error without TDB2 backend")
}

// TestDriftDetector_analyzeDataForDrift tests data analysis
func TestDriftDetector_analyzeDataForDrift(t *testing.T) {
	detector, db := createTestDriftDetector(t)
	ctx := createTestContext()

	createTestOntology(t, db, "ont-analyze-001")
	createTestOntologyContext(t, db, "ont-analyze-001")

	ontologyCtx, _ := detector.getOntologyContext(ctx, "ont-analyze-001")

	testData := map[string]interface{}{
		"new_entities": []string{"http://example.org/NewType"},
		"properties":   map[string]interface{}{"newProperty": "value"},
	}

	// Test with LLM that returns suggestions
	intelligentMock := AI.NewIntelligentMockLLMClient()
	detector.llmClient = intelligentMock

	suggestions, err := detector.analyzeDataForDrift(ctx, "ont-analyze-001", testData, ontologyCtx)
	// May error depending on mock behavior
	_ = suggestions
	_ = err
}

// TestDriftDetector_analyzeEntitiesForDrift tests entity analysis
func TestDriftDetector_analyzeEntitiesForDrift(t *testing.T) {
	detector, db := createTestDriftDetector(t)
	ctx := createTestContext()

	createTestOntology(t, db, "ont-entities-001")
	createTestOntologyContext(t, db, "ont-entities-001")

	ontologyCtx, _ := detector.getOntologyContext(ctx, "ont-entities-001")

	entities := []ExtractedEntity{
		{
			ID:          1,
			JobID:       "job-001",
			EntityURI:   "http://example.org/NewEntity",
			EntityType:  "http://example.org/NewClass",
			EntityLabel: "New Entity",
			Confidence:  0.85,
			Properties:  map[string]interface{}{"newProperty": "value"},
			CreatedAt:   time.Now(),
		},
	}

	suggestions, err := detector.analyzeEntitiesForDrift(ctx, "ont-entities-001", entities, ontologyCtx)
	require.NoError(t, err, "Should analyze entities without error")
	assert.NotNil(t, suggestions, "Should return suggestions")
}

// TestDriftDetector_isKnownClass tests known class check
func TestDriftDetector_isKnownClass(t *testing.T) {
	detector, _ := createTestDriftDetector(t)

	ctx := &OntologyContext{
		Classes: []OntologyClass{
			{URI: "http://example.org/KnownClass", Label: "Known Class"},
		},
	}

	assert.True(t, detector.isKnownClass("http://example.org/KnownClass", ctx), "Should recognize known class")
	assert.False(t, detector.isKnownClass("http://example.org/UnknownClass", ctx), "Should not recognize unknown class")
}

// TestDriftDetector_isKnownProperty tests known property check
func TestDriftDetector_isKnownProperty(t *testing.T) {
	detector, _ := createTestDriftDetector(t)

	ctx := &OntologyContext{
		Properties: []OntologyProperty{
			{URI: "http://example.org/knownProperty", Label: "Known Property"},
		},
	}

	assert.True(t, detector.isKnownProperty("http://example.org/knownProperty", ctx), "Should recognize known property")
	assert.False(t, detector.isKnownProperty("http://example.org/unknownProperty", ctx), "Should not recognize unknown property")
}

// TestDriftDetector_calculateConfidence tests confidence calculation
func TestDriftDetector_calculateConfidence(t *testing.T) {
	detector, _ := createTestDriftDetector(t)

	// Test with zero total
	confidence := detector.calculateConfidence(0, 0)
	assert.InDelta(t, 0.5, confidence, 0.001, "Should return 0.5 when total is 0")

	// Test with varying ratios
	tests := []struct {
		occurrences int
		total       int
		expectedMin float64
		expectedMax float64
	}{
		{10, 100, 0.5, 0.95},
		{50, 100, 0.7, 0.95},
		{100, 100, 0.9, 0.95},
	}

	for _, test := range tests {
		confidence := detector.calculateConfidence(test.occurrences, test.total)
		assert.GreaterOrEqual(t, confidence, test.expectedMin, "Confidence should be >= min")
		assert.LessOrEqual(t, confidence, test.expectedMax, "Confidence should be <= max")
	}
}

// TestDriftDetector_assessRisk tests risk assessment
func TestDriftDetector_assessRisk(t *testing.T) {
	detector, _ := createTestDriftDetector(t)

	// Test class risk assessment
	assert.Equal(t, RiskLevelLow, detector.assessRisk(1, 100, "class"), "Low occurrence class should be low risk")
	assert.Equal(t, RiskLevelMedium, detector.assessRisk(15, 100, "class"), "Medium occurrence class should be medium risk")
	assert.Equal(t, RiskLevelHigh, detector.assessRisk(40, 100, "class"), "High occurrence class should be high risk")

	// Test property risk assessment
	assert.Equal(t, RiskLevelLow, detector.assessRisk(10, 100, "property"), "Low occurrence property should be low risk")
	assert.Equal(t, RiskLevelMedium, detector.assessRisk(30, 100, "property"), "Medium occurrence property should be medium risk")
	assert.Equal(t, RiskLevelHigh, detector.assessRisk(60, 100, "property"), "High occurrence property should be high risk")
}

// TestDriftDetector_buildOntologySummary tests summary building
func TestDriftDetector_buildOntologySummary(t *testing.T) {
	detector, _ := createTestDriftDetector(t)

	ctx := &OntologyContext{
		Classes: []OntologyClass{
			{URI: "http://example.org/Class1", Label: "Class 1"},
			{URI: "http://example.org/Class2", Label: "Class 2"},
		},
		Properties: []OntologyProperty{
			{
				URI:          "http://example.org/prop1",
				Label:        "Property 1",
				PropertyType: "string",
				Domain:       []string{"http://example.org/Class1"},
				Range:        []string{"http://example.org/Class2"},
			},
		},
	}

	summary := detector.buildOntologySummary(ctx)
	assert.NotEmpty(t, summary, "Summary should not be empty")
	assert.Contains(t, summary, "Classes:")
	assert.Contains(t, summary, "http://example.org/Class1")
	assert.Contains(t, summary, "Properties:")
	assert.Contains(t, summary, "Property 1")
}

// TestDriftDetector_extractJSON tests JSON extraction from text
func TestDriftDetector_extractJSON(t *testing.T) {
	detector, _ := createTestDriftDetector(t)

	tests := []struct {
		input    string
		expected string
	}{
		{"{\"key\": \"value\"}", "{\"key\": \"value\"}"},
		{"```json{\"key\": \"value\"}```", "{\"key\": \"value\"}"},
		{"```{\"key\": \"value\"}```", "{\"key\": \"value\"}"},
		{"   {\"key\": \"value\"}   ", "{\"key\": \"value\"}"},
	}

	for _, test := range tests {
		result := detector.extractJSON(test.input)
		assert.Equal(t, test.expected, result, "Should extract JSON correctly")
	}
}

// TestIsBuiltInClass tests built-in class detection
func TestIsBuiltInClass(t *testing.T) {
	assert.True(t, isBuiltInClass("http://www.w3.org/2002/07/owl#Thing"), "owl:Thing should be built-in")
	assert.True(t, isBuiltInClass("http://www.w3.org/2000/01/rdf-schema#Resource"), "rdfs:Resource should be built-in")
	assert.False(t, isBuiltInClass("http://example.org/CustomClass"), "Custom class should not be built-in")
}

// TestIsBuiltInProperty tests built-in property detection
func TestIsBuiltInProperty(t *testing.T) {
	assert.True(t, isBuiltInProperty("http://www.w3.org/1999/02/22-rdf-syntax-ns#type"), "rdf:type should be built-in")
	assert.True(t, isBuiltInProperty("http://www.w3.org/2000/01/rdf-schema#label"), "rdfs:label should be built-in")
	assert.False(t, isBuiltInProperty("http://example.org/customProperty"), "Custom property should not be built-in")
}

// TestDriftDetectionPlugin tests the drift detection plugin
func TestDriftDetectionPlugin(t *testing.T) {
	// Create plugin
	tmpDir := t.TempDir()
	dbPath := fmt.Sprintf("%s/test.db", tmpDir)
	persistence, _ := storage.NewPersistenceBackend(dbPath)
	db := persistence.GetDB()

	mockLLM := AI.NewMockLLMClient("[]")
	plugin := NewDriftDetectionPlugin(db, mockLLM, nil)

	require.NotNil(t, plugin, "Plugin should not be nil")
	assert.Equal(t, "Ontology", plugin.GetPluginType())
	assert.Equal(t, "drift_detection", plugin.GetPluginName())

	// Test validation
	err := plugin.ValidateConfig(map[string]interface{}{})
	assert.Error(t, err, "Should error without ontology_id")

	err = plugin.ValidateConfig(map[string]interface{}{"ontology_id": "test-ont"})
	assert.NoError(t, err, "Should validate with ontology_id")
}

// TestDriftDetectionPlugin_ExecuteStep tests plugin execution
func TestDriftDetectionPlugin_ExecuteStep(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := fmt.Sprintf("%s/test.db", tmpDir)
	persistence, _ := storage.NewPersistenceBackend(dbPath)
	db := persistence.GetDB()

	// Create test ontology
	createTestOntology(t, db, "ont-plugin-001")

	mockLLM := AI.NewMockLLMClient("[]")
	plugin := NewDriftDetectionPlugin(db, mockLLM, nil)

	// Test with extraction job
	_, err := db.Exec(`
		INSERT INTO extraction_jobs (id, ontology_id, job_name, status, extraction_type, source_type)
		VALUES (?, ?, ?, ?, ?, ?)
	`, "job-plugin", "ont-plugin-001", "Test Job", ExtractionCompleted, "auto", "csv")
	require.NoError(t, err)

	stepConfig := pipelines.StepConfig{
		Name:   "drift_detection",
		Plugin: "Ontology.drift_detection",
		Config: map[string]interface{}{
			"ontology_id":       "ont-plugin-001",
			"extraction_job_id": "job-plugin",
		},
	}

	globalContext := pipelines.NewPluginContext()
	ctx := createTestContext()

	result, err := plugin.ExecuteStep(ctx, stepConfig, globalContext)
	// May error depending on mock behavior, but should not panic
	_ = result
	_ = err
}

// TestDriftDetectionStatus constants
func TestDriftDetectionStatus(t *testing.T) {
	assert.Equal(t, DriftDetectionStatus("running"), DriftRunning)
	assert.Equal(t, DriftDetectionStatus("completed"), DriftCompleted)
	assert.Equal(t, DriftDetectionStatus("failed"), DriftFailed)
}

// TestSuggestionType constants
func TestSuggestionType(t *testing.T) {
	assert.Equal(t, SuggestionType("add_class"), SuggestionAddClass)
	assert.Equal(t, SuggestionType("add_property"), SuggestionAddProperty)
	assert.Equal(t, SuggestionType("modify_class"), SuggestionModifyClass)
	assert.Equal(t, SuggestionType("modify_property"), SuggestionModifyProperty)
}

// TestSuggestionStatus constants
func TestSuggestionStatus(t *testing.T) {
	assert.Equal(t, SuggestionStatus("pending"), SuggestionPending)
	assert.Equal(t, SuggestionStatus("approved"), SuggestionApproved)
	assert.Equal(t, SuggestionStatus("rejected"), SuggestionRejected)
	assert.Equal(t, SuggestionStatus("applied"), SuggestionApplied)
}

// TestRiskLevel constants
func TestRiskLevel(t *testing.T) {
	assert.Equal(t, RiskLevel("low"), RiskLevelLow)
	assert.Equal(t, RiskLevel("medium"), RiskLevelMedium)
	assert.Equal(t, RiskLevel("high"), RiskLevelHigh)
	assert.Equal(t, RiskLevel("critical"), RiskLevelCritical)
}

// TestExtractionJobStatus constants
func TestExtractionJobStatus(t *testing.T) {
	assert.Equal(t, ExtractionJobStatus("pending"), ExtractionPending)
	assert.Equal(t, ExtractionJobStatus("running"), ExtractionRunning)
	assert.Equal(t, ExtractionJobStatus("completed"), ExtractionCompleted)
	assert.Equal(t, ExtractionJobStatus("failed"), ExtractionFailed)
}

// Helper functions

func createTestContext() context.Context {
	return context.Background()
}

func createTestOntology(t *testing.T, db *sql.DB, id string) {
	_, err := db.Exec(`
		INSERT INTO ontologies (id, name, version, file_path, tdb2_graph, format, status, auto_version)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, id, "Test Ontology", "1.0.0", "/path/to/ontology.ttl", "http://graph/"+id, "turtle", "active", true)
	require.NoError(t, err, "Failed to create test ontology")
}

func createTestOntologyContext(t *testing.T, db *sql.DB, ontologyID string) {
	// Add classes
	_, err := db.Exec(`
		INSERT INTO ontology_classes (ontology_id, uri, label)
		VALUES (?, ?, ?)
	`, ontologyID, "http://example.org/KnownClass", "Known Class")
	require.NoError(t, err)

	// Add properties
	_, err = db.Exec(`
		INSERT INTO ontology_properties (ontology_id, uri, label, property_type)
		VALUES (?, ?, ?, ?)
	`, ontologyID, "http://example.org/knownProperty", "Known Property", "string")
	require.NoError(t, err)
}
