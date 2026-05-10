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

The REST API creates and updates pipelines with JSON request bodies. The Go pipeline models also carry YAML tags, so YAML is a convenient checked-in representation for generated or manually managed definitions, but there is no separate YAML import endpoint in the current REST API.

Implementation source of truth: `pkg/models/pipeline.go` defines the schema, `pkg/pipeline/service.go` executes steps, and `pkg/api/pipeline_handler.go` decodes REST create/update requests.

```yaml
name: My Pipeline
type: ingestion                  # ingestion | processing | output
description: "Optional description"
trigger_config:
  allow_manual: true
  webhook: false
steps:
  - name: fetch
    plugin: default              # built-in plugin (or name of an installed custom plugin)
    action: http_request         # action exposed by that plugin
    parameters:
      url: "https://api.example.com/data"
      method: GET

  - name: parse
    plugin: default
    action: parse_json
    parameters:
      data: "{{context.fetch.response.body}}"

  - name: store-records
    plugin: default
    action: store_cir_batch
    parameters:
      storage_id: "{{context._parameters.storage_id}}"
      items: "{{context.parse.parsed.items}}"
      source_uri: "https://api.example.com/data"
```

### Top-level fields

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | Human-readable pipeline name |
| `type` | Yes | `ingestion`, `processing`, or `output` |
| `description` | No | Free-text description |
| `trigger_config` | No | Optional manual/webhook trigger settings (`allow_manual`, `webhook`, `secret`, `description`) |
| `steps` | Yes | Ordered list of pipeline steps |

### Step fields

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | Unique step identifier (used in template references) |
| `plugin` | Yes unless `for_each` is set | `default` for built-in actions, `builtin` as an alias, or the name of a custom plugin |
| `action` | Yes unless `for_each` is set | The action to invoke on that plugin |
| `parameters` | No | Key-value map passed to the action. Values can be template strings |
| `output` | No | Aliases values that are already present in the pipeline context before the current step result is stored. Plugin result keys are automatically stored under the step name after the step succeeds. |
| `for_each` | No | Iterates over a resolved array and executes nested `steps` for each item. When present, `plugin` and `action` are ignored. |

### `for_each` fields

| Field | Required | Description |
|-------|----------|-------------|
| `items` | Yes | Template expression resolving to a JSON array |
| `as` | Yes | Loop variable name exposed at `{{context._loop.<as>}}` |
| `steps` | Yes | Sub-steps executed once per item |

```yaml
- name: process-items
  for_each:
    items: "{{context.parse.parsed.items}}"
    as: item
    steps:
      - name: store-item
        plugin: default
        action: store_cir
        parameters:
          storage_id: "{{context._parameters.storage_id}}"
          data: "{{context._loop.item}}"
```

---

## Context and templates

Steps share data via a **pipeline context**. The template syntax is `{{context.<step-name>.<key>}}`, with optional dot-chaining for nested keys. After a step succeeds, every key returned by the plugin action is stored under that step's name.

```yaml
# Step 1 returns response.status_code, response.body, and response.headers
- name: fetch
  plugin: default
  action: http_request
  parameters:
    url: "https://api.example.com/items"

# Step 2 reads the body returned by Step 1
- name: parse
  plugin: default
  action: parse_json
  parameters:
    data: "{{context.fetch.response.body}}"
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

## Built-in plugin actions (`plugin: default` or `plugin: builtin`)

The `default` and `builtin` plugin names both resolve to the built-in action plugin. The current built-in action set is defined in `pkg/pipeline/service.go` and dispatched in `pkg/pipeline/plugin.go`.

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
```

### `poll_http_json`

Fetches a JSON HTTP endpoint and returns only items not already represented in the provided checkpoint. When the upstream sends `ETag` or `Last-Modified`, those values are preserved in the returned checkpoint.

| Parameter | Required | Default | Description |
|-----------|----------|---------|-------------|
| `url` | Yes | — | JSON endpoint URL |
| `method` | No | `GET` | HTTP method |
| `headers` | No | — | Request headers |
| `body` | No | — | Optional request body |
| `items_path` | No | — | Dot-path to the array inside the JSON payload; defaults to `items` or the root array/object |
| `checkpoint` | No | — | Previous checkpoint object |
| `max_checkpoint_items` | No | `200` | Max remembered item hashes |

**Output keys:** `items`, `new_count`, `total_count`, `checkpoint`

### `poll_rss`

Fetches an RSS feed and emits only newly seen items based on checkpoint hashes.

| Parameter | Required | Default | Description |
|-----------|----------|---------|-------------|
| `url` | Yes | — | RSS feed URL |
| `checkpoint` | No | — | Previous checkpoint object |
| `max_checkpoint_items` | No | `200` | Max remembered item hashes |

**Output keys:** `items`, `new_count`, `total_count`, `checkpoint`


### `poll_sql_incremental`

Polls a MySQL or PostgreSQL table incrementally using a cursor column and returns only rows newer than the stored checkpoint cursor.

| Parameter | Required | Default | Description |
|-----------|----------|---------|-------------|
| `driver` | No | `mysql` | `mysql`, `postgresql`, or `pgx` |
| `dsn` | Yes | — | Database connection string |
| `table` | Yes | — | Table or view name to query |
| `cursor_column` | Yes | — | Monotonic column used for incremental progress |
| `limit` | No | `500` | Maximum rows to return per run |
| `checkpoint` | No | — | Previous checkpoint object from `load_checkpoint` |

**Output keys:** `items`, `row_count`, `checkpoint`, `source`

Returned rows include `_source_table`, and the checkpoint stores `last_cursor` so future runs only fetch newly appended data.

### `poll_csv_drop`

Polls a filesystem path glob for unseen CSV files, parses them into structured rows, and remembers processed file hashes in the checkpoint.

| Parameter | Required | Default | Description |
|-----------|----------|---------|-------------|
| `path_glob` | Yes | — | File glob such as `/data/drop/*.csv` or `./demo/*.csv` |
| `has_header` | No | `true` | Whether the first row is a header row |
| `delimiter` | No | `,` | Single-character delimiter |
| `checkpoint` | No | — | Previous checkpoint object from `load_checkpoint` |
| `max_checkpoint_items` | No | `200` | Maximum remembered processed file hashes |

**Output keys:** `items`, `new_count`, `total_count`, `checkpoint`

Returned rows include `_source_file`, making it easy to preserve provenance before storing the records in Mimir storage.


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

### `load_checkpoint` / `save_checkpoint`

Workers are stateless. For incremental ingestion across scheduled runs, persist cursor or dedupe state explicitly with checkpoint actions.

| Parameter | Action | Required | Description |
|-----------|--------|----------|-------------|
| `project_id` | both | No | Defaults to the executing pipeline project |
| `pipeline_id` | both | No | Defaults to the executing pipeline ID |
| `step_name` | both | No | Defaults to the current step name |
| `scope` | both | No | Optional namespace when one step tracks multiple feeds/tables |
| `default` | `load_checkpoint` | No | Default object to return when no checkpoint exists |
| `checkpoint` | `save_checkpoint` | Yes | Object payload to persist |
| `version` | `save_checkpoint` | No | Optimistic lock version from a previous `load_checkpoint` |

**Output keys:**

- `load_checkpoint`: `exists`, `version`, `checkpoint`, `updated_at`
- `save_checkpoint`: `saved`, `version`, `checkpoint`, `updated_at`

```yaml
steps:
  - name: load-feed-state
    plugin: default
    action: load_checkpoint
    parameters:
      step_name: supplier-feed
      scope: daily-prices
      default:
        seen_hashes: []

  - name: fetch-feed
    plugin: default
    action: poll_rss
    parameters:
      url: "https://supplier.example.com/prices.rss"
      checkpoint: "{{context.load-feed-state.checkpoint}}"

  - name: save-feed-state
    plugin: default
    action: save_checkpoint
    parameters:
      step_name: supplier-feed
      scope: daily-prices
      checkpoint: "{{context.fetch-feed.checkpoint}}"
      version: "{{context.load-feed-state.version}}"
```

### `query_sql`

Executes a SQL query and returns rows as structured objects. This is designed for stateless workers: your pipeline loads a checkpoint, interpolates it into the query, then persists the next cursor after the query completes.

| Parameter | Required | Default | Description |
|-----------|----------|---------|-------------|
| `dsn` | Yes | — | Database connection string |
| `query` | Yes | — | SQL query string (template strings supported) |
| `driver` | No | `mysql` | SQL driver name |
| `cursor_column` | No | — | Column whose last returned value becomes `next_cursor` |

**Output keys:** `items`, `row_count`, optionally `next_cursor`, `checkpoint`

```yaml
- name: load-repair-cursor
  plugin: default
  action: load_checkpoint
  parameters:
    step_name: repair-sql
    default:
      last_cursor: "1970-01-01T00:00:00Z"

- name: fetch-repairs
  plugin: default
  action: query_sql
  parameters:
    dsn: "{{context._parameters.mysql_dsn}}"
    query: >-
      SELECT repair_id, amount, updated_at
      FROM repairs
      WHERE updated_at > '{{context.load-repair-cursor.checkpoint.last_cursor}}'
      ORDER BY updated_at
    cursor_column: updated_at

- name: save-repair-cursor
  plugin: default
  action: save_checkpoint
  parameters:
    step_name: repair-sql
    checkpoint: "{{context.fetch-repairs.checkpoint}}"
    version: "{{context.load-repair-cursor.version}}"
```

### `ingest_csv_url`

Fetches a CSV document from a URL, parses rows into structured objects, and applies the same dedupe checkpoint model as `ingest_csv`. ETag / Last-Modified values are carried inside the returned checkpoint when present.

| Parameter | Required | Default | Description |
|-----------|----------|---------|-------------|
| `url` | Yes | — | CSV URL |
| `headers` | No | — | Additional request headers |
| `delimiter` | No | `,` | Single-character delimiter |
| `has_header` | No | `true` | Whether the first row is a header row |
| `checkpoint` | No | — | Previous checkpoint object from `load_checkpoint` or a prior run |

**Output keys:** `items`, `headers`, `new_count`, `total_count`, `checkpoint`


### `ingest_csv`

Parses CSV content supplied directly in the `csv_data` parameter and applies the same dedupe checkpoint model as `ingest_csv_url`.

| Parameter | Required | Default | Description |
|-----------|----------|---------|-------------|
| `csv_data` | Yes | — | CSV document content |
| `delimiter` | No | `,` | Single-character delimiter |
| `has_header` | No | `true` | Whether the first row is a header row |
| `checkpoint` | No | — | Previous checkpoint object from `load_checkpoint` or a prior run |
| `max_checkpoint_items` | No | `200` | Maximum remembered row hashes |

**Output keys:** `items`, `headers`, `new_count`, `total_count`, `checkpoint`

### `store_cir`

Stores one CIR record through a configured Mimir storage backend. Storage-backed built-in actions require the pipeline runtime to have storage integration configured; local and worker execution paths inject this dependency.

| Parameter | Required | Default | Description |
|-----------|----------|---------|-------------|
| `storage_id` | Yes | — | Storage config ID |
| `data` | Yes | — | Record data as an object or JSON string |
| `source_uri` | No | `pipeline://ingestion` | Provenance URI |
| `source_type` | No | `api` | CIR source type |
| `format` | No | `json` | CIR data format |

**Output keys:** `stored`, `affected_items`

### `store_cir_batch`

Stores an array of records as individual CIR entries.

| Parameter | Required | Default | Description |
|-----------|----------|---------|-------------|
| `storage_id` | Yes | — | Storage config ID |
| `items` | Yes | — | Array of records, or a template resolving to a JSON array |
| `source_uri` | No | `pipeline://ingestion` | Provenance URI |
| `source_type` | No | `api` | CIR source type |
| `format` | No | `json` | CIR data format |

**Output keys:** `stored`, `total`

### `send_email`

Sends a plain-text email through SMTP.

| Parameter | Required | Default | Description |
|-----------|----------|---------|-------------|
| `to` | Yes | — | Recipient address |
| `subject` | No | — | Email subject |
| `body` | No | — | Plain-text body |
| `from` | No | `username` | Sender address |
| `smtp_host` | Yes unless `$SMTP_HOST` is set | `$SMTP_HOST` | SMTP host |
| `smtp_port` | No | `$SMTP_PORT` or `587` | SMTP port |
| `username` | No | `$SMTP_USERNAME` | SMTP username |
| `password` | No | `$SMTP_PASSWORD` | SMTP password |

**Output keys:** `sent`, `to`

### `send_webhook`

Posts a JSON payload to an HTTP endpoint.

| Parameter | Required | Default | Description |
|-----------|----------|---------|-------------|
| `url` | Yes | — | Webhook endpoint URL |
| `payload` | No | `{}` | JSON body as a map or string; template strings supported |
| `headers` | No | — | Additional HTTP headers |

**Output keys:** `sent`, `status_code`, `response`


---

## Using custom plugins

Replace `plugin: default` with the manifest `name` of any installed custom pipeline plugin. For example, `OpenLibraryMimirPlugin` installs as `openlibrary` because its `plugin.yaml` sets `name: openlibrary`.

```yaml
steps:
  - name: ingest-data
    plugin: openlibrary   # plugin.yaml name from github.com/Mimir-AIP/OpenLibraryMimirPlugin
    action: search_books
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
curl -X POST http://localhost:8080/api/pipelines/pipe-456/execute \
  -H "Content-Type: application/json" \
  -d '{
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
    "pipelines": ["pipe-456"],
    "name": "Nightly ingest",
    "cron_schedule": "0 2 * * *",
    "enabled": true
  }'
```



## Further reading

- [Custom plugins](./custom-plugins.md) — building pipeline step plugins, storage plugins, and LLM provider plugins
- [OpenAPI specification](./openapi.yaml) — complete API reference
