package workexec

import (
	"fmt"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"log"
	"os"
)

// RunFromEnvironment executes one work task using the current process environment.
// It keeps the existing worker contract so both the standalone worker binary and
// future in-process local execution can share one implementation.
func RunFromEnvironment() error {
	taskID := os.Getenv("WORKTASK_ID")
	taskType := os.Getenv("WORKTASK_TYPE")

	if taskID == "" || taskType == "" {
		return fmt.Errorf("WORKTASK_ID and WORKTASK_TYPE must be set")
	}

	log.Printf("Worker starting for task %s (type: %s)", taskID, taskType)

	orchestratorURL := getOrchestratorURL()
	task, err := getWorkTaskFromAPI(orchestratorURL, taskID)
	if err != nil {
		return fmt.Errorf("failed to get work task details: %w", err)
	}

	if err := updateWorkTaskStatus(orchestratorURL, taskID, models.WorkTaskStatusExecuting, ""); err != nil {
		log.Printf("Warning: Failed to update work task status to executing: %v", err)
	}

	result, err := executeWorkTask(task)
	if err != nil {
		log.Printf("Work task execution failed: %v", err)
		reportWorkTaskCompletion(orchestratorURL, taskID, models.WorkTaskStatusFailed, "", err.Error(), nil)
		return err
	}

	log.Printf("Work task %s completed successfully", taskID)
	reportWorkTaskCompletion(orchestratorURL, taskID, models.WorkTaskStatusCompleted, result.OutputLocation, "", result.Metadata)
	return nil
}

// executeWorkTask executes the work task based on its type
func executeWorkTask(task *models.WorkTask) (*models.WorkTaskResult, error) {
	log.Printf("Executing work task of type: %s", task.Type)

	switch task.Type {
	case models.WorkTaskTypePipelineExecution:
		return executePipeline(task)
	case models.WorkTaskTypeMLTraining:
		return executeMLTraining(task)
	case models.WorkTaskTypeMLInference:
		return executeMLInference(task)
	case models.WorkTaskTypeDigitalTwinProcessing:
		return executeDigitalTwinProcessing(task)
	default:
		return nil, fmt.Errorf("unknown work task type: %s", task.Type)
	}
}
