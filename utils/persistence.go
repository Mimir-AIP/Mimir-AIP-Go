package utils

import (
	"context"
	"encoding/json"
	"time"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
)

// PersistenceBackend defines the interface for persisting jobs and pipelines
type PersistenceBackend interface {
	// Job persistence
	SaveJob(ctx context.Context, job *ScheduledJob) error
	GetJob(ctx context.Context, id string) (*ScheduledJob, error)
	ListJobs(ctx context.Context) ([]*ScheduledJob, error)
	DeleteJob(ctx context.Context, id string) error
	UpdateJob(ctx context.Context, job *ScheduledJob) error

	// Pipeline persistence
	SavePipeline(ctx context.Context, pipeline *PipelineConfig) error
	GetPipeline(ctx context.Context, name string) (*PipelineConfig, error)
	ListPipelines(ctx context.Context) ([]*PipelineConfig, error)
	DeletePipeline(ctx context.Context, name string) error
	UpdatePipeline(ctx context.Context, pipeline *PipelineConfig) error

	// Job execution history
	SaveExecution(ctx context.Context, execution *JobExecutionRecord) error
	GetExecution(ctx context.Context, id string) (*JobExecutionRecord, error)
	ListExecutions(ctx context.Context, jobID string, limit int) ([]*JobExecutionRecord, error)
	DeleteOldExecutions(ctx context.Context, olderThan time.Time) error

	// Health and lifecycle
	Health(ctx context.Context) error
	Close() error
}

// SerializableScheduledJob is a version of ScheduledJob that can be serialized
type SerializableScheduledJob struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Pipeline   string    `json:"pipeline"`
	CronExpr   string    `json:"cron_expr"`
	Enabled    bool      `json:"enabled"`
	NextRun    time.Time `json:"next_run"`
	LastRun    time.Time `json:"last_run"`
	LastResult string    `json:"last_result"` // JSON serialized
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// ToSerializable converts ScheduledJob to SerializableScheduledJob
func (sj *ScheduledJob) ToSerializable() (*SerializableScheduledJob, error) {
	ssj := &SerializableScheduledJob{
		ID:        sj.ID,
		Name:      sj.Name,
		Pipeline:  sj.Pipeline,
		CronExpr:  sj.CronExpr,
		Enabled:   sj.Enabled,
		CreatedAt: sj.CreatedAt,
		UpdatedAt: sj.UpdatedAt,
	}

	if sj.NextRun != nil {
		ssj.NextRun = *sj.NextRun
	}

	if sj.LastRun != nil {
		ssj.LastRun = *sj.LastRun
	}

	if sj.LastResult != nil {
		data, err := json.Marshal(sj.LastResult)
		if err != nil {
			return nil, err
		}
		ssj.LastResult = string(data)
	}

	return ssj, nil
}

// FromSerializable converts SerializableScheduledJob to ScheduledJob
func (ssj *SerializableScheduledJob) FromSerializable() (*ScheduledJob, error) {
	sj := &ScheduledJob{
		ID:        ssj.ID,
		Name:      ssj.Name,
		Pipeline:  ssj.Pipeline,
		CronExpr:  ssj.CronExpr,
		Enabled:   ssj.Enabled,
		CreatedAt: ssj.CreatedAt,
		UpdatedAt: ssj.UpdatedAt,
	}

	if !ssj.NextRun.IsZero() {
		sj.NextRun = &ssj.NextRun
	}

	if !ssj.LastRun.IsZero() {
		sj.LastRun = &ssj.LastRun
	}

	if ssj.LastResult != "" {
		var result PipelineExecutionResult
		if err := json.Unmarshal([]byte(ssj.LastResult), &result); err != nil {
			return nil, err
		}
		sj.LastResult = &result
	}

	return sj, nil
}

// SerializableJobExecution is a version of JobExecutionRecord that can be serialized
type SerializableJobExecution struct {
	ID          string    `json:"id"`
	JobID       string    `json:"job_id"`
	Pipeline    string    `json:"pipeline"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	Duration    int64     `json:"duration"` // nanoseconds
	Status      string    `json:"status"`
	Error       string    `json:"error"`
	Context     string    `json:"context"`     // JSON serialized
	Steps       string    `json:"steps"`       // JSON serialized
	TriggeredBy string    `json:"triggered_by"`
}

// ToSerializable converts JobExecutionRecord to SerializableJobExecution
func (jer *JobExecutionRecord) ToSerializable() (*SerializableJobExecution, error) {
	sje := &SerializableJobExecution{
		ID:          jer.ID,
		JobID:       jer.JobID,
		Pipeline:    jer.Pipeline,
		StartTime:   jer.StartTime,
		Status:      jer.Status,
		Error:       jer.Error,
		TriggeredBy: jer.TriggeredBy,
	}

	if jer.EndTime != nil {
		sje.EndTime = *jer.EndTime
	}

	if jer.Duration != nil {
		sje.Duration = int64(*jer.Duration)
	}

	if jer.Context != nil {
		data, err := json.Marshal(jer.Context)
		if err != nil {
			return nil, err
		}
		sje.Context = string(data)
	}

	if jer.Steps != nil {
		data, err := json.Marshal(jer.Steps)
		if err != nil {
			return nil, err
		}
		sje.Steps = string(data)
	}

	return sje, nil
}

// FromSerializable converts SerializableJobExecution to JobExecutionRecord
func (sje *SerializableJobExecution) FromSerializable() (*JobExecutionRecord, error) {
	jer := &JobExecutionRecord{
		ID:          sje.ID,
		JobID:       sje.JobID,
		Pipeline:    sje.Pipeline,
		StartTime:   sje.StartTime,
		Status:      sje.Status,
		Error:       sje.Error,
		TriggeredBy: sje.TriggeredBy,
	}

	if !sje.EndTime.IsZero() {
		jer.EndTime = &sje.EndTime
	}

	if sje.Duration > 0 {
		duration := time.Duration(sje.Duration)
		jer.Duration = &duration
	}

	if sje.Context != "" {
		var context pipelines.PluginContext
		if err := json.Unmarshal([]byte(sje.Context), &context); err != nil {
			return nil, err
		}
		jer.Context = &context
	}

	if sje.Steps != "" {
		var steps []StepExecutionRecord
		if err := json.Unmarshal([]byte(sje.Steps), &steps); err != nil {
			return nil, err
		}
		jer.Steps = steps
	}

	return jer, nil
}
