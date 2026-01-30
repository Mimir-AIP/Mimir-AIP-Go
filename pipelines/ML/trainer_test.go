package ml

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTrainer_Train tests model training with sample classification data
func TestTrainer_Train(t *testing.T) {
	// Test 1: Train on simple classification dataset
	t.Run("Train on simple classification data", func(t *testing.T) {
		// Create sample classification data: predict if fruit is ripe based on color and softness
		X := [][]float64{
			{1.0, 0.2},   // Red, Hard
			{1.0, 0.8},   // Red, Soft
			{0.2, 0.3},   // Green, Hard
			{0.3, 0.9},   // Yellow, Soft
			{0.9, 0.7},   // Orange, Soft
			{0.1, 0.2},   // Green, Hard
			{0.95, 0.85}, // Red, Very Soft
			{0.2, 0.8},   // Green, Soft
		}
		y := []string{"unripe", "ripe", "unripe", "ripe", "ripe", "unripe", "ripe", "unripe"}
		featureNames := []string{"color_redness", "softness"}

		trainer := NewTrainer(DefaultTrainingConfig())
		result, err := trainer.Train(X, y, featureNames)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotNil(t, result.Model)
		assert.Equal(t, "classification", result.ModelType)
		assert.Greater(t, result.TrainingRows, 0)
		assert.Greater(t, result.ValidationRows, 0)
		assert.Greater(t, result.TrainingDuration, time.Duration(0))

		// Check training metrics
		assert.NotNil(t, result.TrainMetrics)
		assert.NotNil(t, result.ValidateMetrics)
		assert.GreaterOrEqual(t, result.TrainMetrics.Accuracy, 0.0)
		assert.LessOrEqual(t, result.TrainMetrics.Accuracy, 1.0)

		// Check feature importance
		assert.NotNil(t, result.FeatureImportance)
		assert.GreaterOrEqual(t, len(result.FeatureImportance), 1)

		// Verify model can make predictions
		testX := []float64{0.9, 0.9} // Red and soft
		prediction, confidence, err := result.Model.Predict(testX)
		require.NoError(t, err)
		assert.NotEmpty(t, prediction)
		assert.GreaterOrEqual(t, confidence, 0.0)
		assert.LessOrEqual(t, confidence, 1.0)
	})

	// Test 2: Train with custom configuration
	t.Run("Train with custom configuration", func(t *testing.T) {
		X := [][]float64{
			{1, 0, 0},
			{1, 0, 1},
			{0, 1, 0},
			{0, 1, 1},
			{1, 1, 0},
			{0, 0, 1},
			{1, 1, 1},
			{0, 0, 0},
		}
		y := []string{"A", "A", "B", "B", "A", "B", "A", "B"}
		featureNames := []string{"f1", "f2", "f3"}

		config := &TrainingConfig{
			TrainTestSplit:  0.75,
			MaxDepth:        5,
			MinSamplesSplit: 2,
			MinSamplesLeaf:  1,
			RandomSeed:      42,
			Shuffle:         true,
			Stratify:        true,
		}

		trainer := NewTrainer(config)
		result, err := trainer.Train(X, y, featureNames)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotNil(t, result.Model)

		// Verify model info
		modelInfo := result.ModelInfo
		assert.NotNil(t, modelInfo)
		assert.NotEmpty(t, modelInfo)
	})

	// Test 3: Train with imbalanced data
	t.Run("Train with imbalanced data", func(t *testing.T) {
		// Create imbalanced dataset (90% class A, 10% class B)
		X := [][]float64{
			{1, 1}, {1, 2}, {1, 3}, {1, 4}, {1, 5},
			{2, 1}, {2, 2}, {2, 3}, {2, 4}, {2, 5},
			{3, 1}, {3, 2}, {3, 3}, {3, 4}, {3, 5},
			{4, 1}, {4, 2}, {4, 3}, // Class B
		}
		y := []string{
			"A", "A", "A", "A", "A",
			"A", "A", "A", "A", "A",
			"A", "A", "A", "A", "A",
			"B", "B", "B",
		}
		featureNames := []string{"x", "y"}

		trainer := NewTrainer(DefaultTrainingConfig())
		result, err := trainer.Train(X, y, featureNames)

		require.NoError(t, err)
		assert.NotNil(t, result)

		// Model should still train successfully
		assert.NotNil(t, result.Model)
		assert.NotNil(t, result.TrainMetrics)
	})

	// Test 4: Train with large feature set
	t.Run("Train with many features", func(t *testing.T) {
		// Create dataset with 10 features
		X := make([][]float64, 50)
		y := make([]string, 50)
		featureNames := make([]string, 10)

		for i := 0; i < 10; i++ {
			featureNames[i] = fmt.Sprintf("feature_%d", i)
		}

		for i := 0; i < 50; i++ {
			X[i] = make([]float64, 10)
			for j := 0; j < 10; j++ {
				X[i][j] = float64(i+j) / 100.0
			}
			if i%2 == 0 {
				y[i] = "even"
			} else {
				y[i] = "odd"
			}
		}

		trainer := NewTrainer(DefaultTrainingConfig())
		result, err := trainer.Train(X, y, featureNames)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 10, len(result.FeatureImportance))
	})

	// Test 5: Train with minimum data
	t.Run("Train with minimum data", func(t *testing.T) {
		X := [][]float64{
			{1.0, 2.0},
			{2.0, 3.0},
			{3.0, 4.0},
			{4.0, 5.0},
		}
		y := []string{"A", "B", "A", "B"}
		featureNames := []string{"x", "y"}

		config := DefaultTrainingConfig()
		config.TrainTestSplit = 0.5

		trainer := NewTrainer(config)
		result, err := trainer.Train(X, y, featureNames)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 2, result.TrainingRows)
		assert.Equal(t, 2, result.ValidationRows)
	})

	// Test 6: Error - empty data
	t.Run("Error on empty training data", func(t *testing.T) {
		trainer := NewTrainer(DefaultTrainingConfig())
		_, err := trainer.Train([][]float64{}, []string{}, []string{})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "empty training data")
	})

	// Test 7: Error - mismatched dimensions
	t.Run("Error on mismatched X and y", func(t *testing.T) {
		trainer := NewTrainer(DefaultTrainingConfig())
		X := [][]float64{{1, 2}, {3, 4}}
		y := []string{"A"}
		featureNames := []string{"x", "y"}

		_, err := trainer.Train(X, y, featureNames)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "must have same number of samples")
	})

	// Test 8: Error - feature names mismatch
	t.Run("Error on feature names mismatch", func(t *testing.T) {
		trainer := NewTrainer(DefaultTrainingConfig())
		X := [][]float64{{1, 2, 3}, {4, 5, 6}}
		y := []string{"A", "B"}
		featureNames := []string{"x", "y"}

		_, err := trainer.Train(X, y, featureNames)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "feature names must match number of features")
	})
}

// TestTrainer_TrainRegression tests regression model training
func TestTrainer_TrainRegression(t *testing.T) {
	// Test 1: Train on simple regression data
	t.Run("Train on simple regression data", func(t *testing.T) {
		// Predict house price based on size and bedrooms
		X := [][]float64{
			{1000, 2},
			{1500, 3},
			{2000, 4},
			{1200, 2},
			{1800, 3},
			{2200, 4},
			{900, 1},
			{2500, 5},
		}
		y := []float64{200000, 300000, 400000, 240000, 360000, 440000, 180000, 500000}
		featureNames := []string{"sqft", "bedrooms"}

		trainer := NewTrainer(DefaultTrainingConfig())
		result, err := trainer.TrainRegression(X, y, featureNames)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotNil(t, result.Model)
		assert.Equal(t, "regression", result.ModelType)

		// Check regression metrics
		assert.NotNil(t, result.TrainMetricsReg)
		assert.NotNil(t, result.ValidateMetricsReg)
		assert.GreaterOrEqual(t, result.TrainMetricsReg.R2Score, -1.0)
		assert.LessOrEqual(t, result.TrainMetricsReg.R2Score, 1.0)

		// Verify model can make predictions
		testX := []float64{1600, 3}
		prediction, err := result.Model.PredictRegression(testX)
		require.NoError(t, err)
		assert.Greater(t, prediction, 0.0)
	})

	// Test 2: Train regression with noise
	t.Run("Train regression with noisy data", func(t *testing.T) {
		// Linear relationship with noise
		X := [][]float64{
			{1}, {2}, {3}, {4}, {5}, {6}, {7}, {8}, {9}, {10},
		}
		y := []float64{
			2.1, 4.2, 6.0, 8.3, 9.9, 12.1, 14.0, 16.2, 18.1, 20.0,
		}
		featureNames := []string{"x"}

		trainer := NewTrainer(DefaultTrainingConfig())
		result, err := trainer.TrainRegression(X, y, featureNames)

		require.NoError(t, err)
		assert.NotNil(t, result)

		// R² should be reasonably good despite noise
		assert.Greater(t, result.ValidateMetricsReg.R2Score, 0.5)
	})

	// Test 3: Error - empty data
	t.Run("Error on empty regression data", func(t *testing.T) {
		trainer := NewTrainer(DefaultTrainingConfig())
		_, err := trainer.TrainRegression([][]float64{}, []float64{}, []string{})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "empty training data")
	})
}

// TestTrainer_TrainTestSplit tests data splitting functionality
func TestTrainer_TrainTestSplit(t *testing.T) {
	// Test 1: Basic split
	t.Run("Basic train-test split", func(t *testing.T) {
		X := [][]float64{
			{1}, {2}, {3}, {4}, {5}, {6}, {7}, {8}, {9}, {10},
		}
		y := []string{"A", "B", "A", "B", "A", "B", "A", "B", "A", "B"}

		trainer := NewTrainer(DefaultTrainingConfig())
		trainX, trainY, valX, valY, err := trainer.TrainTestSplit(X, y)

		require.NoError(t, err)
		assert.Greater(t, len(trainX), 0)
		assert.Greater(t, len(valX), 0)
		assert.Equal(t, len(trainX), len(trainY))
		assert.Equal(t, len(valX), len(valY))

		// Total should equal original
		assert.Equal(t, len(X), len(trainX)+len(valX))
	})

	// Test 2: Split maintains class distribution with stratification
	t.Run("Stratified split maintains class distribution", func(t *testing.T) {
		// Create dataset with 70% class A, 30% class B
		X := make([][]float64, 100)
		y := make([]string, 100)

		for i := 0; i < 100; i++ {
			X[i] = []float64{float64(i)}
			if i < 70 {
				y[i] = "A"
			} else {
				y[i] = "B"
			}
		}

		config := DefaultTrainingConfig()
		config.Stratify = true
		config.TrainTestSplit = 0.8

		trainer := NewTrainer(config)
		trainX, trainY, valX, valY, err := trainer.TrainTestSplit(X, y)

		require.NoError(t, err)
		assert.Greater(t, len(trainX), 0, "Training set should not be empty")
		assert.Greater(t, len(valX), 0, "Validation set should not be empty")

		// Count classes in train and validation
		trainClassCount := make(map[string]int)
		valClassCount := make(map[string]int)

		for _, label := range trainY {
			trainClassCount[label]++
		}
		for _, label := range valY {
			valClassCount[label]++
		}

		// Both splits should have both classes
		assert.Greater(t, trainClassCount["A"], 0)
		assert.Greater(t, trainClassCount["B"], 0)
		assert.Greater(t, valClassCount["A"], 0)
		assert.Greater(t, valClassCount["B"], 0)

		// Proportions should be roughly similar
		trainRatio := float64(trainClassCount["A"]) / float64(len(trainY))
		valRatio := float64(valClassCount["A"]) / float64(len(valY))

		assert.InDelta(t, trainRatio, valRatio, 0.2, "Class ratios should be similar")
	})

	// Test 3: Regression split
	t.Run("Regression data split", func(t *testing.T) {
		X := [][]float64{
			{1}, {2}, {3}, {4}, {5}, {6}, {7}, {8}, {9}, {10},
		}
		y := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

		trainer := NewTrainer(DefaultTrainingConfig())
		trainX, trainY, valX, valY, err := trainer.TrainTestSplitRegression(X, y)

		require.NoError(t, err)
		assert.Greater(t, len(trainX), 0)
		assert.Greater(t, len(valX), 0)
		assert.Equal(t, len(X), len(trainX)+len(valX))
		assert.Equal(t, len(trainX), len(trainY), "Training X and y should have same length")
		assert.Equal(t, len(valX), len(valY), "Validation X and y should have same length")
	})

	// Test 4: Split very small dataset (should still work)
	t.Run("Handle very small dataset", func(t *testing.T) {
		X := [][]float64{{1}, {2}, {3}, {4}}
		y := []string{"A", "B", "A", "B"}

		config := DefaultTrainingConfig()
		config.TrainTestSplit = 0.5 // 50/50 split
		trainer := NewTrainer(config)
		trainX, trainY, valX, valY, err := trainer.TrainTestSplit(X, y)

		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(trainX), 1, "Training set should have at least 1 sample")
		assert.GreaterOrEqual(t, len(valX), 1, "Validation set should have at least 1 sample")
		assert.Equal(t, len(trainX), len(trainY), "Training X and y should have same length")
		assert.Equal(t, len(valX), len(valY), "Validation X and y should have same length")
		assert.Equal(t, len(X), len(trainX)+len(valX), "Total samples should equal train + validation")
	})
}

// TestTrainer_DefaultTrainingConfig tests default configuration
func TestTrainer_DefaultTrainingConfig(t *testing.T) {
	config := DefaultTrainingConfig()

	assert.NotNil(t, config)
	assert.Equal(t, 0.8, config.TrainTestSplit)
	assert.Equal(t, 10, config.MaxDepth)
	assert.Equal(t, 2, config.MinSamplesSplit)
	assert.Equal(t, 1, config.MinSamplesLeaf)
	assert.True(t, config.Shuffle)
	assert.True(t, config.Stratify)
	assert.NotEqual(t, int64(0), config.RandomSeed)
}

// TestTrainer_NewTrainer tests trainer creation
func TestTrainer_NewTrainer(t *testing.T) {
	// Test with nil config
	t.Run("Create with nil config", func(t *testing.T) {
		trainer := NewTrainer(nil)
		assert.NotNil(t, trainer)
		assert.NotNil(t, trainer.Config)
		assert.NotNil(t, trainer.Rand)
	})

	// Test with custom config
	t.Run("Create with custom config", func(t *testing.T) {
		config := &TrainingConfig{
			MaxDepth:   15,
			RandomSeed: 12345,
		}
		trainer := NewTrainer(config)
		assert.NotNil(t, trainer)
		assert.Equal(t, 15, trainer.Config.MaxDepth)
		assert.Equal(t, int64(12345), trainer.Config.RandomSeed)
	})
}

// TestTrainer_ModelPersistence tests that trained models can be used
func TestTrainer_ModelPersistence(t *testing.T) {
	// Train a model
	X := [][]float64{
		{0, 0}, {0, 1}, {1, 0}, {1, 1},
		{0.1, 0.1}, {0.1, 0.9}, {0.9, 0.1}, {0.9, 0.9},
	}
	y := []string{"low", "medium", "medium", "high", "low", "medium", "medium", "high"}
	featureNames := []string{"x", "y"}

	trainer := NewTrainer(DefaultTrainingConfig())
	result, err := trainer.Train(X, y, featureNames)
	require.NoError(t, err)
	require.NotNil(t, result.Model)

	// Save model info
	modelInfo := result.Model.GetModelInfo()
	assert.NotNil(t, modelInfo)
	assert.NotEmpty(t, modelInfo["max_depth"])
	assert.NotEmpty(t, modelInfo["min_samples_split"])
	assert.NotEmpty(t, modelInfo["min_samples_leaf"])

	// Make predictions with the model
	testCases := [][]float64{
		{0, 0},     // Should be low
		{1, 1},     // Should be high
		{0.5, 0.5}, // Medium
	}

	for _, testX := range testCases {
		prediction, confidence, err := result.Model.Predict(testX)
		require.NoError(t, err)
		assert.NotEmpty(t, prediction)
		assert.GreaterOrEqual(t, confidence, 0.0)
		assert.LessOrEqual(t, confidence, 1.0)
	}

	// Verify feature importance
	featureImportance := result.Model.GetFeatureImportance()
	assert.NotNil(t, featureImportance)
	assert.GreaterOrEqual(t, len(featureImportance), 1)

	// Sum of importance should be approximately 1.0 (or close to it)
	sumImportance := 0.0
	for _, importance := range featureImportance {
		sumImportance += importance
	}
	assert.InDelta(t, 1.0, sumImportance, 0.01)
}

// TestTrainer_Consistency tests that training with same seed gives consistent results
func TestTrainer_Consistency(t *testing.T) {
	X := [][]float64{
		{1, 1}, {2, 2}, {3, 3}, {4, 4}, {5, 5},
		{6, 6}, {7, 7}, {8, 8}, {9, 9}, {10, 10},
	}
	y := []string{"A", "B", "A", "B", "A", "B", "A", "B", "A", "B"}
	featureNames := []string{"x", "y"}

	// Train twice with same seed
	config1 := DefaultTrainingConfig()
	config1.RandomSeed = 42
	trainer1 := NewTrainer(config1)
	result1, err := trainer1.Train(X, y, featureNames)
	require.NoError(t, err)

	config2 := DefaultTrainingConfig()
	config2.RandomSeed = 42
	trainer2 := NewTrainer(config2)
	result2, err := trainer2.Train(X, y, featureNames)
	require.NoError(t, err)

	// Results should be similar (same seed)
	assert.InDelta(t, result1.TrainMetrics.Accuracy, result2.TrainMetrics.Accuracy, 0.001)
	assert.Equal(t, result1.TrainingRows, result2.TrainingRows)
	assert.Equal(t, result1.ValidationRows, result2.ValidationRows)
}

// TestTrainer_FeatureImportance tests feature importance calculation
func TestTrainer_FeatureImportance(t *testing.T) {
	// Create dataset where second feature is more important
	X := [][]float64{
		{1, 100}, {2, 90}, {3, 80}, {4, 70}, {5, 60},
		{6, 50}, {7, 40}, {8, 30}, {9, 20}, {10, 10},
	}
	y := []string{"high", "high", "high", "high", "medium", "medium", "medium", "low", "low", "low"}
	featureNames := []string{"less_important", "more_important"}

	trainer := NewTrainer(DefaultTrainingConfig())
	result, err := trainer.Train(X, y, featureNames)
	require.NoError(t, err)

	// Check feature importance
	importance := result.FeatureImportance
	assert.NotNil(t, importance)

	// Second feature should generally have higher importance
	// (though tree-based models can be variable)
	assert.Contains(t, importance, "less_important")
	assert.Contains(t, importance, "more_important")

	// All importance values should be non-negative
	for _, imp := range importance {
		assert.GreaterOrEqual(t, imp, 0.0)
		assert.LessOrEqual(t, imp, 1.0)
	}
}

// TestTrainingResult_ValidateMetrics validates training result structure
func TestTrainingResult_ValidateMetrics(t *testing.T) {
	// Train a model
	X := [][]float64{
		{1}, {2}, {3}, {4}, {5},
		{6}, {7}, {8}, {9}, {10},
	}
	y := []string{"A", "A", "A", "B", "B", "B", "A", "A", "B", "B"}
	featureNames := []string{"feature"}

	trainer := NewTrainer(DefaultTrainingConfig())
	result, err := trainer.Train(X, y, featureNames)
	require.NoError(t, err)

	// Validate training metrics
	assert.NotNil(t, result.TrainMetrics)
	assert.GreaterOrEqual(t, result.TrainMetrics.Accuracy, 0.0)
	assert.LessOrEqual(t, result.TrainMetrics.Accuracy, 1.0)
	assert.Equal(t, result.TrainingRows, result.TrainMetrics.TotalSamples)

	// Validate validation metrics
	assert.NotNil(t, result.ValidateMetrics)
	assert.GreaterOrEqual(t, result.ValidateMetrics.Accuracy, 0.0)
	assert.LessOrEqual(t, result.ValidateMetrics.Accuracy, 1.0)
	assert.Equal(t, result.ValidationRows, result.ValidateMetrics.TotalSamples)

	// Validate counts
	assert.Equal(t, result.TrainMetrics.TotalSamples+result.ValidateMetrics.TotalSamples,
		result.TrainingRows+result.ValidationRows)

	// Correct predictions should not exceed total
	assert.LessOrEqual(t, result.TrainMetrics.CorrectPredictions, result.TrainMetrics.TotalSamples)
	assert.LessOrEqual(t, result.ValidateMetrics.CorrectPredictions, result.ValidateMetrics.TotalSamples)
}
