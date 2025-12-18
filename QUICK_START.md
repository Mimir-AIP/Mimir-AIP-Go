# Quick Start: Production-Ready Data Ingestion

## 🚀 Run the System

```bash
# Backend
./mimir-aip-server

# Frontend (separate terminal)
cd mimir-aip-frontend && bun run dev

# Open browser
http://localhost:3000
```

## 📤 Upload Data for Auto-Training

1. Navigate to Ontologies page
2. Click on any ontology
3. Click "Train" tab
4. Select data type (CSV/Excel/JSON)
5. Upload your file
6. Click "Start Auto-Training"
7. View results!

## ✨ What Happens Automatically

### Time-Series Data
- ✅ Detects date/time columns
- ✅ Creates monitoring jobs
- ✅ Sets up threshold rules
- ✅ Schedules periodic checks
- ✅ **Jobs survive server restarts**

### ML Training
- ✅ Detects regression targets (numeric)
- ✅ Detects classification targets (categorical)
- ✅ Trains multiple models automatically
- ✅ Returns performance metrics (R², accuracy)
- ✅ Displays results in UI

## 🔄 Test Crash Recovery

```bash
# 1. Upload data and create monitoring jobs
# 2. Check scheduler has jobs:
curl http://localhost:8080/api/v1/scheduler/jobs

# 3. Kill server (Ctrl+C)

# 4. Restart server:
./mimir-aip-server

# 5. Check logs - should see:
#    "🔄 Recovering N scheduled jobs from database..."
#    "✅ Recovered job: ..."

# 6. Verify jobs still running:
curl http://localhost:8080/api/v1/scheduler/jobs
```

## 📝 Test with Sample Data

### CSV Example
```csv
archer_name,date,score,arrows_shot,wind_speed,target_distance
Alice,2024-01-15,285,30,5.2,70
Bob,2024-01-15,312,30,5.2,70
Alice,2024-01-22,298,30,3.1,70
```

### JSON Example
```json
{
  "data": [
    {"name": "Alice", "score": 285, "date": "2024-01-15"},
    {"name": "Bob", "score": 312, "date": "2024-01-15"}
  ]
}
```

## 🎯 Expected Results

### Training Success
```json
{
  "models_created": 2,
  "monitoring_jobs_created": 1,
  "rules_created": 3,
  "trained_models": [
    {
      "model_id": "auto_test_score_1234567890",
      "target_property": "score",
      "model_type": "regression",
      "r2_score": 0.85,
      "rmse": 12.3
    }
  ]
}
```

### Monitoring Jobs
- Created automatically for time-series data
- Runs on schedule (e.g., every 6 hours)
- Checks thresholds and trends
- Creates alerts when needed
- **Persists across restarts**

## 📊 Check Monitoring

```bash
# List all monitoring jobs
curl http://localhost:8080/api/v1/monitoring/jobs

# List all alerts
curl http://localhost:8080/api/v1/monitoring/alerts

# List all rules
curl http://localhost:8080/api/v1/monitoring/rules
```

## 🐛 Troubleshooting

### Upload Fails
- Check file format (CSV/Excel/JSON)
- Ensure file has headers (CSV/Excel)
- Verify JSON structure (array or {data: array})
- Check file size (<10MB recommended)

### No Models Trained
- Check if data has numeric columns (regression)
- Check if data has categorical columns (classification)
- Verify sufficient rows (minimum ~10-20)
- Look for null values (<30% per column)

### Jobs Not Recovering
- Check if storage backend is connected
- Verify scheduler_jobs table exists
- Check server logs for errors
- Ensure jobs were persisted before restart

## 🔍 Logs to Watch

### Backend Startup
```
✅ Database initialized
✅ Scheduler started with 0 jobs
🔄 Recovering N scheduled jobs from database...
✅ Recovered job: auto_monitor_test-ontology_date
✅ Successfully recovered N/N scheduled jobs
```

### Training
```
🎯 Training from data: 10 rows x 6 columns
📊 Time-series data detected, setting up monitoring...
🎯 Detecting ML targets from dataset...
   📈 Regression target: score (confidence: 0.80)
📦 Preparing training data for target: score
🤖 Training regression model for: score
✅ Model trained in 234ms (ID: auto_test_score_1234567890)
✅ Data-based training completed in 1.2s (models: 1, failed: 0)
```

### Crash Recovery
```
🔄 Recovering 3 scheduled jobs from database...
✅ Recovered job: auto_monitor_archery_date (monitoring)
✅ Recovered job: pipeline_daily_sync (pipeline)
✅ Recovered job: auto_monitor_ngo_timestamp (monitoring)
✅ Successfully recovered 3/3 scheduled jobs
```

## 📚 More Documentation

- Full summary: `agent-docs/current/PRODUCTION_READY_SUMMARY.md`
- Sprint 1 details: `agent-docs/current/SPRINT1_COMPLETION.md`
- Plugin guide: `docs/PLUGIN_DEVELOPMENT_GUIDE.md`
- Example plugin: `examples/custom_erp_plugin.go`

## ✅ Quick Validation

```bash
# 1. Backend builds
go build -o mimir-aip-server .

# 2. Frontend builds
cd mimir-aip-frontend && npm run build

# 3. Backend runs
./mimir-aip-server

# 4. API responds
curl http://localhost:8080/api/v1/health

# 5. Upload works (with actual file)
curl -X POST http://localhost:8080/api/v1/auto-train-with-data \
  -H "Content-Type: application/json" \
  -d @test_payload.json
```

## 🎉 Success Indicators

✅ Frontend builds without errors  
✅ Backend builds without errors  
✅ Server starts and logs "Scheduler started"  
✅ File upload returns 200 OK  
✅ Training results include models_created > 0  
✅ Jobs appear in scheduler after training  
✅ Jobs still present after server restart  

**Status: PRODUCTION READY** 🚀
