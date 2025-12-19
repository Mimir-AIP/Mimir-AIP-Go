package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

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

	var plugins []PluginMetadata
	for rows.Next() {
		var p PluginMetadata
		err := rows.Scan(&p.ID, &p.Name, &p.Type, &p.Version, &p.Description, &p.Author, &p.FilePath, &p.IsBuiltin, &p.IsEnabled, &p.CreatedAt, &p.UpdatedAt)
		if err != nil {
			continue
		}
		plugins = append(plugins, p)
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
