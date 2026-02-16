# Scalable Workers

Scalable workers are on-demand Kubernetes Job instances that execute specific tasks (pipeline runs, ML training/inference, digital twin updates) and automatically terminate after completion. Workers are spawned dynamically based on job queue demand and are designed for horizontal scaling across the cluster.

## Worker Architecture

### Worker Types
- **Pipeline Worker**: Executes data ingestion and processing pipelines
- **ML Training Worker**: Handles machine learning model training jobs
- **ML Inference Worker**: Performs model inference and prediction tasks
- **Digital Twin Worker**: Updates and maintains digital twin states

### Worker Lifecycle
1. **Queued**: Job submitted to Redis queue with task specification
2. **Scheduled**: Orchestrator creates Kubernetes Job manifest
3. **Spawned**: Kubernetes schedules Job pod with appropriate resources
4. **Executing**: Worker downloads task data, executes job, uploads results
5. **Completed**: Job pod terminates, results stored, cleanup performed

## Worker Spawning Process

### Job Submission
```typescript
// Orchestrator API endpoint
POST /api/jobs
{
  "type": "pipeline_execution",
  "pipeline_id": "sensor-data-pipeline",
  "project_id": "manufacturing-project",
  "parameters": {
    "input_data": "s3://bucket/sensor-data.csv",
    "output_format": "cir"
  },
  "resource_requirements": {
    "cpu": "2",
    "memory": "4Gi",
    "gpu": false
  }
}
```

### Queue Storage
Jobs are stored in Redis with structured metadata:
```json
{
  "job_id": "job-12345",
  "type": "pipeline_execution",
  "status": "queued",
  "priority": 1,
  "submitted_at": "2026-02-16T16:45:00Z",
  "project_id": "manufacturing-project",
  "task_spec": {
    "pipeline_id": "sensor-data-pipeline",
    "parameters": {...}
  },
  "resource_requirements": {
    "cpu": "2",
    "memory": "4Gi"
  },
  "data_access": {
    "input_datasets": ["s3://bucket/sensor-data.csv"],
    "output_location": "s3://bucket/results/",
    "storage_credentials": "project-storage-secret"
  }
}
```

### Kubernetes Job Creation
Orchestrator generates Job manifest based on job requirements:
```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: worker-job-12345
  labels:
    app: mimir-worker
    job-type: pipeline_execution
    job-id: job-12345
spec:
  ttlSecondsAfterFinished: 300
  template:
    spec:
      serviceAccountName: worker-service-account
      containers:
      - name: worker
        image: mimir/worker:latest
        env:
        - name: JOB_ID
          value: "job-12345"
        - name: JOB_TYPE
          value: "pipeline_execution"
        - name: ORCHESTRATOR_URL
          value: "http://orchestrator:8080"
        - name: REDIS_URL
          valueFrom:
            secretKeyRef:
              name: redis-secret
              key: url
        - name: STORAGE_ACCESS_TOKEN
          valueFrom:
            secretKeyRef:
              name: storage-secret
              key: token
        resources:
          requests:
            cpu: "2"
            memory: "4Gi"
          limits:
            cpu: "4"
            memory: "8Gi"
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

## Task and Data Provisioning

### Data Access Patterns

#### Input Data Transfer
Workers download required data at startup:
```bash
# Worker initialization script
#!/bin/bash

# Download input datasets
for dataset in "${INPUT_DATASETS[@]}"; do
  aws s3 cp "$dataset" /app/data/input/
done

# Download model artifacts if needed
if [ "$JOB_TYPE" = "ml_inference" ]; then
  aws s3 sync "s3://models/$MODEL_ID/" /app/models/
fi

# Download pipeline configuration
curl -H "Authorization: Bearer $ACCESS_TOKEN" \
     "$ORCHESTRATOR_URL/api/pipelines/$PIPELINE_ID/config" \
     -o /app/config/pipeline.yaml
```

#### Secure Credential Injection
Credentials are injected via Kubernetes secrets:
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: job-12345-secrets
type: Opaque
data:
  aws_access_key: <base64-encoded>
  aws_secret_key: <base64-encoded>
  database_url: <base64-encoded>
```

#### Output Data Handling
Results are uploaded before job completion:
```bash
# Worker completion script
#!/bin/bash

# Upload results
aws s3 sync /app/data/output/ "s3://results/$JOB_ID/"

# Update job status
curl -X POST "$ORCHESTRATOR_URL/api/jobs/$JOB_ID/complete" \
     -H "Content-Type: application/json" \
     -d "{\"status\": \"completed\", \"output_location\": \"s3://results/$JOB_ID/\"}"

# Cleanup temporary files
rm -rf /app/data/*
```

### Task Execution Environment

#### Worker Runtime
```python
# worker.py - Main worker execution logic
import os
import json
from kubernetes import client, config

def main():
    job_id = os.environ['JOB_ID']
    job_type = os.environ['JOB_TYPE']

    # Load job specification
    job_spec = get_job_spec(job_id)

    # Initialize storage access
    storage = init_storage_client()

    # Download inputs
    download_inputs(job_spec['data_access'])

    # Execute task based on type
    if job_type == 'pipeline_execution':
        result = execute_pipeline(job_spec)
    elif job_type == 'ml_training':
        result = execute_ml_training(job_spec)
    elif job_type == 'ml_inference':
        result = execute_ml_inference(job_spec)

    # Upload results
    upload_results(result, job_spec['data_access'])

    # Report completion
    report_completion(job_id, result)

if __name__ == '__main__':
    main()
```

## Scaling Decision Logic

### Queue Monitoring
Orchestrator continuously monitors job queue:
```python
# scaling_decision.py
def should_spawn_worker(queue_length, active_workers, max_workers):
    """
    Determine if a new worker should be spawned
    """
    # Always maintain minimum workers
    if active_workers < MIN_WORKERS:
        return True

    # Scale based on queue depth
    if queue_length > QUEUE_THRESHOLD:
        return True

    # Scale based on job priority
    high_priority_jobs = get_high_priority_jobs()
    if high_priority_jobs and active_workers < max_workers:
        return True

    # Scale down if queue is empty
    if queue_length == 0 and active_workers > MIN_WORKERS:
        return False

    return False

# Configuration
MIN_WORKERS = 1
MAX_WORKERS = 50
QUEUE_THRESHOLD = 5
```

### Resource-Aware Scaling
Consider available cluster resources:
```python
def check_cluster_capacity(job_requirements):
    """
    Verify cluster has capacity for job requirements
    """
    available_cpu = get_cluster_available_cpu()
    available_memory = get_cluster_available_memory()

    required_cpu = job_requirements.get('cpu', 1)
    required_memory = parse_memory(job_requirements.get('memory', '1Gi'))

    return available_cpu >= required_cpu and available_memory >= required_memory
```

### Job Type Prioritization
Different scaling rules per job type:
```python
SCALING_RULES = {
    'pipeline_execution': {
        'priority': 1,
        'max_concurrent': 20,
        'resource_weight': 1.0
    },
    'ml_training': {
        'priority': 3,
        'max_concurrent': 5,
        'resource_weight': 3.0  # Higher resource requirements
    },
    'ml_inference': {
        'priority': 2,
        'max_concurrent': 10,
        'resource_weight': 1.5
    },
    'digital_twin_update': {
        'priority': 1,
        'max_concurrent': 15,
        'resource_weight': 1.2
    }
}
```

## Worker Termination and Cleanup

### Automatic Cleanup
Kubernetes Jobs have TTL for automatic cleanup:
```yaml
spec:
  ttlSecondsAfterFinished: 300  # 5 minutes
```

### Manual Cleanup Process
```python
def cleanup_completed_job(job_id):
    """
    Clean up resources after job completion
    """
    # Delete Kubernetes Job
    delete_kubernetes_job(job_id)

    # Clean up temporary storage
    cleanup_temporary_files(job_id)

    # Update job status in database
    update_job_status(job_id, 'cleaned_up')

    # Log completion metrics
    log_job_metrics(job_id)
```

### Failure Handling
Handle worker failures gracefully:
```python
def handle_worker_failure(job_id, failure_reason):
    """
    Handle worker pod failures
    """
    if failure_reason == 'resource_exhausted':
        # Retry with higher resource allocation
        retry_job_with_more_resources(job_id)
    elif failure_reason == 'timeout':
        # Mark as timed out
        mark_job_timeout(job_id)
    else:
        # Generic failure handling
        log_failure_and_retry(job_id, failure_reason)
```

## Monitoring and Metrics

### Worker Metrics
- Job execution time
- Resource utilization
- Success/failure rates
- Queue wait times
- Scaling event frequency

### Scaling Metrics
```prometheus
# Worker pool size
worker_pool_size{type="active"} <current_active_workers>
worker_pool_size{type="pending"} <pending_workers>

# Job queue metrics
job_queue_length <current_queue_depth>
job_processing_rate <jobs_per_minute>

# Resource utilization
worker_cpu_utilization <average_cpu_percent>
worker_memory_utilization <average_memory_percent>
```

### Alerting Rules
```yaml
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: worker-scaling-alerts
spec:
  groups:
  - name: worker.alerts
    rules:
    - alert: WorkerQueueBacklog
      expr: job_queue_length > 20
      for: 5m
      labels:
        severity: warning
      annotations:
        summary: "Worker queue has significant backlog"
    - alert: WorkerResourceExhaustion
      expr: worker_cpu_utilization > 90
      for: 10m
      labels:
        severity: critical
      annotations:
        summary: "Worker pool CPU utilization critically high"
```

## Security Considerations

### Access Control
- Workers run with minimal Kubernetes permissions
- Service accounts scoped to necessary operations
- Network policies restrict worker communication

### Data Security
- Encrypted data transfer
- Temporary credential injection
- Secure deletion of sensitive data post-execution

### Resource Limits
- CPU and memory limits prevent resource exhaustion
- Storage quotas per job
- Execution timeouts prevent runaway processes
