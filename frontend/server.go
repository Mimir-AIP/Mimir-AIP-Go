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

	// Handle API requests
	http.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Proxying API request: %s %s", r.Method, r.URL.Path)
		proxy.ServeHTTP(w, r)
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Proxying health request: %s %s", r.Method, r.URL.Path)
		proxy.ServeHTTP(w, r)
	})

	http.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Proxying ready request: %s %s", r.Method, r.URL.Path)
		proxy.ServeHTTP(w, r)
	})

	// Serve static files from the current directory
	fs := http.FileServer(http.Dir("."))
	http.Handle("/", fs)

	log.Printf("Frontend server starting on port %s", port)
	log.Printf("Proxying API requests to: %s", apiURL)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
