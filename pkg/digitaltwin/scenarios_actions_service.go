package digitaltwin

import (
	"fmt"

	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

func (s *Service) CreateScenario(twinID string, req *models.ScenarioCreateRequest) (*models.Scenario, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	twin, err := s.store.GetDigitalTwin(twinID)
	if err != nil {
		return nil, fmt.Errorf("failed to get digital twin: %w", err)
	}

	scenario, err := s.scenarioManager.CreateScenario(twin, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create scenario: %w", err)
	}

	return scenario, nil
}

// GetScenario retrieves a scenario by ID
func (s *Service) GetScenario(id string) (*models.Scenario, error) {
	scenario, err := s.store.GetScenario(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get scenario: %w", err)
	}
	return scenario, nil
}

// ListScenarios lists scenarios for a digital twin
func (s *Service) ListScenarios(twinID string) ([]*models.Scenario, error) {
	scenarios, err := s.store.ListScenariosByDigitalTwin(twinID)
	if err != nil {
		return nil, fmt.Errorf("failed to list scenarios: %w", err)
	}
	return scenarios, nil
}

// DeleteScenario deletes a scenario
func (s *Service) DeleteScenario(id string) error {
	if err := s.store.DeleteScenario(id); err != nil {
		return fmt.Errorf("failed to delete scenario: %w", err)
	}
	return nil
}

// CreateAction creates a new conditional action
func (s *Service) CreateAction(twinID string, req *models.ActionCreateRequest) (*models.Action, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	twin, err := s.store.GetDigitalTwin(twinID)
	if err != nil {
		return nil, fmt.Errorf("failed to get digital twin: %w", err)
	}

	action, err := s.actionManager.CreateAction(twin, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create action: %w", err)
	}

	return action, nil
}

// GetAction retrieves an action by ID
func (s *Service) GetAction(id string) (*models.Action, error) {
	action, err := s.store.GetAction(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get action: %w", err)
	}
	return action, nil
}

// ListActions lists actions for a digital twin
func (s *Service) ListActions(twinID string) ([]*models.Action, error) {
	actions, err := s.store.ListActionsByDigitalTwin(twinID)
	if err != nil {
		return nil, fmt.Errorf("failed to list actions: %w", err)
	}
	return actions, nil
}

// DeleteAction deletes an action
func (s *Service) DeleteAction(id string) error {
	if err := s.store.DeleteAction(id); err != nil {
		return fmt.Errorf("failed to delete action: %w", err)
	}
	return nil
}

// Helper functions

// ontologyClass holds parsed class information from an OWL/Turtle ontology
type ontologyClass struct {
	Name       string
	Label      string
	Properties []string
}
