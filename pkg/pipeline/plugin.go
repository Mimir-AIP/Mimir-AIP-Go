package pipeline

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"text/template"

	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

// Plugin defines the interface for pipeline step executors
type Plugin interface {
	Execute(action string, params map[string]interface{}, ctx *models.PipelineContext) (map[string]interface{}, error)
}

// DefaultPlugin implements built-in pipeline actions
type DefaultPlugin struct {
	httpClient *http.Client
}

// NewDefaultPlugin creates a new default plugin instance
func NewDefaultPlugin() *DefaultPlugin {
	return &DefaultPlugin{
		httpClient: &http.Client{},
	}
}

// Execute executes a default plugin action
func (p *DefaultPlugin) Execute(action string, params map[string]interface{}, ctx *models.PipelineContext) (map[string]interface{}, error) {
	switch action {
	case "http_request":
		return p.httpRequest(params, ctx)
	case "parse_json":
		return p.parseJSON(params, ctx)
	case "if_else":
		return p.ifElse(params, ctx)
	case "set_context":
		return p.setContext(params, ctx)
	case "get_context":
		return p.getContext(params, ctx)
	case "goto":
		return p.gotoAction(params, ctx)
	default:
		return nil, fmt.Errorf("unknown action: %s", action)
	}
}

// httpRequest makes an HTTP request
func (p *DefaultPlugin) httpRequest(params map[string]interface{}, ctx *models.PipelineContext) (map[string]interface{}, error) {
	// Get parameters
	url, ok := params["url"].(string)
	if !ok {
		return nil, fmt.Errorf("url parameter is required")
	}

	method, ok := params["method"].(string)
	if !ok {
		method = "GET"
	}

	// Resolve template variables in URL
	url = p.ResolveTemplates(url, ctx)

	// Prepare request body if provided
	var body io.Reader
	if bodyData, ok := params["body"]; ok {
		if bodyStr, ok := bodyData.(string); ok {
			bodyStr = p.ResolveTemplates(bodyStr, ctx)
			body = bytes.NewBufferString(bodyStr)
		}
	}

	// Create request
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers if provided
	if headers, ok := params["headers"].(map[string]interface{}); ok {
		for key, value := range headers {
			if strValue, ok := value.(string); ok {
				req.Header.Set(key, p.ResolveTemplates(strValue, ctx))
			}
		}
	}

	// Execute request
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return map[string]interface{}{
		"response": map[string]interface{}{
			"status_code": resp.StatusCode,
			"body":        string(respBody),
			"headers":     resp.Header,
		},
	}, nil
}

// parseJSON parses JSON data
func (p *DefaultPlugin) parseJSON(params map[string]interface{}, ctx *models.PipelineContext) (map[string]interface{}, error) {
	// Get data parameter
	data, ok := params["data"].(string)
	if !ok {
		return nil, fmt.Errorf("data parameter is required")
	}

	// Resolve templates
	data = p.ResolveTemplates(data, ctx)

	// Parse JSON
	var parsed interface{}
	if err := json.Unmarshal([]byte(data), &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return map[string]interface{}{
		"parsed": parsed,
	}, nil
}

// ifElse implements conditional logic
func (p *DefaultPlugin) ifElse(params map[string]interface{}, ctx *models.PipelineContext) (map[string]interface{}, error) {
	// Get condition
	condition, ok := params["condition"]
	if !ok {
		return nil, fmt.Errorf("condition parameter is required")
	}

	// Resolve condition template
	conditionStr := fmt.Sprintf("%v", condition)
	conditionStr = p.ResolveTemplates(conditionStr, ctx)

	// Evaluate condition (simple truthiness check)
	isTrue := conditionStr != "" && conditionStr != "0" && conditionStr != "false" && conditionStr != "null"

	var result string
	if isTrue {
		if ifTrue, ok := params["if_true"].(string); ok {
			result = ifTrue
		}
	} else {
		if ifFalse, ok := params["if_false"].(string); ok {
			result = ifFalse
		}
	}

	return map[string]interface{}{
		"result":    result,
		"condition": isTrue,
	}, nil
}

// setContext sets a value in the context
func (p *DefaultPlugin) setContext(params map[string]interface{}, ctx *models.PipelineContext) (map[string]interface{}, error) {
	key, ok := params["key"].(string)
	if !ok {
		return nil, fmt.Errorf("key parameter is required")
	}

	value, ok := params["value"]
	if !ok {
		return nil, fmt.Errorf("value parameter is required")
	}

	stepName, ok := params["step"].(string)
	if !ok {
		stepName = "_global"
	}

	// Resolve template in value if it's a string
	if strValue, ok := value.(string); ok {
		value = p.ResolveTemplates(strValue, ctx)
	}

	ctx.SetStepData(stepName, key, value)

	return map[string]interface{}{
		"success": true,
	}, nil
}

// getContext retrieves a value from the context
func (p *DefaultPlugin) getContext(params map[string]interface{}, ctx *models.PipelineContext) (map[string]interface{}, error) {
	key, ok := params["key"].(string)
	if !ok {
		return nil, fmt.Errorf("key parameter is required")
	}

	stepName, ok := params["step"].(string)
	if !ok {
		stepName = "_global"
	}

	value, exists := ctx.GetStepData(stepName, key)
	if !exists {
		return map[string]interface{}{
			"exists": false,
			"value":  nil,
		}, nil
	}

	return map[string]interface{}{
		"exists": true,
		"value":  value,
	}, nil
}

// gotoAction implements a goto action for loops
func (p *DefaultPlugin) gotoAction(params map[string]interface{}, ctx *models.PipelineContext) (map[string]interface{}, error) {
	target, ok := params["target"].(string)
	if !ok {
		return nil, fmt.Errorf("target parameter is required")
	}

	return map[string]interface{}{
		"goto":   target,
		"action": "goto",
	}, nil
}

// ResolveTemplates resolves template variables in a string
func (p *DefaultPlugin) ResolveTemplates(input string, ctx *models.PipelineContext) string {
	// Pattern: {{context.step_name.key}}
	pattern := regexp.MustCompile(`\{\{([^}]+)\}\}`)

	result := pattern.ReplaceAllStringFunc(input, func(match string) string {
		// Remove {{ and }}
		expr := strings.TrimSpace(match[2 : len(match)-2])

		// Split by dots
		parts := strings.Split(expr, ".")

		if len(parts) < 2 {
			return match
		}

		// First part should be "context"
		if parts[0] != "context" {
			// Try to use as template
			return p.evaluateTemplate(expr, ctx)
		}

		// Get step name and key
		if len(parts) < 3 {
			return match
		}

		stepName := parts[1]
		key := parts[2]

		// Get value from context
		value, exists := ctx.GetStepData(stepName, key)
		if !exists {
			return match
		}

		// Handle nested access (e.g., {{context.step.key.nested}})
		if len(parts) > 3 {
			for i := 3; i < len(parts); i++ {
				if m, ok := value.(map[string]interface{}); ok {
					value, exists = m[parts[i]]
					if !exists {
						return match
					}
				} else {
					return match
				}
			}
		}

		return fmt.Sprintf("%v", value)
	})

	return result
}

// evaluateTemplate evaluates a template expression
func (p *DefaultPlugin) evaluateTemplate(expr string, ctx *models.PipelineContext) string {
	tmpl, err := template.New("expr").Parse("{{" + expr + "}}")
	if err != nil {
		return ""
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, ctx.Steps); err != nil {
		return ""
	}

	return buf.String()
}
