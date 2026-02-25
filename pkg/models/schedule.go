package models

import "time"

// Schedule represents a recurring schedule that executes pipelines (cron-based)
type Schedule struct {
	ID           string     `json:"id" yaml:"-"`
	ProjectID    string     `json:"project_id" yaml:"-"`
	Name         string     `json:"name" yaml:"name"`
	Pipelines    []string   `json:"pipelines" yaml:"pipelines"`
	CronSchedule string     `json:"cron_schedule" yaml:"cron_schedule"` // Cron expression
	Enabled      bool       `json:"enabled" yaml:"enabled"`
	CreatedAt    time.Time  `json:"created_at" yaml:"-"`
	UpdatedAt    time.Time  `json:"updated_at" yaml:"-"`
	LastRun      *time.Time `json:"last_run,omitempty" yaml:"-"`
	NextRun      *time.Time `json:"next_run,omitempty" yaml:"-"`
}

// ScheduleCreateRequest represents a request to create a new schedule
type ScheduleCreateRequest struct {
	ProjectID    string   `json:"project_id" yaml:"project_id"`
	Name         string   `json:"name" yaml:"name"`
	Pipelines    []string `json:"pipelines" yaml:"pipelines"`
	CronSchedule string   `json:"cron_schedule" yaml:"cron_schedule"`
	Enabled      bool     `json:"enabled" yaml:"enabled"`
}

// ScheduleUpdateRequest represents a request to update a schedule
type ScheduleUpdateRequest struct {
	Name         *string   `json:"name,omitempty"`
	Pipelines    *[]string `json:"pipelines,omitempty"`
	CronSchedule *string   `json:"cron_schedule,omitempty"`
	Enabled      *bool     `json:"enabled,omitempty"`
}

// ScheduleExecution represents a single execution of a scheduled pipeline run
type ScheduleExecution struct {
	ID           string     `json:"id"`
	ScheduleID   string     `json:"schedule_id"`
	ProjectID    string     `json:"project_id"`
	StartedAt    time.Time  `json:"started_at"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	Status       string     `json:"status"`
	PipelineRuns []string   `json:"pipeline_runs"` // Pipeline execution IDs
	Error        string     `json:"error,omitempty"`
}
