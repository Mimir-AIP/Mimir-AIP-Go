package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
)

// MCPServer provides MCP-compatible tool access to plugins
type MCPServer struct {
	registry *pipelines.PluginRegistry
}

// NewMCPServer creates a new MCP server instance
func NewMCPServer(registry *pipelines.PluginRegistry) *MCPServer {
	return &MCPServer{
		registry: registry,
	}
}

// Initialize sets up the MCP tools based on available plugins
func (ms *MCPServer) Initialize() error {
	log.Printf("MCP server initialized with plugin registry")
	return nil
}

// ServeHTTP handles HTTP requests for the MCP server
func (ms *MCPServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Handle tool discovery requests
	if r.URL.Path == "/mcp/tools" && r.Method == "GET" {
		ms.handleToolDiscovery(w, r)
		return
	}

	// Handle tool execution requests
	if r.URL.Path == "/mcp/tools/execute" && r.Method == "POST" {
		ms.handleToolExecution(w, r)
		return
	}

	http.Error(w, "Not found", http.StatusNotFound)
}

// handleToolDiscovery returns available tools in MCP-compatible format
func (ms *MCPServer) handleToolDiscovery(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	tools := make([]map[string]any, 0)
	for pluginType, typePlugins := range ms.registry.GetAllPlugins() {
		for pluginName := range typePlugins {
			toolName := fmt.Sprintf("%s.%s", pluginType, pluginName)
			tools = append(tools, map[string]any{
				"name":        toolName,
				"description": fmt.Sprintf("%s plugin for Mimir AIP pipeline execution", toolName),
				"inputSchema": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"step_config": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"name": map[string]any{
									"type":        "string",
									"description": "Name of the pipeline step",
								},
								"config": map[string]any{
									"type":                 "object",
									"description":          "Configuration parameters for the plugin",
									"additionalProperties": true,
								},
								"output": map[string]any{
									"type":        "string",
									"description": "Output key for storing results in context",
								},
							},
							"required": []string{"name"},
						},
						"context": map[string]any{
							"type":                 "object",
							"description":          "Current pipeline context",
							"additionalProperties": true,
						},
					},
					"required": []string{"step_config"},
				},
			})
		}
	}

	json.NewEncoder(w).Encode(map[string]any{
		"tools": tools,
	})
}

// handleToolExecution executes a specific tool
func (ms *MCPServer) handleToolExecution(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req struct {
		ToolName  string         `json:"tool_name"`
		Arguments map[string]any `json:"arguments"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	// Parse tool name (e.g., "Input.api" -> type: "Input", name: "api")
	toolParts := strings.Split(req.ToolName, ".")
	if len(toolParts) != 2 {
		http.Error(w, "Invalid tool name format", http.StatusBadRequest)
		return
	}

	pluginType := toolParts[0]
	pluginName := toolParts[1]

	// Get the plugin
	plugin, err := ms.registry.GetPlugin(pluginType, pluginName)
	if err != nil {
		http.Error(w, fmt.Sprintf("Plugin not found: %v", err), http.StatusNotFound)
		return
	}

	// Parse arguments
	var params struct {
		StepConfig map[string]any `json:"step_config"`
		Context    map[string]any `json:"context"`
	}

	if argsJSON, err := json.Marshal(req.Arguments); err == nil {
		_ = json.Unmarshal(argsJSON, &params)
	}

	// Convert to plugin types
	stepConfig := pipelines.StepConfig{
		Name:   getStringValue(params.StepConfig, "name"),
		Plugin: req.ToolName,
		Config: getMapValue(params.StepConfig, "config"),
		Output: getStringValue(params.StepConfig, "output"),
	}

	globalContext := pipelines.NewPluginContext()
	if params.Context != nil {
		for k, v := range params.Context {
			globalContext.Set(k, v)
		}
	}

	// Execute the plugin
	result, err := plugin.ExecuteStep(context.Background(), stepConfig, globalContext)
	if err != nil {
		http.Error(w, fmt.Sprintf("Plugin execution failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Return result in MCP-compatible format
	response := map[string]any{
		"success": true,
		"result":  result,
	}

	json.NewEncoder(w).Encode(response)
}

// Helper functions
func getStringValue(data map[string]any, key string) string {
	if data == nil {
		return ""
	}
	if val, exists := data[key]; exists {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func getMapValue(data map[string]any, key string) map[string]any {
	if data == nil {
		return make(map[string]any)
	}
	if val, exists := data[key]; exists {
		if m, ok := val.(map[string]any); ok {
			return m
		}
	}
	return make(map[string]any)
}
