package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/mimir-aip/mimir-aip-go/pkg/analysis"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

// AnalysisHandler exposes resolver review queues and autonomous insights.
type AnalysisHandler struct {
	service *analysis.Service
}

func NewAnalysisHandler(service *analysis.Service) *AnalysisHandler {
	return &AnalysisHandler{service: service}
}

func (h *AnalysisHandler) HandleResolverRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		ProjectID  string   `json:"project_id"`
		StorageIDs []string `json:"storage_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}
	run, items, err := h.service.RunResolver(req.ProjectID, req.StorageIDs)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to run resolver analysis: %v", err), http.StatusBadRequest)
		return
	}
	metrics, err := h.service.ResolverMetrics(req.ProjectID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read resolver metrics: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{"run": run, "review_items": items, "metrics": metrics})
}

func (h *AnalysisHandler) HandleResolverMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	projectID := r.URL.Query().Get("project_id")
	if projectID == "" {
		http.Error(w, "project_id is required", http.StatusBadRequest)
		return
	}
	metrics, err := h.service.ResolverMetrics(projectID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load resolver metrics: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

func (h *AnalysisHandler) HandleReviewItems(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	projectID := r.URL.Query().Get("project_id")
	if projectID == "" {
		http.Error(w, "project_id is required", http.StatusBadRequest)
		return
	}
	items, err := h.service.ListReviewItems(projectID, r.URL.Query().Get("status"))
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list review items: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

func (h *AnalysisHandler) HandleReviewItem(w http.ResponseWriter, r *http.Request) {
	trimmed := strings.TrimPrefix(r.URL.Path, "/api/reviews/")
	parts := strings.Split(strings.Trim(trimmed, "/"), "/")
	if len(parts) != 2 || parts[1] != "decision" {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req models.ReviewDecisionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}
	item, err := h.service.DecideReviewItem(parts[0], &req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to update review item: %v", err), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(item)
}

func (h *AnalysisHandler) HandleInsights(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		projectID := r.URL.Query().Get("project_id")
		if projectID == "" {
			http.Error(w, "project_id is required", http.StatusBadRequest)
			return
		}
		minConfidence := 0.0
		if raw := r.URL.Query().Get("min_confidence"); raw != "" {
			parsed, err := strconv.ParseFloat(raw, 64)
			if err != nil {
				http.Error(w, "min_confidence must be numeric", http.StatusBadRequest)
				return
			}
			minConfidence = parsed
		}
		insights, err := h.service.ListInsights(projectID, r.URL.Query().Get("severity"), minConfidence)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to list insights: %v", err), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(insights)
	case http.MethodPost:
		var req struct {
			ProjectID string `json:"project_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
			return
		}
		run, insights, err := h.service.GenerateProjectInsights(req.ProjectID)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to generate insights: %v", err), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{"run": run, "insights": insights})
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
