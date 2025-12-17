package schema_inference

import (
	"context"
	"testing"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/AI"
	"github.com/stretchr/testify/assert"
)

// MockLLMClient for testing
type MockLLMClient struct {
	responses map[string]string
	callCount int
}

func NewMockLLMClient() *MockLLMClient {
	return &MockLLMClient{
		responses: make(map[string]string),
		callCount: 0,
	}
}

func (m *MockLLMClient) Complete(ctx context.Context, request AI.LLMRequest) (*AI.LLMResponse, error) {
	m.callCount++

	// Extract column name from prompt
	prompt := request.Messages[1].Content

	var response string
	if contains(prompt, "transaction_id") {
		response = `{
			"data_type": "string",
			"ontology_type": "xsd:string",
			"confidence": 0.95,
			"description": "Transaction identifier",
			"constraints": {
				"pattern": "^TXN-[0-9]+$"
			}
		}`
	} else if contains(prompt, "amount") || contains(prompt, "price") {
		response = `{
			"data_type": "float",
			"ontology_type": "xsd:decimal",
			"confidence": 0.92,
			"description": "Monetary amount",
			"constraints": {
				"min_value": 0
			},
			"domain_suggestions": {
				"semantic_type": "currency",
				"unit": "USD"
			}
		}`
	} else {
		response = `{
			"data_type": "string",
			"ontology_type": "xsd:string",
			"confidence": 0.8,
			"description": "Text field"
		}`
	}

	return &AI.LLMResponse{
		Content:      response,
		FinishReason: "stop",
		Model:        "mock-model",
	}, nil
}

func (m *MockLLMClient) CompleteSimple(ctx context.Context, prompt string) (string, error) {
	req := AI.LLMRequest{
		Messages: []AI.LLMMessage{{Role: "user", Content: prompt}},
	}
	resp, err := m.Complete(ctx, req)
	if err != nil {
		return "", err
	}
	return resp.Content, nil
}

func (m *MockLLMClient) GetProvider() AI.LLMProvider {
	return "mock"
}

func (m *MockLLMClient) GetDefaultModel() string {
	return "mock-model"
}

func (m *MockLLMClient) ValidateConfig() error {
	return nil
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestAIFallbackDisabled tests that AI is not called when disabled
func TestAIFallbackDisabled(t *testing.T) {
	mockLLM := NewMockLLMClient()

	config := InferenceConfig{
		SampleSize:          50,
		ConfidenceThreshold: 0.9,   // High threshold
		EnableAIFallback:    false, // AI disabled
	}

	engine := NewSchemaInferenceEngineWithLLM(config, mockLLM)

	// Mixed data that would trigger AI if enabled
	data := []map[string]interface{}{
		{"mixed": "abc", "values": "123"},
		{"mixed": "456", "values": "def"},
	}

	schema, err := engine.InferSchema(data, "test")
	assert.NoError(t, err)
	assert.NotNil(t, schema)
	assert.Equal(t, 0, mockLLM.callCount, "AI should not be called when disabled")
}

// TestAIFallbackEnabled tests that AI is called when confidence is low
func TestAIFallbackEnabled(t *testing.T) {
	mockLLM := NewMockLLMClient()

	config := InferenceConfig{
		SampleSize:          50,
		ConfidenceThreshold: 0.95, // Very high threshold to trigger AI
		EnableAIFallback:    true,
		AIConfidenceBoost:   0.15,
	}

	engine := NewSchemaInferenceEngineWithLLM(config, mockLLM)

	// Data with mixed types to trigger low confidence
	// "mixed" column has strings and numbers, "amount" has numeric strings
	data := []map[string]interface{}{
		{"mixed": "abc", "amount": "100.50"},
		{"mixed": 123, "amount": "200.75"},
		{"mixed": "def", "amount": "50.25"},
		{"mixed": 456, "amount": "75.00"},
	}

	schema, err := engine.InferSchema(data, "transactions")
	assert.NoError(t, err)
	assert.NotNil(t, schema)

	// The "mixed" column should have low confidence and trigger AI
	// Check that at least one column was AI-enhanced
	aiEnhancedCount := 0
	for _, col := range schema.Columns {
		if col.AIEnhanced {
			aiEnhancedCount++
			assert.Greater(t, col.AIConfidence, 0.0, "AI-enhanced columns should have confidence")
		}
	}

	// With a high threshold and mixed data, AI should be called
	assert.Greater(t, mockLLM.callCount, 0, "AI should be called for low-confidence inference")
}

// TestAIEnhancedTypeInfo tests AI-enhanced type information
func TestAIEnhancedTypeInfo(t *testing.T) {
	mockLLM := NewMockLLMClient()

	config := InferenceConfig{
		SampleSize:          50,
		ConfidenceThreshold: 0.95,
		EnableAIFallback:    true,
		AIConfidenceBoost:   0.2,
	}

	engine := NewSchemaInferenceEngineWithLLM(config, mockLLM)

	data := []map[string]interface{}{
		{"price": "19.99"},
		{"price": "29.99"},
		{"price": "39.99"},
	}

	schema, err := engine.InferSchema(data, "products")
	assert.NoError(t, err)
	assert.NotNil(t, schema)

	// Find the price column
	var priceCol *ColumnSchema
	for i := range schema.Columns {
		if schema.Columns[i].Name == "price" {
			priceCol = &schema.Columns[i]
			break
		}
	}

	assert.NotNil(t, priceCol, "Price column should exist")

	if priceCol.AIEnhanced {
		assert.Equal(t, "float", priceCol.DataType, "AI should detect float type for prices")
		assert.NotEmpty(t, priceCol.Description, "AI should provide description")

		// Check for semantic type
		if semanticType, ok := priceCol.Constraints["semantic_type"].(string); ok {
			assert.Equal(t, "currency", semanticType, "Should detect currency semantic type")
		}
	}
}

// TestInferTypeWithAI tests direct AI type inference
func TestInferTypeWithAI(t *testing.T) {
	mockLLM := NewMockLLMClient()

	config := InferenceConfig{
		EnableAIFallback:  true,
		AIConfidenceBoost: 0.15,
	}

	engine := NewSchemaInferenceEngineWithLLM(config, mockLLM)

	values := []interface{}{"TXN-001", "TXN-002", "TXN-003"}

	typeInfo, err := engine.inferTypeWithAI(context.Background(), "transaction_id", values)
	assert.NoError(t, err)
	assert.Equal(t, "string", typeInfo.DataType)
	assert.True(t, typeInfo.AIEnhanced)
	assert.Greater(t, typeInfo.Confidence, 0.0)
	assert.NotEmpty(t, typeInfo.Description)
}

// TestAIPromptBuilding tests the prompt construction
func TestAIPromptBuilding(t *testing.T) {
	config := InferenceConfig{}
	engine := NewSchemaInferenceEngine(config)

	prompt := engine.buildAIPrompt("user_email", []string{"john@example.com", "jane@example.com"})

	assert.Contains(t, prompt, "user_email", "Prompt should contain column name")
	assert.Contains(t, prompt, "john@example.com", "Prompt should contain sample values")
	assert.Contains(t, prompt, "data_type", "Prompt should request data type")
	assert.Contains(t, prompt, "ontology_type", "Prompt should request ontology type")
	assert.Contains(t, prompt, "JSON", "Prompt should request JSON format")
}

// TestExtractJSON tests JSON extraction from various formats
func TestExtractJSON(t *testing.T) {
	config := InferenceConfig{}
	engine := NewSchemaInferenceEngine(config)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Plain JSON",
			input:    `{"type": "string"}`,
			expected: `{"type": "string"}`,
		},
		{
			name:     "Markdown code block",
			input:    "```json\n{\"type\": \"string\"}\n```",
			expected: `{"type": "string"}`,
		},
		{
			name:     "Code block without language",
			input:    "```\n{\"type\": \"string\"}\n```",
			expected: `{"type": "string"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.extractJSON(tt.input)
			assert.JSONEq(t, tt.expected, result)
		})
	}
}

// TestNormalizeDataType tests data type normalization
func TestNormalizeDataType(t *testing.T) {
	config := InferenceConfig{}
	engine := NewSchemaInferenceEngine(config)

	tests := []struct {
		input    string
		expected string
	}{
		{"string", "string"},
		{"String", "string"},
		{"text", "string"},
		{"varchar", "string"},
		{"integer", "integer"},
		{"int", "integer"},
		{"bigint", "integer"},
		{"float", "float"},
		{"double", "float"},
		{"decimal", "float"},
		{"boolean", "boolean"},
		{"bool", "boolean"},
		{"date", "date"},
		{"datetime", "date"},
		{"timestamp", "date"},
		{"unknown", "string"}, // fallback
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := engine.normalizeDataType(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestParseAIResponse tests parsing of AI responses
func TestParseAIResponse(t *testing.T) {
	config := InferenceConfig{}
	engine := NewSchemaInferenceEngine(config)

	jsonResponse := `{
		"data_type": "float",
		"ontology_type": "xsd:decimal",
		"confidence": 0.92,
		"description": "Monetary value",
		"constraints": {
			"min_value": 0,
			"pattern": "^[0-9]+\\.[0-9]{2}$"
		},
		"domain_suggestions": {
			"semantic_type": "currency",
			"unit": "USD"
		}
	}`

	typeInfo, err := engine.parseAIResponse(jsonResponse)
	assert.NoError(t, err)
	assert.Equal(t, "float", typeInfo.DataType)
	assert.Equal(t, "xsd:decimal", typeInfo.OntologyType)
	assert.Equal(t, 0.92, typeInfo.Confidence)
	assert.Equal(t, "Monetary value", typeInfo.Description)
	assert.True(t, typeInfo.AIEnhanced)

	// Check constraints
	assert.Equal(t, float64(0), typeInfo.Constraints["min_value"])
	assert.Equal(t, "currency", typeInfo.Constraints["semantic_type"])
	assert.Equal(t, "USD", typeInfo.Constraints["unit"])
}

// TestConfidenceThreshold tests that threshold properly triggers AI
func TestConfidenceThreshold(t *testing.T) {
	tests := []struct {
		name                string
		confidenceThreshold float64
		data                []map[string]interface{}
		expectAICalls       bool
	}{
		{
			name:                "Low threshold - no AI",
			confidenceThreshold: 0.5,
			data: []map[string]interface{}{
				{"name": "Alice"},
				{"name": "Bob"},
				{"name": "Charlie"},
			},
			expectAICalls: false,
		},
		{
			name:                "High threshold with mixed data - triggers AI",
			confidenceThreshold: 0.99,
			data: []map[string]interface{}{
				{"value": "abc"},
				{"value": 123},
				{"value": "def"},
			},
			expectAICalls: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockLLM := NewMockLLMClient()

			config := InferenceConfig{
				SampleSize:          50,
				ConfidenceThreshold: tt.confidenceThreshold,
				EnableAIFallback:    true,
				AIConfidenceBoost:   0.15,
			}

			engine := NewSchemaInferenceEngineWithLLM(config, mockLLM)

			_, err := engine.InferSchema(tt.data, "test")
			assert.NoError(t, err)

			if tt.expectAICalls {
				assert.Greater(t, mockLLM.callCount, 0, "AI should be called with high threshold and mixed data")
			} else {
				assert.Equal(t, 0, mockLLM.callCount, "AI should not be called with low threshold")
			}
		})
	}
}

// TestInferDataTypeWithConfidence tests confidence calculation
func TestInferDataTypeWithConfidence(t *testing.T) {
	config := InferenceConfig{}
	engine := NewSchemaInferenceEngine(config)

	tests := []struct {
		name               string
		values             []interface{}
		expectedType       string
		expectedConfidence float64
	}{
		{
			name:               "All integers",
			values:             []interface{}{1, 2, 3, 4, 5},
			expectedType:       "integer",
			expectedConfidence: 1.0,
		},
		{
			name:               "All strings",
			values:             []interface{}{"a", "b", "c"},
			expectedType:       "string",
			expectedConfidence: 1.0,
		},
		{
			name:               "Mixed types - 50% strings, 50% integers",
			values:             []interface{}{"a", 1, "b", 2},
			expectedType:       "string", // Will have 50% confidence
			expectedConfidence: 0.5,
		},
		{
			name:               "All floats",
			values:             []interface{}{1.5, 2.7, 3.9},
			expectedType:       "float",
			expectedConfidence: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dataType, ontologyType, confidence := engine.inferDataTypeWithConfidence(tt.values)
			assert.Equal(t, tt.expectedType, dataType)
			assert.NotEmpty(t, ontologyType)
			assert.Equal(t, tt.expectedConfidence, confidence, "Confidence should match expected")
		})
	}
}
