package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"github.com/mimir-aip/mimir-aip-go/pkg/scheduler"
)

// ScheduleHandler handles schedule-related HTTP requests
type ScheduleHandler struct {
	service *scheduler.Service
}

// NewScheduleHandler creates a new schedule handler
func NewScheduleHandler(service *scheduler.Service) *ScheduleHandler {
	return &ScheduleHandler{
		service: service,
	}
}

// HandleSchedules handles schedule list and create operations
func (h *ScheduleHandler) HandleSchedules(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.handleList(w, r)
	case http.MethodPost:
		h.handleCreate(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandleSchedule handles individual schedule operations
func (h *ScheduleHandler) HandleSchedule(w http.ResponseWriter, r *http.Request) {
	// Extract schedule ID from path
	scheduleID := strings.TrimPrefix(r.URL.Path, "/api/schedules/")
	if idx := strings.Index(scheduleID, "/"); idx != -1 {
		scheduleID = scheduleID[:idx]
	}

	switch r.Method {
	case http.MethodGet:
		h.handleGet(w, r, scheduleID)
	case http.MethodPut:
		h.handleUpdate(w, r, scheduleID)
	case http.MethodDelete:
		h.handleDelete(w, r, scheduleID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleList lists all schedules
func (h *ScheduleHandler) handleList(w http.ResponseWriter, r *http.Request) {
	// Check if filtering by project
	projectID := r.URL.Query().Get("project_id")

	var schedules []*models.Schedule
	var err error

	if projectID != "" {
		schedules, err = h.service.ListByProject(projectID)
	} else {
		schedules, err = h.service.List()
	}

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list schedules: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(schedules)
}

// handleCreate creates a new schedule
func (h *ScheduleHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var req models.ScheduleCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	schedule, err := h.service.Create(&req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create schedule: %v", err), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(schedule)
}

// handleGet retrieves a schedule
func (h *ScheduleHandler) handleGet(w http.ResponseWriter, r *http.Request, scheduleID string) {
	schedule, err := h.service.Get(scheduleID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Schedule not found: %v", err), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(schedule)
}

// handleUpdate updates a schedule
func (h *ScheduleHandler) handleUpdate(w http.ResponseWriter, r *http.Request, scheduleID string) {
	var req models.ScheduleUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	schedule, err := h.service.Update(scheduleID, &req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to update schedule: %v", err), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(schedule)
}

// handleDelete deletes a schedule
func (h *ScheduleHandler) handleDelete(w http.ResponseWriter, r *http.Request, scheduleID string) {
	if err := h.service.Delete(scheduleID); err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete schedule: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
