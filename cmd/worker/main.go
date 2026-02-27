package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"plugin"
	"sort"
	"strconv"
	"time"

	"github.com/mimir-aip/mimir-aip-go/pkg/mlmodel/training"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	pipelinepkg "github.com/mimir-aip/mimir-aip-go/pkg/pipeline"
	"github.com/mimir-aip/mimir-aip-go/pkg/plugins"
)

func getOrchestratorURL() string {
	if url := os.Getenv("ORCHESTRATOR_URL"); url != "" {
		return url
	}
	return "http://orchestrator:8080"
}

func main() {
	// Get work task information from environment
	taskID := os.Getenv("WORKTASK_ID")
	taskType := os.Getenv("WORKTASK_TYPE")

	if taskID == "" || taskType == "" {
		log.Fatal("WORKTASK_ID and WORKTASK_TYPE must be set")
	}

	log.Printf("Worker starting for task %s (type: %s)", taskID, taskType)

	// Get orchestrator URL
	orchestratorURL := getOrchestratorURL()

	// Get work task details from orchestrator API
	task, err := getWorkTaskFromAPI(orchestratorURL, taskID)
	if err != nil {
		log.Fatalf("Failed to get work task details: %v", err)
	}

	// Update work task status to executing
	if err := updateWorkTaskStatus(orchestratorURL, taskID, models.WorkTaskStatusExecuting, ""); err != nil {
		log.Printf("Warning: Failed to update work task status to executing: %v", err)
	}

	// Execute the work task
	result, err := executeWorkTask(task)
	if err != nil {
		log.Printf("Work task execution failed: %v", err)
		// Report failure
		reportWorkTaskCompletion(orchestratorURL, taskID, models.WorkTaskStatusFailed, "", err.Error())
		os.Exit(1)
	}

	// Report success
	log.Printf("Work task %s completed successfully", taskID)
	reportWorkTaskCompletion(orchestratorURL, taskID, models.WorkTaskStatusCompleted, result.OutputLocation, "")
}

// executeWorkTask executes the work task based on its type
func executeWorkTask(task *models.WorkTask) (*models.WorkTaskResult, error) {
	log.Printf("Executing work task of type: %s", task.Type)

	switch task.Type {
	case models.WorkTaskTypePipelineExecution:
		return executePipeline(task)
	case models.WorkTaskTypeMLTraining:
		return executeMLTraining(task)
	case models.WorkTaskTypeMLInference:
		return executeMLInference(task)
	case models.WorkTaskTypeDigitalTwinUpdate:
		return executeDigitalTwinUpdate(task)
	default:
		return nil, fmt.Errorf("unknown work task type: %s", task.Type)
	}
}

// executePipeline executes a pipeline work task
func executePipeline(task *models.WorkTask) (*models.WorkTaskResult, error) {
	log.Printf("Executing pipeline: %s", task.TaskSpec.PipelineID)

	// Get orchestrator URL
	orchestratorURL := getOrchestratorURL()

	// Fetch pipeline definition from orchestrator
	pipelineURL := fmt.Sprintf("%s/api/pipelines/%s", orchestratorURL, task.TaskSpec.PipelineID)
	resp, err := http.Get(pipelineURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch pipeline: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch pipeline: status %d", resp.StatusCode)
	}

	var pipeline models.Pipeline
	if err := json.NewDecoder(resp.Body).Decode(&pipeline); err != nil {
		return nil, fmt.Errorf("failed to decode pipeline: %w", err)
	}

	// Execute the pipeline locally
	log.Printf("Executing pipeline %s (%s) with %d steps", pipeline.Name, pipeline.ID, len(pipeline.Steps))

	// Create execution context
	context := models.NewPipelineContext(10485760) // 10MB max size

	// Add parameters from task spec to context
	if task.TaskSpec.Parameters != nil {
		for key, value := range task.TaskSpec.Parameters {
			context.SetStepData("_parameters", key, value)
		}
	}

	// Initialize plugin system via HTTP registry
	// Workers fetch plugins from the orchestrator's plugin registry service
	// Use orchestratorURL that was already set above

	pluginCacheDir := "/tmp/plugins"
	pluginClient := plugins.NewClient(orchestratorURL, pluginCacheDir)

	// Initialize plugin registry with built-in plugins
	pluginRegistry := make(map[string]pipelinepkg.Plugin)
	pluginRegistry["default"] = pipelinepkg.NewDefaultPlugin()
	pluginRegistry["builtin"] = pipelinepkg.NewDefaultPlugin()

	// Discover and load custom plugins needed for this pipeline
	// We'll load them on-demand as we encounter them in the pipeline steps
	uniquePlugins := make(map[string]bool)
	for _, step := range pipeline.Steps {
		if step.Plugin != "default" && step.Plugin != "builtin" {
			uniquePlugins[step.Plugin] = true
		}
	}

	// Download and load each unique plugin
	for pluginName := range uniquePlugins {
		log.Printf("Compiling plugin: %s", pluginName)
		pluginPath, err := pluginClient.CompilePlugin(pluginName)
		if err != nil {
			log.Printf("Warning: Failed to compile plugin %s: %v", pluginName, err)
			continue
		}

		log.Printf("Loading plugin from: %s", pluginPath)

		// Load the plugin using Go's plugin system
		p, err := plugin.Open(pluginPath)
		if err != nil {
			log.Printf("Warning: Failed to open plugin %s: %v", pluginName, err)
			continue
		}

		// Look for the Plugin symbol
		symPlugin, err := p.Lookup("Plugin")
		if err != nil {
			log.Printf("Warning: Plugin %s does not export 'Plugin' symbol: %v", pluginName, err)
			continue
		}

		// Assert that it implements the pipeline.Plugin interface
		pluginInstance, ok := symPlugin.(pipelinepkg.Plugin)
		if !ok {
			log.Printf("Warning: Plugin %s does not implement pipeline.Plugin interface", pluginName)
			continue
		}

		// Register in registry
		pluginRegistry[pluginName] = pluginInstance
		log.Printf("Loaded custom plugin: %s", pluginName)
	}

	// Execute pipeline steps using the plugin system
	startTime := time.Now()
	currentStepIndex := 0
	stepsExecuted := 0

	for currentStepIndex < len(pipeline.Steps) {
		step := pipeline.Steps[currentStepIndex]
		stepsExecuted++

		log.Printf("  Step %d: %s (%s.%s)", currentStepIndex+1, step.Name, step.Plugin, step.Action)

		// Get plugin
		pluginInstance, ok := pluginRegistry[step.Plugin]
		if !ok {
			return nil, fmt.Errorf("unknown plugin: %s", step.Plugin)
		}

		// Execute step
		result, err := pluginInstance.Execute(step.Action, step.Parameters, context)
		if err != nil {
			return nil, fmt.Errorf("step %s failed: %w", step.Name, err)
		}

		// Store step results in context
		for key, value := range result {
			context.SetStepData(step.Name, key, value)
		}

		// Store output values from step configuration
		if step.Output != nil {
			for outputKey, outputTemplate := range step.Output {
				// Resolve template
				if dp, ok := pluginInstance.(*pipelinepkg.DefaultPlugin); ok {
					resolvedValue := dp.ResolveTemplates(outputTemplate, context)
					context.SetStepData(step.Name, outputKey, resolvedValue)
					log.Printf("    Output: %s = %v", outputKey, resolvedValue)
				}
			}
		}

		// Check for goto action
		if gotoTarget, ok := result["goto"].(string); ok {
			// Find target step index
			targetIndex := -1
			for i, s := range pipeline.Steps {
				if s.Name == gotoTarget {
					targetIndex = i
					break
				}
			}

			if targetIndex == -1 {
				return nil, fmt.Errorf("goto target not found: %s", gotoTarget)
			}

			log.Printf("    Jumping to step: %s", gotoTarget)
			currentStepIndex = targetIndex
			continue
		}

		currentStepIndex++
	}

	executionTime := time.Since(startTime)
	log.Printf("Pipeline execution completed: %d steps executed in %v", stepsExecuted, executionTime)

	// Write pipeline execution context to disk
	outputDir := fmt.Sprintf("/tmp/pipeline/%s", task.ID)
	outputPath := fmt.Sprintf("%s/context.json", outputDir)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Printf("Warning: failed to create pipeline output dir: %v", err)
		outputPath = ""
	} else {
		contextJSON, err := json.Marshal(context.Steps)
		if err != nil {
			log.Printf("Warning: failed to marshal pipeline context: %v", err)
			outputPath = ""
		} else if err := os.WriteFile(outputPath, contextJSON, 0644); err != nil {
			log.Printf("Warning: failed to write pipeline context: %v", err)
			outputPath = ""
		}
	}

	// Auto-trigger extraction for ingestion pipelines
	if pipelineType, ok := task.TaskSpec.Parameters["pipeline_type"].(string); ok && pipelineType == "ingestion" {
		triggerExtractionForIngestion(orchestratorURL, task.TaskSpec.ProjectID, task.TaskSpec.PipelineID)
	}

	return &models.WorkTaskResult{
		WorkTaskID:     task.ID,
		Status:         models.WorkTaskStatusCompleted,
		OutputLocation: outputPath,
		Metadata: map[string]any{
			"pipeline_id":       task.TaskSpec.PipelineID,
			"pipeline_name":     pipeline.Name,
			"steps_executed":    len(pipeline.Steps),
			"execution_time_ms": executionTime.Milliseconds(),
			"trigger_type":      task.TaskSpec.Parameters["trigger_type"],
			"triggered_by":      task.TaskSpec.Parameters["triggered_by"],
		},
	}, nil
}

// triggerExtractionForIngestion calls POST /api/extraction/generate-ontology for an ingestion pipeline.
// This is best-effort: failures are logged but do not fail the pipeline task.
func triggerExtractionForIngestion(orchestratorURL, projectID, pipelineID string) {
	payload := map[string]interface{}{
		"project_id":           projectID,
		"storage_ids":          []string{},
		"ontology_name":        "auto-" + pipelineID,
		"include_structured":   true,
		"include_unstructured": true,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		log.Printf("auto-extraction: failed to marshal request: %v", err)
		return
	}

	url := fmt.Sprintf("%s/api/extraction/generate-ontology", orchestratorURL)
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		log.Printf("auto-extraction: HTTP call failed: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		log.Printf("auto-extraction: triggered for project %s (pipeline %s)", projectID, pipelineID)
	} else {
		log.Printf("auto-extraction: unexpected status %d for project %s", resp.StatusCode, projectID)
	}
}

// executeMLTraining executes an ML training work task
func executeMLTraining(task *models.WorkTask) (*models.WorkTaskResult, error) {
	log.Printf("Training ML model: %s", task.TaskSpec.ModelID)

	// Get orchestrator URL
	orchestratorURL := getOrchestratorURL()

	// Get model details from orchestrator
	modelURL := fmt.Sprintf("%s/api/ml-models/%s", orchestratorURL, task.TaskSpec.ModelID)
	resp, err := http.Get(modelURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch model: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch model: status %d", resp.StatusCode)
	}

	var model models.MLModel
	if err := json.NewDecoder(resp.Body).Decode(&model); err != nil {
		return nil, fmt.Errorf("failed to decode model: %w", err)
	}

	// Load training data from storage
	trainingData, err := loadTrainingDataFromStorage(orchestratorURL, task)
	if err != nil {
		reportTrainingFailure(orchestratorURL, task.TaskSpec.ModelID, err.Error())
		return nil, fmt.Errorf("failed to load training data: %w", err)
	}

	// Create trainer factory and get appropriate trainer
	factory := training.NewTrainerFactory()
	trainer, err := factory.GetTrainer(model.Type)
	if err != nil {
		return nil, fmt.Errorf("failed to get trainer: %w", err)
	}

	// Train the model
	log.Printf("Starting training for model type: %s", model.Type)
	result, err := trainer.Train(trainingData, model.TrainingConfig)
	if err != nil {
		// Report training failure to orchestrator
		reportTrainingFailure(orchestratorURL, task.TaskSpec.ModelID, err.Error())
		return nil, fmt.Errorf("training failed: %w", err)
	}

	log.Printf("Training completed - Accuracy: %.2f, Precision: %.2f, Recall: %.2f, F1: %.2f",
		result.PerformanceMetrics.Accuracy,
		result.PerformanceMetrics.Precision,
		result.PerformanceMetrics.Recall,
		result.PerformanceMetrics.F1Score)

	// Save model artifact
	artifactPath := fmt.Sprintf("/tmp/models/%s/model.json", task.TaskSpec.ModelID)
	if err := os.MkdirAll(fmt.Sprintf("/tmp/models/%s", task.TaskSpec.ModelID), 0755); err != nil {
		return nil, fmt.Errorf("failed to create model directory: %w", err)
	}

	artifactData, err := json.Marshal(map[string]any{
		"model_type":    string(model.Type),
		"feature_names": trainingData.FeatureNames,
		"parameters": map[string]any{
			"model_data":         result.ModelData,
			"feature_importance": result.FeatureImportance,
		},
		"metadata": map[string]any{
			"trained_at": time.Now().UTC().Format(time.RFC3339),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal artifact: %w", err)
	}

	if err := os.WriteFile(artifactPath, artifactData, 0644); err != nil {
		return nil, fmt.Errorf("failed to write artifact: %w", err)
	}

	// Report training completion to orchestrator
	if err := reportTrainingCompletion(orchestratorURL, task.TaskSpec.ModelID, artifactPath, result.PerformanceMetrics); err != nil {
		log.Printf("Warning: failed to report training completion: %v", err)
	}

	return &models.WorkTaskResult{
		WorkTaskID:     task.ID,
		Status:         models.WorkTaskStatusCompleted,
		OutputLocation: artifactPath,
		Metadata: map[string]any{
			"model_id":            task.TaskSpec.ModelID,
			"model_type":          string(model.Type),
			"accuracy":            result.PerformanceMetrics.Accuracy,
			"precision":           result.PerformanceMetrics.Precision,
			"recall":              result.PerformanceMetrics.Recall,
			"f1_score":            result.PerformanceMetrics.F1Score,
			"training_epochs":     result.TrainingMetrics.Epoch,
			"training_loss":       result.TrainingMetrics.TrainingLoss,
			"validation_loss":     result.TrainingMetrics.ValidationLoss,
			"training_accuracy":   result.TrainingMetrics.TrainingAccuracy,
			"validation_accuracy": result.TrainingMetrics.ValidationAccuracy,
		},
	}, nil
}

// loadTrainingDataFromStorage retrieves CIR records from storage and converts them to training data.
// It uses storage IDs from the task's DataAccess.InputDatasets, or falls back to the project's
// storage configs if no datasets are specified.
func loadTrainingDataFromStorage(orchestratorURL string, task *models.WorkTask) (*training.TrainingData, error) {
	storageIDs := task.DataAccess.InputDatasets

	// Fall back to project storage configs if no input datasets specified
	if len(storageIDs) == 0 {
		configsURL := fmt.Sprintf("%s/api/storage/configs?project_id=%s", orchestratorURL, task.ProjectID)
		resp, err := http.Get(configsURL)
		if err == nil {
			defer resp.Body.Close()
			var configs []struct {
				ID string `json:"id"`
			}
			if json.NewDecoder(resp.Body).Decode(&configs) == nil {
				for _, c := range configs {
					storageIDs = append(storageIDs, c.ID)
				}
			}
		}
	}

	if len(storageIDs) == 0 {
		return nil, fmt.Errorf("no storage IDs available")
	}

	// Retrieve CIR records from each storage
	var allCIRs []map[string]any
	for _, storageID := range storageIDs {
		retrieveURL := fmt.Sprintf("%s/api/storage/retrieve", orchestratorURL)
		body, _ := json.Marshal(map[string]any{
			"storage_id": storageID,
			"query":      map[string]any{},
		})
		resp, err := http.Post(retrieveURL, "application/json", bytes.NewBuffer(body))
		if err != nil {
			log.Printf("Warning: failed to retrieve from storage %s: %v", storageID, err)
			continue
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			continue
		}

		// The API returns an array of CIR records
		var cirs []struct {
			Data any `json:"data"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&cirs); err != nil {
			continue
		}
		for _, c := range cirs {
			if dm, ok := c.Data.(map[string]any); ok {
				allCIRs = append(allCIRs, dm)
			}
		}
	}

	if len(allCIRs) == 0 {
		return nil, fmt.Errorf("no CIR records found in storage")
	}

	// Determine label column
	labelColumn := "label"
	if task.TaskSpec.Parameters != nil {
		if lc, ok := task.TaskSpec.Parameters["label_column"].(string); ok && lc != "" {
			labelColumn = lc
		}
	}

	features, labels, featureNames := cirMapsToFeatureRows(allCIRs, labelColumn)
	if len(features) == 0 {
		return nil, fmt.Errorf("no usable feature rows extracted from CIR data")
	}

	// Train/test split (default 80/20)
	splitRatio := 0.8
	if task.TaskSpec.Parameters != nil {
		if sr, ok := task.TaskSpec.Parameters["train_test_split"].(float64); ok && sr > 0 {
			splitRatio = sr
		}
	}
	splitIdx := int(float64(len(features)) * splitRatio)
	if splitIdx <= 0 {
		splitIdx = 1
	}
	if splitIdx >= len(features) {
		splitIdx = len(features) - 1
	}

	return &training.TrainingData{
		TrainFeatures: features[:splitIdx],
		TrainLabels:   labels[:splitIdx],
		TestFeatures:  features[splitIdx:],
		TestLabels:    labels[splitIdx:],
		FeatureNames:  featureNames,
		Metadata:      map[string]any{"source": "storage"},
	}, nil
}

// cirMapsToFeatureRows converts a slice of CIR data maps to feature matrix and label vector.
// Numeric fields are used as features; the labelColumn is used as the target.
func cirMapsToFeatureRows(rows []map[string]any, labelColumn string) ([][]float64, []float64, []string) {
	if len(rows) == 0 {
		return nil, nil, nil
	}

	// Collect feature column names from first row
	featureNames := make([]string, 0)
	for k := range rows[0] {
		if k == labelColumn {
			continue
		}
		featureNames = append(featureNames, k)
	}
	sort.Strings(featureNames) // consistent ordering

	features := make([][]float64, 0, len(rows))
	labels := make([]float64, 0, len(rows))

	for _, row := range rows {
		// Extract label
		labelVal := 0.0
		if lv, ok := row[labelColumn]; ok {
			switch v := lv.(type) {
			case float64:
				labelVal = v
			case int:
				labelVal = float64(v)
			case bool:
				if v {
					labelVal = 1
				}
			case string:
				// Parse numeric string or treat as class indicator
				if f, err := strconv.ParseFloat(v, 64); err == nil {
					labelVal = f
				} else if v == "true" || v == "yes" || v == "1" {
					labelVal = 1
				}
			}
		}

		// Extract feature values
		fv := make([]float64, len(featureNames))
		for i, name := range featureNames {
			val, exists := row[name]
			if !exists {
				continue
			}
			switch v := val.(type) {
			case float64:
				fv[i] = v
			case int:
				fv[i] = float64(v)
			case bool:
				if v {
					fv[i] = 1
				}
			case string:
				if f, err := strconv.ParseFloat(v, 64); err == nil {
					fv[i] = f
				} else if v == "true" || v == "yes" {
					fv[i] = 1
				}
			}
		}

		features = append(features, fv)
		labels = append(labels, labelVal)
	}

	return features, labels, featureNames
}

// reportTrainingCompletion reports successful training to orchestrator
func reportTrainingCompletion(orchestratorURL, modelID, artifactPath string, metrics *models.PerformanceMetrics) error {
	url := fmt.Sprintf("%s/api/ml-models/%s/training/complete", orchestratorURL, modelID)
	payload := map[string]any{
		"model_artifact_path": artifactPath,
		"performance_metrics": metrics,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to report completion: status %d", resp.StatusCode)
	}

	return nil
}

// reportTrainingFailure reports training failure to orchestrator
func reportTrainingFailure(orchestratorURL, modelID, reason string) error {
	url := fmt.Sprintf("%s/api/ml-models/%s/training/fail", orchestratorURL, modelID)
	payload := map[string]any{
		"reason": reason,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to report failure: status %d", resp.StatusCode)
	}

	return nil
}

// executeMLInference loads data from storage, runs inference with the trained model, and reports results
func executeMLInference(task *models.WorkTask) (*models.WorkTaskResult, error) {
	log.Printf("Running inference with model: %s", task.TaskSpec.ModelID)

	orchestratorURL := getOrchestratorURL()

	// Fetch model artifact path
	modelURL := fmt.Sprintf("%s/api/ml-models/%s", orchestratorURL, task.TaskSpec.ModelID)
	resp, err := http.Get(modelURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch model: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch model: status %d", resp.StatusCode)
	}
	var model models.MLModel
	if err := json.NewDecoder(resp.Body).Decode(&model); err != nil {
		return nil, fmt.Errorf("failed to decode model: %w", err)
	}

	if model.ModelArtifactPath == "" {
		return nil, fmt.Errorf("model has no trained artifact (status: %s)", model.Status)
	}

	// Load inference data from storage
	inferenceData, err := loadTrainingDataFromStorage(orchestratorURL, task)
	if err != nil {
		return nil, fmt.Errorf("failed to load inference data: %w", err)
	}
	if len(inferenceData.TrainFeatures) == 0 && len(inferenceData.TestFeatures) == 0 {
		return nil, fmt.Errorf("no data available for inference with model %s", task.TaskSpec.ModelID)
	}

	// Read artifact and run inference for each row
	artifactData, err := os.ReadFile(model.ModelArtifactPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read model artifact: %w", err)
	}

	var artifact struct {
		ModelType    string                 `json:"model_type"`
		FeatureNames []string               `json:"feature_names"`
		Parameters   map[string]any `json:"parameters"`
	}
	if err := json.Unmarshal(artifactData, &artifact); err != nil {
		return nil, fmt.Errorf("failed to parse model artifact: %w", err)
	}

	allRows := append(inferenceData.TrainFeatures, inferenceData.TestFeatures...)
	results := make([]map[string]any, 0, len(allRows))

	for _, row := range allRows {
		// Build feature vector in artifact's feature name ordering (positional match)
		features := make([]float64, len(artifact.FeatureNames))
		for i := range artifact.FeatureNames {
			if i < len(row) {
				features[i] = row[i]
			}
		}

		pred, err := workerRunInference(artifact.ModelType, artifact.Parameters, features)
		if err != nil {
			log.Printf("Warning: inference failed for row: %v", err)
			pred = 0.0
		}
		results = append(results, map[string]any{
			"input":      row,
			"prediction": pred,
		})
	}

	// Write results to disk
	outputDir := fmt.Sprintf("/tmp/inference/%s", task.ID)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create inference output dir: %w", err)
	}
	outputPath := fmt.Sprintf("%s/results.json", outputDir)
	resultsJSON, err := json.Marshal(results)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal predictions: %w", err)
	}
	if err := os.WriteFile(outputPath, resultsJSON, 0644); err != nil {
		return nil, fmt.Errorf("failed to write inference results: %w", err)
	}

	log.Printf("Ran inference on %d rows using model %s", len(results), task.TaskSpec.ModelID)

	return &models.WorkTaskResult{
		WorkTaskID:     task.ID,
		Status:         models.WorkTaskStatusCompleted,
		OutputLocation: outputPath,
		Metadata: map[string]any{
			"model_id":         task.TaskSpec.ModelID,
			"predictions_made": len(results),
		},
	}, nil
}

// workerRunInference executes inference for a single feature vector using the given model type and parameters.
// This mirrors the dispatch logic in pkg/digitaltwin/inference.go.
func workerRunInference(modelType string, parameters map[string]any, features []float64) (float64, error) {
	switch modelType {
	case "decision_tree":
		modelDataRaw, ok := parameters["model_data"]
		if !ok {
			return 0.0, fmt.Errorf("model_data missing from artifact parameters for %s", modelType)
		}
		modelJSON, err := json.Marshal(modelDataRaw)
		if err != nil {
			return 0.0, fmt.Errorf("failed to marshal tree data: %w", err)
		}
		var node training.DecisionTreeModel
		if err := json.Unmarshal(modelJSON, &node); err != nil {
			return 0.0, fmt.Errorf("failed to unmarshal decision tree: %w", err)
		}
		return training.TraverseTree(&node, features), nil

	case "random_forest":
		modelDataRaw, ok := parameters["model_data"]
		if !ok {
			return 0.0, fmt.Errorf("model_data missing from artifact parameters for %s", modelType)
		}
		modelJSON, err := json.Marshal(modelDataRaw)
		if err != nil {
			return 0.0, fmt.Errorf("failed to marshal RF data: %w", err)
		}
		var rf training.RandomForestArtifact
		if err := json.Unmarshal(modelJSON, &rf); err != nil {
			return 0.0, fmt.Errorf("failed to unmarshal random forest: %w", err)
		}
		votes := make(map[float64]int)
		for _, tree := range rf.Trees {
			pred := math.Round(training.TraverseTree(tree, features))
			votes[pred]++
		}
		bestCount := 0
		bestClass := 0.0
		for class, count := range votes {
			if count > bestCount {
				bestCount = count
				bestClass = class
			}
		}
		return bestClass, nil

	case "regression":
		weightsRaw, ok := parameters["weights"]
		if !ok {
			return 0.0, fmt.Errorf("model_data missing from artifact parameters for %s", modelType)
		}
		weightsSlice, ok := weightsRaw.([]any)
		if !ok {
			return 0.0, fmt.Errorf("invalid weights format in artifact parameters for %s", modelType)
		}
		intercept := 0.0
		if b, ok := parameters["intercept"]; ok {
			if bFloat, ok := b.(float64); ok {
				intercept = bFloat
			}
		}
		pred := intercept
		for i, w := range weightsSlice {
			if i < len(features) {
				if wFloat, ok := w.(float64); ok {
					pred += wFloat * features[i]
				}
			}
		}
		return pred, nil

	case "neural_network":
		weightsRaw, ok := parameters["weights"]
		if !ok {
			return 0.0, fmt.Errorf("model_data missing from artifact parameters for %s", modelType)
		}
		biasesRaw, ok := parameters["biases"]
		if !ok {
			return 0.0, fmt.Errorf("model_data missing from artifact parameters for %s", modelType)
		}
		weightsJSON, err := json.Marshal(weightsRaw)
		if err != nil {
			return 0.0, fmt.Errorf("failed to marshal NN weights: %w", err)
		}
		biasesJSON, err := json.Marshal(biasesRaw)
		if err != nil {
			return 0.0, fmt.Errorf("failed to marshal NN biases: %w", err)
		}
		var weights [][][]float64
		var biases [][]float64
		if err := json.Unmarshal(weightsJSON, &weights); err != nil {
			return 0.0, fmt.Errorf("failed to unmarshal NN weights: %w", err)
		}
		if err := json.Unmarshal(biasesJSON, &biases); err != nil {
			return 0.0, fmt.Errorf("failed to unmarshal NN biases: %w", err)
		}
		a := make([]float64, len(features))
		copy(a, features)
		for l, w := range weights {
			outSize := len(w)
			z := make([]float64, outSize)
			for j := range outSize {
				z[j] = biases[l][j]
				for k, ak := range a {
					if k < len(w[j]) {
						z[j] += w[j][k] * ak
					}
				}
			}
			a = make([]float64, outSize)
			isOutput := l == len(weights)-1
			for j := range z {
				if isOutput {
					a[j] = 1.0 / (1.0 + math.Exp(-z[j]))
				} else if z[j] > 0 {
					a[j] = z[j]
				}
			}
		}
		if len(a) > 0 {
			return a[0], nil
		}
		return 0.0, nil

	default:
		return 0.0, fmt.Errorf("unsupported model type: %s", modelType)
	}
}

// executeDigitalTwinUpdate triggers a storage sync on the specified digital twin
func executeDigitalTwinUpdate(task *models.WorkTask) (*models.WorkTaskResult, error) {
	log.Printf("Updating digital twin for project: %s", task.ProjectID)

	orchestratorURL := getOrchestratorURL()

	// Get digital twin ID from task parameters
	twinID, _ := task.TaskSpec.Parameters["digital_twin_id"].(string)
	if twinID == "" {
		return nil, fmt.Errorf("digital_twin_id not specified in task parameters")
	}

	// Trigger sync via orchestrator API
	syncURL := fmt.Sprintf("%s/api/digital-twins/%s/sync", orchestratorURL, twinID)
	resp, err := http.Post(syncURL, "application/json", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to trigger sync: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return nil, fmt.Errorf("sync request failed: status %d", resp.StatusCode)
	}

	// Verify by reading back the twin status
	twinURL := fmt.Sprintf("%s/api/digital-twins/%s", orchestratorURL, twinID)
	twinResp, err := http.Get(twinURL)
	entitiesUpdated := 0
	if err == nil {
		defer twinResp.Body.Close()
		var twin models.DigitalTwin
		if json.NewDecoder(twinResp.Body).Decode(&twin) == nil {
			if twin.Metadata != nil {
				if n, ok := twin.Metadata["entities_synced"].(float64); ok {
					entitiesUpdated = int(n)
				}
			}
		}
	}

	log.Printf("Digital twin %s sync complete – %d entities updated", twinID, entitiesUpdated)

	return &models.WorkTaskResult{
		WorkTaskID:     task.ID,
		Status:         models.WorkTaskStatusCompleted,
		OutputLocation: fmt.Sprintf("/tmp/digital-twins/%s/", twinID),
		Metadata: map[string]any{
			"project_id":       task.ProjectID,
			"digital_twin_id":  twinID,
			"entities_updated": entitiesUpdated,
		},
	}, nil
}

// getWorkTaskFromAPI fetches a work task from the orchestrator API
func getWorkTaskFromAPI(orchestratorURL, taskID string) (*models.WorkTask, error) {
	url := fmt.Sprintf("%s/api/worktasks/%s", orchestratorURL, taskID)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch work task: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var task models.WorkTask
	if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
		return nil, fmt.Errorf("failed to decode work task: %w", err)
	}

	return &task, nil
}

// updateWorkTaskStatus updates the work task status via orchestrator API
func updateWorkTaskStatus(orchestratorURL, taskID string, status models.WorkTaskStatus, errorMsg string) error {
	result := models.WorkTaskResult{
		WorkTaskID:   taskID,
		Status:       status,
		ErrorMessage: errorMsg,
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal status update: %w", err)
	}

	url := fmt.Sprintf("%s/api/worktasks/%s", orchestratorURL, taskID)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(resultJSON))
	if err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// reportWorkTaskCompletion reports work task completion to the orchestrator
func reportWorkTaskCompletion(orchestratorURL, taskID string, status models.WorkTaskStatus, outputLocation, errorMsg string) {
	result := models.WorkTaskResult{
		WorkTaskID:     taskID,
		Status:         status,
		OutputLocation: outputLocation,
		ErrorMessage:   errorMsg,
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		log.Printf("Failed to marshal work task result: %v", err)
		return
	}

	url := fmt.Sprintf("%s/api/worktasks/%s", orchestratorURL, taskID)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(resultJSON))
	if err != nil {
		log.Printf("Failed to report work task completion: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Unexpected status code when reporting work task completion: %d", resp.StatusCode)
	}
}
