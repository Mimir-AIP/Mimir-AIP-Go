package AI

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// Common LLM Utilities and Data Structure Tests
// ============================================================================

// TestLLMRequest_Structure tests LLM request structure
func TestLLMRequest_Structure(t *testing.T) {
	request := LLMRequest{
		Messages: []LLMMessage{
			{Role: "system", Content: "You are helpful"},
			{Role: "user", Content: "Hello"},
		},
		Model:       "gpt-4",
		Temperature: 0.7,
		MaxTokens:   1000,
		SystemMsg:   "System message",
		Tools: []LLMTool{
			{
				Name:        "test_tool",
				Description: "A test tool",
				Parameters:  map[string]any{"type": "object"},
			},
		},
		Metadata: map[string]any{"key": "value"},
	}

	assert.Len(t, request.Messages, 2)
	assert.Equal(t, "gpt-4", request.Model)
	assert.InDelta(t, 0.7, request.Temperature, 0.001)
	assert.Equal(t, 1000, request.MaxTokens)
	assert.Equal(t, "System message", request.SystemMsg)
	assert.Len(t, request.Tools, 1)
	assert.NotNil(t, request.Metadata)
}

// TestLLMResponse_Structure tests LLM response structure
func TestLLMResponse_Structure(t *testing.T) {
	response := LLMResponse{
		Content:      "Test response",
		ToolCalls:    []LLMToolCall{{ID: "call_1", Name: "tool1", Arguments: map[string]any{}}},
		FinishReason: "stop",
		Usage: &LLMUsage{
			PromptTokens:     10,
			CompletionTokens: 5,
			TotalTokens:      15,
		},
		Model:    "gpt-4",
		Metadata: map[string]any{"key": "value"},
	}

	assert.Equal(t, "Test response", response.Content)
	assert.Len(t, response.ToolCalls, 1)
	assert.Equal(t, "stop", response.FinishReason)
	assert.NotNil(t, response.Usage)
	assert.Equal(t, 15, response.Usage.TotalTokens)
	assert.Equal(t, "gpt-4", response.Model)
}

// TestLLMMessage_Structure tests LLM message structure
func TestLLMMessage_Structure(t *testing.T) {
	msg := LLMMessage{
		Role:    "assistant",
		Content: "Hello there",
	}

	assert.Equal(t, "assistant", msg.Role)
	assert.Equal(t, "Hello there", msg.Content)
}

// TestLLMTool_Structure tests LLM tool structure
func TestLLMTool_Structure(t *testing.T) {
	tool := LLMTool{
		Name:        "get_weather",
		Description: "Get the weather",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"location": map[string]any{"type": "string"},
			},
			"required": []string{"location"},
		},
	}

	assert.Equal(t, "get_weather", tool.Name)
	assert.Equal(t, "Get the weather", tool.Description)
	assert.NotNil(t, tool.Parameters)
}

// TestLLMToolCall_Structure tests LLM tool call structure
func TestLLMToolCall_Structure(t *testing.T) {
	toolCall := LLMToolCall{
		ID:   "call_123",
		Name: "get_weather",
		Arguments: map[string]any{
			"location": "San Francisco",
		},
	}

	assert.Equal(t, "call_123", toolCall.ID)
	assert.Equal(t, "get_weather", toolCall.Name)
	assert.Equal(t, "San Francisco", toolCall.Arguments["location"])
}

// TestLLMUsage_Structure tests LLM usage structure
func TestLLMUsage_Structure(t *testing.T) {
	usage := LLMUsage{
		PromptTokens:     50,
		CompletionTokens: 25,
		TotalTokens:      75,
	}

	assert.Equal(t, 50, usage.PromptTokens)
	assert.Equal(t, 25, usage.CompletionTokens)
	assert.Equal(t, 75, usage.TotalTokens)
}

// TestLLMClientConfig_Structure tests client config structure
func TestLLMClientConfig_Structure(t *testing.T) {
	config := LLMClientConfig{
		Provider:    ProviderOpenAI,
		APIKey:      "test-key",
		BaseURL:     "https://api.openai.com/v1",
		Model:       "gpt-4",
		Temperature: 0.5,
		MaxTokens:   2000,
		Timeout:     60,
		Metadata:    map[string]any{"custom": "value"},
	}

	assert.Equal(t, ProviderOpenAI, config.Provider)
	assert.Equal(t, "test-key", config.APIKey)
	assert.Equal(t, "https://api.openai.com/v1", config.BaseURL)
	assert.Equal(t, "gpt-4", config.Model)
	assert.InDelta(t, 0.5, config.Temperature, 0.001)
	assert.Equal(t, 2000, config.MaxTokens)
	assert.Equal(t, 60, config.Timeout)
}

// TestGetAvailableModelsForProvider tests model retrieval
func TestGetAvailableModelsForProvider(t *testing.T) {
	// Test OpenAI
	openAIModels := GetAvailableModelsForProvider(ProviderOpenAI)
	assert.NotEmpty(t, openAIModels, "OpenAI should have models")
	assert.Contains(t, openAIModels, "gpt-4o")

	// Test Anthropic
	anthropicModels := GetAvailableModelsForProvider(ProviderAnthropic)
	assert.NotEmpty(t, anthropicModels, "Anthropic should have models")

	// Test OpenRouter
	openRouterModels := GetAvailableModelsForProvider(ProviderOpenRouter)
	// OpenRouter models are fetched dynamically
	assert.NotNil(t, openRouterModels)

	// Test invalid provider
	invalidModels := GetAvailableModelsForProvider("invalid")
	assert.Empty(t, invalidModels, "Invalid provider should return empty")
}

// TestGetProviderInfo tests provider info retrieval
func TestGetProviderInfo(t *testing.T) {
	info := GetProviderInfo(ProviderOpenAI)
	assert.NotNil(t, info)
	assert.Equal(t, ProviderOpenAI, info["provider"])
	assert.Equal(t, "openai", info["name"])
	assert.NotEmpty(t, info["description"])
	assert.NotEmpty(t, info["default_model"])

	info2 := GetProviderInfo(ProviderAnthropic)
	assert.NotNil(t, info2)
	assert.Equal(t, "Anthropic Claude models", info2["description"])

	// Test all providers
	providers := []LLMProvider{
		ProviderOpenAI, ProviderAnthropic, ProviderOpenRouter,
		ProviderOllama, ProviderLocal, ProviderMock,
	}

	for _, provider := range providers {
		info := GetProviderInfo(provider)
		assert.NotNil(t, info, "Provider %s should have info", provider)
		assert.NotEmpty(t, info["description"], "Provider %s should have description", provider)
	}
}

// TestNewLLMClient tests LLM client factory
func TestNewLLMClient(t *testing.T) {
	// Test OpenAI
	client, err := NewLLMClient(LLMClientConfig{
		Provider: ProviderOpenAI,
		APIKey:   "test-key",
	})
	require.NoError(t, err)
	assert.Equal(t, ProviderOpenAI, client.GetProvider())

	// Test Anthropic
	client2, err := NewLLMClient(LLMClientConfig{
		Provider: ProviderAnthropic,
		APIKey:   "test-key",
	})
	require.NoError(t, err)
	assert.Equal(t, ProviderAnthropic, client2.GetProvider())

	// Test Mock
	client3, err := NewLLMClient(LLMClientConfig{
		Provider: ProviderMock,
	})
	require.NoError(t, err)
	assert.Equal(t, ProviderMock, client3.GetProvider())

	// Test invalid provider
	_, err = NewLLMClient(LLMClientConfig{
		Provider: "invalid",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported")
}

// TestLLMClientFactory tests client factory
func TestLLMClientFactory(t *testing.T) {
	factory := NewLLMClientFactory()
	require.NotNil(t, factory)

	// Create mock clients using the existing MockLLMClient from llm_stubs.go
	mockClient1 := NewMockLLMClient("Mock response")
	mockClient2 := NewMockLLMClient("Mock response")

	// Register clients
	factory.RegisterClient("client1", mockClient1)
	factory.RegisterClient("client2", mockClient2)

	// First registered becomes default
	assert.Equal(t, mockClient1, factory.GetDefaultClient())

	// Get by name
	client, err := factory.GetClient("client1")
	require.NoError(t, err)
	assert.Equal(t, mockClient1, client)

	client2, err := factory.GetClient("client2")
	require.NoError(t, err)
	assert.Equal(t, mockClient2, client2)

	// Get non-existent
	_, err = factory.GetClient("nonexistent")
	assert.Error(t, err)

	// Set default
	err = factory.SetDefaultClient("client2")
	require.NoError(t, err)
	assert.Equal(t, mockClient2, factory.GetDefaultClient())

	// Set non-existent as default
	err = factory.SetDefaultClient("nonexistent")
	assert.Error(t, err)
}

// BenchmarkOpenAIClient_Complete benchmarks the OpenAI client
func BenchmarkOpenAIClient_Complete(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := openAIResponseBody{
			Choices: []struct {
				Message struct {
					Role      string `json:"role"`
					Content   string `json:"content"`
					ToolCalls []struct {
						ID       string `json:"id"`
						Type     string `json:"type"`
						Function struct {
							Name      string `json:"name"`
							Arguments string `json:"arguments"`
						} `json:"function"`
					} `json:"tool_calls,omitempty"`
				} `json:"message"`
				FinishReason string `json:"finish_reason"`
			}{
				{
					Message: struct {
						Role      string `json:"role"`
						Content   string `json:"content"`
						ToolCalls []struct {
							ID       string `json:"id"`
							Type     string `json:"type"`
							Function struct {
								Name      string `json:"name"`
								Arguments string `json:"arguments"`
							} `json:"function"`
						} `json:"tool_calls,omitempty"`
					}{
						Content: "Benchmark response",
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client, _ := NewOpenAIClient(LLMClientConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
	})

	ctx := context.Background()
	request := LLMRequest{
		Messages: []LLMMessage{{Role: "user", Content: "Hello"}},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := client.Complete(ctx, request)
		if err != nil {
			b.Fatalf("Request failed: %v", err)
		}
	}
}
