package AI

import (
	"context"
	"fmt"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
)

// LLMPlugin wraps an LLM client as a pipeline plugin
type LLMPlugin struct {
	client   LLMClient
	provider LLMProvider
}

// NewLLMPlugin creates a new LLM plugin from a client
func NewLLMPlugin(client LLMClient, provider LLMProvider) *LLMPlugin {
	return &LLMPlugin{
		client:   client,
		provider: provider,
	}
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

// GetInputSchema returns empty schema for AI plugins (not exposed to agents)
func (p *LLMPlugin) GetInputSchema() map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
}
