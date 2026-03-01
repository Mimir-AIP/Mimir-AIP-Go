package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/mimir-aip/mimir-aip-go/pkg/llm"
)

// LLMHandler handles LLM-related HTTP requests.
type LLMHandler struct {
	service *llm.Service // may be nil when LLM is not configured
}

// NewLLMHandler creates a new LLMHandler.  service may be nil.
func NewLLMHandler(service *llm.Service) *LLMHandler {
	return &LLMHandler{service: service}
}

// llmModelsResponse is the JSON shape returned by GET /api/llm/models.
type llmModelsResponse struct {
	Provider string     `json:"provider"`
	Enabled  bool       `json:"enabled"`
	Models   []llm.Model `json:"models"`
}

// HandleLLMModels handles GET /api/llm/models.
// Returns the provider name, enabled flag, and available model list.
// Never returns 5xx — errors produce an empty model list (graceful degrade).
func (h *LLMHandler) HandleLLMModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	enabled := h.service.IsEnabled()

	resp := llmModelsResponse{
		Provider: h.service.ProviderName(),
		Enabled:  enabled,
		Models:   []llm.Model{},
	}

	if enabled {
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()

		models, err := h.service.ListModels(ctx)
		if err == nil && models != nil {
			resp.Models = models
		}
		// On error: log but return the empty model list — never 5xx.
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp) //nolint:errcheck
}
