# Mimir AIP Pipeline Execution Flow

This document explains how to execute pipeline steps and validate their results.

## Pipeline Step Execution Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        PIPELINE EXECUTION FLOW                               │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  1. CREATE PIPELINE                                                         │
│     ┌──────────────────────────────────────────────────────────────────┐    │
│     │  POST /api/v1/agent/tools/execute                                │    │
│     │  {                                                              │    │
│     │    "tool_name": "create_pipeline",                              │    │
│     │    "input": {                                                   │    │
│     │      "name": "Data Processing Pipeline",                        │    │
│     │      "steps": [                                                 │    │
│     │        {"name": "Read CSV", "plugin": "Input.csv", ...},        │    │
│     │        {"name": "Process Data", "plugin": "Data_Processing", ...},│   │
│     │        {"name": "Save JSON", "plugin": "Output.json", ...}      │    │
│     │      ]                                                          │    │
│     │    }                                                            │    │
│     │  }                                                              │    │
│     └──────────────────────────────────────────────────────────────────┘    │
│                                   ↓                                          │
│  2. EXECUTE PIPELINE                                                        │
│     ┌──────────────────────────────────────────────────────────────────┐    │
│     │  POST /api/v1/agent/tools/execute                                │    │
│     │  {                                                              │    │
│     │    "tool_name": "execute_pipeline",                             │    │
│     │    "input": {                                                   │    │
│     │      "pipeline_id": "pl_abc12345"                               │    │
│     │    }                                                            │    │
│     │  }                                                              │    │
│     └──────────────────────────────────────────────────────────────────┘    │
│                                   ↓                                          │
│  3. VALIDATE RESULTS                                                        │
│     ┌──────────────────────────────────────────────────────────────────┐    │
│     │  Response:                                                       │    │
│     │  {                                                              │    │
│     │    "success": true,                                             │    │
│     │    "result": {                                                  │    │
│     │      "pipeline_id": "pl_abc12345",                              │    │
│     │      "status": "completed",                                     │    │
│     │      "execution_id": "exec_xyz789",                             │    │
│     │      "output": {                                                │    │
│     │        "data": [...],  ← Step 1 output                          │    │
│     │        "processed_data": [...], ← Step 2 output                 │    │
│     │        "result": {...}      ← Step 3 output                     │    │
│     │      }                                                          │    │
│     │    }                                                            │    │
│     │  }                                                              │    │
│     └──────────────────────────────────────────────────────────────────┘    │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Available Pipeline Steps

### Input Plugins (Data Ingestion)
| Plugin | Description | Config |
|--------|-------------|--------|
| `Input.csv` | Read CSV files | `file_path`, `has_headers`, `delimiter` |
| `Input.json` | Read JSON files | `file_path` |
| `Input.api` | HTTP API requests | `url`, `method`, `headers` |
| `Input.xml` | Read XML files | `file_path` |
| `Input.excel` | Read Excel files | `file_path`, `sheet_name` |
| `Input.mysql` | Query MySQL database | `connection_string`, `query` |

### Output Plugins (Data Export)
| Plugin | Description | Config |
|--------|-------------|--------|
| `Output.json` | Write JSON files | `output`, `file_path` |
| `Output.excel` | Write Excel files | `output`, `file_path` |
| `Output.pdf` | Generate PDF reports | `output`, `file_path` |

### AI Plugins (LLM Processing)
| Plugin | Description | Config |
|--------|-------------|--------|
| `AI.openai` | OpenAI GPT models | `model`, `messages`, `temperature` |
| `AI.anthropic` | Claude models | `model`, `messages`, `temperature` |
| `AI.ollama` | Local Ollama models | `model`, `prompt` |
| `AI.local` | Bundled local LLM | `model`, `prompt` |

### Ontology Plugins (Knowledge Graph)
| Plugin | Description | Config |
|--------|-------------|--------|
| `Ontology.extract` | Extract ontology from data | `data_source`, `data_type` |
| `Ontology.query` | Query knowledge graph | `query` |
| `Ontology.management` | Manage ontologies | `operation` |

### ML Plugins (Machine Learning)
| Plugin | Description | Config |
|--------|-------------|--------|
| `ML.classifier` | Train classifier | `training_data`, `target_column` |
| `ML.regression` | Train regression model | `training_data`, `target_column` |
| `ML.evaluator` | Evaluate model performance | `model`, `test_data` |

### Digital Twin Plugins
| Plugin | Description | Config |
|--------|-------------|--------|
| `DigitalTwin.simulation` | Run simulations | `twin_id`, `scenario` |
| `DigitalTwin.whatif` | What-if analysis | `twin_id`, `parameters` |

## Example: Complete Pipeline Execution

### Step 1: Create a Data Processing Pipeline

```bash
curl -X POST http://localhost:8080/api/v1/agent/tools/execute \
  -H "Content-Type: application/json" \
  -d '{
    "tool_name": "create_pipeline",
    "input": {
      "name": "Customer Data Processor",
      "description": "Process customer data from CSV, enrich with AI, save to JSON",
      "steps": [
        {
          "name": "Read Customer CSV",
          "plugin": "Input.csv",
          "config": {
            "file_path": "/data/customers.csv",
            "has_headers": true
          },
          "output": "raw_data"
        },
        {
          "name": "Clean and Transform",
          "plugin": "Data_Processing.transform",
          "config": {
            "operation": "normalize",
            "columns": ["name", "email", "purchase_amount"]
          },
          "input": "raw_data",
          "output": "cleaned_data"
        },
        {
          "name": "Generate Summaries",
          "plugin": "AI.openai",
          "config": {
            "model": "gpt-4o-mini",
            "messages": [
              {"role": "system", "content": "Summarize this customer data"},
              {"role": "user", "content": "{{raw_data}}"}
            ]
          },
          "input": "raw_data",
          "output": "summary"
        },
        {
          "name": "Save Results",
          "plugin": "Output.json",
          "config": {
            "output": "result",
            "file_path": "/data/processed/customers_output.json"
          },
          "input": ["cleaned_data", "summary"]
        }
      ]
    }
  }'
```

### Step 2: Execute the Pipeline

```bash
curl -X POST http://localhost:8080/api/v1/agent/tools/execute \
  -H "Content-Type: application/json" \
  -d '{
    "tool_name": "execute_pipeline",
    "input": {
      "pipeline_id": "pl_abc12345"
    }
  }'
```

### Step 3: Validate Execution Results

```bash
# Check pipeline status
curl -X POST http://localhost:8080/api/v1/agent/tools/execute \
  -H "Content-Type: application/json" \
  -d '{
    "tool_name": "get_pipeline_status",
    "input": {
      "pipeline_id": "pl_abc12345"
    }
  }'

# Expected response:
# {
#   "success": true,
#   "result": {
#     "pipeline_id": "pl_abc12345",
#     "name": "Customer Data Processor",
#     "status": "completed",
#     "last_run": "2026-01-06T15:30:00Z",
#     "steps_executed": 4,
#     "duration_ms": 5234,
#     "outputs": {
#       "raw_data": {"count": 1500, "columns": [...]},
#       "cleaned_data": {"count": 1500, "valid": 1498},
#       "summary": "Customer data shows...",
#       "result": {"file": "/data/processed/customers_output.json"}
#     }
#   }
# }
```

## Validation Strategies

### 1. Check Pipeline Status
```bash
# Use get_pipeline_status to verify completion
POST /api/v1/agent/tools/execute
{
  "tool_name": "get_pipeline_status",
  "input": {"pipeline_id": "pl_abc12345"}
}
```

### 2. Check Output Files Exist
```bash
# Verify output was written
ls -la /data/processed/customers_output.json

# Check file contents
cat /data/processed/customers_output.json | jq '. | keys'
```

### 3. Check Data Quality
```bash
# Verify data was processed correctly
curl -X POST http://localhost:8080/api/v1/agent/tools/execute \
  -H "Content-Type: application/json" \
  -d '{
    "tool_name": "detect_anomalies",
    "input": {
      "twin_id": "twin_xxx",
      "time_range": "24h"
    }
  }'
```

### 4. Schedule for Regular Validation
```bash
# Schedule pipeline to run daily
curl -X POST http://localhost:8080/api/v1/agent/tools/execute \
  -H "Content-Type: application/json" \
  -d '{
    "tool_name": "schedule_pipeline",
    "input": {
      "pipeline_id": "pl_abc12345",
      "cron": "0 6 * * *",
      "name": "Daily Customer Processing"
    }
  }'
```

## Error Handling

If a step fails, the pipeline execution will:
1. Stop at the failing step
2. Return error details including:
   - Step name that failed
   - Error message
   - Partial results (steps before failure)

```json
{
  "success": false,
  "error": "step 2 (Clean and Transform) failed: validation error on column 'email'",
  "result": {
    "pipeline_id": "pl_abc12345",
    "status": "failed",
    "failed_step": 2,
    "outputs": {
      "raw_data": {"count": 1500}
    }
  }
}
```

## Monitoring Pipeline Execution

### View Active Pipelines
```bash
# List all pipelines with their status
curl -X POST http://localhost:8080/api/v1/agent/tools/execute \
  -H "Content-Type: application/json" \
  -d '{
    "tool_name": "list_pipelines",
    "input": {}
  }'
```

### View Scheduled Jobs
```bash
# Check scheduled executions
# (This would be a new agent tool to add)
```

### View Execution History
```bash
# Check past executions
# (This would be a new agent tool to add)
```
