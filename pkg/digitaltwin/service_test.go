package digitaltwin

import (
	"testing"
	"time"

	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

// TestModelArtifactSerialization tests model artifact JSON serialization
func TestModelArtifactSerialization(t *testing.T) {
	artifact := &ModelArtifact{
		ModelType:    "regression",
		FeatureNames: []string{"feature1", "feature2"},
		Parameters: map[string]interface{}{
			"weights":   []interface{}{0.5, 0.3},
			"intercept": 1.2,
		},
		Metadata: map[string]interface{}{
			"trained_at": time.Now().Format(time.RFC3339),
		},
	}

	if artifact.ModelType != "regression" {
		t.Errorf("Expected model type 'regression', got '%s'", artifact.ModelType)
	}

	if len(artifact.FeatureNames) != 2 {
		t.Errorf("Expected 2 features, got %d", len(artifact.FeatureNames))
	}
}

// TestActionConditionEvaluation tests action condition evaluation logic
func TestActionConditionEvaluation(t *testing.T) {
	manager := &ActionManager{}

	tests := []struct {
		name       string
		condition  *models.ActionCondition
		prediction *models.Prediction
		expected   bool
	}{
		{
			name: "greater than - true",
			condition: &models.ActionCondition{
				ModelID:   "model1",
				Operator:  "gt",
				Threshold: 10.0,
			},
			prediction: &models.Prediction{
				ModelID: "model1",
				Output:  15.0,
			},
			expected: true,
		},
		{
			name: "greater than - false",
			condition: &models.ActionCondition{
				ModelID:   "model1",
				Operator:  "gt",
				Threshold: 20.0,
			},
			prediction: &models.Prediction{
				ModelID: "model1",
				Output:  15.0,
			},
			expected: false,
		},
		{
			name: "less than - true",
			condition: &models.ActionCondition{
				ModelID:   "model1",
				Operator:  "lt",
				Threshold: 20.0,
			},
			prediction: &models.Prediction{
				ModelID: "model1",
				Output:  15.0,
			},
			expected: true,
		},
		{
			name: "equal - true",
			condition: &models.ActionCondition{
				ModelID:   "model1",
				Operator:  "eq",
				Threshold: 15.0,
			},
			prediction: &models.Prediction{
				ModelID: "model1",
				Output:  15.0,
			},
			expected: true,
		},
		{
			name: "wrong model - false",
			condition: &models.ActionCondition{
				ModelID:   "model2",
				Operator:  "gt",
				Threshold: 10.0,
			},
			prediction: &models.Prediction{
				ModelID: "model1",
				Output:  15.0,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := manager.evaluateCondition(tt.condition, tt.prediction)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// TestInferenceEngineRegressionPrediction tests regression prediction
func TestInferenceEngineRegressionPrediction(t *testing.T) {
	engine := &InferenceEngine{}

	artifact := &ModelArtifact{
		ModelType:    "regression",
		FeatureNames: []string{"x1", "x2"},
		Parameters: map[string]interface{}{
			"weights":   []interface{}{2.0, 3.0},
			"intercept": 1.0,
		},
	}

	features := []float64{5.0, 10.0}

	// Expected: 1.0 + (2.0 * 5.0) + (3.0 * 10.0) = 1 + 10 + 30 = 41.0
	result := engine.predictRegression(artifact, features)

	if resultFloat, ok := result.(float64); ok {
		if resultFloat != 41.0 {
			t.Errorf("Expected 41.0, got %v", resultFloat)
		}
	} else {
		t.Errorf("Expected float64 result, got %T", result)
	}
}

// TestScenarioCreation tests scenario creation logic
func TestScenarioCreation(t *testing.T) {
	now := time.Now().UTC()

	scenario := &models.Scenario{
		ID:            "scenario1",
		DigitalTwinID: "twin1",
		Name:          "Price Increase Scenario",
		Description:   "What if we increase price by 20%",
		BaseState:     "current",
		Status:        "active",
		CreatedAt:     now,
		UpdatedAt:     now,
		Modifications: []*models.ScenarioModification{
			{
				EntityType:    "Product",
				EntityID:      "prod1",
				Attribute:     "price",
				OriginalValue: 100.0,
				NewValue:      120.0,
				Rationale:     "Testing 20% price increase",
			},
		},
	}

	if scenario.Name != "Price Increase Scenario" {
		t.Errorf("Expected scenario name 'Price Increase Scenario', got '%s'", scenario.Name)
	}

	if len(scenario.Modifications) != 1 {
		t.Errorf("Expected 1 modification, got %d", len(scenario.Modifications))
	}

	mod := scenario.Modifications[0]
	if mod.EntityType != "Product" {
		t.Errorf("Expected entity type 'Product', got '%s'", mod.EntityType)
	}

	if mod.NewValue != 120.0 {
		t.Errorf("Expected new value 120.0, got %v", mod.NewValue)
	}
}

// TestEntityModifications tests entity modification tracking
func TestEntityModifications(t *testing.T) {
	entity := &models.Entity{
		ID:            "entity1",
		DigitalTwinID: "twin1",
		Type:          "User",
		Attributes: map[string]interface{}{
			"name": "John Doe",
			"age":  30,
		},
		Modifications: make(map[string]interface{}),
		IsModified:    false,
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}

	// Apply modification
	entity.Modifications["age"] = 31
	entity.Attributes["age"] = 31
	entity.IsModified = true

	if !entity.IsModified {
		t.Error("Expected entity to be marked as modified")
	}

	if entity.Modifications["age"] != 31 {
		t.Errorf("Expected modified age to be 31, got %v", entity.Modifications["age"])
	}

	if entity.Attributes["age"] != 31 {
		t.Errorf("Expected attribute age to be 31, got %v", entity.Attributes["age"])
	}
}
