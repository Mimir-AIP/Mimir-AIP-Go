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
