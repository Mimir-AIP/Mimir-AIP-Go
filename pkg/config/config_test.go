package config

import (
	"os"
	"testing"
)

// TestLoadConfig tests configuration loading
func TestLoadConfig(t *testing.T) {
	// Set test environment variables
	os.Setenv("ENVIRONMENT", "test")
	os.Setenv("LOG_LEVEL", "debug")
	os.Setenv("PORT", "9090")
	os.Setenv("REDIS_URL", "redis://test:6379")
	os.Setenv("MIN_WORKERS", "2")
	os.Setenv("MAX_WORKERS", "100")
	os.Setenv("QUEUE_THRESHOLD", "10")

	defer func() {
		os.Unsetenv("ENVIRONMENT")
		os.Unsetenv("LOG_LEVEL")
		os.Unsetenv("PORT")
		os.Unsetenv("REDIS_URL")
		os.Unsetenv("MIN_WORKERS")
		os.Unsetenv("MAX_WORKERS")
		os.Unsetenv("QUEUE_THRESHOLD")
	}()

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.Environment != "test" {
		t.Errorf("Expected environment 'test', got '%s'", cfg.Environment)
	}

	if cfg.LogLevel != "debug" {
		t.Errorf("Expected log level 'debug', got '%s'", cfg.LogLevel)
	}

	if cfg.Port != "9090" {
		t.Errorf("Expected port '9090', got '%s'", cfg.Port)
	}

	if cfg.RedisURL != "redis://test:6379" {
		t.Errorf("Expected Redis URL 'redis://test:6379', got '%s'", cfg.RedisURL)
	}

	if cfg.MinWorkers != 2 {
		t.Errorf("Expected MinWorkers 2, got %d", cfg.MinWorkers)
	}

	if cfg.MaxWorkers != 100 {
		t.Errorf("Expected MaxWorkers 100, got %d", cfg.MaxWorkers)
	}

	if cfg.QueueThreshold != 10 {
		t.Errorf("Expected QueueThreshold 10, got %d", cfg.QueueThreshold)
	}
}

// TestLoadConfigDefaults tests default values
func TestLoadConfigDefaults(t *testing.T) {
	// Only set required config
	os.Setenv("REDIS_URL", "redis://localhost:6379")
	defer os.Unsetenv("REDIS_URL")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.Environment != "development" {
		t.Errorf("Expected default environment 'development', got '%s'", cfg.Environment)
	}

	if cfg.LogLevel != "info" {
		t.Errorf("Expected default log level 'info', got '%s'", cfg.LogLevel)
	}

	if cfg.Port != "8080" {
		t.Errorf("Expected default port '8080', got '%s'", cfg.Port)
	}

	if cfg.MinWorkers != 1 {
		t.Errorf("Expected default MinWorkers 1, got %d", cfg.MinWorkers)
	}

	if cfg.MaxWorkers != 50 {
		t.Errorf("Expected default MaxWorkers 50, got %d", cfg.MaxWorkers)
	}
}

// TestLoadConfigMissingRequired tests missing required configuration
func TestLoadConfigMissingRequired(t *testing.T) {
	// Clear REDIS_URL to test error case
	os.Unsetenv("REDIS_URL")

	_, err := LoadConfig()
	if err == nil {
		t.Error("Expected error for missing REDIS_URL, got nil")
	}
}
