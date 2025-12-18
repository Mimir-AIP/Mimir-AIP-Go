package performance

import (
	"fmt"
	"testing"
	"time"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/ML"
)

// TestQuick5KPerformance quickly tests 5K dataset to validate optimizations
// Run with: go test -run TestQuick5KPerformance -v
func TestQuick5KPerformance(t *testing.T) {
	numSamples := 5000
	numFeatures := 10

	t.Logf("Testing ML training with %d samples, %d features", numSamples, numFeatures)

	X := generateDataset(numSamples, numFeatures)
	y := generateLabels(numSamples)
	featureNames := make([]string, numFeatures)
	for i := range featureNames {
		featureNames[i] = fmt.Sprintf("f%d", i)
	}

	start := time.Now()

	config := ml.DefaultTrainingConfig()
	trainer := ml.NewTrainer(config)

	_, err := trainer.Train(X, y, featureNames)
	if err != nil {
		t.Fatalf("Training failed: %v", err)
	}

	duration := time.Since(start)

	t.Logf("\n=== 5K Dataset Results ===")
	t.Logf("Training time: %v", duration)
	t.Logf("Model trained successfully")
	t.Logf("Estimated time for 10K: ~%v", duration*4)
	t.Logf("Estimated time for 50K: ~%v", duration*100)

	// Performance check - with optimization, 5K samples should take <1 minute
	if duration > 60*time.Second {
		t.Errorf("Training time (%v) exceeds 60s for %d samples", duration, numSamples)
	}
}

// TestQuick10KPerformance tests 10K dataset (takes longer)
// Run with: go test -run TestQuick10KPerformance -v -timeout=5m
func TestQuick10KPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping 10K test in short mode")
	}

	numSamples := 10000
	numFeatures := 10

	t.Logf("Testing ML training with %d samples, %d features", numSamples, numFeatures)

	X := generateDataset(numSamples, numFeatures)
	y := generateLabels(numSamples)
	featureNames := make([]string, numFeatures)
	for i := range featureNames {
		featureNames[i] = fmt.Sprintf("f%d", i)
	}

	start := time.Now()

	config := ml.DefaultTrainingConfig()
	trainer := ml.NewTrainer(config)

	_, err := trainer.Train(X, y, featureNames)
	if err != nil {
		t.Fatalf("Training failed: %v", err)
	}

	duration := time.Since(start)

	t.Logf("\n=== 10K Dataset Results ===")
	t.Logf("Training time: %v", duration)
	t.Logf("Estimated time for 50K: ~%v", duration*5)
	t.Logf("Estimated time for 100K: ~%v", duration*10)

	// With optimization, 10K samples should take <5 minutes
	if duration > 5*time.Minute {
		t.Errorf("Training time (%v) exceeds 5 minutes for %d samples", duration, numSamples)
	}
}
