package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"github.com/mimir-aip/mimir-aip-go/pkg/storage"
)

// StoragePluginHandler handles dynamic storage plugin installation/management.
type StoragePluginHandler struct {
	service *storage.Service
}

// NewStoragePluginHandler creates a new StoragePluginHandler.
func NewStoragePluginHandler(service *storage.Service) *StoragePluginHandler {
	return &StoragePluginHandler{service: service}
}

// HandleStoragePlugins handles GET /api/storage-plugins and POST /api/storage-plugins.
func (h *StoragePluginHandler) HandleStoragePlugins(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.handleList(w, r)
	case http.MethodPost:
		h.handleInstall(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandleStoragePlugin handles GET/DELETE /api/storage-plugins/{name}.
func (h *StoragePluginHandler) HandleStoragePlugin(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/api/storage-plugins/")
	name = strings.Trim(name, "/")
	if name == "" {
		http.Error(w, "plugin name is required", http.StatusBadRequest)
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

func (h *StoragePluginHandler) handleList(w http.ResponseWriter, r *http.Request) {
	plugins, err := h.service.ListExternalPlugins()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list storage plugins: %v", err), http.StatusInternalServerError)
		return
	}
	if plugins == nil {
		plugins = []*models.ExternalStoragePlugin{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(plugins)
}

func (h *StoragePluginHandler) handleInstall(w http.ResponseWriter, r *http.Request) {
	var req models.ExternalStoragePluginInstallRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}
	if req.RepositoryURL == "" {
		http.Error(w, "repository_url is required", http.StatusBadRequest)
		return
	}

	record, err := h.service.InstallExternalPlugin(&req)
	if err != nil {
		// Return 422 so the caller gets the error record (status=error) in the body.
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(record)
}

func (h *StoragePluginHandler) handleGet(w http.ResponseWriter, r *http.Request, name string) {
	record, err := h.service.GetExternalPlugin(name)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to get storage plugin: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(record)
}

func (h *StoragePluginHandler) handleUninstall(w http.ResponseWriter, r *http.Request, name string) {
	if err := h.service.UninstallExternalPlugin(name); err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to uninstall storage plugin: %v", err), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
