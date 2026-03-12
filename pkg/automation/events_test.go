package automation

import (
	"testing"
	"time"

	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

type capturePipelineEventListener struct {
	events []PipelineCompletedEvent
}

func (l *capturePipelineEventListener) OnPipelineCompleted(evt PipelineCompletedEvent) {
	l.events = append(l.events, evt)
}

func TestPipelineCompletionBridgeEmitsEventFromPersistedTaskResult(t *testing.T) {
	bridge := NewPipelineCompletionBridge()
	listener := &capturePipelineEventListener{}
	bridge.RegisterListener(listener)

	completedAt := time.Now().UTC()
	task := &models.WorkTask{
		ID:             "task-1",
		Type:           models.WorkTaskTypePipelineExecution,
		Status:         models.WorkTaskStatusCompleted,
		ProjectID:      "project-1",
		CompletedAt:    &completedAt,
		OutputLocation: "/tmp/pipeline/context.json",
		TaskSpec: models.TaskSpec{
			PipelineID: "pipeline-1",
			Parameters: map[string]any{"pipeline_type": string(models.PipelineTypeIngestion)},
		},
		ResultMetadata: map[string]any{"pipeline_type": string(models.PipelineTypeIngestion), "rows": 17},
	}

	bridge.OnWorkTaskStatusChanged(task)

	if len(listener.events) != 1 {
		t.Fatalf("expected one pipeline completion event, got %d", len(listener.events))
	}
	evt := listener.events[0]
	if evt.PipelineID != task.TaskSpec.PipelineID {
		t.Fatalf("expected pipeline id %s, got %s", task.TaskSpec.PipelineID, evt.PipelineID)
	}
	if evt.PipelineType != models.PipelineTypeIngestion {
		t.Fatalf("expected ingestion pipeline type, got %s", evt.PipelineType)
	}
	if evt.OutputLocation != task.OutputLocation {
		t.Fatalf("expected output location %s, got %s", task.OutputLocation, evt.OutputLocation)
	}
	if evt.Metadata["rows"] != 17 {
		t.Fatalf("expected metadata rows=17, got %#v", evt.Metadata)
	}
}

func TestPipelineCompletionBridgeIgnoresNonPipelineTasks(t *testing.T) {
	bridge := NewPipelineCompletionBridge()
	listener := &capturePipelineEventListener{}
	bridge.RegisterListener(listener)

	bridge.OnWorkTaskStatusChanged(&models.WorkTask{
		ID:        "task-2",
		Type:      models.WorkTaskTypeMLInference,
		Status:    models.WorkTaskStatusCompleted,
		ProjectID: "project-1",
	})

	if len(listener.events) != 0 {
		t.Fatalf("expected no events for non-pipeline task, got %d", len(listener.events))
	}
}
