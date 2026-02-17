package project

import (
	"fmt"
	"regexp"
	"time"

	"github.com/google/uuid"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"github.com/mimir-aip/mimir-aip-go/pkg/storage"
)

var (
	projectNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]{3,50}$`)
	versionRegex     = regexp.MustCompile(`^\d+\.\d+\.\d+$`)
)

// Service provides project management operations
type Service struct {
	store *storage.FileStore
}

// NewService creates a new project service
func NewService(store *storage.FileStore) *Service {
	return &Service{
		store: store,
	}
}

// Create creates a new project
func (s *Service) Create(req *models.ProjectCreateRequest) (*models.Project, error) {
	// Validate request
	if err := s.validateCreateRequest(req); err != nil {
		return nil, err
	}

	// Check name uniqueness
	if err := s.checkNameUnique(req.Name); err != nil {
		return nil, err
	}

	// Create project
	project := &models.Project{
		ID:          uuid.New().String(),
		Name:        req.Name,
		Description: req.Description,
		Version:     req.Version,
		Status:      req.Status,
		Metadata: models.ProjectMetadata{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Tags:      req.Tags,
		},
		Components: req.Components,
		Settings:   req.Settings,
	}

	// Set defaults
	if project.Status == "" {
		project.Status = models.ProjectStatusActive
	}
	if project.Settings.Timezone == "" {
		project.Settings.Timezone = "UTC"
	}
	if project.Settings.Environment == "" {
		project.Settings.Environment = "development"
	}
	if project.Metadata.Tags == nil {
		project.Metadata.Tags = []string{}
	}
	if project.Components.Pipelines == nil {
		project.Components.Pipelines = []string{}
	}
	if project.Components.Ontologies == nil {
		project.Components.Ontologies = []string{}
	}
	if project.Components.MLModels == nil {
		project.Components.MLModels = []string{}
	}
	if project.Components.DigitalTwins == nil {
		project.Components.DigitalTwins = []string{}
	}

	// Save project
	if err := s.store.SaveProject(project); err != nil {
		return nil, fmt.Errorf("failed to save project: %w", err)
	}

	return project, nil
}

// Get retrieves a project by ID
func (s *Service) Get(id string) (*models.Project, error) {
	return s.store.GetProject(id)
}

// List lists all projects
func (s *Service) List(query *models.ProjectListQuery) ([]*models.Project, error) {
	projects, err := s.store.ListProjects()
	if err != nil {
		return nil, err
	}

	// Filter by status if specified
	if query != nil && query.Status != "" {
		filtered := make([]*models.Project, 0)
		for _, p := range projects {
			if string(p.Status) == query.Status {
				filtered = append(filtered, p)
			}
		}
		projects = filtered
	}

	// Apply pagination if specified
	if query != nil && query.Limit > 0 {
		start := query.Offset
		end := query.Offset + query.Limit
		if start >= len(projects) {
			return []*models.Project{}, nil
		}
		if end > len(projects) {
			end = len(projects)
		}
		projects = projects[start:end]
	}

	return projects, nil
}

// Update updates a project
func (s *Service) Update(id string, req *models.ProjectUpdateRequest) (*models.Project, error) {
	// Get existing project
	project, err := s.store.GetProject(id)
	if err != nil {
		return nil, err
	}

	// Update fields
	if req.Description != nil {
		project.Description = *req.Description
	}
	if req.Version != nil {
		if err := s.validateVersion(*req.Version); err != nil {
			return nil, err
		}
		project.Version = *req.Version
	}
	if req.Status != nil {
		project.Status = *req.Status
	}
	if req.Settings != nil {
		if req.Settings.Timezone != "" {
			project.Settings.Timezone = req.Settings.Timezone
		}
		if req.Settings.Environment != "" {
			project.Settings.Environment = req.Settings.Environment
		}
	}
	if req.Tags != nil {
		project.Metadata.Tags = *req.Tags
	}

	// Update timestamp
	project.Metadata.UpdatedAt = time.Now()

	// Save project
	if err := s.store.SaveProject(project); err != nil {
		return nil, fmt.Errorf("failed to save project: %w", err)
	}

	return project, nil
}

// Delete deletes a project
func (s *Service) Delete(id string) error {
	// Get project to verify it exists
	project, err := s.store.GetProject(id)
	if err != nil {
		return err
	}

	// Soft delete - update status to archived
	project.Status = models.ProjectStatusArchived
	project.Metadata.UpdatedAt = time.Now()

	if err := s.store.SaveProject(project); err != nil {
		return fmt.Errorf("failed to archive project: %w", err)
	}

	return nil
}

// Clone clones a project
func (s *Service) Clone(id string, newName string) (*models.Project, error) {
	// Get original project
	original, err := s.store.GetProject(id)
	if err != nil {
		return nil, err
	}

	// Validate new name
	if !projectNameRegex.MatchString(newName) {
		return nil, fmt.Errorf("invalid project name: must be 3-50 alphanumeric characters, hyphens, or underscores")
	}

	// Check name uniqueness
	if err := s.checkNameUnique(newName); err != nil {
		return nil, err
	}

	// Create cloned project
	cloned := &models.Project{
		ID:          uuid.New().String(),
		Name:        newName,
		Description: original.Description + " (cloned)",
		Version:     "1.0.0",
		Status:      models.ProjectStatusDraft,
		Metadata: models.ProjectMetadata{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Tags:      append([]string{}, original.Metadata.Tags...),
		},
		Components: models.ProjectComponents{
			Pipelines:    []string{},
			Ontologies:   []string{},
			MLModels:     []string{},
			DigitalTwins: []string{},
		},
		Settings: original.Settings,
	}

	// Save cloned project
	if err := s.store.SaveProject(cloned); err != nil {
		return nil, fmt.Errorf("failed to save cloned project: %w", err)
	}

	return cloned, nil
}

// AddPipeline associates a pipeline with a project
func (s *Service) AddPipeline(projectID, pipelineID string) error {
	project, err := s.store.GetProject(projectID)
	if err != nil {
		return err
	}

	// Check if already associated
	for _, id := range project.Components.Pipelines {
		if id == pipelineID {
			return nil // Already associated
		}
	}

	project.Components.Pipelines = append(project.Components.Pipelines, pipelineID)
	project.Metadata.UpdatedAt = time.Now()

	return s.store.SaveProject(project)
}

// RemovePipeline removes a pipeline association from a project
func (s *Service) RemovePipeline(projectID, pipelineID string) error {
	project, err := s.store.GetProject(projectID)
	if err != nil {
		return err
	}

	// Remove pipeline ID
	filtered := make([]string, 0)
	for _, id := range project.Components.Pipelines {
		if id != pipelineID {
			filtered = append(filtered, id)
		}
	}

	project.Components.Pipelines = filtered
	project.Metadata.UpdatedAt = time.Now()

	return s.store.SaveProject(project)
}

// validateCreateRequest validates a project creation request
func (s *Service) validateCreateRequest(req *models.ProjectCreateRequest) error {
	// Validate name
	if !projectNameRegex.MatchString(req.Name) {
		return fmt.Errorf("invalid project name: must be 3-50 alphanumeric characters, hyphens, or underscores")
	}

	// Validate version if provided
	if req.Version != "" {
		if err := s.validateVersion(req.Version); err != nil {
			return err
		}
	}

	// Validate status if provided
	if req.Status != "" {
		if req.Status != models.ProjectStatusActive &&
			req.Status != models.ProjectStatusArchived &&
			req.Status != models.ProjectStatusDraft {
			return fmt.Errorf("invalid status: must be one of active, archived, draft")
		}
	}

	return nil
}

// validateVersion validates a version string
func (s *Service) validateVersion(version string) error {
	if !versionRegex.MatchString(version) {
		return fmt.Errorf("invalid version: must follow semantic versioning (e.g., 1.0.0)")
	}
	return nil
}

// checkNameUnique checks if a project name is unique
func (s *Service) checkNameUnique(name string) error {
	projects, err := s.store.ListProjects()
	if err != nil {
		return err
	}

	for _, p := range projects {
		if p.Name == name && p.Status != models.ProjectStatusArchived {
			return fmt.Errorf("project name already exists: %s", name)
		}
	}

	return nil
}
