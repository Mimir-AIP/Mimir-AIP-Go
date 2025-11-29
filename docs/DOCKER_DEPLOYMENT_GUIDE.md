# Docker Deployment Guide

## Overview
This guide provides step-by-step instructions for deploying Mimir-AIP-Go using Docker containers, covering both single-container and orchestrated deployments.

## Prerequisites

### System Requirements
- **Docker Engine** 20.10+ or Docker Desktop
- **Docker Compose** (optional, for multi-service deployments)
- **Memory**: Minimum 512MB, Recommended 2GB+
- **Storage**: Minimum 1GB for data and logs
- **Network**: Port 8080 available (or custom port)

### Platform Support
- ✅ **Linux** (amd64, arm64)
- ✅ **macOS** (Intel, Apple Silicon)
- ✅ **Windows** (amd64)
- ✅ **Cloud Platforms** (AWS, GCP, Azure)

## Quick Start

### 1. Clone Repository
```bash
git clone https://github.com/Mimir-AIP/Mimir-AIP-Go.git
cd Mimir-AIP-Go
```

### 2. Build Docker Image
```bash
# Using build script (recommended)
./docker/scripts/build.sh

# Or manual build
docker build -f docker/Dockerfile -t mimir-aip:latest .
```

### 3. Run Container
```bash
# Basic deployment
docker run -d \
  --name mimir-aip \
  -p 8080:8080 \
  mimir-aip:latest

# With persistent storage
docker run -d \
  --name mimir-aip \
  -p 8080:8080 \
  -v mimir_data:/app/data \
  -v mimir_logs:/app/logs \
  mimir-aip:latest
```

### 4. Verify Deployment
```bash
# Check container status
docker ps | grep mimir-aip

# Test health endpoint
curl http://localhost:8080/health

# Test API
curl http://localhost:8080/api/v1/pipelines
```

## Configuration Options

### Environment Variables

| Variable | Default | Description |
|----------|----------|-------------|
| `MIMIR_PORT` | 8080 | Server port |
| `MIMIR_LOG_LEVEL` | INFO | Logging level (DEBUG, INFO, WARN, ERROR) |
| `MIMIR_DATA_DIR` | /app/data | Data storage directory |
| `MIMIR_LOG_DIR` | /app/logs | Log file directory |
| `MIMIR_PLUGIN_DIR` | /app/plugins | Plugin directory |
| `MIMIR_JWT_SECRET` | - | JWT signing secret (generate your own) |
| `MIMIR_MAX_JOBS` | 10 | Maximum concurrent jobs |

### Volume Mounts

| Path | Purpose | Recommended |
|------|---------|-------------|
| `/app/data` | Application data | Named volume or bind mount |
| `/app/logs` | Log files | Named volume or bind mount |
| `/app/config` | Configuration files | Named volume with custom config |
| `/app/plugins` | Plugin files | Named volume or bind mount |

## Deployment Scenarios

### Development Environment
```bash
# Using Docker Compose for development
cd docker
docker compose -f docker-compose.dev.yml up

# With hot reload (if air is configured)
docker compose -f docker-compose.dev.yml up --build
```

### Production Environment
```bash
# Production deployment with Docker Compose
cd docker
docker compose -f docker-compose.yml up -d

# Or manual production deployment
docker run -d \
  --name mimir-aip-prod \
  --restart unless-stopped \
  -p 8080:8080 \
  -v mimir_data:/app/data \
  -v mimir_logs:/app/logs \
  -v mimir_config:/app/config \
  -v mimir_plugins:/app/plugins \
  -e MIMIR_LOG_LEVEL=INFO \
  -e MIMIR_JWT_SECRET=your-secret-key \
  --memory=512m \
  --cpus=1.0 \
  mimir-aip:latest
```

### High Availability Deployment
```yaml
# docker-compose.ha.yml
version: '3.8'

services:
  mimir-aip-1:
    image: mimir-aip:latest
    restart: unless-stopped
    ports:
      - "8080:8080"
    volumes:
      - mimir_data:/app/data
      - mimir_logs:/app/logs
    environment:
      - MIMIR_PORT=8080
      - MIMIR_LOG_LEVEL=INFO
    networks:
      - mimir-network

  mimir-aip-2:
    image: mimir-aip:latest
    restart: unless-stopped
    ports:
      - "8081:8080"
    volumes:
      - mimir_data:/app/data
      - mimir_logs:/app/logs
    environment:
      - MIMIR_PORT=8080
      - MIMIR_LOG_LEVEL=INFO
    networks:
      - mimir-network

  nginx:
    image: nginx:alpine
    restart: unless-stopped
    ports:
      - "80:80"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf
    depends_on:
      - mimir-aip-1
      - mimir-aip-2
    networks:
      - mimir-network

volumes:
  mimir_data:
    driver: local
  mimir_logs:
    driver: local

networks:
  mimir-network:
    driver: bridge
```

## Monitoring and Maintenance

### Health Checks
```bash
# Using health check script
./docker/scripts/health-check.sh

# Manual health check
curl -f http://localhost:8080/health

# Container health status
docker ps --format "table {{.Names}}\t{{.Status}}"
```

### Log Management
```bash
# View logs
docker logs mimir-aip

# Follow logs
docker logs -f mimir-aip

# Log rotation (configured in app)
# Logs automatically rotate at 100MB with 5 backups
```

### Performance Monitoring
```bash
# Container resource usage
docker stats mimir-aip

# Detailed inspection
docker inspect mimir-aip

# Memory usage
docker stats --no-stream --format "table {{.Container}}\t{{.MemUsage}}"
```

## Security Considerations

### Container Security
- ✅ **Non-root user**: Container runs as user 65532:65532
- ✅ **Minimal base image**: Uses distroless Debian
- ✅ **Read-only filesystem**: Where possible
- ✅ **Resource limits**: CPU and memory constraints

### Network Security
```bash
# Behind reverse proxy
docker run -d \
  --name mimir-aip \
  -p 127.0.0.1:8080:8080 \
  mimir-aip:latest

# With TLS termination (nginx)
# Configure nginx to handle HTTPS and forward to container
```

### Secrets Management
```bash
# Using Docker secrets (recommended)
echo "your-jwt-secret" | docker secret create mimir-jwt-secret -

docker run -d \
  --name mimir-aip \
  --secret mimir-jwt-secret \
  -e MIMIR_JWT_SECRET_FILE=/run/secrets/mimir-jwt-secret \
  mimir-aip:latest

# Using environment file
docker run -d \
  --name mimir-aip \
  --env-file .env \
  mimir-aip:latest
```

## Backup and Recovery

### Data Backup
```bash
# Backup volumes
docker run --rm \
  -v mimir_data:/data \
  -v $(pwd):/backup \
  alpine tar czf /backup/mimir-data-$(date +%Y%m%d).tar.gz -C /data .

# Backup configuration
docker cp mimir-aip:/app/config ./backup/config-$(date +%Y%m%d)
```

### Disaster Recovery
```bash
# Restore from backup
docker run --rm \
  -v mimir_data:/data \
  -v $(pwd)/backup:/backup \
  alpine tar xzf /backup/mimir-data-20231201.tar.gz -C /data

# Recreate container
docker stop mimir-aip
docker rm mimir-aip
docker run -d \
  --name mimir-aip \
  -v mimir_data:/app/data \
  -p 8080:8080 \
  mimir-aip:latest
```

## Troubleshooting

### Common Issues

#### Container Won't Start
```bash
# Check logs
docker logs mimir-aip

# Common solutions
# 1. Port conflict
docker run -p 8081:8080 mimir-aip:latest

# 2. Permission issues
sudo chown -R 65532:65532 ./data

# 3. Resource limits
docker run --memory=1g mimir-aip:latest
```

#### Health Check Fails
```bash
# Test manually
docker exec mimir-aip curl http://localhost:8080/health

# Check configuration
docker exec mimir-aip cat /app/config.yaml

# Verify networking
docker network ls
docker network inspect bridge
```

#### Performance Issues
```bash
# Check resource usage
docker stats mimir-aip

# Optimize configuration
# 1. Increase memory limit
docker run --memory=2g mimir-aip:latest

# 2. Add CPU limits
docker run --cpus=2.0 mimir-aip:latest

# 3. Profile application
docker exec mimir-aip /app/mimir-aip-server --profile
```

## Scaling Considerations

### Horizontal Scaling
- Use load balancer (nginx, HAProxy, cloud LB)
- Share storage volumes between instances
- Configure session affinity if needed
- Implement distributed caching (Redis)

### Vertical Scaling
- Increase CPU/memory limits
- Add more storage I/O
- Optimize application configuration
- Use performance monitoring

## CI/CD Integration

### GitHub Actions
```yaml
# .github/workflows/docker.yml
name: Build and Deploy Docker

on:
  push:
    branches: [main]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Build Docker image
        run: |
          ./docker/scripts/build.sh -v ${{ github.sha }}
          
      - name: Push to registry
        if: github.ref == 'refs/heads/main'
        run: |
          echo ${{ secrets.DOCKER_PASSWORD }} | docker login -u ${{ secrets.DOCKER_USERNAME }} --password-stdin
          docker push your-registry/mimir-aip:${{ github.sha }}
```

### Automated Deployment
```bash
# Deployment script
#!/bin/bash
set -e

NEW_VERSION=$1
CURRENT_CONTAINER=$(docker ps -q --filter "name=mimir-aip")

echo "Deploying version: $NEW_VERSION"

# Pull new image
docker pull mimir-aip:$NEW_VERSION

# Stop current container
if [ -n "$CURRENT_CONTAINER" ]; then
    docker stop mimir-aip
    docker rm mimir-aip
fi

# Start new container
docker run -d \
  --name mimir-aip \
  --restart unless-stopped \
  -p 8080:8080 \
  -v mimir_data:/app/data \
  -v mimir_logs:/app/logs \
  mimir-aip:$NEW_VERSION

# Health check
sleep 10
./docker/scripts/health-check.sh

echo "Deployment completed successfully"
```

## Production Checklist

### Pre-Deployment
- [ ] Review security configurations
- [ ] Set up monitoring and alerting
- [ ] Prepare backup strategy
- [ ] Test disaster recovery procedures
- [ ] Configure log aggregation
- [ ] Set up performance monitoring

### Post-Deployment
- [ ] Verify health endpoints
- [ ] Test critical API functionality
- [ ] Check resource utilization
- [ ] Validate logging and monitoring
- [ ] Test backup and recovery
- [ ] Document deployment configuration

---

## Support

### Documentation
- [API Reference](./API_REFERENCE.md)
- [Plugin Development Guide](./PLUGIN_DEVELOPMENT_GUIDE.md)
- [Deployment Plan](./DOCKER_DEPLOYMENT_PLAN.md)

### Community
- [GitHub Issues](https://github.com/Mimir-AIP/Mimir-AIP-Go/issues)
- [GitHub Discussions](https://github.com/Mimir-AIP/Mimir-AIP-Go/discussions)
- [Wiki](https://github.com/Mimir-AIP/Mimir-AIP-Go/wiki)

---

**Status**: ✅ **Production-ready with comprehensive Docker deployment support**