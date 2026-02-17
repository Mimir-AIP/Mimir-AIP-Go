package pipeline

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mimir-aip/mimir-aip-go/pkg/metadatastore"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

func setupTestService(t *testing.T) (*Service, string) {
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

	service := NewService(store)
	return service, tmpDir
}

func cleanupTestService(tmpDir string) {
	os.RemoveAll(tmpDir)
}

func TestPipelineCreate(t *testing.T) {
	service, tmpDir := setupTestService(t)
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
	service, tmpDir := setupTestService(t)
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
	service, tmpDir := setupTestService(t)
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
	service, tmpDir := setupTestService(t)
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
	service, tmpDir := setupTestService(t)
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
	service, tmpDir := setupTestService(t)
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
	service, tmpDir := setupTestService(t)
	defer cleanupTestService(tmpDir)

	// Create a pipeline
	created, err := service.Create(&models.PipelineCreateRequest{
		ProjectID:   "test-project",
		Name:        "test-pipeline",
		Type:        models.PipelineTypeIngestion,
		Description: "Original description",
		Steps: []models.PipelineStep{
			{Name: "step1", Plugin: "default", Action: "set_context"},
		},
	})
	if err != nil {
		t.Fatalf("Failed to create pipeline: %v", err)
	}

	// Update the pipeline
	newDesc := "Updated description"
	newStatus := models.PipelineStatusInactive
	updated, err := service.Update(created.ID, &models.PipelineUpdateRequest{
		Description: &newDesc,
		Status:      &newStatus,
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
}

func TestPipelineDelete(t *testing.T) {
	service, tmpDir := setupTestService(t)
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
