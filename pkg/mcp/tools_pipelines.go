package mcp

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

func registerPipelineTools(s *server.MCPServer, m *MimirMCPServer) {
	// list_pipelines
	s.AddTool(
		mcp.NewTool("list_pipelines",
			mcp.WithDescription("List pipelines, optionally filtered by project"),
			mcp.WithString("project_id",
				mcp.Description("Filter by project ID; omit to list all pipelines"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			projectID := req.GetString("project_id", "")
			var (
				pipelines []*models.Pipeline
				err       error
			)
			if projectID != "" {
				pipelines, err = m.pipelineSvc.ListByProject(projectID)
			} else {
				pipelines, err = m.pipelineSvc.List()
			}
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			data, _ := json.Marshal(pipelines)
			return mcp.NewToolResultText(string(data)), nil
		},
	)

	// get_pipeline
	s.AddTool(
		mcp.NewTool("get_pipeline",
			mcp.WithDescription("Get details of a specific pipeline by ID"),
			mcp.WithString("id",
				mcp.Required(),
				mcp.Description("Pipeline ID"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			id := req.GetString("id", "")
			if id == "" {
				return mcp.NewToolResultError("id is required"), nil
			}
			pipeline, err := m.pipelineSvc.Get(id)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			data, _ := json.Marshal(pipeline)
			return mcp.NewToolResultText(string(data)), nil
		},
	)

	// create_pipeline
	s.AddTool(
		mcp.NewTool("create_pipeline",
			mcp.WithDescription("Create a new pipeline for a project"),
			mcp.WithString("project_id",
				mcp.Required(),
				mcp.Description("Project ID the pipeline belongs to"),
			),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("Pipeline name"),
			),
			mcp.WithString("type",
				mcp.Required(),
				mcp.Description("Pipeline type: ingestion, processing, or output"),
			),
			mcp.WithString("steps",
				mcp.Required(),
				mcp.Description(`JSON array of pipeline steps. Each step: {"name":"step1","plugin":"default","action":"transform","parameters":{}}`),
			),
			mcp.WithString("description",
				mcp.Description("Optional pipeline description"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			projectID := req.GetString("project_id", "")
			name := req.GetString("name", "")
			pType := req.GetString("type", "")
			stepsJSON := req.GetString("steps", "")
			if projectID == "" || name == "" || pType == "" || stepsJSON == "" {
				return mcp.NewToolResultError("project_id, name, type, and steps are required"), nil
			}
			var steps []models.PipelineStep
			if err := json.Unmarshal([]byte(stepsJSON), &steps); err != nil {
				return mcp.NewToolResultError("steps must be a valid JSON array: " + err.Error()), nil
			}
			createReq := &models.PipelineCreateRequest{
				ProjectID:   projectID,
				Name:        name,
				Type:        models.PipelineType(pType),
				Description: req.GetString("description", ""),
				Steps:       steps,
			}
			pipeline, err := m.pipelineSvc.Create(createReq)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			data, _ := json.Marshal(pipeline)
			return mcp.NewToolResultText(string(data)), nil
		},
	)

	// execute_pipeline
	s.AddTool(
		mcp.NewTool("execute_pipeline",
			mcp.WithDescription("Enqueue a pipeline for asynchronous execution; returns a work task ID"),
			mcp.WithString("id",
				mcp.Required(),
				mcp.Description("Pipeline ID to execute"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			id := req.GetString("id", "")
			if id == "" {
				return mcp.NewToolResultError("id is required"), nil
			}
			pipeline, err := m.pipelineSvc.Get(id)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			taskID := uuid.New().String()
			task := &models.WorkTask{
				ID:          taskID,
				Type:        models.WorkTaskTypePipelineExecution,
				ProjectID:   pipeline.ProjectID,
				Priority:    5,
				Status:      models.WorkTaskStatusQueued,
				SubmittedAt: time.Now().UTC(),
				TaskSpec: models.TaskSpec{
					PipelineID: id,
					ProjectID:  pipeline.ProjectID,
					Parameters: map[string]any{"pipeline_id": id},
				},
			}
			if err := m.queue.Enqueue(task); err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			data, _ := json.Marshal(map[string]string{"task_id": taskID, "pipeline_id": id, "status": "queued"})
			return mcp.NewToolResultText(string(data)), nil
		},
	)

	// update_pipeline
	s.AddTool(
		mcp.NewTool("update_pipeline",
			mcp.WithDescription("Update an existing pipeline's metadata or steps"),
			mcp.WithString("id",
				mcp.Required(),
				mcp.Description("Pipeline ID"),
			),
			mcp.WithString("description",
				mcp.Description("New description"),
			),
			mcp.WithString("steps",
				mcp.Description(`Replacement JSON array of pipeline steps`),
			),
			mcp.WithString("status",
				mcp.Description("New status: active, inactive, or archived"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			id := req.GetString("id", "")
			if id == "" {
				return mcp.NewToolResultError("id is required"), nil
			}
			updateReq := &models.PipelineUpdateRequest{}
			if desc := req.GetString("description", ""); desc != "" {
				updateReq.Description = &desc
			}
			if stepsJSON := req.GetString("steps", ""); stepsJSON != "" {
				var steps []models.PipelineStep
				if err := json.Unmarshal([]byte(stepsJSON), &steps); err != nil {
					return mcp.NewToolResultError("steps must be a valid JSON array: " + err.Error()), nil
				}
				updateReq.Steps = &steps
			}
			if st := req.GetString("status", ""); st != "" {
				ps := models.PipelineStatus(st)
				updateReq.Status = &ps
			}
			pipeline, err := m.pipelineSvc.Update(id, updateReq)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			data, _ := json.Marshal(pipeline)
			return mcp.NewToolResultText(string(data)), nil
		},
	)

	// delete_pipeline
	s.AddTool(
		mcp.NewTool("delete_pipeline",
			mcp.WithDescription("Delete a pipeline by ID"),
			mcp.WithString("id",
				mcp.Required(),
				mcp.Description("Pipeline ID"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			id := req.GetString("id", "")
			if id == "" {
				return mcp.NewToolResultError("id is required"), nil
			}
			if err := m.pipelineSvc.Delete(id); err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultText(`{"success":true}`), nil
		},
	)
}
