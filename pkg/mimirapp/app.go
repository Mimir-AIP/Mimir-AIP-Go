package mimirapp

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	adminpkg "github.com/mimir-aip/mimir-aip-go/pkg/admin"
	"github.com/mimir-aip/mimir-aip-go/pkg/analysis"
	"github.com/mimir-aip/mimir-aip-go/pkg/api"
	automationpkg "github.com/mimir-aip/mimir-aip-go/pkg/automation"
	"github.com/mimir-aip/mimir-aip-go/pkg/config"
	"github.com/mimir-aip/mimir-aip-go/pkg/digitaltwin"
	"github.com/mimir-aip/mimir-aip-go/pkg/execution"
	"github.com/mimir-aip/mimir-aip-go/pkg/extraction"
	"github.com/mimir-aip/mimir-aip-go/pkg/k8s"
	"github.com/mimir-aip/mimir-aip-go/pkg/llm"
	"github.com/mimir-aip/mimir-aip-go/pkg/llm/providers"
	mcpserver "github.com/mimir-aip/mimir-aip-go/pkg/mcp"
	"github.com/mimir-aip/mimir-aip-go/pkg/metadatastore"
	"github.com/mimir-aip/mimir-aip-go/pkg/mlmodel"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"github.com/mimir-aip/mimir-aip-go/pkg/ontology"
	"github.com/mimir-aip/mimir-aip-go/pkg/pipeline"
	"github.com/mimir-aip/mimir-aip-go/pkg/plugins"
	"github.com/mimir-aip/mimir-aip-go/pkg/project"
	"github.com/mimir-aip/mimir-aip-go/pkg/queue"
	"github.com/mimir-aip/mimir-aip-go/pkg/scheduler"
	"github.com/mimir-aip/mimir-aip-go/pkg/storage"
	storageplugins "github.com/mimir-aip/mimir-aip-go/pkg/storage/plugins"
)

type Options struct {
	Frontend http.Handler
}

func Run(cfg *config.Config, options Options) error {
	log.Printf("Starting Mimir AIP Orchestrator in %s mode", cfg.Environment)

	var (
		clusterPool *k8s.ClusterPool
		err         error
	)
	if cfg.ExecutionMode == config.ExecutionModeKubernetes {
		if cfg.ClusterConfigFile != "" {
			clusterPool, err = k8s.LoadClusterPool(cfg.ClusterConfigFile, cfg.WorkerAuthToken)
			if err != nil {
				return fmt.Errorf("load cluster config %q: %w", cfg.ClusterConfigFile, err)
			}
			log.Printf("Loaded %d cluster(s) from %s", clusterPool.Len(), cfg.ClusterConfigFile)
		} else {
			clusterPool, err = k8s.NewClusterPool([]k8s.ClusterConfig{{
				Name:            "primary",
				Kubeconfig:      "",
				Namespace:       cfg.WorkerNamespace,
				OrchestratorURL: cfg.OrchestratorURL,
				MaxWorkers:      cfg.MaxWorkers,
				ServiceAccount:  cfg.WorkerServiceAccount,
			}}, cfg.WorkerAuthToken)
			if err != nil {
				return fmt.Errorf("initialize kubernetes client: %w", err)
			}
			log.Println("Connected to Kubernetes cluster (single in-cluster mode)")
		}
	} else if cfg.ExecutionMode != config.ExecutionModeLocal {
		return fmt.Errorf("unknown execution mode %q", cfg.ExecutionMode)
	}

	storageDir, err := resolveStorageDir(cfg)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(storageDir, 0755); err != nil {
		return fmt.Errorf("create storage directory: %w", err)
	}

	dbPath := filepath.Join(storageDir, "mimir.db")
	store, err := metadatastore.NewSQLiteStore(dbPath)
	if err != nil {
		return fmt.Errorf("initialize sqlite storage: %w", err)
	}
	log.Printf("Initialized SQLite storage at: %s", dbPath)

	q, err := queue.NewQueue(store)
	if err != nil {
		return fmt.Errorf("initialize durable job queue: %w", err)
	}
	log.Println("Initialized durable job queue")
	adminService := adminpkg.NewService(store, q)

	var backend execution.Backend
	switch cfg.ExecutionMode {
	case config.ExecutionModeKubernetes:
		backend = execution.NewKubernetesBackend(q, clusterPool, cfg)
	case config.ExecutionModeLocal:
		backend = execution.NewLocalBackend(q, cfg)
	default:
		return fmt.Errorf("execution mode %q is not implemented", cfg.ExecutionMode)
	}

	tempDir := filepath.Join(storageDir, "temp")
	pluginService, err := plugins.NewService(store, tempDir)
	if err != nil {
		return fmt.Errorf("initialize plugin service: %w", err)
	}
	log.Println("Initialized plugin service")

	projectService := project.NewService(store)
	pipelineService := pipeline.NewService(store)
	schedulerService := scheduler.NewService(store, pipelineService, q)
	projectService.SetScheduleDeleter(schedulerService)
	projectService.SetTaskCleaner(q)
	storageService := storage.NewService(store)
	pipelineService.SetStorageSvc(storageService)

	storageService.RegisterPlugin("filesystem", storageplugins.NewFilesystemPlugin())
	storageService.RegisterPlugin("postgresql", storageplugins.NewPostgresPlugin())
	storageService.RegisterPlugin("mysql", storageplugins.NewMySQLPlugin())
	storageService.RegisterPlugin("mongodb", storageplugins.NewMongoDBPlugin())
	storageService.RegisterPlugin("s3", storageplugins.NewS3Plugin())
	storageService.RegisterPlugin("redis", storageplugins.NewRedisPlugin())
	storageService.RegisterPlugin("elasticsearch", storageplugins.NewElasticsearchPlugin())
	storageService.RegisterPlugin("neo4j", storageplugins.NewNeo4jPlugin())

	appDir := resolveAppDir(cfg, storageDir)
	storagePluginDir := os.Getenv("STORAGE_PLUGIN_DIR")
	if storagePluginDir == "" {
		storagePluginDir = filepath.Join(appDir, "storage-plugins")
	}
	pluginLoader, err := storage.NewPluginLoader(appDir, storagePluginDir, tempDir)
	if err != nil {
		log.Printf("Warning: failed to initialise dynamic storage plugin loader: %v — external storage plugins will not be available", err)
	} else {
		storageService.SetPluginLoader(pluginLoader)
		if loadErr := storageService.LoadInstalledExternalPlugins(); loadErr != nil {
			log.Printf("Warning: some external storage plugins failed to reload: %v", loadErr)
		}
		log.Println("Dynamic storage plugin loader ready")
	}

	ontologyService := ontology.NewService(store)
	llmProviderDir := os.Getenv("LLM_PROVIDER_DIR")
	if llmProviderDir == "" {
		llmProviderDir = filepath.Join(appDir, "llm-providers")
	}
	llmLoader, llmLoaderErr := llm.NewLoader(appDir, llmProviderDir, tempDir)
	if llmLoaderErr != nil {
		log.Printf("Warning: LLM provider loader unavailable: %v", llmLoaderErr)
	}

	llmService := llm.NewService(nil, "", false).WithStore(store).WithLoader(llmLoader)
	llmService.RegisterProvider("openrouter", providers.NewOpenRouterProvider(cfg.LLMAPIKey))
	llmService.RegisterProvider("openai_compat", providers.NewOpenAICompatProvider("openai_compat", cfg.LLMBaseURL, cfg.LLMAPIKey))
	if err := llmService.LoadInstalledExternalProviders(); err != nil {
		log.Printf("Warning: some external LLM providers failed to reload: %v", err)
	}
	if cfg.LLMEnabled && cfg.LLMProvider != "" {
		p, err := llmService.GetProvider(cfg.LLMProvider)
		if err != nil {
			log.Printf("Warning: LLM_PROVIDER %q not found in registry — LLM disabled", cfg.LLMProvider)
		} else {
			defaultModel := cfg.LLMModel
			if defaultModel == "" {
				switch cfg.LLMProvider {
				case "openrouter":
					defaultModel = "openrouter/free"
				case "openai_compat":
					defaultModel = "gpt-4o-mini"
				}
			}
			llmService.SetActiveProvider(p, defaultModel)
			log.Printf("LLM provider: %s (model: %s)", cfg.LLMProvider, defaultModel)
		}
	} else {
		log.Println("LLM not configured — extraction uses statistical heuristics only")
	}

	extractionService := extraction.NewService(storageService).WithLLM(llmService)
	analysisService := analysis.NewService(store, extractionService, storageService)
	automationService := automationpkg.NewService(store)
	mlmodelService := mlmodel.NewService(store, ontologyService, storageService, q)
	dtService := digitaltwin.NewService(store, automationService, ontologyService, storageService, mlmodelService, q)
	twinProcessor := digitaltwin.NewProcessor(store, dtService, analysisService, q)
	pipelineCompletionBridge := automationpkg.NewPipelineCompletionBridge()
	pipelineCompletionBridge.RegisterListener(digitaltwin.NewAutomationListener(automationService, twinProcessor))
	q.RegisterListener(pipelineCompletionBridge)
	ensureDefaultTwinProcessingAutomations(dtService, automationService)
	go dtService.StartCacheEviction(context.Background())

	schedulerService.Start()
	defer schedulerService.Stop()
	monitoringService := mlmodel.NewMonitoringService(store, mlmodelService, storageService)
	monitoringService.Start()
	defer monitoringService.Stop()

	server := api.NewServer(q, cfg.Port, cfg.WorkerAuthToken)
	registerHandlers(server, store, q, cfg, projectService, pipelineService, schedulerService, analysisService, pluginService, storageService, ontologyService, extractionService, mlmodelService, dtService, twinProcessor, automationService, adminService)
	if options.Frontend != nil {
		server.RegisterHandler("/", func(w http.ResponseWriter, r *http.Request) {
			options.Frontend.ServeHTTP(w, r)
		})
	}

	executionCtx, stopExecution := context.WithCancel(context.Background())
	defer stopExecution()
	go func() {
		if err := server.Start(); err != nil {
			log.Fatalf("Failed to start API server: %v", err)
		}
	}()
	go backend.Run(executionCtx)

	log.Println("Mimir runtime started successfully")
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	log.Println("Shutting down orchestrator...")
	return nil
}

func registerHandlers(server *api.Server, store metadatastore.MetadataStore, q *queue.Queue, cfg *config.Config, projectService *project.Service, pipelineService *pipeline.Service, schedulerService *scheduler.Service, analysisService *analysis.Service, pluginService *plugins.Service, storageService *storage.Service, ontologyService *ontology.Service, extractionService *extraction.Service, mlmodelService *mlmodel.Service, dtService *digitaltwin.Service, twinProcessor *digitaltwin.Processor, automationService *automationpkg.Service, adminService *adminpkg.Service) {
	projectStateProvider := api.NewProjectStateProvider(store, q)
	projectHandler := api.NewProjectHandler(projectService, projectStateProvider)
	server.RegisterHandler("/api/projects", projectHandler.HandleProjects)
	server.RegisterHandler("/api/projects/", projectHandler.HandleProject)

	pipelineHandler := api.NewPipelineHandler(pipelineService, q)
	server.RegisterHandler("/api/pipelines", pipelineHandler.HandlePipelines)
	server.RegisterHandler("/api/pipelines/", pipelineHandler.HandlePipeline)

	scheduleHandler := api.NewScheduleHandler(schedulerService)
	server.RegisterHandler("/api/schedules", scheduleHandler.HandleSchedules)
	server.RegisterHandler("/api/schedules/", scheduleHandler.HandleSchedule)

	analysisHandler := api.NewAnalysisHandler(analysisService)
	server.RegisterHandler("/api/analysis/resolver", analysisHandler.HandleResolverRun)
	server.RegisterHandler("/api/analysis/resolver/metrics", analysisHandler.HandleResolverMetrics)
	server.RegisterHandler("/api/reviews", analysisHandler.HandleReviewItems)
	server.RegisterHandler("/api/reviews/", analysisHandler.HandleReviewItem)
	server.RegisterHandler("/api/insights", analysisHandler.HandleInsights)

	pluginHandler := api.NewPluginHandler(pluginService)
	server.RegisterHandler("/api/plugins", pluginHandler.HandlePlugins)
	server.RegisterHandler("/api/plugins/", pluginHandler.HandlePlugin)

	storageHandler := api.NewStorageHandler(storageService)
	server.RegisterHandler("/api/storage/configs", storageHandler.HandleStorageConfigs)
	server.RegisterHandler("/api/storage/configs/", storageHandler.HandleStorageConfig)
	server.RegisterHandler("/api/storage/store", storageHandler.HandleStorageStore)
	server.RegisterHandler("/api/storage/retrieve", storageHandler.HandleStorageRetrieve)
	server.RegisterHandler("/api/storage/update", storageHandler.HandleStorageUpdate)
	server.RegisterHandler("/api/storage/delete", storageHandler.HandleStorageDelete)
	server.RegisterHandler("/api/storage/health", storageHandler.HandleStorageHealth)
	server.RegisterHandler("/api/storage/metadata", storageHandler.HandleStorageMetadata)
	server.RegisterHandler("/api/storage/ingestion-health", storageHandler.HandleIngestionHealth)

	storagePluginHandler := api.NewStoragePluginHandler(storageService)
	server.RegisterHandler("/api/storage-plugins", storagePluginHandler.HandleStoragePlugins)
	server.RegisterHandler("/api/storage-plugins/", storagePluginHandler.HandleStoragePlugin)

	ontologyHandler := api.NewOntologyHandler(ontologyService)
	server.RegisterHandler("/api/ontologies", ontologyHandler.HandleOntologies)
	server.RegisterHandler("/api/ontologies/", ontologyHandler.HandleOntology)

	extractionHandler := api.NewExtractionHandler(extractionService, ontologyService)
	server.RegisterHandler("/api/extraction/generate-ontology", extractionHandler.HandleExtractAndGenerate)

	mlmodelHandler := api.NewMLModelHandler(mlmodelService)
	server.RegisterHandler("/api/ml-providers", mlmodelHandler.HandleMLProviders)
	server.RegisterHandler("/api/ml-providers/", mlmodelHandler.HandleMLProviders)
	server.RegisterHandler("/api/ml-models", mlmodelHandler.HandleMLModels)
	server.RegisterHandler("/api/ml-models/", mlmodelHandler.HandleMLModel)
	server.RegisterHandler("/api/ml-models/recommend", mlmodelHandler.HandleMLModelRecommendation)
	server.RegisterHandler("/api/ml-models/train", mlmodelHandler.HandleMLModelTraining)

	dtHandler := api.NewDigitalTwinHandler(dtService, twinProcessor, automationService)
	twinProcessingHandler := api.NewTwinProcessingHandler(twinProcessor)
	server.RegisterHandler("/api/digital-twins", dtHandler.HandleDigitalTwins)
	server.RegisterHandler("/api/digital-twins/", dtHandler.HandleDigitalTwin)
	server.RegisterWorkerHandler("/api/internal/twin-runs/", twinProcessingHandler.HandleInternalTwinRuns)

	mcpSrv := mcpserver.New(projectService, pipelineService, automationService, analysisService, twinProcessor, mlmodelService, dtService, storageService, ontologyService, extractionService, schedulerService, q)
	mcpHandler := mcpSrv.SSEHandler("http://localhost:" + cfg.Port)
	server.RegisterHandler("/mcp/", func(w http.ResponseWriter, r *http.Request) {
		mcpHandler.ServeHTTP(w, r)
	})
	adminHandler := api.NewAdminHandler(adminService)
	server.RegisterHandler("/api/admin/settings/", adminHandler.HandleAdminSettings)
}

func resolveStorageDir(cfg *config.Config) (string, error) {
	if dataDir := os.Getenv("STORAGE_DIR"); dataDir != "" {
		return dataDir, nil
	}
	if cfg.ExecutionMode != config.ExecutionModeLocal {
		return cfg.Environment + "-data", nil
	}
	baseDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve local app data directory: %w", err)
	}
	return filepath.Join(baseDir, "MimirAIP"), nil
}

func resolveAppDir(cfg *config.Config, storageDir string) string {
	if cfg.ExecutionMode == config.ExecutionModeLocal {
		return storageDir
	}
	if appDir := os.Getenv("APP_DIR"); appDir != "" {
		return appDir
	}
	return "/app"
}

func ensureDefaultTwinProcessingAutomations(dtService *digitaltwin.Service, automationService *automationpkg.Service) {
	if dtService == nil || automationService == nil {
		return
	}
	twins, err := dtService.ListDigitalTwins()
	if err != nil {
		log.Printf("Skipping twin automation backfill: %v", err)
		return
	}
	for _, twin := range twins {
		automations, err := automationService.ListByProject(twin.ProjectID)
		if err != nil {
			log.Printf("Skipping automation backfill for project %s: %v", twin.ProjectID, err)
			continue
		}
		found := false
		for _, automation := range automations {
			if automation.TargetType == models.AutomationTargetTypeDigitalTwin && automation.TargetID == twin.ID && automation.TriggerType == models.AutomationTriggerTypePipelineCompleted && automation.ActionType == models.AutomationActionTypeProcessTwin {
				found = true
				break
			}
		}
		if found {
			continue
		}
		_, err = automationService.Create(&models.AutomationCreateRequest{
			ProjectID:   twin.ProjectID,
			Name:        twin.Name + " processing",
			Description: "Default automation: process this twin after ingestion pipelines complete.",
			TargetType:  models.AutomationTargetTypeDigitalTwin,
			TargetID:    twin.ID,
			TriggerType: models.AutomationTriggerTypePipelineCompleted,
			TriggerConfig: map[string]any{
				"pipeline_types": []string{string(models.PipelineTypeIngestion)},
			},
			ActionType: models.AutomationActionTypeProcessTwin,
		})
		if err != nil {
			log.Printf("Failed to backfill default automation for twin %s: %v", twin.ID, err)
		}
	}
}
