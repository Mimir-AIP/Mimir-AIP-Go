// Add to pipelines/Storage/persistence.go around line 100 (in initSchema)

	-- Scheduler jobs table (for crash recovery)
	CREATE TABLE IF NOT EXISTS scheduler_jobs (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		job_type TEXT NOT NULL,  -- "pipeline" or "monitoring"
		pipeline_id TEXT,
		monitoring_job_id TEXT,
		cron_expr TEXT NOT NULL,
		enabled BOOLEAN NOT NULL DEFAULT 1,
		next_run TIMESTAMP,
		last_run TIMESTAMP,
		last_status TEXT,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (monitoring_job_id) REFERENCES monitoring_jobs(id) ON DELETE CASCADE
	);

	-- Scheduler execution log (for audit trail)
	CREATE TABLE IF NOT EXISTS scheduler_executions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		job_id TEXT NOT NULL,
		started_at TIMESTAMP NOT NULL,
		completed_at TIMESTAMP,
		status TEXT NOT NULL,  -- "running", "success", "failed"
		error_message TEXT,
		FOREIGN KEY (job_id) REFERENCES scheduler_jobs(id) ON DELETE CASCADE
	);

	-- Create indexes for performance
	CREATE INDEX IF NOT EXISTS idx_scheduler_jobs_enabled ON scheduler_jobs(enabled);
	CREATE INDEX IF NOT EXISTS idx_scheduler_jobs_next_run ON scheduler_jobs(next_run);
	CREATE INDEX IF NOT EXISTS idx_scheduler_executions_job_id ON scheduler_executions(job_id);
	CREATE INDEX IF NOT EXISTS idx_monitoring_jobs_enabled ON monitoring_jobs(is_enabled);
