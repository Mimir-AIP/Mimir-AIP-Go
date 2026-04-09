package metadatastore

import (
	"testing"
	"time"

	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

func TestSQLiteStoreResetAllClearsMetadataRows(t *testing.T) {
	store, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create sqlite store: %v", err)
	}
	defer store.Close()

	now := time.Now().UTC()
	project := &models.Project{
		ID:          "project-1",
		Name:        "Project 1",
		Description: "test project",
		Version:     "v1",
		Status:      models.ProjectStatusActive,
		Metadata:    models.ProjectMetadata{CreatedAt: now, UpdatedAt: now},
	}
	if err := store.SaveProject(project); err != nil {
		t.Fatalf("failed to save project: %v", err)
	}
	pipeline := &models.Pipeline{
		ID:          "pipeline-1",
		ProjectID:   project.ID,
		Name:        "Pipeline 1",
		Type:        models.PipelineTypeProcessing,
		Description: "test pipeline",
		Steps:       []models.PipelineStep{{Name: "step-1", Plugin: "default", Action: "set_context"}},
		Status:      models.PipelineStatusActive,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := store.SavePipeline(pipeline); err != nil {
		t.Fatalf("failed to save pipeline: %v", err)
	}

	if err := store.ResetAll(); err != nil {
		t.Fatalf("failed to reset metadata store: %v", err)
	}

	projects, err := store.ListProjects()
	if err != nil {
		t.Fatalf("failed to list projects after reset: %v", err)
	}
	if len(projects) != 0 {
		t.Fatalf("expected no projects after reset, got %d", len(projects))
	}
	pipelines, err := store.ListPipelines()
	if err != nil {
		t.Fatalf("failed to list pipelines after reset: %v", err)
	}
	if len(pipelines) != 0 {
		t.Fatalf("expected no pipelines after reset, got %d", len(pipelines))
	}
}
