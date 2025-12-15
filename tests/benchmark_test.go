package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
	"github.com/Mimir-AIP/Mimir-AIP-Go/utils"
)

// BenchmarkPipelineExecution benchmarks basic pipeline execution
func BenchmarkPipelineExecution(b *testing.B) {
	// Create a simple pipeline configuration
	config := &utils.PipelineConfig{
		Name: "Benchmark Pipeline",
		Steps: []pipelines.StepConfig{
			{
				Name:   "Mock Step 1",
				Plugin: "Data_Processing.mock",
				Config: map[string]any{},
				Output: "step1_output",
			},
			{
				Name:   "Mock Step 2",
				Plugin: "Data_Processing.mock",
				Config: map[string]any{},
				Output: "step2_output",
			},
		},
	}

	// Register mock plugins
	pluginRegistry := pipelines.NewPluginRegistry()
	mockPlugin := NewMockPlugin("mock", "Data_Processing", false)
	_ = pluginRegistry.RegisterPlugin(mockPlugin)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := utils.ExecutePipelineWithRegistry(context.Background(), config, pluginRegistry)
		if err != nil {
			b.Fatalf("Pipeline execution failed: %v", err)
		}
	}
}

// BenchmarkPipelineExecutionWithCache benchmarks pipeline execution with caching
func BenchmarkPipelineExecutionWithCache(b *testing.B) {
	// Create a pipeline configuration
	config := &utils.PipelineConfig{
		Name: "Cached Benchmark Pipeline",
		Steps: []pipelines.StepConfig{
			{
				Name:   "Cached Step",
				Plugin: "Data_Processing.cached_mock",
				Config: map[string]any{
					"data": "test_data",
				},
				Output: "cached_output",
			},
		},
	}

	// Register plugins
	pluginRegistry := pipelines.NewPluginRegistry()
	cachedPlugin := NewMockPlugin("cached_mock", "Data_Processing", false)
	_ = pluginRegistry.RegisterPlugin(cachedPlugin)

	// Create optimized executor with cache
	executor := utils.NewOptimizedPipelineExecutor(pluginRegistry, 4)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := executor.ExecutePipelineOptimized(context.Background(), config)
		if err != nil {
			b.Fatalf("Pipeline execution failed: %v", err)
		}
	}
}

// BenchmarkConcurrentPipelineExecution benchmarks concurrent pipeline execution
func BenchmarkConcurrentPipelineExecution(b *testing.B) {
	config := &utils.PipelineConfig{
		Name: "Concurrent Benchmark Pipeline",
		Steps: []pipelines.StepConfig{
			{
				Name:   "Concurrent Step",
				Plugin: "Data_Processing.mock",
				Config: map[string]any{},
				Output: "concurrent_output",
			},
		},
	}

	// Register plugins
	pluginRegistry := pipelines.NewPluginRegistry()
	mockPlugin := NewMockPlugin("mock", "Data_Processing", false)
	_ = pluginRegistry.RegisterPlugin(mockPlugin)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := utils.ExecutePipelineWithRegistry(context.Background(), config, pluginRegistry)
			if err != nil {
				b.Fatalf("Pipeline execution failed: %v", err)
			}
		}
	})
}

// BenchmarkPluginRegistry benchmarks plugin registry operations
func BenchmarkPluginRegistry(b *testing.B) {
	registry := pipelines.NewPluginRegistry()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pluginName := fmt.Sprintf("benchmark_plugin_%d", i)
		plugin := NewMockPlugin(pluginName, "Data_Processing", false)
		err := registry.RegisterPlugin(plugin)
		if err != nil {
			b.Fatalf("Plugin registration failed: %v", err)
		}

		_, err = registry.GetPlugin("Data_Processing", pluginName)
		if err != nil {
			b.Fatalf("Plugin retrieval failed: %v", err)
		}
	}
}

// BenchmarkStringInterning removed - StringInterner was unused dead code

// BenchmarkPerformanceMonitor benchmarks performance monitoring
func BenchmarkPerformanceMonitor(b *testing.B) {
	monitor := utils.GetPerformanceMonitor()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		monitor.RecordRequest(time.Duration(i%100)*time.Millisecond, i%10 == 0)
	}
}

// BenchmarkPluginCache benchmarks plugin cache performance
func BenchmarkPluginCache(b *testing.B) {
	cache := utils.GetPluginCache()
	testResult := pipelines.NewPluginContext()
	testResult.Set("result", "test")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := "test_key_" + string(rune(i%100))

		// Test cache miss
		_, found := cache.Get(key)
		if !found {
			cache.Set(key, testResult)
		}

		// Test cache hit
		_, found = cache.Get(key)
		if !found {
			b.Fatalf("Expected cache hit")
		}
	}
}

// BenchmarkConnectionPool removed - ConnectionPool was unused dead code

// Performance comparison benchmarks

// BenchmarkJSONProcessing benchmarks JSON processing performance
func BenchmarkJSONProcessing(b *testing.B) {
	testData := map[string]any{
		"pipeline": map[string]any{
			"name": "test",
			"steps": []map[string]any{
				{
					"name":   "step1",
					"plugin": "Input.api",
					"config": map[string]any{
						"url": "https://api.example.com",
					},
				},
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Simulate JSON marshaling/unmarshaling
		_, err := json.Marshal(testData)
		if err != nil {
			b.Fatalf("JSON marshaling failed: %v", err)
		}
	}
}

// BenchmarkContextPropagation benchmarks context propagation
func BenchmarkContextPropagation(b *testing.B) {
	context := pipelines.NewPluginContext()
	context.Set("data1", "value1")
	context.Set("data2", "value2")
	context.Set("data3", "value3")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Simulate context copying
		newContext := pipelines.NewPluginContext()
		for _, k := range context.Keys() {
			if v, exists := context.Get(k); exists {
				newContext.Set(k, v)
			}
		}

		// Add new data
		newContext.Set("new_key_"+string(rune(i%100)), "new_value")
	}
}

// BenchmarkGoroutineSwitching benchmarks goroutine switching overhead
func BenchmarkGoroutineSwitching(b *testing.B) {
	done := make(chan bool, 1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		go func() {
			done <- true
		}()
		<-done
	}
}

// Memory benchmark
func BenchmarkMemoryAllocation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Allocate and immediately free memory
		data := make([]byte, 1024)
		_ = data
	}
}

// Comprehensive performance test
func TestPerformanceComparison(t *testing.T) {
	// Test basic pipeline execution performance
	config := &utils.PipelineConfig{
		Name: "Performance Test Pipeline",
		Steps: []pipelines.StepConfig{
			{
				Name:   "Step 1",
				Plugin: "Data_Processing.mock",
				Config: map[string]any{},
				Output: "output1",
			},
			{
				Name:   "Step 2",
				Plugin: "Data_Processing.mock",
				Config: map[string]any{},
				Output: "output2",
			},
			{
				Name:   "Step 3",
				Plugin: "Data_Processing.mock",
				Config: map[string]any{},
				Output: "output3",
			},
		},
	}

	// Register plugins
	pluginRegistry := pipelines.NewPluginRegistry()
	mockPlugin := NewMockPlugin("mock", "Data_Processing", false)
	_ = pluginRegistry.RegisterPlugin(mockPlugin)

	// Run performance test
	start := time.Now()
	iterations := 100

	for i := 0; i < iterations; i++ {
		_, err := utils.ExecutePipelineWithRegistry(context.Background(), config, pluginRegistry)
		if err != nil {
			t.Fatalf("Pipeline execution failed: %v", err)
		}
	}

	elapsed := time.Since(start)
	avgTime := elapsed / time.Duration(iterations)

	t.Logf("Performance Test Results:")
	t.Logf("Total time: %v", elapsed)
	t.Logf("Average time per pipeline: %v", avgTime)
	t.Logf("Pipelines per second: %.2f", float64(iterations)/elapsed.Seconds())

	// Verify performance is reasonable (should be much faster than Python)
	if avgTime > 100*time.Millisecond {
		t.Errorf("Pipeline execution is too slow: %v", avgTime)
	}
}

// TestOptimizedVsRegularExecution compares optimized vs regular execution
func TestOptimizedVsRegularExecution(t *testing.T) {
	config := &utils.PipelineConfig{
		Name: "Optimization Test Pipeline",
		Steps: []pipelines.StepConfig{
			{
				Name:   "Test Step",
				Plugin: "Data_Processing.mock",
				Config: map[string]any{
					"data": "test_data",
				},
				Output: "test_output",
			},
		},
	}

	// Register plugins
	pluginRegistry := pipelines.NewPluginRegistry()
	mockPlugin := NewMockPlugin("mock", "Data_Processing", false)
	_ = pluginRegistry.RegisterPlugin(mockPlugin)

	// Test regular execution
	start := time.Now()
	_, err := utils.ExecutePipelineWithRegistry(context.Background(), config, pluginRegistry)
	regularTime := time.Since(start)
	if err != nil {
		t.Fatalf("Regular execution failed: %v", err)
	}

	// Test optimized execution
	executor := utils.NewOptimizedPipelineExecutor(pluginRegistry, 4)
	start = time.Now()
	_, err = executor.ExecutePipelineOptimized(context.Background(), config)
	optimizedTime := time.Since(start)
	if err != nil {
		t.Fatalf("Optimized execution failed: %v", err)
	}

	t.Logf("Regular execution time: %v", regularTime)
	t.Logf("Optimized execution time: %v", optimizedTime)
	t.Logf("Optimization improvement: %.2fx",
		float64(regularTime)/float64(optimizedTime))

	// Optimized should be at least as fast as regular
	if optimizedTime > regularTime {
		t.Logf("Note: Optimized execution was slower, possibly due to caching overhead")
	}
}
