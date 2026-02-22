# Mimir AIP Deployment Scripts

This directory contains scripts for building and deploying Mimir AIP to Kubernetes clusters.

## Prerequisites

### For Local Deployment (Rancher Desktop)
- Docker Desktop or Rancher Desktop with Kubernetes enabled
- kubectl configured to point to local cluster

### For NUC Server Deployment
- kubectl with NUC server kubeconfig at `~/.kube/config-nuc`
- SSH access to NUC server (`ciaran@192.168.0.101`)
- The `knuc` alias configured in `~/.zshrc`:
  ```bash
  alias knuc='KUBECONFIG=~/.kube/config-nuc kubectl'
  ```

## Scripts

### `build-images.sh`
Builds Docker images for all components (orchestrator, worker, frontend).

**Usage:**
```bash
# Build for local use
./scripts/build-images.sh

# Build and transfer to NUC server
./scripts/build-images.sh --remote ciaran@192.168.0.101
```

### `deploy-local.sh`
Deploys Mimir AIP to a Kubernetes cluster.

**Usage:**
```bash
# Deploy to local Rancher Desktop
./scripts/deploy-local.sh

# Deploy to NUC server
./scripts/deploy-local.sh --nuc

# Deploy with custom kubeconfig
./scripts/deploy-local.sh --kubeconfig /path/to/config
```

### `undeploy-local.sh`
Removes Mimir AIP from a Kubernetes cluster.

**Usage:**
```bash
# Remove from local cluster
./scripts/undeploy-local.sh

# Remove from NUC server
./scripts/undeploy-local.sh --nuc
```

### `full-deploy.sh`
Complete deployment pipeline: builds images, deploys, and runs integration tests.

**Usage:**
```bash
# Full local deployment
./scripts/full-deploy.sh

# Full NUC server deployment
./scripts/full-deploy.sh --nuc

# Deploy without running tests
./scripts/full-deploy.sh --nuc --skip-tests
```

### `run-integration-tests.sh`
Runs integration tests against the deployed system.

**Usage:**
```bash
# Run tests against local deployment
./scripts/run-integration-tests.sh

# Run tests against NUC deployment
KUBECONFIG=~/.kube/config-nuc ./scripts/run-integration-tests.sh
```

## Quick Start

### Deploy to NUC Server

```bash
# One-time setup (if not already done)
# The kubeconfig should already be at ~/.kube/config-nuc
# The knuc alias should be in your ~/.zshrc

# Reload shell to get the alias
source ~/.zshrc

# Full deployment to NUC
./scripts/full-deploy.sh --nuc

# Access the frontend
knuc port-forward -n mimir-aip svc/frontend 8081:80
# Then open http://localhost:8081

# View logs
knuc logs -n mimir-aip -l component=orchestrator -f

# Check status
knuc get all -n mimir-aip
```

### Deploy Locally (Rancher Desktop)

```bash
# Full local deployment
./scripts/full-deploy.sh

# Access the frontend
kubectl port-forward -n mimir-aip svc/frontend 8081:80
# Then open http://localhost:8081
```

## Useful Commands

### NUC Server
```bash
# Get pods
knuc get pods -n mimir-aip

# Get services
knuc get svc -n mimir-aip

# View orchestrator logs
knuc logs -n mimir-aip -l component=orchestrator -f

# View worker logs
knuc logs -n mimir-aip -l app=mimir-worker -f

# Describe a pod
knuc describe pod <pod-name> -n mimir-aip

# Port forward to orchestrator API
knuc port-forward -n mimir-aip svc/orchestrator 8080:8080
```

### Local Cluster
Replace `knuc` with `kubectl` in all the above commands.

## Troubleshooting

### Images not found on NUC
If you see `ImagePullBackOff` errors on the NUC:

```bash
# Rebuild and transfer images
./scripts/build-images.sh --remote ciaran@192.168.0.101

# Then redeploy
./scripts/deploy-local.sh --nuc
```

### Connection issues to NUC
Test your connection:
```bash
# Test kubectl connection
knuc get nodes

# Test SSH connection
ssh ciaran@192.168.0.101 "echo 'Connection successful'"
```

### Check NUC server images
```bash
# SSH to NUC and list images
ssh ciaran@192.168.0.101 "sudo k3s crictl images | grep mimir-aip"
```
