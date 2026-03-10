package pipeline

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/mimir-aip/mimir-aip-go/pkg/metadatastore"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"github.com/mimir-aip/mimir-aip-go/pkg/pluginruntime"
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
	store           metadatastore.MetadataStore
	plugins         *pluginruntime.Registry[Plugin]
	pluginRegistry  PluginRegistry
	storageSvc      CIRStorer
	checkpointStore PipelineCheckpointStore
}

// NewService creates a new pipeline service
func NewService(store metadatastore.MetadataStore) *Service {
	s := &Service{
		store:           store,
		plugins:         pluginruntime.NewRegistry[Plugin](),
		checkpointStore: store,
	}
	s.refreshBuiltinPlugins()
	return s
}

// NewServiceWithRegistry creates a new pipeline service with an external plugin registry
func NewServiceWithRegistry(store metadatastore.MetadataStore, registry PluginRegistry) *Service {
	s := &Service{
		store:           store,
		plugins:         pluginruntime.NewRegistry[Plugin](),
		pluginRegistry:  registry,
		checkpointStore: store,
	}
	s.refreshBuiltinPlugins()
	return s
}

func (s *Service) refreshBuiltinPlugins() {
	dp := NewDefaultPluginWithDeps(s.storageSvc, s.checkpointStore)
	s.plugins.Register("default", dp)
	s.plugins.Register("builtin", dp)
}

// SetStorageSvc injects a CIRStorer into the built-in default/builtin plugins so that
// store_cir and store_cir_batch actions can persist data to Mimir storage.
// Call this after both the pipeline service and the storage service are created.
func (s *Service) SetStorageSvc(svc CIRStorer) {
	s.storageSvc = svc
	s.refreshBuiltinPlugins()
}

// RegisterPlugin registers a plugin
func (s *Service) RegisterPlugin(name string, plugin Plugin) {
	s.plugins.Register(name, plugin)
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

// GetCheckpoint retrieves persisted connector state for a pipeline step.
func (s *Service) GetCheckpoint(projectID, pipelineID, stepName, scope string) (*models.PipelineCheckpoint, error) {
	return s.store.GetPipelineCheckpoint(projectID, pipelineID, stepName, scope)
}

// SaveCheckpoint persists connector state for a pipeline step.
func (s *Service) SaveCheckpoint(checkpoint *models.PipelineCheckpoint) error {
	return s.store.SavePipelineCheckpoint(checkpoint)
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

	execution.Context.SetStepData("_runtime", "project_id", pipeline.ProjectID)
	execution.Context.SetStepData("_runtime", "pipeline_id", pipeline.ID)
	execution.Context.SetStepData("_runtime", "trigger_type", req.TriggerType)

	// Execute pipeline steps
	log.Printf("Executing pipeline %s (%s) - %d steps", pipeline.Name, pipeline.ID, len(pipeline.Steps))

	currentStepIndex := 0
	for currentStepIndex < len(pipeline.Steps) {
		step := pipeline.Steps[currentStepIndex]

		// for_each: iterate over a collection and execute sub-steps for each item
		if step.ForEach != nil {
			log.Printf("  Step %d: %s (for_each)", currentStepIndex+1, step.Name)
			count, err := s.executeForEach(step, execution, pipeline.Steps)
			if err != nil {
				execution.Status = "failed"
				execution.Error = fmt.Sprintf("step %s failed: %v", step.Name, err)
				now := time.Now()
				execution.CompletedAt = &now
				return execution, fmt.Errorf("step %s failed: %w", step.Name, err)
			}
			execution.Context.SetStepData(step.Name, "count", count)
			currentStepIndex++
			continue
		}

		log.Printf("  Step %d: %s (%s.%s)", currentStepIndex+1, step.Name, step.Plugin, step.Action)

		result, gotoTarget, err := s.executeStep(step, execution.Context)
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

		// Check for goto action
		if gotoTarget != "" {
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

// executeStep runs a single pipeline step, returning its result map and any goto target.
func (s *Service) executeStep(step models.PipelineStep, ctx *models.PipelineContext) (map[string]interface{}, string, error) {
	ctx.SetStepData("_runtime", "current_step", step.Name)

	plugin, ok := s.plugins.Get(step.Plugin)
	if !ok && s.pluginRegistry != nil {
		plugin, ok = s.pluginRegistry.GetPlugin(step.Plugin)
	}
	if !ok {
		return nil, "", fmt.Errorf("unknown plugin: %s", step.Plugin)
	}

	result, err := plugin.Execute(step.Action, step.Parameters, ctx)
	if err != nil {
		return nil, "", err
	}

	// Resolve and store declared output mappings
	if step.Output != nil {
		dpPlugin, _ := s.plugins.Get("default")
		dp, isDef := dpPlugin.(*DefaultPlugin)
		for outputKey, outputTemplate := range step.Output {
			if isDef {
				resolvedValue := dp.ResolveTemplates(outputTemplate, ctx)
				ctx.SetStepData(step.Name, outputKey, resolvedValue)
				log.Printf("    Output: %s = %v", outputKey, resolvedValue)
			}
		}
	}

	gotoTarget := ""
	if gt, ok := result["goto"].(string); ok {
		gotoTarget = gt
	}

	return result, gotoTarget, nil
}

// executeForEach iterates over a resolved array and runs the for_each sub-steps
// for each element. Returns the number of items processed.
func (s *Service) executeForEach(step models.PipelineStep, execution *models.PipelineExecution, allSteps []models.PipelineStep) (int, error) {
	fe := step.ForEach

	// Resolve the items array. Items is a template string referencing context.
	var items []interface{}
	dpPlugin, ok := s.plugins.Get("default")
	dp, ok := dpPlugin.(*DefaultPlugin)
	if !ok {
		return 0, fmt.Errorf("for_each requires the default plugin to be registered")
	}

	resolved := dp.ResolveTemplates(fe.Items, execution.Context)
	if err := json.Unmarshal([]byte(resolved), &items); err != nil {
		return 0, fmt.Errorf("for_each items %q must resolve to a JSON array: %w", fe.Items, err)
	}

	as := fe.As
	if as == "" {
		as = "item"
	}

	for i, item := range items {
		// Bind current item into _loop context
		execution.Context.SetStepData("_loop", as, item)
		execution.Context.SetStepData("_loop", "index", i)

		// Execute sub-steps
		for _, subStep := range fe.Steps {
			if subStep.ForEach != nil {
				// Nested for_each
				if _, err := s.executeForEach(subStep, execution, nil); err != nil {
					return i, fmt.Errorf("sub-step %s (iteration %d): %w", subStep.Name, i, err)
				}
				continue
			}

			result, gotoTarget, err := s.executeStep(subStep, execution.Context)
			if err != nil {
				return i, fmt.Errorf("sub-step %s (iteration %d): %w", subStep.Name, i, err)
			}
			for key, value := range result {
				execution.Context.SetStepData(subStep.Name, key, value)
			}
			if gotoTarget != "" {
				log.Printf("    for_each: goto inside sub-steps is not supported, ignoring target %s", gotoTarget)
			}
		}
	}

	return len(items), nil
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

		if step.ForEach == nil {
			if step.Plugin == "" {
				return fmt.Errorf("step plugin is required for step: %s", step.Name)
			}
			if step.Action == "" {
				return fmt.Errorf("step action is required for step: %s", step.Name)
			}
		} else {
			if step.ForEach.Items == "" {
				return fmt.Errorf("for_each.items is required for step: %s", step.Name)
			}
			if len(step.ForEach.Steps) == 0 {
				return fmt.Errorf("for_each.steps must not be empty for step: %s", step.Name)
			}
		}
	}

	return nil
}
