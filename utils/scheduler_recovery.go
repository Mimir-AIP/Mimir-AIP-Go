package utils

import (
	"context"
	"log"
	"time"
)

// PersistJob saves a job to the database for crash recovery
func (s *Scheduler) PersistJob(job *ScheduledJob) error {
	if s.storage == nil {
		return nil // Silently skip if storage not configured
	}

	// Type assert to interface that supports scheduler job persistence
	type schedulerJobSaver interface {
		SaveSchedulerJob(ctx context.Context, id, name, jobType, pipeline, monitoringJobID, cronExpr string, enabled bool, nextRun, lastRun *time.Time) error
	}

	backend, ok := s.storage.(schedulerJobSaver)
	if !ok {
		return nil // Silently skip if not supported
	}

	return backend.SaveSchedulerJob(
		context.Background(),
		job.ID,
		job.Name,
		job.JobType,
		job.Pipeline,
		job.MonitoringJobID,
		job.CronExpr,
		job.Enabled,
		job.NextRun,
		job.LastRun,
	)
}

// RecoverJobsFromDatabase recovers all scheduled jobs from database after restart
func (s *Scheduler) RecoverJobsFromDatabase() error {
	if s.storage == nil {
		log.Println("Storage backend not set, skipping job recovery")
		return nil
	}

	// Type assert to interface that can retrieve jobs
	type schedulerJobRetriever interface {
		GetAllSchedulerJobs(ctx context.Context) ([]SchedulerJobRecord, error)
	}

	retriever, ok := s.storage.(schedulerJobRetriever)
	if !ok {
		log.Println("Storage does not support job retrieval, skipping recovery")
		return nil
	}

	dbJobs, err := retriever.GetAllSchedulerJobs(context.Background())
	if err != nil {
		log.Printf("Failed to retrieve scheduler jobs from database: %v", err)
		return err
	}

	if len(dbJobs) == 0 {
		log.Println("✅ No scheduled jobs to recover")
		return nil
	}

	log.Printf("🔄 Recovering %d scheduled jobs from database...", len(dbJobs))

	s.jobsMutex.Lock()
	defer s.jobsMutex.Unlock()

	recoveredCount := 0
	for _, dbJob := range dbJobs {
		// Skip disabled jobs
		if !dbJob.Enabled {
			continue
		}

		// Validate cron expression
		_, err := parseCronExpression(dbJob.CronExpr)
		if err != nil {
			log.Printf("⚠️  Skipping job %s: invalid cron expression: %v", dbJob.ID, err)
			continue
		}

		// Create ScheduledJob from database record
		job := &ScheduledJob{
			ID:              dbJob.ID,
			Name:            dbJob.Name,
			JobType:         dbJob.JobType,
			Pipeline:        dbJob.Pipeline,
			MonitoringJobID: dbJob.MonitoringJobID,
			CronExpr:        dbJob.CronExpr,
			Enabled:         dbJob.Enabled,
			NextRun:         dbJob.NextRun,
			LastRun:         dbJob.LastRun,
			CreatedAt:       dbJob.CreatedAt,
			UpdatedAt:       dbJob.UpdatedAt,
		}

		// Add to scheduler
		s.jobs[job.ID] = job

		// Update next run if not set or in the past
		if job.NextRun == nil || job.NextRun.Before(time.Now()) {
			s.updateNextRun(job)
		}

		recoveredCount++
		log.Printf("✅ Recovered job: %s (%s)", job.Name, job.JobType)
	}

	log.Printf("✅ Successfully recovered %d/%d scheduled jobs", recoveredCount, len(dbJobs))
	return nil
}

// SchedulerJobRecord represents a scheduler job record from the database
// This mirrors the type from persistence.go to avoid circular imports
type SchedulerJobRecord struct {
	ID              string
	Name            string
	JobType         string
	Pipeline        string
	MonitoringJobID string
	CronExpr        string
	Enabled         bool
	NextRun         *time.Time
	LastRun         *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
