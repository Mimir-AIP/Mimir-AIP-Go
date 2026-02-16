#!/bin/bash

# build-images.sh - Build Docker images for local deployment
set -e

echo "Building Mimir AIP Docker images..."

# Build orchestrator
echo "Building orchestrator image..."
docker build -f cmd/orchestrator/Dockerfile -t mimir-aip/orchestrator:latest .

# Build worker
echo "Building worker image..."
docker build -f cmd/worker/Dockerfile -t mimir-aip/worker:latest .

# Build frontend
echo "Building frontend image..."
docker build -f frontend/Dockerfile -t mimir-aip/frontend:latest .

echo "All images built successfully!"
echo ""
echo "Images:"
echo "  - mimir-aip/orchestrator:latest"
echo "  - mimir-aip/worker:latest"
echo "  - mimir-aip/frontend:latest"
