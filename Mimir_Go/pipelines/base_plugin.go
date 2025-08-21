package pipelines

import (
	"context"
	"fmt"
)

// PluginContext represents the execution context passed between pipeline steps
type PluginContext map[string]interface{}

// StepConfig represents the configuration for a single pipeline step
type StepConfig struct {
	Name   string                 `yaml:"name"`
	Plugin string                 `yaml:"plugin"`
	Config map[string]interface{} `yaml:"config"`
	Output string                 `yaml:"output,omitempty"`
}

// BasePlugin defines the interface that all plugins must implement
type BasePlugin interface {
	// ExecuteStep executes a single pipeline step
	ExecuteStep(ctx context.Context, stepConfig StepConfig, globalContext PluginContext) (PluginContext, error)

	// GetPluginType returns the type of plugin (Input, Data_Processing, AIModels, Output)
	GetPluginType() string

	// GetPluginName returns the name of the plugin
	GetPluginName() string

	// ValidateConfig validates the plugin configuration
	ValidateConfig(config map[string]interface{}) error
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
