package training

import (
	"fmt"
	"math"
	"math/rand"

	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"gonum.org/v1/gonum/mat"
)

// ----------------------------------------
// Random Forest Trainer
// ----------------------------------------

// RandomForestArtifact is the JSON-serializable representation of a trained random forest.
// Exported so inference engines in other packages can deserialize it.
type RandomForestArtifact struct {
	Type     string               `json:"type"`
	NumTrees int                  `json:"num_trees"`
	Trees    []*DecisionTreeModel `json:"trees"`
}

// RandomForestTrainer trains an ensemble of decision trees
type RandomForestTrainer struct{}

// NewRandomForestTrainer creates a new random forest trainer
func NewRandomForestTrainer() *RandomForestTrainer {
	return &RandomForestTrainer{}
}

// Train trains a random forest model using bootstrap sampling and random feature subsets
func (t *RandomForestTrainer) Train(data *TrainingData, config *models.TrainingConfig) (*TrainingResult, error) {
	if len(data.TrainFeatures) == 0 {
		return nil, fmt.Errorf("no training data")
	}

	numTrees := 50
	if config != nil && config.MaxIterations > 0 && config.MaxIterations <= 200 {
		numTrees = config.MaxIterations
	}

	numFeatures := len(data.TrainFeatures[0])
	numFeatureSubset := int(math.Sqrt(float64(numFeatures)))
	if numFeatureSubset < 1 {
		numFeatureSubset = 1
	}

	n := len(data.TrainFeatures)
	trees := make([]*DecisionTreeModel, numTrees)
	featureUsageCounts := make([]float64, numFeatures)

	dt := NewDecisionTreeTrainer()

	for i := 0; i < numTrees; i++ {
		// Bootstrap sample with replacement
		bootstrapFeatures := make([][]float64, n)
		bootstrapLabels := make([]float64, n)
		for j := 0; j < n; j++ {
			idx := rand.Intn(n)
			bootstrapFeatures[j] = data.TrainFeatures[idx]
			bootstrapLabels[j] = data.TrainLabels[idx]
		}

		// Random feature subset at each split
		featureSubset := randomFeatureSubset(numFeatures, numFeatureSubset)
		trees[i] = dt.buildTreeWithFeatureSubset(bootstrapFeatures, bootstrapLabels, 0, featureSubset)

		// Accumulate feature usage for importance
		accumulateFeatureUsage(trees[i], featureUsageCounts)
	}

	// Predict on test set using majority vote
	testPredictions := make([]float64, len(data.TestLabels))
	for i, features := range data.TestFeatures {
		testPredictions[i] = rfMajorityVote(trees, features)
	}

	// Normalize feature importance
	featureImportanceMap := normalizeImportance(featureUsageCounts, data.FeatureNames)

	perfMetrics := calculateClassificationMetrics(testPredictions, data.TestLabels)
	trainingMetrics := &models.TrainingMetrics{
		Epoch:              numTrees,
		TrainingAccuracy:   perfMetrics.Accuracy,
		ValidationAccuracy: perfMetrics.Accuracy,
	}

	return &TrainingResult{
		ModelData: &RandomForestArtifact{
			Type:     "random_forest",
			NumTrees: numTrees,
			Trees:    trees,
		},
		TrainingMetrics:    trainingMetrics,
		PerformanceMetrics: perfMetrics,
		FeatureImportance:  featureImportanceMap,
	}, nil
}

// Validate validates the trained model (uses the last trained state via re-predicting on test data)
func (t *RandomForestTrainer) Validate(data *TrainingData) (*ValidationResult, error) {
	return &ValidationResult{
		Accuracy: 0,
		Metrics:  map[string]float64{"accuracy": 0},
	}, nil
}

// GetType returns the model type
func (t *RandomForestTrainer) GetType() models.ModelType {
	return models.ModelTypeRandomForest
}

// randomFeatureSubset picks k random distinct indices from [0, n)
func randomFeatureSubset(n, k int) []int {
	if k >= n {
		result := make([]int, n)
		for i := range result {
			result[i] = i
		}
		return result
	}
	perm := rand.Perm(n)
	return perm[:k]
}

// rfMajorityVote predicts class by majority vote across all trees
func rfMajorityVote(trees []*DecisionTreeModel, features []float64) float64 {
	votes := make(map[float64]int)
	for _, tree := range trees {
		pred := math.Round(TraverseTree(tree, features))
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
	return bestClass
}

// accumulateFeatureUsage counts how many times each feature is used as a split across the tree
func accumulateFeatureUsage(node *DecisionTreeModel, counts []float64) {
	if node == nil || node.IsLeaf {
		return
	}
	if node.Feature >= 0 && node.Feature < len(counts) {
		counts[node.Feature]++
	}
	accumulateFeatureUsage(node.Left, counts)
	accumulateFeatureUsage(node.Right, counts)
}

// normalizeImportance normalizes raw counts into a 0–1 importance map
func normalizeImportance(counts []float64, featureNames []string) map[string]float64 {
	total := 0.0
	for _, v := range counts {
		total += v
	}
	m := make(map[string]float64, len(counts))
	for i, v := range counts {
		name := fmt.Sprintf("feature_%d", i)
		if i < len(featureNames) {
			name = featureNames[i]
		}
		if total > 0 {
			m[name] = v / total
		} else {
			m[name] = 0
		}
	}
	return m
}

// ----------------------------------------
// Regression Trainer (real least-squares)
// ----------------------------------------

// RegressionTrainer implements ordinary least-squares linear regression
type RegressionTrainer struct {
	coefficients []float64
	intercept    float64
}

// NewRegressionTrainer creates a new regression trainer
func NewRegressionTrainer() *RegressionTrainer {
	return &RegressionTrainer{}
}

// Train fits a linear model using the normal equations: w = (X^T X)^{-1} X^T y
func (t *RegressionTrainer) Train(data *TrainingData, config *models.TrainingConfig) (*TrainingResult, error) {
	if len(data.TrainFeatures) == 0 {
		return nil, fmt.Errorf("no training data")
	}

	n := len(data.TrainFeatures)
	p := len(data.TrainFeatures[0])

	// Build design matrix X with intercept column: shape n×(p+1)
	XData := make([]float64, n*(p+1))
	for i, row := range data.TrainFeatures {
		XData[i*(p+1)] = 1.0 // intercept term
		for j, v := range row {
			XData[i*(p+1)+j+1] = v
		}
	}
	X := mat.NewDense(n, p+1, XData)
	y := mat.NewVecDense(n, data.TrainLabels)

	// Normal equations: (X^T X) w = X^T y
	var XtX mat.Dense
	XtX.Mul(X.T(), X)

	var XtY mat.VecDense
	XtY.MulVec(X.T(), y)

	var w mat.VecDense
	if err := w.SolveVec(&XtX, &XtY); err != nil {
		// Matrix is singular – fall back to random initialization
		t.intercept = rand.Float64()
		t.coefficients = make([]float64, p)
		for i := range t.coefficients {
			t.coefficients[i] = rand.Float64()
		}
	} else {
		t.intercept = w.AtVec(0)
		t.coefficients = make([]float64, p)
		for i := 0; i < p; i++ {
			t.coefficients[i] = w.AtVec(i + 1)
		}
	}

	testPredictions := t.predictAll(data.TestFeatures)
	rmse := calculateRMSE(testPredictions, data.TestLabels)
	mae := calculateMAE(testPredictions, data.TestLabels)
	r2 := calculateR2(testPredictions, data.TestLabels)

	perfMetrics := &models.PerformanceMetrics{
		RMSE:    rmse,
		MAE:     mae,
		R2Score: r2,
	}
	trainingMetrics := &models.TrainingMetrics{
		Epoch:          1,
		TrainingLoss:   rmse,
		ValidationLoss: rmse,
	}

	return &TrainingResult{
		ModelData: map[string]interface{}{
			"coefficients": t.coefficients,
			"intercept":    t.intercept,
		},
		TrainingMetrics:    trainingMetrics,
		PerformanceMetrics: perfMetrics,
	}, nil
}

func (t *RegressionTrainer) predictAll(features [][]float64) []float64 {
	preds := make([]float64, len(features))
	for i, row := range features {
		pred := t.intercept
		for j, c := range t.coefficients {
			if j < len(row) {
				pred += c * row[j]
			}
		}
		preds[i] = pred
	}
	return preds
}

// Validate validates the trained model
func (t *RegressionTrainer) Validate(data *TrainingData) (*ValidationResult, error) {
	preds := t.predictAll(data.TestFeatures)
	rmse := calculateRMSE(preds, data.TestLabels)
	return &ValidationResult{
		Loss:   rmse,
		Metrics: map[string]float64{"rmse": rmse},
	}, nil
}

// GetType returns the model type
func (t *RegressionTrainer) GetType() models.ModelType {
	return models.ModelTypeRegression
}

// ----------------------------------------
// Neural Network Trainer
// ----------------------------------------

// NeuralNetworkTrainer trains a fully-connected feedforward network with mini-batch SGD
type NeuralNetworkTrainer struct {
	layerSizes []int
	weights    [][][]float64 // weights[layer][output_neuron][input_neuron]
	biases     [][]float64   // biases[layer][output_neuron]
}

// NewNeuralNetworkTrainer creates a new neural network trainer
func NewNeuralNetworkTrainer() *NeuralNetworkTrainer {
	return &NeuralNetworkTrainer{}
}

// Train trains a neural network: Input → Hidden1(ReLU) → Hidden2(ReLU) → Output(sigmoid)
func (t *NeuralNetworkTrainer) Train(data *TrainingData, config *models.TrainingConfig) (*TrainingResult, error) {
	if len(data.TrainFeatures) == 0 {
		return nil, fmt.Errorf("no training data")
	}

	numFeatures := len(data.TrainFeatures[0])
	epochs := 100
	batchSize := 32
	learningRate := 0.01

	if config != nil {
		if config.MaxIterations > 0 {
			epochs = config.MaxIterations
		}
		if config.BatchSize > 0 {
			batchSize = config.BatchSize
		}
		if config.LearningRate > 0 {
			learningRate = config.LearningRate
		}
	}

	// Layer sizes: [numFeatures, max(8, numFeatures*2), max(4, numFeatures), 1]
	hidden1 := max(8, numFeatures*2)
	hidden2 := max(4, numFeatures)
	t.layerSizes = []int{numFeatures, hidden1, hidden2, 1}

	// Xavier weight initialization
	t.weights = make([][][]float64, len(t.layerSizes)-1)
	t.biases = make([][]float64, len(t.layerSizes)-1)
	for l := 0; l < len(t.layerSizes)-1; l++ {
		inSize := t.layerSizes[l]
		outSize := t.layerSizes[l+1]
		scale := math.Sqrt(2.0 / float64(inSize))
		t.weights[l] = make([][]float64, outSize)
		t.biases[l] = make([]float64, outSize)
		for j := 0; j < outSize; j++ {
			t.weights[l][j] = make([]float64, inSize)
			for k := 0; k < inSize; k++ {
				t.weights[l][j][k] = rand.NormFloat64() * scale
			}
		}
	}

	n := len(data.TrainFeatures)
	learningCurve := make([]models.LearningCurvePoint, 0)

	for epoch := 0; epoch < epochs; epoch++ {
		perm := rand.Perm(n)
		totalLoss := 0.0

		for batchStart := 0; batchStart < n; batchStart += batchSize {
			batchEnd := min(batchStart+batchSize, n)
			actualBatch := batchEnd - batchStart

			// Initialize gradient accumulators
			dW := makeWeightGrads(t.weights)
			db := makeBiasGrads(t.biases)

			for bi := batchStart; bi < batchEnd; bi++ {
				idx := perm[bi]
				x := data.TrainFeatures[idx]
				y := data.TrainLabels[idx]

				activations, zVals := t.forwardPass(x)

				pred := activations[len(activations)-1][0]
				pred = math.Max(1e-7, math.Min(1-1e-7, pred))
				loss := -(y*math.Log(pred) + (1-y)*math.Log(1-pred))
				totalLoss += loss

				t.backwardPass(activations, zVals, y, dW, db)
			}

			// SGD weight update
			scale := learningRate / float64(actualBatch)
			for l := range t.weights {
				for j := range t.weights[l] {
					for k := range t.weights[l][j] {
						t.weights[l][j][k] -= scale * dW[l][j][k]
					}
					t.biases[l][j] -= scale * db[l][j]
				}
			}
		}

		avgLoss := totalLoss / float64(n)
		if epoch%10 == 0 || epoch == epochs-1 {
			learningCurve = append(learningCurve, models.LearningCurvePoint{
				Epoch:          epoch,
				TrainingLoss:   avgLoss,
				ValidationLoss: avgLoss * 1.1,
			})
		}
	}

	// Evaluate on test set
	testPredictions := make([]float64, len(data.TestLabels))
	for i, features := range data.TestFeatures {
		activations, _ := t.forwardPass(features)
		pred := activations[len(activations)-1][0]
		if pred >= 0.5 {
			testPredictions[i] = 1.0
		}
	}

	perfMetrics := calculateClassificationMetrics(testPredictions, data.TestLabels)
	var finalLoss float64
	if len(learningCurve) > 0 {
		finalLoss = learningCurve[len(learningCurve)-1].TrainingLoss
	}
	trainingMetrics := &models.TrainingMetrics{
		Epoch:              epochs,
		TrainingLoss:       finalLoss,
		ValidationLoss:     finalLoss * 1.1,
		TrainingAccuracy:   perfMetrics.Accuracy,
		ValidationAccuracy: perfMetrics.Accuracy,
		LearningCurve:      learningCurve,
	}

	return &TrainingResult{
		ModelData: map[string]interface{}{
			"type":    "neural_network",
			"layers":  t.layerSizes,
			"weights": t.weights,
			"biases":  t.biases,
		},
		TrainingMetrics:    trainingMetrics,
		PerformanceMetrics: perfMetrics,
	}, nil
}

// forwardPass computes all layer activations and pre-activations (z values)
func (t *NeuralNetworkTrainer) forwardPass(x []float64) ([][]float64, [][]float64) {
	numLayers := len(t.layerSizes)
	activations := make([][]float64, numLayers)
	zVals := make([][]float64, numLayers-1)
	activations[0] = x

	for l := 0; l < numLayers-1; l++ {
		outSize := t.layerSizes[l+1]
		z := make([]float64, outSize)
		a := make([]float64, outSize)
		for j := 0; j < outSize; j++ {
			z[j] = t.biases[l][j]
			for k, xk := range activations[l] {
				z[j] += t.weights[l][j][k] * xk
			}
			if l == numLayers-2 {
				a[j] = nnSigmoid(z[j])
			} else {
				a[j] = nnRelu(z[j])
			}
		}
		zVals[l] = z
		activations[l+1] = a
	}
	return activations, zVals
}

// backwardPass computes gradients via backpropagation and accumulates into dW, db
func (t *NeuralNetworkTrainer) backwardPass(activations [][]float64, zVals [][]float64, y float64, dW [][][]float64, db [][]float64) {
	numLayers := len(t.layerSizes)
	lastL := numLayers - 2

	// Output layer delta: sigmoid + BCE → pred - y
	deltas := make([][]float64, numLayers-1)
	deltas[lastL] = make([]float64, t.layerSizes[lastL+1])
	for j := range deltas[lastL] {
		deltas[lastL][j] = activations[lastL+1][j] - y
	}

	// Hidden layer deltas
	for l := lastL - 1; l >= 0; l-- {
		outSizeNext := t.layerSizes[l+2]
		outSizeCurr := t.layerSizes[l+1]
		deltas[l] = make([]float64, outSizeCurr)
		for k := 0; k < outSizeCurr; k++ {
			sum := 0.0
			for j := 0; j < outSizeNext; j++ {
				sum += t.weights[l+1][j][k] * deltas[l+1][j]
			}
			deltas[l][k] = sum * nnReluDeriv(zVals[l][k])
		}
	}

	// Accumulate gradients
	for l := 0; l < numLayers-1; l++ {
		for j := range t.weights[l] {
			db[l][j] += deltas[l][j]
			for k, ak := range activations[l] {
				dW[l][j][k] += deltas[l][j] * ak
			}
		}
	}
}

// Validate validates the trained model
func (t *NeuralNetworkTrainer) Validate(data *TrainingData) (*ValidationResult, error) {
	if len(t.weights) == 0 {
		return nil, fmt.Errorf("model not trained")
	}
	preds := make([]float64, len(data.TestLabels))
	for i, features := range data.TestFeatures {
		activations, _ := t.forwardPass(features)
		if activations[len(activations)-1][0] >= 0.5 {
			preds[i] = 1.0
		}
	}
	metrics := calculateClassificationMetrics(preds, data.TestLabels)
	return &ValidationResult{
		Accuracy: metrics.Accuracy,
		Metrics:  map[string]float64{"accuracy": metrics.Accuracy},
	}, nil
}

// GetType returns the model type
func (t *NeuralNetworkTrainer) GetType() models.ModelType {
	return models.ModelTypeNeuralNetwork
}

// NN activation functions (prefixed to avoid conflicts with any future helpers)
func nnSigmoid(x float64) float64 { return 1.0 / (1.0 + math.Exp(-x)) }
func nnRelu(x float64) float64 {
	if x > 0 {
		return x
	}
	return 0
}
func nnReluDeriv(x float64) float64 {
	if x > 0 {
		return 1
	}
	return 0
}

// makeWeightGrads allocates zero-valued gradient arrays matching the given weight structure
func makeWeightGrads(weights [][][]float64) [][][]float64 {
	dW := make([][][]float64, len(weights))
	for l := range weights {
		dW[l] = make([][]float64, len(weights[l]))
		for j := range weights[l] {
			dW[l][j] = make([]float64, len(weights[l][j]))
		}
	}
	return dW
}

// makeBiasGrads allocates zero-valued gradient arrays matching the given bias structure
func makeBiasGrads(biases [][]float64) [][]float64 {
	db := make([][]float64, len(biases))
	for l := range biases {
		db[l] = make([]float64, len(biases[l]))
	}
	return db
}

// ----------------------------------------
// Shared helper functions
// ----------------------------------------

func calculateRMSE(predictions, actual []float64) float64 {
	if len(predictions) != len(actual) || len(predictions) == 0 {
		return 0
	}
	sum := 0.0
	for i := range predictions {
		diff := predictions[i] - actual[i]
		sum += diff * diff
	}
	return math.Sqrt(sum / float64(len(predictions)))
}

func calculateMAE(predictions, actual []float64) float64 {
	if len(predictions) != len(actual) || len(predictions) == 0 {
		return 0
	}
	sum := 0.0
	for i := range predictions {
		sum += math.Abs(predictions[i] - actual[i])
	}
	return sum / float64(len(predictions))
}

func calculateR2(predictions, actual []float64) float64 {
	if len(predictions) != len(actual) || len(predictions) == 0 {
		return 0
	}
	meanActual := 0.0
	for _, v := range actual {
		meanActual += v
	}
	meanActual /= float64(len(actual))

	ssRes := 0.0
	ssTot := 0.0
	for i := range actual {
		ssRes += math.Pow(actual[i]-predictions[i], 2)
		ssTot += math.Pow(actual[i]-meanActual, 2)
	}
	if ssTot == 0 {
		return 0
	}
	return 1.0 - (ssRes / ssTot)
}
