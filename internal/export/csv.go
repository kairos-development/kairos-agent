package export

import (
	"encoding/csv"
	"fmt"
	"io"
	"time"
)

// WriteCSV writes trade export rows in operator-local ISO 8601 time.
func WriteCSV(dst io.Writer, timezone string, rows []TradeRecord) error {
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return fmt.Errorf("load export timezone: %w", err)
	}
	writer := csv.NewWriter(dst)
	if err := writer.Write([]string{"date", "pair", "side", "size", "entry_price", "exit_price", "fee", "realized_pnl"}); err != nil {
		return err
	}
	for _, row := range rows {
		values := []string{
			row.Date.In(location).Format(time.RFC3339),
			row.Pair,
			row.Side,
			row.Size.String(),
			row.EntryPrice.String(),
			row.ExitPrice.String(),
			row.Fee.String(),
			row.RealizedPNL.String(),
		}
		if err := writer.Write(values); err != nil {
			return err
		}
	}
	writer.Flush()
	return writer.Error()
}
