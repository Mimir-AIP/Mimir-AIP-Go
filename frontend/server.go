package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	apiURL := os.Getenv("API_URL")
	if apiURL == "" {
		apiURL = "http://localhost:8080"
	}

	// Parse the orchestrator URL
	orchestratorURL, err := url.Parse(apiURL)
	if err != nil {
		log.Fatalf("Failed to parse API_URL: %v", err)
	}

	// Create reverse proxy for API requests
	proxy := httputil.NewSingleHostReverseProxy(orchestratorURL)
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("Frontend proxy error for %s %s: %v", r.Method, r.URL.Path, err)
		http.Error(w, "Upstream API unavailable", http.StatusBadGateway)
	}
	proxy.Director = func(req *http.Request) {
		req.URL.Scheme = orchestratorURL.Scheme
		req.URL.Host = orchestratorURL.Host
		if orchestratorURL.Path != "" && orchestratorURL.Path != "/" {
			req.URL.Path = joinURLPath(orchestratorURL.Path, req.URL.Path)
		}
		req.Host = orchestratorURL.Host
		if _, ok := req.Header["X-Forwarded-Host"]; !ok {
			req.Header.Set("X-Forwarded-Host", req.Host)
		}
		if req.Header.Get("X-Forwarded-Proto") == "" {
			if req.TLS != nil {
				req.Header.Set("X-Forwarded-Proto", "https")
			} else {
				req.Header.Set("X-Forwarded-Proto", "http")
			}
		}
	}
	// Handle proxied API and realtime endpoints.
	for _, path := range []string{"/api/", "/health", "/ready", "/openapi.yaml", "/ws/tasks"} {
		http.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			log.Printf("Proxying request: %s %s", r.Method, r.URL.Path)
			proxy.ServeHTTP(w, r)
		})
	}

	// Serve static files from the current directory
	fs := http.FileServer(http.Dir("."))
	http.Handle("/", fs)

	log.Printf("Frontend server starting on port %s", port)
	log.Printf("Proxying API requests to: %s", apiURL)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func joinURLPath(basePath, requestPath string) string {
	switch {
	case basePath == "" || basePath == "/":
		return requestPath
	case requestPath == "":
		return basePath
	case basePath[len(basePath)-1] == '/' && requestPath[0] == '/':
		return basePath[:len(basePath)-1] + requestPath
	case basePath[len(basePath)-1] != '/' && requestPath[0] != '/':
		return basePath + "/" + requestPath
	default:
		return basePath + requestPath
	}
}
