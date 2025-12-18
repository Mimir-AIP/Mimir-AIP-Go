package ml

import (
	"os"
	"testing"
)

// TestDecisionTreeClassifierIris tests the classifier on a simple iris-like dataset
func TestDecisionTreeClassifierIris(t *testing.T) {
	// Create a simple dataset (iris-like)
	// Features: sepal_length, sepal_width, petal_length, petal_width
	X := [][]float64{
		{5.1, 3.5, 1.4, 0.2}, // setosa
		{4.9, 3.0, 1.4, 0.2}, // setosa
		{4.7, 3.2, 1.3, 0.2}, // setosa
		{7.0, 3.2, 4.7, 1.4}, // versicolor
		{6.4, 3.2, 4.5, 1.5}, // versicolor
		{6.9, 3.1, 4.9, 1.5}, // versicolor
		{6.3, 3.3, 6.0, 2.5}, // virginica
		{5.8, 2.7, 5.1, 1.9}, // virginica
		{7.1, 3.0, 5.9, 2.1}, // virginica
	}

	y := []string{
		"setosa", "setosa", "setosa",
		"versicolor", "versicolor", "versicolor",
		"virginica", "virginica", "virginica",
	}

	featureNames := []string{"sepal_length", "sepal_width", "petal_length", "petal_width"}

	// Train classifier
	classifier := NewDecisionTreeClassifier(5, 2, 1)
	err := classifier.Train(X, y, featureNames)
	if err != nil {
		t.Fatalf("Training failed: %v", err)
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
		predicted, confidence, err := classifier.Predict(tt.input)
		if err != nil {
			t.Errorf("Prediction failed: %v", err)
		}
		if predicted != tt.expected {
			t.Errorf("Expected %s, got %s (confidence: %.2f)", tt.expected, predicted, confidence)
		}
		t.Logf("Predicted: %s (confidence: %.2f)", predicted, confidence)
	}
}

// TestDecisionTreeClassifierSaveLoad tests model persistence
func TestDecisionTreeClassifierSaveLoad(t *testing.T) {
	// Train a simple model
	X := [][]float64{
		{1.0, 2.0}, {2.0, 3.0}, {3.0, 4.0},
		{10.0, 11.0}, {11.0, 12.0}, {12.0, 13.0},
	}
	y := []string{"A", "A", "A", "B", "B", "B"}
	featureNames := []string{"feature1", "feature2"}

	classifier := NewDecisionTreeClassifier(3, 2, 1)
	err := classifier.Train(X, y, featureNames)
	if err != nil {
		t.Fatalf("Training failed: %v", err)
	}

	// Make a prediction before saving
	testInput := []float64{1.5, 2.5}
	pred1, conf1, err := classifier.Predict(testInput)
	if err != nil {
		t.Fatalf("Prediction before save failed: %v", err)
	}

	// Save model
	tmpFile := "/tmp/test_model.json"
	defer os.Remove(tmpFile)

	err = classifier.Save(tmpFile)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Load model
	loadedClassifier := &DecisionTreeClassifier{}
	err = loadedClassifier.Load(tmpFile)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Make same prediction after loading
	pred2, conf2, err := loadedClassifier.Predict(testInput)
	if err != nil {
		t.Fatalf("Prediction after load failed: %v", err)
	}

	// Check predictions match
	if pred1 != pred2 || conf1 != conf2 {
		t.Errorf("Predictions don't match after save/load: (%s, %.2f) vs (%s, %.2f)",
			pred1, conf1, pred2, conf2)
	}

	t.Logf("Save/Load successful: %s (%.2f)", pred2, conf2)
}

// TestTrainer tests the training pipeline
func TestTrainer(t *testing.T) {
	// Create a larger dataset
	X := [][]float64{}
	y := []string{}

	// Class A: small values
	for i := 0; i < 20; i++ {
		X = append(X, []float64{float64(i%5 + 1), float64(i%3 + 1)})
		y = append(y, "A")
	}

	// Class B: large values
	for i := 0; i < 20; i++ {
		X = append(X, []float64{float64(i%5 + 10), float64(i%3 + 10)})
		y = append(y, "B")
	}

	featureNames := []string{"feature1", "feature2"}

	// Train with default config
	config := DefaultTrainingConfig()
	config.TrainTestSplit = 0.75
	trainer := NewTrainer(config)

	result, err := trainer.Train(X, y, featureNames)
	if err != nil {
		t.Fatalf("Training failed: %v", err)
	}

	// Check training results
	if result.TrainMetrics.Accuracy < 0.5 {
		t.Errorf("Training accuracy too low: %.2f", result.TrainMetrics.Accuracy)
	}
	if result.ValidateMetrics.Accuracy < 0.5 {
		t.Errorf("Validation accuracy too low: %.2f", result.ValidateMetrics.Accuracy)
	}

	t.Logf("Training accuracy: %.2f", result.TrainMetrics.Accuracy)
	t.Logf("Validation accuracy: %.2f", result.ValidateMetrics.Accuracy)
	t.Logf("Training rows: %d, Validation rows: %d", result.TrainingRows, result.ValidationRows)
	t.Logf("Training duration: %v", result.TrainingDuration)
	t.Logf("Feature importance: %v", result.FeatureImportance)
}

// TestEvaluationMetrics tests the evaluation metrics
func TestEvaluationMetrics(t *testing.T) {
	yTrue := []string{"A", "A", "A", "B", "B", "B", "C", "C", "C"}
	yPred := []string{"A", "A", "B", "B", "B", "A", "C", "C", "B"}
	classes := []string{"A", "B", "C"}

	metrics, err := CalculateMetrics(yTrue, yPred, classes)
	if err != nil {
		t.Fatalf("Metrics calculation failed: %v", err)
	}

	// Check overall accuracy (6/9 = 0.667)
	expectedAccuracy := 6.0 / 9.0
	if metrics.Accuracy < expectedAccuracy-0.01 || metrics.Accuracy > expectedAccuracy+0.01 {
		t.Errorf("Expected accuracy ~%.2f, got %.2f", expectedAccuracy, metrics.Accuracy)
	}

	// Check that metrics are calculated for all classes
	for _, class := range classes {
		if _, ok := metrics.Precision[class]; !ok {
			t.Errorf("Missing precision for class %s", class)
		}
		if _, ok := metrics.Recall[class]; !ok {
			t.Errorf("Missing recall for class %s", class)
		}
		if _, ok := metrics.F1Score[class]; !ok {
			t.Errorf("Missing F1 score for class %s", class)
		}
	}

	t.Logf("Accuracy: %.2f", metrics.Accuracy)
	t.Logf("Macro Precision: %.2f", metrics.MacroPrecision)
	t.Logf("Macro Recall: %.2f", metrics.MacroRecall)
	t.Logf("Macro F1: %.2f", metrics.MacroF1)
	t.Logf("\n%s", metrics.FormatMetrics())
	t.Logf("\n%s", metrics.FormatConfusionMatrix())
}

// TestPrepareDataFromCSV tests CSV data preparation
func TestPrepareDataFromCSV(t *testing.T) {
	data := [][]string{
		{"feature1", "feature2", "target"},
		{"1.0", "2.0", "A"},
		{"3.0", "4.0", "B"},
		{"5.0", "6.0", "A"},
	}

	X, y, featureNames, err := PrepareDataFromCSV(data, "target")
	if err != nil {
		t.Fatalf("Data preparation failed: %v", err)
	}

	// Check dimensions
	if len(X) != 3 {
		t.Errorf("Expected 3 rows, got %d", len(X))
	}
	if len(y) != 3 {
		t.Errorf("Expected 3 labels, got %d", len(y))
	}
	if len(featureNames) != 2 {
		t.Errorf("Expected 2 features, got %d", len(featureNames))
	}

	// Check feature names
	expectedFeatures := []string{"feature1", "feature2"}
	for i, name := range featureNames {
		if name != expectedFeatures[i] {
			t.Errorf("Expected feature %s, got %s", expectedFeatures[i], name)
		}
	}

	// Check labels
	expectedLabels := []string{"A", "B", "A"}
	for i, label := range y {
		if label != expectedLabels[i] {
			t.Errorf("Expected label %s, got %s", expectedLabels[i], label)
		}
	}

	t.Logf("Features: %v", featureNames)
	t.Logf("X shape: (%d, %d)", len(X), len(X[0]))
	t.Logf("y: %v", y)
}

// TestModelMemoryEstimate tests memory estimation
func TestModelMemoryEstimate(t *testing.T) {
	X := [][]float64{
		{1.0, 2.0}, {2.0, 3.0}, {3.0, 4.0},
		{10.0, 11.0}, {11.0, 12.0}, {12.0, 13.0},
	}
	y := []string{"A", "A", "A", "B", "B", "B"}
	featureNames := []string{"feature1", "feature2"}

	classifier := NewDecisionTreeClassifier(10, 2, 1)
	err := classifier.Train(X, y, featureNames)
	if err != nil {
		t.Fatalf("Training failed: %v", err)
	}

	memoryEstimate := classifier.EstimateMemoryUsage()
	if memoryEstimate == 0 {
		t.Error("Memory estimate should be > 0")
	}

	numNodes := classifier.GetNumNodes()
	depth := classifier.GetDepth()

	t.Logf("Model memory estimate: %d bytes (%.2f KB)", memoryEstimate, float64(memoryEstimate)/1024)
	t.Logf("Number of nodes: %d", numNodes)
	t.Logf("Tree depth: %d", depth)
}

// TestAnomalyDetection tests low confidence detection
func TestAnomalyDetection(t *testing.T) {
	// Train on clear separation
	X := [][]float64{
		{1.0, 1.0}, {1.5, 1.5}, {2.0, 2.0},
		{10.0, 10.0}, {10.5, 10.5}, {11.0, 11.0},
	}
	y := []string{"A", "A", "A", "B", "B", "B"}
	featureNames := []string{"x", "y"}

	classifier := NewDecisionTreeClassifier(5, 2, 1)
	err := classifier.Train(X, y, featureNames)
	if err != nil {
		t.Fatalf("Training failed: %v", err)
	}

	// Test with clear examples
	clearExample := []float64{1.2, 1.2}
	isAnomaly, err := classifier.IsLowConfidence(clearExample, 0.7)
	if err != nil {
		t.Fatalf("Anomaly check failed: %v", err)
	}
	if isAnomaly {
		t.Error("Clear example should not be flagged as anomaly")
	}

	// Test with ambiguous example (between clusters)
	ambiguousExample := []float64{5.5, 5.5}
	confidence, err := classifier.CalculateConfidence(ambiguousExample)
	if err != nil {
		t.Fatalf("Confidence calculation failed: %v", err)
	}

	t.Logf("Clear example confidence: High")
	t.Logf("Ambiguous example confidence: %.2f", confidence)
}

// BenchmarkTraining benchmarks training performance
func BenchmarkTraining(b *testing.B) {
	// Create a dataset
	X := [][]float64{}
	y := []string{}
	for i := 0; i < 100; i++ {
		if i < 50 {
			X = append(X, []float64{float64(i), float64(i * 2)})
			y = append(y, "A")
		} else {
			X = append(X, []float64{float64(i), float64(i * 2)})
			y = append(y, "B")
		}
	}
	featureNames := []string{"f1", "f2"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		classifier := NewDecisionTreeClassifier(10, 2, 1)
		classifier.Train(X, y, featureNames)
	}
}

// BenchmarkPrediction benchmarks prediction performance
func BenchmarkPrediction(b *testing.B) {
	// Train a model
	X := [][]float64{{1, 2}, {2, 3}, {10, 11}, {11, 12}}
	y := []string{"A", "A", "B", "B"}
	featureNames := []string{"f1", "f2"}

	classifier := NewDecisionTreeClassifier(5, 2, 1)
	classifier.Train(X, y, featureNames)

	testInput := []float64{5.5, 6.5}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		classifier.Predict(testInput)
	}
}
