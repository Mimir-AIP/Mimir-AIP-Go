// Responsible for executing a pipeline
package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
)

// PipelineExecutionResult represents the result of a pipeline execution
type PipelineExecutionResult struct {
	Success    bool                     `json:"success"`
	Error      string                   `json:"error,omitempty"`
	Context    *pipelines.PluginContext `json:"context,omitempty"`
	ExecutedAt string                   `json:"executed_at"`
}

// RunPipeline executes a pipeline by name or file path
func RunPipeline(pipeline string, args ...string) error {
	if pipeline == "" {
		return fmt.Errorf("pipeline path cannot be empty")
	}
	if !strings.HasSuffix(pipeline, ".yaml") {
		return fmt.Errorf("pipeline path must end with .yaml")
	}

	// Parse the pipeline configuration
	pipelineConfig, err := ParsePipeline(pipeline)
	if err != nil {
		return fmt.Errorf("failed to parse pipeline: %w", err)
	}

	// Execute the pipeline
	result, err := ExecutePipeline(context.Background(), pipelineConfig)
	if err != nil {
		return fmt.Errorf("failed to execute pipeline: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("pipeline execution failed: %s", result.Error)
	}

	return nil
}

// ExecutePipeline executes a parsed pipeline configuration
func ExecutePipeline(ctx context.Context, config *PipelineConfig) (*PipelineExecutionResult, error) {
	result := &PipelineExecutionResult{
		Success: true,
		Context: pipelines.NewPluginContext(),
	}

	// Get plugin registry (this should be injected or globally available)
	registry := pipelines.NewPluginRegistry()

	// Register default plugins (this will be expanded)
	if err := registerDefaultPlugins(registry); err != nil {
		return nil, fmt.Errorf("failed to register plugins: %w", err)
	}

	// Execute each step
	for i, step := range config.Steps {
		stepResult, err := executeStep(ctx, registry, step, result.Context)
		if err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("step %d (%s) failed: %v", i+1, step.Name, err)
			return result, nil
		}

		// Merge step results into global context
		for _, key := range stepResult.Keys() {
			if value, exists := stepResult.Get(key); exists {
				result.Context.Set(key, value)
			}
		}
	}

	return result, nil
}

// ExecutePipelineWithRegistry executes a parsed pipeline configuration with a specific registry
func ExecutePipelineWithRegistry(ctx context.Context, config *PipelineConfig, registry *pipelines.PluginRegistry) (*PipelineExecutionResult, error) {
	result := &PipelineExecutionResult{
		Success:    true,
		Context:    pipelines.NewPluginContext(),
		ExecutedAt: time.Now().Format(time.RFC3339),
	}

	// Execute each step
	for i, step := range config.Steps {
		stepResult, err := executeStep(ctx, registry, step, result.Context)
		if err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("step %d (%s) failed: %v", i+1, step.Name, err)
			return result, nil // Return result with Success=false, not an error
		}

		// Merge step results into global context
		for _, key := range stepResult.Keys() {
			if value, exists := stepResult.Get(key); exists {
				result.Context.Set(key, value)
			}
		}
	}

	return result, nil
}

// executeStep executes a single pipeline step
func executeStep(ctx context.Context, registry *pipelines.PluginRegistry, step pipelines.StepConfig, context *pipelines.PluginContext) (*pipelines.PluginContext, error) {
	// Parse plugin reference (e.g., "Input.api" -> type: "Input", name: "api")
	pluginParts := strings.Split(step.Plugin, ".")
	if len(pluginParts) != 2 {
		return pipelines.NewPluginContext(), fmt.Errorf("invalid plugin reference format: %s, expected 'Type.Name'", step.Plugin)
	}

	pluginType := pluginParts[0]
	pluginName := pluginParts[1]

	// Get the plugin
	plugin, err := registry.GetPlugin(pluginType, pluginName)
	if err != nil {
		return pipelines.NewPluginContext(), fmt.Errorf("failed to get plugin %s: %w", step.Plugin, err)
	}

	// Validate configuration
	if err := plugin.ValidateConfig(step.Config); err != nil {
		return pipelines.NewPluginContext(), fmt.Errorf("configuration validation failed for plugin %s: %w", step.Plugin, err)
	}

	// Use the provided context (timeout handling can be added later)
	stepCtx := ctx

	// Execute the step
	stepResult, err := plugin.ExecuteStep(stepCtx, step, context)
	if err != nil {
		return pipelines.NewPluginContext(), fmt.Errorf("plugin execution failed: %w", err)
	}

	return stepResult, nil
}

// registerDefaultPlugins registers the built-in plugins
func registerDefaultPlugins(registry *pipelines.PluginRegistry) error {
	// This will be expanded as we implement more plugins
	// For now, we'll register mock plugins for testing

	// Register a real API input plugin
	apiPlugin := &RealAPIPlugin{}
	if err := registry.RegisterPlugin(apiPlugin); err != nil {
		return fmt.Errorf("failed to register API plugin: %w", err)
	}

	// Register a mock HTML output plugin
	htmlPlugin := &MockHTMLPlugin{}
	if err := registry.RegisterPlugin(htmlPlugin); err != nil {
		return fmt.Errorf("failed to register HTML plugin: %w", err)
	}

	return nil
}

// RealAPIPlugin implements actual HTTP requests
type RealAPIPlugin struct{}

func (p *RealAPIPlugin) ExecuteStep(ctx context.Context, stepConfig pipelines.StepConfig, globalContext *pipelines.PluginContext) (*pipelines.PluginContext, error) {
	config := stepConfig.Config

	// Extract configuration
	url, ok := config["url"].(string)
	if !ok || url == "" {
		return nil, fmt.Errorf("url is required in config")
	}

	method, ok := config["method"].(string)
	if !ok || method == "" {
		method = "GET"
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	if headers, ok := config["headers"].(map[string]any); ok {
		for key, value := range headers {
			if strValue, ok := value.(string); ok {
				req.Header.Set(key, strValue)
			}
		}
	}

	// Add default User-Agent if not specified
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "Mimir-AIP/1.0")
	}

	// Add query parameters
	if params, ok := config["params"].(map[string]any); ok {
		q := req.URL.Query()
		for key, value := range params {
			if strValue, ok := value.(string); ok {
				q.Add(key, strValue)
			}
		}
		req.URL.RawQuery = q.Encode()
	}

	// Add request body for POST/PUT
	if method == "POST" || method == "PUT" {
		if data, ok := config["data"].(map[string]any); ok {
			jsonData, err := json.Marshal(data)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal request data: %w", err)
			}
			req.Body = io.NopCloser(bytes.NewReader(jsonData))
			req.Header.Set("Content-Type", "application/json")
		}
	}

	// Execute request
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse JSON response if possible
	var jsonData any
	if err := json.Unmarshal(body, &jsonData); err != nil {
		// If not JSON, use raw body
		jsonData = string(body)
	}

	// Build result
	result := pipelines.NewPluginContext()
	result.Set(stepConfig.Output, map[string]any{
		"url":         resp.Request.URL.String(),
		"status_code": resp.StatusCode,
		"headers":     make(map[string]string),
		"body":        jsonData,
		"timestamp":   time.Now().Format(time.RFC3339),
	})

	// Convert headers
	headers := make(map[string]string)
	for key, values := range resp.Header {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}

	// Update the headers in the result
	if existing, exists := result.Get(stepConfig.Output); exists {
		if resultMap, ok := existing.(map[string]any); ok {
			resultMap["headers"] = headers
			result.Set(stepConfig.Output, resultMap)
		}
	}

	return result, nil
}

func (p *RealAPIPlugin) GetPluginType() string { return "Input" }
func (p *RealAPIPlugin) GetPluginName() string { return "api" }
func (p *RealAPIPlugin) ValidateConfig(config map[string]any) error {
	if _, ok := config["url"]; !ok {
		return fmt.Errorf("url is required")
	}
	return nil
}

// MockHTMLPlugin is a temporary mock implementation for testing
type MockHTMLPlugin struct{}

func (p *MockHTMLPlugin) ExecuteStep(ctx context.Context, stepConfig pipelines.StepConfig, globalContext *pipelines.PluginContext) (*pipelines.PluginContext, error) {
	// Mock implementation - in real implementation this would generate HTML reports
	result := pipelines.NewPluginContext()
	// Use the output name from stepConfig
	outputKey := stepConfig.Output
	if outputKey == "" {
		outputKey = "report_generated"
	}
	result.Set(outputKey, map[string]any{
		"format":    "html",
		"generated": true,
		"timestamp": time.Now().Format(time.RFC3339),
	})
	return result, nil
}

func (p *MockHTMLPlugin) GetPluginType() string { return "Output" }
func (p *MockHTMLPlugin) GetPluginName() string { return "html" }
func (p *MockHTMLPlugin) ValidateConfig(config map[string]any) error {
	return nil // Mock validation
}
