package mcp

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

func registerAnalysisTools(s *server.MCPServer, m *MimirMCPServer) {
	s.AddTool(
		mcp.NewTool("run_resolver_analysis",
			mcp.WithDescription("Run cross-source resolver analysis for a project and persist reviewable findings"),
			mcp.WithString("project_id", mcp.Required(), mcp.Description("Project ID")),
			mcp.WithString("storage_ids", mcp.Required(), mcp.Description("Comma-separated storage config IDs; at least two are required")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			run, items, err := m.analysisSvc.RunResolver(req.GetString("project_id", ""), splitCSV(req.GetString("storage_ids", "")))
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			metrics, err := m.analysisSvc.ResolverMetrics(req.GetString("project_id", ""))
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			data, _ := json.Marshal(map[string]any{"run": run, "review_items": items, "metrics": metrics})
			return mcp.NewToolResultText(string(data)), nil
		},
	)

	s.AddTool(
		mcp.NewTool("get_resolver_metrics",
			mcp.WithDescription("Get resolver precision metrics for one project"),
			mcp.WithString("project_id", mcp.Required(), mcp.Description("Project ID")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			metrics, err := m.analysisSvc.ResolverMetrics(req.GetString("project_id", ""))
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			data, _ := json.Marshal(metrics)
			return mcp.NewToolResultText(string(data)), nil
		},
	)

	s.AddTool(
		mcp.NewTool("list_review_items",
			mcp.WithDescription("List persisted review queue items for one project"),
			mcp.WithString("project_id", mcp.Required(), mcp.Description("Project ID")),
			mcp.WithString("status", mcp.Description("Optional status filter")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			items, err := m.analysisSvc.ListReviewItems(req.GetString("project_id", ""), req.GetString("status", ""))
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			data, _ := json.Marshal(items)
			return mcp.NewToolResultText(string(data)), nil
		},
	)

	s.AddTool(
		mcp.NewTool("decide_review_item",
			mcp.WithDescription("Accept or reject one persisted review item"),
			mcp.WithString("id", mcp.Required(), mcp.Description("Review item ID")),
			mcp.WithString("decision", mcp.Required(), mcp.Description("Decision: accept or reject")),
			mcp.WithString("reviewer", mcp.Description("Optional reviewer identifier")),
			mcp.WithString("rationale", mcp.Description("Optional rationale for the decision")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			item, err := m.analysisSvc.DecideReviewItem(req.GetString("id", ""), &models.ReviewDecisionRequest{
				Decision:  models.ReviewDecision(req.GetString("decision", "")),
				Reviewer:  req.GetString("reviewer", ""),
				Rationale: req.GetString("rationale", ""),
			})
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			data, _ := json.Marshal(item)
			return mcp.NewToolResultText(string(data)), nil
		},
	)

	s.AddTool(
		mcp.NewTool("list_insights",
			mcp.WithDescription("List persisted project insights with optional severity and confidence filters"),
			mcp.WithString("project_id", mcp.Required(), mcp.Description("Project ID")),
			mcp.WithString("severity", mcp.Description("Optional severity filter")),
			mcp.WithString("min_confidence", mcp.Description("Optional minimum confidence value")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			minConfidence := 0.0
			if raw := strings.TrimSpace(req.GetString("min_confidence", "")); raw != "" {
				parsed, err := strconv.ParseFloat(raw, 64)
				if err != nil {
					return mcp.NewToolResultError("min_confidence must be numeric"), nil
				}
				minConfidence = parsed
			}
			insights, err := m.analysisSvc.ListInsights(req.GetString("project_id", ""), req.GetString("severity", ""), minConfidence)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			data, _ := json.Marshal(insights)
			return mcp.NewToolResultText(string(data)), nil
		},
	)

	s.AddTool(
		mcp.NewTool("generate_insights",
			mcp.WithDescription("Generate and persist autonomous insights for one project"),
			mcp.WithString("project_id", mcp.Required(), mcp.Description("Project ID")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			run, insights, err := m.analysisSvc.GenerateProjectInsights(req.GetString("project_id", ""))
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			data, _ := json.Marshal(map[string]any{"run": run, "insights": insights})
			return mcp.NewToolResultText(string(data)), nil
		},
	)
}
