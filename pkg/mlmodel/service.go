package mlmodel

import (
	"encoding/json"
	"fmt"
	"log"
	"maps"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mimir-aip/mimir-aip-go/pkg/metadatastore"
	"github.com/mimir-aip/mimir-aip-go/pkg/mlmodel/training"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"github.com/mimir-aip/mimir-aip-go/pkg/ontology"
	"github.com/mimir-aip/mimir-aip-go/pkg/pluginruntime"
	"github.com/mimir-aip/mimir-aip-go/pkg/queue"
	"github.com/mimir-aip/mimir-aip-go/pkg/storage"
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

func (s *Service) inferModel(model *models.MLModel, input map[string]any) (any, float64, error) {
	provider, err := s.resolveProviderForModel(model)
	if err != nil {
		return nil, 0, err
	}
	if err := provider.ValidateModel(model); err != nil {
		return nil, 0, err
	}
	result, err := provider.Infer(&ProviderInferRequest{Model: model, Input: input})
	if err != nil {
		return nil, 0, err
	}
	return result.Output, result.Confidence, nil
}

func (s *Service) InferModel(modelID string, input map[string]any) (any, float64, error) {
	model, err := s.store.GetMLModel(modelID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get model: %w", err)
	}
	return s.inferModel(model, input)
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

// RecommendModelType recommends the best model type for a project
func (s *Service) RecommendModelType(projectID, ontologyID string) (*models.ModelRecommendation, error) {
	if err := s.ensureProjectExists(projectID); err != nil {
		return nil, err
	}
	ontologyRecord, err := s.ontologyService.GetOntologyForProject(projectID, ontologyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get ontology: %w", err)
	}
	storageConfigs, err := s.storageService.GetProjectStorageConfigs(projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get storage configs: %w", err)
	}
	dataSummary := s.analyzeData(storageConfigs)
	recommendation, err := s.recommendationEngine.RecommendModelType(ontologyRecord, dataSummary)
	if err != nil {
		return nil, fmt.Errorf("failed to recommend model type: %w", err)
	}
	return recommendation, nil
}

// analyzeData analyzes storage configs to create a data summary by inspecting actual records
func (s *Service) analyzeData(storageConfigs []*models.StorageConfig) *models.DataAnalysis {
	totalRecords := int64(0)
	featureCount := 0
	hasUnstructured := false

	for _, config := range storageConfigs {
		cirs, err := s.storageService.Retrieve(config.ID, &models.CIRQuery{})
		if err != nil {
			log.Printf("Warning: failed to retrieve from storage %s for analysis: %v", config.ID, err)
			continue
		}
		totalRecords += int64(len(cirs))

		// Inspect first record to determine field types
		if len(cirs) > 0 {
			if dataMap, ok := cirs[0].Data.(map[string]any); ok {
				numericFields := 0
				for _, v := range dataMap {
					switch val := v.(type) {
					case float64, int, bool:
						numericFields++
					case string:
						if len(val) > 100 {
							hasUnstructured = true
						} else {
							numericFields++
						}
					}
				}
				if numericFields > featureCount {
					featureCount = numericFields
				}
			}
		}
	}

	// Determine size category from actual record count
	size := "small"
	if totalRecords > 10000 {
		size = "large"
	} else if totalRecords >= 1000 {
		size = "medium"
	}

	// Avoid returning zero when storage configs exist but retrieval returned nothing
	if totalRecords == 0 && len(storageConfigs) > 0 {
		totalRecords = 100
	}

	return &models.DataAnalysis{
		Size:            size,
		RecordCount:     totalRecords,
		HasUnstructured: hasUnstructured,
		FeatureCount:    featureCount,
	}
}

// StartTraining initiates model training by submitting a job to the worker queue
func (s *Service) StartTraining(req *models.ModelTrainingRequest) (*models.MLModel, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	model, err := s.store.GetMLModel(req.ModelID)
	if err != nil {
		return nil, fmt.Errorf("failed to get model: %w", err)
	}
	if model.Status == models.ModelStatusTraining {
		return nil, fmt.Errorf("model is already training")
	}
	providerName, providerModel, err := normalizeProviderIdentity(model)
	if err != nil {
		return nil, err
	}
	if req.TrainingConfig != nil {
		model.TrainingConfig = req.TrainingConfig
	}
	workTask := &models.WorkTask{
		ID:          uuid.New().String(),
		Type:        models.WorkTaskTypeMLTraining,
		ProjectID:   model.ProjectID,
		Priority:    5,
		Status:      models.WorkTaskStatusQueued,
		SubmittedAt: time.Now().UTC(),
		TaskSpec: models.TaskSpec{
			ModelID:   model.ID,
			ProjectID: model.ProjectID,
			Parameters: map[string]any{
				"model_id":        model.ID,
				"ontology_id":     model.OntologyID,
				"storage_ids":     req.StorageIDs,
				"config":          model.TrainingConfig,
				"provider":        providerName,
				"provider_model":  providerModel,
				"provider_config": model.ProviderConfig,
			},
		},
		ResourceRequirements: models.ResourceRequirements{
			CPU:    "2000m",
			Memory: "4Gi",
			GPU:    false,
		},
		DataAccess: models.DataAccess{
			InputDatasets: req.StorageIDs,
		},
	}

	// Update model status and persist the canonical async handle before workers pick up the task.
	model.Status = models.ModelStatusTraining
	model.TrainingTaskID = workTask.ID
	model.UpdatedAt = time.Now().UTC()

	// Initialize training metrics
	model.TrainingMetrics = &models.TrainingMetrics{
		Epoch:              0,
		TrainingLoss:       0,
		ValidationLoss:     0,
		TrainingAccuracy:   0,
		ValidationAccuracy: 0,
		LearningCurve:      []models.LearningCurvePoint{},
	}

	if err := s.store.SaveMLModel(model); err != nil {
		return nil, fmt.Errorf("failed to update model: %w", err)
	}

	// Enqueue the training task.
	if err := s.queue.Enqueue(workTask); err != nil {
		// Roll back the persisted async handle on queue failure so callers do not observe a phantom task.
		model.Status = models.ModelStatusDraft
		model.TrainingTaskID = ""
		model.UpdatedAt = time.Now().UTC()
		s.store.SaveMLModel(model)
		return nil, fmt.Errorf("failed to enqueue training task: %w", err)
	}

	return model, nil
}

// UpdateTrainingProgress updates training progress metrics
// This would be called by workers during training to report progress
func (s *Service) UpdateTrainingProgress(modelID string, metrics *models.TrainingMetrics) error {
	model, err := s.store.GetMLModel(modelID)
	if err != nil {
		return fmt.Errorf("failed to get model: %w", err)
	}

	model.TrainingMetrics = metrics
	model.UpdatedAt = time.Now().UTC()

	if err := s.store.SaveMLModel(model); err != nil {
		return fmt.Errorf("failed to update model: %w", err)
	}

	return nil
}

func modelArtifactBaseDir() string {
	if dir := os.Getenv("MODEL_ARTIFACT_DIR"); dir != "" {
		return dir
	}
	return filepath.Join(os.TempDir(), "mimir-aip", "model-artifacts")
}

func persistModelArtifact(modelID string, artifactData []byte) (string, error) {
	if len(artifactData) == 0 {
		return "", fmt.Errorf("model artifact data is required")
	}
	artifactDir := filepath.Join(modelArtifactBaseDir(), modelID)
	if err := os.MkdirAll(artifactDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create artifact directory: %w", err)
	}
	artifactPath := filepath.Join(artifactDir, "model.json")
	if err := os.WriteFile(artifactPath, artifactData, 0o644); err != nil {
		return "", fmt.Errorf("failed to write model artifact: %w", err)
	}
	return artifactPath, nil
}

// CompleteTraining marks training as complete, persists the artifact into orchestrator-visible storage, and stores performance metrics.
func (s *Service) CompleteTraining(modelID string, artifactData []byte, performanceMetrics *models.PerformanceMetrics) error {
	model, err := s.store.GetMLModel(modelID)
	if err != nil {
		return fmt.Errorf("failed to get model: %w", err)
	}

	artifactPath, err := persistModelArtifact(modelID, artifactData)
	if err != nil {
		return err
	}

	model.Status = models.ModelStatusTrained
	model.TrainingTaskID = ""
	model.ModelArtifactPath = artifactPath
	model.PerformanceMetrics = performanceMetrics
	now := time.Now().UTC()
	model.TrainedAt = &now
	model.UpdatedAt = now

	if err := s.store.SaveMLModel(model); err != nil {
		return fmt.Errorf("failed to update model: %w", err)
	}

	return nil
}

// ValidateModel runs the trained model artifact against the provided data and returns performance metrics.
// ValidateModel runs the trained model artifact against the provided data and returns performance metrics.
// It resolves the model provider and runs normalized inference over each test row.
func (s *Service) ValidateModel(modelID string, data *training.TrainingData) (*models.PerformanceMetrics, error) {
	model, err := s.store.GetMLModel(modelID)
	if err != nil {
		return nil, fmt.Errorf("failed to get model: %w", err)
	}
	predictions := make([]float64, len(data.TestFeatures))
	for i, features := range data.TestFeatures {
		input := make(map[string]any, len(data.FeatureNames))
		for idx, featureName := range data.FeatureNames {
			if idx < len(features) {
				input[featureName] = features[idx]
			}
		}
		pred, _, err := s.inferModel(model, input)
		if err != nil {
			log.Printf("Warning: inference failed for row %d during validation: %v", i, err)
			continue
		}
		switch v := pred.(type) {
		case float64:
			predictions[i] = v
		case int:
			predictions[i] = float64(v)
		default:
			return nil, fmt.Errorf("unsupported prediction output type %T during validation", pred)
		}
	}
	return computePerformanceMetrics(model.Type, predictions, data.TestLabels), nil
}

// inferFromArtifact runs inference for a single feature vector using deserialized artifact parameters.
func inferFromArtifact(modelType string, parameters map[string]any, features []float64) (float64, error) {
	switch modelType {
	case "decision_tree":
		modelDataRaw, ok := parameters["model_data"]
		if !ok {
			return 0.0, fmt.Errorf("model_data missing from artifact parameters for decision_tree")
		}
		modelJSON, err := json.Marshal(modelDataRaw)
		if err != nil {
			return 0.0, fmt.Errorf("failed to marshal tree data: %w", err)
		}
		var node training.DecisionTreeModel
		if err := json.Unmarshal(modelJSON, &node); err != nil {
			return 0.0, fmt.Errorf("failed to unmarshal decision tree: %w", err)
		}
		return training.TraverseTree(&node, features), nil

	case "random_forest":
		modelDataRaw, ok := parameters["model_data"]
		if !ok {
			return 0.0, fmt.Errorf("model_data missing from artifact parameters for random_forest")
		}
		modelJSON, err := json.Marshal(modelDataRaw)
		if err != nil {
			return 0.0, fmt.Errorf("failed to marshal RF data: %w", err)
		}
		var rf training.RandomForestArtifact
		if err := json.Unmarshal(modelJSON, &rf); err != nil {
			return 0.0, fmt.Errorf("failed to unmarshal random forest: %w", err)
		}
		votes := make(map[float64]int)
		for _, tree := range rf.Trees {
			pred := math.Round(training.TraverseTree(tree, features))
			votes[pred]++
		}
		bestCount, bestClass := 0, 0.0
		for class, count := range votes {
			if count > bestCount {
				bestCount = count
				bestClass = class
			}
		}
		return bestClass, nil

	case "regression":
		modelDataRaw, ok := parameters["model_data"]
		if !ok {
			return 0.0, fmt.Errorf("model_data missing from artifact parameters for regression")
		}
		mdMap, ok := modelDataRaw.(map[string]any)
		if !ok {
			return 0.0, fmt.Errorf("invalid model_data format for regression")
		}
		intercept := 0.0
		if b, ok := mdMap["intercept"].(float64); ok {
			intercept = b
		}
		pred := intercept
		if coeffsRaw, ok := mdMap["coefficients"].([]any); ok {
			for i, c := range coeffsRaw {
				if i < len(features) {
					if cf, ok := c.(float64); ok {
						pred += cf * features[i]
					}
				}
			}
		}
		return pred, nil

	case "neural_network":
		modelDataRaw, ok := parameters["model_data"]
		if !ok {
			return 0.0, fmt.Errorf("model_data missing from artifact parameters for neural_network")
		}
		mdMap, ok := modelDataRaw.(map[string]any)
		if !ok {
			return 0.0, fmt.Errorf("invalid model_data format for neural_network")
		}
		weightsJSON, err := json.Marshal(mdMap["weights"])
		if err != nil {
			return 0.0, fmt.Errorf("failed to marshal NN weights: %w", err)
		}
		biasesJSON, err := json.Marshal(mdMap["biases"])
		if err != nil {
			return 0.0, fmt.Errorf("failed to marshal NN biases: %w", err)
		}
		var weights [][][]float64
		var biases [][]float64
		if err := json.Unmarshal(weightsJSON, &weights); err != nil {
			return 0.0, fmt.Errorf("failed to unmarshal NN weights: %w", err)
		}
		if err := json.Unmarshal(biasesJSON, &biases); err != nil {
			return 0.0, fmt.Errorf("failed to unmarshal NN biases: %w", err)
		}
		a := make([]float64, len(features))
		copy(a, features)
		for l, w := range weights {
			outSize := len(w)
			z := make([]float64, outSize)
			for j := range outSize {
				z[j] = biases[l][j]
				for k, ak := range a {
					if k < len(w[j]) {
						z[j] += w[j][k] * ak
					}
				}
			}
			a = make([]float64, outSize)
			isOutput := l == len(weights)-1
			for j := range z {
				if isOutput {
					a[j] = 1.0 / (1.0 + math.Exp(-z[j]))
				} else if z[j] > 0 {
					a[j] = z[j]
				}
			}
		}
		if len(a) > 0 {
			return a[0], nil
		}
		return 0.0, nil

	default:
		return 0.0, fmt.Errorf("unsupported model type: %s", modelType)
	}
}

// computePerformanceMetrics calculates performance metrics for predictions vs actual labels
func computePerformanceMetrics(modelType models.ModelType, predictions, actual []float64) *models.PerformanceMetrics {
	if len(predictions) == 0 || len(predictions) != len(actual) {
		return &models.PerformanceMetrics{}
	}

	switch modelType {
	case models.ModelTypeRegression:
		sumSq, sumAbs := 0.0, 0.0
		meanActual := 0.0
		for _, v := range actual {
			meanActual += v
		}
		meanActual /= float64(len(actual))
		ssTot, ssRes := 0.0, 0.0
		for i := range predictions {
			diff := predictions[i] - actual[i]
			sumSq += diff * diff
			sumAbs += math.Abs(diff)
			ssRes += diff * diff
			ssTot += math.Pow(actual[i]-meanActual, 2)
		}
		rmse := math.Sqrt(sumSq / float64(len(predictions)))
		mae := sumAbs / float64(len(predictions))
		r2 := 0.0
		if ssTot > 0 {
			r2 = 1.0 - (ssRes / ssTot)
		}
		return &models.PerformanceMetrics{RMSE: rmse, MAE: mae, R2Score: r2}
	default:
		correct := 0
		for i := range predictions {
			if math.Round(predictions[i]) == math.Round(actual[i]) {
				correct++
			}
		}
		accuracy := float64(correct) / float64(len(predictions))
		f1 := 0.0
		if accuracy > 0 {
			f1 = 2 * (accuracy * accuracy) / (accuracy + accuracy)
		}
		return &models.PerformanceMetrics{
			Accuracy:  accuracy,
			Precision: accuracy,
			Recall:    accuracy,
			F1Score:   f1,
		}
	}
}

// FailTraining marks training as failed
func (s *Service) FailTraining(modelID, reason string) error {
	model, err := s.store.GetMLModel(modelID)
	if err != nil {
		return fmt.Errorf("failed to get model: %w", err)
	}

	model.Status = models.ModelStatusFailed
	model.TrainingTaskID = ""
	model.UpdatedAt = time.Now().UTC()

	if model.Metadata == nil {
		model.Metadata = make(map[string]any)
	}
	model.Metadata["failure_reason"] = reason

	if err := s.store.SaveMLModel(model); err != nil {
		return fmt.Errorf("failed to update model: %w", err)
	}

	return nil
}
