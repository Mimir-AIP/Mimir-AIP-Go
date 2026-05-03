package digitaltwin

import (
	"fmt"
	"strings"

	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

func (s *Service) Query(twinID string, req *models.QueryRequest) (*models.QueryResult, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	twin, err := s.store.GetDigitalTwin(twinID)
	if err != nil {
		return nil, fmt.Errorf("failed to get digital twin: %w", err)
	}

	// Execute SPARQL query
	result, err := s.sparqlEngine.Execute(twin, req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}

	return result, nil
}

// Predict runs ML model prediction on entity data.
//
// When EntityID is provided, the service automatically enriches the input
// feature map with attributes from directly related entities, prefixed by
// their type (e.g. "attendance.days_absent").  This lets models trained on
// cross-source features produce accurate predictions even when called with
// only an entity reference rather than a manually constructed feature vector.
func (s *Service) Predict(twinID string, req *models.PredictionRequest) (*models.Prediction, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	twin, err := s.store.GetDigitalTwin(twinID)
	if err != nil {
		return nil, fmt.Errorf("failed to get digital twin: %w", err)
	}

	// Check if predictions are enabled
	if twin.Config != nil && !twin.Config.EnablePredictions {
		return nil, fmt.Errorf("predictions are disabled for this digital twin")
	}

	// Enrich the input with related entity attributes when an entity ID is
	// given.  This is non-fatal: if enrichment fails the caller-supplied
	// input (or an empty map) is used as-is.
	if req.EntityID != "" {
		if _, err := s.getOwnedEntity(twin.ID, req.EntityID); err != nil {
			return nil, err
		}
		if err := s.enrichPredictionInput(twin.ID, req); err != nil {
			fmt.Printf("Warning: failed to enrich prediction input for entity %s: %v\n", req.EntityID, err)
		}
	}

	// Run prediction through inference engine
	prediction, err := s.inferenceEngine.Predict(twin, req)
	if err != nil {
		return nil, fmt.Errorf("failed to run prediction: %w", err)
	}

	// Save prediction
	if err := s.store.SavePrediction(prediction); err != nil {
		return nil, fmt.Errorf("failed to save prediction: %w", err)
	}

	// Check if prediction triggers any actions
	if err := s.actionManager.EvaluateActions(twin.ID, prediction); err != nil {
		// Log error but don't fail
		fmt.Printf("Warning: failed to evaluate actions: %v\n", err)
	}

	return prediction, nil
}

// enrichPredictionInput loads the target entity's attributes into req.Input
// (if the caller did not supply them) and then merges attributes from all
// directly related entities under type-prefixed keys.
//
// Example: a Student entity with a related AttendanceRecord produces keys like:
//
//	"avg_grade"              → from Student.Attributes
//	"attendancerecord.days_absent" → from the related AttendanceRecord
//
// Related entity attributes are added only when the key is not already present,
// so caller-supplied values always take precedence.  Depth is limited to
// direct (depth-1) relationships to keep the feature vector bounded.
func (s *Service) enrichPredictionInput(twinID string, req *models.PredictionRequest) error {
	entity, err := s.getOwnedEntity(twinID, req.EntityID)
	if err != nil {
		return err
	}

	// Auto-populate input from entity attributes when the caller omitted them.
	if len(req.Input) == 0 {
		req.Input = make(map[string]interface{}, len(entity.Attributes))
		for k, v := range entity.Attributes {
			req.Input[k] = v
		}
	}

	// Merge related entity attributes with a type prefix.
	for _, rel := range entity.Relationships {
		related, err := s.getOwnedEntity(twinID, rel.TargetID)
		if err != nil {
			continue // non-fatal: skip missing or cross-twin related entities
		}
		prefix := strings.ToLower(rel.TargetType) + "."
		for k, v := range related.Attributes {
			key := prefix + k
			if _, exists := req.Input[key]; !exists {
				req.Input[key] = v
			}
		}
	}

	return nil
}

// BatchPredict runs batch predictions
func (s *Service) BatchPredict(twinID string, req *models.BatchPredictionRequest) ([]*models.Prediction, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	twin, err := s.store.GetDigitalTwin(twinID)
	if err != nil {
		return nil, fmt.Errorf("failed to get digital twin: %w", err)
	}

	if twin.Config != nil && !twin.Config.EnablePredictions {
		return nil, fmt.Errorf("predictions are disabled for this digital twin")
	}

	predictions, err := s.inferenceEngine.BatchPredict(twin, req)
	if err != nil {
		return nil, fmt.Errorf("failed to run batch predictions: %w", err)
	}

	// Save all predictions
	for _, prediction := range predictions {
		if err := s.store.SavePrediction(prediction); err != nil {
			fmt.Printf("Warning: failed to save prediction %s: %v\n", prediction.ID, err)
		}
	}

	return predictions, nil
}
