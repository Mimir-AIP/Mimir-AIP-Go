package utils

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// SQLitePersistence implements PersistenceBackend using SQLite
type SQLitePersistence struct {
	db   *sql.DB
	path string
}

// NewSQLitePersistence creates a new SQLite persistence backend
func NewSQLitePersistence(dbPath string) (*SQLitePersistence, error) {
	// Ensure the directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open database connection
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

	sp := &SQLitePersistence{
		db:   db,
		path: dbPath,
	}

	// Initialize schema
	if err := sp.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return sp, nil
}

// initSchema creates the database schema
func (sp *SQLitePersistence) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS scheduled_jobs (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		pipeline TEXT NOT NULL,
		cron_expr TEXT NOT NULL,
		enabled BOOLEAN NOT NULL DEFAULT 1,
		next_run TIMESTAMP,
		last_run TIMESTAMP,
		last_result TEXT,
		created_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_jobs_enabled ON scheduled_jobs(enabled);
	CREATE INDEX IF NOT EXISTS idx_jobs_next_run ON scheduled_jobs(next_run);

	CREATE TABLE IF NOT EXISTS pipelines (
		name TEXT PRIMARY KEY,
		description TEXT,
		config TEXT NOT NULL,
		created_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL
	);

	CREATE TABLE IF NOT EXISTS job_executions (
		id TEXT PRIMARY KEY,
		job_id TEXT NOT NULL,
		pipeline TEXT NOT NULL,
		start_time TIMESTAMP NOT NULL,
		end_time TIMESTAMP,
		duration INTEGER,
		status TEXT NOT NULL,
		error TEXT,
		context TEXT,
		steps TEXT,
		triggered_by TEXT NOT NULL,
		FOREIGN KEY (job_id) REFERENCES scheduled_jobs(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_executions_job_id ON job_executions(job_id);
	CREATE INDEX IF NOT EXISTS idx_executions_start_time ON job_executions(start_time);
	CREATE INDEX IF NOT EXISTS idx_executions_status ON job_executions(status);
	`

	_, err := sp.db.Exec(schema)
	return err
}

// SaveJob saves a scheduled job to the database
func (sp *SQLitePersistence) SaveJob(ctx context.Context, job *ScheduledJob) error {
	sj, err := job.ToSerializable()
	if err != nil {
		return fmt.Errorf("failed to serialize job: %w", err)
	}

	query := `
		INSERT INTO scheduled_jobs (id, name, pipeline, cron_expr, enabled, next_run, last_run, last_result, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	var nextRun, lastRun interface{}
	if !sj.NextRun.IsZero() {
		nextRun = sj.NextRun
	}
	if !sj.LastRun.IsZero() {
		lastRun = sj.LastRun
	}

	_, err = sp.db.ExecContext(ctx, query,
		sj.ID, sj.Name, sj.Pipeline, sj.CronExpr, sj.Enabled,
		nextRun, lastRun, sj.LastResult,
		sj.CreatedAt, sj.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to save job: %w", err)
	}

	return nil
}

// GetJob retrieves a scheduled job by ID
func (sp *SQLitePersistence) GetJob(ctx context.Context, id string) (*ScheduledJob, error) {
	query := `
		SELECT id, name, pipeline, cron_expr, enabled, next_run, last_run, last_result, created_at, updated_at
		FROM scheduled_jobs
		WHERE id = ?
	`

	var sj SerializableScheduledJob
	var nextRun, lastRun sql.NullTime

	err := sp.db.QueryRowContext(ctx, query, id).Scan(
		&sj.ID, &sj.Name, &sj.Pipeline, &sj.CronExpr, &sj.Enabled,
		&nextRun, &lastRun, &sj.LastResult,
		&sj.CreatedAt, &sj.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("job not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get job: %w", err)
	}

	if nextRun.Valid {
		sj.NextRun = nextRun.Time
	}
	if lastRun.Valid {
		sj.LastRun = lastRun.Time
	}

	return sj.FromSerializable()
}

// ListJobs retrieves all scheduled jobs
func (sp *SQLitePersistence) ListJobs(ctx context.Context) ([]*ScheduledJob, error) {
	query := `
		SELECT id, name, pipeline, cron_expr, enabled, next_run, last_run, last_result, created_at, updated_at
		FROM scheduled_jobs
		ORDER BY created_at DESC
	`

	rows, err := sp.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list jobs: %w", err)
	}
	defer rows.Close()

	var jobs []*ScheduledJob
	for rows.Next() {
		var sj SerializableScheduledJob
		var nextRun, lastRun sql.NullTime

		err := rows.Scan(
			&sj.ID, &sj.Name, &sj.Pipeline, &sj.CronExpr, &sj.Enabled,
			&nextRun, &lastRun, &sj.LastResult,
			&sj.CreatedAt, &sj.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan job: %w", err)
		}

		if nextRun.Valid {
			sj.NextRun = nextRun.Time
		}
		if lastRun.Valid {
			sj.LastRun = lastRun.Time
		}

		job, err := sj.FromSerializable()
		if err != nil {
			return nil, fmt.Errorf("failed to deserialize job: %w", err)
		}

		jobs = append(jobs, job)
	}

	return jobs, rows.Err()
}

// DeleteJob deletes a scheduled job
func (sp *SQLitePersistence) DeleteJob(ctx context.Context, id string) error {
	query := `DELETE FROM scheduled_jobs WHERE id = ?`

	result, err := sp.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete job: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("job not found: %s", id)
	}

	return nil
}

// UpdateJob updates a scheduled job
func (sp *SQLitePersistence) UpdateJob(ctx context.Context, job *ScheduledJob) error {
	sj, err := job.ToSerializable()
	if err != nil {
		return fmt.Errorf("failed to serialize job: %w", err)
	}

	query := `
		UPDATE scheduled_jobs
		SET name = ?, pipeline = ?, cron_expr = ?, enabled = ?, next_run = ?, last_run = ?, last_result = ?, updated_at = ?
		WHERE id = ?
	`

	var nextRun, lastRun interface{}
	if !sj.NextRun.IsZero() {
		nextRun = sj.NextRun
	}
	if !sj.LastRun.IsZero() {
		lastRun = sj.LastRun
	}

	result, err := sp.db.ExecContext(ctx, query,
		sj.Name, sj.Pipeline, sj.CronExpr, sj.Enabled,
		nextRun, lastRun, sj.LastResult, sj.UpdatedAt,
		sj.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update job: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("job not found: %s", sj.ID)
	}

	return nil
}

// SavePipeline saves a pipeline configuration
func (sp *SQLitePersistence) SavePipeline(ctx context.Context, pipeline *PipelineConfig) error {
	configJSON, err := json.Marshal(pipeline)
	if err != nil {
		return fmt.Errorf("failed to marshal pipeline config: %w", err)
	}

	query := `
		INSERT INTO pipelines (name, description, config, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
	`

	now := time.Now()
	_, err = sp.db.ExecContext(ctx, query,
		pipeline.Name, pipeline.Description, string(configJSON),
		now, now,
	)

	if err != nil {
		return fmt.Errorf("failed to save pipeline: %w", err)
	}

	return nil
}

// GetPipeline retrieves a pipeline by name
func (sp *SQLitePersistence) GetPipeline(ctx context.Context, name string) (*PipelineConfig, error) {
	query := `
		SELECT config
		FROM pipelines
		WHERE name = ?
	`

	var configJSON string
	err := sp.db.QueryRowContext(ctx, query, name).Scan(&configJSON)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("pipeline not found: %s", name)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get pipeline: %w", err)
	}

	var pipeline PipelineConfig
	if err := json.Unmarshal([]byte(configJSON), &pipeline); err != nil {
		return nil, fmt.Errorf("failed to unmarshal pipeline config: %w", err)
	}

	return &pipeline, nil
}

// ListPipelines retrieves all pipelines
func (sp *SQLitePersistence) ListPipelines(ctx context.Context) ([]*PipelineConfig, error) {
	query := `
		SELECT config
		FROM pipelines
		ORDER BY created_at DESC
	`

	rows, err := sp.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list pipelines: %w", err)
	}
	defer rows.Close()

	var pipelines []*PipelineConfig
	for rows.Next() {
		var configJSON string
		if err := rows.Scan(&configJSON); err != nil {
			return nil, fmt.Errorf("failed to scan pipeline: %w", err)
		}

		var pipeline PipelineConfig
		if err := json.Unmarshal([]byte(configJSON), &pipeline); err != nil {
			return nil, fmt.Errorf("failed to unmarshal pipeline config: %w", err)
		}

		pipelines = append(pipelines, &pipeline)
	}

	return pipelines, rows.Err()
}

// DeletePipeline deletes a pipeline
func (sp *SQLitePersistence) DeletePipeline(ctx context.Context, name string) error {
	query := `DELETE FROM pipelines WHERE name = ?`

	result, err := sp.db.ExecContext(ctx, query, name)
	if err != nil {
		return fmt.Errorf("failed to delete pipeline: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("pipeline not found: %s", name)
	}

	return nil
}

// UpdatePipeline updates a pipeline
func (sp *SQLitePersistence) UpdatePipeline(ctx context.Context, pipeline *PipelineConfig) error {
	configJSON, err := json.Marshal(pipeline)
	if err != nil {
		return fmt.Errorf("failed to marshal pipeline config: %w", err)
	}

	query := `
		UPDATE pipelines
		SET description = ?, config = ?, updated_at = ?
		WHERE name = ?
	`

	result, err := sp.db.ExecContext(ctx, query,
		pipeline.Description, string(configJSON), time.Now(),
		pipeline.Name,
	)
	if err != nil {
		return fmt.Errorf("failed to update pipeline: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("pipeline not found: %s", pipeline.Name)
	}

	return nil
}

// SaveExecution saves a job execution record
func (sp *SQLitePersistence) SaveExecution(ctx context.Context, execution *JobExecutionRecord) error {
	sje, err := execution.ToSerializable()
	if err != nil {
		return fmt.Errorf("failed to serialize execution: %w", err)
	}

	query := `
		INSERT INTO job_executions (id, job_id, pipeline, start_time, end_time, duration, status, error, context, steps, triggered_by)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	var endTime interface{}
	if !sje.EndTime.IsZero() {
		endTime = sje.EndTime
	}

	_, err = sp.db.ExecContext(ctx, query,
		sje.ID, sje.JobID, sje.Pipeline, sje.StartTime,
		endTime, sje.Duration, sje.Status, sje.Error,
		sje.Context, sje.Steps, sje.TriggeredBy,
	)

	if err != nil {
		return fmt.Errorf("failed to save execution: %w", err)
	}

	return nil
}

// GetExecution retrieves a job execution by ID
func (sp *SQLitePersistence) GetExecution(ctx context.Context, id string) (*JobExecutionRecord, error) {
	query := `
		SELECT id, job_id, pipeline, start_time, end_time, duration, status, error, context, steps, triggered_by
		FROM job_executions
		WHERE id = ?
	`

	var sje SerializableJobExecution
	var endTime sql.NullTime

	err := sp.db.QueryRowContext(ctx, query, id).Scan(
		&sje.ID, &sje.JobID, &sje.Pipeline, &sje.StartTime,
		&endTime, &sje.Duration, &sje.Status, &sje.Error,
		&sje.Context, &sje.Steps, &sje.TriggeredBy,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("execution not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get execution: %w", err)
	}

	if endTime.Valid {
		sje.EndTime = endTime.Time
	}

	return sje.FromSerializable()
}

// ListExecutions retrieves job executions for a specific job
func (sp *SQLitePersistence) ListExecutions(ctx context.Context, jobID string, limit int) ([]*JobExecutionRecord, error) {
	query := `
		SELECT id, job_id, pipeline, start_time, end_time, duration, status, error, context, steps, triggered_by
		FROM job_executions
		WHERE job_id = ?
		ORDER BY start_time DESC
		LIMIT ?
	`

	rows, err := sp.db.QueryContext(ctx, query, jobID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list executions: %w", err)
	}
	defer rows.Close()

	var executions []*JobExecutionRecord
	for rows.Next() {
		var sje SerializableJobExecution
		var endTime sql.NullTime

		err := rows.Scan(
			&sje.ID, &sje.JobID, &sje.Pipeline, &sje.StartTime,
			&endTime, &sje.Duration, &sje.Status, &sje.Error,
			&sje.Context, &sje.Steps, &sje.TriggeredBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan execution: %w", err)
		}

		if endTime.Valid {
			sje.EndTime = endTime.Time
		}

		execution, err := sje.FromSerializable()
		if err != nil {
			return nil, fmt.Errorf("failed to deserialize execution: %w", err)
		}

		executions = append(executions, execution)
	}

	return executions, rows.Err()
}

// DeleteOldExecutions deletes executions older than the specified time
func (sp *SQLitePersistence) DeleteOldExecutions(ctx context.Context, olderThan time.Time) error {
	query := `DELETE FROM job_executions WHERE start_time < ?`

	result, err := sp.db.ExecContext(ctx, query, olderThan)
	if err != nil {
		return fmt.Errorf("failed to delete old executions: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	GetLogger().Info(fmt.Sprintf("Deleted %d old execution records", rows), nil, Component("persistence"))

	return nil
}

// Health checks the health of the database connection
func (sp *SQLitePersistence) Health(ctx context.Context) error {
	return sp.db.PingContext(ctx)
}

// Close closes the database connection
func (sp *SQLitePersistence) Close() error {
	return sp.db.Close()
}
