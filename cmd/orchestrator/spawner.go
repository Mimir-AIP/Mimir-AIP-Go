package main

import (
	"log"
	"time"

	"github.com/mimir-aip/mimir-aip-go/pkg/config"
	"github.com/mimir-aip/mimir-aip-go/pkg/k8s"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"github.com/mimir-aip/mimir-aip-go/pkg/queue"
)

// WorkerSpawner manages worker job creation across a pool of Kubernetes clusters.
type WorkerSpawner struct {
	queue  *queue.Queue
	pool   *k8s.ClusterPool
	config *config.Config
}

// NewWorkerSpawner creates a new WorkerSpawner backed by a ClusterPool.
func NewWorkerSpawner(q *queue.Queue, pool *k8s.ClusterPool, cfg *config.Config) *WorkerSpawner {
	return &WorkerSpawner{
		queue:  q,
		pool:   pool,
		config: cfg,
	}
}

// Run starts the worker spawning loop.
func (ws *WorkerSpawner) Run() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		ws.processQueue()
	}
}

// processQueue checks the queue and spawns workers across the cluster pool as needed.
func (ws *WorkerSpawner) processQueue() {
	queueLength, err := ws.queue.QueueLength()
	if err != nil {
		log.Printf("Error getting queue length: %v", err)
		return
	}

	if queueLength == 0 {
		return
	}

	// Query active worker counts per cluster
	counts := ws.pool.ActiveWorkerCounts()
	totalActive := sumCounts(counts)
	totalCapacity := ws.pool.TotalCapacity()

	if !ws.shouldSpawnWorker(queueLength, int64(totalActive), int64(totalCapacity)) {
		log.Printf("Scaling decision: queue=%d, active=%d, capacity=%d, min=%d, threshold=%d - NOT spawning",
			queueLength, totalActive, totalCapacity, ws.config.MinWorkers, ws.config.QueueThreshold)
		return
	}

	// Check per-type concurrency limit before dequeuing
	nextTask, err := ws.queue.PeekNext()
	if err != nil {
		log.Printf("Error peeking at next task: %v", err)
		return
	}
	if nextTask != nil {
		if limit, ok := ws.config.ConcurrencyLimits[string(nextTask.Type)]; ok && limit > 0 {
			activeForType, err := ws.queue.CountActiveByType(nextTask.Type)
			if err != nil {
				log.Printf("Error counting active tasks by type: %v", err)
				return
			}
			if activeForType >= int64(limit) {
				log.Printf("Per-type concurrency limit reached for %s (%d/%d), skipping tick",
					nextTask.Type, activeForType, limit)
				return
			}
		}
	}

	log.Printf("Scaling decision: queue=%d, active=%d, capacity=%d - SPAWNING worker",
		queueLength, totalActive, totalCapacity)

	if nextTask == nil {
		return
	}

	// Select the target cluster before dequeuing so lack of capacity cannot strand the task outside the heap.
	entry := ws.pool.SelectCluster(counts, nextTask.TaskSpec.PreferredCluster)
	if entry == nil {
		log.Printf("No cluster has available capacity; leaving task %s queued", nextTask.ID)
		return
	}

	// Dequeue a work task only after a destination cluster is known.
	task, err := ws.queue.Dequeue()
	if err != nil {
		log.Printf("Error dequeuing work task: %v", err)
		return
	}
	if task == nil {
		return
	}

	task.ClusterName = entry.Config.Name

	// Update task status to scheduled
	if err := ws.queue.UpdateWorkTaskStatus(task.ID, models.WorkTaskStatusScheduled, ""); err != nil {
		log.Printf("Error updating work task status: %v", err)
		return
	}

	// Create worker job on the selected cluster
	if err := entry.Client.CreateWorkerJob(task, ws.config.WorkerImage, entry.Config.OrchestratorURL); err != nil {
		log.Printf("Error creating worker job on cluster %q: %v", entry.Config.Name, err)
		if retryErr := ws.queue.RequeueWithRetry(task.ID, "k8s_job_creation_failed"); retryErr != nil {
			log.Printf("Error requeueing task %s: %v", task.ID, retryErr)
		}
		return
	}

	log.Printf("Spawned worker for task %s (type: %s) on cluster %q", task.ID, task.Type, entry.Config.Name)

	task.KubernetesJobName = "worker-task-" + task.ID
	if err := ws.queue.UpdateWorkTaskStatus(task.ID, models.WorkTaskStatusSpawned, ""); err != nil {
		log.Printf("Error updating work task status to spawned: %v", err)
	}
}

// shouldSpawnWorker determines if a new worker should be spawned.
func (ws *WorkerSpawner) shouldSpawnWorker(queueLength, totalActive, totalCapacity int64) bool {
	// Always maintain minimum workers while there is work to do
	if totalActive < int64(ws.config.MinWorkers) && queueLength > 0 {
		return true
	}

	// Don't exceed total capacity across all clusters
	if totalActive >= totalCapacity {
		return false
	}

	// Scale based on queue depth threshold
	if queueLength > int64(ws.config.QueueThreshold) {
		return true
	}

	return false
}

// sumCounts returns the sum of all values in a cluster-count map.
func sumCounts(counts map[string]int) int {
	total := 0
	for _, v := range counts {
		total += v
	}
	return total
}
