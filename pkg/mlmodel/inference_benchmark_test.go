package mlmodel

import (
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/mimir-aip/mimir-aip-go/pkg/metadatastore"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"github.com/mimir-aip/mimir-aip-go/pkg/ontology"
	"github.com/mimir-aip/mimir-aip-go/pkg/queue"
	"github.com/mimir-aip/mimir-aip-go/pkg/storage"
)

func BenchmarkBuiltinRegressionInferModel(b *testing.B) {
	store, err := metadatastore.NewSQLiteStore(filepath.Join(b.TempDir(), "infer.db"))
	if err != nil {
		b.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()
	q, err := queue.NewQueue(store)
	if err != nil {
		b.Fatalf("failed to create queue: %v", err)
	}
	defer q.Close()

	now := time.Now().UTC()
	project := &models.Project{ID: "project-infer", Name: "project-infer", Version: "v1", Status: models.ProjectStatusActive, Metadata: models.ProjectMetadata{CreatedAt: now, UpdatedAt: now}}
	if err := store.SaveProject(project); err != nil {
		b.Fatalf("failed to save project: %v", err)
	}
	ontologyRecord := &models.Ontology{ID: "ontology-infer", ProjectID: project.ID, Name: "ontology-infer", Content: "@prefix : <http://example.org/mimir#> .\n@prefix owl: <http://www.w3.org/2002/07/owl#> .\n\n:Reading a owl:Class .", Status: "active", CreatedAt: now, UpdatedAt: now}
	if err := store.SaveOntology(ontologyRecord); err != nil {
		b.Fatalf("failed to save ontology: %v", err)
	}

	svc := NewService(store, ontology.NewService(store), storage.NewService(store), q)
	model, err := svc.CreateModel(&models.ModelCreateRequest{ProjectID: project.ID, OntologyID: ontologyRecord.ID, Name: "regression", Type: models.ModelTypeRegression})
	if err != nil {
		b.Fatalf("failed to create model: %v", err)
	}
	artifact, err := json.Marshal(map[string]any{
		"provider":       "builtin",
		"provider_model": "regression",
		"feature_names":  []string{"temperature", "humidity", "pressure"},
		"parameters": map[string]any{
			"model_data": map[string]any{
				"intercept":    10.0,
				"coefficients": []float64{0.5, -0.25, 0.01},
			},
		},
	})
	if err != nil {
		b.Fatalf("failed to marshal artifact: %v", err)
	}
	if err := svc.CompleteTraining(model.ID, artifact, &models.PerformanceMetrics{R2Score: 0.95}); err != nil {
		b.Fatalf("failed to complete training: %v", err)
	}

	input := map[string]any{"temperature": 22.0, "humidity": 60.0, "pressure": 1010.0}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		prediction, confidence, err := svc.InferModel(model.ID, input)
		if err != nil {
			b.Fatalf("InferModel failed: %v", err)
		}
		if prediction == nil || confidence <= 0 {
			b.Fatalf("invalid inference result prediction=%v confidence=%v", prediction, confidence)
		}
	}
}
