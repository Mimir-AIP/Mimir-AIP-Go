package storage

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/mimir-aip/mimir-aip-go/pkg/metadatastore"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

// Service provides storage management operations
type Service struct {
	store   metadatastore.MetadataStore
	plugins map[string]models.StoragePlugin
	mu      sync.RWMutex
}

// NewService creates a new storage service
func NewService(store metadatastore.MetadataStore) *Service {
	return &Service{
		store:   store,
		plugins: make(map[string]models.StoragePlugin),
	}
}

// RegisterPlugin registers a storage plugin
func (s *Service) RegisterPlugin(pluginType string, plugin models.StoragePlugin) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.plugins[pluginType] = plugin
	log.Printf("Registered storage plugin: %s", pluginType)
}

// GetPlugin retrieves a registered plugin by type
func (s *Service) GetPlugin(pluginType string) (models.StoragePlugin, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	plugin, ok := s.plugins[pluginType]
	if !ok {
		return nil, fmt.Errorf("storage plugin not found: %s", pluginType)
	}

	return plugin, nil
}

// CreateStorageConfig creates a new storage configuration for a project
func (s *Service) CreateStorageConfig(projectID, pluginType string, config map[string]interface{}) (*models.StorageConfig, error) {
	// Validate plugin type
	if _, err := s.GetPlugin(pluginType); err != nil {
		return nil, fmt.Errorf("invalid plugin type: %w", err)
	}

	now := time.Now().Format(time.RFC3339)
	storageConfig := &models.StorageConfig{
		ID:         uuid.New().String(),
		ProjectID:  projectID,
		PluginType: pluginType,
		Config:     config,
		Active:     true,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	// Save storage config
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

// DeleteStorageConfig deletes a storage configuration
func (s *Service) DeleteStorageConfig(storageID string) error {
	if err := s.store.DeleteStorageConfig(storageID); err != nil {
		return fmt.Errorf("failed to delete storage config: %w", err)
	}

	log.Printf("Deleted storage config %s", storageID)

	return nil
}

// InitializeStorage initializes storage for a project using the specified configuration
func (s *Service) InitializeStorage(storageID string, ontology *models.OntologyDefinition) error {
	storageConfig, err := s.store.GetStorageConfig(storageID)
	if err != nil {
		return fmt.Errorf("storage config not found: %w", err)
	}

	plugin, err := s.GetPlugin(storageConfig.PluginType)
	if err != nil {
		return err
	}

	// Initialize plugin
	pluginConfig := &models.PluginConfig{
		ConnectionString: getConnectionString(storageConfig.Config),
		Credentials:      getCredentials(storageConfig.Config),
		Options:          getOptions(storageConfig.Config),
	}

	if err := plugin.Initialize(pluginConfig); err != nil {
		return fmt.Errorf("failed to initialize storage plugin: %w", err)
	}

	// Create schema if ontology is provided
	if ontology != nil {
		if err := plugin.CreateSchema(ontology); err != nil {
			return fmt.Errorf("failed to create storage schema: %w", err)
		}
		storageConfig.OntologyID = ontology.Entities[0].Name // Simplified - should store ontology separately
		storageConfig.UpdatedAt = time.Now().Format(time.RFC3339)
		if err := s.store.SaveStorageConfig(storageConfig); err != nil {
			return fmt.Errorf("failed to update storage config: %w", err)
		}
	}

	log.Printf("Initialized storage %s with plugin %s", storageID, storageConfig.PluginType)

	return nil
}

// Store stores CIR data in the specified storage
func (s *Service) Store(storageID string, cir *models.CIR) (*models.StorageResult, error) {
	// Validate CIR
	if err := cir.Validate(); err != nil {
		return nil, fmt.Errorf("invalid CIR: %w", err)
	}

	storageConfig, err := s.store.GetStorageConfig(storageID)
	if err != nil {
		return nil, fmt.Errorf("storage config not found: %w", err)
	}

	if !storageConfig.Active {
		return nil, fmt.Errorf("storage config is not active")
	}

	plugin, err := s.GetPlugin(storageConfig.PluginType)
	if err != nil {
		return nil, err
	}

	// Initialize plugin if not already done
	pluginConfig := &models.PluginConfig{
		ConnectionString: getConnectionString(storageConfig.Config),
		Credentials:      getCredentials(storageConfig.Config),
		Options:          getOptions(storageConfig.Config),
	}

	if err := plugin.Initialize(pluginConfig); err != nil {
		return nil, fmt.Errorf("failed to initialize storage plugin: %w", err)
	}

	// Store data
	result, err := plugin.Store(cir)
	if err != nil {
		return nil, fmt.Errorf("failed to store data: %w", err)
	}

	log.Printf("Stored CIR data in storage %s, affected items: %d", storageID, result.AffectedItems)

	return result, nil
}

// Retrieve retrieves data from storage using a query
func (s *Service) Retrieve(storageID string, query *models.CIRQuery) ([]*models.CIR, error) {
	storageConfig, err := s.store.GetStorageConfig(storageID)
	if err != nil {
		return nil, fmt.Errorf("storage config not found: %w", err)
	}

	if !storageConfig.Active {
		return nil, fmt.Errorf("storage config is not active")
	}

	plugin, err := s.GetPlugin(storageConfig.PluginType)
	if err != nil {
		return nil, err
	}

	// Initialize plugin
	pluginConfig := &models.PluginConfig{
		ConnectionString: getConnectionString(storageConfig.Config),
		Credentials:      getCredentials(storageConfig.Config),
		Options:          getOptions(storageConfig.Config),
	}

	if err := plugin.Initialize(pluginConfig); err != nil {
		return nil, fmt.Errorf("failed to initialize storage plugin: %w", err)
	}

	// Retrieve data
	results, err := plugin.Retrieve(query)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve data: %w", err)
	}

	log.Printf("Retrieved %d CIR objects from storage %s", len(results), storageID)

	return results, nil
}

// Update updates data in storage
func (s *Service) Update(storageID string, query *models.CIRQuery, updates *models.CIRUpdate) (*models.StorageResult, error) {
	storageConfig, err := s.store.GetStorageConfig(storageID)
	if err != nil {
		return nil, fmt.Errorf("storage config not found: %w", err)
	}

	if !storageConfig.Active {
		return nil, fmt.Errorf("storage config is not active")
	}

	plugin, err := s.GetPlugin(storageConfig.PluginType)
	if err != nil {
		return nil, err
	}

	// Initialize plugin
	pluginConfig := &models.PluginConfig{
		ConnectionString: getConnectionString(storageConfig.Config),
		Credentials:      getCredentials(storageConfig.Config),
		Options:          getOptions(storageConfig.Config),
	}

	if err := plugin.Initialize(pluginConfig); err != nil {
		return nil, fmt.Errorf("failed to initialize storage plugin: %w", err)
	}

	// Update data
	result, err := plugin.Update(query, updates)
	if err != nil {
		return nil, fmt.Errorf("failed to update data: %w", err)
	}

	log.Printf("Updated data in storage %s, affected items: %d", storageID, result.AffectedItems)

	return result, nil
}

// Delete deletes data from storage
func (s *Service) Delete(storageID string, query *models.CIRQuery) (*models.StorageResult, error) {
	storageConfig, err := s.store.GetStorageConfig(storageID)
	if err != nil {
		return nil, fmt.Errorf("storage config not found: %w", err)
	}

	if !storageConfig.Active {
		return nil, fmt.Errorf("storage config is not active")
	}

	plugin, err := s.GetPlugin(storageConfig.PluginType)
	if err != nil {
		return nil, err
	}

	// Initialize plugin
	pluginConfig := &models.PluginConfig{
		ConnectionString: getConnectionString(storageConfig.Config),
		Credentials:      getCredentials(storageConfig.Config),
		Options:          getOptions(storageConfig.Config),
	}

	if err := plugin.Initialize(pluginConfig); err != nil {
		return nil, fmt.Errorf("failed to initialize storage plugin: %w", err)
	}

	// Delete data
	result, err := plugin.Delete(query)
	if err != nil {
		return nil, fmt.Errorf("failed to delete data: %w", err)
	}

	log.Printf("Deleted data from storage %s, affected items: %d", storageID, result.AffectedItems)

	return result, nil
}

// GetStorageMetadata retrieves metadata about the storage system
func (s *Service) GetStorageMetadata(storageID string) (*models.StorageMetadata, error) {
	storageConfig, err := s.store.GetStorageConfig(storageID)
	if err != nil {
		return nil, fmt.Errorf("storage config not found: %w", err)
	}

	plugin, err := s.GetPlugin(storageConfig.PluginType)
	if err != nil {
		return nil, err
	}

	// Initialize plugin
	pluginConfig := &models.PluginConfig{
		ConnectionString: getConnectionString(storageConfig.Config),
		Credentials:      getCredentials(storageConfig.Config),
		Options:          getOptions(storageConfig.Config),
	}

	if err := plugin.Initialize(pluginConfig); err != nil {
		return nil, fmt.Errorf("failed to initialize storage plugin: %w", err)
	}

	return plugin.GetMetadata()
}

// HealthCheck performs a health check on the storage
func (s *Service) HealthCheck(storageID string) (bool, error) {
	storageConfig, err := s.store.GetStorageConfig(storageID)
	if err != nil {
		return false, fmt.Errorf("storage config not found: %w", err)
	}

	plugin, err := s.GetPlugin(storageConfig.PluginType)
	if err != nil {
		return false, err
	}

	// Initialize plugin
	pluginConfig := &models.PluginConfig{
		ConnectionString: getConnectionString(storageConfig.Config),
		Credentials:      getCredentials(storageConfig.Config),
		Options:          getOptions(storageConfig.Config),
	}

	if err := plugin.Initialize(pluginConfig); err != nil {
		return false, fmt.Errorf("failed to initialize storage plugin: %w", err)
	}

	return plugin.HealthCheck()
}

// Helper functions to extract config values

func getConnectionString(config map[string]interface{}) string {
	if cs, ok := config["connection_string"].(string); ok {
		return cs
	}
	return ""
}

func getCredentials(config map[string]interface{}) map[string]interface{} {
	if creds, ok := config["credentials"].(map[string]interface{}); ok {
		return creds
	}
	return make(map[string]interface{})
}

func getOptions(config map[string]interface{}) map[string]interface{} {
	if opts, ok := config["options"].(map[string]interface{}); ok {
		return opts
	}
	return make(map[string]interface{})
}
