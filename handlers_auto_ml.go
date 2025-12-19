package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/ML"
	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/Storage"
	"github.com/google/uuid"
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

// AutoTrainWithDataRequest represents a request to train models with uploaded data
type AutoTrainWithDataRequest struct {
	OntologyID           string              `json:"ontology_id"`
	DataSource           ml.DataSourceConfig `json:"data_source"`
	EnableRegression     bool                `json:"enable_regression"`
	EnableClassification bool                `json:"enable_classification"`
	EnableMonitoring     bool                `json:"enable_monitoring"`
	MinConfidence        float64             `json:"min_confidence"`
	MaxModels            int                 `json:"max_models"`
}

// SimpleAutoTrainRequest is a simplified training request matching frontend format
type SimpleAutoTrainRequest struct {
	Data         string  `json:"data"`          // CSV data as string
	TargetColumn string  `json:"target_column"` // Target column name
	ModelName    string  `json:"model_name"`    // Optional model name
	Algorithm    string  `json:"algorithm"`     // Optional algorithm (ignored for now)
	TestSplit    float64 `json:"test_split"`    // Optional test split ratio
}

// handleAutoTrainWithData trains models using uploaded data (simplified version for frontend)
func (s *Server) handleAutoTrainWithData(w http.ResponseWriter, r *http.Request) {
	// Try to parse as simple request first (from frontend)
	var simpleReq SimpleAutoTrainRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&simpleReq); err == nil && simpleReq.Data != "" {
		// Handle simple training request
		s.handleSimpleAutoTrain(w, r, simpleReq)
		return
	}

	// If Data field is empty, treat as error
	http.Error(w, "invalid request: missing 'data' field for simple training", http.StatusBadRequest)
}

// handleSimpleAutoTrain handles simplified training requests from the frontend
func (s *Server) handleSimpleAutoTrain(w http.ResponseWriter, r *http.Request, req SimpleAutoTrainRequest) {
	// Validate request
	if req.Data == "" {
		http.Error(w, "data is required", http.StatusBadRequest)
		return
	}
	if req.TargetColumn == "" {
		http.Error(w, "target_column is required", http.StatusBadRequest)
		return
	}
	if req.ModelName == "" {
		req.ModelName = "Trained Model"
	}
	if req.TestSplit == 0 {
		req.TestSplit = 0.2
	}

	// Parse CSV data into [][]string format
	lines := []string{}
	for _, line := range strings.Split(req.Data, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			lines = append(lines, trimmed)
		}
	}

	if len(lines) < 2 {
		http.Error(w, "CSV data must have at least a header and one data row", http.StatusBadRequest)
		return
	}

	trainData := [][]string{}
	for _, line := range lines {
		row := strings.Split(line, ",")
		for i := range row {
			row[i] = strings.TrimSpace(row[i])
		}
		trainData = append(trainData, row)
	}

	// Determine if classification or regression by checking target column values
	header := trainData[0]
	targetIdx := -1
	for i, col := range header {
		if col == req.TargetColumn {
			targetIdx = i
			break
		}
	}

	if targetIdx == -1 {
		http.Error(w, fmt.Sprintf("target column '%s' not found in CSV data", req.TargetColumn), http.StatusBadRequest)
		return
	}

	// Sample target values to determine type
	modelType := "classification"
	if len(trainData) > 1 {
		// Try to parse first value as float
		firstValue := trainData[1][targetIdx]
		if _, err := strconv.ParseFloat(firstValue, 64); err == nil {
			// Check if all values are numeric (regression)
			allNumeric := true
			uniqueValues := make(map[string]bool)
			for i := 1; i < len(trainData) && i < 10; i++ {
				val := trainData[i][targetIdx]
				uniqueValues[val] = true
				if _, err := strconv.ParseFloat(val, 64); err != nil {
					allNumeric = false
					break
				}
			}
			// Only treat as regression if numeric AND has many unique values (not just 0/1)
			if allNumeric && len(uniqueValues) > 5 {
				modelType = "regression"
			}
		}
	}

	fmt.Printf("🔍 Model type detection: %s for target column '%s' (rows: %d)\n",
		modelType, req.TargetColumn, len(trainData)-1)

	// Create training config
	config := ml.DefaultTrainingConfig()
	config.TrainTestSplit = req.TestSplit

	// Create trainer
	trainer := ml.NewTrainer(config)

	var result *ml.TrainingResult
	var featureNames []string

	// Train based on detected type
	if modelType == "regression" {
		fmt.Printf("🎯 Training REGRESSION model...\n")
		X, y, fnames, err := ml.PrepareRegressionDataFromCSV(trainData, req.TargetColumn)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to prepare regression data: %v", err), http.StatusBadRequest)
			return
		}
		featureNames = fnames

		result, err = trainer.TrainRegression(X, y, featureNames)
		if err != nil {
			http.Error(w, fmt.Sprintf("regression training failed: %v", err), http.StatusInternalServerError)
			return
		}
		fmt.Printf("✅ Regression training complete: R²=%.4f, MAE=%.4f\n",
			result.ValidateMetricsReg.R2Score, result.ValidateMetricsReg.MAE)
	} else {
		fmt.Printf("🎯 Training CLASSIFICATION model...\n")
		X, y, fnames, err := ml.PrepareDataFromCSV(trainData, req.TargetColumn)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to prepare classification data: %v", err), http.StatusBadRequest)
			return
		}
		featureNames = fnames

		result, err = trainer.Train(X, y, featureNames)
		if err != nil {
			http.Error(w, fmt.Sprintf("classification training failed: %v", err), http.StatusInternalServerError)
			return
		}
		fmt.Printf("✅ Classification training complete: Accuracy=%.4f, F1=%.4f\n",
			result.ValidateMetrics.Accuracy, result.ValidateMetrics.MacroF1)
	}

	// Generate model ID and save artifact
	modelID := uuid.New().String()
	artifactDir := "./data/models"
	if err := os.MkdirAll(artifactDir, 0755); err != nil {
		http.Error(w, fmt.Sprintf("failed to create model directory: %v", err), http.StatusInternalServerError)
		return
	}

	artifactPath := filepath.Join(artifactDir, fmt.Sprintf("%s.json", modelID))
	if err := result.Model.Save(artifactPath); err != nil {
		http.Error(w, fmt.Sprintf("failed to save model artifact: %v", err), http.StatusInternalServerError)
		return
	}

	// Get model size
	fileInfo, err := os.Stat(artifactPath)
	var modelSize int64 = 0
	if err == nil {
		modelSize = fileInfo.Size()
	}

	// Serialize metadata
	hyperparamsJSON, _ := json.Marshal(map[string]interface{}{
		"max_depth":         config.MaxDepth,
		"min_samples_split": config.MinSamplesSplit,
		"min_samples_leaf":  config.MinSamplesLeaf,
		"train_test_split":  config.TrainTestSplit,
	})
	featureColumnsJSON, _ := json.Marshal(featureNames)

	var classLabelsJSON, confusionMatrixJSON []byte
	if modelType == "classification" {
		classLabelsJSON, _ = json.Marshal(result.Model.Classes)
		confusionMatrixJSON, _ = json.Marshal(result.ValidateMetrics.ConfusionMatrix)
	}
	featureImportanceJSON, _ := json.Marshal(result.FeatureImportance)

	// Get metrics
	var trainAccuracy, validateAccuracy, precision, recall, f1 float64
	if modelType == "classification" {
		trainAccuracy = result.TrainMetrics.Accuracy
		validateAccuracy = result.ValidateMetrics.Accuracy
		precision = result.ValidateMetrics.MacroPrecision
		recall = result.ValidateMetrics.MacroRecall
		f1 = result.ValidateMetrics.MacroF1
	} else {
		validateAccuracy = result.ValidateMetricsReg.R2Score
	}

	// Create model record (without ontology_id)
	model := &storage.ClassifierModel{
		ID:                modelID,
		OntologyID:        "", // No ontology required for simple training
		Name:              req.ModelName,
		TargetClass:       req.TargetColumn,
		Algorithm:         "decision_tree",
		Hyperparameters:   string(hyperparamsJSON),
		FeatureColumns:    string(featureColumnsJSON),
		ClassLabels:       string(classLabelsJSON),
		TrainAccuracy:     trainAccuracy,
		ValidateAccuracy:  validateAccuracy,
		PrecisionScore:    precision,
		RecallScore:       recall,
		F1Score:           f1,
		ConfusionMatrix:   string(confusionMatrixJSON),
		ModelArtifactPath: artifactPath,
		ModelSizeBytes:    modelSize,
		TrainingRows:      result.TrainingRows,
		ValidationRows:    result.ValidationRows,
		FeatureImportance: string(featureImportanceJSON),
		IsActive:          true,
	}

	if err := s.persistence.CreateClassifierModel(r.Context(), model); err != nil {
		http.Error(w, fmt.Sprintf("failed to save model metadata: %v", err), http.StatusInternalServerError)
		return
	}

	// Build response matching frontend expectations
	response := map[string]interface{}{
		"message":    "Model trained successfully",
		"model_id":   modelID,
		"model_name": req.ModelName,
		"model_type": modelType,
	}

	if modelType == "classification" {
		response["accuracy"] = validateAccuracy
		response["precision"] = precision
		response["recall"] = recall
		response["f1_score"] = f1
	} else {
		// For regression, return R² and error metrics
		response["r2_score"] = validateAccuracy
		response["mae"] = result.ValidateMetricsReg.MAE
		response["rmse"] = result.ValidateMetricsReg.RMSE
		response["mse"] = result.ValidateMetricsReg.MSE
		// Note: Don't return "accuracy" for regression as it's misleading
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}
