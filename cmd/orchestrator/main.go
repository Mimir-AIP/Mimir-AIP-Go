package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/mimir-aip/mimir-aip-go/pkg/api"
	"github.com/mimir-aip/mimir-aip-go/pkg/config"
	"github.com/mimir-aip/mimir-aip-go/pkg/digitaltwin"
	"github.com/mimir-aip/mimir-aip-go/pkg/extraction"
	"github.com/mimir-aip/mimir-aip-go/pkg/k8s"
	mcpserver "github.com/mimir-aip/mimir-aip-go/pkg/mcp"
	"github.com/mimir-aip/mimir-aip-go/pkg/metadatastore"
	"github.com/mimir-aip/mimir-aip-go/pkg/mlmodel"
	"github.com/mimir-aip/mimir-aip-go/pkg/ontology"
	"github.com/mimir-aip/mimir-aip-go/pkg/pipeline"
	"github.com/mimir-aip/mimir-aip-go/pkg/plugins"
	"github.com/mimir-aip/mimir-aip-go/pkg/project"
	"github.com/mimir-aip/mimir-aip-go/pkg/queue"
	"github.com/mimir-aip/mimir-aip-go/pkg/scheduler"
	"github.com/mimir-aip/mimir-aip-go/pkg/storage"
	storageplugins "github.com/mimir-aip/mimir-aip-go/pkg/storage/plugins"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	log.Printf("Starting Mimir AIP Orchestrator in %s mode", cfg.Environment)

	// Initialize in-memory job queue
	q, err := queue.NewQueue()
	if err != nil {
		log.Fatalf("Failed to initialize job queue: %v", err)
	}

	log.Println("Initialized in-memory job queue")

	// Initialize Kubernetes cluster pool
	// When CLUSTER_CONFIG_FILE is set, load multi-cluster config; otherwise use a single
	// in-cluster pool that preserves the existing single-cluster behaviour exactly.
	var clusterPool *k8s.ClusterPool
	if cfg.ClusterConfigFile != "" {
		clusterPool, err = k8s.LoadClusterPool(cfg.ClusterConfigFile, cfg.WorkerAuthToken)
		if err != nil {
			log.Fatalf("Failed to load cluster config from %q: %v", cfg.ClusterConfigFile, err)
		}
		log.Printf("Loaded %d cluster(s) from %s", clusterPool.Len(), cfg.ClusterConfigFile)
	} else {
		// Single in-cluster (backward-compatible default)
		clusterPool, err = k8s.NewClusterPool([]k8s.ClusterConfig{
			{
				Name:            "primary",
				Kubeconfig:      "",
				Namespace:       cfg.WorkerNamespace,
				OrchestratorURL: cfg.OrchestratorURL,
				MaxWorkers:      cfg.MaxWorkers,
				ServiceAccount:  cfg.WorkerServiceAccount,
			},
		}, cfg.WorkerAuthToken)
		if err != nil {
			log.Fatalf("Failed to initialize Kubernetes client: %v", err)
		}
		log.Println("Connected to Kubernetes cluster (single in-cluster mode)")
	}

	// Initialize storage
	storageDir := cfg.Environment + "-data"
	if dataDir := os.Getenv("STORAGE_DIR"); dataDir != "" {
		storageDir = dataDir
	}

	// Use SQLite for storage
	dbPath := storageDir + "/mimir.db"
	if err := os.MkdirAll(storageDir, 0755); err != nil {
		log.Fatalf("Failed to create storage directory: %v", err)
	}

	store, err := metadatastore.NewSQLiteStore(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize SQLite storage: %v", err)
	}
	log.Printf("Initialized SQLite storage at: %s", dbPath)

	// Initialize plugin service (metadata management only - workers compile plugins)
	tempDir := storageDir + "/temp"
	pluginService, err := plugins.NewService(store, tempDir)
	if err != nil {
		log.Fatalf("Failed to initialize plugin service: %v", err)
	}
	log.Println("Initialized plugin service")

	// Note: The orchestrator only manages plugin metadata
	// Workers compile and load plugins from source when executing pipelines

	// Initialize services
	projectService := project.NewService(store)
	pipelineService := pipeline.NewService(store) // Orchestrator doesn't need plugin registry
	schedulerService := scheduler.NewService(store, pipelineService, q)

	// Initialize storage service
	storageService := storage.NewService(store)

	// Register all storage plugins
	storageService.RegisterPlugin("filesystem", storageplugins.NewFilesystemPlugin())
	storageService.RegisterPlugin("postgresql", storageplugins.NewPostgresPlugin())
	storageService.RegisterPlugin("mysql", storageplugins.NewMySQLPlugin())
	storageService.RegisterPlugin("mongodb", storageplugins.NewMongoDBPlugin())
	storageService.RegisterPlugin("s3", storageplugins.NewS3Plugin())
	storageService.RegisterPlugin("redis", storageplugins.NewRedisPlugin())
	storageService.RegisterPlugin("elasticsearch", storageplugins.NewElasticsearchPlugin())
	storageService.RegisterPlugin("neo4j", storageplugins.NewNeo4jPlugin())

	// Initialize ontology and extraction services
	ontologyService := ontology.NewService(store)
	extractionService := extraction.NewService(storageService)

	// Initialize ML model service
	mlmodelService := mlmodel.NewService(store, ontologyService, storageService, q)

	// Initialize digital twin service
	dtService := digitaltwin.NewService(store, ontologyService, storageService, mlmodelService, q)

	// Start prediction cache eviction background job
	go dtService.StartCacheEviction(context.Background())

	log.Println("Initialized project, pipeline, scheduler, storage, ontology, extraction, ML model, and digital twin services")

	// Start scheduler
	schedulerService.Start()
	defer schedulerService.Stop()

	// Start model monitoring service
	monitoringService := mlmodel.NewMonitoringService(store, mlmodelService, storageService)
	monitoringService.Start()
	defer monitoringService.Stop()

	log.Println("Started job scheduler")

	// Start API server in a goroutine
	server := api.NewServer(q, cfg.Port, cfg.WorkerAuthToken)

	// Register project handlers
	projectHandler := api.NewProjectHandler(projectService)
	server.RegisterHandler("/api/projects", projectHandler.HandleProjects)
	server.RegisterHandler("/api/projects/", projectHandler.HandleProject)
	server.RegisterHandler("/api/projects/clone", projectHandler.HandleProjectClone)


	// Register pipeline handlers
	pipelineHandler := api.NewPipelineHandler(pipelineService, q)
	server.RegisterHandler("/api/pipelines", pipelineHandler.HandlePipelines)
	server.RegisterHandler("/api/pipelines/", pipelineHandler.HandlePipeline)
	server.RegisterHandler("/api/pipelines/execute", pipelineHandler.HandlePipelineExecute)

	// Register schedule handlers
	scheduleHandler := api.NewScheduleHandler(schedulerService)
	server.RegisterHandler("/api/schedules", scheduleHandler.HandleSchedules)
	server.RegisterHandler("/api/schedules/", scheduleHandler.HandleSchedule)

	// Register plugin handlers
	pluginHandler := api.NewPluginHandler(pluginService)
	server.RegisterHandler("/api/plugins", pluginHandler.HandlePlugins)
	server.RegisterHandler("/api/plugins/", pluginHandler.HandlePlugin)

	// Register storage handlers
	storageHandler := api.NewStorageHandler(storageService)
	server.RegisterHandler("/api/storage/configs", storageHandler.HandleStorageConfigs)
	server.RegisterHandler("/api/storage/configs/", storageHandler.HandleStorageConfig)
	server.RegisterHandler("/api/storage/store", storageHandler.HandleStorageStore)
	server.RegisterHandler("/api/storage/retrieve", storageHandler.HandleStorageRetrieve)
	server.RegisterHandler("/api/storage/update", storageHandler.HandleStorageUpdate)
	server.RegisterHandler("/api/storage/delete", storageHandler.HandleStorageDelete)
	server.RegisterHandler("/api/storage/health", storageHandler.HandleStorageHealth)

	// Register ontology handlers
	ontologyHandler := api.NewOntologyHandler(ontologyService)
	server.RegisterHandler("/api/ontologies", ontologyHandler.HandleOntologies)
	server.RegisterHandler("/api/ontologies/", ontologyHandler.HandleOntology)

	// Register extraction handler
	extractionHandler := api.NewExtractionHandler(extractionService, ontologyService)
	server.RegisterHandler("/api/extraction/generate-ontology", extractionHandler.HandleExtractAndGenerate)

	// Register ML model handlers
	mlmodelHandler := api.NewMLModelHandler(mlmodelService)
	server.RegisterHandler("/api/ml-models", mlmodelHandler.HandleMLModels)
	server.RegisterHandler("/api/ml-models/", mlmodelHandler.HandleMLModel)
	server.RegisterHandler("/api/ml-models/recommend", mlmodelHandler.HandleMLModelRecommendation)
	server.RegisterHandler("/api/ml-models/train", mlmodelHandler.HandleMLModelTraining)

	// Register digital twin handlers
	dtHandler := api.NewDigitalTwinHandler(dtService)
	server.RegisterHandler("/api/digital-twins", dtHandler.HandleDigitalTwins)
	server.RegisterHandler("/api/digital-twins/", dtHandler.HandleDigitalTwin)

	log.Println("Registered API handlers")

	// Register MCP server at /mcp/ (SSE transport)
	// Clients connect via: GET http://localhost:<port>/mcp/sse
	mcpSrv := mcpserver.New(projectService, pipelineService, mlmodelService, dtService, storageService, ontologyService, extractionService, schedulerService, q)
	mcpHandler := mcpSrv.SSEHandler("http://localhost:" + cfg.Port)
	server.RegisterHandler("/mcp/", func(w http.ResponseWriter, r *http.Request) {
		mcpHandler.ServeHTTP(w, r)
	})
	log.Println("Registered MCP SSE handler at /mcp/")

	go func() {
		if err := server.Start(); err != nil {
			log.Fatalf("Failed to start API server: %v", err)
		}
	}()

	// Start worker spawner
	spawner := NewWorkerSpawner(q, clusterPool, cfg)
	go spawner.Run()

	log.Println("Orchestrator started successfully")

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down orchestrator...")
}

