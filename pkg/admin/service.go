package admin

import (
	"fmt"
	"time"

	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"github.com/mimir-aip/mimir-aip-go/pkg/queue"
)

type MetadataResetter interface {
	ResetAll() error
}

type ResetSummary struct {
	ResetAt time.Time `json:"reset_at"`
	Message string    `json:"message"`
}

var ErrResetBlockedByActiveTasks = fmt.Errorf("factory reset is blocked while queued or active tasks exist")

// Service owns destructive platform-wide administrative operations.
type Service struct {
	store MetadataResetter
	queue *queue.Queue
}

func NewService(store MetadataResetter, q *queue.Queue) *Service {
	return &Service{store: store, queue: q}
}

func (s *Service) FactoryReset() (*ResetSummary, error) {
	if s.store == nil {
		return nil, fmt.Errorf("metadata store is not configured")
	}
	if s.queue == nil {
		return nil, fmt.Errorf("queue is not configured")
	}

	snapshot := s.queue.Snapshot()
	if hasInFlightTasks(snapshot) {
		return nil, ErrResetBlockedByActiveTasks
	}
	if err := s.store.ResetAll(); err != nil {
		return nil, fmt.Errorf("reset metadata store: %w", err)
	}
	s.queue.Reset()
	return &ResetSummary{
		ResetAt: time.Now().UTC(),
		Message: "Mimir metadata has been reset. External data in connected storage backends was not deleted.",
	}, nil
}

func hasInFlightTasks(snapshot *queue.Snapshot) bool {
	if snapshot == nil {
		return false
	}
	for _, status := range []models.WorkTaskStatus{
		models.WorkTaskStatusQueued,
		models.WorkTaskStatusScheduled,
		models.WorkTaskStatusSpawned,
		models.WorkTaskStatusExecuting,
	} {
		if snapshot.TasksByStatus[string(status)] > 0 {
			return true
		}
	}
	return false
}
