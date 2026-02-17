package metadatastore

import "github.com/mimir-aip/mimir-aip-go/pkg/models"

// MetadataStore is the interface for orchestrator metadata persistence
// This stores project definitions, pipeline configurations, and schedules.
// This is NOT the CIR storage system for ingested/ontology data.
type MetadataStore interface {
	// Project operations
	SaveProject(project *models.Project) error
	GetProject(id string) (*models.Project, error)
	ListProjects() ([]*models.Project, error)
	DeleteProject(id string) error

	// Pipeline operations
	SavePipeline(pipeline *models.Pipeline) error
	GetPipeline(id string) (*models.Pipeline, error)
	ListPipelines() ([]*models.Pipeline, error)
	ListPipelinesByProject(projectID string) ([]*models.Pipeline, error)
	DeletePipeline(id string) error

	// Schedule operations
	SaveSchedule(schedule *models.Schedule) error
	GetSchedule(id string) (*models.Schedule, error)
	ListSchedules() ([]*models.Schedule, error)
	ListSchedulesByProject(projectID string) ([]*models.Schedule, error)
	DeleteSchedule(id string) error
}
