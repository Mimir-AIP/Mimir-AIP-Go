package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/Mimir-AIP/Mimir-AIP-Go/utils"
	"github.com/gorilla/mux"
)

// loggingMiddleware logs HTTP requests and responses
func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a response writer wrapper to capture status code
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Generate request ID
		requestID := fmt.Sprintf("%d", time.Now().UnixNano())

		// Add request ID to context
		ctx := context.WithValue(r.Context(), "request_id", requestID)
		r = r.WithContext(ctx)

		// Log request
		utils.GetLogger().Info("HTTP Request",
			utils.String("method", r.Method),
			utils.String("path", r.URL.Path),
			utils.String("remote_addr", r.RemoteAddr),
			utils.String("user_agent", r.Header.Get("User-Agent")),
			utils.RequestID(requestID),
			utils.Component("http"))

		// Call next handler
		next.ServeHTTP(rw, r)

		// Log response
		duration := time.Since(start)
		utils.GetLogger().Info("HTTP Response",
			utils.String("method", r.Method),
			utils.String("path", r.URL.Path),
			utils.Int("status", rw.statusCode),
			utils.Float("duration_ms", duration.Seconds()*1000),
			utils.RequestID(requestID),
			utils.Component("http"))
	})
}

// errorRecoveryMiddleware recovers from panics and logs errors
func (s *Server) errorRecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				// Log the panic
				utils.GetLogger().Error("Panic recovered",
					fmt.Errorf("panic: %v", err),
					utils.String("method", r.Method),
					utils.String("path", r.URL.Path),
					utils.Component("http"))

				// Return 500 error
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// versionMiddleware adds API version information to requests
func (s *Server) versionMiddleware(version string) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Add version to request context
			ctx := context.WithValue(r.Context(), "api_version", version)
			r = r.WithContext(ctx)

			// Add version header to response
			w.Header().Set("X-API-Version", version)

			next.ServeHTTP(w, r)
		})
	}
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
