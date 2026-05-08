package bybit

import (
	"context"
	"testing"
	"time"

	"github.com/kairos-development/kairos-agent/internal/domain/entity"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	connectorpkg "github.com/kairos-development/kairos-contracts/connector"
)

// MockBybitConnector mocks the Bybit connector for testing
type MockBybitConnector struct {
	name              string
	connected         bool
	connectErr        error
	disconnectErr     error
	submitOrderResult string
	submitOrderErr    error
	cancelOrderErr    error
	openOrders        []*connectorpkg.Order
	openOrdersErr     error
	queryOrderResult  *connectorpkg.Order
	queryOrderErr     error
	position          *connectorpkg.Position
	positionErr       error
	balance           *connectorpkg.AccountBalance
	balanceErr        error
	symbol            *connectorpkg.Symbol
	symbolErr         error
	refreshSymbolsErr error
	permissions       *connectorpkg.Permissions
	permissionsErr    error
	ordersChan        chan *connectorpkg.OrderUpdate
	ordersErr         error
	positionsChan     chan *connectorpkg.PositionUpdate
	positionsErr      error
	balanceChan       chan *connectorpkg.BalanceUpdate
	balanceSubErr     error
	tickerChan        chan *connectorpkg.TickerUpdate
	tickerErr         error
}

func (m *MockBybitConnector) Name() string {
	return m.name
}

func (m *MockBybitConnector) Connect(ctx context.Context) error {
	if m.connectErr != nil {
		return m.connectErr
	}
	m.connected = true
	return nil
}

func (m *MockBybitConnector) Disconnect(ctx context.Context) error {
	if m.disconnectErr != nil {
		return m.disconnectErr
	}
	m.connected = false
	return nil
}

func (m *MockBybitConnector) IsConnected() bool {
	return m.connected
}

func (m *MockBybitConnector) SubmitOrder(ctx context.Context, order *connectorpkg.Order) (string, error) {
	return m.submitOrderResult, m.submitOrderErr
}

func (m *MockBybitConnector) CancelOrder(ctx context.Context, orderID string) error {
	return m.cancelOrderErr
}

func (m *MockBybitConnector) GetOpenOrders(ctx context.Context) ([]*connectorpkg.Order, error) {
	return m.openOrders, m.openOrdersErr
}

func (m *MockBybitConnector) QueryOrder(ctx context.Context, orderID string) (*connectorpkg.Order, error) {
	return m.queryOrderResult, m.queryOrderErr
}

func (m *MockBybitConnector) GetPosition(ctx context.Context, symbol string) (*connectorpkg.Position, error) {
	return m.position, m.positionErr
}

func (m *MockBybitConnector) GetBalance(ctx context.Context) (*connectorpkg.AccountBalance, error) {
	return m.balance, m.balanceErr
}

func (m *MockBybitConnector) GetSymbol(ctx context.Context, symbol string) (*connectorpkg.Symbol, error) {
	return m.symbol, m.symbolErr
}

func (m *MockBybitConnector) RefreshSymbols(ctx context.Context) error {
	return m.refreshSymbolsErr
}

func (m *MockBybitConnector) CheckPermissions(ctx context.Context) (*connectorpkg.Permissions, error) {
	return m.permissions, m.permissionsErr
}

func (m *MockBybitConnector) SubscribeOrders(ctx context.Context) (<-chan *connectorpkg.OrderUpdate, error) {
	return m.ordersChan, m.ordersErr
}

func (m *MockBybitConnector) SubscribePositions(ctx context.Context) (<-chan *connectorpkg.PositionUpdate, error) {
	return m.positionsChan, m.positionsErr
}

func (m *MockBybitConnector) SubscribeBalance(ctx context.Context) (<-chan *connectorpkg.BalanceUpdate, error) {
	return m.balanceChan, m.balanceSubErr
}

func (m *MockBybitConnector) SubscribeTicker(ctx context.Context, symbol string) (<-chan *connectorpkg.TickerUpdate, error) {
	return m.tickerChan, m.tickerErr
}

func TestNewAdapter(t *testing.T) {
	mockConn := &MockBybitConnector{}
	adapter := NewAdapter(mockConn)

	require.NotNil(t, adapter)
	assert.Equal(t, mockConn, adapter.conn)
}

func TestAdapter_Name(t *testing.T) {
	mock := &MockBybitConnector{name: "bybit"}
	adapter := NewAdapter(mock)
	assert.Equal(t, "bybit", adapter.Name())
}

func TestAdapter_Connect(t *testing.T) {
	tests := []struct {
		name       string
		connectErr error
		wantErr    bool
	}{
		{
			name:       "successful connect",
			connectErr: nil,
			wantErr:    false,
		},
		{
			name:       "connect error",
			connectErr: assert.AnError,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockBybitConnector{connectErr: tt.connectErr}
			adapter := NewAdapter(mock)
			ctx := context.Background()

			err := adapter.Connect(ctx)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.True(t, adapter.IsConnected())
			}
		})
	}
}

func TestAdapter_Disconnect(t *testing.T) {
	tests := []struct {
		name          string
		disconnectErr error
		wantErr       bool
	}{
		{
			name:          "successful disconnect",
			disconnectErr: nil,
			wantErr:       false,
		},
		{
			name:          "disconnect error",
			disconnectErr: assert.AnError,
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockBybitConnector{
				connected:     true,
				disconnectErr: tt.disconnectErr,
			}
			adapter := NewAdapter(mock)
			ctx := context.Background()

			err := adapter.Disconnect(ctx)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.False(t, adapter.IsConnected())
			}
		})
	}
}

func TestAdapter_IsConnected(t *testing.T) {
	tests := []struct {
		name      string
		connected bool
		expected  bool
	}{
		{
			name:      "connected",
			connected: true,
			expected:  true,
		},
		{
			name:      "disconnected",
			connected: false,
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockBybitConnector{connected: tt.connected}
			adapter := NewAdapter(mock)
			assert.Equal(t, tt.expected, adapter.IsConnected())
		})
	}
}

func TestMapOrderSide(t *testing.T) {
	tests := []struct {
		name     string
		input    entity.OrderSide
		expected connectorpkg.OrderSide
	}{
		{
			name:     "buy side",
			input:    entity.OrderSideBuy,
			expected: connectorpkg.OrderSideBuy,
		},
		{
			name:     "sell side",
			input:    entity.OrderSideSell,
			expected: connectorpkg.OrderSideSell,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapOrderSide(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMapOrderType(t *testing.T) {
	tests := []struct {
		name     string
		input    entity.OrderType
		expected connectorpkg.OrderType
	}{
		{
			name:     "limit order",
			input:    entity.OrderTypeLimit,
			expected: connectorpkg.OrderTypeLimit,
		},
		{
			name:     "market order",
			input:    entity.OrderTypeMarket,
			expected: connectorpkg.OrderTypeMarket,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapOrderType(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMapOrderStatus(t *testing.T) {
	tests := []struct {
		name     string
		input    entity.OrderStatus
		expected connectorpkg.OrderStatus
	}{
		{
			name:     "pending",
			input:    entity.OrderStatusPending,
			expected: connectorpkg.OrderStatusPending,
		},
		{
			name:     "submitted",
			input:    entity.OrderStatusSubmitted,
			expected: connectorpkg.OrderStatusSubmitted,
		},
		{
			name:     "partial",
			input:    entity.OrderStatusPartial,
			expected: connectorpkg.OrderStatusPartial,
		},
		{
			name:     "filled",
			input:    entity.OrderStatusFilled,
			expected: connectorpkg.OrderStatusFilled,
		},
		{
			name:     "canceled",
			input:    entity.OrderStatusCanceled,
			expected: connectorpkg.OrderStatusCanceled,
		},
		{
			name:     "rejected",
			input:    entity.OrderStatusRejected,
			expected: connectorpkg.OrderStatusRejected,
		},
		{
			name:     "unknown defaults to pending",
			input:    entity.OrderStatus("unknown"),
			expected: connectorpkg.OrderStatusPending,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapOrderStatus(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMapTimeInForce(t *testing.T) {
	tests := []struct {
		name     string
		input    entity.TimeInForce
		expected connectorpkg.TimeInForce
	}{
		{
			name:     "GTC",
			input:    entity.TimeInForceGTC,
			expected: connectorpkg.TimeInForceGTC,
		},
		{
			name:     "IOC",
			input:    entity.TimeInForceIOC,
			expected: connectorpkg.TimeInForceIOC,
		},
		{
			name:     "FOK",
			input:    entity.TimeInForceFOK,
			expected: connectorpkg.TimeInForceFOK,
		},
		{
			name:     "unknown defaults to GTC",
			input:    entity.TimeInForce("unknown"),
			expected: connectorpkg.TimeInForceGTC,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapTimeInForce(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMapSymbolStatus(t *testing.T) {
	tests := []struct {
		name     string
		input    connectorpkg.SymbolStatus
		expected entity.SymbolStatus
	}{
		{
			name:     "trading",
			input:    connectorpkg.SymbolStatusTrading,
			expected: entity.SymbolStatusTrading,
		},
		{
			name:     "suspended",
			input:    connectorpkg.SymbolStatusSuspended,
			expected: entity.SymbolStatusSuspended,
		},
		{
			name:     "maintenance",
			input:    connectorpkg.SymbolStatusMaintenance,
			expected: entity.SymbolStatusMaintenance,
		},
		{
			name:     "unknown defaults to suspended",
			input:    connectorpkg.SymbolStatus("unknown"),
			expected: entity.SymbolStatusSuspended,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapSymbolStatus(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestUnmapOrderSideFromConnector(t *testing.T) {
	tests := []struct {
		name     string
		input    connectorpkg.OrderSide
		expected entity.OrderSide
	}{
		{
			name:     "buy side",
			input:    connectorpkg.OrderSideBuy,
			expected: entity.OrderSideBuy,
		},
		{
			name:     "sell side",
			input:    connectorpkg.OrderSideSell,
			expected: entity.OrderSideSell,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := unmapOrderSideFromConnector(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestUnmapOrderTypeFromConnector(t *testing.T) {
	tests := []struct {
		name     string
		input    connectorpkg.OrderType
		expected entity.OrderType
	}{
		{
			name:     "limit order",
			input:    connectorpkg.OrderTypeLimit,
			expected: entity.OrderTypeLimit,
		},
		{
			name:     "market order",
			input:    connectorpkg.OrderTypeMarket,
			expected: entity.OrderTypeMarket,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := unmapOrderTypeFromConnector(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestUnmapOrderStatusFromConnector(t *testing.T) {
	tests := []struct {
		name     string
		input    connectorpkg.OrderStatus
		expected entity.OrderStatus
	}{
		{
			name:     "pending",
			input:    connectorpkg.OrderStatusPending,
			expected: entity.OrderStatusPending,
		},
		{
			name:     "submitted",
			input:    connectorpkg.OrderStatusSubmitted,
			expected: entity.OrderStatusSubmitted,
		},
		{
			name:     "partial",
			input:    connectorpkg.OrderStatusPartial,
			expected: entity.OrderStatusPartial,
		},
		{
			name:     "filled",
			input:    connectorpkg.OrderStatusFilled,
			expected: entity.OrderStatusFilled,
		},
		{
			name:     "canceled",
			input:    connectorpkg.OrderStatusCanceled,
			expected: entity.OrderStatusCanceled,
		},
		{
			name:     "rejected",
			input:    connectorpkg.OrderStatusRejected,
			expected: entity.OrderStatusRejected,
		},
		{
			name:     "unknown defaults to pending",
			input:    connectorpkg.OrderStatus("unknown"),
			expected: entity.OrderStatusPending,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := unmapOrderStatusFromConnector(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestUnmapTimeInForceFromConnector(t *testing.T) {
	tests := []struct {
		name     string
		input    connectorpkg.TimeInForce
		expected entity.TimeInForce
	}{
		{
			name:     "GTC",
			input:    connectorpkg.TimeInForceGTC,
			expected: entity.TimeInForceGTC,
		},
		{
			name:     "IOC",
			input:    connectorpkg.TimeInForceIOC,
			expected: entity.TimeInForceIOC,
		},
		{
			name:     "FOK",
			input:    connectorpkg.TimeInForceFOK,
			expected: entity.TimeInForceFOK,
		},
		{
			name:     "unknown defaults to GTC",
			input:    connectorpkg.TimeInForce("unknown"),
			expected: entity.TimeInForceGTC,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := unmapTimeInForceFromConnector(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestUnmapPositionSideFromConnector(t *testing.T) {
	tests := []struct {
		name     string
		input    connectorpkg.PositionSide
		expected entity.PositionSide
	}{
		{
			name:     "long",
			input:    connectorpkg.PositionSideLong,
			expected: entity.PositionSideLong,
		},
		{
			name:     "short",
			input:    connectorpkg.PositionSideShort,
			expected: entity.PositionSideShort,
		},
		{
			name:     "unknown defaults to flat",
			input:    connectorpkg.PositionSide("unknown"),
			expected: entity.PositionSideFlat,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := unmapPositionSideFromConnector(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestUnmapOrder(t *testing.T) {
	now := time.Now().UTC()
	connOrder := &connectorpkg.Order{
		ID:              "order1",
		ClientOrderID:   "client1",
		ExchangeOrderID: "exchange1",
		StrategyID:      "strategy1",
		Symbol:          "BTCUSDT",
		Side:            connectorpkg.OrderSideBuy,
		Type:            connectorpkg.OrderTypeLimit,
		Status:          connectorpkg.OrderStatusSubmitted,
		TimeInForce:     connectorpkg.TimeInForceGTC,
		Quantity:        decimal.NewFromFloat(0.1),
		Price:           decimal.NewFromInt(50000),
		FilledQty:       decimal.Zero,
		RemainingQty:    decimal.NewFromFloat(0.1),
		AvgFillPrice:    decimal.Zero,
		CreatedAtUTC:    now,
		UpdatedAtUTC:    now,
		SubmittedAtUTC:  &now,
	}

	result := unmapOrder(connOrder)

	assert.Equal(t, "order1", result.ID)
	assert.Equal(t, "client1", result.ClientOrderID)
	assert.Equal(t, "exchange1", result.ExchangeOrderID)
	assert.Equal(t, "strategy1", result.StrategyID)
	assert.Equal(t, "BTCUSDT", result.Symbol)
	assert.Equal(t, entity.OrderSideBuy, result.Side)
	assert.Equal(t, entity.OrderTypeLimit, result.Type)
	assert.Equal(t, entity.OrderStatusSubmitted, result.Status)
	assert.Equal(t, entity.TimeInForceGTC, result.TimeInForce)
	assert.True(t, result.Quantity.Equal(decimal.NewFromFloat(0.1)))
	assert.True(t, result.Price.Equal(decimal.NewFromInt(50000)))
}

func TestUnmapPosition(t *testing.T) {
	connPos := &connectorpkg.Position{
		Symbol:        "BTCUSDT",
		Side:          connectorpkg.PositionSideLong,
		Quantity:      decimal.NewFromFloat(0.5),
		EntryPrice:    decimal.NewFromInt(50000),
		CurrentPrice:  decimal.NewFromInt(51000),
		UnrealizedPnL: decimal.NewFromInt(500),
		RealizedPnL:   decimal.NewFromInt(100),
	}

	result := unmapPosition(connPos)

	assert.Equal(t, "BTCUSDT", result.Symbol)
	assert.Equal(t, entity.PositionSideLong, result.Side)
	assert.True(t, result.Quantity.Equal(decimal.NewFromFloat(0.5)))
	assert.True(t, result.EntryPrice.Equal(decimal.NewFromInt(50000)))
	assert.True(t, result.CurrentPrice.Equal(decimal.NewFromInt(51000)))
	assert.True(t, result.UnrealizedPnL.Equal(decimal.NewFromInt(500)))
	assert.True(t, result.RealizedPnL.Equal(decimal.NewFromInt(100)))
}
func TestAdapter_SubmitOrder(t *testing.T) {
	mock := &MockBybitConnector{submitOrderResult: "ext-order-1", submitOrderErr: nil}
	adapter := NewAdapter(mock)
	order := &entity.Order{ID: "1", ClientOrderID: "c1", Symbol: "BTCUSDT", Side: entity.OrderSideBuy, Type: entity.OrderTypeLimit}
	id, err := adapter.SubmitOrder(context.Background(), order)
	assert.NoError(t, err)
	assert.Equal(t, "ext-order-1", id)
}
func TestAdapter_CancelOrder(t *testing.T) {
	mock := &MockBybitConnector{cancelOrderErr: nil}
	adapter := NewAdapter(mock)
	err := adapter.CancelOrder(context.Background(), "ext-1")
	assert.NoError(t, err)
}
func TestAdapter_GetOpenOrders(t *testing.T) {
	now := time.Now()
	mock := &MockBybitConnector{openOrders: []*connectorpkg.Order{{ID: "1", Symbol: "BTCUSDT", CreatedAtUTC: now}}}
	adapter := NewAdapter(mock)
	orders, err := adapter.GetOpenOrders(context.Background())
	assert.NoError(t, err)
	assert.Len(t, orders, 1)
	assert.Equal(t, "1", orders[0].ID)
}
func TestAdapter_GetOpenOrdersErr(t *testing.T) {
	mock := &MockBybitConnector{openOrdersErr: assert.AnError}
	adapter := NewAdapter(mock)
	_, err := adapter.GetOpenOrders(context.Background())
	assert.Error(t, err)
}
func TestAdapter_QueryOrder(t *testing.T) {
	now := time.Now()
	mock := &MockBybitConnector{queryOrderResult: &connectorpkg.Order{ID: "1", Symbol: "ETHUSDT", CreatedAtUTC: now}}
	adapter := NewAdapter(mock)
	order, err := adapter.QueryOrder(context.Background(), "ext-1")
	assert.NoError(t, err)
	assert.Equal(t, "1", order.ID)
}
func TestAdapter_QueryOrderErr(t *testing.T) {
	mock := &MockBybitConnector{queryOrderErr: assert.AnError}
	adapter := NewAdapter(mock)
	_, err := adapter.QueryOrder(context.Background(), "ext-1")
	assert.Error(t, err)
}
func TestAdapter_GetPosition(t *testing.T) {
	mock := &MockBybitConnector{position: &connectorpkg.Position{Symbol: "BTCUSDT", Side: connectorpkg.PositionSideLong}}
	adapter := NewAdapter(mock)
	pos, err := adapter.GetPosition(context.Background(), "BTCUSDT")
	assert.NoError(t, err)
	assert.Equal(t, entity.PositionSideLong, pos.Side)
}
func TestAdapter_GetPositionErr(t *testing.T) {
	mock := &MockBybitConnector{positionErr: assert.AnError}
	adapter := NewAdapter(mock)
	_, err := adapter.GetPosition(context.Background(), "BTCUSDT")
	assert.Error(t, err)
}
func TestAdapter_GetBalance(t *testing.T) {
	mock := &MockBybitConnector{
		balance: &connectorpkg.AccountBalance{
			Balances: []connectorpkg.Balance{{Asset: "USDT", Total: decimal.NewFromInt(100)}},
		},
	}
	adapter := NewAdapter(mock)
	bal, err := adapter.GetBalance(context.Background())
	assert.NoError(t, err)
	assert.Len(t, bal.Balances, 1)
	assert.Equal(t, "USDT", bal.Balances[0].Asset)
}

func TestAdapter_GetBalanceErr(t *testing.T) {
	mock := &MockBybitConnector{
		balanceErr: assert.AnError,
	}
	adapter := NewAdapter(mock)
	_, err := adapter.GetBalance(context.Background())
	assert.Error(t, err)
}

func TestAdapter_GetSymbol(t *testing.T) {
	mock := &MockBybitConnector{
		symbol: &connectorpkg.Symbol{Name: "BTCUSDT", BaseCurrency: "BTC"},
	}
	adapter := NewAdapter(mock)
	sym, err := adapter.GetSymbol(context.Background(), "BTCUSDT")
	assert.NoError(t, err)
	assert.Equal(t, "BTC", sym.BaseCurrency)
}

func TestAdapter_GetSymbolErr(t *testing.T) {
	mock := &MockBybitConnector{
		symbolErr: assert.AnError,
	}
	adapter := NewAdapter(mock)
	_, err := adapter.GetSymbol(context.Background(), "BTCUSDT")
	assert.Error(t, err)
}

func TestAdapter_CheckPermissionsErr(t *testing.T) {
	mock := &MockBybitConnector{
		permissionsErr: assert.AnError,
	}
	adapter := NewAdapter(mock)
	_, err := adapter.CheckPermissions(context.Background())
	assert.Error(t, err)
}

func TestAdapter_CheckPermissions(t *testing.T) {
	mock := &MockBybitConnector{
		permissions: &connectorpkg.Permissions{
			CanRead:     true,
			CanTrade:    true,
			HasWithdraw: false,
			HasTransfer: false,
		},
	}
	adapter := NewAdapter(mock)
	ctx := context.Background()

	perms, err := adapter.CheckPermissions(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, perms)
	assert.True(t, perms.CanRead)
	assert.True(t, perms.CanTrade)
	assert.False(t, perms.HasWithdraw)
}

func TestAdapter_SubscribeOrders(t *testing.T) {
	ch := make(chan *connectorpkg.OrderUpdate, 1)
	ch <- &connectorpkg.OrderUpdate{OrderID: "1"}
	close(ch)
	mock := &MockBybitConnector{ordersChan: ch}
	adapter := NewAdapter(mock)

	out, err := adapter.SubscribeOrders(context.Background())
	assert.NoError(t, err)
	res := <-out
	assert.Equal(t, "1", res.OrderID)
}

func TestAdapter_SubscribeOrdersErr(t *testing.T) {
	mock := &MockBybitConnector{ordersErr: assert.AnError}
	adapter := NewAdapter(mock)

	_, err := adapter.SubscribeOrders(context.Background())
	assert.Error(t, err)
}

func TestAdapter_SubscribePositions(t *testing.T) {
	ch := make(chan *connectorpkg.PositionUpdate, 1)
	ch <- &connectorpkg.PositionUpdate{Symbol: "BTCUSDT"}
	close(ch)
	mock := &MockBybitConnector{positionsChan: ch}
	adapter := NewAdapter(mock)

	out, err := adapter.SubscribePositions(context.Background())
	assert.NoError(t, err)
	res := <-out
	assert.Equal(t, "BTCUSDT", res.Symbol)
}

func TestAdapter_SubscribePositionsErr(t *testing.T) {
	mock := &MockBybitConnector{positionsErr: assert.AnError}
	adapter := NewAdapter(mock)

	_, err := adapter.SubscribePositions(context.Background())
	assert.Error(t, err)
}

func TestAdapter_SubscribeBalance(t *testing.T) {
	ch := make(chan *connectorpkg.BalanceUpdate, 1)
	ch <- &connectorpkg.BalanceUpdate{Asset: "USDT"}
	close(ch)
	mock := &MockBybitConnector{balanceChan: ch}
	adapter := NewAdapter(mock)

	out, err := adapter.SubscribeBalance(context.Background())
	assert.NoError(t, err)
	res := <-out
	assert.Equal(t, "USDT", res.Asset)
}

func TestAdapter_SubscribeBalanceErr(t *testing.T) {
	mock := &MockBybitConnector{balanceSubErr: assert.AnError}
	adapter := NewAdapter(mock)

	_, err := adapter.SubscribeBalance(context.Background())
	assert.Error(t, err)
}

func TestAdapter_SubscribeTicker(t *testing.T) {
	ch := make(chan *connectorpkg.TickerUpdate, 1)
	ch <- &connectorpkg.TickerUpdate{Symbol: "BTCUSDT"}
	close(ch)
	mock := &MockBybitConnector{tickerChan: ch}
	adapter := NewAdapter(mock)

	out, err := adapter.SubscribeTicker(context.Background(), "BTCUSDT")
	assert.NoError(t, err)
	res := <-out
	assert.Equal(t, "BTCUSDT", res.Symbol)
}

func TestAdapter_SubscribeTickerErr(t *testing.T) {
	mock := &MockBybitConnector{tickerErr: assert.AnError}
	adapter := NewAdapter(mock)

	_, err := adapter.SubscribeTicker(context.Background(), "BTCUSDT")
	assert.Error(t, err)
}

func TestAdapter_RefreshSymbols(t *testing.T) {
	mock := &MockBybitConnector{}
	adapter := NewAdapter(mock)

	err := adapter.RefreshSymbols(context.Background())
	assert.NoError(t, err)

	mock.refreshSymbolsErr = assert.AnError
	err = adapter.RefreshSymbols(context.Background())
	assert.Error(t, err)
}
