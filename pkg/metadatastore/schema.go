package metadatastore

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

	CREATE TABLE IF NOT EXISTS ontology_compilations (
		ontology_id TEXT PRIMARY KEY,
		project_id TEXT NOT NULL,
		content_hash TEXT NOT NULL,
		compiled_at TEXT NOT NULL,
		data TEXT NOT NULL,
		FOREIGN KEY (ontology_id) REFERENCES ontologies(id) ON DELETE CASCADE,
		FOREIGN KEY (project_id) REFERENCES projects(id)
	);

	CREATE INDEX IF NOT EXISTS idx_ontology_compilations_project_id ON ontology_compilations(project_id);

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


CREATE TABLE IF NOT EXISTS dt_sync_runs (
	id TEXT PRIMARY KEY,
	twin_id TEXT NOT NULL,
	trigger_type TEXT NOT NULL,
	triggered_by TEXT,
	status TEXT NOT NULL,
	started_at TEXT NOT NULL,
	completed_at TEXT,
	ontology_version TEXT,
	reconciliation_strategy TEXT,
	data TEXT NOT NULL,
	FOREIGN KEY (twin_id) REFERENCES digital_twins(id) ON DELETE CASCADE
	);

CREATE INDEX IF NOT EXISTS idx_dt_sync_runs_twin_id ON dt_sync_runs(twin_id);
	CREATE INDEX IF NOT EXISTS idx_dt_sync_runs_started_at ON dt_sync_runs(started_at);
	CREATE INDEX IF NOT EXISTS idx_dt_sync_runs_status ON dt_sync_runs(status);


CREATE TABLE IF NOT EXISTS dt_relationship_revisions (
	id TEXT PRIMARY KEY,
	twin_id TEXT NOT NULL,
	sync_run_id TEXT,
	source_entity_id TEXT NOT NULL,
	target_entity_id TEXT NOT NULL,
	relationship_type TEXT NOT NULL,
	revision INTEGER NOT NULL,
	change_type TEXT NOT NULL,
	recorded_at TEXT NOT NULL,
	data TEXT NOT NULL,
	FOREIGN KEY (twin_id) REFERENCES digital_twins(id) ON DELETE CASCADE,
	FOREIGN KEY (sync_run_id) REFERENCES dt_sync_runs(id) ON DELETE SET NULL,
	UNIQUE(twin_id, source_entity_id, target_entity_id, relationship_type, revision)
	);

CREATE INDEX IF NOT EXISTS idx_dt_relationship_revisions_twin_id ON dt_relationship_revisions(twin_id);
	CREATE INDEX IF NOT EXISTS idx_dt_relationship_revisions_source_id ON dt_relationship_revisions(source_entity_id);
	CREATE INDEX IF NOT EXISTS idx_dt_relationship_revisions_target_id ON dt_relationship_revisions(target_entity_id);
	CREATE INDEX IF NOT EXISTS idx_dt_relationship_revisions_recorded_at ON dt_relationship_revisions(recorded_at);

CREATE TABLE IF NOT EXISTS dt_snapshots (
	id TEXT PRIMARY KEY,
	twin_id TEXT NOT NULL,
	sync_run_id TEXT NOT NULL,
	snapshot_kind TEXT NOT NULL,
	entity_state BLOB NOT NULL,
	relationship_state BLOB NOT NULL,
	created_at TEXT NOT NULL,
	entity_revision_high_watermark INTEGER NOT NULL DEFAULT 0,
	relationship_revision_high_watermark INTEGER NOT NULL DEFAULT 0,
	data TEXT NOT NULL,
	FOREIGN KEY (twin_id) REFERENCES digital_twins(id) ON DELETE CASCADE,
	FOREIGN KEY (sync_run_id) REFERENCES dt_sync_runs(id) ON DELETE CASCADE
	);

CREATE INDEX IF NOT EXISTS idx_dt_snapshots_twin_id ON dt_snapshots(twin_id);
	CREATE INDEX IF NOT EXISTS idx_dt_snapshots_sync_run_id ON dt_snapshots(sync_run_id);
	CREATE INDEX IF NOT EXISTS idx_dt_snapshots_created_at ON dt_snapshots(created_at);


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

	CREATE TABLE IF NOT EXISTS plugin_artifacts (
		id TEXT PRIMARY KEY,
		plugin_kind TEXT NOT NULL,
		plugin_name TEXT NOT NULL,
		source_repository TEXT NOT NULL,
		source_ref TEXT NOT NULL,
		source_commit TEXT NOT NULL,
		digest TEXT NOT NULL,
		local_path TEXT NOT NULL,
		size_bytes INTEGER NOT NULL,
		go_version TEXT NOT NULL,
		goos TEXT NOT NULL,
		goarch TEXT NOT NULL,
		host_version TEXT NOT NULL,
		symbol_name TEXT NOT NULL,
		status TEXT NOT NULL,
		error_message TEXT NOT NULL DEFAULT '',
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		UNIQUE(plugin_kind, plugin_name, source_commit, host_version, go_version, goos, goarch, symbol_name)
	);

	CREATE INDEX IF NOT EXISTS idx_plugin_artifacts_lookup ON plugin_artifacts(plugin_kind, plugin_name, status);

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
