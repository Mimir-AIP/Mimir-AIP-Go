package tests

import (
	"github.com/Mimir-AIP/Mimir-AIP-Go/utils"
	"os"
	"testing"
)

func setupPluginDir() {
	_ = os.Mkdir("./pipelines", 0755)
}
func teardownPluginDir() {
	os.Remove("./pipelines")
}

func TestConfigManager_LoadFromFile_Valid(t *testing.T) {
	setupPluginDir()
	defer teardownPluginDir()
	cm := utils.NewConfigManager()
	configYaml := `server:
  host: "127.0.0.1"
  port: 9000
logging:
  level: "debug"
  format: "json"
  output: "stdout"
`
	_ = os.WriteFile("test_config.yaml", []byte(configYaml), 0644)
	defer os.Remove("test_config.yaml")
	if err := cm.LoadFromFile("test_config.yaml"); err != nil {
		t.Fatalf("Failed to load valid config: %v", err)
	}
	cfg := cm.GetConfig()
	if cfg.Server.Host != "127.0.0.1" || cfg.Server.Port != 9000 {
		t.Errorf("Config values not loaded correctly: %+v", cfg.Server)
	}
	if cfg.Logging.Level != "debug" || cfg.Logging.Format != "json" {
		t.Errorf("Logging config not loaded correctly: %+v", cfg.Logging)
	}
}

func TestConfigManager_ReloadConfig(t *testing.T) {
	setupPluginDir()
	defer teardownPluginDir()
	cm := utils.NewConfigManager()
	configYaml := `server:
  host: "127.0.0.1"
  port: 9000
`
	_ = os.WriteFile("test_config.yaml", []byte(configYaml), 0644)
	defer os.Remove("test_config.yaml")
	_ = cm.LoadFromFile("test_config.yaml")
	configYaml2 := `server:
  host: "192.168.1.1"
  port: 8000
`
	_ = os.WriteFile("test_config.yaml", []byte(configYaml2), 0644)
	if err := cm.LoadFromFile("test_config.yaml"); err != nil {
		t.Fatalf("Failed to reload config: %v", err)
	}
	cfg := cm.GetConfig()
	if cfg.Server.Host != "192.168.1.1" || cfg.Server.Port != 8000 {
		t.Errorf("Config reload did not update values: %+v", cfg.Server)
	}
}

func TestConfigManager_LoadFromEnvironment(t *testing.T) {
	setupPluginDir()
	defer teardownPluginDir()
	cm := utils.NewConfigManager()
	os.Setenv("MIMIR_HOST", "envhost")
	os.Setenv("MIMIR_PORT", "12345")
	os.Setenv("MIMIR_LOG_LEVEL", "warn")
	os.Setenv("MIMIR_JWT_SECRET", "envsecret")
	defer os.Unsetenv("MIMIR_HOST")
	defer os.Unsetenv("MIMIR_PORT")
	defer os.Unsetenv("MIMIR_LOG_LEVEL")
	defer os.Unsetenv("MIMIR_JWT_SECRET")
	_ = cm.LoadFromEnvironment()
	cfg := cm.GetConfig()
	if cfg.Server.Host != "envhost" || cfg.Server.Port != 12345 {
		t.Errorf("Env override failed: %+v", cfg.Server)
	}
	if cfg.Logging.Level != "warn" {
		t.Errorf("Env log level override failed: %+v", cfg.Logging)
	}
	if cfg.Security.JWTSecret != "envsecret" {
		t.Errorf("Env JWT secret override failed: %+v", cfg.Security)
	}
}

func TestConfigManager_LoadFromFile_Invalid(t *testing.T) {
	setupPluginDir()
	defer teardownPluginDir()
	cm := utils.NewConfigManager()
	_ = os.WriteFile("invalid_config.yaml", []byte("not: valid: yaml: :"), 0644)
	defer os.Remove("invalid_config.yaml")
	if err := cm.LoadFromFile("invalid_config.yaml"); err == nil {
		t.Error("Expected error for invalid config file, got nil")
	}
}
