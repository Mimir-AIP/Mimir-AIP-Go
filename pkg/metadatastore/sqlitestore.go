package metadatastore

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"

	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

// SQLiteStore provides SQLite-based persistence for projects, pipelines, and schedules
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore creates a new SQLite-based storage instance
func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	// Open database with connection pooling parameters.
	// Format: file:path?param=value
	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(10000)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=foreign_keys(ON)", dbPath)
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

	// Verify WAL mode is enabled (or delete mode for in-memory databases in tests).
	var journalMode string
	if err := db.QueryRow("PRAGMA journal_mode").Scan(&journalMode); err != nil {
		return nil, fmt.Errorf("failed to check journal mode: %w", err)
	}
	// WAL mode should be enabled for file-based databases.
	// In-memory databases will use "delete" or "memory" mode, which is acceptable for testing.
	if journalMode != "wal" && journalMode != "delete" && journalMode != "memory" {
		return nil, fmt.Errorf("unexpected journal mode: got %s", journalMode)
	}

	var foreignKeysEnabled int
	if err := db.QueryRow("PRAGMA foreign_keys").Scan(&foreignKeysEnabled); err != nil {
		return nil, fmt.Errorf("failed to check foreign key enforcement: %w", err)
	}
	if foreignKeysEnabled != 1 {
		return nil, fmt.Errorf("sqlite foreign key enforcement is disabled")
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

	CREATE TABLE IF NOT EXISTS pipeline_checkpoints (
		project_id TEXT NOT NULL,
		pipeline_id TEXT NOT NULL,
		step_name TEXT NOT NULL,
		scope TEXT NOT NULL DEFAULT '',
		version INTEGER NOT NULL,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		data TEXT NOT NULL,
		PRIMARY KEY (project_id, pipeline_id, step_name, scope),
		FOREIGN KEY (project_id) REFERENCES projects(id),
		FOREIGN KEY (pipeline_id) REFERENCES pipelines(id)
	);

	CREATE INDEX IF NOT EXISTS idx_pipeline_checkpoints_pipeline_id ON pipeline_checkpoints(pipeline_id);


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

	CREATE TABLE IF NOT EXISTS work_tasks (
		id TEXT PRIMARY KEY,
		project_id TEXT NOT NULL,
		type TEXT NOT NULL,
		status TEXT NOT NULL,
		submitted_at DATETIME NOT NULL,
		data TEXT NOT NULL,
		FOREIGN KEY (project_id) REFERENCES projects(id)
	);

	CREATE INDEX IF NOT EXISTS idx_work_tasks_project_id ON work_tasks(project_id);
	CREATE INDEX IF NOT EXISTS idx_work_tasks_status ON work_tasks(status);
	CREATE INDEX IF NOT EXISTS idx_work_tasks_submitted_at ON work_tasks(submitted_at);


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

CREATE TABLE IF NOT EXISTS automations (
	id TEXT PRIMARY KEY,
	project_id TEXT NOT NULL,
	target_type TEXT NOT NULL,
	target_id TEXT NOT NULL,
	enabled INTEGER NOT NULL DEFAULT 1,
	trigger_type TEXT NOT NULL,
	action_type TEXT NOT NULL,
	created_at DATETIME NOT NULL,
	updated_at DATETIME NOT NULL,
	data TEXT NOT NULL,
	FOREIGN KEY (project_id) REFERENCES projects(id)
	);
CREATE INDEX IF NOT EXISTS idx_automations_project_id ON automations(project_id);
CREATE INDEX IF NOT EXISTS idx_automations_target_enabled ON automations(project_id, target_type, enabled);

CREATE TABLE IF NOT EXISTS twin_processing_runs (
	id TEXT PRIMARY KEY,
	project_id TEXT NOT NULL,
	twin_id TEXT NOT NULL,
	status TEXT NOT NULL,
	trigger_type TEXT NOT NULL,
	automation_id TEXT,
	requested_at DATETIME NOT NULL,
	started_at DATETIME,
	completed_at DATETIME,
	data TEXT NOT NULL,
	FOREIGN KEY (project_id) REFERENCES projects(id),
	FOREIGN KEY (twin_id) REFERENCES digital_twins(id) ON DELETE CASCADE,
	FOREIGN KEY (automation_id) REFERENCES automations(id)
	);
CREATE INDEX IF NOT EXISTS idx_twin_processing_runs_twin_requested_at ON twin_processing_runs(twin_id, requested_at DESC);
CREATE INDEX IF NOT EXISTS idx_twin_processing_runs_twin_status ON twin_processing_runs(twin_id, status);

CREATE TABLE IF NOT EXISTS alert_events (
	id TEXT PRIMARY KEY,
	project_id TEXT NOT NULL,
	twin_id TEXT NOT NULL,
	processing_run_id TEXT NOT NULL,
	severity TEXT NOT NULL,
	category TEXT NOT NULL,
	created_at DATETIME NOT NULL,
	triggered_export_pipeline_id TEXT,
	triggered_work_task_id TEXT,
	data TEXT NOT NULL,
	FOREIGN KEY (project_id) REFERENCES projects(id),
	FOREIGN KEY (twin_id) REFERENCES digital_twins(id) ON DELETE CASCADE,
	FOREIGN KEY (processing_run_id) REFERENCES twin_processing_runs(id) ON DELETE CASCADE
	);
CREATE INDEX IF NOT EXISTS idx_alert_events_twin_created_at ON alert_events(twin_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_alert_events_project_severity ON alert_events(project_id, severity, created_at DESC);

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

CREATE TABLE IF NOT EXISTS dt_entity_revisions (
	id TEXT PRIMARY KEY,
	entity_id TEXT NOT NULL,
	twin_id TEXT NOT NULL,
	revision INTEGER NOT NULL,
	recorded_at TEXT NOT NULL,
	data TEXT NOT NULL,
	FOREIGN KEY (entity_id) REFERENCES dt_entities(id) ON DELETE CASCADE,
	FOREIGN KEY (twin_id) REFERENCES digital_twins(id) ON DELETE CASCADE,
	UNIQUE(entity_id, revision)
	);

CREATE INDEX IF NOT EXISTS idx_dt_entity_revisions_entity_id ON dt_entity_revisions(entity_id);
	CREATE INDEX IF NOT EXISTS idx_dt_entity_revisions_twin_id ON dt_entity_revisions(twin_id);
	CREATE INDEX IF NOT EXISTS idx_dt_entity_revisions_recorded_at ON dt_entity_revisions(recorded_at);


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

	CREATE TABLE IF NOT EXISTS external_storage_plugins (
		name TEXT PRIMARY KEY,
		version TEXT NOT NULL DEFAULT '',
		description TEXT NOT NULL DEFAULT '',
		author TEXT NOT NULL DEFAULT '',
		repository_url TEXT NOT NULL,
		git_commit_hash TEXT NOT NULL DEFAULT '',
		status TEXT NOT NULL DEFAULT 'active',
		error_message TEXT NOT NULL DEFAULT '',
		installed_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS external_llm_providers (
		name            TEXT PRIMARY KEY,
		repository_url  TEXT NOT NULL,
		git_commit_hash TEXT NOT NULL DEFAULT '',
		status          TEXT NOT NULL DEFAULT 'active',
		error_message   TEXT NOT NULL DEFAULT '',
		installed_at    DATETIME NOT NULL,
		updated_at      DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS analysis_runs (
		id TEXT PRIMARY KEY,
		project_id TEXT NOT NULL,
		kind TEXT NOT NULL,
		status TEXT NOT NULL,
		created_at DATETIME NOT NULL,
		completed_at DATETIME,
		data TEXT NOT NULL,
		FOREIGN KEY (project_id) REFERENCES projects(id)
	);
	CREATE INDEX IF NOT EXISTS idx_analysis_runs_project_id ON analysis_runs(project_id);
	CREATE INDEX IF NOT EXISTS idx_analysis_runs_kind ON analysis_runs(kind);

	CREATE TABLE IF NOT EXISTS review_items (
		id TEXT PRIMARY KEY,
		project_id TEXT NOT NULL,
		run_id TEXT NOT NULL,
		finding_type TEXT NOT NULL,
		status TEXT NOT NULL,
		confidence REAL NOT NULL,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		data TEXT NOT NULL,
		FOREIGN KEY (project_id) REFERENCES projects(id),
		FOREIGN KEY (run_id) REFERENCES analysis_runs(id)
	);
	CREATE INDEX IF NOT EXISTS idx_review_items_project_id ON review_items(project_id);
	CREATE INDEX IF NOT EXISTS idx_review_items_status ON review_items(status);

	CREATE TABLE IF NOT EXISTS insights (
		id TEXT PRIMARY KEY,
		project_id TEXT NOT NULL,
		run_id TEXT NOT NULL,
		type TEXT NOT NULL,
		severity TEXT NOT NULL,
		confidence REAL NOT NULL,
		status TEXT NOT NULL,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		data TEXT NOT NULL,
		FOREIGN KEY (project_id) REFERENCES projects(id),
		FOREIGN KEY (run_id) REFERENCES analysis_runs(id)
	);
	CREATE INDEX IF NOT EXISTS idx_insights_project_id ON insights(project_id);
	CREATE INDEX IF NOT EXISTS idx_insights_severity ON insights(severity);
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

// GetPipelineCheckpoint retrieves persisted connector state for one pipeline step.
func (s *SQLiteStore) GetPipelineCheckpoint(projectID, pipelineID, stepName, scope string) (*models.PipelineCheckpoint, error) {
	var (
		version   int
		createdAt time.Time
		updatedAt time.Time
		data      string
	)

	query := `
		SELECT version, created_at, updated_at, data
		FROM pipeline_checkpoints
		WHERE project_id = ? AND pipeline_id = ? AND step_name = ? AND scope = ?
	`

	err := s.db.QueryRow(query, projectID, pipelineID, stepName, scope).Scan(&version, &createdAt, &updatedAt, &data)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get pipeline checkpoint: %w", err)
	}

	checkpointData := make(map[string]interface{})
	if err := json.Unmarshal([]byte(data), &checkpointData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal pipeline checkpoint: %w", err)
	}

	return &models.PipelineCheckpoint{
		ProjectID:  projectID,
		PipelineID: pipelineID,
		StepName:   stepName,
		Scope:      scope,
		Version:    version,
		Checkpoint: checkpointData,
		CreatedAt:  createdAt,
		UpdatedAt:  updatedAt,
	}, nil
}

// SavePipelineCheckpoint inserts or updates persisted connector state with optimistic versioning.
func (s *SQLiteStore) SavePipelineCheckpoint(checkpoint *models.PipelineCheckpoint) error {
	if checkpoint == nil {
		return fmt.Errorf("pipeline checkpoint is required")
	}
	if checkpoint.ProjectID == "" || checkpoint.PipelineID == "" || checkpoint.StepName == "" {
		return fmt.Errorf("pipeline checkpoint requires project_id, pipeline_id, and step_name")
	}
	if checkpoint.Checkpoint == nil {
		checkpoint.Checkpoint = map[string]interface{}{}
	}

	return s.retryOnBusy(func() error {
		tx, err := s.db.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()

		var (
			currentVersion int
			createdAt      time.Time
		)

		lookupErr := tx.QueryRow(`
			SELECT version, created_at
			FROM pipeline_checkpoints
			WHERE project_id = ? AND pipeline_id = ? AND step_name = ? AND scope = ?
		`, checkpoint.ProjectID, checkpoint.PipelineID, checkpoint.StepName, checkpoint.Scope).Scan(&currentVersion, &createdAt)

		now := time.Now().UTC()
		switch lookupErr {
		case nil:
			if checkpoint.Version != currentVersion {
				return fmt.Errorf("pipeline checkpoint version conflict: expected %d, got %d", currentVersion, checkpoint.Version)
			}
			checkpoint.Version = currentVersion + 1
			checkpoint.CreatedAt = createdAt
			checkpoint.UpdatedAt = now
		case sql.ErrNoRows:
			if checkpoint.Version != 0 {
				return fmt.Errorf("pipeline checkpoint version conflict: expected 0, got %d", checkpoint.Version)
			}
			checkpoint.Version = 1
			checkpoint.CreatedAt = now
			checkpoint.UpdatedAt = now
		default:
			return fmt.Errorf("failed to read existing pipeline checkpoint: %w", lookupErr)
		}

		data, err := json.Marshal(checkpoint.Checkpoint)
		if err != nil {
			return fmt.Errorf("failed to marshal pipeline checkpoint: %w", err)
		}

		if currentVersion == 0 && checkpoint.Version == 1 {
			_, err = tx.Exec(`
				INSERT INTO pipeline_checkpoints (project_id, pipeline_id, step_name, scope, version, created_at, updated_at, data)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			`, checkpoint.ProjectID, checkpoint.PipelineID, checkpoint.StepName, checkpoint.Scope, checkpoint.Version, checkpoint.CreatedAt, checkpoint.UpdatedAt, string(data))
		} else {
			_, err = tx.Exec(`
				UPDATE pipeline_checkpoints
				SET version = ?, updated_at = ?, data = ?
				WHERE project_id = ? AND pipeline_id = ? AND step_name = ? AND scope = ?
			`, checkpoint.Version, checkpoint.UpdatedAt, string(data), checkpoint.ProjectID, checkpoint.PipelineID, checkpoint.StepName, checkpoint.Scope)
		}
		if err != nil {
			return fmt.Errorf("failed to persist pipeline checkpoint: %w", err)
		}

		return tx.Commit()
	}, 5)
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

// SaveWorkTask saves a work task to the database.
func (s *SQLiteStore) SaveWorkTask(task *models.WorkTask) error {
	data, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("failed to marshal work task: %w", err)
	}
	query := `
		INSERT OR REPLACE INTO work_tasks (id, project_id, type, status, submitted_at, data)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	_, err = s.db.Exec(query, task.ID, task.ProjectID, task.Type, task.Status, task.SubmittedAt.UTC(), data)
	if err != nil {
		return fmt.Errorf("failed to save work task: %w", err)
	}
	return nil
}

// GetWorkTask retrieves a work task by ID.
func (s *SQLiteStore) GetWorkTask(id string) (*models.WorkTask, error) {
	var data string
	if err := s.db.QueryRow(`SELECT data FROM work_tasks WHERE id = ?`, id).Scan(&data); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("work task not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get work task: %w", err)
	}
	var task models.WorkTask
	if err := json.Unmarshal([]byte(data), &task); err != nil {
		return nil, fmt.Errorf("failed to unmarshal work task: %w", err)
	}
	return &task, nil
}

// ListWorkTasks lists all persisted work tasks.
func (s *SQLiteStore) ListWorkTasks() ([]*models.WorkTask, error) {
	rows, err := s.db.Query(`SELECT data FROM work_tasks ORDER BY submitted_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("failed to list work tasks: %w", err)
	}
	defer rows.Close()
	tasks := make([]*models.WorkTask, 0)
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			continue
		}
		var task models.WorkTask
		if err := json.Unmarshal([]byte(data), &task); err != nil {
			continue
		}
		tasks = append(tasks, &task)
	}
	return tasks, nil
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

// SaveAutomation saves one explicit automation policy.
func (s *SQLiteStore) SaveAutomation(automation *models.Automation) error {
	data, err := json.Marshal(automation)
	if err != nil {
		return fmt.Errorf("failed to marshal automation: %w", err)
	}
	enabled := 0
	if automation.Enabled {
		enabled = 1
	}
	query := `
		INSERT OR REPLACE INTO automations (
			id, project_id, target_type, target_id, enabled, trigger_type, action_type, created_at, updated_at, data
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err = s.db.Exec(query,
		automation.ID,
		automation.ProjectID,
		automation.TargetType,
		automation.TargetID,
		enabled,
		automation.TriggerType,
		automation.ActionType,
		automation.CreatedAt,
		automation.UpdatedAt,
		data,
	)
	if err != nil {
		return fmt.Errorf("failed to save automation: %w", err)
	}
	return nil
}

// GetAutomation retrieves one automation by ID.
func (s *SQLiteStore) GetAutomation(id string) (*models.Automation, error) {
	var data []byte
	if err := s.db.QueryRow(`SELECT data FROM automations WHERE id = ?`, id).Scan(&data); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("automation not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get automation: %w", err)
	}
	automation := &models.Automation{}
	if err := json.Unmarshal(data, automation); err != nil {
		return nil, fmt.Errorf("failed to unmarshal automation: %w", err)
	}
	return automation, nil
}

// ListAutomationsByProject lists automations for one project ordered by recency.
func (s *SQLiteStore) ListAutomationsByProject(projectID string) ([]*models.Automation, error) {
	rows, err := s.db.Query(`SELECT data FROM automations WHERE project_id = ? ORDER BY created_at DESC`, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list automations: %w", err)
	}
	defer rows.Close()
	automations := make([]*models.Automation, 0)
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, fmt.Errorf("failed to scan automation: %w", err)
		}
		automation := &models.Automation{}
		if err := json.Unmarshal(data, automation); err != nil {
			return nil, fmt.Errorf("failed to unmarshal automation: %w", err)
		}
		automations = append(automations, automation)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating automations: %w", err)
	}
	return automations, nil
}

// DeleteAutomation deletes one automation by ID.
func (s *SQLiteStore) DeleteAutomation(id string) error {
	if _, err := s.db.Exec(`DELETE FROM automations WHERE id = ?`, id); err != nil {
		return fmt.Errorf("failed to delete automation: %w", err)
	}
	return nil
}

// SaveTwinProcessingRun upserts one persisted twin-processing run.
func (s *SQLiteStore) SaveTwinProcessingRun(run *models.TwinProcessingRun) error {
	data, err := json.Marshal(run)
	if err != nil {
		return fmt.Errorf("failed to marshal twin processing run: %w", err)
	}
	query := `
		INSERT OR REPLACE INTO twin_processing_runs (
			id, project_id, twin_id, status, trigger_type, automation_id, requested_at, started_at, completed_at, data
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err = s.db.Exec(query,
		run.ID,
		run.ProjectID,
		run.DigitalTwinID,
		run.Status,
		run.TriggerType,
		nullableString(run.AutomationID),
		run.RequestedAt,
		run.StartedAt,
		run.CompletedAt,
		data,
	)
	if err != nil {
		return fmt.Errorf("failed to save twin processing run: %w", err)
	}
	return nil
}

// GetTwinProcessingRun retrieves one persisted twin-processing run by ID.
func (s *SQLiteStore) GetTwinProcessingRun(id string) (*models.TwinProcessingRun, error) {
	var data []byte
	if err := s.db.QueryRow(`SELECT data FROM twin_processing_runs WHERE id = ?`, id).Scan(&data); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("twin processing run not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get twin processing run: %w", err)
	}
	run := &models.TwinProcessingRun{}
	if err := json.Unmarshal(data, run); err != nil {
		return nil, fmt.Errorf("failed to unmarshal twin processing run: %w", err)
	}
	return run, nil
}

// GetActiveTwinProcessingRun retrieves the newest queued or running run for one twin.
func (s *SQLiteStore) GetActiveTwinProcessingRun(twinID string) (*models.TwinProcessingRun, error) {
	var data []byte
	err := s.db.QueryRow(`
		SELECT data FROM twin_processing_runs
		WHERE twin_id = ? AND status IN (?, ?)
		ORDER BY requested_at DESC LIMIT 1
	`, twinID, models.TwinProcessingRunStatusQueued, models.TwinProcessingRunStatusRunning).Scan(&data)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get active twin processing run: %w", err)
	}
	run := &models.TwinProcessingRun{}
	if err := json.Unmarshal(data, run); err != nil {
		return nil, fmt.Errorf("failed to unmarshal twin processing run: %w", err)
	}
	return run, nil
}

// ListTwinProcessingRunsByDigitalTwin lists persisted runs for one twin ordered by recency.
func (s *SQLiteStore) ListTwinProcessingRunsByDigitalTwin(twinID string, limit int) ([]*models.TwinProcessingRun, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.Query(`SELECT data FROM twin_processing_runs WHERE twin_id = ? ORDER BY requested_at DESC LIMIT ?`, twinID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list twin processing runs: %w", err)
	}
	defer rows.Close()
	runs := make([]*models.TwinProcessingRun, 0)
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, fmt.Errorf("failed to scan twin processing run: %w", err)
		}
		run := &models.TwinProcessingRun{}
		if err := json.Unmarshal(data, run); err != nil {
			return nil, fmt.Errorf("failed to unmarshal twin processing run: %w", err)
		}
		runs = append(runs, run)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating twin processing runs: %w", err)
	}
	return runs, nil
}

// DeleteTwinProcessingRun deletes one persisted run by ID.
func (s *SQLiteStore) DeleteTwinProcessingRun(id string) error {
	if _, err := s.db.Exec(`DELETE FROM twin_processing_runs WHERE id = ?`, id); err != nil {
		return fmt.Errorf("failed to delete twin processing run: %w", err)
	}
	return nil
}

// SaveAlertEvent persists one append-only alert event.
func (s *SQLiteStore) SaveAlertEvent(event *models.AlertEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal alert event: %w", err)
	}
	query := `
		INSERT OR REPLACE INTO alert_events (
			id, project_id, twin_id, processing_run_id, severity, category, created_at, triggered_export_pipeline_id, triggered_work_task_id, data
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err = s.db.Exec(query,
		event.ID,
		event.ProjectID,
		event.DigitalTwinID,
		event.ProcessingRunID,
		event.Severity,
		event.Category,
		event.CreatedAt,
		nullableString(event.TriggeredExportPipelineID),
		nullableString(event.TriggeredWorkTaskID),
		data,
	)
	if err != nil {
		return fmt.Errorf("failed to save alert event: %w", err)
	}
	return nil
}

// GetAlertEvent retrieves one persisted alert event by ID.
func (s *SQLiteStore) GetAlertEvent(id string) (*models.AlertEvent, error) {
	var data []byte
	if err := s.db.QueryRow(`SELECT data FROM alert_events WHERE id = ?`, id).Scan(&data); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("alert event not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get alert event: %w", err)
	}
	event := &models.AlertEvent{}
	if err := json.Unmarshal(data, event); err != nil {
		return nil, fmt.Errorf("failed to unmarshal alert event: %w", err)
	}
	return event, nil
}

// ListAlertEventsByDigitalTwin lists alert events for one twin ordered by recency.
func (s *SQLiteStore) ListAlertEventsByDigitalTwin(twinID string, limit int) ([]*models.AlertEvent, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.Query(`SELECT data FROM alert_events WHERE twin_id = ? ORDER BY created_at DESC LIMIT ?`, twinID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list alert events: %w", err)
	}
	defer rows.Close()
	events := make([]*models.AlertEvent, 0)
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, fmt.Errorf("failed to scan alert event: %w", err)
		}
		event := &models.AlertEvent{}
		if err := json.Unmarshal(data, event); err != nil {
			return nil, fmt.Errorf("failed to unmarshal alert event: %w", err)
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating alert events: %w", err)
	}
	return events, nil
}

// DeleteAlertEvent deletes one alert event by ID.
func (s *SQLiteStore) DeleteAlertEvent(id string) error {
	if _, err := s.db.Exec(`DELETE FROM alert_events WHERE id = ?`, id); err != nil {
		return fmt.Errorf("failed to delete alert event: %w", err)
	}
	return nil
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

// SaveEntity saves an entity to the database and appends a revision snapshot.
func (s *SQLiteStore) SaveEntity(entity *models.Entity) error {
	data, err := json.Marshal(entity)
	if err != nil {
		return fmt.Errorf("failed to marshal entity: %w", err)
	}
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin entity transaction: %w", err)
	}
	defer tx.Rollback()
	query := `
		INSERT INTO dt_entities (
			id, twin_id, entity_type, source_data_id,
			created_at, updated_at, data
		) VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			twin_id = excluded.twin_id,
			entity_type = excluded.entity_type,
			source_data_id = excluded.source_data_id,
			created_at = excluded.created_at,
			updated_at = excluded.updated_at,
			data = excluded.data
	`
	_, err = tx.Exec(query, entity.ID, entity.DigitalTwinID, entity.Type, entity.SourceDataID, entity.CreatedAt.Format(time.RFC3339), entity.UpdatedAt.Format(time.RFC3339), data)
	if err != nil {
		return fmt.Errorf("failed to save entity: %w", err)
	}
	var nextRevision int
	if err := tx.QueryRow(`SELECT COALESCE(MAX(revision), 0) + 1 FROM dt_entity_revisions WHERE entity_id = ?`, entity.ID).Scan(&nextRevision); err != nil {
		return fmt.Errorf("failed to calculate entity revision: %w", err)
	}
	revision := &models.EntityRevision{
		ID:             uuid.New().String(),
		EntityID:       entity.ID,
		DigitalTwinID:  entity.DigitalTwinID,
		Revision:       nextRevision,
		Attributes:     cloneJSONMap(entity.Attributes),
		Modifications:  cloneJSONMap(entity.Modifications),
		ComputedValues: cloneJSONMap(entity.ComputedValues),
		Relationships:  cloneRelationships(entity.Relationships),
		RecordedAt:     entity.UpdatedAt.UTC(),
	}
	revisionData, err := json.Marshal(revision)
	if err != nil {
		return fmt.Errorf("failed to marshal entity revision: %w", err)
	}
	_, err = tx.Exec(`INSERT INTO dt_entity_revisions (id, entity_id, twin_id, revision, recorded_at, data) VALUES (?, ?, ?, ?, ?, ?)`, revision.ID, revision.EntityID, revision.DigitalTwinID, revision.Revision, revision.RecordedAt.Format(time.RFC3339), revisionData)
	if err != nil {
		return fmt.Errorf("failed to save entity revision: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit entity transaction: %w", err)
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

// ListEntityRevisions lists historical entity snapshots for one entity.
func (s *SQLiteStore) ListEntityRevisions(entityID string, limit int) ([]*models.EntityRevision, error) {
	query := `SELECT data FROM dt_entity_revisions WHERE entity_id = ? ORDER BY revision DESC`
	args := []interface{}{entityID}
	if limit > 0 {
		query += ` LIMIT ?`
		args = append(args, limit)
	}
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list entity revisions: %w", err)
	}
	defer rows.Close()
	revisions := make([]*models.EntityRevision, 0)
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, fmt.Errorf("failed to scan entity revision: %w", err)
		}
		var revision models.EntityRevision
		if err := json.Unmarshal(data, &revision); err != nil {
			return nil, fmt.Errorf("failed to unmarshal entity revision: %w", err)
		}
		revisions = append(revisions, &revision)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating entity revisions: %w", err)
	}
	return revisions, nil
}

// ListEntitiesByTypeInTwin lists all entities of a specific type within a digital twin.
// Uses the existing twin_id + entity_type indices for efficient lookup.
func (s *SQLiteStore) ListEntitiesByTypeInTwin(twinID, entityType string) ([]*models.Entity, error) {
	query := `SELECT data FROM dt_entities WHERE twin_id = ? AND entity_type = ? ORDER BY created_at DESC`

	rows, err := s.db.Query(query, twinID, entityType)
	if err != nil {
		return nil, fmt.Errorf("failed to list entities by type: %w", err)
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

// ── Analysis Runs ─────────────────────────────────────────────────────────────

// SaveAnalysisRun upserts a persisted analysis run.
func (s *SQLiteStore) SaveAnalysisRun(run *models.AnalysisRun) error {
	return s.SaveResolverRun(run, nil)
}

// SaveResolverRun atomically persists one resolver run and its current review items.
func (s *SQLiteStore) SaveResolverRun(run *models.AnalysisRun, items []*models.ReviewItem) error {
	return s.retryOnBusy(func() error {
		tx, err := s.db.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin resolver transaction: %w", err)
		}
		if err := saveAnalysisRunTx(tx, run); err != nil {
			_ = tx.Rollback()
			return err
		}
		for _, item := range items {
			if err := saveReviewItemTx(tx, item); err != nil {
				_ = tx.Rollback()
				return err
			}
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit resolver transaction: %w", err)
		}
		return nil
	}, 5)
}

// GetAnalysisRun retrieves a single persisted analysis run by ID.
func (s *SQLiteStore) GetAnalysisRun(id string) (*models.AnalysisRun, error) {
	var data string
	if err := s.db.QueryRow(`SELECT data FROM analysis_runs WHERE id = ?`, id).Scan(&data); err != nil {
		return nil, fmt.Errorf("analysis run not found: %w", err)
	}
	run := &models.AnalysisRun{}
	if err := json.Unmarshal([]byte(data), run); err != nil {
		return nil, fmt.Errorf("failed to unmarshal analysis run: %w", err)
	}
	return run, nil
}

// ListAnalysisRunsByProject lists persisted analysis runs for one project.
func (s *SQLiteStore) ListAnalysisRunsByProject(projectID string) ([]*models.AnalysisRun, error) {
	rows, err := s.db.Query(`SELECT data FROM analysis_runs WHERE project_id = ? ORDER BY created_at DESC`, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list analysis runs: %w", err)
	}
	defer rows.Close()
	result := make([]*models.AnalysisRun, 0)
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, fmt.Errorf("failed to scan analysis run: %w", err)
		}
		run := &models.AnalysisRun{}
		if err := json.Unmarshal([]byte(data), run); err != nil {
			return nil, fmt.Errorf("failed to unmarshal analysis run: %w", err)
		}
		result = append(result, run)
	}
	return result, nil
}

// SaveReviewItem upserts a persisted review item.
func (s *SQLiteStore) SaveReviewItem(item *models.ReviewItem) error {
	return s.retryOnBusy(func() error {
		return saveReviewItemTx(s.db, item)
	}, 5)
}

// GetReviewItem retrieves a single persisted review item by ID.
func (s *SQLiteStore) GetReviewItem(id string) (*models.ReviewItem, error) {
	var data string
	if err := s.db.QueryRow(`SELECT data FROM review_items WHERE id = ?`, id).Scan(&data); err != nil {
		return nil, fmt.Errorf("review item not found: %w", err)
	}
	item := &models.ReviewItem{}
	if err := json.Unmarshal([]byte(data), item); err != nil {
		return nil, fmt.Errorf("failed to unmarshal review item: %w", err)
	}
	return item, nil
}

// GetReviewItemByFindingKey retrieves the most recent persisted review item for one finding key.
func (s *SQLiteStore) GetReviewItemByFindingKey(projectID, findingKey string) (*models.ReviewItem, error) {
	items, err := s.ListReviewItems(projectID)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		if item.FindingKey == findingKey {
			return item, nil
		}
	}
	return nil, nil
}

// ListReviewItems lists review items for a project ordered by recency.
func (s *SQLiteStore) ListReviewItems(projectID string) ([]*models.ReviewItem, error) {
	rows, err := s.db.Query(`SELECT data FROM review_items WHERE project_id = ? ORDER BY created_at DESC`, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list review items: %w", err)
	}
	defer rows.Close()
	result := make([]*models.ReviewItem, 0)
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, fmt.Errorf("failed to scan review item: %w", err)
		}
		item := &models.ReviewItem{}
		if err := json.Unmarshal([]byte(data), item); err != nil {
			return nil, fmt.Errorf("failed to unmarshal review item: %w", err)
		}
		result = append(result, item)
	}
	return result, nil
}

// SaveInsight upserts a persisted insight.
func (s *SQLiteStore) SaveInsight(insight *models.Insight) error {
	return s.retryOnBusy(func() error {
		return saveInsightTx(s.db, insight)
	}, 5)
}

// SaveInsightRun atomically persists one insight run and all generated insights.
func (s *SQLiteStore) SaveInsightRun(run *models.AnalysisRun, insights []*models.Insight) error {
	return s.retryOnBusy(func() error {
		tx, err := s.db.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin insight transaction: %w", err)
		}
		if err := saveAnalysisRunTx(tx, run); err != nil {
			_ = tx.Rollback()
			return err
		}
		for _, insight := range insights {
			if err := saveInsightTx(tx, insight); err != nil {
				_ = tx.Rollback()
				return err
			}
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit insight transaction: %w", err)
		}
		return nil
	}, 5)
}

// GetInsight retrieves a single persisted insight by ID.
func (s *SQLiteStore) GetInsight(id string) (*models.Insight, error) {
	var data string
	if err := s.db.QueryRow(`SELECT data FROM insights WHERE id = ?`, id).Scan(&data); err != nil {
		return nil, fmt.Errorf("insight not found: %w", err)
	}
	insight := &models.Insight{}
	if err := json.Unmarshal([]byte(data), insight); err != nil {
		return nil, fmt.Errorf("failed to unmarshal insight: %w", err)
	}
	return insight, nil
}

// ListInsightsByProject lists persisted insights for one project ordered by recency.
func (s *SQLiteStore) ListInsightsByProject(projectID string) ([]*models.Insight, error) {
	rows, err := s.db.Query(`SELECT data FROM insights WHERE project_id = ? ORDER BY created_at DESC`, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list insights: %w", err)
	}
	defer rows.Close()
	result := make([]*models.Insight, 0)
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, fmt.Errorf("failed to scan insight: %w", err)
		}
		insight := &models.Insight{}
		if err := json.Unmarshal([]byte(data), insight); err != nil {
			return nil, fmt.Errorf("failed to unmarshal insight: %w", err)
		}
		result = append(result, insight)
	}
	return result, nil
}

type sqlExecer interface {
	Exec(query string, args ...any) (sql.Result, error)
}

func saveAnalysisRunTx(exec sqlExecer, run *models.AnalysisRun) error {
	data, err := json.Marshal(run)
	if err != nil {
		return fmt.Errorf("failed to marshal analysis run: %w", err)
	}
	query := `
		INSERT OR REPLACE INTO analysis_runs (id, project_id, kind, status, created_at, completed_at, data)
		VALUES (?, ?, ?, ?, ?, ?, ?)`
	if _, err := exec.Exec(query, run.ID, run.ProjectID, run.Kind, run.Status, run.CreatedAt, run.CompletedAt, string(data)); err != nil {
		return fmt.Errorf("failed to save analysis run: %w", err)
	}
	return nil
}

func saveReviewItemTx(exec sqlExecer, item *models.ReviewItem) error {
	data, err := json.Marshal(item)
	if err != nil {
		return fmt.Errorf("failed to marshal review item: %w", err)
	}
	query := `
		INSERT OR REPLACE INTO review_items (id, project_id, run_id, finding_type, status, confidence, created_at, updated_at, data)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
	if _, err := exec.Exec(query, item.ID, item.ProjectID, item.RunID, item.FindingType, item.Status, item.Confidence, item.CreatedAt, item.UpdatedAt, string(data)); err != nil {
		return fmt.Errorf("failed to save review item: %w", err)
	}
	return nil
}

func saveInsightTx(exec sqlExecer, insight *models.Insight) error {
	data, err := json.Marshal(insight)
	if err != nil {
		return fmt.Errorf("failed to marshal insight: %w", err)
	}
	query := `
		INSERT OR REPLACE INTO insights (id, project_id, run_id, type, severity, confidence, status, created_at, updated_at, data)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	if _, err := exec.Exec(query, insight.ID, insight.ProjectID, insight.RunID, insight.Type, insight.Severity, insight.Confidence, insight.Status, insight.CreatedAt, insight.UpdatedAt, string(data)); err != nil {
		return fmt.Errorf("failed to save insight: %w", err)
	}
	return nil
}

func cloneJSONMap(values map[string]interface{}) map[string]interface{} {
	if values == nil {
		return nil
	}
	cloned := make(map[string]interface{}, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

func cloneRelationships(rels []*models.EntityRelationship) []*models.EntityRelationship {
	if rels == nil {
		return nil
	}
	cloned := make([]*models.EntityRelationship, 0, len(rels))
	for _, rel := range rels {
		if rel == nil {
			continue
		}
		copy := *rel
		copy.Properties = cloneJSONMap(rel.Properties)
		cloned = append(cloned, &copy)
	}
	return cloned
}
