package plugins

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"github.com/mimir-aip/mimir-aip-go/pkg/pipeline"
	"github.com/mimir-aip/mimir-aip-go/pkg/pluginruntime"
)

// Client fetches verified pipeline plugin artifacts from the orchestrator.
type Client struct {
	baseURL    string
	cacheDir   string
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
		baseURL:  baseURL,
		cacheDir: cacheDir,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		runtime: runtime,
		initErr: err,
	}
	return client
}

// CompilePlugin downloads and verifies the orchestrator-built plugin artifact.
func (c *Client) CompilePlugin(name string) (string, error) {
	if c.initErr != nil {
		return "", fmt.Errorf("failed to initialise plugin client: %w", c.initErr)
	}
	artifact, err := c.FetchActiveArtifact(models.PluginKindPipeline, name)
	if err != nil {
		return "", err
	}
	return c.downloadArtifact(artifact)
}

// LoadPlugin loads a downloaded plugin from the local cache.
func (c *Client) LoadPlugin(name string) (pipeline.Plugin, error) {
	if c.initErr != nil {
		return nil, fmt.Errorf("failed to initialise plugin client: %w", c.initErr)
	}
	artifact, err := c.FetchActiveArtifact(models.PluginKindPipeline, name)
	if err != nil {
		return nil, err
	}
	return c.runtime.LoadPath(name, c.cachedArtifactPath(artifact))
}

func (c *Client) FetchActiveArtifact(kind models.PluginKind, name string) (*models.PluginArtifact, error) {
	url := fmt.Sprintf("%s/api/plugin-artifacts?kind=%s&name=%s", c.baseURL, kind, name)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch plugin artifact metadata: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch plugin artifact metadata (status %d): %s", resp.StatusCode, string(body))
	}
	var artifact models.PluginArtifact
	if err := json.NewDecoder(resp.Body).Decode(&artifact); err != nil {
		return nil, fmt.Errorf("failed to decode plugin artifact metadata: %w", err)
	}
	return &artifact, nil
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

func (c *Client) downloadArtifact(artifact *models.PluginArtifact) (string, error) {
	if artifact == nil {
		return "", fmt.Errorf("plugin artifact is required")
	}
	path := c.cachedArtifactPath(artifact)
	if ok, _ := verifyFileDigest(path, artifact.Digest); ok {
		return path, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return "", fmt.Errorf("create plugin cache: %w", err)
	}
	url := fmt.Sprintf("%s/api/plugin-artifacts/%s/download", c.baseURL, artifact.ID)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to download plugin artifact: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to download plugin artifact (status %d): %s", resp.StatusCode, string(body))
	}
	tmp := path + ".tmp"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return "", fmt.Errorf("create plugin artifact cache file: %w", err)
	}
	_, copyErr := io.Copy(out, resp.Body)
	closeErr := out.Close()
	if copyErr != nil {
		_ = os.Remove(tmp)
		return "", fmt.Errorf("write plugin artifact cache file: %w", copyErr)
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return "", fmt.Errorf("close plugin artifact cache file: %w", closeErr)
	}
	if ok, err := verifyFileDigest(tmp, artifact.Digest); err != nil || !ok {
		_ = os.Remove(tmp)
		if err != nil {
			return "", err
		}
		return "", fmt.Errorf("plugin artifact digest mismatch for %s", artifact.ID)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return "", fmt.Errorf("publish plugin artifact cache file: %w", err)
	}
	return path, nil
}

func (c *Client) cachedArtifactPath(artifact *models.PluginArtifact) string {
	digest := strings.TrimPrefix(artifact.Digest, "sha256:")
	return filepath.Join(c.cacheDir, "artifacts", string(artifact.PluginKind), artifact.PluginName, digest+".so")
}

func verifyFileDigest(path, expected string) (bool, error) {
	file, err := os.Open(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("open plugin artifact for verification: %w", err)
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return false, fmt.Errorf("hash plugin artifact: %w", err)
	}
	actual := "sha256:" + hex.EncodeToString(hash.Sum(nil))
	return actual == expected, nil
}
