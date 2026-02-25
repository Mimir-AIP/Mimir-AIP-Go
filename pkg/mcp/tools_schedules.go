package mcp

import (
	"context"
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

func registerScheduleTools(s *server.MCPServer, m *MimirMCPServer) {
	// list_schedules
	s.AddTool(
		mcp.NewTool("list_schedules",
			mcp.WithDescription("List scheduled jobs, optionally filtered by project"),
			mcp.WithString("project_id",
				mcp.Description("Filter by project ID; omit to list all schedules"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			projectID := req.GetString("project_id", "")
			var (
				schedules []*models.Schedule
				err       error
			)
			if projectID != "" {
				schedules, err = m.schedulerSvc.ListByProject(projectID)
			} else {
				schedules, err = m.schedulerSvc.List()
			}
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			data, _ := json.Marshal(schedules)
			return mcp.NewToolResultText(string(data)), nil
		},
	)

	// get_schedule
	s.AddTool(
		mcp.NewTool("get_schedule",
			mcp.WithDescription("Get details of a specific schedule by ID"),
			mcp.WithString("id",
				mcp.Required(),
				mcp.Description("Schedule ID"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			id := req.GetString("id", "")
			if id == "" {
				return mcp.NewToolResultError("id is required"), nil
			}
			schedule, err := m.schedulerSvc.Get(id)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			data, _ := json.Marshal(schedule)
			return mcp.NewToolResultText(string(data)), nil
		},
	)

	// create_schedule
	s.AddTool(
		mcp.NewTool("create_schedule",
			mcp.WithDescription("Create a new cron-based schedule that triggers pipelines"),
			mcp.WithString("project_id",
				mcp.Required(),
				mcp.Description("Project ID"),
			),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("Schedule name"),
			),
			mcp.WithString("cron_schedule",
				mcp.Required(),
				mcp.Description(`Cron expression e.g. "0 * * * *" for every hour`),
			),
			mcp.WithString("pipeline_ids",
				mcp.Required(),
				mcp.Description("Comma-separated list of pipeline IDs to trigger"),
			),
			mcp.WithString("enabled",
				mcp.Description("Whether to enable the schedule immediately (default true)"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			projectID := req.GetString("project_id", "")
			name := req.GetString("name", "")
			cronSchedule := req.GetString("cron_schedule", "")
			pipelineIDsStr := req.GetString("pipeline_ids", "")
			if projectID == "" || name == "" || cronSchedule == "" || pipelineIDsStr == "" {
				return mcp.NewToolResultError("project_id, name, cron_schedule, and pipeline_ids are required"), nil
			}
			enabled := req.GetString("enabled", "true") != "false"
			createReq := &models.ScheduleCreateRequest{
				ProjectID:    projectID,
				Name:         name,
				CronSchedule: cronSchedule,
				Pipelines:    splitCSV(pipelineIDsStr),
				Enabled:      enabled,
			}
			schedule, err := m.schedulerSvc.Create(createReq)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			data, _ := json.Marshal(schedule)
			return mcp.NewToolResultText(string(data)), nil
		},
	)

	// update_schedule
	s.AddTool(
		mcp.NewTool("update_schedule",
			mcp.WithDescription("Update an existing schedule"),
			mcp.WithString("id",
				mcp.Required(),
				mcp.Description("Schedule ID"),
			),
			mcp.WithString("name",
				mcp.Description("New schedule name"),
			),
			mcp.WithString("cron_schedule",
				mcp.Description("New cron expression"),
			),
			mcp.WithString("pipeline_ids",
				mcp.Description("Comma-separated replacement list of pipeline IDs"),
			),
			mcp.WithString("enabled",
				mcp.Description("Enable or disable: true or false"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			id := req.GetString("id", "")
			if id == "" {
				return mcp.NewToolResultError("id is required"), nil
			}
			updateReq := &models.ScheduleUpdateRequest{}
			if name := req.GetString("name", ""); name != "" {
				updateReq.Name = &name
			}
			if cron := req.GetString("cron_schedule", ""); cron != "" {
				updateReq.CronSchedule = &cron
			}
			if pipelineIDsStr := req.GetString("pipeline_ids", ""); pipelineIDsStr != "" {
				pipelines := splitCSV(pipelineIDsStr)
				updateReq.Pipelines = &pipelines
			}
			if enabledStr := req.GetString("enabled", ""); enabledStr != "" {
				b := enabledStr == "true"
				updateReq.Enabled = &b
			}
			schedule, err := m.schedulerSvc.Update(id, updateReq)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			data, _ := json.Marshal(schedule)
			return mcp.NewToolResultText(string(data)), nil
		},
	)

	// delete_schedule
	s.AddTool(
		mcp.NewTool("delete_schedule",
			mcp.WithDescription("Delete a schedule by ID"),
			mcp.WithString("id",
				mcp.Required(),
				mcp.Description("Schedule ID"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			id := req.GetString("id", "")
			if id == "" {
				return mcp.NewToolResultError("id is required"), nil
			}
			if err := m.schedulerSvc.Delete(id); err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultText(`{"success":true}`), nil
		},
	)
}
