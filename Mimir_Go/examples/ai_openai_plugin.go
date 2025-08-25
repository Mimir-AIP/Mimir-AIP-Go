// OpenAI AI Models Plugin Example
// This example shows how to create an AIModels plugin for Mimir AIP

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
)

// OpenAIPlugin implements OpenAI API integration
type OpenAIPlugin struct {
	name    string
	version string
	apiKey  string
	baseURL string
	client  *http.Client
}

// OpenAIMessage represents a chat message
type OpenAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OpenAIRequest represents a request to OpenAI API
type OpenAIRequest struct {
	Model       string          `json:"model"`
	Messages    []OpenAIMessage `json:"messages"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Temperature float64         `json:"temperature,omitempty"`
}

// OpenAIChoice represents a choice in the OpenAI response
type OpenAIChoice struct {
	Message      OpenAIMessage `json:"message"`
	FinishReason string        `json:"finish_reason"`
}

// OpenAIResponse represents the response from OpenAI API
type OpenAIResponse struct {
	ID      string                 `json:"id"`
	Object  string                 `json:"object"`
	Created int64                  `json:"created"`
	Choices []OpenAIChoice         `json:"choices"`
	Usage   map[string]interface{} `json:"usage"`
}

// NewOpenAIPlugin creates a new OpenAI plugin instance
func NewOpenAIPlugin() *OpenAIPlugin {
	return &OpenAIPlugin{
		name:    "OpenAIPlugin",
		version: "1.0.0",
		baseURL: "https://api.openai.com/v1",
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// ExecuteStep executes AI model operations
func (p *OpenAIPlugin) ExecuteStep(ctx context.Context, stepConfig pipelines.StepConfig, globalContext pipelines.PluginContext) (pipelines.PluginContext, error) {
	fmt.Printf("Executing %s step: %s\n", p.name, stepConfig.Name)

	config := stepConfig.Config

	// Validate configuration
	if err := p.ValidateConfig(config); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	// Get API key from config or environment
	apiKey := p.getAPIKey(config)
	if apiKey == "" {
		return nil, fmt.Errorf("OpenAI API key is required")
	}
	p.apiKey = apiKey

	// Get operation type
	operation, _ := config["operation"].(string)
	if operation == "" {
		operation = "chat" // default
	}

	// Execute operation
	switch operation {
	case "chat":
		return p.executeChat(ctx, config, stepConfig.Output)
	case "completion":
		return p.executeCompletion(ctx, config, stepConfig.Output)
	default:
		return nil, fmt.Errorf("unsupported operation: %s", operation)
	}
}

// GetPluginType returns the plugin type
func (p *OpenAIPlugin) GetPluginType() string {
	return "AIModels"
}

// GetPluginName returns the plugin name
func (p *OpenAIPlugin) GetPluginName() string {
	return "openai"
}

// ValidateConfig validates the plugin configuration
func (p *OpenAIPlugin) ValidateConfig(config map[string]interface{}) error {
	operation, _ := config["operation"].(string)
	if operation == "" {
		operation = "chat"
	}

	switch operation {
	case "chat":
		if config["messages"] == nil && config["prompt"] == nil {
			return fmt.Errorf("messages or prompt is required for chat operation")
		}
	case "completion":
		if config["prompt"] == nil {
			return fmt.Errorf("prompt is required for completion operation")
		}
	}

	return nil
}

// getAPIKey retrieves the API key from config or environment
func (p *OpenAIPlugin) getAPIKey(config map[string]interface{}) string {
	// Check config first
	if key, ok := config["api_key"].(string); ok && key != "" {
		return key
	}

	// Check environment variable
	// Note: In production, use proper environment variable access
	// For this example, we'll expect it to be passed in config
	return ""
}

// executeChat executes a chat completion
func (p *OpenAIPlugin) executeChat(ctx context.Context, config map[string]interface{}, outputKey string) (pipelines.PluginContext, error) {
	// Build messages
	var messages []OpenAIMessage

	// Check if messages are provided directly
	if msgs, ok := config["messages"].([]interface{}); ok {
		for _, msg := range msgs {
			if msgMap, ok := msg.(map[string]interface{}); ok {
				role, _ := msgMap["role"].(string)
				content, _ := msgMap["content"].(string)
				messages = append(messages, OpenAIMessage{
					Role:    role,
					Content: content,
				})
			}
		}
	} else if prompt, ok := config["prompt"].(string); ok {
		// Convert single prompt to user message
		messages = []OpenAIMessage{
			{Role: "user", Content: prompt},
		}
	}

	// Get model
	model, _ := config["model"].(string)
	if model == "" {
		model = "gpt-3.5-turbo"
	}

	// Get parameters
	maxTokens := 1000
	if mt, ok := config["max_tokens"].(float64); ok {
		maxTokens = int(mt)
	}

	temperature := 0.7
	if temp, ok := config["temperature"].(float64); ok {
		temperature = temp
	}

	// Create request
	request := OpenAIRequest{
		Model:       model,
		Messages:    messages,
		MaxTokens:   maxTokens,
		Temperature: temperature,
	}

	// Execute request
	response, err := p.makeAPIRequest(ctx, "chat/completions", request)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}

	// Parse response
	var openaiResp OpenAIResponse
	if err := json.Unmarshal(response, &openaiResp); err != nil {
		return nil, fmt.Errorf("failed to parse API response: %w", err)
	}

	// Extract content
	var content string
	if len(openaiResp.Choices) > 0 {
		content = openaiResp.Choices[0].Message.Content
	}

	// Build result
	result := map[string]interface{}{
		"content":       content,
		"model":         openaiResp.Object,
		"usage":         openaiResp.Usage,
		"finish_reason": openaiResp.Choices[0].FinishReason,
		"request_id":    openaiResp.ID,
		"timestamp":     time.Now().Format(time.RFC3339),
	}

	return pipelines.PluginContext{
		outputKey: result,
	}, nil
}

// executeCompletion executes a text completion (legacy)
func (p *OpenAIPlugin) executeCompletion(ctx context.Context, config map[string]interface{}, outputKey string) (pipelines.PluginContext, error) {
	// Implementation for completion API
	// This would be similar to chat but use the completions endpoint
	return nil, fmt.Errorf("completion operation not implemented in this example")
}

// makeAPIRequest makes a request to the OpenAI API
func (p *OpenAIPlugin) makeAPIRequest(ctx context.Context, endpoint string, payload interface{}) ([]byte, error) {
	// Serialize payload
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create request
	url := fmt.Sprintf("%s/%s", p.baseURL, endpoint)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.apiKey))

	// Make request
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: %d - %s", resp.StatusCode, string(body))
	}

	return body, nil
}

// Example usage in pipeline YAML:
//
// pipelines:
//   - name: "AI Pipeline"
//     steps:
//       - name: "Generate Content"
//         plugin: "AIModels.openai"
//         config:
//           operation: "chat"
//           model: "gpt-3.5-turbo"
//           messages:
//             - role: "user"
//               content: "Write a summary of artificial intelligence"
//           max_tokens: 500
//           temperature: 0.7
//           api_key: "your-api-key-here"
//         output: "ai_response"
//       - name: "Process Response"
//         plugin: "Data_Processing.transform"
//         config:
//           operation: "extract_pattern"
//           pattern: "summary: (.*)"
//           input: "ai_response"
//         output: "extracted_summary"
