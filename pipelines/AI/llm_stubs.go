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

// MockLLMClient is a simple mock for testing
type MockLLMClient struct {
	provider LLMProvider
	model    string
	response string
}

// NewMockLLMClient creates a mock LLM client
func NewMockLLMClient(response string) LLMClient {
	return &MockLLMClient{
		provider: "mock",
		model:    "mock-model",
		response: response,
	}
}

func (m *MockLLMClient) Complete(ctx context.Context, request LLMRequest) (*LLMResponse, error) {
	return &LLMResponse{
		Content:      m.response,
		FinishReason: "stop",
		Model:        m.model,
		Usage: &LLMUsage{
			PromptTokens:     100,
			CompletionTokens: 50,
			TotalTokens:      150,
		},
	}, nil
}

func (m *MockLLMClient) CompleteSimple(ctx context.Context, prompt string) (string, error) {
	return m.response, nil
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
