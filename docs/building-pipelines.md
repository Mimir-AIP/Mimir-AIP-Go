# Building Pipelines in Mimir AIP

Pipelines define sequences of processing steps that workers execute as Kubernetes Jobs. Each step calls a plugin action, optionally passes data between steps via a shared context, and produces output that later steps can reference.

---

## Pipeline types

| Type | Purpose |
|------|---------|
| `ingestion` | Pull data from external sources into Mimir storage |
| `processing` | Transform, enrich, or analyse data already in storage |
| `output` | Export or deliver results to external systems |

The `type` field affects how the orchestrator categorises and schedules the pipeline but does not restrict which steps or plugins you can use.

---

## Pipeline structure

Pipelines are defined as JSON (via the API) or YAML (for version-controlled definitions). The schema is identical in both formats.

```yaml
name: My Pipeline
type: ingestion                  # ingestion | processing | output
description: "Optional description"
steps:
  - name: step-one
    plugin: default              # built-in plugin (or name of an installed custom plugin)
    action: http_request         # action exposed by that plugin
    parameters:
      url: "https://api.example.com/data"
      method: GET
    output:
      raw_body: "{{context.step-one.response.body}}"

  - name: step-two
    plugin: default
    action: parse_json
    parameters:
      data: "{{context.step-one.raw_body}}"
    output:
      records: "{{context.step-two.parsed}}"
```

### Top-level fields

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | Human-readable pipeline name |
| `type` | Yes | `ingestion`, `processing`, or `output` |
| `description` | No | Free-text description |
| `steps` | Yes | Ordered list of pipeline steps |

### Step fields

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | Unique step identifier (used in template references) |
| `plugin` | Yes | `default` for built-in actions, or the name of a custom plugin |
| `action` | Yes | The action to invoke on that plugin |
| `parameters` | No | Key-value map passed to the action. Values can be template strings |
| `output` | No | Named values to extract from the step result into context |

---

## Context and templates

Steps share data via a **pipeline context**. The template syntax is `{{context.<step-name>.<key>}}`, with optional dot-chaining for nested keys.

```yaml
# Step 1 sets data
- name: fetch
  plugin: default
  action: http_request
  parameters:
    url: "https://api.example.com/items"
  output:
    body: "{{context.fetch.response.body}}"

# Step 2 reads it
- name: parse
  plugin: default
  action: parse_json
  parameters:
    data: "{{context.fetch.body}}"
```

Pipeline-level parameters passed at execution time are available as `{{context._parameters.<key>}}`:

```yaml
- name: fetch
  plugin: default
  action: http_request
  parameters:
    url: "https://api.example.com/{{context._parameters.resource_id}}"
```

---

## Built-in plugin actions (`plugin: default`)

The `default` (or `builtin`) plugin is always available with no installation required.

### `http_request`

Makes an HTTP request.

| Parameter | Required | Default | Description |
|-----------|----------|---------|-------------|
| `url` | Yes | — | Target URL (template strings supported) |
| `method` | No | `GET` | HTTP method |
| `body` | No | — | Request body string (template strings supported) |
| `headers` | No | — | Map of header name → value |

**Output keys:** `response.status_code`, `response.body`, `response.headers`

```yaml
- name: call-api
  plugin: default
  action: http_request
  parameters:
    url: "https://api.example.com/items"
    method: POST
    headers:
      Content-Type: "application/json"
      Authorization: "Bearer {{context._parameters.api_token}}"
    body: '{"query": "all"}'
  output:
    status: "{{context.call-api.response.status_code}}"
    body:   "{{context.call-api.response.body}}"
```

### `parse_json`

Parses a JSON string into a structured value.

| Parameter | Required | Description |
|-----------|----------|-------------|
| `data` | Yes | JSON string to parse (template strings supported) |

**Output keys:** `parsed` (the decoded value)

```yaml
- name: decode
  plugin: default
  action: parse_json
  parameters:
    data: "{{context.call-api.body}}"
  output:
    items: "{{context.decode.parsed}}"
```

### `set_context`

Explicitly writes a value into the context.

| Parameter | Required | Default | Description |
|-----------|----------|---------|-------------|
| `key` | Yes | — | Key name |
| `value` | Yes | — | Value (template strings supported) |
| `step` | No | `_global` | Step namespace to write into |

```yaml
- name: init
  plugin: default
  action: set_context
  parameters:
    key: base_url
    value: "https://api.example.com"
    step: config
```

### `get_context`

Reads a value from the context.

| Parameter | Required | Default | Description |
|-----------|----------|---------|-------------|
| `key` | Yes | — | Key name |
| `step` | No | `_global` | Step namespace to read from |

**Output keys:** `value`, `exists`

### `if_else`

Conditional branching.

| Parameter | Required | Description |
|-----------|----------|-------------|
| `condition` | Yes | Evaluated as truthy unless empty, `"0"`, `"false"`, or `"null"` |
| `if_true` | No | Value to return when condition is truthy |
| `if_false` | No | Value to return when condition is falsy |

**Output keys:** `result` (the chosen value), `condition` (the evaluated boolean)

```yaml
- name: check
  plugin: default
  action: if_else
  parameters:
    condition: "{{context.fetch.response.status_code}}"
    if_true: "ok"
    if_false: "failed"
  output:
    status_label: "{{context.check.result}}"
```

### `goto`

Jumps to a named step (for simple loops). Use sparingly — unbounded loops will run until the job timeout.

| Parameter | Required | Description |
|-----------|----------|-------------|
| `target` | Yes | Name of the step to jump to |

```yaml
- name: loop-back
  plugin: default
  action: goto
  parameters:
    target: fetch
```

---

## Using custom plugins

Replace `plugin: default` with the name of any installed custom plugin. The name matches the repository slug used at install time (last URL segment, `.git` stripped).

```yaml
steps:
  - name: ingest-data
    plugin: open-library-plugin   # installed from github.com/Mimir-AIP/OpenLibraryMimirPlugin
    action: ingest
    parameters:
      subject: "science"
      limit: 100
```

See [custom-plugins.md](./custom-plugins.md) for how to build and install custom plugins.

---

## API reference

### Create a pipeline

```bash
curl -X POST http://localhost:8080/api/pipelines \
  -H "Content-Type: application/json" \
  -d '{
    "project_id": "proj-123",
    "name": "My Pipeline",
    "type": "ingestion",
    "steps": [
      {
        "name": "fetch",
        "plugin": "default",
        "action": "http_request",
        "parameters": {"url": "https://api.example.com/data"}
      }
    ]
  }'
```

### Execute a pipeline

```bash
curl -X POST http://localhost:8080/api/pipelines/execute \
  -H "Content-Type: application/json" \
  -d '{
    "pipeline_id": "pipe-456",
    "trigger_type": "manual",
    "parameters": {
      "resource_id": "abc123",
      "api_token": "secret"
    }
  }'
```

Parameters passed here are available inside steps as `{{context._parameters.<key>}}`.

### List pipelines

```bash
curl http://localhost:8080/api/pipelines?project_id=proj-123
```

### Update a pipeline

```bash
curl -X PUT http://localhost:8080/api/pipelines/pipe-456 \
  -H "Content-Type: application/json" \
  -d '{"description": "Updated description"}'
```

---

## Scheduling pipelines

Pipelines can be triggered on a cron schedule:

```bash
curl -X POST http://localhost:8080/api/schedules \
  -H "Content-Type: application/json" \
  -d '{
    "project_id": "proj-123",
    "pipeline_id": "pipe-456",
    "name": "Nightly ingest",
    "cron_schedule": "0 2 * * *",
    "enabled": true
  }'
```

---

## Further reading

- [Custom plugins](./custom-plugins.md) — building pipeline step plugins, storage plugins, and LLM provider plugins
- [OpenAPI specification](./openapi.yaml) — complete API reference
