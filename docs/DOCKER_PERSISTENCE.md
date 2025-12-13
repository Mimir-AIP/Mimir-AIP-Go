# Docker Persistence Guide

This guide explains how persistence works in the Mimir AIP unified Docker container and how to manage data across container restarts.

## Quick Start

### Using Docker Compose (Recommended)

The unified Docker compose configuration already includes persistence:

```bash
# Start the container
docker-compose -f docker-compose.unified.yml up -d

# Data is automatically persisted to named volumes
# Jobs, pipelines, and vector embeddings survive container restarts
```

### Manual Docker Run

```bash
docker run -d \
  -p 8080:8080 \
  -v mimir_data:/app/data \
  -v mimir_logs:/app/logs \
  -v mimir_config:/app/config \
  -v mimir_plugins:/app/plugins \
  -v mimir_pipelines:/app/pipelines \
  -e MIMIR_DATABASE_PATH=/app/data/mimir.db \
  -e MIMIR_VECTOR_DB_PATH=/app/data/chromem.db \
  --name mimir-aip-unified \
  mimir-aip:unified
```

## Persisted Data

### What is Stored

1. **SQLite Database** (`/app/data/mimir.db`):
   - Scheduled jobs and configurations
   - Pipeline definitions
   - Job execution history
   - System metadata

2. **Vector Database** (`/app/data/chromem.db`):
   - Document embeddings
   - Semantic search indices
   - Knowledge base vectors

3. **Log Files** (`/app/logs/`):
   - Application logs
   - Next.js logs
   - Error traces

4. **Configuration** (`/app/config/`):
   - Runtime configuration overrides

5. **Plugins** (`/app/plugins/`):
   - Custom plugin binaries

6. **Pipelines** (`/app/pipelines/`):
   - YAML pipeline definitions

## Volume Management

### List Volumes

```bash
docker volume ls | grep mimir
```

Output:
```
local     mimir_data
local     mimir_logs
local     mimir_config
local     mimir_plugins
local     mimir_pipelines
```

### Inspect Volume

```bash
docker volume inspect mimir_data
```

### Volume Locations

On Linux, Docker volumes are typically stored in:
```
/var/lib/docker/volumes/mimir_data/_data
```

## Backup and Restore

### Backup All Data

```bash
# Create backup directory
mkdir -p ./backups/$(date +%Y%m%d)

# Backup data volume
docker run --rm \
  -v mimir_data:/data:ro \
  -v $(pwd)/backups/$(date +%Y%m%d):/backup \
  alpine tar czf /backup/mimir-data.tar.gz -C /data .

# Backup logs
docker run --rm \
  -v mimir_logs:/logs:ro \
  -v $(pwd)/backups/$(date +%Y%m%d):/backup \
  alpine tar czf /backup/mimir-logs.tar.gz -C /logs .

# Backup config
docker run --rm \
  -v mimir_config:/config:ro \
  -v $(pwd)/backups/$(date +%Y%m%d):/backup \
  alpine tar czf /backup/mimir-config.tar.gz -C /config .
```

### Backup Script

Create `backup.sh`:

```bash
#!/bin/bash
set -e

BACKUP_DIR="./backups/$(date +%Y%m%d-%H%M%S)"
mkdir -p "$BACKUP_DIR"

echo "Creating backup in $BACKUP_DIR..."

# Backup each volume
for volume in mimir_data mimir_logs mimir_config mimir_plugins mimir_pipelines; do
    echo "Backing up $volume..."
    docker run --rm \
        -v "${volume}:/volume:ro" \
        -v "${BACKUP_DIR}:/backup" \
        alpine tar czf "/backup/${volume}.tar.gz" -C /volume .
done

echo "Backup completed: $BACKUP_DIR"
echo "Total size: $(du -sh $BACKUP_DIR | cut -f1)"
```

Make it executable:
```bash
chmod +x backup.sh
./backup.sh
```

### Restore from Backup

```bash
# Stop the container
docker-compose -f docker-compose.unified.yml down

# Restore data volume
docker run --rm \
  -v mimir_data:/data \
  -v $(pwd)/backups/20231215:/backup \
  alpine tar xzf /backup/mimir-data.tar.gz -C /data

# Restore logs
docker run --rm \
  -v mimir_logs:/logs \
  -v $(pwd)/backups/20231215:/backup \
  alpine tar xzf /backup/mimir-logs.tar.gz -C /logs

# Restore config
docker run --rm \
  -v mimir_config:/config \
  -v $(pwd)/backups/20231215:/backup \
  alpine tar xzf /backup/mimir-config.tar.gz -C /config

# Restart container
docker-compose -f docker-compose.unified.yml up -d
```

### Automated Backup with Cron

Add to crontab (`crontab -e`):

```cron
# Backup Mimir data daily at 2 AM
0 2 * * * /path/to/mimir/backup.sh

# Cleanup old backups (keep 7 days)
0 3 * * * find /path/to/mimir/backups -mtime +7 -type d -exec rm -rf {} +
```

## Database Management

### Direct SQLite Access

```bash
# Access running container
docker exec -it mimir-aip-unified /bin/sh

# Open database
sqlite3 /app/data/mimir.db

# List tables
.tables

# Query jobs
SELECT * FROM scheduled_jobs;

# Exit
.quit
```

### Export Database

```bash
# Export as SQL
docker exec mimir-aip-unified \
  sqlite3 /app/data/mimir.db .dump > mimir-export.sql

# Export as CSV
docker exec mimir-aip-unified \
  sqlite3 /app/data/mimir.db \
  -header -csv "SELECT * FROM scheduled_jobs;" > jobs.csv
```

### Database Maintenance

```bash
# VACUUM (reclaim space)
docker exec mimir-aip-unified \
  sqlite3 /app/data/mimir.db "VACUUM;"

# Analyze (optimize queries)
docker exec mimir-aip-unified \
  sqlite3 /app/data/mimir.db "ANALYZE;"

# Check integrity
docker exec mimir-aip-unified \
  sqlite3 /app/data/mimir.db "PRAGMA integrity_check;"
```

## Data Migration

### Moving to a New Host

```bash
# On old host: Create backup
./backup.sh

# Copy backup to new host
scp -r backups/20231215 newhost:/path/to/mimir/backups/

# On new host: Start container
docker-compose -f docker-compose.unified.yml up -d

# Stop container
docker-compose -f docker-compose.unified.yml down

# Restore backup
# (use restore commands from above)

# Start container
docker-compose -f docker-compose.unified.yml up -d
```

### Migrating to PostgreSQL (Future)

For production deployments with multiple instances:

1. Export SQLite data
2. Create PostgreSQL database
3. Convert schema to PostgreSQL
4. Import data
5. Update configuration:

```yaml
persistence:
  backend: "postgres"
  connection_string: "postgres://user:pass@db-host:5432/mimir"
```

## Volume Cleanup

### Remove Unused Volumes

```bash
# Remove all unused volumes
docker volume prune

# Remove specific volume (WARNING: DATA LOSS)
docker volume rm mimir_data
```

### Reset All Data

```bash
# Stop container
docker-compose -f docker-compose.unified.yml down

# Remove volumes
docker volume rm mimir_data mimir_logs mimir_config mimir_plugins mimir_pipelines

# Restart (creates fresh volumes)
docker-compose -f docker-compose.unified.yml up -d
```

## Monitoring Disk Usage

### Check Volume Sizes

```bash
#!/bin/bash
for volume in mimir_data mimir_logs mimir_config mimir_plugins mimir_pipelines; do
    size=$(docker system df -v | grep "$volume" | awk '{print $3}')
    echo "$volume: $size"
done
```

### Monitor Database Growth

```bash
# Check database size
docker exec mimir-aip-unified du -h /app/data/mimir.db

# Check vector database size
docker exec mimir-aip-unified du -h /app/data/chromem.db

# Check log directory size
docker exec mimir-aip-unified du -sh /app/logs
```

### Set Up Alerts

```bash
#!/bin/bash
# alert.sh - Send alert if data exceeds threshold

THRESHOLD_GB=10

SIZE=$(docker exec mimir-aip-unified du -sm /app/data | awk '{print $1}')
SIZE_GB=$((SIZE / 1024))

if [ $SIZE_GB -gt $THRESHOLD_GB ]; then
    echo "WARNING: Data directory is ${SIZE_GB}GB (threshold: ${THRESHOLD_GB}GB)"
    # Send notification (email, Slack, etc.)
fi
```

## Performance Optimization

### WAL Mode (Enabled by Default)

The SQLite database uses Write-Ahead Logging for better concurrency:

```sql
PRAGMA journal_mode=WAL;
```

### Connection Pooling

Connection pool settings (already configured):
- Max open connections: 10
- Max idle connections: 5
- Connection max lifetime: 1 hour

### Retention Policy

Configure execution history retention in `config.yaml`:

```yaml
persistence:
  retention_days: 30  # Keep 30 days of execution history
```

## Troubleshooting

### Database Locked

**Symptom**: "database is locked" errors

**Solution**:
1. Check for long-running transactions
2. Ensure WAL mode is enabled
3. Increase busy timeout (already set to 5000ms)

### Out of Disk Space

**Symptom**: "disk I/O error"

**Solutions**:
1. Enable automatic cleanup
2. Reduce retention period
3. Run VACUUM to reclaim space
4. Increase disk allocation

### Corruption

**Symptom**: Database integrity check fails

**Solution**:
1. Stop container
2. Restore from backup
3. If no backup, try `.recover` command:

```bash
sqlite3 mimir.db ".recover" | sqlite3 mimir-recovered.db
```

### Permission Issues

**Symptom**: Cannot write to database

**Solution**:
```bash
# Fix permissions inside container
docker exec -u root mimir-aip-unified \
  chown -R nonroot:nonroot /app/data
```

## Health Checks

### API Health Endpoint

Check persistence status:

```bash
curl http://localhost:8080/health
```

Response:
```json
{
  "status": "healthy",
  "time": "2023-12-15T10:30:00Z",
  "persistence": {
    "status": "healthy",
    "backend": "sqlite",
    "path": "./data/mimir.db"
  }
}
```

### Database Health Check

```bash
docker exec mimir-aip-unified \
  sqlite3 /app/data/mimir.db "SELECT 1;"
```

## Security Best Practices

### File Permissions

Ensure proper permissions:
```bash
docker exec mimir-aip-unified ls -la /app/data
# Should be owned by nonroot:nonroot
```

### Backup Encryption

Encrypt backups containing sensitive data:

```bash
# Encrypt
gpg --symmetric --cipher-algo AES256 mimir-data.tar.gz

# Decrypt
gpg --decrypt mimir-data.tar.gz.gpg > mimir-data.tar.gz
```

### Access Control

Restrict volume access:
```bash
# Only allow root and docker group
sudo chmod 700 /var/lib/docker/volumes/mimir_data
```

## References

- [Docker Volumes Documentation](https://docs.docker.com/storage/volumes/)
- [SQLite Documentation](https://www.sqlite.org/docs.html)
- [Mimir Persistence Architecture](./PERSISTENCE.md)
