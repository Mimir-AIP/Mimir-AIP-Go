#!/bin/bash
set -e

echo "========================================="
echo "Starting Mimir AIP Backend Container"
echo "========================================="

# Create necessary directories
mkdir -p /app/logs
mkdir -p /app/data/tdb2
mkdir -p /app/data/ontologies

# Start Jena Fuseki server in background with update support
echo "Starting Jena Fuseki server on port ${FUSEKI_PORT:-3030}..."
cd /opt/jena
java $JVM_ARGS -jar fuseki-server.jar \
    --port=${FUSEKI_PORT:-3030} \
    --update \
    --loc=/app/data/tdb2 \
    /mimir > /app/logs/fuseki.log 2>&1 &
FUSEKI_PID=$!
echo "Jena Fuseki started with PID $FUSEKI_PID"

# Wait for Fuseki to be ready
echo "Waiting for Jena Fuseki to initialize..."
for i in {1..30}; do
    if curl -s http://localhost:${FUSEKI_PORT:-3030}/$/ping > /dev/null 2>&1; then
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

# Start Go backend server in foreground
echo "Starting Go backend server on port ${MIMIR_PORT:-8080}..."
echo "========================================="
cd /app
exec /app/mimir-aip-server --server ${MIMIR_PORT:-8080}
