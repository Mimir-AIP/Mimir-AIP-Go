package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/mimir-aip/mimir-aip-go/pkg/digitaltwin"
)

// TwinProcessingHandler exposes twin-processing run execution endpoints.
type TwinProcessingHandler struct {
	processor *digitaltwin.Processor
}

func NewTwinProcessingHandler(processor *digitaltwin.Processor) *TwinProcessingHandler {
	return &TwinProcessingHandler{processor: processor}
}

// HandleInternalTwinRuns handles worker-authenticated twin run execution requests.
// Supported path: POST /api/internal/twin-runs/{id}/execute
func (h *TwinProcessingHandler) HandleInternalTwinRuns(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/internal/twin-runs/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] != "execute" {
		http.Error(w, "Invalid path: expected /api/internal/twin-runs/{id}/execute", http.StatusBadRequest)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	run, err := h.processor.ExecuteRun(parts[0])
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to execute twin processing run: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(run)
}
