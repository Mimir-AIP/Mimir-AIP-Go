# Monitoring Job System Design

## Problem Statement

We need continuous monitoring that:
1. Runs time-series analysis on ontology data
2. Detects anomalies, trends, threshold violations
3. Creates alerts when conditions are met
4. Integrates with existing scheduler + pipeline system

## Current Architecture

### Existing Components
- ✅ **Scheduler** (`utils/scheduler.go`): Cron-based job execution
- ✅ **Pipeline System**: YAML-defined data processing workflows
- ✅ **Time-Series Storage** (`storage.TimeSeriesDataPoint`): Database storage
- ✅ **Time-Series Analysis** (`ml.TimeSeries`, `ml.TimeSeriesAnalyzer`): Analytics engine
- ✅ **Monitoring Rules** (`storage.MonitoringRule`): User-defined thresholds/conditions
- ✅ **Alerts** (`storage.CreateAlert`): Alert storage

### The Gap
- ❌ No monitoring job type (scheduler only runs pipelines)
- ❌ No bridge between storage layer and analysis layer
- ❌ No automated rule evaluation

## Proposed Solution

### Architecture: Monitoring as Special Job Type

```
┌─────────────────┐
│   Scheduler     │
│  (every 30s)    │
└────────┬────────┘
         │
    ┌────┴────┐
    │         │
┌───▼───┐ ┌──▼───────┐
│Pipeline│ │Monitoring│
│  Job   │ │   Job    │
└────────┘ └──┬───────┘
              │
    ┌─────────┴──────────┐
    │                    │
┌───▼──────┐      ┌──────▼────┐
│Time-Series│      │ML Prediction│
│ Analysis  │      │  Analysis   │
└───┬───────┘      └──────┬─────┘
    │                     │
    └──────┬──────────────┘
           │
    ┌──────▼─────┐
    │   Alerts   │
    └────────────┘
```

### Implementation Plan

#### Step 1: Extend Scheduler to Support Monitoring Jobs

**Modify `utils/scheduler.go`:**
```go
type ScheduledJob struct {
    // ... existing fields ...
    JobType string `json:"job_type"` // "pipeline" or "monitoring"
    
    // For monitoring jobs
    MonitoringConfig *MonitoringJobConfig `json:"monitoring_config,omitempty"`
}

type MonitoringJobConfig struct {
    OntologyID string   `json:"ontology_id"`
    AnalysisTypes []string `json:"analysis_types"` // ["time_series", "ml_prediction", "anomaly"]
}
```

**Add execution path:**
```go
func (s *Scheduler) executeJob(job *ScheduledJob) {
    switch job.JobType {
    case "pipeline":
        s.executePipeline(job)
    case "monitoring":
        s.executeMonitoring(job)
    }
}

func (s *Scheduler) executeMonitoring(job *ScheduledJob) {
    // Execute monitoring analysis
    executor := ml.NewMonitoringJobExecutor(s.persistence)
    executor.Execute(ctx, job.MonitoringConfig)
}
```

#### Step 2: Create Monitoring Job Executor

**File: `pipelines/ML/monitoring_executor.go`**

Key responsibilities:
1. Fetch active monitoring rules for ontology
2. For each rule:
   - Query time-series data from storage
   - Convert `storage.TimeSeriesDataPoint[]` → `ml.TimeSeries`
   - Run appropriate analysis (trend, anomaly, forecast, threshold)
   - Evaluate rule conditions
   - Create alerts if violated

```go
type MonitoringJobExecutor struct {
    Storage    *storage.PersistenceBackend
    TSAnalyzer *TimeSeriesAnalyzer
}

func (m *MonitoringJobExecutor) Execute(ctx context.Context, config *MonitoringJobConfig) error {
    // 1. Get monitoring rules
    rules, _ := m.Storage.GetMonitoringRules(ctx, "", "")
    
    // 2. Filter by ontology
    for _, rule := range rules {
        if rule.OntologyID == config.OntologyID {
            m.evaluateRule(ctx, rule)
        }
    }
}
```

#### Step 3: Bridge Storage and Analysis Layers

**Problem:** Storage returns `[]TimeSeriesDataPoint`, analysis expects `TimeSeries`

**Solution:** Add converter

```go
func ConvertToTimeSeries(entityID, metricName string, points []storage.TimeSeriesDataPoint) *TimeSeries {
    ts := &TimeSeries{
        EntityID:   entityID,
        MetricName: metricName,
        Points:     make([]TimeSeriesPoint, len(points)),
    }
    
    for i, p := range points {
        ts.Points[i] = TimeSeriesPoint{
            Timestamp: p.Timestamp,
            Value:     p.Value,
        }
    }
    
    return ts
}
```

#### Step 4: Add Monitoring Job API Endpoints

**File: `handlers_monitoring.go`**

```
POST   /api/v1/monitoring/jobs              - Create monitoring job
GET    /api/v1/monitoring/jobs               - List monitoring jobs
GET    /api/v1/monitoring/jobs/{id}          - Get monitoring job
PUT    /api/v1/monitoring/jobs/{id}          - Update monitoring job
DELETE /api/v1/monitoring/jobs/{id}          - Delete monitoring job
POST   /api/v1/monitoring/jobs/{id}/enable   - Enable monitoring job
POST   /api/v1/monitoring/jobs/{id}/disable  - Disable monitoring job

POST   /api/v1/monitoring/rules              - Create monitoring rule
GET    /api/v1/monitoring/rules              - List monitoring rules
DELETE /api/v1/monitoring/rules/{id}         - Delete monitoring rule

GET    /api/v1/monitoring/alerts             - List alerts
POST   /api/v1/monitoring/alerts/{id}/acknowledge - Acknowledge alert
POST   /api/v1/monitoring/alerts/{id}/resolve     - Resolve alert
```

#### Step 5: User Workflow

**Example: Computer Repair Shop**

1. **Set up ingestion pipeline** (if not already exists):
```bash
# User creates inventory_ingestion.yaml
POST /api/v1/scheduler/jobs
{
  "id": "ingest-inventory",
  "name": "Daily Inventory Ingestion",
  "pipeline": "inventory_ingestion.yaml",
  "cron_expr": "0 9 * * *"  # 9 AM daily
}
```

2. **Create monitoring job**:
```bash
POST /api/v1/monitoring/jobs
{
  "id": "monitor-inventory",
  "name": "Inventory Monitoring",
  "ontology_id": "ont_computer_inventory",
  "cron_expr": "*/15 * * * *",  # Every 15 minutes
  "analysis_types": ["time_series", "anomaly"]
}
```

3. **Create monitoring rules**:
```bash
# Alert when stock low
POST /api/v1/monitoring/rules
{
  "ontology_id": "ont_computer_inventory",
  "entity_id": "product_gpu_rtx4090",
  "metric_name": "stock_level",
  "rule_type": "threshold",
  "condition": {
    "operator": "<",
    "value": 5
  },
  "severity": "high"
}

# Alert when prices increasing
POST /api/v1/monitoring/rules
{
  "ontology_id": "ont_computer_inventory",
  "metric_name": "price",
  "rule_type": "trend",
  "condition": {
    "expected": "increasing",
    "min_change_percent": 15
  },
  "severity": "medium"
}
```

4. **View alerts**:
```bash
GET /api/v1/monitoring/alerts?ontology_id=ont_computer_inventory
```

## Key Design Decisions

### 1. Monitoring Jobs vs Pipeline Jobs
- **Separate job types** with different execution paths
- Monitoring jobs don't need YAML pipelines (they're hard-coded analysis)
- Simpler for users: "Create monitoring job" vs "Create pipeline YAML + schedule it"

### 2. Rule-Based vs Hardcoded Analysis
- ✅ **Rule-based** (chosen): Users define what to monitor via API
- ❌ Hardcoded: Code decides what to analyze
- **Why**: Flexibility - users control thresholds, metrics to monitor

### 3. Trigger Types
- **Cron-based** (Phase 1): Run monitoring every N minutes
- **Post-pipeline** (Phase 2): Run monitoring after ingestion completes
- **Webhook-triggered** (Phase 3): External systems trigger monitoring

### 4. Alert Deduplication
- Check if similar alert exists before creating new one
- Don't create "Stock low" alert every 15 minutes
- Create once, user acknowledges/resolves

### 5. Condition Storage Format
- Store as JSON string in database (`condition TEXT`)
- Parse at runtime in executor
- Flexible schema for different rule types

## Implementation Priority

### MVP (Sprint 1 - Task 3):
1. ✅ Time-series storage (already done)
2. ✅ Time-series analysis (already done)
3. ⏳ Converter: `storage.TimeSeriesDataPoint` → `ml.TimeSeries`
4. ⏳ Basic monitoring executor (threshold + trend rules only)
5. ⏳ Scheduler integration (add monitoring job type)
6. ⏳ API endpoints for monitoring jobs

### Phase 2:
- ML prediction monitoring (low-confidence detection)
- Anomaly detection across all metrics
- Post-pipeline triggers
- Alert channels (email, SMS, webhooks)

### Phase 3:
- Dashboard UI for alerts
- Alert history and statistics
- Advanced aggregations (category-level analysis)
- Alert routing rules

## Questions / Decisions Needed

1. **Should monitoring jobs be stored in database or just in-memory?**
   - Proposal: Store in database (add `monitoring_jobs` table)
   - Benefit: Persists across restarts
   - Downside: More complexity

2. **How to handle continuous ingestion?**
   - User has ingestion pipeline already scheduled
   - Monitoring job runs independently on schedule
   - Alternative: Add "post_pipeline" trigger type later

3. **Should we use existing `ScheduledJob` or create `MonitoringJob`?**
   - Proposal: Extend `ScheduledJob` with `JobType` field
   - Benefit: Reuse existing scheduler infrastructure
   - Simpler than two separate systems

## Next Steps

1. Implement converter (`TimeSeriesDataPoint` → `TimeSeries`)
2. Create basic monitoring executor
3. Test with threshold rules
4. Add API endpoints
5. Test end-to-end with sample data
