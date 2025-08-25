# Mimir AIP API Reference

This document provides comprehensive API reference for the Mimir AIP platform.

## Table of Contents

1. [REST API Endpoints](#rest-api-endpoints)
2. [Plugin API](#plugin-api)
3. [MCP Protocol](#mcp-protocol)
4. [Configuration API](#configuration-api)
5. [Monitoring API](#monitoring-api)
6. [Scheduler API](#scheduler-api)

## REST API Endpoints

### Base URL
```
http://localhost:8080/api/v1
```

### Authentication
All endpoints require authentication via API key header:
```
Authorization: Bearer <api_key>
```

### Pipeline Management

#### Execute Pipeline
```http
POST /pipelines/execute
```

Execute a pipeline with the provided configuration.

**Request Body:**
```json
{
  "name": "My Pipeline",
  "steps": [
    {
      "name": "Fetch Data",
      "plugin": "Input.api",
      "config": {
        "url": "https://api.example.com/data",
        "method": "GET"
      },
      "output": "api_data"
    },
    {
      "name": "Process Data",
      "plugin": "Data_Processing.transform",
      "config": {
        "operation": "uppercase"
      },
      "output": "processed_data"
    }
  ]
}
```

**Response:**
```json
{
  "pipeline_id": "pipeline_123",
  "status": "success",
  "execution_time": "1.2s",
  "results": {
    "processed_data": "TRANSFORMED DATA"
  }
}
```

#### Get Pipeline Status
```http
GET /pipelines/{pipeline_id}/status
```

Get the current status of a pipeline execution.

**Response:**
```json
{
  "pipeline_id": "pipeline_123",
  "status": "running",
  "current_step": 2,
  "total_steps": 4,
  "start_time": "2024-01-01T10:00:00Z",
  "estimated_completion": "2024-01-01T10:00:30Z"
}
```

#### List Pipelines
```http
GET /pipelines
```

Get a list of all pipelines.

**Query Parameters:**
- `status` - Filter by status (running, completed, failed)
- `limit` - Maximum number of results (default: 50)
- `offset` - Pagination offset (default: 0)

**Response:**
```json
{
  "pipelines": [
    {
      "id": "pipeline_123",
      "name": "Data Processing Pipeline",
      "status": "completed",
      "created_at": "2024-01-01T10:00:00Z",
      "execution_time": "2.5s"
    }
  ],
  "total": 1
}
```

### Plugin Management

#### List Plugins
```http
GET /plugins
```

Get a list of all available plugins.

**Response:**
```json
{
  "plugins": [
    {
      "type": "Input",
      "name": "api",
      "version": "1.0.0",
      "description": "HTTP API input plugin"
    },
    {
      "type": "Data_Processing",
      "name": "transform",
      "version": "1.0.0",
      "description": "Data transformation plugin"
    }
  ]
}
```

#### Get Plugin Configuration
```http
GET /plugins/{type}/{name}/config
```

Get the configuration schema for a specific plugin.

**Response:**
```json
{
  "plugin": "Input.api",
  "config_schema": {
    "url": {
      "type": "string",
      "required": true,
      "description": "API endpoint URL"
    },
    "method": {
      "type": "string",
      "required": false,
      "default": "GET",
      "enum": ["GET", "POST", "PUT", "DELETE"]
    },
    "headers": {
      "type": "object",
      "required": false,
      "description": "HTTP headers"
    }
  }
}
```

#### Validate Plugin Configuration
```http
POST /plugins/{type}/{name}/validate
```

Validate a plugin configuration.

**Request Body:**
```json
{
  "config": {
    "url": "https://api.example.com/data",
    "method": "GET"
  }
}
```

**Response:**
```json
{
  "valid": true,
  "errors": []
}
```

### Job Scheduling

#### Create Scheduled Job
```http
POST /scheduler/jobs
```

Create a new scheduled job.

**Request Body:**
```json
{
  "name": "Daily Data Sync",
  "pipeline": {
    "name": "Data Sync Pipeline",
    "steps": [...]
  },
  "schedule": "0 2 * * *",
  "enabled": true,
  "timezone": "UTC"
}
```

**Response:**
```json
{
  "job_id": "job_123",
  "name": "Daily Data Sync",
  "next_run": "2024-01-02T02:00:00Z",
  "created_at": "2024-01-01T10:00:00Z"
}
```

#### List Scheduled Jobs
```http
GET /scheduler/jobs
```

Get a list of all scheduled jobs.

**Response:**
```json
{
  "jobs": [
    {
      "id": "job_123",
      "name": "Daily Data Sync",
      "schedule": "0 2 * * *",
      "enabled": true,
      "next_run": "2024-01-02T02:00:00Z",
      "last_run": "2024-01-01T02:00:00Z",
      "last_status": "success"
    }
  ]
}
```

#### Update Scheduled Job
```http
PUT /scheduler/jobs/{job_id}
```

Update an existing scheduled job.

**Request Body:**
```json
{
  "enabled": false,
  "schedule": "0 4 * * *"
}
```

#### Delete Scheduled Job
```http
DELETE /scheduler/jobs/{job_id}
```

Delete a scheduled job.

### Monitoring

#### Get System Status
```http
GET /monitoring/status
```

Get overall system status.

**Response:**
```json
{
  "status": "healthy",
  "uptime": "2h 30m",
  "version": "1.0.0",
  "plugins_loaded": 12,
  "active_jobs": 3,
  "memory_usage": "256MB",
  "cpu_usage": "15%"
}
```

#### Get Pipeline Metrics
```http
GET /monitoring/pipelines
```

Get pipeline execution metrics.

**Query Parameters:**
- `period` - Time period (1h, 24h, 7d, 30d)
- `pipeline_name` - Filter by pipeline name

**Response:**
```json
{
  "metrics": {
    "total_executions": 150,
    "successful_executions": 145,
    "failed_executions": 5,
    "average_execution_time": "1.2s",
    "success_rate": "96.7%"
  },
  "trends": [
    {
      "timestamp": "2024-01-01T10:00:00Z",
      "executions": 10,
      "success_rate": "100%"
    }
  ]
}
```

#### Get Plugin Metrics
```http
GET /monitoring/plugins
```

Get plugin usage metrics.

**Response:**
```json
{
  "plugin_metrics": [
    {
      "plugin": "Input.api",
      "executions": 100,
      "average_execution_time": "0.5s",
      "error_rate": "2.0%"
    }
  ]
}
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

### MCP Server Endpoints

#### List Available Tools
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

#### Execute Tool
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

### 1. Error Handling

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