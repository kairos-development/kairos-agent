package events

import (
	"context"
	"fmt"
	"sync"
)

// Handler processes domain events.
type Handler interface {
	Handle(ctx context.Context, event Event) error
}

// HandlerFunc is a function adapter for Handler.
type HandlerFunc func(ctx context.Context, event Event) error

func (f HandlerFunc) Handle(ctx context.Context, event Event) error {
	return f(ctx, event)
}

// Publisher distributes domain events to registered handlers.
// This is an in-process event bus for decoupling domain logic.
type Publisher struct {
	mu       sync.RWMutex
	handlers map[EventType][]Handler
}

// NewPublisher creates a new event publisher.
func NewPublisher() *Publisher {
	return &Publisher{
		handlers: make(map[EventType][]Handler),
	}
}

// Subscribe registers a handler for a specific event type.
func (p *Publisher) Subscribe(eventType EventType, handler Handler) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.handlers[eventType] = append(p.handlers[eventType], handler)
}

// SubscribeFunc registers a handler function for a specific event type.
func (p *Publisher) SubscribeFunc(eventType EventType, fn func(ctx context.Context, event Event) error) {
	p.Subscribe(eventType, HandlerFunc(fn))
}

// Publish sends an event to all registered handlers for its type.
// Handlers are invoked synchronously in registration order.
// If a handler returns an error, subsequent handlers are still invoked.
func (p *Publisher) Publish(ctx context.Context, event Event) error {
	if event == nil {
		return fmt.Errorf("event is nil")
	}

	p.mu.RLock()
	handlers := append([]Handler(nil), p.handlers[event.Type()]...)
	p.mu.RUnlock()

	var firstErr error
	for _, handler := range handlers {
		if err := safeHandle(ctx, handler, event); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	return firstErr
}

// PublishAsync sends an event to all handlers asynchronously.
// Errors are silently ignored. Use this for non-critical notifications.
func (p *Publisher) PublishAsync(ctx context.Context, event Event) {
	go func() {
		_ = p.Publish(ctx, event)
	}()
}

func safeHandle(ctx context.Context, handler Handler, event Event) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("event handler panic: %v", r)
		}
	}()

	return handler.Handle(ctx, event)
}
