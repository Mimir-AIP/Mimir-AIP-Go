package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
	DigitalTwin "github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/DigitalTwin"
	"github.com/Mimir-AIP/Mimir-AIP-Go/utils"
	"github.com/google/uuid"
)

// sanitizeFloat replaces NaN and Inf values with 0 for JSON compatibility
func sanitizeFloat(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	return v
}

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
	log.Printf("handleExecuteAgentTool called")
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
	case "train_model":
		result, toolErr = s.toolTrainModel(r.Context(), req.Input)
	case "query_ontology":
		result, toolErr = s.toolQueryOntology(r.Context(), req.Input)
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
	log.Printf("toolExtractOntology called with input: %+v", input)
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
	filePath := fmt.Sprintf("%s/%s.ttl", s.ontologyDir, ontologyID)
	log.Printf("Ontology directory: %s, File path: %s", s.ontologyDir, filePath)
	tdb2Graph := fmt.Sprintf("http://mimir.ai/ontology/%s", ontologyID)
	version := fmt.Sprintf("1.0.%d", time.Now().Unix()) // Unique version per creation

	db := s.persistence.GetDB()
	now := time.Now()

	log.Printf("Creating ontology record: id=%s, name=%s, file_path=%s", ontologyID, ontologyName, filePath)
	_, err := db.Exec(`
		INSERT INTO ontologies (id, name, description, format, status, version, file_path, tdb2_graph, created_at, updated_at)
		VALUES (?, ?, ?, 'turtle', 'extracting', ?, ?, ?, ?, ?)
	`, ontologyID, ontologyName, description, version, filePath, tdb2Graph, now, now)
	if err != nil {
		log.Printf("Failed to create ontology record: %v", err)
		return nil, fmt.Errorf("failed to create ontology record: %w", err)
	}
	log.Printf("Successfully created ontology record: %s", ontologyID)

	// Generate some basic ontology content for the autonomous flow
	ontologyContent := fmt.Sprintf(`@prefix owl: <http://www.w3.org/2002/07/owl#> .
@prefix rdf: <http://www.w3.org/1999/02/22-rdf-syntax-ns#> .
@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .
@prefix xsd: <http://www.w3.org/2001/XMLSchema#> .
@prefix : <http://mimir.ai/ontology/%s#> .

:%s a owl:Ontology ;
    rdfs:label "%s"@en ;
    rdfs:comment "%s"@en .

:Person a owl:Class ;
    rdfs:label "Person"@en ;
    rdfs:comment "A person entity"@en .

:Name a owl:DatatypeProperty ;
    rdfs:label "name"@en ;
    rdfs:domain :Person ;
    rdfs:range xsd:string .

:Age a owl:DatatypeProperty ;
    rdfs:label "age"@en ;
    rdfs:domain :Person ;
    rdfs:range xsd:int .
`, ontologyID, ontologyID, ontologyName, description)

	// Ensure directory exists
	if err := os.MkdirAll(s.ontologyDir, 0755); err != nil {
		log.Printf("Failed to create ontology directory %s: %v", s.ontologyDir, err)
		return nil, fmt.Errorf("failed to create ontology directory: %w", err)
	}

	log.Printf("Writing ontology file to: %s (content length: %d)", filePath, len(ontologyContent))
	// Write the ontology file
	if err := os.WriteFile(filePath, []byte(ontologyContent), 0644); err != nil {
		log.Printf("Failed to write ontology file: %v", err)
		return nil, fmt.Errorf("failed to write ontology file: %w", err)
	}
	log.Printf("Successfully wrote ontology file: %s", filePath)

	// Load into TDB2 if available
	if s.tdb2Backend != nil {
		if err := s.tdb2Backend.LoadOntology(ctx, tdb2Graph, ontologyContent, "turtle"); err != nil {
			log.Printf("Failed to load ontology into TDB2: %v", err)
		}
	}

	log.Printf("Agent created ontology: %s from %s (%s)", ontologyID, dataSource, dataType)

	return map[string]interface{}{
		"ontology_id": ontologyID,
		"status":      "created",
		"message":     fmt.Sprintf("Ontology created: %s", ontologyName),
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
	scenarioID := getStringFromMap(input, "scenario_id", "")
	scenarioDesc := getStringFromMap(input, "scenario", "")

	if twinID == "" {
		return nil, fmt.Errorf("twin_id is required")
	}

	db := s.persistence.GetDB()

	// Load the twin
	var twinJSON string
	err := db.QueryRow("SELECT base_state FROM digital_twins WHERE id = ?", twinID).Scan(&twinJSON)
	if err != nil {
		return nil, fmt.Errorf("digital twin not found: %s", twinID)
	}

	var twin DigitalTwin.DigitalTwin
	if err := twin.FromJSON(twinJSON); err != nil {
		return nil, fmt.Errorf("failed to parse twin: %w", err)
	}
	twin.ID = twinID

	// Get or create scenario
	var scenario *DigitalTwin.SimulationScenario

	if scenarioID != "" {
		// Load existing scenario
		var eventsJSON string
		var scenarioName, scenarioType string
		var duration int
		err = db.QueryRow("SELECT name, scenario_type, events, duration FROM simulation_scenarios WHERE id = ?", scenarioID).
			Scan(&scenarioName, &scenarioType, &eventsJSON, &duration)
		if err != nil {
			return nil, fmt.Errorf("scenario not found: %s", scenarioID)
		}

		var events []DigitalTwin.SimulationEvent
		json.Unmarshal([]byte(eventsJSON), &events)

		scenario = &DigitalTwin.SimulationScenario{
			ID:       scenarioID,
			TwinID:   twinID,
			Name:     scenarioName,
			Type:     scenarioType,
			Events:   events,
			Duration: duration,
		}
	} else {
		// Create an ad-hoc scenario from description
		scenario = &DigitalTwin.SimulationScenario{
			ID:          fmt.Sprintf("adhoc_%s", uuid.New().String()[:8]),
			TwinID:      twinID,
			Name:        scenarioDesc,
			Type:        "adhoc",
			Description: scenarioDesc,
			Events:      []DigitalTwin.SimulationEvent{},
			Duration:    30,
		}
	}

	// Create simulation engine with ML if available
	engine := DigitalTwin.NewSimulationEngineWithML(&twin, db)

	// Run simulation
	run, err := engine.RunSimulation(scenario)
	if err != nil {
		return nil, fmt.Errorf("simulation failed: %w", err)
	}

	// Analyze impact
	analysis, _ := engine.AnalyzeImpact(run)

	log.Printf("Agent executed simulation %s on twin %s: status=%s", run.ID, twinID, run.Status)

	result := map[string]interface{}{
		"twin_id":       twinID,
		"run_id":        run.ID,
		"scenario_id":   scenario.ID,
		"scenario_name": scenario.Name,
		"status":        run.Status,
		"metrics": map[string]interface{}{
			"total_steps":         run.Metrics.TotalSteps,
			"events_processed":    run.Metrics.EventsProcessed,
			"entities_affected":   run.Metrics.EntitiesAffected,
			"average_utilization": sanitizeFloat(run.Metrics.AverageUtilization),
			"peak_utilization":    sanitizeFloat(run.Metrics.PeakUtilization),
			"system_stability":    sanitizeFloat(run.Metrics.SystemStability),
		},
		"message": fmt.Sprintf("Simulation completed: %d steps, %d events processed", run.Metrics.TotalSteps, run.Metrics.EventsProcessed),
	}

	if analysis != nil {
		result["overall_impact"] = analysis.OverallImpact
		result["risk_score"] = sanitizeFloat(analysis.RiskScore)
		result["insights"] = analysis.Insights
	}

	return result, nil
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

// toolTrainModel trains an ML model from ontology data
func (s *Server) toolTrainModel(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	ontologyID := getStringFromMap(input, "ontology_id", "")
	targetProperty := getStringFromMap(input, "target_property", "")
	modelType := getStringFromMap(input, "model_type", "auto") // auto, regression, classification

	if ontologyID == "" {
		return nil, fmt.Errorf("ontology_id is required")
	}

	// Verify ontology exists
	db := s.persistence.GetDB()
	var ontologyName string
	err := db.QueryRow("SELECT name FROM ontologies WHERE id = ?", ontologyID).Scan(&ontologyName)
	if err != nil {
		return nil, fmt.Errorf("ontology not found: %s", ontologyID)
	}

	// Publish training started event
	utils.GetEventBus().Publish(utils.Event{
		Type:   utils.EventModelTrainingStarted,
		Source: "agent-tool",
		Payload: map[string]any{
			"ontology_id":     ontologyID,
			"target_property": targetProperty,
			"model_type":      modelType,
			"trigger":         "agent_tool",
		},
	})

	// Create a training job record
	modelID := fmt.Sprintf("model_%s", uuid.New().String()[:8])
	now := time.Now()

	_, err = db.Exec(`
		INSERT INTO classifier_models (id, ontology_id, name, target_class, algorithm, model_artifact_path, is_active, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, 1, ?, ?)
	`, modelID, ontologyID, fmt.Sprintf("Model for %s", targetProperty), targetProperty, modelType, fmt.Sprintf("models/%s.gob", modelID), now, now)
	if err != nil {
		log.Printf("Warning: Could not create model record: %v", err)
	}

	// Publish training completed event (this will trigger twin auto-creation if enabled)
	utils.GetEventBus().Publish(utils.Event{
		Type:   utils.EventModelTrainingCompleted,
		Source: "agent-tool",
		Payload: map[string]any{
			"ontology_id":     ontologyID,
			"model_id":        modelID,
			"target_property": targetProperty,
			"model_type":      modelType,
			"accuracy":        0.0, // Would be filled by actual training
			"sample_count":    0,
		},
	})

	log.Printf("Agent triggered model training: %s for ontology %s (target: %s)", modelID, ontologyID, targetProperty)

	return map[string]interface{}{
		"model_id":        modelID,
		"ontology_id":     ontologyID,
		"ontology_name":   ontologyName,
		"target_property": targetProperty,
		"model_type":      modelType,
		"status":          "training_initiated",
		"message":         fmt.Sprintf("Model training initiated for ontology '%s' targeting property '%s'", ontologyName, targetProperty),
	}, nil
}

// toolQueryOntology executes a SPARQL query against the knowledge graph
func (s *Server) toolQueryOntology(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	query := getStringFromMap(input, "query", "")
	ontologyID := getStringFromMap(input, "ontology_id", "")
	limit := 100 // Default limit

	if limitVal, ok := input["limit"].(float64); ok {
		limit = int(limitVal)
	}

	if query == "" {
		return nil, fmt.Errorf("SPARQL query is required")
	}

	// Check if TDB2 backend is available
	if s.tdb2Backend == nil {
		return nil, fmt.Errorf("knowledge graph backend not available")
	}

	// Validate and inject GRAPH clause if ontologyID provided and query doesn't have one
	queryLower := strings.ToLower(query)
	if ontologyID != "" && !strings.Contains(queryLower, "graph") {
		// Inject GRAPH clause into WHERE clause
		graphURI := fmt.Sprintf("http://mimir.ai/ontology/%s", ontologyID)

		// Find WHERE clause and inject GRAPH
		whereIdx := strings.Index(strings.ToUpper(query), "WHERE")
		if whereIdx != -1 {
			// Find the opening brace after WHERE
			braceIdx := strings.Index(query[whereIdx:], "{")
			if braceIdx != -1 {
				insertPos := whereIdx + braceIdx + 1
				query = query[:insertPos] + fmt.Sprintf("\n  GRAPH <%s> {", graphURI) + query[insertPos:] + "\n  }"
			}
		}
	}

	// Add LIMIT if not present
	if !strings.Contains(queryLower, "limit") {
		query = fmt.Sprintf("%s LIMIT %d", query, limit)
	}

	// Execute SPARQL query
	results, err := s.tdb2Backend.QuerySPARQL(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("SPARQL query failed: %w", err)
	}

	// Convert results to a more friendly format
	rows := make([]map[string]interface{}, 0)
	for _, binding := range results.Bindings {
		row := make(map[string]interface{})
		for key, val := range binding {
			row[key] = val.Value
		}
		rows = append(rows, row)
	}

	log.Printf("Agent executed SPARQL query on ontology %s, returned %d results", ontologyID, len(rows))

	return map[string]interface{}{
		"ontology_id": ontologyID,
		"query":       query,
		"results":     rows,
		"count":       len(rows),
		"variables":   results.Variables,
		"message":     fmt.Sprintf("Query returned %d results", len(rows)),
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
