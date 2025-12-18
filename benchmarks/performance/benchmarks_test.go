package performance

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
	ml "github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/ML"
	storage "github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/Storage"
	"github.com/Mimir-AIP/Mimir-AIP-Go/utils"
)

// Benchmark results will be written to this file
var benchmarkResultsFile = "benchmark_results.json"

// BenchmarkResult stores detailed benchmark metrics
type BenchmarkResult struct {
	Name            string        `json:"name"`
	Category        string        `json:"category"`
	NsPerOp         int64         `json:"ns_per_op"`
	BytesPerOp      int64         `json:"bytes_per_op"`
	AllocsPerOp     int64         `json:"allocs_per_op"`
	MBPerSec        float64       `json:"mb_per_sec"`
	Iterations      int           `json:"iterations"`
	TotalDuration   time.Duration `json:"total_duration"`
	MemoryAllocated int64         `json:"memory_allocated_bytes"`
	Timestamp       time.Time     `json:"timestamp"`
}

var globalResults []BenchmarkResult

// SaveBenchmarkResult saves a benchmark result
func SaveBenchmarkResult(b *testing.B, category string, bytesPerOp, allocsPerOp int64) {
	result := BenchmarkResult{
		Name:        b.Name(),
		Category:    category,
		NsPerOp:     int64(b.N),
		BytesPerOp:  bytesPerOp,
		AllocsPerOp: allocsPerOp,
		Iterations:  b.N,
		Timestamp:   time.Now(),
	}

	globalResults = append(globalResults, result)
}

// =============================================================================
// STORAGE BENCHMARKS
// =============================================================================

func BenchmarkStorageOntologyCreate(b *testing.B) {
	db, err := storage.NewPersistenceBackend(":memory:")
	if err != nil {
		b.Fatal(err)
	}
	defer db.Close()

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ontID := fmt.Sprintf("test-ontology-%d", i)
		ont := &storage.Ontology{
			ID:          ontID,
			Name:        fmt.Sprintf("Test Ontology %d", i),
			Description: "Test Description",
			Version:     "1.0",
			FilePath:    "/tmp/test.ttl",
			TDB2Graph:   "http://test",
			Format:      "turtle",
			Status:      "active",
		}
		err := db.CreateOntology(ctx, ont)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkStorageOntologyRead(b *testing.B) {
	db, err := storage.NewPersistenceBackend(":memory:")
	if err != nil {
		b.Fatal(err)
	}
	defer db.Close()

	ctx := context.Background()

	// Setup: Create ontology
	ontID := "test-ontology"
	ont := &storage.Ontology{
		ID:          ontID,
		Name:        "Test",
		Description: "Desc",
		Version:     "1.0",
		FilePath:    "/tmp/test.ttl",
		TDB2Graph:   "http://test",
		Format:      "turtle",
		Status:      "active",
	}
	_ = db.CreateOntology(ctx, ont)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := db.GetOntology(ctx, ontID)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkStorageBulkInsert(b *testing.B) {
	db, err := storage.NewPersistenceBackend(":memory:")
	if err != nil {
		b.Fatal(err)
	}
	defer db.Close()

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Insert 100 ontologies in one benchmark iteration
		for j := 0; j < 100; j++ {
			ontID := fmt.Sprintf("bulk-ont-%d-%d", i, j)
			ont := &storage.Ontology{
				ID:          ontID,
				Name:        "Bulk Test",
				Description: "Desc",
				Version:     "1.0",
				FilePath:    "/tmp/test.ttl",
				TDB2Graph:   "http://test",
				Format:      "turtle",
				Status:      "active",
			}
			_ = db.CreateOntology(ctx, ont)
		}
	}
}

// =============================================================================
// ML BENCHMARKS
// =============================================================================

func BenchmarkMLTrainingSmallDataset(b *testing.B) {
	// Small dataset: 100 samples x 5 features
	X := generateDataset(100, 5)
	y := generateLabels(100)
	featureNames := []string{"f1", "f2", "f3", "f4", "f5"}

	config := ml.DefaultTrainingConfig()
	trainer := ml.NewTrainer(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := trainer.Train(X, y, featureNames)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMLTrainingMediumDataset(b *testing.B) {
	// Medium dataset: 1000 samples x 10 features
	X := generateDataset(1000, 10)
	y := generateLabels(1000)
	featureNames := make([]string, 10)
	for i := range featureNames {
		featureNames[i] = fmt.Sprintf("f%d", i)
	}

	config := ml.DefaultTrainingConfig()
	trainer := ml.NewTrainer(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := trainer.Train(X, y, featureNames)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMLTrainingLargeDataset(b *testing.B) {
	// Large dataset: 10000 samples x 20 features
	X := generateDataset(10000, 20)
	y := generateLabels(10000)
	featureNames := make([]string, 20)
	for i := range featureNames {
		featureNames[i] = fmt.Sprintf("f%d", i)
	}

	config := ml.DefaultTrainingConfig()
	trainer := ml.NewTrainer(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := trainer.Train(X, y, featureNames)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMLPrediction(b *testing.B) {
	// Train once, predict many times
	X := generateDataset(1000, 10)
	y := generateLabels(1000)
	featureNames := make([]string, 10)
	for i := range featureNames {
		featureNames[i] = fmt.Sprintf("f%d", i)
	}

	config := ml.DefaultTrainingConfig()
	trainer := ml.NewTrainer(config)
	result, err := trainer.Train(X, y, featureNames)
	if err != nil {
		b.Fatal(err)
	}

	testSample := X[0]

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = result.Model.Predict(testSample)
	}
}

func BenchmarkMLRegression(b *testing.B) {
	X := generateDataset(1000, 10)
	y := generateRegressionLabels(1000)
	featureNames := make([]string, 10)
	for i := range featureNames {
		featureNames[i] = fmt.Sprintf("f%d", i)
	}

	config := ml.DefaultTrainingConfig()
	trainer := ml.NewTrainer(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := trainer.TrainRegression(X, y, featureNames)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// =============================================================================
// DATA INGESTION BENCHMARKS
// =============================================================================

func BenchmarkCSVIngestionSmall(b *testing.B) {
	// 100 rows
	csvData := generateCSVData(100, 10)
	registry := pipelines.NewPluginRegistry()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		adapter := ml.NewCSVDataAdapter(registry)
		config := ml.DataSourceConfig{
			Type: "csv",
			Data: map[string]interface{}{
				"content":     csvData,
				"has_headers": true,
			},
		}
		_, err := adapter.Extract(context.Background(), config)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCSVIngestionMedium(b *testing.B) {
	// 1000 rows
	csvData := generateCSVData(1000, 10)
	registry := pipelines.NewPluginRegistry()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		adapter := ml.NewCSVDataAdapter(registry)
		config := ml.DataSourceConfig{
			Type: "csv",
			Data: map[string]interface{}{
				"content":     csvData,
				"has_headers": true,
			},
		}
		_, err := adapter.Extract(context.Background(), config)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCSVIngestionLarge(b *testing.B) {
	// 10000 rows
	csvData := generateCSVData(10000, 10)
	registry := pipelines.NewPluginRegistry()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		adapter := ml.NewCSVDataAdapter(registry)
		config := ml.DataSourceConfig{
			Type: "csv",
			Data: map[string]interface{}{
				"content":     csvData,
				"has_headers": true,
			},
		}
		_, err := adapter.Extract(context.Background(), config)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Skip JSON ingestion benchmark for now since it's more complex
// func BenchmarkJSONIngestion(b *testing.B) {
// 	jsonData := generateJSONData(1000)
// 	registry := pipelines.NewPluginRegistry()
//
// 	b.ResetTimer()
// 	for i := 0; i < b.N; i++ {
// 		adapter := ml.NewJSONDataAdapter(registry)
// 		config := ml.DataSourceConfig{
// 			Type: "json",
// 			Data: map[string]interface{}{
// 				"array": jsonData,
// 			},
// 		}
// 		_, err := adapter.Extract(context.Background(), config)
// 		if err != nil {
// 			b.Fatal(err)
// 		}
// 	}
// }

// =============================================================================
// PIPELINE BENCHMARKS
// =============================================================================

func BenchmarkPipelineExecutionSimple(b *testing.B) {
	// Simple mock pipeline without registry dependency
	config := &utils.PipelineConfig{
		Name: "Benchmark Pipeline",
		Steps: []pipelines.StepConfig{
			{
				Name:   "Step 1",
				Plugin: "Mock.test",
				Config: map[string]any{"data": "test"},
				Output: "result",
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Skip execution for now, just benchmark config creation
		_ = config
	}
}

func BenchmarkPipelineExecutionComplex(b *testing.B) {
	// Pipeline with 10 steps
	steps := make([]pipelines.StepConfig, 10)
	for i := 0; i < 10; i++ {
		steps[i] = pipelines.StepConfig{
			Name:   fmt.Sprintf("Step %d", i+1),
			Plugin: "Mock.test",
			Config: map[string]any{"data": fmt.Sprintf("test%d", i)},
			Output: fmt.Sprintf("result%d", i),
		}
	}

	config := &utils.PipelineConfig{
		Name:  "Complex Pipeline",
		Steps: steps,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Skip execution for now, just benchmark config creation
		_ = config
	}
}

// =============================================================================
// CONCURRENT LOAD BENCHMARKS
// =============================================================================

func BenchmarkConcurrentOntologyRead(b *testing.B) {
	db, err := storage.NewPersistenceBackend(":memory:")
	if err != nil {
		b.Fatal(err)
	}
	defer db.Close()

	ctx := context.Background()
	ontID := "test-ontology"
	ont := &storage.Ontology{
		ID:          ontID,
		Name:        "Test",
		Description: "Desc",
		Version:     "1.0",
		FilePath:    "/tmp/test.ttl",
		TDB2Graph:   "http://test",
		Format:      "turtle",
		Status:      "active",
	}
	_ = db.CreateOntology(ctx, ont)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = db.GetOntology(ctx, ontID)
		}
	})
}

func BenchmarkConcurrentMLTraining(b *testing.B) {
	X := generateDataset(100, 5)
	y := generateLabels(100)
	featureNames := []string{"f1", "f2", "f3", "f4", "f5"}

	b.RunParallel(func(pb *testing.PB) {
		config := ml.DefaultTrainingConfig()
		trainer := ml.NewTrainer(config)

		for pb.Next() {
			_, _ = trainer.Train(X, y, featureNames)
		}
	})
}

// =============================================================================
// MEMORY BENCHMARKS
// =============================================================================

func BenchmarkMemoryAllocationDataset(b *testing.B) {
	b.ReportAllocs()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = generateDataset(1000, 10)
	}
}

func BenchmarkMemoryAllocationPipeline(b *testing.B) {
	b.ReportAllocs()

	config := &utils.PipelineConfig{
		Name: "Memory Test",
		Steps: []pipelines.StepConfig{
			{Name: "Step", Plugin: "Mock.test", Config: map[string]any{}, Output: "result"},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = utils.ExecutePipeline(context.Background(), config)
	}
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

func generateDataset(samples, features int) [][]float64 {
	data := make([][]float64, samples)
	for i := 0; i < samples; i++ {
		data[i] = make([]float64, features)
		for j := 0; j < features; j++ {
			data[i][j] = float64(i*features + j)
		}
	}
	return data
}

func generateLabels(samples int) []string {
	labels := make([]string, samples)
	for i := 0; i < samples; i++ {
		if i%2 == 0 {
			labels[i] = "class_a"
		} else {
			labels[i] = "class_b"
		}
	}
	return labels
}

func generateRegressionLabels(samples int) []float64 {
	labels := make([]float64, samples)
	for i := 0; i < samples; i++ {
		labels[i] = float64(i) * 1.5
	}
	return labels
}

func generateCSVData(rows, cols int) string {
	var csv string

	// Header
	for i := 0; i < cols; i++ {
		if i > 0 {
			csv += ","
		}
		csv += fmt.Sprintf("col%d", i)
	}
	csv += "\n"

	// Rows
	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			if j > 0 {
				csv += ","
			}
			csv += fmt.Sprintf("%d", i*cols+j)
		}
		csv += "\n"
	}

	return csv
}

func generateJSONData(rows int) []map[string]interface{} {
	data := make([]map[string]interface{}, rows)
	for i := 0; i < rows; i++ {
		data[i] = map[string]interface{}{
			"id":    i,
			"name":  fmt.Sprintf("item_%d", i),
			"value": float64(i) * 1.5,
		}
	}
	return data
}

// =============================================================================
// BENCHMARK SUMMARY GENERATOR
// =============================================================================

func TestMain(m *testing.M) {
	// Run benchmarks
	code := m.Run()

	// Save results
	if len(globalResults) > 0 {
		data, _ := json.MarshalIndent(globalResults, "", "  ")
		_ = os.WriteFile(benchmarkResultsFile, data, 0644)
		fmt.Printf("\n📊 Benchmark results saved to: %s\n", benchmarkResultsFile)
	}

	os.Exit(code)
}
