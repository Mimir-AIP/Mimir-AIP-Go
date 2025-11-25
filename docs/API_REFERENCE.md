# Mimir AIP API Reference

This document provides comprehensive API reference for the Mimir AIP platform.

## Table of Contents

1. [REST API Endpoints](#rest-api-endpoints)
2. [Authentication](#authentication)
3. [Pipeline Management](#pipeline-management)
4. [Plugin Management](#plugin-management)
5. [Scheduler Management](#scheduler-management)
6. [Job Monitoring](#job-monitoring)
7. [Performance Monitoring](#performance-monitoring)
8. [Configuration Management](#configuration-management)
9. [Visualization](#visualization)
10. [Plugin API](#plugin-api)
11. [MCP Protocol](#mcp-protocol)
12. [Error Handling](#error-handling)

## REST API Endpoints

### Base URL
```
http://localhost:8080/api/v1
```

### Health Check
```http
GET /
```

**Response:**
```json
{
  "status": "healthy",
  "time": "2024-01-01T10:00:00Z"
}
```

## Authentication

### JWT Authentication
The API uses JWT (JSON Web Tokens) for authentication. Include the token in the Authorization header:
```
Authorization: Bearer <jwt_token>
```

### Login
```http
POST /auth/login
```

**Request Body:**
```json
{
  "username": "admin",
  "password": "password"
}
```

**Response:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": "admin",
  "roles": ["admin"],
  "expires_in": 86400
}
```

### Refresh Token
```http
POST /auth/refresh
```

**Request Body:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

**Response:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_in": 86400
}
```

### Get Current User
```http
GET /auth/me
```

**Response:**
```json
{
  "id": "user_123",
  "username": "admin",
  "roles": ["admin"],
  "active": true
}
```

### List Users (Admin Only)
```http
GET /auth/users
```

**Response:**
```json
{
  "users": [
    {
      "id": "user_123",
      "username": "admin",
      "roles": ["admin"],
      "active": true
    }
  ]
}
```

### Create API Key
```http
POST /auth/apikeys
```

**Request Body:**
```json
{
  "name": "My API Key"
}
```

**Response:**
```json
{
  "key": "sk-1234567890abcdef",
  "name": "My API Key",
  "user_id": "user_123",
  "created": "2024-01-01T10:00:00Z"
}
```

## Pipeline Management

### Execute Pipeline
```http
POST /pipelines/execute
```

Execute a pipeline by name or file path.

**Request Body:**
```json
{
  "pipeline_name": "my_pipeline",
  "pipeline_file": "pipelines/my_pipeline.yaml",
  "context": {
    "input_param": "value"
  }
}
```

**Response:**
```json
{
  "success": true,
  "error": "",
  "context": {
    "result": "processed_data"
  },
  "executed_at": "2024-01-01T10:00:00Z"
}
```

### Pipeline CRUD Operations

#### List Pipelines
```http
GET /pipelines
```

Get a list of all pipelines from configuration.

**Response:**
```json
[
  {
    "name": "data_pipeline",
    "description": "Processes data from API",
    "steps": [
      {
        "name": "fetch_data",
        "plugin": "Input.api",
        "config": {
          "url": "https://api.example.com/data"
        }
      }
    ]
  }
]
```

#### Create Pipeline
```http
POST /pipelines
```

Create a new pipeline in the store.

**Request Body:**
```json
{
  "metadata": {
    "name": "new_pipeline",
    "description": "A new pipeline",
    "version": "1.0.0",
    "author": "user"
  },
  "config": {
    "name": "new_pipeline",
    "steps": [
      {
        "name": "step1",
        "plugin": "Input.api",
        "config": {
          "url": "https://api.example.com"
        }
      }
    ]
  }
}
```

**Response:**
```json
{
  "message": "Pipeline created successfully",
  "pipeline": {
    "id": "pipeline_123",
    "metadata": {
      "name": "new_pipeline",
      "description": "A new pipeline",
      "version": "1.0.0",
      "author": "user",
      "created_at": "2024-01-01T10:00:00Z",
      "updated_at": "2024-01-01T10:00:00Z"
    },
    "config": {
      "name": "new_pipeline",
      "steps": [...]
    }
  }
}
```

#### Get Pipeline
```http
GET /pipelines/{id}
```

Get a specific pipeline by ID.

**Response:**
```json
{
  "id": "pipeline_123",
  "metadata": {
    "name": "my_pipeline",
    "description": "Pipeline description",
    "version": "1.0.0",
    "author": "user",
    "created_at": "2024-01-01T10:00:00Z",
    "updated_at": "2024-01-01T10:00:00Z"
  },
  "config": {
    "name": "my_pipeline",
    "steps": [...]
  }
}
```

#### Update Pipeline
```http
PUT /pipelines/{id}
```

Update an existing pipeline.

**Request Body:**
```json
{
  "metadata": {
    "description": "Updated description"
  },
  "config": {
    "steps": [
      {
        "name": "updated_step",
        "plugin": "Input.api",
        "config": {
          "url": "https://updated-api.com"
        }
      }
    ]
  }
}
```

**Response:**
```json
{
  "message": "Pipeline updated successfully",
  "pipeline": {
    "id": "pipeline_123",
    "metadata": {
      "name": "my_pipeline",
      "description": "Updated description",
      "version": "1.0.1",
      "author": "user",
      "updated_at": "2024-01-01T11:00:00Z"
    },
    "config": {
      "name": "my_pipeline",
      "steps": [...]
    }
  }
}
```

#### Delete Pipeline
```http
DELETE /pipelines/{id}
```

Delete a pipeline.

**Response:**
```json
{
  "message": "Pipeline deleted successfully",
  "id": "pipeline_123"
}
```

#### Clone Pipeline
```http
POST /pipelines/{id}/clone
```

Clone an existing pipeline.

**Request Body:**
```json
{
  "name": "cloned_pipeline"
}
```

**Response:**
```json
{
  "message": "Pipeline cloned successfully",
  "pipeline": {
    "id": "pipeline_456",
    "metadata": {
      "name": "cloned_pipeline",
      "description": "Cloned from original pipeline",
      "version": "1.0.0",
      "author": "user",
      "created_at": "2024-01-01T11:00:00Z"
    },
    "config": {
      "name": "cloned_pipeline",
      "steps": [...]
    }
  }
}
```

#### Validate Pipeline
```http
POST /pipelines/{id}/validate
```

Validate a pipeline configuration.

**Response:**
```json
{
  "valid": true,
  "errors": [],
  "pipeline_id": "pipeline_123"
}
```

#### Get Pipeline History
```http
GET /pipelines/{id}/history
```

Get the version history of a pipeline.

**Response:**
```json
{
  "pipeline_id": "pipeline_123",
  "history": [
    {
      "version": "1.0.0",
      "author": "user",
      "timestamp": "2024-01-01T10:00:00Z",
      "changes": "Initial version"
    },
    {
      "version": "1.0.1",
      "author": "user",
      "timestamp": "2024-01-01T11:00:00Z",
      "changes": "Updated configuration"
    }
  ]
}
```

## Plugin Management

### List Plugins
```http
GET /plugins
```

Get a list of all available plugins.

**Response:**
```json
[
  {
    "type": "Input",
    "name": "api",
    "description": "api plugin"
  },
  {
    "type": "Output",
    "name": "html",
    "description": "html plugin"
  }
]
```

### List Plugins by Type
```http
GET /plugins/{type}
```

Get plugins of a specific type.

**Response:**
```json
[
  {
    "type": "Input",
    "name": "api",
    "description": "api plugin"
  }
]
```

### Get Plugin
```http
GET /plugins/{type}/{name}
```

Get information about a specific plugin.

**Response:**
```json
{
  "type": "Input",
  "name": "api",
  "description": "api plugin"
}
```

## Scheduler Management

### List Jobs
```http
GET /scheduler/jobs
```

Get a list of all scheduled jobs.

**Response:**
```json
[
  {
    "id": "job_123",
    "name": "Daily Sync",
    "pipeline": "data_pipeline.yaml",
    "cron_expr": "0 2 * * *",
    "enabled": true,
    "next_run": "2024-01-02T02:00:00Z",
    "last_run": "2024-01-01T02:00:00Z",
    "last_status": "success",
    "created_at": "2024-01-01T10:00:00Z"
  }
]
```

### Get Job
```http
GET /scheduler/jobs/{id}
```

Get a specific scheduled job.

**Response:**
```json
{
  "id": "job_123",
  "name": "Daily Sync",
  "pipeline": "data_pipeline.yaml",
  "cron_expr": "0 2 * * *",
  "enabled": true,
  "next_run": "2024-01-02T02:00:00Z",
  "last_run": "2024-01-01T02:00:00Z",
  "last_status": "success",
  "created_at": "2024-01-01T10:00:00Z"
}
```

### Create Job
```http
POST /scheduler/jobs
```

Create a new scheduled job.

**Request Body:**
```json
{
  "id": "daily_sync",
  "name": "Daily Data Sync",
  "pipeline": "pipelines/data_pipeline.yaml",
  "cron_expr": "0 2 * * *"
}
```

**Response:**
```json
{
  "message": "Job created successfully",
  "job_id": "daily_sync"
}
```

### Delete Job
```http
DELETE /scheduler/jobs/{id}
```

Delete a scheduled job.

**Response:**
```json
{
  "message": "Job deleted successfully",
  "job_id": "daily_sync"
}
```

### Enable Job
```http
POST /scheduler/jobs/{id}/enable
```

Enable a scheduled job.

**Response:**
```json
{
  "message": "Job enabled successfully",
  "job_id": "daily_sync"
}
```

### Disable Job
```http
POST /scheduler/jobs/{id}/disable
```

Disable a scheduled job.

**Response:**
```json
{
  "message": "Job disabled successfully",
  "job_id": "daily_sync"
}
```

## Agentic Features

### Agent Execute
```http
POST /agent/execute
```

Execute agentic operations (placeholder for LLM integration).

**Response:**
```json
{
  "message": "Agent execution not yet implemented",
  "status": "pending"
}
```

## Job Monitoring

### List Job Executions
```http
GET /jobs
```

Get a list of all job executions.

**Response:**
```json
[
  {
    "id": "exec_123",
    "job_id": "daily_sync",
    "pipeline": "data_pipeline.yaml",
    "status": "completed",
    "start_time": "2024-01-01T02:00:00Z",
    "end_time": "2024-01-01T02:00:05Z",
    "duration": 5.2,
    "success": true,
    "error": "",
    "context": {
      "result": "processed_data"
    }
  }
]
```

### Get Job Execution
```http
GET /jobs/{id}
```

Get a specific job execution.

**Response:**
```json
{
  "id": "exec_123",
  "job_id": "daily_sync",
  "pipeline": "data_pipeline.yaml",
  "status": "completed",
  "start_time": "2024-01-01T02:00:00Z",
  "end_time": "2024-01-01T02:00:05Z",
  "duration": 5.2,
  "success": true,
  "error": "",
  "context": {
    "result": "processed_data"
  }
}
```

### Get Running Jobs
```http
GET /jobs/running
```

Get currently running job executions.

**Response:**
```json
[
  {
    "id": "exec_456",
    "job_id": "hourly_report",
    "pipeline": "report_pipeline.yaml",
    "status": "running",
    "start_time": "2024-01-01T10:30:00Z",
    "duration": 45.2
  }
]
```

### Get Recent Jobs
```http
GET /jobs/recent?limit=10
```

Get recent job executions.

**Query Parameters:**
- `limit` - Maximum number of results (default: 10)

### Get Job Statistics
```http
GET /jobs/statistics
```

Get job execution statistics.

**Response:**
```json
{
  "total_executions": 150,
  "successful_executions": 145,
  "failed_executions": 5,
  "success_rate": 96.7,
  "average_duration": 5.2,
  "total_duration": 780.0
}
```

### Export Jobs
```http
GET /jobs/export
```

Export all job data as JSON.

## Performance Monitoring

### Get Performance Metrics
```http
GET /performance/metrics
```

Get detailed performance metrics.

**Response:**
```json
{
  "memory": {
    "alloc": 1048576,
    "total_alloc": 2097152,
    "sys": 4194304,
    "num_gc": 5
  },
  "cpu": {
    "usage_percent": 15.5
  },
  "goroutines": 12,
  "uptime": 3600.5
}
```

### Get Performance Stats
```http
GET /performance/stats
```

Get performance statistics with system information.

**Response:**
```json
{
  "performance": {
    "memory": {
      "alloc": 1048576,
      "total_alloc": 2097152,
      "sys": 4194304,
      "num_gc": 5
    },
    "cpu": {
      "usage_percent": 15.5
    },
    "goroutines": 12,
    "uptime": 3600.5
  },
  "system": {
    "go_version": "go1.21.0",
    "num_cpu": 4,
    "num_goroutines": 12
  }
}
```

## Configuration Management

### Get Configuration
```http
GET /config
```

Get current configuration.

**Response:**
```json
{
  "server": {
    "host": "0.0.0.0",
    "port": 8080,
    "read_timeout": 30,
    "write_timeout": 30,
    "enable_cors": true,
    "max_requests": 1000
  },
  "plugins": {
    "directories": ["./pipelines", "./plugins"],
    "auto_discovery": true,
    "timeout": 60,
    "max_concurrency": 10
  },
  "scheduler": {
    "enabled": true,
    "max_jobs": 100,
    "default_timeout": 300,
    "history_retention": 30
  },
  "logging": {
    "level": "info",
    "format": "text",
    "output": "stdout",
    "file_path": "./logs/mimir.log",
    "max_size": 100,
    "max_backups": 3,
    "max_age": 30,
    "compress": true
  },
  "security": {
    "enable_auth": false,
    "jwt_secret": "change-me-in-production",
    "token_expiry": 24,
    "allowed_origins": ["*"],
    "rate_limit": 1000,
    "enable_https": false,
    "cert_file": "",
    "key_file": ""
  }
}
```

### Update Configuration
```http
PUT /config
```

Update configuration settings.

**Request Body:**
```json
{
  "logging": {
    "level": "debug"
  },
  "security": {
    "enable_auth": true
  }
}
```

**Response:**
```json
{
  "message": "Configuration updated successfully"
}
```

### Reload Configuration
```http
POST /config/reload
```

Reload configuration from file.

**Response:**
```json
{
  "message": "Configuration reloaded successfully",
  "file": "config.yaml"
}
```

### Save Configuration
```http
POST /config/save
```

Save current configuration to file.

**Request Body:**
```json
{
  "file_path": "config_backup.yaml",
  "format": "yaml"
}
```

**Response:**
```json
{
  "message": "Configuration saved successfully",
  "file": "config_backup.yaml",
  "format": "yaml"
}
```

## Visualization

### Visualize Pipeline
```http
POST /visualize/pipeline
```

Generate ASCII visualization of a pipeline.

**Request Body:**
```json
{
  "pipeline_file": "pipelines/my_pipeline.yaml"
}
```

**Response:**
```
Pipeline: My Pipeline
═══════════════════════

Steps:
├── 1. Fetch Data (Input.api) ✓ 0.2s
├── 2. Process Data (Data_Processing.transform) ▶ 1.5s
└── 3. Save Results (Output.json) ⏳ pending

Status: Running (66% complete)
```

### Visualize System Status
```http
GET /visualize/status
```

Get ASCII visualization of system status.

**Response:**
```
Mimir AIP System Status
════════════════════════

Active Jobs: 3
├── Daily Sync (running)
├── Hourly Report (pending)
└── Weekly Backup (scheduled)

Memory: [████████░░░░░░░░] 512MB / 1GB
CPU:    [████░░░░░░░░░░░░] 25%
Disk:   [██████████████░░] 850MB / 1GB

Recent Activity:
• Pipeline 'Data Sync' completed successfully (2m ago)
• Job 'Hourly Report' scheduled (5m ago)
• Plugin 'Input.api' loaded (1h ago)
```

### Visualize Scheduler
```http
GET /visualize/scheduler
```

Get ASCII visualization of scheduled jobs.

**Response:**
```
Scheduled Jobs
═══════════════

ID: daily_sync
├── Name: Daily Data Sync
├── Schedule: 0 2 * * *
├── Next Run: 2024-01-02 02:00:00
└── Status: enabled

ID: hourly_report
├── Name: Hourly Report
├── Schedule: 0 * * * *
├── Next Run: 2024-01-01 11:00:00
└── Status: enabled
```

### Visualize Plugins
```http
GET /visualize/plugins
```

Get ASCII visualization of available plugins.

**Response:**
```
Available Plugins
═════════════════

Input Plugins:
├── api - HTTP API input plugin
└── file - File input plugin

Data Processing Plugins:
├── transform - Data transformation plugin
└── filter - Data filtering plugin

Output Plugins:
├── json - JSON output plugin
└── html - HTML output plugin
```

### Visualization

#### Get ASCII Pipeline Visualization
```http
GET /visualize/pipeline/{pipeline_id}
```

Get ASCII visualization of a pipeline execution.

**Response:**
```text
Pipeline: Data Processing Pipeline
Status: Running
Progress: [████████████░░░░] 75%

Steps:
├── 1. Fetch Data (Input.api) ✓ 0.2s
├── 2. Process Data (Data_Processing.transform) ▶ 1.5s
└── 3. Save Results (Output.json) ⏳ pending
```

#### Get System Overview
```http
GET /visualize/system
```

Get ASCII visualization of system status.

**Response:**
```text
Mimir AIP System Status
════════════════════════

Active Jobs: 3
├── Daily Sync (running)
├── Hourly Report (pending)
└── Weekly Backup (scheduled)

Memory: [████████░░░░░░░░] 512MB / 1GB
CPU:    [████░░░░░░░░░░░░] 25%
Disk:   [██████████████░░] 850MB / 1GB

Recent Activity:
• Pipeline 'Data Sync' completed successfully (2m ago)
• Job 'Hourly Report' scheduled (5m ago)
• Plugin 'Input.api' loaded (1h ago)
```

## Plugin API

### BasePlugin Interface

```go
type BasePlugin interface {
    ExecuteStep(ctx context.Context, stepConfig StepConfig, globalContext PluginContext) (PluginContext, error)
    GetPluginType() string
    GetPluginName() string
    ValidateConfig(config map[string]interface{}) error
}
```

### Plugin Registration

Plugins are automatically registered based on their type and name:

```go
func (p *MyPlugin) GetPluginType() string {
    return "Data_Processing" // Input, Data_Processing, AIModels, Output
}

func (p *MyPlugin) GetPluginName() string {
    return "my_plugin"
}
```

### Plugin Discovery

The system automatically discovers plugins from configured directories:

```yaml
# config.yaml
plugins:
  directories:
    - "./plugins"
  auto_discovery: true
```

## MCP Protocol

### Base URL
```
http://localhost:8080/mcp
```

### List Tools
```http
GET /mcp/tools
```

Get a list of all available MCP tools.

**Response:**
```json
{
  "tools": [
    {
      "name": "execute_pipeline",
      "description": "Execute a Mimir pipeline",
      "inputSchema": {
        "type": "object",
        "properties": {
          "pipeline": {
            "type": "object",
            "description": "Pipeline configuration"
          }
        },
        "required": ["pipeline"]
      }
    },
    {
      "name": "get_pipeline_status",
      "description": "Get pipeline execution status",
      "inputSchema": {
        "type": "object",
        "properties": {
          "pipeline_id": {
            "type": "string",
            "description": "Pipeline ID"
          }
        },
        "required": ["pipeline_id"]
      }
    }
  ]
}
```

### Execute Tool
```http
POST /mcp/tools/{tool_name}/execute
```

Execute an MCP tool.

**Request Body:**
```json
{
  "arguments": {
    "pipeline": {
      "name": "Test Pipeline",
      "steps": [...]
    }
  }
}
```

**Response:**
```json
{
  "result": {
    "pipeline_id": "pipeline_123",
    "status": "success",
    "results": {...}
  }
}
```

### MCP Client Integration

```python
# Example MCP client usage
import mcp

client = mcp.Client("http://localhost:8080/mcp")

# List available tools
tools = client.list_tools()
print(tools)

# Execute a pipeline
result = client.execute_tool("execute_pipeline", {
    "pipeline": {
        "name": "Data Processing",
        "steps": [...]
    }
})
print(result)
```

## Configuration API

### Configuration Structure

```yaml
# config.yaml
server:
  host: "0.0.0.0"
  port: 8080
  api_key: "your-api-key"

plugins:
  directories:
    - "./plugins"
  auto_discovery: true
  timeout: 60

scheduler:
  enabled: true
  timezone: "UTC"

logging:
  level: "info"
  format: "json"
  file: "./logs/mimir.log"

monitoring:
  enabled: true
  metrics_interval: 30
```

### Environment Variables

```bash
# Server configuration
MIMIR_SERVER_HOST=0.0.0.0
MIMIR_SERVER_PORT=8080
MIMIR_API_KEY=your-api-key

# Plugin configuration
MIMIR_PLUGINS_DIRECTORIES=./plugins
MIMIR_PLUGINS_TIMEOUT=60

# Logging
MIMIR_LOG_LEVEL=info
MIMIR_LOG_FORMAT=json
```

### Configuration Validation

```go
// Validate configuration on startup
config, err := LoadConfig("config.yaml")
if err != nil {
    log.Fatal("Failed to load configuration:", err)
}

if err := config.Validate(); err != nil {
    log.Fatal("Invalid configuration:", err)
}
```

## Monitoring API

### Health Checks

#### Overall Health
```http
GET /health
```

**Response:**
```json
{
  "status": "healthy",
  "checks": {
    "database": "healthy",
    "plugins": "healthy",
    "scheduler": "healthy"
  }
}
```

#### Detailed Health
```http
GET /health/detailed
```

**Response:**
```json
{
  "status": "healthy",
  "timestamp": "2024-01-01T10:00:00Z",
  "checks": [
    {
      "name": "database",
      "status": "healthy",
      "response_time": "5ms"
    },
    {
      "name": "plugins",
      "status": "healthy",
      "loaded_plugins": 12
    }
  ]
}
```

### Metrics Collection

#### Prometheus Metrics
```http
GET /metrics
```

Returns Prometheus-compatible metrics for monitoring.

```
# Pipeline metrics
pipeline_executions_total{status="success"} 145
pipeline_executions_total{status="failed"} 5
pipeline_execution_duration_seconds 1.2

# Plugin metrics
plugin_executions_total{plugin="Input.api"} 100
plugin_execution_duration_seconds{plugin="Input.api"} 0.5

# System metrics
memory_usage_bytes 268435456
cpu_usage_percent 15.5
```

### Log Aggregation

#### Get Logs
```http
GET /logs
```

**Query Parameters:**
- `level` - Log level filter (debug, info, warn, error)
- `since` - Start time (ISO 8601)
- `until` - End time (ISO 8601)
- `limit` - Maximum number of entries

**Response:**
```json
{
  "logs": [
    {
      "timestamp": "2024-01-01T10:00:00Z",
      "level": "info",
      "message": "Pipeline execution started",
      "fields": {
        "pipeline_id": "pipeline_123",
        "pipeline_name": "Data Processing"
      }
    }
  ]
}
```

## Scheduler API

### Cron Expression Format

The scheduler uses standard cron expressions:

```
* * * * *
│ │ │ │ │
│ │ │ │ └─ Day of week (0-7, Sunday = 0 or 7)
│ │ │ └─── Month (1-12)
│ │ └───── Day of month (1-31)
│ └─────── Hour (0-23)
└───────── Minute (0-59)
```

### Examples

```bash
# Every day at 2 AM
0 2 * * *

# Every Monday at 9 AM
0 9 * * 1

# Every 30 minutes
*/30 * * * *

# Every hour on weekdays
0 * * * 1-5
```

### Timezone Support

```go
// Create job with specific timezone
job := &ScheduledJob{
    Name:     "Daily Job",
    Schedule: "0 2 * * *",
    Timezone: "America/New_York",
}
```

### Job States

- `scheduled` - Job is scheduled but not yet running
- `running` - Job is currently executing
- `completed` - Job completed successfully
- `failed` - Job failed during execution
- `disabled` - Job is disabled and won't run

### Error Handling

Jobs that fail will be retried based on the retry configuration:

```yaml
scheduler:
  retry:
    max_attempts: 3
    backoff: "exponential"
    initial_delay: "30s"
```

## Error Handling

### HTTP Status Codes

- `200 OK` - Request successful
- `201 Created` - Resource created
- `400 Bad Request` - Invalid request data
- `401 Unauthorized` - Authentication required
- `403 Forbidden` - Insufficient permissions
- `404 Not Found` - Resource not found
- `409 Conflict` - Resource conflict
- `422 Unprocessable Entity` - Validation error
- `500 Internal Server Error` - Server error

### Error Response Format

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid pipeline configuration",
    "details": {
      "field": "steps[0].config.url",
      "issue": "URL is required"
    }
  }
}
```

### Common Error Codes

- `VALIDATION_ERROR` - Configuration validation failed
- `PLUGIN_NOT_FOUND` - Specified plugin not found
- `EXECUTION_FAILED` - Pipeline execution failed
- `AUTHENTICATION_FAILED` - API key invalid
- `RESOURCE_NOT_FOUND` - Requested resource not found
- `INTERNAL_ERROR` - Internal server error

## Rate Limiting

API endpoints are rate limited to prevent abuse:

```http
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 999
X-RateLimit-Reset: 1640995200
```

### Rate Limit Headers

- `X-RateLimit-Limit` - Maximum requests per hour
- `X-RateLimit-Remaining` - Remaining requests
- `X-RateLimit-Reset` - Time when limit resets (Unix timestamp)

## WebSocket Support

### Real-time Updates

```javascript
// Connect to WebSocket
const ws = new WebSocket('ws://localhost:8080/ws');

// Listen for pipeline updates
ws.onmessage = (event) => {
    const data = JSON.parse(event.data);
    if (data.type === 'pipeline_update') {
        console.log('Pipeline status:', data.status);
    }
};
```

### Supported Events

- `pipeline_started` - Pipeline execution started
- `pipeline_step_completed` - Pipeline step completed
- `pipeline_completed` - Pipeline execution completed
- `pipeline_failed` - Pipeline execution failed
- `job_scheduled` - New job scheduled
- `job_started` - Scheduled job started
- `job_completed` - Scheduled job completed

## Versioning

API versioning follows semantic versioning:

- `/api/v1/` - Current stable version
- `/api/v2/` - Next major version (when available)

### Backward Compatibility

- Minor versions maintain backward compatibility
- Major versions may introduce breaking changes
- Deprecated endpoints include deprecation headers

```http
Deprecation: true
Link: </api/v2/pipelines>; rel="successor-version"
```

## SDKs and Libraries

### Go SDK

```go
import "github.com/Mimir-AIP/Mimir-AIP-Go/sdk"

// Create client
client := sdk.NewClient("http://localhost:8080", "your-api-key")

// Execute pipeline
result, err := client.ExecutePipeline(context.Background(), pipelineConfig)
```

### Python SDK

```python
from mimir_aip import Client

# Create client
client = Client("http://localhost:8080", "your-api-key")

# Execute pipeline
result = client.execute_pipeline(pipeline_config)
```

### JavaScript SDK

```javascript
import { MimirClient } from 'mimir-aip-sdk';

// Create client
const client = new MimirClient('http://localhost:8080', 'your-api-key');

// Execute pipeline
const result = await client.executePipeline(pipelineConfig);
```
## Best Practices

## Testing Best Practices

### 1. How to Run Tests
- All core and plugin tests are located in the `tests/` directory and within plugin folders.
- Run all tests using:
  ```bash
  go test ./...
  ```
- For coverage:
  ```bash
  go test -cover ./...
  ```
- To run a specific test file:
  ```bash
  go test ./tests/data_model_test.go
  ```
- For integration tests, use:
  ```bash
  go test ./tests/integration_test.go
  ```

### 2. How to Add New Tests
- Place new unit tests in the relevant `*_test.go` file in the corresponding package or plugin directory.
- Use Go’s standard `testing` package:
  ```go
  func TestMyFeature(t *testing.T) {
      // ... test logic ...
  }
  ```
- For plugins, add tests in the plugin’s folder (e.g., `ai_openai_plugin_test.go`).
- For pipelines, add YAML-based tests in `test_pipelines/` and use the pipeline runner to validate.

### 3. Coverage Expectations
- **Core:** All major functions, error paths, and configuration logic should be covered.
- **Plugins:** Each plugin should have tests for normal operation, error handling, and edge cases.
- **Error Handling:** Simulate and assert error responses for invalid configs, plugin failures, and network issues.
- **Config:** Test config loading, validation, and environment overrides.
- **Edge Cases:** Add tests for boundary values, empty inputs, and large data sets.

### 4. Simulating Failures and Edge Cases
- Use Go’s mocking libraries (e.g., `github.com/stretchr/testify/mock`) to simulate plugin and API failures.
- Inject errors by passing invalid configs or using mock implementations.
- For LLM/agentic plugins, mock LLM responses and simulate timeouts or malformed outputs.
- Use table-driven tests to cover multiple scenarios.

### 5. Environment Setup for Tests
- Set required environment variables in your shell or use a `.env` file for integration tests.
- Example:
  ```bash
  export MIMIR_API_KEY=test-key
  export MIMIR_SERVER_PORT=8081
  ```
- Ensure dependencies (e.g., databases, external APIs) are available or mocked.
- Use Docker or local services for integration tests if needed.

### 6. Additional Resources
- See `tests/` and `test_pipelines/` for examples.
- Refer to Go’s [testing documentation](https://pkg.go.dev/testing) for advanced usage.
- For plugin-specific testing, see `PLUGIN_DEVELOPMENT_GUIDE.md`.

---


### 1. Error Handling

#### Go Example: Structured Error Logging

```go
import "github.com/Mimir-AIP/Mimir-AIP-Go/utils"

logger := utils.GetLogger()
err := doSomething()
if err != nil {
    logger.Error("Failed to do something", err, utils.Component("my-component"), utils.RequestID("req-123"))
}
```

#### Best Practices
- Always use the structured logger for all error and info logs.
- Include context fields (component, request/user/trace IDs) for traceability.
- Log errors with stack traces for debugging (automatically included).
- Use panic recovery middleware to log stack traces and request context.
- Prefer JSON format for log aggregation and analysis in production.
- Use log levels appropriately (debug/info/warn/error/fatal).

```javascript
try {
    const result = await client.executePipeline(pipelineConfig);
    console.log('Success:', result);
} catch (error) {
    if (error.status === 400) {
        console.error('Validation error:', error.details);
    } else if (error.status === 401) {
        console.error('Authentication failed');
    } else {
        console.error('Unexpected error:', error);
    }
}
```

### 2. Pagination

```javascript
// Handle pagination
let allPipelines = [];
let offset = 0;
const limit = 50;

do {
    const response = await client.listPipelines({ limit, offset });
    allPipelines = allPipelines.concat(response.pipelines);
    offset += limit;
} while (response.pipelines.length === limit);
```

### 3. WebSocket Connection Management

```javascript
class PipelineMonitor {
    constructor() {
        this.ws = null;
        this.reconnectAttempts = 0;
    }

    connect() {
        this.ws = new WebSocket('ws://localhost:8080/ws');

        this.ws.onopen = () => {
            console.log('Connected to pipeline monitor');
            this.reconnectAttempts = 0;
        };

        this.ws.onclose = () => {
            console.log('Disconnected from pipeline monitor');
            this.reconnect();
        };

        this.ws.onerror = (error) => {
            console.error('WebSocket error:', error);
        };
    }

    reconnect() {
        if (this.reconnectAttempts < 5) {
            this.reconnectAttempts++;
            setTimeout(() => this.connect(), 1000 * this.reconnectAttempts);
        }
    }
}
```

### 4. Configuration Management

#### Go Example: Using ConfigManager and Watcher

```go
import "github.com/Mimir-AIP/Mimir-AIP-Go/utils"

// Get the global config manager
cm := utils.GetConfigManager()

// Load config from file and environment
err := utils.LoadGlobalConfig()
if err != nil {
    log.Fatal("Failed to load config:", err)
}

// Add a watcher for live reload
cm.AddWatcher(&MyConfigWatcher{})

type MyConfigWatcher struct{}
func (w *MyConfigWatcher) OnConfigChange(oldConfig, newConfig *utils.Config) {
    log.Printf("Config changed! Reloading subsystems...")
    // Reload any subsystems that depend on config here
}
```

#### Best Practices
- Always use the ConfigManager for all config access and updates.
- Use watchers to reload subsystems on config changes (hot-reload).
- Validate config on startup and after updates.
- Prefer environment variable overrides for secrets and deployment-specific settings.
- Document all config options in config.yaml and API docs.

```javascript
// Load configuration with validation
const config = {
    server: {
        host: process.env.MIMIR_HOST || 'localhost',
        port: parseInt(process.env.MIMIR_PORT) || 8080,
        apiKey: process.env.MIMIR_API_KEY
    },
    retry: {
        maxAttempts: 3,
        backoffMs: 1000
    }
};

// Validate required configuration
if (!config.server.apiKey) {
    throw new Error('MIMIR_API_KEY environment variable is required');
}
```

This comprehensive API reference covers all major aspects of the Mimir AIP platform, providing developers with the information they need to effectively integrate with and extend the system.