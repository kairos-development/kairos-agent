package viewmodel

import (
	"testing"
	"time"

	"github.com/kairos-development/kairos-agent/internal/domain/entity"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDashboard(t *testing.T) {
	now := time.Now().UTC()

	status := entity.RuntimeStatus{
		Mode:             entity.RunModePaperTrading,
		Connectivity:     entity.ConnectivityStateConnected,
		License:          entity.LicenseStateDemo,
		LastUpdatedAtUTC: now.Add(-5 * time.Minute),
		NTPDrift:         50 * time.Millisecond,
		HaltReason:       "stream gap",
		HaltedAtUTC:      &now,
	}

	positions := []*entity.Position{
		{
			ID:            "pos1",
			StrategyID:    "strat1",
			Symbol:        "BTCUSDT",
			Side:          entity.PositionSideLong,
			Quantity:      decimal.NewFromFloat(0.1),
			EntryPrice:    decimal.NewFromInt(50000),
			CurrentPrice:  decimal.NewFromInt(51000),
			UnrealizedPnL: decimal.NewFromInt(100),
			OpenedAtUTC:   now.Add(-1 * time.Hour),
		},
		{
			ID:            "pos2",
			StrategyID:    "strat1",
			Symbol:        "ETHUSDT",
			Side:          entity.PositionSideShort,
			Quantity:      decimal.NewFromFloat(2.5),
			EntryPrice:    decimal.NewFromInt(3200),
			CurrentPrice:  decimal.NewFromInt(3150),
			UnrealizedPnL: decimal.NewFromInt(125),
			OpenedAtUTC:   now.Add(-30 * time.Minute),
		},
	}

	orders := []*entity.Order{
		{
			ID:            "order1",
			ClientOrderID: "client1",
			Symbol:        "BTCUSDT",
			Side:          entity.OrderSideBuy,
			Type:          entity.OrderTypeLimit,
			Status:        entity.OrderStatusSubmitted,
			Quantity:      decimal.NewFromFloat(0.1),
			Price:         decimal.NewFromInt(49000),
		},
	}

	strategies := []*entity.Strategy{
		{
			ID:     "strat1",
			Name:   "Test Strategy",
			Status: entity.StrategyStatusActive,
		},
	}

	dashboard := NewDashboard(status, positions, orders, strategies)

	require.NotNil(t, dashboard)
	assert.Equal(t, "paper_trading", dashboard.SystemStatus.Mode)
	assert.Equal(t, "connected", dashboard.SystemStatus.Connectivity)
	assert.Equal(t, "demo", dashboard.SystemStatus.License)
	assert.Equal(t, "stream gap", dashboard.SystemStatus.HaltReason)
	assert.Equal(t, now.Format("15:04:05"), dashboard.SystemStatus.HaltedAt)
	assert.Equal(t, 2, dashboard.Performance.OpenPositions)
	assert.Equal(t, 1, dashboard.Performance.ActiveOrders)
	assert.Equal(t, "+$225.00", dashboard.Performance.TotalPnL)
	assert.Len(t, dashboard.ActiveStrategies, 1)
	assert.Equal(t, "Test Strategy", dashboard.ActiveStrategies[0].Name)
	assert.Equal(t, "+$225.00", dashboard.ActiveStrategies[0].PnL)
	assert.Len(t, dashboard.ActiveOrders, 1)
	assert.Equal(t, "BTCUSDT", dashboard.ActiveOrders[0].Symbol)
}

func TestNewDashboard_MaxOrders(t *testing.T) {
	status := entity.RuntimeStatus{
		Mode:             entity.RunModeIdle,
		Connectivity:     entity.ConnectivityStateConnected,
		License:          entity.LicenseStateDemo,
		LastUpdatedAtUTC: time.Now().UTC(),
	}

	// Create 10 orders
	orders := make([]*entity.Order, 10)
	for i := 0; i < 10; i++ {
		orders[i] = &entity.Order{
			ID:       string(rune('a' + i)),
			Symbol:   "BTCUSDT",
			Side:     entity.OrderSideBuy,
			Type:     entity.OrderTypeLimit,
			Status:   entity.OrderStatusSubmitted,
			Quantity: decimal.NewFromInt(1),
			Price:    decimal.NewFromInt(50000),
		}
	}

	dashboard := NewDashboard(status, nil, orders, nil)

	// Should limit to 5 orders
	assert.Len(t, dashboard.ActiveOrders, 5)
}

func TestNewPositions(t *testing.T) {
	now := time.Now().UTC()

	positions := []*entity.Position{
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
		{
			ID:            "pos2",
			StrategyID:    "strat2",
			Symbol:        "ETHUSDT",
			Side:          entity.PositionSideShort,
			Quantity:      decimal.NewFromFloat(2.5),
			EntryPrice:    decimal.NewFromInt(3200),
			CurrentPrice:  decimal.NewFromInt(3150),
			UnrealizedPnL: decimal.NewFromInt(125),
			OpenedAtUTC:   now,
		},
	}

	positionsVM := NewPositions(positions)

	require.NotNil(t, positionsVM)
	assert.Len(t, positionsVM.Positions, 2)
	assert.Equal(t, "+$225.00", positionsVM.TotalPnL)
	assert.Equal(t, "BTCUSDT", positionsVM.Positions[0].Symbol)
	assert.Equal(t, "long", positionsVM.Positions[0].Side)
	assert.Equal(t, "+$100.00", positionsVM.Positions[0].PnL)
	assert.Equal(t, "ETHUSDT", positionsVM.Positions[1].Symbol)
	assert.Equal(t, "short", positionsVM.Positions[1].Side)
	assert.Equal(t, "+$125.00", positionsVM.Positions[1].PnL)
}

func TestNewPositions_PnLPercentCalculation(t *testing.T) {
	positions := []*entity.Position{
		{
			ID:            "pos1",
			Symbol:        "BTCUSDT",
			Side:          entity.PositionSideLong,
			Quantity:      decimal.NewFromFloat(1.0),
			EntryPrice:    decimal.NewFromInt(50000),
			CurrentPrice:  decimal.NewFromInt(51000),
			UnrealizedPnL: decimal.NewFromInt(1000),
			OpenedAtUTC:   time.Now().UTC(),
		},
	}

	positionsVM := NewPositions(positions)

	require.NotNil(t, positionsVM)
	assert.Equal(t, "2.00%", positionsVM.Positions[0].PnLPercent)
}

func TestNewPositions_ZeroEntryPrice(t *testing.T) {
	positions := []*entity.Position{
		{
			ID:            "pos1",
			Symbol:        "BTCUSDT",
			Side:          entity.PositionSideLong,
			Quantity:      decimal.NewFromFloat(1.0),
			EntryPrice:    decimal.Zero,
			CurrentPrice:  decimal.NewFromInt(51000),
			UnrealizedPnL: decimal.NewFromInt(1000),
			OpenedAtUTC:   time.Now().UTC(),
		},
	}

	positionsVM := NewPositions(positions)

	require.NotNil(t, positionsVM)
	assert.Equal(t, "0.00%", positionsVM.Positions[0].PnLPercent)
}

func TestNewBalance(t *testing.T) {
	balance := &entity.AccountBalance{
		Balances: []entity.Balance{
			{
				Asset:     "USDT",
				Total:     decimal.NewFromInt(10000),
				Available: decimal.NewFromInt(9500),
				Locked:    decimal.NewFromInt(500),
			},
			{
				Asset:     "BTC",
				Total:     decimal.NewFromFloat(0.5),
				Available: decimal.NewFromFloat(0.4),
				Locked:    decimal.NewFromFloat(0.1),
			},
		},
		UpdatedAtUTC: time.Now().UTC(),
	}

	balanceVM := NewBalance(balance)

	require.NotNil(t, balanceVM)
	assert.Len(t, balanceVM.Balances, 2)
	assert.Equal(t, "USDT", balanceVM.Balances[0].Asset)
	assert.Equal(t, "10000", balanceVM.Balances[0].Total)
	assert.Equal(t, "9500", balanceVM.Balances[0].Available)
	assert.Equal(t, "500", balanceVM.Balances[0].Locked)
}

func TestNewMarket(t *testing.T) {
	marketVM := NewMarket("BTCUSDT")

	require.NotNil(t, marketVM)
	assert.Equal(t, "BTCUSDT", marketVM.Symbol)
	// Placeholder values
	assert.Equal(t, "0.00", marketVM.LastPrice)
}

func TestNewSettings(t *testing.T) {
	cfg := entity.AgentConfig{
		SchemaVersion:       1,
		ExportTimezone:      "UTC",
		Telemetry:           entity.TelemetryConsent{Enabled: true, Profile: entity.TelemetryProfileMinimal, ConsentVersion: "v1"},
		JournalMaxSizeBytes: 100 * 1024 * 1024,
		Risk: entity.RiskConfig{
			MaxPosition: "1000.00",
		},
		Exchange: entity.ExchangeConfig{
			DefaultSymbol: "BTCUSDT",
		},
	}

	settingsVM := NewSettings(cfg)

	require.NotNil(t, settingsVM)
	assert.Len(t, settingsVM.Risk, 1)
	assert.Equal(t, "max_position_size", settingsVM.Risk[0].Key)
	assert.Equal(t, "1000.00", settingsVM.Risk[0].Value)
	assert.Len(t, settingsVM.Trading, 1)
	assert.Equal(t, "default_symbol", settingsVM.Trading[0].Key)
	assert.Equal(t, "BTCUSDT", settingsVM.Trading[0].Value)
	assert.Len(t, settingsVM.System, 4)
}

func TestFormatMoney(t *testing.T) {
	tests := []struct {
		name     string
		amount   decimal.Decimal
		expected string
	}{
		{
			name:     "positive",
			amount:   decimal.NewFromFloat(123.45),
			expected: "+$123.45",
		},
		{
			name:     "negative",
			amount:   decimal.NewFromFloat(-123.45),
			expected: "-$123.45",
		},
		{
			name:     "zero",
			amount:   decimal.Zero,
			expected: "+$0.00",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatMoney(tt.amount)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{
			name:     "milliseconds",
			duration: 500 * time.Millisecond,
			expected: "500ms",
		},
		{
			name:     "seconds",
			duration: 30 * time.Second,
			expected: "30.0s",
		},
		{
			name:     "minutes",
			duration: 5 * time.Minute,
			expected: "5m 0s",
		},
		{
			name:     "hours",
			duration: 2*time.Hour + 30*time.Minute,
			expected: "2h 30m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDuration(tt.duration)
			assert.Equal(t, tt.expected, result)
		})
	}
}
