# Mimir AIP

Mimir AIP is an ontology-driven platform for data aggregation, processing, and analysis. It provides a unified runtime for ingestion pipelines, machine learning model training and inference, digital twin management, guided or advanced project onboarding, and always-on project analysis — all backed by a persistent metadata store and exposed as [Model Context Protocol (MCP)](https://modelcontextprotocol.io) tools for direct use by AI agents and LLM-based workflows. Mimir AIP is built in Go for performance and ease of deployment, with a React-based web frontend for user-friendly management. It runs on Kubernetes, supports a wide range of storage backends, and is extensible through runtime-loaded pipeline plugins, storage plugins, and LLM providers. The platform is designed to stay use-case agnostic: bundled connectors, review queues, and insight generation all compile down to the same core project resources rather than hard-coding one domain-specific workflow.

---

## Contents

- [Architecture](#architecture)
- [Terminology](#terminology)
- [Quick Start](#quick-start)
  - [Docker Compose](#docker-compose)
  - [Kubernetes with Helm](#kubernetes-with-helm)
- [MCP Integration](#mcp-integration)
- [Configuration Reference](#configuration-reference)
- [Building from Source](#building-from-source)

---

## Architecture

Mimir AIP consists of two binaries and an optional web frontend:

```
┌──────────────────────────────────────────────────────┐
│                      Client Layer                    │
│   Web Frontend (port 3000)   │   MCP Client / Agent  │
└──────────────┬───────────────┴──────────┬────────────┘
               │  REST API                │  SSE (MCP)
               ▼                          ▼
┌─────────────────────────────────────────────────────┐
│                    Orchestrator                     │
│  ┌──────────┐  ┌────────────┐  ┌──────────────────┐  │
│  │ Projects │  │ Pipelines  │  │   Ontologies     │  │
│  │ Storage  │  │ Schedules  │  │   ML Models      │  │
│  │Onboarding│  │ Connectors │  │   Digital Twins  │  │
│  │ Analysis │  │   Queue    │  │ Insights / MCP   │  │
│  └──────────┘  └────┬───────┘  └──────────────────┘  │
└─────────────────────┼───────────────────────────────┘
                      │  Kubernetes Jobs
                      ▼
           ┌─────────────────────┐
           │       Workers       │
           │  (pipeline, train,  │
           │   infer, DT sync)   │
           └──────────┬──────────┘
                      │
          ┌───────────▼───────────┐
          │   Storage Backends    │
          │  Filesystem · Postgres│
          │  MySQL · MongoDB · S3 │
          │  Redis · ES · Neo4j   │
          └───────────────────────┘
```

**Orchestrator** — the long-running HTTP server. Manages persistent metadata in SQLite for projects, pipelines, schedules, ontologies, ML models, digital twins, storage configurations, analysis runs, review items, and insights. Exposes the REST API and MCP SSE endpoint. Also hosts the generic connector catalog, materializes bundled connectors into ordinary pipelines and schedules, and runs lightweight recurring control-plane work such as daily project insight generation. Heavy execution (pipeline runs, ML training, inference, digital twin synchronisation) is still delegated to **Workers** as Kubernetes Jobs.

**Worker** — a short-lived binary run as a Kubernetes Job. Reads its task type and parameters from environment variables, calls the orchestrator API to fetch configuration, executes the work, and reports results back. Designed with scalability in mind, supporting concurrent workers across multiple Kubernetes clusters — the orchestrator dispatches jobs to a configurable cluster pool, spilling over from the primary cluster to remote or cloud clusters when capacity is reached.

**Frontend** — a lightweight React single-page application served by a small Go HTTP server. Communicates exclusively with the orchestrator REST API and now exposes both guided and advanced onboarding flows, bundled connector setup, and project-level insights and review surfaces.

---

## Terminology

| Term | Description |
|------|-------------|
| **Project** | Top-level organisational unit. Groups pipelines, ontologies, ML models, digital twins, and storage configurations, and stores project-wide preferences such as onboarding mode. |
| **Onboarding Mode** | Project preference that selects either a guided setup flow (connector-driven) or the full advanced manual workflow. |
| **Storage Config** | A connection definition for a storage backend (filesystem, PostgreSQL, MySQL, MongoDB, S3, Redis, Elasticsearch, or Neo4j). Data is stored and retrieved using the **CIR** (Common Internal Representation) format. |
| **Connector Template** | A bundled, metadata-driven ingestion template that materialises into an ordinary pipeline plus an optional schedule. |
| **Pipeline** | A named, ordered sequence of processing steps (ingestion → processing → output). Pipelines are executed asynchronously by workers. |
| **Schedule** | A cron-based trigger that enqueues one or more pipelines on a recurring basis. |
| **Ontology** | An OWL/Turtle vocabulary that defines the entity types, properties, and relationships for a project domain. Used to structure storage and constrain ML model training. |
| **CIR** | Common Internal Representation — the normalised record format used across all storage backends. Each CIR contains a `source` block (provenance), a `data` block (the payload), and a `metadata` block. |
| **Insight** | A persisted autonomous finding, such as an anomaly spike, trend break, or co-occurrence surge, generated from project storage data. |
| **Review Item** | A persisted reviewable finding — currently used for cross-source link decisions — whose accepted or rejected outcome improves future scoring. |
| **ML Model** | A model definition (type: decision tree, random forest, regression, or neural network) linked to an ontology. Training and inference are executed by workers. |
| **Digital Twin** | A live in-memory graph of entities and their attributes, initialised from an ontology and synchronised from storage. Queryable via a built-in SPARQL engine. |
| **MCP** | [Model Context Protocol](https://modelcontextprotocol.io) — an open standard for exposing tools to AI agents. Mimir exposes MCP tools across the platform's core resources so agent workflows can configure and operate projects directly. |

---

## Typical project workflow

1. Create or select a **Project** and choose either **guided** or **advanced** onboarding.
2. Define a **Storage Config** for where normalised CIR data should land.
3. Create a **Pipeline** manually, or materialise one from a bundled **Connector Template**; add a **Schedule** if the source should run incrementally.
4. Generate or refine an **Ontology** from stored data and cross-source extraction results.
5. Train **ML Models** and initialise or sync **Digital Twins** against the ontology and storage-backed project data.
6. Use the **Insights & Review** surfaces to monitor ingestion behaviour, inspect autonomous findings, and calibrate cross-source link confidence over time.

---


## Quick Start

### Docker Compose

The simplest way to run Mimir AIP locally. Docker Compose starts the orchestrator and frontend; worker jobs are not available without Kubernetes.

**Prerequisites:** Docker and Docker Compose.

```bash
git clone https://github.com/mimir-aip/mimir-aip-go
cd mimir-aip-go

docker compose up --build
```

| Service | URL |
|---------|-----|
| Orchestrator API | http://localhost:8080 |
| Web Frontend | http://localhost:3000 |
| MCP SSE endpoint | http://localhost:8080/mcp/sse |

To stop:

```bash
docker compose down
```

---

### Kubernetes with Helm

For a full deployment including worker job execution.

**Prerequisites:** A running Kubernetes cluster (1.25+), `kubectl` configured, and [Helm 3](https://helm.sh/docs/intro/install/).

Images are published to the GitHub Container Registry and are public — no registry credentials or image build step required.

#### 1. Install the Helm chart

```bash
helm install mimir-aip ./helm/mimir-aip \
  --namespace mimir-aip \
  --create-namespace
```

The chart defaults to `ghcr.io/mimir-aip` and the latest published images. To pin a specific version:

```bash
helm install mimir-aip ./helm/mimir-aip \
  --namespace mimir-aip \
  --create-namespace \
  --set image.tag=0.1.1
```

The chart uses your cluster's default storage class for the orchestrator PVC. To override:

```bash
helm install mimir-aip ./helm/mimir-aip \
  --namespace mimir-aip \
  --create-namespace \
  --set orchestrator.persistence.storageClass=standard
```

#### 2. Access the services

```bash
# Orchestrator API
kubectl port-forward -n mimir-aip svc/mimir-aip-orchestrator 8080:8080

# Web Frontend
kubectl port-forward -n mimir-aip svc/mimir-aip-frontend 3000:80
```

#### 3. Upgrade and uninstall

```bash
# Upgrade to a new release
helm upgrade mimir-aip ./helm/mimir-aip --namespace mimir-aip --set image.tag=0.2.0

# Uninstall (PVC is retained by default)
helm uninstall mimir-aip --namespace mimir-aip
```

#### Custom values

Create a `my-values.yaml` to override defaults without modifying the chart:

```yaml
image:
  tag: 0.1.1          # pin to a specific release

orchestrator:
  logLevel: debug
  maxWorkers: 20
  persistence:
    size: 50Gi
    storageClass: fast-ssd

frontend:
  serviceType: ClusterIP   # use ClusterIP + Ingress instead of LoadBalancer
```

```bash
helm install mimir-aip ./helm/mimir-aip --namespace mimir-aip --create-namespace -f my-values.yaml
```

#### Building custom images

If you need to modify the source and publish your own images:

```bash
export REGISTRY=ghcr.io/your-org

make build-all REGISTRY=$REGISTRY
make push-all  REGISTRY=$REGISTRY

helm install mimir-aip ./helm/mimir-aip \
  --namespace mimir-aip \
  --create-namespace \
  --set image.registry=$REGISTRY
```

---

## MCP Integration

Mimir AIP exposes 64 MCP tools over a Server-Sent Events (SSE) transport at `/mcp/sse`. Any MCP-compatible client can connect.

### Claude Code

Add the following to your Claude Code MCP configuration (`~/.claude/mcp_servers.json` or via `claude mcp add`):

```json
{
  "mcpServers": {
    "mimir": {
      "type": "sse",
      "url": "http://localhost:8080/mcp/sse"
    }
  }
}
```

Then start a Claude Code session — the full Mimir toolset will be available automatically.

### Tool categories

| Category | Tools | Description |
|----------|-------|-------------|
| Projects | 8 | CRUD, clone, component associations |
| Pipelines | 6 | CRUD, execute |
| Schedules | 5 | CRUD |
| Connectors | 2 | List bundled templates, materialize pipelines/schedules |
| Analysis | 4 | Run resolver analysis, inspect metrics, list reviews, decide findings |
| Insights | 2 | List and generate autonomous insights |
| ML Models | 8 | CRUD, train, infer, recommend, monitor |
| Digital Twins | 7 | CRUD, sync, SPARQL query |
| Ontologies | 8 | CRUD, generate from text, extract from storage, inspect ontology text |
| Storage | 10 | Config CRUD, store/retrieve/update/delete data, health and ingestion health |
| Tasks | 3 | List, get, cancel work tasks |
| System | 1 | Platform health |

---

## Configuration Reference

All orchestrator configuration is supplied via environment variables (set in `docker-compose.yaml` or the Helm ConfigMap).

| Variable | Default | Description |
|----------|---------|-------------|
| `ENVIRONMENT` | `production` | Runtime label (`production` or `development`) |
| `LOG_LEVEL` | `info` | Log verbosity (`debug`, `info`, `warn`, `error`) |
| `PORT` | `8080` | Orchestrator HTTP port |
| `STORAGE_DIR` | `/app/data` | Directory for the SQLite database and file storage |
| `MIN_WORKERS` | `1` | Minimum concurrent worker jobs |
| `MAX_WORKERS` | `10` | Maximum concurrent worker jobs |
| `QUEUE_THRESHOLD` | `5` | Queued tasks before spinning up an additional worker |
| `WORKER_NAMESPACE` | _(release namespace)_ | Kubernetes namespace workers are spawned into |
| `WORKER_SERVICE_ACCOUNT` | `mimir-worker` | Service account assigned to worker jobs |

---

## Building from Source

**Prerequisites:** Go 1.21+, Docker (for container builds).

```bash
# Run unit tests
make test

# Build all binaries (native, no Docker)
go build ./cmd/orchestrator
go build ./cmd/worker

# Build Docker images
make build-all

# Run the orchestrator locally against a local SQLite database
make dev-orchestrator
```

## Citation
If you use Mimir AIP in your research, please consider citing:

```
@software{mimir-aip,
  author = {Ciaran McAleer},
  title = {Mimir AIP: An Ontology-Driven Platform for Data Aggregation, Processing, and Analysis},
  year = {2026},
  GitHub repository: \url{"https://github.com/Mimir-AIP/Mimir-AIP-Go"}
}
```

---

**Project Repository**: https://github.com/Mimir-AIP/Mimir-AIP-Go

**Documentation**: https://mimir-aip.github.io/wiki/

**Issues & Support**: https://github.com/Mimir-AIP/Mimir-AIP-Go/issues