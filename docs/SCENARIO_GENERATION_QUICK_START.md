# Scenario Generation Quick Start Guide

## Overview
Automatic scenario generation creates realistic simulation scenarios for Digital Twins during data ingestion. This enables immediate testing and validation of your digital twin models.

## Quick Usage

### Step 1: Create a Digital Twin from Data

```bash
# 1. Upload your data file
curl -X POST http://localhost:8080/api/v1/data/upload \
  -F "file=@your_data.csv" \
  -F "plugin_type=Input" \
  -F "plugin_name=csv"

# Response includes upload_id
{
  "upload_id": "upload_1234567890_your_data.csv",
  "filename": "your_data.csv",
  "message": "File uploaded successfully"
}

# 2. Preview and select data
curl -X POST http://localhost:8080/api/v1/data/preview \
  -H "Content-Type: application/json" \
  -d '{
    "upload_id": "upload_1234567890_your_data.csv",
    "plugin_type": "Input",
    "plugin_name": "csv",
    "max_rows": 10
  }'

# 3. Create Digital Twin with automatic scenarios
curl -X POST http://localhost:8080/api/v1/data/select \
  -H "Content-Type: application/json" \
  -d '{
    "upload_id": "upload_1234567890_your_data.csv",
    "selected_columns": ["name", "email", "department"],
    "create_twin": true
  }'
```

### Step 2: Response with Scenario IDs

```json
{
  "digital_twin": {
    "id": "twin_1734476400",
    "name": "your_data Dataset",
    "entity_count": 5,
    "relationship_count": 3,
    "scenario_ids": [
      "scenario_twin_1734476400_baseline",
      "scenario_twin_1734476400_data_quality",
      "scenario_twin_1734476400_capacity"
    ],
    "scenario_count": 3
  },
  "message": "Ontology and Digital Twin generated successfully"
}
```

### Step 3: List Available Scenarios

```bash
curl http://localhost:8080/api/v1/twins/twin_1734476400/scenarios
```

**Response:**
```json
{
  "scenarios": [
    {
      "id": "scenario_twin_1734476400_baseline",
      "name": "Baseline Operations",
      "description": "Normal operating conditions with no disruptions",
      "scenario_type": "baseline",
      "duration": 30,
      "created_at": "2024-12-17T22:00:00Z"
    },
    {
      "id": "scenario_twin_1734476400_data_quality",
      "name": "Data Quality Issues",
      "description": "Simulates data quality problems including missing values",
      "scenario_type": "data_quality_issue",
      "duration": 40,
      "created_at": "2024-12-17T22:00:00Z"
    },
    {
      "id": "scenario_twin_1734476400_capacity",
      "name": "Capacity Stress Test",
      "description": "Tests system behavior under high load conditions",
      "scenario_type": "capacity_test",
      "duration": 50,
      "created_at": "2024-12-17T22:00:00Z"
    }
  ],
  "count": 3
}
```

### Step 4: Run a Scenario

```bash
# Run the baseline scenario
curl -X POST http://localhost:8080/api/v1/twins/twin_1734476400/scenarios/scenario_twin_1734476400_baseline/run \
  -H "Content-Type: application/json" \
  -d '{
    "snapshot_interval": 10,
    "max_steps": 100
  }'
```

**Response:**
```json
{
  "run_id": "run_abc123",
  "status": "completed",
  "metrics": {
    "total_steps": 30,
    "events_processed": 0,
    "entities_affected": 0,
    "average_utilization": 0.50,
    "peak_utilization": 0.60,
    "system_stability": 1.0,
    "impact_summary": "Simulation completed in 30 steps. System remained stable.",
    "recommendations": ["System performed well under simulated conditions"]
  },
  "message": "Simulation completed successfully"
}
```

### Step 5: View Simulation Results

```bash
# Get detailed simulation run results
curl http://localhost:8080/api/v1/simulations/runs/run_abc123

# Get timeline with snapshots
curl http://localhost:8080/api/v1/simulations/runs/run_abc123/timeline

# Analyze impact
curl http://localhost:8080/api/v1/twins/twin_1734476400/simulations/run_abc123/analyze
```

## Generated Scenario Types

### 1. Baseline Operations (30 steps)
- **Type:** `baseline`
- **Events:** 0
- **Purpose:** Establish normal operating baseline
- **Use Case:** Performance comparison reference

### 2. Data Quality Issues (40 steps)
- **Type:** `data_quality_issue`
- **Events:** 4
  - Step 5: Data source unavailable
  - Step 15: Data validation failure
  - Step 25: Data source restored
  - Step 30: Quality constraint added
- **Purpose:** Test resilience to data problems
- **Use Case:** Data pipeline failure testing

### 3. Capacity Stress Test (50 steps)
- **Type:** `capacity_test`
- **Events:** 6
  - Step 5: Initial demand surge (80% increase)
  - Step 15: Secondary demand surge (120% increase)
  - Step 20: Capacity reduction (30% loss)
  - Step 25: External market shift
  - Step 35: Process optimization applied
  - Step 45: Demand normalization
- **Purpose:** Test system under high load
- **Use Case:** Capacity planning, scaling analysis

## Event Propagation

Events automatically propagate through relationships:

```
Entity A (failure) --[relationship]--> Entity B (60% impact, 2 steps delay)
```

- **Data Quality Scenarios:** 60% impact, 2 step delay
- **Capacity Test Scenarios:** 50-80% impact, 1-2 step delay

## Tips

### Best Practices
1. **Always run baseline first** - Establishes performance reference
2. **Compare scenarios** - Use metrics to compare different conditions
3. **Review propagation** - Check how events cascade through relationships
4. **Monitor bottlenecks** - Identify capacity constraints early

### Interpreting Results

#### System Stability Score (0.0 - 1.0)
- `> 0.8` - Minimal impact, system resilient
- `0.6 - 0.8` - Moderate impact, some degradation
- `0.3 - 0.6` - Severe impact, significant issues
- `< 0.3` - Critical, major failures

#### Utilization Thresholds
- `< 60%` - Normal operation
- `60% - 80%` - Elevated load
- `80% - 95%` - High utilization, monitor closely
- `> 95%` - Overutilized, degraded performance

### Common Patterns

#### Progressive Testing
```bash
# 1. Baseline
curl -X POST .../baseline/run

# 2. Data Quality
curl -X POST .../data_quality/run

# 3. Capacity Test
curl -X POST .../capacity/run

# Compare all three
curl http://localhost:8080/api/v1/simulations/compare \
  -d '{"run_ids": ["run1", "run2", "run3"]}'
```

#### Continuous Monitoring
```bash
# Run all scenarios periodically
for scenario in baseline data_quality capacity; do
  curl -X POST .../scenario_twin_${TWIN_ID}_${scenario}/run
done
```

## Troubleshooting

### No Scenarios Generated
**Cause:** Twin has no entities  
**Solution:** Ensure data ingestion created entities

### Events Don't Propagate
**Cause:** No relationships between entities  
**Solution:** Check relationship inference in ontology generation

### All Events Fail
**Cause:** Entity URIs don't match  
**Solution:** Verify entity URIs in twin structure

## Next Steps

1. **Customize Scenarios** - Create your own scenarios via API
2. **Scheduled Simulations** - Set up periodic scenario execution
3. **Alert Integration** - Connect simulation results to monitoring
4. **Scenario Templates** - Define reusable scenario patterns

## API Endpoints Reference

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/api/v1/data/upload` | POST | Upload data file |
| `/api/v1/data/preview` | POST | Preview parsed data |
| `/api/v1/data/select` | POST | Create twin + scenarios |
| `/api/v1/twins/{id}/scenarios` | GET | List scenarios |
| `/api/v1/twins/{id}/scenarios/{sid}/run` | POST | Run scenario |
| `/api/v1/simulations/runs/{rid}` | GET | Get run results |
| `/api/v1/simulations/runs/{rid}/timeline` | GET | Get timeline |
| `/api/v1/twins/{id}/simulations/{rid}/analyze` | POST | Analyze impact |

## Support

For issues or questions:
- Documentation: `SCENARIO_GENERATION_IMPLEMENTATION.md`
- Tests: `scenario_generation_test.go`
- Source: `handlers.go:1784-2110`
