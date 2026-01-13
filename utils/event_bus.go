package utils

import (
	"sync"
	"time"
)

// Event represents an event in the system
type Event struct {
	Type      string         // Event type (e.g., "pipeline.completed", "anomaly.detected")
	Source    string         // Component that emitted the event
	Payload   map[string]any // Event data
	Timestamp time.Time      // When the event occurred
}

// EventHandler is a function that handles events
type EventHandler func(Event) error

// EventBus manages event publication and subscription
type EventBus struct {
	handlers map[string][]EventHandler
	mu       sync.RWMutex
	logger   *Logger
}

// globalEventBus is the singleton event bus instance
var (
	globalEventBus     *EventBus
	globalEventBusOnce sync.Once
)

// GetEventBus returns the global event bus instance
func GetEventBus() *EventBus {
	globalEventBusOnce.Do(func() {
		globalEventBus = &EventBus{
			handlers: make(map[string][]EventHandler),
			logger:   GetLogger(),
		}
		globalEventBus.logger.Info("Event bus initialized")
	})
	return globalEventBus
}

// NewEventBus creates a new event bus (for testing)
func NewEventBus() *EventBus {
	return &EventBus{
		handlers: make(map[string][]EventHandler),
		logger:   GetLogger(),
	}
}

// Publish publishes an event to all registered handlers
func (eb *EventBus) Publish(event Event) {
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	eb.mu.RLock()
	handlers, exists := eb.handlers[event.Type]
	eb.mu.RUnlock()

	if !exists || len(handlers) == 0 {
		eb.logger.Debug("No handlers registered for event type",
			String("event_type", event.Type),
			String("source", event.Source))
		return
	}

	eb.logger.Info("Publishing event",
		String("event_type", event.Type),
		String("source", event.Source),
		Int("handler_count", len(handlers)))

	// Execute handlers asynchronously
	for _, handler := range handlers {
		go func(h EventHandler, evt Event) {
			if err := h(evt); err != nil {
				eb.logger.Error("Event handler error",
					err,
					String("event_type", evt.Type),
					String("source", evt.Source))
			}
		}(handler, event)
	}
}

// PublishSync publishes an event synchronously (waits for all handlers to complete)
func (eb *EventBus) PublishSync(event Event) error {
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	eb.mu.RLock()
	handlers, exists := eb.handlers[event.Type]
	eb.mu.RUnlock()

	if !exists || len(handlers) == 0 {
		return nil
	}

	eb.logger.Info("Publishing event synchronously",
		String("event_type", event.Type),
		String("source", event.Source),
		Int("handler_count", len(handlers)))

	// Execute handlers synchronously
	for _, handler := range handlers {
		if err := handler(event); err != nil {
			eb.logger.Error("Event handler error",
				err,
				String("event_type", event.Type),
				String("source", event.Source))
			return err
		}
	}

	return nil
}

// Subscribe registers a handler for a specific event type
func (eb *EventBus) Subscribe(eventType string, handler EventHandler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	eb.handlers[eventType] = append(eb.handlers[eventType], handler)

	eb.logger.Info("Event handler subscribed",
		String("event_type", eventType),
		Int("total_handlers", len(eb.handlers[eventType])))
}

// Unsubscribe removes all handlers for an event type
func (eb *EventBus) Unsubscribe(eventType string) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	count := len(eb.handlers[eventType])
	delete(eb.handlers, eventType)

	eb.logger.Info("Event handlers unsubscribed",
		String("event_type", eventType),
		Int("removed_handlers", count))
}

// Clear removes all event handlers (useful for testing)
func (eb *EventBus) Clear() {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	totalHandlers := 0
	for _, handlers := range eb.handlers {
		totalHandlers += len(handlers)
	}

	eb.handlers = make(map[string][]EventHandler)

	eb.logger.Info("Event bus cleared",
		Int("removed_handlers", totalHandlers))
}

// GetSubscriberCount returns the number of handlers for an event type
func (eb *EventBus) GetSubscriberCount(eventType string) int {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	return len(eb.handlers[eventType])
}

// Common event types (constants for consistency)
const (
	// Pipeline events
	EventPipelineStarted   = "pipeline.started"
	EventPipelineCompleted = "pipeline.completed"
	EventPipelineFailed    = "pipeline.failed"

	// Ontology events
	EventOntologyCreated     = "ontology.created"
	EventOntologyUpdated     = "ontology.updated"
	EventExtractionStarted   = "extraction.started"
	EventExtractionCompleted = "extraction.completed"
	EventEntitiesExtracted   = "entities.extracted"

	// ML events
	EventModelTrainingStarted   = "model.training.started"
	EventModelTrainingCompleted = "model.training.completed"
	EventModelTrainingFailed    = "model.training.failed"
	EventPredictionMade         = "prediction.made"

	// Digital twin events
	EventTwinCreated        = "twin.created"
	EventTwinUpdated        = "twin.updated"
	EventSimulationStarted  = "simulation.started"
	EventSimulationComplete = "simulation.completed"

	// Monitoring events
	EventAnomalyDetected   = "anomaly.detected"
	EventAlertCreated      = "alert.created"
	EventThresholdExceeded = "threshold.exceeded"
	EventMonitoringJobRun  = "monitoring.job.run"

	// Job events
	EventJobScheduled = "job.scheduled"
	EventJobStarted   = "job.started"
	EventJobCompleted = "job.completed"
	EventJobFailed    = "job.failed"
)
