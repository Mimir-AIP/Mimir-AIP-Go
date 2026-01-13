package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Mimir-AIP/Mimir-AIP-Go/utils"
	"github.com/gorilla/mux"
)

// handleEnqueuePipeline handles requests to enqueue a pipeline for async execution
func (s *Server) handleEnqueuePipeline(w http.ResponseWriter, r *http.Request) {
	if s.jobQueue == nil {
		writeInternalServerErrorResponse(w, "Job queue not available")
		return
	}

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

	// Create job request
	jobReq := &utils.JobRequest{
		Type:         "pipeline",
		PipelineFile: pipelineFile,
		Context:      req.Context,
	}

	// Enqueue job
	jobInfo, err := s.jobQueue.EnqueueJob(r.Context(), jobReq)
	if err != nil {
		writeInternalServerErrorResponse(w, fmt.Sprintf("Failed to enqueue job: %v", err))
		return
	}

	writeJSONResponse(w, http.StatusAccepted, map[string]any{
		"message": "Pipeline job enqueued successfully",
		"job_id":  jobInfo.ID,
		"status":  jobInfo.Status,
	})
}

// handleEnqueueDigitalTwin handles requests to enqueue a digital twin job
func (s *Server) handleEnqueueDigitalTwin(w http.ResponseWriter, r *http.Request) {
	if s.jobQueue == nil {
		writeInternalServerErrorResponse(w, "Job queue not available")
		return
	}

	var req map[string]any
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequestResponse(w, fmt.Sprintf("Invalid request body: %v", err))
		return
	}

	// Create job request
	jobReq := &utils.JobRequest{
		Type:    "digital_twin",
		Context: req,
	}

	// Enqueue job
	jobInfo, err := s.jobQueue.EnqueueJob(r.Context(), jobReq)
	if err != nil {
		writeInternalServerErrorResponse(w, fmt.Sprintf("Failed to enqueue job: %v", err))
		return
	}

	writeJSONResponse(w, http.StatusAccepted, map[string]any{
		"message": "Digital twin job enqueued successfully",
		"job_id":  jobInfo.ID,
		"status":  jobInfo.Status,
	})
}

// handleGetJobStatus handles requests to get job status
func (s *Server) handleGetJobStatus(w http.ResponseWriter, r *http.Request) {
	if s.jobQueue == nil {
		writeInternalServerErrorResponse(w, "Job queue not available")
		return
	}

	vars := mux.Vars(r)
	jobID := vars["id"]

	if jobID == "" {
		writeBadRequestResponse(w, "Job ID is required")
		return
	}

	// Try to get result
	result, err := s.jobQueue.GetJobResult(r.Context(), jobID)
	if err != nil {
		// Job not completed yet
		writeJSONResponse(w, http.StatusOK, map[string]any{
			"job_id": jobID,
			"status": "processing",
		})
		return
	}

	// Job completed
	writeJSONResponse(w, http.StatusOK, result)
}

// handleWaitForJob handles requests to wait for job completion
func (s *Server) handleWaitForJob(w http.ResponseWriter, r *http.Request) {
	if s.jobQueue == nil {
		writeInternalServerErrorResponse(w, "Job queue not available")
		return
	}

	vars := mux.Vars(r)
	jobID := vars["id"]

	if jobID == "" {
		writeBadRequestResponse(w, "Job ID is required")
		return
	}

	// Get timeout from query parameter (default: 60 seconds)
	timeoutStr := r.URL.Query().Get("timeout")
	timeout := 60 * time.Second
	if timeoutStr != "" {
		if t, err := time.ParseDuration(timeoutStr); err == nil {
			timeout = t
		}
	}

	// Wait for job result
	result, err := s.jobQueue.WaitForJobResult(r.Context(), jobID, timeout)
	if err != nil {
		writeInternalServerErrorResponse(w, fmt.Sprintf("Failed to get job result: %v", err))
		return
	}

	writeJSONResponse(w, http.StatusOK, result)
}

// handleGetQueueStatus handles requests to get queue status
func (s *Server) handleGetQueueStatus(w http.ResponseWriter, r *http.Request) {
	if s.jobQueue == nil {
		writeInternalServerErrorResponse(w, "Job queue not available")
		return
	}

	queueLength, err := s.jobQueue.GetQueueLength(r.Context())
	if err != nil {
		writeInternalServerErrorResponse(w, fmt.Sprintf("Failed to get queue length: %v", err))
		return
	}

	writeJSONResponse(w, http.StatusOK, map[string]any{
		"queue_length": queueLength,
	})
}
