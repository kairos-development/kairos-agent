package engine

import (
	"context"
	"testing"
	"time"

	domainconnector "github.com/kairos-development/kairos-agent/internal/domain/connector"
	"github.com/kairos-development/kairos-agent/internal/domain/entity"
	"github.com/kairos-development/kairos-agent/internal/domain/events"
	"github.com/kairos-development/kairos-agent/internal/service/agent"
	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEngine_DefaultConfig tests default configuration.
func TestEngine_DefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.NotNil(t, cfg)
	assert.Equal(t, 500*time.Millisecond, cfg.MaxNTPDrift)
	assert.Equal(t, 10, cfg.MaxOrdersPerSecond)
	assert.True(t, cfg.MaxPositionSize.Equal(decimal.RequireFromString("10000")))
	assert.True(t, cfg.MaxDailyLoss.Equal(decimal.RequireFromString("1000")))
}

// TestEngine_New tests engine creation.
func TestEngine_New(t *testing.T) {
	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	cfg := DefaultConfig()
	engine := New(ctx, nil, cfg, logger)

	assert.NotNil(t, engine)
	assert.NotNil(t, engine.stateMachine)
	assert.NotNil(t, engine.eventBus)
	assert.Equal(t, StateIdle, engine.State())
}

// TestEngine_NewWithNilConfig tests engine creation with nil config uses defaults.
func TestEngine_NewWithNilConfig(t *testing.T) {
	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	engine := New(ctx, nil, nil, logger)

	assert.NotNil(t, engine)
	assert.NotNil(t, engine.config)
	assert.Equal(t, 500*time.Millisecond, engine.config.MaxNTPDrift)
}

// TestEngine_State tests state getter.
func TestEngine_State(t *testing.T) {
	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	engine := New(ctx, nil, nil, logger)

	assert.Equal(t, StateIdle, engine.State())
}

// TestEngine_TransitionTo tests state transition.
func TestEngine_TransitionTo(t *testing.T) {
	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	engine := New(ctx, nil, nil, logger)

	// Transition to LiveTrading
	err := engine.TransitionTo(StateLiveTrading, "test transition")
	assert.NoError(t, err)
	assert.Equal(t, StateLiveTrading, engine.State())

	// Invalid transition (same state)
	err = engine.TransitionTo(StateLiveTrading, "invalid")
	assert.Error(t, err)
}

// TestEngine_EventBus tests event bus getter.
func TestEngine_EventBus(t *testing.T) {
	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	engine := New(ctx, nil, nil, logger)

	eventBus := engine.EventBus()
	assert.NotNil(t, eventBus)
	assert.Equal(t, engine.eventBus, eventBus)
}

// TestEngine_GetSetConnector tests connector getter/setter.
func TestEngine_GetSetConnector(t *testing.T) {
	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	engine := New(ctx, nil, nil, logger)

	// Initially nil
	assert.Nil(t, engine.GetConnector())

	// Set connector (using nil as mock)
	engine.SetConnector(nil)
	assert.Nil(t, engine.GetConnector())
}

func TestEngine_StreamEventGapHaltsLiveTradingAndReconciles(t *testing.T) {
	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	orderSvc := &streamEventOrderService{
		reconciled: make(chan struct{}, 1),
	}
	conn := &streamEventConnector{
		events: make(chan *domainconnector.StreamEvent, 1),
	}

	engine := New(ctx, orderSvc, nil, logger)
	engine.SetConnector(conn)

	err := engine.Start()
	require.NoError(t, err)
	defer func() {
		_ = engine.Stop()
	}()

	err = engine.TransitionTo(StateLiveTrading, "start live trading")
	require.NoError(t, err)

	conn.events <- &domainconnector.StreamEvent{
		Type:          domainconnector.StreamEventGap,
		Source:        "bybit_private_ws",
		Reason:        "test stream gap",
		OccurredAtUTC: time.Now().UTC(),
	}

	select {
	case <-orderSvc.reconciled:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for reconciliation after stream event")
	}

	require.Eventually(t, func() bool {
		return engine.State() == StateHalted
	}, time.Second, 10*time.Millisecond)
}

func TestEngine_StreamEventReconnectUsesFullReconciler(t *testing.T) {
	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	orderSvc := &streamEventOrderService{
		reconciled: make(chan struct{}, 1),
	}
	reconciler := &streamEventReconciler{
		reconciled: make(chan struct{}, 1),
	}
	conn := &streamEventConnector{
		events: make(chan *domainconnector.StreamEvent, 1),
	}

	engine := New(ctx, orderSvc, nil, logger)
	engine.SetConnector(conn)
	engine.SetReconciliationService(reconciler)

	err := engine.Start()
	require.NoError(t, err)
	defer func() {
		_ = engine.Stop()
	}()

	conn.events <- &domainconnector.StreamEvent{
		Type:          domainconnector.StreamEventReconnected,
		Source:        "bybit_private_ws",
		Reason:        "test reconnect",
		OccurredAtUTC: time.Now().UTC(),
	}

	select {
	case <-reconciler.reconciled:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for full reconciliation after reconnect")
	}

	select {
	case <-orderSvc.reconciled:
		t.Fatal("expected full reconciler to take precedence over order fallback")
	default:
	}
}

// TestEngine_GetSetStorage tests storage getter/setter.
func TestEngine_GetSetStorage(t *testing.T) {
	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	engine := New(ctx, nil, nil, logger)

	// Initially nil
	assert.Nil(t, engine.GetStorage())

	// Set storage (using nil as mock)
	engine.SetStorage(nil)
	assert.Nil(t, engine.GetStorage())
}

// TestEngine_GetSetStrategyExecutor tests strategy executor getter/setter.
func TestEngine_GetSetStrategyExecutor(t *testing.T) {
	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	engine := New(ctx, nil, nil, logger)

	// Initially nil
	assert.Nil(t, engine.GetStrategyExecutor())

	// Set executor (using nil as mock)
	engine.SetStrategyExecutor(nil)
	assert.Nil(t, engine.GetStrategyExecutor())
}

// TestEngine_GetEventBus tests event bus getter.
func TestEngine_GetEventBus(t *testing.T) {
	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	engine := New(ctx, nil, nil, logger)

	eventBus := engine.GetEventBus()
	assert.NotNil(t, eventBus)
	assert.Equal(t, engine.eventBus, eventBus)
}

// TestEngine_StartStop tests engine start and stop.
func TestEngine_StartStop(t *testing.T) {
	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	engine := New(ctx, nil, nil, logger)

	// Start engine
	err := engine.Start()
	require.NoError(t, err)

	// Give workers time to start
	time.Sleep(100 * time.Millisecond)

	// Stop engine
	err = engine.Stop()
	assert.NoError(t, err)

	// State should be Halted
	assert.Equal(t, StateHalted, engine.State())
}

// TestEngine_PublishDomainEvent tests domain event publishing.
func TestEngine_PublishDomainEvent(t *testing.T) {
	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	engine := New(ctx, nil, nil, logger)

	// Subscribe to events
	ch := make(chan Event, 10)
	subscriber := SubscriberFunc(func(ctx context.Context, event Event) {
		ch <- event
	})

	engine.eventBus.Subscribe(EventTypeOrderCreated, QoS1, subscriber)

	// Create mock domain event
	// Note: We can't import domain events here, so we'll test the method exists
	// In integration tests, we would test actual domain event conversion

	// Just verify the method doesn't panic with nil
	engine.PublishDomainEvent(nil)
}

// TestEngine_StateTransitionPublishesEvent tests that state transitions publish events.
func TestEngine_StateTransitionPublishesEvent(t *testing.T) {
	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	engine := New(ctx, nil, nil, logger)

	// Start engine to activate workers
	err := engine.Start()
	require.NoError(t, err)
	defer engine.Stop()

	// Subscribe to state transition events
	ch := make(chan Event, 10)
	subscriber := SubscriberFunc(func(ctx context.Context, event Event) {
		ch <- event
	})

	engine.eventBus.Subscribe(EventTypeStateTransition, QoS1, subscriber)

	// Transition state
	err = engine.TransitionTo(StateLiveTrading, "test")
	require.NoError(t, err)

	// Wait for event
	select {
	case event := <-ch:
		assert.Equal(t, EventTypeStateTransition, event.Type())
		stateEvent, ok := event.(*StateTransitionEvent)
		assert.True(t, ok)
		assert.Equal(t, StateIdle, stateEvent.From)
		assert.Equal(t, StateLiveTrading, stateEvent.To)
		assert.Equal(t, "test", stateEvent.Reason)
	case <-time.After(1 * time.Second):
		t.Fatal("State transition event not received")
	}
}

// TestEngine_MultipleStartStop tests multiple start/stop cycles.
func TestEngine_MultipleStartStop(t *testing.T) {
	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	engine := New(ctx, nil, nil, logger)

	// First cycle
	err := engine.Start()
	require.NoError(t, err)
	time.Sleep(50 * time.Millisecond)
	err = engine.Stop()
	require.NoError(t, err)

	// Reset to Idle for second cycle
	err = engine.TransitionTo(StateIdle, "reset")
	require.NoError(t, err)

	// Second cycle
	err = engine.Start()
	require.NoError(t, err)
	time.Sleep(50 * time.Millisecond)
	err = engine.Stop()
	assert.NoError(t, err)
}

// TestEngine_ContextCancellation tests that engine respects context cancellation.
func TestEngine_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	engine := New(ctx, nil, nil, logger)

	// Start engine
	err := engine.Start()
	require.NoError(t, err)

	// Cancel context
	cancel()

	// Give time for workers to stop
	time.Sleep(100 * time.Millisecond)

	// Stop should still work
	err = engine.Stop()
	assert.NoError(t, err)
}

// TestEngine_ConfigurationValues tests that configuration is properly stored.
func TestEngine_ConfigurationValues(t *testing.T) {
	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	cfg := &Config{
		MaxNTPDrift:        1 * time.Second,
		MaxOrdersPerSecond: 20,
		MaxPositionSize:    decimal.RequireFromString("50000"),
		MaxDailyLoss:       decimal.RequireFromString("5000"),
	}

	engine := New(ctx, nil, cfg, logger)

	assert.Equal(t, 1*time.Second, engine.config.MaxNTPDrift)
	assert.Equal(t, 20, engine.config.MaxOrdersPerSecond)
	assert.True(t, engine.config.MaxPositionSize.Equal(decimal.RequireFromString("50000")))
	assert.True(t, engine.config.MaxDailyLoss.Equal(decimal.RequireFromString("5000")))
}

// TestEngine_NTPSyncWorker tests NTP sync worker execution.
func TestEngine_NTPSyncWorker(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	engine := New(ctx, nil, nil, logger)

	// Start engine to activate workers
	err := engine.Start()
	require.NoError(t, err)
	defer engine.Stop()

	// Wait a bit for worker to run
	time.Sleep(200 * time.Millisecond)

	// NTP worker should have attempted sync
	// We can't verify actual NTP sync in unit tests, but we can verify the engine is running
	assert.NotEqual(t, StateHalted, engine.State())
}

// TestEngine_ReconciliationWorker tests reconciliation worker execution.
func TestEngine_ReconciliationWorker(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	engine := New(ctx, nil, nil, logger)

	// Start engine to activate workers
	err := engine.Start()
	require.NoError(t, err)
	defer engine.Stop()

	// Wait a bit for worker to run
	time.Sleep(200 * time.Millisecond)

	// Reconciliation worker should have run
	// In unit tests, it will just log that no connector is available
}

// TestEngine_PublishDomainEventWithNilEvent tests publishing nil domain event.
func TestEngine_PublishDomainEventWithNilEvent(t *testing.T) {
	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	engine := New(ctx, nil, nil, logger)

	// Should not panic with nil event
	engine.PublishDomainEvent(nil)
}

// TestEngine_WorkersStopOnContextCancel tests that workers stop when context is cancelled.
func TestEngine_WorkersStopOnContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	engine := New(ctx, nil, nil, logger)

	// Start engine
	err := engine.Start()
	require.NoError(t, err)

	// Cancel context
	cancel()

	// Give workers time to stop
	time.Sleep(100 * time.Millisecond)

	// Stop should still work
	err = engine.Stop()
	assert.NoError(t, err)
}

// TestEngine_PublishDomainEventOrderCreated tests publishing OrderCreatedEvent.
func TestEngine_PublishDomainEventOrderCreated(t *testing.T) {
	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	engine := New(ctx, nil, nil, logger)
	err := engine.Start()
	require.NoError(t, err)
	defer engine.Stop()

	// Subscribe to events
	ch := make(chan Event, 10)
	subscriber := SubscriberFunc(func(ctx context.Context, event Event) {
		ch <- event
	})
	engine.eventBus.Subscribe(EventTypeOrderCreated, QoS1, subscriber)

	// Create domain event
	domainEvent := &events.OrderCreatedEvent{
		BaseEvent: events.BaseEvent{
			EventType:      events.EventTypeOrderCreated,
			OccurredAt:     time.Now().UTC(),
			CorrelationID_: "test-123",
		},
		OrderID:       "order-1",
		ClientOrderID: "client-1",
		Symbol:        "BTCUSDT",
		Side:          "BUY",
		OrderType:     "LIMIT",
		Quantity:      decimal.RequireFromString("1"),
		Price:         decimal.RequireFromString("50000"),
	}

	// Publish domain event
	engine.PublishDomainEvent(domainEvent)

	// Wait for event
	select {
	case event := <-ch:
		assert.Equal(t, EventTypeOrderCreated, event.Type())
	case <-time.After(1 * time.Second):
		t.Fatal("Event not received")
	}
}

// TestEngine_PublishDomainEventOrderSubmitted tests publishing OrderSubmittedEvent.
func TestEngine_PublishDomainEventOrderSubmitted(t *testing.T) {
	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	engine := New(ctx, nil, nil, logger)
	err := engine.Start()
	require.NoError(t, err)
	defer engine.Stop()

	// Subscribe to events
	ch := make(chan Event, 10)
	subscriber := SubscriberFunc(func(ctx context.Context, event Event) {
		ch <- event
	})
	engine.eventBus.Subscribe(EventTypeOrderSubmitted, QoS1, subscriber)

	// Create domain event
	domainEvent := &events.OrderSubmittedEvent{
		BaseEvent: events.BaseEvent{
			EventType:      events.EventTypeOrderSubmitted,
			OccurredAt:     time.Now().UTC(),
			CorrelationID_: "test-123",
		},
		OrderID:         "order-1",
		ClientOrderID:   "client-1",
		ExchangeOrderID: "exchange-1",
		SubmittedAtUTC:  time.Now().UTC(),
	}

	// Publish domain event
	engine.PublishDomainEvent(domainEvent)

	// Wait for event
	select {
	case event := <-ch:
		assert.Equal(t, EventTypeOrderSubmitted, event.Type())
	case <-time.After(1 * time.Second):
		t.Fatal("Event not received")
	}
}

// TestEngine_PublishDomainEventOrderFilled tests publishing OrderFilledEvent.
func TestEngine_PublishDomainEventOrderFilled(t *testing.T) {
	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	engine := New(ctx, nil, nil, logger)
	err := engine.Start()
	require.NoError(t, err)
	defer engine.Stop()

	// Subscribe to events
	ch := make(chan Event, 10)
	subscriber := SubscriberFunc(func(ctx context.Context, event Event) {
		ch <- event
	})
	engine.eventBus.Subscribe(EventTypeOrderFilled, QoS1, subscriber)

	// Create domain event
	domainEvent := &events.OrderFilledEvent{
		BaseEvent: events.BaseEvent{
			EventType:      events.EventTypeOrderFilled,
			OccurredAt:     time.Now().UTC(),
			CorrelationID_: "test-123",
		},
		OrderID:      "order-1",
		FilledQty:    decimal.RequireFromString("1"),
		AvgFillPrice: decimal.RequireFromString("50000"),
		FilledAtUTC:  time.Now().UTC(),
	}

	// Publish domain event
	engine.PublishDomainEvent(domainEvent)

	// Wait for event
	select {
	case event := <-ch:
		assert.Equal(t, EventTypeOrderFilled, event.Type())
	case <-time.After(1 * time.Second):
		t.Fatal("Event not received")
	}
}

// TestEngine_PublishDomainEventOrderCanceled tests publishing OrderCanceledEvent.
func TestEngine_PublishDomainEventOrderCanceled(t *testing.T) {
	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	engine := New(ctx, nil, nil, logger)
	err := engine.Start()
	require.NoError(t, err)
	defer engine.Stop()

	// Subscribe to events
	ch := make(chan Event, 10)
	subscriber := SubscriberFunc(func(ctx context.Context, event Event) {
		ch <- event
	})
	engine.eventBus.Subscribe(EventTypeOrderCanceled, QoS1, subscriber)

	// Create domain event
	domainEvent := &events.OrderCanceledEvent{
		BaseEvent: events.BaseEvent{
			EventType:      events.EventTypeOrderCanceled,
			OccurredAt:     time.Now().UTC(),
			CorrelationID_: "test-123",
		},
		OrderID:       "order-1",
		CanceledAtUTC: time.Now().UTC(),
	}

	// Publish domain event
	engine.PublishDomainEvent(domainEvent)

	// Wait for event
	select {
	case event := <-ch:
		assert.Equal(t, EventTypeOrderCanceled, event.Type())
	case <-time.After(1 * time.Second):
		t.Fatal("Event not received")
	}
}

// TestEngine_PublishDomainEventOrderRejected tests publishing OrderRejectedEvent.
func TestEngine_PublishDomainEventOrderRejected(t *testing.T) {
	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	engine := New(ctx, nil, nil, logger)
	err := engine.Start()
	require.NoError(t, err)
	defer engine.Stop()

	// Subscribe to events
	ch := make(chan Event, 10)
	subscriber := SubscriberFunc(func(ctx context.Context, event Event) {
		ch <- event
	})
	engine.eventBus.Subscribe(EventTypeOrderRejected, QoS1, subscriber)

	// Create domain event
	domainEvent := &events.OrderRejectedEvent{
		BaseEvent: events.BaseEvent{
			EventType:      events.EventTypeOrderRejected,
			OccurredAt:     time.Now().UTC(),
			CorrelationID_: "test-123",
		},
		OrderID:       "order-1",
		Reason:        "Insufficient balance",
		RejectedAtUTC: time.Now().UTC(),
	}

	// Publish domain event
	engine.PublishDomainEvent(domainEvent)

	// Wait for event
	select {
	case event := <-ch:
		assert.Equal(t, EventTypeOrderRejected, event.Type())
	case <-time.After(1 * time.Second):
		t.Fatal("Event not received")
	}
}

// TestEngine_ReconciliationWorkerInLiveTrading tests reconciliation worker in live trading state.
func TestEngine_ReconciliationWorkerInLiveTrading(t *testing.T) {
	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	engine := New(ctx, nil, nil, logger)
	err := engine.Start()
	require.NoError(t, err)
	defer engine.Stop()

	// Transition to live trading
	err = engine.TransitionTo(StateLiveTrading, "test")
	require.NoError(t, err)

	// Wait for reconciliation worker to run
	time.Sleep(100 * time.Millisecond)

	// Worker should have run (no orderService, so just logs)
	assert.Equal(t, StateLiveTrading, engine.State())
}

// TestEngine_ReconciliationWorkerInPaperTrading tests reconciliation worker in paper trading state.
func TestEngine_ReconciliationWorkerInPaperTrading(t *testing.T) {
	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	engine := New(ctx, nil, nil, logger)
	err := engine.Start()
	require.NoError(t, err)
	defer engine.Stop()

	// Transition to paper trading
	err = engine.TransitionTo(StatePaperTrading, "test")
	require.NoError(t, err)

	// Wait for reconciliation worker to run
	time.Sleep(100 * time.Millisecond)

	// Worker should have run (no orderService, so just logs)
	assert.Equal(t, StatePaperTrading, engine.State())
}

// TestEngine_StopWhenAlreadyHalted tests stopping when already halted.
func TestEngine_StopWhenAlreadyHalted(t *testing.T) {
	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	engine := New(ctx, nil, nil, logger)
	err := engine.Start()
	require.NoError(t, err)

	// Transition to halted
	err = engine.TransitionTo(StateHalted, "test halt")
	require.NoError(t, err)

	// Stop should work even when already halted
	err = engine.Stop()
	assert.NoError(t, err)
	assert.Equal(t, StateHalted, engine.State())
}

// TestEngine_NewWithNilLogger tests engine creation with nil logger.
func TestEngine_NewWithNilLogger(t *testing.T) {
	ctx := context.Background()

	engine := New(ctx, nil, nil, nil)

	assert.NotNil(t, engine)
	assert.NotNil(t, engine.logger)
}

// TestEngine_MultipleStateTransitions tests multiple state transitions.
func TestEngine_MultipleStateTransitions(t *testing.T) {
	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	engine := New(ctx, nil, nil, logger)
	err := engine.Start()
	require.NoError(t, err)
	defer engine.Stop()

	// Idle -> Scanning
	err = engine.TransitionTo(StateScanning, "start scanning")
	assert.NoError(t, err)
	assert.Equal(t, StateScanning, engine.State())

	// Scanning -> LiveTrading
	err = engine.TransitionTo(StateLiveTrading, "start trading")
	assert.NoError(t, err)
	assert.Equal(t, StateLiveTrading, engine.State())

	// LiveTrading -> PaperTrading
	err = engine.TransitionTo(StatePaperTrading, "switch to paper")
	assert.NoError(t, err)
	assert.Equal(t, StatePaperTrading, engine.State())

	// PaperTrading -> Idle
	err = engine.TransitionTo(StateIdle, "stop trading")
	assert.NoError(t, err)
	assert.Equal(t, StateIdle, engine.State())
}

// TestEngine_LongRunningWorkers tests that workers continue running.
func TestEngine_LongRunningWorkers(t *testing.T) {
	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	engine := New(ctx, nil, nil, logger)
	err := engine.Start()
	require.NoError(t, err)

	// Let workers run for a bit
	time.Sleep(500 * time.Millisecond)

	// Engine should still be running
	assert.NotEqual(t, StateHalted, engine.State())

	// Stop engine
	err = engine.Stop()
	assert.NoError(t, err)
}

// TestEngine_StateTransitionsInAllStates tests transitions from all states.
func TestEngine_StateTransitionsInAllStates(t *testing.T) {
	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	engine := New(ctx, nil, nil, logger)
	err := engine.Start()
	require.NoError(t, err)
	defer engine.Stop()

	states := []State{StateScanning, StateLiveTrading, StatePaperTrading, StateBacktesting, StateOptimizing}

	for _, state := range states {
		// Transition to state
		err = engine.TransitionTo(state, "test")
		assert.NoError(t, err)
		assert.Equal(t, state, engine.State())

		// Transition back to Idle
		err = engine.TransitionTo(StateIdle, "reset")
		assert.NoError(t, err)
		assert.Equal(t, StateIdle, engine.State())
	}
}

type streamEventOrderService struct {
	reconciled chan struct{}
}

type streamEventReconciler struct {
	reconciled chan struct{}
}

func (r *streamEventReconciler) ReconcileAll(ctx context.Context) error {
	select {
	case r.reconciled <- struct{}{}:
	default:
	}
	return nil
}

func (s *streamEventOrderService) CreateOrder(ctx context.Context, req agent.CreateOrderRequest) (*entity.Order, error) {
	return nil, nil
}

func (s *streamEventOrderService) SubmitOrder(ctx context.Context, orderID string) error {
	return nil
}

func (s *streamEventOrderService) CancelOrder(ctx context.Context, orderID string) error {
	return nil
}

func (s *streamEventOrderService) GetOrder(ctx context.Context, orderID string) (*entity.Order, error) {
	return nil, nil
}

func (s *streamEventOrderService) ListOrders(ctx context.Context, strategyID string, limit, offset int) ([]*entity.Order, error) {
	return nil, nil
}

func (s *streamEventOrderService) ReconcileOrders(ctx context.Context) error {
	select {
	case s.reconciled <- struct{}{}:
	default:
	}
	return nil
}

type streamEventConnector struct {
	events chan *domainconnector.StreamEvent
}

func (c *streamEventConnector) Name() string {
	return "stream-test"
}

func (c *streamEventConnector) Connect(ctx context.Context) error {
	return nil
}

func (c *streamEventConnector) Disconnect(ctx context.Context) error {
	return nil
}

func (c *streamEventConnector) IsConnected() bool {
	return true
}

func (c *streamEventConnector) SubmitOrder(ctx context.Context, order *entity.Order) (string, error) {
	return "", nil
}

func (c *streamEventConnector) CancelOrder(ctx context.Context, orderID string) error {
	return nil
}

func (c *streamEventConnector) GetOpenOrders(ctx context.Context) ([]*entity.Order, error) {
	return nil, nil
}

func (c *streamEventConnector) QueryOrder(ctx context.Context, orderID string) (*entity.Order, error) {
	return nil, nil
}

func (c *streamEventConnector) GetPosition(ctx context.Context, symbol string) (*entity.Position, error) {
	return nil, nil
}

func (c *streamEventConnector) GetBalance(ctx context.Context) (*entity.AccountBalance, error) {
	return nil, nil
}

func (c *streamEventConnector) GetSymbol(ctx context.Context, symbol string) (*entity.Symbol, error) {
	return nil, nil
}

func (c *streamEventConnector) RefreshSymbols(ctx context.Context) error {
	return nil
}

func (c *streamEventConnector) CheckPermissions(ctx context.Context) (*domainconnector.Permissions, error) {
	return &domainconnector.Permissions{CanRead: true, CanTrade: true}, nil
}

func (c *streamEventConnector) SubscribeOrders(ctx context.Context) (<-chan *domainconnector.OrderUpdate, error) {
	return nil, nil
}

func (c *streamEventConnector) SubscribePositions(ctx context.Context) (<-chan *domainconnector.PositionUpdate, error) {
	return nil, nil
}

func (c *streamEventConnector) SubscribeBalance(ctx context.Context) (<-chan *domainconnector.BalanceUpdate, error) {
	return nil, nil
}

func (c *streamEventConnector) SubscribeTicker(ctx context.Context, symbol string) (<-chan *domainconnector.TickerUpdate, error) {
	return nil, nil
}

func (c *streamEventConnector) SubscribeStreamEvents(ctx context.Context) (<-chan *domainconnector.StreamEvent, error) {
	return c.events, nil
}
