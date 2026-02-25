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
	s.mux.HandleFunc("/api/worktasks", s.handleWorkTasks)
	s.mux.HandleFunc("/api/worktasks/", s.handleWorkTaskByID)
}

// RegisterHandler adds a custom handler to the server
func (s *Server) RegisterHandler(path string, handler http.HandlerFunc) {
	s.mux.HandleFunc(path, handler)
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

// handleWorkTasks handles work task submission (POST) and listing (GET)
func (s *Server) handleWorkTasks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		s.handleWorkTaskSubmission(w, r)
	case http.MethodGet:
		s.handleWorkTaskList(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleWorkTaskSubmission handles work task submission requests
func (s *Server) handleWorkTaskSubmission(w http.ResponseWriter, r *http.Request) {
	var req models.WorkTaskSubmissionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Validate request
	if req.Type == "" {
		http.Error(w, "WorkTask type is required", http.StatusBadRequest)
		return
	}
	if req.ProjectID == "" {
		http.Error(w, "Project ID is required", http.StatusBadRequest)
		return
	}

	// Create work task
	task := &models.WorkTask{
		ID:                   uuid.New().String(),
		Type:                 req.Type,
		Status:               models.WorkTaskStatusQueued,
		Priority:             req.Priority,
		SubmittedAt:          time.Now(),
		ProjectID:            req.ProjectID,
		TaskSpec:             req.TaskSpec,
		ResourceRequirements: req.ResourceRequirements,
		DataAccess:           req.DataAccess,
	}

	// Set default priority if not specified
	if task.Priority == 0 {
		task.Priority = 1
	}

	// Enqueue work task
	if err := s.queue.Enqueue(task); err != nil {
		http.Error(w, fmt.Sprintf("Failed to enqueue work task: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(task)
}

// handleWorkTaskList handles work task list requests
func (s *Server) handleWorkTaskList(w http.ResponseWriter, r *http.Request) {
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

// handleWorkTaskByID handles work task-specific requests
func (s *Server) handleWorkTaskByID(w http.ResponseWriter, r *http.Request) {
	// Extract work task ID from path (e.g., /api/worktasks/{id})
	taskID := r.URL.Path[len("/api/worktasks/"):]

	switch r.Method {
	case http.MethodGet:
		s.handleGetWorkTask(w, r, taskID)
	case http.MethodPost:
		// Handle work task completion updates
		s.handleWorkTaskUpdate(w, r, taskID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGetWorkTask retrieves a work task by ID
func (s *Server) handleGetWorkTask(w http.ResponseWriter, r *http.Request, taskID string) {
	task, err := s.queue.GetWorkTask(taskID)
	if err != nil {
		http.Error(w, fmt.Sprintf("WorkTask not found: %v", err), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(task)
}

// handleWorkTaskUpdate handles work task status updates
func (s *Server) handleWorkTaskUpdate(w http.ResponseWriter, r *http.Request, taskID string) {
	var result models.WorkTaskResult
	if err := json.NewDecoder(r.Body).Decode(&result); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	if err := s.queue.UpdateWorkTaskStatus(taskID, result.Status, result.ErrorMessage); err != nil {
		http.Error(w, fmt.Sprintf("Failed to update work task status: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}
