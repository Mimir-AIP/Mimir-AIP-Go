# Schema Inference Engine

## Overview

The schema inference engine automatically analyzes structured data and generates schemas with type information, constraints, and relationships. It supports multiple advanced features including AI-enhanced inference and automatic foreign key detection.

## Features

### Core Capabilities
- **Automatic Type Detection**: Infers data types (integer, float, string, boolean, date) from sample data
- **Constraint Detection**: Identifies primary keys, unique columns, required fields
- **Ontology Generation**: Creates OWL ontologies from inferred schemas
- **Relationship Detection**: Discovers relationships between columns

### Advanced Features
- **[AI/LLM Fallback](./README_AI_FALLBACK.md)**: Enhanced type detection using Large Language Models
  - Semantic type detection (email, currency, phone, etc.)
  - Constraint inference with patterns and ranges
  - Low-confidence fallback support
  
- **[Foreign Key Detection](./FK_DETECTION.md)**: Automatic FK relationship discovery
  - Name pattern analysis (*_id, *_ref, fk_*)
  - Cardinality-based detection
  - Value overlap analysis with referential integrity metrics
  - Configurable confidence thresholds

## Quick Start

### Basic Schema Inference

```go
import "github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/Ontology/schema_inference"

// Create engine
config := schema_inference.InferenceConfig{
    SampleSize:        100,
    EnableConstraints: true,
}
engine := schema_inference.NewSchemaInferenceEngine(config)

// Analyze data
data := []map[string]interface{}{
    {"id": 1, "name": "Alice", "age": 30},
    {"id": 2, "name": "Bob", "age": 25},
}

schema, err := engine.InferSchema(data, "users")
```

### With Foreign Key Detection

```go
config := schema_inference.InferenceConfig{
    SampleSize:        100,
    EnableConstraints: true,
    EnableFKDetection: true,   // Enable FK detection
    FKMinConfidence:   0.8,    // 80% confidence threshold
}
engine := schema_inference.NewSchemaInferenceEngine(config)

schema, _ := engine.InferSchema(data, "orders")

// Access detected foreign keys
for _, fk := range schema.ForeignKeys {
    fmt.Printf("%s -> %s (%.1f%% integrity)\n",
        fk.SourceColumn, fk.TargetColumn,
        fk.ReferentialIntegrity*100)
}
```

### With AI Enhancement

```go
import "github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/AI"

// Create LLM client
llmClient := AI.NewOpenAIClient(apiKey, "gpt-4")

config := schema_inference.InferenceConfig{
    SampleSize:          100,
    ConfidenceThreshold: 0.8,      // Trigger AI if < 80%
    EnableAIFallback:    true,      // Enable AI
    AIConfidenceBoost:   0.15,
}

engine := schema_inference.NewSchemaInferenceEngineWithLLM(config, llmClient)
schema, _ := engine.InferSchema(data, "products")
```

## Configuration Options

```go
type InferenceConfig struct {
    SampleSize          int     // Number of rows to analyze (default: 100)
    ConfidenceThreshold float64 // Min confidence for types (default: 0.8)
    EnableRelationships bool    // Detect relationships (default: false)
    EnableConstraints   bool    // Infer constraints (default: false)
    EnableAIFallback    bool    // Use AI for low confidence (default: false)
    AIConfidenceBoost   float64 // AI confidence boost (default: 0.15)
    EnableFKDetection   bool    // Detect foreign keys (default: false)
    FKMinConfidence     float64 // Min FK confidence (default: 0.8)
}
```

## Schema Output

### DataSchema Structure

```go
type DataSchema struct {
    Name          string
    Description   string
    Columns       []ColumnSchema
    Relationships []RelationshipSchema
    ForeignKeys   []ForeignKeyRelationship
    Metadata      map[string]interface{}
    InferredAt    time.Time
}
```

### Column Information

```go
type ColumnSchema struct {
    Name               string
    DataType           string  // "integer", "float", "string", "boolean", "date"
    OntologyType       string  // XSD type (xsd:integer, xsd:string, etc.)
    IsPrimaryKey       bool
    IsForeignKey       bool
    IsRequired         bool
    IsUnique           bool
    SampleValues       []interface{}
    Constraints        map[string]interface{}
    Description        string
    Cardinality        int     // Unique value count
    CardinalityPercent float64 // As % of total rows
    FKMetadata         *ForeignKeyMetadata
}
```

### Foreign Key Information

```go
type ForeignKeyRelationship struct {
    SourceColumn         string
    TargetColumn         string
    Confidence           float64   // Overall confidence
    ReferentialIntegrity float64   // % of matching values
    MatchedValues        int
    TotalValues          int
    DetectionMethods     []string  // Methods used
}
```

## Ontology Generation

Generate OWL ontologies from inferred schemas:

```go
ontologyConfig := schema_inference.OntologyConfig{
    BaseURI:         "http://example.com/ontology/",
    OntologyPrefix:  "ex",
    IncludeMetadata: true,
    ClassNaming:     "pascal",
    PropertyNaming:  "camel",
}

generator := schema_inference.NewOntologyGenerator(ontologyConfig)
ontology, _ := generator.GenerateOntology(schema)

// Ontology includes:
// - Classes for entity types
// - DatatypeProperties for column values
// - ObjectProperties for FK relationships
fmt.Println(ontology.Content)  // Turtle format OWL
```

## Examples

See comprehensive examples in:
- `examples/fk_detection_example.go.txt` - Foreign key detection
- `examples/llm_integration_example.go` - AI-enhanced inference

## Testing

Run the full test suite:

```bash
# All tests
go test -v ./pipelines/Ontology/schema_inference

# FK detection tests only
go test -v ./pipelines/Ontology/schema_inference -run "ForeignKey"

# AI fallback tests only
go test -v ./pipelines/Ontology/schema_inference -run "AI"
```

## Documentation

- **[FK_DETECTION.md](./FK_DETECTION.md)** - Foreign key detection guide
- **[README_AI_FALLBACK.md](./README_AI_FALLBACK.md)** - AI enhancement guide
- **[FK_IMPLEMENTATION_SUMMARY.md](./FK_IMPLEMENTATION_SUMMARY.md)** - Implementation details
- **[IMPLEMENTATION_SUMMARY.md](./IMPLEMENTATION_SUMMARY.md)** - General implementation notes

## Performance

- **Time Complexity**: O(n*m) for n columns and m rows (sample size limited)
- **Space Complexity**: O(n*k) for k unique values per column
- **Optimization**: Configurable sample size balances accuracy vs speed
- **Recommended**: Use sample size of 100-1000 rows for best results

## Limitations

1. **Sample-Based**: Analysis limited to configured sample size
2. **Single-Table FKs**: Value overlap works best within single dataset
3. **Naming Conventions**: FK detection assumes common naming patterns
4. **AI Dependency**: AI features require LLM client configuration

## Future Enhancements

- Composite foreign key support
- Cross-table relationship analysis
- ML-based pattern recognition
- Schema evolution tracking
- Performance optimizations for large datasets
