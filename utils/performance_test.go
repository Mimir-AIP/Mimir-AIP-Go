package utils

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
	"github.com/stretchr/testify/assert"
)

func TestNewPerformanceMonitor(t *testing.T) {
	monitor := NewPerformanceMonitor()

	assert.NotNil(t, monitor)
	assert.NotNil(t, monitor.latencies)
	assert.Equal(t, int64(0), monitor.requestCount)
	assert.Equal(t, time.Duration(0), monitor.totalLatency)
	assert.Equal(t, int64(0), monitor.errorCount)
	assert.False(t, monitor.startTime.IsZero())
}

func TestPerformanceMonitorRecordRequest(t *testing.T) {
	monitor := NewPerformanceMonitor()

	// Record successful request
	monitor.RecordRequest(100*time.Millisecond, false)
	assert.Equal(t, int64(1), monitor.requestCount)
	assert.Equal(t, 100*time.Millisecond, monitor.totalLatency)
	assert.Equal(t, int64(0), monitor.errorCount)
	assert.Len(t, monitor.latencies, 1)

	// Record failed request
	monitor.RecordRequest(200*time.Millisecond, true)
	assert.Equal(t, int64(2), monitor.requestCount)
	assert.Equal(t, 300*time.Millisecond, monitor.totalLatency)
	assert.Equal(t, int64(1), monitor.errorCount)
	assert.Len(t, monitor.latencies, 2)
}

func TestPerformanceMonitorGetMetrics(t *testing.T) {
	monitor := NewPerformanceMonitor()

	// Initial metrics
	metrics := monitor.GetMetrics()
	assert.Equal(t, int64(0), metrics.TotalRequests)
	assert.Equal(t, time.Duration(0), metrics.AverageLatency)
	assert.Equal(t, time.Duration(0), metrics.P95Latency)
	assert.Equal(t, time.Duration(0), metrics.P99Latency)
	assert.Equal(t, float64(0), metrics.ErrorRate)
	assert.Equal(t, float64(0), metrics.RequestsPerSecond)

	// Add some requests
	monitor.RecordRequest(100*time.Millisecond, false)
	monitor.RecordRequest(200*time.Millisecond, false)
	monitor.RecordRequest(300*time.Millisecond, true)

	metrics = monitor.GetMetrics()
	assert.Equal(t, int64(3), metrics.TotalRequests)
	assert.Equal(t, 200*time.Millisecond, metrics.AverageLatency) // (100+200+300)/3
	assert.Equal(t, float64(1.0/3.0), metrics.ErrorRate)          // 1 error out of 3 requests
	assert.True(t, metrics.RequestsPerSecond > 0)

	// Test percentile calculations with more data
	for i := 0; i < 100; i++ {
		monitor.RecordRequest(time.Duration(i)*time.Millisecond, i%10 == 0)
	}

	metrics = monitor.GetMetrics()
	assert.Equal(t, int64(103), metrics.TotalRequests)
	assert.True(t, metrics.P95Latency > 0)
	assert.True(t, metrics.P99Latency > 0)
	assert.True(t, metrics.P99Latency >= metrics.P95Latency)
}

func TestPerformanceMonitorLatencyLimit(t *testing.T) {
	monitor := NewPerformanceMonitor()

	// Add more than 1000 requests to test limit
	for i := 0; i < 1500; i++ {
		monitor.RecordRequest(time.Duration(i)*time.Millisecond, false)
	}

	// Should only keep last 1000 latencies
	assert.Len(t, monitor.latencies, 1000)
	assert.Equal(t, int64(1500), monitor.requestCount) // But count all requests
}

func TestGetPerformanceMonitor(t *testing.T) {
	monitor1 := GetPerformanceMonitor()
	monitor2 := GetPerformanceMonitor()

	// Should return the same instance (singleton)
	assert.Same(t, monitor1, monitor2)
	assert.NotNil(t, monitor1)
}

func TestPerformanceMiddleware(t *testing.T) {
	monitor := GetPerformanceMonitor()

	// Create a test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("test response"))
	})

	// Wrap with middleware
	wrappedHandler := PerformanceMiddleware(handler)

	// Create test request
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	// Record initial metrics
	initialMetrics := monitor.GetMetrics()

	// Wait a bit to ensure measurable latency
	time.Sleep(10 * time.Millisecond)

	// Serve request
	wrappedHandler.ServeHTTP(w, req)

	// Check response
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "test response", w.Body.String())

	// Check that metrics were recorded
	finalMetrics := monitor.GetMetrics()
	assert.Equal(t, initialMetrics.TotalRequests+1, finalMetrics.TotalRequests)
	assert.True(t, finalMetrics.AverageLatency >= initialMetrics.AverageLatency)
}

func TestPerformanceMiddlewareError(t *testing.T) {
	monitor := GetPerformanceMonitor()

	// Create a handler that returns an error
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "test error", http.StatusInternalServerError)
	})

	// Wrap with middleware
	wrappedHandler := PerformanceMiddleware(handler)

	// Create test request
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	// Serve request
	wrappedHandler.ServeHTTP(w, req)

	// Check response
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	// Check that error was recorded
	metrics := monitor.GetMetrics()
	assert.True(t, metrics.ErrorRate > 0)
}

func TestResponseWriter(t *testing.T) {
	// Create a response writer wrapper
	w := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

	// Test default status code
	assert.Equal(t, http.StatusOK, rw.statusCode)

	// Test setting status code
	rw.WriteHeader(http.StatusNotFound)
	assert.Equal(t, http.StatusNotFound, rw.statusCode)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestNewOptimizedPluginCache(t *testing.T) {
	ttl := 5 * time.Minute
	cache := NewOptimizedPluginCache(ttl)

	assert.NotNil(t, cache)
	assert.NotNil(t, cache.cache)
	assert.Equal(t, ttl, cache.ttl)
}

func TestOptimizedPluginCacheGetSet(t *testing.T) {
	cache := NewOptimizedPluginCache(5 * time.Minute)

	// Test getting non-existent key
	result, found := cache.Get("nonexistent")
	assert.NotNil(t, result) // Returns new context
	assert.False(t, found)

	// Test setting and getting
	originalContext := pipelines.NewPluginContext()
	originalContext.Set("test_key", "test_value")

	cache.Set("test_key", originalContext)

	retrievedContext, found := cache.Get("test_key")
	assert.True(t, found)
	assert.NotNil(t, retrievedContext)

	// Verify the data - Get returns the wrapped value from JSONData.Content
	value, exists := retrievedContext.Get("test_key")
	assert.True(t, exists)
	// The value is wrapped in a map by the auto-wrapping logic
	if mapVal, ok := value.(map[string]any); ok {
		assert.Equal(t, "test_value", mapVal["value"])
	} else {
		assert.Equal(t, "test_value", value)
	}
}

func TestOptimizedPluginCacheExpiration(t *testing.T) {
	// Create cache with very short TTL
	cache := NewOptimizedPluginCache(10 * time.Millisecond)

	// Set a value
	originalContext := pipelines.NewPluginContext()
	originalContext.Set("test_key", "test_value")
	cache.Set("test_key", originalContext)

	// Should be found immediately
	_, found := cache.Get("test_key")
	assert.True(t, found)

	// Wait for expiration
	time.Sleep(20 * time.Millisecond)

	// Should not be found after expiration
	_, found = cache.Get("test_key")
	assert.False(t, found)
}

func TestGetPluginCache(t *testing.T) {
	cache1 := GetPluginCache()
	cache2 := GetPluginCache()

	// Should return the same instance (singleton)
	assert.Same(t, cache1, cache2)
	assert.NotNil(t, cache1)
}

func TestNewWorkerPool(t *testing.T) {
	workers := 3
	pool := NewWorkerPool(workers)

	assert.NotNil(t, pool)
	assert.Equal(t, workers, pool.workers)
	assert.NotNil(t, pool.jobQueue)
	assert.NotNil(t, pool.stopChan)

	// Clean up
	pool.Stop()
}

func TestWorkerPoolSubmit(t *testing.T) {
	pool := NewWorkerPool(2)
	defer pool.Stop()

	// Test submitting jobs
	executed := make(chan bool, 3)

	for i := 0; i < 3; i++ {
		pool.Submit(func() {
			executed <- true
		})
	}

	// Wait for all jobs to execute
	for i := 0; i < 3; i++ {
		select {
		case <-executed:
			// Job executed
		case <-time.After(1 * time.Second):
			t.Fatal("Job did not execute within timeout")
		}
	}
}

func TestWorkerPoolStop(t *testing.T) {
	pool := NewWorkerPool(2)

	// Submit some jobs
	for i := 0; i < 5; i++ {
		pool.Submit(func() {
			time.Sleep(10 * time.Millisecond)
		})
	}

	// Stop the pool
	pool.Stop()

	// Try to submit job after stop (should not block or panic)
	pool.Submit(func() {
		t.Error("Job should not execute after pool stop")
	})

	// Wait a bit to ensure no jobs execute
	time.Sleep(50 * time.Millisecond)
}

func TestNewOptimizedPipelineExecutor(t *testing.T) {
	registry := pipelines.NewPluginRegistry()
	workers := 4

	executor := NewOptimizedPipelineExecutor(registry, workers)

	assert.NotNil(t, executor)
	assert.Equal(t, registry, executor.pluginRegistry)
	assert.NotNil(t, executor.pluginCache)
	assert.NotNil(t, executor.workerPool)
}

func TestExecutePipelineOptimized(t *testing.T) {
	registry := pipelines.NewPluginRegistry()
	executor := NewOptimizedPipelineExecutor(registry, 2)

	// Register a mock plugin
	mockPlugin := &MockPlugin{pluginType: "Input", pluginName: "test"}
	registry.RegisterPlugin(mockPlugin)

	config := &PipelineConfig{
		Name:    "test-pipeline",
		Enabled: true,
		Steps: []pipelines.StepConfig{
			{
				Name:   "step1",
				Plugin: "Input.test",
				Config: map[string]any{},
				Output: "output1",
			},
		},
	}

	result, err := executor.ExecutePipelineOptimized(context.Background(), config)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
}

func TestExecuteStepOptimized(t *testing.T) {
	registry := pipelines.NewPluginRegistry()
	executor := NewOptimizedPipelineExecutor(registry, 2)

	// Register a mock plugin
	mockPlugin := &MockPlugin{pluginType: "Input", pluginName: "test"}
	registry.RegisterPlugin(mockPlugin)

	stepConfig := pipelines.StepConfig{
		Name:   "step1",
		Plugin: "Input.test",
		Config: map[string]any{},
		Output: "output1",
	}

	pluginContext := pipelines.NewPluginContext()

	result, err := executor.executeStepOptimized(context.Background(), stepConfig, pluginContext)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestExecuteStepOptimizedInvalidPlugin(t *testing.T) {
	registry := pipelines.NewPluginRegistry()
	executor := NewOptimizedPipelineExecutor(registry, 2)

	stepConfig := pipelines.StepConfig{
		Name:   "step1",
		Plugin: "invalid", // Invalid format
		Config: map[string]any{},
	}

	pluginContext := pipelines.NewPluginContext()

	_, err := executor.executeStepOptimized(context.Background(), stepConfig, pluginContext)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid plugin reference format")
}

func TestExecuteStepOptimizedNonExistentPlugin(t *testing.T) {
	registry := pipelines.NewPluginRegistry()
	executor := NewOptimizedPipelineExecutor(registry, 2)

	stepConfig := pipelines.StepConfig{
		Name:   "step1",
		Plugin: "Nonexistent.plugin",
		Config: map[string]any{},
	}

	pluginContext := pipelines.NewPluginContext()

	_, err := executor.executeStepOptimized(context.Background(), stepConfig, pluginContext)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "plugin plugin of type Nonexistent not found")
}

func TestCreateCacheKey(t *testing.T) {
	stepConfig := pipelines.StepConfig{
		Name:   "test-step",
		Plugin: "Input.test",
		Config: map[string]any{
			"param1": "value1",
			"param2": 42,
		},
	}

	context := pipelines.NewPluginContext()
	context.Set("context_key", "context_value")

	key := createCacheKey(stepConfig, context)

	assert.NotEmpty(t, key)
	assert.Contains(t, key, "Input.test")
	assert.Contains(t, key, "test-step")
	// Context values are wrapped in JSONData, so they won't be included as simple strings
	assert.Contains(t, key, "param1")
}

// TestNewConnectionPool removed - functionality was unused dead code

// TestConnectionPoolGetPut removed - functionality was unused dead code

// TestConnectionPoolClose removed - functionality was unused dead code

// TestNewStringInterner removed - functionality was unused dead code

// TestStringInternerIntern removed - functionality was unused dead code

// TestGetStringInterner removed - functionality was unused dead code

// TestGetActiveGoroutines removed - functionality was unused dead code
