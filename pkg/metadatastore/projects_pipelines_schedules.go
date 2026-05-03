package metadatastore

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"time"
)

// SaveProject saves a project to the database
func (s *SQLiteStore) SaveProject(project *models.Project) error {
	data, err := json.Marshal(project)
	if err != nil {
		return fmt.Errorf("failed to marshal project: %w", err)
	}

	query := `
		INSERT OR REPLACE INTO projects (id, name, description, status, created_at, updated_at, data)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	_, err = s.db.Exec(query,
		project.ID,
		project.Name,
		project.Description,
		project.Status,
		project.Metadata.CreatedAt,
		project.Metadata.UpdatedAt,
		string(data),
	)

	if err != nil {
		return fmt.Errorf("failed to save project: %w", err)
	}

	return nil
}

// GetProject retrieves a project by ID
func (s *SQLiteStore) GetProject(id string) (*models.Project, error) {
	var data string
	query := `SELECT data FROM projects WHERE id = ?`

	err := s.db.QueryRow(query, id).Scan(&data)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("project not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	var project models.Project
	if err := json.Unmarshal([]byte(data), &project); err != nil {
		return nil, fmt.Errorf("failed to unmarshal project: %w", err)
	}

	return &project, nil
}

// ListProjects lists all projects
func (s *SQLiteStore) ListProjects() ([]*models.Project, error) {
	query := `SELECT data FROM projects ORDER BY created_at DESC`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}
	defer rows.Close()

	projects := make([]*models.Project, 0)
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			continue
		}

		var project models.Project
		if err := json.Unmarshal([]byte(data), &project); err != nil {
			continue
		}

		projects = append(projects, &project)
	}

	return projects, nil
}

// DeleteProject deletes a project
func (s *SQLiteStore) DeleteProject(id string) error {
	query := `DELETE FROM projects WHERE id = ?`
	_, err := s.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete project: %w", err)
	}
	return nil
}

// SavePipeline saves a pipeline to the database
func (s *SQLiteStore) SavePipeline(pipeline *models.Pipeline) error {
	data, err := json.Marshal(pipeline)
	if err != nil {
		return fmt.Errorf("failed to marshal pipeline: %w", err)
	}

	query := `
		INSERT OR REPLACE INTO pipelines (id, project_id, name, type, description, status, created_at, updated_at, data)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = s.db.Exec(query,
		pipeline.ID,
		pipeline.ProjectID,
		pipeline.Name,
		pipeline.Type,
		pipeline.Description,
		pipeline.Status,
		pipeline.CreatedAt,
		pipeline.UpdatedAt,
		string(data),
	)

	if err != nil {
		return fmt.Errorf("failed to save pipeline: %w", err)
	}

	return nil
}

// GetPipeline retrieves a pipeline by ID
func (s *SQLiteStore) GetPipeline(id string) (*models.Pipeline, error) {
	var data string
	query := `SELECT data FROM pipelines WHERE id = ?`

	err := s.db.QueryRow(query, id).Scan(&data)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("pipeline not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get pipeline: %w", err)
	}

	var pipeline models.Pipeline
	if err := json.Unmarshal([]byte(data), &pipeline); err != nil {
		return nil, fmt.Errorf("failed to unmarshal pipeline: %w", err)
	}

	return &pipeline, nil
}

// ListPipelines lists all pipelines
func (s *SQLiteStore) ListPipelines() ([]*models.Pipeline, error) {
	query := `SELECT data FROM pipelines ORDER BY created_at DESC`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list pipelines: %w", err)
	}
	defer rows.Close()

	pipelines := make([]*models.Pipeline, 0)
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			continue
		}

		var pipeline models.Pipeline
		if err := json.Unmarshal([]byte(data), &pipeline); err != nil {
			continue
		}

		pipelines = append(pipelines, &pipeline)
	}

	return pipelines, nil
}

// ListPipelinesByProject lists all pipelines for a specific project
func (s *SQLiteStore) ListPipelinesByProject(projectID string) ([]*models.Pipeline, error) {
	query := `SELECT data FROM pipelines WHERE project_id = ? ORDER BY created_at DESC`

	rows, err := s.db.Query(query, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list pipelines: %w", err)
	}
	defer rows.Close()

	pipelines := make([]*models.Pipeline, 0)
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			continue
		}

		var pipeline models.Pipeline
		if err := json.Unmarshal([]byte(data), &pipeline); err != nil {
			continue
		}

		pipelines = append(pipelines, &pipeline)
	}

	return pipelines, nil
}

// DeletePipeline deletes a pipeline
func (s *SQLiteStore) DeletePipeline(id string) error {
	query := `DELETE FROM pipelines WHERE id = ?`
	_, err := s.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete pipeline: %w", err)
	}
	return nil
}

// GetPipelineCheckpoint retrieves persisted connector state for one pipeline step.
func (s *SQLiteStore) GetPipelineCheckpoint(projectID, pipelineID, stepName, scope string) (*models.PipelineCheckpoint, error) {
	var (
		version   int
		createdAt time.Time
		updatedAt time.Time
		data      string
	)

	query := `
		SELECT version, created_at, updated_at, data
		FROM pipeline_checkpoints
		WHERE project_id = ? AND pipeline_id = ? AND step_name = ? AND scope = ?
	`

	err := s.db.QueryRow(query, projectID, pipelineID, stepName, scope).Scan(&version, &createdAt, &updatedAt, &data)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get pipeline checkpoint: %w", err)
	}

	checkpointData := make(map[string]interface{})
	if err := json.Unmarshal([]byte(data), &checkpointData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal pipeline checkpoint: %w", err)
	}

	return &models.PipelineCheckpoint{
		ProjectID:  projectID,
		PipelineID: pipelineID,
		StepName:   stepName,
		Scope:      scope,
		Version:    version,
		Checkpoint: checkpointData,
		CreatedAt:  createdAt,
		UpdatedAt:  updatedAt,
	}, nil
}

// SavePipelineCheckpoint inserts or updates persisted connector state with optimistic versioning.
func (s *SQLiteStore) SavePipelineCheckpoint(checkpoint *models.PipelineCheckpoint) error {
	if checkpoint == nil {
		return fmt.Errorf("pipeline checkpoint is required")
	}
	if checkpoint.ProjectID == "" || checkpoint.PipelineID == "" || checkpoint.StepName == "" {
		return fmt.Errorf("pipeline checkpoint requires project_id, pipeline_id, and step_name")
	}
	if checkpoint.Checkpoint == nil {
		checkpoint.Checkpoint = map[string]interface{}{}
	}

	return s.retryOnBusy(func() error {
		tx, err := s.db.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()

		var (
			currentVersion int
			createdAt      time.Time
		)

		lookupErr := tx.QueryRow(`
			SELECT version, created_at
			FROM pipeline_checkpoints
			WHERE project_id = ? AND pipeline_id = ? AND step_name = ? AND scope = ?
		`, checkpoint.ProjectID, checkpoint.PipelineID, checkpoint.StepName, checkpoint.Scope).Scan(&currentVersion, &createdAt)

		now := time.Now().UTC()
		switch lookupErr {
		case nil:
			if checkpoint.Version != currentVersion {
				return fmt.Errorf("pipeline checkpoint version conflict: expected %d, got %d", currentVersion, checkpoint.Version)
			}
			checkpoint.Version = currentVersion + 1
			checkpoint.CreatedAt = createdAt
			checkpoint.UpdatedAt = now
		case sql.ErrNoRows:
			if checkpoint.Version != 0 {
				return fmt.Errorf("pipeline checkpoint version conflict: expected 0, got %d", checkpoint.Version)
			}
			checkpoint.Version = 1
			checkpoint.CreatedAt = now
			checkpoint.UpdatedAt = now
		default:
			return fmt.Errorf("failed to read existing pipeline checkpoint: %w", lookupErr)
		}

		data, err := json.Marshal(checkpoint.Checkpoint)
		if err != nil {
			return fmt.Errorf("failed to marshal pipeline checkpoint: %w", err)
		}

		if currentVersion == 0 && checkpoint.Version == 1 {
			_, err = tx.Exec(`
				INSERT INTO pipeline_checkpoints (project_id, pipeline_id, step_name, scope, version, created_at, updated_at, data)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			`, checkpoint.ProjectID, checkpoint.PipelineID, checkpoint.StepName, checkpoint.Scope, checkpoint.Version, checkpoint.CreatedAt, checkpoint.UpdatedAt, string(data))
		} else {
			_, err = tx.Exec(`
				UPDATE pipeline_checkpoints
				SET version = ?, updated_at = ?, data = ?
				WHERE project_id = ? AND pipeline_id = ? AND step_name = ? AND scope = ?
			`, checkpoint.Version, checkpoint.UpdatedAt, string(data), checkpoint.ProjectID, checkpoint.PipelineID, checkpoint.StepName, checkpoint.Scope)
		}
		if err != nil {
			return fmt.Errorf("failed to persist pipeline checkpoint: %w", err)
		}

		return tx.Commit()
	}, 5)
}

// SaveSchedule saves a schedule to the database
func (s *SQLiteStore) SaveSchedule(schedule *models.Schedule) error {
	data, err := json.Marshal(schedule)
	if err != nil {
		return fmt.Errorf("failed to marshal schedule: %w", err)
	}

	var lastRun, nextRun *time.Time
	if schedule.LastRun != nil {
		lastRun = schedule.LastRun
	}
	if schedule.NextRun != nil {
		nextRun = schedule.NextRun
	}

	enabled := 0
	if schedule.Enabled {
		enabled = 1
	}

	query := `
		INSERT OR REPLACE INTO schedules (id, project_id, name, cron_schedule, enabled, created_at, updated_at, last_run, next_run, data)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	// Use retry logic for schedule saves since they can happen concurrently during scheduled execution
	err = s.retryOnBusy(func() error {
		_, execErr := s.db.Exec(query,
			schedule.ID,
			schedule.ProjectID,
			schedule.Name,
			schedule.CronSchedule,
			enabled,
			schedule.CreatedAt,
			schedule.UpdatedAt,
			lastRun,
			nextRun,
			string(data),
		)
		return execErr
	}, 5)

	if err != nil {
		return fmt.Errorf("failed to save schedule: %w", err)
	}

	return nil
}

// GetSchedule retrieves a schedule by ID
func (s *SQLiteStore) GetSchedule(id string) (*models.Schedule, error) {
	var data string
	query := `SELECT data FROM schedules WHERE id = ?`

	err := s.db.QueryRow(query, id).Scan(&data)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("schedule not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get schedule: %w", err)
	}

	var schedule models.Schedule
	if err := json.Unmarshal([]byte(data), &schedule); err != nil {
		return nil, fmt.Errorf("failed to unmarshal schedule: %w", err)
	}

	return &schedule, nil
}

// ListSchedules lists all schedules
func (s *SQLiteStore) ListSchedules() ([]*models.Schedule, error) {
	query := `SELECT data FROM schedules ORDER BY created_at DESC`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list schedules: %w", err)
	}
	defer rows.Close()

	schedules := make([]*models.Schedule, 0)
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			continue
		}

		var schedule models.Schedule
		if err := json.Unmarshal([]byte(data), &schedule); err != nil {
			continue
		}

		schedules = append(schedules, &schedule)
	}

	return schedules, nil
}

// ListSchedulesByProject lists all schedules for a specific project
func (s *SQLiteStore) ListSchedulesByProject(projectID string) ([]*models.Schedule, error) {
	query := `SELECT data FROM schedules WHERE project_id = ? ORDER BY created_at DESC`

	rows, err := s.db.Query(query, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list schedules: %w", err)
	}
	defer rows.Close()

	schedules := make([]*models.Schedule, 0)
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			continue
		}

		var schedule models.Schedule
		if err := json.Unmarshal([]byte(data), &schedule); err != nil {
			continue
		}

		schedules = append(schedules, &schedule)
	}

	return schedules, nil
}

// DeleteSchedule deletes a schedule
func (s *SQLiteStore) DeleteSchedule(id string) error {
	query := `DELETE FROM schedules WHERE id = ?`
	_, err := s.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete schedule: %w", err)
	}
	return nil
}

// SaveWorkTask saves a work task to the database.
func (s *SQLiteStore) SaveWorkTask(task *models.WorkTask) error {
	data, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("failed to marshal work task: %w", err)
	}
	query := `
		INSERT OR REPLACE INTO work_tasks (id, project_id, type, status, submitted_at, data)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	_, err = s.db.Exec(query, task.ID, task.ProjectID, task.Type, task.Status, task.SubmittedAt.UTC(), data)
	if err != nil {
		return fmt.Errorf("failed to save work task: %w", err)
	}
	return nil
}

// GetWorkTask retrieves a work task by ID.
func (s *SQLiteStore) GetWorkTask(id string) (*models.WorkTask, error) {
	var data string
	if err := s.db.QueryRow(`SELECT data FROM work_tasks WHERE id = ?`, id).Scan(&data); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("work task not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get work task: %w", err)
	}
	var task models.WorkTask
	if err := json.Unmarshal([]byte(data), &task); err != nil {
		return nil, fmt.Errorf("failed to unmarshal work task: %w", err)
	}
	return &task, nil
}

// ListWorkTasks lists all persisted work tasks.
func (s *SQLiteStore) ListWorkTasks() ([]*models.WorkTask, error) {
	rows, err := s.db.Query(`SELECT data FROM work_tasks ORDER BY submitted_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("failed to list work tasks: %w", err)
	}
	defer rows.Close()
	tasks := make([]*models.WorkTask, 0)
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			continue
		}
		var task models.WorkTask
		if err := json.Unmarshal([]byte(data), &task); err != nil {
			continue
		}
		tasks = append(tasks, &task)
	}
	return tasks, nil
}

// DeleteWorkTasksByProject deletes all persisted work tasks for one project.
func (s *SQLiteStore) DeleteWorkTasksByProject(projectID string) error {
	if _, err := s.db.Exec(`DELETE FROM work_tasks WHERE project_id = ?`, projectID); err != nil {
		return fmt.Errorf("failed to delete work tasks for project %s: %w", projectID, err)
	}
	return nil
}
