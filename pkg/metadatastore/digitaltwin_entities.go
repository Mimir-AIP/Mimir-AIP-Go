package metadatastore

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"time"
)

func (s *SQLiteStore) SaveEntity(entity *models.Entity) error {
	data, err := json.Marshal(entity)
	if err != nil {
		return fmt.Errorf("failed to marshal entity: %w", err)
	}
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin entity transaction: %w", err)
	}
	defer tx.Rollback()
	query := `
		INSERT INTO dt_entities (
			id, twin_id, entity_type, source_data_id,
			created_at, updated_at, data
		) VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			twin_id = excluded.twin_id,
			entity_type = excluded.entity_type,
			source_data_id = excluded.source_data_id,
			created_at = excluded.created_at,
			updated_at = excluded.updated_at,
			data = excluded.data
	`
	_, err = tx.Exec(query, entity.ID, entity.DigitalTwinID, entity.Type, entity.SourceDataID, entity.CreatedAt.Format(time.RFC3339), entity.UpdatedAt.Format(time.RFC3339), data)
	if err != nil {
		return fmt.Errorf("failed to save entity: %w", err)
	}
	var nextRevision int
	if err := tx.QueryRow(`SELECT COALESCE(MAX(revision), 0) + 1 FROM dt_entity_revisions WHERE entity_id = ?`, entity.ID).Scan(&nextRevision); err != nil {
		return fmt.Errorf("failed to calculate entity revision: %w", err)
	}
	revision := &models.EntityRevision{
		ID:             uuid.New().String(),
		EntityID:       entity.ID,
		DigitalTwinID:  entity.DigitalTwinID,
		Revision:       nextRevision,
		Attributes:     cloneJSONMap(entity.Attributes),
		Modifications:  cloneJSONMap(entity.Modifications),
		ComputedValues: cloneJSONMap(entity.ComputedValues),
		Relationships:  cloneRelationships(entity.Relationships),
		RecordedAt:     entity.UpdatedAt.UTC(),
	}
	revisionData, err := json.Marshal(revision)
	if err != nil {
		return fmt.Errorf("failed to marshal entity revision: %w", err)
	}
	_, err = tx.Exec(`INSERT INTO dt_entity_revisions (id, entity_id, twin_id, revision, recorded_at, data) VALUES (?, ?, ?, ?, ?, ?)`, revision.ID, revision.EntityID, revision.DigitalTwinID, revision.Revision, revision.RecordedAt.Format(time.RFC3339), revisionData)
	if err != nil {
		return fmt.Errorf("failed to save entity revision: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit entity transaction: %w", err)
	}
	return nil
}

// GetEntity retrieves an entity by ID
func (s *SQLiteStore) GetEntity(id string) (*models.Entity, error) {
	query := `SELECT data FROM dt_entities WHERE id = ?`

	var data []byte
	err := s.db.QueryRow(query, id).Scan(&data)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("entity not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get entity: %w", err)
	}

	var entity models.Entity
	if err := json.Unmarshal(data, &entity); err != nil {
		return nil, fmt.Errorf("failed to unmarshal entity: %w", err)
	}

	return &entity, nil
}

// ListEntitiesByDigitalTwin lists all entities for a specific digital twin
func (s *SQLiteStore) ListEntitiesByDigitalTwin(twinID string) ([]*models.Entity, error) {
	query := `SELECT data FROM dt_entities WHERE twin_id = ? ORDER BY created_at DESC`

	rows, err := s.db.Query(query, twinID)
	if err != nil {
		return nil, fmt.Errorf("failed to list entities: %w", err)
	}
	defer rows.Close()

	var entities []*models.Entity
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, fmt.Errorf("failed to scan entity: %w", err)
		}

		var entity models.Entity
		if err := json.Unmarshal(data, &entity); err != nil {
			return nil, fmt.Errorf("failed to unmarshal entity: %w", err)
		}

		entities = append(entities, &entity)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating entities: %w", err)
	}

	return entities, nil
}

// ListEntityRevisions lists historical entity snapshots for one entity.
func (s *SQLiteStore) ListEntityRevisions(entityID string, limit int) ([]*models.EntityRevision, error) {
	query := `SELECT data FROM dt_entity_revisions WHERE entity_id = ? ORDER BY revision DESC`
	args := []interface{}{entityID}
	if limit > 0 {
		query += ` LIMIT ?`
		args = append(args, limit)
	}
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list entity revisions: %w", err)
	}
	defer rows.Close()
	revisions := make([]*models.EntityRevision, 0)
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, fmt.Errorf("failed to scan entity revision: %w", err)
		}
		var revision models.EntityRevision
		if err := json.Unmarshal(data, &revision); err != nil {
			return nil, fmt.Errorf("failed to unmarshal entity revision: %w", err)
		}
		revisions = append(revisions, &revision)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating entity revisions: %w", err)
	}
	return revisions, nil
}

// ListEntitiesByTypeInTwin lists all entities of a specific type within a digital twin.
// Uses the existing twin_id + entity_type indices for efficient lookup.
func (s *SQLiteStore) ListEntitiesByTypeInTwin(twinID, entityType string) ([]*models.Entity, error) {
	query := `SELECT data FROM dt_entities WHERE twin_id = ? AND entity_type = ? ORDER BY created_at DESC`

	rows, err := s.db.Query(query, twinID, entityType)
	if err != nil {
		return nil, fmt.Errorf("failed to list entities by type: %w", err)
	}
	defer rows.Close()

	var entities []*models.Entity
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, fmt.Errorf("failed to scan entity: %w", err)
		}
		var entity models.Entity
		if err := json.Unmarshal(data, &entity); err != nil {
			return nil, fmt.Errorf("failed to unmarshal entity: %w", err)
		}
		entities = append(entities, &entity)
	}
	return entities, nil
}

// DeleteEntity deletes an entity
func (s *SQLiteStore) DeleteEntity(id string) error {
	query := `DELETE FROM dt_entities WHERE id = ?`
	_, err := s.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete entity: %w", err)
	}
	return nil
}

func cloneJSONMap(values map[string]interface{}) map[string]interface{} {
	if values == nil {
		return nil
	}
	cloned := make(map[string]interface{}, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

func cloneRelationships(rels []*models.EntityRelationship) []*models.EntityRelationship {
	if rels == nil {
		return nil
	}
	cloned := make([]*models.EntityRelationship, 0, len(rels))
	for _, rel := range rels {
		if rel == nil {
			continue
		}
		copy := *rel
		copy.Properties = cloneJSONMap(rel.Properties)
		cloned = append(cloned, &copy)
	}
	return cloned
}
