package metadatastore

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"time"
)

func (s *SQLiteStore) SaveScenario(scenario *models.Scenario) error {
	data, err := json.Marshal(scenario)
	if err != nil {
		return fmt.Errorf("failed to marshal scenario: %w", err)
	}

	query := `
		INSERT OR REPLACE INTO dt_scenarios (
			id, twin_id, name, description, base_state, status, created_at, data
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = s.db.Exec(query,
		scenario.ID,
		scenario.DigitalTwinID,
		scenario.Name,
		scenario.Description,
		scenario.BaseState,
		scenario.Status,
		scenario.CreatedAt.Format(time.RFC3339),
		data,
	)

	if err != nil {
		return fmt.Errorf("failed to save scenario: %w", err)
	}

	return nil
}

// GetScenario retrieves a scenario by ID
func (s *SQLiteStore) GetScenario(id string) (*models.Scenario, error) {
	query := `SELECT data FROM dt_scenarios WHERE id = ?`

	var data []byte
	err := s.db.QueryRow(query, id).Scan(&data)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("scenario not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get scenario: %w", err)
	}

	var scenario models.Scenario
	if err := json.Unmarshal(data, &scenario); err != nil {
		return nil, fmt.Errorf("failed to unmarshal scenario: %w", err)
	}

	return &scenario, nil
}

// ListScenariosByDigitalTwin lists all scenarios for a specific digital twin
func (s *SQLiteStore) ListScenariosByDigitalTwin(twinID string) ([]*models.Scenario, error) {
	query := `SELECT data FROM dt_scenarios WHERE twin_id = ? ORDER BY created_at DESC`

	rows, err := s.db.Query(query, twinID)
	if err != nil {
		return nil, fmt.Errorf("failed to list scenarios: %w", err)
	}
	defer rows.Close()

	var scenarios []*models.Scenario
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, fmt.Errorf("failed to scan scenario: %w", err)
		}

		var scenario models.Scenario
		if err := json.Unmarshal(data, &scenario); err != nil {
			return nil, fmt.Errorf("failed to unmarshal scenario: %w", err)
		}

		scenarios = append(scenarios, &scenario)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating scenarios: %w", err)
	}

	return scenarios, nil
}

// DeleteScenario deletes a scenario
func (s *SQLiteStore) DeleteScenario(id string) error {
	query := `DELETE FROM dt_scenarios WHERE id = ?`
	_, err := s.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete scenario: %w", err)
	}
	return nil
}

// SaveAction saves an action to the database
func (s *SQLiteStore) SaveAction(action *models.Action) error {
	data, err := json.Marshal(action)
	if err != nil {
		return fmt.Errorf("failed to marshal action: %w", err)
	}

	enabled := 0
	if action.Enabled {
		enabled = 1
	}

	query := `
		INSERT OR REPLACE INTO dt_actions (
			id, twin_id, name, enabled, created_at, updated_at, data
		) VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	_, err = s.db.Exec(query,
		action.ID,
		action.DigitalTwinID,
		action.Name,
		enabled,
		action.CreatedAt.Format(time.RFC3339),
		action.UpdatedAt.Format(time.RFC3339),
		data,
	)

	if err != nil {
		return fmt.Errorf("failed to save action: %w", err)
	}

	return nil
}

// GetAction retrieves an action by ID
func (s *SQLiteStore) GetAction(id string) (*models.Action, error) {
	query := `SELECT data FROM dt_actions WHERE id = ?`

	var data []byte
	err := s.db.QueryRow(query, id).Scan(&data)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("action not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get action: %w", err)
	}

	var action models.Action
	if err := json.Unmarshal(data, &action); err != nil {
		return nil, fmt.Errorf("failed to unmarshal action: %w", err)
	}

	return &action, nil
}

// ListActionsByDigitalTwin lists all actions for a specific digital twin
func (s *SQLiteStore) ListActionsByDigitalTwin(twinID string) ([]*models.Action, error) {
	query := `SELECT data FROM dt_actions WHERE twin_id = ? ORDER BY created_at DESC`

	rows, err := s.db.Query(query, twinID)
	if err != nil {
		return nil, fmt.Errorf("failed to list actions: %w", err)
	}
	defer rows.Close()

	var actions []*models.Action
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, fmt.Errorf("failed to scan action: %w", err)
		}

		var action models.Action
		if err := json.Unmarshal(data, &action); err != nil {
			return nil, fmt.Errorf("failed to unmarshal action: %w", err)
		}

		actions = append(actions, &action)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating actions: %w", err)
	}

	return actions, nil
}

// DeleteAction deletes an action
func (s *SQLiteStore) DeleteAction(id string) error {
	query := `DELETE FROM dt_actions WHERE id = ?`
	_, err := s.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete action: %w", err)
	}
	return nil
}

// SavePrediction saves a prediction to the database
func (s *SQLiteStore) SavePrediction(prediction *models.Prediction) error {
	data, err := json.Marshal(prediction)
	if err != nil {
		return fmt.Errorf("failed to marshal prediction: %w", err)
	}

	query := `
		INSERT OR REPLACE INTO dt_predictions (
			id, twin_id, entity_id, model_id, cached_until, created_at, data
		) VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	_, err = s.db.Exec(query,
		prediction.ID,
		prediction.DigitalTwinID,
		prediction.EntityID,
		prediction.ModelID,
		prediction.ExpiresAt.Format(time.RFC3339),
		prediction.CachedAt.Format(time.RFC3339),
		data,
	)

	if err != nil {
		return fmt.Errorf("failed to save prediction: %w", err)
	}

	return nil
}

// GetPrediction retrieves a prediction by ID
func (s *SQLiteStore) GetPrediction(id string) (*models.Prediction, error) {
	query := `SELECT data FROM dt_predictions WHERE id = ?`

	var data []byte
	err := s.db.QueryRow(query, id).Scan(&data)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("prediction not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get prediction: %w", err)
	}

	var prediction models.Prediction
	if err := json.Unmarshal(data, &prediction); err != nil {
		return nil, fmt.Errorf("failed to unmarshal prediction: %w", err)
	}

	return &prediction, nil
}

// ListPredictionsByEntity lists all predictions for a specific entity
func (s *SQLiteStore) ListPredictionsByEntity(entityID string) ([]*models.Prediction, error) {
	query := `SELECT data FROM dt_predictions WHERE entity_id = ? ORDER BY created_at DESC`

	rows, err := s.db.Query(query, entityID)
	if err != nil {
		return nil, fmt.Errorf("failed to list predictions: %w", err)
	}
	defer rows.Close()

	var predictions []*models.Prediction
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, fmt.Errorf("failed to scan prediction: %w", err)
		}

		var prediction models.Prediction
		if err := json.Unmarshal(data, &prediction); err != nil {
			return nil, fmt.Errorf("failed to unmarshal prediction: %w", err)
		}

		predictions = append(predictions, &prediction)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating predictions: %w", err)
	}

	return predictions, nil
}

// ListPredictionsByDigitalTwin lists all predictions for a specific digital twin
func (s *SQLiteStore) ListPredictionsByDigitalTwin(twinID string) ([]*models.Prediction, error) {
	query := `SELECT data FROM dt_predictions WHERE twin_id = ? ORDER BY created_at DESC`

	rows, err := s.db.Query(query, twinID)
	if err != nil {
		return nil, fmt.Errorf("failed to list predictions: %w", err)
	}
	defer rows.Close()

	var predictions []*models.Prediction
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, fmt.Errorf("failed to scan prediction: %w", err)
		}

		var prediction models.Prediction
		if err := json.Unmarshal(data, &prediction); err != nil {
			return nil, fmt.Errorf("failed to unmarshal prediction: %w", err)
		}

		predictions = append(predictions, &prediction)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating predictions: %w", err)
	}

	return predictions, nil
}

// DeletePrediction deletes a prediction
func (s *SQLiteStore) DeletePrediction(id string) error {
	query := `DELETE FROM dt_predictions WHERE id = ?`
	_, err := s.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete prediction: %w", err)
	}
	return nil
}

// DeleteExpiredPredictions deletes all expired predictions for a digital twin
func (s *SQLiteStore) DeleteExpiredPredictions(twinID string) error {
	query := `DELETE FROM dt_predictions WHERE twin_id = ? AND cached_until < ?`
	_, err := s.db.Exec(query, twinID, time.Now().Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("failed to delete expired predictions: %w", err)
	}
	return nil
}
