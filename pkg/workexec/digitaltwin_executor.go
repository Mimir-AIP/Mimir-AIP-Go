package workexec

import (
	"encoding/json"
	"fmt"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"log"
	"net/http"
)

// executeDigitalTwinProcessing executes one explicit twin-processing run using only
// task parameters persisted by the orchestrator. Workers remain stateless.
func executeDigitalTwinProcessing(task *models.WorkTask) (*models.WorkTaskResult, error) {
	log.Printf("Executing digital twin processing for project: %s", task.ProjectID)

	orchestratorURL := getOrchestratorURL()
	runID, _ := task.TaskSpec.Parameters["processing_run_id"].(string)
	twinID, _ := task.TaskSpec.Parameters["digital_twin_id"].(string)
	if runID == "" {
		return nil, fmt.Errorf("processing_run_id not specified in task parameters")
	}
	if twinID == "" {
		return nil, fmt.Errorf("digital_twin_id not specified in task parameters")
	}

	executeURL := fmt.Sprintf("%s/api/internal/twin-runs/%s/execute", orchestratorURL, runID)
	resp, err := doOrchestratorRequest(http.MethodPost, executeURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to execute twin processing run: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("twin processing run execution failed: status %d", resp.StatusCode)
	}

	var run models.TwinProcessingRun
	if err := json.NewDecoder(resp.Body).Decode(&run); err != nil {
		return nil, fmt.Errorf("failed to decode twin processing run: %w", err)
	}

	outputLocation := fmt.Sprintf("/tmp/digital-twins/%s/runs/%s", twinID, runID)
	log.Printf("Digital twin processing run %s completed with status %s", runID, run.Status)
	return &models.WorkTaskResult{
		WorkTaskID:     task.ID,
		Status:         models.WorkTaskStatusCompleted,
		OutputLocation: outputLocation,
		Metadata: map[string]any{
			"project_id":        task.ProjectID,
			"digital_twin_id":   twinID,
			"processing_run_id": runID,
			"run_status":        run.Status,
			"trigger_type":      run.TriggerType,
			"insight_count":     run.Metrics["insight_count"],
		},
	}, nil
}
