# Scalable Workers

Scalable workers are on-demand Kubernetes Job instances that execute specific work tasks (pipeline runs, ML training/inference, digital twin updates) and automatically terminate after completion. Workers are spawned dynamically based on work task queue demand and are designed for horizontal scaling across the cluster.

## Worker Architecture

### Worker Types
- **Pipeline Worker**: Executes data ingestion and processing pipelines
- **ML Training Worker**: Handles machine learning model training work tasks
- **ML Inference Worker**: Performs model inference and prediction tasks
- **Digital Twin Worker**: Updates and maintains digital twin states

### Worker Lifecycle
1. **Queued**: WorkTask submitted to in-memory queue with task specification
2. **Scheduled**: Orchestrator creates Kubernetes Job manifest
3. **Spawned**: Kubernetes schedules Job pod with appropriate resources
4. **Executing**: Worker downloads task data, executes work task, uploads results
5. **Completed**: Job pod terminates, results stored, cleanup performed

## Worker Spawning Process

### WorkTask Submission
```typescript
// Orchestrator API endpoint
POST /api/worktasks
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
WorkTasks are stored in in-memory queue with structured metadata:
```json
{
  "worktask_id": "task-12345",
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
Orchestrator generates Job manifest based on work task requirements:
```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: worker-task-12345
  labels:
    app: mimir-worker
    worktask-type: pipeline_execution
    worktask-id: task-12345
spec:
  ttlSecondsAfterFinished: 300
  template:
    spec:
      serviceAccountName: worker-service-account
      containers:
      - name: worker
        image: mimir/worker:latest
        env:
        - name: WORKTASK_ID
          value: "task-12345"
        - name: WORKTASK_TYPE
          value: "pipeline_execution"
        - name: ORCHESTRATOR_URL
          value: "http://orchestrator:8080"
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
        - name: worktask-data
          mountPath: /app/data
        - name: model-cache
          mountPath: /app/models
      volumes:
      - name: worktask-data
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
if [ "$WORKTASK_TYPE" = "ml_inference" ]; then
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
  name: task-12345-secrets
type: Opaque
data:
  aws_access_key: <base64-encoded>
  aws_secret_key: <base64-encoded>
  database_url: <base64-encoded>
```

#### Output Data Handling
Results are uploaded before work task completion:
```bash
# Worker completion script
#!/bin/bash

# Upload results
aws s3 sync /app/data/output/ "s3://results/$WORKTASK_ID/"

# Update work task status
curl -X POST "$ORCHESTRATOR_URL/api/worktasks/$WORKTASK_ID/complete" \
     -H "Content-Type: application/json" \
     -d "{\"status\": \"completed\", \"output_location\": \"s3://results/$WORKTASK_ID/\"}"

# Cleanup temporary files
rm -rf /app/data/*
```

### Task Execution Environment

#### Worker Runtime
```go
// worker.go - Main worker execution logic
package main

import (
    "os"
    "log"
)

func main() {
    taskID := os.Getenv("WORKTASK_ID")
    taskType := os.Getenv("WORKTASK_TYPE")

    // Load work task specification
    taskSpec := getWorkTaskSpec(taskID)

    // Initialize storage access
    storage := initStorageClient()

    // Download inputs
    downloadInputs(taskSpec.DataAccess)

    // Execute task based on type
    var result *WorkTaskResult
    switch taskType {
    case "pipeline_execution":
        result = executePipeline(taskSpec)
    case "ml_training":
        result = executeMLTraining(taskSpec)
    case "ml_inference":
        result = executeMLInference(taskSpec)
    }

    // Upload results
    uploadResults(result, taskSpec.DataAccess)

    // Report completion
    reportCompletion(taskID, result)
}
```

## Scaling Decision Logic

### Queue Monitoring
Orchestrator continuously monitors work task queue:
```go
// scaling_decision.go
func shouldSpawnWorker(queueLength, activeWorkers, maxWorkers int64) bool {
    // Always maintain minimum workers
    if activeWorkers < MIN_WORKERS && queueLength > 0 {
        return true
    }

    // Don't exceed max workers
    if activeWorkers >= maxWorkers {
        return false
    }

    // Scale based on queue depth
    if queueLength > QUEUE_THRESHOLD {
        return true
    }

    return false
}

// Configuration
const (
    MIN_WORKERS = 1
    MAX_WORKERS = 50
    QUEUE_THRESHOLD = 5
)
```

### Resource-Aware Scaling
Consider available cluster resources:
```go
func checkClusterCapacity(taskRequirements ResourceRequirements) bool {
    // Verify cluster has capacity for work task requirements
    availableCPU := getClusterAvailableCPU()
    availableMemory := getClusterAvailableMemory()

    requiredCPU := taskRequirements.CPU
    requiredMemory := parseMemory(taskRequirements.Memory)

    return availableCPU >= requiredCPU && availableMemory >= requiredMemory
}
```

### WorkTask Type Prioritization
Different scaling rules per work task type:
```go
var SCALING_RULES = map[WorkTaskType]ScalingRule{
    WorkTaskTypePipelineExecution: {
        Priority:       1,
        MaxConcurrent:  20,
        ResourceWeight: 1.0,
    },
    WorkTaskTypeMLTraining: {
        Priority:       3,
        MaxConcurrent:  5,
        ResourceWeight: 3.0,  // Higher resource requirements
    },
    WorkTaskTypeMLInference: {
        Priority:       2,
        MaxConcurrent:  10,
        ResourceWeight: 1.5,
    },
    WorkTaskTypeDigitalTwinUpdate: {
        Priority:       1,
        MaxConcurrent:  15,
        ResourceWeight: 1.2,
    },
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
```go
func cleanupCompletedWorkTask(taskID string) error {
    // Delete Kubernetes Job
    if err := deleteKubernetesJob(taskID); err != nil {
        return err
    }

    // Clean up temporary storage
    if err := cleanupTemporaryFiles(taskID); err != nil {
        return err
    }

    // Update work task status in queue
    if err := updateWorkTaskStatus(taskID, WorkTaskStatusCleanedUp); err != nil {
        return err
    }

    // Log completion metrics
    logWorkTaskMetrics(taskID)
    return nil
}
```

### Failure Handling
Handle worker failures gracefully:
```go
func handleWorkerFailure(taskID string, failureReason string) error {
    switch failureReason {
    case "resource_exhausted":
        // Retry with higher resource allocation
        return retryWorkTaskWithMoreResources(taskID)
    case "timeout":
        // Mark as timed out
        return markWorkTaskTimeout(taskID)
    default:
        // Generic failure handling
        return logFailureAndRetry(taskID, failureReason)
    }
}
```

## Monitoring and Metrics

### Worker Metrics
- WorkTask execution time
- Resource utilization
- Success/failure rates
- Queue wait times
- Scaling event frequency

### Scaling Metrics
```prometheus
# Worker pool size
worker_pool_size{type="active"} <current_active_workers>
worker_pool_size{type="pending"} <pending_workers>

# WorkTask queue metrics
worktask_queue_length <current_queue_depth>
worktask_processing_rate <tasks_per_minute>

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
      expr: worktask_queue_length > 20
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
- Storage quotas per work task
- Execution timeouts prevent runaway processes
