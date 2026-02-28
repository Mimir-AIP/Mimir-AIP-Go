# Custom Plugins for Mimir AIP

Mimir supports two kinds of runtime-extensible plugins:

1. **Pipeline Step Plugins** — custom processing steps that workers execute inside pipelines
2. **Storage Plugins** — custom storage backends that the orchestrator loads at runtime

---

## Part 1 — Pipeline Step Plugins

Pipeline step plugins let you add new processing steps without modifying the core Mimir codebase. Workers clone, compile, and execute your plugin as a Go plugin (`.so`) at runtime.

### Plugin Interface

Your plugin must implement the `Plugin` interface from the Mimir worker:

```go
type Plugin interface {
    Execute(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error)
}
```

The optional `GetActions() []string` method is supported but not required.

### Repository Structure

```
my-plugin/
├── plugin.go          # Required: exports var Plugin
├── plugin.yaml        # Required: plugin manifest
├── go.mod             # Present in your repo but DELETED by the worker before compilation
└── actions/           # Optional: action handlers (files are flattened to root at compile time)
    ├── ingest.go
    └── transform.go
```

> **Important:** The worker deletes your plugin's `go.mod` and `go.sum` before compilation. Your plugin is compiled as part of the host module using the same dependency versions as the orchestrator. You do not need to pin any Mimir dependencies in your `go.mod` — it exists only for local development / IDE support.

### plugin.yaml

```yaml
name: my-plugin
version: "1.0.0"
description: "Does something useful"
author: "Your Name"
actions:
  - ingest
  - transform
```

### Required Export

Your plugin must export a package-level variable named `Plugin`:

```go
package main

import "context"

type MyPlugin struct{}

func (p *MyPlugin) Execute(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
    // process input...
    return map[string]interface{}{"result": "ok"}, nil
}

func (p *MyPlugin) GetActions() []string {
    return []string{"ingest", "transform"}
}

var Plugin MyPlugin
```

The worker uses `plugin.Lookup("Plugin")` which returns a `*MyPlugin`. Since the interface methods use pointer receivers, `*MyPlugin` satisfies the interface — this is why the export is a value type, not a pointer.

### actions/ Directory

If your plugin has multiple action handlers, place them in an `actions/` subdirectory. The worker's `flattenActionFiles()` moves all `.go` files from `actions/` into the root before compilation:

```
actions/
├── ingest.go    # func init() { register("ingest", handleIngest) }
└── transform.go # func init() { register("transform", handleTransform) }
```

### Installing a Plugin

```bash
curl -X POST http://localhost:8080/api/plugins \
  -H "Content-Type: application/json" \
  -d '{"repository_url": "https://github.com/your-org/my-plugin", "git_ref": "main"}'
```

The worker clones, compiles, and caches the `.so` automatically on next use.

---

## Part 2 — Storage Plugins

Storage plugins let you connect Mimir to any data backend by implementing a Go interface and hosting the plugin in a Git repository. The orchestrator clones, compiles (using `go build -buildmode=plugin`), and loads the `.so` at runtime — no restart required.

### StoragePlugin Interface

```go
// From pkg/models/storage.go
type StoragePlugin interface {
    Initialize(config map[string]interface{}) error
    Store(data []CIR) (*StorageResult, error)
    Retrieve(query CIRQuery) ([]CIR, error)
    Update(query CIRQuery, updates map[string]interface{}) (*StorageResult, error)
    Delete(query CIRQuery) (*StorageResult, error)
    HealthCheck() (bool, error)
    GetCapabilities() StorageCapabilities
    Close() error
}
```

### Repository Structure

```
my-storage-plugin/
├── plugin.go          # Required: exports var Plugin
├── go.mod             # For local development; DELETED by the loader before compilation
└── actions/           # Optional action files (flattened at compile time)
```

### Implementation Skeleton

```go
package main

import (
    "github.com/mimir-aip/mimir-aip-go/pkg/models"
)

type MyStoragePlugin struct {
    cfg map[string]interface{}
}

func (p *MyStoragePlugin) Initialize(config map[string]interface{}) error {
    p.cfg = config
    // open connection...
    return nil
}

func (p *MyStoragePlugin) Store(data []models.CIR) (*models.StorageResult, error) {
    // write records...
    return &models.StorageResult{RecordsAffected: int64(len(data))}, nil
}

func (p *MyStoragePlugin) Retrieve(query models.CIRQuery) ([]models.CIR, error) {
    // query records...
    return nil, nil
}

func (p *MyStoragePlugin) Update(query models.CIRQuery, updates map[string]interface{}) (*models.StorageResult, error) {
    return &models.StorageResult{}, nil
}

func (p *MyStoragePlugin) Delete(query models.CIRQuery) (*models.StorageResult, error) {
    return &models.StorageResult{}, nil
}

func (p *MyStoragePlugin) HealthCheck() (bool, error) { return true, nil }

func (p *MyStoragePlugin) GetCapabilities() models.StorageCapabilities {
    return models.StorageCapabilities{SupportsQuery: true}
}

func (p *MyStoragePlugin) Close() error { return nil }

// Plugin is the exported symbol the loader looks up via plugin.Lookup("Plugin").
var Plugin MyStoragePlugin
```

> **Note:** Go plugins cannot be unloaded from memory once opened. Uninstalling a storage plugin removes it from the registry and deletes the cached `.so`, but a full orchestrator restart is needed for complete removal.

### Installing a Storage Plugin

```bash
curl -X POST http://localhost:8080/api/storage-plugins \
  -H "Content-Type: application/json" \
  -d '{"repository_url": "https://github.com/your-org/my-storage-plugin", "git_ref": "v1.0.0"}'
```

The orchestrator clones the repository, compiles the `.so`, loads it, and registers it as a new storage backend type. The plugin name is derived from the repository name (last path segment, `.git` stripped, lowercased).

### Runtime Requirements

The orchestrator container must have the Go toolchain available at runtime (for `go build -buildmode=plugin`). The official Mimir orchestrator Docker image is based on `golang:1.25` for this reason.

---

## Further Reading

- [OpenAPI specification](./openapi.yaml) — all REST endpoints, including the plugin management APIs
- [Reference pipeline plugin](https://github.com/Mimir-AIP/OpenLibraryMimirPlugin) — a complete, production example
