package metadatastore

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "modernc.org/sqlite"

	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

// SQLiteStore provides SQLite-based persistence for projects, pipelines, and schedules
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore creates a new SQLite-based storage instance
func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	// Open database with connection pooling parameters
	// Format: file:path?param=value
	dsn := fmt.Sprintf("file:%s?_busy_timeout=10000&_journal_mode=WAL&_synchronous=NORMAL", dbPath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	// SetMaxOpenConns: Maximum number of open connections to the database
	// For SQLite, we want this relatively low since writes are serialized anyway
	db.SetMaxOpenConns(10)

	// SetMaxIdleConns: Maximum number of connections in the idle connection pool
	db.SetMaxIdleConns(5)

	// SetConnMaxLifetime: Maximum amount of time a connection may be reused
	db.SetConnMaxLifetime(time.Hour)

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	store := &SQLiteStore{db: db}

	// Verify WAL mode is enabled (or delete mode for in-memory databases in tests)
	var journalMode string
	if err := db.QueryRow("PRAGMA journal_mode").Scan(&journalMode); err != nil {
		return nil, fmt.Errorf("failed to check journal mode: %w", err)
	}
	// WAL mode should be enabled for file-based databases
	// In-memory databases will use "delete" or "memory" mode, which is acceptable for testing
	if journalMode != "wal" && journalMode != "delete" && journalMode != "memory" {
		return nil, fmt.Errorf("unexpected journal mode: got %s", journalMode)
	}

	// Initialize schema
	if err := store.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return store, nil
}

// Close closes the database connection
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

// retryOnBusy retries a database operation if it fails due to SQLITE_BUSY
// This provides an additional safety net on top of the busy_timeout pragma
func (s *SQLiteStore) retryOnBusy(operation func() error, maxRetries int) error {
	var err error
	for i := 0; i < maxRetries; i++ {
		err = operation()
		if err == nil {
			return nil
		}

		// Check if error is SQLITE_BUSY (database is locked)
		if err.Error() == "database is locked (5) (SQLITE_BUSY)" {
			// Exponential backoff: 10ms, 20ms, 40ms, 80ms, 160ms
			backoff := time.Duration(10*(1<<uint(i))) * time.Millisecond
			time.Sleep(backoff)
			continue
		}

		// If it's not a busy error, return immediately
		return err
	}
	return fmt.Errorf("operation failed after %d retries: %w", maxRetries, err)
}

// initSchema creates the database schema if it doesn't exist
func (s *SQLiteStore) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS projects (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		description TEXT,
		status TEXT NOT NULL,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		data TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS pipelines (
		id TEXT PRIMARY KEY,
		project_id TEXT NOT NULL,
		name TEXT NOT NULL,
		type TEXT NOT NULL,
		description TEXT,
		status TEXT NOT NULL,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		data TEXT NOT NULL,
		FOREIGN KEY (project_id) REFERENCES projects(id)
	);

	CREATE INDEX IF NOT EXISTS idx_pipelines_project_id ON pipelines(project_id);

	CREATE TABLE IF NOT EXISTS schedules (
		id TEXT PRIMARY KEY,
		project_id TEXT NOT NULL,
		name TEXT NOT NULL,
		cron_schedule TEXT NOT NULL,
		enabled INTEGER NOT NULL,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		last_run DATETIME,
		next_run DATETIME,
		data TEXT NOT NULL,
		FOREIGN KEY (project_id) REFERENCES projects(id)
	);

	CREATE INDEX IF NOT EXISTS idx_schedules_project_id ON schedules(project_id);

	CREATE TABLE IF NOT EXISTS plugins (
		id TEXT PRIMARY KEY,
		name TEXT UNIQUE NOT NULL,
		version TEXT NOT NULL,
		description TEXT,
		author TEXT,
		repository_url TEXT NOT NULL,
		git_commit_hash TEXT,
		plugin_definition TEXT NOT NULL,
		binary_data BLOB NOT NULL,
		status TEXT NOT NULL,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		last_loaded_at DATETIME
	);

	CREATE INDEX IF NOT EXISTS idx_plugins_name ON plugins(name);
	CREATE INDEX IF NOT EXISTS idx_plugins_status ON plugins(status);

	CREATE TABLE IF NOT EXISTS plugin_actions (
		id TEXT PRIMARY KEY,
		plugin_id TEXT NOT NULL,
		name TEXT NOT NULL,
		description TEXT,
		parameters TEXT,
		returns TEXT,
		FOREIGN KEY (plugin_id) REFERENCES plugins(id) ON DELETE CASCADE,
		UNIQUE(plugin_id, name)
	);

	CREATE INDEX IF NOT EXISTS idx_plugin_actions_plugin_id ON plugin_actions(plugin_id);

	CREATE TABLE IF NOT EXISTS storage_configs (
		id TEXT PRIMARY KEY,
		project_id TEXT NOT NULL,
		plugin_type TEXT NOT NULL,
		ontology_id TEXT,
		active INTEGER NOT NULL DEFAULT 1,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL,
		config TEXT NOT NULL,
		FOREIGN KEY (project_id) REFERENCES projects(id)
	);

	CREATE INDEX IF NOT EXISTS idx_storage_configs_project_id ON storage_configs(project_id);

	CREATE TABLE IF NOT EXISTS ontologies (
		id TEXT PRIMARY KEY,
		project_id TEXT NOT NULL,
		name TEXT NOT NULL,
		description TEXT,
		version TEXT NOT NULL,
		content TEXT NOT NULL,
		status TEXT NOT NULL,
		is_generated INTEGER NOT NULL DEFAULT 0,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL,
		FOREIGN KEY (project_id) REFERENCES projects(id)
	);

	CREATE INDEX IF NOT EXISTS idx_ontologies_project_id ON ontologies(project_id);

	CREATE TABLE IF NOT EXISTS ml_models (
		id TEXT PRIMARY KEY,
		project_id TEXT NOT NULL,
		ontology_id TEXT NOT NULL,
		name TEXT NOT NULL,
		description TEXT,
		type TEXT NOT NULL,
		status TEXT NOT NULL,
		version TEXT NOT NULL,
		is_recommended INTEGER NOT NULL DEFAULT 0,
		recommendation_score INTEGER NOT NULL DEFAULT 0,
		model_artifact_path TEXT,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL,
		trained_at TEXT,
		data TEXT NOT NULL,
		FOREIGN KEY (project_id) REFERENCES projects(id),
		FOREIGN KEY (ontology_id) REFERENCES ontologies(id)
	);

	CREATE INDEX IF NOT EXISTS idx_ml_models_project_id ON ml_models(project_id);
	CREATE INDEX IF NOT EXISTS idx_ml_models_ontology_id ON ml_models(ontology_id);

	CREATE TABLE IF NOT EXISTS digital_twins (
		id TEXT PRIMARY KEY,
		project_id TEXT NOT NULL,
		name TEXT NOT NULL,
		description TEXT,
		ontology_id TEXT NOT NULL,
		status TEXT NOT NULL,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL,
		last_synced_at TEXT,
		data TEXT NOT NULL,
		FOREIGN KEY (project_id) REFERENCES projects(id),
		FOREIGN KEY (ontology_id) REFERENCES ontologies(id)
	);

	CREATE INDEX IF NOT EXISTS idx_digital_twins_project_id ON digital_twins(project_id);
	CREATE INDEX IF NOT EXISTS idx_digital_twins_ontology_id ON digital_twins(ontology_id);

	CREATE TABLE IF NOT EXISTS dt_entities (
		id TEXT PRIMARY KEY,
		twin_id TEXT NOT NULL,
		entity_type TEXT NOT NULL,
		source_data_id TEXT,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL,
		data TEXT NOT NULL,
		FOREIGN KEY (twin_id) REFERENCES digital_twins(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_dt_entities_twin_id ON dt_entities(twin_id);
	CREATE INDEX IF NOT EXISTS idx_dt_entities_type ON dt_entities(entity_type);

	CREATE TABLE IF NOT EXISTS dt_scenarios (
		id TEXT PRIMARY KEY,
		twin_id TEXT NOT NULL,
		name TEXT NOT NULL,
		description TEXT,
		base_state TEXT NOT NULL,
		status TEXT NOT NULL,
		created_at TEXT NOT NULL,
		data TEXT NOT NULL,
		FOREIGN KEY (twin_id) REFERENCES digital_twins(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_dt_scenarios_twin_id ON dt_scenarios(twin_id);

	CREATE TABLE IF NOT EXISTS dt_actions (
		id TEXT PRIMARY KEY,
		twin_id TEXT NOT NULL,
		name TEXT NOT NULL,
		enabled INTEGER NOT NULL DEFAULT 1,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL,
		data TEXT NOT NULL,
		FOREIGN KEY (twin_id) REFERENCES digital_twins(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_dt_actions_twin_id ON dt_actions(twin_id);

	CREATE TABLE IF NOT EXISTS dt_predictions (
		id TEXT PRIMARY KEY,
		twin_id TEXT NOT NULL,
		entity_id TEXT NOT NULL,
		model_id TEXT NOT NULL,
		cached_until TEXT,
		created_at TEXT NOT NULL,
		data TEXT NOT NULL,
		FOREIGN KEY (twin_id) REFERENCES digital_twins(id) ON DELETE CASCADE,
		FOREIGN KEY (entity_id) REFERENCES dt_entities(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_dt_predictions_twin_id ON dt_predictions(twin_id);
	CREATE INDEX IF NOT EXISTS idx_dt_predictions_entity_id ON dt_predictions(entity_id);
	CREATE INDEX IF NOT EXISTS idx_dt_predictions_cached_until ON dt_predictions(cached_until);
	`

	_, err := s.db.Exec(schema)
	return err
}

// SaveProject saves a project to the database
func (s *SQLiteStore) SaveProject(project *models.Project) error {
	data, err := json.Marshal(project)
	if err != nil {
		return fmt.Errorf("failed to marshal project: %w", err)
	}

	query := `
		INSERT OR REPLACE INTO projects (id, name, description, status, created_at, updated_at, data)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	_, err = s.db.Exec(query,
		project.ID,
		project.Name,
		project.Description,
		project.Status,
		project.Metadata.CreatedAt,
		project.Metadata.UpdatedAt,
		string(data),
	)

	if err != nil {
		return fmt.Errorf("failed to save project: %w", err)
	}

	return nil
}

// GetProject retrieves a project by ID
func (s *SQLiteStore) GetProject(id string) (*models.Project, error) {
	var data string
	query := `SELECT data FROM projects WHERE id = ?`

	err := s.db.QueryRow(query, id).Scan(&data)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("project not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	var project models.Project
	if err := json.Unmarshal([]byte(data), &project); err != nil {
		return nil, fmt.Errorf("failed to unmarshal project: %w", err)
	}

	return &project, nil
}

// ListProjects lists all projects
func (s *SQLiteStore) ListProjects() ([]*models.Project, error) {
	query := `SELECT data FROM projects ORDER BY created_at DESC`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}
	defer rows.Close()

	projects := make([]*models.Project, 0)
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			continue
		}

		var project models.Project
		if err := json.Unmarshal([]byte(data), &project); err != nil {
			continue
		}

		projects = append(projects, &project)
	}

	return projects, nil
}

// DeleteProject deletes a project
func (s *SQLiteStore) DeleteProject(id string) error {
	query := `DELETE FROM projects WHERE id = ?`
	_, err := s.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete project: %w", err)
	}
	return nil
}

// SavePipeline saves a pipeline to the database
func (s *SQLiteStore) SavePipeline(pipeline *models.Pipeline) error {
	data, err := json.Marshal(pipeline)
	if err != nil {
		return fmt.Errorf("failed to marshal pipeline: %w", err)
	}

	query := `
		INSERT OR REPLACE INTO pipelines (id, project_id, name, type, description, status, created_at, updated_at, data)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = s.db.Exec(query,
		pipeline.ID,
		pipeline.ProjectID,
		pipeline.Name,
		pipeline.Type,
		pipeline.Description,
		pipeline.Status,
		pipeline.CreatedAt,
		pipeline.UpdatedAt,
		string(data),
	)

	if err != nil {
		return fmt.Errorf("failed to save pipeline: %w", err)
	}

	return nil
}

// GetPipeline retrieves a pipeline by ID
func (s *SQLiteStore) GetPipeline(id string) (*models.Pipeline, error) {
	var data string
	query := `SELECT data FROM pipelines WHERE id = ?`

	err := s.db.QueryRow(query, id).Scan(&data)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("pipeline not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get pipeline: %w", err)
	}

	var pipeline models.Pipeline
	if err := json.Unmarshal([]byte(data), &pipeline); err != nil {
		return nil, fmt.Errorf("failed to unmarshal pipeline: %w", err)
	}

	return &pipeline, nil
}

// ListPipelines lists all pipelines
func (s *SQLiteStore) ListPipelines() ([]*models.Pipeline, error) {
	query := `SELECT data FROM pipelines ORDER BY created_at DESC`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list pipelines: %w", err)
	}
	defer rows.Close()

	pipelines := make([]*models.Pipeline, 0)
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			continue
		}

		var pipeline models.Pipeline
		if err := json.Unmarshal([]byte(data), &pipeline); err != nil {
			continue
		}

		pipelines = append(pipelines, &pipeline)
	}

	return pipelines, nil
}

// ListPipelinesByProject lists all pipelines for a specific project
func (s *SQLiteStore) ListPipelinesByProject(projectID string) ([]*models.Pipeline, error) {
	query := `SELECT data FROM pipelines WHERE project_id = ? ORDER BY created_at DESC`

	rows, err := s.db.Query(query, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list pipelines: %w", err)
	}
	defer rows.Close()

	pipelines := make([]*models.Pipeline, 0)
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			continue
		}

		var pipeline models.Pipeline
		if err := json.Unmarshal([]byte(data), &pipeline); err != nil {
			continue
		}

		pipelines = append(pipelines, &pipeline)
	}

	return pipelines, nil
}

// DeletePipeline deletes a pipeline
func (s *SQLiteStore) DeletePipeline(id string) error {
	query := `DELETE FROM pipelines WHERE id = ?`
	_, err := s.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete pipeline: %w", err)
	}
	return nil
}

// SaveSchedule saves a schedule to the database
func (s *SQLiteStore) SaveSchedule(schedule *models.Schedule) error {
	data, err := json.Marshal(schedule)
	if err != nil {
		return fmt.Errorf("failed to marshal schedule: %w", err)
	}

	var lastRun, nextRun *time.Time
	if schedule.LastRun != nil {
		lastRun = schedule.LastRun
	}
	if schedule.NextRun != nil {
		nextRun = schedule.NextRun
	}

	enabled := 0
	if schedule.Enabled {
		enabled = 1
	}

	query := `
		INSERT OR REPLACE INTO schedules (id, project_id, name, cron_schedule, enabled, created_at, updated_at, last_run, next_run, data)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	// Use retry logic for schedule saves since they can happen concurrently during scheduled execution
	err = s.retryOnBusy(func() error {
		_, execErr := s.db.Exec(query,
			schedule.ID,
			schedule.ProjectID,
			schedule.Name,
			schedule.CronSchedule,
			enabled,
			schedule.CreatedAt,
			schedule.UpdatedAt,
			lastRun,
			nextRun,
			string(data),
		)
		return execErr
	}, 5)

	if err != nil {
		return fmt.Errorf("failed to save schedule: %w", err)
	}

	return nil
}

// GetSchedule retrieves a schedule by ID
func (s *SQLiteStore) GetSchedule(id string) (*models.Schedule, error) {
	var data string
	query := `SELECT data FROM schedules WHERE id = ?`

	err := s.db.QueryRow(query, id).Scan(&data)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("schedule not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get schedule: %w", err)
	}

	var schedule models.Schedule
	if err := json.Unmarshal([]byte(data), &schedule); err != nil {
		return nil, fmt.Errorf("failed to unmarshal schedule: %w", err)
	}

	return &schedule, nil
}

// ListSchedules lists all schedules
func (s *SQLiteStore) ListSchedules() ([]*models.Schedule, error) {
	query := `SELECT data FROM schedules ORDER BY created_at DESC`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list schedules: %w", err)
	}
	defer rows.Close()

	schedules := make([]*models.Schedule, 0)
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			continue
		}

		var schedule models.Schedule
		if err := json.Unmarshal([]byte(data), &schedule); err != nil {
			continue
		}

		schedules = append(schedules, &schedule)
	}

	return schedules, nil
}

// ListSchedulesByProject lists all schedules for a specific project
func (s *SQLiteStore) ListSchedulesByProject(projectID string) ([]*models.Schedule, error) {
	query := `SELECT data FROM schedules WHERE project_id = ? ORDER BY created_at DESC`

	rows, err := s.db.Query(query, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list schedules: %w", err)
	}
	defer rows.Close()

	schedules := make([]*models.Schedule, 0)
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			continue
		}

		var schedule models.Schedule
		if err := json.Unmarshal([]byte(data), &schedule); err != nil {
			continue
		}

		schedules = append(schedules, &schedule)
	}

	return schedules, nil
}

// DeleteSchedule deletes a schedule
func (s *SQLiteStore) DeleteSchedule(id string) error {
	query := `DELETE FROM schedules WHERE id = ?`
	_, err := s.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete schedule: %w", err)
	}
	return nil
}

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

// SaveDigitalTwin saves a digital twin to the database
func (s *SQLiteStore) SaveDigitalTwin(twin *models.DigitalTwin) error {
	data, err := json.Marshal(twin)
	if err != nil {
		return fmt.Errorf("failed to marshal digital twin: %w", err)
	}

	lastSyncAt := ""
	if twin.LastSyncAt != nil {
		lastSyncAt = twin.LastSyncAt.Format(time.RFC3339)
	}

	query := `
		INSERT OR REPLACE INTO digital_twins (
			id, project_id, name, description, ontology_id, status,
			created_at, updated_at, last_synced_at, data
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = s.db.Exec(query,
		twin.ID,
		twin.ProjectID,
		twin.Name,
		twin.Description,
		twin.OntologyID,
		twin.Status,
		twin.CreatedAt.Format(time.RFC3339),
		twin.UpdatedAt.Format(time.RFC3339),
		lastSyncAt,
		data,
	)

	if err != nil {
		return fmt.Errorf("failed to save digital twin: %w", err)
	}

	return nil
}

// GetDigitalTwin retrieves a digital twin by ID
func (s *SQLiteStore) GetDigitalTwin(id string) (*models.DigitalTwin, error) {
	query := `SELECT data FROM digital_twins WHERE id = ?`

	var data []byte
	err := s.db.QueryRow(query, id).Scan(&data)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("digital twin not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get digital twin: %w", err)
	}

	var twin models.DigitalTwin
	if err := json.Unmarshal(data, &twin); err != nil {
		return nil, fmt.Errorf("failed to unmarshal digital twin: %w", err)
	}

	return &twin, nil
}

// ListDigitalTwins lists all digital twins
func (s *SQLiteStore) ListDigitalTwins() ([]*models.DigitalTwin, error) {
	query := `SELECT data FROM digital_twins ORDER BY created_at DESC`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list digital twins: %w", err)
	}
	defer rows.Close()

	var twins []*models.DigitalTwin
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, fmt.Errorf("failed to scan digital twin: %w", err)
		}

		var twin models.DigitalTwin
		if err := json.Unmarshal(data, &twin); err != nil {
			return nil, fmt.Errorf("failed to unmarshal digital twin: %w", err)
		}

		twins = append(twins, &twin)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating digital twins: %w", err)
	}

	return twins, nil
}

// ListDigitalTwinsByProject lists all digital twins for a specific project
func (s *SQLiteStore) ListDigitalTwinsByProject(projectID string) ([]*models.DigitalTwin, error) {
	query := `SELECT data FROM digital_twins WHERE project_id = ? ORDER BY created_at DESC`

	rows, err := s.db.Query(query, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list digital twins: %w", err)
	}
	defer rows.Close()

	var twins []*models.DigitalTwin
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, fmt.Errorf("failed to scan digital twin: %w", err)
		}

		var twin models.DigitalTwin
		if err := json.Unmarshal(data, &twin); err != nil {
			return nil, fmt.Errorf("failed to unmarshal digital twin: %w", err)
		}

		twins = append(twins, &twin)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating digital twins: %w", err)
	}

	return twins, nil
}

// DeleteDigitalTwin deletes a digital twin
func (s *SQLiteStore) DeleteDigitalTwin(id string) error {
	query := `DELETE FROM digital_twins WHERE id = ?`
	_, err := s.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete digital twin: %w", err)
	}
	return nil
}

// SaveEntity saves an entity to the database
func (s *SQLiteStore) SaveEntity(entity *models.Entity) error {
	data, err := json.Marshal(entity)
	if err != nil {
		return fmt.Errorf("failed to marshal entity: %w", err)
	}

	query := `
		INSERT OR REPLACE INTO dt_entities (
			id, twin_id, entity_type, source_data_id,
			created_at, updated_at, data
		) VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	_, err = s.db.Exec(query,
		entity.ID,
		entity.DigitalTwinID,
		entity.Type,
		entity.SourceDataID,
		entity.CreatedAt.Format(time.RFC3339),
		entity.UpdatedAt.Format(time.RFC3339),
		data,
	)

	if err != nil {
		return fmt.Errorf("failed to save entity: %w", err)
	}

	return nil
}

// GetEntity retrieves an entity by ID
func (s *SQLiteStore) GetEntity(id string) (*models.Entity, error) {
	query := `SELECT data FROM dt_entities WHERE id = ?`

	var data []byte
	err := s.db.QueryRow(query, id).Scan(&data)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("entity not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get entity: %w", err)
	}

	var entity models.Entity
	if err := json.Unmarshal(data, &entity); err != nil {
		return nil, fmt.Errorf("failed to unmarshal entity: %w", err)
	}

	return &entity, nil
}

// ListEntitiesByDigitalTwin lists all entities for a specific digital twin
func (s *SQLiteStore) ListEntitiesByDigitalTwin(twinID string) ([]*models.Entity, error) {
	query := `SELECT data FROM dt_entities WHERE twin_id = ? ORDER BY created_at DESC`

	rows, err := s.db.Query(query, twinID)
	if err != nil {
		return nil, fmt.Errorf("failed to list entities: %w", err)
	}
	defer rows.Close()

	var entities []*models.Entity
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, fmt.Errorf("failed to scan entity: %w", err)
		}

		var entity models.Entity
		if err := json.Unmarshal(data, &entity); err != nil {
			return nil, fmt.Errorf("failed to unmarshal entity: %w", err)
		}

		entities = append(entities, &entity)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating entities: %w", err)
	}

	return entities, nil
}

// DeleteEntity deletes an entity
func (s *SQLiteStore) DeleteEntity(id string) error {
	query := `DELETE FROM dt_entities WHERE id = ?`
	_, err := s.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete entity: %w", err)
	}
	return nil
}

// SaveScenario saves a scenario to the database
func (s *SQLiteStore) SaveScenario(scenario *models.Scenario) error {
	data, err := json.Marshal(scenario)
	if err != nil {
		return fmt.Errorf("failed to marshal scenario: %w", err)
	}

	query := `
		INSERT OR REPLACE INTO dt_scenarios (
			id, twin_id, name, description, base_state, status, created_at, data
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = s.db.Exec(query,
		scenario.ID,
		scenario.DigitalTwinID,
		scenario.Name,
		scenario.Description,
		scenario.BaseState,
		scenario.Status,
		scenario.CreatedAt.Format(time.RFC3339),
		data,
	)

	if err != nil {
		return fmt.Errorf("failed to save scenario: %w", err)
	}

	return nil
}

// GetScenario retrieves a scenario by ID
func (s *SQLiteStore) GetScenario(id string) (*models.Scenario, error) {
	query := `SELECT data FROM dt_scenarios WHERE id = ?`

	var data []byte
	err := s.db.QueryRow(query, id).Scan(&data)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("scenario not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get scenario: %w", err)
	}

	var scenario models.Scenario
	if err := json.Unmarshal(data, &scenario); err != nil {
		return nil, fmt.Errorf("failed to unmarshal scenario: %w", err)
	}

	return &scenario, nil
}

// ListScenariosByDigitalTwin lists all scenarios for a specific digital twin
func (s *SQLiteStore) ListScenariosByDigitalTwin(twinID string) ([]*models.Scenario, error) {
	query := `SELECT data FROM dt_scenarios WHERE twin_id = ? ORDER BY created_at DESC`

	rows, err := s.db.Query(query, twinID)
	if err != nil {
		return nil, fmt.Errorf("failed to list scenarios: %w", err)
	}
	defer rows.Close()

	var scenarios []*models.Scenario
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, fmt.Errorf("failed to scan scenario: %w", err)
		}

		var scenario models.Scenario
		if err := json.Unmarshal(data, &scenario); err != nil {
			return nil, fmt.Errorf("failed to unmarshal scenario: %w", err)
		}

		scenarios = append(scenarios, &scenario)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating scenarios: %w", err)
	}

	return scenarios, nil
}

// DeleteScenario deletes a scenario
func (s *SQLiteStore) DeleteScenario(id string) error {
	query := `DELETE FROM dt_scenarios WHERE id = ?`
	_, err := s.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete scenario: %w", err)
	}
	return nil
}

// SaveAction saves an action to the database
func (s *SQLiteStore) SaveAction(action *models.Action) error {
	data, err := json.Marshal(action)
	if err != nil {
		return fmt.Errorf("failed to marshal action: %w", err)
	}

	enabled := 0
	if action.Enabled {
		enabled = 1
	}

	query := `
		INSERT OR REPLACE INTO dt_actions (
			id, twin_id, name, enabled, created_at, updated_at, data
		) VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	_, err = s.db.Exec(query,
		action.ID,
		action.DigitalTwinID,
		action.Name,
		enabled,
		action.CreatedAt.Format(time.RFC3339),
		action.UpdatedAt.Format(time.RFC3339),
		data,
	)

	if err != nil {
		return fmt.Errorf("failed to save action: %w", err)
	}

	return nil
}

// GetAction retrieves an action by ID
func (s *SQLiteStore) GetAction(id string) (*models.Action, error) {
	query := `SELECT data FROM dt_actions WHERE id = ?`

	var data []byte
	err := s.db.QueryRow(query, id).Scan(&data)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("action not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get action: %w", err)
	}

	var action models.Action
	if err := json.Unmarshal(data, &action); err != nil {
		return nil, fmt.Errorf("failed to unmarshal action: %w", err)
	}

	return &action, nil
}

// ListActionsByDigitalTwin lists all actions for a specific digital twin
func (s *SQLiteStore) ListActionsByDigitalTwin(twinID string) ([]*models.Action, error) {
	query := `SELECT data FROM dt_actions WHERE twin_id = ? ORDER BY created_at DESC`

	rows, err := s.db.Query(query, twinID)
	if err != nil {
		return nil, fmt.Errorf("failed to list actions: %w", err)
	}
	defer rows.Close()

	var actions []*models.Action
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, fmt.Errorf("failed to scan action: %w", err)
		}

		var action models.Action
		if err := json.Unmarshal(data, &action); err != nil {
			return nil, fmt.Errorf("failed to unmarshal action: %w", err)
		}

		actions = append(actions, &action)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating actions: %w", err)
	}

	return actions, nil
}

// DeleteAction deletes an action
func (s *SQLiteStore) DeleteAction(id string) error {
	query := `DELETE FROM dt_actions WHERE id = ?`
	_, err := s.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete action: %w", err)
	}
	return nil
}

// SavePrediction saves a prediction to the database
func (s *SQLiteStore) SavePrediction(prediction *models.Prediction) error {
	data, err := json.Marshal(prediction)
	if err != nil {
		return fmt.Errorf("failed to marshal prediction: %w", err)
	}

	query := `
		INSERT OR REPLACE INTO dt_predictions (
			id, twin_id, entity_id, model_id, cached_until, created_at, data
		) VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	_, err = s.db.Exec(query,
		prediction.ID,
		prediction.DigitalTwinID,
		prediction.EntityID,
		prediction.ModelID,
		prediction.ExpiresAt.Format(time.RFC3339),
		prediction.CachedAt.Format(time.RFC3339),
		data,
	)

	if err != nil {
		return fmt.Errorf("failed to save prediction: %w", err)
	}

	return nil
}

// GetPrediction retrieves a prediction by ID
func (s *SQLiteStore) GetPrediction(id string) (*models.Prediction, error) {
	query := `SELECT data FROM dt_predictions WHERE id = ?`

	var data []byte
	err := s.db.QueryRow(query, id).Scan(&data)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("prediction not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get prediction: %w", err)
	}

	var prediction models.Prediction
	if err := json.Unmarshal(data, &prediction); err != nil {
		return nil, fmt.Errorf("failed to unmarshal prediction: %w", err)
	}

	return &prediction, nil
}

// ListPredictionsByEntity lists all predictions for a specific entity
func (s *SQLiteStore) ListPredictionsByEntity(entityID string) ([]*models.Prediction, error) {
	query := `SELECT data FROM dt_predictions WHERE entity_id = ? ORDER BY created_at DESC`

	rows, err := s.db.Query(query, entityID)
	if err != nil {
		return nil, fmt.Errorf("failed to list predictions: %w", err)
	}
	defer rows.Close()

	var predictions []*models.Prediction
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, fmt.Errorf("failed to scan prediction: %w", err)
		}

		var prediction models.Prediction
		if err := json.Unmarshal(data, &prediction); err != nil {
			return nil, fmt.Errorf("failed to unmarshal prediction: %w", err)
		}

		predictions = append(predictions, &prediction)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating predictions: %w", err)
	}

	return predictions, nil
}

// ListPredictionsByDigitalTwin lists all predictions for a specific digital twin
func (s *SQLiteStore) ListPredictionsByDigitalTwin(twinID string) ([]*models.Prediction, error) {
	query := `SELECT data FROM dt_predictions WHERE twin_id = ? ORDER BY created_at DESC`

	rows, err := s.db.Query(query, twinID)
	if err != nil {
		return nil, fmt.Errorf("failed to list predictions: %w", err)
	}
	defer rows.Close()

	var predictions []*models.Prediction
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, fmt.Errorf("failed to scan prediction: %w", err)
		}

		var prediction models.Prediction
		if err := json.Unmarshal(data, &prediction); err != nil {
			return nil, fmt.Errorf("failed to unmarshal prediction: %w", err)
		}

		predictions = append(predictions, &prediction)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating predictions: %w", err)
	}

	return predictions, nil
}

// DeletePrediction deletes a prediction
func (s *SQLiteStore) DeletePrediction(id string) error {
	query := `DELETE FROM dt_predictions WHERE id = ?`
	_, err := s.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete prediction: %w", err)
	}
	return nil
}

// DeleteExpiredPredictions deletes all expired predictions for a digital twin
func (s *SQLiteStore) DeleteExpiredPredictions(twinID string) error {
	query := `DELETE FROM dt_predictions WHERE twin_id = ? AND cached_until < ?`
	_, err := s.db.Exec(query, twinID, time.Now().Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("failed to delete expired predictions: %w", err)
	}
	return nil
}
