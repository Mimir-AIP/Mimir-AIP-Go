package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/ML"
	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/Storage"
	"github.com/Mimir-AIP/Mimir-AIP-Go/utils"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// TrainModelRequest represents a model training request
type TrainModelRequest struct {
	OntologyID     string             `json:"ontology_id"`
	ModelName      string             `json:"model_name"`
	ModelType      string             `json:"model_type"`      // "classification" or "regression"
	Algorithm      string             `json:"algorithm"`       // "decision_tree" or "random_forest"
	TargetColumn   string             `json:"target_column"`
	FeatureColumns []string           `json:"feature_columns,omitempty"` // Optional, auto-detect if empty
	TrainData      [][]string         `json:"train_data"`                // CSV-like data including header
	Config         *ml.TrainingConfig `json:"config,omitempty"`          // Optional training config
}

// PredictRequest represents a prediction request
type PredictRequest struct {
	ModelID   string             `json:"model_id"`
	InputData map[string]float64 `json:"input_data"` // Feature name -> value
}

// handleTrainModel handles POST /api/v1/models/train
func (s *Server) handleTrainModel(w http.ResponseWriter, r *http.Request) {
	var req TrainModelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequestResponse(w, fmt.Sprintf("Invalid request body: %v", err))
		return
	}

	// Validate request
	if req.OntologyID == "" {
		writeBadRequestResponse(w, "ontology_id is required")
		return
	}
	if req.ModelName == "" {
		writeBadRequestResponse(w, "model_name is required")
		return
	}
	if req.TargetColumn == "" {
		writeBadRequestResponse(w, "target_column is required")
		return
	}
	if len(req.TrainData) < 2 {
		writeBadRequestResponse(w, "train_data must have at least header and one row")
		return
	}

	// Verify ontology exists
	ctx := r.Context()
	_, err := s.persistence.GetOntology(ctx, req.OntologyID)
	if err != nil {
		writeErrorResponse(w, http.StatusNotFound, fmt.Sprintf("Ontology not found: %v", err))
		return
	}

	// Default to classification if not specified
	modelType := req.ModelType
	if modelType == "" {
		modelType = "classification"
	}
	if modelType != "classification" && modelType != "regression" {
		writeBadRequestResponse(w, "model_type must be 'classification' or 'regression'")
		return
	}

	// Use provided config or default
	config := req.Config
	if config == nil {
		config = ml.DefaultTrainingConfig()
	}

	// Default to decision tree if not specified
	algorithm := req.Algorithm
	if algorithm == "" {
		algorithm = "decision_tree"
	}
	if algorithm != "decision_tree" && algorithm != "random_forest" {
		writeBadRequestResponse(w, "algorithm must be 'decision_tree' or 'random_forest'")
		return
	}

	// Create trainer
	trainer := ml.NewTrainer(config)

	// Variables to hold result data
	var featureNames []string

	// Train model based on type and algorithm
	var result *ml.TrainingResult
	if modelType == "regression" {
		// Prepare regression data
		X, y, fnames, err := ml.PrepareRegressionDataFromCSV(req.TrainData, req.TargetColumn)
		if err != nil {
			writeBadRequestResponse(w, fmt.Sprintf("Failed to prepare regression data: %v", err))
			return
		}
		featureNames = fnames

		// Train regressor based on algorithm
		if algorithm == "random_forest" {
			result, err = trainer.TrainRandomForestRegression(X, y, featureNames)
		} else {
			result, err = trainer.TrainRegression(X, y, featureNames)
		}
		if err != nil {
			writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Regression training failed: %v", err))
			return
		}
	} else {
		// Prepare classification data
		X, y, fnames, err := ml.PrepareDataFromCSV(req.TrainData, req.TargetColumn)
		if err != nil {
			writeBadRequestResponse(w, fmt.Sprintf("Failed to prepare classification data: %v", err))
			return
		}
		featureNames = fnames

		// Train classifier based on algorithm
		if algorithm == "random_forest" {
			result, err = trainer.TrainRandomForest(X, y, featureNames)
		} else {
			result, err = trainer.Train(X, y, featureNames)
		}
		if err != nil {
			writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Classification training failed: %v", err))
			return
		}
	}

	// Generate model ID and save artifact
	modelID := uuid.New().String()
	artifactDir := "./data/models"
	if err := os.MkdirAll(artifactDir, 0755); err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create model directory: %v", err))
		return
	}

	artifactPath := filepath.Join(artifactDir, fmt.Sprintf("%s.json", modelID))
	
	// Save model based on algorithm type
	if algorithm == "random_forest" {
		if err := result.ModelRF.Save(artifactPath); err != nil {
			writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to save model artifact: %v", err))
			return
		}
	} else {
		if err := result.Model.Save(artifactPath); err != nil {
			writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to save model artifact: %v", err))
			return
		}
	}

	// Get model size
	fileInfo, err := os.Stat(artifactPath)
	var modelSize int64 = 0
	if err == nil {
		modelSize = fileInfo.Size()
	}

	// Serialize metadata as JSON
	hyperparamsJSON, _ := json.Marshal(map[string]interface{}{
		"max_depth":         config.MaxDepth,
		"min_samples_split": config.MinSamplesSplit,
		"min_samples_leaf":  config.MinSamplesLeaf,
		"train_test_split":  config.TrainTestSplit,
		"num_trees":         config.NumTrees,
		"algorithm":         algorithm,
	})
	featureColumnsJSON, _ := json.Marshal(featureNames)

	// Prepare class labels and confusion matrix (classification only)
	var classLabelsJSON, confusionMatrixJSON []byte
	if modelType == "classification" {
		if algorithm == "random_forest" {
			classLabelsJSON, _ = json.Marshal(result.ModelRF.Classes)
		} else {
			classLabelsJSON, _ = json.Marshal(result.Model.Classes)
		}
		confusionMatrixJSON, _ = json.Marshal(result.ValidateMetrics.ConfusionMatrix)
	}

	featureImportanceJSON, _ := json.Marshal(result.FeatureImportance)

	// Get metrics based on model type
	var trainAccuracy, validateAccuracy, precision, recall, f1 float64
	var mae, mse, rmse, r2Score float64

	if modelType == "classification" {
		trainAccuracy = result.TrainMetrics.Accuracy
		validateAccuracy = result.ValidateMetrics.Accuracy
		precision = result.ValidateMetrics.MacroPrecision
		recall = result.ValidateMetrics.MacroRecall
		f1 = result.ValidateMetrics.MacroF1
	} else {
		// For regression, use R² as "accuracy"
		r2Score = result.ValidateMetricsReg.R2Score
		validateAccuracy = r2Score
		mae = result.ValidateMetricsReg.MAE
		mse = result.ValidateMetricsReg.MSE
		rmse = result.ValidateMetricsReg.RMSE
	}

	// Create model record in database
	model := &storage.ClassifierModel{
		ID:                modelID,
		OntologyID:        req.OntologyID,
		Name:              req.ModelName,
		TargetClass:       req.TargetColumn,
		Algorithm:         algorithm,
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

	if err := s.persistence.CreateClassifierModel(ctx, model); err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to save model metadata: %v", err))
		return
	}

	// Create training run record
	var metricsJSON []byte
	if modelType == "classification" {
		metricsJSON, _ = json.Marshal(map[string]interface{}{
			"train_metrics":    result.TrainMetrics,
			"validate_metrics": result.ValidateMetrics,
		})
	} else {
		metricsJSON, _ = json.Marshal(map[string]interface{}{
			"train_metrics_reg":    result.TrainMetricsReg,
			"validate_metrics_reg": result.ValidateMetricsReg,
		})
	}
	configJSON, _ := json.Marshal(config)

	trainingDurationMs := result.TrainingDuration.Milliseconds()
	_, err = s.persistence.CreateTrainingRun(
		ctx,
		modelID,
		result.TrainingRows+result.ValidationRows,
		int(trainingDurationMs),
		trainAccuracy,
		validateAccuracy,
		string(metricsJSON),
		string(configJSON),
		"completed",
		"",
	)
	if err != nil {
		utils.GetLogger().Warn(fmt.Sprintf("Failed to create training run record: %v", err))
	}

	// Build response based on model type
	response := map[string]interface{}{
		"message":              "Model trained successfully",
		"model_id":             modelID,
		"model_name":           req.ModelName,
		"ontology_id":          req.OntologyID,
		"model_type":           modelType,
		"algorithm":            algorithm,
		"training_rows":        result.TrainingRows,
		"validation_rows":      result.ValidationRows,
		"training_duration_ms": trainingDurationMs,
		"feature_importance":   result.FeatureImportance,
		"model_info":           result.ModelInfo,
	}

	if modelType == "classification" {
		response["train_accuracy"] = result.TrainMetrics.Accuracy
		response["validate_accuracy"] = result.ValidateMetrics.Accuracy
		response["precision"] = result.ValidateMetrics.MacroPrecision
		response["recall"] = result.ValidateMetrics.MacroRecall
		response["f1_score"] = result.ValidateMetrics.MacroF1
		response["confusion_matrix"] = result.ValidateMetrics.ConfusionMatrix
	} else {
		response["r2_score"] = r2Score
		response["mae"] = mae
		response["mse"] = mse
		response["rmse"] = rmse
		response["mape"] = result.ValidateMetricsReg.MAPE
		response["validate_metrics"] = result.ValidateMetricsReg
	}

	// Return response
	writeJSONResponse(w, http.StatusCreated, response)
}

// handlePredict handles POST /api/v1/models/{id}/predict
func (s *Server) handlePredict(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	modelID := vars["id"]

	var req PredictRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequestResponse(w, fmt.Sprintf("Invalid request body: %v", err))
		return
	}

	ctx := r.Context()

	// Get model metadata from database
	modelMeta, err := s.persistence.GetClassifierModel(ctx, modelID)
	if err != nil {
		writeErrorResponse(w, http.StatusNotFound, fmt.Sprintf("Model not found: %v", err))
		return
	}

	if !modelMeta.IsActive {
		writeErrorResponse(w, http.StatusBadRequest, "Model is not active")
		return
	}

	// Load model artifact based on algorithm
	var featureNames []string
	var modelType string
	
	// Try loading as Random Forest first
	rf := &ml.RandomForestClassifier{}
	err = rf.Load(modelMeta.ModelArtifactPath)
	if err == nil && len(rf.Trees) > 0 {
		// It's a Random Forest
		featureNames = rf.FeatureNames
		modelType = rf.ModelType
	} else {
		// Try loading as Decision Tree
		classifier := &ml.DecisionTreeClassifier{}
		if err := classifier.Load(modelMeta.ModelArtifactPath); err != nil {
			writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to load model: %v", err))
			return
		}
		featureNames = classifier.FeatureNames
		modelType = classifier.ModelType
		
		// Prepare input features and make prediction for Decision Tree
		features := make([]float64, len(featureNames))
		for i, featureName := range featureNames {
			val, exists := req.InputData[featureName]
			if !exists {
				writeBadRequestResponse(w, fmt.Sprintf("Missing feature: %s", featureName))
				return
			}
			features[i] = val
		}

		// Make prediction based on model type
		if modelType == "regression" {
			// Regression prediction with confidence interval
			value, lower, upper, err := classifier.PredictRegressionWithInterval(features)
			if err != nil {
				writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Prediction failed: %v", err))
				return
			}

			// Return regression prediction
			writeJSONResponse(w, http.StatusOK, map[string]interface{}{
				"model_id":            modelID,
				"algorithm":           "decision_tree",
				"model_type":          "regression",
				"predicted_value":     value,
				"confidence_lower":    lower,
				"confidence_upper":    upper,
				"confidence_interval": fmt.Sprintf("[%.4f, %.4f]", lower, upper),
				"input_features":      req.InputData,
			})
			return
		}

		// Classification prediction
		predictedClass, _, err := classifier.Predict(features)
		if err != nil {
			writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Prediction failed: %v", err))
			return
		}

		// Get class probabilities
		proba, err := classifier.PredictProba(features)
		if err != nil {
			writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get probabilities: %v", err))
			return
		}

		confidence := proba[predictedClass]

		// Check if low confidence (potential anomaly)
		isAnomaly := false
		anomalyReason := ""
		confidenceThreshold := 0.7

		if confidence < confidenceThreshold {
			isAnomaly = true
			anomalyReason = fmt.Sprintf("Low confidence prediction: %.2f < %.2f", confidence, confidenceThreshold)
		}

		// Save prediction to database
		inputDataJSON, _ := json.Marshal(req.InputData)
		err = s.persistence.CreatePrediction(
			ctx,
			modelID,
			string(inputDataJSON),
			predictedClass,
			confidence,
			"", // actual_class unknown
			false,
			isAnomaly,
			anomalyReason,
		)
		if err != nil {
			utils.GetLogger().Warn(fmt.Sprintf("Failed to save prediction: %v", err))
		}

		// If anomaly detected, create anomaly record
		if isAnomaly {
			_, err = s.persistence.CreateAnomaly(
				ctx,
				modelID,
				nil,
				"low_confidence",
				string(inputDataJSON),
				&confidence,
				"",
				"medium",
				"open",
			)
			if err != nil {
				utils.GetLogger().Warn(fmt.Sprintf("Failed to create anomaly record: %v", err))
			}
		}

		// Return response
		writeJSONResponse(w, http.StatusOK, map[string]interface{}{
			"model_id":        modelID,
			"algorithm":       "decision_tree",
			"predicted_class": predictedClass,
			"confidence":      confidence,
			"probabilities":   proba,
			"is_anomaly":      isAnomaly,
			"anomaly_reason":  anomalyReason,
			"input_features":  req.InputData,
		})
		return
	}

	// Random Forest prediction
	features := make([]float64, len(featureNames))
	for i, featureName := range featureNames {
		val, exists := req.InputData[featureName]
		if !exists {
			writeBadRequestResponse(w, fmt.Sprintf("Missing feature: %s", featureName))
			return
		}
		features[i] = val
	}

	// Make prediction based on model type
	if modelType == "regression" {
		// Regression prediction with confidence interval
		value, lower, upper, err := rf.PredictRegressionWithInterval(features)
		if err != nil {
			writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Prediction failed: %v", err))
			return
		}

		// Return regression prediction
		writeJSONResponse(w, http.StatusOK, map[string]interface{}{
			"model_id":            modelID,
			"algorithm":           "random_forest",
			"model_type":          "regression",
			"predicted_value":     value,
			"confidence_lower":    lower,
			"confidence_upper":    upper,
			"confidence_interval": fmt.Sprintf("[%.4f, %.4f]", lower, upper),
			"input_features":      req.InputData,
		})
		return
	}

	// Classification prediction
	predictedClass, confidence, err := rf.Predict(features)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Prediction failed: %v", err))
		return
	}

	// Get class probabilities
	proba, err := rf.PredictProba(features)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get probabilities: %v", err))
		return
	}

	// Check if low confidence (potential anomaly)
	isAnomaly := false
	anomalyReason := ""
	confidenceThreshold := 0.7

	if confidence < confidenceThreshold {
		isAnomaly = true
		anomalyReason = fmt.Sprintf("Low confidence prediction: %.2f < %.2f", confidence, confidenceThreshold)
	}

	// Save prediction to database
	inputDataJSON, _ := json.Marshal(req.InputData)
	err = s.persistence.CreatePrediction(
		ctx,
		modelID,
		string(inputDataJSON),
		predictedClass,
		confidence,
		"", // actual_class unknown
		false,
		isAnomaly,
		anomalyReason,
	)
	if err != nil {
		utils.GetLogger().Warn(fmt.Sprintf("Failed to save prediction: %v", err))
	}

	// If anomaly detected, create anomaly record
	if isAnomaly {
		_, err = s.persistence.CreateAnomaly(
			ctx,
			modelID,
			nil,
			"low_confidence",
			string(inputDataJSON),
			&confidence,
			"",
			"medium",
			"open",
		)
		if err != nil {
			utils.GetLogger().Warn(fmt.Sprintf("Failed to create anomaly record: %v", err))
		}
	}

	// Return response
	writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"model_id":        modelID,
		"algorithm":       "random_forest",
		"predicted_class": predictedClass,
		"confidence":      confidence,
		"probabilities":   proba,
		"is_anomaly":      isAnomaly,
		"anomaly_reason":  anomalyReason,
		"input_features":  req.InputData,
	})
}

// handleListModels handles GET /api/v1/models
func (s *Server) handleListModels(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get query parameters
	ontologyID := r.URL.Query().Get("ontology_id")
	activeOnly := r.URL.Query().Get("active_only") == "true"

	models, err := s.persistence.ListClassifierModels(ctx, ontologyID, activeOnly)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to list models: %v", err))
		return
	}

	writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"models": models,
		"count":  len(models),
	})
}

// handleGetModel handles GET /api/v1/models/{id}
func (s *Server) handleGetModel(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	modelID := vars["id"]

	ctx := r.Context()
	model, err := s.persistence.GetClassifierModel(ctx, modelID)
	if err != nil {
		writeErrorResponse(w, http.StatusNotFound, fmt.Sprintf("Model not found: %v", err))
		return
	}

	writeJSONResponse(w, http.StatusOK, model)
}

// handleDeleteModel handles DELETE /api/v1/models/{id}
func (s *Server) handleDeleteModel(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	modelID := vars["id"]

	ctx := r.Context()

	// Get model to find artifact path
	model, err := s.persistence.GetClassifierModel(ctx, modelID)
	if err != nil {
		writeErrorResponse(w, http.StatusNotFound, fmt.Sprintf("Model not found: %v", err))
		return
	}

	// Delete model artifact file
	if err := os.Remove(model.ModelArtifactPath); err != nil {
		utils.GetLogger().Warn(fmt.Sprintf("Failed to delete model artifact: %v", err))
	}

	// Delete model from database
	if err := s.persistence.DeleteClassifierModel(ctx, modelID); err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to delete model: %v", err))
		return
	}

	writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"message":  "Model deleted successfully",
		"model_id": modelID,
	})
}

// handleUpdateModelStatus handles PATCH /api/v1/models/{id}/status
func (s *Server) handleUpdateModelStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	modelID := vars["id"]

	var req struct {
		IsActive bool `json:"is_active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequestResponse(w, fmt.Sprintf("Invalid request body: %v", err))
		return
	}

	ctx := r.Context()
	if err := s.persistence.UpdateClassifierModelStatus(ctx, modelID, req.IsActive); err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to update model status: %v", err))
		return
	}

	writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"message":   "Model status updated",
		"model_id":  modelID,
		"is_active": req.IsActive,
	})
}

// handleListAnomalies handles GET /api/v1/anomalies
func (s *Server) handleListAnomalies(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get query parameters
	modelID := r.URL.Query().Get("model_id")
	status := r.URL.Query().Get("status")
	severity := r.URL.Query().Get("severity")
	limit := 100
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	anomalies, err := s.persistence.ListAnomalies(ctx, modelID, status, severity, limit)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to list anomalies: %v", err))
		return
	}

	writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"anomalies": anomalies,
		"count":     len(anomalies),
	})
}

// handleUpdateAnomalyStatus handles PATCH /api/v1/anomalies/{id}
func (s *Server) handleUpdateAnomalyStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	anomalyIDStr := vars["id"]

	anomalyID, err := strconv.ParseInt(anomalyIDStr, 10, 64)
	if err != nil {
		writeBadRequestResponse(w, "Invalid anomaly ID")
		return
	}

	var req struct {
		Status          string `json:"status"`
		AssignedTo      string `json:"assigned_to"`
		ResolutionNotes string `json:"resolution_notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequestResponse(w, fmt.Sprintf("Invalid request body: %v", err))
		return
	}

	ctx := r.Context()
	if err := s.persistence.UpdateAnomalyStatus(ctx, anomalyID, req.Status, req.AssignedTo, req.ResolutionNotes); err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to update anomaly: %v", err))
		return
	}

	writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"message":    "Anomaly updated",
		"anomaly_id": anomalyID,
		"status":     req.Status,
	})
}
