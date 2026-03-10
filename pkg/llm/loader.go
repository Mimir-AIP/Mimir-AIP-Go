package llm

import (
	"strings"

	"github.com/mimir-aip/mimir-aip-go/pkg/pluginruntime"
)

// Loader compiles and loads external LLM providers at runtime using the shared
// pluginruntime clone/build/cache/open flow.
type Loader struct {
	runtime *pluginruntime.Loader[Provider]
}

// NewLoader creates a Loader.
func NewLoader(appDir, cacheDir, tempDir string) (*Loader, error) {
	runtime, err := pluginruntime.NewLoader(pluginruntime.BuildSpec[Provider]{
		LogPrefix:      "llm loader",
		AppDir:         appDir,
		CacheDir:       cacheDir,
		TempDir:        tempDir,
		HostPackageDir: "llm-providers",
		ClonePrefix:    "llmp",
		SymbolName:     "Plugin",
		DefaultGitRef:  "main",
		Resolver:       pluginruntime.ResolveSymbol[Provider],
	})
	if err != nil {
		return nil, err
	}
	return &Loader{runtime: runtime}, nil
}

// CompileAndLoad clones, compiles, caches, and loads the provider.
func (l *Loader) CompileAndLoad(name, repoURL, gitRef, commitHash string) (Provider, string, error) {
	return l.runtime.CompileAndLoad(name, repoURL, gitRef, commitHash)
}

// LoadCached opens a previously compiled .so without recompiling.
func (l *Loader) LoadCached(name string) (Provider, error) {
	return l.runtime.LoadCached(name)
}

func (l *Loader) soPath(name string) string {
	return l.runtime.SoPath(name)
}

func (l *Loader) metaPath(name string) string {
	return l.runtime.MetaPath(name)
}

// repoName derives a provider name from a Git URL — last path segment without ".git".
func repoName(repoURL string) string {
	parts := strings.Split(strings.TrimRight(repoURL, "/"), "/")
	name := parts[len(parts)-1]
	return strings.TrimSuffix(name, ".git")
}
