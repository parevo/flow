package engine

import (
	"context"
	"sync"

	"github.com/parevo/flow/internal/models"
)

// EventType represents different workflow events
type EventType string

const (
	EventTypeWorkflowStarted       EventType = "workflow.started"
	EventTypeWorkflowCompleted     EventType = "workflow.completed"
	EventTypeWorkflowFailed        EventType = "workflow.failed"
	EventTypeWorkflowCancelled     EventType = "workflow.cancelled"
	EventTypeStepStarted           EventType = "step.started"
	EventTypeStepCompleted         EventType = "step.completed"
	EventTypeStepFailed            EventType = "step.failed"
	EventTypeStepRetrying          EventType = "step.retrying"
	EventTypeStepWaitingSignal     EventType = "step.waiting_signal"
	EventTypeSignalReceived        EventType = "signal.received"
	EventTypeCompensationTriggered EventType = "compensation.triggered"
)

// Event represents a workflow event
type Event struct {
	Type      EventType
	Namespace string
	Execution *models.Execution
	Step      *models.ExecutionStep
	Workflow  *models.Workflow
	Metadata  map[string]interface{}
}

// EventHandler is the interface for handling workflow events
// Users implement this interface for custom notifications
type EventHandler interface {
	HandleEvent(ctx context.Context, event Event) error
}

// EventBus manages event handlers
type EventBus struct {
	mu       sync.RWMutex
	handlers map[EventType][]EventHandler
}

func NewEventBus() *EventBus {
	return &EventBus{
		handlers: make(map[EventType][]EventHandler),
	}
}

// RegisterHandler adds an event handler for specific event type
func (eb *EventBus) RegisterHandler(eventType EventType, handler EventHandler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	eb.handlers[eventType] = append(eb.handlers[eventType], handler)
}

// RegisterGlobalHandler adds a handler for ALL events
func (eb *EventBus) RegisterGlobalHandler(handler EventHandler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	for eventType := range eb.handlers {
		eb.handlers[eventType] = append(eb.handlers[eventType], handler)
	}

	// Also register for event types that might be added later
	allEventTypes := []EventType{
		EventTypeWorkflowStarted,
		EventTypeWorkflowCompleted,
		EventTypeWorkflowFailed,
		EventTypeWorkflowCancelled,
		EventTypeStepStarted,
		EventTypeStepCompleted,
		EventTypeStepFailed,
		EventTypeStepRetrying,
		EventTypeStepWaitingSignal,
		EventTypeSignalReceived,
		EventTypeCompensationTriggered,
	}

	for _, eventType := range allEventTypes {
		if _, exists := eb.handlers[eventType]; !exists {
			eb.handlers[eventType] = []EventHandler{}
		}
		eb.handlers[eventType] = append(eb.handlers[eventType], handler)
	}
}

// Emit sends event to all registered handlers (async, non-blocking)
func (eb *EventBus) Emit(ctx context.Context, event Event) {
	eb.mu.RLock()
	handlersList := eb.handlers[event.Type]
	if len(handlersList) == 0 {
		eb.mu.RUnlock()
		return
	}
	handlers := make([]EventHandler, len(handlersList))
	copy(handlers, handlersList)
	eb.mu.RUnlock()

	// Execute handlers asynchronously to not block workflow execution
	for _, handler := range handlers {
		go func(h EventHandler) {
			// Intentionally ignoring error to not block workflow execution
			_ = h.HandleEvent(ctx, event)
		}(handler)
	}
}

// EmitSync sends event synchronously (blocking, waits for all handlers)
func (eb *EventBus) EmitSync(ctx context.Context, event Event) []error {
	eb.mu.RLock()
	handlersList := eb.handlers[event.Type]
	if len(handlersList) == 0 {
		eb.mu.RUnlock()
		return nil
	}
	handlers := make([]EventHandler, len(handlersList))
	copy(handlers, handlersList)
	eb.mu.RUnlock()

	var errors []error
	for _, handler := range handlers {
		if err := handler.HandleEvent(ctx, event); err != nil {
			errors = append(errors, err)
		}
	}

	return errors
}
