package export

import (
	"time"

	"github.com/shopspring/decimal"
)

// TradeRecord is the CSV export-layer representation of a closed trade.
type TradeRecord struct {
	Date        time.Time
	Pair        string
	Side        string
	Size        decimal.Decimal
	EntryPrice  decimal.Decimal
	ExitPrice   decimal.Decimal
	Fee         decimal.Decimal
	RealizedPNL decimal.Decimal
}
