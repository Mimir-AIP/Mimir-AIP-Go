package models

import "time"

// ProjectSectionStateKind is the normalized frontend state for one nav section.
type ProjectSectionStateKind string

const (
	ProjectSectionStateInactive   ProjectSectionStateKind = "inactive"
	ProjectSectionStateInProgress ProjectSectionStateKind = "in_progress"
	ProjectSectionStateComplete   ProjectSectionStateKind = "complete"
	ProjectSectionStateError      ProjectSectionStateKind = "error"
	ProjectSectionStateAttention  ProjectSectionStateKind = "attention"
)

// ProjectSectionState summarizes one sidebar section's current backend state.
type ProjectSectionState struct {
	Status ProjectSectionStateKind `json:"status"`
	Detail string                  `json:"detail,omitempty"`
	Count  int                     `json:"count,omitempty"`
	Pulse  bool                    `json:"pulse,omitempty"`
}

// ProjectStateSummary is the backend activity snapshot rendered by the frontend nav.
type ProjectStateSummary struct {
	ProjectID   string                         `json:"project_id"`
	GeneratedAt time.Time                      `json:"generated_at"`
	QueueLength int64                          `json:"queue_length"`
	ActiveTasks int                            `json:"active_tasks"`
	Sections    map[string]ProjectSectionState `json:"sections"`
}
