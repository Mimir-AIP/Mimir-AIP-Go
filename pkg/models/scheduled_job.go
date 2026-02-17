package models

import "time"

// ScheduledJob represents a recurring job that executes pipelines
type ScheduledJob struct {
	ID        string     `json:"id" yaml:"-"`
	ProjectID string     `json:"project_id" yaml:"-"`
	Name      string     `json:"name" yaml:"name"`
	Pipelines []string   `json:"pipelines" yaml:"pipelines"`
	Schedule  string     `json:"schedule" yaml:"schedule"` // Cron expression
	Enabled   bool       `json:"enabled" yaml:"enabled"`
	CreatedAt time.Time  `json:"created_at" yaml:"-"`
	UpdatedAt time.Time  `json:"updated_at" yaml:"-"`
	LastRun   *time.Time `json:"last_run,omitempty" yaml:"-"`
	NextRun   *time.Time `json:"next_run,omitempty" yaml:"-"`
}

// ScheduledJobCreateRequest represents a request to create a new scheduled job
type ScheduledJobCreateRequest struct {
	ProjectID string   `json:"project_id" yaml:"project_id"`
	Name      string   `json:"name" yaml:"name"`
	Pipelines []string `json:"pipelines" yaml:"pipelines"`
	Schedule  string   `json:"schedule" yaml:"schedule"`
	Enabled   bool     `json:"enabled" yaml:"enabled"`
}

// ScheduledJobUpdateRequest represents a request to update a scheduled job
type ScheduledJobUpdateRequest struct {
	Name      *string   `json:"name,omitempty"`
	Pipelines *[]string `json:"pipelines,omitempty"`
	Schedule  *string   `json:"schedule,omitempty"`
	Enabled   *bool     `json:"enabled,omitempty"`
}

// JobExecution represents a single execution of a scheduled job
type JobExecution struct {
	ID           string     `json:"id"`
	JobID        string     `json:"job_id"`
	ProjectID    string     `json:"project_id"`
	StartedAt    time.Time  `json:"started_at"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	Status       string     `json:"status"`
	PipelineRuns []string   `json:"pipeline_runs"` // Pipeline execution IDs
	Error        string     `json:"error,omitempty"`
}
