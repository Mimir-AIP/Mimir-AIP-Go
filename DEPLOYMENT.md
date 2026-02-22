# Mimir AIP - Deployment Guide

This guide covers deploying the Mimir AIP orchestrator and frontend to various environments.

## Quick Start

### Option 1: Docker Compose (Recommended for Local Development)

```bash
# Build and start all services
docker-compose up --build

# Access the services
# Frontend: http://localhost:3000
# Orchestrator API: http://localhost:8080
```

### Option 2: Kubernetes (Production)

```bash
# Build Docker images
make build-all

# Deploy to Kubernetes
make deploy-k8s

# Check status
make status

# Forward ports for local access
make port-forward-frontend  # http://localhost:3000
```

## Detailed Deployment Instructions

### Prerequisites

- Docker 20.10+
- Kubernetes 1.24+ (for k8s deployment)
- kubectl configured with cluster access
- Go 1.21+ (for local development)

### Building Docker Images

#### Build Orchestrator
```bash
make build-orchestrator
# or
docker build -t mimir-aip/orchestrator:latest -f cmd/orchestrator/Dockerfile .
```

#### Build Frontend
```bash
make build-frontend
# or
docker build -t mimir-aip/frontend:latest -f frontend/Dockerfile .
```

#### Build All
```bash
make build-all
```

### Kubernetes Deployment

#### Architecture
The k8s deployment consists of:
- **Namespace**: `mimir-aip` - isolated environment
- **ConfigMap**: Configuration for orchestrator
- **PVC**: Persistent storage for orchestrator data
- **Orchestrator Deployment**: Backend API server
- **Orchestrator Service**: ClusterIP service (internal)
- **Frontend Deployment**: Web UI server
- **Frontend Service**: LoadBalancer service (external access)

#### Deploy to Kubernetes

1. **Deploy all components**:
```bash
make deploy-k8s
```

This will:
- Create the namespace
- Apply all configuration
- Deploy orchestrator and frontend
- Wait for deployments to be ready

2. **Check deployment status**:
```bash
make status
```

3. **View logs**:
```bash
# Orchestrator logs
make logs-orchestrator

# Frontend logs
make logs-frontend

# All logs
make logs-all
```

#### Access the Application

##### Option 1: Port Forwarding (Development)
```bash
# Terminal 1: Forward frontend
make port-forward-frontend

# Terminal 2: Forward orchestrator (optional, for direct API access)
make port-forward-orchestrator

# Access at http://localhost:3000
```

##### Option 2: LoadBalancer (Production)
```bash
# Get the external IP
kubectl get svc frontend -n mimir-aip

# Access via the EXTERNAL-IP shown
```

### Docker Compose Deployment

#### Start Services
```bash
# Build and start in foreground
docker-compose up --build

# Build and start in background
docker-compose up --build -d

# View logs
docker-compose logs -f

# Stop services
docker-compose down

# Stop and remove volumes
docker-compose down -v
```

#### Access Services
- Frontend: http://localhost:3000
- Orchestrator API: http://localhost:8080
- Health Check: http://localhost:8080/health

### Local Development

#### Run Orchestrator Locally
```bash
make dev-orchestrator
# or
cd cmd/orchestrator && go run main.go
```

#### Run Frontend Locally
```bash
make dev-frontend
# or
cd frontend && PORT=3000 API_URL=http://localhost:8080 go run server.go
```

## Configuration

### Environment Variables

#### Orchestrator
- `ENVIRONMENT`: Environment name (development/production)
- `LOG_LEVEL`: Logging level (debug/info/warn/error)
- `PORT`: HTTP server port (default: 8080)
- `STORAGE_DIR`: Data persistence directory
- `MIN_WORKERS`: Minimum worker pods
- `MAX_WORKERS`: Maximum worker pods
- `QUEUE_THRESHOLD`: Queue length threshold for scaling

#### Frontend
- `PORT`: HTTP server port (default: 3000)
- `API_URL`: Orchestrator API URL (default: http://orchestrator:8080)

### Kubernetes Configuration

Edit `k8s/development/01-config.yaml` to modify orchestrator configuration.

### Resource Limits

#### Orchestrator
- Requests: 1Gi memory, 500m CPU
- Limits: 2Gi memory, 1000m CPU

#### Frontend
- Requests: 64Mi memory, 50m CPU
- Limits: 128Mi memory, 100m CPU

## Monitoring and Troubleshooting

### Health Checks

#### Orchestrator
- Liveness: `GET /health`
- Readiness: `GET /ready`

#### Frontend
- Liveness: `GET /`

### View Logs

```bash
# Kubernetes
make logs-orchestrator
make logs-frontend

# Docker Compose
docker-compose logs orchestrator
docker-compose logs frontend
```

### Describe Resources

```bash
make describe-orchestrator
make describe-frontend
```

### Common Issues

#### 1. Frontend cannot connect to orchestrator

**Kubernetes**:
- Check orchestrator service: `kubectl get svc orchestrator -n mimir-aip`
- Verify API_URL env var: `kubectl get deployment frontend -n mimir-aip -o yaml | grep API_URL`

**Docker Compose**:
- Verify network connectivity: `docker-compose exec frontend ping orchestrator`
- Check API_URL: `docker-compose exec frontend env | grep API_URL`

#### 2. Pods not starting

```bash
# Check pod status
kubectl get pods -n mimir-aip

# View pod events
kubectl describe pod <pod-name> -n mimir-aip

# Check logs
kubectl logs <pod-name> -n mimir-aip
```

#### 3. Image pull failures

For local development with `imagePullPolicy: Never`:
```bash
# Build images on the cluster node
make build-all

# For minikube
eval $(minikube docker-env)
make build-all
```

## Cleanup

### Kubernetes
```bash
# Delete all resources
make delete-k8s

# Or delete specific components
kubectl delete -f k8s/development/04-frontend.yaml
kubectl delete -f k8s/development/03-orchestrator.yaml
```

### Docker Compose
```bash
# Stop and remove containers
docker-compose down

# Remove volumes as well
docker-compose down -v
```

### Docker Images
```bash
make clean
```

## Production Considerations

1. **Image Registry**: Push images to a container registry
```bash
docker tag mimir-aip/orchestrator:latest your-registry.com/mimir-aip/orchestrator:v1.0.0
docker push your-registry.com/mimir-aip/orchestrator:v1.0.0
```

2. **Update imagePullPolicy**: Change from `Never` to `Always` or `IfNotPresent` in k8s manifests

3. **Ingress**: Replace LoadBalancer with Ingress for production routing

4. **TLS**: Configure TLS certificates for HTTPS

5. **Secrets**: Use Kubernetes Secrets for sensitive data

6. **Resource Limits**: Adjust based on actual usage patterns

7. **Monitoring**: Add Prometheus/Grafana for metrics

8. **Backup**: Configure PVC backup strategy

## Make Commands Reference

```bash
make help                    # Show all available commands
make build-orchestrator      # Build orchestrator image
make build-frontend          # Build frontend image
make build-all              # Build all images
make deploy-k8s             # Deploy to Kubernetes
make delete-k8s             # Delete k8s resources
make redeploy               # Full rebuild and redeploy
make logs-orchestrator      # View orchestrator logs
make logs-frontend          # View frontend logs
make port-forward-frontend  # Forward frontend port
make port-forward-orchestrator # Forward orchestrator port
make status                 # Check deployment status
make clean                  # Clean Docker images
make dev-frontend           # Run frontend locally
make dev-orchestrator       # Run orchestrator locally
```

## Architecture Diagram

```
┌─────────────────────────────────────────────┐
│              Kubernetes Cluster             │
│  ┌────────────────────────────────────────┐ │
│  │      Namespace: mimir-aip              │ │
│  │                                        │ │
│  │  ┌──────────────┐  ┌───────────────┐  │ │
│  │  │   Frontend   │  │ Orchestrator  │  │ │
│  │  │  Deployment  │  │  Deployment   │  │ │
│  │  │              │  │               │  │ │
│  │  │  - Web UI    │  │  - REST API   │  │ │
│  │  │  - Proxy     │──│  - Worker Mgr │  │ │
│  │  └──────────────┘  │  - Queue      │  │ │
│  │         │          └───────┬───────┘  │ │
│  │         │                  │          │ │
│  │  ┌──────▼──────┐    ┌─────▼────────┐ │ │
│  │  │  Frontend   │    │ Orchestrator │ │ │
│  │  │  Service    │    │   Service    │ │ │
│  │  │ LoadBalancer│    │  ClusterIP   │ │ │
│  │  └─────────────┘    └──────────────┘ │ │
│  │                           │          │ │
│  │                     ┌─────▼────────┐ │ │
│  │                     │     PVC      │ │ │
│  │                     │  (Storage)   │ │ │
│  │                     └──────────────┘ │ │
│  └────────────────────────────────────┘ │
└─────────────────────────────────────────┘
           │
           ▼
    External Access
    http://localhost:3000
```
