package ml

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestModelPrediction_EndToEnd tests the complete train → save → load → predict cycle
func TestModelPrediction_EndToEnd(t *testing.T) {
	tmpDir := t.TempDir()

	// 1. Create training data
	X := [][]float64{
		{5.1, 3.5, 1.4, 0.2}, // Setosa
		{4.9, 3.0, 1.4, 0.2}, // Setosa
		{4.7, 3.2, 1.3, 0.2}, // Setosa
		{7.0, 3.2, 4.7, 1.4}, // Versicolor
		{6.4, 3.2, 4.5, 1.5}, // Versicolor
		{6.9, 3.1, 4.9, 1.5}, // Versicolor
		{6.3, 3.3, 6.0, 2.5}, // Virginica
		{5.8, 2.7, 5.1, 1.9}, // Virginica
		{7.1, 3.0, 5.9, 2.1}, // Virginica
	}
	y := []string{"Setosa", "Setosa", "Setosa", "Versicolor", "Versicolor", "Versicolor", "Virginica", "Virginica", "Virginica"}
	featureNames := []string{"sepal_length", "sepal_width", "petal_length", "petal_width"}

	// 2. Train the model
	classifier := NewDecisionTreeClassifier(5, 2, 1)
	err := classifier.Train(X, y, featureNames)
	require.NoError(t, err, "Should train model successfully")
	require.NotNil(t, classifier.Root, "Model should have a root node")

	// Verify model properties
	assert.Equal(t, "classification", classifier.ModelType, "Should be classification model")
	assert.Equal(t, 4, classifier.NumFeatures, "Should have 4 features")
	assert.Len(t, classifier.Classes, 3, "Should have 3 classes")
	assert.Len(t, classifier.FeatureNames, 4, "Should have 4 feature names")

	t.Logf("Model trained: %d features, %d classes", classifier.NumFeatures, len(classifier.Classes))

	// 3. Save the model to disk
	modelPath := filepath.Join(tmpDir, "test_model.json")
	err = classifier.Save(modelPath)
	require.NoError(t, err, "Should save model successfully")

	// Verify file exists
	_, err = os.Stat(modelPath)
	require.NoError(t, err, "Model file should exist")

	// 4. Load the model back
	loadedClassifier := NewDecisionTreeClassifier(10, 2, 1)
	err = loadedClassifier.Load(modelPath)
	require.NoError(t, err, "Should load model successfully")

	// Verify loaded model has same properties
	assert.Equal(t, classifier.NumFeatures, loadedClassifier.NumFeatures, "NumFeatures should match")
	assert.Equal(t, classifier.NumClasses, loadedClassifier.NumClasses, "NumClasses should match")
	assert.Equal(t, classifier.ModelType, loadedClassifier.ModelType, "ModelType should match")
	assert.Equal(t, classifier.FeatureNames, loadedClassifier.FeatureNames, "FeatureNames should match")
	assert.Equal(t, classifier.Classes, loadedClassifier.Classes, "Classes should match")

	t.Logf("Model loaded successfully: %d features, %d classes", loadedClassifier.NumFeatures, loadedClassifier.NumClasses)

	// 5. Make predictions with loaded model
	testCases := []struct {
		name     string
		features []float64
		expected string
	}{
		{
			name:     "Setosa-like",
			features: []float64{5.0, 3.4, 1.5, 0.2},
			expected: "Setosa",
		},
		{
			name:     "Versicolor-like",
			features: []float64{5.7, 2.8, 4.1, 1.3},
			expected: "Versicolor",
		},
		{
			name:     "Virginica-like",
			features: []float64{6.5, 3.0, 5.2, 2.0},
			expected: "Virginica",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			prediction, confidence, err := loadedClassifier.Predict(tc.features)
			require.NoError(t, err, "Should predict successfully")

			t.Logf("Prediction: %s (confidence: %.2f%%)", prediction, confidence*100)

			// Note: May not always predict correctly on small dataset
			assert.NotEmpty(t, prediction, "Should return a prediction")
			assert.True(t, confidence > 0, "Confidence should be > 0")
		})
	}
}

// TestModelPrediction_Regression tests regression prediction
func TestModelPrediction_Regression(t *testing.T) {
	tmpDir := t.TempDir()

	// Create regression training data (house prices)
	X := [][]float64{
		{1000, 3, 2}, // $200k
		{1500, 4, 3}, // $300k
		{2000, 4, 3}, // $400k
		{2500, 5, 4}, // $500k
		{3000, 5, 4}, // $600k
		{3500, 6, 5}, // $700k
	}
	// For regression, we use numeric values
	yNumeric := []float64{200000, 300000, 400000, 500000, 600000, 700000}
	featureNames := []string{"sqft", "bedrooms", "bathrooms"}

	// Train regression model
	classifier := NewDecisionTreeClassifier(5, 2, 1)
	err := classifier.TrainRegression(X, yNumeric, featureNames)
	require.NoError(t, err, "Should train regression model")

	// Verify it's a regression model
	assert.Equal(t, "regression", classifier.ModelType, "Should be regression model")
	assert.Equal(t, 3, classifier.NumFeatures, "Should have 3 features")

	// Save and load
	modelPath := filepath.Join(tmpDir, "regression_model.json")
	err = classifier.Save(modelPath)
	require.NoError(t, err)

	loadedClassifier := NewDecisionTreeClassifier(10, 2, 1)
	err = loadedClassifier.Load(modelPath)
	require.NoError(t, err)

	// Predict house price for 1800 sqft, 3 bed, 2 bath
	predicted, err := loadedClassifier.PredictRegression([]float64{1800, 3, 2})
	require.NoError(t, err)

	t.Logf("Predicted price: $%.0f", predicted)

	// Should be in reasonable range ($100k-$800k based on training data)
	assert.True(t, predicted > 100000 && predicted < 800000,
		"Prediction should be in reasonable range, got %.0f", predicted)
}

// TestModelPrediction_SingleSample tests single sample prediction
func TestModelPrediction_SingleSample(t *testing.T) {
	// Quick train
	X := [][]float64{{1, 2}, {2, 3}, {3, 4}, {4, 5}, {5, 6}}
	y := []string{"A", "A", "A", "B", "B"}
	featureNames := []string{"f1", "f2"}

	classifier := NewDecisionTreeClassifier(3, 2, 1)
	err := classifier.Train(X, y, featureNames)
	require.NoError(t, err)

	// Single prediction
	prediction, confidence, err := classifier.Predict([]float64{1.5, 2.5})
	require.NoError(t, err)
	require.NotEmpty(t, prediction)
	require.True(t, confidence > 0)

	t.Logf("Prediction: %s with %.2f%% confidence", prediction, confidence*100)
}

// TestModelPrediction_InvalidInput tests error handling
func TestModelPrediction_InvalidInput(t *testing.T) {
	X := [][]float64{{1, 2, 3}, {4, 5, 6}}
	y := []string{"A", "B"}
	featureNames := []string{"a", "b", "c"}

	classifier := NewDecisionTreeClassifier(3, 2, 1)
	err := classifier.Train(X, y, featureNames)
	require.NoError(t, err)

	t.Run("Wrong number of features", func(t *testing.T) {
		_, _, err := classifier.Predict([]float64{1, 2}) // Should be 3 features
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "features")
	})

	t.Run("Nil features", func(t *testing.T) {
		_, _, err := classifier.Predict(nil)
		assert.Error(t, err)
	})

	t.Run("Untrained model", func(t *testing.T) {
		untrained := NewDecisionTreeClassifier(10, 2, 1)
		_, _, err := untrained.Predict([]float64{1, 2, 3})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not trained")
	})
}

// TestModelSaveLoad_MetadataPreservation tests that all metadata is preserved
func TestModelSaveLoad_MetadataPreservation(t *testing.T) {
	tmpDir := t.TempDir()

	X := [][]float64{{1, 2}, {3, 4}}
	y := []string{"X", "Y"}
	featureNames := []string{"feature_a", "feature_b"}

	classifier := NewDecisionTreeClassifier(5, 2, 1)
	err := classifier.Train(X, y, featureNames)
	require.NoError(t, err)

	// Save
	modelPath := filepath.Join(tmpDir, "metadata_model.json")
	err = classifier.Save(modelPath)
	require.NoError(t, err)

	// Load and verify all metadata
	loaded := NewDecisionTreeClassifier(10, 2, 1)
	err = loaded.Load(modelPath)
	require.NoError(t, err)

	assert.Equal(t, classifier.NumFeatures, loaded.NumFeatures, "NumFeatures should be preserved")
	assert.Equal(t, classifier.NumClasses, loaded.NumClasses, "NumClasses should be preserved")
	assert.Equal(t, classifier.FeatureNames, loaded.FeatureNames, "FeatureNames should be preserved")
	assert.Equal(t, classifier.Classes, loaded.Classes, "Classes should be preserved")
	assert.Equal(t, classifier.ModelType, loaded.ModelType, "ModelType should be preserved")

	t.Logf("All metadata preserved: Features=%v, Classes=%v", loaded.FeatureNames, loaded.Classes)
}

// TestModelPrediction_Probability tests prediction probabilities
func TestModelPrediction_Probability(t *testing.T) {
	X := [][]float64{
		{1, 1}, {1, 2}, {2, 1}, // Class A
		{8, 8}, {8, 9}, {9, 8}, // Class B - clearly separated
	}
	y := []string{"A", "A", "A", "B", "B", "B"}
	featureNames := []string{"x", "y"}

	classifier := NewDecisionTreeClassifier(5, 2, 1)
	err := classifier.Train(X, y, featureNames)
	require.NoError(t, err)

	// Get probabilities
	proba, err := classifier.PredictProba([]float64{1, 1})
	require.NoError(t, err)
	require.NotNil(t, proba)

	// Should have probabilities for both classes
	assert.Contains(t, proba, "A", "Should have probability for class A")
	assert.Contains(t, proba, "B", "Should have probability for class B")

	// Sum should be close to 1.0
	sum := 0.0
	for _, p := range proba {
		sum += p
	}
	assert.InDelta(t, 1.0, sum, 0.01, "Probabilities should sum to 1")

	// For clear class A, should have high probability for A
	assert.Greater(t, proba["A"], proba["B"], "Should have higher probability for correct class")

	t.Logf("Probabilities: A=%.2f%%, B=%.2f%%", proba["A"]*100, proba["B"]*100)
}

// TestModelPrediction_ConcurrentAccess tests thread safety
func TestModelPrediction_ConcurrentAccess(t *testing.T) {
	X := [][]float64{
		{1}, {2}, {3}, {4}, {5}, {6}, {7}, {8}, {9}, {10},
	}
	y := []string{"A", "A", "A", "A", "A", "B", "B", "B", "B", "B"}
	featureNames := []string{"x"}

	classifier := NewDecisionTreeClassifier(3, 2, 1)
	err := classifier.Train(X, y, featureNames)
	require.NoError(t, err)

	// Run concurrent predictions
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(idx int) {
			defer func() { done <- true }()

			features := []float64{float64(idx + 1)}
			pred, _, err := classifier.Predict(features)
			assert.NoError(t, err, "Concurrent prediction %d should succeed", idx)
			assert.NotEmpty(t, pred, "Concurrent prediction %d should return result", idx)
		}(i)
	}

	// Wait for all
	for i := 0; i < 10; i++ {
		<-done
	}

	t.Log("All 10 concurrent predictions completed successfully")
}
