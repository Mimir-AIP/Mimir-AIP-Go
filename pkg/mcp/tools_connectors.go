package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

func registerConnectorTools(s *server.MCPServer, m *MimirMCPServer) {
	s.AddTool(
		mcp.NewTool("list_connector_templates",
			mcp.WithDescription("List bundled connector templates available for guided ingestion setup"),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			data, _ := json.Marshal(m.connectorSvc.ListTemplates())
			return mcp.NewToolResultText(string(data)), nil
		},
	)

	s.AddTool(
		mcp.NewTool("materialize_connector",
			mcp.WithDescription("Create a pipeline and optional schedule from a bundled connector template"),
			mcp.WithString("project_id", mcp.Required(), mcp.Description("Project ID")),
			mcp.WithString("kind", mcp.Required(), mcp.Description("Connector template kind")),
			mcp.WithString("name", mcp.Required(), mcp.Description("Pipeline name to create")),
			mcp.WithString("storage_id", mcp.Required(), mcp.Description("Destination storage configuration ID")),
			mcp.WithString("description", mcp.Description("Optional connector description")),
			mcp.WithString("source_config_json", mcp.Required(), mcp.Description("JSON object containing the connector source configuration")),
			mcp.WithString("schedule_cron", mcp.Description("Optional cron expression for recurring execution")),
			mcp.WithString("schedule_name", mcp.Description("Optional schedule name override")),
			mcp.WithString("schedule_enabled", mcp.Description("Optional schedule enabled flag: true or false (default true)")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			sourceConfig, err := parseJSONMap(req.GetString("source_config_json", ""))
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			setupReq := &models.ConnectorSetupRequest{
				ProjectID:    req.GetString("project_id", ""),
				Kind:         req.GetString("kind", ""),
				Name:         req.GetString("name", ""),
				Description:  req.GetString("description", ""),
				StorageID:    req.GetString("storage_id", ""),
				SourceConfig: sourceConfig,
			}
			if cron := strings.TrimSpace(req.GetString("schedule_cron", "")); cron != "" {
				setupReq.Schedule = &models.ConnectorScheduleRequest{
					Name:         req.GetString("schedule_name", ""),
					CronSchedule: cron,
					Enabled:      parseBoolString(req.GetString("schedule_enabled", "true"), true),
				}
			}
			result, err := m.connectorSvc.Materialize(setupReq)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			data, _ := json.Marshal(result)
			return mcp.NewToolResultText(string(data)), nil
		},
	)
}

func parseJSONMap(raw string) (map[string]any, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, fmt.Errorf("source_config_json is required")
	}
	decoded := map[string]any{}
	if err := json.Unmarshal([]byte(trimmed), &decoded); err != nil {
		return nil, fmt.Errorf("source_config_json must be a JSON object: %w", err)
	}
	return decoded, nil
}

func parseBoolString(raw string, fallback bool) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "true", "1", "yes", "on":
		return true
	case "false", "0", "no", "off":
		return false
	default:
		return fallback
	}
}
