package entity

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestBalance_IsZero(t *testing.T) {
	tests := []struct {
		name      string
		total     decimal.Decimal
		available decimal.Decimal
		locked    decimal.Decimal
		expected  bool
	}{
		{
			name:      "all zero",
			total:     decimal.Zero,
			available: decimal.Zero,
			locked:    decimal.Zero,
			expected:  true,
		},
		{
			name:      "non-zero total",
			total:     decimal.NewFromInt(100),
			available: decimal.Zero,
			locked:    decimal.Zero,
			expected:  false,
		},
		{
			name:      "non-zero available",
			total:     decimal.Zero,
			available: decimal.NewFromInt(100),
			locked:    decimal.Zero,
			expected:  false,
		},
		{
			name:      "non-zero locked",
			total:     decimal.Zero,
			available: decimal.Zero,
			locked:    decimal.NewFromInt(100),
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			balance := &Balance{
				Total:     tt.total,
				Available: tt.available,
				Locked:    tt.locked,
			}
			assert.Equal(t, tt.expected, balance.IsZero())
		})
	}
}

func TestBalance_HasSufficientAvailable(t *testing.T) {
	tests := []struct {
		name      string
		available decimal.Decimal
		required  decimal.Decimal
		expected  bool
	}{
		{
			name:      "sufficient balance",
			available: decimal.NewFromInt(1000),
			required:  decimal.NewFromInt(500),
			expected:  true,
		},
		{
			name:      "exact balance",
			available: decimal.NewFromInt(1000),
			required:  decimal.NewFromInt(1000),
			expected:  true,
		},
		{
			name:      "insufficient balance",
			available: decimal.NewFromInt(1000),
			required:  decimal.NewFromInt(1500),
			expected:  false,
		},
		{
			name:      "zero balance",
			available: decimal.Zero,
			required:  decimal.NewFromInt(100),
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			balance := &Balance{
				Available: tt.available,
			}
			assert.Equal(t, tt.expected, balance.HasSufficientAvailable(tt.required))
		})
	}
}

func TestBalance_Fields(t *testing.T) {
	now := time.Now().UTC()
	balance := &Balance{
		Asset:        "USDT",
		Total:        decimal.NewFromInt(10000),
		Available:    decimal.NewFromInt(9500),
		Locked:       decimal.NewFromInt(500),
		UpdatedAtUTC: now,
	}

	assert.Equal(t, "USDT", balance.Asset)
	assert.True(t, balance.Total.Equal(decimal.NewFromInt(10000)))
	assert.True(t, balance.Available.Equal(decimal.NewFromInt(9500)))
	assert.True(t, balance.Locked.Equal(decimal.NewFromInt(500)))
	assert.Equal(t, now, balance.UpdatedAtUTC)
}

func TestAccountBalance_GetBalance(t *testing.T) {
	ab := &AccountBalance{
		Balances: []Balance{
			{Asset: "USDT", Total: decimal.NewFromInt(10000)},
			{Asset: "BTC", Total: decimal.NewFromFloat(0.5)},
			{Asset: "ETH", Total: decimal.NewFromInt(5)},
		},
	}

	tests := []struct {
		name     string
		asset    string
		expected bool
	}{
		{
			name:     "existing asset USDT",
			asset:    "USDT",
			expected: true,
		},
		{
			name:     "existing asset BTC",
			asset:    "BTC",
			expected: true,
		},
		{
			name:     "non-existing asset",
			asset:    "SOL",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			balance := ab.GetBalance(tt.asset)
			if tt.expected {
				assert.NotNil(t, balance)
				assert.Equal(t, tt.asset, balance.Asset)
			} else {
				assert.Nil(t, balance)
			}
		})
	}
}

func TestAccountBalance_TotalEquityUSD(t *testing.T) {
	ab := &AccountBalance{
		Balances: []Balance{
			{Asset: "USDT", Total: decimal.NewFromInt(10000)},
			{Asset: "BTC", Total: decimal.NewFromFloat(0.5)},
			{Asset: "ETH", Total: decimal.NewFromInt(5)},
			{Asset: "SOL", Total: decimal.NewFromInt(100)},
		},
	}

	prices := map[string]decimal.Decimal{
		"BTC": decimal.NewFromInt(50000),
		"ETH": decimal.NewFromInt(3000),
		// SOL price not provided
	}

	totalEquity := ab.TotalEquityUSD(prices)

	// Expected: 10000 (USDT) + 0.5 * 50000 (BTC) + 5 * 3000 (ETH) = 10000 + 25000 + 15000 = 50000
	expected := decimal.NewFromInt(50000)
	assert.True(t, totalEquity.Equal(expected))
}

func TestAccountBalance_TotalEquityUSD_USDAsset(t *testing.T) {
	ab := &AccountBalance{
		Balances: []Balance{
			{Asset: "USD", Total: decimal.NewFromInt(5000)},
			{Asset: "USDT", Total: decimal.NewFromInt(5000)},
		},
	}

	prices := map[string]decimal.Decimal{}

	totalEquity := ab.TotalEquityUSD(prices)

	// Expected: 5000 (USD) + 5000 (USDT) = 10000
	expected := decimal.NewFromInt(10000)
	assert.True(t, totalEquity.Equal(expected))
}

func TestAccountBalance_TotalEquityUSD_NoPrices(t *testing.T) {
	ab := &AccountBalance{
		Balances: []Balance{
			{Asset: "BTC", Total: decimal.NewFromFloat(0.5)},
			{Asset: "ETH", Total: decimal.NewFromInt(5)},
		},
	}

	prices := map[string]decimal.Decimal{}

	totalEquity := ab.TotalEquityUSD(prices)

	// Expected: 0 (no prices provided for non-USD assets)
	assert.True(t, totalEquity.Equal(decimal.Zero))
}

func TestAccountBalance_TotalEquityUSD_EmptyBalances(t *testing.T) {
	ab := &AccountBalance{
		Balances: []Balance{},
	}

	prices := map[string]decimal.Decimal{}

	totalEquity := ab.TotalEquityUSD(prices)

	assert.True(t, totalEquity.Equal(decimal.Zero))
}

func TestBalanceType_Constants(t *testing.T) {
	assert.Equal(t, BalanceType("total"), BalanceTypeTotal)
	assert.Equal(t, BalanceType("available"), BalanceTypeAvailable)
	assert.Equal(t, BalanceType("locked"), BalanceTypeLocked)
}
