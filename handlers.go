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
	// Try to read from config.yaml
	configPath := "config.yaml"
	pipelines, err := utils.ParseAllPipelines(configPath)
	if err != nil {
		// If config.yaml doesn't exist, return empty array
		writeJSONResponse(w, http.StatusOK, []utils.PipelineConfig{})
		return
	}

	// Ensure we never return null, always return empty array if nil
	if pipelines == nil {
		pipelines = []utils.PipelineConfig{}
	}

	writeJSONResponse(w, http.StatusOK, pipelines)
}

// handleGetPipeline handles requests to get a specific pipeline
func (s *Server) handleGetPipeline(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	pipelineName := vars["name"]

	// Try to find pipeline in config
	configPath := "config.yaml"
	pipelines, err := utils.ParseAllPipelines(configPath)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to read pipelines")
		return
	}

	for _, pipeline := range pipelines {
		if pipeline.Name == pipelineName {
			writeJSONResponse(w, http.StatusOK, pipeline)
			return
		}
	}

	writeErrorResponse(w, http.StatusNotFound, "Pipeline not found")
}

// handleListPlugins handles requests to list all plugins
func (s *Server) handleListPlugins(w http.ResponseWriter, r *http.Request) {

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

	writeJSONResponse(w, http.StatusOK, plugins)
}

// handleListPluginsByType handles requests to list plugins of a specific type
func (s *Server) handleListPluginsByType(w http.ResponseWriter, r *http.Request) {

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

	writeJSONResponse(w, http.StatusOK, pluginInfos)
}

// handleGetPlugin handles requests to get information about a specific plugin
func (s *Server) handleGetPlugin(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	pluginType := vars["type"]
	pluginName := vars["name"]

	_, err := s.registry.GetPlugin(pluginType, pluginName)
	if err != nil {
		writeErrorResponse(w, http.StatusNotFound, "Plugin not found")
		return
	}

	pluginInfo := PluginInfo{
		Type:        pluginType,
		Name:        pluginName,
		Description: fmt.Sprintf("%s plugin", pluginName),
	}

	writeJSONResponse(w, http.StatusOK, pluginInfo)
}

// handleAgentExecute handles agentic execution requests (placeholder for now)
func (s *Server) handleAgentExecute(w http.ResponseWriter, r *http.Request) {

	// This will be implemented when we add LLM integration
	response := map[string]any{
		"message": "Agent execution not yet implemented",
		"status":  "pending",
	}

	w.WriteHeader(http.StatusNotImplemented)
	writeJSONResponse(w, http.StatusOK, response)
}

// Scheduler endpoint handlers

// handleListJobs handles requests to list all scheduled jobs
func (s *Server) handleListJobs(w http.ResponseWriter, r *http.Request) {

	jobsMap := s.scheduler.GetJobs()

	// Convert map to array for frontend compatibility
	jobs := make([]*utils.ScheduledJob, 0, len(jobsMap))
	for _, job := range jobsMap {
		jobs = append(jobs, job)
	}

	writeJSONResponse(w, http.StatusOK, jobs)
}

// handleGetJob handles requests to get a specific scheduled job
func (s *Server) handleGetJob(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	jobID := vars["id"]

	job, err := s.scheduler.GetJob(jobID)
	if err != nil {
		writeErrorResponse(w, http.StatusNotFound, fmt.Sprintf("Job not found: %v", err))
		return
	}

	writeJSONResponse(w, http.StatusOK, job)
}

// handleCreateJob handles requests to create a new scheduled job
func (s *Server) handleCreateJob(w http.ResponseWriter, r *http.Request) {

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
		writeErrorResponse(w, http.StatusBadRequest, "Missing required fields: id, name, pipeline, cron_expr")
		return
	}

	err := s.scheduler.AddJob(req.ID, req.Name, req.Pipeline, req.CronExpr)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, fmt.Sprintf("Failed to create job: %v", err))
		return
	}

	response := map[string]any{
		"message": "Job created successfully",
		"job_id":  req.ID,
	}

	writeJSONResponse(w, http.StatusOK, response)
}

// Pipeline CRUD endpoint handlers

// handleCreatePipeline handles requests to create a new pipeline
func (s *Server) handleCreatePipeline(w http.ResponseWriter, r *http.Request) {

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
		writeErrorResponse(w, http.StatusUnauthorized, "User not found in context")
		return
	}

	store := utils.GetPipelineStore()
	pipeline, err := store.CreatePipeline(req.Metadata, req.Config, user.Username)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, fmt.Sprintf("Failed to create pipeline: %v", err))
		return
	}

	response := map[string]any{
		"message":  "Pipeline created successfully",
		"pipeline": pipeline,
	}

	w.WriteHeader(http.StatusCreated)
	writeJSONResponse(w, http.StatusOK, response)
}

// handleUpdatePipeline handles requests to update an existing pipeline
func (s *Server) handleUpdatePipeline(w http.ResponseWriter, r *http.Request) {

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
		writeErrorResponse(w, http.StatusUnauthorized, "User not found in context")
		return
	}

	store := utils.GetPipelineStore()
	pipeline, err := store.UpdatePipeline(pipelineID, req.Metadata, req.Config, user.Username)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, fmt.Sprintf("Failed to update pipeline: %v", err))
		return
	}

	response := map[string]any{
		"message":  "Pipeline updated successfully",
		"pipeline": pipeline,
	}

	writeJSONResponse(w, http.StatusOK, response)
}

// handleDeletePipeline handles requests to delete a pipeline
func (s *Server) handleDeletePipeline(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	pipelineID := vars["id"]

	store := utils.GetPipelineStore()
	err := store.DeletePipeline(pipelineID)
	if err != nil {
		writeErrorResponse(w, http.StatusNotFound, fmt.Sprintf("Failed to delete pipeline: %v", err))
		return
	}

	response := map[string]any{
		"message": "Pipeline deleted successfully",
		"id":      pipelineID,
	}

	writeJSONResponse(w, http.StatusOK, response)
}

// handleClonePipeline handles requests to clone a pipeline
func (s *Server) handleClonePipeline(w http.ResponseWriter, r *http.Request) {

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
		writeErrorResponse(w, http.StatusBadRequest, "Pipeline name is required")
		return
	}

	// Get user from context
	user, ok := utils.GetUserFromContext(r.Context())
	if !ok {
		writeErrorResponse(w, http.StatusUnauthorized, "User not found in context")
		return
	}

	store := utils.GetPipelineStore()
	clonedPipeline, err := store.ClonePipeline(pipelineID, req.Name, user.Username)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, fmt.Sprintf("Failed to clone pipeline: %v", err))
		return
	}

	response := map[string]any{
		"message":  "Pipeline cloned successfully",
		"pipeline": clonedPipeline,
	}

	w.WriteHeader(http.StatusCreated)
	writeJSONResponse(w, http.StatusOK, response)
}

// handleValidatePipeline handles requests to validate a pipeline
func (s *Server) handleValidatePipeline(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	pipelineID := vars["id"]

	store := utils.GetPipelineStore()
	pipeline, err := store.GetPipeline(pipelineID)
	if err != nil {
		writeErrorResponse(w, http.StatusNotFound, fmt.Sprintf("Pipeline not found: %v", err))
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

	writeJSONResponse(w, http.StatusOK, response)
}

// handleGetPipelineHistory handles requests to get pipeline history
func (s *Server) handleGetPipelineHistory(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	pipelineID := vars["id"]

	store := utils.GetPipelineStore()
	history, err := store.GetPipelineHistory(pipelineID)
	if err != nil {
		writeErrorResponse(w, http.StatusNotFound, fmt.Sprintf("Failed to get pipeline history: %v", err))
		return
	}

	response := map[string]any{
		"pipeline_id": pipelineID,
		"history":     history,
	}

	writeJSONResponse(w, http.StatusOK, response)
}

// handleDeleteJob handles requests to delete a scheduled job
func (s *Server) handleDeleteJob(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	jobID := vars["id"]

	err := s.scheduler.RemoveJob(jobID)
	if err != nil {
		writeErrorResponse(w, http.StatusNotFound, fmt.Sprintf("Failed to delete job: %v", err))
		return
	}

	response := map[string]any{
		"message": "Job deleted successfully",
		"job_id":  jobID,
	}

	writeJSONResponse(w, http.StatusOK, response)
}

// handleEnableJob handles requests to enable a scheduled job
func (s *Server) handleEnableJob(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	jobID := vars["id"]

	err := s.scheduler.EnableJob(jobID)
	if err != nil {
		writeErrorResponse(w, http.StatusNotFound, fmt.Sprintf("Failed to enable job: %v", err))
		return
	}

	response := map[string]any{
		"message": "Job enabled successfully",
		"job_id":  jobID,
	}

	writeJSONResponse(w, http.StatusOK, response)
}

// handleDisableJob handles requests to disable a scheduled job
func (s *Server) handleDisableJob(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	jobID := vars["id"]

	err := s.scheduler.DisableJob(jobID)
	if err != nil {
		writeErrorResponse(w, http.StatusNotFound, fmt.Sprintf("Failed to disable job: %v", err))
		return
	}

	response := map[string]any{
		"message": "Job disabled successfully",
		"job_id":  jobID,
	}

	writeJSONResponse(w, http.StatusOK, response)
}

// handleUpdateJob handles requests to update a scheduled job
func (s *Server) handleUpdateJob(w http.ResponseWriter, r *http.Request) {

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
		writeErrorResponse(w, http.StatusBadRequest, fmt.Sprintf("Failed to update job: %v", err))
		return
	}

	response := map[string]any{
		"message": "Job updated successfully",
		"job_id":  jobID,
	}

	writeJSONResponse(w, http.StatusOK, response)
}

// Logging endpoint handlers

// handleGetExecutionLog handles requests to get execution log for a specific execution
func (s *Server) handleGetExecutionLog(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	executionID := vars["id"]

	logger := utils.GetExecutionLogger()
	log, err := logger.GetExecutionLog(executionID)
	if err != nil {
		writeErrorResponse(w, http.StatusNotFound, fmt.Sprintf("Failed to get execution log: %v", err))
		return
	}

	writeJSONResponse(w, http.StatusOK, log)
}

// handleListExecutionLogs handles requests to list execution logs with optional filtering
func (s *Server) handleListExecutionLogs(w http.ResponseWriter, r *http.Request) {

	jobID := r.URL.Query().Get("job_id")
	pipelineID := r.URL.Query().Get("pipeline_id")
	limit := parseLimit(r, 100)

	logger := utils.GetExecutionLogger()
	logs, err := logger.ListLogs(jobID, pipelineID, limit)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to list execution logs: %v", err))
		return
	}

	writeJSONResponse(w, http.StatusOK, logs)
}

// handleGetPipelineLogs handles requests to get all logs for a specific pipeline
func (s *Server) handleGetPipelineLogs(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	pipelineID := vars["id"]

	limit := parseLimit(r, 50)

	logger := utils.GetExecutionLogger()
	logs, err := logger.ListLogs("", pipelineID, limit)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get pipeline logs: %v", err))
		return
	}

	response := map[string]any{
		"pipeline_id": pipelineID,
		"logs":        logs,
	}

	writeJSONResponse(w, http.StatusOK, response)
}

// handleGetJobLogs handles requests to get all logs for a specific job
func (s *Server) handleGetJobLogs(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	jobID := vars["id"]

	limit := parseLimit(r, 50)

	logger := utils.GetExecutionLogger()
	logs, err := logger.ListLogs(jobID, "", limit)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get job logs: %v", err))
		return
	}

	response := map[string]any{
		"job_id": jobID,
		"logs":   logs,
	}

	writeJSONResponse(w, http.StatusOK, response)
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
		writeErrorResponse(w, http.StatusBadRequest, "pipeline_file is required")
		return
	}

	// Parse pipeline configuration
	config, err := utils.ParsePipeline(req.PipelineFile)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, fmt.Sprintf("Failed to parse pipeline: %v", err))
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
		writeErrorResponse(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	token, err := auth.GenerateJWT(user)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to generate token")
		return
	}

	response := map[string]any{
		"token":      token,
		"user":       user.Username,
		"roles":      user.Roles,
		"expires_in": auth.GetTokenExpiry().Seconds(),
	}

	writeJSONResponse(w, http.StatusOK, response)
}

// handleRefreshToken handles token refresh
func (s *Server) handleRefreshToken(w http.ResponseWriter, r *http.Request) {

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
		writeErrorResponse(w, http.StatusUnauthorized, "Invalid token")
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
		writeErrorResponse(w, http.StatusUnauthorized, "User not found")
		return
	}

	// Generate new token
	newToken, err := auth.GenerateJWT(user)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to generate token")
		return
	}

	response := map[string]any{
		"token":      newToken,
		"expires_in": auth.GetTokenExpiry().Seconds(),
	}

	writeJSONResponse(w, http.StatusOK, response)
}

// handleAuthMe returns current user information
func (s *Server) handleAuthMe(w http.ResponseWriter, r *http.Request) {

	user, ok := utils.GetUserFromContext(r.Context())
	if !ok {
		writeErrorResponse(w, http.StatusUnauthorized, "User not found in context")
		return
	}

	response := map[string]any{
		"id":       user.ID,
		"username": user.Username,
		"roles":    user.Roles,
		"active":   user.Active,
	}

	writeJSONResponse(w, http.StatusOK, response)
}

// handleListUsers lists all users (admin only)
func (s *Server) handleListUsers(w http.ResponseWriter, r *http.Request) {

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

	writeJSONResponse(w, http.StatusOK, map[string]any{"users": users})
}

// handleCreateAPIKey creates a new API key for the authenticated user
func (s *Server) handleCreateAPIKey(w http.ResponseWriter, r *http.Request) {

	user, ok := utils.GetUserFromContext(r.Context())
	if !ok {
		writeErrorResponse(w, http.StatusUnauthorized, "User not found in context")
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
		writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create API key: %v", err))
		return
	}

	response := map[string]any{
		"key":     apiKey.Key,
		"name":    apiKey.Name,
		"user_id": apiKey.UserID,
		"created": apiKey.Created,
	}

	writeJSONResponse(w, http.StatusOK, response)
}

// Configuration endpoint handlers

// handleGetConfig handles requests to get current configuration
func (s *Server) handleGetConfig(w http.ResponseWriter, r *http.Request) {

	config := s.config.GetConfig()
	writeJSONResponse(w, http.StatusOK, config)
}

// handleUpdateConfig handles requests to update configuration
func (s *Server) handleUpdateConfig(w http.ResponseWriter, r *http.Request) {

	var updates utils.Config
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, fmt.Sprintf("Invalid request body: %v", err))
		return
	}

	err := s.config.UpdateConfig(&updates)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, fmt.Sprintf("Failed to update configuration: %v", err))
		return
	}

	response := map[string]any{
		"message": "Configuration updated successfully",
	}

	writeJSONResponse(w, http.StatusOK, response)
}

// handleReloadConfig handles requests to reload configuration from file
func (s *Server) handleReloadConfig(w http.ResponseWriter, r *http.Request) {

	configPath := s.config.GetConfigPath()
	if configPath == "" {
		writeErrorResponse(w, http.StatusBadRequest, "No configuration file loaded")
		return
	}

	err := s.config.LoadFromFile(configPath)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to reload configuration: %v", err))
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

	writeJSONResponse(w, http.StatusOK, response)
}

// handleSaveConfig handles requests to save current configuration to file
func (s *Server) handleSaveConfig(w http.ResponseWriter, r *http.Request) {

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
		writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to save configuration: %v", err))
		return
	}

	response := map[string]any{
		"message": "Configuration saved successfully",
		"file":    req.FilePath,
		"format":  req.Format,
	}

	writeJSONResponse(w, http.StatusOK, response)
}

// Performance monitoring endpoint handlers

// handleGetPerformanceMetrics handles requests to get performance metrics
func (s *Server) handleGetPerformanceMetrics(w http.ResponseWriter, r *http.Request) {

	monitor := utils.GetPerformanceMonitor()
	metrics := monitor.GetMetrics()

	writeJSONResponse(w, http.StatusOK, metrics)
}

// handleGetPerformanceStats handles requests to get performance statistics
func (s *Server) handleGetPerformanceStats(w http.ResponseWriter, r *http.Request) {

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

	writeJSONResponse(w, http.StatusOK, stats)
}

// Job monitoring endpoint handlers

// handleListJobExecutions handles requests to list all job executions
func (s *Server) handleListJobExecutions(w http.ResponseWriter, r *http.Request) {

	executions := s.monitor.GetAllExecutions()
	writeJSONResponse(w, http.StatusOK, executions)
}

// handleGetJobExecution handles requests to get a specific job execution
func (s *Server) handleGetJobExecution(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	executionID := vars["id"]

	execution, err := s.monitor.GetExecution(executionID)
	if err != nil {
		writeErrorResponse(w, http.StatusNotFound, fmt.Sprintf("Execution not found: %v", err))
		return
	}

	writeJSONResponse(w, http.StatusOK, execution)
}

// handleGetRunningJobs handles requests to get currently running jobs
func (s *Server) handleGetRunningJobs(w http.ResponseWriter, r *http.Request) {

	running := s.monitor.GetRunningExecutions()
	writeJSONResponse(w, http.StatusOK, running)
}

// handleStopJobExecution handles requests to stop/kill a running job execution
func (s *Server) handleStopJobExecution(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	executionID := vars["id"]

	if executionID == "" {
		writeErrorResponse(w, http.StatusBadRequest, "Missing job execution ID")
		return
	}

	execution, err := s.monitor.GetExecution(executionID)
	if err != nil {
		writeErrorResponse(w, http.StatusNotFound, "Job execution not found")
		return
	}

	if execution.Status != "running" {
		writeErrorResponse(w, http.StatusBadRequest, "Job is not running and cannot be stopped")
		return
	}

	s.monitor.CancelJob(executionID)

	response := map[string]any{
		"message": "Job execution stopped/cancelled successfully",
		"id":      executionID,
	}
	writeJSONResponse(w, http.StatusOK, response)
}

// handleGetJobStatistics handles requests to get job statistics
func (s *Server) handleGetJobStatistics(w http.ResponseWriter, r *http.Request) {

	stats := s.monitor.GetStatistics()
	writeJSONResponse(w, http.StatusOK, stats)
}

// handleGetRecentJobs handles requests to get recent job executions
func (s *Server) handleGetRecentJobs(w http.ResponseWriter, r *http.Request) {

	limit := parseLimit(r, 10)

	recent := s.monitor.GetRecentExecutions(limit)
	writeJSONResponse(w, http.StatusOK, recent)
}

// handleExportJobs handles requests to export job data
func (s *Server) handleExportJobs(w http.ResponseWriter, r *http.Request) {

	data, err := s.monitor.ExportToJSON()
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to export data: %v", err))
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
