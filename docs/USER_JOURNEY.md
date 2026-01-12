# Mimir AIP User Journey

## Vision
Mimir AIP enables users to chat with an AI agent to set up a complete data pipeline, ontology, ML models, and digital twin for their business/organization with minimal technical knowledge.

---

## Current Status (Completed ✓)

### Foundation - LLM Provider/Plugin System ✓

**Backend:**
- ✓ All 8 AI providers registered as separate plugins: openai, anthropic, google, ollama, openrouter, z-ai, azure, mock
- ✓ Plugin configuration storage (GET/PUT/DELETE endpoints at `/api/v1/settings/plugins/{name}/config`)
- ✓ Model fetching endpoints:
  - `/api/v1/ai/providers` - List all providers with status
  - `/api/v1/ai/providers/{provider}/models` - Fetch models from provider API
- ✓ `LLMPluginConfigGetter` function for components to get LLM config from plugins
- ✓ Output plugins (PDF, Excel, JSON) have empty schemas (step params removed)

### Agent Tools ✓ (NEW)

**Endpoint:** `POST /api/v1/agent/tools/execute`

**Available Tools:**
| Tool | Description |
|------|-------------|
| `create_pipeline` | Creates a pipeline from name/description (auto-generates steps) |
| `execute_pipeline` | Executes a pipeline by ID or name |
| `schedule_pipeline` | Schedules a pipeline with cron expression |
| `extract_ontology` | Triggers ontology extraction from data source |
| `list_ontologies` | Lists all ontologies |
| `recommend_models` | Suggests ML models based on use case |
| `create_twin` | Creates a digital twin |
| `get_twin_status` | Returns twin status and health |
| `simulate_scenario` | Runs what-if scenario on digital twin |
| `detect_anomalies` | Detects anomalies in recent data |
| `create_alert` | Creates alert configuration |
| `list_alerts` | Lists all alerts |
| `list_pipelines` | Lists all pipelines |
| `get_pipeline_status` | Returns pipeline status |

**Example Usage:**
```bash
# Create a pipeline from natural language
curl -X POST /api/v1/agent/tools/execute \
  -d '{"tool_name":"create_pipeline","input":{"name":"Sales Pipeline","description":"CSV sales data"}}'

# Get model recommendations
curl -X POST /api/v1/agent/tools/execute \
  -d '{"tool_name":"recommend_models","input":{"use_case":"anomaly_detection"}}'

# List pipelines
curl -X POST /api/v1/agent/tools/execute \
  -d '{"tool_name":"list_pipelines","input":{}}'
```

### Frontend
- ✓ Settings → Plugins page for configuring LLM providers
- ✓ ConfigDialog with dynamic form generation
- ✓ ModelSelect component with searchable dropdown and API fetching
- ✓ Agent chat at `/chat` with "Ask Mimir" button in topbar (accessible from any page)
- ✓ ModelSelector in chat uses configured providers

**API Verification:**
```bash
# All 8 AI plugins registered
curl /api/v1/settings/plugins | jq '.[] | select(.type=="AI") | .name'
# "anthropic", "azure", "google", "mock", "ollama", "openai", "openrouter", "z-ai"

# Model fetching works
curl /api/v1/ai/providers/openai/models
# {"provider":"openai","models":["gpt-4o","gpt-4o-mini",...]}
```

### How LLMs Are Called

All LLM calls now go through the plugin system:
1. Components call `AI.GetLLMClientForProvider(provider, defaultClient)`
2. This checks plugin config for the provider
3. If configured, creates client with stored API key/base_url/model
4. Falls back to default client if not configured

---

## User Journey: Setting Up Mimir for Your Business

### Phase 1: Initial Setup

**Entry Point:** User navigates to Mimir homepage or opens the app

**Step 1.1: Configure LLM Provider (if not already done)**
- User clicks "Settings" → "Plugins" → "Configure" on their preferred LLM provider
- Fills in API key and selects model (models fetched from provider)
- Mimir now has LLM capability for all operations

**Alternative:** User can skip this and the agent will use the default mock provider for demo

---

### Phase 2: Chat-Based Setup with Agent

**Step 2.1: Open Agent Chat**
- User clicks the "Agent" button (accessible from any screen)
- Chat interface opens with a welcome message:
  > "Hi! I'm your Mimir assistant. I can help you set up data ingestion pipelines, create ontologies, build ML models, and create digital twins for your business. Just tell me about your data source and what you want to achieve!"

**Step 2.2: User Describes Their Data**
- User chats: "I have sales data in CSV files on my server, and I want to understand customer patterns and predict anomalies"
- Agent acknowledges and asks clarifying questions via chat if needed:
  - "Where are the CSV files located?"
  - "How often is new data generated?"
  - "What fields does your sales data contain?"

**Step 2.3: Agent Creates Data Ingestion Pipeline**
- Agent uses tools to create a pipeline with:
  - Input: CSV plugin with file path or API endpoint
  - Processing: Any needed transformations
  - Output: JSON/Parquet for storage
- Agent asks user to confirm or adjust the pipeline
- User can approve via chat: "Yes, that looks good"

**Step 2.4: Agent Triggers Data Ingestion & Ontology Creation**
- Agent schedules the pipeline to run (or runs immediately)
- Once data is ingested, agent triggers ontology extraction:
  - Uses LLM (from user's configured provider) for entity/relationship extraction
  - Creates ontology based on data structure
- Agent shows user the extracted ontology for confirmation

**Step 2.5: Agent Recommends ML Models**
- Agent analyzes data and ontology
- Recommends ML models based on use case:
  - "For anomaly detection, I recommend an Isolation Forest model"
  - "For customer segmentation, a K-Means clustering model"
- User reviews and approves via chat

**Step 2.6: Agent Trains Models & Creates Digital Twin**
- Agent trains approved models
- Creates digital twin that:
  - Stores historical data
  - Couples with ML models for predictions
  - Enables what-if analysis

**Step 2.7: Agent Sets Up Monitoring & Alerts**
- Agent configures scheduled pipeline runs for new data
- Sets up anomaly detection comparing new data to digital twin predictions
- Creates alert pipeline (email, webhook, etc.) for anomalies

---

### Phase 3: Ongoing Use

**Step 3.1: Regular Monitoring**
- User can ask agent: "Show me today's anomalies"
- Agent queries digital twin and ML models
- Returns anomaly report with explanations

**Step 3.2: What-If Analysis**
- User asks: "If supplier A is unavailable, how long can production continue?"
- Agent uses digital twin tools to simulate scenario
- Returns prediction: "Based on current inventory levels, production will halt by [date]"

**Step 3.3: Continuous Improvement**
- User chats: "Add a new data source with customer feedback"
- Agent creates new ingestion pipeline
- Updates ontology with new entity types
- Retrains models if needed

---

## Frontend Pages Required

### 1. Home/Dashboard (`/`)
- Welcome message with quick start options
- Agent chat button (floating or prominent)
- Status overview: pipelines running, ontologies, models, digital twin health
- Recent anomalies and alerts

### 2. Agent Chat (`/agent`)
- Chat interface with conversation history
- Tool calls displayed in chat (expandable)
- Confirmation modals for actions requiring approval
- Rich responses: pipeline configs, model recommendations, visualizations

### 3. Pipelines (`/pipelines`)
- List of all pipelines
- Create new pipeline (UI + agent chat option)
- Pipeline editor (visual or YAML)
- Schedule configuration
- Execution history and logs

### 4. Ontologies (`/ontologies`)
- List of ontologies
- View ontology schema (visual graph)
- Ontology versions and evolution
- Entity/relationship browser

### 5. Models (`/models`)
- List of trained models
- Model performance metrics
- Retrain options
- Model recommendations based on ontologies

### 6. Digital Twin (`/digital-twin`)
- Twin status and health
- Data overview
- What-if scenario builder
- Simulation results

### 7. Alerts (`/alerts`)
- List of alert configurations
- Alert history
- Create/edit alert pipelines
- Notification settings

### 8. Settings (`/settings`)
- **Plugins tab**: Configure LLM providers and other plugins
- System configuration
- User preferences

---

## Navigation Structure

```
┌─────────────────────────────────────────────────────────┐
│  Logo    Pipelines  Ontologies  Models  Alerts  [Agent] │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  [Home content based on current state]                  │
│                                                         │
├─────────────────────────────────────────────────────────┤
│  Footer: Settings | Help | User                         │
└─────────────────────────────────────────────────────────┘
```

**Agent Button:** Fixed position, accessible from all pages (bottom-right or top-right)

---

## Agent Tools (Backend)

The agent has access to these tools:

1. **Pipeline Management**
   - `create_pipeline(config)`
   - `update_pipeline(id, config)`
   - `execute_pipeline(id)`
   - `schedule_pipeline(id, cron)`

2. **Ontology Management**
   - `create_ontology(data)`
   - `extract_ontology(pipeline_id)`
   - `get_ontology(id)`

3. **Model Management**
   - `recommend_models(ontology_id, use_case)`
   - `train_model(config)`
   - `get_model_status(id)`

4. **Digital Twin**
   - `create_twin(ontology_id, model_ids)`
   - `simulate_scenario(twin_id, scenario)`
   - `get_twin_status(twin_id)`

5. **Alert Management**
   - `create_alert(config)`
   - `get_anomalies(twin_id, time_range)`

6. **Plugin Configuration**
   - `get_plugin_config(name)`
   - `save_plugin_config(name, config)`

---

## What's Next (Priority Order)

### High Priority

1. **Agent Integration with Chat**
   - Connect agent chat to `/api/v1/agent/tools/execute`
   - When LLM suggests tool calls, execute via agent tools API
   - Display tool results in chat conversation

2. **Dashboard Home Page**
   - Show system overview (pipelines, ontologies, twins, alerts)
   - Quick actions for common tasks
   - Recent anomalies widget

### Medium Priority

3. **Enhanced Pipeline Creation**
   - More sophisticated step generation from natural language
   - Template library for common data sources

4. **What-If Analysis UI**
   - Visual scenario builder
   - Simulation results visualization

### Lower Priority

5. **Advanced Features**
   - Auto-retraining schedules
   - Complex alert pipelines
   - Multi-source data fusion

---

## Implementation Priority Summary

| Feature | Status | Notes |
|---------|--------|-------|
| LLM Provider/Plugin System | ✓ Done | 8 providers, config storage |
| Agent Tools API | ✓ Done | 14 tools for pipeline/ontology/twin/alert management |
| Agent Chat | ✓ Done | /chat, tools panel |
| Settings → Plugins | ✓ Done | Configure LLM providers |
| Model Fetching | ✓ Done | API endpoints for each provider |
| Chat Integration with Tools | 🔄 In Progress | Connect chat to agent tools |
| Dashboard Home | ⏳ Pending | System overview page |
| What-If UI | ⏳ Pending | Visual scenario builder |
| Alert Pipelines | ⏳ Pending | Alert management UI |
| Agent Tools | 🔄 In Progress | Basic tools exist, need expansion |
| Pipeline Creation via Chat | ⏳ Pending | Need create_pipeline tool |
| Ontology Extraction via Chat | ⏳ Pending | Need extract_ontology tool |
| Digital Twin Setup | ⏳ Pending | Need create_twin tool |
| What-If Analysis | ⏳ Pending | Need simulate_scenario tool |
| Alert Pipelines | ⏳ Pending | Need alert configuration |

---

## Frontend Pages Status

| Page | Status | Notes |
|------|--------|-------|
| /chat (Agent) | ✓ Done | Full chat with tools |
| /pipelines | ✓ Exists | List and management |
| /ontologies | ✓ Exists | List and management |
| /models | ✓ Exists | List and management |
| /digital-twins | ✓ Exists | List and management |
| /settings/plugins | ✓ Done | Configure LLM providers |
| /alerts | ⏳ Pending | Need alert management |
| /dashboard | ✓ Exists | May need enhancement |

---

## Quick Start for User

1. **Configure LLM Provider**
   - Go to Settings → Plugins
   - Click "Configure" on your preferred provider (e.g., OpenAI)
   - Enter API key and select model
   - Click Save

2. **Open Agent Chat**
   - Click "Ask Mimir" button in top-right corner
   - Or navigate to /chat directly

3. **Set Up Your Data Pipeline**
   - Chat: "I have sales data in CSV files"
   - Agent creates pipeline configuration
   - Confirm via chat

4. **Create Ontology & Models**
   - Chat: "Analyze my data and create an ontology"
   - Agent runs extraction
   - Review and approve via chat

5. **Set Up Digital Twin**
   - Chat: "Create a digital twin for anomaly detection"
   - Agent sets up twin with ML models
   - Configure alert pipeline

6. **Ongoing Use**
   - Chat: "Are there any anomalies today?"
   - Chat: "What if supplier A is unavailable?"
