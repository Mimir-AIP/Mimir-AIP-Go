package main

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
	"github.com/Mimir-AIP/Mimir-AIP-Go/utils"
)

// BenchmarkSuite runs comprehensive performance benchmarks
func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run benchmark_suite.go <benchmark_type>")
		fmt.Println("Available benchmarks:")
		fmt.Println("  - plugin_execution: Benchmark plugin execution performance")
		fmt.Println("  - pipeline_execution: Benchmark pipeline execution performance")
		fmt.Println("  - memory_usage: Benchmark memory usage patterns")
		fmt.Println("  - concurrent_load: Benchmark concurrent load handling")
		os.Exit(1)
	}

	benchmarkType := os.Args[1]

	switch benchmarkType {
	case "plugin_execution":
		benchmarkPluginExecution()
	case "pipeline_execution":
		benchmarkPipelineExecution()
	case "memory_usage":
		benchmarkMemoryUsage()
	case "concurrent_load":
		benchmarkConcurrentLoad()
	default:
		fmt.Printf("Unknown benchmark type: %s\n", benchmarkType)
		os.Exit(1)
	}
}

func benchmarkPluginExecution() {
	fmt.Println("=== Plugin Execution Benchmark ===")

	registry := pipelines.NewPluginRegistry()

	// Register mock plugins
	registry.RegisterPlugin(&MockPlugin{name: "test1", pluginType: "Data_Processing"})
	registry.RegisterPlugin(&MockPlugin{name: "test2", pluginType: "Data_Processing"})

	stepConfig := pipelines.StepConfig{
		Name:   "Benchmark Step",
		Plugin: "Data_Processing.test1",
		Config: map[string]any{"operation": "echo", "text": "benchmark data"},
		Output: "result",
	}

	// Warm up
	for i := 0; i < 10; i++ {
		plugin, _ := registry.GetPlugin("Data_Processing", "test1")
		plugin.ExecuteStep(context.Background(), stepConfig, pipelines.NewPluginContext())
	}

	// Benchmark
	start := time.Now()
	iterations := 1000

	for i := 0; i < iterations; i++ {
		plugin, _ := registry.GetPlugin("Data_Processing", "test1")
		plugin.ExecuteStep(context.Background(), stepConfig, pipelines.NewPluginContext())
	}

	// Benchmark
	start = time.Now()
	iterations = 1000

	for i := 0; i < iterations; i++ {
		plugin, _ := registry.GetPlugin("Data_Processing", "test1")
		plugin.ExecuteStep(context.Background(), stepConfig, pipelines.NewPluginContext())
	}

	duration := time.Since(start)
	avgDuration := duration / time.Duration(iterations)

	fmt.Printf("Plugin Execution Results:\n")
	fmt.Printf("  Iterations: %d\n", iterations)
	fmt.Printf("  Total Time: %v\n", duration)
	fmt.Printf("  Average Time: %v\n", avgDuration)
	fmt.Printf("  Ops/sec: %.2f\n", float64(time.Second)/float64(avgDuration))
}

func benchmarkPipelineExecution() {
	fmt.Println("=== Pipeline Execution Benchmark ===")

	config := &utils.PipelineConfig{
		Name: "Benchmark Pipeline",
		Steps: []pipelines.StepConfig{
			{
				Name:   "Step 1",
				Plugin: "Data_Processing.transform",
				Config: map[string]any{"operation": "echo", "text": "step1"},
				Output: "step1_output",
			},
			{
				Name:   "Step 2",
				Plugin: "Data_Processing.transform",
				Config: map[string]any{"operation": "echo", "text": "step2"},
				Output: "step2_output",
			},
			{
				Name:   "Step 3",
				Plugin: "Data_Processing.transform",
				Config: map[string]any{"operation": "echo", "text": "step3"},
				Output: "step3_output",
			},
		},
	}

	// Warm up
	for i := 0; i < 5; i++ {
		utils.ExecutePipeline(context.Background(), config)
	}

	// Benchmark
	start := time.Now()
	iterations := 100

	for i := 0; i < iterations; i++ {
		utils.ExecutePipeline(context.Background(), config)
	}

	duration := time.Since(start)
	avgDuration := duration / time.Duration(iterations)

	fmt.Printf("Pipeline Execution Results:\n")
	fmt.Printf("  Iterations: %d\n", iterations)
	fmt.Printf("  Total Time: %v\n", duration)
	fmt.Printf("  Average Time: %v\n", avgDuration)
	fmt.Printf("  Pipelines/sec: %.2f\n", float64(time.Second)/float64(avgDuration))
}

func benchmarkMemoryUsage() {
	fmt.Println("=== Memory Usage Benchmark ===")

	// Baseline memory
	runtime.GC()
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)

	// Execute operations
	registry := pipelines.NewPluginRegistry()
	registry.RegisterPlugin(&MockPlugin{name: "memory_test", pluginType: "Data_Processing"})

	stepConfig := pipelines.StepConfig{
		Name:   "Memory Test",
		Plugin: "Data_Processing.memory_test",
		Config: map[string]any{"operation": "echo", "text": "memory test data"},
		Output: "result",
	}

	plugin, _ := registry.GetPlugin("Data_Processing", "memory_test")

	for i := 0; i < 1000; i++ {
		plugin.ExecuteStep(context.Background(), stepConfig, pipelines.NewPluginContext())
	}

	// Peak memory
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)

	// After GC
	runtime.GC()
	var m3 runtime.MemStats
	runtime.ReadMemStats(&m3)

	fmt.Printf("Memory Usage Results:\n")
	fmt.Printf("  Baseline Memory: %d KB\n", m1.Alloc/1024)
	fmt.Printf("  Peak Memory: %d KB\n", m2.Alloc/1024)
	fmt.Printf("  After GC Memory: %d KB\n", m3.Alloc/1024)
	fmt.Printf("  Memory Growth: %d KB\n", (m2.Alloc-m1.Alloc)/1024)
	fmt.Printf("  GC Efficiency: %.2f%%\n", float64(m2.Alloc-m3.Alloc)/float64(m2.Alloc-m1.Alloc)*100)
}

func benchmarkConcurrentLoad() {
	fmt.Println("=== Concurrent Load Benchmark ===")

	registry := pipelines.NewPluginRegistry()
	registry.RegisterPlugin(&MockPlugin{name: "concurrent_test", pluginType: "Data_Processing"})

	stepConfig := pipelines.StepConfig{
		Name:   "Concurrent Test",
		Plugin: "Data_Processing.concurrent_test",
		Config: map[string]any{"operation": "echo", "text": "concurrent test"},
		Output: "result",
	}

	concurrencyLevels := []int{1, 2, 4, 8, 16, 32}

	for _, concurrency := range concurrencyLevels {
		start := time.Now()

		// Execute concurrent operations
		done := make(chan bool, concurrency)

		for i := 0; i < concurrency; i++ {
			go func() {
				plugin, _ := registry.GetPlugin("Data_Processing", "concurrent_test")
				for j := 0; j < 100; j++ {
					plugin.ExecuteStep(context.Background(), stepConfig, pipelines.NewPluginContext())
				}
				done <- true
			}()
		}

		// Wait for completion
		for i := 0; i < concurrency; i++ {
			<-done
		}

		duration := time.Since(start)
		totalOps := concurrency * 100
		opsPerSec := float64(totalOps) / duration.Seconds()

		fmt.Printf("Concurrency Level %d:\n", concurrency)
		fmt.Printf("  Total Time: %v\n", duration)
		fmt.Printf("  Total Ops: %d\n", totalOps)
		fmt.Printf("  Ops/sec: %.2f\n", opsPerSec)
		fmt.Printf("  Avg Latency: %v\n", duration/time.Duration(totalOps))
		fmt.Println()
	}
}

// MockPlugin for benchmarking
type MockPlugin struct {
	name       string
	pluginType string
}

func (p *MockPlugin) ExecuteStep(ctx context.Context, stepConfig pipelines.StepConfig, globalContext *pipelines.PluginContext) (*pipelines.PluginContext, error) {
	// Simulate some work
	time.Sleep(100 * time.Microsecond)

	result := pipelines.NewPluginContext()
	result.Set(stepConfig.Output, map[string]any{
		"processed": true,
		"timestamp": time.Now(),
	})

	return result, nil
}

func (p *MockPlugin) GetPluginType() string                              { return p.pluginType }
func (p *MockPlugin) GetPluginName() string                              { return p.name }
func (p *MockPlugin) ValidateConfig(config map[string]any) error { return nil }
