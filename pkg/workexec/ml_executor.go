package workexec

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/mimir-aip/mimir-aip-go/pkg/mlmodel"
	"github.com/mimir-aip/mimir-aip-go/pkg/mlmodel/training"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"github.com/mimir-aip/mimir-aip-go/pkg/plugins"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"sort"
	"strconv"
)

func resolveWorkerMLProvider(orchestratorURL string, model *models.MLModel) (mlmodel.Provider, error) {
	providerName := model.Provider
	if providerName == "" {
		providerName = "builtin"
	}
	if providerName == "builtin" {
		return mlmodel.NewBuiltinProvider(), nil
	}
	mlClient := plugins.NewMLClient(orchestratorURL, "/tmp/plugins/ml")
	if _, err := mlClient.CompileProvider(providerName); err != nil {
		return nil, err
	}
	return mlClient.LoadProvider(providerName)
}

func fetchModelWithOwnership(orchestratorURL, projectID, modelID string) (*models.MLModel, error) {
	modelURL := fmt.Sprintf("%s/api/ml-models/%s?project_id=%s", orchestratorURL, modelID, projectID)
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
	return &model, nil
}

// executeMLTraining executes an ML training work task
func executeMLTraining(task *models.WorkTask) (*models.WorkTaskResult, error) {
	log.Printf("Training ML model: %s", task.TaskSpec.ModelID)
	orchestratorURL := getOrchestratorURL()
	model, err := fetchModelWithOwnership(orchestratorURL, task.ProjectID, task.TaskSpec.ModelID)
	if err != nil {
		return nil, err
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
	provider, err := resolveWorkerMLProvider(orchestratorURL, model)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve provider: %w", err)
	}
	result, err := provider.Train(&mlmodel.ProviderTrainRequest{Model: model, TrainingData: trainingData})
	if err != nil {
		reportTrainingFailure(orchestratorURL, task.TaskSpec.ModelID, err.Error())
		return nil, fmt.Errorf("training failed: %w", err)
	}
	if err := reportTrainingCompletion(orchestratorURL, task.TaskSpec.ModelID, result.ArtifactData, result.PerformanceMetrics); err != nil {
		log.Printf("Warning: failed to report training completion: %v", err)
	}
	metadata := map[string]any{
		"model_id":          task.TaskSpec.ModelID,
		"provider":          model.Provider,
		"provider_model":    model.ProviderModel,
		"artifact_uploaded": true,
	}
	if result.TrainingMetrics != nil {
		metadata["training_epochs"] = result.TrainingMetrics.Epoch
		metadata["training_loss"] = result.TrainingMetrics.TrainingLoss
		metadata["validation_loss"] = result.TrainingMetrics.ValidationLoss
		metadata["training_accuracy"] = result.TrainingMetrics.TrainingAccuracy
		metadata["validation_accuracy"] = result.TrainingMetrics.ValidationAccuracy
	}
	if result.PerformanceMetrics != nil {
		metadata["accuracy"] = result.PerformanceMetrics.Accuracy
		metadata["precision"] = result.PerformanceMetrics.Precision
		metadata["recall"] = result.PerformanceMetrics.Recall
		metadata["f1_score"] = result.PerformanceMetrics.F1Score
	}
	for key, value := range result.AdditionalMetadata {
		metadata[key] = value
	}
	return &models.WorkTaskResult{WorkTaskID: task.ID, Status: models.WorkTaskStatusCompleted, Metadata: metadata}, nil
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
			"project_id": task.ProjectID,
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
	model, err := fetchModelWithOwnership(orchestratorURL, task.ProjectID, task.TaskSpec.ModelID)
	if err != nil {
		return nil, err
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
	provider, err := resolveWorkerMLProvider(orchestratorURL, model)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve provider: %w", err)
	}
	artifact, _ := mlmodel.ReadBuiltinArtifactForWorker(model.ModelArtifactPath)
	allRows := append(inferenceData.TrainFeatures, inferenceData.TestFeatures...)
	results := make([]map[string]any, 0, len(allRows))
	inferenceFailures := 0
	for _, row := range allRows {
		input := make(map[string]any)
		featureNames := inferenceData.FeatureNames
		if artifact != nil && len(artifact.FeatureNames) > 0 {
			featureNames = artifact.FeatureNames
		}
		for i, featureName := range featureNames {
			if i < len(row) {
				input[featureName] = row[i]
			}
		}
		result, err := provider.Infer(&mlmodel.ProviderInferRequest{Model: model, Input: input})
		if err != nil {
			inferenceFailures++
			log.Printf("Inference failed for row %v: %v", row, err)
			continue
		}
		results = append(results, map[string]any{"input": input, "prediction": result.Output, "confidence": result.Confidence, "metadata": result.Metadata})
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
			"provider":         model.Provider,
			"provider_model":   model.ProviderModel,
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
