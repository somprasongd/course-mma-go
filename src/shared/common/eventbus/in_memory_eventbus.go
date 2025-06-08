package eventbus

import (
	"context"
	"log"
	"sync"
)

// InMemoryEventBus is a simple event bus
type InMemoryEventBus struct {
	subscribers map[EventName][]IntegrationEventHandler
	mu          sync.RWMutex
}

// NewInMemoryEventBus creates an event bus instance
func NewInMemoryEventBus() *InMemoryEventBus {
	return &InMemoryEventBus{
		subscribers: make(map[EventName][]IntegrationEventHandler),
	}
}

// Subscribe registers a handler for a specific event
func (eb *InMemoryEventBus) Subscribe(eventName EventName, handler IntegrationEventHandler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	eb.subscribers[eventName] = append(eb.subscribers[eventName], handler)
}

// Publish sends an event to all subscribers
func (eb *InMemoryEventBus) Publish(ctx context.Context, event Event) error {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	handlers, ok := eb.subscribers[event.EventName()]
	if !ok {
		return nil
	}

	busCtx := context.WithValue(ctx, "name", "context in event bus")
	for _, handler := range handlers {
		go func(h IntegrationEventHandler) {
			err := h.Handle(busCtx, event)
			if err != nil {
				log.Printf("error handling event %s: %v", event.EventName(), err)
			}
		}(handler)
	}
	return nil
}
