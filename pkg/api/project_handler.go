package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"github.com/mimir-aip/mimir-aip-go/pkg/project"
)

// ProjectHandler handles project-related HTTP requests
type ProjectHandler struct {
	service       *project.Service
	stateProvider *ProjectStateProvider
}

// NewProjectHandler creates a new project handler
func NewProjectHandler(service *project.Service, stateProvider *ProjectStateProvider) *ProjectHandler {
	return &ProjectHandler{
		service:       service,
		stateProvider: stateProvider,
	}
}

// HandleProjects handles project list and create operations
func (h *ProjectHandler) HandleProjects(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.handleList(w, r)
	case http.MethodPost:
		h.handleCreate(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandleProject handles individual project operations.
// Routes:
//   - GET/PUT/DELETE /api/projects/{id}
//   - POST            /api/projects/{id}/archive
//   - POST            /api/projects/{id}/clone
//   - GET             /api/projects/{id}/state-summary
func (h *ProjectHandler) HandleProject(w http.ResponseWriter, r *http.Request) {
	trimmed := strings.TrimPrefix(r.URL.Path, "/api/projects/")
	parts := strings.SplitN(trimmed, "/", 3)
	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "Project ID required", http.StatusBadRequest)
		return
	}
	if len(parts) >= 2 && parts[1] == "state-summary" {
		h.handleStateSummary(w, r, parts[0])
		return
	}
	if len(parts) >= 2 && parts[1] == "archive" && (len(parts) == 2 || parts[2] == "") {
		h.handleArchive(w, r, parts[0])
		return
	}
	if len(parts) >= 2 && parts[1] == "clone" && (len(parts) == 2 || parts[2] == "") {
		h.handleClone(w, r, parts[0])
		return
	}
	if len(parts) >= 2 {
		http.Error(w, "Invalid project subresource", http.StatusNotFound)
		return
	}

	projectID := parts[0]

	switch r.Method {
	case http.MethodGet:
		h.handleGet(w, r, projectID)
	case http.MethodPut:
		h.handleUpdate(w, r, projectID)
	case http.MethodDelete:
		h.handleDelete(w, r, projectID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *ProjectHandler) handleArchive(w http.ResponseWriter, r *http.Request, projectID string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := h.service.Archive(projectID); err != nil {
		status := http.StatusInternalServerError
		if strings.Contains(err.Error(), "project not found") {
			status = http.StatusNotFound
		}
		http.Error(w, fmt.Sprintf("Failed to archive project: %v", err), status)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ProjectHandler) handleClone(w http.ResponseWriter, r *http.Request, projectID string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	cloned, err := h.service.Clone(projectID, req.Name)
	if err != nil {
		status := http.StatusBadRequest
		if strings.Contains(err.Error(), "project not found") {
			status = http.StatusNotFound
		}
		http.Error(w, fmt.Sprintf("Failed to clone project: %v", err), status)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(cloned)
}

// handleList lists all projects
func (h *ProjectHandler) handleList(w http.ResponseWriter, r *http.Request) {
	query := &models.ProjectListQuery{
		Status: r.URL.Query().Get("status"),
	}

	projects, err := h.service.List(query)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list projects: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(projects)
}

// handleCreate creates a new project
func (h *ProjectHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var req models.ProjectCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	project, err := h.service.Create(&req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create project: %v", err), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(project)
}

// handleGet retrieves a project
func (h *ProjectHandler) handleGet(w http.ResponseWriter, r *http.Request, projectID string) {
	project, err := h.service.Get(projectID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Project not found: %v", err), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(project)
}
func (h *ProjectHandler) handleStateSummary(w http.ResponseWriter, r *http.Request, projectID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.stateProvider == nil {
		http.Error(w, "Project state summary is not configured", http.StatusNotImplemented)
		return
	}
	summary, err := h.stateProvider.Summary(projectID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load project state summary: %v", err), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(summary)
}

// handleUpdate updates a project
func (h *ProjectHandler) handleUpdate(w http.ResponseWriter, r *http.Request, projectID string) {
	var req models.ProjectUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	project, err := h.service.Update(projectID, &req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to update project: %v", err), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(project)
}

// handleDelete permanently deletes a project.
func (h *ProjectHandler) handleDelete(w http.ResponseWriter, r *http.Request, projectID string) {
	if err := h.service.Delete(projectID); err != nil {
		status := http.StatusInternalServerError
		if strings.Contains(err.Error(), "project not found") {
			status = http.StatusNotFound
		}
		http.Error(w, fmt.Sprintf("Failed to delete project: %v", err), status)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
