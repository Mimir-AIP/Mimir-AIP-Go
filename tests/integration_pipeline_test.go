package tests

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
	"github.com/Mimir-AIP/Mimir-AIP-Go/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPipelineMultiStepExecution tests complete pipeline execution with multiple steps
func TestPipelineMultiStepExecution(t *testing.T) {
	// Setup test environment
	tempDir, err := os.MkdirTemp("", "mimir_pipeline_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create plugin registry
	registry := pipelines.NewPluginRegistry()

	// Register test plugins
	apiPlugin := &utils.RealAPIPlugin{}
	htmlPlugin := &utils.MockHTMLPlugin{}

	err = registry.RegisterPlugin(apiPlugin)
	require.NoError(t, err)
	err = registry.RegisterPlugin(htmlPlugin)
	require.NoError(t, err)

	t.Run("Sequential Step Execution", func(t *testing.T) {
		// Create a multi-step pipeline
		pipelineContent := `
name: "Multi-Step Test Pipeline"
description: "Tests sequential execution of pipeline steps"
steps:
  - name: "Fetch API Data"
    plugin: "Input.api"
    config:
      url: "https://httpbin.org/json"
      method: "GET"
      headers:
        Accept: "application/json"
        User-Agent: "Mimir-AIP-Test/1.0"
    output: "api_response"
  
  - name: "Fetch Additional Data"
    plugin: "Input.api"
    config:
      url: "https://httpbin.org/uuid"
      method: "GET"
    output: "uuid_data"
  
  - name: "Generate HTML Report"
    plugin: "Output.html"
    config:
      title: "Integration Test Report"
      include_timestamp: true
    output: "html_output"
`

		pipelineFile := filepath.Join(tempDir, "multi_step.yaml")
		err := os.WriteFile(pipelineFile, []byte(pipelineContent), 0644)
		require.NoError(t, err)

		// Parse and execute pipeline
		config, err := utils.ParsePipeline(pipelineFile)
		require.NoError(t, err)
		assert.Equal(t, "Multi-Step Test Pipeline", config.Name)
		assert.Len(t, config.Steps, 3)

		ctx := context.Background()
		result, err := utils.ExecutePipelineWithRegistry(ctx, config, registry)
		require.NoError(t, err)

		// Verify execution success
		require.True(t, result.Success, "Pipeline execution should succeed")
		assert.Empty(t, result.Error)
		assert.NotEmpty(t, result.ExecutedAt)

		// Verify all steps produced outputs
		expectedOutputs := []string{"api_response", "uuid_data", "html_output"}
		for _, output := range expectedOutputs {
			value, exists := result.Context.Get(output)
			assert.True(t, exists, "Output %s should exist in context", output)
			assert.NotNil(t, value, "Output %s should not be nil", output)
		}

		// Verify API response structure
		apiResp, exists := result.Context.Get("api_response")
		require.True(t, exists)

		apiData, ok := apiResp.(map[string]interface{})
		require.True(t, ok, "API response should be a map")
		assert.Contains(t, apiData, "status_code")
		assert.Contains(t, apiData, "body")
	})
}

// TestPipelineContextPassing tests context passing between steps
func TestPipelineContextPassing(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "mimir_context_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	registry := pipelines.NewPluginRegistry()
	err = registry.RegisterPlugin(&utils.RealAPIPlugin{})
	require.NoError(t, err)
	err = registry.RegisterPlugin(&utils.MockHTMLPlugin{})
	require.NoError(t, err)

	t.Run("Context Preservation Across Steps", func(t *testing.T) {
		pipelineContent := `
name: "Context Test Pipeline"
description: "Tests context passing between steps"
steps:
  - name: "Initial Data Fetch"
    plugin: "Input.api"
    config:
      url: "https://httpbin.org/json"
      method: "GET"
    output: "initial_data"
  
  - name: "Process with Context"
    plugin: "Output.html"
    config:
      title: "Context Test"
    output: "processed"
`

		pipelineFile := filepath.Join(tempDir, "context_test.yaml")
		err := os.WriteFile(pipelineFile, []byte(pipelineContent), 0644)
		require.NoError(t, err)

		config, err := utils.ParsePipeline(pipelineFile)
		require.NoError(t, err)

		// Create initial context with custom data
		ctx := context.Background()
		globalContext := pipelines.NewPluginContext()
		globalContext.Set("test_id", "context_test_001")
		globalContext.Set("timestamp", time.Now().Unix())
		globalContext.Set("metadata", map[string]interface{}{
			"version": "1.0",
			"mode":    "test",
			"tags":    []string{"integration", "context"},
		})

		result, err := utils.ExecutePipelineWithRegistry(ctx, config, registry)
		require.NoError(t, err)
		require.True(t, result.Success)

		// Verify initial context is preserved
		testID, exists := result.Context.Get("test_id")
		assert.True(t, exists, "Initial context should be preserved")
		assert.Equal(t, "context_test_001", testID)

		timestamp, exists := result.Context.Get("timestamp")
		assert.True(t, exists)
		assert.NotNil(t, timestamp)

		metadata, exists := result.Context.Get("metadata")
		assert.True(t, exists)
		assert.NotNil(t, metadata)

		// Verify step outputs are added to context
		initialData, exists := result.Context.Get("initial_data")
		assert.True(t, exists, "Step outputs should be added to context")
		assert.NotNil(t, initialData)

		processed, exists := result.Context.Get("processed")
		assert.True(t, exists)
		assert.NotNil(t, processed)
	})
}

// TestPipelineErrorHandling tests error handling and recovery
func TestPipelineErrorHandling(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "mimir_error_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	registry := pipelines.NewPluginRegistry()
	err = registry.RegisterPlugin(&utils.RealAPIPlugin{})
	require.NoError(t, err)

	t.Run("Invalid URL Error Handling", func(t *testing.T) {
		pipelineContent := `
name: "Error Test Pipeline"
description: "Tests error handling in pipeline execution"
steps:
  - name: "Valid Step"
    plugin: "Input.api"
    config:
      url: "https://httpbin.org/status/200"
      method: "GET"
    output: "valid_result"
  
  - name: "Invalid URL Step"
    plugin: "Input.api"
    config:
      url: "invalid://not-a-valid-url"
      method: "GET"
    output: "invalid_result"
  
  - name: "Should Not Execute"
    plugin: "Input.api"
    config:
      url: "https://httpbin.org/json"
      method: "GET"
    output: "skipped_result"
`

		pipelineFile := filepath.Join(tempDir, "error_test.yaml")
		err := os.WriteFile(pipelineFile, []byte(pipelineContent), 0644)
		require.NoError(t, err)

		config, err := utils.ParsePipeline(pipelineFile)
		require.NoError(t, err)

		ctx := context.Background()
		result, err := utils.ExecutePipelineWithRegistry(ctx, config, registry)
		require.NoError(t, err)

		// Pipeline should fail
		require.False(t, result.Success, "Pipeline with invalid step should fail")
		assert.NotEmpty(t, result.Error)
		assert.Contains(t, result.Error, "Invalid URL Step", "Error should mention failing step")

		// Verify results from successful steps are preserved
		validResult, exists := result.Context.Get("valid_result")
		assert.True(t, exists, "Valid step results should be preserved")
		assert.NotNil(t, validResult)

		// Verify failed and subsequent steps don't have results
		_, exists = result.Context.Get("invalid_result")
		assert.False(t, exists, "Failed step should not have results")

		_, exists = result.Context.Get("skipped_result")
		assert.False(t, exists, "Steps after failure should not execute")
	})

	t.Run("Missing Plugin Error", func(t *testing.T) {
		pipelineContent := `
name: "Missing Plugin Test"
steps:
  - name: "Invalid Plugin"
    plugin: "NonExistent.plugin"
    config:
      test: "data"
    output: "result"
`

		pipelineFile := filepath.Join(tempDir, "missing_plugin.yaml")
		err := os.WriteFile(pipelineFile, []byte(pipelineContent), 0644)
		require.NoError(t, err)

		config, err := utils.ParsePipeline(pipelineFile)
		require.NoError(t, err)

		ctx := context.Background()
		result, err := utils.ExecutePipelineWithRegistry(ctx, config, registry)
		require.NoError(t, err)

		require.False(t, result.Success)
		assert.Contains(t, result.Error, "NonExistent.plugin")
	})

	t.Run("Invalid Configuration Error", func(t *testing.T) {
		pipelineContent := `
name: "Invalid Config Test"
steps:
  - name: "Missing Required Config"
    plugin: "Input.api"
    config:
      method: "GET"
    output: "result"
`

		pipelineFile := filepath.Join(tempDir, "invalid_config.yaml")
		err := os.WriteFile(pipelineFile, []byte(pipelineContent), 0644)
		require.NoError(t, err)

		config, err := utils.ParsePipeline(pipelineFile)
		require.NoError(t, err)

		ctx := context.Background()
		result, err := utils.ExecutePipelineWithRegistry(ctx, config, registry)
		require.NoError(t, err)

		require.False(t, result.Success)
		assert.Contains(t, result.Error, "url")
	})
}

// TestPipelinePerformance tests pipeline performance characteristics
func TestPipelinePerformance(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "mimir_perf_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	registry := pipelines.NewPluginRegistry()
	err = registry.RegisterPlugin(&utils.RealAPIPlugin{})
	require.NoError(t, err)
	err = registry.RegisterPlugin(&utils.MockHTMLPlugin{})
	require.NoError(t, err)

	t.Run("Pipeline Execution Timing", func(t *testing.T) {
		pipelineContent := `
name: "Performance Test Pipeline"
steps:
  - name: "API Call"
    plugin: "Input.api"
    config:
      url: "https://httpbin.org/delay/1"
      method: "GET"
    output: "result"
`

		pipelineFile := filepath.Join(tempDir, "perf_test.yaml")
		err := os.WriteFile(pipelineFile, []byte(pipelineContent), 0644)
		require.NoError(t, err)

		config, err := utils.ParsePipeline(pipelineFile)
		require.NoError(t, err)

		const numExecutions = 5
		var durations []time.Duration

		for i := 0; i < numExecutions; i++ {
			start := time.Now()

			ctx := context.Background()
			result, err := utils.ExecutePipelineWithRegistry(ctx, config, registry)

			duration := time.Since(start)
			durations = append(durations, duration)

			require.NoError(t, err)
			require.True(t, result.Success, "Execution %d should succeed", i)
		}

		// Calculate statistics
		var total time.Duration
		min := durations[0]
		max := durations[0]

		for _, d := range durations {
			total += d
			if d < min {
				min = d
			}
			if d > max {
				max = d
			}
		}

		avg := total / time.Duration(numExecutions)

		t.Logf("Pipeline Performance Statistics:")
		t.Logf("  Executions: %d", numExecutions)
		t.Logf("  Average: %v", avg)
		t.Logf("  Min: %v", min)
		t.Logf("  Max: %v", max)
		t.Logf("  Variance: %v", max-min)

		// Performance assertions
		assert.Less(t, avg, 5*time.Second, "Average execution should be under 5s")
		assert.Less(t, max-min, 2*time.Second, "Variance should be reasonable")
	})
}

// TestPipelineConcurrentExecution tests concurrent pipeline executions
func TestPipelineConcurrentExecution(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "mimir_concurrent_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	registry := pipelines.NewPluginRegistry()
	err = registry.RegisterPlugin(&utils.RealAPIPlugin{})
	require.NoError(t, err)

	t.Run("Multiple Concurrent Pipelines", func(t *testing.T) {
		pipelineContent := `
name: "Concurrent Test Pipeline"
steps:
  - name: "API Call"
    plugin: "Input.api"
    config:
      url: "https://httpbin.org/delay/1"
      method: "GET"
    output: "result"
`

		pipelineFile := filepath.Join(tempDir, "concurrent.yaml")
		err := os.WriteFile(pipelineFile, []byte(pipelineContent), 0644)
		require.NoError(t, err)

		config, err := utils.ParsePipeline(pipelineFile)
		require.NoError(t, err)

		const numExecutions = 10
		var wg sync.WaitGroup
		results := make(chan error, numExecutions)

		startTime := time.Now()

		for i := 0; i < numExecutions; i++ {
			wg.Add(1)
			go func(execNum int) {
				defer wg.Done()

				ctx := context.Background()
				result, err := utils.ExecutePipelineWithRegistry(ctx, config, registry)

				if err != nil {
					results <- err
				} else if !result.Success {
					results <- fmt.Errorf("execution %d failed: %s", execNum, result.Error)
				} else {
					results <- nil
				}
			}(i)
		}

		wg.Wait()
		close(results)

		duration := time.Since(startTime)
		t.Logf("Completed %d concurrent executions in %v", numExecutions, duration)

		// Check results
		successCount := 0
		for err := range results {
			if err == nil {
				successCount++
			} else {
				t.Logf("Execution failed: %v", err)
			}
		}

		t.Logf("Success rate: %d/%d (%.1f%%)", successCount, numExecutions,
			float64(successCount)/float64(numExecutions)*100)

		assert.Greater(t, successCount, numExecutions*8/10,
			"At least 80%% of concurrent executions should succeed")
	})
}

// TestPipelineComplexWorkflow tests a realistic complex workflow
func TestPipelineComplexWorkflow(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "mimir_complex_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	registry := pipelines.NewPluginRegistry()
	err = registry.RegisterPlugin(&utils.RealAPIPlugin{})
	require.NoError(t, err)
	err = registry.RegisterPlugin(&utils.MockHTMLPlugin{})
	require.NoError(t, err)

	t.Run("Complex Multi-Source Data Aggregation", func(t *testing.T) {
		pipelineContent := `
name: "Complex Data Aggregation Pipeline"
description: "Fetches data from multiple sources and aggregates"
steps:
  - name: "Fetch User Data"
    plugin: "Input.api"
    config:
      url: "https://httpbin.org/json"
      method: "GET"
    output: "user_data"
  
  - name: "Fetch UUID"
    plugin: "Input.api"
    config:
      url: "https://httpbin.org/uuid"
      method: "GET"
    output: "uuid_data"
  
  - name: "Fetch Headers Info"
    plugin: "Input.api"
    config:
      url: "https://httpbin.org/headers"
      method: "GET"
    output: "headers_data"
  
  - name: "Generate Aggregated Report"
    plugin: "Output.html"
    config:
      title: "Aggregated Data Report"
      include_all_context: true
    output: "final_report"
`

		pipelineFile := filepath.Join(tempDir, "complex.yaml")
		err := os.WriteFile(pipelineFile, []byte(pipelineContent), 0644)
		require.NoError(t, err)

		config, err := utils.ParsePipeline(pipelineFile)
		require.NoError(t, err)

		ctx := context.Background()
		result, err := utils.ExecutePipelineWithRegistry(ctx, config, registry)
		require.NoError(t, err)

		require.True(t, result.Success, "Complex pipeline should succeed")

		// Verify all data sources were fetched
		dataSources := []string{"user_data", "uuid_data", "headers_data", "final_report"}
		for _, source := range dataSources {
			value, exists := result.Context.Get(source)
			assert.True(t, exists, "Data source %s should exist", source)
			assert.NotNil(t, value, "Data source %s should not be nil", source)
		}

		// Verify context contains expected structure
		keys := result.Context.Keys()
		assert.Greater(t, len(keys), 3, "Context should have multiple keys")

		t.Logf("Complex pipeline completed successfully with %d context keys", len(keys))
	})
}
