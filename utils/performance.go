package utils

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
)

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// PerformanceMetrics holds performance statistics
type PerformanceMetrics struct {
	TotalRequests     int64         `json:"total_requests"`
	AverageLatency    time.Duration `json:"average_latency"`
	P95Latency        time.Duration `json:"p95_latency"`
	P99Latency        time.Duration `json:"p99_latency"`
	RequestsPerSecond float64       `json:"requests_per_second"`
	ErrorRate         float64       `json:"error_rate"`
	MemoryUsage       int64         `json:"memory_usage"`
	ActiveGoroutines  int           `json:"active_goroutines"`
}

// PerformanceMonitor tracks performance metrics
type PerformanceMonitor struct {
	requestCount int64
	totalLatency time.Duration
	latencies    []time.Duration
	errorCount   int64
	startTime    time.Time
	mutex        sync.RWMutex
}

// NewPerformanceMonitor creates a new performance monitor
func NewPerformanceMonitor() *PerformanceMonitor {
	return &PerformanceMonitor{
		latencies: make([]time.Duration, 0, 1000),
		startTime: time.Now(),
	}
}

// RecordRequest records a request with its latency and error status
func (pm *PerformanceMonitor) RecordRequest(latency time.Duration, hasError bool) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	pm.requestCount++
	pm.totalLatency += latency
	pm.latencies = append(pm.latencies, latency)

	if hasError {
		pm.errorCount++
	}

	// Keep only last 1000 latencies for percentile calculations
	if len(pm.latencies) > 1000 {
		pm.latencies = pm.latencies[1:]
	}
}

// GetMetrics returns current performance metrics
func (pm *PerformanceMonitor) GetMetrics() *PerformanceMetrics {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	var avgLatency time.Duration
	if pm.requestCount > 0 {
		avgLatency = pm.totalLatency / time.Duration(pm.requestCount)
	}

	var p95Latency, p99Latency time.Duration
	if len(pm.latencies) > 0 {
		sortedLatencies := make([]time.Duration, len(pm.latencies))
		copy(sortedLatencies, pm.latencies)

		// Use efficient sort for percentile calculation
		sort.Slice(sortedLatencies, func(i, j int) bool {
			return sortedLatencies[i] < sortedLatencies[j]
		})

		p95Index := int(float64(len(sortedLatencies)) * 0.95)
		p99Index := int(float64(len(sortedLatencies)) * 0.99)

		if p95Index < len(sortedLatencies) {
			p95Latency = sortedLatencies[p95Index]
		}
		if p99Index < len(sortedLatencies) {
			p99Latency = sortedLatencies[p99Index]
		}
	}

	var errorRate float64
	if pm.requestCount > 0 {
		errorRate = float64(pm.errorCount) / float64(pm.requestCount)
	}

	var rps float64
	elapsed := time.Since(pm.startTime)
	if elapsed.Seconds() > 0 {
		rps = float64(pm.requestCount) / elapsed.Seconds()
	}

	return &PerformanceMetrics{
		TotalRequests:     pm.requestCount,
		AverageLatency:    avgLatency,
		P95Latency:        p95Latency,
		P99Latency:        p99Latency,
		RequestsPerSecond: rps,
		ErrorRate:         errorRate,
		ActiveGoroutines:  getActiveGoroutines(),
	}
}

// Global performance monitor instance
var globalPerformanceMonitor *PerformanceMonitor
var perfOnce sync.Once

// GetPerformanceMonitor returns the global performance monitor instance
func GetPerformanceMonitor() *PerformanceMonitor {
	perfOnce.Do(func() {
		globalPerformanceMonitor = NewPerformanceMonitor()
	})
	return globalPerformanceMonitor
}

// PerformanceMiddleware wraps handlers with performance monitoring
func PerformanceMiddleware(next http.Handler) http.Handler {
	monitor := GetPerformanceMonitor()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a response writer wrapper to capture status code
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Call next handler
		next.ServeHTTP(rw, r)

		// Record metrics
		latency := time.Since(start)
		hasError := rw.statusCode >= 400

		monitor.RecordRequest(latency, hasError)
	})
}

// OptimizedPluginCache provides caching for plugin results
type OptimizedPluginCache struct {
	cache map[string]cacheEntry
	mutex sync.RWMutex
	ttl   time.Duration
}

type cacheEntry struct {
	result   *pipelines.PluginContext
	expireAt time.Time
}

// NewOptimizedPluginCache creates a new plugin cache
func NewOptimizedPluginCache(ttl time.Duration) *OptimizedPluginCache {
	cache := &OptimizedPluginCache{
		cache: make(map[string]cacheEntry),
		ttl:   ttl,
	}

	// Start cleanup goroutine
	go cache.cleanup()

	return cache
}

// Get retrieves a cached result
func (opc *OptimizedPluginCache) Get(key string) (*pipelines.PluginContext, bool) {
	opc.mutex.RLock()
	defer opc.mutex.RUnlock()

	entry, exists := opc.cache[key]
	if !exists {
		return pipelines.NewPluginContext(), false
	}

	if time.Now().After(entry.expireAt) {
		return pipelines.NewPluginContext(), false
	}

	return entry.result, true
}

// Set stores a result in the cache
func (opc *OptimizedPluginCache) Set(key string, result *pipelines.PluginContext) {
	opc.mutex.Lock()
	defer opc.mutex.Unlock()

	opc.cache[key] = cacheEntry{
		result:   result,
		expireAt: time.Now().Add(opc.ttl),
	}
}

// cleanup removes expired entries
func (opc *OptimizedPluginCache) cleanup() {
	ticker := time.NewTicker(opc.ttl)
	defer ticker.Stop()

	for range ticker.C {
		opc.mutex.Lock()
		for key, entry := range opc.cache {
			if time.Now().After(entry.expireAt) {
				delete(opc.cache, key)
			}
		}
		opc.mutex.Unlock()
	}
}

// Global plugin cache instance
var globalPluginCache *OptimizedPluginCache
var cacheOnce sync.Once

// GetPluginCache returns the global plugin cache instance
func GetPluginCache() *OptimizedPluginCache {
	cacheOnce.Do(func() {
		globalPluginCache = NewOptimizedPluginCache(5 * time.Minute) // 5 minute TTL
	})
	return globalPluginCache
}

// OptimizedPipelineExecutor provides optimized pipeline execution
type OptimizedPipelineExecutor struct {
	pluginRegistry *pipelines.PluginRegistry
	pluginCache    *OptimizedPluginCache
	workerPool     *WorkerPool
}

// WorkerPool manages a pool of worker goroutines
type WorkerPool struct {
	workers   int
	jobQueue  chan func()
	stopChan  chan struct{}
	waitGroup sync.WaitGroup
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(workers int) *WorkerPool {
	wp := &WorkerPool{
		workers:  workers,
		jobQueue: make(chan func(), 100),
		stopChan: make(chan struct{}),
	}

	// Start workers
	for i := 0; i < workers; i++ {
		wp.waitGroup.Add(1)
		go wp.worker()
	}

	return wp
}

// Submit submits a job to the worker pool
func (wp *WorkerPool) Submit(job func()) {
	select {
	case wp.jobQueue <- job:
	case <-wp.stopChan:
		return
	}
}

// Stop stops the worker pool
func (wp *WorkerPool) Stop() {
	close(wp.stopChan)
	wp.waitGroup.Wait()
}

// worker runs jobs from the queue
func (wp *WorkerPool) worker() {
	defer wp.waitGroup.Done()

	for {
		select {
		case job := <-wp.jobQueue:
			job()
		case <-wp.stopChan:
			return
		}
	}
}

// NewOptimizedPipelineExecutor creates a new optimized pipeline executor
func NewOptimizedPipelineExecutor(registry *pipelines.PluginRegistry, workers int) *OptimizedPipelineExecutor {
	return &OptimizedPipelineExecutor{
		pluginRegistry: registry,
		pluginCache:    GetPluginCache(),
		workerPool:     NewWorkerPool(workers),
	}
}

// ExecutePipelineOptimized executes a pipeline with optimizations
func (ope *OptimizedPipelineExecutor) ExecutePipelineOptimized(ctx context.Context, config *PipelineConfig) (*PipelineExecutionResult, error) {
	result := &PipelineExecutionResult{
		Success: true,
		Context: pipelines.NewPluginContext(),
	}

	// Execute each step
	for i, step := range config.Steps {
		stepResult, err := ope.executeStepOptimized(ctx, step, result.Context)
		if err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("step %d (%s) failed: %v", i+1, step.Name, err)
			return result, nil
		}

		// Merge step results into global context
		for _, key := range stepResult.Keys() {
			if value, exists := stepResult.Get(key); exists {
				result.Context.Set(key, value)
			}
		}
	}

	return result, nil
}

// executeStepOptimized executes a single pipeline step with optimizations
func (ope *OptimizedPipelineExecutor) executeStepOptimized(ctx context.Context, step pipelines.StepConfig, context *pipelines.PluginContext) (*pipelines.PluginContext, error) {
	// Create cache key from step configuration
	cacheKey := createCacheKey(step, context)

	// Check cache first
	if cachedResult, found := ope.pluginCache.Get(cacheKey); found {
		return cachedResult, nil
	}

	// Parse plugin reference
	pluginParts := strings.Split(step.Plugin, ".")
	if len(pluginParts) != 2 {
		return nil, fmt.Errorf("invalid plugin reference format: %s, expected 'Type.Name'", step.Plugin)
	}

	pluginType := pluginParts[0]
	pluginName := pluginParts[1]

	// Get the plugin
	plugin, err := ope.pluginRegistry.GetPlugin(pluginType, pluginName)
	if err != nil {
		return nil, fmt.Errorf("failed to get plugin %s: %w", step.Plugin, err)
	}

	// Execute the step
	stepResult, err := plugin.ExecuteStep(ctx, step, context)
	if err != nil {
		return nil, fmt.Errorf("plugin execution failed: %w", err)
	}

	// Cache the result
	ope.pluginCache.Set(cacheKey, stepResult)

	return stepResult, nil
}

// createCacheKey creates a cache key from step configuration and context
func createCacheKey(step pipelines.StepConfig, context *pipelines.PluginContext) string {
	// Create a deterministic cache key
	key := fmt.Sprintf("%s:%s:%v", step.Plugin, step.Name, step.Config)

	// Include relevant context values
	for _, k := range context.Keys() {
		if value, exists := context.Get(k); exists {
			if str, ok := value.(string); ok && len(str) < 100 { // Only include short string values
				key += fmt.Sprintf(":%s=%s", k, str)
			}
		}
	}

	return key
}

// getActiveGoroutines returns the number of active goroutines
func getActiveGoroutines() int {
	// This is a simplified implementation
	// In production, you might want to use runtime.NumGoroutine()
	return 0 // Placeholder
}

// Performance optimization utilities

// ConnectionPool manages a pool of reusable connections
type ConnectionPool struct {
	connections chan interface{}
	factory     func() interface{}
	closeFunc   func(interface{})
	maxSize     int
	mutex       sync.Mutex
}

// NewConnectionPool creates a new connection pool
func NewConnectionPool(maxSize int, factory func() interface{}, closeFunc func(interface{})) *ConnectionPool {
	cp := &ConnectionPool{
		connections: make(chan interface{}, maxSize),
		factory:     factory,
		closeFunc:   closeFunc,
		maxSize:     maxSize,
	}

	// Pre-populate pool
	for i := 0; i < maxSize; i++ {
		conn := factory()
		cp.connections <- conn
	}

	return cp
}

// Get retrieves a connection from the pool
func (cp *ConnectionPool) Get() interface{} {
	select {
	case conn := <-cp.connections:
		return conn
	default:
		// Pool is empty, create new connection
		return cp.factory()
	}
}

// Put returns a connection to the pool
func (cp *ConnectionPool) Put(conn interface{}) {
	select {
	case cp.connections <- conn:
	default:
		// Pool is full, close connection
		cp.closeFunc(conn)
	}
}

// Close closes all connections in the pool
func (cp *ConnectionPool) Close() {
	close(cp.connections)
	for conn := range cp.connections {
		cp.closeFunc(conn)
	}
}

// Memory-efficient data structures

// StringInterner provides string interning for memory optimization
type StringInterner struct {
	strings map[string]string
	mutex   sync.RWMutex
}

// NewStringInterner creates a new string interner
func NewStringInterner() *StringInterner {
	return &StringInterner{
		strings: make(map[string]string),
	}
}

// Intern interns a string and returns the interned version
func (si *StringInterner) Intern(s string) string {
	si.mutex.RLock()
	if interned, exists := si.strings[s]; exists {
		si.mutex.RUnlock()
		return interned
	}
	si.mutex.RUnlock()

	si.mutex.Lock()
	defer si.mutex.Unlock()

	// Double-check after acquiring write lock
	if interned, exists := si.strings[s]; exists {
		return interned
	}

	// Intern the string
	si.strings[s] = s
	return s
}

// Size returns the number of interned strings
func (si *StringInterner) Size() int {
	si.mutex.RLock()
	defer si.mutex.RUnlock()
	return len(si.strings)
}

// Global string interner instance
var globalStringInterner *StringInterner
var internerOnce sync.Once

// GetStringInterner returns the global string interner instance
func GetStringInterner() *StringInterner {
	internerOnce.Do(func() {
		globalStringInterner = NewStringInterner()
	})
	return globalStringInterner
}
