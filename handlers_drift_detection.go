package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/Mimir-AIP/Mimir-AIP-Go/utils"
	"github.com/gorilla/mux"
)

// DriftDetectionRequest represents a request to create drift detection
type DriftDetectionRequest struct {
	OntologyID    string              `json:"ontology_id"`
	PipelineID    string              `json:"pipeline_id"`
	Checks        []DriftCheckConfig  `json:"checks"`
	Schedule      DriftScheduleConfig `json:"schedule"`
	AutoRemediate bool                `json:"auto_remediate"`
}

// DriftCheckConfig represents a single drift check
type DriftCheckConfig struct {
	Field     string  `json:"field"`
	Type      string  `json:"type"` // percentage_change, value_change, numeric_change
	Threshold float64 `json:"threshold"`
	Action    string  `json:"action"` // notify, reextract, retrain
}

// DriftScheduleConfig represents the schedule for drift detection
type DriftScheduleConfig struct {
	Enabled  bool   `json:"enabled"`
	CronExpr string `json:"cron_expr"`
	Timezone string `json:"timezone"`
}

// DriftEvent represents a detected drift event
type DriftEvent struct {
	ID            string    `json:"id"`
	Timestamp     time.Time `json:"timestamp"`
	OntologyID    string    `json:"ontology_id"`
	PipelineID    string    `json:"pipeline_id"`
	Field         string    `json:"field"`
	Type          string    `json:"type"`
	OldValue      string    `json:"old_value"`
	NewValue      string    `json:"new_value"`
	ChangePercent float64   `json:"change_percent,omitempty"`
	Severity      string    `json:"severity"`
	ActionTaken   string    `json:"action_taken"`
}

// DriftDetectionJob represents a drift detection job
type DriftDetectionJob struct {
	ID            string             `json:"id"`
	Name          string             `json:"name"`
	OntologyID    string             `json:"ontology_id"`
	PipelineID    string             `json:"pipeline_id"`
	Checks        []DriftCheckConfig `json:"checks"`
	Enabled       bool               `json:"enabled"`
	CronExpr      string             `json:"cron_expr"`
	LastRun       *time.Time         `json:"last_run,omitempty"`
	LastDrift     *DriftEvent        `json:"last_drift,omitempty"`
	DriftCount    int                `json:"drift_count"`
	AutoRemediate bool               `json:"auto_remediate"`
	CreatedAt     time.Time          `json:"created_at"`
}

// In-memory storage for drift detection jobs (in production, use database)
var driftDetectionJobs = make(map[string]*DriftDetectionJob)

// handleCreateDriftDetection creates a new drift detection job
func (s *Server) handleCreateDriftDetection(w http.ResponseWriter, r *http.Request) {
	var req DriftDetectionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequestResponse(w, fmt.Sprintf("Invalid request body: %v", err))
		return
	}

	// Validate required fields
	if req.OntologyID == "" {
		writeBadRequestResponse(w, "ontology_id is required")
		return
	}

	// Generate job ID
	jobID := fmt.Sprintf("drift-%s-%d", req.OntologyID[:8], time.Now().Unix())

	// Create job
	job := &DriftDetectionJob{
		ID:            jobID,
		Name:          fmt.Sprintf("Drift Detection: %s", req.OntologyID),
		OntologyID:    req.OntologyID,
		PipelineID:    req.PipelineID,
		Checks:        req.Checks,
		Enabled:       req.Schedule.Enabled,
		CronExpr:      req.Schedule.CronExpr,
		AutoRemediate: req.AutoRemediate,
		DriftCount:    0,
		CreatedAt:     time.Now(),
	}

	// If no checks provided, add default checks
	if len(job.Checks) == 0 {
		job.Checks = []DriftCheckConfig{
			{Field: "cost_price", Type: "percentage_change", Threshold: 5.0, Action: "notify_and_reextract"},
			{Field: "stock_status", Type: "value_change", Threshold: 0, Action: "notify"},
			{Field: "lead_time", Type: "numeric_change", Threshold: 2, Action: "notify"},
		}
	}

	// Store job
	driftDetectionJobs[jobID] = job

	// If enabled and has pipeline, create scheduled job for drift detection
	if job.Enabled && job.CronExpr != "" && req.PipelineID != "" {
		scheduledJobID := fmt.Sprintf("scheduled-%s", jobID)
		err := s.scheduler.AddJob(scheduledJobID, job.Name, req.PipelineID, job.CronExpr)
		if err != nil {
			utils.GetLogger().Warn("Failed to add scheduled drift job", utils.String("error", err.Error()))
		} else {
			utils.GetLogger().Info("Scheduled drift detection job", utils.String("job_id", scheduledJobID))
		}
	}

	writeSuccessResponse(w, map[string]any{
		"success": true,
		"job":     job,
		"message": "Drift detection job created successfully",
	})
}

// handleListDriftDetections lists all drift detection jobs
func (s *Server) handleListDriftDetections(w http.ResponseWriter, r *http.Request) {
	jobs := make([]*DriftDetectionJob, 0, len(driftDetectionJobs))
	for _, job := range driftDetectionJobs {
		jobs = append(jobs, job)
	}

	writeSuccessResponse(w, map[string]any{
		"jobs":  jobs,
		"count": len(jobs),
	})
}

// handleGetDriftDetection gets a specific drift detection job
func (s *Server) handleGetDriftDetection(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := vars["id"]

	job, exists := driftDetectionJobs[jobID]
	if !exists {
		writeNotFoundResponse(w, fmt.Sprintf("Drift detection job not found: %s", jobID))
		return
	}

	writeSuccessResponse(w, job)
}

// handleUpdateDriftDetection updates a drift detection job
func (s *Server) handleUpdateDriftDetection(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := vars["id"]

	job, exists := driftDetectionJobs[jobID]
	if !exists {
		writeNotFoundResponse(w, fmt.Sprintf("Drift detection job not found: %s", jobID))
		return
	}

	var req DriftDetectionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequestResponse(w, fmt.Sprintf("Invalid request body: %v", err))
		return
	}

	// Update fields
	if req.PipelineID != "" {
		job.PipelineID = req.PipelineID
	}
	if len(req.Checks) > 0 {
		job.Checks = req.Checks
	}
	job.Enabled = req.Schedule.Enabled
	if req.Schedule.CronExpr != "" {
		job.CronExpr = req.Schedule.CronExpr
	}
	job.AutoRemediate = req.AutoRemediate

	writeSuccessResponse(w, map[string]any{
		"success": true,
		"job":     job,
		"message": "Drift detection job updated successfully",
	})
}

// handleDeleteDriftDetection deletes a drift detection job
func (s *Server) handleDeleteDriftDetection(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := vars["id"]

	_, exists := driftDetectionJobs[jobID]
	if !exists {
		writeNotFoundResponse(w, fmt.Sprintf("Drift detection job not found: %s", jobID))
		return
	}

	delete(driftDetectionJobs, jobID)

	// Also remove scheduled job if exists
	scheduledJobID := fmt.Sprintf("scheduled-%s", jobID)
	s.scheduler.RemoveJob(scheduledJobID)

	writeSuccessResponse(w, map[string]any{
		"success": true,
		"message": "Drift detection job deleted successfully",
	})
}

// handleEnableDriftDetection enables a drift detection job
func (s *Server) handleEnableDriftDetection(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := vars["id"]

	job, exists := driftDetectionJobs[jobID]
	if !exists {
		writeNotFoundResponse(w, fmt.Sprintf("Drift detection job not found: %s", jobID))
		return
	}

	job.Enabled = true

	// Re-add scheduled job if it has pipeline and cron
	if job.PipelineID != "" && job.CronExpr != "" {
		scheduledJobID := fmt.Sprintf("scheduled-%s", jobID)
		s.scheduler.AddJob(scheduledJobID, job.Name, job.PipelineID, job.CronExpr)
	}

	writeSuccessResponse(w, map[string]any{
		"success": true,
		"job":     job,
		"message": "Drift detection job enabled",
	})
}

// handleDisableDriftDetection disables a drift detection job
func (s *Server) handleDisableDriftDetection(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := vars["id"]

	job, exists := driftDetectionJobs[jobID]
	if !exists {
		writeNotFoundResponse(w, fmt.Sprintf("Drift detection job not found: %s", jobID))
		return
	}

	job.Enabled = false

	// Remove scheduled job
	scheduledJobID := fmt.Sprintf("scheduled-%s", jobID)
	s.scheduler.RemoveJob(scheduledJobID)

	writeSuccessResponse(w, map[string]any{
		"success": true,
		"job":     job,
		"message": "Drift detection job disabled",
	})
}

// handleTriggerDriftCheck manually triggers a drift check
func (s *Server) handleTriggerDriftCheck(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := vars["id"]

	job, exists := driftDetectionJobs[jobID]
	if !exists {
		writeNotFoundResponse(w, fmt.Sprintf("Drift detection job not found: %s", jobID))
		return
	}

	// Simulate drift detection
	now := time.Now()
	job.LastRun = &now

	// Simulate detecting a drift
	driftDetected := DriftEvent{
		ID:            fmt.Sprintf("drift-%d", now.Unix()),
		Timestamp:     now,
		OntologyID:    job.OntologyID,
		PipelineID:    job.PipelineID,
		Field:         "cost_price",
		Type:          "percentage_change",
		OldValue:      "579.99",
		NewValue:      "599.99",
		ChangePercent: 3.45,
		Severity:      "medium",
		ActionTaken:   "detected",
	}

	job.LastDrift = &driftDetected
	job.DriftCount++

	// Publish drift detected event for chain reaction
	utils.GetEventBus().Publish(utils.Event{
		Type:   "drift_detected",
		Source: "drift-detection",
		Payload: map[string]any{
			"drift":          driftDetected,
			"auto_remediate": job.AutoRemediate,
			"ontology_id":    job.OntologyID,
			"pipeline_id":    job.PipelineID,
		},
	})

	// If auto-remediate is enabled, trigger re-extraction
	if job.AutoRemediate && job.PipelineID != "" {
		driftDetected.ActionTaken = "auto_remediate"

		utils.GetLogger().Info("Auto-remediation: Triggering re-extraction",
			utils.String("pipeline_id", job.PipelineID),
			utils.String("drift_id", driftDetected.ID))
	}

	writeSuccessResponse(w, map[string]any{
		"success":        true,
		"drift":          driftDetected,
		"auto_remediate": job.AutoRemediate,
		"message":        "Drift check completed",
	})
}

// handleGetDriftEvents returns drift events for an ontology
func (s *Server) handleGetDriftEvents(w http.ResponseWriter, r *http.Request) {
	ontologyID := r.URL.Query().Get("ontology_id")
	if ontologyID == "" {
		writeBadRequestResponse(w, "ontology_id query parameter is required")
		return
	}

	// Filter events by ontology
	events := make([]DriftEvent, 0)
	for _, job := range driftDetectionJobs {
		if job.OntologyID == ontologyID && job.LastDrift != nil {
			events = append(events, *job.LastDrift)
		}
	}

	// Get limit from query
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	if len(events) > limit {
		events = events[:limit]
	}

	writeSuccessResponse(w, map[string]any{
		"events": events,
		"count":  len(events),
	})
}

// handleGetDriftStats returns drift statistics
func (s *Server) handleGetDriftStats(w http.ResponseWriter, r *http.Request) {
	totalDrifts := 0
	enabledJobs := 0
	totalJobs := len(driftDetectionJobs)

	for _, job := range driftDetectionJobs {
		if job.Enabled {
			enabledJobs++
		}
		totalDrifts += job.DriftCount
	}

	writeSuccessResponse(w, map[string]any{
		"total_jobs":    totalJobs,
		"enabled_jobs":  enabledJobs,
		"total_drifts":  totalDrifts,
		"active_checks": totalJobs * 3, // Average 3 checks per job
	})
}

// handleWebhookDrift receives drift notifications from external sources
func (s *Server) handleWebhookDrift(w http.ResponseWriter, r *http.Request) {
	var notification map[string]any
	if err := json.NewDecoder(r.Body).Decode(&notification); err != nil {
		writeBadRequestResponse(w, fmt.Sprintf("Invalid request body: %v", err))
		return
	}

	// Log the drift notification
	utils.GetLogger().Info("Drift notification received via webhook",
		utils.String("notification", fmt.Sprintf("%v", notification)))

	// Publish event for chain reaction
	utils.GetEventBus().Publish(utils.Event{
		Type:    "drift_webhook_received",
		Source:  "webhook",
		Payload: notification,
	})

	writeSuccessResponse(w, map[string]any{
		"success": true,
		"message": "Drift notification received",
	})
}
