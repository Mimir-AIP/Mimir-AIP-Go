package config

import (
	"os"
	"strconv"
)

// Config holds the application configuration
type Config struct {
	Environment          string
	LogLevel             string
	Port                 string
	DatabaseURL          string
	OrchestratorURL      string
	JobTimeout           int
	MinWorkers           int
	MaxWorkers           int
	QueueThreshold       int
	StorageAccessToken   string
	WorkerNamespace      string
	WorkerServiceAccount string
	WorkerImage          string
	WorkerCPULimit       string
	WorkerMemoryLimit    string
}

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
	config := &Config{
		Environment:          getEnv("ENVIRONMENT", "development"),
		LogLevel:             getEnv("LOG_LEVEL", "info"),
		Port:                 getEnv("PORT", "8080"),
		DatabaseURL:          getEnv("DATABASE_URL", ""),
		OrchestratorURL:      getEnv("ORCHESTRATOR_URL", "http://localhost:8080"),
		JobTimeout:           getEnvAsInt("JOB_TIMEOUT", 3600),
		MinWorkers:           getEnvAsInt("MIN_WORKERS", 1),
		MaxWorkers:           getEnvAsInt("MAX_WORKERS", 50),
		QueueThreshold:       getEnvAsInt("QUEUE_THRESHOLD", 5),
		StorageAccessToken:   getEnv("STORAGE_ACCESS_TOKEN", ""),
		WorkerNamespace:      getEnv("WORKER_NAMESPACE", "mimir-aip"),
		WorkerServiceAccount: getEnv("WORKER_SERVICE_ACCOUNT", "worker-service-account"),
		WorkerImage:          getEnv("WORKER_IMAGE", "mimir-aip/worker:latest"),
		WorkerCPULimit:       getEnv("WORKER_CPU_LIMIT", "2000m"),
		WorkerMemoryLimit:    getEnv("WORKER_MEMORY_LIMIT", "4Gi"),
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
