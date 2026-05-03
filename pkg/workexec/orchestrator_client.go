package workexec

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"io"
	"log"
	"net/http"
	"os"
)

func getWorkerAuthToken() string {
	return os.Getenv("WORKER_AUTH_TOKEN")
}

func doOrchestratorRequest(method, url string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token := getWorkerAuthToken(); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return http.DefaultClient.Do(req)
}

func getOrchestratorURL() string {
	if url := os.Getenv("ORCHESTRATOR_URL"); url != "" {
		return url
	}
	return "http://orchestrator:8080"
}

// getWorkTaskFromAPI fetches a work task from the orchestrator API
func getWorkTaskFromAPI(orchestratorURL, taskID string) (*models.WorkTask, error) {
	url := fmt.Sprintf("%s/api/worktasks/%s", orchestratorURL, taskID)
	resp, err := doOrchestratorRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch work task: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var task models.WorkTask
	if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
		return nil, fmt.Errorf("failed to decode work task: %w", err)
	}

	return &task, nil
}

// updateWorkTaskStatus updates the work task status via orchestrator API
func updateWorkTaskStatus(orchestratorURL, taskID string, status models.WorkTaskStatus, errorMsg string) error {
	result := models.WorkTaskResult{
		WorkTaskID:   taskID,
		Status:       status,
		ErrorMessage: errorMsg,
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal status update: %w", err)
	}

	url := fmt.Sprintf("%s/api/worktasks/%s", orchestratorURL, taskID)
	resp, err := doOrchestratorRequest(http.MethodPost, url, bytes.NewBuffer(resultJSON))
	if err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// reportWorkTaskCompletion reports work task completion to the orchestrator.
func reportWorkTaskCompletion(orchestratorURL, taskID string, status models.WorkTaskStatus, outputLocation, errorMsg string, metadata map[string]any) {
	result := models.WorkTaskResult{
		WorkTaskID:     taskID,
		Status:         status,
		OutputLocation: outputLocation,
		Metadata:       metadata,
		ErrorMessage:   errorMsg,
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		log.Printf("Failed to marshal work task result: %v", err)
		return
	}

	url := fmt.Sprintf("%s/api/worktasks/%s", orchestratorURL, taskID)
	resp, err := doOrchestratorRequest(http.MethodPost, url, bytes.NewBuffer(resultJSON))
	if err != nil {
		log.Printf("Failed to report work task completion: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Unexpected status code when reporting work task completion: %d", resp.StatusCode)
	}
}
