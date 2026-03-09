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

	// Create a project
	req := &models.ProjectCreateRequest{
		Name:        "original-project",
		Description: "Original description",
	}

	original, err := service.Create(req)
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// Clone the project
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

	expectedDesc := original.Description + " (cloned)"
	if cloned.Description != expectedDesc {
		t.Errorf("Expected description %s, got %s", expectedDesc, cloned.Description)
	}

	if cloned.Status != models.ProjectStatusDraft {
		t.Errorf("Expected status %s, got %s", models.ProjectStatusDraft, cloned.Status)
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
