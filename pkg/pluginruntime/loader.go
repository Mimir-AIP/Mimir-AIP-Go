package pluginruntime

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"plugin"
	"reflect"
	"strings"
)

// SymbolResolver converts a looked-up plugin symbol into the requested runtime type.
type SymbolResolver[T any] func(symbol any) (T, bool)

// BuildSpec describes how a runtime extension should be cloned, compiled, cached,
// and opened inside the host module.
type BuildSpec[T any] struct {
	LogPrefix      string
	AppDir         string
	CacheDir       string
	TempDir        string
	HostPackageDir string
	ClonePrefix    string
	SymbolName     string
	DefaultGitRef  string
	Resolver       SymbolResolver[T]
}

// Loader compiles and loads runtime extensions using a shared clone/build/cache/open flow.
type Loader[T any] struct {
	spec BuildSpec[T]
}

// NewLoader validates the build spec and prepares the cache/temp directories.
func NewLoader[T any](spec BuildSpec[T]) (*Loader[T], error) {
	if spec.LogPrefix == "" {
		spec.LogPrefix = "plugin runtime"
	}
	if spec.SymbolName == "" {
		spec.SymbolName = "Plugin"
	}
	if spec.DefaultGitRef == "" {
		spec.DefaultGitRef = "main"
	}
	if spec.AppDir == "" {
		return nil, fmt.Errorf("%s: app directory is required", spec.LogPrefix)
	}
	if spec.CacheDir == "" {
		return nil, fmt.Errorf("%s: cache directory is required", spec.LogPrefix)
	}
	if spec.TempDir == "" {
		return nil, fmt.Errorf("%s: temp directory is required", spec.LogPrefix)
	}
	if spec.HostPackageDir == "" {
		return nil, fmt.Errorf("%s: host package directory is required", spec.LogPrefix)
	}
	if spec.Resolver == nil {
		return nil, fmt.Errorf("%s: symbol resolver is required", spec.LogPrefix)
	}
	for _, dir := range []string{spec.CacheDir, spec.TempDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("%s: failed to create directory %s: %w", spec.LogPrefix, dir, err)
		}
	}
	return &Loader[T]{spec: spec}, nil
}

// CompileAndLoad clones repoURL, compiles the extension, caches the .so by commit, and loads it.
func (l *Loader[T]) CompileAndLoad(name, repoURL, gitRef, commitHash string) (T, string, error) {
	var zero T
	if strings.TrimSpace(name) == "" {
		return zero, "", fmt.Errorf("%s: extension name is required", l.spec.LogPrefix)
	}
	if strings.TrimSpace(repoURL) == "" {
		return zero, "", fmt.Errorf("%s: repository URL is required", l.spec.LogPrefix)
	}
	if gitRef == "" {
		gitRef = l.spec.DefaultGitRef
	}
	if commitHash != "" && l.isCached(name, commitHash) {
		loaded, err := l.LoadCached(name)
		return loaded, commitHash, err
	}

	cloneDir := filepath.Join(l.spec.TempDir, l.cloneDirName(name))
	_ = os.RemoveAll(cloneDir)
	if err := l.cloneRepo(repoURL, gitRef, commitHash, cloneDir); err != nil {
		return zero, "", err
	}
	defer os.RemoveAll(cloneDir)

	resolvedHash, err := l.resolveCommitHash(cloneDir)
	if err != nil {
		log.Printf("%s: warning — could not resolve commit hash for %s: %v", l.spec.LogPrefix, name, err)
		if commitHash != "" {
			resolvedHash = commitHash
		} else {
			resolvedHash = gitRef
		}
	}
	if l.isCached(name, resolvedHash) {
		loaded, err := l.LoadCached(name)
		return loaded, resolvedHash, err
	}

	if err := flattenActions(cloneDir); err != nil {
		return zero, "", fmt.Errorf("%s: flatten actions: %w", l.spec.LogPrefix, err)
	}
	_ = os.Remove(filepath.Join(cloneDir, "go.mod"))
	_ = os.Remove(filepath.Join(cloneDir, "go.sum"))

	hostPackagePath := filepath.Join(l.spec.HostPackageDir, name)
	hostSourceDir := filepath.Join(l.spec.AppDir, hostPackagePath)
	_ = os.RemoveAll(hostSourceDir)
	if err := os.MkdirAll(filepath.Dir(hostSourceDir), 0755); err != nil {
		return zero, "", fmt.Errorf("%s: mkdir: %w", l.spec.LogPrefix, err)
	}
	if err := os.Rename(cloneDir, hostSourceDir); err != nil {
		return zero, "", fmt.Errorf("%s: move source: %w", l.spec.LogPrefix, err)
	}
	defer os.RemoveAll(hostSourceDir)

	buildCmd := exec.Command("go", "build", "-buildmode=plugin", "-trimpath", "-o", l.SoPath(name), "./"+filepath.ToSlash(hostPackagePath))
	buildCmd.Dir = l.spec.AppDir
	buildCmd.Env = append(os.Environ(), "CGO_ENABLED=1")
	if out, err := buildCmd.CombinedOutput(); err != nil {
		return zero, "", fmt.Errorf("%s: compilation failed for %s:\n%s\n%w", l.spec.LogPrefix, name, string(out), err)
	}

	l.writeMeta(name, resolvedHash)
	loaded, err := l.LoadCached(name)
	return loaded, resolvedHash, err
}

// LoadCached loads a previously compiled extension from the shared cache.
func (l *Loader[T]) LoadCached(name string) (T, error) {
	var zero T
	if _, err := os.Stat(l.SoPath(name)); os.IsNotExist(err) {
		return zero, fmt.Errorf("%s: no cached .so for %s", l.spec.LogPrefix, name)
	}
	plug, err := plugin.Open(l.SoPath(name))
	if err != nil {
		return zero, fmt.Errorf("%s: failed to open .so for %s: %w", l.spec.LogPrefix, name, err)
	}
	symbol, err := plug.Lookup(l.spec.SymbolName)
	if err != nil {
		return zero, fmt.Errorf("%s: %s does not export %q symbol: %w", l.spec.LogPrefix, name, l.spec.SymbolName, err)
	}
	resolved, ok := l.spec.Resolver(symbol)
	if !ok {
		return zero, fmt.Errorf("%s: symbol %q in %s has an incompatible type", l.spec.LogPrefix, l.spec.SymbolName, name)
	}
	return resolved, nil
}

// SoPath returns the shared cache location for the compiled .so.
func (l *Loader[T]) SoPath(name string) string {
	return filepath.Join(l.spec.CacheDir, name+".so")
}

// MetaPath returns the cache metadata location for the compiled .so.
func (l *Loader[T]) MetaPath(name string) string {
	return l.SoPath(name) + ".meta"
}

func (l *Loader[T]) cloneDirName(name string) string {
	prefix := strings.TrimSpace(l.spec.ClonePrefix)
	if prefix == "" {
		prefix = "runtime"
	}
	return prefix + "-" + name + "-clone"
}

func (l *Loader[T]) isCached(name, commitHash string) bool {
	if _, err := os.Stat(l.SoPath(name)); os.IsNotExist(err) {
		return false
	}
	data, err := os.ReadFile(l.MetaPath(name))
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(data)) == commitHash
}

func (l *Loader[T]) writeMeta(name, commitHash string) {
	if err := os.WriteFile(l.MetaPath(name), []byte(commitHash), 0644); err != nil {
		log.Printf("%s: warning — failed to write cache metadata for %s: %v", l.spec.LogPrefix, name, err)
	}
}

func (l *Loader[T]) cloneRepo(repoURL, gitRef, commitHash, dest string) error {
	cloneArgs := []string{"clone"}
	if commitHash == "" && gitRef != "" {
		cloneArgs = append(cloneArgs, "--depth=1", "--branch", gitRef)
	}
	cloneArgs = append(cloneArgs, repoURL, dest)
	cloneCmd := exec.Command("git", cloneArgs...)
	if out, err := cloneCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%s: git clone failed: %w\n%s", l.spec.LogPrefix, err, string(out))
	}

	checkoutTarget := commitHash
	if checkoutTarget == "" {
		return nil
	}
	checkoutCmd := exec.Command("git", "checkout", checkoutTarget)
	checkoutCmd.Dir = dest
	if out, err := checkoutCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%s: git checkout %s failed: %w\n%s", l.spec.LogPrefix, checkoutTarget, err, string(out))
	}
	return nil
}

func (l *Loader[T]) resolveCommitHash(dir string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func flattenActions(dir string) error {
	actionsDir := filepath.Join(dir, "actions")
	if _, err := os.Stat(actionsDir); os.IsNotExist(err) {
		return nil
	}
	entries, err := os.ReadDir(actionsDir)
	if err != nil {
		return fmt.Errorf("read actions dir: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		if err := os.Rename(filepath.Join(actionsDir, entry.Name()), filepath.Join(dir, entry.Name())); err != nil {
			return fmt.Errorf("move %s: %w", entry.Name(), err)
		}
	}
	return nil
}

// ResolveSymbol accepts either a direct implementation or a pointer to one.
func ResolveSymbol[T any](symbol any) (T, bool) {
	var zero T
	if typed, ok := symbol.(T); ok {
		return typed, true
	}
	value := reflect.ValueOf(symbol)
	if !value.IsValid() || value.Kind() != reflect.Ptr || value.IsNil() {
		return zero, false
	}
	inner := value.Elem()
	if !inner.IsValid() {
		return zero, false
	}
	if typed, ok := inner.Interface().(T); ok {
		return typed, true
	}
	return zero, false
}
