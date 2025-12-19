# Mimir AIP: Autonomous System - Detailed Implementation Plan

**Last Updated:** 2025-12-19  
**Status:** Phase 0 - Documentation & Gap Analysis Complete  
**Overall Autonomous Readiness:** ~15%

---

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [Current State Analysis](#current-state-analysis)
3. [Missing Components for Autonomy](#missing-components-for-autonomy)
4. [Detailed Implementation Requirements](#detailed-implementation-requirements)
5. [Implementation Phases](#implementation-phases)
6. [Testing Strategy](#testing-strategy)

---

## Executive Summary

### What Exists Today

Mimir AIP has a **solid foundation** with modular components that work well through manual UI interaction:

- ✅ Data ingestion (CSV, JSON, Excel) via frontend
- ✅ Ontology management (manual OWL/TTL upload, versioning, drift detection)
- ✅ ML model training (manual trigger, works standalone)
- ✅ Digital twin creation and simulation
- ✅ Job scheduling infrastructure (CRUD operations)
- ✅ Schema inference engine (automatic type detection, FK detection, AI fallback)
- ✅ Entity extraction plugins (deterministic, LLM, hybrid)

### The Critical Gap

**The system requires manual user intervention at every stage.** Components exist but are **disconnected** - there's no automation layer that chains them together.

### Autonomous Vision

User uploads a CSV file → System should automatically:

1. **Infer schema** from data
2. **Generate ontology** from schema
3. **Extract entities** using the ontology
4. **Train ML models** based on ontology classes
5. **Create digital twin** with trained models
6. **Monitor continuously** for anomalies
7. **Alert users** when issues are detected

---

## Current State Analysis

### 1. Data Ingestion System

**Status:** 70% Complete | 40% Connected | 10% Autonomous | 80% UI Friendly

#### What Works
- **Frontend:** `/mimir-aip-frontend/src/app/data/upload/page.tsx` (415 lines)
  - Drag-and-drop file upload
  - CSV preview with column detection
  - Plugin selection (CSV, JSON, Excel, XML, Markdown)
  - File size validation (10MB limit)
  - Error handling with toast notifications

- **Backend:** `/handlers.go` - `handleUploadData`, `handleDataImport`
  - File processing via plugin system
  - Data preview generation
  - Storage in SQLite (`imported_data` table)
  - API endpoints: `POST /api/v1/data/upload`, `POST /api/v1/data/import`

- **Plugins:** `/pipelines/Input/`
  - `csv_plugin.go` - CSV parsing with header detection
  - `excel_plugin.go` - XLSX support
  - `json_plugin.go` - JSON data ingestion
  - `xml_plugin.go` - XML parsing
  - `markdown_plugin.go` - Markdown table extraction

#### What's Missing for Autonomy
1. **No automatic post-upload workflow trigger**
   - Data is uploaded and stored, but nothing happens next
   - No job created, no ontology generation triggered
   - Dead end after upload

2. **Database connectors not implemented**
   - MySQL connector (planned but not coded)
   - PostgreSQL connector (planned but not coded)
   - MongoDB connector (planned but not coded)
   - API connectors (REST, GraphQL)

3. **No incremental data ingestion**
   - Only batch uploads work
   - No streaming data support
   - No CDC (Change Data Capture) for databases

4. **Job scheduling integration missing**
   - Can't schedule recurring data imports from UI
   - No link between job scheduler and data sources
   - Manual job creation only via API

#### Implementation Requirements

**File:** `/handlers.go` - Modify `handleDataImport`

```go
// After data import succeeds (line ~150):
if automaticMode {
    // 1. Trigger schema inference
    schemaJob := createSchemaInferenceJob(importID, fileName)
    
    // 2. Create workflow record
    workflowID := createAutonomousWorkflow(importID, userID)
    
    // 3. Trigger next step asynchronously
    go executeSchemaInferenceStep(workflowID, schemaJob)
}
```

**New Files Needed:**
- `/handlers_workflow.go` - Workflow orchestration handlers
- `/pipelines/Workflow/autonomous_orchestrator.go` - Workflow engine
- `/mimir-aip-frontend/src/app/data/upload/page.tsx` - Add "Enable Autonomous Mode" checkbox

**Database Changes:**
```sql
CREATE TABLE autonomous_workflows (
    id TEXT PRIMARY KEY,
    import_id TEXT,
    status TEXT,  -- pending, schema_inference, ontology_creation, ml_training, twin_creation, monitoring, completed, failed
    current_step TEXT,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    completed_at TIMESTAMP,
    metadata TEXT
);

CREATE TABLE workflow_steps (
    id INTEGER PRIMARY KEY,
    workflow_id TEXT,
    step_name TEXT,
    status TEXT,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    error_message TEXT,
    output_data TEXT
);
```

**Estimated Effort:** 3-4 days

---

### 2. Schema Inference & Ontology Creation System

**Status:** 40% Complete | 20% Connected | 5% Autonomous | 60% UI Friendly

#### What Works

**Schema Inference Engine:** `/pipelines/Ontology/schema_inference/`
- ✅ `engine.go` (981 lines) - Comprehensive schema analysis
  - Automatic type detection (integer, float, string, boolean, date)
  - Confidence scoring for type inference
  - Primary key detection (unique + required + name pattern)
  - Foreign key detection (name pattern, cardinality, value overlap)
  - Constraint inference (required, unique)
  - Cardinality analysis
  - Sample value collection

- ✅ `generator.go` (408 lines) - OWL ontology generation
  - Class generation from entities
  - DatatypeProperty for columns
  - ObjectProperty for FK relationships
  - Turtle format output
  - Configurable naming conventions (PascalCase, camelCase, snake_case)

- ✅ **AI Enhancement:** AI/LLM fallback for low-confidence types
  - Semantic type detection (email, phone, currency, URL)
  - Constraint pattern inference
  - Triggered when confidence < threshold (default 0.8)
  - 15% confidence boost when AI used

**Features:**
- Foreign key detection with 3 methods:
  1. Name pattern analysis (`*_id`, `*_ref`, `fk_*`)
  2. Cardinality analysis (5-80% of row count)
  3. Value overlap (≥70% match required)
- Referential integrity calculation
- Multi-method confidence aggregation

**Ontology Management:** `/handlers_ontology.go` (1231 lines)
- Manual OWL/TTL upload: `POST /api/v1/ontology/upload`
- Versioning system (create, list, compare versions)
- Drift detection (from extraction jobs, data, knowledge graph)
- Suggestion management (approve, reject, apply)
- SPARQL query execution
- Entity extraction job management

#### What's Missing for Autonomy

1. **No automatic schema inference trigger after data upload**
   - Engine exists but never called automatically
   - No API endpoint to trigger inference from imported data

2. **No UI for automatic ontology generation**
   - Frontend only supports manual OWL upload
   - Can't select imported data and say "generate ontology"

3. **No automatic ontology creation from inferred schema**
   - Schema inference produces `DataSchema` object
   - Ontology generator produces OWL content
   - But there's no glue code connecting them to the workflow

4. **Entity extraction plugin returns error**
   - Plugin is registered but endpoint fails: `"plugin extraction of type Ontology not found"`
   - Likely registration issue or name mismatch

#### Implementation Requirements

**New API Endpoints:** `/handlers_ontology.go`

```go
// POST /api/v1/data/{importID}/infer-schema
func (s *Server) handleInferSchemaFromImport(w http.ResponseWriter, r *http.Request) {
    importID := mux.Vars(r)["id"]
    
    // 1. Load imported data from storage
    data, err := s.persistence.GetImportedData(ctx, importID)
    
    // 2. Create schema inference engine
    config := schema_inference.InferenceConfig{
        SampleSize:        100,
        EnableConstraints: true,
        EnableFKDetection: true,
        EnableAIFallback:  true,
    }
    engine := schema_inference.NewSchemaInferenceEngineWithLLM(config, s.llmClient)
    
    // 3. Infer schema
    schema, err := engine.InferSchema(data, importName)
    
    // 4. Store schema
    schemaID, err := s.persistence.SaveInferredSchema(ctx, schema)
    
    // 5. Return schema with option to generate ontology
    writeSuccessResponse(w, map[string]any{
        "schema_id": schemaID,
        "schema": schema,
        "next_action": "generate_ontology",
    })
}

// POST /api/v1/schema/{schemaID}/generate-ontology
func (s *Server) handleGenerateOntologyFromSchema(w http.ResponseWriter, r *http.Request) {
    schemaID := mux.Vars(r)["id"]
    
    // 1. Load inferred schema
    schema, err := s.persistence.GetInferredSchema(ctx, schemaID)
    
    // 2. Generate ontology
    ontologyConfig := schema_inference.OntologyConfig{
        BaseURI:        fmt.Sprintf("http://mimir-aip.io/ontology/%s", schemaID),
        OntologyPrefix: "mimir",
        ClassNaming:    "pascal",
        PropertyNaming: "camel",
    }
    generator := schema_inference.NewOntologyGenerator(ontologyConfig)
    ontology, err := generator.GenerateOntology(schema)
    
    // 3. Save ontology to database
    ontologyID, err := s.persistence.CreateOntology(ctx, &Storage.Ontology{
        Name:        ontology.Name,
        Description: ontology.Description,
        Version:     ontology.Version,
        Format:      "turtle",
        FilePath:    saveOntologyFile(ontology.Content),
        TDB2Graph:   ontologyConfig.BaseURI,
        Status:      "active",
    })
    
    // 4. Upload to TDB2
    err = s.tdb2Backend.LoadOntology(ctx, ontology.Content, ontologyConfig.BaseURI)
    
    // 5. Return ontology ID for next step
    writeSuccessResponse(w, map[string]any{
        "ontology_id": ontologyID,
        "next_action": "extract_entities",
    })
}
```

**Frontend Changes:** New page `/mimir-aip-frontend/src/app/data/[id]/schema/page.tsx`

```typescript
// Schema inference and ontology generation UI
export default function SchemaPage({ params }: { params: { id: string } }) {
  const [schema, setSchema] = useState<DataSchema | null>(null);
  const [loading, setLoading] = useState(false);
  
  const handleInferSchema = async () => {
    const response = await fetch(`/api/v1/data/${params.id}/infer-schema`, {
      method: 'POST',
    });
    const data = await response.json();
    setSchema(data.data.schema);
  };
  
  const handleGenerateOntology = async () => {
    const response = await fetch(`/api/v1/schema/${schema.id}/generate-ontology`, {
      method: 'POST',
    });
    // Navigate to ontology page
  };
  
  return (
    // UI showing inferred schema with columns, types, relationships
    // Button to generate ontology
  );
}
```

**Database Changes:**

```sql
CREATE TABLE inferred_schemas (
    id TEXT PRIMARY KEY,
    import_id TEXT,
    name TEXT,
    description TEXT,
    schema_json TEXT,  -- Full DataSchema JSON
    column_count INTEGER,
    fk_count INTEGER,
    created_at TIMESTAMP
);

CREATE TABLE inferred_schema_columns (
    id INTEGER PRIMARY KEY,
    schema_id TEXT,
    column_name TEXT,
    data_type TEXT,
    ontology_type TEXT,
    is_primary_key BOOLEAN,
    is_foreign_key BOOLEAN,
    is_required BOOLEAN,
    is_unique BOOLEAN,
    cardinality INTEGER,
    confidence REAL,
    ai_enhanced BOOLEAN
);
```

**Fix Entity Extraction Plugin Error:**

Check plugin registration in `/server.go` (line 308-313):
```go
// Verify plugin name matches
extractionPlugin := ontology.NewExtractionPlugin(s.persistence.GetDB(), s.tdb2Backend, s.llmClient)
// Plugin reports: GetPluginName() = "extraction", GetPluginType() = "Ontology"
// Registry key should be: "Ontology.extraction"
```

Test with:
```bash
curl -X POST http://localhost:8080/api/v1/ontology/extraction \
  -H "Content-Type: application/json" \
  -d '{
    "ontology_id": "...",
    "source_type": "csv",
    "data": {...}
  }'
```

**Estimated Effort:** 4-5 days

---

### 3. ML Auto-Training System

**Status:** 50% Complete | 30% Connected | 10% Autonomous | 40% UI Friendly

#### What Works

**Backend:** `/handlers_auto_ml.go` (recently fixed)
- ✅ `handleSimpleAutoTrain` - Simplified training endpoint
  - Accepts: `{data, target_column, model_name}`
  - Automatic model type detection (classification vs regression)
  - Creates `DecisionTreeClassifier` or `LinearRegressor`
  - Stores trained model with metrics
  - Fixed: Nullable `ontology_id` in database

- ✅ `/pipelines/ML/auto_trainer.go` - AutoTrainer backend
  - Model training orchestration
  - Hyperparameter tuning (planned, not fully implemented)
  - Model evaluation with proper metrics
  - Storage integration

**API Endpoints:**
- `POST /api/v1/auto-train-with-data` - Train from raw data (WORKING)
- `POST /api/v1/ontology/{id}/auto-train` - Train from ontology (NOT CONNECTED)

**Frontend:** `/mimir-aip-frontend/src/app/models/train/page.tsx`
- Manual training UI with CSV upload
- Target column selection
- Model name input
- Results display with metrics

**Database:** `trained_models` table
- `ontology_id` now nullable (fixed recently)
- Stores model metadata, metrics, file path
- Supports GET/LIST with NULL handling

#### What's Missing for Autonomy

1. **No automatic training trigger after ontology creation**
   - Ontology is created with classes and properties
   - ML training should automatically start
   - Multiple models should be trained for different ontology classes

2. **No ontology → ML integration**
   - Endpoint exists: `POST /api/v1/ontology/{id}/auto-train`
   - But frontend doesn't use it
   - No code to select training data based on ontology classes

3. **No automatic model selection**
   - Currently hardcoded: DecisionTree for classification, Linear for regression
   - AutoML should try multiple algorithms
   - Should select best model based on validation metrics

4. **No continuous retraining**
   - Models trained once and never updated
   - No drift detection on model performance
   - No automatic retraining when new data arrives

#### Implementation Requirements

**New Workflow Handler:** `/handlers_workflow.go`

```go
// Triggered automatically after ontology creation
func (s *Server) triggerAutoMLTraining(ctx context.Context, workflowID, ontologyID string) error {
    // 1. Load ontology metadata
    ontology, err := s.persistence.GetOntology(ctx, ontologyID)
    
    // 2. Get ontology classes (potential ML targets)
    classes, err := s.persistence.GetOntologyClasses(ctx, ontologyID)
    
    // 3. Load original imported data
    workflow, err := s.persistence.GetWorkflow(ctx, workflowID)
    data, err := s.persistence.GetImportedData(ctx, workflow.ImportID)
    
    // 4. For each class, identify potential training targets
    for _, class := range classes {
        // Find columns that match this class
        targetColumn := findTargetColumnForClass(class, data)
        if targetColumn == "" {
            continue
        }
        
        // 5. Train multiple models
        models := []string{"decision_tree", "random_forest", "gradient_boost", "svm"}
        bestModel := ""
        bestScore := 0.0
        
        for _, modelType := range models {
            // Train model
            result, err := s.trainModel(ctx, data, targetColumn, modelType)
            if err != nil {
                log.Printf("Failed to train %s: %v", modelType, err)
                continue
            }
            
            // Track best model
            if result.Score > bestScore {
                bestScore = result.Score
                bestModel = result.ModelID
            }
        }
        
        // 6. Store best model ID with ontology mapping
        s.persistence.LinkModelToOntologyClass(ctx, bestModel, ontologyID, class.URI)
    }
    
    // 7. Update workflow to next step
    s.updateWorkflowStep(ctx, workflowID, "digital_twin_creation")
    
    return nil
}
```

**Enhanced AutoTrainer:** `/pipelines/ML/auto_trainer.go`

```go
// Add ensemble training capability
type AutoTrainer struct {
    // ... existing fields
    modelRegistry map[string]ModelFactory
}

type ModelFactory interface {
    CreateModel(config ModelConfig) TrainableModel
    GetHyperparameterSpace() map[string]interface{}
}

func (at *AutoTrainer) TrainEnsemble(data []map[string]interface{}, target string) (*EnsembleResult, error) {
    var models []*TrainedModel
    
    // Try multiple algorithms
    for modelType, factory := range at.modelRegistry {
        model, err := at.trainSingleModel(data, target, modelType)
        if err != nil {
            continue
        }
        models = append(models, model)
    }
    
    // Select best based on validation metrics
    best := selectBestModel(models)
    
    return &EnsembleResult{
        BestModel: best,
        AllModels: models,
    }, nil
}
```

**New Frontend Page:** `/mimir-aip-frontend/src/app/ontologies/[id]/train/page.tsx`

```typescript
// Automatic ML training UI from ontology
export default function OntologyMLTrainPage({ params }: { params: { id: string } }) {
  const handleAutoTrain = async () => {
    const response = await fetch(`/api/v1/ontology/${params.id}/auto-train`, {
      method: 'POST',
      body: JSON.stringify({
        enable_auto_ml: true,
        algorithms: ['decision_tree', 'random_forest', 'gradient_boost'],
      }),
    });
    
    // Show training progress
    // Display trained models
  };
  
  return (
    <div>
      <h1>Train ML Models from Ontology</h1>
      <Button onClick={handleAutoTrain}>Start Auto-Training</Button>
      {/* Show ontology classes and target columns */}
    </div>
  );
}
```

**Database Changes:**

```sql
CREATE TABLE ontology_model_mappings (
    id INTEGER PRIMARY KEY,
    ontology_id TEXT,
    class_uri TEXT,
    model_id TEXT,
    target_column TEXT,
    confidence REAL,
    created_at TIMESTAMP
);

CREATE TABLE model_training_history (
    id INTEGER PRIMARY KEY,
    workflow_id TEXT,
    model_id TEXT,
    algorithm TEXT,
    validation_score REAL,
    training_duration_ms INTEGER,
    status TEXT,
    created_at TIMESTAMP
);
```

**Estimated Effort:** 5-6 days

---

### 4. Digital Twin Integration

**Status:** 50% Complete | 40% Connected | 10% Autonomous | 70% UI Friendly

#### What Works

**Backend:** `/pipelines/DigitalTwin/` & `/handlers_digital_twin.go`
- ✅ Twin creation with state management
- ✅ Scenario generation (recently fixed to include events)
- ✅ Simulation engine with temporal state
- ✅ Event system for state changes
- ✅ Multiple simulation execution

**Frontend:** `/mimir-aip-frontend/src/app/digital-twins/`
- Manual twin creation UI
- Scenario display and simulation runner
- Results visualization

**Features:**
- State management with `temporal_state.go`
- Event-driven updates
- Scenario builder with `scenario_builder.go`
- Simulation results storage

#### What's Missing for Autonomy

1. **No ML model integration in twins**
   - Twins have state but don't use trained models
   - No predictions during simulation
   - No anomaly detection with models

2. **No automatic twin creation after ML training**
   - Models trained → should create twin automatically
   - Twin should load relevant models for predictions

3. **No continuous data ingestion for twins**
   - Twins created once, never updated
   - No streaming data into twin state
   - No real-time simulation

4. **No anomaly detection during simulation**
   - Simulations run but no alerts
   - No threshold checking
   - No comparison with expected values

#### Implementation Requirements

**Enhanced Digital Twin:** `/pipelines/DigitalTwin/ml_integrated_twin.go` (NEW FILE)

```go
package digitaltwin

import (
    "github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/ML"
)

type MLIntegratedTwin struct {
    *DigitalTwin
    models      map[string]*ML.TrainedModel  // property -> model
    predictor   *ML.Predictor
    anomalyDet  *AnomalyDetector
}

func (mit *MLIntegratedTwin) UpdateStateWithPrediction(ctx context.Context, input map[string]interface{}) error {
    // 1. Get current state
    currentState := mit.GetCurrentState()
    
    // 2. For each property with a model, make prediction
    for property, model := range mit.models {
        prediction, err := mit.predictor.Predict(model, input)
        if err != nil {
            continue
        }
        
        // 3. Update state with prediction
        currentState[property] = prediction
        
        // 4. Check for anomalies
        if mit.anomalyDet != nil {
            if anomaly := mit.anomalyDet.Detect(property, prediction); anomaly != nil {
                mit.handleAnomaly(anomaly)
            }
        }
    }
    
    // 5. Save state
    return mit.SaveState(ctx, currentState)
}

func (mit *MLIntegratedTwin) handleAnomaly(anomaly *Anomaly) {
    // Store anomaly
    mit.storeAnomaly(anomaly)
    
    // Trigger alert
    mit.triggerAlert(anomaly)
}
```

**Workflow Integration:** `/handlers_workflow.go`

```go
func (s *Server) createDigitalTwinFromModels(ctx context.Context, workflowID string) error {
    workflow, _ := s.persistence.GetWorkflow(ctx, workflowID)
    
    // 1. Get trained models for this workflow
    models, _ := s.persistence.GetModelsForWorkflow(ctx, workflowID)
    
    // 2. Create digital twin
    twin := &DigitalTwin{
        Name:        fmt.Sprintf("Twin for %s", workflow.Name),
        Description: "Auto-generated twin with ML models",
        OntologyID:  workflow.OntologyID,
    }
    twinID, _ := s.createDigitalTwin(ctx, twin)
    
    // 3. Link models to twin properties
    for _, model := range models {
        s.linkModelToTwinProperty(ctx, twinID, model.ID, model.TargetColumn)
    }
    
    // 4. Start monitoring
    go s.startTwinMonitoring(ctx, twinID)
    
    return nil
}

func (s *Server) startTwinMonitoring(ctx context.Context, twinID string) {
    ticker := time.NewTicker(5 * time.Minute)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            // Get latest data
            // Update twin state
            // Check for anomalies
        case <-ctx.Done():
            return
        }
    }
}
```

**Database Changes:**

```sql
CREATE TABLE twin_model_links (
    id INTEGER PRIMARY KEY,
    twin_id TEXT,
    model_id TEXT,
    property_name TEXT,
    created_at TIMESTAMP
);

CREATE TABLE anomalies (
    id INTEGER PRIMARY KEY,
    twin_id TEXT,
    property_name TEXT,
    expected_value REAL,
    actual_value REAL,
    deviation REAL,
    severity TEXT,  -- low, medium, high, critical
    detected_at TIMESTAMP,
    acknowledged BOOLEAN,
    acknowledged_by TEXT,
    acknowledged_at TIMESTAMP
);

CREATE TABLE twin_monitoring_jobs (
    id TEXT PRIMARY KEY,
    twin_id TEXT,
    status TEXT,  -- running, paused, stopped
    check_interval_seconds INTEGER,
    last_check_at TIMESTAMP,
    created_at TIMESTAMP
);
```

**Estimated Effort:** 4-5 days

---

### 5. Alerting & Notification System

**Status:** 5% Complete | 0% Connected | 0% Autonomous | 0% UI Friendly

#### What Works
- ❌ Nothing - completely missing

#### What's Missing
1. **No notification plugins**
   - Slack integration
   - Discord webhooks
   - Email notifications
   - SMS alerts (Twilio)

2. **No alert routing**
   - No rules engine
   - No severity-based routing
   - No escalation policies

3. **No alert dashboard**
   - Can't see active alerts
   - Can't acknowledge/silence
   - No alert history

#### Implementation Requirements

**New Notification Plugins:** `/pipelines/Output/` (add to existing plugins)

**File:** `/pipelines/Output/slack_notification_plugin.go`

```go
package output

type SlackNotificationPlugin struct {
    webhookURL string
}

func (p *SlackNotificationPlugin) Send(message NotificationMessage) error {
    payload := map[string]interface{}{
        "text": message.Title,
        "blocks": []map[string]interface{}{
            {
                "type": "section",
                "text": map[string]string{
                    "type": "mrkdwn",
                    "text": message.Body,
                },
            },
            {
                "type": "context",
                "elements": []map[string]string{
                    {
                        "type": "mrkdwn",
                        "text": fmt.Sprintf("*Severity:* %s | *Time:* %s", 
                            message.Severity, message.Timestamp),
                    },
                },
            },
        },
    }
    
    return postToWebhook(p.webhookURL, payload)
}
```

**Alert Manager:** `/utils/alert_manager.go` (NEW FILE)

```go
package utils

type AlertManager struct {
    db             *sql.DB
    notifications  map[string]NotificationPlugin
    routingRules   []AlertRoutingRule
}

type AlertRoutingRule struct {
    Severity     string
    TwinID       string
    Channels     []string  // slack, email, discord
    EscalateAfter time.Duration
}

func (am *AlertManager) ProcessAnomaly(anomaly *Anomaly) error {
    // 1. Create alert
    alert := &Alert{
        ID:          uuid.New().String(),
        Type:        "anomaly",
        Severity:    anomaly.Severity,
        Title:       fmt.Sprintf("Anomaly detected in %s", anomaly.PropertyName),
        Description: fmt.Sprintf("Value %.2f deviates by %.2f from expected", 
            anomaly.ActualValue, anomaly.Deviation),
        TwinID:      anomaly.TwinID,
        CreatedAt:   time.Now(),
        Status:      "active",
    }
    
    // 2. Store alert
    am.storeAlert(alert)
    
    // 3. Find routing rules
    rules := am.findMatchingRules(alert)
    
    // 4. Send notifications
    for _, rule := range rules {
        for _, channel := range rule.Channels {
            if plugin, exists := am.notifications[channel]; exists {
                plugin.Send(alert.ToNotification())
            }
        }
    }
    
    // 5. Schedule escalation if needed
    if rule.EscalateAfter > 0 {
        am.scheduleEscalation(alert, rule)
    }
    
    return nil
}
```

**Alert API:** `/handlers_alerts.go` (NEW FILE)

```go
// GET /api/v1/alerts
func (s *Server) handleListAlerts(w http.ResponseWriter, r *http.Request) {
    status := r.URL.Query().Get("status")  // active, acknowledged, resolved
    severity := r.URL.Query().Get("severity")
    
    alerts, _ := s.alertManager.ListAlerts(status, severity)
    writeSuccessResponse(w, alerts)
}

// POST /api/v1/alerts/{id}/acknowledge
func (s *Server) handleAcknowledgeAlert(w http.ResponseWriter, r *http.Request) {
    alertID := mux.Vars(r)["id"]
    userID := r.Header.Get("X-User-ID")
    
    s.alertManager.AcknowledgeAlert(alertID, userID)
    writeSuccessResponse(w, map[string]string{"status": "acknowledged"})
}

// POST /api/v1/alerts/configure
func (s *Server) handleConfigureAlertRouting(w http.ResponseWriter, r *http.Request) {
    var config AlertRoutingConfig
    json.NewDecoder(r.Body).Decode(&config)
    
    s.alertManager.UpdateRoutingRules(config.Rules)
    writeSuccessResponse(w, map[string]string{"status": "updated"})
}
```

**Frontend:** `/mimir-aip-frontend/src/app/alerts/page.tsx` (NEW FILE)

```typescript
export default function AlertsPage() {
  const [alerts, setAlerts] = useState<Alert[]>([]);
  
  const loadAlerts = async () => {
    const response = await fetch('/api/v1/alerts');
    const data = await response.json();
    setAlerts(data.data);
  };
  
  const acknowledgeAlert = async (alertId: string) => {
    await fetch(`/api/v1/alerts/${alertId}/acknowledge`, { method: 'POST' });
    loadAlerts();
  };
  
  return (
    <div>
      <h1>Alert Dashboard</h1>
      <AlertList alerts={alerts} onAcknowledge={acknowledgeAlert} />
    </div>
  );
}
```

**Database Changes:**

```sql
CREATE TABLE alerts (
    id TEXT PRIMARY KEY,
    type TEXT,  -- anomaly, threshold, system
    severity TEXT,  -- low, medium, high, critical
    title TEXT,
    description TEXT,
    twin_id TEXT,
    workflow_id TEXT,
    status TEXT,  -- active, acknowledged, resolved
    acknowledged_by TEXT,
    acknowledged_at TIMESTAMP,
    resolved_at TIMESTAMP,
    created_at TIMESTAMP
);

CREATE TABLE alert_routing_rules (
    id INTEGER PRIMARY KEY,
    severity TEXT,
    twin_id TEXT,
    channel TEXT,  -- slack, email, discord
    webhook_url TEXT,
    escalate_after_minutes INTEGER,
    enabled BOOLEAN
);

CREATE TABLE alert_notifications (
    id INTEGER PRIMARY KEY,
    alert_id TEXT,
    channel TEXT,
    status TEXT,  -- sent, failed, pending
    sent_at TIMESTAMP,
    error_message TEXT
);
```

**Estimated Effort:** 3-4 days

---

### 6. Job Scheduling Integration

**Status:** 60% Complete | 20% Connected | 10% Autonomous | 50% UI Friendly

#### What Works
- ✅ Job CRUD operations: `/utils/scheduler.go`
- ✅ Cron scheduling support
- ✅ Job status tracking
- ✅ Frontend UI: `/mimir-aip-frontend/src/app/jobs/page.tsx`

#### What's Missing
1. **No UI to create data ingestion jobs**
   - Can't schedule recurring CSV imports
   - Can't link database connections to jobs

2. **No automatic workflow scheduling**
   - After initial autonomous workflow completes
   - Should schedule recurring ingestion + full pipeline

3. **Job execution not tested**
   - Jobs can be created
   - But unclear if they actually execute

#### Implementation Requirements

**Testing:** Verify job execution works

```bash
# Create a test job
curl -X POST http://localhost:8080/api/v1/jobs \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Test Recurring Import",
    "schedule": "*/5 * * * *",
    "pipeline_config": {
      "steps": [
        {
          "name": "import_csv",
          "plugin": "Input.csv",
          "config": {
            "file_path": "/data/products.csv"
          }
        }
      ]
    }
  }'

# Wait 5 minutes, check if job executed
curl http://localhost:8080/api/v1/jobs/{job_id}/executions
```

**Frontend Enhancement:** `/mimir-aip-frontend/src/app/workflows/[id]/schedule/page.tsx`

```typescript
export default function ScheduleWorkflowPage({ params }: { params: { id: string } }) {
  const handleSchedule = async (cronExpression: string) => {
    await fetch('/api/v1/jobs', {
      method: 'POST',
      body: JSON.stringify({
        name: `Recurring workflow ${params.id}`,
        schedule: cronExpression,
        workflow_id: params.id,
      }),
    });
  };
  
  return (
    <div>
      <h1>Schedule Autonomous Workflow</h1>
      <CronEditor onSave={handleSchedule} />
    </div>
  );
}
```

**Estimated Effort:** 2-3 days

---

## Missing Components for Autonomy

### Critical Missing Pieces (Must Have)

1. **Autonomous Workflow Orchestrator** - 0% complete
   - Central state machine managing the full pipeline
   - Chains: Data Ingestion → Schema Inference → Ontology Generation → Entity Extraction → ML Training → Digital Twin → Monitoring
   - Error handling and retry logic
   - Progress tracking

2. **Automatic Trigger System** - 0% complete
   - Webhook after data upload
   - Event bus for component communication
   - Step completion listeners

3. **Data Flow Connectors** - 30% complete
   - Import data → Schema inference ❌
   - Schema → Ontology generation ❌
   - Ontology → Entity extraction ❌
   - Ontology + Data → ML training ❌
   - ML models → Digital twin ❌
   - Twin → Anomaly detection ❌
   - Anomalies → Alerts ❌

4. **Monitoring & Alerting** - 5% complete
   - Continuous monitoring loops
   - Alert routing
   - Notification plugins

### Important But Not Critical

5. **Agent Chat Integration** - 20% complete
   - Currently just Q&A
   - Should allow: "Create pipeline from CSV"
   - Should allow: "Show me ontology for products"

6. **UI Improvements**
   - Workflow status dashboard
   - Real-time progress indicators
   - One-click "Make Autonomous" button

---

## Detailed Implementation Requirements

### Phase 1: Core Automation (Weeks 1-2)

**Goal:** Connect existing components into a working autonomous pipeline

#### Week 1: Workflow Orchestrator

**Tasks:**
1. Create workflow state machine
2. Implement step transitions
3. Add error handling
4. Build progress tracking

**Files to Create:**
- `/pipelines/Workflow/orchestrator.go`
- `/pipelines/Workflow/state_machine.go`
- `/handlers_workflow.go`
- Database migration for workflow tables

**Code Example:**

```go
type WorkflowOrchestrator struct {
    db         *sql.DB
    steps      []WorkflowStep
    currentIdx int
    status     WorkflowStatus
}

func (wo *WorkflowOrchestrator) Execute(ctx context.Context) error {
    for wo.currentIdx < len(wo.steps) {
        step := wo.steps[wo.currentIdx]
        
        // Execute step
        result, err := step.Execute(ctx)
        if err != nil {
            wo.handleError(err, step)
            return err
        }
        
        // Store result for next step
        wo.storeStepResult(step.Name, result)
        
        // Move to next step
        wo.currentIdx++
        wo.updateStatus()
    }
    
    return nil
}
```

#### Week 2: Data Connectors

**Tasks:**
1. Hook data upload to schema inference
2. Hook schema inference to ontology generation
3. Hook ontology to ML training
4. Hook ML models to digital twins

**Files to Modify:**
- `/handlers.go` - Add workflow trigger in `handleDataImport`
- `/handlers_ontology.go` - Add auto-generation endpoints
- `/handlers_auto_ml.go` - Add ontology-based training
- `/handlers_digital_twin.go` - Add model integration

---

### Phase 2: ML & Twin Enhancement (Weeks 3-4)

#### Week 3: Enhanced ML Training

**Tasks:**
1. Implement ensemble training
2. Add automatic model selection
3. Link models to ontology classes
4. Build training dashboard

**Files:**
- `/pipelines/ML/ensemble_trainer.go` (NEW)
- `/pipelines/ML/model_selector.go` (NEW)
- Enhanced `/handlers_auto_ml.go`
- Frontend: `/mimir-aip-frontend/src/app/models/auto-train/page.tsx` (NEW)

#### Week 4: ML-Integrated Digital Twins

**Tasks:**
1. Create ML-integrated twin class
2. Add prediction capability to twins
3. Implement anomaly detection
4. Build twin monitoring system

**Files:**
- `/pipelines/DigitalTwin/ml_integrated_twin.go` (NEW)
- `/pipelines/DigitalTwin/anomaly_detector.go` (NEW)
- `/pipelines/DigitalTwin/monitoring_service.go` (NEW)

---

### Phase 3: Monitoring & Alerting (Week 5)

#### Week 5: Alert System

**Tasks:**
1. Build alert manager
2. Create notification plugins (Slack, Email, Discord)
3. Implement alert routing
4. Build alert dashboard UI

**Files:**
- `/utils/alert_manager.go` (NEW)
- `/pipelines/Output/slack_notification_plugin.go` (NEW)
- `/pipelines/Output/email_notification_plugin.go` (NEW)
- `/handlers_alerts.go` (NEW)
- Frontend: `/mimir-aip-frontend/src/app/alerts/page.tsx` (NEW)

---

### Phase 4: Polish & Testing (Week 6)

#### Week 6: Integration Testing & UI Polish

**Tasks:**
1. End-to-end autonomous pipeline test
2. Performance optimization
3. UI improvements (progress indicators, dashboards)
4. Documentation

**Tests:**
- Upload CSV → verify ontology created
- Verify ML models trained
- Verify digital twin created
- Verify monitoring started
- Trigger anomaly → verify alert sent

---

## Implementation Phases

### Phase 0: Foundation (Current) ✅
- Core components exist
- Manual workflows function
- Gap analysis complete

### Phase 1: Basic Autonomy (Weeks 1-2)
**Deliverable:** Upload CSV → Ontology + ML models created automatically

**Success Criteria:**
- Data upload triggers workflow
- Schema inference runs automatically
- Ontology generated from schema
- ML models trained from ontology
- All steps tracked in database

### Phase 2: ML Integration (Weeks 3-4)
**Deliverable:** Trained models integrated into digital twins with predictions

**Success Criteria:**
- Digital twins load trained models
- Twins make predictions on new data
- Anomaly detection functional
- Twin state updates automatically

### Phase 3: Monitoring (Week 5)
**Deliverable:** Continuous monitoring with alerts

**Success Criteria:**
- Monitoring jobs run on schedule
- Anomalies detected in real-time
- Alerts sent via Slack/Email
- Alert dashboard shows active alerts

### Phase 4: Full Autonomy (Week 6)
**Deliverable:** Complete autonomous system with scheduling

**Success Criteria:**
- Recurring data ingestion jobs
- Full pipeline runs on schedule
- Zero manual intervention needed
- Performance metrics meet targets

---

## Testing Strategy

### Unit Tests
- ✅ Existing: ML training, pipeline execution
- ⚠️ Missing: Workflow orchestrator, alert manager
- **Add:** 50+ new unit tests for autonomous components

### Integration Tests
- ✅ Existing: Digital twin workflow, ML training workflow
- **Add:** End-to-end autonomous pipeline test
- **Add:** Multi-step workflow tests

### E2E Tests (Playwright)
- ✅ Existing: Login, navigation, data upload
- **Add:** Autonomous workflow UI test
- **Add:** Alert dashboard test
- **Add:** Model training from ontology test

### Performance Tests
- Load test: 1000 rows → ontology generation (target: <30s)
- Concurrent workflows: 10 simultaneous pipelines
- Monitoring loop: 100 twins, 5-minute intervals

---

## Summary: Path to Autonomy

### Current Reality
- **15% autonomous:** Components exist but disconnected
- **Manual intervention required:** Every step needs user action
- **Solid foundation:** 70% of code is there, just needs wiring

### Target State
- **95% autonomous:** User uploads data, system does the rest
- **Minimal intervention:** Only for approvals and configuration
- **Intelligent system:** Learns, adapts, and alerts proactively

### Estimated Timeline
- **6 weeks** for complete autonomous system
- **2 weeks** for basic working prototype
- **4 weeks** for production-ready system

### Biggest Challenges
1. **Workflow orchestration complexity** - State management across steps
2. **Error handling** - What happens when a step fails?
3. **Performance** - Can system handle 1000-row CSV in real-time?
4. **Testing** - E2E tests for async multi-step workflows

### Quick Wins (Can implement in 1-2 days)
1. Fix entity extraction plugin registration ✅
2. Add "Infer Schema" button on data upload page
3. Create basic workflow status dashboard
4. Add Slack notification plugin

---

## Next Steps

**Immediate (This Week):**
1. Fix extraction plugin error
2. Create `/handlers_workflow.go` with basic orchestrator
3. Add workflow tables to database
4. Test schema inference → ontology generation manually

**Short Term (Next 2 Weeks):**
1. Implement Phase 1 (Core Automation)
2. Build end-to-end test
3. Deploy to staging environment

**Medium Term (Weeks 3-6):**
1. Implement Phases 2-4
2. Performance testing and optimization
3. Production deployment

---

**Document Status:** Complete  
**Ready for Implementation:** Yes  
**Approval Needed:** Architecture review recommended before Phase 1
