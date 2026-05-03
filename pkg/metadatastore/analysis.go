package metadatastore

import (
	"encoding/json"
	"fmt"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

// ── Analysis Runs ─────────────────────────────────────────────────────────────

// SaveAnalysisRun upserts a persisted analysis run.
func (s *SQLiteStore) SaveAnalysisRun(run *models.AnalysisRun) error {
	return s.SaveResolverRun(run, nil)
}

// SaveResolverRun atomically persists one resolver run and its current review items.
func (s *SQLiteStore) SaveResolverRun(run *models.AnalysisRun, items []*models.ReviewItem) error {
	return s.retryOnBusy(func() error {
		tx, err := s.db.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin resolver transaction: %w", err)
		}
		if err := saveAnalysisRunTx(tx, run); err != nil {
			_ = tx.Rollback()
			return err
		}
		for _, item := range items {
			if err := saveReviewItemTx(tx, item); err != nil {
				_ = tx.Rollback()
				return err
			}
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit resolver transaction: %w", err)
		}
		return nil
	}, 5)
}

// GetAnalysisRun retrieves a single persisted analysis run by ID.
func (s *SQLiteStore) GetAnalysisRun(id string) (*models.AnalysisRun, error) {
	var data string
	if err := s.db.QueryRow(`SELECT data FROM analysis_runs WHERE id = ?`, id).Scan(&data); err != nil {
		return nil, fmt.Errorf("analysis run not found: %w", err)
	}
	run := &models.AnalysisRun{}
	if err := json.Unmarshal([]byte(data), run); err != nil {
		return nil, fmt.Errorf("failed to unmarshal analysis run: %w", err)
	}
	return run, nil
}

// ListAnalysisRunsByProject lists persisted analysis runs for one project.
func (s *SQLiteStore) ListAnalysisRunsByProject(projectID string) ([]*models.AnalysisRun, error) {
	rows, err := s.db.Query(`SELECT data FROM analysis_runs WHERE project_id = ? ORDER BY created_at DESC`, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list analysis runs: %w", err)
	}
	defer rows.Close()
	result := make([]*models.AnalysisRun, 0)
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, fmt.Errorf("failed to scan analysis run: %w", err)
		}
		run := &models.AnalysisRun{}
		if err := json.Unmarshal([]byte(data), run); err != nil {
			return nil, fmt.Errorf("failed to unmarshal analysis run: %w", err)
		}
		result = append(result, run)
	}
	return result, nil
}

// DeleteAnalysisRunsByProject deletes persisted analysis runs for one project.
func (s *SQLiteStore) DeleteAnalysisRunsByProject(projectID string) error {
	if _, err := s.db.Exec(`DELETE FROM analysis_runs WHERE project_id = ?`, projectID); err != nil {
		return fmt.Errorf("failed to delete analysis runs for project %s: %w", projectID, err)
	}
	return nil
}

// SaveReviewItem upserts a persisted review item.
func (s *SQLiteStore) SaveReviewItem(item *models.ReviewItem) error {
	return s.retryOnBusy(func() error {
		return saveReviewItemTx(s.db, item)
	}, 5)
}

// GetReviewItem retrieves a single persisted review item by ID.
func (s *SQLiteStore) GetReviewItem(id string) (*models.ReviewItem, error) {
	var data string
	if err := s.db.QueryRow(`SELECT data FROM review_items WHERE id = ?`, id).Scan(&data); err != nil {
		return nil, fmt.Errorf("review item not found: %w", err)
	}
	item := &models.ReviewItem{}
	if err := json.Unmarshal([]byte(data), item); err != nil {
		return nil, fmt.Errorf("failed to unmarshal review item: %w", err)
	}
	return item, nil
}

// GetReviewItemByFindingKey retrieves the most recent persisted review item for one finding key.
func (s *SQLiteStore) GetReviewItemByFindingKey(projectID, findingKey string) (*models.ReviewItem, error) {
	items, err := s.ListReviewItems(projectID)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		if item.FindingKey == findingKey {
			return item, nil
		}
	}
	return nil, nil
}

// ListReviewItems lists review items for a project ordered by recency.
func (s *SQLiteStore) ListReviewItems(projectID string) ([]*models.ReviewItem, error) {
	rows, err := s.db.Query(`SELECT data FROM review_items WHERE project_id = ? ORDER BY created_at DESC`, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list review items: %w", err)
	}
	defer rows.Close()
	result := make([]*models.ReviewItem, 0)
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, fmt.Errorf("failed to scan review item: %w", err)
		}
		item := &models.ReviewItem{}
		if err := json.Unmarshal([]byte(data), item); err != nil {
			return nil, fmt.Errorf("failed to unmarshal review item: %w", err)
		}
		result = append(result, item)
	}
	return result, nil
}

// DeleteReviewItemsByProject deletes persisted review items for one project.
func (s *SQLiteStore) DeleteReviewItemsByProject(projectID string) error {
	if _, err := s.db.Exec(`DELETE FROM review_items WHERE project_id = ?`, projectID); err != nil {
		return fmt.Errorf("failed to delete review items for project %s: %w", projectID, err)
	}
	return nil
}

// SaveInsight upserts a persisted insight.
func (s *SQLiteStore) SaveInsight(insight *models.Insight) error {
	return s.retryOnBusy(func() error {
		return saveInsightTx(s.db, insight)
	}, 5)
}

// SaveInsightRun atomically persists one insight run and all generated insights.
func (s *SQLiteStore) SaveInsightRun(run *models.AnalysisRun, insights []*models.Insight) error {
	return s.retryOnBusy(func() error {
		tx, err := s.db.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin insight transaction: %w", err)
		}
		if err := saveAnalysisRunTx(tx, run); err != nil {
			_ = tx.Rollback()
			return err
		}
		for _, insight := range insights {
			if err := saveInsightTx(tx, insight); err != nil {
				_ = tx.Rollback()
				return err
			}
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit insight transaction: %w", err)
		}
		return nil
	}, 5)
}

// GetInsight retrieves a single persisted insight by ID.
func (s *SQLiteStore) GetInsight(id string) (*models.Insight, error) {
	var data string
	if err := s.db.QueryRow(`SELECT data FROM insights WHERE id = ?`, id).Scan(&data); err != nil {
		return nil, fmt.Errorf("insight not found: %w", err)
	}
	insight := &models.Insight{}
	if err := json.Unmarshal([]byte(data), insight); err != nil {
		return nil, fmt.Errorf("failed to unmarshal insight: %w", err)
	}
	return insight, nil
}

// ListInsightsByProject lists persisted insights for one project ordered by recency.
func (s *SQLiteStore) ListInsightsByProject(projectID string) ([]*models.Insight, error) {
	rows, err := s.db.Query(`SELECT data FROM insights WHERE project_id = ? ORDER BY created_at DESC`, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list insights: %w", err)
	}
	defer rows.Close()
	result := make([]*models.Insight, 0)
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, fmt.Errorf("failed to scan insight: %w", err)
		}
		insight := &models.Insight{}
		if err := json.Unmarshal([]byte(data), insight); err != nil {
			return nil, fmt.Errorf("failed to unmarshal insight: %w", err)
		}
		result = append(result, insight)
	}
	return result, nil
}

// DeleteInsightsByProject deletes persisted insights for one project.
func (s *SQLiteStore) DeleteInsightsByProject(projectID string) error {
	if _, err := s.db.Exec(`DELETE FROM insights WHERE project_id = ?`, projectID); err != nil {
		return fmt.Errorf("failed to delete insights for project %s: %w", projectID, err)
	}
	return nil
}

func saveAnalysisRunTx(exec sqlExecer, run *models.AnalysisRun) error {
	data, err := json.Marshal(run)
	if err != nil {
		return fmt.Errorf("failed to marshal analysis run: %w", err)
	}
	query := `
		INSERT OR REPLACE INTO analysis_runs (id, project_id, kind, status, created_at, completed_at, data)
		VALUES (?, ?, ?, ?, ?, ?, ?)`
	if _, err := exec.Exec(query, run.ID, run.ProjectID, run.Kind, run.Status, run.CreatedAt, run.CompletedAt, string(data)); err != nil {
		return fmt.Errorf("failed to save analysis run: %w", err)
	}
	return nil
}

func saveReviewItemTx(exec sqlExecer, item *models.ReviewItem) error {
	data, err := json.Marshal(item)
	if err != nil {
		return fmt.Errorf("failed to marshal review item: %w", err)
	}
	query := `
		INSERT OR REPLACE INTO review_items (id, project_id, run_id, finding_type, status, confidence, created_at, updated_at, data)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
	if _, err := exec.Exec(query, item.ID, item.ProjectID, item.RunID, item.FindingType, item.Status, item.Confidence, item.CreatedAt, item.UpdatedAt, string(data)); err != nil {
		return fmt.Errorf("failed to save review item: %w", err)
	}
	return nil
}

func saveInsightTx(exec sqlExecer, insight *models.Insight) error {
	data, err := json.Marshal(insight)
	if err != nil {
		return fmt.Errorf("failed to marshal insight: %w", err)
	}
	query := `
		INSERT OR REPLACE INTO insights (id, project_id, run_id, type, severity, confidence, status, created_at, updated_at, data)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	if _, err := exec.Exec(query, insight.ID, insight.ProjectID, insight.RunID, insight.Type, insight.Severity, insight.Confidence, insight.Status, insight.CreatedAt, insight.UpdatedAt, string(data)); err != nil {
		return fmt.Errorf("failed to save insight: %w", err)
	}
	return nil
}
