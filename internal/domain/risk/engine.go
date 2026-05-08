package risk

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/kairos-development/kairos-agent/internal/domain/entity"
	"github.com/kairos-development/kairos-agent/internal/domain/events"
	"github.com/shopspring/decimal"
)

var (
	// ErrPositionLimitExceeded reports that the position size exceeds the configured limit.
	ErrPositionLimitExceeded = errors.New("position size exceeds limit")

	// ErrDailyLossLimitExceeded reports that the daily loss limit has been breached.
	ErrDailyLossLimitExceeded = errors.New("daily loss limit exceeded")

	// ErrInsufficientBalance reports that available balance is insufficient for the order.
	ErrInsufficientBalance = errors.New("insufficient available balance")

	// ErrSymbolNotTrading reports that the symbol is not available for trading.
	ErrSymbolNotTrading = errors.New("symbol is not available for trading")

	// ErrInvalidOrderQuantity reports that the order quantity violates symbol constraints.
	ErrInvalidOrderQuantity = errors.New("order quantity violates symbol constraints")

	// ErrInvalidOrderPrice reports that the order price violates symbol constraints.
	ErrInvalidOrderPrice = errors.New("order price violates symbol constraints")

	// ErrInvalidNotional reports that the order notional value is below minimum.
	ErrInvalidNotional = errors.New("order notional value below minimum")
)

// Engine enforces risk limits and validates orders before execution.
// The risk engine is mandatory and cannot be disabled per architecture baseline.
type Engine struct {
	mu        sync.RWMutex
	publisher *events.Publisher
}

// NewEngine creates a new risk engine.
func NewEngine(publisher *events.Publisher) *Engine {
	return &Engine{
		publisher: publisher,
	}
}

// ValidateOrder performs comprehensive pre-trade risk checks.
// All monetary calculations use shopspring/decimal to avoid float64 precision issues.
func (e *Engine) ValidateOrder(ctx context.Context, order *entity.Order, strategy *entity.Strategy, symbol *entity.Symbol, balance *entity.Balance, currentPosition *entity.Position) error {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Check symbol trading status
	if !symbol.IsTrading() {
		return ErrSymbolNotTrading
	}

	// Validate order quantity against symbol constraints
	if !symbol.ValidateQuantity(order.Quantity) {
		return ErrInvalidOrderQuantity
	}

	// Validate order price against symbol constraints (limit orders only)
	if order.Type == entity.OrderTypeLimit && !symbol.ValidatePrice(order.Price) {
		return ErrInvalidOrderPrice
	}

	// Validate notional value
	price := order.Price
	if order.Type == entity.OrderTypeMarket {
		// For market orders, use current market price from position or a conservative estimate
		if currentPosition != nil && !currentPosition.CurrentPrice.IsZero() {
			price = currentPosition.CurrentPrice
		} else {
			// Conservative estimate: use a high value to ensure notional check passes
			price = symbol.MaxPrice
		}
	}

	if !symbol.ValidateNotional(order.Quantity, price) {
		return ErrInvalidNotional
	}

	// Check strategy daily loss limit
	if strategy.HasReachedDailyLossLimit() {
		e.publishRiskViolation(ctx, strategy.ID, "daily_loss_limit", strategy.DailyPnL, strategy.MaxDailyLoss, "block_new_entries")
		return ErrDailyLossLimitExceeded
	}

	// Check position size limit
	newPositionSize := order.Quantity
	if currentPosition != nil && !currentPosition.IsFlat() {
		if (order.Side == entity.OrderSideBuy && currentPosition.Side == entity.PositionSideLong) ||
			(order.Side == entity.OrderSideSell && currentPosition.Side == entity.PositionSideShort) {
			newPositionSize = currentPosition.Quantity.Add(order.Quantity)
		}
	}

	if !strategy.MaxPositionSize.IsZero() && newPositionSize.GreaterThan(strategy.MaxPositionSize) {
		e.publishRiskViolation(ctx, strategy.ID, "position_size_limit", newPositionSize, strategy.MaxPositionSize, "reject_order")
		return ErrPositionLimitExceeded
	}

	// Check available balance
	requiredBalance := order.Quantity.Mul(price)
	if balance != nil && !balance.HasSufficientAvailable(requiredBalance) {
		return ErrInsufficientBalance
	}

	// Emit warning if approaching daily loss limit (80% threshold)
	if !strategy.MaxDailyLoss.IsZero() {
		threshold := strategy.MaxDailyLoss.Mul(decimal.NewFromInt(8)).Div(decimal.NewFromInt(10))
		if strategy.DailyPnL.LessThanOrEqual(threshold.Neg()) {
			e.publishRiskWarning(ctx, strategy.ID, "daily_loss_approaching", strategy.DailyPnL, strategy.MaxDailyLoss, decimal.NewFromInt(80))
		}
	}

	return nil
}

// ValidatePositionClose validates a position closing order.
func (e *Engine) ValidatePositionClose(ctx context.Context, order *entity.Order, position *entity.Position, symbol *entity.Symbol) error {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if position.IsFlat() {
		return errors.New("cannot close flat position")
	}

	// Ensure order side is opposite to position side
	if (position.Side == entity.PositionSideLong && order.Side != entity.OrderSideSell) ||
		(position.Side == entity.PositionSideShort && order.Side != entity.OrderSideBuy) {
		return errors.New("order side must be opposite to position side")
	}

	// Ensure order quantity does not exceed position quantity
	if order.Quantity.GreaterThan(position.Quantity) {
		return errors.New("order quantity exceeds position quantity")
	}

	// Validate symbol constraints
	if !symbol.ValidateQuantity(order.Quantity) {
		return ErrInvalidOrderQuantity
	}

	return nil
}

// ShouldForceClose determines if a position must be forcibly closed.
// This occurs when a symbol is delisted or other critical risk events.
func (e *Engine) ShouldForceClose(symbol *entity.Symbol) bool {
	return symbol.RequiresForcedClose()
}

// ShouldBlockNewEntries determines if new entries should be blocked.
// This occurs during maintenance, suspension, or after risk limit breaches.
func (e *Engine) ShouldBlockNewEntries(strategy *entity.Strategy, symbol *entity.Symbol) bool {
	return symbol.BlocksNewEntries() || strategy.HasReachedDailyLossLimit()
}

func (e *Engine) publishRiskViolation(ctx context.Context, strategyID, violationType string, currentValue, limitValue decimal.Decimal, action string) {
	if e.publisher == nil {
		return
	}

	event := &events.RiskViolationEvent{
		BaseEvent: events.BaseEvent{
			EventType:  events.EventTypeRiskViolation,
			OccurredAt: time.Now().UTC(),
		},
		StrategyID:    strategyID,
		ViolationType: violationType,
		CurrentValue:  currentValue,
		LimitValue:    limitValue,
		Action:        action,
	}

	e.publisher.PublishAsync(ctx, event)
}

func (e *Engine) publishRiskWarning(ctx context.Context, strategyID, warningType string, currentValue, limitValue, thresholdPct decimal.Decimal) {
	if e.publisher == nil {
		return
	}

	event := &events.RiskWarningEvent{
		BaseEvent: events.BaseEvent{
			EventType:  events.EventTypeRiskWarning,
			OccurredAt: time.Now().UTC(),
		},
		StrategyID:   strategyID,
		WarningType:  warningType,
		CurrentValue: currentValue,
		LimitValue:   limitValue,
		ThresholdPct: thresholdPct,
	}

	e.publisher.PublishAsync(ctx, event)
}
