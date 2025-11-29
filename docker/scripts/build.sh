#!/bin/bash

# Docker build script for Mimir-AIP-Go
set -e

# Configuration
IMAGE_NAME="mimir-aip"
VERSION=${VERSION:-"latest"}
REGISTRY=${REGISTRY:-""}
PLATFORM=${PLATFORM:-"linux/amd64,linux/arm64"}

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if Docker is available
check_docker() {
    if ! command -v docker &> /dev/null; then
        log_error "Docker is not installed or not in PATH"
        exit 1
    fi
    
    if ! docker info &> /dev/null; then
        log_error "Docker daemon is not running"
        exit 1
    fi
    
    log_info "Docker is available and running"
}

# Build Docker image
build_image() {
    log_info "Building Docker image: ${IMAGE_NAME}:${VERSION}"
    
    # Build arguments
    BUILD_ARGS=""
    if [ -n "$REGISTRY" ]; then
        BUILD_ARGS="--build-arg REGISTRY=${REGISTRY}"
    fi
    
    # Build for multiple platforms if specified
    if [[ "$PLATFORM" == *","* ]]; then
        log_info "Building for multiple platforms: $PLATFORM"
        docker buildx build \
            --platform $PLATFORM \
            --tag ${IMAGE_NAME}:${VERSION} \
            --tag ${IMAGE_NAME}:latest \
            ${BUILD_ARGS} \
            -f docker/Dockerfile \
            .
    else
        log_info "Building for single platform: $PLATFORM"
        docker build \
            --build-arg BUILDPLATFORM=$PLATFORM \
            --tag ${IMAGE_NAME}:${VERSION} \
            --tag ${IMAGE_NAME}:latest \
            ${BUILD_ARGS} \
            -f docker/Dockerfile \
            .
    fi
    
    if [ $? -eq 0 ]; then
        log_info "Build completed successfully"
    else
        log_error "Build failed"
        exit 1
    fi
}

# Push to registry if specified
push_image() {
    if [ -n "$REGISTRY" ]; then
        log_info "Pushing image to registry: $REGISTRY"
        
        # Tag with registry
        docker tag ${IMAGE_NAME}:${VERSION} ${REGISTRY}/${IMAGE_NAME}:${VERSION}
        docker tag ${IMAGE_NAME}:latest ${REGISTRY}/${IMAGE_NAME}:latest
        
        # Push images
        docker push ${REGISTRY}/${IMAGE_NAME}:${VERSION}
        docker push ${REGISTRY}/${IMAGE_NAME}:latest
        
        if [ $? -eq 0 ]; then
            log_info "Push completed successfully"
        else
            log_error "Push failed"
            exit 1
        fi
    else
        log_info "No registry specified, skipping push"
    fi
}

# Main execution
main() {
    log_info "Starting Docker build process for Mimir-AIP-Go"
    
    check_docker
    build_image
    push_image
    
    log_info "Docker build process completed"
    
    # Show image info
    docker images | grep ${IMAGE_NAME}
}

# Help function
show_help() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  -v, --version VERSION    Set image version (default: latest)"
    echo "  -r, --registry REGISTRY   Set container registry"
    echo "  -p, --platform PLATFORM  Set build platform(s) (default: linux/amd64,linux/arm64)"
    echo "  -h, --help             Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0                                    # Build with defaults"
    echo "  $0 -v v1.0.0                        # Build with version"
    echo "  $0 -r docker.io/myorg -v v1.0.0      # Build and push to registry"
    echo "  $0 -p linux/amd64                     # Build for single platform"
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -v|--version)
            VERSION="$2"
            shift 2
            ;;
        -r|--registry)
            REGISTRY="$2"
            shift 2
            ;;
        -p|--platform)
            PLATFORM="$2"
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