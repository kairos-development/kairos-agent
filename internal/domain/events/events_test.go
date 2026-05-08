package events

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestBaseEvent_Type(t *testing.T) {
	event := &BaseEvent{
		EventType: EventTypeOrderCreated,
	}
	assert.Equal(t, EventTypeOrderCreated, event.Type())
}

func TestBaseEvent_OccurredAtUTC(t *testing.T) {
	now := time.Now().UTC()
	event := &BaseEvent{
		OccurredAt: now,
	}
	assert.Equal(t, now, event.OccurredAtUTC())
}

func TestBaseEvent_CorrelationID(t *testing.T) {
	event := &BaseEvent{
		CorrelationID_: "corr123",
	}
	assert.Equal(t, "corr123", event.CorrelationID())
}

func TestOrderCreatedEvent(t *testing.T) {
	now := time.Now().UTC()
	event := &OrderCreatedEvent{
		BaseEvent: BaseEvent{
			EventType:      EventTypeOrderCreated,
			OccurredAt:     now,
			CorrelationID_: "corr1",
		},
		OrderID:       "order1",
		ClientOrderID: "client1",
		StrategyID:    "strat1",
		Symbol:        "BTCUSDT",
		Side:          "buy",
		OrderType:     "limit",
		Quantity:      decimal.NewFromFloat(0.1),
		Price:         decimal.NewFromInt(50000),
	}

	assert.Equal(t, EventTypeOrderCreated, event.Type())
	assert.Equal(t, now, event.OccurredAtUTC())
	assert.Equal(t, "corr1", event.CorrelationID())
	assert.Equal(t, "order1", event.OrderID)
	assert.Equal(t, "BTCUSDT", event.Symbol)
	assert.True(t, event.Quantity.Equal(decimal.NewFromFloat(0.1)))
}

func TestOrderSubmittedEvent(t *testing.T) {
	now := time.Now().UTC()
	event := &OrderSubmittedEvent{
		BaseEvent: BaseEvent{
			EventType:  EventTypeOrderSubmitted,
			OccurredAt: now,
		},
		OrderID:         "order1",
		ClientOrderID:   "client1",
		ExchangeOrderID: "exchange1",
		SubmittedAtUTC:  now,
	}

	assert.Equal(t, EventTypeOrderSubmitted, event.Type())
	assert.Equal(t, "order1", event.OrderID)
	assert.Equal(t, "exchange1", event.ExchangeOrderID)
}

func TestOrderPartialEvent(t *testing.T) {
	event := &OrderPartialEvent{
		BaseEvent: BaseEvent{
			EventType: EventTypeOrderPartial,
		},
		OrderID:      "order1",
		FilledQty:    decimal.NewFromFloat(0.05),
		RemainingQty: decimal.NewFromFloat(0.05),
		AvgFillPrice: decimal.NewFromInt(50000),
	}

	assert.Equal(t, EventTypeOrderPartial, event.Type())
	assert.True(t, event.FilledQty.Equal(decimal.NewFromFloat(0.05)))
	assert.True(t, event.RemainingQty.Equal(decimal.NewFromFloat(0.05)))
}

func TestOrderFilledEvent(t *testing.T) {
	now := time.Now().UTC()
	event := &OrderFilledEvent{
		BaseEvent: BaseEvent{
			EventType: EventTypeOrderFilled,
		},
		OrderID:      "order1",
		FilledQty:    decimal.NewFromFloat(0.1),
		AvgFillPrice: decimal.NewFromInt(50000),
		FilledAtUTC:  now,
	}

	assert.Equal(t, EventTypeOrderFilled, event.Type())
	assert.True(t, event.FilledQty.Equal(decimal.NewFromFloat(0.1)))
	assert.Equal(t, now, event.FilledAtUTC)
}

func TestOrderCanceledEvent(t *testing.T) {
	now := time.Now().UTC()
	event := &OrderCanceledEvent{
		BaseEvent: BaseEvent{
			EventType: EventTypeOrderCanceled,
		},
		OrderID:       "order1",
		Reason:        "User requested",
		CanceledAtUTC: now,
	}

	assert.Equal(t, EventTypeOrderCanceled, event.Type())
	assert.Equal(t, "order1", event.OrderID)
	assert.Equal(t, "User requested", event.Reason)
}

func TestOrderRejectedEvent(t *testing.T) {
	now := time.Now().UTC()
	event := &OrderRejectedEvent{
		BaseEvent: BaseEvent{
			EventType: EventTypeOrderRejected,
		},
		OrderID:       "order1",
		Reason:        "Insufficient balance",
		RejectedAtUTC: now,
	}

	assert.Equal(t, EventTypeOrderRejected, event.Type())
	assert.Equal(t, "Insufficient balance", event.Reason)
}

func TestPositionOpenedEvent(t *testing.T) {
	now := time.Now().UTC()
	event := &PositionOpenedEvent{
		BaseEvent: BaseEvent{
			EventType: EventTypePositionOpened,
		},
		PositionID:  "pos1",
		StrategyID:  "strat1",
		Symbol:      "BTCUSDT",
		Side:        "long",
		Quantity:    decimal.NewFromFloat(0.1),
		EntryPrice:  decimal.NewFromInt(50000),
		OpenedAtUTC: now,
	}

	assert.Equal(t, EventTypePositionOpened, event.Type())
	assert.Equal(t, "pos1", event.PositionID)
	assert.Equal(t, "BTCUSDT", event.Symbol)
	assert.True(t, event.Quantity.Equal(decimal.NewFromFloat(0.1)))
}

func TestPositionUpdatedEvent(t *testing.T) {
	now := time.Now().UTC()
	event := &PositionUpdatedEvent{
		BaseEvent: BaseEvent{
			EventType: EventTypePositionUpdated,
		},
		PositionID:    "pos1",
		CurrentPrice:  decimal.NewFromInt(51000),
		UnrealizedPnL: decimal.NewFromInt(100),
		UpdatedAtUTC:  now,
	}

	assert.Equal(t, EventTypePositionUpdated, event.Type())
	assert.True(t, event.CurrentPrice.Equal(decimal.NewFromInt(51000)))
	assert.True(t, event.UnrealizedPnL.Equal(decimal.NewFromInt(100)))
}

func TestPositionClosedEvent(t *testing.T) {
	now := time.Now().UTC()
	event := &PositionClosedEvent{
		BaseEvent: BaseEvent{
			EventType: EventTypePositionClosed,
		},
		PositionID:  "pos1",
		Symbol:      "BTCUSDT",
		RealizedPnL: decimal.NewFromInt(100),
		ClosedAtUTC: now,
	}

	assert.Equal(t, EventTypePositionClosed, event.Type())
	assert.Equal(t, "BTCUSDT", event.Symbol)
	assert.True(t, event.RealizedPnL.Equal(decimal.NewFromInt(100)))
}

func TestBalanceUpdatedEvent(t *testing.T) {
	now := time.Now().UTC()
	event := &BalanceUpdatedEvent{
		BaseEvent: BaseEvent{
			EventType: EventTypeBalanceUpdated,
		},
		Asset:        "USDT",
		Total:        decimal.NewFromInt(10000),
		Available:    decimal.NewFromInt(9500),
		Locked:       decimal.NewFromInt(500),
		UpdatedAtUTC: now,
	}

	assert.Equal(t, EventTypeBalanceUpdated, event.Type())
	assert.Equal(t, "USDT", event.Asset)
	assert.True(t, event.Total.Equal(decimal.NewFromInt(10000)))
}

func TestRiskViolationEvent(t *testing.T) {
	event := &RiskViolationEvent{
		BaseEvent: BaseEvent{
			EventType: EventTypeRiskViolation,
		},
		ViolationType: "max_position_size",
		Action:        "order_rejected",
		CurrentValue:  decimal.NewFromInt(1500),
		LimitValue:    decimal.NewFromInt(1000),
	}

	assert.Equal(t, EventTypeRiskViolation, event.Type())
	assert.Equal(t, "max_position_size", event.ViolationType)
	assert.Equal(t, "order_rejected", event.Action)
}

func TestRiskWarningEvent(t *testing.T) {
	event := &RiskWarningEvent{
		BaseEvent: BaseEvent{
			EventType: EventTypeRiskWarning,
		},
		WarningType:  "daily_loss_threshold",
		ThresholdPct: decimal.NewFromInt(80),
		CurrentValue: decimal.NewFromInt(400),
		LimitValue:   decimal.NewFromInt(500),
	}

	assert.Equal(t, EventTypeRiskWarning, event.Type())
	assert.Equal(t, "daily_loss_threshold", event.WarningType)
	assert.True(t, event.ThresholdPct.Equal(decimal.NewFromInt(80)))
}

func TestStrategyStartedEvent(t *testing.T) {
	now := time.Now().UTC()
	event := &StrategyStartedEvent{
		BaseEvent: BaseEvent{
			EventType: EventTypeStrategyStarted,
		},
		StrategyID:   "strat1",
		StrategyName: "SMA-Cross",
		StrategyType: "trend_following",
		StartedAtUTC: now,
	}

	assert.Equal(t, EventTypeStrategyStarted, event.Type())
	assert.Equal(t, "strat1", event.StrategyID)
	assert.Equal(t, "SMA-Cross", event.StrategyName)
}

func TestStrategyPausedEvent(t *testing.T) {
	now := time.Now().UTC()
	event := &StrategyPausedEvent{
		BaseEvent: BaseEvent{
			EventType: EventTypeStrategyPaused,
		},
		StrategyID:  "strat1",
		Reason:      "Risk limit reached",
		PausedAtUTC: now,
	}

	assert.Equal(t, EventTypeStrategyPaused, event.Type())
	assert.Equal(t, "Risk limit reached", event.Reason)
}

func TestStrategyStoppedEvent(t *testing.T) {
	now := time.Now().UTC()
	event := &StrategyStoppedEvent{
		BaseEvent: BaseEvent{
			EventType: EventTypeStrategyStopped,
		},
		StrategyID:   "strat1",
		Reason:       "User requested",
		FinalPnL:     decimal.NewFromInt(500),
		StoppedAtUTC: now,
	}

	assert.Equal(t, EventTypeStrategyStopped, event.Type())
	assert.Equal(t, "User requested", event.Reason)
	assert.True(t, event.FinalPnL.Equal(decimal.NewFromInt(500)))
}

func TestEventType_Constants(t *testing.T) {
	assert.Equal(t, EventType("order.created"), EventTypeOrderCreated)
	assert.Equal(t, EventType("order.submitted"), EventTypeOrderSubmitted)
	assert.Equal(t, EventType("order.partial"), EventTypeOrderPartial)
	assert.Equal(t, EventType("order.filled"), EventTypeOrderFilled)
	assert.Equal(t, EventType("order.canceled"), EventTypeOrderCanceled)
	assert.Equal(t, EventType("order.rejected"), EventTypeOrderRejected)
	assert.Equal(t, EventType("position.opened"), EventTypePositionOpened)
	assert.Equal(t, EventType("position.updated"), EventTypePositionUpdated)
	assert.Equal(t, EventType("position.closed"), EventTypePositionClosed)
	assert.Equal(t, EventType("risk.violation"), EventTypeRiskViolation)
	assert.Equal(t, EventType("risk.warning"), EventTypeRiskWarning)
	assert.Equal(t, EventType("balance.updated"), EventTypeBalanceUpdated)
	assert.Equal(t, EventType("strategy.started"), EventTypeStrategyStarted)
	assert.Equal(t, EventType("strategy.paused"), EventTypeStrategyPaused)
	assert.Equal(t, EventType("strategy.stopped"), EventTypeStrategyStopped)
}

func TestNewPublisher(t *testing.T) {
	publisher := NewPublisher()
	assert.NotNil(t, publisher)
	assert.NotNil(t, publisher.handlers)
}

func TestPublisher_Subscribe(t *testing.T) {
	publisher := NewPublisher()
	handler := &mockHandler{}

	publisher.Subscribe(EventTypeOrderCreated, handler)

	// Verify handler was registered
	assert.Len(t, publisher.handlers[EventTypeOrderCreated], 1)
}

func TestPublisher_SubscribeFunc(t *testing.T) {
	publisher := NewPublisher()
	called := false

	handlerFunc := func(ctx context.Context, event Event) error {
		called = true
		return nil
	}

	publisher.SubscribeFunc(EventTypeOrderCreated, handlerFunc)

	// Verify handler was registered
	assert.Len(t, publisher.handlers[EventTypeOrderCreated], 1)

	// Publish event and verify handler was called
	event := &OrderCreatedEvent{
		BaseEvent: BaseEvent{EventType: EventTypeOrderCreated},
	}
	err := publisher.Publish(context.Background(), event)
	assert.NoError(t, err)
	assert.True(t, called)
}

func TestPublisher_Publish_Success(t *testing.T) {
	publisher := NewPublisher()
	handler := &mockHandler{}

	publisher.Subscribe(EventTypeOrderCreated, handler)

	event := &OrderCreatedEvent{
		BaseEvent: BaseEvent{EventType: EventTypeOrderCreated},
		OrderID:   "order1",
	}

	err := publisher.Publish(context.Background(), event)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), handler.callCount.Load())
	assert.Equal(t, event, handler.getLastEvent())
}

func TestPublisher_Publish_MultipleHandlers(t *testing.T) {
	publisher := NewPublisher()
	handler1 := &mockHandler{}
	handler2 := &mockHandler{}

	publisher.Subscribe(EventTypeOrderCreated, handler1)
	publisher.Subscribe(EventTypeOrderCreated, handler2)

	event := &OrderCreatedEvent{
		BaseEvent: BaseEvent{EventType: EventTypeOrderCreated},
	}

	err := publisher.Publish(context.Background(), event)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), handler1.callCount.Load())
	assert.Equal(t, int64(1), handler2.callCount.Load())
}

func TestPublisher_Publish_HandlerError(t *testing.T) {
	publisher := NewPublisher()
	handler := &mockHandler{returnErr: assert.AnError}

	publisher.Subscribe(EventTypeOrderCreated, handler)

	event := &OrderCreatedEvent{
		BaseEvent: BaseEvent{EventType: EventTypeOrderCreated},
	}

	err := publisher.Publish(context.Background(), event)
	assert.Error(t, err)
	assert.Equal(t, assert.AnError, err)
}

func TestPublisher_Publish_FirstErrorReturned(t *testing.T) {
	publisher := NewPublisher()
	handler1 := &mockHandler{returnErr: assert.AnError}
	handler2 := &mockHandler{}

	publisher.Subscribe(EventTypeOrderCreated, handler1)
	publisher.Subscribe(EventTypeOrderCreated, handler2)

	event := &OrderCreatedEvent{
		BaseEvent: BaseEvent{EventType: EventTypeOrderCreated},
	}

	err := publisher.Publish(context.Background(), event)
	assert.Error(t, err)
	assert.Equal(t, assert.AnError, err)
	// Both handlers should still be called
	assert.Equal(t, int64(1), handler1.callCount.Load())
	assert.Equal(t, int64(1), handler2.callCount.Load())
}

func TestPublisher_Publish_NoHandlers(t *testing.T) {
	publisher := NewPublisher()

	event := &OrderCreatedEvent{
		BaseEvent: BaseEvent{EventType: EventTypeOrderCreated},
	}

	err := publisher.Publish(context.Background(), event)
	assert.NoError(t, err)
}

func TestPublisher_PublishAsync(t *testing.T) {
	publisher := NewPublisher()
	handler := &mockHandler{}

	publisher.Subscribe(EventTypeOrderCreated, handler)

	event := &OrderCreatedEvent{
		BaseEvent: BaseEvent{EventType: EventTypeOrderCreated},
	}

	publisher.PublishAsync(context.Background(), event)

	// Wait for async goroutine to complete
	time.Sleep(10 * time.Millisecond)

	assert.Equal(t, int64(1), handler.callCount.Load())
}

func TestHandlerFunc_Handle(t *testing.T) {
	called := false
	var receivedEvent Event

	handlerFunc := HandlerFunc(func(ctx context.Context, event Event) error {
		called = true
		receivedEvent = event
		return nil
	})

	event := &OrderCreatedEvent{
		BaseEvent: BaseEvent{EventType: EventTypeOrderCreated},
	}

	err := handlerFunc.Handle(context.Background(), event)
	assert.NoError(t, err)
	assert.True(t, called)
	assert.Equal(t, event, receivedEvent)
}

func TestPublisher_ConcurrentPublish(t *testing.T) {
	publisher := NewPublisher()
	handler := &mockHandler{}

	publisher.Subscribe(EventTypeOrderCreated, handler)

	event := &OrderCreatedEvent{
		BaseEvent: BaseEvent{EventType: EventTypeOrderCreated},
	}

	// Publish concurrently
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			_ = publisher.Publish(context.Background(), event)
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	assert.Equal(t, int64(10), handler.callCount.Load())
}

// mockHandler is a test helper for Publisher tests
type mockHandler struct {
	mu        sync.Mutex
	callCount atomic.Int64
	lastEvent Event
	returnErr error
}

func (m *mockHandler) Handle(_ context.Context, event Event) error {
	m.callCount.Add(1)
	m.mu.Lock()
	m.lastEvent = event
	m.mu.Unlock()
	return m.returnErr
}

func (m *mockHandler) getLastEvent() Event {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lastEvent
}
