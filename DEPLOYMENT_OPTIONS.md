# Deployment Options for Mimir AIP

Mimir AIP offers two deployment architectures to suit different requirements:

## 1. Unified Architecture (Simple)

**Best for:** Development, testing, small deployments, quick setup

**Characteristics:**
- Single Docker container
- Frontend + Backend + Jena Fuseki in one
- ~244MB image size
- Simple to deploy and manage
- Resource efficient for small workloads

**Quick Start:**
```bash
./build-unified.sh
docker-compose -f docker-compose.unified.yml up -d
```

**Access:**
- Frontend & API: http://localhost:8080

**Documentation:** See main README.md

---

## 2. Distributed Architecture (Scalable)

**Best for:** Production, high-load scenarios, horizontal scaling

**Characteristics:**
- 4 separate containers: Frontend, Backend, Redis, Workers
- Horizontally scalable workers
- Async job processing via Redis queue
- Better failure isolation
- Resource optimization per component

**Quick Start:**
```bash
./build-distributed.sh
./start-distributed.sh up
```

**Scale workers:**
```bash
./start-distributed.sh scale 5
```

**Access:**
- Frontend: http://localhost:3000
- Backend API: http://localhost:8080/api/v1
- Redis: localhost:6379

**Documentation:** See [DISTRIBUTED_ARCHITECTURE.md](DISTRIBUTED_ARCHITECTURE.md)

---

## Comparison

| Feature | Unified | Distributed |
|---------|---------|-------------|
| **Containers** | 1 | 4+ |
| **Setup Complexity** | Simple | Medium |
| **Scalability** | Vertical only | Horizontal |
| **Resource Usage** | ~2GB RAM | ~2.5GB+ base |
| **Job Processing** | Synchronous | Async + Sync |
| **Failure Isolation** | None | Per-service |
| **Worker Scaling** | No | Yes (N instances) |
| **Best For** | Dev/Test | Production |

---

## Migration Path

Both architectures use the same codebase and API. You can:

1. **Start Simple:** Deploy unified architecture for development
2. **Scale Up:** Switch to distributed when needed
3. **Hybrid:** Use unified for development, distributed for production

---

## Choosing an Architecture

### Use Unified Architecture if:
- Quick setup is priority
- Running on single machine
- Development/testing environment
- Small workload (<100 pipelines/day)
- Limited infrastructure

### Use Distributed Architecture if:
- Need horizontal scaling
- High pipeline throughput required
- Want async job processing
- Need worker isolation
- Production deployment
- Multi-region deployment planned

---

## Architecture Files

### Unified
- `Dockerfile.unified` - Combined container
- `docker-compose.unified.yml` - Single service compose
- `build-unified.sh` - Build script
- `start-unified.sh` - Startup script

### Distributed
- `Dockerfile.frontend` - Next.js frontend
- `Dockerfile.backend` - Go API + Jena
- `Dockerfile.worker` - Scalable workers
- `docker-compose.distributed.yml` - Multi-service compose
- `build-distributed.sh` - Build script
- `start-distributed.sh` - Management script

---

## API Compatibility

Both architectures support the same API endpoints:

### Synchronous Execution (Both)
```bash
POST /api/v1/pipelines/execute
```

### Asynchronous Execution (Distributed Only)
```bash
POST /api/v1/queue/pipelines/enqueue
GET  /api/v1/queue/jobs/{id}
GET  /api/v1/queue/jobs/{id}/wait
GET  /api/v1/queue/status
```

When using unified architecture, async endpoints return 503 (not available).

---

## Testing

### Test Unified
```bash
./start-unified.sh
curl http://localhost:8080/health
```

### Test Distributed
```bash
./test-distributed.sh
./start-distributed.sh up
curl http://localhost:8080/health
curl http://localhost:3000
```

See [INTEGRATION_TESTING.md](INTEGRATION_TESTING.md) for detailed test scenarios.

---

## Further Reading

- [DISTRIBUTED_ARCHITECTURE.md](DISTRIBUTED_ARCHITECTURE.md) - Detailed distributed architecture guide
- [INTEGRATION_TESTING.md](INTEGRATION_TESTING.md) - Testing guide
- Main README.md - Core platform documentation
