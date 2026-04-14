package api

import (
	"encoding/base64"
	"encoding/json"
	"errors"
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

func mlErrorStatus(err error) int {
	var projectMismatchErr *mlmodel.ModelProjectMismatchError
	var inUseErr *mlmodel.ModelInUseError
	switch {
	case err == nil:
		return http.StatusOK
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

// HandleMLProviders handles requests for /api/ml-providers and /api/ml-providers/{name}
func (h *MLModelHandler) HandleMLProviders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	providerName := strings.TrimPrefix(r.URL.Path, "/api/ml-providers/")
	if providerName != r.URL.Path && providerName != "" {
		h.handleGetMLProvider(w, r, providerName)
		return
	}
	providers, err := h.service.ListProviderMetadata()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list ML providers: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(providers)
}

func (h *MLModelHandler) handleGetMLProvider(w http.ResponseWriter, r *http.Request, name string) {
	provider, err := h.service.GetProviderMetadata(name)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get ML provider: %v", err), mlErrorStatus(err))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(provider)
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

	// Check for training sub-resources
	if len(parts) >= 2 && parts[1] == "training" {
		if len(parts) >= 3 {
			switch parts[2] {
			case "complete":
				h.handleTrainingComplete(w, r, modelID)
				return
			case "fail":
				h.handleTrainingFail(w, r, modelID)
				return
			}
		}
	}

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
		http.Error(w, fmt.Sprintf("Failed to create model: %v", err), mlErrorStatus(err))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(model)
}

// handleGetMLModel handles GET /api/ml-models/{id}
func (h *MLModelHandler) handleGetMLModel(w http.ResponseWriter, r *http.Request, modelID string) {
	projectID := r.URL.Query().Get("project_id")
	if projectID == "" {
		http.Error(w, "project_id query parameter is required", http.StatusBadRequest)
		return
	}
	model, err := h.service.GetModelForProject(projectID, modelID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get model: %v", err), mlErrorStatus(err))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(model)
}

// handleUpdateMLModel handles PUT /api/ml-models/{id}
func (h *MLModelHandler) handleUpdateMLModel(w http.ResponseWriter, r *http.Request, modelID string) {
	projectID := r.URL.Query().Get("project_id")
	if projectID == "" {
		http.Error(w, "project_id query parameter is required", http.StatusBadRequest)
		return
	}
	var req models.ModelUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}
	model, err := h.service.UpdateModelForProject(projectID, modelID, &req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to update model: %v", err), mlErrorStatus(err))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(model)
}

// handleDeleteMLModel handles DELETE /api/ml-models/{id}
func (h *MLModelHandler) handleDeleteMLModel(w http.ResponseWriter, r *http.Request, modelID string) {
	projectID := r.URL.Query().Get("project_id")
	if projectID == "" {
		http.Error(w, "project_id query parameter is required", http.StatusBadRequest)
		return
	}
	if err := h.service.DeleteModelForProject(projectID, modelID); err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete model: %v", err), mlErrorStatus(err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleTrainingComplete handles POST /api/ml-models/{id}/training/complete
func (h *MLModelHandler) handleTrainingComplete(w http.ResponseWriter, r *http.Request, modelID string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ArtifactDataBase64 string                     `json:"artifact_data_base64"`
		PerformanceMetrics *models.PerformanceMetrics `json:"performance_metrics"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	if req.ArtifactDataBase64 == "" || req.PerformanceMetrics == nil {
		http.Error(w, "artifact_data_base64 and performance_metrics are required", http.StatusBadRequest)
		return
	}

	artifactData, err := base64.StdEncoding.DecodeString(req.ArtifactDataBase64)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid artifact_data_base64: %v", err), http.StatusBadRequest)
		return
	}

	if err := h.service.CompleteTraining(modelID, artifactData, req.PerformanceMetrics); err != nil {
		http.Error(w, fmt.Sprintf("Failed to complete training: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "training completed"})
}

// handleTrainingFail handles POST /api/ml-models/{id}/training/fail
func (h *MLModelHandler) handleTrainingFail(w http.ResponseWriter, r *http.Request, modelID string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	if req.Reason == "" {
		http.Error(w, "reason is required", http.StatusBadRequest)
		return
	}

	if err := h.service.FailTraining(modelID, req.Reason); err != nil {
		http.Error(w, fmt.Sprintf("Failed to mark training as failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "training failed"})
}
