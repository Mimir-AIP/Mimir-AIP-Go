# Mimir AIP Distributed Architecture

## Overview

This document describes the distributed container architecture for Mimir AIP, which separates concerns into dedicated containers for better scalability and maintainability.

## Architecture

The distributed architecture consists of four main components:

```
┌─────────────────┐
│   Frontend      │ ← Next.js standalone (Port 3000)
│   Container     │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│   Backend       │ ← Go API + Jena Fuseki (Port 8080, 3030)
│   Container     │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│   Redis         │ ← Job Queue & Caching (Port 6379)
│   Container     │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│   Worker        │ ← Scalable Pipeline Executors
│   Container(s)  │    (Can scale to N instances)
└─────────────────┘
```

### Components

#### 1. Frontend Container
- **Image**: `mimir-aip:frontend-latest`
- **Technology**: Next.js 14+ (standalone mode)
- **Port**: 3000
- **Purpose**: Serves the web UI
- **Resource**: ~256MB RAM, 0.5 CPU

#### 2. Backend Container
- **Image**: `mimir-aip:backend-latest`
- **Technology**: Go API server + Apache Jena Fuseki
- **Ports**: 8080 (API), 3030 (Fuseki - internal)
- **Purpose**: REST API server, ontology support, job management
- **Resource**: ~1GB RAM, 1 CPU

#### 3. Redis Container
- **Image**: `redis:7-alpine`
- **Port**: 6379
- **Purpose**: Job queue and caching
- **Resource**: Minimal

#### 4. Worker Container(s)
- **Image**: `mimir-aip:worker-latest`
- **Technology**: Go worker process
- **Purpose**: Execute pipelines and digital twin jobs
- **Scalability**: Can be scaled to N instances
- **Resource**: ~512MB RAM, 0.5 CPU per instance

## Quick Start

### Build All Images

```bash
./build-distributed.sh
```

### Start the System

Start with default 2 workers:
```bash
./start-distributed.sh up
```

Start with custom number of workers:
```bash
WORKERS=5 ./start-distributed.sh up
```

Or use docker-compose directly:
```bash
docker-compose -f docker-compose.distributed.yml up -d
```

### Scale Workers

Scale workers dynamically:
```bash
./start-distributed.sh scale 5
```

Or:
```bash
docker-compose -f docker-compose.distributed.yml up -d --scale worker=5
```

### View Logs

```bash
./start-distributed.sh logs
```

Or view specific service logs:
```bash
docker-compose -f docker-compose.distributed.yml logs -f worker
```

### Stop the System

```bash
./start-distributed.sh down
```

## API Usage

### Synchronous Execution (Direct)

Execute a pipeline synchronously (blocks until completion):

```bash
curl -X POST http://localhost:8080/api/v1/pipelines/execute \
  -H "Content-Type: application/json" \
  -d '{
    "pipeline_file": "test_pipeline.yaml",
    "context": {"input": "test data"}
  }'
```

### Asynchronous Execution (Worker Queue)

Enqueue a pipeline for async execution by workers:

```bash
# Enqueue job
curl -X POST http://localhost:8080/api/v1/queue/pipelines/enqueue \
  -H "Content-Type: application/json" \
  -d '{
    "pipeline_file": "test_pipeline.yaml",
    "context": {"input": "test data"}
  }'
```

Response:
```json
{
  "message": "Pipeline job enqueued successfully",
  "job_id": "abc123...",
  "status": "queued"
}
```

Check job status:
```bash
curl http://localhost:8080/api/v1/queue/jobs/{job_id}
```

Wait for job completion (with timeout):
```bash
curl "http://localhost:8080/api/v1/queue/jobs/{job_id}/wait?timeout=60s"
```

### Queue Status

Check current queue length:
```bash
curl http://localhost:8080/api/v1/queue/status
```

## Configuration

### Environment Variables

#### Backend
- `MIMIR_PORT`: API server port (default: 8080)
- `REDIS_URL`: Redis connection URL (default: redis:6379)
- `FUSEKI_URL`: Fuseki server URL (default: http://localhost:3030)
- `MIMIR_LOG_LEVEL`: Log level (default: INFO)

#### Frontend
- `PORT`: Frontend port (default: 3000)
- `NEXT_PUBLIC_API_URL`: Backend API URL (default: http://localhost:8080)

#### Worker
- `REDIS_URL`: Redis connection URL (default: redis:6379)
- `WORKER_CONCURRENCY`: Number of concurrent jobs per worker (default: 5)
- `FUSEKI_URL`: Fuseki server URL for workers that need ontology access

## Scaling Strategy

### Horizontal Scaling

Workers can be scaled horizontally to handle more load:

```bash
# Scale to 10 workers
docker-compose -f docker-compose.distributed.yml up -d --scale worker=10
```

### When to Scale

- **High Queue Length**: If queue status shows many pending jobs
- **High CPU Usage**: If workers are consistently at high CPU
- **Long Job Duration**: If jobs are taking longer to complete

### Monitoring

Monitor worker performance:
```bash
docker stats
```

Check queue status periodically:
```bash
watch -n 2 'curl -s http://localhost:8080/api/v1/queue/status'
```

## Volumes

### Persistent Data

- `backend_data`: Backend application data (SQLite DB, ontologies, TDB2)
- `backend_logs`: Backend logs
- `backend_plugins`: Plugin files
- `backend_pipelines`: Pipeline definitions
- `worker_data`: Worker data (shared across workers)
- `worker_logs`: Worker logs
- `redis_data`: Redis persistence

### Backup

To backup data:
```bash
docker run --rm -v mimir-aip_backend_data:/data -v $(pwd)/backup:/backup alpine tar czf /backup/backend-data.tar.gz /data
```

## Troubleshooting

### Workers Not Processing Jobs

1. Check Redis connection:
```bash
docker-compose -f docker-compose.distributed.yml logs redis
```

2. Check worker logs:
```bash
docker-compose -f docker-compose.distributed.yml logs worker
```

3. Verify Redis connectivity from backend:
```bash
docker exec mimir-backend redis-cli -h redis ping
```

### High Queue Backlog

Scale up workers:
```bash
./start-distributed.sh scale 10
```

### Out of Memory

Adjust resource limits in `docker-compose.distributed.yml`:
```yaml
deploy:
  resources:
    limits:
      memory: 2G
```

## Comparison with Unified Architecture

| Feature | Unified | Distributed |
|---------|---------|-------------|
| Containers | 1 | 4+ |
| Scalability | Limited | Horizontal |
| Resource Usage | ~2GB RAM | ~2.5GB+ (scales with workers) |
| Complexity | Low | Medium |
| Maintenance | Easier | More control |
| Job Processing | Synchronous | Async + Sync |
| Failure Isolation | None | Per-service |

## Migration from Unified

The distributed architecture is a new deployment option alongside the unified container. Both architectures are supported:

- **Unified**: `docker-compose.unified.yml` - Single container, simpler deployment
- **Distributed**: `docker-compose.distributed.yml` - Multiple containers, better scalability

Choose based on your needs:
- Small deployments: Use unified
- Production/scalable: Use distributed

## Development

### Building Individual Containers

```bash
# Backend
docker build -f Dockerfile.backend -t mimir-aip:backend-latest .

# Frontend
docker build -f Dockerfile.frontend -t mimir-aip:frontend-latest .

# Worker
docker build -f Dockerfile.worker -t mimir-aip:worker-latest .
```

### Testing Workers Locally

Start Redis:
```bash
docker run -d -p 6379:6379 redis:7-alpine
```

Run worker locally:
```bash
export REDIS_URL=localhost:6379
go run ./cmd/worker/main.go
```

## Architecture Benefits

1. **Scalability**: Scale workers independently based on load
2. **Resilience**: Worker failures don't affect the API server
3. **Resource Optimization**: Allocate resources per component
4. **Development**: Easier to develop and test individual components
5. **Deployment**: Can deploy workers in different regions/environments
6. **Performance**: Async job processing doesn't block API requests

## Future Enhancements

- [ ] Worker health monitoring dashboard
- [ ] Auto-scaling based on queue length
- [ ] Worker-specific job types (e.g., CPU-intensive, GPU workers)
- [ ] Job priority queue
- [ ] Dead letter queue for failed jobs
- [ ] Distributed tracing across services
- [ ] Metrics collection (Prometheus/Grafana)
