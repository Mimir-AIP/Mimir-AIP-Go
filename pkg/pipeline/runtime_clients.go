package pipeline

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

// HTTPStorageClient persists CIR records through the orchestrator storage API.
type HTTPStorageClient struct {
	baseURL string
	client  *http.Client
}

func NewHTTPStorageClient(baseURL string) *HTTPStorageClient {
	return &HTTPStorageClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *HTTPStorageClient) Store(storageID string, cir *models.CIR) (*models.StorageResult, error) {
	body, err := json.Marshal(models.StorageStoreRequest{StorageID: storageID, CIRData: cir})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal storage request: %w", err)
	}

	resp, err := c.client.Post(c.baseURL+"/api/storage/store", "application/json", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to call storage API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("storage API returned %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var result models.StorageResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode storage response: %w", err)
	}
	return &result, nil
}

// HTTPCheckpointStore persists step checkpoints through the orchestrator pipeline API.
type HTTPCheckpointStore struct {
	baseURL string
	client  *http.Client
}

func NewHTTPCheckpointStore(baseURL string) *HTTPCheckpointStore {
	return &HTTPCheckpointStore{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *HTTPCheckpointStore) GetPipelineCheckpoint(projectID, pipelineID, stepName, scope string) (*models.PipelineCheckpoint, error) {
	query := url.Values{}
	query.Set("step_name", stepName)
	if scope != "" {
		query.Set("scope", scope)
	}
	if projectID != "" {
		query.Set("project_id", projectID)
	}

	resp, err := c.client.Get(fmt.Sprintf("%s/api/pipelines/%s/checkpoints?%s", c.baseURL, pipelineID, query.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch pipeline checkpoint: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("checkpoint API returned %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var checkpoint models.PipelineCheckpoint
	if err := json.NewDecoder(resp.Body).Decode(&checkpoint); err != nil {
		return nil, fmt.Errorf("failed to decode checkpoint response: %w", err)
	}
	return &checkpoint, nil
}

func (c *HTTPCheckpointStore) SavePipelineCheckpoint(checkpoint *models.PipelineCheckpoint) error {
	if checkpoint == nil {
		return fmt.Errorf("checkpoint is required")
	}

	query := url.Values{}
	query.Set("step_name", checkpoint.StepName)
	if checkpoint.Scope != "" {
		query.Set("scope", checkpoint.Scope)
	}
	if checkpoint.ProjectID != "" {
		query.Set("project_id", checkpoint.ProjectID)
	}

	body, err := json.Marshal(checkpoint)
	if err != nil {
		return fmt.Errorf("failed to marshal checkpoint: %w", err)
	}

	req, err := http.NewRequest(http.MethodPut, fmt.Sprintf("%s/api/pipelines/%s/checkpoints?%s", c.baseURL, checkpoint.PipelineID, query.Encode()), bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to build checkpoint request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to save checkpoint: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("checkpoint API returned %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var persisted models.PipelineCheckpoint
	if err := json.NewDecoder(resp.Body).Decode(&persisted); err != nil {
		return fmt.Errorf("failed to decode saved checkpoint: %w", err)
	}
	*checkpoint = persisted
	return nil
}
