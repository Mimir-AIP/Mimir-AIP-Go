#!/bin/bash

# deploy-local.sh - Deploy Mimir AIP to local Rancher Desktop
set -e

echo "Deploying Mimir AIP to local Kubernetes cluster..."

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
echo "To access the frontend, run:"
echo "  kubectl port-forward -n mimir-aip svc/frontend 8081:80"
echo ""
echo "Then open http://localhost:8081 in your browser"
echo ""
echo "To access the orchestrator API directly, run:"
echo "  kubectl port-forward -n mimir-aip svc/orchestrator 8080:8080"
