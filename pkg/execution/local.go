package execution

import (
	"context"
	"log"
	"os"
	"sync"
	"time"

	"github.com/mimir-aip/mimir-aip-go/pkg/config"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"github.com/mimir-aip/mimir-aip-go/pkg/queue"
	"github.com/mimir-aip/mimir-aip-go/pkg/workexec"
)

// LocalBackend executes queued work in-process against the same orchestrator API.
// It preserves the async queue/task lifecycle while removing the Kubernetes job dependency.
type LocalBackend struct {
	queue  *queue.Queue
	config *config.Config

	mu          sync.Mutex
	active      map[string]struct{}
	workerEnvMu sync.Mutex
}

// NewLocalBackend creates an in-process execution backend for local mode.
func NewLocalBackend(q *queue.Queue, cfg *config.Config) *LocalBackend {
	return &LocalBackend{
		queue:  q,
		config: cfg,
		active: make(map[string]struct{}),
	}
}

// Run starts the local dispatch loop until the context is cancelled.
func (lb *LocalBackend) Run(ctx context.Context) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			lb.processQueue(ctx)
		}
	}
}

func (lb *LocalBackend) processQueue(ctx context.Context) {
	for lb.hasCapacity() {
		queueLength, err := lb.queue.QueueLength()
		if err != nil {
			log.Printf("Error getting queue length: %v", err)
			return
		}
		if queueLength == 0 {
			return
		}

		nextTask, err := lb.queue.PeekNext()
		if err != nil {
			log.Printf("Error peeking at next task: %v", err)
			return
		}
		if nextTask == nil {
			return
		}
		if !lb.canRunType(nextTask.Type) {
			return
		}

		task, err := lb.queue.Dequeue()
		if err != nil {
			log.Printf("Error dequeuing work task: %v", err)
			return
		}
		if task == nil {
			return
		}

		if err := lb.queue.UpdateWorkTaskStatus(task.ID, models.WorkTaskStatusScheduled, ""); err != nil {
			log.Printf("Error updating work task %s to scheduled: %v", task.ID, err)
			return
		}

		lb.markActive(task.ID)
		go lb.runTask(ctx, task)
	}
}

func (lb *LocalBackend) runTask(ctx context.Context, task *models.WorkTask) {
	defer lb.unmarkActive(task.ID)

	select {
	case <-ctx.Done():
		return
	default:
	}

	lb.workerEnvMu.Lock()
	defer lb.workerEnvMu.Unlock()

	previousID, hadID := os.LookupEnv("WORKTASK_ID")
	previousType, hadType := os.LookupEnv("WORKTASK_TYPE")
	previousURL, hadURL := os.LookupEnv("ORCHESTRATOR_URL")
	previousToken, hadToken := os.LookupEnv("WORKER_AUTH_TOKEN")
	defer restoreEnv("WORKTASK_ID", previousID, hadID)
	defer restoreEnv("WORKTASK_TYPE", previousType, hadType)
	defer restoreEnv("ORCHESTRATOR_URL", previousURL, hadURL)
	defer restoreEnv("WORKER_AUTH_TOKEN", previousToken, hadToken)

	_ = os.Setenv("WORKTASK_ID", task.ID)
	_ = os.Setenv("WORKTASK_TYPE", string(task.Type))
	_ = os.Setenv("ORCHESTRATOR_URL", lb.config.OrchestratorURL)
	if lb.config.WorkerAuthToken != "" {
		_ = os.Setenv("WORKER_AUTH_TOKEN", lb.config.WorkerAuthToken)
	} else {
		_ = os.Unsetenv("WORKER_AUTH_TOKEN")
	}

	if err := workexec.RunFromEnvironment(); err != nil {
		log.Printf("Local execution failed for task %s: %v", task.ID, err)
	}
}

func (lb *LocalBackend) hasCapacity() bool {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	return len(lb.active) < lb.totalCapacity()
}

func (lb *LocalBackend) totalCapacity() int {
	// pkg/workexec still derives task identity from process environment, so local
	// execution serializes launches until the worker runtime is fully parameterized.
	return 1
}

func (lb *LocalBackend) canRunType(taskType models.WorkTaskType) bool {
	limit, ok := lb.config.ConcurrencyLimits[string(taskType)]
	if !ok || limit <= 0 {
		return true
	}
	activeForType, err := lb.queue.CountActiveByType(taskType)
	if err != nil {
		log.Printf("Error counting active tasks by type: %v", err)
		return false
	}
	return activeForType < int64(limit)
}

func (lb *LocalBackend) markActive(taskID string) {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	lb.active[taskID] = struct{}{}
}

func (lb *LocalBackend) unmarkActive(taskID string) {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	delete(lb.active, taskID)
}

func restoreEnv(key, value string, existed bool) {
	if existed {
		_ = os.Setenv(key, value)
		return
	}
	_ = os.Unsetenv(key)
}
