#!/bin/bash

# full-deploy.sh - Complete deployment pipeline
set -e

# Parse arguments
REMOTE_HOST=""
USE_NUC=false
RUN_TESTS=true

while [[ $# -gt 0 ]]; do
  case $1 in
    --nuc)
      USE_NUC=true
      REMOTE_HOST="ciaran@192.168.0.101"
      shift
      ;;
    --skip-tests)
      RUN_TESTS=false
      shift
      ;;
    *)
      echo "Unknown option: $1"
      echo "Usage: $0 [--nuc] [--skip-tests]"
      exit 1
      ;;
  esac
done

echo "========================================="
if [ "$USE_NUC" = true ]; then
  echo "Mimir AIP - Full NUC Server Deployment"
else
  echo "Mimir AIP - Full Local Deployment"
fi
echo "========================================="
echo ""

# Step 1: Build images
echo "Step 1: Building Docker images..."
if [ "$USE_NUC" = true ]; then
  ./scripts/build-images.sh --remote "$REMOTE_HOST"
else
  ./scripts/build-images.sh
fi

echo ""
echo "========================================="

# Step 2: Deploy to Kubernetes
echo "Step 2: Deploying to Kubernetes..."
if [ "$USE_NUC" = true ]; then
  ./scripts/deploy-local.sh --nuc
else
  ./scripts/deploy-local.sh
fi

echo ""
echo "========================================="

# Step 3: Run integration tests
if [ "$RUN_TESTS" = true ]; then
  echo "Step 3: Running integration tests..."
  echo "Waiting 30 seconds for services to stabilize..."
  sleep 30

  if [ "$USE_NUC" = true ]; then
    KUBECONFIG="$HOME/.kube/config-nuc" ./scripts/run-integration-tests.sh
  else
    ./scripts/run-integration-tests.sh
  fi
  
  echo ""
  echo "========================================="
fi

echo "Deployment Complete!"
echo "========================================="
echo ""
if [ "$USE_NUC" = true ]; then
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
else
  echo "Access the frontend:"
  echo "  kubectl port-forward -n mimir-aip svc/frontend 8081:80"
  echo "  Open http://localhost:8081"
  echo ""
  echo "View logs:"
  echo "  kubectl logs -n mimir-aip -l component=orchestrator -f"
  echo "  kubectl logs -n mimir-aip -l app=mimir-worker -f"
  echo ""
  echo "View status:"
  echo "  kubectl get all -n mimir-aip"
fi
echo ""

# Step 1: Build images
echo "Step 1: Building Docker images..."
./scripts/build-images.sh

echo ""
echo "========================================="

# Step 2: Deploy to Kubernetes
echo "Step 2: Deploying to Kubernetes..."
./scripts/deploy-local.sh

echo ""
echo "========================================="

# Step 3: Run integration tests
echo "Step 3: Running integration tests..."
echo "Waiting 30 seconds for services to stabilize..."
sleep 30

./scripts/run-integration-tests.sh

echo ""
echo "========================================="
echo "Deployment Complete!"
echo "========================================="
echo ""
echo "Access the frontend:"
echo "  kubectl port-forward -n mimir-aip svc/frontend 8081:80"
echo "  Open http://localhost:8081"
echo ""
echo "View logs:"
echo "  kubectl logs -n mimir-aip -l component=orchestrator -f"
echo "  kubectl logs -n mimir-aip -l app=mimir-worker -f"
echo ""
echo "View status:"
echo "  kubectl get all -n mimir-aip"
