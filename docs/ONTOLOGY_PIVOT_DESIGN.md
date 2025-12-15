# Mimir AIP Ontology Pivot - Comprehensive Design Document

## Executive Summary

This document outlines the architecture and implementation plan for transforming Mimir AIP from a data processing pipeline system into a full-fledged **ontology-driven knowledge graph and digital twin platform**, inspired by Palantir AIP and Trustgraph AI's Ontology RAG approach.

**Key Difference from Original Prompt**: This design leverages Mimir AIP's **existing, production-ready foundation**:
- ✅ Mature plugin system with registry and lifecycle management
- ✅ REST API server with authentication, rate limiting, and monitoring
- ✅ Vector storage abstraction (Chromem) ready for extension
- ✅ Pipeline execution engine with context management
- ✅ MCP integration for LLM agent tooling
- ✅ Job scheduling and monitoring infrastructure
- ✅ Comprehensive testing and deployment frameworks

**What We're Building**: An ontology layer that sits **on top** of existing infrastructure, transforming Mimir from pipeline-centric to knowledge-centric.

---

## 1. Architectural Vision

### 1.1 Current Architecture (Mimir AIP v0.0.1)

```
┌─────────────────────────────────────────────────────────────────┐
│                       REST API Server                           │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐         │
│  │  Pipeline    │  │   Plugin     │  │  Scheduler   │         │
│  │  Management  │  │  Management  │  │  (Cron)      │         │
│  └──────────────┘  └──────────────┘  └──────────────┘         │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│                    Plugin Execution Engine                       │
│  ┌───────────┐  ┌───────────┐  ┌───────────┐  ┌───────────┐  │
│  │  Input    │  │   Data    │  │  AI/LLM   │  │  Output   │  │
│  │  Plugins  │  │Processing │  │  Models   │  │  Plugins  │  │
│  └───────────┘  └───────────┘  └───────────┘  └───────────┘  │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│                    Storage Layer (Chromem)                       │
│               Vector Embeddings + Document Store                 │
└─────────────────────────────────────────────────────────────────┘
```

### 1.2 Target Architecture (Ontology-Centric)

```
┌─────────────────────────────────────────────────────────────────┐
│                    Mimir Ontology API Layer                      │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐         │
│  │  Ontology    │  │   Agent      │  │   Digital    │         │
│  │  Management  │  │   Tooling    │  │   Twin API   │         │
│  └──────────────┘  └──────────────┘  └──────────────┘         │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│              Knowledge Graph Processing Pipeline                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  LLM-Powered Entity & Relation Extraction               │   │
│  │  • NER with ontology context                            │   │
│  │  • Relationship mapping                                 │   │
│  │  • Automatic schema evolution                           │   │
│  └─────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│              Dual Storage Architecture                           │
│  ┌────────────────────────┐  ┌────────────────────────┐        │
│  │   RDF/OWL Graph Store  │  │  Vector Store (Chromem)│        │
│  │   • Blazegraph/Jena    │  │  • Semantic embeddings │        │
│  │   • SPARQL endpoint    │  │  • Hybrid search       │        │
│  │   • Ontology schema    │  │                        │        │
│  └────────────────────────┘  └────────────────────────┘        │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│                     Existing Mimir Foundation                    │
│              (Pipelines, Plugins, Scheduler, MCP)                │
└─────────────────────────────────────────────────────────────────┘
```

---

## 2. Core Components Design

### 2.1 Ontology Management Service

**Location**: `pipelines/Ontology/`

**Purpose**: Manage ontology lifecycle, versioning, and validation

**Key Types**:
```go
// ontology_types.go
package Ontology

type OntologyFormat string
const (
    FormatTurtle   OntologyFormat = "turtle"
    FormatRDFXML   OntologyFormat = "rdf/xml"
    FormatJSONLD   OntologyFormat = "json-ld"
)

type OntologyMetadata struct {
    ID          string    `json:"id"`
    Name        string    `json:"name"`
    Version     string    `json:"version"`
    Description string    `json:"description"`
    Format      OntologyFormat `json:"format"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
    Author      string    `json:"author"`
    BaseURI     string    `json:"base_uri"`
}

type Ontology struct {
    Metadata OntologyMetadata       `json:"metadata"`
    Content  []byte                 `json:"content"`  // Raw Turtle/RDF
    Classes  []OntologyClass        `json:"classes"`
    Properties []OntologyProperty   `json:"properties"`
}

type OntologyClass struct {
    URI         string   `json:"uri"`
    Label       string   `json:"label"`
    Description string   `json:"description"`
    SubClassOf  []string `json:"subclass_of"`
    Properties  []string `json:"properties"`
}

type OntologyProperty struct {
    URI         string   `json:"uri"`
    Label       string   `json:"label"`
    PropertyType string  `json:"type"` // ObjectProperty, DataProperty
    Domain      []string `json:"domain"`
    Range       []string `json:"range"`
}
```

**Plugin Interface**:
```go
// ontology_plugin.go
type OntologyPlugin struct {
    store OntologyStore
}

// Operations:
// - upload_ontology: POST ontology file
// - get_ontology: Retrieve by ID
// - list_ontologies: List all with versions
// - validate_ontology: Validate OWL/RDF syntax
// - update_ontology: Version-controlled updates
// - query_schema: Extract classes/properties
```

**Storage**: Use existing Chromem backend + new RDF store integration

---

### 2.2 Knowledge Graph Store Integration

**Technology Decision**: **Apache Jena Fuseki** (Open-source, production-ready RDF store)

**Why Jena over Blazegraph**:
- ✅ Active maintenance (Apache project)
- ✅ Better Go client libraries available
- ✅ TDB2 storage engine optimized for modern hardware
- ✅ Full SPARQL 1.1 support
- ✅ Embedded mode possible (no separate server)
- ✅ Docker deployment ready

**Location**: `pipelines/KnowledgeGraph/`

**Key Types**:
```go
// kg_store.go
package KnowledgeGraph

type GraphStoreBackend interface {
    // Triple operations
    InsertTriples(ctx context.Context, triples []Triple) error
    QuerySPARQL(ctx context.Context, query string) (*QueryResult, error)
    DeleteTriples(ctx context.Context, filter TripleFilter) error
    
    // Graph operations
    CreateGraph(ctx context.Context, graphURI string) error
    ListGraphs(ctx context.Context) ([]string, error)
    
    // Health & stats
    Health(ctx context.Context) error
    Stats(ctx context.Context) (*GraphStats, error)
}

type Triple struct {
    Subject   string `json:"subject"`   // URI or blank node
    Predicate string `json:"predicate"` // Property URI
    Object    string `json:"object"`    // URI, literal, or blank node
    Graph     string `json:"graph,omitempty"`
}

type QueryResult struct {
    Bindings []map[string]interface{} `json:"bindings"`
    Count    int                      `json:"count"`
}

// Fuseki implementation
type FusekiBackend struct {
    baseURL    string
    httpClient *http.Client
    dataset    string
}
```

**Integration with Existing Storage**:
```go
// Extend pipelines/Storage/backend_factory.go
const (
    BackendTypeChromem  BackendType = "chromem"
    BackendTypeRDF      BackendType = "rdf"      // NEW
    BackendTypeHybrid   BackendType = "hybrid"   // NEW: RDF + Vector
)

type HybridBackend struct {
    rdfStore    *KnowledgeGraph.FusekiBackend
    vectorStore *ChromemBackend
}
```

---

### 2.3 Graph Processing Pipeline

**Location**: `pipelines/GraphProcessing/`

**Purpose**: Automated data → knowledge graph transformation

**Architecture**:
```
┌────────────────┐
│   Raw Data     │
│  (CSV/JSON/    │
│   Text/API)    │
└────────┬───────┘
         │
         ↓
┌────────────────────────────────────────┐
│   Step 1: Data Normalization           │
│   • Existing Input plugins             │
│   • Structured data parsing            │
└────────┬───────────────────────────────┘
         │
         ↓
┌────────────────────────────────────────┐
│   Step 2: Entity Extraction            │
│   • LLM-powered NER                    │
│   • Ontology-guided extraction         │
│   • Entity linking/disambiguation      │
└────────┬───────────────────────────────┘
         │
         ↓
┌────────────────────────────────────────┐
│   Step 3: Relation Extraction          │
│   • LLM-powered relationship detection │
│   • Property mapping to ontology       │
│   • Confidence scoring                 │
└────────┬───────────────────────────────┘
         │
         ↓
┌────────────────────────────────────────┐
│   Step 4: Triple Generation            │
│   • RDF triple creation                │
│   • Namespace management               │
│   • Provenance tracking                │
└────────┬───────────────────────────────┘
         │
         ↓
┌────────────────────────────────────────┐
│   Step 5: Graph Insertion              │
│   • Batch SPARQL INSERT                │
│   • Vector embedding generation        │
│   • Dual storage sync                  │
└────────────────────────────────────────┘
```

**Key Components**:

```go
// entity_extractor.go
type EntityExtractor struct {
    llmClient   LLMClient
    ontology    *Ontology.Ontology
}

func (e *EntityExtractor) Extract(ctx context.Context, text string) ([]Entity, error) {
    prompt := e.buildExtractionPrompt(text)
    response, err := e.llmClient.Complete(ctx, prompt)
    if err != nil {
        return nil, err
    }
    return e.parseEntities(response)
}

func (e *EntityExtractor) buildExtractionPrompt(text string) string {
    return fmt.Sprintf(`
You are an expert knowledge graph engineer. Given the ontology below and the input text, 
extract all entities and their relationships.

ONTOLOGY CLASSES:
%s

ONTOLOGY PROPERTIES:
%s

INPUT TEXT:
%s

OUTPUT FORMAT (JSON):
{
    "entities": [
        {
            "text": "extracted entity text",
            "type": "ontology_class_uri",
            "confidence": 0.95
        }
    ],
    "relationships": [
        {
            "subject": "entity1_text",
            "predicate": "property_uri",
            "object": "entity2_text",
            "confidence": 0.90
        }
    ]
}
`, e.formatClasses(), e.formatProperties(), text)
}
```

```go
// triple_builder.go
type TripleBuilder struct {
    baseURI   string
    namespace map[string]string
}

func (tb *TripleBuilder) BuildTriples(entities []Entity, relations []Relation) ([]Triple, error) {
    var triples []Triple
    
    // Create entity triples
    for _, entity := range entities {
        entityURI := tb.createEntityURI(entity)
        
        // rdf:type triple
        triples = append(triples, Triple{
            Subject:   entityURI,
            Predicate: "rdf:type",
            Object:    entity.Type,
        })
        
        // rdfs:label triple
        triples = append(triples, Triple{
            Subject:   entityURI,
            Predicate: "rdfs:label",
            Object:    fmt.Sprintf(`"%s"`, entity.Text),
        })
    }
    
    // Create relationship triples
    for _, rel := range relations {
        subjectURI := tb.findEntityURI(rel.Subject, entities)
        objectURI := tb.findEntityURI(rel.Object, entities)
        
        triples = append(triples, Triple{
            Subject:   subjectURI,
            Predicate: rel.Predicate,
            Object:    objectURI,
        })
    }
    
    return triples, nil
}
```

---

### 2.4 Agent Tooling API (NL → SPARQL)

**Location**: Extend existing `mcp_server.go`

**New MCP Tools**:

1. **`ontology.query_graph`**
   ```json
   {
     "name": "ontology.query_graph",
     "description": "Query the knowledge graph using natural language",
     "input_schema": {
       "type": "object",
       "properties": {
         "question": {
           "type": "string",
           "description": "Natural language question"
         },
         "graph": {
           "type": "string",
           "description": "Graph name (optional)"
         },
         "limit": {
           "type": "integer",
           "default": 10
         }
       }
     }
   }
   ```

2. **`ontology.classify_data`**
   ```json
   {
     "name": "ontology.classify_data",
     "description": "Classify data against ontology using ML model",
     "input_schema": {
       "type": "object",
       "properties": {
         "data": {
           "type": "string",
           "description": "Text or structured data to classify"
         },
         "return_confidence": {
           "type": "boolean",
           "default": true
         }
       }
     }
   }
   ```

3. **`ontology.simulate_event`**
   ```json
   {
     "name": "ontology.simulate_event",
     "description": "Digital twin simulation - predict impact of events",
     "input_schema": {
       "type": "object",
       "properties": {
         "event_description": {
           "type": "string",
           "description": "Natural language event description"
         },
         "entities": {
           "type": "array",
           "description": "Affected entity URIs (optional)"
         }
       }
     }
   }
   ```

**Implementation**:
```go
// nl_to_sparql.go
type NLToSPARQLConverter struct {
    llmClient LLMClient
    ontology  *Ontology.Ontology
    graphStore KnowledgeGraph.GraphStoreBackend
}

func (n *NLToSPARQLConverter) Convert(ctx context.Context, question string) (string, error) {
    prompt := fmt.Sprintf(`
You are a SPARQL query expert. Convert the natural language question to a precise SPARQL query.

ONTOLOGY SCHEMA:
%s

QUESTION:
%s

OUTPUT: Only the SPARQL query, nothing else. Use standard prefixes.
`, n.formatOntologySchema(), question)

    response, err := n.llmClient.Complete(ctx, prompt)
    if err != nil {
        return "", err
    }
    
    // Extract SPARQL query from response
    query := n.extractSPARQL(response)
    
    // Validate query syntax
    if err := n.validateSPARQL(query); err != nil {
        return "", fmt.Errorf("invalid SPARQL: %w", err)
    }
    
    return query, nil
}

func (n *NLToSPARQLConverter) validateSPARQL(query string) error {
    // Security: Check for dangerous operations
    forbidden := []string{"DROP", "DELETE", "INSERT", "CLEAR", "LOAD"}
    queryUpper := strings.ToUpper(query)
    for _, op := range forbidden {
        if strings.Contains(queryUpper, op) {
            return fmt.Errorf("forbidden operation: %s", op)
        }
    }
    
    // Check depth and complexity
    if strings.Count(query, "{") > 5 {
        return fmt.Errorf("query too complex (max 5 nested blocks)")
    }
    
    return nil
}
```

---

### 2.5 ML Classifier Integration (ONNX)

**Location**: `pipelines/ML/`

**Architecture**:
```
┌──────────────────┐
│  Training Data   │ ← User-labeled examples
│  Collection      │
└────────┬─────────┘
         │
         ↓
┌──────────────────────────┐
│  Python Training Service │ ← Separate microservice
│  • scikit-learn          │
│  • Transformers          │
│  • ONNX export           │
└────────┬─────────────────┘
         │
         ↓ (ONNX file)
┌──────────────────────────┐
│  Go Inference Service    │
│  • ONNX Runtime binding  │
│  • Real-time prediction  │
│  • MCP tool integration  │
└──────────────────────────┘
```

**Key Types**:
```go
// classifier.go
type OntologyClassifier struct {
    model      *onnxruntime.Session
    tokenizer  Tokenizer
    classes    []string
}

func (c *OntologyClassifier) Classify(ctx context.Context, input string) (*ClassificationResult, error) {
    // Tokenize input
    tokens := c.tokenizer.Encode(input)
    
    // Run ONNX inference
    outputs, err := c.model.Run(tokens)
    if err != nil {
        return nil, err
    }
    
    // Parse results
    return c.parseClassification(outputs), nil
}

type ClassificationResult struct {
    PredictedClass string  `json:"predicted_class"`
    Confidence     float32 `json:"confidence"`
    AllScores      map[string]float32 `json:"all_scores"`
}
```

**Training API** (Python microservice):
```python
# training_service.py
from flask import Flask, request, jsonify
from sklearn.ensemble import RandomForestClassifier
from transformers import AutoTokenizer, AutoModel
import onnx
import onnxruntime as ort
import numpy as np

app = Flask(__name__)

@app.route('/train', methods=['POST'])
def train_model():
    """
    Expected payload:
    {
        "training_data": [
            {"text": "...", "label": "ontology_class_uri"},
            ...
        ],
        "ontology": {...},
        "model_type": "random_forest" | "transformer"
    }
    """
    data = request.json
    
    # Extract features using transformer embeddings
    model = AutoModel.from_pretrained('sentence-transformers/all-MiniLM-L6-v2')
    tokenizer = AutoTokenizer.from_pretrained('sentence-transformers/all-MiniLM-L6-v2')
    
    X = []
    y = []
    for item in data['training_data']:
        # Get embeddings
        tokens = tokenizer(item['text'], return_tensors='pt', truncation=True, padding=True)
        embeddings = model(**tokens).last_hidden_state.mean(dim=1).detach().numpy()
        X.append(embeddings[0])
        y.append(item['label'])
    
    # Train classifier
    clf = RandomForestClassifier(n_estimators=100, random_state=42)
    clf.fit(X, y)
    
    # Export to ONNX
    from skl2onnx import convert_sklearn
    from skl2onnx.common.data_types import FloatTensorType
    
    initial_type = [('float_input', FloatTensorType([None, X[0].shape[0]]))]
    onnx_model = convert_sklearn(clf, initial_types=initial_type)
    
    # Save model
    model_path = f'/models/{data["model_id"]}.onnx'
    with open(model_path, 'wb') as f:
        f.write(onnx_model.SerializeToString())
    
    return jsonify({
        'model_path': model_path,
        'accuracy': evaluate_model(clf, X, y),
        'classes': list(set(y))
    })
```

---

## 3. Implementation Phases

### Phase 1: Foundation (Weeks 1-2)
**Goal**: Establish ontology management and RDF storage

**Tasks**:
1. Create `pipelines/Ontology/` module
   - Ontology types and validation
   - OWL/Turtle parser integration (use `github.com/knakk/rdf`)
   - Ontology plugin implementation
2. Integrate Apache Jena Fuseki
   - Docker Compose configuration
   - Go client implementation (`pipelines/KnowledgeGraph/fuseki_backend.go`)
   - SPARQL query execution
3. REST API endpoints
   - `POST /api/v1/ontology/upload`
   - `GET /api/v1/ontology/{id}`
   - `GET /api/v1/ontology`
   - `PUT /api/v1/ontology/{id}`
4. Testing
   - Unit tests for ontology validation
   - Integration tests with Fuseki
   - Example ontologies (Dublin Core, FOAF, custom)

**Deliverable**: Working ontology management service with RDF storage

---

### Phase 2: Knowledge Graph Pipeline (Weeks 3-4)
**Goal**: Automated data → graph transformation

**Tasks**:
1. Entity extraction pipeline
   - LLM integration (reuse existing `pipelines/AI/` OpenAI plugin)
   - Ontology-guided prompt engineering
   - Entity extraction plugin (`pipelines/GraphProcessing/entity_extractor_plugin.go`)
2. Relation extraction
   - Relationship detection LLM prompts
   - Property mapping to ontology
   - Relation extraction plugin
3. Triple builder
   - RDF triple generation
   - Namespace management
   - URI generation strategies
4. Batch insertion
   - SPARQL INSERT optimization
   - Transaction handling
   - Error recovery
5. Example pipelines
   - CSV → Knowledge Graph
   - API data → Knowledge Graph
   - Text documents → Knowledge Graph

**Deliverable**: End-to-end data ingestion with automatic graph population

---

### Phase 3: Agent Tooling & Query (Weeks 5-6)
**Goal**: LLM-powered knowledge graph interaction

**Tasks**:
1. Natural language → SPARQL converter
   - Prompt engineering for SPARQL generation
   - Query validation and sanitization
   - Result formatting
2. MCP tool integration
   - `ontology.query_graph` tool
   - `ontology.get_entities` tool
   - `ontology.get_relationships` tool
3. SPARQL query sandboxing
   - Query complexity limits
   - Timeout enforcement
   - Forbidden operation detection
4. Hybrid search
   - Combine RDF graph queries with vector similarity
   - Semantic + structural search
5. Query result caching
   - Leverage existing performance infrastructure
   - SPARQL query result cache

**Deliverable**: LLM agents can query knowledge graph in natural language

---

### Phase 4: ML Classifier & Automation (Weeks 7-8)
**Goal**: Intelligent data classification

**Tasks**:
1. Python training service
   - Flask/FastAPI microservice
   - Model training endpoint
   - ONNX export
2. Go inference integration
   - ONNX Runtime Go bindings
   - Classifier plugin (`pipelines/ML/classifier_plugin.go`)
   - MCP tool for classification
3. Training data collection UI
   - Simple labeling interface (can use existing frontend)
   - Active learning suggestions
4. Model versioning
   - Model registry
   - A/B testing support
   - Performance tracking

**Deliverable**: Automated classification of new data against ontology

---

### Phase 5: Digital Twin & Simulation (Weeks 9-10)
**Goal**: "What-if" analysis and predictive modeling

**Tasks**:
1. Graph analytics
   - Path finding (shortest path, all paths)
   - Centrality measures
   - Community detection
2. Impact analysis
   - Dependency graph traversal
   - Cascading effect simulation
3. Simulation engine
   - Event impact prediction
   - LLM-powered narrative generation
   - MCP tool: `ontology.simulate_event`
4. Temporal reasoning
   - Time-aware queries
   - Historical state reconstruction
   - Trend analysis

**Deliverable**: Agents can perform "what-if" analysis on knowledge graph

---

### Phase 6: Production Readiness (Weeks 11-12)
**Goal**: Deployment, monitoring, documentation

**Tasks**:
1. Deployment automation
   - Unified Docker Compose with Fuseki + Mimir
   - Kubernetes manifests (optional)
   - Environment configuration
2. Monitoring & observability
   - Graph size metrics
   - Query performance tracking
   - Pipeline execution monitoring
3. Documentation
   - Ontology development guide
   - Data modeling best practices
   - API documentation
   - Tutorial examples
4. Performance optimization
   - Query optimization
   - Batch processing tuning
   - Caching strategies

**Deliverable**: Production-ready ontology platform

---

## 4. Technology Stack

### Core Technologies
| Component | Technology | Rationale |
|-----------|-----------|-----------|
| **Primary Language** | Go 1.23+ | Existing Mimir codebase |
| **RDF Store** | Apache Jena Fuseki | Open-source, active maintenance, TDB2 performance |
| **Vector Store** | Chromem (existing) | Already integrated, local-first |
| **Ontology Standard** | OWL 2 + RDF/Turtle | W3C standard, wide tooling support |
| **Query Language** | SPARQL 1.1 | Standard for RDF querying |
| **ML Training** | Python (scikit-learn, Transformers) | Best ML ecosystem |
| **ML Inference** | ONNX Runtime (Go bindings) | Cross-platform, performant |
| **LLM Integration** | OpenAI API (existing plugin) | Already working in Mimir |

### Key Go Libraries (to add)
```bash
go get github.com/knakk/rdf          # RDF/Turtle parsing
go get github.com/piprate/json-gold  # JSON-LD processing
go get github.com/yhat/scrape        # HTML parsing for web data
```

### Docker Services
```yaml
services:
  mimir-aip:
    build: .
    ports:
      - "8080:8080"
    depends_on:
      - fuseki
      - ml-service
  
  fuseki:
    image: stain/jena-fuseki:latest
    ports:
      - "3030:3030"
    volumes:
      - fuseki-data:/fuseki
    environment:
      - ADMIN_PASSWORD=admin
  
  ml-service:
    build: ./ml-service
    ports:
      - "5000:5000"
    volumes:
      - ./models:/models
```

---

## 5. Data Flow Examples

### Example 1: CSV to Knowledge Graph

**Input CSV** (`servers.csv`):
```csv
hostname,ip_address,os,datacenter,owner
db-prod-01,10.0.1.50,Ubuntu 22.04,us-east-1,platform-team
app-prod-02,10.0.1.51,RHEL 8,us-west-2,backend-team
```

**Pipeline** (`csv_to_graph.yaml`):
```yaml
name: "CSV to Knowledge Graph"
steps:
  - name: "Load CSV"
    plugin: "Input.csv"
    config:
      file_path: "servers.csv"
    output: "raw_data"
  
  - name: "Extract Entities"
    plugin: "GraphProcessing.entity_extractor"
    config:
      ontology_id: "infrastructure-ontology-v1"
      source_field: "raw_data"
    output: "entities"
  
  - name: "Build Triples"
    plugin: "GraphProcessing.triple_builder"
    config:
      entities: "entities"
      base_uri: "http://example.org/infra/"
    output: "triples"
  
  - name: "Insert to Graph"
    plugin: "KnowledgeGraph.insert"
    config:
      triples: "triples"
      graph: "infrastructure"
    output: "result"
```

**Generated RDF** (Turtle):
```turtle
@prefix infra: <http://example.org/infra/> .
@prefix rdf: <http://www.w3.org/1999/02/22-rdf-syntax-ns#> .
@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .

infra:server_db-prod-01
    rdf:type infra:Server ;
    rdfs:label "db-prod-01" ;
    infra:hasIPAddress "10.0.1.50" ;
    infra:hasOS "Ubuntu 22.04" ;
    infra:locatedIn infra:datacenter_us-east-1 ;
    infra:ownedBy infra:team_platform-team .

infra:datacenter_us-east-1
    rdf:type infra:Datacenter ;
    rdfs:label "us-east-1" .

infra:team_platform-team
    rdf:type infra:Team ;
    rdfs:label "platform-team" .
```

---

### Example 2: Natural Language Query

**User Query** (via MCP tool):
```
"Show me all Ubuntu servers in us-east-1 that are owned by the platform team"
```

**LLM-Generated SPARQL**:
```sparql
PREFIX infra: <http://example.org/infra/>
PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>

SELECT ?server ?label ?ip WHERE {
    ?server rdf:type infra:Server ;
            rdfs:label ?label ;
            infra:hasIPAddress ?ip ;
            infra:hasOS ?os ;
            infra:locatedIn ?datacenter ;
            infra:ownedBy ?owner .
    
    ?datacenter rdfs:label "us-east-1" .
    ?owner rdfs:label "platform-team" .
    
    FILTER(CONTAINS(LCASE(?os), "ubuntu"))
}
```

**Result**:
```json
{
  "bindings": [
    {
      "server": "http://example.org/infra/server_db-prod-01",
      "label": "db-prod-01",
      "ip": "10.0.1.50"
    }
  ]
}
```

---

### Example 3: Digital Twin Simulation

**Simulation Request** (MCP tool):
```json
{
  "tool": "ontology.simulate_event",
  "arguments": {
    "event_description": "What if server db-prod-01 goes down?"
  }
}
```

**Processing**:
1. LLM extracts entity: `db-prod-01`
2. SPARQL query finds dependencies:
```sparql
SELECT ?dependent WHERE {
    infra:server_db-prod-01 infra:dependsOn ?dependent .
}
UNION
{
    ?dependent infra:dependsOn infra:server_db-prod-01 .
}
```
3. LLM generates impact narrative with dependency context

**Response**:
```json
{
  "event": "Server db-prod-01 failure",
  "affected_entities": [
    "app-prod-02",
    "load-balancer-01",
    "auth-service"
  ],
  "impact_assessment": {
    "severity": "critical",
    "affected_users": "~50,000",
    "services_down": ["authentication", "user-api"],
    "estimated_recovery_time": "30 minutes"
  },
  "narrative": "If db-prod-01 fails, the authentication database becomes unavailable, causing cascading failures in app-prod-02 and auth-service. This affects approximately 50,000 active users. The load balancer will detect the failure within 30 seconds, but recovery requires database failover to db-prod-01-replica, estimated at 30 minutes."
}
```

---

## 6. Integration Points with Existing Mimir

### 6.1 Plugin System Integration
- ✅ **Reuse**: `BasePlugin` interface, `PluginRegistry`, `PluginContext`
- ✅ **Extend**: Add new plugin types:
  - `Ontology` (ontology management)
  - `GraphProcessing` (entity/relation extraction)
  - `KnowledgeGraph` (SPARQL operations)
  - `ML` (classification)

### 6.2 Storage System Integration
- ✅ **Reuse**: `StorageBackend` interface, `BackendFactory`
- ✅ **Extend**: Add `RDFBackend` and `HybridBackend` implementations
- ✅ **Benefit**: Existing storage abstraction makes adding RDF seamless

### 6.3 REST API Integration
- ✅ **Reuse**: `server.go`, `routes.go`, authentication middleware
- ✅ **Extend**: Add new endpoint groups:
  - `/api/v1/ontology/*`
  - `/api/v1/knowledge-graph/*`
  - `/api/v1/ml/classify`
  - `/api/v1/simulation/*`

### 6.4 MCP Server Integration
- ✅ **Reuse**: Existing MCP tool registration system
- ✅ **Extend**: Register new ontology-focused tools
- ✅ **Benefit**: LLM agents immediately gain access to knowledge graph

### 6.5 Scheduler Integration
- ✅ **Reuse**: Existing cron scheduler for periodic tasks
- ✅ **Use Cases**:
  - Scheduled data ingestion → graph updates
  - Periodic ontology evolution checks
  - Model retraining schedules
  - Graph analytics jobs

### 6.6 Monitoring Integration
- ✅ **Reuse**: Performance metrics, job monitoring, health checks
- ✅ **Extend**: Add metrics:
  - Graph size (triples count)
  - Query performance (SPARQL latency)
  - Entity extraction accuracy
  - Classification model performance

---

## 7. Security Considerations

### 7.1 SPARQL Injection Prevention
```go
// sparql_validator.go
type SPARQLValidator struct {
    maxDepth      int
    maxResults    int
    forbiddenOps  []string
}

func (v *SPARQLValidator) Validate(query string) error {
    // Parse query AST
    ast, err := parseSPARQL(query)
    if err != nil {
        return err
    }
    
    // Check for forbidden operations
    if ast.HasMutation() {
        return errors.New("mutation operations not allowed")
    }
    
    // Check depth
    if ast.Depth() > v.maxDepth {
        return errors.New("query too complex")
    }
    
    // Enforce result limit
    if ast.Limit == 0 || ast.Limit > v.maxResults {
        ast.Limit = v.maxResults
    }
    
    return nil
}
```

### 7.2 Ontology Validation
- **Schema validation**: Ensure uploaded ontologies are valid OWL/RDF
- **Namespace isolation**: Prevent namespace collisions
- **Access control**: Role-based ontology management

### 7.3 LLM Prompt Injection Defense
- **Output validation**: Parse LLM responses strictly
- **Fallback mechanisms**: Default to safe queries if LLM fails
- **Rate limiting**: Prevent abuse of LLM-powered endpoints

---

## 8. Testing Strategy

### 8.1 Unit Tests
```bash
# Ontology management
go test ./pipelines/Ontology/...

# Knowledge graph operations
go test ./pipelines/KnowledgeGraph/...

# Graph processing
go test ./pipelines/GraphProcessing/...

# ML classifier
go test ./pipelines/ML/...
```

### 8.2 Integration Tests
```go
// knowledge_graph_integration_test.go
func TestEndToEndKnowledgeGraphPipeline(t *testing.T) {
    // 1. Upload ontology
    ontology := loadTestOntology("test_ontology.ttl")
    ontologyID := uploadOntology(ontology)
    
    // 2. Ingest data
    data := loadTestData("test_data.csv")
    pipelineResult := runPipeline("csv_to_graph.yaml", data)
    assert.NoError(t, pipelineResult.Error)
    
    // 3. Query graph
    query := "SELECT * WHERE { ?s ?p ?o } LIMIT 10"
    results := queryGraph(query)
    assert.Greater(t, len(results), 0)
    
    // 4. NL query
    nlQuery := "Show me all entities of type Person"
    nlResults := queryGraphNL(nlQuery)
    assert.Greater(t, len(nlResults), 0)
}
```

### 8.3 Performance Tests
```go
// Benchmark SPARQL query performance
func BenchmarkSPARQLQuery(b *testing.B) {
    // Pre-populate graph with 100k triples
    populateGraph(100000)
    
    query := `SELECT ?s ?label WHERE {
        ?s rdf:type infra:Server ;
           rdfs:label ?label .
    }`
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        queryGraph(query)
    }
}
```

---

## 9. Documentation Plan

### 9.1 User Documentation
- **Ontology Development Guide**: How to create domain ontologies
- **Data Modeling Best Practices**: Entity/relationship modeling
- **Pipeline Examples**: 10+ example pipelines
- **API Reference**: Complete endpoint documentation
- **MCP Tool Guide**: Using ontology tools with LLM agents

### 9.2 Developer Documentation
- **Architecture Overview**: System design and components
- **Plugin Development**: Creating custom ontology-aware plugins
- **Backend Integration**: Adding new graph store backends
- **Contributing Guide**: Development workflow

### 9.3 Tutorial Series
1. **Getting Started**: Upload first ontology, ingest CSV data
2. **Advanced Queries**: Complex SPARQL patterns
3. **Custom Extractors**: Building domain-specific entity extractors
4. **Digital Twin**: Creating simulations for your domain
5. **Production Deployment**: Scaling and monitoring

---

## 10. Success Criteria

### Functional Requirements
- ✅ Upload and manage ontologies (Turtle, RDF/XML, JSON-LD)
- ✅ Automatically extract entities and relations from data
- ✅ Store knowledge in RDF graph (SPARQL-queryable)
- ✅ Natural language → SPARQL translation
- ✅ Hybrid search (graph + vector)
- ✅ ML classification of data against ontology
- ✅ Digital twin simulation capabilities
- ✅ MCP tool integration for LLM agents

### Performance Requirements
- Handle ontologies with 1000+ classes
- Ingest 10,000+ entities per minute
- SPARQL query response < 200ms (p95)
- Support graphs with 1M+ triples
- NL → SPARQL conversion < 3s

### Quality Requirements
- 90%+ test coverage for new modules
- All endpoints documented in API reference
- No breaking changes to existing Mimir APIs
- Security audit passed (SPARQL injection, prompt injection)

---

## 11. Migration Path for Existing Users

Mimir AIP users can adopt ontology features **incrementally**:

### Stage 1: Optional Enhancement (No Breaking Changes)
- Existing pipelines continue to work without modification
- Ontology features available as new plugins
- No changes to existing API endpoints

### Stage 2: Gradual Adoption
- Users can add ontology-aware steps to existing pipelines
- Example: Add entity extraction after existing data processing
- Backward compatible

### Stage 3: Full Migration (Optional)
- Users who want full ontology-centric approach can refactor pipelines
- Migration tools provided (CSV schema → ontology generator)
- Documentation and examples provided

---

## 12. Future Enhancements (Post-v1)

### Phase 7+: Advanced Features
- **Graph Visualization**: Web-based graph explorer UI
- **Federated Queries**: Query multiple knowledge graphs
- **Graph Evolution**: Schema evolution tools, migration support
- **Advanced Analytics**: Graph neural networks, link prediction
- **Provenance Tracking**: Full lineage tracking for all triples
- **Collaborative Ontology Editing**: Multi-user ontology management
- **Graph Versioning**: Time-travel queries, historical snapshots

---

## 13. Conclusion

This design document provides a **comprehensive, pragmatic roadmap** for transforming Mimir AIP into an ontology-driven knowledge graph platform. Key differentiators:

1. **Builds on Existing Foundation**: Leverages 90% of current Mimir infrastructure
2. **Incremental Adoption**: No breaking changes, users adopt at their own pace
3. **Production-Ready Stack**: Apache Jena, ONNX, proven technologies
4. **LLM-Native**: MCP integration makes knowledge accessible to AI agents
5. **Open Standards**: OWL, RDF, SPARQL - not proprietary
6. **Go-First**: Core in Go for performance, Python only for ML training

**Estimated Timeline**: 12 weeks for full implementation with 1-2 developers

**Next Steps**:
1. Review and approve this design
2. Set up development environment (Fuseki, ML service)
3. Begin Phase 1 implementation
4. Iterate based on feedback

---

**Document Version**: 1.0  
**Author**: Mimir AIP Development Team  
**Date**: 2025-01-15  
**Status**: Ready for Implementation
