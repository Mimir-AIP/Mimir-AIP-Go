package ml

import (
	"context"
	"fmt"
	"testing"
	"time"

	storage "github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/Storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestAutoTrainer creates a test auto trainer with in-memory storage
func createTestAutoTrainer(t *testing.T) *AutoTrainer {
	// Create temporary database
	tmpDir := t.TempDir()
	dbPath := fmt.Sprintf("%s/test.db", tmpDir)

	persistence, err := storage.NewPersistenceBackend(dbPath)
	require.NoError(t, err, "Failed to create persistence backend")

	// Create auto trainer without KG client (will be nil for tests)
	at := NewAutoTrainer(persistence, nil)
	require.NotNil(t, at, "AutoTrainer should not be nil")

	return at
}

// TestNewAutoTrainer tests auto trainer creation
func TestNewAutoTrainer(t *testing.T) {
	at := createTestAutoTrainer(t)

	assert.NotNil(t, at.Storage, "Storage should be initialized")
	assert.NotNil(t, at.Analyzer, "Analyzer should be initialized")
	assert.NotNil(t, at.Extractor, "Extractor should be initialized")
}

// TestAutoTrainOptions_Default tests default options
func TestAutoTrainOptions_Default(t *testing.T) {
	options := DefaultAutoTrainOptions()

	assert.NotNil(t, options, "Default options should not be nil")
	assert.True(t, options.EnableRegression, "Regression should be enabled by default")
	assert.True(t, options.EnableClassification, "Classification should be enabled by default")
	assert.True(t, options.EnableMonitoring, "Monitoring should be enabled by default")
	assert.InDelta(t, 0.6, options.MinConfidence, 0.001, "Min confidence should be 0.6")
	assert.False(t, options.ForceAll, "ForceAll should be false by default")
	assert.Equal(t, 10, options.MaxModels, "Max models should be 10")
}

// TestAutoTrainer_detectTargetsFromDataset tests target detection
func TestAutoTrainer_detectTargetsFromDataset(t *testing.T) {
	at := createTestAutoTrainer(t)
	ctx := context.Background()

	// Create test dataset
	dataset := createTestDataset()

	targets, err := at.detectTargetsFromDataset(ctx, "ont-001", dataset)
	require.NoError(t, err, "Should detect targets without error")
	assert.NotNil(t, targets, "Targets should not be nil")

	// Should find at least one target from the numeric columns
	foundRegression := false

	for _, target := range targets {
		assert.NotEmpty(t, target.ColumnName, "Target should have column name")
		assert.NotEmpty(t, target.ModelType, "Target should have model type")
		assert.Greater(t, target.Confidence, 0.0, "Confidence should be > 0")
		assert.Greater(t, target.SampleSize, 0, "Sample size should be > 0")

		if target.ModelType == "regression" {
			foundRegression = true
		}
	}

	// Dataset has numeric columns suitable for regression
	assert.True(t, foundRegression, "Should find regression targets")
}

// TestAutoTrainer_prepareTrainingDataFromDataset tests data preparation
func TestAutoTrainer_prepareTrainingDataFromDataset(t *testing.T) {
	at := createTestAutoTrainer(t)

	dataset := createTestDataset()

	// Test with a numeric column as target
	target := DatasetMLTarget{
		ColumnName:   "price",
		ModelType:    "regression",
		Confidence:   0.8,
		FeatureCount: 3,
		SampleSize:   5,
	}

	trainingData, err := at.prepareTrainingDataFromDataset(dataset, target)
	require.NoError(t, err, "Should prepare training data without error")
	require.NotNil(t, trainingData, "Training data should not be nil")

	// Verify structure
	assert.NotNil(t, trainingData.X, "Features matrix should exist")
	assert.NotNil(t, trainingData.Y, "Target vector should exist")
	assert.NotEmpty(t, trainingData.FeatureNames, "Should have feature names")

	// Verify dimensions
	if len(trainingData.X) > 0 {
		assert.Equal(t, len(trainingData.X), len(trainingData.Y.([]float64)),
			"Features and targets should have same length")
	}
}

// TestAutoTrainer_prepareTrainingDataFromDataset_Classification tests classification data preparation
func TestAutoTrainer_prepareTrainingDataFromDataset_Classification(t *testing.T) {
	at := createTestAutoTrainer(t)

	// Create dataset with categorical target
	dataset := &UnifiedDataset{
		RowCount:    5,
		ColumnCount: 3,
		Columns: []ColumnMetadata{
			{Name: "feature1", Index: 0, IsNumeric: true},
			{Name: "feature2", Index: 1, IsNumeric: true},
			{Name: "category", Index: 2, IsNumeric: false, UniqueCount: 3},
		},
		Rows: []map[string]interface{}{
			{"feature1": 1.0, "feature2": 2.0, "category": "A"},
			{"feature1": 2.0, "feature2": 3.0, "category": "B"},
			{"feature1": 3.0, "feature2": 4.0, "category": "A"},
			{"feature1": 4.0, "feature2": 5.0, "category": "C"},
			{"feature1": 5.0, "feature2": 6.0, "category": "B"},
		},
	}

	target := DatasetMLTarget{
		ColumnName:   "category",
		ModelType:    "classification",
		Confidence:   0.8,
		FeatureCount: 2,
		SampleSize:   5,
	}

	trainingData, err := at.prepareTrainingDataFromDataset(dataset, target)
	require.NoError(t, err, "Should prepare classification data without error")
	require.NotNil(t, trainingData, "Training data should not be nil")

	// Verify classification target type
	_, ok := trainingData.Y.([]string)
	assert.True(t, ok, "Classification target should be []string")
}

// TestAutoTrainer_sanitizeModelID tests model ID sanitization
func TestAutoTrainer_sanitizeModelID(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Price", "price"},
		{"Product Name", "product_name"},
		{"Sales-Volume", "sales_volume"},
		{"Revenue.2023", "revenue_2023"},
		{"Category/Type", "category_type"},
		{"UPPER_CASE", "upper_case"},
	}

	for _, test := range tests {
		result := sanitizeModelID(test.input)
		assert.Equal(t, test.expected, result, "Sanitizing '%s' should give '%s'", test.input, test.expected)
	}
}

// TestAutoTrainingResult_Structure tests result structure
func TestAutoTrainingResult_Structure(t *testing.T) {
	result := &AutoTrainingResult{
		OntologyID:            "ont-001",
		ModelsCreated:         2,
		ModelsFailed:          1,
		MonitoringJobsCreated: 1,
		RulesCreated:          3,
		TrainedModels: []TrainedModelInfo{
			{
				ModelID:        "model-001",
				TargetProperty: "price",
				ModelType:      "regression",
				R2Score:        0.85,
				SampleCount:    100,
				FeatureCount:   5,
				TrainingTime:   500 * time.Millisecond,
				Confidence:     0.8,
				Reasoning:      "High correlation",
			},
		},
		FailedModels: []FailedModelInfo{
			{
				TargetProperty: "invalid_column",
				ModelType:      "classification",
				ErrorMessage:   "Insufficient data",
				Confidence:     0.3,
			},
		},
		MonitoringSetup: &MonitoringSetupInfo{
			JobID:        "monitor-001",
			MetricsCount: 2,
			RulesCreated: []string{"rule1", "rule2"},
			CronSchedule: "*/15 * * * *",
		},
		TotalDuration: 2 * time.Second,
		Summary:       "Successfully trained 2 models, 1 models failed, setup monitoring with 3 rules in 2s",
	}

	assert.Equal(t, "ont-001", result.OntologyID)
	assert.Equal(t, 2, result.ModelsCreated)
	assert.Equal(t, 1, result.ModelsFailed)
	assert.Len(t, result.TrainedModels, 1)
	assert.Len(t, result.FailedModels, 1)
	assert.NotNil(t, result.MonitoringSetup)
	assert.Greater(t, result.TotalDuration, time.Duration(0))
	assert.NotEmpty(t, result.Summary)
}

// TestTrainedModelInfo_Structure tests trained model info structure
func TestTrainedModelInfo_Structure(t *testing.T) {
	modelInfo := TrainedModelInfo{
		ModelID:        "auto_ont-001_price_1234567890",
		TargetProperty: "price",
		ModelType:      "regression",
		R2Score:        0.92,
		RMSE:           5.5,
		SampleCount:    500,
		FeatureCount:   8,
		TrainingTime:   1200 * time.Millisecond,
		Confidence:     0.85,
		Reasoning:      "Strong linear relationship with features",
	}

	assert.Equal(t, "price", modelInfo.TargetProperty)
	assert.Equal(t, "regression", modelInfo.ModelType)
	assert.InDelta(t, 0.92, modelInfo.R2Score, 0.001)
	assert.Equal(t, 500, modelInfo.SampleCount)
	assert.Equal(t, 8, modelInfo.FeatureCount)
}

// TestFailedModelInfo_Structure tests failed model info structure
func TestFailedModelInfo_Structure(t *testing.T) {
	failedInfo := FailedModelInfo{
		TargetProperty: "unknown_column",
		ModelType:      "classification",
		ErrorMessage:   "Column not found in dataset",
		Confidence:     0.0,
	}

	assert.Equal(t, "unknown_column", failedInfo.TargetProperty)
	assert.Equal(t, "classification", failedInfo.ModelType)
	assert.NotEmpty(t, failedInfo.ErrorMessage)
}

// TestDatasetMLTarget_Structure tests dataset ML target structure
func TestDatasetMLTarget_Structure(t *testing.T) {
	target := DatasetMLTarget{
		ColumnName:   "revenue",
		ModelType:    "regression",
		Confidence:   0.9,
		FeatureCount: 10,
		SampleSize:   1000,
	}

	assert.Equal(t, "revenue", target.ColumnName)
	assert.Equal(t, "regression", target.ModelType)
	assert.InDelta(t, 0.9, target.Confidence, 0.001)
	assert.Equal(t, 10, target.FeatureCount)
	assert.Equal(t, 1000, target.SampleSize)
}

// TestMonitoringSetupInfo_Structure tests monitoring setup info structure
func TestMonitoringSetupInfo_Structure(t *testing.T) {
	monitoringInfo := MonitoringSetupInfo{
		JobID:        "monitor_ont-001_1234567890",
		MetricsCount: 3,
		RulesCreated: []string{"threshold_price", "anomaly_detection", "trend_forecast"},
		CronSchedule: "*/15 * * * *",
	}

	assert.NotEmpty(t, monitoringInfo.JobID)
	assert.Equal(t, 3, monitoringInfo.MetricsCount)
	assert.Len(t, monitoringInfo.RulesCreated, 3)
	assert.Equal(t, "*/15 * * * *", monitoringInfo.CronSchedule)
}

// TestAutoTrainer_generateResultSummary tests summary generation
func TestAutoTrainer_generateResultSummary(t *testing.T) {
	at := createTestAutoTrainer(t)

	// Test with successful models
	result1 := &AutoTrainingResult{
		ModelsCreated:         3,
		ModelsFailed:          0,
		MonitoringJobsCreated: 1,
		RulesCreated:          2,
		TotalDuration:         5 * time.Second,
	}
	summary1 := at.generateResultSummary(result1)
	assert.NotEmpty(t, summary1)
	assert.Contains(t, summary1, "3 models")
	assert.Contains(t, summary1, "monitoring with 2 rules")

	// Test with failures
	result2 := &AutoTrainingResult{
		ModelsCreated: 1,
		ModelsFailed:  2,
		TotalDuration: 3 * time.Second,
	}
	summary2 := at.generateResultSummary(result2)
	assert.Contains(t, summary2, "1 models")
	assert.Contains(t, summary2, "2 models failed")

	// Test with no actions
	result3 := &AutoTrainingResult{
		TotalDuration: 1 * time.Second,
	}
	summary3 := at.generateResultSummary(result3)
	assert.Equal(t, "No actions taken", summary3)
}

// TestAutoTrainer_parseGoalToOptions tests goal parsing
func TestAutoTrainer_parseGoalToOptions(t *testing.T) {
	at := createTestAutoTrainer(t)

	capabilities := &MLCapabilities{} // Empty capabilities for testing

	tests := []struct {
		goal                 string
		expectRegression     bool
		expectClassification bool
		expectMonitoring     bool
		expectForceAll       bool
	}{
		{"Predict price", true, false, false, false},
		{"Predict cost using regression", true, false, false, false},
		{"Classify by category", false, true, false, false},
		{"Classify type", false, true, false, false},
		{"Monitor metrics and alert", false, false, true, false},
		{"Watch for anomalies", false, false, true, false},
		{"Train all models", true, true, false, true},
		{"Do everything", true, true, false, true},
		{"Forecast revenue", true, false, false, false},
	}

	for _, test := range tests {
		options := at.parseGoalToOptions(test.goal, capabilities)
		assert.Equal(t, test.expectRegression, options.EnableRegression,
			"Goal '%s': regression should be %v", test.goal, test.expectRegression)
		assert.Equal(t, test.expectClassification, options.EnableClassification,
			"Goal '%s': classification should be %v", test.goal, test.expectClassification)
		assert.Equal(t, test.expectMonitoring, options.EnableMonitoring,
			"Goal '%s': monitoring should be %v", test.goal, test.expectMonitoring)
		assert.Equal(t, test.expectForceAll, options.ForceAll,
			"Goal '%s': forceAll should be %v", test.goal, test.expectForceAll)
	}
}

// TestAutoTrainer_parseGoalToOptions_PricePrediction tests price prediction parsing
func TestAutoTrainer_parseGoalToOptions_PricePrediction(t *testing.T) {
	at := createTestAutoTrainer(t)
	capabilities := &MLCapabilities{}

	options := at.parseGoalToOptions("predict price", capabilities)
	assert.True(t, options.EnableRegression, "Price prediction should enable regression")
	assert.False(t, options.EnableClassification, "Price prediction should disable classification")
}

// TestAutoTrainer_parseGoalToOptions_CategoryClassification tests category classification parsing
func TestAutoTrainer_parseGoalToOptions_CategoryClassification(t *testing.T) {
	at := createTestAutoTrainer(t)
	capabilities := &MLCapabilities{}

	options := at.parseGoalToOptions("classify by category", capabilities)
	assert.False(t, options.EnableRegression, "Category classification should disable regression")
	assert.True(t, options.EnableClassification, "Category classification should enable classification")
}

// TestAutoTrainer_parseGoalToOptions_Monitoring tests monitoring parsing
func TestAutoTrainer_parseGoalToOptions_Monitoring(t *testing.T) {
	at := createTestAutoTrainer(t)
	capabilities := &MLCapabilities{}

	options := at.parseGoalToOptions("monitor the system and send alerts", capabilities)
	assert.True(t, options.EnableMonitoring, "Monitoring goal should enable monitoring")
}

// TestAutoTrainer_parseGoalToOptions_ForceAll tests force all parsing
func TestAutoTrainer_parseGoalToOptions_ForceAll(t *testing.T) {
	at := createTestAutoTrainer(t)
	capabilities := &MLCapabilities{}

	options := at.parseGoalToOptions("train all possible models", capabilities)
	assert.True(t, options.ForceAll, "Force all goal should set ForceAll")
	assert.Equal(t, 0.0, options.MinConfidence, "Force all should set min confidence to 0")
}

// Helper function to create test dataset
func createTestDataset() *UnifiedDataset {
	return &UnifiedDataset{
		RowCount:    5,
		ColumnCount: 4,
		Columns: []ColumnMetadata{
			{
				Name:        "id",
				Index:       0,
				IsNumeric:   true,
				UniqueCount: 5,
			},
			{
				Name:      "price",
				Index:     1,
				IsNumeric: true,
				Stats: &ColumnStats{
					Min:  10.0,
					Max:  50.0,
					Mean: 30.0,
				},
				UniqueCount: 5,
			},
			{
				Name:      "quantity",
				Index:     2,
				IsNumeric: true,
				Stats: &ColumnStats{
					Min:  1.0,
					Max:  100.0,
					Mean: 50.0,
				},
				UniqueCount: 5,
			},
			{
				Name:        "category",
				Index:       3,
				IsNumeric:   false,
				UniqueCount: 3,
			},
		},
		Rows: []map[string]interface{}{
			{"id": 1, "price": 10.0, "quantity": 50, "category": "A"},
			{"id": 2, "price": 20.0, "quantity": 30, "category": "B"},
			{"id": 3, "price": 30.0, "quantity": 80, "category": "A"},
			{"id": 4, "price": 40.0, "quantity": 20, "category": "C"},
			{"id": 5, "price": 50.0, "quantity": 100, "category": "B"},
		},
	}
}
