# Mimir-AIP: Crash Recovery & Data Ingestion Architecture

## Critical Issues Identified

### 1. CSV-Only Auto-Train Limitation
### 2. No Crash Recovery for Monitoring Jobs
### 3. No Scheduler State Persistence

---

## SOLUTION 1: Universal Data Ingestion

### Problem
Currently, auto-train only accepts CSV via direct upload. This blocks:
- Excel archery scores
- Web-scraped articles (JSON/Markdown)
- IoT sensor data (JSON streams)
- Mobile app data from NGO field workers
- Pipeline-generated datasets

### Architecture Change

#### 1.1 Unified Data Representation

```go
// pipelines/ML/unified_dataset.go

package ml

type UnifiedDataset struct {
    Source           string                    // "csv", "excel", "json", "pipeline", "stream"
    Columns          []ColumnMetadata          
    Rows             []map[string]interface{}  
    RowCount         int
    Metadata         map[string]interface{}    
    TimeSeriesConfig *TimeSeriesDetectionResult 
    CreatedAt        time.Time
}

type ColumnMetadata struct {
    Name           string
    DataType       string  // "string", "numeric", "datetime", "boolean"
    IsNumeric      bool
    IsTimeSeries   bool
    HasNulls       bool
    SampleValues   []interface{}
}

type TimeSeriesDetectionResult struct {
    DateColumn     string
    MetricColumns  []string
    Frequency      string  // "daily", "weekly", "monthly", "irregular"
    StartDate      time.Time
    EndDate        time.Time
    HasGaps        bool
}
```

#### 1.2 Data Source Abstraction

```go
// handlers_auto_ml.go refactor

type AutoTrainRequest struct {
    OntologyID string                 `json:"ontology_id"`
    DataSource AutoTrainDataSource    `json:"data_source"`
    Options    AutoTrainOptions       `json:"options"`
}

type AutoTrainDataSource struct {
    Type       string                 `json:"type"`  // "direct", "pipeline", "ingestion", "storage"
    
    // For direct upload
    Format     string                 `json:"format,omitempty"`  // "csv", "excel", "json"
    Data       interface{}            `json:"data,omitempty"`
    
    // For pipeline-based
    PipelineID string                 `json:"pipeline_id,omitempty"`
    OutputKey  string                 `json:"output_key,omitempty"`
    
    // For storage query
    StorageID  string                 `json:"storage_id,omitempty"`
    Query      map[string]interface{} `json:"query,omitempty"`
}

type AutoTrainOptions struct {
    ForceTimeSeriesDetection bool     `json:"force_timeseries,omitempty"`
    DateColumn               string   `json:"date_column,omitempty"`
    MetricColumns            []string `json:"metric_columns,omitempty"`
    EnableMonitoring         bool     `json:"enable_monitoring"`  // Default: true
}
```

#### 1.3 Adapter Pattern for Data Sources

```go
// pipelines/ML/data_adapters.go

type DataAdapter interface {
    ExtractDataset(source AutoTrainDataSource) (*UnifiedDataset, error)
}

type CSVAdapter struct{}
func (a *CSVAdapter) ExtractDataset(source AutoTrainDataSource) (*UnifiedDataset, error) {
    // Existing CSV logic
}

type ExcelAdapter struct{}
func (a *ExcelAdapter) ExtractDataset(source AutoTrainDataSource) (*UnifiedDataset, error) {
    // Use Input.excel plugin
}

type PipelineAdapter struct {
    storage *storage.PersistenceBackend
}
func (a *PipelineAdapter) ExtractDataset(source AutoTrainDataSource) (*UnifiedDataset, error) {
    // Fetch from pipeline output stored in Storage plugin
}

type JSONAdapter struct{}
func (a *JSONAdapter) ExtractDataset(source AutoTrainDataSource) (*UnifiedDataset, error) {
    // Parse JSON array of objects
}

// Factory
func GetDataAdapter(sourceType string, deps ...interface{}) (DataAdapter, error) {
    switch sourceType {
    case "csv", "direct":
        return &CSVAdapter{}, nil
    case "excel":
        return &ExcelAdapter{}, nil
    case "pipeline":
        return &PipelineAdapter{storage: deps[0].(*storage.PersistenceBackend)}, nil
    case "json":
        return &JSONAdapter{}, nil
    default:
        return nil, fmt.Errorf("unsupported data source type: %s", sourceType)
    }
}
```

### Example Usage

**Archery Scores (Excel):**
```bash
curl -X POST http://localhost:8080/api/v1/ontology/archery-ont/auto-train \
  -H "Content-Type: application/json" \
  -d '{
    "data_source": {
      "type": "excel",
      "format": "excel",
      "data": {
        "file_path": "/uploads/archery_scores_2024.xlsx",
        "sheet_name": "Weekly Scores"
      }
    },
    "options": {
      "date_column": "Week",
      "metric_columns": ["Score", "X_Count", "Indoor_Score"],
      "enable_monitoring": true
    }
  }'
```

**Web-Scraped Articles (Pipeline):**
```bash
curl -X POST http://localhost:8080/api/v1/ontology/news-ont/auto-train \
  -H "Content-Type: application/json" \
  -d '{
    "data_source": {
      "type": "pipeline",
      "pipeline_id": "daily-news-scraper",
      "output_key": "scraped_articles"
    },
    "options": {
      "enable_monitoring": true
    }
  }'
```

**NGO Mobile Data (JSON from Storage):**
```bash
curl -X POST http://localhost:8080/api/v1/ontology/field-data-ont/auto-train \
  -H "Content-Type: application/json" \
  -d '{
    "data_source": {
      "type": "storage",
      "storage_id": "field-reports-db",
      "query": {
        "collection": "daily_reports",
        "filter": {"region": "conflict_zone_A"}
      }
    }
  }'
```

---

## SOLUTION 2: Crash Recovery System

### Problem
When Mimir server crashes or restarts:
1. ❌ All monitoring jobs disappear (not loaded from DB)
2. ❌ Scheduler forgets what was scheduled
3. ❌ No audit trail of what was running
4. ❌ NGO in war-torn area with unstable power = data loss

### Architecture Change

#### 2.1 Persistent Scheduler State

```go
// pipelines/Storage/persistence.go

// Add to PersistenceBackend

func (p *PersistenceBackend) SaveSchedulerJob(job *SchedulerJobRecord) error {
    query := `
        INSERT OR REPLACE INTO scheduler_jobs 
        (id, name, job_type, pipeline_id, monitoring_job_id, cron_expr, enabled, next_run, last_run, last_status, updated_at)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
    `
    _, err := p.db.Exec(query, 
        job.ID, job.Name, job.JobType, job.PipelineID, job.MonitoringJobID, 
        job.CronExpr, job.Enabled, job.NextRun, job.LastRun, job.LastStatus)
    return err
}

func (p *PersistenceBackend) LoadAllSchedulerJobs() ([]*SchedulerJobRecord, error) {
    query := `SELECT id, name, job_type, pipeline_id, monitoring_job_id, cron_expr, enabled, next_run, last_run, last_status FROM scheduler_jobs WHERE enabled = 1`
    rows, err := p.db.Query(query)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var jobs []*SchedulerJobRecord
    for rows.Next() {
        job := &SchedulerJobRecord{}
        err := rows.Scan(&job.ID, &job.Name, &job.JobType, &job.PipelineID, &job.MonitoringJobID, 
                         &job.CronExpr, &job.Enabled, &job.NextRun, &job.LastRun, &job.LastStatus)
        if err != nil {
            return nil, err
        }
        jobs = append(jobs, job)
    }
    return jobs, nil
}

func (p *PersistenceBackend) RecordSchedulerExecution(jobID string, status string, errMsg string) error {
    if status == "running" {
        query := `INSERT INTO scheduler_executions (job_id, started_at, status) VALUES (?, CURRENT_TIMESTAMP, 'running')`
        _, err := p.db.Exec(query, jobID)
        return err
    } else {
        query := `UPDATE scheduler_executions SET completed_at = CURRENT_TIMESTAMP, status = ?, error_message = ? 
                  WHERE job_id = ? AND status = 'running' ORDER BY started_at DESC LIMIT 1`
        _, err := p.db.Exec(query, status, errMsg, jobID)
        return err
    }
}
```

#### 2.2 Scheduler Recovery on Startup

```go
// utils/scheduler.go

// Add recovery method
func (s *Scheduler) RecoverJobsFromDatabase(storage interface{}) error {
    if storage == nil {
        return fmt.Errorf("storage backend required for recovery")
    }
    
    persistence, ok := storage.(*persistence.PersistenceBackend)
    if !ok {
        return fmt.Errorf("invalid storage backend type")
    }

    log.Println("🔄 Recovering scheduled jobs from database...")
    
    jobs, err := persistence.LoadAllSchedulerJobs()
    if err != nil {
        return fmt.Errorf("failed to load jobs: %w", err)
    }

    s.jobsMutex.Lock()
    defer s.jobsMutex.Unlock()

    recovered := 0
    for _, jobRecord := range jobs {
        if !jobRecord.Enabled {
            continue
        }

        job := &ScheduledJob{
            ID:              jobRecord.ID,
            Name:            jobRecord.Name,
            JobType:         jobRecord.JobType,
            Pipeline:        jobRecord.PipelineID,
            MonitoringJobID: jobRecord.MonitoringJobID,
            CronExpr:        jobRecord.CronExpr,
            Enabled:         jobRecord.Enabled,
            NextRun:         jobRecord.NextRun,
            LastRun:         jobRecord.LastRun,
        }

        // Recompute next run time if needed
        if job.NextRun == nil || job.NextRun.Before(time.Now()) {
            nextRun, err := computeNextRun(job.CronExpr, time.Now())
            if err != nil {
                log.Printf("⚠️  Failed to compute next run for job %s: %v", job.ID, err)
                continue
            }
            job.NextRun = &nextRun
        }

        s.jobs[job.ID] = job
        recovered++
    }

    log.Printf("✅ Recovered %d scheduled jobs from database", recovered)
    return nil
}

// Update AddJob to persist
func (s *Scheduler) AddJob(id, name, pipeline, cronExpr string) error {
    // ... existing validation ...

    job := &ScheduledJob{...}
    s.jobs[id] = job

    // PERSIST TO DATABASE
    if s.storage != nil {
        if err := s.persistJob(job); err != nil {
            log.Printf("⚠️  Failed to persist job to database: %v", err)
        }
    }

    return nil
}

func (s *Scheduler) persistJob(job *ScheduledJob) error {
    persistence, ok := s.storage.(*persistence.PersistenceBackend)
    if !ok {
        return fmt.Errorf("invalid storage backend")
    }

    record := &SchedulerJobRecord{
        ID:              job.ID,
        Name:            job.Name,
        JobType:         job.JobType,
        PipelineID:      job.Pipeline,
        MonitoringJobID: job.MonitoringJobID,
        CronExpr:        job.CronExpr,
        Enabled:         job.Enabled,
        NextRun:         job.NextRun,
        LastRun:         job.LastRun,
    }

    return persistence.SaveSchedulerJob(record)
}
```

#### 2.3 Server Startup Sequence

```go
// server.go - NewServer()

func NewServer() *Server {
    // ... existing initialization ...

    // Initialize monitoring executor
    if persistence != nil {
        monitoringExecutor := ml.NewMonitoringExecutor(persistence)
        s.scheduler.SetStorage(persistence)
        s.scheduler.SetMonitoringExecutor(monitoringExecutor)
        log.Println("Monitoring executor initialized")
    }

    // Start the scheduler
    if err := s.scheduler.Start(); err != nil {
        log.Fatalf("Failed to start scheduler: %v", err)
    }

    // 🔄 RECOVER JOBS AFTER SCHEDULER STARTS
    if persistence != nil {
        if err := s.scheduler.RecoverJobsFromDatabase(persistence); err != nil {
            log.Printf("⚠️  Failed to recover scheduler jobs: %v", err)
        }
        
        // Also recover monitoring jobs that were enabled
        if err := s.recoverMonitoringJobs(persistence); err != nil {
            log.Printf("⚠️  Failed to recover monitoring jobs: %v", err)
        }
    }

    return s
}

func (s *Server) recoverMonitoringJobs(persistence *storage.PersistenceBackend) error {
    log.Println("🔄 Recovering enabled monitoring jobs...")
    
    // Query all enabled monitoring jobs
    jobs, err := persistence.ListMonitoringJobs("", true)  // enabled_only=true
    if err != nil {
        return err
    }

    recovered := 0
    for _, job := range jobs {
        // Add to scheduler
        err := s.scheduler.AddMonitoringJob(
            "monitor-"+job.ID,
            job.Name,
            job.ID,
            job.CronExpr,
        )
        if err != nil {
            log.Printf("⚠️  Failed to recover monitoring job %s: %v", job.ID, err)
            continue
        }
        recovered++
    }

    log.Printf("✅ Recovered %d monitoring jobs", recovered)
    return nil
}
```

---

## SOLUTION 3: NGO Resilience Features

### Problem
NGO in war-torn area has:
- Intermittent electricity (server goes offline randomly)
- Limited connectivity (can't rely on cloud backups)
- Critical data that must not be lost

### Architecture Enhancements

#### 3.1 Write-Ahead Logging (Already Enabled! ✅)

```go
// pipelines/Storage/persistence.go:29-30
if _, err := db.Exec("PRAGMA journal_mode = WAL"); err != nil {
    return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
}
```

**What this means:**
- ✅ Writes are durable immediately
- ✅ Crash in middle of write? Database recovers automatically
- ✅ Power loss? WAL replays on restart

#### 3.2 Graceful Shutdown Hook

```go
// server.go:230-260 (already implemented! ✅)

func (s *Server) Shutdown(ctx context.Context) error {
    // 1. Stop scheduler first (prevents new jobs)
    // 2. Wait for running jobs to complete (30s timeout)
    // 3. Close database connections
}
```

#### 3.3 Add Periodic Checkpointing

```go
// server.go - Add background checkpoint task

func (s *Server) startPeriodicCheckpoint() {
    if s.persistence == nil {
        return
    }

    ticker := time.NewTicker(5 * time.Minute)
    go func() {
        for {
            select {
            case <-ticker.C:
                log.Println("📸 Running WAL checkpoint...")
                if _, err := s.persistence.GetDB().Exec("PRAGMA wal_checkpoint(PASSIVE)"); err != nil {
                    log.Printf("⚠️  Checkpoint failed: %v", err)
                } else {
                    log.Println("✅ Checkpoint complete")
                }
            case <-s.ctx.Done():
                ticker.Stop()
                return
            }
        }
    }()
}

// Call in NewServer()
s.startPeriodicCheckpoint()
```

#### 3.4 Database Backup on Shutdown

```go
// server.go - Add to Shutdown()

func (s *Server) Shutdown(ctx context.Context) error {
    // ... existing shutdown logic ...

    // Before closing persistence, create backup
    if s.persistence != nil {
        backupPath := fmt.Sprintf("./data/backups/mimir_%s.db", time.Now().Format("20060102_150405"))
        if err := s.createBackup(backupPath); err != nil {
            log.Printf("⚠️  Backup failed: %v", err)
        } else {
            log.Printf("✅ Database backup created: %s", backupPath)
        }
    }

    return nil
}

func (s *Server) createBackup(backupPath string) error {
    os.MkdirAll(filepath.Dir(backupPath), 0755)
    
    _, err := s.persistence.GetDB().Exec("VACUUM INTO ?", backupPath)
    return err
}
```

---

## Implementation Priority

### Phase 1: Critical (Crash Recovery) - 1-2 days
1. ✅ Add `scheduler_jobs` table to schema
2. ✅ Implement `RecoverJobsFromDatabase()`
3. ✅ Update `AddJob()` / `AddMonitoringJob()` to persist
4. ✅ Add recovery call in `server.go:NewServer()`
5. ✅ Test: Start server → Create jobs → Kill server → Restart → Verify jobs recovered

### Phase 2: Data Ingestion (Universal Auto-Train) - 2-3 days
1. Create `UnifiedDataset` struct
2. Implement adapter pattern for CSV/Excel/JSON/Pipeline
3. Refactor `handleAutoTrain()` to use adapters
4. Update frontend to support multiple data sources
5. Test with Excel, JSON, pipeline outputs

### Phase 3: NGO Resilience - 1 day
1. Add periodic checkpointing
2. Add backup-on-shutdown
3. Test intermittent restart scenarios

---

## Testing Scenarios

### Crash Recovery Test
```bash
# 1. Start server
./mimir-aip-server --server 8080

# 2. Create monitoring job via auto-train
curl -X POST http://localhost:8080/api/v1/ontology/test/auto-train ...

# 3. Verify job is scheduled
curl http://localhost:8080/api/v1/monitoring/jobs

# 4. Kill server (simulate crash)
kill -9 <pid>

# 5. Restart server
./mimir-aip-server --server 8080

# 6. Verify job is recovered
curl http://localhost:8080/api/v1/monitoring/jobs
# Expected: Job still exists, next_run recalculated
```

### Power Loss Simulation (NGO Scenario)
```bash
# Simulate power loss every 30 minutes
while true; do
    sleep 1800
    pkill -9 mimir-aip-server
    sleep 300  # 5 min "power outage"
    ./mimir-aip-server --server 8080 &
done

# Verify: Monitoring continues, no data loss
```

---

## File Changes Required

### New Files
1. `pipelines/ML/unified_dataset.go` - Universal data representation
2. `pipelines/ML/data_adapters.go` - Adapter implementations
3. `docs/DATA_INGESTION_GUIDE.md` - Usage documentation

### Modified Files
1. `pipelines/Storage/persistence.go` - Add scheduler tables, recovery methods
2. `utils/scheduler.go` - Add persistence, recovery logic
3. `handlers_auto_ml.go` - Refactor to use adapters
4. `server.go` - Add recovery calls, checkpoint task
5. `mimir-aip-frontend/src/app/ontologies/[id]/page.tsx` - Update auto-train UI

---

## Summary

**Current State:**
- ❌ CSV-only auto-train
- ❌ No crash recovery
- ❌ Monitoring jobs lost on restart

**After Implementation:**
- ✅ Any data source (CSV, Excel, JSON, pipelines)
- ✅ Full crash recovery (scheduler + monitoring)
- ✅ NGO-resilient (WAL, checkpoints, backups)
- ✅ Audit trail (scheduler execution log)

**Estimated Total Effort:** 4-6 days
**Priority:** **CRITICAL** (Phase 1 must be done ASAP)
