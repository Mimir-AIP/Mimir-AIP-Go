package AI

import (
	"context"
	"fmt"
)

// NewAnthropicClient creates a new Anthropic client (stub for now)
func NewAnthropicClient(config LLMClientConfig) (LLMClient, error) {
	return nil, fmt.Errorf("Anthropic client not yet implemented - coming soon")
}

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

func (m *MockLLMClient) Complete(ctx context.Context, request LLMRequest) (*LLMResponse, error) {
	var content string

	if m.intelligent {
		// Generate intelligent response based on context
		lastMessage := ""
		if len(request.Messages) > 0 {
			lastMessage = request.Messages[len(request.Messages)-1].Content
		}
		content = m.generateIntelligentResponse(lastMessage)
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

	return &LLMResponse{
		Content:      content,
		FinishReason: "stop",
		Model:        m.model,
		Usage: &LLMUsage{
			PromptTokens:     promptTokens,
			CompletionTokens: completionTokens,
			TotalTokens:      promptTokens + completionTokens,
		},
	}, nil
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

	// Digital Twin responses
	if containsStr(msgLower, "digital twin") || containsStr(msgLower, "twin") {
		if containsStr(msgLower, "create") {
			return "I'll help you create a new digital twin. Please specify the system type and initial parameters."
		}
		if containsStr(msgLower, "scenario") {
			return "I can create a scenario for your digital twin. Would you like to simulate a supply disruption, demand spike, or resource constraint?"
		}
		if containsStr(msgLower, "simulation") {
			return "I'll run the simulation. This will take about 30-60 seconds. The results will show predicted system behavior and any bottlenecks."
		}
		return "I'm your Digital Twin assistant. I can help create twins, scenarios, and run simulations. What would you like to do?"
	}

	// ML/Data responses
	if containsStr(msgLower, "train") || containsStr(msgLower, "model") {
		return "I can train a machine learning model on your data. I recommend starting with a decision tree classifier. Would you like me to proceed?"
	}

	if containsStr(msgLower, "data") || containsStr(msgLower, "upload") {
		return "I can help with data ingestion. The system supports CSV, JSON, and Parquet formats. You can upload files or configure automated pipelines."
	}

	// Ontology responses
	if containsStr(msgLower, "ontology") || containsStr(msgLower, "schema") {
		return "I'll help you create an ontology. Would you like to start with a template (manufacturing, healthcare, finance) or build custom?"
	}

	// Pipeline responses
	if containsStr(msgLower, "pipeline") {
		return "I can help you create a data pipeline. What's your source format and what processing do you need?"
	}

	// Help/general
	if containsStr(msgLower, "help") || containsStr(msgLower, "what can") {
		return "I'm Mimir, your AI assistant. I can help with digital twins, data pipelines, ML training, ontology creation, and job scheduling. What would you like to start with?"
	}

	// Default response
	return "I understand your request. I'm here to help with digital twins, data operations, and ML workflows. Could you provide more specific details about what you'd like to accomplish?"
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
