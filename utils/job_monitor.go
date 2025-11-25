package utils

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
)

// JobExecutionRecord represents a single job execution
type JobExecutionRecord struct {
	ID          string                  `json:"id"`
	JobID       string                  `json:"job_id"`
	Pipeline    string                  `json:"pipeline"`
	StartTime   time.Time               `json:"start_time"`
	EndTime     *time.Time              `json:"end_time,omitempty"`
	Duration    *time.Duration          `json:"duration,omitempty"`
	Status      string                  `json:"status"` // "running", "success", "failed", "cancelled"
	Error       string                  `json:"error,omitempty"`
	Context     pipelines.PluginContext `json:"context,omitempty"`
	Steps       []StepExecutionRecord   `json:"steps,omitempty"`
	TriggeredBy string                  `json:"triggered_by"` // "scheduler", "api", "manual"
}

// StepExecutionRecord represents execution of a single pipeline step
type StepExecutionRecord struct {
	StepName  string                  `json:"step_name"`
	Plugin    string                  `json:"plugin"`
	StartTime time.Time               `json:"start_time"`
	EndTime   *time.Time              `json:"end_time,omitempty"`
	Duration  *time.Duration          `json:"duration,omitempty"`
	Status    string                  `json:"status"`
	Error     string                  `json:"error,omitempty"`
	Input     pipelines.PluginContext `json:"input,omitempty"`
	Output    pipelines.PluginContext `json:"output,omitempty"`
}

// JobStatistics represents aggregated job statistics
type JobStatistics struct {
	TotalJobs       int64         `json:"total_jobs"`
	RunningJobs     int64         `json:"running_jobs"`
	SuccessfulJobs  int64         `json:"successful_jobs"`
	FailedJobs      int64         `json:"failed_jobs"`
	AverageDuration time.Duration `json:"average_duration"`
	SuccessRate     float64       `json:"success_rate"`
	LastUpdated     time.Time     `json:"last_updated"`
}

// JobMonitor manages job execution tracking and monitoring
type JobMonitor struct {
	executions map[string]*JobExecutionRecord
	stats      JobStatistics
	mutex      sync.RWMutex
	maxHistory int // Maximum number of records to keep
}

// NewJobMonitor creates a new job monitor
func NewJobMonitor(maxHistory int) *JobMonitor {
	if maxHistory <= 0 {
		maxHistory = 1000 // Default
	}

	return &JobMonitor{
		executions: make(map[string]*JobExecutionRecord),
		maxHistory: maxHistory,
		stats: JobStatistics{
			LastUpdated: time.Now(),
		},
	}
}

// StartJob starts tracking a new job execution
func (jm *JobMonitor) StartJob(jobID, pipeline, triggeredBy string) string {
	jm.mutex.Lock()
	defer jm.mutex.Unlock()

	executionID := fmt.Sprintf("%s_%d", jobID, time.Now().UnixNano())

	record := &JobExecutionRecord{
		ID:          executionID,
		JobID:       jobID,
		Pipeline:    pipeline,
		StartTime:   time.Now(),
		Status:      "running",
		TriggeredBy: triggeredBy,
		Steps:       []StepExecutionRecord{},
	}

	jm.executions[executionID] = record
	jm.updateStats()

	log.Printf("Started job execution: %s (ID: %s)", jobID, executionID)
	return executionID
}

// StartStep starts tracking a pipeline step execution
func (jm *JobMonitor) StartStep(executionID, stepName, plugin string, input pipelines.PluginContext) {
	jm.mutex.Lock()
	defer jm.mutex.Unlock()

	record, exists := jm.executions[executionID]
	if !exists {
		log.Printf("Warning: execution %s not found for step %s", executionID, stepName)
		return
	}

	stepRecord := StepExecutionRecord{
		StepName:  stepName,
		Plugin:    plugin,
		StartTime: time.Now(),
		Status:    "running",
		Input:     input,
	}

	record.Steps = append(record.Steps, stepRecord)
}

// CompleteStep completes tracking of a pipeline step execution
func (jm *JobMonitor) CompleteStep(executionID, stepName string, success bool, err error, output pipelines.PluginContext) {
	jm.mutex.Lock()
	defer jm.mutex.Unlock()

	record, exists := jm.executions[executionID]
	if !exists {
		return
	}

	// Find the step record
	for i := len(record.Steps) - 1; i >= 0; i-- {
		if record.Steps[i].StepName == stepName && record.Steps[i].Status == "running" {
			now := time.Now()
			duration := now.Sub(record.Steps[i].StartTime)

			record.Steps[i].EndTime = &now
			record.Steps[i].Duration = &duration
			record.Steps[i].Output = output

			if success {
				record.Steps[i].Status = "success"
			} else {
				record.Steps[i].Status = "failed"
				if err != nil {
					record.Steps[i].Error = err.Error()
				}
			}
			break
		}
	}
}

// CompleteJob completes tracking of a job execution
func (jm *JobMonitor) CompleteJob(executionID string, success bool, err error, context pipelines.PluginContext) {
	jm.mutex.Lock()
	defer jm.mutex.Unlock()

	record, exists := jm.executions[executionID]
	if !exists {
		return
	}

	now := time.Now()
	duration := now.Sub(record.StartTime)

	record.EndTime = &now
	record.Duration = &duration
	record.Context = context

	if success {
		record.Status = "success"
		jm.stats.SuccessfulJobs++
	} else {
		record.Status = "failed"
		if err != nil {
			record.Error = err.Error()
		}
		jm.stats.FailedJobs++
	}

	jm.updateStats()
	jm.cleanupOldRecords()

	log.Printf("Completed job execution: %s (Status: %s, Duration: %v)", record.JobID, record.Status, duration)
}

// CancelJob marks a job as cancelled
func (jm *JobMonitor) CancelJob(executionID string) {
	jm.mutex.Lock()
	defer jm.mutex.Unlock()

	record, exists := jm.executions[executionID]
	if !exists {
		return
	}

	now := time.Now()
	duration := now.Sub(record.StartTime)

	record.EndTime = &now
	record.Duration = &duration
	record.Status = "cancelled"

	jm.updateStats()

	log.Printf("Cancelled job execution: %s", record.JobID)
}

// GetExecution returns a specific execution record
func (jm *JobMonitor) GetExecution(executionID string) (*JobExecutionRecord, error) {
	jm.mutex.RLock()
	defer jm.mutex.RUnlock()

	record, exists := jm.executions[executionID]
	if !exists {
		return nil, fmt.Errorf("execution not found: %s", executionID)
	}

	// Return a copy to prevent external modifications
	recordCopy := *record
	return &recordCopy, nil
}

// GetAllExecutions returns all execution records
func (jm *JobMonitor) GetAllExecutions() map[string]*JobExecutionRecord {
	jm.mutex.RLock()
	defer jm.mutex.RUnlock()

	executions := make(map[string]*JobExecutionRecord)
	for id, record := range jm.executions {
		// Return copies to prevent external modifications
		recordCopy := *record
		executions[id] = &recordCopy
	}

	return executions
}

// GetRunningExecutions returns currently running executions
func (jm *JobMonitor) GetRunningExecutions() map[string]*JobExecutionRecord {
	jm.mutex.RLock()
	defer jm.mutex.RUnlock()

	running := make(map[string]*JobExecutionRecord)
	for id, record := range jm.executions {
		if record.Status == "running" {
			// Return copy
			recordCopy := *record
			running[id] = &recordCopy
		}
	}

	return running
}

// GetJobExecutions returns executions for a specific job
func (jm *JobMonitor) GetJobExecutions(jobID string) []*JobExecutionRecord {
	jm.mutex.RLock()
	defer jm.mutex.RUnlock()

	var executions []*JobExecutionRecord
	for _, record := range jm.executions {
		if record.JobID == jobID {
			// Return copy
			recordCopy := *record
			executions = append(executions, &recordCopy)
		}
	}

	return executions
}

// GetStatistics returns current job statistics
func (jm *JobMonitor) GetStatistics() JobStatistics {
	jm.mutex.RLock()
	defer jm.mutex.RUnlock()

	// Return a copy with atomic loads for thread safety
	return JobStatistics{
		TotalJobs:       atomic.LoadInt64(&jm.stats.TotalJobs),
		RunningJobs:     atomic.LoadInt64(&jm.stats.RunningJobs),
		SuccessfulJobs:  atomic.LoadInt64(&jm.stats.SuccessfulJobs),
		FailedJobs:      atomic.LoadInt64(&jm.stats.FailedJobs),
		AverageDuration: jm.stats.AverageDuration,
		SuccessRate:     jm.stats.SuccessRate,
		LastUpdated:     jm.stats.LastUpdated,
	}
}

// updateStats recalculates job statistics
func (jm *JobMonitor) updateStats() {
	atomic.StoreInt64(&jm.stats.TotalJobs, int64(len(jm.executions)))
	atomic.StoreInt64(&jm.stats.RunningJobs, 0)
	atomic.StoreInt64(&jm.stats.SuccessfulJobs, 0)
	atomic.StoreInt64(&jm.stats.FailedJobs, 0)

	var totalDuration time.Duration
	var completedCount int

	for _, record := range jm.executions {
		switch record.Status {
		case "running":
			atomic.AddInt64(&jm.stats.RunningJobs, 1)
		case "success":
			atomic.AddInt64(&jm.stats.SuccessfulJobs, 1)
		case "failed":
			atomic.AddInt64(&jm.stats.FailedJobs, 1)
		}

		if record.Status != "running" && record.Duration != nil {
			totalDuration += *record.Duration
			completedCount++
		}
	}

	if completedCount > 0 {
		jm.stats.AverageDuration = totalDuration / time.Duration(completedCount)
	}

	successful := atomic.LoadInt64(&jm.stats.SuccessfulJobs)
	failed := atomic.LoadInt64(&jm.stats.FailedJobs)
	totalCompleted := successful + failed
	if totalCompleted > 0 {
		jm.stats.SuccessRate = float64(successful) / float64(totalCompleted)
	}

	jm.stats.LastUpdated = time.Now()
}

// cleanupOldRecords removes old execution records to prevent memory leaks
func (jm *JobMonitor) cleanupOldRecords() {
	if len(jm.executions) <= jm.maxHistory {
		return
	}

	// Remove oldest completed records first
	var toRemove []string
	completedCount := 0

	for id, record := range jm.executions {
		if record.Status != "running" {
			toRemove = append(toRemove, id)
			completedCount++
			if completedCount >= len(jm.executions)-jm.maxHistory {
				break
			}
		}
	}

	for _, id := range toRemove {
		delete(jm.executions, id)
	}

	log.Printf("Cleaned up %d old execution records", len(toRemove))
}

// ExportToJSON exports all execution data to JSON
func (jm *JobMonitor) ExportToJSON() ([]byte, error) {
	jm.mutex.RLock()
	defer jm.mutex.RUnlock()

	data := map[string]interface{}{
		"executions":  jm.executions,
		"statistics":  jm.stats,
		"exported_at": time.Now(),
	}

	return json.MarshalIndent(data, "", "  ")
}

// GetRecentExecutions returns the most recent executions
func (jm *JobMonitor) GetRecentExecutions(limit int) []*JobExecutionRecord {
	jm.mutex.RLock()
	defer jm.mutex.RUnlock()

	var executions []*JobExecutionRecord
	for _, record := range jm.executions {
		// Return copy
		recordCopy := *record
		executions = append(executions, &recordCopy)
	}

	// Sort by start time (most recent first) - simplified approach
	if len(executions) > limit {
		executions = executions[len(executions)-limit:]
	}

	return executions
}
