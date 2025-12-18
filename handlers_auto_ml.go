package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/ML"
	"github.com/gorilla/mux"
)

// handleGetMLCapabilities returns the ML capabilities discovered from an ontology
func (s *Server) handleGetMLCapabilities(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ontologyID := vars["id"]

	if ontologyID == "" {
		http.Error(w, "ontology ID is required", http.StatusBadRequest)
		return
	}

	// Create analyzer
	analyzer := ml.NewOntologyAnalyzer(s.persistence)

	// Analyze capabilities
	capabilities, err := analyzer.AnalyzeMLCapabilities(r.Context(), ontologyID)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to analyze ontology: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(capabilities)
}

// AutoTrainRequest represents a request to automatically train models
type AutoTrainRequest struct {
	EnableRegression     bool    `json:"enable_regression"`
	EnableClassification bool    `json:"enable_classification"`
	EnableMonitoring     bool    `json:"enable_monitoring"`
	MinConfidence        float64 `json:"min_confidence"`
	ForceAll             bool    `json:"force_all"`
	MaxModels            int     `json:"max_models"`
}

// handleAutoTrain automatically trains models based on ontology analysis
func (s *Server) handleAutoTrain(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ontologyID := vars["id"]

	if ontologyID == "" {
		http.Error(w, "ontology ID is required", http.StatusBadRequest)
		return
	}

	// Parse request
	var req AutoTrainRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Use defaults if parsing fails
		req = AutoTrainRequest{
			EnableRegression:     true,
			EnableClassification: true,
			EnableMonitoring:     true,
			MinConfidence:        0.6,
			ForceAll:             false,
			MaxModels:            10,
		}
	}

	// Convert to AutoTrainOptions
	options := &ml.AutoTrainOptions{
		EnableRegression:     req.EnableRegression,
		EnableClassification: req.EnableClassification,
		EnableMonitoring:     req.EnableMonitoring,
		MinConfidence:        req.MinConfidence,
		ForceAll:             req.ForceAll,
		MaxModels:            req.MaxModels,
	}

	// Create auto-trainer
	autoTrainer := ml.NewAutoTrainer(s.persistence, s.tdb2Backend)

	// Train models
	result, err := autoTrainer.TrainFromOntology(r.Context(), ontologyID, options)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to auto-train models: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(result)
}

// TrainForGoalRequest represents a request to train based on a natural language goal
type TrainForGoalRequest struct {
	Goal string `json:"goal"`
}

// handleTrainForGoal trains models based on a natural language goal
func (s *Server) handleTrainForGoal(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ontologyID := vars["id"]

	if ontologyID == "" {
		http.Error(w, "ontology ID is required", http.StatusBadRequest)
		return
	}

	// Parse request
	var req TrainForGoalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Goal == "" {
		http.Error(w, "goal is required", http.StatusBadRequest)
		return
	}

	// Create auto-trainer
	autoTrainer := ml.NewAutoTrainer(s.persistence, s.tdb2Backend)

	// Train models based on goal
	result, err := autoTrainer.TrainForGoal(r.Context(), ontologyID, req.Goal)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to train for goal: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(result)
}

// handleGetMLSuggestions returns detailed ML suggestions with reasoning
func (s *Server) handleGetMLSuggestions(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ontologyID := vars["id"]

	if ontologyID == "" {
		http.Error(w, "ontology ID is required", http.StatusBadRequest)
		return
	}

	// Create auto-trainer
	autoTrainer := ml.NewAutoTrainer(s.persistence, s.tdb2Backend)

	// Get suggestions
	suggestions, err := autoTrainer.GetAutoTrainingSuggestions(r.Context(), ontologyID)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get suggestions: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"ontology_id": ontologyID,
		"suggestions": suggestions,
		"summary":     suggestions.Summary,
	})
}
