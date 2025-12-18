package ml

import (
	"fmt"
	"math/rand"
	"sort"
	"time"
)

// TrainingConfig holds configuration for training a classifier
type TrainingConfig struct {
	TrainTestSplit  float64 `json:"train_test_split"`  // e.g., 0.8 for 80/20 split
	MaxDepth        int     `json:"max_depth"`         // Maximum tree depth
	MinSamplesSplit int     `json:"min_samples_split"` // Minimum samples to split a node
	MinSamplesLeaf  int     `json:"min_samples_leaf"`  // Minimum samples per leaf
	RandomSeed      int64   `json:"random_seed"`       // For reproducibility
	Shuffle         bool    `json:"shuffle"`           // Shuffle data before split
	Stratify        bool    `json:"stratify"`          // Stratified split (maintain class distribution)
}

// DefaultTrainingConfig returns a training config with sensible defaults
func DefaultTrainingConfig() *TrainingConfig {
	return &TrainingConfig{
		TrainTestSplit:  0.8,
		MaxDepth:        10,
		MinSamplesSplit: 2,
		MinSamplesLeaf:  1,
		RandomSeed:      time.Now().UnixNano(),
		Shuffle:         true,
		Stratify:        true,
	}
}

// TrainingResult holds the results of a training run
type TrainingResult struct {
	Model              *DecisionTreeClassifier `json:"-"` // Don't serialize model directly
	TrainMetrics       *EvaluationMetrics      `json:"train_metrics,omitempty"`
	ValidateMetrics    *EvaluationMetrics      `json:"validate_metrics,omitempty"`
	TrainMetricsReg    *RegressionMetrics      `json:"train_metrics_reg,omitempty"`
	ValidateMetricsReg *RegressionMetrics      `json:"validate_metrics_reg,omitempty"`
	TrainingRows       int                     `json:"training_rows"`
	ValidationRows     int                     `json:"validation_rows"`
	TrainingDuration   time.Duration           `json:"training_duration"`
	FeatureImportance  map[string]float64      `json:"feature_importance"`
	ModelInfo          map[string]interface{}  `json:"model_info"`
	ModelType          string                  `json:"model_type"` // "classification" or "regression"
}

// Trainer orchestrates the training process
type Trainer struct {
	Config *TrainingConfig
	Rand   *rand.Rand
}

// NewTrainer creates a new trainer with the given configuration
func NewTrainer(config *TrainingConfig) *Trainer {
	if config == nil {
		config = DefaultTrainingConfig()
	}
	return &Trainer{
		Config: config,
		Rand:   rand.New(rand.NewSource(config.RandomSeed)),
	}
}

// Train trains a classifier on the provided data
func (t *Trainer) Train(X [][]float64, y []string, featureNames []string) (*TrainingResult, error) {
	if len(X) == 0 || len(y) == 0 {
		return nil, fmt.Errorf("empty training data")
	}
	if len(X) != len(y) {
		return nil, fmt.Errorf("X and y must have same number of samples")
	}
	if len(featureNames) != len(X[0]) {
		return nil, fmt.Errorf("feature names must match number of features")
	}

	startTime := time.Now()

	// Split data into train and validation sets
	trainX, trainY, valX, valY, err := t.TrainTestSplit(X, y)
	if err != nil {
		return nil, fmt.Errorf("failed to split data: %w", err)
	}

	// Train the classifier
	classifier := NewDecisionTreeClassifier(
		t.Config.MaxDepth,
		t.Config.MinSamplesSplit,
		t.Config.MinSamplesLeaf,
	)

	if err := classifier.Train(trainX, trainY, featureNames); err != nil {
		return nil, fmt.Errorf("failed to train classifier: %w", err)
	}

	trainingDuration := time.Since(startTime)

	// Evaluate on training set
	trainMetrics, err := Evaluate(classifier, trainX, trainY)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate on training set: %w", err)
	}

	// Evaluate on validation set
	valMetrics, err := Evaluate(classifier, valX, valY)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate on validation set: %w", err)
	}

	// Get feature importance
	featureImportance := classifier.GetFeatureImportance()

	// Get model info
	modelInfo := classifier.GetModelInfo()

	result := &TrainingResult{
		Model:             classifier,
		TrainMetrics:      trainMetrics,
		ValidateMetrics:   valMetrics,
		TrainingRows:      len(trainX),
		ValidationRows:    len(valX),
		TrainingDuration:  trainingDuration,
		FeatureImportance: featureImportance,
		ModelInfo:         modelInfo,
		ModelType:         "classification",
	}

	return result, nil
}

// TrainRegression trains a regression model on the provided data
func (t *Trainer) TrainRegression(X [][]float64, y []float64, featureNames []string) (*TrainingResult, error) {
	if len(X) == 0 || len(y) == 0 {
		return nil, fmt.Errorf("empty training data")
	}
	if len(X) != len(y) {
		return nil, fmt.Errorf("X and y must have same number of samples")
	}
	if len(featureNames) != len(X[0]) {
		return nil, fmt.Errorf("feature names must match number of features")
	}

	startTime := time.Now()

	// Split data into train and validation sets
	trainX, trainY, valX, valY, err := t.TrainTestSplitRegression(X, y)
	if err != nil {
		return nil, fmt.Errorf("failed to split data: %w", err)
	}

	// Train the regressor
	regressor := NewDecisionTreeClassifier(
		t.Config.MaxDepth,
		t.Config.MinSamplesSplit,
		t.Config.MinSamplesLeaf,
	)

	if err := regressor.TrainRegression(trainX, trainY, featureNames); err != nil {
		return nil, fmt.Errorf("failed to train regressor: %w", err)
	}

	trainingDuration := time.Since(startTime)

	// Evaluate on training set
	trainMetrics, err := EvaluateRegression(regressor, trainX, trainY)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate on training set: %w", err)
	}

	// Evaluate on validation set
	valMetrics, err := EvaluateRegression(regressor, valX, valY)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate on validation set: %w", err)
	}

	// Get feature importance
	featureImportance := regressor.GetFeatureImportance()

	// Get model info
	modelInfo := regressor.GetModelInfo()

	result := &TrainingResult{
		Model:              regressor,
		TrainMetricsReg:    trainMetrics,
		ValidateMetricsReg: valMetrics,
		TrainingRows:       len(trainX),
		ValidationRows:     len(valX),
		TrainingDuration:   trainingDuration,
		FeatureImportance:  featureImportance,
		ModelInfo:          modelInfo,
		ModelType:          "regression",
	}

	return result, nil
}

// TrainTestSplit splits data into training and validation sets
func (t *Trainer) TrainTestSplit(X [][]float64, y []string) ([][]float64, []string, [][]float64, []string, error) {
	n := len(X)
	if n == 0 {
		return nil, nil, nil, nil, fmt.Errorf("empty data")
	}

	// Create indices
	indices := make([]int, n)
	for i := range indices {
		indices[i] = i
	}

	// Shuffle if requested
	if t.Config.Shuffle {
		t.Rand.Shuffle(n, func(i, j int) {
			indices[i], indices[j] = indices[j], indices[i]
		})
	}

	// Stratified split if requested
	if t.Config.Stratify {
		return t.stratifiedSplit(X, y, indices)
	}

	// Regular split
	splitIdx := int(float64(n) * t.Config.TrainTestSplit)
	trainIndices := indices[:splitIdx]
	valIndices := indices[splitIdx:]

	trainX, trainY := selectByIndices(X, y, trainIndices)
	valX, valY := selectByIndices(X, y, valIndices)

	return trainX, trainY, valX, valY, nil
}

// TrainTestSplitRegression splits regression data into training and validation sets
func (t *Trainer) TrainTestSplitRegression(X [][]float64, y []float64) ([][]float64, []float64, [][]float64, []float64, error) {
	n := len(X)
	if n == 0 {
		return nil, nil, nil, nil, fmt.Errorf("empty data")
	}

	// Create indices
	indices := make([]int, n)
	for i := range indices {
		indices[i] = i
	}

	// Shuffle if requested
	if t.Config.Shuffle {
		t.Rand.Shuffle(n, func(i, j int) {
			indices[i], indices[j] = indices[j], indices[i]
		})
	}

	// Regular split (no stratification for regression)
	splitIdx := int(float64(n) * t.Config.TrainTestSplit)
	trainIndices := indices[:splitIdx]
	valIndices := indices[splitIdx:]

	trainX, trainY := selectByIndicesRegression(X, y, trainIndices)
	valX, valY := selectByIndicesRegression(X, y, valIndices)

	return trainX, trainY, valX, valY, nil
}

// stratifiedSplit performs stratified train-test split (maintains class distribution)
func (t *Trainer) stratifiedSplit(X [][]float64, y []string, indices []int) ([][]float64, []string, [][]float64, []string, error) {
	// Group indices by class
	classSamples := make(map[string][]int)
	for _, idx := range indices {
		class := y[idx]
		classSamples[class] = append(classSamples[class], idx)
	}

	var trainIndices, valIndices []int

	// Split each class separately
	for _, samples := range classSamples {
		splitIdx := int(float64(len(samples)) * t.Config.TrainTestSplit)
		if splitIdx == 0 && len(samples) > 0 {
			splitIdx = 1 // Ensure at least one training sample per class
		}
		if splitIdx >= len(samples) {
			splitIdx = len(samples) - 1 // Ensure at least one validation sample
		}

		trainIndices = append(trainIndices, samples[:splitIdx]...)
		valIndices = append(valIndices, samples[splitIdx:]...)
	}

	// Shuffle train and val indices separately
	t.Rand.Shuffle(len(trainIndices), func(i, j int) {
		trainIndices[i], trainIndices[j] = trainIndices[j], trainIndices[i]
	})
	t.Rand.Shuffle(len(valIndices), func(i, j int) {
		valIndices[i], valIndices[j] = valIndices[j], valIndices[i]
	})

	trainX, trainY := selectByIndices(X, y, trainIndices)
	valX, valY := selectByIndices(X, y, valIndices)

	return trainX, trainY, valX, valY, nil
}

// selectByIndices selects samples from X and y using indices
func selectByIndices(X [][]float64, y []string, indices []int) ([][]float64, []string) {
	selectedX := make([][]float64, len(indices))
	selectedY := make([]string, len(indices))

	for i, idx := range indices {
		selectedX[i] = X[idx]
		selectedY[i] = y[idx]
	}

	return selectedX, selectedY
}

// selectByIndicesRegression selects samples from X and y using indices (regression version)
func selectByIndicesRegression(X [][]float64, y []float64, indices []int) ([][]float64, []float64) {
	selectedX := make([][]float64, len(indices))
	selectedY := make([]float64, len(indices))

	for i, idx := range indices {
		selectedX[i] = X[idx]
		selectedY[i] = y[idx]
	}

	return selectedX, selectedY
}

// PrepareDataFromCSV prepares training data from CSV-like data
// data: rows of data (including header as first row)
// targetColumn: name of the target column to predict
// Returns: X (feature matrix), y (labels), featureNames, error
func PrepareDataFromCSV(data [][]string, targetColumn string) ([][]float64, []string, []string, error) {
	if len(data) < 2 {
		return nil, nil, nil, fmt.Errorf("data must have at least header and one row")
	}

	header := data[0]
	rows := data[1:]

	// Find target column index
	targetIdx := -1
	for i, col := range header {
		if col == targetColumn {
			targetIdx = i
			break
		}
	}
	if targetIdx == -1 {
		return nil, nil, nil, fmt.Errorf("target column '%s' not found in header", targetColumn)
	}

	// Get feature column indices (everything except target)
	var featureIndices []int
	var featureNames []string
	for i, col := range header {
		if i != targetIdx {
			featureIndices = append(featureIndices, i)
			featureNames = append(featureNames, col)
		}
	}

	// Extract labels
	y := make([]string, len(rows))
	for i, row := range rows {
		if targetIdx >= len(row) {
			return nil, nil, nil, fmt.Errorf("row %d: target column index out of bounds", i)
		}
		y[i] = row[targetIdx]
	}

	// Extract and convert features to float64
	X := make([][]float64, len(rows))
	for i, row := range rows {
		X[i] = make([]float64, len(featureIndices))
		for j, idx := range featureIndices {
			if idx >= len(row) {
				return nil, nil, nil, fmt.Errorf("row %d: feature column index %d out of bounds", i, idx)
			}
			val, err := convertToFloat(row[idx])
			if err != nil {
				return nil, nil, nil, fmt.Errorf("row %d, column '%s': %w", i, header[idx], err)
			}
			X[i][j] = val
		}
	}

	return X, y, featureNames, nil
}

// PrepareRegressionDataFromCSV prepares regression training data from CSV-like data
// data: rows of data (including header as first row)
// targetColumn: name of the target column to predict (must be numeric)
// Returns: X (feature matrix), y (numeric targets), featureNames, error
func PrepareRegressionDataFromCSV(data [][]string, targetColumn string) ([][]float64, []float64, []string, error) {
	if len(data) < 2 {
		return nil, nil, nil, fmt.Errorf("data must have at least header and one row")
	}

	header := data[0]
	rows := data[1:]

	// Find target column index
	targetIdx := -1
	for i, col := range header {
		if col == targetColumn {
			targetIdx = i
			break
		}
	}
	if targetIdx == -1 {
		return nil, nil, nil, fmt.Errorf("target column '%s' not found in header", targetColumn)
	}

	// Get feature column indices (everything except target)
	var featureIndices []int
	var featureNames []string
	for i, col := range header {
		if i != targetIdx {
			featureIndices = append(featureIndices, i)
			featureNames = append(featureNames, col)
		}
	}

	// Extract numeric targets
	y := make([]float64, len(rows))
	for i, row := range rows {
		if targetIdx >= len(row) {
			return nil, nil, nil, fmt.Errorf("row %d: target column index out of bounds", i)
		}
		val, err := fmt.Sscanf(row[targetIdx], "%f", &y[i])
		if err != nil || val == 0 {
			return nil, nil, nil, fmt.Errorf("row %d: target column '%s' contains non-numeric value '%s'", i, targetColumn, row[targetIdx])
		}
	}

	// Extract and convert features to float64
	X := make([][]float64, len(rows))
	for i, row := range rows {
		X[i] = make([]float64, len(featureIndices))
		for j, idx := range featureIndices {
			if idx >= len(row) {
				return nil, nil, nil, fmt.Errorf("row %d: feature column index %d out of bounds", i, idx)
			}
			val, err := convertToFloat(row[idx])
			if err != nil {
				return nil, nil, nil, fmt.Errorf("row %d, column '%s': %w", i, header[idx], err)
			}
			X[i][j] = val
		}
	}

	return X, y, featureNames, nil
}

// convertToFloat converts a string value to float64
// Handles numeric values and encodes categorical values
func convertToFloat(s string) (float64, error) {
	// Try parsing as float
	var val float64
	_, err := fmt.Sscanf(s, "%f", &val)
	if err == nil {
		return val, nil
	}

	// If not numeric, encode categorically using hash (simple approach)
	// For production, you'd want a proper encoder that's consistent across predictions
	return float64(hashString(s)), nil
}

// hashString creates a simple hash of a string for categorical encoding
func hashString(s string) int {
	hash := 0
	for i, c := range s {
		hash = hash*31 + int(c) + i
	}
	if hash < 0 {
		hash = -hash
	}
	return hash % 10000 // Keep it reasonable
}

// FeatureSelector selects the most important features
type FeatureSelector struct {
	NumFeatures   int  // Number of features to select
	UseImportance bool // Use feature importance for selection
}

// SelectFeatures selects top features based on importance
func (fs *FeatureSelector) SelectFeatures(X [][]float64, y []string, featureNames []string) ([][]float64, []string, error) {
	if fs.NumFeatures >= len(featureNames) {
		return X, featureNames, nil // Return all features
	}

	// Train a quick model to get feature importance
	config := DefaultTrainingConfig()
	config.MaxDepth = 5 // Shallow tree for speed
	trainer := NewTrainer(config)
	result, err := trainer.Train(X, y, featureNames)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to train for feature selection: %w", err)
	}

	// Sort features by importance
	type featureImportance struct {
		name       string
		importance float64
		index      int
	}
	var importances []featureImportance
	for i, name := range featureNames {
		importances = append(importances, featureImportance{
			name:       name,
			importance: result.FeatureImportance[name],
			index:      i,
		})
	}

	sort.Slice(importances, func(i, j int) bool {
		return importances[i].importance > importances[j].importance
	})

	// Select top N features
	selectedIndices := make([]int, fs.NumFeatures)
	selectedNames := make([]string, fs.NumFeatures)
	for i := 0; i < fs.NumFeatures; i++ {
		selectedIndices[i] = importances[i].index
		selectedNames[i] = importances[i].name
	}

	// Extract selected features from X
	selectedX := make([][]float64, len(X))
	for i := range X {
		selectedX[i] = make([]float64, fs.NumFeatures)
		for j, idx := range selectedIndices {
			selectedX[i][j] = X[i][idx]
		}
	}

	return selectedX, selectedNames, nil
}

// HyperparameterTuner tunes hyperparameters using grid search
type HyperparameterTuner struct {
	MaxDepths        []int
	MinSamplesSplits []int
	MinSamplesLeafs  []int
	CrossValidationK int
}

// TuneResult holds hyperparameter tuning results
type TuneResult struct {
	BestConfig   *TrainingConfig   `json:"best_config"`
	BestAccuracy float64           `json:"best_accuracy"`
	AllResults   []TuneResultEntry `json:"all_results"`
}

// TuneResultEntry holds a single hyperparameter combination result
type TuneResultEntry struct {
	MaxDepth        int     `json:"max_depth"`
	MinSamplesSplit int     `json:"min_samples_split"`
	MinSamplesLeaf  int     `json:"min_samples_leaf"`
	Accuracy        float64 `json:"accuracy"`
}

// Tune performs grid search to find best hyperparameters
func (ht *HyperparameterTuner) Tune(X [][]float64, y []string, featureNames []string) (*TuneResult, error) {
	bestAccuracy := 0.0
	var bestConfig *TrainingConfig
	var allResults []TuneResultEntry

	for _, maxDepth := range ht.MaxDepths {
		for _, minSamplesSplit := range ht.MinSamplesSplits {
			for _, minSamplesLeaf := range ht.MinSamplesLeafs {
				// Run cross-validation with these hyperparameters
				cvResults, err := CrossValidate(X, y, featureNames, ht.CrossValidationK, maxDepth, minSamplesSplit, minSamplesLeaf)
				if err != nil {
					continue // Skip this combination
				}

				accuracy := cvResults.MeanAccuracy

				allResults = append(allResults, TuneResultEntry{
					MaxDepth:        maxDepth,
					MinSamplesSplit: minSamplesSplit,
					MinSamplesLeaf:  minSamplesLeaf,
					Accuracy:        accuracy,
				})

				if accuracy > bestAccuracy {
					bestAccuracy = accuracy
					bestConfig = &TrainingConfig{
						TrainTestSplit:  0.8,
						MaxDepth:        maxDepth,
						MinSamplesSplit: minSamplesSplit,
						MinSamplesLeaf:  minSamplesLeaf,
						RandomSeed:      time.Now().UnixNano(),
						Shuffle:         true,
						Stratify:        true,
					}
				}
			}
		}
	}

	if bestConfig == nil {
		return nil, fmt.Errorf("no valid hyperparameter combination found")
	}

	return &TuneResult{
		BestConfig:   bestConfig,
		BestAccuracy: bestAccuracy,
		AllResults:   allResults,
	}, nil
}
