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
if s.taskQueue == nil {
writeInternalServerErrorResponse(w, "Task queue not available")
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

// Create task request
taskReq := &utils.TaskRequest{
Type:         "pipeline",
PipelineFile: pipelineFile,
Context:      req.Context,
}

// Enqueue task
taskInfo, err := s.taskQueue.EnqueueTask(r.Context(), taskReq)
if err != nil {
writeInternalServerErrorResponse(w, fmt.Sprintf("Failed to enqueue task: %v", err))
return
}

writeJSONResponse(w, http.StatusAccepted, map[string]any{
"message": "Pipeline task enqueued successfully",
"task_id": taskInfo.ID,
"status":  taskInfo.Status,
})
}

// handleEnqueueDigitalTwin handles requests to enqueue a digital twin task
func (s *Server) handleEnqueueDigitalTwin(w http.ResponseWriter, r *http.Request) {
if s.taskQueue == nil {
writeInternalServerErrorResponse(w, "Task queue not available")
return
}

var req map[string]any
if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
writeBadRequestResponse(w, fmt.Sprintf("Invalid request body: %v", err))
return
}

// Create task request
taskReq := &utils.TaskRequest{
Type:    "digital_twin",
Context: req,
}

// Enqueue task
taskInfo, err := s.taskQueue.EnqueueTask(r.Context(), taskReq)
if err != nil {
writeInternalServerErrorResponse(w, fmt.Sprintf("Failed to enqueue task: %v", err))
return
}

writeJSONResponse(w, http.StatusAccepted, map[string]any{
"message": "Digital twin task enqueued successfully",
"task_id": taskInfo.ID,
"status":  taskInfo.Status,
})
}

// handleGetTaskStatus handles requests to get task status
func (s *Server) handleGetTaskStatus(w http.ResponseWriter, r *http.Request) {
if s.taskQueue == nil {
writeInternalServerErrorResponse(w, "Task queue not available")
return
}

vars := mux.Vars(r)
taskID := vars["id"]

if taskID == "" {
writeBadRequestResponse(w, "Task ID is required")
return
}

// Try to get result
result, err := s.taskQueue.GetTaskResult(r.Context(), taskID)
if err != nil {
// Task not completed yet
writeJSONResponse(w, http.StatusOK, map[string]any{
"task_id": taskID,
"status":  "processing",
})
return
}

// Task completed
writeJSONResponse(w, http.StatusOK, result)
}

// handleWaitForTask handles requests to wait for task completion
func (s *Server) handleWaitForTask(w http.ResponseWriter, r *http.Request) {
if s.taskQueue == nil {
writeInternalServerErrorResponse(w, "Task queue not available")
return
}

vars := mux.Vars(r)
taskID := vars["id"]

if taskID == "" {
writeBadRequestResponse(w, "Task ID is required")
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

// Wait for task result
result, err := s.taskQueue.WaitForTaskResult(r.Context(), taskID, timeout)
if err != nil {
writeInternalServerErrorResponse(w, fmt.Sprintf("Failed to get task result: %v", err))
return
}

writeJSONResponse(w, http.StatusOK, result)
}

// handleGetQueueStatus handles requests to get queue status
func (s *Server) handleGetQueueStatus(w http.ResponseWriter, r *http.Request) {
if s.taskQueue == nil {
writeInternalServerErrorResponse(w, "Task queue not available")
return
}

queueLength, err := s.taskQueue.GetQueueLength(r.Context())
if err != nil {
writeInternalServerErrorResponse(w, fmt.Sprintf("Failed to get queue length: %v", err))
return
}

writeJSONResponse(w, http.StatusOK, map[string]any{
"queue_length": queueLength,
})
}
