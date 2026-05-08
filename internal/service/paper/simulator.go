package paper

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kairos-development/kairos-agent/internal/domain/entity"
	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
)

// Simulator simulates paper trading without real exchange execution.
// It maintains virtual positions and simulates fills based on market data.
type Simulator struct {
	mu sync.RWMutex

	logger *logrus.Logger

	// Virtual state
	orders    map[string]*entity.Order
	positions map[string]*entity.Position
	balance   decimal.Decimal

	// Market data for fill simulation
	lastPrices map[string]decimal.Decimal

	// Configuration
	fillDelay      time.Duration
	slippageBps    decimal.Decimal
	commissionRate decimal.Decimal
}

// NewSimulator creates a new paper trading simulator.
func NewSimulator(initialBalance decimal.Decimal, logger *logrus.Logger) *Simulator {
	if logger == nil {
		logger = logrus.New()
	}

	return &Simulator{
		logger:         logger,
		orders:         make(map[string]*entity.Order),
		positions:      make(map[string]*entity.Position),
		balance:        initialBalance,
		lastPrices:     make(map[string]decimal.Decimal),
		fillDelay:      100 * time.Millisecond,
		slippageBps:    decimal.NewFromInt(5),        // 0.05% slippage
		commissionRate: decimal.NewFromFloat(0.0006), // 0.06% commission
	}
}

// SubmitOrder simulates order submission.
func (s *Simulator) SubmitOrder(ctx context.Context, order *entity.Order) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Generate exchange order ID
	exchangeOrderID := fmt.Sprintf("paper_%d", time.Now().UnixNano())

	// Clone order for simulation
	simOrder := *order
	simOrder.ExchangeOrderID = exchangeOrderID
	simOrder.Status = entity.OrderStatusSubmitted
	now := time.Now().UTC()
	simOrder.SubmittedAtUTC = &now

	s.orders[exchangeOrderID] = &simOrder

	s.logger.WithFields(logrus.Fields{
		"exchange_order_id": exchangeOrderID,
		"symbol":            order.Symbol,
		"side":              order.Side,
		"quantity":          order.Quantity.String(),
		"price":             order.Price.String(),
	}).Info("Paper order submitted")

	// Simulate fill after delay
	go s.simulateFill(ctx, exchangeOrderID)

	return exchangeOrderID, nil
}

// CancelOrder simulates order cancellation.
func (s *Simulator) CancelOrder(ctx context.Context, orderID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	order, exists := s.orders[orderID]
	if !exists {
		return fmt.Errorf("order not found: %s", orderID)
	}

	if order.IsTerminal() {
		return fmt.Errorf("order already in terminal state: %s", order.Status)
	}

	order.Status = entity.OrderStatusCanceled
	order.UpdatedAtUTC = time.Now().UTC()

	s.logger.WithField("exchange_order_id", orderID).Info("Paper order canceled")

	return nil
}

// GetOrderStatus returns the current status of an order.
func (s *Simulator) GetOrderStatus(ctx context.Context, orderID string) (*entity.Order, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	order, exists := s.orders[orderID]
	if !exists {
		return nil, fmt.Errorf("order not found: %s", orderID)
	}

	// Return a copy
	orderCopy := *order
	return &orderCopy, nil
}

// GetPosition returns the current position for a symbol.
func (s *Simulator) GetPosition(symbol string) (*entity.Position, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	position, exists := s.positions[symbol]
	if !exists {
		return nil, fmt.Errorf("position not found: %s", symbol)
	}

	// Return a copy
	posCopy := *position
	return &posCopy, nil
}

// GetBalance returns the current virtual balance.
func (s *Simulator) GetBalance() decimal.Decimal {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.balance
}

// UpdateMarketPrice updates the last known price for a symbol.
// This is used for fill simulation and position valuation.
func (s *Simulator) UpdateMarketPrice(symbol string, price decimal.Decimal) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.lastPrices[symbol] = price

	// Update unrealized PnL for open positions
	if position, exists := s.positions[symbol]; exists && !position.IsFlat() {
		position.CurrentPrice = price
		position.UnrealizedPnL = s.calculateUnrealizedPnL(position, price)
		position.UpdatedAtUTC = time.Now().UTC()
	}
}

// Reset resets the simulator state.
func (s *Simulator) Reset(initialBalance decimal.Decimal) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.orders = make(map[string]*entity.Order)
	s.positions = make(map[string]*entity.Position)
	s.balance = initialBalance
	s.lastPrices = make(map[string]decimal.Decimal)

	s.logger.WithField("initial_balance", initialBalance.String()).Info("Paper simulator reset")
}

func (s *Simulator) simulateFill(ctx context.Context, orderID string) {
	// Wait for fill delay
	select {
	case <-time.After(s.fillDelay):
	case <-ctx.Done():
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	order, exists := s.orders[orderID]
	if !exists || order.Status != entity.OrderStatusSubmitted {
		return
	}

	// Get market price
	marketPrice, hasPriceData := s.lastPrices[order.Symbol]
	if !hasPriceData {
		// Use order price if no market data
		marketPrice = order.Price
	}

	// Apply slippage
	fillPrice := s.applySlippage(order, marketPrice)

	// Calculate commission
	notional := order.Quantity.Mul(fillPrice)
	commission := notional.Mul(s.commissionRate)

	// Check balance
	requiredBalance := notional.Add(commission)
	if order.Side == entity.OrderSideBuy && s.balance.LessThan(requiredBalance) {
		order.Status = entity.OrderStatusRejected
		order.UpdatedAtUTC = time.Now().UTC()
		s.logger.WithField("order_id", orderID).Warn("Paper order rejected: insufficient balance")
		return
	}

	// Fill order
	order.Status = entity.OrderStatusFilled
	order.FilledQty = order.Quantity
	order.RemainingQty = decimal.Zero
	order.AvgFillPrice = fillPrice
	now := time.Now().UTC()
	order.FilledAtUTC = &now
	order.UpdatedAtUTC = now

	// Update balance
	if order.Side == entity.OrderSideBuy {
		s.balance = s.balance.Sub(requiredBalance)
	} else {
		s.balance = s.balance.Add(notional.Sub(commission))
	}

	// Update position
	s.updatePosition(order, fillPrice, commission)

	s.logger.WithFields(logrus.Fields{
		"order_id":   orderID,
		"fill_price": fillPrice.String(),
		"commission": commission.String(),
		"balance":    s.balance.String(),
	}).Info("Paper order filled")
}

func (s *Simulator) applySlippage(order *entity.Order, marketPrice decimal.Decimal) decimal.Decimal {
	slippageMultiplier := s.slippageBps.Div(decimal.NewFromInt(10000))

	if order.Side == entity.OrderSideBuy {
		// Buy orders get worse price (higher)
		return marketPrice.Mul(decimal.NewFromInt(1).Add(slippageMultiplier))
	}

	// Sell orders get worse price (lower)
	return marketPrice.Mul(decimal.NewFromInt(1).Sub(slippageMultiplier))
}

func (s *Simulator) updatePosition(order *entity.Order, fillPrice decimal.Decimal, commission decimal.Decimal) {
	position, exists := s.positions[order.Symbol]

	if !exists {
		// Create new position
		side := entity.PositionSideLong
		if order.Side == entity.OrderSideSell {
			side = entity.PositionSideShort
		}

		position = &entity.Position{
			ID:            fmt.Sprintf("pos_%s_%d", order.Symbol, time.Now().UnixNano()),
			StrategyID:    order.StrategyID,
			Symbol:        order.Symbol,
			Side:          side,
			Quantity:      order.Quantity,
			EntryPrice:    fillPrice,
			CurrentPrice:  fillPrice,
			UnrealizedPnL: decimal.Zero,
			RealizedPnL:   commission.Neg(), // Commission is realized loss
			OpenedAtUTC:   time.Now().UTC(),
			UpdatedAtUTC:  time.Now().UTC(),
		}

		s.positions[order.Symbol] = position
		return
	}

	// Update existing position
	if (order.Side == entity.OrderSideBuy && position.Side == entity.PositionSideLong) ||
		(order.Side == entity.OrderSideSell && position.Side == entity.PositionSideShort) {
		// Adding to position
		totalCost := position.Quantity.Mul(position.EntryPrice).Add(order.Quantity.Mul(fillPrice))
		totalQty := position.Quantity.Add(order.Quantity)
		position.EntryPrice = totalCost.Div(totalQty)
		position.Quantity = totalQty
		position.RealizedPnL = position.RealizedPnL.Sub(commission)
	} else {
		// Reducing or closing position
		if order.Quantity.GreaterThanOrEqual(position.Quantity) {
			// Closing position
			pnl := s.calculateRealizedPnL(position, fillPrice, order.Quantity)
			position.RealizedPnL = position.RealizedPnL.Add(pnl).Sub(commission)
			position.Quantity = decimal.Zero
			position.UnrealizedPnL = decimal.Zero
		} else {
			// Partial close
			pnl := s.calculateRealizedPnL(position, fillPrice, order.Quantity)
			position.RealizedPnL = position.RealizedPnL.Add(pnl).Sub(commission)
			position.Quantity = position.Quantity.Sub(order.Quantity)
		}
	}

	position.CurrentPrice = fillPrice
	position.UpdatedAtUTC = time.Now().UTC()
}

func (s *Simulator) calculateRealizedPnL(position *entity.Position, exitPrice decimal.Decimal, quantity decimal.Decimal) decimal.Decimal {
	if position.Side == entity.PositionSideLong {
		return exitPrice.Sub(position.EntryPrice).Mul(quantity)
	}
	return position.EntryPrice.Sub(exitPrice).Mul(quantity)
}

func (s *Simulator) calculateUnrealizedPnL(position *entity.Position, currentPrice decimal.Decimal) decimal.Decimal {
	if position.Side == entity.PositionSideLong {
		return currentPrice.Sub(position.EntryPrice).Mul(position.Quantity)
	}
	return position.EntryPrice.Sub(currentPrice).Mul(position.Quantity)
}
