package tui

import (
	"context"
	"sync"

	"github.com/kairos-development/kairos-agent/internal/domain/events"
)

// EventPublisher allows publishing domain events to the TUI.
// This is a simple implementation that can be replaced with a full event bus later.
type EventPublisher struct {
	mu        sync.RWMutex
	listeners []func(events.Event)
}

// NewEventPublisher creates a new event publisher.
func NewEventPublisher() *EventPublisher {
	return &EventPublisher{
		listeners: make([]func(events.Event), 0),
	}
}

// Subscribe adds a listener for domain events.
func (ep *EventPublisher) Subscribe(listener func(events.Event)) {
	ep.mu.Lock()
	defer ep.mu.Unlock()
	ep.listeners = append(ep.listeners, listener)
}

// Publish sends an event to all listeners.
func (ep *EventPublisher) Publish(ctx context.Context, event events.Event) {
	ep.mu.RLock()
	listeners := make([]func(events.Event), len(ep.listeners))
	copy(listeners, ep.listeners)
	ep.mu.RUnlock()

	for _, listener := range listeners {
		// Call listener in goroutine to avoid blocking
		go listener(event)
	}
}
