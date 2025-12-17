package schema_inference

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/AI"
	"github.com/Mimir-AIP/Mimir-AIP-Go/utils"
)

// DataSchema represents the inferred schema of structured data
type DataSchema struct {
	Name          string                 `json:"name"`
	Description   string                 `json:"description"`
	Columns       []ColumnSchema         `json:"columns"`
	Relationships []RelationshipSchema   `json:"relationships"`
	Metadata      map[string]interface{} `json:"metadata"`
	InferredAt    time.Time              `json:"inferred_at"`
}

// ColumnSchema represents schema information for a single column
type ColumnSchema struct {
	Name         string                 `json:"name"`
	DataType     string                 `json:"data_type"`
	OntologyType string                 `json:"ontology_type"`
	IsPrimaryKey bool                   `json:"is_primary_key"`
	IsForeignKey bool                   `json:"is_foreign_key"`
	IsRequired   bool                   `json:"is_required"`
	IsUnique     bool                   `json:"is_unique"`
	SampleValues []interface{}          `json:"sample_values"`
	Constraints  map[string]interface{} `json:"constraints"`
	Description  string                 `json:"description"`
	AIEnhanced   bool                   `json:"ai_enhanced,omitempty"`   // Indicates if AI was used for inference
	AIConfidence float64                `json:"ai_confidence,omitempty"` // AI confidence score if available
}

// RelationshipSchema represents relationships between columns/entities
type RelationshipSchema struct {
	SourceColumn     string  `json:"source_column"`
	TargetColumn     string  `json:"target_column"`
	RelationshipType string  `json:"relationship_type"`
	Cardinality      string  `json:"cardinality"`
	Strength         float64 `json:"strength"`
	Description      string  `json:"description"`
}

// SchemaInferenceEngine analyzes data and creates schemas
type SchemaInferenceEngine struct {
	config    InferenceConfig
	llmClient AI.LLMClient
	logger    *utils.Logger
}

// InferenceConfig configures the inference behavior
type InferenceConfig struct {
	SampleSize          int     `json:"sample_size"`
	ConfidenceThreshold float64 `json:"confidence_threshold"`
	EnableRelationships bool    `json:"enable_relationships"`
	EnableConstraints   bool    `json:"enable_constraints"`
	EnableAIFallback    bool    `json:"enable_ai_fallback"`
	AIConfidenceBoost   float64 `json:"ai_confidence_boost"` // How much to boost confidence when AI is used
}

// NewSchemaInferenceEngine creates a new inference engine
func NewSchemaInferenceEngine(config InferenceConfig) *SchemaInferenceEngine {
	if config.SampleSize <= 0 {
		config.SampleSize = 100
	}
	if config.ConfidenceThreshold <= 0 {
		config.ConfidenceThreshold = 0.8
	}
	if config.AIConfidenceBoost <= 0 {
		config.AIConfidenceBoost = 0.15 // Default 15% confidence boost from AI
	}

	return &SchemaInferenceEngine{
		config: config,
		logger: utils.GetLogger(),
	}
}

// NewSchemaInferenceEngineWithLLM creates a new inference engine with LLM client
func NewSchemaInferenceEngineWithLLM(config InferenceConfig, llmClient AI.LLMClient) *SchemaInferenceEngine {
	engine := NewSchemaInferenceEngine(config)
	engine.llmClient = llmClient
	return engine
}

// SetLLMClient sets the LLM client for AI-enhanced inference
func (e *SchemaInferenceEngine) SetLLMClient(client AI.LLMClient) {
	e.llmClient = client
}

// InferSchema analyzes data rows and creates a schema
func (e *SchemaInferenceEngine) InferSchema(data interface{}, name string) (*DataSchema, error) {
	// Handle different data formats
	switch d := data.(type) {
	case map[string]interface{}:
		// Single row/object
		return e.inferFromObject(d, name)
	case []map[string]interface{}:
		// Array of objects (typical for CSV/JSON)
		return e.inferFromArray(d, name)
	default:
		return nil, fmt.Errorf("unsupported data format: %T", data)
	}
}

// inferFromArray analyzes an array of objects (typical CSV/JSON structure)
func (e *SchemaInferenceEngine) inferFromArray(rows []map[string]interface{}, name string) (*DataSchema, error) {
	if len(rows) == 0 {
		return nil, fmt.Errorf("no data rows provided")
	}

	schema := &DataSchema{
		Name:          name,
		Description:   fmt.Sprintf("Schema inferred from %d data rows", len(rows)),
		Columns:       []ColumnSchema{},
		Relationships: []RelationshipSchema{},
		Metadata:      make(map[string]interface{}),
		InferredAt:    time.Now(),
	}

	// Sample rows for analysis (limit to config sample size)
	sampleSize := e.config.SampleSize
	if len(rows) < sampleSize {
		sampleSize = len(rows)
	}
	sampleRows := rows[:sampleSize]

	// Get all column names
	columnNames := make(map[string]bool)
	for _, row := range sampleRows {
		for colName := range row {
			columnNames[colName] = true
		}
	}

	// Analyze each column
	for colName := range columnNames {
		colSchema := e.analyzeColumn(colName, sampleRows)
		schema.Columns = append(schema.Columns, colSchema)
	}

	// Detect relationships if enabled
	if e.config.EnableRelationships {
		schema.Relationships = e.detectRelationships(schema.Columns, sampleRows)
	}

	// Add metadata
	schema.Metadata["total_rows"] = len(rows)
	schema.Metadata["analyzed_rows"] = sampleSize
	schema.Metadata["column_count"] = len(schema.Columns)
	schema.Metadata["relationship_count"] = len(schema.Relationships)

	return schema, nil
}

// analyzeColumn analyzes a single column across all rows
func (e *SchemaInferenceEngine) analyzeColumn(colName string, rows []map[string]interface{}) ColumnSchema {
	colSchema := ColumnSchema{
		Name:         colName,
		DataType:     "string", // default
		OntologyType: "xsd:string",
		SampleValues: []interface{}{},
		Constraints:  make(map[string]interface{}),
	}

	// Collect sample values and analyze types
	var values []interface{}
	var nullCount int
	var uniqueValues = make(map[string]int)

	for _, row := range rows {
		if val, exists := row[colName]; exists && val != nil {
			values = append(values, val)
			strVal := fmt.Sprintf("%v", val)
			uniqueValues[strVal]++
		} else {
			nullCount++
		}
	}

	// Store sample values (up to 5)
	sampleSize := 5
	if len(values) < sampleSize {
		sampleSize = len(values)
	}
	colSchema.SampleValues = values[:sampleSize]

	// Infer data type with confidence tracking
	typeInfo := e.inferColumnType(context.Background(), colName, values)
	colSchema.DataType = typeInfo.DataType
	colSchema.OntologyType = typeInfo.OntologyType
	colSchema.AIEnhanced = typeInfo.AIEnhanced
	colSchema.AIConfidence = typeInfo.Confidence

	// Check constraints
	totalValues := len(values)
	if e.config.EnableConstraints && totalValues > 0 {
		// Required check
		colSchema.IsRequired = nullCount == 0

		// Unique check (if less than 10% duplicates, consider unique)
		if len(uniqueValues) >= int(float64(totalValues)*0.9) {
			colSchema.IsUnique = true
		}

		// Primary key candidates (unique, required, looks like ID)
		if colSchema.IsUnique && colSchema.IsRequired &&
			e.looksLikePrimaryKey(colName, colSchema.DataType) {
			colSchema.IsPrimaryKey = true
		}

		// Foreign key candidates (references other tables)
		if e.looksLikeForeignKey(colName, colSchema.DataType) {
			colSchema.IsForeignKey = true
		}

		// Merge AI-suggested constraints if available
		if typeInfo.Constraints != nil {
			for key, value := range typeInfo.Constraints {
				colSchema.Constraints[key] = value
			}
		}
	}

	// Generate description (use AI description if available)
	if typeInfo.Description != "" {
		colSchema.Description = typeInfo.Description
	} else {
		colSchema.Description = e.generateColumnDescription(colSchema)
	}

	return colSchema
}

// TypeInfo holds comprehensive type inference results
type TypeInfo struct {
	DataType     string
	OntologyType string
	Confidence   float64
	AIEnhanced   bool
	Description  string
	Constraints  map[string]interface{}
}

// inferColumnType performs type inference with optional AI fallback
func (e *SchemaInferenceEngine) inferColumnType(ctx context.Context, columnName string, values []interface{}) TypeInfo {
	// First try deterministic inference
	dataType, ontologyType, confidence := e.inferDataTypeWithConfidence(values)

	typeInfo := TypeInfo{
		DataType:     dataType,
		OntologyType: ontologyType,
		Confidence:   confidence,
		AIEnhanced:   false,
		Constraints:  make(map[string]interface{}),
	}

	// Check if we should use AI fallback
	if e.shouldUseAIFallback(confidence) {
		e.logger.Info("Type inference confidence below threshold, attempting AI fallback",
			utils.String("column", columnName),
			utils.Float("confidence", confidence),
			utils.Float("threshold", e.config.ConfidenceThreshold))

		// Try AI inference
		aiTypeInfo, err := e.inferTypeWithAI(ctx, columnName, values)
		if err != nil {
			e.logger.Warn("AI fallback failed, using deterministic result",
				utils.String("column", columnName),
				utils.Error(err))
			return typeInfo
		}

		// AI succeeded - merge results
		e.logger.Info("AI fallback successful",
			utils.String("column", columnName),
			utils.String("ai_type", aiTypeInfo.DataType),
			utils.Float("ai_confidence", aiTypeInfo.Confidence))

		return aiTypeInfo
	}

	return typeInfo
}

// shouldUseAIFallback determines if AI fallback should be used
func (e *SchemaInferenceEngine) shouldUseAIFallback(confidence float64) bool {
	return e.config.EnableAIFallback &&
		e.llmClient != nil &&
		confidence < e.config.ConfidenceThreshold
}

// inferTypeWithAI uses LLM to infer column type when deterministic methods have low confidence
func (e *SchemaInferenceEngine) inferTypeWithAI(ctx context.Context, columnName string, sampleValues []interface{}) (TypeInfo, error) {
	if e.llmClient == nil {
		return TypeInfo{}, fmt.Errorf("LLM client not configured")
	}

	// Prepare sample values for prompt (limit to 10 samples)
	maxSamples := 10
	if len(sampleValues) > maxSamples {
		sampleValues = sampleValues[:maxSamples]
	}

	// Convert values to strings for prompt
	var sampleStrs []string
	for _, val := range sampleValues {
		sampleStrs = append(sampleStrs, fmt.Sprintf("%v", val))
	}

	// Create intelligent prompt
	prompt := e.buildAIPrompt(columnName, sampleStrs)

	// Make LLM request
	request := AI.LLMRequest{
		Messages: []AI.LLMMessage{
			{
				Role:    "system",
				Content: "You are a data schema expert. Analyze column data and provide type information in JSON format.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature: 0.1, // Low temperature for consistent results
		MaxTokens:   500,
	}

	response, err := e.llmClient.Complete(ctx, request)
	if err != nil {
		return TypeInfo{}, fmt.Errorf("LLM request failed: %w", err)
	}

	// Parse AI response
	typeInfo, err := e.parseAIResponse(response.Content)
	if err != nil {
		return TypeInfo{}, fmt.Errorf("failed to parse AI response: %w", err)
	}

	// Mark as AI-enhanced and boost confidence
	typeInfo.AIEnhanced = true
	typeInfo.Confidence += e.config.AIConfidenceBoost
	if typeInfo.Confidence > 1.0 {
		typeInfo.Confidence = 1.0
	}

	return typeInfo, nil
}

// buildAIPrompt creates a structured prompt for type inference
func (e *SchemaInferenceEngine) buildAIPrompt(columnName string, sampleValues []string) string {
	samplesJSON, _ := json.Marshal(sampleValues)

	return fmt.Sprintf(`Analyze this database column and infer its type and characteristics:

Column Name: %s
Sample Values: %s

Please provide your analysis in the following JSON format:
{
  "data_type": "string|integer|float|boolean|date",
  "ontology_type": "xsd type (e.g., xsd:string, xsd:integer, xsd:decimal, xsd:boolean, xsd:dateTime)",
  "confidence": 0.0-1.0,
  "description": "brief description of what this column represents",
  "constraints": {
    "pattern": "regex pattern if applicable",
    "min_length": number,
    "max_length": number,
    "min_value": number,
    "max_value": number,
    "enum_values": ["list", "of", "possible", "values"] (if limited set)
  },
  "domain_suggestions": {
    "semantic_type": "email|phone|url|currency|percentage|etc",
    "unit": "meters|dollars|seconds|etc if applicable"
  }
}

Consider:
1. The column name for semantic hints
2. Value patterns, formats, and ranges
3. Potential semantic meaning (email, phone, currency, etc.)
4. Appropriate RDF/OWL ontology types
5. Any constraints that should be enforced

Return ONLY the JSON object, no additional text.`, columnName, string(samplesJSON))
}

// AIResponseFormat represents the expected AI response structure
type AIResponseFormat struct {
	DataType     string                 `json:"data_type"`
	OntologyType string                 `json:"ontology_type"`
	Confidence   float64                `json:"confidence"`
	Description  string                 `json:"description"`
	Constraints  map[string]interface{} `json:"constraints"`
	Domain       struct {
		SemanticType string `json:"semantic_type"`
		Unit         string `json:"unit"`
	} `json:"domain_suggestions"`
}

// parseAIResponse parses the LLM response into TypeInfo
func (e *SchemaInferenceEngine) parseAIResponse(content string) (TypeInfo, error) {
	// Extract JSON from response (handle markdown code blocks)
	jsonStr := e.extractJSON(content)

	var aiResp AIResponseFormat
	if err := json.Unmarshal([]byte(jsonStr), &aiResp); err != nil {
		return TypeInfo{}, fmt.Errorf("failed to unmarshal AI response: %w", err)
	}

	// Validate and normalize data type
	dataType := e.normalizeDataType(aiResp.DataType)
	ontologyType := aiResp.OntologyType
	if ontologyType == "" {
		ontologyType = e.dataTypeToOntology(dataType)
	}

	// Build constraints map
	constraints := make(map[string]interface{})
	if aiResp.Constraints != nil {
		for k, v := range aiResp.Constraints {
			constraints[k] = v
		}
	}

	// Add domain information to constraints
	if aiResp.Domain.SemanticType != "" {
		constraints["semantic_type"] = aiResp.Domain.SemanticType
	}
	if aiResp.Domain.Unit != "" {
		constraints["unit"] = aiResp.Domain.Unit
	}

	return TypeInfo{
		DataType:     dataType,
		OntologyType: ontologyType,
		Confidence:   aiResp.Confidence,
		AIEnhanced:   true,
		Description:  aiResp.Description,
		Constraints:  constraints,
	}, nil
}

// extractJSON extracts JSON from response that might be wrapped in markdown
func (e *SchemaInferenceEngine) extractJSON(content string) string {
	// Trim whitespace
	content = strings.TrimSpace(content)

	// Try to find JSON in markdown code blocks with json language
	jsonPattern := regexp.MustCompile("(?s)```json\\s*({[^`]*})\\s*```")
	matches := jsonPattern.FindStringSubmatch(content)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}

	// Try to find JSON in markdown code blocks without language
	jsonPattern = regexp.MustCompile("(?s)```\\s*({[^`]*})\\s*```")
	matches = jsonPattern.FindStringSubmatch(content)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}

	// If content starts with { and ends with }, it's likely JSON
	if strings.HasPrefix(content, "{") && strings.HasSuffix(content, "}") {
		return content
	}

	// Try to find raw JSON object anywhere in the content
	// Match balanced braces
	startIdx := strings.Index(content, "{")
	if startIdx == -1 {
		return content // No JSON found, return as-is
	}

	// Find matching closing brace
	braceCount := 0
	for i := startIdx; i < len(content); i++ {
		if content[i] == '{' {
			braceCount++
		} else if content[i] == '}' {
			braceCount--
			if braceCount == 0 {
				return strings.TrimSpace(content[startIdx : i+1])
			}
		}
	}

	// Return as-is and let JSON parser handle it
	return content
}

// normalizeDataType ensures the data type is one of our supported types
func (e *SchemaInferenceEngine) normalizeDataType(dataType string) string {
	dataType = strings.ToLower(strings.TrimSpace(dataType))

	switch dataType {
	case "string", "text", "varchar", "char":
		return "string"
	case "integer", "int", "long", "bigint":
		return "integer"
	case "float", "double", "decimal", "number", "numeric":
		return "float"
	case "boolean", "bool":
		return "boolean"
	case "date", "datetime", "timestamp", "time":
		return "date"
	default:
		return "string" // Safe fallback
	}
}

// inferDataType analyzes values to determine the best data type (deprecated - use inferDataTypeWithConfidence)
func (e *SchemaInferenceEngine) inferDataType(values []interface{}) (string, string) {
	dataType, ontologyType, _ := e.inferDataTypeWithConfidence(values)
	return dataType, ontologyType
}

// inferDataTypeWithConfidence analyzes values to determine the best data type with confidence score
func (e *SchemaInferenceEngine) inferDataTypeWithConfidence(values []interface{}) (string, string, float64) {
	if len(values) == 0 {
		return "string", "xsd:string", 0.0
	}

	// Type counters - track the most specific type for each value
	typeCounts := make(map[string]int)

	for _, val := range values {
		if val == nil {
			continue
		}

		strVal := fmt.Sprintf("%v", val)
		detectedType := "string" // default

		// Check types from most specific to least specific
		// Check for boolean first
		if _, err := strconv.ParseBool(strVal); err == nil {
			// Only count as boolean if it's actually a boolean value
			if strVal == "true" || strVal == "false" || strVal == "True" || strVal == "False" {
				detectedType = "boolean"
			}
		}

		// Check for date
		if detectedType == "string" && e.looksLikeDate(strVal) {
			detectedType = "date"
		}

		// Check for integer (must not have decimal point)
		if detectedType == "string" {
			if _, err := strconv.ParseInt(strVal, 10, 64); err == nil {
				detectedType = "integer"
			}
		}

		// Check for float (includes integers, but only if integer check failed)
		if detectedType == "string" {
			if _, err := strconv.ParseFloat(strVal, 64); err == nil {
				detectedType = "float"
			}
		}

		typeCounts[detectedType]++
	}

	// Find most common type
	maxCount := 0
	bestType := "string"
	total := len(values)

	for dataType, count := range typeCounts {
		if count > maxCount {
			maxCount = count
			bestType = dataType
		}
	}

	// Calculate confidence as ratio of most common type
	confidence := 0.0
	if total > 0 {
		confidence = float64(maxCount) / float64(total)
	}

	return bestType, e.dataTypeToOntology(bestType), confidence
}

// dataTypeToOntology converts data types to ontology types
func (e *SchemaInferenceEngine) dataTypeToOntology(dataType string) string {
	switch dataType {
	case "integer":
		return "xsd:integer"
	case "float":
		return "xsd:decimal"
	case "boolean":
		return "xsd:boolean"
	case "date":
		return "xsd:dateTime"
	default:
		return "xsd:string"
	}
}

// looksLikePrimaryKey checks if a column looks like a primary key
func (e *SchemaInferenceEngine) looksLikePrimaryKey(colName string, dataType string) bool {
	name := strings.ToLower(colName)

	// Check name patterns
	if strings.Contains(name, "id") ||
		strings.Contains(name, "key") ||
		strings.HasSuffix(name, "_id") ||
		name == "uuid" ||
		name == "guid" {
		return true
	}

	// Check if it's integer type (common for IDs)
	return dataType == "integer"
}

// looksLikeForeignKey checks if a column looks like a foreign key
func (e *SchemaInferenceEngine) looksLikeForeignKey(colName string, dataType string) bool {
	name := strings.ToLower(colName)

	// Foreign keys often end with _id and reference other tables
	if strings.HasSuffix(name, "_id") && len(name) > 3 {
		return true
	}

	return false
}

// looksLikeDate checks if a string looks like a date
func (e *SchemaInferenceEngine) looksLikeDate(str string) bool {
	// Common date patterns
	datePatterns := []string{
		`^\d{4}-\d{2}-\d{2}$`,                  // YYYY-MM-DD
		`^\d{2}/\d{2}/\d{4}$`,                  // MM/DD/YYYY
		`^\d{4}/\d{2}/\d{2}$`,                  // YYYY/MM/DD
		`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}`, // ISO datetime
	}

	for _, pattern := range datePatterns {
		if matched, _ := regexp.MatchString(pattern, str); matched {
			return true
		}
	}

	return false
}

// generateColumnDescription creates a human-readable description
func (e *SchemaInferenceEngine) generateColumnDescription(col ColumnSchema) string {
	desc := fmt.Sprintf("%s column of type %s", col.Name, col.DataType)

	if col.IsPrimaryKey {
		desc += " (primary key)"
	} else if col.IsForeignKey {
		desc += " (foreign key)"
	}

	if col.IsRequired {
		desc += ", required"
	}

	if col.IsUnique {
		desc += ", unique values"
	}

	return desc
}

// detectRelationships analyzes columns to find relationships
func (e *SchemaInferenceEngine) detectRelationships(columns []ColumnSchema, rows []map[string]interface{}) []RelationshipSchema {
	var relationships []RelationshipSchema

	// Find potential foreign key relationships
	for _, col := range columns {
		if col.IsForeignKey {
			// Look for referenced table (remove _id suffix)
			refTable := strings.TrimSuffix(strings.ToLower(col.Name), "_id")

			// Find potential primary key column that this might reference
			for _, targetCol := range columns {
				if targetCol.IsPrimaryKey &&
					strings.Contains(strings.ToLower(targetCol.Name), refTable) {

					rel := RelationshipSchema{
						SourceColumn:     col.Name,
						TargetColumn:     targetCol.Name,
						RelationshipType: "references",
						Cardinality:      "many-to-one",
						Strength:         0.8,
						Description:      fmt.Sprintf("%s references %s", col.Name, targetCol.Name),
					}
					relationships = append(relationships, rel)
				}
			}
		}
	}

	return relationships
}

// inferFromObject analyzes a single object (for JSON data)
func (e *SchemaInferenceEngine) inferFromObject(obj map[string]interface{}, name string) (*DataSchema, error) {
	// Convert single object to array format
	rows := []map[string]interface{}{obj}
	return e.inferFromArray(rows, name)
}
