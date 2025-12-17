# Foreign Key Detection in Schema Inference

## Overview

The schema inference engine now includes automatic foreign key (FK) detection capabilities. This feature analyzes column names, value distributions, and cross-column value overlaps to automatically identify relationships between columns.

## Features

### 1. Multiple Detection Methods

The FK detection engine uses three complementary methods:

#### Name Pattern Analysis
Detects FKs based on common naming conventions:
- `*_id` pattern (e.g., `user_id`, `product_id`)
- `*_ref` pattern (e.g., `order_ref`)
- `fk_*` pattern (e.g., `fk_customer`)
- `*_fk_*` pattern (e.g., `customer_fk_id`)

**Confidence Scores:**
- `*_id` with matching PK: 0.9
- `*_id` without PK: 0.7
- `*_ref`: 0.7
- `fk_*`: 0.8

#### Cardinality Analysis
Analyzes the uniqueness of values:
- FKs typically have cardinality between 5% and 80% of total rows
- Higher confidence (0.7) for cardinality between 20-60%
- Lower confidence (0.5) for edge cases (5-20% or 60-80%)
- Too low (<5%) suggests enum/category
- Too high (>80%) suggests unique identifier

#### Value Overlap Analysis
Compares actual values across columns:
- Calculates percentage of source values that exist in target column
- Requires minimum 70% overlap for FK consideration
- Provides referential integrity percentage
- High overlap (>90%) = high confidence

### 2. Configuration Options

```go
config := schema_inference.InferenceConfig{
    EnableFKDetection: true,    // Enable FK detection (default: false)
    FKMinConfidence:   0.8,     // Minimum confidence threshold (default: 0.8)
    EnableConstraints: true,    // Required for FK detection
}
```

**Configuration Parameters:**

- `EnableFKDetection` (bool): Enables/disables FK detection
- `FKMinConfidence` (float64): Minimum confidence score (0.0-1.0) for FK relationships
  - Default: 0.8 (80%)
  - Lower values detect more FKs but may include false positives
  - Higher values are more conservative but may miss some FKs

### 3. FK Detection Output

#### Column Metadata
Each detected FK column includes:

```go
type ColumnSchema struct {
    Name               string
    IsForeignKey       bool
    Cardinality        int      // Number of unique values
    CardinalityPercent float64  // Cardinality as % of total rows
    FKMetadata         *ForeignKeyMetadata
}

type ForeignKeyMetadata struct {
    ReferencedColumn string
    Confidence       float64
    DetectionMethod  string  // e.g., "name_pattern,value_overlap"
}
```

#### FK Relationships
Detailed relationship information:

```go
type ForeignKeyRelationship struct {
    SourceColumn         string
    TargetColumn         string
    Confidence           float64  // Overall confidence score
    ReferentialIntegrity float64  // % of values that match (0.0-1.0)
    MatchedValues        int      // Count of matching values
    TotalValues          int      // Total source values
    DetectionMethods     []string // Methods used
}
```

## Usage Examples

### Basic Usage

```go
// Create engine with FK detection enabled
config := schema_inference.InferenceConfig{
    SampleSize:        100,
    EnableConstraints: true,
    EnableFKDetection: true,
    FKMinConfidence:   0.8,
}

engine := schema_inference.NewSchemaInferenceEngine(config)

// Analyze data
data := []map[string]interface{}{
    {"order_id": 1, "user_id": 10, "amount": 50.0},
    {"order_id": 2, "user_id": 11, "amount": 75.0},
    {"order_id": 3, "user_id": 10, "amount": 30.0},
}

schema, err := engine.InferSchema(data, "orders")
if err != nil {
    log.Fatal(err)
}

// Access FK information
for _, col := range schema.Columns {
    if col.IsForeignKey {
        fmt.Printf("FK Column: %s\n", col.Name)
        fmt.Printf("  Cardinality: %d (%.1f%%)\n", 
            col.Cardinality, col.CardinalityPercent*100)
        
        if col.FKMetadata != nil {
            fmt.Printf("  References: %s\n", col.FKMetadata.ReferencedColumn)
            fmt.Printf("  Confidence: %.2f\n", col.FKMetadata.Confidence)
        }
    }
}

// Access FK relationships
for _, fk := range schema.ForeignKeys {
    fmt.Printf("FK Relationship: %s -> %s\n", fk.SourceColumn, fk.TargetColumn)
    fmt.Printf("  Integrity: %.1f%% (%d/%d matches)\n",
        fk.ReferentialIntegrity*100, fk.MatchedValues, fk.TotalValues)
}
```

### Integration with Ontology Generation

FK relationships are automatically converted to OWL ObjectProperties:

```go
// Generate ontology from schema with FKs
ontologyConfig := schema_inference.OntologyConfig{
    BaseURI:        "http://example.com/ontology/",
    OntologyPrefix: "ex",
}

generator := schema_inference.NewOntologyGenerator(ontologyConfig)
ontology, err := generator.GenerateOntology(schema)

// FK relationships become ObjectProperties
for _, prop := range ontology.Properties {
    if prop.Type == "object" {
        fmt.Printf("ObjectProperty: %s\n", prop.Name)
        fmt.Printf("  Domain: %s\n", prop.Domain)
        fmt.Printf("  Range: %s\n", prop.Range)
        // Description includes integrity information
        fmt.Printf("  Description: %s\n", prop.Description)
    }
}
```

## Detection Algorithm

### Workflow

1. **Column Analysis**
   - Calculate cardinality for each column
   - Identify primary key candidates (unique, required, ID-like names)

2. **Name Pattern Detection**
   - Analyze column names for FK patterns
   - Build candidate FK list with confidence scores

3. **Cardinality Filtering**
   - Check if cardinality suggests FK (5-80% of row count)
   - Boost confidence for mid-range cardinality

4. **Value Overlap Analysis**
   - For each candidate FK, compare values with all potential target columns
   - Calculate overlap percentage
   - Filter by minimum overlap threshold (70%)

5. **Confidence Aggregation**
   - Average all detection method confidence scores
   - Filter by minimum confidence threshold
   - Create FK relationships for qualified candidates

6. **Column Metadata Update**
   - Mark columns as foreign keys
   - Attach FK metadata with referenced columns and confidence

### Confidence Calculation

Overall confidence is the average of all applicable detection methods:

```
Confidence = (NameConfidence + CardinalityConfidence + OverlapConfidence) / NumMethods
```

Example:
- Name pattern (`user_id`): 0.9
- Cardinality (30% of rows): 0.7
- Value overlap (95%): 0.95
- **Overall: (0.9 + 0.7 + 0.95) / 3 = 0.85**

## Best Practices

### 1. Confidence Threshold Selection

- **High threshold (0.9+)**: Use for production systems requiring high precision
  - Fewer false positives
  - May miss some valid FKs
  
- **Medium threshold (0.7-0.9)**: Balanced approach (default: 0.8)
  - Good precision/recall trade-off
  - Recommended for most use cases

- **Low threshold (0.5-0.7)**: Exploratory analysis
  - Higher recall, more FK candidates
  - Review results manually

### 2. Sample Size Considerations

```go
config.SampleSize = 1000  // Analyze first 1000 rows
```

- Larger samples provide better cardinality estimates
- Small samples (<100 rows) may miss low-frequency FK values
- Default: 100 rows (good for most datasets)

### 3. Data Preparation

For best results:
- Use consistent column naming conventions
- Include representative sample of FK values
- Ensure data quality (minimize nulls in FK columns)

### 4. Multi-Table Analysis

When analyzing related tables:

```go
// Analyze tables separately
usersSchema := engine.InferSchema(users, "users")
ordersSchema := engine.InferSchema(orders, "orders")

// Cross-reference results
// Check if ordersSchema FKs reference usersSchema PKs
```

## Limitations

1. **Single-Table Detection**: Value overlap detection works best when both source and target columns exist in the same dataset

2. **Composite Keys**: Currently optimized for single-column FKs; composite keys are detected individually

3. **Naming Conventions**: Name-based detection relies on common patterns; custom naming conventions may require threshold adjustments

4. **Sample Size**: Limited sample size may not capture full value distribution

## Testing

Comprehensive test suite includes:

- Name pattern detection tests
- Cardinality-based detection tests
- Value overlap calculation tests
- End-to-end integration tests
- Confidence threshold tests
- Composite key scenarios

Run tests:
```bash
go test -v ./pipelines/Ontology/schema_inference -run ".*ForeignKey.*"
```

## Example Output

```
Schema: orders
Columns: 4

Columns:
  - order_id (integer) [PK, UNIQUE, REQUIRED]
    Cardinality: 5 (100.0%)
  
  - user_id (integer) [FK]
    Cardinality: 3 (60.0%)
    FK Metadata:
      Referenced Column: id
      Confidence: 0.85
      Detection Method: name_pattern,cardinality,value_overlap
  
  - amount (float) [REQUIRED]
    Cardinality: 5 (100.0%)
  
  - status (string) [REQUIRED]
    Cardinality: 2 (40.0%)

Foreign Key Relationships: 1
  1. user_id -> id
     Confidence: 0.85
     Referential Integrity: 100.0% (3/3 values match)
     Detection Methods: name_pattern, cardinality, value_overlap
```

## See Also

- [Schema Inference Engine](./README_AI_FALLBACK.md)
- [Ontology Generation](./IMPLEMENTATION_SUMMARY.md)
- [Example Code](../../../examples/fk_detection_example.go.txt)
