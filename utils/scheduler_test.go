package utils

import (
	"fmt"
	"testing"
	"time"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewScheduler(t *testing.T) {
	registry := pipelines.NewPluginRegistry()
	scheduler := NewScheduler(registry)

	assert.NotNil(t, scheduler)
	assert.Equal(t, registry, scheduler.registry)
	assert.NotNil(t, scheduler.jobs)
	assert.NotNil(t, scheduler.stopChan)
	assert.NotNil(t, scheduler.ctx)
	assert.NotNil(t, scheduler.cancel)
	assert.False(t, scheduler.running)
}

func TestSchedulerStartStop(t *testing.T) {
	registry := pipelines.NewPluginRegistry()
	scheduler := NewScheduler(registry)

	// Test initial start
	err := scheduler.Start()
	assert.NoError(t, err)
	assert.True(t, scheduler.running)

	// Test double start
	err = scheduler.Start()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already running")

	// Test stop
	err = scheduler.Stop()
	assert.NoError(t, err)
	assert.False(t, scheduler.running)

	// Test double stop
	err = scheduler.Stop()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not running")
}

func TestSchedulerAddJob(t *testing.T) {
	registry := pipelines.NewPluginRegistry()
	scheduler := NewScheduler(registry)

	tests := []struct {
		name      string
		id        string
		nameStr   string
		pipeline  string
		cronExpr  string
		expectErr bool
		errMsg    string
	}{
		{
			name:      "valid job",
			id:        "job1",
			nameStr:   "Test Job",
			pipeline:  "test-pipeline",
			cronExpr:  "*/5 * * * *",
			expectErr: false,
		},
		{
			name:      "duplicate job ID",
			id:        "job1",
			nameStr:   "Duplicate Job",
			pipeline:  "test-pipeline",
			cronExpr:  "*/10 * * * *",
			expectErr: true,
			errMsg:    "already exists",
		},
		{
			name:      "invalid cron expression",
			id:        "job2",
			nameStr:   "Invalid Cron",
			pipeline:  "test-pipeline",
			cronExpr:  "invalid",
			expectErr: true,
			errMsg:    "invalid cron expression",
		},
		{
			name:      "wildcard minutes not supported",
			id:        "job3",
			nameStr:   "Wildcard Minutes",
			pipeline:  "test-pipeline",
			cronExpr:  "* * * * *",
			expectErr: true,
			errMsg:    "wildcard minutes not supported",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := scheduler.AddJob(tt.id, tt.nameStr, tt.pipeline, tt.cronExpr)

			if tt.expectErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)

				// Verify job was added correctly
				job, err := scheduler.GetJob(tt.id)
				assert.NoError(t, err)
				assert.Equal(t, tt.id, job.ID)
				assert.Equal(t, tt.nameStr, job.Name)
				assert.Equal(t, tt.pipeline, job.Pipeline)
				assert.Equal(t, tt.cronExpr, job.CronExpr)
				assert.True(t, job.Enabled)
				assert.NotNil(t, job.NextRun)
				assert.NotNil(t, job.CreatedAt)
				assert.NotNil(t, job.UpdatedAt)
			}
		})
	}
}

func TestSchedulerRemoveJob(t *testing.T) {
	registry := pipelines.NewPluginRegistry()
	scheduler := NewScheduler(registry)

	// Add a job first
	err := scheduler.AddJob("job1", "Test Job", "test-pipeline", "*/5 * * * *")
	require.NoError(t, err)

	// Test successful removal
	err = scheduler.RemoveJob("job1")
	assert.NoError(t, err)

	// Verify job is gone
	_, err = scheduler.GetJob("job1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	// Test removing non-existent job
	err = scheduler.RemoveJob("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestSchedulerEnableDisableJob(t *testing.T) {
	registry := pipelines.NewPluginRegistry()
	scheduler := NewScheduler(registry)

	// Add a job first
	err := scheduler.AddJob("job1", "Test Job", "test-pipeline", "*/5 * * * *")
	require.NoError(t, err)

	// Test disable
	err = scheduler.DisableJob("job1")
	assert.NoError(t, err)

	job, err := scheduler.GetJob("job1")
	assert.NoError(t, err)
	assert.False(t, job.Enabled)
	assert.Nil(t, job.NextRun)

	// Test enable
	err = scheduler.EnableJob("job1")
	assert.NoError(t, err)

	job, err = scheduler.GetJob("job1")
	assert.NoError(t, err)
	assert.True(t, job.Enabled)
	assert.NotNil(t, job.NextRun)

	// Test operations on non-existent job
	err = scheduler.EnableJob("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	err = scheduler.DisableJob("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestSchedulerGetJobs(t *testing.T) {
	registry := pipelines.NewPluginRegistry()
	scheduler := NewScheduler(registry)

	// Initially empty
	jobs := scheduler.GetJobs()
	assert.Empty(t, jobs)

	// Add some jobs
	err := scheduler.AddJob("job1", "Job 1", "pipeline1", "*/5 * * * *")
	require.NoError(t, err)
	err = scheduler.AddJob("job2", "Job 2", "pipeline2", "0 */2 * * *")
	require.NoError(t, err)

	jobs = scheduler.GetJobs()
	assert.Len(t, jobs, 2)
	assert.Contains(t, jobs, "job1")
	assert.Contains(t, jobs, "job2")

	// Verify returned jobs are copies (modifying shouldn't affect original)
	jobs["job1"].Name = "Modified"
	originalJob, _ := scheduler.GetJob("job1")
	assert.NotEqual(t, "Modified", originalJob.Name)
}

func TestParseCronExpression(t *testing.T) {
	tests := []struct {
		name      string
		cronExpr  string
		expectErr bool
		errMsg    string
	}{
		{
			name:      "every 5 minutes",
			cronExpr:  "*/5 * * * *",
			expectErr: false,
		},
		{
			name:      "every 10 minutes",
			cronExpr:  "*/10 * * * *",
			expectErr: false,
		},
		{
			name:      "specific minute",
			cronExpr:  "30 * * * *",
			expectErr: false,
		},
		{
			name:      "invalid format - too few parts",
			cronExpr:  "*/5 * * *",
			expectErr: true,
			errMsg:    "invalid cron expression format",
		},
		{
			name:      "invalid format - too many parts",
			cronExpr:  "*/5 * * * * *",
			expectErr: true,
			errMsg:    "invalid cron expression format",
		},
		{
			name:      "wildcard minutes",
			cronExpr:  "* * * * *",
			expectErr: true,
			errMsg:    "wildcard minutes not supported",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseCronExpression(tt.cronExpr)

			if tt.expectErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				assert.True(t, result.IsZero())
			} else {
				assert.NoError(t, err)
				assert.False(t, result.IsZero())
				assert.True(t, result.After(time.Now()))
			}
		})
	}
}

func TestSchedulerUpdateNextRun(t *testing.T) {
	registry := pipelines.NewPluginRegistry()
	scheduler := NewScheduler(registry)

	job := &ScheduledJob{
		ID:       "test-job",
		CronExpr: "*/5 * * * *",
		Enabled:  true,
	}

	// Test enabled job
	scheduler.updateNextRun(job)
	assert.NotNil(t, job.NextRun)
	assert.True(t, job.NextRun.After(time.Now()))

	// Test disabled job
	job.Enabled = false
	scheduler.updateNextRun(job)
	assert.Nil(t, job.NextRun)

	// Test invalid cron expression
	job.CronExpr = "invalid"
	job.Enabled = true
	scheduler.updateNextRun(job)
	assert.Nil(t, job.NextRun)
}

func TestSchedulerLoadJobsFromConfig(t *testing.T) {
	registry := pipelines.NewPluginRegistry()
	scheduler := NewScheduler(registry)

	// Test with non-existent config file
	err := scheduler.LoadJobsFromConfig("nonexistent.yaml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load pipelines")
}

func TestSchedulerExecution(t *testing.T) {
	registry := pipelines.NewPluginRegistry()
	scheduler := NewScheduler(registry)

	// Add a job with a cron that should trigger soon
	err := scheduler.AddJob("test-job", "Test Job", "test-pipeline", "*/1 * * * *")
	require.NoError(t, err)

	// Start scheduler
	err = scheduler.Start()
	require.NoError(t, err)

	// Wait a bit to see if job executes
	time.Sleep(2 * time.Second)

	// Stop scheduler
	err = scheduler.Stop()
	assert.NoError(t, err)

	// Note: In a real test, you'd mock the pipeline execution
	// For now, we just verify the scheduler doesn't crash
}

func TestSchedulerContextCancellation(t *testing.T) {
	registry := pipelines.NewPluginRegistry()
	scheduler := NewScheduler(registry)

	// Start scheduler
	err := scheduler.Start()
	require.NoError(t, err)

	// Cancel context
	scheduler.cancel()

	// Wait for scheduler to stop
	time.Sleep(100 * time.Millisecond)

	// Verify scheduler is no longer running
	assert.False(t, scheduler.running)
}

func TestSchedulerConcurrentAccess(t *testing.T) {
	registry := pipelines.NewPluginRegistry()
	scheduler := NewScheduler(registry)

	// Start scheduler
	err := scheduler.Start()
	require.NoError(t, err)
	defer func() { _ = scheduler.Stop() }()

	// Test concurrent job operations
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			jobID := fmt.Sprintf("job%d", id)
			err := scheduler.AddJob(jobID, fmt.Sprintf("Job %d", id), "pipeline", "*/5 * * * *")
			assert.NoError(t, err)

			job, err := scheduler.GetJob(jobID)
			assert.NoError(t, err)
			assert.NotNil(t, job)

			done <- true
		}(i)
	}

	// Wait for all operations to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all jobs were added
	jobs := scheduler.GetJobs()
	assert.Len(t, jobs, 10)
}
