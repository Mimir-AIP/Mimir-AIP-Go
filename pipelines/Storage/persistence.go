package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// PersistenceBackend manages SQLite database for metadata storage
type PersistenceBackend struct {
	db *sql.DB
}

// NewPersistenceBackend creates a new persistence backend with SQLite
func NewPersistenceBackend(dbPath string) (*PersistenceBackend, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable foreign keys and WAL mode for better concurrency
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}
	if _, err := db.Exec("PRAGMA journal_mode = WAL"); err != nil {
		return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	backend := &PersistenceBackend{db: db}
	if err := backend.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return backend, nil
}

// initSchema creates all necessary tables
func (p *PersistenceBackend) initSchema() error {
	schema := `
	-- Ontologies table
	CREATE TABLE IF NOT EXISTS ontologies (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		description TEXT,
		version TEXT NOT NULL,
		file_path TEXT NOT NULL,
		tdb2_graph TEXT NOT NULL,
		format TEXT NOT NULL DEFAULT 'turtle',
		status TEXT NOT NULL DEFAULT 'active',
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		created_by TEXT,
		metadata TEXT,
		UNIQUE(name, version)
	);

	-- Ontology classes table
	CREATE TABLE IF NOT EXISTS ontology_classes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		ontology_id TEXT NOT NULL,
		uri TEXT NOT NULL,
		label TEXT,
		description TEXT,
		parent_uris TEXT,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (ontology_id) REFERENCES ontologies(id) ON DELETE CASCADE,
		UNIQUE(ontology_id, uri)
	);

	-- Ontology properties table
	CREATE TABLE IF NOT EXISTS ontology_properties (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		ontology_id TEXT NOT NULL,
		uri TEXT NOT NULL,
		label TEXT,
		property_type TEXT NOT NULL,
		domain TEXT,
		range TEXT,
		description TEXT,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (ontology_id) REFERENCES ontologies(id) ON DELETE CASCADE,
		UNIQUE(ontology_id, uri)
	);

	-- Ontology versions table (for tracking evolution)
	CREATE TABLE IF NOT EXISTS ontology_versions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		ontology_id TEXT NOT NULL,
		version TEXT NOT NULL,
		previous_version TEXT,
		changelog TEXT,
		migration_strategy TEXT,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		created_by TEXT,
		FOREIGN KEY (ontology_id) REFERENCES ontologies(id) ON DELETE CASCADE
	);

	-- Ontology changes table (detailed changelog)
	CREATE TABLE IF NOT EXISTS ontology_changes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		version_id INTEGER NOT NULL,
		change_type TEXT NOT NULL,
		entity_type TEXT NOT NULL,
		entity_uri TEXT NOT NULL,
		old_value TEXT,
		new_value TEXT,
		description TEXT,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (version_id) REFERENCES ontology_versions(id) ON DELETE CASCADE
	);

	-- Ontology suggestions table (for drift detection)
	CREATE TABLE IF NOT EXISTS ontology_suggestions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		ontology_id TEXT NOT NULL,
		suggestion_type TEXT NOT NULL,
		entity_type TEXT NOT NULL,
		entity_uri TEXT,
		confidence REAL NOT NULL,
		reasoning TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'pending',
		risk_level TEXT NOT NULL DEFAULT 'low',
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		reviewed_at TIMESTAMP,
		reviewed_by TEXT,
		review_decision TEXT,
		review_notes TEXT,
		FOREIGN KEY (ontology_id) REFERENCES ontologies(id) ON DELETE CASCADE
	);

	-- Extraction jobs table (for entity extraction tracking)
	CREATE TABLE IF NOT EXISTS extraction_jobs (
		id TEXT PRIMARY KEY,
		ontology_id TEXT NOT NULL,
		pipeline_id TEXT,
		job_name TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'pending',
		extraction_type TEXT NOT NULL,
		source_type TEXT NOT NULL,
		source_path TEXT,
		entities_extracted INTEGER DEFAULT 0,
		triples_generated INTEGER DEFAULT 0,
		error_message TEXT,
		started_at TIMESTAMP,
		completed_at TIMESTAMP,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		metadata TEXT,
		FOREIGN KEY (ontology_id) REFERENCES ontologies(id) ON DELETE CASCADE
	);

	-- Extracted entities table (for tracking extracted entities)
	CREATE TABLE IF NOT EXISTS extracted_entities (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		job_id TEXT NOT NULL,
		entity_uri TEXT NOT NULL,
		entity_type TEXT NOT NULL,
		entity_label TEXT,
		confidence REAL,
		source_text TEXT,
		properties TEXT,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (job_id) REFERENCES extraction_jobs(id) ON DELETE CASCADE
	);

	-- Drift detection runs table
	CREATE TABLE IF NOT EXISTS drift_detections (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		ontology_id TEXT NOT NULL,
		detection_type TEXT NOT NULL,
		data_source TEXT NOT NULL,
		suggestions_generated INTEGER DEFAULT 0,
		status TEXT NOT NULL DEFAULT 'running',
		started_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		completed_at TIMESTAMP,
		error_message TEXT,
		FOREIGN KEY (ontology_id) REFERENCES ontologies(id) ON DELETE CASCADE
	);

	-- Auto-update policies table
	CREATE TABLE IF NOT EXISTS auto_update_policies (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		ontology_id TEXT NOT NULL UNIQUE,
		enabled BOOLEAN NOT NULL DEFAULT 0,
		auto_apply_classes BOOLEAN NOT NULL DEFAULT 0,
		auto_apply_properties BOOLEAN NOT NULL DEFAULT 0,
		auto_apply_modify BOOLEAN NOT NULL DEFAULT 0,
		max_risk_level TEXT NOT NULL DEFAULT 'low',
		min_confidence REAL NOT NULL DEFAULT 0.8,
		require_approval TEXT,
		notification_email TEXT,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (ontology_id) REFERENCES ontologies(id) ON DELETE CASCADE
	);

	-- Digital twins table
	CREATE TABLE IF NOT EXISTS digital_twins (
		id TEXT PRIMARY KEY,
		ontology_id TEXT NOT NULL,
		name TEXT NOT NULL,
		description TEXT,
		model_type TEXT NOT NULL,
		base_state TEXT NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (ontology_id) REFERENCES ontologies(id) ON DELETE CASCADE
	);

	-- Simulation scenarios table
	CREATE TABLE IF NOT EXISTS simulation_scenarios (
		id TEXT PRIMARY KEY,
		twin_id TEXT NOT NULL,
		name TEXT NOT NULL,
		description TEXT,
		scenario_type TEXT,
		events TEXT NOT NULL,
		duration INTEGER,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (twin_id) REFERENCES digital_twins(id) ON DELETE CASCADE
	);

	-- Simulation runs table
	CREATE TABLE IF NOT EXISTS simulation_runs (
		id TEXT PRIMARY KEY,
		scenario_id TEXT NOT NULL,
		status TEXT NOT NULL,
		start_time TIMESTAMP,
		end_time TIMESTAMP,
		initial_state TEXT,
		final_state TEXT,
		metrics TEXT,
		events_log TEXT,
		error_message TEXT,
		FOREIGN KEY (scenario_id) REFERENCES simulation_scenarios(id) ON DELETE CASCADE
	);

	-- Temporal state snapshots table
	CREATE TABLE IF NOT EXISTS twin_state_snapshots (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		run_id TEXT NOT NULL,
		timestamp TIMESTAMP NOT NULL,
		step_number INTEGER NOT NULL,
		state TEXT NOT NULL,
		description TEXT,
		metrics TEXT,
		FOREIGN KEY (run_id) REFERENCES simulation_runs(id) ON DELETE CASCADE
	);

	-- Agent conversations table
	CREATE TABLE IF NOT EXISTS agent_conversations (
		id TEXT PRIMARY KEY,
		twin_id TEXT,
		title TEXT NOT NULL,
		model_provider TEXT NOT NULL DEFAULT 'openai',
		model_name TEXT NOT NULL DEFAULT 'gpt-4',
		system_prompt TEXT,
		context_summary TEXT,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (twin_id) REFERENCES digital_twins(id) ON DELETE SET NULL
	);

	-- Agent messages table
	CREATE TABLE IF NOT EXISTS agent_messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		conversation_id TEXT NOT NULL,
		role TEXT NOT NULL,
		content TEXT NOT NULL,
		tool_calls TEXT,
		tool_results TEXT,
		metadata TEXT,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (conversation_id) REFERENCES agent_conversations(id) ON DELETE CASCADE
	);

	-- Create indexes for better query performance
	CREATE INDEX IF NOT EXISTS idx_ontologies_name ON ontologies(name);
	CREATE INDEX IF NOT EXISTS idx_ontologies_status ON ontologies(status);
	CREATE INDEX IF NOT EXISTS idx_ontology_classes_uri ON ontology_classes(uri);
	CREATE INDEX IF NOT EXISTS idx_ontology_properties_uri ON ontology_properties(uri);
	CREATE INDEX IF NOT EXISTS idx_suggestions_status ON ontology_suggestions(status);
	CREATE INDEX IF NOT EXISTS idx_suggestions_ontology ON ontology_suggestions(ontology_id, status);
	CREATE INDEX IF NOT EXISTS idx_extraction_jobs_status ON extraction_jobs(status);
	CREATE INDEX IF NOT EXISTS idx_extraction_jobs_ontology ON extraction_jobs(ontology_id);
	CREATE INDEX IF NOT EXISTS idx_extracted_entities_job ON extracted_entities(job_id);
	CREATE INDEX IF NOT EXISTS idx_drift_detections_ontology ON drift_detections(ontology_id);
	CREATE INDEX IF NOT EXISTS idx_digital_twins_ontology ON digital_twins(ontology_id);
	CREATE INDEX IF NOT EXISTS idx_simulation_scenarios_twin ON simulation_scenarios(twin_id);
	CREATE INDEX IF NOT EXISTS idx_simulation_runs_scenario ON simulation_runs(scenario_id);
	CREATE INDEX IF NOT EXISTS idx_simulation_runs_status ON simulation_runs(status);
	CREATE INDEX IF NOT EXISTS idx_twin_snapshots_run ON twin_state_snapshots(run_id);
	CREATE INDEX IF NOT EXISTS idx_twin_snapshots_step ON twin_state_snapshots(run_id, step_number);
	CREATE INDEX IF NOT EXISTS idx_agent_conversations_twin ON agent_conversations(twin_id);
	CREATE INDEX IF NOT EXISTS idx_agent_conversations_updated ON agent_conversations(updated_at DESC);
	CREATE INDEX IF NOT EXISTS idx_agent_messages_conversation ON agent_messages(conversation_id);
	CREATE INDEX IF NOT EXISTS idx_agent_messages_created ON agent_messages(conversation_id, created_at);

	-- API keys table for LLM provider credentials
	CREATE TABLE IF NOT EXISTS api_keys (
		id TEXT PRIMARY KEY,
		provider TEXT NOT NULL,  -- openai, anthropic, ollama, etc.
		name TEXT NOT NULL,  -- User-friendly name (e.g., "My OpenAI Key")
		key_value TEXT NOT NULL,  -- Encrypted API key
		endpoint_url TEXT,  -- Custom endpoint (for Ollama, etc.)
		is_active BOOLEAN NOT NULL DEFAULT 1,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		last_used_at TIMESTAMP,
		metadata TEXT  -- JSON: extra config like model defaults, rate limits, etc.
	);

	-- Plugins table for tracking installed plugins
	CREATE TABLE IF NOT EXISTS plugins (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL UNIQUE,
		type TEXT NOT NULL,  -- input, output, ai, data_processing, etc.
		version TEXT NOT NULL,
		file_path TEXT NOT NULL,  -- Path to .so/.dll file
		description TEXT,
		author TEXT,
		is_enabled BOOLEAN NOT NULL DEFAULT 1,
		is_builtin BOOLEAN NOT NULL DEFAULT 0,  -- Built-in vs user-uploaded
		config TEXT,  -- JSON: plugin-specific configuration
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_api_keys_provider ON api_keys(provider);
	CREATE INDEX IF NOT EXISTS idx_api_keys_active ON api_keys(is_active);
	CREATE INDEX IF NOT EXISTS idx_plugins_type ON plugins(type);
	CREATE INDEX IF NOT EXISTS idx_plugins_enabled ON plugins(is_enabled);
	`

	_, err := p.db.Exec(schema)
	return err
}

// Ontology represents an ontology in the database
type Ontology struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Version     string    `json:"version"`
	FilePath    string    `json:"file_path"`
	TDB2Graph   string    `json:"tdb2_graph"`
	Format      string    `json:"format"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	CreatedBy   string    `json:"created_by"`
	Metadata    string    `json:"metadata"`
}

// CreateOntology inserts a new ontology into the database
func (p *PersistenceBackend) CreateOntology(ctx context.Context, ont *Ontology) error {
	query := `
		INSERT INTO ontologies (id, name, description, version, file_path, tdb2_graph, format, status, created_by, metadata)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := p.db.ExecContext(ctx, query,
		ont.ID, ont.Name, ont.Description, ont.Version, ont.FilePath,
		ont.TDB2Graph, ont.Format, ont.Status, ont.CreatedBy, ont.Metadata,
	)
	return err
}

// GetOntology retrieves an ontology by ID
func (p *PersistenceBackend) GetOntology(ctx context.Context, id string) (*Ontology, error) {
	query := `
		SELECT id, name, description, version, file_path, tdb2_graph, format, status,
		       created_at, updated_at, created_by, metadata
		FROM ontologies
		WHERE id = ?
	`
	ont := &Ontology{}
	err := p.db.QueryRowContext(ctx, query, id).Scan(
		&ont.ID, &ont.Name, &ont.Description, &ont.Version, &ont.FilePath,
		&ont.TDB2Graph, &ont.Format, &ont.Status, &ont.CreatedAt, &ont.UpdatedAt,
		&ont.CreatedBy, &ont.Metadata,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("ontology not found: %s", id)
	}
	return ont, err
}

// ListOntologies returns all ontologies with optional status filter
func (p *PersistenceBackend) ListOntologies(ctx context.Context, status string) ([]*Ontology, error) {
	var query string
	var args []any
	if status != "" {
		query = `
			SELECT id, name, description, version, file_path, tdb2_graph, format, status,
			       created_at, updated_at, created_by, metadata
			FROM ontologies
			WHERE status = ?
			ORDER BY created_at DESC
		`
		args = append(args, status)
	} else {
		query = `
			SELECT id, name, description, version, file_path, tdb2_graph, format, status,
			       created_at, updated_at, created_by, metadata
			FROM ontologies
			ORDER BY created_at DESC
		`
	}

	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ontologies []*Ontology
	for rows.Next() {
		ont := &Ontology{}
		err := rows.Scan(
			&ont.ID, &ont.Name, &ont.Description, &ont.Version, &ont.FilePath,
			&ont.TDB2Graph, &ont.Format, &ont.Status, &ont.CreatedAt, &ont.UpdatedAt,
			&ont.CreatedBy, &ont.Metadata,
		)
		if err != nil {
			return nil, err
		}
		ontologies = append(ontologies, ont)
	}
	return ontologies, rows.Err()
}

// UpdateOntologyStatus updates the status of an ontology
func (p *PersistenceBackend) UpdateOntologyStatus(ctx context.Context, id, status string) error {
	query := `UPDATE ontologies SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := p.db.ExecContext(ctx, query, status, id)
	return err
}

// DeleteOntology deletes an ontology and all related data (cascades)
func (p *PersistenceBackend) DeleteOntology(ctx context.Context, id string) error {
	query := `DELETE FROM ontologies WHERE id = ?`
	_, err := p.db.ExecContext(ctx, query, id)
	return err
}

// Close closes the database connection
func (p *PersistenceBackend) Close() error {
	return p.db.Close()
}

// GetDB returns the underlying database connection
func (p *PersistenceBackend) GetDB() *sql.DB {
	return p.db
}

// Health checks database connectivity
func (p *PersistenceBackend) Health(ctx context.Context) error {
	return p.db.PingContext(ctx)
}
