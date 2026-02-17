package scheduler

import (
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"

	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"github.com/mimir-aip/mimir-aip-go/pkg/pipeline"
	"github.com/mimir-aip/mimir-aip-go/pkg/storage"
)

// Service provides job scheduling operations
type Service struct {
	store           *storage.FileStore
	pipelineService *pipeline.Service
	cron            *cron.Cron
	jobs            map[string]cron.EntryID // Maps job ID to cron entry ID
}

// NewService creates a new scheduler service
func NewService(store *storage.FileStore, pipelineService *pipeline.Service) *Service {
	return &Service{
		store:           store,
		pipelineService: pipelineService,
		cron:            cron.New(),
		jobs:            make(map[string]cron.EntryID),
	}
}

// Start starts the scheduler
func (s *Service) Start() {
	// Load all enabled jobs and schedule them
	jobs, err := s.store.ListJobs()
	if err != nil {
		log.Printf("Error loading jobs: %v", err)
		return
	}

	for _, job := range jobs {
		if job.Enabled {
			if err := s.scheduleJob(job); err != nil {
				log.Printf("Error scheduling job %s: %v", job.Name, err)
			}
		}
	}

	s.cron.Start()
	log.Println("Job scheduler started")
}

// Stop stops the scheduler
func (s *Service) Stop() {
	s.cron.Stop()
	log.Println("Job scheduler stopped")
}

// Create creates a new scheduled job
func (s *Service) Create(req *models.ScheduledJobCreateRequest) (*models.ScheduledJob, error) {
	// Validate request
	if err := s.validateCreateRequest(req); err != nil {
		return nil, err
	}

	// Create job
	job := &models.ScheduledJob{
		ID:        uuid.New().String(),
		ProjectID: req.ProjectID,
		Name:      req.Name,
		Pipelines: req.Pipelines,
		Schedule:  req.Schedule,
		Enabled:   req.Enabled,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Calculate next run time
	if job.Enabled {
		schedule, err := cron.ParseStandard(job.Schedule)
		if err == nil {
			nextRun := schedule.Next(time.Now())
			job.NextRun = &nextRun
		}
	}

	// Save job
	if err := s.store.SaveJob(job); err != nil {
		return nil, fmt.Errorf("failed to save job: %w", err)
	}

	// Schedule job if enabled
	if job.Enabled {
		if err := s.scheduleJob(job); err != nil {
			log.Printf("Warning: Failed to schedule job %s: %v", job.Name, err)
		}
	}

	return job, nil
}

// Get retrieves a job by ID
func (s *Service) Get(id string) (*models.ScheduledJob, error) {
	return s.store.GetJob(id)
}

// List lists all scheduled jobs
func (s *Service) List() ([]*models.ScheduledJob, error) {
	return s.store.ListJobs()
}

// ListByProject lists all jobs for a specific project
func (s *Service) ListByProject(projectID string) ([]*models.ScheduledJob, error) {
	return s.store.ListJobsByProject(projectID)
}

// Update updates a scheduled job
func (s *Service) Update(id string, req *models.ScheduledJobUpdateRequest) (*models.ScheduledJob, error) {
	// Get existing job
	job, err := s.store.GetJob(id)
	if err != nil {
		return nil, err
	}

	// Unschedule if currently scheduled
	if entryID, ok := s.jobs[id]; ok {
		s.cron.Remove(entryID)
		delete(s.jobs, id)
	}

	// Update fields
	if req.Name != nil {
		job.Name = *req.Name
	}
	if req.Pipelines != nil {
		job.Pipelines = *req.Pipelines
	}
	if req.Schedule != nil {
		// Validate new schedule
		if _, err := cron.ParseStandard(*req.Schedule); err != nil {
			return nil, fmt.Errorf("invalid cron expression: %w", err)
		}
		job.Schedule = *req.Schedule
	}
	if req.Enabled != nil {
		job.Enabled = *req.Enabled
	}

	// Update timestamp
	job.UpdatedAt = time.Now()

	// Calculate next run time
	if job.Enabled {
		schedule, err := cron.ParseStandard(job.Schedule)
		if err == nil {
			nextRun := schedule.Next(time.Now())
			job.NextRun = &nextRun
		}
	} else {
		job.NextRun = nil
	}

	// Save job
	if err := s.store.SaveJob(job); err != nil {
		return nil, fmt.Errorf("failed to save job: %w", err)
	}

	// Reschedule if enabled
	if job.Enabled {
		if err := s.scheduleJob(job); err != nil {
			log.Printf("Warning: Failed to schedule job %s: %v", job.Name, err)
		}
	}

	return job, nil
}

// Delete deletes a scheduled job
func (s *Service) Delete(id string) error {
	// Unschedule if currently scheduled
	if entryID, ok := s.jobs[id]; ok {
		s.cron.Remove(entryID)
		delete(s.jobs, id)
	}

	return s.store.DeleteJob(id)
}

// scheduleJob schedules a job with the cron scheduler
func (s *Service) scheduleJob(job *models.ScheduledJob) error {
	// Parse cron expression
	schedule, err := cron.ParseStandard(job.Schedule)
	if err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}

	// Create job function
	jobFunc := func() {
		s.executeJob(job)
	}

	// Add to cron scheduler
	entryID := s.cron.Schedule(schedule, cron.FuncJob(jobFunc))
	s.jobs[job.ID] = entryID

	log.Printf("Scheduled job %s (%s) with schedule: %s", job.Name, job.ID, job.Schedule)

	return nil
}

// executeJob executes a scheduled job
func (s *Service) executeJob(job *models.ScheduledJob) {
	log.Printf("Executing scheduled job: %s", job.Name)

	// Update last run time
	now := time.Now()
	job.LastRun = &now

	// Calculate next run time
	schedule, err := cron.ParseStandard(job.Schedule)
	if err == nil {
		nextRun := schedule.Next(now)
		job.NextRun = &nextRun
	}

	// Save updated job
	if err := s.store.SaveJob(job); err != nil {
		log.Printf("Error updating job last run time: %v", err)
	}

	// Execute all pipelines
	pipelineRuns := make([]string, 0)
	for _, pipelineID := range job.Pipelines {
		execution, err := s.pipelineService.Execute(pipelineID, &models.PipelineExecutionRequest{
			TriggerType: "scheduled",
			TriggeredBy: job.ID,
		})

		if err != nil {
			log.Printf("Error executing pipeline %s: %v", pipelineID, err)
			continue
		}

		pipelineRuns = append(pipelineRuns, execution.ID)
		log.Printf("  Pipeline %s executed: %s (status: %s)", pipelineID, execution.ID, execution.Status)
	}

	log.Printf("Scheduled job %s completed. Executed %d pipelines", job.Name, len(pipelineRuns))
}

// validateCreateRequest validates a job creation request
func (s *Service) validateCreateRequest(req *models.ScheduledJobCreateRequest) error {
	// Validate name
	if req.Name == "" {
		return fmt.Errorf("job name is required")
	}

	// Validate pipelines
	if len(req.Pipelines) == 0 {
		return fmt.Errorf("job must have at least one pipeline")
	}

	// Verify pipelines exist
	for _, pipelineID := range req.Pipelines {
		if _, err := s.pipelineService.Get(pipelineID); err != nil {
			return fmt.Errorf("pipeline not found: %s", pipelineID)
		}
	}

	// Validate cron expression
	if _, err := cron.ParseStandard(req.Schedule); err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}

	return nil
}
