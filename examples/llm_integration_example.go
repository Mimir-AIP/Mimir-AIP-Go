// LLM Integration Example
// This example shows how to use the OpenAI plugin directly in Go code
// It demonstrates prompt/response handling and error cases

package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
)

// SimpleOpenAIPlugin is a minimal mock for demonstration
// In real usage, import and use the actual OpenAI plugin
type SimpleOpenAIPlugin struct{}

func (p *SimpleOpenAIPlugin) ExecuteStep(ctx context.Context, stepConfig pipelines.StepConfig, globalContext *pipelines.PluginContext) (*pipelines.PluginContext, error) {
	config := stepConfig.Config
	if apiKey, ok := config["api_key"].(string); !ok || apiKey == "" || apiKey == "your-api-key-here" {
		return nil, fmt.Errorf("valid OpenAI API key is required")
	}
	if messages, ok := config["messages"].([]interface{}); ok {
		if len(messages) > 0 {
			if msgMap, ok := messages[0].(map[string]interface{}); ok {
				if content, ok := msgMap["content"].(string); ok {
					// Mock response for demo
					result := pipelines.NewPluginContext()
					result.Set(stepConfig.Output, map[string]interface{}{
						"content":       fmt.Sprintf("Mock AI response to: %s", content),
						"model":         "gpt-3.5-turbo-mock",
						"usage":         map[string]interface{}{"tokens": 42},
						"finish_reason": "stop",
						"request_id":    "mock-id-123",
						"timestamp":     "2025-11-25T10:00:00Z",
					})
					return result, nil
				}
			}
		}
	}
	return nil, fmt.Errorf("invalid messages configuration")
}
func (p *SimpleOpenAIPlugin) GetPluginType() string { return "AIModels" }
func (p *SimpleOpenAIPlugin) GetPluginName() string { return "openai" }
func (p *SimpleOpenAIPlugin) ValidateConfig(config map[string]interface{}) error {
	if config["messages"] == nil {
		return fmt.Errorf("messages are required")
	}
	return nil
}

func runLLMIntegrationExample() {
	plugin := &SimpleOpenAIPlugin{}

	// Example 1: Simple chat completion
	fmt.Println("=== Example 1: Simple Chat ===")
	err := runSimpleChat(plugin)
	if err != nil {
		log.Printf("Simple chat failed: %v", err)
	}

	// Example 2: Structured prompt with response parsing
	fmt.Println("\n=== Example 2: Structured Prompt ===")
	err = runStructuredPrompt(plugin)
	if err != nil {
		log.Printf("Structured prompt failed: %v", err)
	}

	// Example 3: Error handling (missing API key)
	fmt.Println("\n=== Example 3: Error Handling ===")
	err = runErrorCaseExample(plugin)
	if err != nil {
		fmt.Printf("Expected error caught: %v\n", err)
	}
}

// runSimpleChat demonstrates a basic chat completion
func runSimpleChat(plugin *SimpleOpenAIPlugin) error {
	stepConfig := pipelines.StepConfig{
		Name:   "SimpleChat",
		Plugin: "AIModels.openai",
		Config: map[string]interface{}{
			"operation": "chat",
			"model":     "gpt-3.5-turbo",
			"messages": []interface{}{
				map[string]interface{}{
					"role":    "user",
					"content": "What is artificial intelligence in one sentence?",
				},
			},
			"max_tokens":  50,
			"temperature": 0.7,
			"api_key":     getAPIKey(),
		},
		Output: "response",
	}

	result, err := plugin.ExecuteStep(context.Background(), stepConfig, pipelines.NewPluginContext())
	if err != nil {
		return fmt.Errorf("chat execution failed: %w", err)
	}

	if val, exists := result.Get("response"); exists {
		if respMap, ok := val.(map[string]interface{}); ok {
			if content, ok := respMap["content"].(string); ok {
				fmt.Printf("AI Response: %s\n", content)
			}
			if usage, ok := respMap["usage"].(map[string]interface{}); ok {
				fmt.Printf("Token Usage: %+v\n", usage)
			}
		}
	}

	return nil
}

// runStructuredPrompt demonstrates a more complex prompt with expected response format
func runStructuredPrompt(plugin *SimpleOpenAIPlugin) error {
	stepConfig := pipelines.StepConfig{
		Name:   "StructuredPrompt",
		Plugin: "AIModels.openai",
		Config: map[string]interface{}{
			"operation": "chat",
			"model":     "gpt-3.5-turbo",
			"messages": []interface{}{
				map[string]interface{}{
					"role":    "system",
					"content": "You are a helpful assistant. Always respond in JSON format with 'summary' and 'confidence' fields.",
				},
				map[string]interface{}{
					"role":    "user",
					"content": "Summarize the concept of machine learning.",
				},
			},
			"max_tokens":  150,
			"temperature": 0.3,
			"api_key":     getAPIKey(),
		},
		Output: "structured_response",
	}

	result, err := plugin.ExecuteStep(context.Background(), stepConfig, pipelines.NewPluginContext())
	if err != nil {
		return fmt.Errorf("structured prompt execution failed: %w", err)
	}

	if val, exists := result.Get("structured_response"); exists {
		if respMap, ok := val.(map[string]interface{}); ok {
			if content, ok := respMap["content"].(string); ok {
				fmt.Printf("Structured AI Response:\n%s\n", content)
			}
		}
	}

	return nil
}

// runErrorCase demonstrates error handling with invalid API key
func runErrorCaseExample(plugin *SimpleOpenAIPlugin) error {
	stepConfig := pipelines.StepConfig{
		Name:   "ErrorCase",
		Plugin: "AIModels.openai",
		Config: map[string]interface{}{
			"operation": "chat",
			"model":     "gpt-3.5-turbo",
			"messages": []interface{}{
				map[string]interface{}{
					"role":    "user",
					"content": "This should fail",
				},
			},
			"max_tokens":  50,
			"temperature": 0.7,
			"api_key":     "invalid-key",
		},
		Output: "error_response",
	}

	_, err := plugin.ExecuteStep(context.Background(), stepConfig, pipelines.NewPluginContext())
	if err != nil {
		return fmt.Errorf("expected error: %w", err)
	}

	return fmt.Errorf("expected an error but got none")
}

// getAPIKey retrieves API key from environment
func getAPIKey() string {
	if key := os.Getenv("OPENAI_API_KEY"); key != "" {
		return key
	}
	return "your-api-key-here" // Replace with actual key for testing
}
