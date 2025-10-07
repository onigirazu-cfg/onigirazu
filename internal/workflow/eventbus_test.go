package workflow

import (
	"sync"
	"testing"
	"time"
)

// TestNewEventBus tests event bus creation
func TestNewEventBus(t *testing.T) {
	eb := NewEventBus()

	if eb == nil {
		t.Fatal("Expected event bus to be created")
	}

	if eb.subscribers == nil {
		t.Error("Expected subscribers map to be initialized")
	}

	if eb.eventLog == nil {
		t.Error("Expected event log to be initialized")
	}

	if eb.maxLogSize != 1000 {
		t.Errorf("Expected max log size 1000, got %d", eb.maxLogSize)
	}
}

// TestEventBus_Subscribe tests event subscription
func TestEventBus_Subscribe(t *testing.T) {
	eb := NewEventBus()

	handlerCalled := false
	handler := func(event Event) {
		handlerCalled = true
	}

	eb.Subscribe("test.event", handler)

	// Check that handler was added
	eb.mutex.RLock()
	handlers, exists := eb.subscribers["test.event"]
	eb.mutex.RUnlock()

	if !exists {
		t.Fatal("Expected event type to be registered")
	}

	if len(handlers) != 1 {
		t.Errorf("Expected 1 handler, got %d", len(handlers))
	}
}

// TestEventBus_Subscribe_Multiple tests multiple subscriptions
func TestEventBus_Subscribe_Multiple(t *testing.T) {
	eb := NewEventBus()

	handler1 := func(event Event) {}
	handler2 := func(event Event) {}
	handler3 := func(event Event) {}

	eb.Subscribe("test.event", handler1)
	eb.Subscribe("test.event", handler2)
	eb.Subscribe("test.event", handler3)

	eb.mutex.RLock()
	handlers := eb.subscribers["test.event"]
	eb.mutex.RUnlock()

	if len(handlers) != 3 {
		t.Errorf("Expected 3 handlers, got %d", len(handlers))
	}
}

// TestEventBus_Unsubscribe tests event unsubscription
func TestEventBus_Unsubscribe(t *testing.T) {
	eb := NewEventBus()

	handler := func(event Event) {}
	eb.Subscribe("test.event", handler)

	// Verify subscription exists
	eb.mutex.RLock()
	_, exists := eb.subscribers["test.event"]
	eb.mutex.RUnlock()

	if !exists {
		t.Fatal("Expected subscription to exist")
	}

	// Unsubscribe
	eb.Unsubscribe("test.event")

	// Verify subscription removed
	eb.mutex.RLock()
	_, exists = eb.subscribers["test.event"]
	eb.mutex.RUnlock()

	if exists {
		t.Error("Expected subscription to be removed")
	}
}

// TestEventBus_Publish tests event publishing
func TestEventBus_Publish(t *testing.T) {
	eb := NewEventBus()

	var receivedEvent Event
	var wg sync.WaitGroup
	wg.Add(1)

	handler := func(event Event) {
		receivedEvent = event
		wg.Done()
	}

	eb.Subscribe("test.event", handler)

	testData := map[string]interface{}{
		"key": "value",
	}

	eb.Publish("test.event", testData)

	// Wait for handler to be called (with timeout)
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(1 * time.Second):
		t.Fatal("Handler was not called within timeout")
	}

	// Verify event data
	if receivedEvent.Type != "test.event" {
		t.Errorf("Expected event type 'test.event', got '%s'", receivedEvent.Type)
	}

	if receivedEvent.Data == nil {
		t.Fatal("Expected event data to be set")
	}

	data, ok := receivedEvent.Data.(map[string]interface{})
	if !ok {
		t.Fatal("Expected data to be map[string]interface{}")
	}

	if data["key"] != "value" {
		t.Errorf("Expected data['key'] = 'value', got '%v'", data["key"])
	}

	if receivedEvent.Source != "workflow_orchestrator" {
		t.Errorf("Expected source 'workflow_orchestrator', got '%s'", receivedEvent.Source)
	}

	if receivedEvent.ID == "" {
		t.Error("Expected event ID to be generated")
	}

	if receivedEvent.Timestamp.IsZero() {
		t.Error("Expected timestamp to be set")
	}

	if receivedEvent.Metadata == nil {
		t.Error("Expected metadata to be initialized")
	}
}

// TestEventBus_Publish_MultipleHandlers tests publishing to multiple handlers
func TestEventBus_Publish_MultipleHandlers(t *testing.T) {
	eb := NewEventBus()

	var wg sync.WaitGroup
	wg.Add(3)

	callCount := int32(0)
	var mu sync.Mutex

	handler := func(event Event) {
		mu.Lock()
		callCount++
		mu.Unlock()
		wg.Done()
	}

	eb.Subscribe("test.event", handler)
	eb.Subscribe("test.event", handler)
	eb.Subscribe("test.event", handler)

	eb.Publish("test.event", "test data")

	// Wait for all handlers
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(1 * time.Second):
		t.Fatal("Not all handlers were called within timeout")
	}

	mu.Lock()
	finalCount := callCount
	mu.Unlock()

	if finalCount != 3 {
		t.Errorf("Expected 3 handler calls, got %d", finalCount)
	}
}

// TestEventBus_Publish_NoSubscribers tests publishing without subscribers
func TestEventBus_Publish_NoSubscribers(t *testing.T) {
	eb := NewEventBus()

	// Should not panic
	eb.Publish("test.event", "test data")

	// Verify event was logged
	log := eb.GetEventLog()
	if len(log) != 1 {
		t.Errorf("Expected 1 event in log, got %d", len(log))
	}
}

// TestEventBus_GetEventLog tests event log retrieval
func TestEventBus_GetEventLog(t *testing.T) {
	eb := NewEventBus()

	// Publish some events
	eb.Publish("event1", "data1")
	eb.Publish("event2", "data2")
	eb.Publish("event3", "data3")

	log := eb.GetEventLog()

	if len(log) != 3 {
		t.Errorf("Expected 3 events in log, got %d", len(log))
	}

	// Verify events are in order
	if log[0].Type != "event1" {
		t.Errorf("Expected first event type 'event1', got '%s'", log[0].Type)
	}

	if log[1].Type != "event2" {
		t.Errorf("Expected second event type 'event2', got '%s'", log[1].Type)
	}

	if log[2].Type != "event3" {
		t.Errorf("Expected third event type 'event3', got '%s'", log[2].Type)
	}

	// Verify log is a copy (modifying it shouldn't affect original)
	log[0].Type = "modified"
	newLog := eb.GetEventLog()
	if newLog[0].Type == "modified" {
		t.Error("Expected log to be a copy, not a reference")
	}
}

// TestEventBus_LogTrimming tests log size limiting
func TestEventBus_LogTrimming(t *testing.T) {
	eb := NewEventBus()
	eb.maxLogSize = 10

	// Publish more events than max size
	for i := 0; i < 15; i++ {
		eb.Publish("test.event", i)
	}

	log := eb.GetEventLog()

	if len(log) != 10 {
		t.Errorf("Expected log size 10, got %d", len(log))
	}

	// Verify oldest events were removed (should have events 5-14)
	firstEventData, ok := log[0].Data.(int)
	if !ok {
		t.Fatal("Expected data to be int")
	}

	if firstEventData != 5 {
		t.Errorf("Expected first event data 5, got %d", firstEventData)
	}

	lastEventData, ok := log[9].Data.(int)
	if !ok {
		t.Fatal("Expected data to be int")
	}

	if lastEventData != 14 {
		t.Errorf("Expected last event data 14, got %d", lastEventData)
	}
}

// TestEventBus_ConcurrentPublish tests concurrent event publishing
func TestEventBus_ConcurrentPublish(t *testing.T) {
	eb := NewEventBus()

	var wg sync.WaitGroup
	eventCount := 100

	// Publish events concurrently
	for i := 0; i < eventCount; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			eb.Publish("test.event", id)
		}(i)
	}

	wg.Wait()

	log := eb.GetEventLog()
	if len(log) != eventCount {
		t.Errorf("Expected %d events in log, got %d", eventCount, len(log))
	}
}

// TestEventBus_ConcurrentSubscribe tests concurrent subscription
func TestEventBus_ConcurrentSubscribe(t *testing.T) {
	eb := NewEventBus()

	var wg sync.WaitGroup
	handlerCount := 50

	handler := func(event Event) {}

	// Subscribe concurrently
	for i := 0; i < handlerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			eb.Subscribe("test.event", handler)
		}()
	}

	wg.Wait()

	eb.mutex.RLock()
	handlers := eb.subscribers["test.event"]
	eb.mutex.RUnlock()

	if len(handlers) != handlerCount {
		t.Errorf("Expected %d handlers, got %d", handlerCount, len(handlers))
	}
}

// TestGenerateEventID tests event ID generation
func TestGenerateEventID(t *testing.T) {
	id1 := generateEventID()
	time.Sleep(1 * time.Millisecond)
	id2 := generateEventID()

	if id1 == "" {
		t.Error("Expected non-empty event ID")
	}

	if id2 == "" {
		t.Error("Expected non-empty event ID")
	}

	if id1 == id2 {
		t.Error("Expected unique event IDs")
	}

	// Check format
	if len(id1) < 6 {
		t.Error("Expected event ID to have reasonable length")
	}
}

// TestEventBus_HandlerPanic tests that handler panic doesn't crash the system
func TestEventBus_HandlerPanic(t *testing.T) {
	eb := NewEventBus()

	var wg sync.WaitGroup
	wg.Add(1)

	panicHandler := func(event Event) {
		panic("handler panic")
	}

	normalHandler := func(event Event) {
		wg.Done()
	}

	eb.Subscribe("test.event", panicHandler)
	eb.Subscribe("test.event", normalHandler)

	// Should not crash the test
	eb.Publish("test.event", "test data")

	// Wait for normal handler
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success - normal handler was called despite panic in other handler
	case <-time.After(1 * time.Second):
		t.Fatal("Normal handler was not called")
	}
}

// BenchmarkEventBus_Publish benchmarks event publishing
func BenchmarkEventBus_Publish(b *testing.B) {
	eb := NewEventBus()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		eb.Publish("test.event", "test data")
	}
}

// BenchmarkEventBus_PublishWithSubscribers benchmarks publishing with subscribers
func BenchmarkEventBus_PublishWithSubscribers(b *testing.B) {
	eb := NewEventBus()

	handler := func(event Event) {
		// Minimal handler
	}

	for i := 0; i < 10; i++ {
		eb.Subscribe("test.event", handler)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		eb.Publish("test.event", "test data")
	}
}

// BenchmarkEventBus_Subscribe benchmarks subscription
func BenchmarkEventBus_Subscribe(b *testing.B) {
	eb := NewEventBus()

	handler := func(event Event) {}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		eb.Subscribe("test.event", handler)
	}
}
