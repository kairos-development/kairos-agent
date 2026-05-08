package entity

import (
	"time"

	"github.com/shopspring/decimal"
)

// OrderSide identifies the order direction.
type OrderSide string

const (
	OrderSideBuy  OrderSide = "buy"
	OrderSideSell OrderSide = "sell"
)

// OrderType identifies the order execution type.
type OrderType string

const (
	OrderTypeLimit  OrderType = "limit"
	OrderTypeMarket OrderType = "market"
)

// OrderStatus identifies the current order lifecycle state.
type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "pending"
	OrderStatusSubmitted OrderStatus = "submitted"
	OrderStatusPartial   OrderStatus = "partial"
	OrderStatusFilled    OrderStatus = "filled"
	OrderStatusCanceled  OrderStatus = "canceled"
	OrderStatusRejected  OrderStatus = "rejected"
)

// TimeInForce identifies order validity constraints.
type TimeInForce string

const (
	TimeInForceGTC TimeInForce = "GTC" // Good Till Cancel
	TimeInForceIOC TimeInForce = "IOC" // Immediate Or Cancel
	TimeInForceFOK TimeInForce = "FOK" // Fill Or Kill
)

// Order represents a trading order in the domain layer.
// All monetary values use shopspring/decimal to avoid float64 precision issues.
type Order struct {
	ID              string
	ClientOrderID   string
	ExchangeOrderID string
	StrategyID      string
	Symbol          string
	Side            OrderSide
	Type            OrderType
	Status          OrderStatus
	TimeInForce     TimeInForce
	Quantity        decimal.Decimal
	Price           decimal.Decimal
	FilledQty       decimal.Decimal
	RemainingQty    decimal.Decimal
	AvgFillPrice    decimal.Decimal
	CreatedAtUTC    time.Time
	UpdatedAtUTC    time.Time
	SubmittedAtUTC  *time.Time
	FilledAtUTC     *time.Time
}

// IsTerminal returns true if the order has reached a final state.
func (o *Order) IsTerminal() bool {
	return o.Status == OrderStatusFilled ||
		o.Status == OrderStatusCanceled ||
		o.Status == OrderStatusRejected
}

// IsPartiallyFilled returns true if the order has partial fills.
func (o *Order) IsPartiallyFilled() bool {
	return o.Status == OrderStatusPartial && o.FilledQty.GreaterThan(decimal.Zero)
}

// IsPending returns true if the order is awaiting submission.
func (o *Order) IsPending() bool {
	return o.Status == OrderStatusPending
}

// IsActive returns true if the order is live on the exchange.
func (o *Order) IsActive() bool {
	return o.Status == OrderStatusSubmitted || o.Status == OrderStatusPartial
}
