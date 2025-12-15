package AI

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// OpenAIClient implements LLMClient for OpenAI
type OpenAIClient struct {
	apiKey      string
	baseURL     string
	model       string
	temperature float64
	maxTokens   int
	timeout     time.Duration
	client      *http.Client
}

// openAIRequestBody represents the request body for OpenAI API
type openAIRequestBody struct {
	Model       string                   `json:"model"`
	Messages    []map[string]string      `json:"messages"`
	MaxTokens   int                      `json:"max_tokens,omitempty"`
	Temperature float64                  `json:"temperature,omitempty"`
	Tools       []map[string]interface{} `json:"tools,omitempty"`
}

// openAIResponseBody represents the response from OpenAI API
type openAIResponseBody struct {
	ID      string `json:"id"`
	Choices []struct {
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
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// NewOpenAIClient creates a new OpenAI client
func NewOpenAIClient(config LLMClientConfig) (LLMClient, error) {
	apiKey := config.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("OpenAI API key is required")
	}

	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	model := config.Model
	if model == "" {
		model = "gpt-4o-mini"
	}

	timeout := time.Duration(config.Timeout) * time.Second
	if timeout == 0 {
		timeout = 60 * time.Second
	}

	temperature := config.Temperature
	if temperature == 0 {
		temperature = 0.7
	}

	maxTokens := config.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}

	return &OpenAIClient{
		apiKey:      apiKey,
		baseURL:     baseURL,
		model:       model,
		temperature: temperature,
		maxTokens:   maxTokens,
		timeout:     timeout,
		client: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

// Complete sends a completion request to OpenAI
func (c *OpenAIClient) Complete(ctx context.Context, request LLMRequest) (*LLMResponse, error) {
	// Build messages
	messages := make([]map[string]string, 0, len(request.Messages)+1)

	// Add system message if provided
	if request.SystemMsg != "" {
		messages = append(messages, map[string]string{
			"role":    "system",
			"content": request.SystemMsg,
		})
	}

	// Add conversation messages
	for _, msg := range request.Messages {
		messages = append(messages, map[string]string{
			"role":    msg.Role,
			"content": msg.Content,
		})
	}

	// Determine model
	model := request.Model
	if model == "" {
		model = c.model
	}

	// Determine temperature
	temperature := request.Temperature
	if temperature == 0 {
		temperature = c.temperature
	}

	// Determine max tokens
	maxTokens := request.MaxTokens
	if maxTokens == 0 {
		maxTokens = c.maxTokens
	}

	// Build request body
	reqBody := openAIRequestBody{
		Model:       model,
		Messages:    messages,
		MaxTokens:   maxTokens,
		Temperature: temperature,
	}

	// Add tools if provided
	if len(request.Tools) > 0 {
		tools := make([]map[string]interface{}, len(request.Tools))
		for i, tool := range request.Tools {
			tools[i] = map[string]interface{}{
				"type": "function",
				"function": map[string]interface{}{
					"name":        tool.Name,
					"description": tool.Description,
					"parameters":  tool.Parameters,
				},
			}
		}
		reqBody.Tools = tools
	}

	// Marshal request
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	// Send request
	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OpenAI API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse response
	var apiResp openAIResponseBody
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(apiResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	choice := apiResp.Choices[0]
	response := &LLMResponse{
		Content:      choice.Message.Content,
		FinishReason: choice.FinishReason,
		Model:        model,
		Usage: &LLMUsage{
			PromptTokens:     apiResp.Usage.PromptTokens,
			CompletionTokens: apiResp.Usage.CompletionTokens,
			TotalTokens:      apiResp.Usage.TotalTokens,
		},
	}

	// Parse tool calls if present
	if len(choice.Message.ToolCalls) > 0 {
		response.ToolCalls = make([]LLMToolCall, len(choice.Message.ToolCalls))
		for i, tc := range choice.Message.ToolCalls {
			var args map[string]any
			if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
				return nil, fmt.Errorf("failed to parse tool call arguments: %w", err)
			}
			response.ToolCalls[i] = LLMToolCall{
				ID:        tc.ID,
				Name:      tc.Function.Name,
				Arguments: args,
			}
		}
	}

	return response, nil
}

// CompleteSimple sends a simple text completion request
func (c *OpenAIClient) CompleteSimple(ctx context.Context, prompt string) (string, error) {
	response, err := c.Complete(ctx, LLMRequest{
		Messages: []LLMMessage{
			{Role: "user", Content: prompt},
		},
	})
	if err != nil {
		return "", err
	}
	return response.Content, nil
}

// GetProvider returns the provider type
func (c *OpenAIClient) GetProvider() LLMProvider {
	return ProviderOpenAI
}

// GetDefaultModel returns the default model
func (c *OpenAIClient) GetDefaultModel() string {
	return c.model
}

// ValidateConfig validates the client configuration
func (c *OpenAIClient) ValidateConfig() error {
	if c.apiKey == "" {
		return fmt.Errorf("API key is required")
	}
	if c.baseURL == "" {
		return fmt.Errorf("base URL is required")
	}
	if c.model == "" {
		return fmt.Errorf("model is required")
	}
	return nil
}
