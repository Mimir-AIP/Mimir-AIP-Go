package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"github.com/mimir-aip/mimir-aip-go/pkg/queue"
)

func TestHandleHealthReportsDegradedWhenFailedTasksExist(t *testing.T) {
	q, err := queue.NewQueue(nil)
	if err != nil {
		t.Fatalf("failed to create queue: %v", err)
	}
	defer q.Close()

	if err := q.Enqueue(&models.WorkTask{
		ID:          "failed-task",
		Type:        models.WorkTaskTypeMLTraining,
		Status:      models.WorkTaskStatusFailed,
		Priority:    1,
		SubmittedAt: time.Now().UTC(),
		ProjectID:   "project-1",
	}); err != nil {
		t.Fatalf("failed to enqueue failed task: %v", err)
	}

	server := NewServer(q, "0", "")
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	resp := httptest.NewRecorder()

	server.mux.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.Code)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode health response: %v", err)
	}
	if payload["status"] != "degraded" {
		t.Fatalf("expected degraded health status, got %#v", payload["status"])
	}
	if payload["failed_tasks"].(float64) != 1 {
		t.Fatalf("expected failed_tasks=1, got %#v", payload["failed_tasks"])
	}
}

func TestHandleReadyReportsReadyWhenQueueConfigured(t *testing.T) {
	q, err := queue.NewQueue(nil)
	if err != nil {
		t.Fatalf("failed to create queue: %v", err)
	}
	defer q.Close()

	server := NewServer(q, "0", "")
	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	resp := httptest.NewRecorder()

	server.mux.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.Code)
	}

	var payload map[string]string
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode readiness response: %v", err)
	}
	if payload["status"] != "ready" {
		t.Fatalf("expected ready status, got %#v", payload["status"])
	}
}
