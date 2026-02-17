package scheduler

import (
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"

	"github.com/mimir-aip/mimir-aip-go/pkg/metadatastore"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"github.com/mimir-aip/mimir-aip-go/pkg/pipeline"
	"github.com/mimir-aip/mimir-aip-go/pkg/queue"
)

// Service provides job scheduling operations
type Service struct {
	store           metadatastore.MetadataStore
	pipelineService *pipeline.Service
	queue           *queue.Queue
	cron            *cron.Cron
	jobs            map[string]cron.EntryID // Maps job ID to cron entry ID
}

// NewService creates a new scheduler service
func NewService(store metadatastore.MetadataStore, pipelineService *pipeline.Service, q *queue.Queue) *Service {
	return &Service{
		store:           store,
		pipelineService: pipelineService,
		queue:           q,
		cron:            cron.New(),
		jobs:            make(map[string]cron.EntryID),
	}
}

// Start starts the scheduler
func (s *Service) Start() {
	// Load all enabled jobs and schedule them
	jobs, err := s.store.ListSchedules()
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
func (s *Service) Create(req *models.ScheduleCreateRequest) (*models.Schedule, error) {
	// Validate request
	if err := s.validateCreateRequest(req); err != nil {
		return nil, err
	}

	// Create job
	job := &models.Schedule{
		ID:           uuid.New().String(),
		ProjectID:    req.ProjectID,
		Name:         req.Name,
		Pipelines:    req.Pipelines,
		CronSchedule: req.CronSchedule,
		Enabled:      req.Enabled,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Calculate next run time
	if job.Enabled {
		schedule, err := cron.ParseStandard(job.CronSchedule)
		if err == nil {
			nextRun := schedule.Next(time.Now())
			job.NextRun = &nextRun
		}
	}

	// Save job
	if err := s.store.SaveSchedule(job); err != nil {
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
func (s *Service) Get(id string) (*models.Schedule, error) {
	return s.store.GetSchedule(id)
}

// List lists all scheduled jobs
func (s *Service) List() ([]*models.Schedule, error) {
	return s.store.ListSchedules()
}

// ListByProject lists all jobs for a specific project
func (s *Service) ListByProject(projectID string) ([]*models.Schedule, error) {
	return s.store.ListSchedulesByProject(projectID)
}

// Update updates a scheduled job
func (s *Service) Update(id string, req *models.ScheduleUpdateRequest) (*models.Schedule, error) {
	// Get existing job
	job, err := s.store.GetSchedule(id)
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
	if req.CronSchedule != nil {
		// Validate new schedule
		if _, err := cron.ParseStandard(*req.CronSchedule); err != nil {
			return nil, fmt.Errorf("invalid cron expression: %w", err)
		}
		job.CronSchedule = *req.CronSchedule
	}
	if req.Enabled != nil {
		job.Enabled = *req.Enabled
	}

	// Update timestamp
	job.UpdatedAt = time.Now()

	// Calculate next run time
	if job.Enabled {
		schedule, err := cron.ParseStandard(job.CronSchedule)
		if err == nil {
			nextRun := schedule.Next(time.Now())
			job.NextRun = &nextRun
		}
	} else {
		job.NextRun = nil
	}

	// Save job
	if err := s.store.SaveSchedule(job); err != nil {
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

	return s.store.DeleteSchedule(id)
}

// scheduleJob schedules a job with the cron scheduler
func (s *Service) scheduleJob(job *models.Schedule) error {
	// Parse cron expression
	schedule, err := cron.ParseStandard(job.CronSchedule)
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

	log.Printf("Scheduled job %s (%s) with schedule: %s", job.Name, job.ID, job.CronSchedule)

	return nil
}

// executeJob executes a scheduled job by creating WorkTasks for each pipeline
func (s *Service) executeJob(job *models.Schedule) {
	log.Printf("Executing scheduled job: %s", job.Name)

	// Update last run time
	now := time.Now()
	job.LastRun = &now

	// Calculate next run time
	schedule, err := cron.ParseStandard(job.CronSchedule)
	if err == nil {
		nextRun := schedule.Next(now)
		job.NextRun = &nextRun
	}

	// Save updated job
	if err := s.store.SaveSchedule(job); err != nil {
		log.Printf("Error updating job last run time: %v", err)
	}

	// Create WorkTasks for all pipelines
	workTaskIDs := make([]string, 0)
	for _, pipelineID := range job.Pipelines {
		// Verify pipeline exists
		pipeline, err := s.pipelineService.Get(pipelineID)
		if err != nil {
			log.Printf("Error getting pipeline %s: %v", pipelineID, err)
			continue
		}

		// Create WorkTask for pipeline execution
		workTask := &models.WorkTask{
			ID:          uuid.New().String(),
			Type:        models.WorkTaskTypePipelineExecution,
			Status:      models.WorkTaskStatusQueued,
			Priority:    1, // Default priority for scheduled tasks
			SubmittedAt: time.Now(),
			ProjectID:   pipeline.ProjectID,
			TaskSpec: models.TaskSpec{
				PipelineID: pipelineID,
				ProjectID:  pipeline.ProjectID,
				Parameters: map[string]interface{}{
					"trigger_type":  "scheduled",
					"triggered_by":  job.ID,
					"schedule_name": job.Name,
				},
			},
			ResourceRequirements: models.ResourceRequirements{
				CPU:    "500m", // Default resource requirements
				Memory: "1Gi",
				GPU:    false,
			},
			DataAccess: models.DataAccess{
				InputDatasets:  []string{},
				OutputLocation: fmt.Sprintf("s3://results/schedule-%s/pipeline-%s/", job.ID, pipelineID),
			},
		}

		// Enqueue the WorkTask
		if err := s.queue.Enqueue(workTask); err != nil {
			log.Printf("Error enqueuing work task for pipeline %s: %v", pipelineID, err)
			continue
		}

		workTaskIDs = append(workTaskIDs, workTask.ID)
		log.Printf("  Queued WorkTask %s for pipeline %s", workTask.ID, pipelineID)
	}

	log.Printf("Scheduled job %s completed. Queued %d work tasks", job.Name, len(workTaskIDs))
}

// validateCreateRequest validates a job creation request
func (s *Service) validateCreateRequest(req *models.ScheduleCreateRequest) error {
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
	if _, err := cron.ParseStandard(req.CronSchedule); err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}

	return nil
}
