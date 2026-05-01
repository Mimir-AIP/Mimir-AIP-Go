package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"github.com/mimir-aip/mimir-aip-go/pkg/ontology"
)

// OntologyHandler handles ontology-related HTTP requests
type OntologyHandler struct {
	service *ontology.Service
}

// NewOntologyHandler creates a new ontology handler
func NewOntologyHandler(service *ontology.Service) *OntologyHandler {
	return &OntologyHandler{
		service: service,
	}
}

func ontologyErrorStatus(err error) int {
	var projectMismatchErr *ontology.OntologyProjectMismatchError
	var inUseErr *ontology.OntologyInUseError
	var validationErr *ontology.OntologyValidationError
	switch {
	case err == nil:
		return http.StatusOK
	case errors.As(err, &validationErr):
		return http.StatusBadRequest
	case errors.As(err, &projectMismatchErr):
		return http.StatusForbidden
	case errors.As(err, &inUseErr):
		return http.StatusConflict
	case strings.Contains(err.Error(), "not found"):
		return http.StatusNotFound
	case strings.Contains(err.Error(), "invalid"), strings.Contains(err.Error(), "project_id is required"):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

// HandleOntologies handles ontology list and create operations
// GET /api/ontologies?project_id=<id> - List ontologies (optionally filtered by project)
// POST /api/ontologies - Create new ontology
func (h *OntologyHandler) HandleOntologies(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.handleList(w, r)
	case http.MethodPost:
		h.handleCreate(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandleOntology handles individual ontology operations
// GET /api/ontologies/{id} - Get ontology by ID
// PUT /api/ontologies/{id} - Update ontology
// DELETE /api/ontologies/{id} - Delete ontology
func (h *OntologyHandler) HandleOntology(w http.ResponseWriter, r *http.Request) {
	relativePath := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/ontologies/"), "/")
	parts := strings.Split(relativePath, "/")
	ontologyID := parts[0]
	action := ""
	if len(parts) > 1 {
		action = parts[1]
	}

	if ontologyID == "" {
		http.Error(w, "ontology ID is required", http.StatusBadRequest)
		return
	}

	if action != "" {
		switch action {
		case "compiled":
			if r.Method != http.MethodGet {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}
			h.handleGetCompiled(w, r, ontologyID)
		case "diagnostics":
			if r.Method != http.MethodGet {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}
			h.handleGetDiagnostics(w, r, ontologyID)
		default:
			http.NotFound(w, r)
		}
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.handleGet(w, r, ontologyID)
	case http.MethodPut:
		h.handleUpdate(w, r, ontologyID)
	case http.MethodDelete:
		h.handleDelete(w, r, ontologyID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *OntologyHandler) HandleOntologyValidation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req models.OntologyValidationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}
	resp := h.service.ValidateOntology(&req)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleList handles GET /api/ontologies
func (h *OntologyHandler) handleList(w http.ResponseWriter, r *http.Request) {
	projectID := r.URL.Query().Get("project_id")

	var ontologies []*models.Ontology
	var err error

	if projectID != "" {
		ontologies, err = h.service.GetProjectOntologies(projectID)
	} else {
		http.Error(w, "project_id query parameter is required", http.StatusBadRequest)
		return
	}

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list ontologies: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ontologies)
}

// handleCreate handles POST /api/ontologies
func (h *OntologyHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var req models.OntologyCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	ontology, err := h.service.CreateOntology(&req)
	if err != nil {
		h.writeOntologyError(w, "Failed to create ontology", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(ontology)
}

// handleGet handles GET /api/ontologies/{id}
func (h *OntologyHandler) handleGet(w http.ResponseWriter, r *http.Request, ontologyID string) {
	projectID := r.URL.Query().Get("project_id")
	if projectID == "" {
		http.Error(w, "project_id query parameter is required", http.StatusBadRequest)
		return
	}
	ontologyRecord, err := h.service.GetOntologyForProject(projectID, ontologyID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get ontology: %v", err), ontologyErrorStatus(err))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ontologyRecord)
}

func (h *OntologyHandler) handleGetCompiled(w http.ResponseWriter, r *http.Request, ontologyID string) {
	projectID := r.URL.Query().Get("project_id")
	if projectID == "" {
		http.Error(w, "project_id query parameter is required", http.StatusBadRequest)
		return
	}
	compiled, err := h.service.GetCompiledOntologyForProject(projectID, ontologyID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get compiled ontology: %v", err), ontologyErrorStatus(err))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(compiled)
}

func (h *OntologyHandler) handleGetDiagnostics(w http.ResponseWriter, r *http.Request, ontologyID string) {
	projectID := r.URL.Query().Get("project_id")
	if projectID == "" {
		http.Error(w, "project_id query parameter is required", http.StatusBadRequest)
		return
	}
	compiled, err := h.service.GetCompiledOntologyForProject(projectID, ontologyID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get ontology diagnostics: %v", err), ontologyErrorStatus(err))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(compiled.Diagnostics)
}

// handleUpdate handles PUT /api/ontologies/{id}
func (h *OntologyHandler) handleUpdate(w http.ResponseWriter, r *http.Request, ontologyID string) {
	projectID := r.URL.Query().Get("project_id")
	if projectID == "" {
		http.Error(w, "project_id query parameter is required", http.StatusBadRequest)
		return
	}
	var req models.OntologyUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}
	ontologyRecord, err := h.service.UpdateOntologyForProject(projectID, ontologyID, &req)
	if err != nil {
		h.writeOntologyError(w, "Failed to update ontology", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ontologyRecord)
}

// handleDelete handles DELETE /api/ontologies/{id}
func (h *OntologyHandler) handleDelete(w http.ResponseWriter, r *http.Request, ontologyID string) {
	projectID := r.URL.Query().Get("project_id")
	if projectID == "" {
		http.Error(w, "project_id query parameter is required", http.StatusBadRequest)
		return
	}
	if err := h.service.DeleteOntologyForProject(projectID, ontologyID); err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete ontology: %v", err), ontologyErrorStatus(err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *OntologyHandler) writeOntologyError(w http.ResponseWriter, prefix string, err error) {
	var validationErr *ontology.OntologyValidationError
	if errors.As(err, &validationErr) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.OntologyValidationResponse{Valid: false, Diagnostics: validationErr.Diagnostics})
		return
	}
	http.Error(w, fmt.Sprintf("%s: %v", prefix, err), ontologyErrorStatus(err))
}
