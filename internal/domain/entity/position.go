package entity

import (
	"time"

	"github.com/shopspring/decimal"
)

// PositionSide identifies the position direction.
type PositionSide string

const (
	PositionSideLong  PositionSide = "long"
	PositionSideShort PositionSide = "short"
	PositionSideFlat  PositionSide = "flat"
)

// Position represents an open trading position in the domain layer.
// All monetary calculations use shopspring/decimal.
type Position struct {
	ID            string
	StrategyID    string
	Symbol        string
	Side          PositionSide
	Quantity      decimal.Decimal
	EntryPrice    decimal.Decimal
	CurrentPrice  decimal.Decimal
	UnrealizedPnL decimal.Decimal
	RealizedPnL   decimal.Decimal
	OpenedAtUTC   time.Time
	UpdatedAtUTC  time.Time
	ClosedAtUTC   *time.Time
}

// IsFlat returns true if the position has zero quantity.
func (p *Position) IsFlat() bool {
	return p.Quantity.IsZero() || p.Side == PositionSideFlat
}

// IsOpen returns true if the position has non-zero quantity.
func (p *Position) IsOpen() bool {
	return !p.IsFlat()
}

// UpdateUnrealizedPnL recalculates unrealized PnL based on current market price.
// This method enforces decimal arithmetic for all monetary calculations.
func (p *Position) UpdateUnrealizedPnL(currentPrice decimal.Decimal) {
	p.CurrentPrice = currentPrice
	priceDiff := currentPrice.Sub(p.EntryPrice)

	if p.Side == PositionSideShort {
		priceDiff = priceDiff.Neg()
	}

	p.UnrealizedPnL = priceDiff.Mul(p.Quantity)
	p.UpdatedAtUTC = time.Now().UTC()
}

// TotalPnL returns the sum of realized and unrealized PnL.
func (p *Position) TotalPnL() decimal.Decimal {
	return p.RealizedPnL.Add(p.UnrealizedPnL)
}
