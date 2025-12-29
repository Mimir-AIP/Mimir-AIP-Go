package utils

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// PipelineMetadata holds metadata for a pipeline
type PipelineMetadata struct {
	ID          string    `json:"id" yaml:"id"`
	Name        string    `json:"name" yaml:"name"`
	Description string    `json:"description" yaml:"description"`
	Tags        []string  `json:"tags" yaml:"tags"`
	Enabled     bool      `json:"enabled" yaml:"enabled"`
	Schedule    string    `json:"schedule,omitempty" yaml:"schedule,omitempty"`
	CreatedAt   time.Time `json:"created_at" yaml:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" yaml:"updated_at"`
	CreatedBy   string    `json:"created_by" yaml:"created_by"`
	UpdatedBy   string    `json:"updated_by" yaml:"updated_by"`
	Version     int       `json:"version" yaml:"version"`
}

// PipelineDefinition holds the complete pipeline definition
type PipelineDefinition struct {
	Metadata PipelineMetadata `json:"metadata" yaml:"metadata"`
	Config   PipelineConfig   `json:"config" yaml:"config"`
}

// PipelineStore manages pipeline storage and retrieval
type PipelineStore struct {
	storePath string
	pipelines map[string]*PipelineDefinition
	mutex     sync.RWMutex
}

// NewPipelineStore creates a new pipeline store
func NewPipelineStore(storePath string) *PipelineStore {
	return &PipelineStore{
		storePath: storePath,
		pipelines: make(map[string]*PipelineDefinition),
	}
}

// Initialize loads all pipelines from the store path
func (ps *PipelineStore) Initialize() error {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()

	// Create store directory if it doesn't exist
	if err := os.MkdirAll(ps.storePath, 0755); err != nil {
		return fmt.Errorf("failed to create pipeline store directory: %w", err)
	}

	// Load all pipeline files
	err := filepath.WalkDir(ps.storePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || (!strings.HasSuffix(path, ".yaml") && !strings.HasSuffix(path, ".yml") && !strings.HasSuffix(path, ".json")) {
			return nil
		}

		pipeline, err := ps.loadPipelineFromFile(path)
		if err != nil {
			GetLogger().Warn("Failed to load pipeline from file", Error(err), String("file", path))
			return nil // Continue loading other pipelines
		}

		ps.pipelines[pipeline.Metadata.ID] = pipeline
		GetLogger().Info("Loaded pipeline from file", String("id", pipeline.Metadata.ID), String("name", pipeline.Metadata.Name), String("file", path))
		return nil
	})

	GetLogger().Info("Pipeline store initialized", Int("count", len(ps.pipelines)), String("storePath", ps.storePath))
	return err
}

// CreatePipeline creates a new pipeline
func (ps *PipelineStore) CreatePipeline(metadata PipelineMetadata, config PipelineConfig, createdBy string) (*PipelineDefinition, error) {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()

	// Generate ID if not provided
	if metadata.ID == "" {
		metadata.ID = generatePipelineID()
	}

	// Check if pipeline already exists
	if _, exists := ps.pipelines[metadata.ID]; exists {
		return nil, fmt.Errorf("pipeline with ID %s already exists", metadata.ID)
	}

	// Set creation metadata
	now := time.Now()
	metadata.CreatedAt = now
	metadata.UpdatedAt = now
	metadata.CreatedBy = createdBy
	metadata.UpdatedBy = createdBy
	metadata.Version = 1

	// Validate pipeline configuration
	if err := ps.validatePipelineConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid pipeline configuration: %w", err)
	}

	pipeline := &PipelineDefinition{
		Metadata: metadata,
		Config:   config,
	}

	// Save to file
	if err := ps.savePipelineToFile(pipeline); err != nil {
		return nil, fmt.Errorf("failed to save pipeline: %w", err)
	}

	ps.pipelines[metadata.ID] = pipeline

	GetLogger().Info("Pipeline created", String("id", metadata.ID), String("name", metadata.Name), String("created_by", createdBy))
	return pipeline, nil
}

// GetPipeline retrieves a pipeline by ID
func (ps *PipelineStore) GetPipeline(id string) (*PipelineDefinition, error) {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()

	pipeline, exists := ps.pipelines[id]
	if !exists {
		return nil, fmt.Errorf("pipeline with ID %s not found", id)
	}

	return pipeline, nil
}

// ListPipelines returns all pipelines with optional filtering
func (ps *PipelineStore) ListPipelines(filter map[string]any) ([]*PipelineDefinition, error) {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()

	var pipelines []*PipelineDefinition
	for _, pipeline := range ps.pipelines {
		if ps.matchesFilter(pipeline, filter) {
			pipelines = append(pipelines, pipeline)
		}
	}

	// Sort by creation date (newest first)
	sort.Slice(pipelines, func(i, j int) bool {
		return pipelines[i].Metadata.CreatedAt.After(pipelines[j].Metadata.CreatedAt)
	})

	return pipelines, nil
}

// UpdatePipeline updates an existing pipeline
func (ps *PipelineStore) UpdatePipeline(id string, updates *PipelineMetadata, config *PipelineConfig, updatedBy string) (*PipelineDefinition, error) {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()

	pipeline, exists := ps.pipelines[id]
	if !exists {
		return nil, fmt.Errorf("pipeline with ID %s not found", id)
	}

	// Apply metadata updates
	if updates != nil {
		if updates.Name != "" {
			pipeline.Metadata.Name = updates.Name
		}
		if updates.Description != "" {
			pipeline.Metadata.Description = updates.Description
		}
		if updates.Tags != nil {
			pipeline.Metadata.Tags = updates.Tags
		}
		if updates.Schedule != "" {
			pipeline.Metadata.Schedule = updates.Schedule
		}
		pipeline.Metadata.Enabled = updates.Enabled
		pipeline.Metadata.UpdatedBy = updatedBy
		pipeline.Metadata.UpdatedAt = time.Now()
		pipeline.Metadata.Version++
	}

	// Apply config updates
	if config != nil {
		if err := ps.validatePipelineConfig(config); err != nil {
			return nil, fmt.Errorf("invalid pipeline configuration: %w", err)
		}
		pipeline.Config = *config
		pipeline.Metadata.UpdatedAt = time.Now()
		pipeline.Metadata.UpdatedBy = updatedBy
		pipeline.Metadata.Version++
	}

	// Save to file
	if err := ps.savePipelineToFile(pipeline); err != nil {
		return nil, fmt.Errorf("failed to save updated pipeline: %w", err)
	}

	GetLogger().Info("Pipeline updated", String("id", id), String("version", fmt.Sprintf("%d", pipeline.Metadata.Version)), String("updated_by", updatedBy))
	return pipeline, nil
}

// DeletePipeline deletes a pipeline
func (ps *PipelineStore) DeletePipeline(id string) error {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()

	pipeline, exists := ps.pipelines[id]
	if !exists {
		return fmt.Errorf("pipeline with ID %s not found", id)
	}

	// Delete file
	fileName := ps.getPipelineFileName(id, pipeline.Metadata.Name)
	filePath := filepath.Join(ps.storePath, fileName)
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete pipeline file: %w", err)
	}

	delete(ps.pipelines, id)

	GetLogger().Info("Pipeline deleted", String("id", id), String("name", pipeline.Metadata.Name))
	return nil
}

// ValidatePipeline validates a pipeline configuration
func (ps *PipelineStore) ValidatePipeline(pipelineDef *PipelineDefinition) error {
	// Validate metadata
	if pipelineDef.Metadata.Name == "" {
		return fmt.Errorf("pipeline name is required")
	}

	// Validate configuration
	return ps.validatePipelineConfig(&pipelineDef.Config)
}

// ClonePipeline creates a copy of an existing pipeline
func (ps *PipelineStore) ClonePipeline(id, newName, clonedBy string) (*PipelineDefinition, error) {
	original, err := ps.GetPipeline(id)
	if err != nil {
		return nil, err
	}

	newMetadata := original.Metadata
	newMetadata.ID = generatePipelineID()
	newMetadata.Name = newName
	newMetadata.CreatedBy = clonedBy
	newMetadata.UpdatedBy = clonedBy
	newMetadata.CreatedAt = time.Now()
	newMetadata.UpdatedAt = time.Now()
	newMetadata.Version = 1

	return ps.CreatePipeline(newMetadata, original.Config, clonedBy)
}

// GetPipelineHistory returns version history (simplified - just current version)
func (ps *PipelineStore) GetPipelineHistory(id string) ([]*PipelineDefinition, error) {
	pipeline, err := ps.GetPipeline(id)
	if err != nil {
		return nil, err
	}

	return []*PipelineDefinition{pipeline}, nil
}

// Helper methods

func (ps *PipelineStore) matchesFilter(pipeline *PipelineDefinition, filter map[string]any) bool {
	if filter == nil {
		return true
	}

	// Filter by enabled status
	if enabled, ok := filter["enabled"].(bool); ok {
		if pipeline.Metadata.Enabled != enabled {
			return false
		}
	}

	// Filter by tags
	if tags, ok := filter["tags"].([]string); ok {
		for _, requiredTag := range tags {
			found := false
			for _, pipelineTag := range pipeline.Metadata.Tags {
				if pipelineTag == requiredTag {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
	}

	// Filter by name (partial match)
	if nameFilter, ok := filter["name"].(string); ok {
		if !strings.Contains(strings.ToLower(pipeline.Metadata.Name), strings.ToLower(nameFilter)) {
			return false
		}
	}

	return true
}

func (ps *PipelineStore) validatePipelineConfig(config *PipelineConfig) error {
	if config.Name == "" {
		return fmt.Errorf("pipeline name is required")
	}

	if len(config.Steps) == 0 {
		return fmt.Errorf("pipeline must have at least one step")
	}

	// Validate each step
	for i, step := range config.Steps {
		if step.Name == "" {
			return fmt.Errorf("step %d: name is required", i)
		}
		if step.Plugin == "" {
			return fmt.Errorf("step %d: plugin is required", i)
		}
	}

	return nil
}

func (ps *PipelineStore) loadPipelineFromFile(filePath string) (*PipelineDefinition, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read pipeline file: %w", err)
	}

	var pipeline PipelineDefinition
	ext := filepath.Ext(filePath)

	switch strings.ToLower(ext) {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &pipeline); err != nil {
			return nil, fmt.Errorf("failed to parse YAML: %w", err)
		}
	case ".json":
		if err := json.Unmarshal(data, &pipeline); err != nil {
			return nil, fmt.Errorf("failed to parse JSON: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported file format: %s", ext)
	}

	// Validate loaded pipeline
	if err := ps.validatePipelineConfig(&pipeline.Config); err != nil {
		return nil, fmt.Errorf("invalid pipeline configuration: %w", err)
	}

	return &pipeline, nil
}

func (ps *PipelineStore) savePipelineToFile(pipeline *PipelineDefinition) error {
	fileName := ps.getPipelineFileName(pipeline.Metadata.ID, pipeline.Metadata.Name)
	filePath := filepath.Join(ps.storePath, fileName)

	var data []byte
	var err error

	// Use YAML format by default
	if strings.HasSuffix(fileName, ".json") {
		data, err = json.MarshalIndent(pipeline, "", "  ")
	} else {
		data, err = yaml.Marshal(pipeline)
	}

	if err != nil {
		return fmt.Errorf("failed to marshal pipeline: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write pipeline file: %w", err)
	}

	return nil
}

func (ps *PipelineStore) getPipelineFileName(id, name string) string {
	// Sanitize name for filename
	sanitizedName := strings.ReplaceAll(name, " ", "_")
	sanitizedName = strings.ReplaceAll(sanitizedName, "/", "_")
	sanitizedName = strings.ReplaceAll(sanitizedName, "\\", "_")

	return fmt.Sprintf("%s_%s.yaml", id, sanitizedName)
}

func generatePipelineID() string {
	return fmt.Sprintf("pipeline_%d", time.Now().UnixNano())
}

// Global pipeline store instance
var globalPipelineStore *PipelineStore
var pipelineStoreOnce sync.Once

// ResetGlobalPipelineStore resets the global store (for testing)
func ResetGlobalPipelineStore() {
	globalPipelineStore = nil
	pipelineStoreOnce = sync.Once{}
}

// GetPipelineStore returns the global pipeline store instance
func GetPipelineStore() *PipelineStore {
	pipelineStoreOnce.Do(func() {
		globalPipelineStore = NewPipelineStore("./pipelines")
	})
	return globalPipelineStore
}

// InitializeGlobalPipelineStore initializes the global pipeline store
func InitializeGlobalPipelineStore(storePath string) error {
	store := GetPipelineStore()
	GetLogger().Info("=== InitializeGlobalPipelineStore: store pointer", String("ptr", fmt.Sprintf("%p", store)))
	if storePath != "" {
		store.storePath = storePath
	}
	err := store.Initialize()
	GetLogger().Info("=== After Initialize: pipeline count", Int("count", len(store.pipelines)))
	return err
}
