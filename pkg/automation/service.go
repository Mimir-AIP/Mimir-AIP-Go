package automation

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/mimir-aip/mimir-aip-go/pkg/metadatastore"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

// Service manages persisted automation policies and trigger matching.
type Service struct {
	store metadatastore.MetadataStore
}

func NewService(store metadatastore.MetadataStore) *Service {
	return &Service{store: store}
}

func (s *Service) Create(req *models.AutomationCreateRequest) (*models.Automation, error) {
	if err := validateCreateRequest(req); err != nil {
		return nil, err
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	now := time.Now().UTC()
	automation := &models.Automation{
		ID:              uuid.New().String(),
		ProjectID:       req.ProjectID,
		Name:            req.Name,
		Description:     req.Description,
		Enabled:         enabled,
		TargetType:      req.TargetType,
		TargetID:        req.TargetID,
		TriggerType:     req.TriggerType,
		TriggerConfig:   cloneMap(req.TriggerConfig),
		ConditionConfig: cloneMap(req.ConditionConfig),
		ActionType:      req.ActionType,
		ActionConfig:    cloneMap(req.ActionConfig),
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := s.store.SaveAutomation(automation); err != nil {
		return nil, fmt.Errorf("failed to save automation: %w", err)
	}
	return automation, nil
}

func (s *Service) Get(id string) (*models.Automation, error) {
	return s.store.GetAutomation(id)
}

func (s *Service) ListByProject(projectID string) ([]*models.Automation, error) {
	if projectID == "" {
		return nil, fmt.Errorf("project_id is required")
	}
	return s.store.ListAutomationsByProject(projectID)
}

func (s *Service) Update(id string, req *models.AutomationUpdateRequest) (*models.Automation, error) {
	if id == "" {
		return nil, fmt.Errorf("automation id is required")
	}
	automation, err := s.store.GetAutomation(id)
	if err != nil {
		return nil, err
	}
	if req.Name != nil {
		automation.Name = *req.Name
	}
	if req.Description != nil {
		automation.Description = *req.Description
	}
	if req.Enabled != nil {
		automation.Enabled = *req.Enabled
	}
	if req.TriggerType != nil {
		automation.TriggerType = *req.TriggerType
	}
	if req.TriggerConfig != nil {
		automation.TriggerConfig = cloneMap(req.TriggerConfig)
	}
	if req.ConditionConfig != nil {
		automation.ConditionConfig = cloneMap(req.ConditionConfig)
	}
	if req.ActionType != nil {
		automation.ActionType = *req.ActionType
	}
	if req.ActionConfig != nil {
		automation.ActionConfig = cloneMap(req.ActionConfig)
	}
	automation.UpdatedAt = time.Now().UTC()
	if err := validatePersistedAutomation(automation); err != nil {
		return nil, err
	}
	if err := s.store.SaveAutomation(automation); err != nil {
		return nil, fmt.Errorf("failed to save automation: %w", err)
	}
	return automation, nil
}

func (s *Service) Delete(id string) error {
	if id == "" {
		return fmt.Errorf("automation id is required")
	}
	if err := s.store.DeleteAutomation(id); err != nil {
		return fmt.Errorf("failed to delete automation: %w", err)
	}
	return nil
}

func (s *Service) MatchPipelineCompleted(evt PipelineCompletedEvent) ([]*models.Automation, error) {
	if evt.ProjectID == "" {
		return nil, fmt.Errorf("project_id is required")
	}
	automations, err := s.store.ListAutomationsByProject(evt.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list automations: %w", err)
	}
	matched := make([]*models.Automation, 0)
	for _, automation := range automations {
		if !matchesPipelineCompleted(automation, evt) {
			continue
		}
		matched = append(matched, automation)
	}
	return matched, nil
}

func validateCreateRequest(req *models.AutomationCreateRequest) error {
	if req == nil {
		return fmt.Errorf("automation create request is required")
	}
	automation := &models.Automation{
		ProjectID:       req.ProjectID,
		Name:            req.Name,
		TargetType:      req.TargetType,
		TargetID:        req.TargetID,
		TriggerType:     req.TriggerType,
		TriggerConfig:   req.TriggerConfig,
		ConditionConfig: req.ConditionConfig,
		ActionType:      req.ActionType,
		ActionConfig:    req.ActionConfig,
	}
	return validatePersistedAutomation(automation)
}

func validatePersistedAutomation(automation *models.Automation) error {
	if automation == nil {
		return fmt.Errorf("automation is required")
	}
	if automation.ProjectID == "" {
		return fmt.Errorf("project_id is required")
	}
	if automation.Name == "" {
		return fmt.Errorf("name is required")
	}
	switch automation.TargetType {
	case models.AutomationTargetTypePipeline, models.AutomationTargetTypeDigitalTwin:
	default:
		return fmt.Errorf("target_type must be pipeline or digital_twin")
	}
	if automation.TargetID == "" {
		return fmt.Errorf("target_id is required")
	}
	switch automation.TriggerType {
	case models.AutomationTriggerTypePipelineCompleted, models.AutomationTriggerTypeManual:
	default:
		return fmt.Errorf("trigger_type must be pipeline_completed or manual")
	}
	switch automation.ActionType {
	case models.AutomationActionTypeProcessTwin, models.AutomationActionTypeTriggerExportPipeline:
	default:
		return fmt.Errorf("action_type must be process_twin or trigger_export_pipeline")
	}
	return nil
}

func matchesPipelineCompleted(automation *models.Automation, evt PipelineCompletedEvent) bool {
	if automation == nil || !automation.Enabled {
		return false
	}
	if automation.TriggerType != models.AutomationTriggerTypePipelineCompleted {
		return false
	}
	if ids := stringSlice(automation.TriggerConfig, "pipeline_ids"); len(ids) > 0 && !contains(ids, evt.PipelineID) {
		return false
	}
	if types := stringSlice(automation.TriggerConfig, "pipeline_types"); len(types) > 0 && !contains(types, string(evt.PipelineType)) {
		return false
	}
	return true
}

func stringSlice(values map[string]any, key string) []string {
	if values == nil {
		return nil
	}
	raw, ok := values[key]
	if !ok {
		return nil
	}
	switch typed := raw.(type) {
	case []string:
		return append([]string(nil), typed...)
	case []any:
		result := make([]string, 0, len(typed))
		for _, value := range typed {
			if text, ok := value.(string); ok && text != "" {
				result = append(result, text)
			}
		}
		return result
	default:
		return nil
	}
}

func contains(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}
