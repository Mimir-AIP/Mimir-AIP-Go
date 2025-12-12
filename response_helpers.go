package main

import (
	"encoding/json"
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
