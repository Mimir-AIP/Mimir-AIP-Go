# Mimir AIP - User Journey: 13-Step Validation Guide

This guide explains how to execute and validate each of the 13 E2E test steps in the Mimir AIP UI.

---

## Quick Reference

| Step | Action | UI Location | Validation |
|------|--------|-------------|------------|
| 1 | Verify Docker & API | Main page → Dev Tools | Green health check |
| 2 | Test Agent Tools API | Dev Tools console | JSON responses |
| 3 | Create Pipeline | Pipelines page → Create | Pipeline appears in list |
| 4 | Test Chat Interface | Chat page | Tools panel opens |
| 5 | Configure LLM | Settings → Plugins | Provider options visible |
| 6 | View Pipelines | Pipelines page | Pipeline table visible |
| 7 | View Ontologies | Ontologies page | Upload button visible |
| 8 | View Models | Models page | Model cards visible |
| 9 | View Digital Twins | Digital Twins page | Twin cards visible |
| 10 | Create Digital Twin | API call → Twins page | Twin appears in list |
| 11 | Run What-If Scenario | API call → Twin detail | Simulation results |
| 12 | Create Alert | API call → Alerts page | Alert appears in list |
| 13 | End-to-End Flow | Multiple pages | All components working |

---

## Step-by-Step Instructions

### STEP 1: Verify Docker Container and API

**What to do:**
1. Open browser to `http://localhost:8080`
2. Open Developer Tools (F12) → Console tab

**How to execute:**
```bash
# In terminal, verify container is running
docker ps | grep mimir

# Verify API health
curl http://localhost:8080/api/v1/health
```

**Expected result:**
- Browser shows Mimir AIP landing page
- Console shows no errors
- API response: `{"status":"healthy"}`

**Validation:**
- ✅ Main page title: "Mimir AIP"
- ✅ Console: No red error messages
- ✅ API: Health status = "healthy"

---

### STEP 2: Test Agent Tools API Directly

**What to do:**
1. Open Developer Tools → Network tab
2. Or use a tool like Postman or curl

**How to execute (curl):**
```bash
# Test list_pipelines
curl -X POST http://localhost:8080/api/v1/agent/tools/execute \
  -H "Content-Type: application/json" \
  -d '{"tool_name":"list_pipelines","input":{}}'

# Test recommend_models
curl -X POST http://localhost:8080/api/v1/agent/tools/execute \
  -H "Content-Type: application/json" \
  -d '{"tool_name":"recommend_models","input":{"use_case":"anomaly_detection"}}'

# Test list_ontologies
curl -X POST http://localhost:8080/api/v1/agent/tools/execute \
  -H "Content-Type: application/json" \
  -d '{"tool_name":"list_ontologies","input":{}'

# Test list_alerts
curl -X POST http://localhost:8080/api/v1/agent/tools/execute \
  -H "Content-Type: application/json" \
  -d '{"tool_name":"list_alerts","input":{}'
```

**Expected result:**
```json
{
  "success": true,
  "result": {
    "count": 0,
    "pipelines": []
  }
}
```

**Validation:**
- ✅ Each tool returns `"success": true`
- ✅ Response includes expected fields (count, pipelines, etc.)

---

### STEP 3: Create Pipeline via Agent Tools

**What to do:**
1. Navigate to **Pipelines** page (`/pipelines`)
2. Click **"Create Pipeline"** button

**How to execute (UI):**
1. Click "Create Pipeline" button
2. Enter pipeline name: "Test Pipeline"
3. Add steps (or leave empty for default)
4. Click "Save"

**How to execute (curl):**
```bash
curl -X POST http://localhost:8080/api/v1/agent/tools/execute \
  -H "Content-Type: application/json" \
  -d '{
    "tool_name": "create_pipeline",
    "input": {
      "name": "My First Pipeline",
      "description": "Test pipeline created via UI"
    }
  }'
```

**Expected result:**
- Pipeline appears in the pipeline list
- Pipeline has auto-generated ID like `pl_abc12345`

**Validation:**
- ✅ Pipeline appears in table with name "My First Pipeline"
- ✅ Pipeline status = "Active"
- ✅ Pipeline has steps count displayed

---

### STEP 4: Navigate to Chat and Verify Tools

**What to do:**
1. Click **"Chat"** in the sidebar navigation
2. Look for the **"Available Tools"** button/popover

**How to execute:**
1. Click sidebar: **Chat**
2. Wait for chat to load (~2 seconds)
3. Look for button: **"Available Tools"** or **"Tools"**
4. Click to expand the tools panel

**Expected result:**
- Chat interface with text input appears
- "Available Tools" button is visible
- Clicking reveals MCP tools like:
  - `Input.csv`
  - `Output.json`
  - `Ontology.query`
  - `Ontology.extract`

**Validation:**
- ✅ Textarea for typing messages is visible
- ✅ Welcome message appears ("Chat with Mimir" or similar)
- ✅ Tools toggle button is clickable
- ✅ At least 2-4 tools are visible in the panel

---

### STEP 5: Test LLM Provider Configuration

**What to do:**
1. Click **"Settings"** in the sidebar
2. Click **"Plugins"** tab

**How to execute:**
1. Navigate to `/settings`
2. Find and click the "Plugins" tab
3. Look for AI/LLM configuration options

**Expected result:**
- Settings page loads
- Plugins tab shows configurable plugins
- AI provider options visible (OpenAI, Anthropic, Local, etc.)

**Validation:**
- ✅ Settings page title is visible
- ✅ "Plugins" tab is clickable
- ✅ AI provider cards or Configure buttons are visible

---

### STEP 6: Test Pipeline Management Page

**What to do:**
1. Click **"Pipelines"** in the sidebar

**How to execute:**
1. Navigate to `/pipelines`
2. Look for pipeline list
3. Look for "Create Pipeline" button

**Expected result:**
- Pipelines page with table/list of pipelines
- "Create Pipeline" button visible
- Pipeline cards or table rows visible

**Validation:**
- ✅ Page heading contains "Pipeline"
- ✅ "Create Pipeline" button is clickable
- ✅ Pipeline content (table/cards) is visible

---

### STEP 7: Test Ontologies Page

**What to do:**
1. Click **"Knowledge"** or **"Ontologies"** in the sidebar

**How to execute:**
1. Navigate to `/ontologies`
2. Look for ontology table
3. Look for "Upload Ontology" button

**Expected result:**
- Ontologies page with table of ontologies
- "Upload Ontology" button/link visible

**Validation:**
- ✅ Page heading contains "Ontology"
- ✅ Data table is visible
- ✅ "Upload Ontology" button is clickable

---

### STEP 8: Test Models Page

**What to do:**
1. Click **"Models"** in the sidebar

**How to execute:**
1. Navigate to `/models`
2. Scroll through model cards/sections

**Expected result:**
- Models page showing available ML models
- Model categories (Anomaly Detection, Clustering, etc.)

**Validation:**
- ✅ Page heading contains "Model"
- ✅ Model cards or sections are visible

---

### STEP 9: Test Digital Twins Page

**What to do:**
1. Click **"Digital Twins"** in the sidebar

**How to execute:**
1. Navigate to `/digital-twins`
2. Look for digital twin cards/list

**Expected result:**
- Digital Twins page with twin cards
- "Create Twin" or similar button visible

**Validation:**
- ✅ Page heading contains "Twin"
- ✅ Twin cards are visible
- ✅ Create button is available

---

### STEP 10: Create Digital Twin via Agent Tools

**What to do:**
1. Create a twin using the API
2. Verify it appears on the Twins page

**How to execute (curl):**
```bash
curl -X POST http://localhost:8080/api/v1/agent/tools/execute \
  -H "Content-Type: application/json" \
  -d '{
    "tool_name": "create_twin",
    "input": {
      "name": "My Test Twin",
      "description": "Test twin created via API"
    }
  }'
```

**Expected result:**
```json
{
  "success": true,
  "result": {
    "twin_id": "twin_abc12345",
    "name": "My Test Twin",
    "status": "active"
  }
}
```

**Validation:**
1. ✅ API returns `"success": true`
2. ✅ Response includes `twin_id`
3. ✅ Navigate to `/digital-twins` and find the twin in the list
4. ✅ Twin status = "active"

---

### STEP 11: Test What-If Scenario Simulation

**What to do:**
1. Create a twin (from Step 10)
2. Run a simulation using the API

**How to execute (curl):**
```bash
curl -X POST http://localhost:8080/api/v1/agent/tools/execute \
  -H "Content-Type: application/json" \
  -d '{
    "tool_name": "simulate_scenario",
    "input": {
      "twin_id": "twin_abc12345",
      "scenario": "What if supplier A is unavailable?",
      "parameters": {
        "supplier_a_unavailable": true,
        "backup_supplier": "supplier_b"
      }
    }
  }'
```

**Expected result:**
```json
{
  "success": true,
  "result": {
    "twin_id": "twin_abc12345",
    "scenario": "What if supplier A is unavailable?",
    "result": "Simulated 'What if supplier A is unavailable?' with parameters: {...}",
    "prediction": "Based on simulation, expected outcome is: ...",
    "confidence": 0.85
  }
}
```

**Validation:**
1. ✅ API returns `"success": true`
2. ✅ Response includes `result`, `prediction`, and `confidence`
3. ✅ Navigate to twin detail page `/digital-twins/twin_abc12345`
4. ✅ Simulation history shows the run

---

### STEP 12: Create Alert via Agent Tools

**What to do:**
1. Create an alert using the API
2. Verify it appears in the Alerts page

**How to execute (curl):**
```bash
curl -X POST http://localhost:8080/api/v1/agent/tools/execute \
  -H "Content-Type: application/json" \
  -d '{
    "tool_name": "create_alert",
    "input": {
      "title": "High CPU Alert",
      "type": "threshold",
      "entity_id": "server-001",
      "metric_name": "cpu_usage",
      "severity": "high",
      "message": "CPU usage exceeded 90%"
    }
  }'
```

**Expected result:**
```json
{
  "success": true,
  "result": {
    "alert_id": 1,
    "title": "High CPU Alert",
    "status": "active"
  }
}
```

**Validation:**
1. ✅ API returns `"success": true`
2. ✅ Response includes `alert_id`
3. Navigate to `/monitoring/alerts` or similar
4. ✅ Alert appears in the alerts list
5. ✅ Alert status = "active"

---

### STEP 13: End-to-End Flow Test

**What to do:**
Execute a complete workflow combining all features

**How to execute (curl):**

```bash
# Step 1: Create pipeline
PIPELINE_RESP=$(curl -s -X POST http://localhost:8080/api/v1/agent/tools/execute \
  -H "Content-Type: application/json" \
  -d '{"tool_name":"create_pipeline","input":{"name":"E2E-Test-Pipeline","description":"End-to-end test"}}')
echo "Pipeline: $PIPELINE_RESP"

# Step 2: Get model recommendations
MODELS_RESP=$(curl -s -X POST http://localhost:8080/api/v1/agent/tools/execute \
  -H "Content-Type: application/json" \
  -d '{"tool_name":"recommend_models","input":{"use_case":"anomaly_detection"}}')
echo "Models: $MODELS_RESP"

# Step 3: Create digital twin
TWIN_RESP=$(curl -s -X POST http://localhost:8080/api/v1/agent/tools/execute \
  -H "Content-Type: application/json" \
  -d '{"tool_name":"create_twin","input":{"name":"E2E-Twin","description":"For end-to-end test"}}')
echo "Twin: $TWIN_RESP"

# Step 4: Detect anomalies
ANOMALY_RESP=$(curl -s -X POST http://localhost:8080/api/v1/agent/tools/execute \
  -H "Content-Type: application/json" \
  -d '{"tool_name":"detect_anomalies","input":{"twin_id":"twin_xxx","time_range":"24h"}}')
echo "Anomalies: $ANOMALY_RESP"
```

**Expected result:**
```
Pipeline: {"success":true,"result":{"pipeline_id":"pl_xxx",...}}
Models: {"success":true,"result":{"recommendations":[{"name":"Isolation Forest",...},...]}}
Twin: {"success":true,"result":{"twin_id":"twin_xxx",...}}
Anomalies: {"success":true,"result":{"count":0,"anomalies":[]}}
```

**Validation:**
1. ✅ All 4 API calls return `"success": true`
2. ✅ Pipeline ID is returned and can be viewed in UI
3. ✅ Model recommendations include Isolation Forest, One-Class SVM
4. ✅ Digital twin is created and visible in `/digital-twins`
5. ✅ Anomaly detection runs and returns count

---

## Summary Validation Checklist

Run this command to verify the entire system:

```bash
# Quick health check
echo "=== System Health ==="
curl -s http://localhost:8080/api/v1/health
echo ""

echo "=== Available Providers ==="
curl -s http://localhost:8080/api/v1/ai/providers | jq '.[].provider'
echo ""

echo "=== Pipeline Count ==="
curl -s -X POST http://localhost:8080/api/v1/agent/tools/execute \
  -d '{"tool_name":"list_pipelines","input":{}}' | jq '.result.count'
echo ""

echo "=== Twin Count ==="
curl -s -X POST http://localhost:8080/api/v1/agent/tools/execute \
  -d '{"tool_name":"list_twins","input":{}}' 2>/dev/null || echo "Twins API not available"
echo ""

echo "=== Alert Count ==="
curl -s -X POST http://localhost:8080/api/v1/agent/tools/execute \
  -d '{"tool_name":"list_alerts","input":{}}' | jq '.result.count'
```

**Final Validation:**
- ✅ All curl commands return valid JSON
- ✅ No error messages in browser console
- ✅ All 13 steps can be executed manually
- ✅ Data persists across page refreshes
