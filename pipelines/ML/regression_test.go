package ml

import (
	"testing"
)

func TestDecisionTreeRegression(t *testing.T) {
	// Create simple regression dataset: predict house prices based on size and bedrooms
	X := [][]float64{
		{1000, 2}, // 1000 sqft, 2 bedrooms
		{1500, 3}, // 1500 sqft, 3 bedrooms
		{2000, 4}, // 2000 sqft, 4 bedrooms
		{1200, 2}, // 1200 sqft, 2 bedrooms
		{1800, 3}, // 1800 sqft, 3 bedrooms
		{2500, 5}, // 2500 sqft, 5 bedrooms
		{1100, 2}, // 1100 sqft, 2 bedrooms
		{1700, 3}, // 1700 sqft, 3 bedrooms
	}

	// Target: house prices in thousands
	y := []float64{100, 150, 200, 120, 180, 250, 110, 170}

	featureNames := []string{"sqft", "bedrooms"}

	// Train regression model
	regressor := NewDecisionTreeClassifier(5, 2, 1)
	err := regressor.TrainRegression(X, y, featureNames)
	if err != nil {
		t.Fatalf("Training failed: %v", err)
	}

	// Verify model type
	if regressor.ModelType != "regression" {
		t.Errorf("Expected model type 'regression', got '%s'", regressor.ModelType)
	}

	// Test predictions
	testCases := []struct {
		input    []float64
		expected float64
		margin   float64
	}{
		{[]float64{1000, 2}, 100, 20}, // Similar to first training example
		{[]float64{1500, 3}, 150, 20}, // Similar to second training example
		{[]float64{2000, 4}, 200, 30}, // Similar to third training example
	}

	for i, tc := range testCases {
		pred, err := regressor.PredictRegression(tc.input)
		if err != nil {
			t.Errorf("Test case %d: prediction failed: %v", i, err)
			continue
		}

		diff := pred - tc.expected
		if diff < 0 {
			diff = -diff
		}

		if diff > tc.margin {
			t.Errorf("Test case %d: predicted %.2f, expected %.2f ± %.2f", i, pred, tc.expected, tc.margin)
		}
	}
}

func TestDecisionTreeRegressionWithInterval(t *testing.T) {
	// Create simple dataset
	X := [][]float64{
		{1.0}, {2.0}, {3.0}, {4.0}, {5.0},
	}
	y := []float64{2.0, 4.0, 6.0, 8.0, 10.0}

	featureNames := []string{"x"}

	// Train model
	regressor := NewDecisionTreeClassifier(3, 1, 1)
	err := regressor.TrainRegression(X, y, featureNames)
	if err != nil {
		t.Fatalf("Training failed: %v", err)
	}

	// Test prediction with confidence interval
	value, lower, upper, err := regressor.PredictRegressionWithInterval([]float64{3.0})
	if err != nil {
		t.Fatalf("Prediction with interval failed: %v", err)
	}

	// Verify value is within interval
	if value < lower || value > upper {
		t.Errorf("Predicted value %.2f not within interval [%.2f, %.2f]", value, lower, upper)
	}

	// Verify interval makes sense
	if lower > upper {
		t.Errorf("Invalid interval: lower (%.2f) > upper (%.2f)", lower, upper)
	}
}

func TestRegressionMetrics(t *testing.T) {
	yTrue := []float64{100, 150, 200, 120, 180}
	yPred := []float64{105, 145, 195, 125, 175}

	metrics, err := CalculateRegressionMetrics(yTrue, yPred)
	if err != nil {
		t.Fatalf("Failed to calculate metrics: %v", err)
	}

	// Verify basic properties
	if metrics.NumSamples != 5 {
		t.Errorf("Expected 5 samples, got %d", metrics.NumSamples)
	}

	// MAE should be around 5 (all predictions off by 5)
	if metrics.MAE < 4 || metrics.MAE > 6 {
		t.Errorf("Unexpected MAE: %.2f", metrics.MAE)
	}

	// R² should be high (very good predictions)
	if metrics.R2Score < 0.95 {
		t.Errorf("Expected high R² score, got %.4f", metrics.R2Score)
	}

	// RMSE should be low
	if metrics.RMSE > 10 {
		t.Errorf("Unexpected RMSE: %.2f", metrics.RMSE)
	}
}

func TestEvaluateRegression(t *testing.T) {
	// Create and train a model
	X := [][]float64{
		{1.0}, {2.0}, {3.0}, {4.0}, {5.0}, {6.0}, {7.0}, {8.0},
	}
	y := []float64{2.0, 4.0, 6.0, 8.0, 10.0, 12.0, 14.0, 16.0}

	featureNames := []string{"x"}

	regressor := NewDecisionTreeClassifier(5, 2, 1)
	err := regressor.TrainRegression(X, y, featureNames)
	if err != nil {
		t.Fatalf("Training failed: %v", err)
	}

	// Evaluate on test set
	testX := [][]float64{{1.5}, {3.5}, {5.5}, {7.5}}
	testY := []float64{3.0, 7.0, 11.0, 15.0}

	metrics, err := EvaluateRegression(regressor, testX, testY)
	if err != nil {
		t.Fatalf("Evaluation failed: %v", err)
	}

	// Verify metrics exist
	if metrics.NumSamples != 4 {
		t.Errorf("Expected 4 test samples, got %d", metrics.NumSamples)
	}

	// Should have reasonable R² score for linear data
	if metrics.R2Score < 0.5 {
		t.Errorf("Expected reasonable R² score, got %.4f", metrics.R2Score)
	}
}

func TestTrainRegression(t *testing.T) {
	// Prepare data - larger dataset for better train/test split
	X := [][]float64{
		{1.0, 2.0}, {2.0, 3.0}, {3.0, 4.0}, {4.0, 5.0},
		{5.0, 6.0}, {6.0, 7.0}, {7.0, 8.0}, {8.0, 9.0},
		{9.0, 10.0}, {10.0, 11.0}, {11.0, 12.0}, {12.0, 13.0},
	}
	y := []float64{5.0, 8.0, 11.0, 14.0, 17.0, 20.0, 23.0, 26.0, 29.0, 32.0, 35.0, 38.0}

	featureNames := []string{"feature1", "feature2"}

	// Create trainer with custom config
	config := DefaultTrainingConfig()
	config.TrainTestSplit = 0.75
	config.Shuffle = false // Don't shuffle for reproducibility

	trainer := NewTrainer(config)

	// Train regression model
	result, err := trainer.TrainRegression(X, y, featureNames)
	if err != nil {
		t.Fatalf("Training failed: %v", err)
	}

	// Verify result
	if result.Model == nil {
		t.Fatal("Model is nil")
	}

	if result.ModelType != "regression" {
		t.Errorf("Expected model type 'regression', got '%s'", result.ModelType)
	}

	if result.TrainMetricsReg == nil {
		t.Fatal("Training metrics are nil")
	}

	if result.ValidateMetricsReg == nil {
		t.Fatal("Validation metrics are nil")
	}

	// Verify metrics are calculated (values can vary with small datasets)
	if result.ValidateMetricsReg.NumSamples == 0 {
		t.Error("No validation samples")
	}

	// Verify feature importance exists
	if len(result.FeatureImportance) == 0 {
		t.Error("Feature importance is empty")
	}
}

func TestRegressionVarianceCalculation(t *testing.T) {
	values := []float64{1.0, 2.0, 3.0, 4.0, 5.0}
	mean := calculateMean(values)

	// Mean should be 3.0
	if mean != 3.0 {
		t.Errorf("Expected mean 3.0, got %.2f", mean)
	}

	variance := calculateVariance(values, mean)

	// Variance of 1,2,3,4,5 is 2.0
	expectedVariance := 2.0
	if variance < expectedVariance-0.1 || variance > expectedVariance+0.1 {
		t.Errorf("Expected variance %.2f, got %.2f", expectedVariance, variance)
	}
}
