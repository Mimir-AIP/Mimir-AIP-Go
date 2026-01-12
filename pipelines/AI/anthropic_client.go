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

const (
	anthropicAPIURL  = "https://api.anthropic.com/v1"
	anthropicVersion = "2023-06-01"
)

type AnthropicClient struct {
	apiKey      string
	baseURL     string
	model       string
	temperature float64
	maxTokens   int
	timeout     time.Duration
	client      *http.Client
}

type anthropicRequestBody struct {
	Model       string                   `json:"model"`
	Messages    []anthropicMessage       `json:"messages"`
	MaxTokens   int                      `json:"max_tokens,omitempty"`
	Temperature float64                  `json:"temperature,omitempty"`
	System      string                   `json:"system,omitempty"`
	Tools       []map[string]interface{} `json:"tools,omitempty"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicResponseBody struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Role    string `json:"role"`
	Content []struct {
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
	} `json:"content"`
	StopReason string `json:"stop_reason"`
	Usage      struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

func NewAnthropicClient(config LLMClientConfig) (LLMClient, error) {
	apiKey := config.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("Anthropic API key is required")
	}

	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = anthropicAPIURL
	}

	model := config.Model
	if model == "" {
		model = "claude-sonnet-4-20250514"
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

	return &AnthropicClient{
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

func (c *AnthropicClient) Complete(ctx context.Context, request LLMRequest) (*LLMResponse, error) {
	messages := make([]anthropicMessage, 0, len(request.Messages))

	systemMsg := request.SystemMsg
	for _, msg := range request.Messages {
		if msg.Role == "system" {
			systemMsg = msg.Content
		} else {
			messages = append(messages, anthropicMessage{
				Role:    msg.Role,
				Content: msg.Content,
			})
		}
	}

	model := request.Model
	if model == "" {
		model = c.model
	}

	temperature := request.Temperature
	if temperature == 0 {
		temperature = c.temperature
	}

	maxTokens := request.MaxTokens
	if maxTokens == 0 {
		maxTokens = c.maxTokens
	}

	reqBody := anthropicRequestBody{
		Model:       model,
		Messages:    messages,
		MaxTokens:   maxTokens,
		Temperature: temperature,
	}
	if systemMsg != "" {
		reqBody.System = systemMsg
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/messages", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.apiKey)
	httpReq.Header.Set("anthropic-version", anthropicVersion)

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Anthropic API error (status %d): %s", resp.StatusCode, string(body))
	}

	var apiResp anthropicResponseBody
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(apiResp.Content) == 0 {
		return nil, fmt.Errorf("no content in response")
	}

	response := &LLMResponse{
		Model: model,
		Usage: &LLMUsage{
			PromptTokens:     apiResp.Usage.InputTokens,
			CompletionTokens: apiResp.Usage.OutputTokens,
			TotalTokens:      apiResp.Usage.InputTokens + apiResp.Usage.OutputTokens,
		},
	}

	for _, content := range apiResp.Content {
		if content.Type == "text" {
			response.Content += content.Text
		} else if content.Type == "tool_use" {
			response.ToolCalls = append(response.ToolCalls, LLMToolCall{
				ID:   content.ToolUse.ID,
				Name: content.ToolUse.Name,
			})
		}
	}

	response.FinishReason = apiResp.StopReason

	return response, nil
}

func (c *AnthropicClient) CompleteSimple(ctx context.Context, prompt string) (string, error) {
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

func (c *AnthropicClient) GetProvider() LLMProvider {
	return ProviderAnthropic
}

func (c *AnthropicClient) GetDefaultModel() string {
	return c.model
}

func (c *AnthropicClient) ValidateConfig() error {
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
