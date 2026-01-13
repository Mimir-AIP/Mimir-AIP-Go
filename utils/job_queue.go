package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

// JobQueue manages job distribution via Redis
type JobQueue struct {
	redis      *redis.Client
	queueName  string
	resultName string
}

// JobRequest represents a job request
type JobRequest struct {
	Type         string         `json:"type"` // "pipeline", "digital_twin"
	PipelineFile string         `json:"pipeline_file,omitempty"`
	PipelineYAML string         `json:"pipeline_yaml,omitempty"`
	Context      map[string]any `json:"context,omitempty"`
}

// JobInfo represents queued job information
type JobInfo struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Status    string    `json:"status"` // "queued", "processing", "completed", "failed"
	CreatedAt time.Time `json:"created_at"`
}

// NewJobQueue creates a new job queue
func NewJobQueue(redisURL string) (*JobQueue, error) {
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

	return &JobQueue{
		redis:      client,
		queueName:  "mimir:jobs",
		resultName: "mimir:results",
	}, nil
}

// EnqueueJob enqueues a job for processing
func (q *JobQueue) EnqueueJob(ctx context.Context, req *JobRequest) (*JobInfo, error) {
	// Generate job ID
	jobID := uuid.New().String()

	// Create job message
	jobMsg := map[string]any{
		"id":           jobID,
		"type":         req.Type,
		"pipeline_file": req.PipelineFile,
		"pipeline_yaml": req.PipelineYAML,
		"context":      req.Context,
		"created_at":   time.Now().Format(time.RFC3339),
	}

	// Marshal to JSON
	jobData, err := json.Marshal(jobMsg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal job: %w", err)
	}

	// Push to queue
	if err := q.redis.RPush(ctx, q.queueName, jobData).Err(); err != nil {
		return nil, fmt.Errorf("failed to enqueue job: %w", err)
	}

	return &JobInfo{
		ID:        jobID,
		Type:      req.Type,
		Status:    "queued",
		CreatedAt: time.Now(),
	}, nil
}

// GetJobResult gets the result of a job
func (q *JobQueue) GetJobResult(ctx context.Context, jobID string) (map[string]any, error) {
	key := fmt.Sprintf("%s:%s", q.resultName, jobID)
	resultData, err := q.redis.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("job result not found")
		}
		return nil, fmt.Errorf("failed to get job result: %w", err)
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(resultData), &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}

	return result, nil
}

// WaitForJobResult waits for a job result with timeout
func (q *JobQueue) WaitForJobResult(ctx context.Context, jobID string, timeout time.Duration) (map[string]any, error) {
	// Subscribe to job notification
	notificationKey := fmt.Sprintf("mimir:notifications:job:%s", jobID)
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
			return q.GetJobResult(ctx, jobID)
		case <-ticker.C:
			// Check if result exists
			if result, err := q.GetJobResult(ctx, jobID); err == nil {
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
func (q *JobQueue) GetQueueLength(ctx context.Context) (int64, error) {
	return q.redis.LLen(ctx, q.queueName).Result()
}

// Close closes the job queue
func (q *JobQueue) Close() error {
	if q.redis != nil {
		return q.redis.Close()
	}
	return nil
}
