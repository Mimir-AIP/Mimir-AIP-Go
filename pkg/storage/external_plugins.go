package storage

import (
	"fmt"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"log"
	"os"
	"strings"
	"time"
)

func (s *Service) SetPluginLoader(loader *PluginLoader) {
	s.pluginLoader = loader
}

// InstallExternalPlugin clones, compiles, and registers a storage plugin from
// a Git repository. The plugin metadata is persisted so it survives restarts.
func (s *Service) InstallExternalPlugin(req *models.ExternalStoragePluginInstallRequest) (*models.ExternalStoragePlugin, error) {
	if s.pluginLoader == nil {
		return nil, fmt.Errorf("dynamic storage plugin loading is not configured")
	}
	if req.RepositoryURL == "" {
		return nil, fmt.Errorf("repository_url is required")
	}

	gitRef := req.GitRef
	if gitRef == "" {
		gitRef = "main"
	}

	// Derive a plugin name from the repository URL (last path segment without .git).
	name := repoName(req.RepositoryURL)

	// Compile (or use cached .so) and load the plugin.
	sp, commitHash, err := s.pluginLoader.CompileAndLoad(name, req.RepositoryURL, gitRef, "")

	now := time.Now().UTC()
	record := &models.ExternalStoragePlugin{
		Name:          name,
		RepositoryURL: req.RepositoryURL,
		GitCommitHash: commitHash,
		Status:        "active",
		InstalledAt:   now,
		UpdatedAt:     now,
	}

	if err != nil {
		record.Status = "error"
		record.ErrorMessage = err.Error()
		// Persist the error record so the user can see what went wrong.
		_ = s.store.SaveExternalStoragePlugin(record)
		return record, fmt.Errorf("failed to compile storage plugin %s: %w", name, err)
	}

	// Register in the live plugin map.
	s.RegisterPlugin(name, sp)

	if err := s.store.SaveExternalStoragePlugin(record); err != nil {
		return nil, fmt.Errorf("plugin compiled but failed to persist metadata: %w", err)
	}

	log.Printf("Installed external storage plugin: %s @ %s", name, commitHash)
	return record, nil
}

// ListExternalPlugins returns all persisted external storage plugin records.
func (s *Service) ListExternalPlugins() ([]*models.ExternalStoragePlugin, error) {
	return s.store.ListExternalStoragePlugins()
}

// GetExternalPlugin returns metadata for a single external storage plugin.
func (s *Service) GetExternalPlugin(name string) (*models.ExternalStoragePlugin, error) {
	return s.store.GetExternalStoragePlugin(name)
}

// UninstallExternalPlugin removes the plugin from the live registry and
// deletes its persisted metadata. The .so file is removed from the cache.
// Note: Go plugins cannot be unloaded from memory; the process must restart
// for the removal to take full effect on in-flight storage operations.
func (s *Service) UninstallExternalPlugin(name string) error {
	// Remove from live registry.

	s.plugins.Delete(name)

	// Remove .so and meta from cache.
	if s.pluginLoader != nil {
		_ = os.Remove(s.pluginLoader.soPath(name))
		_ = os.Remove(s.pluginLoader.metaPath(name))
	}

	if err := s.store.DeleteExternalStoragePlugin(name); err != nil {
		return fmt.Errorf("failed to delete external storage plugin record: %w", err)
	}

	log.Printf("Uninstalled external storage plugin: %s", name)
	return nil
}

// LoadInstalledExternalPlugins re-registers all persisted external storage
// plugins on startup. If a cached .so exists it is used directly; otherwise
// the plugin is recompiled from its recorded repository URL and commit hash.
func (s *Service) LoadInstalledExternalPlugins() error {
	if s.pluginLoader == nil {
		return nil
	}

	records, err := s.store.ListExternalStoragePlugins()
	if err != nil {
		return fmt.Errorf("failed to list persisted storage plugins: %w", err)
	}

	var loadErrs []string
	for _, rec := range records {
		if rec.Status == "error" {
			log.Printf("Skipping external storage plugin %s (status=error)", rec.Name)
			continue
		}

		// Try cached .so first.
		sp, loadErr := s.pluginLoader.LoadCached(rec.Name)
		if loadErr != nil {
			// Cache miss — recompile from stored repo+commit.
			log.Printf("Recompiling external storage plugin %s from %s @ %s", rec.Name, rec.RepositoryURL, rec.GitCommitHash)
			var compileHash string
			sp, compileHash, loadErr = s.pluginLoader.CompileAndLoad(
				rec.Name, rec.RepositoryURL, rec.GitCommitHash, rec.GitCommitHash,
			)
			if loadErr != nil {
				msg := fmt.Sprintf("%s: %v", rec.Name, loadErr)
				loadErrs = append(loadErrs, msg)
				rec.Status = "error"
				rec.ErrorMessage = loadErr.Error()
				_ = s.store.SaveExternalStoragePlugin(rec)
				continue
			}
			_ = compileHash
		}

		s.RegisterPlugin(rec.Name, sp)
		log.Printf("Loaded external storage plugin: %s", rec.Name)
	}

	if len(loadErrs) > 0 {
		return fmt.Errorf("failed to load %d external storage plugin(s): %s", len(loadErrs), strings.Join(loadErrs, "; "))
	}
	return nil
}

// repoName derives a plugin name from a Git URL — last path segment without ".git".
func repoName(repoURL string) string {
	parts := strings.Split(strings.TrimRight(repoURL, "/"), "/")
	name := parts[len(parts)-1]
	name = strings.TrimSuffix(name, ".git")
	return strings.ToLower(name)
}
