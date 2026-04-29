package plugins

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"

	"github.com/mimir-aip/mimir-aip-go/pkg/metadatastore"
	"github.com/mimir-aip/mimir-aip-go/pkg/mlmodel"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"github.com/mimir-aip/mimir-aip-go/pkg/pipeline"
	"github.com/mimir-aip/mimir-aip-go/pkg/pluginruntime"
)

// ValidationError means a plugin repository or manifest was reachable but does
// not satisfy the current host runtime contract.
type ValidationError struct {
	Err error
}

func (e *ValidationError) Error() string { return e.Err.Error() }
func (e *ValidationError) Unwrap() error { return e.Err }

// Service provides plugin metadata management.
type Service struct {
	store   metadatastore.MetadataStore
	tempDir string // Directory for temporary Git clones and validation builds
	appDir  string // Host module root used for install-time plugin validation builds
}

// NewService creates a new plugin service.
func NewService(store metadatastore.MetadataStore, tempDir string) (*Service, error) {
	// Create temp directory if it doesn't exist
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	appDir := os.Getenv("APP_DIR")
	if appDir == "" {
		if _, err := os.Stat("/app/go.mod"); err == nil {
			appDir = "/app"
		} else if wd, err := os.Getwd(); err == nil {
			appDir = wd
		} else {
			return nil, fmt.Errorf("resolve app directory: %w", err)
		}
	}

	return &Service{
		store:   store,
		tempDir: tempDir,
		appDir:  appDir,
	}, nil
}

// InstallPlugin installs a plugin by validating its manifest and runtime symbol,
// then storing its metadata. Workers still compile their own local copy when a
// task uses the plugin, but invalid repositories fail at install time instead of
// later during pipeline execution.
func (s *Service) InstallPlugin(req *models.PluginInstallRequest) (*models.Plugin, error) {
	// Create unique temp directory for this installation
	installID := uuid.New().String()
	repoDir := filepath.Join(s.tempDir, installID)
	defer os.RemoveAll(repoDir) // Cleanup after installation

	log.Printf("Cloning plugin repository from %s", req.RepositoryURL)

	// Clone the repository
	gitRef := req.GitRef
	if gitRef == "" {
		gitRef = "main" // Default branch
	}

	if err := s.cloneRepo(req.RepositoryURL, gitRef, repoDir); err != nil {
		return nil, &ValidationError{Err: fmt.Errorf("failed to clone repository: %w", err)}
	}

	// Parse plugin.yaml
	pluginDefPath := filepath.Join(repoDir, "plugin.yaml")
	pluginDef, err := s.parsePluginDefinition(pluginDefPath)
	if err != nil {
		return nil, &ValidationError{Err: fmt.Errorf("failed to parse plugin definition: %w", err)}
	}

	// Validate plugin definition
	if err := s.validatePluginDefinition(pluginDef); err != nil {
		return nil, &ValidationError{Err: fmt.Errorf("invalid plugin definition: %w", err)}
	}

	// Check if plugin already exists
	existing, _ := s.store.GetPlugin(pluginDef.Name)
	if existing != nil {
		return nil, fmt.Errorf("plugin %s already exists (use update to modify)", pluginDef.Name)
	}

	// Get current commit hash
	commitHash, err := s.getCommitHash(repoDir)
	if err != nil {
		log.Printf("Warning: Failed to get commit hash: %v", err)
		commitHash = ""
	}

	if err := s.validateRuntimeBuild(pluginDef, req.RepositoryURL, gitRef, commitHash); err != nil {
		return nil, &ValidationError{Err: err}
	}

	// Create plugin metadata (no binary - workers will compile from source)
	now := time.Now()
	pluginID := uuid.New().String()

	plugin := &models.Plugin{
		ID:               pluginID,
		Name:             pluginDef.Name,
		Version:          pluginDef.Version,
		Description:      pluginDef.Description,
		Author:           pluginDef.Author,
		RepositoryURL:    req.RepositoryURL,
		GitCommitHash:    commitHash,
		PluginDefinition: *pluginDef,
		Status:           models.PluginStatusActive,
		CreatedAt:        now,
		UpdatedAt:        now,
		Actions:          make([]models.PluginAction, 0),
	}

	// Create action entries
	for _, actionSchema := range pluginDef.Actions {
		actionID := uuid.New().String()
		action := models.PluginAction{
			ID:          actionID,
			PluginID:    pluginID,
			Name:        actionSchema.Name,
			Description: actionSchema.Description,
			Parameters:  actionSchema.Parameters,
			Returns:     actionSchema.Returns,
		}
		plugin.Actions = append(plugin.Actions, action)
	}

	// Save metadata to database (no binary)
	if err := s.store.SavePlugin(plugin, nil); err != nil {
		return nil, fmt.Errorf("failed to save plugin: %w", err)
	}

	log.Printf("Successfully installed plugin %s version %s", plugin.Name, plugin.Version)
	return plugin, nil
}

// ListPlugins lists all installed plugins
func (s *Service) ListPlugins() ([]*models.Plugin, error) {
	return s.store.ListPlugins()
}

// GetPluginMetadata retrieves plugin metadata from the database
func (s *Service) GetPluginMetadata(name string) (*models.Plugin, error) {
	return s.store.GetPlugin(name)
}

// UninstallPlugin removes a plugin
func (s *Service) UninstallPlugin(name string) error {
	// Remove from database. Already loaded Go plugin code cannot be unloaded from
	// running worker/orchestrator processes; new tasks will stop resolving it from
	// metadata, but old processes may retain opened symbols until they exit.
	return s.store.DeletePlugin(name)
}

// UpdatePlugin updates a plugin to the latest version from Git
func (s *Service) UpdatePlugin(name string, gitRef string) (*models.Plugin, error) {
	// Get existing plugin
	existing, err := s.store.GetPlugin(name)
	if err != nil {
		return nil, fmt.Errorf("plugin not found: %w", err)
	}

	// Create temp directory
	installID := uuid.New().String()
	repoDir := filepath.Join(s.tempDir, installID)
	defer os.RemoveAll(repoDir)

	// Clone repository
	if gitRef == "" {
		if existing.GitCommitHash != "" {
			gitRef = existing.GitCommitHash
		} else {
			gitRef = "main"
		}
	}

	if err := s.cloneRepo(existing.RepositoryURL, gitRef, repoDir); err != nil {
		return nil, &ValidationError{Err: fmt.Errorf("failed to clone repository: %w", err)}
	}

	// Parse plugin definition
	pluginDefPath := filepath.Join(repoDir, "plugin.yaml")
	pluginDef, err := s.parsePluginDefinition(pluginDefPath)
	if err != nil {
		return nil, &ValidationError{Err: fmt.Errorf("failed to parse plugin definition: %w", err)}
	}

	// Validate
	if err := s.validatePluginDefinition(pluginDef); err != nil {
		return nil, &ValidationError{Err: fmt.Errorf("invalid plugin definition: %w", err)}
	}

	// Ensure name matches
	if pluginDef.Name != name {
		return nil, &ValidationError{Err: fmt.Errorf("plugin name mismatch: expected %s, got %s", name, pluginDef.Name)}
	}

	// Get commit hash
	commitHash, _ := s.getCommitHash(repoDir)
	if err := s.validateRuntimeBuild(pluginDef, existing.RepositoryURL, gitRef, commitHash); err != nil {
		return nil, &ValidationError{Err: err}
	}

	// Update metadata (no binary - workers will compile from source)
	now := time.Now()
	existing.Version = pluginDef.Version
	existing.Description = pluginDef.Description
	existing.Author = pluginDef.Author
	existing.GitCommitHash = commitHash
	existing.PluginDefinition = *pluginDef
	existing.UpdatedAt = now
	existing.Status = models.PluginStatusActive

	// Update actions
	existing.Actions = make([]models.PluginAction, 0)
	for _, actionSchema := range pluginDef.Actions {
		actionID := uuid.New().String()
		action := models.PluginAction{
			ID:          actionID,
			PluginID:    existing.ID,
			Name:        actionSchema.Name,
			Description: actionSchema.Description,
			Parameters:  actionSchema.Parameters,
			Returns:     actionSchema.Returns,
		}
		existing.Actions = append(existing.Actions, action)
	}

	// Save to database (no binary)
	if err := s.store.SavePlugin(existing, nil); err != nil {
		return nil, fmt.Errorf("failed to save updated plugin: %w", err)
	}

	log.Printf("Successfully updated plugin %s to version %s", name, existing.Version)
	return existing, nil
}

func (s *Service) validateRuntimeBuild(def *models.PluginDefinition, repoURL, gitRef, commitHash string) error {
	if len(def.Actions) > 0 {
		loader, err := pluginruntime.NewLoader(pluginruntime.BuildSpec[pipeline.Plugin]{
			LogPrefix:      "pipeline plugin validation",
			AppDir:         s.appDir,
			CacheDir:       filepath.Join(s.tempDir, "validation-cache", "pipeline"),
			TempDir:        filepath.Join(s.tempDir, "validation-cache", "pipeline", "tmp"),
			HostPackageDir: "plugins",
			ClonePrefix:    "pp-validate",
			SymbolName:     "Plugin",
			DefaultGitRef:  "main",
			Resolver:       pluginruntime.ResolveSymbol[pipeline.Plugin],
		})
		if err != nil {
			return fmt.Errorf("pipeline plugin validation unavailable: %w", err)
		}
		if _, _, err := loader.CompileAndLoad(def.Name, repoURL, gitRef, commitHash); err != nil {
			return fmt.Errorf("pipeline plugin validation failed: %w", err)
		}
	}
	if def.MLProvider != nil {
		providerName := def.MLProvider.Name
		if providerName == "" {
			providerName = def.Name
		}
		loader, err := pluginruntime.NewLoader(pluginruntime.BuildSpec[mlmodel.Provider]{
			LogPrefix:      "ml provider validation",
			AppDir:         s.appDir,
			CacheDir:       filepath.Join(s.tempDir, "validation-cache", "ml"),
			TempDir:        filepath.Join(s.tempDir, "validation-cache", "ml", "tmp"),
			HostPackageDir: "plugins",
			ClonePrefix:    "mlp-validate",
			SymbolName:     "MLProvider",
			DefaultGitRef:  "main",
			Resolver:       pluginruntime.ResolveSymbol[mlmodel.Provider],
		})
		if err != nil {
			return fmt.Errorf("ml provider validation unavailable: %w", err)
		}
		if _, _, err := loader.CompileAndLoad(providerName, repoURL, gitRef, commitHash); err != nil {
			return fmt.Errorf("ml provider validation failed: %w", err)
		}
	}
	return nil
}

// cloneRepo clones a Git repository to the specified directory
func (s *Service) cloneRepo(repoURL, gitRef, destDir string) error {
	cmd := exec.Command("git", "clone", "--depth=1", "--branch", gitRef, repoURL, destDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git clone failed: %w\nOutput: %s", err, string(output))
	}
	return nil
}

// getCommitHash gets the current commit hash from a Git repository
func (s *Service) getCommitHash(repoDir string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = repoDir
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output[:40]), nil // First 40 chars is the commit hash
}

// parsePluginDefinition parses the plugin.yaml file
func (s *Service) parsePluginDefinition(path string) (*models.PluginDefinition, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read plugin.yaml: %w", err)
	}

	var def models.PluginDefinition
	if err := yaml.Unmarshal(data, &def); err != nil {
		return nil, fmt.Errorf("failed to parse plugin.yaml: %w", err)
	}

	return &def, nil
}

// validatePluginDefinition validates a plugin definition
func (s *Service) validatePluginDefinition(def *models.PluginDefinition) error {
	if def.Name == "" {
		return fmt.Errorf("plugin name is required")
	}
	if def.Version == "" {
		return fmt.Errorf("plugin version is required")
	}
	if len(def.Actions) == 0 && def.MLProvider == nil {
		return fmt.Errorf("plugin must define at least one pipeline action or one ml_provider")
	}
	actionNames := make(map[string]bool)
	for _, action := range def.Actions {
		if action.Name == "" {
			return fmt.Errorf("action name is required")
		}
		if actionNames[action.Name] {
			return fmt.Errorf("duplicate action name: %s", action.Name)
		}
		actionNames[action.Name] = true
	}
	if def.MLProvider != nil {
		if def.MLProvider.Name == "" {
			def.MLProvider.Name = def.Name
		}
		if def.MLProvider.Name == "builtin" {
			return fmt.Errorf("ml provider name 'builtin' is reserved")
		}
		if len(def.MLProvider.Models) == 0 {
			return fmt.Errorf("ml_provider must define at least one model")
		}
	}
	return nil
}
