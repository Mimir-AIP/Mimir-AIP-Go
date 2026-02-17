package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

// FileStore provides file-based persistence for projects, pipelines, and jobs
type FileStore struct {
	basePath  string
	mu        sync.RWMutex
	projects  map[string]*models.Project
	pipelines map[string]*models.Pipeline
	jobs      map[string]*models.ScheduledJob
}

// NewFileStore creates a new file-based storage instance
func NewFileStore(basePath string) (*FileStore, error) {
	// Create base directory if it doesn't exist
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	// Create subdirectories
	for _, dir := range []string{"projects", "pipelines", "jobs"} {
		if err := os.MkdirAll(filepath.Join(basePath, dir), 0755); err != nil {
			return nil, fmt.Errorf("failed to create %s directory: %w", dir, err)
		}
	}

	fs := &FileStore{
		basePath:  basePath,
		projects:  make(map[string]*models.Project),
		pipelines: make(map[string]*models.Pipeline),
		jobs:      make(map[string]*models.ScheduledJob),
	}

	// Load existing data
	if err := fs.loadAll(); err != nil {
		return nil, fmt.Errorf("failed to load existing data: %w", err)
	}

	return fs, nil
}

// loadAll loads all data from disk
func (fs *FileStore) loadAll() error {
	// Load projects
	if err := fs.loadProjects(); err != nil {
		return err
	}

	// Load pipelines
	if err := fs.loadPipelines(); err != nil {
		return err
	}

	// Load jobs
	if err := fs.loadJobs(); err != nil {
		return err
	}

	return nil
}

// loadProjects loads all projects from disk
func (fs *FileStore) loadProjects() error {
	projectsDir := filepath.Join(fs.basePath, "projects")
	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		data, err := os.ReadFile(filepath.Join(projectsDir, entry.Name()))
		if err != nil {
			continue
		}

		var project models.Project
		if err := json.Unmarshal(data, &project); err != nil {
			continue
		}

		fs.projects[project.ID] = &project
	}

	return nil
}

// loadPipelines loads all pipelines from disk
func (fs *FileStore) loadPipelines() error {
	pipelinesDir := filepath.Join(fs.basePath, "pipelines")
	entries, err := os.ReadDir(pipelinesDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		data, err := os.ReadFile(filepath.Join(pipelinesDir, entry.Name()))
		if err != nil {
			continue
		}

		var pipeline models.Pipeline
		if err := json.Unmarshal(data, &pipeline); err != nil {
			continue
		}

		fs.pipelines[pipeline.ID] = &pipeline
	}

	return nil
}

// loadJobs loads all jobs from disk
func (fs *FileStore) loadJobs() error {
	jobsDir := filepath.Join(fs.basePath, "jobs")
	entries, err := os.ReadDir(jobsDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		data, err := os.ReadFile(filepath.Join(jobsDir, entry.Name()))
		if err != nil {
			continue
		}

		var job models.ScheduledJob
		if err := json.Unmarshal(data, &job); err != nil {
			continue
		}

		fs.jobs[job.ID] = &job
	}

	return nil
}

// SaveProject saves a project to disk
func (fs *FileStore) SaveProject(project *models.Project) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	data, err := json.MarshalIndent(project, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal project: %w", err)
	}

	path := filepath.Join(fs.basePath, "projects", project.ID+".json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write project file: %w", err)
	}

	fs.projects[project.ID] = project
	return nil
}

// GetProject retrieves a project by ID
func (fs *FileStore) GetProject(id string) (*models.Project, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	project, ok := fs.projects[id]
	if !ok {
		return nil, fmt.Errorf("project not found: %s", id)
	}

	return project, nil
}

// ListProjects lists all projects
func (fs *FileStore) ListProjects() ([]*models.Project, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	projects := make([]*models.Project, 0, len(fs.projects))
	for _, project := range fs.projects {
		projects = append(projects, project)
	}

	return projects, nil
}

// DeleteProject deletes a project
func (fs *FileStore) DeleteProject(id string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	path := filepath.Join(fs.basePath, "projects", id+".json")
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete project file: %w", err)
	}

	delete(fs.projects, id)
	return nil
}

// SavePipeline saves a pipeline to disk
func (fs *FileStore) SavePipeline(pipeline *models.Pipeline) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	data, err := json.MarshalIndent(pipeline, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal pipeline: %w", err)
	}

	path := filepath.Join(fs.basePath, "pipelines", pipeline.ID+".json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write pipeline file: %w", err)
	}

	fs.pipelines[pipeline.ID] = pipeline
	return nil
}

// GetPipeline retrieves a pipeline by ID
func (fs *FileStore) GetPipeline(id string) (*models.Pipeline, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	pipeline, ok := fs.pipelines[id]
	if !ok {
		return nil, fmt.Errorf("pipeline not found: %s", id)
	}

	return pipeline, nil
}

// ListPipelines lists all pipelines
func (fs *FileStore) ListPipelines() ([]*models.Pipeline, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	pipelines := make([]*models.Pipeline, 0, len(fs.pipelines))
	for _, pipeline := range fs.pipelines {
		pipelines = append(pipelines, pipeline)
	}

	return pipelines, nil
}

// ListPipelinesByProject lists all pipelines for a specific project
func (fs *FileStore) ListPipelinesByProject(projectID string) ([]*models.Pipeline, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	pipelines := make([]*models.Pipeline, 0)
	for _, pipeline := range fs.pipelines {
		if pipeline.ProjectID == projectID {
			pipelines = append(pipelines, pipeline)
		}
	}

	return pipelines, nil
}

// DeletePipeline deletes a pipeline
func (fs *FileStore) DeletePipeline(id string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	path := filepath.Join(fs.basePath, "pipelines", id+".json")
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete pipeline file: %w", err)
	}

	delete(fs.pipelines, id)
	return nil
}

// SaveJob saves a scheduled job to disk
func (fs *FileStore) SaveJob(job *models.ScheduledJob) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	data, err := json.MarshalIndent(job, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal job: %w", err)
	}

	path := filepath.Join(fs.basePath, "jobs", job.ID+".json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write job file: %w", err)
	}

	fs.jobs[job.ID] = job
	return nil
}

// GetJob retrieves a scheduled job by ID
func (fs *FileStore) GetJob(id string) (*models.ScheduledJob, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	job, ok := fs.jobs[id]
	if !ok {
		return nil, fmt.Errorf("job not found: %s", id)
	}

	return job, nil
}

// ListJobs lists all scheduled jobs
func (fs *FileStore) ListJobs() ([]*models.ScheduledJob, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	jobs := make([]*models.ScheduledJob, 0, len(fs.jobs))
	for _, job := range fs.jobs {
		jobs = append(jobs, job)
	}

	return jobs, nil
}

// ListJobsByProject lists all jobs for a specific project
func (fs *FileStore) ListJobsByProject(projectID string) ([]*models.ScheduledJob, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	jobs := make([]*models.ScheduledJob, 0)
	for _, job := range fs.jobs {
		if job.ProjectID == projectID {
			jobs = append(jobs, job)
		}
	}

	return jobs, nil
}

// DeleteJob deletes a scheduled job
func (fs *FileStore) DeleteJob(id string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	path := filepath.Join(fs.basePath, "jobs", id+".json")
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete job file: %w", err)
	}

	delete(fs.jobs, id)
	return nil
}
