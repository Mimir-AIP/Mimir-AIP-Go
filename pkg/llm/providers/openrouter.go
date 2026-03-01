package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/mimir-aip/mimir-aip-go/pkg/llm"
)

const (
	openRouterBaseURL = "https://openrouter.ai/api/v1"
	// openRouterExtraHeaders are required by OpenRouter to identify the calling
	// application. Passed verbatim to doOpenAICompatCompletion.
	openRouterExtraHeaders = `{"HTTP-Referer":"https://mimir-aip.io","X-Title":"Mimir AIP"}`
)

// OpenRouterProvider implements llm.Provider for the OpenRouter gateway.
// It discovers free models automatically by inspecting pricing fields and
// prepends a synthetic "openrouter/auto" entry that lets the gateway pick the
// best available model.
type OpenRouterProvider struct {
	apiKey string
	client *http.Client
}

// NewOpenRouterProvider creates an OpenRouter-backed provider.
func NewOpenRouterProvider(apiKey string) *OpenRouterProvider {
	return &OpenRouterProvider{
		apiKey: apiKey,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// Name returns the provider identifier.
func (p *OpenRouterProvider) Name() string { return "openrouter" }

// ListModels fetches the live model catalogue from OpenRouter.
// Models with prompt pricing of "0" (or "0.0", "0.00") are marked IsFree.
// A synthetic "openrouter/auto" model is always prepended so callers can let
// OpenRouter pick the best available model automatically.
func (p *OpenRouterProvider) ListModels(ctx context.Context) ([]llm.Model, error) {
	url := openRouterBaseURL + "/models"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("openrouter: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openrouter: GET %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("openrouter: GET %s: status %d: %s", url, resp.StatusCode, string(body))
	}

	var result struct {
		Data []struct {
			ID            string `json:"id"`
			Name          string `json:"name"`
			ContextLength int    `json:"context_length"`
			Pricing       struct {
				Prompt     string `json:"prompt"`
				Completion string `json:"completion"`
			} `json:"pricing"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("openrouter: decode models: %w", err)
	}

	// Synthetic router entries always come first.
	// openrouter/free routes automatically to the best available free model —
	// useful for demos and development without any API cost.
	// openrouter/auto routes to the best available model regardless of price.
	models := []llm.Model{
		{
			ID:           "openrouter/free",
			Name:         "OpenRouter Free (best available free model)",
			IsFree:       true,
			ProviderName: "openrouter",
		},
		{
			ID:           "openrouter/auto",
			Name:         "OpenRouter Auto (best available)",
			IsFree:       false,
			ProviderName: "openrouter",
		},
	}

	for _, m := range result.Data {
		models = append(models, llm.Model{
			ID:            m.ID,
			Name:          m.Name,
			ContextLength: m.ContextLength,
			IsFree:        isFreePrice(m.Pricing.Prompt),
			ProviderName:  "openrouter",
		})
	}
	return models, nil
}

// Complete sends a chat-completion request via the OpenRouter API.
func (p *OpenRouterProvider) Complete(ctx context.Context, req llm.CompletionRequest) (llm.CompletionResponse, error) {
	endpoint := openRouterBaseURL + "/chat/completions"
	return doOpenAICompatCompletion(ctx, p.client, endpoint, p.apiKey, openRouterExtraHeaders, req)
}

// isFreePrice returns true when a pricing string represents zero cost.
// OpenRouter expresses free models as "0", "0.0", "0.00", etc.
func isFreePrice(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	// Strip any trailing zeros after decimal point.
	for _, free := range []string{"0", "0.0", "0.00", "0.000"} {
		if s == free {
			return true
		}
	}
	return false
}
