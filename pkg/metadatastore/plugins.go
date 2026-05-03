package metadatastore

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"time"
)

// SavePlugin saves a plugin and its binary data to the database
func (s *SQLiteStore) SavePlugin(plugin *models.Plugin, binaryData []byte) error {
	// Marshal plugin definition to JSON
	definitionJSON, err := json.Marshal(plugin.PluginDefinition)
	if err != nil {
		return fmt.Errorf("failed to marshal plugin definition: %w", err)
	}

	// Start transaction
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Save plugin metadata
	pluginQuery := `
		INSERT OR REPLACE INTO plugins 
		(id, name, version, description, author, repository_url, git_commit_hash, 
		 plugin_definition, binary_data, status, created_at, updated_at, last_loaded_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	var lastLoadedAt interface{}
	if plugin.LastLoadedAt != nil {
		lastLoadedAt = plugin.LastLoadedAt
	}

	_, err = tx.Exec(pluginQuery,
		plugin.ID,
		plugin.Name,
		plugin.Version,
		plugin.Description,
		plugin.Author,
		plugin.RepositoryURL,
		plugin.GitCommitHash,
		string(definitionJSON),
		binaryData,
		plugin.Status,
		plugin.CreatedAt,
		plugin.UpdatedAt,
		lastLoadedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to save plugin: %w", err)
	}

	// Delete existing actions for this plugin
	_, err = tx.Exec(`DELETE FROM plugin_actions WHERE plugin_id = ?`, plugin.ID)
	if err != nil {
		return fmt.Errorf("failed to delete old actions: %w", err)
	}

	// Save plugin actions
	actionQuery := `
		INSERT INTO plugin_actions (id, plugin_id, name, description, parameters, returns)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	for _, action := range plugin.Actions {
		parametersJSON, err := json.Marshal(action.Parameters)
		if err != nil {
			return fmt.Errorf("failed to marshal action parameters: %w", err)
		}

		returnsJSON, err := json.Marshal(action.Returns)
		if err != nil {
			return fmt.Errorf("failed to marshal action returns: %w", err)
		}

		_, err = tx.Exec(actionQuery,
			action.ID,
			action.PluginID,
			action.Name,
			action.Description,
			string(parametersJSON),
			string(returnsJSON),
		)
		if err != nil {
			return fmt.Errorf("failed to save action %s: %w", action.Name, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetPlugin retrieves a plugin by name
func (s *SQLiteStore) GetPlugin(name string) (*models.Plugin, error) {
	query := `
		SELECT id, name, version, description, author, repository_url, git_commit_hash,
		       plugin_definition, status, created_at, updated_at, last_loaded_at
		FROM plugins WHERE name = ?
	`

	var plugin models.Plugin
	var definitionJSON string
	var lastLoadedAt sql.NullTime

	err := s.db.QueryRow(query, name).Scan(
		&plugin.ID,
		&plugin.Name,
		&plugin.Version,
		&plugin.Description,
		&plugin.Author,
		&plugin.RepositoryURL,
		&plugin.GitCommitHash,
		&definitionJSON,
		&plugin.Status,
		&plugin.CreatedAt,
		&plugin.UpdatedAt,
		&lastLoadedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("plugin not found: %s", name)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get plugin: %w", err)
	}

	// Unmarshal plugin definition
	if err := json.Unmarshal([]byte(definitionJSON), &plugin.PluginDefinition); err != nil {
		return nil, fmt.Errorf("failed to unmarshal plugin definition: %w", err)
	}

	if lastLoadedAt.Valid {
		plugin.LastLoadedAt = &lastLoadedAt.Time
	}

	// Get actions
	actionsQuery := `
		SELECT id, plugin_id, name, description, parameters, returns
		FROM plugin_actions WHERE plugin_id = ?
	`

	rows, err := s.db.Query(actionsQuery, plugin.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to query actions: %w", err)
	}
	defer rows.Close()

	plugin.Actions = make([]models.PluginAction, 0)
	for rows.Next() {
		var action models.PluginAction
		var parametersJSON, returnsJSON string

		if err := rows.Scan(
			&action.ID,
			&action.PluginID,
			&action.Name,
			&action.Description,
			&parametersJSON,
			&returnsJSON,
		); err != nil {
			continue
		}

		if err := json.Unmarshal([]byte(parametersJSON), &action.Parameters); err != nil {
			continue
		}

		if err := json.Unmarshal([]byte(returnsJSON), &action.Returns); err != nil {
			continue
		}

		plugin.Actions = append(plugin.Actions, action)
	}

	return &plugin, nil
}

// GetPluginBinary retrieves just the binary data for a plugin
func (s *SQLiteStore) GetPluginBinary(name string) ([]byte, error) {
	query := `SELECT binary_data FROM plugins WHERE name = ?`

	var binaryData []byte
	err := s.db.QueryRow(query, name).Scan(&binaryData)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("plugin not found: %s", name)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get plugin binary: %w", err)
	}

	return binaryData, nil
}

// ListPlugins retrieves all plugins
func (s *SQLiteStore) ListPlugins() ([]*models.Plugin, error) {
	query := `
		SELECT id, name, version, description, author, repository_url, git_commit_hash,
		       plugin_definition, status, created_at, updated_at, last_loaded_at
		FROM plugins ORDER BY name
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list plugins: %w", err)
	}
	defer rows.Close()

	plugins := make([]*models.Plugin, 0)
	for rows.Next() {
		var plugin models.Plugin
		var definitionJSON string
		var lastLoadedAt sql.NullTime

		if err := rows.Scan(
			&plugin.ID,
			&plugin.Name,
			&plugin.Version,
			&plugin.Description,
			&plugin.Author,
			&plugin.RepositoryURL,
			&plugin.GitCommitHash,
			&definitionJSON,
			&plugin.Status,
			&plugin.CreatedAt,
			&plugin.UpdatedAt,
			&lastLoadedAt,
		); err != nil {
			continue
		}

		if err := json.Unmarshal([]byte(definitionJSON), &plugin.PluginDefinition); err != nil {
			continue
		}

		if lastLoadedAt.Valid {
			plugin.LastLoadedAt = &lastLoadedAt.Time
		}

		// Get action count (lightweight - don't load full action details for list)
		var actionCount int
		s.db.QueryRow(`SELECT COUNT(*) FROM plugin_actions WHERE plugin_id = ?`, plugin.ID).Scan(&actionCount)

		// We'll populate a summary in the actions array
		plugin.Actions = make([]models.PluginAction, 0)

		plugins = append(plugins, &plugin)
	}

	return plugins, nil
}

// DeletePlugin deletes a plugin and its actions
func (s *SQLiteStore) DeletePlugin(name string) error {
	query := `DELETE FROM plugins WHERE name = ?`
	result, err := s.db.Exec(query, name)
	if err != nil {
		return fmt.Errorf("failed to delete plugin: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("plugin not found: %s", name)
	}

	return nil
}

// UpdatePluginStatus updates the status of a plugin
func (s *SQLiteStore) UpdatePluginStatus(name string, status models.PluginStatus) error {
	query := `UPDATE plugins SET status = ?, updated_at = ? WHERE name = ?`
	result, err := s.db.Exec(query, status, time.Now(), name)
	if err != nil {
		return fmt.Errorf("failed to update plugin status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("plugin not found: %s", name)
	}

	return nil
}

// ── External Storage Plugins ─────────────────────────────────────────────────

// SaveExternalStoragePlugin upserts an external storage plugin record.
func (s *SQLiteStore) SaveExternalStoragePlugin(plugin *models.ExternalStoragePlugin) error {
	query := `
		INSERT OR REPLACE INTO external_storage_plugins
		(name, version, description, author, repository_url, git_commit_hash,
		 status, error_message, installed_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	return s.retryOnBusy(func() error {
		_, err := s.db.Exec(query,
			plugin.Name, plugin.Version, plugin.Description, plugin.Author,
			plugin.RepositoryURL, plugin.GitCommitHash,
			plugin.Status, plugin.ErrorMessage,
			plugin.InstalledAt.UTC(), plugin.UpdatedAt.UTC(),
		)
		return err
	}, 5)
}

// GetExternalStoragePlugin retrieves a single external storage plugin by name.
func (s *SQLiteStore) GetExternalStoragePlugin(name string) (*models.ExternalStoragePlugin, error) {
	query := `
		SELECT name, version, description, author, repository_url, git_commit_hash,
		       status, error_message, installed_at, updated_at
		FROM external_storage_plugins WHERE name = ?`
	row := s.db.QueryRow(query, name)
	p := &models.ExternalStoragePlugin{}
	if err := row.Scan(
		&p.Name, &p.Version, &p.Description, &p.Author,
		&p.RepositoryURL, &p.GitCommitHash,
		&p.Status, &p.ErrorMessage,
		&p.InstalledAt, &p.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("external storage plugin not found: %w", err)
	}
	return p, nil
}

// ListExternalStoragePlugins returns all external storage plugin records.
func (s *SQLiteStore) ListExternalStoragePlugins() ([]*models.ExternalStoragePlugin, error) {
	query := `
		SELECT name, version, description, author, repository_url, git_commit_hash,
		       status, error_message, installed_at, updated_at
		FROM external_storage_plugins ORDER BY name`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list external storage plugins: %w", err)
	}
	defer rows.Close()
	var plugins []*models.ExternalStoragePlugin
	for rows.Next() {
		p := &models.ExternalStoragePlugin{}
		if err := rows.Scan(
			&p.Name, &p.Version, &p.Description, &p.Author,
			&p.RepositoryURL, &p.GitCommitHash,
			&p.Status, &p.ErrorMessage,
			&p.InstalledAt, &p.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan external storage plugin: %w", err)
		}
		plugins = append(plugins, p)
	}
	return plugins, nil
}

// DeleteExternalStoragePlugin removes an external storage plugin record.
func (s *SQLiteStore) DeleteExternalStoragePlugin(name string) error {
	return s.retryOnBusy(func() error {
		_, err := s.db.Exec(`DELETE FROM external_storage_plugins WHERE name = ?`, name)
		return err
	}, 5)
}

// ── External LLM Providers ────────────────────────────────────────────────────

// SaveExternalLLMProvider upserts an external LLM provider record.
func (s *SQLiteStore) SaveExternalLLMProvider(p *models.ExternalLLMProvider) error {
	query := `
		INSERT OR REPLACE INTO external_llm_providers
		(name, repository_url, git_commit_hash, status, error_message, installed_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`
	return s.retryOnBusy(func() error {
		_, err := s.db.Exec(query,
			p.Name, p.RepositoryURL, p.GitCommitHash,
			p.Status, p.ErrorMessage,
			p.InstalledAt, p.UpdatedAt,
		)
		return err
	}, 5)
}

// GetExternalLLMProvider retrieves a single external LLM provider by name.
func (s *SQLiteStore) GetExternalLLMProvider(name string) (*models.ExternalLLMProvider, error) {
	query := `
		SELECT name, repository_url, git_commit_hash, status, error_message, installed_at, updated_at
		FROM external_llm_providers WHERE name = ?`
	row := s.db.QueryRow(query, name)
	p := &models.ExternalLLMProvider{}
	if err := row.Scan(
		&p.Name, &p.RepositoryURL, &p.GitCommitHash,
		&p.Status, &p.ErrorMessage,
		&p.InstalledAt, &p.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("external LLM provider not found: %w", err)
	}
	return p, nil
}

// ListExternalLLMProviders returns all external LLM provider records.
func (s *SQLiteStore) ListExternalLLMProviders() ([]*models.ExternalLLMProvider, error) {
	query := `
		SELECT name, repository_url, git_commit_hash, status, error_message, installed_at, updated_at
		FROM external_llm_providers ORDER BY name`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list external LLM providers: %w", err)
	}
	defer rows.Close()
	var providers []*models.ExternalLLMProvider
	for rows.Next() {
		p := &models.ExternalLLMProvider{}
		if err := rows.Scan(
			&p.Name, &p.RepositoryURL, &p.GitCommitHash,
			&p.Status, &p.ErrorMessage,
			&p.InstalledAt, &p.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan external LLM provider: %w", err)
		}
		providers = append(providers, p)
	}
	return providers, nil
}

// DeleteExternalLLMProvider removes an external LLM provider record.
func (s *SQLiteStore) DeleteExternalLLMProvider(name string) error {
	return s.retryOnBusy(func() error {
		_, err := s.db.Exec(`DELETE FROM external_llm_providers WHERE name = ?`, name)
		return err
	}, 5)
}
