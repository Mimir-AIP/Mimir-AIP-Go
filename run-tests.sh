#!/bin/bash

# Mimir AIP Test Runner
# Supports both Docker and background process modes

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default values
MODE="${MODE:-background}"  # 'docker' or 'background'
TEST_TYPE="${TEST_TYPE:-all}"  # 'unit', 'integration', 'e2e', or 'all'
VERBOSE="${VERBOSE:-false}"

# Ports
BACKEND_PORT=8080
FRONTEND_PORT=3000

# Functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

# Cleanup function
cleanup() {
    log_info "Cleaning up..."
    
    if [ "$MODE" = "background" ]; then
        # Kill background processes
        if [ -n "$BACKEND_PID" ]; then
            log_info "Stopping backend (PID: $BACKEND_PID)"
            kill $BACKEND_PID 2>/dev/null || true
            wait $BACKEND_PID 2>/dev/null || true
        fi
        
        if [ -n "$FRONTEND_PID" ]; then
            log_info "Stopping frontend (PID: $FRONTEND_PID)"
            kill $FRONTEND_PID 2>/dev/null || true
            wait $FRONTEND_PID 2>/dev/null || true
        fi
    elif [ "$MODE" = "docker" ]; then
        log_info "Stopping Docker containers"
        docker-compose -f docker-compose.unified.yml down 2>/dev/null || true
    fi
    
    log_success "Cleanup complete"
}

# Set trap to cleanup on exit
trap cleanup EXIT

# Wait for service to be ready
wait_for_service() {
    local url=$1
    local name=$2
    local max_attempts=${3:-30}
    local attempt=0
    
    log_info "Waiting for $name to be ready at $url..."
    
    while [ $attempt -lt $max_attempts ]; do
        if curl -s "$url" > /dev/null 2>&1; then
            log_success "$name is ready!"
            return 0
        fi
        
        attempt=$((attempt + 1))
        sleep 1
    done
    
    log_error "$name failed to start after ${max_attempts}s"
    return 1
}

# Run unit tests
run_unit_tests() {
    log_info "Running unit tests..."
    
    if [ "$VERBOSE" = "true" ]; then
        go test ./utils ./pipelines/Input -v -run "TestExecute|TestParse|TestRun|TestCSV" 2>&1 | tee test-output-unit.log
    else
        go test ./utils ./pipelines/Input -run "TestExecute|TestParse|TestRun|TestCSV" 2>&1 | tee test-output-unit.log
    fi
    
    local exit_code=$?
    
    if [ $exit_code -eq 0 ]; then
        log_success "Unit tests passed!"
    else
        log_error "Unit tests failed!"
    fi
    
    return $exit_code
}

# Run integration tests
run_integration_tests() {
    log_info "Running integration tests..."
    
    if [ "$VERBOSE" = "true" ]; then
        go test -run "TestHappyPath" -v 2>&1 | tee test-output-integration.log
    else
        go test -run "TestHappyPath" 2>&1 | tee test-output-integration.log
    fi
    
    local exit_code=$?
    
    if [ $exit_code -eq 0 ]; then
        log_success "Integration tests passed!"
    else
        log_error "Integration tests failed!"
    fi
    
    return $exit_code
}

# Run E2E tests
run_e2e_tests() {
    log_info "Running E2E tests..."
    
    cd mimir-aip-frontend
    
    # Install Playwright browsers if needed
    if [ ! -d "node_modules/@playwright/test" ]; then
        log_info "Installing Playwright..."
        npm install
        npx playwright install chromium
    fi
    
    if [ "$VERBOSE" = "true" ]; then
        npx playwright test --reporter=line 2>&1 | tee ../test-output-e2e.log
    else
        npx playwright test 2>&1 | tee ../test-output-e2e.log
    fi
    
    local exit_code=$?
    
    cd ..
    
    if [ $exit_code -eq 0 ]; then
        log_success "E2E tests passed!"
    else
        log_error "E2E tests failed!"
    fi
    
    return $exit_code
}

# Start services in background mode
start_background_services() {
    log_info "Starting services in background mode..."
    
    # Start backend
    log_info "Starting Go backend on port $BACKEND_PORT..."
    go run . --server $BACKEND_PORT > backend.log 2>&1 &
    BACKEND_PID=$!
    log_info "Backend started with PID: $BACKEND_PID"
    
    # Wait for backend to be ready
    if ! wait_for_service "http://localhost:$BACKEND_PORT/health" "Backend" 60; then
        log_error "Backend failed to start. Check backend.log"
        cat backend.log
        return 1
    fi
    
    # Start frontend
    log_info "Starting frontend on port $FRONTEND_PORT..."
    cd mimir-aip-frontend
    npm run dev > ../frontend.log 2>&1 &
    FRONTEND_PID=$!
    cd ..
    log_info "Frontend started with PID: $FRONTEND_PID"
    
    # Wait for frontend to be ready
    if ! wait_for_service "http://localhost:$FRONTEND_PORT" "Frontend" 120; then
        log_error "Frontend failed to start. Check frontend.log"
        cat frontend.log
        return 1
    fi
    
    log_success "All services started successfully!"
    return 0
}

# Start services in Docker mode
start_docker_services() {
    log_info "Starting services in Docker mode..."
    
    # Build and start
    log_info "Building and starting Docker containers..."
    docker-compose -f docker-compose.unified.yml up --build -d 2>&1 | tee docker-build.log
    
    if [ ${PIPESTATUS[0]} -ne 0 ]; then
        log_error "Docker build/start failed! Check docker-build.log"
        return 1
    fi
    
    # Wait for services to be ready
    if ! wait_for_service "http://localhost:$BACKEND_PORT/health" "Mimir API" 120; then
        log_error "Docker service failed to start"
        docker-compose -f docker-compose.unified.yml logs 2>&1 | tail -50
        return 1
    fi
    
    log_success "Docker services started successfully!"
    return 0
}

# Print usage
usage() {
    cat << EOF
Mimir AIP Test Runner

Usage: $0 [OPTIONS]

Options:
    -m, --mode MODE         Test mode: 'docker' or 'background' (default: background)
    -t, --type TYPE         Test type: 'unit', 'integration', 'e2e', or 'all' (default: all)
    -v, --verbose           Enable verbose output
    -h, --help              Show this help message

Examples:
    # Run all tests with background processes (fastest for development)
    $0

    # Run only unit tests
    $0 -t unit

    # Run all tests using Docker (most realistic)
    $0 -m docker

    # Run integration tests with verbose output
    $0 -t integration -v

    # Run E2E tests in Docker mode
    $0 -m docker -t e2e

Environment Variables:
    MODE        Same as --mode
    TEST_TYPE   Same as --type
    VERBOSE     Same as --verbose (true/false)

EOF
}

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -m|--mode)
            MODE="$2"
            shift 2
            ;;
        -t|--type)
            TEST_TYPE="$2"
            shift 2
            ;;
        -v|--verbose)
            VERBOSE="true"
            shift
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            log_error "Unknown option: $1"
            usage
            exit 1
            ;;
    esac
done

# Validate mode
if [[ "$MODE" != "docker" && "$MODE" != "background" ]]; then
    log_error "Invalid mode: $MODE. Must be 'docker' or 'background'"
    exit 1
fi

# Validate test type
if [[ "$TEST_TYPE" != "unit" && "$TEST_TYPE" != "integration" && "$TEST_TYPE" != "e2e" && "$TEST_TYPE" != "all" ]]; then
    log_error "Invalid test type: $TEST_TYPE. Must be 'unit', 'integration', 'e2e', or 'all'"
    exit 1
fi

# Main execution
log_info "Starting Mimir AIP Test Runner"
log_info "Mode: $MODE"
log_info "Test Type: $TEST_TYPE"
log_info "Verbose: $VERBOSE"
echo ""

EXIT_CODE=0

# Run tests based on type
if [ "$TEST_TYPE" = "unit" ] || [ "$TEST_TYPE" = "all" ]; then
    echo "========================================"
    log_info "RUNNING UNIT TESTS"
    echo "========================================"
    if ! run_unit_tests; then
        EXIT_CODE=1
    fi
    echo ""
fi

if [ "$TEST_TYPE" = "integration" ] || [ "$TEST_TYPE" = "all" ]; then
    echo "========================================"
    log_info "RUNNING INTEGRATION TESTS"
    echo "========================================"
    
    # Integration tests don't need running services - they create their own
    if ! run_integration_tests; then
        EXIT_CODE=1
    fi
    echo ""
fi

if [ "$TEST_TYPE" = "e2e" ] || [ "$TEST_TYPE" = "all" ]; then
    echo "========================================"
    log_info "RUNNING E2E TESTS"
    echo "========================================"
    
    # Start services
    if [ "$MODE" = "docker" ]; then
        if ! start_docker_services; then
            EXIT_CODE=1
        fi
    else
        if ! start_background_services; then
            EXIT_CODE=1
        fi
    fi
    
    # Run E2E tests if services started
    if [ $EXIT_CODE -eq 0 ]; then
        if ! run_e2e_tests; then
            EXIT_CODE=1
        fi
    fi
    echo ""
fi

# Summary
echo "========================================"
if [ $EXIT_CODE -eq 0 ]; then
    log_success "ALL TESTS PASSED!"
else
    log_error "SOME TESTS FAILED!"
    
    # Show relevant logs
    if [ -f "test-output-unit.log" ]; then
        echo ""
        log_info "Unit test output: test-output-unit.log"
    fi
    if [ -f "test-output-integration.log" ]; then
        echo ""
        log_info "Integration test output: test-output-integration.log"
    fi
    if [ -f "test-output-e2e.log" ]; then
        echo ""
        log_info "E2E test output: test-output-e2e.log"
    fi
fi
echo "========================================"

exit $EXIT_CODE