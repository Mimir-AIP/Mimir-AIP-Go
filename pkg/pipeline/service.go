package pipeline

import (
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/mimir-aip/mimir-aip-go/pkg/metadatastore"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

const (
	DefaultContextMaxSize = 10485760 // 10MB
)

// PluginRegistry interface for accessing plugins
type PluginRegistry interface {
	GetPlugin(name string) (Plugin, bool)
}

// Service provides pipeline management and execution operations
type Service struct {
	store          metadatastore.MetadataStore
	plugins        map[string]Plugin
	pluginRegistry PluginRegistry
}

// NewService creates a new pipeline service
func NewService(store metadatastore.MetadataStore) *Service {
	s := &Service{
		store:   store,
		plugins: make(map[string]Plugin),
	}

	// Register default plugin
	s.RegisterPlugin("default", NewDefaultPlugin())
	s.RegisterPlugin("builtin", NewDefaultPlugin()) // Alias for default

	return s
}

// NewServiceWithRegistry creates a new pipeline service with an external plugin registry
func NewServiceWithRegistry(store metadatastore.MetadataStore, registry PluginRegistry) *Service {
	s := &Service{
		store:          store,
		plugins:        make(map[string]Plugin),
		pluginRegistry: registry,
	}

	// Register default plugin
	s.RegisterPlugin("default", NewDefaultPlugin())
	s.RegisterPlugin("builtin", NewDefaultPlugin()) // Alias for default

	return s
}

// RegisterPlugin registers a plugin
func (s *Service) RegisterPlugin(name string, plugin Plugin) {
	s.plugins[name] = plugin
}

// Create creates a new pipeline
func (s *Service) Create(req *models.PipelineCreateRequest) (*models.Pipeline, error) {
	// Validate request
	if err := s.validateCreateRequest(req); err != nil {
		return nil, err
	}

	// Create pipeline
	pipeline := &models.Pipeline{
		ID:          uuid.New().String(),
		ProjectID:   req.ProjectID,
		Name:        req.Name,
		Type:        req.Type,
		Description: req.Description,
		Steps:       req.Steps,
		Status:      models.PipelineStatusActive,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Save pipeline
	if err := s.store.SavePipeline(pipeline); err != nil {
		return nil, fmt.Errorf("failed to save pipeline: %w", err)
	}

	return pipeline, nil
}

// Get retrieves a pipeline by ID
func (s *Service) Get(id string) (*models.Pipeline, error) {
	return s.store.GetPipeline(id)
}

// List lists all pipelines
func (s *Service) List() ([]*models.Pipeline, error) {
	return s.store.ListPipelines()
}

// ListByProject lists all pipelines for a specific project
func (s *Service) ListByProject(projectID string) ([]*models.Pipeline, error) {
	return s.store.ListPipelinesByProject(projectID)
}

// Update updates a pipeline
func (s *Service) Update(id string, req *models.PipelineUpdateRequest) (*models.Pipeline, error) {
	// Get existing pipeline
	pipeline, err := s.store.GetPipeline(id)
	if err != nil {
		return nil, err
	}

	// Update fields
	if req.Description != nil {
		pipeline.Description = *req.Description
	}
	if req.Steps != nil {
		pipeline.Steps = *req.Steps
	}
	if req.Status != nil {
		pipeline.Status = *req.Status
	}

	// Update timestamp
	pipeline.UpdatedAt = time.Now()

	// Save pipeline
	if err := s.store.SavePipeline(pipeline); err != nil {
		return nil, fmt.Errorf("failed to save pipeline: %w", err)
	}

	return pipeline, nil
}

// Delete deletes a pipeline
func (s *Service) Delete(id string) error {
	return s.store.DeletePipeline(id)
}

// Execute executes a pipeline
func (s *Service) Execute(pipelineID string, req *models.PipelineExecutionRequest) (*models.PipelineExecution, error) {
	// Get pipeline
	pipeline, err := s.store.GetPipeline(pipelineID)
	if err != nil {
		return nil, err
	}

	// Create execution record
	execution := &models.PipelineExecution{
		ID:          uuid.New().String(),
		PipelineID:  pipelineID,
		ProjectID:   pipeline.ProjectID,
		Status:      "running",
		StartedAt:   time.Now(),
		Context:     models.NewPipelineContext(DefaultContextMaxSize),
		TriggerType: req.TriggerType,
		TriggeredBy: req.TriggeredBy,
	}

	// Add initial parameters to context if provided
	if req.Parameters != nil {
		for key, value := range req.Parameters {
			execution.Context.SetStepData("_parameters", key, value)
		}
	}

	// Execute pipeline steps
	log.Printf("Executing pipeline %s (%s) - %d steps", pipeline.Name, pipeline.ID, len(pipeline.Steps))

	currentStepIndex := 0
	for currentStepIndex < len(pipeline.Steps) {
		step := pipeline.Steps[currentStepIndex]

		log.Printf("  Step %d: %s (%s.%s)", currentStepIndex+1, step.Name, step.Plugin, step.Action)

		// Get plugin - check local registry first, then external registry
		plugin, ok := s.plugins[step.Plugin]
		if !ok && s.pluginRegistry != nil {
			plugin, ok = s.pluginRegistry.GetPlugin(step.Plugin)
		}
		if !ok {
			execution.Status = "failed"
			execution.Error = fmt.Sprintf("unknown plugin: %s", step.Plugin)
			now := time.Now()
			execution.CompletedAt = &now
			return execution, fmt.Errorf("unknown plugin: %s", step.Plugin)
		}

		// Execute step
		result, err := plugin.Execute(step.Action, step.Parameters, execution.Context)
		if err != nil {
			execution.Status = "failed"
			execution.Error = fmt.Sprintf("step %s failed: %v", step.Name, err)
			now := time.Now()
			execution.CompletedAt = &now
			log.Printf("    Error: %v", err)
			return execution, fmt.Errorf("step %s failed: %w", step.Name, err)
		}

		// Store step results in context
		for key, value := range result {
			execution.Context.SetStepData(step.Name, key, value)
		}

		// Store output values from step configuration
		if step.Output != nil {
			for outputKey, outputTemplate := range step.Output {
				// Resolve template
				plugin := s.plugins["default"]
				if dp, ok := plugin.(*DefaultPlugin); ok {
					resolvedValue := dp.ResolveTemplates(outputTemplate, execution.Context)
					execution.Context.SetStepData(step.Name, outputKey, resolvedValue)
					log.Printf("    Output: %s = %v", outputKey, resolvedValue)
				}
			}
		}

		// Check for goto action
		if gotoTarget, ok := result["goto"].(string); ok {
			// Find target step index
			targetIndex := -1
			for i, s := range pipeline.Steps {
				if s.Name == gotoTarget {
					targetIndex = i
					break
				}
			}

			if targetIndex == -1 {
				execution.Status = "failed"
				execution.Error = fmt.Sprintf("goto target not found: %s", gotoTarget)
				now := time.Now()
				execution.CompletedAt = &now
				return execution, fmt.Errorf("goto target not found: %s", gotoTarget)
			}

			log.Printf("    Jumping to step: %s", gotoTarget)
			currentStepIndex = targetIndex
			continue
		}

		currentStepIndex++
	}

	// Mark execution as completed
	execution.Status = "completed"
	now := time.Now()
	execution.CompletedAt = &now

	log.Printf("Pipeline execution completed: %s", execution.ID)

	return execution, nil
}

// validateCreateRequest validates a pipeline creation request
func (s *Service) validateCreateRequest(req *models.PipelineCreateRequest) error {
	// Validate name
	if req.Name == "" {
		return fmt.Errorf("pipeline name is required")
	}

	// Validate type
	if req.Type != models.PipelineTypeIngestion &&
		req.Type != models.PipelineTypeProcessing &&
		req.Type != models.PipelineTypeOutput {
		return fmt.Errorf("invalid pipeline type: must be one of ingestion, processing, output")
	}

	// Validate steps
	if len(req.Steps) == 0 {
		return fmt.Errorf("pipeline must have at least one step")
	}

	// Validate step names are unique
	stepNames := make(map[string]bool)
	for _, step := range req.Steps {
		if step.Name == "" {
			return fmt.Errorf("step name is required")
		}
		if stepNames[step.Name] {
			return fmt.Errorf("duplicate step name: %s", step.Name)
		}
		stepNames[step.Name] = true

		if step.Plugin == "" {
			return fmt.Errorf("step plugin is required for step: %s", step.Name)
		}
		if step.Action == "" {
			return fmt.Errorf("step action is required for step: %s", step.Name)
		}
	}

	return nil
}
