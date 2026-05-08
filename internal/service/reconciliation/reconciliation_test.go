package reconciliation

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/kairos-development/kairos-agent/internal/domain/entity"
	"github.com/kairos-development/kairos-agent/internal/domain/events"
	"github.com/kairos-development/kairos-agent/internal/domain/storage"
	"github.com/shopspring/decimal"
)

// Mock implementations

type mockExchangeConnector struct {
	getOpenOrdersFunc    func(ctx context.Context) ([]*entity.Order, error)
	getPositionsFunc     func(ctx context.Context) ([]*entity.Position, error)
	queryOrderStatusFunc func(ctx context.Context, exchangeOrderID string) (*entity.Order, error)
}

func (m *mockExchangeConnector) GetOpenOrders(ctx context.Context) ([]*entity.Order, error) {
	if m.getOpenOrdersFunc != nil {
		return m.getOpenOrdersFunc(ctx)
	}
	return []*entity.Order{}, nil
}

func (m *mockExchangeConnector) GetPositions(ctx context.Context) ([]*entity.Position, error) {
	if m.getPositionsFunc != nil {
		return m.getPositionsFunc(ctx)
	}
	return []*entity.Position{}, nil
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

type mockOrderRepository struct {
	orders map[string]*entity.Order
}

func newMockOrderRepository() *mockOrderRepository {
	return &mockOrderRepository{
		orders: make(map[string]*entity.Order),
	}
}

func (m *mockOrderRepository) ListActive(ctx context.Context) ([]*entity.Order, error) {
	var active []*entity.Order
	for _, order := range m.orders {
		if order.IsActive() {
			active = append(active, order)
		}
	}
	return active, nil
}

func (m *mockOrderRepository) Update(ctx context.Context, order *entity.Order) error {
	m.orders[order.ID] = order
	return nil
}

func (m *mockOrderRepository) Create(ctx context.Context, order *entity.Order) error {
	m.orders[order.ID] = order
	return nil
}

func (m *mockOrderRepository) GetByID(ctx context.Context, orderID string) (*entity.Order, error) {
	order, exists := m.orders[orderID]
	if !exists {
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

func (m *mockOrderRepository) ListInFlight(ctx context.Context, since time.Time) ([]*entity.Order, error) {
	return nil, nil
}

type mockPositionRepository struct {
	positions map[string]*entity.Position
}

func newMockPositionRepository() *mockPositionRepository {
	return &mockPositionRepository{
		positions: make(map[string]*entity.Position),
	}
}

func (m *mockPositionRepository) ListAll(ctx context.Context) ([]*entity.Position, error) {
	var all []*entity.Position
	for _, pos := range m.positions {
		all = append(all, pos)
	}
	return all, nil
}

func (m *mockPositionRepository) Update(ctx context.Context, position *entity.Position) error {
	m.positions[position.ID] = position
	return nil
}

func (m *mockPositionRepository) Create(ctx context.Context, position *entity.Position) error {
	m.positions[position.ID] = position
	return nil
}

func (m *mockPositionRepository) GetByID(ctx context.Context, positionID string) (*entity.Position, error) {
	pos, exists := m.positions[positionID]
	if !exists {
		return nil, storage.ErrNotFound
	}
	return pos, nil
}

func (m *mockOrderRepository) ListByStrategy(ctx context.Context, strategyID string, limit, offset int) ([]*entity.Order, error) {
	return nil, nil
}

func (m *mockPositionRepository) GetByStrategyAndSymbol(ctx context.Context, strategyID, symbol string) (*entity.Position, error) {
	for _, pos := range m.positions {
		if pos.StrategyID == strategyID && pos.Symbol == symbol {
			return pos, nil
		}
	}
	return nil, storage.ErrNotFound
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
	var open []*entity.Position
	for _, pos := range m.positions {
		if !pos.IsFlat() {
			open = append(open, pos)
		}
	}
	return open, nil
}

func TestNewReconciler(t *testing.T) {
	connector := &mockExchangeConnector{}
	orderRepo := newMockOrderRepository()
	positionRepo := newMockPositionRepository()
	publisher := events.NewPublisher()

	reconciler := NewReconciler(connector, orderRepo, positionRepo, publisher, nil)

	if reconciler == nil {
		t.Fatal("expected reconciler to be created")
	}

	if reconciler.connector == nil {
		t.Error("expected connector to be set")
	}

	if reconciler.orderRepo == nil {
		t.Error("expected orderRepo to be set")
	}

	if reconciler.positionRepo == nil {
		t.Error("expected positionRepo to be set")
	}
}

func TestReconciler_ReconcileOrders_NoOrders(t *testing.T) {
	connector := &mockExchangeConnector{}
	orderRepo := newMockOrderRepository()
	positionRepo := newMockPositionRepository()
	publisher := events.NewPublisher()

	reconciler := NewReconciler(connector, orderRepo, positionRepo, publisher, nil)

	ctx := context.Background()
	result, err := reconciler.ReconcileOrders(ctx)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.OrdersUpdated != 0 {
		t.Errorf("expected 0 orders updated, got %d", result.OrdersUpdated)
	}

	if len(result.ZombieOrders) != 0 {
		t.Errorf("expected 0 zombie orders, got %d", len(result.ZombieOrders))
	}
}

func TestReconciler_ReconcileOrders_ZombieOrder(t *testing.T) {
	connector := &mockExchangeConnector{
		getOpenOrdersFunc: func(ctx context.Context) ([]*entity.Order, error) {
			// Return empty list (no orders on exchange)
			return []*entity.Order{}, nil
		},
	}

	orderRepo := newMockOrderRepository()
	positionRepo := newMockPositionRepository()
	publisher := events.NewPublisher()

	// Add local order that doesn't exist on exchange
	localOrder := &entity.Order{
		ID:              "order_1",
		ExchangeOrderID: "exchange_order_123",
		Status:          entity.OrderStatusSubmitted,
		Symbol:          "BTCUSDT",
		Side:            entity.OrderSideBuy,
		Quantity:        decimal.NewFromFloat(0.1),
	}
	orderRepo.Create(context.Background(), localOrder)

	reconciler := NewReconciler(connector, orderRepo, positionRepo, publisher, nil)

	ctx := context.Background()
	result, err := reconciler.ReconcileOrders(ctx)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.OrdersUpdated != 1 {
		t.Errorf("expected 1 order updated, got %d", result.OrdersUpdated)
	}

	if len(result.ZombieOrders) != 1 {
		t.Errorf("expected 1 zombie order, got %d", len(result.ZombieOrders))
	}

	// Check order was marked as canceled
	updatedOrder, _ := orderRepo.GetByID(ctx, "order_1")
	if updatedOrder.Status != entity.OrderStatusCanceled {
		t.Errorf("expected status canceled, got %s", updatedOrder.Status)
	}
}

func TestReconciler_ReconcileOrders_StateMismatch(t *testing.T) {
	exchangeOrder := &entity.Order{
		ID:              "order_1",
		ExchangeOrderID: "exchange_order_123",
		Status:          entity.OrderStatusFilled,
		FilledQty:       decimal.NewFromFloat(0.1),
		RemainingQty:    decimal.Zero,
		AvgFillPrice:    decimal.NewFromInt(50000),
	}

	connector := &mockExchangeConnector{
		getOpenOrdersFunc: func(ctx context.Context) ([]*entity.Order, error) {
			return []*entity.Order{exchangeOrder}, nil
		},
	}

	orderRepo := newMockOrderRepository()
	positionRepo := newMockPositionRepository()
	publisher := events.NewPublisher()

	// Add local order with different state
	localOrder := &entity.Order{
		ID:              "order_1",
		ExchangeOrderID: "exchange_order_123",
		Status:          entity.OrderStatusSubmitted,
		FilledQty:       decimal.Zero,
		RemainingQty:    decimal.NewFromFloat(0.1),
		AvgFillPrice:    decimal.Zero,
	}
	orderRepo.Create(context.Background(), localOrder)

	reconciler := NewReconciler(connector, orderRepo, positionRepo, publisher, nil)

	ctx := context.Background()
	result, err := reconciler.ReconcileOrders(ctx)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.OrdersUpdated != 1 {
		t.Errorf("expected 1 order updated, got %d", result.OrdersUpdated)
	}

	// Check order was updated to match exchange
	updatedOrder, _ := orderRepo.GetByID(ctx, "order_1")
	if updatedOrder.Status != entity.OrderStatusFilled {
		t.Errorf("expected status filled, got %s", updatedOrder.Status)
	}

	if !updatedOrder.FilledQty.Equal(exchangeOrder.FilledQty) {
		t.Errorf("expected filled qty %s, got %s", exchangeOrder.FilledQty.String(), updatedOrder.FilledQty.String())
	}
}

func TestReconciler_ReconcileOrders_PartialFill(t *testing.T) {
	exchangeOrder := &entity.Order{
		ID:              "order_1",
		ExchangeOrderID: "exchange_order_123",
		Status:          entity.OrderStatusPartial,
		FilledQty:       decimal.NewFromFloat(0.05),
		RemainingQty:    decimal.NewFromFloat(0.05),
		AvgFillPrice:    decimal.NewFromInt(50000),
	}

	connector := &mockExchangeConnector{
		getOpenOrdersFunc: func(ctx context.Context) ([]*entity.Order, error) {
			return []*entity.Order{exchangeOrder}, nil
		},
	}

	orderRepo := newMockOrderRepository()
	positionRepo := newMockPositionRepository()
	publisher := events.NewPublisher()

	localOrder := &entity.Order{
		ID:              "order_1",
		ExchangeOrderID: "exchange_order_123",
		Status:          entity.OrderStatusSubmitted,
		FilledQty:       decimal.Zero,
		RemainingQty:    decimal.NewFromFloat(0.1),
	}
	orderRepo.Create(context.Background(), localOrder)

	reconciler := NewReconciler(connector, orderRepo, positionRepo, publisher, nil)

	ctx := context.Background()
	result, err := reconciler.ReconcileOrders(ctx)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(result.PartialFills) != 1 {
		t.Errorf("expected 1 partial fill, got %d", len(result.PartialFills))
	}
}

func TestReconciler_ReconcileOrders_SkipNoExchangeOrderID(t *testing.T) {
	connector := &mockExchangeConnector{
		getOpenOrdersFunc: func(ctx context.Context) ([]*entity.Order, error) {
			return []*entity.Order{}, nil
		},
	}

	orderRepo := newMockOrderRepository()
	positionRepo := newMockPositionRepository()
	publisher := events.NewPublisher()

	// Add local order without exchange order ID (not yet submitted)
	localOrder := &entity.Order{
		ID:              "order_1",
		ExchangeOrderID: "",
		Status:          entity.OrderStatusPending,
	}
	orderRepo.Create(context.Background(), localOrder)

	reconciler := NewReconciler(connector, orderRepo, positionRepo, publisher, nil)

	ctx := context.Background()
	result, err := reconciler.ReconcileOrders(ctx)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Should skip order without exchange ID
	if result.OrdersUpdated != 0 {
		t.Errorf("expected 0 orders updated, got %d", result.OrdersUpdated)
	}
}

func TestReconciler_ReconcilePositions_NoPositions(t *testing.T) {
	connector := &mockExchangeConnector{}
	orderRepo := newMockOrderRepository()
	positionRepo := newMockPositionRepository()
	publisher := events.NewPublisher()

	reconciler := NewReconciler(connector, orderRepo, positionRepo, publisher, nil)

	ctx := context.Background()
	result, err := reconciler.ReconcilePositions(ctx)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.PositionsUpdated != 0 {
		t.Errorf("expected 0 positions updated, got %d", result.PositionsUpdated)
	}
}

func TestReconciler_ReconcilePositions_PositionClosed(t *testing.T) {
	exchangePosition := &entity.Position{
		Symbol:   "BTCUSDT",
		Quantity: decimal.Zero, // Flat position
	}

	connector := &mockExchangeConnector{
		getPositionsFunc: func(ctx context.Context) ([]*entity.Position, error) {
			return []*entity.Position{exchangePosition}, nil
		},
	}

	orderRepo := newMockOrderRepository()
	positionRepo := newMockPositionRepository()
	publisher := events.NewPublisher()

	// Add local position that is still open
	localPosition := &entity.Position{
		ID:       "pos_1",
		Symbol:   "BTCUSDT",
		Quantity: decimal.NewFromFloat(0.1),
		Side:     entity.PositionSideLong,
	}
	positionRepo.Create(context.Background(), localPosition)

	reconciler := NewReconciler(connector, orderRepo, positionRepo, publisher, nil)

	ctx := context.Background()
	result, err := reconciler.ReconcilePositions(ctx)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.PositionsUpdated != 1 {
		t.Errorf("expected 1 position updated, got %d", result.PositionsUpdated)
	}

	// Check position was updated to flat
	updatedPosition, _ := positionRepo.GetByID(ctx, "pos_1")
	if !updatedPosition.Quantity.IsZero() {
		t.Errorf("expected position to be flat, got quantity %s", updatedPosition.Quantity.String())
	}
}

func TestReconciler_ReconcilePositions_StateMismatch(t *testing.T) {
	exchangePosition := &entity.Position{
		Symbol:        "BTCUSDT",
		Quantity:      decimal.NewFromFloat(0.15),
		CurrentPrice:  decimal.NewFromInt(51000),
		UnrealizedPnL: decimal.NewFromInt(150),
	}

	connector := &mockExchangeConnector{
		getPositionsFunc: func(ctx context.Context) ([]*entity.Position, error) {
			return []*entity.Position{exchangePosition}, nil
		},
	}

	orderRepo := newMockOrderRepository()
	positionRepo := newMockPositionRepository()
	publisher := events.NewPublisher()

	localPosition := &entity.Position{
		ID:            "pos_1",
		Symbol:        "BTCUSDT",
		Quantity:      decimal.NewFromFloat(0.1),
		CurrentPrice:  decimal.NewFromInt(50000),
		UnrealizedPnL: decimal.NewFromInt(100),
	}
	positionRepo.Create(context.Background(), localPosition)

	reconciler := NewReconciler(connector, orderRepo, positionRepo, publisher, nil)

	ctx := context.Background()
	result, err := reconciler.ReconcilePositions(ctx)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.PositionsUpdated != 1 {
		t.Errorf("expected 1 position updated, got %d", result.PositionsUpdated)
	}

	// Check position was updated to match exchange
	updatedPosition, _ := positionRepo.GetByID(ctx, "pos_1")
	if !updatedPosition.Quantity.Equal(exchangePosition.Quantity) {
		t.Errorf("expected quantity %s, got %s", exchangePosition.Quantity.String(), updatedPosition.Quantity.String())
	}

	if !updatedPosition.CurrentPrice.Equal(exchangePosition.CurrentPrice) {
		t.Errorf("expected price %s, got %s", exchangePosition.CurrentPrice.String(), updatedPosition.CurrentPrice.String())
	}
}

func TestReconciler_ReconcileAll(t *testing.T) {
	connector := &mockExchangeConnector{
		getOpenOrdersFunc: func(ctx context.Context) ([]*entity.Order, error) {
			return []*entity.Order{}, nil
		},
		getPositionsFunc: func(ctx context.Context) ([]*entity.Position, error) {
			return []*entity.Position{}, nil
		},
	}

	orderRepo := newMockOrderRepository()
	positionRepo := newMockPositionRepository()
	publisher := events.NewPublisher()

	reconciler := NewReconciler(connector, orderRepo, positionRepo, publisher, nil)

	ctx := context.Background()
	err := reconciler.ReconcileAll(ctx)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Check last reconcile time was updated
	lastReconcile := reconciler.LastReconcileAt()
	if lastReconcile.IsZero() {
		t.Error("expected last reconcile time to be set")
	}
}

func TestReconciler_ReconcileOrders_ConnectorError(t *testing.T) {
	connector := &mockExchangeConnector{
		getOpenOrdersFunc: func(ctx context.Context) ([]*entity.Order, error) {
			return nil, fmt.Errorf("connection error")
		},
	}

	orderRepo := newMockOrderRepository()
	positionRepo := newMockPositionRepository()
	publisher := events.NewPublisher()

	reconciler := NewReconciler(connector, orderRepo, positionRepo, publisher, nil)

	ctx := context.Background()
	_, err := reconciler.ReconcileOrders(ctx)

	if err == nil {
		t.Fatal("expected error when connector fails")
	}
}

func TestReconciler_ReconcilePositions_ConnectorError(t *testing.T) {
	connector := &mockExchangeConnector{
		getPositionsFunc: func(ctx context.Context) ([]*entity.Position, error) {
			return nil, fmt.Errorf("connection error")
		},
	}

	orderRepo := newMockOrderRepository()
	positionRepo := newMockPositionRepository()
	publisher := events.NewPublisher()

	reconciler := NewReconciler(connector, orderRepo, positionRepo, publisher, nil)

	ctx := context.Background()
	_, err := reconciler.ReconcilePositions(ctx)

	if err == nil {
		t.Fatal("expected error when connector fails")
	}
}

func TestReconciler_LastReconcileAt(t *testing.T) {
	connector := &mockExchangeConnector{}
	orderRepo := newMockOrderRepository()
	positionRepo := newMockPositionRepository()
	publisher := events.NewPublisher()

	reconciler := NewReconciler(connector, orderRepo, positionRepo, publisher, nil)

	// Initially should be zero
	lastReconcile := reconciler.LastReconcileAt()
	if !lastReconcile.IsZero() {
		t.Error("expected last reconcile time to be zero initially")
	}

	// After reconciliation should be set
	ctx := context.Background()
	_, _ = reconciler.ReconcileOrders(ctx)

	lastReconcile = reconciler.LastReconcileAt()
	if lastReconcile.IsZero() {
		t.Error("expected last reconcile time to be set after reconciliation")
	}
}
