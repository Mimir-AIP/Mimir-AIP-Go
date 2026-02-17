package plugins

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

// Client is a client for downloading plugins from the registry
type Client struct {
	baseURL    string
	httpClient *http.Client
	cacheDir   string
}

// NewClient creates a new plugin registry client
func NewClient(baseURL, cacheDir string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		cacheDir: cacheDir,
	}
}

// CompilePlugin compiles a plugin from source in the local environment
// This ensures the plugin is built with the exact same Go toolchain as the worker
func (c *Client) CompilePlugin(name string) (string, error) {
	// Ensure cache directory exists
	if err := os.MkdirAll(c.cacheDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Fetch plugin metadata to get repository URL and commit
	metadata, err := c.FetchPluginMetadata(name)
	if err != nil {
		return "", fmt.Errorf("failed to fetch plugin metadata: %w", err)
	}

	// Create cache file path
	cachePath := filepath.Join(c.cacheDir, fmt.Sprintf("%s.so", name))

	// Check if we already have this version cached
	if c.isCached(cachePath, metadata.Version, metadata.GitCommitHash, metadata.UpdatedAt.Format(time.RFC3339)) {
		fmt.Printf("Plugin %s already cached\n", name)
		return cachePath, nil
	}

	// Build plugin INSIDE /app/plugins/<name> to use the same module context
	pluginDir := filepath.Join("/app/plugins", name)
	if err := os.RemoveAll(pluginDir); err != nil {
		return "", fmt.Errorf("failed to clean plugin directory: %w", err)
	}
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create plugin directory: %w", err)
	}
	defer os.RemoveAll(pluginDir)

	fmt.Printf("Cloning plugin repository: %s\n", metadata.RepositoryURL)

	// Clone plugin repository directly into /app/plugins/<name>
	if err := c.cloneRepository(metadata.RepositoryURL, metadata.GitCommitHash, pluginDir); err != nil {
		return "", fmt.Errorf("failed to clone repository: %w", err)
	}

	fmt.Printf("Flattening action files\n")

	// Flatten action files (move from actions/ to root)
	if err := c.flattenActionFiles(pluginDir); err != nil {
		return "", fmt.Errorf("failed to flatten action files: %w", err)
	}

	fmt.Printf("Removing plugin go.mod (will use parent /app/go.mod)\n")

	// Remove the plugin's go.mod and go.sum - we'll use /app's module instead
	os.Remove(filepath.Join(pluginDir, "go.mod"))
	os.Remove(filepath.Join(pluginDir, "go.sum"))

	// Print checksum for debugging
	fmt.Printf("Plugin build info (using /app source):\n")
	cmd := exec.Command("sh", "-c", "md5sum /app/pkg/models/*.go | head -3")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()

	fmt.Printf("Compiling plugin binary (no go mod tidy needed - using /app module)\n")

	// Compile plugin directly - no go mod tidy needed since we're using /app's go.mod
	buildCmd := exec.Command("go", "build", "-buildmode=plugin", "-trimpath", "-o", cachePath, "./plugins/"+name)
	buildCmd.Dir = "/app"
	buildCmd.Env = append(os.Environ(), "CGO_ENABLED=1")

	if output, err := buildCmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("plugin compilation failed: %w\nOutput: %s", err, string(output))
	}

	fmt.Printf("Plugin compiled successfully\n")

	// Check build ID
	fmt.Printf("Plugin build ID:\n")
	buildIDCmd := exec.Command("go", "tool", "buildid", cachePath)
	buildIDCmd.Stdout = os.Stdout
	buildIDCmd.Stderr = os.Stderr
	buildIDCmd.Run()

	// Check module info
	fmt.Printf("Plugin module info (all):\n")
	versionCmd := exec.Command("go", "version", "-m", cachePath)
	versionCmd.Stdout = os.Stdout
	versionCmd.Stderr = os.Stderr
	versionCmd.Run()

	// Store version metadata for cache validation
	c.storeCacheMetadata(cachePath, metadata.Version, metadata.GitCommitHash, metadata.UpdatedAt.Format(time.RFC3339))

	return cachePath, nil
}

// cloneRepository clones a git repository at a specific commit
func (c *Client) cloneRepository(repoURL, commit, targetDir string) error {
	// Clone repository
	cloneCmd := exec.Command("git", "clone", repoURL, targetDir)
	if output, err := cloneCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git clone failed: %w\nOutput: %s", err, string(output))
	}

	// Checkout specific commit
	checkoutCmd := exec.Command("git", "checkout", commit)
	checkoutCmd.Dir = targetDir
	if output, err := checkoutCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git checkout failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// flattenActionFiles moves action files from actions/ subdirectory to root
func (c *Client) flattenActionFiles(dir string) error {
	actionsDir := filepath.Join(dir, "actions")

	// Check if actions directory exists
	if _, err := os.Stat(actionsDir); os.IsNotExist(err) {
		return nil // No actions directory, nothing to flatten
	}

	// Read all files in actions directory
	entries, err := os.ReadDir(actionsDir)
	if err != nil {
		return fmt.Errorf("failed to read actions directory: %w", err)
	}

	// Move each .go file to root
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}

		src := filepath.Join(actionsDir, entry.Name())
		dst := filepath.Join(dir, entry.Name())

		if err := os.Rename(src, dst); err != nil {
			return fmt.Errorf("failed to move %s: %w", entry.Name(), err)
		}
	}

	return nil
}

// isCached checks if a plugin is already cached with the same version
func (c *Client) isCached(path, version, commit, updated string) bool {
	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}

	// Check metadata file
	metaPath := path + ".meta"
	data, err := os.ReadFile(metaPath)
	if err != nil {
		return false
	}

	// Format: "version:commit:updated"
	expected := fmt.Sprintf("%s:%s:%s", version, commit, updated)
	return string(data) == expected
}

// storeCacheMetadata stores version info for cache validation
func (c *Client) storeCacheMetadata(path, version, commit, updated string) {
	metaPath := path + ".meta"
	data := fmt.Sprintf("%s:%s:%s", version, commit, updated)
	os.WriteFile(metaPath, []byte(data), 0644)
}

// LoadPluginFromCache loads a plugin from the local cache
func (c *Client) LoadPluginFromCache(name string) (string, error) {
	cachePath := filepath.Join(c.cacheDir, fmt.Sprintf("%s.so", name))
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		return "", fmt.Errorf("plugin %s not in cache", name)
	}
	return cachePath, nil
}

// FetchPluginMetadata fetches plugin metadata from the registry
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
