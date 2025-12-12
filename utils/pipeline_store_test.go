package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestNewPipelineStore(t *testing.T) {
	storePath := "./test_store"
	store := NewPipelineStore(storePath)

	assert.NotNil(t, store)
	assert.Equal(t, storePath, store.storePath)
	assert.NotNil(t, store.pipelines)
	assert.Equal(t, 0, len(store.pipelines))
}

func TestPipelineStoreInitialize(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "pipeline_store_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	store := NewPipelineStore(tempDir)

	err = store.Initialize()
	assert.NoError(t, err)

	// Check that directory was created
	_, err = os.Stat(tempDir)
	assert.NoError(t, err)
}

func TestPipelineStoreCreatePipeline(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "pipeline_store_create_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	store := NewPipelineStore(tempDir)
	err = store.Initialize()
	require.NoError(t, err)

	metadata := PipelineMetadata{
		Name:        "test-pipeline",
		Description: "A test pipeline",
		Tags:        []string{"test", "example"},
		Enabled:     true,
		Schedule:    "0 0 * * *",
	}

	config := PipelineConfig{
		Name:    "test-pipeline",
		Enabled: true,
		Steps: []pipelines.StepConfig{
			{
				Name:   "step1",
				Plugin: "Input.test",
				Config: map[string]interface{}{"param": "value"},
			},
		},
	}

	pipeline, err := store.CreatePipeline(metadata, config, "test-user")
	assert.NoError(t, err)
	assert.NotNil(t, pipeline)

	// Check that metadata was set correctly
	assert.Equal(t, "test-pipeline", pipeline.Metadata.Name)
	assert.Equal(t, "A test pipeline", pipeline.Metadata.Description)
	assert.Equal(t, []string{"test", "example"}, pipeline.Metadata.Tags)
	assert.True(t, pipeline.Metadata.Enabled)
	assert.Equal(t, "0 0 * * *", pipeline.Metadata.Schedule)
	assert.Equal(t, "test-user", pipeline.Metadata.CreatedBy)
	assert.Equal(t, "test-user", pipeline.Metadata.UpdatedBy)
	assert.Equal(t, 1, pipeline.Metadata.Version)
	assert.False(t, pipeline.Metadata.CreatedAt.IsZero())
	assert.False(t, pipeline.Metadata.UpdatedAt.IsZero())

	// Check that pipeline was stored
	storedPipeline, err := store.GetPipeline(pipeline.Metadata.ID)
	assert.NoError(t, err)
	assert.Equal(t, pipeline.Metadata.ID, storedPipeline.Metadata.ID)
}

func TestPipelineStoreCreateDuplicatePipeline(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "pipeline_store_duplicate_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	store := NewPipelineStore(tempDir)
	err = store.Initialize()
	require.NoError(t, err)

	metadata := PipelineMetadata{
		ID:   "duplicate-id",
		Name: "test-pipeline",
	}

	config := PipelineConfig{
		Name:    "test-pipeline",
		Enabled: true,
		Steps: []pipelines.StepConfig{
			{Name: "step1", Plugin: "Input.test"},
		},
	}

	// Create first pipeline
	_, err = store.CreatePipeline(metadata, config, "test-user")
	assert.NoError(t, err)

	// Try to create duplicate
	_, err = store.CreatePipeline(metadata, config, "test-user")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestPipelineStoreGetPipeline(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "pipeline_store_get_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	store := NewPipelineStore(tempDir)
	err = store.Initialize()
	require.NoError(t, err)

	// Test getting non-existent pipeline
	_, err = store.GetPipeline("non-existent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	// Create a pipeline
	metadata := PipelineMetadata{Name: "test-pipeline"}
	config := PipelineConfig{
		Name:    "test-pipeline",
		Enabled: true,
		Steps: []pipelines.StepConfig{
			{Name: "step1", Plugin: "Input.test"},
		},
	}

	pipeline, err := store.CreatePipeline(metadata, config, "test-user")
	require.NoError(t, err)

	// Get the pipeline
	retrievedPipeline, err := store.GetPipeline(pipeline.Metadata.ID)
	assert.NoError(t, err)
	assert.Equal(t, pipeline.Metadata.ID, retrievedPipeline.Metadata.ID)
	assert.Equal(t, "test-pipeline", retrievedPipeline.Metadata.Name)
}

func TestPipelineStoreListPipelines(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "pipeline_store_list_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	store := NewPipelineStore(tempDir)
	err = store.Initialize()
	require.NoError(t, err)

	// Create multiple pipelines
	for i := 0; i < 3; i++ {
		metadata := PipelineMetadata{
			Name:    fmt.Sprintf("pipeline-%d", i),
			Enabled: i%2 == 0, // Alternate enabled/disabled
			Tags:    []string{fmt.Sprintf("tag-%d", i)},
		}
		config := PipelineConfig{
			Name:    "test-pipeline",
			Enabled: true,
			Steps: []pipelines.StepConfig{
				{Name: "step1", Plugin: "Input.test"},
			},
		}
		_, err := store.CreatePipeline(metadata, config, "test-user")
		require.NoError(t, err)
	}

	// List all pipelines
	allPipelines, err := store.ListPipelines(nil)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(allPipelines))

	// List only enabled pipelines
	enabledFilter := map[string]interface{}{"enabled": true}
	enabledPipelines, err := store.ListPipelines(enabledFilter)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(enabledPipelines)) // pipeline-0 and pipeline-2

	// Filter by name
	nameFilter := map[string]interface{}{"name": "pipeline-1"}
	namePipelines, err := store.ListPipelines(nameFilter)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(namePipelines))
	assert.Equal(t, "pipeline-1", namePipelines[0].Metadata.Name)

	// Filter by tags
	tagFilter := map[string]interface{}{"tags": []string{"tag-0"}}
	tagPipelines, err := store.ListPipelines(tagFilter)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(tagPipelines))
	assert.Equal(t, "pipeline-0", tagPipelines[0].Metadata.Name)
}

func TestPipelineStoreUpdatePipeline(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "pipeline_store_update_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	store := NewPipelineStore(tempDir)
	err = store.Initialize()
	require.NoError(t, err)

	// Create a pipeline
	metadata := PipelineMetadata{
		Name:        "original-name",
		Description: "Original description",
		Tags:        []string{"original"},
		Enabled:     true,
	}
	config := PipelineConfig{
		Name:    "original-name",
		Enabled: true,
		Steps: []pipelines.StepConfig{
			{Name: "step1", Plugin: "Input.test"},
		},
	}

	pipeline, err := store.CreatePipeline(metadata, config, "original-user")
	require.NoError(t, err)
	originalVersion := pipeline.Metadata.Version

	// Update metadata
	updatedMetadata := &PipelineMetadata{
		Name:        "updated-name",
		Description: "Updated description",
		Tags:        []string{"updated", "new-tag"},
		Enabled:     false,
	}

	updatedPipeline, err := store.UpdatePipeline(pipeline.Metadata.ID, updatedMetadata, nil, "update-user")
	assert.NoError(t, err)
	assert.Equal(t, "updated-name", updatedPipeline.Metadata.Name)
	assert.Equal(t, "Updated description", updatedPipeline.Metadata.Description)
	assert.Equal(t, []string{"updated", "new-tag"}, updatedPipeline.Metadata.Tags)
	assert.False(t, updatedPipeline.Metadata.Enabled)
	assert.Equal(t, "update-user", updatedPipeline.Metadata.UpdatedBy)
	assert.Equal(t, originalVersion+1, updatedPipeline.Metadata.Version)

	// Update config
	newConfig := &PipelineConfig{
		Name:    "updated-name",
		Enabled: true,
		Steps: []pipelines.StepConfig{
			{Name: "new-step1", Plugin: "Output.test"},
			{Name: "new-step2", Plugin: "Processing.test"},
		},
	}

	reupdatedPipeline, err := store.UpdatePipeline(pipeline.Metadata.ID, nil, newConfig, "config-user")
	assert.NoError(t, err)
	assert.Equal(t, 2, len(reupdatedPipeline.Config.Steps))
	assert.Equal(t, "new-step1", reupdatedPipeline.Config.Steps[0].Name)
	assert.Equal(t, "config-user", reupdatedPipeline.Metadata.UpdatedBy)
	assert.Equal(t, originalVersion+2, reupdatedPipeline.Metadata.Version)
}

func TestPipelineStoreUpdateNonExistentPipeline(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "pipeline_store_update_nonexistent_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	store := NewPipelineStore(tempDir)
	err = store.Initialize()
	require.NoError(t, err)

	updates := &PipelineMetadata{Name: "updated"}
	_, err = store.UpdatePipeline("non-existent", updates, nil, "test-user")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestPipelineStoreDeletePipeline(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "pipeline_store_delete_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	store := NewPipelineStore(tempDir)
	err = store.Initialize()
	require.NoError(t, err)

	// Create a pipeline
	metadata := PipelineMetadata{Name: "test-pipeline"}
	config := PipelineConfig{
		Name:    "test-pipeline",
		Enabled: true,
		Steps: []pipelines.StepConfig{
			{Name: "step1", Plugin: "Input.test"},
		},
	}

	pipeline, err := store.CreatePipeline(metadata, config, "test-user")
	require.NoError(t, err)

	// Delete the pipeline
	err = store.DeletePipeline(pipeline.Metadata.ID)
	assert.NoError(t, err)

	// Verify it's gone
	_, err = store.GetPipeline(pipeline.Metadata.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestPipelineStoreDeleteNonExistentPipeline(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "pipeline_store_delete_nonexistent_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	store := NewPipelineStore(tempDir)
	err = store.Initialize()
	require.NoError(t, err)

	err = store.DeletePipeline("non-existent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestPipelineStoreValidatePipeline(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "pipeline_store_validate_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	store := NewPipelineStore(tempDir)
	err = store.Initialize()
	require.NoError(t, err)

	// Test valid pipeline
	validPipeline := &PipelineDefinition{
		Metadata: PipelineMetadata{Name: "valid-pipeline"},
		Config: PipelineConfig{
			Name:    "valid-pipeline",
			Enabled: true,
			Steps: []pipelines.StepConfig{
				{Name: "step1", Plugin: "Input.test"},
			},
		},
	}

	err = store.ValidatePipeline(validPipeline)
	assert.NoError(t, err)

	// Test invalid pipeline - missing name
	invalidPipeline := &PipelineDefinition{
		Metadata: PipelineMetadata{},
		Config: PipelineConfig{
			Name:    "",
			Enabled: true,
			Steps: []pipelines.StepConfig{
				{Name: "step1", Plugin: "Input.test"},
			},
		},
	}

	err = store.ValidatePipeline(invalidPipeline)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "name is required")

	// Test invalid pipeline - no steps
	invalidPipeline2 := &PipelineDefinition{
		Metadata: PipelineMetadata{Name: "no-steps"},
		Config: PipelineConfig{
			Name:    "no-steps",
			Enabled: true,
			Steps:   []pipelines.StepConfig{}, // Empty steps
		},
	}

	err = store.ValidatePipeline(invalidPipeline2)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "at least one step")
}

func TestPipelineStoreClonePipeline(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "pipeline_store_clone_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	store := NewPipelineStore(tempDir)
	err = store.Initialize()
	require.NoError(t, err)

	// Create original pipeline
	metadata := PipelineMetadata{
		Name:        "original-pipeline",
		Description: "Original description",
		Tags:        []string{"original"},
		Enabled:     true,
	}
	config := PipelineConfig{
		Name:    "original-pipeline",
		Enabled: true,
		Steps: []pipelines.StepConfig{
			{Name: "step1", Plugin: "Input.test"},
		},
	}

	original, err := store.CreatePipeline(metadata, config, "original-user")
	require.NoError(t, err)

	// Clone the pipeline
	cloned, err := store.ClonePipeline(original.Metadata.ID, "cloned-pipeline", "clone-user")
	assert.NoError(t, err)
	assert.NotEqual(t, original.Metadata.ID, cloned.Metadata.ID)
	assert.Equal(t, "cloned-pipeline", cloned.Metadata.Name)
	assert.Equal(t, "Original description", cloned.Metadata.Description)
	assert.Equal(t, []string{"original"}, cloned.Metadata.Tags)
	assert.Equal(t, "clone-user", cloned.Metadata.CreatedBy)
	assert.Equal(t, "clone-user", cloned.Metadata.UpdatedBy)
	assert.Equal(t, 1, cloned.Metadata.Version)
	assert.Equal(t, 1, len(cloned.Config.Steps))
}

func TestPipelineStoreGetPipelineHistory(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "pipeline_store_history_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	store := NewPipelineStore(tempDir)
	err = store.Initialize()
	require.NoError(t, err)

	// Create a pipeline
	metadata := PipelineMetadata{Name: "test-pipeline"}
	config := PipelineConfig{
		Name:    "test-pipeline",
		Enabled: true,
		Steps: []pipelines.StepConfig{
			{Name: "step1", Plugin: "Input.test"},
		},
	}

	pipeline, err := store.CreatePipeline(metadata, config, "test-user")
	require.NoError(t, err)

	// Get history
	history, err := store.GetPipelineHistory(pipeline.Metadata.ID)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(history))
	assert.Equal(t, pipeline.Metadata.ID, history[0].Metadata.ID)
}

func TestPipelineStoreLoadPipelineFromFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "pipeline_store_load_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	store := NewPipelineStore(tempDir)

	// Create a valid YAML pipeline file
	pipelineDef := &PipelineDefinition{
		Metadata: PipelineMetadata{
			ID:          "test-id",
			Name:        "test-pipeline",
			Description: "Test pipeline",
			Enabled:     true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			Version:     1,
		},
		Config: PipelineConfig{
			Name:    "test-pipeline",
			Enabled: true,
			Steps: []pipelines.StepConfig{
				{Name: "step1", Plugin: "Input.test"},
			},
		},
	}

	yamlData, err := yaml.Marshal(pipelineDef)
	require.NoError(t, err)

	filePath := filepath.Join(tempDir, "test-pipeline.yaml")
	err = os.WriteFile(filePath, yamlData, 0644)
	require.NoError(t, err)

	// Load the pipeline
	loadedPipeline, err := store.loadPipelineFromFile(filePath)
	assert.NoError(t, err)
	assert.Equal(t, "test-id", loadedPipeline.Metadata.ID)
	assert.Equal(t, "test-pipeline", loadedPipeline.Metadata.Name)
	assert.Equal(t, 1, len(loadedPipeline.Config.Steps))
}

func TestPipelineStoreSavePipelineToFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "pipeline_store_save_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	store := NewPipelineStore(tempDir)

	pipelineDef := &PipelineDefinition{
		Metadata: PipelineMetadata{
			ID:          "test-id",
			Name:        "test-pipeline",
			Description: "Test pipeline",
			Enabled:     true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			Version:     1,
		},
		Config: PipelineConfig{
			Name:    "test-pipeline",
			Enabled: true,
			Steps: []pipelines.StepConfig{
				{Name: "step1", Plugin: "Input.test"},
			},
		},
	}

	// Save the pipeline
	err = store.savePipelineToFile(pipelineDef)
	assert.NoError(t, err)

	// Check that file was created
	fileName := store.getPipelineFileName(pipelineDef.Metadata.ID, pipelineDef.Metadata.Name)
	filePath := filepath.Join(tempDir, fileName)
	_, err = os.Stat(filePath)
	assert.NoError(t, err)

	// Load and verify content
	loadedPipeline, err := store.loadPipelineFromFile(filePath)
	assert.NoError(t, err)
	assert.Equal(t, pipelineDef.Metadata.ID, loadedPipeline.Metadata.ID)
	assert.Equal(t, pipelineDef.Metadata.Name, loadedPipeline.Metadata.Name)
}

func TestPipelineStoreGetPipelineFileName(t *testing.T) {
	store := NewPipelineStore("./test")

	tests := []struct {
		id       string
		name     string
		expected string
	}{
		{
			id:       "pipeline_123",
			name:     "test pipeline",
			expected: "pipeline_123_test_pipeline.yaml",
		},
		{
			id:       "pipeline_456",
			name:     "path/to/file",
			expected: "pipeline_456_path_to_file.yaml",
		},
		{
			id:       "pipeline_789",
			name:     "file\\with\\backslashes",
			expected: "pipeline_789_file_with_backslashes.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := store.getPipelineFileName(tt.id, tt.name)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGeneratePipelineID(t *testing.T) {
	id1 := generatePipelineID()
	id2 := generatePipelineID()

	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.NotEqual(t, id1, id2)
	assert.Contains(t, id1, "pipeline_")
	assert.Contains(t, id2, "pipeline_")
}

func TestGlobalPipelineStore(t *testing.T) {
	// Reset global store
	globalPipelineStore = nil
	pipelineStoreOnce = sync.Once{}

	store := GetPipelineStore()
	assert.NotNil(t, store)
	assert.Equal(t, "./pipelines", store.storePath)

	// Test singleton behavior
	store2 := GetPipelineStore()
	assert.Equal(t, store, store2)
}

func TestInitializeGlobalPipelineStore(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "global_pipeline_store_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Reset global store
	globalPipelineStore = nil
	pipelineStoreOnce = sync.Once{}

	err = InitializeGlobalPipelineStore(tempDir)
	assert.NoError(t, err)

	store := GetPipelineStore()
	assert.Equal(t, tempDir, store.storePath)
}
