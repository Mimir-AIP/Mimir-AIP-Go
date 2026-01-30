package utils

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"plugin"
	"strings"
	"time"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
)

// DiscoveredPlugin represents a plugin discovered from the filesystem
type DiscoveredPlugin struct {
	Name         string                 `json:"name"`
	Type         string                 `json:"type"`
	Version      string                 `json:"version"`
	Description  string                 `json:"description"`
	Author       string                 `json:"author"`
	FilePath     string                 `json:"file_path"`
	ConfigPath   string                 `json:"config_path,omitempty"`
	Config       map[string]interface{} `json:"config,omitempty"`
	DiscoveredAt time.Time              `json:"discovered_at"`
}

// PluginDiscovery handles auto-discovery of plugins from the filesystem
type PluginDiscovery struct {
	pluginDir string
	registry  *pipelines.PluginRegistry
	logger    *Logger
	db        *sql.DB
}

// NewPluginDiscovery creates a new plugin discovery instance
func NewPluginDiscovery(pluginDir string, registry *pipelines.PluginRegistry) *PluginDiscovery {
	return &PluginDiscovery{
		pluginDir: pluginDir,
		registry:  registry,
		logger:    GetLogger(),
	}
}

// SetDB sets the database for storing plugin metadata
func (pd *PluginDiscovery) SetDB(db *sql.DB) {
	pd.db = db
}

// DiscoverPlugins scans the plugins directory and discovers all .so files
func (pd *PluginDiscovery) DiscoverPlugins() ([]DiscoveredPlugin, error) {
	// Create plugins directory if it doesn't exist
	if err := os.MkdirAll(pd.pluginDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create plugins directory: %w", err)
	}

	var discovered []DiscoveredPlugin

	// Walk the plugins directory
	err := filepath.Walk(pd.pluginDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check for .so files (Linux) or .dll (Windows)
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".so" && ext != ".dll" {
			return nil
		}

		// Extract plugin info from filename and directory structure
		plugin := pd.extractPluginInfo(path)
		discovered = append(discovered, plugin)

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk plugins directory: %w", err)
	}

	pd.logger.Info("Discovered plugins",
		Int("count", len(discovered)),
		String("directory", pd.pluginDir))

	return discovered, nil
}

// extractPluginInfo extracts plugin information from the file path and directory structure
func (pd *PluginDiscovery) extractPluginInfo(filePath string) DiscoveredPlugin {
	name := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))
	dir := filepath.Dir(filePath)

	// Determine plugin type from directory name
	pluginType := "Custom"
	if relDir, err := filepath.Rel(pd.pluginDir, dir); err == nil && relDir != "." {
		// Use parent directory as type (e.g., /app/plugins/Input/plugin.so)
		typeParts := strings.Split(relDir, string(filepath.Separator))
		if len(typeParts) > 0 && typeParts[0] != "" {
			pluginType = typeParts[0]
		}
	}

	// Look for configuration file
	configPath := pd.findConfigFile(filePath)
	config := pd.loadPluginConfig(configPath)

	// Override with config values if available
	if config != nil {
		if v, ok := config["name"].(string); ok && v != "" {
			name = v
		}
		if v, ok := config["type"].(string); ok && v != "" {
			pluginType = v
		}
	}

	return DiscoveredPlugin{
		Name:         name,
		Type:         pluginType,
		Version:      "1.0.0",
		Description:  fmt.Sprintf("Auto-discovered %s plugin", pluginType),
		Author:       "Unknown",
		FilePath:     filePath,
		ConfigPath:   configPath,
		Config:       config,
		DiscoveredAt: time.Now(),
	}
}

// findConfigFile looks for a configuration file associated with the plugin
func (pd *PluginDiscovery) findConfigFile(pluginPath string) string {
	base := strings.TrimSuffix(pluginPath, filepath.Ext(pluginPath))

	// Look for config.json, config.yaml, or plugin.json
	candidates := []string{
		base + ".json",
		base + ".yaml",
		base + ".yml",
		base + "_config.json",
		filepath.Join(filepath.Dir(pluginPath), "plugin.json"),
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	return ""
}

// loadPluginConfig loads plugin configuration from a file
func (pd *PluginDiscovery) loadPluginConfig(configPath string) map[string]interface{} {
	if configPath == "" {
		return nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		// Try YAML parsing if JSON fails
		return nil
	}

	return config
}

// LoadAndRegisterPlugin loads a plugin from a .so file and registers it
func (pd *PluginDiscovery) LoadAndRegisterPlugin(discovered DiscoveredPlugin) error {
	// Open the plugin
	plug, err := plugin.Open(discovered.FilePath)
	if err != nil {
		return fmt.Errorf("failed to open plugin %s: %w", discovered.Name, err)
	}

	// Look for the plugin factory function
	factoryName := "New" + strings.Title(discovered.Name) + "Plugin"
	factory, err := plug.Lookup(factoryName)
	if err != nil {
		// Try alternative naming convention
		factory, err = plug.Lookup("NewPlugin")
		if err != nil {
			return fmt.Errorf("plugin %s missing factory function: %w", discovered.Name, err)
		}
	}

	// Invoke factory to create plugin instance
	factoryFunc, ok := factory.(func() pipelines.BasePlugin)
	if !ok {
		return fmt.Errorf("plugin %s factory has wrong signature", discovered.Name)
	}

	pluginInstance := factoryFunc()
	if pluginInstance == nil {
		return fmt.Errorf("plugin %s factory returned nil", discovered.Name)
	}

	// Register the plugin
	if err := pd.registry.RegisterPlugin(pluginInstance); err != nil {
		return fmt.Errorf("failed to register plugin %s: %w", discovered.Name, err)
	}

	pd.logger.Info("Auto-registered plugin",
		String("name", pluginInstance.GetPluginName()),
		String("type", pluginInstance.GetPluginType()),
		String("file", discovered.FilePath))

	// Store metadata in database if available
	if pd.db != nil {
		pd.storePluginMetadata(discovered, pluginInstance)
	}

	return nil
}

// storePluginMetadata stores plugin metadata in the database
func (pd *PluginDiscovery) storePluginMetadata(discovered DiscoveredPlugin, plugin pipelines.BasePlugin) {
	// This is a best-effort operation
	if pd.db == nil {
		return
	}

	// The actual database storage would be implemented here
	// For now, just log that we would store it
	pd.logger.Debug("Would store plugin metadata",
		String("name", discovered.Name),
		String("type", discovered.Type))
}

// AutoDiscoverAndRegister performs full auto-discovery and registration
func (pd *PluginDiscovery) AutoDiscoverAndRegister() error {
	plugins, err := pd.DiscoverPlugins()
	if err != nil {
		return err
	}

	successCount := 0
	for _, plugin := range plugins {
		if err := pd.LoadAndRegisterPlugin(plugin); err != nil {
			pd.logger.Warn("Failed to auto-register plugin",
				String("name", plugin.Name),
				String("error", err.Error()))
			continue
		}
		successCount++
	}

	pd.logger.Info("Auto-discovery complete",
		Int("discovered", len(plugins)),
		Int("registered", successCount))

	return nil
}

// RefreshPlugins re-scans the plugins directory and registers any new plugins
func (pd *PluginDiscovery) RefreshPlugins() error {
	pd.logger.Info("Refreshing plugins directory")
	return pd.AutoDiscoverAndRegister()
}

// InitializePluginDiscovery initializes and runs plugin discovery
func InitializePluginDiscovery(pluginDir string, registry *pipelines.PluginRegistry, db *sql.DB) {
	if pluginDir == "" {
		pluginDir = "/app/plugins"
	}

	discovery := NewPluginDiscovery(pluginDir, registry)
	if db != nil {
		discovery.SetDB(db)
	}

	if err := discovery.AutoDiscoverAndRegister(); err != nil {
		GetLogger().Warn("Plugin auto-discovery encountered errors",
			String("error", err.Error()))
	}
}
