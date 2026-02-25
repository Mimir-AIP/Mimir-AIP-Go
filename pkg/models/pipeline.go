package models

import "time"

// PipelineType represents the type of pipeline
type PipelineType string

const (
	PipelineTypeIngestion  PipelineType = "ingestion"
	PipelineTypeProcessing PipelineType = "processing"
	PipelineTypeOutput     PipelineType = "output"
)

// PipelineStatus represents the status of a pipeline
type PipelineStatus string

const (
	PipelineStatusActive   PipelineStatus = "active"
	PipelineStatusInactive PipelineStatus = "inactive"
	PipelineStatusDraft    PipelineStatus = "draft"
)

// Pipeline represents a data processing pipeline
type Pipeline struct {
	ID          string         `json:"id" yaml:"-"`
	ProjectID   string         `json:"project_id" yaml:"-"`
	Name        string         `json:"name" yaml:"name"`
	Type        PipelineType   `json:"type" yaml:"type"`
	Description string         `json:"description" yaml:"description,omitempty"`
	Steps       []PipelineStep `json:"steps" yaml:"steps"`
	Status      PipelineStatus `json:"status" yaml:"status,omitempty"`
	CreatedAt   time.Time      `json:"created_at" yaml:"-"`
	UpdatedAt   time.Time      `json:"updated_at" yaml:"-"`
}

// PipelineStep represents a single step in a pipeline
type PipelineStep struct {
	Name       string                 `json:"name" yaml:"name"`
	Plugin     string                 `json:"plugin" yaml:"plugin"`
	Action     string                 `json:"action" yaml:"action"`
	Parameters map[string]interface{} `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	Output     map[string]string      `json:"output,omitempty" yaml:"output,omitempty"`
}

// PipelineExecution represents a single execution of a pipeline
type PipelineExecution struct {
	ID          string           `json:"id"`
	PipelineID  string           `json:"pipeline_id"`
	ProjectID   string           `json:"project_id"`
	Status      string           `json:"status"`
	StartedAt   time.Time        `json:"started_at"`
	CompletedAt *time.Time       `json:"completed_at,omitempty"`
	Context     *PipelineContext `json:"context,omitempty"`
	Error       string           `json:"error,omitempty"`
	TriggerType string           `json:"trigger_type"` // manual, scheduled, automatic
	TriggeredBy string           `json:"triggered_by,omitempty"`
}

// PipelineContext represents the execution context for a pipeline
type PipelineContext struct {
	Steps   map[string]map[string]interface{} `json:"steps"`
	MaxSize int                               `json:"max_size"`
}

// NewPipelineContext creates a new pipeline context
func NewPipelineContext(maxSize int) *PipelineContext {
	return &PipelineContext{
		Steps:   make(map[string]map[string]interface{}),
		MaxSize: maxSize,
	}
}

// SetStepData stores data for a specific step
func (pc *PipelineContext) SetStepData(stepName string, key string, value interface{}) {
	if pc.Steps[stepName] == nil {
		pc.Steps[stepName] = make(map[string]interface{})
	}
	pc.Steps[stepName][key] = value
}

// GetStepData retrieves data from a specific step
func (pc *PipelineContext) GetStepData(stepName string, key string) (interface{}, bool) {
	if stepData, ok := pc.Steps[stepName]; ok {
		value, exists := stepData[key]
		return value, exists
	}
	return nil, false
}

// GetAllStepData retrieves all data for a specific step
func (pc *PipelineContext) GetAllStepData(stepName string) (map[string]interface{}, bool) {
	stepData, ok := pc.Steps[stepName]
	return stepData, ok
}

// PipelineCreateRequest represents a request to create a new pipeline
type PipelineCreateRequest struct {
	ProjectID   string         `json:"project_id"`
	Name        string         `json:"name" yaml:"name"`
	Type        PipelineType   `json:"type" yaml:"type"`
	Description string         `json:"description" yaml:"description,omitempty"`
	Steps       []PipelineStep `json:"steps" yaml:"steps"`
}

// PipelineUpdateRequest represents a request to update a pipeline
type PipelineUpdateRequest struct {
	Description *string         `json:"description,omitempty"`
	Steps       *[]PipelineStep `json:"steps,omitempty"`
	Status      *PipelineStatus `json:"status,omitempty"`
}

// PipelineExecutionRequest represents a request to execute a pipeline
type PipelineExecutionRequest struct {
	TriggerType string                 `json:"trigger_type"`
	TriggeredBy string                 `json:"triggered_by,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}
