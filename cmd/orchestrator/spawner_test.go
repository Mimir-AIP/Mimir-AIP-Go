package main

import (
	"testing"
	"time"

	"github.com/mimir-aip/mimir-aip-go/pkg/config"
	"github.com/mimir-aip/mimir-aip-go/pkg/k8s"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"github.com/mimir-aip/mimir-aip-go/pkg/queue"
)

func TestProcessQueueLeavesTaskQueuedWhenNoClusterHasCapacity(t *testing.T) {
	q, err := queue.NewQueue(nil)
	if err != nil {
		t.Fatalf("failed to create queue: %v", err)
	}
	defer q.Close()

	task := &models.WorkTask{
		ID:          "task-1",
		Type:        models.WorkTaskTypePipelineExecution,
		Status:      models.WorkTaskStatusQueued,
		Priority:    1,
		SubmittedAt: time.Now().UTC(),
		ProjectID:   "project-1",
		TaskSpec: models.TaskSpec{
			ProjectID:  "project-1",
			PipelineID: "pipeline-1",
		},
		MaxRetries: 1,
	}
	if err := q.Enqueue(task); err != nil {
		t.Fatalf("failed to enqueue task: %v", err)
	}

	spawner := NewWorkerSpawner(q, &k8s.ClusterPool{}, &config.Config{
		MinWorkers:        1,
		QueueThreshold:    0,
		ConcurrencyLimits: map[string]int{},
	})

	spawner.processQueue()

	queueLength, err := q.QueueLength()
	if err != nil {
		t.Fatalf("failed to read queue length: %v", err)
	}
	if queueLength != 1 {
		t.Fatalf("expected task to remain queued when no cluster has capacity, got queue length %d", queueLength)
	}

	persisted, err := q.GetWorkTask(task.ID)
	if err != nil {
		t.Fatalf("failed to reload queued task: %v", err)
	}
	if persisted.Status != models.WorkTaskStatusQueued {
		t.Fatalf("expected task status to remain queued, got %s", persisted.Status)
	}

	peeked, err := q.PeekNext()
	if err != nil {
		t.Fatalf("failed to peek queue: %v", err)
	}
	if peeked == nil || peeked.ID != task.ID {
		t.Fatalf("expected task %s to remain at the front of the queue, got %#v", task.ID, peeked)
	}
}
