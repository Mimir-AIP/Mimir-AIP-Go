package digitaltwin

import "testing"

func TestPredictRegressionReadsNestedModelData(t *testing.T) {
	engine := &InferenceEngine{}
	artifact := &ModelArtifact{
		ModelType:    "regression",
		FeatureNames: []string{"x1", "x2"},
		Parameters: map[string]interface{}{
			"model_data": map[string]interface{}{
				"coefficients": []interface{}{2.0, 3.0},
				"intercept":    1.0,
			},
		},
	}

	prediction := engine.predictRegression(artifact, []float64{1.0, 2.0})
	if prediction.(float64) != 9.0 {
		t.Fatalf("expected regression prediction 9.0, got %v", prediction)
	}
}
