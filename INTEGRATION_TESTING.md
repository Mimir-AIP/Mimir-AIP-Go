# Integration Testing Guide for Distributed Architecture

This guide provides examples of how to test the distributed architecture once deployed.

## Prerequisites

1. Start the distributed architecture:
   ```bash
   ./start-distributed.sh up
   ```

2. Verify all services are running:
   ```bash
   docker-compose -f docker-compose.distributed.yml ps
   ```

## Test 1: Health Check

Verify that the backend is healthy:

```bash
curl http://localhost:8080/health
```

Expected response:
```json
{
  "status": "healthy",
  "time": "2026-01-13T23:35:00Z"
}
```

## Test 2: Frontend Access

Open your browser and navigate to:
```
http://localhost:3000
```

The Next.js frontend should load and be able to communicate with the backend.

## Test 3: Synchronous Pipeline Execution

Execute a pipeline synchronously (direct execution, no workers):

```bash
curl -X POST http://localhost:8080/api/v1/pipelines/execute \
  -H "Content-Type: application/json" \
  -d '{
    "pipeline_file": "test_pipeline.yaml"
  }'
```

This will block until the pipeline completes.

## Test 4: Asynchronous Pipeline Execution (Workers)

### Step 1: Enqueue a Job

```bash
JOB_RESPONSE=$(curl -X POST http://localhost:8080/api/v1/queue/pipelines/enqueue \
  -H "Content-Type: application/json" \
  -d '{
    "pipeline_file": "test_pipeline.yaml",
    "context": {"input": "test data"}
  }')

echo $JOB_RESPONSE
```

Expected response:
```json
{
  "message": "Pipeline job enqueued successfully",
  "job_id": "abc123-def456-...",
  "status": "queued"
}
```

### Step 2: Extract Job ID

```bash
JOB_ID=$(echo $JOB_RESPONSE | jq -r '.job_id')
echo "Job ID: $JOB_ID"
```

### Step 3: Check Job Status

```bash
curl http://localhost:8080/api/v1/queue/jobs/$JOB_ID
```

Responses:
- While processing:
  ```json
  {
    "job_id": "abc123-def456-...",
    "status": "processing"
  }
  ```

- When complete:
  ```json
  {
    "id": "abc123-def456-...",
    "success": true,
    "context": {...},
    "executed_at": "2026-01-13T23:35:00Z",
    "worker_id": "worker-hostname-123"
  }
  ```

### Step 4: Wait for Completion

```bash
curl "http://localhost:8080/api/v1/queue/jobs/$JOB_ID/wait?timeout=60s"
```

This will block for up to 60 seconds waiting for the job to complete.

## Test 5: Queue Status

Check the current queue status:

```bash
curl http://localhost:8080/api/v1/queue/status
```

Expected response:
```json
{
  "queue_length": 3
}
```

## Test 6: Worker Scaling

### Check Current Workers

```bash
docker-compose -f docker-compose.distributed.yml ps worker
```

### Scale to 5 Workers

```bash
./start-distributed.sh scale 5
```

### Verify Scaling

```bash
docker-compose -f docker-compose.distributed.yml ps worker
```

You should see 5 worker instances running.

## Test 7: Load Testing

Create a script to enqueue multiple jobs:

```bash
#!/bin/bash
for i in {1..10}; do
  curl -X POST http://localhost:8080/api/v1/queue/pipelines/enqueue \
    -H "Content-Type: application/json" \
    -d "{
      \"pipeline_file\": \"test_pipeline.yaml\",
      \"context\": {\"iteration\": $i}
    }" &
done
wait

echo "All jobs enqueued"
```

Monitor queue status:
```bash
watch -n 1 'curl -s http://localhost:8080/api/v1/queue/status'
```

## Test 8: Worker Logs

View worker logs to see job processing:

```bash
docker-compose -f docker-compose.distributed.yml logs -f worker
```

Expected output:
```
worker_1  | Worker worker-container-123 starting...
worker_1  | Connected to Redis at redis:6379
worker_1  | Processing job abc123-def456-...
worker_1  | Job abc123-def456-... completed successfully
```

## Test 9: Redis Direct Access

Connect to Redis to inspect the queue:

```bash
docker exec -it mimir-redis redis-cli

# Inside Redis CLI
127.0.0.1:6379> LLEN mimir:jobs
(integer) 5

127.0.0.1:6379> KEYS mimir:results:*
1) "mimir:results:abc123-def456-..."

127.0.0.1:6379> GET mimir:results:abc123-def456-...
"{\"id\":\"abc123-...\",\"success\":true,\"executed_at\":\"...\"}"
```

## Test 10: Failure Handling

### Submit Invalid Job

```bash
curl -X POST http://localhost:8080/api/v1/queue/pipelines/enqueue \
  -H "Content-Type: application/json" \
  -d '{
    "pipeline_file": "nonexistent.yaml"
  }'
```

The job will be enqueued but should fail during execution.

### Check Result

```bash
curl http://localhost:8080/api/v1/queue/jobs/$JOB_ID
```

Expected response for failed job:
```json
{
  "id": "abc123-def456-...",
  "success": false,
  "error": "failed to parse pipeline: ...",
  "executed_at": "2026-01-13T23:35:00Z",
  "worker_id": "worker-hostname-123"
}
```

## Test 11: Performance Comparison

### Synchronous Execution Time

```bash
time curl -X POST http://localhost:8080/api/v1/pipelines/execute \
  -H "Content-Type: application/json" \
  -d '{"pipeline_file": "test_pipeline.yaml"}'
```

### Asynchronous Execution Time (Enqueue Only)

```bash
time curl -X POST http://localhost:8080/api/v1/queue/pipelines/enqueue \
  -H "Content-Type: application/json" \
  -d '{"pipeline_file": "test_pipeline.yaml"}'
```

The async version should return much faster (just enqueue time).

## Test 12: Digital Twin Jobs

Enqueue a digital twin job:

```bash
curl -X POST http://localhost:8080/api/v1/queue/digital-twins/enqueue \
  -H "Content-Type: application/json" \
  -d '{
    "twin_id": "test-twin-1",
    "operation": "simulate"
  }'
```

## Cleanup

After testing, stop all services:

```bash
./start-distributed.sh down
```

To clean up all data (including volumes):

```bash
./start-distributed.sh clean
```

## Troubleshooting

### Workers Not Processing Jobs

1. Check Redis connection:
   ```bash
   docker-compose -f docker-compose.distributed.yml logs redis
   ```

2. Check worker logs:
   ```bash
   docker-compose -f docker-compose.distributed.yml logs worker
   ```

3. Verify Redis is reachable from backend:
   ```bash
   docker exec mimir-backend redis-cli -h redis ping
   ```

### Jobs Stuck in Queue

1. Check queue length:
   ```bash
   curl http://localhost:8080/api/v1/queue/status
   ```

2. Restart workers:
   ```bash
   docker-compose -f docker-compose.distributed.yml restart worker
   ```

3. Scale up workers:
   ```bash
   ./start-distributed.sh scale 10
   ```

### Redis Connection Issues

1. Check if Redis is running:
   ```bash
   docker-compose -f docker-compose.distributed.yml ps redis
   ```

2. Check Redis logs:
   ```bash
   docker-compose -f docker-compose.distributed.yml logs redis
   ```

3. Restart Redis:
   ```bash
   docker-compose -f docker-compose.distributed.yml restart redis
   ```
