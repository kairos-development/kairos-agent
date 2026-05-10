package sqlite

import (
	"fmt"
	"time"

	"github.com/shopspring/decimal"
)

func parseDecimalField(field string, raw string) (decimal.Decimal, error) {
	value, err := decimal.NewFromString(raw)
	if err != nil {
		return decimal.Zero, fmt.Errorf("parse decimal %s=%q: %w", field, raw, err)
	}
	return value, nil
}

func parseTimeField(field string, raw string) (time.Time, error) {
	value, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse timestamp %s=%q: %w", field, raw, err)
	}
	return value.UTC(), nil
}
