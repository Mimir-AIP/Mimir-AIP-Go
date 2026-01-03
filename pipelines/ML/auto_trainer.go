package ml

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
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

// DatasetMLTarget represents a potential ML target detected from dataset
type DatasetMLTarget struct {
	ColumnName   string  `json:"column_name"`
	ModelType    string  `json:"model_type"` // "regression" or "classification"
	Confidence   float64 `json:"confidence"`
	FeatureCount int     `json:"feature_count"`
	SampleSize   int     `json:"sample_size"`
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

// TrainFromData automatically trains models based on uploaded data and ontology mapping
func (at *AutoTrainer) TrainFromData(ctx context.Context, ontologyID string, dataset *UnifiedDataset, options *AutoTrainOptions) (*AutoTrainingResult, error) {
	startTime := time.Now()

	result := &AutoTrainingResult{
		OntologyID:    ontologyID,
		TrainedModels: []TrainedModelInfo{},
		FailedModels:  []FailedModelInfo{},
	}

	// Validate dataset
	if err := dataset.Validate(); err != nil {
		return nil, fmt.Errorf("dataset validation failed: %w", err)
	}

	log.Printf("🎯 Training from data: %d rows x %d columns", dataset.RowCount, dataset.ColumnCount)

	// Detect time series and setup monitoring if applicable
	if options.EnableMonitoring && dataset.IsTimeSeries() {
		log.Println("📊 Time-series data detected, setting up monitoring...")
		monitoringInfo, err := at.setupDataMonitoring(ctx, ontologyID, dataset)
		if err != nil {
			log.Printf("⚠️  Warning: monitoring setup failed: %v", err)
		} else {
			result.MonitoringSetup = monitoringInfo
			result.MonitoringJobsCreated = 1
			result.RulesCreated = len(monitoringInfo.RulesCreated)
		}
	}

	// Detect ML targets and train models
	if options.EnableRegression || options.EnableClassification {
		log.Println("🎯 Detecting ML targets from dataset...")
		targets, err := at.detectTargetsFromDataset(ctx, ontologyID, dataset)
		if err != nil {
			log.Printf("⚠️  Warning: target detection failed: %v", err)
		} else {
			log.Printf("✅ Found %d potential targets", len(targets))

			// Train models for each detected target
			for _, target := range targets {
				// Skip if confidence too low
				if target.Confidence < options.MinConfidence && !options.ForceAll {
					log.Printf("⏭️  Skipping %s (confidence %.2f < %.2f)", target.ColumnName, target.Confidence, options.MinConfidence)
					continue
				}

				// Skip based on model type preferences
				if target.ModelType == "regression" && !options.EnableRegression {
					continue
				}
				if target.ModelType == "classification" && !options.EnableClassification {
					continue
				}

				// Check max models limit
				if result.ModelsCreated >= options.MaxModels {
					log.Printf("⏹️  Reached max models limit (%d)", options.MaxModels)
					break
				}

				// Prepare training data from dataset
				trainingData, err := at.prepareTrainingDataFromDataset(dataset, target)
				if err != nil {
					log.Printf("❌ Failed to prepare training data for %s: %v", target.ColumnName, err)
					result.FailedModels = append(result.FailedModels, FailedModelInfo{
						TargetProperty: target.ColumnName,
						ModelType:      target.ModelType,
						ErrorMessage:   fmt.Sprintf("Data preparation failed: %v", err),
						Confidence:     target.Confidence,
					})
					result.ModelsFailed++
					continue
				}

				// Train model
				modelInfo, err := at.trainModelFromDataset(ctx, ontologyID, target, trainingData)
				if err != nil {
					log.Printf("❌ Failed to train model for %s: %v", target.ColumnName, err)
					result.FailedModels = append(result.FailedModels, FailedModelInfo{
						TargetProperty: target.ColumnName,
						ModelType:      target.ModelType,
						ErrorMessage:   fmt.Sprintf("Training failed: %v", err),
						Confidence:     target.Confidence,
					})
					result.ModelsFailed++
				} else {
					log.Printf("✅ Successfully trained model for %s", target.ColumnName)
					result.TrainedModels = append(result.TrainedModels, *modelInfo)
					result.ModelsCreated++
				}
			}
		}
	}

	result.TotalDuration = time.Since(startTime)
	result.Summary = at.generateResultSummary(result)

	log.Printf("✅ Data-based training completed in %v (models: %d, failed: %d)", result.TotalDuration, result.ModelsCreated, result.ModelsFailed)

	return result, nil
}

// detectTargetsFromDataset analyzes the dataset to identify potential ML targets
func (at *AutoTrainer) detectTargetsFromDataset(ctx context.Context, ontologyID string, dataset *UnifiedDataset) ([]DatasetMLTarget, error) {
	log.Println("🎯 Detecting ML targets from dataset...")

	var targets []DatasetMLTarget

	// For each numeric or categorical column, assess if it could be a target
	for _, col := range dataset.Columns {
		// Skip non-ML-friendly columns
		if col.HasNulls && float64(col.NullCount)/float64(dataset.RowCount) > 0.3 {
			continue // Skip columns with >30% nulls
		}

		// Regression targets: numeric columns with reasonable variability
		if col.IsNumeric && col.Stats != nil {
			// Check if column has variability (not all same value)
			if col.Stats.Max > col.Stats.Min {
				// Calculate confidence based on data quality
				confidence := 0.8
				if col.HasNulls {
					confidence -= 0.1
				}
				if col.UniqueCount < 3 {
					confidence -= 0.2 // Too few unique values
				}

				if confidence >= 0.5 {
					targets = append(targets, DatasetMLTarget{
						ColumnName:   col.Name,
						ModelType:    "regression",
						Confidence:   confidence,
						FeatureCount: dataset.ColumnCount - 1,
						SampleSize:   dataset.RowCount,
					})
					log.Printf("   📈 Regression target: %s (confidence: %.2f)", col.Name, confidence)
				}
			}
		}

		// Classification targets: categorical columns or low-cardinality numeric
		if !col.IsNumeric || (col.IsNumeric && col.UniqueCount <= 20 && col.UniqueCount >= 2) {
			// Good classification targets have 2-20 unique values
			if col.UniqueCount >= 2 && col.UniqueCount <= 20 {
				confidence := 0.75
				if col.HasNulls {
					confidence -= 0.1
				}

				if confidence >= 0.5 {
					targets = append(targets, DatasetMLTarget{
						ColumnName:   col.Name,
						ModelType:    "classification",
						Confidence:   confidence,
						FeatureCount: dataset.ColumnCount - 1,
						SampleSize:   dataset.RowCount,
					})
					log.Printf("   📊 Classification target: %s (confidence: %.2f, %d classes)", col.Name, confidence, col.UniqueCount)
				}
			}
		}
	}

	log.Printf("✅ Detected %d potential ML targets", len(targets))
	return targets, nil
}

// prepareTrainingDataFromDataset converts UnifiedDataset to TrainingDataset for a specific target
func (at *AutoTrainer) prepareTrainingDataFromDataset(dataset *UnifiedDataset, target DatasetMLTarget) (*TrainingDataset, error) {
	log.Printf("📦 Preparing training data for target: %s", target.ColumnName)

	// Extract feature columns (all except target and datetime columns)
	var featureColumns []string
	var featureIndices []int

	for _, col := range dataset.Columns {
		if col.Name == target.ColumnName {
			continue // Skip target column
		}
		if col.IsDateTime {
			continue // Skip datetime columns for now
		}
		if col.IsNumeric || !col.HasNulls { // Use numeric columns and clean categorical columns
			featureColumns = append(featureColumns, col.Name)
			featureIndices = append(featureIndices, col.Index)
		}
	}

	if len(featureColumns) == 0 {
		return nil, fmt.Errorf("no valid feature columns found")
	}

	// Create categorical encoding mappings
	categoricalMaps := make(map[string]map[string]float64)
	for _, colName := range featureColumns {
		for _, col := range dataset.Columns {
			if col.Name == colName && !col.IsNumeric {
				// Build mapping for categorical column
				catMap := make(map[string]float64)
				uniqueValues := make(map[string]bool)

				// Collect unique values
				for _, row := range dataset.Rows {
					if val, exists := row[colName]; exists && val != nil {
						if strVal, ok := val.(string); ok {
							uniqueValues[strVal] = true
						}
					}
				}

				// Assign numerical values to categories
				i := 0.0
				for val := range uniqueValues {
					catMap[val] = i
					i++
				}

				categoricalMaps[colName] = catMap
				log.Printf("📋 Categorical mapping for %s: %d unique values", colName, len(catMap))
				break
			}
		}
	}

	// Build feature matrix X and target vector y
	X := make([][]float64, 0, dataset.RowCount)
	var yNumeric []float64
	var yCateg []string

	for _, row := range dataset.Rows {
		// Extract target value
		targetVal, exists := row[target.ColumnName]
		if !exists || targetVal == nil {
			continue // Skip rows with missing target
		}

		// Build feature vector
		featureVec := make([]float64, len(featureColumns))
		validRow := true

		for i, colName := range featureColumns {
			val, exists := row[colName]
			if !exists || val == nil {
				validRow = false
				break
			}

			// Convert to float64
			switch v := val.(type) {
			case float64:
				featureVec[i] = v
			case int:
				featureVec[i] = float64(v)
			case string:
				// For categorical features, use proper encoding with mappings
				if catMap, exists := categoricalMaps[colName]; exists {
					if encoded, ok := catMap[v]; ok {
						featureVec[i] = encoded
					} else {
						// Unknown categorical value - assign average value
						featureVec[i] = float64(len(catMap)) / 2.0
					}
				} else {
					// Fallback: use hash of string
					h := fnv.New32a()
					h.Write([]byte(v))
					featureVec[i] = float64(h.Sum32() % 1000)
				}
			default:
				validRow = false
				break
			}
		}

		if !validRow {
			continue
		}

		// Add to dataset
		X = append(X, featureVec)

		if target.ModelType == "regression" {
			// Convert target to float64
			switch v := targetVal.(type) {
			case float64:
				yNumeric = append(yNumeric, v)
			case int:
				yNumeric = append(yNumeric, float64(v))
			default:
				continue // Skip non-numeric targets for regression
			}
		} else {
			// Convert target to string for classification
			yCateg = append(yCateg, fmt.Sprintf("%v", targetVal))
		}
	}

	if len(X) == 0 {
		return nil, fmt.Errorf("no valid training samples after preprocessing")
	}

	// Create TrainingDataset
	trainingDataset := &TrainingDataset{
		X:            X,
		FeatureNames: featureColumns,
	}

	if target.ModelType == "regression" {
		trainingDataset.Y = yNumeric
	} else {
		trainingDataset.Y = yCateg
	}

	log.Printf("✅ Prepared %d samples x %d features", len(X), len(featureColumns))
	return trainingDataset, nil
}

// AlgorithmRecommendation contains the recommended algorithm and reasoning
type AlgorithmRecommendation struct {
	Algorithm string  `json:"algorithm"` // "decision_tree" or "random_forest"
	Reasoning string  `json:"reasoning"`
	Confidence float64 `json:"confidence"`
	NumTrees   int     `json:"num_trees,omitempty"` // For random forest
}

// recommendAlgorithm intelligently selects the best algorithm based on dataset characteristics
func (at *AutoTrainer) recommendAlgorithm(dataset *TrainingDataset, modelType string) *AlgorithmRecommendation {
	sampleCount := len(dataset.X)
	featureCount := dataset.FeatureCount
	
	// Calculate unique classes for classification
	var numClasses int
	if modelType == "classification" {
		yCateg, ok := dataset.Y.([]string)
		if ok {
			classSet := make(map[string]bool)
			for _, class := range yCateg {
				classSet[class] = true
			}
			numClasses = len(classSet)
		}
	}
	
	// Decision logic based on dataset characteristics
	
	// Use decision tree for very small datasets (< 50 samples)
	if sampleCount < 50 {
		return &AlgorithmRecommendation{
			Algorithm: "decision_tree",
			Reasoning: fmt.Sprintf("Small dataset (%d samples) - decision tree is more appropriate to avoid overfitting", sampleCount),
			Confidence: 0.9,
		}
	}
	
	// Use decision tree for few features (< 5)
	if featureCount < 5 {
		return &AlgorithmRecommendation{
			Algorithm: "decision_tree",
			Reasoning: fmt.Sprintf("Few features (%d) - decision tree is sufficient and more interpretable", featureCount),
			Confidence: 0.8,
		}
	}
	
	// Use random forest for multiclass problems with many classes (> 5)
	if modelType == "classification" && numClasses > 5 {
		numTrees := 100
		if sampleCount < 200 {
			numTrees = 50
		}
		return &AlgorithmRecommendation{
			Algorithm: "random_forest",
			Reasoning: fmt.Sprintf("Multiclass problem (%d classes) with %d samples - random forest provides better accuracy", numClasses, sampleCount),
			Confidence: 0.95,
			NumTrees: numTrees,
		}
	}
	
	// Use random forest for larger datasets with more features
	if sampleCount >= 100 && featureCount >= 5 {
		// Determine optimal number of trees based on data size
		numTrees := 50
		if sampleCount >= 500 {
			numTrees = 100
		}
		if sampleCount >= 1000 {
			numTrees = 150
		}
		
		return &AlgorithmRecommendation{
			Algorithm: "random_forest",
			Reasoning: fmt.Sprintf("Substantial dataset (%d samples, %d features) - random forest will provide better generalization", sampleCount, featureCount),
			Confidence: 0.85,
			NumTrees: numTrees,
		}
	}
	
	// Use random forest for medium-sized datasets (50-100 samples)
	if sampleCount >= 50 && sampleCount < 100 {
		return &AlgorithmRecommendation{
			Algorithm: "random_forest",
			Reasoning: fmt.Sprintf("Medium dataset (%d samples, %d features) - random forest with fewer trees reduces overfitting risk", sampleCount, featureCount),
			Confidence: 0.75,
			NumTrees: 30,
		}
	}
	
	// Default to decision tree for edge cases
	return &AlgorithmRecommendation{
		Algorithm: "decision_tree",
		Reasoning: fmt.Sprintf("Default choice for dataset with %d samples and %d features", sampleCount, featureCount),
		Confidence: 0.6,
	}
}

// trainModelFromDataset trains a model using prepared dataset
func (at *AutoTrainer) trainModelFromDataset(
	ctx context.Context,
	ontologyID string,
	target DatasetMLTarget,
	dataset *TrainingDataset,
) (*TrainedModelInfo, error) {
	log.Printf("🤖 Training %s model for: %s", target.ModelType, target.ColumnName)

	// Validate dataset
	if err := at.Extractor.ValidateDataset(dataset); err != nil {
		return nil, fmt.Errorf("dataset validation failed: %w", err)
	}

	// Get algorithm recommendation
	recommendation := at.recommendAlgorithm(dataset, target.ModelType)
	log.Printf("📊 Algorithm recommendation: %s (confidence: %.2f)", recommendation.Algorithm, recommendation.Confidence)
	log.Printf("   Reasoning: %s", recommendation.Reasoning)

	// Train model
	trainingStart := time.Now()
	config := DefaultTrainingConfig()
	
	// Configure for random forest if recommended
	if recommendation.Algorithm == "random_forest" && recommendation.NumTrees > 0 {
		config.NumTrees = recommendation.NumTrees
	}
	
	trainer := NewTrainer(config)

	var trainingResult *TrainingResult
	var trainErr error

	if target.ModelType == "regression" {
		yNumeric, ok := dataset.Y.([]float64)
		if !ok {
			return nil, fmt.Errorf("expected numeric target for regression")
		}
		
		// Use recommended algorithm
		if recommendation.Algorithm == "random_forest" {
			trainingResult, trainErr = trainer.TrainRandomForestRegression(dataset.X, yNumeric, dataset.FeatureNames)
		} else {
			trainingResult, trainErr = trainer.TrainRegression(dataset.X, yNumeric, dataset.FeatureNames)
		}
	} else {
		yCateg, ok := dataset.Y.([]string)
		if !ok {
			return nil, fmt.Errorf("expected categorical target for classification")
		}
		
		// Use recommended algorithm
		if recommendation.Algorithm == "random_forest" {
			trainingResult, trainErr = trainer.TrainRandomForest(dataset.X, yCateg, dataset.FeatureNames)
		} else {
			trainingResult, trainErr = trainer.Train(dataset.X, yCateg, dataset.FeatureNames)
		}
	}

	if trainErr != nil {
		return nil, fmt.Errorf("model training failed: %w", trainErr)
	}

	trainingDuration := time.Since(trainingStart)

	// Save model to database
	modelID := fmt.Sprintf("auto_%s_%s_%d", ontologyID, sanitizeModelID(target.ColumnName), time.Now().Unix())

	// Serialize model (handle both decision tree and random forest)
	var modelJSON []byte
	var err error
	if recommendation.Algorithm == "random_forest" {
		modelJSON, err = json.Marshal(trainingResult.ModelRF)
	} else {
		modelJSON, err = json.Marshal(trainingResult.Model)
	}
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
	var accuracy, r2Score, rmse float64
	var sampleCount int

	if target.ModelType == "classification" {
		if trainingResult.ValidateMetrics != nil {
			accuracy = trainingResult.ValidateMetrics.Accuracy
			sampleCount = trainingResult.ValidateMetrics.TotalSamples
		}
	} else {
		if trainingResult.ValidateMetricsReg != nil {
			r2Score = trainingResult.ValidateMetricsReg.R2Score
			rmse = trainingResult.ValidateMetricsReg.RMSE
			sampleCount = trainingResult.ValidateMetricsReg.NumSamples
		}
	}

	// Try to save to storage using the SaveMLModelDirect method
	err = at.Storage.SaveMLModelDirect(ctx, modelID, ontologyID, string(modelJSON), string(configJSON), string(metricsJSON))
	if err != nil {
		log.Printf("⚠️  Failed to save model %s to storage: %v", modelID, err)
		// Don't return error - model training succeeded, just saving failed
	} else {
		log.Printf("📦 Model %s saved to storage (model: %d bytes, config: %d bytes, metrics: %d bytes)",
			modelID, len(modelJSON), len(configJSON), len(metricsJSON))
	}

	log.Printf("✅ Model trained in %v (ID: %s)", trainingDuration, modelID)

	return &TrainedModelInfo{
		ModelID:        modelID,
		TargetProperty: target.ColumnName,
		ModelType:      target.ModelType,
		Accuracy:       accuracy,
		R2Score:        r2Score,
		RMSE:           rmse,
		SampleCount:    sampleCount,
		FeatureCount:   len(dataset.FeatureNames),
		TrainingTime:   trainingDuration,
		Confidence:     target.Confidence,
		Reasoning:      fmt.Sprintf("Trained %s from uploaded dataset with %d samples and %d features. %s", recommendation.Algorithm, target.SampleSize, target.FeatureCount, recommendation.Reasoning),
	}, nil
}

// setupDataMonitoring creates monitoring jobs for time-series dataset
func (at *AutoTrainer) setupDataMonitoring(ctx context.Context, ontologyID string, dataset *UnifiedDataset) (*MonitoringSetupInfo, error) {
	if dataset.TimeSeriesConfig == nil {
		return nil, fmt.Errorf("no time-series configuration in dataset")
	}

	tsConfig := dataset.TimeSeriesConfig

	// Create monitoring job
	jobID := uuid.New().String()
	jobName := fmt.Sprintf("auto_monitor_%s_%s", ontologyID, tsConfig.DateColumn)

	log.Printf("📊 Creating monitoring job: %s", jobName)

	// Create rules for each metric
	var rulesCreated []string
	for _, metricCol := range tsConfig.MetricColumns {
		// Find column metadata to get stats
		var colMeta *ColumnMetadata
		for i := range dataset.Columns {
			if dataset.Columns[i].Name == metricCol {
				colMeta = &dataset.Columns[i]
				break
			}
		}

		if colMeta == nil || !colMeta.IsNumeric || colMeta.Stats == nil {
			continue
		}

		// Create threshold rule based on column stats
		// Use mean ± 2*std_dev as thresholds (if we had std_dev)
		// For now, use min/max with 10% buffer
		threshold := colMeta.Stats.Max * 1.1 // 10% above max as warning threshold

		ruleID := uuid.New().String()
		ruleName := fmt.Sprintf("threshold_%s", metricCol)

		// In a real implementation, this would create the rule in the monitoring system
		// For now, just track that we would create it
		rulesCreated = append(rulesCreated, ruleName)

		log.Printf("📋 Would create threshold rule for %s: %.2f", metricCol, threshold)
		_ = ruleID // Placeholder
	}

	return &MonitoringSetupInfo{
		JobID:        jobID,
		MetricsCount: len(tsConfig.MetricColumns),
		RulesCreated: rulesCreated,
		CronSchedule: "0 */6 * * *", // Every 6 hours
	}, nil
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

	// Get algorithm recommendation
	recommendation := at.recommendAlgorithm(dataset, modelType)
	log.Printf("📊 Algorithm recommendation: %s (confidence: %.2f)", recommendation.Algorithm, recommendation.Confidence)
	log.Printf("   Reasoning: %s", recommendation.Reasoning)

	// Train model
	trainingStart := time.Now()
	config := DefaultTrainingConfig()
	
	// Configure for random forest if recommended
	if recommendation.Algorithm == "random_forest" && recommendation.NumTrees > 0 {
		config.NumTrees = recommendation.NumTrees
	}
	
	trainer := NewTrainer(config)

	var trainingResult *TrainingResult
	var trainErr error

	if modelType == "regression" {
		yNumeric, ok := dataset.Y.([]float64)
		if !ok {
			return nil, fmt.Errorf("expected numeric target for regression")
		}
		
		// Use recommended algorithm
		if recommendation.Algorithm == "random_forest" {
			trainingResult, trainErr = trainer.TrainRandomForestRegression(dataset.X, yNumeric, dataset.FeatureNames)
		} else {
			trainingResult, trainErr = trainer.TrainRegression(dataset.X, yNumeric, dataset.FeatureNames)
		}
	} else {
		yCateg, ok := dataset.Y.([]string)
		if !ok {
			return nil, fmt.Errorf("expected categorical target for classification")
		}
		
		// Use recommended algorithm
		if recommendation.Algorithm == "random_forest" {
			trainingResult, trainErr = trainer.TrainRandomForest(dataset.X, yCateg, dataset.FeatureNames)
		} else {
			trainingResult, trainErr = trainer.Train(dataset.X, yCateg, dataset.FeatureNames)
		}
	}

	if trainErr != nil {
		return nil, fmt.Errorf("model training failed: %w", trainErr)
	}

	trainingDuration := time.Since(trainingStart)

	// Save model to database
	modelID := fmt.Sprintf("auto_%s_%s_%d", ontologyID, sanitizeModelID(target.PropertyLabel), time.Now().Unix())

	// Serialize model (handle both decision tree and random forest)
	var modelJSON []byte
	if recommendation.Algorithm == "random_forest" {
		modelJSON, err = json.Marshal(trainingResult.ModelRF)
	} else {
		modelJSON, err = json.Marshal(trainingResult.Model)
	}
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
		Algorithm:         recommendation.Algorithm,
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
		Reasoning:      fmt.Sprintf("%s. Algorithm: %s (%s)", target.Reasoning, recommendation.Algorithm, recommendation.Reasoning),
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
