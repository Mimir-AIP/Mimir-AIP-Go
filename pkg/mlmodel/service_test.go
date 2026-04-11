package mlmodel

import (
	"errors"
	"testing"
	"time"

	"github.com/mimir-aip/mimir-aip-go/pkg/metadatastore"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"github.com/mimir-aip/mimir-aip-go/pkg/ontology"
	"github.com/mimir-aip/mimir-aip-go/pkg/queue"
	"github.com/mimir-aip/mimir-aip-go/pkg/storage"
)

func setupMLService(t *testing.T) (*Service, metadatastore.MetadataStore, func()) {
	t.Helper()
	store, err := metadatastore.NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	q, err := queue.NewQueue(store)
	if err != nil {
		t.Fatalf("failed to create queue: %v", err)
	}
	now := time.Now().UTC()
	for _, project := range []*models.Project{
		{ID: "project-a", Name: "project-a", Description: "A", Version: "v1", Status: models.ProjectStatusActive, Metadata: models.ProjectMetadata{CreatedAt: now, UpdatedAt: now}},
		{ID: "project-b", Name: "project-b", Description: "B", Version: "v1", Status: models.ProjectStatusActive, Metadata: models.ProjectMetadata{CreatedAt: now, UpdatedAt: now}},
	} {
		if err := store.SaveProject(project); err != nil {
			t.Fatalf("failed to save project: %v", err)
		}
	}
	ontSvc := ontology.NewService(store)
	storageSvc := storage.NewService(store)
	service := NewService(store, ontSvc, storageSvc, q)
	return service, store, func() {
		_ = q.Close()
		_ = store.Close()
	}
}

func saveTestOntology(t *testing.T, store metadatastore.MetadataStore, projectID, ontologyID string) {
	t.Helper()
	if err := store.SaveOntology(&models.Ontology{ID: ontologyID, ProjectID: projectID, Name: ontologyID, Content: "@prefix : <http://example.org/mimir#> .\n@prefix owl: <http://www.w3.org/2002/07/owl#> .\n\n:Entity a owl:Class .", Status: "active", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}); err != nil {
		t.Fatalf("failed to save ontology: %v", err)
	}
}

func TestCreateModelRejectsMissingProject(t *testing.T) {
	service, store, cleanup := setupMLService(t)
	defer cleanup()
	saveTestOntology(t, store, "project-a", "ontology-a")

	_, err := service.CreateModel(&models.ModelCreateRequest{ProjectID: "missing-project", OntologyID: "ontology-a", Name: "Model", Type: models.ModelTypeDecisionTree})
	if err == nil {
		t.Fatal("expected create to fail when project does not exist")
	}
}

func TestDeleteModelRejectsReferencedModel(t *testing.T) {
	service, store, cleanup := setupMLService(t)
	defer cleanup()
	saveTestOntology(t, store, "project-a", "ontology-a")

	model, err := service.CreateModel(&models.ModelCreateRequest{ProjectID: "project-a", OntologyID: "ontology-a", Name: "Model", Type: models.ModelTypeDecisionTree})
	if err != nil {
		t.Fatalf("failed to create model: %v", err)
	}
	if err := store.SaveDigitalTwin(&models.DigitalTwin{ID: "twin-1", ProjectID: "project-a", OntologyID: "ontology-a", Name: "Twin", Status: "active", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}); err != nil {
		t.Fatalf("failed to save twin: %v", err)
	}
	if err := store.SaveAction(&models.Action{ID: "action-1", DigitalTwinID: "twin-1", Name: "Action", Enabled: true, Condition: &models.ActionCondition{ModelID: model.ID, Operator: "gt", Threshold: 0.8}, Trigger: &models.ActionTrigger{PipelineID: "pipeline-1"}, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}); err != nil {
		t.Fatalf("failed to save action: %v", err)
	}

	err = service.DeleteModel(model.ID)
	var inUseErr *ModelInUseError
	if !errors.As(err, &inUseErr) {
		t.Fatalf("expected ModelInUseError, got %v", err)
	}
}
