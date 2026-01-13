#!/bin/bash
#
# Start Script for Mimir AIP Distributed Architecture
# Starts frontend, backend, workers, and Redis
#

set -e  # Exit on error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
COMPOSE_FILE="docker-compose.distributed.yml"
WORKERS="${WORKERS:-2}"  # Default 2 workers

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Mimir AIP Distributed Architecture${NC}"
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
    echo -e "${RED}Error: docker-compose is not installed${NC}"
    echo "Please install docker-compose"
    exit 1
fi

# Check if compose file exists
if [ ! -f "$COMPOSE_FILE" ]; then
    echo -e "${RED}Error: $COMPOSE_FILE not found${NC}"
    exit 1
fi

echo -e "${YELLOW}Starting Mimir AIP distributed architecture${NC}"
echo -e "${YELLOW}Workers: ${WORKERS}${NC}"
echo ""

# Parse command line arguments
CMD="${1:-up}"

case "$CMD" in
    up|start)
        echo -e "${BLUE}Starting services...${NC}"
        docker-compose -f "$COMPOSE_FILE" up -d --scale worker="$WORKERS"
        
        echo ""
        echo -e "${GREEN}✓ Services started successfully!${NC}"
        echo ""
        echo -e "${BLUE}Service status:${NC}"
        docker-compose -f "$COMPOSE_FILE" ps
        echo ""
        echo -e "${BLUE}Access points:${NC}"
        echo -e "  ${GREEN}Frontend:${NC}    http://localhost:3000"
        echo -e "  ${GREEN}Backend API:${NC} http://localhost:8080/api/v1"
        echo -e "  ${GREEN}Health:${NC}      http://localhost:8080/health"
        echo -e "  ${GREEN}Redis:${NC}       localhost:6379"
        echo ""
        echo -e "${BLUE}View logs:${NC}"
        echo -e "  ${YELLOW}docker-compose -f $COMPOSE_FILE logs -f${NC}"
        echo ""
        echo -e "${BLUE}Scale workers:${NC}"
        echo -e "  ${YELLOW}docker-compose -f $COMPOSE_FILE up -d --scale worker=5${NC}"
        ;;
    
    down|stop)
        echo -e "${BLUE}Stopping services...${NC}"
        docker-compose -f "$COMPOSE_FILE" down
        echo -e "${GREEN}✓ Services stopped${NC}"
        ;;
    
    restart)
        echo -e "${BLUE}Restarting services...${NC}"
        docker-compose -f "$COMPOSE_FILE" restart
        echo -e "${GREEN}✓ Services restarted${NC}"
        ;;
    
    logs)
        docker-compose -f "$COMPOSE_FILE" logs -f
        ;;
    
    ps|status)
        docker-compose -f "$COMPOSE_FILE" ps
        ;;
    
    scale)
        if [ -z "$2" ]; then
            echo -e "${RED}Error: Please specify number of workers${NC}"
            echo "Usage: $0 scale <number>"
            exit 1
        fi
        echo -e "${BLUE}Scaling workers to $2...${NC}"
        docker-compose -f "$COMPOSE_FILE" up -d --scale worker="$2"
        echo -e "${GREEN}✓ Scaled to $2 workers${NC}"
        docker-compose -f "$COMPOSE_FILE" ps
        ;;
    
    build)
        echo -e "${BLUE}Building images...${NC}"
        docker-compose -f "$COMPOSE_FILE" build
        echo -e "${GREEN}✓ Images built${NC}"
        ;;
    
    clean)
        echo -e "${BLUE}Cleaning up...${NC}"
        docker-compose -f "$COMPOSE_FILE" down -v
        echo -e "${GREEN}✓ Cleaned up (including volumes)${NC}"
        ;;
    
    help|--help|-h)
        echo "Mimir AIP Distributed Architecture Manager"
        echo ""
        echo "Usage: $0 [command]"
        echo ""
        echo "Commands:"
        echo "  up, start      Start all services (default)"
        echo "  down, stop     Stop all services"
        echo "  restart        Restart all services"
        echo "  logs           View logs (follow mode)"
        echo "  ps, status     Show service status"
        echo "  scale <n>      Scale workers to n instances"
        echo "  build          Build all images"
        echo "  clean          Stop and remove all volumes"
        echo "  help           Show this help message"
        echo ""
        echo "Environment variables:"
        echo "  WORKERS        Number of worker instances (default: 2)"
        echo ""
        echo "Examples:"
        echo "  $0 up                    # Start with 2 workers"
        echo "  WORKERS=5 $0 up          # Start with 5 workers"
        echo "  $0 scale 3               # Scale to 3 workers"
        echo "  $0 logs                  # View logs"
        ;;
    
    *)
        echo -e "${RED}Unknown command: $CMD${NC}"
        echo "Use '$0 help' for usage information"
        exit 1
        ;;
esac

echo -e "${BLUE}========================================${NC}"
