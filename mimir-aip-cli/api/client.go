package api

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client manages communication with the backend API
// Add fields for base URL, auth token, etc.
type Client struct {
	BaseURL   string
	AuthToken string
	HTTP      *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		BaseURL: baseURL,
		HTTP:    &http.Client{Timeout: 10 * time.Second},
	}
}

// HealthCheck pings the /health endpoint
func (c *Client) HealthCheck() error {
	resp, err := c.HTTP.Get(c.BaseURL + "/health")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("health check failed: %s", resp.Status)
	}
	return nil
}

// Pipelines
func (c *Client) ListPipelines() ([]byte, error) { return c.get("/api/v1/pipelines") }
func (c *Client) CreatePipeline(data []byte) ([]byte, error) {
	return c.post("/api/v1/pipelines", data)
}
func (c *Client) GetPipeline(id string) ([]byte, error) { return c.get("/api/v1/pipelines/" + id) }
func (c *Client) UpdatePipeline(id string, data []byte) ([]byte, error) {
	return c.put("/api/v1/pipelines/"+id, data)
}
func (c *Client) DeletePipeline(id string) error {
	_, err := c.delete("/api/v1/pipelines/" + id)
	return err
}
func (c *Client) ClonePipeline(id string) ([]byte, error) {
	return c.post("/api/v1/pipelines/"+id+"/clone", nil)
}
func (c *Client) ValidatePipeline(id string) ([]byte, error) {
	return c.post("/api/v1/pipelines/"+id+"/validate", nil)
}
func (c *Client) GetPipelineHistory(id string) ([]byte, error) {
	return c.get("/api/v1/pipelines/" + id + "/history")
}
func (c *Client) ExecutePipeline(data []byte) ([]byte, error) {
	return c.post("/api/v1/pipelines/execute", data)
}

// Jobs
func (c *Client) ListJobs() ([]byte, error)        { return c.get("/api/v1/scheduler/jobs") }
func (c *Client) GetJob(id string) ([]byte, error) { return c.get("/api/v1/scheduler/jobs/" + id) }
func (c *Client) CreateJob(data []byte) ([]byte, error) {
	return c.post("/api/v1/scheduler/jobs", data)
}
func (c *Client) DeleteJob(id string) error {
	_, err := c.delete("/api/v1/scheduler/jobs/" + id)
	return err
}
func (c *Client) EnableJob(id string) error {
	_, err := c.post("/api/v1/scheduler/jobs/"+id+"/enable", nil)
	return err
}
func (c *Client) DisableJob(id string) error {
	_, err := c.post("/api/v1/scheduler/jobs/"+id+"/disable", nil)
	return err
}

// Plugins
func (c *Client) ListPlugins() ([]byte, error) { return c.get("/api/v1/plugins") }
func (c *Client) ListPluginsByType(typ string) ([]byte, error) {
	return c.get("/api/v1/plugins/" + typ)
}
func (c *Client) GetPlugin(typ, name string) ([]byte, error) {
	return c.get("/api/v1/plugins/" + typ + "/" + name)
}

// Config
func (c *Client) GetConfig() ([]byte, error)               { return c.get("/api/v1/config") }
func (c *Client) UpdateConfig(data []byte) ([]byte, error) { return c.put("/api/v1/config", data) }
func (c *Client) ReloadConfig() error                      { _, err := c.post("/api/v1/config/reload", nil); return err }
func (c *Client) SaveConfig() error                        { _, err := c.post("/api/v1/config/save", nil); return err }

// Performance
func (c *Client) GetPerformanceMetrics() ([]byte, error) { return c.get("/api/v1/performance/metrics") }
func (c *Client) GetPerformanceStats() ([]byte, error)   { return c.get("/api/v1/performance/stats") }

// Visualization
func (c *Client) VisualizePipeline(data []byte) ([]byte, error) {
	return c.post("/api/v1/visualize/pipeline", data)
}
func (c *Client) VisualizeStatus() ([]byte, error)    { return c.get("/api/v1/visualize/status") }
func (c *Client) VisualizeScheduler() ([]byte, error) { return c.get("/api/v1/visualize/scheduler") }
func (c *Client) VisualizePlugins() ([]byte, error)   { return c.get("/api/v1/visualize/plugins") }

// Auth
func (c *Client) Login(data []byte) ([]byte, error) { return c.post("/api/v1/auth/login", data) }
func (c *Client) RefreshToken(data []byte) ([]byte, error) {
	return c.post("/api/v1/auth/refresh", data)
}
func (c *Client) AuthMe() ([]byte, error)    { return c.get("/api/v1/auth/me") }
func (c *Client) ListUsers() ([]byte, error) { return c.get("/api/v1/auth/users") }
func (c *Client) CreateAPIKey(data []byte) ([]byte, error) {
	return c.post("/api/v1/auth/apikeys", data)
}

// Helper HTTP methods
func (c *Client) get(path string) ([]byte, error) {
	req, err := http.NewRequest("GET", c.BaseURL+path, nil)
	if err != nil {
		return nil, err
	}
	c.setAuth(req)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return readBody(resp)
}
func (c *Client) post(path string, data []byte) ([]byte, error) {
	req, err := http.NewRequest("POST", c.BaseURL+path, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	c.setAuth(req)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return readBody(resp)
}
func (c *Client) put(path string, data []byte) ([]byte, error) {
	req, err := http.NewRequest("PUT", c.BaseURL+path, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	c.setAuth(req)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return readBody(resp)
}
func (c *Client) delete(path string) ([]byte, error) {
	req, err := http.NewRequest("DELETE", c.BaseURL+path, nil)
	if err != nil {
		return nil, err
	}
	c.setAuth(req)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return readBody(resp)
}
func (c *Client) setAuth(req *http.Request) {
	if c.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.AuthToken)
	}
}
func readBody(resp *http.Response) ([]byte, error) {
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP error: %s", resp.Status)
	}
	return io.ReadAll(resp.Body)
}
