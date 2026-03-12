package models

import "time"

type AutomationTargetType string

const (
	AutomationTargetTypePipeline    AutomationTargetType = "pipeline"
	AutomationTargetTypeDigitalTwin AutomationTargetType = "digital_twin"
)

type AutomationTriggerType string

const (
	AutomationTriggerTypePipelineCompleted AutomationTriggerType = "pipeline_completed"
	AutomationTriggerTypeManual            AutomationTriggerType = "manual"
)

type AutomationActionType string

const (
	AutomationActionTypeProcessTwin           AutomationActionType = "process_twin"
	AutomationActionTypeTriggerExportPipeline AutomationActionType = "trigger_export_pipeline"
)

// Automation stores one explicit, user-configured automation policy.
// Target, trigger, condition, and action are intentionally generic so the
// orchestrator stays domain-agnostic while product surfaces can present more
// opinionated workflows such as "Twin Automation".
type Automation struct {
	ID              string                `json:"id"`
	ProjectID       string                `json:"project_id"`
	Name            string                `json:"name"`
	Description     string                `json:"description,omitempty"`
	Enabled         bool                  `json:"enabled"`
	TargetType      AutomationTargetType  `json:"target_type"`
	TargetID        string                `json:"target_id"`
	TriggerType     AutomationTriggerType `json:"trigger_type"`
	TriggerConfig   map[string]any        `json:"trigger_config,omitempty"`
	ConditionConfig map[string]any        `json:"condition_config,omitempty"`
	ActionType      AutomationActionType  `json:"action_type"`
	ActionConfig    map[string]any        `json:"action_config,omitempty"`
	CreatedAt       time.Time             `json:"created_at"`
	UpdatedAt       time.Time             `json:"updated_at"`
}

// AutomationCreateRequest creates one explicit automation policy.
type AutomationCreateRequest struct {
	ProjectID       string                `json:"project_id"`
	Name            string                `json:"name"`
	Description     string                `json:"description,omitempty"`
	Enabled         *bool                 `json:"enabled,omitempty"`
	TargetType      AutomationTargetType  `json:"target_type"`
	TargetID        string                `json:"target_id"`
	TriggerType     AutomationTriggerType `json:"trigger_type"`
	TriggerConfig   map[string]any        `json:"trigger_config,omitempty"`
	ConditionConfig map[string]any        `json:"condition_config,omitempty"`
	ActionType      AutomationActionType  `json:"action_type"`
	ActionConfig    map[string]any        `json:"action_config,omitempty"`
}

// AutomationUpdateRequest updates one explicit automation policy.
type AutomationUpdateRequest struct {
	Name            *string                `json:"name,omitempty"`
	Description     *string                `json:"description,omitempty"`
	Enabled         *bool                  `json:"enabled,omitempty"`
	TriggerType     *AutomationTriggerType `json:"trigger_type,omitempty"`
	TriggerConfig   map[string]any         `json:"trigger_config,omitempty"`
	ConditionConfig map[string]any         `json:"condition_config,omitempty"`
	ActionType      *AutomationActionType  `json:"action_type,omitempty"`
	ActionConfig    map[string]any         `json:"action_config,omitempty"`
}

type TwinProcessingRunStatus string

const (
	TwinProcessingRunStatusQueued     TwinProcessingRunStatus = "queued"
	TwinProcessingRunStatusRunning    TwinProcessingRunStatus = "running"
	TwinProcessingRunStatusCompleted  TwinProcessingRunStatus = "completed"
	TwinProcessingRunStatusFailed     TwinProcessingRunStatus = "failed"
	TwinProcessingRunStatusSuperseded TwinProcessingRunStatus = "superseded"
)

type TwinProcessingTriggerType string

const (
	TwinProcessingTriggerTypeManual            TwinProcessingTriggerType = "manual"
	TwinProcessingTriggerTypePipelineCompleted TwinProcessingTriggerType = "pipeline_completed"
)

type TwinProcessingStageStatus string

const (
	TwinProcessingStageStatusQueued    TwinProcessingStageStatus = "queued"
	TwinProcessingStageStatusRunning   TwinProcessingStageStatus = "running"
	TwinProcessingStageStatusCompleted TwinProcessingStageStatus = "completed"
	TwinProcessingStageStatusFailed    TwinProcessingStageStatus = "failed"
	TwinProcessingStageStatusSkipped   TwinProcessingStageStatus = "skipped"
)

// TwinProcessingStageState captures one stage within a processing run.
type TwinProcessingStageState struct {
	Status      TwinProcessingStageStatus `json:"status"`
	StartedAt   *time.Time                `json:"started_at,omitempty"`
	CompletedAt *time.Time                `json:"completed_at,omitempty"`
	Metrics     map[string]any            `json:"metrics,omitempty"`
	Error       string                    `json:"error,omitempty"`
}

// TwinProcessingRun is the operator-facing execution record for processing a twin.
type TwinProcessingRun struct {
	ID                string                              `json:"id"`
	ProjectID         string                              `json:"project_id"`
	DigitalTwinID     string                              `json:"digital_twin_id"`
	Status            TwinProcessingRunStatus             `json:"status"`
	TriggerType       TwinProcessingTriggerType           `json:"trigger_type"`
	TriggerRef        string                              `json:"trigger_ref,omitempty"`
	AutomationID      string                              `json:"automation_id,omitempty"`
	RequestedAt       time.Time                           `json:"requested_at"`
	StartedAt         *time.Time                          `json:"started_at,omitempty"`
	CompletedAt       *time.Time                          `json:"completed_at,omitempty"`
	RerunRequested    bool                                `json:"rerun_requested,omitempty"`
	SupersededByRunID string                              `json:"superseded_by_run_id,omitempty"`
	StageStates       map[string]TwinProcessingStageState `json:"stage_states,omitempty"`
	Metrics           map[string]any                      `json:"metrics,omitempty"`
	Error             string                              `json:"error,omitempty"`
}

// TwinProcessingRunCreateRequest queues one explicit twin-processing run.
type TwinProcessingRunCreateRequest struct {
	TriggerType  TwinProcessingTriggerType `json:"trigger_type"`
	TriggerRef   string                    `json:"trigger_ref,omitempty"`
	AutomationID string                    `json:"automation_id,omitempty"`
}

// AlertSeverity reuses the existing low/medium/high/critical severity scale used
// elsewhere in Mimir so alerting does not invent a parallel severity system.
type AlertSeverity = InsightSeverity

// AlertEvent is an append-only operational event emitted when automation
// criteria are met during twin processing. Alerts are intentionally not
// deduplicated by default; downstream export pipelines own routing/noise policy.
type AlertEvent struct {
	ID                        string         `json:"id"`
	ProjectID                 string         `json:"project_id"`
	DigitalTwinID             string         `json:"digital_twin_id"`
	ProcessingRunID           string         `json:"processing_run_id"`
	SourceInsightID           string         `json:"source_insight_id,omitempty"`
	AutomationID              string         `json:"automation_id,omitempty"`
	Severity                  AlertSeverity  `json:"severity"`
	Category                  string         `json:"category"`
	Title                     string         `json:"title"`
	Message                   string         `json:"message"`
	OntologyContext           map[string]any `json:"ontology_context,omitempty"`
	EntityRefs                []string       `json:"entity_refs,omitempty"`
	TriggeredExportPipelineID string         `json:"triggered_export_pipeline_id,omitempty"`
	TriggeredWorkTaskID       string         `json:"triggered_work_task_id,omitempty"`
	CreatedAt                 time.Time      `json:"created_at"`
}
