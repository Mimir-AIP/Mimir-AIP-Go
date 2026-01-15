#!/bin/bash
# Phase 1 Implementation Verification Script
# Tests all critical UI enhancements against unified Docker container

BASE_URL="http://localhost:8080"
API_BASE="$BASE_URL/api/v1"

echo "🚀 Mimir AIP Phase 1 Verification"
echo "=================================="
echo ""

# Color codes
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test counter
PASSED=0
FAILED=0

# Helper function
check_endpoint() {
    local endpoint=$1
    local description=$2
    local expected_field=$3
    
    echo -n "Testing: $description... "
    response=$(curl -s "$endpoint")
    
    if [ -z "$expected_field" ]; then
        if [ ! -z "$response" ]; then
            echo -e "${GREEN}✓ PASS${NC}"
            PASSED=$((PASSED + 1))
            return 0
        fi
    else
        if echo "$response" | grep -q "$expected_field"; then
            echo -e "${GREEN}✓ PASS${NC}"
            PASSED=$((PASSED + 1))
            return 0
        fi
    fi
    
    echo -e "${RED}✗ FAIL${NC}"
    FAILED=$((FAILED + 1))
    return 1
}

echo "1️⃣  System Health Checks"
echo "------------------------"
check_endpoint "$BASE_URL/health" "Health endpoint" "healthy"
check_endpoint "$API_BASE/ontologies" "Ontologies API" "id"
check_endpoint "$API_BASE/models" "ML Models API" "models"
check_endpoint "$API_BASE/knowledge-graph" "Knowledge Graph API" "message"
check_endpoint "$API_BASE/pipelines" "Pipelines API" "id"
echo ""

echo "2️⃣  Frontend Page Availability"
echo "-------------------------------"
check_endpoint "$BASE_URL" "Homepage" "Mimir AIP"
check_endpoint "$BASE_URL/ontologies" "Ontologies page" "Mimir AIP"
check_endpoint "$BASE_URL/knowledge-graph" "Knowledge Graph page" "Mimir AIP"
check_endpoint "$BASE_URL/models" "ML Models page" "Mimir AIP"
check_endpoint "$BASE_URL/digital-twins" "Digital Twins page" "Mimir AIP"
check_endpoint "$BASE_URL/pipelines" "Pipelines page" "Mimir AIP"
echo ""

echo "3️⃣  Phase 1 Feature Verification"
echo "---------------------------------"

# Get first ontology ID
ONTOLOGY_ID=$(curl -s "$API_BASE/ontologies" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
if [ ! -z "$ONTOLOGY_ID" ]; then
    echo -e "${YELLOW}Using ontology: $ONTOLOGY_ID${NC}"
    check_endpoint "$BASE_URL/ontologies/$ONTOLOGY_ID" "Ontology details page" "Mimir AIP"
    check_endpoint "$API_BASE/ontologies/$ONTOLOGY_ID/classes" "Ontology classes API" ""
    check_endpoint "$API_BASE/ontologies/$ONTOLOGY_ID/properties" "Ontology properties API" ""
    check_endpoint "$API_BASE/ontologies/$ONTOLOGY_ID/instances" "Ontology instances API" ""
else
    echo -e "${YELLOW}⚠ No ontologies found, skipping ontology tests${NC}"
fi
echo ""

# Get first model ID
MODEL_ID=$(curl -s "$API_BASE/models" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
if [ ! -z "$MODEL_ID" ]; then
    echo -e "${YELLOW}Using model: $MODEL_ID${NC}"
    check_endpoint "$BASE_URL/models/$MODEL_ID" "ML model details page" "Mimir AIP"
    check_endpoint "$API_BASE/models/$MODEL_ID" "ML model API" "id"
else
    echo -e "${YELLOW}⚠ No models found, skipping model tests${NC}"
fi
echo ""

echo "4️⃣  Component File Verification"
echo "--------------------------------"

# Check if new components exist in the container
if docker exec mimir-aip-unified test -f /app/frontend/.next/server/pages-manifest.json; then
    echo -e "${GREEN}✓ Frontend build exists in container${NC}"
    PASSED=$((PASSED + 1))
else
    echo -e "${RED}✗ Frontend build missing${NC}"
    FAILED=$((FAILED + 1))
fi

if docker exec mimir-aip-unified test -f /app/mimir-aip-server; then
    echo -e "${GREEN}✓ Backend binary exists in container${NC}"
    PASSED=$((PASSED + 1))
else
    echo -e "${RED}✗ Backend binary missing${NC}"
    FAILED=$((FAILED + 1))
fi

echo ""
echo "=================================="
echo "📊 Test Results Summary"
echo "=================================="
echo -e "Passed: ${GREEN}$PASSED${NC}"
echo -e "Failed: ${RED}$FAILED${NC}"
echo ""

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}✅ All Phase 1 features verified successfully!${NC}"
    exit 0
else
    echo -e "${YELLOW}⚠️  Some tests failed. Review above for details.${NC}"
    exit 1
fi
