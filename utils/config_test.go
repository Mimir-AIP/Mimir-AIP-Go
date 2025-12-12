package utils

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestNewConfigManager(t *testing.T) {
	cm := NewConfigManager()

	assert.NotNil(t, cm)
	assert.NotNil(t, cm.config)
	assert.Empty(t, cm.configPath)
	assert.Empty(t, cm.watchers)
	assert.Equal(t, "0.0.0.0", cm.config.Server.Host)
	assert.Equal(t, 8080, cm.config.Server.Port)
	assert.Equal(t, "info", cm.config.Logging.Level)
}

func TestConfigManager_LoadFromFile_YAML(t *testing.T) {
	// Create temporary plugin directory
	tempDir := t.TempDir()
	pluginDir := filepath.Join(tempDir, "pipelines")
	err := os.MkdirAll(pluginDir, 0755)
	require.NoError(t, err)

	cm := NewConfigManager()

	configContent := `
server:
  host: "127.0.0.1"
  port: 9000
  read_timeout: 60
  write_timeout: 60
  enable_cors: false
  max_requests: 500
plugins:
  directories:
    - "` + pluginDir + `"
  auto_discovery: false
  timeout: 120
  max_concurrency: 20
logging:
  level: "debug"
  format: "json"
  output: "file"
  file_path: "/tmp/test.log"
  max_size: 200
  max_backups: 5
  max_age: 60
  compress: false
security:
  enable_auth: true
  jwt_secret: "test-secret"
  token_expiry: 48
  allowed_origins:
    - "https://example.com"
  rate_limit: 500
  enable_https: true
  cert_file: "/tmp/cert.pem"
  key_file: "/tmp/key.pem"
`

	configFile := filepath.Join(tempDir, "config.yaml")
	err = os.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)

	err = cm.LoadFromFile(configFile)
	assert.NoError(t, err)
	assert.Equal(t, configFile, cm.GetConfigPath())

	config := cm.GetConfig()
	assert.Equal(t, "127.0.0.1", config.Server.Host)
	assert.Equal(t, 9000, config.Server.Port)
	assert.Equal(t, 60, config.Server.ReadTimeout)
	assert.Equal(t, 60, config.Server.WriteTimeout)
	assert.False(t, config.Server.EnableCORS)
	assert.Equal(t, 500, config.Server.MaxRequests)

	assert.Equal(t, []string{pluginDir}, config.Plugins.Directories)
	assert.False(t, config.Plugins.AutoDiscovery)
	assert.Equal(t, 120, config.Plugins.Timeout)
	assert.Equal(t, 20, config.Plugins.MaxConcurrency)

	assert.Equal(t, "debug", config.Logging.Level)
	assert.Equal(t, "json", config.Logging.Format)
	assert.Equal(t, "file", config.Logging.Output)
	assert.Equal(t, "/tmp/test.log", config.Logging.FilePath)
	assert.Equal(t, 200, config.Logging.MaxSize)
	assert.Equal(t, 5, config.Logging.MaxBackups)
	assert.Equal(t, 60, config.Logging.MaxAge)
	assert.False(t, config.Logging.Compress)

	assert.True(t, config.Security.EnableAuth)
	assert.Equal(t, "test-secret", config.Security.JWTSecret)
	assert.Equal(t, 48, config.Security.TokenExpiry)
	assert.Equal(t, []string{"https://example.com"}, config.Security.AllowedOrigins)
	assert.Equal(t, 500, config.Security.RateLimit)
	assert.True(t, config.Security.EnableHTTPS)
	assert.Equal(t, "/tmp/cert.pem", config.Security.CertFile)
	assert.Equal(t, "/tmp/key.pem", config.Security.KeyFile)
}

func TestConfigManager_LoadFromFile_JSON(t *testing.T) {
	tempDir := t.TempDir()
	pluginDir := filepath.Join(tempDir, "pipelines")
	err := os.MkdirAll(pluginDir, 0755)
	require.NoError(t, err)

	cm := NewConfigManager()

	configContent := `{
  "server": {
    "host": "192.168.1.1",
    "port": 8081
  },
  "logging": {
    "level": "warn",
    "format": "text"
  },
  "plugins": {
    "directories": ["` + pluginDir + `"]
  }
}`

	configFile := filepath.Join(tempDir, "config.json")
	err = os.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)

	err = cm.LoadFromFile(configFile)
	assert.NoError(t, err)

	config := cm.GetConfig()
	assert.Equal(t, "192.168.1.1", config.Server.Host)
	assert.Equal(t, 8081, config.Server.Port)
	assert.Equal(t, "warn", config.Logging.Level)
	assert.Equal(t, "text", config.Logging.Format)
}

func TestConfigManager_LoadFromFile_UnsupportedFormat(t *testing.T) {
	cm := NewConfigManager()

	configFile := filepath.Join(t.TempDir(), "config.txt")
	err := os.WriteFile(configFile, []byte("some content"), 0644)
	require.NoError(t, err)

	err = cm.LoadFromFile(configFile)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported config file format")
}

func TestConfigManager_LoadFromFile_InvalidYAML(t *testing.T) {
	cm := NewConfigManager()

	configFile := filepath.Join(t.TempDir(), "invalid.yaml")
	err := os.WriteFile(configFile, []byte("invalid: yaml: content:"), 0644)
	require.NoError(t, err)

	err = cm.LoadFromFile(configFile)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse YAML config")
}

func TestConfigManager_LoadFromFile_InvalidJSON(t *testing.T) {
	cm := NewConfigManager()

	configFile := filepath.Join(t.TempDir(), "invalid.json")
	err := os.WriteFile(configFile, []byte("{ invalid json }"), 0644)
	require.NoError(t, err)

	err = cm.LoadFromFile(configFile)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse JSON config")
}

func TestConfigManager_LoadFromFile_NonExistentFile(t *testing.T) {
	cm := NewConfigManager()

	err := cm.LoadFromFile("/non/existent/file.yaml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read config file")
}

func TestConfigManager_LoadFromFile_InvalidPort(t *testing.T) {
	tempDir := t.TempDir()
	pluginDir := filepath.Join(tempDir, "pipelines")
	err := os.MkdirAll(pluginDir, 0755)
	require.NoError(t, err)

	cm := NewConfigManager()

	configContent := `
server:
  port: 70000
plugins:
  directories:
    - "` + pluginDir + `"
`

	configFile := filepath.Join(tempDir, "config.yaml")
	err = os.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)

	err = cm.LoadFromFile(configFile)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid server port")
}

func TestConfigManager_LoadFromFile_InvalidLogLevel(t *testing.T) {
	tempDir := t.TempDir()
	pluginDir := filepath.Join(tempDir, "pipelines")
	err := os.MkdirAll(pluginDir, 0755)
	require.NoError(t, err)

	cm := NewConfigManager()

	configContent := `
logging:
  level: "invalid"
plugins:
  directories:
    - "` + pluginDir + `"
`

	configFile := filepath.Join(tempDir, "config.yaml")
	err = os.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)

	err = cm.LoadFromFile(configFile)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid log level")
}

func TestConfigManager_LoadFromFile_NonExistentPluginDir(t *testing.T) {
	cm := NewConfigManager()

	configContent := `
plugins:
  directories:
    - "/non/existent/directory"
`

	configFile := filepath.Join(t.TempDir(), "config.yaml")
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)

	err = cm.LoadFromFile(configFile)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "plugin directory does not exist")
}

func TestConfigManager_LoadFromEnvironment(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		expected func(*Config)
		setup    func()
		teardown func()
	}{
		{
			name: "valid environment variables",
			envVars: map[string]string{
				"MIMIR_HOST":       "envhost",
				"MIMIR_PORT":       "12345",
				"MIMIR_LOG_LEVEL":  "debug",
				"MIMIR_JWT_SECRET": "envsecret",
			},
			expected: func(c *Config) {
				assert.Equal(t, "envhost", c.Server.Host)
				assert.Equal(t, 12345, c.Server.Port)
				assert.Equal(t, "debug", c.Logging.Level)
				assert.Equal(t, "envsecret", c.Security.JWTSecret)
			},
		},
		{
			name: "invalid port environment variable",
			envVars: map[string]string{
				"MIMIR_PORT": "invalid",
			},
			expected: func(c *Config) {
				assert.Equal(t, 8080, c.Server.Port) // Should remain default
			},
		},
		{
			name: "empty environment variables",
			envVars: map[string]string{
				"MIMIR_HOST": "",
				"MIMIR_PORT": "",
			},
			expected: func(c *Config) {
				assert.Equal(t, "0.0.0.0", c.Server.Host)
				assert.Equal(t, 8080, c.Server.Port)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}
			// Cleanup environment variables
			defer func() {
				for key := range tt.envVars {
					os.Unsetenv(key)
				}
			}()

			cm := NewConfigManager()
			err := cm.LoadFromEnvironment()
			assert.NoError(t, err)

			config := cm.GetConfig()
			tt.expected(config)
		})
	}
}

func TestConfigManager_SaveToFile_YAML(t *testing.T) {
	tempDir := t.TempDir()
	pluginDir := filepath.Join(tempDir, "pipelines")
	err := os.MkdirAll(pluginDir, 0755)
	require.NoError(t, err)

	cm := NewConfigManager()

	// Create a config with valid plugin directory and save directly
	config := &Config{
		Server: ServerConfig{
			Host: "192.168.1.100",
			Port: 9090,
		},
		Plugins: PluginsConfig{
			Directories: []string{pluginDir},
		},
		Logging: LoggingConfig{
			Level: "debug",
		},
	}

	// Set the config directly (bypassing validation for this test)
	cm.config = config

	configFile := filepath.Join(tempDir, "saved_config.yaml")
	err = cm.SaveToFile(configFile)
	assert.NoError(t, err)

	// Verify file was created and contains valid YAML
	data, err := os.ReadFile(configFile)
	require.NoError(t, err)

	var savedConfig Config
	err = yaml.Unmarshal(data, &savedConfig)
	require.NoError(t, err)

	assert.Equal(t, "192.168.1.100", savedConfig.Server.Host)
	assert.Equal(t, 9090, savedConfig.Server.Port)
	assert.Equal(t, "debug", savedConfig.Logging.Level)
}

func TestConfigManager_SaveToFile_JSON(t *testing.T) {
	tempDir := t.TempDir()
	pluginDir := filepath.Join(tempDir, "pipelines")
	err := os.MkdirAll(pluginDir, 0755)
	require.NoError(t, err)

	cm := NewConfigManager()

	// Create a config with valid plugin directory and save directly
	config := &Config{
		Server: ServerConfig{
			Host: "192.168.1.200",
			Port: 7070,
		},
		Plugins: PluginsConfig{
			Directories: []string{pluginDir},
		},
		Logging: LoggingConfig{
			Level: "warn",
		},
	}

	// Set the config directly (bypassing validation for this test)
	cm.config = config

	configFile := filepath.Join(tempDir, "saved_config.json")
	err = cm.SaveToFile(configFile)
	assert.NoError(t, err)

	// Verify file was created and contains valid JSON
	data, err := os.ReadFile(configFile)
	require.NoError(t, err)

	var savedConfig Config
	err = json.Unmarshal(data, &savedConfig)
	require.NoError(t, err)

	assert.Equal(t, "192.168.1.200", savedConfig.Server.Host)
	assert.Equal(t, 7070, savedConfig.Server.Port)
	assert.Equal(t, "warn", savedConfig.Logging.Level)
}

func TestConfigManager_SaveToFile_UnsupportedFormat(t *testing.T) {
	cm := NewConfigManager()

	configFile := filepath.Join(t.TempDir(), "config.txt")
	err := cm.SaveToFile(configFile)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported config file format")
}

func TestConfigManager_GetConfig(t *testing.T) {
	cm := NewConfigManager()

	config1 := cm.GetConfig()
	config2 := cm.GetConfig()

	// Should return copies, not the same object
	assert.NotSame(t, config1, config2)
	// But values should be equal
	assert.Equal(t, config1, config2)

	// Modifying returned config should not affect internal state
	config1.Server.Host = "modified"
	config3 := cm.GetConfig()
	assert.NotEqual(t, "modified", config3.Server.Host)
}

func TestConfigManager_UpdateConfig(t *testing.T) {
	tempDir := t.TempDir()
	pluginDir := filepath.Join(tempDir, "pipelines")
	err := os.MkdirAll(pluginDir, 0755)
	require.NoError(t, err)

	cm := NewConfigManager()

	// Set up initial config with valid plugin directory
	initialConfig := cm.GetConfig()
	initialConfig.Plugins.Directories = []string{pluginDir}
	cm.config = initialConfig // Set directly to bypass validation

	// Test updating fields that are actually merged by mergeConfigs
	updates := &Config{
		Server: ServerConfig{
			Host: "updated-host",
			Port: 9999,
		},
		Logging: LoggingConfig{
			Level: "debug",
		},
	}

	err = cm.UpdateConfig(updates)
	assert.NoError(t, err)

	config := cm.GetConfig()
	assert.Equal(t, "updated-host", config.Server.Host)
	assert.Equal(t, 9999, config.Server.Port)
	assert.Equal(t, "debug", config.Logging.Level)
}

func TestConfigManager_UpdateConfig_Invalid(t *testing.T) {
	cm := NewConfigManager()

	updates := &Config{
		Server: ServerConfig{
			Port: 70000, // Invalid port
		},
	}

	err := cm.UpdateConfig(updates)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid server port")
}

func TestConfigManager_Watchers(t *testing.T) {
	tempDir := t.TempDir()
	pluginDir := filepath.Join(tempDir, "pipelines")
	err := os.MkdirAll(pluginDir, 0755)
	require.NoError(t, err)

	cm := NewConfigManager()

	// Set up initial config with valid plugin directory
	initialConfig := cm.GetConfig()
	initialConfig.Plugins.Directories = []string{pluginDir}
	cm.config = initialConfig // Set directly to bypass validation

	// Create a mock watcher
	watcher := &mockConfigWatcher{
		calls: make([]configChangeCall, 0),
	}

	cm.AddWatcher(watcher)
	assert.Contains(t, cm.watchers, watcher)

	// Update config to trigger watcher (using fields that are actually merged)
	updates := &Config{
		Server: ServerConfig{
			Host: "watcher-test",
		},
		Logging: LoggingConfig{
			Level: "debug",
		},
	}

	err = cm.UpdateConfig(updates)
	require.NoError(t, err)

	// Check watcher was called
	assert.Len(t, watcher.calls, 1)
	assert.Equal(t, "watcher-test", watcher.calls[0].newConfig.Server.Host)

	// Remove watcher
	cm.RemoveWatcher(watcher)
	assert.NotContains(t, cm.watchers, watcher)
}

func TestConfigManager_ConcurrentAccess(t *testing.T) {
	cm := NewConfigManager()

	var wg sync.WaitGroup
	numGoroutines := 10

	// Test concurrent reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_ = cm.GetConfig()
			}
		}()
	}

	// Test concurrent writes
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			tempDir := t.TempDir()
			pluginDir := filepath.Join(tempDir, "pipelines")
			_ = os.MkdirAll(pluginDir, 0755)

			updates := &Config{
				Server: ServerConfig{
					Host: "concurrent-test",
					Port: 8000 + i,
				},
				Plugins: PluginsConfig{
					Directories: []string{pluginDir},
				},
			}
			_ = cm.UpdateConfig(updates)
		}(i)
	}

	wg.Wait()

	// Should not panic and config should be consistent
	config := cm.GetConfig()
	assert.NotNil(t, config)
}

func TestGetConfigManager_Singleton(t *testing.T) {
	cm1 := GetConfigManager()
	cm2 := GetConfigManager()

	assert.Same(t, cm1, cm2)
}

func TestLoadGlobalConfig(t *testing.T) {
	// Create a temporary config file
	tempDir := t.TempDir()
	pluginDir := filepath.Join(tempDir, "pipelines")
	err := os.MkdirAll(pluginDir, 0755)
	require.NoError(t, err)

	configContent := `
server:
  host: "global-test"
  port: 8888
plugins:
  directories:
    - "` + pluginDir + `"
`

	configFile := filepath.Join(tempDir, "config.yaml")
	err = os.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)

	// Change working directory to temp dir
	originalWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(originalWd) }()

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	err = LoadGlobalConfig()
	assert.NoError(t, err)

	cm := GetConfigManager()
	config := cm.GetConfig()
	assert.Equal(t, "global-test", config.Server.Host)
	assert.Equal(t, 8888, config.Server.Port)
}

func TestContains(t *testing.T) {
	tests := []struct {
		name  string
		slice []string
		item  string
		want  bool
	}{
		{
			name:  "item exists",
			slice: []string{"a", "b", "c"},
			item:  "b",
			want:  true,
		},
		{
			name:  "item does not exist",
			slice: []string{"a", "b", "c"},
			item:  "d",
			want:  false,
		},
		{
			name:  "empty slice",
			slice: []string{},
			item:  "a",
			want:  false,
		},
		{
			name:  "nil slice",
			slice: nil,
			item:  "a",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := contains(tt.slice, tt.item)
			assert.Equal(t, tt.want, got)
		})
	}
}

// Mock implementations for testing

type mockConfigWatcher struct {
	calls []configChangeCall
}

type configChangeCall struct {
	oldConfig *Config
	newConfig *Config
}

func (m *mockConfigWatcher) OnConfigChange(oldConfig, newConfig *Config) {
	m.calls = append(m.calls, configChangeCall{
		oldConfig: oldConfig,
		newConfig: newConfig,
	})
}
