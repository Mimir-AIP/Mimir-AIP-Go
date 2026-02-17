package queue

import (
	"container/heap"
	"fmt"
	"sync"
	"time"

	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

// Queue provides in-memory work task queue operations with priority support
type Queue struct {
	mu        sync.RWMutex
	pq        *PriorityQueue
	workTasks map[string]*models.WorkTask
	taskIndex map[string]int // Maps task ID to index in priority queue
}

// NewQueue creates a new in-memory queue instance
func NewQueue() (*Queue, error) {
	pq := make(PriorityQueue, 0)
	heap.Init(&pq)

	return &Queue{
		pq:        &pq,
		workTasks: make(map[string]*models.WorkTask),
		taskIndex: make(map[string]int),
	}, nil
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

// UpdateWorkTaskStatus updates the status of a work task
func (q *Queue) UpdateWorkTaskStatus(taskID string, status models.WorkTaskStatus, errorMsg string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	task, ok := q.workTasks[taskID]
	if !ok {
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
