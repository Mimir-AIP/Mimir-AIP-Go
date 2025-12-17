# AI/LLM Fallback for Schema Inference

## Overview

The schema inference engine now supports AI/LLM fallback for improved type detection when deterministic methods have low confidence. This feature uses Large Language Models to analyze ambiguous or complex data patterns and provide enhanced type information with semantic understanding.

## Features

- **Confidence-Based Triggering**: AI fallback automatically activates when deterministic type inference confidence falls below a configurable threshold
- **Semantic Type Detection**: Identifies semantic types like email, phone, currency, URL, etc.
- **Constraint Inference**: Suggests patterns, ranges, and validation constraints
- **Ontology Mapping**: Recommends appropriate RDF/OWL ontology types
- **Graceful Degradation**: Falls back to deterministic results if AI is unavailable
- **Configurable**: Can be completely disabled or fine-tuned via configuration

## Configuration

### InferenceConfig Options

```go
type InferenceConfig struct {
    SampleSize          int     `json:"sample_size"`           // Number of rows to analyze
    ConfidenceThreshold float64 `json:"confidence_threshold"`  // Min confidence for deterministic inference (0.0-1.0)
    EnableRelationships bool    `json:"enable_relationships"`  // Detect column relationships
    EnableConstraints   bool    `json:"enable_constraints"`    // Infer constraints
    EnableAIFallback    bool    `json:"enable_ai_fallback"`    // Enable AI-enhanced inference
    AIConfidenceBoost   float64 `json:"ai_confidence_boost"`   // Confidence boost from AI (default: 0.15)
}
```

### Example Configuration

```go
config := schema_inference.InferenceConfig{
    SampleSize:          100,
    ConfidenceThreshold: 0.8,   // Trigger AI if confidence < 80%
    EnableAIFallback:    true,   // Enable AI fallback
    AIConfidenceBoost:   0.15,   // Add 15% confidence when AI is used
    EnableRelationships: true,
    EnableConstraints:   true,
}
```

## Usage

### Basic Usage (Without AI)

```go
config := schema_inference.InferenceConfig{
    SampleSize:          100,
    ConfidenceThreshold: 0.8,
    EnableAIFallback:    false, // AI disabled
}

engine := schema_inference.NewSchemaInferenceEngine(config)

data := []map[string]interface{}{
    {"id": 1, "name": "John", "age": 30},
    {"id": 2, "name": "Jane", "age": 25},
}

schema, err := engine.InferSchema(data, "users")
```

### AI-Enhanced Inference

```go
// Create LLM client
llmConfig := AI.LLMClientConfig{
    Provider: AI.ProviderOpenAI,
    APIKey:   os.Getenv("OPENAI_API_KEY"),
    Model:    "gpt-4",
}

llmClient, err := AI.NewLLMClient(llmConfig)
if err != nil {
    log.Fatal(err)
}

// Configure inference with AI
config := schema_inference.InferenceConfig{
    SampleSize:          100,
    ConfidenceThreshold: 0.8,
    EnableAIFallback:    true,
    AIConfidenceBoost:   0.15,
}

// Create engine with LLM client
engine := schema_inference.NewSchemaInferenceEngineWithLLM(config, llmClient)

// Infer schema - AI will be used for ambiguous columns
data := []map[string]interface{}{
    {"transaction_id": "TXN-001", "amount": "100.50", "status": "completed"},
    {"transaction_id": "TXN-002", "amount": "250.75", "status": "pending"},
}

schema, err := engine.InferSchema(data, "transactions")

// Check which columns were AI-enhanced
for _, col := range schema.Columns {
    if col.AIEnhanced {
        fmt.Printf("Column '%s' enhanced by AI (confidence: %.2f)\n", 
            col.Name, col.AIConfidence)
    }
}
```

### Adding LLM Client to Existing Engine

```go
engine := schema_inference.NewSchemaInferenceEngine(config)

// Add LLM client later
engine.SetLLMClient(llmClient)
```

## How It Works

### 1. Deterministic Type Inference

The engine first attempts to infer types using deterministic methods:
- Pattern matching (integers, floats, booleans, dates)
- Statistical analysis (type frequency)
- Confidence calculation based on type consistency

### 2. Confidence Check

If the confidence score is below the threshold:
- **Confidence >= Threshold**: Use deterministic result
- **Confidence < Threshold**: Trigger AI fallback (if enabled)

### 3. AI Inference

When triggered, the AI:
1. Receives column name and sample values
2. Analyzes semantic meaning and patterns
3. Returns structured JSON with:
   - Data type
   - Ontology type
   - Confidence score
   - Description
   - Constraints (patterns, ranges, enums)
   - Domain suggestions (semantic types, units)

### 4. Result Merging

The AI response is parsed and merged with the schema:
- AI confidence is boosted by `AIConfidenceBoost`
- Constraints are added to the column
- `AIEnhanced` flag is set
- Original deterministic result is used as fallback if AI fails

## AI Response Format

The AI returns structured JSON:

```json
{
  "data_type": "string",
  "ontology_type": "xsd:string",
  "confidence": 0.95,
  "description": "Transaction identifier with prefix pattern",
  "constraints": {
    "pattern": "^TXN-[0-9]+$",
    "min_length": 7,
    "max_length": 20
  },
  "domain_suggestions": {
    "semantic_type": "transaction_id",
    "unit": null
  }
}
```

## Enhanced Column Information

When AI is used, columns include additional fields:

```go
type ColumnSchema struct {
    Name         string                 `json:"name"`
    DataType     string                 `json:"data_type"`
    OntologyType string                 `json:"ontology_type"`
    // ... standard fields ...
    
    // AI-specific fields
    AIEnhanced   bool                   `json:"ai_enhanced,omitempty"`
    AIConfidence float64                `json:"ai_confidence,omitempty"`
    Constraints  map[string]interface{} `json:"constraints"` // May include semantic_type, unit, pattern
}
```

## Examples

### Detecting Currency

**Input:**
```go
{"amount": "$1,234.56"}
{"amount": "$987.65"}
```

**AI Detection:**
- `data_type`: "string" (formatted with symbol)
- `semantic_type`: "currency"
- `unit`: "USD"
- `pattern`: "^\\$[0-9,]+\\.[0-9]{2}$"

### Detecting Phone Numbers

**Input:**
```go
{"phone": "+1-555-0123"}
{"phone": "+1-555-0456"}
```

**AI Detection:**
- `data_type`: "string"
- `semantic_type`: "phone"
- `pattern`: "^\\+[1-9]\\d{1,14}$"
- `description`: "Phone number in E.164 format"

### Detecting Transaction IDs

**Input:**
```go
{"txn_id": "TXN-001"}
{"txn_id": "TXN-002"}
```

**AI Detection:**
- `data_type`: "string"
- `semantic_type`: "transaction_id"
- `pattern`: "^TXN-[0-9]+$"
- `description`: "Transaction identifier with prefix"

## Performance Considerations

### When AI is Called

- Only for columns with confidence < threshold
- Typical: 0-30% of columns trigger AI
- Pure numeric/string data rarely needs AI

### Latency

- AI calls are made per-column
- ~100-500ms per AI request
- Consider caching for repeated inferences

### Cost

- Each AI call consumes LLM API tokens
- Typical: 300-500 tokens per column
- Use higher threshold to reduce AI usage

## Error Handling

```go
schema, err := engine.InferSchema(data, "test")
// Main inference error

// Individual column AI failures are logged but don't fail the whole inference
// The deterministic result is used as fallback
```

Log messages:
```
INFO: Type inference confidence below threshold, attempting AI fallback
INFO: AI fallback successful (column=amount, confidence=0.92)
WARN: AI fallback failed, using deterministic result (error=...)
```

## Best Practices

1. **Set Appropriate Threshold**: 
   - Lower (0.6-0.7): More deterministic, less AI usage
   - Higher (0.8-0.9): More AI enhancement, better semantic detection

2. **Sample Size**:
   - 50-100 samples is usually sufficient
   - More samples = better deterministic confidence

3. **LLM Model Selection**:
   - GPT-4: Best accuracy, higher cost
   - GPT-3.5: Good balance
   - Local models (Ollama): Free but variable quality

4. **Caching**:
   - Cache schemas for repeated datasets
   - Consider schema versioning

5. **Fallback Strategy**:
   - Always enable `EnableConstraints` for richer base inference
   - Test with AI disabled first to ensure deterministic quality

## Testing

Run the AI fallback tests:

```bash
go test -v ./pipelines/Ontology/schema_inference/ -run TestAI
```

Test coverage includes:
- AI disabled behavior
- AI enabled with low confidence
- AI enhanced type information
- JSON extraction and parsing
- Confidence calculations
- Error handling

## API Reference

### Constructor Functions

```go
// Create engine without LLM
func NewSchemaInferenceEngine(config InferenceConfig) *SchemaInferenceEngine

// Create engine with LLM client
func NewSchemaInferenceEngineWithLLM(config InferenceConfig, llmClient AI.LLMClient) *SchemaInferenceEngine
```

### Methods

```go
// Set or update LLM client
func (e *SchemaInferenceEngine) SetLLMClient(client AI.LLMClient)

// Infer schema (AI used automatically if configured)
func (e *SchemaInferenceEngine) InferSchema(data interface{}, name string) (*DataSchema, error)
```

### Internal Methods (for advanced usage)

```go
// Direct AI inference for a column
func (e *SchemaInferenceEngine) inferTypeWithAI(ctx context.Context, columnName string, sampleValues []interface{}) (TypeInfo, error)

// Check if AI should be used
func (e *SchemaInferenceEngine) shouldUseAIFallback(confidence float64) bool
```

## Troubleshooting

### AI Not Being Called

- Check `EnableAIFallback` is true
- Verify LLM client is set
- Ensure confidence < threshold (check logs)
- Confirm data has low confidence (mixed types)

### AI Returning Errors

- Verify API key is valid
- Check network connectivity
- Review LLM rate limits
- Ensure model supports function calling

### Unexpected Type Detection

- Review sample data quality
- Adjust confidence threshold
- Check AI prompt in logs
- Consider providing more samples

## Future Enhancements

Planned improvements:
- Batch AI requests for multiple columns
- Schema learning from user feedback
- Custom prompt templates
- Confidence calibration
- Multi-model ensemble
- Streaming inference for large datasets

## See Also

- [Ontology Implementation Guide](../../docs/ONTOLOGY_IMPLEMENTATION_GUIDE.md)
- [AI Plugin Development](../../docs/PLUGIN_DEVELOPMENT_GUIDE.md)
- [Schema Inference Examples](../../examples/schema_inference_ai_fallback.go.txt)
