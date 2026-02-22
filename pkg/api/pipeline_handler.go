package api

import (
	"encoding/json"
	"fmt"
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
	// Check if this is an execute request
	if strings.HasSuffix(r.URL.Path, "/execute") {
		h.HandlePipelineExecute(w, r)
		return
	}

	// Extract pipeline ID from path
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

// HandlePipelineExecute handles pipeline execution by creating a WorkTask
func (h *PipelineHandler) HandlePipelineExecute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract pipeline ID from path
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/pipelines/"), "/")
	if len(parts) < 2 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	pipelineID := parts[0]

	// Get pipeline to verify it exists and get project ID
	pipeline, err := h.service.Get(pipelineID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Pipeline not found: %v", err), http.StatusNotFound)
		return
	}

	// Parse request body
	var req models.PipelineExecutionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Create WorkTask for pipeline execution
	workTask := &models.WorkTask{
		ID:          uuid.New().String(),
		Type:        models.WorkTaskTypePipelineExecution,
		Status:      models.WorkTaskStatusQueued,
		Priority:    1, // Default priority for manual executions
		SubmittedAt: time.Now(),
		ProjectID:   pipeline.ProjectID,
		TaskSpec: models.TaskSpec{
			PipelineID: pipelineID,
			ProjectID:  pipeline.ProjectID,
			Parameters: map[string]interface{}{
				"trigger_type": req.TriggerType,
				"triggered_by": req.TriggeredBy,
			},
		},
		ResourceRequirements: models.ResourceRequirements{
			CPU:    "500m", // Default resource requirements
			Memory: "1Gi",
			GPU:    false,
		},
		DataAccess: models.DataAccess{
			InputDatasets:  []string{},
			OutputLocation: fmt.Sprintf("s3://results/pipeline-%s/", pipelineID),
		},
	}

	// Add any custom parameters from the request
	if req.Parameters != nil {
		for key, value := range req.Parameters {
			workTask.TaskSpec.Parameters[key] = value
		}
	}

	// Enqueue the WorkTask
	if err := h.queue.Enqueue(workTask); err != nil {
		http.Error(w, fmt.Sprintf("Failed to enqueue work task: %v", err), http.StatusInternalServerError)
		return
	}

	// Return the WorkTask as the response
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"work_task_id": workTask.ID,
		"pipeline_id":  pipelineID,
		"status":       "queued",
		"message":      "Pipeline execution has been queued as a work task",
	})
}

// handleList lists all pipelines
func (h *PipelineHandler) handleList(w http.ResponseWriter, r *http.Request) {
	// Check if filtering by project
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

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(pipelines)
}

// handleCreate creates a new pipeline
func (h *PipelineHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var req models.PipelineCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	pipeline, err := h.service.Create(&req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create pipeline: %v", err), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(pipeline)
}

// handleGet retrieves a pipeline
func (h *PipelineHandler) handleGet(w http.ResponseWriter, r *http.Request, pipelineID string) {
	pipeline, err := h.service.Get(pipelineID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Pipeline not found: %v", err), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(pipeline)
}

// handleUpdate updates a pipeline
func (h *PipelineHandler) handleUpdate(w http.ResponseWriter, r *http.Request, pipelineID string) {
	var req models.PipelineUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	pipeline, err := h.service.Update(pipelineID, &req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to update pipeline: %v", err), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(pipeline)
}

// handleDelete deletes a pipeline
func (h *PipelineHandler) handleDelete(w http.ResponseWriter, r *http.Request, pipelineID string) {
	if err := h.service.Delete(pipelineID); err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete pipeline: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
