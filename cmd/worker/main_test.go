package main

import (
	"math"
	"testing"
)

func TestWorkerRunInferenceRegressionUsesModelData(t *testing.T) {
	parameters := map[string]any{
		"model_data": map[string]any{
			"coefficients": []any{2.0, 3.0},
			"intercept":    1.0,
		},
	}

	pred, err := workerRunInference("regression", parameters, []float64{1.0, 2.0})
	if err != nil {
		t.Fatalf("workerRunInference returned error: %v", err)
	}

	want := 9.0 // 1 + (2*1) + (3*2)
	if math.Abs(pred-want) > 1e-9 {
		t.Fatalf("unexpected prediction: got %.6f want %.6f", pred, want)
	}
}

func TestWorkerRunInferenceNeuralNetworkUsesModelData(t *testing.T) {
	parameters := map[string]any{
		"model_data": map[string]any{
			"weights": []any{
				[]any{
					[]any{1.0, 2.0},
				},
			},
			"biases": []any{
				[]any{0.5},
			},
		},
	}

	pred, err := workerRunInference("neural_network", parameters, []float64{1.0, 1.0})
	if err != nil {
		t.Fatalf("workerRunInference returned error: %v", err)
	}

	// sigmoid(0.5 + 1*1 + 2*1) = sigmoid(3.5)
	want := 1.0 / (1.0 + math.Exp(-3.5))
	if math.Abs(pred-want) > 1e-9 {
		t.Fatalf("unexpected prediction: got %.12f want %.12f", pred, want)
	}
}
