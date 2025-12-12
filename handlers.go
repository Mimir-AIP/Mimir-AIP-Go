package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
	"github.com/Mimir-AIP/Mimir-AIP-Go/utils"
	"github.com/gorilla/mux"
)

// handleHealth handles health check requests
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSONResponse(w, http.StatusOK, map[string]any{
		"status": "healthy",
		"time":   time.Now().Format(time.RFC3339),
	})
}

// handleExecutePipeline handles pipeline execution requests
func (s *Server) handleExecutePipeline(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req PipelineExecutionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequestResponse(w, fmt.Sprintf("Invalid request body: %v", err))
		return
	}

	// Determine pipeline to execute
	pipelineFile := req.PipelineFile
	if pipelineFile == "" && req.PipelineName != "" {
		// Try to find pipeline by name in config
		pipelineFile = fmt.Sprintf("pipelines/%s.yaml", req.PipelineName)
	}

	if pipelineFile == "" {
		writeBadRequestResponse(w, "Either pipeline_name or pipeline_file must be provided")
		return
	}

	// Parse and execute pipeline
	config, err := utils.ParsePipeline(pipelineFile)
	if err != nil {
		writeBadRequestResponse(w, fmt.Sprintf("Failed to parse pipeline: %v", err))
		return
	}

	// Initialize context
	ctx := context.Background()
	globalContext := pipelines.NewPluginContext()
	if req.Context != nil {
		for k, v := range req.Context {
			globalContext.Set(k, v)
		}
	}

	// Execute pipeline
	result, err := utils.ExecutePipeline(ctx, config)
	if err != nil {
		writeInternalServerErrorResponse(w, fmt.Sprintf("Pipeline execution failed: %v", err))
		return
	}

	// Return response
	response := PipelineExecutionResponse{
		Success:    result.Success,
		Error:      result.Error,
		Context:    result.Context,
		ExecutedAt: time.Now().Format(time.RFC3339),
	}

	writeSuccessResponse(w, response)
}

// handleListPipelines handles requests to list all pipelines
func (s *Server) handleListPipelines(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Try to read from config.yaml
	configPath := "config.yaml"
	pipelines, err := utils.ParseAllPipelines(configPath)
	if err != nil {
		// If config.yaml doesn't exist, return empty array
		_ = json.NewEncoder(w).Encode([]utils.PipelineConfig{})
		return
	}

	// Ensure we never return null, always return empty array if nil
	if pipelines == nil {
		pipelines = []utils.PipelineConfig{}
	}

	json.NewEncoder(w).Encode(pipelines)
}

// handleGetPipeline handles requests to get a specific pipeline
func (s *Server) handleGetPipeline(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	pipelineName := vars["name"]

	// Try to find pipeline in config
	configPath := "config.yaml"
	pipelines, err := utils.ParseAllPipelines(configPath)
	if err != nil {
		http.Error(w, "Failed to read pipelines", http.StatusInternalServerError)
		return
	}

	for _, pipeline := range pipelines {
		if pipeline.Name == pipelineName {
			json.NewEncoder(w).Encode(pipeline)
			return
		}
	}

	http.Error(w, "Pipeline not found", http.StatusNotFound)
}

// handleListPlugins handles requests to list all plugins
func (s *Server) handleListPlugins(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var plugins []PluginInfo
	for pluginType, typePlugins := range s.registry.GetAllPlugins() {
		for pluginName := range typePlugins {
			plugins = append(plugins, PluginInfo{
				Type:        pluginType,
				Name:        pluginName,
				Description: fmt.Sprintf("%s plugin", pluginName),
			})
		}
	}

	json.NewEncoder(w).Encode(plugins)
}

// handleListPluginsByType handles requests to list plugins of a specific type
func (s *Server) handleListPluginsByType(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	pluginType := vars["type"]

	plugins := s.registry.GetPluginsByType(pluginType)
	var pluginInfos []PluginInfo
	for pluginName := range plugins {
		pluginInfos = append(pluginInfos, PluginInfo{
			Type:        pluginType,
			Name:        pluginName,
			Description: fmt.Sprintf("%s plugin", pluginName),
		})
	}

	json.NewEncoder(w).Encode(pluginInfos)
}

// handleGetPlugin handles requests to get information about a specific plugin
func (s *Server) handleGetPlugin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	pluginType := vars["type"]
	pluginName := vars["name"]

	_, err := s.registry.GetPlugin(pluginType, pluginName)
	if err != nil {
		http.Error(w, "Plugin not found", http.StatusNotFound)
		return
	}

	pluginInfo := PluginInfo{
		Type:        pluginType,
		Name:        pluginName,
		Description: fmt.Sprintf("%s plugin", pluginName),
	}

	json.NewEncoder(w).Encode(pluginInfo)
}

// handleAgentExecute handles agentic execution requests (placeholder for now)
func (s *Server) handleAgentExecute(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// This will be implemented when we add LLM integration
	response := map[string]any{
		"message": "Agent execution not yet implemented",
		"status":  "pending",
	}

	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(response)
}

// Scheduler endpoint handlers

// handleListJobs handles requests to list all scheduled jobs
func (s *Server) handleListJobs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	jobsMap := s.scheduler.GetJobs()

	// Convert map to array for frontend compatibility
	jobs := make([]*utils.ScheduledJob, 0, len(jobsMap))
	for _, job := range jobsMap {
		jobs = append(jobs, job)
	}

	json.NewEncoder(w).Encode(jobs)
}

// handleGetJob handles requests to get a specific scheduled job
func (s *Server) handleGetJob(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	jobID := vars["id"]

	job, err := s.scheduler.GetJob(jobID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Job not found: %v", err), http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(job)
}

// handleCreateJob handles requests to create a new scheduled job
func (s *Server) handleCreateJob(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		Pipeline string `json:"pipeline"`
		CronExpr string `json:"cron_expr"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequestResponse(w, fmt.Sprintf("Invalid request body: %v", err))
		return
	}

	if req.ID == "" || req.Name == "" || req.Pipeline == "" || req.CronExpr == "" {
		http.Error(w, "Missing required fields: id, name, pipeline, cron_expr", http.StatusBadRequest)
		return
	}

	err := s.scheduler.AddJob(req.ID, req.Name, req.Pipeline, req.CronExpr)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create job: %v", err), http.StatusBadRequest)
		return
	}

	response := map[string]any{
		"message": "Job created successfully",
		"job_id":  req.ID,
	}

	json.NewEncoder(w).Encode(response)
}

// Pipeline CRUD endpoint handlers

// handleCreatePipeline handles requests to create a new pipeline
func (s *Server) handleCreatePipeline(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req struct {
		Metadata utils.PipelineMetadata `json:"metadata"`
		Config   utils.PipelineConfig   `json:"config"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequestResponse(w, fmt.Sprintf("Invalid request body: %v", err))
		return
	}

	// Get user from context
	user, ok := utils.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "User not found in context", http.StatusUnauthorized)
		return
	}

	store := utils.GetPipelineStore()
	pipeline, err := store.CreatePipeline(req.Metadata, req.Config, user.Username)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create pipeline: %v", err), http.StatusBadRequest)
		return
	}

	response := map[string]any{
		"message":  "Pipeline created successfully",
		"pipeline": pipeline,
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// handleUpdatePipeline handles requests to update an existing pipeline
func (s *Server) handleUpdatePipeline(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	pipelineID := vars["id"]

	var req struct {
		Metadata *utils.PipelineMetadata `json:"metadata,omitempty"`
		Config   *utils.PipelineConfig   `json:"config,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequestResponse(w, fmt.Sprintf("Invalid request body: %v", err))
		return
	}

	// Get user from context
	user, ok := utils.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "User not found in context", http.StatusUnauthorized)
		return
	}

	store := utils.GetPipelineStore()
	pipeline, err := store.UpdatePipeline(pipelineID, req.Metadata, req.Config, user.Username)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to update pipeline: %v", err), http.StatusBadRequest)
		return
	}

	response := map[string]any{
		"message":  "Pipeline updated successfully",
		"pipeline": pipeline,
	}

	json.NewEncoder(w).Encode(response)
}

// handleDeletePipeline handles requests to delete a pipeline
func (s *Server) handleDeletePipeline(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	pipelineID := vars["id"]

	store := utils.GetPipelineStore()
	err := store.DeletePipeline(pipelineID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete pipeline: %v", err), http.StatusNotFound)
		return
	}

	response := map[string]any{
		"message": "Pipeline deleted successfully",
		"id":      pipelineID,
	}

	json.NewEncoder(w).Encode(response)
}

// handleClonePipeline handles requests to clone a pipeline
func (s *Server) handleClonePipeline(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	pipelineID := vars["id"]

	var req struct {
		Name string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequestResponse(w, fmt.Sprintf("Invalid request body: %v", err))
		return
	}

	if req.Name == "" {
		http.Error(w, "Pipeline name is required", http.StatusBadRequest)
		return
	}

	// Get user from context
	user, ok := utils.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "User not found in context", http.StatusUnauthorized)
		return
	}

	store := utils.GetPipelineStore()
	clonedPipeline, err := store.ClonePipeline(pipelineID, req.Name, user.Username)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to clone pipeline: %v", err), http.StatusBadRequest)
		return
	}

	response := map[string]any{
		"message":  "Pipeline cloned successfully",
		"pipeline": clonedPipeline,
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// handleValidatePipeline handles requests to validate a pipeline
func (s *Server) handleValidatePipeline(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	pipelineID := vars["id"]

	store := utils.GetPipelineStore()
	pipeline, err := store.GetPipeline(pipelineID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Pipeline not found: %v", err), http.StatusNotFound)
		return
	}

	err = store.ValidatePipeline(pipeline)
	if err != nil {
		response := map[string]any{
			"valid":       false,
			"errors":      []string{err.Error()},
			"pipeline_id": pipelineID,
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := map[string]any{
		"valid":       true,
		"errors":      []string{},
		"pipeline_id": pipelineID,
	}

	json.NewEncoder(w).Encode(response)
}

// handleGetPipelineHistory handles requests to get pipeline history
func (s *Server) handleGetPipelineHistory(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	pipelineID := vars["id"]

	store := utils.GetPipelineStore()
	history, err := store.GetPipelineHistory(pipelineID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get pipeline history: %v", err), http.StatusNotFound)
		return
	}

	response := map[string]any{
		"pipeline_id": pipelineID,
		"history":     history,
	}

	json.NewEncoder(w).Encode(response)
}

// handleDeleteJob handles requests to delete a scheduled job
func (s *Server) handleDeleteJob(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	jobID := vars["id"]

	err := s.scheduler.RemoveJob(jobID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete job: %v", err), http.StatusNotFound)
		return
	}

	response := map[string]any{
		"message": "Job deleted successfully",
		"job_id":  jobID,
	}

	json.NewEncoder(w).Encode(response)
}

// handleEnableJob handles requests to enable a scheduled job
func (s *Server) handleEnableJob(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	jobID := vars["id"]

	err := s.scheduler.EnableJob(jobID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to enable job: %v", err), http.StatusNotFound)
		return
	}

	response := map[string]any{
		"message": "Job enabled successfully",
		"job_id":  jobID,
	}

	json.NewEncoder(w).Encode(response)
}

// handleDisableJob handles requests to disable a scheduled job
func (s *Server) handleDisableJob(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	jobID := vars["id"]

	err := s.scheduler.DisableJob(jobID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to disable job: %v", err), http.StatusNotFound)
		return
	}

	response := map[string]any{
		"message": "Job disabled successfully",
		"job_id":  jobID,
	}

	json.NewEncoder(w).Encode(response)
}

// handleUpdateJob handles requests to update a scheduled job
func (s *Server) handleUpdateJob(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	jobID := vars["id"]

	var req struct {
		Name     *string `json:"name,omitempty"`
		Pipeline *string `json:"pipeline,omitempty"`
		CronExpr *string `json:"cron_expr,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequestResponse(w, fmt.Sprintf("Invalid request body: %v", err))
		return
	}

	err := s.scheduler.UpdateJob(jobID, req.Name, req.Pipeline, req.CronExpr)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to update job: %v", err), http.StatusBadRequest)
		return
	}

	response := map[string]any{
		"message": "Job updated successfully",
		"job_id":  jobID,
	}

	json.NewEncoder(w).Encode(response)
}

// Logging endpoint handlers

// handleGetExecutionLog handles requests to get execution log for a specific execution
func (s *Server) handleGetExecutionLog(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	executionID := vars["id"]

	logger := utils.GetExecutionLogger()
	log, err := logger.GetExecutionLog(executionID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get execution log: %v", err), http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(log)
}

// handleListExecutionLogs handles requests to list execution logs with optional filtering
func (s *Server) handleListExecutionLogs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	jobID := r.URL.Query().Get("job_id")
	pipelineID := r.URL.Query().Get("pipeline_id")
	limit := 100 // default

	if limitParam := r.URL.Query().Get("limit"); limitParam != "" {
		if l, err := fmt.Sscanf(limitParam, "%d", &limit); err != nil || l != 1 {
			limit = 100
		}
	}

	logger := utils.GetExecutionLogger()
	logs, err := logger.ListLogs(jobID, pipelineID, limit)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list execution logs: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(logs)
}

// handleGetPipelineLogs handles requests to get all logs for a specific pipeline
func (s *Server) handleGetPipelineLogs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	pipelineID := vars["id"]

	limit := 50 // default
	if limitParam := r.URL.Query().Get("limit"); limitParam != "" {
		if l, err := fmt.Sscanf(limitParam, "%d", &limit); err != nil || l != 1 {
			limit = 50
		}
	}

	logger := utils.GetExecutionLogger()
	logs, err := logger.ListLogs("", pipelineID, limit)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get pipeline logs: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]any{
		"pipeline_id": pipelineID,
		"logs":        logs,
	}

	json.NewEncoder(w).Encode(response)
}

// handleGetJobLogs handles requests to get all logs for a specific job
func (s *Server) handleGetJobLogs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	jobID := vars["id"]

	limit := 50 // default
	if limitParam := r.URL.Query().Get("limit"); limitParam != "" {
		if l, err := fmt.Sscanf(limitParam, "%d", &limit); err != nil || l != 1 {
			limit = 50
		}
	}

	logger := utils.GetExecutionLogger()
	logs, err := logger.ListLogs(jobID, "", limit)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get job logs: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]any{
		"job_id": jobID,
		"logs":   logs,
	}

	json.NewEncoder(w).Encode(response)
}


// Visualization endpoint handlers

// handleVisualizePipeline handles requests to visualize a pipeline
func (s *Server) handleVisualizePipeline(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")

	var req struct {
		PipelineFile string `json:"pipeline_file"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequestResponse(w, fmt.Sprintf("Invalid request body: %v", err))
		return
	}

	if req.PipelineFile == "" {
		http.Error(w, "pipeline_file is required", http.StatusBadRequest)
		return
	}

	// Parse pipeline configuration
	config, err := utils.ParsePipeline(req.PipelineFile)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse pipeline: %v", err), http.StatusBadRequest)
		return
	}

	// Generate visualization
	visualizer := utils.NewASCIIVisualizer()
	visualization := visualizer.VisualizePipeline(config)

	_, _ = w.Write([]byte(visualization))
}

// Authentication endpoint handlers

// handleLogin handles user login
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequestResponse(w, fmt.Sprintf("Invalid request body: %v", err))
		return
	}

	auth := utils.GetAuthManager()
	user, err := auth.AuthenticateUser(req.Username, req.Password)
	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	token, err := auth.GenerateJWT(user)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	response := map[string]any{
		"token":      token,
		"user":       user.Username,
		"roles":      user.Roles,
		"expires_in": auth.GetTokenExpiry().Seconds(),
	}

	json.NewEncoder(w).Encode(response)
}

// handleRefreshToken handles token refresh
func (s *Server) handleRefreshToken(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req struct {
		Token string `json:"token"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequestResponse(w, fmt.Sprintf("Invalid request body: %v", err))
		return
	}

	auth := utils.GetAuthManager()
	claims, err := auth.ValidateJWT(req.Token)
	if err != nil {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	// Find user
	var user *utils.User
	users := auth.GetUsers()
	for _, u := range users {
		if u.ID == claims.UserID {
			user = u
			break
		}
	}

	if user == nil {
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	// Generate new token
	newToken, err := auth.GenerateJWT(user)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	response := map[string]any{
		"token":      newToken,
		"expires_in": auth.GetTokenExpiry().Seconds(),
	}

	json.NewEncoder(w).Encode(response)
}

// handleAuthMe returns current user information
func (s *Server) handleAuthMe(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	user, ok := utils.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "User not found in context", http.StatusUnauthorized)
		return
	}

	response := map[string]any{
		"id":       user.ID,
		"username": user.Username,
		"roles":    user.Roles,
		"active":   user.Active,
	}

	json.NewEncoder(w).Encode(response)
}

// handleListUsers lists all users (admin only)
func (s *Server) handleListUsers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	auth := utils.GetAuthManager()
	var users []map[string]any

	allUsers := auth.GetUsers()
	for _, user := range allUsers {
		users = append(users, map[string]any{
			"id":       user.ID,
			"username": user.Username,
			"roles":    user.Roles,
			"active":   user.Active,
		})
	}

	json.NewEncoder(w).Encode(map[string]any{"users": users})
}

// handleCreateAPIKey creates a new API key for the authenticated user
func (s *Server) handleCreateAPIKey(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	user, ok := utils.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "User not found in context", http.StatusUnauthorized)
		return
	}

	var req struct {
		Name string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequestResponse(w, fmt.Sprintf("Invalid request body: %v", err))
		return
	}

	if req.Name == "" {
		req.Name = "API Key"
	}

	auth := utils.GetAuthManager()
	apiKey, err := auth.CreateAPIKey(user.ID, req.Name)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create API key: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]any{
		"key":     apiKey.Key,
		"name":    apiKey.Name,
		"user_id": apiKey.UserID,
		"created": apiKey.Created,
	}

	json.NewEncoder(w).Encode(response)
}

// Configuration endpoint handlers

// handleGetConfig handles requests to get current configuration
func (s *Server) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	config := s.config.GetConfig()
	json.NewEncoder(w).Encode(config)
}

// handleUpdateConfig handles requests to update configuration
func (s *Server) handleUpdateConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var updates utils.Config
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	err := s.config.UpdateConfig(&updates)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to update configuration: %v", err), http.StatusBadRequest)
		return
	}

	response := map[string]any{
		"message": "Configuration updated successfully",
	}

	json.NewEncoder(w).Encode(response)
}

// handleReloadConfig handles requests to reload configuration from file
func (s *Server) handleReloadConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	configPath := s.config.GetConfigPath()
	if configPath == "" {
		http.Error(w, "No configuration file loaded", http.StatusBadRequest)
		return
	}

	err := s.config.LoadFromFile(configPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to reload configuration: %v", err), http.StatusInternalServerError)
		return
	}

	// Also reload environment variables
	err = s.config.LoadFromEnvironment()
	if err != nil {
		log.Printf("Warning: Failed to reload environment config: %v", err)
	}

	response := map[string]any{
		"message": "Configuration reloaded successfully",
		"file":    configPath,
	}

	json.NewEncoder(w).Encode(response)
}

// handleSaveConfig handles requests to save current configuration to file
func (s *Server) handleSaveConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req struct {
		FilePath string `json:"file_path,omitempty"`
		Format   string `json:"format,omitempty"` // "yaml" or "json"
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// If no body provided, use default values
		req.Format = "yaml"
	}

	if req.FilePath == "" {
		req.FilePath = "config.yaml"
	}

	if req.Format == "" {
		if strings.HasSuffix(req.FilePath, ".json") {
			req.Format = "json"
		} else {
			req.Format = "yaml"
		}
	}

	// Ensure file has correct extension
	if req.Format == "json" && !strings.HasSuffix(req.FilePath, ".json") {
		req.FilePath += ".json"
	} else if req.Format == "yaml" && !strings.HasSuffix(req.FilePath, ".yaml") && !strings.HasSuffix(req.FilePath, ".yml") {
		req.FilePath += ".yaml"
	}

	err := s.config.SaveToFile(req.FilePath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to save configuration: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]any{
		"message": "Configuration saved successfully",
		"file":    req.FilePath,
		"format":  req.Format,
	}

	json.NewEncoder(w).Encode(response)
}

// Performance monitoring endpoint handlers

// handleGetPerformanceMetrics handles requests to get performance metrics
func (s *Server) handleGetPerformanceMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	monitor := utils.GetPerformanceMonitor()
	metrics := monitor.GetMetrics()

	json.NewEncoder(w).Encode(metrics)
}

// handleGetPerformanceStats handles requests to get performance statistics
func (s *Server) handleGetPerformanceStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	monitor := utils.GetPerformanceMonitor()
	metrics := monitor.GetMetrics()

	// Add additional system stats
	stats := map[string]any{
		"performance": metrics,
		"system": map[string]any{
			"go_version":     runtime.Version(),
			"num_cpu":        runtime.NumCPU(),
			"num_goroutines": runtime.NumGoroutine(),
		},
	}

	json.NewEncoder(w).Encode(stats)
}

// Job monitoring endpoint handlers

// handleListJobExecutions handles requests to list all job executions
func (s *Server) handleListJobExecutions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	executions := s.monitor.GetAllExecutions()
	json.NewEncoder(w).Encode(executions)
}

// handleGetJobExecution handles requests to get a specific job execution
func (s *Server) handleGetJobExecution(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	executionID := vars["id"]

	execution, err := s.monitor.GetExecution(executionID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Execution not found: %v", err), http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(execution)
}

// handleGetRunningJobs handles requests to get currently running jobs
func (s *Server) handleGetRunningJobs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	running := s.monitor.GetRunningExecutions()
	json.NewEncoder(w).Encode(running)
}

// handleStopJobExecution handles requests to stop/kill a running job execution
func (s *Server) handleStopJobExecution(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	executionID := vars["id"]

	if executionID == "" {
		http.Error(w, "Missing job execution ID", http.StatusBadRequest)
		return
	}

	execution, err := s.monitor.GetExecution(executionID)
	if err != nil {
		http.Error(w, "Job execution not found", http.StatusNotFound)
		return
	}

	if execution.Status != "running" {
		http.Error(w, "Job is not running and cannot be stopped", http.StatusBadRequest)
		return
	}

	s.monitor.CancelJob(executionID)

	response := map[string]any{
		"message": "Job execution stopped/cancelled successfully",
		"id":      executionID,
	}
	json.NewEncoder(w).Encode(response)
}

// handleGetJobStatistics handles requests to get job statistics
func (s *Server) handleGetJobStatistics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	stats := s.monitor.GetStatistics()
	json.NewEncoder(w).Encode(stats)
}

// handleGetRecentJobs handles requests to get recent job executions
func (s *Server) handleGetRecentJobs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	limit := 10 // Default limit
	if limitParam := r.URL.Query().Get("limit"); limitParam != "" {
		if l, err := fmt.Sscanf(limitParam, "%d", &limit); err != nil || l != 1 {
			limit = 10
		}
	}

	recent := s.monitor.GetRecentExecutions(limit)
	json.NewEncoder(w).Encode(recent)
}

// handleExportJobs handles requests to export job data
func (s *Server) handleExportJobs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	data, err := s.monitor.ExportToJSON()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to export data: %v", err), http.StatusInternalServerError)
		return
	}

	_, _ = w.Write(data)
}

// handleVisualizeStatus handles requests to visualize system status
func (s *Server) handleVisualizeStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")

	visualizer := utils.NewASCIIVisualizer()

	var output strings.Builder

	// System overview
	output.WriteString(visualizer.VisualizePluginRegistry(s.registry))
	output.WriteString("\n")

	// Scheduler status
	jobs := s.scheduler.GetJobs()
	output.WriteString(visualizer.VisualizeSchedulerJobs(jobs))

	_, _ = w.Write([]byte(output.String()))
}

// handleVisualizeScheduler handles requests to visualize scheduler status
func (s *Server) handleVisualizeScheduler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")

	jobs := s.scheduler.GetJobs()
	visualizer := utils.NewASCIIVisualizer()
	visualization := visualizer.VisualizeSchedulerJobs(jobs)

	_, _ = w.Write([]byte(visualization))
}

// handleVisualizePlugins handles requests to visualize available plugins
func (s *Server) handleVisualizePlugins(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")

	visualizer := utils.NewASCIIVisualizer()
	visualization := visualizer.VisualizePluginRegistry(s.registry)

	_, _ = w.Write([]byte(visualization))
}
