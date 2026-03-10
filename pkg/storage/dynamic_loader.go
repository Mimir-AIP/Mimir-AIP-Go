package storage

import (
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"github.com/mimir-aip/mimir-aip-go/pkg/pluginruntime"
)

// PluginLoader compiles and loads external storage plugins at runtime using the
// shared pluginruntime clone/build/cache/open flow.
type PluginLoader struct {
	runtime *pluginruntime.Loader[models.StoragePlugin]
}

// NewPluginLoader creates a PluginLoader.
func NewPluginLoader(appDir, cacheDir, tempDir string) (*PluginLoader, error) {
	runtime, err := pluginruntime.NewLoader(pluginruntime.BuildSpec[models.StoragePlugin]{
		LogPrefix:      "storage plugin loader",
		AppDir:         appDir,
		CacheDir:       cacheDir,
		TempDir:        tempDir,
		HostPackageDir: "storage-plugins",
		ClonePrefix:    "sp",
		SymbolName:     "Plugin",
		DefaultGitRef:  "main",
		Resolver:       pluginruntime.ResolveSymbol[models.StoragePlugin],
	})
	if err != nil {
		return nil, err
	}
	return &PluginLoader{runtime: runtime}, nil
}

// CompileAndLoad clones, compiles, caches, and loads the storage plugin.
func (l *PluginLoader) CompileAndLoad(name, repoURL, gitRef, commitHash string) (models.StoragePlugin, string, error) {
	return l.runtime.CompileAndLoad(name, repoURL, gitRef, commitHash)
}

// LoadCached opens a previously compiled .so without recompiling.
func (l *PluginLoader) LoadCached(name string) (models.StoragePlugin, error) {
	return l.runtime.LoadCached(name)
}

func (l *PluginLoader) soPath(name string) string {
	return l.runtime.SoPath(name)
}

func (l *PluginLoader) metaPath(name string) string {
	return l.runtime.MetaPath(name)
}
