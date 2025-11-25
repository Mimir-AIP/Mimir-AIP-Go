# Mimir AIP - Deployment Guide

## Docker Deployment

### Prerequisites
- Docker 20.10+ and Docker Compose 2.0+
- Sufficient disk space for logs and data
- Network access for external API calls

### Quick Start with Docker Compose

```bash
# Clone repository
git clone https://github.com/Mimir-AIP/Mimir-AIP-Go.git
cd Mimir-AIP-Go

# Create required directories
mkdir -p config logs data output

# Set environment variables
cp .env.example .env
# Edit .env with your configuration

# Start services
docker-compose up -d
```

### Environment Configuration

Create `.env` file:
```bash
# Server Configuration
MIMIR_SERVER_HOST=0.0.0.0
MIMIR_SERVER_PORT=8080
MIMIR_API_KEY=your-secure-api-key

# Logging
MIMIR_LOG_LEVEL=info
MIMIR_LOG_FORMAT=json

# External Services (Optional)
OPENAI_API_KEY=your-openai-key
ANTHROPIC_API_KEY=your-anthropic-key

# Data Persistence
MIMIR_DATA_DIR=./data
MIMIR_LOG_DIR=./logs
MIMIR_OUTPUT_DIR=./output
```

### Docker Compose Configuration

```yaml
version: '3.8'

services:
  mimir:
    image: mimir-aip:latest
    container_name: mimir-aip
    restart: unless-stopped
    ports:
      - "8080:8080"
    volumes:
      - ./config:/app/config
      - ./data:/app/data
      - ./logs:/app/logs
      - ./output:/app/output
    environment:
      - MIMIR_SERVER_HOST=0.0.0.0
      - MIMIR_SERVER_PORT=8080
      - MIMIR_API_KEY=${MIMIR_API_KEY}
      - MIMIR_LOG_LEVEL=${MIMIR_LOG_LEVEL:-info}
      - MIMIR_LOG_FORMAT=${MIMIR_LOG_FORMAT:-json}
      - OPENAI_API_KEY=${OPENAI_API_KEY}
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s

  # Optional: Redis for caching
  redis:
    image: redis:7-alpine
    container_name: mimir-redis
    restart: unless-stopped
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data
    command: redis-server --appendonly yes

  # Optional: PostgreSQL for persistent storage
  postgres:
    image: postgres:15-alpine
    container_name: mimir-postgres
    restart: unless-stopped
    ports:
      - "5432:5432"
    environment:
      - POSTGRES_DB=mimir
      - POSTGRES_USER=mimir
      - POSTGRES_PASSWORD=${POSTGRES_PASSWORD:-securepassword}
    volumes:
      - postgres_data:/var/lib/postgresql/data

volumes:
  redis_data:
  postgres_data:
```

### Production Dockerfile

```dockerfile
# Multi-stage build for production
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -o mimir ./cmd/mimir

# Production image
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata curl

# Create app user
RUN addgroup -g 1000 mimir && \
    adduser -D -s /bin/sh -u 1000 -G mimir mimir

# Set up directories
WORKDIR /app
RUN mkdir -p config data logs output && \
    chown -R mimir:mimir /app

# Copy binary
COPY --from=builder /app/mimir .
RUN chmod +x mimir

# Switch to non-root user
USER mimir

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=40s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# Expose port
EXPOSE 8080

# Run application
CMD ["./mimir"]
```

### Building and Running

```bash
# Build image
docker build -t mimir-aip:latest .

# Run container
docker run -d \
  --name mimir-aip \
  -p 8080:8080 \
  -v $(pwd)/config:/app/config \
  -v $(pwd)/data:/app/data \
  -v $(pwd)/logs:/app/logs \
  -v $(pwd)/output:/app/output \
  -e MIMIR_API_KEY=your-secure-api-key \
  -e OPENAI_API_KEY=your-openai-key \
  mimir-aip:latest
```

## Kubernetes Deployment

### Namespace and ConfigMap

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: mimir-aip
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: mimir-config
  namespace: mimir-aip
data:
  config.yaml: |
    server:
      host: "0.0.0.0"
      port: 8080
    logging:
      level: "info"
      format: "json"
    plugins:
      directories:
        - "./plugins"
      auto_discovery: true
```

### Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: mimir-aip
  namespace: mimir-aip
spec:
  replicas: 3
  selector:
    matchLabels:
      app: mimir-aip
  template:
    metadata:
      labels:
        app: mimir-aip
    spec:
      containers:
      - name: mimir-aip
        image: mimir-aip:latest
        ports:
        - containerPort: 8080
        env:
        - name: MIMIR_SERVER_HOST
          value: "0.0.0.0"
        - name: MIMIR_SERVER_PORT
          value: "8080"
        - name: MIMIR_API_KEY
          valueFrom:
            secretKeyRef:
              name: mimir-secrets
              key: api-key
        - name: OPENAI_API_KEY
          valueFrom:
            secretKeyRef:
              name: mimir-secrets
              key: openai-key
        resources:
          requests:
            memory: "128Mi"
            cpu: "100m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
        volumeMounts:
        - name: config
          mountPath: /app/config
        - name: data
          mountPath: /app/data
        - name: logs
          mountPath: /app/logs
      volumes:
      - name: config
        configMap:
          name: mimir-config
      - name: data
        persistentVolumeClaim:
          claimName: mimir-data
      - name: logs
        persistentVolumeClaim:
          claimName: mimir-logs
```

### Service

```yaml
apiVersion: v1
kind: Service
metadata:
  name: mimir-aip-service
  namespace: mimir-aip
spec:
  selector:
    app: mimir-aip
  ports:
  - protocol: TCP
    port: 80
    targetPort: 8080
  type: ClusterIP
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: mimir-aip-ingress
  namespace: mimir-aip
spec:
  rules:
  - host: mimir.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: mimir-aip-service
            port:
              number: 80
```

## Monitoring and Scaling

### Horizontal Pod Autoscaler

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: mimir-aip-hpa
  namespace: mimir-aip
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: mimir-aip
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
```

### Prometheus Monitoring

```yaml
apiVersion: v1
kind: ServiceMonitor
metadata:
  name: mimir-aip
  namespace: mimir-aip
spec:
  selector:
    matchLabels:
      app: mimir-aip
  endpoints:
  - port: metrics
    interval: 30s
    path: /metrics
```

## Security Considerations

### Secrets Management

```bash
# Create Kubernetes secrets
kubectl create secret generic mimir-secrets \
  --from-literal=api-key=your-secure-api-key \
  --from-literal=openai-key=your-openai-key \
  --namespace=mimir-aip
```

### Network Policies

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: mimir-aip-netpol
  namespace: mimir-aip
spec:
  podSelector:
    matchLabels:
      app: mimir-aip
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: ingress-nginx
    ports:
    - protocol: TCP
      port: 8080
  egress:
  - to: []
    ports:
    - protocol: TCP
      port: 443  # HTTPS for external APIs
    - protocol: TCP
      port: 80   # HTTP for external APIs
```

## Troubleshooting

### Common Issues

1. **Container fails to start**
   - Check environment variables in .env file
   - Verify volume mounts exist
   - Review container logs: `docker logs mimir-aip`

2. **Health check failures**
   - Ensure port 8080 is accessible
   - Check API key configuration
   - Verify plugin loading

3. **Performance issues**
   - Monitor resource usage
   - Check log levels
   - Review plugin execution times

### Logs and Debugging

```bash
# View container logs
docker logs -f mimir-aip

# Kubernetes logs
kubectl logs -f deployment/mimir-aip -n mimir-aip

# Debug with elevated logging
docker run -e MIMIR_LOG_LEVEL=debug mimir-aip:latest
```

## Production Checklist

- [ ] Environment variables configured
- [ ] Secrets management in place
- [ ] Resource limits set appropriately
- [ ] Health checks configured
- [ ] Monitoring and alerting enabled
- [ ] Backup strategy for data
- [ ] SSL/TLS termination configured
- [ ] Network policies applied
- [ ] Autoscaling configured
- [ ] Log rotation configured
- [ ] Security scanning completed