package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"github.com/mimir-aip/mimir-aip-go/pkg/scheduler"
)

// JobHandler handles scheduled job-related HTTP requests
type JobHandler struct {
	service *scheduler.Service
}

// NewJobHandler creates a new job handler
func NewJobHandler(service *scheduler.Service) *JobHandler {
	return &JobHandler{
		service: service,
	}
}

// HandleJobs handles job list and create operations
func (h *JobHandler) HandleJobs(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.handleList(w, r)
	case http.MethodPost:
		h.handleCreate(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandleJob handles individual job operations
func (h *JobHandler) HandleJob(w http.ResponseWriter, r *http.Request) {
	// Extract job ID from path
	jobID := strings.TrimPrefix(r.URL.Path, "/api/jobs/")
	if idx := strings.Index(jobID, "/"); idx != -1 {
		jobID = jobID[:idx]
	}

	switch r.Method {
	case http.MethodGet:
		h.handleGet(w, r, jobID)
	case http.MethodPut:
		h.handleUpdate(w, r, jobID)
	case http.MethodDelete:
		h.handleDelete(w, r, jobID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleList lists all scheduled jobs
func (h *JobHandler) handleList(w http.ResponseWriter, r *http.Request) {
	// Check if filtering by project
	projectID := r.URL.Query().Get("project_id")

	var jobs []*models.ScheduledJob
	var err error

	if projectID != "" {
		jobs, err = h.service.ListByProject(projectID)
	} else {
		jobs, err = h.service.List()
	}

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list jobs: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(jobs)
}

// handleCreate creates a new scheduled job
func (h *JobHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var req models.ScheduledJobCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	job, err := h.service.Create(&req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create job: %v", err), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(job)
}

// handleGet retrieves a scheduled job
func (h *JobHandler) handleGet(w http.ResponseWriter, r *http.Request, jobID string) {
	job, err := h.service.Get(jobID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Job not found: %v", err), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(job)
}

// handleUpdate updates a scheduled job
func (h *JobHandler) handleUpdate(w http.ResponseWriter, r *http.Request, jobID string) {
	var req models.ScheduledJobUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	job, err := h.service.Update(jobID, &req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to update job: %v", err), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(job)
}

// handleDelete deletes a scheduled job
func (h *JobHandler) handleDelete(w http.ResponseWriter, r *http.Request, jobID string) {
	if err := h.service.Delete(jobID); err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete job: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
