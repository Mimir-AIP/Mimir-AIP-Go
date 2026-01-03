package main

import (
	"fmt"
	"log"

	ml "github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/ML"
)

// Example demonstrating Random Forest usage for both classification and regression

func main() {
	fmt.Println("=== Random Forest Examples ===\n")

	// Example 1: Classification with Iris-like dataset
	classificationExample()

	fmt.Println()

	// Example 2: Regression with housing prices
	regressionExample()
}

func classificationExample() {
	fmt.Println("Example 1: Random Forest Classification")
	fmt.Println("----------------------------------------")

	// Create iris-like dataset
	// Features: sepal_length, sepal_width, petal_length, petal_width
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

	// Configure training
	config := ml.DefaultTrainingConfig()
	config.NumTrees = 50 // Use 50 trees in the forest
	config.MaxDepth = 10
	config.TrainTestSplit = 0.8

	// Create trainer and train Random Forest
	trainer := ml.NewTrainer(config)
	result, err := trainer.TrainRandomForest(X, y, featureNames)
	if err != nil {
		log.Fatalf("Training failed: %v", err)
	}

	// Display training results
	fmt.Printf("Training completed in %v\n", result.TrainingDuration)
	fmt.Printf("Training samples: %d, Validation samples: %d\n", result.TrainingRows, result.ValidationRows)
	fmt.Printf("Training accuracy: %.2f%%\n", result.TrainMetrics.Accuracy*100)
	fmt.Printf("Validation accuracy: %.2f%%\n", result.ValidateMetrics.Accuracy*100)
	fmt.Printf("F1-score: %.4f\n", result.ValidateMetrics.MacroF1)
	fmt.Printf("OOB Score: %.2f%%\n", result.ModelRF.OOBScore*100)

	// Display feature importance
	fmt.Println("\nFeature Importance:")
	for feature, importance := range result.FeatureImportance {
		fmt.Printf("  %s: %.4f\n", feature, importance)
	}

	// Make predictions
	fmt.Println("\nMaking predictions:")
	testSamples := []struct {
		features []float64
		expected string
	}{
		{[]float64{5.0, 3.4, 1.5, 0.2}, "setosa"},
		{[]float64{6.5, 3.0, 4.6, 1.4}, "versicolor"},
		{[]float64{6.7, 3.1, 5.6, 2.4}, "virginica"},
	}

	for _, sample := range testSamples {
		predicted, confidence, err := result.ModelRF.Predict(sample.features)
		if err != nil {
			log.Printf("Prediction failed: %v", err)
			continue
		}

		// Get class probabilities
		proba, _ := result.ModelRF.PredictProba(sample.features)

		fmt.Printf("  Sample: %v\n", sample.features)
		fmt.Printf("    Predicted: %s (confidence: %.2f%%)\n", predicted, confidence*100)
		fmt.Printf("    Expected: %s\n", sample.expected)
		fmt.Printf("    Probabilities: %v\n", proba)
	}

	// Save model
	modelPath := "/tmp/iris_random_forest.json"
	if err := result.ModelRF.Save(modelPath); err != nil {
		log.Printf("Failed to save model: %v", err)
	} else {
		fmt.Printf("\nModel saved to: %s\n", modelPath)
	}
}

func regressionExample() {
	fmt.Println("Example 2: Random Forest Regression")
	fmt.Println("------------------------------------")

	// Create housing price dataset
	// Features: [square_feet, bedrooms, bathrooms, age]
	X := [][]float64{
		{1500, 3, 2, 10}, {2000, 4, 3, 5}, {1200, 2, 1, 15},
		{2500, 4, 3, 2}, {1800, 3, 2, 8}, {3000, 5, 4, 1},
		{1100, 2, 1, 20}, {2200, 4, 3, 7}, {1600, 3, 2, 12},
		{2800, 5, 3, 3}, {1400, 2, 2, 18}, {1900, 3, 2, 9},
	}

	// Target: price in thousands
	y := []float64{
		250, 350, 180,
		450, 300, 550,
		150, 380, 270,
		500, 220, 320,
	}

	featureNames := []string{"square_feet", "bedrooms", "bathrooms", "age"}

	// Configure training
	config := ml.DefaultTrainingConfig()
	config.NumTrees = 100 // Use 100 trees for regression
	config.MaxDepth = 15
	config.TrainTestSplit = 0.75

	// Create trainer and train Random Forest
	trainer := ml.NewTrainer(config)
	result, err := trainer.TrainRandomForestRegression(X, y, featureNames)
	if err != nil {
		log.Fatalf("Training failed: %v", err)
	}

	// Display training results
	fmt.Printf("Training completed in %v\n", result.TrainingDuration)
	fmt.Printf("Training samples: %d, Validation samples: %d\n", result.TrainingRows, result.ValidationRows)
	fmt.Printf("R² Score (training): %.4f\n", result.TrainMetricsReg.R2Score)
	fmt.Printf("R² Score (validation): %.4f\n", result.ValidateMetricsReg.R2Score)
	fmt.Printf("MAE (validation): %.2f\n", result.ValidateMetricsReg.MAE)
	fmt.Printf("RMSE (validation): %.2f\n", result.ValidateMetricsReg.RMSE)
	fmt.Printf("OOB R² Score: %.4f\n", result.ModelRF.OOBScore)

	// Display feature importance
	fmt.Println("\nFeature Importance:")
	for feature, importance := range result.FeatureImportance {
		fmt.Printf("  %s: %.4f\n", feature, importance)
	}

	// Make predictions with confidence intervals
	fmt.Println("\nMaking predictions:")
	testSamples := []struct {
		features []float64
		desc     string
	}{
		{[]float64{1700, 3, 2, 10}, "Medium house, 10 years old"},
		{[]float64{2500, 4, 3, 5}, "Large house, 5 years old"},
		{[]float64{1200, 2, 1, 20}, "Small house, 20 years old"},
	}

	for _, sample := range testSamples {
		predicted, lower, upper, err := result.ModelRF.PredictRegressionWithInterval(sample.features)
		if err != nil {
			log.Printf("Prediction failed: %v", err)
			continue
		}

		fmt.Printf("  %s\n", sample.desc)
		fmt.Printf("    Features: %v\n", sample.features)
		fmt.Printf("    Predicted price: $%.0fk\n", predicted)
		fmt.Printf("    95%% confidence interval: [$%.0fk, $%.0fk]\n", lower, upper)
	}

	// Save model
	modelPath := "/tmp/housing_random_forest.json"
	if err := result.ModelRF.Save(modelPath); err != nil {
		log.Printf("Failed to save model: %v", err)
	} else {
		fmt.Printf("\nModel saved to: %s\n", modelPath)
	}
}
