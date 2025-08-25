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

// Server represents the Mimir AIP server
type Server struct {
	router    *mux.Router
	registry  *pipelines.PluginRegistry
	mcpServer *MCPServer
	scheduler *utils.Scheduler
	monitor   *utils.JobMonitor
	config    *utils.ConfigManager
}

// PipelineExecutionRequest represents a request to execute a pipeline
type PipelineExecutionRequest struct {
	PipelineName string                 `json:"pipeline_name,omitempty"`
	PipelineFile string                 `json:"pipeline_file,omitempty"`
	Context      map[string]interface{} `json:"context,omitempty"`
}

// PipelineExecutionResponse represents the response from pipeline execution
type PipelineExecutionResponse struct {
	Success    bool                    `json:"success"`
	Error      string                  `json:"error,omitempty"`
	Context    pipelines.PluginContext `json:"context,omitempty"`
	ExecutedAt string                  `json:"executed_at"`
}

// PluginInfo represents information about a plugin
type PluginInfo struct {
	Type        string `json:"type"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// NewServer creates a new Mimir AIP server
func NewServer() *Server {
	registry := pipelines.NewPluginRegistry()

	s := &Server{
		router:    mux.NewRouter(),
		registry:  registry,
		mcpServer: NewMCPServer(registry),
		scheduler: utils.NewScheduler(registry),
		monitor:   utils.NewJobMonitor(1000), // Keep last 1000 executions
		config:    utils.GetConfigManager(),
	}

	s.registerDefaultPlugins()
	s.setupRoutes()
	s.mcpServer.Initialize()

	// Load configuration
	if err := utils.LoadGlobalConfig(); err != nil {
		log.Printf("Failed to load configuration: %v", err)
	}

	// Initialize logging
	if err := utils.InitLogger(s.config.GetConfig().Logging); err != nil {
		log.Printf("Failed to initialize logger: %v", err)
	}

	// Initialize pipeline store
	if err := utils.InitializeGlobalPipelineStore("./pipelines"); err != nil {
		log.Printf("Failed to initialize pipeline store: %v", err)
	}

	// Start the scheduler
	if err := s.scheduler.Start(); err != nil {
		utils.GetLogger().Error("Failed to start scheduler", err, utils.Component("server"))
	}

	return s
}

// registerDefaultPlugins registers the built-in plugins
func (s *Server) registerDefaultPlugins() {
	// Register real plugins
	apiPlugin := &utils.RealAPIPlugin{}
	htmlPlugin := &utils.MockHTMLPlugin{}

	s.registry.RegisterPlugin(apiPlugin)
	s.registry.RegisterPlugin(htmlPlugin)
}

// setupRoutes sets up the HTTP routes
func (s *Server) setupRoutes() {
	// Add middleware
	s.router.Use(s.loggingMiddleware)
	s.router.Use(s.errorRecoveryMiddleware)
	s.router.Use(s.corsMiddleware)
	s.router.Use(utils.SecurityHeadersMiddleware)
	s.router.Use(utils.InputValidationMiddleware)
	s.router.Use(utils.PerformanceMiddleware)

	// Initialize authentication if enabled
	if s.config.GetConfig().Security.EnableAuth {
		if err := utils.InitAuthManager(s.config.GetConfig().Security); err != nil {
			utils.GetLogger().Error("Failed to initialize authentication", err, utils.Component("server"))
		}
	}

	// Health check
	s.router.HandleFunc("/health", s.handleHealth).Methods("GET")

	// Pipeline execution
	s.router.HandleFunc("/api/v1/pipelines/execute", s.handleExecutePipeline).Methods("POST")

	// Pipeline management
	s.router.HandleFunc("/api/v1/pipelines", s.handleListPipelines).Methods("GET")
	s.router.HandleFunc("/api/v1/pipelines", s.handleCreatePipeline).Methods("POST")
	s.router.HandleFunc("/api/v1/pipelines/{id}", s.handleGetPipeline).Methods("GET")
	s.router.HandleFunc("/api/v1/pipelines/{id}", s.handleUpdatePipeline).Methods("PUT")
	s.router.HandleFunc("/api/v1/pipelines/{id}", s.handleDeletePipeline).Methods("DELETE")
	s.router.HandleFunc("/api/v1/pipelines/{id}/clone", s.handleClonePipeline).Methods("POST")
	s.router.HandleFunc("/api/v1/pipelines/{id}/validate", s.handleValidatePipeline).Methods("POST")
	s.router.HandleFunc("/api/v1/pipelines/{id}/history", s.handleGetPipelineHistory).Methods("GET")

	// Plugin management
	s.router.HandleFunc("/api/v1/plugins", s.handleListPlugins).Methods("GET")
	s.router.HandleFunc("/api/v1/plugins/{type}", s.handleListPluginsByType).Methods("GET")
	s.router.HandleFunc("/api/v1/plugins/{type}/{name}", s.handleGetPlugin).Methods("GET")

	// Agentic features
	s.router.HandleFunc("/api/v1/agent/execute", s.handleAgentExecute).Methods("POST")

	// MCP endpoints
	s.router.PathPrefix("/mcp").Handler(s.mcpServer)

	// Scheduler endpoints
	s.router.HandleFunc("/api/v1/scheduler/jobs", s.handleListJobs).Methods("GET")
	s.router.HandleFunc("/api/v1/scheduler/jobs/{id}", s.handleGetJob).Methods("GET")
	s.router.HandleFunc("/api/v1/scheduler/jobs", s.handleCreateJob).Methods("POST")
	s.router.HandleFunc("/api/v1/scheduler/jobs/{id}", s.handleDeleteJob).Methods("DELETE")
	s.router.HandleFunc("/api/v1/scheduler/jobs/{id}/enable", s.handleEnableJob).Methods("POST")
	s.router.HandleFunc("/api/v1/scheduler/jobs/{id}/disable", s.handleDisableJob).Methods("POST")

	// Visualization endpoints
	s.router.HandleFunc("/api/v1/visualize/pipeline", s.handleVisualizePipeline).Methods("POST")
	s.router.HandleFunc("/api/v1/visualize/status", s.handleVisualizeStatus).Methods("GET")
	s.router.HandleFunc("/api/v1/visualize/scheduler", s.handleVisualizeScheduler).Methods("GET")
	s.router.HandleFunc("/api/v1/visualize/plugins", s.handleVisualizePlugins).Methods("GET")

	// Performance monitoring endpoints
	s.router.HandleFunc("/api/v1/performance/metrics", s.handleGetPerformanceMetrics).Methods("GET")
	s.router.HandleFunc("/api/v1/performance/stats", s.handleGetPerformanceStats).Methods("GET")

	// Job monitoring endpoints
	s.router.HandleFunc("/api/v1/jobs", s.handleListJobExecutions).Methods("GET")
	s.router.HandleFunc("/api/v1/jobs/{id}", s.handleGetJobExecution).Methods("GET")
	s.router.HandleFunc("/api/v1/jobs/running", s.handleGetRunningJobs).Methods("GET")
	s.router.HandleFunc("/api/v1/jobs/recent", s.handleGetRecentJobs).Methods("GET")
	s.router.HandleFunc("/api/v1/jobs/export", s.handleExportJobs).Methods("GET")
	s.router.HandleFunc("/api/v1/jobs/statistics", s.handleGetJobStatistics).Methods("GET")

	// Configuration endpoints
	s.router.HandleFunc("/api/v1/config", s.handleGetConfig).Methods("GET")
	s.router.HandleFunc("/api/v1/config", s.handleUpdateConfig).Methods("PUT")
	s.router.HandleFunc("/api/v1/config/reload", s.handleReloadConfig).Methods("POST")
	s.router.HandleFunc("/api/v1/config/save", s.handleSaveConfig).Methods("POST")

	// Authentication endpoints
	auth := utils.GetAuthManager()
	s.router.HandleFunc("/api/v1/auth/login", s.handleLogin).Methods("POST")
	s.router.HandleFunc("/api/v1/auth/refresh", s.handleRefreshToken).Methods("POST")
	s.router.HandleFunc("/api/v1/auth/me", s.handleAuthMe).Methods("GET")
	s.router.HandleFunc("/api/v1/auth/users", s.handleListUsers).Methods("GET")
	s.router.HandleFunc("/api/v1/auth/apikeys", s.handleCreateAPIKey).Methods("POST")

	// Protected endpoints with authentication
	protected := s.router.PathPrefix("/api/v1/protected").Subrouter()
	protected.Use(auth.AuthMiddleware([]string{})) // Require authentication
	protected.HandleFunc("/pipelines", s.handleExecutePipeline).Methods("POST")
	protected.HandleFunc("/scheduler/jobs", s.handleCreateJob).Methods("POST")
	protected.HandleFunc("/config", s.handleUpdateConfig).Methods("PUT")
}

// Start starts the HTTP server
func (s *Server) Start(port string) error {
	log.Printf("Starting Mimir AIP server on port %s", port)
	return http.ListenAndServe(":"+port, s.router)
}

// handleHealth handles health check requests
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "healthy",
		"time":   time.Now().Format(time.RFC3339),
	})
}

// handleExecutePipeline handles pipeline execution requests
func (s *Server) handleExecutePipeline(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req PipelineExecutionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Determine pipeline to execute
	pipelineFile := req.PipelineFile
	if pipelineFile == "" && req.PipelineName != "" {
		// Try to find pipeline by name in config
		pipelineFile = fmt.Sprintf("pipelines/%s.yaml", req.PipelineName)
	}

	if pipelineFile == "" {
		http.Error(w, "Either pipeline_name or pipeline_file must be provided", http.StatusBadRequest)
		return
	}

	// Parse and execute pipeline
	config, err := utils.ParsePipeline(pipelineFile)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse pipeline: %v", err), http.StatusBadRequest)
		return
	}

	// Initialize context
	ctx := context.Background()
	globalContext := make(pipelines.PluginContext)
	if req.Context != nil {
		for k, v := range req.Context {
			globalContext[k] = v
		}
	}

	// Execute pipeline
	result, err := utils.ExecutePipeline(ctx, config)
	if err != nil {
		http.Error(w, fmt.Sprintf("Pipeline execution failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Return response
	response := PipelineExecutionResponse{
		Success:    result.Success,
		Error:      result.Error,
		Context:    result.Context,
		ExecutedAt: time.Now().Format(time.RFC3339),
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// handleListPipelines handles requests to list all pipelines
func (s *Server) handleListPipelines(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Try to read from config.yaml
	configPath := "config.yaml"
	pipelines, err := utils.ParseAllPipelines(configPath)
	if err != nil {
		// If config.yaml doesn't exist, return empty array
		json.NewEncoder(w).Encode([]utils.PipelineConfig{})
		return
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
	response := map[string]interface{}{
		"message": "Agent execution not yet implemented",
		"status":  "pending",
	}

	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(response)
}

// Middleware functions

// loggingMiddleware logs HTTP requests and responses
func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a response writer wrapper to capture status code
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Generate request ID
		requestID := fmt.Sprintf("%d", time.Now().UnixNano())

		// Add request ID to context
		ctx := context.WithValue(r.Context(), "request_id", requestID)
		r = r.WithContext(ctx)

		// Log request
		utils.GetLogger().Info("HTTP Request",
			utils.String("method", r.Method),
			utils.String("path", r.URL.Path),
			utils.String("remote_addr", r.RemoteAddr),
			utils.String("user_agent", r.Header.Get("User-Agent")),
			utils.RequestID(requestID),
			utils.Component("http"))

		// Call next handler
		next.ServeHTTP(rw, r)

		// Log response
		duration := time.Since(start)
		utils.GetLogger().Info("HTTP Response",
			utils.String("method", r.Method),
			utils.String("path", r.URL.Path),
			utils.Int("status", rw.statusCode),
			utils.Float("duration_ms", duration.Seconds()*1000),
			utils.RequestID(requestID),
			utils.Component("http"))
	})
}

// errorRecoveryMiddleware recovers from panics and logs errors
func (s *Server) errorRecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				// Log the panic
				utils.GetLogger().Error("Panic recovered",
					fmt.Errorf("panic: %v", err),
					utils.String("method", r.Method),
					utils.String("path", r.URL.Path),
					utils.Component("http"))

				// Return 500 error
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// corsMiddleware handles CORS headers
func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		config := s.config.GetConfig()

		if config.Server.EnableCORS {
			// Set CORS headers
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

			// Handle preflight requests
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Scheduler endpoint handlers

// handleListJobs handles requests to list all scheduled jobs
func (s *Server) handleListJobs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	jobs := s.scheduler.GetJobs()
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
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
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

	response := map[string]interface{}{
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
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
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

	response := map[string]interface{}{
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
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
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

	response := map[string]interface{}{
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

	response := map[string]interface{}{
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
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
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

	response := map[string]interface{}{
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
		response := map[string]interface{}{
			"valid":       false,
			"errors":      []string{err.Error()},
			"pipeline_id": pipelineID,
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := map[string]interface{}{
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

	response := map[string]interface{}{
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

	response := map[string]interface{}{
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

	response := map[string]interface{}{
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

	response := map[string]interface{}{
		"message": "Job disabled successfully",
		"job_id":  jobID,
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
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
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

	w.Write([]byte(visualization))
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
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
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

	response := map[string]interface{}{
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
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
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

	response := map[string]interface{}{
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

	response := map[string]interface{}{
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
	var users []map[string]interface{}

	allUsers := auth.GetUsers()
	for _, user := range allUsers {
		users = append(users, map[string]interface{}{
			"id":       user.ID,
			"username": user.Username,
			"roles":    user.Roles,
			"active":   user.Active,
		})
	}

	json.NewEncoder(w).Encode(map[string]interface{}{"users": users})
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
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
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

	response := map[string]interface{}{
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

	response := map[string]interface{}{
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

	response := map[string]interface{}{
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

	response := map[string]interface{}{
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
	stats := map[string]interface{}{
		"performance": metrics,
		"system": map[string]interface{}{
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

	w.Write(data)
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

	w.Write([]byte(output.String()))
}

// handleVisualizeScheduler handles requests to visualize scheduler status
func (s *Server) handleVisualizeScheduler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")

	jobs := s.scheduler.GetJobs()
	visualizer := utils.NewASCIIVisualizer()
	visualization := visualizer.VisualizeSchedulerJobs(jobs)

	w.Write([]byte(visualization))
}

// handleVisualizePlugins handles requests to visualize available plugins
func (s *Server) handleVisualizePlugins(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")

	visualizer := utils.NewASCIIVisualizer()
	visualization := visualizer.VisualizePluginRegistry(s.registry)

	w.Write([]byte(visualization))
}
