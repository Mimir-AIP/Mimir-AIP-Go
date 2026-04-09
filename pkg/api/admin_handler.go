package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	adminpkg "github.com/mimir-aip/mimir-aip-go/pkg/admin"
)

// AdminHandler handles system-wide administrative operations.
type AdminHandler struct {
	service *adminpkg.Service
}

func NewAdminHandler(service *adminpkg.Service) *AdminHandler {
	return &AdminHandler{service: service}
}

func (h *AdminHandler) HandleAdminSettings(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/admin/settings")
	if path == "/factory-reset" || path == "factory-reset" {
		h.handleFactoryReset(w, r)
		return
	}
	http.Error(w, "Not found", http.StatusNotFound)
}

func (h *AdminHandler) handleFactoryReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.service == nil {
		http.Error(w, "Admin service is not configured", http.StatusNotImplemented)
		return
	}

	summary, err := h.service.FactoryReset()
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, adminpkg.ErrResetBlockedByActiveTasks) {
			status = http.StatusConflict
		}
		http.Error(w, err.Error(), status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(summary)
}
