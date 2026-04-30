package plugins

import (
	"fmt"
	"net/http"
	"path/filepath"
	"time"

	"github.com/mimir-aip/mimir-aip-go/pkg/mlmodel"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"github.com/mimir-aip/mimir-aip-go/pkg/pluginruntime"
)

// MLClient fetches verified ML provider artifacts from the orchestrator.
type MLClient struct {
	baseURL    string
	cacheDir   string
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
		cacheDir:   cacheDir,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		runtime:    runtime,
		initErr:    err,
	}
}

func (c *MLClient) CompileProvider(name string) (string, error) {
	if c.initErr != nil {
		return "", fmt.Errorf("failed to initialise ML client: %w", c.initErr)
	}
	artifactClient := &Client{baseURL: c.baseURL, cacheDir: c.cacheDir, httpClient: c.httpClient}
	artifact, err := artifactClient.FetchActiveArtifact(models.PluginKindMLProvider, name)
	if err != nil {
		return "", err
	}
	return artifactClient.downloadArtifact(artifact)
}

func (c *MLClient) LoadProvider(name string) (mlmodel.Provider, error) {
	if c.initErr != nil {
		return nil, fmt.Errorf("failed to initialise ML client: %w", c.initErr)
	}
	artifactClient := &Client{baseURL: c.baseURL, cacheDir: c.cacheDir, httpClient: c.httpClient}
	artifact, err := artifactClient.FetchActiveArtifact(models.PluginKindMLProvider, name)
	if err != nil {
		return nil, err
	}
	return c.runtime.LoadPath(name, artifactClient.cachedArtifactPath(artifact))
}
