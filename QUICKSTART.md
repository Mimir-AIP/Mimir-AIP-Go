# Mimir AIP - Quick Start Guide

## Prerequisites

### For Local Development (Rancher Desktop)
- Rancher Desktop installed and running with Kubernetes enabled
- Go 1.21+ installed
- kubectl configured

### For NUC Server Deployment
- Go 1.21+ installed
- kubectl configured with NUC server access
- `knuc` alias configured (see [scripts/README.md](scripts/README.md))
- SSH access to NUC server

## 5-Minute Setup

Choose your deployment target:
- **Option A**: Deploy to NUC Server (recommended for testing on remote hardware)
- **Option B**: Deploy Locally (Rancher Desktop)

### Option A: Deploy to NUC Server

### 1. Clone and Navigate
```bash
cd /path/to/Mimir-AIP-Go
```

### 2. Run Full Deployment
```bash
./scripts/full-deploy.sh --nuc
```

This single command will:
- Build all Docker images locally
- Transfer images to NUC server
- Deploy to NUC Kubernetes cluster
- Run integration tests
- Verify everything works

### 3. Access the Frontend

In a new terminal:
```bash
knuc port-forward -n mimir-aip svc/frontend 8081:80
```

Open http://localhost:8081 in your browser

### 4. View Logs

**Orchestrator:**
```bash
knuc logs -n mimir-aip -l component=orchestrator -f
```

**Workers:**
```bash
knuc logs -n mimir-aip -l app=mimir-worker -f
```

### Option B: Deploy Locally (Rancher Desktop)

### 1. Clone and Navigate
```bash
cd /path/to/Mimir-AIP-Go
```

### 2. Run Full Deployment
```bash
./scripts/full-deploy.sh
```

This single command will:
- Build all Docker images
- Deploy to Kubernetes
- Run integration tests
- Verify everything works

### 3. Access the Frontend

In a new terminal:
```bash
kubectl port-forward -n mimir-aip svc/frontend 8081:80
```

Open http://localhost:8081 in your browser

### 4. Submit a Test Job

You can submit a job through the web UI or via curl:

```bash
curl -X POST http://localhost:8080/api/jobs \
  -H "Content-Type: application/json" \
  -d '{
    "type": "pipeline_execution",
    "priority": 1,
    "project_id": "test-project",
    "task_spec": {
      "pipeline_id": "test-pipeline",
      "parameters": {}
    },
    "resource_requirements": {
      "cpu": "500m",
      "memory": "1Gi",
      "gpu": false
    },
    "data_access": {
      "input_datasets": [],
      "output_location": "s3://test/results/"
    }
  }'
```

### 5. Watch the Worker Spawn

In another terminal, watch the worker jobs being created:
```bash
kubectl get jobs -n mimir-aip -w
```

### 6. View Logs

**Orchestrator:**
```bash
kubectl logs -n mimir-aip -l component=orchestrator -f
```

**Workers:**
```bash
kubectl logs -n mimir-aip -l app=mimir-worker -f
```

## Common Tasks

### Rebuild and Deploy After Code Changes

**For NUC Server:**
```bash
./scripts/build-images.sh --remote ciaran@192.168.0.101
knuc rollout restart deployment/orchestrator -n mimir-aip
knuc rollout restart deployment/frontend -n mimir-aip
```

**For Local:**
```bash
./scripts/build-images.sh
kubectl rollout restart deployment/orchestrator -n mimir-aip
kubectl rollout restart deployment/frontend -n mimir-aip
```

### Run Tests

**Against NUC Server:**
```bash
KUBECONFIG=~/.kube/config-nuc ./scripts/run-integration-tests.sh
```

**Against Local:**
```bash
# Unit tests
go test ./pkg/... -v

# Integration tests
./scripts/run-integration-tests.sh
```

### Clean Up

**NUC Server:**
```bash
./scripts/undeploy-local.sh --nuc
```

**Local:**
```bash
./scripts/undeploy-local.sh
```

## Troubleshooting

### Port Already in Use
If you see "port already in use" errors:
```bash
# Find and kill the process
lsof -ti:8080 | xargs kill -9
lsof -ti:8081 | xargs kill -9
```

### Pods Not Starting (NUC Server)
Check pod status:
```bash
knuc get pods -n mimir-aip
knuc describe pod <pod-name> -n mimir-aip
```

### Pods Not Starting (Local)
Check pod status:
```bash
kubectl get pods -n mimir-aip
kubectl describe pod <pod-name> -n mimir-aip
```

### Images Not Found on NUC
If you see `ImagePullBackOff` errors on NUC:
```bash
./scripts/build-images.sh --remote ciaran@192.168.0.101
./scripts/deploy-local.sh --nuc
```

### Connection Issues to NUC
Test your connection:
```bash
knuc get nodes
ssh ciaran@192.168.0.101 "echo 'Connection successful'"
```

### Need Fresh Start (NUC)
```bash
./scripts/undeploy-local.sh --nuc
knuc delete namespace mimir-aip
./scripts/full-deploy.sh --nuc
```

### Need Fresh Start (Local)
```bash
./scripts/undeploy-local.sh
kubectl delete namespace mimir-aip
./scripts/full-deploy.sh
```

## Next Steps

Once the infrastructure is running:
1. Explore the API endpoints (see README.md)
2. Review the architecture plans in Plan/
3. Start implementing additional components (Storage, Projects, Pipelines, etc.)

For detailed information, see README.md
