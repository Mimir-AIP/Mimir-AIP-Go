package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/AI"
	"github.com/gorilla/mux"
)

// PluginMetadata represents plugin metadata stored in database
type PluginMetadata struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	Version     string    `json:"version"`
	Description string    `json:"description"`
	Author      string    `json:"author,omitempty"`
	FilePath    string    `json:"file_path,omitempty"`
	IsBuiltin   bool      `json:"is_builtin"`
	IsEnabled   bool      `json:"is_enabled"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// UpdatePluginRequest represents a plugin update request
type UpdatePluginRequest struct {
	IsEnabled *bool `json:"is_enabled,omitempty"`
}

// AIProviderInfo represents information about an AI provider
type AIProviderInfo struct {
	Provider          string   `json:"provider"`
	Name              string   `json:"name"`
	Available         bool     `json:"available"`
	Configured        bool     `json:"configured"`
	Models            []string `json:"models,omitempty"`
	DefaultModel      string   `json:"default_model,omitempty"`
	RequiresAPIKey    bool     `json:"requires_api_key"`
	SupportsCustomURL bool     `json:"supports_custom_url"`
	HasCodingPlan     bool     `json:"has_coding_plan,omitempty"` // For Z.ai
	Description       string   `json:"description"`
}

// handleListAIProviders returns information about available AI providers
func (s *Server) handleListAIProviders(w http.ResponseWriter, r *http.Request) {
	providers := []AIProviderInfo{
		{
			Provider:          "openai",
			Name:              "OpenAI",
			Available:         false,
			Configured:        false,
			Models:            AI.GetAvailableModelsForProvider(AI.ProviderOpenAI),
			DefaultModel:      "gpt-4o",
			RequiresAPIKey:    true,
			SupportsCustomURL: false,
			Description:       "OpenAI GPT-4 and GPT-3.5 models",
		},
		{
			Provider:          "anthropic",
			Name:              "Anthropic",
			Available:         false,
			Configured:        false,
			Models:            AI.GetAvailableModelsForProvider(AI.ProviderAnthropic),
			DefaultModel:      "claude-sonnet-4-20250514",
			RequiresAPIKey:    true,
			SupportsCustomURL: false,
			Description:       "Anthropic Claude models",
		},
		{
			Provider:          "openrouter",
			Name:              "OpenRouter",
			Available:         false,
			Configured:        false,
			Models:            []string{}, // Fetched from API when configured
			DefaultModel:      "",
			RequiresAPIKey:    true,
			SupportsCustomURL: false,
			Description:       "OpenRouter aggregation platform - models fetched dynamically",
		},
		{
			Provider:          "z-ai",
			Name:              "Z.ai",
			Available:         false,
			Configured:        false,
			Models:            AI.GetAvailableModelsForProvider(AI.ProviderZAi),
			DefaultModel:      "claude-sonnet-4-20250514",
			RequiresAPIKey:    true,
			SupportsCustomURL: false,
			HasCodingPlan:     true,
			Description:       "Z.ai API with optional coding plan",
		},
		{
			Provider:          "ollama",
			Name:              "Ollama",
			Available:         false,
			Configured:        false,
			Models:            []string{}, // Fetched from local Ollama when configured
			DefaultModel:      "",
			RequiresAPIKey:    false,
			SupportsCustomURL: true,
			Description:       "Local LLM via Ollama - models fetched from local server",
		},
		{
			Provider:          "local",
			Name:              "Local LLM",
			Available:         true,
			Configured:        false,
			Models:            AI.GetAvailableModelsForProvider(AI.ProviderLocal),
			DefaultModel:      "tinyllama-1.1b-chat.q4_0.gguf",
			RequiresAPIKey:    false,
			SupportsCustomURL: false,
			Description:       "Bundled local LLM - runs offline with TinyLlama or Phi-2",
		},
		{
			Provider:       "mock",
			Name:           "Mock",
			Available:      true,
			Configured:     true,
			Models:         AI.GetAvailableModelsForProvider(AI.ProviderMock),
			DefaultModel:   "mock-gpt-4",
			RequiresAPIKey: false,
			Description:    "Mock LLM for testing and demos",
		},
	}

	// Check which providers are actually available and configured
	for i := range providers {
		// Check if configured in database
		if s.persistence != nil {
			db := s.persistence.GetDB()
			var configJSON []byte
			err := db.QueryRow("SELECT config FROM plugin_config WHERE plugin_name = ?", providers[i].Provider).Scan(&configJSON)
			if err == nil && configJSON != nil {
				providers[i].Configured = true
				// Extract model from config
				var config map[string]interface{}
				json.Unmarshal(configJSON, &config)
				if model, ok := config["model"].(string); ok && model != "" {
					providers[i].DefaultModel = model
				}
				// Check if API key is set
				if apiKey, ok := config["api_key"].(string); ok && apiKey != "" {
					providers[i].Available = true
				}
			}
		}
	}

	writeJSONResponse(w, http.StatusOK, providers)
}

// handleFetchProviderModels fetches available models from an LLM provider API
func (s *Server) handleFetchProviderModels(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	provider := vars["provider"]

	// Get config from database if available
	var apiKey, baseURL string
	if s.persistence != nil {
		db := s.persistence.GetDB()
		var configJSON []byte
		err := db.QueryRow("SELECT config FROM plugin_config WHERE plugin_name = ?", provider).Scan(&configJSON)
		if err == nil && configJSON != nil {
			var config map[string]interface{}
			json.Unmarshal(configJSON, &config)
			if key, ok := config["api_key"].(string); ok {
				apiKey = key
			}
			if url, ok := config["base_url"].(string); ok {
				baseURL = url
			}
		}
	}

	var models []string
	var err error

	switch AI.LLMProvider(provider) {
	case AI.ProviderOpenAI:
		models, err = fetchOpenAIModels(apiKey)
	case AI.ProviderOpenRouter:
		models, err = fetchOpenRouterModels()
	case AI.ProviderOllama:
		models, err = fetchOllamaModels(baseURL)
	case AI.ProviderAnthropic:
		// Anthropic doesn't have a public models endpoint, use static list
		models = []string{"claude-sonnet-4-20250514", "claude-haiku-3-20250506", "claude-opus-4-20250506"}
	case AI.ProviderGoogle:
		models, err = fetchGoogleModels(apiKey)
	case AI.ProviderAzure:
		// Azure models depend on deployment configuration
		models = []string{"gpt-4o", "gpt-4o-mini", "gpt-35-turbo"}
	case AI.ProviderZAi:
		models = []string{"claude-sonnet-4-20250514", "deepseek-coder"}
	case AI.ProviderLocal:
		models = []string{"tinyllama-1.1b-chat.q4_0.gguf", "phi-2.q4_0.gguf", "gemma-2b-it.q4_0.gguf"}
	case AI.ProviderMock:
		models = []string{"mock-gpt-4", "mock-claude-3"}
	default:
		writeErrorResponse(w, http.StatusNotFound, "Unknown provider")
		return
	}

	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to fetch models: %v", err))
		return
	}

	writeJSONResponse(w, http.StatusOK, map[string]any{
		"provider": provider,
		"models":   models,
	})
}

func fetchOpenAIModels(apiKey string) ([]string, error) {
	if apiKey == "" {
		return []string{"gpt-4o", "gpt-4o-mini", "gpt-4-turbo", "gpt-3.5-turbo"}, nil
	}

	req, err := http.NewRequest("GET", "https://api.openai.com/v1/models", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OpenAI API returned status %d", resp.StatusCode)
	}

	var result struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	models := make([]string, 0, len(result.Data))
	for _, m := range result.Data {
		models = append(models, m.ID)
	}
	return models, nil
}

func fetchOpenRouterModels() ([]string, error) {
	req, err := http.NewRequest("GET", "https://openrouter.ai/api/v1/models", nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OpenRouter API returned status %d", resp.StatusCode)
	}

	var result struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	models := make([]string, 0, len(result.Data))
	for _, m := range result.Data {
		models = append(models, m.ID)
	}
	return models, nil
}

func fetchOllamaModels(baseURL string) ([]string, error) {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	req, err := http.NewRequest("GET", baseURL+"/api/tags", nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Ollama API returned status %d", resp.StatusCode)
	}

	var result struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	models := make([]string, 0, len(result.Models))
	for _, m := range result.Models {
		models = append(models, m.Name)
	}
	return models, nil
}

func fetchGoogleModels(apiKey string) ([]string, error) {
	url := "https://generativelanguage.googleapis.com/v1beta/models?key=" + apiKey
	if apiKey == "" {
		return []string{"gemini-1.5-pro", "gemini-1.5-flash", "gemini-1.0-pro"}, nil
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Google API returned status %d", resp.StatusCode)
	}

	var result struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	models := make([]string, 0, len(result.Models))
	for _, m := range result.Models {
		// Extract just the model name from "models/model-name"
		name := m.Name
		if len(name) > 8 {
			models = append(models, name[8:])
		}
	}
	return models, nil
}

// handleListPluginMetadata lists all plugins with metadata
func (s *Server) handleListPluginMetadata(w http.ResponseWriter, r *http.Request) {
	if s.persistence == nil {
		// Fallback to registry-only if no database
		s.handleListPlugins(w, r)
		return
	}

	db := s.persistence.GetDB()

	// Get all plugin metadata from database
	query := `
		SELECT id, name, type, version, description, author, file_path, is_builtin, is_enabled, created_at, updated_at
		FROM plugin_metadata
		ORDER BY is_builtin DESC, type ASC, name ASC
	`
	rows, err := db.Query(query)
	if err != nil {
		// If table doesn't exist, initialize it
		if err := s.initializePluginMetadata(); err != nil {
			writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to initialize plugin metadata: %v", err))
			return
		}
		// Retry query
		rows, err = db.Query(query)
		if err != nil {
			writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to query plugins: %v", err))
			return
		}
	}
	defer rows.Close()

	var plugins []map[string]any
	for rows.Next() {
		var p PluginMetadata
		err := rows.Scan(&p.ID, &p.Name, &p.Type, &p.Version, &p.Description, &p.Author, &p.FilePath, &p.IsBuiltin, &p.IsEnabled, &p.CreatedAt, &p.UpdatedAt)
		if err != nil {
			continue
		}

		// Get input_schema from registry if available
		var inputSchema map[string]any
		if plugin, err := s.registry.GetPlugin(p.Type, p.Name); err == nil {
			inputSchema = plugin.GetInputSchema()
		}

		pluginMap := map[string]any{
			"id":          p.ID,
			"name":        p.Name,
			"type":        p.Type,
			"version":     p.Version,
			"description": p.Description,
			"author":      p.Author,
			"file_path":   p.FilePath,
			"is_builtin":  p.IsBuiltin,
			"is_enabled":  p.IsEnabled,
			"created_at":  p.CreatedAt,
			"updated_at":  p.UpdatedAt,
		}
		if inputSchema != nil {
			pluginMap["input_schema"] = inputSchema
		}
		plugins = append(plugins, pluginMap)
	}

	writeJSONResponse(w, http.StatusOK, plugins)
}

// handleUploadPlugin handles plugin file uploads
func (s *Server) handleUploadPlugin(w http.ResponseWriter, r *http.Request) {
	if s.persistence == nil {
		writeErrorResponse(w, http.StatusServiceUnavailable, "Plugin management requires database")
		return
	}

	// Parse multipart form (max 50MB)
	err := r.ParseMultipartForm(50 << 20)
	if err != nil {
		writeBadRequestResponse(w, fmt.Sprintf("Failed to parse form: %v", err))
		return
	}

	file, header, err := r.FormFile("plugin")
	if err != nil {
		writeBadRequestResponse(w, fmt.Sprintf("Failed to read plugin file: %v", err))
		return
	}
	defer file.Close()

	// Validate file extension
	ext := filepath.Ext(header.Filename)
	if ext != ".so" && ext != ".dll" {
		writeBadRequestResponse(w, "Invalid file type. Only .so (Linux) or .dll (Windows) files are supported")
		return
	}

	// Create plugins directory
	pluginDir := "./data/plugins"
	os.MkdirAll(pluginDir, 0755)

	// Generate unique filename
	filename := fmt.Sprintf("%d_%s", time.Now().Unix(), header.Filename)
	filePath := filepath.Join(pluginDir, filename)

	// Save file
	dst, err := os.Create(filePath)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to save plugin: %v", err))
		return
	}
	defer dst.Close()

	_, err = io.Copy(dst, file)
	if err != nil {
		os.Remove(filePath)
		writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to write plugin: %v", err))
		return
	}

	// Create plugin metadata
	pluginID := fmt.Sprintf("plugin_%d", time.Now().Unix())
	metadata := PluginMetadata{
		ID:          pluginID,
		Name:        header.Filename,
		Type:        "Custom",
		Version:     "1.0.0",
		Description: "User-uploaded plugin",
		FilePath:    filePath,
		IsBuiltin:   false,
		IsEnabled:   false, // Disabled by default for safety
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Save to database
	db := s.persistence.GetDB()
	query := `
		INSERT INTO plugin_metadata (id, name, type, version, description, author, file_path, is_builtin, is_enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err = db.Exec(query, metadata.ID, metadata.Name, metadata.Type, metadata.Version, metadata.Description, metadata.Author, metadata.FilePath, metadata.IsBuiltin, metadata.IsEnabled, metadata.CreatedAt, metadata.UpdatedAt)
	if err != nil {
		os.Remove(filePath)
		writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to save metadata: %v", err))
		return
	}

	writeJSONResponse(w, http.StatusOK, map[string]any{
		"message": "Plugin uploaded successfully. Enable it to activate.",
		"plugin":  metadata,
	})
}

// handleUpdatePlugin updates plugin metadata (enable/disable)
func (s *Server) handleUpdatePlugin(w http.ResponseWriter, r *http.Request) {
	if s.persistence == nil {
		writeErrorResponse(w, http.StatusServiceUnavailable, "Plugin management requires database")
		return
	}

	vars := mux.Vars(r)
	pluginID := vars["id"]

	var req UpdatePluginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequestResponse(w, fmt.Sprintf("Invalid request: %v", err))
		return
	}

	db := s.persistence.GetDB()

	// Check if plugin exists
	var isBuiltin bool
	err := db.QueryRow("SELECT is_builtin FROM plugin_metadata WHERE id = ?", pluginID).Scan(&isBuiltin)
	if err != nil {
		writeErrorResponse(w, http.StatusNotFound, "Plugin not found")
		return
	}

	// Update plugin
	if req.IsEnabled != nil {
		query := `UPDATE plugin_metadata SET is_enabled = ?, updated_at = ? WHERE id = ?`
		_, err := db.Exec(query, *req.IsEnabled, time.Now(), pluginID)
		if err != nil {
			writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to update plugin: %v", err))
			return
		}
	}

	// Get updated plugin
	var p PluginMetadata
	query := `SELECT id, name, type, version, description, author, file_path, is_builtin, is_enabled, created_at, updated_at FROM plugin_metadata WHERE id = ?`
	err = db.QueryRow(query, pluginID).Scan(&p.ID, &p.Name, &p.Type, &p.Version, &p.Description, &p.Author, &p.FilePath, &p.IsBuiltin, &p.IsEnabled, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to retrieve plugin: %v", err))
		return
	}

	writeJSONResponse(w, http.StatusOK, p)
}

// handleDeletePlugin deletes a user-uploaded plugin
func (s *Server) handleDeletePlugin(w http.ResponseWriter, r *http.Request) {
	if s.persistence == nil {
		writeErrorResponse(w, http.StatusServiceUnavailable, "Plugin management requires database")
		return
	}

	vars := mux.Vars(r)
	pluginID := vars["id"]

	db := s.persistence.GetDB()

	// Check if plugin exists and is not builtin
	var isBuiltin bool
	var filePath string
	err := db.QueryRow("SELECT is_builtin, file_path FROM plugin_metadata WHERE id = ?", pluginID).Scan(&isBuiltin, &filePath)
	if err != nil {
		writeErrorResponse(w, http.StatusNotFound, "Plugin not found")
		return
	}

	if isBuiltin {
		writeErrorResponse(w, http.StatusForbidden, "Cannot delete built-in plugins")
		return
	}

	// Delete file if exists
	if filePath != "" {
		os.Remove(filePath)
	}

	// Delete from database
	_, err = db.Exec("DELETE FROM plugin_metadata WHERE id = ?", pluginID)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to delete plugin: %v", err))
		return
	}

	writeJSONResponse(w, http.StatusOK, map[string]any{
		"message": "Plugin deleted successfully",
	})
}

// handleReloadPlugin reloads a plugin without server restart
func (s *Server) handleReloadPlugin(w http.ResponseWriter, r *http.Request) {
	if s.persistence == nil {
		writeErrorResponse(w, http.StatusServiceUnavailable, "Plugin management requires database")
		return
	}

	vars := mux.Vars(r)
	pluginID := vars["id"]

	db := s.persistence.GetDB()

	// Check if plugin exists
	var name string
	err := db.QueryRow("SELECT name FROM plugin_metadata WHERE id = ?", pluginID).Scan(&name)
	if err != nil {
		writeErrorResponse(w, http.StatusNotFound, "Plugin not found")
		return
	}

	// TODO: Implement actual plugin reloading logic
	// For now, just return success
	// In a full implementation, you would:
	// 1. Unload the plugin from registry
	// 2. Reload the plugin file
	// 3. Re-register it

	writeJSONResponse(w, http.StatusOK, map[string]any{
		"message": fmt.Sprintf("Plugin '%s' reload triggered (requires server restart for full effect)", name),
	})
}

// initializePluginMetadata creates the plugin_metadata table and seeds it with built-in plugins
func (s *Server) initializePluginMetadata() error {
	db := s.persistence.GetDB()

	// Create table
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS plugin_metadata (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		type TEXT NOT NULL,
		version TEXT NOT NULL,
		description TEXT,
		author TEXT,
		file_path TEXT,
		is_builtin BOOLEAN NOT NULL DEFAULT 0,
		is_enabled BOOLEAN NOT NULL DEFAULT 1,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL
	);
	`
	_, err := db.Exec(createTableSQL)
	if err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	// Seed with built-in plugins from registry
	now := time.Now()
	for pluginType, typePlugins := range s.registry.GetAllPlugins() {
		for pluginName, plugin := range typePlugins {
			pluginID := fmt.Sprintf("builtin_%s_%s", pluginType, pluginName)

			// Check if already exists
			var count int
			err := db.QueryRow("SELECT COUNT(*) FROM plugin_metadata WHERE id = ?", pluginID).Scan(&count)
			if err != nil || count > 0 {
				continue
			}

			description := fmt.Sprintf("%s plugin for %s", pluginName, pluginType)
			if pluginType == "Input" {
				description = fmt.Sprintf("Input plugin for reading %s files", pluginName)
			} else if pluginType == "Output" {
				description = fmt.Sprintf("Output plugin for writing %s files", pluginName)
			}

			// Get version from plugin if available
			version := "1.0.0"
			// Try to get schema which might have version info
			schema := plugin.GetInputSchema()
			if v, ok := schema["version"].(string); ok && v != "" {
				version = v
			}

			query := `
				INSERT INTO plugin_metadata (id, name, type, version, description, author, file_path, is_builtin, is_enabled, created_at, updated_at)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			`
			_, err = db.Exec(query, pluginID, pluginName, pluginType, version, description, "Mimir AIP", "", true, true, now, now)
			if err != nil {
				return fmt.Errorf("failed to insert plugin metadata: %w", err)
			}
		}
	}

	return nil
}
