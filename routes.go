package main

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"

	"github.com/Mimir-AIP/Mimir-AIP-Go/utils"
	"github.com/gorilla/mux"
)

// setupRoutes sets up the HTTP routes with API versioning
func (s *Server) setupRoutes() {
	// Add middleware
	s.router.Use(s.loggingMiddleware)
	s.router.Use(s.errorRecoveryMiddleware)
	s.router.Use(s.defaultUserMiddleware) // Inject anonymous user when auth is disabled
	s.router.Use(utils.SecurityHeadersMiddleware)
	s.router.Use(utils.InputValidationMiddleware)
	s.router.Use(utils.PerformanceMiddleware)

	// Initialize authentication if enabled
	if s.config.GetConfig().Security.EnableAuth {
		if err := utils.InitAuthManager(s.config.GetConfig().Security); err != nil {
			utils.GetLogger().Error("Failed to initialize authentication", err, utils.Component("server"))
		}
	}

	// Create API version subrouters
	v1 := s.router.PathPrefix("/api/v1").Subrouter()
	v1.Use(s.defaultUserMiddleware) // Add first, before version middleware
	v1.Use(s.versionMiddleware("v1"))

	// Health check (no version)
	s.router.HandleFunc("/health", s.handleHealth).Methods("GET")

	// Pipeline execution
	v1.HandleFunc("/pipelines/execute", s.handleExecutePipeline).Methods("POST")

	// Pipeline management
	v1.HandleFunc("/pipelines", s.handleListPipelines).Methods("GET")
	v1.HandleFunc("/pipelines", s.handleCreatePipeline).Methods("POST")
	v1.HandleFunc("/pipelines/{id}", s.handleGetPipeline).Methods("GET")
	v1.HandleFunc("/pipelines/{id}", s.handleUpdatePipeline).Methods("PUT")
	v1.HandleFunc("/pipelines/{id}", s.handleDeletePipeline).Methods("DELETE")
	v1.HandleFunc("/pipelines/{id}/clone", s.handleClonePipeline).Methods("POST")
	v1.HandleFunc("/pipelines/{id}/validate", s.handleValidatePipeline).Methods("POST")
	v1.HandleFunc("/pipelines/{id}/history", s.handleGetPipelineHistory).Methods("GET")
	v1.HandleFunc("/pipelines/{id}/logs", s.handleGetPipelineLogs).Methods("GET")

	// Plugin management
	v1.HandleFunc("/plugins", s.handleListPlugins).Methods("GET")
	v1.HandleFunc("/plugins/{type}", s.handleListPluginsByType).Methods("GET")
	v1.HandleFunc("/plugins/{type}/{name}", s.handleGetPlugin).Methods("GET")

	// Plugin configuration
	v1.HandleFunc("/plugins/config", s.handleListPluginConfigs).Methods("GET")
	v1.HandleFunc("/plugins/{name}/config", s.handleGetPluginConfig).Methods("GET")
	v1.HandleFunc("/plugins/{name}/config", s.handleSetPluginConfig).Methods("PUT", "POST")
	v1.HandleFunc("/plugins/{name}/config", s.handleDeletePluginConfig).Methods("DELETE")

	// AI Providers management
	v1.HandleFunc("/ai/providers", s.handleListAIProviders).Methods("GET")
	v1.HandleFunc("/ai/providers/{provider}/models", s.handleFetchProviderModels).Methods("GET")

	// Agentic features
	v1.HandleFunc("/agent/execute", s.handleAgentExecute).Methods("POST")

	// MCP endpoints (no version prefix)
	s.router.PathPrefix("/mcp").Handler(s.mcpServer)

	// Scheduler endpoints
	v1.HandleFunc("/scheduler/jobs", s.handleListJobs).Methods("GET")
	v1.HandleFunc("/scheduler/jobs/{id}", s.handleGetJob).Methods("GET")
	v1.HandleFunc("/scheduler/jobs", s.handleCreateJob).Methods("POST")
	v1.HandleFunc("/scheduler/jobs/{id}", s.handleUpdateJob).Methods("PUT")
	v1.HandleFunc("/scheduler/jobs/{id}", s.handleDeleteJob).Methods("DELETE")
	v1.HandleFunc("/scheduler/jobs/{id}/enable", s.handleEnableJob).Methods("POST")
	v1.HandleFunc("/scheduler/jobs/{id}/disable", s.handleDisableJob).Methods("POST")
	v1.HandleFunc("/scheduler/jobs/{id}/logs", s.handleGetJobLogs).Methods("GET")

	// Execution logs endpoints
	v1.HandleFunc("/logs/executions", s.handleListExecutionLogs).Methods("GET")
	v1.HandleFunc("/logs/executions/{id}", s.handleGetExecutionLog).Methods("GET")

	// Visualization endpoints
	v1.HandleFunc("/visualize/pipeline", s.handleVisualizePipeline).Methods("POST")
	v1.HandleFunc("/visualize/status", s.handleVisualizeStatus).Methods("GET")
	v1.HandleFunc("/visualize/scheduler", s.handleVisualizeScheduler).Methods("GET")
	v1.HandleFunc("/visualize/plugins", s.handleVisualizePlugins).Methods("GET")

	// Performance monitoring endpoints
	v1.HandleFunc("/performance/metrics", s.handleGetPerformanceMetrics).Methods("GET")
	v1.HandleFunc("/performance/stats", s.handleGetPerformanceStats).Methods("GET")

	// Ontology management endpoints
	v1.HandleFunc("/ontology", s.handleListOntologies).Methods("GET")
	v1.HandleFunc("/ontologies", s.handleListOntologies).Methods("GET")
	v1.HandleFunc("/ontology", s.handleUploadOntology).Methods("POST")
	v1.HandleFunc("/ontology/{id}", s.handleGetOntology).Methods("GET")
	v1.HandleFunc("/ontology/{id}", s.handleUpdateOntology).Methods("PUT")
	v1.HandleFunc("/ontology/{id}", s.handleDeleteOntology).Methods("DELETE")
	v1.HandleFunc("/ontology/{id}/validate", s.handleValidateOntology).Methods("POST")
	v1.HandleFunc("/ontology/{id}/stats", s.handleOntologyStats).Methods("GET")
	v1.HandleFunc("/ontology/{id}/export", s.handleExportOntology).Methods("GET")
	v1.HandleFunc("/ontology/validate", s.handleValidateOntology).Methods("POST")

	// Ontology versioning endpoints
	v1.HandleFunc("/ontology/{id}/versions", s.handleListVersions).Methods("GET")
	v1.HandleFunc("/ontology/{id}/versions", s.handleCreateVersion).Methods("POST")
	v1.HandleFunc("/ontology/{id}/versions/compare", s.handleCompareVersions).Methods("GET")
	v1.HandleFunc("/ontology/{id}/versions/{vid}", s.handleGetVersion).Methods("GET")
	v1.HandleFunc("/ontology/{id}/versions/{vid}", s.handleDeleteVersion).Methods("DELETE")

	// Drift detection endpoints
	v1.HandleFunc("/ontology/{id}/drift/detect", s.handleTriggerDriftDetection).Methods("POST")
	v1.HandleFunc("/ontology/{id}/drift/history", s.handleGetDriftHistory).Methods("GET")

	// Suggestion management endpoints
	v1.HandleFunc("/ontology/{id}/suggestions", s.handleListSuggestions).Methods("GET")
	v1.HandleFunc("/ontology/{id}/suggestions/summary", s.handleGetSuggestionSummary).Methods("GET")
	v1.HandleFunc("/ontology/{id}/suggestions/{sid}", s.handleGetSuggestion).Methods("GET")
	v1.HandleFunc("/ontology/{id}/suggestions/{sid}/approve", s.handleApproveSuggestion).Methods("POST")
	v1.HandleFunc("/ontology/{id}/suggestions/{sid}/reject", s.handleRejectSuggestion).Methods("POST")
	v1.HandleFunc("/ontology/{id}/suggestions/{sid}/apply", s.handleApplySuggestion).Methods("POST")

	// Knowledge Graph endpoints
	v1.HandleFunc("/kg/query", s.handleSPARQLQuery).Methods("POST")
	v1.HandleFunc("/kg/nl-query", s.handleNLQuery).Methods("POST")
	v1.HandleFunc("/kg/stats", s.handleKnowledgeGraphStats).Methods("GET")
	v1.HandleFunc("/kg/subgraph", s.handleGetSubgraph).Methods("GET")
	v1.HandleFunc("/knowledge-graph/path-finding", s.handlePathFinding).Methods("POST")
	v1.HandleFunc("/knowledge-graph/reasoning", s.handleReasoning).Methods("POST")

	// Entity extraction endpoints
	v1.HandleFunc("/extraction/jobs", s.handleListExtractionJobs).Methods("GET")
	v1.HandleFunc("/extraction/jobs", s.handleCreateExtractionJob).Methods("POST")
	v1.HandleFunc("/extraction/jobs/{id}", s.handleGetExtractionJob).Methods("GET")

	// Job monitoring endpoints
	v1.HandleFunc("/jobs", s.handleListJobExecutions).Methods("GET")
	v1.HandleFunc("/jobs/{id}", s.handleGetJobExecution).Methods("GET")
	v1.HandleFunc("/jobs/running", s.handleGetRunningJobs).Methods("GET")
	v1.HandleFunc("/jobs/recent", s.handleGetRecentJobs).Methods("GET")
	v1.HandleFunc("/jobs/export", s.handleExportJobs).Methods("GET")
	v1.HandleFunc("/jobs/statistics", s.handleGetJobStatistics).Methods("GET")
	v1.HandleFunc("/jobs/{id}/stop", s.handleStopJobExecution).Methods("POST")

	// Digital Twin endpoints
	v1.HandleFunc("/twins", s.handleListTwins).Methods("GET")
	v1.HandleFunc("/twin", s.handleListTwins).Methods("GET")
	v1.HandleFunc("/twin/create", s.handleCreateTwin).Methods("POST")
	v1.HandleFunc("/twin/{id}", s.handleGetTwin).Methods("GET")
	v1.HandleFunc("/twin/{id}", s.handleUpdateTwin).Methods("PUT")
	v1.HandleFunc("/twin/{id}", s.handleDeleteTwin).Methods("DELETE")
	v1.HandleFunc("/twin/{id}/state", s.handleGetTwinState).Methods("GET")
	v1.HandleFunc("/twin/{id}/scenarios", s.handleListScenarios).Methods("GET")
	v1.HandleFunc("/twin/{id}/scenarios", s.handleCreateScenario).Methods("POST")
	v1.HandleFunc("/twin/{id}/scenarios/{sid}/run", s.handleRunSimulation).Methods("POST")
	v1.HandleFunc("/twin/{id}/runs/{rid}", s.handleGetSimulationRun).Methods("GET")
	v1.HandleFunc("/twin/{id}/runs/{rid}/timeline", s.handleGetSimulationTimeline).Methods("GET")
	v1.HandleFunc("/twin/{id}/runs/{rid}/analyze", s.handleAnalyzeImpact).Methods("POST")

	// Smart Digital Twin endpoints (AI-powered)
	v1.HandleFunc("/twin/{id}/whatif", s.handleWhatIfAnalysis).Methods("POST")                  // Natural language what-if analysis
	v1.HandleFunc("/twin/{id}/smart-scenarios", s.handleGenerateSmartScenarios).Methods("POST") // Auto-generate relevant scenarios
	v1.HandleFunc("/twin/{id}/analyze", s.handleAnalyzeOntology).Methods("GET")                 // Analyze ontology for patterns/risks
	v1.HandleFunc("/twin/{id}/insights", s.handleGetInsights).Methods("GET")                    // Proactive insights & suggestions

	// Autonomous Workflow endpoints
	v1.HandleFunc("/workflows", s.handleListWorkflows).Methods("GET")
	v1.HandleFunc("/workflows", s.handleCreateWorkflow).Methods("POST")
	v1.HandleFunc("/workflows/{id}", s.handleGetWorkflow).Methods("GET")
	v1.HandleFunc("/workflows/{id}/execute", s.handleExecuteWorkflow).Methods("POST")
	utils.GetLogger().Info("Registered workflow routes on v1 subrouter")

	// Schema inference endpoints
	v1.HandleFunc("/data/{id}/infer-schema", s.handleInferSchemaFromImport).Methods("POST")
	v1.HandleFunc("/schema/{id}/generate-ontology", s.handleGenerateOntologyFromSchema).Methods("POST")

	// System/Version endpoints
	v1.HandleFunc("/version", s.handleVersion).Methods("GET")

	// Agent Chat endpoints
	v1.HandleFunc("/chat", s.handleListConversations).Methods("GET")
	v1.HandleFunc("/chat", s.handleCreateConversation).Methods("POST")
	v1.HandleFunc("/chat/{id}", s.handleGetConversation).Methods("GET")
	v1.HandleFunc("/chat/{id}", s.handleUpdateConversation).Methods("PUT")
	v1.HandleFunc("/chat/{id}", s.handleDeleteConversation).Methods("DELETE")
	v1.HandleFunc("/chat/{id}/message", s.handleSendMessage).Methods("POST")

	// Agent Tools endpoints
	s.setupAgentToolRoutes()

	// Data Ingestion endpoints
	v1.HandleFunc("/data/plugins", s.handleListInputPlugins).Methods("GET")
	v1.HandleFunc("/data/upload", s.handleUploadData).Methods("POST")
	v1.HandleFunc("/data/preview", s.handlePreviewData).Methods("POST")
	v1.HandleFunc("/data/select", s.handleSelectData).Methods("POST")
	v1.HandleFunc("/data/import", s.handleDataImport).Methods("POST")

	// Machine Learning endpoints
	v1.HandleFunc("/models/train", s.handleTrainModel).Methods("POST")
	v1.HandleFunc("/models", s.handleListModels).Methods("GET")
	v1.HandleFunc("/models/{id}", s.handleGetModel).Methods("GET")
	v1.HandleFunc("/models/{id}", s.handleDeleteModel).Methods("DELETE")
	v1.HandleFunc("/models/{id}/predict", s.handlePredict).Methods("POST")
	v1.HandleFunc("/models/{id}/status", s.handleUpdateModelStatus).Methods("PATCH")

	// Auto-ML endpoints (ontology-driven)
	v1.HandleFunc("/ontology/{id}/ml-capabilities", s.handleGetMLCapabilities).Methods("GET")
	v1.HandleFunc("/ontology/{id}/auto-train", s.handleAutoTrain).Methods("POST")
	v1.HandleFunc("/ontology/{id}/train-for-goal", s.handleTrainForGoal).Methods("POST")
	v1.HandleFunc("/ontology/{id}/ml-suggestions", s.handleGetMLSuggestions).Methods("GET")

	// Data-based Auto-ML endpoint (new - accepts CSV/Excel/JSON data)
	v1.HandleFunc("/auto-train-with-data", s.handleAutoTrainWithData).Methods("POST")

	// Type inference endpoints
	v1.HandleFunc("/ontology/{id}/infer-types", s.handleInferTypes).Methods("POST")
	v1.HandleFunc("/ontology/{id}/inferred-types", s.handleSaveTypeInferences).Methods("POST")
	v1.HandleFunc("/ontology/{id}/inferred-types", s.handleGetTypeInferences).Methods("GET")

	// Monitoring endpoints
	v1.HandleFunc("/monitoring/jobs", s.handleListMonitoringJobs).Methods("GET")
	v1.HandleFunc("/monitoring/jobs", s.handleCreateMonitoringJob).Methods("POST")
	v1.HandleFunc("/monitoring/jobs/{id}", s.handleGetMonitoringJob).Methods("GET")
	v1.HandleFunc("/monitoring/jobs/{id}", s.handleUpdateMonitoringJob).Methods("PUT")
	v1.HandleFunc("/monitoring/jobs/{id}", s.handleDeleteMonitoringJob).Methods("DELETE")
	v1.HandleFunc("/monitoring/jobs/{id}/enable", s.handleEnableMonitoringJob).Methods("POST")
	v1.HandleFunc("/monitoring/jobs/{id}/disable", s.handleDisableMonitoringJob).Methods("POST")
	v1.HandleFunc("/monitoring/jobs/{id}/runs", s.handleGetMonitoringJobRuns).Methods("GET")
	v1.HandleFunc("/monitoring/rules", s.handleListMonitoringRules).Methods("GET")
	v1.HandleFunc("/monitoring/rules", s.handleCreateMonitoringRule).Methods("POST")
	v1.HandleFunc("/monitoring/rules/{id}", s.handleDeleteMonitoringRule).Methods("DELETE")
	v1.HandleFunc("/monitoring/alerts", s.handleListAlerts).Methods("GET")
	v1.HandleFunc("/monitoring/alerts/{id}", s.handleAcknowledgeAlert).Methods("PATCH")

	// Anomaly Detection endpoints
	v1.HandleFunc("/anomalies", s.handleListAnomalies).Methods("GET")
	v1.HandleFunc("/anomalies/{id}", s.handleUpdateAnomalyStatus).Methods("PATCH")

	// Configuration endpoints
	v1.HandleFunc("/config", s.handleGetConfig).Methods("GET")
	v1.HandleFunc("/config", s.handleUpdateConfig).Methods("PUT")
	v1.HandleFunc("/config/reload", s.handleReloadConfig).Methods("POST")
	v1.HandleFunc("/config/save", s.handleSaveConfig).Methods("POST")

	// Authentication endpoints
	auth := utils.GetAuthManager()
	v1.HandleFunc("/auth/login", s.handleLogin).Methods("POST")
	v1.HandleFunc("/auth/refresh", s.handleRefreshToken).Methods("POST")
	v1.HandleFunc("/auth/check", s.handleAuthCheck).Methods("GET")
	v1.HandleFunc("/auth/me", s.handleAuthMe).Methods("GET")
	v1.HandleFunc("/auth/logout", s.handleLogout).Methods("POST")
	v1.HandleFunc("/auth/users", s.handleListUsers).Methods("GET")
	v1.HandleFunc("/auth/apikeys", s.handleCreateAPIKey).Methods("POST")

	// Settings - API Keys Management
	v1.HandleFunc("/settings/api-keys", s.handleListAPIKeys).Methods("GET")
	v1.HandleFunc("/settings/api-keys", s.handleCreateAPIKeyFromSettings).Methods("POST")
	v1.HandleFunc("/settings/api-keys/{id}", s.handleUpdateAPIKey).Methods("PUT")
	v1.HandleFunc("/settings/api-keys/{id}", s.handleDeleteAPIKey).Methods("DELETE")
	v1.HandleFunc("/settings/api-keys/{id}/test", s.handleTestAPIKey).Methods("POST")

	// Data management endpoints
	v1.HandleFunc("/settings/data/clear", s.handleClearData).Methods("POST")

	// Plugin management endpoints
	v1.HandleFunc("/settings/plugins", s.handleListPluginMetadata).Methods("GET")
	v1.HandleFunc("/settings/plugins/upload", s.handleUploadPlugin).Methods("POST")
	v1.HandleFunc("/settings/plugins/{id}", s.handleUpdatePlugin).Methods("PUT")
	v1.HandleFunc("/settings/plugins/{id}", s.handleDeletePlugin).Methods("DELETE")
	v1.HandleFunc("/settings/plugins/{id}/reload", s.handleReloadPlugin).Methods("POST")

	// Protected endpoints with authentication
	protected := v1.PathPrefix("/protected").Subrouter()
	protected.Use(auth.AuthMiddleware([]string{})) // Require authentication
	protected.HandleFunc("/pipelines", s.handleExecutePipeline).Methods("POST")
	protected.HandleFunc("/scheduler/jobs", s.handleCreateJob).Methods("POST")
	protected.HandleFunc("/config", s.handleUpdateConfig).Methods("PUT")

	// ========================================
	// Frontend Proxy to Next.js Server
	// ========================================
	// Proxy non-API requests to Next.js server running on port 3000
	nextJSURL := os.Getenv("NEXTJS_URL")
	if nextJSURL == "" {
		nextJSURL = "http://localhost:3001"
	}

	// Parse Next.js backend URL
	nextJSBackend, err := url.Parse(nextJSURL)
	if err == nil {
		utils.GetLogger().Info("Proxying frontend requests to: " + nextJSURL)

		// Create reverse proxy to Next.js
		proxy := httputil.NewSingleHostReverseProxy(nextJSBackend)

		// Catch-all: proxy everything that's not matched by registered routes to Next.js
		// Note: gorilla/mux will match registered routes first, so this only handles
		// unmatched paths (frontend pages, static assets, etc.)
		s.router.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			proxy.ServeHTTP(w, r)
		})
	} else {
		utils.GetLogger().Warn("Failed to parse Next.js URL, serving API only")
	}

	// Debug: Log all registered routes
	utils.GetLogger().Info("=== Registered Routes ===")
	err = s.router.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		pathTemplate, _ := route.GetPathTemplate()
		methods, _ := route.GetMethods()
		utils.GetLogger().Info(fmt.Sprintf("  %v %s", methods, pathTemplate))
		return nil
	})
	if err != nil {
		utils.GetLogger().Warn(fmt.Sprintf("Failed to walk routes: %v", err))
	}
	utils.GetLogger().Info("=== End Routes ===")
}
