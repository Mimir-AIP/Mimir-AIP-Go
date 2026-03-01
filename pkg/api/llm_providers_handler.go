package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/mimir-aip/mimir-aip-go/pkg/llm"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

// LLMProvidersHandler handles dynamic LLM provider installation/management.
type LLMProvidersHandler struct {
	service *llm.Service
}

// NewLLMProvidersHandler creates a new LLMProvidersHandler.
func NewLLMProvidersHandler(service *llm.Service) *LLMProvidersHandler {
	return &LLMProvidersHandler{service: service}
}

// HandleLLMProviders handles GET /api/llm/providers and POST /api/llm/providers.
func (h *LLMProvidersHandler) HandleLLMProviders(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.handleList(w, r)
	case http.MethodPost:
		h.handleInstall(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandleLLMProvider handles GET/DELETE /api/llm/providers/{name}.
func (h *LLMProvidersHandler) HandleLLMProvider(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/api/llm/providers/")
	name = strings.Trim(name, "/")
	if name == "" {
		http.Error(w, "provider name is required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.handleGet(w, r, name)
	case http.MethodDelete:
		h.handleUninstall(w, r, name)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *LLMProvidersHandler) handleList(w http.ResponseWriter, r *http.Request) {
	providers, err := h.service.ListExternalProviders()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list LLM providers: %v", err), http.StatusInternalServerError)
		return
	}
	if providers == nil {
		providers = []*models.ExternalLLMProvider{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(providers) //nolint:errcheck
}

func (h *LLMProvidersHandler) handleInstall(w http.ResponseWriter, r *http.Request) {
	var req models.ExternalLLMProviderInstallRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}
	if req.RepositoryURL == "" {
		http.Error(w, "repository_url is required", http.StatusBadRequest)
		return
	}

	record, err := h.service.InstallExternalProvider(&req)
	if err != nil {
		// Return 422 so the caller gets the error record in the body.
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()}) //nolint:errcheck
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(record) //nolint:errcheck
}

func (h *LLMProvidersHandler) handleGet(w http.ResponseWriter, r *http.Request, name string) {
	record, err := h.service.GetExternalProvider(name)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to get LLM provider: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(record) //nolint:errcheck
}

func (h *LLMProvidersHandler) handleUninstall(w http.ResponseWriter, r *http.Request, name string) {
	if err := h.service.UninstallExternalProvider(name); err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to uninstall LLM provider: %v", err), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
