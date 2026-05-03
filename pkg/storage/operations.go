package storage

import (
	"fmt"
	"log"

	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

func (s *Service) StoreForProject(projectID, storageID string, cir *models.CIR) (*models.StorageResult, error) {
	storageConfig, err := s.getOwnedStorageConfig(projectID, storageID)
	if err != nil {
		return nil, err
	}
	return s.storeWithConfig(storageConfig, cir)
}

func (s *Service) RetrieveForProject(projectID, storageID string, query *models.CIRQuery) ([]*models.CIR, error) {
	storageConfig, err := s.getOwnedStorageConfig(projectID, storageID)
	if err != nil {
		return nil, err
	}
	return s.retrieveWithConfig(storageConfig, query)
}

func (s *Service) UpdateForProject(projectID, storageID string, query *models.CIRQuery, updates *models.CIRUpdate) (*models.StorageResult, error) {
	storageConfig, err := s.getOwnedStorageConfig(projectID, storageID)
	if err != nil {
		return nil, err
	}
	return s.updateWithConfig(storageConfig, query, updates)
}

func (s *Service) DeleteForProject(projectID, storageID string, query *models.CIRQuery) (*models.StorageResult, error) {
	storageConfig, err := s.getOwnedStorageConfig(projectID, storageID)
	if err != nil {
		return nil, err
	}
	return s.deleteWithConfig(storageConfig, query)
}

func (s *Service) GetStorageMetadataForProject(projectID, storageID string) (*models.StorageMetadata, error) {
	storageConfig, err := s.getOwnedStorageConfig(projectID, storageID)
	if err != nil {
		return nil, err
	}
	return s.metadataWithConfig(storageConfig)
}

func (s *Service) HealthCheckForProject(projectID, storageID string) (bool, error) {
	storageConfig, err := s.getOwnedStorageConfig(projectID, storageID)
	if err != nil {
		return false, err
	}
	return s.healthWithConfig(storageConfig)
}

func (s *Service) storeWithConfig(storageConfig *models.StorageConfig, cir *models.CIR) (*models.StorageResult, error) {
	if err := cir.Validate(); err != nil {
		return nil, fmt.Errorf("invalid CIR: %w", err)
	}
	if !storageConfig.Active {
		return nil, fmt.Errorf("storage config is not active")
	}
	plugin, err := s.configuredPlugin(storageConfig)
	if err != nil {
		return nil, err
	}
	if storageConfig.OntologyID != "" {
		compiled, err := s.store.GetCompiledOntology(storageConfig.OntologyID)
		if err != nil {
			return nil, fmt.Errorf("compiled ontology not found for storage %s: %w", storageConfig.ID, err)
		}
		mapping, err := mapCIRToOntology(cir, compiled)
		if err != nil {
			return nil, fmt.Errorf("failed to map CIR to ontology: %w", err)
		}
		cir.Metadata.Ontology = mapping
	}
	result, err := plugin.Store(cir)
	if err != nil {
		return nil, fmt.Errorf("failed to store data: %w", err)
	}
	log.Printf("Stored CIR data in storage %s, affected items: %d", storageConfig.ID, result.AffectedItems)
	return result, nil
}

func (s *Service) retrieveWithConfig(storageConfig *models.StorageConfig, query *models.CIRQuery) ([]*models.CIR, error) {
	if !storageConfig.Active {
		return nil, fmt.Errorf("storage config is not active")
	}
	plugin, err := s.configuredPlugin(storageConfig)
	if err != nil {
		return nil, err
	}
	results, err := plugin.Retrieve(query)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve data: %w", err)
	}
	log.Printf("Retrieved %d CIR objects from storage %s", len(results), storageConfig.ID)
	return results, nil
}

func (s *Service) updateWithConfig(storageConfig *models.StorageConfig, query *models.CIRQuery, updates *models.CIRUpdate) (*models.StorageResult, error) {
	if !storageConfig.Active {
		return nil, fmt.Errorf("storage config is not active")
	}
	plugin, err := s.configuredPlugin(storageConfig)
	if err != nil {
		return nil, err
	}
	result, err := plugin.Update(query, updates)
	if err != nil {
		return nil, fmt.Errorf("failed to update data: %w", err)
	}
	log.Printf("Updated data in storage %s, affected items: %d", storageConfig.ID, result.AffectedItems)
	return result, nil
}

func (s *Service) deleteWithConfig(storageConfig *models.StorageConfig, query *models.CIRQuery) (*models.StorageResult, error) {
	if !storageConfig.Active {
		return nil, fmt.Errorf("storage config is not active")
	}
	plugin, err := s.configuredPlugin(storageConfig)
	if err != nil {
		return nil, err
	}
	result, err := plugin.Delete(query)
	if err != nil {
		return nil, fmt.Errorf("failed to delete data: %w", err)
	}
	log.Printf("Deleted data from storage %s, affected items: %d", storageConfig.ID, result.AffectedItems)
	return result, nil
}

func (s *Service) metadataWithConfig(storageConfig *models.StorageConfig) (*models.StorageMetadata, error) {
	plugin, err := s.configuredPlugin(storageConfig)
	if err != nil {
		return nil, err
	}
	return plugin.GetMetadata()
}

func (s *Service) healthWithConfig(storageConfig *models.StorageConfig) (bool, error) {
	plugin, err := s.configuredPlugin(storageConfig)
	if err != nil {
		return false, err
	}
	return plugin.HealthCheck()
}

// Store stores CIR data in the specified storage
func (s *Service) Store(storageID string, cir *models.CIR) (*models.StorageResult, error) {
	storageConfig, err := s.store.GetStorageConfig(storageID)
	if err != nil {
		return nil, fmt.Errorf("storage config not found: %w", err)
	}
	return s.storeWithConfig(storageConfig, cir)
}

// Retrieve retrieves data from storage using a query
func (s *Service) Retrieve(storageID string, query *models.CIRQuery) ([]*models.CIR, error) {
	storageConfig, err := s.store.GetStorageConfig(storageID)
	if err != nil {
		return nil, fmt.Errorf("storage config not found: %w", err)
	}
	return s.retrieveWithConfig(storageConfig, query)
}

// Update updates data in storage
func (s *Service) Update(storageID string, query *models.CIRQuery, updates *models.CIRUpdate) (*models.StorageResult, error) {
	storageConfig, err := s.store.GetStorageConfig(storageID)
	if err != nil {
		return nil, fmt.Errorf("storage config not found: %w", err)
	}
	return s.updateWithConfig(storageConfig, query, updates)
}

// Delete deletes data from storage
func (s *Service) Delete(storageID string, query *models.CIRQuery) (*models.StorageResult, error) {
	storageConfig, err := s.store.GetStorageConfig(storageID)
	if err != nil {
		return nil, fmt.Errorf("storage config not found: %w", err)
	}
	return s.deleteWithConfig(storageConfig, query)
}

// GetStorageMetadata retrieves metadata about the storage system
func (s *Service) GetStorageMetadata(storageID string) (*models.StorageMetadata, error) {
	storageConfig, err := s.store.GetStorageConfig(storageID)
	if err != nil {
		return nil, fmt.Errorf("storage config not found: %w", err)
	}
	return s.metadataWithConfig(storageConfig)
}

// HealthCheck performs a health check on the storage
func (s *Service) HealthCheck(storageID string) (bool, error) {
	storageConfig, err := s.store.GetStorageConfig(storageID)
	if err != nil {
		return false, fmt.Errorf("storage config not found: %w", err)
	}
	return s.healthWithConfig(storageConfig)
}
