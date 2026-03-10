package project

import (
	"fmt"
	"regexp"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/mimir-aip/mimir-aip-go/pkg/metadatastore"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

var (
	projectNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]{3,50}$`)
	versionRegex     = regexp.MustCompile(`^\d+\.\d+\.\d+$`)

	// Association validation errors
	ErrComponentNotFound        = fmt.Errorf("component not found")
	ErrComponentProjectMismatch = fmt.Errorf("component belongs to a different project")
)

// Service provides project management operations
type Service struct {
	store metadatastore.MetadataStore
}

// NewService creates a new project service
func NewService(store metadatastore.MetadataStore) *Service {
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
	if project.Settings.OnboardingMode == "" {
		project.Settings.OnboardingMode = models.ProjectOnboardingModeAdvanced
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
		if req.Settings.OnboardingMode != "" {
			project.Settings.OnboardingMode = req.Settings.OnboardingMode
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

	if err := s.requirePipelineOwnership(projectID, pipelineID); err != nil {
		return err
	}

	if slices.Contains(project.Components.Pipelines, pipelineID) {
		return nil
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

// AddOntology associates an ontology with a project
func (s *Service) AddOntology(projectID, ontologyID string) error {
	project, err := s.store.GetProject(projectID)
	if err != nil {
		return err
	}

	if err := s.requireOntologyOwnership(projectID, ontologyID); err != nil {
		return err
	}

	if slices.Contains(project.Components.Ontologies, ontologyID) {
		return nil
	}

	project.Components.Ontologies = append(project.Components.Ontologies, ontologyID)
	project.Metadata.UpdatedAt = time.Now()

	return s.store.SaveProject(project)
}

// RemoveOntology removes an ontology association from a project
func (s *Service) RemoveOntology(projectID, ontologyID string) error {
	project, err := s.store.GetProject(projectID)
	if err != nil {
		return err
	}

	filtered := make([]string, 0)
	for _, id := range project.Components.Ontologies {
		if id != ontologyID {
			filtered = append(filtered, id)
		}
	}

	project.Components.Ontologies = filtered
	project.Metadata.UpdatedAt = time.Now()

	return s.store.SaveProject(project)
}

// AddMLModel associates an ML model with a project
func (s *Service) AddMLModel(projectID, modelID string) error {
	project, err := s.store.GetProject(projectID)
	if err != nil {
		return err
	}

	if err := s.requireMLModelOwnership(projectID, modelID); err != nil {
		return err
	}

	if slices.Contains(project.Components.MLModels, modelID) {
		return nil
	}

	project.Components.MLModels = append(project.Components.MLModels, modelID)
	project.Metadata.UpdatedAt = time.Now()

	return s.store.SaveProject(project)
}

// RemoveMLModel removes an ML model association from a project
func (s *Service) RemoveMLModel(projectID, modelID string) error {
	project, err := s.store.GetProject(projectID)
	if err != nil {
		return err
	}

	filtered := make([]string, 0)
	for _, id := range project.Components.MLModels {
		if id != modelID {
			filtered = append(filtered, id)
		}
	}

	project.Components.MLModels = filtered
	project.Metadata.UpdatedAt = time.Now()

	return s.store.SaveProject(project)
}

// AddDigitalTwin associates a digital twin with a project
func (s *Service) AddDigitalTwin(projectID, twinID string) error {
	project, err := s.store.GetProject(projectID)
	if err != nil {
		return err
	}

	if err := s.requireDigitalTwinOwnership(projectID, twinID); err != nil {
		return err
	}

	if slices.Contains(project.Components.DigitalTwins, twinID) {
		return nil
	}

	project.Components.DigitalTwins = append(project.Components.DigitalTwins, twinID)
	project.Metadata.UpdatedAt = time.Now()

	return s.store.SaveProject(project)
}

// RemoveDigitalTwin removes a digital twin association from a project
func (s *Service) RemoveDigitalTwin(projectID, twinID string) error {
	project, err := s.store.GetProject(projectID)
	if err != nil {
		return err
	}

	filtered := make([]string, 0)
	for _, id := range project.Components.DigitalTwins {
		if id != twinID {
			filtered = append(filtered, id)
		}
	}

	project.Components.DigitalTwins = filtered
	project.Metadata.UpdatedAt = time.Now()

	return s.store.SaveProject(project)
}

// AddStorage associates a storage config with a project
func (s *Service) AddStorage(projectID, storageID string) error {
	project, err := s.store.GetProject(projectID)
	if err != nil {
		return err
	}

	if err := s.requireStorageOwnership(projectID, storageID); err != nil {
		return err
	}

	if slices.Contains(project.Components.StorageConfigs, storageID) {
		return nil
	}

	project.Components.StorageConfigs = append(project.Components.StorageConfigs, storageID)
	project.Metadata.UpdatedAt = time.Now()

	return s.store.SaveProject(project)
}

// RemoveStorage removes a storage config association from a project
func (s *Service) RemoveStorage(projectID, storageID string) error {
	project, err := s.store.GetProject(projectID)
	if err != nil {
		return err
	}

	filtered := make([]string, 0)
	for _, id := range project.Components.StorageConfigs {
		if id != storageID {
			filtered = append(filtered, id)
		}
	}

	project.Components.StorageConfigs = filtered
	project.Metadata.UpdatedAt = time.Now()

	return s.store.SaveProject(project)
}

func (s *Service) requirePipelineOwnership(projectID, pipelineID string) error {
	pipeline, err := s.store.GetPipeline(pipelineID)
	if err != nil {
		return fmt.Errorf("%w: pipeline %s", ErrComponentNotFound, pipelineID)
	}
	if pipeline.ProjectID != projectID {
		return fmt.Errorf("%w: pipeline %s belongs to project %s", ErrComponentProjectMismatch, pipelineID, pipeline.ProjectID)
	}
	return nil
}

func (s *Service) requireOntologyOwnership(projectID, ontologyID string) error {
	ontology, err := s.store.GetOntology(ontologyID)
	if err != nil {
		return fmt.Errorf("%w: ontology %s", ErrComponentNotFound, ontologyID)
	}
	if ontology.ProjectID != projectID {
		return fmt.Errorf("%w: ontology %s belongs to project %s", ErrComponentProjectMismatch, ontologyID, ontology.ProjectID)
	}
	return nil
}

func (s *Service) requireMLModelOwnership(projectID, modelID string) error {
	model, err := s.store.GetMLModel(modelID)
	if err != nil {
		return fmt.Errorf("%w: ml model %s", ErrComponentNotFound, modelID)
	}
	if model.ProjectID != projectID {
		return fmt.Errorf("%w: ml model %s belongs to project %s", ErrComponentProjectMismatch, modelID, model.ProjectID)
	}
	return nil
}

func (s *Service) requireDigitalTwinOwnership(projectID, twinID string) error {
	twin, err := s.store.GetDigitalTwin(twinID)
	if err != nil {
		return fmt.Errorf("%w: digital twin %s", ErrComponentNotFound, twinID)
	}
	if twin.ProjectID != projectID {
		return fmt.Errorf("%w: digital twin %s belongs to project %s", ErrComponentProjectMismatch, twinID, twin.ProjectID)
	}
	return nil
}

func (s *Service) requireStorageOwnership(projectID, storageID string) error {
	storageConfig, err := s.store.GetStorageConfig(storageID)
	if err != nil {
		return fmt.Errorf("%w: storage config %s", ErrComponentNotFound, storageID)
	}
	if storageConfig.ProjectID != projectID {
		return fmt.Errorf("%w: storage config %s belongs to project %s", ErrComponentProjectMismatch, storageID, storageConfig.ProjectID)
	}
	return nil
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
