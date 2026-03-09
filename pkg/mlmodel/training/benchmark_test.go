package training

import (
	"math"
	"testing"

	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

func benchmarkTrainingData(samples int, features int) *TrainingData {
	trainFeatures := make([][]float64, samples)
	trainLabels := make([]float64, samples)
	testFeatures := make([][]float64, samples/4)
	testLabels := make([]float64, samples/4)
	featureNames := make([]string, features)

	for i := 0; i < features; i++ {
		featureNames[i] = "f" + string(rune('a'+(i%26)))
	}

	for i := 0; i < samples; i++ {
		row := make([]float64, features)
		sum := 0.0
		for j := 0; j < features; j++ {
			v := float64((i*(j+3))%17) / 16.0
			row[j] = v
			sum += (float64(j%5) + 1.0) * v
		}
		trainFeatures[i] = row
		if sum > float64(features)/2.2 {
			trainLabels[i] = 1.0
		}
	}

	for i := 0; i < len(testFeatures); i++ {
		row := make([]float64, features)
		sum := 0.0
		for j := 0; j < features; j++ {
			v := float64(((i+7)*(j+5))%19) / 18.0
			row[j] = v
			sum += (float64(j%5) + 1.0) * v
		}
		testFeatures[i] = row
		if sum > float64(features)/2.2 {
			testLabels[i] = 1.0
		}
	}

	return &TrainingData{
		TrainFeatures: trainFeatures,
		TrainLabels:   trainLabels,
		TestFeatures:  testFeatures,
		TestLabels:    testLabels,
		FeatureNames:  featureNames,
	}
}

func benchmarkRegressionData(samples int, features int) *TrainingData {
	trainFeatures := make([][]float64, samples)
	trainLabels := make([]float64, samples)
	testFeatures := make([][]float64, samples/4)
	testLabels := make([]float64, samples/4)
	featureNames := make([]string, features)

	for i := 0; i < features; i++ {
		featureNames[i] = "x" + string(rune('a'+(i%26)))
	}

	for i := 0; i < samples; i++ {
		row := make([]float64, features)
		y := 2.5
		for j := 0; j < features; j++ {
			v := float64((i*(j+1))%23) / 22.0
			row[j] = v
			y += (float64(j) + 1.0) * 0.2 * v
		}
		trainFeatures[i] = row
		trainLabels[i] = y
	}

	for i := 0; i < len(testFeatures); i++ {
		row := make([]float64, features)
		y := 2.5
		for j := 0; j < features; j++ {
			v := float64(((i+11)*(j+2))%29) / 28.0
			row[j] = v
			y += (float64(j) + 1.0) * 0.2 * v
		}
		testFeatures[i] = row
		testLabels[i] = y
	}

	return &TrainingData{
		TrainFeatures: trainFeatures,
		TrainLabels:   trainLabels,
		TestFeatures:  testFeatures,
		TestLabels:    testLabels,
		FeatureNames:  featureNames,
	}
}

func BenchmarkRandomForestTrain_AccuracyAndThroughput(b *testing.B) {
	data := benchmarkTrainingData(4000, 16)
	trainer := NewRandomForestTrainer()
	config := &models.TrainingConfig{MaxIterations: 20}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := trainer.Train(data, config)
		if err != nil {
			b.Fatalf("Train failed: %v", err)
		}
		b.ReportMetric(result.PerformanceMetrics.Accuracy, "accuracy")
		if result.PerformanceMetrics.Accuracy < 0.70 {
			b.Fatalf("accuracy below floor: %.3f", result.PerformanceMetrics.Accuracy)
		}
	}
}

func BenchmarkRegressionTrain_RMSE(b *testing.B) {
	data := benchmarkRegressionData(3000, 12)
	trainer := NewRegressionTrainer()
	config := &models.TrainingConfig{}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := trainer.Train(data, config)
		if err != nil {
			b.Fatalf("Train failed: %v", err)
		}
		rmse := result.PerformanceMetrics.RMSE
		b.ReportMetric(rmse, "rmse")
		if math.IsNaN(rmse) || math.IsInf(rmse, 0) {
			b.Fatalf("invalid RMSE metric: %v", rmse)
		}
	}
}
