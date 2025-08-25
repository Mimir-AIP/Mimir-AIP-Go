package tests

import (
	"testing"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
)

// TestPluginRegistry tests the plugin registry functionality
func TestPluginRegistry(t *testing.T) {
	registry := pipelines.NewPluginRegistry()

	// Test registering a plugin
	mockPlugin := NewMockPlugin("test_plugin", "Data_Processing", false)
	err := registry.RegisterPlugin(mockPlugin)
	if err != nil {
		t.Fatalf("Failed to register plugin: %v", err)
	}

	// Test retrieving the plugin
	retrievedPlugin, err := registry.GetPlugin("Data_Processing", "test_plugin")
	if err != nil {
		t.Fatalf("Failed to get plugin: %v", err)
	}

	if retrievedPlugin.GetPluginName() != "test_plugin" {
		t.Errorf("Expected plugin name 'test_plugin', got '%s'", retrievedPlugin.GetPluginName())
	}

	if retrievedPlugin.GetPluginType() != "Data_Processing" {
		t.Errorf("Expected plugin type 'Data_Processing', got '%s'", retrievedPlugin.GetPluginType())
	}
}

// TestPluginRegistry_DuplicateRegistration tests registering duplicate plugins
func TestPluginRegistry_DuplicateRegistration(t *testing.T) {
	registry := pipelines.NewPluginRegistry()

	mockPlugin1 := NewMockPlugin("duplicate_plugin", "Data_Processing", false)
	mockPlugin2 := NewMockPlugin("duplicate_plugin", "Data_Processing", false)

	// Register first plugin
	err := registry.RegisterPlugin(mockPlugin1)
	if err != nil {
		t.Fatalf("Failed to register first plugin: %v", err)
	}

	// Try to register duplicate plugin
	err = registry.RegisterPlugin(mockPlugin2)
	if err == nil {
		t.Fatal("Expected error when registering duplicate plugin, got nil")
	}
}

// TestPluginRegistry_GetPluginsByType tests getting plugins by type
func TestPluginRegistry_GetPluginsByType(t *testing.T) {
	registry := pipelines.NewPluginRegistry()

	// Register multiple plugins of different types
	registry.RegisterPlugin(NewMockPlugin("data_plugin1", "Data_Processing", false))
	registry.RegisterPlugin(NewMockPlugin("data_plugin2", "Data_Processing", false))
	registry.RegisterPlugin(NewMockPlugin("input_plugin1", "Input", false))

	// Get Data_Processing plugins
	dataPlugins := registry.GetPluginsByType("Data_Processing")
	if len(dataPlugins) != 2 {
		t.Fatalf("Expected 2 Data_Processing plugins, got %d", len(dataPlugins))
	}

	// Get Input plugins
	inputPlugins := registry.GetPluginsByType("Input")
	if len(inputPlugins) != 1 {
		t.Fatalf("Expected 1 Input plugin, got %d", len(inputPlugins))
	}

	// Get non-existent type
	emptyPlugins := registry.GetPluginsByType("NonExistent")
	if len(emptyPlugins) != 0 {
		t.Fatalf("Expected 0 plugins for non-existent type, got %d", len(emptyPlugins))
	}
}

// TestPluginRegistry_GetAllPlugins tests getting all registered plugins
func TestPluginRegistry_GetAllPlugins(t *testing.T) {
	registry := pipelines.NewPluginRegistry()

	// Register plugins of different types
	registry.RegisterPlugin(NewMockPlugin("data_plugin", "Data_Processing", false))
	registry.RegisterPlugin(NewMockPlugin("input_plugin", "Input", false))
	registry.RegisterPlugin(NewMockPlugin("output_plugin", "Output", false))

	allPlugins := registry.GetAllPlugins()

	// Check that we have 3 types
	if len(allPlugins) != 3 {
		t.Fatalf("Expected 3 plugin types, got %d", len(allPlugins))
	}

	// Check each type has the correct count
	if len(allPlugins["Data_Processing"]) != 1 {
		t.Errorf("Expected 1 Data_Processing plugin, got %d", len(allPlugins["Data_Processing"]))
	}
	if len(allPlugins["Input"]) != 1 {
		t.Errorf("Expected 1 Input plugin, got %d", len(allPlugins["Input"]))
	}
	if len(allPlugins["Output"]) != 1 {
		t.Errorf("Expected 1 Output plugin, got %d", len(allPlugins["Output"]))
	}
}

// TestPluginRegistry_ListPluginTypes tests listing all plugin types
func TestPluginRegistry_ListPluginTypes(t *testing.T) {
	registry := pipelines.NewPluginRegistry()

	// Register plugins of different types
	registry.RegisterPlugin(NewMockPlugin("data_plugin", "Data_Processing", false))
	registry.RegisterPlugin(NewMockPlugin("input_plugin", "Input", false))
	registry.RegisterPlugin(NewMockPlugin("ai_plugin", "AIModels", false))

	types := registry.ListPluginTypes()

	// Check that we have 3 types
	if len(types) != 3 {
		t.Fatalf("Expected 3 plugin types, got %d", len(types))
	}

	// Check that all expected types are present
	expectedTypes := map[string]bool{
		"Data_Processing": false,
		"Input":           false,
		"AIModels":        false,
	}

	for _, pluginType := range types {
		if _, exists := expectedTypes[pluginType]; exists {
			expectedTypes[pluginType] = true
		} else {
			t.Errorf("Unexpected plugin type: %s", pluginType)
		}
	}

	// Check that all expected types were found
	for pluginType, found := range expectedTypes {
		if !found {
			t.Errorf("Expected plugin type not found: %s", pluginType)
		}
	}
}

// TestPluginValidation tests plugin configuration validation
func TestPluginValidation(t *testing.T) {
	mockPlugin := NewMockPlugin("validation_plugin", "Data_Processing", false)

	// Test valid configuration
	err := mockPlugin.ValidateConfig(map[string]interface{}{})
	if err != nil {
		t.Fatalf("Valid configuration should not fail: %v", err)
	}

	// Test invalid configuration
	err = mockPlugin.ValidateConfig(map[string]interface{}{"invalid": true})
	if err == nil {
		t.Fatal("Invalid configuration should fail validation")
	}
}

// BenchmarkPluginRegistry benchmarks plugin registry operations
func BenchmarkPluginRegistry(b *testing.B) {
	registry := pipelines.NewPluginRegistry()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		plugin := NewMockPlugin("benchmark_plugin", "Data_Processing", false)
		err := registry.RegisterPlugin(plugin)
		if err != nil {
			b.Fatalf("Plugin registration failed: %v", err)
		}

		_, err = registry.GetPlugin("Data_Processing", "benchmark_plugin")
		if err != nil {
			b.Fatalf("Plugin retrieval failed: %v", err)
		}
	}
}
