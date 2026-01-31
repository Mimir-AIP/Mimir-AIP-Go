package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
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

	// Configure connection pool for SQLite
	// WAL mode allows multiple readers, so we can have more connections
	// But we limit to prevent excessive lock contention
	db.SetMaxOpenConns(10)                 // Allow up to 10 concurrent connections
	db.SetMaxIdleConns(2)                  // Keep 2 idle connections
	db.SetConnMaxLifetime(0)               // Connections never expire
	db.SetConnMaxIdleTime(5 * time.Minute) // Idle connections expire after 5min

	// Enable foreign keys and WAL mode for better concurrency
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}
	if _, err := db.Exec("PRAGMA journal_mode = WAL"); err != nil {
		return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	// Set busy timeout to 30 seconds to handle concurrent access
	if _, err := db.Exec("PRAGMA busy_timeout = 30000"); err != nil {
		return nil, fmt.Errorf("failed to set busy timeout: %w", err)
	}

	// Optimize WAL mode performance
	if _, err := db.Exec("PRAGMA synchronous = NORMAL"); err != nil {
		return nil, fmt.Errorf("failed to set synchronous mode: %w", err)
	}
	if _, err := db.Exec("PRAGMA cache_size = -64000"); err != nil {
		return nil, fmt.Errorf("failed to set cache size: %w", err)
	}
	if _, err := db.Exec("PRAGMA temp_store = MEMORY"); err != nil {
		return nil, fmt.Errorf("failed to set temp store: %w", err)
	}
	if _, err := db.Exec("PRAGMA mmap_size = 30000000000"); err != nil {
		return nil, fmt.Errorf("failed to set mmap size: %w", err)
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
		auto_version BOOLEAN NOT NULL DEFAULT 1,
		auto_train_models BOOLEAN NOT NULL DEFAULT 0,
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
		auto_version BOOLEAN NOT NULL DEFAULT 1,
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
		ontology_id TEXT,
		name TEXT NOT NULL,
		description TEXT,
		model_type TEXT NOT NULL DEFAULT 'default_model',
		base_state TEXT NOT NULL DEFAULT '{}',
		status TEXT NOT NULL DEFAULT 'active',
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
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

	-- Classifier models table (for ML pipeline)
	CREATE TABLE IF NOT EXISTS classifier_models (
		id TEXT PRIMARY KEY,
		ontology_id TEXT,  -- Made nullable to support standalone training
		name TEXT NOT NULL,
		target_class TEXT NOT NULL,
		algorithm TEXT NOT NULL DEFAULT 'decision_tree',
		hyperparameters TEXT,  -- JSON: max_depth, min_samples_split, etc.
		feature_columns TEXT,  -- JSON array: ["col1", "col2", ...]
		class_labels TEXT,  -- JSON array: ["class1", "class2", ...]
		train_accuracy REAL,
		validate_accuracy REAL,
		precision_score REAL,
		recall_score REAL,
		f1_score REAL,
		confusion_matrix TEXT,  -- JSON: 2D array
		model_artifact_path TEXT NOT NULL,  -- Path to serialized model file
		model_size_bytes INTEGER,
		training_rows INTEGER,
		validation_rows INTEGER,
		feature_importance TEXT,  -- JSON: {"feature": importance}
		is_active BOOLEAN NOT NULL DEFAULT 1,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (ontology_id) REFERENCES ontologies(id) ON DELETE CASCADE
	);

	-- Model training runs table (for tracking training history)
	CREATE TABLE IF NOT EXISTS model_training_runs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		model_id TEXT NOT NULL,
		dataset_size INTEGER NOT NULL,
		training_duration_ms INTEGER,
		epochs INTEGER DEFAULT 1,
		final_train_accuracy REAL,
		final_validate_accuracy REAL,
		metrics TEXT,  -- JSON: full metrics snapshot
		config TEXT,  -- JSON: hyperparameters used
		status TEXT NOT NULL DEFAULT 'running',  -- running, completed, failed
		error_message TEXT,
		started_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		completed_at TIMESTAMP,
		FOREIGN KEY (model_id) REFERENCES classifier_models(id) ON DELETE CASCADE
	);

	-- Model predictions table (for tracking inference and anomaly detection)
	CREATE TABLE IF NOT EXISTS model_predictions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		model_id TEXT NOT NULL,
		input_data TEXT NOT NULL,  -- JSON: feature values
		predicted_class TEXT NOT NULL,
		confidence REAL NOT NULL,
		actual_class TEXT,  -- If known (for validation)
		is_correct BOOLEAN,  -- If actual_class is known
		is_anomaly BOOLEAN DEFAULT 0,  -- Flagged as anomaly if confidence < threshold
		anomaly_reason TEXT,  -- Why flagged as anomaly
		predicted_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (model_id) REFERENCES classifier_models(id) ON DELETE CASCADE
	);

	-- Anomalies table (for anomaly detection system)
	CREATE TABLE IF NOT EXISTS anomalies (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		model_id TEXT NOT NULL,
		prediction_id INTEGER,  -- Link to prediction if applicable
		anomaly_type TEXT NOT NULL,  -- 'low_confidence', 'constraint_violation', 'outlier', 'drift'
		data_row TEXT NOT NULL,  -- JSON: raw data that triggered anomaly
		confidence REAL,  -- Model confidence (if prediction-based)
		violations TEXT,  -- JSON: list of constraint violations
		severity TEXT NOT NULL DEFAULT 'medium',  -- 'low', 'medium', 'high', 'critical'
		status TEXT NOT NULL DEFAULT 'open',  -- 'open', 'investigating', 'resolved', 'false_positive'
		assigned_to TEXT,
		resolution_notes TEXT,
		detected_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		resolved_at TIMESTAMP,
		FOREIGN KEY (model_id) REFERENCES classifier_models(id) ON DELETE CASCADE,
		FOREIGN KEY (prediction_id) REFERENCES model_predictions(id) ON DELETE SET NULL
	);

	-- Data quality metrics table (for profiling and monitoring)
	CREATE TABLE IF NOT EXISTS data_quality_metrics (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		ontology_id TEXT NOT NULL,
		metric_type TEXT NOT NULL,  -- 'completeness', 'validity', 'consistency', 'accuracy'
		entity_type TEXT,  -- Which class/table
		column_name TEXT,  -- Which property/column
		metric_value REAL NOT NULL,
		threshold_min REAL,
		threshold_max REAL,
		is_passing BOOLEAN NOT NULL,
		details TEXT,  -- JSON: additional context
		measured_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (ontology_id) REFERENCES ontologies(id) ON DELETE CASCADE
	);

	-- Create indexes for ML tables
	CREATE INDEX IF NOT EXISTS idx_classifier_models_ontology ON classifier_models(ontology_id);
	CREATE INDEX IF NOT EXISTS idx_classifier_models_active ON classifier_models(is_active);
	CREATE INDEX IF NOT EXISTS idx_training_runs_model ON model_training_runs(model_id);
	CREATE INDEX IF NOT EXISTS idx_training_runs_status ON model_training_runs(status);
	CREATE INDEX IF NOT EXISTS idx_predictions_model ON model_predictions(model_id);
	CREATE INDEX IF NOT EXISTS idx_predictions_anomaly ON model_predictions(is_anomaly);
	CREATE INDEX IF NOT EXISTS idx_predictions_predicted_at ON model_predictions(predicted_at DESC);
	CREATE INDEX IF NOT EXISTS idx_anomalies_model ON anomalies(model_id);
	CREATE INDEX IF NOT EXISTS idx_anomalies_status ON anomalies(status);
	CREATE INDEX IF NOT EXISTS idx_anomalies_severity ON anomalies(severity);
	CREATE INDEX IF NOT EXISTS idx_anomalies_detected_at ON anomalies(detected_at DESC);
	CREATE INDEX IF NOT EXISTS idx_data_quality_ontology ON data_quality_metrics(ontology_id);
	CREATE INDEX IF NOT EXISTS idx_data_quality_measured_at ON data_quality_metrics(measured_at DESC);

	-- Time series data table (for continuous monitoring)
	CREATE TABLE IF NOT EXISTS time_series_data (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		entity_id TEXT NOT NULL,  -- e.g., "product_123", "supply_medical"
		metric_name TEXT NOT NULL,  -- e.g., "price", "stock_level", "deliveries"
		value REAL NOT NULL,
		timestamp TIMESTAMP NOT NULL,
		ontology_id TEXT,  -- Optional link to ontology
		metadata TEXT,  -- JSON: additional context
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (ontology_id) REFERENCES ontologies(id) ON DELETE SET NULL
	);

	-- Time series analysis results table
	CREATE TABLE IF NOT EXISTS time_series_analyses (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		entity_id TEXT NOT NULL,
		metric_name TEXT NOT NULL,
		analysis_type TEXT NOT NULL,  -- 'trend', 'anomaly', 'forecast'
		window_days INTEGER,
		result TEXT NOT NULL,  -- JSON: analysis results
		analyzed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	-- Alerts table (for continuous monitoring alerts)
	CREATE TABLE IF NOT EXISTS alerts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		ontology_id TEXT,  -- Optional link to ontology
		alert_type TEXT NOT NULL,  -- 'trend', 'anomaly', 'threshold', 'forecast'
		entity_id TEXT NOT NULL,
		metric_name TEXT NOT NULL,
		severity TEXT NOT NULL,  -- 'low', 'medium', 'high', 'critical'
		title TEXT NOT NULL,
		message TEXT NOT NULL,
		details TEXT,  -- JSON: additional details
		value REAL,  -- The metric value that triggered the alert
		threshold REAL,  -- The threshold value (if applicable)
		status TEXT NOT NULL DEFAULT 'active',  -- 'active', 'acknowledged', 'resolved', 'dismissed'
		acknowledged_by TEXT,
		acknowledged_at TIMESTAMP,
		resolved_at TIMESTAMP,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (ontology_id) REFERENCES ontologies(id) ON DELETE SET NULL
	);

	-- Monitoring rules table (user-defined thresholds and rules)
	CREATE TABLE IF NOT EXISTS monitoring_rules (
		id TEXT PRIMARY KEY,
		ontology_id TEXT,
		entity_id TEXT,  -- NULL means apply to all entities
		metric_name TEXT NOT NULL,
		rule_type TEXT NOT NULL,  -- 'threshold', 'trend', 'anomaly', 'forecast'
		condition TEXT NOT NULL,  -- JSON: rule conditions
		severity TEXT NOT NULL DEFAULT 'medium',
		is_enabled BOOLEAN NOT NULL DEFAULT 1,
		alert_channels TEXT,  -- JSON: ['email', 'webhook', 'agent_chat']
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (ontology_id) REFERENCES ontologies(id) ON DELETE CASCADE
	);

	-- Monitoring jobs table (scheduled monitoring tasks)
	CREATE TABLE IF NOT EXISTS monitoring_jobs (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		ontology_id TEXT NOT NULL,
		description TEXT,
		cron_expr TEXT NOT NULL,
		metrics TEXT NOT NULL,  -- JSON: array of metric names to monitor
		rules TEXT,  -- JSON: array of rule IDs to evaluate
		is_enabled BOOLEAN NOT NULL DEFAULT 1,
		last_run_at TIMESTAMP,
		last_run_status TEXT,  -- 'success', 'failed', 'partial'
		last_run_alerts INTEGER DEFAULT 0,  -- Count of alerts created in last run
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (ontology_id) REFERENCES ontologies(id) ON DELETE CASCADE
	);

	-- Monitoring job execution history
	CREATE TABLE IF NOT EXISTS monitoring_job_runs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		job_id TEXT NOT NULL,
		started_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		completed_at TIMESTAMP,
		status TEXT NOT NULL,  -- 'running', 'success', 'failed', 'partial'
		metrics_checked INTEGER DEFAULT 0,
		rules_evaluated INTEGER DEFAULT 0,
		alerts_created INTEGER DEFAULT 0,
		error_message TEXT,
		FOREIGN KEY (job_id) REFERENCES monitoring_jobs(id) ON DELETE CASCADE
	);

	-- Alert actions table (defines automated actions when alerts are triggered)
	CREATE TABLE IF NOT EXISTS alert_actions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		rule_id TEXT,  -- Optional: link to specific monitoring rule
		alert_type TEXT,  -- Optional: trigger for specific alert types ('threshold', 'anomaly', etc.)
		severity TEXT,  -- Optional: trigger for specific severity levels
		action_type TEXT NOT NULL,  -- 'execute_pipeline', 'send_email', 'webhook', 'agent_notification'
		config TEXT NOT NULL,  -- JSON: action-specific configuration
		is_enabled BOOLEAN NOT NULL DEFAULT 1,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (rule_id) REFERENCES monitoring_rules(id) ON DELETE CASCADE
	);

	-- Alert action execution history
	CREATE TABLE IF NOT EXISTS alert_action_executions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		action_id INTEGER NOT NULL,
		alert_id INTEGER NOT NULL,
		status TEXT NOT NULL,  -- 'success', 'failed'
		started_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		completed_at TIMESTAMP,
		error_message TEXT,
		result TEXT,  -- JSON: execution result details
		FOREIGN KEY (action_id) REFERENCES alert_actions(id) ON DELETE CASCADE,
		FOREIGN KEY (alert_id) REFERENCES alerts(id) ON DELETE CASCADE
	);

	-- Scheduler jobs table (for crash recovery of scheduled jobs)
	CREATE TABLE IF NOT EXISTS scheduler_jobs (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		job_type TEXT NOT NULL,  -- 'pipeline' or 'monitoring'
		pipeline TEXT,  -- Pipeline YAML path (for pipeline jobs)
		monitoring_job_id TEXT,  -- Reference to monitoring_jobs (for monitoring jobs)
		cron_expr TEXT NOT NULL,
		is_enabled BOOLEAN NOT NULL DEFAULT 1,
		next_run TIMESTAMP,
		last_run TIMESTAMP,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (monitoring_job_id) REFERENCES monitoring_jobs(id) ON DELETE CASCADE
	);

	-- Create indexes for time series tables
	CREATE INDEX IF NOT EXISTS idx_ts_data_entity_metric ON time_series_data(entity_id, metric_name);
	CREATE INDEX IF NOT EXISTS idx_ts_data_timestamp ON time_series_data(timestamp DESC);
	CREATE INDEX IF NOT EXISTS idx_ts_data_ontology ON time_series_data(ontology_id);
	CREATE INDEX IF NOT EXISTS idx_ts_analyses_entity ON time_series_analyses(entity_id, metric_name);
	CREATE INDEX IF NOT EXISTS idx_ts_analyses_type ON time_series_analyses(analysis_type);
	CREATE INDEX IF NOT EXISTS idx_alerts_status ON alerts(status);
	CREATE INDEX IF NOT EXISTS idx_alerts_severity ON alerts(severity);
	CREATE INDEX IF NOT EXISTS idx_alerts_entity ON alerts(entity_id, metric_name);
	CREATE INDEX IF NOT EXISTS idx_alerts_ontology ON alerts(ontology_id);
	CREATE INDEX IF NOT EXISTS idx_alerts_created_at ON alerts(created_at DESC);
	CREATE INDEX IF NOT EXISTS idx_monitoring_rules_entity ON monitoring_rules(entity_id, metric_name);
	CREATE INDEX IF NOT EXISTS idx_monitoring_rules_enabled ON monitoring_rules(is_enabled);
	CREATE INDEX IF NOT EXISTS idx_scheduler_jobs_enabled ON scheduler_jobs(is_enabled);
	CREATE INDEX IF NOT EXISTS idx_scheduler_jobs_next_run ON scheduler_jobs(next_run);
	CREATE INDEX IF NOT EXISTS idx_scheduler_jobs_type ON scheduler_jobs(job_type);


	`

	_, err := p.db.Exec(schema)
	if err != nil {
		return err
	}

	// Run migrations for existing databases
	return p.runMigrations()
}

// runMigrations applies schema updates for existing databases
func (p *PersistenceBackend) runMigrations() error {
	migrations := []string{
		// Alerts table columns
		`ALTER TABLE alerts ADD COLUMN ontology_id TEXT;`,
		`ALTER TABLE alerts ADD COLUMN value REAL;`,
		`ALTER TABLE alerts ADD COLUMN threshold REAL;`,
		`CREATE INDEX IF NOT EXISTS idx_alerts_ontology ON alerts(ontology_id);`,

		// Ontologies: auto-twin creation flag
		`ALTER TABLE ontologies ADD COLUMN auto_create_twins BOOLEAN NOT NULL DEFAULT 0;`,

		// Ontologies: auto-versioning flag (default true)
		`ALTER TABLE ontologies ADD COLUMN auto_version BOOLEAN NOT NULL DEFAULT 1;`,

		// Auto-update policies: auto-versioning flag (default true)
		`ALTER TABLE auto_update_policies ADD COLUMN auto_version BOOLEAN NOT NULL DEFAULT 1;`,

		// Digital twins: link to model that created it
		`ALTER TABLE digital_twins ADD COLUMN model_id TEXT;`,
		`CREATE INDEX IF NOT EXISTS idx_digital_twins_model ON digital_twins(model_id);`,
	}

	for _, migration := range migrations {
		// Ignore errors for columns that already exist
		_, _ = p.db.Exec(migration)
	}

	return nil
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
	AutoVersion bool      `json:"auto_version"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	CreatedBy   string    `json:"created_by"`
	Metadata    string    `json:"metadata"`
}

// CreateOntology inserts a new ontology into the database
func (p *PersistenceBackend) CreateOntology(ctx context.Context, ont *Ontology) error {
	query := `
		INSERT INTO ontologies (id, name, description, version, file_path, tdb2_graph, format, status, auto_version, created_by, metadata)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := p.db.ExecContext(ctx, query,
		ont.ID, ont.Name, ont.Description, ont.Version, ont.FilePath,
		ont.TDB2Graph, ont.Format, ont.Status, ont.AutoVersion, ont.CreatedBy, ont.Metadata,
	)
	return err
}

// GetOntology retrieves an ontology by ID
func (p *PersistenceBackend) GetOntology(ctx context.Context, id string) (*Ontology, error) {
	query := `
		SELECT id, name, description, version, file_path, tdb2_graph, format, status, auto_version,
		       created_at, updated_at, created_by, metadata
		FROM ontologies
		WHERE id = ?
	`
	ont := &Ontology{}
	var createdBy, metadata sql.NullString
	err := p.db.QueryRowContext(ctx, query, id).Scan(
		&ont.ID, &ont.Name, &ont.Description, &ont.Version, &ont.FilePath,
		&ont.TDB2Graph, &ont.Format, &ont.Status, &ont.AutoVersion, &ont.CreatedAt, &ont.UpdatedAt,
		&createdBy, &metadata,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("ontology not found: %s", id)
	}
	ont.CreatedBy = createdBy.String
	ont.Metadata = metadata.String
	return ont, err
}

// ListOntologies returns all ontologies with optional status filter
func (p *PersistenceBackend) ListOntologies(ctx context.Context, status string) ([]*Ontology, error) {
	var query string
	var args []any
	if status != "" {
		query = `
			SELECT id, name, description, version, file_path, tdb2_graph, format, status, auto_version,
			       created_at, updated_at, created_by, metadata
			FROM ontologies
			WHERE status = ?
			ORDER BY created_at DESC
		`
		args = append(args, status)
	} else {
		query = `
			SELECT id, name, description, version, file_path, tdb2_graph, format, status, auto_version,
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
		var createdBy, metadata sql.NullString
		err := rows.Scan(
			&ont.ID, &ont.Name, &ont.Description, &ont.Version, &ont.FilePath,
			&ont.TDB2Graph, &ont.Format, &ont.Status, &ont.AutoVersion, &ont.CreatedAt, &ont.UpdatedAt,
			&createdBy, &metadata,
		)
		if err != nil {
			return nil, err
		}
		ont.CreatedBy = createdBy.String
		ont.Metadata = metadata.String
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

// UpdateOntology updates an ontology's metadata and content
func (p *PersistenceBackend) UpdateOntology(ctx context.Context, ont *Ontology) error {
	query := `
		UPDATE ontologies 
		SET name = ?, description = ?, version = ?, file_path = ?, 
		    tdb2_graph = ?, format = ?, status = ?, auto_version = ?,
		    updated_at = CURRENT_TIMESTAMP, metadata = ?
		WHERE id = ?
	`
	_, err := p.db.ExecContext(ctx, query,
		ont.Name, ont.Description, ont.Version, ont.FilePath,
		ont.TDB2Graph, ont.Format, ont.Status, ont.AutoVersion,
		ont.Metadata, ont.ID,
	)
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

// CreateDigitalTwin creates a new digital twin in the database
func (p *PersistenceBackend) CreateDigitalTwin(ctx context.Context, id, ontologyID, name, description, modelType, baseState string) error {
	query := `
		INSERT INTO digital_twins (id, ontology_id, name, description, model_type, base_state, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`
	_, err := p.db.ExecContext(ctx, query, id, ontologyID, name, description, modelType, baseState)
	if err != nil {
		return fmt.Errorf("failed to create digital twin: %w", err)
	}
	return nil
}

// GetDigitalTwin retrieves a digital twin by ID
func (p *PersistenceBackend) GetDigitalTwin(ctx context.Context, id string) (map[string]interface{}, error) {
	query := `SELECT id, ontology_id, name, description, model_type, base_state, created_at, updated_at FROM digital_twins WHERE id = ?`

	var twin struct {
		ID          string
		OntologyID  string
		Name        string
		Description sql.NullString
		ModelType   string
		BaseState   string
		CreatedAt   time.Time
		UpdatedAt   time.Time
	}

	err := p.db.QueryRowContext(ctx, query, id).Scan(
		&twin.ID, &twin.OntologyID, &twin.Name, &twin.Description,
		&twin.ModelType, &twin.BaseState, &twin.CreatedAt, &twin.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("digital twin not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get digital twin: %w", err)
	}

	result := map[string]interface{}{
		"id":          twin.ID,
		"ontology_id": twin.OntologyID,
		"name":        twin.Name,
		"model_type":  twin.ModelType,
		"base_state":  twin.BaseState,
		"created_at":  twin.CreatedAt.Format(time.RFC3339),
		"updated_at":  twin.UpdatedAt.Format(time.RFC3339),
	}

	if twin.Description.Valid {
		result["description"] = twin.Description.String
	}

	return result, nil
}

// ListDigitalTwins retrieves all digital twins
func (p *PersistenceBackend) ListDigitalTwins(ctx context.Context) ([]map[string]interface{}, error) {
	query := `SELECT id, ontology_id, name, description, model_type, created_at, updated_at FROM digital_twins ORDER BY created_at DESC`

	rows, err := p.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list digital twins: %w", err)
	}
	defer rows.Close()

	var twins []map[string]interface{}
	for rows.Next() {
		var twin struct {
			ID          string
			OntologyID  string
			Name        string
			Description sql.NullString
			ModelType   string
			CreatedAt   time.Time
			UpdatedAt   time.Time
		}

		err := rows.Scan(&twin.ID, &twin.OntologyID, &twin.Name, &twin.Description, &twin.ModelType, &twin.CreatedAt, &twin.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan digital twin: %w", err)
		}

		result := map[string]interface{}{
			"id":          twin.ID,
			"ontology_id": twin.OntologyID,
			"name":        twin.Name,
			"model_type":  twin.ModelType,
			"created_at":  twin.CreatedAt.Format(time.RFC3339),
			"updated_at":  twin.UpdatedAt.Format(time.RFC3339),
		}

		if twin.Description.Valid {
			result["description"] = twin.Description.String
		}

		twins = append(twins, result)
	}

	return twins, rows.Err()
}

// ClassifierModel represents a classifier model in the database
type ClassifierModel struct {
	ID                string    `json:"id"`
	OntologyID        string    `json:"ontology_id"`
	Name              string    `json:"name"`
	TargetClass       string    `json:"target_class"`
	Algorithm         string    `json:"algorithm"`
	Hyperparameters   string    `json:"hyperparameters"` // JSON
	FeatureColumns    string    `json:"feature_columns"` // JSON array
	ClassLabels       string    `json:"class_labels"`    // JSON array
	TrainAccuracy     float64   `json:"train_accuracy"`
	ValidateAccuracy  float64   `json:"validate_accuracy"`
	PrecisionScore    float64   `json:"precision_score"`
	RecallScore       float64   `json:"recall_score"`
	F1Score           float64   `json:"f1_score"`
	ConfusionMatrix   string    `json:"confusion_matrix"` // JSON
	ModelArtifactPath string    `json:"model_artifact_path"`
	ModelSizeBytes    int64     `json:"model_size_bytes"`
	TrainingRows      int       `json:"training_rows"`
	ValidationRows    int       `json:"validation_rows"`
	FeatureImportance string    `json:"feature_importance"` // JSON
	IsActive          bool      `json:"is_active"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// SaveMLModelDirect saves an ML model with direct JSON inputs (used by auto_trainer)
func (p *PersistenceBackend) SaveMLModelDirect(ctx context.Context, modelID, ontologyID, modelJSON, configJSON, metricsJSON string) error {
	// Parse the metrics to extract accuracy information
	var metrics struct {
		TrainAccuracy     float64 `json:"train_accuracy,omitempty"`
		ValidateAccuracy  float64 `json:"validate_accuracy,omitempty"`
		PrecisionScore    float64 `json:"precision,omitempty"`
		RecallScore       float64 `json:"recall,omitempty"`
		F1Score           float64 `json:"f1,omitempty"`
		ConfusionMatrix   string  `json:"confusion_matrix,omitempty"`
		FeatureImportance string  `json:"feature_importance,omitempty"`
		TrainingRows      int     `json:"training_rows,omitempty"`
		ValidationRows    int     `json:"validation_rows,omitempty"`
	}

	if err := json.Unmarshal([]byte(metricsJSON), &metrics); err != nil {
		// If parsing fails, use default values
		metrics.TrainAccuracy = 0
		metrics.ValidateAccuracy = 0
	}

	// Parse config to get algorithm and other details
	var config struct {
		Algorithm       string `json:"algorithm,omitempty"`
		TargetClass     string `json:"target_class,omitempty"`
		Hyperparameters string `json:"hyperparameters,omitempty"`
		FeatureColumns  string `json:"feature_columns,omitempty"`
		ClassLabels     string `json:"class_labels,omitempty"`
	}

	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		config.Algorithm = "random_forest" // default
	}

	// Create model artifact path (for now, store JSON directly)
	modelArtifactPath := fmt.Sprintf("ml_models/%s.json", modelID)

	// Create classifier model entry
	model := &ClassifierModel{
		ID:                modelID,
		OntologyID:        ontologyID,
		Name:              fmt.Sprintf("AutoML Model %s", modelID),
		TargetClass:       config.TargetClass,
		Algorithm:         config.Algorithm,
		Hyperparameters:   config.Hyperparameters,
		FeatureColumns:    config.FeatureColumns,
		ClassLabels:       config.ClassLabels,
		TrainAccuracy:     metrics.TrainAccuracy,
		ValidateAccuracy:  metrics.ValidateAccuracy,
		PrecisionScore:    metrics.PrecisionScore,
		RecallScore:       metrics.RecallScore,
		F1Score:           metrics.F1Score,
		ConfusionMatrix:   metrics.ConfusionMatrix,
		ModelArtifactPath: modelArtifactPath,
		ModelSizeBytes:    int64(len(modelJSON)),
		TrainingRows:      metrics.TrainingRows,
		ValidationRows:    metrics.ValidationRows,
		FeatureImportance: metrics.FeatureImportance,
		IsActive:          true,
	}

	return p.CreateClassifierModel(ctx, model)
}

// CreateClassifierModel creates a new classifier model in the database
func (p *PersistenceBackend) CreateClassifierModel(ctx context.Context, model *ClassifierModel) error {
	query := `
		INSERT INTO classifier_models (
			id, ontology_id, name, target_class, algorithm, hyperparameters,
			feature_columns, class_labels, train_accuracy, validate_accuracy,
			precision_score, recall_score, f1_score, confusion_matrix,
			model_artifact_path, model_size_bytes, training_rows, validation_rows,
			feature_importance, is_active, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`

	// Handle nullable ontology_id
	var ontologyID interface{}
	if model.OntologyID == "" {
		ontologyID = nil
	} else {
		ontologyID = model.OntologyID
	}

	_, err := p.db.ExecContext(ctx, query,
		model.ID, ontologyID, model.Name, model.TargetClass, model.Algorithm,
		model.Hyperparameters, model.FeatureColumns, model.ClassLabels,
		model.TrainAccuracy, model.ValidateAccuracy, model.PrecisionScore,
		model.RecallScore, model.F1Score, model.ConfusionMatrix,
		model.ModelArtifactPath, model.ModelSizeBytes, model.TrainingRows,
		model.ValidationRows, model.FeatureImportance, model.IsActive,
	)
	if err != nil {
		return fmt.Errorf("failed to create classifier model: %w", err)
	}
	return nil
}

// GetClassifierModel retrieves a classifier model by ID
func (p *PersistenceBackend) GetClassifierModel(ctx context.Context, id string) (*ClassifierModel, error) {
	query := `
		SELECT id, ontology_id, name, target_class, algorithm, hyperparameters,
		       feature_columns, class_labels, train_accuracy, validate_accuracy,
		       precision_score, recall_score, f1_score, confusion_matrix,
		       model_artifact_path, model_size_bytes, training_rows, validation_rows,
		       feature_importance, is_active, created_at, updated_at
		FROM classifier_models
		WHERE id = ?
	`
	model := &ClassifierModel{}
	var ontologyID, hyperparameters, featureColumns, classLabels sql.NullString
	var confusionMatrix, modelArtifactPath, featureImportance sql.NullString
	var trainAccuracy, validateAccuracy, precisionScore, recallScore, f1Score sql.NullFloat64
	var modelSizeBytes sql.NullInt64
	var trainingRows, validationRows sql.NullInt32
	err := p.db.QueryRowContext(ctx, query, id).Scan(
		&model.ID, &ontologyID, &model.Name, &model.TargetClass,
		&model.Algorithm, &hyperparameters, &featureColumns,
		&classLabels, &trainAccuracy, &validateAccuracy,
		&precisionScore, &recallScore, &f1Score,
		&confusionMatrix, &modelArtifactPath, &modelSizeBytes,
		&trainingRows, &validationRows, &featureImportance,
		&model.IsActive, &model.CreatedAt, &model.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("classifier model not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get classifier model: %w", err)
	}
	model.OntologyID = ontologyID.String
	model.Hyperparameters = hyperparameters.String
	model.FeatureColumns = featureColumns.String
	model.ClassLabels = classLabels.String
	model.ConfusionMatrix = confusionMatrix.String
	model.ModelArtifactPath = modelArtifactPath.String
	model.FeatureImportance = featureImportance.String
	// Handle NULL float values
	if trainAccuracy.Valid {
		model.TrainAccuracy = trainAccuracy.Float64
	}
	if validateAccuracy.Valid {
		model.ValidateAccuracy = validateAccuracy.Float64
	}
	if precisionScore.Valid {
		model.PrecisionScore = precisionScore.Float64
	}
	if recallScore.Valid {
		model.RecallScore = recallScore.Float64
	}
	if f1Score.Valid {
		model.F1Score = f1Score.Float64
	}
	// Handle NULL int values
	if modelSizeBytes.Valid {
		model.ModelSizeBytes = modelSizeBytes.Int64
	}
	if trainingRows.Valid {
		model.TrainingRows = int(trainingRows.Int32)
	}
	if validationRows.Valid {
		model.ValidationRows = int(validationRows.Int32)
	}
	return model, nil
}

// ListClassifierModels retrieves all classifier models for an ontology
func (p *PersistenceBackend) ListClassifierModels(ctx context.Context, ontologyID string, activeOnly bool) ([]*ClassifierModel, error) {
	var query string
	var args []any

	if activeOnly {
		query = `
			SELECT id, ontology_id, name, target_class, algorithm, hyperparameters,
			       feature_columns, class_labels, train_accuracy, validate_accuracy,
			       precision_score, recall_score, f1_score, confusion_matrix,
			       model_artifact_path, model_size_bytes, training_rows, validation_rows,
			       feature_importance, is_active, created_at, updated_at
			FROM classifier_models
			WHERE ontology_id = ? AND is_active = 1
			ORDER BY created_at DESC
		`
		args = append(args, ontologyID)
	} else if ontologyID != "" {
		query = `
			SELECT id, ontology_id, name, target_class, algorithm, hyperparameters,
			       feature_columns, class_labels, train_accuracy, validate_accuracy,
			       precision_score, recall_score, f1_score, confusion_matrix,
			       model_artifact_path, model_size_bytes, training_rows, validation_rows,
			       feature_importance, is_active, created_at, updated_at
			FROM classifier_models
			WHERE ontology_id = ?
			ORDER BY created_at DESC
		`
		args = append(args, ontologyID)
	} else {
		query = `
			SELECT id, ontology_id, name, target_class, algorithm, hyperparameters,
			       feature_columns, class_labels, train_accuracy, validate_accuracy,
			       precision_score, recall_score, f1_score, confusion_matrix,
			       model_artifact_path, model_size_bytes, training_rows, validation_rows,
			       feature_importance, is_active, created_at, updated_at
			FROM classifier_models
			ORDER BY created_at DESC
		`
	}

	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list classifier models: %w", err)
	}
	defer rows.Close()

	var models []*ClassifierModel
	for rows.Next() {
		model := &ClassifierModel{}
		var ontologyID, hyperparameters, featureColumns, classLabels sql.NullString
		var confusionMatrix, modelArtifactPath, featureImportance sql.NullString
		var trainAccuracy, validateAccuracy, precisionScore, recallScore, f1Score sql.NullFloat64
		var modelSizeBytes sql.NullInt64
		var trainingRows, validationRows sql.NullInt32
		err := rows.Scan(
			&model.ID, &ontologyID, &model.Name, &model.TargetClass,
			&model.Algorithm, &hyperparameters, &featureColumns,
			&classLabels, &trainAccuracy, &validateAccuracy,
			&precisionScore, &recallScore, &f1Score,
			&confusionMatrix, &modelArtifactPath, &modelSizeBytes,
			&trainingRows, &validationRows, &featureImportance,
			&model.IsActive, &model.CreatedAt, &model.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan classifier model: %w", err)
		}
		model.OntologyID = ontologyID.String
		model.Hyperparameters = hyperparameters.String
		model.FeatureColumns = featureColumns.String
		model.ClassLabels = classLabels.String
		model.ConfusionMatrix = confusionMatrix.String
		model.ModelArtifactPath = modelArtifactPath.String
		model.FeatureImportance = featureImportance.String

		// Handle nullable float fields
		if trainAccuracy.Valid {
			model.TrainAccuracy = trainAccuracy.Float64
		}
		if validateAccuracy.Valid {
			model.ValidateAccuracy = validateAccuracy.Float64
		}
		if precisionScore.Valid {
			model.PrecisionScore = precisionScore.Float64
		}
		if recallScore.Valid {
			model.RecallScore = recallScore.Float64
		}
		if f1Score.Valid {
			model.F1Score = f1Score.Float64
		}

		// Handle nullable int fields
		if modelSizeBytes.Valid {
			model.ModelSizeBytes = modelSizeBytes.Int64
		}
		if trainingRows.Valid {
			model.TrainingRows = int(trainingRows.Int32)
		}
		if validationRows.Valid {
			model.ValidationRows = int(validationRows.Int32)
		}

		models = append(models, model)
	}

	return models, rows.Err()
}

// UpdateClassifierModelStatus updates the is_active status of a model
func (p *PersistenceBackend) UpdateClassifierModelStatus(ctx context.Context, id string, isActive bool) error {
	query := `UPDATE classifier_models SET is_active = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := p.db.ExecContext(ctx, query, isActive, id)
	if err != nil {
		return fmt.Errorf("failed to update classifier model status: %w", err)
	}
	return nil
}

// DeleteClassifierModel deletes a classifier model
func (p *PersistenceBackend) DeleteClassifierModel(ctx context.Context, id string) error {
	query := `DELETE FROM classifier_models WHERE id = ?`
	_, err := p.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete classifier model: %w", err)
	}
	return nil
}

// CreateTrainingRun creates a new training run record
func (p *PersistenceBackend) CreateTrainingRun(ctx context.Context, modelID string, datasetSize, trainingDurationMs int, finalTrainAccuracy, finalValidateAccuracy float64, metrics, config, status, errorMessage string) (int64, error) {
	query := `
		INSERT INTO model_training_runs (
			model_id, dataset_size, training_duration_ms, final_train_accuracy,
			final_validate_accuracy, metrics, config, status, error_message,
			started_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
	`
	result, err := p.db.ExecContext(ctx, query,
		modelID, datasetSize, trainingDurationMs, finalTrainAccuracy,
		finalValidateAccuracy, metrics, config, status, errorMessage,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to create training run: %w", err)
	}

	runID, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get training run ID: %w", err)
	}
	return runID, nil
}

// UpdateTrainingRunStatus updates a training run's status and completion time
func (p *PersistenceBackend) UpdateTrainingRunStatus(ctx context.Context, runID int64, status, errorMessage string) error {
	query := `
		UPDATE model_training_runs
		SET status = ?, error_message = ?, completed_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`
	_, err := p.db.ExecContext(ctx, query, status, errorMessage, runID)
	if err != nil {
		return fmt.Errorf("failed to update training run status: %w", err)
	}
	return nil
}

// CreatePrediction creates a new prediction record
func (p *PersistenceBackend) CreatePrediction(ctx context.Context, modelID, inputData, predictedClass string, confidence float64, actualClass string, isCorrect, isAnomaly bool, anomalyReason string) error {
	query := `
		INSERT INTO model_predictions (
			model_id, input_data, predicted_class, confidence, actual_class,
			is_correct, is_anomaly, anomaly_reason, predicted_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
	`
	var actualClassPtr *string
	if actualClass != "" {
		actualClassPtr = &actualClass
	}

	_, err := p.db.ExecContext(ctx, query,
		modelID, inputData, predictedClass, confidence, actualClassPtr,
		isCorrect, isAnomaly, anomalyReason,
	)
	if err != nil {
		return fmt.Errorf("failed to create prediction: %w", err)
	}
	return nil
}

// CreateAnomaly creates a new anomaly record
func (p *PersistenceBackend) CreateAnomaly(ctx context.Context, modelID string, predictionID *int64, anomalyType, dataRow string, confidence *float64, violations, severity, status string) (int64, error) {
	query := `
		INSERT INTO anomalies (
			model_id, prediction_id, anomaly_type, data_row, confidence,
			violations, severity, status, detected_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
	`
	result, err := p.db.ExecContext(ctx, query,
		modelID, predictionID, anomalyType, dataRow, confidence,
		violations, severity, status,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to create anomaly: %w", err)
	}

	anomalyID, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get anomaly ID: %w", err)
	}
	return anomalyID, nil
}

// ListAnomalies retrieves anomalies with optional filters
func (p *PersistenceBackend) ListAnomalies(ctx context.Context, modelID, status, severity string, limit int) ([]map[string]interface{}, error) {
	query := `
		SELECT id, model_id, prediction_id, anomaly_type, data_row, confidence,
		       violations, severity, status, assigned_to, resolution_notes,
		       detected_at, resolved_at
		FROM anomalies
		WHERE 1=1
	`
	var args []any

	if modelID != "" {
		query += " AND model_id = ?"
		args = append(args, modelID)
	}
	if status != "" {
		query += " AND status = ?"
		args = append(args, status)
	}
	if severity != "" {
		query += " AND severity = ?"
		args = append(args, severity)
	}

	query += " ORDER BY detected_at DESC"

	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}

	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list anomalies: %w", err)
	}
	defer rows.Close()

	var anomalies []map[string]interface{}
	for rows.Next() {
		var anomaly struct {
			ID              int64
			ModelID         string
			PredictionID    sql.NullInt64
			AnomalyType     string
			DataRow         string
			Confidence      sql.NullFloat64
			Violations      sql.NullString
			Severity        string
			Status          string
			AssignedTo      sql.NullString
			ResolutionNotes sql.NullString
			DetectedAt      time.Time
			ResolvedAt      sql.NullTime
		}

		err := rows.Scan(
			&anomaly.ID, &anomaly.ModelID, &anomaly.PredictionID, &anomaly.AnomalyType,
			&anomaly.DataRow, &anomaly.Confidence, &anomaly.Violations,
			&anomaly.Severity, &anomaly.Status, &anomaly.AssignedTo,
			&anomaly.ResolutionNotes, &anomaly.DetectedAt, &anomaly.ResolvedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan anomaly: %w", err)
		}

		result := map[string]interface{}{
			"id":           anomaly.ID,
			"model_id":     anomaly.ModelID,
			"anomaly_type": anomaly.AnomalyType,
			"data_row":     anomaly.DataRow,
			"severity":     anomaly.Severity,
			"status":       anomaly.Status,
			"detected_at":  anomaly.DetectedAt.Format(time.RFC3339),
		}

		if anomaly.PredictionID.Valid {
			result["prediction_id"] = anomaly.PredictionID.Int64
		}
		if anomaly.Confidence.Valid {
			result["confidence"] = anomaly.Confidence.Float64
		}
		if anomaly.Violations.Valid {
			result["violations"] = anomaly.Violations.String
		}
		if anomaly.AssignedTo.Valid {
			result["assigned_to"] = anomaly.AssignedTo.String
		}
		if anomaly.ResolutionNotes.Valid {
			result["resolution_notes"] = anomaly.ResolutionNotes.String
		}
		if anomaly.ResolvedAt.Valid {
			result["resolved_at"] = anomaly.ResolvedAt.Time.Format(time.RFC3339)
		}

		anomalies = append(anomalies, result)
	}

	return anomalies, rows.Err()
}

// UpdateAnomalyStatus updates an anomaly's status and resolution info
func (p *PersistenceBackend) UpdateAnomalyStatus(ctx context.Context, id int64, status, assignedTo, resolutionNotes string) error {
	query := `
		UPDATE anomalies
		SET status = ?, assigned_to = ?, resolution_notes = ?,
		    resolved_at = CASE WHEN ? IN ('resolved', 'false_positive') THEN CURRENT_TIMESTAMP ELSE resolved_at END
		WHERE id = ?
	`
	_, err := p.db.ExecContext(ctx, query, status, assignedTo, resolutionNotes, status, id)
	if err != nil {
		return fmt.Errorf("failed to update anomaly status: %w", err)
	}
	return nil
}

// AddTimeSeriesPoint adds a single data point to time series
func (p *PersistenceBackend) AddTimeSeriesPoint(ctx context.Context, entityID, metricName string, value float64, timestamp time.Time, ontologyID, metadata string) error {
	query := `
		INSERT INTO time_series_data (entity_id, metric_name, value, timestamp, ontology_id, metadata)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	_, err := p.db.ExecContext(ctx, query, entityID, metricName, value, timestamp, ontologyID, metadata)
	if err != nil {
		return fmt.Errorf("failed to add time series point: %w", err)
	}
	return nil
}

// AddTimeSeriesPoints adds multiple data points in a batch
func (p *PersistenceBackend) AddTimeSeriesPoints(ctx context.Context, points []TimeSeriesPoint) error {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO time_series_data (entity_id, metric_name, value, timestamp, ontology_id, metadata)
		VALUES (?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, point := range points {
		_, err := stmt.ExecContext(ctx, point.EntityID, point.MetricName, point.Value, point.Timestamp, point.OntologyID, point.Metadata)
		if err != nil {
			return fmt.Errorf("failed to insert point: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

// TimeSeriesPoint represents a data point for batch insertion
type TimeSeriesPoint struct {
	EntityID   string
	MetricName string
	Value      float64
	Timestamp  time.Time
	OntologyID string
	Metadata   string
}

// GetTimeSeries retrieves time series data for an entity/metric
func (p *PersistenceBackend) GetTimeSeries(ctx context.Context, entityID, metricName string, fromTime, toTime time.Time, limit int) ([]TimeSeriesDataPoint, error) {
	query := `
		SELECT id, entity_id, metric_name, value, timestamp, ontology_id, metadata
		FROM time_series_data
		WHERE entity_id = ? AND metric_name = ?
		  AND timestamp >= ? AND timestamp <= ?
		ORDER BY timestamp DESC
	`

	args := []interface{}{entityID, metricName, fromTime, toTime}
	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}

	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query time series: %w", err)
	}
	defer rows.Close()

	var points []TimeSeriesDataPoint
	for rows.Next() {
		var point TimeSeriesDataPoint
		var ontologyID, metadata sql.NullString

		err := rows.Scan(&point.ID, &point.EntityID, &point.MetricName, &point.Value, &point.Timestamp, &ontologyID, &metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to scan time series point: %w", err)
		}

		if ontologyID.Valid {
			point.OntologyID = ontologyID.String
		}
		if metadata.Valid {
			point.Metadata = metadata.String
		}

		points = append(points, point)
	}

	return points, rows.Err()
}

// TimeSeriesDataPoint represents a retrieved time series data point
type TimeSeriesDataPoint struct {
	ID         int64     `json:"id"`
	EntityID   string    `json:"entity_id"`
	MetricName string    `json:"metric_name"`
	Value      float64   `json:"value"`
	Timestamp  time.Time `json:"timestamp"`
	OntologyID string    `json:"ontology_id,omitempty"`
	Metadata   string    `json:"metadata,omitempty"`
}

// SaveTimeSeriesAnalysis saves analysis results
func (p *PersistenceBackend) SaveTimeSeriesAnalysis(ctx context.Context, entityID, metricName, analysisType string, windowDays int, result string) error {
	query := `
		INSERT INTO time_series_analyses (entity_id, metric_name, analysis_type, window_days, result)
		VALUES (?, ?, ?, ?, ?)
	`
	_, err := p.db.ExecContext(ctx, query, entityID, metricName, analysisType, windowDays, result)
	if err != nil {
		return fmt.Errorf("failed to save time series analysis: %w", err)
	}
	return nil
}

// CreateAlert creates a new alert
func (p *PersistenceBackend) CreateAlert(ctx context.Context, alertType, entityID, metricName, severity, title, message, details string) (int64, error) {
	query := `
		INSERT INTO alerts (alert_type, entity_id, metric_name, severity, title, message, details)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	result, err := p.db.ExecContext(ctx, query, alertType, entityID, metricName, severity, title, message, details)
	if err != nil {
		return 0, fmt.Errorf("failed to create alert: %w", err)
	}

	alertID, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get alert ID: %w", err)
	}
	return alertID, nil
}

// ListAlerts retrieves alerts with optional filters
func (p *PersistenceBackend) ListAlerts(ctx context.Context, status, severity, entityID string, limit int) ([]map[string]interface{}, error) {
	query := `
		SELECT id, alert_type, entity_id, metric_name, severity, title, message, details,
		       status, acknowledged_by, acknowledged_at, resolved_at, created_at
		FROM alerts
		WHERE 1=1
	`
	var args []interface{}

	if status != "" {
		query += " AND status = ?"
		args = append(args, status)
	}
	if severity != "" {
		query += " AND severity = ?"
		args = append(args, severity)
	}
	if entityID != "" {
		query += " AND entity_id = ?"
		args = append(args, entityID)
	}

	query += " ORDER BY created_at DESC"

	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}

	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list alerts: %w", err)
	}
	defer rows.Close()

	var alerts []map[string]interface{}
	for rows.Next() {
		var alert struct {
			ID             int64
			AlertType      string
			EntityID       string
			MetricName     string
			Severity       string
			Title          string
			Message        string
			Details        sql.NullString
			Status         string
			AcknowledgedBy sql.NullString
			AcknowledgedAt sql.NullTime
			ResolvedAt     sql.NullTime
			CreatedAt      time.Time
		}

		err := rows.Scan(
			&alert.ID, &alert.AlertType, &alert.EntityID, &alert.MetricName,
			&alert.Severity, &alert.Title, &alert.Message, &alert.Details,
			&alert.Status, &alert.AcknowledgedBy, &alert.AcknowledgedAt,
			&alert.ResolvedAt, &alert.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan alert: %w", err)
		}

		result := map[string]interface{}{
			"id":          alert.ID,
			"alert_type":  alert.AlertType,
			"entity_id":   alert.EntityID,
			"metric_name": alert.MetricName,
			"severity":    alert.Severity,
			"title":       alert.Title,
			"message":     alert.Message,
			"status":      alert.Status,
			"created_at":  alert.CreatedAt.Format(time.RFC3339),
		}

		if alert.Details.Valid {
			result["details"] = alert.Details.String
		}
		if alert.AcknowledgedBy.Valid {
			result["acknowledged_by"] = alert.AcknowledgedBy.String
		}
		if alert.AcknowledgedAt.Valid {
			result["acknowledged_at"] = alert.AcknowledgedAt.Time.Format(time.RFC3339)
		}
		if alert.ResolvedAt.Valid {
			result["resolved_at"] = alert.ResolvedAt.Time.Format(time.RFC3339)
		}

		alerts = append(alerts, result)
	}

	return alerts, rows.Err()
}

// UpdateAlertStatus updates an alert's status
func (p *PersistenceBackend) UpdateAlertStatus(ctx context.Context, alertID int64, status, acknowledgedBy string) error {
	query := `
		UPDATE alerts
		SET status = ?, acknowledged_by = ?,
		    acknowledged_at = CASE WHEN ? != 'active' THEN CURRENT_TIMESTAMP ELSE acknowledged_at END,
		    resolved_at = CASE WHEN ? IN ('resolved', 'dismissed') THEN CURRENT_TIMESTAMP ELSE resolved_at END
		WHERE id = ?
	`
	_, err := p.db.ExecContext(ctx, query, status, acknowledgedBy, status, status, alertID)
	if err != nil {
		return fmt.Errorf("failed to update alert status: %w", err)
	}
	return nil
}

// CreateMonitoringRule creates a monitoring rule
func (p *PersistenceBackend) CreateMonitoringRule(ctx context.Context, id, ontologyID, entityID, metricName, ruleType, condition, severity string, isEnabled bool, alertChannels string) error {
	query := `
		INSERT INTO monitoring_rules (id, ontology_id, entity_id, metric_name, rule_type, condition, severity, is_enabled, alert_channels)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := p.db.ExecContext(ctx, query, id, ontologyID, entityID, metricName, ruleType, condition, severity, isEnabled, alertChannels)
	if err != nil {
		return fmt.Errorf("failed to create monitoring rule: %w", err)
	}
	return nil
}

// GetMonitoringRules retrieves enabled monitoring rules
func (p *PersistenceBackend) GetMonitoringRules(ctx context.Context, entityID, metricName string) ([]MonitoringRule, error) {
	query := `
		SELECT id, ontology_id, entity_id, metric_name, rule_type, condition, severity, is_enabled, alert_channels, created_at, updated_at
		FROM monitoring_rules
		WHERE is_enabled = 1
	`
	var args []interface{}

	if entityID != "" {
		query += " AND (entity_id = ? OR entity_id IS NULL)"
		args = append(args, entityID)
	}
	if metricName != "" {
		query += " AND metric_name = ?"
		args = append(args, metricName)
	}

	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get monitoring rules: %w", err)
	}
	defer rows.Close()

	var rules []MonitoringRule
	for rows.Next() {
		var rule MonitoringRule
		var ontologyID, entityIDNull sql.NullString

		err := rows.Scan(
			&rule.ID, &ontologyID, &entityIDNull, &rule.MetricName,
			&rule.RuleType, &rule.Condition, &rule.Severity, &rule.IsEnabled,
			&rule.AlertChannels, &rule.CreatedAt, &rule.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan monitoring rule: %w", err)
		}

		if ontologyID.Valid {
			rule.OntologyID = ontologyID.String
		}
		if entityIDNull.Valid {
			rule.EntityID = entityIDNull.String
		}

		rules = append(rules, rule)
	}

	return rules, rows.Err()
}

// MonitoringRule represents a monitoring rule
type MonitoringRule struct {
	ID            string    `json:"id"`
	OntologyID    string    `json:"ontology_id"`
	EntityID      string    `json:"entity_id"`
	MetricName    string    `json:"metric_name"`
	RuleType      string    `json:"rule_type"`
	Condition     string    `json:"condition"`
	Severity      string    `json:"severity"`
	IsEnabled     bool      `json:"is_enabled"`
	AlertChannels string    `json:"alert_channels"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// MonitoringJob represents a scheduled monitoring task
type MonitoringJob struct {
	ID            string     `json:"id"`
	Name          string     `json:"name"`
	OntologyID    string     `json:"ontology_id"`
	Description   string     `json:"description"`
	CronExpr      string     `json:"cron_expr"`
	Metrics       string     `json:"metrics"` // JSON array
	Rules         string     `json:"rules"`   // JSON array
	IsEnabled     bool       `json:"is_enabled"`
	LastRunAt     *time.Time `json:"last_run_at"`
	LastRunStatus string     `json:"last_run_status"`
	LastRunAlerts int        `json:"last_run_alerts"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// MonitoringJobRun represents execution history for a monitoring job
type MonitoringJobRun struct {
	ID             int        `json:"id"`
	JobID          string     `json:"job_id"`
	StartedAt      time.Time  `json:"started_at"`
	CompletedAt    *time.Time `json:"completed_at"`
	Status         string     `json:"status"`
	MetricsChecked int        `json:"metrics_checked"`
	AlertsCreated  int        `json:"alerts_created"`
	ErrorMessage   string     `json:"error_message"`
}

// CreateMonitoringJob creates a new monitoring job
func (p *PersistenceBackend) CreateMonitoringJob(ctx context.Context, job *MonitoringJob) error {
	query := `
		INSERT INTO monitoring_jobs (
			id, name, ontology_id, description, cron_expr, metrics, rules,
			is_enabled, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	now := time.Now()
	_, err := p.db.ExecContext(ctx, query,
		job.ID, job.Name, job.OntologyID, job.Description, job.CronExpr,
		job.Metrics, job.Rules, job.IsEnabled, now, now,
	)
	if err != nil {
		return fmt.Errorf("failed to create monitoring job: %w", err)
	}
	job.CreatedAt = now
	job.UpdatedAt = now
	return nil
}

// GetMonitoringJob retrieves a monitoring job by ID
func (p *PersistenceBackend) GetMonitoringJob(ctx context.Context, id string) (*MonitoringJob, error) {
	query := `
		SELECT id, name, ontology_id, description, cron_expr, metrics, rules,
			is_enabled, last_run_at, last_run_status, last_run_alerts,
			created_at, updated_at
		FROM monitoring_jobs
		WHERE id = ?
	`
	job := &MonitoringJob{}
	var lastRunAt sql.NullTime
	var lastRunStatus, lastRunAlerts sql.NullString

	err := p.db.QueryRowContext(ctx, query, id).Scan(
		&job.ID, &job.Name, &job.OntologyID, &job.Description, &job.CronExpr,
		&job.Metrics, &job.Rules, &job.IsEnabled, &lastRunAt, &lastRunStatus,
		&lastRunAlerts, &job.CreatedAt, &job.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("monitoring job not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get monitoring job: %w", err)
	}

	if lastRunAt.Valid {
		job.LastRunAt = &lastRunAt.Time
	}
	if lastRunStatus.Valid {
		job.LastRunStatus = lastRunStatus.String
	}
	if lastRunAlerts.Valid {
		// Convert string to int
		if alerts, err := strconv.Atoi(lastRunAlerts.String); err == nil {
			job.LastRunAlerts = alerts
		}
	}

	return job, nil
}

// ListMonitoringJobs retrieves monitoring jobs with optional filters
func (p *PersistenceBackend) ListMonitoringJobs(ctx context.Context, ontologyID string, enabledOnly bool) ([]*MonitoringJob, error) {
	query := `
		SELECT id, name, ontology_id, description, cron_expr, metrics, rules,
			is_enabled, last_run_at, last_run_status, last_run_alerts,
			created_at, updated_at
		FROM monitoring_jobs
		WHERE 1=1
	`
	var args []interface{}

	if ontologyID != "" {
		query += " AND ontology_id = ?"
		args = append(args, ontologyID)
	}
	if enabledOnly {
		query += " AND is_enabled = 1"
	}

	query += " ORDER BY created_at DESC"

	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list monitoring jobs: %w", err)
	}
	defer rows.Close()

	var jobs []*MonitoringJob
	for rows.Next() {
		job := &MonitoringJob{}
		var lastRunAt sql.NullTime
		var lastRunStatus, lastRunAlerts sql.NullString

		err := rows.Scan(
			&job.ID, &job.Name, &job.OntologyID, &job.Description, &job.CronExpr,
			&job.Metrics, &job.Rules, &job.IsEnabled, &lastRunAt, &lastRunStatus,
			&lastRunAlerts, &job.CreatedAt, &job.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan monitoring job: %w", err)
		}

		if lastRunAt.Valid {
			job.LastRunAt = &lastRunAt.Time
		}
		if lastRunStatus.Valid {
			job.LastRunStatus = lastRunStatus.String
		}
		if lastRunAlerts.Valid {
			if alerts, err := strconv.Atoi(lastRunAlerts.String); err == nil {
				job.LastRunAlerts = alerts
			}
		}

		jobs = append(jobs, job)
	}

	return jobs, rows.Err()
}

// UpdateMonitoringJob updates an existing monitoring job
func (p *PersistenceBackend) UpdateMonitoringJob(ctx context.Context, job *MonitoringJob) error {
	query := `
		UPDATE monitoring_jobs
		SET name = ?, description = ?, cron_expr = ?, metrics = ?, rules = ?,
			is_enabled = ?, updated_at = ?
		WHERE id = ?
	`
	now := time.Now()
	result, err := p.db.ExecContext(ctx, query,
		job.Name, job.Description, job.CronExpr, job.Metrics, job.Rules,
		job.IsEnabled, now, job.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update monitoring job: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("monitoring job not found: %s", job.ID)
	}

	job.UpdatedAt = now
	return nil
}

// DeleteMonitoringJob deletes a monitoring job
func (p *PersistenceBackend) DeleteMonitoringJob(ctx context.Context, id string) error {
	query := `DELETE FROM monitoring_jobs WHERE id = ?`
	result, err := p.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete monitoring job: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("monitoring job not found: %s", id)
	}

	return nil
}

// UpdateMonitoringJobStatus updates the last run status of a monitoring job
func (p *PersistenceBackend) UpdateMonitoringJobStatus(ctx context.Context, jobID string, status string, alertsCreated int) error {
	query := `
		UPDATE monitoring_jobs
		SET last_run_at = ?, last_run_status = ?, last_run_alerts = ?, updated_at = ?
		WHERE id = ?
	`
	now := time.Now()
	_, err := p.db.ExecContext(ctx, query, now, status, alertsCreated, now, jobID)
	if err != nil {
		return fmt.Errorf("failed to update monitoring job status: %w", err)
	}
	return nil
}

// RecordMonitoringRun creates a record of a monitoring job execution
func (p *PersistenceBackend) RecordMonitoringRun(ctx context.Context, run *MonitoringJobRun) error {
	query := `
		INSERT INTO monitoring_job_runs (
			job_id, started_at, completed_at, status, metrics_checked,
			alerts_created, error_message
		) VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	result, err := p.db.ExecContext(ctx, query,
		run.JobID, run.StartedAt, run.CompletedAt, run.Status,
		run.MetricsChecked, run.AlertsCreated, run.ErrorMessage,
	)
	if err != nil {
		return fmt.Errorf("failed to record monitoring run: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert ID: %w", err)
	}
	run.ID = int(id)

	return nil
}

// GetMonitoringJobRuns retrieves execution history for a monitoring job
func (p *PersistenceBackend) GetMonitoringJobRuns(ctx context.Context, jobID string, limit int) ([]*MonitoringJobRun, error) {
	query := `
		SELECT id, job_id, started_at, completed_at, status, metrics_checked,
			alerts_created, error_message
		FROM monitoring_job_runs
		WHERE job_id = ?
		ORDER BY started_at DESC
	`
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := p.db.QueryContext(ctx, query, jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to get monitoring job runs: %w", err)
	}
	defer rows.Close()

	var runs []*MonitoringJobRun
	for rows.Next() {
		run := &MonitoringJobRun{}
		var completedAt sql.NullTime
		var errorMessage sql.NullString

		err := rows.Scan(
			&run.ID, &run.JobID, &run.StartedAt, &completedAt, &run.Status,
			&run.MetricsChecked, &run.AlertsCreated, &errorMessage,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan monitoring job run: %w", err)
		}

		if completedAt.Valid {
			run.CompletedAt = &completedAt.Time
		}
		if errorMessage.Valid {
			run.ErrorMessage = errorMessage.String
		}

		runs = append(runs, run)
	}

	return runs, rows.Err()
}

// SaveSchedulerJob saves a scheduler job to the database
func (p *PersistenceBackend) SaveSchedulerJob(ctx context.Context, id, name, jobType, pipeline, monitoringJobID, cronExpr string, enabled bool, nextRun, lastRun *time.Time) error {
	query := `
		INSERT INTO scheduler_jobs (id, name, job_type, pipeline, monitoring_job_id, cron_expr, is_enabled, next_run, last_run, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			job_type = excluded.job_type,
			pipeline = excluded.pipeline,
			monitoring_job_id = excluded.monitoring_job_id,
			cron_expr = excluded.cron_expr,
			is_enabled = excluded.is_enabled,
			next_run = excluded.next_run,
			last_run = excluded.last_run,
			updated_at = CURRENT_TIMESTAMP
	`

	var nextRunVal, lastRunVal interface{}
	if nextRun != nil {
		nextRunVal = *nextRun
	}
	if lastRun != nil {
		lastRunVal = *lastRun
	}

	_, err := p.db.ExecContext(ctx, query, id, name, jobType, pipeline, monitoringJobID, cronExpr, enabled, nextRunVal, lastRunVal)
	return err
}

// GetAllSchedulerJobs retrieves all scheduler jobs from the database
func (p *PersistenceBackend) GetAllSchedulerJobs(ctx context.Context) ([]SchedulerJobRecord, error) {
	query := `
		SELECT id, name, job_type, pipeline, monitoring_job_id, cron_expr, is_enabled, next_run, last_run, created_at, updated_at
		FROM scheduler_jobs
		ORDER BY created_at DESC
	`

	rows, err := p.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query scheduler jobs: %w", err)
	}
	defer rows.Close()

	var jobs []SchedulerJobRecord
	for rows.Next() {
		job := SchedulerJobRecord{}
		var pipeline, monitoringJobID sql.NullString
		var nextRun, lastRun sql.NullTime

		err := rows.Scan(
			&job.ID, &job.Name, &job.JobType, &pipeline, &monitoringJobID,
			&job.CronExpr, &job.Enabled, &nextRun, &lastRun,
			&job.CreatedAt, &job.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan scheduler job: %w", err)
		}

		if pipeline.Valid {
			job.Pipeline = pipeline.String
		}
		if monitoringJobID.Valid {
			job.MonitoringJobID = monitoringJobID.String
		}
		if nextRun.Valid {
			job.NextRun = &nextRun.Time
		}
		if lastRun.Valid {
			job.LastRun = &lastRun.Time
		}

		jobs = append(jobs, job)
	}

	return jobs, rows.Err()
}

// DeleteSchedulerJob deletes a scheduler job from the database
func (p *PersistenceBackend) DeleteSchedulerJob(ctx context.Context, id string) error {
	query := `DELETE FROM scheduler_jobs WHERE id = ?`
	_, err := p.db.ExecContext(ctx, query, id)
	return err
}

// UpdateSchedulerJobStatus updates the enabled status of a scheduler job
func (p *PersistenceBackend) UpdateSchedulerJobStatus(ctx context.Context, id string, enabled bool) error {
	query := `UPDATE scheduler_jobs SET is_enabled = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := p.db.ExecContext(ctx, query, enabled, id)
	return err
}

// SchedulerJobRecord represents a scheduler job record in the database
type SchedulerJobRecord struct {
	ID              string     `json:"id"`
	Name            string     `json:"name"`
	JobType         string     `json:"job_type"`
	Pipeline        string     `json:"pipeline,omitempty"`
	MonitoringJobID string     `json:"monitoring_job_id,omitempty"`
	CronExpr        string     `json:"cron_expr"`
	Enabled         bool       `json:"enabled"`
	NextRun         *time.Time `json:"next_run,omitempty"`
	LastRun         *time.Time `json:"last_run,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// ==================== API KEYS ====================

// StoredAPIKey represents an API key in the database
type StoredAPIKey struct {
	ID          string     `json:"id"`
	Provider    string     `json:"provider"`
	Name        string     `json:"name"`
	KeyValue    string     `json:"-"` // Never returned - encrypted
	EndpointURL string     `json:"endpoint_url,omitempty"`
	IsActive    bool       `json:"is_active"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
	Metadata    string     `json:"metadata,omitempty"`
}

// CreateAPIKey inserts a new API key into the database
func (p *PersistenceBackend) CreateAPIKey(ctx context.Context, key *StoredAPIKey) error {
	query := `
		INSERT INTO api_keys (id, provider, name, key_value, endpoint_url, is_active, created_at, updated_at, metadata)
		VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, ?)
	`
	_, err := p.db.ExecContext(ctx, query,
		key.ID, key.Provider, key.Name, key.KeyValue, key.EndpointURL, key.IsActive, key.Metadata,
	)
	return err
}

// GetAPIKey retrieves an API key by ID
func (p *PersistenceBackend) GetAPIKey(ctx context.Context, id string) (*StoredAPIKey, error) {
	query := `
		SELECT id, provider, name, key_value, endpoint_url, is_active, created_at, updated_at, last_used_at, metadata
		FROM api_keys
		WHERE id = ?
	`
	key := &StoredAPIKey{}
	var lastUsedAt sql.NullTime
	var metadata sql.NullString
	err := p.db.QueryRowContext(ctx, query, id).Scan(
		&key.ID, &key.Provider, &key.Name, &key.KeyValue, &key.EndpointURL,
		&key.IsActive, &key.CreatedAt, &key.UpdatedAt, &lastUsedAt, &metadata,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("API key not found: %s", id)
	}
	if err != nil {
		return nil, err
	}
	if lastUsedAt.Valid {
		key.LastUsedAt = &lastUsedAt.Time
	}
	if metadata.Valid {
		key.Metadata = metadata.String
	}
	return key, nil
}

// ListAPIKeys returns all API keys
func (p *PersistenceBackend) ListAPIKeys(ctx context.Context) ([]*StoredAPIKey, error) {
	query := `
		SELECT id, provider, name, key_value, endpoint_url, is_active, created_at, updated_at, last_used_at, metadata
		FROM api_keys
		ORDER BY created_at DESC
	`
	rows, err := p.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []*StoredAPIKey
	for rows.Next() {
		key := &StoredAPIKey{}
		var lastUsedAt sql.NullTime
		var metadata sql.NullString
		if err := rows.Scan(
			&key.ID, &key.Provider, &key.Name, &key.KeyValue, &key.EndpointURL,
			&key.IsActive, &key.CreatedAt, &key.UpdatedAt, &lastUsedAt, &metadata,
		); err != nil {
			return nil, err
		}
		if lastUsedAt.Valid {
			key.LastUsedAt = &lastUsedAt.Time
		}
		if metadata.Valid {
			key.Metadata = metadata.String
		}
		keys = append(keys, key)
	}
	return keys, rows.Err()
}

// UpdateAPIKey updates an API key's active status
func (p *PersistenceBackend) UpdateAPIKey(ctx context.Context, id string, isActive bool) error {
	query := `UPDATE api_keys SET is_active = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := p.db.ExecContext(ctx, query, isActive, id)
	return err
}

// UpdateAPIKeyLastUsed updates the last_used_at timestamp
func (p *PersistenceBackend) UpdateAPIKeyLastUsed(ctx context.Context, id string) error {
	query := `UPDATE api_keys SET last_used_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := p.db.ExecContext(ctx, query, id)
	return err
}

// DeleteAPIKey deletes an API key
func (p *PersistenceBackend) DeleteAPIKey(ctx context.Context, id string) error {
	query := `DELETE FROM api_keys WHERE id = ?`
	_, err := p.db.ExecContext(ctx, query, id)
	return err
}

// ==================== DATA MANAGEMENT ====================

// ClearAllPipelines deletes all pipeline files and data
func (p *PersistenceBackend) ClearAllPipelines(ctx context.Context) error {
	query := `DELETE FROM pipelines`
	_, err := p.db.ExecContext(ctx, query)
	return err
}

// ClearAllJobs deletes all scheduler jobs
func (p *PersistenceBackend) ClearAllJobs(ctx context.Context) error {
	query := `DELETE FROM scheduler_jobs`
	_, err := p.db.ExecContext(ctx, query)
	return err
}

// ClearAllDigitalTwins deletes all digital twins
func (p *PersistenceBackend) ClearAllDigitalTwins(ctx context.Context) error {
	query := `DELETE FROM digital_twins`
	_, err := p.db.ExecContext(ctx, query)
	return err
}

// ClearAllOntologies deletes all ontologies
func (p *PersistenceBackend) ClearAllOntologies(ctx context.Context) error {
	query := `DELETE FROM ontologies`
	_, err := p.db.ExecContext(ctx, query)
	return err
}

// ClearAllJobsExecutionHistory deletes all job execution history
func (p *PersistenceBackend) ClearAllJobsExecutionHistory(ctx context.Context) error {
	query := `DELETE FROM job_executions`
	_, err := p.db.ExecContext(ctx, query)
	return err
}
