package metadatastore

import (
	"database/sql"
	"fmt"

	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

func (s *SQLiteStore) SavePluginArtifact(a *models.PluginArtifact) error {
	query := `
		INSERT OR REPLACE INTO plugin_artifacts
		(id, plugin_kind, plugin_name, source_repository, source_ref, source_commit, digest, local_path,
		 size_bytes, go_version, goos, goarch, host_version, symbol_name, status, error_message, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := s.db.Exec(query,
		a.ID,
		a.PluginKind,
		a.PluginName,
		a.SourceRepository,
		a.SourceRef,
		a.SourceCommit,
		a.Digest,
		a.LocalPath,
		a.SizeBytes,
		a.GoVersion,
		a.GOOS,
		a.GOARCH,
		a.HostVersion,
		a.SymbolName,
		a.Status,
		a.ErrorMessage,
		a.CreatedAt,
		a.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to save plugin artifact: %w", err)
	}
	return nil
}

func (s *SQLiteStore) GetPluginArtifact(id string) (*models.PluginArtifact, error) {
	row := s.db.QueryRow(`
		SELECT id, plugin_kind, plugin_name, source_repository, source_ref, source_commit, digest, local_path,
		       size_bytes, go_version, goos, goarch, host_version, symbol_name, status, error_message, created_at, updated_at
		FROM plugin_artifacts WHERE id = ?
	`, id)
	artifact, err := scanPluginArtifact(row)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("plugin artifact not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get plugin artifact: %w", err)
	}
	return artifact, nil
}

func (s *SQLiteStore) GetActivePluginArtifact(kind models.PluginKind, name string) (*models.PluginArtifact, error) {
	row := s.db.QueryRow(`
		SELECT id, plugin_kind, plugin_name, source_repository, source_ref, source_commit, digest, local_path,
		       size_bytes, go_version, goos, goarch, host_version, symbol_name, status, error_message, created_at, updated_at
		FROM plugin_artifacts
		WHERE plugin_kind = ? AND plugin_name = ? AND status = ?
		ORDER BY updated_at DESC LIMIT 1
	`, kind, name, models.PluginArtifactStatusActive)
	artifact, err := scanPluginArtifact(row)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("active plugin artifact not found: %s/%s", kind, name)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get active plugin artifact: %w", err)
	}
	return artifact, nil
}

func (s *SQLiteStore) ListPluginArtifacts(kind models.PluginKind, name string) ([]*models.PluginArtifact, error) {
	rows, err := s.db.Query(`
		SELECT id, plugin_kind, plugin_name, source_repository, source_ref, source_commit, digest, local_path,
		       size_bytes, go_version, goos, goarch, host_version, symbol_name, status, error_message, created_at, updated_at
		FROM plugin_artifacts
		WHERE plugin_kind = ? AND plugin_name = ?
		ORDER BY updated_at DESC
	`, kind, name)
	if err != nil {
		return nil, fmt.Errorf("failed to list plugin artifacts: %w", err)
	}
	defer rows.Close()

	artifacts := make([]*models.PluginArtifact, 0)
	for rows.Next() {
		artifact, err := scanPluginArtifact(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan plugin artifact: %w", err)
		}
		artifacts = append(artifacts, artifact)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate plugin artifacts: %w", err)
	}
	return artifacts, nil
}

func (s *SQLiteStore) DeletePluginArtifacts(kind models.PluginKind, name string) error {
	_, err := s.db.Exec(`DELETE FROM plugin_artifacts WHERE plugin_kind = ? AND plugin_name = ?`, kind, name)
	if err != nil {
		return fmt.Errorf("failed to delete plugin artifacts: %w", err)
	}
	return nil
}

type pluginArtifactScanner interface {
	Scan(dest ...any) error
}

func scanPluginArtifact(scanner pluginArtifactScanner) (*models.PluginArtifact, error) {
	var artifact models.PluginArtifact
	if err := scanner.Scan(
		&artifact.ID,
		&artifact.PluginKind,
		&artifact.PluginName,
		&artifact.SourceRepository,
		&artifact.SourceRef,
		&artifact.SourceCommit,
		&artifact.Digest,
		&artifact.LocalPath,
		&artifact.SizeBytes,
		&artifact.GoVersion,
		&artifact.GOOS,
		&artifact.GOARCH,
		&artifact.HostVersion,
		&artifact.SymbolName,
		&artifact.Status,
		&artifact.ErrorMessage,
		&artifact.CreatedAt,
		&artifact.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &artifact, nil
}
