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
// Anthropic Client Tests
// ============================================================================

// TestAnthropicClient_New tests Anthropic client creation
func TestAnthropicClient_New(t *testing.T) {
	// Test with missing API key
	_, err := NewAnthropicClient(LLMClientConfig{})
	assert.Error(t, err, "Should error without API key")
	assert.Contains(t, err.Error(), "API key is required")

	// Test with API key
	client, err := NewAnthropicClient(LLMClientConfig{
		APIKey: "test-api-key",
		Model:  "claude-sonnet-4-20250514",
	})
	require.NoError(t, err, "Should create client with API key")
	require.NotNil(t, client, "Client should not be nil")

	assert.Equal(t, "claude-sonnet-4-20250514", client.GetDefaultModel())
	assert.Equal(t, ProviderAnthropic, client.GetProvider())
}

// TestAnthropicClient_Complete tests Anthropic completion with mock server
func TestAnthropicClient_Complete(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/messages", r.URL.Path)
		assert.Equal(t, "test-api-key", r.Header.Get("x-api-key"))
		assert.Equal(t, "2023-06-01", r.Header.Get("anthropic-version"))

		// Send response
		response := anthropicResponseBody{
			ID:   "test-response",
			Type: "message",
			Role: "assistant",
			Content: []struct {
				Type    string `json:"type"`
				Text    string `json:"text"`
				ToolUse *struct {
					ID    string `json:"id"`
					Name  string `json:"name"`
					Input any    `json:"input"`
				} `json:"tool_use,omitempty"`
				ToolResult *struct {
					ToolUseID string `json:"tool_use_id"`
					Content   string `json:"content"`
				} `json:"tool_result,omitempty"`
			}{
				{Type: "text", Text: "Anthropic test response"},
			},
			StopReason: "end_turn",
			Usage: struct {
				InputTokens  int `json:"input_tokens"`
				OutputTokens int `json:"output_tokens"`
			}{
				InputTokens:  20,
				OutputTokens: 10,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client, err := NewAnthropicClient(LLMClientConfig{
		APIKey:  "test-api-key",
		BaseURL: server.URL,
		Model:   "claude-sonnet-4-20250514",
	})
	require.NoError(t, err)

	ctx := context.Background()
	request := LLMRequest{
		Messages: []LLMMessage{
			{Role: "user", Content: "Hello Claude"},
		},
		SystemMsg: "You are a helpful assistant",
	}

	response, err := client.Complete(ctx, request)
	require.NoError(t, err, "Should complete successfully")
	assert.Equal(t, "Anthropic test response", response.Content)
	assert.Equal(t, "end_turn", response.FinishReason)
	assert.NotNil(t, response.Usage)
	assert.Equal(t, 20, response.Usage.PromptTokens)
	assert.Equal(t, 10, response.Usage.CompletionTokens)
	assert.Equal(t, 30, response.Usage.TotalTokens)
}

// TestAnthropicClient_CompleteSimple tests simple completion
func TestAnthropicClient_CompleteSimple(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := anthropicResponseBody{
			Content: []struct {
				Type    string `json:"type"`
				Text    string `json:"text"`
				ToolUse *struct {
					ID    string `json:"id"`
					Name  string `json:"name"`
					Input any    `json:"input"`
				} `json:"tool_use,omitempty"`
				ToolResult *struct {
					ToolUseID string `json:"tool_use_id"`
					Content   string `json:"content"`
				} `json:"tool_result,omitempty"`
			}{
				{Type: "text", Text: "Simple Anthropic response"},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client, err := NewAnthropicClient(LLMClientConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
	})
	require.NoError(t, err)

	ctx := context.Background()
	response, err := client.CompleteSimple(ctx, "Hello")
	require.NoError(t, err)
	assert.Equal(t, "Simple Anthropic response", response)
}
