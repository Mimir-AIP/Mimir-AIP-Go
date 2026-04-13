package mcp

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"time"
)

func registerMLModelTools(s *server.MCPServer, m *MimirMCPServer) {
	// list_ml_providers
	s.AddTool(
		mcp.NewTool("list_ml_providers",
			mcp.WithDescription("List builtin and plugin-backed ML providers"),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			providers, err := m.mlSvc.ListProviderMetadata()
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			data, _ := json.Marshal(providers)
			return mcp.NewToolResultText(string(data)), nil
		},
	)

	// get_ml_provider
	s.AddTool(
		mcp.NewTool("get_ml_provider",
			mcp.WithDescription("Get metadata for a specific ML provider"),
			mcp.WithString("name", mcp.Required(), mcp.Description("Provider name")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			name := req.GetString("name", "")
			if name == "" {
				return mcp.NewToolResultError("name is required"), nil
			}
			provider, err := m.mlSvc.GetProviderMetadata(name)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			data, _ := json.Marshal(provider)
			return mcp.NewToolResultText(string(data)), nil
		},
	)

	// list_ml_models
	s.AddTool(
		mcp.NewTool("list_ml_models",
			mcp.WithDescription("List ML models for a specific project"),
			mcp.WithString("project_id",
				mcp.Required(),
				mcp.Description("Project ID"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			projectID := req.GetString("project_id", "")
			if projectID == "" {
				return mcp.NewToolResultError("project_id is required"), nil
			}
			mlModels, err := m.mlSvc.ListProjectModels(projectID)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			data, _ := json.Marshal(mlModels)
			return mcp.NewToolResultText(string(data)), nil
		},
	)

	// get_ml_model
	s.AddTool(
		mcp.NewTool("get_ml_model",
			mcp.WithDescription("Get details of a specific ML model by ID"),
			mcp.WithString("project_id", mcp.Required(), mcp.Description("Owning project ID")),
			mcp.WithString("id", mcp.Required(), mcp.Description("ML model ID")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			projectID := req.GetString("project_id", "")
			id := req.GetString("id", "")
			if projectID == "" || id == "" {
				return mcp.NewToolResultError("project_id and id are required"), nil
			}
			model, err := m.mlSvc.GetModelForProject(projectID, id)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			data, _ := json.Marshal(model)
			return mcp.NewToolResultText(string(data)), nil
		},
	)

	// create_ml_model
	s.AddTool(
		mcp.NewTool("create_ml_model",
			mcp.WithDescription("Create a new ML model definition for a project"),
			mcp.WithString("project_id", mcp.Required(), mcp.Description("Project ID")),
			mcp.WithString("ontology_id", mcp.Required(), mcp.Description("Ontology ID that defines the model's domain")),
			mcp.WithString("name", mcp.Required(), mcp.Description("Model name")),
			mcp.WithString("type", mcp.Description("Builtin model type: decision_tree, random_forest, regression, or neural_network")),
			mcp.WithString("provider", mcp.Description("Optional ML provider name (defaults to builtin)")),
			mcp.WithString("provider_model", mcp.Description("Provider-specific model identifier")),
			mcp.WithString("provider_config_json", mcp.Description("Optional provider-specific config as JSON object")),
			mcp.WithString("description", mcp.Description("Optional model description")),
			mcp.WithString("config", mcp.Description(`Optional JSON training config e.g. {"train_test_split":0.8,"random_seed":42,"max_depth":5}`)),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			projectID := req.GetString("project_id", "")
			ontologyID := req.GetString("ontology_id", "")
			name := req.GetString("name", "")
			modelType := req.GetString("type", "")
			provider := req.GetString("provider", "")
			providerModel := req.GetString("provider_model", "")
			if projectID == "" || ontologyID == "" || name == "" || (modelType == "" && provider == "") {
				return mcp.NewToolResultError("project_id, ontology_id, name, and either type or provider are required"), nil
			}
			createReq := &models.ModelCreateRequest{ProjectID: projectID, OntologyID: ontologyID, Name: name, Type: models.ModelType(modelType), Provider: provider, ProviderModel: providerModel, Description: req.GetString("description", "")}
			if cfgStr := req.GetString("config", ""); cfgStr != "" {
				var cfg models.TrainingConfig
				if err := json.Unmarshal([]byte(cfgStr), &cfg); err != nil {
					return mcp.NewToolResultError("config must be valid JSON: " + err.Error()), nil
				}
				createReq.TrainingConfig = &cfg
			}
			if providerCfgStr := req.GetString("provider_config_json", ""); providerCfgStr != "" {
				providerCfg, err := parseJSONMap(providerCfgStr)
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				createReq.ProviderConfig = providerCfg
			}
			model, err := m.mlSvc.CreateModel(createReq)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			data, _ := json.Marshal(model)
			return mcp.NewToolResultText(string(data)), nil
		},
	)

	// train_ml_model
	s.AddTool(
		mcp.NewTool("train_ml_model",
			mcp.WithDescription("Start asynchronous training for an ML model; returns the model with updated status"),
			mcp.WithString("model_id",
				mcp.Required(),
				mcp.Description("ML model ID to train"),
			),
			mcp.WithString("storage_ids",
				mcp.Description("Comma-separated list of storage config IDs to use as training data"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			modelID := req.GetString("model_id", "")
			if modelID == "" {
				return mcp.NewToolResultError("model_id is required"), nil
			}
			trainReq := &models.ModelTrainingRequest{ModelID: modelID}
			if ids := req.GetString("storage_ids", ""); ids != "" {
				trainReq.StorageIDs = splitCSV(ids)
			}
			model, err := m.mlSvc.StartTraining(trainReq)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			data, _ := json.Marshal(model)
			return mcp.NewToolResultText(string(data)), nil
		},
	)

	// run_inference
	s.AddTool(
		mcp.NewTool("run_inference",
			mcp.WithDescription("Enqueue an ML inference job; returns a work task ID to track progress"),
			mcp.WithString("model_id",
				mcp.Required(),
				mcp.Description("Trained ML model ID"),
			),
			mcp.WithString("storage_id",
				mcp.Required(),
				mcp.Description("Storage config ID containing the data to run inference on"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			modelID := req.GetString("model_id", "")
			storageID := req.GetString("storage_id", "")
			if modelID == "" || storageID == "" {
				return mcp.NewToolResultError("model_id and storage_id are required"), nil
			}
			model, err := m.mlSvc.GetModel(modelID)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			taskID := uuid.New().String()
			task := &models.WorkTask{
				ID:          taskID,
				Type:        models.WorkTaskTypeMLInference,
				ProjectID:   model.ProjectID,
				Priority:    5,
				Status:      models.WorkTaskStatusQueued,
				SubmittedAt: time.Now().UTC(),
				TaskSpec: models.TaskSpec{
					ModelID:   modelID,
					ProjectID: model.ProjectID,
					Parameters: map[string]any{
						"model_id":   modelID,
						"storage_id": storageID,
					},
				},
				DataAccess: models.DataAccess{
					InputDatasets: []string{storageID},
				},
			}
			if err := m.queue.Enqueue(task); err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			data, _ := json.Marshal(map[string]string{
				"task_id":    taskID,
				"model_id":   modelID,
				"storage_id": storageID,
				"status":     "queued",
			})
			return mcp.NewToolResultText(string(data)), nil
		},
	)

	// update_ml_model
	s.AddTool(
		mcp.NewTool("update_ml_model",
			mcp.WithDescription("Update an existing ML model's metadata or status"),
			mcp.WithString("project_id", mcp.Required(), mcp.Description("Owning project ID")),
			mcp.WithString("id", mcp.Required(), mcp.Description("ML model ID")),
			mcp.WithString("name", mcp.Description("New model name")),
			mcp.WithString("description", mcp.Description("New description")),
			mcp.WithString("status", mcp.Description("New status: draft, training, trained, failed, degraded, deprecated, or archived")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			projectID := req.GetString("project_id", "")
			id := req.GetString("id", "")
			if projectID == "" || id == "" {
				return mcp.NewToolResultError("project_id and id are required"), nil
			}
			updateReq := &models.ModelUpdateRequest{}
			if name := req.GetString("name", ""); name != "" {
				updateReq.Name = &name
			}
			if desc := req.GetString("description", ""); desc != "" {
				updateReq.Description = &desc
			}
			if st := req.GetString("status", ""); st != "" {
				ms := models.ModelStatus(st)
				updateReq.Status = &ms
			}
			model, err := m.mlSvc.UpdateModelForProject(projectID, id, updateReq)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			data, _ := json.Marshal(model)
			return mcp.NewToolResultText(string(data)), nil
		},
	)

	// delete_ml_model
	s.AddTool(
		mcp.NewTool("delete_ml_model",
			mcp.WithDescription("Delete an ML model by ID"),
			mcp.WithString("project_id", mcp.Required(), mcp.Description("Owning project ID")),
			mcp.WithString("id", mcp.Required(), mcp.Description("ML model ID")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			projectID := req.GetString("project_id", "")
			id := req.GetString("id", "")
			if projectID == "" || id == "" {
				return mcp.NewToolResultError("project_id and id are required"), nil
			}
			if err := m.mlSvc.DeleteModelForProject(projectID, id); err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultText(`{"success":true}`), nil
		},
	)

	// recommend_model
	s.AddTool(
		mcp.NewTool("recommend_model",
			mcp.WithDescription("Recommend the best ML model type for a project based on its ontology and data"),
			mcp.WithString("project_id",
				mcp.Required(),
				mcp.Description("Project ID"),
			),
			mcp.WithString("ontology_id",
				mcp.Required(),
				mcp.Description("Ontology ID describing the data domain"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			projectID := req.GetString("project_id", "")
			ontologyID := req.GetString("ontology_id", "")
			if projectID == "" || ontologyID == "" {
				return mcp.NewToolResultError("project_id and ontology_id are required"), nil
			}
			recommendation, err := m.mlSvc.RecommendModelType(projectID, ontologyID)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			data, _ := json.Marshal(recommendation)
			return mcp.NewToolResultText(string(data)), nil
		},
	)
}
