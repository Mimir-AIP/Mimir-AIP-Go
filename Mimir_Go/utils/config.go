package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// Config represents the main application configuration
type Config struct {
	Server    ServerConfig    `yaml:"server" json:"server"`
	Plugins   PluginsConfig   `yaml:"plugins" json:"plugins"`
	Scheduler SchedulerConfig `yaml:"scheduler" json:"scheduler"`
	Logging   LoggingConfig   `yaml:"logging" json:"logging"`
	Security  SecurityConfig  `yaml:"security" json:"security"`
}

// ServerConfig holds server-related configuration
type ServerConfig struct {
	Host         string `yaml:"host" json:"host"`
	Port         int    `yaml:"port" json:"port"`
	ReadTimeout  int    `yaml:"read_timeout" json:"read_timeout"`   // seconds
	WriteTimeout int    `yaml:"write_timeout" json:"write_timeout"` // seconds
	EnableCORS   bool   `yaml:"enable_cors" json:"enable_cors"`
	MaxRequests  int    `yaml:"max_requests" json:"max_requests"`
}

// PluginsConfig holds plugin-related configuration
type PluginsConfig struct {
	Directories    []string               `yaml:"directories" json:"directories"`
	AutoDiscovery  bool                   `yaml:"auto_discovery" json:"auto_discovery"`
	Timeout        int                    `yaml:"timeout" json:"timeout"` // seconds
	MaxConcurrency int                    `yaml:"max_concurrency" json:"max_concurrency"`
	PluginConfigs  map[string]interface{} `yaml:"plugin_configs" json:"plugin_configs"`
}

// SchedulerConfig holds scheduler-related configuration
type SchedulerConfig struct {
	Enabled          bool   `yaml:"enabled" json:"enabled"`
	MaxJobs          int    `yaml:"max_jobs" json:"max_jobs"`
	DefaultTimeout   int    `yaml:"default_timeout" json:"default_timeout"`     // seconds
	HistoryRetention int    `yaml:"history_retention" json:"history_retention"` // days
	WorkingDirectory string `yaml:"working_directory" json:"working_directory"`
}

// LoggingConfig holds logging-related configuration
type LoggingConfig struct {
	Level      string `yaml:"level" json:"level"`   // debug, info, warn, error
	Format     string `yaml:"format" json:"format"` // json, text
	Output     string `yaml:"output" json:"output"` // stdout, file, both
	FilePath   string `yaml:"file_path" json:"file_path"`
	MaxSize    int    `yaml:"max_size" json:"max_size"` // MB
	MaxBackups int    `yaml:"max_backups" json:"max_backups"`
	MaxAge     int    `yaml:"max_age" json:"max_age"` // days
	Compress   bool   `yaml:"compress" json:"compress"`
}

// SecurityConfig holds security-related configuration
type SecurityConfig struct {
	EnableAuth     bool     `yaml:"enable_auth" json:"enable_auth"`
	JWTSecret      string   `yaml:"jwt_secret" json:"jwt_secret"`
	TokenExpiry    int      `yaml:"token_expiry" json:"token_expiry"` // hours
	AllowedOrigins []string `yaml:"allowed_origins" json:"allowed_origins"`
	RateLimit      int      `yaml:"rate_limit" json:"rate_limit"` // requests per minute
	EnableHTTPS    bool     `yaml:"enable_https" json:"enable_https"`
	CertFile       string   `yaml:"cert_file" json:"cert_file"`
	KeyFile        string   `yaml:"key_file" json:"key_file"`
}

// ConfigManager manages application configuration
type ConfigManager struct {
	config     *Config
	configPath string
	mutex      sync.RWMutex
	watchers   []ConfigChangeWatcher
}

// ConfigChangeWatcher is called when configuration changes
type ConfigChangeWatcher interface {
	OnConfigChange(oldConfig, newConfig *Config)
}

// NewConfigManager creates a new configuration manager
func NewConfigManager() *ConfigManager {
	return &ConfigManager{
		config:   getDefaultConfig(),
		watchers: make([]ConfigChangeWatcher, 0),
	}
}

// LoadFromFile loads configuration from a file
func (cm *ConfigManager) LoadFromFile(configPath string) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var newConfig Config
	ext := filepath.Ext(configPath)

	switch strings.ToLower(ext) {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &newConfig); err != nil {
			return fmt.Errorf("failed to parse YAML config: %w", err)
		}
	case ".json":
		if err := json.Unmarshal(data, &newConfig); err != nil {
			return fmt.Errorf("failed to parse JSON config: %w", err)
		}
	default:
		return fmt.Errorf("unsupported config file format: %s", ext)
	}

	// Merge with defaults
	mergedConfig := cm.mergeWithDefaults(&newConfig)

	// Validate configuration
	if err := cm.validateConfig(mergedConfig); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Notify watchers
	oldConfig := cm.config
	for _, watcher := range cm.watchers {
		watcher.OnConfigChange(oldConfig, mergedConfig)
	}

	cm.config = mergedConfig
	cm.configPath = configPath

	return nil
}

// LoadFromEnvironment loads configuration from environment variables
func (cm *ConfigManager) LoadFromEnvironment() error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	// Override config with environment variables
	if host := os.Getenv("MIMIR_HOST"); host != "" {
		cm.config.Server.Host = host
	}

	if port := os.Getenv("MIMIR_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			cm.config.Server.Port = p
		}
	}

	if logLevel := os.Getenv("MIMIR_LOG_LEVEL"); logLevel != "" {
		cm.config.Logging.Level = logLevel
	}

	if jwtSecret := os.Getenv("MIMIR_JWT_SECRET"); jwtSecret != "" {
		cm.config.Security.JWTSecret = jwtSecret
	}

	return nil
}

// SaveToFile saves current configuration to a file
func (cm *ConfigManager) SaveToFile(configPath string) error {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	var data []byte
	var err error

	ext := filepath.Ext(configPath)
	switch strings.ToLower(ext) {
	case ".yaml", ".yml":
		data, err = yaml.Marshal(cm.config)
		if err != nil {
			return fmt.Errorf("failed to marshal config to YAML: %w", err)
		}
	case ".json":
		data, err = json.MarshalIndent(cm.config, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal config to JSON: %w", err)
		}
	default:
		return fmt.Errorf("unsupported config file format: %s", ext)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	cm.configPath = configPath
	return nil
}

// GetConfig returns the current configuration
func (cm *ConfigManager) GetConfig() *Config {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	// Return a copy to prevent external modifications
	configCopy := *cm.config
	return &configCopy
}

// UpdateConfig updates the configuration
func (cm *ConfigManager) UpdateConfig(updates *Config) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	// Merge updates with current config
	mergedConfig := cm.mergeConfigs(cm.config, updates)

	// Validate new configuration
	if err := cm.validateConfig(mergedConfig); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Notify watchers
	oldConfig := cm.config
	for _, watcher := range cm.watchers {
		watcher.OnConfigChange(oldConfig, mergedConfig)
	}

	cm.config = mergedConfig
	return nil
}

// AddWatcher adds a configuration change watcher
func (cm *ConfigManager) AddWatcher(watcher ConfigChangeWatcher) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	cm.watchers = append(cm.watchers, watcher)
}

// RemoveWatcher removes a configuration change watcher
func (cm *ConfigManager) RemoveWatcher(watcher ConfigChangeWatcher) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	for i, w := range cm.watchers {
		if w == watcher {
			cm.watchers = append(cm.watchers[:i], cm.watchers[i+1:]...)
			break
		}
	}
}

// GetConfigPath returns the current configuration file path
func (cm *ConfigManager) GetConfigPath() string {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	return cm.configPath
}

// getDefaultConfig returns the default configuration
func getDefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host:         "0.0.0.0",
			Port:         8080,
			ReadTimeout:  30,
			WriteTimeout: 30,
			EnableCORS:   true,
			MaxRequests:  1000,
		},
		Plugins: PluginsConfig{
			Directories:    []string{"./pipelines"},
			AutoDiscovery:  true,
			Timeout:        60,
			MaxConcurrency: 10,
			PluginConfigs:  make(map[string]interface{}),
		},
		Scheduler: SchedulerConfig{
			Enabled:          true,
			MaxJobs:          100,
			DefaultTimeout:   300,
			HistoryRetention: 30,
			WorkingDirectory: "./",
		},
		Logging: LoggingConfig{
			Level:      "info",
			Format:     "text",
			Output:     "stdout",
			FilePath:   "./logs/mimir.log",
			MaxSize:    100,
			MaxBackups: 3,
			MaxAge:     30,
			Compress:   true,
		},
		Security: SecurityConfig{
			EnableAuth:     false,
			JWTSecret:      "change-me-in-production",
			TokenExpiry:    24,
			AllowedOrigins: []string{"*"},
			RateLimit:      1000,
			EnableHTTPS:    false,
			CertFile:       "",
			KeyFile:        "",
		},
	}
}

// mergeWithDefaults merges user config with defaults
func (cm *ConfigManager) mergeWithDefaults(userConfig *Config) *Config {
	defaultConfig := getDefaultConfig()

	// Start with defaults
	merged := *defaultConfig

	// Override server config
	if userConfig.Server.Host != "" {
		merged.Server.Host = userConfig.Server.Host
	}
	if userConfig.Server.Port != 0 {
		merged.Server.Port = userConfig.Server.Port
	}
	if userConfig.Server.ReadTimeout != 0 {
		merged.Server.ReadTimeout = userConfig.Server.ReadTimeout
	}
	if userConfig.Server.WriteTimeout != 0 {
		merged.Server.WriteTimeout = userConfig.Server.WriteTimeout
	}
	if userConfig.Server.MaxRequests != 0 {
		merged.Server.MaxRequests = userConfig.Server.MaxRequests
	}
	merged.Server.EnableCORS = userConfig.Server.EnableCORS

	// Override plugins config
	if len(userConfig.Plugins.Directories) > 0 {
		merged.Plugins.Directories = userConfig.Plugins.Directories
	}
	merged.Plugins.AutoDiscovery = userConfig.Plugins.AutoDiscovery
	if userConfig.Plugins.Timeout != 0 {
		merged.Plugins.Timeout = userConfig.Plugins.Timeout
	}
	if userConfig.Plugins.MaxConcurrency != 0 {
		merged.Plugins.MaxConcurrency = userConfig.Plugins.MaxConcurrency
	}
	if userConfig.Plugins.PluginConfigs != nil {
		merged.Plugins.PluginConfigs = userConfig.Plugins.PluginConfigs
	}

	// Override scheduler config
	merged.Scheduler.Enabled = userConfig.Scheduler.Enabled
	if userConfig.Scheduler.MaxJobs != 0 {
		merged.Scheduler.MaxJobs = userConfig.Scheduler.MaxJobs
	}
	if userConfig.Scheduler.DefaultTimeout != 0 {
		merged.Scheduler.DefaultTimeout = userConfig.Scheduler.DefaultTimeout
	}
	if userConfig.Scheduler.HistoryRetention != 0 {
		merged.Scheduler.HistoryRetention = userConfig.Scheduler.HistoryRetention
	}
	if userConfig.Scheduler.WorkingDirectory != "" {
		merged.Scheduler.WorkingDirectory = userConfig.Scheduler.WorkingDirectory
	}

	// Override logging config
	if userConfig.Logging.Level != "" {
		merged.Logging.Level = userConfig.Logging.Level
	}
	if userConfig.Logging.Format != "" {
		merged.Logging.Format = userConfig.Logging.Format
	}
	if userConfig.Logging.Output != "" {
		merged.Logging.Output = userConfig.Logging.Output
	}
	if userConfig.Logging.FilePath != "" {
		merged.Logging.FilePath = userConfig.Logging.FilePath
	}
	if userConfig.Logging.MaxSize != 0 {
		merged.Logging.MaxSize = userConfig.Logging.MaxSize
	}
	if userConfig.Logging.MaxBackups != 0 {
		merged.Logging.MaxBackups = userConfig.Logging.MaxBackups
	}
	if userConfig.Logging.MaxAge != 0 {
		merged.Logging.MaxAge = userConfig.Logging.MaxAge
	}
	merged.Logging.Compress = userConfig.Logging.Compress

	// Override security config
	merged.Security.EnableAuth = userConfig.Security.EnableAuth
	if userConfig.Security.JWTSecret != "" {
		merged.Security.JWTSecret = userConfig.Security.JWTSecret
	}
	if userConfig.Security.TokenExpiry != 0 {
		merged.Security.TokenExpiry = userConfig.Security.TokenExpiry
	}
	if len(userConfig.Security.AllowedOrigins) > 0 {
		merged.Security.AllowedOrigins = userConfig.Security.AllowedOrigins
	}
	if userConfig.Security.RateLimit != 0 {
		merged.Security.RateLimit = userConfig.Security.RateLimit
	}
	merged.Security.EnableHTTPS = userConfig.Security.EnableHTTPS
	if userConfig.Security.CertFile != "" {
		merged.Security.CertFile = userConfig.Security.CertFile
	}
	if userConfig.Security.KeyFile != "" {
		merged.Security.KeyFile = userConfig.Security.KeyFile
	}

	return &merged
}

// mergeConfigs merges two configurations
func (cm *ConfigManager) mergeConfigs(base, updates *Config) *Config {
	merged := *base

	// This is a simplified merge - in production, you'd want a more sophisticated merge
	if updates.Server.Host != "" {
		merged.Server.Host = updates.Server.Host
	}
	if updates.Server.Port != 0 {
		merged.Server.Port = updates.Server.Port
	}
	if updates.Logging.Level != "" {
		merged.Logging.Level = updates.Logging.Level
	}

	return &merged
}

// validateConfig validates the configuration
func (cm *ConfigManager) validateConfig(config *Config) error {
	// Validate server config
	if config.Server.Port < 1 || config.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", config.Server.Port)
	}

	// Validate logging level
	validLevels := []string{"debug", "info", "warn", "error"}
	if !contains(validLevels, config.Logging.Level) {
		return fmt.Errorf("invalid log level: %s", config.Logging.Level)
	}

	// Validate log format
	validFormats := []string{"json", "text"}
	if !contains(validFormats, config.Logging.Format) {
		return fmt.Errorf("invalid log format: %s", config.Logging.Format)
	}

	// Validate log output
	validOutputs := []string{"stdout", "file", "both"}
	if !contains(validOutputs, config.Logging.Output) {
		return fmt.Errorf("invalid log output: %s", config.Logging.Output)
	}

	// Validate plugin directories exist
	for _, dir := range config.Plugins.Directories {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			return fmt.Errorf("plugin directory does not exist: %s", dir)
		}
	}

	return nil
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// Global configuration manager instance
var globalConfigManager *ConfigManager
var configOnce sync.Once

// GetConfigManager returns the global configuration manager instance
func GetConfigManager() *ConfigManager {
	configOnce.Do(func() {
		globalConfigManager = NewConfigManager()
	})
	return globalConfigManager
}

// LoadGlobalConfig loads configuration from default locations
func LoadGlobalConfig() error {
	cm := GetConfigManager()

	// Try to load from various config file locations
	configPaths := []string{
		"./config.yaml",
		"./config.yml",
		"./mimir.yaml",
		"./mimir.yml",
		"./config.json",
		"./mimir.json",
		"/etc/mimir/config.yaml",
		"/etc/mimir/config.yml",
	}

	for _, path := range configPaths {
		if _, err := os.Stat(path); err == nil {
			if err := cm.LoadFromFile(path); err == nil {
				break
			}
		}
	}

	// Load environment variables (overrides file config)
	return cm.LoadFromEnvironment()
}
