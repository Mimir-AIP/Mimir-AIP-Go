package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/mimir-aip/mimir-aip-go/pkg/llm"
)

// OpenAICompatProvider implements llm.Provider for any OpenAI-compatible API
// (OpenAI itself, local Ollama, vLLM, LM Studio, etc.) by pointing at a
// configurable base URL.
type OpenAICompatProvider struct {
	name    string
	baseURL string
	apiKey  string
	client  *http.Client
}

// NewOpenAICompatProvider creates a provider that speaks the OpenAI wire
// format.  baseURL should be the root URL without a trailing slash,
// e.g. "https://api.openai.com" or "http://localhost:11434".
func NewOpenAICompatProvider(name, baseURL, apiKey string) *OpenAICompatProvider {
	return &OpenAICompatProvider{
		name:    name,
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

// Name returns the provider identifier.
func (p *OpenAICompatProvider) Name() string { return p.name }

// ListModels calls GET {baseURL}/v1/models and returns the model list.
// IsFree is always false for OpenAI-compat providers (billing is external).
func (p *OpenAICompatProvider) ListModels(ctx context.Context) ([]llm.Model, error) {
	url := p.baseURL + "/v1/models"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("openai_compat: build request: %w", err)
	}
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openai_compat: GET %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("openai_compat: GET %s: status %d: %s", url, resp.StatusCode, string(body))
	}

	var result struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("openai_compat: decode models: %w", err)
	}

	models := make([]llm.Model, 0, len(result.Data))
	for _, m := range result.Data {
		models = append(models, llm.Model{
			ID:           m.ID,
			Name:         m.ID,
			IsFree:       false,
			ProviderName: p.name,
		})
	}
	return models, nil
}

// Complete sends a chat-completion request.
func (p *OpenAICompatProvider) Complete(ctx context.Context, req llm.CompletionRequest) (llm.CompletionResponse, error) {
	endpoint := p.baseURL + "/v1/chat/completions"
	return doOpenAICompatCompletion(ctx, p.client, endpoint, p.apiKey, "", req)
}

// ── Shared helper ──────────────────────────────────────────────────────────────

// openAIRequest is the wire format for a chat-completion request.
type openAIRequest struct {
	Model       string       `json:"model"`
	Messages    []llm.Message `json:"messages"`
	MaxTokens   int          `json:"max_tokens,omitempty"`
	Temperature float64      `json:"temperature"`
}

// openAIResponse is the minimal wire format for a chat-completion response.
type openAIResponse struct {
	Model   string `json:"model"`
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

// doOpenAICompatCompletion sends a POST to endpoint using the OpenAI chat wire
// format.  extraHeaders is a raw JSON object string (e.g. `{"X-Title":"foo"}`)
// that is merged into the request headers — pass "" to skip.  Used by the
// OpenRouter provider to add its required custom headers without duplicating the
// HTTP logic.
func doOpenAICompatCompletion(
	ctx context.Context,
	client *http.Client,
	endpoint string,
	apiKey string,
	extraHeaders string,
	req llm.CompletionRequest,
) (llm.CompletionResponse, error) {
	body := openAIRequest{
		Model:       req.Model,
		Messages:    req.Messages,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return llm.CompletionResponse{}, fmt.Errorf("llm: marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return llm.CompletionResponse{}, fmt.Errorf("llm: build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	}

	// Apply any provider-specific extra headers.
	if extraHeaders != "" {
		var headers map[string]string
		if jsonErr := json.Unmarshal([]byte(extraHeaders), &headers); jsonErr == nil {
			for k, v := range headers {
				httpReq.Header.Set(k, v)
			}
		}
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		return llm.CompletionResponse{}, fmt.Errorf("llm: POST %s: %w", endpoint, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(resp.Body)
		return llm.CompletionResponse{}, fmt.Errorf("llm: POST %s: status %d: %s", endpoint, resp.StatusCode, string(errBody))
	}

	var result openAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return llm.CompletionResponse{}, fmt.Errorf("llm: decode response: %w", err)
	}

	content := ""
	if len(result.Choices) > 0 {
		content = result.Choices[0].Message.Content
	}

	return llm.CompletionResponse{
		Content:      content,
		Model:        result.Model,
		InputTokens:  result.Usage.PromptTokens,
		OutputTokens: result.Usage.CompletionTokens,
	}, nil
}
