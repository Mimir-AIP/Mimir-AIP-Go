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
	// list_ml_models
	s.AddTool(
		mcp.NewTool("list_ml_models",
			mcp.WithDescription("List ML models, optionally filtered by project"),
			mcp.WithString("project_id",
				mcp.Description("Filter by project ID; omit to list all models"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			projectID := req.GetString("project_id", "")
			var (
				mlModels []*models.MLModel
				err      error
			)
			if projectID != "" {
				mlModels, err = m.mlSvc.ListProjectModels(projectID)
			} else {
				// List all by iterating; fall back to an empty list if no project given
				// The service only supports listing by project, so return helpful guidance
				data, _ := json.Marshal(map[string]string{
					"message": "Provide project_id to list models for a specific project",
				})
				return mcp.NewToolResultText(string(data)), nil
			}
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
			mcp.WithString("id",
				mcp.Required(),
				mcp.Description("ML model ID"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			id := req.GetString("id", "")
			if id == "" {
				return mcp.NewToolResultError("id is required"), nil
			}
			model, err := m.mlSvc.GetModel(id)
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
			mcp.WithString("project_id",
				mcp.Required(),
				mcp.Description("Project ID"),
			),
			mcp.WithString("ontology_id",
				mcp.Required(),
				mcp.Description("Ontology ID that defines the model's domain"),
			),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("Model name"),
			),
			mcp.WithString("type",
				mcp.Required(),
				mcp.Description("Model type: decision_tree, random_forest, regression, or neural_network"),
			),
			mcp.WithString("description",
				mcp.Description("Optional model description"),
			),
			mcp.WithString("config",
				mcp.Description(`Optional JSON training config e.g. {"train_test_split":0.8,"random_seed":42,"max_depth":5}`),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			projectID := req.GetString("project_id", "")
			ontologyID := req.GetString("ontology_id", "")
			name := req.GetString("name", "")
			modelType := req.GetString("type", "")
			if projectID == "" || ontologyID == "" || name == "" || modelType == "" {
				return mcp.NewToolResultError("project_id, ontology_id, name, and type are required"), nil
			}
			createReq := &models.ModelCreateRequest{
				ProjectID:   projectID,
				OntologyID:  ontologyID,
				Name:        name,
				Type:        models.ModelType(modelType),
				Description: req.GetString("description", ""),
			}
			if cfgStr := req.GetString("config", ""); cfgStr != "" {
				var cfg models.TrainingConfig
				if err := json.Unmarshal([]byte(cfgStr), &cfg); err != nil {
					return mcp.NewToolResultError("config must be valid JSON: " + err.Error()), nil
				}
				createReq.TrainingConfig = &cfg
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
			mcp.WithString("id",
				mcp.Required(),
				mcp.Description("ML model ID"),
			),
			mcp.WithString("name",
				mcp.Description("New model name"),
			),
			mcp.WithString("description",
				mcp.Description("New description"),
			),
			mcp.WithString("status",
				mcp.Description("New status: created, training, trained, failed, or deprecated"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			id := req.GetString("id", "")
			if id == "" {
				return mcp.NewToolResultError("id is required"), nil
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
			model, err := m.mlSvc.UpdateModel(id, updateReq)
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
			mcp.WithString("id",
				mcp.Required(),
				mcp.Description("ML model ID"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			id := req.GetString("id", "")
			if id == "" {
				return mcp.NewToolResultError("id is required"), nil
			}
			if err := m.mlSvc.DeleteModel(id); err != nil {
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
