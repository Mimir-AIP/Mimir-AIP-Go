// Entry point for Mimir AIP CLI
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/Mimir-AIP/Mimir-AIP-Go/utils"
	"github.com/rs/cors"
)

const mimirVersion = "v0.0.1"

// TODO- Add a endpoint to the server to return the version
func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		// No arguments: parse config.yaml for enabled pipelines
		configPath := filepath.Join(".", "config.yaml")
		pipelines, err := utils.GetEnabledPipelines(configPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading config.yaml: %v\n", err)
			os.Exit(1)
		}
		for _, pipeline := range pipelines {
			runPipelineWithParseAndName(pipeline)
		}
		return
	}

	switch args[0] {
	case "-h", "--help", "help":
		printHelp()
		return
	case "--version", "-v":
		fmt.Println("Mimir version:", mimirVersion)
		return
	case "--pipeline":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Error: --pipeline requires a pipeline name or file path")
			os.Exit(1)
		}
		runPipelineWithParseAndName(args[1])
		return
	case "--server":
		port := "8080"
		if len(args) > 1 {
			port = args[1]
		}
		runServer(port)
		return
	default:
		fmt.Fprintln(os.Stderr, "Unknown argument. Use --help for usage.")
		os.Exit(1)
	}
}

func runPipelineWithParseAndName(pipeline string) {
	// Parse the pipeline before running
	if _, err := utils.ParsePipeline(pipeline); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing pipeline %s: %v\n", pipeline, err)
		return
	}
	// Try to get pipeline name from YAML
	name, nameErr := utils.GetPipelineName(pipeline)
	displayName := pipeline
	if nameErr == nil && name != "" {
		displayName = name
	}
	if err := utils.RunPipeline(pipeline); err != nil {
		fmt.Fprintf(os.Stderr, "Error running pipeline %s: %v\n", displayName, err)
	}
}

func runServer(port string) {
	server := NewServer()

	// Create HTTP server with proper configuration
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000", "http://localhost:8080", "*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})

	httpServer := &http.Server{
		Addr:         ":" + port,
		Handler:      c.Handler(server.router),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Starting Mimir AIP server on port %s", port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Error starting server: %v", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	<-quit
	log.Println("Shutting down server...")

	// Give outstanding requests 30 seconds to complete
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("HTTP server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

func printHelp() { // TODO look into using a TUI framework, will keep things modular for now to aid later refactoring if I go with that route
	fmt.Println("Usage:")
	fmt.Println("  --pipeline <pipeline name/file path>   Run specified pipeline")
	fmt.Println("  --server [port]                        Start HTTP server (default port: 8080)")
	fmt.Println("  (no arguments)                        Run enabled pipelines from config.yaml")
	fmt.Println("  -h, --help, help                      Show this help message")
	fmt.Println("  -v, --version                        Show Mimir version")
}
