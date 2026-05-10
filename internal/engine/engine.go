package engine

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"github.com/kairos-development/kairos-agent/internal/circuitbreaker"
	domainconnector "github.com/kairos-development/kairos-agent/internal/domain/connector"
	"github.com/kairos-development/kairos-agent/internal/domain/events"
	"github.com/kairos-development/kairos-agent/internal/metrics"
	"github.com/kairos-development/kairos-agent/internal/service/agent"
	"github.com/kairos-development/kairos-agent/internal/strategy/wasm"
	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
)

// StorageProvider defines the interface for storage operations.
type StorageProvider interface {
	Close() error
}

// ReconciliationService defines full state reconciliation operations.
type ReconciliationService interface {
	// ReconcileAll reconciles local orders and positions with exchange REST state.
	ReconcileAll(ctx context.Context) error
}

// Engine is the core trading engine that orchestrates all components.
type Engine struct {
	mu sync.RWMutex

	// Core components
	stateMachine *StateMachine
	eventBus     *EventBus
	logger       *logrus.Logger
	metrics      *metrics.Metrics

	// Circuit breakers
	ntpCircuitBreaker *circuitbreaker.CircuitBreaker

	// NTP failure tracking
	ntpConsecutiveFailures int
	ntpLastSuccessfulSync  time.Time

	// Services
	orderService     agent.OrderService
	reconciler       ReconciliationService
	connector        domainconnector.Connector
	storage          StorageProvider
	strategyExecutor *wasm.StrategyExecutor

	// Context and lifecycle
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Configuration
	config *Config
}

// Config contains engine configuration.
type Config struct {
	// NTP drift tolerance in milliseconds
	MaxNTPDrift time.Duration

	// Rate limiter settings
	MaxOrdersPerSecond int

	// Risk settings
	MaxPositionSize decimal.Decimal
	MaxDailyLoss    decimal.Decimal

	// NTP failure tolerance - number of consecutive failures before halting
	NTPFailureTolerance int

	// NTP degraded mode - allow trading with stale NTP data for this duration
	NTPDegradedModeDuration time.Duration
}

// DefaultConfig returns default engine configuration.
func DefaultConfig() *Config {
	return &Config{
		MaxNTPDrift:             500 * time.Millisecond,
		MaxOrdersPerSecond:      10,
		MaxPositionSize:         decimal.RequireFromString("10000"),
		MaxDailyLoss:            decimal.RequireFromString("1000"),
		NTPFailureTolerance:     5,                // Allow 5 consecutive failures
		NTPDegradedModeDuration: 10 * time.Minute, // Allow 10 minutes of stale NTP data
	}
}

// New creates a new trading engine.
func New(ctx context.Context, orderService agent.OrderService, config *Config, logger *logrus.Logger) *Engine {
	if config == nil {
		config = DefaultConfig()
	}

	if logger == nil {
		logger = logrus.New()
	}

	ctx, cancel := context.WithCancel(ctx)

	logger.WithFields(logrus.Fields{
		"max_ntp_drift":         config.MaxNTPDrift,
		"max_orders_per_second": config.MaxOrdersPerSecond,
		"max_position_size":     config.MaxPositionSize,
		"max_daily_loss":        config.MaxDailyLoss,
	}).Info("Engine initialized with config")

	// Initialize metrics
	m := metrics.New("kairos")

	// Initialize circuit breakers
	ntpCB := circuitbreaker.New("ntp", circuitbreaker.Config{
		MaxRequests: 3,
		Interval:    60 * time.Second,
		Timeout:     30 * time.Second,
		ReadyToTrip: func(counts circuitbreaker.Counts) bool {
			// Trip after 3 consecutive failures
			return counts.ConsecutiveFailures >= 3
		},
		OnStateChange: func(name string, from circuitbreaker.State, to circuitbreaker.State) {
			logger.WithFields(logrus.Fields{
				"circuit_breaker": name,
				"from":            from.String(),
				"to":              to.String(),
			}).Warn("Circuit breaker state changed")
		},
	})

	return &Engine{
		stateMachine:      NewStateMachine(logger),
		eventBus:          NewEventBus(ctx, logger),
		logger:            logger,
		metrics:           m,
		ntpCircuitBreaker: ntpCB,
		orderService:      orderService,
		ctx:               ctx,
		cancel:            cancel,
		config:            config,
	}
}

// Start starts the engine and all background workers.
func (e *Engine) Start() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.logger.Info("Starting engine")

	// Subscribe to state transitions
	stateTransitionCh := make(chan StateTransition, 10)
	e.stateMachine.Subscribe(stateTransitionCh)

	// Start state transition listener
	e.wg.Add(1)
	go e.handleStateTransitions(stateTransitionCh)

	// Start NTP sync worker
	e.wg.Add(1)
	go e.ntpSyncWorker()

	// Start reconciliation worker
	e.wg.Add(1)
	go e.reconciliationWorker()

	// Start stream lifecycle worker
	e.wg.Add(1)
	go e.streamEventWorker()

	// Start metrics worker
	e.wg.Add(1)
	go e.metricsWorker()

	e.logger.Info("Engine started successfully")
	return nil
}

// Stop gracefully stops the engine.
func (e *Engine) Stop() error {
	e.logger.Info("Stopping engine")

	// Transition to Halted state (if not already halted)
	if e.stateMachine.Current() != StateHalted {
		if err := e.stateMachine.Transition(StateHalted, "shutdown requested"); err != nil {
			e.logger.WithError(err).Error("Failed to transition to halted state")
			return fmt.Errorf("transition to halted: %w", err)
		}
	}

	// Cancel context to stop all workers
	e.mu.Lock()
	e.cancel()
	e.mu.Unlock()

	// Wait for all workers to finish
	e.wg.Wait()

	// Shutdown event bus
	e.eventBus.Shutdown()

	e.logger.Info("Engine stopped successfully")
	return nil
}

// State returns the current engine state.
func (e *Engine) State() State {
	return e.stateMachine.Current()
}

// TransitionTo attempts to transition the engine to a new state.
func (e *Engine) TransitionTo(state State, reason string) error {
	e.logger.WithFields(logrus.Fields{
		"from":   e.stateMachine.Current().String(),
		"to":     state.String(),
		"reason": reason,
	}).Info("Transitioning state")

	return e.stateMachine.Transition(state, reason)
}

// EventBus returns the event bus for subscribing to events.
func (e *Engine) EventBus() *EventBus {
	return e.eventBus
}

// handleStateTransitions listens for state transitions and publishes events.
func (e *Engine) handleStateTransitions(ch <-chan StateTransition) {
	defer e.wg.Done()
	defer e.recoverWorker("state_transitions")

	for {
		select {
		case transition, ok := <-ch:
			if !ok {
				return
			}
			e.logger.WithFields(logrus.Fields{
				"from":      transition.From.String(),
				"to":        transition.To.String(),
				"reason":    transition.Reason,
				"timestamp": transition.Timestamp,
			}).Info("State transition occurred")

			// Record metrics
			e.metrics.StateTransitions.WithLabelValues(
				transition.From.String(),
				transition.To.String(),
			).Inc()

			// Publish state transition event to event bus
			event := &StateTransitionEvent{
				BaseEvent: BaseEvent{
					EventType: EventTypeStateTransition,
					EventQoS:  QoS1, // Critical event
				},
				From:      transition.From,
				To:        transition.To,
				Timestamp: transition.Timestamp,
				Reason:    transition.Reason,
			}
			e.eventBus.Publish(event)

		case <-e.ctx.Done():
			return
		}
	}
}

// ntpSyncWorker periodically checks NTP drift and halts trading if drift is too high.
func (e *Engine) ntpSyncWorker() {
	defer e.wg.Done()
	defer e.recoverWorker("ntp_sync")

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	ntpWorker := NewNTPSyncWorker(e.config.MaxNTPDrift)

	for {
		select {
		case <-ticker.C:
			var drift time.Duration

			// Use circuit breaker for NTP call
			err := e.ntpCircuitBreaker.Execute(e.ctx, func() error {
				d, err := ntpWorker.Check(e.ctx)
				drift = d
				return err
			})

			if err != nil {
				e.mu.Lock()
				e.ntpConsecutiveFailures++
				failures := e.ntpConsecutiveFailures
				lastSuccess := e.ntpLastSuccessfulSync
				e.mu.Unlock()

				if err == circuitbreaker.ErrCircuitOpen {
					e.logger.WithField("consecutive_failures", failures).Warn("NTP circuit breaker is open")
				} else {
					e.logger.WithError(err).WithField("consecutive_failures", failures).Warn("NTP sync check failed")
				}
				e.metrics.NTPSyncErrors.Inc()

				// Check if we should halt due to NTP failures
				timeSinceLastSuccess := time.Since(lastSuccess)
				shouldHalt := false

				if failures >= e.config.NTPFailureTolerance {
					e.logger.WithFields(logrus.Fields{
						"consecutive_failures": failures,
						"tolerance":            e.config.NTPFailureTolerance,
					}).Warn("NTP failure tolerance exceeded")
					shouldHalt = true
				}

				if !lastSuccess.IsZero() && timeSinceLastSuccess > e.config.NTPDegradedModeDuration {
					e.logger.WithFields(logrus.Fields{
						"time_since_last_success": timeSinceLastSuccess,
						"degraded_mode_duration":  e.config.NTPDegradedModeDuration,
					}).Warn("NTP degraded mode duration exceeded")
					shouldHalt = true
				}

				if shouldHalt {
					e.logger.Error("Halting trading due to prolonged NTP failures")
					if err := e.stateMachine.Transition(StateHalted, "NTP sync failures exceeded tolerance"); err != nil {
						e.logger.WithError(err).Error("Failed to transition to halted state")
					}

					// Publish alert
					alert := &AlertEvent{
						BaseEvent: BaseEvent{
							EventType: EventTypeAlert,
							EventQoS:  QoS1,
						},
						Level:     AlertLevelCritical,
						Message:   fmt.Sprintf("NTP sync failures exceeded tolerance: %d consecutive failures", failures),
						Timestamp: time.Now().UTC(),
					}
					e.eventBus.Publish(alert)
				} else {
					e.logger.WithFields(logrus.Fields{
						"consecutive_failures":    failures,
						"time_since_last_success": timeSinceLastSuccess,
					}).Info("Operating in NTP degraded mode")
				}

				continue
			}

			// NTP sync successful - reset failure counter
			e.mu.Lock()
			e.ntpConsecutiveFailures = 0
			e.ntpLastSuccessfulSync = time.Now().UTC()
			e.mu.Unlock()

			e.logger.WithField("drift_ms", drift.Milliseconds()).Debug("NTP drift checked")

			// Update metrics
			e.metrics.NTPDrift.Set(float64(drift.Milliseconds()))
			e.metrics.NTPLastSync.Set(float64(ntpWorker.LastSync().Unix()))

			if !ntpWorker.IsHealthy() {
				// Halt trading due to excessive time drift
				e.logger.WithField("drift_ms", drift.Milliseconds()).Error("NTP drift exceeds threshold")
				if err := e.stateMachine.Transition(StateHalted, fmt.Sprintf("NTP drift too high: %v", drift)); err != nil {
					e.logger.WithError(err).Error("Failed to transition to halted state")
				}

				// Publish alert
				alert := &AlertEvent{
					BaseEvent: BaseEvent{
						EventType: EventTypeAlert,
						EventQoS:  QoS1,
					},
					Level:     AlertLevelCritical,
					Message:   fmt.Sprintf("NTP drift exceeds threshold: %v", drift),
					Timestamp: time.Now().UTC(),
				}
				e.eventBus.Publish(alert)
			}

		case <-e.ctx.Done():
			return
		}
	}
}

// streamEventWorker reacts to connector WebSocket lifecycle events.
func (e *Engine) streamEventWorker() {
	defer e.wg.Done()
	defer e.recoverWorker("stream_events")

	retryTicker := time.NewTicker(1 * time.Second)
	defer retryTicker.Stop()

	var streamEvents <-chan *domainconnector.StreamEvent

	for {
		if streamEvents == nil {
			conn := e.GetConnector()
			if conn == nil {
				if !e.waitForStreamEventRetry(retryTicker) {
					return
				}
				continue
			}

			subscriber, ok := conn.(domainconnector.StreamEventSubscriber)
			if !ok {
				if !e.waitForStreamEventRetry(retryTicker) {
					return
				}
				continue
			}

			ch, err := subscriber.SubscribeStreamEvents(e.ctx)
			if err != nil {
				e.logger.WithError(err).Debug("Stream event subscription unavailable")
				if !e.waitForStreamEventRetry(retryTicker) {
					return
				}
				continue
			}
			streamEvents = ch
		}

		select {
		case event, ok := <-streamEvents:
			if !ok {
				streamEvents = nil
				if !e.waitForStreamEventRetry(retryTicker) {
					return
				}
				continue
			}
			e.handleStreamEvent(event)

		case <-e.ctx.Done():
			return
		}
	}
}

func (e *Engine) waitForStreamEventRetry(ticker *time.Ticker) bool {
	select {
	case <-ticker.C:
		return true
	case <-e.ctx.Done():
		return false
	}
}

func (e *Engine) handleStreamEvent(event *domainconnector.StreamEvent) {
	if event == nil {
		return
	}

	occurredAt := event.OccurredAtUTC
	if occurredAt.IsZero() {
		occurredAt = time.Now().UTC()
	}

	e.logger.WithFields(logrus.Fields{
		"type":        event.Type,
		"source":      event.Source,
		"reason":      event.Reason,
		"occurred_at": occurredAt,
	}).Warn("Connector stream lifecycle event received")

	switch event.Type {
	case domainconnector.StreamEventDisconnected:
		e.publishStreamLifecycleAlert(AlertLevelWarning, event, "Connector stream disconnected")
		e.haltLiveTradingForStreamEvent(event)

	case domainconnector.StreamEventGap:
		e.publishStreamLifecycleAlert(AlertLevelCritical, event, "Connector stream gap detected")
		e.haltLiveTradingForStreamEvent(event)
		e.reconcileAfterStreamEvent(event)

	case domainconnector.StreamEventReconnected:
		e.publishStreamLifecycleAlert(AlertLevelWarning, event, "Connector stream reconnected")
		e.reconcileAfterStreamEvent(event)

	default:
		e.publishStreamLifecycleAlert(AlertLevelWarning, event, "Connector stream event received")
	}
}

func (e *Engine) haltLiveTradingForStreamEvent(event *domainconnector.StreamEvent) {
	if e.stateMachine.Current() != StateLiveTrading {
		return
	}

	reason := fmt.Sprintf("connector stream %s: %s", event.Type, event.Reason)
	if err := e.stateMachine.Transition(StateHalted, reason); err != nil {
		e.logger.WithError(err).WithField("stream_event", event.Type).Error("Failed to halt live trading after stream event")
	}
}

func (e *Engine) reconcileAfterStreamEvent(event *domainconnector.StreamEvent) {
	ctx, cancel := context.WithTimeout(e.ctx, 30*time.Second)
	defer cancel()

	e.mu.RLock()
	reconciler := e.reconciler
	orderService := e.orderService
	e.mu.RUnlock()

	if reconciler != nil {
		if err := reconciler.ReconcileAll(ctx); err != nil {
			e.publishStreamReconciliationError(event, err)
			return
		}
		e.logger.WithField("stream_event", event.Type).Info("Full reconciliation completed after stream event")
		return
	}

	if orderService != nil {
		if err := orderService.ReconcileOrders(ctx); err != nil {
			e.publishStreamReconciliationError(event, err)
			return
		}
		e.logger.WithField("stream_event", event.Type).Info("Order reconciliation completed after stream event")
		return
	}

	e.logger.WithField("stream_event", event.Type).Warn("No reconciliation service configured for stream event")
}

func (e *Engine) publishStreamLifecycleAlert(level AlertLevel, event *domainconnector.StreamEvent, message string) {
	alert := &AlertEvent{
		BaseEvent: BaseEvent{
			EventType: EventTypeAlert,
			EventQoS:  QoS1,
		},
		Level:     level,
		Message:   fmt.Sprintf("%s: %s", message, event.Reason),
		Timestamp: time.Now().UTC(),
	}
	e.eventBus.Publish(alert)
}

func (e *Engine) publishStreamReconciliationError(event *domainconnector.StreamEvent, err error) {
	e.logger.WithError(err).WithField("stream_event", event.Type).Error("Reconciliation after stream event failed")

	errorEvent := &ErrorEvent{
		BaseEvent: BaseEvent{
			EventType: EventTypeError,
			EventQoS:  QoS1,
		},
		Error:     err,
		Message:   fmt.Sprintf("Reconciliation after stream event failed: %s", event.Type),
		Timestamp: time.Now().UTC(),
	}
	e.eventBus.Publish(errorEvent)
}

// reconciliationWorker periodically reconciles local state with exchange state.
func (e *Engine) reconciliationWorker() {
	defer e.wg.Done()
	defer e.recoverWorker("reconciliation")

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Only reconcile if in trading state
			state := e.stateMachine.Current()
			if state != StateLiveTrading && state != StatePaperTrading {
				continue
			}

			e.logger.Info("Starting order reconciliation")

			// Reconcile orders
			if e.orderService != nil {
				if err := e.orderService.ReconcileOrders(e.ctx); err != nil {
					// Log error but don't halt
					e.logger.WithError(err).Error("Order reconciliation failed")

					event := &ErrorEvent{
						BaseEvent: BaseEvent{
							EventType: EventTypeError,
							EventQoS:  QoS1,
						},
						Error:     err,
						Message:   "Order reconciliation failed",
						Timestamp: time.Now().UTC(),
					}
					e.eventBus.Publish(event)
				} else {
					e.logger.Info("Order reconciliation completed successfully")
				}
			}

		case <-e.ctx.Done():
			return
		}
	}
}

// metricsWorker periodically updates system metrics.
func (e *Engine) metricsWorker() {
	defer e.wg.Done()
	defer e.recoverWorker("metrics")

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Update system metrics
			e.metrics.UpdateSystemMetrics()

			// Update current state metric
			e.metrics.CurrentState.Set(float64(e.stateMachine.Current()))

			// Update event bus metrics
			droppedEvents := e.eventBus.GetDroppedEventCount()
			e.metrics.RecordEventDropped(droppedEvents)

			e.logger.Debug("Metrics updated")

		case <-e.ctx.Done():
			return
		}
	}
}

func (e *Engine) recoverWorker(name string) {
	r := recover()
	if r == nil {
		return
	}

	e.logger.WithFields(logrus.Fields{
		"worker": name,
		"panic":  r,
		"stack":  string(debug.Stack()),
	}).Error("Engine worker panic recovered")

	if err := e.stateMachine.Transition(StateHalted, fmt.Sprintf("%s worker panic recovered", name)); err != nil {
		e.logger.WithError(err).WithField("worker", name).Error("Failed to transition to halted after worker panic")
	}

	alert := &AlertEvent{
		BaseEvent: BaseEvent{
			EventType: EventTypeAlert,
			EventQoS:  QoS1,
		},
		Level:     AlertLevelCritical,
		Message:   fmt.Sprintf("Engine worker %s recovered from panic", name),
		Timestamp: time.Now().UTC(),
	}
	e.eventBus.Publish(alert)
}

// StateTransitionEvent is published when the engine state changes.
type StateTransitionEvent struct {
	BaseEvent
	From      State
	To        State
	Timestamp time.Time
	Reason    string
}

// AlertLevel defines the severity of an alert.
type AlertLevel int

const (
	AlertLevelInfo AlertLevel = iota
	AlertLevelWarning
	AlertLevelCritical
)

// AlertEvent is published for system alerts.
type AlertEvent struct {
	BaseEvent
	Level     AlertLevel
	Message   string
	Timestamp time.Time
}

// ErrorEvent is published when an error occurs.
type ErrorEvent struct {
	BaseEvent
	Error     error
	Message   string
	Timestamp time.Time
}

// PublishDomainEvent converts domain events to engine events and publishes them.
func (e *Engine) PublishDomainEvent(domainEvent events.Event) {
	var engineEvent Event

	switch ev := domainEvent.(type) {
	case *events.OrderCreatedEvent:
		engineEvent = &BaseEvent{
			EventType: EventTypeOrderCreated,
			EventQoS:  QoS1,
		}
	case *events.OrderSubmittedEvent:
		engineEvent = &BaseEvent{
			EventType: EventTypeOrderSubmitted,
			EventQoS:  QoS1,
		}
	case *events.OrderFilledEvent:
		engineEvent = &BaseEvent{
			EventType: EventTypeOrderFilled,
			EventQoS:  QoS1,
		}
	case *events.OrderCanceledEvent:
		engineEvent = &BaseEvent{
			EventType: EventTypeOrderCanceled,
			EventQoS:  QoS1,
		}
	case *events.OrderRejectedEvent:
		engineEvent = &BaseEvent{
			EventType: EventTypeOrderRejected,
			EventQoS:  QoS1,
		}
	default:
		// Unknown event type
		_ = ev
		return
	}

	e.eventBus.Publish(engineEvent)
}

// GetConnector returns the connector instance.
func (e *Engine) GetConnector() domainconnector.Connector {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.connector
}

// SetConnector sets the connector instance.
func (e *Engine) SetConnector(c domainconnector.Connector) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.connector = c
}

// SetReconciliationService sets the full reconciliation service.
func (e *Engine) SetReconciliationService(reconciler ReconciliationService) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.reconciler = reconciler
}

// GetStorage returns the storage provider instance.
func (e *Engine) GetStorage() StorageProvider {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.storage
}

// SetStorage sets the storage provider instance.
func (e *Engine) SetStorage(s StorageProvider) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.storage = s
}

// GetStrategyExecutor returns the strategy executor instance.
func (e *Engine) GetStrategyExecutor() *wasm.StrategyExecutor {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.strategyExecutor
}

// SetStrategyExecutor sets the strategy executor instance.
func (e *Engine) SetStrategyExecutor(se *wasm.StrategyExecutor) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.strategyExecutor = se
}

// GetEventBus returns the event bus instance.
func (e *Engine) GetEventBus() *EventBus {
	return e.eventBus
}
