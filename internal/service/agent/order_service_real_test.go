package agent

import (
	"context"
	"errors"
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
type mockOrderRepo struct {
	createFunc     func(ctx context.Context, order *entity.Order) error
	updateFunc     func(ctx context.Context, order *entity.Order) error
	getByIDFunc    func(ctx context.Context, id string) (*entity.Order, error)
	listByStrategy func(ctx context.Context, strategyID string, limit, offset int) ([]*entity.Order, error)
	listActiveFunc func(ctx context.Context) ([]*entity.Order, error)
}

func (m *mockOrderRepo) Create(ctx context.Context, order *entity.Order) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, order)
	}
	return nil
}

func (m *mockOrderRepo) Update(ctx context.Context, order *entity.Order) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, order)
	}
	return nil
}

func (m *mockOrderRepo) GetByID(ctx context.Context, id string) (*entity.Order, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return nil, storage.ErrNotFound
}

func (m *mockOrderRepo) ListByStrategy(ctx context.Context, strategyID string, limit, offset int) ([]*entity.Order, error) {
	if m.listByStrategy != nil {
		return m.listByStrategy(ctx, strategyID, limit, offset)
	}
	return nil, nil
}

func (m *mockOrderRepo) ListActive(ctx context.Context) ([]*entity.Order, error) {
	if m.listActiveFunc != nil {
		return m.listActiveFunc(ctx)
	}
	return nil, nil
}

func (m *mockOrderRepo) Delete(ctx context.Context, id string) error {
	return nil
}

func (m *mockOrderRepo) ListByStatus(ctx context.Context, status entity.OrderStatus, limit, offset int) ([]*entity.Order, error) {
	return nil, nil
}

func (m *mockOrderRepo) GetByClientOrderID(ctx context.Context, clientOrderID string) (*entity.Order, error) {
	return nil, storage.ErrNotFound
}

func (m *mockOrderRepo) GetByExchangeOrderID(ctx context.Context, exchangeOrderID string) (*entity.Order, error) {
	return nil, storage.ErrNotFound
}

func (m *mockOrderRepo) ListInFlight(ctx context.Context, since time.Time) ([]*entity.Order, error) {
	return nil, nil
}

type mockPositionRepo struct {
	getByStrategyAndSymbolFunc func(ctx context.Context, strategyID, symbol string) (*entity.Position, error)
}

func (m *mockPositionRepo) GetByStrategyAndSymbol(ctx context.Context, strategyID, symbol string) (*entity.Position, error) {
	if m.getByStrategyAndSymbolFunc != nil {
		return m.getByStrategyAndSymbolFunc(ctx, strategyID, symbol)
	}
	return nil, storage.ErrNotFound
}

func (m *mockPositionRepo) Create(ctx context.Context, position *entity.Position) error {
	return nil
}

func (m *mockPositionRepo) Update(ctx context.Context, position *entity.Position) error {
	return nil
}

func (m *mockPositionRepo) GetByID(ctx context.Context, id string) (*entity.Position, error) {
	return nil, storage.ErrNotFound
}

func (m *mockPositionRepo) ListByStrategy(ctx context.Context, strategyID string) ([]*entity.Position, error) {
	return nil, nil
}

func (m *mockPositionRepo) ListOpen(ctx context.Context) ([]*entity.Position, error) {
	return nil, nil
}

func (m *mockPositionRepo) Delete(ctx context.Context, id string) error {
	return nil
}

type mockStrategyRepo struct {
	getByIDFunc func(ctx context.Context, id string) (*entity.Strategy, error)
}

func (m *mockStrategyRepo) GetByID(ctx context.Context, id string) (*entity.Strategy, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return nil, storage.ErrNotFound
}

func (m *mockStrategyRepo) Create(ctx context.Context, strategy *entity.Strategy) error {
	return nil
}

func (m *mockStrategyRepo) Update(ctx context.Context, strategy *entity.Strategy) error {
	return nil
}

func (m *mockStrategyRepo) List(ctx context.Context, limit, offset int) ([]*entity.Strategy, error) {
	return nil, nil
}

func (m *mockStrategyRepo) Delete(ctx context.Context, id string) error {
	return nil
}

func (m *mockStrategyRepo) ListActive(ctx context.Context) ([]*entity.Strategy, error) {
	return nil, nil
}

type mockBalanceRepo struct {
	getLatestFunc func(ctx context.Context) (*entity.AccountBalance, error)
}

func (m *mockBalanceRepo) GetLatest(ctx context.Context) (*entity.AccountBalance, error) {
	if m.getLatestFunc != nil {
		return m.getLatestFunc(ctx)
	}
	return nil, nil
}

func (m *mockBalanceRepo) Save(ctx context.Context, balance *entity.AccountBalance) error {
	return nil
}

func (m *mockBalanceRepo) GetHistory(ctx context.Context, start, end time.Time) ([]*entity.AccountBalance, error) {
	return nil, nil
}

type mockSymbolRepo struct {
	getByNameFunc func(ctx context.Context, name string) (*entity.Symbol, error)
}

func (m *mockSymbolRepo) GetByName(ctx context.Context, name string) (*entity.Symbol, error) {
	if m.getByNameFunc != nil {
		return m.getByNameFunc(ctx, name)
	}
	return nil, storage.ErrNotFound
}

func (m *mockSymbolRepo) Save(ctx context.Context, symbol *entity.Symbol) error {
	return nil
}

func (m *mockSymbolRepo) List(ctx context.Context) ([]*entity.Symbol, error) {
	return nil, nil
}

func (m *mockSymbolRepo) Delete(ctx context.Context, name string) error {
	return nil
}

func (m *mockSymbolRepo) ListByStatus(ctx context.Context, status entity.SymbolStatus) ([]*entity.Symbol, error) {
	return nil, nil
}

func (m *mockSymbolRepo) Upsert(ctx context.Context, symbol *entity.Symbol) error {
	return nil
}

func TestOrderService_NewOrderService(t *testing.T) {
	svc := NewOrderService(
		&mockOrderRepo{},
		&mockPositionRepo{},
		&mockStrategyRepo{},
		&mockBalanceRepo{},
		&mockSymbolRepo{},
		risk.NewEngine(nil),
		&ConnectorAdapter{},
		nil,
	)

	assert.NotNil(t, svc)
}

func TestOrderService_CreateOrder_Success(t *testing.T) {
	strategy := &entity.Strategy{
		ID:     "strategy-1",
		Status: entity.StrategyStatusActive,
	}

	symbol := &entity.Symbol{
		Name:          "BTCUSDT",
		QuoteCurrency: "USDT",
		Status:        entity.SymbolStatusTrading,
		MinOrderQty:   decimal.NewFromFloat(0.001),
		MaxOrderQty:   decimal.NewFromInt(1000),
		MinPrice:      decimal.NewFromInt(1),
		MaxPrice:      decimal.NewFromInt(100000),
		TickSize:      decimal.NewFromFloat(0.01),
		StepSize:      decimal.NewFromFloat(0.001),
		MinNotional:   decimal.NewFromInt(10),
	}

	balance := &entity.AccountBalance{
		Balances: []entity.Balance{
			{
				Asset:     "USDT",
				Total:     decimal.NewFromInt(10000),
				Available: decimal.NewFromInt(10000),
			},
		},
	}

	var createdOrder *entity.Order

	svc := NewOrderService(
		&mockOrderRepo{
			createFunc: func(ctx context.Context, order *entity.Order) error {
				createdOrder = order
				return nil
			},
		},
		&mockPositionRepo{},
		&mockStrategyRepo{
			getByIDFunc: func(ctx context.Context, id string) (*entity.Strategy, error) {
				return strategy, nil
			},
		},
		&mockBalanceRepo{
			getLatestFunc: func(ctx context.Context) (*entity.AccountBalance, error) {
				return balance, nil
			},
		},
		&mockSymbolRepo{
			getByNameFunc: func(ctx context.Context, name string) (*entity.Symbol, error) {
				return symbol, nil
			},
		},
		risk.NewEngine(nil),
		&ConnectorAdapter{},
		nil,
	)

	req := CreateOrderRequest{
		StrategyID:  "strategy-1",
		Symbol:      "BTCUSDT",
		Side:        entity.OrderSideBuy,
		Type:        entity.OrderTypeLimit,
		Quantity:    decimal.NewFromFloat(0.1),
		Price:       decimal.NewFromInt(50000),
		TimeInForce: entity.TimeInForceGTC,
	}

	order, err := svc.CreateOrder(context.Background(), req)
	require.NoError(t, err)
	assert.NotNil(t, order)
	assert.Equal(t, "strategy-1", order.StrategyID)
	assert.Equal(t, "BTCUSDT", order.Symbol)
	assert.Equal(t, entity.OrderStatusPending, order.Status)
	assert.NotNil(t, createdOrder)
}

func TestOrderService_CreateOrder_StrategyNotFound(t *testing.T) {
	svc := NewOrderService(
		&mockOrderRepo{},
		&mockPositionRepo{},
		&mockStrategyRepo{
			getByIDFunc: func(ctx context.Context, id string) (*entity.Strategy, error) {
				return nil, storage.ErrNotFound
			},
		},
		&mockBalanceRepo{},
		&mockSymbolRepo{},
		risk.NewEngine(nil),
		&ConnectorAdapter{},
		nil,
	)

	req := CreateOrderRequest{
		StrategyID: "nonexistent",
		Symbol:     "BTCUSDT",
	}

	_, err := svc.CreateOrder(context.Background(), req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get strategy")
}

func TestOrderService_CreateOrder_StrategyCannotTrade(t *testing.T) {
	strategy := &entity.Strategy{
		ID:     "strategy-1",
		Status: entity.StrategyStatusStopped,
	}

	svc := NewOrderService(
		&mockOrderRepo{},
		&mockPositionRepo{},
		&mockStrategyRepo{
			getByIDFunc: func(ctx context.Context, id string) (*entity.Strategy, error) {
				return strategy, nil
			},
		},
		&mockBalanceRepo{},
		&mockSymbolRepo{},
		risk.NewEngine(nil),
		&ConnectorAdapter{},
		nil,
	)

	req := CreateOrderRequest{
		StrategyID: "strategy-1",
		Symbol:     "BTCUSDT",
	}

	_, err := svc.CreateOrder(context.Background(), req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "strategy cannot trade")
}

func TestOrderService_CreateOrder_SymbolNotFound(t *testing.T) {
	strategy := &entity.Strategy{
		ID:     "strategy-1",
		Status: entity.StrategyStatusActive,
	}

	svc := NewOrderService(
		&mockOrderRepo{},
		&mockPositionRepo{},
		&mockStrategyRepo{
			getByIDFunc: func(ctx context.Context, id string) (*entity.Strategy, error) {
				return strategy, nil
			},
		},
		&mockBalanceRepo{},
		&mockSymbolRepo{
			getByNameFunc: func(ctx context.Context, name string) (*entity.Symbol, error) {
				return nil, storage.ErrNotFound
			},
		},
		risk.NewEngine(nil),
		&ConnectorAdapter{},
		nil,
	)

	req := CreateOrderRequest{
		StrategyID: "strategy-1",
		Symbol:     "INVALID",
	}

	_, err := svc.CreateOrder(context.Background(), req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get symbol")
}

func TestOrderService_SubmitOrder_Success(t *testing.T) {
	order := &entity.Order{
		ID:            "order-1",
		ClientOrderID: "client-1",
		Status:        entity.OrderStatusPending,
	}

	var updatedOrder *entity.Order

	conn := &mockConnector{
		submitOrderFunc: func(ctx context.Context, o *entity.Order) (string, error) {
			return "exchange-123", nil
		},
	}

	svc := NewOrderService(
		&mockOrderRepo{
			getByIDFunc: func(ctx context.Context, id string) (*entity.Order, error) {
				return order, nil
			},
			updateFunc: func(ctx context.Context, o *entity.Order) error {
				updatedOrder = o
				return nil
			},
		},
		&mockPositionRepo{},
		&mockStrategyRepo{},
		&mockBalanceRepo{},
		&mockSymbolRepo{},
		risk.NewEngine(nil),
		NewConnectorAdapter(conn),
		nil,
	)

	err := svc.SubmitOrder(context.Background(), "order-1")
	require.NoError(t, err)
	assert.NotNil(t, updatedOrder)
	assert.Equal(t, "exchange-123", updatedOrder.ExchangeOrderID)
	assert.Equal(t, entity.OrderStatusSubmitted, updatedOrder.Status)
}

func TestOrderService_SubmitOrder_OrderNotFound(t *testing.T) {
	svc := NewOrderService(
		&mockOrderRepo{
			getByIDFunc: func(ctx context.Context, id string) (*entity.Order, error) {
				return nil, storage.ErrNotFound
			},
		},
		&mockPositionRepo{},
		&mockStrategyRepo{},
		&mockBalanceRepo{},
		&mockSymbolRepo{},
		risk.NewEngine(nil),
		&ConnectorAdapter{},
		nil,
	)

	err := svc.SubmitOrder(context.Background(), "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get order")
}

func TestOrderService_SubmitOrder_OrderNotPending(t *testing.T) {
	order := &entity.Order{
		ID:     "order-1",
		Status: entity.OrderStatusFilled,
	}

	svc := NewOrderService(
		&mockOrderRepo{
			getByIDFunc: func(ctx context.Context, id string) (*entity.Order, error) {
				return order, nil
			},
		},
		&mockPositionRepo{},
		&mockStrategyRepo{},
		&mockBalanceRepo{},
		&mockSymbolRepo{},
		risk.NewEngine(nil),
		&ConnectorAdapter{},
		nil,
	)

	err := svc.SubmitOrder(context.Background(), "order-1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "order is not pending")
}

func TestOrderService_SubmitOrder_ExchangeError(t *testing.T) {
	order := &entity.Order{
		ID:     "order-1",
		Status: entity.OrderStatusPending,
	}

	conn := &mockConnector{
		submitOrderFunc: func(ctx context.Context, o *entity.Order) (string, error) {
			return "", errors.New("exchange error")
		},
	}

	svc := NewOrderService(
		&mockOrderRepo{
			getByIDFunc: func(ctx context.Context, id string) (*entity.Order, error) {
				return order, nil
			},
			updateFunc: func(ctx context.Context, o *entity.Order) error {
				return nil
			},
		},
		&mockPositionRepo{},
		&mockStrategyRepo{},
		&mockBalanceRepo{},
		&mockSymbolRepo{},
		risk.NewEngine(nil),
		NewConnectorAdapter(conn),
		nil,
	)

	err := svc.SubmitOrder(context.Background(), "order-1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "submit order to exchange")
}

func TestOrderService_CancelOrder_Success(t *testing.T) {
	order := &entity.Order{
		ID:              "order-1",
		ExchangeOrderID: "exchange-123",
		Status:          entity.OrderStatusSubmitted,
	}

	conn := &mockConnector{
		cancelOrderFunc: func(ctx context.Context, orderID string) error {
			assert.Equal(t, "exchange-123", orderID)
			return nil
		},
	}

	svc := NewOrderService(
		&mockOrderRepo{
			getByIDFunc: func(ctx context.Context, id string) (*entity.Order, error) {
				return order, nil
			},
			updateFunc: func(ctx context.Context, o *entity.Order) error {
				assert.Equal(t, entity.OrderStatusCanceled, o.Status)
				return nil
			},
		},
		&mockPositionRepo{},
		&mockStrategyRepo{},
		&mockBalanceRepo{},
		&mockSymbolRepo{},
		risk.NewEngine(nil),
		NewConnectorAdapter(conn),
		nil,
	)

	err := svc.CancelOrder(context.Background(), "order-1")
	assert.NoError(t, err)
}

func TestOrderService_CancelOrder_OrderNotActive(t *testing.T) {
	order := &entity.Order{
		ID:     "order-1",
		Status: entity.OrderStatusFilled,
	}

	svc := NewOrderService(
		&mockOrderRepo{
			getByIDFunc: func(ctx context.Context, id string) (*entity.Order, error) {
				return order, nil
			},
		},
		&mockPositionRepo{},
		&mockStrategyRepo{},
		&mockBalanceRepo{},
		&mockSymbolRepo{},
		risk.NewEngine(nil),
		&ConnectorAdapter{},
		nil,
	)

	err := svc.CancelOrder(context.Background(), "order-1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "order is not active")
}

func TestOrderService_GetOrder_Success(t *testing.T) {
	expectedOrder := &entity.Order{
		ID:     "order-1",
		Symbol: "BTCUSDT",
	}

	svc := NewOrderService(
		&mockOrderRepo{
			getByIDFunc: func(ctx context.Context, id string) (*entity.Order, error) {
				return expectedOrder, nil
			},
		},
		&mockPositionRepo{},
		&mockStrategyRepo{},
		&mockBalanceRepo{},
		&mockSymbolRepo{},
		risk.NewEngine(nil),
		&ConnectorAdapter{},
		nil,
	)

	order, err := svc.GetOrder(context.Background(), "order-1")
	require.NoError(t, err)
	assert.Equal(t, expectedOrder, order)
}

func TestOrderService_ListOrders_Success(t *testing.T) {
	expectedOrders := []*entity.Order{
		{ID: "order-1"},
		{ID: "order-2"},
	}

	svc := NewOrderService(
		&mockOrderRepo{
			listByStrategy: func(ctx context.Context, strategyID string, limit, offset int) ([]*entity.Order, error) {
				assert.Equal(t, "strategy-1", strategyID)
				assert.Equal(t, 10, limit)
				assert.Equal(t, 0, offset)
				return expectedOrders, nil
			},
		},
		&mockPositionRepo{},
		&mockStrategyRepo{},
		&mockBalanceRepo{},
		&mockSymbolRepo{},
		risk.NewEngine(nil),
		&ConnectorAdapter{},
		nil,
	)

	orders, err := svc.ListOrders(context.Background(), "strategy-1", 10, 0)
	require.NoError(t, err)
	assert.Equal(t, expectedOrders, orders)
}

func TestOrderService_ReconcileOrders_Success(t *testing.T) {
	localOrder := &entity.Order{
		ID:              "order-1",
		ExchangeOrderID: "exchange-123",
		Status:          entity.OrderStatusSubmitted,
		FilledQty:       decimal.Zero,
	}

	exchangeOrder := &entity.Order{
		ID:           "order-1",
		Status:       entity.OrderStatusFilled,
		FilledQty:    decimal.NewFromFloat(0.1),
		RemainingQty: decimal.Zero,
		AvgFillPrice: decimal.NewFromInt(50000),
	}

	conn := &mockConnector{
		queryOrderFunc: func(ctx context.Context, orderID string) (*entity.Order, error) {
			assert.Equal(t, "exchange-123", orderID)
			return exchangeOrder, nil
		},
	}

	var updatedOrder *entity.Order

	svc := NewOrderService(
		&mockOrderRepo{
			listActiveFunc: func(ctx context.Context) ([]*entity.Order, error) {
				return []*entity.Order{localOrder}, nil
			},
			updateFunc: func(ctx context.Context, o *entity.Order) error {
				updatedOrder = o
				return nil
			},
		},
		&mockPositionRepo{},
		&mockStrategyRepo{},
		&mockBalanceRepo{},
		&mockSymbolRepo{},
		risk.NewEngine(nil),
		NewConnectorAdapter(conn),
		nil,
	)

	err := svc.ReconcileOrders(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, updatedOrder)
	assert.Equal(t, entity.OrderStatusFilled, updatedOrder.Status)
	assert.True(t, updatedOrder.FilledQty.Equal(decimal.NewFromFloat(0.1)))
}

func TestOrderService_ReconcileOrders_SkipsOrdersWithoutExchangeID(t *testing.T) {
	localOrder := &entity.Order{
		ID:              "order-1",
		ExchangeOrderID: "",
		Status:          entity.OrderStatusPending,
	}

	svc := NewOrderService(
		&mockOrderRepo{
			listActiveFunc: func(ctx context.Context) ([]*entity.Order, error) {
				return []*entity.Order{localOrder}, nil
			},
		},
		&mockPositionRepo{},
		&mockStrategyRepo{},
		&mockBalanceRepo{},
		&mockSymbolRepo{},
		risk.NewEngine(nil),
		&ConnectorAdapter{},
		nil,
	)

	err := svc.ReconcileOrders(context.Background())
	assert.NoError(t, err)
}

func TestOrderService_ReconcileOrders_ContinuesOnQueryError(t *testing.T) {
	localOrder := &entity.Order{
		ID:              "order-1",
		ExchangeOrderID: "exchange-123",
		Status:          entity.OrderStatusSubmitted,
	}

	conn := &mockConnector{
		queryOrderFunc: func(ctx context.Context, orderID string) (*entity.Order, error) {
			return nil, errors.New("query error")
		},
	}

	svc := NewOrderService(
		&mockOrderRepo{
			listActiveFunc: func(ctx context.Context) ([]*entity.Order, error) {
				return []*entity.Order{localOrder}, nil
			},
		},
		&mockPositionRepo{},
		&mockStrategyRepo{},
		&mockBalanceRepo{},
		&mockSymbolRepo{},
		risk.NewEngine(nil),
		NewConnectorAdapter(conn),
		nil,
	)

	err := svc.ReconcileOrders(context.Background())
	assert.NoError(t, err)
}

func TestOrderService_PublishOrderCreated_NilPublisher(t *testing.T) {
	svc := &orderService{
		publisher: nil,
	}

	order := &entity.Order{ID: "order-1"}
	svc.publishOrderCreated(context.Background(), order)
}

func TestOrderService_PublishOrderCreated_WithPublisher(t *testing.T) {
	publisher := events.NewPublisher()

	svc := &orderService{
		publisher: publisher,
	}

	order := &entity.Order{
		ID:            "order-1",
		ClientOrderID: "client-1",
		StrategyID:    "strategy-1",
		Symbol:        "BTCUSDT",
		Side:          entity.OrderSideBuy,
		Type:          entity.OrderTypeLimit,
		Quantity:      decimal.NewFromFloat(0.1),
		Price:         decimal.NewFromInt(50000),
	}

	svc.publishOrderCreated(context.Background(), order)
	time.Sleep(10 * time.Millisecond)
}

func TestOrderService_PublishOrderSubmitted_NilPublisher(t *testing.T) {
	svc := &orderService{
		publisher: nil,
	}

	order := &entity.Order{ID: "order-1"}
	svc.publishOrderSubmitted(context.Background(), order)
}

func TestOrderService_PublishOrderCanceled_NilPublisher(t *testing.T) {
	svc := &orderService{
		publisher: nil,
	}

	order := &entity.Order{ID: "order-1"}
	svc.publishOrderCanceled(context.Background(), order, "test")
}

func TestOrderService_PublishOrderRejected_NilPublisher(t *testing.T) {
	svc := &orderService{
		publisher: nil,
	}

	order := &entity.Order{ID: "order-1"}
	svc.publishOrderRejected(context.Background(), order, "test")
}
