#!/bin/bash

# Script to run comprehensive E2E tests against deployed Mimir AIP
# This includes tests for pipelines, ontologies, ML models, and digital twins

set -e

KUBECONFIG="${KUBECONFIG:-$HOME/.kube/config}"
NAMESPACE="mimir-aip"
PORT=8080

echo "Running comprehensive E2E tests against deployed Mimir AIP..."

# Setup port forwarding
echo "Setting up port forwarding..."
kubectl --kubeconfig="$KUBECONFIG" port-forward -n $NAMESPACE svc/orchestrator $PORT:$PORT > /dev/null 2>&1 &
PF_PID=$!

# Cleanup function
cleanup() {
    echo "Cleaning up port forward..."
    kill $PF_PID 2>/dev/null || true
    wait $PF_PID 2>/dev/null || true
}

# Set trap to cleanup on exit
trap cleanup EXIT INT TERM

# Wait for port forward to be ready
echo "Waiting for port forward to be ready..."
sleep 3

# Check if orchestrator is reachable
if ! curl -sf http://localhost:$PORT/health > /dev/null; then
    echo "Error: Orchestrator is not reachable on port $PORT"
    exit 1
fi

echo "Running comprehensive E2E tests..."
go test -v ./tests/integration -run TestE2EComprehensiveWorkflow -timeout 5m

echo ""
echo "Comprehensive E2E tests complete!"
