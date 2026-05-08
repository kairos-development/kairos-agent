package events

import (
	"time"

	"github.com/shopspring/decimal"
)

// EventType identifies the domain event category.
type EventType string

const (
	EventTypeOrderCreated    EventType = "order.created"
	EventTypeOrderSubmitted  EventType = "order.submitted"
	EventTypeOrderPartial    EventType = "order.partial"
	EventTypeOrderFilled     EventType = "order.filled"
	EventTypeOrderCanceled   EventType = "order.canceled"
	EventTypeOrderRejected   EventType = "order.rejected"
	EventTypeOrderTimeout    EventType = "order.timeout"
	EventTypePositionOpened  EventType = "position.opened"
	EventTypePositionUpdated EventType = "position.updated"
	EventTypePositionClosed  EventType = "position.closed"
	EventTypeRiskViolation   EventType = "risk.violation"
	EventTypeRiskWarning     EventType = "risk.warning"
	EventTypeBalanceUpdated  EventType = "balance.updated"
	EventTypeStrategyStarted EventType = "strategy.started"
	EventTypeStrategyPaused  EventType = "strategy.paused"
	EventTypeStrategyStopped EventType = "strategy.stopped"
)

// Event is the base domain event interface.
type Event interface {
	Type() EventType
	OccurredAtUTC() time.Time
	CorrelationID() string
}

// BaseEvent provides common event fields.
type BaseEvent struct {
	EventType      EventType
	OccurredAt     time.Time
	CorrelationID_ string
}

func (e *BaseEvent) Type() EventType {
	return e.EventType
}

func (e *BaseEvent) OccurredAtUTC() time.Time {
	return e.OccurredAt
}

func (e *BaseEvent) CorrelationID() string {
	return e.CorrelationID_
}

// OrderCreatedEvent is emitted when a new order is created locally.
type OrderCreatedEvent struct {
	BaseEvent
	OrderID       string
	ClientOrderID string
	StrategyID    string
	Symbol        string
	Side          string
	OrderType     string
	Quantity      decimal.Decimal
	Price         decimal.Decimal
}

// OrderSubmittedEvent is emitted when an order is successfully submitted to the exchange.
type OrderSubmittedEvent struct {
	BaseEvent
	OrderID         string
	ClientOrderID   string
	ExchangeOrderID string
	SubmittedAtUTC  time.Time
}

// OrderPartialEvent is emitted when an order receives a partial fill.
type OrderPartialEvent struct {
	BaseEvent
	OrderID      string
	FilledQty    decimal.Decimal
	RemainingQty decimal.Decimal
	AvgFillPrice decimal.Decimal
}

// OrderFilledEvent is emitted when an order is completely filled.
type OrderFilledEvent struct {
	BaseEvent
	OrderID      string
	FilledQty    decimal.Decimal
	AvgFillPrice decimal.Decimal
	FilledAtUTC  time.Time
}

// OrderCanceledEvent is emitted when an order is canceled.
type OrderCanceledEvent struct {
	BaseEvent
	OrderID       string
	CanceledAtUTC time.Time
	Reason        string
}

// OrderRejectedEvent is emitted when an order is rejected by the exchange.
type OrderRejectedEvent struct {
	BaseEvent
	OrderID       string
	RejectedAtUTC time.Time
	Reason        string
}

// OrderTimeoutEvent is emitted when an order times out without confirmation.
type OrderTimeoutEvent struct {
	BaseEvent
	OrderID       string
	ClientOrderID string
	TimeoutAt     time.Time
}

// PositionOpenedEvent is emitted when a new position is opened.
type PositionOpenedEvent struct {
	BaseEvent
	PositionID  string
	StrategyID  string
	Symbol      string
	Side        string
	Quantity    decimal.Decimal
	EntryPrice  decimal.Decimal
	OpenedAtUTC time.Time
}

// PositionUpdatedEvent is emitted when a position is modified.
type PositionUpdatedEvent struct {
	BaseEvent
	PositionID    string
	Quantity      decimal.Decimal
	CurrentPrice  decimal.Decimal
	UnrealizedPnL decimal.Decimal
	UpdatedAtUTC  time.Time
}

// PositionClosedEvent is emitted when a position is fully closed.
type PositionClosedEvent struct {
	BaseEvent
	PositionID  string
	StrategyID  string
	Symbol      string
	RealizedPnL decimal.Decimal
	ClosedAtUTC time.Time
}

// RiskViolationEvent is emitted when a risk limit is breached.
type RiskViolationEvent struct {
	BaseEvent
	StrategyID    string
	ViolationType string
	CurrentValue  decimal.Decimal
	LimitValue    decimal.Decimal
	Action        string
}

// RiskWarningEvent is emitted when approaching a risk limit.
type RiskWarningEvent struct {
	BaseEvent
	StrategyID   string
	WarningType  string
	CurrentValue decimal.Decimal
	LimitValue   decimal.Decimal
	ThresholdPct decimal.Decimal
}

// BalanceUpdatedEvent is emitted when account balance changes.
type BalanceUpdatedEvent struct {
	BaseEvent
	Asset        string
	Total        decimal.Decimal
	Available    decimal.Decimal
	Locked       decimal.Decimal
	UpdatedAtUTC time.Time
}

// StrategyStartedEvent is emitted when a strategy begins execution.
type StrategyStartedEvent struct {
	BaseEvent
	StrategyID   string
	StrategyName string
	StrategyType string
	StartedAtUTC time.Time
}

// StrategyPausedEvent is emitted when a strategy is paused.
type StrategyPausedEvent struct {
	BaseEvent
	StrategyID  string
	PausedAtUTC time.Time
	Reason      string
}

// StrategyStoppedEvent is emitted when a strategy is stopped.
type StrategyStoppedEvent struct {
	BaseEvent
	StrategyID   string
	StoppedAtUTC time.Time
	Reason       string
	FinalPnL     decimal.Decimal
}
