package entity

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestSymbol_IsTrading(t *testing.T) {
	tests := []struct {
		name     string
		status   SymbolStatus
		expected bool
	}{
		{
			name:     "trading status",
			status:   SymbolStatusTrading,
			expected: true,
		},
		{
			name:     "maintenance status",
			status:   SymbolStatusMaintenance,
			expected: false,
		},
		{
			name:     "suspended status",
			status:   SymbolStatusSuspended,
			expected: false,
		},
		{
			name:     "delisted status",
			status:   SymbolStatusDelisted,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			symbol := &Symbol{Status: tt.status}
			assert.Equal(t, tt.expected, symbol.IsTrading())
		})
	}
}

func TestSymbol_IsMaintenance(t *testing.T) {
	tests := []struct {
		name     string
		status   SymbolStatus
		expected bool
	}{
		{
			name:     "maintenance status",
			status:   SymbolStatusMaintenance,
			expected: true,
		},
		{
			name:     "trading status",
			status:   SymbolStatusTrading,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			symbol := &Symbol{Status: tt.status}
			assert.Equal(t, tt.expected, symbol.IsMaintenance())
		})
	}
}

func TestSymbol_IsSuspended(t *testing.T) {
	tests := []struct {
		name     string
		status   SymbolStatus
		expected bool
	}{
		{
			name:     "suspended status",
			status:   SymbolStatusSuspended,
			expected: true,
		},
		{
			name:     "trading status",
			status:   SymbolStatusTrading,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			symbol := &Symbol{Status: tt.status}
			assert.Equal(t, tt.expected, symbol.IsSuspended())
		})
	}
}

func TestSymbol_IsDelisted(t *testing.T) {
	tests := []struct {
		name     string
		status   SymbolStatus
		expected bool
	}{
		{
			name:     "delisted status",
			status:   SymbolStatusDelisted,
			expected: true,
		},
		{
			name:     "trading status",
			status:   SymbolStatusTrading,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			symbol := &Symbol{Status: tt.status}
			assert.Equal(t, tt.expected, symbol.IsDelisted())
		})
	}
}

func TestSymbol_BlocksNewEntries(t *testing.T) {
	tests := []struct {
		name     string
		status   SymbolStatus
		expected bool
	}{
		{
			name:     "trading allows entries",
			status:   SymbolStatusTrading,
			expected: false,
		},
		{
			name:     "maintenance blocks entries",
			status:   SymbolStatusMaintenance,
			expected: true,
		},
		{
			name:     "suspended blocks entries",
			status:   SymbolStatusSuspended,
			expected: true,
		},
		{
			name:     "delisted blocks entries",
			status:   SymbolStatusDelisted,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			symbol := &Symbol{Status: tt.status}
			assert.Equal(t, tt.expected, symbol.BlocksNewEntries())
		})
	}
}

func TestSymbol_RequiresForcedClose(t *testing.T) {
	tests := []struct {
		name     string
		status   SymbolStatus
		expected bool
	}{
		{
			name:     "delisted requires forced close",
			status:   SymbolStatusDelisted,
			expected: true,
		},
		{
			name:     "trading does not require forced close",
			status:   SymbolStatusTrading,
			expected: false,
		},
		{
			name:     "maintenance does not require forced close",
			status:   SymbolStatusMaintenance,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			symbol := &Symbol{Status: tt.status}
			assert.Equal(t, tt.expected, symbol.RequiresForcedClose())
		})
	}
}

func TestSymbol_ValidateQuantity(t *testing.T) {
	symbol := &Symbol{
		MinOrderQty: decimal.NewFromFloat(0.001),
		MaxOrderQty: decimal.NewFromInt(100),
		StepSize:    decimal.NewFromFloat(0.001),
	}

	tests := []struct {
		name     string
		quantity decimal.Decimal
		expected bool
	}{
		{
			name:     "valid quantity",
			quantity: decimal.NewFromFloat(0.1),
			expected: true,
		},
		{
			name:     "minimum quantity",
			quantity: decimal.NewFromFloat(0.001),
			expected: true,
		},
		{
			name:     "maximum quantity",
			quantity: decimal.NewFromInt(100),
			expected: true,
		},
		{
			name:     "below minimum",
			quantity: decimal.NewFromFloat(0.0001),
			expected: false,
		},
		{
			name:     "above maximum",
			quantity: decimal.NewFromInt(101),
			expected: false,
		},
		{
			name:     "not aligned to step size",
			quantity: decimal.NewFromFloat(0.1005),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, symbol.ValidateQuantity(tt.quantity))
		})
	}
}

func TestSymbol_ValidatePrice(t *testing.T) {
	symbol := &Symbol{
		MinPrice: decimal.NewFromInt(1),
		MaxPrice: decimal.NewFromInt(100000),
		TickSize: decimal.NewFromInt(1),
	}

	tests := []struct {
		name     string
		price    decimal.Decimal
		expected bool
	}{
		{
			name:     "valid price",
			price:    decimal.NewFromInt(50000),
			expected: true,
		},
		{
			name:     "minimum price",
			price:    decimal.NewFromInt(1),
			expected: true,
		},
		{
			name:     "maximum price",
			price:    decimal.NewFromInt(100000),
			expected: true,
		},
		{
			name:     "below minimum",
			price:    decimal.NewFromFloat(0.5),
			expected: false,
		},
		{
			name:     "above maximum",
			price:    decimal.NewFromInt(100001),
			expected: false,
		},
		{
			name:     "not aligned to tick size",
			price:    decimal.NewFromFloat(50000.5),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, symbol.ValidatePrice(tt.price))
		})
	}
}

func TestSymbol_ValidateNotional(t *testing.T) {
	symbol := &Symbol{
		MinNotional: decimal.NewFromInt(10),
	}

	tests := []struct {
		name     string
		quantity decimal.Decimal
		price    decimal.Decimal
		expected bool
	}{
		{
			name:     "valid notional",
			quantity: decimal.NewFromFloat(0.1),
			price:    decimal.NewFromInt(50000),
			expected: true,
		},
		{
			name:     "exact minimum notional",
			quantity: decimal.NewFromFloat(0.1),
			price:    decimal.NewFromInt(100),
			expected: true,
		},
		{
			name:     "below minimum notional",
			quantity: decimal.NewFromFloat(0.001),
			price:    decimal.NewFromInt(5),
			expected: false,
		},
		{
			name:     "zero notional",
			quantity: decimal.Zero,
			price:    decimal.NewFromInt(50000),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, symbol.ValidateNotional(tt.quantity, tt.price))
		})
	}
}

func TestSymbolStatus_Constants(t *testing.T) {
	assert.Equal(t, SymbolStatus("trading"), SymbolStatusTrading)
	assert.Equal(t, SymbolStatus("maintenance"), SymbolStatusMaintenance)
	assert.Equal(t, SymbolStatus("suspended"), SymbolStatusSuspended)
	assert.Equal(t, SymbolStatus("delisted"), SymbolStatusDelisted)
}
