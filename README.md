# Mimir AIP - Infrastructure & Scaling

This directory contains the complete infrastructure and scaling implementation for the Mimir AIP platform.

## Overview

The Mimir AIP infrastructure consists of three main components deployed as Kubernetes containers:

1. **Orchestrator** - Central coordination service managing projects, scheduling jobs, and coordinating workers
2. **Worker** - Scalable job execution containers that spawn on-demand based on queue depth
3. **Frontend** - Static web interface for system monitoring and job submission

## Technology Stack

- **Backend**: Go 1.21
- **Frontend**: Static HTML/CSS/JavaScript
- **Queue**: Redis
- **Container Orchestration**: Kubernetes
- **Local Development**: Rancher Desktop

## Project Structure

```
.
├── cmd/
│   ├── orchestrator/    # Orchestrator server main application
│   │   ├── main.go
│   │   └── Dockerfile
│   └── worker/          # Worker main application
│       ├── main.go
│       └── Dockerfile
├── pkg/
│   ├── api/            # REST API server
│   ├── config/         # Configuration management
│   ├── k8s/            # Kubernetes client
│   ├── models/         # Data models
│   └── queue/          # Redis job queue
├── frontend/           # Static web interface
│   ├── index.html
│   ├── styles.css
│   ├── app.js
│   ├── server.go
│   └── Dockerfile
├── k8s/
│   ├── development/    # Kubernetes manifests for local dev
│   └── production/     # Kubernetes manifests for production
├── scripts/            # Deployment and testing scripts
├── tests/
│   ├── unit/          # Unit tests
│   └── integration/   # Integration tests
└── Plan/              # Architecture and planning documents
```

## Prerequisites

### For Local Development (Rancher Desktop)

1. **Install Rancher Desktop**
   - Download from https://rancherdesktop.io/
   - Enable Kubernetes
   - Select dockerd (moby) as the container runtime

2. **Install Go 1.21+**
   ```bash
   brew install go
   ```

3. **Install kubectl**
   ```bash
   brew install kubectl
   ```

## Quick Start

### 1. Build Docker Images

Build all Docker images for local deployment:

```bash
./scripts/build-images.sh
```

This will build:
- `mimir-aip/orchestrator:latest`
- `mimir-aip/worker:latest`
- `mimir-aip/frontend:latest`

### 2. Deploy to Kubernetes

Deploy the entire stack to your local Rancher Desktop cluster:

```bash
./scripts/deploy-local.sh
```

This will:
- Create the `mimir-aip` namespace
- Deploy Redis
- Deploy the Orchestrator
- Deploy the Frontend
- Create necessary ConfigMaps and Secrets

### 3. Access the System

**Frontend (Web UI):**
```bash
kubectl port-forward -n mimir-aip svc/frontend 8081:80
```
Then open http://localhost:8081 in your browser

**Orchestrator API:**
```bash
kubectl port-forward -n mimir-aip svc/orchestrator 8080:8080
```
API available at http://localhost:8080

### 4. Run Tests

**Unit Tests:**
```bash
go test ./pkg/... -v
```

**Integration Tests:**
```bash
./scripts/run-integration-tests.sh
```

### Full Deployment Pipeline

Run the complete deployment and testing pipeline:

```bash
./scripts/full-deploy.sh
```

This will:
1. Build all Docker images
2. Deploy to Kubernetes
3. Wait for services to stabilize
4. Run integration tests

## API Endpoints

### Orchestrator API

#### Health & Status
- `GET /health` - Health check
- `GET /ready` - Readiness check

#### Job Management
- `POST /api/jobs` - Submit a new job
- `GET /api/jobs` - Get queue status
- `GET /api/jobs/{id}` - Get job details
- `POST /api/jobs/{id}` - Update job status

### Job Submission Example

```bash
curl -X POST http://localhost:8080/api/jobs \
  -H "Content-Type: application/json" \
  -d '{
    "type": "pipeline_execution",
    "priority": 1,
    "project_id": "my-project",
    "task_spec": {
      "pipeline_id": "data-pipeline",
      "parameters": {}
    },
    "resource_requirements": {
      "cpu": "500m",
      "memory": "1Gi",
      "gpu": false
    },
    "data_access": {
      "input_datasets": [],
      "output_location": "s3://bucket/results/"
    }
  }'
```

## Monitoring & Debugging

### View Logs

**Orchestrator logs:**
```bash
kubectl logs -n mimir-aip -l component=orchestrator -f
```

**Worker logs:**
```bash
kubectl logs -n mimir-aip -l app=mimir-worker -f
```

**Redis logs:**
```bash
kubectl logs -n mimir-aip -l component=redis -f
```

### Check Pod Status

```bash
kubectl get pods -n mimir-aip
```

### Check Services

```bash
kubectl get svc -n mimir-aip
```

### View Worker Jobs

```bash
kubectl get jobs -n mimir-aip
```

### Access Kubernetes Dashboard

If you have the Kubernetes dashboard enabled in Rancher Desktop:
```bash
kubectl proxy
```
Then access http://localhost:8001/api/v1/namespaces/kubernetes-dashboard/services/https:kubernetes-dashboard:/proxy/

## Scaling Configuration

The orchestrator manages worker scaling based on queue depth. Configuration is set via environment variables in the deployment manifest:

- `MIN_WORKERS`: Minimum number of workers to maintain (default: 1)
- `MAX_WORKERS`: Maximum number of workers allowed (default: 10)
- `QUEUE_THRESHOLD`: Queue length that triggers scaling (default: 5)

To modify these, edit `k8s/development/03-orchestrator.yaml` and redeploy.

## Worker Job Types

The system supports four job types:

1. **pipeline_execution** - Data ingestion and processing pipelines
2. **ml_training** - Machine learning model training
3. **ml_inference** - Model inference and predictions
4. **digital_twin_update** - Digital twin state updates

Each job type can have different resource requirements specified in the job submission request.

## Troubleshooting

### Images not found

If you see "ImagePullBackOff" errors, ensure you built the images:
```bash
./scripts/build-images.sh
```

And that `imagePullPolicy: Never` is set in the deployment manifests.

### Redis connection errors

Check if Redis is running:
```bash
kubectl get pods -n mimir-aip -l component=redis
```

View Redis logs:
```bash
kubectl logs -n mimir-aip -l component=redis
```

### Workers not spawning

Check orchestrator logs for errors:
```bash
kubectl logs -n mimir-aip -l component=orchestrator -f
```

Verify the orchestrator has proper RBAC permissions:
```bash
kubectl get rolebinding -n mimir-aip
```

### Port forwarding issues

Make sure no other processes are using the ports:
```bash
lsof -i :8080
lsof -i :8081
```

## Cleanup

To remove the entire deployment:

```bash
./scripts/undeploy-local.sh
```

To also remove the namespace:
```bash
kubectl delete namespace mimir-aip
```

## Development Workflow

1. Make changes to source code
2. Run unit tests: `go test ./pkg/... -v`
3. Build images: `./scripts/build-images.sh`
4. Deploy: `./scripts/deploy-local.sh`
5. Run integration tests: `./scripts/run-integration-tests.sh`
6. Monitor logs and verify functionality

## Next Steps

This infrastructure provides the foundation for:
- Storage system integration
- Project management
- Pipeline execution
- Ontology management
- ML model training/inference
- Digital twin creation and management

See the respective plan files in the `Plan/` directory for details on each component.

## Architecture Diagrams

For detailed architecture information, see:
- `Plan/Infrastructure/InfrastructurePlan.md` - Complete infrastructure specification
- `Plan/Scaling/ScalingPlan.md` - Worker scaling architecture
- `Plan/MimirAIPOverallPlan.md` - Overall system architecture
