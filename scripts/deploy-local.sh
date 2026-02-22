#!/bin/bash

# deploy-local.sh - Deploy Mimir AIP to Kubernetes cluster
set -e

# Parse arguments
KUBECONFIG_FILE=""
USE_NUC=false

while [[ $# -gt 0 ]]; do
  case $1 in
    --nuc)
      USE_NUC=true
      KUBECONFIG_FILE="$HOME/.kube/config-nuc"
      shift
      ;;
    --kubeconfig)
      KUBECONFIG_FILE="$2"
      shift 2
      ;;
    *)
      echo "Unknown option: $1"
      echo "Usage: $0 [--nuc] [--kubeconfig <path>]"
      exit 1
      ;;
  esac
done

# Set kubeconfig if specified
if [ -n "$KUBECONFIG_FILE" ]; then
  export KUBECONFIG="$KUBECONFIG_FILE"
  echo "Using kubeconfig: $KUBECONFIG_FILE"
fi

CLUSTER_INFO=$(kubectl cluster-info | head -1)
echo "Deploying Mimir AIP to Kubernetes cluster..."
echo "Cluster: $CLUSTER_INFO"
echo ""

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
if [ "$USE_NUC" = true ]; then
  echo "Deployed to NUC server!"
  echo ""
  echo "To access the frontend:"
  echo "  kubectl --kubeconfig ~/.kube/config-nuc port-forward -n mimir-aip svc/frontend 8081:80"
  echo "  Or use the alias: knuc port-forward -n mimir-aip svc/frontend 8081:80"
  echo ""
  echo "Then open http://localhost:8081 in your browser"
  echo ""
  echo "To access the orchestrator API directly:"
  echo "  knuc port-forward -n mimir-aip svc/orchestrator 8080:8080"
else
  echo "To access the frontend, run:"
  echo "  kubectl port-forward -n mimir-aip svc/frontend 8081:80"
  echo ""
  echo "Then open http://localhost:8081 in your browser"
  echo ""
  echo "To access the orchestrator API directly, run:"
  echo "  kubectl port-forward -n mimir-aip svc/orchestrator 8080:8080"
fi
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
