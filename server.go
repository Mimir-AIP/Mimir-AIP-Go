package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
	AI "github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/AI"
	DP "github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/Data_Processing"
	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/Input"
	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/KnowledgeGraph"
	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/ML"
	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/Ontology"
	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/Output"
	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/Storage"
	"github.com/Mimir-AIP/Mimir-AIP-Go/utils"
	"github.com/gorilla/mux"
)

// getEnvOrDefault returns environment variable value or default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Server represents the Mimir AIP server
type Server struct {
	router      *mux.Router
	registry    *pipelines.PluginRegistry
	mcpServer   *MCPServer
	scheduler   *utils.Scheduler
	monitor     *utils.JobMonitor
	config      *utils.ConfigManager
	persistence *storage.PersistenceBackend
	tdb2Backend *knowledgegraph.TDB2Backend
	llmClient   AI.LLMClient
	llmClients  map[AI.LLMProvider]AI.LLMClient // Map of all available LLM clients by provider
	ontologyDir string                          // Directory for ontology files
}

// PipelineExecutionRequest represents a request to execute a pipeline
type PipelineExecutionRequest struct {
	PipelineID   string         `json:"pipeline_id,omitempty"`   // API-created pipeline ID
	PipelineName string         `json:"pipeline_name,omitempty"` // Legacy: pipeline name
	PipelineFile string         `json:"pipeline_file,omitempty"` // Legacy: YAML file path
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
	Type            string   `json:"type"`
	Name            string   `json:"name"`
	Description     string   `json:"description,omitempty"`
	AvailableModels []string `json:"available_models,omitempty"`
}

// NewServer creates a new Mimir AIP server
func NewServer() *Server {
	registry := pipelines.NewPluginRegistry()

	// Initialize persistence backend (SQLite)
	dbPath := os.Getenv("MIMIR_DB_PATH")
	if dbPath == "" {
		dbPath = "./data/mimir.db"
	}

	persistence, err := storage.NewPersistenceBackend(dbPath)
	if err != nil {
		log.Printf("Failed to initialize persistence backend: %v", err)
		log.Printf("Ontology features will be disabled")
		persistence = nil
	}

	// Initialize TDB2 backend (Jena Fuseki)
	fusekiURL := os.Getenv("FUSEKI_URL")
	if fusekiURL == "" {
		fusekiURL = "http://localhost:3030"
	}
	dataset := os.Getenv("FUSEKI_DATASET")
	if dataset == "" {
		dataset = "mimir"
	}
	tdb2Backend := knowledgegraph.NewTDB2Backend(fusekiURL, dataset)

	// Initialize LLM client map for configurable plugins
	llmClients := make(map[AI.LLMProvider]AI.LLMClient)

	// Mock LLM client is always available for testing/demo
	mockClient := AI.NewIntelligentMockLLMClient()
	llmClients[AI.ProviderMock] = mockClient

	// Register all AI providers as plugins (they use placeholder clients until configured)
	providers := []AI.LLMProvider{
		AI.ProviderOpenAI,
		AI.ProviderAnthropic,
		AI.ProviderOllama,
		AI.ProviderLocal,
		AI.ProviderAzure,
		AI.ProviderGoogle,
		AI.ProviderOpenRouter,
		AI.ProviderZAi,
	}

	for _, provider := range providers {
		llmClients[provider] = AI.NewUnconfiguredClient(provider)
	}

	// Select primary client (default to mock for demo)
	primaryProvider := AI.ProviderMock
	if os.Getenv("LLM_PROVIDER") != "" {
		if _, ok := llmClients[AI.LLMProvider(os.Getenv("LLM_PROVIDER"))]; ok {
			primaryProvider = AI.LLMProvider(os.Getenv("LLM_PROVIDER"))
		}
	}
	llmClient := llmClients[primaryProvider]
	log.Printf("✅ LLM system initialized with %d providers", len(llmClients))

	s := &Server{
		router:      mux.NewRouter(),
		registry:    registry,
		mcpServer:   NewMCPServer(registry),
		scheduler:   utils.NewScheduler(registry),
		monitor:     utils.NewJobMonitor(1000), // Keep last 1000 executions
		config:      utils.GetConfigManager(),
		persistence: persistence,
		tdb2Backend: tdb2Backend,
		llmClient:   llmClient,
		llmClients:  llmClients,
	}

	// Set the LLM plugin config getter so LLM clients can fetch their configuration
	AI.LLMPluginConfigGetter = func(pluginName string) (map[string]interface{}, error) {
		if s.persistence == nil {
			return nil, fmt.Errorf("no persistence backend")
		}
		db := s.persistence.GetDB()
		var configJSON []byte
		err := db.QueryRow("SELECT config FROM plugin_config WHERE plugin_name = ?", pluginName).Scan(&configJSON)
		if err != nil {
			return nil, err
		}
		if configJSON == nil {
			return nil, fmt.Errorf("no config found")
		}
		var config map[string]interface{}
		if err := json.Unmarshal(configJSON, &config); err != nil {
			return nil, err
		}
		return config, nil
	}

	s.registerDefaultPlugins()

	// Initialize data adapter registry (must be after plugin registration)
	ml.InitializeGlobalAdapterRegistry(s.registry)
	log.Println("✅ Data adapter registry initialized with plugin integration")

	// Load configuration BEFORE setting up routes so middleware can check config
	if err := utils.LoadGlobalConfig(); err != nil {
		log.Printf("Failed to load configuration: %v", err)
	}

	s.setupRoutes()
	_ = s.mcpServer.Initialize()

	// Initialize logging
	if err := utils.InitLogger(s.config.GetConfig().Logging); err != nil {
		log.Printf("Failed to initialize logger: %v", err)
	}

	// Initialize pipeline store
	if err := utils.InitializeGlobalPipelineStore("./pipelines"); err != nil {
		log.Printf("Failed to initialize pipeline store: %v", err)
	}

	// Initialize pipeline auto-extraction (must be after plugin registration and pipeline store)
	utils.InitializePipelineAutoExtraction(s.registry, utils.GetPipelineStore())

	// Initialize alert action executor (must be after persistence backend and plugin registry)
	if persistence != nil {
		utils.InitializeAlertActionExecutor(persistence.GetDB(), s.registry)
		// Enable email sender to read SMTP config from database
		utils.SetEmailSenderDB(persistence.GetDB())
	}

	// Initialize auto-ML handler (must be after persistence and TDB2 backends)
	ml.InitializeAutoMLHandler(persistence, tdb2Backend)

	// Initialize twin auto-creator (creates twins when models finish training)
	if persistence != nil {
		utils.InitializeTwinAutoCreator(persistence.GetDB())
	}

	// Start the scheduler
	if err := s.scheduler.Start(); err != nil {
		utils.GetLogger().Error("Failed to start scheduler", err, utils.Component("server"))
	}

	// Initialize monitoring executor and connect to scheduler
	if persistence != nil {
		monitoringExecutor := ml.NewMonitoringExecutor(persistence)
		s.scheduler.SetStorage(persistence)
		s.scheduler.SetMonitoringExecutor(monitoringExecutor)
		log.Println("Monitoring executor initialized and connected to scheduler")

		// Recover scheduled jobs from database after crash/restart
		if err := s.scheduler.RecoverJobsFromDatabase(); err != nil {
			log.Printf("⚠️  Failed to recover scheduled jobs: %v", err)
		}
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

	// Register input plugins for data ingestion
	csvPlugin := Input.NewCSVPlugin()
	markdownPlugin := Input.NewMarkdownPlugin()
	excelPlugin := Input.NewExcelPlugin()
	xmlPlugin := Input.NewXMLPlugin() // NEW: Demo plugin for auto-discovery
	jsonInputPlugin := Input.NewJSONPlugin()
	firmsPlugin := Input.NewFIRMSPlugin()
	polymarketPlugin := Input.NewPolymarketPlugin()
	gdeltPlugin := Input.NewGDELTPlugin()
	validatePlugin := DP.NewValidatePlugin()
	dedupePlugin := DP.NewDeduplicatePlugin()
	// Register Data_Processing plugins
	transformPlugin := DP.NewTransformPlugin()

	if err := s.registry.RegisterPlugin(csvPlugin); err != nil {
		log.Printf("Failed to register CSV plugin: %v", err)
	} else {
		log.Println("Registered CSV input plugin")
	}

	if err := s.registry.RegisterPlugin(markdownPlugin); err != nil {
		log.Printf("Failed to register Markdown plugin: %v", err)
	} else {
		log.Println("Registered Markdown input plugin")
	}

	if err := s.registry.RegisterPlugin(excelPlugin); err != nil {
		log.Printf("Failed to register Excel plugin: %v", err)
	} else {
		log.Println("Registered Excel input plugin")
	}

	if err := s.registry.RegisterPlugin(xmlPlugin); err != nil {
		log.Printf("Failed to register XML plugin: %v", err)
	} else {
		log.Println("Registered XML input plugin")
	}

	if err := s.registry.RegisterPlugin(jsonInputPlugin); err != nil {
		log.Printf("Failed to register JSON input plugin: %v", err)
	} else {
		log.Println("Registered JSON input plugin")
	}

	if err := s.registry.RegisterPlugin(firmsPlugin); err != nil {
		log.Printf("Failed to register FIRMS input plugin: %v", err)
	} else {
		log.Println("Registered FIRMS input plugin")
	}

	if err := s.registry.RegisterPlugin(polymarketPlugin); err != nil {
		log.Printf("Failed to register Polymarket input plugin: %v", err)
	} else {
		log.Println("Registered Polymarket input plugin")
	}

	if err := s.registry.RegisterPlugin(gdeltPlugin); err != nil {
		log.Printf("Failed to register GDELT input plugin: %v", err)
	} else {
		log.Println("Registered GDELT input plugin")
	}

	if err := s.registry.RegisterPlugin(transformPlugin); err != nil {
		log.Printf("Failed to register Data_Processing.transform plugin: %v", err)
	} else {
		log.Println("Registered Data_Processing.transform plugin")
	}

	if err := s.registry.RegisterPlugin(validatePlugin); err != nil {
		log.Printf("Failed to register Data_Processing.validate plugin: %v", err)
	} else {
		log.Println("Registered Data_Processing.validate plugin")
	}

	if err := s.registry.RegisterPlugin(dedupePlugin); err != nil {
		log.Printf("Failed to register Data_Processing.deduplicate plugin: %v", err)
	} else {
		log.Println("Registered Data_Processing.deduplicate plugin")
	}

	// Register output plugins
	jsonOutputPlugin := Output.NewJSONPlugin()
	if err := s.registry.RegisterPlugin(jsonOutputPlugin); err != nil {
		log.Printf("Failed to register JSON output plugin: %v", err)
	} else {
		log.Println("Registered JSON output plugin")
	}

	// Register Excel output plugin
	excelOutputPlugin := Output.NewExcelPlugin()
	if err := s.registry.RegisterPlugin(excelOutputPlugin); err != nil {
		log.Printf("Failed to register Excel output plugin: %v", err)
	} else {
		log.Println("Registered Excel output plugin")
	}

	// Register PDF output plugin
	pdfOutputPlugin := Output.NewPDFPlugin()
	if err := s.registry.RegisterPlugin(pdfOutputPlugin); err != nil {
		log.Printf("Failed to register PDF output plugin: %v", err)
	} else {
		log.Println("Registered PDF output plugin")
	}

	// Register ontology plugins if persistence is available
	if s.persistence != nil && s.tdb2Backend != nil {
		ontologyDir := os.Getenv("ONTOLOGY_DIR")
		if ontologyDir == "" {
			ontologyDir = "./data/ontologies"
		}

		// Create ontology directory if it doesn't exist
		if err := os.MkdirAll(ontologyDir, 0755); err != nil {
			log.Printf("Failed to create ontology directory: %v", err)
		} else {
			// Store ontology directory in server
			s.ontologyDir = ontologyDir

			// Register ontology management plugin
			ontologyPlugin := ontology.NewManagementPlugin(s.persistence, s.tdb2Backend, ontologyDir)
			if err := s.registry.RegisterPlugin(ontologyPlugin); err != nil {
				log.Printf("Failed to register ontology management plugin: %v", err)
			} else {
				log.Println("Registered ontology management plugin")
			}

			// Register extraction plugin (requires database access)
			extractionPlugin := ontology.NewExtractionPlugin(s.persistence.GetDB(), s.tdb2Backend, s.llmClient)
			if err := s.registry.RegisterPlugin(extractionPlugin); err != nil {
				log.Printf("Failed to register extraction plugin: %v", err)
			} else {
				log.Println("Registered extraction plugin")
			}

			// Register NL query plugin (requires LLM client)
			nlQueryPlugin := ontology.NewNLQueryPlugin(s.persistence.GetDB(), s.tdb2Backend, s.llmClient)
			if err := s.registry.RegisterPlugin(nlQueryPlugin); err != nil {
				log.Printf("Failed to register NL query plugin: %v", err)
			} else {
				log.Println("Registered NL query plugin")
			}
		}
	}

	// Initialize plugin configuration table
	if err := s.initializePluginConfig(); err != nil {
		log.Printf("Failed to initialize plugin config: %v", err)
	}

	// Register AI LLM plugins - register all available providers
	log.Println("Registering AI LLM plugins...")

	for provider, client := range s.llmClients {
		llmPlugin := AI.NewLLMPlugin(client, provider)
		if err := s.registry.RegisterPlugin(llmPlugin); err != nil {
			log.Printf("Failed to register %s LLM plugin: %v", provider, err)
		} else {
			log.Printf("Registered %s LLM plugin", provider)
		}
	}

	// Initialize plugin metadata table and seed from registry
	if err := s.initializePluginMetadata(); err != nil {
		log.Printf("Failed to initialize plugin metadata: %v", err)
	}
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

		// 3. Close ontology backends
		if s.persistence != nil {
			log.Println("Closing persistence backend...")
			if err := s.persistence.Close(); err != nil {
				log.Printf("Error closing persistence backend: %v", err)
			}
		}

		if s.tdb2Backend != nil {
			log.Println("Closing TDB2 backend...")
			if err := s.tdb2Backend.Close(); err != nil {
				log.Printf("Error closing TDB2 backend: %v", err)
			}
		}

		// 4. Close any open connections or resources
		log.Println("Cleaning up resources...")

		// 5. Flush any pending logs
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
