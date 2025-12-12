#!/bin/bash
#
# Build Script for Mimir AIP Unified Docker Container
# Builds both Go backend and Next.js frontend in a single container
#

set -e  # Exit on error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
IMAGE_NAME="mimir-aip"
TAG="${TAG:-unified}"
DOCKERFILE="Dockerfile.unified"

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Mimir AIP Unified Docker Build${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Check if Docker is installed
if ! command -v docker &> /dev/null; then
    echo -e "${RED}Error: Docker is not installed${NC}"
    echo "Please install Docker from https://docs.docker.com/get-docker/"
    exit 1
fi

# Check if Dockerfile exists
if [ ! -f "$DOCKERFILE" ]; then
    echo -e "${RED}Error: $DOCKERFILE not found${NC}"
    exit 1
fi

echo -e "${YELLOW}Building image: ${IMAGE_NAME}:${TAG}${NC}"
echo -e "${YELLOW}Dockerfile: ${DOCKERFILE}${NC}"
echo ""

# Build the image
echo -e "${BLUE}Step 1/3: Building Docker image...${NC}"
docker build -f "$DOCKERFILE" -t "${IMAGE_NAME}:${TAG}" . || {
    echo -e "${RED}Build failed!${NC}"
    exit 1
}

echo ""
echo -e "${GREEN}✓ Build completed successfully!${NC}"
echo ""

# Show image info
echo -e "${BLUE}Step 2/3: Image information${NC}"
docker images "${IMAGE_NAME}:${TAG}" --format "table {{.Repository}}\t{{.Tag}}\t{{.Size}}\t{{.CreatedAt}}"

echo ""
echo -e "${BLUE}Step 3/3: Next steps${NC}"
echo -e "${GREEN}Image built: ${IMAGE_NAME}:${TAG}${NC}"
echo ""
echo "To run the container:"
echo -e "  ${YELLOW}docker run -d -p 8080:8080 --name mimir-aip-unified ${IMAGE_NAME}:${TAG}${NC}"
echo ""
echo "To run with Docker Compose:"
echo -e "  ${YELLOW}docker-compose -f docker-compose.unified.yml up -d${NC}"
echo ""
echo "To access the application:"
echo -e "  ${GREEN}Frontend:${NC}    http://localhost:8080"
echo -e "  ${GREEN}API:${NC}         http://localhost:8080/api/v1"
echo -e "  ${GREEN}Health:${NC}      http://localhost:8080/health"
echo ""
echo -e "${BLUE}========================================${NC}"
