package execution

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/mimir-aip/mimir-aip-go/pkg/config"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"github.com/mimir-aip/mimir-aip-go/pkg/queue"
)

func TestLocalBackendProcessesTaskInProcess(t *testing.T) {
	q, err := queue.NewQueue(nil)
	if err != nil {
		t.Fatalf("failed to create queue: %v", err)
	}
	defer q.Close()

	task := &models.WorkTask{
		ID:          "task-1",
		Type:        models.WorkTaskTypeDigitalTwinProcessing,
		Status:      models.WorkTaskStatusQueued,
		Priority:    1,
		SubmittedAt: time.Now().UTC(),
		ProjectID:   "project-1",
		TaskSpec: models.TaskSpec{
			ProjectID: "project-1",
			Parameters: map[string]any{
				"processing_run_id": "run-1",
				"digital_twin_id":   "twin-1",
			},
		},
		MaxRetries: 1,
	}
	if err := q.Enqueue(task); err != nil {
		t.Fatalf("failed to enqueue task: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/worktasks/", func(w http.ResponseWriter, r *http.Request) {
		taskID := strings.TrimPrefix(r.URL.Path, "/api/worktasks/")
		switch r.Method {
		case http.MethodGet:
			task, err := q.GetWorkTask(taskID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			json.NewEncoder(w).Encode(task)
		case http.MethodPost:
			var result models.WorkTaskResult
			if err := json.NewDecoder(r.Body).Decode(&result); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if err := q.ApplyWorkTaskResult(taskID, &result); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/internal/twin-runs/run-1/execute", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(models.TwinProcessingRun{
			ID:          "run-1",
			Status:      models.TwinProcessingRunStatusCompleted,
			TriggerType: models.TwinProcessingTriggerTypeManual,
			Metrics: map[string]any{
				"insight_count": int64(2),
			},
		})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	backend := NewLocalBackend(q, &config.Config{
		ExecutionMode:     config.ExecutionModeLocal,
		OrchestratorURL:   server.URL,
		MaxWorkers:        1,
		ConcurrencyLimits: map[string]int{string(models.WorkTaskTypeDigitalTwinProcessing): 1},
	})

	backend.processQueue(context.Background())

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		persisted, err := q.GetWorkTask(task.ID)
		if err != nil {
			t.Fatalf("failed to get work task: %v", err)
		}
		if persisted.Status == models.WorkTaskStatusCompleted {
			if persisted.ResultMetadata["processing_run_id"] != "run-1" {
				t.Fatalf("expected processing_run_id metadata, got %#v", persisted.ResultMetadata)
			}
			return
		}
		time.Sleep(25 * time.Millisecond)
	}

	persisted, err := q.GetWorkTask(task.ID)
	if err != nil {
		t.Fatalf("failed to get final work task: %v", err)
	}
	t.Fatalf("expected task completion, got status %s", persisted.Status)
}
