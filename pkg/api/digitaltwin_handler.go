package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	automationpkg "github.com/mimir-aip/mimir-aip-go/pkg/automation"
	"github.com/mimir-aip/mimir-aip-go/pkg/digitaltwin"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
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
	twin, err := h.service.GetDigitalTwin(id)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get digital twin: %v", err), http.StatusNotFound)
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

	twin, err := h.service.UpdateDigitalTwin(id, &req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to update digital twin: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(twin)
}

// handleDeleteDigitalTwin handles DELETE /api/digital-twins/{id}
func (h *DigitalTwinHandler) handleDeleteDigitalTwin(w http.ResponseWriter, r *http.Request, id string) {
	if err := h.service.DeleteDigitalTwin(id); err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete digital twin: %v", err), http.StatusInternalServerError)
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

	task, err := h.service.EnqueueSync(id)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to sync digital twin: %v", err), http.StatusInternalServerError)
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

// handleDigitalTwinEntities handles GET /api/digital-twins/{id}/entities
func (h *DigitalTwinHandler) handleDigitalTwinEntities(w http.ResponseWriter, r *http.Request, twinID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	entities, err := h.service.ListEntities(twinID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list entities: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entities)
}

// handleDigitalTwinEntity handles GET/PUT /api/digital-twins/{twinID}/entities/{entityID}
func (h *DigitalTwinHandler) handleDigitalTwinEntity(w http.ResponseWriter, r *http.Request, twinID, entityID string) {
	switch r.Method {
	case http.MethodGet:
		entity, err := h.service.GetEntity(entityID)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to get entity: %v", err), http.StatusNotFound)
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

		entity, err := h.service.UpdateEntity(entityID, &req)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to update entity: %v", err), http.StatusInternalServerError)
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
	limit := 20
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		parsedLimit, err := strconv.Atoi(rawLimit)
		if err != nil || parsedLimit <= 0 {
			http.Error(w, "limit must be a positive integer", http.StatusBadRequest)
			return
		}
		limit = parsedLimit
	}
	revisions, err := h.service.GetEntityHistory(entityID, limit)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get entity history: %v", err), http.StatusInternalServerError)
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

	entities, err := h.service.GetRelatedEntities(twinID, entityID, relationshipType)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get related entities: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entities)
}

// handleDigitalTwinQuery handles POST /api/digital-twins/{id}/query
func (h *DigitalTwinHandler) handleDigitalTwinQuery(w http.ResponseWriter, r *http.Request, twinID string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.QueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	if err := req.Validate(); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	result, err := h.service.Query(twinID, &req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to execute query: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// handleDigitalTwinPredict handles POST /api/digital-twins/{id}/predict
func (h *DigitalTwinHandler) handleDigitalTwinPredict(w http.ResponseWriter, r *http.Request, twinID string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check if it's a batch prediction request
	var rawReq map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&rawReq); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Check if inputs field exists (batch request)
	if inputs, ok := rawReq["inputs"]; ok && inputs != nil {
		// Batch prediction
		reqBytes, _ := json.Marshal(rawReq)
		var batchReq models.BatchPredictionRequest
		if err := json.Unmarshal(reqBytes, &batchReq); err != nil {
			http.Error(w, fmt.Sprintf("Invalid batch prediction request: %v", err), http.StatusBadRequest)
			return
		}

		if err := batchReq.Validate(); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}

		predictions, err := h.service.BatchPredict(twinID, &batchReq)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to run batch predictions: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(predictions)
	} else {
		// Single prediction
		reqBytes, _ := json.Marshal(rawReq)
		var predReq models.PredictionRequest
		if err := json.Unmarshal(reqBytes, &predReq); err != nil {
			http.Error(w, fmt.Sprintf("Invalid prediction request: %v", err), http.StatusBadRequest)
			return
		}

		if err := predReq.Validate(); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}

		prediction, err := h.service.Predict(twinID, &predReq)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to run prediction: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(prediction)
	}
}

func (h *DigitalTwinHandler) handleDigitalTwinRuns(w http.ResponseWriter, r *http.Request, twinID string) {
	switch r.Method {
	case http.MethodGet:
		limit := parsePositiveInt(r.URL.Query().Get("limit"), 50)
		runs, err := h.processor.ListRuns(twinID, limit)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to list twin processing runs: %v", err), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(runs)
	case http.MethodPost:
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
	alert, err := h.processor.ReviewAlert(twinID, alertID, &req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to review alert event: %v", err), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(alert)
}

func (h *DigitalTwinHandler) handleDigitalTwinAutomations(w http.ResponseWriter, r *http.Request, twinID string) {
	twin, err := h.service.GetDigitalTwin(twinID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get digital twin: %v", err), http.StatusNotFound)
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

// handleDigitalTwinScenarios handles GET/POST /api/digital-twins/{id}/scenarios
func (h *DigitalTwinHandler) handleDigitalTwinScenarios(w http.ResponseWriter, r *http.Request, twinID string) {
	switch r.Method {
	case http.MethodGet:
		scenarios, err := h.service.ListScenarios(twinID)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to list scenarios: %v", err), http.StatusInternalServerError)
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

		scenario, err := h.service.CreateScenario(twinID, &req)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to create scenario: %v", err), http.StatusInternalServerError)
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
	switch r.Method {
	case http.MethodGet:
		scenario, err := h.service.GetScenario(scenarioID)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to get scenario: %v", err), http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(scenario)

	case http.MethodDelete:
		if err := h.service.DeleteScenario(scenarioID); err != nil {
			http.Error(w, fmt.Sprintf("Failed to delete scenario: %v", err), http.StatusInternalServerError)
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
		actions, err := h.service.ListActions(twinID)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to list actions: %v", err), http.StatusInternalServerError)
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

		action, err := h.service.CreateAction(twinID, &req)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to create action: %v", err), http.StatusInternalServerError)
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
	switch r.Method {
	case http.MethodGet:
		action, err := h.service.GetAction(actionID)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to get action: %v", err), http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(action)

	case http.MethodDelete:
		if err := h.service.DeleteAction(actionID); err != nil {
			http.Error(w, fmt.Sprintf("Failed to delete action: %v", err), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
