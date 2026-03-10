package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mimir-aip/mimir-aip-go/pkg/connectors"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

// ConnectorsHandler exposes the bundled connector catalog and materialization API.
type ConnectorsHandler struct {
	service *connectors.Service
}

// NewConnectorsHandler creates a new connectors API handler.
func NewConnectorsHandler(service *connectors.Service) *ConnectorsHandler {
	return &ConnectorsHandler{service: service}
}

// HandleConnectors handles GET /api/connectors and POST /api/connectors.
func (h *ConnectorsHandler) HandleConnectors(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(h.service.ListTemplates())
	case http.MethodPost:
		var req models.ConnectorSetupRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
			return
		}
		result, err := h.service.Materialize(&req)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to materialize connector: %v", err), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(result)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
