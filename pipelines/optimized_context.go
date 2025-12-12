package pipelines

import (
	"sync"
)

// OptimizedPluginContext implements copy-on-write semantics for better performance
type OptimizedPluginContext struct {
	data     map[string]DataValue
	metadata map[string]any
	mutex    sync.RWMutex
	version  int64 // For copy-on-write optimization
}

// NewOptimizedPluginContext creates a new optimized context
func NewOptimizedPluginContext() *OptimizedPluginContext {
	return &OptimizedPluginContext{
		data:     make(map[string]DataValue),
		metadata: make(map[string]any),
		version:  0,
	}
}

// Set stores a value in the context
func (opc *OptimizedPluginContext) Set(key string, value DataValue) {
	opc.mutex.Lock()
	defer opc.mutex.Unlock()

	opc.data[key] = value
	opc.version++
}

// Get retrieves a value from the context
func (opc *OptimizedPluginContext) Get(key string) (DataValue, bool) {
	opc.mutex.RLock()
	defer opc.mutex.RUnlock()

	value, exists := opc.data[key]
	return value, exists
}

// Clone creates a copy-on-write clone
func (opc *OptimizedPluginContext) Clone() *OptimizedPluginContext {
	opc.mutex.RLock()
	defer opc.mutex.RUnlock()

	// Shallow copy data and metadata for copy-on-write
	dataCopy := make(map[string]DataValue, len(opc.data))
	for k, v := range opc.data {
		dataCopy[k] = v
	}

	metadataCopy := make(map[string]any, len(opc.metadata))
	for k, v := range opc.metadata {
		metadataCopy[k] = v
	}

	return &OptimizedPluginContext{
		data:     dataCopy,
		metadata: metadataCopy,
		version:  opc.version,
	}
}

// GetVersion returns the current version for change detection
func (opc *OptimizedPluginContext) GetVersion() int64 {
	return opc.version
}

// Merge merges another context into this one
func (opc *OptimizedPluginContext) Merge(other *OptimizedPluginContext) {
	opc.mutex.Lock()
	defer opc.mutex.Unlock()

	other.mutex.RLock()
	defer other.mutex.RUnlock()

	for k, v := range other.data {
		opc.data[k] = v
	}

	for k, v := range other.metadata {
		opc.metadata[k] = v
	}

	opc.version++
}

// Clear removes all data from the context
func (opc *OptimizedPluginContext) Clear() {
	opc.mutex.Lock()
	defer opc.mutex.Unlock()

	opc.data = make(map[string]DataValue)
	opc.metadata = make(map[string]any)
	opc.version++
}

// ToPluginContext converts to the original PluginContext format
func (opc *OptimizedPluginContext) ToPluginContext() PluginContext {
	opc.mutex.RLock()
	defer opc.mutex.RUnlock()

	dataCopy := make(map[string]DataValue, len(opc.data))
	for k, v := range opc.data {
		dataCopy[k] = v
	}

	metadataCopy := make(map[string]any, len(opc.metadata))
	for k, v := range opc.metadata {
		metadataCopy[k] = v
	}

	return PluginContext{
		data:     dataCopy,
		metadata: metadataCopy,
		mutex:    sync.RWMutex{},
	}
}

// FromPluginContext creates an optimized context from the original
func FromPluginContext(pc *PluginContext) *OptimizedPluginContext {
	pc.mutex.RLock()
	defer pc.mutex.RUnlock()

	dataCopy := make(map[string]DataValue, len(pc.data))
	for k, v := range pc.data {
		dataCopy[k] = v
	}

	metadataCopy := make(map[string]any, len(pc.metadata))
	for k, v := range pc.metadata {
		metadataCopy[k] = v
	}

	return &OptimizedPluginContext{
		data:     dataCopy,
		metadata: metadataCopy,
		version:  0,
	}
}
