#!/bin/bash
set -e

echo "========================================="
echo "Starting Mimir AIP Unified Container"
echo "========================================="

# Create logs directory if it doesn't exist
mkdir -p /app/logs

# Start Next.js frontend server in background
echo "Starting Next.js frontend server on port 3000..."
cd /app/frontend
NODE_ENV=production node server.js > /app/logs/nextjs.log 2>&1 &
NEXTJS_PID=$!
echo "Next.js started with PID $NEXTJS_PID"

# Wait for Next.js to be ready
echo "Waiting for Next.js to initialize..."
for i in {1..30}; do
    if curl -s http://localhost:3000 > /dev/null 2>&1; then
        echo "Next.js is ready!"
        break
    fi
    if [ $i -eq 30 ]; then
        echo "ERROR: Next.js failed to start within 30 seconds"
        cat /app/logs/nextjs.log 2>/dev/null || echo "No Next.js logs found"
        kill $NEXTJS_PID 2>/dev/null || true
        exit 1
    fi
    sleep 1
done

# Start Go backend server in foreground
echo "Starting Go backend server on port 8080..."
echo "========================================="
cd /app
exec /app/mimir-aip-server --server 8080
