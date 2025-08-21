package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
	"github.com/Mimir-AIP/Mimir-AIP-Go/utils"
	"github.com/gorilla/mux"
)

// Server represents the Mimir AIP server
type Server struct {
	router    *mux.Router
	registry  *pipelines.PluginRegistry
	mcpServer *MCPServer
}

// PipelineExecutionRequest represents a request to execute a pipeline
type PipelineExecutionRequest struct {
	PipelineName string                 `json:"pipeline_name,omitempty"`
	PipelineFile string                 `json:"pipeline_file,omitempty"`
	Context      map[string]interface{} `json:"context,omitempty"`
}

// PipelineExecutionResponse represents the response from pipeline execution
type PipelineExecutionResponse struct {
	Success    bool                    `json:"success"`
	Error      string                  `json:"error,omitempty"`
	Context    pipelines.PluginContext `json:"context,omitempty"`
	ExecutedAt string                  `json:"executed_at"`
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
	}

	s.registerDefaultPlugins()
	s.setupRoutes()
	s.mcpServer.Initialize()

	return s
}

// registerDefaultPlugins registers the built-in plugins
func (s *Server) registerDefaultPlugins() {
	// Register real plugins
	apiPlugin := &utils.RealAPIPlugin{}
	htmlPlugin := &utils.MockHTMLPlugin{}

	s.registry.RegisterPlugin(apiPlugin)
	s.registry.RegisterPlugin(htmlPlugin)
}

// setupRoutes sets up the HTTP routes
func (s *Server) setupRoutes() {
	// Health check
	s.router.HandleFunc("/health", s.handleHealth).Methods("GET")

	// Pipeline execution
	s.router.HandleFunc("/api/v1/pipelines/execute", s.handleExecutePipeline).Methods("POST")

	// Pipeline management
	s.router.HandleFunc("/api/v1/pipelines", s.handleListPipelines).Methods("GET")
	s.router.HandleFunc("/api/v1/pipelines/{name}", s.handleGetPipeline).Methods("GET")

	// Plugin management
	s.router.HandleFunc("/api/v1/plugins", s.handleListPlugins).Methods("GET")
	s.router.HandleFunc("/api/v1/plugins/{type}", s.handleListPluginsByType).Methods("GET")
	s.router.HandleFunc("/api/v1/plugins/{type}/{name}", s.handleGetPlugin).Methods("GET")

	// Agentic features
	s.router.HandleFunc("/api/v1/agent/execute", s.handleAgentExecute).Methods("POST")

	// MCP endpoints
	s.router.PathPrefix("/mcp").Handler(s.mcpServer)
}

// Start starts the HTTP server
func (s *Server) Start(port string) error {
	log.Printf("Starting Mimir AIP server on port %s", port)
	return http.ListenAndServe(":"+port, s.router)
}

// handleHealth handles health check requests
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "healthy",
		"time":   time.Now().Format(time.RFC3339),
	})
}

// handleExecutePipeline handles pipeline execution requests
func (s *Server) handleExecutePipeline(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req PipelineExecutionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Determine pipeline to execute
	pipelineFile := req.PipelineFile
	if pipelineFile == "" && req.PipelineName != "" {
		// Try to find pipeline by name in config
		pipelineFile = fmt.Sprintf("pipelines/%s.yaml", req.PipelineName)
	}

	if pipelineFile == "" {
		http.Error(w, "Either pipeline_name or pipeline_file must be provided", http.StatusBadRequest)
		return
	}

	// Parse and execute pipeline
	config, err := utils.ParsePipeline(pipelineFile)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse pipeline: %v", err), http.StatusBadRequest)
		return
	}

	// Initialize context
	ctx := context.Background()
	globalContext := make(pipelines.PluginContext)
	if req.Context != nil {
		for k, v := range req.Context {
			globalContext[k] = v
		}
	}

	// Execute pipeline
	result, err := utils.ExecutePipeline(ctx, config)
	if err != nil {
		http.Error(w, fmt.Sprintf("Pipeline execution failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Return response
	response := PipelineExecutionResponse{
		Success:    result.Success,
		Error:      result.Error,
		Context:    result.Context,
		ExecutedAt: time.Now().Format(time.RFC3339),
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// handleListPipelines handles requests to list all pipelines
func (s *Server) handleListPipelines(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Try to read from config.yaml
	configPath := "config.yaml"
	pipelines, err := utils.ParseAllPipelines(configPath)
	if err != nil {
		// If config.yaml doesn't exist, return empty array
		json.NewEncoder(w).Encode([]utils.PipelineConfig{})
		return
	}

	json.NewEncoder(w).Encode(pipelines)
}

// handleGetPipeline handles requests to get a specific pipeline
func (s *Server) handleGetPipeline(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	pipelineName := vars["name"]

	// Try to find pipeline in config
	configPath := "config.yaml"
	pipelines, err := utils.ParseAllPipelines(configPath)
	if err != nil {
		http.Error(w, "Failed to read pipelines", http.StatusInternalServerError)
		return
	}

	for _, pipeline := range pipelines {
		if pipeline.Name == pipelineName {
			json.NewEncoder(w).Encode(pipeline)
			return
		}
	}

	http.Error(w, "Pipeline not found", http.StatusNotFound)
}

// handleListPlugins handles requests to list all plugins
func (s *Server) handleListPlugins(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var plugins []PluginInfo
	for pluginType, typePlugins := range s.registry.GetAllPlugins() {
		for pluginName := range typePlugins {
			plugins = append(plugins, PluginInfo{
				Type:        pluginType,
				Name:        pluginName,
				Description: fmt.Sprintf("%s plugin", pluginName),
			})
		}
	}

	json.NewEncoder(w).Encode(plugins)
}

// handleListPluginsByType handles requests to list plugins of a specific type
func (s *Server) handleListPluginsByType(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	pluginType := vars["type"]

	plugins := s.registry.GetPluginsByType(pluginType)
	var pluginInfos []PluginInfo
	for pluginName := range plugins {
		pluginInfos = append(pluginInfos, PluginInfo{
			Type:        pluginType,
			Name:        pluginName,
			Description: fmt.Sprintf("%s plugin", pluginName),
		})
	}

	json.NewEncoder(w).Encode(pluginInfos)
}

// handleGetPlugin handles requests to get information about a specific plugin
func (s *Server) handleGetPlugin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	pluginType := vars["type"]
	pluginName := vars["name"]

	_, err := s.registry.GetPlugin(pluginType, pluginName)
	if err != nil {
		http.Error(w, "Plugin not found", http.StatusNotFound)
		return
	}

	pluginInfo := PluginInfo{
		Type:        pluginType,
		Name:        pluginName,
		Description: fmt.Sprintf("%s plugin", pluginName),
	}

	json.NewEncoder(w).Encode(pluginInfo)
}

// handleAgentExecute handles agentic execution requests (placeholder for now)
func (s *Server) handleAgentExecute(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// This will be implemented when we add LLM integration
	response := map[string]interface{}{
		"message": "Agent execution not yet implemented",
		"status":  "pending",
	}

	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(response)
}
