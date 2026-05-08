package qos

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kairos-development/kairos-agent/internal/domain/events"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockHandler struct {
	calls   atomic.Int64
	lastErr error
}

func (m *mockHandler) Handle(ctx context.Context, event events.Event) error {
	m.calls.Add(1)
	return m.lastErr
}

func TestNewBus(t *testing.T) {
	logger := logrus.New()
	cfg := DefaultConfig()
	bus := NewBus(cfg, logger)
	defer bus.Close()

	assert.NotNil(t, bus)
	assert.NotNil(t, bus.publisher)
}

func TestSubscribeAndPublish_QoS1(t *testing.T) {
	logger := logrus.New()
	cfg := DefaultConfig()
	bus := NewBus(cfg, logger)
	defer bus.Close()

	handler := &mockHandler{}
	event := &events.OrderCreatedEvent{
		BaseEvent: events.BaseEvent{
			EventType:      events.EventTypeOrderCreated,
			OccurredAt:     time.Now().UTC(),
			CorrelationID_: "test-123",
		},
		OrderID: "order-1",
	}

	bus.Subscribe(events.EventTypeOrderCreated, handler, QoS1_AtLeastOnce)
	err := bus.Publish(context.Background(), event)
	require.NoError(t, err)
	assert.Equal(t, int64(1), handler.calls.Load())
}

func TestPublishAsync_QoS0(t *testing.T) {
	logger := logrus.New()
	cfg := DefaultConfig()
	cfg.BufferSize = 10
	bus := NewBus(cfg, logger)
	defer bus.Close()

	handler := &mockHandler{}
	event := &events.OrderCreatedEvent{
		BaseEvent: events.BaseEvent{
			EventType:      events.EventTypeOrderCreated,
			OccurredAt:     time.Now().UTC(),
			CorrelationID_: "test-456",
		},
		OrderID: "order-2",
	}

	bus.Subscribe(events.EventTypeOrderCreated, handler, QoS0_AtMostOnce)
	bus.PublishAsync(context.Background(), event)

	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, int64(1), handler.calls.Load())
}

func TestQoS1_Retry(t *testing.T) {
	logger := logrus.New()
	cfg := DefaultConfig()
	cfg.MaxRetries = 2
	cfg.RetryInterval = 10 * time.Millisecond
	bus := NewBus(cfg, logger)
	defer bus.Close()

	handler := &mockHandler{lastErr: assert.AnError}
	event := &events.OrderCreatedEvent{
		BaseEvent: events.BaseEvent{
			EventType:      events.EventTypeOrderCreated,
			OccurredAt:     time.Now().UTC(),
			CorrelationID_: "test-789",
		},
		OrderID: "order-3",
	}

	bus.Subscribe(events.EventTypeOrderCreated, handler, QoS1_AtLeastOnce)
	err := bus.Publish(context.Background(), event)
	assert.Error(t, err)
	assert.Equal(t, int64(3), handler.calls.Load()) // 1 initial + 2 retries
}

func TestQoS0_DropOnFullBuffer(t *testing.T) {
	logger := logrus.New()
	cfg := DefaultConfig()
	cfg.BufferSize = 2
	bus := NewBus(cfg, logger)

	// Close the bus first so asyncWorker stops draining the channel.
	// This ensures PublishAsync hits the full-buffer path reliably.
	bus.Close()

	event := &events.OrderCreatedEvent{
		BaseEvent: events.BaseEvent{
			EventType:      events.EventTypeOrderCreated,
			OccurredAt:     time.Now().UTC(),
			CorrelationID_: "test-drop",
		},
		OrderID: "order-drop",
	}

	// With the bus closed, the asyncCh buffer is no longer being drained,
	// so filling it beyond capacity should record drops.
	bus.PublishAsync(context.Background(), event)
	bus.PublishAsync(context.Background(), event)
	bus.PublishAsync(context.Background(), event) // Should drop

	_, dropped, _ := bus.Stats()
	assert.GreaterOrEqual(t, dropped, int64(1))
}

func TestClose(t *testing.T) {
	logger := logrus.New()
	cfg := DefaultConfig()
	bus := NewBus(cfg, logger)

	err := bus.Close()
	assert.NoError(t, err)

	_, dropped, _ := bus.Stats()
	assert.Equal(t, int64(0), dropped)
}
