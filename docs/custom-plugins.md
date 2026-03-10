# Custom Plugins for Mimir AIP

Mimir supports three kinds of runtime-extensible plugins, all using the same pattern: implement a Go interface, host the code in a Git repository, and install via the REST API. Internally, all three now use the same shared runtime loader and cache path, so plugin authors get a consistent contract without Mimir hard-coding separate extension systems for each subsystem.

---

## Part 1 — Pipeline Step Plugins

Pipeline step plugins add new processing steps that workers execute inside pipelines. Workers clone, compile, and cache the `.so` on first use via the shared runtime loader.

### Interface

```go
// pkg/pipeline/plugin.go
type Plugin interface {
    Execute(action string, params map[string]interface{}, ctx *models.PipelineContext) (map[string]interface{}, error)
}
```

### Repository structure

```
my-plugin/
├── plugin.go       # Required: must export var Plugin
├── plugin.yaml     # Required: plugin manifest
├── go.mod          # For local dev; DELETED by the worker before compilation
└── actions/        # Optional: .go files here are flattened to root at compile time
    ├── ingest.go
    └── transform.go
```

> **Important:** The worker deletes your `go.mod` and `go.sum` before compilation. Your plugin is compiled as part of the host module, using its exact dependency versions. Keep `go.mod` only for IDE support.

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

### plugin.go

```go
package main

import (
    "context"
    "fmt"

    "github.com/mimir-aip/mimir-aip-go/pkg/models"
)

type MyPlugin struct{}

func (p *MyPlugin) Execute(action string, params map[string]interface{}, ctx *models.PipelineContext) (map[string]interface{}, error) {
    switch action {
    case "ingest":
        return p.ingest(params, ctx)
    case "transform":
        return p.transform(params, ctx)
    default:
        return nil, fmt.Errorf("unknown action: %s", action)
    }
}

func (p *MyPlugin) ingest(params map[string]interface{}, ctx *models.PipelineContext) (map[string]interface{}, error) {
    _ = context.Background() // use ctx for timeouts etc.
    // ... your logic
    return map[string]interface{}{"records_ingested": 42}, nil
}

func (p *MyPlugin) transform(params map[string]interface{}, ctx *models.PipelineContext) (map[string]interface{}, error) {
    // ... your logic
    return map[string]interface{}{"success": true}, nil
}

// Plugin is the symbol the worker loads via plugin.Lookup("Plugin").
var Plugin MyPlugin
```

The worker calls `plugin.Lookup("Plugin")` which returns `*MyPlugin`. Methods with pointer receivers on `*MyPlugin` satisfy the interface — this is why the export is a value type, not a pointer.

### Installing

```bash
curl -X POST http://localhost:8080/api/plugins \
  -H "Content-Type: application/json" \
  -d '{"repository_url": "https://github.com/your-org/my-plugin", "git_ref": "main"}'
```

The worker clones, compiles, and caches the `.so` automatically on next use via the shared runtime loader. For pipeline plugins, the install-time name comes from `plugin.yaml` (`name:`), so choose a stable manifest name and use that same name in pipeline step definitions.

---

## Part 2 — Storage Plugins

Storage plugins connect Mimir to any data backend. The orchestrator installs and reloads these plugins through the shared runtime loader at startup and on `POST /api/storage-plugins`.

### Interface

```go
// pkg/models/storage.go
type StoragePlugin interface {
    Initialize(config *PluginConfig) error
    CreateSchema(ontology *OntologyDefinition) error
    Store(cir *CIR) (*StorageResult, error)
    Retrieve(query *CIRQuery) ([]*CIR, error)
    Update(query *CIRQuery, updates *CIRUpdate) (*StorageResult, error)
    Delete(query *CIRQuery) (*StorageResult, error)
    GetMetadata() (*StorageMetadata, error)
    HealthCheck() (bool, error)
}

type PluginConfig struct {
    ConnectionString string                 `json:"connection_string"`
    Credentials      map[string]interface{} `json:"credentials,omitempty"`
    Options          map[string]interface{} `json:"options,omitempty"`
}
```

### Repository structure

```
my-storage-plugin/
├── plugin.go       # Required: must export var Plugin
├── go.mod          # For local dev; DELETED by the loader before compilation
└── actions/        # Optional: flattened at compile time
```

### plugin.go

```go
package main

import "github.com/mimir-aip/mimir-aip-go/pkg/models"

type MyStoragePlugin struct {
    cfg *models.PluginConfig
}

func (p *MyStoragePlugin) Initialize(config *models.PluginConfig) error {
    p.cfg = config
    // open connection using config.ConnectionString / config.Credentials
    return nil
}

func (p *MyStoragePlugin) CreateSchema(ontology *models.OntologyDefinition) error {
    // create tables / indexes / collections to match the ontology
    return nil
}

func (p *MyStoragePlugin) Store(cir *models.CIR) (*models.StorageResult, error) {
    // write the CIR record
    return &models.StorageResult{Success: true, AffectedItems: 1}, nil
}

func (p *MyStoragePlugin) Retrieve(query *models.CIRQuery) ([]*models.CIR, error) {
    // query and return CIR records
    return nil, nil
}

func (p *MyStoragePlugin) Update(query *models.CIRQuery, updates *models.CIRUpdate) (*models.StorageResult, error) {
    return &models.StorageResult{Success: true, AffectedItems: 0}, nil
}

func (p *MyStoragePlugin) Delete(query *models.CIRQuery) (*models.StorageResult, error) {
    return &models.StorageResult{Success: true, AffectedItems: 0}, nil
}

func (p *MyStoragePlugin) GetMetadata() (*models.StorageMetadata, error) {
    return &models.StorageMetadata{StorageType: "my-backend"}, nil
}

func (p *MyStoragePlugin) HealthCheck() (bool, error) {
    // ping the backend
    return true, nil
}

// Plugin is the symbol the orchestrator loads via plugin.Lookup("Plugin").
var Plugin MyStoragePlugin
```

### Installing

```bash
curl -X POST http://localhost:8080/api/storage-plugins \
  -H "Content-Type: application/json" \
  -d '{"repository_url": "https://github.com/your-org/my-storage-plugin", "git_ref": "v1.0.0"}'
```

The orchestrator clones, compiles, caches, and loads the `.so` through the shared runtime loader, then registers it as a new storage backend type. From that point it is available as a `plugin_type` when creating storage configs.

### Managing storage plugins

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/storage-plugins` | List installed external plugins |
| `POST` | `/api/storage-plugins` | Install from Git repo |
| `GET` | `/api/storage-plugins/{name}` | Get plugin metadata |
| `DELETE` | `/api/storage-plugins/{name}` | Uninstall (204) |

> **Note:** Go plugins cannot be unloaded from memory once opened. Uninstalling removes the registry entry and cached `.so`; a full orchestrator restart is required for complete removal.

---

## Part 3 — LLM Provider Plugins

LLM provider plugins add new language model back-ends (beyond the built-in OpenRouter and OpenAI-compatible providers). The orchestrator installs them through the same shared runtime loader used by the other plugin categories.

### Interface

```go
// pkg/llm/provider.go
type Provider interface {
    Name() string
    ListModels(ctx context.Context) ([]Model, error)
    Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error)
}

type CompletionRequest struct {
    Model       string
    Messages    []Message  // {Role: "system"|"user"|"assistant", Content: string}
    MaxTokens   int
    Temperature float64
}

type CompletionResponse struct {
    Content      string
    Model        string
    InputTokens  int
    OutputTokens int
}

type Model struct {
    ID            string `json:"id"`
    Name          string `json:"name"`
    ContextLength int    `json:"context_length"`
    IsFree        bool   `json:"is_free"`
    ProviderName  string `json:"provider_name"`
}
```

### Repository structure

```
my-llm-provider/
├── plugin.go       # Required: must export var Plugin
├── go.mod          # For local dev; DELETED by the loader before compilation
└── actions/        # Optional: flattened at compile time
```

### plugin.go

```go
package main

import (
    "context"
    "fmt"

    "github.com/mimir-aip/mimir-aip-go/pkg/llm"
)

type MyProvider struct{}

func (p *MyProvider) Name() string { return "my-provider" }

func (p *MyProvider) ListModels(ctx context.Context) ([]llm.Model, error) {
    return []llm.Model{
        {ID: "my-model-v1", Name: "My Model v1", ContextLength: 8192},
    }, nil
}

func (p *MyProvider) Complete(ctx context.Context, req llm.CompletionRequest) (llm.CompletionResponse, error) {
    // call your LLM API here
    _ = req
    return llm.CompletionResponse{
        Content: "response text",
        Model:   req.Model,
    }, fmt.Errorf("not implemented")
}

// Plugin is the symbol the orchestrator loads via plugin.Lookup("Plugin").
var Plugin MyProvider
```

### Installing

```bash
curl -X POST http://localhost:8080/api/llm/providers \
  -H "Content-Type: application/json" \
  -d '{"repository_url": "https://github.com/your-org/my-llm-provider", "git_ref": "main"}'
```

Returns `201` with the provider record on success, or `422` with an error body if compilation fails. The provider name is derived from the repository name.

Once installed, activate it by setting `LLM_PROVIDER=<name>` in the orchestrator environment and restarting — or call `SetActiveProvider` programmatically.

### Managing LLM providers

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/llm/providers` | List installed external providers |
| `POST` | `/api/llm/providers` | Install from Git repo |
| `GET` | `/api/llm/providers/{name}` | Get provider metadata |
| `DELETE` | `/api/llm/providers/{name}` | Uninstall (204) |

### Built-in providers

Two providers are always registered without installation:

| Name | Env vars | Notes |
|------|----------|-------|
| `openrouter` | `LLM_API_KEY` | Access to hundreds of models; set `LLM_MODEL=openrouter/free` for free tier |
| `openai_compat` | `LLM_API_KEY`, `LLM_BASE_URL` | Any OpenAI-compatible endpoint (OpenAI, Ollama, vLLM, etc.) |

---

## Runtime requirements

All three plugin types require the Go toolchain to be available inside the orchestrator/worker container at runtime (for `go build -buildmode=plugin`). The official Mimir Docker images are based on a Go builder image for this reason.

---

## Further reading

- [Building pipelines](./building-pipelines.md) — how to use pipeline step plugins in a pipeline definition
- [OpenAPI specification](./openapi.yaml) — all REST endpoints including plugin management APIs
- [Reference pipeline plugin](https://github.com/Mimir-AIP/OpenLibraryMimirPlugin) — a complete, production example
