package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

// PluginConfig represents configuration for a plugin
type PluginConfig struct {
	ID         string          `json:"id"`
	PluginName string          `json:"plugin_name"`
	PluginType string          `json:"plugin_type"`
	Config     json.RawMessage `json:"config"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
}

// handleGetPluginConfig retrieves configuration for a specific plugin
func (s *Server) handleGetPluginConfig(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	pluginName := vars["name"]

	if s.persistence == nil {
		writeErrorResponse(w, http.StatusServiceUnavailable, "Database not available")
		return
	}

	db := s.persistence.GetDB()

	var config PluginConfig
	err := db.QueryRow(`
		SELECT id, plugin_name, plugin_type, config, created_at, updated_at
		FROM plugin_config
		WHERE plugin_name = ?
	`, pluginName).Scan(&config.ID, &config.PluginName, &config.PluginType, &config.Config, &config.CreatedAt, &config.UpdatedAt)

	if err != nil {
		// Return empty config if not found
		writeJSONResponse(w, http.StatusOK, map[string]any{
			"plugin_name": pluginName,
			"configured":  false,
			"config":      nil,
		})
		return
	}

	writeJSONResponse(w, http.StatusOK, map[string]any{
		"id":          config.ID,
		"plugin_name": config.PluginName,
		"plugin_type": config.PluginType,
		"configured":  true,
		"config":      config.Config,
		"created_at":  config.CreatedAt,
		"updated_at":  config.UpdatedAt,
	})
}

// handleSetPluginConfig saves configuration for a plugin
func (s *Server) handleSetPluginConfig(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	pluginName := vars["name"]

	if s.persistence == nil {
		writeErrorResponse(w, http.StatusServiceUnavailable, "Database not available")
		return
	}

	var req struct {
		Config map[string]any `json:"config"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequestResponse(w, fmt.Sprintf("Invalid request: %v", err))
		return
	}

	// Get plugin type from registry
	pluginType := "Unknown"
	for pType, plugins := range s.registry.GetAllPlugins() {
		if _, ok := plugins[pluginName]; ok {
			pluginType = pType
			break
		}
	}

	db := s.persistence.GetDB()

	// Serialize config to JSON
	configJSON, err := json.Marshal(req.Config)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to serialize config: %v", err))
		return
	}

	now := time.Now()

	// Check if config already exists
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM plugin_config WHERE plugin_name = ?", pluginName).Scan(&count)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to check existing config: %v", err))
		return
	}

	if count > 0 {
		// Update existing config
		_, err = db.Exec(`
			UPDATE plugin_config
			SET config = ?, updated_at = ?
			WHERE plugin_name = ?
		`, configJSON, now, pluginName)
	} else {
		// Insert new config
		configID := fmt.Sprintf("config_%s_%d", pluginName, now.Unix())
		_, err = db.Exec(`
			INSERT INTO plugin_config (id, plugin_name, plugin_type, config, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?)
		`, configID, pluginName, pluginType, configJSON, now, now)
	}

	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to save config: %v", err))
		return
	}

	writeJSONResponse(w, http.StatusOK, map[string]any{
		"message":     "Configuration saved successfully",
		"plugin_name": pluginName,
		"configured":  true,
	})
}

// handleDeletePluginConfig deletes configuration for a plugin
func (s *Server) handleDeletePluginConfig(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	pluginName := vars["name"]

	if s.persistence == nil {
		writeErrorResponse(w, http.StatusServiceUnavailable, "Database not available")
		return
	}

	db := s.persistence.GetDB()

	result, err := db.Exec("DELETE FROM plugin_config WHERE plugin_name = ?", pluginName)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to delete config: %v", err))
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		writeErrorResponse(w, http.StatusNotFound, "Configuration not found")
		return
	}

	writeJSONResponse(w, http.StatusOK, map[string]any{
		"message":     "Configuration deleted successfully",
		"plugin_name": pluginName,
	})
}

// handleListPluginConfigs lists all plugin configurations
func (s *Server) handleListPluginConfigs(w http.ResponseWriter, r *http.Request) {
	if s.persistence == nil {
		writeErrorResponse(w, http.StatusServiceUnavailable, "Database not available")
		return
	}

	db := s.persistence.GetDB()

	rows, err := db.Query(`
		SELECT id, plugin_name, plugin_type, config, created_at, updated_at
		FROM plugin_config
		ORDER BY plugin_type, plugin_name
	`)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to query configs: %v", err))
		return
	}
	defer rows.Close()

	var configs []map[string]any
	for rows.Next() {
		var config PluginConfig
		var configData json.RawMessage
		err := rows.Scan(&config.ID, &config.PluginName, &config.PluginType, &configData, &config.CreatedAt, &config.UpdatedAt)
		if err != nil {
			continue
		}

		// Mask sensitive fields
		var configMap map[string]any
		json.Unmarshal(configData, &configMap)
		maskSensitiveFields(configMap)

		configs = append(configs, map[string]any{
			"id":          config.ID,
			"plugin_name": config.PluginName,
			"plugin_type": config.PluginType,
			"configured":  true,
			"config":      configMap,
			"created_at":  config.CreatedAt,
			"updated_at":  config.UpdatedAt,
		})
	}

	writeJSONResponse(w, http.StatusOK, configs)
}

// maskSensitiveFields masks API keys and passwords in config
func maskSensitiveFields(config map[string]any) {
	sensitiveFields := []string{"api_key", "apikey", "api_key", "password", "secret", "token"}
	for _, field := range sensitiveFields {
		if _, ok := config[field]; ok {
			config[field] = "********"
		}
	}
}

// initializePluginConfig creates the plugin_config table
func (s *Server) initializePluginConfig() error {
	if s.persistence == nil {
		return nil
	}

	db := s.persistence.GetDB()

	createTableSQL := `
	CREATE TABLE IF NOT EXISTS plugin_config (
		id TEXT PRIMARY KEY,
		plugin_name TEXT NOT NULL,
		plugin_type TEXT NOT NULL,
		config TEXT,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		UNIQUE(plugin_name)
	);
	`
	_, err := db.Exec(createTableSQL)
	if err != nil {
		return fmt.Errorf("failed to create plugin_config table: %w", err)
	}

	return nil
}
