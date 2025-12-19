package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/ML"
	storage "github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/Storage"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// --- Monitoring Job Handlers ---

// handleCreateMonitoringJob creates a new monitoring job
// POST /api/v1/monitoring/jobs
func (s *Server) handleCreateMonitoringJob(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string   `json:"name"`
		OntologyID  string   `json:"ontology_id"`
		Description string   `json:"description"`
		CronExpr    string   `json:"cron_expr"`
		Metrics     []string `json:"metrics"`
		Rules       []string `json:"rules"`
		IsEnabled   bool     `json:"is_enabled"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequestResponse(w, "Invalid request body")
		return
	}

	// Validate required fields
	if req.Name == "" || req.OntologyID == "" || req.CronExpr == "" {
		writeBadRequestResponse(w, "Missing required fields: name, ontology_id, cron_expr")
		return
	}

	if len(req.Metrics) == 0 {
		writeBadRequestResponse(w, "At least one metric is required")
		return
	}

	// Marshal metrics and rules to JSON
	metricsJSON, err := json.Marshal(req.Metrics)
	if err != nil {
		writeInternalServerErrorResponse(w, "Failed to encode metrics")
		return
	}

	rulesJSON := "[]"
	if len(req.Rules) > 0 {
		rulesBytes, err := json.Marshal(req.Rules)
		if err != nil {
			writeInternalServerErrorResponse(w, "Failed to encode rules")
			return
		}
		rulesJSON = string(rulesBytes)
	}

	// Create monitoring job
	job := &storage.MonitoringJob{
		ID:          uuid.New().String(),
		Name:        req.Name,
		OntologyID:  req.OntologyID,
		Description: req.Description,
		CronExpr:    req.CronExpr,
		Metrics:     string(metricsJSON),
		Rules:       rulesJSON,
		IsEnabled:   req.IsEnabled,
	}

	ctx := context.Background()
	if err := s.persistence.CreateMonitoringJob(ctx, job); err != nil {
		writeInternalServerErrorResponse(w, fmt.Sprintf("Failed to create monitoring job: %v", err))
		return
	}

	// Add to scheduler if enabled
	if req.IsEnabled && s.scheduler != nil {
		if err := s.scheduler.AddMonitoringJob(job.ID, job.Name, job.ID, job.CronExpr); err != nil {
			writeInternalServerErrorResponse(w, fmt.Sprintf("Failed to schedule job: %v", err))
			return
		}
	}

	writeOperationSuccessResponse(w, "Monitoring job created successfully", "id", job.ID)
}

// handleListMonitoringJobs lists monitoring jobs with optional filters
// GET /api/v1/monitoring/jobs?ontology_id=xxx&enabled_only=true
func (s *Server) handleListMonitoringJobs(w http.ResponseWriter, r *http.Request) {
	ontologyID := r.URL.Query().Get("ontology_id")
	enabledOnly := r.URL.Query().Get("enabled_only") == "true"

	ctx := context.Background()
	jobs, err := s.persistence.ListMonitoringJobs(ctx, ontologyID, enabledOnly)
	if err != nil {
		writeInternalServerErrorResponse(w, fmt.Sprintf("Failed to list monitoring jobs: %v", err))
		return
	}

	// Ensure jobs is never nil - return empty array instead
	if jobs == nil {
		jobs = []*storage.MonitoringJob{}
	}

	writeSuccessResponse(w, map[string]any{
		"jobs":  jobs,
		"count": len(jobs),
	})
}

// handleGetMonitoringJob retrieves a specific monitoring job
// GET /api/v1/monitoring/jobs/{id}
func (s *Server) handleGetMonitoringJob(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := vars["id"]

	ctx := context.Background()
	job, err := s.persistence.GetMonitoringJob(ctx, jobID)
	if err != nil {
		writeNotFoundResponse(w, fmt.Sprintf("Monitoring job not found: %v", err))
		return
	}

	writeSuccessResponse(w, job)
}

// handleUpdateMonitoringJob updates an existing monitoring job
// PUT /api/v1/monitoring/jobs/{id}
func (s *Server) handleUpdateMonitoringJob(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := vars["id"]

	var req struct {
		Name        string   `json:"name"`
		Description string   `json:"description"`
		CronExpr    string   `json:"cron_expr"`
		Metrics     []string `json:"metrics"`
		Rules       []string `json:"rules"`
		IsEnabled   bool     `json:"is_enabled"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequestResponse(w, "Invalid request body")
		return
	}

	ctx := context.Background()

	// Get existing job
	job, err := s.persistence.GetMonitoringJob(ctx, jobID)
	if err != nil {
		writeNotFoundResponse(w, "Monitoring job not found")
		return
	}

	// Update fields
	if req.Name != "" {
		job.Name = req.Name
	}
	if req.Description != "" {
		job.Description = req.Description
	}
	if req.CronExpr != "" {
		job.CronExpr = req.CronExpr
	}
	if len(req.Metrics) > 0 {
		metricsJSON, err := json.Marshal(req.Metrics)
		if err != nil {
			writeInternalServerErrorResponse(w, "Failed to encode metrics")
			return
		}
		job.Metrics = string(metricsJSON)
	}
	if len(req.Rules) > 0 {
		rulesJSON, err := json.Marshal(req.Rules)
		if err != nil {
			writeInternalServerErrorResponse(w, "Failed to encode rules")
			return
		}
		job.Rules = string(rulesJSON)
	}
	job.IsEnabled = req.IsEnabled

	// Update in database
	if err := s.persistence.UpdateMonitoringJob(ctx, job); err != nil {
		writeInternalServerErrorResponse(w, fmt.Sprintf("Failed to update monitoring job: %v", err))
		return
	}

	writeSuccessResponse(w, map[string]any{
		"message": "Monitoring job updated successfully",
		"job":     job,
	})
}

// handleDeleteMonitoringJob deletes a monitoring job
// DELETE /api/v1/monitoring/jobs/{id}
func (s *Server) handleDeleteMonitoringJob(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := vars["id"]

	ctx := context.Background()

	// Remove from scheduler first
	if s.scheduler != nil {
		s.scheduler.RemoveJob(jobID)
	}

	// Delete from database
	if err := s.persistence.DeleteMonitoringJob(ctx, jobID); err != nil {
		writeInternalServerErrorResponse(w, fmt.Sprintf("Failed to delete monitoring job: %v", err))
		return
	}

	writeSuccessResponse(w, map[string]any{
		"message": "Monitoring job deleted successfully",
	})
}

// handleEnableMonitoringJob enables a monitoring job
// POST /api/v1/monitoring/jobs/{id}/enable
func (s *Server) handleEnableMonitoringJob(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := vars["id"]

	ctx := context.Background()
	job, err := s.persistence.GetMonitoringJob(ctx, jobID)
	if err != nil {
		writeNotFoundResponse(w, "Monitoring job not found")
		return
	}

	job.IsEnabled = true
	if err := s.persistence.UpdateMonitoringJob(ctx, job); err != nil {
		writeInternalServerErrorResponse(w, fmt.Sprintf("Failed to enable job: %v", err))
		return
	}

	// Add to scheduler
	if s.scheduler != nil {
		if err := s.scheduler.AddMonitoringJob(job.ID, job.Name, job.ID, job.CronExpr); err != nil {
			writeInternalServerErrorResponse(w, fmt.Sprintf("Failed to schedule job: %v", err))
			return
		}
	}

	writeSuccessResponse(w, map[string]any{
		"message": "Monitoring job enabled successfully",
	})
}

// handleDisableMonitoringJob disables a monitoring job
// POST /api/v1/monitoring/jobs/{id}/disable
func (s *Server) handleDisableMonitoringJob(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := vars["id"]

	ctx := context.Background()
	job, err := s.persistence.GetMonitoringJob(ctx, jobID)
	if err != nil {
		writeNotFoundResponse(w, "Monitoring job not found")
		return
	}

	job.IsEnabled = false
	if err := s.persistence.UpdateMonitoringJob(ctx, job); err != nil {
		writeInternalServerErrorResponse(w, fmt.Sprintf("Failed to disable job: %v", err))
		return
	}

	// Remove from scheduler
	if s.scheduler != nil {
		s.scheduler.RemoveJob(jobID)
	}

	writeSuccessResponse(w, map[string]any{
		"message": "Monitoring job disabled successfully",
	})
}

// handleGetMonitoringJobRuns retrieves execution history for a monitoring job
// GET /api/v1/monitoring/jobs/{id}/runs?limit=10
func (s *Server) handleGetMonitoringJobRuns(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := vars["id"]

	limit := 10
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		fmt.Sscanf(limitStr, "%d", &limit)
	}

	ctx := context.Background()
	runs, err := s.persistence.GetMonitoringJobRuns(ctx, jobID, limit)
	if err != nil {
		writeInternalServerErrorResponse(w, fmt.Sprintf("Failed to get job runs: %v", err))
		return
	}

	writeSuccessResponse(w, map[string]any{
		"runs":  runs,
		"count": len(runs),
	})
}

// --- Monitoring Rule Handlers ---

// handleCreateMonitoringRule creates a new monitoring rule
// POST /api/v1/monitoring/rules
func (s *Server) handleCreateMonitoringRule(w http.ResponseWriter, r *http.Request) {
	var req struct {
		OntologyID    string                 `json:"ontology_id"`
		EntityID      string                 `json:"entity_id"`
		MetricName    string                 `json:"metric_name"`
		RuleType      string                 `json:"rule_type"` // "threshold", "trend", "anomaly"
		Condition     map[string]interface{} `json:"condition"`
		Severity      string                 `json:"severity"`
		IsEnabled     bool                   `json:"is_enabled"`
		AlertChannels string                 `json:"alert_channels"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequestResponse(w, "Invalid request body")
		return
	}

	// Validate required fields
	if req.OntologyID == "" || req.MetricName == "" || req.RuleType == "" {
		writeBadRequestResponse(w, "Missing required fields: ontology_id, metric_name, rule_type")
		return
	}

	// Marshal condition to JSON
	conditionJSON, err := json.Marshal(req.Condition)
	if err != nil {
		writeInternalServerErrorResponse(w, "Failed to encode condition")
		return
	}

	// Create monitoring rule
	ctx := context.Background()
	ruleID := uuid.New().String()
	if err := s.persistence.CreateMonitoringRule(ctx, ruleID, req.OntologyID, req.EntityID, req.MetricName,
		req.RuleType, string(conditionJSON), req.Severity, req.IsEnabled, req.AlertChannels); err != nil {
		writeInternalServerErrorResponse(w, fmt.Sprintf("Failed to create monitoring rule: %v", err))
		return
	}

	writeOperationSuccessResponse(w, "Monitoring rule created successfully", "id", ruleID)
}

// handleListMonitoringRules lists monitoring rules with optional filters
// GET /api/v1/monitoring/rules?entity_id=xxx&metric_name=yyy
func (s *Server) handleListMonitoringRules(w http.ResponseWriter, r *http.Request) {
	entityID := r.URL.Query().Get("entity_id")
	metricName := r.URL.Query().Get("metric_name")

	ctx := context.Background()
	rules, err := s.persistence.GetMonitoringRules(ctx, entityID, metricName)
	if err != nil {
		writeInternalServerErrorResponse(w, fmt.Sprintf("Failed to list monitoring rules: %v", err))
		return
	}

	writeSuccessResponse(w, map[string]any{
		"rules": rules,
		"count": len(rules),
	})
}

// handleDeleteMonitoringRule deletes a monitoring rule
// DELETE /api/v1/monitoring/rules/{id}
func (s *Server) handleDeleteMonitoringRule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ruleID := vars["id"]

	ctx := context.Background()
	query := `DELETE FROM monitoring_rules WHERE id = ?`
	_, err := s.persistence.GetDB().ExecContext(ctx, query, ruleID)
	if err != nil {
		writeInternalServerErrorResponse(w, fmt.Sprintf("Failed to delete rule: %v", err))
		return
	}

	writeSuccessResponse(w, map[string]any{
		"message": "Monitoring rule deleted successfully",
	})
}

// --- Alert Handlers ---

// handleListAlerts lists alerts with optional filters
// GET /api/v1/monitoring/alerts?ontology_id=xxx&status=active&severity=high
func (s *Server) handleListAlerts(w http.ResponseWriter, r *http.Request) {
	ontologyID := r.URL.Query().Get("ontology_id")
	status := r.URL.Query().Get("status")
	severity := r.URL.Query().Get("severity")

	ctx := context.Background()
	ruleEngine := ml.NewRuleEngine(s.persistence)

	// Get alerts
	alerts, err := ruleEngine.GetActiveAlerts(ctx, ontologyID)
	if err != nil {
		writeInternalServerErrorResponse(w, fmt.Sprintf("Failed to list alerts: %v", err))
		return
	}

	// Filter by status and severity if provided
	var filteredAlerts []ml.Alert
	for _, alert := range alerts {
		if status != "" && alert.Status != status {
			continue
		}
		if severity != "" && alert.Severity != severity {
			continue
		}
		filteredAlerts = append(filteredAlerts, alert)
	}

	// Ensure alerts is never nil - return empty array instead
	if filteredAlerts == nil {
		filteredAlerts = []ml.Alert{}
	}

	writeSuccessResponse(w, map[string]any{
		"alerts": filteredAlerts,
		"count":  len(filteredAlerts),
	})
}

// handleAcknowledgeAlert acknowledges/resolves an alert
// PATCH /api/v1/monitoring/alerts/{id}
func (s *Server) handleAcknowledgeAlert(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	alertID := vars["id"]

	var req struct {
		Status string `json:"status"` // "acknowledged" or "resolved"
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequestResponse(w, "Invalid request body")
		return
	}

	ctx := context.Background()
	ruleEngine := ml.NewRuleEngine(s.persistence)

	if req.Status == "resolved" {
		if err := ruleEngine.ResolveAlert(ctx, alertID); err != nil {
			writeInternalServerErrorResponse(w, fmt.Sprintf("Failed to resolve alert: %v", err))
			return
		}
	}

	writeSuccessResponse(w, map[string]any{
		"message": fmt.Sprintf("Alert %s successfully", req.Status),
	})
}
