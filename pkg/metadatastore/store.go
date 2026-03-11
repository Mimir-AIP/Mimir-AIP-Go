package metadatastore

import "github.com/mimir-aip/mimir-aip-go/pkg/models"

// MetadataStore is the interface for orchestrator metadata persistence
// This stores project definitions, pipeline configurations, and schedules.
// This is NOT the CIR storage system for ingested/ontology data.
type MetadataStore interface {
	// Project operations
	SaveProject(project *models.Project) error
	GetProject(id string) (*models.Project, error)
	ListProjects() ([]*models.Project, error)
	DeleteProject(id string) error

	// Pipeline operations
	SavePipeline(pipeline *models.Pipeline) error
	GetPipeline(id string) (*models.Pipeline, error)
	ListPipelines() ([]*models.Pipeline, error)
	ListPipelinesByProject(projectID string) ([]*models.Pipeline, error)
	DeletePipeline(id string) error
	GetPipelineCheckpoint(projectID, pipelineID, stepName, scope string) (*models.PipelineCheckpoint, error)
	SavePipelineCheckpoint(checkpoint *models.PipelineCheckpoint) error

	// Schedule operations
	SaveSchedule(schedule *models.Schedule) error
	GetSchedule(id string) (*models.Schedule, error)
	ListSchedules() ([]*models.Schedule, error)
	ListSchedulesByProject(projectID string) ([]*models.Schedule, error)
	DeleteSchedule(id string) error

	// Plugin operations
	SavePlugin(plugin *models.Plugin, binaryData []byte) error
	GetPlugin(name string) (*models.Plugin, error)
	GetPluginBinary(name string) ([]byte, error)
	ListPlugins() ([]*models.Plugin, error)
	DeletePlugin(name string) error
	UpdatePluginStatus(name string, status models.PluginStatus) error

	// External storage plugin operations
	SaveExternalStoragePlugin(plugin *models.ExternalStoragePlugin) error
	GetExternalStoragePlugin(name string) (*models.ExternalStoragePlugin, error)
	ListExternalStoragePlugins() ([]*models.ExternalStoragePlugin, error)
	DeleteExternalStoragePlugin(name string) error

	// External LLM provider operations
	SaveExternalLLMProvider(p *models.ExternalLLMProvider) error
	GetExternalLLMProvider(name string) (*models.ExternalLLMProvider, error)
	ListExternalLLMProviders() ([]*models.ExternalLLMProvider, error)
	DeleteExternalLLMProvider(name string) error

	// Analysis operations
	SaveAnalysisRun(run *models.AnalysisRun) error
	SaveResolverRun(run *models.AnalysisRun, items []*models.ReviewItem) error
	GetAnalysisRun(id string) (*models.AnalysisRun, error)
	ListAnalysisRunsByProject(projectID string) ([]*models.AnalysisRun, error)
	SaveReviewItem(item *models.ReviewItem) error
	GetReviewItem(id string) (*models.ReviewItem, error)
	GetReviewItemByFindingKey(projectID, findingKey string) (*models.ReviewItem, error)
	ListReviewItems(projectID string) ([]*models.ReviewItem, error)
	SaveInsight(insight *models.Insight) error
	SaveInsightRun(run *models.AnalysisRun, insights []*models.Insight) error
	GetInsight(id string) (*models.Insight, error)
	ListInsightsByProject(projectID string) ([]*models.Insight, error)
	// Storage operations
	SaveStorageConfig(config *models.StorageConfig) error
	GetStorageConfig(id string) (*models.StorageConfig, error)
	ListStorageConfigs() ([]*models.StorageConfig, error)
	ListStorageConfigsByProject(projectID string) ([]*models.StorageConfig, error)
	DeleteStorageConfig(id string) error

	// Ontology operations
	SaveOntology(ontology *models.Ontology) error
	GetOntology(id string) (*models.Ontology, error)
	ListOntologies() ([]*models.Ontology, error)
	ListOntologiesByProject(projectID string) ([]*models.Ontology, error)
	DeleteOntology(id string) error

	// ML Model operations
	SaveMLModel(model *models.MLModel) error
	GetMLModel(id string) (*models.MLModel, error)
	ListMLModels() ([]*models.MLModel, error)
	ListMLModelsByProject(projectID string) ([]*models.MLModel, error)
	DeleteMLModel(id string) error

	// Digital Twin operations
	SaveDigitalTwin(twin *models.DigitalTwin) error
	GetDigitalTwin(id string) (*models.DigitalTwin, error)
	ListDigitalTwins() ([]*models.DigitalTwin, error)
	ListDigitalTwinsByProject(projectID string) ([]*models.DigitalTwin, error)
	DeleteDigitalTwin(id string) error

	// Digital Twin Entity operations
	SaveEntity(entity *models.Entity) error
	GetEntity(id string) (*models.Entity, error)
	ListEntitiesByDigitalTwin(twinID string) ([]*models.Entity, error)
	ListEntitiesByTypeInTwin(twinID, entityType string) ([]*models.Entity, error)
	DeleteEntity(id string) error

	// Digital Twin Scenario operations
	SaveScenario(scenario *models.Scenario) error
	GetScenario(id string) (*models.Scenario, error)
	ListScenariosByDigitalTwin(twinID string) ([]*models.Scenario, error)
	DeleteScenario(id string) error

	// Digital Twin Action operations
	SaveAction(action *models.Action) error
	GetAction(id string) (*models.Action, error)
	ListActionsByDigitalTwin(twinID string) ([]*models.Action, error)
	DeleteAction(id string) error

	// Digital Twin Prediction operations
	SavePrediction(prediction *models.Prediction) error
	GetPrediction(id string) (*models.Prediction, error)
	ListPredictionsByEntity(entityID string) ([]*models.Prediction, error)
	ListPredictionsByDigitalTwin(twinID string) ([]*models.Prediction, error)
	DeletePrediction(id string) error
	DeleteExpiredPredictions(twinID string) error
}
