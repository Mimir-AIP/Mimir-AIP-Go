package mlmodel

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"github.com/mimir-aip/mimir-aip-go/pkg/pluginruntime"
)

func newExternalProviderLoader() (*pluginruntime.Loader[Provider], error) {
	cacheDir := filepath.Join(os.TempDir(), "mimir-aip", "ml-provider-cache")
	return pluginruntime.NewLoader(pluginruntime.BuildSpec[Provider]{
		LogPrefix:      "ml provider loader",
		AppDir:         "/app",
		CacheDir:       cacheDir,
		TempDir:        filepath.Join(cacheDir, "tmp"),
		HostPackageDir: "plugins",
		ClonePrefix:    "mlp",
		SymbolName:     "MLProvider",
		DefaultGitRef:  "main",
		Resolver:       pluginruntime.ResolveSymbol[Provider],
	})
}

func (s *Service) loadExternalProvider(name string) (Provider, error) {
	if s.providerLoader == nil {
		loader, err := newExternalProviderLoader()
		if err != nil {
			return nil, err
		}
		s.providerLoader = loader
	}
	plugin, err := s.store.GetPlugin(name)
	if err != nil {
		return nil, fmt.Errorf("provider plugin not found: %w", err)
	}
	if plugin.PluginDefinition.MLProvider == nil {
		return nil, fmt.Errorf("plugin %s does not declare an ML provider", name)
	}
	artifact, err := s.store.GetActivePluginArtifact(models.PluginKindMLProvider, name)
	if err != nil {
		return nil, fmt.Errorf("provider artifact not found: %w", err)
	}
	provider, err := s.providerLoader.LoadPath(name, artifact.LocalPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load provider %s: %w", name, err)
	}
	return provider, nil
}
