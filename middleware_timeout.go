package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/Mimir-AIP/Mimir-AIP-Go/utils"
)

// TimeoutMiddleware wraps HTTP handlers with a timeout to prevent indefinite hangs
// This is critical for preventing API lockups during high load or slow operations
func TimeoutMiddleware(timeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logger := utils.GetLogger()

			// Create a context with timeout
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()

			// Replace request context with timeout context
			r = r.WithContext(ctx)

			// Channel to signal completion
			done := make(chan struct{})

			// Run handler in goroutine
			go func() {
				defer func() {
					if rec := recover(); rec != nil {
						logger.Error("Panic in HTTP handler",
							fmt.Errorf("panic: %v", rec),
							utils.String("path", r.URL.Path),
							utils.Component("middleware"))
					}
					close(done)
				}()
				next.ServeHTTP(w, r)
			}()

			// Wait for completion or timeout
			select {
			case <-done:
				// Handler completed successfully
				return
			case <-ctx.Done():
				// Timeout occurred
				logger.Warn("Request timeout",
					utils.String("path", r.URL.Path),
					utils.String("method", r.Method),
					utils.String("timeout", timeout.String()),
					utils.Component("middleware"))

				// Only write error if headers haven't been sent yet
				if w.Header().Get("Content-Type") == "" {
					writeErrorResponse(w, http.StatusGatewayTimeout,
						"Request timeout - operation took too long")
				}
				return
			}
		})
	}
}

// APITimeoutMiddleware applies different timeouts for different API routes
func APITimeoutMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Determine timeout based on route
			timeout := 30 * time.Second // Default timeout

			// Longer timeout for operations that query large datasets or run simulations
			if r.URL.Path == "/api/v1/twins" ||
				r.URL.Path == "/api/v1/ontologies" ||
				r.Method == "POST" && (r.URL.Path == "/api/v1/twins/create" ||
					r.URL.Path == "/api/v1/simulations/run") {
				timeout = 60 * time.Second
			}

			// Even longer for batch operations
			if r.Method == "POST" && r.URL.Path == "/api/v1/extraction/batch" {
				timeout = 120 * time.Second
			}

			// Apply timeout
			TimeoutMiddleware(timeout)(next).ServeHTTP(w, r)
		})
	}
}
