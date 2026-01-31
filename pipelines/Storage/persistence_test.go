package storage

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPersistenceBackend_DatabaseInitialization tests database initialization with WAL mode and pragmas
func TestPersistenceBackend_DatabaseInitialization(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	backend, err := NewPersistenceBackend(dbPath)
	require.NoError(t, err, "Failed to create persistence backend")
	defer backend.Close()

	// Verify database connection
	ctx := context.Background()
	err = backend.Health(ctx)
	assert.NoError(t, err, "Database health check failed")

	// Verify WAL mode is enabled
	db := backend.GetDB()
	var journalMode string
	err = db.QueryRow("PRAGMA journal_mode").Scan(&journalMode)
	require.NoError(t, err)
	assert.Equal(t, "wal", journalMode, "WAL mode should be enabled")

	// Verify foreign keys are enabled
	var foreignKeys int
	err = db.QueryRow("PRAGMA foreign_keys").Scan(&foreignKeys)
	require.NoError(t, err)
	assert.Equal(t, 1, foreignKeys, "Foreign keys should be enabled")

	// Verify busy timeout
	var busyTimeout int
	err = db.QueryRow("PRAGMA busy_timeout").Scan(&busyTimeout)
	require.NoError(t, err)
	assert.Equal(t, 30000, busyTimeout, "Busy timeout should be 30000ms")

	// Verify synchronous mode
	var syncMode int
	err = db.QueryRow("PRAGMA synchronous").Scan(&syncMode)
	require.NoError(t, err)
	assert.Equal(t, 1, syncMode, "Synchronous mode should be NORMAL (1)")
}

// TestPersistenceBackend_SchemaCreation tests that all tables are created correctly
func TestPersistenceBackend_SchemaCreation(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	backend, err := NewPersistenceBackend(dbPath)
	require.NoError(t, err)
	defer backend.Close()

	db := backend.GetDB()

	// List of expected tables
	expectedTables := []string{
		"ontologies",
		"ontology_classes",
		"ontology_properties",
		"ontology_versions",
		"ontology_changes",
		"ontology_suggestions",
		"extraction_jobs",
		"extracted_entities",
		"drift_detections",
		"auto_update_policies",
		"digital_twins",
		"simulation_scenarios",
		"simulation_runs",
		"twin_state_snapshots",
		"agent_conversations",
		"agent_messages",
		"api_keys",
		"plugins",
		"classifier_models",
		"model_training_runs",
		"model_predictions",
		"anomalies",
		"data_quality_metrics",
		"time_series_data",
		"time_series_analyses",
		"alerts",
		"monitoring_rules",
		"monitoring_jobs",
		"monitoring_job_runs",
		"alert_actions",
		"alert_action_executions",
		"scheduler_jobs",
	}

	// Verify each table exists
	for _, table := range expectedTables {
		var count int
		query := fmt.Sprintf("SELECT count(*) FROM sqlite_master WHERE type='table' AND name='%s'", table)
		err := db.QueryRow(query).Scan(&count)
		require.NoError(t, err, "Failed to check table %s", table)
		assert.Equal(t, 1, count, "Table %s should exist", table)
	}
}

// TestPersistenceBackend_OntologyCRUD tests ontology CRUD operations
func TestPersistenceBackend_OntologyCRUD(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	backend, err := NewPersistenceBackend(dbPath)
	require.NoError(t, err)
	defer backend.Close()

	ctx := context.Background()

	// Create
	ontology := &Ontology{
		ID:          "ont-001",
		Name:        "Test Ontology",
		Description: "Test Description",
		Version:     "1.0.0",
		FilePath:    "/path/to/ontology.ttl",
		TDB2Graph:   "http://example.org/ontology/test",
		Format:      "turtle",
		Status:      "active",
		AutoVersion: true,
		CreatedBy:   "test-user",
		Metadata:    `{"key": "value"}`,
	}

	err = backend.CreateOntology(ctx, ontology)
	require.NoError(t, err, "Failed to create ontology")

	// Read
	retrieved, err := backend.GetOntology(ctx, ontology.ID)
	require.NoError(t, err, "Failed to get ontology")
	assert.Equal(t, ontology.ID, retrieved.ID)
	assert.Equal(t, ontology.Name, retrieved.Name)
	assert.Equal(t, ontology.Description, retrieved.Description)
	assert.Equal(t, ontology.Version, retrieved.Version)
	assert.Equal(t, ontology.FilePath, retrieved.FilePath)
	assert.Equal(t, ontology.TDB2Graph, retrieved.TDB2Graph)
	assert.Equal(t, ontology.Format, retrieved.Format)
	assert.Equal(t, ontology.Status, retrieved.Status)
	assert.Equal(t, ontology.AutoVersion, retrieved.AutoVersion)
	assert.Equal(t, ontology.CreatedBy, retrieved.CreatedBy)
	assert.Equal(t, ontology.Metadata, retrieved.Metadata)
	assert.False(t, retrieved.CreatedAt.IsZero())
	assert.False(t, retrieved.UpdatedAt.IsZero())

	// Update
	ontology.Name = "Updated Ontology"
	ontology.Description = "Updated Description"
	err = backend.UpdateOntology(ctx, ontology)
	require.NoError(t, err, "Failed to update ontology")

	retrieved, err = backend.GetOntology(ctx, ontology.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Ontology", retrieved.Name)
	assert.Equal(t, "Updated Description", retrieved.Description)

	// Update status
	err = backend.UpdateOntologyStatus(ctx, ontology.ID, "inactive")
	require.NoError(t, err, "Failed to update ontology status")

	retrieved, err = backend.GetOntology(ctx, ontology.ID)
	require.NoError(t, err)
	assert.Equal(t, "inactive", retrieved.Status)

	// List
	ontologies, err := backend.ListOntologies(ctx, "")
	require.NoError(t, err, "Failed to list ontologies")
	assert.Len(t, ontologies, 1)

	// List with status filter
	ontologies, err = backend.ListOntologies(ctx, "inactive")
	require.NoError(t, err)
	assert.Len(t, ontologies, 1)

	ontologies, err = backend.ListOntologies(ctx, "active")
	require.NoError(t, err)
	assert.Len(t, ontologies, 0)

	// Delete
	err = backend.DeleteOntology(ctx, ontology.ID)
	require.NoError(t, err, "Failed to delete ontology")

	// Verify deletion
	_, err = backend.GetOntology(ctx, ontology.ID)
	assert.Error(t, err, "Should error when getting deleted ontology")
}

// TestPersistenceBackend_OntologyNotFound tests error handling for non-existent ontology
func TestPersistenceBackend_OntologyNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	backend, err := NewPersistenceBackend(dbPath)
	require.NoError(t, err)
	defer backend.Close()

	ctx := context.Background()

	_, err = backend.GetOntology(ctx, "non-existent-id")
	assert.Error(t, err, "Should error when ontology not found")
	assert.Contains(t, err.Error(), "not found")
}

// TestPersistenceBackend_DigitalTwinCRUD tests digital twin CRUD operations
func TestPersistenceBackend_DigitalTwinCRUD(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	backend, err := NewPersistenceBackend(dbPath)
	require.NoError(t, err)
	defer backend.Close()

	ctx := context.Background()

	// Create
	twinID := "twin-001"
	ontologyID := "ont-001"
	name := "Test Digital Twin"
	description := "Test Description"
	modelType := "supply_chain"
	baseState := `{"status": "active", "capacity": 100}`

	err = backend.CreateDigitalTwin(ctx, twinID, ontologyID, name, description, modelType, baseState)
	require.NoError(t, err, "Failed to create digital twin")

	// Read
	retrieved, err := backend.GetDigitalTwin(ctx, twinID)
	require.NoError(t, err, "Failed to get digital twin")
	assert.Equal(t, twinID, retrieved["id"])
	assert.Equal(t, ontologyID, retrieved["ontology_id"])
	assert.Equal(t, name, retrieved["name"])
	assert.Equal(t, description, retrieved["description"])
	assert.Equal(t, modelType, retrieved["model_type"])
	assert.Equal(t, baseState, retrieved["base_state"])
	assert.NotNil(t, retrieved["created_at"])
	assert.NotNil(t, retrieved["updated_at"])

	// List
	twins, err := backend.ListDigitalTwins(ctx)
	require.NoError(t, err, "Failed to list digital twins")
	assert.Len(t, twins, 1)
}

// TestPersistenceBackend_DigitalTwinNotFound tests error handling for non-existent twin
func TestPersistenceBackend_DigitalTwinNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	backend, err := NewPersistenceBackend(dbPath)
	require.NoError(t, err)
	defer backend.Close()

	ctx := context.Background()

	_, err = backend.GetDigitalTwin(ctx, "non-existent-id")
	assert.Error(t, err, "Should error when twin not found")
	assert.Contains(t, err.Error(), "not found")
}

// TestPersistenceBackend_TransactionHandling tests transaction handling
func TestPersistenceBackend_TransactionHandling(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	backend, err := NewPersistenceBackend(dbPath)
	require.NoError(t, err)
	defer backend.Close()

	db := backend.GetDB()
	ctx := context.Background()

	// Test explicit transaction
	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err, "Failed to begin transaction")

	// Insert within transaction
	_, err = tx.ExecContext(ctx, `
		INSERT INTO ontologies (id, name, description, version, file_path, tdb2_graph, format, status, auto_version)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "tx-test", "TX Test", "Desc", "1.0", "/path", "http://graph", "turtle", "active", true)
	require.NoError(t, err, "Failed to insert within transaction")

	// Commit
	err = tx.Commit()
	require.NoError(t, err, "Failed to commit transaction")

	// Verify data exists
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM ontologies WHERE id = ?", "tx-test").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count, "Data should exist after commit")

	// Test rollback
	tx, err = db.BeginTx(ctx, nil)
	require.NoError(t, err)

	_, err = tx.ExecContext(ctx, `
		INSERT INTO ontologies (id, name, description, version, file_path, tdb2_graph, format, status, auto_version)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "tx-rollback", "TX Rollback", "Desc", "1.0", "/path", "http://graph", "turtle", "active", true)
	require.NoError(t, err)

	// Rollback
	err = tx.Rollback()
	require.NoError(t, err, "Failed to rollback transaction")

	// Verify data does not exist
	err = db.QueryRow("SELECT COUNT(*) FROM ontologies WHERE id = ?", "tx-rollback").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count, "Data should not exist after rollback")
}

// TestPersistenceBackend_ConcurrentAccess tests concurrent database access
func TestPersistenceBackend_ConcurrentAccess(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	backend, err := NewPersistenceBackend(dbPath)
	require.NoError(t, err)
	defer backend.Close()

	ctx := context.Background()

	// Create initial ontology
	ontology := &Ontology{
		ID:        "concurrent-test",
		Name:      "Concurrent Test",
		Version:   "1.0.0",
		FilePath:  "/path",
		TDB2Graph: "http://graph",
		Format:    "turtle",
		Status:    "active",
	}
	err = backend.CreateOntology(ctx, ontology)
	require.NoError(t, err)

	// Run concurrent reads
	var wg sync.WaitGroup
	numGoroutines := 10
	numReads := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numReads; j++ {
				_, err := backend.GetOntology(ctx, ontology.ID)
				if err != nil {
					t.Errorf("Goroutine %d, Read %d: Failed to get ontology: %v", id, j, err)
				}
			}
		}(i)
	}

	wg.Wait()
}

// TestPersistenceBackend_ClassifierModelCRUD tests classifier model CRUD operations
func TestPersistenceBackend_ClassifierModelCRUD(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	backend, err := NewPersistenceBackend(dbPath)
	require.NoError(t, err)
	defer backend.Close()

	ctx := context.Background()

	// Create
	model := &ClassifierModel{
		ID:                "model-001",
		OntologyID:        "ont-001",
		Name:              "Test Model",
		TargetClass:       "Product",
		Algorithm:         "decision_tree",
		Hyperparameters:   `{"max_depth": 10}`,
		FeatureColumns:    `["price", "quantity"]`,
		ClassLabels:       `["A", "B", "C"]`,
		TrainAccuracy:     0.95,
		ValidateAccuracy:  0.92,
		PrecisionScore:    0.91,
		RecallScore:       0.93,
		F1Score:           0.92,
		ConfusionMatrix:   `[[10, 2], [1, 20]]`,
		ModelArtifactPath: "/path/to/model.json",
		ModelSizeBytes:    1024,
		TrainingRows:      1000,
		ValidationRows:    200,
		FeatureImportance: `{"price": 0.6, "quantity": 0.4}`,
		IsActive:          true,
	}

	err = backend.CreateClassifierModel(ctx, model)
	require.NoError(t, err, "Failed to create classifier model")

	// Read
	retrieved, err := backend.GetClassifierModel(ctx, model.ID)
	require.NoError(t, err, "Failed to get classifier model")
	assert.Equal(t, model.ID, retrieved.ID)
	assert.Equal(t, model.OntologyID, retrieved.OntologyID)
	assert.Equal(t, model.Name, retrieved.Name)
	assert.Equal(t, model.TargetClass, retrieved.TargetClass)
	assert.Equal(t, model.Algorithm, retrieved.Algorithm)
	assert.InDelta(t, model.TrainAccuracy, retrieved.TrainAccuracy, 0.001)
	assert.InDelta(t, model.ValidateAccuracy, retrieved.ValidateAccuracy, 0.001)
	assert.Equal(t, model.IsActive, retrieved.IsActive)

	// Update status
	err = backend.UpdateClassifierModelStatus(ctx, model.ID, false)
	require.NoError(t, err, "Failed to update model status")

	retrieved, err = backend.GetClassifierModel(ctx, model.ID)
	require.NoError(t, err)
	assert.False(t, retrieved.IsActive)

	// List
	models, err := backend.ListClassifierModels(ctx, "ont-001", false)
	require.NoError(t, err, "Failed to list classifier models")
	assert.Len(t, models, 1)

	// List active only
	models, err = backend.ListClassifierModels(ctx, "ont-001", true)
	require.NoError(t, err)
	assert.Len(t, models, 0) // Model is now inactive

	// Delete
	err = backend.DeleteClassifierModel(ctx, model.ID)
	require.NoError(t, err, "Failed to delete classifier model")

	// Verify deletion
	_, err = backend.GetClassifierModel(ctx, model.ID)
	assert.Error(t, err, "Should error when getting deleted model")
}

// TestPersistenceBackend_TrainingRunCRUD tests training run CRUD operations
func TestPersistenceBackend_TrainingRunCRUD(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	backend, err := NewPersistenceBackend(dbPath)
	require.NoError(t, err)
	defer backend.Close()

	ctx := context.Background()

	// Create training run
	runID, err := backend.CreateTrainingRun(
		ctx,
		"model-001",
		1000,
		5000,
		0.95,
		0.92,
		`{"accuracy": 0.92}`,
		`{"epochs": 10}`,
		"completed",
		"",
	)
	require.NoError(t, err, "Failed to create training run")
	assert.Greater(t, runID, int64(0), "Run ID should be positive")

	// Update status
	err = backend.UpdateTrainingRunStatus(ctx, runID, "failed", "Out of memory")
	require.NoError(t, err, "Failed to update training run status")
}

// TestPersistenceBackend_PredictionCRUD tests prediction CRUD operations
func TestPersistenceBackend_PredictionCRUD(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	backend, err := NewPersistenceBackend(dbPath)
	require.NoError(t, err)
	defer backend.Close()

	ctx := context.Background()

	// Create prediction
	err = backend.CreatePrediction(
		ctx,
		"model-001",
		`{"price": 100, "quantity": 5}`,
		"ClassA",
		0.85,
		"ClassA",
		true,
		false,
		"",
	)
	require.NoError(t, err, "Failed to create prediction")

	// Create anomaly prediction
	err = backend.CreatePrediction(
		ctx,
		"model-001",
		`{"price": -10, "quantity": 0}`,
		"Unknown",
		0.15,
		"",
		false,
		true,
		"Low confidence outlier",
	)
	require.NoError(t, err, "Failed to create anomaly prediction")
}

// TestPersistenceBackend_AnomalyCRUD tests anomaly CRUD operations
func TestPersistenceBackend_AnomalyCRUD(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	backend, err := NewPersistenceBackend(dbPath)
	require.NoError(t, err)
	defer backend.Close()

	ctx := context.Background()

	predictionID := int64(1)

	// Create anomaly
	anomalyID, err := backend.CreateAnomaly(
		ctx,
		"model-001",
		&predictionID,
		"low_confidence",
		`{"price": -10}`,
		func() *float64 { v := 0.15; return &v }(),
		`["value out of range"]`,
		"high",
		"open",
	)
	require.NoError(t, err, "Failed to create anomaly")
	assert.Greater(t, anomalyID, int64(0), "Anomaly ID should be positive")

	// List anomalies
	anomalies, err := backend.ListAnomalies(ctx, "model-001", "open", "high", 10)
	require.NoError(t, err, "Failed to list anomalies")
	assert.GreaterOrEqual(t, len(anomalies), 1, "Should have at least one anomaly")
}

// TestPersistenceBackend_SaveMLModelDirect tests the SaveMLModelDirect method
func TestPersistenceBackend_SaveMLModelDirect(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	backend, err := NewPersistenceBackend(dbPath)
	require.NoError(t, err)
	defer backend.Close()

	ctx := context.Background()

	modelJSON := `{"tree": {"feature": "price", "threshold": 100}}`
	configJSON := `{"algorithm": "decision_tree", "target_class": "Category", "hyperparameters": "{}", "feature_columns": "[]", "class_labels": "[]"}`
	metricsJSON := `{"train_accuracy": 0.95, "validate_accuracy": 0.92, "precision": 0.91, "recall": 0.93, "f1": 0.92, "training_rows": 1000, "validation_rows": 200}`

	err = backend.SaveMLModelDirect(ctx, "direct-model-001", "ont-001", modelJSON, configJSON, metricsJSON)
	require.NoError(t, err, "Failed to save ML model directly")

	// Verify model was created
	model, err := backend.GetClassifierModel(ctx, "direct-model-001")
	require.NoError(t, err, "Failed to get saved model")
	assert.Equal(t, "direct-model-001", model.ID)
	assert.InDelta(t, 0.95, model.TrainAccuracy, 0.001)
	assert.InDelta(t, 0.92, model.ValidateAccuracy, 0.001)
	assert.Equal(t, 1000, model.TrainingRows)
	assert.Equal(t, 200, model.ValidationRows)
}

// TestPersistenceBackend_Close tests database close operation
func TestPersistenceBackend_Close(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	backend, err := NewPersistenceBackend(dbPath)
	require.NoError(t, err)

	err = backend.Close()
	assert.NoError(t, err, "Failed to close database")

	// Verify database is closed by trying to use it
	db := backend.GetDB()
	ctx := context.Background()
	_, err = db.ExecContext(ctx, "SELECT 1")
	assert.Error(t, err, "Should error when using closed database")
}

// TestPersistenceBackend_InvalidDBPath tests error handling for invalid database path
func TestPersistenceBackend_InvalidDBPath(t *testing.T) {
	// Try to create database in non-existent nested directory without proper permissions
	invalidPath := "/nonexistent/path/that/cannot/be/created/test.db"

	_, err := NewPersistenceBackend(invalidPath)
	assert.Error(t, err, "Should error with invalid database path")
}

// TestPersistenceBackend_ForeignKeyEnforcement tests foreign key constraint enforcement
func TestPersistenceBackend_ForeignKeyEnforcement(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	backend, err := NewPersistenceBackend(dbPath)
	require.NoError(t, err)
	defer backend.Close()

	db := backend.GetDB()
	ctx := context.Background()

	// Try to insert a class with non-existent ontology ID
	_, err = db.ExecContext(ctx, `
		INSERT INTO ontology_classes (ontology_id, uri, label)
		VALUES (?, ?, ?)
	`, "non-existent-ontology", "http://example.org/Class", "Test Class")

	assert.Error(t, err, "Should error when violating foreign key constraint")
	assert.Contains(t, err.Error(), "FOREIGN KEY", "Error should mention foreign key constraint")
}

// TestPersistenceBackend_Migrations tests that migrations run successfully
func TestPersistenceBackend_Migrations(t *testing.T) {
	// Create a database file first
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Create initial database
	backend1, err := NewPersistenceBackend(dbPath)
	require.NoError(t, err)
	backend1.Close()

	// Re-open to trigger migrations
	backend2, err := NewPersistenceBackend(dbPath)
	require.NoError(t, err)
	defer backend2.Close()

	// Verify database is functional after migrations
	ctx := context.Background()
	err = backend2.Health(ctx)
	assert.NoError(t, err, "Database should be healthy after migrations")
}

// BenchmarkPersistenceBackend_CreateOntology benchmarks ontology creation
func BenchmarkPersistenceBackend_CreateOntology(b *testing.B) {
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "bench.db")

	backend, err := NewPersistenceBackend(dbPath)
	require.NoError(b, err)
	defer backend.Close()

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ontology := &Ontology{
			ID:        fmt.Sprintf("bench-ont-%d", i),
			Name:      "Benchmark Ontology",
			Version:   "1.0.0",
			FilePath:  "/path",
			TDB2Graph: "http://graph",
			Format:    "turtle",
			Status:    "active",
		}
		err := backend.CreateOntology(ctx, ontology)
		if err != nil {
			b.Fatalf("Failed to create ontology: %v", err)
		}
	}
}

// BenchmarkPersistenceBackend_GetOntology benchmarks ontology retrieval
func BenchmarkPersistenceBackend_GetOntology(b *testing.B) {
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "bench.db")

	backend, err := NewPersistenceBackend(dbPath)
	require.NoError(b, err)
	defer backend.Close()

	ctx := context.Background()

	// Create ontology
	ontology := &Ontology{
		ID:        "bench-get-ont",
		Name:      "Benchmark Ontology",
		Version:   "1.0.0",
		FilePath:  "/path",
		TDB2Graph: "http://graph",
		Format:    "turtle",
		Status:    "active",
	}
	err = backend.CreateOntology(ctx, ontology)
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := backend.GetOntology(ctx, ontology.ID)
		if err != nil {
			b.Fatalf("Failed to get ontology: %v", err)
		}
	}
}
