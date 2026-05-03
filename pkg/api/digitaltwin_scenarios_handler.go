package api

import (
	"encoding/json"
	"fmt"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"net/http"
)

func (h *DigitalTwinHandler) handleDigitalTwinScenarios(w http.ResponseWriter, r *http.Request, twinID string) {
	switch r.Method {
	case http.MethodGet:
		scenarios, err := h.service.ListScenariosForProject(projectIDFromRequest(r), twinID)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to list scenarios: %v", err), digitalTwinErrorStatus(err))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(scenarios)
	case http.MethodPost:
		var req models.ScenarioCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
			return
		}
		if err := req.Validate(); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}
		scenario, err := h.service.CreateScenarioForProject(projectIDFromRequest(r), twinID, &req)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to create scenario: %v", err), digitalTwinErrorStatus(err))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(scenario)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleDigitalTwinScenario handles GET/DELETE /api/digital-twins/{twinID}/scenarios/{scenarioID}
func (h *DigitalTwinHandler) handleDigitalTwinScenario(w http.ResponseWriter, r *http.Request, twinID, scenarioID string) {
	if _, err := h.getOwnedTwin(r, twinID); err != nil {
		http.Error(w, fmt.Sprintf("Failed to get digital twin: %v", err), digitalTwinErrorStatus(err))
		return
	}
	switch r.Method {
	case http.MethodGet:
		scenario, err := h.service.GetScenarioInTwin(twinID, scenarioID)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to get scenario: %v", err), digitalTwinErrorStatus(err))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(scenario)
	case http.MethodDelete:
		if err := h.service.DeleteScenarioInTwin(twinID, scenarioID); err != nil {
			http.Error(w, fmt.Sprintf("Failed to delete scenario: %v", err), digitalTwinErrorStatus(err))
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleDigitalTwinActions handles GET/POST /api/digital-twins/{id}/actions
func (h *DigitalTwinHandler) handleDigitalTwinActions(w http.ResponseWriter, r *http.Request, twinID string) {
	switch r.Method {
	case http.MethodGet:
		actions, err := h.service.ListActionsForProject(projectIDFromRequest(r), twinID)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to list actions: %v", err), digitalTwinErrorStatus(err))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(actions)
	case http.MethodPost:
		var req models.ActionCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
			return
		}
		if err := req.Validate(); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}
		action, err := h.service.CreateActionForProject(projectIDFromRequest(r), twinID, &req)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to create action: %v", err), digitalTwinErrorStatus(err))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(action)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleDigitalTwinAction handles GET/DELETE /api/digital-twins/{twinID}/actions/{actionID}
func (h *DigitalTwinHandler) handleDigitalTwinAction(w http.ResponseWriter, r *http.Request, twinID, actionID string) {
	if _, err := h.getOwnedTwin(r, twinID); err != nil {
		http.Error(w, fmt.Sprintf("Failed to get digital twin: %v", err), digitalTwinErrorStatus(err))
		return
	}
	switch r.Method {
	case http.MethodGet:
		action, err := h.service.GetActionInTwin(twinID, actionID)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to get action: %v", err), digitalTwinErrorStatus(err))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(action)
	case http.MethodDelete:
		if err := h.service.DeleteActionInTwin(twinID, actionID); err != nil {
			http.Error(w, fmt.Sprintf("Failed to delete action: %v", err), digitalTwinErrorStatus(err))
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
