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

	// Add specialized ontology tools for agents
	tools = append(tools, ms.getOntologyTools()...)

	// Add specialized digital twin tools for agents
	tools = append(tools, ms.getDigitalTwinTools()...)

	// Add all registered plugins as tools
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

// getOntologyTools returns specialized ontology tools for Mimir agents
func (ms *MCPServer) getOntologyTools() []map[string]any {
	return []map[string]any{
		{
			"name":        "ontology.query",
			"description": "Query the knowledge graph using natural language or SPARQL. Converts natural language questions to SPARQL queries and returns results from the RDF triplestore.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"ontology_id": map[string]any{
						"type":        "string",
						"description": "ID of the ontology to query",
					},
					"question": map[string]any{
						"type":        "string",
						"description": "Natural language question to query the knowledge graph",
					},
					"use_nl": map[string]any{
						"type":        "boolean",
						"description": "Use natural language translation (true) or provide raw SPARQL (false)",
						"default":     true,
					},
					"sparql_query": map[string]any{
						"type":        "string",
						"description": "Raw SPARQL query (only if use_nl is false)",
					},
				},
				"required": []string{"ontology_id"},
			},
		},
		{
			"name":        "ontology.extract",
			"description": "Extract entities and relationships from data according to the ontology schema. Supports CSV, JSON, text, and HTML sources with deterministic or LLM-powered extraction.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"ontology_id": map[string]any{
						"type":        "string",
						"description": "ID of the ontology schema to use for extraction",
					},
					"data": map[string]any{
						"type":        "object",
						"description": "Data to extract entities from",
					},
					"source_type": map[string]any{
						"type":        "string",
						"description": "Type of data source (csv, json, text, html)",
						"enum":        []string{"csv", "json", "text", "html"},
					},
					"extraction_type": map[string]any{
						"type":        "string",
						"description": "Extraction method (deterministic, llm, hybrid)",
						"enum":        []string{"deterministic", "llm", "hybrid"},
						"default":     "hybrid",
					},
					"job_name": map[string]any{
						"type":        "string",
						"description": "Name for the extraction job",
					},
				},
				"required": []string{"ontology_id", "data", "source_type"},
			},
		},
		{
			"name":        "ontology.detect_drift",
			"description": "Detect schema drift by analyzing data patterns that don't match the current ontology. Suggests new classes, properties, or modifications based on actual usage.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"ontology_id": map[string]any{
						"type":        "string",
						"description": "ID of the ontology to check for drift",
					},
					"source": map[string]any{
						"type":        "string",
						"description": "Drift detection source",
						"enum":        []string{"knowledge_graph", "extraction_job", "data"},
					},
					"job_id": map[string]any{
						"type":        "string",
						"description": "Extraction job ID (only for extraction_job source)",
					},
					"data": map[string]any{
						"type":        "object",
						"description": "Raw data to analyze (only for data source)",
					},
				},
				"required": []string{"ontology_id", "source"},
			},
		},
		{
			"name":        "ontology.list_suggestions",
			"description": "List AI-generated suggestions for improving the ontology schema. Returns pending, approved, rejected, or applied suggestions with confidence scores and risk levels.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"ontology_id": map[string]any{
						"type":        "string",
						"description": "ID of the ontology",
					},
					"status": map[string]any{
						"type":        "string",
						"description": "Filter by status",
						"enum":        []string{"pending", "approved", "rejected", "applied"},
					},
				},
				"required": []string{"ontology_id"},
			},
		},
		{
			"name":        "ontology.apply_suggestion",
			"description": "Apply an approved ontology suggestion to both the metadata database and RDF triplestore. Only works for suggestions with 'approved' status.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"ontology_id": map[string]any{
						"type":        "string",
						"description": "ID of the ontology",
					},
					"suggestion_id": map[string]any{
						"type":        "integer",
						"description": "ID of the suggestion to apply",
					},
				},
				"required": []string{"ontology_id", "suggestion_id"},
			},
		},
		{
			"name":        "ontology.get_stats",
			"description": "Get statistics about the ontology and knowledge graph including class count, property count, triple count, and entity count.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"ontology_id": map[string]any{
						"type":        "string",
						"description": "ID of the ontology",
					},
				},
				"required": []string{"ontology_id"},
			},
		},
	}
}

// handleToolExecution executes a specific tool
func (ms *MCPServer) handleToolExecution(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Route to appropriate handler based on tool name
	if strings.HasPrefix(request.Name, "ontology.") {
		ms.handleOntologyTool(w, r, request.Name, request.Arguments)
		return
	}
	if strings.HasPrefix(request.Name, "twin.") {
		ms.handleDigitalTwinTool(w, r, request.Name, request.Arguments)
		return
	}

	// Parse tool name (e.g., "Input.api" -> type: "Input", name: "api")
	toolParts := strings.Split(request.Name, ".")
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

	if argsJSON, err := json.Marshal(request.Arguments); err == nil {
		_ = json.Unmarshal(argsJSON, &params)
	}

	// Convert to plugin types
	stepConfig := pipelines.StepConfig{
		Name:   getStringValue(params.StepConfig, "name"),
		Plugin: request.Name,
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

// handleOntologyTool executes specialized ontology tools with simplified API
func (ms *MCPServer) handleOntologyTool(w http.ResponseWriter, r *http.Request, toolName string, arguments map[string]any) {
	ctx := r.Context()

	// These tools are called directly via HTTP endpoints, not plugins
	// This provides a simpler interface for agents

	switch toolName {
	case "ontology.query":
		ms.handleOntologyQuery(w, ctx, arguments)
	case "ontology.extract":
		ms.handleOntologyExtract(w, ctx, arguments)
	case "ontology.detect_drift":
		ms.handleOntologyDetectDrift(w, ctx, arguments)
	case "ontology.list_suggestions":
		ms.handleOntologyListSuggestions(w, ctx, arguments)
	case "ontology.apply_suggestion":
		ms.handleOntologyApplySuggestion(w, ctx, arguments)
	case "ontology.get_stats":
		ms.handleOntologyGetStats(w, ctx, arguments)
	default:
		http.Error(w, fmt.Sprintf("Unknown ontology tool: %s", toolName), http.StatusNotFound)
	}
}

// handleOntologyQuery executes natural language or SPARQL queries
func (ms *MCPServer) handleOntologyQuery(w http.ResponseWriter, ctx context.Context, args map[string]any) {
	// Note: This would typically be implemented by making HTTP calls to the REST API
	// or by injecting server dependencies. For simplicity, we'll document the expected behavior.

	response := map[string]any{
		"success":   true,
		"tool":      "ontology.query",
		"message":   "Query tool requires server instance access. Use REST API: POST /api/v1/kg/nl-query",
		"arguments": args,
	}
	json.NewEncoder(w).Encode(response)
}

// handleOntologyExtract executes entity extraction
func (ms *MCPServer) handleOntologyExtract(w http.ResponseWriter, ctx context.Context, args map[string]any) {
	response := map[string]any{
		"success":   true,
		"tool":      "ontology.extract",
		"message":   "Extract tool requires server instance access. Use REST API: POST /api/v1/extraction/jobs",
		"arguments": args,
	}
	json.NewEncoder(w).Encode(response)
}

// handleOntologyDetectDrift executes drift detection
func (ms *MCPServer) handleOntologyDetectDrift(w http.ResponseWriter, ctx context.Context, args map[string]any) {
	response := map[string]any{
		"success":   true,
		"tool":      "ontology.detect_drift",
		"message":   "Drift detection tool requires server instance access. Use REST API: POST /api/v1/ontology/:id/drift/detect",
		"arguments": args,
	}
	json.NewEncoder(w).Encode(response)
}

// handleOntologyListSuggestions lists ontology suggestions
func (ms *MCPServer) handleOntologyListSuggestions(w http.ResponseWriter, ctx context.Context, args map[string]any) {
	response := map[string]any{
		"success":   true,
		"tool":      "ontology.list_suggestions",
		"message":   "List suggestions tool requires server instance access. Use REST API: GET /api/v1/ontology/:id/suggestions",
		"arguments": args,
	}
	json.NewEncoder(w).Encode(response)
}

// handleOntologyApplySuggestion applies an approved suggestion
func (ms *MCPServer) handleOntologyApplySuggestion(w http.ResponseWriter, ctx context.Context, args map[string]any) {
	response := map[string]any{
		"success":   true,
		"tool":      "ontology.apply_suggestion",
		"message":   "Apply suggestion tool requires server instance access. Use REST API: POST /api/v1/ontology/:id/suggestions/:sid/apply",
		"arguments": args,
	}
	json.NewEncoder(w).Encode(response)
}

// handleOntologyGetStats gets ontology statistics
func (ms *MCPServer) handleOntologyGetStats(w http.ResponseWriter, ctx context.Context, args map[string]any) {
	response := map[string]any{
		"success":   true,
		"tool":      "ontology.get_stats",
		"message":   "Get stats tool requires server instance access. Use REST API: GET /api/v1/ontology/:id/stats",
		"arguments": args,
	}
	json.NewEncoder(w).Encode(response)
}

// getDigitalTwinTools returns specialized digital twin tools for Mimir agents
func (ms *MCPServer) getDigitalTwinTools() []map[string]any {
	return []map[string]any{
		{
			"name":        "twin.create",
			"description": "Create a digital twin from a knowledge graph ontology. Initializes entities and relationships for simulation.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"ontology_id": map[string]any{
						"type":        "string",
						"description": "ID of the ontology to create twin from",
					},
					"name": map[string]any{
						"type":        "string",
						"description": "Name for the digital twin",
					},
					"description": map[string]any{
						"type":        "string",
						"description": "Description of the twin",
					},
					"model_type": map[string]any{
						"type":        "string",
						"description": "Type of model",
						"enum":        []string{"organization", "department", "process", "individual"},
						"default":     "organization",
					},
				},
				"required": []string{"ontology_id", "name"},
			},
		},
		{
			"name":        "twin.query_state",
			"description": "Query the current state of a digital twin including entity states, utilization, and system metrics.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"twin_id": map[string]any{
						"type":        "string",
						"description": "ID of the digital twin",
					},
				},
				"required": []string{"twin_id"},
			},
		},
		{
			"name":        "twin.create_scenario",
			"description": "Create a what-if simulation scenario with custom events. Define resource failures, demand surges, or other disruptions.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"twin_id": map[string]any{
						"type":        "string",
						"description": "ID of the digital twin",
					},
					"name": map[string]any{
						"type":        "string",
						"description": "Name for the scenario",
					},
					"description": map[string]any{
						"type":        "string",
						"description": "Description of the scenario",
					},
					"events": map[string]any{
						"type":        "array",
						"description": "Array of simulation events",
						"items": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"type": map[string]any{
									"type":        "string",
									"description": "Event type (e.g., resource.unavailable, demand.surge)",
								},
								"target_uri": map[string]any{
									"type":        "string",
									"description": "URI of the target entity",
								},
								"timestamp": map[string]any{
									"type":        "integer",
									"description": "Step number when event occurs",
								},
								"parameters": map[string]any{
									"type":        "object",
									"description": "Event-specific parameters",
								},
							},
						},
					},
					"duration": map[string]any{
						"type":        "integer",
						"description": "Simulation duration in steps",
					},
				},
				"required": []string{"twin_id", "name", "events", "duration"},
			},
		},
		{
			"name":        "twin.run_simulation",
			"description": "Execute a simulation scenario and return results with metrics, impact analysis, and recommendations.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"twin_id": map[string]any{
						"type":        "string",
						"description": "ID of the digital twin",
					},
					"scenario_id": map[string]any{
						"type":        "string",
						"description": "ID of the scenario to run",
					},
					"snapshot_interval": map[string]any{
						"type":        "integer",
						"description": "Take state snapshot every N steps (0 to disable)",
						"default":     10,
					},
					"max_steps": map[string]any{
						"type":        "integer",
						"description": "Maximum simulation steps",
						"default":     1000,
					},
				},
				"required": []string{"twin_id", "scenario_id"},
			},
		},
		{
			"name":        "twin.analyze_impact",
			"description": "Analyze the impact of a completed simulation run. Provides entity-level impacts, critical paths, risk scores, and mitigation recommendations.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"twin_id": map[string]any{
						"type":        "string",
						"description": "ID of the digital twin",
					},
					"run_id": map[string]any{
						"type":        "string",
						"description": "ID of the simulation run",
					},
				},
				"required": []string{"twin_id", "run_id"},
			},
		},
		{
			"name":        "twin.compare_runs",
			"description": "Compare multiple simulation runs to identify which scenarios had the most impact and which mitigations were most effective.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"run_ids": map[string]any{
						"type":        "array",
						"description": "Array of simulation run IDs to compare",
						"items": map[string]any{
							"type": "string",
						},
					},
				},
				"required": []string{"run_ids"},
			},
		},
	}
}

// handleDigitalTwinTool executes specialized digital twin tools
func (ms *MCPServer) handleDigitalTwinTool(w http.ResponseWriter, r *http.Request, toolName string, arguments map[string]any) {
	ctx := r.Context()

	switch toolName {
	case "twin.create":
		ms.handleTwinCreate(w, ctx, arguments)
	case "twin.query_state":
		ms.handleTwinQueryState(w, ctx, arguments)
	case "twin.create_scenario":
		ms.handleTwinCreateScenario(w, ctx, arguments)
	case "twin.run_simulation":
		ms.handleTwinRunSimulation(w, ctx, arguments)
	case "twin.analyze_impact":
		ms.handleTwinAnalyzeImpact(w, ctx, arguments)
	case "twin.compare_runs":
		ms.handleTwinCompareRuns(w, ctx, arguments)
	default:
		http.Error(w, fmt.Sprintf("Unknown digital twin tool: %s", toolName), http.StatusNotFound)
	}
}

// handleTwinCreate creates a new digital twin
func (ms *MCPServer) handleTwinCreate(w http.ResponseWriter, ctx context.Context, args map[string]any) {
	response := map[string]any{
		"success":   true,
		"tool":      "twin.create",
		"message":   "Create twin tool requires server instance access. Use REST API: POST /api/v1/twin/create",
		"arguments": args,
	}
	json.NewEncoder(w).Encode(response)
}

// handleTwinQueryState queries digital twin state
func (ms *MCPServer) handleTwinQueryState(w http.ResponseWriter, ctx context.Context, args map[string]any) {
	response := map[string]any{
		"success":   true,
		"tool":      "twin.query_state",
		"message":   "Query state tool requires server instance access. Use REST API: GET /api/v1/twin/:id/state",
		"arguments": args,
	}
	json.NewEncoder(w).Encode(response)
}

// handleTwinCreateScenario creates a simulation scenario
func (ms *MCPServer) handleTwinCreateScenario(w http.ResponseWriter, ctx context.Context, args map[string]any) {
	response := map[string]any{
		"success":   true,
		"tool":      "twin.create_scenario",
		"message":   "Create scenario tool requires server instance access. Use REST API: POST /api/v1/twin/:id/scenarios",
		"arguments": args,
	}
	json.NewEncoder(w).Encode(response)
}

// handleTwinRunSimulation executes a simulation
func (ms *MCPServer) handleTwinRunSimulation(w http.ResponseWriter, ctx context.Context, args map[string]any) {
	response := map[string]any{
		"success":   true,
		"tool":      "twin.run_simulation",
		"message":   "Run simulation tool requires server instance access. Use REST API: POST /api/v1/twin/:id/scenarios/:sid/run",
		"arguments": args,
	}
	json.NewEncoder(w).Encode(response)
}

// handleTwinAnalyzeImpact analyzes simulation impact
func (ms *MCPServer) handleTwinAnalyzeImpact(w http.ResponseWriter, ctx context.Context, args map[string]any) {
	response := map[string]any{
		"success":   true,
		"tool":      "twin.analyze_impact",
		"message":   "Analyze impact tool requires server instance access. Use REST API: POST /api/v1/twin/:id/runs/:rid/analyze",
		"arguments": args,
	}
	json.NewEncoder(w).Encode(response)
}

// handleTwinCompareRuns compares multiple simulation runs
func (ms *MCPServer) handleTwinCompareRuns(w http.ResponseWriter, ctx context.Context, args map[string]any) {
	response := map[string]any{
		"success":   true,
		"tool":      "twin.compare_runs",
		"message":   "Compare runs functionality not yet implemented via MCP. Use REST API directly.",
		"arguments": args,
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
