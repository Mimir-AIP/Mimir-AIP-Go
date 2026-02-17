package config

import (
	"os"
	"strconv"
)

// Config holds the application configuration
type Config struct {
	Environment        string
	LogLevel           string
	Port               string
	DatabaseURL        string
	OrchestratorURL    string
	JobTimeout         int
	MinWorkers         int
	MaxWorkers         int
	QueueThreshold     int
	StorageAccessToken string
}

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
	config := &Config{
		Environment:        getEnv("ENVIRONMENT", "development"),
		LogLevel:           getEnv("LOG_LEVEL", "info"),
		Port:               getEnv("PORT", "8080"),
		DatabaseURL:        getEnv("DATABASE_URL", ""),
		OrchestratorURL:    getEnv("ORCHESTRATOR_URL", "http://localhost:8080"),
		JobTimeout:         getEnvAsInt("JOB_TIMEOUT", 3600),
		MinWorkers:         getEnvAsInt("MIN_WORKERS", 1),
		MaxWorkers:         getEnvAsInt("MAX_WORKERS", 50),
		QueueThreshold:     getEnvAsInt("QUEUE_THRESHOLD", 5),
		StorageAccessToken: getEnv("STORAGE_ACCESS_TOKEN", ""),
	}

	return config, nil
}

// getEnv retrieves an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// getEnvAsInt retrieves an environment variable as an integer or returns a default value
func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}
