package sdk

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBaseStrategyParametersAndReset(t *testing.T) {
	base := &BaseStrategy{Parameters: map[string]interface{}{"period": 14}}
	assert.Equal(t, 14, base.GetParameters()["period"])
	require.NoError(t, base.SetParameters(map[string]interface{}{"period": 20}))
	assert.Equal(t, 20, base.GetParameters()["period"])
	require.NoError(t, base.Reset())
}

func TestIndicators(t *testing.T) {
	prices := []decimal.Decimal{decimal.NewFromInt(1), decimal.NewFromInt(2), decimal.NewFromInt(3), decimal.NewFromInt(4), decimal.NewFromInt(5)}
	assert.True(t, SimpleMovingAverage(prices, 3).Equal(decimal.NewFromInt(4)))
	assert.True(t, SimpleMovingAverage(prices, 10).IsZero())
	assert.False(t, ExponentialMovingAverage(prices, 3).IsZero())
	assert.True(t, ExponentialMovingAverage(nil, 3).IsZero())
	assert.False(t, RSI(prices, 3).IsZero())
	assert.True(t, RSI(prices[:2], 3).IsZero())
	assert.True(t, RSI([]decimal.Decimal{decimal.NewFromInt(1), decimal.NewFromInt(2), decimal.NewFromInt(3), decimal.NewFromInt(4)}, 3).Equal(decimal.NewFromInt(100)))
	macd := MACD(prices, 2, 3, 3)
	require.NotNil(t, macd)
	bands := BollingerBands(prices, 3, decimal.NewFromInt(2))
	require.NotNil(t, bands)
	assert.False(t, bands.MiddleBand.IsZero())
	assert.True(t, BollingerBands(prices[:2], 3, decimal.NewFromInt(2)).MiddleBand.IsZero())
}
