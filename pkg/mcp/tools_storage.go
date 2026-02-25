package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

func registerStorageTools(s *server.MCPServer, m *MimirMCPServer) {
	// list_storage_configs
	s.AddTool(
		mcp.NewTool("list_storage_configs",
			mcp.WithDescription("List storage configurations, optionally filtered by project"),
			mcp.WithString("project_id",
				mcp.Description("Filter by project ID; omit to get an informational response"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			projectID := req.GetString("project_id", "")
			if projectID == "" {
				data, _ := json.Marshal(map[string]string{
					"message": "Provide project_id to list storage configs for a specific project",
				})
				return mcp.NewToolResultText(string(data)), nil
			}
			configs, err := m.storageSvc.GetProjectStorageConfigs(projectID)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			data, _ := json.Marshal(configs)
			return mcp.NewToolResultText(string(data)), nil
		},
	)

	// create_storage_config
	s.AddTool(
		mcp.NewTool("create_storage_config",
			mcp.WithDescription("Create a new storage configuration for a project"),
			mcp.WithString("project_id",
				mcp.Required(),
				mcp.Description("Project ID"),
			),
			mcp.WithString("type",
				mcp.Required(),
				mcp.Description("Storage plugin type: filesystem, postgresql, mysql, mongodb, s3, redis, elasticsearch, or neo4j"),
			),
			mcp.WithString("config",
				mcp.Required(),
				mcp.Description(`JSON object with plugin-specific config e.g. {"path":"/data"} for filesystem or {"connection_string":"postgres://..."} for postgresql`),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			projectID := req.GetString("project_id", "")
			pluginType := req.GetString("type", "")
			configStr := req.GetString("config", "")
			if projectID == "" || pluginType == "" || configStr == "" {
				return mcp.NewToolResultError("project_id, type, and config are required"), nil
			}
			var cfg map[string]interface{}
			if err := json.Unmarshal([]byte(configStr), &cfg); err != nil {
				return mcp.NewToolResultError("config must be a valid JSON object: " + err.Error()), nil
			}
			storageConfig, err := m.storageSvc.CreateStorageConfig(projectID, pluginType, cfg)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			data, _ := json.Marshal(storageConfig)
			return mcp.NewToolResultText(string(data)), nil
		},
	)

	// store_data
	s.AddTool(
		mcp.NewTool("store_data",
			mcp.WithDescription("Store one or more CIR (Common Internal Representation) records in a storage backend"),
			mcp.WithString("storage_id",
				mcp.Required(),
				mcp.Description("Storage config ID"),
			),
			mcp.WithString("data",
				mcp.Required(),
				mcp.Description(`JSON array of CIR objects. Minimal example: [{"version":"1.0","source":{"type":"api","uri":"manual","timestamp":"2024-01-01T00:00:00Z","format":"json"},"data":{"key":"value"},"metadata":{}}]`),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			storageID := req.GetString("storage_id", "")
			dataStr := req.GetString("data", "")
			if storageID == "" || dataStr == "" {
				return mcp.NewToolResultError("storage_id and data are required"), nil
			}
			var cirs []*models.CIR
			if err := json.Unmarshal([]byte(dataStr), &cirs); err != nil {
				return mcp.NewToolResultError("data must be a valid JSON array of CIR objects: " + err.Error()), nil
			}
			if len(cirs) == 0 {
				return mcp.NewToolResultError("data array must not be empty"), nil
			}

			// Set default timestamps if missing
			for _, c := range cirs {
				if c.Source.Timestamp.IsZero() {
					c.Source.Timestamp = time.Now().UTC()
				}
			}

			stored := 0
			var lastErr error
			for _, cir := range cirs {
				if _, err := m.storageSvc.Store(storageID, cir); err != nil {
					lastErr = err
				} else {
					stored++
				}
			}
			result := map[string]any{
				"stored": stored,
				"total":  len(cirs),
			}
			if lastErr != nil {
				result["last_error"] = lastErr.Error()
			}
			data, _ := json.Marshal(result)
			return mcp.NewToolResultText(string(data)), nil
		},
	)

	// retrieve_data
	s.AddTool(
		mcp.NewTool("retrieve_data",
			mcp.WithDescription("Retrieve CIR records from a storage backend"),
			mcp.WithString("storage_id",
				mcp.Required(),
				mcp.Description("Storage config ID"),
			),
			mcp.WithString("entity_type",
				mcp.Description("Filter records by entity type"),
			),
			mcp.WithString("limit",
				mcp.Description("Maximum number of records to return (default 100)"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			storageID := req.GetString("storage_id", "")
			if storageID == "" {
				return mcp.NewToolResultError("storage_id is required"), nil
			}
			query := &models.CIRQuery{
				EntityType: req.GetString("entity_type", ""),
				Limit:      req.GetInt("limit", 100),
			}
			cirs, err := m.storageSvc.Retrieve(storageID, query)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			data, _ := json.Marshal(map[string]any{
				"records": cirs,
				"count":   len(cirs),
			})
			return mcp.NewToolResultText(string(data)), nil
		},
	)

	// get_storage_config
	s.AddTool(
		mcp.NewTool("get_storage_config",
			mcp.WithDescription("Get a specific storage configuration by ID"),
			mcp.WithString("id",
				mcp.Required(),
				mcp.Description("Storage config ID"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			id := req.GetString("id", "")
			if id == "" {
				return mcp.NewToolResultError("id is required"), nil
			}
			config, err := m.storageSvc.GetStorageConfig(id)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			data, _ := json.Marshal(config)
			return mcp.NewToolResultText(string(data)), nil
		},
	)

	// update_storage_config
	s.AddTool(
		mcp.NewTool("update_storage_config",
			mcp.WithDescription("Update a storage configuration's plugin config or active state"),
			mcp.WithString("id",
				mcp.Required(),
				mcp.Description("Storage config ID"),
			),
			mcp.WithString("config",
				mcp.Description("JSON object with updated plugin-specific config"),
			),
			mcp.WithString("active",
				mcp.Description("Set active state: true or false"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			id := req.GetString("id", "")
			if id == "" {
				return mcp.NewToolResultError("id is required"), nil
			}
			var cfg map[string]interface{}
			if cfgStr := req.GetString("config", ""); cfgStr != "" {
				if err := json.Unmarshal([]byte(cfgStr), &cfg); err != nil {
					return mcp.NewToolResultError("config must be a valid JSON object: " + err.Error()), nil
				}
			}
			var active *bool
			if activeStr := req.GetString("active", ""); activeStr != "" {
				b := activeStr == "true"
				active = &b
			}
			if err := m.storageSvc.UpdateStorageConfig(id, cfg, active); err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultText(`{"success":true}`), nil
		},
	)

	// delete_storage_config
	s.AddTool(
		mcp.NewTool("delete_storage_config",
			mcp.WithDescription("Delete a storage configuration by ID"),
			mcp.WithString("id",
				mcp.Required(),
				mcp.Description("Storage config ID"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			id := req.GetString("id", "")
			if id == "" {
				return mcp.NewToolResultError("id is required"), nil
			}
			if err := m.storageSvc.DeleteStorageConfig(id); err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultText(`{"success":true}`), nil
		},
	)

	// update_data
	s.AddTool(
		mcp.NewTool("update_data",
			mcp.WithDescription("Update CIR records in a storage backend matching a query"),
			mcp.WithString("storage_id",
				mcp.Required(),
				mcp.Description("Storage config ID"),
			),
			mcp.WithString("query",
				mcp.Required(),
				mcp.Description(`JSON CIRQuery to select records e.g. {"entity_type":"Sensor","limit":100}`),
			),
			mcp.WithString("updates",
				mcp.Required(),
				mcp.Description(`JSON CIRUpdate with fields to set e.g. {"set_fields":{"status":"inactive"}}`),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			storageID := req.GetString("storage_id", "")
			queryStr := req.GetString("query", "")
			updatesStr := req.GetString("updates", "")
			if storageID == "" || queryStr == "" || updatesStr == "" {
				return mcp.NewToolResultError("storage_id, query, and updates are required"), nil
			}
			var query models.CIRQuery
			if err := json.Unmarshal([]byte(queryStr), &query); err != nil {
				return mcp.NewToolResultError("query must be valid JSON: " + err.Error()), nil
			}
			var updates models.CIRUpdate
			if err := json.Unmarshal([]byte(updatesStr), &updates); err != nil {
				return mcp.NewToolResultError("updates must be valid JSON: " + err.Error()), nil
			}
			result, err := m.storageSvc.Update(storageID, &query, &updates)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			data, _ := json.Marshal(result)
			return mcp.NewToolResultText(string(data)), nil
		},
	)

	// delete_data
	s.AddTool(
		mcp.NewTool("delete_data",
			mcp.WithDescription("Delete CIR records from a storage backend matching a query"),
			mcp.WithString("storage_id",
				mcp.Required(),
				mcp.Description("Storage config ID"),
			),
			mcp.WithString("query",
				mcp.Required(),
				mcp.Description(`JSON CIRQuery to select records for deletion e.g. {"entity_type":"Sensor"}`),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			storageID := req.GetString("storage_id", "")
			queryStr := req.GetString("query", "")
			if storageID == "" || queryStr == "" {
				return mcp.NewToolResultError("storage_id and query are required"), nil
			}
			var query models.CIRQuery
			if err := json.Unmarshal([]byte(queryStr), &query); err != nil {
				return mcp.NewToolResultError("query must be valid JSON: " + err.Error()), nil
			}
			result, err := m.storageSvc.Delete(storageID, &query)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			data, _ := json.Marshal(result)
			return mcp.NewToolResultText(string(data)), nil
		},
	)

	// storage_health
	s.AddTool(
		mcp.NewTool("storage_health",
			mcp.WithDescription("Check whether a storage backend is reachable and healthy"),
			mcp.WithString("storage_id",
				mcp.Required(),
				mcp.Description("Storage config ID"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			storageID := req.GetString("storage_id", "")
			if storageID == "" {
				return mcp.NewToolResultError("storage_id is required"), nil
			}
			healthy, err := m.storageSvc.HealthCheck(storageID)
			if err != nil {
				result, _ := json.Marshal(map[string]any{
					"storage_id": storageID,
					"healthy":    false,
					"error":      fmt.Sprintf("%v", err),
				})
				return mcp.NewToolResultText(string(result)), nil
			}
			result, _ := json.Marshal(map[string]any{
				"storage_id": storageID,
				"healthy":    healthy,
			})
			return mcp.NewToolResultText(string(result)), nil
		},
	)
}
