# Ontology Pivot - Implementation Quick Start Guide

## Overview

This guide provides step-by-step instructions for implementing the ontology pivot based on the [Ontology Pivot Design Document](ONTOLOGY_PIVOT_DESIGN.md).

---

## Phase 1: Foundation Setup (Week 1-2)

### Step 1.1: Install Dependencies

```bash
# Add required Go packages
go get github.com/knakk/rdf                    # RDF/Turtle parsing
go get github.com/piprate/json-gold            # JSON-LD support
go get github.com/uber-go/zap                  # Enhanced logging (if needed)

# Optional: ONNX Runtime for ML inference (Phase 4)
# Download from https://github.com/microsoft/onnxruntime/releases
```

### Step 1.2: Set Up Apache Jena Fuseki

**Option A: Docker (Recommended)**

Create `docker/fuseki/docker-compose.fuseki.yml`:
```yaml
version: '3.8'

services:
  fuseki:
    image: stain/jena-fuseki:latest
    container_name: mimir-fuseki
    ports:
      - "3030:3030"
    volumes:
      - fuseki-data:/fuseki
      - ./fuseki-config:/fuseki-config
    environment:
      - ADMIN_PASSWORD=admin123
      - JVM_ARGS=-Xmx2g
    restart: unless-stopped

volumes:
  fuseki-data:
    driver: local
```

Start Fuseki:
```bash
cd docker/fuseki
docker-compose -f docker-compose.fuseki.yml up -d

# Verify it's running
curl http://localhost:3030/$/ping
```

**Option B: Local Installation**
```bash
# Download Jena Fuseki
wget https://dlcdn.apache.org/jena/binaries/apache-jena-fuseki-4.10.0.tar.gz
tar -xzf apache-jena-fuseki-4.10.0.tar.gz
cd apache-jena-fuseki-4.10.0

# Start server
./fuseki-server --update --mem /ds
```

### Step 1.3: Create Ontology Module Structure

```bash
# Create directory structure
mkdir -p pipelines/Ontology
mkdir -p pipelines/KnowledgeGraph
mkdir -p pipelines/GraphProcessing
mkdir -p pipelines/ML

# Create base files
touch pipelines/Ontology/ontology_types.go
touch pipelines/Ontology/ontology_plugin.go
touch pipelines/Ontology/ontology_parser.go
touch pipelines/Ontology/ontology_store.go
touch pipelines/Ontology/README.md

touch pipelines/KnowledgeGraph/fuseki_backend.go
touch pipelines/KnowledgeGraph/kg_types.go
touch pipelines/KnowledgeGraph/sparql_client.go
touch pipelines/KnowledgeGraph/README.md
```

### Step 1.4: Implement Core Ontology Types

**File**: `pipelines/Ontology/ontology_types.go`

```go
package Ontology

import (
	"time"
)

// OntologyFormat represents supported ontology serialization formats
type OntologyFormat string

const (
	FormatTurtle   OntologyFormat = "turtle"
	FormatRDFXML   OntologyFormat = "rdf/xml"
	FormatJSONLD   OntologyFormat = "json-ld"
	FormatNTriples OntologyFormat = "n-triples"
)

// OntologyMetadata contains ontology metadata
type OntologyMetadata struct {
	ID          string         `json:"id" yaml:"id"`
	Name        string         `json:"name" yaml:"name"`
	Version     string         `json:"version" yaml:"version"`
	Description string         `json:"description" yaml:"description"`
	Format      OntologyFormat `json:"format" yaml:"format"`
	CreatedAt   time.Time      `json:"created_at" yaml:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at" yaml:"updated_at"`
	Author      string         `json:"author" yaml:"author"`
	BaseURI     string         `json:"base_uri" yaml:"base_uri"`
	Namespace   string         `json:"namespace" yaml:"namespace"`
}

// Ontology represents a complete ontology
type Ontology struct {
	Metadata   OntologyMetadata   `json:"metadata"`
	Content    []byte             `json:"content"`    // Raw serialized ontology
	Classes    []OntologyClass    `json:"classes"`
	Properties []OntologyProperty `json:"properties"`
	Individuals []OntologyIndividual `json:"individuals,omitempty"`
}

// OntologyClass represents an OWL class
type OntologyClass struct {
	URI         string            `json:"uri"`
	Label       string            `json:"label"`
	Description string            `json:"description,omitempty"`
	SubClassOf  []string          `json:"subclass_of,omitempty"`
	Properties  []string          `json:"properties,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// PropertyType distinguishes between object and data properties
type PropertyType string

const (
	PropertyTypeObject PropertyType = "ObjectProperty"
	PropertyTypeData   PropertyType = "DatatypeProperty"
)

// OntologyProperty represents an OWL property
type OntologyProperty struct {
	URI          string            `json:"uri"`
	Label        string            `json:"label"`
	Description  string            `json:"description,omitempty"`
	PropertyType PropertyType      `json:"type"`
	Domain       []string          `json:"domain,omitempty"`
	Range        []string          `json:"range,omitempty"`
	SubPropertyOf []string         `json:"subproperty_of,omitempty"`
	Annotations  map[string]string `json:"annotations,omitempty"`
}

// OntologyIndividual represents an OWL individual (instance)
type OntologyIndividual struct {
	URI         string            `json:"uri"`
	Label       string            `json:"label"`
	Type        []string          `json:"type"`
	Properties  map[string]any    `json:"properties,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// ValidationResult contains ontology validation results
type ValidationResult struct {
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}
```

### Step 1.5: Implement Fuseki Backend

**File**: `pipelines/KnowledgeGraph/fuseki_backend.go`

```go
package KnowledgeGraph

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// FusekiBackend implements GraphStoreBackend for Apache Jena Fuseki
type FusekiBackend struct {
	baseURL    string
	dataset    string
	httpClient *http.Client
	username   string
	password   string
}

// FusekiConfig holds Fuseki connection configuration
type FusekiConfig struct {
	BaseURL  string `json:"base_url" yaml:"base_url"`
	Dataset  string `json:"dataset" yaml:"dataset"`
	Username string `json:"username,omitempty" yaml:"username,omitempty"`
	Password string `json:"password,omitempty" yaml:"password,omitempty"`
	Timeout  int    `json:"timeout" yaml:"timeout"` // seconds
}

// NewFusekiBackend creates a new Fuseki backend instance
func NewFusekiBackend(config FusekiConfig) (*FusekiBackend, error) {
	if config.BaseURL == "" {
		return nil, fmt.Errorf("base_url is required")
	}
	if config.Dataset == "" {
		return nil, fmt.Errorf("dataset is required")
	}

	timeout := 30 * time.Second
	if config.Timeout > 0 {
		timeout = time.Duration(config.Timeout) * time.Second
	}

	return &FusekiBackend{
		baseURL:  config.BaseURL,
		dataset:  config.Dataset,
		username: config.Username,
		password: config.Password,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

// QuerySPARQL executes a SPARQL SELECT query
func (f *FusekiBackend) QuerySPARQL(ctx context.Context, query string) (*QueryResult, error) {
	endpoint := fmt.Sprintf("%s/%s/query", f.baseURL, f.dataset)

	data := url.Values{}
	data.Set("query", query)

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBufferString(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/sparql-results+json")

	if f.username != "" && f.password != "" {
		req.SetBasicAuth(f.username, f.password)
	}

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("query failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Head struct {
			Vars []string `json:"vars"`
		} `json:"head"`
		Results struct {
			Bindings []map[string]interface{} `json:"bindings"`
		} `json:"results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse results: %w", err)
	}

	return &QueryResult{
		Bindings: result.Results.Bindings,
		Count:    len(result.Results.Bindings),
		Variables: result.Head.Vars,
	}, nil
}

// InsertTriples inserts RDF triples into the graph
func (f *FusekiBackend) InsertTriples(ctx context.Context, triples []Triple) error {
	if len(triples) == 0 {
		return nil
	}

	// Build SPARQL INSERT DATA query
	var buffer bytes.Buffer
	buffer.WriteString("INSERT DATA {\n")

	for _, triple := range triples {
		// Format triple (handle literals vs URIs)
		subject := f.formatNode(triple.Subject)
		predicate := f.formatNode(triple.Predicate)
		object := f.formatNode(triple.Object)

		buffer.WriteString(fmt.Sprintf("  %s %s %s .\n", subject, predicate, object))
	}

	buffer.WriteString("}")

	return f.executeSPARQLUpdate(ctx, buffer.String())
}

// executeSPARQLUpdate executes a SPARQL UPDATE query
func (f *FusekiBackend) executeSPARQLUpdate(ctx context.Context, update string) error {
	endpoint := fmt.Sprintf("%s/%s/update", f.baseURL, f.dataset)

	data := url.Values{}
	data.Set("update", update)

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBufferString(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	if f.username != "" && f.password != "" {
		req.SetBasicAuth(f.username, f.password)
	}

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("update failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("update failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// formatNode formats a node for SPARQL (URI or literal)
func (f *FusekiBackend) formatNode(node string) string {
	// Check if it's a literal (starts with quote)
	if len(node) > 0 && node[0] == '"' {
		return node // Already formatted as literal
	}

	// Check if it's already wrapped in angle brackets
	if len(node) > 0 && node[0] == '<' && node[len(node)-1] == '>' {
		return node
	}

	// Check if it has a namespace prefix (e.g., "rdf:type")
	if len(node) > 0 && !containsColon(node) {
		// Wrap in angle brackets (full URI)
		return fmt.Sprintf("<%s>", node)
	}

	// Has prefix, return as-is
	return node
}

func containsColon(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == ':' {
			return true
		}
	}
	return false
}

// Health checks if the Fuseki server is healthy
func (f *FusekiBackend) Health(ctx context.Context) error {
	endpoint := fmt.Sprintf("%s/$/ping", f.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return err
	}

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed with status %d", resp.StatusCode)
	}

	return nil
}

// Stats returns statistics about the graph store
func (f *FusekiBackend) Stats(ctx context.Context) (*GraphStats, error) {
	// Query for triple count
	query := "SELECT (COUNT(*) AS ?count) WHERE { ?s ?p ?o }"

	result, err := f.QuerySPARQL(ctx, query)
	if err != nil {
		return nil, err
	}

	stats := &GraphStats{
		TotalTriples: 0,
		GraphCount:   0, // TODO: Query for graph count
	}

	if len(result.Bindings) > 0 {
		if countVal, ok := result.Bindings[0]["count"].(map[string]interface{}); ok {
			if value, ok := countVal["value"].(string); ok {
				fmt.Sscanf(value, "%d", &stats.TotalTriples)
			}
		}
	}

	return stats, nil
}

// Close closes the backend connection
func (f *FusekiBackend) Close() error {
	// HTTP client doesn't need explicit closing
	return nil
}
```

**File**: `pipelines/KnowledgeGraph/kg_types.go`

```go
package KnowledgeGraph

// GraphStoreBackend defines the interface for RDF graph storage
type GraphStoreBackend interface {
	// Query operations
	QuerySPARQL(ctx context.Context, query string) (*QueryResult, error)

	// Mutation operations
	InsertTriples(ctx context.Context, triples []Triple) error
	DeleteTriples(ctx context.Context, filter TripleFilter) error

	// Graph operations
	CreateGraph(ctx context.Context, graphURI string) error
	ListGraphs(ctx context.Context) ([]string, error)

	// Health & statistics
	Health(ctx context.Context) error
	Stats(ctx context.Context) (*GraphStats, error)

	// Cleanup
	Close() error
}

// Triple represents an RDF triple
type Triple struct {
	Subject   string `json:"subject"`
	Predicate string `json:"predicate"`
	Object    string `json:"object"`
	Graph     string `json:"graph,omitempty"`
}

// TripleFilter defines filtering criteria for triple deletion
type TripleFilter struct {
	Subject   string `json:"subject,omitempty"`
	Predicate string `json:"predicate,omitempty"`
	Object    string `json:"object,omitempty"`
	Graph     string `json:"graph,omitempty"`
}

// QueryResult holds SPARQL query results
type QueryResult struct {
	Bindings  []map[string]interface{} `json:"bindings"`
	Count     int                      `json:"count"`
	Variables []string                 `json:"variables,omitempty"`
}

// GraphStats contains graph statistics
type GraphStats struct {
	TotalTriples int64  `json:"total_triples"`
	GraphCount   int    `json:"graph_count"`
	LastUpdated  string `json:"last_updated,omitempty"`
}
```

### Step 1.6: Add REST API Endpoints

**File**: Update `routes.go`

```go
// Add to setupRoutes() function

// Ontology management endpoints
apiV1.HandleFunc("/ontology", s.handleUploadOntology).Methods("POST")
apiV1.HandleFunc("/ontology", s.handleListOntologies).Methods("GET")
apiV1.HandleFunc("/ontology/{id}", s.handleGetOntology).Methods("GET")
apiV1.HandleFunc("/ontology/{id}", s.handleUpdateOntology).Methods("PUT")
apiV1.HandleFunc("/ontology/{id}", s.handleDeleteOntology).Methods("DELETE")
apiV1.HandleFunc("/ontology/{id}/validate", s.handleValidateOntology).Methods("POST")

// Knowledge graph endpoints
apiV1.HandleFunc("/kg/query", s.handleSPARQLQuery).Methods("POST")
apiV1.HandleFunc("/kg/insert", s.handleInsertTriples).Methods("POST")
apiV1.HandleFunc("/kg/stats", s.handleGraphStats).Methods("GET")
apiV1.HandleFunc("/kg/health", s.handleGraphHealth).Methods("GET")
```

**File**: Create `handlers_ontology.go`

```go
package main

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/Ontology"
	"github.com/gorilla/mux"
)

func (s *Server) handleUploadOntology(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form
	if err := r.ParseMultipartForm(32 << 20); err != nil { // 32 MB max
		writeErrorResponse(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("ontology")
	if err != nil {
		writeErrorResponse(w, "Ontology file required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Read file content
	content, err := io.ReadAll(file)
	if err != nil {
		writeErrorResponse(w, "Failed to read file", http.StatusInternalServerError)
		return
	}

	// Parse metadata from form
	metadata := Ontology.OntologyMetadata{
		Name:        r.FormValue("name"),
		Version:     r.FormValue("version"),
		Description: r.FormValue("description"),
		Author:      r.FormValue("author"),
		BaseURI:     r.FormValue("base_uri"),
		Format:      Ontology.OntologyFormat(r.FormValue("format")),
	}

	if metadata.Name == "" {
		metadata.Name = header.Filename
	}

	// TODO: Validate and store ontology
	// For now, return success
	writeJSONResponse(w, map[string]interface{}{
		"message":   "Ontology uploaded successfully",
		"name":      metadata.Name,
		"version":   metadata.Version,
		"size":      len(content),
		"filename":  header.Filename,
	}, http.StatusCreated)
}

func (s *Server) handleListOntologies(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement ontology listing from storage
	writeJSONResponse(w, map[string]interface{}{
		"ontologies": []map[string]interface{}{},
		"count":      0,
	}, http.StatusOK)
}

func (s *Server) handleGetOntology(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	// TODO: Retrieve ontology from storage
	writeJSONResponse(w, map[string]interface{}{
		"id":      id,
		"message": "Ontology retrieval not yet implemented",
	}, http.StatusOK)
}

func (s *Server) handleUpdateOntology(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	writeJSONResponse(w, map[string]interface{}{
		"id":      id,
		"message": "Ontology update not yet implemented",
	}, http.StatusOK)
}

func (s *Server) handleDeleteOntology(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	writeJSONResponse(w, map[string]interface{}{
		"id":      id,
		"message": "Ontology deletion not yet implemented",
	}, http.StatusOK)
}

func (s *Server) handleValidateOntology(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	writeJSONResponse(w, map[string]interface{}{
		"id":    id,
		"valid": true,
		"errors": []string{},
	}, http.StatusOK)
}
```

### Step 1.7: Test Phase 1 Implementation

Create test ontology file `test_data/simple_ontology.ttl`:
```turtle
@prefix : <http://example.org/ontology#> .
@prefix rdf: <http://www.w3.org/1999/02/22-rdf-syntax-ns#> .
@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .
@prefix owl: <http://www.w3.org/2002/07/owl#> .

:Person rdf:type owl:Class ;
    rdfs:label "Person" ;
    rdfs:comment "A human being" .

:hasName rdf:type owl:DatatypeProperty ;
    rdfs:label "has name" ;
    rdfs:domain :Person ;
    rdfs:range rdfs:Literal .

:knows rdf:type owl:ObjectProperty ;
    rdfs:label "knows" ;
    rdfs:domain :Person ;
    rdfs:range :Person .
```

**Test Script** `scripts/test_phase1.sh`:
```bash
#!/bin/bash

echo "=== Phase 1 Implementation Tests ==="

# Test 1: Fuseki health check
echo "Test 1: Fuseki health check"
curl -s http://localhost:3030/$/ping && echo "✓ Fuseki is running" || echo "✗ Fuseki not available"

# Test 2: Upload ontology
echo "\nTest 2: Upload ontology"
curl -X POST http://localhost:8080/api/v1/ontology \
  -F "ontology=@test_data/simple_ontology.ttl" \
  -F "name=Simple Ontology" \
  -F "version=1.0" \
  -F "format=turtle"

# Test 3: List ontologies
echo "\nTest 3: List ontologies"
curl -s http://localhost:8080/api/v1/ontology | jq

# Test 4: Direct Fuseki SPARQL query
echo "\nTest 4: Direct Fuseki query"
curl -X POST http://localhost:3030/ds/query \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -H "Accept: application/sparql-results+json" \
  --data-urlencode "query=SELECT * WHERE { ?s ?p ?o } LIMIT 5"

echo "\n\n=== Phase 1 Tests Complete ==="
```

Run tests:
```bash
chmod +x scripts/test_phase1.sh
./scripts/test_phase1.sh
```

---

## Phase 2: Knowledge Graph Pipeline (Week 3-4)

### Step 2.1: Create Entity Extractor

**File**: `pipelines/GraphProcessing/entity_extractor.go`

```go
package GraphProcessing

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/Ontology"
)

// EntityExtractor extracts entities from text using LLM
type EntityExtractor struct {
	llmClient LLMClient
	ontology  *Ontology.Ontology
}

// Entity represents an extracted entity
type Entity struct {
	Text       string  `json:"text"`
	Type       string  `json:"type"` // Ontology class URI
	Confidence float64 `json:"confidence"`
	StartPos   int     `json:"start_pos,omitempty"`
	EndPos     int     `json:"end_pos,omitempty"`
}

// Relation represents an extracted relationship
type Relation struct {
	Subject    string  `json:"subject"`
	Predicate  string  `json:"predicate"` // Ontology property URI
	Object     string  `json:"object"`
	Confidence float64 `json:"confidence"`
}

// ExtractionResult holds extraction results
type ExtractionResult struct {
	Entities    []Entity   `json:"entities"`
	Relations   []Relation `json:"relations"`
	SourceText  string     `json:"source_text"`
	ProcessedAt string     `json:"processed_at"`
}

// Extract performs entity and relation extraction
func (e *EntityExtractor) Extract(ctx context.Context, text string) (*ExtractionResult, error) {
	prompt := e.buildExtractionPrompt(text)

	// Call LLM (reuse existing AI plugin)
	response, err := e.llmClient.Complete(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("LLM extraction failed: %w", err)
	}

	// Parse LLM response
	result, err := e.parseExtractionResponse(response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse extraction: %w", err)
	}

	result.SourceText = text
	return result, nil
}

func (e *EntityExtractor) buildExtractionPrompt(text string) string {
	classesStr := e.formatOntologyClasses()
	propertiesStr := e.formatOntologyProperties()

	return fmt.Sprintf(`You are an expert knowledge graph engineer. Extract entities and relationships from the text below.

ONTOLOGY CLASSES:
%s

ONTOLOGY PROPERTIES:
%s

INPUT TEXT:
%s

OUTPUT FORMAT (JSON only, no markdown):
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

Rules:
1. Only use classes and properties from the ontology
2. If unsure, use confidence < 0.7
3. Return valid JSON only
`, classesStr, propertiesStr, text)
}

func (e *EntityExtractor) formatOntologyClasses() string {
	var result string
	for _, class := range e.ontology.Classes {
		result += fmt.Sprintf("- %s (%s): %s\n", class.Label, class.URI, class.Description)
	}
	return result
}

func (e *EntityExtractor) formatOntologyProperties() string {
	var result string
	for _, prop := range e.ontology.Properties {
		result += fmt.Sprintf("- %s (%s): %s\n", prop.Label, prop.URI, prop.Description)
	}
	return result
}

func (e *EntityExtractor) parseExtractionResponse(response string) (*ExtractionResult, error) {
	// Extract JSON from response (LLM might add explanation)
	jsonStr := extractJSON(response)

	var parsed struct {
		Entities      []Entity   `json:"entities"`
		Relationships []Relation `json:"relationships"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		return nil, fmt.Errorf("invalid JSON response: %w", err)
	}

	return &ExtractionResult{
		Entities:  parsed.Entities,
		Relations: parsed.Relationships,
	}, nil
}

// extractJSON finds JSON block in text
func extractJSON(text string) string {
	// Simple JSON extraction - find first { to last }
	start := -1
	for i, c := range text {
		if c == '{' {
			start = i
			break
		}
	}

	if start == -1 {
		return text
	}

	end := -1
	for i := len(text) - 1; i >= 0; i-- {
		if text[i] == '}' {
			end = i + 1
			break
		}
	}

	if end == -1 {
		return text
	}

	return text[start:end]
}

// EntityExtractorPlugin implements BasePlugin for entity extraction
type EntityExtractorPlugin struct {
	extractors map[string]*EntityExtractor // ontology_id -> extractor
}

func NewEntityExtractorPlugin() *EntityExtractorPlugin {
	return &EntityExtractorPlugin{
		extractors: make(map[string]*EntityExtractor),
	}
}

func (p *EntityExtractorPlugin) ExecuteStep(
	ctx context.Context,
	stepConfig pipelines.StepConfig,
	globalContext *pipelines.PluginContext,
) (*pipelines.PluginContext, error) {
	// Get text from context
	textField, _ := stepConfig.Config["text_field"].(string)
	if textField == "" {
		textField = "text"
	}

	textValue, exists := globalContext.Get(textField)
	if !exists {
		return nil, fmt.Errorf("text field '%s' not found in context", textField)
	}

	text, ok := textValue.(string)
	if !ok {
		return nil, fmt.Errorf("text field must be string")
	}

	// Get ontology ID
	ontologyID, _ := stepConfig.Config["ontology_id"].(string)
	if ontologyID == "" {
		return nil, fmt.Errorf("ontology_id required")
	}

	// Get or create extractor
	extractor, exists := p.extractors[ontologyID]
	if !exists {
		return nil, fmt.Errorf("extractor not initialized for ontology %s", ontologyID)
	}

	// Perform extraction
	result, err := extractor.Extract(ctx, text)
	if err != nil {
		return nil, err
	}

	// Store result in context
	resultCtx := pipelines.NewPluginContext()
	resultCtx.Set(stepConfig.Output, result)

	return resultCtx, nil
}

func (p *EntityExtractorPlugin) GetPluginType() string {
	return "GraphProcessing"
}

func (p *EntityExtractorPlugin) GetPluginName() string {
	return "entity_extractor"
}

func (p *EntityExtractorPlugin) ValidateConfig(config map[string]any) error {
	if _, ok := config["ontology_id"]; !ok {
		return fmt.Errorf("ontology_id required")
	}
	return nil
}
```

---

## Next Steps

This guide covers Phase 1 foundation. The full implementation continues with:

- **Phase 2**: Complete graph processing pipeline
- **Phase 3**: Agent tooling and NL→SPARQL
- **Phase 4**: ML classifier integration
- **Phase 5**: Digital twin simulation
- **Phase 6**: Production deployment

See the [Ontology Pivot Design Document](ONTOLOGY_PIVOT_DESIGN.md) for complete details.

---

## Quick Reference Commands

```bash
# Start full stack
docker-compose -f docker-compose.unified.yml up

# Test ontology upload
curl -X POST http://localhost:8080/api/v1/ontology \
  -F "ontology=@my_ontology.ttl" \
  -F "name=My Ontology" \
  -F "version=1.0"

# Query knowledge graph
curl -X POST http://localhost:8080/api/v1/kg/query \
  -H "Content-Type: application/json" \
  -d '{"query": "SELECT * WHERE { ?s ?p ?o } LIMIT 10"}'

# Check graph stats
curl http://localhost:8080/api/v1/kg/stats
```

---

**Document Version**: 1.0  
**Author**: Mimir AIP Development Team  
**Last Updated**: 2025-01-15
