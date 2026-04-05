package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"github.com/mimir-aip/mimir-aip-go/pkg/pipeline"
	"github.com/mimir-aip/mimir-aip-go/pkg/queue"
)

// PipelineHandler handles pipeline-related HTTP requests
type PipelineHandler struct {
	service *pipeline.Service
	queue   *queue.Queue
}

// NewPipelineHandler creates a new pipeline handler
func NewPipelineHandler(service *pipeline.Service, q *queue.Queue) *PipelineHandler {
	return &PipelineHandler{
		service: service,
		queue:   q,
	}
}

func sanitizePipeline(p *models.Pipeline) *models.Pipeline {
	if p == nil {
		return nil
	}
	clone := *p
	if p.TriggerConfig != nil {
		trigger := *p.TriggerConfig
		trigger.Secret = ""
		clone.TriggerConfig = &trigger
	}
	return &clone
}

func (h *PipelineHandler) enqueuePipelineExecution(p *models.Pipeline, req *models.PipelineTriggerRequest) (*models.WorkTask, error) {
	triggerType := strings.TrimSpace(req.TriggerType)
	if triggerType == "" {
		triggerType = "manual"
	}
	triggeredBy := strings.TrimSpace(req.TriggeredBy)
	if triggeredBy == "" {
		switch triggerType {
		case "webhook":
			triggeredBy = "pipeline_webhook"
		case "system":
			triggeredBy = "system"
		default:
			triggeredBy = "manual_request"
		}
	}

	workTask := &models.WorkTask{
		ID:          uuid.New().String(),
		Type:        models.WorkTaskTypePipelineExecution,
		Status:      models.WorkTaskStatusQueued,
		Priority:    1,
		SubmittedAt: time.Now().UTC(),
		ProjectID:   p.ProjectID,
		TaskSpec: models.TaskSpec{
			PipelineID: p.ID,
			ProjectID:  p.ProjectID,
			Parameters: map[string]interface{}{
				"trigger_type":  triggerType,
				"triggered_by":  triggeredBy,
				"pipeline_type": p.Type,
			},
		},
		ResourceRequirements: models.ResourceRequirements{CPU: "500m", Memory: "1Gi", GPU: false},
		DataAccess: models.DataAccess{
			InputDatasets:  []string{},
			OutputLocation: fmt.Sprintf("s3://results/pipeline-%s/", p.ID),
		},
	}
	if req.Parameters != nil {
		for key, value := range req.Parameters {
			workTask.TaskSpec.Parameters[key] = value
		}
	}
	if req.SourceEventID != "" {
		workTask.TaskSpec.Parameters["source_event_id"] = req.SourceEventID
	}
	if err := h.queue.Enqueue(workTask); err != nil {
		return nil, err
	}
	return workTask, nil
}

func writeQueuedPipelineResponse(w http.ResponseWriter, workTask *models.WorkTask, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"work_task_id": workTask.ID,
		"pipeline_id":  workTask.TaskSpec.PipelineID,
		"status":       "queued",
		"message":      message,
	})
}

// HandlePipelines handles pipeline list and create operations
func (h *PipelineHandler) HandlePipelines(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.handleList(w, r)
	case http.MethodPost:
		h.handleCreate(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandlePipeline handles individual pipeline operations
func (h *PipelineHandler) HandlePipeline(w http.ResponseWriter, r *http.Request) {
	if strings.HasSuffix(r.URL.Path, "/execute") {
		h.HandlePipelineExecute(w, r)
		return
	}
	if strings.HasSuffix(r.URL.Path, "/trigger") {
		h.HandlePipelineTrigger(w, r)
		return
	}
	if strings.HasSuffix(r.URL.Path, "/webhook") {
		h.HandlePipelineWebhook(w, r)
		return
	}
	if strings.HasSuffix(r.URL.Path, "/checkpoints") {
		h.HandlePipelineCheckpoint(w, r)
		return
	}

	pipelineID := strings.TrimPrefix(r.URL.Path, "/api/pipelines/")
	if idx := strings.Index(pipelineID, "/"); idx != -1 {
		pipelineID = pipelineID[:idx]
	}

	switch r.Method {
	case http.MethodGet:
		h.handleGet(w, r, pipelineID)
	case http.MethodPut:
		h.handleUpdate(w, r, pipelineID)
	case http.MethodDelete:
		h.handleDelete(w, r, pipelineID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandlePipelineCheckpoint handles GET/PUT /api/pipelines/{id}/checkpoints?step_name=...&scope=...
func (h *PipelineHandler) HandlePipelineCheckpoint(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/pipelines/"), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] != "checkpoints" {
		http.Error(w, "Invalid path: expected /api/pipelines/{id}/checkpoints", http.StatusBadRequest)
		return
	}
	pipelineID := parts[0]

	pipelineDef, err := h.service.Get(pipelineID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Pipeline not found: %v", err), http.StatusNotFound)
		return
	}

	stepName := r.URL.Query().Get("step_name")
	if stepName == "" {
		http.Error(w, "step_name query parameter is required", http.StatusBadRequest)
		return
	}
	scope := r.URL.Query().Get("scope")

	switch r.Method {
	case http.MethodGet:
		checkpoint, err := h.service.GetCheckpoint(pipelineDef.ProjectID, pipelineID, stepName, scope)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to load checkpoint: %v", err), http.StatusInternalServerError)
			return
		}
		if checkpoint == nil {
			http.Error(w, "Checkpoint not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(checkpoint)
	case http.MethodPut:
		var checkpoint models.PipelineCheckpoint
		if err := json.NewDecoder(r.Body).Decode(&checkpoint); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
			return
		}
		checkpoint.ProjectID = pipelineDef.ProjectID
		checkpoint.PipelineID = pipelineID
		checkpoint.StepName = stepName
		checkpoint.Scope = scope
		if err := h.service.SaveCheckpoint(&checkpoint); err != nil {
			http.Error(w, fmt.Sprintf("Failed to save checkpoint: %v", err), http.StatusConflict)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(checkpoint)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandlePipelineExecute handles pipeline execution by creating a WorkTask
func (h *PipelineHandler) HandlePipelineExecute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/pipelines/"), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] != "execute" {
		http.Error(w, "Invalid path: expected /api/pipelines/{id}/execute", http.StatusBadRequest)
		return
	}
	pipelineDef, err := h.service.Get(parts[0])
	if err != nil {
		http.Error(w, fmt.Sprintf("Pipeline not found: %v", err), http.StatusNotFound)
		return
	}
	var req models.PipelineExecutionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}
	workTask, err := h.enqueuePipelineExecution(pipelineDef, &models.PipelineTriggerRequest{
		TriggerType: req.TriggerType,
		TriggeredBy: req.TriggeredBy,
		Parameters:  req.Parameters,
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to enqueue work task: %v", err), http.StatusInternalServerError)
		return
	}
	writeQueuedPipelineResponse(w, workTask, "Pipeline execution has been queued as a work task")
}

// HandlePipelineTrigger handles POST /api/pipelines/{id}/trigger for manual or system-triggered execution.
func (h *PipelineHandler) HandlePipelineTrigger(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/pipelines/"), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] != "trigger" {
		http.Error(w, "Invalid path: expected /api/pipelines/{id}/trigger", http.StatusBadRequest)
		return
	}
	pipelineDef, err := h.service.Get(parts[0])
	if err != nil {
		http.Error(w, fmt.Sprintf("Pipeline not found: %v", err), http.StatusNotFound)
		return
	}
	if pipelineDef.TriggerConfig != nil && !pipelineDef.TriggerConfig.AllowManual {
		http.Error(w, "manual trigger is disabled for this pipeline", http.StatusForbidden)
		return
	}
	var req models.PipelineTriggerRequest
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
			http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
			return
		}
	}
	if err := req.Validate(); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}
	if req.TriggerType == "webhook" {
		http.Error(w, "use /webhook endpoint for webhook-triggered execution", http.StatusBadRequest)
		return
	}
	workTask, err := h.enqueuePipelineExecution(pipelineDef, &req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to enqueue pipeline trigger: %v", err), http.StatusInternalServerError)
		return
	}
	writeQueuedPipelineResponse(w, workTask, "Pipeline trigger has been queued as a work task")
}

// HandlePipelineWebhook handles POST /api/pipelines/{id}/webhook for authenticated remote triggers.
func (h *PipelineHandler) HandlePipelineWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/pipelines/"), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] != "webhook" {
		http.Error(w, "Invalid path: expected /api/pipelines/{id}/webhook", http.StatusBadRequest)
		return
	}
	pipelineDef, err := h.service.Get(parts[0])
	if err != nil {
		http.Error(w, fmt.Sprintf("Pipeline not found: %v", err), http.StatusNotFound)
		return
	}
	if pipelineDef.TriggerConfig == nil || !pipelineDef.TriggerConfig.Webhook || strings.TrimSpace(pipelineDef.TriggerConfig.Secret) == "" {
		http.Error(w, "webhook trigger is not configured for this pipeline", http.StatusForbidden)
		return
	}

	providedToken := strings.TrimSpace(r.Header.Get("X-Mimir-Webhook-Token"))
	if providedToken == "" {
		providedToken = r.URL.Query().Get("token")
	}
	var req models.PipelineTriggerRequest
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
			http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
			return
		}
	}
	if providedToken == "" {
		providedToken = req.WebhookToken
	}
	if providedToken != pipelineDef.TriggerConfig.Secret {
		http.Error(w, "invalid webhook token", http.StatusUnauthorized)
		return
	}
	req.TriggerType = "webhook"
	if req.TriggeredBy == "" {
		req.TriggeredBy = "pipeline_webhook"
	}
	if err := req.Validate(); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}
	workTask, err := h.enqueuePipelineExecution(pipelineDef, &req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to enqueue webhook trigger: %v", err), http.StatusInternalServerError)
		return
	}
	writeQueuedPipelineResponse(w, workTask, "Pipeline webhook trigger has been queued as a work task")
}

// handleList lists all pipelines
func (h *PipelineHandler) handleList(w http.ResponseWriter, r *http.Request) {
	projectID := r.URL.Query().Get("project_id")

	var pipelines []*models.Pipeline
	var err error
	if projectID != "" {
		pipelines, err = h.service.ListByProject(projectID)
	} else {
		pipelines, err = h.service.List()
	}
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list pipelines: %v", err), http.StatusInternalServerError)
		return
	}
	sanitized := make([]*models.Pipeline, 0, len(pipelines))
	for _, p := range pipelines {
		sanitized = append(sanitized, sanitizePipeline(p))
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(sanitized)
}

// handleCreate creates a new pipeline
func (h *PipelineHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var req models.PipelineCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	pipelineDef, err := h.service.Create(&req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create pipeline: %v", err), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(sanitizePipeline(pipelineDef))
}

// handleGet retrieves a pipeline
func (h *PipelineHandler) handleGet(w http.ResponseWriter, r *http.Request, pipelineID string) {
	pipelineDef, err := h.service.Get(pipelineID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Pipeline not found: %v", err), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(sanitizePipeline(pipelineDef))
}

// handleUpdate updates a pipeline
func (h *PipelineHandler) handleUpdate(w http.ResponseWriter, r *http.Request, pipelineID string) {
	var req models.PipelineUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	pipelineDef, err := h.service.Update(pipelineID, &req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to update pipeline: %v", err), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(sanitizePipeline(pipelineDef))
}

// handleDelete deletes a pipeline
func (h *PipelineHandler) handleDelete(w http.ResponseWriter, r *http.Request, pipelineID string) {
	if err := h.service.Delete(pipelineID); err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete pipeline: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
