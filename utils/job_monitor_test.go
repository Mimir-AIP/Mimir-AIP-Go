package utils

import (
	"fmt"
	"testing"
	"time"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
	"github.com/stretchr/testify/assert"
)

func TestNewJobMonitor(t *testing.T) {
	tests := []struct {
		name       string
		maxHistory int
		expect     int
	}{
		{
			name:       "default max history",
			maxHistory: 0,
			expect:     1000,
		},
		{
			name:       "custom max history",
			maxHistory: 500,
			expect:     500,
		},
		{
			name:       "negative max history",
			maxHistory: -100,
			expect:     1000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			monitor := NewJobMonitor(tt.maxHistory)
			assert.NotNil(t, monitor)
			assert.Equal(t, tt.expect, monitor.maxHistory)
			assert.NotNil(t, monitor.executions)
			assert.Equal(t, int64(0), monitor.stats.TotalJobs)
			assert.False(t, monitor.stats.LastUpdated.IsZero())
		})
	}
}

func TestJobMonitorStartJob(t *testing.T) {
	monitor := NewJobMonitor(100)

	executionID := monitor.StartJob("job1", "test-pipeline", "scheduler")

	assert.NotEmpty(t, executionID)
	assert.Contains(t, executionID, "job1")

	// Verify execution record
	record, err := monitor.GetExecution(executionID)
	assert.NoError(t, err)
	assert.Equal(t, executionID, record.ID)
	assert.Equal(t, "job1", record.JobID)
	assert.Equal(t, "test-pipeline", record.Pipeline)
	assert.Equal(t, "scheduler", record.TriggeredBy)
	assert.Equal(t, "running", record.Status)
	assert.False(t, record.StartTime.IsZero())
	assert.Nil(t, record.EndTime)
	assert.Nil(t, record.Duration)
}

func TestJobMonitorStartStep(t *testing.T) {
	monitor := NewJobMonitor(100)

	executionID := monitor.StartJob("job1", "test-pipeline", "scheduler")
	inputContext := pipelines.NewPluginContext()
	inputContext.Set("test_input", "test_value")

	monitor.StartStep(executionID, "step1", "Input.test", inputContext)

	// Verify step record
	record, err := monitor.GetExecution(executionID)
	assert.NoError(t, err)
	assert.Len(t, record.Steps, 1)

	step := record.Steps[0]
	assert.Equal(t, "step1", step.StepName)
	assert.Equal(t, "Input.test", step.Plugin)
	assert.Equal(t, "running", step.Status)
	assert.False(t, step.StartTime.IsZero())
	assert.Nil(t, step.EndTime)
	assert.Nil(t, step.Duration)

	// Test with non-existent execution ID
	monitor.StartStep("nonexistent", "step2", "Output.test", inputContext)
	// Should not panic, just log a warning
}

func TestJobMonitorCompleteStep(t *testing.T) {
	monitor := NewJobMonitor(100)

	executionID := monitor.StartJob("job1", "test-pipeline", "scheduler")
	inputContext := pipelines.NewPluginContext()
	outputContext := pipelines.NewPluginContext()
	outputContext.Set("test_output", "output_value")

	// Start and complete a step successfully
	monitor.StartStep(executionID, "step1", "Input.test", inputContext)
	monitor.CompleteStep(executionID, "step1", true, nil, outputContext)

	// Verify step completion
	record, err := monitor.GetExecution(executionID)
	assert.NoError(t, err)
	assert.Len(t, record.Steps, 1)

	step := record.Steps[0]
	assert.Equal(t, "success", step.Status)
	assert.NotNil(t, step.EndTime)
	assert.NotNil(t, step.Duration)
	assert.True(t, step.Duration.Nanoseconds() > 0)

	// Test step completion with error
	monitor.StartStep(executionID, "step2", "Output.test", inputContext)
	testErr := assert.AnError
	monitor.CompleteStep(executionID, "step2", false, testErr, outputContext)

	record, err = monitor.GetExecution(executionID)
	assert.NoError(t, err)
	assert.Len(t, record.Steps, 2)

	step2 := record.Steps[1]
	assert.Equal(t, "failed", step2.Status)
	assert.Equal(t, testErr.Error(), step2.Error)

	// Test completing non-existent step
	monitor.CompleteStep(executionID, "nonexistent", true, nil, outputContext)
	// Should not panic

	// Test with non-existent execution
	monitor.CompleteStep("nonexistent", "step1", true, nil, outputContext)
	// Should not panic
}

func TestJobMonitorCompleteJob(t *testing.T) {
	monitor := NewJobMonitor(100)

	executionID := monitor.StartJob("job1", "test-pipeline", "scheduler")
	finalContext := pipelines.NewPluginContext()
	finalContext.Set("final_result", "success")

	// Complete job successfully
	monitor.CompleteJob(executionID, true, nil, finalContext)

	// Verify job completion
	record, err := monitor.GetExecution(executionID)
	assert.NoError(t, err)
	assert.Equal(t, "success", record.Status)
	assert.NotNil(t, record.EndTime)
	assert.NotNil(t, record.Duration)
	assert.True(t, record.Duration.Nanoseconds() > 0)

	// Test job completion with error
	executionID2 := monitor.StartJob("job2", "test-pipeline", "api")
	testErr := assert.AnError
	monitor.CompleteJob(executionID2, false, testErr, finalContext)

	record2, err := monitor.GetExecution(executionID2)
	assert.NoError(t, err)
	assert.Equal(t, "failed", record2.Status)
	assert.Equal(t, testErr.Error(), record2.Error)

	// Test with non-existent execution
	monitor.CompleteJob("nonexistent", true, nil, finalContext)
	// Should not panic
}

func TestJobMonitorCancelJob(t *testing.T) {
	monitor := NewJobMonitor(100)

	executionID := monitor.StartJob("job1", "test-pipeline", "scheduler")

	// Cancel job
	monitor.CancelJob(executionID)

	// Verify job cancellation
	record, err := monitor.GetExecution(executionID)
	assert.NoError(t, err)
	assert.Equal(t, "cancelled", record.Status)
	assert.NotNil(t, record.EndTime)
	assert.NotNil(t, record.Duration)

	// Test with non-existent execution
	monitor.CancelJob("nonexistent")
	// Should not panic
}

func TestJobMonitorGetExecution(t *testing.T) {
	monitor := NewJobMonitor(100)

	// Test non-existent execution
	_, err := monitor.GetExecution("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "execution not found")

	// Test existing execution
	executionID := monitor.StartJob("job1", "test-pipeline", "scheduler")
	record, err := monitor.GetExecution(executionID)
	assert.NoError(t, err)
	assert.Equal(t, executionID, record.ID)

	// Verify returned record is a copy (modifying shouldn't affect original)
	record.Status = "modified"
	originalRecord, err := monitor.GetExecution(executionID)
	assert.NoError(t, err)
	assert.NotEqual(t, "modified", originalRecord.Status)
}

func TestJobMonitorGetAllExecutions(t *testing.T) {
	monitor := NewJobMonitor(100)

	// Initially empty
	executions := monitor.GetAllExecutions()
	assert.Empty(t, executions)

	// Add some executions
	id1 := monitor.StartJob("job1", "pipeline1", "scheduler")
	id2 := monitor.StartJob("job2", "pipeline2", "api")

	executions = monitor.GetAllExecutions()
	assert.Len(t, executions, 2)
	assert.Contains(t, executions, id1)
	assert.Contains(t, executions, id2)

	// Verify returned executions are copies
	for _, record := range executions {
		record.Status = "modified"
	}

	// Original should be unchanged
	originalRecord, err := monitor.GetExecution(id1)
	assert.NoError(t, err)
	assert.NotEqual(t, "modified", originalRecord.Status)
}

func TestJobMonitorGetRunningExecutions(t *testing.T) {
	monitor := NewJobMonitor(100)

	// Initially empty
	running := monitor.GetRunningExecutions()
	assert.Empty(t, running)

	// Add running executions
	id1 := monitor.StartJob("job1", "pipeline1", "scheduler")
	id2 := monitor.StartJob("job2", "pipeline2", "api")

	running = monitor.GetRunningExecutions()
	assert.Len(t, running, 2)
	assert.Contains(t, running, id1)
	assert.Contains(t, running, id2)

	// Complete one execution
	monitor.CompleteJob(id1, true, nil, pipelines.NewPluginContext())

	running = monitor.GetRunningExecutions()
	assert.Len(t, running, 1)
	assert.Contains(t, running, id2)
	assert.NotContains(t, running, id1)
}

func TestJobMonitorGetJobExecutions(t *testing.T) {
	monitor := NewJobMonitor(100)

	// Initially empty
	executions := monitor.GetJobExecutions("job1")
	assert.Empty(t, executions)

	// Add executions for different jobs
	id1 := monitor.StartJob("job1", "pipeline1", "scheduler")
	id2 := monitor.StartJob("job1", "pipeline1", "api")
	id3 := monitor.StartJob("job2", "pipeline2", "manual")

	// Get executions for job1
	job1Executions := monitor.GetJobExecutions("job1")
	assert.Len(t, job1Executions, 2)

	executionIDs := make([]string, 0, 2)
	for _, record := range job1Executions {
		executionIDs = append(executionIDs, record.ID)
	}
	assert.Contains(t, executionIDs, id1)
	assert.Contains(t, executionIDs, id2)

	// Get executions for job2
	job2Executions := monitor.GetJobExecutions("job2")
	assert.Len(t, job2Executions, 1)
	assert.Equal(t, id3, job2Executions[0].ID)

	// Get executions for non-existent job
	nonExistentExecutions := monitor.GetJobExecutions("nonexistent")
	assert.Empty(t, nonExistentExecutions)
}

func TestJobMonitorGetStatistics(t *testing.T) {
	monitor := NewJobMonitor(100)

	// Initial statistics
	stats := monitor.GetStatistics()
	assert.Equal(t, int64(0), stats.TotalJobs)
	assert.Equal(t, int64(0), stats.RunningJobs)
	assert.Equal(t, int64(0), stats.SuccessfulJobs)
	assert.Equal(t, int64(0), stats.FailedJobs)
	assert.Equal(t, float64(0), stats.SuccessRate)
	assert.Equal(t, time.Duration(0), stats.AverageDuration)
	assert.False(t, stats.LastUpdated.IsZero())

	// Add some executions
	id1 := monitor.StartJob("job1", "pipeline1", "scheduler")
	id2 := monitor.StartJob("job2", "pipeline2", "api")
	monitor.StartJob("job3", "pipeline3", "manual")

	// Complete some jobs
	monitor.CompleteJob(id1, true, nil, pipelines.NewPluginContext())
	monitor.CompleteJob(id2, false, assert.AnError, pipelines.NewPluginContext())
	// one job remains running

	stats = monitor.GetStatistics()
	assert.Equal(t, int64(3), stats.TotalJobs)
	assert.Equal(t, int64(1), stats.RunningJobs)
	assert.Equal(t, int64(1), stats.SuccessfulJobs)
	assert.Equal(t, int64(1), stats.FailedJobs)
	assert.Equal(t, float64(0.5), stats.SuccessRate)
	assert.True(t, stats.AverageDuration > 0)
}

func TestJobMonitorCleanupOldRecords(t *testing.T) {
	monitor := NewJobMonitor(2) // Small limit to trigger cleanup

	// Add more executions than the limit
	id1 := monitor.StartJob("job1", "pipeline1", "scheduler")
	id2 := monitor.StartJob("job2", "pipeline2", "api")
	id3 := monitor.StartJob("job3", "pipeline3", "manual")

	// Complete first two jobs
	monitor.CompleteJob(id1, true, nil, pipelines.NewPluginContext())
	monitor.CompleteJob(id2, true, nil, pipelines.NewPluginContext())

	// Add another job to trigger cleanup
	id4 := monitor.StartJob("job4", "pipeline4", "scheduler")
	_ = id3 // Use id3 to avoid unused variable error

	// Check that old completed records were cleaned up
	executions := monitor.GetAllExecutions()
	assert.Len(t, executions, 3) // id3 (running), id4 (running), and one of id1/id2

	// Verify running jobs are not cleaned up
	assert.Contains(t, executions, id3)
	assert.Contains(t, executions, id4)
}

func TestJobMonitorExportToJSON(t *testing.T) {
	monitor := NewJobMonitor(100)

	// Add some data
	id1 := monitor.StartJob("job1", "pipeline1", "scheduler")
	monitor.StartStep(id1, "step1", "Input.test", pipelines.NewPluginContext())
	monitor.CompleteStep(id1, "step1", true, nil, pipelines.NewPluginContext())
	monitor.CompleteJob(id1, true, nil, pipelines.NewPluginContext())

	// Export to JSON
	data, err := monitor.ExportToJSON()
	assert.NoError(t, err)
	assert.NotEmpty(t, data)

	// Verify JSON structure (basic check)
	assert.Contains(t, string(data), "executions")
	assert.Contains(t, string(data), "statistics")
	assert.Contains(t, string(data), "exported_at")
}

func TestJobMonitorGetRecentExecutions(t *testing.T) {
	monitor := NewJobMonitor(100)

	// Initially empty
	recent := monitor.GetRecentExecutions(10)
	assert.Empty(t, recent)

	// Add some executions with delays
	monitor.StartJob("job1", "pipeline1", "scheduler")
	time.Sleep(10 * time.Millisecond)
	monitor.StartJob("job2", "pipeline2", "api")
	time.Sleep(10 * time.Millisecond)
	monitor.StartJob("job3", "pipeline3", "manual")

	// Get recent executions
	recent = monitor.GetRecentExecutions(2)
	assert.Len(t, recent, 2)

	// Note: The current implementation doesn't sort, it just takes the last N
	// In a real implementation, you'd want to sort by start time
}

func TestJobMonitorConcurrentAccess(t *testing.T) {
	monitor := NewJobMonitor(100)

	// Test concurrent job operations
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			jobID := fmt.Sprintf("job%d", id)
			executionID := monitor.StartJob(jobID, "pipeline", "scheduler")

			monitor.StartStep(executionID, "step1", "Input.test", pipelines.NewPluginContext())
			monitor.CompleteStep(executionID, "step1", true, nil, pipelines.NewPluginContext())
			monitor.CompleteJob(executionID, true, nil, pipelines.NewPluginContext())

			done <- true
		}(i)
	}

	// Wait for all operations to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all jobs were recorded
	stats := monitor.GetStatistics()
	assert.Equal(t, int64(10), stats.TotalJobs)
	assert.Equal(t, int64(10), stats.SuccessfulJobs)
	assert.Equal(t, int64(0), stats.RunningJobs)
}

func TestJobStatisticsAtomicOperations(t *testing.T) {
	monitor := NewJobMonitor(100)

	// Test that statistics are properly updated atomically
	id1 := monitor.StartJob("job1", "pipeline1", "scheduler")
	id2 := monitor.StartJob("job2", "pipeline2", "api")

	// Complete jobs concurrently
	go func() {
		monitor.CompleteJob(id1, true, nil, pipelines.NewPluginContext())
	}()
	go func() {
		monitor.CompleteJob(id2, false, assert.AnError, pipelines.NewPluginContext())
	}()

	// Wait a bit for completion
	time.Sleep(100 * time.Millisecond)

	stats := monitor.GetStatistics()
	assert.Equal(t, int64(2), stats.TotalJobs)
	assert.Equal(t, int64(1), stats.SuccessfulJobs)
	assert.Equal(t, int64(1), stats.FailedJobs)
	assert.Equal(t, float64(0.5), stats.SuccessRate)
}
