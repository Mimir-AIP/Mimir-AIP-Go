package ml

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/KnowledgeGraph"
	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/Ontology"
	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/Storage"
	"github.com/google/uuid"
)

// AutoTrainer automatically trains ML models based on ontology analysis
type AutoTrainer struct {
	Storage   *storage.PersistenceBackend
	KGClient  *knowledgegraph.TDB2Backend
	Analyzer  *OntologyAnalyzer
	Extractor *KGDataExtractor
}

// NewAutoTrainer creates a new auto-trainer
func NewAutoTrainer(
	store *storage.PersistenceBackend,
	kgClient *knowledgegraph.TDB2Backend,
) *AutoTrainer {
	return &AutoTrainer{
		Storage:   store,
		KGClient:  kgClient,
		Analyzer:  NewOntologyAnalyzer(store),
		Extractor: NewKGDataExtractor(store, kgClient),
	}
}

// AutoTrainingResult holds the results of automatic model training
type AutoTrainingResult struct {
	OntologyID            string               `json:"ontology_id"`
	ModelsCreated         int                  `json:"models_created"`
	ModelsFailed          int                  `json:"models_failed"`
	MonitoringJobsCreated int                  `json:"monitoring_jobs_created"`
	RulesCreated          int                  `json:"rules_created"`
	TrainedModels         []TrainedModelInfo   `json:"trained_models"`
	FailedModels          []FailedModelInfo    `json:"failed_models"`
	MonitoringSetup       *MonitoringSetupInfo `json:"monitoring_setup,omitempty"`
	TotalDuration         time.Duration        `json:"total_duration"`
	Summary               string               `json:"summary"`
}

// TrainedModelInfo contains information about a successfully trained model
type TrainedModelInfo struct {
	ModelID        string        `json:"model_id"`
	TargetProperty string        `json:"target_property"`
	ModelType      string        `json:"model_type"`
	Accuracy       float64       `json:"accuracy,omitempty"` // Classification
	R2Score        float64       `json:"r2_score,omitempty"` // Regression
	RMSE           float64       `json:"rmse,omitempty"`     // Regression
	SampleCount    int           `json:"sample_count"`
	FeatureCount   int           `json:"feature_count"`
	TrainingTime   time.Duration `json:"training_time"`
	Confidence     float64       `json:"confidence"`
	Reasoning      string        `json:"reasoning"`
}

// FailedModelInfo contains information about a model that failed to train
type FailedModelInfo struct {
	TargetProperty string  `json:"target_property"`
	ModelType      string  `json:"model_type"`
	ErrorMessage   string  `json:"error_message"`
	Confidence     float64 `json:"confidence"`
}

// MonitoringSetupInfo contains information about monitoring setup
type MonitoringSetupInfo struct {
	JobID        string   `json:"job_id"`
	MetricsCount int      `json:"metrics_count"`
	RulesCreated []string `json:"rules_created"`
	CronSchedule string   `json:"cron_schedule"`
}

// TrainFromOntology automatically trains models based on ontology analysis
func (at *AutoTrainer) TrainFromOntology(ctx context.Context, ontologyID string, options *AutoTrainOptions) (*AutoTrainingResult, error) {
	startTime := time.Now()

	// Analyze ontology capabilities
	capabilities, err := at.Analyzer.AnalyzeMLCapabilities(ctx, ontologyID)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze ontology: %w", err)
	}

	result := &AutoTrainingResult{
		OntologyID:    ontologyID,
		TrainedModels: []TrainedModelInfo{},
		FailedModels:  []FailedModelInfo{},
	}

	// Get ontology properties for training
	allProperties, err := at.Analyzer.getOntologyProperties(ontologyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get ontology properties: %w", err)
	}

	// Train regression models
	if options.EnableRegression {
		for _, target := range capabilities.RegressionTargets {
			// Skip low-confidence targets unless forced
			if target.Confidence < options.MinConfidence && !options.ForceAll {
				continue
			}

			modelInfo, err := at.trainModelForTarget(ctx, ontologyID, target, allProperties, "regression")
			if err != nil {
				result.FailedModels = append(result.FailedModels, FailedModelInfo{
					TargetProperty: target.PropertyLabel,
					ModelType:      "regression",
					ErrorMessage:   err.Error(),
					Confidence:     target.Confidence,
				})
				result.ModelsFailed++
			} else {
				result.TrainedModels = append(result.TrainedModels, *modelInfo)
				result.ModelsCreated++
			}
		}
	}

	// Train classification models
	if options.EnableClassification {
		for _, target := range capabilities.ClassificationTargets {
			// Skip low-confidence targets unless forced
			if target.Confidence < options.MinConfidence && !options.ForceAll {
				continue
			}

			modelInfo, err := at.trainModelForTarget(ctx, ontologyID, target, allProperties, "classification")
			if err != nil {
				result.FailedModels = append(result.FailedModels, FailedModelInfo{
					TargetProperty: target.PropertyLabel,
					ModelType:      "classification",
					ErrorMessage:   err.Error(),
					Confidence:     target.Confidence,
				})
				result.ModelsFailed++
			} else {
				result.TrainedModels = append(result.TrainedModels, *modelInfo)
				result.ModelsCreated++
			}
		}
	}

	// Setup monitoring if enabled
	if options.EnableMonitoring && len(capabilities.TimeSeriesMetrics) > 0 {
		monitoringInfo, err := at.setupAutomaticMonitoring(ctx, ontologyID, capabilities)
		if err != nil {
			// Don't fail the entire operation if monitoring setup fails
			fmt.Printf("Warning: monitoring setup failed: %v\n", err)
		} else {
			result.MonitoringSetup = monitoringInfo
			result.MonitoringJobsCreated = 1
			result.RulesCreated = len(monitoringInfo.RulesCreated)
		}
	}

	result.TotalDuration = time.Since(startTime)
	result.Summary = at.generateResultSummary(result)

	return result, nil
}

// trainModelForTarget trains a model for a specific target property
func (at *AutoTrainer) trainModelForTarget(
	ctx context.Context,
	ontologyID string,
	target MLTarget,
	allProperties []ontology.OntologyProperty,
	modelType string,
) (*TrainedModelInfo, error) {
	// Find the target property object
	var targetProp ontology.OntologyProperty
	found := false
	for _, prop := range allProperties {
		if prop.Label == target.PropertyLabel {
			targetProp = prop
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("target property %s not found in ontology", target.PropertyLabel)
	}

	// Find feature properties
	featureProps := []ontology.OntologyProperty{}
	for _, featureName := range target.SuggestedFeatures {
		for _, prop := range allProperties {
			if prop.Label == featureName {
				featureProps = append(featureProps, prop)
				break
			}
		}
	}

	if len(featureProps) == 0 {
		return nil, fmt.Errorf("no feature properties found for target %s", target.PropertyLabel)
	}

	// Extract training data from knowledge graph
	dataset, err := at.Extractor.ExtractTrainingData(ctx, ontologyID, targetProp, featureProps)
	if err != nil {
		return nil, fmt.Errorf("failed to extract training data: %w", err)
	}

	// Validate dataset
	if err := at.Extractor.ValidateDataset(dataset); err != nil {
		return nil, fmt.Errorf("dataset validation failed: %w", err)
	}

	// Train model
	trainingStart := time.Now()
	config := DefaultTrainingConfig()
	trainer := NewTrainer(config)

	var trainingResult *TrainingResult
	var trainErr error

	if modelType == "regression" {
		yNumeric, ok := dataset.Y.([]float64)
		if !ok {
			return nil, fmt.Errorf("expected numeric target for regression")
		}
		trainingResult, trainErr = trainer.TrainRegression(dataset.X, yNumeric, dataset.FeatureNames)
	} else {
		yCateg, ok := dataset.Y.([]string)
		if !ok {
			return nil, fmt.Errorf("expected categorical target for classification")
		}
		trainingResult, trainErr = trainer.Train(dataset.X, yCateg, dataset.FeatureNames)
	}

	if trainErr != nil {
		return nil, fmt.Errorf("model training failed: %w", trainErr)
	}

	trainingDuration := time.Since(trainingStart)

	// Save model to database
	modelID := fmt.Sprintf("auto_%s_%s_%d", ontologyID, sanitizeModelID(target.PropertyLabel), time.Now().Unix())

	// Serialize model
	modelJSON, err := json.Marshal(trainingResult.Model)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize model: %w", err)
	}

	// Serialize config
	configJSON, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize config: %w", err)
	}

	// Serialize metrics
	metricsJSON, err := json.Marshal(trainingResult)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize metrics: %w", err)
	}

	// Prepare metrics based on model type
	var trainAcc, valAcc, precision, recall, f1 float64
	if modelType == "classification" && trainingResult.ValidateMetrics != nil {
		valAcc = trainingResult.ValidateMetrics.Accuracy
		precision = trainingResult.ValidateMetrics.MacroPrecision
		recall = trainingResult.ValidateMetrics.MacroRecall
		f1 = trainingResult.ValidateMetrics.MacroF1
	}
	if modelType == "classification" && trainingResult.TrainMetrics != nil {
		trainAcc = trainingResult.TrainMetrics.Accuracy
	}

	// Serialize class labels or store empty for regression
	classLabelsJSON := "[]"
	if modelType == "classification" {
		labels := at.Extractor.GetClassLabels(dataset)
		if len(labels) > 0 {
			labelsBytes, _ := json.Marshal(labels)
			classLabelsJSON = string(labelsBytes)
		}
	}

	// Serialize feature columns
	featuresBytes, _ := json.Marshal(dataset.FeatureNames)
	featureColumnsJSON := string(featuresBytes)

	// Save to database
	classifierModel := &storage.ClassifierModel{
		ID:                modelID,
		Name:              fmt.Sprintf("Auto: Predict %s", target.PropertyLabel),
		OntologyID:        ontologyID,
		TargetClass:       target.PropertyLabel,
		Algorithm:         fmt.Sprintf("decision_tree_%s", modelType),
		Hyperparameters:   string(configJSON),
		FeatureColumns:    featureColumnsJSON,
		ClassLabels:       classLabelsJSON,
		TrainAccuracy:     trainAcc,
		ValidateAccuracy:  valAcc,
		PrecisionScore:    precision,
		RecallScore:       recall,
		F1Score:           f1,
		ConfusionMatrix:   "[]",
		ModelArtifactPath: string(modelJSON), // Store serialized model here
		ModelSizeBytes:    int64(len(modelJSON)),
		TrainingRows:      trainingResult.TrainingRows,
		ValidationRows:    trainingResult.ValidationRows,
		FeatureImportance: string(metricsJSON),
		IsActive:          true,
		CreatedAt:         time.Now(),
	}

	err = at.Storage.CreateClassifierModel(ctx, classifierModel)
	if err != nil {
		return nil, fmt.Errorf("failed to save model: %w", err)
	}

	// Build result info
	modelInfo := &TrainedModelInfo{
		ModelID:        modelID,
		TargetProperty: target.PropertyLabel,
		ModelType:      modelType,
		SampleCount:    dataset.SampleCount,
		FeatureCount:   dataset.FeatureCount,
		TrainingTime:   trainingDuration,
		Confidence:     target.Confidence,
		Reasoning:      target.Reasoning,
	}

	if modelType == "regression" && trainingResult.ValidateMetricsReg != nil {
		modelInfo.R2Score = trainingResult.ValidateMetricsReg.R2Score
		modelInfo.RMSE = trainingResult.ValidateMetricsReg.RMSE
	} else if modelType == "classification" && trainingResult.ValidateMetrics != nil {
		modelInfo.Accuracy = trainingResult.ValidateMetrics.Accuracy
	}

	return modelInfo, nil
}

// setupAutomaticMonitoring creates monitoring jobs and rules based on capabilities
func (at *AutoTrainer) setupAutomaticMonitoring(
	ctx context.Context,
	ontologyID string,
	capabilities *MLCapabilities,
) (*MonitoringSetupInfo, error) {
	// Generate monitoring job ID
	jobID := fmt.Sprintf("monitor_%s_%d", ontologyID, time.Now().Unix())

	info := &MonitoringSetupInfo{
		JobID:        jobID,
		MetricsCount: len(capabilities.TimeSeriesMetrics),
		RulesCreated: []string{},
		CronSchedule: "*/15 * * * *", // Every 15 minutes
	}

	// If no metrics to monitor, return early
	if len(capabilities.TimeSeriesMetrics) == 0 {
		return info, nil
	}

	// Collect all metric names (use PropertyLabel as metric name)
	metricNames := []string{}
	for _, metric := range capabilities.TimeSeriesMetrics {
		metricNames = append(metricNames, metric.PropertyLabel)
	}

	// Marshal metrics to JSON
	metricsJSON, err := json.Marshal(metricNames)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metrics: %w", err)
	}

	// Create monitoring rules based on suggestions
	ruleIDs := []string{}
	for _, rule := range capabilities.MonitoringRules {
		ruleID := uuid.New().String()

		// Marshal condition to JSON string
		conditionBytes, err := json.Marshal(rule.Condition)
		if err != nil {
			log.Printf("[AutoTrainer] Warning: Failed to marshal condition for rule %s: %v", rule.RuleName, err)
			continue
		}
		conditionJSON := string(conditionBytes)

		// Create the monitoring rule
		// Note: entityID is empty for ontology-level monitoring
		// metricName is the PropertyLabel
		err = at.Storage.CreateMonitoringRule(
			ctx,
			ruleID,
			ontologyID,
			"",                 // entityID - empty for ontology-level rules
			rule.PropertyLabel, // metricName
			rule.RuleType,
			conditionJSON,
			rule.Severity,
			true, // enabled
			"",   // alert channels (future: email, slack, etc.)
		)
		if err != nil {
			// Log error but continue with other rules
			log.Printf("[AutoTrainer] Warning: Failed to create monitoring rule %s: %v", rule.RuleName, err)
			continue
		}

		ruleIDs = append(ruleIDs, ruleID)
		info.RulesCreated = append(info.RulesCreated, rule.RuleName)
	}

	// Marshal rule IDs to JSON
	rulesJSON := "[]"
	if len(ruleIDs) > 0 {
		rulesBytes, err := json.Marshal(ruleIDs)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal rule IDs: %w", err)
		}
		rulesJSON = string(rulesBytes)
	}

	// Create the monitoring job
	job := &storage.MonitoringJob{
		ID:          jobID,
		Name:        fmt.Sprintf("Auto-Monitor: %s", ontologyID),
		OntologyID:  ontologyID,
		Description: fmt.Sprintf("Automatically monitors %d metrics with %d rules", len(metricNames), len(ruleIDs)),
		CronExpr:    info.CronSchedule,
		Metrics:     string(metricsJSON),
		Rules:       rulesJSON,
		IsEnabled:   true,
	}

	err = at.Storage.CreateMonitoringJob(ctx, job)
	if err != nil {
		return nil, fmt.Errorf("failed to create monitoring job: %w", err)
	}

	log.Printf("[AutoTrainer] Created monitoring job '%s' with %d metrics and %d rules",
		jobID, len(metricNames), len(ruleIDs))

	return info, nil
}

// generateResultSummary creates a human-readable summary
func (at *AutoTrainer) generateResultSummary(result *AutoTrainingResult) string {
	parts := []string{}

	if result.ModelsCreated > 0 {
		parts = append(parts, fmt.Sprintf("trained %d models", result.ModelsCreated))
	}

	if result.ModelsFailed > 0 {
		parts = append(parts, fmt.Sprintf("%d models failed", result.ModelsFailed))
	}

	if result.MonitoringJobsCreated > 0 {
		parts = append(parts, fmt.Sprintf("setup monitoring with %d rules", result.RulesCreated))
	}

	if len(parts) == 0 {
		return "No actions taken"
	}

	return fmt.Sprintf("Successfully %s in %s", strings.Join(parts, ", "), result.TotalDuration.Round(time.Millisecond))
}

// AutoTrainOptions configures what gets trained automatically
type AutoTrainOptions struct {
	EnableRegression     bool    `json:"enable_regression"`
	EnableClassification bool    `json:"enable_classification"`
	EnableMonitoring     bool    `json:"enable_monitoring"`
	MinConfidence        float64 `json:"min_confidence"` // Minimum confidence to auto-train (0.0-1.0)
	ForceAll             bool    `json:"force_all"`      // Train even low-confidence targets
	MaxModels            int     `json:"max_models"`     // Maximum number of models to train
}

// DefaultAutoTrainOptions returns default auto-training options
func DefaultAutoTrainOptions() *AutoTrainOptions {
	return &AutoTrainOptions{
		EnableRegression:     true,
		EnableClassification: true,
		EnableMonitoring:     true,
		MinConfidence:        0.6,
		ForceAll:             false,
		MaxModels:            10,
	}
}

// TrainForGoal trains models based on a natural language goal
func (at *AutoTrainer) TrainForGoal(ctx context.Context, ontologyID, goal string) (*AutoTrainingResult, error) {
	// Analyze ontology
	capabilities, err := at.Analyzer.AnalyzeMLCapabilities(ctx, ontologyID)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze ontology: %w", err)
	}

	// Parse goal to determine what to train
	options := at.parseGoalToOptions(goal, capabilities)

	// Train with parsed options
	return at.TrainFromOntology(ctx, ontologyID, options)
}

// parseGoalToOptions converts a natural language goal to training options
func (at *AutoTrainer) parseGoalToOptions(goal string, capabilities *MLCapabilities) *AutoTrainOptions {
	options := DefaultAutoTrainOptions()
	lowerGoal := strings.ToLower(goal)

	// Parse intent
	if strings.Contains(lowerGoal, "predict") || strings.Contains(lowerGoal, "forecast") {
		// Determine if regression or classification
		if strings.Contains(lowerGoal, "price") || strings.Contains(lowerGoal, "cost") ||
			strings.Contains(lowerGoal, "revenue") || strings.Contains(lowerGoal, "value") {
			options.EnableRegression = true
			options.EnableClassification = false
		} else if strings.Contains(lowerGoal, "category") || strings.Contains(lowerGoal, "class") ||
			strings.Contains(lowerGoal, "type") {
			options.EnableClassification = true
			options.EnableRegression = false
		}
	}

	if strings.Contains(lowerGoal, "monitor") || strings.Contains(lowerGoal, "alert") || strings.Contains(lowerGoal, "watch") {
		options.EnableMonitoring = true
	}

	if strings.Contains(lowerGoal, "all") || strings.Contains(lowerGoal, "everything") {
		options.ForceAll = true
		options.MinConfidence = 0.0
	}

	return options
}

// sanitizeModelID creates a safe model ID from a property label
func sanitizeModelID(label string) string {
	sanitized := strings.ToLower(label)
	sanitized = strings.ReplaceAll(sanitized, " ", "_")
	sanitized = strings.ReplaceAll(sanitized, "-", "_")
	sanitized = strings.ReplaceAll(sanitized, ".", "_")
	sanitized = strings.ReplaceAll(sanitized, "/", "_")
	return sanitized
}

// GetAutoTrainingSuggestions returns detailed suggestions without training
func (at *AutoTrainer) GetAutoTrainingSuggestions(ctx context.Context, ontologyID string) (*MLCapabilities, error) {
	return at.Analyzer.AnalyzeMLCapabilities(ctx, ontologyID)
}
