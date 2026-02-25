package models

import "time"

// WorkTaskType represents the type of infrastructure work task to be executed by a worker
type WorkTaskType string

const (
	WorkTaskTypePipelineExecution WorkTaskType = "pipeline_execution"
	WorkTaskTypeMLTraining        WorkTaskType = "ml_training"
	WorkTaskTypeMLInference       WorkTaskType = "ml_inference"
	WorkTaskTypeDigitalTwinUpdate WorkTaskType = "digital_twin_update"
)

// WorkTaskStatus represents the current status of a work task
type WorkTaskStatus string

const (
	WorkTaskStatusQueued    WorkTaskStatus = "queued"
	WorkTaskStatusScheduled WorkTaskStatus = "scheduled"
	WorkTaskStatusSpawned   WorkTaskStatus = "spawned"
	WorkTaskStatusExecuting WorkTaskStatus = "executing"
	WorkTaskStatusCompleted WorkTaskStatus = "completed"
	WorkTaskStatusFailed    WorkTaskStatus = "failed"
	WorkTaskStatusTimeout   WorkTaskStatus = "timeout"
	WorkTaskStatusCancelled WorkTaskStatus = "cancelled"
)

// ResourceRequirements defines the compute resources needed for a work task
type ResourceRequirements struct {
	CPU    string `json:"cpu"`
	Memory string `json:"memory"`
	GPU    bool   `json:"gpu"`
}

// DataAccess defines how a work task accesses input/output data
type DataAccess struct {
	InputDatasets      []string `json:"input_datasets"`
	OutputLocation     string   `json:"output_location"`
	StorageCredentials string   `json:"storage_credentials"`
}

// TaskSpec contains work task-specific parameters
type TaskSpec struct {
	PipelineID string                 `json:"pipeline_id,omitempty"`
	ModelID    string                 `json:"model_id,omitempty"`
	ProjectID  string                 `json:"project_id,omitempty"`
	Parameters map[string]interface{} `json:"parameters"`
}

// WorkTask represents an infrastructure work unit to be executed by a worker
type WorkTask struct {
	ID                   string               `json:"worktask_id"`
	Type                 WorkTaskType         `json:"type"`
	Status               WorkTaskStatus       `json:"status"`
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

// WorkTaskSubmissionRequest represents a request to submit a new work task
type WorkTaskSubmissionRequest struct {
	Type                 WorkTaskType         `json:"type"`
	Priority             int                  `json:"priority"`
	ProjectID            string               `json:"project_id"`
	TaskSpec             TaskSpec             `json:"task_spec"`
	ResourceRequirements ResourceRequirements `json:"resource_requirements"`
	DataAccess           DataAccess           `json:"data_access"`
}

// WorkTaskResult represents the outcome of a completed work task
type WorkTaskResult struct {
	WorkTaskID     string                 `json:"worktask_id"`
	Status         WorkTaskStatus         `json:"status"`
	OutputLocation string                 `json:"output_location"`
	Metadata       map[string]interface{} `json:"metadata"`
	ErrorMessage   string                 `json:"error_message,omitempty"`
}
