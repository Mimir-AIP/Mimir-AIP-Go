package storage

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"log"
	"strings"
	"time"
)

// CreateStorageConfig creates a new storage configuration for a project.
func (s *Service) CreateStorageConfig(projectID, pluginType string, config map[string]interface{}) (*models.StorageConfig, error) {
	return s.CreateStorageConfigWithOntology(projectID, pluginType, config, "")
}

// CreateStorageConfigWithOntology creates a storage configuration and, when ontologyID is provided,
// initializes the backend schema from the persisted compiled ontology before making the config active.
func (s *Service) CreateStorageConfigWithOntology(projectID, pluginType string, config map[string]interface{}, ontologyID string) (*models.StorageConfig, error) {
	if err := s.ensureProjectExists(projectID); err != nil {
		return nil, err
	}
	plugin, err := s.GetPlugin(pluginType)
	if err != nil {
		return nil, fmt.Errorf("invalid plugin type: %w", err)
	}
	if config == nil {
		config = map[string]interface{}{}
	}
	if err := s.validateStoragePluginConfig(pluginType, config); err != nil {
		return nil, fmt.Errorf("invalid storage config: %w", err)
	}

	var ontologyDefinition *models.OntologyDefinition
	ontologyID = strings.TrimSpace(ontologyID)
	if ontologyID != "" {
		ontologyRecord, err := s.store.GetOntology(ontologyID)
		if err != nil {
			return nil, fmt.Errorf("ontology not found: %w", err)
		}
		if ontologyRecord.ProjectID != projectID {
			return nil, fmt.Errorf("ontology %s belongs to project %s, not %s", ontologyID, ontologyRecord.ProjectID, projectID)
		}
		compiled, err := s.store.GetCompiledOntology(ontologyID)
		if err != nil {
			return nil, fmt.Errorf("compiled ontology not found for storage initialization: %w", err)
		}
		ontologyDefinition = ontologyDefinitionFromCompiled(compiled)

		pluginConfig := &models.PluginConfig{
			ConnectionString: getConnectionString(config),
			Credentials:      getCredentials(config),
			Options:          getOptions(config),
		}
		if err := plugin.Initialize(pluginConfig); err != nil {
			return nil, fmt.Errorf("failed to initialize storage plugin: %w", err)
		}
		if err := plugin.CreateSchema(ontologyDefinition); err != nil {
			return nil, fmt.Errorf("failed to create ontology-backed storage schema: %w", err)
		}
	}

	now := time.Now().Format(time.RFC3339)
	storageConfig := &models.StorageConfig{
		ID:         uuid.New().String(),
		ProjectID:  projectID,
		PluginType: pluginType,
		Config:     config,
		OntologyID: ontologyID,
		Active:     true,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := s.store.SaveStorageConfig(storageConfig); err != nil {
		return nil, fmt.Errorf("failed to save storage config: %w", err)
	}
	log.Printf("Created storage config %s for project %s using plugin %s", storageConfig.ID, projectID, pluginType)
	return storageConfig, nil
}

// GetStorageConfig retrieves a storage configuration by ID
func (s *Service) GetStorageConfig(storageID string) (*models.StorageConfig, error) {
	return s.store.GetStorageConfig(storageID)
}

// GetProjectStorageConfigs retrieves all storage configurations for a project
func (s *Service) GetProjectStorageConfigs(projectID string) ([]*models.StorageConfig, error) {
	return s.store.ListStorageConfigsByProject(projectID)
}

// UpdateStorageConfig updates a storage configuration
func (s *Service) UpdateStorageConfig(storageID string, config map[string]interface{}, active *bool) error {
	storageConfig, err := s.store.GetStorageConfig(storageID)
	if err != nil {
		return fmt.Errorf("storage config not found: %w", err)
	}
	if config != nil {
		if err := s.validateStoragePluginConfig(storageConfig.PluginType, config); err != nil {
			return fmt.Errorf("invalid storage config: %w", err)
		}
		storageConfig.Config = config
	}
	if active != nil {
		storageConfig.Active = *active
	}
	storageConfig.UpdatedAt = time.Now().Format(time.RFC3339)
	if err := s.store.SaveStorageConfig(storageConfig); err != nil {
		return fmt.Errorf("failed to update storage config: %w", err)
	}
	log.Printf("Updated storage config %s", storageID)
	return nil
}

// StorageConfigInUseError reports the persisted resources that still reference a storage config.
type StorageConfigInUseError struct {
	StorageID  string
	References []string
}

func (e *StorageConfigInUseError) Error() string {
	return fmt.Sprintf("storage config %s is still referenced by %s", e.StorageID, strings.Join(e.References, ", "))
}

type StorageConfigProjectMismatchError struct {
	StorageID         string
	ExpectedProjectID string
	ActualProjectID   string
}

func (e *StorageConfigProjectMismatchError) Error() string {
	return fmt.Sprintf("storage config %s belongs to project %s, not %s", e.StorageID, e.ActualProjectID, e.ExpectedProjectID)
}

func (s *Service) ensureProjectExists(projectID string) error {
	if strings.TrimSpace(projectID) == "" {
		return fmt.Errorf("project_id is required")
	}
	if _, err := s.store.GetProject(projectID); err != nil {
		return fmt.Errorf("project not found: %w", err)
	}
	return nil
}

func (s *Service) getOwnedStorageConfig(projectID, storageID string) (*models.StorageConfig, error) {
	if strings.TrimSpace(projectID) == "" {
		return nil, fmt.Errorf("project_id is required")
	}
	storageConfig, err := s.store.GetStorageConfig(storageID)
	if err != nil {
		return nil, fmt.Errorf("storage config not found: %w", err)
	}
	if storageConfig.ProjectID != projectID {
		return nil, &StorageConfigProjectMismatchError{StorageID: storageID, ExpectedProjectID: projectID, ActualProjectID: storageConfig.ProjectID}
	}
	return storageConfig, nil
}

func (s *Service) GetOwnedStorageConfig(projectID, storageID string) (*models.StorageConfig, error) {
	return s.getOwnedStorageConfig(projectID, storageID)
}

func (s *Service) validateStoragePluginConfig(pluginType string, config map[string]interface{}) error {
	storageConfig := &models.StorageConfig{PluginType: pluginType, Config: config}
	if _, err := s.configuredPlugin(storageConfig); err != nil {
		return err
	}
	return nil
}

// DeleteStorageConfig deletes a storage configuration once no persisted project-owned resources still reference it.
func (s *Service) DeleteStorageConfig(storageID string) error {
	storageConfig, err := s.store.GetStorageConfig(storageID)
	if err != nil {
		return fmt.Errorf("storage config not found: %w", err)
	}

	references, err := s.findStorageReferences(storageConfig.ProjectID, storageID)
	if err != nil {
		return err
	}
	if len(references) > 0 {
		return &StorageConfigInUseError{StorageID: storageID, References: references}
	}
	if err := s.store.DeleteStorageConfig(storageID); err != nil {
		return fmt.Errorf("failed to delete storage config: %w", err)
	}

	log.Printf("Deleted storage config %s", storageID)
	return nil
}

func (s *Service) findStorageReferences(projectID, storageID string) ([]string, error) {
	references := make([]string, 0)

	pipelines, err := s.store.ListPipelinesByProject(projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list project pipelines: %w", err)
	}
	for _, pipeline := range pipelines {
		if jsonContainsExactString(pipeline, storageID) {
			references = append(references, fmt.Sprintf("pipeline %s", pipeline.ID))
		}
	}

	schedules, err := s.store.ListSchedulesByProject(projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list project schedules: %w", err)
	}
	for _, schedule := range schedules {
		if jsonContainsExactString(schedule, storageID) {
			references = append(references, fmt.Sprintf("schedule %s", schedule.ID))
		}
	}

	modelsList, err := s.store.ListMLModelsByProject(projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list project ml models: %w", err)
	}
	for _, model := range modelsList {
		if jsonContainsExactString(model, storageID) {
			references = append(references, fmt.Sprintf("ml model %s", model.ID))
		}
	}

	twins, err := s.store.ListDigitalTwinsByProject(projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list project digital twins: %w", err)
	}
	for _, twin := range twins {
		if jsonContainsExactString(twin, storageID) {
			references = append(references, fmt.Sprintf("digital twin %s", twin.ID))
		}
	}

	automations, err := s.store.ListAutomationsByProject(projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list project automations: %w", err)
	}
	for _, automation := range automations {
		if jsonContainsExactString(automation, storageID) {
			references = append(references, fmt.Sprintf("automation %s", automation.ID))
		}
	}

	return references, nil
}

func jsonContainsExactString(value any, needle string) bool {
	encoded, err := json.Marshal(value)
	if err != nil {
		return false
	}
	return strings.Contains(string(encoded), fmt.Sprintf("%q", needle))
}
