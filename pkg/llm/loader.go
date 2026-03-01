package llm

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"plugin"
	"strings"
)

// Loader compiles and loads external LLM providers at runtime.
// It mirrors the approach used by storage.PluginLoader:
//  1. Clone the Git repository.
//  2. Flatten the optional actions/ subdirectory to the root.
//  3. Remove the provider's go.mod/go.sum (the host module is used instead).
//  4. Compile with go build -buildmode=plugin using the host module at appDir.
//  5. Open the resulting .so and resolve the "Plugin" symbol (a Provider).
//
// Compiled .so files are cached at cacheDir so restarts do not require
// recompilation (unless the provider was updated).
type Loader struct {
	appDir   string // root of the mimir-aip-go module (e.g. /app)
	cacheDir string // where .so files are written (e.g. /app/llm-providers)
	tempDir  string // scratch space for Git clones
}

// NewLoader creates a Loader.
//   - appDir:   the directory that contains the host go.mod (usually /app).
//   - cacheDir: where compiled .so files are cached between runs.
//   - tempDir:  temporary directory for git clones (cleaned up after each compile).
func NewLoader(appDir, cacheDir, tempDir string) (*Loader, error) {
	for _, d := range []string{cacheDir, tempDir} {
		if err := os.MkdirAll(d, 0755); err != nil {
			return nil, fmt.Errorf("llm loader: failed to create directory %s: %w", d, err)
		}
	}
	return &Loader{appDir: appDir, cacheDir: cacheDir, tempDir: tempDir}, nil
}

// soPath returns the expected .so path for a provider name.
func (l *Loader) soPath(name string) string {
	return filepath.Join(l.cacheDir, name+".so")
}

// metaPath returns the cache metadata file path for a provider.
func (l *Loader) metaPath(name string) string {
	return l.soPath(name) + ".meta"
}

// isCached reports whether a compiled .so exists and was built from commitHash.
func (l *Loader) isCached(name, commitHash string) bool {
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
func (l *Loader) writeMeta(name, commitHash string) {
	if err := os.WriteFile(l.metaPath(name), []byte(commitHash), 0644); err != nil {
		log.Printf("llm loader: warning — failed to write cache metadata for %s: %v", name, err)
	}
}

// CompileAndLoad clones repoURL at commitHash (or gitRef if commitHash is
// empty), compiles the provider, and returns a loaded Provider.
// If a valid cached .so already exists for commitHash it is used directly.
func (l *Loader) CompileAndLoad(name, repoURL, gitRef, commitHash string) (Provider, string, error) {
	if gitRef == "" {
		gitRef = "main"
	}

	// Use cached .so if the commit hasn't changed.
	if commitHash != "" && l.isCached(name, commitHash) {
		log.Printf("llm loader: using cached .so for %s @ %s", name, commitHash[:8])
		p, err := l.openSO(name)
		return p, commitHash, err
	}

	// Clone into a fresh temp directory.
	cloneDir := filepath.Join(l.tempDir, "llmp-"+name+"-clone")
	_ = os.RemoveAll(cloneDir)
	if err := l.cloneRepo(repoURL, gitRef, cloneDir); err != nil {
		return nil, "", err
	}
	defer os.RemoveAll(cloneDir)

	// Resolve the actual commit hash so we can cache by it.
	resolvedHash, err := l.resolveCommitHash(cloneDir)
	if err != nil {
		log.Printf("llm loader: warning — could not resolve commit hash for %s: %v", name, err)
		resolvedHash = gitRef
	}

	// If the resolved hash is already cached we can skip recompilation.
	if l.isCached(name, resolvedHash) {
		log.Printf("llm loader: using cached .so for %s @ %s", name, resolvedHash[:minLen(8, len(resolvedHash))])
		p, err := l.openSO(name)
		return p, resolvedHash, err
	}

	// Flatten actions/ subdirectory (same behaviour as the pipeline plugin loader).
	if err := flattenLLMActions(cloneDir); err != nil {
		return nil, "", fmt.Errorf("llm loader: flatten actions: %w", err)
	}

	// Remove the provider's own go.mod — we compile inside the host module.
	_ = os.Remove(filepath.Join(cloneDir, "go.mod"))
	_ = os.Remove(filepath.Join(cloneDir, "go.sum"))

	// Place provider source inside the host module so it can be built as a
	// package path relative to appDir.
	providerPkg := filepath.Join("llm-providers", name)
	providerSrcDir := filepath.Join(l.appDir, providerPkg)
	_ = os.RemoveAll(providerSrcDir)
	if err := os.MkdirAll(filepath.Dir(providerSrcDir), 0755); err != nil {
		return nil, "", fmt.Errorf("llm loader: mkdir: %w", err)
	}
	if err := os.Rename(cloneDir, providerSrcDir); err != nil {
		return nil, "", fmt.Errorf("llm loader: move provider source: %w", err)
	}
	defer os.RemoveAll(providerSrcDir)

	// Compile.
	soOut := l.soPath(name)
	buildCmd := exec.Command("go", "build",
		"-buildmode=plugin",
		"-trimpath",
		"-o", soOut,
		"./"+providerPkg,
	)
	buildCmd.Dir = l.appDir
	buildCmd.Env = append(os.Environ(), "CGO_ENABLED=1")
	if out, err := buildCmd.CombinedOutput(); err != nil {
		return nil, "", fmt.Errorf("llm loader: compilation failed for %s:\n%s\n%w", name, string(out), err)
	}
	log.Printf("llm loader: compiled %s successfully", name)

	l.writeMeta(name, resolvedHash)

	p, err := l.openSO(name)
	return p, resolvedHash, err
}

// LoadCached opens a previously compiled .so without recompiling.
// Returns an error if the .so does not exist.
func (l *Loader) LoadCached(name string) (Provider, error) {
	if _, err := os.Stat(l.soPath(name)); os.IsNotExist(err) {
		return nil, fmt.Errorf("llm loader: no cached .so for %s — provider must be reinstalled", name)
	}
	return l.openSO(name)
}

// openSO opens the .so at soPath(name) and resolves the "Plugin" symbol.
func (l *Loader) openSO(name string) (Provider, error) {
	p, err := plugin.Open(l.soPath(name))
	if err != nil {
		return nil, fmt.Errorf("llm loader: failed to open .so for %s: %w", name, err)
	}
	sym, err := p.Lookup("Plugin")
	if err != nil {
		return nil, fmt.Errorf("llm loader: %s does not export 'Plugin' symbol: %w", name, err)
	}
	// plugin.Lookup returns a pointer to the package-level variable.
	provider, ok := sym.(Provider)
	if !ok {
		return nil, fmt.Errorf("llm loader: Plugin symbol in %s does not implement llm.Provider", name)
	}
	return provider, nil
}

// cloneRepo runs git clone --depth=1 --branch <ref> <url> <dest>.
func (l *Loader) cloneRepo(repoURL, gitRef, dest string) error {
	cmd := exec.Command("git", "clone", "--depth=1", "--branch", gitRef, repoURL, dest)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("llm loader: git clone failed: %w\n%s", err, string(out))
	}
	return nil
}

// resolveCommitHash returns the HEAD SHA of a local git repository.
func (l *Loader) resolveCommitHash(dir string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// flattenLLMActions moves .go files from actions/ to the provider root.
func flattenLLMActions(dir string) error {
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

// repoName derives a provider name from a Git URL — last path segment without ".git".
func repoName(repoURL string) string {
	parts := strings.Split(strings.TrimRight(repoURL, "/"), "/")
	name := parts[len(parts)-1]
	return strings.TrimSuffix(name, ".git")
}

func minLen(a, b int) int {
	if a < b {
		return a
	}
	return b
}
