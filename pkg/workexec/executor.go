package workexec

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/mimir-aip/mimir-aip-go/pkg/mlmodel/training"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	pipelinepkg "github.com/mimir-aip/mimir-aip-go/pkg/pipeline"
	"github.com/mimir-aip/mimir-aip-go/pkg/pluginruntime"
	"github.com/mimir-aip/mimir-aip-go/pkg/plugins"
)

// RunFromEnvironment executes one work task using the current process environment.
// It keeps the existing worker contract so both the standalone worker binary and
// future in-process local execution can share one implementation.
func RunFromEnvironment() error {
	taskID := os.Getenv("WORKTASK_ID")
	taskType := os.Getenv("WORKTASK_TYPE")

	if taskID == "" || taskType == "" {
		return fmt.Errorf("WORKTASK_ID and WORKTASK_TYPE must be set")
	}

	log.Printf("Worker starting for task %s (type: %s)", taskID, taskType)

	orchestratorURL := getOrchestratorURL()
	task, err := getWorkTaskFromAPI(orchestratorURL, taskID)
	if err != nil {
		return fmt.Errorf("failed to get work task details: %w", err)
	}

	if err := updateWorkTaskStatus(orchestratorURL, taskID, models.WorkTaskStatusExecuting, ""); err != nil {
		log.Printf("Warning: Failed to update work task status to executing: %v", err)
	}

	result, err := executeWorkTask(task)
	if err != nil {
		log.Printf("Work task execution failed: %v", err)
		reportWorkTaskCompletion(orchestratorURL, taskID, models.WorkTaskStatusFailed, "", err.Error(), nil)
		return err
	}

	log.Printf("Work task %s completed successfully", taskID)
	reportWorkTaskCompletion(orchestratorURL, taskID, models.WorkTaskStatusCompleted, result.OutputLocation, "", result.Metadata)
	return nil
}

func getWorkerAuthToken() string {
	return os.Getenv("WORKER_AUTH_TOKEN")
}

func doOrchestratorRequest(method, url string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token := getWorkerAuthToken(); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return http.DefaultClient.Do(req)
}

func getOrchestratorURL() string {
	if url := os.Getenv("ORCHESTRATOR_URL"); url != "" {
		return url
	}
	return "http://orchestrator:8080"
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
	case models.WorkTaskTypeDigitalTwinProcessing:
		return executeDigitalTwinProcessing(task)
	default:
		return nil, fmt.Errorf("unknown work task type: %s", task.Type)
	}
}

// executePipeline executes a pipeline work task
func executePipeline(task *models.WorkTask) (*models.WorkTaskResult, error) {
	log.Printf("Executing pipeline: %s", task.TaskSpec.PipelineID)

	orchestratorURL := getOrchestratorURL()

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

	log.Printf("Executing pipeline %s (%s) with %d steps", pipeline.Name, pipeline.ID, len(pipeline.Steps))

	context := models.NewPipelineContext(10485760)
	if task.TaskSpec.Parameters != nil {
		for key, value := range task.TaskSpec.Parameters {
			context.SetStepData("_parameters", key, value)
		}
	}

	context.SetStepData("_runtime", "project_id", pipeline.ProjectID)
	context.SetStepData("_runtime", "pipeline_id", pipeline.ID)
	context.SetStepData("_runtime", "trigger_type", fmt.Sprintf("%v", task.TaskSpec.Parameters["trigger_type"]))

	pluginCacheDir := "/tmp/plugins"
	pluginClient := plugins.NewClient(orchestratorURL, pluginCacheDir)

	pluginRegistry := pluginruntime.NewRegistry[pipelinepkg.Plugin]()
	storageClient := pipelinepkg.NewHTTPStorageClient(orchestratorURL)
	checkpointClient := pipelinepkg.NewHTTPCheckpointStore(orchestratorURL)
	pluginRegistry.Register("default", pipelinepkg.NewDefaultPluginWithDeps(storageClient, checkpointClient))
	pluginRegistry.Register("builtin", pipelinepkg.NewDefaultPluginWithDeps(storageClient, checkpointClient))

	uniquePlugins := make(map[string]bool)
	for _, step := range pipeline.Steps {
		if step.Plugin != "default" && step.Plugin != "builtin" {
			uniquePlugins[step.Plugin] = true
		}
	}

	for pluginName := range uniquePlugins {
		log.Printf("Compiling plugin: %s", pluginName)
		pluginPath, err := pluginClient.CompilePlugin(pluginName)
		if err != nil {
			log.Printf("Warning: Failed to compile plugin %s: %v", pluginName, err)
			continue
		}

		log.Printf("Loading plugin from: %s", pluginPath)
		pluginInstance, err := pluginClient.LoadPlugin(pluginName)
		if err != nil {
			log.Printf("Warning: Failed to load plugin %s: %v", pluginName, err)
			continue
		}

		pluginRegistry.Register(pluginName, pluginInstance)
		log.Printf("Loaded custom plugin: %s", pluginName)
	}

	startTime := time.Now()
	currentStepIndex := 0
	stepsExecuted := 0

	for currentStepIndex < len(pipeline.Steps) {
		step := pipeline.Steps[currentStepIndex]
		stepsExecuted++

		log.Printf("  Step %d: %s (%s.%s)", currentStepIndex+1, step.Name, step.Plugin, step.Action)

		pluginInstance, ok := pluginRegistry.Get(step.Plugin)
		if !ok {
			return nil, fmt.Errorf("unknown plugin: %s", step.Plugin)
		}

		context.SetStepData("_runtime", "current_step", step.Name)

		result, err := pluginInstance.Execute(step.Action, step.Parameters, context)
		if err != nil {
			return nil, fmt.Errorf("step %s failed: %w", step.Name, err)
		}

		for key, value := range result {
			context.SetStepData(step.Name, key, value)
		}

		if step.Output != nil {
			for outputKey, outputTemplate := range step.Output {
				if dp, ok := pluginInstance.(*pipelinepkg.DefaultPlugin); ok {
					resolvedValue := dp.ResolveTemplates(outputTemplate, context)
					context.SetStepData(step.Name, outputKey, resolvedValue)
					log.Printf("    Output: %s = %v", outputKey, resolvedValue)
				}
			}
		}

		if gotoTarget, ok := result["goto"].(string); ok {
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
//
// It discovers ALL storage configs for the project (not just those used by
// the current pipeline) so that cross-source link detection sees the full
// dataset. This enables a unified ontology across multiple ingestion pipelines
// feeding the same project.
//
// This is best-effort: failures are logged but do not fail the pipeline task.
func triggerExtractionForIngestion(orchestratorURL, projectID, pipelineID string) {
	storageIDs := fetchProjectStorageIDs(orchestratorURL, projectID)

	payload := map[string]interface{}{
		"project_id":           projectID,
		"storage_ids":          storageIDs,
		"ontology_name":        "auto-" + projectID,
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
		log.Printf("auto-extraction: triggered for project %s (pipeline %s, %d storage sources)", projectID, pipelineID, len(storageIDs))
	} else {
		log.Printf("auto-extraction: unexpected status %d for project %s", resp.StatusCode, projectID)
	}
}

// fetchProjectStorageIDs retrieves the IDs of all storage configs for a project.
// Returns an empty slice on any error (extraction will still run but without data).
func fetchProjectStorageIDs(orchestratorURL, projectID string) []string {
	url := fmt.Sprintf("%s/api/storage/configs?project_id=%s", orchestratorURL, projectID)
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("auto-extraction: failed to fetch storage configs: %v", err)
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Printf("auto-extraction: unexpected status %d fetching storage configs", resp.StatusCode)
		return nil
	}
	var configs []struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&configs); err != nil {
		log.Printf("auto-extraction: failed to decode storage configs: %v", err)
		return nil
	}
	ids := make([]string, 0, len(configs))
	for _, c := range configs {
		if c.ID != "" {
			ids = append(ids, c.ID)
		}
	}
	return ids
}

// executeMLTraining executes an ML training work task
func executeMLTraining(task *models.WorkTask) (*models.WorkTaskResult, error) {
	log.Printf("Training ML model: %s", task.TaskSpec.ModelID)

	orchestratorURL := getOrchestratorURL()

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

	labelColumn := "label"
	if lc, ok := task.TaskSpec.Parameters["label_column"].(string); ok && lc != "" {
		labelColumn = lc
	}
	splitRatio := 0.8
	if sr, ok := task.TaskSpec.Parameters["train_test_split"].(float64); ok && sr > 0 {
		splitRatio = sr
	}

	var trainingData *training.TrainingData
	if twinID, ok := task.TaskSpec.Parameters["digital_twin_id"].(string); ok && twinID != "" {
		log.Printf("Loading training data from digital twin %s", twinID)
		trainingData, err = loadTrainingDataFromDigitalTwin(orchestratorURL, twinID, labelColumn, splitRatio)
		if err != nil {
			log.Printf("Warning: failed to load from digital twin (%v); falling back to raw CIR data", err)
			trainingData, err = loadTrainingDataFromStorage(orchestratorURL, task)
		}
	} else {
		trainingData, err = loadTrainingDataFromStorage(orchestratorURL, task)
	}
	if err != nil {
		reportTrainingFailure(orchestratorURL, task.TaskSpec.ModelID, err.Error())
		return nil, fmt.Errorf("failed to load training data: %w", err)
	}

	factory := training.NewTrainerFactory()
	trainer, err := factory.GetTrainer(model.Type)
	if err != nil {
		return nil, fmt.Errorf("failed to get trainer: %w", err)
	}

	log.Printf("Starting training for model type: %s", model.Type)
	result, err := trainer.Train(trainingData, model.TrainingConfig)
	if err != nil {
		reportTrainingFailure(orchestratorURL, task.TaskSpec.ModelID, err.Error())
		return nil, fmt.Errorf("training failed: %w", err)
	}

	log.Printf("Training completed - Accuracy: %.2f, Precision: %.2f, Recall: %.2f, F1: %.2f",
		result.PerformanceMetrics.Accuracy,
		result.PerformanceMetrics.Precision,
		result.PerformanceMetrics.Recall,
		result.PerformanceMetrics.F1Score)

	artifactData, err := json.Marshal(map[string]any{
		"model_type":    string(model.Type),
		"feature_names": trainingData.FeatureNames,
		"parameters": map[string]any{
			"model_data":         result.ModelData,
			"feature_importance": result.FeatureImportance,
		},
		"metadata": map[string]any{
			"trained_at":   time.Now().UTC().Format(time.RFC3339),
			"label_column": labelColumn,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal artifact: %w", err)
	}

	if err := reportTrainingCompletion(orchestratorURL, task.TaskSpec.ModelID, artifactData, result.PerformanceMetrics); err != nil {
		log.Printf("Warning: failed to report training completion: %v", err)
	}

	return &models.WorkTaskResult{
		WorkTaskID: task.ID,
		Status:     models.WorkTaskStatusCompleted,
		Metadata: map[string]any{
			"model_id":            task.TaskSpec.ModelID,
			"model_type":          string(model.Type),
			"artifact_uploaded":   true,
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
		Metadata: map[string]any{
			"source":       "storage",
			"label_column": labelColumn,
		},
	}, nil
}

// loadTrainingDataFromDigitalTwin fetches resolved entity attributes from a
// digital twin and converts them into a training dataset.
//
// Unlike loadTrainingDataFromStorage (which reads raw, per-source CIR records),
// this function reads the digital twin's already-resolved entity list where each
// entity's Attributes map contains merged data from ALL contributing storage
// sources. For cross-source projects (e.g. grades DB + attendance DB), this
// means a single training row for student_id=42 contains both their grade
// average AND their attendance count — enabling models to learn relationships
// that span source boundaries.
//
// Falls back: callers should fall back to loadTrainingDataFromStorage on error.
func loadTrainingDataFromDigitalTwin(orchestratorURL, twinID, labelColumn string, splitRatio float64) (*training.TrainingData, error) {
	url := fmt.Sprintf("%s/api/digital-twins/%s/entities", orchestratorURL, twinID)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch entities: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d fetching digital twin entities", resp.StatusCode)
	}

	var entities []struct {
		Attributes map[string]any `json:"attributes"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&entities); err != nil {
		return nil, fmt.Errorf("failed to decode entities: %w", err)
	}

	rows := make([]map[string]any, 0, len(entities))
	for _, e := range entities {
		if len(e.Attributes) > 0 {
			rows = append(rows, e.Attributes)
		}
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("no entities with attributes found in digital twin %s", twinID)
	}

	features, labels, featureNames := cirMapsToFeatureRows(rows, labelColumn)
	if len(features) == 0 {
		return nil, fmt.Errorf("no usable feature rows from digital twin entities (twin: %s)", twinID)
	}

	splitIdx := int(float64(len(features)) * splitRatio)
	if splitIdx <= 0 {
		splitIdx = 1
	}
	if splitIdx >= len(features) {
		splitIdx = len(features) - 1
	}

	log.Printf("Loaded %d training rows from digital twin %s (%d features)", len(features), twinID, len(featureNames))

	return &training.TrainingData{
		TrainFeatures: features[:splitIdx],
		TrainLabels:   labels[:splitIdx],
		TestFeatures:  features[splitIdx:],
		TestLabels:    labels[splitIdx:],
		FeatureNames:  featureNames,
		Metadata: map[string]any{
			"source":       "digital_twin",
			"twin_id":      twinID,
			"label_column": labelColumn,
		},
	}, nil
}

// cirMapsToFeatureRows converts a slice of CIR data maps to feature matrix and label vector.
// Numeric fields are used as features; the labelColumn is used as the target.
func cirMapsToFeatureRows(rows []map[string]any, labelColumn string) ([][]float64, []float64, []string) {
	if len(rows) == 0 {
		return nil, nil, nil
	}

	featureNames := make([]string, 0)
	for k := range rows[0] {
		if k == labelColumn {
			continue
		}
		featureNames = append(featureNames, k)
	}
	sort.Strings(featureNames)

	features := make([][]float64, 0, len(rows))
	labels := make([]float64, 0, len(rows))

	for _, row := range rows {
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
				if f, err := strconv.ParseFloat(v, 64); err == nil {
					labelVal = f
				} else if v == "true" || v == "yes" || v == "1" {
					labelVal = 1
				}
			}
		}

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

// reportTrainingCompletion reports successful training to orchestrator.
func reportTrainingCompletion(orchestratorURL, modelID string, artifactData []byte, metrics *models.PerformanceMetrics) error {
	url := fmt.Sprintf("%s/api/ml-models/%s/training/complete", orchestratorURL, modelID)
	payload := map[string]any{
		"artifact_data_base64": base64.StdEncoding.EncodeToString(artifactData),
		"performance_metrics":  metrics,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err := doOrchestratorRequest(http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("orchestrator returned status %d: %s", resp.StatusCode, string(body))
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

	inferenceData, err := loadTrainingDataFromStorage(orchestratorURL, task)
	if err != nil {
		return nil, fmt.Errorf("failed to load inference data: %w", err)
	}
	if len(inferenceData.TrainFeatures) == 0 && len(inferenceData.TestFeatures) == 0 {
		return nil, fmt.Errorf("no data available for inference with model %s", task.TaskSpec.ModelID)
	}

	artifactData, err := os.ReadFile(model.ModelArtifactPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read model artifact: %w", err)
	}

	var artifact struct {
		ModelType    string         `json:"model_type"`
		FeatureNames []string       `json:"feature_names"`
		Parameters   map[string]any `json:"parameters"`
	}
	if err := json.Unmarshal(artifactData, &artifact); err != nil {
		return nil, fmt.Errorf("failed to parse model artifact: %w", err)
	}

	allRows := append(inferenceData.TrainFeatures, inferenceData.TestFeatures...)
	results := make([]map[string]any, 0, len(allRows))

	inferenceFailures := 0
	for _, row := range allRows {
		features := make([]float64, len(artifact.FeatureNames))
		for i := range artifact.FeatureNames {
			if i < len(row) {
				features[i] = row[i]
			}
		}

		pred, err := workerRunInference(artifact.ModelType, artifact.Parameters, features)
		if err != nil {
			inferenceFailures++
			log.Printf("Inference failed for row %v: %v", row, err)
			continue
		}
		results = append(results, map[string]any{
			"input":      row,
			"prediction": pred,
		})
	}

	if inferenceFailures > 0 {
		return nil, fmt.Errorf("inference failed for %d/%d rows", inferenceFailures, len(allRows))
	}

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
		modelDataRaw, ok := parameters["model_data"]
		if !ok {
			return 0.0, fmt.Errorf("model_data missing from artifact parameters for %s", modelType)
		}
		modelData, ok := modelDataRaw.(map[string]any)
		if !ok {
			return 0.0, fmt.Errorf("invalid model_data format in artifact parameters for %s", modelType)
		}

		coeffsRaw, ok := modelData["coefficients"]
		if !ok {
			return 0.0, fmt.Errorf("coefficients missing from model_data for %s", modelType)
		}
		weightsSlice, ok := coeffsRaw.([]any)
		if !ok {
			return 0.0, fmt.Errorf("invalid coefficients format in model_data for %s", modelType)
		}
		intercept := 0.0
		if b, ok := modelData["intercept"]; ok {
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
		modelDataRaw, ok := parameters["model_data"]
		if !ok {
			return 0.0, fmt.Errorf("model_data missing from artifact parameters for %s", modelType)
		}
		modelData, ok := modelDataRaw.(map[string]any)
		if !ok {
			return 0.0, fmt.Errorf("invalid model_data format in artifact parameters for %s", modelType)
		}
		weightsRaw, ok := modelData["weights"]
		if !ok {
			return 0.0, fmt.Errorf("weights missing from model_data for %s", modelType)
		}
		biasesRaw, ok := modelData["biases"]
		if !ok {
			return 0.0, fmt.Errorf("biases missing from model_data for %s", modelType)
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

// executeDigitalTwinProcessing executes one explicit twin-processing run using only
// task parameters persisted by the orchestrator. Workers remain stateless.
func executeDigitalTwinProcessing(task *models.WorkTask) (*models.WorkTaskResult, error) {
	log.Printf("Executing digital twin processing for project: %s", task.ProjectID)

	orchestratorURL := getOrchestratorURL()
	runID, _ := task.TaskSpec.Parameters["processing_run_id"].(string)
	twinID, _ := task.TaskSpec.Parameters["digital_twin_id"].(string)
	if runID == "" {
		return nil, fmt.Errorf("processing_run_id not specified in task parameters")
	}
	if twinID == "" {
		return nil, fmt.Errorf("digital_twin_id not specified in task parameters")
	}

	executeURL := fmt.Sprintf("%s/api/internal/twin-runs/%s/execute", orchestratorURL, runID)
	resp, err := doOrchestratorRequest(http.MethodPost, executeURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to execute twin processing run: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("twin processing run execution failed: status %d", resp.StatusCode)
	}

	var run models.TwinProcessingRun
	if err := json.NewDecoder(resp.Body).Decode(&run); err != nil {
		return nil, fmt.Errorf("failed to decode twin processing run: %w", err)
	}

	outputLocation := fmt.Sprintf("/tmp/digital-twins/%s/runs/%s", twinID, runID)
	log.Printf("Digital twin processing run %s completed with status %s", runID, run.Status)
	return &models.WorkTaskResult{
		WorkTaskID:     task.ID,
		Status:         models.WorkTaskStatusCompleted,
		OutputLocation: outputLocation,
		Metadata: map[string]any{
			"project_id":        task.ProjectID,
			"digital_twin_id":   twinID,
			"processing_run_id": runID,
			"run_status":        run.Status,
			"trigger_type":      run.TriggerType,
			"insight_count":     run.Metrics["insight_count"],
		},
	}, nil
}

// getWorkTaskFromAPI fetches a work task from the orchestrator API
func getWorkTaskFromAPI(orchestratorURL, taskID string) (*models.WorkTask, error) {
	url := fmt.Sprintf("%s/api/worktasks/%s", orchestratorURL, taskID)
	resp, err := doOrchestratorRequest(http.MethodGet, url, nil)
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
	resp, err := doOrchestratorRequest(http.MethodPost, url, bytes.NewBuffer(resultJSON))
	if err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// reportWorkTaskCompletion reports work task completion to the orchestrator.
func reportWorkTaskCompletion(orchestratorURL, taskID string, status models.WorkTaskStatus, outputLocation, errorMsg string, metadata map[string]any) {
	result := models.WorkTaskResult{
		WorkTaskID:     taskID,
		Status:         status,
		OutputLocation: outputLocation,
		Metadata:       metadata,
		ErrorMessage:   errorMsg,
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		log.Printf("Failed to marshal work task result: %v", err)
		return
	}

	url := fmt.Sprintf("%s/api/worktasks/%s", orchestratorURL, taskID)
	resp, err := doOrchestratorRequest(http.MethodPost, url, bytes.NewBuffer(resultJSON))
	if err != nil {
		log.Printf("Failed to report work task completion: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Unexpected status code when reporting work task completion: %d", resp.StatusCode)
	}
}
