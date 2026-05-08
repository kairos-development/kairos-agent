package sqlite

import (
	"context"
	"fmt"
	"time"

	"github.com/kairos-development/kairos-agent/internal/domain/entity"
	"github.com/kairos-development/kairos-agent/internal/storage/dual"
)

type DualOrderRepository struct {
	writer *dual.Writer
}

func NewDualOrderRepository(writer *dual.Writer) *DualOrderRepository {
	return &DualOrderRepository{writer: writer}
}

func (r *DualOrderRepository) Create(ctx context.Context, order *entity.Order) error {
	query := `
	INSERT INTO orders (
		id, client_order_id, exchange_order_id, strategy_id, symbol,
		side, type, status, time_in_force, quantity, price,
		filled_qty, remaining_qty, avg_fill_price,
		created_at_utc, updated_at_utc, submitted_at_utc, filled_at_utc
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	err := r.writer.WriteCritical(ctx, query,
		order.ID,
		order.ClientOrderID,
		nullString(order.ExchangeOrderID),
		order.StrategyID,
		order.Symbol,
		order.Side,
		order.Type,
		order.Status,
		order.TimeInForce,
		order.Quantity.String(),
		order.Price.String(),
		order.FilledQty.String(),
		order.RemainingQty.String(),
		order.AvgFillPrice.String(),
		order.CreatedAtUTC.Format(time.RFC3339Nano),
		order.UpdatedAtUTC.Format(time.RFC3339Nano),
		nullTime(order.SubmittedAtUTC),
		nullTime(order.FilledAtUTC),
	)

	if err != nil {
		if isConstraintError(err) {
			return ErrDuplicateKey
		}
		return fmt.Errorf("create order (dual): %w", err)
	}
	return nil
}

func (r *DualOrderRepository) WriteAnalyticsEvent(ctx context.Context, eventType string, data interface{}) {
	r.writer.WriteAnalytics(eventType, data)
}
