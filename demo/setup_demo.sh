#!/bin/bash
# Complete Demo Setup Script - Creates all demo data from scratch

set -e

echo "🚀 Mimir AIP - Complete Demo Setup"
echo "==================================="
echo ""

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

API_BASE="http://localhost:8080/api/v1"

echo -e "${BLUE}Step 1: Creating Demo Pipeline...${NC}"
echo "-----------------------------------"

# Create the demo pipeline via API
PIPELINE_RESULT=$(curl -s -X POST "${API_BASE}/pipelines" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Demo Ingestion",
    "description": "Import demo parts data from CSV",
    "config": {
      "name": "demo-ingestion",
      "steps": [
        {
          "name": "read-csv",
          "plugin": "Input.csv",
          "config": {
            "file_path": "/app/pipelines/demo_parts.csv",
            "has_headers": true
          }
        }
      ]
    },
    "enabled": true,
    "auto_extract_ontology": true
  }')

PIPELINE_ID=$(echo $PIPELINE_RESULT | python3 -c "import sys,json; print(json.load(sys.stdin).get('id',''))" 2>/dev/null || echo "")

if [ -z "$PIPELINE_ID" ]; then
    echo -e "${YELLOW}⚠️  Failed to create pipeline or pipeline already exists${NC}"
    # Try to get existing pipeline ID
    PIPELINE_ID=$(curl -s "${API_BASE}/pipelines" | python3 -c "import sys,json; data=json.load(sys.stdin); print(data[0]['id'] if data else '')" 2>/dev/null || echo "")
fi

if [ -n "$PIPELINE_ID" ]; then
    echo -e "${GREEN}✅ Pipeline created: $PIPELINE_ID${NC}"
    
    # Execute the pipeline
    echo ""
    echo -e "${BLUE}Step 2: Executing Pipeline to Create Ontology...${NC}"
    echo "-----------------------------------"
    
    EXEC_RESULT=$(curl -s -X POST "${API_BASE}/pipelines/execute" \
      -H "Content-Type: application/json" \
      -d "{\"pipeline_id\": \"$PIPELINE_ID\"}")
    
    if echo "$EXEC_RESULT" | grep -q '"success":true'; then
        echo -e "${GREEN}✅ Pipeline executed successfully${NC}"
    else
        echo -e "${YELLOW}⚠️  Pipeline execution may have issues${NC}"
        echo "$EXEC_RESULT"
    fi
else
    echo -e "${YELLOW}⚠️  No pipeline ID available${NC}"
fi

# Wait for extraction to complete
echo ""
echo "⏳ Waiting for data extraction (5 seconds)..."
sleep 5

echo ""
echo -e "${BLUE}Step 3: Checking Created Data...${NC}"
echo "-----------------------------------"

# Check ontologies
ONTOLOGY_COUNT=$(curl -s "${API_BASE}/ontologies" | python3 -c "import sys,json; print(len(json.load(sys.stdin)))" 2>/dev/null || echo "0")
echo -e "${GREEN}✅ Ontologies: $ONTOLOGY_COUNT${NC}"

# Get first ontology ID for twin creation
ONTOLOGY_ID=$(curl -s "${API_BASE}/ontologies" | python3 -c "import sys,json; data=json.load(sys.stdin); print(data[0]['id'] if data else '')" 2>/dev/null || echo "")

echo ""
echo -e "${BLUE}Step 4: Creating Digital Twin...${NC}"
echo "-----------------------------------"

if [ -n "$ONTOLOGY_ID" ]; then
    TWIN_RESULT=$(curl -s -X POST "${API_BASE}/twin/create" \
      -H "Content-Type: application/json" \
      -d "{
        \"name\": \"Repair Shop Operations Twin\",
        \"description\": \"Digital twin for repair shop with parts inventory\",
        \"ontology_id\": \"$ONTOLOGY_ID\",
        \"model_type\": \"organization\"
      }")
    
    if echo "$TWIN_RESULT" | grep -q '"success":true'; then
        echo -e "${GREEN}✅ Digital twin created${NC}"
    else
        echo -e "${YELLOW}⚠️  Twin creation result:${NC}"
        echo "$TWIN_RESULT" | python3 -m json.tool 2>/dev/null || echo "$TWIN_RESULT"
    fi
else
    echo -e "${YELLOW}⚠️  No ontology available for twin creation${NC}"
fi

echo ""
echo -e "${BLUE}Step 5: Creating Scheduled Jobs...${NC}"
echo "-----------------------------------"

if [ -n "$PIPELINE_ID" ]; then
    # Create continuous ingestion job
    JOB1_RESULT=$(curl -s -X POST "${API_BASE}/scheduler/jobs" \
      -H "Content-Type: application/json" \
      -d "{
        \"id\": \"parts-continuous-ingestion\",
        \"name\": \"Parts CSV - Continuous Monitoring\",
        \"pipeline\": \"$PIPELINE_ID\",
        \"cron_expr\": \"*/5 * * * *\"
      }")
    
    if echo "$JOB1_RESULT" | grep -q '"success":true\|\"id\"'; then
        echo -e "${GREEN}✅ Scheduled job created: parts-continuous-ingestion${NC}"
    else
        echo -e "${YELLOW}⚠️  Job creation result:${NC}"
        echo "$JOB1_RESULT"
    fi
    
    # Enable the job
    curl -s -X POST "${API_BASE}/scheduler/jobs/parts-continuous-ingestion/enable" > /dev/null 2>&1 || true
fi

echo ""
echo -e "${BLUE}Step 6: Creating Drift Detection...${NC}"
echo "-----------------------------------"

if [ -n "$ONTOLOGY_ID" ] && [ -n "$PIPELINE_ID" ]; then
    DRIFT_RESULT=$(curl -s -X POST "${API_BASE}/drift/jobs" \
      -H "Content-Type: application/json" \
      -d "{
        \"ontology_id\": \"$ONTOLOGY_ID\",
        \"pipeline_id\": \"$PIPELINE_ID\",
        \"auto_remediate\": true,
        \"schedule\": {
          \"enabled\": true,
          \"cron_expr\": \"0 */2 * * *\"
        }
      }")
    
    if echo "$DRIFT_RESULT" | grep -q '"success":true'; then
        echo -e "${GREEN}✅ Drift detection created${NC}"
    fi
fi

echo ""
echo -e "${GREEN}===================================${NC}"
echo -e "${GREEN}✅ Demo Setup Complete!${NC}"
echo -e "${GREEN}===================================${NC}"
echo ""
echo "📊 Summary:"
echo "   • Pipeline: $([ -n "$PIPELINE_ID" ] && echo "✅ Created" || echo "❌ Failed")"
echo "   • Ontology: $([ "$ONTOLOGY_COUNT" -gt 0 ] && echo "✅ $ONTOLOGY_COUNT created" || echo "❌ None")"
echo "   • Scheduled Jobs: ✅ Created"
echo ""
echo "🌐 Access the demo at:"
echo "   http://localhost:8080"
echo ""
echo "📚 API Endpoints:"
echo "   • Pipelines:  curl http://localhost:8080/api/v1/pipelines"
echo "   • Ontologies: curl http://localhost:8080/api/v1/ontologies"
echo "   • Twins:      curl http://localhost:8080/api/v1/twins"
echo ""
