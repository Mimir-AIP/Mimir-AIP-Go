package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mimir-aip/mimir-aip-go/pkg/api"
	"github.com/mimir-aip/mimir-aip-go/pkg/config"
	"github.com/mimir-aip/mimir-aip-go/pkg/extraction"
	"github.com/mimir-aip/mimir-aip-go/pkg/k8s"
	"github.com/mimir-aip/mimir-aip-go/pkg/metadatastore"
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

	// Initialize Kubernetes client
	k8sClient, err := k8s.NewClient("mimir-aip")
	if err != nil {
		log.Fatalf("Failed to initialize Kubernetes client: %v", err)
	}

	log.Println("Connected to Kubernetes cluster")

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

	// Register filesystem storage plugin
	filesystemPlugin := storageplugins.NewFilesystemPlugin()
	storageService.RegisterPlugin("filesystem", filesystemPlugin)

	// Initialize ontology and extraction services
	ontologyService := ontology.NewService(store)
	extractionService := extraction.NewService(storageService)

	log.Println("Initialized project, pipeline, scheduler, storage, ontology, and extraction services")

	// Start scheduler
	schedulerService.Start()
	defer schedulerService.Stop()

	log.Println("Started job scheduler")

	// Start API server in a goroutine
	server := api.NewServer(q, cfg.Port)

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

	log.Println("Registered API handlers")

	go func() {
		if err := server.Start(); err != nil {
			log.Fatalf("Failed to start API server: %v", err)
		}
	}()

	// Start worker spawner
	spawner := NewWorkerSpawner(q, k8sClient, cfg)
	go spawner.Run()

	log.Println("Orchestrator started successfully")

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down orchestrator...")
}

// WorkerSpawner manages worker job creation and monitoring
type WorkerSpawner struct {
	queue     *queue.Queue
	k8sClient *k8s.Client
	config    *config.Config
}

// NewWorkerSpawner creates a new worker spawner
func NewWorkerSpawner(q *queue.Queue, k8sClient *k8s.Client, cfg *config.Config) *WorkerSpawner {
	return &WorkerSpawner{
		queue:     q,
		k8sClient: k8sClient,
		config:    cfg,
	}
}

// Run starts the worker spawning loop
func (ws *WorkerSpawner) Run() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		ws.processQueue()
	}
}

// processQueue checks the queue and spawns workers as needed
func (ws *WorkerSpawner) processQueue() {
	queueLength, err := ws.queue.QueueLength()
	if err != nil {
		log.Printf("Error getting queue length: %v", err)
		return
	}

	if queueLength == 0 {
		return
	}

	activeWorkers, err := ws.k8sClient.GetActiveWorkerCount()
	if err != nil {
		log.Printf("Error getting active worker count: %v", err)
		return
	}

	// Check if we should spawn a worker
	if !ws.shouldSpawnWorker(queueLength, int64(activeWorkers)) {
		log.Printf("Scaling decision: queue=%d, active=%d, min=%d, max=%d, threshold=%d - NOT spawning (limit reached or queue too small)",
			queueLength, activeWorkers, ws.config.MinWorkers, ws.config.MaxWorkers, ws.config.QueueThreshold)
		return
	}

	log.Printf("Scaling decision: queue=%d, active=%d, min=%d, max=%d, threshold=%d - SPAWNING worker",
		queueLength, activeWorkers, ws.config.MinWorkers, ws.config.MaxWorkers, ws.config.QueueThreshold)

	// Dequeue a work task
	task, err := ws.queue.Dequeue()
	if err != nil {
		log.Printf("Error dequeuing work task: %v", err)
		return
	}

	if task == nil {
		return // No tasks available
	}

	// Update task status to scheduled
	if err := ws.queue.UpdateWorkTaskStatus(task.ID, models.WorkTaskStatusScheduled, ""); err != nil {
		log.Printf("Error updating work task status: %v", err)
		return
	}

	// Create worker job
	workerImage := "mimir-aip/worker:latest"
	if err := ws.k8sClient.CreateWorkerJob(task, workerImage); err != nil {
		log.Printf("Error creating worker job: %v", err)
		// Update task status to failed
		ws.queue.UpdateWorkTaskStatus(task.ID, models.WorkTaskStatusFailed, err.Error())
		return
	}

	log.Printf("Spawned worker for task %s (type: %s)", task.ID, task.Type)

	// Update task status to spawned
	task.KubernetesJobName = "worker-task-" + task.ID
	if err := ws.queue.UpdateWorkTaskStatus(task.ID, models.WorkTaskStatusSpawned, ""); err != nil {
		log.Printf("Error updating work task status: %v", err)
	}
}

// shouldSpawnWorker determines if a new worker should be spawned
func (ws *WorkerSpawner) shouldSpawnWorker(queueLength, activeWorkers int64) bool {
	// Always maintain minimum workers
	if activeWorkers < int64(ws.config.MinWorkers) && queueLength > 0 {
		return true
	}

	// Don't exceed max workers
	if activeWorkers >= int64(ws.config.MaxWorkers) {
		return false
	}

	// Scale based on queue depth
	if queueLength > int64(ws.config.QueueThreshold) {
		return true
	}

	return false
}
