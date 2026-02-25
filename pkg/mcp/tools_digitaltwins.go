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

func registerDigitalTwinTools(s *server.MCPServer, m *MimirMCPServer) {
	// list_digital_twins
	s.AddTool(
		mcp.NewTool("list_digital_twins",
			mcp.WithDescription("List digital twins, optionally filtered by project"),
			mcp.WithString("project_id",
				mcp.Description("Filter by project ID; omit to list all digital twins"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			projectID := req.GetString("project_id", "")
			var (
				twins []*models.DigitalTwin
				err   error
			)
			if projectID != "" {
				twins, err = m.dtSvc.ListDigitalTwinsByProject(projectID)
			} else {
				twins, err = m.dtSvc.ListDigitalTwins()
			}
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			data, _ := json.Marshal(twins)
			return mcp.NewToolResultText(string(data)), nil
		},
	)

	// get_digital_twin
	s.AddTool(
		mcp.NewTool("get_digital_twin",
			mcp.WithDescription("Get details of a specific digital twin by ID"),
			mcp.WithString("id",
				mcp.Required(),
				mcp.Description("Digital twin ID"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			id := req.GetString("id", "")
			if id == "" {
				return mcp.NewToolResultError("id is required"), nil
			}
			twin, err := m.dtSvc.GetDigitalTwin(id)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			data, _ := json.Marshal(twin)
			return mcp.NewToolResultText(string(data)), nil
		},
	)

	// create_digital_twin
	s.AddTool(
		mcp.NewTool("create_digital_twin",
			mcp.WithDescription("Create a new digital twin for a project"),
			mcp.WithString("project_id",
				mcp.Required(),
				mcp.Description("Project ID"),
			),
			mcp.WithString("ontology_id",
				mcp.Required(),
				mcp.Description("Ontology ID that defines the twin's entity model"),
			),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("Digital twin name"),
			),
			mcp.WithString("description",
				mcp.Description("Optional description"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			projectID := req.GetString("project_id", "")
			ontologyID := req.GetString("ontology_id", "")
			name := req.GetString("name", "")
			if projectID == "" || ontologyID == "" || name == "" {
				return mcp.NewToolResultError("project_id, ontology_id, and name are required"), nil
			}
			createReq := &models.DigitalTwinCreateRequest{
				ProjectID:   projectID,
				OntologyID:  ontologyID,
				Name:        name,
				Description: req.GetString("description", ""),
			}
			twin, err := m.dtSvc.CreateDigitalTwin(createReq)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			data, _ := json.Marshal(twin)
			return mcp.NewToolResultText(string(data)), nil
		},
	)

	// sync_digital_twin
	s.AddTool(
		mcp.NewTool("sync_digital_twin",
			mcp.WithDescription("Enqueue a digital twin sync job to update entities from storage; returns a work task ID"),
			mcp.WithString("id",
				mcp.Required(),
				mcp.Description("Digital twin ID to sync"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			id := req.GetString("id", "")
			if id == "" {
				return mcp.NewToolResultError("id is required"), nil
			}
			twin, err := m.dtSvc.GetDigitalTwin(id)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			taskID := uuid.New().String()
			task := &models.WorkTask{
				ID:          taskID,
				Type:        models.WorkTaskTypeDigitalTwinUpdate,
				ProjectID:   twin.ProjectID,
				Priority:    5,
				Status:      models.WorkTaskStatusQueued,
				SubmittedAt: time.Now().UTC(),
				TaskSpec: models.TaskSpec{
					ProjectID:  twin.ProjectID,
					Parameters: map[string]any{"digital_twin_id": id},
				},
			}
			if err := m.queue.Enqueue(task); err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			data, _ := json.Marshal(map[string]string{
				"task_id":         taskID,
				"digital_twin_id": id,
				"status":          "queued",
			})
			return mcp.NewToolResultText(string(data)), nil
		},
	)

	// query_digital_twin
	s.AddTool(
		mcp.NewTool("query_digital_twin",
			mcp.WithDescription("Execute a SPARQL query against a digital twin's entity graph"),
			mcp.WithString("id",
				mcp.Required(),
				mcp.Description("Digital twin ID"),
			),
			mcp.WithString("sparql_query",
				mcp.Required(),
				mcp.Description(`SPARQL query string e.g. SELECT ?s ?type WHERE { ?s a ?type } LIMIT 10`),
			),
			mcp.WithString("limit",
				mcp.Description("Maximum number of results to return (default 100)"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			id := req.GetString("id", "")
			query := req.GetString("sparql_query", "")
			if id == "" || query == "" {
				return mcp.NewToolResultError("id and sparql_query are required"), nil
			}
			limit := req.GetInt("limit", 100)
			queryReq := &models.QueryRequest{
				Query: query,
				Limit: limit,
			}
			result, err := m.dtSvc.Query(id, queryReq)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			data, _ := json.Marshal(result)
			return mcp.NewToolResultText(string(data)), nil
		},
	)
}
