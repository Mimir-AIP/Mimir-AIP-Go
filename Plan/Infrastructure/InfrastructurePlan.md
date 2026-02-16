# Infrastructure

The Mimir AIP infrastructure consists of three main components deployed as Kubernetes containers: Frontend, Orchestrator, and scalable Workers. All components are containerized and deployed using Kubernetes manifests.

## Component Overview

### Frontend
Single Kubernetes deployment serving the web interface. Provides REST API endpoints for project management, pipeline execution, and system monitoring.

### Orchestrator
Central coordination service that manages projects, schedules jobs, and coordinates between components. Acts as the main API gateway and job dispatcher.

### Workers
Scalable pool of job execution containers. Each worker handles one job at a time (pipeline execution, ML training/inference, digital twin operations) and terminates after completion.

## Development Environment

### Local Development Setup
- Use Rancher Desktop for local Kubernetes cluster
- Deploy all components to local cluster during development
- Use port forwarding for local access to services
- Enable Kubernetes dashboard for monitoring

### Development Deployment Commands
```bash
# Deploy to local Rancher Desktop cluster
kubectl apply -f k8s/development/

# Port forward services for local access
kubectl port-forward svc/orchestrator 8080:8080
kubectl port-forward svc/frontend 3000:3000

# View logs
kubectl logs -f deployment/orchestrator
kubectl logs -f deployment/worker-pool
```

## Production Environment

### Kubernetes Manifests

#### Orchestrator Deployment
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: orchestrator
  labels:
    app: mimir-aip
    component: orchestrator
spec:
  replicas: 1
  selector:
    matchLabels:
      app: mimir-aip
      component: orchestrator
  template:
    metadata:
      labels:
        app: mimir-aip
        component: orchestrator
    spec:
      containers:
      - name: orchestrator
        image: mimir-aip/orchestrator:latest
        ports:
        - containerPort: 8080
          name: http
        env:
        - name: ENVIRONMENT
          value: "production"
        - name: REDIS_URL
          valueFrom:
            secretKeyRef:
              name: redis-secret
              key: url
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: db-secret
              key: url
        resources:
          requests:
            memory: "512Mi"
            cpu: "250m"
          limits:
            memory: "1Gi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
        volumeMounts:
        - name: config-volume
          mountPath: /app/config
      volumes:
      - name: config-volume
        configMap:
          name: orchestrator-config
```

#### Orchestrator Service
```yaml
apiVersion: v1
kind: Service
metadata:
  name: orchestrator
  labels:
    app: mimir-aip
    component: orchestrator
spec:
  selector:
    app: mimir-aip
    component: orchestrator
  ports:
  - name: http
    port: 8080
    targetPort: 8080
  type: ClusterIP
```

#### Worker Deployment (Job-based)
```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: worker-job-{{job_id}}
  labels:
    app: mimir-aip
    component: worker
    job-type: {{job_type}}
spec:
  ttlSecondsAfterFinished: 300
  template:
    spec:
      containers:
      - name: worker
        image: mimir-aip/worker:latest
        env:
        - name: JOB_ID
          value: "{{job_id}}"
        - name: JOB_TYPE
          value: "{{job_type}}"
        - name: ORCHESTRATOR_URL
          value: "http://orchestrator:8080"
        - name: REDIS_URL
          valueFrom:
            secretKeyRef:
              name: redis-secret
              key: url
        resources:
          requests:
            memory: "1Gi"
            cpu: "500m"
          limits:
            memory: "4Gi"
            cpu: "2000m"
        volumeMounts:
        - name: job-data
          mountPath: /app/data
        - name: model-cache
          mountPath: /app/models
      volumes:
      - name: job-data
        emptyDir: {}
      - name: model-cache
        persistentVolumeClaim:
          claimName: model-cache-pvc
      restartPolicy: Never
```

#### Frontend Deployment
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: frontend
  labels:
    app: mimir-aip
    component: frontend
spec:
  replicas: 1
  selector:
    matchLabels:
      app: mimir-aip
      component: frontend
  template:
    metadata:
      labels:
        app: mimir-aip
        component: frontend
    spec:
      containers:
      - name: frontend
        image: mimir-aip/frontend:latest
        ports:
        - containerPort: 3000
          name: http
        env:
        - name: API_URL
          value: "http://orchestrator:8080"
        resources:
          requests:
            memory: "128Mi"
            cpu: "100m"
          limits:
            memory: "256Mi"
            cpu: "200m"
        livenessProbe:
          httpGet:
            path: /
            port: 3000
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /
            port: 3000
          initialDelaySeconds: 5
          periodSeconds: 5
```

#### Frontend Service
```yaml
apiVersion: v1
kind: Service
metadata:
  name: frontend
  labels:
    app: mimir-aip
    component: frontend
spec:
  selector:
    app: mimir-aip
    component: frontend
  ports:
  - name: http
    port: 80
    targetPort: 3000
  type: LoadBalancer
```

### Worker Scaling Architecture

#### Job Queue System
- Redis-based job queue for task distribution
- Job types: pipeline_execution, ml_training, ml_inference, digital_twin_update
- Priority queuing for different job types

#### Scaling Logic
1. Orchestrator receives job request
2. Creates Kubernetes Job manifest with job-specific parameters
3. Submits Job to cluster
4. Monitors Job completion via Kubernetes API
5. Collects results and cleans up completed Jobs

#### Resource Allocation
- Pipeline jobs: 1-2 CPU cores, 2-4GB RAM
- ML training: 4-8 CPU cores, 8-16GB RAM, GPU support
- ML inference: 1-2 CPU cores, 2-4GB RAM
- Digital twin: 2-4 CPU cores, 4-8GB RAM

### Networking & Service Discovery

#### Internal Communication
- Orchestrator exposes REST API on port 8080
- Workers communicate with orchestrator via HTTP
- Frontend proxies API calls to orchestrator
- All services use Kubernetes DNS for discovery

#### External Access
- Frontend service exposed via LoadBalancer
- API endpoints accessible via frontend
- WebSocket support for real-time job monitoring

### Storage Configuration

#### Persistent Volumes
```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: model-cache-pvc
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 100Gi
  storageClassName: standard
```

#### ConfigMaps
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: orchestrator-config
data:
  config.yaml: |
    environment: production
    log_level: info
    job_timeout: 3600
```

### Monitoring & Observability

#### Health Checks
- `/health` endpoint for liveness probes
- `/ready` endpoint for readiness probes
- `/metrics` endpoint for Prometheus metrics

#### Logging
- Structured JSON logging to stdout
- Log aggregation via Kubernetes logging
- Error tracking and alerting

### Configuration Management

#### Environment Variables
- `ENVIRONMENT`: development|staging|production
- `LOG_LEVEL`: debug|info|warn|error
- `REDIS_URL`: Redis connection string
- `DATABASE_URL`: Database connection string

#### Secrets
- Database credentials
- Redis authentication
- API keys for external services

### Security

#### Network Policies
```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: orchestrator-policy
spec:
  podSelector:
    matchLabels:
      component: orchestrator
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - podSelector:
        matchLabels:
          component: frontend
    - podSelector:
        matchLabels:
          component: worker
  egress:
  - to:
    - podSelector:
        matchLabels:
          component: redis
    - podSelector:
          component: database
```

#### RBAC
- Service accounts for each component
- Minimal required permissions
- Pod security standards

### Deployment Process

#### Development
1. Build container images locally
2. Push to local registry (if using Rancher Desktop)
3. Apply Kubernetes manifests
4. Test functionality
5. Debug using kubectl logs/port-forward

#### Production
1. CI/CD pipeline builds images
2. Push to container registry
3. Update Helm chart versions
4. Deploy via GitOps (ArgoCD/Flux)
5. Run integration tests
6. Monitor deployment health

### Scaling Considerations

#### Horizontal Scaling
- Orchestrator: Single replica (stateful, not easily scalable)
- Frontend: Multiple replicas behind load balancer
- Workers: Job-based scaling (unlimited horizontal scaling)

#### Vertical Scaling
- Resource limits based on job type
- Autoscaling based on queue length
- GPU support for ML workloads

#### Distributed Scaling
- Workers can run across multiple nodes
- Support for multi-cluster deployments
- Regional distribution for global deployments
