# Custom Plugins for Mimir AIP

Mimir has four runtime extension surfaces:

1. **Pipeline step plugins** installed through `/api/plugins` and used by pipeline workers.
2. **ML provider plugins** declared in `plugin.yaml` and installed through `/api/plugins`; provider metadata is listed through `/api/ml-providers`.
3. **Storage plugins** installed through `/api/storage-plugins` and loaded by the orchestrator.
4. **LLM provider plugins** installed through `/api/llm/providers` and loaded by the orchestrator.

All four use the shared Go plugin runtime loader, but they do **not** share one cache directory. Cache paths are owned by the subsystem that loads the plugin. Go plugins are trusted in-process code: install only repositories you control or have audited.

---

## Runtime model and safety boundaries

The loader clones the repository, optionally checks out the requested ref, flattens `.go` files from `actions/` into the repository root, removes plugin-local `go.mod`/`go.sum`, compiles the package inside the Mimir host module with `go build -buildmode=plugin`, then opens the expected exported symbol with `plugin.Open`.

Important consequences:

- The orchestrator validates `/api/plugins` installs against the current host before metadata is saved. Workers still compile their own local artifact before task execution.
- Storage and LLM providers are compiled and opened by the orchestrator during install.
- The worker image must include the Go toolchain and CGO support because pipeline and ML provider plugins compile inside worker pods.
- Use immutable commit SHAs for production installs. Branch names such as `main` are convenient for development but are not repeatable.
- Go plugins cannot be unloaded from a running process. Deleting a plugin removes metadata/cache entries for future operations, but already opened symbols may remain until the worker/orchestrator process exits.
- Plugins run with the privileges of the process that loads them, including filesystem, network, and environment access.

---

## Pipeline step plugins

Pipeline step plugins add new actions that workers execute inside pipeline steps.

### Interface

```go
// pkg/pipeline/plugin.go
type Plugin interface {
    Execute(action string, params map[string]interface{}, ctx *models.PipelineContext) (map[string]interface{}, error)
}
```

### Repository structure

```text
my-plugin/
├── plugin.go       # Required: must export var Plugin
├── plugin.yaml     # Required manifest
├── go.mod          # For local dev; removed before host-module compilation
└── actions/        # Optional: .go files flattened to root at compile time
```

### plugin.yaml

`actions` must be structured action schemas, not a list of strings:

```yaml
name: my-plugin
version: "1.0.0"
description: "Does something useful"
author: "Your Name"
domains:
  - pipeline
actions:
  - name: ingest
    description: "Ingests records from a source"
    parameters:
      - name: url
        type: string
        required: true
        description: "Source URL"
    returns:
      - name: records_ingested
        type: number
        description: "Number of records ingested"
  - name: transform
    description: "Transforms records already in pipeline context"
```

### plugin.go

```go
package main

import (
    "fmt"

    "github.com/mimir-aip/mimir-aip-go/pkg/models"
)

type MyPlugin struct{}

func (p *MyPlugin) Execute(action string, params map[string]interface{}, ctx *models.PipelineContext) (map[string]interface{}, error) {
    switch action {
    case "ingest":
        return map[string]interface{}{"records_ingested": 42}, nil
    case "transform":
        return map[string]interface{}{"success": true}, nil
    default:
        return nil, fmt.Errorf("unknown action: %s", action)
    }
}

// Plugin is the symbol loaded with plugin.Lookup("Plugin").
var Plugin MyPlugin
```

### Installing

```bash
curl -X POST http://localhost:8080/api/plugins \
  -H "Content-Type: application/json" \
  -d '{"repository_url": "https://github.com/your-org/my-plugin", "git_ref": "<commit-sha>"}'
```

Use the manifest `name` in pipeline step definitions:

```yaml
steps:
  - name: ingest-custom
    plugin: my-plugin
    action: ingest
    parameters:
      url: https://example.com/data.json
```

---

## ML provider plugins

ML provider plugins add training/inference backends beyond the built-in tabular models. They are installed through `/api/plugins` because their metadata lives in `plugin.yaml`, then listed through `/api/ml-providers` for model creation.

### plugin.yaml

```yaml
name: acme-ml
version: "1.0.0"
description: "Acme ML provider"
author: "Acme"
domains:
  - ml
ml_provider:
  name: acme
  display_name: Acme ML
  description: External Acme training and inference provider
  supports_training: true
  supports_inference: true
  capabilities:
    - train
    - infer
  models:
    - name: acme-classifier
      display_name: Acme Classifier
      capabilities:
        - classify
```

### Runtime symbol

The compiled package must export `MLProvider`, satisfying `pkg/mlmodel.Provider`.

```go
package main

import (
	"github.com/mimir-aip/mimir-aip-go/pkg/mlmodel"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

type AcmeProvider struct{}

func (p *AcmeProvider) Metadata() models.MLProviderMetadata { /* ... */ }
func (p *AcmeProvider) ValidateModel(model *models.MLModel) error { return nil }
func (p *AcmeProvider) Train(req *mlmodel.ProviderTrainRequest) (*mlmodel.ProviderTrainResult, error) { /* ... */ }
func (p *AcmeProvider) Infer(req *mlmodel.ProviderInferRequest) (*mlmodel.ProviderInferResult, error) { /* ... */ }

var MLProvider AcmeProvider
```

Install with the same `/api/plugins` endpoint as pipeline plugins. The frontend lists installed providers in the **Plugins → ML Providers** tab and exposes providers when creating ML models.

---

## Storage plugins

Storage plugins connect Mimir to data backends. The orchestrator installs and reloads them through `/api/storage-plugins`.

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
```

### Repository structure

```text
my-storage-plugin/
├── plugin.go       # Required: must export var Plugin
├── go.mod          # For local dev; removed before host-module compilation
└── actions/        # Optional: flattened at compile time
```

Storage plugin names are currently derived from the repository URL's last path segment, not from a manifest. `version`, `description`, and `author` fields in storage plugin records are reserved and may be empty.

### Installing

```bash
curl -X POST http://localhost:8080/api/storage-plugins \
  -H "Content-Type: application/json" \
  -d '{"repository_url": "https://github.com/your-org/my-storage-plugin", "git_ref": "<commit-sha>"}'
```

After installation, the plugin name appears as a selectable `plugin_type` when creating storage configs in the frontend.

---

## LLM provider plugins

LLM provider plugins add language model backends beyond the built-in OpenRouter and OpenAI-compatible providers. The orchestrator installs them through `/api/llm/providers`.

### Interface

```go
// pkg/llm/provider.go
type Provider interface {
    Name() string
    ListModels(ctx context.Context) ([]Model, error)
    Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error)
}
```

### Runtime symbol

The package must export `Plugin` satisfying `pkg/llm.Provider`.

```go
package main

import (
    "context"

    "github.com/mimir-aip/mimir-aip-go/pkg/llm"
)

type MyProvider struct{}

func (p *MyProvider) Name() string { return "my-provider" }
func (p *MyProvider) ListModels(ctx context.Context) ([]llm.Model, error) { /* ... */ }
func (p *MyProvider) Complete(ctx context.Context, req llm.CompletionRequest) (llm.CompletionResponse, error) { /* ... */ }

var Plugin MyProvider
```

### Installing

```bash
curl -X POST http://localhost:8080/api/llm/providers \
  -H "Content-Type: application/json" \
  -d '{"repository_url": "https://github.com/your-org/my-llm-provider", "git_ref": "<commit-sha>"}'
```

The provider name is derived from the repository name. Activate an external provider through the LLM service configuration.

---

## REST summary

| Surface | Install/list endpoint | Runtime symbol | Name source |
|---|---|---|---|
| Pipeline action plugin | `/api/plugins` | `Plugin` | `plugin.yaml:name` |
| ML provider plugin | `/api/plugins`, listed via `/api/ml-providers` | `MLProvider` | `plugin.yaml:ml_provider.name` or `plugin.yaml:name` |
| Storage plugin | `/api/storage-plugins` | `Plugin` | Repository URL basename |
| LLM provider plugin | `/api/llm/providers` | `Plugin` | Repository URL basename |

---

## Operational recommendations

- Prefer commit SHAs over branches for production installs.
- Treat plugin repositories as privileged code. Review them like changes to the orchestrator or worker.
- Expect first use on a new worker pod to pay clone/build cost unless artifacts have been prewarmed.
- In distributed clusters, use rollout policies that avoid cold-start compile storms for the same plugin.
- Restart workers/orchestrators after uninstalling or replacing plugins when strict removal is required.

---

## Further reading

- [Building pipelines](./building-pipelines.md) — how to use pipeline step plugins in a pipeline definition
- [OpenAPI specification](./openapi.yaml) — all REST endpoints including plugin management APIs
