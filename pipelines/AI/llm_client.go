package AI

import (
	"context"
	"fmt"
)

// LLMProvider represents different LLM providers
type LLMProvider string

const (
	ProviderOpenAI    LLMProvider = "openai"
	ProviderAnthropic LLMProvider = "anthropic"
	ProviderOllama    LLMProvider = "ollama"
	ProviderAzure     LLMProvider = "azure"
	ProviderGoogle    LLMProvider = "google"
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
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", config.Provider)
	}
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
