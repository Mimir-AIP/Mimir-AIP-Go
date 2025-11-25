package main

import (
	"encoding/json"
	"net/http"
)

// Response helpers for common HTTP response patterns

// writeJSONResponse writes a JSON response with the given status code
func writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// writeErrorResponse writes an error response with the given status code and message
func writeErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error":  message,
		"status": "error",
	})
}

// writeSuccessResponse writes a success response with the given data
func writeSuccessResponse(w http.ResponseWriter, data interface{}) {
	writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    data,
	})
}

// writeCreatedResponse writes a 201 Created response with the given data
func writeCreatedResponse(w http.ResponseWriter, data interface{}) {
	writeJSONResponse(w, http.StatusCreated, map[string]interface{}{
		"success": true,
		"data":    data,
	})
}

// writeNotFoundResponse writes a 404 Not Found response
func writeNotFoundResponse(w http.ResponseWriter, resource string) {
	writeErrorResponse(w, http.StatusNotFound, resource+" not found")
}

// writeBadRequestResponse writes a 400 Bad Request response
func writeBadRequestResponse(w http.ResponseWriter, message string) {
	writeErrorResponse(w, http.StatusBadRequest, message)
}

// writeUnauthorizedResponse writes a 401 Unauthorized response
func writeUnauthorizedResponse(w http.ResponseWriter, message string) {
	if message == "" {
		message = "Unauthorized"
	}
	writeErrorResponse(w, http.StatusUnauthorized, message)
}

// writeInternalServerErrorResponse writes a 500 Internal Server Error response
func writeInternalServerErrorResponse(w http.ResponseWriter, message string) {
	if message == "" {
		message = "Internal Server Error"
	}
	writeErrorResponse(w, http.StatusInternalServerError, message)
}

// writeNotImplementedResponse writes a 501 Not Implemented response
func writeNotImplementedResponse(w http.ResponseWriter, message string) {
	if message == "" {
		message = "Not Implemented"
	}
	writeErrorResponse(w, http.StatusNotImplemented, message)
}
