package mcp

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerSystemTools(s *server.MCPServer, _ *MimirMCPServer) {
	s.AddTool(
		mcp.NewTool("health_check",
			mcp.WithDescription("Check the health status of the Mimir AIP platform"),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return mcp.NewToolResultText(`{"status":"healthy","platform":"Mimir AIP","version":"1.0.0"}`), nil
		},
	)
}
