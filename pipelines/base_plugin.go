package pipelines

import (
	"context"
	"fmt"
	"sync"
)

// PluginContext represents the execution context passed between pipeline steps
type PluginContext struct {
	data     map[string]DataValue
	metadata map[string]any
	mutex    sync.RWMutex
}

// NewPluginContext creates a new PluginContext instance
func NewPluginContext() *PluginContext {
	return &PluginContext{
		data:     make(map[string]DataValue),
		metadata: make(map[string]any),
	}
}

// Get retrieves a value by key
func (pc *PluginContext) Get(key string) (any, bool) {
	pc.mutex.RLock()
	defer pc.mutex.RUnlock()

	if data, exists := pc.data[key]; exists {
		switch v := data.(type) {
		case *JSONData:
			return v.Content, true
		case *BinaryData:
			return v.Content, true
		case *TimeSeriesData:
			return v.Points, true
		case *ImageData:
			return v.Content, true
		default:
			return data, true
		}
	}
	return nil, false
}

// GetTyped retrieves a typed DataValue by key
func (pc *PluginContext) GetTyped(key string) (DataValue, bool) {
	pc.mutex.RLock()
	defer pc.mutex.RUnlock()
	data, exists := pc.data[key]
	return data, exists
}

// Set stores a value by key (auto-wraps in appropriate DataValue type)
func (pc *PluginContext) Set(key string, value any) {
	pc.mutex.Lock()
	defer pc.mutex.Unlock()

	var dataValue DataValue
	switch v := value.(type) {
	case map[string]any:
		dataValue = NewJSONData(v)
	case []byte:
		dataValue = NewBinaryData(v, "application/octet-stream")
	case []TimePoint:
		tsData := NewTimeSeriesData()
		tsData.Points = v
		dataValue = tsData
	default:
		// For other types, wrap in JSONData
		dataValue = NewJSONData(map[string]any{"value": v})
	}

	pc.data[key] = dataValue
}

// SetTyped stores a typed DataValue by key
func (pc *PluginContext) SetTyped(key string, value DataValue) {
	pc.mutex.Lock()
	defer pc.mutex.Unlock()
	pc.data[key] = value
}

// Delete removes a value by key
func (pc *PluginContext) Delete(key string) {
	pc.mutex.Lock()
	defer pc.mutex.Unlock()
	delete(pc.data, key)
}

// Keys returns all keys in the context
func (pc *PluginContext) Keys() []string {
	pc.mutex.RLock()
	defer pc.mutex.RUnlock()
	keys := make([]string, 0, len(pc.data))
	for k := range pc.data {
		keys = append(keys, k)
	}
	return keys
}

// Size returns the number of items in the context
func (pc *PluginContext) Size() int {
	pc.mutex.RLock()
	defer pc.mutex.RUnlock()
	return len(pc.data)
}

// Clear removes all data from the context
func (pc *PluginContext) Clear() {
	pc.mutex.Lock()
	defer pc.mutex.Unlock()
	pc.data = make(map[string]DataValue)
	pc.metadata = make(map[string]any)
}

// Clone creates a deep copy of the context
func (pc *PluginContext) Clone() *PluginContext {
	pc.mutex.RLock()
	defer pc.mutex.RUnlock()

	newCtx := NewPluginContext()
	for k, v := range pc.data {
		newCtx.data[k] = v.Clone()
	}
	for k, v := range pc.metadata {
		newCtx.metadata[k] = v
	}
	return newCtx
}

// GetMetadata retrieves metadata by key
func (pc *PluginContext) GetMetadata(key string) (any, bool) {
	pc.mutex.RLock()
	defer pc.mutex.RUnlock()
	value, exists := pc.metadata[key]
	return value, exists
}

// SetMetadata stores metadata by key
func (pc *PluginContext) SetMetadata(key string, value any) {
	pc.mutex.Lock()
	defer pc.mutex.Unlock()
	pc.metadata[key] = value
}

// StepConfig represents the configuration for a single pipeline step
type StepConfig struct {
	Name   string         `yaml:"name"`
	Plugin string         `yaml:"plugin"`
	Config map[string]any `yaml:"config"`
	Output string         `yaml:"output,omitempty"`
}

// BasePlugin defines the interface that all plugins must implement
type BasePlugin interface {
	// ExecuteStep executes a single pipeline step
	ExecuteStep(ctx context.Context, stepConfig StepConfig, globalContext *PluginContext) (*PluginContext, error)

	// GetPluginType returns the type of plugin (Input, Data_Processing, AIModels, Output)
	GetPluginType() string

	// GetPluginName returns the name of the plugin
	GetPluginName() string

	// ValidateConfig validates the plugin configuration
	ValidateConfig(config map[string]any) error
}

// PluginRegistry holds all registered plugins
type PluginRegistry struct {
	plugins map[string]map[string]BasePlugin // pluginType -> pluginName -> plugin
}

// NewPluginRegistry creates a new plugin registry
func NewPluginRegistry() *PluginRegistry {
	return &PluginRegistry{
		plugins: make(map[string]map[string]BasePlugin),
	}
}

// RegisterPlugin registers a plugin in the registry
func (pr *PluginRegistry) RegisterPlugin(plugin BasePlugin) error {
	pluginType := plugin.GetPluginType()
	pluginName := plugin.GetPluginName()

	if pr.plugins[pluginType] == nil {
		pr.plugins[pluginType] = make(map[string]BasePlugin)
	}

	if _, exists := pr.plugins[pluginType][pluginName]; exists {
		return fmt.Errorf("plugin %s of type %s already registered", pluginName, pluginType)
	}

	pr.plugins[pluginType][pluginName] = plugin
	return nil
}

// GetPlugin retrieves a plugin by type and name
func (pr *PluginRegistry) GetPlugin(pluginType, pluginName string) (BasePlugin, error) {
	if typePlugins, exists := pr.plugins[pluginType]; exists {
		if plugin, exists := typePlugins[pluginName]; exists {
			return plugin, nil
		}
	}
	return nil, fmt.Errorf("plugin %s of type %s not found", pluginName, pluginType)
}

// GetPluginsByType returns all plugins of a specific type
func (pr *PluginRegistry) GetPluginsByType(pluginType string) map[string]BasePlugin {
	if plugins, exists := pr.plugins[pluginType]; exists {
		return plugins
	}
	return make(map[string]BasePlugin)
}

// GetAllPlugins returns all registered plugins
func (pr *PluginRegistry) GetAllPlugins() map[string]map[string]BasePlugin {
	return pr.plugins
}

// ListPluginTypes returns all available plugin types
func (pr *PluginRegistry) ListPluginTypes() []string {
	var types []string
	for pluginType := range pr.plugins {
		types = append(types, pluginType)
	}
	return types
}
