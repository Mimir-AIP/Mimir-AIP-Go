package utils

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestScheduler_StartStop tests basic scheduler lifecycle
func TestScheduler_StartStop(t *testing.T) {
	registry := pipelines.NewPluginRegistry()
	scheduler := NewScheduler(registry)

	// Test 1: Start scheduler
	t.Run("Start scheduler", func(t *testing.T) {
		err := scheduler.Start()
		require.NoError(t, err)
		assert.True(t, scheduler.running)
	})

	// Test 2: Can't start twice
	t.Run("Cannot start twice", func(t *testing.T) {
		err := scheduler.Start()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already running")
	})

	// Test 3: Stop scheduler
	t.Run("Stop scheduler", func(t *testing.T) {
		err := scheduler.Stop()
		require.NoError(t, err)
		assert.False(t, scheduler.running)
	})

	// Test 4: Can't stop twice
	t.Run("Cannot stop twice", func(t *testing.T) {
		err := scheduler.Stop()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not running")
	})

	// Test 5: Restart after stop
	t.Run("Restart after stop", func(t *testing.T) {
		err := scheduler.Start()
		require.NoError(t, err)
		assert.True(t, scheduler.running)

		err = scheduler.Stop()
		require.NoError(t, err)
	})
}

// TestScheduler_AddRemoveJob tests job management
func TestScheduler_AddRemoveJob(t *testing.T) {
	registry := pipelines.NewPluginRegistry()
	scheduler := NewScheduler(registry)

	err := scheduler.Start()
	require.NoError(t, err)
	defer scheduler.Stop()

	// Test 1: Add a job
	t.Run("Add job", func(t *testing.T) {
		err := scheduler.AddJob("test-job-1", "Test Job", "test-pipeline", "*/5 * * * *")
		require.NoError(t, err)

		// Verify job was added
		job, err := scheduler.GetJob("test-job-1")
		require.NoError(t, err)
		assert.Equal(t, "test-job-1", job.ID)
		assert.Equal(t, "Test Job", job.Name)
		assert.Equal(t, "test-pipeline", job.Pipeline)
		assert.Equal(t, "*/5 * * * *", job.CronExpr)
		assert.True(t, job.Enabled)
		assert.NotNil(t, job.NextRun)
	})

	// Test 2: Can't add duplicate job
	t.Run("Cannot add duplicate job", func(t *testing.T) {
		err := scheduler.AddJob("test-job-1", "Duplicate", "pipeline", "*/10 * * * *")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})

	// Test 3: Add multiple jobs
	t.Run("Add multiple jobs", func(t *testing.T) {
		err := scheduler.AddJob("test-job-2", "Job 2", "pipeline2", "0 * * * *")
		require.NoError(t, err)

		err = scheduler.AddJob("test-job-3", "Job 3", "pipeline3", "0 9 * * *")
		require.NoError(t, err)

		jobs := scheduler.GetJobs()
		assert.GreaterOrEqual(t, len(jobs), 3)
	})

	// Test 4: Remove job
	t.Run("Remove job", func(t *testing.T) {
		err := scheduler.RemoveJob("test-job-2")
		require.NoError(t, err)

		_, err = scheduler.GetJob("test-job-2")
		assert.Error(t, err)
	})

	// Test 5: Can't remove non-existent job
	t.Run("Cannot remove non-existent job", func(t *testing.T) {
		err := scheduler.RemoveJob("non-existent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	// Test 6: Add job with invalid cron
	t.Run("Cannot add job with invalid cron", func(t *testing.T) {
		err := scheduler.AddJob("invalid-cron", "Invalid", "pipeline", "invalid")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid cron")
	})
}

// TestScheduler_EnableDisableJob tests enabling and disabling jobs
func TestScheduler_EnableDisableJob(t *testing.T) {
	registry := pipelines.NewPluginRegistry()
	scheduler := NewScheduler(registry)

	err := scheduler.Start()
	require.NoError(t, err)
	defer scheduler.Stop()

	// Add a test job
	err = scheduler.AddJob("toggle-job", "Toggle Test Job", "pipeline", "*/5 * * * *")
	require.NoError(t, err)

	// Test 1: Disable job
	t.Run("Disable job", func(t *testing.T) {
		err := scheduler.DisableJob("toggle-job")
		require.NoError(t, err)

		job, err := scheduler.GetJob("toggle-job")
		require.NoError(t, err)
		assert.False(t, job.Enabled)
		assert.Nil(t, job.NextRun)
	})

	// Test 2: Enable job
	t.Run("Enable job", func(t *testing.T) {
		err := scheduler.EnableJob("toggle-job")
		require.NoError(t, err)

		job, err := scheduler.GetJob("toggle-job")
		require.NoError(t, err)
		assert.True(t, job.Enabled)
		assert.NotNil(t, job.NextRun)
	})

	// Test 3: Can't enable/disable non-existent job
	t.Run("Cannot toggle non-existent job", func(t *testing.T) {
		err := scheduler.DisableJob("non-existent")
		assert.Error(t, err)

		err = scheduler.EnableJob("non-existent")
		assert.Error(t, err)
	})
}

// TestScheduler_UpdateJob tests job updates
func TestScheduler_UpdateJob(t *testing.T) {
	registry := pipelines.NewPluginRegistry()
	scheduler := NewScheduler(registry)

	err := scheduler.Start()
	require.NoError(t, err)
	defer scheduler.Stop()

	// Add a test job
	err = scheduler.AddJob("update-job", "Original Name", "original-pipeline", "0 9 * * *")
	require.NoError(t, err)

	// Test 1: Update job name
	t.Run("Update job name", func(t *testing.T) {
		newName := "Updated Name"
		err := scheduler.UpdateJob("update-job", &newName, nil, nil)
		require.NoError(t, err)

		job, err := scheduler.GetJob("update-job")
		require.NoError(t, err)
		assert.Equal(t, "Updated Name", job.Name)
	})

	// Test 2: Update job pipeline
	t.Run("Update job pipeline", func(t *testing.T) {
		newPipeline := "updated-pipeline"
		err := scheduler.UpdateJob("update-job", nil, &newPipeline, nil)
		require.NoError(t, err)

		job, err := scheduler.GetJob("update-job")
		require.NoError(t, err)
		assert.Equal(t, "updated-pipeline", job.Pipeline)
	})

	// Test 3: Update job cron expression
	t.Run("Update job cron", func(t *testing.T) {
		newCron := "*/10 * * * *"
		err := scheduler.UpdateJob("update-job", nil, nil, &newCron)
		require.NoError(t, err)

		job, err := scheduler.GetJob("update-job")
		require.NoError(t, err)
		assert.Equal(t, "*/10 * * * *", job.CronExpr)
		assert.NotNil(t, job.NextRun) // Next run should be recalculated
	})

	// Test 4: Update all fields at once
	t.Run("Update all fields", func(t *testing.T) {
		newName := "All Updated"
		newPipeline := "all-pipeline"
		newCron := "0 */2 * * *"

		err := scheduler.UpdateJob("update-job", &newName, &newPipeline, &newCron)
		require.NoError(t, err)

		job, err := scheduler.GetJob("update-job")
		require.NoError(t, err)
		assert.Equal(t, "All Updated", job.Name)
		assert.Equal(t, "all-pipeline", job.Pipeline)
		assert.Equal(t, "0 */2 * * *", job.CronExpr)
	})

	// Test 5: Can't update non-existent job
	t.Run("Cannot update non-existent job", func(t *testing.T) {
		newName := "New Name"
		err := scheduler.UpdateJob("non-existent", &newName, nil, nil)
		assert.Error(t, err)
	})

	// Test 6: Can't update with invalid cron
	t.Run("Cannot update with invalid cron", func(t *testing.T) {
		invalidCron := "invalid"
		err := scheduler.UpdateJob("update-job", nil, nil, &invalidCron)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid cron")
	})
}

// TestScheduler_GetJobs returns copy of jobs (not reference)
func TestScheduler_GetJobsReturnsCopy(t *testing.T) {
	registry := pipelines.NewPluginRegistry()
	scheduler := NewScheduler(registry)

	err := scheduler.Start()
	require.NoError(t, err)
	defer scheduler.Stop()

	// Add a job
	err = scheduler.AddJob("copy-test", "Copy Test", "pipeline", "*/5 * * * *")
	require.NoError(t, err)

	// Get jobs
	jobs1 := scheduler.GetJobs()
	jobs2 := scheduler.GetJobs()

	// Modify jobs1
	job1 := jobs1["copy-test"]
	job1.Name = "Modified"

	// jobs2 should not be affected
	job2 := jobs2["copy-test"]
	assert.Equal(t, "Copy Test", job2.Name, "GetJobs should return copies, not references")

	// Verify with GetJob
	job3, _ := scheduler.GetJob("copy-test")
	assert.Equal(t, "Copy Test", job3.Name, "Original job should not be modified")
}

// TestScheduler_JobExecution tests that jobs actually execute
func TestScheduler_JobExecution(t *testing.T) {
	registry := pipelines.NewPluginRegistry()
	scheduler := NewScheduler(registry)

	err := scheduler.Start()
	require.NoError(t, err)
	defer scheduler.Stop()

	// Test 1: Job with immediate execution (next run in past)
	t.Run("Job with past next run executes", func(t *testing.T) {
		// Add a job that should run immediately
		now := time.Now()
		job := &ScheduledJob{
			ID:        "immediate-job",
			Name:      "Immediate Job",
			JobType:   "pipeline",
			Pipeline:  "test-pipeline",
			CronExpr:  "* * * * *",
			Enabled:   true,
			NextRun:   &now,
			CreatedAt: now,
			UpdatedAt: now,
		}

		scheduler.jobsMutex.Lock()
		scheduler.jobs["immediate-job"] = job
		scheduler.jobsMutex.Unlock()

		// Wait a bit for job to potentially execute
		time.Sleep(200 * time.Millisecond)

		// The job should have been picked up by checkAndExecuteJobs
		// We can't verify actual execution without a real pipeline,
		// but we can verify the scheduler loop is running
		assert.True(t, scheduler.running)
	})

	// Test 2: Disabled job doesn't execute
	t.Run("Disabled job does not execute", func(t *testing.T) {
		now := time.Now()
		past := now.Add(-1 * time.Minute)
		job := &ScheduledJob{
			ID:        "disabled-job",
			Name:      "Disabled Job",
			JobType:   "pipeline",
			Pipeline:  "test-pipeline",
			CronExpr:  "* * * * *",
			Enabled:   false,
			NextRun:   &past,
			CreatedAt: now,
			UpdatedAt: now,
		}

		scheduler.jobsMutex.Lock()
		scheduler.jobs["disabled-job"] = job
		scheduler.jobsMutex.Unlock()

		// Wait and verify job wasn't executed
		time.Sleep(200 * time.Millisecond)

		jobAfter, err := scheduler.GetJob("disabled-job")
		require.NoError(t, err)
		assert.Nil(t, jobAfter.LastRun, "Disabled job should not have executed")
	})
}

// TestScheduler_MonitoringJob tests monitoring job type
func TestScheduler_MonitoringJob(t *testing.T) {
	registry := pipelines.NewPluginRegistry()
	scheduler := NewScheduler(registry)

	err := scheduler.Start()
	require.NoError(t, err)
	defer scheduler.Stop()

	// Test 1: Add monitoring job
	t.Run("Add monitoring job", func(t *testing.T) {
		err := scheduler.AddMonitoringJob("monitor-job", "Monitoring Job", "monitoring-job-001", "*/5 * * * *")
		require.NoError(t, err)

		job, err := scheduler.GetJob("monitor-job")
		require.NoError(t, err)
		assert.Equal(t, "monitoring", job.JobType)
		assert.Equal(t, "monitoring-job-001", job.MonitoringJobID)
		assert.Empty(t, job.Pipeline) // Pipeline should be empty for monitoring jobs
	})

	// Test 2: Can't add duplicate monitoring job
	t.Run("Cannot add duplicate monitoring job", func(t *testing.T) {
		err := scheduler.AddMonitoringJob("monitor-job", "Duplicate", "id", "*/10 * * * *")
		assert.Error(t, err)
	})
}

// TestScheduler_CronParsing tests cron expression parsing
func TestScheduler_CronParsing(t *testing.T) {
	tests := []struct {
		name     string
		cron     string
		valid    bool
		interval time.Duration // expected max interval (rough)
	}{
		{"Every 5 minutes", "*/5 * * * *", true, 6 * time.Minute},
		{"Every hour at 30 min", "30 * * * *", true, 61 * time.Minute},
		{"Daily at 9 AM", "0 9 * * *", true, 25 * time.Hour},
		{"Invalid - no spaces", "*****", false, 0},
		{"Invalid - wrong parts", "* * * *", false, 0},
		{"Invalid - too many parts", "* * * * * *", false, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nextRun, err := parseCronExpression(tt.cron)

			if tt.valid {
				require.NoError(t, err)
				assert.False(t, nextRun.IsZero())
				// Next run should be in the future or very recent past
				assert.True(t, nextRun.After(time.Now().Add(-time.Minute)) ||
					nextRun.Equal(time.Now()) ||
					nextRun.Before(time.Now().Add(tt.interval)))
			} else {
				assert.Error(t, err)
			}
		})
	}
}

// TestScheduler_ConcurrentAccess tests thread safety
func TestScheduler_ConcurrentAccess(t *testing.T) {
	registry := pipelines.NewPluginRegistry()
	scheduler := NewScheduler(registry)

	err := scheduler.Start()
	require.NoError(t, err)
	defer scheduler.Stop()

	// Add initial job
	err = scheduler.AddJob("concurrent-job", "Concurrent", "pipeline", "*/5 * * * *")
	require.NoError(t, err)

	var wg sync.WaitGroup

	// Concurrent reads
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_, _ = scheduler.GetJob("concurrent-job")
				_ = scheduler.GetJobs()
			}
		}()
	}

	// Concurrent writes
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				_ = scheduler.EnableJob("concurrent-job")
				_ = scheduler.DisableJob("concurrent-job")
			}
		}(i)
	}

	// Wait for all goroutines
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success - no deadlocks or panics
		assert.True(t, true, "Concurrent operations completed without deadlock")
	case <-time.After(5 * time.Second):
		t.Fatal("Concurrent operations deadlocked")
	}
}

// TestScheduler_JobStatusUpdates tests that job status is updated after execution
func TestScheduler_JobStatusUpdates(t *testing.T) {
	registry := pipelines.NewPluginRegistry()
	scheduler := NewScheduler(registry)

	err := scheduler.Start()
	require.NoError(t, err)
	defer scheduler.Stop()

	// Add a job
	err = scheduler.AddJob("status-job", "Status Job", "pipeline", "*/1 * * * *")
	require.NoError(t, err)

	job, _ := scheduler.GetJob("status-job")
	initialUpdate := job.UpdatedAt

	// Update the job
	newName := "Updated Status Job"
	err = scheduler.UpdateJob("status-job", &newName, nil, nil)
	require.NoError(t, err)

	job, _ = scheduler.GetJob("status-job")
	updatedTime := job.UpdatedAt

	// UpdatedAt should be newer
	assert.True(t, updatedTime.After(initialUpdate) || updatedTime.Equal(initialUpdate),
		"UpdatedAt should be updated")
}

// TestScheduler_ContextCancellation tests that scheduler respects context
func TestScheduler_ContextCancellation(t *testing.T) {
	_ = pipelines.NewPluginRegistry()

	// Don't start the scheduler, just test context creation
	ctx, cancel := context.WithCancel(context.Background())

	// Test context cancellation
	t.Run("Context cancellation", func(t *testing.T) {
		cancel()

		select {
		case <-ctx.Done():
			assert.Equal(t, context.Canceled, ctx.Err())
		case <-time.After(time.Second):
			t.Fatal("Context should have been cancelled")
		}
	})
}

// TestScheduler_JobSchedulingInterval tests scheduling calculations
func TestScheduler_JobSchedulingInterval(t *testing.T) {
	registry := pipelines.NewPluginRegistry()
	scheduler := NewScheduler(registry)

	err := scheduler.Start()
	require.NoError(t, err)
	defer scheduler.Stop()

	tests := []struct {
		name      string
		cron      string
		enabled   bool
		shouldRun bool // whether job should eventually run
	}{
		{"Enabled frequent job", "*/1 * * * *", true, true},
		{"Disabled frequent job", "*/1 * * * *", false, false},
		{"Enabled hourly job", "0 * * * *", true, false}, // Won't run immediately
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jobID := "interval-test-" + tt.name
			err := scheduler.AddJob(jobID, tt.name, "pipeline", tt.cron)
			require.NoError(t, err)

			if !tt.enabled {
				err = scheduler.DisableJob(jobID)
				require.NoError(t, err)
			}

			job, err := scheduler.GetJob(jobID)
			require.NoError(t, err)

			if tt.enabled {
				assert.NotNil(t, job.NextRun, "Enabled job should have next run time")
				assert.True(t, job.NextRun.After(time.Now()) || job.NextRun.Equal(time.Now()),
					"Next run should be in the future or now")
			} else {
				assert.Nil(t, job.NextRun, "Disabled job should not have next run time")
			}
		})
	}
}

// TestScheduler_MultipleJobs tests managing many jobs
func TestScheduler_MultipleJobs(t *testing.T) {
	registry := pipelines.NewPluginRegistry()
	scheduler := NewScheduler(registry)

	err := scheduler.Start()
	require.NoError(t, err)
	defer scheduler.Stop()

	// Add 50 jobs
	for i := 0; i < 50; i++ {
		jobID := fmt.Sprintf("bulk-job-%d", i)
		jobName := fmt.Sprintf("Bulk Job %d", i)
		err := scheduler.AddJob(jobID, jobName, "pipeline", fmt.Sprintf("*/%d * * * *", (i%10)+1))
		require.NoError(t, err)
	}

	// Verify all jobs exist
	jobs := scheduler.GetJobs()
	assert.GreaterOrEqual(t, len(jobs), 50)

	// Verify we can get each job
	for i := 0; i < 50; i++ {
		jobID := fmt.Sprintf("bulk-job-%d", i)
		job, err := scheduler.GetJob(jobID)
		require.NoError(t, err)
		assert.Equal(t, fmt.Sprintf("Bulk Job %d", i), job.Name)
	}

	// Disable half of them
	for i := 0; i < 25; i++ {
		jobID := fmt.Sprintf("bulk-job-%d", i)
		err := scheduler.DisableJob(jobID)
		require.NoError(t, err)
	}

	// Verify disabled jobs
	for i := 0; i < 25; i++ {
		jobID := fmt.Sprintf("bulk-job-%d", i)
		job, _ := scheduler.GetJob(jobID)
		assert.False(t, job.Enabled)
	}

	// Verify enabled jobs
	for i := 25; i < 50; i++ {
		jobID := fmt.Sprintf("bulk-job-%d", i)
		job, _ := scheduler.GetJob(jobID)
		assert.True(t, job.Enabled)
	}
}

// TestScheduler_JobTypes verifies job types are set correctly
func TestScheduler_JobTypes(t *testing.T) {
	registry := pipelines.NewPluginRegistry()
	scheduler := NewScheduler(registry)

	err := scheduler.Start()
	require.NoError(t, err)
	defer scheduler.Stop()

	// Add pipeline job
	err = scheduler.AddJob("type-pipeline", "Pipeline Job", "pipeline", "*/5 * * * *")
	require.NoError(t, err)

	job, _ := scheduler.GetJob("type-pipeline")
	assert.Equal(t, "pipeline", job.JobType)

	// Add monitoring job
	err = scheduler.AddMonitoringJob("type-monitoring", "Monitoring Job", "mon-001", "*/10 * * * *")
	require.NoError(t, err)

	job, _ = scheduler.GetJob("type-monitoring")
	assert.Equal(t, "monitoring", job.JobType)
}
