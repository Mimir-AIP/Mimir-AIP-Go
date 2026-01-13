package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Mimir-AIP/Mimir-AIP-Go/utils"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type APIKeyResponse struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

// CreateAPIKeyRequest represents request to create an API key
type CreateAPIKeyRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	ExpiresIn   int    `json:"expires_in,omitempty"` // Days until expiration
}

// handleListAPIKeys lists all API keys (without exposing actual key values)
func (s *Server) handleListAPIKeys(w http.ResponseWriter, r *http.Request) {
	if s.persistence == nil {
		writeErrorResponse(w, http.StatusServiceUnavailable, "API key management requires database")
		return
	}

	db := s.persistence.GetDB()

	// Query all API keys (don't return key_value)
	query := `
		SELECT id, provider, name, endpoint_url, is_active, created_at, updated_at, last_used_at, metadata
		FROM api_keys
		ORDER BY created_at DESC
	`
	rows, err := db.Query(query)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to query API keys: %v", err))
		return
	}
	defer rows.Close()

	var keys []map[string]interface{}
	for rows.Next() {
		var id, provider, name, endpointURL, metadata sql.NullString
		var isActive bool
		var createdAt, updatedAt, lastUsedAt sql.NullTime

		err := rows.Scan(&id, &provider, &name, &endpointURL, &isActive, &createdAt, &updatedAt, &lastUsedAt, &metadata)
		if err != nil {
			continue
		}

		key := map[string]interface{}{
			"id":        id.String,
			"provider":  provider.String,
			"name":      name.String,
			"is_active": isActive,
		}

		if endpointURL.Valid {
			key["endpoint_url"] = endpointURL.String
		}
		if createdAt.Valid {
			key["created_at"] = createdAt.Time.Format(time.RFC3339)
		}
		if updatedAt.Valid {
			key["updated_at"] = updatedAt.Time.Format(time.RFC3339)
		}
		if lastUsedAt.Valid {
			key["last_used_at"] = lastUsedAt.Time.Format(time.RFC3339)
		}
		if metadata.Valid && metadata.String != "" {
			var meta map[string]interface{}
			if err := json.Unmarshal([]byte(metadata.String), &meta); err == nil {
				key["metadata"] = meta
			}
		}

		keys = append(keys, key)
	}

	if keys == nil {
		keys = []map[string]interface{}{}
	}

	writeJSONResponse(w, http.StatusOK, keys)
}

// handleUpdateAPIKey updates an existing API key
func (s *Server) handleUpdateAPIKey(w http.ResponseWriter, r *http.Request) {
	if s.persistence == nil {
		writeErrorResponse(w, http.StatusServiceUnavailable, "API key management requires database")
		return
	}

	vars := mux.Vars(r)
	keyID := vars["id"]

	if keyID == "" {
		writeBadRequestResponse(w, "Key ID is required")
		return
	}

	var req struct {
		Name        *string                `json:"name,omitempty"`
		KeyValue    *string                `json:"key_value,omitempty"`
		IsActive    *bool                  `json:"is_active,omitempty"`
		EndpointURL *string                `json:"endpoint_url,omitempty"`
		Metadata    map[string]interface{} `json:"metadata,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequestResponse(w, fmt.Sprintf("Invalid request: %v", err))
		return
	}

	db := s.persistence.GetDB()

	// Check if API key exists
	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM api_keys WHERE id = ?)", keyID).Scan(&exists)
	if err != nil || !exists {
		writeErrorResponse(w, http.StatusNotFound, "API key not found")
		return
	}

	// Build dynamic update query
	updates := []string{}
	args := []interface{}{}

	if req.Name != nil {
		updates = append(updates, "name = ?")
		args = append(args, *req.Name)
	}
	if req.KeyValue != nil {
		// Encrypt the new key value
		encrypted, err := utils.EncryptAPIKey(*req.KeyValue)
		if err != nil {
			writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to encrypt API key: %v", err))
			return
		}
		updates = append(updates, "key_value = ?")
		args = append(args, encrypted)
	}
	if req.IsActive != nil {
		updates = append(updates, "is_active = ?")
		args = append(args, *req.IsActive)
	}
	if req.EndpointURL != nil {
		updates = append(updates, "endpoint_url = ?")
		args = append(args, *req.EndpointURL)
	}
	if req.Metadata != nil {
		metadataJSON, _ := json.Marshal(req.Metadata)
		updates = append(updates, "metadata = ?")
		args = append(args, string(metadataJSON))
	}

	if len(updates) == 0 {
		writeBadRequestResponse(w, "No fields to update")
		return
	}

	// Always update updated_at
	updates = append(updates, "updated_at = ?")
	args = append(args, time.Now())
	args = append(args, keyID)

	query := fmt.Sprintf("UPDATE api_keys SET %s WHERE id = ?", joinStrings(updates, ", "))
	_, err = db.Exec(query, args...)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to update API key: %v", err))
		return
	}

	writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"message": "API key updated successfully",
		"id":      keyID,
	})
}

// joinStrings is a helper to join strings with a separator
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}

// handleDeleteAPIKey deletes an API key
func (s *Server) handleDeleteAPIKey(w http.ResponseWriter, r *http.Request) {
	if s.persistence == nil {
		writeErrorResponse(w, http.StatusServiceUnavailable, "API key management requires database")
		return
	}

	vars := mux.Vars(r)
	keyID := vars["id"]

	if keyID == "" {
		writeBadRequestResponse(w, "Key ID is required")
		return
	}

	db := s.persistence.GetDB()

	// Delete the API key
	result, err := db.Exec("DELETE FROM api_keys WHERE id = ?", keyID)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to delete API key: %v", err))
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		writeErrorResponse(w, http.StatusNotFound, "API key not found")
		return
	}

	utils.GetLogger().Info("API key deleted", utils.Component("settings"))

	writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"message": "API key deleted successfully",
		"id":      keyID,
	})
}

// handleTestAPIKey tests if an API key is valid
func (s *Server) handleTestAPIKey(w http.ResponseWriter, r *http.Request) {
	if s.persistence == nil {
		writeErrorResponse(w, http.StatusServiceUnavailable, "API key management requires database")
		return
	}

	vars := mux.Vars(r)
	keyID := vars["id"]

	if keyID == "" {
		writeBadRequestResponse(w, "Key ID is required")
		return
	}

	db := s.persistence.GetDB()

	// Get the API key
	var encryptedKey, provider string
	var isActive bool
	err := db.QueryRow("SELECT key_value, provider, is_active FROM api_keys WHERE id = ?", keyID).Scan(&encryptedKey, &provider, &isActive)
	if err != nil {
		writeErrorResponse(w, http.StatusNotFound, "API key not found")
		return
	}

	if !isActive {
		writeJSONResponse(w, http.StatusOK, map[string]interface{}{
			"success": false,
			"message": "API key is inactive",
		})
		return
	}

	// Decrypt the key
	_, err = utils.DecryptAPIKey(encryptedKey)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to decrypt API key - key may be corrupted")
		return
	}

	// For now, just verify we can decrypt it. In the future, make actual API test call to provider
	writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("API key is valid and can be decrypted (provider: %s)", provider),
	})
}

// handleCreateAPIKeyFromSettings creates a new API key for LLM provider credentials
func (s *Server) handleCreateAPIKeyFromSettings(w http.ResponseWriter, r *http.Request) {
	if s.persistence == nil {
		writeErrorResponse(w, http.StatusServiceUnavailable, "API key management requires database")
		return
	}

	var req struct {
		Provider    string                 `json:"provider"`               // openai, anthropic, ollama, etc.
		Name        string                 `json:"name"`                   // User-friendly name
		KeyValue    string                 `json:"key_value"`              // The actual API key
		EndpointURL string                 `json:"endpoint_url,omitempty"` // Custom endpoint (for Ollama, etc.)
		Metadata    map[string]interface{} `json:"metadata,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequestResponse(w, fmt.Sprintf("Invalid request: %v", err))
		return
	}

	// Validate required fields
	if req.Provider == "" {
		writeBadRequestResponse(w, "Provider is required (e.g., 'openai', 'anthropic', 'ollama')")
		return
	}
	if req.Name == "" {
		writeBadRequestResponse(w, "Name is required")
		return
	}
	if req.KeyValue == "" {
		writeBadRequestResponse(w, "Key value is required")
		return
	}

	// Generate a unique ID
	keyID := uuid.New().String()

	// Encrypt the API key
	encryptedKey, err := utils.EncryptAPIKey(req.KeyValue)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to encrypt API key: %v", err))
		return
	}

	// Prepare metadata JSON
	var metadataJSON string
	if req.Metadata != nil {
		metadataBytes, _ := json.Marshal(req.Metadata)
		metadataJSON = string(metadataBytes)
	}

	db := s.persistence.GetDB()

	// Insert into database
	query := `
		INSERT INTO api_keys (id, provider, name, key_value, endpoint_url, is_active, created_at, updated_at, metadata)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	now := time.Now()
	_, err = db.Exec(query, keyID, req.Provider, req.Name, encryptedKey, req.EndpointURL, true, now, now, metadataJSON)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create API key: %v", err))
		return
	}

	utils.GetLogger().Info("API key created", utils.Component("settings"))

	// Return the created key info (without the actual key value for security)
	writeJSONResponse(w, http.StatusCreated, map[string]interface{}{
		"id":           keyID,
		"provider":     req.Provider,
		"name":         req.Name,
		"endpoint_url": req.EndpointURL,
		"is_active":    true,
		"created_at":   now.Format(time.RFC3339),
		"message":      "API key created successfully",
	})
}

// ClearDataRequest represents the request to clear data
type ClearDataRequest struct {
	Target string `json:"target"`
}

// handleClearData handles data clearing requests (disabled for safety)
func (s *Server) handleClearData(w http.ResponseWriter, r *http.Request) {
	utils.GetLogger().Info("Data clear requested but disabled for safety", utils.Component("settings"))
	writeErrorResponse(w, http.StatusNotImplemented, "Data management is disabled for safety")
}
