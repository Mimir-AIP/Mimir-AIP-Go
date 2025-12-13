# Persistence Architecture for Mimir AIP

## Overview

The Mimir AIP platform now includes a comprehensive persistence layer designed to store jobs, pipelines, execution history, and vector database data. This persistence layer ensures data durability across container restarts and provides a foundation for future distributed architectures.

## Architecture

### Components

The persistence architecture consists of three main components:

1. **Relational Database (SQLite/PostgreSQL/MySQL)**: Stores structured data including:
   - Scheduled jobs and their configurations
   - Pipeline definitions
   - Job execution history and metrics
   
2. **Vector Database (Chromem)**: Stores vector embeddings for:
   - Document embeddings
   - Semantic search indices
   - Knowledge base vectors

3. **File Storage**: Stores:
   - Configuration files
   - Log files
   - Plugin binaries
   - Temporary data

### Persistence Backend Interface

The system uses an abstraction layer (`PersistenceBackend`) that allows different storage backends to be used interchangeably:

```go
type PersistenceBackend interface {
    // Job persistence
    SaveJob(ctx context.Context, job *ScheduledJob) error
    GetJob(ctx context.Context, id string) (*ScheduledJob, error)
    ListJobs(ctx context.Context) ([]*ScheduledJob, error)
    DeleteJob(ctx context.Context, id string) error
    UpdateJob(ctx context.Context, job *ScheduledJob) error

    // Pipeline persistence
    SavePipeline(ctx context.Context, pipeline *PipelineConfig) error
    GetPipeline(ctx context.Context, name string) (*PipelineConfig, error)
    ListPipelines(ctx context.Context) ([]*PipelineConfig, error)
    DeletePipeline(ctx context.Context, name string) error
    UpdatePipeline(ctx context.Context, pipeline *PipelineConfig) error

    // Job execution history
    SaveExecution(ctx context.Context, execution *JobExecutionRecord) error
    GetExecution(ctx context.Context, id string) (*JobExecutionRecord, error)
    ListExecutions(ctx context.Context, jobID string, limit int) ([]*JobExecutionRecord, error)
    DeleteOldExecutions(ctx context.Context, olderThan time.Time) error

    // Health and lifecycle
    Health(ctx context.Context) error
    Close() error
}
```

## Configuration

### Configuration File (config.yaml)

```yaml
persistence:
  enabled: true
  backend: "sqlite"                      # sqlite, postgres, mysql
  database_path: "./data/mimir.db"       # For SQLite
  connection_string: ""                  # For PostgreSQL/MySQL
  vector_db_path: "./data/chromem.db"    # Chromem vector DB path
  enable_vector_db: true
  backup_enabled: false
  backup_path: "./backups"
  retention_days: 30                     # Execution history retention
```

### Environment Variables

Override configuration using environment variables:

```bash
MIMIR_DATABASE_PATH=/app/data/mimir.db
MIMIR_VECTOR_DB_PATH=/app/data/chromem.db
```

## SQLite Backend

### Database Schema

#### scheduled_jobs Table
```sql
CREATE TABLE scheduled_jobs (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    pipeline TEXT NOT NULL,
    cron_expr TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT 1,
    next_run TIMESTAMP,
    last_run TIMESTAMP,
    last_result TEXT,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);
```

#### pipelines Table
```sql
CREATE TABLE pipelines (
    name TEXT PRIMARY KEY,
    description TEXT,
    config TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);
```

#### job_executions Table
```sql
CREATE TABLE job_executions (
    id TEXT PRIMARY KEY,
    job_id TEXT NOT NULL,
    pipeline TEXT NOT NULL,
    start_time TIMESTAMP NOT NULL,
    end_time TIMESTAMP,
    duration INTEGER,
    status TEXT NOT NULL,
    error TEXT,
    context TEXT,
    steps TEXT,
    triggered_by TEXT NOT NULL,
    FOREIGN KEY (job_id) REFERENCES scheduled_jobs(id) ON DELETE CASCADE
);
```

### Features

- **WAL Mode**: Write-Ahead Logging for improved concurrency
- **Connection Pooling**: Optimized connection management
- **Automatic Schema Migration**: Schema created on first run
- **Indexes**: Optimized queries with proper indexing
- **ACID Transactions**: Full transaction support

## Docker Integration

### Volume Configuration

The unified Docker container uses named volumes for persistence:

```yaml
volumes:
  mimir_data:      # SQLite DB, Vector DB, persistent data
  mimir_logs:      # Application logs
  mimir_config:    # Configuration files
  mimir_plugins:   # Plugin binaries
  mimir_pipelines: # Pipeline definitions
```

### Volume Mounts

```yaml
volumes:
  - mimir_data:/app/data
  - mimir_logs:/app/logs
  - mimir_config:/app/config
  - mimir_plugins:/app/plugins
  - mimir_pipelines:/app/pipelines
```

### Environment Variables in Docker

```yaml
environment:
  - MIMIR_DATABASE_PATH=/app/data/mimir.db
  - MIMIR_VECTOR_DB_PATH=/app/data/chromem.db
```

## Usage

### Starting the Server with Persistence

When the server starts, it automatically:

1. Loads configuration from `config.yaml`
2. Initializes the persistence backend (SQLite by default)
3. Creates database schema if it doesn't exist
4. Loads existing scheduled jobs from the database
5. Starts the scheduler with persisted jobs

### Job Lifecycle with Persistence

```
Create Job → Save to DB → Schedule
   ↓
Execute Job → Save Execution → Update Job Status
   ↓
Update Job → Update in DB
   ↓
Delete Job → Remove from DB
```

### Data Flow

```
API Request
   ↓
Server Handler
   ↓
Scheduler/Monitor
   ↓
Persistence Layer
   ↓
SQLite Database
```

## Backup and Recovery

### Manual Backup

```bash
# Backup SQLite database
cp /app/data/mimir.db /backups/mimir-$(date +%Y%m%d).db

# Backup vector database
cp /app/data/chromem.db /backups/chromem-$(date +%Y%m%d).db
```

### Restore from Backup

```bash
# Stop the server
docker-compose -f docker-compose.unified.yml down

# Restore database
cp /backups/mimir-20231215.db /app/data/mimir.db
cp /backups/chromem-20231215.db /app/data/chromem.db

# Restart server
docker-compose -f docker-compose.unified.yml up -d
```

### Docker Volume Backup

```bash
# Create backup of all volumes
docker run --rm \
  -v mimir_data:/data \
  -v $(pwd)/backups:/backup \
  alpine tar czf /backup/mimir-data-backup.tar.gz -C /data .

# Restore volume from backup
docker run --rm \
  -v mimir_data:/data \
  -v $(pwd)/backups:/backup \
  alpine tar xzf /backup/mimir-data-backup.tar.gz -C /data
```

## Future Scalability

### Distributed Architecture Support

The persistence layer is designed to support future distributed deployments:

#### 1. External Database Support

Switch to PostgreSQL or MySQL for multi-instance deployments:

```yaml
persistence:
  backend: "postgres"
  connection_string: "postgres://user:pass@db-host:5432/mimir"
```

#### 2. Distributed Job Queue

Future implementation will support:
- Job queue for worker distribution
- Worker registration and health checks
- Load balancing across workers
- Job priority and scheduling

#### 3. Shared Storage

For distributed setups:
- Centralized vector database (e.g., Qdrant, Weaviate)
- Shared file storage (e.g., S3, NFS)
- Distributed cache (e.g., Redis)

### Architecture Diagram (Future)

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   API Server 1  │     │   API Server 2  │     │   API Server N  │
└────────┬────────┘     └────────┬────────┘     └────────┬────────┘
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 │
                    ┌────────────▼────────────┐
                    │  PostgreSQL / MySQL DB  │
                    │  (Jobs, Pipelines)      │
                    └─────────────────────────┘
                                 │
         ┌───────────────────────┼───────────────────────┐
         │                       │                       │
┌────────▼────────┐     ┌────────▼────────┐     ┌────────▼────────┐
│   Worker 1      │     │   Worker 2      │     │   Worker N      │
│  (CPU-bound)    │     │  (I/O-bound)    │     │  (GPU-bound)    │
└─────────────────┘     └─────────────────┘     └─────────────────┘
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 │
                    ┌────────────▼────────────┐
                    │   Vector Database       │
                    │   (Qdrant/Weaviate)     │
                    └─────────────────────────┘
```

## Performance Considerations

### SQLite Optimizations

- **WAL Mode**: Enables concurrent reads during writes
- **Busy Timeout**: Configurable timeout for lock contention
- **Connection Pooling**: Reuses connections efficiently
- **Prepared Statements**: Improves query performance

### Retention Policy

Automatic cleanup of old execution records:

```go
// Cleanup executions older than retention period
retentionDate := time.Now().AddDate(0, 0, -config.Persistence.RetentionDays)
persistence.DeleteOldExecutions(ctx, retentionDate)
```

### Indexing Strategy

Indexes are created for common query patterns:
- Job lookup by ID
- Job filtering by enabled status
- Execution lookup by job_id
- Execution sorting by start_time
- Execution filtering by status

## Migration Path

### From In-Memory to SQLite

Current deployments automatically migrate when persistence is enabled:

1. Enable persistence in config.yaml
2. Restart server
3. Existing jobs are lost (one-time migration)
4. Create jobs again - they will be persisted

### From SQLite to PostgreSQL

For production deployments:

1. Export data from SQLite
2. Create PostgreSQL schema
3. Import data to PostgreSQL
4. Update configuration
5. Restart servers

Example migration script structure:

```bash
#!/bin/bash
# Export from SQLite
sqlite3 mimir.db ".dump" > mimir-export.sql

# Convert to PostgreSQL format (requires manual adjustments)
# Import to PostgreSQL
psql -h db-host -U user -d mimir < mimir-converted.sql
```

## Monitoring and Maintenance

### Health Checks

The persistence layer includes health checks:

```go
if err := persistence.Health(ctx); err != nil {
    log.Printf("Persistence health check failed: %v", err)
}
```

### Database Maintenance

Periodic maintenance tasks:

```bash
# SQLite VACUUM (reclaim space)
sqlite3 /app/data/mimir.db "VACUUM;"

# Analyze for query optimization
sqlite3 /app/data/mimir.db "ANALYZE;"
```

### Monitoring Metrics

Key metrics to monitor:
- Database file size
- Query latency
- Connection pool utilization
- Failed persistence operations
- Execution history growth rate

## Security Considerations

### File Permissions

SQLite database files should have restricted permissions:

```bash
chmod 600 /app/data/mimir.db
chown nonroot:nonroot /app/data/mimir.db
```

### Backup Encryption

For sensitive data, encrypt backups:

```bash
# Encrypt backup
gpg --symmetric --cipher-algo AES256 mimir.db

# Decrypt backup
gpg --decrypt mimir.db.gpg > mimir.db
```

### Connection Security

For PostgreSQL/MySQL:
- Use SSL/TLS connections
- Strong password policies
- Network isolation
- Principle of least privilege

## Troubleshooting

### Common Issues

#### Database Locked Error

```
Error: database is locked
```

**Solution**: Increase busy timeout or use WAL mode (already enabled)

#### Schema Migration Failed

```
Error: failed to initialize schema
```

**Solution**: Check file permissions, ensure directory exists

#### Out of Disk Space

```
Error: disk I/O error
```

**Solution**: 
- Enable automatic cleanup
- Reduce retention period
- Increase disk space

### Debug Mode

Enable detailed logging:

```yaml
logging:
  level: "debug"
```

## References

- [SQLite Documentation](https://www.sqlite.org/docs.html)
- [Chromem-go Documentation](https://github.com/philippgille/chromem-go)
- [Docker Volumes](https://docs.docker.com/storage/volumes/)
- [Go database/sql Package](https://pkg.go.dev/database/sql)
