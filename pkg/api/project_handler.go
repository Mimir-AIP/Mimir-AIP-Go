package api

import (
	"encoding/json"
	"errors"
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

// HandleProject handles individual project operations and component associations.
// Routes:
//   - GET/PUT/DELETE /api/projects/{id}
//   - POST/DELETE    /api/projects/{id}/{componentType}/{componentId}
func (h *ProjectHandler) HandleProject(w http.ResponseWriter, r *http.Request) {
	// Parse path segments after /api/projects/
	trimmed := strings.TrimPrefix(r.URL.Path, "/api/projects/")
	parts := strings.SplitN(trimmed, "/", 3)

	// Three segments means component association route
	if len(parts) >= 3 && parts[2] != "" {
		h.HandleProjectComponent(w, r)
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

// HandleProjectComponent handles POST/DELETE for project component associations.
// Path format: /api/projects/{id}/{componentType}/{componentId}
// Supported componentTypes: pipelines, ontologies, mlmodels, digitaltwins, storage
func (h *ProjectHandler) HandleProjectComponent(w http.ResponseWriter, r *http.Request) {
	// Parse path: /api/projects/{id}/{componentType}/{componentId}
	trimmed := strings.TrimPrefix(r.URL.Path, "/api/projects/")
	parts := strings.SplitN(trimmed, "/", 3)
	if len(parts) < 3 {
		http.Error(w, "Invalid path: expected /api/projects/{id}/{componentType}/{componentId}", http.StatusBadRequest)
		return
	}
	projectID := parts[0]
	componentType := parts[1]
	componentID := parts[2]

	var err error
	switch r.Method {
	case http.MethodPost:
		switch componentType {
		case "pipelines":
			err = h.service.AddPipeline(projectID, componentID)
		case "ontologies":
			err = h.service.AddOntology(projectID, componentID)
		case "mlmodels":
			err = h.service.AddMLModel(projectID, componentID)
		case "digitaltwins":
			err = h.service.AddDigitalTwin(projectID, componentID)
		case "storage":
			err = h.service.AddStorage(projectID, componentID)
		default:
			http.Error(w, fmt.Sprintf("Unknown component type: %s", componentType), http.StatusBadRequest)
			return
		}
	case http.MethodDelete:
		switch componentType {
		case "pipelines":
			err = h.service.RemovePipeline(projectID, componentID)
		case "ontologies":
			err = h.service.RemoveOntology(projectID, componentID)
		case "mlmodels":
			err = h.service.RemoveMLModel(projectID, componentID)
		case "digitaltwins":
			err = h.service.RemoveDigitalTwin(projectID, componentID)
		case "storage":
			err = h.service.RemoveStorage(projectID, componentID)
		default:
			http.Error(w, fmt.Sprintf("Unknown component type: %s", componentType), http.StatusBadRequest)
			return
		}
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, project.ErrComponentNotFound):
			status = http.StatusNotFound
		case errors.Is(err, project.ErrComponentProjectMismatch):
			status = http.StatusBadRequest
		}
		http.Error(w, fmt.Sprintf("Failed to update project component: %v", err), status)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
