package entity

import (
	"time"

	"github.com/shopspring/decimal"
)

// BalanceType identifies the balance category.
type BalanceType string

const (
	BalanceTypeTotal     BalanceType = "total"
	BalanceTypeAvailable BalanceType = "available"
	BalanceTypeLocked    BalanceType = "locked"
)

// Balance represents account balance in the domain layer.
// All monetary values use shopspring/decimal.
type Balance struct {
	Asset        string
	Total        decimal.Decimal
	Available    decimal.Decimal
	Locked       decimal.Decimal
	UpdatedAtUTC time.Time
}

// IsZero returns true if all balance fields are zero.
func (b *Balance) IsZero() bool {
	return b.Total.IsZero() && b.Available.IsZero() && b.Locked.IsZero()
}

// HasSufficientAvailable returns true if available balance meets the required amount.
func (b *Balance) HasSufficientAvailable(required decimal.Decimal) bool {
	return b.Available.GreaterThanOrEqual(required)
}

// AccountBalance represents the complete account balance state.
type AccountBalance struct {
	Balances     []Balance
	UpdatedAtUTC time.Time
}

// GetBalance returns the balance for a specific asset.
func (ab *AccountBalance) GetBalance(asset string) *Balance {
	for i := range ab.Balances {
		if ab.Balances[i].Asset == asset {
			return &ab.Balances[i]
		}
	}
	return nil
}

// TotalEquityUSD returns the total account equity in USD terms.
// This requires price conversion which should be handled by the service layer.
func (ab *AccountBalance) TotalEquityUSD(prices map[string]decimal.Decimal) decimal.Decimal {
	total := decimal.Zero
	for _, bal := range ab.Balances {
		if bal.Asset == "USD" || bal.Asset == "USDT" {
			total = total.Add(bal.Total)
		} else if price, ok := prices[bal.Asset]; ok {
			total = total.Add(bal.Total.Mul(price))
		}
	}
	return total
}
