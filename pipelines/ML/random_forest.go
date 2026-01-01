package ml

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"
	"sync"
	"time"
)

// RandomForestClassifier implements a Random Forest ensemble model
// Supports both classification and regression tasks
type RandomForestClassifier struct {
	Trees            []*DecisionTreeClassifier `json:"trees"`
	TreeFeatures     [][]int                   `json:"tree_features"`      // Feature indices used by each tree
	NumTrees         int                       `json:"num_trees"`
	MaxDepth         int                       `json:"max_depth"`
	MinSamplesSplit  int                       `json:"min_samples_split"`
	MinSamplesLeaf   int                       `json:"min_samples_leaf"`
	MaxFeatures      int                       `json:"max_features"`       // Number of features to consider per split
	Bootstrap        bool                      `json:"bootstrap"`          // Use bootstrap sampling
	OOBScore         float64                   `json:"oob_score"`          // Out-of-bag score
	FeatureNames     []string                  `json:"feature_names"`
	Classes          []string                  `json:"classes"`
	NumFeatures      int                       `json:"num_features"`
	NumClasses       int                       `json:"num_classes"`
	ModelType        string                    `json:"model_type"` // "classification" or "regression"
	Rand             *rand.Rand                `json:"-"`          // Random number generator
	RandomSeed       int64                     `json:"random_seed"`
}

// NewRandomForestClassifier creates a new Random Forest classifier
func NewRandomForestClassifier(numTrees, maxDepth, minSamplesSplit, minSamplesLeaf int) *RandomForestClassifier {
	if numTrees <= 0 {
		numTrees = 100 // Default
	}
	if maxDepth <= 0 {
		maxDepth = 10 // Default
	}
	if minSamplesSplit <= 0 {
		minSamplesSplit = 2 // Default
	}
	if minSamplesLeaf <= 0 {
		minSamplesLeaf = 1 // Default
	}

	seed := time.Now().UnixNano()
	return &RandomForestClassifier{
		NumTrees:        numTrees,
		MaxDepth:        maxDepth,
		MinSamplesSplit: minSamplesSplit,
		MinSamplesLeaf:  minSamplesLeaf,
		Bootstrap:       true,
		Rand:            rand.New(rand.NewSource(seed)),
		RandomSeed:      seed,
	}
}

// Train builds the random forest from training data (classification)
func (rf *RandomForestClassifier) Train(X [][]float64, y []string, featureNames []string) error {
	if len(X) == 0 {
		return fmt.Errorf("empty training data")
	}
	if len(X) != len(y) {
		return fmt.Errorf("X and y must have same number of samples")
	}
	if len(featureNames) != len(X[0]) {
		return fmt.Errorf("feature names must match number of features")
	}

	rf.FeatureNames = featureNames
	rf.NumFeatures = len(X[0])
	rf.Classes = uniqueStrings(y)
	rf.NumClasses = len(rf.Classes)
	rf.ModelType = "classification"

	// Set max features for random feature selection (sqrt of total features)
	rf.MaxFeatures = int(math.Sqrt(float64(rf.NumFeatures)))
	if rf.MaxFeatures < 1 {
		rf.MaxFeatures = 1
	}

	// Train trees in parallel
	rf.Trees = make([]*DecisionTreeClassifier, rf.NumTrees)
	rf.TreeFeatures = make([][]int, rf.NumTrees)
	
	var wg sync.WaitGroup
	var mu sync.Mutex
	errors := make([]error, 0)

	for i := 0; i < rf.NumTrees; i++ {
		wg.Add(1)
		go func(treeIdx int) {
			defer wg.Done()

			// Create bootstrap sample
			bootX, bootY := rf.bootstrapSample(X, y)

			// Create and train a tree with random feature selection
			tree := NewDecisionTreeClassifier(rf.MaxDepth, rf.MinSamplesSplit, rf.MinSamplesLeaf)
			
			// Use a subset of features for this tree
			selectedFeatures := rf.selectRandomFeatures()
			subX, subFeatureNames := rf.extractFeatures(bootX, selectedFeatures)

			err := tree.Train(subX, bootY, subFeatureNames)
			if err != nil {
				mu.Lock()
				errors = append(errors, fmt.Errorf("tree %d training failed: %w", treeIdx, err))
				mu.Unlock()
				return
			}
			
			mu.Lock()
			rf.Trees[treeIdx] = tree
			rf.TreeFeatures[treeIdx] = selectedFeatures
			mu.Unlock()
		}(i)
	}

	wg.Wait()

	if len(errors) > 0 {
		return fmt.Errorf("training errors: %v", errors[0])
	}

	// Calculate OOB score
	rf.OOBScore = rf.calculateOOBScore(X, y)

	return nil
}

// TrainRegression builds the random forest from training data (regression)
func (rf *RandomForestClassifier) TrainRegression(X [][]float64, y []float64, featureNames []string) error {
	if len(X) == 0 {
		return fmt.Errorf("empty training data")
	}
	if len(X) != len(y) {
		return fmt.Errorf("X and y must have same number of samples")
	}
	if len(featureNames) != len(X[0]) {
		return fmt.Errorf("feature names must match number of features")
	}

	rf.FeatureNames = featureNames
	rf.NumFeatures = len(X[0])
	rf.ModelType = "regression"

	// Set max features for random feature selection
	rf.MaxFeatures = int(math.Sqrt(float64(rf.NumFeatures)))
	if rf.MaxFeatures < 1 {
		rf.MaxFeatures = 1
	}

	// Train trees in parallel
	rf.Trees = make([]*DecisionTreeClassifier, rf.NumTrees)
	rf.TreeFeatures = make([][]int, rf.NumTrees)
	
	var wg sync.WaitGroup
	var mu sync.Mutex
	errors := make([]error, 0)

	for i := 0; i < rf.NumTrees; i++ {
		wg.Add(1)
		go func(treeIdx int) {
			defer wg.Done()

			// Create bootstrap sample
			bootX, bootY := rf.bootstrapSampleRegression(X, y)

			// Create and train a tree
			tree := NewDecisionTreeClassifier(rf.MaxDepth, rf.MinSamplesSplit, rf.MinSamplesLeaf)
			
			// Use a subset of features for this tree
			selectedFeatures := rf.selectRandomFeatures()
			subX, subFeatureNames := rf.extractFeatures(bootX, selectedFeatures)

			err := tree.TrainRegression(subX, bootY, subFeatureNames)
			if err != nil {
				mu.Lock()
				errors = append(errors, fmt.Errorf("tree %d training failed: %w", treeIdx, err))
				mu.Unlock()
				return
			}
			
			mu.Lock()
			rf.Trees[treeIdx] = tree
			rf.TreeFeatures[treeIdx] = selectedFeatures
			mu.Unlock()
		}(i)
	}

	wg.Wait()

	if len(errors) > 0 {
		return fmt.Errorf("training errors: %v", errors[0])
	}

	// Calculate OOB score for regression
	rf.OOBScore = rf.calculateOOBScoreRegression(X, y)

	return nil
}

// bootstrapSample creates a bootstrap sample (with replacement)
func (rf *RandomForestClassifier) bootstrapSample(X [][]float64, y []string) ([][]float64, []string) {
	n := len(X)
	bootX := make([][]float64, n)
	bootY := make([]string, n)

	for i := 0; i < n; i++ {
		idx := rf.Rand.Intn(n)
		bootX[i] = X[idx]
		bootY[i] = y[idx]
	}

	return bootX, bootY
}

// bootstrapSampleRegression creates a bootstrap sample for regression
func (rf *RandomForestClassifier) bootstrapSampleRegression(X [][]float64, y []float64) ([][]float64, []float64) {
	n := len(X)
	bootX := make([][]float64, n)
	bootY := make([]float64, n)

	for i := 0; i < n; i++ {
		idx := rf.Rand.Intn(n)
		bootX[i] = X[idx]
		bootY[i] = y[idx]
	}

	return bootX, bootY
}

// selectRandomFeatures randomly selects features for a tree
func (rf *RandomForestClassifier) selectRandomFeatures() []int {
	// Select max_features random features
	features := make([]int, rf.NumFeatures)
	for i := range features {
		features[i] = i
	}

	// Shuffle and take first MaxFeatures
	rf.Rand.Shuffle(len(features), func(i, j int) {
		features[i], features[j] = features[j], features[i]
	})

	return features[:rf.MaxFeatures]
}

// extractFeatures extracts selected features from data
func (rf *RandomForestClassifier) extractFeatures(X [][]float64, features []int) ([][]float64, []string) {
	subX := make([][]float64, len(X))
	subFeatureNames := make([]string, len(features))

	for i := range X {
		subX[i] = make([]float64, len(features))
		for j, fIdx := range features {
			subX[i][j] = X[i][fIdx]
		}
	}

	for i, fIdx := range features {
		subFeatureNames[i] = rf.FeatureNames[fIdx]
	}

	return subX, subFeatureNames
}

// Predict predicts the class for a single sample (classification)
func (rf *RandomForestClassifier) Predict(x []float64) (string, float64, error) {
	if len(rf.Trees) == 0 {
		return "", 0.0, fmt.Errorf("model not trained")
	}
	if rf.ModelType != "classification" {
		return "", 0.0, fmt.Errorf("model is not a classification model")
	}
	if len(x) != rf.NumFeatures {
		return "", 0.0, fmt.Errorf("expected %d features, got %d", rf.NumFeatures, len(x))
	}

	// Collect votes from all trees
	votes := make(map[string]int)
	for i, tree := range rf.Trees {
		if tree == nil {
			continue
		}
		
		// Extract features used by this tree
		treeFeatures := make([]float64, len(rf.TreeFeatures[i]))
		for j, fIdx := range rf.TreeFeatures[i] {
			treeFeatures[j] = x[fIdx]
		}
		
		predicted, _, err := tree.Predict(treeFeatures)
		if err != nil {
			continue // Skip failed predictions
		}
		votes[predicted]++
	}

	if len(votes) == 0 {
		return "", 0.0, fmt.Errorf("no valid predictions from trees")
	}

	// Find majority vote
	maxVotes := 0
	majorityClass := ""
	for class, count := range votes {
		if count > maxVotes {
			maxVotes = count
			majorityClass = class
		}
	}

	confidence := float64(maxVotes) / float64(len(rf.Trees))
	return majorityClass, confidence, nil
}

// PredictProba predicts class probabilities for a single sample
func (rf *RandomForestClassifier) PredictProba(x []float64) (map[string]float64, error) {
	if len(rf.Trees) == 0 {
		return nil, fmt.Errorf("model not trained")
	}
	if rf.ModelType != "classification" {
		return nil, fmt.Errorf("model is not a classification model")
	}
	if len(x) != rf.NumFeatures {
		return nil, fmt.Errorf("expected %d features, got %d", rf.NumFeatures, len(x))
	}

	// Collect votes from all trees
	votes := make(map[string]int)
	validTrees := 0

	for i, tree := range rf.Trees {
		if tree == nil {
			continue
		}
		
		// Extract features used by this tree
		treeFeatures := make([]float64, len(rf.TreeFeatures[i]))
		for j, fIdx := range rf.TreeFeatures[i] {
			treeFeatures[j] = x[fIdx]
		}
		
		predicted, _, err := tree.Predict(treeFeatures)
		if err != nil {
			continue
		}
		votes[predicted]++
		validTrees++
	}

	if validTrees == 0 {
		return nil, fmt.Errorf("no valid predictions from trees")
	}

	// Convert votes to probabilities
	proba := make(map[string]float64)
	for class, count := range votes {
		proba[class] = float64(count) / float64(validTrees)
	}

	// Ensure all classes have a probability
	for _, class := range rf.Classes {
		if _, exists := proba[class]; !exists {
			proba[class] = 0.0
		}
	}

	return proba, nil
}

// PredictRegression predicts a numeric value for a single sample (regression)
func (rf *RandomForestClassifier) PredictRegression(x []float64) (float64, error) {
	if len(rf.Trees) == 0 {
		return 0.0, fmt.Errorf("model not trained")
	}
	if rf.ModelType != "regression" {
		return 0.0, fmt.Errorf("model is not a regression model")
	}
	if len(x) != rf.NumFeatures {
		return 0.0, fmt.Errorf("expected %d features, got %d", rf.NumFeatures, len(x))
	}

	// Average predictions from all trees
	sum := 0.0
	validTrees := 0

	for i, tree := range rf.Trees {
		if tree == nil {
			continue
		}
		
		// Extract features used by this tree
		treeFeatures := make([]float64, len(rf.TreeFeatures[i]))
		for j, fIdx := range rf.TreeFeatures[i] {
			treeFeatures[j] = x[fIdx]
		}
		
		predicted, err := tree.PredictRegression(treeFeatures)
		if err != nil {
			continue
		}
		sum += predicted
		validTrees++
	}

	if validTrees == 0 {
		return 0.0, fmt.Errorf("no valid predictions from trees")
	}

	return sum / float64(validTrees), nil
}

// PredictRegressionWithInterval predicts with confidence interval
func (rf *RandomForestClassifier) PredictRegressionWithInterval(x []float64) (value, lower, upper float64, err error) {
	if len(rf.Trees) == 0 {
		return 0.0, 0.0, 0.0, fmt.Errorf("model not trained")
	}
	if rf.ModelType != "regression" {
		return 0.0, 0.0, 0.0, fmt.Errorf("model is not a regression model")
	}
	if len(x) != rf.NumFeatures {
		return 0.0, 0.0, 0.0, fmt.Errorf("expected %d features, got %d", rf.NumFeatures, len(x))
	}

	// Collect predictions from all trees
	predictions := make([]float64, 0, len(rf.Trees))
	for i, tree := range rf.Trees {
		if tree == nil {
			continue
		}
		
		// Extract features used by this tree
		treeFeatures := make([]float64, len(rf.TreeFeatures[i]))
		for j, fIdx := range rf.TreeFeatures[i] {
			treeFeatures[j] = x[fIdx]
		}
		
		predicted, err := tree.PredictRegression(treeFeatures)
		if err != nil {
			continue
		}
		predictions = append(predictions, predicted)
	}

	if len(predictions) == 0 {
		return 0.0, 0.0, 0.0, fmt.Errorf("no valid predictions from trees")
	}

	// Calculate mean
	mean := calculateMean(predictions)

	// Calculate standard deviation
	variance := calculateVariance(predictions, mean)
	stdDev := math.Sqrt(variance)

	// 95% confidence interval
	lower = mean - 1.96*stdDev
	upper = mean + 1.96*stdDev

	return mean, lower, upper, nil
}

// calculateOOBScore calculates out-of-bag score for classification
func (rf *RandomForestClassifier) calculateOOBScore(X [][]float64, y []string) float64 {
	// For simplicity, use a basic implementation
	// In production, track which samples were OOB for each tree
	correct := 0
	total := 0

	for i := range X {
		predicted, _, err := rf.Predict(X[i])
		if err != nil {
			continue
		}
		if predicted == y[i] {
			correct++
		}
		total++
	}

	if total == 0 {
		return 0.0
	}

	return float64(correct) / float64(total)
}

// calculateOOBScoreRegression calculates out-of-bag score for regression
func (rf *RandomForestClassifier) calculateOOBScoreRegression(X [][]float64, y []float64) float64 {
	// Calculate R² score on training data (approximation of OOB)
	sumSquaredError := 0.0
	mean := calculateMean(y)
	sumSquaredTotal := 0.0

	for i := range X {
		predicted, err := rf.PredictRegression(X[i])
		if err != nil {
			continue
		}
		sumSquaredError += (y[i] - predicted) * (y[i] - predicted)
		sumSquaredTotal += (y[i] - mean) * (y[i] - mean)
	}

	if sumSquaredTotal == 0 {
		return 0.0
	}

	return 1.0 - (sumSquaredError / sumSquaredTotal)
}

// GetFeatureImportance calculates feature importance across all trees
func (rf *RandomForestClassifier) GetFeatureImportance() map[string]float64 {
	importance := make(map[string]float64)
	for _, name := range rf.FeatureNames {
		importance[name] = 0.0
	}

	// Average importance across all trees
	for _, tree := range rf.Trees {
		if tree == nil {
			continue
		}
		treeImportance := tree.GetFeatureImportance()
		for name, val := range treeImportance {
			importance[name] += val
		}
	}

	// Normalize by number of trees
	for name := range importance {
		importance[name] /= float64(len(rf.Trees))
	}

	return importance
}

// Save saves the random forest model to a JSON file
func (rf *RandomForestClassifier) Save(path string) error {
	data, err := json.MarshalIndent(rf, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal model: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write model file: %w", err)
	}

	return nil
}

// Load loads a random forest model from a JSON file
func (rf *RandomForestClassifier) Load(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read model file: %w", err)
	}

	if err := json.Unmarshal(data, rf); err != nil {
		return fmt.Errorf("failed to unmarshal model: %w", err)
	}

	// Reinitialize random number generator
	if rf.RandomSeed == 0 {
		rf.RandomSeed = time.Now().UnixNano()
	}
	rf.Rand = rand.New(rand.NewSource(rf.RandomSeed))

	return nil
}

// GetModelInfo returns summary information about the random forest
func (rf *RandomForestClassifier) GetModelInfo() map[string]interface{} {
	avgDepth := 0
	totalNodes := 0

	for _, tree := range rf.Trees {
		if tree != nil {
			avgDepth += tree.GetDepth()
			totalNodes += tree.GetNumNodes()
		}
	}

	if len(rf.Trees) > 0 {
		avgDepth /= len(rf.Trees)
		totalNodes /= len(rf.Trees)
	}

	info := map[string]interface{}{
		"algorithm":         "random_forest",
		"num_trees":         rf.NumTrees,
		"num_features":      rf.NumFeatures,
		"num_classes":       rf.NumClasses,
		"max_depth":         rf.MaxDepth,
		"avg_tree_depth":    avgDepth,
		"avg_nodes_per_tree": totalNodes,
		"max_features":      rf.MaxFeatures,
		"bootstrap":         rf.Bootstrap,
		"oob_score":         rf.OOBScore,
		"min_samples_split": rf.MinSamplesSplit,
		"min_samples_leaf":  rf.MinSamplesLeaf,
		"feature_names":     rf.FeatureNames,
		"classes":           rf.Classes,
		"model_type":        rf.ModelType,
	}

	return info
}

// Validate checks if the model is valid and ready for predictions
func (rf *RandomForestClassifier) Validate() error {
	if len(rf.Trees) == 0 {
		return fmt.Errorf("model has no trees")
	}
	if len(rf.FeatureNames) == 0 {
		return fmt.Errorf("model has no feature names")
	}
	if rf.ModelType == "classification" && len(rf.Classes) == 0 {
		return fmt.Errorf("classification model has no classes")
	}
	if rf.NumFeatures != len(rf.FeatureNames) {
		return fmt.Errorf("num_features mismatch")
	}

	// Check at least some trees are valid
	validTrees := 0
	for _, tree := range rf.Trees {
		if tree != nil && tree.Root != nil {
			validTrees++
		}
	}
	if validTrees == 0 {
		return fmt.Errorf("no valid trees in forest")
	}

	return nil
}
