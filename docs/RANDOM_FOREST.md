# Random Forest Implementation

## Overview

This POC implements a complete Random Forest ensemble model that integrates seamlessly with the existing ML infrastructure. The implementation supports both **classification** and **regression** tasks.

## Features

### Core Functionality
- ✅ **Ensemble Learning**: Combines multiple decision trees using bagging (bootstrap aggregating)
- ✅ **Feature Randomization**: Each tree trains on a random subset of features (√n features per tree)
- ✅ **Bootstrap Sampling**: Creates diverse training sets through sampling with replacement
- ✅ **Parallel Training**: Trains multiple trees concurrently using goroutines
- ✅ **OOB Scoring**: Out-of-bag error estimation for model validation
- ✅ **Classification & Regression**: Full support for both task types
- ✅ **Confidence Intervals**: Provides prediction intervals for regression tasks
- ✅ **Model Persistence**: Save/load models as JSON files
- ✅ **Feature Importance**: Calculates feature importance across the ensemble

### API Integration
- ✅ **RESTful API**: Integrated into existing `/api/v1/models/train` endpoint
- ✅ **Algorithm Selection**: Choose between `decision_tree` or `random_forest`
- ✅ **Backward Compatible**: Works alongside existing decision tree models
- ✅ **Dynamic Loading**: Automatically detects model algorithm during prediction

## Architecture

```
RandomForestClassifier
├── Trees: []*DecisionTreeClassifier  // Ensemble of decision trees
├── TreeFeatures: [][]int              // Feature indices per tree
├── NumTrees: int                      // Number of trees (default: 100)
├── MaxDepth: int                      // Maximum tree depth
├── MaxFeatures: int                   // Features per split (√n)
├── Bootstrap: bool                    // Use bootstrap sampling
└── OOBScore: float64                  // Out-of-bag validation score
```

## Usage

### 1. Training via API

#### Classification Example
```bash
curl -X POST http://localhost:8080/api/v1/models/train \
  -H "Content-Type: application/json" \
  -d '{
    "ontology_id": "iris-ontology",
    "model_name": "Iris Random Forest",
    "model_type": "classification",
    "algorithm": "random_forest",
    "target_column": "species",
    "train_data": [
      ["sepal_length", "sepal_width", "petal_length", "petal_width", "species"],
      ["5.1", "3.5", "1.4", "0.2", "setosa"],
      ["7.0", "3.2", "4.7", "1.4", "versicolor"],
      ["6.3", "3.3", "6.0", "2.5", "virginica"]
    ],
    "config": {
      "num_trees": 100,
      "max_depth": 10,
      "train_test_split": 0.8
    }
  }'
```

#### Regression Example
```bash
curl -X POST http://localhost:8080/api/v1/models/train \
  -H "Content-Type: application/json" \
  -d '{
    "ontology_id": "housing-ontology",
    "model_name": "Housing Price Predictor",
    "model_type": "regression",
    "algorithm": "random_forest",
    "target_column": "price",
    "train_data": [
      ["square_feet", "bedrooms", "bathrooms", "age", "price"],
      ["1500", "3", "2", "10", "250000"],
      ["2000", "4", "3", "5", "350000"]
    ],
    "config": {
      "num_trees": 100,
      "max_depth": 15,
      "train_test_split": 0.75
    }
  }'
```

### 2. Training Programmatically

```go
package main

import (
    "fmt"
    ml "github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/ML"
)

func main() {
    // Prepare data
    X := [][]float64{
        {5.1, 3.5, 1.4, 0.2},
        {7.0, 3.2, 4.7, 1.4},
        {6.3, 3.3, 6.0, 2.5},
    }
    y := []string{"setosa", "versicolor", "virginica"}
    features := []string{"sepal_length", "sepal_width", "petal_length", "petal_width"}

    // Configure training
    config := ml.DefaultTrainingConfig()
    config.NumTrees = 100
    config.MaxDepth = 10
    config.TrainTestSplit = 0.8

    // Train model
    trainer := ml.NewTrainer(config)
    result, err := trainer.TrainRandomForest(X, y, features)
    if err != nil {
        panic(err)
    }

    // Display results
    fmt.Printf("Validation Accuracy: %.2f%%\n", result.ValidateMetrics.Accuracy*100)
    fmt.Printf("OOB Score: %.2f%%\n", result.ModelRF.OOBScore*100)
    
    // Make predictions
    sample := []float64{5.0, 3.4, 1.5, 0.2}
    predicted, confidence, _ := result.ModelRF.Predict(sample)
    fmt.Printf("Predicted: %s (%.2f%% confidence)\n", predicted, confidence*100)

    // Save model
    result.ModelRF.Save("/tmp/my_model.json")
}
```

### 3. Making Predictions

```bash
curl -X POST http://localhost:8080/api/v1/models/{model_id}/predict \
  -H "Content-Type: application/json" \
  -d '{
    "input_data": {
      "sepal_length": 5.1,
      "sepal_width": 3.5,
      "petal_length": 1.4,
      "petal_width": 0.2
    }
  }'
```

Response for classification:
```json
{
  "model_id": "uuid-here",
  "algorithm": "random_forest",
  "predicted_class": "setosa",
  "confidence": 0.96,
  "probabilities": {
    "setosa": 0.96,
    "versicolor": 0.04,
    "virginica": 0.0
  },
  "is_anomaly": false,
  "input_features": {...}
}
```

Response for regression:
```json
{
  "model_id": "uuid-here",
  "algorithm": "random_forest",
  "model_type": "regression",
  "predicted_value": 284.5,
  "confidence_lower": 239.2,
  "confidence_upper": 328.7,
  "confidence_interval": "[239.2, 328.7]",
  "input_features": {...}
}
```

## Configuration Options

| Parameter | Default | Description |
|-----------|---------|-------------|
| `num_trees` | 100 | Number of trees in the forest |
| `max_depth` | 10 | Maximum depth of each tree |
| `min_samples_split` | 2 | Minimum samples required to split a node |
| `min_samples_leaf` | 1 | Minimum samples required at leaf nodes |
| `train_test_split` | 0.8 | Train/validation split ratio |
| `bootstrap` | true | Use bootstrap sampling (bagging) |

## Performance Characteristics

### Advantages over Single Decision Trees
- **Higher Accuracy**: Ensemble reduces overfitting
- **Better Generalization**: Bootstrap sampling creates diversity
- **Robust Predictions**: Averaging reduces variance
- **Feature Importance**: More reliable importance scores

### Parallel Training Performance
- Trains multiple trees concurrently using goroutines
- Near-linear speedup with available CPU cores
- Example: 100 trees trained in ~1-2ms on modern hardware

### Memory Usage
- ~200 bytes per node (similar to single decision tree)
- Total: NumTrees × AvgNodesPerTree × 200 bytes
- Example: 100 trees with 100 nodes each ≈ 2MB

## Testing

Run the test suite:
```bash
go test ./pipelines/ML -v -run TestRandomForest
```

Test coverage:
- ✅ Classification (iris dataset)
- ✅ Regression (numeric targets)
- ✅ OOB score calculation
- ✅ Model persistence (save/load)
- ✅ Performance comparison with single tree
- ✅ Feature importance calculation
- ✅ Parallel training
- ✅ Edge cases and error handling

## Examples

See `examples/random_forest_example.go` for comprehensive examples:
```bash
go run examples/random_forest_example.go
```

This demonstrates:
1. Classification with iris dataset
2. Regression with housing prices
3. Feature importance analysis
4. Confidence intervals for predictions
5. Model persistence

## Implementation Details

### Algorithm Overview
1. **Bootstrap Sampling**: Create N datasets by sampling with replacement
2. **Feature Randomization**: Select √n random features per tree
3. **Parallel Training**: Train trees concurrently using goroutines
4. **Aggregation**: 
   - Classification: Majority voting
   - Regression: Average predictions
5. **OOB Validation**: Use out-of-bag samples for unbiased error estimation

### Key Components

#### RandomForestClassifier
- Main ensemble model structure
- Manages collection of decision trees
- Handles feature mapping per tree

#### Training Methods
- `TrainRandomForest()` - Classification
- `TrainRandomForestRegression()` - Regression
- Uses existing `TrainingConfig` with `num_trees` parameter

#### Prediction Methods
- `Predict()` - Classification with majority voting
- `PredictProba()` - Class probabilities
- `PredictRegression()` - Average prediction
- `PredictRegressionWithInterval()` - With 95% confidence interval

### Feature Importance
Calculated by aggregating importance scores across all trees:
```
importance(feature) = Σ(tree_importance(feature)) / num_trees
```

## Future Enhancements

Potential improvements for production use:
- [ ] Variable importance using permutation
- [ ] Sample weights for imbalanced datasets
- [ ] Feature subsampling at each split (Extra Trees)
- [ ] Warm-start training (add more trees to existing model)
- [ ] Proximity matrix for outlier detection
- [ ] Parallel prediction for batch inference
- [ ] Model compression (prune less important trees)

## References

- Breiman, L. (2001). "Random Forests". Machine Learning. 45 (1): 5–32.
- Hastie, T., Tibshirani, R., & Friedman, J. (2009). The Elements of Statistical Learning.

## License

Same as parent project (see root LICENSE file)
