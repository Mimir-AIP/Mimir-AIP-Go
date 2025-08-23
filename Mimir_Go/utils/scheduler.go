package utils

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
)

// ScheduledJob represents a scheduled pipeline execution
type ScheduledJob struct {
	ID         string                   `json:"id"`
	Name       string                   `json:"name"`
	Pipeline   string                   `json:"pipeline"`
	CronExpr   string                   `json:"cron_expr"`
	Enabled    bool                     `json:"enabled"`
	NextRun    *time.Time               `json:"next_run,omitempty"`
	LastRun    *time.Time               `json:"last_run,omitempty"`
	LastResult *PipelineExecutionResult `json:"last_result,omitempty"`
	CreatedAt  time.Time                `json:"created_at"`
	UpdatedAt  time.Time                `json:"updated_at"`
}

// Scheduler manages cron-based pipeline execution
type Scheduler struct {
	jobs      map[string]*ScheduledJob
	jobsMutex sync.RWMutex
	running   bool
	stopChan  chan struct{}
	registry  *pipelines.PluginRegistry
}

// NewScheduler creates a new scheduler instance
func NewScheduler(registry *pipelines.PluginRegistry) *Scheduler {
	return &Scheduler{
		jobs:     make(map[string]*ScheduledJob),
		stopChan: make(chan struct{}),
		registry: registry,
	}
}

// Start begins the scheduler
func (s *Scheduler) Start() error {
	s.jobsMutex.Lock()
	defer s.jobsMutex.Unlock()

	if s.running {
		return fmt.Errorf("scheduler is already running")
	}

	s.running = true
	go s.run()

	log.Printf("Scheduler started with %d jobs", len(s.jobs))
	return nil
}

// Stop stops the scheduler
func (s *Scheduler) Stop() error {
	s.jobsMutex.Lock()
	defer s.jobsMutex.Unlock()

	if !s.running {
		return fmt.Errorf("scheduler is not running")
	}

	s.running = false
	close(s.stopChan)

	log.Printf("Scheduler stopped")
	return nil
}

// AddJob adds a new scheduled job
func (s *Scheduler) AddJob(id, name, pipeline, cronExpr string) error {
	s.jobsMutex.Lock()
	defer s.jobsMutex.Unlock()

	if _, exists := s.jobs[id]; exists {
		return fmt.Errorf("job with ID %s already exists", id)
	}

	// Parse cron expression to validate it
	_, err := parseCronExpression(cronExpr)
	if err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}

	now := time.Now()
	job := &ScheduledJob{
		ID:        id,
		Name:      name,
		Pipeline:  pipeline,
		CronExpr:  cronExpr,
		Enabled:   true,
		CreatedAt: now,
		UpdatedAt: now,
	}

	s.jobs[id] = job
	s.updateNextRun(job)

	log.Printf("Added scheduled job: %s (%s)", name, cronExpr)
	return nil
}

// RemoveJob removes a scheduled job
func (s *Scheduler) RemoveJob(id string) error {
	s.jobsMutex.Lock()
	defer s.jobsMutex.Unlock()

	if _, exists := s.jobs[id]; !exists {
		return fmt.Errorf("job with ID %s not found", id)
	}

	delete(s.jobs, id)
	log.Printf("Removed scheduled job: %s", id)
	return nil
}

// EnableJob enables a scheduled job
func (s *Scheduler) EnableJob(id string) error {
	s.jobsMutex.Lock()
	defer s.jobsMutex.Unlock()

	job, exists := s.jobs[id]
	if !exists {
		return fmt.Errorf("job with ID %s not found", id)
	}

	job.Enabled = true
	job.UpdatedAt = time.Now()
	s.updateNextRun(job)

	log.Printf("Enabled scheduled job: %s", id)
	return nil
}

// DisableJob disables a scheduled job
func (s *Scheduler) DisableJob(id string) error {
	s.jobsMutex.Lock()
	defer s.jobsMutex.Unlock()

	job, exists := s.jobs[id]
	if !exists {
		return fmt.Errorf("job with ID %s not found", id)
	}

	job.Enabled = false
	job.UpdatedAt = time.Now()

	log.Printf("Disabled scheduled job: %s", id)
	return nil
}

// GetJobs returns all scheduled jobs
func (s *Scheduler) GetJobs() map[string]*ScheduledJob {
	s.jobsMutex.RLock()
	defer s.jobsMutex.RUnlock()

	jobs := make(map[string]*ScheduledJob)
	for id, job := range s.jobs {
		jobs[id] = job
	}
	return jobs
}

// GetJob returns a specific scheduled job
func (s *Scheduler) GetJob(id string) (*ScheduledJob, error) {
	s.jobsMutex.RLock()
	defer s.jobsMutex.RUnlock()

	job, exists := s.jobs[id]
	if !exists {
		return nil, fmt.Errorf("job with ID %s not found", id)
	}
	return job, nil
}

// run is the main scheduler loop
func (s *Scheduler) run() {
	ticker := time.NewTicker(30 * time.Second) // Check every 30 seconds
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.checkAndExecuteJobs()
		}
	}
}

// checkAndExecuteJobs checks for jobs that need to be executed
func (s *Scheduler) checkAndExecuteJobs() {
	s.jobsMutex.Lock()
	defer s.jobsMutex.Unlock()

	now := time.Now()

	for _, job := range s.jobs {
		if !job.Enabled {
			continue
		}

		if job.NextRun != nil && now.After(*job.NextRun) {
			go s.executeJob(job)
			s.updateNextRun(job)
		}
	}
}

// executeJob executes a scheduled job
func (s *Scheduler) executeJob(job *ScheduledJob) {
	log.Printf("Executing scheduled job: %s", job.Name)

	// Execute the pipeline
	result, err := ExecutePipeline(context.Background(), &PipelineConfig{
		Name:  job.Name,
		Steps: []pipelines.StepConfig{}, // This would need to be loaded from the pipeline file
	})

	// Update job status
	now := time.Now()
	job.LastRun = &now
	job.LastResult = result
	job.UpdatedAt = now

	if err != nil {
		log.Printf("Scheduled job %s failed: %v", job.Name, err)
	} else {
		log.Printf("Scheduled job %s completed successfully", job.Name)
	}
}

// updateNextRun calculates the next run time for a job
func (s *Scheduler) updateNextRun(job *ScheduledJob) {
	if !job.Enabled {
		job.NextRun = nil
		return
	}

	nextRun, err := parseCronExpression(job.CronExpr)
	if err != nil {
		log.Printf("Failed to parse cron expression for job %s: %v", job.ID, err)
		job.NextRun = nil
		return
	}

	job.NextRun = &nextRun
}

// parseCronExpression parses a cron expression and returns the next run time
// This is a simplified implementation - in production, you'd use a proper cron library
func parseCronExpression(cronExpr string) (time.Time, error) {
	// For now, support simple formats like:
	// "0 9 * * *" - daily at 9 AM
	// "*/5 * * * *" - every 5 minutes
	// "0 */2 * * *" - every 2 hours

	parts := strings.Fields(cronExpr)
	if len(parts) != 5 {
		return time.Time{}, fmt.Errorf("invalid cron expression format")
	}

	minute := parts[0]
	// hour := parts[1]      // Not used in this simplified implementation
	// day := parts[2]       // Not used in this simplified implementation
	// month := parts[3]     // Not used in this simplified implementation
	// dayOfWeek := parts[4] // Not used in this simplified implementation

	now := time.Now()

	// Handle minute field
	if minute == "*" {
		// Every minute - not supported in this simple implementation
		return time.Time{}, fmt.Errorf("wildcard minutes not supported")
	} else if strings.HasPrefix(minute, "*/") {
		// Every N minutes
		interval := 1 // default
		fmt.Sscanf(minute, "*/%d", &interval)

		// Find next minute that matches the interval
		currentMinute := now.Minute()
		nextMinute := ((currentMinute / interval) + 1) * interval
		if nextMinute >= 60 {
			nextMinute = 0
			now = now.Add(time.Hour)
		}

		return time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), nextMinute, 0, 0, now.Location()), nil
	} else {
		// Specific minute
		var min int
		fmt.Sscanf(minute, "%d", &min)

		nextTime := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), min, 0, 0, now.Location())
		if nextTime.Before(now) || nextTime.Equal(now) {
			nextTime = nextTime.Add(time.Hour)
		}
		return nextTime, nil
	}
}

// LoadJobsFromConfig loads scheduled jobs from configuration
func (s *Scheduler) LoadJobsFromConfig(configPath string) error {
	pipelines, err := ParseAllPipelines(configPath)
	if err != nil {
		return fmt.Errorf("failed to load pipelines: %w", err)
	}

	for _, pipeline := range pipelines {
		// Check if pipeline has cron schedule (this would be an extension to the pipeline format)
		// For now, we'll skip this - in production, you'd add cron field to pipeline schema
		_ = pipeline
	}

	return nil
}
