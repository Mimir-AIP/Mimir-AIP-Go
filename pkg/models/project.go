package models

import "time"

// ProjectStatus represents the status of a project
type ProjectStatus string

const (
	ProjectStatusActive   ProjectStatus = "active"
	ProjectStatusArchived ProjectStatus = "archived"
	ProjectStatusDraft    ProjectStatus = "draft"
)

// Project represents a workspace for a specific use case
type Project struct {
	ID          string            `json:"id" yaml:"-"`
	Name        string            `json:"name" yaml:"name"`
	Description string            `json:"description" yaml:"description"`
	Version     string            `json:"version" yaml:"version"`
	Status      ProjectStatus     `json:"status" yaml:"status"`
	Metadata    ProjectMetadata   `json:"metadata" yaml:"metadata"`
	Components  ProjectComponents `json:"components" yaml:"components"`
	Settings    ProjectSettings   `json:"settings" yaml:"settings"`
}

// ProjectMetadata contains project metadata
type ProjectMetadata struct {
	CreatedAt time.Time `json:"created_at" yaml:"created_at"`
	UpdatedAt time.Time `json:"updated_at" yaml:"updated_at"`
	Tags      []string  `json:"tags" yaml:"tags"`
}

// ProjectComponents references associated resources
type ProjectComponents struct {
	Pipelines    []string `json:"pipelines" yaml:"pipelines"`
	Ontologies   []string `json:"ontologies" yaml:"ontologies"`
	MLModels     []string `json:"ml_models" yaml:"ml_models"`
	DigitalTwins []string `json:"digital_twins" yaml:"digital_twins"`
}

// ProjectSettings contains project configuration
type ProjectSettings struct {
	Timezone    string `json:"timezone" yaml:"timezone"`
	Environment string `json:"environment" yaml:"environment"`
}

// ProjectCreateRequest represents a request to create a new project
type ProjectCreateRequest struct {
	Name        string            `json:"name" yaml:"name"`
	Description string            `json:"description" yaml:"description"`
	Version     string            `json:"version" yaml:"version"`
	Status      ProjectStatus     `json:"status" yaml:"status"`
	Components  ProjectComponents `json:"components" yaml:"components"`
	Settings    ProjectSettings   `json:"settings" yaml:"settings"`
	Tags        []string          `json:"tags" yaml:"tags"`
}

// ProjectUpdateRequest represents a request to update a project
type ProjectUpdateRequest struct {
	Description *string          `json:"description,omitempty"`
	Version     *string          `json:"version,omitempty"`
	Status      *ProjectStatus   `json:"status,omitempty"`
	Settings    *ProjectSettings `json:"settings,omitempty"`
	Tags        *[]string        `json:"tags,omitempty"`
}

// ProjectListQuery represents query parameters for listing projects
type ProjectListQuery struct {
	Status string
	Tags   []string
	Limit  int
	Offset int
}
