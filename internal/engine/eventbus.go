package engine

import (
	"context"
	"runtime/debug"
	"sync"
	"sync/atomic"

	"github.com/sirupsen/logrus"
)

// EventType identifies the type of event.
type EventType string

const (
	// Market data events
	EventTypeMarketData EventType = "market_data"
	EventTypeTicker     EventType = "ticker"

	// Trading signal events
	EventTypeSignal EventType = "signal"

	// Order lifecycle events
	EventTypeOrderIntent    EventType = "order_intent"
	EventTypeOrderCreated   EventType = "order_created"
	EventTypeOrderSubmitted EventType = "order_submitted"
	EventTypeOrderFilled    EventType = "order_filled"
	EventTypeOrderPartial   EventType = "order_partial"
	EventTypeOrderCanceled  EventType = "order_canceled"
	EventTypeOrderRejected  EventType = "order_rejected"

	// Position events
	EventTypePositionUpdate EventType = "position_update"

	// Balance events
	EventTypeBalanceUpdate EventType = "balance_update"

	// Risk events
	EventTypeRiskViolation EventType = "risk_violation"
	EventTypeRiskWarning   EventType = "risk_warning"

	// System events
	EventTypeStateTransition EventType = "state_transition"
	EventTypeAlert           EventType = "alert"
	EventTypeError           EventType = "error"
)

// QoS defines the quality of service level for event delivery.
type QoS int

const (
	// QoS0 (Best Effort) - events may be dropped if consumer is slow.
	// Used for high-frequency market data that can be skipped.
	QoS0 QoS = 0

	// QoS1 (Guaranteed) - events are never dropped, buffered if needed.
	// Used for critical events like orders, balances, risk violations.
	QoS1 QoS = 1
)

// Event is the base interface for all events in the system.
type Event interface {
	Type() EventType
	QoS() QoS
}

// Subscriber receives events from the event bus.
type Subscriber interface {
	OnEvent(ctx context.Context, event Event)
}

// SubscriberFunc is a function adapter for Subscriber interface.
type SubscriberFunc func(ctx context.Context, event Event)

func (f SubscriberFunc) OnEvent(ctx context.Context, event Event) {
	f(ctx, event)
}

// SubscriptionID uniquely identifies a subscription.
type SubscriptionID uint64

// EventBus implements a non-blocking event distribution system with QoS support.
type EventBus struct {
	mu            sync.RWMutex
	subscribers   map[EventType]map[SubscriptionID]*subscription
	nextID        uint64
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
	logger        *logrus.Logger
	droppedEvents uint64 // Atomic counter for dropped QoS0 events
	closed        atomic.Bool
}

type subscription struct {
	id       SubscriptionID
	qos      QoS
	ch       chan Event
	sub      Subscriber
	cancel   context.CancelFunc
	stopOnce sync.Once
}

// NewEventBus creates a new event bus.
func NewEventBus(ctx context.Context, logger *logrus.Logger) *EventBus {
	if logger == nil {
		logger = logrus.New()
	}

	ctx, cancel := context.WithCancel(ctx)
	return &EventBus{
		subscribers: make(map[EventType]map[SubscriptionID]*subscription),
		nextID:      1,
		ctx:         ctx,
		cancel:      cancel,
		logger:      logger,
	}
}

// Subscribe registers a subscriber for specific event types.
// Returns a SubscriptionID that can be used to unsubscribe.
// For QoS0, uses a small buffer and non-blocking send.
// For QoS1, uses a large buffer to prevent drops.
func (eb *EventBus) Subscribe(eventType EventType, qos QoS, sub Subscriber) SubscriptionID {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	if eb.closed.Load() || eb.ctx.Err() != nil {
		return 0
	}

	// Generate unique subscription ID
	id := SubscriptionID(atomic.AddUint64(&eb.nextID, 1))

	// Create channel with appropriate buffer size
	var ch chan Event
	if qos == QoS0 {
		ch = make(chan Event, 10) // Small buffer for best-effort
	} else {
		ch = make(chan Event, 1000) // Large buffer for guaranteed delivery
	}

	// Create subscription context
	subCtx, subCancel := context.WithCancel(eb.ctx)

	s := &subscription{
		id:     id,
		qos:    qos,
		ch:     ch,
		sub:    sub,
		cancel: subCancel,
	}

	// Initialize map if needed
	if eb.subscribers[eventType] == nil {
		eb.subscribers[eventType] = make(map[SubscriptionID]*subscription)
	}

	eb.subscribers[eventType][id] = s

	// Start goroutine to deliver events to subscriber
	eb.wg.Add(1)
	go eb.deliverEvents(subCtx, s)

	eb.logger.WithFields(logrus.Fields{
		"subscription_id": id,
		"event_type":      eventType,
		"qos":             qos,
	}).Debug("Subscriber registered")

	return id
}

// Unsubscribe removes a subscriber and stops its delivery goroutine.
func (eb *EventBus) Unsubscribe(eventType EventType, id SubscriptionID) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	subs, ok := eb.subscribers[eventType]
	if !ok {
		return
	}

	sub, ok := subs[id]
	if !ok {
		return
	}

	// Stop the delivery goroutine
	sub.stopOnce.Do(func() {
		sub.cancel()
		close(sub.ch)
	})

	delete(subs, id)

	// Clean up empty map
	if len(subs) == 0 {
		delete(eb.subscribers, eventType)
	}

	eb.logger.WithFields(logrus.Fields{
		"subscription_id": id,
		"event_type":      eventType,
	}).Debug("Subscriber unregistered")
}

// Publish sends an event to all subscribers of that event type.
func (eb *EventBus) Publish(event Event) {
	if event == nil || eb.closed.Load() {
		return
	}
	select {
	case <-eb.ctx.Done():
		return
	default:
	}

	eb.mu.RLock()
	eventType := event.Type()
	subs, ok := eb.subscribers[eventType]
	if !ok {
		eb.mu.RUnlock()
		return
	}

	// Copy subscription slice to avoid holding RLock during channel sends.
	// This prevents deadlock when a subscriber's OnEvent calls Subscribe/Unsubscribe.
	channels := make([]*subscription, 0, len(subs))
	for _, sub := range subs {
		channels = append(channels, sub)
	}
	eb.mu.RUnlock()

	for _, sub := range channels {
		if sub.qos == QoS0 {
			// QoS0: Non-blocking send, drop if channel full
			select {
			case sub.ch <- event:
			default:
				// Event dropped - consumer too slow
				dropped := atomic.AddUint64(&eb.droppedEvents, 1)
				if dropped%100 == 0 { // Log every 100th drop to avoid spam
					eb.logger.WithFields(logrus.Fields{
						"event_type":      eventType,
						"total_dropped":   dropped,
						"subscription_id": sub.id,
					}).Warn("QoS0 events being dropped due to slow consumer")
				}
			}
		} else {
			// QoS1: Blocking send, guaranteed delivery
			select {
			case sub.ch <- event:
			case <-eb.ctx.Done():
				return
			}
		}
	}
}

// deliverEvents runs in a goroutine to deliver events to a subscriber.
func (eb *EventBus) deliverEvents(ctx context.Context, sub *subscription) {
	defer eb.wg.Done()

	for {
		select {
		case event, ok := <-sub.ch:
			if !ok {
				// Channel closed, exit
				return
			}
			eb.safeOnEvent(ctx, sub, event)
		case <-ctx.Done():
			// Drain remaining events before exit
			for {
				select {
				case event, ok := <-sub.ch:
					if !ok {
						return
					}
					eb.safeOnEvent(ctx, sub, event)
				default:
					return
				}
			}
		}
	}
}

func (eb *EventBus) safeOnEvent(ctx context.Context, sub *subscription, event Event) {
	defer func() {
		if r := recover(); r != nil {
			eb.logger.WithFields(logrus.Fields{
				"subscription_id": sub.id,
				"event_type":      event.Type(),
				"panic":           r,
				"stack":           string(debug.Stack()),
			}).Error("Event subscriber panic recovered")
		}
	}()

	sub.sub.OnEvent(ctx, event)
}

// Shutdown stops the event bus and waits for all delivery goroutines to exit.
func (eb *EventBus) Shutdown() {
	if !eb.closed.CompareAndSwap(false, true) {
		return
	}

	eb.mu.Lock()

	// Cancel all subscriptions
	for eventType, subs := range eb.subscribers {
		for id, sub := range subs {
			sub.stopOnce.Do(func() {
				sub.cancel()
				close(sub.ch)
			})
			delete(subs, id)
		}
		delete(eb.subscribers, eventType)
	}

	eb.mu.Unlock()

	// Cancel main context
	eb.cancel()

	// Wait for all delivery goroutines to exit
	eb.wg.Wait()

	eb.logger.Info("EventBus shutdown complete")
}

// GetDroppedEventCount returns the total number of dropped QoS0 events.
func (eb *EventBus) GetDroppedEventCount() uint64 {
	return atomic.LoadUint64(&eb.droppedEvents)
}

// BaseEvent provides common fields for all events.
type BaseEvent struct {
	EventType EventType
	EventQoS  QoS
}

func (e *BaseEvent) Type() EventType {
	return e.EventType
}

func (e *BaseEvent) QoS() QoS {
	return e.EventQoS
}
