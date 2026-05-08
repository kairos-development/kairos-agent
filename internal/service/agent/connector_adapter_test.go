package agent

import (
	"context"
	"errors"
	"testing"

	"github.com/kairos-development/kairos-agent/internal/domain/connector"
	"github.com/kairos-development/kairos-agent/internal/domain/entity"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockConnector struct {
	submitOrderFunc        func(ctx context.Context, order *entity.Order) (string, error)
	cancelOrderFunc        func(ctx context.Context, orderID string) error
	queryOrderFunc         func(ctx context.Context, orderID string) (*entity.Order, error)
	getPositionFunc        func(ctx context.Context, symbol string) (*entity.Position, error)
	getBalanceFunc         func(ctx context.Context) (*entity.AccountBalance, error)
	getSymbolFunc          func(ctx context.Context, symbol string) (*entity.Symbol, error)
	nameFunc               func() string
	connectFunc            func(ctx context.Context) error
	disconnectFunc         func(ctx context.Context) error
	isConnectedFunc        func() bool
	getOpenOrdersFunc      func(ctx context.Context) ([]*entity.Order, error)
	refreshSymbolsFunc     func(ctx context.Context) error
	checkPermissionsFunc   func(ctx context.Context) (*connector.Permissions, error)
	subscribeOrdersFunc    func(ctx context.Context) (<-chan *connector.OrderUpdate, error)
	subscribePositionsFunc func(ctx context.Context) (<-chan *connector.PositionUpdate, error)
	subscribeBalanceFunc   func(ctx context.Context) (<-chan *connector.BalanceUpdate, error)
	subscribeTickerFunc    func(ctx context.Context, symbol string) (<-chan *connector.TickerUpdate, error)
}

func (m *mockConnector) SubmitOrder(ctx context.Context, order *entity.Order) (string, error) {
	if m.submitOrderFunc != nil {
		return m.submitOrderFunc(ctx, order)
	}
	return "", nil
}

func (m *mockConnector) CancelOrder(ctx context.Context, orderID string) error {
	if m.cancelOrderFunc != nil {
		return m.cancelOrderFunc(ctx, orderID)
	}
	return nil
}

func (m *mockConnector) QueryOrder(ctx context.Context, orderID string) (*entity.Order, error) {
	if m.queryOrderFunc != nil {
		return m.queryOrderFunc(ctx, orderID)
	}
	return nil, nil
}

func (m *mockConnector) GetPosition(ctx context.Context, symbol string) (*entity.Position, error) {
	if m.getPositionFunc != nil {
		return m.getPositionFunc(ctx, symbol)
	}
	return nil, nil
}

func (m *mockConnector) GetBalance(ctx context.Context) (*entity.AccountBalance, error) {
	if m.getBalanceFunc != nil {
		return m.getBalanceFunc(ctx)
	}
	return nil, nil
}

func (m *mockConnector) GetSymbol(ctx context.Context, symbol string) (*entity.Symbol, error) {
	if m.getSymbolFunc != nil {
		return m.getSymbolFunc(ctx, symbol)
	}
	return nil, nil
}

func (m *mockConnector) Name() string {
	if m.nameFunc != nil {
		return m.nameFunc()
	}
	return "mock"
}

func (m *mockConnector) Connect(ctx context.Context) error {
	if m.connectFunc != nil {
		return m.connectFunc(ctx)
	}
	return nil
}

func (m *mockConnector) Disconnect(ctx context.Context) error {
	if m.disconnectFunc != nil {
		return m.disconnectFunc(ctx)
	}
	return nil
}

func (m *mockConnector) IsConnected() bool {
	if m.isConnectedFunc != nil {
		return m.isConnectedFunc()
	}
	return true
}

func (m *mockConnector) GetOpenOrders(ctx context.Context) ([]*entity.Order, error) {
	if m.getOpenOrdersFunc != nil {
		return m.getOpenOrdersFunc(ctx)
	}
	return nil, nil
}

func (m *mockConnector) RefreshSymbols(ctx context.Context) error {
	if m.refreshSymbolsFunc != nil {
		return m.refreshSymbolsFunc(ctx)
	}
	return nil
}

func (m *mockConnector) CheckPermissions(ctx context.Context) (*connector.Permissions, error) {
	if m.checkPermissionsFunc != nil {
		return m.checkPermissionsFunc(ctx)
	}
	return &connector.Permissions{}, nil
}

func (m *mockConnector) SubscribeOrders(ctx context.Context) (<-chan *connector.OrderUpdate, error) {
	if m.subscribeOrdersFunc != nil {
		return m.subscribeOrdersFunc(ctx)
	}
	ch := make(chan *connector.OrderUpdate)
	close(ch)
	return ch, nil
}

func (m *mockConnector) SubscribePositions(ctx context.Context) (<-chan *connector.PositionUpdate, error) {
	if m.subscribePositionsFunc != nil {
		return m.subscribePositionsFunc(ctx)
	}
	ch := make(chan *connector.PositionUpdate)
	close(ch)
	return ch, nil
}

func (m *mockConnector) SubscribeBalance(ctx context.Context) (<-chan *connector.BalanceUpdate, error) {
	if m.subscribeBalanceFunc != nil {
		return m.subscribeBalanceFunc(ctx)
	}
	ch := make(chan *connector.BalanceUpdate)
	close(ch)
	return ch, nil
}

func (m *mockConnector) SubscribeTicker(ctx context.Context, symbol string) (<-chan *connector.TickerUpdate, error) {
	if m.subscribeTickerFunc != nil {
		return m.subscribeTickerFunc(ctx, symbol)
	}
	ch := make(chan *connector.TickerUpdate)
	close(ch)
	return ch, nil
}

func TestNewConnectorAdapter(t *testing.T) {
	conn := &mockConnector{}
	adapter := NewConnectorAdapter(conn)

	assert.NotNil(t, adapter)
	assert.Equal(t, conn, adapter.conn)
}

func TestConnectorAdapter_SubmitOrder_Success(t *testing.T) {
	expectedOrderID := "exchange-order-123"
	conn := &mockConnector{
		submitOrderFunc: func(ctx context.Context, order *entity.Order) (string, error) {
			return expectedOrderID, nil
		},
	}
	adapter := NewConnectorAdapter(conn)

	order := &entity.Order{
		ID:     "order-123",
		Symbol: "BTCUSDT",
	}

	orderID, err := adapter.SubmitOrder(context.Background(), order)
	require.NoError(t, err)
	assert.Equal(t, expectedOrderID, orderID)
}

func TestConnectorAdapter_SubmitOrder_Error(t *testing.T) {
	expectedErr := errors.New("exchange error")
	conn := &mockConnector{
		submitOrderFunc: func(ctx context.Context, order *entity.Order) (string, error) {
			return "", expectedErr
		},
	}
	adapter := NewConnectorAdapter(conn)

	order := &entity.Order{
		ID:     "order-123",
		Symbol: "BTCUSDT",
	}

	_, err := adapter.SubmitOrder(context.Background(), order)
	assert.ErrorIs(t, err, expectedErr)
}

func TestConnectorAdapter_CancelOrder_Success(t *testing.T) {
	conn := &mockConnector{
		cancelOrderFunc: func(ctx context.Context, orderID string) error {
			assert.Equal(t, "order-123", orderID)
			return nil
		},
	}
	adapter := NewConnectorAdapter(conn)

	err := adapter.CancelOrder(context.Background(), "order-123")
	assert.NoError(t, err)
}

func TestConnectorAdapter_CancelOrder_Error(t *testing.T) {
	expectedErr := errors.New("cancel error")
	conn := &mockConnector{
		cancelOrderFunc: func(ctx context.Context, orderID string) error {
			return expectedErr
		},
	}
	adapter := NewConnectorAdapter(conn)

	err := adapter.CancelOrder(context.Background(), "order-123")
	assert.ErrorIs(t, err, expectedErr)
}

func TestConnectorAdapter_QueryOrder_Success(t *testing.T) {
	expectedOrder := &entity.Order{
		ID:     "order-123",
		Symbol: "BTCUSDT",
		Status: entity.OrderStatusFilled,
	}
	conn := &mockConnector{
		queryOrderFunc: func(ctx context.Context, orderID string) (*entity.Order, error) {
			assert.Equal(t, "order-123", orderID)
			return expectedOrder, nil
		},
	}
	adapter := NewConnectorAdapter(conn)

	order, err := adapter.QueryOrder(context.Background(), "order-123")
	require.NoError(t, err)
	assert.Equal(t, expectedOrder, order)
}

func TestConnectorAdapter_QueryOrder_Error(t *testing.T) {
	expectedErr := errors.New("query error")
	conn := &mockConnector{
		queryOrderFunc: func(ctx context.Context, orderID string) (*entity.Order, error) {
			return nil, expectedErr
		},
	}
	adapter := NewConnectorAdapter(conn)

	_, err := adapter.QueryOrder(context.Background(), "order-123")
	assert.ErrorIs(t, err, expectedErr)
}

func TestConnectorAdapter_GetPosition_Success(t *testing.T) {
	expectedPosition := &entity.Position{
		ID:         "pos-123",
		Symbol:     "BTCUSDT",
		Side:       entity.PositionSideLong,
		Quantity:   decimal.NewFromFloat(0.1),
		EntryPrice: decimal.NewFromInt(50000),
	}
	conn := &mockConnector{
		getPositionFunc: func(ctx context.Context, symbol string) (*entity.Position, error) {
			assert.Equal(t, "BTCUSDT", symbol)
			return expectedPosition, nil
		},
	}
	adapter := NewConnectorAdapter(conn)

	position, err := adapter.GetPosition(context.Background(), "BTCUSDT")
	require.NoError(t, err)
	assert.Equal(t, expectedPosition, position)
}

func TestConnectorAdapter_GetPosition_Error(t *testing.T) {
	expectedErr := errors.New("position error")
	conn := &mockConnector{
		getPositionFunc: func(ctx context.Context, symbol string) (*entity.Position, error) {
			return nil, expectedErr
		},
	}
	adapter := NewConnectorAdapter(conn)

	_, err := adapter.GetPosition(context.Background(), "BTCUSDT")
	assert.ErrorIs(t, err, expectedErr)
}

func TestConnectorAdapter_GetBalance_Success(t *testing.T) {
	expectedBalance := &entity.AccountBalance{
		Balances: []entity.Balance{
			{
				Asset:     "USDT",
				Total:     decimal.NewFromInt(10000),
				Available: decimal.NewFromInt(9500),
				Locked:    decimal.NewFromInt(500),
			},
		},
	}
	conn := &mockConnector{
		getBalanceFunc: func(ctx context.Context) (*entity.AccountBalance, error) {
			return expectedBalance, nil
		},
	}
	adapter := NewConnectorAdapter(conn)

	balance, err := adapter.GetBalance(context.Background())
	require.NoError(t, err)
	assert.Equal(t, expectedBalance, balance)
}

func TestConnectorAdapter_GetBalance_Error(t *testing.T) {
	expectedErr := errors.New("balance error")
	conn := &mockConnector{
		getBalanceFunc: func(ctx context.Context) (*entity.AccountBalance, error) {
			return nil, expectedErr
		},
	}
	adapter := NewConnectorAdapter(conn)

	_, err := adapter.GetBalance(context.Background())
	assert.ErrorIs(t, err, expectedErr)
}

func TestConnectorAdapter_GetSymbol_Success(t *testing.T) {
	expectedSymbol := &entity.Symbol{
		Name:   "BTCUSDT",
		Status: entity.SymbolStatusTrading,
	}
	conn := &mockConnector{
		getSymbolFunc: func(ctx context.Context, symbol string) (*entity.Symbol, error) {
			assert.Equal(t, "BTCUSDT", symbol)
			return expectedSymbol, nil
		},
	}
	adapter := NewConnectorAdapter(conn)

	symbol, err := adapter.GetSymbol(context.Background(), "BTCUSDT")
	require.NoError(t, err)
	assert.Equal(t, expectedSymbol, symbol)
}

func TestConnectorAdapter_GetSymbol_Error(t *testing.T) {
	expectedErr := errors.New("symbol error")
	conn := &mockConnector{
		getSymbolFunc: func(ctx context.Context, symbol string) (*entity.Symbol, error) {
			return nil, expectedErr
		},
	}
	adapter := NewConnectorAdapter(conn)

	_, err := adapter.GetSymbol(context.Background(), "BTCUSDT")
	assert.ErrorIs(t, err, expectedErr)
}
