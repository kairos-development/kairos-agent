package agent

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/kairos-development/kairos-agent/internal/domain/entity"
	"github.com/kairos-development/kairos-agent/internal/domain/events"
	"github.com/kairos-development/kairos-agent/internal/domain/risk"
	"github.com/kairos-development/kairos-agent/internal/domain/storage"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock repositories
type mockOrderRepository struct {
	orders        map[string]*entity.Order
	createErr     error
	updateErr     error
	getErr        error
	listErr       error
	listActiveErr error
}

func newMockOrderRepository() *mockOrderRepository {
	return &mockOrderRepository{
		orders: make(map[string]*entity.Order),
	}
}

func (m *mockOrderRepository) Create(ctx context.Context, order *entity.Order) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.orders[order.ID] = order
	return nil
}

func (m *mockOrderRepository) Update(ctx context.Context, order *entity.Order) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.orders[order.ID] = order
	return nil
}

func (m *mockOrderRepository) GetByID(ctx context.Context, id string) (*entity.Order, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	order, ok := m.orders[id]
	if !ok {
		return nil, storage.ErrNotFound
	}
	return order, nil
}

func (m *mockOrderRepository) GetByClientOrderID(ctx context.Context, clientOrderID string) (*entity.Order, error) {
	for _, order := range m.orders {
		if order.ClientOrderID == clientOrderID {
			return order, nil
		}
	}
	return nil, storage.ErrNotFound
}

func (m *mockOrderRepository) GetByExchangeOrderID(ctx context.Context, exchangeOrderID string) (*entity.Order, error) {
	for _, order := range m.orders {
		if order.ExchangeOrderID == exchangeOrderID {
			return order, nil
		}
	}
	return nil, storage.ErrNotFound
}

func (m *mockOrderRepository) ListByStrategy(ctx context.Context, strategyID string, limit, offset int) ([]*entity.Order, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	var result []*entity.Order
	for _, order := range m.orders {
		if order.StrategyID == strategyID {
			result = append(result, order)
		}
	}
	return result, nil
}

func (m *mockOrderRepository) ListActive(ctx context.Context) ([]*entity.Order, error) {
	if m.listActiveErr != nil {
		return nil, m.listActiveErr
	}
	var result []*entity.Order
	for _, order := range m.orders {
		if order.IsActive() {
			result = append(result, order)
		}
	}
	return result, nil
}

func (m *mockOrderRepository) ListInFlight(ctx context.Context, since time.Time) ([]*entity.Order, error) {
	var result []*entity.Order
	for _, order := range m.orders {
		if order.CreatedAtUTC.After(since) {
			result = append(result, order)
		}
	}
	return result, nil
}

type mockPositionRepository struct {
	positions map[string]*entity.Position
	getErr    error
}

func newMockPositionRepository() *mockPositionRepository {
	return &mockPositionRepository{
		positions: make(map[string]*entity.Position),
	}
}

func (m *mockPositionRepository) Create(ctx context.Context, position *entity.Position) error {
	m.positions[position.ID] = position
	return nil
}

func (m *mockPositionRepository) Update(ctx context.Context, position *entity.Position) error {
	m.positions[position.ID] = position
	return nil
}

func (m *mockPositionRepository) GetByID(ctx context.Context, id string) (*entity.Position, error) {
	pos, ok := m.positions[id]
	if !ok {
		return nil, storage.ErrNotFound
	}
	return pos, nil
}

func (m *mockPositionRepository) GetByStrategyAndSymbol(ctx context.Context, strategyID, symbol string) (*entity.Position, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	key := strategyID + ":" + symbol
	pos, ok := m.positions[key]
	if !ok {
		return nil, storage.ErrNotFound
	}
	return pos, nil
}

func (m *mockPositionRepository) ListByStrategy(ctx context.Context, strategyID string) ([]*entity.Position, error) {
	var result []*entity.Position
	for _, pos := range m.positions {
		if pos.StrategyID == strategyID {
			result = append(result, pos)
		}
	}
	return result, nil
}

func (m *mockPositionRepository) ListOpen(ctx context.Context) ([]*entity.Position, error) {
	var result []*entity.Position
	for _, pos := range m.positions {
		if pos.ClosedAtUTC == nil {
			result = append(result, pos)
		}
	}
	return result, nil
}

type mockStrategyRepository struct {
	strategies map[string]*entity.Strategy
	getErr     error
}

func newMockStrategyRepository() *mockStrategyRepository {
	return &mockStrategyRepository{
		strategies: make(map[string]*entity.Strategy),
	}
}

func (m *mockStrategyRepository) Create(ctx context.Context, strategy *entity.Strategy) error {
	m.strategies[strategy.ID] = strategy
	return nil
}

func (m *mockStrategyRepository) Update(ctx context.Context, strategy *entity.Strategy) error {
	m.strategies[strategy.ID] = strategy
	return nil
}

func (m *mockStrategyRepository) GetByID(ctx context.Context, id string) (*entity.Strategy, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	strategy, ok := m.strategies[id]
	if !ok {
		return nil, storage.ErrNotFound
	}
	return strategy, nil
}

func (m *mockStrategyRepository) List(ctx context.Context, limit, offset int) ([]*entity.Strategy, error) {
	var result []*entity.Strategy
	for _, s := range m.strategies {
		result = append(result, s)
	}
	return result, nil
}

func (m *mockStrategyRepository) ListActive(ctx context.Context) ([]*entity.Strategy, error) {
	var result []*entity.Strategy
	for _, s := range m.strategies {
		if s.Status == entity.StrategyStatusActive {
			result = append(result, s)
		}
	}
	return result, nil
}

func (m *mockStrategyRepository) Delete(ctx context.Context, id string) error {
	delete(m.strategies, id)
	return nil
}

type mockBalanceRepository struct {
	balance *entity.AccountBalance
	getErr  error
}

func newMockBalanceRepository() *mockBalanceRepository {
	return &mockBalanceRepository{}
}

func (m *mockBalanceRepository) Save(ctx context.Context, balance *entity.AccountBalance) error {
	m.balance = balance
	return nil
}

func (m *mockBalanceRepository) GetLatest(ctx context.Context) (*entity.AccountBalance, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.balance, nil
}

func (m *mockBalanceRepository) GetHistory(ctx context.Context, from, to time.Time) ([]*entity.AccountBalance, error) {
	if m.balance == nil {
		return []*entity.AccountBalance{}, nil
	}
	return []*entity.AccountBalance{m.balance}, nil
}

type mockSymbolRepository struct {
	symbols map[string]*entity.Symbol
	getErr  error
}

func newMockSymbolRepository() *mockSymbolRepository {
	return &mockSymbolRepository{
		symbols: make(map[string]*entity.Symbol),
	}
}

func (m *mockSymbolRepository) Upsert(ctx context.Context, symbol *entity.Symbol) error {
	m.symbols[symbol.Name] = symbol
	return nil
}

func (m *mockSymbolRepository) GetByName(ctx context.Context, name string) (*entity.Symbol, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	symbol, ok := m.symbols[name]
	if !ok {
		return nil, storage.ErrNotFound
	}
	return symbol, nil
}

func (m *mockSymbolRepository) List(ctx context.Context) ([]*entity.Symbol, error) {
	var result []*entity.Symbol
	for _, s := range m.symbols {
		result = append(result, s)
	}
	return result, nil
}

func (m *mockSymbolRepository) ListByStatus(ctx context.Context, status entity.SymbolStatus) ([]*entity.Symbol, error) {
	var result []*entity.Symbol
	for _, s := range m.symbols {
		if s.Status == status {
			result = append(result, s)
		}
	}
	return result, nil
}

type mockConnectorAdapter struct {
	submitOrderErr  error
	cancelOrderErr  error
	queryOrderErr   error
	exchangeOrderID string
	queryResult     *entity.Order
}

func (m *mockConnectorAdapter) SubmitOrder(ctx context.Context, order *entity.Order) (string, error) {
	if m.submitOrderErr != nil {
		return "", m.submitOrderErr
	}
	if m.exchangeOrderID != "" {
		return m.exchangeOrderID, nil
	}
	return "exchange_order_123", nil
}

func (m *mockConnectorAdapter) CancelOrder(ctx context.Context, exchangeOrderID string) error {
	return m.cancelOrderErr
}

func (m *mockConnectorAdapter) QueryOrder(ctx context.Context, exchangeOrderID string) (*entity.Order, error) {
	if m.queryOrderErr != nil {
		return nil, m.queryOrderErr
	}
	return m.queryResult, nil
}

type eventCapture struct {
	events []events.Event
	mu     sync.Mutex
}

func (e *eventCapture) Handle(ctx context.Context, event events.Event) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.events = append(e.events, event)
	return nil
}

func (e *eventCapture) Count() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return len(e.events)
}

func (e *eventCapture) LastEvent() events.Event {
	e.mu.Lock()
	defer e.mu.Unlock()
	if len(e.events) == 0 {
		return nil
	}
	return e.events[len(e.events)-1]
}

// testOrderService wraps orderService for testing with mock connector
type testOrderService struct {
	orderRepo    storage.OrderRepository
	positionRepo storage.PositionRepository
	strategyRepo storage.StrategyRepository
	balanceRepo  storage.BalanceRepository
	symbolRepo   storage.SymbolRepository
	riskEngine   *risk.Engine
	connector    *mockConnectorAdapter
	publisher    *events.Publisher
}

func (s *testOrderService) CreateOrder(ctx context.Context, req CreateOrderRequest) (*entity.Order, error) {
	// Inline the CreateOrder logic with mock connector
	strategy, err := s.strategyRepo.GetByID(ctx, req.StrategyID)
	if err != nil {
		return nil, fmt.Errorf("get strategy: %w", err)
	}

	if !strategy.CanTrade() {
		return nil, fmt.Errorf("strategy cannot trade in current state: %s", strategy.Status)
	}

	symbol, err := s.symbolRepo.GetByName(ctx, req.Symbol)
	if err != nil {
		return nil, fmt.Errorf("get symbol: %w", err)
	}

	position, err := s.positionRepo.GetByStrategyAndSymbol(ctx, req.StrategyID, req.Symbol)
	if err != nil && err != storage.ErrNotFound {
		return nil, fmt.Errorf("get position: %w", err)
	}

	balances, err := s.balanceRepo.GetLatest(ctx)
	if err != nil {
		return nil, fmt.Errorf("get balance: %w", err)
	}

	var balance *entity.Balance
	if balances != nil {
		balance = balances.GetBalance(symbol.QuoteCurrency)
	}

	order := &entity.Order{
		ID:            generateOrderID(),
		ClientOrderID: generateClientOrderID(),
		StrategyID:    req.StrategyID,
		Symbol:        req.Symbol,
		Side:          req.Side,
		Type:          req.Type,
		Status:        entity.OrderStatusPending,
		TimeInForce:   req.TimeInForce,
		Quantity:      req.Quantity,
		Price:         req.Price,
		FilledQty:     decimal.Zero,
		RemainingQty:  req.Quantity,
		AvgFillPrice:  decimal.Zero,
		CreatedAtUTC:  time.Now().UTC(),
		UpdatedAtUTC:  time.Now().UTC(),
	}

	if err := s.riskEngine.ValidateOrder(ctx, order, strategy, symbol, balance, position); err != nil {
		return nil, fmt.Errorf("risk validation failed: %w", err)
	}

	if err := s.orderRepo.Create(ctx, order); err != nil {
		return nil, fmt.Errorf("create order: %w", err)
	}

	s.publishOrderCreated(ctx, order)

	return order, nil
}

func (s *testOrderService) SubmitOrder(ctx context.Context, orderID string) error {
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("get order: %w", err)
	}

	if !order.IsPending() {
		return fmt.Errorf("order is not pending: %s", order.Status)
	}

	exchangeOrderID, err := s.connector.SubmitOrder(ctx, order)
	if err != nil {
		order.Status = entity.OrderStatusRejected
		order.UpdatedAtUTC = time.Now().UTC()
		_ = s.orderRepo.Update(ctx, order)
		s.publishOrderRejected(ctx, order, err.Error())
		return fmt.Errorf("submit order to exchange: %w", err)
	}

	now := time.Now().UTC()
	order.ExchangeOrderID = exchangeOrderID
	order.Status = entity.OrderStatusSubmitted
	order.SubmittedAtUTC = &now
	order.UpdatedAtUTC = now

	if err := s.orderRepo.Update(ctx, order); err != nil {
		return fmt.Errorf("update order: %w", err)
	}

	s.publishOrderSubmitted(ctx, order)

	return nil
}

func (s *testOrderService) CancelOrder(ctx context.Context, orderID string) error {
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("get order: %w", err)
	}

	if !order.IsActive() {
		return fmt.Errorf("order is not active: %s", order.Status)
	}

	if err := s.connector.CancelOrder(ctx, order.ExchangeOrderID); err != nil {
		return fmt.Errorf("cancel order on exchange: %w", err)
	}

	order.Status = entity.OrderStatusCanceled
	order.UpdatedAtUTC = time.Now().UTC()

	if err := s.orderRepo.Update(ctx, order); err != nil {
		return fmt.Errorf("update order: %w", err)
	}

	s.publishOrderCanceled(ctx, order, "user_requested")

	return nil
}

func (s *testOrderService) GetOrder(ctx context.Context, orderID string) (*entity.Order, error) {
	return s.orderRepo.GetByID(ctx, orderID)
}

func (s *testOrderService) ListOrders(ctx context.Context, strategyID string, limit, offset int) ([]*entity.Order, error) {
	return s.orderRepo.ListByStrategy(ctx, strategyID, limit, offset)
}

func (s *testOrderService) ReconcileOrders(ctx context.Context) error {
	localOrders, err := s.orderRepo.ListActive(ctx)
	if err != nil {
		return fmt.Errorf("list active orders: %w", err)
	}

	for _, order := range localOrders {
		if order.ExchangeOrderID == "" {
			continue
		}

		exchangeOrder, err := s.connector.QueryOrder(ctx, order.ExchangeOrderID)
		if err != nil {
			continue
		}

		order.Status = exchangeOrder.Status
		order.FilledQty = exchangeOrder.FilledQty
		order.RemainingQty = exchangeOrder.RemainingQty
		order.AvgFillPrice = exchangeOrder.AvgFillPrice
		order.UpdatedAtUTC = time.Now().UTC()

		if err := s.orderRepo.Update(ctx, order); err != nil {
			return fmt.Errorf("update order during reconciliation: %w", err)
		}
	}

	return nil
}

func (s *testOrderService) publishOrderCreated(ctx context.Context, order *entity.Order) {
	if s.publisher == nil {
		return
	}

	event := &events.OrderCreatedEvent{
		BaseEvent: events.BaseEvent{
			EventType:  events.EventTypeOrderCreated,
			OccurredAt: time.Now().UTC(),
		},
		OrderID:       order.ID,
		ClientOrderID: order.ClientOrderID,
		StrategyID:    order.StrategyID,
		Symbol:        order.Symbol,
		Side:          string(order.Side),
		OrderType:     string(order.Type),
		Quantity:      order.Quantity,
		Price:         order.Price,
	}

	_ = s.publisher.Publish(ctx, event)
}

func (s *testOrderService) publishOrderSubmitted(ctx context.Context, order *entity.Order) {
	if s.publisher == nil {
		return
	}

	event := &events.OrderSubmittedEvent{
		BaseEvent: events.BaseEvent{
			EventType:  events.EventTypeOrderSubmitted,
			OccurredAt: time.Now().UTC(),
		},
		OrderID:         order.ID,
		ClientOrderID:   order.ClientOrderID,
		ExchangeOrderID: order.ExchangeOrderID,
		SubmittedAtUTC:  *order.SubmittedAtUTC,
	}

	_ = s.publisher.Publish(ctx, event)
}

func (s *testOrderService) publishOrderCanceled(ctx context.Context, order *entity.Order, reason string) {
	if s.publisher == nil {
		return
	}

	event := &events.OrderCanceledEvent{
		BaseEvent: events.BaseEvent{
			EventType:  events.EventTypeOrderCanceled,
			OccurredAt: time.Now().UTC(),
		},
		OrderID:       order.ID,
		CanceledAtUTC: time.Now().UTC(),
		Reason:        reason,
	}

	_ = s.publisher.Publish(ctx, event)
}

func (s *testOrderService) publishOrderRejected(ctx context.Context, order *entity.Order, reason string) {
	if s.publisher == nil {
		return
	}

	event := &events.OrderRejectedEvent{
		BaseEvent: events.BaseEvent{
			EventType:  events.EventTypeOrderRejected,
			OccurredAt: time.Now().UTC(),
		},
		OrderID:       order.ID,
		RejectedAtUTC: time.Now().UTC(),
		Reason:        reason,
	}

	_ = s.publisher.Publish(ctx, event)
}

func setupOrderService() (*testOrderService, *mockOrderRepository, *mockConnectorAdapter, *events.Publisher, *eventCapture) {
	orderRepo := newMockOrderRepository()
	positionRepo := newMockPositionRepository()
	strategyRepo := newMockStrategyRepository()
	balanceRepo := newMockBalanceRepository()
	symbolRepo := newMockSymbolRepository()
	connector := &mockConnectorAdapter{}
	publisher := events.NewPublisher()
	capture := &eventCapture{}

	// Subscribe to all event types
	publisher.Subscribe(events.EventTypeOrderCreated, capture)
	publisher.Subscribe(events.EventTypeOrderSubmitted, capture)
	publisher.Subscribe(events.EventTypeOrderCanceled, capture)
	publisher.Subscribe(events.EventTypeOrderRejected, capture)

	riskEngine := risk.NewEngine(publisher)

	// Setup default test data
	strategyRepo.strategies["strat1"] = &entity.Strategy{
		ID:     "strat1",
		Status: entity.StrategyStatusActive,
	}

	symbolRepo.symbols["BTCUSDT"] = &entity.Symbol{
		Name:          "BTCUSDT",
		BaseCurrency:  "BTC",
		QuoteCurrency: "USDT",
		Status:        entity.SymbolStatusTrading,
		MinOrderQty:   decimal.NewFromFloat(0.001),
		MaxOrderQty:   decimal.NewFromInt(100),
		MinPrice:      decimal.NewFromInt(1),
		MaxPrice:      decimal.NewFromInt(100000),
		TickSize:      decimal.NewFromInt(1),
		StepSize:      decimal.NewFromFloat(0.001),
		MinNotional:   decimal.NewFromInt(10),
	}

	balanceRepo.balance = &entity.AccountBalance{
		Balances: []entity.Balance{
			{
				Asset:     "USDT",
				Total:     decimal.NewFromInt(10000),
				Available: decimal.NewFromInt(9500),
				Locked:    decimal.NewFromInt(500),
			},
		},
	}

	svc := &testOrderService{
		orderRepo:    orderRepo,
		positionRepo: positionRepo,
		strategyRepo: strategyRepo,
		balanceRepo:  balanceRepo,
		symbolRepo:   symbolRepo,
		riskEngine:   riskEngine,
		connector:    connector,
		publisher:    publisher,
	}

	return svc, orderRepo, connector, publisher, capture
}

func TestNewOrderService(t *testing.T) {
	svc, _, _, _, _ := setupOrderService()
	require.NotNil(t, svc)
	assert.NotNil(t, svc.orderRepo)
	assert.NotNil(t, svc.riskEngine)
}

func TestCreateOrder_Success(t *testing.T) {
	svc, orderRepo, _, _, capture := setupOrderService()
	ctx := context.Background()

	req := CreateOrderRequest{
		StrategyID:  "strat1",
		Symbol:      "BTCUSDT",
		Side:        entity.OrderSideBuy,
		Type:        entity.OrderTypeLimit,
		Quantity:    decimal.NewFromFloat(0.1),
		Price:       decimal.NewFromInt(50000),
		TimeInForce: entity.TimeInForceGTC,
	}

	order, err := svc.CreateOrder(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, order)
	assert.Equal(t, "strat1", order.StrategyID)
	assert.Equal(t, "BTCUSDT", order.Symbol)
	assert.Equal(t, entity.OrderSideBuy, order.Side)
	assert.Equal(t, entity.OrderStatusPending, order.Status)
	assert.True(t, order.Quantity.Equal(decimal.NewFromFloat(0.1)))

	// Verify order was persisted
	assert.Len(t, orderRepo.orders, 1)

	// Verify event was published
	assert.Equal(t, 1, capture.Count())
	assert.Equal(t, events.EventTypeOrderCreated, capture.LastEvent().Type())
}

func TestCreateOrder_StrategyNotFound(t *testing.T) {
	svc, _, _, _, _ := setupOrderService()
	ctx := context.Background()

	req := CreateOrderRequest{
		StrategyID:  "nonexistent",
		Symbol:      "BTCUSDT",
		Side:        entity.OrderSideBuy,
		Type:        entity.OrderTypeLimit,
		Quantity:    decimal.NewFromFloat(0.1),
		Price:       decimal.NewFromInt(50000),
		TimeInForce: entity.TimeInForceGTC,
	}

	_, err := svc.CreateOrder(ctx, req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get strategy")
}

func TestCreateOrder_StrategyCannotTrade(t *testing.T) {
	svc, _, _, _, _ := setupOrderService()
	ctx := context.Background()

	// Set strategy to idle state
	strategyRepo := svc.strategyRepo.(*mockStrategyRepository)
	strategyRepo.strategies["strat1"].Status = entity.StrategyStatusIdle

	req := CreateOrderRequest{
		StrategyID:  "strat1",
		Symbol:      "BTCUSDT",
		Side:        entity.OrderSideBuy,
		Type:        entity.OrderTypeLimit,
		Quantity:    decimal.NewFromFloat(0.1),
		Price:       decimal.NewFromInt(50000),
		TimeInForce: entity.TimeInForceGTC,
	}

	_, err := svc.CreateOrder(ctx, req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "strategy cannot trade")
}

func TestCreateOrder_SymbolNotFound(t *testing.T) {
	svc, _, _, _, _ := setupOrderService()
	ctx := context.Background()

	req := CreateOrderRequest{
		StrategyID:  "strat1",
		Symbol:      "NONEXISTENT",
		Side:        entity.OrderSideBuy,
		Type:        entity.OrderTypeLimit,
		Quantity:    decimal.NewFromFloat(0.1),
		Price:       decimal.NewFromInt(50000),
		TimeInForce: entity.TimeInForceGTC,
	}

	_, err := svc.CreateOrder(ctx, req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get symbol")
}

func TestSubmitOrder_Success(t *testing.T) {
	svc, orderRepo, connector, _, capture := setupOrderService()
	ctx := context.Background()

	// Create a pending order
	order := &entity.Order{
		ID:           "order1",
		StrategyID:   "strat1",
		Symbol:       "BTCUSDT",
		Side:         entity.OrderSideBuy,
		Type:         entity.OrderTypeLimit,
		Status:       entity.OrderStatusPending,
		Quantity:     decimal.NewFromFloat(0.1),
		Price:        decimal.NewFromInt(50000),
		CreatedAtUTC: time.Now().UTC(),
		UpdatedAtUTC: time.Now().UTC(),
	}
	orderRepo.orders[order.ID] = order

	connector.exchangeOrderID = "exchange_123"

	err := svc.SubmitOrder(ctx, "order1")
	require.NoError(t, err)

	// Verify order status updated
	updatedOrder := orderRepo.orders["order1"]
	assert.Equal(t, entity.OrderStatusSubmitted, updatedOrder.Status)
	assert.Equal(t, "exchange_123", updatedOrder.ExchangeOrderID)
	assert.NotNil(t, updatedOrder.SubmittedAtUTC)

	// Verify event was published
	assert.Equal(t, 1, capture.Count())
	assert.Equal(t, events.EventTypeOrderSubmitted, capture.LastEvent().Type())
}

func TestSubmitOrder_OrderNotFound(t *testing.T) {
	svc, _, _, _, _ := setupOrderService()
	ctx := context.Background()

	err := svc.SubmitOrder(ctx, "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get order")
}

func TestSubmitOrder_OrderNotPending(t *testing.T) {
	svc, orderRepo, _, _, _ := setupOrderService()
	ctx := context.Background()

	// Create a submitted order
	order := &entity.Order{
		ID:     "order1",
		Status: entity.OrderStatusSubmitted,
	}
	orderRepo.orders[order.ID] = order

	err := svc.SubmitOrder(ctx, "order1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "order is not pending")
}

func TestSubmitOrder_ExchangeError(t *testing.T) {
	svc, orderRepo, connector, _, capture := setupOrderService()
	ctx := context.Background()

	// Create a pending order
	order := &entity.Order{
		ID:           "order1",
		Status:       entity.OrderStatusPending,
		CreatedAtUTC: time.Now().UTC(),
		UpdatedAtUTC: time.Now().UTC(),
	}
	orderRepo.orders[order.ID] = order

	connector.submitOrderErr = errors.New("exchange error")

	err := svc.SubmitOrder(ctx, "order1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "submit order to exchange")

	// Verify order status updated to rejected
	updatedOrder := orderRepo.orders["order1"]
	assert.Equal(t, entity.OrderStatusRejected, updatedOrder.Status)

	// Verify rejection event was published
	assert.Equal(t, 1, capture.Count())
	assert.Equal(t, events.EventTypeOrderRejected, capture.LastEvent().Type())
}

func TestCancelOrder_Success(t *testing.T) {
	svc, orderRepo, _, _, capture := setupOrderService()
	ctx := context.Background()

	// Create an active order
	order := &entity.Order{
		ID:              "order1",
		Status:          entity.OrderStatusSubmitted,
		ExchangeOrderID: "exchange_123",
		CreatedAtUTC:    time.Now().UTC(),
		UpdatedAtUTC:    time.Now().UTC(),
	}
	orderRepo.orders[order.ID] = order

	err := svc.CancelOrder(ctx, "order1")
	require.NoError(t, err)

	// Verify order status updated
	updatedOrder := orderRepo.orders["order1"]
	assert.Equal(t, entity.OrderStatusCanceled, updatedOrder.Status)

	// Verify event was published
	assert.Equal(t, 1, capture.Count())
	assert.Equal(t, events.EventTypeOrderCanceled, capture.LastEvent().Type())
}

func TestCancelOrder_OrderNotFound(t *testing.T) {
	svc, _, _, _, _ := setupOrderService()
	ctx := context.Background()

	err := svc.CancelOrder(ctx, "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get order")
}

func TestCancelOrder_OrderNotActive(t *testing.T) {
	svc, orderRepo, _, _, _ := setupOrderService()
	ctx := context.Background()

	// Create a canceled order
	order := &entity.Order{
		ID:     "order1",
		Status: entity.OrderStatusCanceled,
	}
	orderRepo.orders[order.ID] = order

	err := svc.CancelOrder(ctx, "order1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "order is not active")
}

func TestCancelOrder_ExchangeError(t *testing.T) {
	svc, orderRepo, connector, _, _ := setupOrderService()
	ctx := context.Background()

	// Create an active order
	order := &entity.Order{
		ID:              "order1",
		Status:          entity.OrderStatusSubmitted,
		ExchangeOrderID: "exchange_123",
	}
	orderRepo.orders[order.ID] = order

	connector.cancelOrderErr = errors.New("exchange error")

	err := svc.CancelOrder(ctx, "order1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cancel order on exchange")
}

func TestGetOrder_Success(t *testing.T) {
	svc, orderRepo, _, _, _ := setupOrderService()
	ctx := context.Background()

	// Create an order
	order := &entity.Order{
		ID:     "order1",
		Symbol: "BTCUSDT",
	}
	orderRepo.orders[order.ID] = order

	retrieved, err := svc.GetOrder(ctx, "order1")
	require.NoError(t, err)
	assert.Equal(t, "order1", retrieved.ID)
	assert.Equal(t, "BTCUSDT", retrieved.Symbol)
}

func TestGetOrder_NotFound(t *testing.T) {
	svc, _, _, _, _ := setupOrderService()
	ctx := context.Background()

	_, err := svc.GetOrder(ctx, "nonexistent")
	assert.ErrorIs(t, err, storage.ErrNotFound)
}

func TestListOrders_Success(t *testing.T) {
	svc, orderRepo, _, _, _ := setupOrderService()
	ctx := context.Background()

	// Create multiple orders
	orderRepo.orders["order1"] = &entity.Order{ID: "order1", StrategyID: "strat1"}
	orderRepo.orders["order2"] = &entity.Order{ID: "order2", StrategyID: "strat1"}
	orderRepo.orders["order3"] = &entity.Order{ID: "order3", StrategyID: "strat2"}

	orders, err := svc.ListOrders(ctx, "strat1", 10, 0)
	require.NoError(t, err)
	assert.Len(t, orders, 2)
}

func TestReconcileOrders_Success(t *testing.T) {
	svc, orderRepo, connector, _, _ := setupOrderService()
	ctx := context.Background()

	// Create active orders
	now := time.Now().UTC()
	orderRepo.orders["order1"] = &entity.Order{
		ID:              "order1",
		Status:          entity.OrderStatusSubmitted,
		ExchangeOrderID: "exchange_123",
		FilledQty:       decimal.Zero,
		RemainingQty:    decimal.NewFromFloat(0.1),
		SubmittedAtUTC:  &now,
	}

	// Setup connector to return updated order
	connector.queryResult = &entity.Order{
		Status:       entity.OrderStatusFilled,
		FilledQty:    decimal.NewFromFloat(0.1),
		RemainingQty: decimal.Zero,
		AvgFillPrice: decimal.NewFromInt(50000),
	}

	err := svc.ReconcileOrders(ctx)
	require.NoError(t, err)

	// Verify order was updated
	updatedOrder := orderRepo.orders["order1"]
	assert.Equal(t, entity.OrderStatusFilled, updatedOrder.Status)
	assert.True(t, updatedOrder.FilledQty.Equal(decimal.NewFromFloat(0.1)))
}

func TestReconcileOrders_SkipsOrdersWithoutExchangeID(t *testing.T) {
	svc, orderRepo, _, _, _ := setupOrderService()
	ctx := context.Background()

	// Create order without exchange ID
	orderRepo.orders["order1"] = &entity.Order{
		ID:              "order1",
		Status:          entity.OrderStatusPending,
		ExchangeOrderID: "",
	}

	err := svc.ReconcileOrders(ctx)
	require.NoError(t, err)

	// Verify order was not updated
	order := orderRepo.orders["order1"]
	assert.Equal(t, entity.OrderStatusPending, order.Status)
}

func TestReconcileOrders_ContinuesOnQueryError(t *testing.T) {
	svc, orderRepo, connector, _, _ := setupOrderService()
	ctx := context.Background()

	now := time.Now().UTC()
	orderRepo.orders["order1"] = &entity.Order{
		ID:              "order1",
		Status:          entity.OrderStatusSubmitted,
		ExchangeOrderID: "exchange_123",
		SubmittedAtUTC:  &now,
	}

	connector.queryOrderErr = errors.New("query error")

	// Should not return error, just continue
	err := svc.ReconcileOrders(ctx)
	require.NoError(t, err)
}

func TestPublishOrderCreated_NilPublisher(t *testing.T) {
	svc, _, _, _, _ := setupOrderService()
	svc.publisher = nil
	ctx := context.Background()

	order := &entity.Order{ID: "order1"}

	// Should not panic
	svc.publishOrderCreated(ctx, order)
}

func TestPublishOrderSubmitted_NilPublisher(t *testing.T) {
	svc, _, _, _, _ := setupOrderService()
	svc.publisher = nil
	ctx := context.Background()

	now := time.Now().UTC()
	order := &entity.Order{ID: "order1", SubmittedAtUTC: &now}

	// Should not panic
	svc.publishOrderSubmitted(ctx, order)
}

func TestPublishOrderCanceled_NilPublisher(t *testing.T) {
	svc, _, _, _, _ := setupOrderService()
	svc.publisher = nil
	ctx := context.Background()

	order := &entity.Order{ID: "order1"}

	// Should not panic
	svc.publishOrderCanceled(ctx, order, "test")
}

func TestPublishOrderRejected_NilPublisher(t *testing.T) {
	svc, _, _, _, _ := setupOrderService()
	svc.publisher = nil
	ctx := context.Background()

	order := &entity.Order{ID: "order1"}

	// Should not panic
	svc.publishOrderRejected(ctx, order, "test")
}

func TestGenerateOrderID(t *testing.T) {
	id1 := generateOrderID()
	time.Sleep(1 * time.Millisecond)
	id2 := generateOrderID()

	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.NotEqual(t, id1, id2)
	assert.Contains(t, id1, "ord_")
}

func TestGenerateClientOrderID(t *testing.T) {
	id1 := generateClientOrderID()
	time.Sleep(1 * time.Millisecond)
	id2 := generateClientOrderID()

	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.NotEqual(t, id1, id2)
	assert.Contains(t, id1, "cli_")
}
