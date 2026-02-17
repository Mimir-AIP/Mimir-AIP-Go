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
	service *project.Service
}

// NewProjectHandler creates a new project handler
func NewProjectHandler(service *project.Service) *ProjectHandler {
	return &ProjectHandler{
		service: service,
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

// HandleProject handles individual project operations
func (h *ProjectHandler) HandleProject(w http.ResponseWriter, r *http.Request) {
	// Extract project ID from path
	projectID := strings.TrimPrefix(r.URL.Path, "/api/projects/")
	if idx := strings.Index(projectID, "/"); idx != -1 {
		projectID = projectID[:idx]
	}

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

// HandleProjectClone handles project cloning
func (h *ProjectHandler) HandleProjectClone(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract project ID from path
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/projects/"), "/")
	if len(parts) < 2 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	projectID := parts[0]

	// Parse request body
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Clone project
	cloned, err := h.service.Clone(projectID, req.Name)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to clone project: %v", err), http.StatusInternalServerError)
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

// handleDelete deletes a project
func (h *ProjectHandler) handleDelete(w http.ResponseWriter, r *http.Request, projectID string) {
	if err := h.service.Delete(projectID); err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete project: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
