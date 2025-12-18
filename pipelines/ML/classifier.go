package ml

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
)

// DecisionTreeNode represents a node in the decision tree
type DecisionTreeNode struct {
	IsLeaf       bool              `json:"is_leaf"`
	Class        string            `json:"class,omitempty"`         // For leaf nodes (classification)
	ClassCounts  map[string]int    `json:"class_counts,omitempty"`  // Distribution at leaf (classification)
	Confidence   float64           `json:"confidence"`              // Confidence score
	Feature      string            `json:"feature,omitempty"`       // Feature to split on
	FeatureIndex int               `json:"feature_index,omitempty"` // Index of feature
	Threshold    float64           `json:"threshold,omitempty"`     // Split threshold
	Left         *DecisionTreeNode `json:"left,omitempty"`          // Left child (<=)
	Right        *DecisionTreeNode `json:"right,omitempty"`         // Right child (>)
	SamplesCount int               `json:"samples_count"`           // Number of samples at this node
	Depth        int               `json:"depth"`                   // Depth in tree
	// Regression-specific fields
	NumericValue  float64   `json:"numeric_value,omitempty"`  // Mean value for regression leaf
	NumericValues []float64 `json:"numeric_values,omitempty"` // Sample values at leaf (for variance)
	NodeType      string    `json:"node_type"`                // "classification" or "regression"
}

// DecisionTreeClassifier implements a lightweight decision tree
type DecisionTreeClassifier struct {
	Root            *DecisionTreeNode `json:"root"`
	MaxDepth        int               `json:"max_depth"`
	MinSamplesSplit int               `json:"min_samples_split"`
	MinSamplesLeaf  int               `json:"min_samples_leaf"`
	FeatureNames    []string          `json:"feature_names"`
	Classes         []string          `json:"classes"`
	NumFeatures     int               `json:"num_features"`
	NumClasses      int               `json:"num_classes"`
	ModelType       string            `json:"model_type"` // "classification" or "regression"
}

// NewDecisionTreeClassifier creates a new decision tree classifier with default hyperparameters
func NewDecisionTreeClassifier(maxDepth, minSamplesSplit, minSamplesLeaf int) *DecisionTreeClassifier {
	if maxDepth <= 0 {
		maxDepth = 10 // Default
	}
	if minSamplesSplit <= 0 {
		minSamplesSplit = 2 // Default
	}
	if minSamplesLeaf <= 0 {
		minSamplesLeaf = 1 // Default
	}

	return &DecisionTreeClassifier{
		MaxDepth:        maxDepth,
		MinSamplesSplit: minSamplesSplit,
		MinSamplesLeaf:  minSamplesLeaf,
	}
}

// Train builds the decision tree from training data
// X: feature matrix (rows = samples, cols = features)
// y: target labels (one per sample)
func (dt *DecisionTreeClassifier) Train(X [][]float64, y []string, featureNames []string) error {
	if len(X) == 0 {
		return fmt.Errorf("empty training data")
	}
	if len(X) != len(y) {
		return fmt.Errorf("X and y must have same number of samples")
	}
	if len(featureNames) != len(X[0]) {
		return fmt.Errorf("feature names must match number of features")
	}

	dt.FeatureNames = featureNames
	dt.NumFeatures = len(X[0])
	dt.Classes = uniqueStrings(y)
	dt.NumClasses = len(dt.Classes)
	dt.ModelType = "classification"

	// Build the tree recursively
	indices := make([]int, len(X))
	for i := range indices {
		indices[i] = i
	}

	dt.Root = dt.buildTree(X, y, indices, 0)
	return nil
}

// TrainRegression builds a regression decision tree from training data
// X: feature matrix (rows = samples, cols = features)
// y: target values (numeric, one per sample)
func (dt *DecisionTreeClassifier) TrainRegression(X [][]float64, y []float64, featureNames []string) error {
	if len(X) == 0 {
		return fmt.Errorf("empty training data")
	}
	if len(X) != len(y) {
		return fmt.Errorf("X and y must have same number of samples")
	}
	if len(featureNames) != len(X[0]) {
		return fmt.Errorf("feature names must match number of features")
	}

	dt.FeatureNames = featureNames
	dt.NumFeatures = len(X[0])
	dt.ModelType = "regression"

	// Build the tree recursively
	indices := make([]int, len(X))
	for i := range indices {
		indices[i] = i
	}

	dt.Root = dt.buildTreeRegression(X, y, indices, 0)
	return nil
}

// buildTree recursively builds the decision tree
func (dt *DecisionTreeClassifier) buildTree(X [][]float64, y []string, indices []int, depth int) *DecisionTreeNode {
	node := &DecisionTreeNode{
		SamplesCount: len(indices),
		Depth:        depth,
	}

	// Extract labels for current samples
	currentLabels := make([]string, len(indices))
	for i, idx := range indices {
		currentLabels[i] = y[idx]
	}

	// Count class occurrences
	classCounts := countClasses(currentLabels)
	node.ClassCounts = classCounts

	// Find majority class
	majorityClass, majorityCount := getMajorityClass(classCounts)
	node.Class = majorityClass
	node.Confidence = float64(majorityCount) / float64(len(indices))

	// Check stopping criteria
	if depth >= dt.MaxDepth || len(indices) < dt.MinSamplesSplit || len(classCounts) == 1 {
		node.IsLeaf = true
		return node
	}

	// Find best split
	bestFeature, bestThreshold, bestGain := dt.findBestSplit(X, y, indices)
	if bestGain <= 0 {
		node.IsLeaf = true
		return node
	}

	// Split data
	leftIndices, rightIndices := dt.splitData(X, indices, bestFeature, bestThreshold)

	// Check minimum samples per leaf
	if len(leftIndices) < dt.MinSamplesLeaf || len(rightIndices) < dt.MinSamplesLeaf {
		node.IsLeaf = true
		return node
	}

	// Create internal node
	node.IsLeaf = false
	node.Feature = dt.FeatureNames[bestFeature]
	node.FeatureIndex = bestFeature
	node.Threshold = bestThreshold

	// Recursively build left and right subtrees
	node.Left = dt.buildTree(X, y, leftIndices, depth+1)
	node.Right = dt.buildTree(X, y, rightIndices, depth+1)

	return node
}

// buildTreeRegression recursively builds a regression decision tree
func (dt *DecisionTreeClassifier) buildTreeRegression(X [][]float64, y []float64, indices []int, depth int) *DecisionTreeNode {
	node := &DecisionTreeNode{
		SamplesCount: len(indices),
		Depth:        depth,
		NodeType:     "regression",
	}

	// Extract values for current samples
	currentValues := make([]float64, len(indices))
	for i, idx := range indices {
		currentValues[i] = y[idx]
	}

	// Calculate mean and variance
	mean := calculateMean(currentValues)
	variance := calculateVariance(currentValues, mean)
	node.NumericValue = mean
	node.NumericValues = currentValues

	// Check stopping criteria
	if depth >= dt.MaxDepth || len(indices) < dt.MinSamplesSplit || variance < 1e-7 {
		node.IsLeaf = true
		return node
	}

	// Find best split
	bestFeature, bestThreshold, bestGain := dt.findBestSplitRegression(X, y, indices)
	if bestGain <= 0 {
		node.IsLeaf = true
		return node
	}

	// Split data
	leftIndices, rightIndices := dt.splitData(X, indices, bestFeature, bestThreshold)

	// Check minimum samples per leaf
	if len(leftIndices) < dt.MinSamplesLeaf || len(rightIndices) < dt.MinSamplesLeaf {
		node.IsLeaf = true
		return node
	}

	// Create internal node
	node.IsLeaf = false
	node.Feature = dt.FeatureNames[bestFeature]
	node.FeatureIndex = bestFeature
	node.Threshold = bestThreshold

	// Recursively build left and right subtrees
	node.Left = dt.buildTreeRegression(X, y, leftIndices, depth+1)
	node.Right = dt.buildTreeRegression(X, y, rightIndices, depth+1)

	return node
}

// findBestSplit finds the best feature and threshold to split on
func (dt *DecisionTreeClassifier) findBestSplit(X [][]float64, y []string, indices []int) (int, float64, float64) {
	bestGain := 0.0
	bestFeature := -1
	bestThreshold := 0.0

	currentLabels := make([]string, len(indices))
	for i, idx := range indices {
		currentLabels[i] = y[idx]
	}
	parentGini := dt.giniImpurity(currentLabels)

	// Try each feature
	for feature := 0; feature < dt.NumFeatures; feature++ {
		// Get unique values for this feature
		values := make([]float64, len(indices))
		for i, idx := range indices {
			values[i] = X[idx][feature]
		}

		// Get candidate thresholds (midpoints between unique values)
		thresholds := getThresholds(values)

		// Try each threshold
		for _, threshold := range thresholds {
			leftIndices, rightIndices := dt.splitData(X, indices, feature, threshold)

			if len(leftIndices) == 0 || len(rightIndices) == 0 {
				continue
			}

			// Calculate weighted Gini impurity
			leftLabels := make([]string, len(leftIndices))
			for i, idx := range leftIndices {
				leftLabels[i] = y[idx]
			}
			rightLabels := make([]string, len(rightIndices))
			for i, idx := range rightIndices {
				rightLabels[i] = y[idx]
			}

			leftGini := dt.giniImpurity(leftLabels)
			rightGini := dt.giniImpurity(rightLabels)

			n := float64(len(indices))
			nLeft := float64(len(leftIndices))
			nRight := float64(len(rightIndices))

			weightedGini := (nLeft/n)*leftGini + (nRight/n)*rightGini
			gain := parentGini - weightedGini

			if gain > bestGain {
				bestGain = gain
				bestFeature = feature
				bestThreshold = threshold
			}
		}
	}

	return bestFeature, bestThreshold, bestGain
}

// giniImpurity calculates the Gini impurity of a set of labels
func (dt *DecisionTreeClassifier) giniImpurity(labels []string) float64 {
	if len(labels) == 0 {
		return 0.0
	}

	counts := countClasses(labels)
	n := float64(len(labels))
	gini := 1.0

	for _, count := range counts {
		p := float64(count) / n
		gini -= p * p
	}

	return gini
}

// splitData splits indices based on feature and threshold
func (dt *DecisionTreeClassifier) splitData(X [][]float64, indices []int, feature int, threshold float64) ([]int, []int) {
	var leftIndices, rightIndices []int

	for _, idx := range indices {
		if X[idx][feature] <= threshold {
			leftIndices = append(leftIndices, idx)
		} else {
			rightIndices = append(rightIndices, idx)
		}
	}

	return leftIndices, rightIndices
}

// Predict predicts the class for a single sample
func (dt *DecisionTreeClassifier) Predict(x []float64) (string, float64, error) {
	if dt.Root == nil {
		return "", 0.0, fmt.Errorf("model not trained")
	}
	if len(x) != dt.NumFeatures {
		return "", 0.0, fmt.Errorf("expected %d features, got %d", dt.NumFeatures, len(x))
	}

	return dt.traverseTree(dt.Root, x), dt.Root.Confidence, nil
}

// traverseTree traverses the tree to make a prediction
func (dt *DecisionTreeClassifier) traverseTree(node *DecisionTreeNode, x []float64) string {
	if node.IsLeaf {
		return node.Class
	}

	if x[node.FeatureIndex] <= node.Threshold {
		return dt.traverseTree(node.Left, x)
	}
	return dt.traverseTree(node.Right, x)
}

// PredictProba predicts class probabilities for a single sample
func (dt *DecisionTreeClassifier) PredictProba(x []float64) (map[string]float64, error) {
	if dt.Root == nil {
		return nil, fmt.Errorf("model not trained")
	}
	if len(x) != dt.NumFeatures {
		return nil, fmt.Errorf("expected %d features, got %d", dt.NumFeatures, len(x))
	}

	leafNode := dt.traverseToLeaf(dt.Root, x)
	proba := make(map[string]float64)

	total := 0
	for _, count := range leafNode.ClassCounts {
		total += count
	}

	for class, count := range leafNode.ClassCounts {
		proba[class] = float64(count) / float64(total)
	}

	return proba, nil
}

// traverseToLeaf traverses to the leaf node
func (dt *DecisionTreeClassifier) traverseToLeaf(node *DecisionTreeNode, x []float64) *DecisionTreeNode {
	if node.IsLeaf {
		return node
	}

	if x[node.FeatureIndex] <= node.Threshold {
		return dt.traverseToLeaf(node.Left, x)
	}
	return dt.traverseToLeaf(node.Right, x)
}

// Save saves the model to a JSON file
func (dt *DecisionTreeClassifier) Save(path string) error {
	data, err := json.MarshalIndent(dt, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal model: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write model file: %w", err)
	}

	return nil
}

// Load loads a model from a JSON file
func (dt *DecisionTreeClassifier) Load(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read model file: %w", err)
	}

	if err := json.Unmarshal(data, dt); err != nil {
		return fmt.Errorf("failed to unmarshal model: %w", err)
	}

	return nil
}

// GetFeatureImportance calculates feature importance based on how much each feature reduces impurity
func (dt *DecisionTreeClassifier) GetFeatureImportance() map[string]float64 {
	importance := make(map[string]float64)
	for _, name := range dt.FeatureNames {
		importance[name] = 0.0
	}

	if dt.Root != nil {
		dt.calculateImportance(dt.Root, importance)
	}

	// Normalize
	total := 0.0
	for _, val := range importance {
		total += val
	}
	if total > 0 {
		for name := range importance {
			importance[name] /= total
		}
	}

	return importance
}

// calculateImportance recursively calculates feature importance
func (dt *DecisionTreeClassifier) calculateImportance(node *DecisionTreeNode, importance map[string]float64) {
	if node.IsLeaf {
		return
	}

	// Importance is weighted by number of samples at this node
	importance[node.Feature] += float64(node.SamplesCount)

	if node.Left != nil {
		dt.calculateImportance(node.Left, importance)
	}
	if node.Right != nil {
		dt.calculateImportance(node.Right, importance)
	}
}

// Prune removes branches that don't improve accuracy
func (dt *DecisionTreeClassifier) Prune(X [][]float64, y []string) {
	// TODO: Implement pruning using validation set
	// For now, this is a placeholder for future optimization
}

// GetDepth returns the maximum depth of the tree
func (dt *DecisionTreeClassifier) GetDepth() int {
	if dt.Root == nil {
		return 0
	}
	return dt.getNodeDepth(dt.Root)
}

func (dt *DecisionTreeClassifier) getNodeDepth(node *DecisionTreeNode) int {
	if node.IsLeaf {
		return node.Depth
	}

	leftDepth := dt.getNodeDepth(node.Left)
	rightDepth := dt.getNodeDepth(node.Right)

	if leftDepth > rightDepth {
		return leftDepth
	}
	return rightDepth
}

// GetNumNodes returns the total number of nodes in the tree
func (dt *DecisionTreeClassifier) GetNumNodes() int {
	if dt.Root == nil {
		return 0
	}
	return dt.countNodes(dt.Root)
}

func (dt *DecisionTreeClassifier) countNodes(node *DecisionTreeNode) int {
	if node == nil {
		return 0
	}
	return 1 + dt.countNodes(node.Left) + dt.countNodes(node.Right)
}

// Helper functions

func countClasses(labels []string) map[string]int {
	counts := make(map[string]int)
	for _, label := range labels {
		counts[label]++
	}
	return counts
}

func getMajorityClass(classCounts map[string]int) (string, int) {
	maxClass := ""
	maxCount := 0
	for class, count := range classCounts {
		if count > maxCount {
			maxClass = class
			maxCount = count
		}
	}
	return maxClass, maxCount
}

func uniqueStrings(strs []string) []string {
	seen := make(map[string]bool)
	unique := []string{}
	for _, s := range strs {
		if !seen[s] {
			seen[s] = true
			unique = append(unique, s)
		}
	}
	sort.Strings(unique)
	return unique
}

func getThresholds(values []float64) []float64 {
	if len(values) == 0 {
		return nil
	}

	// Get unique sorted values
	uniqueVals := make([]float64, 0, len(values))
	seen := make(map[float64]bool)
	for _, v := range values {
		if !seen[v] {
			seen[v] = true
			uniqueVals = append(uniqueVals, v)
		}
	}

	if len(uniqueVals) == 1 {
		return nil
	}

	sort.Float64s(uniqueVals)

	// Create thresholds as midpoints
	thresholds := make([]float64, len(uniqueVals)-1)
	for i := 0; i < len(uniqueVals)-1; i++ {
		thresholds[i] = (uniqueVals[i] + uniqueVals[i+1]) / 2.0
	}

	return thresholds
}

// EstimateMemoryUsage estimates the memory usage of the tree in bytes
func (dt *DecisionTreeClassifier) EstimateMemoryUsage() int64 {
	if dt.Root == nil {
		return 0
	}

	// Rough estimate: each node ~200 bytes (struct overhead + pointers)
	numNodes := dt.GetNumNodes()
	nodeSize := int64(200)

	// Add storage for feature names and classes
	metadataSize := int64(len(dt.FeatureNames)*20 + len(dt.Classes)*20)

	return int64(numNodes)*nodeSize + metadataSize
}

// Validate checks if the model is valid and ready for predictions
func (dt *DecisionTreeClassifier) Validate() error {
	if dt.Root == nil {
		return fmt.Errorf("model has no root node")
	}
	if len(dt.FeatureNames) == 0 {
		return fmt.Errorf("model has no feature names")
	}
	if len(dt.Classes) == 0 {
		return fmt.Errorf("model has no classes")
	}
	if dt.NumFeatures != len(dt.FeatureNames) {
		return fmt.Errorf("num_features mismatch")
	}
	return nil
}

// GetModelInfo returns summary information about the model
func (dt *DecisionTreeClassifier) GetModelInfo() map[string]interface{} {
	info := map[string]interface{}{
		"algorithm":             "decision_tree",
		"num_features":          dt.NumFeatures,
		"num_classes":           dt.NumClasses,
		"max_depth":             dt.MaxDepth,
		"actual_depth":          dt.GetDepth(),
		"num_nodes":             dt.GetNumNodes(),
		"min_samples_split":     dt.MinSamplesSplit,
		"min_samples_leaf":      dt.MinSamplesLeaf,
		"feature_names":         dt.FeatureNames,
		"classes":               dt.Classes,
		"memory_estimate_bytes": dt.EstimateMemoryUsage(),
	}
	return info
}

// CalculateConfidence calculates confidence for a prediction
func (dt *DecisionTreeClassifier) CalculateConfidence(x []float64) (float64, error) {
	if dt.Root == nil {
		return 0.0, fmt.Errorf("model not trained")
	}

	leafNode := dt.traverseToLeaf(dt.Root, x)
	return leafNode.Confidence, nil
}

// IsLowConfidence checks if a prediction has low confidence (potential anomaly)
func (dt *DecisionTreeClassifier) IsLowConfidence(x []float64, threshold float64) (bool, error) {
	confidence, err := dt.CalculateConfidence(x)
	if err != nil {
		return false, err
	}
	return confidence < threshold, nil
}

// findBestSplitRegression finds the best feature and threshold to split on for regression
func (dt *DecisionTreeClassifier) findBestSplitRegression(X [][]float64, y []float64, indices []int) (int, float64, float64) {
	bestGain := 0.0
	bestFeature := -1
	bestThreshold := 0.0

	currentValues := make([]float64, len(indices))
	for i, idx := range indices {
		currentValues[i] = y[idx]
	}
	parentVariance := calculateVariance(currentValues, calculateMean(currentValues))

	// Try each feature
	for feature := 0; feature < dt.NumFeatures; feature++ {
		// Get unique values for this feature
		values := make([]float64, len(indices))
		for i, idx := range indices {
			values[i] = X[idx][feature]
		}

		// Get candidate thresholds (midpoints between unique values)
		thresholds := getThresholds(values)

		// Try each threshold
		for _, threshold := range thresholds {
			leftIndices, rightIndices := dt.splitData(X, indices, feature, threshold)

			if len(leftIndices) == 0 || len(rightIndices) == 0 {
				continue
			}

			// Calculate weighted variance
			leftValues := make([]float64, len(leftIndices))
			for i, idx := range leftIndices {
				leftValues[i] = y[idx]
			}
			rightValues := make([]float64, len(rightIndices))
			for i, idx := range rightIndices {
				rightValues[i] = y[idx]
			}

			leftMean := calculateMean(leftValues)
			rightMean := calculateMean(rightValues)
			leftVariance := calculateVariance(leftValues, leftMean)
			rightVariance := calculateVariance(rightValues, rightMean)

			n := float64(len(indices))
			nLeft := float64(len(leftIndices))
			nRight := float64(len(rightIndices))

			weightedVariance := (nLeft/n)*leftVariance + (nRight/n)*rightVariance
			gain := parentVariance - weightedVariance

			if gain > bestGain {
				bestGain = gain
				bestFeature = feature
				bestThreshold = threshold
			}
		}
	}

	return bestFeature, bestThreshold, bestGain
}

// PredictRegression predicts a numeric value for a single sample
func (dt *DecisionTreeClassifier) PredictRegression(x []float64) (float64, error) {
	if dt.Root == nil {
		return 0.0, fmt.Errorf("model not trained")
	}
	if dt.ModelType != "regression" {
		return 0.0, fmt.Errorf("model is not a regression model")
	}
	if len(x) != dt.NumFeatures {
		return 0.0, fmt.Errorf("expected %d features, got %d", dt.NumFeatures, len(x))
	}

	return dt.traverseTreeRegression(dt.Root, x), nil
}

// traverseTreeRegression traverses the tree to make a regression prediction
func (dt *DecisionTreeClassifier) traverseTreeRegression(node *DecisionTreeNode, x []float64) float64 {
	if node.IsLeaf {
		return node.NumericValue
	}

	if x[node.FeatureIndex] <= node.Threshold {
		return dt.traverseTreeRegression(node.Left, x)
	}
	return dt.traverseTreeRegression(node.Right, x)
}

// PredictRegressionWithInterval predicts a numeric value with confidence interval
func (dt *DecisionTreeClassifier) PredictRegressionWithInterval(x []float64) (value, lower, upper float64, err error) {
	if dt.Root == nil {
		return 0.0, 0.0, 0.0, fmt.Errorf("model not trained")
	}
	if dt.ModelType != "regression" {
		return 0.0, 0.0, 0.0, fmt.Errorf("model is not a regression model")
	}
	if len(x) != dt.NumFeatures {
		return 0.0, 0.0, 0.0, fmt.Errorf("expected %d features, got %d", dt.NumFeatures, len(x))
	}

	leafNode := dt.traverseToLeafRegression(dt.Root, x)
	value = leafNode.NumericValue

	// Calculate standard error based on leaf values
	if len(leafNode.NumericValues) > 1 {
		mean := calculateMean(leafNode.NumericValues)
		variance := calculateVariance(leafNode.NumericValues, mean)
		stdError := math.Sqrt(variance)

		// 95% confidence interval (approximately ±2 standard errors)
		lower = value - 2*stdError
		upper = value + 2*stdError
	} else {
		// Single value at leaf, no confidence interval
		lower = value
		upper = value
	}

	return value, lower, upper, nil
}

// traverseToLeafRegression traverses to the leaf node for regression
func (dt *DecisionTreeClassifier) traverseToLeafRegression(node *DecisionTreeNode, x []float64) *DecisionTreeNode {
	if node.IsLeaf {
		return node
	}

	if x[node.FeatureIndex] <= node.Threshold {
		return dt.traverseToLeafRegression(node.Left, x)
	}
	return dt.traverseToLeafRegression(node.Right, x)
}

// Helper functions for regression

func calculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0.0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func calculateVariance(values []float64, mean float64) float64 {
	if len(values) == 0 {
		return 0.0
	}
	variance := 0.0
	for _, v := range values {
		diff := v - mean
		variance += diff * diff
	}
	return variance / float64(len(values))
}
