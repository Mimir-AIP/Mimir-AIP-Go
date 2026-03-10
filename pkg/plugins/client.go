package plugins

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"time"

	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"github.com/mimir-aip/mimir-aip-go/pkg/pipeline"
	"github.com/mimir-aip/mimir-aip-go/pkg/pluginruntime"
)

// Client fetches pipeline plugin metadata from the orchestrator and compiles the
// plugin locally with the worker's Go toolchain.
type Client struct {
	baseURL    string
	httpClient *http.Client
	runtime    *pluginruntime.Loader[pipeline.Plugin]
	initErr    error
}

// NewClient creates a new plugin registry client.
func NewClient(baseURL, cacheDir string) *Client {
	runtime, err := pluginruntime.NewLoader(pluginruntime.BuildSpec[pipeline.Plugin]{
		LogPrefix:      "pipeline plugin loader",
		AppDir:         "/app",
		CacheDir:       cacheDir,
		TempDir:        filepath.Join(cacheDir, "tmp"),
		HostPackageDir: "plugins",
		ClonePrefix:    "pp",
		SymbolName:     "Plugin",
		DefaultGitRef:  "main",
		Resolver:       pluginruntime.ResolveSymbol[pipeline.Plugin],
	})
	client := &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		runtime: runtime,
		initErr: err,
	}
	return client
}

// CompilePlugin compiles a plugin from source in the local worker environment.
func (c *Client) CompilePlugin(name string) (string, error) {
	if c.initErr != nil {
		return "", fmt.Errorf("failed to initialise plugin client: %w", c.initErr)
	}
	metadata, err := c.FetchPluginMetadata(name)
	if err != nil {
		return "", fmt.Errorf("failed to fetch plugin metadata: %w", err)
	}
	if _, _, err := c.runtime.CompileAndLoad(name, metadata.RepositoryURL, metadata.GitCommitHash, metadata.GitCommitHash); err != nil {
		return "", fmt.Errorf("failed to compile plugin: %w", err)
	}
	return c.runtime.SoPath(name), nil
}

// LoadPlugin loads a compiled plugin from the local cache.
func (c *Client) LoadPlugin(name string) (pipeline.Plugin, error) {
	if c.initErr != nil {
		return nil, fmt.Errorf("failed to initialise plugin client: %w", c.initErr)
	}
	return c.runtime.LoadCached(name)
}

// FetchPluginMetadata fetches plugin metadata from the registry.
func (c *Client) FetchPluginMetadata(name string) (*models.Plugin, error) {
	url := fmt.Sprintf("%s/api/plugins/%s", c.baseURL, name)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch plugin metadata: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch plugin metadata (status %d): %s", resp.StatusCode, string(body))
	}

	var plugin models.Plugin
	if err := json.NewDecoder(resp.Body).Decode(&plugin); err != nil {
		return nil, fmt.Errorf("failed to decode plugin metadata: %w", err)
	}

	return &plugin, nil
}
