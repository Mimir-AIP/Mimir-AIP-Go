package automation

import (
	"sync"
	"time"

	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

// PipelineCompletedEvent is emitted when a worker-backed pipeline task finishes successfully.
// It carries only persisted task/result data so listeners never depend on worker-local state.
type PipelineCompletedEvent struct {
	ProjectID      string
	PipelineID     string
	PipelineType   models.PipelineType
	WorkTaskID     string
	OutputLocation string
	CompletedAt    time.Time
	Metadata       map[string]any
}

// PipelineCompletionListener handles completed pipeline events.
type PipelineCompletionListener interface {
	OnPipelineCompleted(evt PipelineCompletedEvent)
}

// PipelineCompletionBridge adapts generic work-task status updates into typed
// pipeline completion events for automation/orchestration listeners.
type PipelineCompletionBridge struct {
	mu        sync.RWMutex
	listeners []PipelineCompletionListener
}

func NewPipelineCompletionBridge() *PipelineCompletionBridge {
	return &PipelineCompletionBridge{listeners: make([]PipelineCompletionListener, 0)}
}

func (b *PipelineCompletionBridge) RegisterListener(listener PipelineCompletionListener) {
	if listener == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.listeners = append(b.listeners, listener)
}

func (b *PipelineCompletionBridge) OnWorkTaskStatusChanged(task *models.WorkTask) {
	evt, ok := pipelineCompletedEvent(task)
	if !ok {
		return
	}

	b.mu.RLock()
	listeners := append([]PipelineCompletionListener(nil), b.listeners...)
	b.mu.RUnlock()
	for _, listener := range listeners {
		listener.OnPipelineCompleted(evt)
	}
}

func pipelineCompletedEvent(task *models.WorkTask) (PipelineCompletedEvent, bool) {
	if task == nil || task.Type != models.WorkTaskTypePipelineExecution || task.Status != models.WorkTaskStatusCompleted {
		return PipelineCompletedEvent{}, false
	}
	pipelineID := task.TaskSpec.PipelineID
	if pipelineID == "" {
		return PipelineCompletedEvent{}, false
	}

	pipelineType := readPipelineType(task.ResultMetadata)
	if pipelineType == "" {
		pipelineType = readPipelineType(task.TaskSpec.Parameters)
	}
	if pipelineType == "" {
		return PipelineCompletedEvent{}, false
	}

	completedAt := time.Now().UTC()
	if task.CompletedAt != nil {
		completedAt = *task.CompletedAt
	}

	return PipelineCompletedEvent{
		ProjectID:      task.ProjectID,
		PipelineID:     pipelineID,
		PipelineType:   pipelineType,
		WorkTaskID:     task.ID,
		OutputLocation: task.OutputLocation,
		CompletedAt:    completedAt,
		Metadata:       cloneMap(task.ResultMetadata),
	}, true
}

func readPipelineType(values map[string]any) models.PipelineType {
	if values == nil {
		return ""
	}
	if raw, ok := values["pipeline_type"]; ok {
		switch typed := raw.(type) {
		case models.PipelineType:
			return typed
		case string:
			return models.PipelineType(typed)
		}
	}
	return ""
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
