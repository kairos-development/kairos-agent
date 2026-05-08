package entity

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestPosition_IsFlat(t *testing.T) {
	tests := []struct {
		name     string
		quantity decimal.Decimal
		side     PositionSide
		expected bool
	}{
		{
			name:     "zero quantity is flat",
			quantity: decimal.Zero,
			side:     PositionSideLong,
			expected: true,
		},
		{
			name:     "flat side is flat",
			quantity: decimal.NewFromInt(1),
			side:     PositionSideFlat,
			expected: true,
		},
		{
			name:     "non-zero long is not flat",
			quantity: decimal.NewFromFloat(0.1),
			side:     PositionSideLong,
			expected: false,
		},
		{
			name:     "non-zero short is not flat",
			quantity: decimal.NewFromFloat(0.1),
			side:     PositionSideShort,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pos := &Position{
				Quantity: tt.quantity,
				Side:     tt.side,
			}
			assert.Equal(t, tt.expected, pos.IsFlat())
		})
	}
}

func TestPosition_IsOpen(t *testing.T) {
	tests := []struct {
		name     string
		quantity decimal.Decimal
		side     PositionSide
		expected bool
	}{
		{
			name:     "zero quantity is not open",
			quantity: decimal.Zero,
			side:     PositionSideLong,
			expected: false,
		},
		{
			name:     "flat side is not open",
			quantity: decimal.NewFromInt(1),
			side:     PositionSideFlat,
			expected: false,
		},
		{
			name:     "non-zero long is open",
			quantity: decimal.NewFromFloat(0.1),
			side:     PositionSideLong,
			expected: true,
		},
		{
			name:     "non-zero short is open",
			quantity: decimal.NewFromFloat(0.1),
			side:     PositionSideShort,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pos := &Position{
				Quantity: tt.quantity,
				Side:     tt.side,
			}
			assert.Equal(t, tt.expected, pos.IsOpen())
		})
	}
}

func TestPosition_UpdateUnrealizedPnL_Long(t *testing.T) {
	pos := &Position{
		Side:       PositionSideLong,
		Quantity:   decimal.NewFromFloat(0.1),
		EntryPrice: decimal.NewFromInt(50000),
	}

	// Price goes up - profit
	pos.UpdateUnrealizedPnL(decimal.NewFromInt(51000))
	assert.True(t, pos.CurrentPrice.Equal(decimal.NewFromInt(51000)))
	assert.True(t, pos.UnrealizedPnL.Equal(decimal.NewFromInt(100))) // (51000 - 50000) * 0.1 = 100

	// Price goes down - loss
	pos.UpdateUnrealizedPnL(decimal.NewFromInt(49000))
	assert.True(t, pos.CurrentPrice.Equal(decimal.NewFromInt(49000)))
	assert.True(t, pos.UnrealizedPnL.Equal(decimal.NewFromInt(-100))) // (49000 - 50000) * 0.1 = -100
}

func TestPosition_UpdateUnrealizedPnL_Short(t *testing.T) {
	pos := &Position{
		Side:       PositionSideShort,
		Quantity:   decimal.NewFromFloat(0.1),
		EntryPrice: decimal.NewFromInt(50000),
	}

	// Price goes down - profit for short
	pos.UpdateUnrealizedPnL(decimal.NewFromInt(49000))
	assert.True(t, pos.CurrentPrice.Equal(decimal.NewFromInt(49000)))
	assert.True(t, pos.UnrealizedPnL.Equal(decimal.NewFromInt(100))) // -(49000 - 50000) * 0.1 = 100

	// Price goes up - loss for short
	pos.UpdateUnrealizedPnL(decimal.NewFromInt(51000))
	assert.True(t, pos.CurrentPrice.Equal(decimal.NewFromInt(51000)))
	assert.True(t, pos.UnrealizedPnL.Equal(decimal.NewFromInt(-100))) // -(51000 - 50000) * 0.1 = -100
}

func TestPosition_UpdateUnrealizedPnL_UpdatesTimestamp(t *testing.T) {
	pos := &Position{
		Side:         PositionSideLong,
		Quantity:     decimal.NewFromFloat(0.1),
		EntryPrice:   decimal.NewFromInt(50000),
		UpdatedAtUTC: time.Now().UTC().Add(-1 * time.Hour),
	}

	oldTime := pos.UpdatedAtUTC
	time.Sleep(10 * time.Millisecond)

	pos.UpdateUnrealizedPnL(decimal.NewFromInt(51000))

	assert.True(t, pos.UpdatedAtUTC.After(oldTime))
}

func TestPosition_TotalPnL(t *testing.T) {
	tests := []struct {
		name          string
		realizedPnL   decimal.Decimal
		unrealizedPnL decimal.Decimal
		expected      decimal.Decimal
	}{
		{
			name:          "both positive",
			realizedPnL:   decimal.NewFromInt(100),
			unrealizedPnL: decimal.NewFromInt(50),
			expected:      decimal.NewFromInt(150),
		},
		{
			name:          "both negative",
			realizedPnL:   decimal.NewFromInt(-100),
			unrealizedPnL: decimal.NewFromInt(-50),
			expected:      decimal.NewFromInt(-150),
		},
		{
			name:          "mixed positive/negative",
			realizedPnL:   decimal.NewFromInt(100),
			unrealizedPnL: decimal.NewFromInt(-50),
			expected:      decimal.NewFromInt(50),
		},
		{
			name:          "zero values",
			realizedPnL:   decimal.Zero,
			unrealizedPnL: decimal.Zero,
			expected:      decimal.Zero,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pos := &Position{
				RealizedPnL:   tt.realizedPnL,
				UnrealizedPnL: tt.unrealizedPnL,
			}
			assert.True(t, pos.TotalPnL().Equal(tt.expected))
		})
	}
}

func TestPosition_Fields(t *testing.T) {
	now := time.Now().UTC()
	closedAt := now.Add(1 * time.Hour)

	pos := &Position{
		ID:            "pos123",
		StrategyID:    "strat1",
		Symbol:        "BTCUSDT",
		Side:          PositionSideLong,
		Quantity:      decimal.NewFromFloat(0.1),
		EntryPrice:    decimal.NewFromInt(50000),
		CurrentPrice:  decimal.NewFromInt(51000),
		UnrealizedPnL: decimal.NewFromInt(100),
		RealizedPnL:   decimal.NewFromInt(50),
		OpenedAtUTC:   now,
		UpdatedAtUTC:  now,
		ClosedAtUTC:   &closedAt,
	}

	assert.Equal(t, "pos123", pos.ID)
	assert.Equal(t, "strat1", pos.StrategyID)
	assert.Equal(t, "BTCUSDT", pos.Symbol)
	assert.Equal(t, PositionSideLong, pos.Side)
	assert.True(t, pos.Quantity.Equal(decimal.NewFromFloat(0.1)))
	assert.True(t, pos.EntryPrice.Equal(decimal.NewFromInt(50000)))
	assert.True(t, pos.CurrentPrice.Equal(decimal.NewFromInt(51000)))
	assert.True(t, pos.UnrealizedPnL.Equal(decimal.NewFromInt(100)))
	assert.True(t, pos.RealizedPnL.Equal(decimal.NewFromInt(50)))
	assert.Equal(t, now, pos.OpenedAtUTC)
	assert.Equal(t, now, pos.UpdatedAtUTC)
	assert.NotNil(t, pos.ClosedAtUTC)
	assert.Equal(t, closedAt, *pos.ClosedAtUTC)
}

func TestPositionSide_Constants(t *testing.T) {
	assert.Equal(t, PositionSide("long"), PositionSideLong)
	assert.Equal(t, PositionSide("short"), PositionSideShort)
	assert.Equal(t, PositionSide("flat"), PositionSideFlat)
}
