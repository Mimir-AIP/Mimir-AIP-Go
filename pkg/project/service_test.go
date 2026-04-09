package project

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/mimir-aip/mimir-aip-go/pkg/metadatastore"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

func setupTestService(t *testing.T) (*Service, string) {
	// Create temporary directory for test storage
	tmpDir, err := os.MkdirTemp("", "project-test-*")
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

func TestProjectCreate(t *testing.T) {
	service, tmpDir := setupTestService(t)
	defer cleanupTestService(tmpDir)

	req := &models.ProjectCreateRequest{
		Name:        "test-project",
		Description: "A test project",
	}

	project, err := service.Create(req)
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	if project.ID == "" {
		t.Error("Expected project ID to be set")
	}

	if project.Name != req.Name {
		t.Errorf("Expected name %s, got %s", req.Name, project.Name)
	}

	if project.Description != req.Description {
		t.Errorf("Expected description %s, got %s", req.Description, project.Description)
	}

	if project.Status != models.ProjectStatusActive {
		t.Errorf("Expected status %s, got %s", models.ProjectStatusActive, project.Status)
	}

	if project.Metadata.CreatedAt.IsZero() {
		t.Error("Expected CreatedAt to be set")
	}

	if project.Metadata.UpdatedAt.IsZero() {
		t.Error("Expected UpdatedAt to be set")
	}
}

func TestProjectCreateValidation(t *testing.T) {
	service, tmpDir := setupTestService(t)
	defer cleanupTestService(tmpDir)

	// Test missing name
	req := &models.ProjectCreateRequest{
		Name: "",
	}

	_, err := service.Create(req)
	if err == nil {
		t.Error("Expected error for missing name")
	}
}

func TestProjectGet(t *testing.T) {
	service, tmpDir := setupTestService(t)
	defer cleanupTestService(tmpDir)

	// Create a project
	req := &models.ProjectCreateRequest{
		Name: "test-project",
	}

	created, err := service.Create(req)
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// Get the project
	retrieved, err := service.Get(created.ID)
	if err != nil {
		t.Fatalf("Failed to get project: %v", err)
	}

	if retrieved.ID != created.ID {
		t.Errorf("Expected ID %s, got %s", created.ID, retrieved.ID)
	}

	if retrieved.Name != created.Name {
		t.Errorf("Expected name %s, got %s", created.Name, retrieved.Name)
	}

	// Test non-existent project
	_, err = service.Get("non-existent")
	if err == nil {
		t.Error("Expected error for non-existent project")
	}
}

func TestProjectList(t *testing.T) {
	service, tmpDir := setupTestService(t)
	defer cleanupTestService(tmpDir)

	// Create multiple projects
	for i := 1; i <= 3; i++ {
		req := &models.ProjectCreateRequest{
			Name: "test-project-" + string(rune('0'+i)),
		}
		_, err := service.Create(req)
		if err != nil {
			t.Fatalf("Failed to create project: %v", err)
		}
	}

	// List all projects
	projects, err := service.List(&models.ProjectListQuery{})
	if err != nil {
		t.Fatalf("Failed to list projects: %v", err)
	}

	if len(projects) != 3 {
		t.Errorf("Expected 3 projects, got %d", len(projects))
	}
}

func TestProjectUpdate(t *testing.T) {
	service, tmpDir := setupTestService(t)
	defer cleanupTestService(tmpDir)

	// Create a project
	req := &models.ProjectCreateRequest{
		Name: "test-project",
	}

	created, err := service.Create(req)
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// Store original UpdatedAt
	originalUpdatedAt := created.Metadata.UpdatedAt

	// Sleep a small amount to ensure timestamp changes
	time.Sleep(100 * time.Millisecond)

	// Update the project
	newDesc := "Updated description"
	newStatus := models.ProjectStatusArchived
	updateReq := &models.ProjectUpdateRequest{
		Description: &newDesc,
		Status:      &newStatus,
	}

	updated, err := service.Update(created.ID, updateReq)
	if err != nil {
		t.Fatalf("Failed to update project: %v", err)
	}

	if updated.Description != newDesc {
		t.Errorf("Expected description %s, got %s", newDesc, updated.Description)
	}

	if updated.Status != newStatus {
		t.Errorf("Expected status %s, got %s", newStatus, updated.Status)
	}

	if !updated.Metadata.UpdatedAt.After(originalUpdatedAt) {
		t.Errorf("Expected UpdatedAt to be updated. Created: %v, Updated: %v",
			originalUpdatedAt, updated.Metadata.UpdatedAt)
	}
}

func TestProjectDelete(t *testing.T) {
	service, tmpDir := setupTestService(t)
	defer cleanupTestService(tmpDir)

	// Create a project
	req := &models.ProjectCreateRequest{
		Name: "test-project",
	}

	created, err := service.Create(req)
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// Delete the project (soft delete)
	err = service.Delete(created.ID)
	if err != nil {
		t.Fatalf("Failed to delete project: %v", err)
	}

	// Verify soft deletion - project still exists but is archived
	deleted, err := service.Get(created.ID)
	if err != nil {
		t.Fatalf("Failed to get deleted project: %v", err)
	}

	if deleted.Status != models.ProjectStatusArchived {
		t.Errorf("Expected status %s after deletion, got %s", models.ProjectStatusArchived, deleted.Status)
	}

	// Verify project still exists in database (soft delete)
	retrieved, err := service.Get(created.ID)
	if err != nil {
		t.Error("Expected project to still exist in database (soft delete)")
	}
	if retrieved.Status != models.ProjectStatusArchived {
		t.Errorf("Expected archived status, got %s", retrieved.Status)
	}
}

func TestProjectClone(t *testing.T) {
	service, tmpDir := setupTestService(t)
	defer cleanupTestService(tmpDir)

	original, err := service.Create(&models.ProjectCreateRequest{
		Name:        "original-project",
		Description: "Original description",
		Version:     "2.3.4",
	})
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	now := time.Now().UTC()
	ontology := &models.Ontology{
		ID:        "ontology-original",
		ProjectID: original.ID,
		Name:      "OperationsOntology",
		Content:   "@prefix ex: <http://example.com/> .",
		Status:    "active",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := service.store.SaveOntology(ontology); err != nil {
		t.Fatalf("failed to save ontology: %v", err)
	}

	storageConfig := &models.StorageConfig{
		ID:         "storage-original",
		ProjectID:  original.ID,
		PluginType: "filesystem",
		Config: map[string]interface{}{
			"path": "./data/original",
		},
		OntologyID: ontology.ID,
		Active:     true,
		CreatedAt:  now.Format(time.RFC3339),
		UpdatedAt:  now.Format(time.RFC3339),
	}
	if err := service.store.SaveStorageConfig(storageConfig); err != nil {
		t.Fatalf("failed to save storage config: %v", err)
	}

	pipeline := &models.Pipeline{
		ID:        "pipeline-original",
		ProjectID: original.ID,
		Name:      "daily-ingest",
		Type:      models.PipelineTypeIngestion,
		Steps: []models.PipelineStep{{
			Name:   "store",
			Plugin: "builtin",
			Action: "store_cir",
			Parameters: map[string]interface{}{
				"storage_id": storageConfig.ID,
			},
		}},
		Status:    models.PipelineStatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := service.store.SavePipeline(pipeline); err != nil {
		t.Fatalf("failed to save pipeline: %v", err)
	}

	schedule := &models.Schedule{
		ID:           "schedule-original",
		ProjectID:    original.ID,
		Name:         "nightly",
		Pipelines:    []string{pipeline.ID},
		CronSchedule: "0 0 * * *",
		Enabled:      true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := service.store.SaveSchedule(schedule); err != nil {
		t.Fatalf("failed to save schedule: %v", err)
	}

	trainedAt := now
	model := &models.MLModel{
		ID:                "model-original",
		ProjectID:         original.ID,
		OntologyID:        ontology.ID,
		Name:              "RiskModel",
		Type:              models.ModelTypeDecisionTree,
		Status:            models.ModelStatusTrained,
		Version:           "1.0.0",
		ModelArtifactPath: "/tmp/original-model.bin",
		CreatedAt:         now,
		UpdatedAt:         now,
		TrainedAt:         &trainedAt,
	}
	if err := service.store.SaveMLModel(model); err != nil {
		t.Fatalf("failed to save model: %v", err)
	}

	twin := &models.DigitalTwin{
		ID:         "twin-original",
		ProjectID:  original.ID,
		OntologyID: ontology.ID,
		Name:       "PlantTwin",
		Status:     "error",
		Config: &models.DigitalTwinConfig{
			StorageIDs: []string{storageConfig.ID},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := service.store.SaveDigitalTwin(twin); err != nil {
		t.Fatalf("failed to save digital twin: %v", err)
	}

	automation := &models.Automation{
		ID:          "automation-original",
		ProjectID:   original.ID,
		Name:        "Export on alert",
		Enabled:     true,
		TargetType:  models.AutomationTargetTypeDigitalTwin,
		TargetID:    twin.ID,
		TriggerType: models.AutomationTriggerTypeManual,
		ActionType:  models.AutomationActionTypeTriggerExportPipeline,
		ActionConfig: map[string]any{
			"pipeline_id": pipeline.ID,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := service.store.SaveAutomation(automation); err != nil {
		t.Fatalf("failed to save automation: %v", err)
	}

	cloned, err := service.Clone(original.ID, "cloned-project")
	if err != nil {
		t.Fatalf("Failed to clone project: %v", err)
	}

	if cloned.ID == original.ID {
		t.Error("Expected cloned project to have different ID")
	}
	if cloned.Name != "cloned-project" {
		t.Errorf("Expected name 'cloned-project', got %s", cloned.Name)
	}
	if cloned.Description != original.Description {
		t.Errorf("Expected description %s, got %s", original.Description, cloned.Description)
	}
	if cloned.Version != original.Version {
		t.Errorf("Expected version %s, got %s", original.Version, cloned.Version)
	}
	if cloned.Status != models.ProjectStatusDraft {
		t.Errorf("Expected status %s, got %s", models.ProjectStatusDraft, cloned.Status)
	}

	clonedOntologies, err := service.store.ListOntologiesByProject(cloned.ID)
	if err != nil || len(clonedOntologies) != 1 {
		t.Fatalf("expected 1 cloned ontology, got %d (err=%v)", len(clonedOntologies), err)
	}
	clonedStorageConfigs, err := service.store.ListStorageConfigsByProject(cloned.ID)
	if err != nil || len(clonedStorageConfigs) != 1 {
		t.Fatalf("expected 1 cloned storage config, got %d (err=%v)", len(clonedStorageConfigs), err)
	}
	clonedPipelines, err := service.store.ListPipelinesByProject(cloned.ID)
	if err != nil || len(clonedPipelines) != 1 {
		t.Fatalf("expected 1 cloned pipeline, got %d (err=%v)", len(clonedPipelines), err)
	}
	clonedSchedules, err := service.store.ListSchedulesByProject(cloned.ID)
	if err != nil || len(clonedSchedules) != 1 {
		t.Fatalf("expected 1 cloned schedule, got %d (err=%v)", len(clonedSchedules), err)
	}
	clonedModels, err := service.store.ListMLModelsByProject(cloned.ID)
	if err != nil || len(clonedModels) != 1 {
		t.Fatalf("expected 1 cloned model, got %d (err=%v)", len(clonedModels), err)
	}
	clonedTwins, err := service.store.ListDigitalTwinsByProject(cloned.ID)
	if err != nil || len(clonedTwins) != 1 {
		t.Fatalf("expected 1 cloned twin, got %d (err=%v)", len(clonedTwins), err)
	}
	clonedAutomations, err := service.store.ListAutomationsByProject(cloned.ID)
	if err != nil || len(clonedAutomations) != 1 {
		t.Fatalf("expected 1 cloned automation, got %d (err=%v)", len(clonedAutomations), err)
	}

	clonedOntology := clonedOntologies[0]
	clonedStorage := clonedStorageConfigs[0]
	clonedPipeline := clonedPipelines[0]
	clonedSchedule := clonedSchedules[0]
	clonedModel := clonedModels[0]
	clonedTwin := clonedTwins[0]
	clonedAutomation := clonedAutomations[0]

	if clonedOntology.ID == ontology.ID {
		t.Fatal("expected cloned ontology to get a new ID")
	}
	if clonedStorage.OntologyID != clonedOntology.ID {
		t.Fatalf("expected cloned storage ontology %s, got %s", clonedOntology.ID, clonedStorage.OntologyID)
	}
	if got := clonedPipeline.Steps[0].Parameters["storage_id"]; got != clonedStorage.ID {
		t.Fatalf("expected cloned pipeline storage_id %s, got %#v", clonedStorage.ID, got)
	}
	if len(clonedSchedule.Pipelines) != 1 || clonedSchedule.Pipelines[0] != clonedPipeline.ID {
		t.Fatalf("expected cloned schedule to target cloned pipeline %s, got %#v", clonedPipeline.ID, clonedSchedule.Pipelines)
	}
	if clonedSchedule.Enabled {
		t.Fatal("expected cloned schedule to be disabled by default")
	}
	if clonedModel.OntologyID != clonedOntology.ID {
		t.Fatalf("expected cloned model ontology %s, got %s", clonedOntology.ID, clonedModel.OntologyID)
	}
	if clonedModel.Status != models.ModelStatusDraft {
		t.Fatalf("expected cloned model status draft, got %s", clonedModel.Status)
	}
	if clonedModel.ModelArtifactPath != "" || clonedModel.TrainedAt != nil {
		t.Fatalf("expected cloned model runtime training state to be cleared, got artifact=%q trained_at=%v", clonedModel.ModelArtifactPath, clonedModel.TrainedAt)
	}
	if clonedTwin.OntologyID != clonedOntology.ID {
		t.Fatalf("expected cloned twin ontology %s, got %s", clonedOntology.ID, clonedTwin.OntologyID)
	}
	if clonedTwin.Status != "active" {
		t.Fatalf("expected cloned twin to reset to active, got %s", clonedTwin.Status)
	}
	if clonedTwin.Config == nil || len(clonedTwin.Config.StorageIDs) != 1 || clonedTwin.Config.StorageIDs[0] != clonedStorage.ID {
		t.Fatalf("expected cloned twin storage IDs [%s], got %#v", clonedStorage.ID, clonedTwin.Config)
	}
	if clonedAutomation.TargetID != clonedTwin.ID {
		t.Fatalf("expected cloned automation target %s, got %s", clonedTwin.ID, clonedAutomation.TargetID)
	}
	if got := clonedAutomation.ActionConfig["pipeline_id"]; got != clonedPipeline.ID {
		t.Fatalf("expected cloned automation pipeline_id %s, got %#v", clonedPipeline.ID, got)
	}

	if !slices.Contains(cloned.Components.Ontologies, clonedOntology.ID) ||
		!slices.Contains(cloned.Components.StorageConfigs, clonedStorage.ID) ||
		!slices.Contains(cloned.Components.Pipelines, clonedPipeline.ID) ||
		!slices.Contains(cloned.Components.MLModels, clonedModel.ID) ||
		!slices.Contains(cloned.Components.DigitalTwins, clonedTwin.ID) {
		t.Fatalf("expected cloned project component lists to reference cloned resources: %#v", cloned.Components)
	}
}

func TestProjectListByStatus(t *testing.T) {
	service, tmpDir := setupTestService(t)
	defer cleanupTestService(tmpDir)

	// Create active project
	activeReq := &models.ProjectCreateRequest{
		Name: "active-project",
	}
	active, err := service.Create(activeReq)
	if err != nil {
		t.Fatalf("Failed to create active project: %v", err)
	}

	// Create archived project
	archivedReq := &models.ProjectCreateRequest{
		Name: "archived-project",
	}
	archived, err := service.Create(archivedReq)
	if err != nil {
		t.Fatalf("Failed to create archived project: %v", err)
	}

	// Archive the second project
	archivedStatus := models.ProjectStatusArchived
	_, err = service.Update(archived.ID, &models.ProjectUpdateRequest{
		Status: &archivedStatus,
	})
	if err != nil {
		t.Fatalf("Failed to archive project: %v", err)
	}

	// List active projects
	activeProjects, err := service.List(&models.ProjectListQuery{
		Status: string(models.ProjectStatusActive),
	})
	if err != nil {
		t.Fatalf("Failed to list active projects: %v", err)
	}

	if len(activeProjects) != 1 {
		t.Errorf("Expected 1 active project, got %d", len(activeProjects))
	}

	if activeProjects[0].ID != active.ID {
		t.Errorf("Expected active project ID %s, got %s", active.ID, activeProjects[0].ID)
	}

	// List archived projects
	archivedProjects, err := service.List(&models.ProjectListQuery{
		Status: string(models.ProjectStatusArchived),
	})
	if err != nil {
		t.Fatalf("Failed to list archived projects: %v", err)
	}

	if len(archivedProjects) != 1 {
		t.Errorf("Expected 1 archived project, got %d", len(archivedProjects))
	}
}

func TestAddComponentRejectsCrossProjectAssociation(t *testing.T) {
	testCases := []struct {
		name           string
		componentID    string
		setupComponent func(service *Service, projectID, componentID string) error
		add            func(service *Service, projectID, componentID string) error
		getComponents  func(project *models.Project) []string
	}{
		{
			name:        "pipeline",
			componentID: "pipe-cross-project",
			setupComponent: func(service *Service, projectID, componentID string) error {
				return service.store.SavePipeline(&models.Pipeline{
					ID:        componentID,
					ProjectID: projectID,
					Name:      "pipeline",
					Type:      models.PipelineTypeIngestion,
					Status:    models.PipelineStatusDraft,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				})
			},
			add: func(service *Service, projectID, componentID string) error {
				return service.AddPipeline(projectID, componentID)
			},
			getComponents: func(project *models.Project) []string { return project.Components.Pipelines },
		},
		{
			name:        "ontology",
			componentID: "ont-cross-project",
			setupComponent: func(service *Service, projectID, componentID string) error {
				return service.store.SaveOntology(&models.Ontology{
					ID:        componentID,
					ProjectID: projectID,
					Name:      "ontology",
					Content:   "@prefix ex: <http://example.com/> .",
					Status:    "draft",
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				})
			},
			add: func(service *Service, projectID, componentID string) error {
				return service.AddOntology(projectID, componentID)
			},
			getComponents: func(project *models.Project) []string { return project.Components.Ontologies },
		},
		{
			name:        "ml-model",
			componentID: "ml-cross-project",
			setupComponent: func(service *Service, projectID, componentID string) error {
				return service.store.SaveMLModel(&models.MLModel{
					ID:         componentID,
					ProjectID:  projectID,
					OntologyID: "ontology",
					Name:       "model",
					Type:       models.ModelTypeDecisionTree,
					Status:     models.ModelStatusDraft,
					Version:    "1.0.0",
					CreatedAt:  time.Now(),
					UpdatedAt:  time.Now(),
				})
			},
			add: func(service *Service, projectID, componentID string) error {
				return service.AddMLModel(projectID, componentID)
			},
			getComponents: func(project *models.Project) []string { return project.Components.MLModels },
		},
		{
			name:        "digital-twin",
			componentID: "dt-cross-project",
			setupComponent: func(service *Service, projectID, componentID string) error {
				return service.store.SaveDigitalTwin(&models.DigitalTwin{
					ID:         componentID,
					ProjectID:  projectID,
					OntologyID: "ontology",
					Name:       "twin",
					Status:     "active",
					CreatedAt:  time.Now(),
					UpdatedAt:  time.Now(),
				})
			},
			add: func(service *Service, projectID, componentID string) error {
				return service.AddDigitalTwin(projectID, componentID)
			},
			getComponents: func(project *models.Project) []string { return project.Components.DigitalTwins },
		},
		{
			name:        "storage-config",
			componentID: "store-cross-project",
			setupComponent: func(service *Service, projectID, componentID string) error {
				return service.store.SaveStorageConfig(&models.StorageConfig{
					ID:         componentID,
					ProjectID:  projectID,
					PluginType: "filesystem",
					Config: map[string]interface{}{
						"base_path": "/tmp/mimir-tests",
					},
					Active:    true,
					CreatedAt: time.Now().Format(time.RFC3339),
					UpdatedAt: time.Now().Format(time.RFC3339),
				})
			},
			add: func(service *Service, projectID, componentID string) error {
				return service.AddStorage(projectID, componentID)
			},
			getComponents: func(project *models.Project) []string { return project.Components.StorageConfigs },
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			service, tmpDir := setupTestService(t)
			defer cleanupTestService(tmpDir)

			targetProject, err := service.Create(&models.ProjectCreateRequest{Name: fmt.Sprintf("target-%s", tc.name)})
			if err != nil {
				t.Fatalf("failed to create target project: %v", err)
			}
			ownerProject, err := service.Create(&models.ProjectCreateRequest{Name: fmt.Sprintf("owner-%s", tc.name)})
			if err != nil {
				t.Fatalf("failed to create owner project: %v", err)
			}

			if err := tc.setupComponent(service, ownerProject.ID, tc.componentID); err != nil {
				t.Fatalf("failed to save test component: %v", err)
			}

			err = tc.add(service, targetProject.ID, tc.componentID)
			if !errors.Is(err, ErrComponentProjectMismatch) {
				t.Fatalf("expected ErrComponentProjectMismatch, got: %v", err)
			}

			updatedProject, err := service.Get(targetProject.ID)
			if err != nil {
				t.Fatalf("failed to get target project: %v", err)
			}
			if slices.Contains(tc.getComponents(updatedProject), tc.componentID) {
				t.Fatalf("component %s should not be associated with project %s", tc.componentID, targetProject.ID)
			}
		})
	}
}

func TestAddComponentRejectsMissingComponent(t *testing.T) {
	service, tmpDir := setupTestService(t)
	defer cleanupTestService(tmpDir)

	project, err := service.Create(&models.ProjectCreateRequest{Name: "missing-component-project"})
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	err = service.AddPipeline(project.ID, "missing-pipeline")
	if !errors.Is(err, ErrComponentNotFound) {
		t.Fatalf("expected ErrComponentNotFound, got: %v", err)
	}
}
