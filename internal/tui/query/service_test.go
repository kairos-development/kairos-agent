package query

import (
	"context"
	"testing"
	"time"

	"github.com/kairos-development/kairos-agent/internal/domain/entity"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock implementations

type mockRuntimeReader struct {
	status entity.RuntimeStatus
	err    error
}

func (m *mockRuntimeReader) Status(ctx context.Context) (entity.RuntimeStatus, error) {
	return m.status, m.err
}

type mockPositionReader struct {
	positions []*entity.Position
	err       error
}

func (m *mockPositionReader) ListOpen(ctx context.Context) ([]*entity.Position, error) {
	return m.positions, m.err
}

type mockBalanceReader struct {
	balance *entity.AccountBalance
	err     error
}

func (m *mockBalanceReader) GetLatest(ctx context.Context) (*entity.AccountBalance, error) {
	return m.balance, m.err
}

type mockOrderReader struct {
	orders []*entity.Order
	err    error
}

func (m *mockOrderReader) ListActive(ctx context.Context) ([]*entity.Order, error) {
	return m.orders, m.err
}

type mockConfigReader struct {
	config entity.AgentConfig
	err    error
}

func (m *mockConfigReader) Config(ctx context.Context) (entity.AgentConfig, error) {
	return m.config, m.err
}

type mockStrategyReader struct {
	strategies []*entity.Strategy
	err        error
}

func (m *mockStrategyReader) ListActive(ctx context.Context) ([]*entity.Strategy, error) {
	return m.strategies, m.err
}

func TestQueryService_GetDashboard(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC()

	runtime := &mockRuntimeReader{
		status: entity.RuntimeStatus{
			Mode:             entity.RunModePaperTrading,
			Connectivity:     entity.ConnectivityStateConnected,
			License:          entity.LicenseStateDemo,
			LastUpdatedAtUTC: now,
			NTPDrift:         50 * time.Millisecond,
		},
	}

	positions := &mockPositionReader{
		positions: []*entity.Position{
			{
				ID:            "pos1",
				StrategyID:    "strat1",
				Symbol:        "BTCUSDT",
				Side:          entity.PositionSideLong,
				Quantity:      decimal.NewFromFloat(0.1),
				EntryPrice:    decimal.NewFromInt(50000),
				CurrentPrice:  decimal.NewFromInt(51000),
				UnrealizedPnL: decimal.NewFromInt(100),
				OpenedAtUTC:   now,
			},
		},
	}

	orders := &mockOrderReader{
		orders: []*entity.Order{
			{
				ID:       "order1",
				Symbol:   "BTCUSDT",
				Side:     entity.OrderSideBuy,
				Type:     entity.OrderTypeLimit,
				Status:   entity.OrderStatusSubmitted,
				Quantity: decimal.NewFromFloat(0.1),
				Price:    decimal.NewFromInt(49000),
			},
		},
	}

	strategies := &mockStrategyReader{
		strategies: []*entity.Strategy{
			{
				ID:     "strat1",
				Name:   "Test Strategy",
				Status: entity.StrategyStatusActive,
			},
		},
	}

	svc := NewQueryService(
		runtime,
		positions,
		&mockBalanceReader{},
		orders,
		&mockConfigReader{},
		strategies,
	)

	dashboard, err := svc.GetDashboard(ctx)

	require.NoError(t, err)
	require.NotNil(t, dashboard)
	assert.Equal(t, "paper_trading", dashboard.SystemStatus.Mode)
	assert.Equal(t, 1, dashboard.Performance.OpenPositions)
	assert.Equal(t, 1, dashboard.Performance.ActiveOrders)
	assert.Len(t, dashboard.ActiveStrategies, 1)
	assert.Len(t, dashboard.ActiveOrders, 1)
}

func TestQueryService_GetDashboard_RuntimeError(t *testing.T) {
	ctx := context.Background()

	runtime := &mockRuntimeReader{
		err: assert.AnError,
	}

	svc := NewQueryService(
		runtime,
		&mockPositionReader{},
		&mockBalanceReader{},
		&mockOrderReader{},
		&mockConfigReader{},
		&mockStrategyReader{},
	)

	dashboard, err := svc.GetDashboard(ctx)

	assert.Error(t, err)
	assert.Nil(t, dashboard)
}

func TestQueryService_GetPositions(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC()

	positions := &mockPositionReader{
		positions: []*entity.Position{
			{
				ID:            "pos1",
				Symbol:        "BTCUSDT",
				Side:          entity.PositionSideLong,
				Quantity:      decimal.NewFromFloat(0.1),
				EntryPrice:    decimal.NewFromInt(50000),
				CurrentPrice:  decimal.NewFromInt(51000),
				UnrealizedPnL: decimal.NewFromInt(100),
				OpenedAtUTC:   now,
			},
			{
				ID:            "pos2",
				Symbol:        "ETHUSDT",
				Side:          entity.PositionSideShort,
				Quantity:      decimal.NewFromFloat(2.5),
				EntryPrice:    decimal.NewFromInt(3200),
				CurrentPrice:  decimal.NewFromInt(3150),
				UnrealizedPnL: decimal.NewFromInt(125),
				OpenedAtUTC:   now,
			},
		},
	}

	svc := NewQueryService(
		&mockRuntimeReader{},
		positions,
		&mockBalanceReader{},
		&mockOrderReader{},
		&mockConfigReader{},
		&mockStrategyReader{},
	)

	positionsVM, err := svc.GetPositions(ctx)

	require.NoError(t, err)
	require.NotNil(t, positionsVM)
	assert.Len(t, positionsVM.Positions, 2)
	assert.Equal(t, "+$225.00", positionsVM.TotalPnL)
}

func TestQueryService_GetPositions_Error(t *testing.T) {
	ctx := context.Background()

	positions := &mockPositionReader{
		err: assert.AnError,
	}

	svc := NewQueryService(
		&mockRuntimeReader{},
		positions,
		&mockBalanceReader{},
		&mockOrderReader{},
		&mockConfigReader{},
		&mockStrategyReader{},
	)

	positionsVM, err := svc.GetPositions(ctx)

	assert.Error(t, err)
	assert.Nil(t, positionsVM)
}

func TestQueryService_GetBalance(t *testing.T) {
	ctx := context.Background()

	balance := &mockBalanceReader{
		balance: &entity.AccountBalance{
			Balances: []entity.Balance{
				{
					Asset:     "USDT",
					Total:     decimal.NewFromInt(10000),
					Available: decimal.NewFromInt(9500),
					Locked:    decimal.NewFromInt(500),
				},
			},
			UpdatedAtUTC: time.Now().UTC(),
		},
	}

	svc := NewQueryService(
		&mockRuntimeReader{},
		&mockPositionReader{},
		balance,
		&mockOrderReader{},
		&mockConfigReader{},
		&mockStrategyReader{},
	)

	balanceVM, err := svc.GetBalance(ctx)

	require.NoError(t, err)
	require.NotNil(t, balanceVM)
	assert.Len(t, balanceVM.Balances, 1)
	assert.Equal(t, "USDT", balanceVM.Balances[0].Asset)
}

func TestQueryService_GetBalance_Error(t *testing.T) {
	ctx := context.Background()

	balance := &mockBalanceReader{
		err: assert.AnError,
	}

	svc := NewQueryService(
		&mockRuntimeReader{},
		&mockPositionReader{},
		balance,
		&mockOrderReader{},
		&mockConfigReader{},
		&mockStrategyReader{},
	)

	balanceVM, err := svc.GetBalance(ctx)

	assert.Error(t, err)
	assert.Nil(t, balanceVM)
}

func TestQueryService_GetMarket(t *testing.T) {
	ctx := context.Background()

	svc := NewQueryService(
		&mockRuntimeReader{},
		&mockPositionReader{},
		&mockBalanceReader{},
		&mockOrderReader{},
		&mockConfigReader{},
		&mockStrategyReader{},
	)

	marketVM, err := svc.GetMarket(ctx, "BTCUSDT")

	require.NoError(t, err)
	require.NotNil(t, marketVM)
	assert.Equal(t, "BTCUSDT", marketVM.Symbol)
}

func TestQueryService_GetSettings(t *testing.T) {
	ctx := context.Background()

	config := &mockConfigReader{
		config: entity.AgentConfig{
			SchemaVersion:  1,
			ExportTimezone: "UTC",
			Telemetry:      entity.TelemetryConsent{Enabled: true, Profile: entity.TelemetryProfileMinimal, ConsentVersion: "v1"},
			Risk: entity.RiskConfig{
				MaxPosition: "1000.00",
			},
			Exchange: entity.ExchangeConfig{
				DefaultSymbol: "BTCUSDT",
			},
		},
	}

	svc := NewQueryService(
		&mockRuntimeReader{},
		&mockPositionReader{},
		&mockBalanceReader{},
		&mockOrderReader{},
		config,
		&mockStrategyReader{},
	)

	settingsVM, err := svc.GetSettings(ctx)

	require.NoError(t, err)
	require.NotNil(t, settingsVM)
	assert.Len(t, settingsVM.Risk, 1)
	assert.Len(t, settingsVM.Trading, 1)
	assert.Len(t, settingsVM.System, 4)
}

func TestQueryService_GetSettings_Error(t *testing.T) {
	ctx := context.Background()

	config := &mockConfigReader{
		err: assert.AnError,
	}

	svc := NewQueryService(
		&mockRuntimeReader{},
		&mockPositionReader{},
		&mockBalanceReader{},
		&mockOrderReader{},
		config,
		&mockStrategyReader{},
	)

	settingsVM, err := svc.GetSettings(ctx)

	assert.Error(t, err)
	assert.Nil(t, settingsVM)
}
