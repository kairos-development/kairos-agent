package storage

import (
	"time"

	"github.com/shopspring/decimal"
)

// RateState is the storage-layer representation of a persisted token bucket.
type RateState struct {
	BucketID        string
	Tokens          decimal.Decimal
	Capacity        decimal.Decimal
	RefillPerSecond decimal.Decimal
	UpdatedAtUTC    time.Time
}

// PartialFill is the storage-layer representation of an in-flight partial fill.
type PartialFill struct {
	ClientOrderID string
	Symbol        string
	FilledQty     decimal.Decimal
	RemainingQty  decimal.Decimal
	AvgFillPrice  decimal.Decimal
	UpdatedAtUTC  time.Time
}

// ExportTradeRecord is the storage-layer representation of a closed trade row.
type ExportTradeRecord struct {
	Date        time.Time
	Pair        string
	Side        string
	Size        decimal.Decimal
	EntryPrice  decimal.Decimal
	ExitPrice   decimal.Decimal
	Fee         decimal.Decimal
	RealizedPNL decimal.Decimal
}
