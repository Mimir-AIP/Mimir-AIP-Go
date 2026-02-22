#!/bin/bash

# run-e2e-tests.sh - Run E2E tests against deployed system
set -e

echo "Running E2E tests against deployed Mimir AIP..."

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

# Run E2E tests
echo "Running E2E tests..."
ORCHESTRATOR_URL=http://localhost:8080 go test -v ./tests/integration -run "TestE2E.*" -timeout 5m

echo ""
echo "E2E tests complete!"
