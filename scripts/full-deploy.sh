#!/bin/bash

# full-deploy.sh - Complete deployment pipeline for local development
set -e

echo "========================================="
echo "Mimir AIP - Full Local Deployment"
echo "========================================="
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
