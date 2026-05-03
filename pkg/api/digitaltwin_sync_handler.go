package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

func (h *DigitalTwinHandler) handleDigitalTwinSyncRuns(w http.ResponseWriter, r *http.Request, twinID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	limit := 50
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		parsedLimit, err := strconv.Atoi(rawLimit)
		if err != nil || parsedLimit <= 0 {
			http.Error(w, "limit must be a positive integer", http.StatusBadRequest)
			return
		}
		limit = parsedLimit
	}
	runs, err := h.service.ListSyncRunsForProject(projectIDFromRequest(r), twinID, limit)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list sync runs: %v", err), digitalTwinErrorStatus(err))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(runs)
}

func (h *DigitalTwinHandler) handleDigitalTwinSyncRun(w http.ResponseWriter, r *http.Request, twinID, runID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	run, err := h.service.GetSyncRunForProject(projectIDFromRequest(r), twinID, runID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get sync run: %v", err), digitalTwinErrorStatus(err))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(run)
}
