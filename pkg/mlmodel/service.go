package mlmodel

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/mimir-aip/mimir-aip-go/pkg/metadatastore"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"github.com/mimir-aip/mimir-aip-go/pkg/ontology"
	"github.com/mimir-aip/mimir-aip-go/pkg/pluginruntime"
	"github.com/mimir-aip/mimir-aip-go/pkg/queue"
	"github.com/mimir-aip/mimir-aip-go/pkg/storage"
	"maps"
	"sort"
	"strings"
	"time"
)

// Service manages ML models and training
type Service struct {
	store                metadatastore.MetadataStore
	ontologyService      *ontology.Service
	storageService       *storage.Service
	queue                *queue.Queue
	recommendationEngine *RecommendationEngine
	providers            *ProviderRegistry
	providerLoader       *pluginruntime.Loader[Provider]
}

// NewService creates a new ML model service
func NewService(
	store metadatastore.MetadataStore,
	ontologyService *ontology.Service,
	storageService *storage.Service,
	q *queue.Queue,
) *Service {
	providers := NewProviderRegistry()
	providers.Register("builtin", NewBuiltinProvider())
	return &Service{
		store:                store,
		ontologyService:      ontologyService,
		storageService:       storageService,
		queue:                q,
		recommendationEngine: NewRecommendationEngine(),
		providers:            providers,
	}
}

type ModelProjectMismatchError struct {
	ModelID           string
	ExpectedProjectID string
	ActualProjectID   string
}

func (e *ModelProjectMismatchError) Error() string {
	return fmt.Sprintf("ml model %s belongs to project %s, not %s", e.ModelID, e.ActualProjectID, e.ExpectedProjectID)
}

type ModelInUseError struct {
	ModelID    string
	References []string
}

func (e *ModelInUseError) Error() string {
	return fmt.Sprintf("ml model %s is still referenced by %s", e.ModelID, strings.Join(e.References, ", "))
}

func (s *Service) ensureProjectExists(projectID string) error {
	if projectID == "" {
		return fmt.Errorf("project_id is required")
	}
	if _, err := s.store.GetProject(projectID); err != nil {
		return fmt.Errorf("project not found: %w", err)
	}
	return nil
}

func (s *Service) getOwnedModel(projectID, modelID string) (*models.MLModel, error) {
	if projectID == "" {
		return nil, fmt.Errorf("project_id is required")
	}
	model, err := s.store.GetMLModel(modelID)
	if err != nil {
		return nil, fmt.Errorf("model not found: %w", err)
	}
	if model.ProjectID != projectID {
		return nil, &ModelProjectMismatchError{ModelID: modelID, ExpectedProjectID: projectID, ActualProjectID: model.ProjectID}
	}
	return model, nil
}

func (s *Service) resolveProviderMetadata(providerName string) (models.MLProviderMetadata, error) {
	provider, ok := s.providers.Get(providerName)
	if ok {
		return provider.Metadata(), nil
	}
	plugin, err := s.store.GetPlugin(providerName)
	if err != nil {
		return models.MLProviderMetadata{}, fmt.Errorf("provider not found: %w", err)
	}
	if plugin.PluginDefinition.MLProvider == nil {
		return models.MLProviderMetadata{}, fmt.Errorf("plugin %s does not declare an ML provider", providerName)
	}
	return *plugin.PluginDefinition.MLProvider, nil
}

func (s *Service) resolveProviderForModel(model *models.MLModel) (Provider, error) {
	providerName, _, err := normalizeProviderIdentity(model)
	if err != nil {
		return nil, err
	}
	provider, ok := s.providers.Get(providerName)
	if ok {
		return provider, nil
	}
	return s.loadExternalProvider(providerName)
}

func (s *Service) normalizeModelDefinition(req *models.ModelCreateRequest) (string, string, error) {
	provider := req.Provider
	providerModel := req.ProviderModel
	if provider == "" {
		provider = "builtin"
	}
	if provider == "builtin" {
		if providerModel == "" {
			providerModel = string(req.Type)
		}
		if providerModel == "" {
			return "", "", fmt.Errorf("type is required for builtin provider")
		}
		req.Type = models.ModelType(providerModel)
	}
	metadata, err := s.resolveProviderMetadata(provider)
	if err != nil {
		return "", "", err
	}
	if providerModel == "" {
		return "", "", fmt.Errorf("provider_model is required")
	}
	for _, candidate := range metadata.Models {
		if candidate.Name == providerModel {
			return provider, providerModel, nil
		}
	}
	return "", "", fmt.Errorf("provider %s does not support model %s", provider, providerModel)
}

func (s *Service) ListProviderMetadata() ([]models.MLProviderMetadata, error) {
	providers := make([]models.MLProviderMetadata, 0)
	seen := make(map[string]bool)
	for _, name := range s.providers.Names() {
		provider, ok := s.providers.Get(name)
		if !ok {
			continue
		}
		metadata := provider.Metadata()
		providers = append(providers, metadata)
		seen[metadata.Name] = true
	}
	pluginsList, err := s.store.ListPlugins()
	if err != nil {
		return nil, fmt.Errorf("failed to list plugins: %w", err)
	}
	for _, plugin := range pluginsList {
		if plugin == nil || plugin.PluginDefinition.MLProvider == nil {
			continue
		}
		metadata := *plugin.PluginDefinition.MLProvider
		if seen[metadata.Name] {
			continue
		}
		providers = append(providers, metadata)
		seen[metadata.Name] = true
	}
	sort.Slice(providers, func(i, j int) bool { return providers[i].Name < providers[j].Name })
	return providers, nil
}

func (s *Service) GetProviderMetadata(name string) (*models.MLProviderMetadata, error) {
	metadata, err := s.resolveProviderMetadata(name)
	if err != nil {
		return nil, err
	}
	return &metadata, nil
}

func (s *Service) GetModelForProject(projectID, modelID string) (*models.MLModel, error) {
	return s.getOwnedModel(projectID, modelID)
}

// CreateModel creates a new ML model
func (s *Service) CreateModel(req *models.ModelCreateRequest) (*models.MLModel, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	if err := s.ensureProjectExists(req.ProjectID); err != nil {
		return nil, err
	}
	if _, err := s.ontologyService.GetOntologyForProject(req.ProjectID, req.OntologyID); err != nil {
		return nil, fmt.Errorf("failed to get ontology: %w", err)
	}
	providerName, providerModel, err := s.normalizeModelDefinition(req)
	if err != nil {
		return nil, err
	}
	model := &models.MLModel{
		ID:             uuid.New().String(),
		ProjectID:      req.ProjectID,
		OntologyID:     req.OntologyID,
		Name:           req.Name,
		Description:    req.Description,
		Type:           req.Type,
		Provider:       providerName,
		ProviderModel:  providerModel,
		ProviderConfig: req.ProviderConfig,
		Status:         models.ModelStatusDraft,
		Version:        "1.0",
		TrainingConfig: req.TrainingConfig,
		Metadata:       req.Metadata,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}
	if model.TrainingConfig == nil {
		model.TrainingConfig = &models.TrainingConfig{TrainTestSplit: 0.8, RandomSeed: 42}
	}
	if err := s.store.SaveMLModel(model); err != nil {
		return nil, fmt.Errorf("failed to save model: %w", err)
	}
	return model, nil
}

// GetModel retrieves an ML model by ID
func (s *Service) GetModel(id string) (*models.MLModel, error) {
	model, err := s.store.GetMLModel(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get model: %w", err)
	}
	return model, nil
}

// UpdateModel updates an existing ML model
func (s *Service) UpdateModel(id string, req *models.ModelUpdateRequest) (*models.MLModel, error) {
	model, err := s.store.GetMLModel(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get model: %w", err)
	}

	// Update fields
	if req.Name != nil {
		model.Name = *req.Name
	}
	if req.Description != nil {
		model.Description = *req.Description
	}
	if req.Status != nil {
		model.Status = *req.Status
	}
	if req.TrainingMetrics != nil {
		model.TrainingMetrics = req.TrainingMetrics
	}
	if req.PerformanceMetrics != nil {
		model.PerformanceMetrics = req.PerformanceMetrics
	}
	if req.Metadata != nil {
		if model.Metadata == nil {
			model.Metadata = make(map[string]any)
		}
		maps.Copy(model.Metadata, req.Metadata)
	}

	model.UpdatedAt = time.Now().UTC()

	if err := s.store.SaveMLModel(model); err != nil {
		return nil, fmt.Errorf("failed to update model: %w", err)
	}

	return model, nil
}

func (s *Service) UpdateModelForProject(projectID, id string, req *models.ModelUpdateRequest) (*models.MLModel, error) {
	if _, err := s.getOwnedModel(projectID, id); err != nil {
		return nil, err
	}
	return s.UpdateModel(id, req)
}

// DeleteModel deletes an ML model
func (s *Service) DeleteModel(id string) error {
	references, err := s.findModelReferences(id)
	if err != nil {
		return err
	}
	if len(references) > 0 {
		return &ModelInUseError{ModelID: id, References: references}
	}
	if err := s.store.DeleteMLModel(id); err != nil {
		return fmt.Errorf("failed to delete model: %w", err)
	}
	return nil
}

func (s *Service) DeleteModelForProject(projectID, id string) error {
	if _, err := s.getOwnedModel(projectID, id); err != nil {
		return err
	}
	return s.DeleteModel(id)
}

// ListProjectModels lists all models for a project
func (s *Service) ListProjectModels(projectID string) ([]*models.MLModel, error) {
	models, err := s.store.ListMLModelsByProject(projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list models: %w", err)
	}
	return models, nil
}

func (s *Service) findModelReferences(modelID string) ([]string, error) {
	references := make([]string, 0)
	twins, err := s.store.ListDigitalTwins()
	if err != nil {
		return nil, fmt.Errorf("failed to list digital twins: %w", err)
	}
	for _, twin := range twins {
		if twin == nil {
			continue
		}
		actions, err := s.store.ListActionsByDigitalTwin(twin.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to list actions for twin %s: %w", twin.ID, err)
		}
		for _, action := range actions {
			if action != nil && action.Condition != nil && action.Condition.ModelID == modelID {
				references = append(references, fmt.Sprintf("digital twin action %s", action.ID))
			}
		}
		predictions, err := s.store.ListPredictionsByDigitalTwin(twin.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to list predictions for twin %s: %w", twin.ID, err)
		}
		for _, prediction := range predictions {
			if prediction != nil && prediction.ModelID == modelID {
				references = append(references, fmt.Sprintf("prediction %s", prediction.ID))
			}
		}
	}
	tasks, err := s.store.ListWorkTasks()
	if err != nil {
		return nil, fmt.Errorf("failed to list work tasks: %w", err)
	}
	for _, task := range tasks {
		if task == nil || task.TaskSpec.ModelID != modelID {
			continue
		}
		switch task.Status {
		case models.WorkTaskStatusQueued, models.WorkTaskStatusScheduled, models.WorkTaskStatusSpawned, models.WorkTaskStatusExecuting:
			references = append(references, fmt.Sprintf("work task %s", task.ID))
		}
	}
	return references, nil
}
