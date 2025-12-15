# Mimir AIP Ontology Pivot - Revised Design (Single Container)

## Executive Summary

This revised design addresses key deployment and integration concerns:

1. **Single Unified Container** - Embedded Jena TDB2 (no separate services)
2. **Agent Tooling as Plugins** - Ontology operations are first-class plugins
3. **Frontend Integration** - Full UI for ontology management, visualization, queries
4. **Persistence First** - Builds on SQLite persistence PR foundation
5. **Hybrid Extraction** - Deterministic for structured data, LLM for unstructured

---

## 1. Architecture Changes from Original Design

### Original Design Issues
❌ Separate Fuseki container (complexity)
❌ Agent tools outside plugin system (inconsistent)
❌ No frontend consideration
❌ LLM-only extraction (expensive, slow for structured data)

### Revised Architecture
✅ **Single container** with embedded Jena TDB2
✅ **Ontology plugins** following existing `BasePlugin` interface
✅ **Frontend pages** for ontology management
✅ **Hybrid extraction** - deterministic + LLM where needed
✅ **Shared persistence** - SQLite for metadata, TDB2 for triples

---

## 2. Single Container Architecture

### Container Structure
```
mimir-aip-unified (Single Container: ~300MB)
├── Go Backend (Port 8080)
│   ├── REST API
│   ├── Plugin System (including ontology plugins)
│   ├── MCP Server
│   └── Embedded Jena TDB2 (via JNI or HTTP wrapper)
├── Next.js Frontend (Port 3000, proxied via /ui/*)
└── Persistent Storage
    ├── /app/data/sqlite.db (Metadata: jobs, pipelines, ontologies)
    ├── /app/data/tdb2/ (RDF triple store)
    └── /app/data/chromem/ (Vector embeddings)
```

### Why Embedded Jena TDB2?

**Option 1: Apache Jena TDB2 Java Library** (Recommended)
```
Pros:
✅ Native Java library, battle-tested
✅ Excellent performance (millions of triples)
✅ No network overhead
✅ Single process, single container
✅ File-based persistence (/app/data/tdb2/)

Cons:
❌ Requires JVM in container (adds ~50-80MB)
❌ Go ↔ Java bridge needed

Integration: Use `github.com/jnigi` or spawn Java subprocess
```

**Option 2: HDT (Header Dictionary Triples)** (Alternative)
```
Pros:
✅ Pure file format, very compact
✅ Go library available: github.com/knakk/rdf
✅ Excellent for read-heavy workloads
✅ No JVM needed

Cons:
❌ Read-optimized (slow writes)
❌ No SPARQL engine (need to implement)
❌ Less mature than Jena
```

**Decision: Use Jena TDB2 with Java subprocess wrapper**

### Updated Docker Structure

**Dockerfile.unified** (updated):
```dockerfile
# Multi-stage build
FROM node:20-alpine AS frontend-builder
WORKDIR /frontend
COPY mimir-aip-frontend/package*.json ./
RUN npm ci
COPY mimir-aip-frontend/ ./
RUN npm run build

FROM golang:1.23-alpine AS backend-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
RUN CGO_ENABLED=0 go build -o mimir-aip-server .

# Final unified image
FROM eclipse-temurin:21-jre-alpine
# Use JRE base for Jena TDB2 support

# Install runtime dependencies
RUN apk add --no-cache ca-certificates

WORKDIR /app

# Copy Jena TDB2 library
COPY --from=jena-fuseki:latest /jena /opt/jena

# Copy backend binary
COPY --from=backend-builder /app/mimir-aip-server .
COPY --from=backend-builder /app/pipelines ./pipelines

# Copy frontend build
COPY --from=frontend-builder /frontend/.next/standalone ./frontend
COPY --from=frontend-builder /frontend/.next/static ./frontend/.next/static
COPY --from=frontend-builder /frontend/public ./frontend/public

# Create data directories
RUN mkdir -p /app/data/tdb2 /app/data/chromem /app/logs /app/config

# Environment
ENV MIMIR_PORT=8080 \
    MIMIR_DATA_DIR=/app/data \
    JENA_HOME=/opt/jena \
    PATH="$PATH:/opt/jena/bin"

# Expose single port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run backend (which serves frontend)
CMD ["./mimir-aip-server", "--server"]
```

**Estimated Container Size**: ~300MB (vs 244MB current, adds JRE + Jena)

---

## 3. Persistence Layer Integration

### Building on SQLite PR

The existing PR provides:
- ✅ SQLite backend with WAL mode
- ✅ Job persistence
- ✅ Pipeline storage
- ✅ Execution history

### Ontology Metadata Extension

**New Tables** (add to existing schema):

```sql
-- Ontology metadata
CREATE TABLE ontologies (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    version TEXT NOT NULL,
    description TEXT,
    format TEXT NOT NULL, -- turtle, rdf/xml, json-ld
    base_uri TEXT NOT NULL,
    namespace TEXT,
    author TEXT,
    file_path TEXT NOT NULL, -- Path to .ttl file in /app/data/ontologies/
    tdb2_graph TEXT, -- Named graph URI in TDB2
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(name, version)
);

-- Ontology classes (denormalized for quick access)
CREATE TABLE ontology_classes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    ontology_id TEXT NOT NULL,
    uri TEXT NOT NULL,
    label TEXT NOT NULL,
    description TEXT,
    parent_uris TEXT, -- JSON array
    FOREIGN KEY (ontology_id) REFERENCES ontologies(id) ON DELETE CASCADE,
    UNIQUE(ontology_id, uri)
);

CREATE INDEX idx_classes_ontology ON ontology_classes(ontology_id);
CREATE INDEX idx_classes_label ON ontology_classes(label);

-- Ontology properties
CREATE TABLE ontology_properties (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    ontology_id TEXT NOT NULL,
    uri TEXT NOT NULL,
    label TEXT NOT NULL,
    property_type TEXT NOT NULL, -- ObjectProperty, DatatypeProperty
    domain_uris TEXT, -- JSON array
    range_uris TEXT, -- JSON array
    FOREIGN KEY (ontology_id) REFERENCES ontologies(id) ON DELETE CASCADE,
    UNIQUE(ontology_id, uri)
);

CREATE INDEX idx_properties_ontology ON ontology_properties(ontology_id);

-- Entity extraction jobs (track processing)
CREATE TABLE extraction_jobs (
    id TEXT PRIMARY KEY,
    ontology_id TEXT NOT NULL,
    source_type TEXT NOT NULL, -- csv, json, text, api
    source_path TEXT,
    status TEXT NOT NULL, -- pending, running, completed, failed
    entities_extracted INTEGER DEFAULT 0,
    triples_generated INTEGER DEFAULT 0,
    error_message TEXT,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (ontology_id) REFERENCES ontologies(id) ON DELETE CASCADE
);

CREATE INDEX idx_extraction_ontology ON extraction_jobs(ontology_id);
CREATE INDEX idx_extraction_status ON extraction_jobs(status);
```

### Dual Storage Strategy

```go
// pipelines/Storage/hybrid_backend.go
type HybridStorageBackend struct {
    // Metadata (fast lookups, relations)
    sqliteDB *sql.DB
    
    // RDF triples (semantic queries)
    tdb2Store *TDB2Backend
    
    // Vector embeddings (similarity search)
    chromemStore *ChromemBackend
}

// Store data in all relevant backends
func (h *HybridStorageBackend) StoreEntity(entity Entity) error {
    // 1. Generate RDF triples
    triples := h.entityToTriples(entity)
    
    // 2. Store in TDB2
    if err := h.tdb2Store.InsertTriples(ctx, triples); err != nil {
        return err
    }
    
    // 3. Generate embedding and store in Chromem
    embedding := h.generateEmbedding(entity.Text)
    doc := chromem.Document{
        ID: entity.URI,
        Content: entity.Text,
        Metadata: map[string]any{
            "type": entity.Type,
            "ontology": entity.OntologyID,
        },
    }
    if err := h.chromemStore.Store(ctx, doc); err != nil {
        return err
    }
    
    // 4. Update SQLite metadata (optional, for quick stats)
    _, err := h.sqliteDB.Exec(
        "INSERT INTO entity_stats (ontology_id, class_uri, count) VALUES (?, ?, 1) ON CONFLICT(ontology_id, class_uri) DO UPDATE SET count = count + 1",
        entity.OntologyID, entity.Type,
    )
    
    return err
}
```

---

## 4. Ontology Plugin System

### Plugin Types (Extend Existing Categories)

All ontology operations are **plugins**, not separate services:

```go
// pipelines/Ontology/plugin_types.go
package Ontology

// Existing plugin types:
// - Input
// - Data_Processing  
// - AIModels
// - Output
// - Storage (from existing Storage plugin)

// NEW: Ontology plugin type
const PluginTypeOntology = "Ontology"

// Ontology plugin operations (sub-plugins)
type OntologyOperations string

const (
    OpUpload        OntologyOperations = "upload"      // Upload ontology file
    OpValidate      OntologyOperations = "validate"    // Validate OWL/RDF
    OpQuery         OntologyOperations = "query"       // SPARQL query
    OpExtract       OntologyOperations = "extract"     // Entity extraction
    OpClassify      OntologyOperations = "classify"    // Classify data
    OpSimulate      OntologyOperations = "simulate"    // Digital twin
    OpStats         OntologyOperations = "stats"       // Graph statistics
)
```

### Core Ontology Plugins

#### 1. Ontology Management Plugin
```go
// pipelines/Ontology/management_plugin.go
type OntologyManagementPlugin struct {
    storage    *sql.DB
    tdb2Backend *TDB2Backend
}

func (p *OntologyManagementPlugin) ExecuteStep(
    ctx context.Context,
    stepConfig pipelines.StepConfig,
    globalContext *pipelines.PluginContext,
) (*pipelines.PluginContext, error) {
    
    operation := stepConfig.Config["operation"].(string)
    
    switch operation {
    case "upload":
        return p.uploadOntology(ctx, stepConfig, globalContext)
    case "validate":
        return p.validateOntology(ctx, stepConfig, globalContext)
    case "list":
        return p.listOntologies(ctx, stepConfig, globalContext)
    case "get":
        return p.getOntology(ctx, stepConfig, globalContext)
    default:
        return nil, fmt.Errorf("unknown operation: %s", operation)
    }
}

func (p *OntologyManagementPlugin) GetPluginType() string {
    return "Ontology"
}

func (p *OntologyManagementPlugin) GetPluginName() string {
    return "management"
}
```

**Example Pipeline Usage**:
```yaml
name: "Upload Infrastructure Ontology"
steps:
  - name: "Upload Ontology"
    plugin: "Ontology.management"
    config:
      operation: "upload"
      file_path: "/app/ontologies/infrastructure.ttl"
      name: "Infrastructure Ontology"
      version: "1.0"
    output: "ontology_metadata"
```

#### 2. SPARQL Query Plugin
```go
// pipelines/Ontology/query_plugin.go
type SPARQLQueryPlugin struct {
    tdb2Backend *TDB2Backend
    validator   *SPARQLValidator
}

func (p *SPARQLQueryPlugin) ExecuteStep(
    ctx context.Context,
    stepConfig pipelines.StepConfig,
    globalContext *pipelines.PluginContext,
) (*pipelines.PluginContext, error) {
    
    query := stepConfig.Config["query"].(string)
    
    // Validate query (security)
    if err := p.validator.Validate(query); err != nil {
        return nil, fmt.Errorf("invalid query: %w", err)
    }
    
    // Execute query
    results, err := p.tdb2Backend.Query(ctx, query)
    if err != nil {
        return nil, err
    }
    
    // Store results
    resultCtx := pipelines.NewPluginContext()
    resultCtx.Set(stepConfig.Output, results)
    
    return resultCtx, nil
}

func (p *SPARQLQueryPlugin) GetPluginType() string {
    return "Ontology"
}

func (p *SPARQLQueryPlugin) GetPluginName() string {
    return "sparql_query"
}
```

**Example Pipeline Usage**:
```yaml
name: "Query Infrastructure"
steps:
  - name: "Find Ubuntu Servers"
    plugin: "Ontology.sparql_query"
    config:
      query: |
        PREFIX infra: <http://example.org/infra/>
        SELECT ?server ?ip WHERE {
          ?server a infra:Server ;
                  infra:hasOS ?os ;
                  infra:hasIPAddress ?ip .
          FILTER(CONTAINS(?os, "Ubuntu"))
        }
      limit: 100
    output: "servers"
```

#### 3. Entity Extraction Plugin (Hybrid Approach)
```go
// pipelines/Ontology/extraction_plugin.go
type EntityExtractionPlugin struct {
    ontologyStore  *sql.DB
    tdb2Backend    *TDB2Backend
    llmClient      LLMClient // Optional, for unstructured data
}

func (p *EntityExtractionPlugin) ExecuteStep(
    ctx context.Context,
    stepConfig pipelines.StepConfig,
    globalContext *pipelines.PluginContext,
) (*pipelines.PluginContext, error) {
    
    ontologyID := stepConfig.Config["ontology_id"].(string)
    sourceType := stepConfig.Config["source_type"].(string)
    
    // Load ontology
    ontology, err := p.loadOntology(ontologyID)
    if err != nil {
        return nil, err
    }
    
    // Choose extraction strategy
    var extractor Extractor
    switch sourceType {
    case "csv", "json":
        extractor = NewDeterministicExtractor(ontology)
    case "text", "html":
        extractor = NewLLMExtractor(ontology, p.llmClient)
    case "hybrid":
        extractor = NewHybridExtractor(ontology, p.llmClient)
    default:
        return nil, fmt.Errorf("unsupported source type: %s", sourceType)
    }
    
    // Get source data from context
    sourceData, _ := globalContext.Get(stepConfig.Config["source_field"].(string))
    
    // Extract entities
    result, err := extractor.Extract(ctx, sourceData)
    if err != nil {
        return nil, err
    }
    
    // Store results
    resultCtx := pipelines.NewPluginContext()
    resultCtx.Set(stepConfig.Output, result)
    
    return resultCtx, nil
}
```

---

## 5. Hybrid Entity Extraction Strategy

### When to Use What

| Data Type | Approach | Rationale |
|-----------|----------|-----------|
| **CSV with headers** | Deterministic | Fast, accurate, column names map to properties |
| **JSON with schema** | Deterministic | Known structure, direct mapping |
| **Database records** | Deterministic | Schema-driven, no ambiguity |
| **Plain text** | LLM | Unstructured, needs understanding |
| **HTML/Web scraping** | Hybrid | Structure (DOM) + content (LLM) |
| **PDFs** | Hybrid | Layout analysis + text extraction |
| **Mixed sources** | Hybrid | Use best approach per section |

### Deterministic Extractor (Fast, Free)

```go
// pipelines/Ontology/deterministic_extractor.go
type DeterministicExtractor struct {
    ontology  *Ontology
    mappings  map[string]string // column/field -> property URI
}

func (e *DeterministicExtractor) Extract(ctx context.Context, data any) (*ExtractionResult, error) {
    switch v := data.(type) {
    case []map[string]any: // JSON array
        return e.extractFromJSON(v)
    case string: // CSV content
        return e.extractFromCSV(v)
    default:
        return nil, fmt.Errorf("unsupported data type")
    }
}

func (e *DeterministicExtractor) extractFromCSV(csvContent string) (*ExtractionResult, error) {
    reader := csv.NewReader(strings.NewReader(csvContent))
    records, err := reader.ReadAll()
    if err != nil {
        return nil, err
    }
    
    if len(records) == 0 {
        return &ExtractionResult{}, nil
    }
    
    // Headers -> Property mapping
    headers := records[0]
    mappings := e.inferMappings(headers)
    
    var entities []Entity
    var triples []Triple
    
    for i, record := range records[1:] {
        entityURI := fmt.Sprintf("%s/entity_%d", e.ontology.Metadata.BaseURI, i)
        
        // Infer entity type from data
        entityType := e.inferType(headers, record)
        
        // Create entity
        entity := Entity{
            URI:  entityURI,
            Type: entityType,
            Properties: make(map[string]any),
        }
        
        // Map columns to properties
        for j, value := range record {
            if j < len(headers) {
                propURI := mappings[headers[j]]
                entity.Properties[propURI] = value
                
                // Create triple
                triples = append(triples, Triple{
                    Subject:   entityURI,
                    Predicate: propURI,
                    Object:    formatLiteral(value),
                })
            }
        }
        
        entities = append(entities, entity)
    }
    
    return &ExtractionResult{
        Entities: entities,
        Triples:  triples,
    }, nil
}

// inferMappings uses fuzzy matching to map column names to ontology properties
func (e *DeterministicExtractor) inferMappings(headers []string) map[string]string {
    mappings := make(map[string]string)
    
    for _, header := range headers {
        // Normalize header
        normalized := strings.ToLower(strings.ReplaceAll(header, "_", " "))
        
        // Find best matching property
        bestMatch := ""
        bestScore := 0.0
        
        for _, prop := range e.ontology.Properties {
            score := similarity(normalized, strings.ToLower(prop.Label))
            if score > bestScore {
                bestScore = score
                bestMatch = prop.URI
            }
        }
        
        if bestScore > 0.7 { // Threshold
            mappings[header] = bestMatch
        } else {
            // Create temporary property
            mappings[header] = fmt.Sprintf("%s/prop_%s", e.ontology.Metadata.BaseURI, header)
        }
    }
    
    return mappings
}

// Simple Levenshtein-based similarity
func similarity(a, b string) float64 {
    // Implementation of string similarity metric
    // Could use github.com/agnivade/levenshtein
    return 0.0 // Placeholder
}
```

### LLM Extractor (Flexible, Expensive)

```go
// pipelines/Ontology/llm_extractor.go
type LLMExtractor struct {
    ontology  *Ontology
    llmClient LLMClient
}

func (e *LLMExtractor) Extract(ctx context.Context, data any) (*ExtractionResult, error) {
    text, ok := data.(string)
    if !ok {
        return nil, fmt.Errorf("LLM extractor requires text input")
    }
    
    // Build extraction prompt
    prompt := e.buildPrompt(text)
    
    // Call LLM
    response, err := e.llmClient.Complete(ctx, prompt)
    if err != nil {
        return nil, err
    }
    
    // Parse response
    return e.parseResponse(response)
}

func (e *LLMExtractor) buildPrompt(text string) string {
    // Compact ontology representation for prompt
    classesDesc := ""
    for _, class := range e.ontology.Classes {
        classesDesc += fmt.Sprintf("- %s (%s)\n", class.Label, class.URI)
    }
    
    propertiesDesc := ""
    for _, prop := range e.ontology.Properties {
        propertiesDesc += fmt.Sprintf("- %s (%s): domain=%v, range=%v\n", 
            prop.Label, prop.URI, prop.Domain, prop.Range)
    }
    
    return fmt.Sprintf(`Extract entities and relationships from the text using the ontology below.

ONTOLOGY CLASSES:
%s

PROPERTIES:
%s

TEXT:
%s

OUTPUT (JSON only):
{
  "entities": [
    {"uri": "...", "type": "class_uri", "label": "...", "confidence": 0.95}
  ],
  "relationships": [
    {"subject": "entity_uri", "predicate": "property_uri", "object": "entity_uri", "confidence": 0.90}
  ]
}
`, classesDesc, propertiesDesc, text)
}
```

### Hybrid Extractor (Best of Both)

```go
// pipelines/Ontology/hybrid_extractor.go
type HybridExtractor struct {
    deterministic *DeterministicExtractor
    llm           *LLMExtractor
}

func (e *HybridExtractor) Extract(ctx context.Context, data any) (*ExtractionResult, error) {
    // Try deterministic first
    result, err := e.deterministic.Extract(ctx, data)
    if err == nil && result.Confidence > 0.8 {
        return result, nil
    }
    
    // Fall back to LLM for low-confidence extractions
    llmResult, err := e.llm.Extract(ctx, data)
    if err != nil {
        // If LLM fails, return deterministic result anyway
        return result, nil
    }
    
    // Merge results (LLM can fill gaps in deterministic extraction)
    return e.mergeResults(result, llmResult), nil
}
```

**Cost Comparison** (1000 entities):
- Deterministic: ~0ms per entity, $0
- LLM: ~500ms per entity, ~$0.50-$2.00 (depending on model)
- Hybrid: ~50ms average per entity, ~$0.10-$0.40 (only uses LLM when needed)

---

## 6. Frontend Integration

### New Frontend Pages

```
mimir-aip-frontend/src/app/
├── ontologies/              # NEW
│   ├── page.tsx            # List ontologies
│   ├── [id]/               # Ontology details
│   │   └── page.tsx
│   ├── upload/             # Upload ontology
│   │   └── page.tsx
│   └── visualize/          # Graph visualization
│       └── page.tsx
├── knowledge-graph/         # NEW
│   ├── page.tsx            # Query interface
│   ├── browser/            # Browse entities
│   │   └── page.tsx
│   └── stats/              # Statistics dashboard
│       └── page.tsx
├── extraction/              # NEW
│   ├── page.tsx            # Extraction jobs
│   └── [id]/               # Job details
│       └── page.tsx
└── pipelines/               # EXISTING (add ontology templates)
    └── templates/           # NEW
        └── page.tsx
```

### Ontology Management Page

**File**: `mimir-aip-frontend/src/app/ontologies/page.tsx`

```tsx
"use client";
import { useState, useEffect } from "react";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import Link from "next/link";

interface Ontology {
  id: string;
  name: string;
  version: string;
  description: string;
  format: string;
  base_uri: string;
  class_count: number;
  property_count: number;
  created_at: string;
}

export default function OntologiesPage() {
  const [ontologies, setOntologies] = useState<Ontology[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchOntologies();
  }, []);

  async function fetchOntologies() {
    const res = await fetch("/api/v1/ontology");
    const data = await res.json();
    setOntologies(data.ontologies || []);
    setLoading(false);
  }

  return (
    <div>
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold text-orange">Ontologies</h1>
        <div className="flex gap-2">
          <Button asChild>
            <Link href="/ontologies/upload">Upload Ontology</Link>
          </Button>
          <Button variant="outline" asChild>
            <Link href="/ontologies/templates">Browse Templates</Link>
          </Button>
        </div>
      </div>

      {loading && <p>Loading...</p>}

      {!loading && ontologies.length === 0 && (
        <Card className="bg-navy text-white border-blue p-8 text-center">
          <p className="text-white/60 mb-4">No ontologies found</p>
          <Button asChild>
            <Link href="/ontologies/upload">Upload Your First Ontology</Link>
          </Button>
        </Card>
      )}

      {!loading && ontologies.length > 0 && (
        <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-6">
          {ontologies.map((ont) => (
            <Card key={ont.id} className="bg-navy text-white border-blue p-6">
              <div className="flex justify-between items-start mb-3">
                <h2 className="text-xl font-bold text-orange">{ont.name}</h2>
                <Badge className="bg-blue text-white">{ont.version}</Badge>
              </div>
              
              <p className="text-sm text-white/60 mb-4">{ont.description}</p>
              
              <div className="space-y-2 text-sm mb-4">
                <div className="flex justify-between">
                  <span className="text-white/60">Classes:</span>
                  <span className="font-semibold">{ont.class_count}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-white/60">Properties:</span>
                  <span className="font-semibold">{ont.property_count}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-white/60">Format:</span>
                  <span className="font-mono text-xs">{ont.format}</span>
                </div>
              </div>

              <div className="flex gap-2">
                <Button asChild size="sm" variant="default">
                  <Link href={`/ontologies/${ont.id}`}>View</Link>
                </Button>
                <Button asChild size="sm" variant="outline">
                  <Link href={`/knowledge-graph?ontology=${ont.id}`}>
                    Query Graph
                  </Link>
                </Button>
              </div>
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}
```

### Knowledge Graph Query Interface

**File**: `mimir-aip-frontend/src/app/knowledge-graph/page.tsx`

```tsx
"use client";
import { useState } from "react";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { Table } from "@/components/ui/table";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";

export default function KnowledgeGraphPage() {
  const [query, setQuery] = useState("");
  const [nlQuery, setNlQuery] = useState("");
  const [results, setResults] = useState<any>(null);
  const [loading, setLoading] = useState(false);
  const [activeTab, setActiveTab] = useState<"sparql" | "natural">("natural");

  async function executeSPARQL() {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/kg/query", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ query }),
      });
      const data = await res.json();
      setResults(data);
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  }

  async function executeNaturalLanguage() {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/kg/nl-query", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ question: nlQuery }),
      });
      const data = await res.json();
      setResults(data);
      if (data.generated_query) {
        setQuery(data.generated_query);
      }
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  }

  return (
    <div>
      <h1 className="text-2xl font-bold text-orange mb-6">
        Knowledge Graph Query
      </h1>

      <Tabs value={activeTab} onValueChange={(v) => setActiveTab(v as any)}>
        <TabsList className="mb-4">
          <TabsTrigger value="natural">Natural Language</TabsTrigger>
          <TabsTrigger value="sparql">SPARQL</TabsTrigger>
        </TabsList>

        <TabsContent value="natural">
          <Card className="bg-navy text-white border-blue p-6 mb-6">
            <Label className="text-white mb-2 block">
              Ask a question in plain English:
            </Label>
            <Textarea
              value={nlQuery}
              onChange={(e) => setNlQuery(e.target.value)}
              placeholder="e.g., Show me all servers running Ubuntu in us-east-1"
              className="bg-blue/10 border-blue text-white mb-4 min-h-[100px]"
            />
            <Button onClick={executeNaturalLanguage} disabled={loading}>
              {loading ? "Querying..." : "Ask"}
            </Button>
          </Card>
        </TabsContent>

        <TabsContent value="sparql">
          <Card className="bg-navy text-white border-blue p-6 mb-6">
            <Label className="text-white mb-2 block">SPARQL Query:</Label>
            <Textarea
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="SELECT ?s ?p ?o WHERE { ?s ?p ?o } LIMIT 10"
              className="bg-blue/10 border-blue text-white mb-4 font-mono text-sm min-h-[150px]"
            />
            <Button onClick={executeSPARQL} disabled={loading}>
              {loading ? "Executing..." : "Execute Query"}
            </Button>
          </Card>
        </TabsContent>
      </Tabs>

      {/* Results Display */}
      {results && (
        <Card className="bg-navy text-white border-blue p-6">
          <h2 className="text-xl font-bold text-orange mb-4">
            Results ({results.count} rows)
          </h2>
          
          {activeTab === "natural" && results.generated_query && (
            <div className="mb-4 p-4 bg-blue/10 rounded">
              <p className="text-sm text-white/60 mb-2">Generated SPARQL:</p>
              <code className="text-xs font-mono">{results.generated_query}</code>
            </div>
          )}

          <div className="overflow-x-auto">
            <Table className="text-white">
              <thead>
                <tr>
                  {results.variables?.map((v: string) => (
                    <th key={v} className="text-left p-2 border-b border-blue">
                      {v}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {results.bindings?.map((row: any, i: number) => (
                  <tr key={i} className="border-b border-blue/30">
                    {results.variables?.map((v: string) => (
                      <td key={v} className="p-2">
                        {row[v]?.value || "-"}
                      </td>
                    ))}
                  </tr>
                ))}
              </tbody>
            </Table>
          </div>
        </Card>
      )}
    </div>
  );
}
```

### API Client Updates

**File**: `mimir-aip-frontend/src/lib/api.ts` (add to existing)

```typescript
// Ontology Management
export interface Ontology {
  id: string;
  name: string;
  version: string;
  description: string;
  format: string;
  base_uri: string;
  class_count: number;
  property_count: number;
  created_at: string;
}

export async function getOntologies(): Promise<Ontology[]> {
  const res = await fetch("/api/v1/ontology");
  if (!res.ok) throw new Error("Failed to fetch ontologies");
  const data = await res.json();
  return data.ontologies || [];
}

export async function getOntology(id: string): Promise<Ontology> {
  const res = await fetch(`/api/v1/ontology/${id}`);
  if (!res.ok) throw new Error("Failed to fetch ontology");
  return res.json();
}

export async function uploadOntology(file: File, metadata: any): Promise<any> {
  const formData = new FormData();
  formData.append("ontology", file);
  Object.keys(metadata).forEach((key) => {
    formData.append(key, metadata[key]);
  });

  const res = await fetch("/api/v1/ontology", {
    method: "POST",
    body: formData,
  });

  if (!res.ok) throw new Error("Failed to upload ontology");
  return res.json();
}

// Knowledge Graph
export async function querySPARQL(query: string): Promise<any> {
  const res = await fetch("/api/v1/kg/query", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ query }),
  });

  if (!res.ok) throw new Error("SPARQL query failed");
  return res.json();
}

export async function naturalLanguageQuery(question: string): Promise<any> {
  const res = await fetch("/api/v1/kg/nl-query", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ question }),
  });

  if (!res.ok) throw new Error("Natural language query failed");
  return res.json();
}

// Extraction Jobs
export interface ExtractionJob {
  id: string;
  ontology_id: string;
  source_type: string;
  status: string;
  entities_extracted: number;
  triples_generated: number;
  created_at: string;
}

export async function getExtractionJobs(): Promise<ExtractionJob[]> {
  const res = await fetch("/api/v1/extraction/jobs");
  if (!res.ok) throw new Error("Failed to fetch extraction jobs");
  const data = await res.json();
  return data.jobs || [];
}

export async function createExtractionJob(ontologyId: string, source: any): Promise<any> {
  const res = await fetch("/api/v1/extraction/jobs", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ ontology_id: ontologyId, source }),
  });

  if (!res.ok) throw new Error("Failed to create extraction job");
  return res.json();
}
```

---

## 7. Updated Implementation Phases

### Phase 1: Foundation (Weeks 1-2)
✅ **Goal**: Embedded Jena TDB2 + Ontology plugins working

**Tasks**:
1. **Jena Integration**
   - Add JRE to Dockerfile.unified
   - Create TDB2 wrapper (Java subprocess or JNI)
   - Test basic triple insert/query
   
2. **Persistence Layer**
   - Merge SQLite persistence PR
   - Add ontology metadata tables
   - Implement hybrid storage backend

3. **Ontology Plugins**
   - Create `pipelines/Ontology/` module
   - Implement management plugin (upload, list, get)
   - Implement SPARQL query plugin
   - Register plugins in main registry

4. **Frontend - Ontology Management**
   - Create `/ontologies` page
   - Create `/ontologies/upload` page
   - Add API client functions

**Deliverable**: Can upload ontology, view it in UI, query via SPARQL

---

### Phase 2: Entity Extraction (Weeks 3-4)
✅ **Goal**: Automated data → knowledge graph with hybrid extraction

**Tasks**:
1. **Deterministic Extractor**
   - CSV → entities (column mapping)
   - JSON → entities (schema mapping)
   - Implement fuzzy property matching

2. **LLM Extractor**
   - Reuse existing OpenAI plugin
   - Build extraction prompts
   - Parse LLM JSON responses

3. **Hybrid Extractor**
   - Combine deterministic + LLM
   - Confidence-based fallback
   - Cost optimization

4. **Extraction Plugin**
   - Implement entity extraction plugin
   - Support multiple source types
   - Store extracted triples in TDB2

5. **Frontend - Extraction Jobs**
   - Create `/extraction` page
   - Job status tracking
   - Results visualization

**Deliverable**: Upload CSV, auto-extract entities, view in knowledge graph

---

### Phase 3: Natural Language Queries (Weeks 5-6)
✅ **Goal**: Query knowledge graph in plain English

**Tasks**:
1. **NL → SPARQL Plugin**
   - Implement NL query plugin
   - LLM prompt for SPARQL generation
   - Query validation & security

2. **MCP Integration**
   - Register ontology plugins as MCP tools
   - `ontology.query` tool
   - `ontology.extract` tool
   - `ontology.classify` tool (Phase 4)

3. **Frontend - Query Interface**
   - Create `/knowledge-graph` page
   - Dual tabs: Natural Language + SPARQL
   - Results table display
   - Show generated SPARQL from NL

**Deliverable**: LLM agents + users can query knowledge graph naturally

---

### Phase 4: Classification & Analytics (Weeks 7-8)
✅ **Goal**: ML-powered classification, graph analytics

**Tasks**:
1. **Classification Plugin**
   - Implement classification plugin
   - Support rule-based classification (fast)
   - Optional ML-based (ONNX, Phase 4b)

2. **Graph Analytics Plugin**
   - Path finding
   - Centrality measures
   - Subgraph queries

3. **Frontend - Analytics Dashboard**
   - Graph statistics
   - Entity type distribution
   - Relationship visualization

**Deliverable**: Automatic classification, graph insights

---

### Phase 5: Digital Twin (Weeks 9-10)
✅ **Goal**: "What-if" simulation engine

**Tasks**:
1. **Simulation Plugin**
   - Impact analysis queries
   - Dependency traversal
   - LLM-powered narrative generation

2. **Temporal Queries**
   - Time-aware SPARQL
   - Historical snapshots (versioning)

3. **Frontend - Simulation UI**
   - Simulation builder
   - Impact reports
   - Scenario comparison

**Deliverable**: Simulate events, predict impacts

---

### Phase 6: Production (Weeks 11-12)
✅ **Goal**: Deployment, docs, optimization

**Tasks**:
1. **Performance Tuning**
   - TDB2 indexing optimization
   - Query caching
   - Batch insert optimization

2. **Documentation**
   - User guides (frontend workflows)
   - API documentation
   - Ontology development guide
   - Example ontologies + datasets

3. **Testing**
   - Integration tests (full pipeline)
   - Performance benchmarks
   - Security audit

**Deliverable**: Production-ready ontology platform

---

## 8. MCP Tool Registration (Agent Integration)

### Register Ontology Plugins as MCP Tools

**File**: Update `mcp_server.go`

```go
// Add ontology tools to MCP server
func (m *MCPServer) registerOntologyTools() {
    // Register SPARQL query tool
    m.registerTool(MCPTool{
        Name: "ontology.query",
        Description: "Query the knowledge graph using SPARQL or natural language",
        InputSchema: map[string]interface{}{
            "type": "object",
            "properties": map[string]interface{}{
                "query": map[string]interface{}{
                    "type": "string",
                    "description": "SPARQL query or natural language question",
                },
                "mode": map[string]interface{}{
                    "type": "string",
                    "enum": []string{"sparql", "natural"},
                    "default": "natural",
                },
            },
            "required": []string{"query"},
        },
        Handler: m.handleOntologyQuery,
    })

    // Register entity extraction tool
    m.registerTool(MCPTool{
        Name: "ontology.extract",
        Description: "Extract entities and relationships from data using an ontology",
        InputSchema: map[string]interface{}{
            "type": "object",
            "properties": map[string]interface{}{
                "ontology_id": map[string]interface{}{
                    "type": "string",
                    "description": "ID of the ontology to use",
                },
                "data": map[string]interface{}{
                    "type": "string",
                    "description": "Data to extract from (text, CSV, JSON)",
                },
                "source_type": map[string]interface{}{
                    "type": "string",
                    "enum": []string{"text", "csv", "json", "hybrid"},
                },
            },
            "required": []string{"ontology_id", "data"},
        },
        Handler: m.handleEntityExtraction,
    })

    // Register classification tool
    m.registerTool(MCPTool{
        Name: "ontology.classify",
        Description: "Classify data against ontology classes",
        InputSchema: map[string]interface{}{
            "type": "object",
            "properties": map[string]interface{}{
                "ontology_id": map[string]interface{}{
                    "type": "string",
                },
                "data": map[string]interface{}{
                    "type": "string",
                },
            },
            "required": []string{"ontology_id", "data"},
        },
        Handler: m.handleClassification,
    })
}

func (m *MCPServer) handleOntologyQuery(args map[string]interface{}) (interface{}, error) {
    query := args["query"].(string)
    mode := args["mode"].(string)
    
    if mode == "natural" {
        // Use NL query plugin
        return m.executePlugin("Ontology.nl_query", map[string]interface{}{
            "question": query,
        })
    }
    
    // Use SPARQL query plugin
    return m.executePlugin("Ontology.sparql_query", map[string]interface{}{
        "query": query,
    })
}
```

**LLM Agent Example**:
```python
# Using Claude with MCP
from anthropic import Anthropic

client = Anthropic()

response = client.messages.create(
    model="claude-3-5-sonnet-20241022",
    max_tokens=1024,
    tools=[
        {
            "name": "ontology.query",
            "description": "Query the Mimir knowledge graph",
            "input_schema": {
                "type": "object",
                "properties": {
                    "query": {"type": "string"},
                    "mode": {"type": "string", "enum": ["sparql", "natural"]}
                }
            }
        }
    ],
    messages=[
        {"role": "user", "content": "Find all servers running Ubuntu in the us-east-1 datacenter"}
    ]
)

# Claude will use ontology.query tool automatically
```

---

## 9. Docker Compose Configuration

**File**: Update `docker-compose.unified.yml`

```yaml
version: '3.8'

services:
  mimir-aip-unified:
    build:
      context: .
      dockerfile: Dockerfile.unified
    image: mimir-aip:unified-ontology
    container_name: mimir-aip-unified
    restart: unless-stopped
    
    ports:
      - "8080:8080"
    
    environment:
      # Existing
      - MIMIR_PORT=8080
      - MIMIR_LOG_LEVEL=INFO
      - MIMIR_DATA_DIR=/app/data
      
      # NEW: Ontology/Knowledge Graph
      - MIMIR_ONTOLOGY_DIR=/app/data/ontologies
      - MIMIR_TDB2_DIR=/app/data/tdb2
      - MIMIR_ENABLE_ONTOLOGY=true
      - JENA_HOME=/opt/jena
      
      # NEW: Extraction
      - MIMIR_EXTRACTION_MODE=hybrid  # deterministic, llm, hybrid
      - OPENAI_API_KEY=${OPENAI_API_KEY:-}  # Optional for LLM extraction
      
      # Database paths (from persistence PR)
      - MIMIR_DATABASE_PATH=/app/data/mimir.db
      - MIMIR_VECTOR_DB_PATH=/app/data/chromem
    
    volumes:
      # Existing
      - mimir_data:/app/data
      - mimir_logs:/app/logs
      - mimir_config:/app/config
      - mimir_plugins:/app/plugins
      - mimir_pipelines:/app/pipelines
      
      # NEW: Ontology files
      - mimir_ontologies:/app/data/ontologies
      - mimir_tdb2:/app/data/tdb2
    
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
    
    deploy:
      resources:
        limits:
          cpus: '2.0'
          memory: 2G  # Increased for JVM + TDB2
        reservations:
          cpus: '1.0'
          memory: 1G

volumes:
  mimir_data:
  mimir_logs:
  mimir_config:
  mimir_plugins:
  mimir_pipelines:
  mimir_ontologies:  # NEW
  mimir_tdb2:        # NEW

networks:
  mimir-network:
    driver: bridge
```

---

## 10. Success Metrics & Testing

### Phase 1 Acceptance Criteria
- ✅ Jena TDB2 embedded, can insert/query triples
- ✅ Upload ontology via UI, stored in SQLite + file system
- ✅ List ontologies in frontend
- ✅ Execute basic SPARQL query via plugin
- ✅ Unified container builds successfully (~300MB)

### Phase 2 Acceptance Criteria
- ✅ CSV upload → auto-extract entities (deterministic)
- ✅ Text input → extract entities (LLM)
- ✅ Hybrid extractor chooses best approach
- ✅ Extracted triples stored in TDB2
- ✅ View extraction jobs in UI

### Phase 3 Acceptance Criteria
- ✅ Natural language query → SPARQL → results
- ✅ MCP tools registered and callable
- ✅ LLM agent can query knowledge graph
- ✅ Query results displayed in UI table

### Performance Targets
- **Container Size**: <350MB (unified + JRE + Jena)
- **TDB2 Storage**: Support 1M+ triples
- **SPARQL Query**: <200ms p95 latency
- **Entity Extraction**: 
  - Deterministic: <10ms per entity
  - LLM: <1s per entity (with caching)
  - Hybrid: <100ms average
- **Ontology Upload**: <5s for 1000-class ontology

---

## 11. Summary of Changes from Original Design

| Aspect | Original Design | Revised Design |
|--------|----------------|----------------|
| **Deployment** | Separate Fuseki container | Embedded Jena TDB2 in unified container |
| **Container Count** | 2-3 (Mimir + Fuseki + ML) | 1 (all-in-one) |
| **Agent Tools** | Separate API endpoints | Part of plugin system |
| **Extraction** | LLM-only | Hybrid (deterministic + LLM) |
| **Frontend** | Not specified | Full UI for ontology management |
| **Persistence** | New design | Builds on existing SQLite PR |
| **Storage** | RDF-only | Dual (TDB2 + Chromem hybrid) |

---

## 12. Next Steps

1. **Review & Approve** this revised design
2. **Merge PR #1** (SQLite persistence) - foundation for ontology metadata
3. **Phase 1 Kickoff**: 
   - Add JRE + Jena to Dockerfile
   - Create TDB2 wrapper
   - Implement ontology management plugin
   - Build `/ontologies` frontend page
4. **Weekly Iteration**: Deploy → test → iterate

**Estimated Timeline**: 12 weeks remains realistic

---

**Document Version**: 2.0 (Revised)  
**Author**: Mimir AIP Development Team  
**Date**: 2025-01-15  
**Status**: Ready for Phase 1 Implementation

**Key Benefits**:
- ✅ **One container** - simpler deployment
- ✅ **Plugin-based** - consistent architecture  
- ✅ **Full UI** - complete user experience
- ✅ **Cost-optimized** - deterministic extraction where possible
- ✅ **Builds on existing work** - persistence PR, frontend patterns
