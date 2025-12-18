# Ontology-Driven Automatic ML Training Design

## The Core Problem

**Current State (BAD):**
```bash
# User needs to know ML terminology
POST /api/v1/models/train
{
  "model_type": "regression",        # ❌ User needs to know what regression is
  "target_column": "price",          # ❌ User needs to know ML concepts
  "feature_columns": ["sqft", "bedrooms"],  # ❌ Manual feature selection
  "train_data": [...]                # ❌ Manual data preparation
}
```

**What We Want (GOOD):**
```bash
# User just describes intent in natural language
POST /api/v1/ontologies/{id}/analyze
{
  "goal": "predict product prices based on inventory data"
}

# OR even simpler - Mimir auto-suggests based on ontology
GET /api/v1/ontologies/{id}/ml-suggestions
# Returns: "I can predict prices, stock depletion dates, and detect anomalies"
```

## Design Philosophy: Ontology is the Schema + ML Specification

### Key Insight
The **ontology already tells us everything we need to know** about the data:

1. **Property Types** → Model Type
   - `Range: xsd:integer`, `xsd:float`, `xsd:decimal` → **Regression**
   - `Range: xsd:string`, `xsd:boolean`, URI → **Classification**
   - `Range: xsd:dateTime` → **Time-Series Analysis**

2. **Property Semantics** → Training Intent
   - `rdfs:label: "price"` → This is a target for prediction
   - `rdfs:label: "stock_level"` → Can monitor for thresholds
   - `rdfs:label: "status"` → Categorical classification target

3. **Domain/Range Relationships** → Feature Engineering
   - `domain: :Product`, `range: :Category` → Use category as feature
   - Object properties create entity relationships → Graph features

4. **Temporal Properties** → Monitoring Setup
   - Properties with `xsd:dateTime` values → Automatically create time-series
   - Update frequency → Auto-suggest monitoring intervals

## Proposed Architecture

### Layer 1: Ontology Analysis Engine

**File: `pipelines/ML/ontology_analyzer.go`**

```go
type OntologyAnalyzer struct {
    Storage *storage.PersistenceBackend
}

// Analyze inspects ontology and returns ML capabilities
func (oa *OntologyAnalyzer) AnalyzeMLCapabilities(ontologyID string) (*MLCapabilities, error) {
    // 1. Get all properties for this ontology
    // 2. Identify numeric properties (regression targets)
    // 3. Identify categorical properties (classification targets)
    // 4. Identify temporal properties (time-series monitoring)
    // 5. Identify relationships (features)
    // 6. Return structured capabilities
}

type MLCapabilities struct {
    OntologyID          string                 `json:"ontology_id"`
    RegressionTargets   []MLTarget             `json:"regression_targets"`
    ClassificationTargets []MLTarget           `json:"classification_targets"`
    TimeSeriesMetrics   []TimeSeriesMetric     `json:"timeseries_metrics"`
    MonitoringRules     []SuggestedMonitoringRule `json:"suggested_rules"`
}

type MLTarget struct {
    PropertyURI   string   `json:"property_uri"`
    PropertyLabel string   `json:"property_label"`
    Description   string   `json:"description"`
    SuggestedFeatures []string `json:"suggested_features"` // Other properties to use
    Confidence    float64  `json:"confidence"`
    Reasoning     string   `json:"reasoning"`
}
```

**Example Analysis Result:**
```json
{
  "ontology_id": "ont_computer_shop",
  "regression_targets": [
    {
      "property_uri": "shop:price",
      "property_label": "Price",
      "description": "Product price (numeric)",
      "suggested_features": ["shop:stock_level", "shop:category", "shop:rating"],
      "confidence": 0.95,
      "reasoning": "Price is numeric (xsd:decimal), has sufficient related properties for prediction"
    },
    {
      "property_uri": "shop:profit_margin",
      "property_label": "Profit Margin",
      "suggested_features": ["shop:price", "shop:cost", "shop:category"],
      "confidence": 0.90,
      "reasoning": "Profit margin can be predicted from price and cost relationships"
    }
  ],
  "classification_targets": [
    {
      "property_uri": "shop:category",
      "property_label": "Product Category",
      "suggested_features": ["shop:name", "shop:description", "shop:price_range"],
      "confidence": 0.85,
      "reasoning": "Category is categorical (xsd:string), can classify products"
    }
  ],
  "timeseries_metrics": [
    {
      "property_uri": "shop:stock_level",
      "property_label": "Stock Level",
      "metric_type": "gauge",
      "suggested_monitoring": ["threshold", "trend", "anomaly"],
      "reasoning": "Stock level changes over time, critical for inventory management"
    },
    {
      "property_uri": "shop:price",
      "property_label": "Price",
      "metric_type": "gauge",
      "suggested_monitoring": ["trend", "forecast"],
      "reasoning": "Price trends indicate market conditions"
    }
  ],
  "suggested_rules": [
    {
      "rule_type": "threshold",
      "metric": "shop:stock_level",
      "condition": {"operator": "<", "value": 5},
      "severity": "high",
      "reasoning": "Low stock alerts are critical for business continuity"
    },
    {
      "rule_type": "trend",
      "metric": "shop:price",
      "condition": {"expected": "increasing", "min_change_percent": 15},
      "severity": "medium",
      "reasoning": "Significant price increases may indicate market changes"
    }
  ]
}
```

### Layer 2: Auto-Training System

**File: `pipelines/ML/auto_trainer.go`**

```go
type AutoTrainer struct {
    Storage    *storage.PersistenceBackend
    Analyzer   *OntologyAnalyzer
    Trainer    *Trainer
}

// TrainFromOntology automatically trains models based on ontology
func (at *AutoTrainer) TrainFromOntology(ctx context.Context, ontologyID string) (*AutoTrainingResult, error) {
    // 1. Analyze ontology capabilities
    capabilities, _ := at.Analyzer.AnalyzeMLCapabilities(ontologyID)
    
    // 2. Query data from knowledge graph
    data, _ := at.queryOntologyData(ontologyID)
    
    // 3. For each suggested target, train a model
    for _, target := range capabilities.RegressionTargets {
        at.trainRegressionModel(ontologyID, target, data)
    }
    
    for _, target := range capabilities.ClassificationTargets {
        at.trainClassificationModel(ontologyID, target, data)
    }
    
    // 4. Create monitoring jobs automatically
    for _, metric := range capabilities.TimeSeriesMetrics {
        at.createMonitoringJob(ontologyID, metric)
    }
    
    // 5. Create monitoring rules automatically
    for _, rule := range capabilities.SuggestedRules {
        at.createMonitoringRule(ontologyID, rule)
    }
}

// TrainForGoal trains model(s) based on natural language goal
func (at *AutoTrainer) TrainForGoal(ctx context.Context, ontologyID, goal string) (*AutoTrainingResult, error) {
    // 1. Use LLM to parse goal
    //    "predict product prices" → {target: "price", type: "regression"}
    
    // 2. Analyze ontology to find matching properties
    
    // 3. Train appropriate model
    
    // 4. Return results
}
```

### Layer 3: Data Extraction from Knowledge Graph

**File: `pipelines/ML/kg_data_extractor.go`**

The key insight: **Data is already in the knowledge graph**, we just need to query it!

```go
type KGDataExtractor struct {
    Storage *storage.PersistenceBackend
}

// ExtractTrainingData queries KG and converts to ML-ready format
func (kde *KGDataExtractor) ExtractTrainingData(
    ontologyID string,
    targetProperty string,
    featureProperties []string,
) (X [][]float64, y interface{}, featureNames []string, err error) {
    
    // 1. Query all entities of relevant class from KG
    //    SELECT ?entity ?prop1 ?prop2 ... ?target
    //    WHERE {
    //      ?entity rdf:type :Product .
    //      ?entity :stock_level ?prop1 .
    //      ?entity :category ?prop2 .
    //      ?entity :price ?target .
    //    }
    
    // 2. Convert RDF results to ML format
    //    - Numeric values → keep as-is
    //    - Categorical values → encode (label encoding or one-hot)
    //    - URIs → encode entity IDs
    //    - Missing values → handle (mean imputation, drop, etc.)
    
    // 3. Detect target type
    //    - All numeric → regression (return []float64)
    //    - All strings/URIs → classification (return []string)
    
    // 4. Return ML-ready data
}
```

**SPARQL Example:**
```sparql
PREFIX shop: <http://example.org/shop#>
PREFIX xsd: <http://www.w3.org/2001/XMLSchema#>

SELECT ?product ?stock ?category ?rating ?price
WHERE {
  ?product rdf:type shop:Product .
  ?product shop:stock_level ?stock .
  ?product shop:category ?category .
  ?product shop:rating ?rating .
  ?product shop:price ?price .
  FILTER (xsd:decimal(?price) > 0)
}
```

### Layer 4: Smart Feature Engineering

**Based on Ontology Relationships:**

```go
func (oa *OntologyAnalyzer) suggestFeatures(targetProperty string, properties []OntologyProperty) []string {
    features := []string{}
    
    for _, prop := range properties {
        if prop.URI == targetProperty {
            continue // Don't use target as feature
        }
        
        // Heuristics:
        // 1. Numeric properties are good features
        if isNumericRange(prop.Range) {
            features = append(features, prop.URI)
        }
        
        // 2. Categorical properties with low cardinality
        if isLowCardinalityCategorical(prop) {
            features = append(features, prop.URI)
        }
        
        // 3. Properties in same domain as target
        if hasSameDomain(prop, targetProperty) {
            features = append(features, prop.URI)
        }
        
        // 4. Properties with semantic similarity (use LLM)
        if semanticallyRelated(prop.Label, targetProperty.Label) {
            features = append(features, prop.URI)
        }
    }
    
    return features
}
```

### Layer 5: Automatic Monitoring Setup

```go
func (at *AutoTrainer) setupAutomaticMonitoring(ontologyID string, capabilities *MLCapabilities) error {
    // 1. Create monitoring job for ontology
    monitoringJob := &MonitoringJob{
        ID:          fmt.Sprintf("monitor_%s", ontologyID),
        Name:        fmt.Sprintf("Auto-Monitor %s", ontologyID),
        OntologyID:  ontologyID,
        CronExpr:    "*/15 * * * *", // Every 15 minutes
        JobType:     "time_series_analysis",
        Enabled:     true,
    }
    at.Storage.CreateMonitoringJob(monitoringJob)
    
    // 2. Create rules for each time-series metric
    for _, metric := range capabilities.TimeSeriesMetrics {
        for _, ruleType := range metric.SuggestedMonitoring {
            rule := at.createRule(ontologyID, metric, ruleType)
            at.Storage.CreateMonitoringRule(rule)
        }
    }
}
```

## User Workflows

### Workflow 1: Zero-Code Setup (Computer Shop)

```bash
# 1. User uploads CSV
POST /api/v1/data/upload
{
  "file": "inventory.csv",
  "ontology_name": "Computer Shop Inventory"
}

# Mimir auto-generates ontology with:
# - Classes: Product, Category
# - Properties: price (xsd:decimal), stock_level (xsd:int), category (xsd:string), ...

# 2. Mimir analyzes ontology
# Internally calls: OntologyAnalyzer.AnalyzeMLCapabilities()

# 3. User asks: "What can you do with this data?"
GET /api/v1/ontologies/ont_computer_shop/ml-capabilities

# Response:
{
  "message": "I can help with the following:",
  "capabilities": {
    "predictions": [
      "Predict product prices (regression)",
      "Predict profit margins (regression)",
      "Classify product categories (classification)"
    ],
    "monitoring": [
      "Alert when stock levels are low",
      "Detect unusual price changes",
      "Forecast when items will be out of stock"
    ]
  }
}

# 4. User says: "Yes, do all of that"
POST /api/v1/ontologies/ont_computer_shop/auto-train
{
  "enable_all": true
}

# Mimir automatically:
# ✅ Trains regression model: predict price from (stock, category, rating)
# ✅ Trains regression model: predict margin from (price, cost, category)
# ✅ Trains classifier: predict category from (name, description)
# ✅ Creates monitoring job (runs every 15 min)
# ✅ Creates threshold rule: stock < 5 → high severity alert
# ✅ Creates trend rule: price increasing > 15% → medium severity alert
# ✅ Creates forecast rule: stock depletion < 7 days → high severity alert

# 5. User schedules daily data ingestion
POST /api/v1/scheduler/jobs
{
  "pipeline": "fetch_inventory_daily.yaml",
  "cron_expr": "0 9 * * *"
}

# DONE! Zero ML knowledge required.
```

### Workflow 2: Goal-Based Training (NGO)

```bash
# 1. User has uploaded aid delivery data
# Ontology auto-generated with:
# - Classes: SupplyDelivery, Location, Resource
# - Properties: delivery_date, quantity, location, resource_type, ...

# 2. User describes goal in natural language
POST /api/v1/ontologies/ont_aid_logistics/train-for-goal
{
  "goal": "I need to know how many days until we run out of medical supplies"
}

# Mimir:
# - Parses goal with LLM: "days until depletion" → regression target
# - Finds property: "quantity" with temporal data
# - Trains forecast model: predict depletion date from (current_stock, delivery_rate, consumption_rate)
# - Creates monitoring rule: alert when depletion < 7 days
# - Returns model ID + monitoring job ID

# 3. User can ask follow-up
POST /api/v1/ontologies/ont_aid_logistics/train-for-goal
{
  "goal": "Detect when disease outbreaks are starting"
}

# Mimir:
# - Parses goal: "detect outbreaks" → anomaly detection
# - Finds relevant properties: disease_cases, location, date
# - Sets up time-series anomaly detection
# - Creates alert rule: spike in cases → critical alert
```

### Workflow 3: Hybrid (User Tweaks Auto-Suggestions)

```bash
# 1. Get ML suggestions
GET /api/v1/ontologies/ont_shop/ml-suggestions

# Response:
{
  "suggested_models": [
    {
      "id": "model_price_prediction",
      "target": "price",
      "model_type": "regression",
      "features": ["stock_level", "category", "rating"],
      "confidence": 0.95
    }
  ]
}

# 2. User accepts but wants to exclude a feature
POST /api/v1/ontologies/ont_shop/train-model
{
  "suggestion_id": "model_price_prediction",
  "exclude_features": ["rating"]  # User knows rating is noisy
}

# Mimir trains with user's preferences
```

## Implementation Priorities

### Phase 1 (Current Sprint): Foundation
1. ✅ Regression + classification working
2. ✅ Time-series analysis working
3. ⏳ **OntologyAnalyzer** - Analyze properties, suggest models
4. ⏳ **KGDataExtractor** - Query KG, convert to ML format
5. ⏳ **AutoTrainer** - Train models from ontology

### Phase 2: Intelligence
6. LLM-based goal parsing ("predict X" → identify target property)
7. Smart feature selection (semantic similarity, correlation analysis)
8. Automatic hyperparameter tuning
9. Model versioning (retrain when ontology evolves)

### Phase 3: Advanced
10. Graph features (entity relationships as features)
11. Multi-model ensembles
12. Explainable AI (why did model predict X?)
13. Active learning (request labels for uncertain predictions)

## Key Design Decisions

### 1. **When to Auto-Train?**

**Triggers:**
- User explicitly requests: `POST /ontologies/{id}/auto-train`
- After ontology creation (if data available)
- After drift detection suggests changes
- Scheduled re-training (weekly/monthly)

**NOT auto-train when:**
- Insufficient data (< 50 samples)
- No clear target properties
- User hasn't opted in

### 2. **How to Handle Ambiguity?**

**Example:** Ontology has both `price` and `cost` (both numeric).

**Solution:** Rank by confidence + ask user
```json
{
  "message": "I found multiple prediction targets. Which would you like?",
  "targets": [
    {"property": "price", "confidence": 0.95, "reason": "Most common prediction target"},
    {"property": "cost", "confidence": 0.80, "reason": "Also numeric, less data available"}
  ]
}
```

### 3. **How to Store Auto-Generated Models?**

**Database Schema Addition:**
```sql
ALTER TABLE classifier_models ADD COLUMN auto_generated BOOLEAN DEFAULT 0;
ALTER TABLE classifier_models ADD COLUMN generation_reasoning TEXT;
ALTER TABLE classifier_models ADD COLUMN user_approved BOOLEAN DEFAULT 0;
```

**Workflow:**
- Auto-generated models marked as `auto_generated = 1`
- Shown to user for approval
- Can be deleted/re-trained easily
- User can promote to "approved" status

### 4. **Privacy & Control**

**User must opt-in:**
```bash
POST /api/v1/ontologies/{id}/settings
{
  "auto_ml_enabled": true,
  "auto_monitoring_enabled": true,
  "auto_training_frequency": "weekly"
}
```

**User can always:**
- View what Mimir is doing
- Disable auto-training
- Delete auto-generated models
- Override suggestions

## Example: End-to-End Computer Shop

```
1. User uploads inventory.csv
   ↓
2. Mimir generates ontology
   - Classes: Product(name, price, stock, category, rating)
   ↓
3. OntologyAnalyzer runs
   - Detects: price (regression target)
   - Detects: category (classification target)
   - Detects: stock (time-series monitoring)
   ↓
4. User calls: POST /ontologies/ont_shop/ml-capabilities
   - Returns: "I can predict prices, classify categories, monitor stock"
   ↓
5. User calls: POST /ontologies/ont_shop/auto-train {"enable_all": true}
   ↓
6. AutoTrainer executes:
   - KGDataExtractor queries: SELECT ?product ?stock ?category ?rating ?price
   - Converts to X (features) and y (targets)
   - Trains regression model: price ← f(stock, category, rating)
   - Saves model with auto_generated=1
   - Creates monitoring job: check stock every 15 min
   - Creates rule: IF stock < 5 THEN alert (high severity)
   ↓
7. User schedules ingestion: POST /scheduler/jobs {cron: "0 9 * * *"}
   ↓
8. System runs autonomously:
   - 9 AM: Ingestion pipeline fetches new data
   - 9:15 AM: Monitoring job runs
   - If stock < 5: Alert created
   - User gets notification: "GPU RTX 4090 stock low (3 units)"
```

## Next Steps

1. Implement `OntologyAnalyzer` (analyze properties, suggest targets)
2. Implement `KGDataExtractor` (SPARQL query → ML data)
3. Implement basic `AutoTrainer` (train from ontology)
4. Add API endpoint: `POST /ontologies/{id}/ml-capabilities`
5. Add API endpoint: `POST /ontologies/{id}/auto-train`
6. Test with sample data

**This transforms Mimir from "ML toolkit" to "intelligent assistant that understands your data".**
