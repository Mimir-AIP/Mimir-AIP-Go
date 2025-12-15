# Ontology Evolution & Schema Management

## Problem Statement

Businesses evolve: databases change, new systems are added, data structures shift. Your ontology needs to evolve too, without breaking existing data or requiring manual intervention.

**Real-World Scenario**:
```
Month 1: Company has "Server" class with properties: hostname, ip_address, os
Month 6: Add "Container" class, "Server" gains new property: kubernetes_node
Month 12: Deprecate physical servers, everything moves to cloud VMs
         Add "CloudInstance" class with region, instance_type, cost
```

How do we handle this without:
- ❌ Losing historical data
- ❌ Breaking existing queries/pipelines
- ❌ Requiring manual data migration
- ❌ Downtime

---

## Solution: Automated Ontology Evolution System

### Core Components

1. **Schema Drift Detection** - Automatically detect when source data doesn't match ontology
2. **Ontology Versioning** - Track changes, maintain compatibility
3. **Migration Pipelines** - Automatically update existing triples
4. **Backward Compatibility** - Old queries still work
5. **Schema Suggestions** - LLM proposes ontology updates based on new data

---

## 1. Ontology Versioning System

### Version Management

**Schema Addition** (extend existing `ontologies` table):

```sql
-- From existing design
CREATE TABLE ontologies (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    version TEXT NOT NULL,
    description TEXT,
    format TEXT NOT NULL,
    base_uri TEXT NOT NULL,
    namespace TEXT,
    author TEXT,
    file_path TEXT NOT NULL,
    tdb2_graph TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(name, version)
);

-- NEW: Version tracking and relationships
CREATE TABLE ontology_versions (
    id TEXT PRIMARY KEY,
    ontology_name TEXT NOT NULL,
    version TEXT NOT NULL,
    parent_version TEXT, -- Previous version
    change_type TEXT NOT NULL, -- 'minor' (additive), 'major' (breaking)
    changelog TEXT, -- JSON array of changes
    is_active BOOLEAN DEFAULT TRUE,
    deprecated_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (parent_version) REFERENCES ontology_versions(id),
    UNIQUE(ontology_name, version)
);

-- NEW: Track what changed between versions
CREATE TABLE ontology_changes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    version_id TEXT NOT NULL,
    change_type TEXT NOT NULL, -- 'add_class', 'add_property', 'deprecate_class', 'rename_property'
    entity_type TEXT NOT NULL, -- 'class', 'property', 'individual'
    entity_uri TEXT NOT NULL,
    old_value TEXT, -- JSON of old definition
    new_value TEXT, -- JSON of new definition
    migration_required BOOLEAN DEFAULT FALSE,
    migration_status TEXT DEFAULT 'pending', -- 'pending', 'in_progress', 'completed', 'failed'
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (version_id) REFERENCES ontology_versions(id)
);

CREATE INDEX idx_changes_version ON ontology_changes(version_id);
CREATE INDEX idx_changes_status ON ontology_changes(migration_status);
```

### Semantic Versioning for Ontologies

```
version: MAJOR.MINOR.PATCH

MAJOR: Breaking changes (rename/delete classes, change property domains)
MINOR: Additive changes (new classes, new properties)
PATCH: Clarifications (label changes, description updates)
```

**Example Evolution**:
```
v1.0.0: Initial ontology (Server, hasIPAddress, hasOS)
v1.1.0: Add Container class, add hasKubernetesNode property (MINOR - additive)
v1.2.0: Add CloudInstance class (MINOR - additive)
v2.0.0: Deprecate Server, migrate to CloudInstance (MAJOR - breaking)
```

---

## 2. Schema Drift Detection

### Automatic Detection Plugin

**File**: `pipelines/Ontology/drift_detector_plugin.go`

```go
package Ontology

import (
    "context"
    "encoding/json"
    "fmt"
)

// DriftDetector detects when incoming data doesn't match ontology
type DriftDetector struct {
    ontology      *Ontology
    sqliteDB      *sql.DB
    llmClient     LLMClient // For intelligent suggestions
    alertThreshold float64  // % of unmapped fields before alerting
}

// DriftReport contains detected schema drift
type DriftReport struct {
    OntologyID     string           `json:"ontology_id"`
    OntologyVersion string          `json:"ontology_version"`
    DetectedAt     time.Time        `json:"detected_at"`
    SourceType     string           `json:"source_type"` // csv, json, database
    UnmappedFields []UnmappedField  `json:"unmapped_fields"`
    NewPatterns    []DataPattern    `json:"new_patterns"`
    Suggestions    []OntologySuggestion `json:"suggestions"`
    Severity       string           `json:"severity"` // low, medium, high
}

type UnmappedField struct {
    FieldName   string   `json:"field_name"`
    DataType    string   `json:"data_type"`
    SampleValues []string `json:"sample_values"`
    Frequency   int      `json:"frequency"` // How many times seen
    FirstSeen   time.Time `json:"first_seen"`
}

type DataPattern struct {
    Pattern     string   `json:"pattern"`
    Description string   `json:"description"`
    Examples    []string `json:"examples"`
}

type OntologySuggestion struct {
    Type        string `json:"type"` // 'add_class', 'add_property', 'extend_domain'
    EntityURI   string `json:"entity_uri"`
    Label       string `json:"label"`
    Definition  string `json:"definition"`
    Justification string `json:"justification"`
    Confidence  float64 `json:"confidence"`
}

// DetectDrift analyzes incoming data for schema drift
func (d *DriftDetector) DetectDrift(ctx context.Context, data interface{}, sourceType string) (*DriftReport, error) {
    report := &DriftReport{
        OntologyID:      d.ontology.Metadata.ID,
        OntologyVersion: d.ontology.Metadata.Version,
        DetectedAt:      time.Now(),
        SourceType:      sourceType,
    }

    // Extract schema from data
    schema := d.extractSchema(data, sourceType)
    
    // Compare with ontology
    unmapped := d.findUnmappedFields(schema)
    report.UnmappedFields = unmapped
    
    if len(unmapped) == 0 {
        return report, nil // No drift
    }
    
    // Calculate severity
    unmappedRatio := float64(len(unmapped)) / float64(len(schema.Fields))
    if unmappedRatio > 0.3 {
        report.Severity = "high"
    } else if unmappedRatio > 0.1 {
        report.Severity = "medium"
    } else {
        report.Severity = "low"
    }
    
    // Detect patterns
    patterns := d.detectPatterns(unmapped)
    report.NewPatterns = patterns
    
    // Generate suggestions using LLM
    suggestions, err := d.generateSuggestions(ctx, unmapped, patterns)
    if err == nil {
        report.Suggestions = suggestions
    }
    
    // Store drift report
    d.storeDriftReport(report)
    
    return report, nil
}

// extractSchema extracts field names and types from data
func (d *DriftDetector) extractSchema(data interface{}, sourceType string) *DataSchema {
    switch sourceType {
    case "csv":
        return d.extractCSVSchema(data)
    case "json":
        return d.extractJSONSchema(data)
    case "database":
        return d.extractDatabaseSchema(data)
    default:
        return &DataSchema{}
    }
}

type DataSchema struct {
    Fields []SchemaField
}

type SchemaField struct {
    Name      string
    Type      string
    Nullable  bool
    Samples   []string
}

// findUnmappedFields compares data schema with ontology
func (d *DriftDetector) findUnmappedFields(schema *DataSchema) []UnmappedField {
    var unmapped []UnmappedField
    
    for _, field := range schema.Fields {
        // Try to find matching property in ontology
        matched := false
        
        for _, prop := range d.ontology.Properties {
            // Fuzzy match: normalize field name and property label
            if d.fuzzyMatch(field.Name, prop.Label) {
                matched = true
                break
            }
        }
        
        if !matched {
            unmapped = append(unmapped, UnmappedField{
                FieldName:    field.Name,
                DataType:     field.Type,
                SampleValues: field.Samples,
                Frequency:    1,
                FirstSeen:    time.Now(),
            })
        }
    }
    
    return unmapped
}

// generateSuggestions uses LLM to suggest ontology changes
func (d *DriftDetector) generateSuggestions(ctx context.Context, unmapped []UnmappedField, patterns []DataPattern) ([]OntologySuggestion, error) {
    prompt := d.buildSuggestionPrompt(unmapped, patterns)
    
    response, err := d.llmClient.Complete(ctx, prompt)
    if err != nil {
        return nil, err
    }
    
    return d.parseSuggestions(response)
}

func (d *DriftDetector) buildSuggestionPrompt(unmapped []UnmappedField, patterns []DataPattern) string {
    unmappedJSON, _ := json.MarshalIndent(unmapped, "", "  ")
    patternsJSON, _ := json.MarshalIndent(patterns, "", "  ")
    
    ontologyDesc := ""
    for _, class := range d.ontology.Classes {
        ontologyDesc += fmt.Sprintf("- Class: %s (%s)\n", class.Label, class.URI)
    }
    for _, prop := range d.ontology.Properties {
        ontologyDesc += fmt.Sprintf("- Property: %s (%s)\n", prop.Label, prop.URI)
    }
    
    return fmt.Sprintf(`You are an ontology engineer. Analyze the unmapped fields and suggest ontology changes.

CURRENT ONTOLOGY:
%s

UNMAPPED FIELDS:
%s

DETECTED PATTERNS:
%s

Provide suggestions as JSON array:
{
  "suggestions": [
    {
      "type": "add_property",
      "entity_uri": "http://example.org/ont/hasKubernetesNode",
      "label": "has Kubernetes node",
      "definition": "Links a container to its Kubernetes node",
      "justification": "Field 'kubernetes_node' appears frequently with node names",
      "confidence": 0.92
    }
  ]
}

Rules:
1. Prefer extending existing classes over creating new ones
2. Use ontology namespace for new URIs
3. Explain your reasoning
4. Only suggest changes with confidence > 0.7
`, ontologyDesc, string(unmappedJSON), string(patternsJSON))
}
```

### Drift Detection Plugin

```go
// DriftDetectorPlugin integrates drift detection into pipeline
type DriftDetectorPlugin struct {
    detector *DriftDetector
}

func (p *DriftDetectorPlugin) ExecuteStep(
    ctx context.Context,
    stepConfig pipelines.StepConfig,
    globalContext *pipelines.PluginContext,
) (*pipelines.PluginContext, error) {
    
    // Get data from context
    dataField := stepConfig.Config["data_field"].(string)
    data, _ := globalContext.Get(dataField)
    
    sourceType := stepConfig.Config["source_type"].(string)
    
    // Detect drift
    report, err := p.detector.DetectDrift(ctx, data, sourceType)
    if err != nil {
        return nil, err
    }
    
    // Store report in context
    resultCtx := pipelines.NewPluginContext()
    resultCtx.Set(stepConfig.Output, report)
    
    // Alert if high severity
    if report.Severity == "high" {
        p.sendAlert(report)
    }
    
    return resultCtx, nil
}

func (p *DriftDetectorPlugin) GetPluginType() string {
    return "Ontology"
}

func (p *DriftDetectorPlugin) GetPluginName() string {
    return "drift_detector"
}
```

**Example Pipeline** (run this on every data ingestion):

```yaml
name: "Ingest with Drift Detection"
steps:
  - name: "Load CSV"
    plugin: "Input.csv"
    config:
      file_path: "/data/servers.csv"
    output: "raw_data"
  
  - name: "Check for Schema Drift"
    plugin: "Ontology.drift_detector"
    config:
      ontology_id: "infrastructure-v1"
      data_field: "raw_data"
      source_type: "csv"
      alert_threshold: 0.1  # Alert if >10% fields unmapped
    output: "drift_report"
  
  - name: "Extract Entities (if drift is low)"
    plugin: "Ontology.extraction"
    config:
      ontology_id: "infrastructure-v1"
      source_field: "raw_data"
      source_type: "csv"
    output: "entities"
```

---

## 3. Semi-Automated Ontology Updates

### Three Update Modes

#### Mode 1: Fully Automated (Low Risk)
**For**: Adding new optional properties, new subclasses

```go
// AutoUpdatePolicy defines when to auto-apply changes
type AutoUpdatePolicy struct {
    AutoAddProperties  bool `json:"auto_add_properties"`  // New optional properties
    AutoAddClasses     bool `json:"auto_add_classes"`     // New subclasses
    RequireApproval    bool `json:"require_approval"`     // Require human review
    MinConfidence      float64 `json:"min_confidence"`    // Minimum LLM confidence
}

// ApplySuggestion automatically applies a suggestion if policy allows
func (om *OntologyManager) ApplySuggestion(ctx context.Context, suggestion OntologySuggestion) error {
    policy := om.getUpdatePolicy()
    
    // Check if auto-apply is allowed
    if !om.canAutoApply(suggestion, policy) {
        return om.queueForApproval(suggestion)
    }
    
    // Apply change
    switch suggestion.Type {
    case "add_property":
        return om.addProperty(ctx, suggestion)
    case "add_class":
        return om.addClass(ctx, suggestion)
    default:
        return om.queueForApproval(suggestion)
    }
}

func (om *OntologyManager) canAutoApply(suggestion OntologySuggestion, policy AutoUpdatePolicy) bool {
    if suggestion.Confidence < policy.MinConfidence {
        return false
    }
    
    switch suggestion.Type {
    case "add_property":
        return policy.AutoAddProperties && !policy.RequireApproval
    case "add_class":
        return policy.AutoAddClasses && !policy.RequireApproval
    default:
        return false // Breaking changes always need approval
    }
}
```

#### Mode 2: Suggested with Approval (Medium Risk)
**For**: Modifying existing properties, deprecating classes

```sql
-- Suggestion queue for human review
CREATE TABLE ontology_suggestions (
    id TEXT PRIMARY KEY,
    ontology_id TEXT NOT NULL,
    current_version TEXT NOT NULL,
    suggestion_type TEXT NOT NULL,
    entity_uri TEXT NOT NULL,
    suggestion_data TEXT NOT NULL, -- JSON of suggestion
    justification TEXT,
    confidence REAL,
    status TEXT DEFAULT 'pending', -- 'pending', 'approved', 'rejected', 'applied'
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    reviewed_by TEXT,
    reviewed_at TIMESTAMP,
    applied_at TIMESTAMP,
    FOREIGN KEY (ontology_id) REFERENCES ontologies(id)
);

CREATE INDEX idx_suggestions_status ON ontology_suggestions(status);
CREATE INDEX idx_suggestions_ontology ON ontology_suggestions(ontology_id, status);
```

**Frontend UI** for reviewing suggestions:

**File**: `mimir-aip-frontend/src/app/ontologies/[id]/suggestions/page.tsx`

```tsx
"use client";
import { useState, useEffect } from "react";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";

interface Suggestion {
  id: string;
  type: string;
  entity_uri: string;
  label: string;
  definition: string;
  justification: string;
  confidence: number;
  status: string;
  created_at: string;
}

export default function SuggestionsPage({ params }: { params: { id: string } }) {
  const [suggestions, setSuggestions] = useState<Suggestion[]>([]);

  useEffect(() => {
    fetchSuggestions();
  }, []);

  async function fetchSuggestions() {
    const res = await fetch(`/api/v1/ontology/${params.id}/suggestions`);
    const data = await res.json();
    setSuggestions(data.suggestions || []);
  }

  async function approveSuggestion(id: string) {
    await fetch(`/api/v1/ontology/suggestions/${id}/approve`, { method: "POST" });
    fetchSuggestions();
  }

  async function rejectSuggestion(id: string) {
    await fetch(`/api/v1/ontology/suggestions/${id}/reject`, { method: "POST" });
    fetchSuggestions();
  }

  return (
    <div>
      <h1 className="text-2xl font-bold text-orange mb-6">
        Ontology Suggestions
      </h1>

      {suggestions.map((sugg) => (
        <Card key={sugg.id} className="bg-navy text-white border-blue p-6 mb-4">
          <div className="flex justify-between items-start mb-4">
            <div>
              <h2 className="text-xl font-bold text-orange">{sugg.label}</h2>
              <Badge className="mt-2">{sugg.type}</Badge>
            </div>
            <div className="text-right">
              <div className="text-sm text-white/60">Confidence</div>
              <div className="text-lg font-bold">
                {(sugg.confidence * 100).toFixed(0)}%
              </div>
            </div>
          </div>

          <div className="space-y-2 mb-4">
            <div>
              <span className="text-white/60">URI:</span>
              <code className="ml-2 text-sm">{sugg.entity_uri}</code>
            </div>
            <div>
              <span className="text-white/60">Definition:</span>
              <p className="ml-2 text-sm">{sugg.definition}</p>
            </div>
            <div>
              <span className="text-white/60">Justification:</span>
              <p className="ml-2 text-sm italic">{sugg.justification}</p>
            </div>
          </div>

          {sugg.status === "pending" && (
            <div className="flex gap-2">
              <Button
                variant="default"
                size="sm"
                onClick={() => approveSuggestion(sugg.id)}
              >
                Approve & Apply
              </Button>
              <Button
                variant="outline"
                size="sm"
                onClick={() => rejectSuggestion(sugg.id)}
              >
                Reject
              </Button>
            </div>
          )}

          {sugg.status === "approved" && (
            <Badge className="bg-green-500">Applied</Badge>
          )}
        </Card>
      ))}
    </div>
  );
}
```

#### Mode 3: Manual Only (High Risk)
**For**: Breaking changes, major refactoring

Human explicitly uploads new ontology version.

---

## 4. Data Migration Strategies

### Automatic Migration Pipeline

When ontology is updated, existing triples need migration.

**File**: `pipelines/Ontology/migration_plugin.go`

```go
package Ontology

// MigrationStrategy defines how to migrate data
type MigrationStrategy string

const (
    StrategyInPlace    MigrationStrategy = "in_place"    // Update triples directly
    StrategyDual       MigrationStrategy = "dual"        // Keep both old and new
    StrategySnapshot   MigrationStrategy = "snapshot"    // Create new graph, keep old
)

// MigrationPlan contains instructions for migrating data
type MigrationPlan struct {
    FromVersion string              `json:"from_version"`
    ToVersion   string              `json:"to_version"`
    Strategy    MigrationStrategy   `json:"strategy"`
    Changes     []MigrationChange   `json:"changes"`
    EstimatedTriples int64          `json:"estimated_triples"`
    EstimatedTime    time.Duration  `json:"estimated_time"`
}

type MigrationChange struct {
    Type        string `json:"type"` // 'rename_property', 'merge_classes', 'add_property'
    OldURI      string `json:"old_uri"`
    NewURI      string `json:"new_uri"`
    SPARQLUpdate string `json:"sparql_update"`
}

// MigrationExecutor executes migration plans
type MigrationExecutor struct {
    tdb2Backend *TDB2Backend
    sqliteDB    *sql.DB
}

// ExecuteMigration runs a migration plan
func (m *MigrationExecutor) ExecuteMigration(ctx context.Context, plan *MigrationPlan) error {
    // Start transaction
    tx, err := m.sqliteDB.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    defer tx.Rollback()
    
    // Record migration start
    migrationID := generateID()
    _, err = tx.Exec(`
        INSERT INTO migration_history (id, ontology_id, from_version, to_version, status, started_at)
        VALUES (?, ?, ?, ?, 'in_progress', CURRENT_TIMESTAMP)
    `, migrationID, plan.FromVersion, plan.ToVersion)
    if err != nil {
        return err
    }
    
    // Execute each change
    for _, change := range plan.Changes {
        if err := m.executeChange(ctx, change); err != nil {
            // Rollback on error
            tx.Exec("UPDATE migration_history SET status = 'failed', error = ?, completed_at = CURRENT_TIMESTAMP WHERE id = ?", 
                err.Error(), migrationID)
            return err
        }
    }
    
    // Mark complete
    _, err = tx.Exec(`
        UPDATE migration_history 
        SET status = 'completed', completed_at = CURRENT_TIMESTAMP 
        WHERE id = ?
    `, migrationID)
    if err != nil {
        return err
    }
    
    return tx.Commit()
}

func (m *MigrationExecutor) executeChange(ctx context.Context, change MigrationChange) error {
    switch change.Type {
    case "rename_property":
        return m.renameProperty(ctx, change)
    case "add_property":
        return m.addPropertyToExisting(ctx, change)
    case "merge_classes":
        return m.mergeClasses(ctx, change)
    case "deprecate_class":
        return m.deprecateClass(ctx, change)
    default:
        return fmt.Errorf("unknown migration type: %s", change.Type)
    }
}

// renameProperty updates all triples using old property to use new property
func (m *MigrationExecutor) renameProperty(ctx context.Context, change MigrationChange) error {
    // SPARQL UPDATE to rename property
    update := fmt.Sprintf(`
        DELETE { ?s <%s> ?o }
        INSERT { ?s <%s> ?o }
        WHERE { ?s <%s> ?o }
    `, change.OldURI, change.NewURI, change.OldURI)
    
    return m.tdb2Backend.ExecuteUpdate(ctx, update)
}

// addPropertyToExisting adds default values for new required properties
func (m *MigrationExecutor) addPropertyToExisting(ctx context.Context, change MigrationChange) error {
    // Example: Add "hasCloudProvider" property to all CloudInstance entities
    update := fmt.Sprintf(`
        INSERT {
            ?instance <%s> "unknown"
        }
        WHERE {
            ?instance rdf:type <%s> .
            FILTER NOT EXISTS { ?instance <%s> ?value }
        }
    `, change.NewURI, change.OldURI, change.NewURI)
    
    return m.tdb2Backend.ExecuteUpdate(ctx, update)
}

// mergeClasses migrates instances from one class to another
func (m *MigrationExecutor) mergeClasses(ctx context.Context, change MigrationChange) error {
    // Example: Merge "PhysicalServer" into "Server"
    update := fmt.Sprintf(`
        DELETE { ?instance rdf:type <%s> }
        INSERT { ?instance rdf:type <%s> }
        WHERE { ?instance rdf:type <%s> }
    `, change.OldURI, change.NewURI, change.OldURI)
    
    return m.tdb2Backend.ExecuteUpdate(ctx, update)
}
```

### Migration Strategies Compared

#### Strategy 1: In-Place Migration
**Use**: Minor changes (property renames, adding optional properties)

```
Before:                    After:
Server --hasIP--> "1.2.3.4"    Server --hasIPAddress--> "1.2.3.4"
```

**Pros**: 
- ✅ No data duplication
- ✅ Immediate effect
- ✅ Simple

**Cons**:
- ❌ No rollback (need backup)
- ❌ Downtime required

#### Strategy 2: Dual Schema (Transition Period)
**Use**: Major changes that need gradual migration

```
Month 1-3: Both schemas exist
Server --hasIP--> "1.2.3.4"          (old)
Server --hasIPAddress--> "1.2.3.4"   (new)

Month 3: Deprecate old schema
Server --hasIPAddress--> "1.2.3.4"   (new only)
```

**Pros**:
- ✅ No breaking changes
- ✅ Gradual migration
- ✅ Easy rollback

**Cons**:
- ❌ Data duplication
- ❌ Synchronization complexity

**Implementation**:
```go
// Maintain both properties during transition
func (m *MigrationExecutor) dualSchemaSync(ctx context.Context, change MigrationChange) error {
    // When old property is set, also set new property
    update := fmt.Sprintf(`
        INSERT {
            ?s <%s> ?o
        }
        WHERE {
            ?s <%s> ?o .
            FILTER NOT EXISTS { ?s <%s> ?value }
        }
    `, change.NewURI, change.OldURI, change.NewURI)
    
    return m.tdb2Backend.ExecuteUpdate(ctx, update)
}
```

#### Strategy 3: Snapshot (Version-Based Graphs)
**Use**: Breaking changes, major refactoring

```
TDB2 Graphs:
- graph:infrastructure:v1.0.0  (original data)
- graph:infrastructure:v2.0.0  (migrated data)

Old queries → v1.0.0 graph (still works)
New queries → v2.0.0 graph
```

**Pros**:
- ✅ Perfect rollback (just switch back)
- ✅ Historical data preserved
- ✅ Zero downtime

**Cons**:
- ❌ Storage doubles
- ❌ More complex queries

---

## 5. Backward Compatibility Layer

### Query Translation

Old queries should still work after ontology changes.

**File**: `pipelines/Ontology/query_translator.go`

```go
package Ontology

// QueryTranslator translates queries from old ontology versions to new
type QueryTranslator struct {
    migrations map[string]*MigrationPlan // version -> plan
}

// TranslateQuery updates a SPARQL query to use current ontology version
func (t *QueryTranslator) TranslateQuery(query string, fromVersion string, toVersion string) (string, error) {
    // Parse SPARQL query
    ast, err := parseSPARQL(query)
    if err != nil {
        return "", err
    }
    
    // Get migration path
    plan := t.getMigrationPlan(fromVersion, toVersion)
    if plan == nil {
        return query, nil // No changes needed
    }
    
    // Apply transformations
    for _, change := range plan.Changes {
        switch change.Type {
        case "rename_property":
            ast = t.renamePropertyInQuery(ast, change.OldURI, change.NewURI)
        case "rename_class":
            ast = t.renameClassInQuery(ast, change.OldURI, change.NewURI)
        }
    }
    
    // Serialize back to SPARQL
    return ast.String(), nil
}

// Example translation
// Old query (v1.0):
//   SELECT ?server WHERE { ?server ont:hasIP ?ip }
// New query (v2.0):
//   SELECT ?server WHERE { ?server ont:hasIPAddress ?ip }
```

### API Version Headers

Support multiple ontology versions simultaneously:

```go
// HandleSPARQLQuery supports version-specific queries
func (s *Server) handleSPARQLQuery(w http.ResponseWriter, r *http.Request) {
    var req struct {
        Query   string `json:"query"`
        Version string `json:"version"` // Optional, defaults to latest
    }
    
    json.NewDecoder(r.Body).Decode(&req)
    
    // Get current ontology version
    currentVersion := s.getCurrentOntologyVersion()
    
    // Translate query if needed
    if req.Version != "" && req.Version != currentVersion {
        translator := NewQueryTranslator()
        translatedQuery, err := translator.TranslateQuery(req.Query, req.Version, currentVersion)
        if err != nil {
            writeErrorResponse(w, "Failed to translate query", http.StatusBadRequest)
            return
        }
        req.Query = translatedQuery
    }
    
    // Execute query
    results, err := s.tdb2Backend.Query(r.Context(), req.Query)
    // ...
}
```

---

## 6. Scheduled Drift Detection & Auto-Update

### Cron Job for Continuous Monitoring

**Configuration** `config.yaml`:

```yaml
ontology:
  drift_detection:
    enabled: true
    schedule: "0 2 * * *"  # 2 AM daily
    sources:
      - type: database
        connection: "postgres://user:pass@host/db"
        tables: ["servers", "containers", "deployments"]
      - type: csv
        path: "/data/exports/*.csv"
      - type: api
        url: "https://api.internal.com/infrastructure"
    
    auto_update:
      enabled: true
      policy:
        auto_add_properties: true
        auto_add_classes: false
        require_approval: true
        min_confidence: 0.85
    
    alerts:
      enabled: true
      webhook: "https://slack.com/webhook/..."
      email: "ops@company.com"
      threshold: "medium"  # Alert on medium+ severity
```

### Scheduled Pipeline

```yaml
# Registered as cron job via scheduler
name: "Daily Ontology Drift Check"
schedule: "0 2 * * *"
steps:
  - name: "Check Database Schema"
    plugin: "Ontology.drift_detector"
    config:
      ontology_id: "infrastructure-v2"
      source_type: "database"
      connection: "${DB_CONNECTION_STRING}"
      tables: ["servers", "containers", "deployments"]
    output: "db_drift"
  
  - name: "Check CSV Exports"
    plugin: "Ontology.drift_detector"
    config:
      ontology_id: "infrastructure-v2"
      source_type: "csv"
      path: "/data/exports/*.csv"
    output: "csv_drift"
  
  - name: "Aggregate Drift Reports"
    plugin: "Data_Processing.aggregate"
    config:
      inputs: ["db_drift", "csv_drift"]
    output: "combined_drift"
  
  - name: "Auto-Apply Low Risk Changes"
    plugin: "Ontology.auto_update"
    config:
      drift_report: "combined_drift"
      policy:
        auto_add_properties: true
        min_confidence: 0.85
    output: "update_result"
  
  - name: "Send Alert if High Severity"
    plugin: "Output.webhook"
    config:
      url: "${SLACK_WEBHOOK}"
      condition: "combined_drift.severity == 'high'"
      message: "High severity schema drift detected! Review suggestions at ${MIMIR_URL}/ontologies/suggestions"
```

---

## 7. Migration Workflow Example

### Real-World Scenario

**Initial State** (Month 1):
```turtle
# infrastructure-v1.0.0.ttl
:Server rdf:type owl:Class .
:hasIP rdf:type owl:DatatypeProperty ;
    rdfs:domain :Server ;
    rdfs:range xsd:string .
:hasOS rdf:type owl:DatatypeProperty ;
    rdfs:domain :Server ;
    rdfs:range xsd:string .
```

**New Data Appears** (Month 3):
```csv
hostname,ip_address,os,kubernetes_node,cloud_provider
web-01,10.0.1.5,Ubuntu,node-1,AWS
```

**Drift Detection Triggers**:
```json
{
  "severity": "medium",
  "unmapped_fields": [
    {
      "field_name": "kubernetes_node",
      "data_type": "string",
      "frequency": 150
    },
    {
      "field_name": "cloud_provider",
      "data_type": "string",
      "frequency": 150
    }
  ],
  "suggestions": [
    {
      "type": "add_property",
      "entity_uri": "http://example.org/ont/hasKubernetesNode",
      "label": "has Kubernetes node",
      "confidence": 0.92,
      "justification": "New field 'kubernetes_node' consistently contains node identifiers"
    },
    {
      "type": "add_property",
      "entity_uri": "http://example.org/ont/hasCloudProvider",
      "label": "has cloud provider",
      "confidence": 0.88,
      "justification": "Field 'cloud_provider' contains AWS, GCP, Azure values"
    }
  ]
}
```

**Auto-Update** (if policy allows):
```turtle
# infrastructure-v1.1.0.ttl (automatically generated)
:Server rdf:type owl:Class .

# Existing properties (unchanged)
:hasIP rdf:type owl:DatatypeProperty ;
    rdfs:domain :Server ;
    rdfs:range xsd:string .
:hasOS rdf:type owl:DatatypeProperty ;
    rdfs:domain :Server ;
    rdfs:range xsd:string .

# NEW: Auto-added properties
:hasKubernetesNode rdf:type owl:DatatypeProperty ;
    rdfs:domain :Server ;
    rdfs:range xsd:string ;
    rdfs:label "has Kubernetes node" ;
    rdfs:comment "Auto-added on 2025-01-15 due to schema drift" .

:hasCloudProvider rdf:type owl:DatatypeProperty ;
    rdfs:domain :Server ;
    rdfs:range xsd:string ;
    rdfs:label "has cloud provider" ;
    rdfs:comment "Auto-added on 2025-01-15 due to schema drift" .
```

**Migration** (automatic):
```sparql
# No migration needed for additive changes
# New properties will be used for future ingestion
# Existing data unchanged
```

**User Notification**:
```
Slack Message:
"🔄 Ontology 'Infrastructure' updated to v1.1.0

Changes:
✅ Added property: hasKubernetesNode (confidence: 92%)
✅ Added property: hasCloudProvider (confidence: 88%)

150 new entities detected with these properties.
Next ingestion will include them.

Review: https://mimir.yourcompany.com/ontologies/infrastructure/changelog
"
```

---

## 8. Frontend UI for Evolution Management

### Ontology Changelog Page

**File**: `mimir-aip-frontend/src/app/ontologies/[id]/changelog/page.tsx`

```tsx
"use client";
import { useState, useEffect } from "react";
import { Card } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";

interface ChangelogEntry {
  version: string;
  change_type: string; // 'minor', 'major', 'patch'
  changes: Change[];
  created_at: string;
}

interface Change {
  type: string;
  entity_uri: string;
  label: string;
  description: string;
}

export default function ChangelogPage({ params }: { params: { id: string } }) {
  const [changelog, setChangelog] = useState<ChangelogEntry[]>([]);

  useEffect(() => {
    fetchChangelog();
  }, []);

  async function fetchChangelog() {
    const res = await fetch(`/api/v1/ontology/${params.id}/changelog`);
    const data = await res.json();
    setChangelog(data.versions || []);
  }

  function getVersionBadgeColor(type: string) {
    switch (type) {
      case "major": return "bg-red-500";
      case "minor": return "bg-blue-500";
      case "patch": return "bg-green-500";
      default: return "bg-gray-500";
    }
  }

  return (
    <div>
      <h1 className="text-2xl font-bold text-orange mb-6">Ontology Changelog</h1>

      <div className="space-y-4">
        {changelog.map((entry) => (
          <Card key={entry.version} className="bg-navy text-white border-blue p-6">
            <div className="flex justify-between items-start mb-4">
              <div>
                <h2 className="text-xl font-bold">Version {entry.version}</h2>
                <p className="text-sm text-white/60">{entry.created_at}</p>
              </div>
              <Badge className={`${getVersionBadgeColor(entry.change_type)} text-white`}>
                {entry.change_type.toUpperCase()}
              </Badge>
            </div>

            <div className="space-y-2">
              {entry.changes.map((change, i) => (
                <div key={i} className="flex items-start gap-3 p-3 bg-blue/10 rounded">
                  <Badge variant="outline" className="shrink-0">
                    {change.type}
                  </Badge>
                  <div className="flex-1">
                    <p className="font-semibold">{change.label}</p>
                    <code className="text-xs text-white/60">{change.entity_uri}</code>
                    {change.description && (
                      <p className="text-sm text-white/80 mt-1">{change.description}</p>
                    )}
                  </div>
                </div>
              ))}
            </div>
          </Card>
        ))}
      </div>
    </div>
  );
}
```

---

## 9. Summary: Evolution Workflow

### Complete Workflow

```
┌─────────────────────────────────────────────────────────────┐
│ 1. Continuous Monitoring (Daily Cron Job)                  │
│    - Check database schemas                                 │
│    - Analyze CSV exports                                    │
│    - Monitor API responses                                  │
└─────────────────────┬───────────────────────────────────────┘
                      │
                      ↓
┌─────────────────────────────────────────────────────────────┐
│ 2. Drift Detection                                          │
│    - Compare data schema with ontology                      │
│    - Identify unmapped fields                               │
│    - Calculate severity                                     │
└─────────────────────┬───────────────────────────────────────┘
                      │
                      ↓
┌─────────────────────────────────────────────────────────────┐
│ 3. LLM Suggestion Generation                                │
│    - Analyze patterns in unmapped fields                    │
│    - Generate ontology change suggestions                   │
│    - Provide confidence scores                              │
└─────────────────────┬───────────────────────────────────────┘
                      │
                      ↓
        ┌─────────────┴─────────────┐
        │                           │
        ↓                           ↓
┌───────────────────┐      ┌───────────────────┐
│ Low Risk          │      │ High Risk         │
│ (Auto-apply)      │      │ (Human Review)    │
│                   │      │                   │
│ - Add properties  │      │ - Rename classes  │
│ - New subclasses  │      │ - Delete props    │
│ - High confidence │      │ - Breaking change │
└────────┬──────────┘      └─────────┬─────────┘
         │                           │
         ↓                           ↓
┌───────────────────┐      ┌───────────────────┐
│ Auto-Update       │      │ Suggestion Queue  │
│ - Create v1.x.0   │      │ - UI review       │
│ - Add to ontology │      │ - Approve/Reject  │
│ - Notify user     │      │                   │
└────────┬──────────┘      └─────────┬─────────┘
         │                           │
         │   ┌───────────────────────┘
         │   │
         ↓   ↓
┌─────────────────────────────────────────────────────────────┐
│ 4. Migration Planning                                       │
│    - Determine strategy (in-place, dual, snapshot)          │
│    - Generate SPARQL UPDATE statements                      │
│    - Estimate time and resource requirements                │
└─────────────────────┬───────────────────────────────────────┘
                      │
                      ↓
┌─────────────────────────────────────────────────────────────┐
│ 5. Migration Execution                                      │
│    - Execute SPARQL updates on TDB2                         │
│    - Update SQLite metadata                                 │
│    - Track progress                                         │
└─────────────────────┬───────────────────────────────────────┘
                      │
                      ↓
┌─────────────────────────────────────────────────────────────┐
│ 6. Verification & Notification                              │
│    - Verify data integrity                                  │
│    - Send notifications (Slack, Email)                      │
│    - Update documentation                                   │
│    - Log changelog                                          │
└─────────────────────────────────────────────────────────────┘
```

---

## 10. Configuration Example

**File**: Update `config.yaml`

```yaml
ontology:
  # Evolution settings
  evolution:
    enabled: true
    
    # Drift detection
    drift_detection:
      enabled: true
      schedule: "0 2 * * *"  # Daily at 2 AM
      alert_threshold: 0.1   # Alert if >10% fields unmapped
      
      sources:
        - name: "Production Database"
          type: database
          connection: "${DATABASE_URL}"
          tables: ["servers", "containers", "deployments", "services"]
        
        - name: "CSV Exports"
          type: csv
          path: "/data/exports/*.csv"
          watch: true  # Monitor for new files
        
        - name: "Infrastructure API"
          type: api
          url: "${INFRA_API_URL}/export"
          schedule: "0 */6 * * *"  # Every 6 hours
    
    # Auto-update policy
    auto_update:
      enabled: true
      policy:
        auto_add_properties: true   # Automatically add new properties
        auto_add_classes: false     # Require approval for new classes
        auto_add_subclasses: true   # Automatically add subclasses
        require_approval: true      # Queue changes for review
        min_confidence: 0.85        # Minimum LLM confidence
      
      # Migration strategy
      migration:
        default_strategy: "dual"    # dual, in_place, snapshot
        transition_period: "30d"    # Keep old schema for 30 days
        backup_before_migration: true
    
    # Notifications
    alerts:
      enabled: true
      channels:
        - type: slack
          webhook: "${SLACK_WEBHOOK}"
          severity: ["medium", "high"]
        
        - type: email
          to: ["ops@company.com", "data-team@company.com"]
          severity: ["high"]
        
        - type: webhook
          url: "${CUSTOM_WEBHOOK}"
          severity: ["medium", "high"]
  
  # Versioning
  versioning:
    enabled: true
    strategy: "semantic"  # semantic, timestamp, incremental
    backup_versions: 10   # Keep last 10 versions
    changelog_format: "markdown"
```

---

## 11. API Endpoints Summary

```go
// Ontology Evolution Endpoints

// GET /api/v1/ontology/{id}/versions
// List all versions of an ontology

// GET /api/v1/ontology/{id}/changelog
// Get changelog between versions

// GET /api/v1/ontology/{id}/suggestions
// Get pending suggestions for review

// POST /api/v1/ontology/suggestions/{suggestionId}/approve
// Approve and apply a suggestion

// POST /api/v1/ontology/suggestions/{suggestionId}/reject
// Reject a suggestion

// GET /api/v1/ontology/{id}/drift
// Get drift detection reports

// POST /api/v1/ontology/{id}/migrate
// Manually trigger migration to new version

// GET /api/v1/ontology/{id}/migration/status
// Check migration status

// POST /api/v1/ontology/{id}/rollback
// Rollback to previous version
```

---

## 12. Key Takeaways

### ✅ Businesses Get:
1. **Automatic Adaptation** - System learns from new data
2. **Zero Manual Work** - Low-risk changes auto-applied
3. **Safety Net** - High-risk changes require approval
4. **No Data Loss** - Multiple migration strategies
5. **Backward Compatible** - Old queries still work
6. **Audit Trail** - Complete changelog of all changes

### ✅ Implementation Priorities:
1. **Phase 1**: Drift detection (alerts only, no auto-update)
2. **Phase 2**: Suggestion system with manual approval
3. **Phase 3**: Auto-apply for low-risk changes
4. **Phase 4**: Automatic migration with dual-schema support

### ✅ Storage Requirements:
- SQLite: +3 tables (ontology_versions, ontology_changes, ontology_suggestions)
- TDB2: Named graphs for versioning (optional)
- Disk: ~20% overhead for dual-schema period

---

**Document Version**: 1.0  
**Status**: Ready for Review  
**Integration**: Extends [ONTOLOGY_PIVOT_REVISED.md](ONTOLOGY_PIVOT_REVISED.md)
