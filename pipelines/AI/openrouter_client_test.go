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
// OpenRouter Client Tests
// ============================================================================

// TestOpenRouterClient_New tests OpenRouter client creation
func TestOpenRouterClient_New(t *testing.T) {
	// Test with missing API key
	_, err := NewOpenRouterClient(LLMClientConfig{})
	assert.Error(t, err, "Should error without API key")
	assert.Contains(t, err.Error(), "API key is required")

	// Test with API key
	client, err := NewOpenRouterClient(LLMClientConfig{
		APIKey: "test-api-key",
		Model:  "anthropic/claude-sonnet-4-20250514",
	})
	require.NoError(t, err, "Should create client with API key")
	require.NotNil(t, client, "Client should not be nil")

	assert.Equal(t, "anthropic/claude-sonnet-4-20250514", client.GetDefaultModel())
	assert.Equal(t, ProviderOpenRouter, client.GetProvider())
}

// TestOpenRouterClient_Complete tests OpenRouter completion with mock server
func TestOpenRouterClient_Complete(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/chat/completions", r.URL.Path)
		assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))
		assert.Equal(t, "https://mimir-aip.io", r.Header.Get("HTTP-Referer"))
		assert.Equal(t, "Mimir AIP", r.Header.Get("X-Title"))

		// Send response
		response := openRouterResponseBody{
			ID: "test-response",
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
						Role:    "assistant",
						Content: "OpenRouter test response",
					},
					FinishReason: "stop",
				},
			},
			Usage: struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
				TotalTokens      int `json:"total_tokens"`
			}{
				PromptTokens:     15,
				CompletionTokens: 8,
				TotalTokens:      23,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client, err := NewOpenRouterClient(LLMClientConfig{
		APIKey:  "test-api-key",
		BaseURL: server.URL,
		Model:   "anthropic/claude-sonnet-4-20250514",
	})
	require.NoError(t, err)

	ctx := context.Background()
	request := LLMRequest{
		Messages: []LLMMessage{
			{Role: "user", Content: "Hello via OpenRouter"},
		},
	}

	response, err := client.Complete(ctx, request)
	require.NoError(t, err, "Should complete successfully")
	assert.Equal(t, "OpenRouter test response", response.Content)
	assert.Equal(t, "stop", response.FinishReason)
	assert.NotNil(t, response.Usage)
}

// TestOpenRouterClient_CompleteSimple tests simple completion
func TestOpenRouterClient_CompleteSimple(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := openRouterResponseBody{
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
						Content: "Simple OpenRouter response",
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client, err := NewOpenRouterClient(LLMClientConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
	})
	require.NoError(t, err)

	ctx := context.Background()
	response, err := client.CompleteSimple(ctx, "Hello")
	require.NoError(t, err)
	assert.Equal(t, "Simple OpenRouter response", response)
}
