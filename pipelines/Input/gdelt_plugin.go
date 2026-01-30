package Input

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
)

// GDELTPlugin fetches data from GDELT API
type GDELTPlugin struct{}

func NewGDELTPlugin() *GDELTPlugin { return &GDELTPlugin{} }

func (p *GDELTPlugin) ExecuteStep(ctx context.Context, stepConfig pipelines.StepConfig, globalContext *pipelines.PluginContext) (*pipelines.PluginContext, error) {
	cfg := stepConfig.Config
	q, _ := cfg["query"].(string)
	if q == "" {
		return nil, fmt.Errorf("query is required")
	}
	timespan, _ := cfg["timespan"].(string)
	if timespan == "" {
		timespan = "7days"
	}

	// Use GDELT v2 API doc endpoint (example: https://api.gdeltproject.org/api/v2/doc/doc?query=...)
	base := "https://api.gdeltproject.org/api/v2/doc/doc"
	u, _ := url.Parse(base)
	qparams := u.Query()
	qparams.Set("query", q)
	qparams.Set("mode", "artlist")
	u.RawQuery = qparams.Encode()

	req, _ := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	req.Header.Set("User-Agent", "Mimir-AIP/1.0")
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gdelt request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read gdelt response: %w", err)
	}

	var parsed any
	if err := json.Unmarshal(body, &parsed); err != nil {
		// return raw body as string if JSON fail
		parsed = string(body)
	}

	out := pipelines.NewPluginContext()
	outKey := stepConfig.Output
	if outKey == "" {
		outKey = "gdelt"
	}
	out.Set(outKey, map[string]any{"data": parsed, "fetched_at": time.Now().Format(time.RFC3339)})
	return out, nil
}

func (p *GDELTPlugin) GetPluginType() string { return "Input" }
func (p *GDELTPlugin) GetPluginName() string { return "gdelt" }

func (p *GDELTPlugin) ValidateConfig(config map[string]any) error {
	if _, ok := config["query"].(string); !ok {
		return fmt.Errorf("query required")
	}
	return nil
}

func (p *GDELTPlugin) GetInputSchema() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{"query": map[string]any{"type": "string"}, "timespan": map[string]any{"type": "string"}}, "required": []string{"query"}}
}
