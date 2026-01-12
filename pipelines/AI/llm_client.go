package AI

import (
	"context"
	"fmt"
)

// LLMProvider represents different LLM providers
type LLMProvider string

const (
	ProviderOpenAI     LLMProvider = "openai"
	ProviderAnthropic  LLMProvider = "anthropic"
	ProviderOllama     LLMProvider = "ollama"
	ProviderAzure      LLMProvider = "azure"
	ProviderGoogle     LLMProvider = "google"
	ProviderOpenRouter LLMProvider = "openrouter"
	ProviderZAi        LLMProvider = "z-ai"
	ProviderMock       LLMProvider = "mock" // For cost-free testing
	ProviderLocal      LLMProvider = "local"
)

// LLMMessage represents a single message in a conversation
type LLMMessage struct {
	Role    string `json:"role"`    // system, user, assistant
	Content string `json:"content"` // message content
}

// LLMRequest represents a request to an LLM
type LLMRequest struct {
	Messages    []LLMMessage   `json:"messages"`
	Model       string         `json:"model,omitempty"`
	Temperature float64        `json:"temperature,omitempty"`
	MaxTokens   int            `json:"max_tokens,omitempty"`
	SystemMsg   string         `json:"system,omitempty"` // System message (some providers handle separately)
	Tools       []LLMTool      `json:"tools,omitempty"`  // Function calling tools
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// LLMResponse represents a response from an LLM
type LLMResponse struct {
	Content      string         `json:"content"`
	ToolCalls    []LLMToolCall  `json:"tool_calls,omitempty"`
	FinishReason string         `json:"finish_reason,omitempty"` // stop, length, tool_calls
	Usage        *LLMUsage      `json:"usage,omitempty"`
	Model        string         `json:"model,omitempty"`
	Metadata     map[string]any `json:"metadata,omitempty"`
}

// LLMTool represents a function that can be called by the LLM
type LLMTool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"` // JSON schema for parameters
}

// LLMToolCall represents a function call requested by the LLM
type LLMToolCall struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

// LLMUsage tracks token usage
type LLMUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// LLMClient is the interface that all LLM providers must implement
type LLMClient interface {
	// Complete sends a completion request to the LLM
	Complete(ctx context.Context, request LLMRequest) (*LLMResponse, error)

	// CompleteSimple is a convenience method for simple text completion
	CompleteSimple(ctx context.Context, prompt string) (string, error)

	// GetProvider returns the provider type
	GetProvider() LLMProvider

	// GetDefaultModel returns the default model for this provider
	GetDefaultModel() string

	// ValidateConfig validates the client configuration
	ValidateConfig() error
}

// LLMClientConfig holds configuration for creating LLM clients
type LLMClientConfig struct {
	Provider    LLMProvider    `json:"provider"`
	APIKey      string         `json:"api_key,omitempty"`
	BaseURL     string         `json:"base_url,omitempty"`    // For custom endpoints (Azure, Ollama)
	Model       string         `json:"model,omitempty"`       // Default model
	Temperature float64        `json:"temperature,omitempty"` // Default temperature
	MaxTokens   int            `json:"max_tokens,omitempty"`  // Default max tokens
	Timeout     int            `json:"timeout,omitempty"`     // Request timeout in seconds
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// NewLLMClient creates a new LLM client based on the provider
func NewLLMClient(config LLMClientConfig) (LLMClient, error) {
	switch config.Provider {
	case ProviderOpenAI:
		return NewOpenAIClient(config)
	case ProviderAnthropic:
		return NewAnthropicClient(config)
	case ProviderOllama:
		return NewOllamaClient(config)
	case ProviderAzure:
		return NewAzureOpenAIClient(config)
	case ProviderGoogle:
		return NewGoogleClient(config)
	case ProviderOpenRouter:
		return NewOpenRouterClient(config)
	case ProviderZAi:
		return NewZAiClient(config)
	case ProviderLocal:
		return NewLocalLLMClient(LocalLLMConfig{
			ModelPath:   config.BaseURL,
			ModelName:   config.Model,
			Temperature: config.Temperature,
			MaxTokens:   config.MaxTokens,
		})
	case ProviderMock:
		return NewIntelligentMockLLMClient(), nil
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", config.Provider)
	}
}

// GetAvailableModelsForProvider returns placeholder models - actual models are fetched from API when configured
func GetAvailableModelsForProvider(provider LLMProvider) []string {
	switch provider {
	case ProviderOpenAI:
		return []string{"gpt-4o", "gpt-4o-mini", "gpt-4-turbo", "gpt-3.5-turbo"}
	case ProviderAnthropic:
		return []string{"claude-sonnet-4-20250514", "claude-haiku-3-20250506", "claude-opus-4-20250506"}
	case ProviderOpenRouter:
		return []string{} // Fetched dynamically from OpenRouter API
	case ProviderZAi:
		return []string{"claude-sonnet-4-20250514", "deepseek-coder"}
	case ProviderOllama:
		return []string{} // Fetched dynamically from Ollama API
	case ProviderLocal:
		return []string{"tinyllama-1.1b-chat.q4_0.gguf", "phi-2.q4_0.gguf", "gemma-2b-it.q4_0.gguf"}
	case ProviderMock:
		return []string{"mock-gpt-4", "mock-claude-3"}
	default:
		return []string{}
	}
}

// GetProviderInfo returns information about a provider
func GetProviderInfo(provider LLMProvider) map[string]any {
	descriptions := map[LLMProvider]string{
		ProviderOpenAI:     "OpenAI GPT-4 and GPT-3.5 models",
		ProviderAnthropic:  "Anthropic Claude models",
		ProviderOllama:     "Local LLM via Ollama (privacy-preserving)",
		ProviderLocal:      "Bundled local LLM (no API calls, runs offline)",
		ProviderAzure:      "Microsoft Azure OpenAI Service",
		ProviderGoogle:     "Google Gemini models",
		ProviderOpenRouter: "OpenRouter aggregation platform",
		ProviderZAi:        "Z.ai API with coding plan support",
		ProviderMock:       "Mock LLM for testing and demos",
	}

	defaultModels := map[LLMProvider]string{
		ProviderOpenAI:     "gpt-4o-mini",
		ProviderAnthropic:  "claude-sonnet-4-20250514",
		ProviderOllama:     "llama3.2",
		ProviderLocal:      "tinyllama-1.1b-chat.q4_0.gguf",
		ProviderAzure:      "gpt-4o-mini",
		ProviderGoogle:     "gemini-1.5-pro",
		ProviderOpenRouter: "anthropic/claude-sonnet-4-20250514",
		ProviderZAi:        "claude-sonnet-4-20250514",
		ProviderMock:       "mock-gpt-4",
	}

	return map[string]any{
		"provider":      provider,
		"name":          string(provider),
		"description":   descriptions[provider],
		"default_model": defaultModels[provider],
	}
}

// LLMPluginConfigGetter is a function that retrieves plugin configuration
// This is set by the server to enable LLM clients to fetch their config
var LLMPluginConfigGetter func(pluginName string) (map[string]interface{}, error)

// GetLLMClientForProvider creates an LLM client for the specified provider
// It first checks if there's a saved configuration for this provider plugin
func GetLLMClientForProvider(provider LLMProvider, defaultClient LLMClient) (LLMClient, error) {
	pluginName := string(provider)

	// Try to get config from plugin config system
	if LLMPluginConfigGetter != nil {
		config, err := LLMPluginConfigGetter(pluginName)
		if err == nil && config != nil {
			// Build LLMClientConfig from stored config
			clientConfig := LLMClientConfig{
				Provider: provider,
				APIKey:   getStringFromConfig(config, "api_key"),
				BaseURL:  getStringFromConfig(config, "base_url"),
				Model:    getStringFromConfig(config, "model"),
			}

			// Handle Z.ai coding plan
			if provider == ProviderZAi {
				if useCodingPlan, ok := config["use_coding_plan"].(bool); ok && useCodingPlan {
					clientConfig.Metadata = map[string]any{"use_coding_plan": true}
				}
			}

			// Only create new client if we have at least a model configured
			if clientConfig.Model != "" {
				return NewLLMClient(clientConfig)
			}
		}
	}

	// Fall back to default client
	return defaultClient, nil
}

func getStringFromConfig(config map[string]interface{}, key string) string {
	if val, ok := config[key].(string); ok {
		return val
	}
	return ""
}

// LLMClientFactory manages multiple LLM clients
type LLMClientFactory struct {
	clients       map[string]LLMClient
	defaultClient LLMClient
}

// NewLLMClientFactory creates a new factory
func NewLLMClientFactory() *LLMClientFactory {
	return &LLMClientFactory{
		clients: make(map[string]LLMClient),
	}
}

// RegisterClient registers a client with a name
func (f *LLMClientFactory) RegisterClient(name string, client LLMClient) {
	f.clients[name] = client
	if f.defaultClient == nil {
		f.defaultClient = client
	}
}

// GetClient returns a client by name
func (f *LLMClientFactory) GetClient(name string) (LLMClient, error) {
	client, ok := f.clients[name]
	if !ok {
		return nil, fmt.Errorf("LLM client not found: %s", name)
	}
	return client, nil
}

// GetDefaultClient returns the default client
func (f *LLMClientFactory) GetDefaultClient() LLMClient {
	return f.defaultClient
}

// SetDefaultClient sets the default client
func (f *LLMClientFactory) SetDefaultClient(name string) error {
	client, ok := f.clients[name]
	if !ok {
		return fmt.Errorf("LLM client not found: %s", name)
	}
	f.defaultClient = client
	return nil
}
