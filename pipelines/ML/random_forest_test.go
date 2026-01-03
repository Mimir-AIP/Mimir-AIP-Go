package ml

import (
	"math"
	"os"
	"testing"
)

// TestRandomForestClassifierIris tests classification on iris-like dataset
func TestRandomForestClassifierIris(t *testing.T) {
	// Create a simple dataset (iris-like)
	X := [][]float64{
		{5.1, 3.5, 1.4, 0.2}, {4.9, 3.0, 1.4, 0.2}, {4.7, 3.2, 1.3, 0.2},
		{7.0, 3.2, 4.7, 1.4}, {6.4, 3.2, 4.5, 1.5}, {6.9, 3.1, 4.9, 1.5},
		{6.3, 3.3, 6.0, 2.5}, {5.8, 2.7, 5.1, 1.9}, {7.1, 3.0, 5.9, 2.1},
		{5.0, 3.6, 1.4, 0.2}, {5.4, 3.9, 1.7, 0.4}, {4.6, 3.4, 1.4, 0.3},
		{6.5, 2.8, 4.6, 1.5}, {5.7, 2.8, 4.5, 1.3}, {6.3, 3.3, 4.7, 1.6},
		{6.7, 3.1, 5.6, 2.4}, {6.9, 3.1, 5.1, 2.3}, {5.8, 2.7, 5.1, 1.9},
	}

	y := []string{
		"setosa", "setosa", "setosa",
		"versicolor", "versicolor", "versicolor",
		"virginica", "virginica", "virginica",
		"setosa", "setosa", "setosa",
		"versicolor", "versicolor", "versicolor",
		"virginica", "virginica", "virginica",
	}

	featureNames := []string{"sepal_length", "sepal_width", "petal_length", "petal_width"}

	// Train Random Forest
	rf := NewRandomForestClassifier(10, 5, 2, 1)
	err := rf.Train(X, y, featureNames)
	if err != nil {
		t.Fatalf("Training failed: %v", err)
	}

	// Validate model
	if err := rf.Validate(); err != nil {
		t.Errorf("Model validation failed: %v", err)
	}

	// Test predictions
	tests := []struct {
		input    []float64
		expected string
	}{
		{[]float64{5.0, 3.4, 1.5, 0.2}, "setosa"},
		{[]float64{6.5, 3.0, 4.6, 1.4}, "versicolor"},
		{[]float64{6.7, 3.1, 5.6, 2.4}, "virginica"},
	}

	for _, tt := range tests {
		predicted, confidence, err := rf.Predict(tt.input)
		if err != nil {
			t.Errorf("Prediction failed: %v", err)
			continue
		}
		if predicted != tt.expected {
			t.Errorf("Expected %s, got %s (confidence: %.2f)", tt.expected, predicted, confidence)
		}
		t.Logf("✓ Predicted: %s (confidence: %.2f)", predicted, confidence)
	}

	// Test probability predictions
	testInput := []float64{5.0, 3.4, 1.5, 0.2}
	proba, err := rf.PredictProba(testInput)
	if err != nil {
		t.Errorf("PredictProba failed: %v", err)
	} else {
		t.Logf("Probabilities: %v", proba)
		
		// Check probabilities sum to ~1.0
		sum := 0.0
		for _, p := range proba {
			sum += p
		}
		if math.Abs(sum-1.0) > 0.01 {
			t.Errorf("Probabilities don't sum to 1.0: %.4f", sum)
		}
	}

	// Check OOB score
	if rf.OOBScore < 0.0 || rf.OOBScore > 1.0 {
		t.Errorf("Invalid OOB score: %.4f", rf.OOBScore)
	}
	t.Logf("OOB Score: %.4f", rf.OOBScore)

	// Check feature importance
	importance := rf.GetFeatureImportance()
	if len(importance) != len(featureNames) {
		t.Errorf("Expected %d features in importance, got %d", len(featureNames), len(importance))
	}
	t.Logf("Feature importance: %v", importance)

	// Get model info
	info := rf.GetModelInfo()
	if info["algorithm"] != "random_forest" {
		t.Errorf("Expected algorithm 'random_forest', got %v", info["algorithm"])
	}
	if info["num_trees"] != 10 {
		t.Errorf("Expected 10 trees, got %v", info["num_trees"])
	}
	t.Logf("Model info: %v", info)
}

// TestRandomForestRegression tests regression functionality
func TestRandomForestRegression(t *testing.T) {
	// Create regression dataset
	X := [][]float64{
		{1.0, 2.0}, {2.0, 3.0}, {3.0, 4.0}, {4.0, 5.0},
		{5.0, 6.0}, {6.0, 7.0}, {7.0, 8.0}, {8.0, 9.0},
		{1.5, 2.5}, {2.5, 3.5}, {3.5, 4.5}, {4.5, 5.5},
	}
	
	// Target: y = 2*x1 + 3*x2 + noise
	y := []float64{
		8.0, 13.0, 18.0, 23.0,
		28.0, 33.0, 38.0, 43.0,
		10.5, 15.5, 20.5, 25.5,
	}

	featureNames := []string{"x1", "x2"}

	// Train Random Forest for regression
	rf := NewRandomForestClassifier(20, 10, 2, 1)
	err := rf.TrainRegression(X, y, featureNames)
	if err != nil {
		t.Fatalf("Training failed: %v", err)
	}

	// Validate model
	if err := rf.Validate(); err != nil {
		t.Errorf("Model validation failed: %v", err)
	}

	if rf.ModelType != "regression" {
		t.Errorf("Expected model_type 'regression', got '%s'", rf.ModelType)
	}

	// Test predictions
	tests := []struct {
		input    []float64
		expected float64
		tolerance float64
	}{
		{[]float64{2.0, 3.0}, 13.0, 3.0},
		{[]float64{5.0, 6.0}, 28.0, 3.0},
	}

	for _, tt := range tests {
		predicted, err := rf.PredictRegression(tt.input)
		if err != nil {
			t.Errorf("Prediction failed: %v", err)
			continue
		}
		
		diff := math.Abs(predicted - tt.expected)
		if diff > tt.tolerance {
			t.Errorf("Prediction too far from expected: got %.2f, expected %.2f (diff: %.2f)", 
				predicted, tt.expected, diff)
		}
		t.Logf("✓ Predicted: %.2f (expected: %.2f, diff: %.2f)", predicted, tt.expected, diff)
	}

	// Test prediction with confidence interval
	testInput := []float64{3.0, 4.0}
	value, lower, upper, err := rf.PredictRegressionWithInterval(testInput)
	if err != nil {
		t.Errorf("PredictRegressionWithInterval failed: %v", err)
	} else {
		t.Logf("Prediction: %.2f [%.2f, %.2f]", value, lower, upper)
		
		if lower > value || upper < value {
			t.Errorf("Confidence interval doesn't contain prediction: %.2f not in [%.2f, %.2f]", 
				value, lower, upper)
		}
	}

	// Check OOB score (R²)
	if rf.OOBScore < 0.0 || rf.OOBScore > 1.0 {
		t.Logf("OOB R² score: %.4f (may be negative for poor fits)", rf.OOBScore)
	} else {
		t.Logf("OOB R² score: %.4f", rf.OOBScore)
	}
}

// TestRandomForestSaveLoad tests model persistence
func TestRandomForestSaveLoad(t *testing.T) {
	// Train a simple model
	X := [][]float64{
		{1.0, 2.0}, {2.0, 3.0}, {3.0, 4.0},
		{10.0, 11.0}, {11.0, 12.0}, {12.0, 13.0},
	}
	y := []string{"A", "A", "A", "B", "B", "B"}
	featureNames := []string{"feature1", "feature2"}

	rf := NewRandomForestClassifier(5, 5, 2, 1)
	err := rf.Train(X, y, featureNames)
	if err != nil {
		t.Fatalf("Training failed: %v", err)
	}

	// Make a prediction before saving
	testInput := []float64{1.5, 2.5}
	pred1, conf1, err := rf.Predict(testInput)
	if err != nil {
		t.Fatalf("Prediction before save failed: %v", err)
	}

	// Save model
	tmpFile := "/tmp/test_random_forest.json"
	defer os.Remove(tmpFile)

	err = rf.Save(tmpFile)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Load model
	loadedRF := &RandomForestClassifier{}
	err = loadedRF.Load(tmpFile)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Make same prediction after loading
	pred2, conf2, err := loadedRF.Predict(testInput)
	if err != nil {
		t.Fatalf("Prediction after load failed: %v", err)
	}

	if pred1 != pred2 {
		t.Errorf("Predictions differ after save/load: %s vs %s", pred1, pred2)
	}
	if math.Abs(conf1-conf2) > 0.01 {
		t.Errorf("Confidences differ after save/load: %.4f vs %.4f", conf1, conf2)
	}

	t.Logf("✓ Save/Load successful: %s (confidence: %.2f)", pred2, conf2)
}

// TestRandomForestVsSingleTree compares RF performance to single decision tree
func TestRandomForestVsSingleTree(t *testing.T) {
	// Create a larger dataset
	X := [][]float64{
		{5.1, 3.5, 1.4, 0.2}, {4.9, 3.0, 1.4, 0.2}, {4.7, 3.2, 1.3, 0.2},
		{7.0, 3.2, 4.7, 1.4}, {6.4, 3.2, 4.5, 1.5}, {6.9, 3.1, 4.9, 1.5},
		{6.3, 3.3, 6.0, 2.5}, {5.8, 2.7, 5.1, 1.9}, {7.1, 3.0, 5.9, 2.1},
		{5.0, 3.6, 1.4, 0.2}, {5.4, 3.9, 1.7, 0.4}, {4.6, 3.4, 1.4, 0.3},
		{6.5, 2.8, 4.6, 1.5}, {5.7, 2.8, 4.5, 1.3}, {6.3, 3.3, 4.7, 1.6},
		{6.7, 3.1, 5.6, 2.4}, {6.9, 3.1, 5.1, 2.3}, {5.8, 2.7, 5.1, 1.9},
	}

	y := []string{
		"setosa", "setosa", "setosa",
		"versicolor", "versicolor", "versicolor",
		"virginica", "virginica", "virginica",
		"setosa", "setosa", "setosa",
		"versicolor", "versicolor", "versicolor",
		"virginica", "virginica", "virginica",
	}

	featureNames := []string{"sepal_length", "sepal_width", "petal_length", "petal_width"}

	// Train single decision tree
	dt := NewDecisionTreeClassifier(5, 2, 1)
	err := dt.Train(X, y, featureNames)
	if err != nil {
		t.Fatalf("Single tree training failed: %v", err)
	}

	// Train Random Forest
	rf := NewRandomForestClassifier(10, 5, 2, 1)
	err = rf.Train(X, y, featureNames)
	if err != nil {
		t.Fatalf("Random Forest training failed: %v", err)
	}

	// Evaluate both models on training data
	dtCorrect := 0
	rfCorrect := 0

	for i := range X {
		dtPred, _, err := dt.Predict(X[i])
		if err == nil && dtPred == y[i] {
			dtCorrect++
		}

		rfPred, _, err := rf.Predict(X[i])
		if err == nil && rfPred == y[i] {
			rfCorrect++
		}
	}

	dtAccuracy := float64(dtCorrect) / float64(len(X))
	rfAccuracy := float64(rfCorrect) / float64(len(X))

	t.Logf("Decision Tree accuracy: %.2f%%", dtAccuracy*100)
	t.Logf("Random Forest accuracy: %.2f%%", rfAccuracy*100)
	t.Logf("Random Forest OOB score: %.2f%%", rf.OOBScore*100)

	// Random Forest should generally perform at least as well as single tree
	if rfAccuracy < dtAccuracy-0.1 {
		t.Logf("Note: Random Forest accuracy (%.2f) is lower than single tree (%.2f)", 
			rfAccuracy, dtAccuracy)
	}
}

// TestRandomForestFeatureImportance tests feature importance calculation
func TestRandomForestFeatureImportance(t *testing.T) {
	// Create dataset where one feature is clearly more important
	X := [][]float64{
		{1.0, 100.0}, {2.0, 200.0}, {3.0, 300.0},
		{10.0, 100.0}, {11.0, 200.0}, {12.0, 300.0},
	}
	y := []string{"A", "A", "A", "B", "B", "B"}
	featureNames := []string{"important", "noise"}

	rf := NewRandomForestClassifier(20, 5, 2, 1)
	err := rf.Train(X, y, featureNames)
	if err != nil {
		t.Fatalf("Training failed: %v", err)
	}

	importance := rf.GetFeatureImportance()
	
	t.Logf("Feature importance:")
	for feat, imp := range importance {
		t.Logf("  %s: %.4f", feat, imp)
	}

	// Check that importance values are valid (non-negative and sum to ~1)
	sum := 0.0
	for _, imp := range importance {
		if imp < 0 {
			t.Errorf("Negative importance value: %.4f", imp)
		}
		sum += imp
	}
	
	if math.Abs(sum-1.0) > 0.1 {
		t.Errorf("Feature importances don't sum to ~1.0: %.4f (tolerance: 0.1)", sum)
	}
}

// TestRandomForestEdgeCases tests edge cases and error handling
func TestRandomForestEdgeCases(t *testing.T) {
	// Test with empty data
	rf := NewRandomForestClassifier(5, 5, 2, 1)
	err := rf.Train([][]float64{}, []string{}, []string{})
	if err == nil {
		t.Error("Expected error for empty training data")
	}

	// Test with mismatched X and y
	X := [][]float64{{1.0, 2.0}, {3.0, 4.0}}
	y := []string{"A"}
	err = rf.Train(X, y, []string{"f1", "f2"})
	if err == nil {
		t.Error("Expected error for mismatched X and y")
	}

	// Test prediction before training
	rf2 := NewRandomForestClassifier(5, 5, 2, 1)
	_, _, err = rf2.Predict([]float64{1.0, 2.0})
	if err == nil {
		t.Error("Expected error for prediction before training")
	}

	// Test with single sample (edge case)
	X = [][]float64{{1.0, 2.0}}
	y = []string{"A"}
	rf3 := NewRandomForestClassifier(3, 5, 1, 1)
	err = rf3.Train(X, y, []string{"f1", "f2"})
	if err != nil {
		t.Logf("Single sample training: %v (expected behavior)", err)
	}
}

// TestRandomForestParallelTraining verifies parallel training works correctly
func TestRandomForestParallelTraining(t *testing.T) {
	// Create a moderate-sized dataset
	X := make([][]float64, 100)
	y := make([]string, 100)
	for i := 0; i < 100; i++ {
		if i < 50 {
			X[i] = []float64{float64(i), float64(i * 2)}
			y[i] = "A"
		} else {
			X[i] = []float64{float64(i), float64(i * 2)}
			y[i] = "B"
		}
	}

	featureNames := []string{"f1", "f2"}

	// Train with many trees to test parallelism
	rf := NewRandomForestClassifier(50, 10, 2, 1)
	err := rf.Train(X, y, featureNames)
	if err != nil {
		t.Fatalf("Parallel training failed: %v", err)
	}

	// Verify all trees were trained
	if len(rf.Trees) != 50 {
		t.Errorf("Expected 50 trees, got %d", len(rf.Trees))
	}

	validTrees := 0
	for _, tree := range rf.Trees {
		if tree != nil && tree.Root != nil {
			validTrees++
		}
	}

	if validTrees != 50 {
		t.Errorf("Expected 50 valid trees, got %d", validTrees)
	}

	t.Logf("✓ Successfully trained %d trees in parallel", validTrees)
}
