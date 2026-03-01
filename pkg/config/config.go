package config

import (
	"encoding/json"
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
	// Multi-cluster dispatch
	ClusterConfigFile string // path to YAML cluster config; empty = single in-cluster behaviour
	WorkerAuthToken   string // shared Bearer token for /api/worktasks/* paths; empty = disabled
	// Per-type concurrency limits (map of task type → max simultaneous workers)
	ConcurrencyLimits map[string]int
	// LLM provider configuration (all optional — LLM features degrade gracefully when unset)
	LLMEnabled  bool
	LLMProvider string // "openrouter" | "openai_compat" | any registered provider name
	LLMAPIKey   string
	LLMBaseURL  string // required for openai_compat
	LLMModel    string // defaults per-provider when empty
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
		ClusterConfigFile:    getEnv("CLUSTER_CONFIG_FILE", ""),
		WorkerAuthToken:      getEnv("WORKER_AUTH_TOKEN", ""),
		ConcurrencyLimits:    getEnvAsConcurrencyLimits("WORKER_CONCURRENCY_LIMITS"),
		LLMEnabled:           getEnvAsBool("LLM_ENABLED", false),
		LLMProvider:          getEnv("LLM_PROVIDER", ""),
		LLMAPIKey:            getEnv("LLM_API_KEY", ""),
		LLMBaseURL:           getEnv("LLM_BASE_URL", ""),
		LLMModel:             getEnv("LLM_MODEL", ""),
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

// getEnvAsConcurrencyLimits parses a JSON map from an env var, e.g.
// '{"ml_training":5,"pipeline_execution":20}'. Falls back to safe defaults.
func getEnvAsConcurrencyLimits(key string) map[string]int {
	defaults := map[string]int{
		"ml_training":        5,
		"ml_inference":       10,
		"pipeline_execution": 20,
		"digital_twin_update": 10,
	}
	raw := os.Getenv(key)
	if raw == "" {
		return defaults
	}
	var parsed map[string]int
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return defaults
	}
	// Merge: env values override defaults
	for k, v := range parsed {
		defaults[k] = v
	}
	return defaults
}

// getEnvAsBool retrieves an environment variable as a boolean or returns a default value
func getEnvAsBool(key string, defaultValue bool) bool {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.ParseBool(valueStr)
	if err != nil {
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
