package metadatastore

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

func saveOntologyExec(exec sqlExecer, ontology *models.Ontology) error {
	if ontology == nil {
		return fmt.Errorf("ontology cannot be nil")
	}
	isGenerated := 0
	if ontology.IsGenerated {
		isGenerated = 1
	}
	_, err := exec.Exec(`
		INSERT OR REPLACE INTO ontologies (id, project_id, name, description, version, content, status, is_generated, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, ontology.ID, ontology.ProjectID, ontology.Name, ontology.Description, ontology.Version, ontology.Content, ontology.Status, isGenerated, ontology.CreatedAt.Format(time.RFC3339), ontology.UpdatedAt.Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("failed to save ontology: %w", err)
	}
	return nil
}

func saveOntologyCompilationExec(exec sqlExecer, compiled *models.CompiledOntology) error {
	if compiled == nil {
		return fmt.Errorf("compiled ontology cannot be nil")
	}
	data, err := json.Marshal(compiled)
	if err != nil {
		return fmt.Errorf("marshal compiled ontology: %w", err)
	}
	_, err = exec.Exec(`
		INSERT OR REPLACE INTO ontology_compilations (ontology_id, project_id, content_hash, compiled_at, data)
		VALUES (?, ?, ?, ?, ?)
	`, compiled.OntologyID, compiled.ProjectID, compiled.ContentHash, compiled.CompiledAt.Format(time.RFC3339), string(data))
	if err != nil {
		return fmt.Errorf("failed to save ontology compilation: %w", err)
	}
	return nil
}

// SaveOntologyWithCompilation atomically stores raw ontology content and its canonical compiled graph.
func (s *SQLiteStore) SaveOntologyWithCompilation(ontology *models.Ontology, compiled *models.CompiledOntology) error {
	return s.retryOnBusy(func() error {
		tx, err := s.db.Begin()
		if err != nil {
			return fmt.Errorf("begin ontology compilation transaction: %w", err)
		}
		defer tx.Rollback()
		if err := saveOntologyExec(tx, ontology); err != nil {
			return err
		}
		if err := saveOntologyCompilationExec(tx, compiled); err != nil {
			return err
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit ontology compilation transaction: %w", err)
		}
		return nil
	}, 5)
}

// SaveOntologyCompilation persists the canonical compiled graph for an existing ontology.
func (s *SQLiteStore) SaveOntologyCompilation(compiled *models.CompiledOntology) error {
	return s.retryOnBusy(func() error {
		return saveOntologyCompilationExec(s.db, compiled)
	}, 5)
}

// GetCompiledOntology retrieves the canonical compiled graph for an ontology.
func (s *SQLiteStore) GetCompiledOntology(ontologyID string) (*models.CompiledOntology, error) {
	var data string
	if err := s.db.QueryRow(`SELECT data FROM ontology_compilations WHERE ontology_id = ?`, ontologyID).Scan(&data); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("compiled ontology not found: %s", ontologyID)
		}
		return nil, fmt.Errorf("failed to get compiled ontology: %w", err)
	}
	var compiled models.CompiledOntology
	if err := json.Unmarshal([]byte(data), &compiled); err != nil {
		return nil, fmt.Errorf("decode compiled ontology: %w", err)
	}
	return &compiled, nil
}

// DeleteOntologyCompilation removes a compiled ontology graph without deleting the raw ontology document.
func (s *SQLiteStore) DeleteOntologyCompilation(ontologyID string) error {
	_, err := s.db.Exec(`DELETE FROM ontology_compilations WHERE ontology_id = ?`, ontologyID)
	if err != nil {
		return fmt.Errorf("failed to delete ontology compilation: %w", err)
	}
	return nil
}
