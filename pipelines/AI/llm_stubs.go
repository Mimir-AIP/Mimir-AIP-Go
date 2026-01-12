package AI

import (
	"context"
	"fmt"
)

// NewOllamaClient creates a new Ollama client (stub for now)
func NewOllamaClient(config LLMClientConfig) (LLMClient, error) {
	return nil, fmt.Errorf("Ollama client not yet implemented - coming soon")
}

// NewAzureOpenAIClient creates a new Azure OpenAI client (stub for now)
func NewAzureOpenAIClient(config LLMClientConfig) (LLMClient, error) {
	return nil, fmt.Errorf("Azure OpenAI client not yet implemented - coming soon")
}

// NewGoogleClient creates a new Google (Gemini) client (stub for now)
func NewGoogleClient(config LLMClientConfig) (LLMClient, error) {
	return nil, fmt.Errorf("Google client not yet implemented - coming soon")
}

// MockLLMClient is an intelligent mock for cost-free testing
type MockLLMClient struct {
	provider    LLMProvider
	model       string
	response    string // Fixed response if set
	intelligent bool   // If true, generate context-aware responses
}

// NewMockLLMClient creates a mock LLM client with fixed response
func NewMockLLMClient(response string) LLMClient {
	return &MockLLMClient{
		provider:    "mock",
		model:       "mock-gpt-4",
		response:    response,
		intelligent: false,
	}
}

// NewIntelligentMockLLMClient creates a context-aware mock client for E2E testing
func NewIntelligentMockLLMClient() LLMClient {
	return &MockLLMClient{
		provider:    "mock",
		model:       "mock-gpt-4",
		intelligent: true,
	}
}

// NewIntelligentMockLLMClientWithModel creates a context-aware mock client with specified model
func NewIntelligentMockLLMClientWithModel(modelName string) LLMClient {
	return &MockLLMClient{
		provider:    "mock",
		model:       modelName,
		intelligent: true,
	}
}

func (m *MockLLMClient) Complete(ctx context.Context, request LLMRequest) (*LLMResponse, error) {
	var content string
	var toolCalls []LLMToolCall
	finishReason := "stop"

	if m.intelligent {
		// Generate intelligent response based on context
		lastMessage := ""
		if len(request.Messages) > 0 {
			lastMessage = request.Messages[len(request.Messages)-1].Content
		}

		// Check for TRIGGER_TOOL command
		if len(lastMessage) > 13 && lastMessage[:13] == "TRIGGER_TOOL:" {
			toolName := ""
			for i := 13; i < len(lastMessage); i++ {
				if lastMessage[i] != ' ' && lastMessage[i] != '\t' && lastMessage[i] != '\n' {
					toolName = lastMessage[i:]
					break
				}
			}
			// Trim trailing whitespace
			for len(toolName) > 0 && (toolName[len(toolName)-1] == ' ' || toolName[len(toolName)-1] == '\t' || toolName[len(toolName)-1] == '\n') {
				toolName = toolName[:len(toolName)-1]
			}

			if toolName != "" {
				toolCalls = m.generateToolCall(toolName)
				finishReason = "tool_calls"
				content = "" // No content when making tool calls
			}
		} else {
			content = m.generateIntelligentResponse(lastMessage)
		}
	} else {
		// Use fixed response
		content = m.response
	}

	// Count tokens (simple word-based approximation)
	promptTokens := 0
	for _, msg := range request.Messages {
		promptTokens += len(msg.Content) / 4 // Rough token estimate
	}
	completionTokens := len(content) / 4

	response := &LLMResponse{
		Content:      content,
		FinishReason: finishReason,
		Model:        m.model,
		Usage: &LLMUsage{
			PromptTokens:     promptTokens,
			CompletionTokens: completionTokens,
			TotalTokens:      promptTokens + completionTokens,
		},
	}

	if len(toolCalls) > 0 {
		response.ToolCalls = toolCalls
	}

	return response, nil
}

func (m *MockLLMClient) CompleteSimple(ctx context.Context, prompt string) (string, error) {
	response, err := m.Complete(ctx, LLMRequest{
		Messages: []LLMMessage{{Role: "user", Content: prompt}},
	})
	if err != nil {
		return "", err
	}
	return response.Content, nil
}

func (m *MockLLMClient) GetProvider() LLMProvider {
	return m.provider
}

func (m *MockLLMClient) GetDefaultModel() string {
	return m.model
}

func (m *MockLLMClient) ValidateConfig() error {
	return nil
}

// generateIntelligentResponse creates context-aware responses for testing
func (m *MockLLMClient) generateIntelligentResponse(message string) string {
	msgLower := toLower(message)

	// Check if this is Claude model for different personality
	isClaude := containsStr(m.model, "claude")

	// Digital Twin responses
	if containsStr(msgLower, "digital twin") || containsStr(msgLower, "twin") {
		if containsStr(msgLower, "create") {
			if isClaude {
				return "I'd be happy to help you create a new digital twin. Please specify the system type and initial parameters you have in mind."
			}
			return "I'll help you create a new digital twin. Please specify the system type and initial parameters."
		}
		if containsStr(msgLower, "scenario") {
			if isClaude {
				return "I'd be delighted to create a scenario for your digital twin. Would you like to simulate a supply disruption, demand spike, or resource constraint?"
			}
			return "I can create a scenario for your digital twin. Would you like to simulate a supply disruption, demand spike, or resource constraint?"
		}
		if containsStr(msgLower, "simulation") {
			if isClaude {
				return "I'll be happy to run the simulation for you. This will take approximately 30-60 seconds. The results will show predicted system behavior and identify any bottlenecks."
			}
			return "I'll run the simulation. This will take about 30-60 seconds. The results will show predicted system behavior and any bottlenecks."
		}
		if isClaude {
			return "I'm your Digital Twin assistant, and I'd be happy to help! I can assist with creating twins, scenarios, and running simulations. What would you like to do?"
		}
		return "I'm your Digital Twin assistant. I can help create twins, scenarios, and run simulations. What would you like to do?"
	}

	// ML/Data responses
	if containsStr(msgLower, "train") || containsStr(msgLower, "model") {
		if isClaude {
			return "I'd be happy to train a machine learning model on your data. I'd recommend starting with a decision tree classifier. Would you like me to proceed?"
		}
		return "I can train a machine learning model on your data. I recommend starting with a decision tree classifier. Would you like me to proceed?"
	}

	if containsStr(msgLower, "data") || containsStr(msgLower, "upload") {
		if isClaude {
			return "I'd be pleased to help with data ingestion. The system supports CSV, JSON, and Parquet formats. You can upload files or configure automated pipelines."
		}
		return "I can help with data ingestion. The system supports CSV, JSON, and Parquet formats. You can upload files or configure automated pipelines."
	}

	// Ontology responses
	if containsStr(msgLower, "ontology") || containsStr(msgLower, "schema") {
		if isClaude {
			return "I'd be happy to help you create an ontology. Would you like to start with a template (manufacturing, healthcare, finance) or build a custom one?"
		}
		return "I'll help you create an ontology. Would you like to start with a template (manufacturing, healthcare, finance) or build custom?"
	}

	// Pipeline responses
	if containsStr(msgLower, "pipeline") {
		if isClaude {
			return "I'd be glad to help you create a data pipeline. What's your source format and what processing would you like to do?"
		}
		return "I can help you create a data pipeline. What's your source format and what processing do you need?"
	}

	// Help/general
	if containsStr(msgLower, "help") || containsStr(msgLower, "what can") {
		if isClaude {
			return "I'm Mimir, your AI assistant, and I'd be happy to help! I can assist with digital twins, data pipelines, ML training, ontology creation, and job scheduling. What would you like to start with?"
		}
		return "I'm Mimir, your AI assistant. I can help with digital twins, data pipelines, ML training, ontology creation, and job scheduling. What would you like to start with?"
	}

	// Greetings
	if containsStr(msgLower, "hello") || containsStr(msgLower, "hi") {
		if isClaude {
			return "Hello! I'd be delighted to assist you today. I'm here to help with digital twins, data operations, and ML workflows. What can I help you with?"
		}
		return "Hello! I'm here to help with digital twins, data operations, and ML workflows. What can I do for you?"
	}

	// Default response
	if isClaude {
		return "I understand your request. I'd be happy to help with digital twins, data operations, and ML workflows. Could you provide more specific details about what you'd like to accomplish?"
	}
	return "I understand your request. I'm here to help with digital twins, data operations, and ML workflows. Could you provide more specific details about what you'd like to accomplish?"
}

// generateToolCall creates mock tool calls for testing
func (m *MockLLMClient) generateToolCall(toolName string) []LLMToolCall {
	toolNameLower := toLower(toolName)

	switch toolNameLower {
	case "input.csv":
		return []LLMToolCall{
			{
				ID:   "call_input_csv",
				Name: "Input.csv",
				Arguments: map[string]any{
					"step_config": map[string]any{
						"name": "read_csv",
						"config": map[string]any{
							"file_path":   "/tmp/test_products.csv",
							"delimiter":   ",",
							"has_headers": true,
						},
						"output": "csv_data",
					},
				},
			},
		}
	case "input.xml":
		return []LLMToolCall{
			{
				ID:   "call_input_xml",
				Name: "Input.xml",
				Arguments: map[string]any{
					"step_config": map[string]any{
						"name": "read_xml",
						"config": map[string]any{
							"file_path": "/tmp/test_data.xml",
						},
						"output": "xml_data",
					},
				},
			},
		}
	case "input.excel":
		return []LLMToolCall{
			{
				ID:   "call_input_excel",
				Name: "Input.excel",
				Arguments: map[string]any{
					"step_config": map[string]any{
						"name": "read_excel",
						"config": map[string]any{
							"file_path":   "/tmp/test_data.xlsx",
							"sheet_name":  "Sheet1",
							"has_headers": true,
						},
						"output": "excel_data",
					},
				},
			},
		}
	case "ontology.query":
		return []LLMToolCall{
			{
				ID:   "call_ontology_query",
				Name: "Ontology.query",
				Arguments: map[string]any{
					"step_config": map[string]any{
						"name": "query_ontology",
						"config": map[string]any{
							"ontology_id": "test-ontology",
							"question":    "What products are in stock?",
							"use_nl":      true,
						},
						"output": "query_results",
					},
				},
			},
		}
	case "ontology.management":
		return []LLMToolCall{
			{
				ID:   "call_ontology_mgmt",
				Name: "Ontology.management",
				Arguments: map[string]any{
					"step_config": map[string]any{
						"name": "list_ontologies",
						"config": map[string]any{
							"operation": "list",
						},
						"output": "ontologies",
					},
				},
			},
		}
	case "ontology.extract":
		return []LLMToolCall{
			{
				ID:   "call_extract",
				Name: "Ontology.extract",
				Arguments: map[string]any{
					"step_config": map[string]any{
						"name": "extract_entities",
						"config": map[string]any{
							"ontology_id": "test-ontology",
							"data":        "/tmp/test_products.csv",
							"source_type": "csv",
						},
						"output": "extracted_entities",
					},
				},
			},
		}
	case "create_scenario":
		return []LLMToolCall{
			{
				ID:   "call_1",
				Name: "create_scenario",
				Arguments: map[string]any{
					"scenario_type": "supply_disruption",
					"severity":      "high",
					"duration_days": 14,
					"description":   "Supply chain disruption test scenario",
				},
			},
		}
	case "run_simulation":
		return []LLMToolCall{
			{
				ID:   "call_2",
				Name: "run_simulation",
				Arguments: map[string]any{
					"simulation_type": "monte_carlo",
					"iterations":      1000,
					"time_horizon":    30,
				},
			},
		}
	case "query_ontology":
		return []LLMToolCall{
			{
				ID:   "call_3",
				Name: "query_ontology",
				Arguments: map[string]any{
					"query_type": "sparql",
					"query":      "SELECT ?s ?p ?o WHERE { ?s ?p ?o } LIMIT 10",
				},
			},
		}
	case "train_model":
		return []LLMToolCall{
			{
				ID:   "call_4",
				Name: "train_model",
				Arguments: map[string]any{
					"model_type": "decision_tree",
					"target":     "quality_score",
					"features":   []string{"temperature", "pressure", "humidity"},
					"test_split": 0.2,
					"max_depth":  10,
				},
			},
		}
	case "create_pipeline":
		return []LLMToolCall{
			{
				ID:   "call_5",
				Name: "create_pipeline",
				Arguments: map[string]any{
					"pipeline_name": "data_ingestion_pipeline",
					"source_type":   "csv",
					"schedule":      "daily",
				},
			},
		}
	case "analyze_data":
		return []LLMToolCall{
			{
				ID:   "call_6",
				Name: "analyze_data",
				Arguments: map[string]any{
					"dataset":       "products",
					"analysis_type": "profiling",
					"include_stats": true,
				},
			},
		}
	default:
		// Generic tool call for unknown tools
		return []LLMToolCall{
			{
				ID:   "call_generic",
				Name: toolName,
				Arguments: map[string]any{
					"action": "execute",
					"status": "pending",
				},
			},
		}
	}
}

// Helper function to convert string to lowercase
func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			result[i] = c + 32
		} else {
			result[i] = c
		}
	}
	return string(result)
}

// Helper function to check if string contains substring (case insensitive)
func containsStr(s, substr string) bool {
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
