package queue

import (
	"container/heap"
	"fmt"
	"sync"
	"time"

	"github.com/mimir-aip/mimir-aip-go/pkg/metadatastore"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

// WorkTaskListener receives status-change events for work tasks.
type WorkTaskListener interface {
	OnWorkTaskStatusChanged(task *models.WorkTask)
}

// Queue provides work task queue operations with priority support backed by optional metadata persistence.
type Queue struct {
	mu        sync.RWMutex
	store     metadatastore.MetadataStore
	pq        *PriorityQueue
	workTasks map[string]*models.WorkTask
	listeners []WorkTaskListener
}

// Snapshot summarizes queue depth and known task counts for health/metrics surfaces.
type Snapshot struct {
	QueueLength   int64          `json:"queue_length"`
	TasksByStatus map[string]int `json:"tasks_by_status"`
	TasksByType   map[string]int `json:"tasks_by_type"`
	FailedTasks   int            `json:"failed_tasks"`
	TotalTasks    int            `json:"total_tasks"`
}

// NewQueue creates a queue instance. When store is non-nil, work tasks are persisted and reconstructed on startup.
func NewQueue(store metadatastore.MetadataStore) (*Queue, error) {
	pq := make(PriorityQueue, 0)
	heap.Init(&pq)
	q := &Queue{
		store:     store,
		pq:        &pq,
		workTasks: make(map[string]*models.WorkTask),
		listeners: make([]WorkTaskListener, 0),
	}
	if err := q.loadPersistedTasks(); err != nil {
		return nil, err
	}
	return q, nil
}

func (q *Queue) loadPersistedTasks() error {
	if q.store == nil {
		return nil
	}
	tasks, err := q.store.ListWorkTasks()
	if err != nil {
		return fmt.Errorf("failed to load persisted work tasks: %w", err)
	}
	for _, task := range tasks {
		if task == nil {
			continue
		}
		restored := cloneWorkTask(task)
		if restored.Status == models.WorkTaskStatusScheduled {
			restored.Status = models.WorkTaskStatusQueued
			restored.ClusterName = ""
			restored.KubernetesJobName = ""
			if err := q.store.SaveWorkTask(restored); err != nil {
				return fmt.Errorf("failed to recover scheduled work task %s: %w", restored.ID, err)
			}
		}
		q.workTasks[restored.ID] = restored
		if restored.Status == models.WorkTaskStatusQueued {
			heap.Push(q.pq, &PriorityQueueItem{TaskID: restored.ID, Priority: priorityScore(restored, restored.SubmittedAt)})
		}
	}
	return nil
}

func priorityScore(task *models.WorkTask, basis time.Time) float64 {
	at := basis
	if at.IsZero() {
		at = time.Now().UTC()
	}
	return float64(at.Unix()) / float64(task.Priority+1)
}

func (q *Queue) persistTask(task *models.WorkTask) error {
	if q.store == nil {
		return nil
	}
	return q.store.SaveWorkTask(task)
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

// Enqueue adds a work task to the queue.
func (q *Queue) Enqueue(task *models.WorkTask) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if task.SubmittedAt.IsZero() {
		task.SubmittedAt = time.Now().UTC()
	}
	if task.Status == "" {
		task.Status = models.WorkTaskStatusQueued
	}
	if err := q.persistTask(task); err != nil {
		return err
	}
	heap.Push(q.pq, &PriorityQueueItem{TaskID: task.ID, Priority: priorityScore(task, task.SubmittedAt)})
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
	previous := cloneWorkTask(task)
	task.Status = status
	if errorMsg != "" {
		task.ErrorMessage = errorMsg
	}
	now := time.Now().UTC()
	switch status {
	case models.WorkTaskStatusExecuting:
		task.StartedAt = &now
	case models.WorkTaskStatusCompleted, models.WorkTaskStatusFailed, models.WorkTaskStatusTimeout, models.WorkTaskStatusCancelled:
		task.CompletedAt = &now
	}
	if err := q.persistTask(task); err != nil {
		q.workTasks[taskID] = previous
		q.mu.Unlock()
		return err
	}
	taskSnapshot := cloneWorkTask(task)
	listeners := append([]WorkTaskListener(nil), q.listeners...)
	q.mu.Unlock()
	q.notifyListeners(taskSnapshot, listeners)
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
	previous := cloneWorkTask(task)
	task.Status = result.Status
	task.OutputLocation = result.OutputLocation
	task.ResultMetadata = cloneMap(result.Metadata)
	task.ErrorMessage = result.ErrorMessage
	now := time.Now().UTC()
	switch result.Status {
	case models.WorkTaskStatusExecuting:
		task.StartedAt = &now
	case models.WorkTaskStatusCompleted, models.WorkTaskStatusFailed, models.WorkTaskStatusTimeout, models.WorkTaskStatusCancelled:
		task.CompletedAt = &now
	}
	if err := q.persistTask(task); err != nil {
		q.workTasks[taskID] = previous
		q.mu.Unlock()
		return err
	}
	taskSnapshot := cloneWorkTask(task)
	listeners := append([]WorkTaskListener(nil), q.listeners...)
	q.mu.Unlock()
	q.notifyListeners(taskSnapshot, listeners)
	return nil
}

// Snapshot returns a consistent queue/task summary for health and metrics endpoints.
func (q *Queue) Snapshot() *Snapshot {
	q.mu.RLock()
	defer q.mu.RUnlock()

	snapshot := &Snapshot{
		QueueLength:   int64(q.pq.Len()),
		TasksByStatus: make(map[string]int),
		TasksByType:   make(map[string]int),
		TotalTasks:    len(q.workTasks),
	}
	for _, task := range q.workTasks {
		snapshot.TasksByStatus[string(task.Status)]++
		snapshot.TasksByType[string(task.Type)]++
		if task.Status == models.WorkTaskStatusFailed {
			snapshot.FailedTasks++
		}
	}
	return snapshot
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
	previous := cloneWorkTask(task)

	if task.RetryCount < task.MaxRetries {
		task.RetryCount++
		task.RetryReason = reason
		task.Status = models.WorkTaskStatusQueued
		task.KubernetesJobName = ""
		task.ClusterName = ""
		task.ErrorMessage = ""
		if err := q.persistTask(task); err != nil {
			q.workTasks[taskID] = previous
			q.mu.Unlock()
			return err
		}
		heap.Push(q.pq, &PriorityQueueItem{TaskID: task.ID, Priority: priorityScore(task, time.Now().UTC().Add(time.Duration(task.RetryCount)*time.Minute))})
		taskSnapshot := cloneWorkTask(task)
		listeners := append([]WorkTaskListener(nil), q.listeners...)
		q.mu.Unlock()
		q.notifyListeners(taskSnapshot, listeners)
		return nil
	}

	now := time.Now().UTC()
	task.Status = models.WorkTaskStatusFailed
	task.CompletedAt = &now
	task.ErrorMessage = fmt.Sprintf("failed after %d retries: %s", task.RetryCount, reason)
	if err := q.persistTask(task); err != nil {
		q.workTasks[taskID] = previous
		q.mu.Unlock()
		return err
	}
	taskSnapshot := cloneWorkTask(task)
	listeners := append([]WorkTaskListener(nil), q.listeners...)
	q.mu.Unlock()
	q.notifyListeners(taskSnapshot, listeners)
	return nil
}

func cloneWorkTask(task *models.WorkTask) *models.WorkTask {
	if task == nil {
		return nil
	}
	clone := *task
	clone.TaskSpec.Parameters = cloneMap(task.TaskSpec.Parameters)
	clone.DataAccess.InputDatasets = append([]string(nil), task.DataAccess.InputDatasets...)
	clone.ResultMetadata = cloneMap(task.ResultMetadata)
	if task.StartedAt != nil {
		startedAt := *task.StartedAt
		clone.StartedAt = &startedAt
	}
	if task.CompletedAt != nil {
		completedAt := *task.CompletedAt
		clone.CompletedAt = &completedAt
	}
	return &clone
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

// Reset clears all in-memory queued and tracked work tasks after the backing store has been reset.
func (q *Queue) Reset() {
	q.mu.Lock()
	defer q.mu.Unlock()
	pq := make(PriorityQueue, 0)
	heap.Init(&pq)
	q.pq = &pq
	q.workTasks = make(map[string]*models.WorkTask)
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
