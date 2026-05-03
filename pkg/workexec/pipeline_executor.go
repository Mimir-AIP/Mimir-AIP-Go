package workexec

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	pipelinepkg "github.com/mimir-aip/mimir-aip-go/pkg/pipeline"
	"github.com/mimir-aip/mimir-aip-go/pkg/pluginruntime"
	"github.com/mimir-aip/mimir-aip-go/pkg/plugins"
	"log"
	"net/http"
	"os"
	"time"
)

// executePipeline executes a pipeline work task
func executePipeline(task *models.WorkTask) (*models.WorkTaskResult, error) {
	log.Printf("Executing pipeline: %s", task.TaskSpec.PipelineID)

	orchestratorURL := getOrchestratorURL()

	pipelineURL := fmt.Sprintf("%s/api/pipelines/%s", orchestratorURL, task.TaskSpec.PipelineID)
	resp, err := http.Get(pipelineURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch pipeline: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch pipeline: status %d", resp.StatusCode)
	}

	var pipeline models.Pipeline
	if err := json.NewDecoder(resp.Body).Decode(&pipeline); err != nil {
		return nil, fmt.Errorf("failed to decode pipeline: %w", err)
	}

	log.Printf("Executing pipeline %s (%s) with %d steps", pipeline.Name, pipeline.ID, len(pipeline.Steps))

	context := models.NewPipelineContext(10485760)
	if task.TaskSpec.Parameters != nil {
		for key, value := range task.TaskSpec.Parameters {
			context.SetStepData("_parameters", key, value)
		}
	}

	context.SetStepData("_runtime", "project_id", pipeline.ProjectID)
	context.SetStepData("_runtime", "pipeline_id", pipeline.ID)
	context.SetStepData("_runtime", "trigger_type", fmt.Sprintf("%v", task.TaskSpec.Parameters["trigger_type"]))

	pluginCacheDir := "/tmp/plugins"
	pluginClient := plugins.NewClient(orchestratorURL, pluginCacheDir)

	pluginRegistry := pluginruntime.NewRegistry[pipelinepkg.Plugin]()
	storageClient := pipelinepkg.NewHTTPStorageClient(orchestratorURL)
	checkpointClient := pipelinepkg.NewHTTPCheckpointStore(orchestratorURL)
	pluginRegistry.Register("default", pipelinepkg.NewDefaultPluginWithDeps(storageClient, checkpointClient))
	pluginRegistry.Register("builtin", pipelinepkg.NewDefaultPluginWithDeps(storageClient, checkpointClient))

	uniquePlugins := make(map[string]bool)
	for _, step := range pipeline.Steps {
		if step.Plugin != "default" && step.Plugin != "builtin" {
			uniquePlugins[step.Plugin] = true
		}
	}

	for pluginName := range uniquePlugins {
		log.Printf("Compiling plugin: %s", pluginName)
		pluginPath, err := pluginClient.CompilePlugin(pluginName)
		if err != nil {
			return nil, fmt.Errorf("failed to compile plugin %s: %w", pluginName, err)
		}

		log.Printf("Loading plugin from: %s", pluginPath)
		pluginInstance, err := pluginClient.LoadPlugin(pluginName)
		if err != nil {
			return nil, fmt.Errorf("failed to load plugin %s: %w", pluginName, err)
		}

		pluginRegistry.Register(pluginName, pluginInstance)
		log.Printf("Loaded custom plugin: %s", pluginName)
	}

	startTime := time.Now()
	currentStepIndex := 0
	stepsExecuted := 0

	for currentStepIndex < len(pipeline.Steps) {
		step := pipeline.Steps[currentStepIndex]
		stepsExecuted++

		log.Printf("  Step %d: %s (%s.%s)", currentStepIndex+1, step.Name, step.Plugin, step.Action)

		pluginInstance, ok := pluginRegistry.Get(step.Plugin)
		if !ok {
			return nil, fmt.Errorf("unknown plugin: %s", step.Plugin)
		}

		context.SetStepData("_runtime", "current_step", step.Name)

		result, err := pluginInstance.Execute(step.Action, step.Parameters, context)
		if err != nil {
			return nil, fmt.Errorf("step %s failed: %w", step.Name, err)
		}

		for key, value := range result {
			context.SetStepData(step.Name, key, value)
		}

		if step.Output != nil {
			for outputKey, outputTemplate := range step.Output {
				if dp, ok := pluginInstance.(*pipelinepkg.DefaultPlugin); ok {
					resolvedValue := dp.ResolveTemplates(outputTemplate, context)
					context.SetStepData(step.Name, outputKey, resolvedValue)
					log.Printf("    Output: %s = %v", outputKey, resolvedValue)
				}
			}
		}

		if gotoTarget, ok := result["goto"].(string); ok {
			targetIndex := -1
			for i, s := range pipeline.Steps {
				if s.Name == gotoTarget {
					targetIndex = i
					break
				}
			}

			if targetIndex == -1 {
				return nil, fmt.Errorf("goto target not found: %s", gotoTarget)
			}

			log.Printf("    Jumping to step: %s", gotoTarget)
			currentStepIndex = targetIndex
			continue
		}

		currentStepIndex++
	}

	executionTime := time.Since(startTime)
	log.Printf("Pipeline execution completed: %d steps executed in %v", stepsExecuted, executionTime)

	outputDir := fmt.Sprintf("/tmp/pipeline/%s", task.ID)
	outputPath := fmt.Sprintf("%s/context.json", outputDir)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Printf("Warning: failed to create pipeline output dir: %v", err)
		outputPath = ""
	} else {
		contextJSON, err := json.Marshal(context.Steps)
		if err != nil {
			log.Printf("Warning: failed to marshal pipeline context: %v", err)
			outputPath = ""
		} else if err := os.WriteFile(outputPath, contextJSON, 0644); err != nil {
			log.Printf("Warning: failed to write pipeline context: %v", err)
			outputPath = ""
		}
	}

	if pipelineType, ok := task.TaskSpec.Parameters["pipeline_type"].(string); ok && pipelineType == "ingestion" {
		triggerExtractionForIngestion(orchestratorURL, task.TaskSpec.ProjectID, task.TaskSpec.PipelineID)
	}

	return &models.WorkTaskResult{
		WorkTaskID:     task.ID,
		Status:         models.WorkTaskStatusCompleted,
		OutputLocation: outputPath,
		Metadata: map[string]any{
			"pipeline_id":       task.TaskSpec.PipelineID,
			"pipeline_name":     pipeline.Name,
			"steps_executed":    len(pipeline.Steps),
			"execution_time_ms": executionTime.Milliseconds(),
			"trigger_type":      task.TaskSpec.Parameters["trigger_type"],
			"triggered_by":      task.TaskSpec.Parameters["triggered_by"],
		},
	}, nil
}

// triggerExtractionForIngestion calls POST /api/extraction/generate-ontology for an ingestion pipeline.
//
// It discovers ALL storage configs for the project (not just those used by
// the current pipeline) so that cross-source link detection sees the full
// dataset. This enables a unified ontology across multiple ingestion pipelines
// feeding the same project.
//
// This is best-effort: failures are logged but do not fail the pipeline task.
func triggerExtractionForIngestion(orchestratorURL, projectID, pipelineID string) {
	storageIDs := fetchProjectStorageIDs(orchestratorURL, projectID)

	payload := map[string]interface{}{
		"project_id":           projectID,
		"storage_ids":          storageIDs,
		"ontology_name":        "auto-" + projectID,
		"include_structured":   true,
		"include_unstructured": true,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		log.Printf("auto-extraction: failed to marshal request: %v", err)
		return
	}

	url := fmt.Sprintf("%s/api/extraction/generate-ontology", orchestratorURL)
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		log.Printf("auto-extraction: HTTP call failed: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		log.Printf("auto-extraction: triggered for project %s (pipeline %s, %d storage sources)", projectID, pipelineID, len(storageIDs))
	} else {
		log.Printf("auto-extraction: unexpected status %d for project %s", resp.StatusCode, projectID)
	}
}

// fetchProjectStorageIDs retrieves the IDs of all storage configs for a project.
// Returns an empty slice on any error (extraction will still run but without data).
func fetchProjectStorageIDs(orchestratorURL, projectID string) []string {
	url := fmt.Sprintf("%s/api/storage/configs?project_id=%s", orchestratorURL, projectID)
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("auto-extraction: failed to fetch storage configs: %v", err)
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Printf("auto-extraction: unexpected status %d fetching storage configs", resp.StatusCode)
		return nil
	}
	var configs []struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&configs); err != nil {
		log.Printf("auto-extraction: failed to decode storage configs: %v", err)
		return nil
	}
	ids := make([]string, 0, len(configs))
	for _, c := range configs {
		if c.ID != "" {
			ids = append(ids, c.ID)
		}
	}
	return ids
}
