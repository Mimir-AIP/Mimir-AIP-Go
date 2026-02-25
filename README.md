# Mimir AIP

Mimir AIP is an ontology-driven platform for data aggregation, processing and analysis. It aims to provide a unified runtime for data ingestion pipelines, machine learning model training and inference, and digital twin management — all backed by a persistent metadata store and exposed as [Model Context Protocol (MCP)](https://modelcontextprotocol.io) tools for direct use by AI agents and LLM-based workflows, you are free to use the platform directly or via your favourite agent tooling. Mimir AIP is built in Go for performance and ease of deployment, with a React/TypeScript frontend for user-friendly management. It runs on Kubernetes and supports a wide range of storage backends, additionally it is an extensible system, making it easy to design and build custom plugins for new data sources, ML model types, or processing steps. Mimir AIP aims to offer an accessible yet powerful solution targetting small and medium sized enterprises looking to leverage their data and derive insights without the overhead of building a custom platform from scratch, or relying on locked-in SaaS solutions. 

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
│  ┌──────────┐  ┌──────────┐  ┌──────────────────┐  │
│  │ Projects │  │Pipelines │  │   ML Models      │  │
│  │ Ontology │  │Schedules │  │   Digital Twins  │  │
│  │ Storage  │  │  Queue   │  │   MCP Server     │  │
│  └──────────┘  └────┬─────┘  └──────────────────┘  │
│          SQLite     │                               │
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

**Orchestrator** — the long-running HTTP server. Manages all persistent metadata (projects, pipelines, ontologies, ML models, digital twins, storage configurations, schedules) in SQLite. Exposes a REST API and an MCP SSE endpoint. Spawns **Workers** as Kubernetes Jobs when pipeline execution, ML training, ML inference, or digital twin synchronisation is required.

**Worker** — a short-lived binary run as a Kubernetes Job. Reads its task type and parameters from environment variables, calls the orchestrator API to fetch configuration, executes the work, and reports results back. Designed with scalability in mind, allowing multiple workers to run concurrently (currently workers run on the same cluster as the orchestrator, however in future releases we plan to support remote workers running in different clusters or on cloud compute services)

**Frontend** — a lightweight React/TypeScript single-page application served by a small Go HTTP server. Communicates exclusively with the orchestrator REST API.

---

## Terminology

| Term | Description |
|------|-------------|
| **Project** | Top-level organisational unit. Groups pipelines, ontologies, ML models, digital twins, and storage configurations. |
| **Pipeline** | A named, ordered sequence of processing steps (ingestion → processing → output). Pipelines are executed asynchronously by workers. |
| **Schedule** | A cron-based trigger that enqueues one or more pipelines on a recurring basis. |
| **Ontology** | An OWL/Turtle vocabulary that defines the entity types, properties, and relationships for a project domain. Used to structure storage and constrain ML model training. |
| **Storage Config** | A connection definition for a storage backend (filesystem, PostgreSQL, MySQL, MongoDB, S3, Redis, Elasticsearch, or Neo4j). Data is stored and retrieved using the **CIR** (Common Internal Representation) format. |
| **CIR** | Common Internal Representation — the normalised record format used across all storage backends. Each CIR contains a `source` block (provenance), a `data` block (the payload), and a `metadata` block. |
| **ML Model** | A model definition (type: decision tree, random forest, regression, or neural network) linked to an ontology. Training and inference are executed by workers. |
| **Digital Twin** | A live in-memory graph of entities and their attributes, initialised from an ontology and synchronised from storage. Queryable via a built-in SPARQL engine. |
| **MCP** | [Model Context Protocol](https://modelcontextprotocol.io) — an open standard for exposing tools to AI agents. Mimir exposes 55 tools covering all platform resources, allowing users to interact with the system within the enviroment of their favourite tools and leverage natural language to configure and operate Mimir AIP. |

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

#### 1. Build and push images

```bash
# Replace with your registry, e.g. ghcr.io/your-org
export REGISTRY=ghcr.io/your-org

make build-all REGISTRY=$REGISTRY
make push-all  REGISTRY=$REGISTRY
```

#### 2. Install the Helm chart

```bash
helm install mimir-aip ./helm/mimir-aip \
  --namespace mimir-aip \
  --create-namespace \
  --set image.registry=$REGISTRY
```

The chart uses your cluster's default storage class for the orchestrator PVC. To override:

```bash
helm install mimir-aip ./helm/mimir-aip \
  --namespace mimir-aip \
  --create-namespace \
  --set image.registry=$REGISTRY \
  --set orchestrator.persistence.storageClass=standard
```

#### 3. Access the services

```bash
# Orchestrator API
kubectl port-forward -n mimir-aip svc/mimir-aip-orchestrator 8080:8080

# Web Frontend
kubectl port-forward -n mimir-aip svc/mimir-aip-frontend 3000:80
```

#### 4. Upgrade and uninstall

```bash
# Upgrade after pushing new images
helm upgrade mimir-aip ./helm/mimir-aip --namespace mimir-aip --set image.registry=$REGISTRY

# Uninstall (PVC is retained by default)
helm uninstall mimir-aip --namespace mimir-aip
```

#### Custom values

Create a `my-values.yaml` to override defaults without modifying the chart:

```yaml
image:
  registry: ghcr.io/your-org
  tag: v1.2.0

orchestrator:
  logLevel: debug
  maxWorkers: 20
  persistence:
    size: 50Gi
    storageClass: fast-ssd

frontend:
  serviceType: ClusterIP   # Use ClusterIP + Ingress instead of LoadBalancer
```

```bash
helm install mimir-aip ./helm/mimir-aip --namespace mimir-aip --create-namespace -f my-values.yaml
```

---

## MCP Integration

Mimir AIP exposes 55 MCP tools over a Server-Sent Events (SSE) transport at `/mcp/sse`. Any MCP-compatible client can connect.

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
| ML Models | 7 | CRUD, train, infer, recommend |
| Digital Twins | 7 | CRUD, sync, SPARQL query |
| Ontologies | 6 | CRUD, generate from text, extract from storage |
| Storage | 8 | Config CRUD, store/retrieve/update/delete data, health check |
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
