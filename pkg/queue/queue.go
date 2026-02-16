package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

const (
	jobQueueKey     = "mimir:jobs:queue"
	jobDataKeyFmt   = "mimir:jobs:data:%s"
	jobStatusKeyFmt = "mimir:jobs:status:%s"
)

// Queue provides Redis-backed job queue operations
type Queue struct {
	client *redis.Client
	ctx    context.Context
}

// NewQueue creates a new queue instance
func NewQueue(redisURL string) (*Queue, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse redis URL: %w", err)
	}

	client := redis.NewClient(opt)

	// Test connection
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return &Queue{
		client: client,
		ctx:    ctx,
	}, nil
}

// Enqueue adds a job to the queue
func (q *Queue) Enqueue(job *models.Job) error {
	// Serialize job data
	jobData, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job: %w", err)
	}

	// Store job data
	jobDataKey := fmt.Sprintf(jobDataKeyFmt, job.ID)
	if err := q.client.Set(q.ctx, jobDataKey, jobData, 24*time.Hour).Err(); err != nil {
		return fmt.Errorf("failed to store job data: %w", err)
	}

	// Add to queue with priority (lower score = higher priority)
	score := float64(time.Now().Unix()) / float64(job.Priority+1)
	if err := q.client.ZAdd(q.ctx, jobQueueKey, &redis.Z{
		Score:  score,
		Member: job.ID,
	}).Err(); err != nil {
		return fmt.Errorf("failed to enqueue job: %w", err)
	}

	return nil
}

// Dequeue retrieves the next job from the queue
func (q *Queue) Dequeue() (*models.Job, error) {
	// Get the job with the lowest score (highest priority, oldest first)
	result, err := q.client.ZPopMin(q.ctx, jobQueueKey, 1).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to dequeue job: %w", err)
	}

	if len(result) == 0 {
		return nil, nil // No jobs available
	}

	jobID := result[0].Member.(string)

	// Retrieve job data
	jobDataKey := fmt.Sprintf(jobDataKeyFmt, jobID)
	jobData, err := q.client.Get(q.ctx, jobDataKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve job data: %w", err)
	}

	var job models.Job
	if err := json.Unmarshal([]byte(jobData), &job); err != nil {
		return nil, fmt.Errorf("failed to unmarshal job: %w", err)
	}

	return &job, nil
}

// GetJob retrieves a job by ID
func (q *Queue) GetJob(jobID string) (*models.Job, error) {
	jobDataKey := fmt.Sprintf(jobDataKeyFmt, jobID)
	jobData, err := q.client.Get(q.ctx, jobDataKey).Result()
	if err == redis.Nil {
		return nil, fmt.Errorf("job not found: %s", jobID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve job: %w", err)
	}

	var job models.Job
	if err := json.Unmarshal([]byte(jobData), &job); err != nil {
		return nil, fmt.Errorf("failed to unmarshal job: %w", err)
	}

	return &job, nil
}

// UpdateJobStatus updates the status of a job
func (q *Queue) UpdateJobStatus(jobID string, status models.JobStatus, errorMsg string) error {
	job, err := q.GetJob(jobID)
	if err != nil {
		return err
	}

	job.Status = status
	if errorMsg != "" {
		job.ErrorMessage = errorMsg
	}

	now := time.Now()
	switch status {
	case models.JobStatusExecuting:
		job.StartedAt = &now
	case models.JobStatusCompleted, models.JobStatusFailed, models.JobStatusTimeout, models.JobStatusCancelled:
		job.CompletedAt = &now
	}

	// Update job data
	jobData, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job: %w", err)
	}

	jobDataKey := fmt.Sprintf(jobDataKeyFmt, jobID)
	if err := q.client.Set(q.ctx, jobDataKey, jobData, 24*time.Hour).Err(); err != nil {
		return fmt.Errorf("failed to update job data: %w", err)
	}

	return nil
}

// QueueLength returns the current length of the job queue
func (q *Queue) QueueLength() (int64, error) {
	length, err := q.client.ZCard(q.ctx, jobQueueKey).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get queue length: %w", err)
	}
	return length, nil
}

// GetHighPriorityJobs returns jobs with priority above a threshold
func (q *Queue) GetHighPriorityJobs(minPriority int) ([]*models.Job, error) {
	// Get all job IDs from the queue
	jobIDs, err := q.client.ZRange(q.ctx, jobQueueKey, 0, -1).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get job IDs: %w", err)
	}

	var highPriorityJobs []*models.Job
	for _, jobID := range jobIDs {
		job, err := q.GetJob(jobID)
		if err != nil {
			continue
		}
		if job.Priority >= minPriority {
			highPriorityJobs = append(highPriorityJobs, job)
		}
	}

	return highPriorityJobs, nil
}

// Close closes the Redis connection
func (q *Queue) Close() error {
	return q.client.Close()
}
