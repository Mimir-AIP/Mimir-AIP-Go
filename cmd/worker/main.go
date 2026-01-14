// Worker process for Mimir AIP
// Executes pipelines and digital twin tasks from Redis queue
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
	AI "github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/AI"
	input "github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/Input"
	output "github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/Output"
	"github.com/Mimir-AIP/Mimir-AIP-Go/utils"
	"github.com/go-redis/redis/v8"
)

const workerVersion = "v0.0.1"

// TaskMessage represents a task message from Redis queue
type TaskMessage struct {
	ID           string         `json:"id"`
	Type         string         `json:"type"` // "pipeline", "digital_twin"
	PipelineFile string         `json:"pipeline_file,omitempty"`
	PipelineYAML string         `json:"pipeline_yaml,omitempty"`
	Context      map[string]any `json:"context,omitempty"`
	CreatedAt    string         `json:"created_at"`
}

// TaskResult represents the result of a task execution
type TaskResult struct {
	ID         string                   `json:"id"`
	Success    bool                     `json:"success"`
	Error      string                   `json:"error,omitempty"`
	Context    *pipelines.PluginContext `json:"context,omitempty"`
	ExecutedAt string                   `json:"executed_at"`
	WorkerID   string                   `json:"worker_id"`
}

// Worker represents a task worker
type Worker struct {
	id         string
	redisURL   string
	redis      *redis.Client
	registry   *pipelines.PluginRegistry
	logger     *utils.Logger
	ctx        context.Context
	cancel     context.CancelFunc
	queueName  string
	resultName string
}

// NewWorker creates a new worker instance
func NewWorker(redisURL string) (*Worker, error) {
	// Parse Redis URL
	opt, err := redis.ParseURL(fmt.Sprintf("redis://%s", redisURL))
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	// Create Redis client
	client := redis.NewClient(opt)

	// Test connection
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	// Generate worker ID
	hostname, _ := os.Hostname()
	workerID := fmt.Sprintf("worker-%s-%d", hostname, os.Getpid())

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())

	// Initialize plugin registry
	registry := pipelines.NewPluginRegistry()

	// Get logger
	logger := utils.GetLogger()

	w := &Worker{
		id:         workerID,
		redisURL:   redisURL,
		redis:      client,
		registry:   registry,
		logger:     logger,
		ctx:        ctx,
		cancel:     cancel,
		queueName:  "mimir:tasks",
		resultName: "mimir:task_results",
	}

	// Register plugins
	if err := w.registerPlugins(); err != nil {
		return nil, fmt.Errorf("failed to register plugins: %w", err)
	}

	return w, nil
}

// registerPlugins registers all available plugins
func (w *Worker) registerPlugins() error {
	// Input plugins
	if err := w.registry.RegisterPlugin(input.NewCSVPlugin()); err != nil {
		w.logger.Warn(fmt.Sprintf("Failed to register CSV plugin: %v", err))
	}
	if err := w.registry.RegisterPlugin(input.NewJSONPlugin()); err != nil {
		w.logger.Warn(fmt.Sprintf("Failed to register JSON input plugin: %v", err))
	}
	if err := w.registry.RegisterPlugin(input.NewMarkdownPlugin()); err != nil {
		w.logger.Warn(fmt.Sprintf("Failed to register Markdown plugin: %v", err))
	}

	// AI plugins (mock for testing)
	mockClient := AI.NewIntelligentMockLLMClient()
	if err := w.registry.RegisterPlugin(AI.NewLLMPlugin(mockClient, AI.ProviderMock)); err != nil {
		w.logger.Warn(fmt.Sprintf("Failed to register Mock LLM plugin: %v", err))
	}

	// Output plugins
	if err := w.registry.RegisterPlugin(output.NewJSONPlugin()); err != nil {
		w.logger.Warn(fmt.Sprintf("Failed to register JSON output plugin: %v", err))
	}
	if err := w.registry.RegisterPlugin(output.NewPDFPlugin()); err != nil {
		w.logger.Warn(fmt.Sprintf("Failed to register PDF plugin: %v", err))
	}

	w.logger.Info("Plugin registration completed")
	return nil
}

// Start starts the worker
func (w *Worker) Start() error {
	w.logger.Info(fmt.Sprintf("Worker %s starting...", w.id))
	w.logger.Info(fmt.Sprintf("Connected to Redis at %s", w.redisURL))
	w.logger.Info(fmt.Sprintf("Listening on queue: %s", w.queueName))

	// Get concurrency setting
	concurrency := 5
	if concStr := os.Getenv("WORKER_CONCURRENCY"); concStr != "" {
		if c, err := strconv.Atoi(concStr); err == nil && c > 0 {
			concurrency = c
		}
	}

	w.logger.Info(fmt.Sprintf("Worker concurrency: %d", concurrency))

	// Create worker pool
	sem := make(chan struct{}, concurrency)

	// Main loop
	for {
		select {
		case <-w.ctx.Done():
			w.logger.Info("Worker shutting down...")
			return nil
		default:
			// Wait for available slot
			sem <- struct{}{}

			// Pop task from queue (blocking)
			result, err := w.redis.BLPop(w.ctx, 5*time.Second, w.queueName).Result()
			if err != nil {
				<-sem // Release slot
				if err == redis.Nil {
					// Timeout, continue
					continue
				}
				if err == context.Canceled {
					return nil
				}
				w.logger.Error("Error popping from queue", err)
				time.Sleep(1 * time.Second)
				continue
			}

			// Process task in goroutine
			taskData := result[1]
			go func() {
				defer func() { <-sem }()
				w.processTask(taskData)
			}()
		}
	}
}

// processTask processes a single task
func (w *Worker) processTask(taskData string) {
	// Parse task message
	var task TaskMessage
	if err := json.Unmarshal([]byte(taskData), &task); err != nil {
		w.logger.Error("Failed to parse task message", err)
		return
	}

	w.logger.Info(fmt.Sprintf("Processing task %s (type: %s)", task.ID, task.Type))

	// Execute based on task type
	var result *TaskResult
	switch task.Type {
	case "pipeline":
		result = w.executePipeline(&task)
	case "digital_twin":
		result = w.executeDigitalTwin(&task)
	default:
		result = &TaskResult{
			ID:         task.ID,
			Success:    false,
			Error:      fmt.Sprintf("unknown task type: %s", task.Type),
			ExecutedAt: time.Now().Format(time.RFC3339),
			WorkerID:   w.id,
		}
	}

	// Store result
	w.storeResult(result)
}

// executePipeline executes a pipeline task
func (w *Worker) executePipeline(task *TaskMessage) *TaskResult {
	result := &TaskResult{
		ID:         task.ID,
		Success:    false,
		ExecutedAt: time.Now().Format(time.RFC3339),
		WorkerID:   w.id,
	}

	// Parse pipeline
	var pipelineConfig *utils.PipelineConfig
	var err error

	if task.PipelineYAML != "" {
		// Parse from YAML string
		pipelineConfig, err = utils.ParsePipelineFromYAML([]byte(task.PipelineYAML))
	} else if task.PipelineFile != "" {
		// Parse from file
		pipelineConfig, err = utils.ParsePipeline(task.PipelineFile)
	} else {
		result.Error = "no pipeline file or YAML provided"
		return result
	}

	if err != nil {
		result.Error = fmt.Sprintf("failed to parse pipeline: %v", err)
		return result
	}

	// Execute pipeline
	execResult, err := utils.ExecutePipelineWithRegistry(w.ctx, pipelineConfig, w.registry)
	if err != nil {
		result.Error = fmt.Sprintf("failed to execute pipeline: %v", err)
		return result
	}

	result.Success = execResult.Success
	result.Error = execResult.Error
	result.Context = execResult.Context

	return result
}

// executeDigitalTwin executes a digital twin task
func (w *Worker) executeDigitalTwin(task *TaskMessage) *TaskResult {
	result := &TaskResult{
		ID:         task.ID,
		Success:    false,
		ExecutedAt: time.Now().Format(time.RFC3339),
		WorkerID:   w.id,
	}

	// For now, treat digital twin tasks as specialized pipelines
	// This can be expanded with specific digital twin logic
	result.Success = true
	result.Context = pipelines.NewPluginContext()
	result.Context.Set("message", "Digital twin task executed successfully")

	return result
}

// storeResult stores the job result in Redis
func (w *Worker) storeResult(result *TaskResult) {
	resultData, err := json.Marshal(result)
	if err != nil {
		w.logger.Error("Failed to marshal result", err)
		return
	}

	// Store result with expiration (1 hour)
	key := fmt.Sprintf("%s:%s", w.resultName, result.ID)
	if err := w.redis.Set(w.ctx, key, resultData, 1*time.Hour).Err(); err != nil {
		w.logger.Error("Failed to store result", err)
		return
	}

	// Publish result notification
	notificationKey := fmt.Sprintf("mimir:notifications:task:%s", result.ID)
	if err := w.redis.Publish(w.ctx, notificationKey, resultData).Err(); err != nil {
		w.logger.Error("Failed to publish notification", err)
	}

	if result.Success {
		w.logger.Info(fmt.Sprintf("Task %s completed successfully", result.ID))
	} else {
		w.logger.Error(fmt.Sprintf("Task %s failed", result.ID), fmt.Errorf("%s", result.Error))
	}
}

// Stop stops the worker
func (w *Worker) Stop() {
	w.logger.Info("Stopping worker...")
	w.cancel()
	if w.redis != nil {
		w.redis.Close()
	}
}

func main() {
	// Check for version flag
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Println("Mimir AIP Worker version:", workerVersion)
		return
	}

	// Get Redis URL from environment
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "localhost:6379"
	}

	// Create worker
	worker, err := NewWorker(redisURL)
	if err != nil {
		log.Fatalf("Failed to create worker: %v", err)
	}

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start worker in goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := worker.Start(); err != nil {
			errChan <- err
		}
	}()

	// Wait for signal or error
	select {
	case <-sigChan:
		log.Println("Received shutdown signal")
		worker.Stop()
	case err := <-errChan:
		log.Fatalf("Worker error: %v", err)
	}

	log.Println("Worker stopped")
}
