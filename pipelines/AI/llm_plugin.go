package AI

import (
	"context"
	"fmt"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
)

// LLMPluginConfig holds configuration for an LLM plugin
type LLMPluginConfig struct {
	APIKey        string  `json:"api_key,omitempty"`
	BaseURL       string  `json:"base_url,omitempty"`
	Model         string  `json:"model"`
	Temperature   float64 `json:"temperature,omitempty"`
	MaxTokens     int     `json:"max_tokens,omitempty"`
	UseCodingPlan bool    `json:"use_coding_plan,omitempty"` // For Z.ai only
}

// LLMPlugin wraps an LLM client as a pipeline plugin with configuration support
type LLMPlugin struct {
	client   LLMClient
	provider LLMProvider
	config   LLMPluginConfig
}

// NewLLMPlugin creates a new LLM plugin from a client
func NewLLMPlugin(client LLMClient, provider LLMProvider) *LLMPlugin {
	return &LLMPlugin{
		client:   client,
		provider: provider,
		config: LLMPluginConfig{
			Model: client.GetDefaultModel(),
		},
	}
}

// NewLLMPluginWithConfig creates a new LLM plugin with configuration
func NewLLMPluginWithConfig(provider LLMProvider, config LLMPluginConfig) (*LLMPlugin, error) {
	metadata := map[string]any{}
	if config.UseCodingPlan {
		metadata["use_coding_plan"] = true
	}

	client, err := NewLLMClient(LLMClientConfig{
		Provider: provider,
		APIKey:   config.APIKey,
		BaseURL:  config.BaseURL,
		Model:    config.Model,
		Metadata: metadata,
	})
	if err != nil {
		return nil, err
	}
	return &LLMPlugin{
		client:   client,
		provider: provider,
		config:   config,
	}, nil
}

// NewLLMPluginPlaceholder creates a placeholder LLM plugin without requiring API keys
// The client will be created when the plugin is actually used with valid credentials
func NewLLMPluginPlaceholder(provider LLMProvider) *LLMPlugin {
	// Create a placeholder client that returns an error when used without configuration
	placeholderClient := &unconfiguredClient{provider: provider}
	return &LLMPlugin{
		client:   placeholderClient,
		provider: provider,
		config: LLMPluginConfig{
			Model: GetProviderInfo(provider)["default_model"].(string),
		},
	}
}

// NewUnconfiguredClient creates a placeholder client that fails when used
func NewUnconfiguredClient(provider LLMProvider) LLMClient {
	return &unconfiguredClient{provider: provider}
}

// unconfiguredClient is a placeholder that fails when used without configuration
type unconfiguredClient struct {
	provider LLMProvider
}

func (c *unconfiguredClient) Complete(ctx context.Context, request LLMRequest) (*LLMResponse, error) {
	return nil, fmt.Errorf("%s provider requires configuration. Please set API key in settings.", c.provider)
}

func (c *unconfiguredClient) CompleteSimple(ctx context.Context, prompt string) (string, error) {
	return "", fmt.Errorf("%s provider requires configuration. Please set API key in settings.", c.provider)
}

func (c *unconfiguredClient) GetProvider() LLMProvider {
	return c.provider
}

func (c *unconfiguredClient) GetDefaultModel() string {
	info := GetProviderInfo(c.provider)
	if defaultModel, ok := info["default_model"].(string); ok {
		return defaultModel
	}
	return ""
}

func (c *unconfiguredClient) ValidateConfig() error {
	return fmt.Errorf("provider not configured")
}

// GetConfig returns the plugin configuration
func (p *LLMPlugin) GetConfig() LLMPluginConfig {
	return p.config
}

// UpdateConfig updates the plugin configuration
func (p *LLMPlugin) UpdateConfig(config LLMPluginConfig) error {
	// Check if credentials changed
	needsRecreate := config.APIKey != p.config.APIKey || config.BaseURL != p.config.BaseURL || config.Model != p.config.Model
	needsRecreate = needsRecreate || (p.provider == ProviderZAi && config.UseCodingPlan != p.config.UseCodingPlan)

	if needsRecreate {
		metadata := map[string]any{}
		if config.UseCodingPlan {
			metadata["use_coding_plan"] = true
		}

		client, err := NewLLMClient(LLMClientConfig{
			Provider: p.provider,
			APIKey:   config.APIKey,
			BaseURL:  config.BaseURL,
			Model:    config.Model,
			Metadata: metadata,
		})
		if err != nil {
			return err
		}
		p.client = client
	}
	p.config = config
	return nil
}

// GetProvider returns the provider type
func (p *LLMPlugin) GetProvider() LLMProvider {
	return p.provider
}

// GetDefaultModel returns the default model for this provider
func (p *LLMPlugin) GetDefaultModel() string {
	return p.client.GetDefaultModel()
}

// ExecuteStep executes an LLM completion step
func (p *LLMPlugin) ExecuteStep(ctx context.Context, stepConfig pipelines.StepConfig, globalContext *pipelines.PluginContext) (*pipelines.PluginContext, error) {
	config := stepConfig.Config

	// Extract messages from config
	messagesRaw, ok := config["messages"]
	if !ok {
		return nil, fmt.Errorf("messages field is required")
	}

	messages, ok := messagesRaw.([]LLMMessage)
	if !ok {
		// Try to parse from array of maps
		messagesArray, ok := messagesRaw.([]interface{})
		if !ok {
			return nil, fmt.Errorf("messages must be an array")
		}

		messages = make([]LLMMessage, len(messagesArray))
		for i, msgRaw := range messagesArray {
			msgMap, ok := msgRaw.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("message %d must be an object", i)
			}

			role, _ := msgMap["role"].(string)
			content, _ := msgMap["content"].(string)
			messages[i] = LLMMessage{
				Role:    role,
				Content: content,
			}
		}
	}

	// Build request
	request := LLMRequest{
		Messages: messages,
	}

	if model, ok := config["model"].(string); ok {
		request.Model = model
	}

	if temp, ok := config["temperature"].(float64); ok {
		request.Temperature = temp
	}

	if maxTokens, ok := config["max_tokens"].(float64); ok {
		request.MaxTokens = int(maxTokens)
	}

	// Execute completion
	response, err := p.client.Complete(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("LLM completion failed: %w", err)
	}

	// Store response in context
	globalContext.Set("llm_response", response.Content)
	globalContext.Set("llm_tool_calls", response.ToolCalls)
	globalContext.Set("llm_usage", response.Usage)
	globalContext.Set("llm_model", response.Model)

	return globalContext, nil
}

// GetPluginType returns "AI"
func (p *LLMPlugin) GetPluginType() string {
	return "AI"
}

// GetPluginName returns the provider name
func (p *LLMPlugin) GetPluginName() string {
	return string(p.provider)
}

// ValidateConfig validates the plugin configuration
func (p *LLMPlugin) ValidateConfig(config map[string]any) error {
	if _, ok := config["messages"]; !ok {
		return fmt.Errorf("messages field is required")
	}
	return nil
}

// GetInputSchema returns the JSON Schema for plugin configuration
func (p *LLMPlugin) GetInputSchema() map[string]any {
	properties := map[string]any{
		"api_key": map[string]any{
			"type":        "string",
			"description": "API key for authentication",
			"format":      "password",
		},
		"model": map[string]any{
			"type":            "string",
			"description":     "Model to use for completions",
			"dynamic_model":   true, // Frontend should fetch models from API
			"model_fetch_url": fmt.Sprintf("/api/v1/ai/providers/%s/models", p.provider),
		},
		"temperature": map[string]any{
			"type":        "number",
			"description": "Temperature for sampling (0-1)",
			"minimum":     0,
			"maximum":     1,
			"default":     0.7,
		},
		"max_tokens": map[string]any{
			"type":        "integer",
			"description": "Maximum tokens in response",
			"minimum":     1,
			"maximum":     8192,
		},
	}

	// Add Z.ai specific option
	if p.provider == ProviderZAi {
		properties["use_coding_plan"] = map[string]any{
			"type":        "boolean",
			"description": "Use Z.ai coding plan API instead of main API",
			"default":     false,
		}
	}

	// Add base URL for custom endpoints
	if p.provider == ProviderOllama || p.provider == ProviderAzure {
		properties["base_url"] = map[string]any{
			"type":        "string",
			"description": "Custom API endpoint URL",
			"format":      "uri",
		}
	}

	return map[string]any{
		"type":       "object",
		"properties": properties,
		"required":   []string{"model"},
	}
}
