package tests

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	storage "github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/Storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAPIPerformanceConcurrentRequests tests API performance under concurrent load
func TestAPIPerformanceConcurrentRequests(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	tmpDB := fmt.Sprintf("/tmp/api_performance_test_%d.db", os.Getpid())
	defer os.Remove(tmpDB)

	persistence, err := storage.NewPersistenceBackend(tmpDB)
	require.NoError(t, err)
	defer persistence.Close()

	ms := NewMockServer()
	testServer := ms.Start()
	defer testServer.Close()

	t.Run("Concurrent Health Check Requests", func(t *testing.T) {
		concurrency := 100
		requestsPerWorker := 10

		var wg sync.WaitGroup
		var successCount int64
		var errorCount int64

		startTime := time.Now()

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < requestsPerWorker; j++ {
					resp, err := http.Get(testServer.URL + "/health")
					if err != nil {
						atomic.AddInt64(&errorCount, 1)
						continue
					}
					resp.Body.Close()

					if resp.StatusCode == http.StatusOK {
						atomic.AddInt64(&successCount, 1)
					} else {
						atomic.AddInt64(&errorCount, 1)
					}
				}
			}()
		}

		wg.Wait()
		duration := time.Since(startTime)

		totalRequests := int64(concurrency * requestsPerWorker)
		assert.Equal(t, totalRequests, successCount, "All requests should succeed")
		assert.Equal(t, int64(0), errorCount, "No errors should occur")

		requestsPerSecond := float64(totalRequests) / duration.Seconds()
		averageLatency := duration / time.Duration(totalRequests)

		t.Logf("Performance Metrics:")
		t.Logf("  Total Requests: %d", totalRequests)
		t.Logf("  Concurrent Workers: %d", concurrency)
		t.Logf("  Duration: %v", duration)
		t.Logf("  Requests/Second: %.2f", requestsPerSecond)
		t.Logf("  Average Latency: %v", averageLatency)

		// Performance assertions
		assert.Greater(t, requestsPerSecond, 100.0, "Should handle at least 100 req/s")
		assert.Less(t, averageLatency, 50*time.Millisecond, "Average latency should be under 50ms")
	})
}

// TestAPIPerformanceResponseTime tests API response time under various conditions
func TestAPIPerformanceResponseTime(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	ms := NewMockServer()
	testServer := ms.Start()
	defer testServer.Close()

	endpoints := []struct {
		name     string
		method   string
		path     string
		maxLatency time.Duration
	}{
		{"Health Check", "GET", "/health", 10 * time.Millisecond},
		{"List Plugins", "GET", "/api/v1/plugins", 20 * time.Millisecond},
		{"List Pipelines", "GET", "/api/v1/pipelines", 20 * time.Millisecond},
		{"Performance Metrics", "GET", "/api/v1/performance/metrics", 20 * time.Millisecond},
	}

	for _, endpoint := range endpoints {
		t.Run(endpoint.name, func(t *testing.T) {
			iterations := 100
			var totalDuration time.Duration

			for i := 0; i < iterations; i++ {
				start := time.Now()

				var resp *http.Response
				var err error

				switch endpoint.method {
				case "GET":
					resp, err = http.Get(testServer.URL + endpoint.path)
				case "POST":
					resp, err = http.Post(testServer.URL+endpoint.path, "application/json", bytes.NewBuffer([]byte("{}")))
				}

				require.NoError(t, err)
				resp.Body.Close()

				duration := time.Since(start)
				totalDuration += duration
			}

			averageLatency := totalDuration / time.Duration(iterations)

			t.Logf("%s Average Latency: %v", endpoint.name, averageLatency)
			assert.Less(t, averageLatency, endpoint.maxLatency,
				fmt.Sprintf("%s should respond in under %v", endpoint.name, endpoint.maxLatency))
		})
	}
}

// TestEncryptionPerformance tests encryption/decryption performance
func TestEncryptionPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	// This test would require importing the encryption package
	// and testing encryption performance directly
	t.Skip("Encryption performance tested in utils/encryption_test.go benchmarks")
}

// TestMemoryUsage tests memory consumption under load
func TestMemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory test in short mode")
	}

	ms := NewMockServer()
	testServer := ms.Start()
	defer testServer.Close()

	// Warm up
	for i := 0; i < 10; i++ {
		resp, _ := http.Get(testServer.URL + "/health")
		resp.Body.Close()
	}

	// Measure memory before load
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)

	// Generate load
	concurrency := 50
	requestsPerWorker := 100
	var wg sync.WaitGroup

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < requestsPerWorker; j++ {
				resp, err := http.Get(testServer.URL + "/health")
				if err == nil {
					resp.Body.Close()
				}
			}
		}()
	}

	wg.Wait()

	// Force garbage collection
	runtime.GC()
	time.Sleep(100 * time.Millisecond)

	// Measure memory after load
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)

	allocatedMemory := m2.Alloc - m1.Alloc
	totalMemory := m2.TotalAlloc - m1.TotalAlloc

	t.Logf("Memory Usage:")
	t.Logf("  Allocated: %d KB", allocatedMemory/1024)
	t.Logf("  Total Allocated: %d KB", totalMemory/1024)
	t.Logf("  Number of GC runs: %d", m2.NumGC-m1.NumGC)

	// Memory should not grow excessively (less than 100MB for this test)
	assert.Less(t, allocatedMemory, uint64(100*1024*1024), "Memory usage should be under 100MB")
}

// TestDatabasePerformance tests database operation performance
func TestDatabasePerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	tmpDB := fmt.Sprintf("/tmp/db_performance_test_%d.db", os.Getpid())
	defer os.Remove(tmpDB)

	persistence, err := storage.NewPersistenceBackend(tmpDB)
	require.NoError(t, err)
	defer persistence.Close()

	db := persistence.GetDB()

	t.Run("Bulk Insert Performance", func(t *testing.T) {
		iterations := 1000
		startTime := time.Now()

		for i := 0; i < iterations; i++ {
			query := `
				INSERT INTO api_keys (id, provider, name, key_value, is_active, created_at, updated_at)
				VALUES (?, ?, ?, ?, ?, ?, ?)
			`
			_, err := db.Exec(query,
				fmt.Sprintf("key_%d", i),
				"test_provider",
				fmt.Sprintf("Test Key %d", i),
				"encrypted_value",
				true,
				time.Now(),
				time.Now(),
			)
			require.NoError(t, err)
		}

		duration := time.Since(startTime)
		insertsPerSecond := float64(iterations) / duration.Seconds()
		averageLatency := duration / time.Duration(iterations)

		t.Logf("Database Insert Performance:")
		t.Logf("  Total Inserts: %d", iterations)
		t.Logf("  Duration: %v", duration)
		t.Logf("  Inserts/Second: %.2f", insertsPerSecond)
		t.Logf("  Average Latency: %v", averageLatency)

		assert.Greater(t, insertsPerSecond, 100.0, "Should handle at least 100 inserts/s")
	})

	t.Run("Bulk Query Performance", func(t *testing.T) {
		iterations := 1000
		startTime := time.Now()

		for i := 0; i < iterations; i++ {
			rows, err := db.Query("SELECT id, provider, name FROM api_keys LIMIT 10")
			require.NoError(t, err)
			rows.Close()
		}

		duration := time.Since(startTime)
		queriesPerSecond := float64(iterations) / duration.Seconds()
		averageLatency := duration / time.Duration(iterations)

		t.Logf("Database Query Performance:")
		t.Logf("  Total Queries: %d", iterations)
		t.Logf("  Duration: %v", duration)
		t.Logf("  Queries/Second: %.2f", queriesPerSecond)
		t.Logf("  Average Latency: %v", averageLatency)

		assert.Greater(t, queriesPerSecond, 500.0, "Should handle at least 500 queries/s")
	})
}

// TestThroughputUnderLoad tests system throughput under sustained load
func TestThroughputUnderLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping throughput test in short mode")
	}

	ms := NewMockServer()
	testServer := ms.Start()
	defer testServer.Close()

	testDuration := 10 * time.Second
	concurrency := 20

	var requestCount int64
	var errorCount int64
	var wg sync.WaitGroup

	startTime := time.Now()
	stopTime := startTime.Add(testDuration)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for time.Now().Before(stopTime) {
				resp, err := http.Get(testServer.URL + "/health")
				if err != nil {
					atomic.AddInt64(&errorCount, 1)
					continue
				}
				resp.Body.Close()

				if resp.StatusCode == http.StatusOK {
					atomic.AddInt64(&requestCount, 1)
				} else {
					atomic.AddInt64(&errorCount, 1)
				}
			}
		}()
	}

	wg.Wait()
	actualDuration := time.Since(startTime)

	throughput := float64(requestCount) / actualDuration.Seconds()
	errorRate := float64(errorCount) / float64(requestCount+errorCount) * 100

	t.Logf("Throughput Test Results:")
	t.Logf("  Duration: %v", actualDuration)
	t.Logf("  Total Requests: %d", requestCount)
	t.Logf("  Errors: %d", errorCount)
	t.Logf("  Throughput: %.2f req/s", throughput)
	t.Logf("  Error Rate: %.2f%%", errorRate)

	assert.Greater(t, throughput, 500.0, "Should maintain at least 500 req/s throughput")
	assert.Less(t, errorRate, 1.0, "Error rate should be less than 1%")
}

// TestScalability tests how the system scales with increasing load
func TestScalability(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping scalability test in short mode")
	}

	ms := NewMockServer()
	testServer := ms.Start()
	defer testServer.Close()

	concurrencyLevels := []int{1, 10, 50, 100, 200}
	requestsPerWorker := 100

	results := make(map[int]float64)

	for _, concurrency := range concurrencyLevels {
		var wg sync.WaitGroup
		var successCount int64

		startTime := time.Now()

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < requestsPerWorker; j++ {
					resp, err := http.Get(testServer.URL + "/health")
					if err == nil && resp.StatusCode == http.StatusOK {
						atomic.AddInt64(&successCount, 1)
						resp.Body.Close()
					}
				}
			}()
		}

		wg.Wait()
		duration := time.Since(startTime)

		throughput := float64(successCount) / duration.Seconds()
		results[concurrency] = throughput

		t.Logf("Concurrency %d: %.2f req/s", concurrency, throughput)
	}

	// Verify that throughput scales reasonably (should not degrade significantly)
	for i := 1; i < len(concurrencyLevels); i++ {
		prevLevel := concurrencyLevels[i-1]
		currentLevel := concurrencyLevels[i]

		prevThroughput := results[prevLevel]
		currentThroughput := results[currentLevel]

		// Throughput should not drop by more than 50% as concurrency increases
		degradation := (prevThroughput - currentThroughput) / prevThroughput * 100

		t.Logf("Degradation from %d to %d workers: %.2f%%", prevLevel, currentLevel, degradation)
		assert.Less(t, degradation, 50.0,
			fmt.Sprintf("Performance should not degrade more than 50%% from %d to %d workers",
				prevLevel, currentLevel))
	}
}
