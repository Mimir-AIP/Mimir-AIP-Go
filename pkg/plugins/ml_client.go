package plugins

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"time"

	"github.com/mimir-aip/mimir-aip-go/pkg/mlmodel"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"github.com/mimir-aip/mimir-aip-go/pkg/pluginruntime"
)

// MLClient fetches ML provider plugin metadata from the orchestrator and compiles the
// provider locally with the worker's Go toolchain.
type MLClient struct {
	baseURL    string
	httpClient *http.Client
	runtime    *pluginruntime.Loader[mlmodel.Provider]
	initErr    error
}

func NewMLClient(baseURL, cacheDir string) *MLClient {
	runtime, err := pluginruntime.NewLoader(pluginruntime.BuildSpec[mlmodel.Provider]{
		LogPrefix:      "ml provider loader",
		AppDir:         "/app",
		CacheDir:       cacheDir,
		TempDir:        filepath.Join(cacheDir, "tmp"),
		HostPackageDir: "plugins",
		ClonePrefix:    "mlp",
		SymbolName:     "MLProvider",
		DefaultGitRef:  "main",
		Resolver:       pluginruntime.ResolveSymbol[mlmodel.Provider],
	})
	return &MLClient{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		runtime:    runtime,
		initErr:    err,
	}
}

func (c *MLClient) CompileProvider(name string) (string, error) {
	if c.initErr != nil {
		return "", fmt.Errorf("failed to initialise ML client: %w", c.initErr)
	}
	metadata, err := c.FetchPluginMetadata(name)
	if err != nil {
		return "", fmt.Errorf("failed to fetch plugin metadata: %w", err)
	}
	if _, _, err := c.runtime.CompileAndLoad(name, metadata.RepositoryURL, metadata.GitCommitHash, metadata.GitCommitHash); err != nil {
		return "", fmt.Errorf("failed to compile ML provider: %w", err)
	}
	return c.runtime.SoPath(name), nil
}

func (c *MLClient) LoadProvider(name string) (mlmodel.Provider, error) {
	if c.initErr != nil {
		return nil, fmt.Errorf("failed to initialise ML client: %w", c.initErr)
	}
	return c.runtime.LoadCached(name)
}

func (c *MLClient) FetchPluginMetadata(name string) (*models.Plugin, error) {
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
