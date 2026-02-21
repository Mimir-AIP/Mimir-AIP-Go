package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"plugin"
	"time"

	"github.com/mimir-aip/mimir-aip-go/pkg/mlmodel/training"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	pipelinepkg "github.com/mimir-aip/mimir-aip-go/pkg/pipeline"
	"github.com/mimir-aip/mimir-aip-go/pkg/plugins"
)

func main() {
	// Get work task information from environment
	taskID := os.Getenv("WORKTASK_ID")
	taskType := os.Getenv("WORKTASK_TYPE")

	if taskID == "" || taskType == "" {
		log.Fatal("WORKTASK_ID and WORKTASK_TYPE must be set")
	}

	log.Printf("Worker starting for task %s (type: %s)", taskID, taskType)

	// Get orchestrator URL
	orchestratorURL := os.Getenv("ORCHESTRATOR_URL")
	if orchestratorURL == "" {
		orchestratorURL = "http://orchestrator:8080"
	}

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
	orchestratorURL := os.Getenv("ORCHESTRATOR_URL")
	if orchestratorURL == "" {
		orchestratorURL = "http://orchestrator:8080"
	}

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

	return &models.WorkTaskResult{
		WorkTaskID:     task.ID,
		Status:         models.WorkTaskStatusCompleted,
		OutputLocation: fmt.Sprintf("s3://results/%s/", task.ID),
		Metadata: map[string]interface{}{
			"pipeline_id":       task.TaskSpec.PipelineID,
			"pipeline_name":     pipeline.Name,
			"steps_executed":    len(pipeline.Steps),
			"execution_time_ms": executionTime.Milliseconds(),
			"trigger_type":      task.TaskSpec.Parameters["trigger_type"],
			"triggered_by":      task.TaskSpec.Parameters["triggered_by"],
		},
	}, nil
}

// executeMLTraining executes an ML training work task
func executeMLTraining(task *models.WorkTask) (*models.WorkTaskResult, error) {
	log.Printf("Training ML model: %s", task.TaskSpec.ModelID)

	// Get orchestrator URL
	orchestratorURL := os.Getenv("ORCHESTRATOR_URL")
	if orchestratorURL == "" {
		orchestratorURL = "http://orchestrator:8080"
	}

	// Get model details from orchestrator
	modelURL := fmt.Sprintf("%s/api/mlmodels/%s", orchestratorURL, task.TaskSpec.ModelID)
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

	// Generate synthetic training data
	// In production, this would load from storage_ids specified in task spec
	trainingData := generateSyntheticTrainingData(50, 4)

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

	artifactData, err := json.Marshal(map[string]interface{}{
		"model_type":    string(model.Type),
		"feature_names": trainingData.FeatureNames,
		"parameters": map[string]interface{}{
			"model_data":         result.ModelData,
			"feature_importance": result.FeatureImportance,
		},
		"metadata": map[string]interface{}{
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
		Metadata: map[string]interface{}{
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

// generateSyntheticTrainingData creates synthetic data for demonstration
func generateSyntheticTrainingData(samples, features int) *training.TrainingData {
	// 70/30 train/test split
	trainSamples := int(float64(samples) * 0.7)
	testSamples := samples - trainSamples

	trainFeatures := make([][]float64, trainSamples)
	trainLabels := make([]float64, trainSamples)
	testFeatures := make([][]float64, testSamples)
	testLabels := make([]float64, testSamples)

	// Generate random data with some pattern
	for i := 0; i < trainSamples; i++ {
		trainFeatures[i] = make([]float64, features)
		sum := 0.0
		for j := 0; j < features; j++ {
			trainFeatures[i][j] = rand.Float64() * 10
			sum += trainFeatures[i][j]
		}
		// Simple decision boundary: label 1 if sum > threshold, else 0
		if sum > float64(features)*5 {
			trainLabels[i] = 1
		} else {
			trainLabels[i] = 0
		}
	}

	for i := 0; i < testSamples; i++ {
		testFeatures[i] = make([]float64, features)
		sum := 0.0
		for j := 0; j < features; j++ {
			testFeatures[i][j] = rand.Float64() * 10
			sum += testFeatures[i][j]
		}
		if sum > float64(features)*5 {
			testLabels[i] = 1
		} else {
			testLabels[i] = 0
		}
	}

	featureNames := make([]string, features)
	for i := 0; i < features; i++ {
		featureNames[i] = fmt.Sprintf("feature_%d", i)
	}

	return &training.TrainingData{
		TrainFeatures: trainFeatures,
		TrainLabels:   trainLabels,
		TestFeatures:  testFeatures,
		TestLabels:    testLabels,
		FeatureNames:  featureNames,
		Metadata:      make(map[string]interface{}),
	}
}

// reportTrainingCompletion reports successful training to orchestrator
func reportTrainingCompletion(orchestratorURL, modelID, artifactPath string, metrics *models.PerformanceMetrics) error {
	url := fmt.Sprintf("%s/api/mlmodels/%s/training/complete", orchestratorURL, modelID)
	payload := map[string]interface{}{
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
	url := fmt.Sprintf("%s/api/mlmodels/%s/training/fail", orchestratorURL, modelID)
	payload := map[string]interface{}{
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

// executeMLInference executes an ML inference work task
func executeMLInference(task *models.WorkTask) (*models.WorkTaskResult, error) {
	log.Printf("Running inference with model: %s", task.TaskSpec.ModelID)

	// Simulate ML inference
	time.Sleep(1 * time.Second)

	return &models.WorkTaskResult{
		WorkTaskID:     task.ID,
		Status:         models.WorkTaskStatusCompleted,
		OutputLocation: fmt.Sprintf("s3://predictions/%s/", task.ID),
		Metadata: map[string]interface{}{
			"model_id":         task.TaskSpec.ModelID,
			"predictions_made": 500,
		},
	}, nil
}

// executeDigitalTwinUpdate executes a digital twin update work task
func executeDigitalTwinUpdate(task *models.WorkTask) (*models.WorkTaskResult, error) {
	log.Printf("Updating digital twin for project: %s", task.ProjectID)

	// Simulate digital twin update
	time.Sleep(2 * time.Second)

	return &models.WorkTaskResult{
		WorkTaskID:     task.ID,
		Status:         models.WorkTaskStatusCompleted,
		OutputLocation: fmt.Sprintf("s3://digital-twins/%s/", task.ProjectID),
		Metadata: map[string]interface{}{
			"project_id":       task.ProjectID,
			"entities_updated": 100,
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
