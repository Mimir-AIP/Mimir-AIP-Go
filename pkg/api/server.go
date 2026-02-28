package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mimir-aip/mimir-aip-go/pkg/api/doc"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"github.com/mimir-aip/mimir-aip-go/pkg/queue"
	"github.com/mimir-aip/mimir-aip-go/pkg/ws"
)

// Server provides HTTP API endpoints
type Server struct {
	queue           *queue.Queue
	port            string
	mux             *http.ServeMux
	workerAuthToken string // if non-empty, required as Bearer token on /api/worktasks/* paths
	hub             *ws.Hub
}

// NewServer creates a new API server.
// workerAuthToken gates the worker-facing /api/worktasks/* paths when non-empty.
func NewServer(q *queue.Queue, port string, workerAuthToken string) *Server {
	hub := ws.NewHub()
	go hub.Run()

	// Wire queue status-change events to the WebSocket hub
	q.OnStatusChange = func(task *models.WorkTask) {
		data, err := json.Marshal(map[string]interface{}{
			"event": "task_update",
			"task":  task,
		})
		if err == nil {
			hub.Broadcast(data)
		}
	}

	s := &Server{
		queue:           q,
		port:            port,
		mux:             http.NewServeMux(),
		workerAuthToken: workerAuthToken,
		hub:             hub,
	}

	s.registerRoutes()
	return s
}

// workerAuthMiddleware returns an http.HandlerFunc that validates the Authorization: Bearer
// header when a workerAuthToken is configured. Requests without a valid token receive 401.
func (s *Server) workerAuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	if s.workerAuthToken == "" {
		return next
	}
	return func(w http.ResponseWriter, r *http.Request) {
		const prefix = "Bearer "
		auth := r.Header.Get("Authorization")
		if len(auth) <= len(prefix) || auth[:len(prefix)] != prefix || auth[len(prefix):] != s.workerAuthToken {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

// registerRoutes sets up the HTTP routes
func (s *Server) registerRoutes() {
	s.mux.HandleFunc("/health", s.handleHealth)
	s.mux.HandleFunc("/ready", s.handleReady)
	s.mux.HandleFunc("/api/worktasks", s.workerAuthMiddleware(s.handleWorkTasks))
	s.mux.HandleFunc("/api/worktasks/", s.workerAuthMiddleware(s.handleWorkTaskByID))
	s.mux.HandleFunc("/api/metrics", s.handleMetrics)
	s.mux.HandleFunc("/ws/tasks", ws.WSHandler(s.hub))
	s.mux.HandleFunc("/openapi.yaml", s.handleOpenAPISpec)
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

	// Set default MaxRetries based on task type
	if task.MaxRetries == 0 {
		switch task.Type {
		case models.WorkTaskTypeMLTraining:
			task.MaxRetries = 2
		case models.WorkTaskTypePipelineExecution, models.WorkTaskTypeMLInference, models.WorkTaskTypeDigitalTwinUpdate:
			task.MaxRetries = 3
		default:
			task.MaxRetries = 3
		}
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

// MetricsResponse is the JSON shape returned by GET /api/metrics.
type MetricsResponse struct {
	Queue         QueueMetrics       `json:"queue"`
	TasksByStatus map[string]int     `json:"tasks_by_status"`
	TasksByType   map[string]int     `json:"tasks_by_type"`
	Timestamp     time.Time          `json:"timestamp"`
}

// QueueMetrics contains queue depth information.
type QueueMetrics struct {
	Length int64 `json:"length"`
}

// handleMetrics serves GET /api/metrics with task counts aggregated from the queue.
func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	tasks, err := s.queue.ListWorkTasks()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list tasks: %v", err), http.StatusInternalServerError)
		return
	}

	byStatus := make(map[string]int)
	byType := make(map[string]int)
	for _, t := range tasks {
		byStatus[string(t.Status)]++
		byType[string(t.Type)]++
	}

	qLen, _ := s.queue.QueueLength()

	resp := MetricsResponse{
		Queue:         QueueMetrics{Length: qLen},
		TasksByStatus: byStatus,
		TasksByType:   byType,
		Timestamp:     time.Now().UTC(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
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

// retryableSignals are error strings that indicate transient failures worth retrying.
var retryableSignals = []string{"OOMKilled", "Evicted", "DeadlineExceeded"}

// handleWorkTaskUpdate handles work task status updates
func (s *Server) handleWorkTaskUpdate(w http.ResponseWriter, r *http.Request, taskID string) {
	var result models.WorkTaskResult
	if err := json.NewDecoder(r.Body).Decode(&result); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// When a worker reports failure, check if the error is retryable
	if result.Status == models.WorkTaskStatusFailed && result.ErrorMessage != "" {
		for _, sig := range retryableSignals {
			if strings.Contains(result.ErrorMessage, sig) {
				if err := s.queue.RequeueWithRetry(taskID, result.ErrorMessage); err != nil {
					log.Printf("Failed to requeue task %s: %v", taskID, err)
					// Fall through to normal failure handling
					break
				}
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]string{"status": "requeued_for_retry"})
				return
			}
		}
	}

	if err := s.queue.UpdateWorkTaskStatus(taskID, result.Status, result.ErrorMessage); err != nil {
		http.Error(w, fmt.Sprintf("Failed to update work task status: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

// handleOpenAPISpec serves GET /openapi.yaml — the live, auto-generated spec.
func (s *Server) handleOpenAPISpec(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	spec, err := doc.GenerateSpec()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to generate spec: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/yaml")
	fmt.Fprint(w, spec)
}
