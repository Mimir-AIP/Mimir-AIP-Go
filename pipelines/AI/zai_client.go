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
	zAiAPIURL       = "https://api.z-ai.com/v1"
	zAiCodingAPIURL = "https://api.z-ai.com/v1/coding"
	zAiDefaultModel = "z-ai-model"
	zAiCodingModel  = "z-ai-coding-model"
)

type ZAiClient struct {
	apiKey        string
	baseURL       string
	codingURL     string
	model         string
	codingModel   string
	temperature   float64
	maxTokens     int
	timeout       time.Duration
	client        *http.Client
	useCodingPlan bool
}

type zAiRequestBody struct {
	Model       string                   `json:"model"`
	Messages    []map[string]string      `json:"messages"`
	MaxTokens   int                      `json:"max_tokens,omitempty"`
	Temperature float64                  `json:"temperature,omitempty"`
	Tools       []map[string]interface{} `json:"tools,omitempty"`
}

type zAiResponseBody struct {
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

func NewZAiClient(config LLMClientConfig) (LLMClient, error) {
	apiKey := config.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("Z_AI_API_KEY")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("Z.ai API key is required")
	}

	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = zAiAPIURL
	}

	codingURL := zAiCodingAPIURL
	if urlStr, ok := config.Metadata["coding_url"].(string); ok && urlStr != "" {
		codingURL = urlStr
	}

	model := config.Model
	if model == "" {
		model = zAiDefaultModel
	}

	codingModel := zAiCodingModel
	if modelStr, ok := config.Metadata["coding_model"].(string); ok && modelStr != "" {
		codingModel = modelStr
	}

	useCodingPlan := false
	if codingPlan, ok := config.Metadata["use_coding_plan"].(bool); ok {
		useCodingPlan = codingPlan
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

	return &ZAiClient{
		apiKey:        apiKey,
		baseURL:       baseURL,
		codingURL:     codingURL,
		model:         model,
		codingModel:   codingModel,
		temperature:   temperature,
		maxTokens:     maxTokens,
		timeout:       timeout,
		useCodingPlan: useCodingPlan,
		client: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

func (c *ZAiClient) Complete(ctx context.Context, request LLMRequest) (*LLMResponse, error) {
	messages := make([]map[string]string, 0, len(request.Messages)+1)

	if request.SystemMsg != "" {
		messages = append(messages, map[string]string{
			"role":    "system",
			"content": request.SystemMsg,
		})
	}

	for _, msg := range request.Messages {
		messages = append(messages, map[string]string{
			"role":    msg.Role,
			"content": msg.Content,
		})
	}

	model := request.Model
	if model == "" {
		if c.useCodingPlan {
			model = c.codingModel
		} else {
			model = c.model
		}
	}

	temperature := request.Temperature
	if temperature == 0 {
		temperature = c.temperature
	}

	maxTokens := request.MaxTokens
	if maxTokens == 0 {
		maxTokens = c.maxTokens
	}

	reqBody := zAiRequestBody{
		Model:       model,
		Messages:    messages,
		MaxTokens:   maxTokens,
		Temperature: temperature,
	}

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

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	var url string
	if c.useCodingPlan {
		url = c.codingURL + "/chat/completions"
	} else {
		url = c.baseURL + "/chat/completions"
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

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
		return nil, fmt.Errorf("Z.ai API error (status %d): %s", resp.StatusCode, string(body))
	}

	var apiResp zAiResponseBody
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

func (c *ZAiClient) CompleteSimple(ctx context.Context, prompt string) (string, error) {
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

func (c *ZAiClient) GetProvider() LLMProvider {
	return ProviderZAi
}

func (c *ZAiClient) GetDefaultModel() string {
	if c.useCodingPlan {
		return c.codingModel
	}
	return c.model
}

func (c *ZAiClient) ValidateConfig() error {
	if c.apiKey == "" {
		return fmt.Errorf("API key is required")
	}
	if c.baseURL == "" {
		return fmt.Errorf("base URL is required")
	}
	if c.model == "" && c.codingModel == "" {
		return fmt.Errorf("model is required")
	}
	return nil
}
