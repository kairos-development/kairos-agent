package router

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/kairos-development/kairos-agent/internal/domain/entity"
	"github.com/kairos-development/kairos-agent/internal/domain/events"
	"github.com/shopspring/decimal"
)

// Mock implementations for testing

type mockExchangeConnector struct {
	submitOrderFunc      func(ctx context.Context, order *entity.Order) (string, error)
	cancelOrderFunc      func(ctx context.Context, exchangeOrderID string) error
	queryOrderStatusFunc func(ctx context.Context, exchangeOrderID string) (*entity.Order, error)
}

func (m *mockExchangeConnector) SubmitOrder(ctx context.Context, order *entity.Order) (string, error) {
	if m.submitOrderFunc != nil {
		return m.submitOrderFunc(ctx, order)
	}
	return "exchange_order_123", nil
}

func (m *mockExchangeConnector) CancelOrder(ctx context.Context, exchangeOrderID string) error {
	if m.cancelOrderFunc != nil {
		return m.cancelOrderFunc(ctx, exchangeOrderID)
	}
	return nil
}

func (m *mockExchangeConnector) QueryOrderStatus(ctx context.Context, exchangeOrderID string) (*entity.Order, error) {
	if m.queryOrderStatusFunc != nil {
		return m.queryOrderStatusFunc(ctx, exchangeOrderID)
	}
	return &entity.Order{
		ExchangeOrderID: exchangeOrderID,
		Status:          entity.OrderStatusFilled,
	}, nil
}

type mockPaperSimulator struct {
	submitOrderFunc    func(ctx context.Context, order *entity.Order) (string, error)
	cancelOrderFunc    func(ctx context.Context, orderID string) error
	getOrderStatusFunc func(ctx context.Context, orderID string) (*entity.Order, error)
}

func (m *mockPaperSimulator) SubmitOrder(ctx context.Context, order *entity.Order) (string, error) {
	if m.submitOrderFunc != nil {
		return m.submitOrderFunc(ctx, order)
	}
	return "paper_order_123", nil
}

func (m *mockPaperSimulator) CancelOrder(ctx context.Context, orderID string) error {
	if m.cancelOrderFunc != nil {
		return m.cancelOrderFunc(ctx, orderID)
	}
	return nil
}

func (m *mockPaperSimulator) GetOrderStatus(ctx context.Context, orderID string) (*entity.Order, error) {
	if m.getOrderStatusFunc != nil {
		return m.getOrderStatusFunc(ctx, orderID)
	}
	return &entity.Order{
		ExchangeOrderID: orderID,
		Status:          entity.OrderStatusFilled,
	}, nil
}

type mockTradingGate struct {
	err error
}

func (m *mockTradingGate) CheckNewEntry(ctx context.Context) error {
	return m.err
}

func TestNewOrderRouter(t *testing.T) {
	config := entity.DefaultRouterConfig()
	connector := &mockExchangeConnector{}
	simulator := &mockPaperSimulator{}
	publisher := events.NewPublisher()

	router := NewOrderRouter(config, connector, simulator, publisher, nil)

	if router == nil {
		t.Fatal("expected router to be created")
	}

	if router.config.Mode != entity.RoutingModePaper {
		t.Errorf("expected default mode to be paper, got %s", router.config.Mode)
	}

	if len(router.inFlight) != 0 {
		t.Errorf("expected empty in-flight map, got %d", len(router.inFlight))
	}
}

func TestOrderRouter_SubmitOrder_BlockedByTradingGate(t *testing.T) {
	config := entity.DefaultRouterConfig()
	config.Mode = entity.RoutingModeLive

	submitted := false
	connector := &mockExchangeConnector{
		submitOrderFunc: func(ctx context.Context, order *entity.Order) (string, error) {
			submitted = true
			return "exchange_order_123", nil
		},
	}
	router := NewOrderRouter(config, connector, &mockPaperSimulator{}, events.NewPublisher(), nil)
	router.SetTradingGate(&mockTradingGate{err: fmt.Errorf("blocked")})

	order := &entity.Order{
		ID:           "order_1",
		StrategyID:   "strategy_1",
		Symbol:       "BTCUSDT",
		Side:         entity.OrderSideBuy,
		Type:         entity.OrderTypeLimit,
		Quantity:     decimal.NewFromFloat(0.1),
		Price:        decimal.NewFromInt(50000),
		CreatedAtUTC: time.Now().UTC(),
	}

	_, err := router.SubmitOrder(context.Background(), order)

	if err == nil {
		t.Fatal("expected trading gate error")
	}
	if submitted {
		t.Fatal("expected connector submit not to be called")
	}
	if router.GetInFlightCount() != 0 {
		t.Fatalf("expected no in-flight order after gate rejection, got %d", router.GetInFlightCount())
	}
}

func TestOrderRouter_SubmitOrder_LiveMode(t *testing.T) {
	config := entity.DefaultRouterConfig()
	config.Mode = entity.RoutingModeLive

	exchangeOrderID := "live_order_456"
	connector := &mockExchangeConnector{
		submitOrderFunc: func(ctx context.Context, order *entity.Order) (string, error) {
			return exchangeOrderID, nil
		},
	}
	simulator := &mockPaperSimulator{}
	publisher := events.NewPublisher()

	router := NewOrderRouter(config, connector, simulator, publisher, nil)

	order := &entity.Order{
		ID:           "order_1",
		StrategyID:   "strategy_1",
		Symbol:       "BTCUSDT",
		Side:         entity.OrderSideBuy,
		Type:         entity.OrderTypeLimit,
		Quantity:     decimal.NewFromFloat(0.1),
		Price:        decimal.NewFromInt(50000),
		CreatedAtUTC: time.Now().UTC(),
	}

	ctx := context.Background()
	resultOrderID, err := router.SubmitOrder(ctx, order)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if resultOrderID != exchangeOrderID {
		t.Errorf("expected order ID %s, got %s", exchangeOrderID, resultOrderID)
	}

	if order.ClientOrderID == "" {
		t.Error("expected ClientOrderID to be generated")
	}

	if router.GetInFlightCount() != 1 {
		t.Errorf("expected 1 in-flight order, got %d", router.GetInFlightCount())
	}
}

func TestOrderRouter_SubmitOrder_PaperMode(t *testing.T) {
	config := entity.DefaultRouterConfig()
	config.Mode = entity.RoutingModePaper

	paperOrderID := "paper_order_789"
	connector := &mockExchangeConnector{}
	simulator := &mockPaperSimulator{
		submitOrderFunc: func(ctx context.Context, order *entity.Order) (string, error) {
			return paperOrderID, nil
		},
	}
	publisher := events.NewPublisher()

	router := NewOrderRouter(config, connector, simulator, publisher, nil)

	order := &entity.Order{
		ID:           "order_2",
		StrategyID:   "strategy_1",
		Symbol:       "ETHUSDT",
		Side:         entity.OrderSideSell,
		Type:         entity.OrderTypeMarket,
		Quantity:     decimal.NewFromFloat(1.0),
		CreatedAtUTC: time.Now().UTC(),
	}

	ctx := context.Background()
	resultOrderID, err := router.SubmitOrder(ctx, order)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if resultOrderID != paperOrderID {
		t.Errorf("expected order ID %s, got %s", paperOrderID, resultOrderID)
	}

	if router.GetInFlightCount() != 1 {
		t.Errorf("expected 1 in-flight order, got %d", router.GetInFlightCount())
	}
}

func TestOrderRouter_SubmitOrder_Idempotency(t *testing.T) {
	config := entity.DefaultRouterConfig()
	config.EnableIdempotency = true

	callCount := 0
	connector := &mockExchangeConnector{}
	simulator := &mockPaperSimulator{
		submitOrderFunc: func(ctx context.Context, order *entity.Order) (string, error) {
			callCount++
			return "paper_order_idempotent", nil
		},
	}
	publisher := events.NewPublisher()

	router := NewOrderRouter(config, connector, simulator, publisher, nil)

	order := &entity.Order{
		ID:            "order_3",
		ClientOrderID: "client_order_fixed",
		StrategyID:    "strategy_1",
		Symbol:        "BTCUSDT",
		Side:          entity.OrderSideBuy,
		Type:          entity.OrderTypeLimit,
		Quantity:      decimal.NewFromFloat(0.1),
		Price:         decimal.NewFromInt(50000),
		CreatedAtUTC:  time.Now().UTC(),
	}

	ctx := context.Background()

	// First submission
	orderID1, err := router.SubmitOrder(ctx, order)
	if err != nil {
		t.Fatalf("expected no error on first submit, got %v", err)
	}

	// Second submission with same ClientOrderID
	orderID2, err := router.SubmitOrder(ctx, order)
	if err != nil {
		t.Fatalf("expected no error on second submit, got %v", err)
	}

	if orderID1 != orderID2 {
		t.Errorf("expected same order ID for idempotent requests, got %s and %s", orderID1, orderID2)
	}

	if callCount != 1 {
		t.Errorf("expected submitOrder to be called once, got %d", callCount)
	}

	if router.GetInFlightCount() != 1 {
		t.Errorf("expected 1 in-flight order, got %d", router.GetInFlightCount())
	}
}

func TestOrderRouter_SubmitOrder_MaxInFlightLimit(t *testing.T) {
	config := entity.DefaultRouterConfig()
	config.MaxInFlightOrders = 2

	connector := &mockExchangeConnector{}
	simulator := &mockPaperSimulator{}
	publisher := events.NewPublisher()

	router := NewOrderRouter(config, connector, simulator, publisher, nil)

	ctx := context.Background()

	// Submit first order
	order1 := &entity.Order{
		ID:           "order_1",
		StrategyID:   "strategy_1",
		Symbol:       "BTCUSDT",
		Side:         entity.OrderSideBuy,
		Quantity:     decimal.NewFromFloat(0.1),
		CreatedAtUTC: time.Now().UTC(),
	}
	_, err := router.SubmitOrder(ctx, order1)
	if err != nil {
		t.Fatalf("expected no error on first order, got %v", err)
	}

	// Submit second order
	order2 := &entity.Order{
		ID:           "order_2",
		StrategyID:   "strategy_1",
		Symbol:       "ETHUSDT",
		Side:         entity.OrderSideBuy,
		Quantity:     decimal.NewFromFloat(1.0),
		CreatedAtUTC: time.Now().UTC(),
	}
	_, err = router.SubmitOrder(ctx, order2)
	if err != nil {
		t.Fatalf("expected no error on second order, got %v", err)
	}

	// Submit third order (should fail)
	order3 := &entity.Order{
		ID:           "order_3",
		StrategyID:   "strategy_1",
		Symbol:       "SOLUSDT",
		Side:         entity.OrderSideBuy,
		Quantity:     decimal.NewFromFloat(10.0),
		CreatedAtUTC: time.Now().UTC(),
	}
	_, err = router.SubmitOrder(ctx, order3)
	if err == nil {
		t.Fatal("expected error when exceeding max in-flight orders")
	}
}

func TestOrderRouter_CancelOrder_LiveMode(t *testing.T) {
	config := entity.DefaultRouterConfig()
	config.Mode = entity.RoutingModeLive

	cancelCalled := false
	connector := &mockExchangeConnector{
		cancelOrderFunc: func(ctx context.Context, exchangeOrderID string) error {
			cancelCalled = true
			return nil
		},
	}
	simulator := &mockPaperSimulator{}
	publisher := events.NewPublisher()

	router := NewOrderRouter(config, connector, simulator, publisher, nil)

	ctx := context.Background()
	err := router.CancelOrder(ctx, "exchange_order_123")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !cancelCalled {
		t.Error("expected connector CancelOrder to be called")
	}
}

func TestOrderRouter_CancelOrder_PaperMode(t *testing.T) {
	config := entity.DefaultRouterConfig()
	config.Mode = entity.RoutingModePaper

	cancelCalled := false
	connector := &mockExchangeConnector{}
	simulator := &mockPaperSimulator{
		cancelOrderFunc: func(ctx context.Context, orderID string) error {
			cancelCalled = true
			return nil
		},
	}
	publisher := events.NewPublisher()

	router := NewOrderRouter(config, connector, simulator, publisher, nil)

	ctx := context.Background()
	err := router.CancelOrder(ctx, "paper_order_123")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !cancelCalled {
		t.Error("expected simulator CancelOrder to be called")
	}
}

func TestOrderRouter_QueryOrderStatus(t *testing.T) {
	config := entity.DefaultRouterConfig()
	config.Mode = entity.RoutingModeLive

	expectedOrder := &entity.Order{
		ExchangeOrderID: "exchange_order_123",
		Status:          entity.OrderStatusFilled,
		FilledQty:       decimal.NewFromFloat(0.1),
	}

	connector := &mockExchangeConnector{
		queryOrderStatusFunc: func(ctx context.Context, exchangeOrderID string) (*entity.Order, error) {
			return expectedOrder, nil
		},
	}
	simulator := &mockPaperSimulator{}
	publisher := events.NewPublisher()

	router := NewOrderRouter(config, connector, simulator, publisher, nil)

	ctx := context.Background()
	order, err := router.QueryOrderStatus(ctx, "exchange_order_123")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if order.ExchangeOrderID != expectedOrder.ExchangeOrderID {
		t.Errorf("expected order ID %s, got %s", expectedOrder.ExchangeOrderID, order.ExchangeOrderID)
	}

	if order.Status != expectedOrder.Status {
		t.Errorf("expected status %s, got %s", expectedOrder.Status, order.Status)
	}
}

func TestOrderRouter_ConfirmOrder(t *testing.T) {
	config := entity.DefaultRouterConfig()
	connector := &mockExchangeConnector{}
	simulator := &mockPaperSimulator{}
	publisher := events.NewPublisher()

	router := NewOrderRouter(config, connector, simulator, publisher, nil)

	// Add order to in-flight
	order := &entity.Order{
		ID:            "order_1",
		ClientOrderID: "client_order_1",
		StrategyID:    "strategy_1",
		Symbol:        "BTCUSDT",
		Side:          entity.OrderSideBuy,
		Quantity:      decimal.NewFromFloat(0.1),
		CreatedAtUTC:  time.Now().UTC(),
	}

	ctx := context.Background()
	_, err := router.SubmitOrder(ctx, order)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if router.GetInFlightCount() != 1 {
		t.Errorf("expected 1 in-flight order, got %d", router.GetInFlightCount())
	}

	// Confirm order
	router.ConfirmOrder(order.ClientOrderID)

	if router.GetInFlightCount() != 0 {
		t.Errorf("expected 0 in-flight orders after confirm, got %d", router.GetInFlightCount())
	}
}

func TestOrderRouter_CleanupTimedOutOrders(t *testing.T) {
	config := entity.DefaultRouterConfig()
	config.TimeoutSeconds = 1 // 1 second timeout

	connector := &mockExchangeConnector{}
	simulator := &mockPaperSimulator{}
	publisher := events.NewPublisher()

	router := NewOrderRouter(config, connector, simulator, publisher, nil)

	// Add order to in-flight
	order := &entity.Order{
		ID:           "order_1",
		StrategyID:   "strategy_1",
		Symbol:       "BTCUSDT",
		Side:         entity.OrderSideBuy,
		Quantity:     decimal.NewFromFloat(0.1),
		CreatedAtUTC: time.Now().UTC(),
	}

	ctx := context.Background()
	_, err := router.SubmitOrder(ctx, order)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if router.GetInFlightCount() != 1 {
		t.Errorf("expected 1 in-flight order, got %d", router.GetInFlightCount())
	}

	// Wait for timeout
	time.Sleep(1500 * time.Millisecond)

	// Cleanup timed out orders
	cleaned := router.CleanupTimedOutOrders(ctx)

	if cleaned != 1 {
		t.Errorf("expected 1 order to be cleaned, got %d", cleaned)
	}

	if router.GetInFlightCount() != 0 {
		t.Errorf("expected 0 in-flight orders after cleanup, got %d", router.GetInFlightCount())
	}
}

func TestOrderRouter_SetMode(t *testing.T) {
	config := entity.DefaultRouterConfig()
	config.Mode = entity.RoutingModePaper

	connector := &mockExchangeConnector{}
	simulator := &mockPaperSimulator{}
	publisher := events.NewPublisher()

	router := NewOrderRouter(config, connector, simulator, publisher, nil)

	if router.GetMode() != entity.RoutingModePaper {
		t.Errorf("expected initial mode to be paper, got %s", router.GetMode())
	}

	router.SetMode(entity.RoutingModeLive)

	if router.GetMode() != entity.RoutingModeLive {
		t.Errorf("expected mode to be live after SetMode, got %s", router.GetMode())
	}
}

func TestOrderRouter_GenerateClientOrderID_Deterministic(t *testing.T) {
	config := entity.DefaultRouterConfig()
	connector := &mockExchangeConnector{}
	simulator := &mockPaperSimulator{}
	publisher := events.NewPublisher()

	router := NewOrderRouter(config, connector, simulator, publisher, nil)

	order := &entity.Order{
		ID:           "order_1",
		StrategyID:   "strategy_1",
		Symbol:       "BTCUSDT",
		Side:         entity.OrderSideBuy,
		CreatedAtUTC: time.Date(2026, 5, 2, 0, 0, 0, 0, time.UTC),
	}

	clientOrderID1 := router.generateClientOrderID(order)
	clientOrderID2 := router.generateClientOrderID(order)

	if clientOrderID1 != clientOrderID2 {
		t.Errorf("expected deterministic client order ID, got %s and %s", clientOrderID1, clientOrderID2)
	}

	if len(clientOrderID1) != 64 {
		t.Errorf("expected SHA256 hash length of 64, got %d", len(clientOrderID1))
	}
}

func TestOrderRouter_SubmitOrder_ConnectorNil(t *testing.T) {
	config := entity.DefaultRouterConfig()
	config.Mode = entity.RoutingModeLive

	publisher := events.NewPublisher()
	router := NewOrderRouter(config, nil, nil, publisher, nil)

	order := &entity.Order{
		ID:           "order_1",
		StrategyID:   "strategy_1",
		Symbol:       "BTCUSDT",
		Side:         entity.OrderSideBuy,
		Quantity:     decimal.NewFromFloat(0.1),
		CreatedAtUTC: time.Now().UTC(),
	}

	ctx := context.Background()
	_, err := router.SubmitOrder(ctx, order)

	if err == nil {
		t.Fatal("expected error when connector is nil")
	}
}

func TestOrderRouter_SubmitOrder_SimulatorNil(t *testing.T) {
	config := entity.DefaultRouterConfig()
	config.Mode = entity.RoutingModePaper

	publisher := events.NewPublisher()
	router := NewOrderRouter(config, nil, nil, publisher, nil)

	order := &entity.Order{
		ID:           "order_1",
		StrategyID:   "strategy_1",
		Symbol:       "BTCUSDT",
		Side:         entity.OrderSideBuy,
		Quantity:     decimal.NewFromFloat(0.1),
		CreatedAtUTC: time.Now().UTC(),
	}

	ctx := context.Background()
	_, err := router.SubmitOrder(ctx, order)

	if err == nil {
		t.Fatal("expected error when simulator is nil")
	}
}

func TestOrderRouter_QueryOrderStatus_PaperMode(t *testing.T) {
	config := entity.DefaultRouterConfig()
	config.Mode = entity.RoutingModePaper

	expectedOrder := &entity.Order{
		ExchangeOrderID: "paper_order_123",
		Status:          entity.OrderStatusFilled,
		FilledQty:       decimal.NewFromFloat(0.1),
	}

	connector := &mockExchangeConnector{}
	simulator := &mockPaperSimulator{
		getOrderStatusFunc: func(ctx context.Context, orderID string) (*entity.Order, error) {
			return expectedOrder, nil
		},
	}
	publisher := events.NewPublisher()

	router := NewOrderRouter(config, connector, simulator, publisher, nil)

	ctx := context.Background()
	order, err := router.QueryOrderStatus(ctx, "paper_order_123")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if order.ExchangeOrderID != expectedOrder.ExchangeOrderID {
		t.Errorf("expected order ID %s, got %s", expectedOrder.ExchangeOrderID, order.ExchangeOrderID)
	}
}

func TestOrderRouter_QueryOrderStatus_WithFallback(t *testing.T) {
	config := entity.DefaultRouterConfig()
	config.Mode = entity.RoutingModeLive
	config.EnableStatusFallback = true

	connector := &mockExchangeConnector{
		queryOrderStatusFunc: func(ctx context.Context, exchangeOrderID string) (*entity.Order, error) {
			return nil, fmt.Errorf("query failed")
		},
	}
	simulator := &mockPaperSimulator{}
	publisher := events.NewPublisher()

	router := NewOrderRouter(config, connector, simulator, publisher, nil)

	// Add order to in-flight
	order := &entity.Order{
		ID:              "order_1",
		ExchangeOrderID: "exchange_order_123",
		StrategyID:      "strategy_1",
		Symbol:          "BTCUSDT",
		Side:            entity.OrderSideBuy,
		Quantity:        decimal.NewFromFloat(0.1),
		CreatedAtUTC:    time.Now().UTC(),
	}

	ctx := context.Background()
	_, _ = router.SubmitOrder(ctx, order)

	// Query should fallback to in-flight cache
	result, err := router.QueryOrderStatus(ctx, "exchange_order_123")

	if err != nil {
		t.Fatalf("expected no error with fallback, got %v", err)
	}

	if result == nil {
		t.Fatal("expected order from fallback cache")
	}
}

func TestOrderRouter_CancelOrder_UnknownMode(t *testing.T) {
	config := entity.DefaultRouterConfig()
	config.Mode = "unknown"

	connector := &mockExchangeConnector{}
	simulator := &mockPaperSimulator{}
	publisher := events.NewPublisher()

	router := NewOrderRouter(config, connector, simulator, publisher, nil)

	ctx := context.Background()
	err := router.CancelOrder(ctx, "order_123")

	if err == nil {
		t.Fatal("expected error for unknown routing mode")
	}
}

func TestOrderRouter_CleanupTimedOutOrders_NoTimeout(t *testing.T) {
	config := entity.DefaultRouterConfig()
	config.TimeoutSeconds = 10

	connector := &mockExchangeConnector{}
	simulator := &mockPaperSimulator{}
	publisher := events.NewPublisher()

	router := NewOrderRouter(config, connector, simulator, publisher, nil)

	// Add order to in-flight
	order := &entity.Order{
		ID:           "order_1",
		StrategyID:   "strategy_1",
		Symbol:       "BTCUSDT",
		Side:         entity.OrderSideBuy,
		Quantity:     decimal.NewFromFloat(0.1),
		CreatedAtUTC: time.Now().UTC(),
	}

	ctx := context.Background()
	_, err := router.SubmitOrder(ctx, order)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Cleanup immediately (no timeout yet)
	cleaned := router.CleanupTimedOutOrders(ctx)

	if cleaned != 0 {
		t.Errorf("expected 0 orders to be cleaned, got %d", cleaned)
	}

	if router.GetInFlightCount() != 1 {
		t.Errorf("expected 1 in-flight order, got %d", router.GetInFlightCount())
	}
}
