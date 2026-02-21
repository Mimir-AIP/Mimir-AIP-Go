#!/bin/bash

# deploy-nuc.sh - Deploy Mimir AIP to NUC server (builds on NUC)
set -e

NUC_HOST="ciaran@192.168.0.101"
REMOTE_BUILD_DIR="/tmp/mimir-aip-build"

echo "========================================="
echo "Mimir AIP - NUC Server Deployment"
echo "========================================="
echo ""

# Step 1: Sync code to NUC
echo "Step 1: Syncing code to NUC server..."
echo "Cleaning old build directory on NUC..."
ssh "$NUC_HOST" "rm -rf $REMOTE_BUILD_DIR && mkdir -p $REMOTE_BUILD_DIR"

echo "Copying source code to NUC..."
rsync -av --exclude='.git' --exclude='node_modules' --exclude='*.log' \
  /Users/ciaran/Documents/Github/Mimir-AIP-Go/ "$NUC_HOST:$REMOTE_BUILD_DIR/"

echo ""
echo "========================================="

# Step 2: Build and import images on NUC
echo "Step 2: Building Docker images on NUC server..."
ssh "$NUC_HOST" 'set -e; cd /tmp/mimir-aip-build && \
  echo "Building orchestrator image..." && \
  docker build -f cmd/orchestrator/Dockerfile -t mimir-aip/orchestrator:latest . && \
  echo "Building worker image..." && \
  docker build -f cmd/worker/Dockerfile -t mimir-aip/worker:latest . && \
  echo "Building frontend image..." && \
  docker build -f frontend/Dockerfile -t mimir-aip/frontend:latest . && \
  echo "Saving images to tar files..." && \
  docker save mimir-aip/orchestrator:latest -o /tmp/orchestrator.tar && \
  docker save mimir-aip/worker:latest -o /tmp/worker.tar && \
  docker save mimir-aip/frontend:latest -o /tmp/frontend.tar && \
  echo "Images built and saved successfully!"'

echo "Importing images into k3s (this may prompt for sudo password)..."
ssh -t "$NUC_HOST" 'sudo ctr -n k8s.io images import /tmp/orchestrator.tar && \
  sudo ctr -n k8s.io images import /tmp/worker.tar && \
  sudo ctr -n k8s.io images import /tmp/frontend.tar && \
  rm /tmp/orchestrator.tar /tmp/worker.tar /tmp/frontend.tar && \
  echo "Images imported successfully!"'

echo ""
echo "========================================="

# Step 3: Deploy to Kubernetes
echo "Step 3: Deploying to Kubernetes..."
export KUBECONFIG="$HOME/.kube/config-nuc"

# Apply Kubernetes manifests
echo "Applying Kubernetes manifests..."
kubectl apply -f k8s/development/

echo ""
echo "Waiting for deployments to be ready..."
kubectl wait --for=condition=ready pod -l app=mimir-aip -n mimir-aip --timeout=300s || true

echo ""
echo "Deployment status:"
kubectl get pods -n mimir-aip

echo ""
echo "Services:"
kubectl get svc -n mimir-aip

echo ""
echo "========================================="
echo "Deployment Complete!"
echo "========================================="
echo ""
echo "Access the frontend:"
echo "  knuc port-forward -n mimir-aip svc/frontend 8081:80"
echo "  Open http://localhost:8081"
echo ""
echo "View logs:"
echo "  knuc logs -n mimir-aip -l component=orchestrator -f"
echo "  knuc logs -n mimir-aip -l app=mimir-worker -f"
echo ""
echo "View status:"
echo "  knuc get all -n mimir-aip"
echo ""
