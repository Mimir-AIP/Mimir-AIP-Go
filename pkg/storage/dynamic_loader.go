package storage

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"plugin"
	"strings"

	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

// PluginLoader compiles and loads external storage plugins at runtime.
// It mirrors the approach used by the worker for pipeline plugins:
//  1. Clone the Git repository.
//  2. Flatten the optional actions/ subdirectory to the root.
//  3. Remove the plugin's go.mod/go.sum (the host module is used instead).
//  4. Compile with go build -buildmode=plugin using the host module at appDir.
//  5. Open the resulting .so and resolve the "Plugin" symbol.
//
// Compiled .so files are cached at cacheDir so that restarts do not require
// recompilation (unless the plugin was updated).
type PluginLoader struct {
	appDir   string // root of the mimir-aip-go module (e.g. /app)
	cacheDir string // where .so files are written (e.g. /app/storage-plugins)
	tempDir  string // scratch space for Git clones
}

// NewPluginLoader creates a PluginLoader.
//   - appDir:   the directory that contains the host go.mod (usually /app).
//   - cacheDir: where compiled .so files are cached between runs.
//   - tempDir:  temporary directory for git clones (cleaned up after each compile).
func NewPluginLoader(appDir, cacheDir, tempDir string) (*PluginLoader, error) {
	for _, d := range []string{cacheDir, tempDir} {
		if err := os.MkdirAll(d, 0755); err != nil {
			return nil, fmt.Errorf("storage plugin loader: failed to create directory %s: %w", d, err)
		}
	}
	return &PluginLoader{appDir: appDir, cacheDir: cacheDir, tempDir: tempDir}, nil
}

// soPath returns the expected .so path for a plugin name.
func (l *PluginLoader) soPath(name string) string {
	return filepath.Join(l.cacheDir, name+".so")
}

// metaPath returns the cache metadata file path for a plugin.
func (l *PluginLoader) metaPath(name string) string {
	return l.soPath(name) + ".meta"
}

// isCached reports whether a compiled .so exists and was built from commitHash.
func (l *PluginLoader) isCached(name, commitHash string) bool {
	if _, err := os.Stat(l.soPath(name)); os.IsNotExist(err) {
		return false
	}
	data, err := os.ReadFile(l.metaPath(name))
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(data)) == commitHash
}

// writeMeta stores the commit hash alongside the .so for cache validation.
func (l *PluginLoader) writeMeta(name, commitHash string) {
	if err := os.WriteFile(l.metaPath(name), []byte(commitHash), 0644); err != nil {
		log.Printf("storage plugin loader: warning — failed to write cache metadata for %s: %v", name, err)
	}
}

// CompileAndLoad clones repoURL at commitHash (or gitRef if commitHash is
// empty), compiles the plugin, and returns a loaded StoragePlugin.
// If a valid cached .so already exists for commitHash it is used directly.
func (l *PluginLoader) CompileAndLoad(name, repoURL, gitRef, commitHash string) (models.StoragePlugin, string, error) {
	if gitRef == "" {
		gitRef = "main"
	}

	// Use cached .so if the commit hasn't changed.
	if commitHash != "" && l.isCached(name, commitHash) {
		log.Printf("storage plugin loader: using cached .so for %s @ %s", name, commitHash[:8])
		sp, err := l.openSO(name)
		return sp, commitHash, err
	}

	// Clone into a fresh temp directory.
	cloneDir := filepath.Join(l.tempDir, "sp-"+name+"-clone")
	_ = os.RemoveAll(cloneDir)
	if err := l.cloneRepo(repoURL, gitRef, cloneDir); err != nil {
		return nil, "", err
	}
	defer os.RemoveAll(cloneDir)

	// Resolve the actual commit hash so we can cache by it.
	resolvedHash, err := l.commitHash(cloneDir)
	if err != nil {
		log.Printf("storage plugin loader: warning — could not resolve commit hash for %s: %v", name, err)
		resolvedHash = gitRef
	}

	// If the resolved hash is already cached we can skip recompilation.
	if l.isCached(name, resolvedHash) {
		log.Printf("storage plugin loader: using cached .so for %s @ %s", name, resolvedHash[:min(8, len(resolvedHash))])
		sp, err := l.openSO(name)
		return sp, resolvedHash, err
	}

	// Flatten actions/ subdirectory (same behaviour as the pipeline plugin loader).
	if err := flattenActions(cloneDir); err != nil {
		return nil, "", fmt.Errorf("storage plugin loader: flatten actions: %w", err)
	}

	// Remove the plugin's own go.mod — we compile inside the host module.
	_ = os.Remove(filepath.Join(cloneDir, "go.mod"))
	_ = os.Remove(filepath.Join(cloneDir, "go.sum"))

	// Place plugin source inside the host module so it can be built as a
	// package path relative to appDir.
	pluginPkg := filepath.Join("storage-plugins", name)
	pluginSrcDir := filepath.Join(l.appDir, pluginPkg)
	_ = os.RemoveAll(pluginSrcDir)
	if err := os.MkdirAll(filepath.Dir(pluginSrcDir), 0755); err != nil {
		return nil, "", fmt.Errorf("storage plugin loader: mkdir: %w", err)
	}
	if err := os.Rename(cloneDir, pluginSrcDir); err != nil {
		return nil, "", fmt.Errorf("storage plugin loader: move plugin source: %w", err)
	}
	defer os.RemoveAll(pluginSrcDir)

	// Compile.
	soOut := l.soPath(name)
	buildCmd := exec.Command("go", "build",
		"-buildmode=plugin",
		"-trimpath",
		"-o", soOut,
		"./"+pluginPkg,
	)
	buildCmd.Dir = l.appDir
	buildCmd.Env = append(os.Environ(), "CGO_ENABLED=1")
	if out, err := buildCmd.CombinedOutput(); err != nil {
		return nil, "", fmt.Errorf("storage plugin loader: compilation failed for %s:\n%s\n%w", name, string(out), err)
	}
	log.Printf("storage plugin loader: compiled %s successfully", name)

	l.writeMeta(name, resolvedHash)

	sp, err := l.openSO(name)
	return sp, resolvedHash, err
}

// LoadCached opens a previously compiled .so without recompiling.
// Returns an error if the .so does not exist.
func (l *PluginLoader) LoadCached(name string) (models.StoragePlugin, error) {
	if _, err := os.Stat(l.soPath(name)); os.IsNotExist(err) {
		return nil, fmt.Errorf("storage plugin loader: no cached .so for %s — plugin must be reinstalled", name)
	}
	return l.openSO(name)
}

// openSO opens the .so at soPath(name) and resolves the "Plugin" symbol.
func (l *PluginLoader) openSO(name string) (models.StoragePlugin, error) {
	p, err := plugin.Open(l.soPath(name))
	if err != nil {
		return nil, fmt.Errorf("storage plugin loader: failed to open .so for %s: %w", name, err)
	}
	sym, err := p.Lookup("Plugin")
	if err != nil {
		return nil, fmt.Errorf("storage plugin loader: %s does not export 'Plugin' symbol: %w", name, err)
	}
	// plugin.Lookup returns a pointer to the package-level variable.
	// If the plugin declares "var Plugin MyPlugin" with pointer-receiver methods,
	// sym is *MyPlugin which satisfies models.StoragePlugin directly.
	sp, ok := sym.(models.StoragePlugin)
	if !ok {
		return nil, fmt.Errorf("storage plugin loader: Plugin symbol in %s does not implement models.StoragePlugin", name)
	}
	return sp, nil
}

// cloneRepo runs git clone --depth=1 --branch <ref> <url> <dest>.
func (l *PluginLoader) cloneRepo(repoURL, gitRef, dest string) error {
	cmd := exec.Command("git", "clone", "--depth=1", "--branch", gitRef, repoURL, dest)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("storage plugin loader: git clone failed: %w\n%s", err, string(out))
	}
	return nil
}

// commitHash returns the HEAD SHA of a local git repository.
func (l *PluginLoader) commitHash(dir string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// flattenActions moves .go files from actions/ to the plugin root, matching
// the same behaviour as pkg/plugins/client.go for pipeline plugins.
func flattenActions(dir string) error {
	actionsDir := filepath.Join(dir, "actions")
	if _, err := os.Stat(actionsDir); os.IsNotExist(err) {
		return nil
	}
	entries, err := os.ReadDir(actionsDir)
	if err != nil {
		return fmt.Errorf("read actions dir: %w", err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		if err := os.Rename(
			filepath.Join(actionsDir, e.Name()),
			filepath.Join(dir, e.Name()),
		); err != nil {
			return fmt.Errorf("move %s: %w", e.Name(), err)
		}
	}
	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
