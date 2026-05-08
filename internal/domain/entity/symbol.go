package entity

import (
	"time"

	"github.com/shopspring/decimal"
)

// SymbolStatus identifies the trading status of a symbol.
type SymbolStatus string

const (
	SymbolStatusTrading     SymbolStatus = "trading"
	SymbolStatusMaintenance SymbolStatus = "maintenance"
	SymbolStatusSuspended   SymbolStatus = "suspended"
	SymbolStatusDelisted    SymbolStatus = "delisted"
)

// Symbol represents a tradable instrument with exchange-specific constraints.
type Symbol struct {
	Name          string
	BaseCurrency  string
	QuoteCurrency string
	Status        SymbolStatus
	MinOrderQty   decimal.Decimal
	MaxOrderQty   decimal.Decimal
	MinPrice      decimal.Decimal
	MaxPrice      decimal.Decimal
	TickSize      decimal.Decimal
	StepSize      decimal.Decimal
	MinNotional   decimal.Decimal
	MakerFee      decimal.Decimal
	TakerFee      decimal.Decimal
	UpdatedAtUTC  time.Time
}

// IsTrading returns true if the symbol is available for trading.
func (s *Symbol) IsTrading() bool {
	return s.Status == SymbolStatusTrading
}

// IsMaintenance returns true if the symbol is under maintenance.
func (s *Symbol) IsMaintenance() bool {
	return s.Status == SymbolStatusMaintenance
}

// IsSuspended returns true if trading is suspended.
func (s *Symbol) IsSuspended() bool {
	return s.Status == SymbolStatusSuspended
}

// IsDelisted returns true if the symbol has been delisted.
func (s *Symbol) IsDelisted() bool {
	return s.Status == SymbolStatusDelisted
}

// BlocksNewEntries returns true if new positions should not be opened.
func (s *Symbol) BlocksNewEntries() bool {
	return s.Status != SymbolStatusTrading
}

// RequiresForcedClose returns true if existing positions must be closed.
func (s *Symbol) RequiresForcedClose() bool {
	return s.Status == SymbolStatusDelisted
}

// ValidateQuantity checks if the quantity meets symbol constraints.
func (s *Symbol) ValidateQuantity(qty decimal.Decimal) bool {
	if qty.LessThan(s.MinOrderQty) || qty.GreaterThan(s.MaxOrderQty) {
		return false
	}

	// Check step size alignment
	remainder := qty.Mod(s.StepSize)
	return remainder.IsZero()
}

// ValidatePrice checks if the price meets symbol constraints.
func (s *Symbol) ValidatePrice(price decimal.Decimal) bool {
	if price.LessThan(s.MinPrice) || price.GreaterThan(s.MaxPrice) {
		return false
	}

	// Check tick size alignment
	remainder := price.Mod(s.TickSize)
	return remainder.IsZero()
}

// ValidateNotional checks if the order notional value meets minimum requirements.
func (s *Symbol) ValidateNotional(qty, price decimal.Decimal) bool {
	notional := qty.Mul(price)
	return notional.GreaterThanOrEqual(s.MinNotional)
}
