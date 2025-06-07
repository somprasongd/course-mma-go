package domain

import (
	"context"
	"fmt"
	"sync"
)

var (
	ErrInvalidEvent = fmt.Errorf("invalid domain event")
)

// DomainEventHandler คือฟังก์ชันที่ handle event โดยเฉพาะ
type DomainEventHandler interface {
	Handle(ctx context.Context, event DomainEvent) error
}

// DomainEventDispatcher is the centralized event dispatcher
type DomainEventDispatcher interface {
	Register(eventType EventName, handler DomainEventHandler)
	Dispatch(ctx context.Context, events []DomainEvent) error
}

// simpleDomainEventDispatcher manages event handlers
type simpleDomainEventDispatcher struct {
	handlers map[EventName][]DomainEventHandler
	mu       sync.RWMutex
}

// NewSimpleDomainEventDispatcher creates a new dispatcher
func NewSimpleDomainEventDispatcher() DomainEventDispatcher {
	return &simpleDomainEventDispatcher{handlers: make(map[EventName][]DomainEventHandler)}
}

// Register handler สำหรับแต่ละ event name
func (d *simpleDomainEventDispatcher) Register(eventType EventName, handler DomainEventHandler) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.handlers[eventType] = append(d.handlers[eventType], handler)
}

// Dispatch จะ loop event และ call handler ที่ลงทะเบียนไว้
func (d *simpleDomainEventDispatcher) Dispatch(ctx context.Context, events []DomainEvent) error {
	for _, event := range events {
		d.mu.RLock()
		handlers := append([]DomainEventHandler(nil), d.handlers[event.EventName()]...) // เป็นการ copy slice เพื่อหลีกเลี่ยง race ถ้า handler ถูกแก้ไขระหว่าง dispatch
		d.mu.RUnlock()

		for _, handler := range handlers {
			err := func(h DomainEventHandler) error {
				err := h.Handle(ctx, event)
				if err != nil {
					return fmt.Errorf("error handling event %s: %w", event.EventName(), err)
				}
				return nil
			}(handler)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
