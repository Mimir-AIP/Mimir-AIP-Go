#!/bin/bash

# Health check script for Mimir-AIP-Go container
set -e

# Configuration
CONTAINER_NAME="mimir-aip-server"
HEALTH_URL="http://localhost:8080/api/v1/health"
TIMEOUT=${TIMEOUT:-10}
RETRIES=${RETRIES:-3}

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if container is running
check_container() {
    if ! docker ps | grep -q $CONTAINER_NAME; then
        log_error "Container $CONTAINER_NAME is not running"
        return 1
    fi
    
    log_info "Container $CONTAINER_NAME is running"
    return 0
}

# Health check endpoint
check_health() {
    local attempt=1
    
    while [ $attempt -le $RETRIES ]; do
        log_info "Health check attempt $attempt/$RETRIES"
        
        # Check if container responds to health endpoint
        if curl -f -s --max-time $TIMEOUT "$HEALTH_URL" > /dev/null; then
            log_info "Health check passed"
            return 0
        fi
        
        if [ $attempt -lt $RETRIES ]; then
            log_warn "Health check failed, retrying in 5 seconds..."
            sleep 5
        fi
        
        ((attempt++))
    done
    
    log_error "Health check failed after $RETRIES attempts"
    return 1
}

# Check container logs for errors
check_logs() {
    log_info "Checking recent container logs for errors..."
    
    # Get last 50 lines of logs
    local error_count=$(docker logs --tail 50 $CONTAINER_NAME 2>&1 | grep -i "error\|fatal\|panic" | wc -l)
    
    if [ $error_count -gt 0 ]; then
        log_warn "Found $error_count error(s) in recent logs"
        docker logs --tail 10 $CONTAINER_NAME 2>&1 | grep -i "error\|fatal\|panic"
    else
        log_info "No errors found in recent logs"
    fi
}

# Check resource usage
check_resources() {
    log_info "Checking container resource usage..."
    
    # Get container stats
    local stats=$(docker stats --no-stream --format "table {{.CPUPerc}}\t{{.MemUsage}}\t{{.MemPerc}}" $CONTAINER_NAME)
    
    if [ -n "$stats" ]; then
        log_info "Resource usage:"
        echo "$stats"
    else
        log_warn "Could not retrieve resource usage"
    fi
}

# Main health check
main() {
    log_info "Starting health check for $CONTAINER_NAME"
    
    if ! check_container; then
        exit 1
    fi
    
    if ! check_health; then
        exit 1
    fi
    
    check_logs
    check_resources
    
    log_info "Health check completed successfully"
}

# Help function
show_help() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  -c, --container NAME    Container name (default: mimir-aip-server)"
    echo "  -u, --url URL          Health check URL (default: http://localhost:8080/api/v1/health)"
    echo "  -t, --timeout SECONDS   Request timeout (default: 10)"
    echo "  -r, --retries COUNT    Number of retries (default: 3)"
    echo "  -h, --help             Show this help message"
}

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -c|--container)
            CONTAINER_NAME="$2"
            shift 2
            ;;
        -u|--url)
            HEALTH_URL="$2"
            shift 2
            ;;
        -t|--timeout)
            TIMEOUT="$2"
            shift 2
            ;;
        -r|--retries)
            RETRIES="$2"
            shift 2
            ;;
        -h|--help)
            show_help
            exit 0
            ;;
        *)
            log_error "Unknown option: $1"
            show_help
            exit 1
            ;;
    esac
done

# Run main function
main