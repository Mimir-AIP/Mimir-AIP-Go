package plugins

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"plugin"
	"time"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"

	"github.com/mimir-aip/mimir-aip-go/pkg/metadatastore"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	pipelinepkg "github.com/mimir-aip/mimir-aip-go/pkg/pipeline"
)

// Service provides plugin management operations
type Service struct {
	store     metadatastore.MetadataStore
	registry  map[string]pipelinepkg.Plugin
	pluginDir string // Directory to store compiled plugins
	tempDir   string // Directory for temporary Git clones
}

// NewService creates a new plugin service
func NewService(store metadatastore.MetadataStore, pluginDir, tempDir string) (*Service, error) {
	// Create directories if they don't exist
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create plugin directory: %w", err)
	}
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	s := &Service{
		store:     store,
		registry:  make(map[string]pipelinepkg.Plugin),
		pluginDir: pluginDir,
		tempDir:   tempDir,
	}

	// Register default plugins
	s.registry["default"] = pipelinepkg.NewDefaultPlugin()
	s.registry["builtin"] = pipelinepkg.NewDefaultPlugin()

	return s, nil
}

// LoadPersistedPlugins loads all active plugins from the database into the registry
func (s *Service) LoadPersistedPlugins() error {
	plugins, err := s.store.ListPlugins()
	if err != nil {
		return fmt.Errorf("failed to list plugins: %w", err)
	}

	for _, pluginMeta := range plugins {
		if pluginMeta.Status != models.PluginStatusActive {
			log.Printf("Skipping plugin %s with status %s", pluginMeta.Name, pluginMeta.Status)
			continue
		}

		if err := s.LoadPlugin(pluginMeta.Name); err != nil {
			log.Printf("Warning: Failed to load plugin %s: %v", pluginMeta.Name, err)
			// Mark as error but continue
			s.store.UpdatePluginStatus(pluginMeta.Name, models.PluginStatusError)
			continue
		}

		log.Printf("Loaded plugin: %s (version %s)", pluginMeta.Name, pluginMeta.Version)
	}

	return nil
}

// InstallPlugin installs a plugin from a Git repository
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
		return nil, fmt.Errorf("failed to clone repository: %w", err)
	}

	// Parse plugin.yaml
	pluginDefPath := filepath.Join(repoDir, "plugin.yaml")
	pluginDef, err := s.parsePluginDefinition(pluginDefPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse plugin definition: %w", err)
	}

	// Validate plugin definition
	if err := s.validatePluginDefinition(pluginDef); err != nil {
		return nil, fmt.Errorf("invalid plugin definition: %w", err)
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

	// Build the plugin
	log.Printf("Building plugin %s version %s", pluginDef.Name, pluginDef.Version)
	binaryData, err := s.buildPlugin(repoDir, pluginDef.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to build plugin: %w", err)
	}

	// Create plugin metadata
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

	// Save to database
	if err := s.store.SavePlugin(plugin, binaryData); err != nil {
		return nil, fmt.Errorf("failed to save plugin: %w", err)
	}

	// Load into registry
	if err := s.LoadPlugin(plugin.Name); err != nil {
		log.Printf("Warning: Failed to load plugin into registry: %v", err)
		s.store.UpdatePluginStatus(plugin.Name, models.PluginStatusError)
		return plugin, fmt.Errorf("plugin saved but failed to load: %w", err)
	}

	log.Printf("Successfully installed plugin %s version %s", plugin.Name, plugin.Version)
	return plugin, nil
}

// LoadPlugin loads a plugin from the database into the registry
func (s *Service) LoadPlugin(name string) error {
	// Get plugin metadata
	pluginMeta, err := s.store.GetPlugin(name)
	if err != nil {
		return fmt.Errorf("failed to get plugin metadata: %w", err)
	}

	// Get plugin binary
	binaryData, err := s.store.GetPluginBinary(name)
	if err != nil {
		return fmt.Errorf("failed to get plugin binary: %w", err)
	}

	// Write binary to disk
	pluginPath := filepath.Join(s.pluginDir, fmt.Sprintf("%s.so", name))
	if err := os.WriteFile(pluginPath, binaryData, 0755); err != nil {
		return fmt.Errorf("failed to write plugin binary: %w", err)
	}

	// Open the plugin
	p, err := plugin.Open(pluginPath)
	if err != nil {
		return fmt.Errorf("failed to open plugin: %w", err)
	}

	// Look for the Plugin symbol (must be exported)
	symPlugin, err := p.Lookup("Plugin")
	if err != nil {
		return fmt.Errorf("plugin does not export 'Plugin' symbol: %w", err)
	}

	// Assert that it implements the pipeline.Plugin interface
	pluginInstance, ok := symPlugin.(pipelinepkg.Plugin)
	if !ok {
		return fmt.Errorf("plugin does not implement pipeline.Plugin interface")
	}

	// Register in registry
	s.registry[name] = pluginInstance

	// Update last loaded timestamp
	now := time.Now()
	pluginMeta.LastLoadedAt = &now
	pluginMeta.Status = models.PluginStatusActive
	pluginMeta.UpdatedAt = now

	// Re-save to update timestamp (use existing binary)
	if err := s.store.SavePlugin(pluginMeta, binaryData); err != nil {
		log.Printf("Warning: Failed to update plugin metadata: %v", err)
	}

	return nil
}

// GetPlugin retrieves a plugin from the registry (implements pipeline.PluginRegistry interface)
func (s *Service) GetPlugin(name string) (pipelinepkg.Plugin, bool) {
	p, ok := s.registry[name]
	return p, ok
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
	// Remove from registry
	delete(s.registry, name)

	// Remove plugin file
	pluginPath := filepath.Join(s.pluginDir, fmt.Sprintf("%s.so", name))
	os.Remove(pluginPath) // Ignore error if file doesn't exist

	// Remove from database
	return s.store.DeletePlugin(name)
}

// UpdatePlugin updates a plugin to the latest version from Git
func (s *Service) UpdatePlugin(name string, gitRef string) (*models.Plugin, error) {
	// Get existing plugin
	existing, err := s.store.GetPlugin(name)
	if err != nil {
		return nil, fmt.Errorf("plugin not found: %w", err)
	}

	// Uninstall current version (but keep metadata for reinstall on failure)
	delete(s.registry, name)

	// Create temp directory
	installID := uuid.New().String()
	repoDir := filepath.Join(s.tempDir, installID)
	defer os.RemoveAll(repoDir)

	// Clone repository
	if gitRef == "" {
		gitRef = "main"
	}

	if err := s.cloneRepo(existing.RepositoryURL, gitRef, repoDir); err != nil {
		return nil, fmt.Errorf("failed to clone repository: %w", err)
	}

	// Parse plugin definition
	pluginDefPath := filepath.Join(repoDir, "plugin.yaml")
	pluginDef, err := s.parsePluginDefinition(pluginDefPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse plugin definition: %w", err)
	}

	// Validate
	if err := s.validatePluginDefinition(pluginDef); err != nil {
		return nil, fmt.Errorf("invalid plugin definition: %w", err)
	}

	// Ensure name matches
	if pluginDef.Name != name {
		return nil, fmt.Errorf("plugin name mismatch: expected %s, got %s", name, pluginDef.Name)
	}

	// Get commit hash
	commitHash, _ := s.getCommitHash(repoDir)

	// Build plugin
	log.Printf("Building updated plugin %s version %s", pluginDef.Name, pluginDef.Version)
	binaryData, err := s.buildPlugin(repoDir, pluginDef.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to build plugin: %w", err)
	}

	// Update metadata
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

	// Save to database
	if err := s.store.SavePlugin(existing, binaryData); err != nil {
		return nil, fmt.Errorf("failed to save updated plugin: %w", err)
	}

	// Load into registry
	if err := s.LoadPlugin(name); err != nil {
		log.Printf("Warning: Failed to load updated plugin: %v", err)
		s.store.UpdatePluginStatus(name, models.PluginStatusError)
		return existing, fmt.Errorf("plugin updated but failed to load: %w", err)
	}

	log.Printf("Successfully updated plugin %s to version %s", name, existing.Version)
	return existing, nil
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
	if len(def.Actions) == 0 {
		return fmt.Errorf("plugin must define at least one action")
	}

	// Validate actions
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

	return nil
}

// buildPlugin builds the plugin as a Go plugin (.so file)
func (s *Service) buildPlugin(repoDir, pluginName string) ([]byte, error) {
	outputPath := filepath.Join(s.tempDir, fmt.Sprintf("%s.so", pluginName))

	// Build command: go build -buildmode=plugin -o output.so
	cmd := exec.Command("go", "build", "-buildmode=plugin", "-o", outputPath)
	cmd.Dir = repoDir
	cmd.Env = append(os.Environ(), "CGO_ENABLED=1") // Required for plugin mode

	// Capture output for debugging
	var stdout, stderr io.Writer
	stdout = os.Stdout
	stderr = os.Stderr
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	log.Printf("Building plugin in %s", repoDir)
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("go build failed: %w", err)
	}

	// Read the compiled binary
	binaryData, err := os.ReadFile(outputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read compiled plugin: %w", err)
	}

	return binaryData, nil
}
