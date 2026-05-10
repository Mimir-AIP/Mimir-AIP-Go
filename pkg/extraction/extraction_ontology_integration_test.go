package extraction

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/mimir-aip/mimir-aip-go/pkg/metadatastore"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"github.com/mimir-aip/mimir-aip-go/pkg/ontology"
	"github.com/mimir-aip/mimir-aip-go/pkg/storage"
	storageplugins "github.com/mimir-aip/mimir-aip-go/pkg/storage/plugins"
)

func TestExtractFromStorageFeedsGeneratedOntologyWithCrossSourceLinks(t *testing.T) {
	store, err := metadatastore.NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	now := time.Now().UTC()
	project := &models.Project{
		ID:          "project-extraction",
		Name:        "Extraction Project",
		Description: "tests extraction into ontology",
		Version:     "v1",
		Status:      models.ProjectStatusActive,
		Metadata:    models.ProjectMetadata{CreatedAt: now, UpdatedAt: now},
	}
	if err := store.SaveProject(project); err != nil {
		t.Fatalf("failed to save project: %v", err)
	}

	storageSvc := storage.NewService(store)
	storageSvc.RegisterPlugin("filesystem", storageplugins.NewFilesystemPlugin())
	studentsCfg, err := storageSvc.CreateStorageConfig(project.ID, "filesystem", map[string]interface{}{"options": map[string]interface{}{"base_path": t.TempDir()}})
	if err != nil {
		t.Fatalf("failed to create students storage config: %v", err)
	}
	gradesCfg, err := storageSvc.CreateStorageConfig(project.ID, "filesystem", map[string]interface{}{"options": map[string]interface{}{"base_path": t.TempDir()}})
	if err != nil {
		t.Fatalf("failed to create grades storage config: %v", err)
	}

	students := models.NewCIR(models.SourceTypeDatabase, "db://school/students", models.DataFormatJSON, []interface{}{
		map[string]interface{}{"student_id": 1001, "name": "Ada Lovelace", "cohort": "A"},
		map[string]interface{}{"student_id": 1002, "name": "Grace Hopper", "cohort": "A"},
		map[string]interface{}{"student_id": 1003, "name": "Katherine Johnson", "cohort": "B"},
	})
	students.SetParameter("entity_type", "Student")
	if _, err := storageSvc.StoreForProject(project.ID, studentsCfg.ID, students); err != nil {
		t.Fatalf("failed to store students: %v", err)
	}

	grades := models.NewCIR(models.SourceTypeDatabase, "db://school/grades", models.DataFormatJSON, []interface{}{
		map[string]interface{}{"grade_id": 1, "student_id": 1001, "score": 91.0, "subject": "math"},
		map[string]interface{}{"grade_id": 2, "student_id": 1002, "score": 87.0, "subject": "math"},
		map[string]interface{}{"grade_id": 3, "student_id": 1003, "score": 95.0, "subject": "physics"},
		map[string]interface{}{"grade_id": 4, "student_id": 1001, "score": 89.0, "subject": "physics"},
	})
	grades.SetParameter("entity_type", "Grade")
	if _, err := storageSvc.StoreForProject(project.ID, gradesCfg.ID, grades); err != nil {
		t.Fatalf("failed to store grades: %v", err)
	}

	result, err := NewService(storageSvc).ExtractFromStorage(project.ID, []string{studentsCfg.ID, gradesCfg.ID}, true, false)
	if err != nil {
		t.Fatalf("ExtractFromStorage failed: %v", err)
	}
	if len(result.Entities) != 7 {
		t.Fatalf("expected 7 row entities after extraction, got %d: %+v", len(result.Entities), result.Entities)
	}
	if len(result.CrossSourceLinks) == 0 {
		t.Fatalf("expected cross-source student_id link, got none")
	}
	best := result.CrossSourceLinks[0]
	if best.ColumnA != "student_id" || best.ColumnB != "student_id" {
		t.Fatalf("expected strongest cross-source link on student_id, got %+v", best)
	}

	ontologySvc := ontology.NewService(store)
	generated, err := ontologySvc.GenerateFromExtraction(project.ID, "Generated School Ontology", result)
	if err != nil {
		t.Fatalf("GenerateFromExtraction failed: %v", err)
	}
	content := generated.Content
	for _, required := range []string{":Student a owl:Class", ":Grade a owl:Class", ":score a owl:DatatypeProperty", ":studentIdLinksStudentId a owl:ObjectProperty", ":crossSourceLink \"true\"^^xsd:boolean"} {
		if !strings.Contains(content, required) {
			t.Fatalf("generated ontology missing %q:\n%s", required, content)
		}
	}
	compiled, err := store.GetCompiledOntology(generated.ID)
	if err != nil {
		t.Fatalf("expected generated ontology to compile: %v", err)
	}
	if len(compiled.Classes) < 2 {
		t.Fatalf("expected compiled classes for generated ontology, got %+v", compiled.Classes)
	}
}

func TestExtractFromStoragePagesBeyondFirstThousand(t *testing.T) {
	store, err := metadatastore.NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	now := time.Now().UTC()
	project := &models.Project{
		ID:          "project-paginated-extraction",
		Name:        "Paginated Extraction Project",
		Description: "tests extraction pagination",
		Version:     "v1",
		Status:      models.ProjectStatusActive,
		Metadata:    models.ProjectMetadata{CreatedAt: now, UpdatedAt: now},
	}
	if err := store.SaveProject(project); err != nil {
		t.Fatalf("failed to save project: %v", err)
	}

	storageSvc := storage.NewService(store)
	storageSvc.RegisterPlugin("filesystem", storageplugins.NewFilesystemPlugin())
	cfg, err := storageSvc.CreateStorageConfig(project.ID, "filesystem", map[string]interface{}{"options": map[string]interface{}{"base_path": t.TempDir()}})
	if err != nil {
		t.Fatalf("failed to create storage config: %v", err)
	}

	rows := make([]interface{}, 1005)
	for i := range rows {
		rows[i] = map[string]interface{}{"reading_id": i + 1, "value": float64(i)}
	}
	cir := models.NewCIR(models.SourceTypeDatabase, "db://sensor/readings", models.DataFormatJSON, rows)
	cir.SetParameter("entity_type", "Reading")
	if _, err := storageSvc.StoreForProject(project.ID, cfg.ID, cir); err != nil {
		t.Fatalf("failed to store readings: %v", err)
	}

	result, err := NewService(storageSvc).ExtractFromStorage(project.ID, []string{cfg.ID}, true, false)
	if err != nil {
		t.Fatalf("ExtractFromStorage failed: %v", err)
	}
	if len(result.Entities) != len(rows) {
		t.Fatalf("expected extraction to page through %d entities, got %d", len(rows), len(result.Entities))
	}
}

type failingExtractionStoragePlugin struct{}

func (f failingExtractionStoragePlugin) Initialize(config *models.PluginConfig) error { return nil }
func (f failingExtractionStoragePlugin) CreateSchema(ontology *models.OntologyDefinition) error {
	return nil
}
func (f failingExtractionStoragePlugin) Store(cir *models.CIR) (*models.StorageResult, error) {
	return &models.StorageResult{Success: true, AffectedItems: 1}, nil
}
func (f failingExtractionStoragePlugin) Retrieve(query *models.CIRQuery) ([]*models.CIR, error) {
	return nil, fmt.Errorf("storage unavailable")
}
func (f failingExtractionStoragePlugin) Update(query *models.CIRQuery, updates *models.CIRUpdate) (*models.StorageResult, error) {
	return &models.StorageResult{Success: true, AffectedItems: 0}, nil
}
func (f failingExtractionStoragePlugin) Delete(query *models.CIRQuery) (*models.StorageResult, error) {
	return &models.StorageResult{Success: true, AffectedItems: 0}, nil
}
func (f failingExtractionStoragePlugin) GetMetadata() (*models.StorageMetadata, error) {
	return &models.StorageMetadata{StorageType: "failing-extraction"}, nil
}
func (f failingExtractionStoragePlugin) HealthCheck() (bool, error) {
	return false, fmt.Errorf("storage unavailable")
}

func TestExtractFromStorageReturnsRetrievalFailure(t *testing.T) {
	store, err := metadatastore.NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	now := time.Now().UTC()
	project := &models.Project{
		ID:          "project-failing-extraction",
		Name:        "Failing Extraction Project",
		Description: "tests extraction retrieval failure",
		Version:     "v1",
		Status:      models.ProjectStatusActive,
		Metadata:    models.ProjectMetadata{CreatedAt: now, UpdatedAt: now},
	}
	if err := store.SaveProject(project); err != nil {
		t.Fatalf("failed to save project: %v", err)
	}

	storageSvc := storage.NewService(store)
	storageSvc.RegisterPlugin("failing-extraction", failingExtractionStoragePlugin{})
	cfg, err := storageSvc.CreateStorageConfig(project.ID, "failing-extraction", map[string]interface{}{"connection_string": "mock://failing"})
	if err != nil {
		t.Fatalf("failed to create storage config: %v", err)
	}

	_, err = NewService(storageSvc).ExtractFromStorage(project.ID, []string{cfg.ID}, true, false)
	if err == nil {
		t.Fatal("expected retrieval failure")
	}
	if !strings.Contains(err.Error(), "retrieve extraction data") {
		t.Fatalf("expected contextual retrieval error, got %v", err)
	}
}
