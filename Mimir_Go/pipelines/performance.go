package pipelines

import (
	"sync"
	"time"
)

// CacheEntry represents a cached data value with metadata
type CacheEntry struct {
	Data        DataValue
	Timestamp   time.Time
	AccessCount int
	LastAccess  time.Time
}

// DataCache provides caching for data values
type DataCache struct {
	cache   map[string]*CacheEntry
	mutex   sync.RWMutex
	maxSize int
	ttl     time.Duration
}

// NewDataCache creates a new data cache
func NewDataCache(maxSize int, ttl time.Duration) *DataCache {
	return &DataCache{
		cache:   make(map[string]*CacheEntry),
		maxSize: maxSize,
		ttl:     ttl,
	}
}

// Get retrieves a value from cache
func (dc *DataCache) Get(key string) (DataValue, bool) {
	dc.mutex.RLock()
	defer dc.mutex.RUnlock()

	entry, exists := dc.cache[key]
	if !exists {
		return nil, false
	}

	// Check TTL
	if time.Since(entry.Timestamp) > dc.ttl {
		return nil, false
	}

	// Update access statistics
	entry.AccessCount++
	entry.LastAccess = time.Now()

	return entry.Data, true
}

// Put stores a value in cache
func (dc *DataCache) Put(key string, value DataValue) {
	dc.mutex.Lock()
	defer dc.mutex.Unlock()

	// Check if we need to evict
	if len(dc.cache) >= dc.maxSize {
		dc.evictLRU()
	}

	dc.cache[key] = &CacheEntry{
		Data:        value,
		Timestamp:   time.Now(),
		AccessCount: 1,
		LastAccess:  time.Now(),
	}
}

// evictLRU removes the least recently used entry
func (dc *DataCache) evictLRU() {
	var oldestKey string
	var oldestTime time.Time

	for key, entry := range dc.cache {
		if oldestKey == "" || entry.LastAccess.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.LastAccess
		}
	}

	if oldestKey != "" {
		delete(dc.cache, oldestKey)
	}
}

// Clear removes all entries from cache
func (dc *DataCache) Clear() {
	dc.mutex.Lock()
	defer dc.mutex.Unlock()
	dc.cache = make(map[string]*CacheEntry)
}

// Size returns the current cache size
func (dc *DataCache) Size() int {
	dc.mutex.RLock()
	defer dc.mutex.RUnlock()
	return len(dc.cache)
}

// ObjectPool provides object pooling for data types
type ObjectPool struct {
	pool sync.Pool
}

// NewJSONDataPool creates a pool for JSONData objects
func NewJSONDataPool() *ObjectPool {
	return &ObjectPool{
		pool: sync.Pool{
			New: func() interface{} {
				return &JSONData{}
			},
		},
	}
}

// NewBinaryDataPool creates a pool for BinaryData objects
func NewBinaryDataPool() *ObjectPool {
	return &ObjectPool{
		pool: sync.Pool{
			New: func() interface{} {
				return &BinaryData{}
			},
		},
	}
}

// NewTimeSeriesDataPool creates a pool for TimeSeriesData objects
func NewTimeSeriesDataPool() *ObjectPool {
	return &ObjectPool{
		pool: sync.Pool{
			New: func() interface{} {
				return NewTimeSeriesData()
			},
		},
	}
}

// Get retrieves an object from the pool
func (op *ObjectPool) Get() interface{} {
	return op.pool.Get()
}

// Put returns an object to the pool
func (op *ObjectPool) Put(obj interface{}) {
	// Reset object state before returning to pool
	switch v := obj.(type) {
	case *JSONData:
		v.Content = nil
	case *BinaryData:
		v.Content = nil
		v.MIMEType = ""
	case *TimeSeriesData:
		v.Points = v.Points[:0] // Keep slice but clear contents
		v.Metadata = nil
	}
	op.pool.Put(obj)
}

// LazyDataValue provides lazy loading for large data
type LazyDataValue struct {
	key        string
	dataLoader func() (DataValue, error)
	loadedData DataValue
	isLoaded   bool
	mutex      sync.Mutex
}

// NewLazyDataValue creates a new lazy data value
func NewLazyDataValue(key string, loader func() (DataValue, error)) *LazyDataValue {
	return &LazyDataValue{
		key:        key,
		dataLoader: loader,
		isLoaded:   false,
	}
}

// Load loads the data if not already loaded
func (ldv *LazyDataValue) Load() error {
	ldv.mutex.Lock()
	defer ldv.mutex.Unlock()

	if !ldv.isLoaded {
		data, err := ldv.dataLoader()
		if err != nil {
			return err
		}
		ldv.loadedData = data
		ldv.isLoaded = true
	}
	return nil
}

// GetData returns the loaded data, loading it if necessary
func (ldv *LazyDataValue) GetData() (DataValue, error) {
	if !ldv.isLoaded {
		if err := ldv.Load(); err != nil {
			return nil, err
		}
	}
	return ldv.loadedData, nil
}

// IsLoaded returns whether the data has been loaded
func (ldv *LazyDataValue) IsLoaded() bool {
	ldv.mutex.Lock()
	defer ldv.mutex.Unlock()
	return ldv.isLoaded
}

// MemoryManager provides memory usage tracking and management
type MemoryManager struct {
	allocatedData map[string]int
	totalMemory   int64
	maxMemory     int64
	mutex         sync.RWMutex
}

// NewMemoryManager creates a new memory manager
func NewMemoryManager(maxMemory int64) *MemoryManager {
	return &MemoryManager{
		allocatedData: make(map[string]int),
		maxMemory:     maxMemory,
	}
}

// Allocate tracks memory allocation for a data key
func (mm *MemoryManager) Allocate(key string, size int) bool {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	newTotal := mm.totalMemory + int64(size)
	if newTotal > mm.maxMemory {
		return false // Would exceed memory limit
	}

	mm.allocatedData[key] = size
	mm.totalMemory = newTotal
	return true
}

// Deallocate frees memory for a data key
func (mm *MemoryManager) Deallocate(key string) {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	if size, exists := mm.allocatedData[key]; exists {
		mm.totalMemory -= int64(size)
		delete(mm.allocatedData, key)
	}
}

// GetTotalMemory returns current total memory usage
func (mm *MemoryManager) GetTotalMemory() int64 {
	mm.mutex.RLock()
	defer mm.mutex.RUnlock()
	return mm.totalMemory
}

// GetMaxMemory returns maximum allowed memory
func (mm *MemoryManager) GetMaxMemory() int64 {
	return mm.maxMemory
}

// GetMemoryUsage returns memory usage as a percentage
func (mm *MemoryManager) GetMemoryUsage() float64 {
	mm.mutex.RLock()
	defer mm.mutex.RUnlock()
	if mm.maxMemory == 0 {
		return 0
	}
	return float64(mm.totalMemory) / float64(mm.maxMemory) * 100
}

// StreamingDataProcessor provides streaming processing for large datasets
type StreamingDataProcessor struct {
	bufferSize int
	processor  func([]byte) error
}

// NewStreamingDataProcessor creates a new streaming processor
func NewStreamingDataProcessor(bufferSize int, processor func([]byte) error) *StreamingDataProcessor {
	return &StreamingDataProcessor{
		bufferSize: bufferSize,
		processor:  processor,
	}
}

// ProcessStream processes data in chunks
func (sdp *StreamingDataProcessor) ProcessStream(data []byte) error {
	chunkSize := sdp.bufferSize
	for i := 0; i < len(data); i += chunkSize {
		end := i + chunkSize
		if end > len(data) {
			end = len(data)
		}

		chunk := data[i:end]
		if err := sdp.processor(chunk); err != nil {
			return err
		}
	}
	return nil
}

// PerformanceMetrics tracks performance statistics
type PerformanceMetrics struct {
	OperationCount int64
	TotalTime      time.Duration
	AverageTime    time.Duration
	MinTime        time.Duration
	MaxTime        time.Duration
	mutex          sync.Mutex
}

// NewPerformanceMetrics creates new performance metrics
func NewPerformanceMetrics() *PerformanceMetrics {
	return &PerformanceMetrics{
		MinTime: time.Hour, // Initialize to large value
	}
}

// RecordOperation records the duration of an operation
func (pm *PerformanceMetrics) RecordOperation(duration time.Duration) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	pm.OperationCount++
	pm.TotalTime += duration

	if duration < pm.MinTime {
		pm.MinTime = duration
	}
	if duration > pm.MaxTime {
		pm.MaxTime = duration
	}

	if pm.OperationCount > 0 {
		pm.AverageTime = pm.TotalTime / time.Duration(pm.OperationCount)
	}
}

// GetStats returns current performance statistics
func (pm *PerformanceMetrics) GetStats() (int64, time.Duration, time.Duration, time.Duration) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()
	return pm.OperationCount, pm.AverageTime, pm.MinTime, pm.MaxTime
}

// Global instances for performance optimization
var (
	// Default cache with 1000 entries, 30-minute TTL
	DefaultDataCache = NewDataCache(1000, 30*time.Minute)

	// Object pools for different data types
	JSONDataPool       = NewJSONDataPool()
	BinaryDataPool     = NewBinaryDataPool()
	TimeSeriesDataPool = NewTimeSeriesDataPool()

	// Memory manager with 1GB limit
	DefaultMemoryManager = NewMemoryManager(1024 * 1024 * 1024) // 1GB

	// Performance metrics for different operations
	SerializationMetrics   = NewPerformanceMetrics()
	DeserializationMetrics = NewPerformanceMetrics()
	CacheHitMetrics        = NewPerformanceMetrics()
	CacheMissMetrics       = NewPerformanceMetrics()
)
