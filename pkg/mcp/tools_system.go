package mcp

import (
	"context"
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerSystemTools(s *server.MCPServer, m *MimirMCPServer) {
	s.AddTool(
		mcp.NewTool("health_check",
			mcp.WithDescription("Check the health status of the Mimir AIP platform"),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			payload := map[string]interface{}{"status": "unhealthy", "ready": false}
			if m != nil && m.queue != nil {
				snapshot := m.queue.Snapshot()
				status := "healthy"
				if snapshot.FailedTasks > 0 {
					status = "degraded"
				}
				payload = map[string]interface{}{
					"status":          status,
					"ready":           true,
					"queue_length":    snapshot.QueueLength,
					"failed_tasks":    snapshot.FailedTasks,
					"tasks_by_status": snapshot.TasksByStatus,
					"tasks_by_type":   snapshot.TasksByType,
				}
			}
			data, err := json.Marshal(payload)
			if err != nil {
				return nil, err
			}
			return mcp.NewToolResultText(string(data)), nil
		},
	)
}
