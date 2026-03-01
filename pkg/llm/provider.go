package llm

import "context"

// Model represents an LLM model available from a provider.
type Model struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	ContextLength int    `json:"context_length"`
	IsFree        bool   `json:"is_free"`
	ProviderName  string `json:"provider_name"`
}

// Message is a single turn in a chat conversation.
type Message struct {
	Role    string `json:"role"`    // "system" | "user" | "assistant"
	Content string `json:"content"`
}

// CompletionRequest describes a chat-completion call.
type CompletionRequest struct {
	Model       string
	Messages    []Message
	MaxTokens   int
	Temperature float64
}

// CompletionResponse holds the result of a chat-completion call.
type CompletionResponse struct {
	Content      string
	Model        string
	InputTokens  int
	OutputTokens int
}

// Provider is the interface that all LLM back-ends must implement.
type Provider interface {
	// Name returns a human-readable identifier for this provider.
	Name() string
	// ListModels fetches the live model catalogue.  No caching is applied here;
	// caching is handled by Service.
	ListModels(ctx context.Context) ([]Model, error)
	// Complete sends a chat-completion request and returns the response.
	Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error)
}
