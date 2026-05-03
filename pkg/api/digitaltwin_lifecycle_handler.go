package api

import (
	"encoding/json"
	"fmt"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"net/http"
)

// handleListDigitalTwins handles GET /api/digital-twins
func (h *DigitalTwinHandler) handleListDigitalTwins(w http.ResponseWriter, r *http.Request) {
	projectID := r.URL.Query().Get("project_id")

	var twins []*models.DigitalTwin
	var err error

	if projectID != "" {
		twins, err = h.service.ListDigitalTwinsByProject(projectID)
	} else {
		http.Error(w, "project_id parameter is required", http.StatusBadRequest)
		return
	}

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list digital twins: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(twins)
}

// handleCreateDigitalTwin handles POST /api/digital-twins
func (h *DigitalTwinHandler) handleCreateDigitalTwin(w http.ResponseWriter, r *http.Request) {
	var req models.DigitalTwinCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	if err := req.Validate(); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	twin, err := h.service.CreateDigitalTwin(&req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create digital twin: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(twin)
}

// handleGetDigitalTwin handles GET /api/digital-twins/{id}
func (h *DigitalTwinHandler) handleGetDigitalTwin(w http.ResponseWriter, r *http.Request, id string) {
	twin, err := h.service.GetDigitalTwinForProject(projectIDFromRequest(r), id)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get digital twin: %v", err), digitalTwinErrorStatus(err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(twin)
}

// handleUpdateDigitalTwin handles PUT /api/digital-twins/{id}
func (h *DigitalTwinHandler) handleUpdateDigitalTwin(w http.ResponseWriter, r *http.Request, id string) {
	var req models.DigitalTwinUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	twin, err := h.service.UpdateDigitalTwinForProject(projectIDFromRequest(r), id, &req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to update digital twin: %v", err), digitalTwinErrorStatus(err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(twin)
}

// handleDeleteDigitalTwin handles DELETE /api/digital-twins/{id}
func (h *DigitalTwinHandler) handleDeleteDigitalTwin(w http.ResponseWriter, r *http.Request, id string) {
	if err := h.service.DeleteDigitalTwinForProject(projectIDFromRequest(r), id); err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete digital twin: %v", err), digitalTwinErrorStatus(err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleSyncDigitalTwin handles POST /api/digital-twins/{id}/sync
func (h *DigitalTwinHandler) handleSyncDigitalTwin(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	task, err := h.service.EnqueueSyncForProject(projectIDFromRequest(r), id)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to sync digital twin: %v", err), digitalTwinErrorStatus(err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"work_task_id":    task.ID,
		"digital_twin_id": id,
		"status":          "queued",
		"message":         "Digital twin sync has been queued as a work task",
	})
}
