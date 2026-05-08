package tui

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/kairos-development/kairos-agent/internal/domain/events"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEventPublisher(t *testing.T) {
	ep := NewEventPublisher()

	require.NotNil(t, ep)
	assert.NotNil(t, ep.listeners)
	assert.Len(t, ep.listeners, 0)
}

func TestEventPublisher_Subscribe(t *testing.T) {
	ep := NewEventPublisher()

	listener := func(event events.Event) {}

	ep.Subscribe(listener)

	assert.Len(t, ep.listeners, 1)
}

func TestEventPublisher_Subscribe_Multiple(t *testing.T) {
	ep := NewEventPublisher()

	listener1 := func(event events.Event) {}
	listener2 := func(event events.Event) {}
	listener3 := func(event events.Event) {}

	ep.Subscribe(listener1)
	ep.Subscribe(listener2)
	ep.Subscribe(listener3)

	assert.Len(t, ep.listeners, 3)
}

func TestEventPublisher_Publish_SingleListener(t *testing.T) {
	ep := NewEventPublisher()
	ctx := context.Background()

	var receivedEvent events.Event
	var wg sync.WaitGroup
	wg.Add(1)

	listener := func(event events.Event) {
		receivedEvent = event
		wg.Done()
	}

	ep.Subscribe(listener)

	event := &events.OrderCreatedEvent{
		OrderID:  "order1",
		Symbol:   "BTCUSDT",
		Side:     "BUY",
		Quantity: decimal.NewFromFloat(0.1),
		Price:    decimal.NewFromInt(50000),
	}

	ep.Publish(ctx, event)

	// Wait for async listener to complete
	wg.Wait()

	require.NotNil(t, receivedEvent)
	orderEvent, ok := receivedEvent.(*events.OrderCreatedEvent)
	require.True(t, ok)
	assert.Equal(t, "order1", orderEvent.OrderID)
	assert.Equal(t, "BTCUSDT", orderEvent.Symbol)
}

func TestEventPublisher_Publish_MultipleListeners(t *testing.T) {
	ep := NewEventPublisher()
	ctx := context.Background()

	var wg sync.WaitGroup
	wg.Add(3)

	var received1, received2, received3 events.Event

	listener1 := func(event events.Event) {
		received1 = event
		wg.Done()
	}

	listener2 := func(event events.Event) {
		received2 = event
		wg.Done()
	}

	listener3 := func(event events.Event) {
		received3 = event
		wg.Done()
	}

	ep.Subscribe(listener1)
	ep.Subscribe(listener2)
	ep.Subscribe(listener3)

	event := &events.PositionOpenedEvent{
		PositionID:  "pos1",
		Symbol:      "BTCUSDT",
		Side:        "LONG",
		Quantity:    decimal.NewFromFloat(0.1),
		EntryPrice:  decimal.NewFromInt(50000),
		OpenedAtUTC: time.Now().UTC(),
	}

	ep.Publish(ctx, event)

	// Wait for all async listeners to complete
	wg.Wait()

	assert.NotNil(t, received1)
	assert.NotNil(t, received2)
	assert.NotNil(t, received3)

	// All listeners should receive the same event
	assert.Equal(t, event, received1)
	assert.Equal(t, event, received2)
	assert.Equal(t, event, received3)
}

func TestEventPublisher_Publish_NoListeners(t *testing.T) {
	ep := NewEventPublisher()
	ctx := context.Background()

	event := &events.OrderCreatedEvent{
		OrderID: "order1",
		Symbol:  "BTCUSDT",
	}

	// Should not panic with no listeners
	assert.NotPanics(t, func() {
		ep.Publish(ctx, event)
	})
}

func TestEventPublisher_Publish_AsyncExecution(t *testing.T) {
	ep := NewEventPublisher()
	ctx := context.Background()

	var wg sync.WaitGroup
	wg.Add(1)

	// Listener that takes some time
	listener := func(event events.Event) {
		time.Sleep(10 * time.Millisecond)
		wg.Done()
	}

	ep.Subscribe(listener)

	event := &events.OrderCreatedEvent{
		OrderID: "order1",
		Symbol:  "BTCUSDT",
	}

	start := time.Now()
	ep.Publish(ctx, event)
	elapsed := time.Since(start)

	// Publish should return immediately (async execution)
	assert.Less(t, elapsed, 5*time.Millisecond)

	// Wait for listener to complete
	wg.Wait()
}

func TestEventPublisher_Publish_ConcurrentPublish(t *testing.T) {
	ep := NewEventPublisher()
	ctx := context.Background()

	var mu sync.Mutex
	receivedCount := 0

	listener := func(event events.Event) {
		mu.Lock()
		receivedCount++
		mu.Unlock()
	}

	ep.Subscribe(listener)

	// Publish multiple events concurrently
	var wg sync.WaitGroup
	numEvents := 10
	wg.Add(numEvents)

	for i := 0; i < numEvents; i++ {
		go func(id int) {
			defer wg.Done()
			event := &events.OrderCreatedEvent{
				OrderID: string(rune('a' + id)),
				Symbol:  "BTCUSDT",
			}
			ep.Publish(ctx, event)
		}(i)
	}

	wg.Wait()

	// Give async listeners time to complete
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	count := receivedCount
	mu.Unlock()

	assert.Equal(t, numEvents, count)
}

func TestEventPublisher_Subscribe_ConcurrentSubscribe(t *testing.T) {
	ep := NewEventPublisher()

	var wg sync.WaitGroup
	numListeners := 10
	wg.Add(numListeners)

	for i := 0; i < numListeners; i++ {
		go func() {
			defer wg.Done()
			listener := func(event events.Event) {}
			ep.Subscribe(listener)
		}()
	}

	wg.Wait()

	assert.Len(t, ep.listeners, numListeners)
}

func TestEventPublisher_Publish_DifferentEventTypes(t *testing.T) {
	ep := NewEventPublisher()
	ctx := context.Background()

	var wg sync.WaitGroup
	var mu sync.Mutex
	receivedEvents := []events.Event{}

	listener := func(event events.Event) {
		mu.Lock()
		receivedEvents = append(receivedEvents, event)
		mu.Unlock()
		wg.Done()
	}

	ep.Subscribe(listener)

	events := []events.Event{
		&events.OrderCreatedEvent{OrderID: "1", Symbol: "BTCUSDT"},
		&events.PositionOpenedEvent{PositionID: "1", Symbol: "ETHUSDT"},
		&events.BalanceUpdatedEvent{Asset: "USDT", Total: decimal.NewFromInt(10000)},
		&events.RiskViolationEvent{ViolationType: "test"},
		&events.StrategyStartedEvent{StrategyID: "1", StrategyName: "test"},
	}

	wg.Add(len(events))

	for _, event := range events {
		ep.Publish(ctx, event)
	}

	wg.Wait()

	assert.Len(t, receivedEvents, len(events))
}

func TestEventPublisher_Publish_ListenerPanic(t *testing.T) {
	ep := NewEventPublisher()
	ctx := context.Background()

	var wg sync.WaitGroup
	wg.Add(1)

	// Listener that panics
	listener1 := func(event events.Event) {
		defer wg.Done()
		defer func() {
			if r := recover(); r != nil {
				// Expected panic, recover from it
			}
		}()
		panic("test panic")
	}

	// Normal listener
	var mu sync.Mutex
	var received2 events.Event
	wg.Add(1)
	listener2 := func(event events.Event) {
		mu.Lock()
		received2 = event
		mu.Unlock()
		wg.Done()
	}

	ep.Subscribe(listener1)
	ep.Subscribe(listener2)

	event := &events.OrderCreatedEvent{
		OrderID: "order1",
		Symbol:  "BTCUSDT",
	}

	// Should not panic even if one listener panics
	assert.NotPanics(t, func() {
		ep.Publish(ctx, event)
		wg.Wait()
	})

	// Second listener should still receive the event
	mu.Lock()
	defer mu.Unlock()
	assert.NotNil(t, received2)
}

func TestEventPublisher_ThreadSafety(t *testing.T) {
	ep := NewEventPublisher()
	ctx := context.Background()

	var wg sync.WaitGroup
	numOperations := 100

	// Concurrent subscribes
	wg.Add(numOperations)
	for i := 0; i < numOperations; i++ {
		go func() {
			defer wg.Done()
			listener := func(event events.Event) {}
			ep.Subscribe(listener)
		}()
	}

	// Concurrent publishes
	wg.Add(numOperations)
	for i := 0; i < numOperations; i++ {
		go func(id int) {
			defer wg.Done()
			event := &events.OrderCreatedEvent{
				OrderID: string(rune('a' + id%26)),
				Symbol:  "BTCUSDT",
			}
			ep.Publish(ctx, event)
		}(i)
	}

	wg.Wait()

	// Should have all listeners subscribed
	assert.Len(t, ep.listeners, numOperations)
}
