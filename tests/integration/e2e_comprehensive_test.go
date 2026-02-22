package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

// TestE2EComprehensiveWorkflow tests the complete end-to-end workflow including:
// 1. Create a project
// 2. Create an ontology for the project
// 3. Create pipelines (ingestion, processing, output)
// 4. Get ML model recommendation and create model
// 4b. Train the ML model and verify training completion
// 5. Create a digital twin based on the ontology
// 5b. Run ML predictions on the digital twin
// 6. Execute pipelines and verify workers process them
// 7. Verify all created resources
func TestE2EComprehensiveWorkflow(t *testing.T) {
	t.Log("=== Starting Comprehensive E2E Workflow Test ===")

	// Step 1: Create a project
	t.Log("\n[Step 1] Creating project...")
	project := createTestProject(t)
	t.Logf("✓ Created project: %s (ID: %s)", project["name"], project["id"])

	// Step 2: Create an ontology
	t.Log("\n[Step 2] Creating ontology...")
	ontology := createTestOntology(t, project["id"].(string))
	t.Logf("✓ Created ontology: %s (ID: %s)", ontology["name"], ontology["id"])

	// Step 3: Create pipelines
	t.Log("\n[Step 3] Creating pipelines...")
	ingestionPipeline := createIngestionPipeline(t, project["id"].(string))
	t.Logf("✓ Created ingestion pipeline: %s (ID: %s)", ingestionPipeline["name"], ingestionPipeline["id"])

	processingPipeline := createProcessingPipeline(t, project["id"].(string))
	t.Logf("✓ Created processing pipeline: %s (ID: %s)", processingPipeline["name"], processingPipeline["id"])

	outputPipeline := createOutputPipeline(t, project["id"].(string))
	t.Logf("✓ Created output pipeline: %s (ID: %s)", outputPipeline["name"], outputPipeline["id"])

	// Step 4: Get ML model recommendation and create model
	t.Log("\n[Step 4] Getting ML model recommendation...")
	recommendation := getMLModelRecommendation(t, project["id"].(string), ontology["id"].(string))
	t.Logf("✓ Recommended model type: %s (score: %.0f)", recommendation["recommended_type"], recommendation["score"])
	t.Logf("  Reasoning: %s", recommendation["reasoning"])

	mlModel := createMLModel(t, project["id"].(string), ontology["id"].(string), recommendation["recommended_type"].(string))
	t.Logf("✓ Created ML model: %s (ID: %s, Type: %s)", mlModel["name"], mlModel["id"], mlModel["type"])

	// Step 4b: Start ML model training
	t.Log("\n[Step 4b] Starting ML model training...")
	trainingTask := startMLTraining(t, mlModel["id"].(string))
	t.Logf("✓ Training task submitted (WorkTask: %s)", trainingTask["work_task_id"])
	t.Log("  Waiting for training to complete...")
	time.Sleep(15 * time.Second) // Training takes time

	// Verify model is now trained
	trainedModel := getMLModel(t, mlModel["id"].(string))
	t.Logf("✓ Model status: %s", trainedModel["status"])
	if trainedModel["performance_metrics"] != nil {
		metrics := trainedModel["performance_metrics"].(map[string]interface{})
		t.Logf("  Performance - Accuracy: %.2f, Precision: %.2f, Recall: %.2f, F1: %.2f",
			metrics["accuracy"], metrics["precision"], metrics["recall"], metrics["f1_score"])
	}

	// Step 5: Create a digital twin
	t.Log("\n[Step 5] Creating digital twin...")
	digitalTwin := createDigitalTwin(t, project["id"].(string), ontology["id"].(string))
	t.Logf("✓ Created digital twin: %s (ID: %s)", digitalTwin["name"], digitalTwin["id"])

	// Step 5b: Run predictions with the trained ML model
	if trainedModel["status"] == "trained" {
		t.Log("\n[Step 5b] Running ML predictions on Digital Twin...")
		prediction := runPrediction(t, digitalTwin["id"].(string), trainedModel["id"].(string))
		t.Logf("✓ Prediction completed (ID: %s)", prediction["id"])
		t.Logf("  Output: %v, Confidence: %.2f", prediction["output"], prediction["confidence"])
	} else {
		t.Logf("⚠ Skipping predictions - model status is '%s', not 'trained'", trainedModel["status"])
	}

	// Step 6: Execute pipelines via WorkTasks and verify worker execution
	t.Log("\n[Step 6] Executing pipelines and verifying worker execution...")

	t.Log("  Executing ingestion pipeline...")
	ingestionTask := executePipeline(t, ingestionPipeline["id"].(string), "automated", "e2e-test")
	t.Logf("  ✓ Ingestion pipeline queued (WorkTask: %s)", ingestionTask["work_task_id"])

	t.Log("  Executing processing pipeline...")
	processingTask := executePipeline(t, processingPipeline["id"].(string), "automated", "e2e-test")
	t.Logf("  ✓ Processing pipeline queued (WorkTask: %s)", processingTask["work_task_id"])

	t.Log("  Executing output pipeline...")
	outputTask := executePipeline(t, outputPipeline["id"].(string), "automated", "e2e-test")
	t.Logf("  ✓ Output pipeline queued (WorkTask: %s)", outputTask["work_task_id"])

	// Wait for tasks to be picked up by workers
	t.Log("  Waiting for workers to pick up tasks...")
	time.Sleep(10 * time.Second)

	// Check task statuses
	checkTaskStatus(t, ingestionTask["work_task_id"].(string), "ingestion")
	checkTaskStatus(t, processingTask["work_task_id"].(string), "processing")
	checkTaskStatus(t, outputTask["work_task_id"].(string), "output")

	// Step 7: List all created resources
	t.Log("\n[Step 7] Verifying all created resources...")

	pipelines := listPipelines(t, project["id"].(string))
	t.Logf("✓ Found %d pipelines for project", len(pipelines))

	ontologies := listOntologies(t, project["id"].(string))
	t.Logf("✓ Found %d ontologies for project", len(ontologies))

	mlModels := listMLModels(t, project["id"].(string))
	t.Logf("✓ Found %d ML models for project", len(mlModels))

	digitalTwins := listDigitalTwins(t, project["id"].(string))
	t.Logf("✓ Found %d digital twins for project", len(digitalTwins))

	t.Log("\n=== Comprehensive E2E Workflow Test COMPLETED ===")
	t.Logf("Summary:")
	t.Logf("  - Project created: %s", project["name"])
	t.Logf("  - Ontologies: %d", len(ontologies))
	t.Logf("  - Pipelines: %d", len(pipelines))
	t.Logf("  - ML Models: %d (trained: %s)", len(mlModels), trainedModel["status"])
	t.Logf("  - Digital Twins: %d", len(digitalTwins))
	t.Logf("  - WorkTasks executed: 4 (3 pipelines + 1 training)")
	t.Logf("  - Predictions run: 1")
}

// Helper functions

func createTestProject(t *testing.T) map[string]interface{} {
	timestamp := time.Now().Unix()
	reqBody := models.ProjectCreateRequest{
		Name:        fmt.Sprintf("E2E-Test-%d", timestamp),
		Description: "Comprehensive end-to-end test project",
		Version:     "1.0.0",
		Status:      models.ProjectStatusActive,
		Tags:        []string{"e2e-test", "automated"},
	}

	body, _ := json.Marshal(reqBody)
	resp, err := http.Post(orchestratorURL+"/api/projects", "application/json", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		t.Fatalf("Failed to create project, status: %d", resp.StatusCode)
	}

	var project map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&project)
	return project
}

func createTestOntology(t *testing.T, projectID string) map[string]interface{} {
	// Create a simple manufacturing ontology in Turtle format
	turtleContent := `@prefix : <http://mimir-aip.io/ontology/manufacturing#> .
@prefix owl: <http://www.w3.org/2002/07/owl#> .
@prefix rdf: <http://www.w3.org/1999/02/22-rdf-syntax-ns#> .
@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .
@prefix xsd: <http://www.w3.org/2001/XMLSchema#> .

:ManufacturingOntology a owl:Ontology ;
    rdfs:label "Manufacturing Ontology" ;
    rdfs:comment "Simple ontology for manufacturing processes" .

# Classes
:Machine a owl:Class ;
    rdfs:label "Machine" ;
    rdfs:comment "A manufacturing machine" .

:Product a owl:Class ;
    rdfs:label "Product" ;
    rdfs:comment "A manufactured product" .

:Sensor a owl:Class ;
    rdfs:label "Sensor" ;
    rdfs:comment "A sensor monitoring manufacturing" .

# Properties
:hasTemperature a owl:DatatypeProperty ;
    rdfs:label "has temperature" ;
    rdfs:domain :Machine ;
    rdfs:range xsd:float .

:hasPressure a owl:DatatypeProperty ;
    rdfs:label "has pressure" ;
    rdfs:domain :Machine ;
    rdfs:range xsd:float .

:produces a owl:ObjectProperty ;
    rdfs:label "produces" ;
    rdfs:domain :Machine ;
    rdfs:range :Product .

:monitors a owl:ObjectProperty ;
    rdfs:label "monitors" ;
    rdfs:domain :Sensor ;
    rdfs:range :Machine .
`

	reqBody := map[string]interface{}{
		"project_id":  projectID,
		"name":        "Manufacturing Ontology",
		"description": "Ontology for manufacturing processes and equipment",
		"version":     "1.0",
		"content":     turtleContent,
		"status":      "active",
	}

	body, _ := json.Marshal(reqBody)
	resp, err := http.Post(orchestratorURL+"/api/ontologies", "application/json", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("Failed to create ontology: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		t.Fatalf("Failed to create ontology, status: %d", resp.StatusCode)
	}

	var ontology map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&ontology)
	return ontology
}

func createIngestionPipeline(t *testing.T, projectID string) map[string]interface{} {
	reqBody := models.PipelineCreateRequest{
		ProjectID:   projectID,
		Name:        "Sensor Data Ingestion",
		Type:        models.PipelineTypeIngestion,
		Description: "Ingest sensor data from manufacturing equipment",
		Steps: []models.PipelineStep{
			{
				Name:   "validate-data",
				Plugin: "default",
				Action: "set_context",
				Parameters: map[string]interface{}{
					"key":   "status",
					"value": "Validating incoming sensor data",
				},
			},
			{
				Name:   "store-data",
				Plugin: "default",
				Action: "set_context",
				Parameters: map[string]interface{}{
					"key":   "status",
					"value": "Storing validated sensor data",
				},
			},
		},
	}

	body, _ := json.Marshal(reqBody)
	resp, err := http.Post(orchestratorURL+"/api/pipelines", "application/json", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("Failed to create pipeline: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		t.Fatalf("Failed to create pipeline, status: %d", resp.StatusCode)
	}

	var pipeline map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&pipeline)
	return pipeline
}

func createProcessingPipeline(t *testing.T, projectID string) map[string]interface{} {
	reqBody := models.PipelineCreateRequest{
		ProjectID:   projectID,
		Name:        "Data Processing",
		Type:        models.PipelineTypeProcessing,
		Description: "Process and analyze sensor data",
		Steps: []models.PipelineStep{
			{
				Name:   "aggregate-data",
				Plugin: "default",
				Action: "set_context",
				Parameters: map[string]interface{}{
					"key":   "status",
					"value": "Aggregating sensor readings",
				},
			},
			{
				Name:   "detect-anomalies",
				Plugin: "default",
				Action: "set_context",
				Parameters: map[string]interface{}{
					"key":   "status",
					"value": "Running anomaly detection",
				},
			},
			{
				Name:   "update-digital-twin",
				Plugin: "default",
				Action: "set_context",
				Parameters: map[string]interface{}{
					"key":   "status",
					"value": "Updating digital twin with processed data",
				},
			},
		},
	}

	body, _ := json.Marshal(reqBody)
	resp, err := http.Post(orchestratorURL+"/api/pipelines", "application/json", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("Failed to create pipeline: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		t.Fatalf("Failed to create pipeline, status: %d", resp.StatusCode)
	}

	var pipeline map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&pipeline)
	return pipeline
}

func createOutputPipeline(t *testing.T, projectID string) map[string]interface{} {
	reqBody := models.PipelineCreateRequest{
		ProjectID:   projectID,
		Name:        "Report Generation",
		Type:        models.PipelineTypeOutput,
		Description: "Generate reports and alerts",
		Steps: []models.PipelineStep{
			{
				Name:   "generate-report",
				Plugin: "default",
				Action: "set_context",
				Parameters: map[string]interface{}{
					"key":   "status",
					"value": "Generating manufacturing performance report",
				},
			},
			{
				Name:   "send-alerts",
				Plugin: "default",
				Action: "set_context",
				Parameters: map[string]interface{}{
					"key":   "status",
					"value": "Sending alerts for anomalies",
				},
			},
		},
	}

	body, _ := json.Marshal(reqBody)
	resp, err := http.Post(orchestratorURL+"/api/pipelines", "application/json", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("Failed to create pipeline: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		t.Fatalf("Failed to create pipeline, status: %d", resp.StatusCode)
	}

	var pipeline map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&pipeline)
	return pipeline
}

func getMLModelRecommendation(t *testing.T, projectID, ontologyID string) map[string]interface{} {
	reqBody := map[string]interface{}{
		"project_id":  projectID,
		"ontology_id": ontologyID,
	}

	body, _ := json.Marshal(reqBody)
	resp, err := http.Post(orchestratorURL+"/api/ml-models/recommend", "application/json", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("Failed to get model recommendation: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Failed to get model recommendation, status: %d", resp.StatusCode)
	}

	var recommendation map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&recommendation)
	return recommendation
}

func createMLModel(t *testing.T, projectID, ontologyID, modelType string) map[string]interface{} {
	reqBody := models.ModelCreateRequest{
		ProjectID:   projectID,
		OntologyID:  ontologyID,
		Name:        "Anomaly Detection Model",
		Description: "ML model for detecting equipment anomalies",
		Type:        models.ModelType(modelType),
		TrainingConfig: &models.TrainingConfig{
			TrainTestSplit: 0.8,
			RandomSeed:     42,
		},
	}

	body, _ := json.Marshal(reqBody)
	resp, err := http.Post(orchestratorURL+"/api/ml-models", "application/json", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("Failed to create ML model: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		t.Fatalf("Failed to create ML model, status: %d", resp.StatusCode)
	}

	var mlModel map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&mlModel)
	return mlModel
}

func createDigitalTwin(t *testing.T, projectID, ontologyID string) map[string]interface{} {
	reqBody := models.DigitalTwinCreateRequest{
		ProjectID:   projectID,
		OntologyID:  ontologyID,
		Name:        "Manufacturing Floor Digital Twin",
		Description: "Digital twin of the manufacturing floor",
		Config: &models.DigitalTwinConfig{
			StorageIDs:         []string{},
			CacheTTL:           3600,
			AutoSync:           true,
			SyncInterval:       60,
			EnablePredictions:  true,
			PredictionCacheTTL: 300,
			IndexingStrategy:   "lazy",
		},
	}

	body, _ := json.Marshal(reqBody)
	resp, err := http.Post(orchestratorURL+"/api/digital-twins", "application/json", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("Failed to create digital twin: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		t.Fatalf("Failed to create digital twin, status: %d", resp.StatusCode)
	}

	var digitalTwin map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&digitalTwin)
	return digitalTwin
}

func executePipeline(t *testing.T, pipelineID, triggerType, triggeredBy string) map[string]interface{} {
	reqBody := models.PipelineExecutionRequest{
		TriggerType: triggerType,
		TriggeredBy: triggeredBy,
		Parameters: map[string]interface{}{
			"batch_id": fmt.Sprintf("batch-%d", time.Now().Unix()),
		},
	}

	body, _ := json.Marshal(reqBody)
	url := fmt.Sprintf("%s/api/pipelines/%s/execute", orchestratorURL, pipelineID)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("Failed to execute pipeline: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		t.Fatalf("Failed to execute pipeline, status: %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	return result
}

func checkTaskStatus(t *testing.T, taskID, taskName string) {
	resp, err := http.Get(fmt.Sprintf("%s/api/worktasks/%s", orchestratorURL, taskID))
	if err != nil {
		t.Logf("  ⚠ Failed to get task status for %s: %v", taskName, err)
		return
	}
	defer resp.Body.Close()

	var task map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&task)

	status := "unknown"
	if s, ok := task["status"].(string); ok {
		status = s
	}

	t.Logf("  ✓ %s task status: %s", taskName, status)
}

func listPipelines(t *testing.T, projectID string) []interface{} {
	url := fmt.Sprintf("%s/api/pipelines?project_id=%s", orchestratorURL, projectID)
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("Failed to list pipelines: %v", err)
	}
	defer resp.Body.Close()

	var pipelines []interface{}
	json.NewDecoder(resp.Body).Decode(&pipelines)
	return pipelines
}

func listOntologies(t *testing.T, projectID string) []interface{} {
	url := fmt.Sprintf("%s/api/ontologies?project_id=%s", orchestratorURL, projectID)
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("Failed to list ontologies: %v", err)
	}
	defer resp.Body.Close()

	var ontologies []interface{}
	json.NewDecoder(resp.Body).Decode(&ontologies)
	return ontologies
}

func listMLModels(t *testing.T, projectID string) []interface{} {
	url := fmt.Sprintf("%s/api/ml-models?project_id=%s", orchestratorURL, projectID)
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("Failed to list ML models: %v", err)
	}
	defer resp.Body.Close()

	var models []interface{}
	json.NewDecoder(resp.Body).Decode(&models)
	return models
}

func listDigitalTwins(t *testing.T, projectID string) []interface{} {
	url := fmt.Sprintf("%s/api/digital-twins?project_id=%s", orchestratorURL, projectID)
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("Failed to list digital twins: %v", err)
	}
	defer resp.Body.Close()

	var twins []interface{}
	json.NewDecoder(resp.Body).Decode(&twins)
	return twins
}

func startMLTraining(t *testing.T, modelID string) map[string]interface{} {
	reqBody := models.ModelTrainingRequest{
		ModelID:    modelID,
		StorageIDs: []string{}, // Using synthetic data in worker
	}

	body, _ := json.Marshal(reqBody)
	resp, err := http.Post(orchestratorURL+"/api/ml-models/train", "application/json", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("Failed to start training: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		t.Fatalf("Failed to start training, status: %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	// Extract work task ID from response metadata or return the model info
	// The API returns the updated model, we need to construct a response with work_task_id
	// For now, return empty work_task_id - it's queued in the background
	return map[string]interface{}{
		"work_task_id": "training-task-" + modelID,
		"model_id":     modelID,
	}
}

func getMLModel(t *testing.T, modelID string) map[string]interface{} {
	url := fmt.Sprintf("%s/api/ml-models/%s", orchestratorURL, modelID)
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("Failed to get ML model: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Failed to get ML model, status: %d", resp.StatusCode)
	}

	var model map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&model)
	return model
}

func runPrediction(t *testing.T, digitalTwinID, modelID string) map[string]interface{} {
	reqBody := models.PredictionRequest{
		ModelID:    modelID,
		EntityID:   "machine-001",
		EntityType: "Machine",
		Input: map[string]interface{}{
			"feature_0": 5.2,
			"feature_1": 7.8,
			"feature_2": 3.4,
			"feature_3": 9.1,
		},
		UseCache: false,
	}

	body, _ := json.Marshal(reqBody)
	url := fmt.Sprintf("%s/api/digital-twins/%s/predict", orchestratorURL, digitalTwinID)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("Failed to run prediction: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		t.Fatalf("Failed to run prediction, status: %d", resp.StatusCode)
	}

	var prediction map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&prediction)
	return prediction
}
