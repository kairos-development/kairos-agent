package paper

import (
	"context"
	"testing"
	"time"

	"github.com/kairos-development/kairos-agent/internal/domain/entity"
	"github.com/shopspring/decimal"
)

func TestNewSimulator(t *testing.T) {
	initialBalance := decimal.NewFromInt(10000)
	sim := NewSimulator(initialBalance, nil)

	if sim == nil {
		t.Fatal("expected simulator to be created")
	}

	if !sim.balance.Equal(initialBalance) {
		t.Errorf("expected balance %s, got %s", initialBalance.String(), sim.balance.String())
	}

	if len(sim.orders) != 0 {
		t.Errorf("expected empty orders map, got %d", len(sim.orders))
	}

	if len(sim.positions) != 0 {
		t.Errorf("expected empty positions map, got %d", len(sim.positions))
	}
}

func TestSimulator_SubmitOrder(t *testing.T) {
	initialBalance := decimal.NewFromInt(10000)
	sim := NewSimulator(initialBalance, nil)

	order := &entity.Order{
		ID:         "order_1",
		StrategyID: "strategy_1",
		Symbol:     "BTCUSDT",
		Side:       entity.OrderSideBuy,
		Type:       entity.OrderTypeLimit,
		Quantity:   decimal.NewFromFloat(0.1),
		Price:      decimal.NewFromInt(50000),
	}

	ctx := context.Background()
	exchangeOrderID, err := sim.SubmitOrder(ctx, order)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if exchangeOrderID == "" {
		t.Error("expected exchange order ID to be generated")
	}

	if len(sim.orders) != 1 {
		t.Errorf("expected 1 order in simulator, got %d", len(sim.orders))
	}

	storedOrder, exists := sim.orders[exchangeOrderID]
	if !exists {
		t.Fatal("expected order to be stored")
	}

	if storedOrder.Status != entity.OrderStatusSubmitted {
		t.Errorf("expected status submitted, got %s", storedOrder.Status)
	}
}

func TestSimulator_SubmitOrder_Fill(t *testing.T) {
	initialBalance := decimal.NewFromInt(10000)
	sim := NewSimulator(initialBalance, nil)
	sim.fillDelay = 50 * time.Millisecond

	// Set market price
	sim.UpdateMarketPrice("BTCUSDT", decimal.NewFromInt(50000))

	order := &entity.Order{
		ID:         "order_1",
		StrategyID: "strategy_1",
		Symbol:     "BTCUSDT",
		Side:       entity.OrderSideBuy,
		Type:       entity.OrderTypeLimit,
		Quantity:   decimal.NewFromFloat(0.1),
		Price:      decimal.NewFromInt(50000),
	}

	ctx := context.Background()
	exchangeOrderID, err := sim.SubmitOrder(ctx, order)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Wait for fill
	time.Sleep(200 * time.Millisecond)

	storedOrder, err := sim.GetOrderStatus(ctx, exchangeOrderID)
	if err != nil {
		t.Fatalf("expected order to be stored, got error: %v", err)
	}

	if storedOrder.Status != entity.OrderStatusFilled {
		t.Errorf("expected status filled, got %s", storedOrder.Status)
	}

	if !storedOrder.FilledQty.Equal(order.Quantity) {
		t.Errorf("expected filled qty %s, got %s", order.Quantity.String(), storedOrder.FilledQty.String())
	}

	// Check position was created
	position, err := sim.GetPosition("BTCUSDT")
	if err != nil {
		t.Fatalf("expected position to be created, got error: %v", err)
	}

	if position.Side != entity.PositionSideLong {
		t.Errorf("expected long position, got %s", position.Side)
	}

	if !position.Quantity.Equal(order.Quantity) {
		t.Errorf("expected position quantity %s, got %s", order.Quantity.String(), position.Quantity.String())
	}
}

func TestSimulator_SubmitOrder_InsufficientBalance(t *testing.T) {
	initialBalance := decimal.NewFromInt(100)
	sim := NewSimulator(initialBalance, nil)
	sim.fillDelay = 50 * time.Millisecond

	// Set market price
	sim.UpdateMarketPrice("BTCUSDT", decimal.NewFromInt(50000))

	order := &entity.Order{
		ID:         "order_1",
		StrategyID: "strategy_1",
		Symbol:     "BTCUSDT",
		Side:       entity.OrderSideBuy,
		Type:       entity.OrderTypeLimit,
		Quantity:   decimal.NewFromFloat(0.1),
		Price:      decimal.NewFromInt(50000),
	}

	ctx := context.Background()
	exchangeOrderID, err := sim.SubmitOrder(ctx, order)
	if err != nil {
		t.Fatalf("expected no error on submit, got %v", err)
	}

	// Wait for fill attempt
	time.Sleep(200 * time.Millisecond)

	storedOrder, err := sim.GetOrderStatus(ctx, exchangeOrderID)
	if err != nil {
		t.Fatalf("expected order to be stored, got error: %v", err)
	}

	if storedOrder.Status != entity.OrderStatusRejected {
		t.Errorf("expected status rejected, got %s", storedOrder.Status)
	}
}

func TestSimulator_CancelOrder(t *testing.T) {
	initialBalance := decimal.NewFromInt(10000)
	sim := NewSimulator(initialBalance, nil)

	order := &entity.Order{
		ID:         "order_1",
		StrategyID: "strategy_1",
		Symbol:     "BTCUSDT",
		Side:       entity.OrderSideBuy,
		Type:       entity.OrderTypeLimit,
		Quantity:   decimal.NewFromFloat(0.1),
		Price:      decimal.NewFromInt(50000),
	}

	ctx := context.Background()
	exchangeOrderID, err := sim.SubmitOrder(ctx, order)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Cancel order
	err = sim.CancelOrder(ctx, exchangeOrderID)
	if err != nil {
		t.Fatalf("expected no error on cancel, got %v", err)
	}

	storedOrder, exists := sim.orders[exchangeOrderID]
	if !exists {
		t.Fatal("expected order to be stored")
	}

	if storedOrder.Status != entity.OrderStatusCanceled {
		t.Errorf("expected status canceled, got %s", storedOrder.Status)
	}
}

func TestSimulator_CancelOrder_NotFound(t *testing.T) {
	initialBalance := decimal.NewFromInt(10000)
	sim := NewSimulator(initialBalance, nil)

	ctx := context.Background()
	err := sim.CancelOrder(ctx, "nonexistent_order")

	if err == nil {
		t.Fatal("expected error when canceling nonexistent order")
	}
}

func TestSimulator_CancelOrder_AlreadyTerminal(t *testing.T) {
	initialBalance := decimal.NewFromInt(10000)
	sim := NewSimulator(initialBalance, nil)
	sim.fillDelay = 50 * time.Millisecond

	sim.UpdateMarketPrice("BTCUSDT", decimal.NewFromInt(50000))

	order := &entity.Order{
		ID:         "order_1",
		StrategyID: "strategy_1",
		Symbol:     "BTCUSDT",
		Side:       entity.OrderSideBuy,
		Type:       entity.OrderTypeLimit,
		Quantity:   decimal.NewFromFloat(0.1),
		Price:      decimal.NewFromInt(50000),
	}

	ctx := context.Background()
	exchangeOrderID, err := sim.SubmitOrder(ctx, order)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Wait for fill
	time.Sleep(200 * time.Millisecond)

	// Try to cancel filled order
	err = sim.CancelOrder(ctx, exchangeOrderID)
	if err == nil {
		t.Fatal("expected error when canceling filled order")
	}
}

func TestSimulator_GetOrderStatus(t *testing.T) {
	initialBalance := decimal.NewFromInt(10000)
	sim := NewSimulator(initialBalance, nil)

	order := &entity.Order{
		ID:         "order_1",
		StrategyID: "strategy_1",
		Symbol:     "BTCUSDT",
		Side:       entity.OrderSideBuy,
		Type:       entity.OrderTypeLimit,
		Quantity:   decimal.NewFromFloat(0.1),
		Price:      decimal.NewFromInt(50000),
	}

	ctx := context.Background()
	exchangeOrderID, err := sim.SubmitOrder(ctx, order)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Get order status
	retrievedOrder, err := sim.GetOrderStatus(ctx, exchangeOrderID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if retrievedOrder.ExchangeOrderID != exchangeOrderID {
		t.Errorf("expected order ID %s, got %s", exchangeOrderID, retrievedOrder.ExchangeOrderID)
	}
}

func TestSimulator_GetOrderStatus_NotFound(t *testing.T) {
	initialBalance := decimal.NewFromInt(10000)
	sim := NewSimulator(initialBalance, nil)

	ctx := context.Background()
	_, err := sim.GetOrderStatus(ctx, "nonexistent_order")

	if err == nil {
		t.Fatal("expected error when getting nonexistent order")
	}
}

func TestSimulator_GetPosition(t *testing.T) {
	initialBalance := decimal.NewFromInt(10000)
	sim := NewSimulator(initialBalance, nil)
	sim.fillDelay = 50 * time.Millisecond

	sim.UpdateMarketPrice("BTCUSDT", decimal.NewFromInt(50000))

	order := &entity.Order{
		ID:         "order_1",
		StrategyID: "strategy_1",
		Symbol:     "BTCUSDT",
		Side:       entity.OrderSideBuy,
		Type:       entity.OrderTypeLimit,
		Quantity:   decimal.NewFromFloat(0.1),
		Price:      decimal.NewFromInt(50000),
	}

	ctx := context.Background()
	_, err := sim.SubmitOrder(ctx, order)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Wait for fill
	time.Sleep(200 * time.Millisecond)

	// Get position
	position, err := sim.GetPosition("BTCUSDT")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if position.Symbol != "BTCUSDT" {
		t.Errorf("expected symbol BTCUSDT, got %s", position.Symbol)
	}

	if !position.Quantity.Equal(order.Quantity) {
		t.Errorf("expected quantity %s, got %s", order.Quantity.String(), position.Quantity.String())
	}
}

func TestSimulator_GetPosition_NotFound(t *testing.T) {
	initialBalance := decimal.NewFromInt(10000)
	sim := NewSimulator(initialBalance, nil)

	_, err := sim.GetPosition("BTCUSDT")

	if err == nil {
		t.Fatal("expected error when getting nonexistent position")
	}
}

func TestSimulator_GetBalance(t *testing.T) {
	initialBalance := decimal.NewFromInt(10000)
	sim := NewSimulator(initialBalance, nil)

	balance := sim.GetBalance()

	if !balance.Equal(initialBalance) {
		t.Errorf("expected balance %s, got %s", initialBalance.String(), balance.String())
	}
}

func TestSimulator_UpdateMarketPrice(t *testing.T) {
	initialBalance := decimal.NewFromInt(10000)
	sim := NewSimulator(initialBalance, nil)
	sim.fillDelay = 50 * time.Millisecond

	sim.UpdateMarketPrice("BTCUSDT", decimal.NewFromInt(50000))

	// Create position
	order := &entity.Order{
		ID:         "order_1",
		StrategyID: "strategy_1",
		Symbol:     "BTCUSDT",
		Side:       entity.OrderSideBuy,
		Type:       entity.OrderTypeLimit,
		Quantity:   decimal.NewFromFloat(0.1),
		Price:      decimal.NewFromInt(50000),
	}

	ctx := context.Background()
	_, err := sim.SubmitOrder(ctx, order)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	// Update market price
	newPrice := decimal.NewFromInt(51000)
	sim.UpdateMarketPrice("BTCUSDT", newPrice)

	position, err := sim.GetPosition("BTCUSDT")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !position.CurrentPrice.Equal(newPrice) {
		t.Errorf("expected current price %s, got %s", newPrice.String(), position.CurrentPrice.String())
	}

	// Check unrealized PnL was updated
	expectedPnL := newPrice.Sub(position.EntryPrice).Mul(position.Quantity)
	if position.UnrealizedPnL.LessThan(decimal.Zero) {
		t.Errorf("expected positive unrealized PnL, got %s", position.UnrealizedPnL.String())
	}

	// Allow for commission impact
	if position.UnrealizedPnL.GreaterThan(expectedPnL) {
		t.Errorf("expected unrealized PnL around %s, got %s", expectedPnL.String(), position.UnrealizedPnL.String())
	}
}

func TestSimulator_Reset(t *testing.T) {
	initialBalance := decimal.NewFromInt(10000)
	sim := NewSimulator(initialBalance, nil)

	// Add some state
	order := &entity.Order{
		ID:         "order_1",
		StrategyID: "strategy_1",
		Symbol:     "BTCUSDT",
		Side:       entity.OrderSideBuy,
		Type:       entity.OrderTypeLimit,
		Quantity:   decimal.NewFromFloat(0.1),
		Price:      decimal.NewFromInt(50000),
	}

	ctx := context.Background()
	_, _ = sim.SubmitOrder(ctx, order)

	sim.UpdateMarketPrice("BTCUSDT", decimal.NewFromInt(50000))

	// Reset
	newBalance := decimal.NewFromInt(20000)
	sim.Reset(newBalance)

	if !sim.balance.Equal(newBalance) {
		t.Errorf("expected balance %s after reset, got %s", newBalance.String(), sim.balance.String())
	}

	if len(sim.orders) != 0 {
		t.Errorf("expected empty orders after reset, got %d", len(sim.orders))
	}

	if len(sim.positions) != 0 {
		t.Errorf("expected empty positions after reset, got %d", len(sim.positions))
	}

	if len(sim.lastPrices) != 0 {
		t.Errorf("expected empty prices after reset, got %d", len(sim.lastPrices))
	}
}

func TestSimulator_ClosePosition(t *testing.T) {
	initialBalance := decimal.NewFromInt(10000)
	sim := NewSimulator(initialBalance, nil)
	sim.fillDelay = 50 * time.Millisecond

	sim.UpdateMarketPrice("BTCUSDT", decimal.NewFromInt(50000))

	// Open position
	buyOrder := &entity.Order{
		ID:         "order_1",
		StrategyID: "strategy_1",
		Symbol:     "BTCUSDT",
		Side:       entity.OrderSideBuy,
		Type:       entity.OrderTypeLimit,
		Quantity:   decimal.NewFromFloat(0.1),
		Price:      decimal.NewFromInt(50000),
	}

	ctx := context.Background()
	_, err := sim.SubmitOrder(ctx, buyOrder)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	// Update price
	sim.UpdateMarketPrice("BTCUSDT", decimal.NewFromInt(51000))

	// Close position
	sellOrder := &entity.Order{
		ID:         "order_2",
		StrategyID: "strategy_1",
		Symbol:     "BTCUSDT",
		Side:       entity.OrderSideSell,
		Type:       entity.OrderTypeLimit,
		Quantity:   decimal.NewFromFloat(0.1),
		Price:      decimal.NewFromInt(51000),
	}

	_, err = sim.SubmitOrder(ctx, sellOrder)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	// Check position is closed
	position, err := sim.GetPosition("BTCUSDT")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !position.Quantity.IsZero() {
		t.Errorf("expected position to be closed, got quantity %s", position.Quantity.String())
	}

	// Check realized PnL is positive (profit from price increase)
	if position.RealizedPnL.LessThanOrEqual(decimal.Zero) {
		t.Errorf("expected positive realized PnL, got %s", position.RealizedPnL.String())
	}
}
