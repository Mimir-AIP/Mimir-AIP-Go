package mcp

import (
	"context"
	"encoding/json"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

func registerTaskTools(s *server.MCPServer, m *MimirMCPServer) {
	// list_work_tasks
	s.AddTool(
		mcp.NewTool("list_work_tasks",
			mcp.WithDescription("List work tasks in the queue, optionally filtered by status or type"),
			mcp.WithString("status",
				mcp.Description("Filter by task status: queued, scheduled, spawned, executing, completed, failed, timeout, cancelled"),
			),
			mcp.WithString("type",
				mcp.Description("Filter by task type: pipeline_execution, ml_training, ml_inference, digital_twin_processing"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			filterStatus := models.WorkTaskStatus(req.GetString("status", ""))
			filterType := models.WorkTaskType(req.GetString("type", ""))

			tasks, err := m.queue.ListWorkTasks()
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			// Apply filters
			filtered := make([]*models.WorkTask, 0, len(tasks))
			for _, t := range tasks {
				if filterStatus != "" && t.Status != filterStatus {
					continue
				}
				if filterType != "" && t.Type != filterType {
					continue
				}
				filtered = append(filtered, t)
			}
			data, _ := json.Marshal(filtered)
			return mcp.NewToolResultText(string(data)), nil
		},
	)

	// get_work_task
	s.AddTool(
		mcp.NewTool("get_work_task",
			mcp.WithDescription("Get the current state of a specific work task by ID"),
			mcp.WithString("id",
				mcp.Required(),
				mcp.Description("Work task ID"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			id := req.GetString("id", "")
			if id == "" {
				return mcp.NewToolResultError("id is required"), nil
			}
			task, err := m.queue.GetWorkTask(id)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			data, _ := json.Marshal(task)
			return mcp.NewToolResultText(string(data)), nil
		},
	)

	// wait_for_task
	s.AddTool(
		mcp.NewTool("wait_for_task",
			mcp.WithDescription("Poll a work task until it reaches a terminal state (completed, failed, timeout, cancelled) or the timeout expires"),
			mcp.WithString("id",
				mcp.Required(),
				mcp.Description("Work task ID"),
			),
			mcp.WithString("timeout_seconds",
				mcp.Description("Maximum seconds to wait (default 300, max 600)"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			id := req.GetString("id", "")
			if id == "" {
				return mcp.NewToolResultError("id is required"), nil
			}
			timeoutSec := req.GetInt("timeout_seconds", 300)
			if timeoutSec <= 0 || timeoutSec > 600 {
				timeoutSec = 300
			}

			deadline := time.Now().Add(time.Duration(timeoutSec) * time.Second)
			terminal := map[models.WorkTaskStatus]bool{
				models.WorkTaskStatusCompleted: true,
				models.WorkTaskStatusFailed:    true,
				models.WorkTaskStatusTimeout:   true,
				models.WorkTaskStatusCancelled: true,
			}

			for time.Now().Before(deadline) {
				task, err := m.queue.GetWorkTask(id)
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				if terminal[task.Status] {
					data, _ := json.Marshal(task)
					return mcp.NewToolResultText(string(data)), nil
				}
				select {
				case <-ctx.Done():
					return mcp.NewToolResultError("context cancelled while waiting for task"), nil
				case <-time.After(2 * time.Second):
				}
			}

			// Return current state on timeout
			task, err := m.queue.GetWorkTask(id)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			data, _ := json.Marshal(map[string]any{
				"task":      task,
				"timed_out": true,
				"message":   "task did not reach a terminal state within the timeout",
			})
			return mcp.NewToolResultText(string(data)), nil
		},
	)
}
