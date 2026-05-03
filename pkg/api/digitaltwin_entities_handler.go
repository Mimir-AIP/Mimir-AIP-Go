package api

import (
	"encoding/json"
	"fmt"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"net/http"
	"strconv"
	"strings"
)

// handleDigitalTwinState handles GET /api/digital-twins/{id}/state?at_run=...
func (h *DigitalTwinHandler) handleDigitalTwinState(w http.ResponseWriter, r *http.Request, twinID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	atRun := strings.TrimSpace(r.URL.Query().Get("at_run"))
	if atRun == "" {
		http.Error(w, "at_run query parameter is required", http.StatusBadRequest)
		return
	}
	state, err := h.service.GetStateAtRunForProject(projectIDFromRequest(r), twinID, atRun)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to reconstruct twin state: %v", err), digitalTwinErrorStatus(err))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(state)
}

// handleDigitalTwinEntities handles GET /api/digital-twins/{id}/entities
func (h *DigitalTwinHandler) handleDigitalTwinEntities(w http.ResponseWriter, r *http.Request, twinID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	entities, err := h.service.ListEntitiesForProject(projectIDFromRequest(r), twinID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list entities: %v", err), digitalTwinErrorStatus(err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entities)
}

// handleDigitalTwinEntity handles GET/PUT /api/digital-twins/{twinID}/entities/{entityID}
func (h *DigitalTwinHandler) handleDigitalTwinEntity(w http.ResponseWriter, r *http.Request, twinID, entityID string) {
	if _, err := h.getOwnedTwin(r, twinID); err != nil {
		http.Error(w, fmt.Sprintf("Failed to get digital twin: %v", err), digitalTwinErrorStatus(err))
		return
	}
	switch r.Method {
	case http.MethodGet:
		entity, err := h.service.GetEntityInTwin(twinID, entityID)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to get entity: %v", err), digitalTwinErrorStatus(err))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(entity)
	case http.MethodPut:
		var req models.EntityUpdateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
			return
		}
		if err := req.Validate(); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}
		entity, err := h.service.UpdateEntityInTwin(twinID, entityID, &req)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to update entity: %v", err), digitalTwinErrorStatus(err))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(entity)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleDigitalTwinEntityHistory handles GET /api/digital-twins/{twinID}/entities/{entityID}/history
func (h *DigitalTwinHandler) handleDigitalTwinEntityHistory(w http.ResponseWriter, r *http.Request, twinID, entityID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if _, err := h.getOwnedTwin(r, twinID); err != nil {
		http.Error(w, fmt.Sprintf("Failed to get digital twin: %v", err), digitalTwinErrorStatus(err))
		return
	}
	limit := 20
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		parsedLimit, err := strconv.Atoi(rawLimit)
		if err != nil || parsedLimit <= 0 {
			http.Error(w, "limit must be a positive integer", http.StatusBadRequest)
			return
		}
		limit = parsedLimit
	}
	revisions, err := h.service.GetEntityHistoryInTwin(twinID, entityID, limit)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get entity history: %v", err), digitalTwinErrorStatus(err))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(revisions)
}

// handleDigitalTwinEntityRelated handles GET /api/digital-twins/{twinID}/entities/{entityID}/related
func (h *DigitalTwinHandler) handleDigitalTwinEntityRelated(w http.ResponseWriter, r *http.Request, twinID, entityID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	relationshipType := r.URL.Query().Get("relationship")

	entities, err := h.service.GetRelatedEntitiesForProject(projectIDFromRequest(r), twinID, entityID, relationshipType)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get related entities: %v", err), digitalTwinErrorStatus(err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entities)
}
