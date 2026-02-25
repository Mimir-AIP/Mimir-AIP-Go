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
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

// Service provides plugin metadata management
// Note: Plugins are compiled by workers, not the orchestrator
type Service struct {
	store   metadatastore.MetadataStore
	tempDir string // Directory for temporary Git clones
}

// NewService creates a new plugin service
func NewService(store metadatastore.MetadataStore, tempDir string) (*Service, error) {
	// Create temp directory if it doesn't exist
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	return &Service{
		store:   store,
		tempDir: tempDir,
	}, nil
}

// InstallPlugin installs a plugin by storing its metadata
// Workers will compile the plugin from source when needed
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

	log.Printf("Successfully installed plugin %s version %s (workers will compile from source)", plugin.Name, plugin.Version)
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

	log.Printf("Successfully updated plugin %s to version %s (workers will compile from source)", name, existing.Version)
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
