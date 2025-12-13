package utils

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSQLitePersistence_JobCRUD(t *testing.T) {
	// Create temporary database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	persistence, err := NewSQLitePersistence(dbPath)
	require.NoError(t, err)
	defer persistence.Close()

	ctx := context.Background()

	// Test Create Job
	now := time.Now()
	job := &ScheduledJob{
		ID:        "test-job-1",
		Name:      "Test Job",
		Pipeline:  "test-pipeline.yaml",
		CronExpr:  "0 9 * * *",
		Enabled:   true,
		CreatedAt: now,
		UpdatedAt: now,
	}

	err = persistence.SaveJob(ctx, job)
	assert.NoError(t, err)

	// Test Read Job
	retrieved, err := persistence.GetJob(ctx, "test-job-1")
	assert.NoError(t, err)
	assert.Equal(t, job.ID, retrieved.ID)
	assert.Equal(t, job.Name, retrieved.Name)
	assert.Equal(t, job.Pipeline, retrieved.Pipeline)
	assert.Equal(t, job.CronExpr, retrieved.CronExpr)
	assert.Equal(t, job.Enabled, retrieved.Enabled)

	// Test Update Job
	job.Name = "Updated Test Job"
	job.Enabled = false
	job.UpdatedAt = time.Now()

	err = persistence.UpdateJob(ctx, job)
	assert.NoError(t, err)

	retrieved, err = persistence.GetJob(ctx, "test-job-1")
	assert.NoError(t, err)
	assert.Equal(t, "Updated Test Job", retrieved.Name)
	assert.False(t, retrieved.Enabled)

	// Test List Jobs
	job2 := &ScheduledJob{
		ID:        "test-job-2",
		Name:      "Test Job 2",
		Pipeline:  "test-pipeline2.yaml",
		CronExpr:  "0 10 * * *",
		Enabled:   true,
		CreatedAt: now,
		UpdatedAt: now,
	}
	err = persistence.SaveJob(ctx, job2)
	assert.NoError(t, err)

	jobs, err := persistence.ListJobs(ctx)
	assert.NoError(t, err)
	assert.Len(t, jobs, 2)

	// Test Delete Job
	err = persistence.DeleteJob(ctx, "test-job-1")
	assert.NoError(t, err)

	_, err = persistence.GetJob(ctx, "test-job-1")
	assert.Error(t, err)

	jobs, err = persistence.ListJobs(ctx)
	assert.NoError(t, err)
	assert.Len(t, jobs, 1)
}

func TestSQLitePersistence_PipelineCRUD(t *testing.T) {
	// Create temporary database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	persistence, err := NewSQLitePersistence(dbPath)
	require.NoError(t, err)
	defer persistence.Close()

	ctx := context.Background()

	// Test Create Pipeline
	pipeline := &PipelineConfig{
		Name:        "test-pipeline",
		Description: "A test pipeline",
		Steps: []pipelines.StepConfig{
			{
				Name:   "step1",
				Plugin: "test.plugin",
			},
		},
	}

	err = persistence.SavePipeline(ctx, pipeline)
	assert.NoError(t, err)

	// Test Read Pipeline
	retrieved, err := persistence.GetPipeline(ctx, "test-pipeline")
	assert.NoError(t, err)
	assert.Equal(t, pipeline.Name, retrieved.Name)
	assert.Equal(t, pipeline.Description, retrieved.Description)
	assert.Len(t, retrieved.Steps, 1)

	// Test Update Pipeline
	pipeline.Description = "Updated description"
	err = persistence.UpdatePipeline(ctx, pipeline)
	assert.NoError(t, err)

	retrieved, err = persistence.GetPipeline(ctx, "test-pipeline")
	assert.NoError(t, err)
	assert.Equal(t, "Updated description", retrieved.Description)

	// Test List Pipelines
	pipeline2 := &PipelineConfig{
		Name:        "test-pipeline-2",
		Description: "Another test pipeline",
		Steps:       []pipelines.StepConfig{},
	}
	err = persistence.SavePipeline(ctx, pipeline2)
	assert.NoError(t, err)

	pipelines, err := persistence.ListPipelines(ctx)
	assert.NoError(t, err)
	assert.Len(t, pipelines, 2)

	// Test Delete Pipeline
	err = persistence.DeletePipeline(ctx, "test-pipeline")
	assert.NoError(t, err)

	_, err = persistence.GetPipeline(ctx, "test-pipeline")
	assert.Error(t, err)

	pipelines, err = persistence.ListPipelines(ctx)
	assert.NoError(t, err)
	assert.Len(t, pipelines, 1)
}

func TestSQLitePersistence_ExecutionCRUD(t *testing.T) {
	// Create temporary database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	persistence, err := NewSQLitePersistence(dbPath)
	require.NoError(t, err)
	defer persistence.Close()

	ctx := context.Background()

	// Create a job first
	job := &ScheduledJob{
		ID:        "test-job",
		Name:      "Test Job",
		Pipeline:  "test-pipeline.yaml",
		CronExpr:  "0 9 * * *",
		Enabled:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err = persistence.SaveJob(ctx, job)
	require.NoError(t, err)

	// Test Create Execution
	startTime := time.Now()
	endTime := startTime.Add(5 * time.Second)
	duration := endTime.Sub(startTime)

	execution := &JobExecutionRecord{
		ID:          "exec-1",
		JobID:       "test-job",
		Pipeline:    "test-pipeline.yaml",
		StartTime:   startTime,
		EndTime:     &endTime,
		Duration:    &duration,
		Status:      "success",
		TriggeredBy: "scheduler",
	}

	err = persistence.SaveExecution(ctx, execution)
	assert.NoError(t, err)

	// Test Read Execution
	retrieved, err := persistence.GetExecution(ctx, "exec-1")
	assert.NoError(t, err)
	assert.Equal(t, execution.ID, retrieved.ID)
	assert.Equal(t, execution.JobID, retrieved.JobID)
	assert.Equal(t, execution.Status, retrieved.Status)

	// Test List Executions
	execution2 := &JobExecutionRecord{
		ID:          "exec-2",
		JobID:       "test-job",
		Pipeline:    "test-pipeline.yaml",
		StartTime:   time.Now(),
		Status:      "running",
		TriggeredBy: "api",
	}
	err = persistence.SaveExecution(ctx, execution2)
	assert.NoError(t, err)

	executions, err := persistence.ListExecutions(ctx, "test-job", 10)
	assert.NoError(t, err)
	assert.Len(t, executions, 2)

	// Test Delete Old Executions
	cutoffTime := time.Now().Add(1 * time.Hour)
	err = persistence.DeleteOldExecutions(ctx, cutoffTime)
	assert.NoError(t, err)

	executions, err = persistence.ListExecutions(ctx, "test-job", 10)
	assert.NoError(t, err)
	// Both executions should be deleted since they're older than cutoff
	assert.Len(t, executions, 0)
}

func TestSQLitePersistence_Health(t *testing.T) {
	// Create temporary database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	persistence, err := NewSQLitePersistence(dbPath)
	require.NoError(t, err)
	defer persistence.Close()

	ctx := context.Background()

	// Test Health Check
	err = persistence.Health(ctx)
	assert.NoError(t, err)

	// Close and test health again
	persistence.Close()
	err = persistence.Health(ctx)
	assert.Error(t, err)
}

func TestSQLitePersistence_Serialization(t *testing.T) {
	// Test ScheduledJob serialization
	now := time.Now()
	nextRun := now.Add(1 * time.Hour)
	lastRun := now.Add(-1 * time.Hour)

	job := &ScheduledJob{
		ID:       "test-job",
		Name:     "Test Job",
		Pipeline: "test.yaml",
		CronExpr: "0 9 * * *",
		Enabled:  true,
		NextRun:  &nextRun,
		LastRun:  &lastRun,
		LastResult: &PipelineExecutionResult{
			Success: true,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Test ToSerializable
	serializable, err := job.ToSerializable()
	assert.NoError(t, err)
	assert.Equal(t, job.ID, serializable.ID)
	assert.Equal(t, job.Name, serializable.Name)

	// Test FromSerializable
	deserialized, err := serializable.FromSerializable()
	assert.NoError(t, err)
	assert.Equal(t, job.ID, deserialized.ID)
	assert.Equal(t, job.Name, deserialized.Name)
	assert.Equal(t, job.Enabled, deserialized.Enabled)
}

func TestSQLitePersistence_ConcurrentAccess(t *testing.T) {
	// Create temporary database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	persistence, err := NewSQLitePersistence(dbPath)
	require.NoError(t, err)
	defer persistence.Close()

	ctx := context.Background()

	// Test concurrent writes
	done := make(chan bool)
	errors := make(chan error, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			job := &ScheduledJob{
				ID:        string(rune('a' + id)),
				Name:      "Test Job",
				Pipeline:  "test.yaml",
				CronExpr:  "0 9 * * *",
				Enabled:   true,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			if err := persistence.SaveJob(ctx, job); err != nil {
				errors <- err
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("Concurrent write error: %v", err)
	}

	// Verify all jobs were saved
	jobs, err := persistence.ListJobs(ctx)
	assert.NoError(t, err)
	assert.Len(t, jobs, 10)
}

func TestSQLitePersistence_DatabaseCreation(t *testing.T) {
	// Test that database directory is created if it doesn't exist
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "subdir", "nested", "test.db")

	persistence, err := NewSQLitePersistence(dbPath)
	require.NoError(t, err)
	defer persistence.Close()

	// Verify database file exists
	_, err = os.Stat(dbPath)
	assert.NoError(t, err)
}

func TestSQLitePersistence_EmptyResults(t *testing.T) {
	// Create temporary database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	persistence, err := NewSQLitePersistence(dbPath)
	require.NoError(t, err)
	defer persistence.Close()

	ctx := context.Background()

	// Test empty list
	jobs, err := persistence.ListJobs(ctx)
	assert.NoError(t, err)
	assert.Empty(t, jobs)

	pipelines, err := persistence.ListPipelines(ctx)
	assert.NoError(t, err)
	assert.Empty(t, pipelines)

	executions, err := persistence.ListExecutions(ctx, "nonexistent", 10)
	assert.NoError(t, err)
	assert.Empty(t, executions)

	// Test nonexistent get
	_, err = persistence.GetJob(ctx, "nonexistent")
	assert.Error(t, err)

	_, err = persistence.GetPipeline(ctx, "nonexistent")
	assert.Error(t, err)

	_, err = persistence.GetExecution(ctx, "nonexistent")
	assert.Error(t, err)

	// Test nonexistent delete
	err = persistence.DeleteJob(ctx, "nonexistent")
	assert.Error(t, err)

	err = persistence.DeletePipeline(ctx, "nonexistent")
	assert.Error(t, err)
}
