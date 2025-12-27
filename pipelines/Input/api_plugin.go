package Input

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
)

// APIPlugin handles polling REST APIs and extracting JSON data
type APIPlugin struct{}

// GetPluginType returns the plugin type
func (p *APIPlugin) GetPluginType() string {
	return "Input"
}

// GetPluginName returns the plugin name
func (p *APIPlugin) GetPluginName() string {
	return "api"
}

// ValidateConfig validates API polling configuration
func (p *APIPlugin) ValidateConfig(config map[string]any) error {
	if config["url"] == nil {
		return fmt.Errorf("url is required")
	}
	if config["method"] == nil {
		return fmt.Errorf("method is required")
	}
	return nil
}

// ExecuteStep polls API and returns JSON data
func (p *APIPlugin) ExecuteStep(ctx context.Context, stepConfig pipelines.StepConfig, globalContext *pipelines.PluginContext) (*pipelines.PluginContext, error) {
	// Extract configuration
	url, _ := stepConfig.Config["url"].(string)
	method, _ := stepConfig.Config["method"].(string)
	headers, _ := stepConfig.Config["headers"].(map[string]string)
	if headers == nil {
		headers = make(map[string]string)
	}

	// Optional polling configuration
	interval, _ := stepConfig.Config["poll_interval"].(int) // seconds
	if interval == 0 {
		interval = 300 // 5 minutes default
	}

	var results []map[string]interface{}
	var columns []string

	// Create HTTP client
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Make request
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse JSON response
	var jsonResponse interface{}
	if err := json.Unmarshal(body, &jsonResponse); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	// Convert to structured format
	switch v := jsonResponse.(type) {
	case map[string]interface{}:
		results = []map[string]interface{}{v}
		// Extract columns from keys
		for key := range v {
			columns = append(columns, key)
		}
	case []interface{}:
		for _, item := range v {
			if itemMap, ok := item.(map[string]interface{}); ok {
				results = append(results, itemMap)
				// Extract columns from first item
				if len(columns) == 0 {
					for key := range itemMap {
						columns = append(columns, key)
					}
				}
			}
		}
	default:
		results = []map[string]interface{}{
			{"data": jsonResponse},
		}
		columns = []string{"data"}
	}

	// Create result context
	result := pipelines.NewPluginContext()
	result.Set("api_response", map[string]any{
		"status_code": resp.StatusCode,
		"headers":     resp.Header,
		"body":        string(body),
		"json_data":   jsonResponse,
	})
	result.Set("extracted_data", map[string]any{
		"row_count":    len(results),
		"column_count": len(columns),
		"columns":      columns,
		"rows":         results,
	})
	result.SetMetadata("source_type", "rest_api")
	result.SetMetadata("api_url", url)
	result.SetMetadata("api_method", method)
	result.SetMetadata("extracted_at", time.Now().Format(time.RFC3339))
	result.SetMetadata("status_code", resp.StatusCode)
	result.SetMetadata("content_type", resp.Header.Get("Content-Type"))

	return result, nil
}

// GetInputSchema returns the input schema for the API plugin
func (p *APIPlugin) GetInputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"url": map[string]any{
				"type":        "string",
				"description": "API endpoint URL to poll",
				"required":    true,
				"format":      "uri",
			},
			"method": map[string]any{
				"type":        "string",
				"description": "HTTP method (GET, POST, PUT, DELETE)",
				"required":    true,
				"enum":        []string{"GET", "POST", "PUT", "DELETE"},
				"default":     "GET",
			},
			"headers": map[string]any{
				"type":        "object",
				"description": "HTTP headers to include in request",
				"required":    false,
				"additionalProperties": map[string]any{
					"type": "string",
				},
			},
			"poll_interval": map[string]any{
				"type":        "integer",
				"description": "Polling interval in seconds",
				"required":    false,
				"default":     300,
				"minimum":     10,
			},
		},
	}
}
