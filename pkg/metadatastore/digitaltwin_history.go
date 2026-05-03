package metadatastore

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"time"
)

func (s *SQLiteStore) SaveRelationshipRevision(revision *models.RelationshipRevision) error {
	data, err := json.Marshal(revision)
	if err != nil {
		return fmt.Errorf("failed to marshal relationship revision: %w", err)
	}
	query := `
		INSERT INTO dt_relationship_revisions (id, twin_id, sync_run_id, source_entity_id, target_entity_id, relationship_type, revision, change_type, recorded_at, data)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err = s.db.Exec(query, revision.ID, revision.DigitalTwinID, nullableString(revision.SyncRunID), revision.SourceEntityID, revision.TargetEntityID, revision.RelationshipType, revision.Revision, revision.ChangeType, revision.RecordedAt.Format(time.RFC3339), data)
	if err != nil {
		return fmt.Errorf("failed to save relationship revision: %w", err)
	}
	return nil
}

// ListRelationshipRevisions lists temporal relationship changes for a twin, optionally scoped to one entity.
func (s *SQLiteStore) ListRelationshipRevisions(twinID, entityID string, limit int) ([]*models.RelationshipRevision, error) {
	if limit <= 0 {
		limit = 50
	}
	query := `SELECT data FROM dt_relationship_revisions WHERE twin_id = ?`
	args := []interface{}{twinID}
	if entityID != "" {
		query += ` AND (source_entity_id = ? OR target_entity_id = ?)`
		args = append(args, entityID, entityID)
	}
	query += ` ORDER BY recorded_at DESC LIMIT ?`
	args = append(args, limit)
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list relationship revisions: %w", err)
	}
	defer rows.Close()
	revisions := make([]*models.RelationshipRevision, 0)
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, fmt.Errorf("failed to scan relationship revision: %w", err)
		}
		revision := &models.RelationshipRevision{}
		if err := json.Unmarshal(data, revision); err != nil {
			return nil, fmt.Errorf("failed to unmarshal relationship revision: %w", err)
		}
		revisions = append(revisions, revision)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating relationship revisions: %w", err)
	}
	return revisions, nil
}

// SaveTwinSnapshot persists one checkpoint snapshot for a digital twin.
func (s *SQLiteStore) SaveTwinSnapshot(snapshot *models.TwinSnapshot) error {
	metadata, err := json.Marshal(snapshot.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal twin snapshot metadata: %w", err)
	}
	query := `
		INSERT INTO dt_snapshots (id, twin_id, sync_run_id, snapshot_kind, entity_state, relationship_state, created_at, entity_revision_high_watermark, relationship_revision_high_watermark, data)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			twin_id = excluded.twin_id,
			sync_run_id = excluded.sync_run_id,
			snapshot_kind = excluded.snapshot_kind,
			entity_state = excluded.entity_state,
			relationship_state = excluded.relationship_state,
			created_at = excluded.created_at,
			entity_revision_high_watermark = excluded.entity_revision_high_watermark,
			relationship_revision_high_watermark = excluded.relationship_revision_high_watermark,
			data = excluded.data
	`
	_, err = s.db.Exec(query, snapshot.ID, snapshot.DigitalTwinID, snapshot.SyncRunID, snapshot.SnapshotKind, snapshot.EntityState, snapshot.RelationshipState, snapshot.CreatedAt.Format(time.RFC3339), snapshot.EntityRevisionHighWatermark, snapshot.RelationshipRevisionHighWatermark, metadata)
	if err != nil {
		return fmt.Errorf("failed to save twin snapshot: %w", err)
	}
	return nil
}

// GetTwinSnapshot retrieves one checkpoint snapshot by ID.
func (s *SQLiteStore) GetTwinSnapshot(id string) (*models.TwinSnapshot, error) {
	return s.getTwinSnapshot(`SELECT id, twin_id, sync_run_id, snapshot_kind, entity_state, relationship_state, created_at, entity_revision_high_watermark, relationship_revision_high_watermark, data FROM dt_snapshots WHERE id = ?`, id)
}

// GetTwinSnapshotByRun retrieves the checkpoint associated with one sync run.
func (s *SQLiteStore) GetTwinSnapshotByRun(twinID, syncRunID string) (*models.TwinSnapshot, error) {
	return s.getTwinSnapshot(`SELECT id, twin_id, sync_run_id, snapshot_kind, entity_state, relationship_state, created_at, entity_revision_high_watermark, relationship_revision_high_watermark, data FROM dt_snapshots WHERE twin_id = ? AND sync_run_id = ?`, twinID, syncRunID)
}

// ListTwinSnapshots lists persisted snapshots for one twin ordered by recency.
func (s *SQLiteStore) ListTwinSnapshots(twinID string, limit int) ([]*models.TwinSnapshot, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.db.Query(`SELECT id, twin_id, sync_run_id, snapshot_kind, entity_state, relationship_state, created_at, entity_revision_high_watermark, relationship_revision_high_watermark, data FROM dt_snapshots WHERE twin_id = ? ORDER BY created_at DESC LIMIT ?`, twinID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list twin snapshots: %w", err)
	}
	defer rows.Close()
	snapshots := make([]*models.TwinSnapshot, 0)
	for rows.Next() {
		snapshot, err := scanTwinSnapshot(rows)
		if err != nil {
			return nil, err
		}
		snapshots = append(snapshots, snapshot)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating twin snapshots: %w", err)
	}
	return snapshots, nil
}

func (s *SQLiteStore) getTwinSnapshot(query string, args ...interface{}) (*models.TwinSnapshot, error) {
	snapshot, err := scanTwinSnapshot(s.db.QueryRow(query, args...))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("twin snapshot not found")
		}
		return nil, err
	}
	return snapshot, nil
}

type snapshotScanner interface {
	Scan(dest ...interface{}) error
}

func scanTwinSnapshot(scanner snapshotScanner) (*models.TwinSnapshot, error) {
	var id string
	var twinID string
	var syncRunID string
	var snapshotKind string
	var entityState []byte
	var relationshipState []byte
	var createdAt string
	var entityHW int
	var relationshipHW int
	var metadata []byte
	if err := scanner.Scan(&id, &twinID, &syncRunID, &snapshotKind, &entityState, &relationshipState, &createdAt, &entityHW, &relationshipHW, &metadata); err != nil {
		return nil, err
	}
	createdAtTime, err := time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse snapshot timestamp: %w", err)
	}
	metadataMap := make(map[string]interface{})
	if len(metadata) > 0 {
		if err := json.Unmarshal(metadata, &metadataMap); err != nil {
			return nil, fmt.Errorf("failed to unmarshal snapshot metadata: %w", err)
		}
	}
	return &models.TwinSnapshot{
		ID:                                id,
		DigitalTwinID:                     twinID,
		SyncRunID:                         syncRunID,
		SnapshotKind:                      snapshotKind,
		EntityState:                       entityState,
		RelationshipState:                 relationshipState,
		CreatedAt:                         createdAtTime,
		EntityRevisionHighWatermark:       entityHW,
		RelationshipRevisionHighWatermark: relationshipHW,
		Metadata:                          metadataMap,
	}, nil
}

// SaveTwinProcessingRun upserts one persisted twin-processing run.
func (s *SQLiteStore) SaveTwinProcessingRun(run *models.TwinProcessingRun) error {
	data, err := json.Marshal(run)
	if err != nil {
		return fmt.Errorf("failed to marshal twin processing run: %w", err)
	}
	query := `
		INSERT OR REPLACE INTO twin_processing_runs (
			id, project_id, twin_id, status, trigger_type, automation_id, requested_at, started_at, completed_at, data
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err = s.db.Exec(query,
		run.ID,
		run.ProjectID,
		run.DigitalTwinID,
		run.Status,
		run.TriggerType,
		nullableString(run.AutomationID),
		run.RequestedAt,
		run.StartedAt,
		run.CompletedAt,
		data,
	)
	if err != nil {
		return fmt.Errorf("failed to save twin processing run: %w", err)
	}
	return nil
}

// GetTwinProcessingRun retrieves one persisted twin-processing run by ID.
func (s *SQLiteStore) GetTwinProcessingRun(id string) (*models.TwinProcessingRun, error) {
	var data []byte
	if err := s.db.QueryRow(`SELECT data FROM twin_processing_runs WHERE id = ?`, id).Scan(&data); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("twin processing run not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get twin processing run: %w", err)
	}
	run := &models.TwinProcessingRun{}
	if err := json.Unmarshal(data, run); err != nil {
		return nil, fmt.Errorf("failed to unmarshal twin processing run: %w", err)
	}
	return run, nil
}

// GetActiveTwinProcessingRun retrieves the newest queued or running run for one twin.
func (s *SQLiteStore) GetActiveTwinProcessingRun(twinID string) (*models.TwinProcessingRun, error) {
	var data []byte
	err := s.db.QueryRow(`
		SELECT data FROM twin_processing_runs
		WHERE twin_id = ? AND status IN (?, ?)
		ORDER BY requested_at DESC LIMIT 1
	`, twinID, models.TwinProcessingRunStatusQueued, models.TwinProcessingRunStatusRunning).Scan(&data)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get active twin processing run: %w", err)
	}
	run := &models.TwinProcessingRun{}
	if err := json.Unmarshal(data, run); err != nil {
		return nil, fmt.Errorf("failed to unmarshal twin processing run: %w", err)
	}
	return run, nil
}

// ListTwinProcessingRunsByDigitalTwin lists persisted runs for one twin ordered by recency.
func (s *SQLiteStore) ListTwinProcessingRunsByDigitalTwin(twinID string, limit int) ([]*models.TwinProcessingRun, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.Query(`SELECT data FROM twin_processing_runs WHERE twin_id = ? ORDER BY requested_at DESC LIMIT ?`, twinID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list twin processing runs: %w", err)
	}
	defer rows.Close()
	runs := make([]*models.TwinProcessingRun, 0)
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, fmt.Errorf("failed to scan twin processing run: %w", err)
		}
		run := &models.TwinProcessingRun{}
		if err := json.Unmarshal(data, run); err != nil {
			return nil, fmt.Errorf("failed to unmarshal twin processing run: %w", err)
		}
		runs = append(runs, run)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating twin processing runs: %w", err)
	}
	return runs, nil
}

// DeleteTwinProcessingRun deletes one persisted run by ID.
func (s *SQLiteStore) DeleteTwinProcessingRun(id string) error {
	if _, err := s.db.Exec(`DELETE FROM twin_processing_runs WHERE id = ?`, id); err != nil {
		return fmt.Errorf("failed to delete twin processing run: %w", err)
	}
	return nil
}

// SaveAlertEvent persists one append-only alert event.
func (s *SQLiteStore) SaveAlertEvent(event *models.AlertEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal alert event: %w", err)
	}
	query := `
		INSERT OR REPLACE INTO alert_events (
			id, project_id, twin_id, processing_run_id, severity, category, created_at, triggered_export_pipeline_id, triggered_work_task_id, data
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err = s.db.Exec(query,
		event.ID,
		event.ProjectID,
		event.DigitalTwinID,
		event.ProcessingRunID,
		event.Severity,
		event.Category,
		event.CreatedAt,
		nullableString(event.TriggeredExportPipelineID),
		nullableString(event.TriggeredWorkTaskID),
		data,
	)
	if err != nil {
		return fmt.Errorf("failed to save alert event: %w", err)
	}
	return nil
}

// GetAlertEvent retrieves one persisted alert event by ID.
func (s *SQLiteStore) GetAlertEvent(id string) (*models.AlertEvent, error) {
	var data []byte
	if err := s.db.QueryRow(`SELECT data FROM alert_events WHERE id = ?`, id).Scan(&data); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("alert event not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get alert event: %w", err)
	}
	event := &models.AlertEvent{}
	if err := json.Unmarshal(data, event); err != nil {
		return nil, fmt.Errorf("failed to unmarshal alert event: %w", err)
	}
	return event, nil
}

// ListAlertEventsByDigitalTwin lists alert events for one twin ordered by recency.
func (s *SQLiteStore) ListAlertEventsByDigitalTwin(twinID string, limit int) ([]*models.AlertEvent, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.Query(`SELECT data FROM alert_events WHERE twin_id = ? ORDER BY created_at DESC LIMIT ?`, twinID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list alert events: %w", err)
	}
	defer rows.Close()
	events := make([]*models.AlertEvent, 0)
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, fmt.Errorf("failed to scan alert event: %w", err)
		}
		event := &models.AlertEvent{}
		if err := json.Unmarshal(data, event); err != nil {
			return nil, fmt.Errorf("failed to unmarshal alert event: %w", err)
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating alert events: %w", err)
	}
	return events, nil
}

// DeleteAlertEvent deletes one alert event by ID.
func (s *SQLiteStore) DeleteAlertEvent(id string) error {
	if _, err := s.db.Exec(`DELETE FROM alert_events WHERE id = ?`, id); err != nil {
		return fmt.Errorf("failed to delete alert event: %w", err)
	}
	return nil
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}
