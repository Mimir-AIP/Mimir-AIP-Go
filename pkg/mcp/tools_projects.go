package mcp

import (
	"context"
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

func registerProjectTools(s *server.MCPServer, m *MimirMCPServer) {
	// list_projects
	s.AddTool(
		mcp.NewTool("list_projects",
			mcp.WithDescription("List all projects in the Mimir platform"),
			mcp.WithString("status",
				mcp.Description("Filter by project status: active, archived, or draft"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			query := &models.ProjectListQuery{Status: req.GetString("status", "")}
			projects, err := m.projectSvc.List(query)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			data, _ := json.Marshal(projects)
			return mcp.NewToolResultText(string(data)), nil
		},
	)

	// get_project
	s.AddTool(
		mcp.NewTool("get_project",
			mcp.WithDescription("Get details of a specific project by ID"),
			mcp.WithString("id",
				mcp.Required(),
				mcp.Description("Project ID"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			id := req.GetString("id", "")
			if id == "" {
				return mcp.NewToolResultError("id is required"), nil
			}
			project, err := m.projectSvc.Get(id)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			data, _ := json.Marshal(project)
			return mcp.NewToolResultText(string(data)), nil
		},
	)

	// create_project
	s.AddTool(
		mcp.NewTool("create_project",
			mcp.WithDescription("Create a new project in the Mimir platform"),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("Project name: 3-50 alphanumeric characters, hyphens, or underscores"),
			),
			mcp.WithString("description",
				mcp.Description("Human-readable description of the project"),
			),
			mcp.WithString("version",
				mcp.Description("Semantic version string e.g. 1.0.0"),
			),
			mcp.WithString("tags",
				mcp.Description("Comma-separated list of tags e.g. production,ml,iot"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			name := req.GetString("name", "")
			if name == "" {
				return mcp.NewToolResultError("name is required"), nil
			}
			createReq := &models.ProjectCreateRequest{
				Name:        name,
				Description: req.GetString("description", ""),
				Version:     req.GetString("version", ""),
			}
			if tagsStr := req.GetString("tags", ""); tagsStr != "" {
				createReq.Tags = splitCSV(tagsStr)
			}
			project, err := m.projectSvc.Create(createReq)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			data, _ := json.Marshal(project)
			return mcp.NewToolResultText(string(data)), nil
		},
	)

	// update_project
	s.AddTool(
		mcp.NewTool("update_project",
			mcp.WithDescription("Update an existing project's metadata"),
			mcp.WithString("id",
				mcp.Required(),
				mcp.Description("Project ID"),
			),
			mcp.WithString("description",
				mcp.Description("New description"),
			),
			mcp.WithString("version",
				mcp.Description("New semantic version e.g. 2.0.0"),
			),
			mcp.WithString("status",
				mcp.Description("New status: active, archived, or draft"),
			),
			mcp.WithString("tags",
				mcp.Description("Comma-separated replacement tag list"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			id := req.GetString("id", "")
			if id == "" {
				return mcp.NewToolResultError("id is required"), nil
			}
			updateReq := &models.ProjectUpdateRequest{}
			if desc := req.GetString("description", ""); desc != "" {
				updateReq.Description = &desc
			}
			if ver := req.GetString("version", ""); ver != "" {
				updateReq.Version = &ver
			}
			if st := req.GetString("status", ""); st != "" {
				ps := models.ProjectStatus(st)
				updateReq.Status = &ps
			}
			if tagsStr := req.GetString("tags", ""); tagsStr != "" {
				tags := splitCSV(tagsStr)
				updateReq.Tags = &tags
			}
			project, err := m.projectSvc.Update(id, updateReq)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			data, _ := json.Marshal(project)
			return mcp.NewToolResultText(string(data)), nil
		},
	)

	// delete_project
	s.AddTool(
		mcp.NewTool("delete_project",
			mcp.WithDescription("Delete (archive) a project by ID"),
			mcp.WithString("id",
				mcp.Required(),
				mcp.Description("Project ID"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			id := req.GetString("id", "")
			if id == "" {
				return mcp.NewToolResultError("id is required"), nil
			}
			if err := m.projectSvc.Delete(id); err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultText(`{"success":true}`), nil
		},
	)

	// clone_project
	s.AddTool(
		mcp.NewTool("clone_project",
			mcp.WithDescription("Clone an existing project under a new name"),
			mcp.WithString("id",
				mcp.Required(),
				mcp.Description("Source project ID"),
			),
			mcp.WithString("new_name",
				mcp.Required(),
				mcp.Description("Name for the cloned project"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			id := req.GetString("id", "")
			newName := req.GetString("new_name", "")
			if id == "" {
				return mcp.NewToolResultError("id is required"), nil
			}
			if newName == "" {
				return mcp.NewToolResultError("new_name is required"), nil
			}
			project, err := m.projectSvc.Clone(id, newName)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			data, _ := json.Marshal(project)
			return mcp.NewToolResultText(string(data)), nil
		},
	)
}
