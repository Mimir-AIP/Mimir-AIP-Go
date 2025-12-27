package pipelines

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// MockPlugin for testing
type MockPlugin struct {
	pluginType string
	pluginName string
	shouldFail bool
}

func (m *MockPlugin) ExecuteStep(ctx context.Context, stepConfig StepConfig, globalContext *PluginContext) (*PluginContext, error) {
	if m.shouldFail {
		return nil, assert.AnError
	}

	result := NewPluginContext()
	result.Set(stepConfig.Output, map[string]any{
		"plugin":  m.pluginName,
		"success": true,
	})
	return result, nil
}

func (m *MockPlugin) GetPluginType() string { return m.pluginType }
func (m *MockPlugin) GetPluginName() string { return m.pluginName }
func (m *MockPlugin) ValidateConfig(config map[string]any) error {
	if m.shouldFail {
		return assert.AnError
	}
	return nil
}

func (m *MockPlugin) GetInputSchema() map[string]any {
	return map[string]any{}
}

func TestNewPluginContext(t *testing.T) {
	ctx := NewPluginContext()

	assert.NotNil(t, ctx)
	assert.NotNil(t, ctx.data)
	assert.NotNil(t, ctx.metadata)
	assert.Equal(t, 0, ctx.Size())
}

func TestPluginContextSetAndGet(t *testing.T) {
	ctx := NewPluginContext()

	// Test setting and getting a value
	testData := map[string]any{"key": "value"}
	ctx.Set("test_key", testData)

	value, exists := ctx.Get("test_key")
	assert.True(t, exists)
	assert.Equal(t, testData, value)
}

func TestPluginContextSetAndGetTyped(t *testing.T) {
	ctx := NewPluginContext()

	// Test setting and getting typed data
	jsonData := NewJSONData(map[string]any{"key": "value"})
	ctx.SetTyped("typed_key", jsonData)

	retrievedData, exists := ctx.GetTyped("typed_key")
	assert.True(t, exists)
	assert.Equal(t, jsonData, retrievedData)
	assert.Equal(t, "json", retrievedData.Type())
}

func TestPluginContextDelete(t *testing.T) {
	ctx := NewPluginContext()

	// Set a value
	ctx.Set("test_key", "test_value")

	// Verify it exists
	_, exists := ctx.Get("test_key")
	assert.True(t, exists)

	// Delete it
	ctx.Delete("test_key")

	// Verify it's gone
	_, exists = ctx.Get("test_key")
	assert.False(t, exists)
}

func TestPluginContextKeys(t *testing.T) {
	ctx := NewPluginContext()

	// Initially empty
	assert.Equal(t, 0, len(ctx.Keys()))

	// Add some keys
	ctx.Set("key1", "value1")
	ctx.Set("key2", "value2")
	ctx.Set("key3", "value3")

	keys := ctx.Keys()
	assert.Equal(t, 3, len(keys))
	assert.Contains(t, keys, "key1")
	assert.Contains(t, keys, "key2")
	assert.Contains(t, keys, "key3")
}

func TestPluginContextSize(t *testing.T) {
	ctx := NewPluginContext()

	assert.Equal(t, 0, ctx.Size())

	ctx.Set("key1", "value1")
	assert.Equal(t, 1, ctx.Size())

	ctx.Set("key2", "value2")
	assert.Equal(t, 2, ctx.Size())

	ctx.Delete("key1")
	assert.Equal(t, 1, ctx.Size())
}

func TestPluginContextClear(t *testing.T) {
	ctx := NewPluginContext()

	// Add some data
	ctx.Set("key1", "value1")
	ctx.Set("key2", "value2")
	ctx.SetMetadata("meta1", "meta_value")

	assert.Equal(t, 2, ctx.Size())

	// Clear everything
	ctx.Clear()

	assert.Equal(t, 0, ctx.Size())
	assert.Equal(t, 0, len(ctx.Keys()))

	// Check metadata is also cleared
	_, exists := ctx.GetMetadata("meta1")
	assert.False(t, exists)
}

func TestPluginContextClone(t *testing.T) {
	ctx := NewPluginContext()

	// Add some data
	testData := map[string]any{"key": "value"}
	ctx.Set("key1", testData)
	ctx.SetMetadata("meta1", "meta_value")

	// Clone the context
	clonedCtx := ctx.Clone()

	// Note: Clone returns a new PluginContext but we test functionality over pointer equality
	assert.NotNil(t, clonedCtx)
	assert.Equal(t, ctx.Size(), clonedCtx.Size())

	// Verify data is the same but not the same objects
	originalValue, _ := ctx.Get("key1")
	clonedValue, _ := clonedCtx.Get("key1")
	assert.Equal(t, originalValue, clonedValue)

	// Verify metadata is copied
	originalMeta, _ := ctx.GetMetadata("meta1")
	clonedMeta, _ := clonedCtx.GetMetadata("meta1")
	assert.Equal(t, originalMeta, clonedMeta)

	// Modify original and verify clone is unaffected
	ctx.Set("new_key", "new_value")
	assert.Equal(t, 2, ctx.Size())
	assert.Equal(t, 1, clonedCtx.Size())
}

func TestPluginContextMetadata(t *testing.T) {
	ctx := NewPluginContext()

	// Test setting and getting metadata
	ctx.SetMetadata("meta1", "meta_value1")
	ctx.SetMetadata("meta2", 42)

	value1, exists1 := ctx.GetMetadata("meta1")
	value2, exists2 := ctx.GetMetadata("meta2")

	assert.True(t, exists1)
	assert.Equal(t, "meta_value1", value1)
	assert.True(t, exists2)
	assert.Equal(t, 42, value2)

	// Test non-existent metadata
	_, exists3 := ctx.GetMetadata("nonexistent")
	assert.False(t, exists3)
}

func TestPluginContextAutoWrapping(t *testing.T) {
	ctx := NewPluginContext()

	// Test auto-wrapping of different types
	ctx.Set("map_key", map[string]any{"key": "value"})
	ctx.Set("bytes_key", []byte("test data"))
	ctx.Set("string_key", "test string")
	ctx.Set("int_key", 42)

	// Check map is wrapped in JSONData
	mapData, exists := ctx.GetTyped("map_key")
	assert.True(t, exists)
	assert.Equal(t, "json", mapData.Type())

	// Check bytes are wrapped in BinaryData
	bytesData, exists := ctx.GetTyped("bytes_key")
	assert.True(t, exists)
	assert.Equal(t, "binary", bytesData.Type())

	// Check other types are wrapped in JSONData
	stringData, exists := ctx.GetTyped("string_key")
	assert.True(t, exists)
	assert.Equal(t, "json", stringData.Type())

	intData, exists := ctx.GetTyped("int_key")
	assert.True(t, exists)
	assert.Equal(t, "json", intData.Type())
}

func TestNewPluginRegistry(t *testing.T) {
	registry := NewPluginRegistry()

	assert.NotNil(t, registry)
	assert.NotNil(t, registry.plugins)
	assert.Equal(t, 0, len(registry.ListPluginTypes()))
}

func TestPluginRegistryRegisterPlugin(t *testing.T) {
	registry := NewPluginRegistry()
	plugin := &MockPlugin{pluginType: "Input", pluginName: "test"}

	// Register plugin
	err := registry.RegisterPlugin(plugin)
	assert.NoError(t, err)

	// Check plugin is registered
	retrievedPlugin, err := registry.GetPlugin("Input", "test")
	assert.NoError(t, err)
	assert.Equal(t, plugin, retrievedPlugin)

	// Check plugin types
	types := registry.ListPluginTypes()
	assert.Equal(t, 1, len(types))
	assert.Contains(t, types, "Input")
}

func TestPluginRegistryRegisterDuplicatePlugin(t *testing.T) {
	registry := NewPluginRegistry()
	plugin1 := &MockPlugin{pluginType: "Input", pluginName: "test"}
	plugin2 := &MockPlugin{pluginType: "Input", pluginName: "test"}

	// Register first plugin
	err := registry.RegisterPlugin(plugin1)
	assert.NoError(t, err)

	// Try to register duplicate
	err = registry.RegisterPlugin(plugin2)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
}

func TestPluginRegistryGetPlugin(t *testing.T) {
	registry := NewPluginRegistry()
	plugin := &MockPlugin{pluginType: "Input", pluginName: "test"}

	// Test getting non-existent plugin
	_, err := registry.GetPlugin("Input", "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	_, err = registry.GetPlugin("Nonexistent", "test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	// Register plugin and get it
	err = registry.RegisterPlugin(plugin)
	assert.NoError(t, err)

	retrievedPlugin, err := registry.GetPlugin("Input", "test")
	assert.NoError(t, err)
	assert.Equal(t, plugin, retrievedPlugin)
}

func TestPluginRegistryGetPluginsByType(t *testing.T) {
	registry := NewPluginRegistry()

	// Register multiple plugins of same type
	plugin1 := &MockPlugin{pluginType: "Input", pluginName: "test1"}
	plugin2 := &MockPlugin{pluginType: "Input", pluginName: "test2"}
	plugin3 := &MockPlugin{pluginType: "Output", pluginName: "test3"}

	_ = registry.RegisterPlugin(plugin1)
	_ = registry.RegisterPlugin(plugin2)
	_ = registry.RegisterPlugin(plugin3)

	// Get plugins by type
	inputPlugins := registry.GetPluginsByType("Input")
	assert.Equal(t, 2, len(inputPlugins))
	assert.Equal(t, plugin1, inputPlugins["test1"])
	assert.Equal(t, plugin2, inputPlugins["test2"])

	outputPlugins := registry.GetPluginsByType("Output")
	assert.Equal(t, 1, len(outputPlugins))
	assert.Equal(t, plugin3, outputPlugins["test3"])

	// Get non-existent type
	nonexistentPlugins := registry.GetPluginsByType("Nonexistent")
	assert.Equal(t, 0, len(nonexistentPlugins))
}

func TestPluginRegistryGetAllPlugins(t *testing.T) {
	registry := NewPluginRegistry()

	// Register plugins
	plugin1 := &MockPlugin{pluginType: "Input", pluginName: "test1"}
	plugin2 := &MockPlugin{pluginType: "Output", pluginName: "test2"}

	_ = registry.RegisterPlugin(plugin1)
	_ = registry.RegisterPlugin(plugin2)

	// Get all plugins
	allPlugins := registry.GetAllPlugins()
	assert.Equal(t, 2, len(allPlugins))
	assert.Equal(t, plugin1, allPlugins["Input"]["test1"])
	assert.Equal(t, plugin2, allPlugins["Output"]["test2"])
}

func TestPluginRegistryListPluginTypes(t *testing.T) {
	registry := NewPluginRegistry()

	// Initially empty
	assert.Equal(t, 0, len(registry.ListPluginTypes()))

	// Register plugins
	plugin1 := &MockPlugin{pluginType: "Input", pluginName: "test1"}
	plugin2 := &MockPlugin{pluginType: "Output", pluginName: "test2"}
	plugin3 := &MockPlugin{pluginType: "Processing", pluginName: "test3"}

	_ = registry.RegisterPlugin(plugin1)
	_ = registry.RegisterPlugin(plugin2)
	_ = registry.RegisterPlugin(plugin3)

	// List types
	types := registry.ListPluginTypes()
	assert.Equal(t, 3, len(types))
	assert.Contains(t, types, "Input")
	assert.Contains(t, types, "Output")
	assert.Contains(t, types, "Processing")
}

func TestStepConfig(t *testing.T) {
	config := StepConfig{
		Name:   "test-step",
		Plugin: "Input.test",
		Config: map[string]any{
			"param1": "value1",
			"param2": 42,
		},
		Output: "output-key",
	}

	assert.Equal(t, "test-step", config.Name)
	assert.Equal(t, "Input.test", config.Plugin)
	assert.Equal(t, "value1", config.Config["param1"])
	assert.Equal(t, 42, config.Config["param2"])
	assert.Equal(t, "output-key", config.Output)
}

func TestBasePluginInterface(t *testing.T) {
	plugin := &MockPlugin{pluginType: "Test", pluginName: "mock"}

	// Test interface methods
	assert.Equal(t, "Test", plugin.GetPluginType())
	assert.Equal(t, "mock", plugin.GetPluginName())

	// Test validation
	err := plugin.ValidateConfig(map[string]any{})
	assert.NoError(t, err)

	// Test execution
	ctx := context.Background()
	stepConfig := StepConfig{Name: "test", Plugin: "Test.mock"}
	globalContext := NewPluginContext()

	result, err := plugin.ExecuteStep(ctx, stepConfig, globalContext)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestPluginContextConcurrency(t *testing.T) {
	ctx := NewPluginContext()

	// Test concurrent access
	done := make(chan bool, 2)

	// Goroutine 1: write data
	go func() {
		for i := 0; i < 100; i++ {
			ctx.Set("key1", i)
		}
		done <- true
	}()

	// Goroutine 2: read data
	go func() {
		for i := 0; i < 100; i++ {
			ctx.Get("key1")
			ctx.Size()
		}
		done <- true
	}()

	// Wait for both to complete
	<-done
	<-done

	// Should not have any race conditions
	assert.True(t, true)
}

func TestPluginContextComplexDataTypes(t *testing.T) {
	ctx := NewPluginContext()

	// Test with complex nested data
	complexData := map[string]any{
		"nested": map[string]any{
			"array":  []any{1, 2, 3},
			"string": "test",
		},
		"number": 42,
	}

	ctx.Set("complex", complexData)

	retrieved, exists := ctx.Get("complex")
	assert.True(t, exists)
	assert.Equal(t, complexData, retrieved)

	// Test with time series data
	timePoints := []TimePoint{
		{Timestamp: time.Now(), Value: 100},
		{Timestamp: time.Now(), Value: 200},
	}

	ctx.Set("timeseries", timePoints)

	retrievedTS, exists := ctx.Get("timeseries")
	assert.True(t, exists)
	assert.Equal(t, timePoints, retrievedTS)
}
