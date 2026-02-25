# Digital Twin

The digital twin will be a dynamic, ontology-backed representation of the project. It will be built using the ontology as a blueprint to determine what entities, attributes and relationships to include. The digital twin will not store a complete copy of all the data, rather it will reference the data stored in the storage object, and only store any changes or modifications to the data. The digital twin will be a location for performing analysis and predictive modelling of the data, it will reference the ML models for making predictions based on the data(not just the ingested data but through the digital twin you can 'alter' project data by creating hypothetical scenarios, e.g. what if I change this attribute of this entity to be higher/lower, how does that impact the predictions etc.). The digital twin will also be a location for querying the data using SPARQL queries. Withing the digital twin the user can setup actions(e.g. for the ingested data, if the ml model detects this value or a value above/below a certain threshold, trigger a export pipeline which could for example generate a report or send a push notification etc.).

## Architecture

### Data Referencing Model
The digital twin uses a hybrid storage approach to minimize overhead while enabling scenario simulation:
- **Reference Layer**: Points to original CIR-formatted data stored in the storage layer
- **Delta Layer**: Stores only modifications, additions, or hypothetical changes
- **Computed Layer**: Caches frequently accessed computed values and ML model predictions

### Integration Points
The digital twin integrates with core system components:
- **Storage Layer**: Retrieves CIR data via storage plugins for entity population
- **ML Models**: Leverages trained models for predictions and anomaly detection
- **Pipelines**: Triggers output pipelines based on conditions or schedules
- **Ontology**: Uses OWL definitions for semantic query validation and result interpretation

## Core Capabilities

### SPARQL Querying
The digital twin supports complex SPARQL queries for data analysis:

```sparql
PREFIX : <http://example.org/mimir#>
SELECT ?user ?product ?price
WHERE {
  ?user a :User .
  ?product a :Product .
  ?purchase :purchaser ?user ;
            :item ?product .
  ?product :price ?price .
  FILTER(?price > 1000)
}
ORDER BY DESC(?price)
```

### Predictive Modeling Integration
ML models are bound to ontology entities for real-time predictions:
- **Point Predictions**: Single entity predictions
- **Batch Predictions**: Multiple entity predictions
- **Anomaly Detection**: Outlier identification
- **Trend Analysis**: Forecasting based on historical patterns

### Scenario Simulation (What-If Analysis)
Users can create hypothetical scenarios by modifying entity attributes:

```json
{
  "scenario": {
    "name": "Price Increase Impact",
    "modifications": [
      {
        "entity": "Product",
        "id": "widget-123",
        "attribute": "price",
        "original_value": 99.99,
        "new_value": 119.99
      }
    ],
    "predictions": [
      {
        "model": "sales_forecast",
        "impact": "Expected 15% decrease in monthly sales volume"
      }
    ]
  }
}
```

## Digital Twin API

### Core Interface
The digital twin exposes a REST API for programmatic access:

```go
type DigitalTwinAPI interface {
    // Query operations
    Query(sparqlQuery string) (*QueryResult, error)
    GetEntity(entityType, id string) (*Entity, error)
    GetRelatedEntities(entityId, relationship string) ([]*Entity, error)

    // Scenario operations
    CreateScenario(modifications []*ScenarioModification) (*Scenario, error)
    UpdateEntity(entityId string, updates map[string]interface{}) (*Entity, error)

    // Prediction operations
    Predict(modelName string, input *PredictionInput) (*Prediction, error)
    BatchPredict(modelName string, inputs []*PredictionInput) ([]*Prediction, error)

    // Action management
    CreateAction(action *ActionDefinition) (*Action, error)
    ListActions() ([]*Action, error)
    TriggerAction(actionId string) (*ActionResult, error)
}
```

### Scenario Management
Scenarios enable exploration of alternative realities:

```go
type Scenario struct {
    ID          string                  `json:"id"`
    Name        string                  `json:"name"`
    Description *string                 `json:"description,omitempty"`
    BaseState   string                  `json:"base_state"` // "current" or "historical"
    Modifications []*ScenarioModification `json:"modifications"`
    Predictions []*ScenarioPrediction   `json:"predictions"`
    Created     time.Time               `json:"created"`
    Status      string                  `json:"status"` // "active" or "archived"
}

type ScenarioModification struct {
    EntityType   string      `json:"entity_type"`
    EntityId     string      `json:"entity_id"`
    Attribute    string      `json:"attribute"`
    OriginalValue interface{} `json:"original_value"`
    NewValue     interface{} `json:"new_value"`
    Rationale    *string     `json:"rationale,omitempty"`
}
```

## Action System

### Simple Conditional Triggers
Actions in the digital twin follow a simple pattern: when a condition is met, trigger an output pipeline with specified parameters. This keeps the system focused on data-driven automation without complex logic.

### Action Definition Schema
```yaml
name: high_value_purchase_alert
condition:
  model: purchase_value_predictor
  operator: gt
  threshold: 5000
trigger:
  pipeline: executive_notification
  parameters:
    recipient: "executives@company.com"
    priority: high
enabled: true
```

## Integration with ML Models

### Model Binding
ML models are bound to ontology entities through semantic mapping:
- **Entity-Level Models**: Predict attributes for specific entity types
- **Relationship Models**: Predict connections between entities
- **Aggregate Models**: Provide system-level insights and KPIs

### Prediction Caching
To optimize performance, predictions are cached with:
- **Time-Based Expiration**: Automatic refresh intervals
- **Change-Based Invalidation**: Cache updates when underlying data changes
- **Scenario Isolation**: Separate caching for hypothetical scenarios

## Query Optimization

### Indexing Strategy
The digital twin uses ontology-aware indexing:
- **Entity Indexes**: Fast lookup by entity type and ID
- **Attribute Indexes**: Range and equality queries on attributes
- **Relationship Indexes**: Efficient traversal of entity relationships

### Query Planning
SPARQL queries are optimized using:
- **Ontology Reasoning**: Leverage OWL axioms for query expansion
- **Join Optimization**: Minimize data retrieval for complex queries
- **Predicate Pushdown**: Filter data at the storage layer when possible

## Data Synchronization

### Change Detection
The digital twin monitors for data changes:
- **Storage Layer Hooks**: Notifications when CIR data is modified
- **Ontology Updates**: Automatic restructuring when ontology changes
- **Model Retraining Triggers**: Alerts when data patterns shift significantly

### Consistency Management
Ensures data consistency across scenarios:
- **Base State Locking**: Prevents conflicts during scenario creation
- **Delta Merging**: Safely combines multiple modifications
- **Conflict Resolution**: Handles concurrent scenario modifications

## Security and Access Control

### Ontology-Based Permissions
Access control leverages ontology structure:
- **Entity-Level Permissions**: Control access to specific entity types
- **Attribute-Level Permissions**: Restrict visibility of sensitive attributes
- **Relationship Permissions**: Control traversal of entity connections

### Scenario Isolation
Scenarios are sandboxed to prevent unauthorized access:
- **User Ownership**: Scenarios belong to their creators
- **Sharing Permissions**: Controlled sharing with other users
- **Audit Logging**: Track all scenario modifications and queries

## Performance Considerations

### Caching Layers
Multiple caching strategies optimize performance:
- **Query Result Cache**: Cache frequent SPARQL query results
- **Entity Cache**: Cache frequently accessed entities
- **Prediction Cache**: Cache ML model predictions

### Scalability
The digital twin scales through:
- **Horizontal Partitioning**: Distribute entities across multiple instances
- **Read Replicas**: Scale query performance with read-only copies
- **Async Processing**: Background processing for heavy computations

## Example Usage

### Basic SPARQL Query
```sparql
PREFIX : <http://example.org/mimir#>
SELECT ?product ?name ?predictedSales
WHERE {
  ?product a :Product ;
           :name ?name .
  ?prediction :targetEntity ?product ;
              :model :salesPredictor ;
              :value ?predictedSales .
}
```

### Scenario Creation
```json
{
  "name": "Holiday Price Promotion",
  "modifications": [
    {
      "entityType": "Product",
      "entityId": "seasonal-item-1",
      "attribute": "price",
      "newValue": 79.99,
      "rationale": "20% discount for holiday season"
    }
  ]
}
```

### Action Setup
```yaml
name: high_value_alert
condition:
  model: sales_predictor
  operator: gt
  threshold: 10000
trigger:
  pipeline: executive_report
  parameters:
    alert_type: "high_value_sale"
    priority: "high"
```

