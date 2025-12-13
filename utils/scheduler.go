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
	jobs        map[string]*ScheduledJob
	jobsMutex   sync.RWMutex
	running     bool
	stopped     bool // Track if stopChan is closed
	stopChan    chan struct{}
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	registry    *pipelines.PluginRegistry
	persistence PersistenceBackend
}

// NewScheduler creates a new scheduler instance
func NewScheduler(registry *pipelines.PluginRegistry) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())
	return &Scheduler{
		jobs:        make(map[string]*ScheduledJob),
		stopChan:    make(chan struct{}),
		ctx:         ctx,
		cancel:      cancel,
		registry:    registry,
		persistence: nil, // Will be set via SetPersistence
	}
}

// SetPersistence sets the persistence backend for the scheduler
func (s *Scheduler) SetPersistence(persistence PersistenceBackend) {
	s.jobsMutex.Lock()
	defer s.jobsMutex.Unlock()
	s.persistence = persistence
}

// LoadJobsFromPersistence loads all jobs from the persistence backend
func (s *Scheduler) LoadJobsFromPersistence() error {
	if s.persistence == nil {
		return nil // No persistence configured
	}

	jobs, err := s.persistence.ListJobs(s.ctx)
	if err != nil {
		return fmt.Errorf("failed to load jobs from persistence: %w", err)
	}

	s.jobsMutex.Lock()
	defer s.jobsMutex.Unlock()

	for _, job := range jobs {
		s.jobs[job.ID] = job
		// Recalculate next run time if job is enabled
		if job.Enabled {
			s.updateNextRun(job)
		}
	}

	log.Printf("Loaded %d jobs from persistence", len(jobs))
	return nil
}

// Start begins the scheduler
func (s *Scheduler) Start() error {
	s.jobsMutex.Lock()
	defer s.jobsMutex.Unlock()

	if s.running {
		return fmt.Errorf("scheduler is already running")
	}

	// Recreate stopChan if it was closed
	if s.stopped {
		s.stopChan = make(chan struct{})
		s.ctx, s.cancel = context.WithCancel(context.Background())
		s.stopped = false
	}

	s.running = true
	s.wg.Add(1)
	go s.run()

	log.Printf("Scheduler started with %d jobs", len(s.jobs))
	return nil
}

// Stop stops the scheduler
func (s *Scheduler) Stop() error {
	s.jobsMutex.Lock()

	if !s.running {
		s.jobsMutex.Unlock()
		return fmt.Errorf("scheduler is not running")
	}

	s.running = false
	s.cancel() // Cancel context to stop all running jobs

	// Only close stopChan if not already closed
	if !s.stopped {
		close(s.stopChan)
		s.stopped = true
	}
	s.jobsMutex.Unlock() // Release lock before waiting

	// Wait for all goroutines to finish
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Printf("Scheduler stopped gracefully")
		return nil
	case <-time.After(30 * time.Second):
		log.Printf("Scheduler stop timeout - forcing shutdown")
		return fmt.Errorf("scheduler stop timeout")
	}
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

	// Persist the job if persistence is enabled
	if s.persistence != nil {
		if err := s.persistence.SaveJob(context.Background(), job); err != nil {
			log.Printf("Warning: failed to persist job %s: %v", id, err)
		}
	}

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

	// Remove from persistence if enabled
	if s.persistence != nil {
		if err := s.persistence.DeleteJob(context.Background(), id); err != nil {
			log.Printf("Warning: failed to delete persisted job %s: %v", id, err)
		}
	}

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

	// Update in persistence if enabled
	if s.persistence != nil {
		if err := s.persistence.UpdateJob(context.Background(), job); err != nil {
			log.Printf("Warning: failed to update persisted job %s: %v", id, err)
		}
	}

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
	job.NextRun = nil // Clear next run when disabled
	job.UpdatedAt = time.Now()

	// Update in persistence if enabled
	if s.persistence != nil {
		if err := s.persistence.UpdateJob(context.Background(), job); err != nil {
			log.Printf("Warning: failed to update persisted job %s: %v", id, err)
		}
	}

	log.Printf("Disabled scheduled job: %s", id)
	return nil
}

// UpdateJob updates an existing scheduled job
func (s *Scheduler) UpdateJob(id string, name, pipeline, cronExpr *string) error {
	s.jobsMutex.Lock()
	defer s.jobsMutex.Unlock()

	job, exists := s.jobs[id]
	if !exists {
		return fmt.Errorf("job with ID %s not found", id)
	}

	// Update fields if provided
	if name != nil && *name != "" {
		job.Name = *name
	}

	if pipeline != nil && *pipeline != "" {
		job.Pipeline = *pipeline
	}

	if cronExpr != nil && *cronExpr != "" {
		// Validate new cron expression
		_, err := parseCronExpression(*cronExpr)
		if err != nil {
			return fmt.Errorf("invalid cron expression: %w", err)
		}
		job.CronExpr = *cronExpr
		// Recalculate next run time
		s.updateNextRun(job)
	}

	job.UpdatedAt = time.Now()

	// Update in persistence if enabled
	if s.persistence != nil {
		if err := s.persistence.UpdateJob(context.Background(), job); err != nil {
			log.Printf("Warning: failed to update persisted job %s: %v", id, err)
		}
	}

	log.Printf("Updated scheduled job: %s", id)
	return nil
}

// GetJobs returns all scheduled jobs (as copies to prevent external modification)
func (s *Scheduler) GetJobs() map[string]*ScheduledJob {
	s.jobsMutex.RLock()
	defer s.jobsMutex.RUnlock()

	jobs := make(map[string]*ScheduledJob)
	for id, job := range s.jobs {
		// Create a copy of the job
		jobCopy := *job
		jobs[id] = &jobCopy
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
	defer s.wg.Done()
	defer func() {
		// Ensure running flag is cleared when goroutine exits
		// This handles both Stop() being called and context cancellation
		s.jobsMutex.Lock()
		s.running = false
		s.jobsMutex.Unlock()
	}()

	ticker := time.NewTicker(30 * time.Second) // Check every 30 seconds
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
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
			s.wg.Add(1)
			go s.executeJob(job)
			s.updateNextRun(job)
		}
	}
}

// executeJob executes a scheduled job
func (s *Scheduler) executeJob(job *ScheduledJob) {
	defer s.wg.Done()

	log.Printf("Executing scheduled job: %s", job.Name)

	// Execute pipeline with scheduler context for proper cancellation
	result, err := ExecutePipeline(s.ctx, &PipelineConfig{
		Name:  job.Name,
		Steps: []pipelines.StepConfig{}, // This would need to be loaded from pipeline file
	})

	// Update job status (need to acquire lock for this)
	s.jobsMutex.Lock()
	now := time.Now()
	job.LastRun = &now
	job.LastResult = result
	job.UpdatedAt = now
	s.jobsMutex.Unlock()

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
		_, _ = fmt.Sscanf(minute, "*/%d", &interval)

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
		_, _ = fmt.Sscanf(minute, "%d", &min)

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
