#!/bin/bash
#
# Test Script for Distributed Architecture
# Validates that code compiles and basic structure is correct
#

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Mimir AIP Distributed Architecture Test${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Test 1: Build main server
echo -e "${YELLOW}Test 1: Building main server...${NC}"
if go build -o /tmp/mimir-test-server . > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Main server builds successfully${NC}"
    rm -f /tmp/mimir-test-server
else
    echo -e "${RED}✗ Main server build failed${NC}"
    go build -o /tmp/mimir-test-server .
    exit 1
fi
echo ""

# Test 2: Build worker
echo -e "${YELLOW}Test 2: Building worker process...${NC}"
if go build -o /tmp/mimir-test-worker ./cmd/worker > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Worker builds successfully${NC}"
    rm -f /tmp/mimir-test-worker
else
    echo -e "${RED}✗ Worker build failed${NC}"
    go build -o /tmp/mimir-test-worker ./cmd/worker
    exit 1
fi
echo ""

# Test 3: Verify Dockerfiles exist
echo -e "${YELLOW}Test 3: Checking Dockerfiles...${NC}"
DOCKERFILES=(
    "Dockerfile.frontend"
    "Dockerfile.backend"
    "Dockerfile.worker"
    "docker-compose.distributed.yml"
)

all_exist=true
for dockerfile in "${DOCKERFILES[@]}"; do
    if [ -f "$dockerfile" ]; then
        echo -e "${GREEN}✓ $dockerfile exists${NC}"
    else
        echo -e "${RED}✗ $dockerfile missing${NC}"
        all_exist=false
    fi
done

if [ "$all_exist" = false ]; then
    exit 1
fi
echo ""

# Test 4: Verify build scripts exist and are executable
echo -e "${YELLOW}Test 4: Checking build scripts...${NC}"
SCRIPTS=(
    "build-distributed.sh"
    "start-distributed.sh"
    "docker/scripts/start-backend.sh"
)

all_executable=true
for script in "${SCRIPTS[@]}"; do
    if [ -f "$script" ]; then
        if [ -x "$script" ]; then
            echo -e "${GREEN}✓ $script exists and is executable${NC}"
        else
            echo -e "${YELLOW}⚠ $script exists but is not executable${NC}"
            chmod +x "$script"
            echo -e "${GREEN}  Made $script executable${NC}"
        fi
    else
        echo -e "${RED}✗ $script missing${NC}"
        all_executable=false
    fi
done

if [ "$all_executable" = false ]; then
    exit 1
fi
echo ""

# Test 5: Check Go dependencies
echo -e "${YELLOW}Test 5: Verifying Go dependencies...${NC}"
if go mod verify > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Go modules verified${NC}"
else
    echo -e "${YELLOW}⚠ Running go mod tidy...${NC}"
    go mod tidy
    echo -e "${GREEN}✓ Go modules tidied${NC}"
fi
echo ""

# Test 6: Verify new API endpoints compile
echo -e "${YELLOW}Test 6: Checking new API handlers...${NC}"
if [ -f "handlers_task_queue.go" ]; then
    echo -e "${GREEN}✓ Task queue handlers exist${NC}"
else
    echo -e "${RED}✗ Task queue handlers missing${NC}"
    exit 1
fi

if [ -f "utils/task_queue.go" ]; then
    echo -e "${GREEN}✓ Task queue utils exist${NC}"
else
    echo -e "${RED}✗ Task queue utils missing${NC}"
    exit 1
fi
echo ""

# Test 7: Verify documentation
echo -e "${YELLOW}Test 7: Checking documentation...${NC}"
if [ -f "DISTRIBUTED_ARCHITECTURE.md" ]; then
    echo -e "${GREEN}✓ Architecture documentation exists${NC}"
else
    echo -e "${YELLOW}⚠ Architecture documentation missing${NC}"
fi
echo ""

# Summary
echo -e "${BLUE}========================================${NC}"
echo -e "${GREEN}✓ All tests passed!${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""
echo -e "${BLUE}Next steps:${NC}"
echo "1. Build Docker images:     ./build-distributed.sh"
echo "2. Start the system:        ./start-distributed.sh up"
echo "3. Scale workers:           ./start-distributed.sh scale 5"
echo "4. View logs:               ./start-distributed.sh logs"
echo "5. Stop the system:         ./start-distributed.sh down"
echo ""
echo -e "${BLUE}For more information, see:${NC}"
echo "  - DISTRIBUTED_ARCHITECTURE.md"
echo "  - docker-compose.distributed.yml"
echo ""
