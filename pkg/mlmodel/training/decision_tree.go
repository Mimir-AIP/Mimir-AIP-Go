package training

import (
	"fmt"
	"math"
	"math/rand"

	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"gonum.org/v1/gonum/mat"
	"gonum.org/v1/gonum/stat"
)

// DecisionTreeTrainer implements decision tree training
// NOTE: This is a basic implementation. Production systems would use more sophisticated algorithms.
type DecisionTreeTrainer struct {
	maxDepth   int
	minSamples int
	treeModel  *DecisionTreeModel
}

// DecisionTreeModel represents a simple decision tree
type DecisionTreeModel struct {
	Feature   int
	Threshold float64
	Left      *DecisionTreeModel
	Right     *DecisionTreeModel
	Value     float64 // Leaf prediction value
	IsLeaf    bool
}

// NewDecisionTreeTrainer creates a new decision tree trainer
func NewDecisionTreeTrainer() *DecisionTreeTrainer {
	return &DecisionTreeTrainer{
		maxDepth:   10,
		minSamples: 2,
	}
}

// Train trains a decision tree model
func (t *DecisionTreeTrainer) Train(data *TrainingData, config *models.TrainingConfig) (*TrainingResult, error) {
	if len(data.TrainFeatures) == 0 {
		return nil, fmt.Errorf("no training data provided")
	}

	// Build the decision tree
	t.treeModel = t.buildTree(data.TrainFeatures, data.TrainLabels, 0)

	// Make predictions on test set
	testPredictions := make([]float64, len(data.TestLabels))
	for i, features := range data.TestFeatures {
		testPredictions[i] = t.predict(t.treeModel, features)
	}

	// Calculate performance metrics
	perfMetrics := calculateClassificationMetrics(testPredictions, data.TestLabels)

	// Training metrics
	trainingMetrics := &models.TrainingMetrics{
		Epoch:              1,
		TrainingLoss:       0,
		ValidationLoss:     0,
		TrainingAccuracy:   perfMetrics.Accuracy,
		ValidationAccuracy: perfMetrics.Accuracy,
	}

	return &TrainingResult{
		ModelData:          t.treeModel,
		TrainingMetrics:    trainingMetrics,
		PerformanceMetrics: perfMetrics,
		FeatureImportance:  calculateFeatureImportance(len(data.FeatureNames)),
	}, nil
}

// buildTree recursively builds a decision tree
func (t *DecisionTreeTrainer) buildTree(features [][]float64, labels []float64, depth int) *DecisionTreeModel {
	// Stop conditions
	if depth >= t.maxDepth || len(labels) < t.minSamples || isHomogeneous(labels) {
		return &DecisionTreeModel{
			IsLeaf: true,
			Value:  mean(labels),
		}
	}

	// Find best split
	bestFeature, bestThreshold, bestGain := t.findBestSplit(features, labels)
	if bestGain <= 0 {
		return &DecisionTreeModel{
			IsLeaf: true,
			Value:  mean(labels),
		}
	}

	// Split data
	leftFeatures, leftLabels, rightFeatures, rightLabels := splitData(features, labels, bestFeature, bestThreshold)

	// Recursively build subtrees
	return &DecisionTreeModel{
		Feature:   bestFeature,
		Threshold: bestThreshold,
		Left:      t.buildTree(leftFeatures, leftLabels, depth+1),
		Right:     t.buildTree(rightFeatures, rightLabels, depth+1),
		IsLeaf:    false,
	}
}

// findBestSplit finds the best feature and threshold to split on
func (t *DecisionTreeTrainer) findBestSplit(features [][]float64, labels []float64) (int, float64, float64) {
	if len(features) == 0 {
		return 0, 0, 0
	}

	numFeatures := len(features[0])
	bestFeature := 0
	bestThreshold := 0.0
	bestGain := 0.0

	parentImpurity := giniImpurity(labels)

	// Try each feature
	for feature := 0; feature < numFeatures; feature++ {
		// Get unique values for this feature
		values := make([]float64, len(features))
		for i, row := range features {
			values[i] = row[feature]
		}

		// Try splitting at the median
		threshold := median(values)

		// Calculate information gain
		_, leftLabels, _, rightLabels := splitData(features, labels, feature, threshold)
		if len(leftLabels) == 0 || len(rightLabels) == 0 {
			continue
		}

		leftWeight := float64(len(leftLabels)) / float64(len(labels))
		rightWeight := float64(len(rightLabels)) / float64(len(labels))

		gain := parentImpurity - (leftWeight*giniImpurity(leftLabels) + rightWeight*giniImpurity(rightLabels))

		if gain > bestGain {
			bestGain = gain
			bestFeature = feature
			bestThreshold = threshold
		}
	}

	return bestFeature, bestThreshold, bestGain
}

// predict makes a prediction using the decision tree
func (t *DecisionTreeTrainer) predict(node *DecisionTreeModel, features []float64) float64 {
	if node.IsLeaf {
		return node.Value
	}

	if features[node.Feature] <= node.Threshold {
		return t.predict(node.Left, features)
	}
	return t.predict(node.Right, features)
}

// Validate validates the trained model
func (t *DecisionTreeTrainer) Validate(data *TrainingData) (*ValidationResult, error) {
	if t.treeModel == nil {
		return nil, fmt.Errorf("model not trained")
	}

	predictions := make([]float64, len(data.TestLabels))
	for i, features := range data.TestFeatures {
		predictions[i] = t.predict(t.treeModel, features)
	}

	perfMetrics := calculateClassificationMetrics(predictions, data.TestLabels)

	return &ValidationResult{
		Loss:     0,
		Accuracy: perfMetrics.Accuracy,
		Metrics: map[string]float64{
			"accuracy":  perfMetrics.Accuracy,
			"precision": perfMetrics.Precision,
			"recall":    perfMetrics.Recall,
			"f1_score":  perfMetrics.F1Score,
		},
	}, nil
}

// GetType returns the model type
func (t *DecisionTreeTrainer) GetType() models.ModelType {
	return models.ModelTypeDecisionTree
}

// Helper functions

func isHomogeneous(labels []float64) bool {
	if len(labels) == 0 {
		return true
	}
	first := labels[0]
	for _, v := range labels {
		if v != first {
			return false
		}
	}
	return true
}

func mean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func median(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	return stat.Quantile(0.5, stat.Empirical, values, nil)
}

func giniImpurity(labels []float64) float64 {
	if len(labels) == 0 {
		return 0
	}

	// Count class frequencies
	counts := make(map[float64]int)
	for _, label := range labels {
		counts[label]++
	}

	impurity := 1.0
	total := float64(len(labels))
	for _, count := range counts {
		prob := float64(count) / total
		impurity -= prob * prob
	}

	return impurity
}

func splitData(features [][]float64, labels []float64, feature int, threshold float64) ([][]float64, []float64, [][]float64, []float64) {
	leftFeatures := [][]float64{}
	leftLabels := []float64{}
	rightFeatures := [][]float64{}
	rightLabels := []float64{}

	for i, row := range features {
		if row[feature] <= threshold {
			leftFeatures = append(leftFeatures, row)
			leftLabels = append(leftLabels, labels[i])
		} else {
			rightFeatures = append(rightFeatures, row)
			rightLabels = append(rightLabels, labels[i])
		}
	}

	return leftFeatures, leftLabels, rightFeatures, rightLabels
}

func calculateClassificationMetrics(predictions, actual []float64) *models.PerformanceMetrics {
	if len(predictions) != len(actual) {
		return &models.PerformanceMetrics{Accuracy: 0}
	}

	// Calculate accuracy
	correct := 0
	for i := range predictions {
		pred := math.Round(predictions[i])
		act := math.Round(actual[i])
		if pred == act {
			correct++
		}
	}
	accuracy := float64(correct) / float64(len(predictions))

	// For simplicity, use accuracy as approximation for precision/recall
	// In production, would calculate proper confusion matrix
	precision := accuracy
	recall := accuracy
	f1 := 2 * (precision * recall) / (precision + recall)

	return &models.PerformanceMetrics{
		Accuracy:  accuracy,
		Precision: precision,
		Recall:    recall,
		F1Score:   f1,
	}
}

func calculateRegressionMetrics(predictions, actual []float64) *models.PerformanceMetrics {
	if len(predictions) != len(actual) {
		return &models.PerformanceMetrics{}
	}

	// RMSE
	sumSquaredError := 0.0
	for i := range predictions {
		diff := predictions[i] - actual[i]
		sumSquaredError += diff * diff
	}
	rmse := math.Sqrt(sumSquaredError / float64(len(predictions)))

	// MAE
	sumAbsError := 0.0
	for i := range predictions {
		sumAbsError += math.Abs(predictions[i] - actual[i])
	}
	mae := sumAbsError / float64(len(predictions))

	// R² Score
	meanActual := mean(actual)
	ssTotal := 0.0
	ssRes := 0.0
	for i := range actual {
		ssTotal += math.Pow(actual[i]-meanActual, 2)
		ssRes += math.Pow(actual[i]-predictions[i], 2)
	}
	r2 := 1.0 - (ssRes / ssTotal)

	return &models.PerformanceMetrics{
		RMSE:    rmse,
		MAE:     mae,
		R2Score: r2,
	}
}

func calculateFeatureImportance(numFeatures int) map[string]float64 {
	// Placeholder: In production, would calculate actual feature importance
	importance := make(map[string]float64)
	for i := 0; i < numFeatures; i++ {
		importance[fmt.Sprintf("feature_%d", i)] = rand.Float64()
	}
	return importance
}

// Matrix helpers using gonum
func toMatrix(data [][]float64) *mat.Dense {
	if len(data) == 0 {
		return mat.NewDense(0, 0, nil)
	}
	rows := len(data)
	cols := len(data[0])
	flat := make([]float64, rows*cols)
	for i, row := range data {
		for j, val := range row {
			flat[i*cols+j] = val
		}
	}
	return mat.NewDense(rows, cols, flat)
}
