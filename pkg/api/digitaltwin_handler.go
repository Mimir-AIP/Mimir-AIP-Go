package api

import (
	"errors"
	automationpkg "github.com/mimir-aip/mimir-aip-go/pkg/automation"
	"github.com/mimir-aip/mimir-aip-go/pkg/digitaltwin"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"net/http"
	"strings"
)

// DigitalTwinHandler handles digital twin HTTP requests.
type DigitalTwinHandler struct {
	service           *digitaltwin.Service
	processor         *digitaltwin.Processor
	automationService *automationpkg.Service
}

// NewDigitalTwinHandler creates a new digital twin handler.
func NewDigitalTwinHandler(service *digitaltwin.Service, processor *digitaltwin.Processor, automationService *automationpkg.Service) *DigitalTwinHandler {
	return &DigitalTwinHandler{
		service:           service,
		processor:         processor,
		automationService: automationService,
	}
}

func digitalTwinErrorStatus(err error) int {
	var projectMismatchErr *digitaltwin.DigitalTwinProjectMismatchError
	switch {
	case err == nil:
		return http.StatusOK
	case errors.As(err, &projectMismatchErr):
		return http.StatusForbidden
	case strings.Contains(err.Error(), "belongs to project"):
		return http.StatusForbidden
	case strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "does not belong"):
		return http.StatusNotFound
	case strings.Contains(err.Error(), "project_id is required") || strings.Contains(err.Error(), "invalid") || strings.Contains(err.Error(), "ontology"):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

func projectIDFromRequest(r *http.Request) string {
	return strings.TrimSpace(r.URL.Query().Get("project_id"))
}

func (h *DigitalTwinHandler) getOwnedTwin(r *http.Request, twinID string) (*models.DigitalTwin, error) {
	return h.service.GetDigitalTwinForProject(projectIDFromRequest(r), twinID)
}

// HandleDigitalTwins handles requests for /api/digital-twins
// GET: List all digital twins (optionally filtered by project_id)
// POST: Create a new digital twin
func (h *DigitalTwinHandler) HandleDigitalTwins(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.handleListDigitalTwins(w, r)
	case http.MethodPost:
		h.handleCreateDigitalTwin(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandleDigitalTwin handles requests for /api/digital-twins/{id}
// GET: Get a specific digital twin
// PUT: Update a digital twin
// DELETE: Delete a digital twin
func (h *DigitalTwinHandler) HandleDigitalTwin(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/digital-twins/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "Digital twin ID required", http.StatusBadRequest)
		return
	}
	twinID := parts[0]

	// Check for sub-resources
	if len(parts) > 1 {
		switch parts[1] {
		case "sync":
			h.handleSyncDigitalTwin(w, r, twinID)
			return
		case "entities":
			if len(parts) == 2 {
				h.handleDigitalTwinEntities(w, r, twinID)
			} else if len(parts) >= 4 && parts[3] == "related" {
				h.handleDigitalTwinEntityRelated(w, r, twinID, parts[2])
			} else if len(parts) >= 4 && parts[3] == "history" {
				h.handleDigitalTwinEntityHistory(w, r, twinID, parts[2])
			} else {
				h.handleDigitalTwinEntity(w, r, twinID, parts[2])
			}
			return
		case "query":
			h.handleDigitalTwinQuery(w, r, twinID)
			return
		case "state":
			h.handleDigitalTwinState(w, r, twinID)
			return
		case "predict":
			h.handleDigitalTwinPredict(w, r, twinID)
			return
		case "scenarios":
			if len(parts) == 2 {
				h.handleDigitalTwinScenarios(w, r, twinID)
			} else {
				h.handleDigitalTwinScenario(w, r, twinID, parts[2])
			}
			return
		case "actions":
			if len(parts) == 2 {
				h.handleDigitalTwinActions(w, r, twinID)
			} else {
				h.handleDigitalTwinAction(w, r, twinID, parts[2])
			}
			return
		case "runs":
			if len(parts) == 2 {
				h.handleDigitalTwinRuns(w, r, twinID)
			} else {
				h.handleDigitalTwinRun(w, r, twinID, parts[2])
			}
			return
		case "alerts":
			if len(parts) == 2 {
				h.handleDigitalTwinAlerts(w, r, twinID)
			} else if len(parts) == 4 && parts[3] == "approval" {
				h.handleDigitalTwinAlertApproval(w, r, twinID, parts[2])
			} else {
				http.Error(w, "Alert route not found", http.StatusNotFound)
			}
			return
		case "history":
			if len(parts) == 3 && parts[2] == "runs" {
				h.handleDigitalTwinSyncRuns(w, r, twinID)
				return
			}
			if len(parts) == 4 && parts[2] == "runs" {
				h.handleDigitalTwinSyncRun(w, r, twinID, parts[3])
				return
			}
		case "automations":
			if len(parts) == 2 {
				h.handleDigitalTwinAutomations(w, r, twinID)
			} else {
				h.handleDigitalTwinAutomation(w, r, twinID, parts[2])
			}
			return
		}
	}

	switch r.Method {
	case http.MethodGet:
		h.handleGetDigitalTwin(w, r, twinID)
	case http.MethodPut:
		h.handleUpdateDigitalTwin(w, r, twinID)
	case http.MethodDelete:
		h.handleDeleteDigitalTwin(w, r, twinID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
