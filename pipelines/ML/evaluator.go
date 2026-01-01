package ml

import (
	"fmt"
	"math"
	"sort"
)

// EvaluationMetrics holds various classification metrics
type EvaluationMetrics struct {
	Accuracy           float64                   `json:"accuracy"`
	Precision          map[string]float64        `json:"precision"`        // Per-class
	Recall             map[string]float64        `json:"recall"`           // Per-class
	F1Score            map[string]float64        `json:"f1_score"`         // Per-class
	MacroPrecision     float64                   `json:"macro_precision"`  // Average across classes
	MacroRecall        float64                   `json:"macro_recall"`     // Average across classes
	MacroF1            float64                   `json:"macro_f1"`         // Average F1
	ConfusionMatrix    map[string]map[string]int `json:"confusion_matrix"` // Actual -> Predicted -> Count
	Support            map[string]int            `json:"support"`          // Number of samples per class
	TotalSamples       int                       `json:"total_samples"`
	CorrectPredictions int                       `json:"correct_predictions"`
}

// ConfusionMatrix represents a confusion matrix
type ConfusionMatrix struct {
	Matrix  map[string]map[string]int `json:"matrix"`  // Actual -> Predicted -> Count
	Classes []string                  `json:"classes"` // Ordered class labels
}

// Evaluate evaluates a classifier on test data
func Evaluate(classifier *DecisionTreeClassifier, X [][]float64, yTrue []string) (*EvaluationMetrics, error) {
	if len(X) == 0 || len(yTrue) == 0 {
		return nil, fmt.Errorf("empty test data")
	}
	if len(X) != len(yTrue) {
		return nil, fmt.Errorf("X and yTrue must have same length")
	}

	// Make predictions
	yPred := make([]string, len(X))
	for i, x := range X {
		pred, _, err := classifier.Predict(x)
		if err != nil {
			return nil, fmt.Errorf("prediction failed at index %d: %w", i, err)
		}
		yPred[i] = pred
	}

	// Calculate metrics
	return CalculateMetrics(yTrue, yPred, classifier.Classes)
}

// CalculateMetrics calculates all evaluation metrics
func CalculateMetrics(yTrue, yPred []string, classes []string) (*EvaluationMetrics, error) {
	if len(yTrue) != len(yPred) {
		return nil, fmt.Errorf("yTrue and yPred must have same length")
	}

	metrics := &EvaluationMetrics{
		Precision:       make(map[string]float64),
		Recall:          make(map[string]float64),
		F1Score:         make(map[string]float64),
		Support:         make(map[string]int),
		ConfusionMatrix: make(map[string]map[string]int),
		TotalSamples:    len(yTrue),
	}

	// Initialize confusion matrix
	for _, actual := range classes {
		metrics.ConfusionMatrix[actual] = make(map[string]int)
		for _, pred := range classes {
			metrics.ConfusionMatrix[actual][pred] = 0
		}
	}

	// Populate confusion matrix and count support
	for i := range yTrue {
		actual := yTrue[i]
		predicted := yPred[i]

		if metrics.ConfusionMatrix[actual] == nil {
			metrics.ConfusionMatrix[actual] = make(map[string]int)
		}
		metrics.ConfusionMatrix[actual][predicted]++
		metrics.Support[actual]++

		if actual == predicted {
			metrics.CorrectPredictions++
		}
	}

	// Calculate accuracy
	metrics.Accuracy = float64(metrics.CorrectPredictions) / float64(metrics.TotalSamples)

	// Calculate per-class metrics
	for _, class := range classes {
		tp := metrics.ConfusionMatrix[class][class] // True positives

		// False negatives: predicted other classes when actual was this class
		fn := 0
		for _, predClass := range classes {
			if predClass != class {
				fn += metrics.ConfusionMatrix[class][predClass]
			}
		}

		// False positives: predicted this class when actual was other class
		fp := 0
		for _, actualClass := range classes {
			if actualClass != class {
				fp += metrics.ConfusionMatrix[actualClass][class]
			}
		}

		// Precision = TP / (TP + FP)
		if tp+fp > 0 {
			metrics.Precision[class] = float64(tp) / float64(tp+fp)
		} else {
			metrics.Precision[class] = 0.0
		}

		// Recall = TP / (TP + FN)
		if tp+fn > 0 {
			metrics.Recall[class] = float64(tp) / float64(tp+fn)
		} else {
			metrics.Recall[class] = 0.0
		}

		// F1 Score = 2 * (Precision * Recall) / (Precision + Recall)
		prec := metrics.Precision[class]
		rec := metrics.Recall[class]
		if prec+rec > 0 {
			metrics.F1Score[class] = 2 * (prec * rec) / (prec + rec)
		} else {
			metrics.F1Score[class] = 0.0
		}
	}

	// Calculate macro averages
	sumPrecision := 0.0
	sumRecall := 0.0
	sumF1 := 0.0
	numClasses := len(classes)

	for _, class := range classes {
		sumPrecision += metrics.Precision[class]
		sumRecall += metrics.Recall[class]
		sumF1 += metrics.F1Score[class]
	}

	if numClasses > 0 {
		metrics.MacroPrecision = sumPrecision / float64(numClasses)
		metrics.MacroRecall = sumRecall / float64(numClasses)
		metrics.MacroF1 = sumF1 / float64(numClasses)
	}

	return metrics, nil
}

// CrossValidate performs k-fold cross-validation
func CrossValidate(X [][]float64, y []string, featureNames []string, k int, maxDepth, minSamplesSplit, minSamplesLeaf int) (*CrossValidationResults, error) {
	if k <= 1 {
		return nil, fmt.Errorf("k must be greater than 1")
	}
	if len(X) < k {
		return nil, fmt.Errorf("not enough samples for %d-fold cross-validation", k)
	}

	results := &CrossValidationResults{
		K:              k,
		FoldAccuracies: make([]float64, k),
		FoldPrecisions: make([]float64, k),
		FoldRecalls:    make([]float64, k),
		FoldF1Scores:   make([]float64, k),
	}

	// Perform k-fold cross-validation
	for fold := 0; fold < k; fold++ {
		// Split data into train and validation
		trainX, trainY, valX, valY := kFoldSplit(X, y, fold, k)

		// Train model on training fold
		classifier := NewDecisionTreeClassifier(maxDepth, minSamplesSplit, minSamplesLeaf)
		if err := classifier.Train(trainX, trainY, featureNames); err != nil {
			return nil, fmt.Errorf("training failed at fold %d: %w", fold, err)
		}

		// Evaluate on validation fold
		metrics, err := Evaluate(classifier, valX, valY)
		if err != nil {
			return nil, fmt.Errorf("evaluation failed at fold %d: %w", fold, err)
		}

		results.FoldAccuracies[fold] = metrics.Accuracy
		results.FoldPrecisions[fold] = metrics.MacroPrecision
		results.FoldRecalls[fold] = metrics.MacroRecall
		results.FoldF1Scores[fold] = metrics.MacroF1
	}

	// Calculate mean and standard deviation
	results.MeanAccuracy, results.StdAccuracy = meanStd(results.FoldAccuracies)
	results.MeanPrecision, results.StdPrecision = meanStd(results.FoldPrecisions)
	results.MeanRecall, results.StdRecall = meanStd(results.FoldRecalls)
	results.MeanF1Score, results.StdF1Score = meanStd(results.FoldF1Scores)

	return results, nil
}

// CrossValidationResults holds k-fold cross-validation results
type CrossValidationResults struct {
	K              int       `json:"k"`
	FoldAccuracies []float64 `json:"fold_accuracies"`
	FoldPrecisions []float64 `json:"fold_precisions"`
	FoldRecalls    []float64 `json:"fold_recalls"`
	FoldF1Scores   []float64 `json:"fold_f1_scores"`
	MeanAccuracy   float64   `json:"mean_accuracy"`
	StdAccuracy    float64   `json:"std_accuracy"`
	MeanPrecision  float64   `json:"mean_precision"`
	StdPrecision   float64   `json:"std_precision"`
	MeanRecall     float64   `json:"mean_recall"`
	StdRecall      float64   `json:"std_recall"`
	MeanF1Score    float64   `json:"mean_f1_score"`
	StdF1Score     float64   `json:"std_f1_score"`
}

// kFoldSplit splits data into train and validation sets for k-fold cross-validation
func kFoldSplit(X [][]float64, y []string, fold int, k int) ([][]float64, []string, [][]float64, []string) {
	n := len(X)
	foldSize := n / k
	valStart := fold * foldSize
	valEnd := valStart + foldSize
	if fold == k-1 {
		valEnd = n // Last fold gets remainder
	}

	// Validation set
	valX := X[valStart:valEnd]
	valY := y[valStart:valEnd]

	// Training set (everything except validation)
	trainX := append([][]float64{}, X[:valStart]...)
	trainX = append(trainX, X[valEnd:]...)
	trainY := append([]string{}, y[:valStart]...)
	trainY = append(trainY, y[valEnd:]...)

	return trainX, trainY, valX, valY
}

// meanStd calculates mean and standard deviation
func meanStd(values []float64) (float64, float64) {
	if len(values) == 0 {
		return 0.0, 0.0
	}

	// Calculate mean
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	mean := sum / float64(len(values))

	// Calculate standard deviation
	sumSq := 0.0
	for _, v := range values {
		diff := v - mean
		sumSq += diff * diff
	}
	std := math.Sqrt(sumSq / float64(len(values)))

	return mean, std
}

// FormatMetrics returns a human-readable string representation of metrics
func (m *EvaluationMetrics) FormatMetrics() string {
	output := fmt.Sprintf("Overall Accuracy: %.4f\n", m.Accuracy)
	output += fmt.Sprintf("Total Samples: %d\n", m.TotalSamples)
	output += fmt.Sprintf("Correct Predictions: %d\n\n", m.CorrectPredictions)

	output += "Per-Class Metrics:\n"
	for class, prec := range m.Precision {
		rec := m.Recall[class]
		f1 := m.F1Score[class]
		support := m.Support[class]
		output += fmt.Sprintf("  Class '%s' (n=%d):\n", class, support)
		output += fmt.Sprintf("    Precision: %.4f\n", prec)
		output += fmt.Sprintf("    Recall:    %.4f\n", rec)
		output += fmt.Sprintf("    F1-Score:  %.4f\n", f1)
	}

	output += fmt.Sprintf("\nMacro Averages:\n")
	output += fmt.Sprintf("  Precision: %.4f\n", m.MacroPrecision)
	output += fmt.Sprintf("  Recall:    %.4f\n", m.MacroRecall)
	output += fmt.Sprintf("  F1-Score:  %.4f\n", m.MacroF1)

	return output
}

// FormatConfusionMatrix returns a formatted confusion matrix
func (m *EvaluationMetrics) FormatConfusionMatrix() string {
	// Get sorted class names
	var classes []string
	for class := range m.Support {
		classes = append(classes, class)
	}
	sort.Strings(classes)

	output := "Confusion Matrix:\n"
	output += "Actual \\ Predicted | "
	for _, class := range classes {
		output += fmt.Sprintf("%10s ", class)
	}
	output += "\n"
	output += "-------------------+"
	for range classes {
		output += "-----------"
	}
	output += "\n"

	for _, actual := range classes {
		output += fmt.Sprintf("%-18s | ", actual)
		for _, pred := range classes {
			count := m.ConfusionMatrix[actual][pred]
			output += fmt.Sprintf("%10d ", count)
		}
		output += "\n"
	}

	return output
}

// GetWorstPredictions identifies the most common misclassifications
func (m *EvaluationMetrics) GetWorstPredictions(topN int) []MisclassificationInfo {
	var misclassifications []MisclassificationInfo

	for actual, predictions := range m.ConfusionMatrix {
		for pred, count := range predictions {
			if actual != pred && count > 0 {
				misclassifications = append(misclassifications, MisclassificationInfo{
					ActualClass:    actual,
					PredictedClass: pred,
					Count:          count,
					Percentage:     float64(count) / float64(m.Support[actual]) * 100,
				})
			}
		}
	}

	// Sort by count (descending)
	sort.Slice(misclassifications, func(i, j int) bool {
		return misclassifications[i].Count > misclassifications[j].Count
	})

	// Return top N
	if topN > 0 && topN < len(misclassifications) {
		return misclassifications[:topN]
	}
	return misclassifications
}

// MisclassificationInfo holds information about a misclassification
type MisclassificationInfo struct {
	ActualClass    string  `json:"actual_class"`
	PredictedClass string  `json:"predicted_class"`
	Count          int     `json:"count"`
	Percentage     float64 `json:"percentage"`
}

// RegressionMetrics holds various regression metrics
type RegressionMetrics struct {
	MAE        float64 `json:"mae"`         // Mean Absolute Error
	MSE        float64 `json:"mse"`         // Mean Squared Error
	RMSE       float64 `json:"rmse"`        // Root Mean Squared Error
	R2Score    float64 `json:"r2_score"`    // R² (coefficient of determination)
	MAPE       float64 `json:"mape"`        // Mean Absolute Percentage Error
	MaxError   float64 `json:"max_error"`   // Maximum absolute error
	NumSamples int     `json:"num_samples"` // Number of samples
	MeanActual float64 `json:"mean_actual"` // Mean of actual values
	MeanPred   float64 `json:"mean_pred"`   // Mean of predicted values
	StdActual  float64 `json:"std_actual"`  // Std dev of actual values
	StdPred    float64 `json:"std_pred"`    // Std dev of predicted values
}

// EvaluateRegression evaluates a regression model on test data
func EvaluateRegression(classifier *DecisionTreeClassifier, X [][]float64, yTrue []float64) (*RegressionMetrics, error) {
	if len(X) == 0 || len(yTrue) == 0 {
		return nil, fmt.Errorf("empty test data")
	}
	if len(X) != len(yTrue) {
		return nil, fmt.Errorf("X and yTrue must have same length")
	}
	if classifier.ModelType != "regression" {
		return nil, fmt.Errorf("model is not a regression model")
	}

	// Make predictions
	yPred := make([]float64, len(X))
	for i, x := range X {
		pred, err := classifier.PredictRegression(x)
		if err != nil {
			return nil, fmt.Errorf("prediction failed at index %d: %w", i, err)
		}
		yPred[i] = pred
	}

	// Calculate metrics
	return CalculateRegressionMetrics(yTrue, yPred)
}

// CalculateRegressionMetrics calculates all regression evaluation metrics
func CalculateRegressionMetrics(yTrue, yPred []float64) (*RegressionMetrics, error) {
	if len(yTrue) != len(yPred) {
		return nil, fmt.Errorf("yTrue and yPred must have same length")
	}
	if len(yTrue) == 0 {
		return nil, fmt.Errorf("empty arrays")
	}

	n := len(yTrue)
	metrics := &RegressionMetrics{
		NumSamples: n,
	}

	// Calculate means
	sumTrue := 0.0
	sumPred := 0.0
	for i := 0; i < n; i++ {
		sumTrue += yTrue[i]
		sumPred += yPred[i]
	}
	metrics.MeanActual = sumTrue / float64(n)
	metrics.MeanPred = sumPred / float64(n)

	// Calculate errors and variances
	sumAbsError := 0.0
	sumSqError := 0.0
	sumAbsPercError := 0.0
	maxError := 0.0
	sumSqDiffTrue := 0.0
	sumSqDiffPred := 0.0
	validMAPECount := 0

	for i := 0; i < n; i++ {
		diff := yTrue[i] - yPred[i]
		absDiff := math.Abs(diff)

		// MAE
		sumAbsError += absDiff

		// MSE
		sumSqError += diff * diff

		// Max Error
		if absDiff > maxError {
			maxError = absDiff
		}

		// MAPE (skip zero values to avoid division by zero)
		if math.Abs(yTrue[i]) > 1e-10 {
			sumAbsPercError += (absDiff / math.Abs(yTrue[i])) * 100
			validMAPECount++
		}

		// Variance calculations
		sumSqDiffTrue += (yTrue[i] - metrics.MeanActual) * (yTrue[i] - metrics.MeanActual)
		sumSqDiffPred += (yPred[i] - metrics.MeanPred) * (yPred[i] - metrics.MeanPred)
	}

	// Calculate metrics
	metrics.MAE = sumAbsError / float64(n)
	metrics.MSE = sumSqError / float64(n)
	metrics.RMSE = math.Sqrt(metrics.MSE)
	metrics.MaxError = maxError

	if validMAPECount > 0 {
		metrics.MAPE = sumAbsPercError / float64(validMAPECount)
	}

	// Calculate R² score
	// R² = 1 - (SS_res / SS_tot)
	// SS_res = sum of squared residuals = MSE * n
	// SS_tot = total sum of squares = variance * n
	if sumSqDiffTrue > 0 {
		ssTot := sumSqDiffTrue
		ssRes := sumSqError
		metrics.R2Score = 1 - (ssRes / ssTot)
	} else {
		// If all actual values are the same, R² is undefined
		metrics.R2Score = 0.0
	}

	// Calculate standard deviations
	metrics.StdActual = math.Sqrt(sumSqDiffTrue / float64(n))
	metrics.StdPred = math.Sqrt(sumSqDiffPred / float64(n))

	return metrics, nil
}

// FormatRegressionMetrics returns a human-readable string representation of regression metrics
func (m *RegressionMetrics) FormatRegressionMetrics() string {
	output := "Regression Metrics:\n"
	output += fmt.Sprintf("  Samples:               %d\n", m.NumSamples)
	output += fmt.Sprintf("  R² Score:              %.4f\n", m.R2Score)
	output += fmt.Sprintf("  Mean Absolute Error:   %.4f\n", m.MAE)
	output += fmt.Sprintf("  Mean Squared Error:    %.4f\n", m.MSE)
	output += fmt.Sprintf("  Root Mean Squared Err: %.4f\n", m.RMSE)
	output += fmt.Sprintf("  Mean Abs. Percent Err: %.2f%%\n", m.MAPE)
	output += fmt.Sprintf("  Max Error:             %.4f\n", m.MaxError)
	output += fmt.Sprintf("\nActual Values:\n")
	output += fmt.Sprintf("  Mean:                  %.4f\n", m.MeanActual)
	output += fmt.Sprintf("  Std Dev:               %.4f\n", m.StdActual)
	output += fmt.Sprintf("\nPredicted Values:\n")
	output += fmt.Sprintf("  Mean:                  %.4f\n", m.MeanPred)
	output += fmt.Sprintf("  Std Dev:               %.4f\n", m.StdPred)
	return output
}

// EvaluateRandomForest evaluates a Random Forest classifier on test data
func EvaluateRandomForest(rf *RandomForestClassifier, X [][]float64, yTrue []string) (*EvaluationMetrics, error) {
if len(X) == 0 || len(yTrue) == 0 {
return nil, fmt.Errorf("empty test data")
}
if len(X) != len(yTrue) {
return nil, fmt.Errorf("X and yTrue must have same length")
}

// Make predictions
yPred := make([]string, len(X))
for i, x := range X {
pred, _, err := rf.Predict(x)
if err != nil {
return nil, fmt.Errorf("prediction failed at index %d: %w", i, err)
}
yPred[i] = pred
}

// Calculate metrics
return CalculateMetrics(yTrue, yPred, rf.Classes)
}

// EvaluateRandomForestRegression evaluates a Random Forest regressor on test data
func EvaluateRandomForestRegression(rf *RandomForestClassifier, X [][]float64, yTrue []float64) (*RegressionMetrics, error) {
if len(X) == 0 || len(yTrue) == 0 {
return nil, fmt.Errorf("empty test data")
}
if len(X) != len(yTrue) {
return nil, fmt.Errorf("X and yTrue must have same length")
}

// Make predictions
yPred := make([]float64, len(X))
for i, x := range X {
pred, err := rf.PredictRegression(x)
if err != nil {
return nil, fmt.Errorf("prediction failed at index %d: %w", i, err)
}
yPred[i] = pred
}

// Calculate regression metrics
return CalculateRegressionMetrics(yTrue, yPred)
}
