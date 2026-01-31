package AI

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// OpenAI Client Tests
// ============================================================================

// TestOpenAIClient_New tests OpenAI client creation
func TestOpenAIClient_New(t *testing.T) {
	// Test with missing API key
	_, err := NewOpenAIClient(LLMClientConfig{})
	assert.Error(t, err, "Should error without API key")
	assert.Contains(t, err.Error(), "API key is required")

	// Test with API key
	client, err := NewOpenAIClient(LLMClientConfig{
		APIKey: "test-api-key",
		Model:  "gpt-4o-mini",
	})
	require.NoError(t, err, "Should create client with API key")
	require.NotNil(t, client, "Client should not be nil")

	// Test defaults
	assert.Equal(t, "gpt-4o-mini", client.GetDefaultModel())
	assert.Equal(t, ProviderOpenAI, client.GetProvider())
}

// TestOpenAIClient_New_WithDefaults tests default configuration
func TestOpenAIClient_New_WithDefaults(t *testing.T) {
	client, err := NewOpenAIClient(LLMClientConfig{
		APIKey: "test-key",
	})
	require.NoError(t, err)

	// Verify defaults
	assert.Equal(t, "gpt-4o-mini", client.GetDefaultModel(), "Should have default model")
	assert.Equal(t, ProviderOpenAI, client.GetProvider())

	// Validate config
	err = client.ValidateConfig()
	assert.NoError(t, err, "Config should be valid")
}

// TestOpenAIClient_Complete tests OpenAI completion with mock server
func TestOpenAIClient_Complete(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/chat/completions", r.URL.Path)
		assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// Parse request body
		var reqBody openAIRequestBody
		err := json.NewDecoder(r.Body).Decode(&reqBody)
		require.NoError(t, err)
		assert.NotEmpty(t, reqBody.Model)
		assert.NotEmpty(t, reqBody.Messages)

		// Send response
		response := openAIResponseBody{
			ID: "test-response-id",
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
						Content: "This is a test response",
					},
					FinishReason: "stop",
				},
			},
			Usage: struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
				TotalTokens      int `json:"total_tokens"`
			}{
				PromptTokens:     10,
				CompletionTokens: 5,
				TotalTokens:      15,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create client with mock server
	client, err := NewOpenAIClient(LLMClientConfig{
		APIKey:  "test-api-key",
		BaseURL: server.URL,
		Model:   "gpt-4o-mini",
	})
	require.NoError(t, err)

	// Test completion
	ctx := context.Background()
	request := LLMRequest{
		Messages: []LLMMessage{
			{Role: "user", Content: "Hello"},
		},
	}

	response, err := client.Complete(ctx, request)
	require.NoError(t, err, "Should complete successfully")
	require.NotNil(t, response, "Response should not be nil")

	assert.Equal(t, "This is a test response", response.Content)
	assert.Equal(t, "stop", response.FinishReason)
	assert.Equal(t, "gpt-4o-mini", response.Model)
	assert.NotNil(t, response.Usage)
	assert.Equal(t, 10, response.Usage.PromptTokens)
	assert.Equal(t, 5, response.Usage.CompletionTokens)
	assert.Equal(t, 15, response.Usage.TotalTokens)
}

// TestOpenAIClient_Complete_Error tests error handling
func TestOpenAIClient_Complete_Error(t *testing.T) {
	// Create mock server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "Invalid API key"}`))
	}))
	defer server.Close()

	client, err := NewOpenAIClient(LLMClientConfig{
		APIKey:  "invalid-key",
		BaseURL: server.URL,
	})
	require.NoError(t, err)

	ctx := context.Background()
	request := LLMRequest{
		Messages: []LLMMessage{
			{Role: "user", Content: "Hello"},
		},
	}

	_, err = client.Complete(ctx, request)
	assert.Error(t, err, "Should return error for 401")
	assert.Contains(t, err.Error(), "401")
}

// TestOpenAIClient_CompleteSimple tests simple completion
func TestOpenAIClient_CompleteSimple(t *testing.T) {
	// Create mock server
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
						Content: "Simple response",
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client, err := NewOpenAIClient(LLMClientConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
	})
	require.NoError(t, err)

	ctx := context.Background()
	response, err := client.CompleteSimple(ctx, "Hello")
	require.NoError(t, err)
	assert.Equal(t, "Simple response", response)
}

// TestOpenAIClient_Complete_WithTools tests completion with tools
func TestOpenAIClient_Complete_WithTools(t *testing.T) {
	// Create mock server that returns tool calls
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
						Content: "",
						ToolCalls: []struct {
							ID       string `json:"id"`
							Type     string `json:"type"`
							Function struct {
								Name      string `json:"name"`
								Arguments string `json:"arguments"`
							} `json:"function"`
						}{
							{
								ID:   "call_123",
								Type: "function",
								Function: struct {
									Name      string `json:"name"`
									Arguments string `json:"arguments"`
								}{
									Name:      "get_weather",
									Arguments: `{"location": "San Francisco"}`,
								},
							},
						},
					},
					FinishReason: "tool_calls",
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client, err := NewOpenAIClient(LLMClientConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
	})
	require.NoError(t, err)

	ctx := context.Background()
	request := LLMRequest{
		Messages: []LLMMessage{
			{Role: "user", Content: "What's the weather?"},
		},
		Tools: []LLMTool{
			{
				Name:        "get_weather",
				Description: "Get weather for a location",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"location": map[string]any{"type": "string"},
					},
				},
			},
		},
	}

	response, err := client.Complete(ctx, request)
	require.NoError(t, err)
	assert.Equal(t, "tool_calls", response.FinishReason)
	assert.Len(t, response.ToolCalls, 1)
	assert.Equal(t, "get_weather", response.ToolCalls[0].Name)
	assert.Equal(t, "call_123", response.ToolCalls[0].ID)
}

// TestClient_Timeout tests timeout handling
func TestClient_Timeout(t *testing.T) {
	// Create slow server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := NewOpenAIClient(LLMClientConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Timeout: 1, // 1 second timeout
	})
	require.NoError(t, err)

	ctx := context.Background()
	request := LLMRequest{
		Messages: []LLMMessage{{Role: "user", Content: "Hello"}},
	}

	start := time.Now()
	_, err = client.Complete(ctx, request)
	elapsed := time.Since(start)

	assert.Error(t, err, "Should timeout")
	assert.Less(t, elapsed, 1500*time.Millisecond, "Should fail quickly due to timeout")
}
