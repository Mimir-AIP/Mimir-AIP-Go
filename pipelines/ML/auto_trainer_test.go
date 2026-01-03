package ml

import (
	"testing"
)

// TestAlgorithmRecommendation tests the intelligent algorithm selection
func TestAlgorithmRecommendation(t *testing.T) {
	at := &AutoTrainer{}

	tests := []struct {
		name                string
		sampleCount         int
		featureCount        int
		numClasses          int
		modelType           string
		expectedAlgorithm   string
		expectedConfidence  float64
		minConfidence       float64
	}{
		{
			name:              "Small dataset - should use decision tree",
			sampleCount:       30,
			featureCount:      5,
			modelType:         "classification",
			expectedAlgorithm: "decision_tree",
			minConfidence:     0.8,
		},
		{
			name:              "Few features - should use decision tree",
			sampleCount:       100,
			featureCount:      3,
			modelType:         "classification",
			expectedAlgorithm: "decision_tree",
			minConfidence:     0.7,
		},
		{
			name:              "Large multiclass - should use random forest",
			sampleCount:       200,
			featureCount:      10,
			numClasses:        8,
			modelType:         "classification",
			expectedAlgorithm: "random_forest",
			minConfidence:     0.9,
		},
		{
			name:              "Large dataset many features - should use random forest",
			sampleCount:       500,
			featureCount:      15,
			modelType:         "regression",
			expectedAlgorithm: "random_forest",
			minConfidence:     0.8,
		},
		{
			name:              "Medium dataset - should use random forest",
			sampleCount:       75,
			featureCount:      8,
			modelType:         "classification",
			expectedAlgorithm: "random_forest",
			minConfidence:     0.7,
		},
		{
			name:              "Very large dataset - should use random forest with more trees",
			sampleCount:       1500,
			featureCount:      20,
			modelType:         "regression",
			expectedAlgorithm: "random_forest",
			minConfidence:     0.8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock dataset
			dataset := &TrainingDataset{
				X:            make([][]float64, tt.sampleCount),
				FeatureCount: tt.featureCount,
				SampleCount:  tt.sampleCount,
			}

			// Initialize X with dummy data
			for i := 0; i < tt.sampleCount; i++ {
				dataset.X[i] = make([]float64, tt.featureCount)
			}

			// Create Y based on model type
			if tt.modelType == "classification" {
				yCateg := make([]string, tt.sampleCount)
				for i := 0; i < tt.sampleCount; i++ {
					if tt.numClasses > 0 {
						yCateg[i] = string(rune('A' + (i % tt.numClasses)))
					} else {
						yCateg[i] = "A" // Default single class
					}
				}
				dataset.Y = yCateg
			} else {
				yNumeric := make([]float64, tt.sampleCount)
				for i := 0; i < tt.sampleCount; i++ {
					yNumeric[i] = float64(i)
				}
				dataset.Y = yNumeric
			}

			// Get recommendation
			recommendation := at.recommendAlgorithm(dataset, tt.modelType)

			// Verify algorithm
			if recommendation.Algorithm != tt.expectedAlgorithm {
				t.Errorf("Expected algorithm %s, got %s", tt.expectedAlgorithm, recommendation.Algorithm)
			}

			// Verify confidence
			if recommendation.Confidence < tt.minConfidence {
				t.Errorf("Expected confidence >= %.2f, got %.2f", tt.minConfidence, recommendation.Confidence)
			}

			// Verify reasoning exists
			if recommendation.Reasoning == "" {
				t.Error("Reasoning should not be empty")
			}

			// Verify NumTrees for random forest
			if recommendation.Algorithm == "random_forest" {
				if recommendation.NumTrees <= 0 {
					t.Errorf("Random forest should have NumTrees > 0, got %d", recommendation.NumTrees)
				}

				// Check tree count scaling with dataset size
				if tt.sampleCount >= 1000 && recommendation.NumTrees < 100 {
					t.Errorf("Large dataset should use more trees, got %d", recommendation.NumTrees)
				}
			}

			t.Logf("✓ Algorithm: %s (confidence: %.2f)", recommendation.Algorithm, recommendation.Confidence)
			t.Logf("  Reasoning: %s", recommendation.Reasoning)
			if recommendation.Algorithm == "random_forest" {
				t.Logf("  Trees: %d", recommendation.NumTrees)
			}
		})
	}
}

// TestAlgorithmRecommendationEdgeCases tests edge cases in algorithm selection
func TestAlgorithmRecommendationEdgeCases(t *testing.T) {
	at := &AutoTrainer{}

	t.Run("Empty classes defaults to binary", func(t *testing.T) {
		dataset := &TrainingDataset{
			X:            make([][]float64, 100),
			Y:            []string{"A", "B", "A", "B"},
			FeatureCount: 10,
			SampleCount:  100,
		}
		for i := range dataset.X {
			dataset.X[i] = make([]float64, 10)
		}

		recommendation := at.recommendAlgorithm(dataset, "classification")
		if recommendation.Algorithm == "" {
			t.Error("Should always return an algorithm")
		}
	})

	t.Run("Regression with many features", func(t *testing.T) {
		dataset := &TrainingDataset{
			X:            make([][]float64, 200),
			Y:            make([]float64, 200),
			FeatureCount: 50,
			SampleCount:  200,
		}
		for i := range dataset.X {
			dataset.X[i] = make([]float64, 50)
		}

		recommendation := at.recommendAlgorithm(dataset, "regression")
		if recommendation.Algorithm != "random_forest" {
			t.Error("Should recommend random forest for many features")
		}
	})
}
