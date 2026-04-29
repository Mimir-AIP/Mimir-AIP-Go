package pluginruntime

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"plugin"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	defaultCommandTimeout = 2 * time.Minute
	maxCommandOutput      = 64 * 1024
)

var buildLocks sync.Map

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
	BuildTimeout   time.Duration
	Resolver       SymbolResolver[T]
}

// Loader compiles and loads runtime extensions using a shared clone/build/cache/open flow.
type Loader[T any] struct {
	spec BuildSpec[T]
}

type cacheMetadata struct {
	CommitHash     string `json:"commit_hash"`
	ArtifactPath   string `json:"artifact_path"`
	GoVersion      string `json:"go_version"`
	GOOS           string `json:"goos"`
	GOARCH         string `json:"goarch"`
	HostPackageDir string `json:"host_package_dir"`
	SymbolName     string `json:"symbol_name"`
	BuiltAt        string `json:"built_at"`
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
	if spec.BuildTimeout == 0 {
		spec.BuildTimeout = defaultCommandTimeout
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

// CompileAndLoad clones repoURL, compiles the extension, caches the .so by commit,
// and loads it. Builds for the same logical plugin are serialized across goroutines
// and cooperating processes to avoid cache and source directory races.
func (l *Loader[T]) CompileAndLoad(name, repoURL, gitRef, commitHash string) (T, string, error) {
	var zero T
	name = strings.TrimSpace(name)
	if name == "" {
		return zero, "", fmt.Errorf("%s: extension name is required", l.spec.LogPrefix)
	}
	if err := ValidateRepositoryURL(repoURL); err != nil {
		return zero, "", fmt.Errorf("%s: %w", l.spec.LogPrefix, err)
	}
	if gitRef == "" {
		gitRef = l.spec.DefaultGitRef
	}

	unlock, err := l.lock(name)
	if err != nil {
		return zero, "", err
	}
	defer unlock()

	if commitHash != "" && l.isCached(name, commitHash) {
		loaded, err := l.LoadCached(name)
		return loaded, commitHash, err
	}

	cloneDir := filepath.Join(l.spec.TempDir, l.uniqueDirName(name, "clone"))
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

	hostPackagePath := filepath.Join(l.spec.HostPackageDir, l.uniqueDirName(name, "src"))
	hostSourceDir := filepath.Join(l.spec.AppDir, hostPackagePath)
	if err := os.MkdirAll(filepath.Dir(hostSourceDir), 0755); err != nil {
		return zero, "", fmt.Errorf("%s: mkdir: %w", l.spec.LogPrefix, err)
	}
	if err := os.Rename(cloneDir, hostSourceDir); err != nil {
		return zero, "", fmt.Errorf("%s: move source: %w", l.spec.LogPrefix, err)
	}
	defer os.RemoveAll(hostSourceDir)

	artifactPath := l.artifactPath(name, resolvedHash)
	tmpArtifactPath := artifactPath + ".tmp." + l.uniqueSuffix()
	buildCmd := exec.Command("go", "build", "-buildmode=plugin", "-trimpath", "-o", tmpArtifactPath, "./"+filepath.ToSlash(hostPackagePath))
	buildCmd.Dir = l.spec.AppDir
	buildCmd.Env = append(os.Environ(), "CGO_ENABLED=1")
	if out, err := l.runCommand(buildCmd); err != nil {
		_ = os.Remove(tmpArtifactPath)
		return zero, "", fmt.Errorf("%s: compilation failed for %s:\n%s\n%w", l.spec.LogPrefix, name, out, err)
	}
	if err := os.Rename(tmpArtifactPath, artifactPath); err != nil {
		_ = os.Remove(tmpArtifactPath)
		return zero, "", fmt.Errorf("%s: publish compiled artifact for %s: %w", l.spec.LogPrefix, name, err)
	}

	if err := l.writeMeta(name, resolvedHash, artifactPath); err != nil {
		return zero, "", err
	}
	loaded, err := l.LoadCached(name)
	return loaded, resolvedHash, err
}

// LoadCached loads a previously compiled extension from the shared cache.
func (l *Loader[T]) LoadCached(name string) (T, error) {
	var zero T
	artifactPath := l.SoPath(name)
	if _, err := os.Stat(artifactPath); os.IsNotExist(err) {
		return zero, fmt.Errorf("%s: no cached .so for %s", l.spec.LogPrefix, name)
	}
	plug, err := plugin.Open(artifactPath)
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

// SoPath returns the cache location for the currently selected compiled .so.
func (l *Loader[T]) SoPath(name string) string {
	meta, err := l.readMeta(name)
	if err == nil && meta.ArtifactPath != "" {
		return meta.ArtifactPath
	}
	return filepath.Join(l.spec.CacheDir, l.safeName(name)+".so")
}

// MetaPath returns the cache metadata location for the compiled .so.
func (l *Loader[T]) MetaPath(name string) string {
	return filepath.Join(l.spec.CacheDir, l.safeName(name)+".so.meta")
}

func (l *Loader[T]) isCached(name, commitHash string) bool {
	meta, err := l.readMeta(name)
	if err != nil {
		return false
	}
	if meta.CommitHash != commitHash || meta.GoVersion != runtime.Version() || meta.GOOS != runtime.GOOS || meta.GOARCH != runtime.GOARCH || meta.HostPackageDir != l.spec.HostPackageDir || meta.SymbolName != l.spec.SymbolName {
		return false
	}
	if _, err := os.Stat(meta.ArtifactPath); err != nil {
		return false
	}
	return true
}

func (l *Loader[T]) writeMeta(name, commitHash, artifactPath string) error {
	meta := cacheMetadata{
		CommitHash:     commitHash,
		ArtifactPath:   artifactPath,
		GoVersion:      runtime.Version(),
		GOOS:           runtime.GOOS,
		GOARCH:         runtime.GOARCH,
		HostPackageDir: l.spec.HostPackageDir,
		SymbolName:     l.spec.SymbolName,
		BuiltAt:        time.Now().UTC().Format(time.RFC3339Nano),
	}
	data, err := json.Marshal(meta)
	if err != nil {
		return fmt.Errorf("%s: marshal cache metadata for %s: %w", l.spec.LogPrefix, name, err)
	}
	tmpMetaPath := l.MetaPath(name) + ".tmp." + l.uniqueSuffix()
	if err := os.WriteFile(tmpMetaPath, data, 0644); err != nil {
		return fmt.Errorf("%s: write cache metadata for %s: %w", l.spec.LogPrefix, name, err)
	}
	if err := os.Rename(tmpMetaPath, l.MetaPath(name)); err != nil {
		_ = os.Remove(tmpMetaPath)
		return fmt.Errorf("%s: publish cache metadata for %s: %w", l.spec.LogPrefix, name, err)
	}
	return nil
}

func (l *Loader[T]) readMeta(name string) (cacheMetadata, error) {
	data, err := os.ReadFile(l.MetaPath(name))
	if err != nil {
		return cacheMetadata{}, err
	}
	var meta cacheMetadata
	if err := json.Unmarshal(data, &meta); err != nil {
		legacyCommit := strings.TrimSpace(string(data))
		if legacyCommit == "" {
			return cacheMetadata{}, err
		}
		return cacheMetadata{CommitHash: legacyCommit, ArtifactPath: filepath.Join(l.spec.CacheDir, l.safeName(name)+".so")}, nil
	}
	return meta, nil
}

func (l *Loader[T]) cloneRepo(repoURL, gitRef, commitHash, dest string) error {
	cloneArgs := []string{"clone"}
	if commitHash == "" && gitRef != "" {
		cloneArgs = append(cloneArgs, "--depth=1", "--branch", gitRef)
	}
	cloneArgs = append(cloneArgs, repoURL, dest)
	cloneCmd := exec.Command("git", cloneArgs...)
	if out, err := l.runCommand(cloneCmd); err != nil {
		return fmt.Errorf("%s: git clone failed: %w\n%s", l.spec.LogPrefix, err, out)
	}

	checkoutTarget := commitHash
	if checkoutTarget == "" {
		return nil
	}
	checkoutCmd := exec.Command("git", "checkout", checkoutTarget)
	checkoutCmd.Dir = dest
	if out, err := l.runCommand(checkoutCmd); err != nil {
		return fmt.Errorf("%s: git checkout %s failed: %w\n%s", l.spec.LogPrefix, checkoutTarget, err, out)
	}
	return nil
}

func (l *Loader[T]) resolveCommitHash(dir string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = dir
	out, err := l.runCommand(cmd)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

func (l *Loader[T]) runCommand(cmd *exec.Cmd) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), l.spec.BuildTimeout)
	defer cancel()
	ctxCmd := exec.CommandContext(ctx, cmd.Path, cmd.Args[1:]...)
	ctxCmd.Dir = cmd.Dir
	ctxCmd.Env = cmd.Env
	out, err := ctxCmd.CombinedOutput()
	output := string(out)
	if len(output) > maxCommandOutput {
		output = output[:maxCommandOutput] + "\n... output truncated ..."
	}
	if ctx.Err() == context.DeadlineExceeded {
		return output, fmt.Errorf("command timed out after %s", l.spec.BuildTimeout)
	}
	return output, err
}

func (l *Loader[T]) lock(name string) (func(), error) {
	safe := l.safeName(name)
	lockValue, _ := buildLocks.LoadOrStore(safe, &sync.Mutex{})
	mutex := lockValue.(*sync.Mutex)
	mutex.Lock()

	lockPath := filepath.Join(l.spec.CacheDir, safe+".lock")
	file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		mutex.Unlock()
		return nil, fmt.Errorf("%s: open build lock for %s: %w", l.spec.LogPrefix, name, err)
	}
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX); err != nil {
		_ = file.Close()
		mutex.Unlock()
		return nil, fmt.Errorf("%s: acquire build lock for %s: %w", l.spec.LogPrefix, name, err)
	}
	return func() {
		_ = syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
		_ = file.Close()
		mutex.Unlock()
	}, nil
}

func (l *Loader[T]) artifactPath(name, commitHash string) string {
	fingerprint := sha256.Sum256([]byte(strings.Join([]string{
		l.safeName(name),
		commitHash,
		runtime.Version(),
		runtime.GOOS,
		runtime.GOARCH,
		l.spec.HostPackageDir,
		l.spec.SymbolName,
	}, "\x00")))
	return filepath.Join(l.spec.CacheDir, l.safeName(name)+"-"+hex.EncodeToString(fingerprint[:])[:16]+".so")
}

func (l *Loader[T]) safeName(name string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "plugin"
	}
	var b strings.Builder
	for _, r := range trimmed {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			b.WriteRune(r)
		} else {
			b.WriteRune('_')
		}
	}
	return b.String()
}

func (l *Loader[T]) uniqueDirName(name, purpose string) string {
	return strings.Join([]string{l.safeName(l.spec.ClonePrefix), l.safeName(name), purpose, l.uniqueSuffix()}, "-")
}

func (l *Loader[T]) uniqueSuffix() string {
	return fmt.Sprintf("%d-%d", os.Getpid(), time.Now().UnixNano())
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
