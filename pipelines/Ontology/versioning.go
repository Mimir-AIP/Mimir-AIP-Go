package ontology

import (
	"database/sql"
	"fmt"
)

// VersionDiff represents the difference between two versions
type VersionDiff struct {
	FromVersion string           `json:"from_version"`
	ToVersion   string           `json:"to_version"`
	Changes     []OntologyChange `json:"changes"`
	Summary     DiffSummary      `json:"summary"`
}

// DiffSummary provides aggregate statistics for a version diff
type DiffSummary struct {
	ClassesAdded       int `json:"classes_added"`
	ClassesRemoved     int `json:"classes_removed"`
	ClassesModified    int `json:"classes_modified"`
	PropertiesAdded    int `json:"properties_added"`
	PropertiesRemoved  int `json:"properties_removed"`
	PropertiesModified int `json:"properties_modified"`
	TotalChanges       int `json:"total_changes"`
}

// VersioningService manages ontology versioning operations
type VersioningService struct {
	db *sql.DB
}

// NewVersioningService creates a new versioning service
func NewVersioningService(db *sql.DB) *VersioningService {
	return &VersioningService{db: db}
}

// CreateVersion creates a new version snapshot of an ontology
func (v *VersioningService) CreateVersion(ontologyID, version, changelog, createdBy string) (*OntologyVersion, error) {
	// Get previous version
	var previousVersion sql.NullString
	err := v.db.QueryRow(`
		SELECT version FROM ontology_versions 
		WHERE ontology_id = ? 
		ORDER BY created_at DESC 
		LIMIT 1
	`, ontologyID).Scan(&previousVersion)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get previous version: %w", err)
	}

	// Insert new version
	result, err := v.db.Exec(`
		INSERT INTO ontology_versions (ontology_id, version, previous_version, changelog, created_by)
		VALUES (?, ?, ?, ?, ?)
	`, ontologyID, version, previousVersion, changelog, createdBy)
	if err != nil {
		return nil, fmt.Errorf("failed to create version: %w", err)
	}

	versionID, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get version ID: %w", err)
	}

	// Retrieve the created version
	return v.GetVersionByID(int(versionID))
}

// GetVersionByID retrieves a version by its ID
func (v *VersioningService) GetVersionByID(versionID int) (*OntologyVersion, error) {
	version := &OntologyVersion{}
	var previousVersion, migrationStrategy, createdBy sql.NullString

	err := v.db.QueryRow(`
		SELECT id, ontology_id, version, previous_version, changelog, 
		       migration_strategy, created_at, created_by
		FROM ontology_versions
		WHERE id = ?
	`, versionID).Scan(
		&version.ID, &version.OntologyID, &version.Version,
		&previousVersion, &version.Changelog, &migrationStrategy,
		&version.CreatedAt, &createdBy,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get version: %w", err)
	}

	if previousVersion.Valid {
		version.PreviousVersion = previousVersion.String
	}
	if migrationStrategy.Valid {
		version.MigrationStrategy = MigrationStrategy(migrationStrategy.String)
	}
	if createdBy.Valid {
		version.CreatedBy = createdBy.String
	}

	return version, nil
}

// GetVersions retrieves all versions for an ontology
func (v *VersioningService) GetVersions(ontologyID string) ([]OntologyVersion, error) {
	rows, err := v.db.Query(`
		SELECT id, ontology_id, version, previous_version, changelog,
		       migration_strategy, created_at, created_by
		FROM ontology_versions
		WHERE ontology_id = ?
		ORDER BY created_at DESC
	`, ontologyID)
	if err != nil {
		return nil, fmt.Errorf("failed to query versions: %w", err)
	}
	defer rows.Close()

	var versions []OntologyVersion
	for rows.Next() {
		version := OntologyVersion{}
		var previousVersion, migrationStrategy, createdBy sql.NullString

		err := rows.Scan(
			&version.ID, &version.OntologyID, &version.Version,
			&previousVersion, &version.Changelog, &migrationStrategy,
			&version.CreatedAt, &createdBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan version: %w", err)
		}

		if previousVersion.Valid {
			version.PreviousVersion = previousVersion.String
		}
		if migrationStrategy.Valid {
			version.MigrationStrategy = MigrationStrategy(migrationStrategy.String)
		}
		if createdBy.Valid {
			version.CreatedBy = createdBy.String
		}

		versions = append(versions, version)
	}

	return versions, nil
}

// AddChange records a change in a version
func (v *VersioningService) AddChange(versionID int, changeType ChangeType, entityType, entityURI, oldValue, newValue, description string) error {
	_, err := v.db.Exec(`
		INSERT INTO ontology_changes (version_id, change_type, entity_type, entity_uri, old_value, new_value, description)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, versionID, string(changeType), entityType, entityURI, oldValue, newValue, description)

	if err != nil {
		return fmt.Errorf("failed to add change: %w", err)
	}
	return nil
}

// GetChanges retrieves all changes for a version
func (v *VersioningService) GetChanges(versionID int) ([]OntologyChange, error) {
	rows, err := v.db.Query(`
		SELECT id, version_id, change_type, entity_type, entity_uri, 
		       old_value, new_value, description, created_at
		FROM ontology_changes
		WHERE version_id = ?
		ORDER BY created_at ASC
	`, versionID)
	if err != nil {
		return nil, fmt.Errorf("failed to query changes: %w", err)
	}
	defer rows.Close()

	var changes []OntologyChange
	for rows.Next() {
		change := OntologyChange{}
		var oldValue, newValue, description sql.NullString
		var changeTypeStr string

		err := rows.Scan(
			&change.ID, &change.VersionID, &changeTypeStr,
			&change.EntityType, &change.EntityURI, &oldValue,
			&newValue, &description, &change.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan change: %w", err)
		}

		change.ChangeType = ChangeType(changeTypeStr)

		if oldValue.Valid {
			change.OldValue = oldValue.String
		}
		if newValue.Valid {
			change.NewValue = newValue.String
		}
		if description.Valid {
			change.Description = description.String
		}

		changes = append(changes, change)
	}

	return changes, nil
}

// CompareVersions compares two versions and returns the differences
func (v *VersioningService) CompareVersions(ontologyID, version1, version2 string) (*VersionDiff, error) {
	// Get version IDs
	var v1ID, v2ID int
	err := v.db.QueryRow(`
		SELECT id FROM ontology_versions 
		WHERE ontology_id = ? AND version = ?
	`, ontologyID, version1).Scan(&v1ID)
	if err != nil {
		return nil, fmt.Errorf("version1 not found: %w", err)
	}

	err = v.db.QueryRow(`
		SELECT id FROM ontology_versions 
		WHERE ontology_id = ? AND version = ?
	`, ontologyID, version2).Scan(&v2ID)
	if err != nil {
		return nil, fmt.Errorf("version2 not found: %w", err)
	}

	// Ensure v1 is older than v2
	if v1ID > v2ID {
		v1ID, v2ID = v2ID, v1ID
		version1, version2 = version2, version1
	}

	// Get all changes between versions
	rows, err := v.db.Query(`
		SELECT id, version_id, change_type, entity_type, entity_uri,
		       old_value, new_value, description, created_at
		FROM ontology_changes
		WHERE version_id > ? AND version_id <= ?
		ORDER BY version_id ASC, created_at ASC
	`, v1ID, v2ID)
	if err != nil {
		return nil, fmt.Errorf("failed to query changes: %w", err)
	}
	defer rows.Close()

	diff := &VersionDiff{
		FromVersion: version1,
		ToVersion:   version2,
		Changes:     []OntologyChange{},
		Summary:     DiffSummary{},
	}

	for rows.Next() {
		change := OntologyChange{}
		var oldValue, newValue, description sql.NullString
		var changeTypeStr string

		err := rows.Scan(
			&change.ID, &change.VersionID, &changeTypeStr,
			&change.EntityType, &change.EntityURI, &oldValue,
			&newValue, &description, &change.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan change: %w", err)
		}

		change.ChangeType = ChangeType(changeTypeStr)

		if oldValue.Valid {
			change.OldValue = oldValue.String
		}
		if newValue.Valid {
			change.NewValue = newValue.String
		}
		if description.Valid {
			change.Description = description.String
		}

		diff.Changes = append(diff.Changes, change)

		// Update summary statistics
		switch change.EntityType {
		case "class":
			switch string(change.ChangeType) {
			case "added", "add_class":
				diff.Summary.ClassesAdded++
			case "removed", "remove_class":
				diff.Summary.ClassesRemoved++
			case "modified", "modify_class":
				diff.Summary.ClassesModified++
			}
		case "property":
			switch string(change.ChangeType) {
			case "added", "add_property":
				diff.Summary.PropertiesAdded++
			case "removed", "remove_property":
				diff.Summary.PropertiesRemoved++
			case "modified", "modify_property":
				diff.Summary.PropertiesModified++
			}
		}
		diff.Summary.TotalChanges++
	}

	return diff, nil
}

// DeleteVersion deletes a version and all its changes
func (v *VersioningService) DeleteVersion(versionID int) error {
	// Check if this is the latest version
	var count int
	err := v.db.QueryRow(`
		SELECT COUNT(*) FROM ontology_versions
		WHERE id > ?
	`, versionID).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check version order: %w", err)
	}

	if count > 0 {
		return fmt.Errorf("cannot delete version: newer versions exist")
	}

	// Delete the version (cascade will delete changes)
	result, err := v.db.Exec(`
		DELETE FROM ontology_versions WHERE id = ?
	`, versionID)
	if err != nil {
		return fmt.Errorf("failed to delete version: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("version not found")
	}

	return nil
}

// GetLatestVersion retrieves the latest version for an ontology
func (v *VersioningService) GetLatestVersion(ontologyID string) (*OntologyVersion, error) {
	version := &OntologyVersion{}
	var previousVersion, migrationStrategy, createdBy sql.NullString

	err := v.db.QueryRow(`
		SELECT id, ontology_id, version, previous_version, changelog,
		       migration_strategy, created_at, created_by
		FROM ontology_versions
		WHERE ontology_id = ?
		ORDER BY created_at DESC
		LIMIT 1
	`, ontologyID).Scan(
		&version.ID, &version.OntologyID, &version.Version,
		&previousVersion, &version.Changelog, &migrationStrategy,
		&version.CreatedAt, &createdBy,
	)
	if err == sql.ErrNoRows {
		return nil, nil // No versions yet
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get latest version: %w", err)
	}

	if previousVersion.Valid {
		version.PreviousVersion = previousVersion.String
	}
	if migrationStrategy.Valid {
		version.MigrationStrategy = MigrationStrategy(migrationStrategy.String)
	}
	if createdBy.Valid {
		version.CreatedBy = createdBy.String
	}

	return version, nil
}
