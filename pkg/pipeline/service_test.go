package pipeline

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mimir-aip/mimir-aip-go/pkg/metadatastore"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

func savePipelineTestProject(t *testing.T, store metadatastore.MetadataStore, projectID string) {
	t.Helper()
	now := time.Now().UTC()
	project := &models.Project{
		ID:          projectID,
		Name:        projectID,
		Description: "test project",
		Version:     "v1",
		Status:      models.ProjectStatusActive,
		Metadata:    models.ProjectMetadata{CreatedAt: now, UpdatedAt: now},
	}
	if err := store.SaveProject(project); err != nil {
		t.Fatalf("failed to save project %s: %v", projectID, err)
	}
}

func setupTestService(t *testing.T) (*Service, metadatastore.MetadataStore, string) {
	// Create temporary directory for test storage
	tmpDir, err := os.MkdirTemp("", "pipeline-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	dbPath := filepath.Join(tmpDir, "test.db")
	store, err := metadatastore.NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create SQLite store: %v", err)
	}
	savePipelineTestProject(t, store, "test-project")
	savePipelineTestProject(t, store, "project-1")
	savePipelineTestProject(t, store, "project-2")

	service := NewService(store)
	return service, store, tmpDir
}

func cleanupTestService(tmpDir string) {
	os.RemoveAll(tmpDir)
}

func TestPipelineCreate(t *testing.T) {
	service, _, tmpDir := setupTestService(t)
	defer cleanupTestService(tmpDir)

	req := &models.PipelineCreateRequest{
		ProjectID:   "test-project",
		Name:        "test-pipeline",
		Type:        models.PipelineTypeIngestion,
		Description: "A test pipeline",
		Steps: []models.PipelineStep{
			{
				Name:   "step1",
				Plugin: "default",
				Action: "set_context",
				Parameters: map[string]interface{}{
					"key":   "test",
					"value": "hello",
				},
			},
		},
	}

	pipeline, err := service.Create(req)
	if err != nil {
		t.Fatalf("Failed to create pipeline: %v", err)
	}

	if pipeline.ID == "" {
		t.Error("Expected pipeline ID to be set")
	}

	if pipeline.Name != req.Name {
		t.Errorf("Expected name %s, got %s", req.Name, pipeline.Name)
	}

	if pipeline.Type != req.Type {
		t.Errorf("Expected type %s, got %s", req.Type, pipeline.Type)
	}

	if pipeline.Status != models.PipelineStatusActive {
		t.Errorf("Expected status %s, got %s", models.PipelineStatusActive, pipeline.Status)
	}

	if len(pipeline.Steps) != 1 {
		t.Errorf("Expected 1 step, got %d", len(pipeline.Steps))
	}
}

func TestPipelineCreateValidation(t *testing.T) {
	service, _, tmpDir := setupTestService(t)
	defer cleanupTestService(tmpDir)

	tests := []struct {
		name string
		req  *models.PipelineCreateRequest
	}{
		{
			name: "missing name",
			req: &models.PipelineCreateRequest{
				ProjectID: "test",
				Type:      models.PipelineTypeIngestion,
				Steps: []models.PipelineStep{
					{Name: "step1", Plugin: "default", Action: "set_context"},
				},
			},
		},
		{
			name: "invalid type",
			req: &models.PipelineCreateRequest{
				ProjectID: "test",
				Name:      "test",
				Type:      "invalid",
				Steps: []models.PipelineStep{
					{Name: "step1", Plugin: "default", Action: "set_context"},
				},
			},
		},
		{
			name: "no steps",
			req: &models.PipelineCreateRequest{
				ProjectID: "test",
				Name:      "test",
				Type:      models.PipelineTypeIngestion,
				Steps:     []models.PipelineStep{},
			},
		},
		{
			name: "duplicate step names",
			req: &models.PipelineCreateRequest{
				ProjectID: "test",
				Name:      "test",
				Type:      models.PipelineTypeIngestion,
				Steps: []models.PipelineStep{
					{Name: "step1", Plugin: "default", Action: "set_context"},
					{Name: "step1", Plugin: "default", Action: "set_context"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.Create(tt.req)
			if err == nil {
				t.Errorf("Expected error for %s", tt.name)
			}
		})
	}
}

func TestPipelineGet(t *testing.T) {
	service, _, tmpDir := setupTestService(t)
	defer cleanupTestService(tmpDir)

	// Create a pipeline
	req := &models.PipelineCreateRequest{
		ProjectID: "test-project",
		Name:      "test-pipeline",
		Type:      models.PipelineTypeIngestion,
		Steps: []models.PipelineStep{
			{Name: "step1", Plugin: "default", Action: "set_context"},
		},
	}

	created, err := service.Create(req)
	if err != nil {
		t.Fatalf("Failed to create pipeline: %v", err)
	}

	// Get the pipeline
	retrieved, err := service.Get(created.ID)
	if err != nil {
		t.Fatalf("Failed to get pipeline: %v", err)
	}

	if retrieved.ID != created.ID {
		t.Errorf("Expected ID %s, got %s", created.ID, retrieved.ID)
	}

	if retrieved.Name != created.Name {
		t.Errorf("Expected name %s, got %s", created.Name, retrieved.Name)
	}
}

func TestPipelineList(t *testing.T) {
	service, _, tmpDir := setupTestService(t)
	defer cleanupTestService(tmpDir)

	// Create multiple pipelines
	for i := 1; i <= 3; i++ {
		req := &models.PipelineCreateRequest{
			ProjectID: "test-project",
			Name:      "test-pipeline-" + string(rune('0'+i)),
			Type:      models.PipelineTypeIngestion,
			Steps: []models.PipelineStep{
				{Name: "step1", Plugin: "default", Action: "set_context"},
			},
		}
		_, err := service.Create(req)
		if err != nil {
			t.Fatalf("Failed to create pipeline: %v", err)
		}
	}

	// List all pipelines
	pipelines, err := service.List()
	if err != nil {
		t.Fatalf("Failed to list pipelines: %v", err)
	}

	if len(pipelines) != 3 {
		t.Errorf("Expected 3 pipelines, got %d", len(pipelines))
	}
}

func TestPipelineListByProject(t *testing.T) {
	service, _, tmpDir := setupTestService(t)
	defer cleanupTestService(tmpDir)

	// Create pipelines for different projects
	project1Pipelines := 2
	project2Pipelines := 3

	for i := 0; i < project1Pipelines; i++ {
		req := &models.PipelineCreateRequest{
			ProjectID: "project-1",
			Name:      "pipeline-" + string(rune('0'+i)),
			Type:      models.PipelineTypeIngestion,
			Steps: []models.PipelineStep{
				{Name: "step1", Plugin: "default", Action: "set_context"},
			},
		}
		_, err := service.Create(req)
		if err != nil {
			t.Fatalf("Failed to create pipeline: %v", err)
		}
	}

	for i := 0; i < project2Pipelines; i++ {
		req := &models.PipelineCreateRequest{
			ProjectID: "project-2",
			Name:      "pipeline-" + string(rune('0'+i)),
			Type:      models.PipelineTypeIngestion,
			Steps: []models.PipelineStep{
				{Name: "step1", Plugin: "default", Action: "set_context"},
			},
		}
		_, err := service.Create(req)
		if err != nil {
			t.Fatalf("Failed to create pipeline: %v", err)
		}
	}

	// List project 1 pipelines
	project1List, err := service.ListByProject("project-1")
	if err != nil {
		t.Fatalf("Failed to list project-1 pipelines: %v", err)
	}

	if len(project1List) != project1Pipelines {
		t.Errorf("Expected %d pipelines for project-1, got %d", project1Pipelines, len(project1List))
	}

	// List project 2 pipelines
	project2List, err := service.ListByProject("project-2")
	if err != nil {
		t.Fatalf("Failed to list project-2 pipelines: %v", err)
	}

	if len(project2List) != project2Pipelines {
		t.Errorf("Expected %d pipelines for project-2, got %d", project2Pipelines, len(project2List))
	}
}

func TestPipelineExecution(t *testing.T) {
	service, _, tmpDir := setupTestService(t)
	defer cleanupTestService(tmpDir)

	// Create a simple pipeline
	pipeline, err := service.Create(&models.PipelineCreateRequest{
		ProjectID: "test-project",
		Name:      "test-pipeline",
		Type:      models.PipelineTypeProcessing,
		Steps: []models.PipelineStep{
			{
				Name:   "set_value",
				Plugin: "default",
				Action: "set_context",
				Parameters: map[string]interface{}{
					"key":   "greeting",
					"value": "hello world",
				},
			},
			{
				Name:   "get_value",
				Plugin: "default",
				Action: "get_context",
				Parameters: map[string]interface{}{
					"key": "greeting",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Failed to create pipeline: %v", err)
	}

	// Execute the pipeline
	execution, err := service.Execute(pipeline.ID, &models.PipelineExecutionRequest{
		TriggerType: "manual",
		TriggeredBy: "test-user",
	})
	if err != nil {
		t.Fatalf("Failed to execute pipeline: %v", err)
	}

	if execution.ID == "" {
		t.Error("Expected execution ID to be set")
	}

	if execution.Status != "completed" {
		t.Errorf("Expected status 'completed', got %s", execution.Status)
	}

	if execution.PipelineID != pipeline.ID {
		t.Errorf("Expected pipeline ID %s, got %s", pipeline.ID, execution.PipelineID)
	}

	// Check context has the expected data
	// The set_context action stores in "_global" by default
	stepData := execution.Context.Steps["_global"]
	if stepData == nil {
		t.Error("Expected _global step data in context")
	} else {
		// The greeting should be in the step data
		if greetingVal, ok := stepData["greeting"]; ok {
			if greetingVal != "hello world" {
				t.Errorf("Expected 'hello world', got %v", greetingVal)
			}
		} else {
			t.Error("Expected greeting key in _global step data")
		}
	}
}

func TestPipelineUpdate(t *testing.T) {
	service, _, tmpDir := setupTestService(t)
	defer cleanupTestService(tmpDir)

	created, err := service.Create(&models.PipelineCreateRequest{
		ProjectID:   "test-project",
		Name:        "test-pipeline",
		Type:        models.PipelineTypeIngestion,
		Description: "Original description",
		Steps: []models.PipelineStep{
			{Name: "step1", Plugin: "default", Action: "set_context", Parameters: map[string]interface{}{"key": "x", "value": "y"}},
		},
		TriggerConfig: &models.PipelineTriggerConfig{AllowManual: true, Webhook: true, Secret: "original-secret"},
	})
	if err != nil {
		t.Fatalf("Failed to create pipeline: %v", err)
	}

	newDesc := "Updated description"
	newStatus := models.PipelineStatusInactive
	updated, err := service.Update(created.ID, &models.PipelineUpdateRequest{
		Description: &newDesc,
		Status:      &newStatus,
		TriggerConfig: &models.PipelineTriggerConfig{
			AllowManual: false,
			Webhook:     true,
			Secret:      "",
		},
	})
	if err != nil {
		t.Fatalf("Failed to update pipeline: %v", err)
	}

	if updated.Description != newDesc {
		t.Errorf("Expected description %s, got %s", newDesc, updated.Description)
	}
	if updated.Status != newStatus {
		t.Errorf("Expected status %s, got %s", newStatus, updated.Status)
	}
	if updated.TriggerConfig == nil || updated.TriggerConfig.Secret != "original-secret" {
		t.Fatalf("expected webhook secret to be preserved on update, got %#v", updated.TriggerConfig)
	}
	if updated.TriggerConfig.AllowManual {
		t.Fatalf("expected manual trigger to be updated to false")
	}
}

func TestPipelineDelete(t *testing.T) {
	service, _, tmpDir := setupTestService(t)
	defer cleanupTestService(tmpDir)

	// Create a pipeline
	created, err := service.Create(&models.PipelineCreateRequest{
		ProjectID: "test-project",
		Name:      "test-pipeline",
		Type:      models.PipelineTypeIngestion,
		Steps: []models.PipelineStep{
			{Name: "step1", Plugin: "default", Action: "set_context"},
		},
	})
	if err != nil {
		t.Fatalf("Failed to create pipeline: %v", err)
	}

	// Delete the pipeline
	err = service.Delete(created.ID)
	if err != nil {
		t.Fatalf("Failed to delete pipeline: %v", err)
	}

	// Verify deletion
	_, err = service.Get(created.ID)
	if err == nil {
		t.Error("Expected error when getting deleted pipeline")
	}
}

func TestPipelineCreateRejectsMissingProject(t *testing.T) {
	service, _, tmpDir := setupTestService(t)
	defer cleanupTestService(tmpDir)

	_, err := service.Create(&models.PipelineCreateRequest{
		ProjectID: "missing-project",
		Name:      "invalid-pipeline",
		Type:      models.PipelineTypeIngestion,
		Steps:     []models.PipelineStep{{Name: "step1", Plugin: "default", Action: "set_context", Parameters: map[string]interface{}{"key": "test", "value": "hello"}}},
	})
	if err == nil {
		t.Fatal("expected create to fail when project does not exist")
	}
}

func TestPipelineCreateRejectsUnknownPluginActionAndGoto(t *testing.T) {
	service, _, tmpDir := setupTestService(t)
	defer cleanupTestService(tmpDir)

	tests := []struct {
		name string
		step models.PipelineStep
	}{
		{
			name: "unknown plugin",
			step: models.PipelineStep{Name: "step1", Plugin: "missing", Action: "set_context"},
		},
		{
			name: "unknown builtin action",
			step: models.PipelineStep{Name: "step1", Plugin: "default", Action: "missing_action"},
		},
		{
			name: "goto missing target",
			step: models.PipelineStep{Name: "step1", Plugin: "default", Action: "goto", Parameters: map[string]interface{}{"target": "does-not-exist"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.Create(&models.PipelineCreateRequest{
				ProjectID: "test-project",
				Name:      "invalid-pipeline",
				Type:      models.PipelineTypeIngestion,
				Steps:     []models.PipelineStep{tt.step},
			})
			if err == nil {
				t.Fatalf("expected validation error for %s", tt.name)
			}
		})
	}
}

func TestPipelineDeleteRejectsReferencedPipeline(t *testing.T) {
	service, store, tmpDir := setupTestService(t)
	defer cleanupTestService(tmpDir)

	created, err := service.Create(&models.PipelineCreateRequest{
		ProjectID: "test-project",
		Name:      "referenced-pipeline",
		Type:      models.PipelineTypeIngestion,
		Steps:     []models.PipelineStep{{Name: "step1", Plugin: "default", Action: "set_context", Parameters: map[string]interface{}{"key": "x", "value": "y"}}},
	})
	if err != nil {
		t.Fatalf("failed to create pipeline: %v", err)
	}

	schedule := &models.Schedule{ID: "schedule-1", ProjectID: "test-project", Name: "nightly", Pipelines: []string{created.ID}, CronSchedule: "0 * * * *", Enabled: true, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	if err := store.SaveSchedule(schedule); err != nil {
		t.Fatalf("failed to save schedule: %v", err)
	}

	automation := &models.Automation{
		ID:            "automation-1",
		ProjectID:     "test-project",
		Name:          "trigger export",
		Enabled:       true,
		TargetType:    models.AutomationTargetTypeDigitalTwin,
		TargetID:      "twin-1",
		TriggerType:   models.AutomationTriggerTypePipelineCompleted,
		TriggerConfig: map[string]any{"pipeline_ids": []string{created.ID}},
		ActionType:    models.AutomationActionTypeTriggerExportPipeline,
		ActionConfig:  map[string]any{"pipeline_id": created.ID},
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}
	if err := store.SaveAutomation(automation); err != nil {
		t.Fatalf("failed to save automation: %v", err)
	}

	ontology := &models.Ontology{ID: "ontology-1", ProjectID: "test-project", Name: "Ontology", Content: "@prefix ex: <http://example.com/> .", Status: "active", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	if err := store.SaveOntology(ontology); err != nil {
		t.Fatalf("failed to save ontology: %v", err)
	}
	twin := &models.DigitalTwin{ID: "twin-1", ProjectID: "test-project", OntologyID: ontology.ID, Name: "Twin", Status: "active", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	if err := store.SaveDigitalTwin(twin); err != nil {
		t.Fatalf("failed to save twin: %v", err)
	}
	action := &models.Action{ID: "action-1", DigitalTwinID: twin.ID, Name: "Export", Enabled: true, Trigger: &models.ActionTrigger{PipelineID: created.ID}, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	if err := store.SaveAction(action); err != nil {
		t.Fatalf("failed to save action: %v", err)
	}

	if err := store.SaveWorkTask(&models.WorkTask{ID: "task-1", ProjectID: "test-project", Type: models.WorkTaskTypePipelineExecution, Status: models.WorkTaskStatusQueued, Priority: 1, SubmittedAt: time.Now().UTC(), TaskSpec: models.TaskSpec{PipelineID: created.ID, ProjectID: "test-project"}}); err != nil {
		t.Fatalf("failed to save work task: %v", err)
	}

	err = service.Delete(created.ID)
	var inUseErr *PipelineInUseError
	if !errors.As(err, &inUseErr) {
		t.Fatalf("expected PipelineInUseError, got %v", err)
	}
	if len(inUseErr.References) < 4 {
		t.Fatalf("expected multiple references, got %#v", inUseErr.References)
	}
}
