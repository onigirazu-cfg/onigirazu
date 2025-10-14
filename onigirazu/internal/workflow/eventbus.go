package workflow

import (
	"fmt"
	"sync"
	"time"
)

// EventBus manages event publishing and subscription
type EventBus struct {
	subscribers map[string][]EventHandler
	mutex       sync.RWMutex
	eventLog    []Event
	maxLogSize  int
}

// EventHandler handles events
type EventHandler func(event Event)

// Event represents an event in the system
type Event struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Data      interface{}            `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
	Source    string                 `json:"source"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// NewEventBus creates a new event bus
func NewEventBus() *EventBus {
	return &EventBus{
		subscribers: make(map[string][]EventHandler),
		eventLog:    make([]Event, 0),
		maxLogSize:  1000,
	}
}

// Subscribe subscribes to events of a specific type
func (eb *EventBus) Subscribe(eventType string, handler EventHandler) {
	eb.mutex.Lock()
	defer eb.mutex.Unlock()

	if _, exists := eb.subscribers[eventType]; !exists {
		eb.subscribers[eventType] = make([]EventHandler, 0)
	}

	eb.subscribers[eventType] = append(eb.subscribers[eventType], handler)
}

// Unsubscribe removes all handlers for an event type
func (eb *EventBus) Unsubscribe(eventType string) {
	eb.mutex.Lock()
	defer eb.mutex.Unlock()

	delete(eb.subscribers, eventType)
}

// Publish publishes an event
func (eb *EventBus) Publish(eventType string, data interface{}) {
	event := Event{
		ID:        generateEventID(),
		Type:      eventType,
		Data:      data,
		Timestamp: time.Now(),
		Source:    "workflow_orchestrator",
		Metadata:  make(map[string]interface{}),
	}

	// Add to event log
	eb.addToLog(event)

	// Notify subscribers
	eb.mutex.RLock()
	handlers, exists := eb.subscribers[eventType]
	eb.mutex.RUnlock()

	if exists {
		for _, handler := range handlers {
			go func(h EventHandler) {
				defer func() {
					if r := recover(); r != nil {
						// Log panic but don't crash
						// In production, you might want to log this properly
						_ = r
					}
				}()
				h(event)
			}(handler) // Handle asynchronously with panic recovery
		}
	}
}

// GetEventLog returns the event log
func (eb *EventBus) GetEventLog() []Event {
	eb.mutex.RLock()
	defer eb.mutex.RUnlock()

	log := make([]Event, len(eb.eventLog))
	copy(log, eb.eventLog)
	return log
}

// addToLog adds an event to the log
func (eb *EventBus) addToLog(event Event) {
	eb.mutex.Lock()
	defer eb.mutex.Unlock()

	eb.eventLog = append(eb.eventLog, event)

	// Trim log if it exceeds max size
	if len(eb.eventLog) > eb.maxLogSize {
		eb.eventLog = eb.eventLog[len(eb.eventLog)-eb.maxLogSize:]
	}
}

// generateEventID generates a unique event ID
func generateEventID() string {
	return fmt.Sprintf("event_%d", time.Now().UnixNano())
}
