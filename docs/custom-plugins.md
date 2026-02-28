# Writing Custom Plugins for Mimir AIP

Mimir AIP supports two distinct plugin extension points:

| Type | When to use | Loading mechanism |
|------|-------------|-------------------|
| **Pipeline plugin** | Add new actions to pipeline steps (call external APIs, transform data, integrate services) | Dynamic — installed via the API, compiled by workers at runtime from a Git repository |
| **Storage plugin** | Add a new storage backend (a database or service that CIR data can be stored in and retrieved from) | Static — compiled into the orchestrator binary, requires a custom Docker image |

---

## Part 1 — Pipeline Plugins

Pipeline plugins extend what pipeline steps can do. The [OpenLibrary plugin](https://github.com/Mimir-AIP/OpenLibraryMimirPlugin) is a real-world example you can reference.

### How they work

When a worker executes a pipeline step that references a custom plugin, it:
1. Clones the plugin's Git repository (already registered via the API).
2. Compiles the plugin source using `go build -buildmode=plugin`.
3. Loads the compiled `.so` and looks for an exported `Plugin` symbol.
4. Calls `Plugin.Execute(action, params, ctx)` for each step that uses the plugin.

### Required structure

A pipeline plugin repository must contain these files:

```
my-plugin/
├── plugin.go        # Exports the Plugin symbol and routes actions
├── plugin.yaml      # Manifest — describes the plugin and its actions
├── go.mod           # Declares the module (mimir version is not required — see below)
└── actions/         # Optional: split action implementations across files
    ├── search.go
    └── fetch.go
```

The worker's compiler:
1. Clones the repository into a temporary directory.
2. Moves all `.go` files from `actions/` to the root (so your actions can live in a clean subdirectory).
3. **Deletes the plugin's `go.mod` and `go.sum`** — the worker uses the host module's `go.mod` instead, ensuring the plugin builds against the exact same `mimir-aip-go` source the worker was compiled from.
4. Runs `go build -buildmode=plugin` from `/app` (the worker image root).

### `plugin.go` — the Go interface

Your plugin must be in `package main` and export a variable named `Plugin` that satisfies the following interface (defined in `pkg/pipeline/plugin.go`):

```go
type Plugin interface {
    Execute(action string, params map[string]interface{}, ctx *models.PipelineContext) (map[string]interface{}, error)
}
```

Optionally implement `GetActions() []string` to advertise which actions your plugin supports — this is surfaced in the API and UI.

**Minimal example (`plugin.go`):**

```go
package main

import "github.com/mimir-aip/mimir-aip-go/pkg/models"

// Plugin is the symbol the worker loader looks for.
// Declare it as a value type — the loader calls plugin.Lookup("Plugin") which
// returns a pointer to this package-level variable, so pointer-receiver methods
// on MyPlugin are automatically available via the interface.
var Plugin MyPlugin

type MyPlugin struct{}

func (p *MyPlugin) Execute(action string, params map[string]interface{}, ctx *models.PipelineContext) (map[string]interface{}, error) {
    switch action {
    case "do_something":
        return doSomething(params, ctx)
    default:
        return nil, nil // unknown actions return nil, nil (not an error)
    }
}

func (p *MyPlugin) GetActions() []string {
    return []string{"do_something"}
}
```

**Example with `actions/` layout (`actions/something.go`):**

```go
package main  // must be package main, even in actions/

import "github.com/mimir-aip/mimir-aip-go/pkg/models"

func doSomething(params map[string]interface{}, ctx *models.PipelineContext) (map[string]interface{}, error) {
    input, _ := params["input"].(string)
    return map[string]interface{}{"result": input + "_processed"}, nil
}
```

#### `models.PipelineContext`

The context is passed through every step of a pipeline run. Use it to share data between steps:

```go
// Read data set by a previous step
value, exists := ctx.GetStepData("previous_step_name", "key")

// Write data for downstream steps
ctx.SetStepData("my_step_name", "key", value)
```

Template variables in step parameters (`{{context.step_name.key}}`) are resolved by the executor before `Execute` is called, so `params` already contain the resolved values.

#### Return values

Return a `map[string]interface{}` containing the outputs you want to expose to downstream steps. Keys in this map become available as `{{context.<step_name>.<key>}}` in subsequent steps.

Return `nil, nil` for unknown actions (the executor will skip to the next step). Return a non-nil error to fail the step.

---

### `plugin.yaml` — the manifest

The manifest describes the plugin and documents each action's parameters and return values. It is stored by the orchestrator and surfaced in the API and UI.

```yaml
name: my-plugin           # Unique identifier used in pipeline step definitions
version: 1.0.0
description: One-line description of what the plugin does
author: Your Name

actions:
  - name: do_something
    description: Does something useful
    parameters:
      - name: input
        type: string
        required: true
        description: The input value
      - name: limit
        type: number
        required: false
        description: Optional limit (default 10)
    returns:
      - name: result
        type: string
        description: The processed result
      - name: count
        type: number
        description: Number of items processed
```

**Supported parameter types:** `string`, `number`, `boolean`, `array`, `object`

---

### `go.mod`

Include a standard `go.mod` so the repository is a valid Go module and IDEs can resolve types while you write the plugin:

```
module github.com/your-org/my-plugin

go 1.21

require github.com/mimir-aip/mimir-aip-go v0.1.0
```

> **Important:** The worker **deletes** your `go.mod` before compiling. The plugin is built inside `/app` using the worker image's own `go.mod`, so the module path and `mimir-aip-go` version you declare here have no effect at compile time. The `go.mod` is there for your development environment only.

---

### Installing a pipeline plugin

Once your plugin is in a Git repository, install it via the REST API or MCP:

**REST API:**
```bash
curl -X POST http://localhost:8080/api/plugins \
  -H "Content-Type: application/json" \
  -d '{"repository_url": "https://github.com/your-org/my-plugin"}'
```

**MCP (in Claude Code or another MCP client):**
```
install_plugin repository_url=https://github.com/your-org/my-plugin
```

To pin a specific branch or tag:
```bash
curl -X POST http://localhost:8080/api/plugins \
  -H "Content-Type: application/json" \
  -d '{"repository_url": "https://github.com/your-org/my-plugin", "git_ref": "v1.2.0"}'
```

---

### Using a plugin in a pipeline

Reference the plugin by its `name` from `plugin.yaml` in any pipeline step:

```yaml
steps:
  - name: fetch_data
    plugin: my-plugin
    action: do_something
    parameters:
      input: "hello"
      limit: 5

  - name: use_result
    plugin: default
    action: http_request
    parameters:
      url: "https://example.com/webhook"
      method: POST
      body: '{"value": "{{context.fetch_data.result}}"}'
```

The built-in `default` plugin provides: `http_request`, `parse_json`, `if_else`, `set_context`, `get_context`, `goto`.

---

### Updating and uninstalling

```bash
# Pull the latest version from the repository
curl -X PUT http://localhost:8080/api/plugins/my-plugin \
  -H "Content-Type: application/json" \
  -d '{}'

# Pin to a specific version
curl -X PUT http://localhost:8080/api/plugins/my-plugin \
  -H "Content-Type: application/json" \
  -d '{"git_ref": "v1.3.0"}'

# Uninstall
curl -X DELETE http://localhost:8080/api/plugins/my-plugin
```

---

## Part 2 — Storage Plugins

Storage plugins add new backend types to Mimir's CIR storage layer. Unlike pipeline plugins, they are **compiled into the orchestrator binary** and require rebuilding the Docker image. This gives maximum performance and no runtime compilation overhead.

Eight backends are bundled: `filesystem`, `postgresql`, `mysql`, `mongodb`, `s3`, `redis`, `elasticsearch`, `neo4j`. Build a custom plugin when you need a backend not in this list.

### The interface

Implement `models.StoragePlugin` (defined in `pkg/models/storage.go`):

```go
type StoragePlugin interface {
    // Initialize is called once when the storage config is first used.
    // config.ConnectionString is the primary DSN/URL.
    // config.Options is a free-form map of additional settings.
    Initialize(config *PluginConfig) error

    // CreateSchema creates or updates storage structures (tables, indices, collections)
    // based on the provided ontology definition. Called when a storage config is activated.
    CreateSchema(ontology *OntologyDefinition) error

    // Store writes a single CIR record to the backend.
    Store(cir *CIR) (*StorageResult, error)

    // Retrieve queries CIR records matching the given query.
    // An empty CIRQuery returns all records.
    Retrieve(query *CIRQuery) ([]*CIR, error)

    // Update applies the given updates to records matching the query.
    Update(query *CIRQuery, updates *CIRUpdate) (*StorageResult, error)

    // Delete removes records matching the query.
    Delete(query *CIRQuery) (*StorageResult, error)

    // GetMetadata returns static metadata about the backend.
    GetMetadata() (*StorageMetadata, error)

    // HealthCheck tests whether the backend is reachable and operational.
    HealthCheck() (bool, error)
}
```

Key types are all in `pkg/models/storage.go`:

| Type | Description |
|------|-------------|
| `PluginConfig` | `ConnectionString string`, `Credentials map[string]interface{}`, `Options map[string]interface{}` |
| `CIR` | `ID`, `Source`, `Data`, `Metadata` — the normalised record format |
| `CIRQuery` | `EntityType`, `Filters []CIRCondition`, `OrderBy`, `Limit`, `Offset` |
| `CIRUpdate` | `Filters []CIRCondition`, `Updates map[string]interface{}` |
| `StorageResult` | `Success bool`, `AffectedItems int`, `Error string` |
| `StorageMetadata` | `StorageType string`, `Version string`, `Capabilities []string` |

### Implementation skeleton

Create a new file in `pkg/storage/plugins/`:

```go
// pkg/storage/plugins/mybackend.go
package plugins

import (
    "fmt"
    "github.com/mimir-aip/mimir-aip-go/pkg/models"
)

// MyBackendPlugin implements models.StoragePlugin for MyBackend.
type MyBackendPlugin struct {
    client      *mybackend.Client // replace with your client type
    initialized bool
}

func NewMyBackendPlugin() *MyBackendPlugin {
    return &MyBackendPlugin{}
}

func (p *MyBackendPlugin) Initialize(config *models.PluginConfig) error {
    dsn := config.ConnectionString
    if dsn == "" {
        return fmt.Errorf("mybackend: connection_string is required")
    }

    client, err := mybackend.Connect(dsn)
    if err != nil {
        return fmt.Errorf("mybackend: failed to connect: %w", err)
    }

    p.client = client
    p.initialized = true
    return nil
}

func (p *MyBackendPlugin) CreateSchema(ontology *models.OntologyDefinition) error {
    if !p.initialized {
        return fmt.Errorf("mybackend: not initialized")
    }
    // Create collections/tables/indices for each entity type in the ontology
    for _, entity := range ontology.Entities {
        if err := p.client.EnsureCollection(entity.Name); err != nil {
            return fmt.Errorf("mybackend: CreateSchema: %w", err)
        }
    }
    return nil
}

func (p *MyBackendPlugin) Store(cir *models.CIR) (*models.StorageResult, error) {
    if !p.initialized {
        return nil, fmt.Errorf("mybackend: not initialized")
    }
    // Translate the CIR record to your backend's format and write it
    if err := p.client.Insert(cir.ID, cir.Data); err != nil {
        return &models.StorageResult{Success: false, Error: err.Error()}, nil
    }
    return &models.StorageResult{Success: true, AffectedItems: 1}, nil
}

func (p *MyBackendPlugin) Retrieve(query *models.CIRQuery) ([]*models.CIR, error) {
    if !p.initialized {
        return nil, fmt.Errorf("mybackend: not initialized")
    }
    // Translate CIRQuery conditions to your query language and execute
    records, err := p.client.Query(translateQuery(query))
    if err != nil {
        return nil, fmt.Errorf("mybackend: Retrieve: %w", err)
    }
    // Map results back to []*models.CIR
    return toCIRSlice(records), nil
}

func (p *MyBackendPlugin) Update(query *models.CIRQuery, updates *models.CIRUpdate) (*models.StorageResult, error) {
    // Implement update logic
    return &models.StorageResult{Success: true}, nil
}

func (p *MyBackendPlugin) Delete(query *models.CIRQuery) (*models.StorageResult, error) {
    // Implement delete logic
    return &models.StorageResult{Success: true}, nil
}

func (p *MyBackendPlugin) GetMetadata() (*models.StorageMetadata, error) {
    return &models.StorageMetadata{
        StorageType:  "mybackend",
        Version:      "1.0.0",
        Capabilities: []string{"store", "retrieve", "update", "delete"},
    }, nil
}

func (p *MyBackendPlugin) HealthCheck() (bool, error) {
    if !p.initialized {
        return false, fmt.Errorf("not initialized")
    }
    if err := p.client.Ping(); err != nil {
        return false, err
    }
    return true, nil
}
```

### Registering the plugin

Open `cmd/orchestrator/main.go` and add your plugin alongside the existing eight:

```go
import (
    // ... existing imports ...
    storageplugins "github.com/mimir-aip/mimir-aip-go/pkg/storage/plugins"
)

// In the storage registration block:
storageService.RegisterPlugin("filesystem",     storageplugins.NewFilesystemPlugin())
storageService.RegisterPlugin("postgresql",     storageplugins.NewPostgresPlugin())
// ... other built-in plugins ...
storageService.RegisterPlugin("mybackend",      storageplugins.NewMyBackendPlugin()) // ← add this
```

The string `"mybackend"` becomes the `plugin_type` value used when creating a storage config:

```bash
curl -X POST http://localhost:8080/api/storage/configs \
  -H "Content-Type: application/json" \
  -d '{
    "project_id": "<project-id>",
    "plugin_type": "mybackend",
    "config": {
      "connection_string": "mybackend://localhost:9000/mydb"
    }
  }'
```

### Adding Go dependencies

Add your backend's client library to the project's `go.mod`:

```bash
cd /path/to/mimir-aip-go
go get github.com/example/mybackend-go@latest
go mod tidy
```

### Building a custom Docker image

After registering your plugin and adding its dependency, rebuild the orchestrator image:

```bash
# From the repository root
docker build -f cmd/orchestrator/Dockerfile -t my-registry/mimir-orchestrator:custom .
```

Update your Helm values or `docker-compose.yaml` to use the custom image tag.

---

## Reference — CIR format

All storage operations use the Common Internal Representation (CIR):

```json
{
  "id": "uuid",
  "source": {
    "type": "pipeline",
    "pipeline_id": "...",
    "step_name": "..."
  },
  "data": {
    "entity_type": "SomeEntity",
    "field_a": "value",
    "field_b": 42
  },
  "metadata": {
    "ingested_at": "2026-01-01T00:00:00Z",
    "tags": ["raw"]
  }
}
```

`data` is the primary payload. `source` records provenance. `metadata` is for operational tags and timestamps. The `entity_type` field inside `data` is used by the digital twin sync to classify entities.

## Reference — built-in actions

The `default` plugin (no installation required) provides:

| Action | Description |
|--------|-------------|
| `http_request` | Make an HTTP request. Params: `url`, `method`, `headers`, `body`. |
| `parse_json` | Parse a JSON string. Params: `data`. Returns: `parsed`. |
| `if_else` | Conditional branch. Params: `condition`, `if_true`, `if_false`. |
| `set_context` | Write a value to the pipeline context. Params: `step`, `key`, `value`. |
| `get_context` | Read a value from the pipeline context. Params: `step`, `key`. |
| `goto` | Jump to a named step. Params: `target`. |
