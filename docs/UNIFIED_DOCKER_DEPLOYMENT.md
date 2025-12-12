# Unified Docker Deployment Guide

## Overview

This guide covers deploying Mimir AIP as a **single unified Docker container** that serves both:
- **Backend API** (Go server on port 8080)
- **Frontend UI** (Next.js React application)

## Quick Start

### Option 1: Using Docker Compose (Recommended)

```bash
# Build and start the unified container
docker-compose -f docker-compose.unified.yml up -d

# View logs
docker-compose -f docker-compose.unified.yml logs -f

# Stop the container
docker-compose -f docker-compose.unified.yml down
```

Access the application:
- **Frontend UI**: http://localhost:8080
- **API Endpoints**: http://localhost:8080/api/v1/*
- **Health Check**: http://localhost:8080/health

### Option 2: Using Docker CLI

```bash
# Build the image
docker build -f Dockerfile.unified -t mimir-aip:unified .

# Run the container
docker run -d \
  --name mimir-aip-unified \
  -p 8080:8080 \
  -v mimir_data:/app/data \
  -v mimir_logs:/app/logs \
  -v mimir_config:/app/config \
  -e MIMIR_LOG_LEVEL=INFO \
  mimir-aip:unified

# View logs
docker logs -f mimir-aip-unified

# Stop and remove
docker stop mimir-aip-unified
docker rm mimir-aip-unified
```

## Architecture

### Multi-Stage Build

The `Dockerfile.unified` uses a 3-stage build process:

1. **Stage 1: Frontend Builder**
   - Uses `node:20-alpine` with Bun
   - Builds Next.js application
   - Creates standalone output for production

2. **Stage 2: Backend Builder**
   - Uses `golang:1.23-alpine`
   - Compiles Go server with optimizations
   - Creates static binary

3. **Stage 3: Runtime**
   - Uses `gcr.io/distroless/static-debian12` (minimal, secure)
   - Copies frontend static files
   - Copies backend binary
   - Runs as non-root user (UID 65532)

### Directory Structure in Container

```
/app/
├── mimir-aip-server          # Go backend binary
├── config.yaml                # Configuration file
├── frontend/                  # Frontend static files
│   ├── .next/
│   │   ├── static/           # Next.js static assets
│   │   └── standalone/       # Server files
│   ├── public/               # Public assets
│   └── index.html            # Main HTML file
├── data/                      # Persistent data (volume)
├── logs/                      # Application logs (volume)
└── plugins/                   # Plugin directory (volume)
```

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `MIMIR_PORT` | `8080` | Server port |
| `MIMIR_LOG_LEVEL` | `INFO` | Log level (DEBUG, INFO, WARN, ERROR) |
| `MIMIR_DATA_DIR` | `/app/data` | Data storage directory |
| `MIMIR_LOG_DIR` | `/app/logs` | Log file directory |
| `MIMIR_PLUGIN_DIR` | `/app/plugins` | Plugin directory |
| `MIMIR_FRONTEND_DIR` | `/app/frontend` | Frontend static files directory |

### Ports

- **8080**: HTTP server (API + Frontend)

## Routing

The Go server handles routing as follows:

1. **API Routes** (`/api/*`)
   - All `/api/v1/*` requests → Backend API handlers
   - JSON responses with CORS enabled

2. **Static Files**
   - `/_next/static/*` → Next.js compiled assets
   - `/static/*` → Public files (images, etc.)
   - `/favicon.ico` → Site favicon

3. **Frontend SPA** (`/*`)
   - All other routes → Serve `index.html`
   - Enables client-side routing

## Volumes

Persistent data is stored in Docker volumes:

```yaml
volumes:
  mimir_data:      # Application data
  mimir_logs:      # Log files
  mimir_config:    # Configuration files
  mimir_plugins:   # Custom plugins
```

### Backup Volumes

```bash
# Backup data volume
docker run --rm \
  -v mimir_data:/data \
  -v $(pwd):/backup \
  alpine tar czf /backup/mimir-data-backup.tar.gz /data

# Restore data volume
docker run --rm \
  -v mimir_data:/data \
  -v $(pwd):/backup \
  alpine tar xzf /backup/mimir-data-backup.tar.gz -C /
```

## Health Checks

The container includes automatic health checks:

```yaml
healthcheck:
  test: ["/app/mimir-aip-server", "--version"]
  interval: 30s
  timeout: 10s
  retries: 3
  start_period: 40s
```

Check health status:
```bash
docker inspect --format='{{.State.Health.Status}}' mimir-aip-unified
```

## Resource Limits

Default resource constraints:

```yaml
resources:
  limits:
    cpus: '2.0'
    memory: 1G
  reservations:
    cpus: '1.0'
    memory: 512M
```

Adjust in `docker-compose.unified.yml` as needed for your workload.

## Production Deployment

### Security Best Practices

1. **Use TLS/HTTPS**
   ```bash
   # Add reverse proxy (nginx, traefik, etc.)
   # Or use built-in TLS (requires certificate)
   docker run -d \
     -p 443:8080 \
     -v /path/to/certs:/certs \
     -e MIMIR_TLS_CERT=/certs/server.crt \
     -e MIMIR_TLS_KEY=/certs/server.key \
     mimir-aip:unified
   ```

2. **Enable Authentication**
   - Edit `config.yaml` to enable JWT authentication
   - Set strong secrets for token signing

3. **Network Isolation**
   - Use Docker networks to isolate services
   - Don't expose port 8080 externally (use reverse proxy)

4. **Read-Only Filesystem** (Optional)
   ```yaml
   read_only: true
   tmpfs:
     - /tmp
     - /app/logs  # If logs should be ephemeral
   ```

### Monitoring

**View Logs**:
```bash
# Real-time logs
docker-compose -f docker-compose.unified.yml logs -f

# Last 100 lines
docker-compose -f docker-compose.unified.yml logs --tail=100

# Specific service logs
docker logs mimir-aip-unified
```

**Performance Metrics**:
```bash
# Container stats
docker stats mimir-aip-unified

# API metrics endpoint
curl http://localhost:8080/api/v1/performance/stats
```

### Scaling

For high availability, use Docker Swarm or Kubernetes:

**Docker Swarm**:
```bash
docker swarm init
docker stack deploy -c docker-compose.unified.yml mimir-stack
docker service scale mimir-stack_mimir-aip-unified=3
```

**Kubernetes**:
```bash
# Convert docker-compose to k8s manifest
kompose convert -f docker-compose.unified.yml

# Deploy to k8s
kubectl apply -f mimir-deployment.yaml
kubectl scale deployment mimir-aip-unified --replicas=3
```

## Troubleshooting

### Container Won't Start

```bash
# Check logs for errors
docker logs mimir-aip-unified

# Inspect container
docker inspect mimir-aip-unified

# Check health status
docker inspect --format='{{json .State.Health}}' mimir-aip-unified | jq
```

### Frontend Not Loading

1. **Check if frontend files exist**:
   ```bash
   docker exec mimir-aip-unified ls -la /app/frontend
   ```

2. **Verify MIMIR_FRONTEND_DIR**:
   ```bash
   docker exec mimir-aip-unified env | grep FRONTEND
   ```

3. **Check logs for routing errors**:
   ```bash
   docker logs mimir-aip-unified 2>&1 | grep -i frontend
   ```

### API Not Responding

1. **Check if server is listening**:
   ```bash
   docker exec mimir-aip-unified netstat -tlnp
   # Or test from inside container
   docker exec mimir-aip-unified wget -O- http://localhost:8080/health
   ```

2. **Verify CORS settings** (if calling from different origin):
   - Check `main.go` for allowed origins
   - Update CORS configuration if needed

### Volume Permission Issues

If you encounter permission errors:

```bash
# Check volume ownership
docker run --rm -v mimir_data:/data alpine ls -la /data

# Fix permissions (if needed)
docker run --rm -v mimir_data:/data alpine chown -R 65532:65532 /data
```

## Updating

### Update to Latest Version

```bash
# Pull latest code
git pull origin main

# Rebuild image
docker-compose -f docker-compose.unified.yml build

# Restart with new image
docker-compose -f docker-compose.unified.yml up -d
```

### Zero-Downtime Update

1. Build new image with version tag:
   ```bash
   docker build -f Dockerfile.unified -t mimir-aip:unified-v2 .
   ```

2. Start new container on different port:
   ```bash
   docker run -d --name mimir-new -p 8081:8080 mimir-aip:unified-v2
   ```

3. Test new container:
   ```bash
   curl http://localhost:8081/health
   ```

4. Update load balancer/reverse proxy to point to new container

5. Stop old container:
   ```bash
   docker stop mimir-aip-unified
   docker rm mimir-aip-unified
   ```

## Development vs Production

### Development Setup

For local development, use separate containers:

```bash
# Terminal 1: Run backend
go run main.go --server

# Terminal 2: Run frontend
cd mimir-aip-frontend
bun run dev
```

### Production Setup

Use the unified Docker container for production:

```bash
docker-compose -f docker-compose.unified.yml up -d
```

## Additional Resources

- **API Documentation**: See `docs/API_REFERENCE.md`
- **Plugin Development**: See `docs/PLUGIN_DEVELOPMENT_GUIDE.md`
- **Configuration**: See `config.yaml` for all options

## Support

For issues or questions:
- GitHub Issues: https://github.com/Mimir-AIP/Mimir-AIP-Go/issues
- Documentation: `docs/`
