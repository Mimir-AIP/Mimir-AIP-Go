package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

// TaskQueue manages task distribution via Redis
type TaskQueue struct {
	redis      *redis.Client
	queueName  string
	resultName string
}

// TaskRequest represents a task request
type TaskRequest struct {
	Type         string         `json:"type"` // "pipeline", "digital_twin"
	PipelineFile string         `json:"pipeline_file,omitempty"`
	PipelineYAML string         `json:"pipeline_yaml,omitempty"`
	Context      map[string]any `json:"context,omitempty"`
}

// TaskInfo represents queued task information
type TaskInfo struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Status    string    `json:"status"` // "queued", "processing", "completed", "failed"
	CreatedAt time.Time `json:"created_at"`
}

// NewTaskQueue creates a new task queue
func NewTaskQueue(redisURL string) (*TaskQueue, error) {
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

	return &TaskQueue{
		redis:      client,
		queueName:  "mimir:tasks",
		resultName: "mimir:task_results",
	}, nil
}

// EnqueueTask enqueues a task for processing
func (q *TaskQueue) EnqueueTask(ctx context.Context, req *TaskRequest) (*TaskInfo, error) {
	// Generate task ID
	taskID := uuid.New().String()

	// Create task message
	taskMsg := map[string]any{
		"id":            taskID,
		"type":          req.Type,
		"pipeline_file": req.PipelineFile,
		"pipeline_yaml": req.PipelineYAML,
		"context":       req.Context,
		"created_at":    time.Now().Format(time.RFC3339),
	}

	// Marshal to JSON
	taskData, err := json.Marshal(taskMsg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal task: %w", err)
	}

	// Push to queue
	if err := q.redis.RPush(ctx, q.queueName, taskData).Err(); err != nil {
		return nil, fmt.Errorf("failed to enqueue task: %w", err)
	}

	return &TaskInfo{
		ID:        taskID,
		Type:      req.Type,
		Status:    "queued",
		CreatedAt: time.Now(),
	}, nil
}

// GetTaskResult gets the result of a task
func (q *TaskQueue) GetTaskResult(ctx context.Context, taskID string) (map[string]any, error) {
	key := fmt.Sprintf("%s:%s", q.resultName, taskID)
	resultData, err := q.redis.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("task result not found")
		}
		return nil, fmt.Errorf("failed to get task result: %w", err)
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(resultData), &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}

	return result, nil
}

// WaitForTaskResult waits for a task result with timeout
func (q *TaskQueue) WaitForTaskResult(ctx context.Context, taskID string, timeout time.Duration) (map[string]any, error) {
	// Subscribe to task notification
	notificationKey := fmt.Sprintf("mimir:notifications:task:%s", taskID)
	pubsub := q.redis.Subscribe(ctx, notificationKey)
	defer pubsub.Close()

	// Create timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Wait for notification or check periodically
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeoutCtx.Done():
			// Timeout - try to get result one last time
			return q.GetTaskResult(ctx, taskID)
		case <-ticker.C:
			// Check if result exists
			if result, err := q.GetTaskResult(ctx, taskID); err == nil {
				return result, nil
			}
		case msg := <-pubsub.Channel():
			// Received notification
			var result map[string]any
			if err := json.Unmarshal([]byte(msg.Payload), &result); err != nil {
				continue
			}
			return result, nil
		}
	}
}

// GetQueueLength gets the current queue length
func (q *TaskQueue) GetQueueLength(ctx context.Context) (int64, error) {
	return q.redis.LLen(ctx, q.queueName).Result()
}

// Close closes the task queue
func (q *TaskQueue) Close() error {
	if q.redis != nil {
		return q.redis.Close()
	}
	return nil
}
