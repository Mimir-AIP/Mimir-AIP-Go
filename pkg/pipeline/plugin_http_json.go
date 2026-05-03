package pipeline

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"io"
	"net/http"
)

// ── HTTP ──────────────────────────────────────────────────────────────────────

// httpRequest makes an HTTP request
func (p *DefaultPlugin) httpRequest(params map[string]interface{}, ctx *models.PipelineContext) (map[string]interface{}, error) {
	url, ok := params["url"].(string)
	if !ok {
		return nil, fmt.Errorf("url parameter is required")
	}

	method, ok := params["method"].(string)
	if !ok {
		method = "GET"
	}

	url = p.ResolveTemplates(url, ctx)

	var body io.Reader
	if bodyData, ok := params["body"]; ok {
		if bodyStr, ok := bodyData.(string); ok {
			bodyStr = p.ResolveTemplates(bodyStr, ctx)
			body = bytes.NewBufferString(bodyStr)
		}
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if headers, ok := params["headers"].(map[string]interface{}); ok {
		for key, value := range headers {
			if strValue, ok := value.(string); ok {
				req.Header.Set(key, p.ResolveTemplates(strValue, ctx))
			}
		}
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

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

// ── JSON ──────────────────────────────────────────────────────────────────────

// parseJSON parses JSON data
func (p *DefaultPlugin) parseJSON(params map[string]interface{}, ctx *models.PipelineContext) (map[string]interface{}, error) {
	data, ok := params["data"].(string)
	if !ok {
		return nil, fmt.Errorf("data parameter is required")
	}

	data = p.ResolveTemplates(data, ctx)

	var parsed interface{}
	if err := json.Unmarshal([]byte(data), &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return map[string]interface{}{
		"parsed": parsed,
	}, nil
}
