#!/bin/bash

# undeploy-local.sh - Remove Mimir AIP from local Kubernetes cluster
set -e

echo "Removing Mimir AIP from local Kubernetes cluster..."

kubectl delete -f k8s/development/ || true

echo ""
echo "Waiting for pods to terminate..."
kubectl wait --for=delete pod -l app=mimir-aip -n mimir-aip --timeout=60s || true

echo ""
echo "Mimir AIP has been removed from the cluster"
