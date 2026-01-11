package utils

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestEventBus_PublishSubscribe(t *testing.T) {
	eb := NewEventBus()
	defer eb.Clear()

	receivedEvent := false
	var receivedPayload map[string]any

	handler := func(event Event) error {
		receivedEvent = true
		receivedPayload = event.Payload
		return nil
	}

	eb.Subscribe("test.event", handler)

	event := Event{
		Type:   "test.event",
		Source: "test",
		Payload: map[string]any{
			"key": "value",
		},
	}

	eb.PublishSync(event)

	if !receivedEvent {
		t.Error("Expected event to be received")
	}

	if receivedPayload["key"] != "value" {
		t.Errorf("Expected payload key=value, got %v", receivedPayload)
	}
}

func TestEventBus_MultipleHandlers(t *testing.T) {
	eb := NewEventBus()
	defer eb.Clear()

	var count int32

	handler1 := func(event Event) error {
		atomic.AddInt32(&count, 1)
		return nil
	}

	handler2 := func(event Event) error {
		atomic.AddInt32(&count, 1)
		return nil
	}

	handler3 := func(event Event) error {
		atomic.AddInt32(&count, 1)
		return nil
	}

	eb.Subscribe("multi.event", handler1)
	eb.Subscribe("multi.event", handler2)
	eb.Subscribe("multi.event", handler3)

	event := Event{
		Type:   "multi.event",
		Source: "test",
	}

	eb.PublishSync(event)

	if atomic.LoadInt32(&count) != 3 {
		t.Errorf("Expected 3 handlers to execute, got %d", count)
	}
}

func TestEventBus_DifferentEventTypes(t *testing.T) {
	eb := NewEventBus()
	defer eb.Clear()

	var eventType1Received, eventType2Received bool

	handler1 := func(event Event) error {
		eventType1Received = true
		return nil
	}

	handler2 := func(event Event) error {
		eventType2Received = true
		return nil
	}

	eb.Subscribe("event.type1", handler1)
	eb.Subscribe("event.type2", handler2)

	// Publish event type 1
	eb.PublishSync(Event{Type: "event.type1", Source: "test"})

	if !eventType1Received {
		t.Error("Expected event type 1 to be received")
	}

	if eventType2Received {
		t.Error("Did not expect event type 2 to be received")
	}

	// Reset and publish event type 2
	eventType1Received = false
	eventType2Received = false

	eb.PublishSync(Event{Type: "event.type2", Source: "test"})

	if eventType1Received {
		t.Error("Did not expect event type 1 to be received")
	}

	if !eventType2Received {
		t.Error("Expected event type 2 to be received")
	}
}

func TestEventBus_AsyncPublish(t *testing.T) {
	eb := NewEventBus()
	defer eb.Clear()

	var count int32
	var wg sync.WaitGroup

	wg.Add(5)

	handler := func(event Event) error {
		atomic.AddInt32(&count, 1)
		wg.Done()
		return nil
	}

	// Subscribe 5 handlers
	for i := 0; i < 5; i++ {
		eb.Subscribe("async.event", handler)
	}

	// Publish asynchronously
	eb.Publish(Event{Type: "async.event", Source: "test"})

	// Wait for all handlers to complete
	wg.Wait()

	if atomic.LoadInt32(&count) != 5 {
		t.Errorf("Expected 5 handlers to execute, got %d", count)
	}
}

func TestEventBus_HandlerError(t *testing.T) {
	eb := NewEventBus()
	defer eb.Clear()

	errorHandler := func(event Event) error {
		return errors.New("handler error")
	}

	eb.Subscribe("error.event", errorHandler)

	// Should not panic when handler returns error
	err := eb.PublishSync(Event{Type: "error.event", Source: "test"})

	if err == nil {
		t.Error("Expected error from handler")
	}
}

func TestEventBus_Unsubscribe(t *testing.T) {
	eb := NewEventBus()
	defer eb.Clear()

	receivedEvent := false

	handler := func(event Event) error {
		receivedEvent = true
		return nil
	}

	eb.Subscribe("test.event", handler)

	if eb.GetSubscriberCount("test.event") != 1 {
		t.Errorf("Expected 1 subscriber, got %d", eb.GetSubscriberCount("test.event"))
	}

	eb.Unsubscribe("test.event")

	if eb.GetSubscriberCount("test.event") != 0 {
		t.Errorf("Expected 0 subscribers after unsubscribe, got %d", eb.GetSubscriberCount("test.event"))
	}

	eb.PublishSync(Event{Type: "test.event", Source: "test"})

	if receivedEvent {
		t.Error("Did not expect event to be received after unsubscribe")
	}
}

func TestEventBus_Clear(t *testing.T) {
	eb := NewEventBus()

	eb.Subscribe("event1", func(e Event) error { return nil })
	eb.Subscribe("event2", func(e Event) error { return nil })
	eb.Subscribe("event3", func(e Event) error { return nil })

	if eb.GetSubscriberCount("event1") == 0 {
		t.Error("Expected subscribers before clear")
	}

	eb.Clear()

	if eb.GetSubscriberCount("event1") != 0 || eb.GetSubscriberCount("event2") != 0 || eb.GetSubscriberCount("event3") != 0 {
		t.Error("Expected no subscribers after clear")
	}
}

func TestEventBus_NoHandlers(t *testing.T) {
	eb := NewEventBus()
	defer eb.Clear()

	// Should not panic when publishing to event type with no handlers
	eb.Publish(Event{Type: "no.handlers", Source: "test"})
	eb.PublishSync(Event{Type: "no.handlers", Source: "test"})
}

func TestEventBus_TimestampAutoSet(t *testing.T) {
	eb := NewEventBus()
	defer eb.Clear()

	var receivedTimestamp time.Time

	handler := func(event Event) error {
		receivedTimestamp = event.Timestamp
		return nil
	}

	eb.Subscribe("timestamp.event", handler)

	before := time.Now()
	eb.PublishSync(Event{Type: "timestamp.event", Source: "test"})
	after := time.Now()

	if receivedTimestamp.IsZero() {
		t.Error("Expected timestamp to be set automatically")
	}

	if receivedTimestamp.Before(before) || receivedTimestamp.After(after) {
		t.Errorf("Timestamp %v not within expected range %v - %v", receivedTimestamp, before, after)
	}
}

func TestEventBus_ConcurrentPublish(t *testing.T) {
	eb := NewEventBus()
	defer eb.Clear()

	var count int32

	handler := func(event Event) error {
		atomic.AddInt32(&count, 1)
		return nil
	}

	eb.Subscribe("concurrent.event", handler)

	// Publish concurrently from multiple goroutines
	var wg sync.WaitGroup
	concurrency := 100

	wg.Add(concurrency)

	for i := 0; i < concurrency; i++ {
		go func() {
			defer wg.Done()
			eb.PublishSync(Event{Type: "concurrent.event", Source: "test"})
		}()
	}

	wg.Wait()

	if atomic.LoadInt32(&count) != int32(concurrency) {
		t.Errorf("Expected %d events, got %d", concurrency, count)
	}
}

func TestEventBus_GlobalInstance(t *testing.T) {
	// Get global instance
	eb1 := GetEventBus()
	eb2 := GetEventBus()

	// Should be the same instance
	if eb1 != eb2 {
		t.Error("Expected GetEventBus to return the same instance")
	}

	// Clean up
	eb1.Clear()
}

func TestEventBus_PayloadData(t *testing.T) {
	eb := NewEventBus()
	defer eb.Clear()

	var receivedPipelineID string
	var receivedData any

	handler := func(event Event) error {
		receivedPipelineID = event.Payload["pipeline_id"].(string)
		receivedData = event.Payload["data"]
		return nil
	}

	eb.Subscribe("pipeline.completed", handler)

	event := Event{
		Type:   "pipeline.completed",
		Source: "pipeline-service",
		Payload: map[string]any{
			"pipeline_id": "pipeline-123",
			"data": map[string]any{
				"rows_processed": 1000,
				"status":         "success",
			},
		},
	}

	eb.PublishSync(event)

	if receivedPipelineID != "pipeline-123" {
		t.Errorf("Expected pipeline_id=pipeline-123, got %s", receivedPipelineID)
	}

	dataMap := receivedData.(map[string]any)
	if dataMap["rows_processed"] != 1000 {
		t.Errorf("Expected rows_processed=1000, got %v", dataMap["rows_processed"])
	}
}

// Benchmark tests
func BenchmarkEventBus_Publish(b *testing.B) {
	eb := NewEventBus()
	defer eb.Clear()

	handler := func(event Event) error {
		return nil
	}

	eb.Subscribe("bench.event", handler)

	event := Event{
		Type:   "bench.event",
		Source: "benchmark",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		eb.Publish(event)
	}
}

func BenchmarkEventBus_PublishSync(b *testing.B) {
	eb := NewEventBus()
	defer eb.Clear()

	handler := func(event Event) error {
		return nil
	}

	eb.Subscribe("bench.event", handler)

	event := Event{
		Type:   "bench.event",
		Source: "benchmark",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		eb.PublishSync(event)
	}
}
