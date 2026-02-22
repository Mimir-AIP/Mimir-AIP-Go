#!/bin/bash

# build-images.sh - Build Docker images and optionally push to remote server
set -e

# Parse arguments
REMOTE_HOST=""
while [[ $# -gt 0 ]]; do
  case $1 in
    --remote)
      REMOTE_HOST="$2"
      shift 2
      ;;
    *)
      echo "Unknown option: $1"
      echo "Usage: $0 [--remote <user@host>]"
      exit 1
      ;;
  esac
done

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

# If remote host specified, save and transfer images
if [ -n "$REMOTE_HOST" ]; then
  echo ""
  echo "Transferring images to remote server: $REMOTE_HOST..."
  
  # Save images to tar files
  echo "Saving orchestrator image..."
  docker save mimir-aip/orchestrator:latest | gzip > /tmp/orchestrator.tar.gz
  
  echo "Saving worker image..."
  docker save mimir-aip/worker:latest | gzip > /tmp/worker.tar.gz
  
  echo "Saving frontend image..."
  docker save mimir-aip/frontend:latest | gzip > /tmp/frontend.tar.gz
  
  # Transfer images to remote server
  echo "Transferring images to $REMOTE_HOST..."
  scp /tmp/orchestrator.tar.gz /tmp/worker.tar.gz /tmp/frontend.tar.gz "$REMOTE_HOST:/tmp/"
  
  # Load images on remote server
  echo "Loading images on remote server..."
  ssh "$REMOTE_HOST" "gunzip -c /tmp/orchestrator.tar.gz | sudo k3s ctr images import - && \
                       gunzip -c /tmp/worker.tar.gz | sudo k3s ctr images import - && \
                       gunzip -c /tmp/frontend.tar.gz | sudo k3s ctr images import - && \
                       rm /tmp/orchestrator.tar.gz /tmp/worker.tar.gz /tmp/frontend.tar.gz"
  
  # Cleanup local tar files
  rm /tmp/orchestrator.tar.gz /tmp/worker.tar.gz /tmp/frontend.tar.gz
  
  echo "Images successfully transferred and loaded on remote server!"
fi
