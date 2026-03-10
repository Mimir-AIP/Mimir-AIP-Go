package models

import "time"

type AnalysisRunKind string

const (
	AnalysisRunKindResolver AnalysisRunKind = "resolver"
	AnalysisRunKindInsights AnalysisRunKind = "insights"
)

type AnalysisRunStatus string

const (
	AnalysisRunStatusCompleted AnalysisRunStatus = "completed"
	AnalysisRunStatusFailed    AnalysisRunStatus = "failed"
)

// AnalysisRun records one scored analysis pass over project data.
type AnalysisRun struct {
	ID               string            `json:"id"`
	ProjectID        string            `json:"project_id"`
	Kind             AnalysisRunKind   `json:"kind"`
	Status           AnalysisRunStatus `json:"status"`
	SourceIDs        []string          `json:"source_ids,omitempty"`
	AlgorithmVersion string            `json:"algorithm_version,omitempty"`
	PolicyVersion    string            `json:"policy_version,omitempty"`
	Coverage         map[string]any    `json:"coverage,omitempty"`
	Metrics          map[string]any    `json:"metrics,omitempty"`
	Error            string            `json:"error,omitempty"`
	CreatedAt        time.Time         `json:"created_at"`
	CompletedAt      *time.Time        `json:"completed_at,omitempty"`
}

type ReviewItemStatus string

const (
	ReviewItemStatusPending      ReviewItemStatus = "pending"
	ReviewItemStatusAccepted     ReviewItemStatus = "accepted"
	ReviewItemStatusRejected     ReviewItemStatus = "rejected"
	ReviewItemStatusAutoAccepted ReviewItemStatus = "auto_accepted"
)

type ReviewDecision string

const (
	ReviewDecisionAccept ReviewDecision = "accept"
	ReviewDecisionReject ReviewDecision = "reject"
)

// ReviewItem stores a persisted reviewable finding snapshot.
type ReviewItem struct {
	ID                string           `json:"id"`
	ProjectID         string           `json:"project_id"`
	RunID             string           `json:"run_id"`
	FindingType       string           `json:"finding_type"`
	Status            ReviewItemStatus `json:"status"`
	SuggestedDecision string           `json:"suggested_decision,omitempty"`
	Confidence        float64          `json:"confidence"`
	Payload           map[string]any   `json:"payload"`
	Evidence          map[string]any   `json:"evidence,omitempty"`
	Rationale         string           `json:"rationale,omitempty"`
	Reviewer          string           `json:"reviewer,omitempty"`
	ReviewedAt        *time.Time       `json:"reviewed_at,omitempty"`
	CreatedAt         time.Time        `json:"created_at"`
	UpdatedAt         time.Time        `json:"updated_at"`
}

// ReviewDecisionRequest applies a human decision to a review item.
type ReviewDecisionRequest struct {
	Decision  ReviewDecision `json:"decision"`
	Rationale string         `json:"rationale,omitempty"`
	Reviewer  string         `json:"reviewer,omitempty"`
}

type InsightSeverity string

const (
	InsightSeverityLow      InsightSeverity = "low"
	InsightSeverityMedium   InsightSeverity = "medium"
	InsightSeverityHigh     InsightSeverity = "high"
	InsightSeverityCritical InsightSeverity = "critical"
)

// Insight stores one persisted autonomous finding.
type Insight struct {
	ID              string          `json:"id"`
	ProjectID       string          `json:"project_id"`
	RunID           string          `json:"run_id"`
	Type            string          `json:"type"`
	Severity        InsightSeverity `json:"severity"`
	Confidence      float64         `json:"confidence"`
	Explanation     string          `json:"explanation"`
	SuggestedAction string          `json:"suggested_action,omitempty"`
	Evidence        map[string]any  `json:"evidence,omitempty"`
	Status          string          `json:"status"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}
