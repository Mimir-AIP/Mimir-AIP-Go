package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// Response helpers for common HTTP response patterns

// writeJSONResponse writes a JSON response with the given status code
func writeJSONResponse(w http.ResponseWriter, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// writeErrorResponse writes an error response with the given status code and message
func writeErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]any{
		"error":  message,
		"status": "error",
	})
}

// writeSuccessResponse writes a success response with the given data
func writeSuccessResponse(w http.ResponseWriter, data any) {
	writeJSONResponse(w, http.StatusOK, map[string]any{
		"success": true,
		"data":    data,
	})
}

// writeBadRequestResponse writes a 400 Bad Request response
func writeBadRequestResponse(w http.ResponseWriter, message string) {
	writeErrorResponse(w, http.StatusBadRequest, message)
}

// writeInternalServerErrorResponse writes a 500 Internal Server Error response
func writeInternalServerErrorResponse(w http.ResponseWriter, message string) {
	if message == "" {
		message = "Internal Server Error"
	}
	writeErrorResponse(w, http.StatusInternalServerError, message)
}

// writeOperationSuccessResponse writes a success response for CRUD operations
func writeOperationSuccessResponse(w http.ResponseWriter, message, idKey, idValue string) {
	writeJSONResponse(w, http.StatusOK, map[string]any{
		"message": message,
		idKey:     idValue,
	})
}

// parseLimit extracts and validates a limit parameter from the request, returning default if invalid
func parseLimit(r *http.Request, defaultLimit int) int {
	limitParam := r.URL.Query().Get("limit")
	if limitParam == "" {
		return defaultLimit
	}

	var limit int
	if n, err := fmt.Sscanf(limitParam, "%d", &limit); err == nil && n == 1 && limit > 0 {
		return limit
	}
	return defaultLimit
}
