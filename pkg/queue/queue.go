package queue

import (
	"container/heap"
	"fmt"
	"sync"
	"time"

	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

// WorkTaskListener receives status-change events for work tasks.
type WorkTaskListener interface {
	OnWorkTaskStatusChanged(task *models.WorkTask)
}

// Queue provides in-memory work task queue operations with priority support.
type Queue struct {
	mu        sync.RWMutex
	pq        *PriorityQueue
	workTasks map[string]*models.WorkTask
	taskIndex map[string]int // Maps task ID to index in priority queue
	listeners []WorkTaskListener
}

// NewQueue creates a new in-memory queue instance.
func NewQueue() (*Queue, error) {
	pq := make(PriorityQueue, 0)
	heap.Init(&pq)

	return &Queue{
		pq:        &pq,
		workTasks: make(map[string]*models.WorkTask),
		taskIndex: make(map[string]int),
		listeners: make([]WorkTaskListener, 0),
	}, nil
}

// RegisterListener registers a work-task status listener.
func (q *Queue) RegisterListener(listener WorkTaskListener) {
	if listener == nil {
		return
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	q.listeners = append(q.listeners, listener)
}

// Enqueue adds a work task to the queue
func (q *Queue) Enqueue(task *models.WorkTask) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	// Calculate priority score (lower = higher priority, older first)
	score := float64(time.Now().Unix()) / float64(task.Priority+1)

	item := &PriorityQueueItem{
		TaskID:   task.ID,
		Priority: score,
	}

	// Add to priority queue
	heap.Push(q.pq, item)

	// Store task data
	q.workTasks[task.ID] = task

	return nil
}

// Dequeue retrieves the next work task from the queue
func (q *Queue) Dequeue() (*models.WorkTask, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.pq.Len() == 0 {
		return nil, nil // No tasks available
	}

	// Pop highest priority task
	item := heap.Pop(q.pq).(*PriorityQueueItem)

	// Retrieve task data
	task, ok := q.workTasks[item.TaskID]
	if !ok {
		return nil, fmt.Errorf("work task data not found: %s", item.TaskID)
	}

	// Note: We keep the task in q.workTasks for status tracking
	// It will be cleaned up by UpdateWorkTaskStatus when completed

	return task, nil
}

// GetWorkTask retrieves a work task by ID
func (q *Queue) GetWorkTask(taskID string) (*models.WorkTask, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	task, ok := q.workTasks[taskID]
	if !ok {
		return nil, fmt.Errorf("work task not found: %s", taskID)
	}

	return task, nil
}

// UpdateWorkTaskStatus updates the status of a work task.
func (q *Queue) UpdateWorkTaskStatus(taskID string, status models.WorkTaskStatus, errorMsg string) error {
	q.mu.Lock()
	task, ok := q.workTasks[taskID]
	if !ok {
		q.mu.Unlock()
		return fmt.Errorf("work task not found: %s", taskID)
	}

	task.Status = status
	if errorMsg != "" {
		task.ErrorMessage = errorMsg
	}

	now := time.Now()
	switch status {
	case models.WorkTaskStatusExecuting:
		task.StartedAt = &now
	case models.WorkTaskStatusCompleted, models.WorkTaskStatusFailed, models.WorkTaskStatusTimeout, models.WorkTaskStatusCancelled:
		task.CompletedAt = &now
	}

	taskSnapshot := *task
	listeners := append([]WorkTaskListener(nil), q.listeners...)
	q.mu.Unlock()

	q.notifyListeners(&taskSnapshot, listeners)
	return nil
}

// ApplyWorkTaskResult stores a worker-reported result, updates task status, and notifies listeners.
func (q *Queue) ApplyWorkTaskResult(taskID string, result *models.WorkTaskResult) error {
	if result == nil {
		return fmt.Errorf("work task result is required")
	}
	q.mu.Lock()
	task, ok := q.workTasks[taskID]
	if !ok {
		q.mu.Unlock()
		return fmt.Errorf("work task not found: %s", taskID)
	}
	task.Status = result.Status
	task.OutputLocation = result.OutputLocation
	task.ResultMetadata = cloneMap(result.Metadata)
	task.ErrorMessage = result.ErrorMessage
	now := time.Now()
	switch result.Status {
	case models.WorkTaskStatusExecuting:
		task.StartedAt = &now
	case models.WorkTaskStatusCompleted, models.WorkTaskStatusFailed, models.WorkTaskStatusTimeout, models.WorkTaskStatusCancelled:
		task.CompletedAt = &now
	}
	taskSnapshot := *task
	listeners := append([]WorkTaskListener(nil), q.listeners...)
	q.mu.Unlock()
	q.notifyListeners(&taskSnapshot, listeners)
	return nil
}

// QueueLength returns the current length of the work task queue
func (q *Queue) QueueLength() (int64, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	return int64(q.pq.Len()), nil
}

// GetHighPriorityWorkTasks returns work tasks with priority above a threshold
func (q *Queue) GetHighPriorityWorkTasks(minPriority int) ([]*models.WorkTask, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	var highPriorityTasks []*models.WorkTask
	for _, task := range q.workTasks {
		if task.Priority >= minPriority && task.Status == models.WorkTaskStatusQueued {
			highPriorityTasks = append(highPriorityTasks, task)
		}
	}

	return highPriorityTasks, nil
}

// ListWorkTasks returns all known work tasks (all statuses).
func (q *Queue) ListWorkTasks() ([]*models.WorkTask, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	tasks := make([]*models.WorkTask, 0, len(q.workTasks))
	for _, task := range q.workTasks {
		tasks = append(tasks, task)
	}
	return tasks, nil
}

// CountActiveByType returns the number of tasks with status spawned or executing for a given type.
func (q *Queue) CountActiveByType(taskType models.WorkTaskType) (int64, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	var count int64
	for _, task := range q.workTasks {
		if task.Type == taskType &&
			(task.Status == models.WorkTaskStatusSpawned || task.Status == models.WorkTaskStatusExecuting) {
			count++
		}
	}
	return count, nil
}

// PeekNext returns the highest-priority queued task without removing it from the queue.
// Returns nil if no queued tasks exist.
func (q *Queue) PeekNext() (*models.WorkTask, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	if q.pq.Len() == 0 {
		return nil, nil
	}

	// The heap root is the minimum (highest priority)
	item := (*q.pq)[0]
	task, ok := q.workTasks[item.TaskID]
	if !ok {
		return nil, fmt.Errorf("work task data not found: %s", item.TaskID)
	}
	return task, nil
}

// RequeueWithRetry re-queues a failed task if it has remaining retries, otherwise marks it failed.
func (q *Queue) RequeueWithRetry(taskID string, reason string) error {
	q.mu.Lock()
	task, ok := q.workTasks[taskID]
	if !ok {
		q.mu.Unlock()
		return fmt.Errorf("work task not found: %s", taskID)
	}

	if task.RetryCount < task.MaxRetries {
		task.RetryCount++
		task.RetryReason = reason
		task.Status = models.WorkTaskStatusQueued
		task.KubernetesJobName = ""
		task.ErrorMessage = ""

		// Re-add to priority queue with slightly lower priority to deprioritise retries.
		score := float64(time.Now().Unix())/float64(task.Priority+1) + float64(task.RetryCount)*60
		item := &PriorityQueueItem{
			TaskID:   task.ID,
			Priority: score,
		}
		heap.Push(q.pq, item)

		taskSnapshot := *task
		listeners := append([]WorkTaskListener(nil), q.listeners...)
		q.mu.Unlock()
		q.notifyListeners(&taskSnapshot, listeners)
		return nil
	}

	// Exhausted retries — mark permanently failed.
	now := time.Now()
	task.Status = models.WorkTaskStatusFailed
	task.CompletedAt = &now
	task.ErrorMessage = fmt.Sprintf("failed after %d retries: %s", task.RetryCount, reason)
	taskSnapshot := *task
	listeners := append([]WorkTaskListener(nil), q.listeners...)
	q.mu.Unlock()
	q.notifyListeners(&taskSnapshot, listeners)
	return nil
}

func (q *Queue) notifyListeners(task *models.WorkTask, listeners []WorkTaskListener) {
	for _, listener := range listeners {
		listener.OnWorkTaskStatusChanged(task)
	}
}

func cloneMap(values map[string]any) map[string]any {
	if values == nil {
		return nil
	}
	cloned := make(map[string]any, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

// Close closes the queue (no-op for in-memory implementation)
func (q *Queue) Close() error {
	return nil
}

// PriorityQueueItem represents an item in the priority queue
type PriorityQueueItem struct {
	TaskID   string
	Priority float64 // Lower value = higher priority
	index    int     // Index in heap
}

// PriorityQueue implements heap.Interface
type PriorityQueue []*PriorityQueueItem

func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Less(i, j int) bool {
	// Lower priority value = higher priority
	return pq[i].Priority < pq[j].Priority
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *PriorityQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*PriorityQueueItem)
	item.index = n
	*pq = append(*pq, item)
}

func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil  // avoid memory leak
	item.index = -1 // for safety
	*pq = old[0 : n-1]
	return item
}
