package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/DigitalTwin"
	knowledgegraph "github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/KnowledgeGraph"
	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/Ontology/schema_inference"
	storage "github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/Storage"
	"github.com/Mimir-AIP/Mimir-AIP-Go/utils"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// Version information
const (
	MimirMajorVersion = 0
	MimirMinorVersion = 2
	MimirPatchVersion = 0
	MimirVersionType  = "dev" // "release", "feature", "bugfix", "dev"
)

// GetMimirVersion returns the full version string
func GetMimirVersion() string {
	version := fmt.Sprintf("v%d.%d.%d", MimirMajorVersion, MimirMinorVersion, MimirPatchVersion)
	if MimirVersionType != "release" {
		version += "-" + MimirVersionType
	}
	return version
}

// handleVersion returns version information
func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	type VersionResponse struct {
		Version string `json:"version"`
		Build   string `json:"build"`
		Commit  string `json:"commit,omitempty"`
	}

	response := VersionResponse{
		Version: GetMimirVersion(),
		Build:   fmt.Sprintf("Go %s (%s/%s)", runtime.Compiler, runtime.GOOS, runtime.GOARCH),
	}

	writeJSONResponse(w, http.StatusOK, response)
}

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

	// Check if pipeline_id is provided (API-created pipeline)
	if req.PipelineID != "" {
		// Get pipeline from store
		store := utils.GetPipelineStore()
		pipeline, err := store.GetPipeline(req.PipelineID)
		if err != nil || pipeline == nil {
			writeErrorResponse(w, http.StatusNotFound, fmt.Sprintf("Pipeline not found: %s", req.PipelineID))
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

		// Convert PipelineDefinition to PipelineConfig for execution
		config := &pipeline.Config

		// Execute pipeline using the server's registry
		result, err := utils.ExecutePipelineWithRegistry(ctx, config, s.registry)
		if err != nil {
			utils.GetEventBus().Publish(utils.Event{
				Type:   utils.EventPipelineFailed,
				Source: "pipeline-execution-handler",
				Payload: map[string]any{
					"pipeline_id":   req.PipelineID,
					"pipeline_name": pipeline.Name,
					"error":         err.Error(),
				},
			})
			writeInternalServerErrorResponse(w, fmt.Sprintf("Pipeline execution failed: %v", err))
			return
		}

		// Publish appropriate event based on result
		if result.Success {
			event := utils.Event{
				Type:   utils.EventPipelineCompleted,
				Source: "pipeline-execution-handler",
				Payload: map[string]any{
					"pipeline_id":   req.PipelineID,
					"pipeline_name": pipeline.Name,
					"context":       result.Context,
				},
			}
			utils.GetLogger().Info("Publishing pipeline completion event",
				utils.String("pipeline_id", req.PipelineID),
				utils.String("pipeline_name", pipeline.Name),
				utils.String("event_type", event.Type))
			utils.GetEventBus().Publish(event)
			utils.GetLogger().Info("Pipeline completion event published",
				utils.String("pipeline_id", req.PipelineID))
		} else {
			utils.GetEventBus().Publish(utils.Event{
				Type:   utils.EventPipelineFailed,
				Source: "pipeline-execution-handler",
				Payload: map[string]any{
					"pipeline_id":   req.PipelineID,
					"pipeline_name": pipeline.Name,
					"error":         result.Error,
				},
			})
		}

		// Return response
		response := PipelineExecutionResponse{
			Success:    result.Success,
			Error:      result.Error,
			Context:    result.Context,
			ExecutedAt: time.Now().Format(time.RFC3339),
		}

		writeSuccessResponse(w, response)
		return
	}

	// Legacy: Determine pipeline file to execute (YAML-based pipelines)
	pipelineFile := req.PipelineFile
	if pipelineFile == "" && req.PipelineName != "" {
		// Try to find pipeline by name in config
		pipelineFile = fmt.Sprintf("pipelines/%s.yaml", req.PipelineName)
	}

	if pipelineFile == "" {
		writeBadRequestResponse(w, "Either pipeline_id, pipeline_name or pipeline_file must be provided")
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
		// Publish pipeline failed event
		utils.GetEventBus().Publish(utils.Event{
			Type:   utils.EventPipelineFailed,
			Source: "pipeline-execution-handler",
			Payload: map[string]any{
				"pipeline_file": pipelineFile,
				"pipeline_name": req.PipelineName,
				"error":         err.Error(),
			},
		})
		writeInternalServerErrorResponse(w, fmt.Sprintf("Pipeline execution failed: %v", err))
		return
	}

	// Publish appropriate event based on result
	if result.Success {
		utils.GetEventBus().Publish(utils.Event{
			Type:   utils.EventPipelineCompleted,
			Source: "pipeline-execution-handler",
			Payload: map[string]any{
				"pipeline_file": pipelineFile,
				"pipeline_name": req.PipelineName,
				"context":       result.Context,
			},
		})
	} else {
		utils.GetEventBus().Publish(utils.Event{
			Type:   utils.EventPipelineFailed,
			Source: "pipeline-execution-handler",
			Payload: map[string]any{
				"pipeline_file": pipelineFile,
				"pipeline_name": req.PipelineName,
				"error":         result.Error,
			},
		})
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

// handleExecutePipelineByID handles pipeline execution requests by pipeline ID
func (s *Server) handleExecutePipelineByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	pipelineID := vars["id"]

	// Get pipeline from store
	store := utils.GetPipelineStore()
	pipeline, err := store.GetPipeline(pipelineID)
	if err != nil || pipeline == nil {
		writeErrorResponse(w, http.StatusNotFound, fmt.Sprintf("Pipeline not found: %s", pipelineID))
		return
	}

	// Parse optional execution context from request body
	var req struct {
		Context map[string]any `json:"context"`
	}
	if r.Body != nil && r.Body != http.NoBody {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			// Ignore parse errors - context is optional
			req.Context = make(map[string]any)
		}
	}

	// Initialize context
	ctx := context.Background()
	globalContext := pipelines.NewPluginContext()
	if req.Context != nil {
		for k, v := range req.Context {
			globalContext.Set(k, v)
		}
	}

	// Convert PipelineDefinition to PipelineConfig for execution
	config := &pipeline.Config

	// Execute pipeline using the server's registry
	result, err := utils.ExecutePipelineWithRegistry(ctx, config, s.registry)
	if err != nil {
		// Publish pipeline failed event
		utils.GetEventBus().Publish(utils.Event{
			Type:   utils.EventPipelineFailed,
			Source: "pipeline-execution-handler",
			Payload: map[string]any{
				"pipeline_id":   pipelineID,
				"pipeline_name": pipeline.Name,
				"error":         err.Error(),
			},
		})
		writeInternalServerErrorResponse(w, fmt.Sprintf("Pipeline execution failed: %v", err))
		return
	}

	// Publish appropriate event based on result
	utils.GetLogger().Info("Pipeline execution completed",
		utils.String("pipeline_id", pipelineID),
		utils.Bool("result_success", result.Success),
		utils.String("result_error", result.Error))

	if result.Success {
		event := utils.Event{
			Type:   utils.EventPipelineCompleted,
			Source: "pipeline-execution-handler",
			Payload: map[string]any{
				"pipeline_id":   pipelineID,
				"pipeline_name": pipeline.Name,
				"context":       result.Context,
			},
		}
		utils.GetLogger().Info("Publishing EventPipelineCompleted",
			utils.String("pipeline_id", pipelineID),
			utils.String("event_type", event.Type))
		utils.GetEventBus().Publish(event)
		utils.GetLogger().Info("EventPipelineCompleted published",
			utils.String("pipeline_id", pipelineID))
	} else {
		utils.GetEventBus().Publish(utils.Event{
			Type:   utils.EventPipelineFailed,
			Source: "pipeline-execution-handler",
			Payload: map[string]any{
				"pipeline_id":   pipelineID,
				"pipeline_name": pipeline.Name,
				"error":         result.Error,
			},
		})
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
	log.Printf("=== handleListPipelines CALLED ===")
	store := utils.GetPipelineStore()
	log.Printf("=== Store pointer: %p ===", store)
	pipelines, err := store.ListPipelines(nil)
	log.Printf("=== ListPipelines returned %d pipelines, err=%v ===", len(pipelines), err)
	if err != nil {
		log.Printf("ERROR: Failed to list pipelines: %v", err)
		writeJSONResponse(w, http.StatusOK, []*utils.PipelineDefinition{})
		return
	}

	// DEBUG: Log what we're about to return
	log.Printf("DEBUG: Returning %d pipelines for JSON encoding", len(pipelines))

	// Manually flatten for JSON encoding - json.NewEncoder doesn't call custom MarshalJSON on slices
	flattened := make([]map[string]interface{}, len(pipelines))
	for i, p := range pipelines {
		log.Printf("DEBUG: Flattening Pipeline[%d] - Metadata.ID=%s, Pipeline.ID=%s, Pipeline.Name=%s", i, p.Metadata.ID, p.ID, p.Name)
		flattened[i] = map[string]interface{}{
			"id":       p.ID,
			"name":     p.Name,
			"metadata": p.Metadata,
			"config":   p.Config,
		}
		log.Printf("DEBUG: Flattened Pipeline[%d] - id=%s, name=%s", i, flattened[i]["id"], flattened[i]["name"])
	}

	log.Printf("DEBUG: Sending flattened response with %d pipelines", len(flattened))
	writeJSONResponse(w, http.StatusOK, flattened)
}

// handleGetPipeline handles requests to get a specific pipeline
func (s *Server) handleGetPipeline(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	pipelineID := vars["id"]

	// Try to find pipeline in the dynamic store first (for API-created pipelines)
	store := utils.GetPipelineStore()
	pipeline, err := store.GetPipeline(pipelineID)
	if err == nil && pipeline != nil {
		writeJSONResponse(w, http.StatusOK, pipeline)
		return
	}

	// Fall back to config file for static pipelines (matched by name)
	configPath := "config.yaml"
	pipelines, err := utils.ParseAllPipelines(configPath)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to read pipelines")
		return
	}

	for _, p := range pipelines {
		if p.Name == pipelineID {
			writeJSONResponse(w, http.StatusOK, p)
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

	// Automatically create scheduled job for continuous monitoring
	// Use metadata.Schedule if provided, otherwise default to every 5 minutes
	scheduleCron := pipeline.Metadata.Schedule
	if scheduleCron == "" {
		scheduleCron = "*/5 * * * *" // Default: every 5 minutes
	}

	jobID := fmt.Sprintf("auto-scheduled-%s", pipeline.Metadata.ID)
	jobName := fmt.Sprintf("Auto: %s", pipeline.Metadata.Name)

	if err := s.scheduler.AddJob(jobID, jobName, pipeline.Metadata.ID, scheduleCron); err != nil {
		utils.GetLogger().Warn("Failed to auto-create scheduled job",
			utils.String("pipeline_id", pipeline.Metadata.ID),
			utils.String("error", err.Error()))
	} else {
		utils.GetLogger().Info("Auto-created scheduled job for pipeline",
			utils.String("job_id", jobID),
			utils.String("pipeline_id", pipeline.Metadata.ID),
			utils.String("schedule", scheduleCron))
	}

	// Store auto-monitoring info in response
	response := map[string]any{
		"message":          "Pipeline created successfully",
		"pipeline":         pipeline,
		"auto_scheduled":   true,
		"schedule_cron":    scheduleCron,
		"scheduled_job_id": jobID,
	}

	writeJSONResponse(w, http.StatusCreated, response)
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

// handleAuthCheck returns authentication status (for frontend auth checks)
func (s *Server) handleAuthCheck(w http.ResponseWriter, r *http.Request) {
	user, ok := utils.GetUserFromContext(r.Context())
	if !ok {
		writeJSONResponse(w, http.StatusUnauthorized, map[string]any{
			"authenticated": false,
			"error":         "No valid session",
		})
		return
	}

	writeJSONResponse(w, http.StatusOK, map[string]any{
		"authenticated": true,
		"user": map[string]any{
			"username": user.Username,
			"roles":    user.Roles,
		},
	})
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

// handleLogout handles user logout
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	// In a JWT-based system, logout is typically client-side (remove token)
	// But we can add any server-side cleanup here if needed

	writeJSONResponse(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "Logged out successfully",
	})
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

// Data ingestion handlers for flexible plugin-based data upload and processing

// ExtendedPluginInfo represents detailed information about an input plugin
type ExtendedPluginInfo struct {
	Type             string         `json:"type"`
	Name             string         `json:"name"`
	Description      string         `json:"description"`
	ConfigSchema     map[string]any `json:"config_schema"`
	SupportedFormats []string       `json:"supported_formats"`
}

// DataUploadRequest represents a file upload request
type DataUploadRequest struct {
	PluginType string         `json:"plugin_type"`
	PluginName string         `json:"plugin_name"`
	Config     map[string]any `json:"config"`
	File       []byte         `json:"file,omitempty"` // For API uploads
	FileName   string         `json:"file_name,omitempty"`
}

// DataPreviewRequest represents a request to preview parsed data
type DataPreviewRequest struct {
	UploadID   string         `json:"upload_id"`
	PluginType string         `json:"plugin_type"`
	PluginName string         `json:"plugin_name"`
	Config     map[string]any `json:"config"`
	MaxRows    int            `json:"max_rows,omitempty"` // Limit preview rows
	Profile    bool           `json:"profile,omitempty"`  // Whether to include profiling
}

// ColumnProfile represents comprehensive profiling statistics for a column
type ColumnProfile struct {
	ColumnName       string           `json:"column_name"`
	DataType         string           `json:"data_type"`
	TotalCount       int              `json:"total_count"`
	DistinctCount    int              `json:"distinct_count"`
	DistinctPercent  float64          `json:"distinct_percent"`
	NullCount        int              `json:"null_count"`
	NullPercent      float64          `json:"null_percent"`
	MinValue         any              `json:"min_value,omitempty"`
	MaxValue         any              `json:"max_value,omitempty"`
	Mean             float64          `json:"mean,omitempty"`
	Median           float64          `json:"median,omitempty"`
	StdDev           float64          `json:"std_dev,omitempty"`
	MinLength        int              `json:"min_length,omitempty"`
	MaxLength        int              `json:"max_length,omitempty"`
	AvgLength        float64          `json:"avg_length,omitempty"`
	TopValues        []ValueFrequency `json:"top_values"`
	DataQualityScore float64          `json:"data_quality_score"`
	QualityIssues    []string         `json:"quality_issues"`
}

// ValueFrequency represents a value and its frequency
type ValueFrequency struct {
	Value     any     `json:"value"`
	Count     int     `json:"count"`
	Frequency float64 `json:"frequency"`
}

// DataProfileSummary represents overall profiling summary
type DataProfileSummary struct {
	TotalRows            int             `json:"total_rows"`
	TotalColumns         int             `json:"total_columns"`
	TotalDistinctValues  int             `json:"total_distinct_values"`
	OverallQualityScore  float64         `json:"overall_quality_score"`
	SuggestedPrimaryKeys []string        `json:"suggested_primary_keys"`
	ColumnProfiles       []ColumnProfile `json:"column_profiles"`
}

// DataSelection represents selected data for ontology generation
type DataSelection struct {
	UploadID        string             `json:"upload_id"`
	SelectedColumns []string           `json:"selected_columns"`
	ColumnMappings  map[string]string  `json:"column_mappings,omitempty"` // column -> property name
	Relationships   []RelationshipSpec `json:"relationships,omitempty"`
	CreateTwin      bool               `json:"create_twin,omitempty"` // Whether to create a Digital Twin
}

// RelationshipSpec defines a relationship between columns/entities
type RelationshipSpec struct {
	SourceColumn     string  `json:"source_column"`
	TargetColumn     string  `json:"target_column"`
	RelationshipType string  `json:"relationship_type"`
	Strength         float64 `json:"strength,omitempty"`
}

// handleListInputPlugins lists all available input plugins for data ingestion
func (s *Server) handleListInputPlugins(w http.ResponseWriter, r *http.Request) {
	plugins := []ExtendedPluginInfo{}

	// Get all plugins from registry
	allPlugins := s.registry.GetAllPlugins()

	// Filter for Input plugins
	if inputPlugins, exists := allPlugins["Input"]; exists {
		for name := range inputPlugins {
			pluginInfo := ExtendedPluginInfo{
				Type:        "Input",
				Name:        name,
				Description: fmt.Sprintf("%s input plugin", name),
			}

			// Add basic config schema
			pluginInfo.ConfigSchema = map[string]any{
				"type": "object",
				"properties": map[string]any{
					"file_path": map[string]any{
						"type":        "string",
						"description": "Path to the input file",
					},
				},
				"required": []string{"file_path"},
			}

			// Add supported formats based on plugin name
			switch name {
			case "csv":
				pluginInfo.SupportedFormats = []string{"csv", "tsv", "txt"}
				pluginInfo.Description = "CSV and delimited text files"
				pluginInfo.ConfigSchema = map[string]any{
					"type": "object",
					"properties": map[string]any{
						"file_path": map[string]any{
							"type":        "string",
							"description": "Path to the CSV file",
						},
						"has_headers": map[string]any{
							"type":        "boolean",
							"description": "Whether first row contains headers",
							"default":     true,
						},
						"delimiter": map[string]any{
							"type":        "string",
							"description": "Field delimiter",
							"default":     ",",
						},
					},
					"required": []string{"file_path"},
				}
			case "markdown":
				pluginInfo.SupportedFormats = []string{"md", "markdown"}
				pluginInfo.Description = "Markdown documents with sections and metadata"
				pluginInfo.ConfigSchema = map[string]any{
					"type": "object",
					"properties": map[string]any{
						"file_path": map[string]any{
							"type":        "string",
							"description": "Path to the Markdown file",
						},
						"extract_sections": map[string]any{
							"type":        "boolean",
							"description": "Extract headings and sections",
							"default":     true,
						},
						"extract_links": map[string]any{
							"type":        "boolean",
							"description": "Extract links and images",
							"default":     true,
						},
						"extract_metadata": map[string]any{
							"type":        "boolean",
							"description": "Extract YAML frontmatter",
							"default":     true,
						},
					},
					"required": []string{"file_path"},
				}
			case "excel":
				pluginInfo.SupportedFormats = []string{"xlsx", "xls"}
				pluginInfo.Description = "Excel spreadsheets (.xlsx)"
				pluginInfo.ConfigSchema = map[string]any{
					"type": "object",
					"properties": map[string]any{
						"file_path": map[string]any{
							"type":        "string",
							"description": "Path to the Excel file",
						},
						"sheet_name": map[string]any{
							"type":        "string",
							"description": "Sheet name to read (optional)",
						},
						"has_headers": map[string]any{
							"type":        "boolean",
							"description": "Whether first row contains headers",
							"default":     true,
						},
					},
					"required": []string{"file_path"},
				}
			}

			plugins = append(plugins, pluginInfo)
		}
	}

	writeJSONResponse(w, http.StatusOK, map[string]any{
		"plugins": plugins,
	})
}

// handleUploadData handles file uploads for data ingestion
func (s *Server) handleUploadData(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form
	err := r.ParseMultipartForm(32 << 20) // 32MB max
	if err != nil {
		writeBadRequestResponse(w, fmt.Sprintf("Failed to parse multipart form: %v", err))
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeBadRequestResponse(w, fmt.Sprintf("Failed to get file: %v", err))
		return
	}
	defer file.Close()

	// Get plugin info from form
	pluginType := r.FormValue("plugin_type")
	pluginName := r.FormValue("plugin_name")

	if pluginType == "" || pluginName == "" {
		writeBadRequestResponse(w, "plugin_type and plugin_name are required")
		return
	}

	// Read file content
	fileContent := make([]byte, header.Size)
	_, err = file.Read(fileContent)
	if err != nil {
		writeBadRequestResponse(w, fmt.Sprintf("Failed to read file: %v", err))
		return
	}

	// Generate upload ID and create temporary file path
	uploadID := fmt.Sprintf("upload_%d_%s", time.Now().Unix(), header.Filename)
	tempDir := "/tmp/mimir-uploads"
	os.MkdirAll(tempDir, 0755) // Create directory if it doesn't exist
	tempFilePath := fmt.Sprintf("%s/%s", tempDir, uploadID)

	// Save file to temporary location
	tempFile, err := os.Create(tempFilePath)
	if err != nil {
		writeBadRequestResponse(w, fmt.Sprintf("Failed to create temp file: %v", err))
		return
	}
	defer tempFile.Close()

	// Reset file reader to beginning
	file.Seek(0, 0)
	_, err = io.Copy(tempFile, file)
	if err != nil {
		os.Remove(tempFilePath) // Clean up on error
		writeBadRequestResponse(w, fmt.Sprintf("Failed to save file: %v", err))
		return
	}

	// Basic validation based on plugin type
	err = s.validateUploadedFile(pluginType, pluginName, header.Filename, fileContent)
	if err != nil {
		writeBadRequestResponse(w, fmt.Sprintf("File validation failed: %v", err))
		return
	}

	writeJSONResponse(w, http.StatusOK, map[string]any{
		"upload_id":   uploadID,
		"filename":    header.Filename,
		"size":        header.Size,
		"plugin_type": pluginType,
		"plugin_name": pluginName,
		"message":     "File uploaded successfully",
	})
}

// handlePreviewData previews parsed data from uploaded file
func (s *Server) handlePreviewData(w http.ResponseWriter, r *http.Request) {
	var req DataPreviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequestResponse(w, fmt.Sprintf("Invalid request body: %v", err))
		return
	}

	if req.UploadID == "" || req.PluginType == "" || req.PluginName == "" {
		writeBadRequestResponse(w, "upload_id, plugin_type, and plugin_name are required")
		return
	}

	// Set default max rows if not specified
	if req.MaxRows <= 0 {
		req.MaxRows = 100
	}

	// Get the plugin
	plugin, err := s.registry.GetPlugin(req.PluginType, req.PluginName)
	if err != nil {
		writeBadRequestResponse(w, fmt.Sprintf("Plugin not found: %v", err))
		return
	}

	// For now, we'll simulate file access (in real implementation, retrieve from storage)
	// Create a temporary context with file path
	globalContext := pipelines.NewPluginContext()

	// Set file path in config (this would normally come from stored upload metadata)
	if req.Config == nil {
		req.Config = make(map[string]any)
	}
	req.Config["file_path"] = fmt.Sprintf("/tmp/mimir-uploads/%s", req.UploadID)

	// Execute plugin to parse data
	stepConfig := pipelines.StepConfig{
		Name:   "data_preview",
		Plugin: fmt.Sprintf("%s.%s", req.PluginType, req.PluginName),
		Config: req.Config,
		Output: "preview_data",
	}

	result, err := plugin.ExecuteStep(r.Context(), stepConfig, globalContext)
	if err != nil {
		writeBadRequestResponse(w, fmt.Sprintf("Data parsing failed: %v", err))
		return
	}

	// Get parsed data
	parsedData, ok := result.Get("preview_data")
	if !ok {
		writeBadRequestResponse(w, "Failed to get parsed data from plugin")
		return
	}

	// Limit rows for preview
	limitedData := s.limitPreviewData(parsedData, req.MaxRows)

	response := map[string]any{
		"upload_id":    req.UploadID,
		"plugin_type":  req.PluginType,
		"plugin_name":  req.PluginName,
		"data":         limitedData,
		"preview_rows": req.MaxRows,
		"message":      "Data preview generated successfully",
	}

	// Check if profiling is requested via query parameter or request body
	profileParam := r.URL.Query().Get("profile")
	shouldProfile := req.Profile || profileParam == "true"

	// Add profiling if requested
	if shouldProfile {
		dataMap, ok := parsedData.(map[string]any)
		if ok {
			// Profile the full dataset (before limiting)
			profileSummary := profileDataset(dataMap, 10000) // Sample up to 10k rows for profiling
			response["profile"] = profileSummary
			response["message"] = "Data preview with profiling generated successfully"
		}
	}

	writeJSONResponse(w, http.StatusOK, response)
}

// handleSelectData handles data column/relationship selection for ontology generation
func (s *Server) handleSelectData(w http.ResponseWriter, r *http.Request) {
	var selection DataSelection
	if err := json.NewDecoder(r.Body).Decode(&selection); err != nil {
		writeBadRequestResponse(w, fmt.Sprintf("Invalid request body: %v", err))
		return
	}

	if selection.UploadID == "" {
		writeBadRequestResponse(w, "upload_id is required")
		return
	}

	if len(selection.SelectedColumns) == 0 {
		writeBadRequestResponse(w, "At least one column must be selected")
		return
	}

	// Generate ontology from selection
	ontology, err := s.generateOntologyFromSelection(selection)
	if err != nil {
		writeBadRequestResponse(w, fmt.Sprintf("Ontology generation failed: %v", err))
		return
	}

	response := map[string]any{
		"selection": selection,
		"ontology":  ontology,
		"message":   "Ontology generated successfully from selected data",
	}

	// Save ontology to database if persistence is available
	if s.persistence != nil {
		ontologyID, _ := ontology["id"].(string)
		ontologyName, _ := ontology["name"].(string)
		ontologyDesc, _ := ontology["description"].(string)
		ontologyVersion, _ := ontology["version"].(string)
		ontologyContent, _ := ontology["content"].(string)
		utils.GetLogger().Info(fmt.Sprintf("Ontology content type: %T, length: %d", ontology["content"], len(ontologyContent)))
		if ontologyContent == "" {
			utils.GetLogger().Warn("Ontology content is empty, skipping file write")
			return
		}

		ont := &storage.Ontology{
			ID:          ontologyID,
			Name:        ontologyName,
			Description: ontologyDesc,
			Version:     ontologyVersion,
			FilePath:    filepath.Join(s.ontologyDir, ontologyID+".ttl"),
			TDB2Graph:   "http://mimir.ai/ontology/" + ontologyID,
			Format:      "turtle",
			Status:      "active",
			CreatedBy:   "autonomous_flow",
			Metadata:    "{}",
		}

		err = s.persistence.CreateOntology(r.Context(), ont)
		if err != nil {
			utils.GetLogger().Warn(fmt.Sprintf("Failed to persist ontology: %v", err))
		} else {
			// Also save the content to a file
			utils.GetLogger().Info(fmt.Sprintf("Writing ontology file to: %s (content length: %d)", ont.FilePath, len(ontologyContent)))
			err = os.WriteFile(ont.FilePath, []byte(ontologyContent), 0644)
			if err != nil {
				utils.GetLogger().Warn(fmt.Sprintf("Failed to write ontology file: %v", err))
			} else {
				utils.GetLogger().Info(fmt.Sprintf("Successfully wrote ontology file: %s", ont.FilePath))
			}

			// Load ontology into TDB2 knowledge graph if available
			if s.tdb2Backend != nil {
				err = s.tdb2Backend.LoadOntology(r.Context(), ont.TDB2Graph, ontologyContent, "turtle")
				if err != nil {
					utils.GetLogger().Warn(fmt.Sprintf("Failed to load ontology into TDB2: %v", err))
					response["kg_warning"] = "Ontology saved but failed to load into knowledge graph"
				} else {
					utils.GetLogger().Info(fmt.Sprintf("Loaded ontology %s into TDB2 graph %s", ontologyID, ont.TDB2Graph))
					response["kg_graph"] = ont.TDB2Graph
				}
			}
		}
	}

	// Optionally create Digital Twin from ontology
	if selection.CreateTwin {
		twin, err := s.createDigitalTwinFromOntology(r.Context(), ontology)
		if err != nil {
			utils.GetLogger().Warn(fmt.Sprintf("Failed to create digital twin: %v", err))
			response["twin_error"] = err.Error()
		} else {
			// Extract scenario IDs from twin base state
			scenarioIDs := []string{}
			if scenarios, ok := twin.BaseState["default_scenarios"].([]string); ok {
				scenarioIDs = scenarios
			}

			response["digital_twin"] = map[string]any{
				"id":                 twin.ID,
				"name":               twin.Name,
				"description":        twin.Description,
				"model_type":         twin.ModelType,
				"entity_count":       len(twin.Entities),
				"relationship_count": len(twin.Relationships),
				"scenario_ids":       scenarioIDs,
				"scenario_count":     len(scenarioIDs),
			}
			response["message"] = "Ontology and Digital Twin generated successfully"
		}
	}

	writeJSONResponse(w, http.StatusOK, response)
}

// validateUploadedFile performs basic validation based on plugin type
func (s *Server) validateUploadedFile(pluginType, pluginName, filename string, content []byte) error {
	// Check file extension
	switch pluginName {
	case "csv":
		if !strings.HasSuffix(strings.ToLower(filename), ".csv") &&
			!strings.HasSuffix(strings.ToLower(filename), ".tsv") &&
			!strings.HasSuffix(strings.ToLower(filename), ".txt") {
			return fmt.Errorf("invalid file type for CSV plugin: %s", filename)
		}
	case "markdown":
		if !strings.HasSuffix(strings.ToLower(filename), ".md") &&
			!strings.HasSuffix(strings.ToLower(filename), ".markdown") {
			return fmt.Errorf("invalid file type for Markdown plugin: %s", filename)
		}
	case "excel":
		if !strings.HasSuffix(strings.ToLower(filename), ".xlsx") &&
			!strings.HasSuffix(strings.ToLower(filename), ".xls") {
			return fmt.Errorf("invalid file type for Excel plugin: %s", filename)
		}
	}

	// Basic content validation
	if len(content) == 0 {
		return fmt.Errorf("file is empty")
	}

	// Plugin-specific validation could go here
	switch pluginName {
	case "csv":
		// Check if it looks like CSV data
		contentStr := string(content)
		if !strings.Contains(contentStr, ",") && !strings.Contains(contentStr, "\t") {
			return fmt.Errorf("file does not appear to be CSV or delimited data")
		}
	case "markdown":
		// Check if it has markdown-like content
		contentStr := string(content)
		if !strings.Contains(contentStr, "#") && !strings.Contains(contentStr, "[") {
			return fmt.Errorf("file does not appear to be Markdown format")
		}
	}

	return nil
}

// limitPreviewData limits the number of rows returned for preview
func (s *Server) limitPreviewData(data any, maxRows int) any {
	// This is a simple implementation - in reality, you'd need to handle
	// the specific data structure returned by each plugin
	dataMap, ok := data.(map[string]any)
	if !ok {
		return data
	}

	// Check if it has rows array
	if rows, exists := dataMap["rows"]; exists {
		if rowsSlice, ok := rows.([]any); ok && len(rowsSlice) > maxRows {
			dataMap["rows"] = rowsSlice[:maxRows]
			dataMap["preview_limited"] = true
			dataMap["total_rows"] = len(rowsSlice)
		}
	}

	return dataMap
}

// generateOntologyFromSelection creates an ontology from selected data using schema inference
func (s *Server) generateOntologyFromSelection(selection DataSelection) (map[string]any, error) {
	// First, we need to retrieve the original data from the upload
	// For now, we'll create a mock dataset based on the selection
	// In a real implementation, you'd retrieve the actual parsed data

	// Create mock data structure based on selected columns
	mockData := make([]map[string]interface{}, 3) // Sample rows
	for i := range mockData {
		mockData[i] = make(map[string]interface{})
		for _, col := range selection.SelectedColumns {
			// Generate sample data based on column name patterns
			mockData[i][col] = s.generateSampleValue(col, i+1)
		}
	}

	// Use schema inference engine
	inferenceConfig := schema_inference.InferenceConfig{
		SampleSize:          100,
		ConfidenceThreshold: 0.8,
		EnableRelationships: true,
		EnableConstraints:   true,
	}
	inferenceEngine := schema_inference.NewSchemaInferenceEngine(inferenceConfig)

	// Infer schema from the mock data
	schema, err := inferenceEngine.InferSchema(mockData, fmt.Sprintf("Dataset_%s", selection.UploadID))
	if err != nil {
		return nil, fmt.Errorf("schema inference failed: %w", err)
	}

	// Filter schema to only include selected columns
	filteredColumns := []schema_inference.ColumnSchema{}
	for _, col := range schema.Columns {
		for _, selectedCol := range selection.SelectedColumns {
			if col.Name == selectedCol {
				filteredColumns = append(filteredColumns, col)
				break
			}
		}
	}
	schema.Columns = filteredColumns

	// Use ontology generator
	generatorConfig := schema_inference.OntologyConfig{
		BaseURI:         "http://mimir-aip.io/ontology/",
		OntologyPrefix:  "mimir",
		IncludeMetadata: true,
		IncludeComments: true,
		ClassNaming:     "pascal",
		PropertyNaming:  "camel",
	}
	generator := schema_inference.NewOntologyGenerator(generatorConfig)

	// Generate ontology
	ontology, err := generator.GenerateOntology(schema)
	if err != nil {
		return nil, fmt.Errorf("ontology generation failed: %w", err)
	}

	// Convert to API response format
	utils.GetLogger().Info(fmt.Sprintf("Generated ontology %s with content length: %d", ontology.ID, len(ontology.Content)))
	result := map[string]any{
		"id":          ontology.ID,
		"name":        ontology.Name,
		"description": ontology.Description,
		"version":     ontology.Version,
		"format":      ontology.Format,
		"content":     ontology.Content,
		"classes":     ontology.Classes,
		"properties":  ontology.Properties,
		"metadata":    ontology.Metadata,
		"created_at":  ontology.GeneratedAt.Format(time.RFC3339),
	}

	return result, nil
}

// createDigitalTwinFromOntology creates a Digital Twin from an ontology
func (s *Server) createDigitalTwinFromOntology(ctx context.Context, ontologyData map[string]any) (*DigitalTwin.DigitalTwin, error) {
	// Extract ontology information
	ontologyID, _ := ontologyData["id"].(string)
	ontologyName, _ := ontologyData["name"].(string)
	classes, _ := ontologyData["classes"].([]any)
	properties, _ := ontologyData["properties"].([]any)

	// Generate twin ID
	twinID := fmt.Sprintf("twin_%d", time.Now().Unix())

	// Create digital twin structure
	twin := DigitalTwin.NewDigitalTwin(twinID, ontologyID, "data_model", ontologyName)
	twin.Description = fmt.Sprintf("Digital Twin created from ontology: %s", ontologyName)

	// Create entities from ontology classes
	for _, classAny := range classes {
		classMap, ok := classAny.(map[string]any)
		if !ok {
			continue
		}

		className, _ := classMap["name"].(string)
		classURI, _ := classMap["uri"].(string)

		// Create sample entities for each class (in a real implementation,
		// these would come from the actual data)
		for i := 1; i <= 3; i++ {
			entityURI := fmt.Sprintf("%s/%d", classURI, i)
			entityLabel := fmt.Sprintf("%s %d", className, i)

			entity := DigitalTwin.NewTwinEntity(entityURI, classURI, entityLabel)

			// Add properties to entity based on ontology datatype properties
			for _, propAny := range properties {
				propMap, ok := propAny.(map[string]any)
				if !ok {
					continue
				}

				propType, _ := propMap["type"].(string)
				if propType != "datatype" {
					continue
				}

				propDomain, _ := propMap["domain"].(string)
				if propDomain != classURI {
					continue
				}

				propName, _ := propMap["name"].(string)
				// Generate sample value based on property name
				entity.Properties[propName] = s.generateSampleValue(propName, i)
			}

			twin.Entities = append(twin.Entities, *entity)
		}
	}

	// Create relationships from object properties
	relationshipID := 1
	for _, propAny := range properties {
		propMap, ok := propAny.(map[string]any)
		if !ok {
			continue
		}

		propType, _ := propMap["type"].(string)
		if propType != "object" {
			continue
		}

		propName, _ := propMap["name"].(string)
		propDomain, _ := propMap["domain"].(string)
		propRange, _ := propMap["range"].(string)

		// Create sample relationships between entities
		// In a real implementation, relationships would be inferred from data
		for i := 0; i < len(twin.Entities)-1; i++ {
			if twin.Entities[i].Type == propDomain && twin.Entities[i+1].Type == propRange {
				relID := fmt.Sprintf("rel_%d", relationshipID)
				rel := DigitalTwin.NewTwinRelationship(
					relID,
					twin.Entities[i].URI,
					twin.Entities[i+1].URI,
					propName,
					0.8, // Default relationship strength
				)
				twin.Relationships = append(twin.Relationships, *rel)
				relationshipID++
			}
		}
	}

	// Serialize twin state for storage
	twinState := map[string]interface{}{
		"entities":      twin.Entities,
		"relationships": twin.Relationships,
	}
	twinStateJSON, err := json.Marshal(twinState)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal twin state: %w", err)
	}

	// Store in database
	if s.persistence != nil {
		err = s.persistence.CreateDigitalTwin(
			ctx,
			twin.ID,
			twin.OntologyID,
			twin.Name,
			twin.Description,
			twin.ModelType,
			string(twinStateJSON),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to persist digital twin: %w", err)
		}
	}

	// Generate and save default scenarios for the new twin
	scenarios := generateDefaultScenarios(twin)
	scenarioIDs := []string{}

	for _, scenario := range scenarios {
		err := saveScenarioToDatabase(ctx, s.persistence, &scenario)
		if err != nil {
			utils.GetLogger().Warn(fmt.Sprintf("Failed to save scenario %s: %v", scenario.Name, err))
		} else {
			scenarioIDs = append(scenarioIDs, scenario.ID)
		}
	}

	// Store scenario IDs in the twin's base state for reference
	if len(scenarioIDs) > 0 {
		twin.BaseState["default_scenarios"] = scenarioIDs
	}

	return twin, nil
}

// generateDefaultScenarios generates realistic simulation scenarios for a newly created Digital Twin
func generateDefaultScenarios(twin *DigitalTwin.DigitalTwin) []DigitalTwin.SimulationScenario {
	scenarios := []DigitalTwin.SimulationScenario{}

	// Scenario 1: Baseline - Normal operations with no events
	baselineScenario := DigitalTwin.SimulationScenario{
		ID:          fmt.Sprintf("scenario_%s_baseline", twin.ID),
		TwinID:      twin.ID,
		Name:        "Baseline Operations",
		Description: "Normal operating conditions with no disruptions. Establishes performance baseline for comparison.",
		Type:        "baseline",
		Events:      []DigitalTwin.SimulationEvent{},
		Duration:    30, // 30 steps
		CreatedAt:   time.Now(),
	}
	scenarios = append(scenarios, baselineScenario)

	// Scenario 2: Data Quality Issues - Simulates missing/invalid data
	dataQualityEvents := generateDataQualityEvents(twin)
	dataQualityScenario := DigitalTwin.SimulationScenario{
		ID:          fmt.Sprintf("scenario_%s_data_quality", twin.ID),
		TwinID:      twin.ID,
		Name:        "Data Quality Issues",
		Description: "Simulates data quality problems including missing values, invalid data, and entity unavailability.",
		Type:        "data_quality_issue",
		Events:      dataQualityEvents,
		Duration:    40, // 40 steps
		CreatedAt:   time.Now(),
	}
	scenarios = append(scenarios, dataQualityScenario)

	// Scenario 3: Capacity Test - Simulates high volume/load
	capacityEvents := generateCapacityTestEvents(twin)
	capacityScenario := DigitalTwin.SimulationScenario{
		ID:          fmt.Sprintf("scenario_%s_capacity", twin.ID),
		TwinID:      twin.ID,
		Name:        "Capacity Stress Test",
		Description: "Tests system behavior under high load conditions with demand surges and increased utilization.",
		Type:        "capacity_test",
		Events:      capacityEvents,
		Duration:    50, // 50 steps
		CreatedAt:   time.Now(),
	}
	scenarios = append(scenarios, capacityScenario)

	return scenarios
}

// generateDataQualityEvents creates events simulating data quality issues
func generateDataQualityEvents(twin *DigitalTwin.DigitalTwin) []DigitalTwin.SimulationEvent {
	events := []DigitalTwin.SimulationEvent{}

	if len(twin.Entities) == 0 {
		return events
	}

	// Select entities to target (up to 30% of entities)
	targetCount := len(twin.Entities) / 3
	if targetCount < 1 {
		targetCount = 1
	}
	if targetCount > len(twin.Entities) {
		targetCount = len(twin.Entities)
	}

	// Event 1: Resource unavailable at step 5 (data source failure)
	if len(twin.Entities) > 0 {
		targetEntity := twin.Entities[0]
		event := DigitalTwin.CreateEvent(
			DigitalTwin.EventResourceUnavailable,
			targetEntity.URI,
			5,
			map[string]interface{}{
				"reason": "Data source unavailable",
			},
		)
		event.WithSeverity(DigitalTwin.SeverityHigh)

		// Add propagation rules if there are relationships
		if len(twin.Relationships) > 0 {
			// Find relationship types connected to this entity
			relTypes := getEntityRelationshipTypes(twin, targetEntity.URI)
			for _, relType := range relTypes {
				event.WithPropagation(relType, 0.6, 2) // 60% impact, 2 step delay
			}
		}

		events = append(events, *event)
	}

	// Event 2: Process failure at step 15 (data validation failure)
	if len(twin.Entities) > 1 {
		targetEntity := twin.Entities[1]
		event := DigitalTwin.CreateEvent(
			DigitalTwin.EventProcessFailure,
			targetEntity.URI,
			15,
			map[string]interface{}{
				"reason": "Data validation failure - invalid schema",
			},
		)
		event.WithSeverity(DigitalTwin.SeverityMedium)

		events = append(events, *event)
	}

	// Event 3: Resource restoration at step 25
	if len(twin.Entities) > 0 {
		targetEntity := twin.Entities[0]
		event := DigitalTwin.CreateEvent(
			DigitalTwin.EventResourceAvailable,
			targetEntity.URI,
			25,
			map[string]interface{}{
				"reason": "Data source restored",
			},
		)
		event.WithSeverity(DigitalTwin.SeverityLow)

		events = append(events, *event)
	}

	// Event 4: Data quality constraint at step 30 (if we have more entities)
	if len(twin.Entities) > 2 {
		targetEntity := twin.Entities[2]
		event := DigitalTwin.CreateEvent(
			DigitalTwin.EventPolicyConstraintAdd,
			targetEntity.URI,
			30,
			map[string]interface{}{
				"constraint":      "data_quality_check",
				"capacity_impact": 0.85, // 15% capacity reduction
			},
		)
		event.WithSeverity(DigitalTwin.SeverityMedium)

		events = append(events, *event)
	}

	return events
}

// generateCapacityTestEvents creates events simulating high load scenarios
func generateCapacityTestEvents(twin *DigitalTwin.DigitalTwin) []DigitalTwin.SimulationEvent {
	events := []DigitalTwin.SimulationEvent{}

	if len(twin.Entities) == 0 {
		return events
	}

	// Event 1: Initial demand surge at step 5
	if len(twin.Entities) > 0 {
		targetEntity := twin.Entities[0]
		event := DigitalTwin.CreateEvent(
			DigitalTwin.EventDemandSurge,
			targetEntity.URI,
			5,
			map[string]interface{}{
				"increase_factor": 1.8, // 80% increase
				"reason":          "Peak load scenario",
			},
		)
		event.WithSeverity(DigitalTwin.SeverityMedium)

		// Propagate to related entities
		relTypes := getEntityRelationshipTypes(twin, targetEntity.URI)
		for _, relType := range relTypes {
			event.WithPropagation(relType, 0.7, 1) // 70% impact, 1 step delay
		}

		events = append(events, *event)
	}

	// Event 2: Secondary demand surge at step 15 (cascading load)
	if len(twin.Entities) > 1 {
		targetEntity := twin.Entities[len(twin.Entities)/2]
		event := DigitalTwin.CreateEvent(
			DigitalTwin.EventDemandSurge,
			targetEntity.URI,
			15,
			map[string]interface{}{
				"increase_factor": 2.2, // 120% increase
				"reason":          "Compound load increase",
			},
		)
		event.WithSeverity(DigitalTwin.SeverityHigh)

		relTypes := getEntityRelationshipTypes(twin, targetEntity.URI)
		for _, relType := range relTypes {
			event.WithPropagation(relType, 0.8, 1)
		}

		events = append(events, *event)
	}

	// Event 3: Capacity reduction at step 20 (resource constraint under load)
	if len(twin.Entities) > 2 {
		targetEntity := twin.Entities[len(twin.Entities)-1]
		event := DigitalTwin.CreateEvent(
			DigitalTwin.EventResourceCapacityChange,
			targetEntity.URI,
			20,
			map[string]interface{}{
				"multiplier": 0.7, // 30% capacity reduction
				"reason":     "Resource throttling under high load",
			},
		)
		event.WithSeverity(DigitalTwin.SeverityHigh)

		events = append(events, *event)
	}

	// Event 4: External market shift at step 25 (additional load from external factor)
	if len(twin.Entities) > 0 {
		targetEntity := twin.Entities[0]
		event := DigitalTwin.CreateEvent(
			DigitalTwin.EventExternalMarketShift,
			targetEntity.URI,
			25,
			map[string]interface{}{
				"demand_impact": 1.5,
				"reason":        "Market surge affecting demand patterns",
			},
		)
		event.WithSeverity(DigitalTwin.SeverityCritical)

		relTypes := getEntityRelationshipTypes(twin, targetEntity.URI)
		for _, relType := range relTypes {
			event.WithPropagation(relType, 0.5, 2)
		}

		events = append(events, *event)
	}

	// Event 5: Process optimization at step 35 (mitigation)
	if len(twin.Entities) > 1 {
		targetEntity := twin.Entities[1]
		event := DigitalTwin.CreateEvent(
			DigitalTwin.EventProcessOptimization,
			targetEntity.URI,
			35,
			map[string]interface{}{
				"efficiency_gain": 0.25, // 25% efficiency improvement
				"reason":          "Load balancing optimization applied",
			},
		)
		event.WithSeverity(DigitalTwin.SeverityLow)

		events = append(events, *event)
	}

	// Event 6: Demand normalization at step 45
	if len(twin.Entities) > 0 {
		targetEntity := twin.Entities[0]
		event := DigitalTwin.CreateEvent(
			DigitalTwin.EventDemandDrop,
			targetEntity.URI,
			45,
			map[string]interface{}{
				"decrease_factor": 0.6, // Return to 60% of peak
				"reason":          "Load returning to normal levels",
			},
		)
		event.WithSeverity(DigitalTwin.SeverityLow)

		events = append(events, *event)
	}

	return events
}

// getEntityRelationshipTypes returns unique relationship types for an entity
func getEntityRelationshipTypes(twin *DigitalTwin.DigitalTwin, entityURI string) []string {
	typeMap := make(map[string]bool)
	types := []string{}

	for _, rel := range twin.Relationships {
		if rel.SourceURI == entityURI || rel.TargetURI == entityURI {
			if !typeMap[rel.Type] {
				typeMap[rel.Type] = true
				types = append(types, rel.Type)
			}
		}
	}

	return types
}

// saveScenarioToDatabase persists a simulation scenario to the database
func saveScenarioToDatabase(ctx context.Context, persistence interface{}, scenario *DigitalTwin.SimulationScenario) error {
	// Get database connection from persistence backend
	type dbProvider interface {
		GetDB() *sql.DB
	}

	provider, ok := persistence.(dbProvider)
	if !ok {
		return fmt.Errorf("persistence backend does not provide database access")
	}

	db := provider.GetDB()
	if db == nil {
		return fmt.Errorf("database connection is nil")
	}

	// Serialize events to JSON
	eventsJSON, err := json.Marshal(scenario.Events)
	if err != nil {
		return fmt.Errorf("failed to marshal events: %w", err)
	}

	// Insert scenario into database
	query := `
		INSERT INTO simulation_scenarios (id, twin_id, name, description, scenario_type, events, duration, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = db.ExecContext(ctx, query,
		scenario.ID,
		scenario.TwinID,
		scenario.Name,
		scenario.Description,
		scenario.Type,
		string(eventsJSON),
		scenario.Duration,
		scenario.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to insert scenario: %w", err)
	}

	return nil
}

// generateSampleValue creates sample data for a given column
func (s *Server) generateSampleValue(columnName string, rowIndex int) interface{} {
	name := strings.ToLower(columnName)

	// Generate different types of sample data based on column name patterns
	if strings.Contains(name, "id") || strings.HasSuffix(name, "_id") {
		return rowIndex
	}

	if strings.Contains(name, "name") {
		names := []string{"John Doe", "Jane Smith", "Bob Johnson", "Alice Brown", "Charlie Wilson"}
		return names[(rowIndex-1)%len(names)]
	}

	if strings.Contains(name, "email") {
		emails := []string{"john@example.com", "jane@example.com", "bob@example.com", "alice@example.com", "charlie@example.com"}
		return emails[(rowIndex-1)%len(emails)]
	}

	if strings.Contains(name, "age") {
		return 25 + (rowIndex * 5)
	}

	if strings.Contains(name, "price") || strings.Contains(name, "cost") || strings.Contains(name, "amount") {
		return 10.99 + float64(rowIndex*10)
	}

	if strings.Contains(name, "active") || strings.Contains(name, "enabled") {
		return rowIndex%2 == 1
	}

	if strings.Contains(name, "date") || strings.Contains(name, "created") || strings.Contains(name, "updated") {
		return fmt.Sprintf("2024-01-%02d", rowIndex)
	}

	// Default to string
	return fmt.Sprintf("Value %d", rowIndex)
}

// DataImportRequest represents a request to import CSV data into TDB2
type DataImportRequest struct {
	UploadID   string `json:"upload_id"`
	OntologyID string `json:"ontology_id"`
}

// handleDataImport imports CSV data into the TDB2 knowledge graph as RDF triples
func (s *Server) handleDataImport(w http.ResponseWriter, r *http.Request) {
	var req DataImportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequestResponse(w, fmt.Sprintf("Invalid request body: %v", err))
		return
	}

	if req.UploadID == "" || req.OntologyID == "" {
		writeBadRequestResponse(w, "upload_id and ontology_id are required")
		return
	}

	// Check if TDB2 backend is available
	if s.tdb2Backend == nil {
		writeErrorResponse(w, http.StatusServiceUnavailable, "TDB2 backend is not available")
		return
	}

	// Check if persistence backend is available
	if s.persistence == nil {
		writeErrorResponse(w, http.StatusServiceUnavailable, "Persistence backend is not available")
		return
	}

	// Get ontology metadata
	ontology, err := s.persistence.GetOntology(r.Context(), req.OntologyID)
	if err != nil {
		writeErrorResponse(w, http.StatusNotFound, fmt.Sprintf("Ontology not found: %v", err))
		return
	}

	// Get the uploaded file path
	uploadPath := fmt.Sprintf("/tmp/mimir-uploads/%s", req.UploadID)
	if _, err := os.Stat(uploadPath); os.IsNotExist(err) {
		writeErrorResponse(w, http.StatusNotFound, fmt.Sprintf("Upload file not found: %s", req.UploadID))
		return
	}

	// Read and parse the CSV file
	csvData, err := s.readAndParseCSV(uploadPath)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to parse CSV data: %v", err))
		return
	}

	// Convert CSV rows to RDF triples
	triples, stats, err := s.csvToTriples(csvData, req.OntologyID, ontology.TDB2Graph)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to convert CSV to triples: %v", err))
		return
	}

	// Insert triples into TDB2 in batches
	batchSize := 1000
	totalInserted := 0
	for i := 0; i < len(triples); i += batchSize {
		end := i + batchSize
		if end > len(triples) {
			end = len(triples)
		}
		batch := triples[i:end]

		err := s.tdb2Backend.InsertTriples(r.Context(), batch)
		if err != nil {
			writeErrorResponse(w, http.StatusInternalServerError,
				fmt.Sprintf("Failed to insert triples (batch %d-%d): %v", i, end, err))
			return
		}
		totalInserted += len(batch)
	}

	// Return success response with statistics
	response := map[string]any{
		"message":          "Data imported successfully",
		"upload_id":        req.UploadID,
		"ontology_id":      req.OntologyID,
		"graph_uri":        ontology.TDB2Graph,
		"entities_created": stats["entities_created"],
		"triples_created":  totalInserted,
		"rows_processed":   stats["rows_processed"],
	}

	writeJSONResponse(w, http.StatusOK, response)
}

// readAndParseCSV reads and parses a CSV file
func (s *Server) readAndParseCSV(filePath string) ([]map[string]interface{}, error) {
	// Use the CSV plugin to parse the file
	plugin, err := s.registry.GetPlugin("Input", "csv")
	if err != nil {
		return nil, fmt.Errorf("CSV plugin not found: %w", err)
	}

	// Create a context for plugin execution
	globalContext := pipelines.NewPluginContext()
	stepConfig := pipelines.StepConfig{
		Name:   "parse_csv",
		Plugin: "Input.csv",
		Config: map[string]any{
			"file_path":   filePath,
			"has_headers": true,
		},
		Output: "csv_data",
	}

	// Execute the plugin
	result, err := plugin.ExecuteStep(context.Background(), stepConfig, globalContext)
	if err != nil {
		return nil, fmt.Errorf("failed to parse CSV: %w", err)
	}

	// Get the parsed data
	parsedData, ok := result.Get("csv_data")
	if !ok {
		return nil, fmt.Errorf("failed to get parsed CSV data from plugin")
	}

	// Convert to []map[string]interface{}
	dataMap, ok := parsedData.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected data format from CSV plugin")
	}

	rows, ok := dataMap["rows"].([]any)
	if !ok {
		return nil, fmt.Errorf("no rows found in CSV data")
	}

	// Convert to proper format
	var csvRows []map[string]interface{}
	for _, row := range rows {
		if rowMap, ok := row.(map[string]interface{}); ok {
			csvRows = append(csvRows, rowMap)
		}
	}

	return csvRows, nil
}

// csvToTriples converts CSV rows to RDF triples based on ontology structure
func (s *Server) csvToTriples(csvData []map[string]interface{}, ontologyID, graphURI string) ([]knowledgegraph.Triple, map[string]int, error) {
	var triples []knowledgegraph.Triple
	stats := map[string]int{
		"entities_created": 0,
		"rows_processed":   0,
	}

	// Base URI for entity generation
	baseURI := "http://mimir-aip.io/data"

	// Determine entity type from first row column names
	// In a production system, this would use the ontology schema
	entityType := "Entity"
	if len(csvData) > 0 {
		// Use the ontology ID as the entity type
		entityType = strings.ReplaceAll(ontologyID, "-", "_")
	}

	// Process each row
	for rowIndex, row := range csvData {
		// Generate entity URI
		entityURI := fmt.Sprintf("%s/%s/%s/%d", baseURI, ontologyID, entityType, rowIndex)

		// Create rdf:type triple
		triples = append(triples, knowledgegraph.Triple{
			Subject:   entityURI,
			Predicate: "http://www.w3.org/1999/02/22-rdf-syntax-ns#type",
			Object:    fmt.Sprintf("http://mimir-aip.io/ontology/%s#%s", ontologyID, entityType),
			Graph:     graphURI,
		})

		// Create datatype property triples for each column
		for columnName, value := range row {
			if value == nil {
				continue
			}

			// Create property URI from column name
			propertyName := strings.ReplaceAll(columnName, " ", "_")
			propertyName = strings.ReplaceAll(propertyName, "-", "_")
			propertyURI := fmt.Sprintf("http://mimir-aip.io/ontology/%s#%s", ontologyID, propertyName)

			// Convert value to string
			valueStr := fmt.Sprintf("%v", value)

			// Create the triple
			triples = append(triples, knowledgegraph.Triple{
				Subject:   entityURI,
				Predicate: propertyURI,
				Object:    valueStr,
				Graph:     graphURI,
			})
		}

		stats["entities_created"]++
		stats["rows_processed"]++
	}

	return triples, stats, nil
}

// profileColumnData calculates comprehensive statistics for a column
func profileColumnData(columnName string, values []any) ColumnProfile {
	profile := ColumnProfile{
		ColumnName:    columnName,
		QualityIssues: []string{},
		TopValues:     []ValueFrequency{},
	}

	totalCount := len(values)
	profile.TotalCount = totalCount

	if totalCount == 0 {
		profile.DataType = "unknown"
		profile.DataQualityScore = 0.0
		profile.QualityIssues = append(profile.QualityIssues, "No data available")
		return profile
	}

	// Track distinct values and nulls
	distinctValues := make(map[string]bool)
	valueCounts := make(map[string]int)
	nullCount := 0

	// For numeric analysis
	var numericValues []float64
	isNumeric := true

	// For string length analysis
	var stringLengths []int

	// Analyze each value
	for _, val := range values {
		// Handle null/empty values
		if val == nil || val == "" {
			nullCount++
			continue
		}

		valueStr := fmt.Sprintf("%v", val)
		distinctValues[valueStr] = true
		valueCounts[valueStr]++

		// Track string lengths
		stringLengths = append(stringLengths, len(valueStr))

		// Try to parse as numeric
		if numVal, err := parseNumeric(val); err == nil {
			numericValues = append(numericValues, numVal)
		} else {
			isNumeric = false
		}
	}

	// Calculate basic stats
	profile.DistinctCount = len(distinctValues)
	profile.NullCount = nullCount

	nonNullCount := totalCount - nullCount
	if totalCount > 0 {
		profile.DistinctPercent = float64(profile.DistinctCount) / math.Max(1, float64(nonNullCount)) * 100
		profile.NullPercent = float64(nullCount) / float64(totalCount) * 100
	}

	// Determine data type
	if isNumeric && len(numericValues) > 0 {
		profile.DataType = "numeric"

		// Calculate numeric statistics
		profile.Mean = calculateMean(numericValues)
		profile.Median = calculateMedian(numericValues)
		profile.StdDev = calculateStdDev(numericValues, profile.Mean)
		profile.MinValue = findMin(numericValues)
		profile.MaxValue = findMax(numericValues)
	} else {
		profile.DataType = "string"

		// Calculate string length statistics
		if len(stringLengths) > 0 {
			profile.MinLength = minInt(stringLengths)
			profile.MaxLength = maxInt(stringLengths)
			profile.AvgLength = avgInt(stringLengths)
		}
	}

	// Calculate top 5 most frequent values
	profile.TopValues = getTopValues(valueCounts, nonNullCount, 5)

	// Detect data quality issues
	profile.QualityIssues = detectQualityIssues(profile, nonNullCount)

	// Calculate data quality score (0-1.0)
	profile.DataQualityScore = calculateDataQualityScore(profile, nonNullCount)

	return profile
}

// parseNumeric attempts to parse a value as numeric
func parseNumeric(val any) (float64, error) {
	switch v := val.(type) {
	case int:
		return float64(v), nil
	case int32:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case float32:
		return float64(v), nil
	case float64:
		return v, nil
	case string:
		return strconv.ParseFloat(v, 64)
	default:
		return 0, fmt.Errorf("not numeric")
	}
}

// calculateMean calculates the arithmetic mean
func calculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

// calculateMedian calculates the median value
func calculateMedian(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)

	n := len(sorted)
	if n%2 == 0 {
		return (sorted[n/2-1] + sorted[n/2]) / 2.0
	}
	return sorted[n/2]
}

// calculateStdDev calculates standard deviation
func calculateStdDev(values []float64, mean float64) float64 {
	if len(values) == 0 {
		return 0
	}

	sumSquares := 0.0
	for _, v := range values {
		diff := v - mean
		sumSquares += diff * diff
	}

	variance := sumSquares / float64(len(values))
	return math.Sqrt(variance)
}

// findMin finds the minimum numeric value
func findMin(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	min := values[0]
	for _, v := range values[1:] {
		if v < min {
			min = v
		}
	}
	return min
}

// findMax finds the maximum numeric value
func findMax(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	max := values[0]
	for _, v := range values[1:] {
		if v > max {
			max = v
		}
	}
	return max
}

// minInt finds minimum integer value
func minInt(values []int) int {
	if len(values) == 0 {
		return 0
	}
	min := values[0]
	for _, v := range values[1:] {
		if v < min {
			min = v
		}
	}
	return min
}

// maxInt finds maximum integer value
func maxInt(values []int) int {
	if len(values) == 0 {
		return 0
	}
	max := values[0]
	for _, v := range values[1:] {
		if v > max {
			max = v
		}
	}
	return max
}

// avgInt calculates average of integers
func avgInt(values []int) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0
	for _, v := range values {
		sum += v
	}
	return float64(sum) / float64(len(values))
}

// getTopValues returns top N most frequent values
func getTopValues(valueCounts map[string]int, totalCount int, topN int) []ValueFrequency {
	// Convert map to slice for sorting
	type pair struct {
		value string
		count int
	}

	var pairs []pair
	for value, count := range valueCounts {
		pairs = append(pairs, pair{value, count})
	}

	// Sort by count descending
	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].count > pairs[j].count
	})

	// Take top N
	n := topN
	if len(pairs) < n {
		n = len(pairs)
	}

	result := make([]ValueFrequency, n)
	for i := 0; i < n; i++ {
		freq := 0.0
		if totalCount > 0 {
			freq = float64(pairs[i].count) / float64(totalCount) * 100
		}
		result[i] = ValueFrequency{
			Value:     pairs[i].value,
			Count:     pairs[i].count,
			Frequency: freq,
		}
	}

	return result
}

// detectQualityIssues identifies potential data quality problems
func detectQualityIssues(profile ColumnProfile, nonNullCount int) []string {
	issues := []string{}

	// High null rate
	if profile.NullPercent > 50 {
		issues = append(issues, fmt.Sprintf("High null rate (%.1f%%)", profile.NullPercent))
	} else if profile.NullPercent > 25 {
		issues = append(issues, fmt.Sprintf("Moderate null rate (%.1f%%)", profile.NullPercent))
	}

	// Low cardinality (may indicate categorical data or quality issues)
	if nonNullCount > 10 && profile.DistinctCount < 5 {
		issues = append(issues, fmt.Sprintf("Very low cardinality (%d distinct values)", profile.DistinctCount))
	}

	// Check for potential duplicates in what should be unique
	if profile.DistinctPercent < 80 && nonNullCount > 100 {
		issues = append(issues, fmt.Sprintf("Low uniqueness (%.1f%% distinct)", profile.DistinctPercent))
	}

	// Check for single dominant value
	if len(profile.TopValues) > 0 && profile.TopValues[0].Frequency > 80 {
		issues = append(issues, fmt.Sprintf("Single value dominates (%.1f%%)", profile.TopValues[0].Frequency))
	}

	// Check for extreme string length variance
	if profile.DataType == "string" && profile.MaxLength > 0 {
		lengthRatio := float64(profile.MaxLength) / math.Max(1, float64(profile.MinLength))
		if lengthRatio > 100 {
			issues = append(issues, "Extreme length variance in string values")
		}
	}

	// Check for potential ID columns with gaps
	columnNameLower := strings.ToLower(profile.ColumnName)
	if strings.Contains(columnNameLower, "id") && profile.DistinctPercent < 95 && nonNullCount > 20 {
		issues = append(issues, "Potential ID column with missing or duplicate values")
	}

	return issues
}

// calculateDataQualityScore computes an overall quality score (0-1.0)
func calculateDataQualityScore(profile ColumnProfile, nonNullCount int) float64 {
	score := 1.0

	// Penalize high null rate
	score -= profile.NullPercent / 200.0 // Max penalty: 0.5 for 100% nulls

	// Penalize if there are critical quality issues
	criticalIssues := 0
	for _, issue := range profile.QualityIssues {
		if strings.Contains(issue, "High null") ||
			strings.Contains(issue, "Very low cardinality") ||
			strings.Contains(issue, "duplicate values") {
			criticalIssues++
		}
	}
	score -= float64(criticalIssues) * 0.15

	// Bonus for high data completeness
	if profile.NullPercent < 5 {
		score += 0.1
	}

	// Bonus for good cardinality (for non-ID columns)
	columnNameLower := strings.ToLower(profile.ColumnName)
	if !strings.Contains(columnNameLower, "id") {
		if nonNullCount > 10 && profile.DistinctCount >= 5 && profile.DistinctCount < nonNullCount {
			score += 0.05
		}
	}

	// Ensure score is in valid range
	if score < 0 {
		score = 0
	}
	if score > 1.0 {
		score = 1.0
	}

	return math.Round(score*100) / 100 // Round to 2 decimal places
}

// profileDataset generates comprehensive profiling for entire dataset
func profileDataset(data map[string]any, sampleSize int) DataProfileSummary {
	summary := DataProfileSummary{
		SuggestedPrimaryKeys: []string{},
		ColumnProfiles:       []ColumnProfile{},
	}

	// Extract rows from data
	rowsAny, ok := data["rows"].([]any)
	if !ok || len(rowsAny) == 0 {
		return summary
	}

	summary.TotalRows = len(rowsAny)

	// Sample data if needed (for large datasets)
	sampled := rowsAny
	if sampleSize > 0 && len(rowsAny) > sampleSize {
		sampled = sampleRows(rowsAny, sampleSize)
	}

	// Get column names from first row
	if len(sampled) == 0 {
		return summary
	}

	firstRow, ok := sampled[0].(map[string]any)
	if !ok {
		return summary
	}

	// Extract column values
	columnValues := make(map[string][]any)
	for colName := range firstRow {
		columnValues[colName] = []any{}
	}

	for _, rowAny := range sampled {
		row, ok := rowAny.(map[string]any)
		if !ok {
			continue
		}
		for colName, val := range row {
			columnValues[colName] = append(columnValues[colName], val)
		}
	}

	summary.TotalColumns = len(columnValues)

	// Profile each column
	totalDistinct := 0
	totalQualityScore := 0.0

	for colName, values := range columnValues {
		profile := profileColumnData(colName, values)
		summary.ColumnProfiles = append(summary.ColumnProfiles, profile)
		totalDistinct += profile.DistinctCount
		totalQualityScore += profile.DataQualityScore
	}

	summary.TotalDistinctValues = totalDistinct

	// Calculate overall quality score
	if len(summary.ColumnProfiles) > 0 {
		summary.OverallQualityScore = math.Round((totalQualityScore/float64(len(summary.ColumnProfiles)))*100) / 100
	}

	// Suggest primary key columns (high uniqueness, low nulls)
	for _, profile := range summary.ColumnProfiles {
		if profile.DistinctPercent > 95 && profile.NullPercent < 5 && len(sampled) > 10 {
			summary.SuggestedPrimaryKeys = append(summary.SuggestedPrimaryKeys, profile.ColumnName)
		}
	}

	return summary
}

// sampleRows returns a random sample of rows
func sampleRows(rows []any, sampleSize int) []any {
	if sampleSize >= len(rows) {
		return rows
	}

	// Use systematic sampling for consistency
	step := len(rows) / sampleSize
	if step < 1 {
		step = 1
	}

	sampled := make([]any, 0, sampleSize)
	for i := 0; i < len(rows) && len(sampled) < sampleSize; i += step {
		sampled = append(sampled, rows[i])
	}

	return sampled
}

// POST /api/v1/data/:id/infer-schema - Infer schema from imported data
func (s *Server) handleInferSchemaFromImport(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	importID := vars["id"]

	if importID == "" {
		writeBadRequestResponse(w, "import ID is required")
		return
	}

	var req struct {
		EnableAIFallback  bool   `json:"enable_ai_fallback"`
		EnableFKDetection bool   `json:"enable_fk_detection"`
		PluginType        string `json:"plugin_type"` // e.g. "Input"
		PluginName        string `json:"plugin_name"` // e.g. "csv"
	}
	json.NewDecoder(r.Body).Decode(&req)

	// Default to CSV plugin if not specified
	if req.PluginType == "" {
		req.PluginType = "Input"
	}
	if req.PluginName == "" {
		req.PluginName = "csv"
	}

	// 1. Get the plugin to parse the uploaded file
	plugin, err := s.registry.GetPlugin(req.PluginType, req.PluginName)
	if err != nil {
		writeBadRequestResponse(w, fmt.Sprintf("Plugin not found: %v", err))
		return
	}

	// 2. Parse the uploaded file
	filePath := fmt.Sprintf("/tmp/mimir-uploads/%s", importID)
	stepConfig := pipelines.StepConfig{
		Name:   "schema_inference_data_load",
		Plugin: fmt.Sprintf("%s.%s", req.PluginType, req.PluginName),
		Config: map[string]interface{}{
			"file_path": filePath,
		},
		Output: "parsed_data",
	}

	globalContext := pipelines.NewPluginContext()

	result, err := plugin.ExecuteStep(r.Context(), stepConfig, globalContext)
	if err != nil {
		writeInternalServerErrorResponse(w, fmt.Sprintf("Failed to parse data: %v", err))
		return
	}

	parsedData, ok := result.Get("parsed_data")
	if !ok {
		writeInternalServerErrorResponse(w, "Failed to get parsed data from plugin")
		return
	}

	// 3. Convert parsed data to []map[string]interface{} for schema inference
	var dataRows []map[string]interface{}
	switch data := parsedData.(type) {
	case []map[string]interface{}:
		dataRows = data
	case map[string]interface{}:
		// Handle case where plugin returns wrapper
		if rows, ok := data["rows"].([]map[string]interface{}); ok {
			dataRows = rows
		} else if dataArray, ok := data["data"].([]map[string]interface{}); ok {
			dataRows = dataArray
		} else {
			writeInternalServerErrorResponse(w, "Unexpected data format from plugin")
			return
		}
	default:
		writeInternalServerErrorResponse(w, fmt.Sprintf("Unsupported data type: %T", parsedData))
		return
	}

	if len(dataRows) == 0 {
		writeBadRequestResponse(w, "No data rows found in uploaded file")
		return
	}

	// 4. Create schema inference engine with proper config
	config := schema_inference.InferenceConfig{
		SampleSize:          100,
		ConfidenceThreshold: 0.8,
		EnableRelationships: true,
		EnableConstraints:   true,
		EnableAIFallback:    req.EnableAIFallback,
		AIConfidenceBoost:   0.15,
		EnableFKDetection:   req.EnableFKDetection,
		FKMinConfidence:     0.8,
	}

	engine := schema_inference.NewSchemaInferenceEngine(config)

	// 5. Infer schema from data
	datasetName := fmt.Sprintf("Dataset_%s", strings.TrimPrefix(importID, "upload_"))
	schema, err := engine.InferSchema(dataRows, datasetName)
	if err != nil {
		writeInternalServerErrorResponse(w, fmt.Sprintf("Schema inference failed: %v", err))
		return
	}

	// 6. Calculate overall confidence (average of column confidences)
	var totalConfidence float64
	var confidenceCount int
	for _, column := range schema.Columns {
		totalConfidence += column.AIConfidence
		confidenceCount++
	}
	avgConfidence := 0.85 // Default confidence
	if confidenceCount > 0 {
		avgConfidence = totalConfidence / float64(confidenceCount)
	}

	// Generate schema ID
	schemaID := uuid.New().String()

	// Return success response
	writeSuccessResponse(w, map[string]interface{}{
		"schema_id":    schemaID,
		"schema":       schema,
		"column_count": len(schema.Columns),
		"fk_count":     len(schema.ForeignKeys),
		"confidence":   avgConfidence,
		"ai_enhanced":  req.EnableAIFallback,
		"next_action":  "generate_ontology",
		"message":      "Schema inferred successfully",
	})
}

// handleGenerateOntologyFromSchema generates an OWL ontology from an inferred schema
func (s *Server) handleGenerateOntologyFromSchema(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	schemaID := vars["id"]

	if schemaID == "" {
		writeBadRequestResponse(w, "schema ID is required")
		return
	}

	// Parse request body for configuration options
	type OntologyGenerationRequest struct {
		BaseURI        string `json:"base_uri,omitempty"`
		OntologyPrefix string `json:"ontology_prefix,omitempty"`
		Format         string `json:"format,omitempty"` // turtle, rdfxml, etc.
	}

	var req OntologyGenerationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Use defaults if body is empty or invalid
		req.Format = "turtle"
	}

	// Set defaults
	if req.Format == "" {
		req.Format = "turtle"
	}
	if req.BaseURI == "" {
		req.BaseURI = fmt.Sprintf("http://mimir-aip.io/ontology/%s", schemaID)
	}
	if req.OntologyPrefix == "" {
		req.OntologyPrefix = "mimir"
	}

	// Load inferred schema from database
	var schemaJSON string
	var schemaName string
	query := `SELECT schema_json, name FROM inferred_schemas WHERE id = ?`
	err := s.persistence.GetDB().QueryRow(query, schemaID).Scan(&schemaJSON, &schemaName)
	if err == sql.ErrNoRows {
		writeNotFoundResponse(w, "Schema not found")
		return
	}
	if err != nil {
		writeInternalServerErrorResponse(w, fmt.Sprintf("Failed to load schema: %v", err))
		return
	}

	// Parse schema JSON into DataSchema object
	var schema schema_inference.DataSchema
	if err := json.Unmarshal([]byte(schemaJSON), &schema); err != nil {
		writeInternalServerErrorResponse(w, fmt.Sprintf("Failed to parse schema JSON: %v", err))
		return
	}

	// Create ontology generator configuration
	config := schema_inference.OntologyConfig{
		BaseURI:         req.BaseURI,
		OntologyPrefix:  req.OntologyPrefix,
		ClassNaming:     "pascal", // PascalCase for classes
		PropertyNaming:  "camel",  // camelCase for properties
		IncludeMetadata: true,
		IncludeComments: true,
	}

	// Generate ontology
	generator := schema_inference.NewOntologyGenerator(config)
	ontology, err := generator.GenerateOntology(&schema)
	if err != nil {
		writeInternalServerErrorResponse(w, fmt.Sprintf("Failed to generate ontology: %v", err))
		return
	}

	// Save ontology file to disk
	ontologyDir := s.ontologyDir
	if ontologyDir == "" {
		ontologyDir = "./data/ontologies"
	}
	if err := os.MkdirAll(ontologyDir, 0755); err != nil {
		writeInternalServerErrorResponse(w, fmt.Sprintf("Failed to create ontology directory: %v", err))
		return
	}

	ontologyPath := fmt.Sprintf("%s/%s.ttl", ontologyDir, ontology.ID)
	if err := os.WriteFile(ontologyPath, []byte(ontology.Content), 0644); err != nil {
		writeInternalServerErrorResponse(w, fmt.Sprintf("Failed to save ontology file: %v", err))
		return
	}

	// Store in database (ontologies table)
	ontologyID := uuid.New().String()
	insertQuery := `
		INSERT INTO ontologies (id, name, description, version, format, file_path, tdb2_graph, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err = s.persistence.GetDB().Exec(insertQuery,
		ontologyID,
		ontology.Name,
		ontology.Description,
		ontology.Version,
		req.Format,
		ontologyPath,
		req.BaseURI,
		"active",
		time.Now(),
	)
	if err != nil {
		writeInternalServerErrorResponse(w, fmt.Sprintf("Failed to store ontology metadata: %v", err))
		return
	}

	// Upload to TDB2 if available
	tdb2Loaded := false
	if s.tdb2Backend != nil {
		if err := s.tdb2Backend.LoadOntology(r.Context(), req.BaseURI, ontology.Content, req.Format); err != nil {
			log.Printf("Warning: Failed to load ontology into TDB2: %v (ontology_id: %s)", err, ontologyID)
			// Don't fail the request if TDB2 upload fails
		} else {
			tdb2Loaded = true
		}
	}

	// Return response
	writeSuccessResponse(w, map[string]interface{}{
		"ontology_id":    ontologyID,
		"name":           ontology.Name,
		"description":    ontology.Description,
		"version":        ontology.Version,
		"class_count":    len(ontology.Classes),
		"property_count": len(ontology.Properties),
		"file_path":      ontologyPath,
		"graph_uri":      req.BaseURI,
		"tdb2_loaded":    tdb2Loaded,
		"next_action":    "entity_extraction",
		"message":        "Ontology generated successfully",
	})
}
