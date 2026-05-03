package digitaltwin

import (
	"fmt"
	automationpkg "github.com/mimir-aip/mimir-aip-go/pkg/automation"
	"github.com/mimir-aip/mimir-aip-go/pkg/metadatastore"
	"github.com/mimir-aip/mimir-aip-go/pkg/mlmodel"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"github.com/mimir-aip/mimir-aip-go/pkg/ontology"
	"github.com/mimir-aip/mimir-aip-go/pkg/queue"
	"github.com/mimir-aip/mimir-aip-go/pkg/storage"
	"strings"
)

// Service manages digital twin operations.
type Service struct {
	store             metadatastore.MetadataStore
	automationService *automationpkg.Service
	ontologyService   *ontology.Service
	storageService    *storage.Service
	mlService         *mlmodel.Service
	queue             *queue.Queue
	inferenceEngine   *InferenceEngine
	sparqlEngine      *SPARQLEngine
	scenarioManager   *ScenarioManager
	actionManager     *ActionManager
}

type DigitalTwinProjectMismatchError struct {
	DigitalTwinID     string
	ExpectedProjectID string
	ActualProjectID   string
}

func (e *DigitalTwinProjectMismatchError) Error() string {
	return fmt.Sprintf("digital twin %s belongs to project %s, not %s", e.DigitalTwinID, e.ActualProjectID, e.ExpectedProjectID)
}

func (s *Service) getOwnedDigitalTwin(projectID, twinID string) (*models.DigitalTwin, error) {
	if strings.TrimSpace(projectID) == "" {
		return nil, fmt.Errorf("project_id is required")
	}
	twin, err := s.store.GetDigitalTwin(twinID)
	if err != nil {
		return nil, fmt.Errorf("failed to get digital twin: %w", err)
	}
	if twin.ProjectID != projectID {
		return nil, &DigitalTwinProjectMismatchError{
			DigitalTwinID:     twin.ID,
			ExpectedProjectID: projectID,
			ActualProjectID:   twin.ProjectID,
		}
	}
	return twin, nil
}

func (s *Service) getOwnedEntity(twinID, entityID string) (*models.Entity, error) {
	entity, err := s.store.GetEntity(entityID)
	if err != nil {
		return nil, fmt.Errorf("failed to get entity: %w", err)
	}
	if entity.DigitalTwinID != twinID {
		return nil, fmt.Errorf("entity %s does not belong to digital twin %s", entityID, twinID)
	}
	return entity, nil
}

func (s *Service) getOwnedScenario(twinID, scenarioID string) (*models.Scenario, error) {
	scenario, err := s.store.GetScenario(scenarioID)
	if err != nil {
		return nil, fmt.Errorf("failed to get scenario: %w", err)
	}
	if scenario.DigitalTwinID != twinID {
		return nil, fmt.Errorf("scenario %s does not belong to digital twin %s", scenarioID, twinID)
	}
	return scenario, nil
}

func (s *Service) getOwnedAction(twinID, actionID string) (*models.Action, error) {
	action, err := s.store.GetAction(actionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get action: %w", err)
	}
	if action.DigitalTwinID != twinID {
		return nil, fmt.Errorf("action %s does not belong to digital twin %s", actionID, twinID)
	}
	return action, nil
}

func (s *Service) GetDigitalTwinForProject(projectID, twinID string) (*models.DigitalTwin, error) {
	return s.getOwnedDigitalTwin(projectID, twinID)
}

func (s *Service) UpdateDigitalTwinForProject(projectID, twinID string, req *models.DigitalTwinUpdateRequest) (*models.DigitalTwin, error) {
	if _, err := s.getOwnedDigitalTwin(projectID, twinID); err != nil {
		return nil, err
	}
	return s.UpdateDigitalTwin(twinID, req)
}

func (s *Service) DeleteDigitalTwinForProject(projectID, twinID string) error {
	if _, err := s.getOwnedDigitalTwin(projectID, twinID); err != nil {
		return err
	}
	return s.DeleteDigitalTwin(twinID)
}

func (s *Service) EnqueueSyncForProject(projectID, twinID string) (*models.WorkTask, error) {
	twin, err := s.getOwnedDigitalTwin(projectID, twinID)
	if err != nil {
		return nil, err
	}
	return s.EnqueueSync(twin.ID)
}

func (s *Service) SyncWithStorageForProject(projectID, twinID string, opts *models.TwinSyncOptions) error {
	twin, err := s.getOwnedDigitalTwin(projectID, twinID)
	if err != nil {
		return err
	}
	return s.SyncWithStorageWithOptions(twin.ID, opts)
}

func (s *Service) GetStateAtRunForProject(projectID, twinID, runID string) (*models.ReconstructedTwinState, error) {
	twin, err := s.getOwnedDigitalTwin(projectID, twinID)
	if err != nil {
		return nil, err
	}
	return s.GetStateAtRun(twin.ID, runID)
}

func (s *Service) ListSyncRunsForProject(projectID, twinID string, limit int) ([]*models.TwinSyncRun, error) {
	twin, err := s.getOwnedDigitalTwin(projectID, twinID)
	if err != nil {
		return nil, err
	}
	return s.ListSyncRuns(twin.ID, limit)
}

func (s *Service) GetSyncRunForProject(projectID, twinID, runID string) (*models.TwinSyncRun, error) {
	twin, err := s.getOwnedDigitalTwin(projectID, twinID)
	if err != nil {
		return nil, err
	}
	return s.GetSyncRun(twin.ID, runID)
}

func (s *Service) QueryForProject(projectID, twinID string, req *models.QueryRequest) (*models.QueryResult, error) {
	twin, err := s.getOwnedDigitalTwin(projectID, twinID)
	if err != nil {
		return nil, err
	}
	return s.Query(twin.ID, req)
}

func (s *Service) PredictForProject(projectID, twinID string, req *models.PredictionRequest) (*models.Prediction, error) {
	twin, err := s.getOwnedDigitalTwin(projectID, twinID)
	if err != nil {
		return nil, err
	}
	return s.Predict(twin.ID, req)
}

func (s *Service) BatchPredictForProject(projectID, twinID string, req *models.BatchPredictionRequest) ([]*models.Prediction, error) {
	twin, err := s.getOwnedDigitalTwin(projectID, twinID)
	if err != nil {
		return nil, err
	}
	return s.BatchPredict(twin.ID, req)
}

func (s *Service) GetEntityInTwin(twinID, entityID string) (*models.Entity, error) {
	return s.getOwnedEntity(twinID, entityID)
}

func (s *Service) UpdateEntityInTwin(twinID, entityID string, req *models.EntityUpdateRequest) (*models.Entity, error) {
	if _, err := s.getOwnedEntity(twinID, entityID); err != nil {
		return nil, err
	}
	return s.UpdateEntity(entityID, req)
}

func (s *Service) GetEntityHistoryInTwin(twinID, entityID string, limit int) ([]*models.EntityRevision, error) {
	if _, err := s.getOwnedEntity(twinID, entityID); err != nil {
		return nil, err
	}
	return s.GetEntityHistory(entityID, limit)
}

func (s *Service) GetScenarioInTwin(twinID, scenarioID string) (*models.Scenario, error) {
	return s.getOwnedScenario(twinID, scenarioID)
}

func (s *Service) DeleteScenarioInTwin(twinID, scenarioID string) error {
	if _, err := s.getOwnedScenario(twinID, scenarioID); err != nil {
		return err
	}
	return s.DeleteScenario(scenarioID)
}

func (s *Service) GetActionInTwin(twinID, actionID string) (*models.Action, error) {
	return s.getOwnedAction(twinID, actionID)
}

func (s *Service) DeleteActionInTwin(twinID, actionID string) error {
	if _, err := s.getOwnedAction(twinID, actionID); err != nil {
		return err
	}
	return s.DeleteAction(actionID)
}

func (s *Service) ListActionsForProject(projectID, twinID string) ([]*models.Action, error) {
	twin, err := s.getOwnedDigitalTwin(projectID, twinID)
	if err != nil {
		return nil, err
	}
	return s.ListActions(twin.ID)
}

func (s *Service) ListScenariosForProject(projectID, twinID string) ([]*models.Scenario, error) {
	twin, err := s.getOwnedDigitalTwin(projectID, twinID)
	if err != nil {
		return nil, err
	}
	return s.ListScenarios(twin.ID)
}

func (s *Service) ListEntitiesForProject(projectID, twinID string) ([]*models.Entity, error) {
	twin, err := s.getOwnedDigitalTwin(projectID, twinID)
	if err != nil {
		return nil, err
	}
	return s.ListEntities(twin.ID)
}

func (s *Service) CreateScenarioForProject(projectID, twinID string, req *models.ScenarioCreateRequest) (*models.Scenario, error) {
	twin, err := s.getOwnedDigitalTwin(projectID, twinID)
	if err != nil {
		return nil, err
	}
	return s.CreateScenario(twin.ID, req)
}

func (s *Service) CreateActionForProject(projectID, twinID string, req *models.ActionCreateRequest) (*models.Action, error) {
	twin, err := s.getOwnedDigitalTwin(projectID, twinID)
	if err != nil {
		return nil, err
	}
	return s.CreateAction(twin.ID, req)
}

func (s *Service) GetRelatedEntitiesForProject(projectID, twinID, entityID, relationshipType string) ([]*models.Entity, error) {
	twin, err := s.getOwnedDigitalTwin(projectID, twinID)
	if err != nil {
		return nil, err
	}
	return s.GetRelatedEntities(twin.ID, entityID, relationshipType)
}

// NewService creates a new digital twin service.
func NewService(
	store metadatastore.MetadataStore,
	automationService *automationpkg.Service,
	ontologyService *ontology.Service,
	storageService *storage.Service,
	mlService *mlmodel.Service,
	q *queue.Queue,
) *Service {
	s := &Service{
		store:             store,
		automationService: automationService,
		ontologyService:   ontologyService,
		storageService:    storageService,
		mlService:         mlService,
		queue:             q,
	}

	// Initialize sub-components.
	s.inferenceEngine = NewInferenceEngine(mlService, store)
	s.sparqlEngine = NewSPARQLEngine(store, ontologyService)
	s.scenarioManager = NewScenarioManager(store, s.inferenceEngine)
	s.actionManager = NewActionManager(store, q)

	return s
}
