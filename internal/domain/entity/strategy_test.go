package entity

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestStrategy_IsActive(t *testing.T) {
	tests := []struct {
		name     string
		status   StrategyStatus
		expected bool
	}{
		{
			name:     "active status",
			status:   StrategyStatusActive,
			expected: true,
		},
		{
			name:     "idle status",
			status:   StrategyStatusIdle,
			expected: false,
		},
		{
			name:     "paused status",
			status:   StrategyStatusPaused,
			expected: false,
		},
		{
			name:     "stopped status",
			status:   StrategyStatusStopped,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			strategy := &Strategy{Status: tt.status}
			assert.Equal(t, tt.expected, strategy.IsActive())
		})
	}
}

func TestStrategy_IsPaused(t *testing.T) {
	tests := []struct {
		name     string
		status   StrategyStatus
		expected bool
	}{
		{
			name:     "paused status",
			status:   StrategyStatusPaused,
			expected: true,
		},
		{
			name:     "active status",
			status:   StrategyStatusActive,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			strategy := &Strategy{Status: tt.status}
			assert.Equal(t, tt.expected, strategy.IsPaused())
		})
	}
}

func TestStrategy_IsStopped(t *testing.T) {
	tests := []struct {
		name     string
		status   StrategyStatus
		expected bool
	}{
		{
			name:     "stopped status",
			status:   StrategyStatusStopped,
			expected: true,
		},
		{
			name:     "active status",
			status:   StrategyStatusActive,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			strategy := &Strategy{Status: tt.status}
			assert.Equal(t, tt.expected, strategy.IsStopped())
		})
	}
}

func TestStrategy_CanTrade(t *testing.T) {
	tests := []struct {
		name     string
		status   StrategyStatus
		expected bool
	}{
		{
			name:     "active can trade",
			status:   StrategyStatusActive,
			expected: true,
		},
		{
			name:     "idle cannot trade",
			status:   StrategyStatusIdle,
			expected: false,
		},
		{
			name:     "paused cannot trade",
			status:   StrategyStatusPaused,
			expected: false,
		},
		{
			name:     "stopped cannot trade",
			status:   StrategyStatusStopped,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			strategy := &Strategy{Status: tt.status}
			assert.Equal(t, tt.expected, strategy.CanTrade())
		})
	}
}

func TestStrategy_HasReachedDailyLossLimit(t *testing.T) {
	tests := []struct {
		name         string
		dailyPnL     decimal.Decimal
		maxDailyLoss decimal.Decimal
		expected     bool
	}{
		{
			name:         "no limit set",
			dailyPnL:     decimal.NewFromInt(-1000),
			maxDailyLoss: decimal.Zero,
			expected:     false,
		},
		{
			name:         "within limit",
			dailyPnL:     decimal.NewFromInt(-500),
			maxDailyLoss: decimal.NewFromInt(1000),
			expected:     false,
		},
		{
			name:         "at limit",
			dailyPnL:     decimal.NewFromInt(-1000),
			maxDailyLoss: decimal.NewFromInt(1000),
			expected:     true,
		},
		{
			name:         "exceeded limit",
			dailyPnL:     decimal.NewFromInt(-1500),
			maxDailyLoss: decimal.NewFromInt(1000),
			expected:     true,
		},
		{
			name:         "positive pnl",
			dailyPnL:     decimal.NewFromInt(500),
			maxDailyLoss: decimal.NewFromInt(1000),
			expected:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			strategy := &Strategy{
				DailyPnL:     tt.dailyPnL,
				MaxDailyLoss: tt.maxDailyLoss,
			}
			assert.Equal(t, tt.expected, strategy.HasReachedDailyLossLimit())
		})
	}
}

func TestStrategy_WinRate(t *testing.T) {
	tests := []struct {
		name          string
		totalTrades   int
		winningTrades int
		expected      string
	}{
		{
			name:          "no trades",
			totalTrades:   0,
			winningTrades: 0,
			expected:      "0",
		},
		{
			name:          "50% win rate",
			totalTrades:   10,
			winningTrades: 5,
			expected:      "50",
		},
		{
			name:          "100% win rate",
			totalTrades:   10,
			winningTrades: 10,
			expected:      "100",
		},
		{
			name:          "0% win rate",
			totalTrades:   10,
			winningTrades: 0,
			expected:      "0",
		},
		{
			name:          "66.67% win rate",
			totalTrades:   3,
			winningTrades: 2,
			expected:      "66.66666666666667",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			strategy := &Strategy{
				TotalTrades:   tt.totalTrades,
				WinningTrades: tt.winningTrades,
			}
			winRate := strategy.WinRate()
			assert.Equal(t, tt.expected, winRate.String())
		})
	}
}

func TestStrategy_UpdatePnL(t *testing.T) {
	strategy := &Strategy{
		CurrentPnL:    decimal.NewFromInt(100),
		DailyPnL:      decimal.NewFromInt(50),
		TotalTrades:   5,
		WinningTrades: 3,
		LosingTrades:  2,
	}

	// Add winning trade
	strategy.UpdatePnL(decimal.NewFromInt(50))

	assert.True(t, strategy.CurrentPnL.Equal(decimal.NewFromInt(150)))
	assert.True(t, strategy.DailyPnL.Equal(decimal.NewFromInt(100)))
	assert.Equal(t, 6, strategy.TotalTrades)
	assert.Equal(t, 4, strategy.WinningTrades)
	assert.Equal(t, 2, strategy.LosingTrades)

	// Add losing trade
	strategy.UpdatePnL(decimal.NewFromInt(-30))

	assert.True(t, strategy.CurrentPnL.Equal(decimal.NewFromInt(120)))
	assert.True(t, strategy.DailyPnL.Equal(decimal.NewFromInt(70)))
	assert.Equal(t, 7, strategy.TotalTrades)
	assert.Equal(t, 4, strategy.WinningTrades)
	assert.Equal(t, 3, strategy.LosingTrades)

	// Add breakeven trade (zero PnL)
	strategy.UpdatePnL(decimal.Zero)

	assert.True(t, strategy.CurrentPnL.Equal(decimal.NewFromInt(120)))
	assert.True(t, strategy.DailyPnL.Equal(decimal.NewFromInt(70)))
	assert.Equal(t, 8, strategy.TotalTrades)
	assert.Equal(t, 4, strategy.WinningTrades)
	assert.Equal(t, 3, strategy.LosingTrades)
}

func TestStrategy_UpdatePnL_UpdatesTimestamp(t *testing.T) {
	strategy := &Strategy{
		UpdatedAtUTC: time.Now().UTC().Add(-1 * time.Hour),
	}

	oldTime := strategy.UpdatedAtUTC
	time.Sleep(10 * time.Millisecond)

	strategy.UpdatePnL(decimal.NewFromInt(100))

	assert.True(t, strategy.UpdatedAtUTC.After(oldTime))
}

func TestStrategy_ResetDailyPnL(t *testing.T) {
	strategy := &Strategy{
		DailyPnL:     decimal.NewFromInt(500),
		UpdatedAtUTC: time.Now().UTC().Add(-1 * time.Hour),
	}

	oldTime := strategy.UpdatedAtUTC
	time.Sleep(10 * time.Millisecond)

	strategy.ResetDailyPnL()

	assert.True(t, strategy.DailyPnL.Equal(decimal.Zero))
	assert.True(t, strategy.UpdatedAtUTC.After(oldTime))
}

func TestStrategy_Fields(t *testing.T) {
	now := time.Now().UTC()
	startedAt := now.Add(-1 * time.Hour)

	strategy := &Strategy{
		ID:              "strat1",
		Name:            "SMA-Cross",
		Type:            StrategyTypePaper,
		Status:          StrategyStatusActive,
		Symbol:          "BTCUSDT",
		MaxPositionSize: decimal.NewFromInt(1),
		MaxDailyLoss:    decimal.NewFromInt(1000),
		CurrentPnL:      decimal.NewFromInt(500),
		DailyPnL:        decimal.NewFromInt(200),
		TotalTrades:     10,
		WinningTrades:   6,
		LosingTrades:    4,
		CreatedAtUTC:    now,
		UpdatedAtUTC:    now,
		StartedAtUTC:    &startedAt,
	}

	assert.Equal(t, "strat1", strategy.ID)
	assert.Equal(t, "SMA-Cross", strategy.Name)
	assert.Equal(t, StrategyTypePaper, strategy.Type)
	assert.Equal(t, StrategyStatusActive, strategy.Status)
	assert.Equal(t, "BTCUSDT", strategy.Symbol)
	assert.True(t, strategy.MaxPositionSize.Equal(decimal.NewFromInt(1)))
	assert.True(t, strategy.MaxDailyLoss.Equal(decimal.NewFromInt(1000)))
	assert.True(t, strategy.CurrentPnL.Equal(decimal.NewFromInt(500)))
	assert.True(t, strategy.DailyPnL.Equal(decimal.NewFromInt(200)))
	assert.Equal(t, 10, strategy.TotalTrades)
	assert.Equal(t, 6, strategy.WinningTrades)
	assert.Equal(t, 4, strategy.LosingTrades)
	assert.NotNil(t, strategy.StartedAtUTC)
}

func TestStrategyStatus_Constants(t *testing.T) {
	assert.Equal(t, StrategyStatus("idle"), StrategyStatusIdle)
	assert.Equal(t, StrategyStatus("active"), StrategyStatusActive)
	assert.Equal(t, StrategyStatus("paused"), StrategyStatusPaused)
	assert.Equal(t, StrategyStatus("stopped"), StrategyStatusStopped)
}

func TestStrategyType_Constants(t *testing.T) {
	assert.Equal(t, StrategyType("backtest"), StrategyTypeBacktest)
	assert.Equal(t, StrategyType("paper"), StrategyTypePaper)
	assert.Equal(t, StrategyType("live"), StrategyTypeLive)
}
