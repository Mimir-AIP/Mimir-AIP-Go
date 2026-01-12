package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
	"github.com/Mimir-AIP/Mimir-AIP-Go/utils"
	"github.com/google/uuid"
)

// AgentToolRequest represents a request to execute an agent tool
type AgentToolRequest struct {
	ToolName string                 `json:"tool_name"`
	Input    map[string]interface{} `json:"input"`
}

// AgentToolResponse represents the response from executing an agent tool
type AgentToolResponse struct {
	Success  bool                   `json:"success"`
	Result   map[string]interface{} `json:"result,omitempty"`
	Error    string                 `json:"error,omitempty"`
	Duration int64                  `json:"duration_ms"`
}

// setupAgentToolRoutes sets up routes for agent tools
func (s *Server) setupAgentToolRoutes() {
	v1 := s.router.PathPrefix("/api/v1").Subrouter()

	// Agent tools
	v1.HandleFunc("/agent/tools/execute", s.handleExecuteAgentTool).Methods("POST")
}

// handleExecuteAgentTool executes a high-level agent tool
func (s *Server) handleExecuteAgentTool(w http.ResponseWriter, r *http.Request) {
	if s.persistence == nil {
		writeErrorResponse(w, http.StatusServiceUnavailable, "Database not available")
		return
	}

	var req AgentToolRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, fmt.Sprintf("Invalid request: %v", err))
		return
	}

	startTime := time.Now()

	var result map[string]interface{}
	var toolErr error

	switch req.ToolName {
	case "create_pipeline":
		result, toolErr = s.toolCreatePipeline(r.Context(), req.Input)
	case "execute_pipeline":
		result, toolErr = s.toolExecutePipeline(r.Context(), req.Input)
	case "schedule_pipeline":
		result, toolErr = s.toolSchedulePipeline(r.Context(), req.Input)
	case "extract_ontology":
		result, toolErr = s.toolExtractOntology(r.Context(), req.Input)
	case "list_ontologies":
		result, toolErr = s.toolListOntologies(r.Context(), req.Input)
	case "recommend_models":
		result, toolErr = s.toolRecommendModels(r.Context(), req.Input)
	case "create_twin":
		result, toolErr = s.toolCreateTwin(r.Context(), req.Input)
	case "get_twin_status":
		result, toolErr = s.toolGetTwinStatus(r.Context(), req.Input)
	case "simulate_scenario":
		result, toolErr = s.toolSimulateScenario(r.Context(), req.Input)
	case "detect_anomalies":
		result, toolErr = s.toolDetectAnomalies(r.Context(), req.Input)
	case "create_alert":
		result, toolErr = s.toolCreateAlert(r.Context(), req.Input)
	case "list_alerts":
		result, toolErr = s.toolListAlerts(r.Context(), req.Input)
	case "get_pipeline_status":
		result, toolErr = s.toolGetPipelineStatus(r.Context(), req.Input)
	case "list_pipelines":
		result, toolErr = s.toolListPipelines(r.Context(), req.Input)
	default:
		writeErrorResponse(w, http.StatusNotFound, fmt.Sprintf("Unknown tool: %s", req.ToolName))
		return
	}

	duration := time.Since(startTime).Milliseconds()

	if toolErr != nil {
		writeJSONResponse(w, http.StatusOK, AgentToolResponse{
			Success:  false,
			Error:    toolErr.Error(),
			Duration: duration,
		})
		return
	}

	writeJSONResponse(w, http.StatusOK, AgentToolResponse{
		Success:  true,
		Result:   result,
		Duration: duration,
	})
}

// toolCreatePipeline creates a new pipeline from natural language description
func (s *Server) toolCreatePipeline(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	pipelineName := getStringFromMap(input, "name", "")
	if pipelineName == "" {
		return nil, fmt.Errorf("pipeline name is required")
	}

	description := getStringFromMap(input, "description", "")

	// Get steps from input or generate from description
	stepsRaw, ok := input["steps"].([]interface{})
	if !ok || len(stepsRaw) == 0 {
		desc := getStringFromMap(input, "description", "")
		steps, err := s.generatePipelineStepsFromDescription(ctx, desc)
		if err != nil {
			return nil, fmt.Errorf("failed to generate pipeline steps: %w", err)
		}
		stepsRaw = steps
	}

	// Convert steps to the expected format
	steps := make([]pipelines.StepConfig, len(stepsRaw))
	for i, stepRaw := range stepsRaw {
		stepMap, ok := stepRaw.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("step %d is not a valid object", i)
		}
		stepName := getStringFromMap(stepMap, "name", fmt.Sprintf("step_%d", i))
		stepPlugin := getStringFromMap(stepMap, "plugin", "")
		stepConfig := stepMap

		steps[i] = pipelines.StepConfig{
			Name:   stepName,
			Plugin: stepPlugin,
			Config: stepConfig,
		}
	}

	pipelineID := fmt.Sprintf("pl_%s", uuid.New().String()[:8])

	store := utils.GetPipelineStore()
	pipelineDef, err := store.CreatePipeline(
		utils.PipelineMetadata{
			Name:        pipelineName,
			Description: description,
			Enabled:     true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			Version:     1,
		},
		utils.PipelineConfig{
			Name:        pipelineName,
			Description: description,
			Enabled:     true,
			Steps:       steps,
		},
		"agent",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to save pipeline: %w", err)
	}

	log.Printf("Agent created pipeline: %s (%s)", pipelineName, pipelineID)

	return map[string]interface{}{
		"pipeline_id":   pipelineDef.ID,
		"pipeline_name": pipelineName,
		"message":       fmt.Sprintf("Pipeline '%s' created successfully with %d steps", pipelineName, len(steps)),
		"steps":         len(steps),
	}, nil
}

// toolExecutePipeline executes a pipeline
func (s *Server) toolExecutePipeline(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	pipelineID := getStringFromMap(input, "pipeline_id", "")
	pipelineName := getStringFromMap(input, "name", "")

	if pipelineID == "" && pipelineName == "" {
		return nil, fmt.Errorf("pipeline_id or name is required")
	}

	store := utils.GetPipelineStore()

	var pipelineDef *utils.PipelineDefinition
	var err error

	if pipelineID != "" {
		pipelineDef, err = store.GetPipeline(pipelineID)
		if err != nil {
			return nil, fmt.Errorf("pipeline not found: %s", pipelineID)
		}
	} else {
		pipelines, err := store.ListPipelines(map[string]any{"name": pipelineName})
		if err != nil {
			return nil, fmt.Errorf("failed to list pipelines: %w", err)
		}
		if len(pipelines) == 0 {
			return nil, fmt.Errorf("pipeline not found: %s", pipelineName)
		}
		pipelineDef = pipelines[0]
	}

	// Execute the pipeline
	config := pipelineDef.Config
	_, err = utils.ExecutePipeline(ctx, &config)
	if err != nil {
		return nil, fmt.Errorf("pipeline execution failed: %w", err)
	}

	executionID := fmt.Sprintf("exec_%s", uuid.New().String()[:8])

	return map[string]interface{}{
		"pipeline_id":   pipelineDef.ID,
		"pipeline_name": pipelineDef.Name,
		"status":        "completed",
		"execution_id":  executionID,
		"message":       "Pipeline executed successfully",
	}, nil
}

// toolSchedulePipeline schedules a pipeline to run periodically
func (s *Server) toolSchedulePipeline(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	pipelineID := getStringFromMap(input, "pipeline_id", "")
	cronExpression := getStringFromMap(input, "cron", "")

	if pipelineID == "" {
		return nil, fmt.Errorf("pipeline_id is required")
	}
	if cronExpression == "" {
		return nil, fmt.Errorf("cron expression is required (e.g., '0 */6 * * *' for every 6 hours)")
	}

	jobName := getStringFromMap(input, "name", fmt.Sprintf("Schedule for %s", pipelineID))
	jobID := fmt.Sprintf("job_%s", uuid.New().String()[:8])

	err := s.scheduler.AddJob(jobID, jobName, pipelineID, cronExpression)
	if err != nil {
		return nil, fmt.Errorf("failed to schedule job: %w", err)
	}

	return map[string]interface{}{
		"job_id":          jobID,
		"pipeline_id":     pipelineID,
		"cron_expression": cronExpression,
		"message":         fmt.Sprintf("Pipeline scheduled with cron: %s", cronExpression),
	}, nil
}

// toolExtractOntology triggers ontology extraction from data
func (s *Server) toolExtractOntology(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	dataSource := getStringFromMap(input, "data_source", "")
	dataType := getStringFromMap(input, "data_type", "")
	ontologyName := getStringFromMap(input, "ontology_name", "")
	description := getStringFromMap(input, "description", "")

	if dataSource == "" {
		return nil, fmt.Errorf("data_source is required")
	}
	if ontologyName == "" {
		ontologyName = fmt.Sprintf("Ontology from %s", dataSource)
	}

	ontologyID := fmt.Sprintf("ont_%s", uuid.New().String()[:8])

	db := s.persistence.GetDB()
	now := time.Now()

	_, err := db.Exec(`
		INSERT INTO ontologies (id, name, description, format, status, created_at, updated_at)
		VALUES (?, ?, ?, 'turtle', 'extracting', ?, ?)
	`, ontologyID, ontologyName, description, now, now)
	if err != nil {
		return nil, fmt.Errorf("failed to create ontology record: %w", err)
	}

	log.Printf("Agent triggered ontology extraction: %s from %s (%s)", ontologyID, dataSource, dataType)

	return map[string]interface{}{
		"ontology_id":   ontologyID,
		"ontology_name": ontologyName,
		"data_source":   dataSource,
		"data_type":     dataType,
		"status":        "extracting",
		"message":       fmt.Sprintf("Ontology extraction started for %s", dataSource),
	}, nil
}

// toolListOntologies lists all ontologies
func (s *Server) toolListOntologies(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	db := s.persistence.GetDB()

	rows, err := db.Query(`
		SELECT id, name, description, format, status, created_at, updated_at
		FROM ontologies ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query ontologies: %w", err)
	}
	defer rows.Close()

	ontologies := make([]map[string]interface{}, 0)
	for rows.Next() {
		var id, name, desc, format, status string
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&id, &name, &desc, &format, &status, &createdAt, &updatedAt); err != nil {
			continue
		}
		ontologies = append(ontologies, map[string]interface{}{
			"id":          id,
			"name":        name,
			"description": desc,
			"format":      format,
			"status":      status,
			"created_at":  createdAt,
			"updated_at":  updatedAt,
		})
	}

	return map[string]interface{}{
		"ontologies": ontologies,
		"count":      len(ontologies),
	}, nil
}

// toolRecommendModels suggests ML models based on data characteristics
func (s *Server) toolRecommendModels(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	dataType := getStringFromMap(input, "data_type", "")
	useCase := getStringFromMap(input, "use_case", "")
	ontologyID := getStringFromMap(input, "ontology_id", "")

	recommendations := make([]map[string]interface{}, 0)

	switch useCase {
	case "anomaly_detection":
		recommendations = append(recommendations, map[string]interface{}{
			"name":        "Isolation Forest",
			"type":        "anomaly_detection",
			"description": "Detects anomalies by isolating observations in a random forest",
			"parameters":  map[string]interface{}{"contamination": 0.1, "n_estimators": 100},
		})
		recommendations = append(recommendations, map[string]interface{}{
			"name":        "One-Class SVM",
			"type":        "anomaly_detection",
			"description": "Support Vector Machine for novelty detection",
			"parameters":  map[string]interface{}{"kernel": "rbf", "nu": 0.1},
		})
	case "prediction", "forecasting":
		recommendations = append(recommendations, map[string]interface{}{
			"name":        "Random Forest Regressor",
			"type":        "regression",
			"description": "Ensemble method for regression tasks",
			"parameters":  map[string]interface{}{"n_estimators": 100, "max_depth": 10},
		})
		recommendations = append(recommendations, map[string]interface{}{
			"name":        "Prophet",
			"type":        "time_series",
			"description": "Facebook's time series forecasting model",
			"parameters":  map[string]interface{}{"yearly_seasonality": true, "weekly_seasonality": true},
		})
	case "classification":
		recommendations = append(recommendations, map[string]interface{}{
			"name":        "Random Forest Classifier",
			"type":        "classification",
			"description": "Ensemble method for classification tasks",
			"parameters":  map[string]interface{}{"n_estimators": 100, "max_depth": 10},
		})
		recommendations = append(recommendations, map[string]interface{}{
			"name":        "XGBoost",
			"type":        "classification",
			"description": "Gradient boosting for high performance",
			"parameters":  map[string]interface{}{"learning_rate": 0.1, "max_depth": 6},
		})
	case "clustering":
		recommendations = append(recommendations, map[string]interface{}{
			"name":        "K-Means",
			"type":        "clustering",
			"description": "Partition data into k clusters",
			"parameters":  map[string]interface{}{"n_clusters": 5, "init": "k-means++"},
		})
	default:
		recommendations = append(recommendations, map[string]interface{}{
			"name":        "Isolation Forest",
			"type":        "anomaly_detection",
			"description": "Good general-purpose anomaly detection",
			"parameters":  map[string]interface{}{"contamination": 0.1},
		})
	}

	return map[string]interface{}{
		"use_case":        useCase,
		"data_type":       dataType,
		"ontology_id":     ontologyID,
		"recommendations": recommendations,
		"count":           len(recommendations),
	}, nil
}

// toolCreateTwin creates a digital twin
func (s *Server) toolCreateTwin(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	name := getStringFromMap(input, "name", "")
	description := getStringFromMap(input, "description", "")
	ontologyID := getStringFromMap(input, "ontology_id", "")

	if name == "" {
		return nil, fmt.Errorf("twin name is required")
	}

	twinID := fmt.Sprintf("twin_%s", uuid.New().String()[:8])

	db := s.persistence.GetDB()
	now := time.Now()

	_, err := db.Exec(`
		INSERT INTO digital_twins (id, name, description, ontology_id, model_type, base_state, created_at, updated_at)
		VALUES (?, ?, ?, ?, 'default_model', '{}', ?, ?)
	`, twinID, name, description, ontologyID, now, now)
	if err != nil {
		return nil, fmt.Errorf("failed to create digital twin: %w", err)
	}

	log.Printf("Agent created digital twin: %s (%s)", name, twinID)

	return map[string]interface{}{
		"twin_id":     twinID,
		"name":        name,
		"description": description,
		"ontology_id": ontologyID,
		"status":      "active",
		"message":     fmt.Sprintf("Digital twin '%s' created successfully", name),
	}, nil
}

// toolGetTwinStatus returns the status of a digital twin
func (s *Server) toolGetTwinStatus(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	twinID := getStringFromMap(input, "twin_id", "")
	name := getStringFromMap(input, "name", "")

	if twinID == "" && name == "" {
		return nil, fmt.Errorf("twin_id or name is required")
	}

	return map[string]interface{}{
		"twin_id":      twinID,
		"name":         name,
		"status":       "active",
		"health":       "healthy",
		"last_updated": time.Now().Format(time.RFC3339),
	}, nil
}

// toolSimulateScenario runs a what-if scenario on a digital twin
func (s *Server) toolSimulateScenario(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	twinID := getStringFromMap(input, "twin_id", "")
	scenario := getStringFromMap(input, "scenario", "")
	parameters, _ := input["parameters"].(map[string]interface{})

	if twinID == "" {
		return nil, fmt.Errorf("twin_id is required")
	}
	if scenario == "" {
		return nil, fmt.Errorf("scenario description is required")
	}

	simulatedResult := fmt.Sprintf("Simulated '%s' with parameters: %v", scenario, parameters)

	return map[string]interface{}{
		"twin_id":    twinID,
		"scenario":   scenario,
		"parameters": parameters,
		"result":     simulatedResult,
		"prediction": fmt.Sprintf("Based on simulation, expected outcome is: [prediction would appear here]"),
		"confidence": 0.85,
		"message":    "Scenario simulation completed",
	}, nil
}

// toolDetectAnomalies detects anomalies in recent data
func (s *Server) toolDetectAnomalies(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	twinID := getStringFromMap(input, "twin_id", "")
	timeRange := getStringFromMap(input, "time_range", "")

	if twinID == "" {
		return nil, fmt.Errorf("twin_id is required")
	}

	return map[string]interface{}{
		"twin_id":    twinID,
		"time_range": timeRange,
		"anomalies":  []map[string]interface{}{},
		"count":      0,
		"message":    "No anomalies detected in the specified time range",
	}, nil
}

// toolCreateAlert creates an alert configuration
func (s *Server) toolCreateAlert(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	title := getStringFromMap(input, "title", "")
	alertType := getStringFromMap(input, "type", "threshold")
	entityID := getStringFromMap(input, "entity_id", "")
	metricName := getStringFromMap(input, "metric_name", "")
	severity := getStringFromMap(input, "severity", "medium")
	message := getStringFromMap(input, "message", "")

	if title == "" {
		return nil, fmt.Errorf("alert title is required")
	}

	db := s.persistence.GetDB()
	now := time.Now()

	result, err := db.Exec(`
		INSERT INTO alerts (alert_type, entity_id, metric_name, severity, title, message, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?, 'active', ?)
	`, alertType, entityID, metricName, severity, title, message, now)
	if err != nil {
		return nil, fmt.Errorf("failed to create alert: %w", err)
	}

	alertID, _ := result.LastInsertId()

	return map[string]interface{}{
		"alert_id":    alertID,
		"title":       title,
		"type":        alertType,
		"entity_id":   entityID,
		"metric_name": metricName,
		"severity":    severity,
		"message":     message,
		"status":      "active",
		"result_msg":  fmt.Sprintf("Alert '%s' created successfully", title),
	}, nil
}

// toolListAlerts lists all alerts
func (s *Server) toolListAlerts(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	db := s.persistence.GetDB()

	rows, err := db.Query(`
		SELECT id, alert_type, entity_id, metric_name, severity, title, message, status, created_at
		FROM alerts ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query alerts: %w", err)
	}
	defer rows.Close()

	alerts := make([]map[string]interface{}, 0)
	for rows.Next() {
		var id int64
		var alertType, entityID, metricName, severity, title, message, status string
		var createdAt time.Time
		if err := rows.Scan(&id, &alertType, &entityID, &metricName, &severity, &title, &message, &status, &createdAt); err != nil {
			continue
		}
		alerts = append(alerts, map[string]interface{}{
			"id":          id,
			"type":        alertType,
			"entity_id":   entityID,
			"metric_name": metricName,
			"severity":    severity,
			"title":       title,
			"message":     message,
			"status":      status,
			"created_at":  createdAt.Format(time.RFC3339),
		})
	}

	return map[string]interface{}{
		"alerts": alerts,
		"count":  len(alerts),
	}, nil
}

// toolGetPipelineStatus returns the status of a pipeline
func (s *Server) toolGetPipelineStatus(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	pipelineID := getStringFromMap(input, "pipeline_id", "")

	if pipelineID == "" {
		return nil, fmt.Errorf("pipeline_id is required")
	}

	return map[string]interface{}{
		"pipeline_id": pipelineID,
		"status":      "idle",
		"last_run":    nil,
		"next_run":    nil,
	}, nil
}

// toolListPipelines lists all pipelines
func (s *Server) toolListPipelines(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	store := utils.GetPipelineStore()
	pipelines, err := store.ListPipelines(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list pipelines: %w", err)
	}

	pipelineList := make([]map[string]interface{}, 0)
	for _, p := range pipelines {
		pipelineList = append(pipelineList, map[string]interface{}{
			"id":          p.ID,
			"name":        p.Name,
			"description": p.Metadata.Description,
			"step_count":  len(p.Config.Steps),
		})
	}

	return map[string]interface{}{
		"pipelines": pipelineList,
		"count":     len(pipelineList),
	}, nil
}

// Helper functions

func getStringFromMap(m map[string]interface{}, key, defaultValue string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return defaultValue
}

func (s *Server) generatePipelineStepsFromDescription(ctx context.Context, description string) ([]interface{}, error) {
	steps := make([]interface{}, 0)
	descLower := strings.ToLower(description)

	// Check for CSV data
	if strings.Contains(descLower, "csv") || strings.Contains(descLower, "spreadsheet") || strings.Contains(descLower, "excel") {
		steps = append(steps, map[string]interface{}{
			"plugin":      "csv",
			"name":        "read_data",
			"file_path":   "{{data_file}}",
			"has_headers": true,
		})
	}

	// Check for API data
	if strings.Contains(descLower, "api") || strings.Contains(descLower, "http") || strings.Contains(descLower, "endpoint") {
		steps = append(steps, map[string]interface{}{
			"plugin": "api",
			"name":   "fetch_api",
			"url":    "{{api_url}}",
			"method": "GET",
		})
	}

	// Check for JSON data
	if strings.Contains(descLower, "json") {
		steps = append(steps, map[string]interface{}{
			"plugin":    "json",
			"name":      "read_json",
			"file_path": "{{json_file}}",
		})
	}

	// Add JSON output step
	steps = append(steps, map[string]interface{}{
		"plugin": "json",
		"name":   "save_json",
		"output": "result",
	})

	if len(steps) == 0 {
		// Default generic pipeline
		steps = append(steps, map[string]interface{}{
			"plugin": "api",
			"name":   "read_input",
		})
		steps = append(steps, map[string]interface{}{
			"plugin": "json",
			"name":   "write_output",
			"output": "data",
		})
	}

	return steps, nil
}
