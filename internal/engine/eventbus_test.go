package engine

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// TestEventBus_NewEventBusWithNilLogger tests event bus creation with nil logger.
func TestEventBus_NewEventBusWithNilLogger(t *testing.T) {
	ctx := context.Background()

	eb := NewEventBus(ctx, nil)

	assert.NotNil(t, eb)
	assert.NotNil(t, eb.logger)
	eb.Shutdown()
}

// TestEventBus_SubscribeAndPublish tests basic subscribe and publish functionality.
func TestEventBus_SubscribeAndPublish(t *testing.T) {
	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel) // Suppress logs in tests

	eb := NewEventBus(ctx, logger)
	defer eb.Shutdown()

	// Create test event
	testEvent := &BaseEvent{
		EventType: EventTypeAlert,
		EventQoS:  QoS1,
	}

	// Subscribe
	ch := make(chan Event, 10)
	subscriber := SubscriberFunc(func(ctx context.Context, event Event) {
		ch <- event
	})

	id := eb.Subscribe(EventTypeAlert, QoS1, subscriber)
	assert.NotZero(t, id)

	// Publish event
	eb.Publish(testEvent)

	// Wait for event
	select {
	case received := <-ch:
		assert.Equal(t, EventTypeAlert, received.Type())
		assert.Equal(t, QoS1, received.QoS())
	case <-time.After(1 * time.Second):
		t.Fatal("Event not received")
	}
}

// TestEventBus_Unsubscribe tests unsubscribe functionality.
func TestEventBus_Unsubscribe(t *testing.T) {
	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	eb := NewEventBus(ctx, logger)
	defer eb.Shutdown()

	// Subscribe
	ch := make(chan Event, 10)
	subscriber := SubscriberFunc(func(ctx context.Context, event Event) {
		ch <- event
	})

	id := eb.Subscribe(EventTypeAlert, QoS1, subscriber)

	// Unsubscribe
	eb.Unsubscribe(EventTypeAlert, id)

	// Publish event
	testEvent := &BaseEvent{
		EventType: EventTypeAlert,
		EventQoS:  QoS1,
	}
	eb.Publish(testEvent)

	// Should not receive event
	select {
	case <-ch:
		t.Fatal("Received event after unsubscribe")
	case <-time.After(100 * time.Millisecond):
		// Expected - no event received
	}
}

// TestEventBus_QoS0_DropsOnSlowConsumer tests QoS0 drops events when consumer is slow.
func TestEventBus_QoS0_DropsOnSlowConsumer(t *testing.T) {
	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	eb := NewEventBus(ctx, logger)
	defer eb.Shutdown()

	// Create slow subscriber
	var received int32
	subscriber := SubscriberFunc(func(ctx context.Context, event Event) {
		atomic.AddInt32(&received, 1)
		time.Sleep(50 * time.Millisecond) // Slow consumer
	})

	eb.Subscribe(EventTypeMarketData, QoS0, subscriber)

	// Give subscriber time to start
	time.Sleep(100 * time.Millisecond)

	// Publish many events quickly
	for i := 0; i < 50; i++ {
		testEvent := &BaseEvent{
			EventType: EventTypeMarketData,
			EventQoS:  QoS0,
		}
		eb.Publish(testEvent)
	}

	// Wait for processing
	time.Sleep(500 * time.Millisecond)

	// Should have dropped some events (buffer is only 10)
	receivedCount := atomic.LoadInt32(&received)
	assert.Less(t, receivedCount, int32(50), "QoS0 should drop events on slow consumer")
	assert.Greater(t, receivedCount, int32(0), "Should receive at least some events")

	// Check dropped count
	dropped := eb.GetDroppedEventCount()
	assert.Greater(t, dropped, uint64(0), "Should have dropped events")
}

// TestEventBus_QoS1_GuaranteedDelivery tests QoS1 guarantees delivery.
func TestEventBus_QoS1_GuaranteedDelivery(t *testing.T) {
	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	eb := NewEventBus(ctx, logger)
	defer eb.Shutdown()

	// Create subscriber
	var received int32
	var wg sync.WaitGroup
	subscriber := SubscriberFunc(func(ctx context.Context, event Event) {
		atomic.AddInt32(&received, 1)
		wg.Done()
	})

	eb.Subscribe(EventTypeOrderFilled, QoS1, subscriber)

	// Publish events
	eventCount := 50
	wg.Add(eventCount)

	for i := 0; i < eventCount; i++ {
		testEvent := &BaseEvent{
			EventType: EventTypeOrderFilled,
			EventQoS:  QoS1,
		}
		eb.Publish(testEvent)
	}

	// Wait for all events
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// All events received
		assert.Equal(t, int32(eventCount), atomic.LoadInt32(&received))
	case <-time.After(5 * time.Second):
		t.Fatal("Not all QoS1 events received")
	}
}

// TestEventBus_MultipleSubscribers tests multiple subscribers to same event type.
func TestEventBus_MultipleSubscribers(t *testing.T) {
	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	eb := NewEventBus(ctx, logger)
	defer eb.Shutdown()

	// Create multiple subscribers
	var received1, received2, received3 int32

	sub1 := SubscriberFunc(func(ctx context.Context, event Event) {
		atomic.AddInt32(&received1, 1)
	})
	sub2 := SubscriberFunc(func(ctx context.Context, event Event) {
		atomic.AddInt32(&received2, 1)
	})
	sub3 := SubscriberFunc(func(ctx context.Context, event Event) {
		atomic.AddInt32(&received3, 1)
	})

	eb.Subscribe(EventTypeSignal, QoS1, sub1)
	eb.Subscribe(EventTypeSignal, QoS1, sub2)
	eb.Subscribe(EventTypeSignal, QoS1, sub3)

	// Publish event
	testEvent := &BaseEvent{
		EventType: EventTypeSignal,
		EventQoS:  QoS1,
	}
	eb.Publish(testEvent)

	// Wait for delivery
	time.Sleep(100 * time.Millisecond)

	// All subscribers should receive
	assert.Equal(t, int32(1), atomic.LoadInt32(&received1))
	assert.Equal(t, int32(1), atomic.LoadInt32(&received2))
	assert.Equal(t, int32(1), atomic.LoadInt32(&received3))
}

// TestEventBus_Shutdown tests graceful shutdown.
func TestEventBus_Shutdown(t *testing.T) {
	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	eb := NewEventBus(ctx, logger)

	// Subscribe
	var received int32
	subscriber := SubscriberFunc(func(ctx context.Context, event Event) {
		atomic.AddInt32(&received, 1)
	})

	eb.Subscribe(EventTypeAlert, QoS1, subscriber)

	// Publish event
	testEvent := &BaseEvent{
		EventType: EventTypeAlert,
		EventQoS:  QoS1,
	}
	eb.Publish(testEvent)

	// Wait for event to be processed
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, int32(1), atomic.LoadInt32(&received))

	// Shutdown
	eb.Shutdown()

	// Publish after shutdown should not panic and should not be delivered
	eb.Publish(testEvent)
	time.Sleep(100 * time.Millisecond)

	// Should not receive new events after shutdown
	assert.Equal(t, int32(1), atomic.LoadInt32(&received), "Should not receive events after shutdown")
}

// TestEventBus_ConcurrentPublish tests concurrent publishing from multiple goroutines.
func TestEventBus_ConcurrentPublish(t *testing.T) {
	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	eb := NewEventBus(ctx, logger)
	defer eb.Shutdown()

	// Create subscriber
	var received int32
	subscriber := SubscriberFunc(func(ctx context.Context, event Event) {
		atomic.AddInt32(&received, 1)
	})

	eb.Subscribe(EventTypeTicker, QoS1, subscriber)

	// Publish from multiple goroutines
	var wg sync.WaitGroup
	goroutines := 10
	eventsPerGoroutine := 100

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < eventsPerGoroutine; j++ {
				testEvent := &BaseEvent{
					EventType: EventTypeTicker,
					EventQoS:  QoS1,
				}
				eb.Publish(testEvent)
			}
		}()
	}

	wg.Wait()

	// Wait for all events to be processed
	time.Sleep(500 * time.Millisecond)

	// Should receive all events
	expectedCount := int32(goroutines * eventsPerGoroutine)
	actualCount := atomic.LoadInt32(&received)
	assert.Equal(t, expectedCount, actualCount, "Should receive all events from concurrent publishers")
}

// TestEventBus_ContextCancellation tests that context cancellation stops delivery.
func TestEventBus_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	eb := NewEventBus(ctx, logger)

	// Create subscriber
	var received int32
	subscriber := SubscriberFunc(func(ctx context.Context, event Event) {
		atomic.AddInt32(&received, 1)
	})

	eb.Subscribe(EventTypeAlert, QoS1, subscriber)

	// Publish event
	testEvent := &BaseEvent{
		EventType: EventTypeAlert,
		EventQoS:  QoS1,
	}
	eb.Publish(testEvent)

	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, int32(1), atomic.LoadInt32(&received))

	// Cancel context
	cancel()
	time.Sleep(100 * time.Millisecond)

	// Publish after cancellation
	eb.Publish(testEvent)
	time.Sleep(100 * time.Millisecond)

	// Should not receive new events after cancellation
	assert.Equal(t, int32(1), atomic.LoadInt32(&received), "Should not receive events after context cancellation")
}
