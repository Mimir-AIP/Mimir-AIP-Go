package models

import "time"

// JobType represents the type of job to be executed
type JobType string

const (
	JobTypePipelineExecution JobType = "pipeline_execution"
	JobTypeMLTraining        JobType = "ml_training"
	JobTypeMLInference       JobType = "ml_inference"
	JobTypeDigitalTwinUpdate JobType = "digital_twin_update"
)

// JobStatus represents the current status of a job
type JobStatus string

const (
	JobStatusQueued    JobStatus = "queued"
	JobStatusScheduled JobStatus = "scheduled"
	JobStatusSpawned   JobStatus = "spawned"
	JobStatusExecuting JobStatus = "executing"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
	JobStatusTimeout   JobStatus = "timeout"
	JobStatusCancelled JobStatus = "cancelled"
)

// ResourceRequirements defines the compute resources needed for a job
type ResourceRequirements struct {
	CPU    string `json:"cpu"`
	Memory string `json:"memory"`
	GPU    bool   `json:"gpu"`
}

// DataAccess defines how a job accesses input/output data
type DataAccess struct {
	InputDatasets      []string `json:"input_datasets"`
	OutputLocation     string   `json:"output_location"`
	StorageCredentials string   `json:"storage_credentials"`
}

// TaskSpec contains job-specific parameters
type TaskSpec struct {
	PipelineID string                 `json:"pipeline_id,omitempty"`
	ModelID    string                 `json:"model_id,omitempty"`
	ProjectID  string                 `json:"project_id,omitempty"`
	Parameters map[string]interface{} `json:"parameters"`
}

// Job represents a work unit to be executed by a worker
type Job struct {
	ID                   string               `json:"job_id"`
	Type                 JobType              `json:"type"`
	Status               JobStatus            `json:"status"`
	Priority             int                  `json:"priority"`
	SubmittedAt          time.Time            `json:"submitted_at"`
	StartedAt            *time.Time           `json:"started_at,omitempty"`
	CompletedAt          *time.Time           `json:"completed_at,omitempty"`
	ProjectID            string               `json:"project_id"`
	TaskSpec             TaskSpec             `json:"task_spec"`
	ResourceRequirements ResourceRequirements `json:"resource_requirements"`
	DataAccess           DataAccess           `json:"data_access"`
	ErrorMessage         string               `json:"error_message,omitempty"`
	KubernetesJobName    string               `json:"kubernetes_job_name,omitempty"`
}

// JobSubmissionRequest represents a request to submit a new job
type JobSubmissionRequest struct {
	Type                 JobType              `json:"type"`
	Priority             int                  `json:"priority"`
	ProjectID            string               `json:"project_id"`
	TaskSpec             TaskSpec             `json:"task_spec"`
	ResourceRequirements ResourceRequirements `json:"resource_requirements"`
	DataAccess           DataAccess           `json:"data_access"`
}

// JobResult represents the outcome of a completed job
type JobResult struct {
	JobID          string                 `json:"job_id"`
	Status         JobStatus              `json:"status"`
	OutputLocation string                 `json:"output_location"`
	Metadata       map[string]interface{} `json:"metadata"`
	ErrorMessage   string                 `json:"error_message,omitempty"`
}
