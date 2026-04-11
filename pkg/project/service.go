package project

import (
	"encoding/json"
	"fmt"
	"regexp"
	"time"

	"github.com/google/uuid"
	"github.com/mimir-aip/mimir-aip-go/pkg/metadatastore"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

var (
	projectNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]{3,50}$`)
	versionRegex     = regexp.MustCompile(`^\d+\.\d+\.\d+$`)
)

// scheduleDeleter removes a persisted schedule and any runtime scheduler registration.
type scheduleDeleter interface {
	Delete(id string) error
}

// projectTaskCleaner removes project-scoped work tasks from persistence and in-memory queue state.
type projectTaskCleaner interface {
	DeleteTasksByProject(projectID string) error
}

// Service provides project management operations
type Service struct {
	store           metadatastore.MetadataStore
	scheduleDeleter scheduleDeleter
	taskCleaner     projectTaskCleaner
}

// NewService creates a new project service
func NewService(store metadatastore.MetadataStore) *Service {
	return &Service{
		store: store,
	}
}

// SetScheduleDeleter wires runtime-aware schedule cleanup into project deletion.
func (s *Service) SetScheduleDeleter(deleter scheduleDeleter) {
	s.scheduleDeleter = deleter
}

// SetTaskCleaner wires in-memory queue cleanup into project deletion.
func (s *Service) SetTaskCleaner(cleaner projectTaskCleaner) {
	s.taskCleaner = cleaner
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
		Settings: req.Settings,
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

// Archive marks a project archived without deleting its persisted resources.
func (s *Service) Archive(id string) error {
	project, err := s.store.GetProject(id)
	if err != nil {
		return err
	}

	project.Status = models.ProjectStatusArchived
	project.Metadata.UpdatedAt = time.Now().UTC()
	if err := s.store.SaveProject(project); err != nil {
		return fmt.Errorf("failed to archive project: %w", err)
	}
	return nil
}

// Delete permanently deletes a project and all persisted project-owned resources.
func (s *Service) Delete(id string) error {
	if _, err := s.store.GetProject(id); err != nil {
		return err
	}

	if err := s.deleteProjectArtifacts(id); err != nil {
		return err
	}

	schedules, err := s.store.ListSchedulesByProject(id)
	if err != nil {
		return fmt.Errorf("failed to list project schedules: %w", err)
	}
	for _, schedule := range schedules {
		if schedule == nil {
			continue
		}
		if s.scheduleDeleter != nil {
			if err := s.scheduleDeleter.Delete(schedule.ID); err != nil {
				return fmt.Errorf("failed to delete project schedule %s: %w", schedule.ID, err)
			}
			continue
		}
		if err := s.store.DeleteSchedule(schedule.ID); err != nil {
			return fmt.Errorf("failed to delete project schedule %s: %w", schedule.ID, err)
		}
	}

	pipelines, err := s.store.ListPipelinesByProject(id)
	if err != nil {
		return fmt.Errorf("failed to list project pipelines: %w", err)
	}
	for _, pipeline := range pipelines {
		if pipeline == nil {
			continue
		}
		if err := s.store.DeletePipeline(pipeline.ID); err != nil {
			return fmt.Errorf("failed to delete project pipeline %s: %w", pipeline.ID, err)
		}
	}

	automations, err := s.store.ListAutomationsByProject(id)
	if err != nil {
		return fmt.Errorf("failed to list project automations: %w", err)
	}
	for _, automation := range automations {
		if automation == nil {
			continue
		}
		if err := s.store.DeleteAutomation(automation.ID); err != nil {
			return fmt.Errorf("failed to delete project automation %s: %w", automation.ID, err)
		}
	}

	twins, err := s.store.ListDigitalTwinsByProject(id)
	if err != nil {
		return fmt.Errorf("failed to list project digital twins: %w", err)
	}
	for _, twin := range twins {
		if twin == nil {
			continue
		}
		if err := s.store.DeleteDigitalTwin(twin.ID); err != nil {
			return fmt.Errorf("failed to delete project digital twin %s: %w", twin.ID, err)
		}
	}

	mlModels, err := s.store.ListMLModelsByProject(id)
	if err != nil {
		return fmt.Errorf("failed to list project ml models: %w", err)
	}
	for _, model := range mlModels {
		if model == nil {
			continue
		}
		if err := s.store.DeleteMLModel(model.ID); err != nil {
			return fmt.Errorf("failed to delete project ml model %s: %w", model.ID, err)
		}
	}

	storageConfigs, err := s.store.ListStorageConfigsByProject(id)
	if err != nil {
		return fmt.Errorf("failed to list project storage configs: %w", err)
	}
	for _, config := range storageConfigs {
		if config == nil {
			continue
		}
		if err := s.store.DeleteStorageConfig(config.ID); err != nil {
			return fmt.Errorf("failed to delete project storage config %s: %w", config.ID, err)
		}
	}

	ontologies, err := s.store.ListOntologiesByProject(id)
	if err != nil {
		return fmt.Errorf("failed to list project ontologies: %w", err)
	}
	for _, ontology := range ontologies {
		if ontology == nil {
			continue
		}
		if err := s.store.DeleteOntology(ontology.ID); err != nil {
			return fmt.Errorf("failed to delete project ontology %s: %w", ontology.ID, err)
		}
	}

	if err := s.store.DeleteProject(id); err != nil {
		return fmt.Errorf("failed to delete project: %w", err)
	}
	return nil
}

func (s *Service) deleteProjectArtifacts(projectID string) error {
	if s.taskCleaner != nil {
		if err := s.taskCleaner.DeleteTasksByProject(projectID); err != nil {
			return fmt.Errorf("failed to delete project work tasks: %w", err)
		}
	} else if err := s.store.DeleteWorkTasksByProject(projectID); err != nil {
		return fmt.Errorf("failed to delete project work tasks: %w", err)
	}

	if err := s.store.DeleteReviewItemsByProject(projectID); err != nil {
		return fmt.Errorf("failed to delete project review items: %w", err)
	}
	if err := s.store.DeleteInsightsByProject(projectID); err != nil {
		return fmt.Errorf("failed to delete project insights: %w", err)
	}
	if err := s.store.DeleteAnalysisRunsByProject(projectID); err != nil {
		return fmt.Errorf("failed to delete project analysis runs: %w", err)
	}
	return nil
}

// Clone deep-clones a project's persisted configuration into a new draft project.
func (s *Service) Clone(id string, newName string) (*models.Project, error) {
	original, err := s.store.GetProject(id)
	if err != nil {
		return nil, err
	}

	if !projectNameRegex.MatchString(newName) {
		return nil, fmt.Errorf("invalid project name: must be 3-50 alphanumeric characters, hyphens, or underscores")
	}
	if err := s.checkNameUnique(newName); err != nil {
		return nil, err
	}

	ontologies, err := s.store.ListOntologiesByProject(id)
	if err != nil {
		return nil, fmt.Errorf("failed to list project ontologies: %w", err)
	}
	storageConfigs, err := s.store.ListStorageConfigsByProject(id)
	if err != nil {
		return nil, fmt.Errorf("failed to list project storage configs: %w", err)
	}
	pipelines, err := s.store.ListPipelinesByProject(id)
	if err != nil {
		return nil, fmt.Errorf("failed to list project pipelines: %w", err)
	}
	schedules, err := s.store.ListSchedulesByProject(id)
	if err != nil {
		return nil, fmt.Errorf("failed to list project schedules: %w", err)
	}
	mlModels, err := s.store.ListMLModelsByProject(id)
	if err != nil {
		return nil, fmt.Errorf("failed to list project ml models: %w", err)
	}
	twins, err := s.store.ListDigitalTwinsByProject(id)
	if err != nil {
		return nil, fmt.Errorf("failed to list project digital twins: %w", err)
	}
	automations, err := s.store.ListAutomationsByProject(id)
	if err != nil {
		return nil, fmt.Errorf("failed to list project automations: %w", err)
	}

	clonedProjectID := uuid.New().String()
	idMap := map[string]string{original.ID: clonedProjectID}
	for _, ontology := range ontologies {
		idMap[ontology.ID] = uuid.New().String()
	}
	for _, config := range storageConfigs {
		idMap[config.ID] = uuid.New().String()
	}
	for _, pipeline := range pipelines {
		idMap[pipeline.ID] = uuid.New().String()
	}
	for _, schedule := range schedules {
		idMap[schedule.ID] = uuid.New().String()
	}
	for _, model := range mlModels {
		idMap[model.ID] = uuid.New().String()
	}
	for _, twin := range twins {
		idMap[twin.ID] = uuid.New().String()
	}
	for _, automation := range automations {
		idMap[automation.ID] = uuid.New().String()
	}

	now := time.Now().UTC()
	cloned := &models.Project{
		ID:          clonedProjectID,
		Name:        newName,
		Description: original.Description,
		Version:     original.Version,
		Status:      models.ProjectStatusDraft,
		Metadata: models.ProjectMetadata{
			CreatedAt: now,
			UpdatedAt: now,
			Tags:      append([]string{}, original.Metadata.Tags...),
		},
		Settings: original.Settings,
	}
	if err := s.store.SaveProject(cloned); err != nil {
		return nil, fmt.Errorf("failed to save cloned project: %w", err)
	}
	for _, ontology := range ontologies {
		copy, err := cloneWithIDRemap(ontology, idMap)
		if err != nil {
			return nil, fmt.Errorf("failed to clone ontology %s: %w", ontology.ID, err)
		}
		copy.ID = idMap[ontology.ID]
		copy.ProjectID = clonedProjectID
		copy.CreatedAt = now
		copy.UpdatedAt = now
		if err := s.store.SaveOntology(copy); err != nil {
			return nil, fmt.Errorf("failed to save cloned ontology %s: %w", ontology.ID, err)
		}
	}

	for _, config := range storageConfigs {
		copy, err := cloneWithIDRemap(config, idMap)
		if err != nil {
			return nil, fmt.Errorf("failed to clone storage config %s: %w", config.ID, err)
		}
		copy.ID = idMap[config.ID]
		copy.ProjectID = clonedProjectID
		copy.CreatedAt = now.Format(time.RFC3339)
		copy.UpdatedAt = copy.CreatedAt
		if err := s.store.SaveStorageConfig(copy); err != nil {
			return nil, fmt.Errorf("failed to save cloned storage config %s: %w", config.ID, err)
		}
	}

	for _, pipeline := range pipelines {
		copy, err := cloneWithIDRemap(pipeline, idMap)
		if err != nil {
			return nil, fmt.Errorf("failed to clone pipeline %s: %w", pipeline.ID, err)
		}
		copy.ID = idMap[pipeline.ID]
		copy.ProjectID = clonedProjectID
		copy.CreatedAt = now
		copy.UpdatedAt = now
		if err := s.store.SavePipeline(copy); err != nil {
			return nil, fmt.Errorf("failed to save cloned pipeline %s: %w", pipeline.ID, err)
		}
	}

	for _, schedule := range schedules {
		copy, err := cloneWithIDRemap(schedule, idMap)
		if err != nil {
			return nil, fmt.Errorf("failed to clone schedule %s: %w", schedule.ID, err)
		}
		copy.ID = idMap[schedule.ID]
		copy.ProjectID = clonedProjectID
		copy.Enabled = false
		copy.CreatedAt = now
		copy.UpdatedAt = now
		copy.LastRun = nil
		copy.NextRun = nil
		if err := s.store.SaveSchedule(copy); err != nil {
			return nil, fmt.Errorf("failed to save cloned schedule %s: %w", schedule.ID, err)
		}
	}

	for _, model := range mlModels {
		copy, err := cloneWithIDRemap(model, idMap)
		if err != nil {
			return nil, fmt.Errorf("failed to clone ml model %s: %w", model.ID, err)
		}
		copy.ID = idMap[model.ID]
		copy.ProjectID = clonedProjectID
		copy.Status = models.ModelStatusDraft
		copy.TrainingTaskID = ""
		copy.TrainingMetrics = nil
		copy.PerformanceMetrics = nil
		copy.ModelArtifactPath = ""
		copy.TrainedAt = nil
		copy.CreatedAt = now
		copy.UpdatedAt = now
		if err := s.store.SaveMLModel(copy); err != nil {
			return nil, fmt.Errorf("failed to save cloned ml model %s: %w", model.ID, err)
		}
	}

	for _, twin := range twins {
		copy, err := cloneWithIDRemap(twin, idMap)
		if err != nil {
			return nil, fmt.Errorf("failed to clone digital twin %s: %w", twin.ID, err)
		}
		copy.ID = idMap[twin.ID]
		copy.ProjectID = clonedProjectID
		copy.Status = "active"
		copy.LastSyncAt = nil
		copy.CreatedAt = now
		copy.UpdatedAt = now
		if err := s.store.SaveDigitalTwin(copy); err != nil {
			return nil, fmt.Errorf("failed to save cloned digital twin %s: %w", twin.ID, err)
		}
	}

	for _, automation := range automations {
		copy, err := cloneWithIDRemap(automation, idMap)
		if err != nil {
			return nil, fmt.Errorf("failed to clone automation %s: %w", automation.ID, err)
		}
		copy.ID = idMap[automation.ID]
		copy.ProjectID = clonedProjectID
		copy.CreatedAt = now
		copy.UpdatedAt = now
		if err := s.store.SaveAutomation(copy); err != nil {
			return nil, fmt.Errorf("failed to save cloned automation %s: %w", automation.ID, err)
		}
	}

	cloned.Metadata.UpdatedAt = now
	if err := s.store.SaveProject(cloned); err != nil {
		return nil, fmt.Errorf("failed to finalize cloned project: %w", err)
	}

	return cloned, nil
}

func cloneWithIDRemap[T any](src *T, idMap map[string]string) (*T, error) {
	encoded, err := json.Marshal(src)
	if err != nil {
		return nil, err
	}

	var generic any
	if err := json.Unmarshal(encoded, &generic); err != nil {
		return nil, err
	}

	remapped, err := json.Marshal(remapIDs(generic, idMap))
	if err != nil {
		return nil, err
	}

	var dst T
	if err := json.Unmarshal(remapped, &dst); err != nil {
		return nil, err
	}
	return &dst, nil
}

func remapIDs(value any, idMap map[string]string) any {
	switch typed := value.(type) {
	case string:
		if mapped, ok := idMap[typed]; ok {
			return mapped
		}
		return typed
	case []any:
		out := make([]any, len(typed))
		for i, item := range typed {
			out[i] = remapIDs(item, idMap)
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, item := range typed {
			mappedKey := key
			if mapped, ok := idMap[key]; ok {
				mappedKey = mapped
			}
			out[mappedKey] = remapIDs(item, idMap)
		}
		return out
	default:
		return value
	}
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
