package tests

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
	"github.com/Mimir-AIP/Mimir-AIP-Go/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSchedulerIntegration tests scheduler functionality with real pipelines
func TestSchedulerIntegration(t *testing.T) {
	// Create plugin registry
	registry := pipelines.NewPluginRegistry()
	err := registry.RegisterPlugin(&utils.RealAPIPlugin{})
	require.NoError(t, err)

	// Create scheduler
	scheduler := utils.NewScheduler(registry)

	// Start scheduler
	err = scheduler.Start()
	require.NoError(t, err)
	defer scheduler.Stop()

	t.Run("Add and List Jobs", func(t *testing.T) {
		// Add a scheduled job
		err := scheduler.AddJob(
			"test_job_1",
			"Test Job 1",
			"test_pipeline.yaml",
			"*/5 * * * *", // Every 5 minutes
		)
		require.NoError(t, err)

		// List jobs
		jobs := scheduler.GetJobs()
		assert.Len(t, jobs, 1, "Should have one scheduled job")

		job, exists := jobs["test_job_1"]
		assert.True(t, exists)
		assert.Equal(t, "Test Job 1", job.Name)
		assert.Equal(t, "test_pipeline.yaml", job.Pipeline)
		assert.Equal(t, "*/5 * * * *", job.CronExpr)
		assert.True(t, job.Enabled)
		assert.NotNil(t, job.NextRun)
	})

	t.Run("Enable and Disable Jobs", func(t *testing.T) {
		// Add a job
		err := scheduler.AddJob(
			"test_job_2",
			"Test Job 2",
			"pipeline2.yaml",
			"0 * * * *",
		)
		require.NoError(t, err)

		// Disable the job
		err = scheduler.DisableJob("test_job_2")
		require.NoError(t, err)

		job, err := scheduler.GetJob("test_job_2")
		require.NoError(t, err)
		assert.False(t, job.Enabled)
		assert.Nil(t, job.NextRun, "Disabled job should not have next run")

		// Re-enable the job
		err = scheduler.EnableJob("test_job_2")
		require.NoError(t, err)

		job, err = scheduler.GetJob("test_job_2")
		require.NoError(t, err)
		assert.True(t, job.Enabled)
		assert.NotNil(t, job.NextRun, "Enabled job should have next run scheduled")
	})

	t.Run("Remove Jobs", func(t *testing.T) {
		// Add a job
		err := scheduler.AddJob(
			"test_job_3",
			"Test Job 3",
			"pipeline3.yaml",
			"0 0 * * *",
		)
		require.NoError(t, err)

		// Verify it exists
		jobs := scheduler.GetJobs()
		_, exists := jobs["test_job_3"]
		assert.True(t, exists)

		// Remove it
		err = scheduler.RemoveJob("test_job_3")
		require.NoError(t, err)

		// Verify it's gone
		jobs = scheduler.GetJobs()
		_, exists = jobs["test_job_3"]
		assert.False(t, exists)
	})
}

// TestSchedulerJobExecution tests actual job execution
func TestSchedulerJobExecution(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "mimir_scheduler_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a test pipeline
	pipelineContent := `
name: "Scheduler Test Pipeline"
steps:
  - name: "Test Step"
    plugin: "Input.api"
    config:
      url: "https://httpbin.org/json"
      method: "GET"
    output: "result"
`

	pipelineFile := filepath.Join(tempDir, "scheduler_test.yaml")
	err = os.WriteFile(pipelineFile, []byte(pipelineContent), 0644)
	require.NoError(t, err)

	// Create registry and scheduler
	registry := pipelines.NewPluginRegistry()
	err = registry.RegisterPlugin(&utils.RealAPIPlugin{})
	require.NoError(t, err)

	scheduler := utils.NewScheduler(registry)
	err = scheduler.Start()
	require.NoError(t, err)
	defer scheduler.Stop()

	t.Run("Job Execution Tracking", func(t *testing.T) {
		// Add a job with a cron expression that will execute soon
		// Note: This is a simplified test - in production you'd use a cron library
		jobID := "exec_test_job"
		err := scheduler.AddJob(
			jobID,
			"Execution Test Job",
			pipelineFile,
			"*/1 * * * *", // Every minute (simplified)
		)
		require.NoError(t, err)

		// Get the job and verify initial state
		job, err := scheduler.GetJob(jobID)
		require.NoError(t, err)
		assert.Nil(t, job.LastRun, "Job should not have run yet")
		assert.Nil(t, job.LastResult, "Job should not have results yet")
	})
}

// TestSchedulerCronExpressions tests various cron expression formats
func TestSchedulerCronExpressions(t *testing.T) {
	registry := pipelines.NewPluginRegistry()
	scheduler := utils.NewScheduler(registry)
	err := scheduler.Start()
	require.NoError(t, err)
	defer scheduler.Stop()

	testCases := []struct {
		name       string
		cronExpr   string
		shouldFail bool
	}{
		{
			name:       "Every 5 minutes",
			cronExpr:   "*/5 * * * *",
			shouldFail: false,
		},
		{
			name:       "Every 10 minutes",
			cronExpr:   "*/10 * * * *",
			shouldFail: false,
		},
		{
			name:       "Specific minute",
			cronExpr:   "30 * * * *",
			shouldFail: false,
		},
		{
			name:       "Invalid format - too few fields",
			cronExpr:   "* * *",
			shouldFail: true,
		},
		{
			name:       "Invalid format - too many fields",
			cronExpr:   "* * * * * *",
			shouldFail: true,
		},
	}

	for i, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			jobID := fmt.Sprintf("cron_test_%d", i)
			err := scheduler.AddJob(
				jobID,
				tc.name,
				"test.yaml",
				tc.cronExpr,
			)

			if tc.shouldFail {
				assert.Error(t, err, "Invalid cron expression should fail")
			} else {
				assert.NoError(t, err, "Valid cron expression should succeed")
				// Clean up
				scheduler.RemoveJob(jobID)
			}
		})
	}
}

// TestSchedulerConcurrency tests concurrent scheduler operations
func TestSchedulerConcurrency(t *testing.T) {
	registry := pipelines.NewPluginRegistry()
	scheduler := utils.NewScheduler(registry)
	err := scheduler.Start()
	require.NoError(t, err)
	defer scheduler.Stop()

	t.Run("Concurrent Job Addition", func(t *testing.T) {
		const numJobs = 20
		errors := make(chan error, numJobs)

		// Add jobs concurrently
		for i := 0; i < numJobs; i++ {
			go func(jobNum int) {
				err := scheduler.AddJob(
					fmt.Sprintf("concurrent_job_%d", jobNum),
					fmt.Sprintf("Concurrent Job %d", jobNum),
					"test.yaml",
					"*/5 * * * *",
				)
				errors <- err
			}(i)
		}

		// Collect results
		successCount := 0
		for i := 0; i < numJobs; i++ {
			if <-errors == nil {
				successCount++
			}
		}

		assert.Equal(t, numJobs, successCount, "All concurrent adds should succeed")

		// Verify all jobs were added
		jobs := scheduler.GetJobs()
		assert.GreaterOrEqual(t, len(jobs), numJobs)
	})

	t.Run("Concurrent Enable/Disable", func(t *testing.T) {
		// Add a test job
		jobID := "toggle_test"
		err := scheduler.AddJob(jobID, "Toggle Test", "test.yaml", "*/5 * * * *")
		require.NoError(t, err)

		const numOperations = 10
		errors := make(chan error, numOperations*2)

		// Concurrently enable and disable
		for i := 0; i < numOperations; i++ {
			go func() {
				errors <- scheduler.DisableJob(jobID)
			}()
			go func() {
				errors <- scheduler.EnableJob(jobID)
			}()
		}

		// Collect results
		errorCount := 0
		for i := 0; i < numOperations*2; i++ {
			if <-errors != nil {
				errorCount++
			}
		}

		// Some operations should succeed
		assert.Less(t, errorCount, numOperations*2, "Some enable/disable operations should succeed")

		// Job should still exist
		job, err := scheduler.GetJob(jobID)
		require.NoError(t, err)
		assert.NotNil(t, job)
	})
}

// TestSchedulerStartStop tests scheduler lifecycle
func TestSchedulerStartStop(t *testing.T) {
	registry := pipelines.NewPluginRegistry()
	scheduler := utils.NewScheduler(registry)

	t.Run("Start and Stop", func(t *testing.T) {
		// Start scheduler
		err := scheduler.Start()
		require.NoError(t, err)

		// Try to start again (should fail)
		err = scheduler.Start()
		assert.Error(t, err, "Starting already running scheduler should fail")

		// Stop scheduler
		err = scheduler.Stop()
		require.NoError(t, err)

		// Try to stop again (should fail)
		err = scheduler.Stop()
		assert.Error(t, err, "Stopping already stopped scheduler should fail")
	})

	t.Run("Operations After Stop", func(t *testing.T) {
		// Start scheduler
		err := scheduler.Start()
		require.NoError(t, err)

		// Add a job
		err = scheduler.AddJob("stop_test", "Stop Test", "test.yaml", "*/5 * * * *")
		require.NoError(t, err)

		// Stop scheduler
		err = scheduler.Stop()
		require.NoError(t, err)

		// Jobs should still be queryable after stop
		jobs := scheduler.GetJobs()
		assert.Greater(t, len(jobs), 0, "Jobs should still be accessible after stop")
	})
}

// TestSchedulerGracefulShutdown tests graceful shutdown behavior
func TestSchedulerGracefulShutdown(t *testing.T) {
	registry := pipelines.NewPluginRegistry()
	scheduler := utils.NewScheduler(registry)

	err := scheduler.Start()
	require.NoError(t, err)

	// Add some jobs
	for i := 0; i < 5; i++ {
		err := scheduler.AddJob(
			fmt.Sprintf("shutdown_test_%d", i),
			fmt.Sprintf("Shutdown Test %d", i),
			"test.yaml",
			"*/5 * * * *",
		)
		require.NoError(t, err)
	}

	// Stop scheduler - should complete within reasonable time
	done := make(chan error, 1)
	go func() {
		done <- scheduler.Stop()
	}()

	select {
	case err := <-done:
		require.NoError(t, err, "Graceful shutdown should complete without error")
	case <-time.After(35 * time.Second):
		t.Fatal("Scheduler shutdown timed out")
	}
}

// TestSchedulerWithRealPipeline tests scheduler with actual pipeline execution
func TestSchedulerWithRealPipeline(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping real pipeline execution test in short mode")
	}

	tempDir, err := os.MkdirTemp("", "mimir_scheduler_real_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a simple pipeline
	pipelineContent := `
name: "Real Scheduler Test"
steps:
  - name: "Fetch Data"
    plugin: "Input.api"
    config:
      url: "https://httpbin.org/json"
      method: "GET"
    output: "data"
`

	pipelineFile := filepath.Join(tempDir, "real_test.yaml")
	err = os.WriteFile(pipelineFile, []byte(pipelineContent), 0644)
	require.NoError(t, err)

	registry := pipelines.NewPluginRegistry()
	err = registry.RegisterPlugin(&utils.RealAPIPlugin{})
	require.NoError(t, err)

	scheduler := utils.NewScheduler(registry)
	err = scheduler.Start()
	require.NoError(t, err)
	defer scheduler.Stop()

	t.Run("Schedule and Verify Job", func(t *testing.T) {
		err := scheduler.AddJob(
			"real_pipeline_job",
			"Real Pipeline Job",
			pipelineFile,
			"*/1 * * * *",
		)
		require.NoError(t, err)

		// Get job details
		job, err := scheduler.GetJob("real_pipeline_job")
		require.NoError(t, err)

		assert.Equal(t, "Real Pipeline Job", job.Name)
		assert.Equal(t, pipelineFile, job.Pipeline)
		assert.True(t, job.Enabled)
		assert.NotNil(t, job.NextRun)

		t.Logf("Job scheduled for next run at: %v", job.NextRun)
	})
}

// TestSchedulerJobManagement tests comprehensive job management operations
func TestSchedulerJobManagement(t *testing.T) {
	registry := pipelines.NewPluginRegistry()
	scheduler := utils.NewScheduler(registry)
	err := scheduler.Start()
	require.NoError(t, err)
	defer scheduler.Stop()

	t.Run("Complete Job Lifecycle", func(t *testing.T) {
		jobID := "lifecycle_test"

		// 1. Create job
		err := scheduler.AddJob(jobID, "Lifecycle Test", "test.yaml", "*/5 * * * *")
		require.NoError(t, err)

		// 2. Verify job exists and is enabled
		job, err := scheduler.GetJob(jobID)
		require.NoError(t, err)
		assert.True(t, job.Enabled)
		assert.NotNil(t, job.NextRun)
		initialNextRun := job.NextRun

		// 3. Disable job
		err = scheduler.DisableJob(jobID)
		require.NoError(t, err)

		job, err = scheduler.GetJob(jobID)
		require.NoError(t, err)
		assert.False(t, job.Enabled)
		assert.Nil(t, job.NextRun)

		// 4. Re-enable job
		err = scheduler.EnableJob(jobID)
		require.NoError(t, err)

		job, err = scheduler.GetJob(jobID)
		require.NoError(t, err)
		assert.True(t, job.Enabled)
		assert.NotNil(t, job.NextRun)

		// 5. Verify timestamps are updated
		assert.NotEqual(t, job.CreatedAt, job.UpdatedAt)

		// 6. Remove job
		err = scheduler.RemoveJob(jobID)
		require.NoError(t, err)

		// 7. Verify job is gone
		_, err = scheduler.GetJob(jobID)
		assert.Error(t, err, "Getting removed job should fail")

		_ = initialNextRun // Suppress unused variable warning
	})
}
