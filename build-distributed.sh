#!/bin/bash
#
# Build Script for Mimir AIP Distributed Docker Containers
# Builds separate containers for frontend, backend, and workers
#

set -e  # Exit on error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
IMAGE_PREFIX="mimir-aip"
TAG="${TAG:-latest}"

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Mimir AIP Distributed Docker Build${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Check if Docker is installed
if ! command -v docker &> /dev/null; then
    echo -e "${RED}Error: Docker is not installed${NC}"
    echo "Please install Docker from https://docs.docker.com/get-docker/"
    exit 1
fi

# Check if docker-compose is installed
if ! command -v docker-compose &> /dev/null; then
    echo -e "${YELLOW}Warning: docker-compose is not installed${NC}"
    echo "You can still build individual images, but won't be able to use docker-compose"
fi

echo -e "${YELLOW}Building distributed architecture images${NC}"
echo -e "${YELLOW}Tag: ${TAG}${NC}"
echo ""

# Build Backend
echo -e "${BLUE}Step 1/3: Building Backend image...${NC}"
docker build -f Dockerfile.backend -t "${IMAGE_PREFIX}:backend-${TAG}" . || {
    echo -e "${RED}Backend build failed!${NC}"
    exit 1
}
echo -e "${GREEN}✓ Backend built successfully${NC}"
echo ""

# Build Frontend
echo -e "${BLUE}Step 2/3: Building Frontend image...${NC}"
docker build -f Dockerfile.frontend -t "${IMAGE_PREFIX}:frontend-${TAG}" . || {
    echo -e "${RED}Frontend build failed!${NC}"
    exit 1
}
echo -e "${GREEN}✓ Frontend built successfully${NC}"
echo ""

# Build Worker
echo -e "${BLUE}Step 3/3: Building Worker image...${NC}"
docker build -f Dockerfile.worker -t "${IMAGE_PREFIX}:worker-${TAG}" . || {
    echo -e "${RED}Worker build failed!${NC}"
    exit 1
}
echo -e "${GREEN}✓ Worker built successfully${NC}"
echo ""

# Show image info
echo -e "${BLUE}Image information:${NC}"
docker images "${IMAGE_PREFIX}:*-${TAG}" --format "table {{.Repository}}\t{{.Tag}}\t{{.Size}}\t{{.CreatedAt}}"

echo ""
echo -e "${GREEN}✓ All images built successfully!${NC}"
echo ""
echo -e "${BLUE}Next steps:${NC}"
echo ""
echo "To start the distributed architecture:"
echo -e "  ${YELLOW}docker-compose -f docker-compose.distributed.yml up -d${NC}"
echo ""
echo "To scale workers (e.g., to 3 instances):"
echo -e "  ${YELLOW}docker-compose -f docker-compose.distributed.yml up -d --scale worker=3${NC}"
echo ""
echo "To access the application:"
echo -e "  ${GREEN}Frontend:${NC}    http://localhost:3000"
echo -e "  ${GREEN}Backend API:${NC} http://localhost:8080/api/v1"
echo -e "  ${GREEN}Health:${NC}      http://localhost:8080/health"
echo -e "  ${GREEN}Redis:${NC}       localhost:6379"
echo ""
echo -e "${BLUE}========================================${NC}"
