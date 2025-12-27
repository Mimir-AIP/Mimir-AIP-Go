package DigitalTwin

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	ml "github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/ML"
)

// MLPredictor integrates ML models with digital twin simulations
type MLPredictor struct {
	db         *sql.DB
	models     map[string]*LoadedModel
	ontologyID string
}

// LoadedModel represents a loaded ML model ready for predictions
type LoadedModel struct {
	ID           string
	Name         string
	ModelType    string // "classification" or "regression"
	TargetColumn string
	Features     []string
	Classifier   *ml.DecisionTreeClassifier
	LoadedAt     time.Time
}

// PredictionResult contains the result of an ML prediction
type PredictionResult struct {
	EntityURI       string             `json:"entity_uri"`
	PredictedValue  interface{}        `json:"predicted_value"`
	Confidence      float64            `json:"confidence"`
	Probabilities   map[string]float64 `json:"probabilities,omitempty"`
	ModelUsed       string             `json:"model_used"`
	InputFeatures   map[string]float64 `json:"input_features"`
	UsedFallback    bool               `json:"used_fallback"`
	FallbackReason  string             `json:"fallback_reason,omitempty"`
}

// NewMLPredictor creates a new ML predictor for a specific ontology
func NewMLPredictor(db *sql.DB, ontologyID string) *MLPredictor {
	return &MLPredictor{
		db:         db,
		models:     make(map[string]*LoadedModel),
		ontologyID: ontologyID,
	}
}

// LoadModels loads all available ML models for the ontology
func (mp *MLPredictor) LoadModels(ctx context.Context) error {
	if mp.db == nil {
		return fmt.Errorf("database not available")
	}

	query := `
		SELECT id, name, model_type, target_column, feature_columns, model_artifact_path
		FROM ml_models
		WHERE ontology_id = ? AND is_active = 1
	`
	rows, err := mp.db.QueryContext(ctx, query, mp.ontologyID)
	if err != nil {
		return fmt.Errorf("failed to query models: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id, name, modelType, targetColumn, featureColumnsJSON, artifactPath string
		if err := rows.Scan(&id, &name, &modelType, &targetColumn, &featureColumnsJSON, &artifactPath); err != nil {
			continue
		}

		// Parse feature columns
		var features []string
		if err := json.Unmarshal([]byte(featureColumnsJSON), &features); err != nil {
			continue
		}

		// Load classifier
		classifier := &ml.DecisionTreeClassifier{}
		if err := classifier.Load(artifactPath); err != nil {
			continue
		}

		mp.models[targetColumn] = &LoadedModel{
			ID:           id,
			Name:         name,
			ModelType:    modelType,
			TargetColumn: targetColumn,
			Features:     features,
			Classifier:   classifier,
			LoadedAt:     time.Now(),
		}
	}

	return nil
}

// HasModels returns true if any models are loaded
func (mp *MLPredictor) HasModels() bool {
	return len(mp.models) > 0
}

// GetLoadedModels returns info about loaded models
func (mp *MLPredictor) GetLoadedModels() []map[string]interface{} {
	result := []map[string]interface{}{}
	for _, m := range mp.models {
		result = append(result, map[string]interface{}{
			"id":            m.ID,
			"name":          m.Name,
			"model_type":    m.ModelType,
			"target_column": m.TargetColumn,
			"features":      m.Features,
		})
	}
	return result
}

// PredictEntityState uses ML models to predict entity state changes
func (mp *MLPredictor) PredictEntityState(entity *TwinEntity, event *SimulationEvent, currentState *EntityState) (*PredictionResult, error) {
	result := &PredictionResult{
		EntityURI:     entity.URI,
		InputFeatures: make(map[string]float64),
	}

	// Determine which prediction is relevant for this event type
	targetMetric := mp.eventToTargetMetric(event.Type)
	
	// Check if we have a model for this prediction
	model, exists := mp.models[targetMetric]
	if !exists {
		// Try to find any relevant model
		model = mp.findRelevantModel(event.Type, entity.Type)
	}

	if model == nil {
		// Use fallback prediction
		result.UsedFallback = true
		result.FallbackReason = "No ML model available for this prediction"
		result.PredictedValue = mp.fallbackPrediction(event, currentState)
		result.Confidence = 0.5
		return result, nil
	}

	// Build feature vector from entity state and event
	features, err := mp.buildFeatureVector(model, entity, event, currentState)
	if err != nil {
		result.UsedFallback = true
		result.FallbackReason = fmt.Sprintf("Failed to build features: %v", err)
		result.PredictedValue = mp.fallbackPrediction(event, currentState)
		result.Confidence = 0.5
		return result, nil
	}

	result.InputFeatures = features
	result.ModelUsed = model.Name

	// Convert features map to slice in correct order
	featureSlice := make([]float64, len(model.Features))
	for i, fname := range model.Features {
		if val, ok := features[fname]; ok {
			featureSlice[i] = val
		}
	}

	// Make prediction
	if model.ModelType == "regression" {
		value, lower, upper, err := model.Classifier.PredictRegressionWithInterval(featureSlice)
		if err != nil {
			result.UsedFallback = true
			result.FallbackReason = fmt.Sprintf("Prediction failed: %v", err)
			result.PredictedValue = mp.fallbackPrediction(event, currentState)
			result.Confidence = 0.5
			return result, nil
		}
		result.PredictedValue = value
		// Confidence based on interval width relative to value
		intervalWidth := upper - lower
		if value != 0 {
			result.Confidence = 1.0 - (intervalWidth / (value * 2))
			if result.Confidence < 0 {
				result.Confidence = 0.3
			}
		} else {
			result.Confidence = 0.7
		}
	} else {
		predictedClass, _, err := model.Classifier.Predict(featureSlice)
		if err != nil {
			result.UsedFallback = true
			result.FallbackReason = fmt.Sprintf("Prediction failed: %v", err)
			result.PredictedValue = mp.fallbackPrediction(event, currentState)
			result.Confidence = 0.5
			return result, nil
		}
		result.PredictedValue = predictedClass

		// Get probabilities
		proba, err := model.Classifier.PredictProba(featureSlice)
		if err == nil {
			result.Probabilities = proba
			if conf, ok := proba[predictedClass]; ok {
				result.Confidence = conf
			}
		}
	}

	return result, nil
}

// PredictImpactPropagation predicts how impact should propagate through relationships
func (mp *MLPredictor) PredictImpactPropagation(
	sourceEntity *TwinEntity,
	relationship *TwinRelationship,
	targetEntity *TwinEntity,
	sourceChange map[string]interface{},
) (float64, error) {
	// Look for a propagation model
	model := mp.findRelevantModel("propagation", relationship.Type)
	if model == nil {
		// Default propagation based on relationship strength
		return relationship.Strength * 0.7, nil
	}

	// Build features for propagation prediction
	features := map[string]float64{
		"source_utilization":     sourceEntity.State.Utilization,
		"source_capacity":        sourceEntity.State.Capacity,
		"target_utilization":     targetEntity.State.Utilization,
		"target_capacity":        targetEntity.State.Capacity,
		"relationship_strength":  relationship.Strength,
	}

	// Add change magnitude
	if change, ok := sourceChange["utilization"].(float64); ok {
		features["change_magnitude"] = change
	}

	// Convert to slice
	featureSlice := make([]float64, len(model.Features))
	for i, fname := range model.Features {
		if val, ok := features[fname]; ok {
			featureSlice[i] = val
		}
	}

	// Predict propagation factor
	if model.ModelType == "regression" {
		value, _, _, err := model.Classifier.PredictRegressionWithInterval(featureSlice)
		if err != nil {
			return relationship.Strength * 0.7, nil
		}
		// Clamp to valid range
		if value < 0 {
			value = 0
		}
		if value > 1 {
			value = 1
		}
		return value, nil
	}

	// For classification, use confidence as propagation factor
	_, _, err := model.Classifier.Predict(featureSlice)
	if err != nil {
		return relationship.Strength * 0.7, nil
	}
	proba, _ := model.Classifier.PredictProba(featureSlice)
	
	// Use max probability as propagation factor
	maxProba := 0.5
	for _, p := range proba {
		if p > maxProba {
			maxProba = p
		}
	}
	return maxProba * relationship.Strength, nil
}

// PredictSystemMetrics predicts overall system metrics
func (mp *MLPredictor) PredictSystemMetrics(state *TwinState, eventType string) map[string]float64 {
	predictions := make(map[string]float64)

	// Calculate current averages
	var totalUtil, totalCap float64
	for _, es := range state.Entities {
		totalUtil += es.Utilization
		totalCap += es.Capacity
	}
	avgUtil := totalUtil / float64(len(state.Entities))
	avgCap := totalCap / float64(len(state.Entities))

	// Look for system-level models
	for target, model := range mp.models {
		if !strings.HasPrefix(target, "system_") {
			continue
		}

		features := map[string]float64{
			"avg_utilization": avgUtil,
			"avg_capacity":    avgCap,
			"entity_count":    float64(len(state.Entities)),
			"active_events":   float64(len(state.ActiveEvents)),
		}

		featureSlice := make([]float64, len(model.Features))
		for i, fname := range model.Features {
			if val, ok := features[fname]; ok {
				featureSlice[i] = val
			}
		}

		if model.ModelType == "regression" {
			value, _, _, err := model.Classifier.PredictRegressionWithInterval(featureSlice)
			if err == nil {
				predictions[strings.TrimPrefix(target, "system_")] = value
			}
		}
	}

	return predictions
}

// Helper functions

func (mp *MLPredictor) eventToTargetMetric(eventType string) string {
	switch eventType {
	case "entity.unavailable", "external.disruption":
		return "availability"
	case "resource.decrease", "resource.increase":
		return "capacity"
	case "demand.surge", "demand.drop":
		return "utilization"
	case "cost.increase":
		return "cost"
	case "staff.shortage":
		return "workforce"
	default:
		return "impact"
	}
}

func (mp *MLPredictor) findRelevantModel(eventType, entityType string) *LoadedModel {
	// Try to find a model matching the event/entity type
	for target, model := range mp.models {
		targetLower := strings.ToLower(target)
		eventLower := strings.ToLower(eventType)
		entityLower := strings.ToLower(entityType)

		if strings.Contains(targetLower, strings.Split(eventLower, ".")[0]) {
			return model
		}
		if strings.Contains(targetLower, entityLower) {
			return model
		}
	}
	return nil
}

func (mp *MLPredictor) buildFeatureVector(
	model *LoadedModel,
	entity *TwinEntity,
	event *SimulationEvent,
	state *EntityState,
) (map[string]float64, error) {
	features := make(map[string]float64)

	// Extract features from entity state
	features["utilization"] = state.Utilization
	features["capacity"] = state.Capacity
	features["available"] = boolToFloat(state.Available)

	// Add entity metrics if present
	for k, v := range state.Metrics {
		features[k] = v
	}

	// Add event parameters
	if params := event.Parameters; params != nil {
		for k, v := range params {
			switch val := v.(type) {
			case float64:
				features[k] = val
			case int:
				features[k] = float64(val)
			case bool:
				features[k] = boolToFloat(val)
			}
		}
	}

	// Add entity properties
	for k, v := range entity.Properties {
		switch val := v.(type) {
		case float64:
			features["prop_"+k] = val
		case int:
			features["prop_"+k] = float64(val)
		}
	}

	return features, nil
}

func (mp *MLPredictor) fallbackPrediction(event *SimulationEvent, state *EntityState) interface{} {
	// Simple rule-based fallback
	switch event.Type {
	case "entity.unavailable":
		return map[string]interface{}{
			"available":   false,
			"utilization": 0.0,
		}
	case "resource.decrease":
		decrease := 0.3
		if p, ok := event.Parameters["decrease_percent"].(float64); ok {
			decrease = p / 100.0
		}
		return map[string]interface{}{
			"capacity": state.Capacity * (1 - decrease),
		}
	case "resource.increase":
		increase := 0.3
		if p, ok := event.Parameters["increase_percent"].(float64); ok {
			increase = p / 100.0
		}
		return map[string]interface{}{
			"capacity": state.Capacity * (1 + increase),
		}
	case "demand.surge":
		surge := 0.5
		if p, ok := event.Parameters["increase_percent"].(float64); ok {
			surge = p / 100.0
		}
		return map[string]interface{}{
			"utilization": state.Utilization * (1 + surge),
		}
	case "demand.drop":
		drop := 0.3
		if p, ok := event.Parameters["decrease_percent"].(float64); ok {
			drop = p / 100.0
		}
		return map[string]interface{}{
			"utilization": state.Utilization * (1 - drop),
		}
	default:
		return map[string]interface{}{
			"status": "affected",
		}
	}
}

func boolToFloat(b bool) float64 {
	if b {
		return 1.0
	}
	return 0.0
}

