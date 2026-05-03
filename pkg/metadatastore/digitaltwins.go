package metadatastore

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"time"
)

func (s *SQLiteStore) SaveDigitalTwin(twin *models.DigitalTwin) error {
	data, err := json.Marshal(twin)
	if err != nil {
		return fmt.Errorf("failed to marshal digital twin: %w", err)
	}

	lastSyncAt := ""
	if twin.LastSyncAt != nil {
		lastSyncAt = twin.LastSyncAt.Format(time.RFC3339)
	}

	query := `
		INSERT INTO digital_twins (
			id, project_id, name, description, ontology_id, status,
			created_at, updated_at, last_synced_at, data
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			project_id = excluded.project_id,
			name = excluded.name,
			description = excluded.description,
			ontology_id = excluded.ontology_id,
			status = excluded.status,
			created_at = excluded.created_at,
			updated_at = excluded.updated_at,
			last_synced_at = excluded.last_synced_at,
			data = excluded.data
	`

	_, err = s.db.Exec(query,
		twin.ID,
		twin.ProjectID,
		twin.Name,
		twin.Description,
		twin.OntologyID,
		twin.Status,
		twin.CreatedAt.Format(time.RFC3339),
		twin.UpdatedAt.Format(time.RFC3339),
		lastSyncAt,
		data,
	)

	if err != nil {
		return fmt.Errorf("failed to save digital twin: %w", err)
	}

	return nil
}

// GetDigitalTwin retrieves a digital twin by ID
func (s *SQLiteStore) GetDigitalTwin(id string) (*models.DigitalTwin, error) {
	query := `SELECT data FROM digital_twins WHERE id = ?`

	var data []byte
	err := s.db.QueryRow(query, id).Scan(&data)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("digital twin not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get digital twin: %w", err)
	}

	var twin models.DigitalTwin
	if err := json.Unmarshal(data, &twin); err != nil {
		return nil, fmt.Errorf("failed to unmarshal digital twin: %w", err)
	}

	return &twin, nil
}

// ListDigitalTwins lists all digital twins
func (s *SQLiteStore) ListDigitalTwins() ([]*models.DigitalTwin, error) {
	query := `SELECT data FROM digital_twins ORDER BY created_at DESC`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list digital twins: %w", err)
	}
	defer rows.Close()

	var twins []*models.DigitalTwin
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, fmt.Errorf("failed to scan digital twin: %w", err)
		}

		var twin models.DigitalTwin
		if err := json.Unmarshal(data, &twin); err != nil {
			return nil, fmt.Errorf("failed to unmarshal digital twin: %w", err)
		}

		twins = append(twins, &twin)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating digital twins: %w", err)
	}

	return twins, nil
}

// ListDigitalTwinsByProject lists all digital twins for a specific project
func (s *SQLiteStore) ListDigitalTwinsByProject(projectID string) ([]*models.DigitalTwin, error) {
	query := `SELECT data FROM digital_twins WHERE project_id = ? ORDER BY created_at DESC`

	rows, err := s.db.Query(query, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list digital twins: %w", err)
	}
	defer rows.Close()

	var twins []*models.DigitalTwin
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, fmt.Errorf("failed to scan digital twin: %w", err)
		}

		var twin models.DigitalTwin
		if err := json.Unmarshal(data, &twin); err != nil {
			return nil, fmt.Errorf("failed to unmarshal digital twin: %w", err)
		}

		twins = append(twins, &twin)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating digital twins: %w", err)
	}

	return twins, nil
}

// DeleteDigitalTwin deletes a digital twin and every persisted record scoped to it.
func (s *SQLiteStore) DeleteDigitalTwin(id string) error {
	return s.retryOnBusy(func() error {
		tx, err := s.db.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin digital twin delete transaction: %w", err)
		}
		defer tx.Rollback()

		deleteStatements := []struct {
			query string
			args  []any
		}{
			{query: `DELETE FROM automations WHERE target_type = ? AND target_id = ?`, args: []any{models.AutomationTargetTypeDigitalTwin, id}},
			{query: `DELETE FROM alert_events WHERE twin_id = ?`, args: []any{id}},
			{query: `DELETE FROM twin_processing_runs WHERE twin_id = ?`, args: []any{id}},
			{query: `DELETE FROM dt_predictions WHERE twin_id = ?`, args: []any{id}},
			{query: `DELETE FROM dt_actions WHERE twin_id = ?`, args: []any{id}},
			{query: `DELETE FROM dt_scenarios WHERE twin_id = ?`, args: []any{id}},
			{query: `DELETE FROM dt_relationship_revisions WHERE twin_id = ?`, args: []any{id}},
			{query: `DELETE FROM dt_snapshots WHERE twin_id = ?`, args: []any{id}},
			{query: `DELETE FROM dt_sync_runs WHERE twin_id = ?`, args: []any{id}},
			{query: `DELETE FROM dt_entity_revisions WHERE twin_id = ?`, args: []any{id}},
			{query: `DELETE FROM dt_entities WHERE twin_id = ?`, args: []any{id}},
			{query: `DELETE FROM digital_twins WHERE id = ?`, args: []any{id}},
		}
		for _, stmt := range deleteStatements {
			if _, err := tx.Exec(stmt.query, stmt.args...); err != nil {
				return fmt.Errorf("failed to execute digital twin delete step %q: %w", stmt.query, err)
			}
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit digital twin delete transaction: %w", err)
		}
		return nil
	}, 5)
}

// SaveAutomation saves one explicit automation policy.
func (s *SQLiteStore) SaveAutomation(automation *models.Automation) error {
	data, err := json.Marshal(automation)
	if err != nil {
		return fmt.Errorf("failed to marshal automation: %w", err)
	}
	enabled := 0
	if automation.Enabled {
		enabled = 1
	}
	query := `
		INSERT OR REPLACE INTO automations (
			id, project_id, target_type, target_id, enabled, trigger_type, action_type, created_at, updated_at, data
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err = s.db.Exec(query,
		automation.ID,
		automation.ProjectID,
		automation.TargetType,
		automation.TargetID,
		enabled,
		automation.TriggerType,
		automation.ActionType,
		automation.CreatedAt,
		automation.UpdatedAt,
		data,
	)
	if err != nil {
		return fmt.Errorf("failed to save automation: %w", err)
	}
	return nil
}

// GetAutomation retrieves one automation by ID.
func (s *SQLiteStore) GetAutomation(id string) (*models.Automation, error) {
	var data []byte
	if err := s.db.QueryRow(`SELECT data FROM automations WHERE id = ?`, id).Scan(&data); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("automation not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get automation: %w", err)
	}
	automation := &models.Automation{}
	if err := json.Unmarshal(data, automation); err != nil {
		return nil, fmt.Errorf("failed to unmarshal automation: %w", err)
	}
	return automation, nil
}

// ListAutomationsByProject lists automations for one project ordered by recency.
func (s *SQLiteStore) ListAutomationsByProject(projectID string) ([]*models.Automation, error) {
	rows, err := s.db.Query(`SELECT data FROM automations WHERE project_id = ? ORDER BY created_at DESC`, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list automations: %w", err)
	}
	defer rows.Close()
	automations := make([]*models.Automation, 0)
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, fmt.Errorf("failed to scan automation: %w", err)
		}
		automation := &models.Automation{}
		if err := json.Unmarshal(data, automation); err != nil {
			return nil, fmt.Errorf("failed to unmarshal automation: %w", err)
		}
		automations = append(automations, automation)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating automations: %w", err)
	}
	return automations, nil
}

// DeleteAutomation deletes one automation by ID.
func (s *SQLiteStore) DeleteAutomation(id string) error {
	if _, err := s.db.Exec(`DELETE FROM automations WHERE id = ?`, id); err != nil {
		return fmt.Errorf("failed to delete automation: %w", err)
	}
	return nil
}

// SaveTwinSyncRun upserts one persisted twin sync run.
func (s *SQLiteStore) SaveTwinSyncRun(run *models.TwinSyncRun) error {
	data, err := json.Marshal(run)
	if err != nil {
		return fmt.Errorf("failed to marshal twin sync run: %w", err)
	}
	query := `
		INSERT INTO dt_sync_runs (id, twin_id, trigger_type, triggered_by, status, started_at, completed_at, ontology_version, reconciliation_strategy, data)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			twin_id = excluded.twin_id,
			trigger_type = excluded.trigger_type,
			triggered_by = excluded.triggered_by,
			status = excluded.status,
			started_at = excluded.started_at,
			completed_at = excluded.completed_at,
			ontology_version = excluded.ontology_version,
			reconciliation_strategy = excluded.reconciliation_strategy,
			data = excluded.data
	`
	_, err = s.db.Exec(query, run.ID, run.DigitalTwinID, run.TriggerType, nullableString(run.TriggeredBy), run.Status, run.StartedAt, run.CompletedAt, nullableString(run.OntologyVersion), nullableString(run.ReconciliationStrategy), data)
	if err != nil {
		return fmt.Errorf("failed to save twin sync run: %w", err)
	}
	return nil
}

// GetTwinSyncRun retrieves one persisted twin sync run by ID.
func (s *SQLiteStore) GetTwinSyncRun(id string) (*models.TwinSyncRun, error) {
	var data []byte
	if err := s.db.QueryRow(`SELECT data FROM dt_sync_runs WHERE id = ?`, id).Scan(&data); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("twin sync run not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get twin sync run: %w", err)
	}
	run := &models.TwinSyncRun{}
	if err := json.Unmarshal(data, run); err != nil {
		return nil, fmt.Errorf("failed to unmarshal twin sync run: %w", err)
	}
	return run, nil
}

// ListTwinSyncRuns lists persisted sync runs for one twin ordered by recency.
func (s *SQLiteStore) ListTwinSyncRuns(twinID string, limit int) ([]*models.TwinSyncRun, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.Query(`SELECT data FROM dt_sync_runs WHERE twin_id = ? ORDER BY started_at DESC LIMIT ?`, twinID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list twin sync runs: %w", err)
	}
	defer rows.Close()
	runs := make([]*models.TwinSyncRun, 0)
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, fmt.Errorf("failed to scan twin sync run: %w", err)
		}
		run := &models.TwinSyncRun{}
		if err := json.Unmarshal(data, run); err != nil {
			return nil, fmt.Errorf("failed to unmarshal twin sync run: %w", err)
		}
		runs = append(runs, run)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating twin sync runs: %w", err)
	}
	return runs, nil
}
