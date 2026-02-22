#!/bin/bash

# undeploy-local.sh - Remove Mimir AIP from Kubernetes cluster
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

echo "Removing Mimir AIP from Kubernetes cluster..."

kubectl delete -f k8s/development/ || true

echo ""
echo "Waiting for pods to terminate..."
kubectl wait --for=delete pod -l app=mimir-aip -n mimir-aip --timeout=60s || true

echo ""
echo "Mimir AIP has been removed from the cluster"
