package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"github.com/mimir-aip/mimir-aip-go/pkg/storage"
)

// StorageHandler handles storage-related API requests
type StorageHandler struct {
	service *storage.Service
}

// NewStorageHandler creates a new storage handler
func NewStorageHandler(service *storage.Service) *StorageHandler {
	return &StorageHandler{
		service: service,
	}
}

// HandleStorageConfigs handles requests to /api/storage/configs
func (h *StorageHandler) HandleStorageConfigs(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listStorageConfigs(w, r)
	case http.MethodPost:
		h.createStorageConfig(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandleStorageConfig handles requests to /api/storage/configs/{id}
func (h *StorageHandler) HandleStorageConfig(w http.ResponseWriter, r *http.Request) {
	// Extract ID from path
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 4 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	id := parts[3]

	switch r.Method {
	case http.MethodGet:
		h.getStorageConfig(w, r, id)
	case http.MethodPut, http.MethodPatch:
		h.updateStorageConfig(w, r, id)
	case http.MethodDelete:
		h.deleteStorageConfig(w, r, id)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandleStorageStore handles requests to /api/storage/store
func (h *StorageHandler) HandleStorageStore(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.StorageStoreRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	result, err := h.service.Store(req.StorageID, req.CIRData)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to store data: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// HandleStorageRetrieve handles requests to /api/storage/retrieve
func (h *StorageHandler) HandleStorageRetrieve(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.StorageQueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	results, err := h.service.Retrieve(req.StorageID, req.Query)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to retrieve data: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

// HandleStorageUpdate handles requests to /api/storage/update
func (h *StorageHandler) HandleStorageUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.StorageUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	result, err := h.service.Update(req.StorageID, req.Query, req.Updates)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to update data: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// HandleStorageDelete handles requests to /api/storage/delete
func (h *StorageHandler) HandleStorageDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.StorageDeleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	result, err := h.service.Delete(req.StorageID, req.Query)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete data: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// HandleStorageHealth handles requests to /api/storage/{id}/health
func (h *StorageHandler) HandleStorageHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract ID from path
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 3 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	storageID := parts[2]

	healthy, err := h.service.HealthCheck(storageID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"healthy": false,
			"error":   err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"healthy": healthy,
	})
}

// listStorageConfigs lists all storage configurations
func (h *StorageHandler) listStorageConfigs(w http.ResponseWriter, r *http.Request) {
	// Check if filtering by project
	projectID := r.URL.Query().Get("project_id")

	var configs []*models.StorageConfig
	var err error

	if projectID != "" {
		configs, err = h.service.GetProjectStorageConfigs(projectID)
	} else {
		// List all (would need to add this to service)
		http.Error(w, "project_id parameter is required", http.StatusBadRequest)
		return
	}

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list storage configs: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(configs)
}

// createStorageConfig creates a new storage configuration
func (h *StorageHandler) createStorageConfig(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ProjectID  string                 `json:"project_id"`
		PluginType string                 `json:"plugin_type"`
		Config     map[string]interface{} `json:"config"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	if req.ProjectID == "" {
		http.Error(w, "project_id is required", http.StatusBadRequest)
		return
	}

	if req.PluginType == "" {
		http.Error(w, "plugin_type is required", http.StatusBadRequest)
		return
	}

	config, err := h.service.CreateStorageConfig(req.ProjectID, req.PluginType, req.Config)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create storage config: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(config)
}

// getStorageConfig retrieves a storage configuration by ID
func (h *StorageHandler) getStorageConfig(w http.ResponseWriter, r *http.Request, id string) {
	config, err := h.service.GetStorageConfig(id)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get storage config: %v", err), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(config)
}

// updateStorageConfig updates a storage configuration
func (h *StorageHandler) updateStorageConfig(w http.ResponseWriter, r *http.Request, id string) {
	var req struct {
		Config map[string]interface{} `json:"config"`
		Active *bool                  `json:"active"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	if err := h.service.UpdateStorageConfig(id, req.Config, req.Active); err != nil {
		http.Error(w, fmt.Sprintf("Failed to update storage config: %v", err), http.StatusInternalServerError)
		return
	}

	// Get updated config
	config, err := h.service.GetStorageConfig(id)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get updated config: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(config)
}

// deleteStorageConfig deletes a storage configuration
func (h *StorageHandler) deleteStorageConfig(w http.ResponseWriter, r *http.Request, id string) {
	if err := h.service.DeleteStorageConfig(id); err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete storage config: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
