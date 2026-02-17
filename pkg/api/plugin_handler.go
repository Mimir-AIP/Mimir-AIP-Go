package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"github.com/mimir-aip/mimir-aip-go/pkg/plugins"
)

// PluginHandler handles plugin-related HTTP requests
type PluginHandler struct {
	service *plugins.Service
}

// NewPluginHandler creates a new plugin handler
func NewPluginHandler(service *plugins.Service) *PluginHandler {
	return &PluginHandler{
		service: service,
	}
}

// HandlePlugins handles plugin list and install operations
func (h *PluginHandler) HandlePlugins(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.handleList(w, r)
	case http.MethodPost:
		h.handleInstall(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandlePlugin handles individual plugin operations
func (h *PluginHandler) HandlePlugin(w http.ResponseWriter, r *http.Request) {
	// Extract plugin name from path
	pluginName := strings.TrimPrefix(r.URL.Path, "/api/plugins/")
	if idx := strings.Index(pluginName, "/"); idx != -1 {
		// Check for special endpoints
		parts := strings.Split(pluginName, "/")
		if len(parts) == 2 {
			switch parts[1] {
			case "reload":
				// Deprecated - orchestrator no longer loads plugins
				http.Error(w, "Plugin reload not supported - workers compile from source", http.StatusBadRequest)
				return
			}
		}
		pluginName = parts[0]
	}

	switch r.Method {
	case http.MethodGet:
		h.handleGet(w, r, pluginName)
	case http.MethodPut:
		h.handleUpdate(w, r, pluginName)
	case http.MethodDelete:
		h.handleUninstall(w, r, pluginName)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleList lists all plugins
func (h *PluginHandler) handleList(w http.ResponseWriter, r *http.Request) {
	plugins, err := h.service.ListPlugins()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list plugins: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(plugins)
}

// handleInstall installs a new plugin from a Git repository
func (h *PluginHandler) handleInstall(w http.ResponseWriter, r *http.Request) {
	var req models.PluginInstallRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Validate request
	if req.RepositoryURL == "" {
		http.Error(w, "repository_url is required", http.StatusBadRequest)
		return
	}

	plugin, err := h.service.InstallPlugin(&req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to install plugin: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(plugin)
}

// handleGet retrieves a specific plugin
func (h *PluginHandler) handleGet(w http.ResponseWriter, r *http.Request, pluginName string) {
	plugin, err := h.service.GetPluginMetadata(pluginName)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to get plugin: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(plugin)
}

// handleUpdate updates a plugin to the latest version
func (h *PluginHandler) handleUpdate(w http.ResponseWriter, r *http.Request, pluginName string) {
	var req models.PluginUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	gitRef := ""
	if req.GitRef != nil {
		gitRef = *req.GitRef
	}

	plugin, err := h.service.UpdatePlugin(pluginName, gitRef)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to update plugin: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(plugin)
}

// handleUninstall uninstalls a plugin
func (h *PluginHandler) handleUninstall(w http.ResponseWriter, r *http.Request, pluginName string) {
	if err := h.service.UninstallPlugin(pluginName); err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to uninstall plugin: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
