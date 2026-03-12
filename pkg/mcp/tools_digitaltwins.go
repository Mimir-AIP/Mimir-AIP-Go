package mcp

import (
	"context"
	"encoding/json"

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

	// update_digital_twin
	s.AddTool(
		mcp.NewTool("update_digital_twin",
			mcp.WithDescription("Update an existing digital twin's metadata or status"),
			mcp.WithString("id",
				mcp.Required(),
				mcp.Description("Digital twin ID"),
			),
			mcp.WithString("name",
				mcp.Description("New name"),
			),
			mcp.WithString("description",
				mcp.Description("New description"),
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
			updateReq := &models.DigitalTwinUpdateRequest{}
			if name := req.GetString("name", ""); name != "" {
				updateReq.Name = &name
			}
			if desc := req.GetString("description", ""); desc != "" {
				updateReq.Description = &desc
			}
			if st := req.GetString("status", ""); st != "" {
				updateReq.Status = &st
			}
			twin, err := m.dtSvc.UpdateDigitalTwin(id, updateReq)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			data, _ := json.Marshal(twin)
			return mcp.NewToolResultText(string(data)), nil
		},
	)

	// delete_digital_twin
	s.AddTool(
		mcp.NewTool("delete_digital_twin",
			mcp.WithDescription("Delete a digital twin by ID"),
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
			if err := m.dtSvc.DeleteDigitalTwin(id); err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultText(`{"success":true}`), nil
		},
	)

	// process_digital_twin
	s.AddTool(
		mcp.NewTool("process_digital_twin",
			mcp.WithDescription("Queue one explicit digital twin processing run and return the queued run record"),
			mcp.WithString("id",
				mcp.Required(),
				mcp.Description("Digital twin ID to process"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			id := req.GetString("id", "")
			if id == "" {
				return mcp.NewToolResultError("id is required"), nil
			}
			run, err := m.twinProcessor.RequestRun(id, &models.TwinProcessingRunCreateRequest{
				TriggerType: models.TwinProcessingTriggerTypeManual,
				TriggerRef:  "mcp/manual",
			})
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			data, _ := json.Marshal(run)
			return mcp.NewToolResultText(string(data)), nil
		},
	)

	// list_twin_processing_runs
	s.AddTool(
		mcp.NewTool("list_twin_processing_runs",
			mcp.WithDescription("List recent processing runs for one digital twin"),
			mcp.WithString("id", mcp.Required(), mcp.Description("Digital twin ID")),
			mcp.WithString("limit", mcp.Description("Optional maximum number of runs to return")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			id := req.GetString("id", "")
			if id == "" {
				return mcp.NewToolResultError("id is required"), nil
			}
			limit := req.GetInt("limit", 25)
			runs, err := m.twinProcessor.ListRuns(id, limit)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			data, _ := json.Marshal(runs)
			return mcp.NewToolResultText(string(data)), nil
		},
	)

	// list_twin_alert_events
	s.AddTool(
		mcp.NewTool("list_twin_alert_events",
			mcp.WithDescription("List recent alert events emitted for one digital twin"),
			mcp.WithString("id", mcp.Required(), mcp.Description("Digital twin ID")),
			mcp.WithString("limit", mcp.Description("Optional maximum number of alerts to return")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			id := req.GetString("id", "")
			if id == "" {
				return mcp.NewToolResultError("id is required"), nil
			}
			limit := req.GetInt("limit", 50)
			alerts, err := m.twinProcessor.ListAlerts(id, limit)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			data, _ := json.Marshal(alerts)
			return mcp.NewToolResultText(string(data)), nil
		},
	)

	// list_twin_automations
	s.AddTool(
		mcp.NewTool("list_twin_automations",
			mcp.WithDescription("List explicit automations scoped to one digital twin"),
			mcp.WithString("id", mcp.Required(), mcp.Description("Digital twin ID")),
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
			automations, err := m.automationSvc.ListByProject(twin.ProjectID)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			filtered := make([]*models.Automation, 0)
			for _, automation := range automations {
				if automation.TargetType == models.AutomationTargetTypeDigitalTwin && automation.TargetID == id {
					filtered = append(filtered, automation)
				}
			}
			data, _ := json.Marshal(filtered)
			return mcp.NewToolResultText(string(data)), nil
		},
	)

	// create_twin_automation
	s.AddTool(
		mcp.NewTool("create_twin_automation",
			mcp.WithDescription("Create a twin-scoped automation. Target metadata is derived from the digital twin route."),
			mcp.WithString("id", mcp.Required(), mcp.Description("Digital twin ID")),
			mcp.WithString("name", mcp.Required(), mcp.Description("Automation name")),
			mcp.WithString("description", mcp.Description("Optional automation description")),
			mcp.WithString("trigger_type", mcp.Description("pipeline_completed or manual")),
			mcp.WithString("trigger_config_json", mcp.Description("Optional trigger config JSON object")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			id := req.GetString("id", "")
			name := req.GetString("name", "")
			if id == "" || name == "" {
				return mcp.NewToolResultError("id and name are required"), nil
			}
			twin, err := m.dtSvc.GetDigitalTwin(id)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			triggerConfig := map[string]any{}
			if raw := req.GetString("trigger_config_json", ""); raw != "" {
				decoded, err := parseJSONMap(raw)
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				triggerConfig = decoded
			}
			triggerType := req.GetString("trigger_type", string(models.AutomationTriggerTypePipelineCompleted))
			automation, err := m.automationSvc.Create(&models.AutomationCreateRequest{
				ProjectID:     twin.ProjectID,
				Name:          name,
				Description:   req.GetString("description", ""),
				TargetType:    models.AutomationTargetTypeDigitalTwin,
				TargetID:      id,
				TriggerType:   models.AutomationTriggerType(triggerType),
				TriggerConfig: triggerConfig,
				ActionType:    models.AutomationActionTypeProcessTwin,
			})
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			data, _ := json.Marshal(automation)
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
