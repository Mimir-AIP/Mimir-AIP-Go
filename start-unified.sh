#!/bin/bash
set -e

echo "========================================="
echo "Starting Mimir AIP Unified Container"
echo "========================================="

# Create necessary directories
mkdir -p /app/logs
mkdir -p /app/data/tdb2
mkdir -p /app/data/ontologies

# Start Jena Fuseki server in background with update support
echo "Starting Jena Fuseki server on port 3030..."
cd /opt/jena
java $JVM_ARGS -jar fuseki-server.jar \
    --port=3030 \
    --update \
    --loc=/app/data/tdb2 \
    /mimir > /app/logs/fuseki.log 2>&1 &
FUSEKI_PID=$!
echo "Jena Fuseki started with PID $FUSEKI_PID"

# Wait for Fuseki to be ready
echo "Waiting for Jena Fuseki to initialize..."
for i in {1..30}; do
    if curl -s http://localhost:3030/$/ping > /dev/null 2>&1; then
        echo "Jena Fuseki is ready!"
        break
    fi
    if [ $i -eq 30 ]; then
        echo "ERROR: Jena Fuseki failed to start within 30 seconds"
        cat /app/logs/fuseki.log 2>/dev/null || echo "No Fuseki logs found"
        kill $FUSEKI_PID 2>/dev/null || true
        exit 1
    fi
    sleep 1
done

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
        kill $FUSEKI_PID 2>/dev/null || true
        exit 1
    fi
    sleep 1
done

# Start Go backend server in foreground
echo "Starting Go backend server on port 8080..."
echo "========================================="
cd /app
/app/mimir-aip-server --server 8080
