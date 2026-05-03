package api

import (
	"encoding/json"
	"fmt"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"net/http"
	"strconv"
)

func (h *DigitalTwinHandler) handleDigitalTwinRuns(w http.ResponseWriter, r *http.Request, twinID string) {
	switch r.Method {
	case http.MethodGet:
		if _, err := h.getOwnedTwin(r, twinID); err != nil {
			http.Error(w, fmt.Sprintf("Failed to get digital twin: %v", err), digitalTwinErrorStatus(err))
			return
		}
		limit := parsePositiveInt(r.URL.Query().Get("limit"), 50)
		runs, err := h.processor.ListRuns(twinID, limit)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to list twin processing runs: %v", err), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(runs)
	case http.MethodPost:
		if _, err := h.getOwnedTwin(r, twinID); err != nil {
			http.Error(w, fmt.Sprintf("Failed to get digital twin: %v", err), digitalTwinErrorStatus(err))
			return
		}
		run, err := h.processor.RequestRun(twinID, &models.TwinProcessingRunCreateRequest{
			TriggerType: models.TwinProcessingTriggerTypeManual,
			TriggerRef:  "api/manual",
		})
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to queue twin processing run: %v", err), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(run)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *DigitalTwinHandler) handleDigitalTwinRun(w http.ResponseWriter, r *http.Request, twinID, runID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if _, err := h.getOwnedTwin(r, twinID); err != nil {
		http.Error(w, fmt.Sprintf("Failed to get digital twin: %v", err), digitalTwinErrorStatus(err))
		return
	}
	run, err := h.processor.GetRun(twinID, runID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get twin processing run: %v", err), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(run)
}

func (h *DigitalTwinHandler) handleDigitalTwinAlerts(w http.ResponseWriter, r *http.Request, twinID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if _, err := h.getOwnedTwin(r, twinID); err != nil {
		http.Error(w, fmt.Sprintf("Failed to get digital twin: %v", err), digitalTwinErrorStatus(err))
		return
	}
	limit := parsePositiveInt(r.URL.Query().Get("limit"), 100)
	alerts, err := h.processor.ListAlerts(twinID, limit)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list alert events: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(alerts)
}

func (h *DigitalTwinHandler) handleDigitalTwinAlertApproval(w http.ResponseWriter, r *http.Request, twinID, alertID string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req models.AlertApprovalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}
	if err := req.Validate(); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}
	if _, err := h.getOwnedTwin(r, twinID); err != nil {
		http.Error(w, fmt.Sprintf("Failed to get digital twin: %v", err), digitalTwinErrorStatus(err))
		return
	}
	alert, err := h.processor.ReviewAlert(twinID, alertID, &req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to review alert event: %v", err), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(alert)
}

func (h *DigitalTwinHandler) handleDigitalTwinAutomations(w http.ResponseWriter, r *http.Request, twinID string) {
	twin, err := h.getOwnedTwin(r, twinID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get digital twin: %v", err), digitalTwinErrorStatus(err))
		return
	}
	switch r.Method {
	case http.MethodGet:
		automations, err := h.automationService.ListByProject(twin.ProjectID)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to list automations: %v", err), http.StatusInternalServerError)
			return
		}
		filtered := make([]*models.Automation, 0)
		for _, automation := range automations {
			if automation.TargetType == models.AutomationTargetTypeDigitalTwin && automation.TargetID == twinID {
				filtered = append(filtered, automation)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(filtered)
	case http.MethodPost:
		var req models.AutomationCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
			return
		}
		req.ProjectID = twin.ProjectID
		req.TargetType = models.AutomationTargetTypeDigitalTwin
		req.TargetID = twinID
		if req.TriggerType == "" {
			req.TriggerType = models.AutomationTriggerTypePipelineCompleted
		}
		if req.ActionType == "" {
			req.ActionType = models.AutomationActionTypeProcessTwin
		}
		automation, err := h.automationService.Create(&req)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to create automation: %v", err), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(automation)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *DigitalTwinHandler) handleDigitalTwinAutomation(w http.ResponseWriter, r *http.Request, twinID, automationID string) {
	if _, err := h.getOwnedTwin(r, twinID); err != nil {
		http.Error(w, fmt.Sprintf("Failed to get digital twin: %v", err), digitalTwinErrorStatus(err))
		return
	}
	automation, err := h.automationService.Get(automationID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get automation: %v", err), http.StatusNotFound)
		return
	}
	if automation.TargetType != models.AutomationTargetTypeDigitalTwin || automation.TargetID != twinID {
		http.Error(w, "Automation does not belong to this digital twin", http.StatusNotFound)
		return
	}
	switch r.Method {
	case http.MethodGet:
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(automation)
	case http.MethodPut:
		var req models.AutomationUpdateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
			return
		}
		updated, err := h.automationService.Update(automationID, &req)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to update automation: %v", err), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(updated)
	case http.MethodDelete:
		if err := h.automationService.Delete(automationID); err != nil {
			http.Error(w, fmt.Sprintf("Failed to delete automation: %v", err), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func parsePositiveInt(raw string, fallback int) int {
	if raw == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}
