#!/bin/bash

set -e

echo "=================================="
echo "Mimir AIP - Quick Start Script"
echo "=================================="
echo ""

# Color codes
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check prerequisites
echo -e "${BLUE}Checking prerequisites...${NC}"

if ! command -v docker &> /dev/null; then
    echo -e "${YELLOW}Docker is not installed. Please install Docker first.${NC}"
    exit 1
fi

if ! command -v kubectl &> /dev/null; then
    echo -e "${YELLOW}kubectl is not installed. Skipping Kubernetes option.${NC}"
    K8S_AVAILABLE=false
else
    K8S_AVAILABLE=true
fi

echo -e "${GREEN}✓ Docker found${NC}"
if [ "$K8S_AVAILABLE" = true ]; then
    echo -e "${GREEN}✓ kubectl found${NC}"
fi

echo ""
echo "Select deployment option:"
echo "1) Docker Compose (Local Development)"
echo "2) Kubernetes (Cluster Deployment)"
echo "3) Build Docker Images Only"
echo "4) Exit"
echo ""
read -p "Enter your choice [1-4]: " choice

case $choice in
    1)
        echo ""
        echo -e "${BLUE}Starting deployment with Docker Compose...${NC}"
        echo ""
        
        # Build and start services
        docker-compose up --build -d
        
        echo ""
        echo -e "${GREEN}✓ Services started successfully!${NC}"
        echo ""
        echo "Access points:"
        echo "  - Frontend: http://localhost:3000"
        echo "  - Orchestrator API: http://localhost:8080"
        echo "  - Health Check: http://localhost:8080/health"
        echo ""
        echo "View logs:"
        echo "  docker-compose logs -f"
        echo ""
        echo "Stop services:"
        echo "  docker-compose down"
        ;;
        
    2)
        if [ "$K8S_AVAILABLE" = false ]; then
            echo -e "${YELLOW}kubectl not available. Cannot deploy to Kubernetes.${NC}"
            exit 1
        fi
        
        echo ""
        echo -e "${BLUE}Building Docker images...${NC}"
        make build-all
        
        echo ""
        echo -e "${BLUE}Deploying to Kubernetes...${NC}"
        make deploy-k8s
        
        echo ""
        echo -e "${GREEN}✓ Deployment complete!${NC}"
        echo ""
        echo "Check status:"
        echo "  make status"
        echo ""
        echo "Access frontend (in a new terminal):"
        echo "  make port-forward-frontend"
        echo "  Then open: http://localhost:3000"
        echo ""
        echo "View logs:"
        echo "  make logs-frontend"
        echo "  make logs-orchestrator"
        ;;
        
    3)
        echo ""
        echo -e "${BLUE}Building Docker images...${NC}"
        
        echo ""
        echo "Building orchestrator..."
        docker build -t mimir-aip/orchestrator:latest -f cmd/orchestrator/Dockerfile .
        
        echo ""
        echo "Building frontend..."
        docker build -t mimir-aip/frontend:latest -f frontend/Dockerfile .
        
        echo ""
        echo -e "${GREEN}✓ Images built successfully!${NC}"
        echo ""
        echo "Images:"
        docker images | grep mimir-aip
        ;;
        
    4)
        echo "Exiting..."
        exit 0
        ;;
        
    *)
        echo -e "${YELLOW}Invalid choice. Exiting.${NC}"
        exit 1
        ;;
esac

echo ""
echo -e "${GREEN}Done!${NC}"
