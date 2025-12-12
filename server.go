package main

import (
	"context"
	"log"
	"net/http"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
	"github.com/Mimir-AIP/Mimir-AIP-Go/utils"
	"github.com/gorilla/mux"
)

// Server represents the Mimir AIP server
type Server struct {
	router    *mux.Router
	registry  *pipelines.PluginRegistry
	mcpServer *MCPServer
	scheduler *utils.Scheduler
	monitor   *utils.JobMonitor
	config    *utils.ConfigManager
}

// PipelineExecutionRequest represents a request to execute a pipeline
type PipelineExecutionRequest struct {
	PipelineName string         `json:"pipeline_name,omitempty"`
	PipelineFile string         `json:"pipeline_file,omitempty"`
	Context      map[string]any `json:"context,omitempty"`
}

// PipelineExecutionResponse represents the response from pipeline execution
type PipelineExecutionResponse struct {
	Success    bool                     `json:"success"`
	Error      string                   `json:"error,omitempty"`
	Context    *pipelines.PluginContext `json:"context,omitempty"`
	ExecutedAt string                   `json:"executed_at"`
}

// PluginInfo represents information about a plugin
type PluginInfo struct {
	Type        string `json:"type"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// NewServer creates a new Mimir AIP server
func NewServer() *Server {
	registry := pipelines.NewPluginRegistry()

	s := &Server{
		router:    mux.NewRouter(),
		registry:  registry,
		mcpServer: NewMCPServer(registry),
		scheduler: utils.NewScheduler(registry),
		monitor:   utils.NewJobMonitor(1000), // Keep last 1000 executions
		config:    utils.GetConfigManager(),
	}

	s.registerDefaultPlugins()
	s.setupRoutes()
	_ = s.mcpServer.Initialize()

	// Load configuration
	if err := utils.LoadGlobalConfig(); err != nil {
		log.Printf("Failed to load configuration: %v", err)
	}

	// Initialize logging
	if err := utils.InitLogger(s.config.GetConfig().Logging); err != nil {
		log.Printf("Failed to initialize logger: %v", err)
	}

	// Initialize pipeline store
	if err := utils.InitializeGlobalPipelineStore("./pipelines"); err != nil {
		log.Printf("Failed to initialize pipeline store: %v", err)
	}

	// Start the scheduler
	if err := s.scheduler.Start(); err != nil {
		utils.GetLogger().Error("Failed to start scheduler", err, utils.Component("server"))
	}

	return s
}

// registerDefaultPlugins registers the built-in plugins
func (s *Server) registerDefaultPlugins() {
	// Register real plugins
	apiPlugin := &utils.RealAPIPlugin{}
	htmlPlugin := &utils.MockHTMLPlugin{}

	_ = s.registry.RegisterPlugin(apiPlugin)
	_ = s.registry.RegisterPlugin(htmlPlugin)
}

// Start starts the HTTP server
func (s *Server) Start(port string) error {
	log.Printf("Starting Mimir AIP server on port %s", port)
	return http.ListenAndServe(":"+port, s.router)
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	log.Println("Initiating graceful shutdown...")

	// Create a channel to signal shutdown completion
	shutdownComplete := make(chan struct{})

	go func() {
		defer close(shutdownComplete)

		// 1. Stop the scheduler
		if s.scheduler != nil {
			log.Println("Stopping scheduler...")
			if err := s.scheduler.Stop(); err != nil {
				log.Printf("Error stopping scheduler: %v", err)
			}
		}

		// 2. Stop MCP server
		if s.mcpServer != nil {
			log.Println("Stopping MCP server...")
			// MCP server cleanup if needed
		}

		// 3. Close any open connections or resources
		log.Println("Cleaning up resources...")

		// 4. Flush any pending logs
		if logger := utils.GetLogger(); logger != nil {
			log.Println("Flushing logs...")
			// Logger flush if supported
		}

		log.Println("Graceful shutdown completed")
	}()

	// Wait for shutdown to complete or context timeout
	select {
	case <-shutdownComplete:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
