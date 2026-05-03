package metadatastore

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"time"
)

// SaveStorageConfig saves a storage configuration to the database
func (s *SQLiteStore) SaveStorageConfig(config *models.StorageConfig) error {
	configJSON, err := json.Marshal(config.Config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	active := 0
	if config.Active {
		active = 1
	}

	query := `
		INSERT OR REPLACE INTO storage_configs (id, project_id, plugin_type, ontology_id, active, created_at, updated_at, config)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = s.db.Exec(query,
		config.ID,
		config.ProjectID,
		config.PluginType,
		config.OntologyID,
		active,
		config.CreatedAt,
		config.UpdatedAt,
		string(configJSON),
	)

	if err != nil {
		return fmt.Errorf("failed to save storage config: %w", err)
	}

	return nil
}

// GetStorageConfig retrieves a storage configuration by ID
func (s *SQLiteStore) GetStorageConfig(id string) (*models.StorageConfig, error) {
	var configJSON string
	var active int
	var config models.StorageConfig

	query := `SELECT id, project_id, plugin_type, ontology_id, active, created_at, updated_at, config FROM storage_configs WHERE id = ?`

	err := s.db.QueryRow(query, id).Scan(
		&config.ID,
		&config.ProjectID,
		&config.PluginType,
		&config.OntologyID,
		&active,
		&config.CreatedAt,
		&config.UpdatedAt,
		&configJSON,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("storage config not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get storage config: %w", err)
	}

	config.Active = active == 1

	if err := json.Unmarshal([]byte(configJSON), &config.Config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}

// ListStorageConfigs lists all storage configurations
func (s *SQLiteStore) ListStorageConfigs() ([]*models.StorageConfig, error) {
	query := `SELECT id, project_id, plugin_type, ontology_id, active, created_at, updated_at, config FROM storage_configs ORDER BY created_at DESC`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list storage configs: %w", err)
	}
	defer rows.Close()

	configs := make([]*models.StorageConfig, 0)
	for rows.Next() {
		var configJSON string
		var active int
		var config models.StorageConfig

		if err := rows.Scan(
			&config.ID,
			&config.ProjectID,
			&config.PluginType,
			&config.OntologyID,
			&active,
			&config.CreatedAt,
			&config.UpdatedAt,
			&configJSON,
		); err != nil {
			continue
		}

		config.Active = active == 1

		if err := json.Unmarshal([]byte(configJSON), &config.Config); err != nil {
			continue
		}

		configs = append(configs, &config)
	}

	return configs, nil
}

// ListStorageConfigsByProject lists storage configurations for a specific project
func (s *SQLiteStore) ListStorageConfigsByProject(projectID string) ([]*models.StorageConfig, error) {
	query := `SELECT id, project_id, plugin_type, ontology_id, active, created_at, updated_at, config FROM storage_configs WHERE project_id = ? ORDER BY created_at DESC`

	rows, err := s.db.Query(query, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list storage configs for project: %w", err)
	}
	defer rows.Close()

	configs := make([]*models.StorageConfig, 0)
	for rows.Next() {
		var configJSON string
		var active int
		var config models.StorageConfig

		if err := rows.Scan(
			&config.ID,
			&config.ProjectID,
			&config.PluginType,
			&config.OntologyID,
			&active,
			&config.CreatedAt,
			&config.UpdatedAt,
			&configJSON,
		); err != nil {
			continue
		}

		config.Active = active == 1

		if err := json.Unmarshal([]byte(configJSON), &config.Config); err != nil {
			continue
		}

		configs = append(configs, &config)
	}

	return configs, nil
}

// DeleteStorageConfig deletes a storage configuration
func (s *SQLiteStore) DeleteStorageConfig(id string) error {
	query := `DELETE FROM storage_configs WHERE id = ?`
	_, err := s.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete storage config: %w", err)
	}
	return nil
}

// SaveOntology saves an ontology to the database
func (s *SQLiteStore) SaveOntology(ontology *models.Ontology) error {
	isGenerated := 0
	if ontology.IsGenerated {
		isGenerated = 1
	}

	query := `
		INSERT OR REPLACE INTO ontologies (id, project_id, name, description, version, content, status, is_generated, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.Exec(query,
		ontology.ID,
		ontology.ProjectID,
		ontology.Name,
		ontology.Description,
		ontology.Version,
		ontology.Content,
		ontology.Status,
		isGenerated,
		ontology.CreatedAt.Format(time.RFC3339),
		ontology.UpdatedAt.Format(time.RFC3339),
	)

	if err != nil {
		return fmt.Errorf("failed to save ontology: %w", err)
	}

	return nil
}

// GetOntology retrieves an ontology by ID
func (s *SQLiteStore) GetOntology(id string) (*models.Ontology, error) {
	var ontology models.Ontology
	var isGenerated int
	var createdAt, updatedAt string

	query := `SELECT id, project_id, name, description, version, content, status, is_generated, created_at, updated_at FROM ontologies WHERE id = ?`

	err := s.db.QueryRow(query, id).Scan(
		&ontology.ID,
		&ontology.ProjectID,
		&ontology.Name,
		&ontology.Description,
		&ontology.Version,
		&ontology.Content,
		&ontology.Status,
		&isGenerated,
		&createdAt,
		&updatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("ontology not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get ontology: %w", err)
	}

	ontology.IsGenerated = isGenerated == 1

	// Parse timestamps
	if ontology.CreatedAt, err = time.Parse(time.RFC3339, createdAt); err != nil {
		return nil, fmt.Errorf("failed to parse created_at: %w", err)
	}
	if ontology.UpdatedAt, err = time.Parse(time.RFC3339, updatedAt); err != nil {
		return nil, fmt.Errorf("failed to parse updated_at: %w", err)
	}

	return &ontology, nil
}

// ListOntologies lists all ontologies
func (s *SQLiteStore) ListOntologies() ([]*models.Ontology, error) {
	query := `SELECT id, project_id, name, description, version, content, status, is_generated, created_at, updated_at FROM ontologies ORDER BY created_at DESC`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list ontologies: %w", err)
	}
	defer rows.Close()

	ontologies := make([]*models.Ontology, 0)
	for rows.Next() {
		var ontology models.Ontology
		var isGenerated int
		var createdAt, updatedAt string

		if err := rows.Scan(
			&ontology.ID,
			&ontology.ProjectID,
			&ontology.Name,
			&ontology.Description,
			&ontology.Version,
			&ontology.Content,
			&ontology.Status,
			&isGenerated,
			&createdAt,
			&updatedAt,
		); err != nil {
			continue
		}

		ontology.IsGenerated = isGenerated == 1

		// Parse timestamps
		if ontology.CreatedAt, err = time.Parse(time.RFC3339, createdAt); err != nil {
			continue
		}
		if ontology.UpdatedAt, err = time.Parse(time.RFC3339, updatedAt); err != nil {
			continue
		}

		ontologies = append(ontologies, &ontology)
	}

	return ontologies, nil
}

// ListOntologiesByProject lists ontologies for a specific project
func (s *SQLiteStore) ListOntologiesByProject(projectID string) ([]*models.Ontology, error) {
	query := `SELECT id, project_id, name, description, version, content, status, is_generated, created_at, updated_at FROM ontologies WHERE project_id = ? ORDER BY created_at DESC`

	rows, err := s.db.Query(query, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list ontologies for project: %w", err)
	}
	defer rows.Close()

	ontologies := make([]*models.Ontology, 0)
	for rows.Next() {
		var ontology models.Ontology
		var isGenerated int
		var createdAt, updatedAt string

		if err := rows.Scan(
			&ontology.ID,
			&ontology.ProjectID,
			&ontology.Name,
			&ontology.Description,
			&ontology.Version,
			&ontology.Content,
			&ontology.Status,
			&isGenerated,
			&createdAt,
			&updatedAt,
		); err != nil {
			continue
		}

		ontology.IsGenerated = isGenerated == 1

		// Parse timestamps
		if ontology.CreatedAt, err = time.Parse(time.RFC3339, createdAt); err != nil {
			continue
		}
		if ontology.UpdatedAt, err = time.Parse(time.RFC3339, updatedAt); err != nil {
			continue
		}

		ontologies = append(ontologies, &ontology)
	}

	return ontologies, nil
}

// DeleteOntology deletes an ontology
func (s *SQLiteStore) DeleteOntology(id string) error {
	query := `DELETE FROM ontologies WHERE id = ?`
	_, err := s.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete ontology: %w", err)
	}
	return nil
}

// SaveMLModel saves an ML model to the database
func (s *SQLiteStore) SaveMLModel(model *models.MLModel) error {
	data, err := json.Marshal(model)
	if err != nil {
		return fmt.Errorf("failed to marshal ML model: %w", err)
	}

	trainedAtStr := ""
	if model.TrainedAt != nil {
		trainedAtStr = model.TrainedAt.Format(time.RFC3339)
	}

	query := `
		INSERT OR REPLACE INTO ml_models (
			id, project_id, ontology_id, name, description, type, status, version,
			is_recommended, recommendation_score, model_artifact_path,
			created_at, updated_at, trained_at, data
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = s.db.Exec(query,
		model.ID,
		model.ProjectID,
		model.OntologyID,
		model.Name,
		model.Description,
		model.Type,
		model.Status,
		model.Version,
		model.IsRecommended,
		model.RecommendationScore,
		model.ModelArtifactPath,
		model.CreatedAt.Format(time.RFC3339),
		model.UpdatedAt.Format(time.RFC3339),
		trainedAtStr,
		data,
	)

	if err != nil {
		return fmt.Errorf("failed to save ML model: %w", err)
	}

	return nil
}

// GetMLModel retrieves an ML model by ID
func (s *SQLiteStore) GetMLModel(id string) (*models.MLModel, error) {
	query := `SELECT data FROM ml_models WHERE id = ?`

	var data []byte
	err := s.db.QueryRow(query, id).Scan(&data)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("ML model not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get ML model: %w", err)
	}

	var model models.MLModel
	if err := json.Unmarshal(data, &model); err != nil {
		return nil, fmt.Errorf("failed to unmarshal ML model: %w", err)
	}

	return &model, nil
}

// ListMLModels lists all ML models
func (s *SQLiteStore) ListMLModels() ([]*models.MLModel, error) {
	query := `SELECT data FROM ml_models ORDER BY created_at DESC`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list ML models: %w", err)
	}
	defer rows.Close()

	var mlModels []*models.MLModel
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, fmt.Errorf("failed to scan ML model: %w", err)
		}

		var model models.MLModel
		if err := json.Unmarshal(data, &model); err != nil {
			return nil, fmt.Errorf("failed to unmarshal ML model: %w", err)
		}

		mlModels = append(mlModels, &model)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating ML models: %w", err)
	}

	return mlModels, nil
}

// ListMLModelsByProject lists all ML models for a specific project
func (s *SQLiteStore) ListMLModelsByProject(projectID string) ([]*models.MLModel, error) {
	query := `SELECT data FROM ml_models WHERE project_id = ? ORDER BY created_at DESC`

	rows, err := s.db.Query(query, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list ML models: %w", err)
	}
	defer rows.Close()

	var mlModels []*models.MLModel
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, fmt.Errorf("failed to scan ML model: %w", err)
		}

		var model models.MLModel
		if err := json.Unmarshal(data, &model); err != nil {
			return nil, fmt.Errorf("failed to unmarshal ML model: %w", err)
		}

		mlModels = append(mlModels, &model)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating ML models: %w", err)
	}

	return mlModels, nil
}

// DeleteMLModel deletes an ML model
func (s *SQLiteStore) DeleteMLModel(id string) error {
	query := `DELETE FROM ml_models WHERE id = ?`
	_, err := s.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete ML model: %w", err)
	}
	return nil
}
