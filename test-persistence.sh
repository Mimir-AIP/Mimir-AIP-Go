#!/bin/bash
# Integration test for persistence across restarts

set -e

echo "=========================================="
echo "Testing Persistence Across Restarts"
echo "=========================================="
echo ""

# Clean up any existing test data
echo "Cleaning up test data..."
rm -rf /tmp/mimir-test-data
mkdir -p /tmp/mimir-test-data

# Export environment variables for test database
export MIMIR_DATABASE_PATH=/tmp/mimir-test-data/test.db
export MIMIR_VECTOR_DB_PATH=/tmp/mimir-test-data/chromem.db

# Start server in background
echo "Starting server (first time)..."
timeout 3 ./mimir-aip-server --server 9999 > /tmp/mimir-test.log 2>&1 &
SERVER_PID=$!

# Wait for server to start
sleep 2

# Create a job via API
echo "Creating a test job..."
curl -X POST http://localhost:9999/api/v1/scheduler/jobs \
  -H "Content-Type: application/json" \
  -d '{
    "id": "test-job-persist",
    "name": "Test Persistence Job",
    "pipeline": "test.yaml",
    "cron_expr": "0 9 * * *"
  }' \
  -s -o /dev/null

# Verify job was created
echo "Verifying job creation..."
JOBS=$(curl -s http://localhost:9999/api/v1/scheduler/jobs)
if echo "$JOBS" | grep -q "test-job-persist"; then
  echo "✓ Job created successfully"
else
  echo "✗ Failed to create job"
  exit 1
fi

# Stop server gracefully
echo "Stopping server..."
kill -SIGTERM $SERVER_PID
wait $SERVER_PID 2>/dev/null || true
sleep 1

# Verify database file exists
if [ -f "$MIMIR_DATABASE_PATH" ]; then
  echo "✓ Database file created"
else
  echo "✗ Database file not found"
  exit 1
fi

# Check database contents
echo "Checking database contents..."
JOB_COUNT=$(sqlite3 "$MIMIR_DATABASE_PATH" "SELECT COUNT(*) FROM scheduled_jobs WHERE id='test-job-persist';")
if [ "$JOB_COUNT" = "1" ]; then
  echo "✓ Job persisted to database"
else
  echo "✗ Job not found in database"
  exit 1
fi

# Start server again
echo "Starting server (second time)..."
timeout 3 ./mimir-aip-server --server 9999 > /tmp/mimir-test2.log 2>&1 &
SERVER_PID=$!

# Wait for server to start
sleep 2

# Verify job is still there
echo "Verifying job persistence after restart..."
JOBS=$(curl -s http://localhost:9999/api/v1/scheduler/jobs)
if echo "$JOBS" | grep -q "test-job-persist"; then
  echo "✓ Job persisted across restart"
else
  echo "✗ Job not found after restart"
  kill -SIGTERM $SERVER_PID 2>/dev/null || true
  exit 1
fi

# Stop server
echo "Stopping server..."
kill -SIGTERM $SERVER_PID
wait $SERVER_PID 2>/dev/null || true

# Clean up
echo "Cleaning up..."
rm -rf /tmp/mimir-test-data
rm -f /tmp/mimir-test.log /tmp/mimir-test2.log

echo ""
echo "=========================================="
echo "✓ All persistence tests passed!"
echo "=========================================="
