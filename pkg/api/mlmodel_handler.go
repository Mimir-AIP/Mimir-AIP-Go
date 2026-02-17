package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/mimir-aip/mimir-aip-go/pkg/mlmodel"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

// MLModelHandler handles ML model HTTP requests
type MLModelHandler struct {
	service *mlmodel.Service
}

// NewMLModelHandler creates a new ML model handler
func NewMLModelHandler(service *mlmodel.Service) *MLModelHandler {
	return &MLModelHandler{
		service: service,
	}
}

// HandleMLModels handles requests for /api/ml-models
// GET: List all models (optionally filtered by project_id)
// POST: Create a new model
func (h *MLModelHandler) HandleMLModels(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.handleListMLModels(w, r)
	case http.MethodPost:
		h.handleCreateMLModel(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandleMLModel handles requests for /api/ml-models/{id}
// GET: Get a specific model
// PUT: Update a model
// DELETE: Delete a model
func (h *MLModelHandler) HandleMLModel(w http.ResponseWriter, r *http.Request) {
	// Extract model ID from path
	path := strings.TrimPrefix(r.URL.Path, "/api/ml-models/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "Model ID required", http.StatusBadRequest)
		return
	}
	modelID := parts[0]

	switch r.Method {
	case http.MethodGet:
		h.handleGetMLModel(w, r, modelID)
	case http.MethodPut:
		h.handleUpdateMLModel(w, r, modelID)
	case http.MethodDelete:
		h.handleDeleteMLModel(w, r, modelID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandleMLModelRecommendation handles POST /api/ml-models/recommend
func (h *MLModelHandler) HandleMLModelRecommendation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.ModelRecommendationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	if err := req.Validate(); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	recommendation, err := h.service.RecommendModelType(req.ProjectID, req.OntologyID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to recommend model type: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(recommendation)
}

// HandleMLModelTraining handles POST /api/ml-models/train
func (h *MLModelHandler) HandleMLModelTraining(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.ModelTrainingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	if err := req.Validate(); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	model, err := h.service.StartTraining(&req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to start training: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(model)
}

// handleListMLModels handles GET /api/ml-models
func (h *MLModelHandler) handleListMLModels(w http.ResponseWriter, r *http.Request) {
	projectID := r.URL.Query().Get("project_id")

	var mlModels []*models.MLModel
	var err error

	if projectID != "" {
		mlModels, err = h.service.ListProjectModels(projectID)
	} else {
		http.Error(w, "project_id parameter is required", http.StatusBadRequest)
		return
	}

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list models: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(mlModels)
}

// handleCreateMLModel handles POST /api/ml-models
func (h *MLModelHandler) handleCreateMLModel(w http.ResponseWriter, r *http.Request) {
	var req models.ModelCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	if err := req.Validate(); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	model, err := h.service.CreateModel(&req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create model: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(model)
}

// handleGetMLModel handles GET /api/ml-models/{id}
func (h *MLModelHandler) handleGetMLModel(w http.ResponseWriter, r *http.Request, modelID string) {
	model, err := h.service.GetModel(modelID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Model not found", http.StatusNotFound)
		} else {
			http.Error(w, fmt.Sprintf("Failed to get model: %v", err), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(model)
}

// handleUpdateMLModel handles PUT /api/ml-models/{id}
func (h *MLModelHandler) handleUpdateMLModel(w http.ResponseWriter, r *http.Request, modelID string) {
	var req models.ModelUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	model, err := h.service.UpdateModel(modelID, &req)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Model not found", http.StatusNotFound)
		} else {
			http.Error(w, fmt.Sprintf("Failed to update model: %v", err), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(model)
}

// handleDeleteMLModel handles DELETE /api/ml-models/{id}
func (h *MLModelHandler) handleDeleteMLModel(w http.ResponseWriter, r *http.Request, modelID string) {
	if err := h.service.DeleteModel(modelID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Model not found", http.StatusNotFound)
		} else {
			http.Error(w, fmt.Sprintf("Failed to delete model: %v", err), http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
