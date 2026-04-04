package metadatastore

import (
	"strings"
	"testing"
	"time"

	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

func TestNewSQLiteStoreEnablesForeignKeys(t *testing.T) {
	store, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create sqlite store: %v", err)
	}
	defer store.Close()

	project := &models.Project{
		ID:          "project-1",
		Name:        "Project 1",
		Description: "test project",
		Version:     "v1",
		Status:      models.ProjectStatusActive,
		Metadata: models.ProjectMetadata{
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		},
	}
	if err := store.SaveProject(project); err != nil {
		t.Fatalf("failed to save project: %v", err)
	}

	pipeline := &models.Pipeline{
		ID:          "pipeline-1",
		ProjectID:   project.ID,
		Name:        "Pipeline 1",
		Type:        models.PipelineTypeIngestion,
		Description: "test pipeline",
		Status:      models.PipelineStatusActive,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	if err := store.SavePipeline(pipeline); err != nil {
		t.Fatalf("failed to save pipeline: %v", err)
	}

	if err := store.DeleteProject(project.ID); err == nil {
		t.Fatal("expected deleting a project with dependent pipelines to fail when foreign keys are enabled")
	} else if !strings.Contains(err.Error(), "FOREIGN KEY") {
		t.Fatalf("expected foreign key error, got %v", err)
	}
}

func TestSavePipelineRejectsMissingProject(t *testing.T) {
	store, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create sqlite store: %v", err)
	}
	defer store.Close()

	pipeline := &models.Pipeline{
		ID:          "pipeline-orphan",
		ProjectID:   "missing-project",
		Name:        "Orphan Pipeline",
		Type:        models.PipelineTypeIngestion,
		Description: "should fail",
		Status:      models.PipelineStatusActive,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	if err := store.SavePipeline(pipeline); err == nil {
		t.Fatal("expected foreign key error for pipeline referencing a missing project")
	} else if !strings.Contains(err.Error(), "FOREIGN KEY") {
		t.Fatalf("expected foreign key error, got %v", err)
	}
}
