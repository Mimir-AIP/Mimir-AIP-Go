#!/bin/bash

# run-integration-tests.sh - Run integration tests against deployed system
set -e

echo "Running integration tests against deployed Mimir AIP..."

# Port forward orchestrator for testing
echo "Setting up port forwarding..."
kubectl port-forward -n mimir-aip svc/orchestrator 8080:8080 &
PF_PID=$!

# Cleanup function
cleanup() {
  echo "Cleaning up port forward..."
  kill $PF_PID 2>/dev/null || true
}
trap cleanup EXIT

# Wait for port forward to be ready
sleep 5

# Run integration tests
echo "Running tests..."
ORCHESTRATOR_URL=http://localhost:8080 go test ./tests/integration/... -v

echo ""
echo "Integration tests complete!"
