package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"github.com/mimir-aip/mimir-aip-go/pkg/queue"
)

// Server provides HTTP API endpoints
type Server struct {
	queue *queue.Queue
	port  string
	mux   *http.ServeMux
}

// NewServer creates a new API server
func NewServer(q *queue.Queue, port string) *Server {
	s := &Server{
		queue: q,
		port:  port,
		mux:   http.NewServeMux(),
	}

	s.registerRoutes()
	return s
}

// registerRoutes sets up the HTTP routes
func (s *Server) registerRoutes() {
	s.mux.HandleFunc("/health", s.handleHealth)
	s.mux.HandleFunc("/ready", s.handleReady)
	s.mux.HandleFunc("/api/jobs", s.handleJobs)
	s.mux.HandleFunc("/api/jobs/", s.handleJobByID)
}

// Start starts the HTTP server
func (s *Server) Start() error {
	addr := fmt.Sprintf(":%s", s.port)
	log.Printf("Starting API server on %s", addr)
	return http.ListenAndServe(addr, s.mux)
}

// handleHealth handles health check requests
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

// handleReady handles readiness check requests
func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	// Check if queue is accessible
	_, err := s.queue.QueueLength()
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"status": "not ready", "error": err.Error()})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
}

// handleJobs handles job submission (POST) and listing (GET)
func (s *Server) handleJobs(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		s.handleJobSubmission(w, r)
	case http.MethodGet:
		s.handleJobList(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleJobSubmission handles job submission requests
func (s *Server) handleJobSubmission(w http.ResponseWriter, r *http.Request) {
	var req models.JobSubmissionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Validate request
	if req.Type == "" {
		http.Error(w, "Job type is required", http.StatusBadRequest)
		return
	}
	if req.ProjectID == "" {
		http.Error(w, "Project ID is required", http.StatusBadRequest)
		return
	}

	// Create job
	job := &models.Job{
		ID:                   uuid.New().String(),
		Type:                 req.Type,
		Status:               models.JobStatusQueued,
		Priority:             req.Priority,
		SubmittedAt:          time.Now(),
		ProjectID:            req.ProjectID,
		TaskSpec:             req.TaskSpec,
		ResourceRequirements: req.ResourceRequirements,
		DataAccess:           req.DataAccess,
	}

	// Set default priority if not specified
	if job.Priority == 0 {
		job.Priority = 1
	}

	// Enqueue job
	if err := s.queue.Enqueue(job); err != nil {
		http.Error(w, fmt.Sprintf("Failed to enqueue job: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(job)
}

// handleJobList handles job list requests
func (s *Server) handleJobList(w http.ResponseWriter, r *http.Request) {
	queueLength, err := s.queue.QueueLength()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get queue length: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"queue_length": queueLength,
	})
}

// handleJobByID handles job-specific requests
func (s *Server) handleJobByID(w http.ResponseWriter, r *http.Request) {
	// Extract job ID from path (e.g., /api/jobs/{id})
	jobID := r.URL.Path[len("/api/jobs/"):]

	switch r.Method {
	case http.MethodGet:
		s.handleGetJob(w, r, jobID)
	case http.MethodPost:
		// Handle job completion updates
		s.handleJobUpdate(w, r, jobID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGetJob retrieves a job by ID
func (s *Server) handleGetJob(w http.ResponseWriter, r *http.Request, jobID string) {
	job, err := s.queue.GetJob(jobID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Job not found: %v", err), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(job)
}

// handleJobUpdate handles job status updates
func (s *Server) handleJobUpdate(w http.ResponseWriter, r *http.Request, jobID string) {
	var result models.JobResult
	if err := json.NewDecoder(r.Body).Decode(&result); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Update job status
	if err := s.queue.UpdateJobStatus(jobID, result.Status, result.ErrorMessage); err != nil {
		http.Error(w, fmt.Sprintf("Failed to update job: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}
