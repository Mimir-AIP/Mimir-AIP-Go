package models

import (
	"fmt"
	"time"
)

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

// AlertApprovalStatus tracks whether an alert-triggered export needs human action.
type AlertApprovalStatus string

const (
	AlertApprovalStatusNotRequired AlertApprovalStatus = "not_required"
	AlertApprovalStatusPending     AlertApprovalStatus = "pending"
	AlertApprovalStatusApproved    AlertApprovalStatus = "approved"
	AlertApprovalStatusRejected    AlertApprovalStatus = "rejected"
)

// AlertApprovalDecision is the operator decision for a pending alert action.
type AlertApprovalDecision string

const (
	AlertApprovalDecisionApprove AlertApprovalDecision = "approve"
	AlertApprovalDecisionReject  AlertApprovalDecision = "reject"
)

// AlertExecutionStatus tracks whether the matched action actually queued downstream work.
type AlertExecutionStatus string

const (
	AlertExecutionStatusPendingApproval AlertExecutionStatus = "pending_approval"
	AlertExecutionStatusQueued          AlertExecutionStatus = "queued"
	AlertExecutionStatusFailed          AlertExecutionStatus = "failed"
	AlertExecutionStatusRejected        AlertExecutionStatus = "rejected"
	AlertExecutionStatusSkipped         AlertExecutionStatus = "skipped"
	AlertExecutionStatusNotApplicable   AlertExecutionStatus = "not_applicable"
)

// AlertEvent is an append-only operational event emitted when automation
// criteria are met during twin processing. Alerts are intentionally not
// deduplicated by default; downstream export pipelines own routing/noise policy.
type AlertEvent struct {
	ID                        string               `json:"id"`
	ProjectID                 string               `json:"project_id"`
	DigitalTwinID             string               `json:"digital_twin_id"`
	ProcessingRunID           string               `json:"processing_run_id"`
	SourceInsightID           string               `json:"source_insight_id,omitempty"`
	AutomationID              string               `json:"automation_id,omitempty"`
	ActionID                  string               `json:"action_id,omitempty"`
	Severity                  AlertSeverity        `json:"severity"`
	Category                  string               `json:"category"`
	Title                     string               `json:"title"`
	Message                   string               `json:"message"`
	OntologyContext           map[string]any       `json:"ontology_context,omitempty"`
	EntityRefs                []string             `json:"entity_refs,omitempty"`
	RequestedExportPipelineID string               `json:"requested_export_pipeline_id,omitempty"`
	RequestedTriggerParams    map[string]any       `json:"requested_trigger_params,omitempty"`
	ApprovalStatus            AlertApprovalStatus  `json:"approval_status,omitempty"`
	ExecutionStatus           AlertExecutionStatus `json:"execution_status,omitempty"`
	ExecutionError            string               `json:"execution_error,omitempty"`
	ApprovalRequestedAt       *time.Time           `json:"approval_requested_at,omitempty"`
	ApprovalResolvedAt        *time.Time           `json:"approval_resolved_at,omitempty"`
	ApprovalActor             string               `json:"approval_actor,omitempty"`
	ApprovalNote              string               `json:"approval_note,omitempty"`
	TriggeredExportPipelineID string               `json:"triggered_export_pipeline_id,omitempty"`
	TriggeredWorkTaskID       string               `json:"triggered_work_task_id,omitempty"`
	CreatedAt                 time.Time            `json:"created_at"`
}

// AlertApprovalRequest applies an operator decision to one pending alert.
type AlertApprovalRequest struct {
	Decision AlertApprovalDecision `json:"decision"`
	Actor    string                `json:"actor,omitempty"`
	Note     string                `json:"note,omitempty"`
}

// Validate checks whether the approval request is well-formed.
func (r *AlertApprovalRequest) Validate() error {
	switch r.Decision {
	case AlertApprovalDecisionApprove, AlertApprovalDecisionReject:
		return nil
	default:
		return fmt.Errorf("decision must be approve or reject")
	}
}
